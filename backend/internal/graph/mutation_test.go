package graph

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/swatkatz/advisorhub/backend/internal/alert"
	"github.com/swatkatz/advisorhub/backend/internal/graph/model"
)

// Test Anchor 17: sendAlert delegates to AlertService.Send
func TestSendAlert(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	f.alertService.addAlert(&alert.Alert{
		ID: "al1", ClientID: "c1", Severity: alert.SeverityCritical,
		Status: alert.StatusOpen, Summary: "Over-contribution",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	mr := &mutationResolver{f.resolver}
	msg := "custom message"
	result, err := mr.SendAlert(ctx, "al1", &msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result.Status) != "ACTED" {
		t.Errorf("expected ACTED status, got %s", result.Status)
	}
	if len(f.alertService.sendCalls) != 1 {
		t.Errorf("expected 1 send call, got %d", len(f.alertService.sendCalls))
	}
	if *f.alertService.sendCalls[0].Message != "custom message" {
		t.Errorf("expected custom message, got %v", f.alertService.sendCalls[0].Message)
	}
}

// Test Anchor 18: trackAlert delegates to AlertService.Track
func TestTrackAlert(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	f.alertService.addAlert(&alert.Alert{
		ID: "al1", ClientID: "c1", Severity: alert.SeverityCritical,
		Status: alert.StatusOpen, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	mr := &mutationResolver{f.resolver}
	result, err := mr.TrackAlert(ctx, "al1", "Follow up")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result.Status) != "ACTED" {
		t.Errorf("expected ACTED status, got %s", result.Status)
	}
	if len(f.alertService.trackCalls) != 1 || f.alertService.trackCalls[0].ActionItemText != "Follow up" {
		t.Errorf("expected Track call with 'Follow up'")
	}
}

// Test Anchor 19: snoozeAlert delegates to AlertService.Snooze
func TestSnoozeAlert(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	f.alertService.addAlert(&alert.Alert{
		ID: "al1", ClientID: "c1", Severity: alert.SeverityCritical,
		Status: alert.StatusOpen, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	mr := &mutationResolver{f.resolver}
	result, err := mr.SnoozeAlert(ctx, "al1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result.Status) != "SNOOZED" {
		t.Errorf("expected SNOOZED status, got %s", result.Status)
	}
	if len(f.alertService.snoozeCalls) != 1 || f.alertService.snoozeCalls[0].Until != nil {
		t.Errorf("expected Snooze call with nil until")
	}
}

// Test Anchor 20: acknowledgeAlert transitions INFO to CLOSED
func TestAcknowledgeAlert(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	f.alertService.addAlert(&alert.Alert{
		ID: "al1", ClientID: "c1", Severity: alert.SeverityInfo,
		Status: alert.StatusOpen, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	mr := &mutationResolver{f.resolver}
	result, err := mr.AcknowledgeAlert(ctx, "al1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result.Status) != "CLOSED" {
		t.Errorf("expected CLOSED status, got %s", result.Status)
	}
}

// Test Anchor 21: sendAlert error propagates as GraphQL error
func TestSendAlert_ErrorPropagation(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	f.alertService.sendErr = errors.New("cannot send INFO alert")

	mr := &mutationResolver{f.resolver}
	_, err := mr.SendAlert(ctx, "al1", nil)
	if err == nil {
		t.Fatal("expected error from send")
	}
	if err.Error() != "cannot send INFO alert" {
		t.Errorf("expected 'cannot send INFO alert', got %s", err.Error())
	}
}

// Test Anchor 22: createActionItem delegates correctly
func TestCreateActionItem(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	alertID := "al1"
	mr := &mutationResolver{f.resolver}
	result, err := mr.CreateActionItem(ctx, model.CreateActionItemInput{
		ClientID: "c1",
		AlertID:  &alertID,
		Text:     "Follow up on RRSP",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "Follow up on RRSP" {
		t.Errorf("expected text 'Follow up on RRSP', got %s", result.Text)
	}
	if string(result.Status) != "PENDING" {
		t.Errorf("expected PENDING status, got %s", result.Status)
	}
}

// Test Anchor 23: updateActionItem delegates correctly
func TestUpdateActionItem(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	// Pre-create an item.
	item, _ := f.actionItemSvc.CreateActionItem(ctx, "c1", nil, "Original text", nil)

	newText := "Updated text"
	mr := &mutationResolver{f.resolver}
	result, err := mr.UpdateActionItem(ctx, item.ID, model.UpdateActionItemInput{
		Text: &newText,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "Updated text" {
		t.Errorf("expected 'Updated text', got %s", result.Text)
	}
}

// Test Anchor 24: updateActionItem error propagation
func TestUpdateActionItem_ErrorPropagation(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	f.actionItemSvc.updateErr = errors.New("invalid status transition")

	mr := &mutationResolver{f.resolver}
	_, err := mr.UpdateActionItem(ctx, "ai1", model.UpdateActionItemInput{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "invalid status transition" {
		t.Errorf("expected 'invalid status transition', got %s", err.Error())
	}
}

// Test Anchor 25: addNote with hardcoded advisor ID
func TestAddNote(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	mr := &mutationResolver{f.resolver}
	result, err := mr.AddNote(ctx, "c1", "Important note about RRSP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "Important note about RRSP" {
		t.Errorf("expected note text, got %s", result.Text)
	}

	// Verify the note was stored with advisor ID a1.
	notes, _ := f.noteRepo.GetNotes(ctx, "c1", "a1")
	if len(notes) != 1 {
		t.Fatalf("expected 1 note stored, got %d", len(notes))
	}
	if notes[0].AdvisorID != "a1" {
		t.Errorf("expected advisor ID a1, got %s", notes[0].AdvisorID)
	}
}
