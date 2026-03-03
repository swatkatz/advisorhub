package alert

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/swatkatz/advisorhub/backend/internal/actionitem"
	"github.com/swatkatz/advisorhub/backend/internal/eventbus"
)

// --- Test infrastructure ---

type mockEventBus struct {
	published []eventbus.EventEnvelope
}

func (b *mockEventBus) Publish(_ context.Context, env eventbus.EventEnvelope) error {
	b.published = append(b.published, env)
	return nil
}

func (b *mockEventBus) Subscribe(_ string) <-chan eventbus.EventEnvelope {
	return make(chan eventbus.EventEnvelope)
}

func (b *mockEventBus) lastEventOfType(t string) *eventbus.EventEnvelope {
	for i := len(b.published) - 1; i >= 0; i-- {
		if b.published[i].Type == t {
			return &b.published[i]
		}
	}
	return nil
}

type mockActionItemService struct {
	created []createdActionItem
	closed  []closedActionItem
	nextID  int
}

type createdActionItem struct {
	clientID string
	alertID  *string
	text     string
}

type closedActionItem struct {
	id             string
	resolutionNote string
}

func (s *mockActionItemService) CreateActionItem(_ context.Context, clientID string, alertID *string, text string, _ *time.Time) (*actionitem.ActionItem, error) {
	s.nextID++
	id := fmt.Sprintf("ai_%d", s.nextID)
	s.created = append(s.created, createdActionItem{clientID: clientID, alertID: alertID, text: text})
	return &actionitem.ActionItem{
		ID:       id,
		ClientID: clientID,
		AlertID:  alertID,
		Text:     text,
		Status:   actionitem.ActionItemStatusPending,
	}, nil
}

func (s *mockActionItemService) CloseActionItem(_ context.Context, id string, resolutionNote string) (*actionitem.ActionItem, error) {
	s.closed = append(s.closed, closedActionItem{id: id, resolutionNote: resolutionNote})
	return &actionitem.ActionItem{ID: id, Status: actionitem.ActionItemStatusClosed}, nil
}

func (s *mockActionItemService) GetActionItem(_ context.Context, id string) (*actionitem.ActionItem, error) {
	return &actionitem.ActionItem{ID: id}, nil
}

func (s *mockActionItemService) GetActionItemsByClientID(_ context.Context, _ string) ([]actionitem.ActionItem, error) {
	return nil, nil
}

func (s *mockActionItemService) GetActionItemsByAlertID(_ context.Context, _ string) ([]actionitem.ActionItem, error) {
	return nil, nil
}

func (s *mockActionItemService) UpdateActionItem(_ context.Context, _ string, _ *string, _ *actionitem.ActionItemStatus, _ *time.Time) (*actionitem.ActionItem, error) {
	return nil, nil
}

type failingEnhancer struct{}

func (e *failingEnhancer) Enhance(_ context.Context, _ *Alert) error {
	return errors.New("LLM error")
}

// --- Helpers ---

var testNow = time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC)

func fixedClock() time.Time { return testNow }

func newTestService() (*alertService, *MemoryAlertRepository, *mockEventBus, *mockActionItemService) {
	repo := NewMemoryAlertRepository()
	bus := &mockEventBus{}
	ais := &mockActionItemService{}
	enhancer := &StubEnhancer{}
	svc := NewAlertService(repo, bus, ais, enhancer, fixedClock).(*alertService)
	return svc, repo, bus, ais
}

func makeEvent(eventType string, payload map[string]any) eventbus.EventEnvelope {
	data, _ := json.Marshal(payload)
	return eventbus.EventEnvelope{
		ID:         fmt.Sprintf("evt_%s_test", eventType),
		Type:       eventType,
		EntityID:   getPayloadString(payload, "client_id"),
		EntityType: eventbus.EntityTypeClient,
		Payload:    data,
		Source:     eventbus.SourceReactive,
		Timestamp:  testNow,
	}
}

func getPayloadString(p map[string]any, key string) string {
	if v, ok := p[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func seedAlert(repo *MemoryAlertRepository, a *Alert) {
	repo.CreateAlert(context.Background(), a)
}

// --- Test Anchors 1-3: Event→Alert Mapping ---

func TestProcessEvent_Mapping(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		event        eventbus.EventEnvelope
		wantCreated  bool
		wantCategory string
		wantSeverity AlertSeverity
		wantCondKey  string
	}{
		{
			name: "1: OverContributionDetected creates CRITICAL alert",
			event: makeEvent(EventOverContributionDetected, map[string]any{
				"client_id":    "c1",
				"account_type": "RRSP",
				"excess":       2300.0,
			}),
			wantCreated:  true,
			wantCategory: "over_contribution",
			wantSeverity: SeverityCritical,
			wantCondKey:  "overcontrib:c1:RRSP",
		},
		{
			name: "2: DividendReceived creates INFO alert",
			event: makeEvent(EventDividendReceived, map[string]any{
				"client_id": "c9",
				"amount":    1240.0,
			}),
			wantCreated:  true,
			wantCategory: "dividend_received",
			wantSeverity: SeverityInfo,
			wantCondKey:  "dividend_received:c9",
		},
		{
			name: "3: Unrecognized event type is discarded",
			event: makeEvent("UnknownEventType", map[string]any{
				"client_id": "c1",
			}),
			wantCreated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo, _, _ := newTestService()
			err := svc.ProcessEvent(ctx, tt.event)
			if err != nil {
				t.Fatalf("ProcessEvent returned error: %v", err)
			}

			if !tt.wantCreated {
				alerts, _ := repo.GetAlertsByClientID(ctx, "c1")
				if len(alerts) != 0 {
					t.Errorf("expected no alert created, got %d", len(alerts))
				}
				return
			}

			clientID := getPayloadString(
				mustParsePayload(tt.event.Payload), "client_id")
			alerts, _ := repo.GetAlertsByClientID(ctx, clientID)
			if len(alerts) != 1 {
				t.Fatalf("expected 1 alert, got %d", len(alerts))
			}

			a := alerts[0]
			if a.Category != tt.wantCategory {
				t.Errorf("category = %q, want %q", a.Category, tt.wantCategory)
			}
			if a.Severity != tt.wantSeverity {
				t.Errorf("severity = %q, want %q", a.Severity, tt.wantSeverity)
			}
			if a.ConditionKey != tt.wantCondKey {
				t.Errorf("condition_key = %q, want %q", a.ConditionKey, tt.wantCondKey)
			}
			if a.Status != StatusOpen {
				t.Errorf("status = %q, want OPEN", a.Status)
			}
		})
	}
}

