package graph

import (
	"context"
	"testing"
	"time"

	"github.com/swatkatz/advisorhub/backend/internal/actionitem"
	"github.com/swatkatz/advisorhub/backend/internal/alert"
	"github.com/swatkatz/advisorhub/backend/internal/client"
	"github.com/swatkatz/advisorhub/backend/internal/contribution"
	"github.com/swatkatz/advisorhub/backend/internal/graph/model"
	"github.com/swatkatz/advisorhub/backend/internal/transfer"
)

// Test Anchor 1: clients sorted by health rank (RED first, YELLOW, GREEN)
func TestClients_SortedByHealthRank(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	f.clientRepo.addClient(client.Client{ID: "c1", AdvisorID: "a1", Name: "Alice"})
	f.clientRepo.addClient(client.Client{ID: "c2", AdvisorID: "a1", Name: "Bob"})
	f.clientRepo.addClient(client.Client{ID: "c3", AdvisorID: "a1", Name: "Carol"})

	f.alertService.healthByClient["c1"] = alert.HealthGreen
	f.alertService.healthByClient["c2"] = alert.HealthYellow
	f.alertService.healthByClient["c3"] = alert.HealthRed

	qr := &queryResolver{f.resolver}
	clients, err := qr.Clients(ctx, "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clients) != 3 {
		t.Fatalf("expected 3 clients, got %d", len(clients))
	}
	if clients[0].ID != "c3" {
		t.Errorf("expected RED client c3 first, got %s", clients[0].ID)
	}
	if clients[1].ID != "c2" {
		t.Errorf("expected YELLOW client c2 second, got %s", clients[1].ID)
	}
	if clients[2].ID != "c1" {
		t.Errorf("expected GREEN client c1 third, got %s", clients[2].ID)
	}
}

// Test Anchor 2: clients with same health sorted alphabetically
func TestClients_SameHealthSortedByName(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	f.clientRepo.addClient(client.Client{ID: "c1", AdvisorID: "a1", Name: "Zara"})
	f.clientRepo.addClient(client.Client{ID: "c2", AdvisorID: "a1", Name: "Alice"})

	f.alertService.healthByClient["c1"] = alert.HealthGreen
	f.alertService.healthByClient["c2"] = alert.HealthGreen

	qr := &queryResolver{f.resolver}
	clients, err := qr.Clients(ctx, "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clients) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(clients))
	}
	if clients[0].Name != "Alice" {
		t.Errorf("expected Alice first, got %s", clients[0].Name)
	}
	if clients[1].Name != "Zara" {
		t.Errorf("expected Zara second, got %s", clients[1].Name)
	}
}

// Test Anchor 3: client(id) returns client with scalar fields
func TestClient_Found(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()
	dob := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	meeting := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	f.clientRepo.addClient(client.Client{
		ID: "c1", AdvisorID: "a1", Name: "Alice",
		Email: "alice@test.com", DateOfBirth: dob, LastMeetingDate: meeting,
	})

	qr := &queryResolver{f.resolver}
	c, err := qr.Client(ctx, "c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ID != "c1" || c.Name != "Alice" || c.Email != "alice@test.com" {
		t.Errorf("unexpected client fields: %+v", c)
	}
}

// Test Anchor 4: client(id) with invalid ID returns error
func TestClient_NotFound(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	qr := &queryResolver{f.resolver}
	_, err := qr.Client(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent client")
	}
}

// Test Anchor 5: alerts sorted by severity rank
func TestAlerts_SortedBySeverity(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()
	now := time.Now()

	f.alertRepo.addAlertForAdvisor("a1", alert.Alert{ID: "al1", ClientID: "c1", Severity: alert.SeverityInfo, Status: alert.StatusOpen, CreatedAt: now})
	f.alertRepo.addAlertForAdvisor("a1", alert.Alert{ID: "al2", ClientID: "c1", Severity: alert.SeverityCritical, Status: alert.StatusOpen, CreatedAt: now})
	f.alertRepo.addAlertForAdvisor("a1", alert.Alert{ID: "al3", ClientID: "c1", Severity: alert.SeverityAdvisory, Status: alert.StatusOpen, CreatedAt: now})
	f.alertRepo.addAlertForAdvisor("a1", alert.Alert{ID: "al4", ClientID: "c1", Severity: alert.SeverityUrgent, Status: alert.StatusOpen, CreatedAt: now})

	qr := &queryResolver{f.resolver}
	alerts, err := qr.Alerts(ctx, "a1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alerts) != 4 {
		t.Fatalf("expected 4 alerts, got %d", len(alerts))
	}
	expected := []string{"CRITICAL", "URGENT", "ADVISORY", "INFO"}
	for i, a := range alerts {
		if string(a.Severity) != expected[i] {
			t.Errorf("position %d: expected %s, got %s", i, expected[i], a.Severity)
		}
	}
}

