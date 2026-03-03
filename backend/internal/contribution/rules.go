package contribution

import "time"

// Account type constants (local to this package; cross-context uses strings).
const (
	AccountTypeRRSP   = "RRSP"
	AccountTypeTFSA   = "TFSA"
	AccountTypeFHSA   = "FHSA"
	AccountTypeRESP   = "RESP"
	AccountTypeNonReg = "NON_REG"
)

// Annual limits.
const (
	DefaultRRSPLimit = 32490.0
	TFSAAnnualLimit  = 7000.0
	FHSAAnnualLimit  = 8000.0
	CESGEligibleMax  = 2500.0
	CESGMatchRate    = 0.20
	PenaltyRate      = 0.01 // 1% per month on excess
)

// Lifetime caps (owned by account context's AccountType, but we need constants for computation).
const (
	FHSALifetimeCap = 40000.0
	RESPLifetimeCap = 50000.0
)

// Event types emitted by the contribution engine.
const (
	EventOverContributionDetected = "OverContributionDetected"
	EventCESGGap                  = "CESGGap"
	EventContributionProcessed    = "ContributionProcessed"
)


// Deadline computes the contribution deadline for an account type in a given tax year.
func Deadline(accountType string, taxYear int) *time.Time {
	switch accountType {
	case AccountTypeRRSP:
		// 60 days after year-end = March 1 (or March 2 in leap years)
		d := time.Date(taxYear+1, time.January, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, 59)
		return &d
	case AccountTypeTFSA, AccountTypeFHSA:
		d := time.Date(taxYear, time.December, 31, 0, 0, 0, 0, time.UTC)
		return &d
	default:
		// RESP and NON_REG have no deadline
		return nil
	}
}
