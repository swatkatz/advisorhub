package temporal

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// Test anchor 2: Client born 1955-11-08, referenceDate 2026-03-02,
// RRIF_CONVERSION rule (age 71, within_days 365) → AgeMilestone emitted
// (turns 71 at year-end, within 365-day horizon).
// Test anchor 3: Client born 1990-07-22, same rule → no event (age at year-end 36).
func TestAgeApproaching_Client(t *testing.T) {
	tests := []struct {
		name          string
		dob           time.Time
		clientName    string
		referenceDate time.Time
		rule          TemporalRule
		wantEmit      bool
		wantAge       int
		wantTargetAge int
		wantYearTurn  int
	}{
		{
			name:          "client turning 71 this year, within 365-day horizon",
			dob:           time.Date(1955, 11, 8, 0, 0, 0, 0, time.UTC),
			clientName:    "Marcus Chen",
			referenceDate: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
			rule:          Rules[0], // RRIF_CONVERSION
			wantEmit:      true,
			wantAge:       70,
			wantTargetAge: 71,
			wantYearTurn:  2026,
		},
		{
			name:          "young client, age at year-end 36, below target 71",
			dob:           time.Date(1990, 7, 22, 0, 0, 0, 0, time.UTC),
			clientName:    "Swati Gupta",
			referenceDate: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
			rule:          Rules[0], // RRIF_CONVERSION
			wantEmit:      false,
		},
		{
			name:          "client already past target age, milestone year in past",
			dob:           time.Date(1950, 1, 15, 0, 0, 0, 0, time.UTC),
			clientName:    "Old Client",
			referenceDate: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
			rule:          Rules[0], // RRIF_CONVERSION
			// yearTurning = 1950+71 = 2021 < 2026 → past milestone, skip
			wantEmit: false,
		},
		{
			name:          "client turning target age next year, outside 365-day horizon",
			dob:           time.Date(1955, 11, 8, 0, 0, 0, 0, time.UTC),
			clientName:    "Marcus Chen",
			referenceDate: time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
			rule:          Rules[0], // RRIF_CONVERSION
			// yearTurning = 2026, Jan 1 2026 is 366 days from Dec 31 2024, 366 > 365 → skip
			wantEmit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := newMockEventBus()
			clients := newMockClientRepo()
			clients.AddClient("advisor1", Client{
				ID:          "c1",
				Name:        tt.clientName,
				DateOfBirth: tt.dob,
			})

			s := NewScanner(clients, newMockAccountRepo(), newMockRESPBenRepo(), newMockContribEngine(), bus)

			result, err := s.RunSweep(context.Background(), "advisor1", tt.referenceDate)
			if err != nil {
				t.Fatalf("RunSweep returned error: %v", err)
			}

			events := bus.EventsByType(EventAgeMilestone)

			if tt.wantEmit {
				if len(events) == 0 {
					t.Fatal("expected AgeMilestone event but none emitted")
				}

				evt := events[0]
				if evt.EntityType != EntityTypeClient {
					t.Errorf("EntityType = %q, want %q", evt.EntityType, EntityTypeClient)
				}
				if evt.EntityID != "c1" {
					t.Errorf("EntityID = %q, want %q", evt.EntityID, "c1")
				}
				if evt.Source != "TEMPORAL" {
					t.Errorf("Source = %q, want TEMPORAL", evt.Source)
				}

				var payload map[string]any
				if err := json.Unmarshal(evt.Payload, &payload); err != nil {
					t.Fatalf("unmarshal payload: %v", err)
				}
				if int(payload["target_age"].(float64)) != tt.wantTargetAge {
					t.Errorf("target_age = %v, want %d", payload["target_age"], tt.wantTargetAge)
				}
				if int(payload["current_age"].(float64)) != tt.wantAge {
					t.Errorf("current_age = %v, want %d", payload["current_age"], tt.wantAge)
				}
				if int(payload["year_turning"].(float64)) != tt.wantYearTurn {
					t.Errorf("year_turning = %v, want %d", payload["year_turning"], tt.wantYearTurn)
				}
			} else {
				if len(events) > 0 {
					t.Errorf("expected no AgeMilestone events but got %d", len(events))
				}
			}

			if result.RulesEvaluated == 0 {
				t.Error("RulesEvaluated should be > 0")
			}
		})
	}
}

