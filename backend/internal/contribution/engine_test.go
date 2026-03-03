package contribution

import (
	"context"
	"encoding/json"
	"sort"
	"testing"
	"time"
)

func setupEngine() (*engine, *memoryContributionRepo, *memoryAccountRepo, *memoryRESPBeneficiaryRepo, *memoryEventBus) {
	contribRepo := newMemoryContributionRepo()
	accountRepo := newMemoryAccountRepo()
	respRepo := newMemoryRESPBeneficiaryRepo()
	bus := newMemoryEventBus()
	e := &engine{
		repo:     contribRepo,
		accounts: accountRepo,
		resp:     respRepo,
		bus:      bus,
		now:      func() time.Time { return time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC) },
	}
	return e, contribRepo, accountRepo, respRepo, bus
}

// Test Anchor 1: RRSP room with contributions across two institutions.
func TestGetRoom_RRSP_TwoInstitutions(t *testing.T) {
	e, contribRepo, accountRepo, _, _ := setupEngine()
	ctx := context.Background()

	accountRepo.addAccount(Account{ID: "a1", ClientID: "c1", AccountType: AccountTypeRRSP, Institution: "Wealthsimple"})
	accountRepo.addAccount(Account{ID: "a2", ClientID: "c1", AccountType: AccountTypeRRSP, Institution: "RBC", IsExternal: true})

	contribRepo.addLimit(ClientContributionLimit{ID: "ccl1", ClientID: "c1", TaxYear: 2026, RRSPDeductionLimit: 32490})
	contribRepo.addContribution(Contribution{ID: "ct1", ClientID: "c1", AccountID: "a1", AccountType: AccountTypeRRSP, Amount: 12000, Date: time.Now(), TaxYear: 2026})
	contribRepo.addContribution(Contribution{ID: "ct2", ClientID: "c1", AccountID: "a2", AccountType: AccountTypeRRSP, Amount: 8000, Date: time.Now(), TaxYear: 2026})

	room, err := e.GetRoom(ctx, "c1", AccountTypeRRSP, 2026)
	if err != nil {
		t.Fatalf("GetRoom: %v", err)
	}
	if room != 12490 {
		t.Errorf("expected room 12490, got %v", room)
	}
}

// Test Anchor 3: TFSA at exactly the limit returns 0 room.
func TestGetRoom_TFSA_ExactlyAtLimit(t *testing.T) {
	e, contribRepo, _, _, _ := setupEngine()
	ctx := context.Background()

	contribRepo.addContribution(Contribution{ID: "ct1", ClientID: "c1", AccountID: "a1", AccountType: AccountTypeTFSA, Amount: 7000, Date: time.Now(), TaxYear: 2026})

	room, err := e.GetRoom(ctx, "c1", AccountTypeTFSA, 2026)
	if err != nil {
		t.Fatalf("GetRoom: %v", err)
	}
	if room != 0 {
		t.Errorf("expected room 0, got %v", room)
	}
}

// Test Anchor 15: No ClientContributionLimit for the year uses default $32,490 cap.
func TestGetRoom_RRSP_NoLimitUsesDefault(t *testing.T) {
	e, contribRepo, _, _, _ := setupEngine()
	ctx := context.Background()

	contribRepo.addContribution(Contribution{ID: "ct1", ClientID: "c1", AccountID: "a1", AccountType: AccountTypeRRSP, Amount: 10000, Date: time.Now(), TaxYear: 2026})

	room, err := e.GetRoom(ctx, "c1", AccountTypeRRSP, 2026)
	if err != nil {
		t.Fatalf("GetRoom: %v", err)
	}
	expected := DefaultRRSPLimit - 10000
	if room != expected {
		t.Errorf("expected room %v, got %v", expected, room)
	}
}

// RESP and NON_REG return 0 room.
func TestGetRoom_RESP_ReturnsZero(t *testing.T) {
	e, _, _, _, _ := setupEngine()
	ctx := context.Background()

	room, err := e.GetRoom(ctx, "c1", AccountTypeRESP, 2026)
	if err != nil {
		t.Fatalf("GetRoom: %v", err)
	}
	if room != 0 {
		t.Errorf("expected room 0, got %v", room)
	}
}

