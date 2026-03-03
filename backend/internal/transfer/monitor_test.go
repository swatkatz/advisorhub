package transfer

import (
	"context"
	"testing"
	"time"
)

var testNow = time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC)

// --- Repository tests (anchors 1-5, 16) ---

func TestGetTransfer_ValidID(t *testing.T) {
	_, repo, _ := setupMonitor(testNow)
	repo.addTransfer(Transfer{
		ID:                "t1",
		ClientID:          "c8",
		SourceInstitution: "TD",
		AccountType:       AccountTypeRRSP,
		Amount:            67400,
		Status:            StatusDocumentsSubmitted,
		InitiatedAt:       testNow.AddDate(0, 0, -20),
		LastStatusChange:  testNow.AddDate(0, 0, -18),
	})

	result, err := repo.GetTransfer(context.Background(), "t1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "t1" {
		t.Errorf("expected ID t1, got %s", result.ID)
	}
	if result.ClientID != "c8" {
		t.Errorf("expected ClientID c8, got %s", result.ClientID)
	}
	if result.SourceInstitution != "TD" {
		t.Errorf("expected SourceInstitution TD, got %s", result.SourceInstitution)
	}
	if result.AccountType != AccountTypeRRSP {
		t.Errorf("expected AccountType RRSP, got %s", result.AccountType)
	}
	if result.Amount != 67400 {
		t.Errorf("expected Amount 67400, got %f", result.Amount)
	}
	if result.Status != StatusDocumentsSubmitted {
		t.Errorf("expected Status DOCUMENTS_SUBMITTED, got %s", result.Status)
	}
}

func TestGetTransfer_InvalidID(t *testing.T) {
	_, repo, _ := setupMonitor(testNow)

	_, err := repo.GetTransfer(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for invalid ID, got nil")
	}
}

func TestGetTransfersByClientID_MultipleTransfers(t *testing.T) {
	_, repo, _ := setupMonitor(testNow)
	repo.addTransfer(Transfer{ID: "t1", ClientID: "c1", Status: StatusInitiated})
	repo.addTransfer(Transfer{ID: "t2", ClientID: "c1", Status: StatusInReview})
	repo.addTransfer(Transfer{ID: "t3", ClientID: "c2", Status: StatusInitiated})

	result, err := repo.GetTransfersByClientID(context.Background(), "c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 transfers, got %d", len(result))
	}
}

func TestGetTransfersByClientID_NoTransfers(t *testing.T) {
	_, repo, _ := setupMonitor(testNow)

	result, err := repo.GetTransfersByClientID(context.Background(), "c99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty slice, got %d transfers", len(result))
	}
}

func TestGetActiveTransfers_ExcludesInvested(t *testing.T) {
	_, repo, _ := setupMonitor(testNow)
	repo.addTransfer(Transfer{ID: "t1", Status: StatusInitiated})
	repo.addTransfer(Transfer{ID: "t2", Status: StatusDocumentsSubmitted})
	repo.addTransfer(Transfer{ID: "t3", Status: StatusInReview})
	repo.addTransfer(Transfer{ID: "t4", Status: StatusInTransit})
	repo.addTransfer(Transfer{ID: "t5", Status: StatusInvested})

	result, err := repo.GetActiveTransfers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 4 {
		t.Fatalf("expected 4 active transfers, got %d", len(result))
	}
	for _, tr := range result {
		if tr.Status == StatusInvested {
			t.Errorf("INVESTED transfer should not be in active results: %s", tr.ID)
		}
	}
}

