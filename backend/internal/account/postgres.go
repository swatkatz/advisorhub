package account

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type accountRow struct {
	ID                        string    `db:"id"`
	ClientID                  string    `db:"client_id"`
	AccountType               string    `db:"account_type"`
	Institution               string    `db:"institution"`
	Balance                   float64   `db:"balance"`
	IsExternal                bool      `db:"is_external"`
	RESPBeneficiaryID         *string   `db:"resp_beneficiary_id"`
	FHSALifetimeContributions float64   `db:"fhsa_lifetime_contributions"`
	LastActivityDate          time.Time `db:"last_activity_date"`
}

func accountFromRow(r accountRow) Account {
	return Account{
		ID:                        r.ID,
		ClientID:                  r.ClientID,
		AccountType:               AccountType(r.AccountType),
		Institution:               r.Institution,
		Balance:                   r.Balance,
		IsExternal:                r.IsExternal,
		RESPBeneficiaryID:         r.RESPBeneficiaryID,
		FHSALifetimeContributions: r.FHSALifetimeContributions,
		LastActivityDate:          r.LastActivityDate,
	}
}

type respBeneficiaryRow struct {
	ID                    string    `db:"id"`
	ClientID              string    `db:"client_id"`
	Name                  string    `db:"name"`
	DateOfBirth           time.Time `db:"date_of_birth"`
	LifetimeContributions float64   `db:"lifetime_contributions"`
}

func respBeneficiaryFromRow(r respBeneficiaryRow) RESPBeneficiary {
	return RESPBeneficiary{
		ID:                    r.ID,
		ClientID:              r.ClientID,
		Name:                  r.Name,
		DateOfBirth:           r.DateOfBirth,
		LifetimeContributions: r.LifetimeContributions,
	}
}

// PostgresAccountRepo implements AccountRepository using PostgreSQL.
type PostgresAccountRepo struct {
	db *sqlx.DB
}

// NewPostgresAccountRepo creates a new PostgresAccountRepo.
func NewPostgresAccountRepo(db *sqlx.DB) *PostgresAccountRepo {
	return &PostgresAccountRepo{db: db}
}

func (r *PostgresAccountRepo) GetAccount(ctx context.Context, id string) (*Account, error) {
	var row accountRow
	err := r.db.GetContext(ctx, &row, "SELECT id, client_id, account_type, institution, balance, is_external, resp_beneficiary_id, fhsa_lifetime_contributions, last_activity_date FROM accounts WHERE id = $1", id)
	if err != nil {
		return nil, fmt.Errorf("getting account %s: %w", id, err)
	}
	a := accountFromRow(row)
	return &a, nil
}

func (r *PostgresAccountRepo) GetAccountsByClientID(ctx context.Context, clientID string) ([]Account, error) {
	var rows []accountRow
	err := r.db.SelectContext(ctx, &rows, "SELECT id, client_id, account_type, institution, balance, is_external, resp_beneficiary_id, fhsa_lifetime_contributions, last_activity_date FROM accounts WHERE client_id = $1 ORDER BY id", clientID)
	if err != nil {
		return nil, fmt.Errorf("getting accounts for client %s: %w", clientID, err)
	}
	result := make([]Account, len(rows))
	for i, row := range rows {
		result[i] = accountFromRow(row)
	}
	return result, nil
}

func (r *PostgresAccountRepo) UpdateFHSALifetimeContributions(ctx context.Context, accountID string, total float64) error {
	_, err := r.db.ExecContext(ctx, "UPDATE accounts SET fhsa_lifetime_contributions = $1 WHERE id = $2", total, accountID)
	if err != nil {
		return fmt.Errorf("updating FHSA lifetime contributions for account %s: %w", accountID, err)
	}
	return nil
}

// PostgresRESPBeneficiaryRepo implements RESPBeneficiaryRepository using PostgreSQL.
type PostgresRESPBeneficiaryRepo struct {
	db *sqlx.DB
}

// NewPostgresRESPBeneficiaryRepo creates a new PostgresRESPBeneficiaryRepo.
func NewPostgresRESPBeneficiaryRepo(db *sqlx.DB) *PostgresRESPBeneficiaryRepo {
	return &PostgresRESPBeneficiaryRepo{db: db}
}

func (r *PostgresRESPBeneficiaryRepo) GetRESPBeneficiary(ctx context.Context, id string) (*RESPBeneficiary, error) {
	var row respBeneficiaryRow
	err := r.db.GetContext(ctx, &row, "SELECT id, client_id, name, date_of_birth, lifetime_contributions FROM resp_beneficiaries WHERE id = $1", id)
	if err != nil {
		return nil, fmt.Errorf("getting RESP beneficiary %s: %w", id, err)
	}
	b := respBeneficiaryFromRow(row)
	return &b, nil
}

func (r *PostgresRESPBeneficiaryRepo) GetRESPBeneficiariesByClientID(ctx context.Context, clientID string) ([]RESPBeneficiary, error) {
	var rows []respBeneficiaryRow
	err := r.db.SelectContext(ctx, &rows, "SELECT id, client_id, name, date_of_birth, lifetime_contributions FROM resp_beneficiaries WHERE client_id = $1 ORDER BY id", clientID)
	if err != nil {
		return nil, fmt.Errorf("getting RESP beneficiaries for client %s: %w", clientID, err)
	}
	result := make([]RESPBeneficiary, len(rows))
	for i, row := range rows {
		result[i] = respBeneficiaryFromRow(row)
	}
	return result, nil
}

func (r *PostgresRESPBeneficiaryRepo) UpdateLifetimeContributions(ctx context.Context, beneficiaryID string, total float64) error {
	_, err := r.db.ExecContext(ctx, "UPDATE resp_beneficiaries SET lifetime_contributions = $1 WHERE id = $2", total, beneficiaryID)
	if err != nil {
		return fmt.Errorf("updating lifetime contributions for RESP beneficiary %s: %w", beneficiaryID, err)
	}
	return nil
}
