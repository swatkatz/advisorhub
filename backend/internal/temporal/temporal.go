package temporal

import (
	"context"
	"time"

	"github.com/swatkatz/advisorhub/backend/internal/eventbus"
)

// CheckType identifies which check function a TemporalRule dispatches to.
type CheckType string

const (
	CheckTypeAgeApproaching   CheckType = "AGE_APPROACHING"
	CheckTypeDeadlineWithRoom CheckType = "DEADLINE_WITH_ROOM"
	CheckTypeDaysSince        CheckType = "DAYS_SINCE"
	CheckTypeBalanceIdle      CheckType = "BALANCE_IDLE"
)

// Event types emitted by the temporal scanner.
const (
	EventDeadlineApproaching = "DeadlineApproaching"
	EventAgeMilestone        = "AgeMilestone"
	EventEngagementStale     = "EngagementStale"
	EventCashUninvested      = "CashUninvested"
)

// Account type constants (untyped strings for map[string]any compatibility in rule Params).
const (
	AccountTypeRRSP   = "RRSP"
	AccountTypeTFSA   = "TFSA"
	AccountTypeFHSA   = "FHSA"
	AccountTypeRESP   = "RESP"
	AccountTypeNonReg = "NON_REG"
)

// TemporalRule defines a rule the scanner evaluates during a sweep.
// Rules are hardcoded Go constants, not database rows.
type TemporalRule struct {
	Name       string
	CheckType  CheckType
	EntityType eventbus.EntityType
	Params     map[string]any
	EventType  string
}

// Rules is the hardcoded set of temporal rules evaluated during each sweep.
var Rules = []TemporalRule{
	{
		Name:       "RRIF_CONVERSION",
		CheckType:  CheckTypeAgeApproaching,
		EntityType: eventbus.EntityTypeClient,
		Params:     map[string]any{"age": 71, "within_days": 365},
		EventType:  EventAgeMilestone,
	},
	{
		Name:       "RESP_LAST_CESG",
		CheckType:  CheckTypeAgeApproaching,
		EntityType: eventbus.EntityTypeRESPBeneficiary,
		Params:     map[string]any{"age": 17, "within_days": 365},
		EventType:  EventAgeMilestone,
	},
	{
		Name:       "RRSP_DEADLINE",
		CheckType:  CheckTypeDeadlineWithRoom,
		EntityType: eventbus.EntityTypeAccount,
		Params:     map[string]any{"account_type": AccountTypeRRSP, "within_days": 30},
		EventType:  EventDeadlineApproaching,
	},
	{
		Name:       "TFSA_DEADLINE",
		CheckType:  CheckTypeDeadlineWithRoom,
		EntityType: eventbus.EntityTypeAccount,
		Params:     map[string]any{"account_type": AccountTypeTFSA, "within_days": 30},
		EventType:  EventDeadlineApproaching,
	},
	{
		Name:       "FHSA_DEADLINE",
		CheckType:  CheckTypeDeadlineWithRoom,
		EntityType: eventbus.EntityTypeAccount,
		Params:     map[string]any{"account_type": AccountTypeFHSA, "within_days": 30},
		EventType:  EventDeadlineApproaching,
	},
	{
		Name:       "ENGAGEMENT_STALE",
		CheckType:  CheckTypeDaysSince,
		EntityType: eventbus.EntityTypeClient,
		Params:     map[string]any{"field": "last_meeting_date", "threshold": 180},
		EventType:  EventEngagementStale,
	},
	{
		Name:       "CASH_UNINVESTED",
		CheckType:  CheckTypeBalanceIdle,
		EntityType: eventbus.EntityTypeAccount,
		Params:     map[string]any{"min_balance": 5000.0, "idle_days": 30},
		EventType:  EventCashUninvested,
	},
}

// ScannerResult contains the outcome of a sweep.
type ScannerResult struct {
	EventsEmitted   int
	RulesEvaluated  int
	EntitiesChecked int
	Duration        time.Duration
}

// TemporalScanner evaluates temporal rules and emits events for matches.
type TemporalScanner interface {
	RunSweep(ctx context.Context, advisorID string, referenceDate time.Time) (*ScannerResult, error)
}

// deadline computes the contribution deadline for an account type in a given tax year.
// RRSP: 60 days after year-end (March 1 non-leap, March 2 leap of taxYear+1).
// TFSA/FHSA: Dec 31 of taxYear.
func deadline(accountType string, taxYear int) *time.Time {
	switch accountType {
	case AccountTypeRRSP:
		d := time.Date(taxYear+1, time.January, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, 59)
		return &d
	case AccountTypeTFSA, AccountTypeFHSA:
		d := time.Date(taxYear, time.December, 31, 0, 0, 0, 0, time.UTC)
		return &d
	default:
		return nil
	}
}