func TestCreateTransfer_SetsLastStatusChange(t *testing.T) {
	_, repo, _ := setupMonitor(testNow)
	initiatedAt := time.Date(2026, 2, 12, 9, 0, 0, 0, time.UTC)
	transfer := &Transfer{
		ID:                "t1",
		ClientID:          "c1",
		SourceInstitution: "RBC",
		AccountType:       AccountTypeRRSP,
		Amount:            42000,
		Status:            StatusInitiated,
		InitiatedAt:       initiatedAt,
	}

	result, err := repo.CreateTransfer(context.Background(), transfer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.LastStatusChange.Equal(initiatedAt) {
		t.Errorf("expected LastStatusChange=%v, got %v", initiatedAt, result.LastStatusChange)
	}

	// Verify it's persisted
	fetched, err := repo.GetTransfer(context.Background(), "t1")
	if err != nil {
		t.Fatalf("unexpected error fetching created transfer: %v", err)
	}
	if !fetched.LastStatusChange.Equal(initiatedAt) {
		t.Errorf("persisted LastStatusChange=%v, expected %v", fetched.LastStatusChange, initiatedAt)
	}
}

// --- Computed fields tests (anchors 13-15) ---

func TestDaysInCurrentStage(t *testing.T) {
	tr := Transfer{
		LastStatusChange: testNow.AddDate(0, 0, -10),
	}
	days := tr.DaysInCurrentStage(testNow)
	if days != 10 {
		t.Errorf("expected 10 days, got %d", days)
	}
}

func TestIsStuck_True(t *testing.T) {
	tr := Transfer{
		Status:           StatusInReview,
		LastStatusChange: testNow.AddDate(0, 0, -15), // 15 days, threshold 14
	}
	if !tr.IsStuck(testNow) {
		t.Error("expected IsStuck=true for IN_REVIEW with 15 days (threshold 14)")
	}
}

func TestIsStuck_False(t *testing.T) {
	tr := Transfer{
		Status:           StatusInReview,
		LastStatusChange: testNow.AddDate(0, 0, -8), // 8 days, threshold 14
	}
	if tr.IsStuck(testNow) {
		t.Error("expected IsStuck=false for IN_REVIEW with 8 days (threshold 14)")
	}
}

// --- Stuck detection tests (anchors 6-12) ---

func TestCheckStuck_DocumentsSubmitted18Days(t *testing.T) {
	m, repo, bus := setupMonitor(testNow)
	repo.addTransfer(Transfer{
		ID:                "t1",
		ClientID:          "c8",
		SourceInstitution: "TD",
		AccountType:       AccountTypeRRSP,
		Amount:            67400,
		Status:            StatusDocumentsSubmitted,
		LastStatusChange:  testNow.AddDate(0, 0, -18),
	})

	results, err := m.CheckStuckTransfers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Signal != SignalStuckDetected {
		t.Errorf("expected STUCK_DETECTED, got %s", results[0].Signal)
	}

	events := bus.eventsByType(EventTransferStuck)
	if len(events) != 1 {
		t.Fatalf("expected 1 TransferStuck event, got %d", len(events))
	}
	payload := parsePayload(t, events[0].Payload)
	if payload["days_in_stage"] != float64(18) {
		t.Errorf("expected days_in_stage=18, got %v", payload["days_in_stage"])
	}
	if payload["stuck_threshold"] != float64(10) {
		t.Errorf("expected stuck_threshold=10, got %v", payload["stuck_threshold"])
	}
}

func TestCheckStuck_InTransit3Days_NoChange(t *testing.T) {
	m, repo, bus := setupMonitor(testNow)
	repo.addTransfer(Transfer{
		ID:               "t1",
		ClientID:         "c1",
		Status:           StatusInTransit,
		LastStatusChange: testNow.AddDate(0, 0, -3), // 3 days, threshold 14
	})

	results, err := m.CheckStuckTransfers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Signal != SignalNoChange {
		t.Errorf("expected NO_CHANGE, got %s", results[0].Signal)
	}

	events := bus.eventsByType(EventTransferStuck)
	if len(events) != 0 {
		t.Errorf("expected no TransferStuck events, got %d", len(events))
	}
}

func TestCheckStuck_InitiatedExactly5Days_NoChange(t *testing.T) {
	m, repo, bus := setupMonitor(testNow)
	repo.addTransfer(Transfer{
		ID:               "t1",
		ClientID:         "c1",
		Status:           StatusInitiated,
		LastStatusChange: testNow.AddDate(0, 0, -5), // exactly 5 days, threshold 5
	})

	results, err := m.CheckStuckTransfers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].Signal != SignalNoChange {
		t.Errorf("expected NO_CHANGE at exactly threshold, got %s", results[0].Signal)
	}

	events := bus.eventsByType(EventTransferStuck)
	if len(events) != 0 {
		t.Errorf("expected no TransferStuck events at exactly threshold, got %d", len(events))
	}
}