// --- Test Anchors 4-9: Dedup ---

func TestProcessEvent_Dedup(t *testing.T) {
	ctx := context.Background()

	t.Run("4: No existing alert creates new OPEN with signal CREATED", func(t *testing.T) {
		svc, repo, bus, _ := newTestService()
		event := makeEvent(EventOverContributionDetected, map[string]any{
			"client_id":    "c1",
			"account_type": "RRSP",
			"excess":       2300.0,
		})

		svc.ProcessEvent(ctx, event)

		alerts, _ := repo.GetAlertsByClientID(ctx, "c1")
		if len(alerts) != 1 {
			t.Fatalf("expected 1 alert, got %d", len(alerts))
		}
		if alerts[0].Status != StatusOpen {
			t.Errorf("status = %q, want OPEN", alerts[0].Status)
		}
		// Signal CREATED → AlertCreated event emitted
		if bus.lastEventOfType(EventAlertCreated) == nil {
			t.Error("expected AlertCreated event to be emitted")
		}
	})

	t.Run("5: Existing OPEN alert updates payload", func(t *testing.T) {
		svc, repo, _, _ := newTestService()

		// First event
		event1 := makeEvent(EventOverContributionDetected, map[string]any{
			"client_id":    "c1",
			"account_type": "RRSP",
			"excess":       2300.0,
		})
		svc.ProcessEvent(ctx, event1)

		alerts, _ := repo.GetAlertsByClientID(ctx, "c1")
		originalID := alerts[0].ID

		// Second event with updated excess
		event2 := makeEvent(EventOverContributionDetected, map[string]any{
			"client_id":    "c1",
			"account_type": "RRSP",
			"excess":       3000.0,
		})
		svc.ProcessEvent(ctx, event2)

		alerts, _ = repo.GetAlertsByClientID(ctx, "c1")
		if len(alerts) != 1 {
			t.Fatalf("expected 1 alert (dedup), got %d", len(alerts))
		}
		if alerts[0].ID != originalID {
			t.Error("expected same alert ID (dedup)")
		}

		var payload map[string]any
		json.Unmarshal(alerts[0].Payload, &payload)
		if payload["excess"].(float64) != 3000.0 {
			t.Errorf("payload not updated, excess = %v", payload["excess"])
		}
	})

	t.Run("6: SNOOZED alert with expired snooze reopens to OPEN", func(t *testing.T) {
		svc, repo, bus, _ := newTestService()

		pastSnooze := testNow.Add(-1 * time.Hour)
		seedAlert(repo, &Alert{
			ID:                  "a_snoozed_expired",
			ConditionKey:        "transfer_stuck:t1",
			ClientID:            "c8",
			Severity:            SeverityCritical,
			Category:            "transfer_stuck",
			Status:              StatusSnoozed,
			SnoozedUntil:        &pastSnooze,
			Payload:             json.RawMessage(`{"transfer_id":"t1","client_id":"c8"}`),
			LinkedActionItemIDs: []string{},
			CreatedAt:           testNow.Add(-24 * time.Hour),
			UpdatedAt:           testNow.Add(-24 * time.Hour),
		})

		event := makeEvent(EventTransferStuck, map[string]any{
			"transfer_id": "t1",
			"client_id":   "c8",
			"days_in_stage": 20,
		})
		svc.ProcessEvent(ctx, event)

		a, _ := repo.GetAlert(ctx, "a_snoozed_expired")
		if a.Status != StatusOpen {
			t.Errorf("status = %q, want OPEN (reopened)", a.Status)
		}
		if a.SnoozedUntil != nil {
			t.Error("snoozed_until should be nil after reopening")
		}
		// Signal REOPENED → AlertUpdated emitted
		evt := bus.lastEventOfType(EventAlertUpdated)
		if evt == nil {
			t.Error("expected AlertUpdated event for REOPENED")
		}
	})

	t.Run("7: SNOOZED alert with future snooze updates silently", func(t *testing.T) {
		svc, repo, bus, _ := newTestService()

		futureSnooze := testNow.Add(24 * time.Hour)
		seedAlert(repo, &Alert{
			ID:                  "a_snoozed_future",
			ConditionKey:        "transfer_stuck:t1",
			ClientID:            "c8",
			Severity:            SeverityCritical,
			Category:            "transfer_stuck",
			Status:              StatusSnoozed,
			SnoozedUntil:        &futureSnooze,
			Payload:             json.RawMessage(`{"transfer_id":"t1","client_id":"c8"}`),
			LinkedActionItemIDs: []string{},
			CreatedAt:           testNow.Add(-24 * time.Hour),
			UpdatedAt:           testNow.Add(-24 * time.Hour),
		})

		event := makeEvent(EventTransferStuck, map[string]any{
			"transfer_id":   "t1",
			"client_id":     "c8",
			"days_in_stage": 22,
		})
		svc.ProcessEvent(ctx, event)

		a, _ := repo.GetAlert(ctx, "a_snoozed_future")
		if a.Status != StatusSnoozed {
			t.Errorf("status = %q, want SNOOZED", a.Status)
		}
		// No event emitted for silent update
		if len(bus.published) != 0 {
			t.Errorf("expected no events emitted for silent update, got %d", len(bus.published))
		}
	})

	t.Run("8: ACTED alert updates payload silently", func(t *testing.T) {
		svc, repo, bus, _ := newTestService()

		seedAlert(repo, &Alert{
			ID:                  "a_acted",
			ConditionKey:        "cesg_gap:c1:resp_ben_1:2026",
			ClientID:            "c1",
			Severity:            SeverityUrgent,
			Category:            "cesg_gap",
			Status:              StatusActed,
			Payload:             json.RawMessage(`{"client_id":"c1","beneficiary_id":"resp_ben_1","tax_year":2026}`),
			LinkedActionItemIDs: []string{"ai_1"},
			CreatedAt:           testNow.Add(-24 * time.Hour),
			UpdatedAt:           testNow.Add(-24 * time.Hour),
		})

		event := makeEvent(EventCESGGap, map[string]any{
			"client_id":      "c1",
			"beneficiary_id": "resp_ben_1",
			"tax_year":       2026,
			"gap_amount":     700.0,
		})
		svc.ProcessEvent(ctx, event)

		a, _ := repo.GetAlert(ctx, "a_acted")
		if a.Status != StatusActed {
			t.Errorf("status = %q, want ACTED", a.Status)
		}
		// No event emitted for silent update
		if len(bus.published) != 0 {
			t.Errorf("expected no events for ACTED silent update, got %d", len(bus.published))
		}
	})

	t.Run("9: CLOSED alert with same condition_key creates new alert", func(t *testing.T) {
		svc, repo, _, _ := newTestService()

		resolvedAt := testNow.Add(-48 * time.Hour)
		seedAlert(repo, &Alert{
			ID:                  "a_closed",
			ConditionKey:        "engagement_stale:c5",
			ClientID:            "c5",
			Severity:            SeverityAdvisory,
			Category:            "engagement_stale",
			Status:              StatusClosed,
			Payload:             json.RawMessage(`{"client_id":"c5"}`),
			LinkedActionItemIDs: []string{},
			CreatedAt:           testNow.Add(-72 * time.Hour),
			UpdatedAt:           testNow.Add(-48 * time.Hour),
			ResolvedAt:          &resolvedAt,
		})

		event := makeEvent(EventEngagementStale, map[string]any{
			"client_id":  "c5",
			"days_since": 190,
		})
		svc.ProcessEvent(ctx, event)

		alerts, _ := repo.GetAlertsByClientID(ctx, "c5")
		if len(alerts) != 2 {
			t.Fatalf("expected 2 alerts (old closed + new), got %d", len(alerts))
		}
		var newAlert *Alert
		for i := range alerts {
			if alerts[i].ID != "a_closed" {
				newAlert = &alerts[i]
			}
		}
		if newAlert == nil {
			t.Fatal("expected a new alert separate from the closed one")
		}
		if newAlert.Status != StatusOpen {
			t.Errorf("new alert status = %q, want OPEN", newAlert.Status)
		}
	})
}

