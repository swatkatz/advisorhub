package account

import (
	"context"
	"fmt"
	"sort"
)

// MemoryAccountRepo is an in-memory implementation of AccountRepository for testing.
type MemoryAccountRepo struct {
	accounts map[string]Account
}

func newMemoryAccountRepo() *MemoryAccountRepo {
	return &MemoryAccountRepo{accounts: make(map[string]Account)}
}

func (r *MemoryAccountRepo) GetAccount(_ context.Context, id string) (*Account, error) {
	a, ok := r.accounts[id]
	if !ok {
		return nil, fmt.Errorf("getting account: not found: %s", id)
	}
	return &a, nil
}

func (r *MemoryAccountRepo) GetAccountsByClientID(_ context.Context, clientID string) ([]Account, error) {
	var result []Account
	for _, a := range r.accounts {
		if a.ClientID == clientID {
			result = append(result, a)
		}
	}
	if result == nil {
		result = []Account{}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func (r *MemoryAccountRepo) UpdateFHSALifetimeContributions(_ context.Context, accountID string, total float64) error {
	a, ok := r.accounts[accountID]
	if !ok {
		return fmt.Errorf("updating FHSA lifetime contributions: not found: %s", accountID)
	}
	a.FHSALifetimeContributions = total
	r.accounts[accountID] = a
	return nil
}

// MemoryRESPBeneficiaryRepo is an in-memory implementation of RESPBeneficiaryRepository for testing.
type MemoryRESPBeneficiaryRepo struct {
	beneficiaries map[string]RESPBeneficiary
}

func newMemoryRESPBeneficiaryRepo() *MemoryRESPBeneficiaryRepo {
	return &MemoryRESPBeneficiaryRepo{beneficiaries: make(map[string]RESPBeneficiary)}
}

func (r *MemoryRESPBeneficiaryRepo) GetRESPBeneficiary(_ context.Context, id string) (*RESPBeneficiary, error) {
	b, ok := r.beneficiaries[id]
	if !ok {
		return nil, fmt.Errorf("getting RESP beneficiary: not found: %s", id)
	}
	return &b, nil
}

func (r *MemoryRESPBeneficiaryRepo) GetRESPBeneficiariesByClientID(_ context.Context, clientID string) ([]RESPBeneficiary, error) {
	var result []RESPBeneficiary
	for _, b := range r.beneficiaries {
		if b.ClientID == clientID {
			result = append(result, b)
		}
	}
	if result == nil {
		result = []RESPBeneficiary{}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func (r *MemoryRESPBeneficiaryRepo) UpdateLifetimeContributions(_ context.Context, beneficiaryID string, total float64) error {
	b, ok := r.beneficiaries[beneficiaryID]
	if !ok {
		return fmt.Errorf("updating RESP beneficiary lifetime contributions: not found: %s", beneficiaryID)
	}
	b.LifetimeContributions = total
	r.beneficiaries[beneficiaryID] = b
	return nil
}
