package account

import (
	"context"
	"time"
)

// AccountType represents the type of financial account.
type AccountType string

const (
	AccountTypeRRSP   AccountType = "RRSP"
	AccountTypeTFSA   AccountType = "TFSA"
	AccountTypeFHSA   AccountType = "FHSA"
	AccountTypeRESP   AccountType = "RESP"
	AccountTypeNonReg AccountType = "NON_REG"
)

// LifetimeCap returns the lifetime contribution cap for the account type.
// Returns nil for types with no lifetime cap.
func (t AccountType) LifetimeCap() *float64 {
	switch t {
	case AccountTypeFHSA:
		cap := 40000.0
		return &cap
	case AccountTypeRESP:
		cap := 50000.0
		return &cap
	default:
		return nil
	}
}

// Account represents a financial account belonging to a client.
type Account struct {
	ID                       string
	ClientID                 string
	AccountType              AccountType
	Institution              string
	Balance                  float64
	IsExternal               bool
	RESPBeneficiaryID        *string
	FHSALifetimeContributions float64
}

// RESPBeneficiary represents a child beneficiary of an RESP account.
type RESPBeneficiary struct {
	ID                    string
	ClientID              string
	Name                  string
	DateOfBirth           time.Time
	LifetimeContributions float64
}

// AccountRepository provides access to account data.
type AccountRepository interface {
	GetAccount(ctx context.Context, id string) (*Account, error)
	GetAccountsByClientID(ctx context.Context, clientID string) ([]Account, error)
	UpdateFHSALifetimeContributions(ctx context.Context, accountID string, total float64) error
}

// RESPBeneficiaryRepository provides access to RESP beneficiary data.
type RESPBeneficiaryRepository interface {
	GetRESPBeneficiary(ctx context.Context, id string) (*RESPBeneficiary, error)
	GetRESPBeneficiariesByClientID(ctx context.Context, clientID string) ([]RESPBeneficiary, error)
	UpdateLifetimeContributions(ctx context.Context, beneficiaryID string, total float64) error
}