func TestGetRoom_NON_REG_ReturnsZero(t *testing.T) {
	e, _, _, _, _ := setupEngine()
	ctx := context.Background()

	room, err := e.GetRoom(ctx, "c1", AccountTypeNonReg, 2026)
	if err != nil {
		t.Fatalf("GetRoom: %v", err)
	}
	if room != 0 {
		t.Errorf("expected room 0, got %v", room)
	}
}

// parsePayload is a test helper to unmarshal event payload JSON.
func parsePayload(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return m
}

// Test Anchor 2: RRSP over-contribution with two institutions.
func TestAnalyzeClient_RRSP_OverContribution(t *testing.T) {
	e, contribRepo, accountRepo, _, bus := setupEngine()
	ctx := context.Background()

	accountRepo.addAccount(Account{ID: "a1", ClientID: "c1", AccountType: AccountTypeRRSP, Institution: "Wealthsimple"})
	accountRepo.addAccount(Account{ID: "a2", ClientID: "c1", AccountType: AccountTypeRRSP, Institution: "RBC", IsExternal: true})

	contribRepo.addLimit(ClientContributionLimit{ID: "ccl1", ClientID: "c1", TaxYear: 2026, RRSPDeductionLimit: 31560})
	contribRepo.addContribution(Contribution{ID: "ct1", ClientID: "c1", AccountID: "a1", AccountType: AccountTypeRRSP, Amount: 18860, Date: time.Now(), TaxYear: 2026})
	contribRepo.addContribution(Contribution{ID: "ct2", ClientID: "c1", AccountID: "a2", AccountType: AccountTypeRRSP, Amount: 15000, Date: time.Now(), TaxYear: 2026})

	err := e.AnalyzeClient(ctx, "c1", 2026)
	if err != nil {
		t.Fatalf("AnalyzeClient: %v", err)
	}

	events := bus.eventsByType(EventOverContributionDetected)
	if len(events) == 0 {
		t.Fatal("expected OverContributionDetected event")
	}

	p := parsePayload(t, events[0].Payload)
	if p["account_type"] != "RRSP" {
		t.Errorf("expected account_type RRSP, got %v", p["account_type"])
	}
	if p["excess"].(float64) != 2300 {
		t.Errorf("expected excess 2300, got %v", p["excess"])
	}
	if p["penalty_per_month"].(float64) != 23 {
		t.Errorf("expected penalty 23, got %v", p["penalty_per_month"])
	}
	if p["reason"] != "annual_limit" {
		t.Errorf("expected reason annual_limit, got %v", p["reason"])
	}

	institutions := p["institutions_involved"].([]any)
	instNames := make([]string, len(institutions))
	for i, v := range institutions {
		instNames[i] = v.(string)
	}
	sort.Strings(instNames)
	if len(instNames) != 2 || instNames[0] != "RBC" || instNames[1] != "Wealthsimple" {
		t.Errorf("expected [RBC, Wealthsimple], got %v", instNames)
	}
}

// Test Anchor 4: TFSA over-contribution.
func TestAnalyzeClient_TFSA_OverContribution(t *testing.T) {
	e, contribRepo, accountRepo, _, bus := setupEngine()
	ctx := context.Background()

	accountRepo.addAccount(Account{ID: "a1", ClientID: "c1", AccountType: AccountTypeTFSA, Institution: "Wealthsimple"})
	contribRepo.addContribution(Contribution{ID: "ct1", ClientID: "c1", AccountID: "a1", AccountType: AccountTypeTFSA, Amount: 7500, Date: time.Now(), TaxYear: 2026})

	err := e.AnalyzeClient(ctx, "c1", 2026)
	if err != nil {
		t.Fatalf("AnalyzeClient: %v", err)
	}

	events := bus.eventsByType(EventOverContributionDetected)
	var tfsaEvent *EventEnvelope
	for i, ev := range events {
		p := parsePayload(t, ev.Payload)
		if p["account_type"] == "TFSA" {
			tfsaEvent = &events[i]
			break
		}
	}
	if tfsaEvent == nil {
		t.Fatal("expected TFSA OverContributionDetected event")
	}

	p := parsePayload(t, tfsaEvent.Payload)
	if p["excess"].(float64) != 500 {
		t.Errorf("expected excess 500, got %v", p["excess"])
	}
	if p["reason"] != "annual_limit" {
		t.Errorf("expected reason annual_limit, got %v", p["reason"])
	}
}