func TestCheckStuck_Initiated6Days_Stuck(t *testing.T) {
	m, repo, bus := setupMonitor(testNow)
	repo.addTransfer(Transfer{
		ID:               "t1",
		ClientID:         "c1",
		Status:           StatusInitiated,
		LastStatusChange: testNow.AddDate(0, 0, -6), // 6 days, threshold 5
	})

	results, err := m.CheckStuckTransfers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].Signal != SignalStuckDetected {
		t.Errorf("expected STUCK_DETECTED, got %s", results[0].Signal)
	}

	events := bus.eventsByType(EventTransferStuck)
	if len(events) != 1 {
		t.Fatalf("expected 1 TransferStuck event, got %d", len(events))
	}
}

func TestCheckStuck_InvestedTransfer_Skipped(t *testing.T) {
	m, repo, bus := setupMonitor(testNow)
	repo.addTransfer(Transfer{
		ID:               "t1",
		ClientID:         "c1",
		Status:           StatusInvested,
		LastStatusChange: testNow.AddDate(0, 0, -30),
	})

	results, err := m.CheckStuckTransfers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for INVESTED transfer, got %d", len(results))
	}

	events := bus.allEvents()
	if len(events) != 0 {
		t.Errorf("expected no events for INVESTED transfer, got %d", len(events))
	}
}

func TestCheckStuck_MixedTransfers(t *testing.T) {
	m, repo, bus := setupMonitor(testNow)
	// Stuck: INITIATED for 6 days (threshold 5)
	repo.addTransfer(Transfer{
		ID:                "t1",
		ClientID:          "c1",
		SourceInstitution: "TD",
		AccountType:       AccountTypeRRSP,
		Amount:            50000,
		Status:            StatusInitiated,
		LastStatusChange:  testNow.AddDate(0, 0, -6),
	})
	// Not stuck: IN_REVIEW for 3 days (threshold 14)
	repo.addTransfer(Transfer{
		ID:               "t2",
		ClientID:         "c2",
		Status:           StatusInReview,
		LastStatusChange: testNow.AddDate(0, 0, -3),
	})
	// Not stuck: RECEIVED for 2 days (threshold 5)
	repo.addTransfer(Transfer{
		ID:               "t3",
		ClientID:         "c3",
		Status:           StatusReceived,
		LastStatusChange: testNow.AddDate(0, 0, -2),
	})

	results, err := m.CheckStuckTransfers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	stuck := 0
	noChange := 0
	for _, r := range results {
		switch r.Signal {
		case SignalStuckDetected:
			stuck++
		case SignalNoChange:
			noChange++
		}
	}
	if stuck != 1 {
		t.Errorf("expected 1 STUCK_DETECTED, got %d", stuck)
	}
	if noChange != 2 {
		t.Errorf("expected 2 NO_CHANGE, got %d", noChange)
	}

	events := bus.eventsByType(EventTransferStuck)
	if len(events) != 1 {
		t.Errorf("expected 1 TransferStuck event, got %d", len(events))
	}
}

func TestCheckStuck_NoActiveTransfers(t *testing.T) {
	m, _, bus := setupMonitor(testNow)

	results, err := m.CheckStuckTransfers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty repo, got %d", len(results))
	}

	events := bus.allEvents()
	if len(events) != 0 {
		t.Errorf("expected no events for empty repo, got %d", len(events))
	}
}

// --- Status transition tests (anchors 17-18) ---

