package actionitem

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// actionItemRow maps to the action_items table.
type actionItemRow struct {
	ID             string            `db:"id"`
	ClientID       string            `db:"client_id"`
	AlertID        *string           `db:"alert_id"`
	Text           string            `db:"text"`
	Status         ActionItemStatus  `db:"status"`
	DueDate        *time.Time        `db:"due_date"`
	CreatedAt      time.Time         `db:"created_at"`
	ResolvedAt     *time.Time        `db:"resolved_at"`
	ResolutionNote *string           `db:"resolution_note"`
}

func actionItemFromRow(r actionItemRow) ActionItem {
	return ActionItem{
		ID:             r.ID,
		ClientID:       r.ClientID,
		AlertID:        r.AlertID,
		Text:           r.Text,
		Status:         r.Status,
		DueDate:        r.DueDate,
		CreatedAt:      r.CreatedAt,
		ResolvedAt:     r.ResolvedAt,
		ResolutionNote: r.ResolutionNote,
	}
}

// PostgresActionItemRepo implements ActionItemRepository using PostgreSQL.
type PostgresActionItemRepo struct {
	db *sqlx.DB
}

// NewPostgresActionItemRepo creates a new PostgresActionItemRepo.
func NewPostgresActionItemRepo(db *sqlx.DB) *PostgresActionItemRepo {
	return &PostgresActionItemRepo{db: db}
}

func (r *PostgresActionItemRepo) CreateActionItem(ctx context.Context, item *ActionItem) (*ActionItem, error) {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO action_items (id, client_id, alert_id, text, status, due_date, created_at, resolved_at, resolution_note)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		item.ID, item.ClientID, item.AlertID, item.Text, item.Status,
		item.DueDate, item.CreatedAt, item.ResolvedAt, item.ResolutionNote)
	if err != nil {
		return nil, fmt.Errorf("creating action item: %w", err)
	}
	result := *item
	return &result, nil
}

func (r *PostgresActionItemRepo) GetActionItem(ctx context.Context, id string) (*ActionItem, error) {
	var row actionItemRow
	err := r.db.GetContext(ctx, &row,
		`SELECT id, client_id, alert_id, text, status, due_date, created_at, resolved_at, resolution_note
		 FROM action_items WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("getting action item %s: %w", id, err)
	}
	item := actionItemFromRow(row)
	return &item, nil
}

func (r *PostgresActionItemRepo) GetActionItemsByClientID(ctx context.Context, clientID string) ([]ActionItem, error) {
	var rows []actionItemRow
	err := r.db.SelectContext(ctx, &rows,
		`SELECT id, client_id, alert_id, text, status, due_date, created_at, resolved_at, resolution_note
		 FROM action_items WHERE client_id = $1 ORDER BY id`, clientID)
	if err != nil {
		return nil, fmt.Errorf("getting action items for client %s: %w", clientID, err)
	}
	items := make([]ActionItem, len(rows))
	for i, row := range rows {
		items[i] = actionItemFromRow(row)
	}
	return items, nil
}

func (r *PostgresActionItemRepo) GetActionItemsByAlertID(ctx context.Context, alertID string) ([]ActionItem, error) {
	var rows []actionItemRow
	err := r.db.SelectContext(ctx, &rows,
		`SELECT id, client_id, alert_id, text, status, due_date, created_at, resolved_at, resolution_note
		 FROM action_items WHERE alert_id = $1 ORDER BY id`, alertID)
	if err != nil {
		return nil, fmt.Errorf("getting action items for alert %s: %w", alertID, err)
	}
	items := make([]ActionItem, len(rows))
	for i, row := range rows {
		items[i] = actionItemFromRow(row)
	}
	return items, nil
}

func (r *PostgresActionItemRepo) UpdateActionItem(ctx context.Context, item *ActionItem) (*ActionItem, error) {
	_, err := r.db.ExecContext(ctx,
		`UPDATE action_items SET text = $2, status = $3, due_date = $4, resolved_at = $5, resolution_note = $6
		 WHERE id = $1`,
		item.ID, item.Text, item.Status, item.DueDate, item.ResolvedAt, item.ResolutionNote)
	if err != nil {
		return nil, fmt.Errorf("updating action item %s: %w", item.ID, err)
	}
	result := *item
	return &result, nil
}
