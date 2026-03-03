package account

import (
	"context"
	"testing"
	"time"
)

// --- Test Anchor #15: FHSA LifetimeCap returns $40,000 ---

func TestLifetimeCap_FHSA_Returns40000(t *testing.T) {
	cap := AccountTypeFHSA.LifetimeCap()
	if cap == nil {
		t.Fatal("expected non-nil cap for FHSA")
	}
	if *cap != 40000.0 {
		t.Errorf("expected 40000, got %f", *cap)
	}
}

// --- Test Anchor #16: RESP LifetimeCap returns $50,000 ---

func TestLifetimeCap_RESP_Returns50000(t *testing.T) {
	cap := AccountTypeRESP.LifetimeCap()
	if cap == nil {
		t.Fatal("expected non-nil cap for RESP")
	}
	if *cap != 50000.0 {
		t.Errorf("expected 50000, got %f", *cap)
	}
}

// --- Test Anchor #17: RRSP LifetimeCap returns nil ---

func TestLifetimeCap_RRSP_ReturnsNil(t *testing.T) {
	cap := AccountTypeRRSP.LifetimeCap()
	if cap != nil {
		t.Errorf("expected nil cap for RRSP, got %f", *cap)
	}
}

func TestLifetimeCap_TFSA_ReturnsNil(t *testing.T) {
	cap := AccountTypeTFSA.LifetimeCap()
	if cap != nil {
		t.Errorf("expected nil cap for TFSA, got %f", *cap)
	}
}

func TestLifetimeCap_NonReg_ReturnsNil(t *testing.T) {
	cap := AccountTypeNonReg.LifetimeCap()
	if cap != nil {
		t.Errorf("expected nil cap for NON_REG, got %f", *cap)
	}
}

// --- Test Anchor #1: GetAccount with valid ID returns account ---

func TestGetAccount_ValidID_ReturnsAccount(t *testing.T) {
	repo := newMemoryAccountRepo()
	ctx := context.Background()

	benID := "resp_ben_1"
	repo.accounts["acc1"] = Account{
		ID:                        "acc1",
		ClientID:                  "c1",
		AccountType:               AccountTypeRRSP,
		Institution:               "Wealthsimple",
		Balance:                   50000.0,
		IsExternal:                false,
		RESPBeneficiaryID:         nil,
		FHSALifetimeContributions: 0,
	}
	repo.accounts["acc2"] = Account{
		ID:                        "acc2",
		ClientID:                  "c1",
		AccountType:               AccountTypeRESP,
		Institution:               "Wealthsimple",
		Balance:                   25000.0,
		IsExternal:                false,
		RESPBeneficiaryID:         &benID,
		FHSALifetimeContributions: 0,
	}

	a, err := repo.GetAccount(ctx, "acc1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.ID != "acc1" {
		t.Errorf("expected ID acc1, got %s", a.ID)
	}
	if a.ClientID != "c1" {
		t.Errorf("expected ClientID c1, got %s", a.ClientID)
	}
	if a.AccountType != AccountTypeRRSP {
		t.Errorf("expected AccountType RRSP, got %s", a.AccountType)
	}
	if a.Institution != "Wealthsimple" {
		t.Errorf("expected Institution Wealthsimple, got %s", a.Institution)
	}
	if a.Balance != 50000.0 {
		t.Errorf("expected Balance 50000, got %f", a.Balance)
	}
	if a.IsExternal != false {
		t.Error("expected IsExternal false")
	}
	if a.RESPBeneficiaryID != nil {
		t.Errorf("expected nil RESPBeneficiaryID, got %s", *a.RESPBeneficiaryID)
	}
	if a.FHSALifetimeContributions != 0 {
		t.Errorf("expected FHSALifetimeContributions 0, got %f", a.FHSALifetimeContributions)
	}
}

// --- Test Anchor #2: GetAccount with invalid ID returns error ---

func TestGetAccount_InvalidID_ReturnsError(t *testing.T) {
	repo := newMemoryAccountRepo()
	ctx := context.Background()

	_, err := repo.GetAccount(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for invalid ID")
	}
}

// --- Test Anchor #3: GetAccountsByClientID returns all accounts ---

