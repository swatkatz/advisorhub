package temporal

import (
	"context"
	"sync"
)

// mockClientRepo is an in-memory ClientRepository for testing.
type mockClientRepo struct {
	clients map[string][]Client // advisorID → clients
}

func newMockClientRepo() *mockClientRepo {
	return &mockClientRepo{clients: make(map[string][]Client)}
}

func (r *mockClientRepo) AddClient(advisorID string, c Client) {
	r.clients[advisorID] = append(r.clients[advisorID], c)
}

func (r *mockClientRepo) GetClients(_ context.Context, advisorID string) ([]Client, error) {
	return r.clients[advisorID], nil
}

// mockAccountRepo is an in-memory AccountRepository for testing.
type mockAccountRepo struct {
	accounts map[string][]Account // clientID → accounts
}

func newMockAccountRepo() *mockAccountRepo {
	return &mockAccountRepo{accounts: make(map[string][]Account)}
}

func (r *mockAccountRepo) AddAccount(a Account) {
	r.accounts[a.ClientID] = append(r.accounts[a.ClientID], a)
}

func (r *mockAccountRepo) GetAccountsByClientID(_ context.Context, clientID string) ([]Account, error) {
	return r.accounts[clientID], nil
}

// mockRESPBenRepo is an in-memory RESPBeneficiaryRepository for testing.
type mockRESPBenRepo struct {
	beneficiaries map[string][]RESPBeneficiary // clientID → beneficiaries
}

func newMockRESPBenRepo() *mockRESPBenRepo {
	return &mockRESPBenRepo{beneficiaries: make(map[string][]RESPBeneficiary)}
}

func (r *mockRESPBenRepo) AddBeneficiary(b RESPBeneficiary) {
	r.beneficiaries[b.ClientID] = append(r.beneficiaries[b.ClientID], b)
}

func (r *mockRESPBenRepo) GetRESPBeneficiariesByClientID(_ context.Context, clientID string) ([]RESPBeneficiary, error) {
	return r.beneficiaries[clientID], nil
}

// mockContribEngine is a mock ContributionEngine for testing.
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

func (e *mockContribEngine) GetRoom(_ context.Context, clientID string, accountType string, taxYear int) (float64, error) {
	key := roomKey(clientID, accountType, taxYear)
	if err, ok := e.errors[key]; ok {
		return 0, err
	}
	return e.rooms[key], nil
}

// mockEventBus captures published events for assertion.
type mockEventBus struct {
	mu     sync.Mutex
	events []EventEnvelope
}

func newMockEventBus() *mockEventBus {
	return &mockEventBus{}
}

func (b *mockEventBus) Publish(_ context.Context, envelope EventEnvelope) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, envelope)
	return nil
}

func (b *mockEventBus) Events() []EventEnvelope {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := make([]EventEnvelope, len(b.events))
	copy(result, b.events)
	return result
}

func (b *mockEventBus) EventsByType(eventType string) []EventEnvelope {
	b.mu.Lock()
	defer b.mu.Unlock()
	var result []EventEnvelope
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