// --- Test Anchors 10-14: Enhancement ---

func TestProcessEvent_Enhancement(t *testing.T) {
	ctx := context.Background()

	t.Run("10: CREATED signal triggers enhancer with summary+draft", func(t *testing.T) {
		svc, repo, _, _ := newTestService()

		event := makeEvent(EventOverContributionDetected, map[string]any{
			"client_id":    "c1",
			"account_type": "RRSP",
			"excess":       2300.0,
		})
		svc.ProcessEvent(ctx, event)

		alerts, _ := repo.GetAlertsByClientID(ctx, "c1")
		a := alerts[0]
		expectedSummary := fmt.Sprintf("enhanced:%s", a.ID)
		if a.Summary != expectedSummary {
			t.Errorf("summary = %q, want %q", a.Summary, expectedSummary)
		}
		if a.DraftMessage == nil {
			t.Fatal("draft_message should be set for over_contribution")
		}
		expectedDraft := fmt.Sprintf("draft:%s", a.ID)
		if *a.DraftMessage != expectedDraft {
			t.Errorf("draft_message = %q, want %q", *a.DraftMessage, expectedDraft)
		}
	})

	t.Run("11: REOPENED signal triggers enhancer", func(t *testing.T) {
		svc, repo, _, _ := newTestService()

		pastSnooze := testNow.Add(-1 * time.Hour)
		seedAlert(repo, &Alert{
			ID:                  "a_reopen",
			ConditionKey:        "transfer_stuck:t1",
			ClientID:            "c8",
			Severity:            SeverityCritical,
			Category:            "transfer_stuck",
			Status:              StatusSnoozed,
			SnoozedUntil:        &pastSnooze,
			Payload:             json.RawMessage(`{"transfer_id":"t1","client_id":"c8"}`),
			LinkedActionItemIDs: []string{},
			CreatedAt:           testNow.Add(-24 * time.Hour),
			UpdatedAt:           testNow.Add(-24 * time.Hour),
		})

		event := makeEvent(EventTransferStuck, map[string]any{
			"transfer_id":   "t1",
			"client_id":     "c8",
			"days_in_stage": 22,
		})
		svc.ProcessEvent(ctx, event)

		a, _ := repo.GetAlert(ctx, "a_reopen")
		expectedSummary := "enhanced:a_reopen"
		if a.Summary != expectedSummary {
			t.Errorf("summary = %q, want %q", a.Summary, expectedSummary)
		}
	})

	t.Run("12: UPDATED signal does not trigger enhancer", func(t *testing.T) {
		svc, repo, _, _ := newTestService()

		// Create initial alert
		event := makeEvent(EventOverContributionDetected, map[string]any{
			"client_id":    "c1",
			"account_type": "RRSP",
			"excess":       2300.0,
		})
		svc.ProcessEvent(ctx, event)

		alerts, _ := repo.GetAlertsByClientID(ctx, "c1")
		originalSummary := alerts[0].Summary

		// Update with new data
		event2 := makeEvent(EventOverContributionDetected, map[string]any{
			"client_id":    "c1",
			"account_type": "RRSP",
			"excess":       3000.0,
		})
		svc.ProcessEvent(ctx, event2)

		alerts, _ = repo.GetAlertsByClientID(ctx, "c1")
		if alerts[0].Summary != originalSummary {
			t.Errorf("summary changed on UPDATED (should not re-enhance)")
		}
	})

	t.Run("13: INFO alert gets summary but no draft_message", func(t *testing.T) {
		svc, repo, _, _ := newTestService()

		event := makeEvent(EventDividendReceived, map[string]any{
			"client_id": "c9",
			"amount":    1240.0,
		})
		svc.ProcessEvent(ctx, event)

		alerts, _ := repo.GetAlertsByClientID(ctx, "c9")
		a := alerts[0]
		if a.Summary == "" {
			t.Error("summary should be set for INFO alerts")
		}
		if a.DraftMessage != nil {
			t.Errorf("draft_message should be nil for INFO alerts, got %q", *a.DraftMessage)
		}
	})

	t.Run("14: Enhancer failure leaves alert with empty summary", func(t *testing.T) {
		repo := NewMemoryAlertRepository()
		bus := &mockEventBus{}
		ais := &mockActionItemService{}
		svc := NewAlertService(repo, bus, ais, &failingEnhancer{}, fixedClock)

		event := makeEvent(EventOverContributionDetected, map[string]any{
			"client_id":    "c1",
			"account_type": "RRSP",
			"excess":       2300.0,
		})
		err := svc.ProcessEvent(ctx, event)
		if err != nil {
			t.Fatalf("ProcessEvent should not return error when enhancer fails: %v", err)
		}

		alerts, _ := repo.GetAlertsByClientID(ctx, "c1")
		if len(alerts) != 1 {
			t.Fatalf("expected 1 alert, got %d", len(alerts))
		}
		if alerts[0].Summary != "" {
			t.Errorf("summary should be empty after enhancer failure, got %q", alerts[0].Summary)
		}
		if alerts[0].DraftMessage != nil {
			t.Errorf("draft_message should be nil after enhancer failure")
		}
	})
}