func TestUpdateTransferStatus_ValidTransition(t *testing.T) {
	_, repo, _ := setupMonitor(testNow)
	repo.addTransfer(Transfer{
		ID:               "t1",
		ClientID:         "c1",
		Status:           StatusInitiated,
		LastStatusChange: testNow.AddDate(0, 0, -3),
	})

	result, err := repo.UpdateTransferStatus(context.Background(), "t1", StatusDocumentsSubmitted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != StatusDocumentsSubmitted {
		t.Errorf("expected status DOCUMENTS_SUBMITTED, got %s", result.Status)
	}
	if !result.LastStatusChange.Equal(testNow) {
		t.Errorf("expected LastStatusChange=%v, got %v", testNow, result.LastStatusChange)
	}
}

func TestUpdateTransferStatus_InvalidTransition_SkipStage(t *testing.T) {
	_, repo, _ := setupMonitor(testNow)
	repo.addTransfer(Transfer{
		ID:               "t1",
		ClientID:         "c1",
		Status:           StatusInitiated,
		LastStatusChange: testNow.AddDate(0, 0, -3),
	})

	_, err := repo.UpdateTransferStatus(context.Background(), "t1", StatusInTransit)
	if err == nil {
		t.Fatal("expected error for skipping stages, got nil")
	}

	// Verify transfer is unchanged
	transfer, _ := repo.GetTransfer(context.Background(), "t1")
	if transfer.Status != StatusInitiated {
		t.Errorf("expected status unchanged at INITIATED, got %s", transfer.Status)
	}
}

// --- Stuck detection decision table (table-driven) ---

func TestStuckDetection_DecisionTable(t *testing.T) {
	tests := []struct {
		name           string
		status         TransferStatus
		daysInStage    int
		expectedSignal CheckSignal
		expectEvent    bool
	}{
		{"INITIATED_at_threshold", StatusInitiated, 5, SignalNoChange, false},
		{"INITIATED_over_threshold", StatusInitiated, 6, SignalStuckDetected, true},
		{"DOCS_SUBMITTED_under", StatusDocumentsSubmitted, 7, SignalNoChange, false},
		{"DOCS_SUBMITTED_over", StatusDocumentsSubmitted, 11, SignalStuckDetected, true},
		{"IN_REVIEW_under", StatusInReview, 10, SignalNoChange, false},
		{"IN_REVIEW_over", StatusInReview, 15, SignalStuckDetected, true},
		{"IN_TRANSIT_under", StatusInTransit, 12, SignalNoChange, false},
		{"IN_TRANSIT_over", StatusInTransit, 15, SignalStuckDetected, true},
		{"RECEIVED_at_threshold", StatusReceived, 5, SignalNoChange, false},
		{"RECEIVED_over", StatusReceived, 6, SignalStuckDetected, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, repo, bus := setupMonitor(testNow)
			repo.addTransfer(Transfer{
				ID:               "t1",
				ClientID:         "c1",
				Status:           tt.status,
				LastStatusChange: testNow.AddDate(0, 0, -tt.daysInStage),
			})

			results, err := m.CheckStuckTransfers(context.Background())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}
			if results[0].Signal != tt.expectedSignal {
				t.Errorf("expected signal %s, got %s", tt.expectedSignal, results[0].Signal)
			}

			events := bus.eventsByType(EventTransferStuck)
			if tt.expectEvent && len(events) != 1 {
				t.Errorf("expected 1 TransferStuck event, got %d", len(events))
			}
			if !tt.expectEvent && len(events) != 0 {
				t.Errorf("expected no TransferStuck events, got %d", len(events))
			}
		})
	}
}

// --- Status transition pipeline (table-driven) ---

func TestStatusTransition_FullPipeline(t *testing.T) {
	transitions := []struct {
		from TransferStatus
		to   TransferStatus
	}{
		{StatusInitiated, StatusDocumentsSubmitted},
		{StatusDocumentsSubmitted, StatusInReview},
		{StatusInReview, StatusInTransit},
		{StatusInTransit, StatusReceived},
		{StatusReceived, StatusInvested},
	}

	_, repo, _ := setupMonitor(testNow)
	repo.addTransfer(Transfer{
		ID:               "t1",
		ClientID:         "c1",
		Status:           StatusInitiated,
		LastStatusChange: testNow.AddDate(0, 0, -1),
	})

	for _, tt := range transitions {
		result, err := repo.UpdateTransferStatus(context.Background(), "t1", tt.to)
		if err != nil {
			t.Fatalf("transition %s → %s: unexpected error: %v", tt.from, tt.to, err)
		}
		if result.Status != tt.to {
			t.Errorf("transition %s → %s: expected status %s, got %s", tt.from, tt.to, tt.to, result.Status)
		}
	}
}

func TestStatusTransition_TerminalInvested(t *testing.T) {
	_, repo, _ := setupMonitor(testNow)
	repo.addTransfer(Transfer{
		ID:     "t1",
		Status: StatusInvested,
	})

	_, err := repo.UpdateTransferStatus(context.Background(), "t1", StatusInitiated)
	if err == nil {
		t.Fatal("expected error for transition from terminal INVESTED, got nil")
	}
}
