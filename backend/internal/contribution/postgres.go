package contribution

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type contributionRow struct {
	ID          string    `db:"id"`
	ClientID    string    `db:"client_id"`
	AccountID   string    `db:"account_id"`
	AccountType string    `db:"account_type"`
	Amount      float64   `db:"amount"`
	Date        time.Time `db:"date"`
	TaxYear     int       `db:"tax_year"`
}

func contributionFromRow(r contributionRow) Contribution {
	return Contribution{
		ID:          r.ID,
		ClientID:    r.ClientID,
		AccountID:   r.AccountID,
		AccountType: r.AccountType,
		Amount:      r.Amount,
		Date:        r.Date,
		TaxYear:     r.TaxYear,
	}
}

type clientContributionLimitRow struct {
	ID                 string  `db:"id"`
	ClientID           string  `db:"client_id"`
	TaxYear            int     `db:"tax_year"`
	RRSPDeductionLimit float64 `db:"rrsp_deduction_limit"`
}

func limitFromRow(r clientContributionLimitRow) ClientContributionLimit {
	return ClientContributionLimit{
		ID:                 r.ID,
		ClientID:           r.ClientID,
		TaxYear:            r.TaxYear,
		RRSPDeductionLimit: r.RRSPDeductionLimit,
	}
}

// PostgresContributionRepo implements ContributionRepository using PostgreSQL.
type PostgresContributionRepo struct {
	db *sqlx.DB
}

// NewPostgresContributionRepo creates a new PostgresContributionRepo.
func NewPostgresContributionRepo(db *sqlx.DB) *PostgresContributionRepo {
	return &PostgresContributionRepo{db: db}
}

func (r *PostgresContributionRepo) GetContributionsByClient(ctx context.Context, clientID string, taxYear int) ([]Contribution, error) {
	var rows []contributionRow
	err := r.db.SelectContext(ctx, &rows,
		"SELECT id, client_id, account_id, account_type, amount, date, tax_year FROM contributions WHERE client_id = $1 AND tax_year = $2 ORDER BY id",
		clientID, taxYear)
	if err != nil {
		return nil, fmt.Errorf("getting contributions for client %s year %d: %w", clientID, taxYear, err)
	}
	result := make([]Contribution, len(rows))
	for i, row := range rows {
		result[i] = contributionFromRow(row)
	}
	return result, nil
}

func (r *PostgresContributionRepo) RecordContribution(ctx context.Context, c *Contribution) (*Contribution, error) {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO contributions (id, client_id, account_id, account_type, amount, date, tax_year) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		c.ID, c.ClientID, c.AccountID, c.AccountType, c.Amount, c.Date, c.TaxYear)
	if err != nil {
		return nil, fmt.Errorf("recording contribution: %w", err)
	}
	return c, nil
}

func (r *PostgresContributionRepo) GetClientContributionLimit(ctx context.Context, clientID string, taxYear int) (*ClientContributionLimit, error) {
	var row clientContributionLimitRow
	err := r.db.GetContext(ctx, &row,
		"SELECT id, client_id, tax_year, rrsp_deduction_limit FROM client_contribution_limits WHERE client_id = $1 AND tax_year = $2",
		clientID, taxYear)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("getting contribution limit for client %s year %d: %w", clientID, taxYear, err)
	}
	l := limitFromRow(row)
	return &l, nil
}

func (r *PostgresContributionRepo) SaveClientContributionLimit(ctx context.Context, limit *ClientContributionLimit) (*ClientContributionLimit, error) {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO client_contribution_limits (id, client_id, tax_year, rrsp_deduction_limit)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (client_id, tax_year) DO UPDATE SET rrsp_deduction_limit = $4`,
		limit.ID, limit.ClientID, limit.TaxYear, limit.RRSPDeductionLimit)
	if err != nil {
		return nil, fmt.Errorf("saving contribution limit: %w", err)
	}
	return limit, nil
}