func TestGetAccountsByClientID_ReturnsAllAccounts(t *testing.T) {
	repo := newMemoryAccountRepo()
	ctx := context.Background()

	repo.accounts["acc1"] = Account{ID: "acc1", ClientID: "c1", AccountType: AccountTypeRRSP, Institution: "Wealthsimple", Balance: 50000}
	repo.accounts["acc2"] = Account{ID: "acc2", ClientID: "c1", AccountType: AccountTypeTFSA, Institution: "Wealthsimple", Balance: 30000}
	repo.accounts["acc3"] = Account{ID: "acc3", ClientID: "c1", AccountType: AccountTypeRRSP, Institution: "RBC", Balance: 45000, IsExternal: true}
	repo.accounts["acc4"] = Account{ID: "acc4", ClientID: "c2", AccountType: AccountTypeTFSA, Institution: "TD", Balance: 20000}

	accounts, err := repo.GetAccountsByClientID(ctx, "c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(accounts) != 3 {
		t.Fatalf("expected 3 accounts, got %d", len(accounts))
	}

	// Verify includes both internal and external
	hasExternal := false
	for _, a := range accounts {
		if a.IsExternal {
			hasExternal = true
		}
	}
	if !hasExternal {
		t.Error("expected at least one external account")
	}
}

// --- Test Anchor #4: GetAccountsByClientID with no accounts returns empty slice ---

func TestGetAccountsByClientID_NoAccounts_ReturnsEmptySlice(t *testing.T) {
	repo := newMemoryAccountRepo()
	ctx := context.Background()

	accounts, err := repo.GetAccountsByClientID(ctx, "c99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if accounts == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(accounts) != 0 {
		t.Errorf("expected 0 accounts, got %d", len(accounts))
	}
}

// --- Test Anchor #5: RESP account has non-null resp_beneficiary_id ---

func TestRESPAccount_HasBeneficiaryID(t *testing.T) {
	repo := newMemoryAccountRepo()
	ctx := context.Background()

	benID := "resp_ben_1"
	repo.accounts["acc1"] = Account{
		ID:                "acc1",
		ClientID:          "c1",
		AccountType:       AccountTypeRESP,
		Institution:       "Wealthsimple",
		Balance:           25000,
		RESPBeneficiaryID: &benID,
	}

	a, err := repo.GetAccount(ctx, "acc1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.RESPBeneficiaryID == nil {
		t.Fatal("expected non-nil RESPBeneficiaryID for RESP account")
	}
	if *a.RESPBeneficiaryID != "resp_ben_1" {
		t.Errorf("expected resp_ben_1, got %s", *a.RESPBeneficiaryID)
	}
}

// --- Test Anchor #6: Non-RESP account has null resp_beneficiary_id ---

func TestNonRESPAccount_HasNilBeneficiaryID(t *testing.T) {
	repo := newMemoryAccountRepo()
	ctx := context.Background()

	repo.accounts["acc1"] = Account{
		ID:                "acc1",
		ClientID:          "c1",
		AccountType:       AccountTypeRRSP,
		Institution:       "Wealthsimple",
		Balance:           50000,
		RESPBeneficiaryID: nil,
	}

	a, err := repo.GetAccount(ctx, "acc1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.RESPBeneficiaryID != nil {
		t.Errorf("expected nil RESPBeneficiaryID for non-RESP account, got %s", *a.RESPBeneficiaryID)
	}
}

// --- Test Anchor #11: FHSA account reflects fhsa_lifetime_contributions ---

func TestFHSAAccount_ReflectsLifetimeContributions(t *testing.T) {
	repo := newMemoryAccountRepo()
	ctx := context.Background()

	repo.accounts["acc1"] = Account{
		ID:                        "acc1",
		ClientID:                  "c1",
		AccountType:               AccountTypeFHSA,
		Institution:               "Wealthsimple",
		Balance:                   15000,
		FHSALifetimeContributions: 15000,
	}

	a, err := repo.GetAccount(ctx, "acc1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.FHSALifetimeContributions != 15000 {
		t.Errorf("expected FHSALifetimeContributions 15000, got %f", a.FHSALifetimeContributions)
	}
}

// --- Test Anchor #12: Non-FHSA account has fhsa_lifetime_contributions = 0 ---

func TestNonFHSAAccount_HasZeroLifetimeContributions(t *testing.T) {
	repo := newMemoryAccountRepo()
	ctx := context.Background()

	repo.accounts["acc1"] = Account{
		ID:                        "acc1",
		ClientID:                  "c1",
		AccountType:               AccountTypeRRSP,
		Institution:               "Wealthsimple",
		Balance:                   50000,
		FHSALifetimeContributions: 0,
	}

	a, err := repo.GetAccount(ctx, "acc1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.FHSALifetimeContributions != 0 {
		t.Errorf("expected FHSALifetimeContributions 0, got %f", a.FHSALifetimeContributions)
	}
}

// --- Test Anchor #13: UpdateFHSALifetimeContributions updates the value ---

