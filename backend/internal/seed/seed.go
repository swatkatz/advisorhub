// Package seed provides a data loader that populates the database with
// seed data on startup. It inserts advisors, households, clients, accounts,
// RESP beneficiaries, contributions, transfers, goals, and advisor notes,
// then emits pre-computed events for mocked scenarios.
package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/swatkatz/advisorhub/backend/internal/eventbus"
)

// SeedLoader loads seed data into the database.
type SeedLoader interface {
	Load(ctx context.Context) error
}

// Deps holds all dependencies needed by the seed loader.
type Deps struct {
	DB  *sqlx.DB
	Bus eventbus.EventBus
}

type loader struct {
	db  *sqlx.DB
	bus eventbus.EventBus
	est *time.Location
	now time.Time
}

// New creates a new SeedLoader.
func New(deps Deps) SeedLoader {
	est, _ := time.LoadLocation("America/New_York")
	return &loader{
		db:  deps.DB,
		bus: deps.Bus,
		est: est,
		now: time.Now().In(est),
	}
}

func (l *loader) Load(ctx context.Context) error {
	// Idempotency check: if advisor adv1 exists, skip seeding.
	var count int
	err := l.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM advisors WHERE id = 'adv1'")
	if err != nil {
		return fmt.Errorf("checking for existing seed data: %w", err)
	}
	if count > 0 {
		return nil
	}

	// Insert in dependency order.
	if err := l.seedAdvisors(ctx); err != nil {
		return fmt.Errorf("seeding advisors: %w", err)
	}
	if err := l.seedHouseholds(ctx); err != nil {
		return fmt.Errorf("seeding households: %w", err)
	}
	if err := l.seedClients(ctx); err != nil {
		return fmt.Errorf("seeding clients: %w", err)
	}
	if err := l.seedRESPBeneficiaries(ctx); err != nil {
		return fmt.Errorf("seeding RESP beneficiaries: %w", err)
	}
	if err := l.seedAccounts(ctx); err != nil {
		return fmt.Errorf("seeding accounts: %w", err)
	}
	if err := l.seedContributions(ctx); err != nil {
		return fmt.Errorf("seeding contributions: %w", err)
	}
	if err := l.seedClientContributionLimits(ctx); err != nil {
		return fmt.Errorf("seeding client contribution limits: %w", err)
	}
	if err := l.seedTransfers(ctx); err != nil {
		return fmt.Errorf("seeding transfers: %w", err)
	}
	if err := l.seedGoals(ctx); err != nil {
		return fmt.Errorf("seeding goals: %w", err)
	}
	if err := l.seedAdvisorNotes(ctx); err != nil {
		return fmt.Errorf("seeding advisor notes: %w", err)
	}

	// Emit pre-computed events after all data is inserted.
	if err := l.emitEvents(ctx); err != nil {
		return fmt.Errorf("emitting seed events: %w", err)
	}

	return nil
}

func (l *loader) seedAdvisors(ctx context.Context) error {
	_, err := l.db.ExecContext(ctx,
		"INSERT INTO advisors (id, name, email, role) VALUES ($1, $2, $3, $4)",
		"adv1", "Shruti K.", "shruti@wealthsimple.com", "Financial Advisor")
	return err
}

