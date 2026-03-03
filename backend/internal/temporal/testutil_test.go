package temporal

import (
	"context"
	"fmt"
	"sync"

	"github.com/swatkatz/advisorhub/backend/internal/account"
	"github.com/swatkatz/advisorhub/backend/internal/client"
	"github.com/swatkatz/advisorhub/backend/internal/contribution"
	"github.com/swatkatz/advisorhub/backend/internal/eventbus"
)

// mockClientRepo is an in-memory client.ClientRepository for testing.
type mockClientRepo struct {
	clients map[string][]client.Client // advisorID → clients
}

func newMockClientRepo() *mockClientRepo {
	return &mockClientRepo{clients: make(map[string][]client.Client)}
}

func (r *mockClientRepo) AddClient(advisorID string, c client.Client) {
	r.clients[advisorID] = append(r.clients[advisorID], c)
}

func (r *mockClientRepo) GetClient(_ context.Context, id string) (*client.Client, error) {
	for _, clients := range r.clients {
		for i, c := range clients {
			if c.ID == id {
				return &clients[i], nil
			}
		}
	}
	return nil, fmt.Errorf("client %s not found", id)
}

func (r *mockClientRepo) GetClients(_ context.Context, advisorID string) ([]client.Client, error) {
	return r.clients[advisorID], nil
}

func (r *mockClientRepo) GetClientsByHouseholdID(_ context.Context, _ string) ([]client.Client, error) {
	return nil, nil
}

// mockAccountRepo is an in-memory account.AccountRepository for testing.
type mockAccountRepo struct {
	accounts map[string][]account.Account // clientID → accounts
}

func newMockAccountRepo() *mockAccountRepo {
	return &mockAccountRepo{accounts: make(map[string][]account.Account)}
}

func (r *mockAccountRepo) AddAccount(a account.Account) {
	r.accounts[a.ClientID] = append(r.accounts[a.ClientID], a)
}

func (r *mockAccountRepo) GetAccount(_ context.Context, id string) (*account.Account, error) {
	for _, accounts := range r.accounts {
		for i, a := range accounts {
			if a.ID == id {
				return &accounts[i], nil
			}
		}
	}
	return nil, fmt.Errorf("account %s not found", id)
}

func (r *mockAccountRepo) GetAccountsByClientID(_ context.Context, clientID string) ([]account.Account, error) {
	return r.accounts[clientID], nil
}

func (r *mockAccountRepo) UpdateFHSALifetimeContributions(_ context.Context, _ string, _ float64) error {
	return nil
}

// mockRESPBenRepo is an in-memory account.RESPBeneficiaryRepository for testing.
type mockRESPBenRepo struct {
	beneficiaries map[string][]account.RESPBeneficiary // clientID → beneficiaries
}

func newMockRESPBenRepo() *mockRESPBenRepo {
	return &mockRESPBenRepo{beneficiaries: make(map[string][]account.RESPBeneficiary)}
}

func (r *mockRESPBenRepo) AddBeneficiary(b account.RESPBeneficiary) {
	r.beneficiaries[b.ClientID] = append(r.beneficiaries[b.ClientID], b)
}

func (r *mockRESPBenRepo) GetRESPBeneficiary(_ context.Context, id string) (*account.RESPBeneficiary, error) {
	for _, bens := range r.beneficiaries {
		for i, b := range bens {
			if b.ID == id {
				return &bens[i], nil
			}
		}
	}
	return nil, fmt.Errorf("beneficiary %s not found", id)
}

func (r *mockRESPBenRepo) GetRESPBeneficiariesByClientID(_ context.Context, clientID string) ([]account.RESPBeneficiary, error) {
	return r.beneficiaries[clientID], nil
}

func (r *mockRESPBenRepo) UpdateLifetimeContributions(_ context.Context, _ string, _ float64) error {
	return nil
}

// mockContribEngine is a mock contribution.ContributionEngine for testing.
type mockContribEngine struct {
	rooms  map[string]float64 // "clientID:accountType:taxYear" → room
	errors map[string]error   // "clientID:accountType:taxYear" → error
}

func newMockContribEngine() *mockContribEngine {
	return &mockContribEngine{
		rooms:  make(map[string]float64),
		errors: make(map[string]error),
	}
}

func roomKey(clientID, accountType string, taxYear int) string {
	return clientID + ":" + accountType + ":" + itoa(taxYear)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

func (e *mockContribEngine) SetRoom(clientID, accountType string, taxYear int, room float64) {
	e.rooms[roomKey(clientID, accountType, taxYear)] = room
}

func (e *mockContribEngine) SetError(clientID, accountType string, taxYear int, err error) {
	e.errors[roomKey(clientID, accountType, taxYear)] = err
}

func (e *mockContribEngine) AnalyzeClient(_ context.Context, _ string, _ int) error {
	return nil
}

func (e *mockContribEngine) GetContributionSummary(_ context.Context, _ string, _ int) (*contribution.ContributionSummary, error) {
	return nil, nil
}

func (e *mockContribEngine) GetRoom(_ context.Context, clientID string, accountType string, taxYear int) (float64, error) {
	key := roomKey(clientID, accountType, taxYear)
	if err, ok := e.errors[key]; ok {
		return 0, err
	}
	return e.rooms[key], nil
}

func (e *mockContribEngine) RecordContribution(_ context.Context, _ *contribution.Contribution) (*contribution.Contribution, error) {
	return nil, nil
}

// mockEventBus captures published events for assertion.
type mockEventBus struct {
	mu     sync.Mutex
	events []eventbus.EventEnvelope
}

func newMockEventBus() *mockEventBus {
	return &mockEventBus{}
}

func (b *mockEventBus) Publish(_ context.Context, envelope eventbus.EventEnvelope) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, envelope)
	return nil
}

func (b *mockEventBus) Subscribe(_ string) <-chan eventbus.EventEnvelope {
	return make(chan eventbus.EventEnvelope)
}

func (b *mockEventBus) Events() []eventbus.EventEnvelope {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := make([]eventbus.EventEnvelope, len(b.events))
	copy(result, b.events)
	return result
}

func (b *mockEventBus) EventsByType(eventType string) []eventbus.EventEnvelope {
	b.mu.Lock()
	defer b.mu.Unlock()
	var result []eventbus.EventEnvelope
	for _, e := range b.events {
		if e.Type == eventType {
			result = append(result, e)
		}
	}
	return result
}

func (b *mockEventBus) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = nil
}
