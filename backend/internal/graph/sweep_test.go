package graph

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/swatkatz/advisorhub/backend/internal/alert"
	"github.com/swatkatz/advisorhub/backend/internal/client"
	"github.com/swatkatz/advisorhub/backend/internal/eventbus"
)

// Test Anchor 26: runMorningSweep calls all producers
func TestRunMorningSweep_CallsAllProducers(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	f.clientRepo.addClient(client.Client{ID: "c1", AdvisorID: "a1", Name: "Alice"})
	f.clientRepo.addClient(client.Client{ID: "c2", AdvisorID: "a1", Name: "Bob"})

	mr := &mutationResolver{f.resolver}
	result, err := mr.RunMorningSweep(ctx, "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify contribution engine was called for each client.
	if len(f.contribEngine.analyzeClients) != 2 {
		t.Errorf("expected 2 AnalyzeClient calls, got %d", len(f.contribEngine.analyzeClients))
	}

	// Verify transfer monitor was called.
	if !f.transferMon.called {
		t.Error("expected TransferMonitor.CheckStuckTransfers to be called")
	}

	// Verify temporal scanner was called.
	if !f.temporalScan.called {
		t.Error("expected TemporalScanner.RunSweep to be called")
	}

	if result.Duration == "" {
		t.Error("expected non-empty duration")
	}
}

// Test Anchor 27: sweep counts AlertCreated and AlertUpdated events
func TestRunMorningSweep_CountsEvents(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	f.clientRepo.addClient(client.Client{ID: "c1", AdvisorID: "a1", Name: "Alice"})

	// Publish some alert events before the sweep runs (simulating what producers would do).
	go func() {
		time.Sleep(10 * time.Millisecond)
		for i := 0; i < 3; i++ {
			f.eventBus.Publish(ctx, eventbus.EventEnvelope{
				Type:    alert.EventAlertCreated,
				Payload: json.RawMessage(`{"alert_id":"al1"}`),
			})
		}
		f.eventBus.Publish(ctx, eventbus.EventEnvelope{
			Type:    alert.EventAlertUpdated,
			Payload: json.RawMessage(`{"alert_id":"al2"}`),
		})
	}()

	mr := &mutationResolver{f.resolver}
	result, err := mr.RunMorningSweep(ctx, "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.AlertsGenerated != 3 {
		t.Errorf("expected alertsGenerated=3, got %d", result.AlertsGenerated)
	}
	if result.AlertsUpdated != 1 {
		t.Errorf("expected alertsUpdated=1, got %d", result.AlertsUpdated)
	}
}

// Test Anchor 28: sweep with no events returns zeros
func TestRunMorningSweep_NoEvents(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	mr := &mutationResolver{f.resolver}
	result, err := mr.RunMorningSweep(ctx, "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.AlertsGenerated != 0 || result.AlertsUpdated != 0 || result.AlertsSkipped != 0 {
		t.Errorf("expected all zeros, got gen=%d upd=%d skip=%d",
			result.AlertsGenerated, result.AlertsUpdated, result.AlertsSkipped)
	}
}

// Test Anchor 29: partial failure continues sweep
func TestRunMorningSweep_PartialFailure(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	f.clientRepo.addClient(client.Client{ID: "c1", AdvisorID: "a1", Name: "Alice"})
	f.clientRepo.addClient(client.Client{ID: "c2", AdvisorID: "a1", Name: "Bob"})

	f.contribEngine.analyzeErr["c1"] = errors.New("analysis failed")

	mr := &mutationResolver{f.resolver}
	result, err := mr.RunMorningSweep(ctx, "a1")
	if err != nil {
		t.Fatalf("expected no error on partial failure, got %v", err)
	}

	// Both clients should have been attempted.
	if len(f.contribEngine.analyzeClients) != 2 {
		t.Errorf("expected 2 AnalyzeClient calls, got %d", len(f.contribEngine.analyzeClients))
	}

	if result.Duration == "" {
		t.Error("expected non-empty duration")
	}
}

// Test Anchor 30: sweep returns non-empty duration
func TestRunMorningSweep_Duration(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	mr := &mutationResolver{f.resolver}
	result, err := mr.RunMorningSweep(ctx, "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Duration == "" {
		t.Error("expected non-empty duration string")
	}
}