// Test anchor 4: RESP beneficiary born 2009-05-10, referenceDate 2025-06-01,
// RESP_LAST_CESG rule (age 17, within_days 365) → AgeMilestone emitted
// (turns 17 in 2026, within 365-day horizon from June 2025).
// Test anchor 5: Beneficiary born 2015-04-20, referenceDate 2026-03-02 → no event
// (age at year-end 11, below target 17).
func TestAgeApproaching_RESPBeneficiary(t *testing.T) {
	tests := []struct {
		name          string
		dob           time.Time
		benName       string
		referenceDate time.Time
		wantEmit      bool
		wantTargetAge int
		wantYearTurn  int
	}{
		{
			name:          "beneficiary turning 17 next year, within 365-day horizon",
			dob:           time.Date(2009, 5, 10, 0, 0, 0, 0, time.UTC),
			benName:       "Teen Beneficiary",
			referenceDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
			wantEmit:      true,
			wantTargetAge: 17,
			wantYearTurn:  2026,
		},
		{
			name:          "young beneficiary, age at year-end 11, below target",
			dob:           time.Date(2015, 4, 20, 0, 0, 0, 0, time.UTC),
			benName:       "Young Beneficiary",
			referenceDate: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
			wantEmit:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := newMockEventBus()
			clients := newMockClientRepo()
			clients.AddClient("advisor1", Client{
				ID:   "c1",
				Name: "Parent Client",
			})

			respBen := newMockRESPBenRepo()
			respBen.AddBeneficiary(RESPBeneficiary{
				ID:          "resp_ben_1",
				ClientID:    "c1",
				Name:        tt.benName,
				DateOfBirth: tt.dob,
			})

			s := NewScanner(clients, newMockAccountRepo(), respBen, newMockContribEngine(), bus)
			_, err := s.RunSweep(context.Background(), "advisor1", tt.referenceDate)
			if err != nil {
				t.Fatalf("RunSweep returned error: %v", err)
			}

			events := bus.EventsByType(EventAgeMilestone)

			if tt.wantEmit {
				// Filter for RESP beneficiary events (EntityType = RESPBeneficiary)
				var benEvents []EventEnvelope
				for _, e := range events {
					if e.EntityType == EntityTypeRESPBeneficiary {
						benEvents = append(benEvents, e)
					}
				}
				if len(benEvents) == 0 {
					t.Fatal("expected AgeMilestone event for beneficiary but none emitted")
				}

				evt := benEvents[0]
				if evt.EntityID != "resp_ben_1" {
					t.Errorf("EntityID = %q, want %q", evt.EntityID, "resp_ben_1")
				}

				var payload map[string]any
				if err := json.Unmarshal(evt.Payload, &payload); err != nil {
					t.Fatalf("unmarshal payload: %v", err)
				}
				if int(payload["target_age"].(float64)) != tt.wantTargetAge {
					t.Errorf("target_age = %v, want %d", payload["target_age"], tt.wantTargetAge)
				}
				if int(payload["year_turning"].(float64)) != tt.wantYearTurn {
					t.Errorf("year_turning = %v, want %d", payload["year_turning"], tt.wantYearTurn)
				}
				if payload["beneficiary_id"] != "resp_ben_1" {
					t.Errorf("beneficiary_id = %v, want resp_ben_1", payload["beneficiary_id"])
				}
			} else {
				var benEvents []EventEnvelope
				for _, e := range events {
					if e.EntityType == EntityTypeRESPBeneficiary {
						benEvents = append(benEvents, e)
					}
				}
				if len(benEvents) > 0 {
					t.Errorf("expected no AgeMilestone events for beneficiary but got %d", len(benEvents))
				}
			}
		})
	}
}