// Test Anchor 5: FHSA lifetime cap exceeded.
func TestAnalyzeClient_FHSA_LifetimeCap(t *testing.T) {
	e, contribRepo, accountRepo, _, bus := setupEngine()
	ctx := context.Background()

	accountRepo.addAccount(Account{
		ID: "a1", ClientID: "c1", AccountType: AccountTypeFHSA,
		Institution: "Wealthsimple", FHSALifetimeContributions: 44000, // $38K prior + $6K this year
	})
	// $6,000 this year — within annual $8K limit, but lifetime is $44K which exceeds $40K cap
	contribRepo.addContribution(Contribution{ID: "ct1", ClientID: "c1", AccountID: "a1", AccountType: AccountTypeFHSA, Amount: 6000, Date: time.Now(), TaxYear: 2026})

	err := e.AnalyzeClient(ctx, "c1", 2026)
	if err != nil {
		t.Fatalf("AnalyzeClient: %v", err)
	}

	events := bus.eventsByType(EventOverContributionDetected)
	// Should have a lifetime_cap event (FHSA annual is fine at $6K < $8K, but lifetime $44K > $40K)
	var lifetimeEvent *EventEnvelope
	for i, ev := range events {
		p := parsePayload(t, ev.Payload)
		if p["reason"] == "lifetime_cap" {
			lifetimeEvent = &events[i]
			break
		}
	}
	if lifetimeEvent == nil {
		t.Fatal("expected FHSA lifetime_cap OverContributionDetected event")
	}

	p := parsePayload(t, lifetimeEvent.Payload)
	if p["excess"].(float64) != 4000 {
		t.Errorf("expected excess 4000, got %v", p["excess"])
	}
}

// Test Anchor 6: FHSA annual limit hit (not lifetime).
func TestAnalyzeClient_FHSA_AnnualLimit(t *testing.T) {
	e, contribRepo, accountRepo, _, bus := setupEngine()
	ctx := context.Background()

	accountRepo.addAccount(Account{
		ID: "a1", ClientID: "c1", AccountType: AccountTypeFHSA,
		Institution: "Wealthsimple", FHSALifetimeContributions: 25000,
	})
	contribRepo.addContribution(Contribution{ID: "ct1", ClientID: "c1", AccountID: "a1", AccountType: AccountTypeFHSA, Amount: 9000, Date: time.Now(), TaxYear: 2026})

	err := e.AnalyzeClient(ctx, "c1", 2026)
	if err != nil {
		t.Fatalf("AnalyzeClient: %v", err)
	}

	events := bus.eventsByType(EventOverContributionDetected)
	var annualEvent *EventEnvelope
	for i, ev := range events {
		p := parsePayload(t, ev.Payload)
		if p["reason"] == "annual_limit" && p["account_type"] == "FHSA" {
			annualEvent = &events[i]
			break
		}
	}
	if annualEvent == nil {
		t.Fatal("expected FHSA annual_limit OverContributionDetected event")
	}

	p := parsePayload(t, annualEvent.Payload)
	if p["excess"].(float64) != 1000 {
		t.Errorf("expected excess 1000, got %v", p["excess"])
	}
}

