package contribution

import (
	"context"
	"fmt"
	"sync"

	"github.com/swatkatz/advisorhub/backend/internal/account"
	"github.com/swatkatz/advisorhub/backend/internal/eventbus"
)

// memoryContributionRepo is an in-memory ContributionRepository for testing.
type memoryContributionRepo struct {
	mu           sync.RWMutex
	contributions []Contribution
	limits        []ClientContributionLimit
}

func newMemoryContributionRepo() *memoryContributionRepo {
	return &memoryContributionRepo{}
}

func (r *memoryContributionRepo) GetContributionsByClient(_ context.Context, clientID string, taxYear int) ([]Contribution, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []Contribution
	for _, c := range r.contributions {
		if c.ClientID == clientID && c.TaxYear == taxYear {
			result = append(result, c)
		}
	}
	return result, nil
}

func (r *memoryContributionRepo) RecordContribution(_ context.Context, c *Contribution) (*Contribution, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c.ID == "" {
		c.ID = fmt.Sprintf("contrib_%d", len(r.contributions)+1)
	}
	r.contributions = append(r.contributions, *c)
	return c, nil
}

func (r *memoryContributionRepo) GetClientContributionLimit(_ context.Context, clientID string, taxYear int) (*ClientContributionLimit, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, l := range r.limits {
		if l.ClientID == clientID && l.TaxYear == taxYear {
			return &l, nil
		}
	}
	return nil, nil
}

func (r *memoryContributionRepo) SaveClientContributionLimit(_ context.Context, limit *ClientContributionLimit) (*ClientContributionLimit, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit.ID == "" {
		limit.ID = fmt.Sprintf("ccl_%d", len(r.limits)+1)
	}
	// Upsert by client_id + tax_year
	for i, l := range r.limits {
		if l.ClientID == limit.ClientID && l.TaxYear == limit.TaxYear {
			r.limits[i] = *limit
			return limit, nil
		}
	}
	r.limits = append(r.limits, *limit)
	return limit, nil
}

// addContribution is a test helper to seed contribution data.
func (r *memoryContributionRepo) addContribution(c Contribution) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.contributions = append(r.contributions, c)
}

// addLimit is a test helper to seed contribution limit data.
func (r *memoryContributionRepo) addLimit(l ClientContributionLimit) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.limits = append(r.limits, l)
}

// memoryAccountRepo is an in-memory account.AccountRepository for testing.
type memoryAccountRepo struct {
	mu       sync.RWMutex
	accounts []account.Account
}

func newMemoryAccountRepo() *memoryAccountRepo {
	return &memoryAccountRepo{}
}

func (r *memoryAccountRepo) GetAccount(_ context.Context, id string) (*account.Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, a := range r.accounts {
		if a.ID == id {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("account %s not found", id)
}

func (r *memoryAccountRepo) GetAccountsByClientID(_ context.Context, clientID string) ([]account.Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []account.Account
	for _, a := range r.accounts {
		if a.ClientID == clientID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (r *memoryAccountRepo) UpdateFHSALifetimeContributions(_ context.Context, accountID string, total float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, a := range r.accounts {
		if a.ID == accountID {
			r.accounts[i].FHSALifetimeContributions = total
			return nil
		}
	}
	return fmt.Errorf("account %s not found", accountID)
}

func (r *memoryAccountRepo) addAccount(a account.Account) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.accounts = append(r.accounts, a)
}

// memoryRESPBeneficiaryRepo is an in-memory account.RESPBeneficiaryRepository for testing.
type memoryRESPBeneficiaryRepo struct {
	mu            sync.RWMutex
	beneficiaries []account.RESPBeneficiary
}

func newMemoryRESPBeneficiaryRepo() *memoryRESPBeneficiaryRepo {
	return &memoryRESPBeneficiaryRepo{}
}

func (r *memoryRESPBeneficiaryRepo) GetRESPBeneficiary(_ context.Context, id string) (*account.RESPBeneficiary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, b := range r.beneficiaries {
		if b.ID == id {
			return &b, nil
		}
	}
	return nil, fmt.Errorf("beneficiary %s not found", id)
}

func (r *memoryRESPBeneficiaryRepo) GetRESPBeneficiariesByClientID(_ context.Context, clientID string) ([]account.RESPBeneficiary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []account.RESPBeneficiary
	for _, b := range r.beneficiaries {
		if b.ClientID == clientID {
			result = append(result, b)
		}
	}
	return result, nil
}

func (r *memoryRESPBeneficiaryRepo) UpdateLifetimeContributions(_ context.Context, beneficiaryID string, total float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, b := range r.beneficiaries {
		if b.ID == beneficiaryID {
			r.beneficiaries[i].LifetimeContributions = total
			return nil
		}
	}
	return fmt.Errorf("beneficiary %s not found", beneficiaryID)
}

func (r *memoryRESPBeneficiaryRepo) addBeneficiary(b account.RESPBeneficiary) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.beneficiaries = append(r.beneficiaries, b)
}

// memoryEventBus captures published events for test assertions.
type memoryEventBus struct {
	mu     sync.Mutex
	events []eventbus.EventEnvelope
}

func newMemoryEventBus() *memoryEventBus {
	return &memoryEventBus{}
}

func (b *memoryEventBus) Publish(_ context.Context, envelope eventbus.EventEnvelope) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, envelope)
	return nil
}

func (b *memoryEventBus) Subscribe(_ string) <-chan eventbus.EventEnvelope {
	return make(chan eventbus.EventEnvelope)
}

func (b *memoryEventBus) publishedEvents() []eventbus.EventEnvelope {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := make([]eventbus.EventEnvelope, len(b.events))
	copy(result, b.events)
	return result
}

func (b *memoryEventBus) eventsByType(eventType string) []eventbus.EventEnvelope {
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

func (b *memoryEventBus) reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = nil
}