// Test anchor 6: RRSP account, referenceDate near deadline, room $8200 → emit.
// Test anchor 7: Same but room = 0 → no emit (fully contributed).
// Test anchor 8: RRSP account, referenceDate far from deadline → no emit.
// Test anchor 9: Two RRSP accounts same client, room $5000 → exactly one event (grouped by client).
func TestDeadlineWithRoom(t *testing.T) {
	// RRSP deadline for tax year 2025 = 60 days after Dec 31, 2025 = March 1, 2026.
	// referenceDate Feb 19, 2026 → 10 days to March 1.
	tests := []struct {
		name          string
		referenceDate time.Time
		accounts      []Account
		room          float64
		roomErr       error
		wantEmit      bool
		wantDaysUntil int
		wantRoom      float64
	}{
		{
			name:          "within 30 days, room available",
			referenceDate: time.Date(2026, 2, 19, 0, 0, 0, 0, time.UTC),
			accounts: []Account{
				{ID: "acc1", ClientID: "c1", AccountType: AccountTypeRRSP, Balance: 50000},
			},
			room:          8200,
			wantEmit:      true,
			wantDaysUntil: 10, // Feb 19 to March 1
			wantRoom:      8200,
		},
		{
			name:          "within 30 days, no room (fully contributed)",
			referenceDate: time.Date(2026, 2, 19, 0, 0, 0, 0, time.UTC),
			accounts: []Account{
				{ID: "acc1", ClientID: "c1", AccountType: AccountTypeRRSP, Balance: 50000},
			},
			room:     0,
			wantEmit: false,
		},
		{
			name:          "outside 30-day window",
			referenceDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
			accounts: []Account{
				{ID: "acc1", ClientID: "c1", AccountType: AccountTypeRRSP, Balance: 50000},
			},
			room:     5000,
			wantEmit: false, // 45 days to March 1 > 30
		},
		{
			name:          "two RRSP accounts same client, emits once per client",
			referenceDate: time.Date(2026, 2, 19, 0, 0, 0, 0, time.UTC),
			accounts: []Account{
				{ID: "acc1", ClientID: "c1", AccountType: AccountTypeRRSP, Institution: "WS", Balance: 30000},
				{ID: "acc2", ClientID: "c1", AccountType: AccountTypeRRSP, Institution: "RBC", Balance: 20000},
			},
			room:          5000,
			wantEmit:      true,
			wantDaysUntil: 10,
			wantRoom:      5000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := newMockEventBus()
			clients := newMockClientRepo()
			clients.AddClient("advisor1", Client{ID: "c1", Name: "Test Client"})

			acctRepo := newMockAccountRepo()
			for _, a := range tt.accounts {
				acctRepo.AddAccount(a)
			}

			contrib := newMockContribEngine()
			// Set room for taxYear = referenceDate.Year() - 1 (RRSP deadline is for prev tax year)
			contrib.SetRoom("c1", AccountTypeRRSP, tt.referenceDate.Year()-1, tt.room)
			if tt.roomErr != nil {
				contrib.SetError("c1", AccountTypeRRSP, tt.referenceDate.Year()-1, tt.roomErr)
			}

			s := NewScanner(clients, acctRepo, newMockRESPBenRepo(), contrib, bus)
			_, err := s.RunSweep(context.Background(), "advisor1", tt.referenceDate)
			if err != nil {
				t.Fatalf("RunSweep returned error: %v", err)
			}

			events := bus.EventsByType(EventDeadlineApproaching)

			if tt.wantEmit {
				// Filter for RRSP deadline events
				var rrspEvents []EventEnvelope
				for _, e := range events {
					var p map[string]any
					json.Unmarshal(e.Payload, &p)
					if p["account_type"] == AccountTypeRRSP {
						rrspEvents = append(rrspEvents, e)
					}
				}
				if len(rrspEvents) == 0 {
					t.Fatal("expected DeadlineApproaching event but none emitted")
				}
				if len(rrspEvents) > 1 {
					t.Errorf("expected exactly 1 DeadlineApproaching event per client, got %d", len(rrspEvents))
				}

				var payload map[string]any
				json.Unmarshal(rrspEvents[0].Payload, &payload)
				if int(payload["days_until"].(float64)) != tt.wantDaysUntil {
					t.Errorf("days_until = %v, want %d", payload["days_until"], tt.wantDaysUntil)
				}
				if payload["room_remaining"].(float64) != tt.wantRoom {
					t.Errorf("room_remaining = %v, want %v", payload["room_remaining"], tt.wantRoom)
				}
			} else {
				var rrspEvents []EventEnvelope
				for _, e := range events {
					var p map[string]any
					json.Unmarshal(e.Payload, &p)
					if p["account_type"] == AccountTypeRRSP {
						rrspEvents = append(rrspEvents, e)
					}
				}
				if len(rrspEvents) > 0 {
					t.Errorf("expected no RRSP DeadlineApproaching events but got %d", len(rrspEvents))
				}
			}
		})
	}
}