// Test Anchor 10: Over-contribution with 3 institutions.
func TestAnalyzeClient_RRSP_ThreeInstitutions(t *testing.T) {
	e, contribRepo, accountRepo, _, bus := setupEngine()
	ctx := context.Background()

	accountRepo.addAccount(Account{ID: "a1", ClientID: "c1", AccountType: AccountTypeRRSP, Institution: "Wealthsimple"})
	accountRepo.addAccount(Account{ID: "a2", ClientID: "c1", AccountType: AccountTypeRRSP, Institution: "RBC", IsExternal: true})
	accountRepo.addAccount(Account{ID: "a3", ClientID: "c1", AccountType: AccountTypeRRSP, Institution: "TD", IsExternal: true})

	contribRepo.addLimit(ClientContributionLimit{ID: "ccl1", ClientID: "c1", TaxYear: 2026, RRSPDeductionLimit: 30000})
	contribRepo.addContribution(Contribution{ID: "ct1", ClientID: "c1", AccountID: "a1", AccountType: AccountTypeRRSP, Amount: 15000, Date: time.Now(), TaxYear: 2026})
	contribRepo.addContribution(Contribution{ID: "ct2", ClientID: "c1", AccountID: "a2", AccountType: AccountTypeRRSP, Amount: 10000, Date: time.Now(), TaxYear: 2026})
	contribRepo.addContribution(Contribution{ID: "ct3", ClientID: "c1", AccountID: "a3", AccountType: AccountTypeRRSP, Amount: 8000, Date: time.Now(), TaxYear: 2026})

	err := e.AnalyzeClient(ctx, "c1", 2026)
	if err != nil {
		t.Fatalf("AnalyzeClient: %v", err)
	}

	events := bus.eventsByType(EventOverContributionDetected)
	if len(events) == 0 {
		t.Fatal("expected OverContributionDetected event")
	}

	p := parsePayload(t, events[0].Payload)
	institutions := p["institutions_involved"].([]any)
	instNames := make([]string, len(institutions))
	for i, v := range institutions {
		instNames[i] = v.(string)
	}
	sort.Strings(instNames)
	if len(instNames) != 3 {
		t.Errorf("expected 3 institutions, got %v", instNames)
	}
	expected := []string{"RBC", "TD", "Wealthsimple"}
	for i, name := range expected {
		if instNames[i] != name {
			t.Errorf("expected institution %s at index %d, got %s", name, i, instNames[i])
		}
	}

	// Verify excess = 33000 - 30000 = 3000
	if p["excess"].(float64) != 3000 {
		t.Errorf("expected excess 3000, got %v", p["excess"])
	}
}

// Test Anchor 7: CESG gap emitted for beneficiary with contributions < $2500.
func TestAnalyzeClient_CESGGap(t *testing.T) {
	e, contribRepo, accountRepo, respRepo, bus := setupEngine()
	ctx := context.Background()

	benID := "resp_ben_1"
	accountRepo.addAccount(Account{
		ID: "a1", ClientID: "c1", AccountType: AccountTypeRESP,
		Institution: "Wealthsimple", RESPBeneficiaryID: &benID,
	})
	respRepo.addBeneficiary(RESPBeneficiary{
		ID: benID, ClientID: "c1", Name: "Test Child",
		LifetimeContributions: 38200,
	})
	contribRepo.addContribution(Contribution{
		ID: "ct1", ClientID: "c1", AccountID: "a1",
		AccountType: AccountTypeRESP, Amount: 1800, Date: time.Now(), TaxYear: 2026,
	})

	err := e.AnalyzeClient(ctx, "c1", 2026)
	if err != nil {
		t.Fatalf("AnalyzeClient: %v", err)
	}

	events := bus.eventsByType(EventCESGGap)
	if len(events) == 0 {
		t.Fatal("expected CESGGap event")
	}

	p := parsePayload(t, events[0].Payload)
	if p["gap_amount"].(float64) != 700 {
		t.Errorf("expected gap_amount 700, got %v", p["gap_amount"])
	}
	if p["potential_grant_loss"].(float64) != 140 {
		t.Errorf("expected potential_grant_loss 140, got %v", p["potential_grant_loss"])
	}
	if p["beneficiary_id"] != benID {
		t.Errorf("expected beneficiary_id %s, got %v", benID, p["beneficiary_id"])
	}
}

