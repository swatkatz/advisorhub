package eventbus

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func makeEnvelope(eventType, entityID string) EventEnvelope {
	return EventEnvelope{
		ID:         "evt-1",
		Type:       eventType,
		EntityID:   entityID,
		EntityType: EntityTypeClient,
		Payload:    json.RawMessage(`{"key":"value"}`),
		Source:     SourceReactive,
		Timestamp:  time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC),
	}
}

func TestSubscriberReceivesMatchingEvent(t *testing.T) {
	bus := New()
	ch := bus.Subscribe("OverContributionDetected")

	env := makeEnvelope("OverContributionDetected", "c1")
	if err := bus.Publish(context.Background(), env); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	select {
	case got := <-ch:
		if got.ID != env.ID {
			t.Errorf("got ID %q, want %q", got.ID, env.ID)
		}
		if got.Type != env.Type {
			t.Errorf("got Type %q, want %q", got.Type, env.Type)
		}
		if got.EntityID != env.EntityID {
			t.Errorf("got EntityID %q, want %q", got.EntityID, env.EntityID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestSubscriberDoesNotReceiveNonMatchingEvent(t *testing.T) {
	bus := New()
	ch := bus.Subscribe("OverContributionDetected")

	env := makeEnvelope("TransferStuck", "t1")
	if err := bus.Publish(context.Background(), env); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	select {
	case got := <-ch:
		t.Fatalf("should not have received event, got %+v", got)
	case <-time.After(50 * time.Millisecond):
		// Expected: no event received.
	}
}

func TestTwoSubscribersReceiveIndependentCopies(t *testing.T) {
	bus := New()
	ch1 := bus.Subscribe("TransferStuck")
	ch2 := bus.Subscribe("TransferStuck")

	env := makeEnvelope("TransferStuck", "t1")
	if err := bus.Publish(context.Background(), env); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	for i, ch := range []<-chan EventEnvelope{ch1, ch2} {
		select {
		case got := <-ch:
			if got.ID != env.ID {
				t.Errorf("subscriber %d: got ID %q, want %q", i, got.ID, env.ID)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d: timed out waiting for event", i)
		}
	}
}

func TestPublishWithNoSubscribers(t *testing.T) {
	bus := New()
	env := makeEnvelope("UnsubscribedEventType", "x1")
	if err := bus.Publish(context.Background(), env); err != nil {
		t.Fatalf("publish with no subscribers should succeed, got: %v", err)
	}
}

func TestOrderedDelivery(t *testing.T) {
	bus := New()
	ch := bus.Subscribe("OrderTest")

	ids := []string{"a", "b", "c"}
	for _, id := range ids {
		env := makeEnvelope("OrderTest", id)
		env.ID = id
		if err := bus.Publish(context.Background(), env); err != nil {
			t.Fatalf("publish failed: %v", err)
		}
	}

	for _, wantID := range ids {
		select {
		case got := <-ch:
			if got.ID != wantID {
				t.Errorf("got ID %q, want %q", got.ID, wantID)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for event %q", wantID)
		}
	}
}

func TestNonBlockingPublish(t *testing.T) {
	bus := New()
	// Subscribe but never read — simulates slow consumer.
	_ = bus.Subscribe("SlowConsumer")

	// Fill the buffer completely.
	for i := range defaultBufferSize + 10 {
		env := makeEnvelope("SlowConsumer", "x1")
		env.ID = "evt-" + time.Now().Format("150405") + "-" + string(rune('a'+i%26))
		if err := bus.Publish(context.Background(), env); err != nil {
			t.Fatalf("publish should not block or error, got: %v", err)
		}
	}
	// If we get here without hanging, the test passes.
}

func TestUnknownEntityTypeRoutesNormally(t *testing.T) {
	bus := New()
	ch := bus.Subscribe("CustomEvent")

	env := EventEnvelope{
		ID:         "evt-custom",
		Type:       "CustomEvent",
		EntityID:   "x1",
		EntityType: EntityType("SomeNewType"),
		Payload:    json.RawMessage(`{}`),
		Source:     SourceReactive,
		Timestamp:  time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC),
	}
	if err := bus.Publish(context.Background(), env); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	select {
	case got := <-ch:
		if got.EntityType != EntityType("SomeNewType") {
			t.Errorf("got EntityType %q, want %q", got.EntityType, "SomeNewType")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestNoReplayOfHistoricalEvents(t *testing.T) {
	bus := New()

	// Publish before subscribing.
	env := makeEnvelope("ReplayTest", "x1")
	if err := bus.Publish(context.Background(), env); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	// Subscribe after the event was published.
	ch := bus.Subscribe("ReplayTest")

	// Should not receive the historical event.
	select {
	case got := <-ch:
		t.Fatalf("should not have received historical event, got %+v", got)
	case <-time.After(50 * time.Millisecond):
		// Expected: no event received.
	}

	// But should receive new events.
	env2 := makeEnvelope("ReplayTest", "x2")
	env2.ID = "evt-new"
	if err := bus.Publish(context.Background(), env2); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	select {
	case got := <-ch:
		if got.ID != "evt-new" {
			t.Errorf("got ID %q, want %q", got.ID, "evt-new")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for new event")
	}
}