// Test anchor 10: last_meeting_date 178 days before referenceDate, threshold 180 → no emit.
// Test anchor 11: last_meeting_date 202 days before referenceDate → emit with days_since 202.
// Test anchor 16: same rule, different reference dates → different days_since values.
func TestDaysSince(t *testing.T) {
	tests := []struct {
		name            string
		lastMeetingDate time.Time
		referenceDate   time.Time
		wantEmit        bool
		wantDaysSince   int
	}{
		{
			name:            "178 days, below 180 threshold",
			lastMeetingDate: time.Date(2025, 9, 5, 0, 0, 0, 0, time.UTC),
			referenceDate:   time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
			wantEmit:        false, // 178 < 180
		},
		{
			name:            "202 days, above 180 threshold",
			lastMeetingDate: time.Date(2025, 8, 12, 0, 0, 0, 0, time.UTC),
			referenceDate:   time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
			wantEmit:        true,
			wantDaysSince:   202,
		},
		{
			name:            "same client different referenceDate proves no time.Now usage",
			lastMeetingDate: time.Date(2025, 8, 12, 0, 0, 0, 0, time.UTC),
			referenceDate:   time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC),
			wantEmit:        true,
			wantDaysSince:   233, // 202 + 31 days
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := newMockEventBus()
			clients := newMockClientRepo()
			clients.AddClient("advisor1", Client{
				ID:              "c5",
				Name:            "Elena Vasquez",
				LastMeetingDate: tt.lastMeetingDate,
			})

			s := NewScanner(clients, newMockAccountRepo(), newMockRESPBenRepo(), newMockContribEngine(), bus)
			_, err := s.RunSweep(context.Background(), "advisor1", tt.referenceDate)
			if err != nil {
				t.Fatalf("RunSweep returned error: %v", err)
			}

			events := bus.EventsByType(EventEngagementStale)

			if tt.wantEmit {
				if len(events) == 0 {
					t.Fatal("expected EngagementStale event but none emitted")
				}
				evt := events[0]
				if evt.EntityType != EntityTypeClient {
					t.Errorf("EntityType = %q, want %q", evt.EntityType, EntityTypeClient)
				}

				var payload map[string]any
				json.Unmarshal(evt.Payload, &payload)
				if int(payload["days_since"].(float64)) != tt.wantDaysSince {
					t.Errorf("days_since = %v, want %d", payload["days_since"], tt.wantDaysSince)
				}
			} else {
				if len(events) > 0 {
					t.Errorf("expected no EngagementStale events but got %d", len(events))
				}
			}
		})
	}
}

// Test anchor 12: Internal account, balance $45200, idle 34 days → emit.
// Test anchor 13: External account, balance $50000, idle 60 days → no emit (external skipped).
// Test anchor 14: Internal account, balance $3000, idle 45 days → no emit (below min_balance).
func TestBalanceIdle(t *testing.T) {
	tests := []struct {
		name          string
		account       Account
		referenceDate time.Time
		wantEmit      bool
		wantIdleDays  int
	}{
		{
			name: "internal, high balance, idle 34 days",
			account: Account{
				ID:               "acc1",
				ClientID:         "c4",
				AccountType:      "NON_REG",
				Balance:          45200,
				IsExternal:       false,
				LastActivityDate: time.Date(2026, 1, 27, 0, 0, 0, 0, time.UTC),
			},
			referenceDate: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
			wantEmit:      true,
			wantIdleDays:  34,
		},
		{
			name: "external account skipped regardless of balance and idle time",
			account: Account{
				ID:               "acc2",
				ClientID:         "c4",
				AccountType:      "NON_REG",
				Balance:          50000,
				IsExternal:       true,
				LastActivityDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			referenceDate: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
			wantEmit:      false,
		},
		{
			name: "internal, balance below threshold",
			account: Account{
				ID:               "acc3",
				ClientID:         "c4",
				AccountType:      "NON_REG",
				Balance:          3000,
				IsExternal:       false,
				LastActivityDate: time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC),
			},
			referenceDate: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
			wantEmit:      false,
		},
		{
			name: "internal, high balance, recent activity (not idle enough)",
			account: Account{
				ID:               "acc4",
				ClientID:         "c4",
				AccountType:      "NON_REG",
				Balance:          10000,
				IsExternal:       false,
				LastActivityDate: time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
			},
			referenceDate: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
			wantEmit:      false, // 15 days < 30
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := newMockEventBus()
			clients := newMockClientRepo()
			clients.AddClient("advisor1", Client{ID: "c4", Name: "Rohan Gupta"})

			acctRepo := newMockAccountRepo()
			acctRepo.AddAccount(tt.account)

			s := NewScanner(clients, acctRepo, newMockRESPBenRepo(), newMockContribEngine(), bus)
			_, err := s.RunSweep(context.Background(), "advisor1", tt.referenceDate)
			if err != nil {
				t.Fatalf("RunSweep returned error: %v", err)
			}

			events := bus.EventsByType(EventCashUninvested)

			if tt.wantEmit {
				if len(events) == 0 {
					t.Fatal("expected CashUninvested event but none emitted")
				}
				evt := events[0]
				if evt.EntityType != EntityTypeAccount {
					t.Errorf("EntityType = %q, want %q", evt.EntityType, EntityTypeAccount)
				}
				if evt.EntityID != tt.account.ID {
					t.Errorf("EntityID = %q, want %q", evt.EntityID, tt.account.ID)
				}

				var payload map[string]any
				json.Unmarshal(evt.Payload, &payload)
				if int(payload["idle_days"].(float64)) != tt.wantIdleDays {
					t.Errorf("idle_days = %v, want %d", payload["idle_days"], tt.wantIdleDays)
				}
			} else {
				if len(events) > 0 {
					t.Errorf("expected no CashUninvested events but got %d", len(events))
				}
			}
		})
	}
}