// --- Test Anchors 15-18: Send ---

func TestSend(t *testing.T) {
	ctx := context.Background()

	t.Run("15: Send with nil message uses draft_message, transitions to SNOOZED", func(t *testing.T) {
		svc, repo, bus, ais := newTestService()

		draft := "Please review your RRSP contributions."
		seedAlert(repo, &Alert{
			ID:                  "a_send",
			ConditionKey:        "overcontrib:c1:RRSP",
			ClientID:            "c1",
			Severity:            SeverityCritical,
			Category:            "over_contribution",
			Status:              StatusOpen,
			DraftMessage:        &draft,
			Payload:             json.RawMessage(`{}`),
			LinkedActionItemIDs: []string{},
			CreatedAt:           testNow.Add(-1 * time.Hour),
			UpdatedAt:           testNow.Add(-1 * time.Hour),
		})

		result, err := svc.Send(ctx, "a_send", nil)
		if err != nil {
			t.Fatalf("Send returned error: %v", err)
		}

		if result.Status != StatusSnoozed {
			t.Errorf("status = %q, want SNOOZED", result.Status)
		}
		if len(result.LinkedActionItemIDs) != 1 {
			t.Fatalf("expected 1 linked action item, got %d", len(result.LinkedActionItemIDs))
		}

		expectedSnooze := testNow.Add(7 * 24 * time.Hour)
		if result.SnoozedUntil == nil || !result.SnoozedUntil.Equal(expectedSnooze) {
			t.Errorf("snoozed_until = %v, want %v", result.SnoozedUntil, expectedSnooze)
		}

		if len(ais.created) != 1 {
			t.Fatalf("expected 1 action item created, got %d", len(ais.created))
		}
		if ais.created[0].text != draft {
			t.Errorf("action item text = %q, want %q", ais.created[0].text, draft)
		}

		if bus.lastEventOfType(EventAlertUpdated) == nil {
			t.Error("expected AlertUpdated event")
		}
	})

	t.Run("16: Send with custom message uses it instead of draft", func(t *testing.T) {
		svc, repo, _, ais := newTestService()

		draft := "original draft"
		seedAlert(repo, &Alert{
			ID:                  "a_send_custom",
			ConditionKey:        "overcontrib:c1:TFSA",
			ClientID:            "c1",
			Severity:            SeverityCritical,
			Category:            "over_contribution",
			Status:              StatusOpen,
			DraftMessage:        &draft,
			Payload:             json.RawMessage(`{}`),
			LinkedActionItemIDs: []string{},
			CreatedAt:           testNow,
			UpdatedAt:           testNow,
		})

		customMsg := "custom message"
		svc.Send(ctx, "a_send_custom", &customMsg)

		if len(ais.created) != 1 {
			t.Fatalf("expected 1 action item, got %d", len(ais.created))
		}
		if ais.created[0].text != "custom message" {
			t.Errorf("action item text = %q, want %q", ais.created[0].text, "custom message")
		}
	})

	t.Run("17: Send on CLOSED alert returns error", func(t *testing.T) {
		svc, repo, _, _ := newTestService()

		resolvedAt := testNow
		seedAlert(repo, &Alert{
			ID:                  "a_closed_send",
			ConditionKey:        "overcontrib:c1:RRSP",
			ClientID:            "c1",
			Severity:            SeverityCritical,
			Category:            "over_contribution",
			Status:              StatusClosed,
			Payload:             json.RawMessage(`{}`),
			LinkedActionItemIDs: []string{},
			CreatedAt:           testNow,
			UpdatedAt:           testNow,
			ResolvedAt:          &resolvedAt,
		})

		_, err := svc.Send(ctx, "a_closed_send", nil)
		if !errors.Is(err, ErrAlertClosed) {
			t.Errorf("expected ErrAlertClosed, got %v", err)
		}
	})

	t.Run("18: Send on INFO alert returns error", func(t *testing.T) {
		svc, repo, _, _ := newTestService()

		seedAlert(repo, &Alert{
			ID:                  "a_info_send",
			ConditionKey:        "dividend_received:c9",
			ClientID:            "c9",
			Severity:            SeverityInfo,
			Category:            "dividend_received",
			Status:              StatusOpen,
			Payload:             json.RawMessage(`{}`),
			LinkedActionItemIDs: []string{},
			CreatedAt:           testNow,
			UpdatedAt:           testNow,
		})

		_, err := svc.Send(ctx, "a_info_send", nil)
		if !errors.Is(err, ErrInfoAlert) {
			t.Errorf("expected ErrInfoAlert, got %v", err)
		}
	})
}

