package graph

import (
	"context"
	"testing"
	"time"

	"github.com/swatkatz/advisorhub/backend/internal/account"
	"github.com/swatkatz/advisorhub/backend/internal/actionitem"
	"github.com/swatkatz/advisorhub/backend/internal/alert"
	"github.com/swatkatz/advisorhub/backend/internal/client"
	"github.com/swatkatz/advisorhub/backend/internal/graph/model"
)

// Test Anchor 36: Client.aum sums all accounts (internal + external)
func TestClientAum(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	f.accountRepo.addAccount(account.Account{ID: "a1", ClientID: "c1", Balance: 100000, IsExternal: false})
	f.accountRepo.addAccount(account.Account{ID: "a2", ClientID: "c1", Balance: 200000, IsExternal: false})
	f.accountRepo.addAccount(account.Account{ID: "a3", ClientID: "c1", Balance: 50000, IsExternal: true})

	cr := &clientResolver{f.resolver}
	aum, err := cr.Aum(ctx, &model.Client{ID: "c1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if aum != 350000 {
		t.Errorf("expected AUM 350000, got %f", aum)
	}
}

// Test Anchor 37: Client.accounts returns only internal
func TestClientAccounts_InternalOnly(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	f.accountRepo.addAccount(account.Account{ID: "a1", ClientID: "c1", Balance: 100000, IsExternal: false})
	f.accountRepo.addAccount(account.Account{ID: "a2", ClientID: "c1", Balance: 200000, IsExternal: false})
	f.accountRepo.addAccount(account.Account{ID: "a3", ClientID: "c1", Balance: 50000, IsExternal: true})

	cr := &clientResolver{f.resolver}
	accounts, err := cr.Accounts(ctx, &model.Client{ID: "c1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(accounts) != 2 {
		t.Errorf("expected 2 internal accounts, got %d", len(accounts))
	}
}

// Test Anchor 38: Client.externalAccounts returns only external
func TestClientExternalAccounts(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	f.accountRepo.addAccount(account.Account{ID: "a1", ClientID: "c1", Balance: 100000, IsExternal: false})
	f.accountRepo.addAccount(account.Account{ID: "a2", ClientID: "c1", Balance: 200000, IsExternal: false})
	f.accountRepo.addAccount(account.Account{ID: "a3", ClientID: "c1", Balance: 50000, IsExternal: true})

	cr := &clientResolver{f.resolver}
	accounts, err := cr.ExternalAccounts(ctx, &model.Client{ID: "c1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(accounts) != 1 {
		t.Errorf("expected 1 external account, got %d", len(accounts))
	}
}

// Test Anchor 39: Client.health returns RED for CRITICAL alert
func TestClientHealth_Red(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	f.alertService.healthByClient["c1"] = alert.HealthRed

	cr := &clientResolver{f.resolver}
	health, err := cr.Health(ctx, &model.Client{ID: "c1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if health != model.HealthStatusRed {
		t.Errorf("expected RED, got %s", health)
	}
}

// Test Anchor 40: Client.household returns nil when no household
func TestClientHousehold_Nil(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	cr := &clientResolver{f.resolver}
	h, err := cr.Household(ctx, &model.Client{ID: "c1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h != nil {
		t.Errorf("expected nil household, got %+v", h)
	}
}

// Test Anchor 41: Alert.linkedActionItems returns items
func TestAlertLinkedActionItems(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	f.actionItemSvc.addItem(actionitem.ActionItem{ID: "ai1", ClientID: "c1", Text: "Item 1", Status: actionitem.ActionItemStatusPending, CreatedAt: time.Now()})
	f.actionItemSvc.addItem(actionitem.ActionItem{ID: "ai2", ClientID: "c1", Text: "Item 2", Status: actionitem.ActionItemStatusPending, CreatedAt: time.Now()})

	ar := &alertResolver{f.resolver}
	items, err := ar.LinkedActionItems(ctx, &model.Alert{
		LinkedActionItems: []*model.ActionItem{{ID: "ai1"}, {ID: "ai2"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 linked items, got %d", len(items))
	}
}

// Test Anchor 42: Alert.linkedActionItems returns empty list for no IDs
func TestAlertLinkedActionItems_Empty(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	ar := &alertResolver{f.resolver}
	items, err := ar.LinkedActionItems(ctx, &model.Alert{
		LinkedActionItems: []*model.ActionItem{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

// Test Anchor 43: ActionItem.alert returns nil when alertID is nil
func TestActionItemAlert_Nil(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	air := &actionItemResolver{f.resolver}
	a, err := air.Alert(ctx, &model.ActionItem{Alert: nil})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a != nil {
		t.Errorf("expected nil alert, got %+v", a)
	}
}

// Test Anchor 44: ActionItem.alert returns linked alert
func TestActionItemAlert_Found(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	f.alertRepo.addAlert(alert.Alert{
		ID: "al1", ClientID: "c1", Severity: alert.SeverityCritical,
		Status: alert.StatusOpen, Summary: "Test", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	air := &actionItemResolver{f.resolver}
	a, err := air.Alert(ctx, &model.ActionItem{Alert: &model.Alert{ID: "al1"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a == nil || a.ID != "al1" {
		t.Errorf("expected alert al1, got %v", a)
	}
}

// Test: Household.members resolves
func TestHouseholdMembers(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	hid := "h1"
	f.clientRepo.addClient(client.Client{ID: "c1", AdvisorID: "a1", Name: "Swati", HouseholdID: &hid})
	f.clientRepo.addClient(client.Client{ID: "c2", AdvisorID: "a1", Name: "Rohan", HouseholdID: &hid})

	hr := &householdResolver{f.resolver}
	members, err := hr.Members(ctx, &model.Household{ID: "h1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("expected 2 members, got %d", len(members))
	}
}

// Test: Transfer.client resolves via stub pattern
func TestTransferClient(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	f.clientRepo.addClient(client.Client{ID: "c1", AdvisorID: "a1", Name: "Alice"})

	tr := &transferResolver{f.resolver}
	c, err := tr.Client(ctx, &model.Transfer{Client: &model.Client{ID: "c1"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Name != "Alice" {
		t.Errorf("expected Alice, got %s", c.Name)
	}
}

// Test: ActionItem.client resolves via stub pattern
func TestActionItemClient(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	f.clientRepo.addClient(client.Client{ID: "c1", AdvisorID: "a1", Name: "Alice"})

	air := &actionItemResolver{f.resolver}
	c, err := air.Client(ctx, &model.ActionItem{
		ID:     "ai1",
		Client: &model.Client{ID: "c1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Name != "Alice" {
		t.Errorf("expected Alice, got %s", c.Name)
	}
}

// Test: Alert.client resolves via stub pattern
func TestAlertClient(t *testing.T) {
	f := newTestFixture()
	ctx := context.Background()

	f.clientRepo.addClient(client.Client{ID: "c1", AdvisorID: "a1", Name: "Alice"})

	ar := &alertResolver{f.resolver}
	c, err := ar.Client(ctx, &model.Alert{
		ID:     "al1",
		Client: &model.Client{ID: "c1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Name != "Alice" {
		t.Errorf("expected Alice, got %s", c.Name)
	}
}