// Test Anchor 8: No CESG gap when contributions reach $2500.
func TestAnalyzeClient_NoCESGGap_FullMatch(t *testing.T) {
	e, contribRepo, accountRepo, respRepo, bus := setupEngine()
	ctx := context.Background()

	benID := "resp_ben_1"
	accountRepo.addAccount(Account{
		ID: "a1", ClientID: "c1", AccountType: AccountTypeRESP,
		Institution: "Wealthsimple", RESPBeneficiaryID: &benID,
	})
	respRepo.addBeneficiary(RESPBeneficiary{
		ID: benID, ClientID: "c1", Name: "Test Child",
		LifetimeContributions: 30000,
	})
	contribRepo.addContribution(Contribution{
		ID: "ct1", ClientID: "c1", AccountID: "a1",
		AccountType: AccountTypeRESP, Amount: 2500, Date: time.Now(), TaxYear: 2026,
	})

	err := e.AnalyzeClient(ctx, "c1", 2026)
	if err != nil {
		t.Fatalf("AnalyzeClient: %v", err)
	}

	events := bus.eventsByType(EventCESGGap)
	if len(events) != 0 {
		t.Errorf("expected no CESGGap events, got %d", len(events))
	}
}

// Test Anchor 9: No CESG gap when beneficiary lifetime >= $50K.
func TestAnalyzeClient_NoCESGGap_LifetimeCapReached(t *testing.T) {
	e, contribRepo, accountRepo, respRepo, bus := setupEngine()
	ctx := context.Background()

	benID := "resp_ben_1"
	accountRepo.addAccount(Account{
		ID: "a1", ClientID: "c1", AccountType: AccountTypeRESP,
		Institution: "Wealthsimple", RESPBeneficiaryID: &benID,
	})
	respRepo.addBeneficiary(RESPBeneficiary{
		ID: benID, ClientID: "c1", Name: "Test Child",
		LifetimeContributions: 50000, // at lifetime cap
	})
	contribRepo.addContribution(Contribution{
		ID: "ct1", ClientID: "c1", AccountID: "a1",
		AccountType: AccountTypeRESP, Amount: 1000, Date: time.Now(), TaxYear: 2026,
	})

	err := e.AnalyzeClient(ctx, "c1", 2026)
	if err != nil {
		t.Fatalf("AnalyzeClient: %v", err)
	}

	events := bus.eventsByType(EventCESGGap)
	if len(events) != 0 {
		t.Errorf("expected no CESGGap events (lifetime cap reached), got %d", len(events))
	}
}