// --- Test Anchors 19-20: Track ---

func TestTrack(t *testing.T) {
	ctx := context.Background()

	t.Run("19: Track creates action item and transitions to SNOOZED", func(t *testing.T) {
		svc, repo, bus, ais := newTestService()

		seedAlert(repo, &Alert{
			ID:                  "a_track",
			ConditionKey:        "deadline_approaching:c3:RRSP:2026",
			ClientID:            "c3",
			Severity:            SeverityUrgent,
			Category:            "deadline_approaching",
			Status:              StatusOpen,
			Payload:             json.RawMessage(`{}`),
			LinkedActionItemIDs: []string{},
			CreatedAt:           testNow,
			UpdatedAt:           testNow,
		})

		result, err := svc.Track(ctx, "a_track", "Follow up on RRSP contribution")
		if err != nil {
			t.Fatalf("Track returned error: %v", err)
		}

		if result.Status != StatusSnoozed {
			t.Errorf("status = %q, want SNOOZED", result.Status)
		}
		expectedSnooze := testNow.Add(3 * 24 * time.Hour)
		if result.SnoozedUntil == nil || !result.SnoozedUntil.Equal(expectedSnooze) {
			t.Errorf("snoozed_until = %v, want %v", result.SnoozedUntil, expectedSnooze)
		}
		if len(ais.created) != 1 || ais.created[0].text != "Follow up on RRSP contribution" {
			t.Error("expected action item with correct text")
		}
		if bus.lastEventOfType(EventAlertUpdated) == nil {
			t.Error("expected AlertUpdated event")
		}
	})

	t.Run("20: Track on SNOOZED alert resets snooze timer", func(t *testing.T) {
		svc, repo, _, _ := newTestService()

		futureSnooze := testNow.Add(1 * time.Hour)
		seedAlert(repo, &Alert{
			ID:                  "a_track_snoozed",
			ConditionKey:        "deadline_approaching:c3:RRSP:2026",
			ClientID:            "c3",
			Severity:            SeverityUrgent,
			Category:            "deadline_approaching",
			Status:              StatusSnoozed,
			SnoozedUntil:        &futureSnooze,
			Payload:             json.RawMessage(`{}`),
			LinkedActionItemIDs: []string{},
			CreatedAt:           testNow,
			UpdatedAt:           testNow,
		})

		result, err := svc.Track(ctx, "a_track_snoozed", "Additional follow up")
		if err != nil {
			t.Fatalf("Track returned error: %v", err)
		}

		if result.Status != StatusSnoozed {
			t.Errorf("status = %q, want SNOOZED", result.Status)
		}
		// Snooze duration resets to category default (3 days for deadline_approaching)
		expectedSnooze := testNow.Add(3 * 24 * time.Hour)
		if !result.SnoozedUntil.Equal(expectedSnooze) {
			t.Errorf("snoozed_until = %v, want %v (reset to category default)", result.SnoozedUntil, expectedSnooze)
		}
	})
}

