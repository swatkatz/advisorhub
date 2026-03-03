package contribution

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/swatkatz/advisorhub/backend/internal/account"
	"github.com/swatkatz/advisorhub/backend/internal/eventbus"
)

// engine implements ContributionEngine.
type engine struct {
	repo     ContributionRepository
	accounts account.AccountRepository
	resp     account.RESPBeneficiaryRepository
	bus      eventbus.EventBus
	now      func() time.Time
}

// NewEngine creates a new ContributionEngine.
func NewEngine(
	repo ContributionRepository,
	accounts account.AccountRepository,
	resp account.RESPBeneficiaryRepository,
	bus eventbus.EventBus,
) ContributionEngine {
	return &engine{
		repo:     repo,
		accounts: accounts,
		resp:     resp,
		bus:      bus,
		now:      func() time.Time { return time.Now() },
	}
}

// getAnnualLimit returns the annual contribution limit for an account type.
func (e *engine) getAnnualLimit(ctx context.Context, clientID string, accountType string, taxYear int) (float64, error) {
	switch accountType {
	case AccountTypeRRSP:
		limit, err := e.repo.GetClientContributionLimit(ctx, clientID, taxYear)
		if err != nil {
			return 0, fmt.Errorf("getting RRSP limit: %w", err)
		}
		if limit == nil {
			return DefaultRRSPLimit, nil
		}
		return limit.RRSPDeductionLimit, nil
	case AccountTypeTFSA:
		return TFSAAnnualLimit, nil
	case AccountTypeFHSA:
		return FHSAAnnualLimit, nil
	default:
		return 0, nil
	}
}

// sumContributionsByType sums contributions grouped by account type.
func sumContributionsByType(contributions []Contribution) map[string]float64 {
	totals := make(map[string]float64)
	for _, c := range contributions {
		totals[c.AccountType] += c.Amount
	}
	return totals
}

func (e *engine) GetRoom(ctx context.Context, clientID string, accountType string, taxYear int) (float64, error) {
	if accountType == AccountTypeRESP || accountType == AccountTypeNonReg {
		return 0, nil
	}

	annualLimit, err := e.getAnnualLimit(ctx, clientID, accountType, taxYear)
	if err != nil {
		return 0, err
	}

	contributions, err := e.repo.GetContributionsByClient(ctx, clientID, taxYear)
	if err != nil {
		return 0, fmt.Errorf("getting contributions: %w", err)
	}

	var contributed float64
	for _, c := range contributions {
		if c.AccountType == accountType {
			contributed += c.Amount
		}
	}

	room := annualLimit - contributed
	if room < 0 {
		room = 0
	}
	return room, nil
}

func (e *engine) AnalyzeClient(ctx context.Context, clientID string, taxYear int) error {
	contributions, err := e.repo.GetContributionsByClient(ctx, clientID, taxYear)
	if err != nil {
		return fmt.Errorf("getting contributions: %w", err)
	}

	accounts, err := e.accounts.GetAccountsByClientID(ctx, clientID)
	if err != nil {
		return fmt.Errorf("getting accounts: %w", err)
	}

	// Build account ID → institution lookup
	acctInstitution := make(map[string]string)
	for _, a := range accounts {
		acctInstitution[a.ID] = a.Institution
	}

	// Check RRSP, TFSA, FHSA for over-contributions
	for _, acctType := range []string{AccountTypeRRSP, AccountTypeTFSA, AccountTypeFHSA} {
		if err := e.checkOverContribution(ctx, clientID, acctType, taxYear, contributions, acctInstitution); err != nil {
			return err
		}
	}

	// FHSA lifetime check
	if err := e.checkFHSALifetime(ctx, clientID, accounts); err != nil {
		return err
	}

	// CESG gap detection for RESP beneficiaries
	if err := e.checkCESGGaps(ctx, clientID, taxYear, contributions, accounts); err != nil {
		return err
	}

	return nil
}

func (e *engine) checkOverContribution(ctx context.Context, clientID string, accountType string, taxYear int, contributions []Contribution, acctInstitution map[string]string) error {
	annualLimit, err := e.getAnnualLimit(ctx, clientID, accountType, taxYear)
	if err != nil {
		return err
	}
	if annualLimit == 0 {
		return nil
	}

	var contributed float64
	institutionAmounts := make(map[string]float64)
	for _, c := range contributions {
		if c.AccountType == accountType {
			contributed += c.Amount
			inst := acctInstitution[c.AccountID]
			if inst == "" {
				inst = "Unknown"
			}
			institutionAmounts[inst] += c.Amount
		}
	}

	if contributed > annualLimit {
		excess := contributed - annualLimit
		penalty := math.Round(excess*PenaltyRate*100) / 100

		institutions := make([]string, 0, len(institutionAmounts))
		for inst := range institutionAmounts {
			institutions = append(institutions, inst)
		}

		payload := map[string]any{
			"client_id":             clientID,
			"account_type":          accountType,
			"reason":                "annual_limit",
			"limit":                 annualLimit,
			"contributed":           contributed,
			"excess":                excess,
			"penalty_per_month":     penalty,
			"institutions_involved": institutions,
		}
		if err := e.publishEvent(ctx, clientID, EventOverContributionDetected, payload); err != nil {
			return err
		}
	} else {
		// Emit ContributionProcessed for non-over-contributed types
		remaining := annualLimit - contributed
		payload := map[string]any{
			"client_id":      clientID,
			"account_type":   accountType,
			"remaining_room": remaining,
		}
		if err := e.publishEvent(ctx, clientID, EventContributionProcessed, payload); err != nil {
			return err
		}
	}

	return nil
}