// Test Anchor 11: GetContributionSummary returns entries for each account type held.
func TestGetContributionSummary_MultipleTypes(t *testing.T) {
	e, contribRepo, accountRepo, _, _ := setupEngine()
	ctx := context.Background()

	accountRepo.addAccount(Account{ID: "a1", ClientID: "c1", AccountType: AccountTypeRRSP, Institution: "Wealthsimple"})
	accountRepo.addAccount(Account{ID: "a2", ClientID: "c1", AccountType: AccountTypeTFSA, Institution: "Wealthsimple"})
	accountRepo.addAccount(Account{ID: "a3", ClientID: "c1", AccountType: AccountTypeFHSA, Institution: "Wealthsimple"})

	contribRepo.addLimit(ClientContributionLimit{ID: "ccl1", ClientID: "c1", TaxYear: 2026, RRSPDeductionLimit: 32490})
	contribRepo.addContribution(Contribution{ID: "ct1", ClientID: "c1", AccountID: "a1", AccountType: AccountTypeRRSP, Amount: 10000, Date: time.Now(), TaxYear: 2026})
	contribRepo.addContribution(Contribution{ID: "ct2", ClientID: "c1", AccountID: "a2", AccountType: AccountTypeTFSA, Amount: 3000, Date: time.Now(), TaxYear: 2026})
	contribRepo.addContribution(Contribution{ID: "ct3", ClientID: "c1", AccountID: "a3", AccountType: AccountTypeFHSA, Amount: 2000, Date: time.Now(), TaxYear: 2026})

	summary, err := e.GetContributionSummary(ctx, "c1", 2026)
	if err != nil {
		t.Fatalf("GetContributionSummary: %v", err)
	}

	if summary.ClientID != "c1" {
		t.Errorf("expected client_id c1, got %s", summary.ClientID)
	}
	if summary.TaxYear != 2026 {
		t.Errorf("expected tax_year 2026, got %d", summary.TaxYear)
	}
	if len(summary.Accounts) != 3 {
		t.Fatalf("expected 3 account entries, got %d", len(summary.Accounts))
	}

	// Check each type
	typeMap := make(map[string]AccountContribution)
	for _, ac := range summary.Accounts {
		typeMap[ac.AccountType] = ac
	}

	// RRSP
	rrsp := typeMap[AccountTypeRRSP]
	if rrsp.AnnualLimit != 32490 {
		t.Errorf("RRSP annual_limit: expected 32490, got %v", rrsp.AnnualLimit)
	}
	if rrsp.Contributed != 10000 {
		t.Errorf("RRSP contributed: expected 10000, got %v", rrsp.Contributed)
	}
	if rrsp.Remaining != 22490 {
		t.Errorf("RRSP remaining: expected 22490, got %v", rrsp.Remaining)
	}
	if rrsp.IsOverContributed {
		t.Error("RRSP should not be over-contributed")
	}
	if rrsp.Deadline == nil {
		t.Error("RRSP should have a deadline")
	}
	if rrsp.DaysUntilDeadline == nil {
		t.Error("RRSP should have days_until_deadline")
	}

	// TFSA
	tfsa := typeMap[AccountTypeTFSA]
	if tfsa.AnnualLimit != 7000 {
		t.Errorf("TFSA annual_limit: expected 7000, got %v", tfsa.AnnualLimit)
	}
	if tfsa.Contributed != 3000 {
		t.Errorf("TFSA contributed: expected 3000, got %v", tfsa.Contributed)
	}
	if tfsa.Remaining != 4000 {
		t.Errorf("TFSA remaining: expected 4000, got %v", tfsa.Remaining)
	}

	// FHSA
	fhsa := typeMap[AccountTypeFHSA]
	if fhsa.AnnualLimit != 8000 {
		t.Errorf("FHSA annual_limit: expected 8000, got %v", fhsa.AnnualLimit)
	}
	if fhsa.Contributed != 2000 {
		t.Errorf("FHSA contributed: expected 2000, got %v", fhsa.Contributed)
	}
	if fhsa.Remaining != 6000 {
		t.Errorf("FHSA remaining: expected 6000, got %v", fhsa.Remaining)
	}
}

// Test Anchor 14: No contributions returns contributed=0 and remaining=annual_limit.
func TestGetContributionSummary_NoContributions(t *testing.T) {
	e, _, accountRepo, _, _ := setupEngine()
	ctx := context.Background()

	accountRepo.addAccount(Account{ID: "a1", ClientID: "c1", AccountType: AccountTypeRRSP, Institution: "Wealthsimple"})
	accountRepo.addAccount(Account{ID: "a2", ClientID: "c1", AccountType: AccountTypeTFSA, Institution: "Wealthsimple"})

	summary, err := e.GetContributionSummary(ctx, "c1", 2026)
	if err != nil {
		t.Fatalf("GetContributionSummary: %v", err)
	}

	if len(summary.Accounts) != 2 {
		t.Fatalf("expected 2 account entries, got %d", len(summary.Accounts))
	}

	for _, ac := range summary.Accounts {
		if ac.Contributed != 0 {
			t.Errorf("%s contributed: expected 0, got %v", ac.AccountType, ac.Contributed)
		}
		if ac.AccountType == AccountTypeRRSP && ac.Remaining != DefaultRRSPLimit {
			t.Errorf("RRSP remaining: expected %v, got %v", DefaultRRSPLimit, ac.Remaining)
		}
		if ac.AccountType == AccountTypeTFSA && ac.Remaining != TFSAAnnualLimit {
			t.Errorf("TFSA remaining: expected %v, got %v", TFSAAnnualLimit, ac.Remaining)
		}
	}
}