// --- Test Anchors 21-23: Snooze ---

func TestSnooze(t *testing.T) {
	ctx := context.Background()

	t.Run("21: Snooze with nil uses category default", func(t *testing.T) {
		svc, repo, bus, _ := newTestService()

		seedAlert(repo, &Alert{
			ID:                  "a_snooze",
			ConditionKey:        "transfer_stuck:t1",
			ClientID:            "c8",
			Severity:            SeverityCritical,
			Category:            "transfer_stuck",
			Status:              StatusOpen,
			Payload:             json.RawMessage(`{}`),
			LinkedActionItemIDs: []string{},
			CreatedAt:           testNow,
			UpdatedAt:           testNow,
		})

		result, err := svc.Snooze(ctx, "a_snooze", nil)
		if err != nil {
			t.Fatalf("Snooze returned error: %v", err)
		}

		if result.Status != StatusSnoozed {
			t.Errorf("status = %q, want SNOOZED", result.Status)
		}
		expectedSnooze := testNow.Add(5 * 24 * time.Hour)
		if result.SnoozedUntil == nil || !result.SnoozedUntil.Equal(expectedSnooze) {
			t.Errorf("snoozed_until = %v, want %v", result.SnoozedUntil, expectedSnooze)
		}
		if bus.lastEventOfType(EventAlertUpdated) == nil {
			t.Error("expected AlertUpdated event")
		}
	})

	t.Run("22: Snooze with specific time uses it", func(t *testing.T) {
		svc, repo, _, _ := newTestService()

		seedAlert(repo, &Alert{
			ID:                  "a_snooze_specific",
			ConditionKey:        "transfer_stuck:t2",
			ClientID:            "c8",
			Severity:            SeverityCritical,
			Category:            "transfer_stuck",
			Status:              StatusOpen,
			Payload:             json.RawMessage(`{}`),
			LinkedActionItemIDs: []string{},
			CreatedAt:           testNow,
			UpdatedAt:           testNow,
		})

		specificTime := testNow.Add(10 * 24 * time.Hour)
		result, _ := svc.Snooze(ctx, "a_snooze_specific", &specificTime)

		if result.SnoozedUntil == nil || !result.SnoozedUntil.Equal(specificTime) {
			t.Errorf("snoozed_until = %v, want %v", result.SnoozedUntil, specificTime)
		}
	})

	t.Run("23: Snooze on CLOSED alert returns error", func(t *testing.T) {
		svc, repo, _, _ := newTestService()

		resolvedAt := testNow
		seedAlert(repo, &Alert{
			ID:                  "a_snooze_closed",
			ConditionKey:        "transfer_stuck:t3",
			ClientID:            "c8",
			Severity:            SeverityCritical,
			Category:            "transfer_stuck",
			Status:              StatusClosed,
			Payload:             json.RawMessage(`{}`),
			LinkedActionItemIDs: []string{},
			CreatedAt:           testNow,
			UpdatedAt:           testNow,
			ResolvedAt:          &resolvedAt,
		})

		_, err := svc.Snooze(ctx, "a_snooze_closed", nil)
		if !errors.Is(err, ErrAlertClosed) {
			t.Errorf("expected ErrAlertClosed, got %v", err)
		}
	})
}

// --- Test Anchors 24-26: Acknowledge ---

