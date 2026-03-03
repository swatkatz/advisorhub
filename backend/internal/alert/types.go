package alert

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/swatkatz/advisorhub/backend/internal/eventbus"
)

// AlertSeverity indicates the urgency level of an alert.
type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "CRITICAL"
	SeverityUrgent   AlertSeverity = "URGENT"
	SeverityAdvisory AlertSeverity = "ADVISORY"
	SeverityInfo     AlertSeverity = "INFO"
)

// AlertStatus tracks the lifecycle state of an alert.
type AlertStatus string

const (
	StatusOpen    AlertStatus = "OPEN"
	StatusSnoozed AlertStatus = "SNOOZED"
	StatusActed   AlertStatus = "ACTED"
	StatusClosed  AlertStatus = "CLOSED"
)

// HealthStatus is computed from a client's most severe non-CLOSED alert.
type HealthStatus string

const (
	HealthGreen  HealthStatus = "GREEN"
	HealthYellow HealthStatus = "YELLOW"
	HealthRed    HealthStatus = "RED"
)

// ProcessSignal indicates the outcome of processing an event.
type ProcessSignal string

const (
	SignalCreated  ProcessSignal = "CREATED"
	SignalReopened ProcessSignal = "REOPENED"
	SignalUpdated  ProcessSignal = "UPDATED"
	SignalNoChange ProcessSignal = "NO_CHANGE"
)

var (
	ErrAlertNotFound = errors.New("alert not found")
	ErrAlertClosed   = errors.New("alert is closed")
	ErrInfoAlert     = errors.New("action not available for INFO alerts")
	ErrNotInfoAlert  = errors.New("acknowledge only available for INFO alerts")
)

// Alert event type constants (produced by this context).
const (
	EventAlertCreated = "AlertCreated"
	EventAlertUpdated = "AlertUpdated"
	EventAlertClosed  = "AlertClosed"
)

// Alert represents a system-generated notification for an advisor.
type Alert struct {
	ID                  string          `db:"id"`
	ConditionKey        string          `db:"condition_key"`
	ClientID            string          `db:"client_id"`
	Severity            AlertSeverity   `db:"severity"`
	Category            string          `db:"category"`
	Status              AlertStatus     `db:"status"`
	SnoozedUntil        *time.Time      `db:"snoozed_until"`
	Payload             json.RawMessage `db:"payload"`
	Summary             string          `db:"summary"`
	DraftMessage        *string         `db:"draft_message"`
	LinkedActionItemIDs []string        `db:"linked_action_item_ids"`
	CreatedAt           time.Time       `db:"created_at"`
	UpdatedAt           time.Time       `db:"updated_at"`
	ResolvedAt          *time.Time      `db:"resolved_at"`
}

// AlertRepository defines persistence operations for alerts.
type AlertRepository interface {
	FindByConditionKey(ctx context.Context, conditionKey string) (*Alert, error)
	GetAlert(ctx context.Context, id string) (*Alert, error)
	GetAlertsByClientID(ctx context.Context, clientID string) ([]Alert, error)
	GetAlertsByAdvisorID(ctx context.Context, advisorID string) ([]Alert, error)
	CreateAlert(ctx context.Context, alert *Alert) (*Alert, error)
	UpdateAlert(ctx context.Context, alert *Alert) (*Alert, error)
}

// AlertService orchestrates alert processing and advisor actions.
type AlertService interface {
	ProcessEvent(ctx context.Context, envelope eventbus.EventEnvelope) error
	Send(ctx context.Context, alertID string, message *string) (*Alert, error)
	Track(ctx context.Context, alertID string, actionItemText string) (*Alert, error)
	Snooze(ctx context.Context, alertID string, until *time.Time) (*Alert, error)
	Acknowledge(ctx context.Context, alertID string) (*Alert, error)
	Close(ctx context.Context, alertID string) (*Alert, error)
	ComputeHealthStatus(ctx context.Context, clientID string) (HealthStatus, error)
}

// Enhancer generates natural language summaries and draft messages for alerts.
type Enhancer interface {
	Enhance(ctx context.Context, alert *Alert) error
}
