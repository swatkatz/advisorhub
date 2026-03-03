package actionitem

import (
	"context"
	"testing"
	"time"
)

// fixedClock returns a function that always returns the given time.
func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func newTestService() (*Service, *MemoryActionItemRepo) {
	repo := newMemoryActionItemRepo()
	svc := NewService(repo)
	return svc, repo
}

// --- Test Anchor 1: Create with all fields ---

func TestCreateActionItem_AllFields(t *testing.T) {
	svc, _ := newTestService()
	now := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC)
	svc.clock = fixedClock(now)
	ctx := context.Background()

	alertID := "alert-1"
	dueDate := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	item, err := svc.CreateActionItem(ctx, "c1", &alertID, "Follow up on RRSP", &dueDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if item.ID == "" {
		t.Error("expected non-empty ID")
	}
	if item.ClientID != "c1" {
		t.Errorf("expected client_id c1, got %s", item.ClientID)
	}
	if item.AlertID == nil || *item.AlertID != "alert-1" {
		t.Errorf("expected alert_id alert-1, got %v", item.AlertID)
	}
	if item.Text != "Follow up on RRSP" {
		t.Errorf("expected text 'Follow up on RRSP', got %s", item.Text)
	}
	if item.Status != ActionItemStatusPending {
		t.Errorf("expected status PENDING, got %s", item.Status)
	}
	if item.DueDate == nil || !item.DueDate.Equal(dueDate) {
		t.Errorf("expected due_date %v, got %v", dueDate, item.DueDate)
	}
	if !item.CreatedAt.Equal(now) {
		t.Errorf("expected created_at %v, got %v", now, item.CreatedAt)
	}
	if item.ResolvedAt != nil {
		t.Errorf("expected resolved_at nil, got %v", item.ResolvedAt)
	}
	if item.ResolutionNote != nil {
		t.Errorf("expected resolution_note nil, got %v", item.ResolutionNote)
	}
}

// --- Test Anchor 2: Create with nil alertID ---

func TestCreateActionItem_NilAlertID(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	item, err := svc.CreateActionItem(ctx, "c1", nil, "Manual task", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.AlertID != nil {
		t.Errorf("expected alert_id nil, got %v", item.AlertID)
	}
}

// --- Test Anchor 3: Create with nil dueDate ---

func TestCreateActionItem_NilDueDate(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	alertID := "alert-1"
	item, err := svc.CreateActionItem(ctx, "c1", &alertID, "No deadline", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.DueDate != nil {
		t.Errorf("expected due_date nil, got %v", item.DueDate)
	}
}

// --- Test Anchor 4: GetActionItem returns correct item ---

func TestGetActionItem_ReturnsCorrectItem(t *testing.T) {
	svc, _ := newTestService()
	now := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC)
	svc.clock = fixedClock(now)
	ctx := context.Background()

	alertID := "alert-1"
	dueDate := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	created, _ := svc.CreateActionItem(ctx, "c1", &alertID, "Test item", &dueDate)

	item, err := svc.GetActionItem(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, item.ID)
	}
	if item.ClientID != "c1" {
		t.Errorf("expected client_id c1, got %s", item.ClientID)
	}
	if item.AlertID == nil || *item.AlertID != "alert-1" {
		t.Errorf("expected alert_id alert-1, got %v", item.AlertID)
	}
	if item.Text != "Test item" {
		t.Errorf("expected text 'Test item', got %s", item.Text)
	}
	if item.Status != ActionItemStatusPending {
		t.Errorf("expected status PENDING, got %s", item.Status)
	}
}

// --- Test Anchor 5: GetActionItem with invalid ID ---

func TestGetActionItem_InvalidID_ReturnsError(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.GetActionItem(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for invalid ID, got nil")
	}
}

// --- Test Anchor 6: GetActionItemsByClientID returns all items ---

