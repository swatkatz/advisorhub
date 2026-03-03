package alert

import (
	"fmt"
	"time"

	"github.com/swatkatz/advisorhub/backend/internal/eventbus"
)

// Event type constants for incoming domain events.
const (
	EventOverContributionDetected = "OverContributionDetected"
	EventCESGGap                  = "CESGGap"
	EventTransferStuck            = "TransferStuck"
	EventDeadlineApproaching      = "DeadlineApproaching"
	EventAgeMilestone             = "AgeMilestone"
	EventEngagementStale          = "EngagementStale"
	EventCashUninvested           = "CashUninvested"
	EventPortfolioDrift           = "PortfolioDrift"
	EventTaxLossOpportunity       = "TaxLossOpportunity"
	EventTransferCompleted        = "TransferCompleted"
	EventContributionProcessed    = "ContributionProcessed"
	EventDividendReceived         = "DividendReceived"
)

// AlertCategoryRule maps an event type to alert properties.
type AlertCategoryRule struct {
	EventType          string
	Category           string
	Severity           AlertSeverity
	NeedsDraft         bool
	AutoSnoozeDuration time.Duration
	BuildConditionKey  func(payload map[string]any, env eventbus.EventEnvelope) string
	ExtractClientID    func(payload map[string]any, env eventbus.EventEnvelope) string
}

