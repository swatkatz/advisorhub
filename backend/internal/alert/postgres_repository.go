package alert

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type postgresAlertRepository struct {
	db *sqlx.DB
}

// NewPostgresAlertRepository creates a PostgreSQL-backed AlertRepository.
func NewPostgresAlertRepository(db *sqlx.DB) AlertRepository {
	return &postgresAlertRepository{db: db}
}

// alertRow mirrors the alerts table with pq.StringArray for linked_action_item_ids.
type alertRow struct {
	Alert
	LinkedActionItemIDsPQ pq.StringArray `db:"linked_action_item_ids"`
}

func rowToAlert(r alertRow) *Alert {
	a := r.Alert
	a.LinkedActionItemIDs = []string(r.LinkedActionItemIDsPQ)
	if a.LinkedActionItemIDs == nil {
		a.LinkedActionItemIDs = []string{}
	}
	return &a
}

func (r *postgresAlertRepository) FindByConditionKey(ctx context.Context, conditionKey string) (*Alert, error) {
	var row alertRow
	err := r.db.GetContext(ctx, &row,
		`SELECT id, condition_key, client_id, severity, category, status,
		        snoozed_until, payload, summary, draft_message,
		        linked_action_item_ids, created_at, updated_at, resolved_at
		 FROM alerts
		 WHERE condition_key = $1 AND status != 'CLOSED'
		 ORDER BY created_at DESC
		 LIMIT 1`, conditionKey)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("finding alert by condition key: %w", err)
	}
	return rowToAlert(row), nil
}

func (r *postgresAlertRepository) GetAlert(ctx context.Context, id string) (*Alert, error) {
	var row alertRow
	err := r.db.GetContext(ctx, &row,
		`SELECT id, condition_key, client_id, severity, category, status,
		        snoozed_until, payload, summary, draft_message,
		        linked_action_item_ids, created_at, updated_at, resolved_at
		 FROM alerts WHERE id = $1`, id)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, ErrAlertNotFound
		}
		return nil, fmt.Errorf("getting alert: %w", err)
	}
	return rowToAlert(row), nil
}

func (r *postgresAlertRepository) GetAlertsByClientID(ctx context.Context, clientID string) ([]Alert, error) {
	var rows []alertRow
	err := r.db.SelectContext(ctx, &rows,
		`SELECT id, condition_key, client_id, severity, category, status,
		        snoozed_until, payload, summary, draft_message,
		        linked_action_item_ids, created_at, updated_at, resolved_at
		 FROM alerts WHERE client_id = $1 ORDER BY id`, clientID)
	if err != nil {
		return nil, fmt.Errorf("getting alerts by client: %w", err)
	}
	result := make([]Alert, len(rows))
	for i, row := range rows {
		result[i] = *rowToAlert(row)
	}
	return result, nil
}

func (r *postgresAlertRepository) GetAlertsByAdvisorID(ctx context.Context, advisorID string) ([]Alert, error) {
	var rows []alertRow
	err := r.db.SelectContext(ctx, &rows,
		`SELECT a.id, a.condition_key, a.client_id, a.severity, a.category, a.status,
		        a.snoozed_until, a.payload, a.summary, a.draft_message,
		        a.linked_action_item_ids, a.created_at, a.updated_at, a.resolved_at
		 FROM alerts a
		 JOIN clients c ON a.client_id = c.id
		 WHERE c.advisor_id = $1
		 ORDER BY a.id`, advisorID)
	if err != nil {
		return nil, fmt.Errorf("getting alerts by advisor: %w", err)
	}
	result := make([]Alert, len(rows))
	for i, row := range rows {
		result[i] = *rowToAlert(row)
	}
	return result, nil
}

func (r *postgresAlertRepository) CreateAlert(ctx context.Context, alert *Alert) (*Alert, error) {
	payloadBytes, err := json.Marshal(json.RawMessage(alert.Payload))
	if err != nil {
		return nil, fmt.Errorf("marshaling payload: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO alerts (id, condition_key, client_id, severity, category, status,
		                     snoozed_until, payload, summary, draft_message,
		                     linked_action_item_ids, created_at, updated_at, resolved_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		alert.ID, alert.ConditionKey, alert.ClientID, alert.Severity, alert.Category, alert.Status,
		alert.SnoozedUntil, payloadBytes, alert.Summary, alert.DraftMessage,
		pq.StringArray(alert.LinkedActionItemIDs), alert.CreatedAt, alert.UpdatedAt, alert.ResolvedAt)
	if err != nil {
		return nil, fmt.Errorf("creating alert: %w", err)
	}
	return alert, nil
}

func (r *postgresAlertRepository) UpdateAlert(ctx context.Context, alert *Alert) (*Alert, error) {
	payloadBytes, err := json.Marshal(json.RawMessage(alert.Payload))
	if err != nil {
		return nil, fmt.Errorf("marshaling payload: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`UPDATE alerts SET condition_key = $2, client_id = $3, severity = $4, category = $5,
		                   status = $6, snoozed_until = $7, payload = $8, summary = $9,
		                   draft_message = $10, linked_action_item_ids = $11,
		                   updated_at = $12, resolved_at = $13
		 WHERE id = $1`,
		alert.ID, alert.ConditionKey, alert.ClientID, alert.Severity, alert.Category,
		alert.Status, alert.SnoozedUntil, payloadBytes, alert.Summary,
		alert.DraftMessage, pq.StringArray(alert.LinkedActionItemIDs),
		alert.UpdatedAt, alert.ResolvedAt)
	if err != nil {
		return nil, fmt.Errorf("updating alert: %w", err)
	}
	return alert, nil
}