// Test anchor 1: All 7 hardcoded rules are evaluated in a sweep.
func TestRunSweep_AllRulesEvaluated(t *testing.T) {
	bus := newMockEventBus()
	clients := newMockClientRepo()
	clients.AddClient("advisor1", Client{
		ID:              "c1",
		Name:            "Test Client",
		DateOfBirth:     time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		LastMeetingDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})

	s := NewScanner(clients, newMockAccountRepo(), newMockRESPBenRepo(), newMockContribEngine(), bus)
	result, err := s.RunSweep(context.Background(), "advisor1", time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("RunSweep returned error: %v", err)
	}

	if result.RulesEvaluated != 7 {
		t.Errorf("RulesEvaluated = %d, want 7", result.RulesEvaluated)
	}
}

// Test anchor 15: GetRoom error for one client during DEADLINE_WITH_ROOM,
// other clients still evaluated.
func TestRunSweep_ErrorContinues(t *testing.T) {
	bus := newMockEventBus()
	clients := newMockClientRepo()
	clients.AddClient("advisor1", Client{ID: "c1", Name: "Client One"})
	clients.AddClient("advisor1", Client{ID: "c2", Name: "Client Two"})

	acctRepo := newMockAccountRepo()
	acctRepo.AddAccount(Account{ID: "acc1", ClientID: "c1", AccountType: AccountTypeRRSP, Balance: 50000})
	acctRepo.AddAccount(Account{ID: "acc2", ClientID: "c2", AccountType: AccountTypeRRSP, Balance: 50000})

	contrib := newMockContribEngine()
	refDate := time.Date(2026, 2, 19, 0, 0, 0, 0, time.UTC)
	taxYear := refDate.Year() - 1 // RRSP deadline is for prev tax year
	contrib.SetError("c1", AccountTypeRRSP, taxYear, errors.New("database error"))
	contrib.SetRoom("c2", AccountTypeRRSP, taxYear, 5000)

	s := NewScanner(clients, acctRepo, newMockRESPBenRepo(), contrib, bus)
	result, err := s.RunSweep(context.Background(), "advisor1", refDate)
	if err != nil {
		t.Fatalf("RunSweep returned error: %v", err)
	}

	// c1 should be skipped (error), c2 should emit
	events := bus.EventsByType(EventDeadlineApproaching)
	var rrspEvents []EventEnvelope
	for _, e := range events {
		var p map[string]any
		json.Unmarshal(e.Payload, &p)
		if p["account_type"] == AccountTypeRRSP {
			rrspEvents = append(rrspEvents, e)
		}
	}

	if len(rrspEvents) != 1 {
		t.Fatalf("expected 1 RRSP DeadlineApproaching event, got %d", len(rrspEvents))
	}

	var payload map[string]any
	json.Unmarshal(rrspEvents[0].Payload, &payload)
	if payload["client_id"] != "c2" {
		t.Errorf("expected event for c2, got %v", payload["client_id"])
	}

	if result.RulesEvaluated != 7 {
		t.Errorf("RulesEvaluated = %d, want 7", result.RulesEvaluated)
	}
}