func (e *engine) checkFHSALifetime(ctx context.Context, clientID string, accounts []account.Account) error {
	for _, acct := range accounts {
		if acct.AccountType != AccountTypeFHSA {
			continue
		}
		lifetimeTotal := acct.FHSALifetimeContributions
		if lifetimeTotal >= FHSALifetimeCap {
			excess := lifetimeTotal - FHSALifetimeCap
			penalty := math.Round(excess*PenaltyRate*100) / 100
			payload := map[string]any{
				"client_id":             clientID,
				"account_type":          AccountTypeFHSA,
				"reason":                "lifetime_cap",
				"limit":                 FHSALifetimeCap,
				"contributed":           lifetimeTotal,
				"excess":                excess,
				"penalty_per_month":     penalty,
				"institutions_involved": []string{acct.Institution},
			}
			if err := e.publishEvent(ctx, clientID, EventOverContributionDetected, payload); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *engine) checkCESGGaps(ctx context.Context, clientID string, taxYear int, contributions []Contribution, accounts []account.Account) error {
	beneficiaries, err := e.resp.GetRESPBeneficiariesByClientID(ctx, clientID)
	if err != nil {
		return fmt.Errorf("getting RESP beneficiaries: %w", err)
	}

	if len(beneficiaries) == 0 {
		return nil
	}

	// Map beneficiary ID → set of account IDs
	beneficiaryAccounts := make(map[string]map[string]bool)
	for _, acct := range accounts {
		if acct.AccountType == AccountTypeRESP && acct.RESPBeneficiaryID != nil {
			if _, ok := beneficiaryAccounts[*acct.RESPBeneficiaryID]; !ok {
				beneficiaryAccounts[*acct.RESPBeneficiaryID] = make(map[string]bool)
			}
			beneficiaryAccounts[*acct.RESPBeneficiaryID][acct.ID] = true
		}
	}

	for _, ben := range beneficiaries {
		// Skip if lifetime cap reached
		if ben.LifetimeContributions >= RESPLifetimeCap {
			continue
		}

		// Sum YTD contributions for this beneficiary's RESP accounts
		acctIDs := beneficiaryAccounts[ben.ID]
		var contributedYTD float64
		for _, c := range contributions {
			if c.AccountType == AccountTypeRESP && acctIDs[c.AccountID] {
				contributedYTD += c.Amount
			}
		}

		if contributedYTD < CESGEligibleMax {
			gap := CESGEligibleMax - contributedYTD
			grantLoss := gap * CESGMatchRate
			payload := map[string]any{
				"client_id":            clientID,
				"beneficiary_id":       ben.ID,
				"contributed_ytd":      contributedYTD,
				"cesg_eligible_max":    CESGEligibleMax,
				"gap_amount":           gap,
				"potential_grant_loss": grantLoss,
			}
			if err := e.publishEvent(ctx, clientID, EventCESGGap, payload); err != nil {
				return err
			}
		}
	}

	return nil
}

func (e *engine) GetContributionSummary(ctx context.Context, clientID string, taxYear int) (*ContributionSummary, error) {
	contributions, err := e.repo.GetContributionsByClient(ctx, clientID, taxYear)
	if err != nil {
		return nil, fmt.Errorf("getting contributions: %w", err)
	}

	accounts, err := e.accounts.GetAccountsByClientID(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("getting accounts: %w", err)
	}

	// Determine which account types the client holds
	heldTypes := make(map[string]bool)
	for _, a := range accounts {
		heldTypes[string(a.AccountType)] = true
	}

	// Sum contributions by type
	contributedByType := sumContributionsByType(contributions)

	now := e.now()
	var acctContribs []AccountContribution

	for _, acctType := range []string{AccountTypeRRSP, AccountTypeTFSA, AccountTypeFHSA, AccountTypeRESP, AccountTypeNonReg} {
		if !heldTypes[acctType] {
			continue
		}

		annualLimit, err := e.getAnnualLimit(ctx, clientID, acctType, taxYear)
		if err != nil {
			return nil, err
		}

		contributed := contributedByType[acctType]
		remaining := annualLimit - contributed
		if remaining < 0 {
			remaining = 0
		}

		isOver := contributed > annualLimit && annualLimit > 0
		var overAmount, penalty float64
		if isOver {
			overAmount = contributed - annualLimit
			penalty = math.Round(overAmount*PenaltyRate*100) / 100
		}

		deadline := Deadline(acctType, taxYear)
		var daysUntil *int
		if deadline != nil {
			days := int(deadline.Sub(now).Hours() / 24)
			daysUntil = &days
		}

		acctContribs = append(acctContribs, AccountContribution{
			AccountType:       acctType,
			AnnualLimit:       annualLimit,
			Contributed:       contributed,
			Remaining:         remaining,
			IsOverContributed: isOver,
			OverAmount:        overAmount,
			PenaltyPerMonth:   penalty,
			Deadline:          deadline,
			DaysUntilDeadline: daysUntil,
		})
	}

	return &ContributionSummary{
		ClientID: clientID,
		TaxYear:  taxYear,
		Accounts: acctContribs,
	}, nil
}

func (e *engine) RecordContribution(ctx context.Context, contribution *Contribution) (*Contribution, error) {
	result, err := e.repo.RecordContribution(ctx, contribution)
	if err != nil {
		return nil, fmt.Errorf("recording contribution: %w", err)
	}

	// After recording, recompute and update lifetime totals for FHSA and RESP
	switch contribution.AccountType {
	case AccountTypeFHSA:
		if err := e.updateFHSALifetime(ctx, contribution.ClientID, contribution.AccountID); err != nil {
			return nil, err
		}
	case AccountTypeRESP:
		if err := e.updateRESPBeneficiaryLifetime(ctx, contribution.ClientID, contribution.AccountID); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (e *engine) updateFHSALifetime(ctx context.Context, clientID string, accountID string) error {
	// Get all contributions for this client (all years) to recompute lifetime
	// For prototype, we sum all FHSA contributions for this specific account
	contributions, err := e.getAllContributionsForAccount(ctx, clientID, accountID)
	if err != nil {
		return err
	}

	var total float64
	for _, c := range contributions {
		total += c.Amount
	}

	return e.accounts.UpdateFHSALifetimeContributions(ctx, accountID, total)
}

func (e *engine) updateRESPBeneficiaryLifetime(ctx context.Context, clientID string, accountID string) error {
	// Look up which beneficiary this account is linked to
	accounts, err := e.accounts.GetAccountsByClientID(ctx, clientID)
	if err != nil {
		return fmt.Errorf("getting accounts: %w", err)
	}

	var beneficiaryID string
	for _, a := range accounts {
		if a.ID == accountID && a.RESPBeneficiaryID != nil {
			beneficiaryID = *a.RESPBeneficiaryID
			break
		}
	}
	if beneficiaryID == "" {
		return nil // no linked beneficiary
	}

	// Sum all RESP contributions across all accounts linked to this beneficiary
	var total float64
	benAccounts := make(map[string]bool)
	for _, a := range accounts {
		if a.AccountType == AccountTypeRESP && a.RESPBeneficiaryID != nil && *a.RESPBeneficiaryID == beneficiaryID {
			benAccounts[a.ID] = true
		}
	}

	contributions, err := e.getAllContributionsForClient(ctx, clientID)
	if err != nil {
		return err
	}
	for _, c := range contributions {
		if c.AccountType == AccountTypeRESP && benAccounts[c.AccountID] {
			total += c.Amount
		}
	}

	return e.resp.UpdateLifetimeContributions(ctx, beneficiaryID, total)
}

// getAllContributionsForAccount returns all contributions for a specific account across all years.
// For the prototype, we query a few likely years. In production, this would be a dedicated query.
func (e *engine) getAllContributionsForAccount(ctx context.Context, clientID string, accountID string) ([]Contribution, error) {
	// Query a range of years to cover all contributions
	var all []Contribution
	currentYear := e.now().Year()
	for year := currentYear - 10; year <= currentYear; year++ {
		contribs, err := e.repo.GetContributionsByClient(ctx, clientID, year)
		if err != nil {
			return nil, fmt.Errorf("getting contributions for year %d: %w", year, err)
		}
		for _, c := range contribs {
			if c.AccountID == accountID {
				all = append(all, c)
			}
		}
	}
	return all, nil
}

// getAllContributionsForClient returns all contributions for a client across all years.
func (e *engine) getAllContributionsForClient(ctx context.Context, clientID string) ([]Contribution, error) {
	var all []Contribution
	currentYear := e.now().Year()
	for year := currentYear - 10; year <= currentYear; year++ {
		contribs, err := e.repo.GetContributionsByClient(ctx, clientID, year)
		if err != nil {
			return nil, fmt.Errorf("getting contributions for year %d: %w", year, err)
		}
		all = append(all, contribs...)
	}
	return all, nil
}

func (e *engine) publishEvent(ctx context.Context, clientID string, eventType string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling event payload: %w", err)
	}
	return e.bus.Publish(ctx, eventbus.EventEnvelope{
		ID:         fmt.Sprintf("evt_%s_%s_%d", eventType, clientID, e.now().UnixNano()),
		Type:       eventType,
		EntityID:   clientID,
		EntityType: eventbus.EntityTypeClient,
		Payload:    data,
		Source:     eventbus.SourceReactive,
		Timestamp:  e.now(),
	})
}
