package transfer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

// memoryTransferRepo is an in-memory implementation of TransferRepository for tests.
type memoryTransferRepo struct {
	mu        sync.RWMutex
	transfers []Transfer
	now       func() time.Time
}

func newMemoryTransferRepo(now func() time.Time) *memoryTransferRepo {
	return &memoryTransferRepo{
		now: now,
	}
}

func (r *memoryTransferRepo) addTransfer(t Transfer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.transfers = append(r.transfers, t)
}

func (r *memoryTransferRepo) GetTransfer(_ context.Context, id string) (*Transfer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, t := range r.transfers {
		if t.ID == id {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("transfer not found: %s", id)
}

func (r *memoryTransferRepo) GetTransfersByClientID(_ context.Context, clientID string) ([]Transfer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []Transfer
	for _, t := range r.transfers {
		if t.ClientID == clientID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (r *memoryTransferRepo) GetActiveTransfers(_ context.Context) ([]Transfer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []Transfer
	for _, t := range r.transfers {
		if t.Status != StatusInvested {
			result = append(result, t)
		}
	}
	return result, nil
}

func (r *memoryTransferRepo) CreateTransfer(_ context.Context, transfer *Transfer) (*Transfer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	transfer.LastStatusChange = transfer.InitiatedAt
	r.transfers = append(r.transfers, *transfer)
	return transfer, nil
}

func (r *memoryTransferRepo) UpdateTransferStatus(_ context.Context, id string, newStatus TransferStatus) (*Transfer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, t := range r.transfers {
		if t.ID == id {
			expected, err := nextStatus(t.Status)
			if err != nil {
				return nil, fmt.Errorf("updating transfer %s: %w", id, err)
			}
			if newStatus != expected {
				return nil, fmt.Errorf("invalid transition from %s to %s: expected %s", t.Status, newStatus, expected)
			}
			r.transfers[i].Status = newStatus
			r.transfers[i].LastStatusChange = r.now()
			result := r.transfers[i]
			return &result, nil
		}
	}
	return nil, fmt.Errorf("transfer not found: %s", id)
}

// memoryEventBus captures published events for test assertions.
type memoryEventBus struct {
	mu     sync.Mutex
	events []EventEnvelope
}

func newMemoryEventBus() *memoryEventBus {
	return &memoryEventBus{}
}

func (b *memoryEventBus) Publish(_ context.Context, envelope EventEnvelope) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, envelope)
	return nil
}

func (b *memoryEventBus) eventsByType(eventType string) []EventEnvelope {
	b.mu.Lock()
	defer b.mu.Unlock()
	var result []EventEnvelope
	for _, e := range b.events {
		if e.Type == eventType {
			result = append(result, e)
		}
	}
	return result
}

func (b *memoryEventBus) allEvents() []EventEnvelope {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := make([]EventEnvelope, len(b.events))
	copy(result, b.events)
	return result
}

// setupMonitor creates a monitor with in-memory dependencies for testing.
func setupMonitor(now time.Time) (*monitor, *memoryTransferRepo, *memoryEventBus) {
	nowFn := func() time.Time { return now }
	repo := newMemoryTransferRepo(nowFn)
	bus := newMemoryEventBus()
	m := &monitor{
		repo: repo,
		bus:  bus,
		now:  nowFn,
	}
	return m, repo, bus
}

// parsePayload unmarshals an event payload into a map for test assertions.
func parsePayload(t *testing.T, data json.RawMessage) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return m
}
