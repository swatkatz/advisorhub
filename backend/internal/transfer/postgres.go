package transfer

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// transferRow maps to the transfers database table.
type transferRow struct {
	ID                string    `db:"id"`
	ClientID          string    `db:"client_id"`
	SourceInstitution string    `db:"source_institution"`
	AccountType       string    `db:"account_type"`
	Amount            float64   `db:"amount"`
	Status            string    `db:"status"`
	InitiatedAt       time.Time `db:"initiated_at"`
	LastStatusChange  time.Time `db:"last_status_change"`
}

func transferFromRow(r transferRow) Transfer {
	return Transfer{
		ID:                r.ID,
		ClientID:          r.ClientID,
		SourceInstitution: r.SourceInstitution,
		AccountType:       r.AccountType,
		Amount:            r.Amount,
		Status:            TransferStatus(r.Status),
		InitiatedAt:       r.InitiatedAt,
		LastStatusChange:  r.LastStatusChange,
	}
}

// PostgresTransferRepo implements TransferRepository backed by PostgreSQL.
type PostgresTransferRepo struct {
	db  *sqlx.DB
	now func() time.Time
}

// NewPostgresTransferRepo creates a new PostgreSQL-backed repository.
func NewPostgresTransferRepo(db *sqlx.DB, now func() time.Time) *PostgresTransferRepo {
	return &PostgresTransferRepo{db: db, now: now}
}

func (r *PostgresTransferRepo) GetTransfer(ctx context.Context, id string) (*Transfer, error) {
	var row transferRow
	err := r.db.GetContext(ctx, &row,
		"SELECT id, client_id, source_institution, account_type, amount, status, initiated_at, last_status_change FROM transfers WHERE id = $1",
		id)
	if err != nil {
		return nil, fmt.Errorf("getting transfer %s: %w", id, err)
	}
	t := transferFromRow(row)
	return &t, nil
}

func (r *PostgresTransferRepo) GetTransfersByClientID(ctx context.Context, clientID string) ([]Transfer, error) {
	var rows []transferRow
	err := r.db.SelectContext(ctx, &rows,
		"SELECT id, client_id, source_institution, account_type, amount, status, initiated_at, last_status_change FROM transfers WHERE client_id = $1 ORDER BY id",
		clientID)
	if err != nil {
		return nil, fmt.Errorf("getting transfers for client %s: %w", clientID, err)
	}
	result := make([]Transfer, len(rows))
	for i, row := range rows {
		result[i] = transferFromRow(row)
	}
	return result, nil
}

func (r *PostgresTransferRepo) GetActiveTransfers(ctx context.Context) ([]Transfer, error) {
	var rows []transferRow
	err := r.db.SelectContext(ctx, &rows,
		"SELECT id, client_id, source_institution, account_type, amount, status, initiated_at, last_status_change FROM transfers WHERE status != 'INVESTED' ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("getting active transfers: %w", err)
	}
	result := make([]Transfer, len(rows))
	for i, row := range rows {
		result[i] = transferFromRow(row)
	}
	return result, nil
}

func (r *PostgresTransferRepo) CreateTransfer(ctx context.Context, transfer *Transfer) (*Transfer, error) {
	transfer.LastStatusChange = transfer.InitiatedAt
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO transfers (id, client_id, source_institution, account_type, amount, status, initiated_at, last_status_change)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		transfer.ID, transfer.ClientID, transfer.SourceInstitution, transfer.AccountType,
		transfer.Amount, string(transfer.Status), transfer.InitiatedAt, transfer.LastStatusChange)
	if err != nil {
		return nil, fmt.Errorf("creating transfer %s: %w", transfer.ID, err)
	}
	return transfer, nil
}

func (r *PostgresTransferRepo) UpdateTransferStatus(ctx context.Context, id string, newStatus TransferStatus) (*Transfer, error) {
	current, err := r.GetTransfer(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("updating transfer status: %w", err)
	}

	expected, err := nextStatus(current.Status)
	if err != nil {
		return nil, fmt.Errorf("updating transfer %s: %w", id, err)
	}
	if newStatus != expected {
		return nil, fmt.Errorf("invalid transition from %s to %s: expected %s", current.Status, newStatus, expected)
	}

	now := r.now()
	_, err = r.db.ExecContext(ctx,
		"UPDATE transfers SET status = $1, last_status_change = $2 WHERE id = $3",
		string(newStatus), now, id)
	if err != nil {
		return nil, fmt.Errorf("updating transfer %s status: %w", id, err)
	}

	current.Status = newStatus
	current.LastStatusChange = now
	return current, nil
}
