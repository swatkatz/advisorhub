package transfer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/swatkatz/advisorhub/backend/internal/eventbus"
)

// monitor implements the TransferMonitor interface.
type monitor struct {
	repo TransferRepository
	bus  eventbus.EventBus
	now  func() time.Time
}

// NewMonitor creates a new TransferMonitor.
func NewMonitor(repo TransferRepository, bus eventbus.EventBus) TransferMonitor {
	return &monitor{
		repo: repo,
		bus:  bus,
		now:  func() time.Time { return time.Now() },
	}
}

// CheckStuckTransfers iterates all active (non-INVESTED) transfers and emits
// TransferStuck events for those exceeding their stage threshold.
func (m *monitor) CheckStuckTransfers(ctx context.Context) ([]TransferCheckResult, error) {
	active, err := m.repo.GetActiveTransfers(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting active transfers: %w", err)
	}

	results := make([]TransferCheckResult, 0, len(active))
	now := m.now()

	for _, t := range active {
		days := t.DaysInCurrentStage(now)
		threshold, ok := StageThreshold[t.Status]
		if !ok {
			continue
		}

		if days > threshold {
			if err := m.emitTransferStuck(ctx, &t, days, threshold, now); err != nil {
				return nil, fmt.Errorf("emitting TransferStuck for %s: %w", t.ID, err)
			}
			results = append(results, TransferCheckResult{
				TransferID: t.ID,
				Signal:     SignalStuckDetected,
			})
		} else {
			results = append(results, TransferCheckResult{
				TransferID: t.ID,
				Signal:     SignalNoChange,
			})
		}
	}

	return results, nil
}

func (m *monitor) emitTransferStuck(ctx context.Context, t *Transfer, daysInStage, threshold int, now time.Time) error {
	payload := map[string]any{
		"transfer_id":        t.ID,
		"client_id":          t.ClientID,
		"source_institution": t.SourceInstitution,
		"account_type":       t.AccountType,
		"amount":             t.Amount,
		"status":             string(t.Status),
		"days_in_stage":      daysInStage,
		"stuck_threshold":    threshold,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling TransferStuck payload: %w", err)
	}
	return m.bus.Publish(ctx, eventbus.EventEnvelope{
		ID:         fmt.Sprintf("evt_%s_%s_%d", EventTransferStuck, t.ID, now.UnixNano()),
		Type:       EventTransferStuck,
		EntityID:   t.ID,
		EntityType: eventbus.EntityTypeTransfer,
		Payload:    data,
		Source:     eventbus.SourceReactive,
		Timestamp:  now,
	})
}