// Test Anchor 6: within same severity, more recent first
func TestAlerts_SameSeveritySortedByRecency(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	older := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	f.alertRepo.addAlertForAdvisor("a1", alert.Alert{ID: "old", ClientID: "c1", Severity: alert.SeverityCritical, Status: alert.StatusOpen, CreatedAt: older})
	f.alertRepo.addAlertForAdvisor("a1", alert.Alert{ID: "new", ClientID: "c1", Severity: alert.SeverityCritical, Status: alert.StatusOpen, CreatedAt: newer})

	qr := &queryResolver{f.resolver}
	alerts, err := qr.Alerts(ctx, "a1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alerts[0].ID != "new" {
		t.Errorf("expected newer alert first, got %s", alerts[0].ID)
	}
	if alerts[1].ID != "old" {
		t.Errorf("expected older alert second, got %s", alerts[1].ID)
	}
}

// Test Anchor 7: filter by severity
func TestAlerts_FilterBySeverity(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()
	now := time.Now()

	f.alertRepo.addAlertForAdvisor("a1", alert.Alert{ID: "al1", ClientID: "c1", Severity: alert.SeverityCritical, Status: alert.StatusOpen, CreatedAt: now})
	f.alertRepo.addAlertForAdvisor("a1", alert.Alert{ID: "al2", ClientID: "c1", Severity: alert.SeverityCritical, Status: alert.StatusOpen, CreatedAt: now})
	f.alertRepo.addAlertForAdvisor("a1", alert.Alert{ID: "al3", ClientID: "c1", Severity: alert.SeverityUrgent, Status: alert.StatusOpen, CreatedAt: now})
	f.alertRepo.addAlertForAdvisor("a1", alert.Alert{ID: "al4", ClientID: "c1", Severity: alert.SeverityAdvisory, Status: alert.StatusOpen, CreatedAt: now})

	sev := model.AlertSeverityCritical
	qr := &queryResolver{f.resolver}
	alerts, err := qr.Alerts(ctx, "a1", &model.AlertFilter{Severity: &sev})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alerts) != 2 {
		t.Fatalf("expected 2 CRITICAL alerts, got %d", len(alerts))
	}
}

// Test Anchor 8: filter by clientId
func TestAlerts_FilterByClientID(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()
	now := time.Now()

	f.alertRepo.addAlertForAdvisor("a1", alert.Alert{ID: "al1", ClientID: "c1", Severity: alert.SeverityCritical, Status: alert.StatusOpen, CreatedAt: now})
	f.alertRepo.addAlertForAdvisor("a1", alert.Alert{ID: "al2", ClientID: "c1", Severity: alert.SeverityUrgent, Status: alert.StatusOpen, CreatedAt: now})
	f.alertRepo.addAlertForAdvisor("a1", alert.Alert{ID: "al3", ClientID: "c2", Severity: alert.SeverityCritical, Status: alert.StatusOpen, CreatedAt: now})
	f.alertRepo.addAlertForAdvisor("a1", alert.Alert{ID: "al4", ClientID: "c2", Severity: alert.SeverityAdvisory, Status: alert.StatusOpen, CreatedAt: now})

	clientID := "c1"
	qr := &queryResolver{f.resolver}
	alerts, err := qr.Alerts(ctx, "a1", &model.AlertFilter{ClientID: &clientID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alerts) != 2 {
		t.Fatalf("expected 2 alerts for c1, got %d", len(alerts))
	}
}

// Test Anchor 9: combined filter (severity + status)
func TestAlerts_CombinedFilter(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()
	now := time.Now()

	f.alertRepo.addAlertForAdvisor("a1", alert.Alert{ID: "al1", ClientID: "c1", Severity: alert.SeverityUrgent, Status: alert.StatusOpen, CreatedAt: now})
	f.alertRepo.addAlertForAdvisor("a1", alert.Alert{ID: "al2", ClientID: "c1", Severity: alert.SeverityUrgent, Status: alert.StatusSnoozed, CreatedAt: now})
	f.alertRepo.addAlertForAdvisor("a1", alert.Alert{ID: "al3", ClientID: "c1", Severity: alert.SeverityCritical, Status: alert.StatusOpen, CreatedAt: now})

	sev := model.AlertSeverityUrgent
	status := model.AlertStatusOpen
	qr := &queryResolver{f.resolver}
	alerts, err := qr.Alerts(ctx, "a1", &model.AlertFilter{Severity: &sev, Status: &status})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert matching URGENT+OPEN, got %d", len(alerts))
	}
	if alerts[0].ID != "al1" {
		t.Errorf("expected al1, got %s", alerts[0].ID)
	}
}