func TestGetActionItemsByClientID_ReturnsAll(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	alertID1 := "alert-1"
	alertID2 := "alert-2"
	svc.CreateActionItem(ctx, "c1", &alertID1, "Alert item 1", nil)
	svc.CreateActionItem(ctx, "c1", &alertID2, "Alert item 2", nil)
	svc.CreateActionItem(ctx, "c1", nil, "Manual item", nil)
	svc.CreateActionItem(ctx, "c2", nil, "Other client", nil)

	items, err := svc.GetActionItemsByClientID(ctx, "c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items for c1, got %d", len(items))
	}
}

// --- Test Anchor 7: GetActionItemsByClientID with no items ---

func TestGetActionItemsByClientID_Empty_ReturnsEmptySlice(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	items, err := svc.GetActionItemsByClientID(ctx, "c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if items == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

// --- Test Anchor 8: GetActionItemsByAlertID returns linked items ---

func TestGetActionItemsByAlertID_ReturnsLinked(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	alertID := "alert-1"
	svc.CreateActionItem(ctx, "c1", &alertID, "Item 1", nil)
	svc.CreateActionItem(ctx, "c1", &alertID, "Item 2", nil)
	otherAlert := "alert-2"
	svc.CreateActionItem(ctx, "c1", &otherAlert, "Other alert item", nil)

	items, err := svc.GetActionItemsByAlertID(ctx, "alert-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items for alert-1, got %d", len(items))
	}
}

// --- Test Anchor 9: GetActionItemsByAlertID with no items ---

func TestGetActionItemsByAlertID_Empty_ReturnsEmptySlice(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	items, err := svc.GetActionItemsByAlertID(ctx, "alert-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if items == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

// --- Test Anchor 10: PENDING -> IN_PROGRESS ---

func TestUpdateActionItem_PendingToInProgress(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.CreateActionItem(ctx, "c1", nil, "Task", nil)

	status := ActionItemStatusInProgress
	updated, err := svc.UpdateActionItem(ctx, created.ID, nil, &status, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != ActionItemStatusInProgress {
		t.Errorf("expected IN_PROGRESS, got %s", updated.Status)
	}
	if updated.ResolvedAt != nil {
		t.Errorf("expected resolved_at nil, got %v", updated.ResolvedAt)
	}
}

// --- Test Anchor 11: IN_PROGRESS -> DONE ---

func TestUpdateActionItem_InProgressToDone(t *testing.T) {
	svc, _ := newTestService()
	now := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC)
	svc.clock = fixedClock(now)
	ctx := context.Background()

	created, _ := svc.CreateActionItem(ctx, "c1", nil, "Task", nil)
	inProgress := ActionItemStatusInProgress
	svc.UpdateActionItem(ctx, created.ID, nil, &inProgress, nil)

	done := ActionItemStatusDone
	updated, err := svc.UpdateActionItem(ctx, created.ID, nil, &done, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != ActionItemStatusDone {
		t.Errorf("expected DONE, got %s", updated.Status)
	}
	if updated.ResolvedAt == nil {
		t.Fatal("expected resolved_at to be set")
	}
	if !updated.ResolvedAt.Equal(now) {
		t.Errorf("expected resolved_at %v, got %v", now, *updated.ResolvedAt)
	}
}

// --- Test Anchor 12: PENDING -> DONE (skip IN_PROGRESS) ---

func TestUpdateActionItem_PendingToDone(t *testing.T) {
	svc, _ := newTestService()
	now := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC)
	svc.clock = fixedClock(now)
	ctx := context.Background()

	created, _ := svc.CreateActionItem(ctx, "c1", nil, "Task", nil)

	done := ActionItemStatusDone
	updated, err := svc.UpdateActionItem(ctx, created.ID, nil, &done, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != ActionItemStatusDone {
		t.Errorf("expected DONE, got %s", updated.Status)
	}
	if updated.ResolvedAt == nil {
		t.Fatal("expected resolved_at to be set")
	}
}

// --- Test Anchor 13: Update text only ---

func TestUpdateActionItem_TextOnly(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.CreateActionItem(ctx, "c1", nil, "Original text", nil)

	newText := "Updated text"
	updated, err := svc.UpdateActionItem(ctx, created.ID, &newText, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Text != "Updated text" {
		t.Errorf("expected 'Updated text', got %s", updated.Text)
	}
	if updated.Status != ActionItemStatusPending {
		t.Errorf("expected status unchanged (PENDING), got %s", updated.Status)
	}
}

// --- Test Anchor 14: Update dueDate only ---

func TestUpdateActionItem_DueDateOnly(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.CreateActionItem(ctx, "c1", nil, "Task", nil)

	newDueDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	updated, err := svc.UpdateActionItem(ctx, created.ID, nil, nil, &newDueDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.DueDate == nil || !updated.DueDate.Equal(newDueDate) {
		t.Errorf("expected due_date %v, got %v", newDueDate, updated.DueDate)
	}
	if updated.Status != ActionItemStatusPending {
		t.Errorf("expected status unchanged (PENDING), got %s", updated.Status)
	}
}

// --- Test Anchor 15: IN_PROGRESS -> PENDING (invalid) ---

func TestUpdateActionItem_InProgressToPending_Error(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.CreateActionItem(ctx, "c1", nil, "Task", nil)
	inProgress := ActionItemStatusInProgress
	svc.UpdateActionItem(ctx, created.ID, nil, &inProgress, nil)

	pending := ActionItemStatusPending
	_, err := svc.UpdateActionItem(ctx, created.ID, nil, &pending, nil)
	if err == nil {
		t.Fatal("expected error for IN_PROGRESS -> PENDING, got nil")
	}

	// Verify no fields modified.
	item, _ := svc.GetActionItem(ctx, created.ID)
	if item.Status != ActionItemStatusInProgress {
		t.Errorf("expected status still IN_PROGRESS, got %s", item.Status)
	}
}

// --- Test Anchor 16: DONE -> IN_PROGRESS (invalid) ---

func TestUpdateActionItem_DoneToInProgress_Error(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.CreateActionItem(ctx, "c1", nil, "Task", nil)
	done := ActionItemStatusDone
	svc.UpdateActionItem(ctx, created.ID, nil, &done, nil)

	inProgress := ActionItemStatusInProgress
	_, err := svc.UpdateActionItem(ctx, created.ID, nil, &inProgress, nil)
	if err == nil {
		t.Fatal("expected error for DONE -> IN_PROGRESS, got nil")
	}
}

// --- Test Anchor 17: DONE -> PENDING (invalid) ---

func TestUpdateActionItem_DoneToPending_Error(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.CreateActionItem(ctx, "c1", nil, "Task", nil)
	done := ActionItemStatusDone
	svc.UpdateActionItem(ctx, created.ID, nil, &done, nil)

	pending := ActionItemStatusPending
	_, err := svc.UpdateActionItem(ctx, created.ID, nil, &pending, nil)
	if err == nil {
		t.Fatal("expected error for DONE -> PENDING, got nil")
	}
}

// --- Test Anchor 18: CLOSED item, any update -> error ---

func TestUpdateActionItem_Closed_AnyUpdate_Error(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.CreateActionItem(ctx, "c1", nil, "Task", nil)
	svc.CloseActionItem(ctx, created.ID, "Auto-closed")

	newText := "Updated"
	_, err := svc.UpdateActionItem(ctx, created.ID, &newText, nil, nil)
	if err == nil {
		t.Fatal("expected error for updating CLOSED item, got nil")
	}
}

// --- Test Anchor 19: PENDING -> CLOSED via UpdateActionItem (invalid) ---

func TestUpdateActionItem_PendingToClosed_Error(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, _ := svc.CreateActionItem(ctx, "c1", nil, "Task", nil)

	closed := ActionItemStatusClosed
	_, err := svc.UpdateActionItem(ctx, created.ID, nil, &closed, nil)
	if err == nil {
		t.Fatal("expected error for PENDING -> CLOSED via UpdateActionItem, got nil")
	}
}

// --- Test Anchor 20: CloseActionItem on PENDING ---

func TestCloseActionItem_Pending(t *testing.T) {
	svc, _ := newTestService()
	now := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC)
	svc.clock = fixedClock(now)
	ctx := context.Background()

	created, _ := svc.CreateActionItem(ctx, "c1", nil, "Task", nil)

	note := "Auto-closed: over_contribution condition resolved on 2026-03-02"
	item, err := svc.CloseActionItem(ctx, created.ID, note)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Status != ActionItemStatusClosed {
		t.Errorf("expected CLOSED, got %s", item.Status)
	}
	if item.ResolvedAt == nil {
		t.Fatal("expected resolved_at to be set")
	}
	if !item.ResolvedAt.Equal(now) {
		t.Errorf("expected resolved_at %v, got %v", now, *item.ResolvedAt)
	}
	if item.ResolutionNote == nil || *item.ResolutionNote != note {
		t.Errorf("expected resolution_note %q, got %v", note, item.ResolutionNote)
	}
}

// --- Test Anchor 21: CloseActionItem on IN_PROGRESS ---

func TestCloseActionItem_InProgress(t *testing.T) {
	svc, _ := newTestService()
	now := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC)
	svc.clock = fixedClock(now)
	ctx := context.Background()

	created, _ := svc.CreateActionItem(ctx, "c1", nil, "Task", nil)
	inProgress := ActionItemStatusInProgress
	svc.UpdateActionItem(ctx, created.ID, nil, &inProgress, nil)

	item, err := svc.CloseActionItem(ctx, created.ID, "Cascade close")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Status != ActionItemStatusClosed {
		t.Errorf("expected CLOSED, got %s", item.Status)
	}
	if item.ResolvedAt == nil {
		t.Fatal("expected resolved_at to be set")
	}
}

// --- Test Anchor 22: CloseActionItem on DONE overwrites resolved_at ---

func TestCloseActionItem_Done_OverwritesResolvedAt(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	doneTime := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	svc.clock = fixedClock(doneTime)
	created, _ := svc.CreateActionItem(ctx, "c1", nil, "Task", nil)
	done := ActionItemStatusDone
	svc.UpdateActionItem(ctx, created.ID, nil, &done, nil)

	// Verify resolved_at was set to doneTime.
	doneItem, _ := svc.GetActionItem(ctx, created.ID)
	if doneItem.ResolvedAt == nil || !doneItem.ResolvedAt.Equal(doneTime) {
		t.Fatalf("expected resolved_at %v after DONE, got %v", doneTime, doneItem.ResolvedAt)
	}

	closeTime := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC)
	svc.clock = fixedClock(closeTime)

	item, err := svc.CloseActionItem(ctx, created.ID, "Cascade close")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Status != ActionItemStatusClosed {
		t.Errorf("expected CLOSED, got %s", item.Status)
	}
	if item.ResolvedAt == nil || !item.ResolvedAt.Equal(closeTime) {
		t.Errorf("expected resolved_at overwritten to %v, got %v", closeTime, item.ResolvedAt)
	}
}

// --- Test Anchor 23: CloseActionItem on already CLOSED (idempotent) ---

func TestCloseActionItem_AlreadyClosed_Idempotent(t *testing.T) {
	svc, _ := newTestService()
	closeTime := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC)
	svc.clock = fixedClock(closeTime)
	ctx := context.Background()

	created, _ := svc.CreateActionItem(ctx, "c1", nil, "Task", nil)
	note := "First close"
	svc.CloseActionItem(ctx, created.ID, note)

	// Second close — should be a no-op.
	laterTime := time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)
	svc.clock = fixedClock(laterTime)
	item, err := svc.CloseActionItem(ctx, created.ID, "Second close attempt")
	if err != nil {
		t.Fatalf("expected no error on idempotent close, got %v", err)
	}
	if item.Status != ActionItemStatusClosed {
		t.Errorf("expected CLOSED, got %s", item.Status)
	}
	// resolved_at and resolution_note should be unchanged from first close.
	if item.ResolvedAt == nil || !item.ResolvedAt.Equal(closeTime) {
		t.Errorf("expected resolved_at unchanged at %v, got %v", closeTime, item.ResolvedAt)
	}
	if item.ResolutionNote == nil || *item.ResolutionNote != note {
		t.Errorf("expected resolution_note unchanged %q, got %v", note, item.ResolutionNote)
	}
}
