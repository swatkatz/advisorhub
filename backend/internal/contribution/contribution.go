// Package contribution implements contribution room calculation, over-contribution
// detection, CESG gap detection, and FHSA lifetime tracking.
package contribution

import (
	"context"
	"time"
)

// Contribution is a record of money contributed to an account.
type Contribution struct {
	ID          string
	ClientID    string
	AccountID   string
	AccountType string // denormalized from Account for aggregation
	Amount      float64
	Date        time.Time
	TaxYear     int
}

// ClientContributionLimit holds the per-client RRSP deduction limit for a tax year.
type ClientContributionLimit struct {
	ID                 string
	ClientID           string
	TaxYear            int
	RRSPDeductionLimit float64
}

// ContributionSummary is computed per-client per-year, not persisted.
type ContributionSummary struct {
	ClientID string
	TaxYear  int
	Accounts []AccountContribution
}

// AccountContribution is a computed summary for one account type.
type AccountContribution struct {
	AccountType       string
	AnnualLimit       float64
	Contributed       float64
	Remaining         float64
	IsOverContributed bool
	OverAmount        float64
	PenaltyPerMonth   float64
	Deadline          *time.Time
	DaysUntilDeadline *int
}

// ContributionRepository is the data access interface for contributions.
type ContributionRepository interface {
	GetContributionsByClient(ctx context.Context, clientID string, taxYear int) ([]Contribution, error)
	RecordContribution(ctx context.Context, contribution *Contribution) (*Contribution, error)
	GetClientContributionLimit(ctx context.Context, clientID string, taxYear int) (*ClientContributionLimit, error)
	SaveClientContributionLimit(ctx context.Context, limit *ClientContributionLimit) (*ClientContributionLimit, error)
}

// ContributionEngine is the public interface for contribution analysis.
type ContributionEngine interface {
	AnalyzeClient(ctx context.Context, clientID string, taxYear int) error
	GetContributionSummary(ctx context.Context, clientID string, taxYear int) (*ContributionSummary, error)
	GetRoom(ctx context.Context, clientID string, accountType string, taxYear int) (float64, error)
	RecordContribution(ctx context.Context, contribution *Contribution) (*Contribution, error)
}