// CategoryRules maps event types to their alert category rules.
var CategoryRules = map[string]AlertCategoryRule{
	EventOverContributionDetected: {
		EventType:          EventOverContributionDetected,
		Category:           "over_contribution",
		Severity:           SeverityCritical,
		NeedsDraft:         true,
		AutoSnoozeDuration: 7 * 24 * time.Hour,
		BuildConditionKey: func(p map[string]any, env eventbus.EventEnvelope) string {
			return fmt.Sprintf("overcontrib:%s:%s", getString(p, "client_id"), getString(p, "account_type"))
		},
		ExtractClientID: clientIDFromPayload,
	},
	EventTransferStuck: {
		EventType:          EventTransferStuck,
		Category:           "transfer_stuck",
		Severity:           SeverityCritical,
		NeedsDraft:         true,
		AutoSnoozeDuration: 5 * 24 * time.Hour,
		BuildConditionKey: func(p map[string]any, env eventbus.EventEnvelope) string {
			return fmt.Sprintf("transfer_stuck:%s", getString(p, "transfer_id"))
		},
		ExtractClientID: clientIDFromPayload,
	},
	EventDeadlineApproaching: {
		EventType:          EventDeadlineApproaching,
		Category:           "deadline_approaching",
		Severity:           SeverityUrgent,
		NeedsDraft:         true,
		AutoSnoozeDuration: 3 * 24 * time.Hour,
		BuildConditionKey: func(p map[string]any, env eventbus.EventEnvelope) string {
			return fmt.Sprintf("deadline_approaching:%s:%s:%s",
				getString(p, "client_id"), getString(p, "account_type"), getString(p, "tax_year"))
		},
		ExtractClientID: clientIDFromPayload,
	},
	EventAgeMilestone: {
		EventType:          EventAgeMilestone,
		Category:           "age_milestone",
		Severity:           SeverityUrgent,
		NeedsDraft:         true,
		AutoSnoozeDuration: 14 * 24 * time.Hour,
		BuildConditionKey: func(p map[string]any, env eventbus.EventEnvelope) string {
			return fmt.Sprintf("age_milestone:%s:%s", env.EntityID, getString(p, "target_age"))
		},
		ExtractClientID: clientIDFromPayload,
	},
	EventCESGGap: {
		EventType:          EventCESGGap,
		Category:           "cesg_gap",
		Severity:           SeverityUrgent,
		NeedsDraft:         true,
		AutoSnoozeDuration: 14 * 24 * time.Hour,
		BuildConditionKey: func(p map[string]any, env eventbus.EventEnvelope) string {
			return fmt.Sprintf("cesg_gap:%s:%s:%s",
				getString(p, "client_id"), getString(p, "beneficiary_id"), getString(p, "tax_year"))
		},
		ExtractClientID: clientIDFromPayload,
	},
	EventEngagementStale: {
		EventType:          EventEngagementStale,
		Category:           "engagement_stale",
		Severity:           SeverityAdvisory,
		NeedsDraft:         true,
		AutoSnoozeDuration: 14 * 24 * time.Hour,
		BuildConditionKey: func(p map[string]any, env eventbus.EventEnvelope) string {
			return fmt.Sprintf("engagement_stale:%s", getString(p, "client_id"))
		},
		ExtractClientID: clientIDFromPayload,
	},
	EventCashUninvested: {
		EventType:          EventCashUninvested,
		Category:           "cash_uninvested",
		Severity:           SeverityAdvisory,
		NeedsDraft:         true,
		AutoSnoozeDuration: 14 * 24 * time.Hour,
		BuildConditionKey: func(p map[string]any, env eventbus.EventEnvelope) string {
			return fmt.Sprintf("cash_uninvested:%s", getString(p, "account_id"))
		},
		ExtractClientID: clientIDFromPayload,
	},
	EventPortfolioDrift: {
		EventType:          EventPortfolioDrift,
		Category:           "portfolio_drift",
		Severity:           SeverityAdvisory,
		NeedsDraft:         true,
		AutoSnoozeDuration: 14 * 24 * time.Hour,
		BuildConditionKey: func(p map[string]any, env eventbus.EventEnvelope) string {
			return fmt.Sprintf("portfolio_drift:%s", getString(p, "client_id"))
		},
		ExtractClientID: clientIDFromPayload,
	},
	EventTaxLossOpportunity: {
		EventType:          EventTaxLossOpportunity,
		Category:           "tax_loss_opportunity",
		Severity:           SeverityAdvisory,
		NeedsDraft:         true,
		AutoSnoozeDuration: 14 * 24 * time.Hour,
		BuildConditionKey: func(p map[string]any, env eventbus.EventEnvelope) string {
			return fmt.Sprintf("tax_loss:%s:%s", getString(p, "client_id"), getString(p, "holding"))
		},
		ExtractClientID: clientIDFromPayload,
	},
	EventTransferCompleted: {
		EventType:  EventTransferCompleted,
		Category:   "transfer_completed",
		Severity:   SeverityInfo,
		NeedsDraft: false,
		BuildConditionKey: func(p map[string]any, env eventbus.EventEnvelope) string {
			return fmt.Sprintf("transfer_completed:%s", getString(p, "transfer_id"))
		},
		ExtractClientID: clientIDFromPayload,
	},
	EventContributionProcessed: {
		EventType:  EventContributionProcessed,
		Category:   "contribution_processed",
		Severity:   SeverityInfo,
		NeedsDraft: false,
		BuildConditionKey: func(p map[string]any, env eventbus.EventEnvelope) string {
			return fmt.Sprintf("contribution_processed:%s:%s",
				getString(p, "client_id"), getString(p, "account_type"))
		},
		ExtractClientID: clientIDFromPayload,
	},
	EventDividendReceived: {
		EventType:  EventDividendReceived,
		Category:   "dividend_received",
		Severity:   SeverityInfo,
		NeedsDraft: false,
		BuildConditionKey: func(p map[string]any, env eventbus.EventEnvelope) string {
			return fmt.Sprintf("dividend_received:%s", getString(p, "client_id"))
		},
		ExtractClientID: clientIDFromPayload,
	},
}

// NeedsDraft returns whether a category's alerts should include a draft message.
func NeedsDraft(category string) bool {
	for _, rule := range CategoryRules {
		if rule.Category == category {
			return rule.NeedsDraft
		}
	}
	return false
}

// GetAutoSnoozeDuration returns the auto-snooze duration for a category.
func GetAutoSnoozeDuration(category string) time.Duration {
	for _, rule := range CategoryRules {
		if rule.Category == category {
			return rule.AutoSnoozeDuration
		}
	}
	return 14 * 24 * time.Hour
}

// getString extracts a string value from a parsed JSON map.
// Handles string values directly and converts float64 (JSON numbers) to integer strings.
func getString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// clientIDFromPayload extracts client_id from the event payload,
// falling back to EntityID on the envelope when EntityType is Client.
func clientIDFromPayload(p map[string]any, env eventbus.EventEnvelope) string {
	if id := getString(p, "client_id"); id != "" {
		return id
	}
	if env.EntityType == eventbus.EntityTypeClient {
		return env.EntityID
	}
	return ""
}