func TestAcknowledge(t *testing.T) {
	ctx := context.Background()

	t.Run("24: Acknowledge closes INFO alert", func(t *testing.T) {
		svc, repo, bus, _ := newTestService()

		seedAlert(repo, &Alert{
			ID:                  "a_ack",
			ConditionKey:        "dividend_received:c9",
			ClientID:            "c9",
			Severity:            SeverityInfo,
			Category:            "dividend_received",
			Status:              StatusOpen,
			Payload:             json.RawMessage(`{}`),
			LinkedActionItemIDs: []string{},
			CreatedAt:           testNow,
			UpdatedAt:           testNow,
		})

		result, err := svc.Acknowledge(ctx, "a_ack")
		if err != nil {
			t.Fatalf("Acknowledge returned error: %v", err)
		}

		if result.Status != StatusClosed {
			t.Errorf("status = %q, want CLOSED", result.Status)
		}
		if result.ResolvedAt == nil {
			t.Error("resolved_at should be set")
		}
		if bus.lastEventOfType(EventAlertClosed) == nil {
			t.Error("expected AlertClosed event")
		}
	})

	t.Run("25: Acknowledge on CRITICAL alert returns error", func(t *testing.T) {
		svc, repo, _, _ := newTestService()

		seedAlert(repo, &Alert{
			ID:                  "a_ack_critical",
			ConditionKey:        "overcontrib:c1:RRSP",
			ClientID:            "c1",
			Severity:            SeverityCritical,
			Category:            "over_contribution",
			Status:              StatusOpen,
			Payload:             json.RawMessage(`{}`),
			LinkedActionItemIDs: []string{},
			CreatedAt:           testNow,
			UpdatedAt:           testNow,
		})

		_, err := svc.Acknowledge(ctx, "a_ack_critical")
		if !errors.Is(err, ErrNotInfoAlert) {
			t.Errorf("expected ErrNotInfoAlert, got %v", err)
		}
	})

	t.Run("26: Acknowledge on already-CLOSED INFO alert returns error", func(t *testing.T) {
		svc, repo, _, _ := newTestService()

		resolvedAt := testNow
		seedAlert(repo, &Alert{
			ID:                  "a_ack_closed",
			ConditionKey:        "dividend_received:c9",
			ClientID:            "c9",
			Severity:            SeverityInfo,
			Category:            "dividend_received",
			Status:              StatusClosed,
			Payload:             json.RawMessage(`{}`),
			LinkedActionItemIDs: []string{},
			CreatedAt:           testNow,
			UpdatedAt:           testNow,
			ResolvedAt:          &resolvedAt,
		})

		_, err := svc.Acknowledge(ctx, "a_ack_closed")
		if !errors.Is(err, ErrAlertClosed) {
			t.Errorf("expected ErrAlertClosed, got %v", err)
		}
	})
}

// --- Test Anchors 27-29: Close (cascade) ---

func TestClose(t *testing.T) {
	ctx := context.Background()

	t.Run("27: Close cascades to linked ActionItems", func(t *testing.T) {
		svc, repo, bus, ais := newTestService()

		seedAlert(repo, &Alert{
			ID:                  "a_close_cascade",
			ConditionKey:        "overcontrib:c1:RRSP",
			ClientID:            "c1",
			Severity:            SeverityCritical,
			Category:            "over_contribution",
			Status:              StatusOpen,
			Payload:             json.RawMessage(`{}`),
			LinkedActionItemIDs: []string{"ai_100", "ai_101"},
			CreatedAt:           testNow,
			UpdatedAt:           testNow,
		})

		result, err := svc.Close(ctx, "a_close_cascade")
		if err != nil {
			t.Fatalf("Close returned error: %v", err)
		}

		if result.Status != StatusClosed {
			t.Errorf("status = %q, want CLOSED", result.Status)
		}
		if result.ResolvedAt == nil {
			t.Error("resolved_at should be set")
		}

		if len(ais.closed) != 2 {
			t.Fatalf("expected 2 action items closed, got %d", len(ais.closed))
		}
		if ais.closed[0].id != "ai_100" || ais.closed[1].id != "ai_101" {
			t.Error("expected ai_100 and ai_101 to be closed")
		}
		for _, c := range ais.closed {
			if c.resolutionNote == "" {
				t.Error("resolution note should not be empty")
			}
		}

		if bus.lastEventOfType(EventAlertClosed) == nil {
			t.Error("expected AlertClosed event")
		}
	})

	t.Run("28: Close without linked ActionItems", func(t *testing.T) {
		svc, repo, _, ais := newTestService()

		seedAlert(repo, &Alert{
			ID:                  "a_close_no_ai",
			ConditionKey:        "engagement_stale:c5",
			ClientID:            "c5",
			Severity:            SeverityAdvisory,
			Category:            "engagement_stale",
			Status:              StatusOpen,
			Payload:             json.RawMessage(`{}`),
			LinkedActionItemIDs: []string{},
			CreatedAt:           testNow,
			UpdatedAt:           testNow,
		})

		result, err := svc.Close(ctx, "a_close_no_ai")
		if err != nil {
			t.Fatalf("Close returned error: %v", err)
		}

		if result.Status != StatusClosed {
			t.Errorf("status = %q, want CLOSED", result.Status)
		}
		if len(ais.closed) != 0 {
			t.Errorf("expected no cascade calls, got %d", len(ais.closed))
		}
	})

	t.Run("29: Close on already-CLOSED alert is idempotent", func(t *testing.T) {
		svc, repo, bus, _ := newTestService()

		resolvedAt := testNow
		seedAlert(repo, &Alert{
			ID:                  "a_close_idempotent",
			ConditionKey:        "engagement_stale:c5",
			ClientID:            "c5",
			Severity:            SeverityAdvisory,
			Category:            "engagement_stale",
			Status:              StatusClosed,
			Payload:             json.RawMessage(`{}`),
			LinkedActionItemIDs: []string{},
			CreatedAt:           testNow,
			UpdatedAt:           testNow,
			ResolvedAt:          &resolvedAt,
		})

		result, err := svc.Close(ctx, "a_close_idempotent")
		if err != nil {
			t.Fatalf("Close on already-closed should not error: %v", err)
		}
		if result.Status != StatusClosed {
			t.Errorf("status = %q, want CLOSED", result.Status)
		}
		if len(bus.published) != 0 {
			t.Errorf("expected no events for idempotent close, got %d", len(bus.published))
		}
	})
}

