package graph

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/swatkatz/advisorhub/backend/internal/alert"
	"github.com/swatkatz/advisorhub/backend/internal/client"
	"github.com/swatkatz/advisorhub/backend/internal/eventbus"
	"github.com/swatkatz/advisorhub/backend/internal/graph/model"
)

// Test Anchor 31: alertFeed receives CREATED events for matching advisor
func TestAlertFeed_Created(t *testing.T) {
	f := newTestFixture()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f.clientRepo.addClient(client.Client{ID: "c1", AdvisorID: "a1", Name: "Alice"})
	f.alertRepo.addAlert(alert.Alert{
		ID: "al1", ClientID: "c1", Severity: alert.SeverityCritical,
		Status: alert.StatusOpen, Summary: "Test", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	sr := &subscriptionResolver{f.resolver}
	ch, err := sr.AlertFeed(ctx, "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Publish an AlertCreated event.
	f.eventBus.Publish(ctx, eventbus.EventEnvelope{
		Type:    alert.EventAlertCreated,
		Payload: json.RawMessage(`{"alert_id":"al1"}`),
		Source:  eventbus.SourceSystem,
	})

	select {
	case event := <-ch:
		if event.Type != model.AlertEventTypeCreated {
			t.Errorf("expected CREATED event type, got %s", event.Type)
		}
		if event.Alert.ID != "al1" {
			t.Errorf("expected alert al1, got %s", event.Alert.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for alert event")
	}
}

// Test Anchor 32: alertFeed receives UPDATED events
func TestAlertFeed_Updated(t *testing.T) {
	f := newTestFixture()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f.clientRepo.addClient(client.Client{ID: "c1", AdvisorID: "a1", Name: "Alice"})
	f.alertRepo.addAlert(alert.Alert{
		ID: "al1", ClientID: "c1", Severity: alert.SeverityCritical,
		Status: alert.StatusActed, Summary: "Test", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	sr := &subscriptionResolver{f.resolver}
	ch, err := sr.AlertFeed(ctx, "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	f.eventBus.Publish(ctx, eventbus.EventEnvelope{
		Type:    alert.EventAlertUpdated,
		Payload: json.RawMessage(`{"alert_id":"al1"}`),
		Source:  eventbus.SourceSystem,
	})

	select {
	case event := <-ch:
		if event.Type != model.AlertEventTypeUpdated {
			t.Errorf("expected UPDATED event type, got %s", event.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for alert event")
	}
}

// Test Anchor 33: alertFeed receives CLOSED events
func TestAlertFeed_Closed(t *testing.T) {
	f := newTestFixture()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f.clientRepo.addClient(client.Client{ID: "c1", AdvisorID: "a1", Name: "Alice"})
	f.alertRepo.addAlert(alert.Alert{
		ID: "al1", ClientID: "c1", Severity: alert.SeverityCritical,
		Status: alert.StatusClosed, Summary: "Test", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	sr := &subscriptionResolver{f.resolver}
	ch, err := sr.AlertFeed(ctx, "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	f.eventBus.Publish(ctx, eventbus.EventEnvelope{
		Type:    alert.EventAlertClosed,
		Payload: json.RawMessage(`{"alert_id":"al1"}`),
		Source:  eventbus.SourceSystem,
	})

	select {
	case event := <-ch:
		if event.Type != model.AlertEventTypeClosed {
			t.Errorf("expected CLOSED event type, got %s", event.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for alert event")
	}
}

// Test Anchor 34: alertFeed filters events for wrong advisor
func TestAlertFeed_FiltersByAdvisor(t *testing.T) {
	f := newTestFixture()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Client belongs to advisor a2, not a1.
	f.clientRepo.addClient(client.Client{ID: "c1", AdvisorID: "a2", Name: "Bob"})
	f.alertRepo.addAlert(alert.Alert{
		ID: "al1", ClientID: "c1", Severity: alert.SeverityCritical,
		Status: alert.StatusOpen, Summary: "Test", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	sr := &subscriptionResolver{f.resolver}
	ch, err := sr.AlertFeed(ctx, "a1") // Subscribe as a1
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Publish event for a2's client.
	f.eventBus.Publish(ctx, eventbus.EventEnvelope{
		Type:    alert.EventAlertCreated,
		Payload: json.RawMessage(`{"alert_id":"al1"}`),
		Source:  eventbus.SourceSystem,
	})

	select {
	case event := <-ch:
		t.Errorf("expected no event for wrong advisor, got %+v", event)
	case <-time.After(500 * time.Millisecond):
		// Correct: event was filtered out.
	}
}

// Test Anchor 35: alertFeed stops when context is cancelled
func TestAlertFeed_StopsOnCancel(t *testing.T) {
	f := newTestFixture()
	ctx, cancel := context.WithCancel(context.Background())

	sr := &subscriptionResolver{f.resolver}
	ch, err := sr.AlertFeed(ctx, "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Cancel the context.
	cancel()

	// Channel should eventually close.
	select {
	case _, ok := <-ch:
		if ok {
			// Might get a zero value before close — that's fine.
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for channel close after cancel")
	}
}