// Test Anchor 10: alert(id) returns full alert
func TestAlert_Found(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	f.alertRepo.addAlert(alert.Alert{
		ID: "al1", ClientID: "c1", Severity: alert.SeverityCritical,
		Category: "over_contribution", Status: alert.StatusOpen,
		Summary: "Test summary", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	qr := &queryResolver{f.resolver}
	a, err := qr.Alert(ctx, "al1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.ID != "al1" || a.Category != "over_contribution" {
		t.Errorf("unexpected alert: %+v", a)
	}
}

// Test Anchor 11: alert(id) with invalid ID returns error
func TestAlert_NotFound(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	qr := &queryResolver{f.resolver}
	_, err := qr.Alert(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent alert")
	}
}

// Test Anchor 12+13: contributionSummary with lifetimeCap enrichment
func TestContributionSummary_LifetimeCapEnrichment(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	f.contribEngine.summaries["c1:2026"] = &contribution.ContributionSummary{
		ClientID: "c1",
		TaxYear:  2026,
		Accounts: []contribution.AccountContribution{
			{AccountType: "RRSP", AnnualLimit: 32490, Contributed: 20000, Remaining: 12490},
			{AccountType: "FHSA", AnnualLimit: 8000, Contributed: 5000, Remaining: 3000},
		},
	}

	qr := &queryResolver{f.resolver}
	summary, err := qr.ContributionSummary(ctx, "c1", 2026)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(summary.Accounts) != 2 {
		t.Fatalf("expected 2 account contributions, got %d", len(summary.Accounts))
	}

	for _, ac := range summary.Accounts {
		if string(ac.AccountType) == "RRSP" {
			if ac.LifetimeCap != nil {
				t.Errorf("RRSP should have nil lifetimeCap, got %v", *ac.LifetimeCap)
			}
		}
		if string(ac.AccountType) == "FHSA" {
			if ac.LifetimeCap == nil || *ac.LifetimeCap != 40000 {
				t.Errorf("FHSA should have lifetimeCap 40000, got %v", ac.LifetimeCap)
			}
		}
	}
}

// Test Anchor 14: transfers aggregated across clients (including INVESTED)
func TestTransfers_AggregatedAcrossClients(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()
	now := time.Now()

	f.clientRepo.addClient(client.Client{ID: "c1", AdvisorID: "a1", Name: "Alice"})
	f.clientRepo.addClient(client.Client{ID: "c2", AdvisorID: "a1", Name: "Bob"})
	f.clientRepo.addClient(client.Client{ID: "c3", AdvisorID: "a1", Name: "Carol"})

	f.transferRepo.addTransfer(transfer.Transfer{ID: "t1", ClientID: "c1", Status: transfer.StatusInTransit, LastStatusChange: now, InitiatedAt: now})
	f.transferRepo.addTransfer(transfer.Transfer{ID: "t2", ClientID: "c1", Status: transfer.StatusInitiated, LastStatusChange: now, InitiatedAt: now})
	f.transferRepo.addTransfer(transfer.Transfer{ID: "t3", ClientID: "c2", Status: transfer.StatusInvested, LastStatusChange: now, InitiatedAt: now})

	qr := &queryResolver{f.resolver}
	transfers, err := qr.Transfers(ctx, "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(transfers) != 3 {
		t.Fatalf("expected 3 transfers (including INVESTED), got %d", len(transfers))
	}
}

// Test Anchor 15: action items sorted by status rank
func TestActionItems_SortedByStatus(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	f.actionItemSvc.addItem(actionitem.ActionItem{ID: "ai1", ClientID: "c1", Status: actionitem.ActionItemStatusDone, Text: "Done item", CreatedAt: time.Now()})
	f.actionItemSvc.addItem(actionitem.ActionItem{ID: "ai2", ClientID: "c1", Status: actionitem.ActionItemStatusPending, Text: "Pending item", CreatedAt: time.Now()})
	f.actionItemSvc.addItem(actionitem.ActionItem{ID: "ai3", ClientID: "c1", Status: actionitem.ActionItemStatusInProgress, Text: "In progress", CreatedAt: time.Now()})

	clientID := "c1"
	qr := &queryResolver{f.resolver}
	items, err := qr.ActionItems(ctx, &clientID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	expected := []string{"PENDING", "IN_PROGRESS", "DONE"}
	for i, item := range items {
		if string(item.Status) != expected[i] {
			t.Errorf("position %d: expected %s, got %s", i, expected[i], item.Status)
		}
	}
}

// Test Anchor 16: PENDING action items with due date before nulls
func TestActionItems_DueDateNullsLast(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	dueDate := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	f.actionItemSvc.addItem(actionitem.ActionItem{ID: "ai1", ClientID: "c1", Status: actionitem.ActionItemStatusPending, Text: "No date", CreatedAt: time.Now()})
	f.actionItemSvc.addItem(actionitem.ActionItem{ID: "ai2", ClientID: "c1", Status: actionitem.ActionItemStatusPending, Text: "Has date", DueDate: &dueDate, CreatedAt: time.Now()})

	clientID := "c1"
	qr := &queryResolver{f.resolver}
	items, err := qr.ActionItems(ctx, &clientID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].ID != "ai2" {
		t.Errorf("expected item with due date first, got %s", items[0].ID)
	}
	if items[1].ID != "ai1" {
		t.Errorf("expected item without due date second, got %s", items[1].ID)
	}
}