func (l *loader) seedHouseholds(ctx context.Context) error {
	households := []struct {
		id, name string
	}{
		{"h1", "Gupta Family"},
		{"h2", "Williams Family"},
	}
	for _, h := range households {
		_, err := l.db.ExecContext(ctx,
			"INSERT INTO households (id, name) VALUES ($1, $2)", h.id, h.name)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *loader) seedClients(ctx context.Context) error {
	type clientSeed struct {
		id, name, email string
		householdID     *string
		dob             time.Time
		lastMeeting     time.Time
	}

	h1 := "h1"
	h2 := "h2"

	clients := []clientSeed{
		{"c1", "Priya Sharma", "priya.sharma@email.com", nil, l.date(1988, 3, 15), l.date(2025, 12, 14)},
		{"c2", "Marcus Chen", "marcus.chen@email.com", nil, l.date(1955, 11, 8), l.date(2026, 1, 22)},
		{"c3", "Swati Gupta", "swati.gupta@email.com", &h1, l.date(1990, 7, 22), l.date(2026, 2, 10)},
		{"c4", "Rohan Gupta", "rohan.gupta@email.com", &h1, l.date(1989, 1, 30), l.date(2026, 2, 10)},
		{"c5", "Elena Vasquez", "elena.vasquez@email.com", nil, l.date(1982, 9, 11), l.date(2025, 9, 5)},
		{"c6", "James Williams", "james.williams@email.com", &h2, l.date(1975, 4, 18), l.date(2026, 2, 25)},
		{"c7", "Tanya Williams", "tanya.williams@email.com", &h2, l.date(1977, 8, 3), l.date(2026, 2, 25)},
		{"c8", "Amir Patel", "amir.patel@email.com", nil, l.date(1993, 6, 27), l.date(2025, 11, 18)},
		{"c9", "Sophie Tremblay", "sophie.tremblay@email.com", nil, l.date(1970, 12, 1), l.date(2026, 1, 30)},
		{"c10", "David Kim", "david.kim@email.com", nil, l.date(1980, 5, 14), l.date(2025, 8, 12)},
	}

	for _, c := range clients {
		_, err := l.db.ExecContext(ctx,
			"INSERT INTO clients (id, advisor_id, household_id, name, email, date_of_birth, last_meeting_date) VALUES ($1, $2, $3, $4, $5, $6, $7)",
			c.id, "adv1", c.householdID, c.name, c.email, c.dob, c.lastMeeting)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *loader) seedRESPBeneficiaries(ctx context.Context) error {
	type benSeed struct {
		id, clientID, name string
		dob                time.Time
		lifetime           float64
	}

	beneficiaries := []benSeed{
		// Priya's son
		{"ben_c1_1", "c1", "Arjun Sharma", l.date(2019, 6, 15), 38200},
		// David's children — oldest turns 17 next year (2027)
		{"ben_c10_1", "c10", "Ethan Kim", l.date(2010, 8, 20), 24000},
		{"ben_c10_2", "c10", "Lily Kim", l.date(2013, 4, 22), 20000},
	}

	for _, b := range beneficiaries {
		_, err := l.db.ExecContext(ctx,
			"INSERT INTO resp_beneficiaries (id, client_id, name, date_of_birth, lifetime_contributions) VALUES ($1, $2, $3, $4, $5)",
			b.id, b.clientID, b.name, b.dob, b.lifetime)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *loader) seedAccounts(ctx context.Context) error {
	type acctSeed struct {
		id, clientID, acctType, institution string
		balance                             float64
		isExternal                          bool
		respBeneficiaryID                   *string
		fhsaLifetime                        float64
		lastActivity                        time.Time
	}

	benC1 := "ben_c1_1"
	benC10_1 := "ben_c10_1"

	recentActivity := l.daysAgo(5)

	accounts := []acctSeed{
		// Priya (c1) — AUM ~$485K internal
		// RRSP over-contribution scenario: $18,860 WS + $15,000 RBC = $33,860 vs $31,560 limit
		{"acc_c1_rrsp_ws", "c1", "RRSP", "Wealthsimple", 218800, false, nil, 0, recentActivity},
		{"acc_c1_rrsp_rbc", "c1", "RRSP", "RBC", 45000, true, nil, 0, recentActivity},
		{"acc_c1_tfsa_ws", "c1", "TFSA", "Wealthsimple", 152000, false, nil, 0, recentActivity},
		{"acc_c1_tfsa_rbc", "c1", "TFSA", "RBC", 12000, true, nil, 0, recentActivity},
		{"acc_c1_fhsa_ws", "c1", "FHSA", "Wealthsimple", 24000, false, nil, 24000, recentActivity},
		{"acc_c1_resp_ws", "c1", "RESP", "Wealthsimple", 38200, false, &benC1, 0, recentActivity},

		// Marcus (c2) — AUM $1.25M, RRIF conversion scenario (turns 71 in Nov 2026)
		{"acc_c2_rrsp_ws", "c2", "RRSP", "Wealthsimple", 620000, false, nil, 0, recentActivity},
		{"acc_c2_tfsa_ws", "c2", "TFSA", "Wealthsimple", 380000, false, nil, 0, recentActivity},
		{"acc_c2_nonreg_ws", "c2", "NON_REG", "Wealthsimple", 250000, false, nil, 0, recentActivity},

		// Swati (c3) — RRSP deadline scenario, $8,200 room remaining
		{"acc_c3_rrsp_ws", "c3", "RRSP", "Wealthsimple", 185000, false, nil, 0, recentActivity},
		{"acc_c3_tfsa_ws", "c3", "TFSA", "Wealthsimple", 95000, false, nil, 0, recentActivity},
		{"acc_c3_fhsa_ws", "c3", "FHSA", "Wealthsimple", 16000, false, nil, 16000, recentActivity},

		// Rohan (c4) — NON_REG $45,200 cash, idle 34 days
		{"acc_c4_nonreg_ws", "c4", "NON_REG", "Wealthsimple", 45200, false, nil, 0, l.daysAgo(34)},
		{"acc_c4_tfsa_ws", "c4", "TFSA", "Wealthsimple", 194800, false, nil, 0, recentActivity},

		// Elena (c5) — engagement stale scenario
		{"acc_c5_rrsp_ws", "c5", "RRSP", "Wealthsimple", 195000, false, nil, 0, recentActivity},
		{"acc_c5_tfsa_ws", "c5", "TFSA", "Wealthsimple", 115000, false, nil, 0, recentActivity},

		// James (c6) — transfer completed, NON_REG received $185K
		{"acc_c6_nonreg_ws", "c6", "NON_REG", "Wealthsimple", 185000, false, nil, 0, recentActivity},
		{"acc_c6_rrsp_ws", "c6", "RRSP", "Wealthsimple", 815000, false, nil, 0, recentActivity},
		{"acc_c6_tfsa_ws", "c6", "TFSA", "Wealthsimple", 400000, false, nil, 0, recentActivity},

		// Tanya (c7) — portfolio drift scenario
		{"acc_c7_rrsp_ws", "c7", "RRSP", "Wealthsimple", 400000, false, nil, 0, recentActivity},
		{"acc_c7_tfsa_ws", "c7", "TFSA", "Wealthsimple", 300000, false, nil, 0, recentActivity},

		// Amir (c8) — pending transfer, tax loss scenario
		{"acc_c8_rrsp_ws", "c8", "RRSP", "Wealthsimple", 120000, false, nil, 0, recentActivity},
		{"acc_c8_tfsa_ws", "c8", "TFSA", "Wealthsimple", 75000, false, nil, 0, recentActivity},

		// Sophie (c9) — dividend scenario
		{"acc_c9_rrsp_ws", "c9", "RRSP", "Wealthsimple", 420000, false, nil, 0, recentActivity},
		{"acc_c9_tfsa_ws", "c9", "TFSA", "Wealthsimple", 280000, false, nil, 0, recentActivity},
		{"acc_c9_nonreg_ws", "c9", "NON_REG", "Wealthsimple", 190000, false, nil, 0, recentActivity},

		// David (c10) — RESP, engagement stale
		{"acc_c10_resp_ws", "c10", "RESP", "Wealthsimple", 44000, false, &benC10_1, 0, recentActivity},
		{"acc_c10_rrsp_ws", "c10", "RRSP", "Wealthsimple", 320000, false, nil, 0, recentActivity},
		{"acc_c10_tfsa_ws", "c10", "TFSA", "Wealthsimple", 176000, false, nil, 0, recentActivity},
	}

	for _, a := range accounts {
		_, err := l.db.ExecContext(ctx,
			`INSERT INTO accounts (id, client_id, account_type, institution, balance, is_external, resp_beneficiary_id, fhsa_lifetime_contributions, last_activity_date)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			a.id, a.clientID, a.acctType, a.institution, a.balance, a.isExternal, a.respBeneficiaryID, a.fhsaLifetime, a.lastActivity)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *loader) seedContributions(ctx context.Context) error {
	type contribSeed struct {
		id, clientID, accountID, accountType string
		amount                               float64
		date                                 time.Time
		taxYear                              int
	}

	contributions := []contribSeed{
		// Priya (c1): RRSP over-contribution — $18,860 WS + $15,000 RBC = $33,860 vs $31,560 limit
		{"contrib_c1_1", "c1", "acc_c1_rrsp_ws", "RRSP", 18860, l.date(2026, 1, 15), 2025},
		{"contrib_c1_2", "c1", "acc_c1_rrsp_rbc", "RRSP", 15000, l.date(2026, 2, 1), 2025},
		// Priya TFSA
		{"contrib_c1_3", "c1", "acc_c1_tfsa_ws", "TFSA", 7000, l.date(2026, 1, 5), 2026},
		// Priya FHSA
		{"contrib_c1_4", "c1", "acc_c1_fhsa_ws", "FHSA", 8000, l.date(2026, 1, 10), 2026},
		// Priya RESP: $1,800 ytd (needs $700 more for full CESG)
		{"contrib_c1_5", "c1", "acc_c1_resp_ws", "RESP", 1800, l.date(2026, 2, 15), 2026},

		// Marcus (c2): modest RRSP contribution
		{"contrib_c2_1", "c2", "acc_c2_rrsp_ws", "RRSP", 15000, l.date(2026, 1, 20), 2025},

		// Swati (c3): RRSP contributed $24,290 (limit $32,490 - $8,200 room remaining)
		{"contrib_c3_1", "c3", "acc_c3_rrsp_ws", "RRSP", 24290, l.date(2026, 1, 25), 2025},
		// Swati TFSA
		{"contrib_c3_2", "c3", "acc_c3_tfsa_ws", "TFSA", 7000, l.date(2026, 1, 8), 2026},

		// Rohan (c4): TFSA
		{"contrib_c4_1", "c4", "acc_c4_tfsa_ws", "TFSA", 7000, l.date(2026, 1, 12), 2026},

		// Elena (c5): RRSP
		{"contrib_c5_1", "c5", "acc_c5_rrsp_ws", "RRSP", 20000, l.date(2026, 1, 18), 2025},

		// Amir (c8): TFSA
		{"contrib_c8_1", "c8", "acc_c8_tfsa_ws", "TFSA", 7000, l.date(2026, 1, 6), 2026},

		// Sophie (c9): RRSP + TFSA
		{"contrib_c9_1", "c9", "acc_c9_rrsp_ws", "RRSP", 25000, l.date(2026, 1, 14), 2025},
		{"contrib_c9_2", "c9", "acc_c9_tfsa_ws", "TFSA", 7000, l.date(2026, 2, 1), 2026},

		// David (c10): RESP contribution
		{"contrib_c10_1", "c10", "acc_c10_resp_ws", "RESP", 2500, l.date(2026, 1, 20), 2026},
	}

	for _, c := range contributions {
		_, err := l.db.ExecContext(ctx,
			"INSERT INTO contributions (id, client_id, account_id, account_type, amount, date, tax_year) VALUES ($1, $2, $3, $4, $5, $6, $7)",
			c.id, c.clientID, c.accountID, c.accountType, c.amount, c.date, c.taxYear)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *loader) seedClientContributionLimits(ctx context.Context) error {
	type limitSeed struct {
		id, clientID string
		taxYear      int
		rrspLimit    float64
	}

	limits := []limitSeed{
		// Priya: limit $31,560 (18% of $175,333 earned income)
		{"ccl_c1_2025", "c1", 2025, 31560},
		{"ccl_c2_2025", "c2", 2025, 32490},
		{"ccl_c3_2025", "c3", 2025, 32490},
		{"ccl_c4_2025", "c4", 2025, 32490},
		{"ccl_c5_2025", "c5", 2025, 32490},
		{"ccl_c6_2025", "c6", 2025, 32490},
		{"ccl_c7_2025", "c7", 2025, 32490},
		{"ccl_c8_2025", "c8", 2025, 32490},
		{"ccl_c9_2025", "c9", 2025, 32490},
		{"ccl_c10_2025", "c10", 2025, 32490},
	}

	for _, cl := range limits {
		_, err := l.db.ExecContext(ctx,
			"INSERT INTO client_contribution_limits (id, client_id, tax_year, rrsp_deduction_limit) VALUES ($1, $2, $3, $4)",
			cl.id, cl.clientID, cl.taxYear, cl.rrspLimit)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *loader) seedTransfers(ctx context.Context) error {
	type transferSeed struct {
		id, clientID, source, acctType string
		amount                         float64
		status                         string
		initiatedAt                    time.Time
		daysInStage                    int
	}

	transfers := []transferSeed{
		{"t1", "c8", "TD", "RRSP", 67400, "DOCUMENTS_SUBMITTED", l.date(2026, 2, 1), 18},
		{"t2", "c6", "Scotia", "NON_REG", 185000, "INVESTED", l.date(2026, 1, 15), 0},
		{"t3", "c1", "RBC", "RRSP", 42000, "IN_TRANSIT", l.date(2026, 2, 20), 3},
		{"t4", "c10", "BMO", "TFSA", 28500, "IN_REVIEW", l.date(2026, 2, 18), 5},
		{"t5", "c5", "Desjardins", "RRSP", 55000, "INITIATED", l.date(2026, 2, 26), 2},
		{"t6", "c9", "National Bank", "NON_REG", 120000, "IN_TRANSIT", l.date(2026, 2, 15), 6},
	}

	for _, t := range transfers {
		lastStatusChange := l.daysAgo(t.daysInStage)
		if t.status == "INVESTED" {
			lastStatusChange = l.now
		}
		_, err := l.db.ExecContext(ctx,
			"INSERT INTO transfers (id, client_id, source_institution, account_type, amount, status, initiated_at, last_status_change) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
			t.id, t.clientID, t.source, t.acctType, t.amount, t.status, t.initiatedAt, lastStatusChange)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *loader) seedGoals(ctx context.Context) error {
	type goalSeed struct {
		id, clientID string
		householdID  *string
		name         string
		targetAmount *float64
		targetDate   *time.Time
		progressPct  int
		status       string
	}

	h1 := "h1"
	h2 := "h2"
	amt120k := 120000.0
	amt2m := 2000000.0
	amt200k := 200000.0
	amt30k := 30000.0
	amt800k := 800000.0
	amt150k := 150000.0
	amt25k := 25000.0
	amt1_5m := 1500000.0
	amt80k := 80000.0
	td2030 := l.date(2030, 12, 31)
	td2026 := l.date(2026, 11, 30)
	td2028 := l.date(2028, 6, 30)
	td2027 := l.date(2027, 3, 31)
	td2042 := l.date(2042, 9, 30)
	td2036 := l.date(2036, 12, 31)
	td2029 := l.date(2029, 12, 31)
	td2028b := l.date(2028, 12, 31)
	td2030b := l.date(2030, 9, 30)

	goals := []goalSeed{
		{"goal_c1_1", "c1", nil, "First home (FHSA)", &amt120k, &td2030, 28, "BEHIND"},
		{"goal_c2_1", "c2", nil, "Retirement at 65", &amt2m, &td2026, 85, "ON_TRACK"},
		{"goal_h1_1", "c3", &h1, "First home", &amt200k, &td2028, 45, "ON_TRACK"},
		{"goal_c3_1", "c3", nil, "Mat leave savings", &amt30k, &td2027, 90, "AHEAD"},
		{"goal_c5_1", "c5", nil, "Retirement at 60", &amt800k, &td2042, 42, "BEHIND"},
		{"goal_h2_1", "c6", &h2, "Kids education (RESP)", &amt150k, &td2036, 68, "ON_TRACK"},
		{"goal_c8_1", "c8", nil, "Emergency fund", &amt25k, &td2029, 60, "ON_TRACK"},
		{"goal_c9_1", "c9", nil, "Early retirement at 58", &amt1_5m, &td2028b, 72, "ON_TRACK"},
		{"goal_c10_1", "c10", nil, "Son's university", &amt80k, &td2030b, 55, "BEHIND"},
	}

	for _, g := range goals {
		_, err := l.db.ExecContext(ctx,
			"INSERT INTO goals (id, client_id, household_id, name, target_amount, target_date, progress_pct, status) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
			g.id, g.clientID, g.householdID, g.name, g.targetAmount, g.targetDate, g.progressPct, g.status)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *loader) seedAdvisorNotes(ctx context.Context) error {
	type noteSeed struct {
		id, clientID string
		date         time.Time
		text         string
	}

	notes := []noteSeed{
		// Priya (c1)
		{"note_c1_1", "c1", l.date(2025, 12, 14),
			"Discussed RRSP contribution strategy. Priya wasn't aware of employer RRSP contribution at RBC. Need to follow up on withdrawal options to avoid over-contribution."},
		{"note_c1_2", "c1", l.date(2025, 10, 5),
			"RESP review — son Arjun's education fund on track. Discussed CESG matching and importance of contributing at least $2,500 annually for full grant."},
		{"note_c1_3", "c1", l.date(2025, 8, 20),
			"First meeting to discuss FHSA strategy for home purchase goal. Priya targets $120K down payment by 2030."},

		// Marcus (c2)
		{"note_c2_1", "c2", l.date(2026, 1, 22),
			"Marcus wants to discuss RRIF conversion options. Prefers gradual drawdown strategy. Concerned about tax implications of mandatory minimum withdrawals."},
		{"note_c2_2", "c2", l.date(2025, 10, 15),
			"Portfolio review — strong performance in 2025. Marcus comfortable with current allocation. Retirement income plan on track."},

		// Swati (c3)
		{"note_c3_1", "c3", l.date(2026, 2, 10),
			"Discussed RRSP strategy. Swati wants to maximize before returning from mat leave. $8,200 room remaining — deadline March 3."},
		{"note_c3_2", "c3", l.date(2025, 11, 20),
			"Joint meeting with Rohan. First home purchase timeline moved to 2028. Both want to keep FHSA contributions on schedule."},
		{"note_c3_3", "c3", l.date(2025, 9, 8),
			"Mat leave savings goal nearly complete at 90%. Swati considering redirecting future savings to RRSP once mat leave fund is fully topped up."},

		// Rohan (c4)
		{"note_c4_1", "c4", l.date(2026, 2, 10),
			"Rohan mentioned wanting to keep non-reg cash liquid for home down payment, but hasn't deployed the $45K sitting in account for over a month."},
		{"note_c4_2", "c4", l.date(2025, 11, 20),
			"Joint meeting with Swati about home purchase plans. Rohan handles the non-registered savings side."},

		// Elena (c5)
		{"note_c5_1", "c5", l.date(2025, 9, 5),
			"Discussed retirement timeline — Elena wants to retire at 60. Current savings rate needs to increase. Mentioned mortgage renewal coming up in April 2026."},
		{"note_c5_2", "c5", l.date(2025, 6, 12),
			"Mid-year review. Elena concerned about retirement gap. Suggested increasing RRSP contributions and reviewing investment mix."},
		{"note_c5_3", "c5", l.date(2025, 3, 15),
			"Annual planning session. Set retirement goal at $800K by 2042. Current progress at 42% — needs acceleration."},

		// James (c6)
		{"note_c6_1", "c6", l.date(2026, 2, 25),
			"Scotia transfer of $185K non-reg account just completed. Discussed investment strategy for the new funds — wants balanced portfolio."},
		{"note_c6_2", "c6", l.date(2025, 12, 10),
			"Initiated transfer from Scotia. James consolidating all accounts to Wealthsimple for simplicity. Expected 4-6 week timeline."},
		{"note_c6_3", "c6", l.date(2025, 10, 1),
			"Joint meeting with Tanya about kids' education funding. RESP on track at 68% of $150K target. Discussed rebalancing RESP holdings."},

		// Tanya (c7)
		{"note_c7_1", "c7", l.date(2026, 2, 25),
			"Portfolio review — noticed tech allocation has drifted to 42% vs 30% target. Discussed rebalancing options but Tanya wants to wait for Q1 earnings."},
		{"note_c7_2", "c7", l.date(2025, 11, 15),
			"Tanya interested in adding ESG component to portfolio. Will research options and present at next meeting."},

		// Amir (c8)
		{"note_c8_1", "c8", l.date(2025, 11, 18),
			"RRSP transfer from TD initiated — $67,400. Amir frustrated with TD's slow processing. Documents submitted, waiting for review."},
		{"note_c8_2", "c8", l.date(2025, 9, 22),
			"Reviewed tax-loss harvesting opportunities. Canadian Energy ETF showing $3,200 unrealized loss — could offset gains from earlier in the year."},
		{"note_c8_3", "c8", l.date(2025, 7, 10),
			"Emergency fund goal on track at 60%. Amir wants to reach $25K before end of 2029. Comfortable with current savings rate."},

		// Sophie (c9)
		{"note_c9_1", "c9", l.date(2026, 1, 30),
			"Quarterly review — Sophie received $1,240 dividend. Discussed reinvestment strategy. Early retirement goal at 72% — on track for age 58 target."},
		{"note_c9_2", "c9", l.date(2025, 10, 25),
			"Sophie exploring options to accelerate retirement timeline. Discussed increasing non-reg contributions and optimizing for tax-efficient growth."},
		{"note_c9_3", "c9", l.date(2025, 8, 5),
			"Transfer from National Bank in progress — $120K non-reg account. Sophie consolidating for better visibility and lower fees."},

		// David (c10)
		{"note_c10_1", "c10", l.date(2025, 8, 12),
			"RESP strategy review — oldest child Ethan turns 17 next year. Need to maximize remaining CESG years. Current lifetime at $24K for Ethan, $20K for Lily."},
		{"note_c10_2", "c10", l.date(2025, 5, 20),
			"David concerned about university costs increasing. Discussed adjusting the $80K target. Son's university goal at 55% — needs to be accelerated."},
		{"note_c10_3", "c10", l.date(2025, 3, 8),
			"Annual RESP contribution of $2,500 for both children completed. Full CESG match of $500 per child secured for 2025."},
	}

	for _, n := range notes {
		_, err := l.db.ExecContext(ctx,
			"INSERT INTO advisor_notes (id, client_id, advisor_id, date, text) VALUES ($1, $2, $3, $4, $5)",
			n.id, n.clientID, "adv1", n.date, n.text)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *loader) emitEvents(ctx context.Context) error {
	events := []eventbus.EventEnvelope{
		{
			ID:         "seed_evt_1",
			Type:       "PortfolioDrift",
			EntityID:   "c9",
			EntityType: eventbus.EntityTypeClient,
			Payload:    mustJSON(map[string]any{"client_id": "c9", "drift_pct": 12, "current_allocation": map[string]any{"tech": 42, "bonds": 20, "intl": 18, "realestate": 10, "cash": 10}, "target_allocation": map[string]any{"tech": 30, "bonds": 25, "intl": 20, "realestate": 15, "cash": 10}}),
			Source:     eventbus.SourceAnalytical,
			Timestamp:  l.now,
		},
		{
			ID:         "seed_evt_2",
			Type:       "TaxLossOpportunity",
			EntityID:   "c8",
			EntityType: eventbus.EntityTypeClient,
			Payload:    mustJSON(map[string]any{"client_id": "c8", "holding": "Canadian Energy ETF", "unrealized_loss": 3200}),
			Source:     eventbus.SourceAnalytical,
			Timestamp:  l.now,
		},
		{
			ID:         "seed_evt_3",
			Type:       "DividendReceived",
			EntityID:   "c9",
			EntityType: eventbus.EntityTypeClient,
			Payload:    mustJSON(map[string]any{"client_id": "c9", "amount": 1240}),
			Source:     eventbus.SourceReactive,
			Timestamp:  l.now,
		},
		{
			ID:         "seed_evt_4",
			Type:       "ContributionProcessed",
			EntityID:   "c6",
			EntityType: eventbus.EntityTypeClient,
			Payload:    mustJSON(map[string]any{"client_id": "c6", "account_type": "NON_REG", "amount": 185000}),
			Source:     eventbus.SourceReactive,
			Timestamp:  l.now,
		},
		{
			ID:         "seed_evt_5",
			Type:       "TransferCompleted",
			EntityID:   "t2",
			EntityType: eventbus.EntityTypeTransfer,
			Payload:    mustJSON(map[string]any{"transfer_id": "t2", "client_id": "c6", "source_institution": "Scotia", "account_type": "NON_REG", "amount": 185000}),
			Source:     eventbus.SourceReactive,
			Timestamp:  l.now,
		},
	}

	for _, evt := range events {
		if err := l.bus.Publish(ctx, evt); err != nil {
			return err
		}
	}
	return nil
}

// date constructs a date in EST.
func (l *loader) date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, l.est)
}

// daysAgo returns now minus the given number of days.
func (l *loader) daysAgo(days int) time.Time {
	return l.now.AddDate(0, 0, -days)
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("marshaling seed event payload: %v", err))
	}
	return b
}