// --- Test Anchors 30-34: Health Status ---

func TestComputeHealthStatus(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		alerts   []Alert
		clientID string
		want     HealthStatus
	}{
		{
			name: "30: CRITICAL alert → RED",
			alerts: []Alert{
				{ID: "a1", ClientID: "c1", Severity: SeverityCritical, Status: StatusOpen},
				{ID: "a2", ClientID: "c1", Severity: SeverityAdvisory, Status: StatusOpen},
			},
			clientID: "c1",
			want:     HealthRed,
		},
		{
			name: "31: ADVISORY only → YELLOW",
			alerts: []Alert{
				{ID: "a3", ClientID: "c5", Severity: SeverityAdvisory, Status: StatusOpen},
			},
			clientID: "c5",
			want:     HealthYellow,
		},
		{
			name: "32: Only CLOSED alerts → GREEN",
			alerts: []Alert{
				{ID: "a4", ClientID: "c6", Severity: SeverityCritical, Status: StatusClosed},
			},
			clientID: "c6",
			want:     HealthGreen,
		},
		{
			name: "33: INFO only non-CLOSED → GREEN",
			alerts: []Alert{
				{ID: "a5", ClientID: "c6", Severity: SeverityInfo, Status: StatusOpen},
			},
			clientID: "c6",
			want:     HealthGreen,
		},
		{
			name:     "34: No alerts → GREEN",
			alerts:   []Alert{},
			clientID: "c7",
			want:     HealthGreen,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo, _, _ := newTestService()
			for i := range tt.alerts {
				seedAlert(repo, &tt.alerts[i])
			}

			got, err := svc.ComputeHealthStatus(ctx, tt.clientID)
			if err != nil {
				t.Fatalf("ComputeHealthStatus error: %v", err)
			}
			if got != tt.want {
				t.Errorf("health = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- Test Anchors 35-37: Events Emitted ---

func TestEventsEmitted(t *testing.T) {
	ctx := context.Background()

	t.Run("35: ProcessEvent emits AlertCreated with correct fields", func(t *testing.T) {
		svc, _, bus, _ := newTestService()

		event := makeEvent(EventOverContributionDetected, map[string]any{
			"client_id":    "c1",
			"account_type": "RRSP",
			"excess":       2300.0,
		})
		svc.ProcessEvent(ctx, event)

		evt := bus.lastEventOfType(EventAlertCreated)
		if evt == nil {
			t.Fatal("expected AlertCreated event")
		}
		if evt.Source != eventbus.SourceSystem {
			t.Errorf("source = %q, want SYSTEM", evt.Source)
		}

		var payload map[string]any
		json.Unmarshal(evt.Payload, &payload)
		if payload["client_id"] != "c1" {
			t.Errorf("payload client_id = %v, want c1", payload["client_id"])
		}
		if payload["severity"] != "CRITICAL" {
			t.Errorf("payload severity = %v, want CRITICAL", payload["severity"])
		}
		if payload["category"] != "over_contribution" {
			t.Errorf("payload category = %v, want over_contribution", payload["category"])
		}
		if payload["condition_key"] != "overcontrib:c1:RRSP" {
			t.Errorf("payload condition_key = %v, want overcontrib:c1:RRSP", payload["condition_key"])
		}
	})

	t.Run("36: Send emits AlertUpdated", func(t *testing.T) {
		svc, repo, bus, _ := newTestService()

		draft := "msg"
		seedAlert(repo, &Alert{
			ID:                  "a_evt_send",
			ConditionKey:        "overcontrib:c1:RRSP",
			ClientID:            "c1",
			Severity:            SeverityCritical,
			Category:            "over_contribution",
			Status:              StatusOpen,
			DraftMessage:        &draft,
			Payload:             json.RawMessage(`{}`),
			LinkedActionItemIDs: []string{},
			CreatedAt:           testNow,
			UpdatedAt:           testNow,
		})

		svc.Send(ctx, "a_evt_send", nil)

		evt := bus.lastEventOfType(EventAlertUpdated)
		if evt == nil {
			t.Fatal("expected AlertUpdated event after Send")
		}
		if evt.Source != eventbus.SourceSystem {
			t.Errorf("source = %q, want SYSTEM", evt.Source)
		}
	})

	t.Run("37: Close emits AlertClosed", func(t *testing.T) {
		svc, repo, bus, _ := newTestService()

		seedAlert(repo, &Alert{
			ID:                  "a_evt_close",
			ConditionKey:        "engagement_stale:c5",
			ClientID:            "c5",
			Severity:            SeverityAdvisory,
			Category:            "engagement_stale",
			Status:              StatusOpen,
			Payload:             json.RawMessage(`{}`),
			LinkedActionItemIDs: []string{},
			CreatedAt:           testNow,
			UpdatedAt:           testNow,
		})

		svc.Close(ctx, "a_evt_close")

		evt := bus.lastEventOfType(EventAlertClosed)
		if evt == nil {
			t.Fatal("expected AlertClosed event after Close")
		}
		if evt.Source != eventbus.SourceSystem {
			t.Errorf("source = %q, want SYSTEM", evt.Source)
		}
	})
}

// --- Helpers ---

func mustParsePayload(data json.RawMessage) map[string]any {
	var m map[string]any
	json.Unmarshal(data, &m)
	return m
}