func TestUpdateFHSALifetimeContributions_UpdatesValue(t *testing.T) {
	repo := newMemoryAccountRepo()
	ctx := context.Background()

	repo.accounts["acc1"] = Account{
		ID:                        "acc1",
		ClientID:                  "c1",
		AccountType:               AccountTypeFHSA,
		Institution:               "Wealthsimple",
		Balance:                   10000,
		FHSALifetimeContributions: 10000,
	}

	err := repo.UpdateFHSALifetimeContributions(ctx, "acc1", 18000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	a, err := repo.GetAccount(ctx, "acc1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.FHSALifetimeContributions != 18000 {
		t.Errorf("expected FHSALifetimeContributions 18000, got %f", a.FHSALifetimeContributions)
	}
}

// --- Test Anchor #7: GetRESPBeneficiary with valid ID returns beneficiary ---

func TestGetRESPBeneficiary_ValidID_ReturnsBeneficiary(t *testing.T) {
	repo := newMemoryRESPBeneficiaryRepo()
	ctx := context.Background()

	dob := time.Date(2015, 6, 15, 0, 0, 0, 0, time.UTC)
	repo.beneficiaries["resp_ben_1"] = RESPBeneficiary{
		ID:                    "resp_ben_1",
		ClientID:              "c1",
		Name:                  "Arjun Sharma",
		DateOfBirth:           dob,
		LifetimeContributions: 38200,
	}

	b, err := repo.GetRESPBeneficiary(ctx, "resp_ben_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.ID != "resp_ben_1" {
		t.Errorf("expected ID resp_ben_1, got %s", b.ID)
	}
	if b.ClientID != "c1" {
		t.Errorf("expected ClientID c1, got %s", b.ClientID)
	}
	if b.Name != "Arjun Sharma" {
		t.Errorf("expected Name Arjun Sharma, got %s", b.Name)
	}
	if !b.DateOfBirth.Equal(dob) {
		t.Errorf("expected DateOfBirth %v, got %v", dob, b.DateOfBirth)
	}
	if b.LifetimeContributions != 38200 {
		t.Errorf("expected LifetimeContributions 38200, got %f", b.LifetimeContributions)
	}
}

// --- Test Anchor #8: GetRESPBeneficiary with invalid ID returns error ---

func TestGetRESPBeneficiary_InvalidID_ReturnsError(t *testing.T) {
	repo := newMemoryRESPBeneficiaryRepo()
	ctx := context.Background()

	_, err := repo.GetRESPBeneficiary(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for invalid ID")
	}
}

// --- Test Anchor #9: GetRESPBeneficiariesByClientID returns all beneficiaries ---

func TestGetRESPBeneficiariesByClientID_ReturnsBoth(t *testing.T) {
	repo := newMemoryRESPBeneficiaryRepo()
	ctx := context.Background()

	repo.beneficiaries["resp_ben_1"] = RESPBeneficiary{
		ID:          "resp_ben_1",
		ClientID:    "c1",
		Name:        "Child One",
		DateOfBirth: time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	repo.beneficiaries["resp_ben_2"] = RESPBeneficiary{
		ID:          "resp_ben_2",
		ClientID:    "c1",
		Name:        "Child Two",
		DateOfBirth: time.Date(2018, 6, 15, 0, 0, 0, 0, time.UTC),
	}
	repo.beneficiaries["resp_ben_3"] = RESPBeneficiary{
		ID:          "resp_ben_3",
		ClientID:    "c2",
		Name:        "Other Child",
		DateOfBirth: time.Date(2012, 3, 20, 0, 0, 0, 0, time.UTC),
	}

	bens, err := repo.GetRESPBeneficiariesByClientID(ctx, "c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bens) != 2 {
		t.Fatalf("expected 2 beneficiaries, got %d", len(bens))
	}
}

// --- Test Anchor #10: GetRESPBeneficiariesByClientID with no beneficiaries returns empty slice ---

func TestGetRESPBeneficiariesByClientID_NoBeneficiaries_ReturnsEmptySlice(t *testing.T) {
	repo := newMemoryRESPBeneficiaryRepo()
	ctx := context.Background()

	bens, err := repo.GetRESPBeneficiariesByClientID(ctx, "c99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bens == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(bens) != 0 {
		t.Errorf("expected 0 beneficiaries, got %d", len(bens))
	}
}

// --- Test Anchor #14: UpdateLifetimeContributions updates the value ---

func TestUpdateLifetimeContributions_UpdatesValue(t *testing.T) {
	repo := newMemoryRESPBeneficiaryRepo()
	ctx := context.Background()

	repo.beneficiaries["resp_ben_1"] = RESPBeneficiary{
		ID:                    "resp_ben_1",
		ClientID:              "c1",
		Name:                  "Child One",
		DateOfBirth:           time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC),
		LifetimeContributions: 30000,
	}

	err := repo.UpdateLifetimeContributions(ctx, "resp_ben_1", 32500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	b, err := repo.GetRESPBeneficiary(ctx, "resp_ben_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.LifetimeContributions != 32500 {
		t.Errorf("expected LifetimeContributions 32500, got %f", b.LifetimeContributions)
	}
}