// Test Anchor 12: FHSA contribution updates lifetime total via AccountRepository.
func TestRecordContribution_FHSA_UpdatesLifetime(t *testing.T) {
	e, contribRepo, accountRepo, _, _ := setupEngine()
	ctx := context.Background()

	accountRepo.addAccount(Account{
		ID: "a1", ClientID: "c1", AccountType: AccountTypeFHSA,
		Institution: "Wealthsimple", FHSALifetimeContributions: 10000,
	})

	// Record first contribution
	_, err := e.RecordContribution(ctx, &Contribution{
		ID: "ct1", ClientID: "c1", AccountID: "a1",
		AccountType: AccountTypeFHSA, Amount: 5000, Date: time.Now(), TaxYear: 2026,
	})
	if err != nil {
		t.Fatalf("RecordContribution: %v", err)
	}

	// Verify lifetime was updated: should recompute from records
	// Repo now has 1 contribution of $5000 for this account
	accts, _ := accountRepo.GetAccountsByClientID(ctx, "c1")
	for _, a := range accts {
		if a.ID == "a1" {
			if a.FHSALifetimeContributions != 5000 {
				t.Errorf("expected lifetime 5000, got %v", a.FHSALifetimeContributions)
			}
		}
	}

	// Record second contribution
	_, err = e.RecordContribution(ctx, &Contribution{
		ID: "ct2", ClientID: "c1", AccountID: "a1",
		AccountType: AccountTypeFHSA, Amount: 3000, Date: time.Now(), TaxYear: 2026,
	})
	if err != nil {
		t.Fatalf("RecordContribution: %v", err)
	}

	// Lifetime should now be $8000 (recomputed from all contributions for this account)
	accts, _ = accountRepo.GetAccountsByClientID(ctx, "c1")
	for _, a := range accts {
		if a.ID == "a1" {
			if a.FHSALifetimeContributions != 8000 {
				t.Errorf("expected lifetime 8000, got %v", a.FHSALifetimeContributions)
			}
		}
	}

	// Verify contributions were recorded in the repo
	contribs, _ := contribRepo.GetContributionsByClient(ctx, "c1", 2026)
	if len(contribs) != 2 {
		t.Errorf("expected 2 contributions, got %d", len(contribs))
	}
}

// Test Anchor 13: RESP contribution updates beneficiary lifetime via RESPBeneficiaryRepository.
func TestRecordContribution_RESP_UpdatesBeneficiaryLifetime(t *testing.T) {
	e, _, accountRepo, respRepo, _ := setupEngine()
	ctx := context.Background()

	benID := "resp_ben_1"
	accountRepo.addAccount(Account{
		ID: "a1", ClientID: "c1", AccountType: AccountTypeRESP,
		Institution: "Wealthsimple", RESPBeneficiaryID: &benID,
	})
	respRepo.addBeneficiary(RESPBeneficiary{
		ID: benID, ClientID: "c1", Name: "Test Child",
		LifetimeContributions: 20000,
	})

	_, err := e.RecordContribution(ctx, &Contribution{
		ID: "ct1", ClientID: "c1", AccountID: "a1",
		AccountType: AccountTypeRESP, Amount: 2500, Date: time.Now(), TaxYear: 2026,
	})
	if err != nil {
		t.Fatalf("RecordContribution: %v", err)
	}

	// Beneficiary lifetime should be updated (recomputed from contribution records)
	bens, _ := respRepo.GetRESPBeneficiariesByClientID(ctx, "c1")
	for _, b := range bens {
		if b.ID == benID {
			if b.LifetimeContributions != 2500 {
				t.Errorf("expected beneficiary lifetime 2500, got %v", b.LifetimeContributions)
			}
		}
	}
}
