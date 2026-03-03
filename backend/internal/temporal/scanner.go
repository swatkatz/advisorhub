package temporal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// scanner implements TemporalScanner.
type scanner struct {
	clients  ClientRepository
	accounts AccountRepository
	respBen  RESPBeneficiaryRepository
	contrib  ContributionEngine
	bus      EventBus
}

// NewScanner creates a new TemporalScanner.
func NewScanner(
	clients ClientRepository,
	accounts AccountRepository,
	respBen RESPBeneficiaryRepository,
	contrib ContributionEngine,
	bus EventBus,
) TemporalScanner {
	return &scanner{
		clients:  clients,
		accounts: accounts,
		respBen:  respBen,
		contrib:  contrib,
		bus:      bus,
	}
}

func (s *scanner) RunSweep(ctx context.Context, advisorID string, referenceDate time.Time) (*ScannerResult, error) {
	start := time.Now()

	result := &ScannerResult{}

	// Fetch all clients once, reuse across all rules.
	allClients, err := s.clients.GetClients(ctx, advisorID)
	if err != nil {
		return nil, fmt.Errorf("fetching clients: %w", err)
	}

	for _, rule := range Rules {
		result.RulesEvaluated++

		switch rule.CheckType {
		case CheckTypeAgeApproaching:
			emitted, checked := s.checkAgeApproaching(ctx, rule, allClients, referenceDate)
			result.EventsEmitted += emitted
			result.EntitiesChecked += checked

		case CheckTypeDeadlineWithRoom:
			emitted, checked := s.checkDeadlineWithRoom(ctx, rule, allClients, referenceDate)
			result.EventsEmitted += emitted
			result.EntitiesChecked += checked

		case CheckTypeDaysSince:
			emitted, checked := s.checkDaysSince(ctx, rule, allClients, referenceDate)
			result.EventsEmitted += emitted
			result.EntitiesChecked += checked

		case CheckTypeBalanceIdle:
			emitted, checked := s.checkBalanceIdle(ctx, rule, allClients, referenceDate)
			result.EventsEmitted += emitted
			result.EntitiesChecked += checked
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}

// checkAgeApproaching evaluates AGE_APPROACHING rules against clients or RESP beneficiaries.
// The check finds the year the entity turns targetAge and emits if that year
// is the current year or its start (Jan 1) is within withinDays of referenceDate.
func (s *scanner) checkAgeApproaching(ctx context.Context, rule TemporalRule, clients []Client, referenceDate time.Time) (emitted, checked int) {
	targetAge := intParam(rule.Params, "age")
	withinDays := intParam(rule.Params, "within_days")

	if rule.EntityType == EntityTypeClient {
		for _, c := range clients {
			checked++
			if yearTurning, ok := ageApproachingMatch(c.DateOfBirth, targetAge, withinDays, referenceDate); ok {
				payload := map[string]any{
					"client_id":    c.ID,
					"name":         c.Name,
					"rule":         rule.Name,
					"target_age":   targetAge,
					"current_age":  ageOn(c.DateOfBirth, referenceDate),
					"year_turning": yearTurning,
				}
				if err := s.publishEvent(ctx, rule.EventType, c.ID, EntityTypeClient, payload, referenceDate); err != nil {
					log.Printf("temporal: error publishing %s for client %s: %v", rule.EventType, c.ID, err)
				} else {
					emitted++
				}
			}
		}
	} else if rule.EntityType == EntityTypeRESPBeneficiary {
		for _, c := range clients {
			beneficiaries, err := s.respBen.GetRESPBeneficiariesByClientID(ctx, c.ID)
			if err != nil {
				log.Printf("temporal: error fetching beneficiaries for client %s: %v", c.ID, err)
				continue
			}
			for _, ben := range beneficiaries {
				checked++
				if yearTurning, ok := ageApproachingMatch(ben.DateOfBirth, targetAge, withinDays, referenceDate); ok {
					payload := map[string]any{
						"client_id":      c.ID,
						"beneficiary_id": ben.ID,
						"name":           ben.Name,
						"rule":           rule.Name,
						"target_age":     targetAge,
						"current_age":    ageOn(ben.DateOfBirth, referenceDate),
						"year_turning":   yearTurning,
					}
					if err := s.publishEvent(ctx, rule.EventType, ben.ID, EntityTypeRESPBeneficiary, payload, referenceDate); err != nil {
						log.Printf("temporal: error publishing %s for beneficiary %s: %v", rule.EventType, ben.ID, err)
					} else {
						emitted++
					}
				}
			}
		}
	}

	return emitted, checked
}

// ageApproachingMatch determines if an entity with the given DOB will reach targetAge
// within the horizon window. Returns (yearTurning, true) if the milestone year is
// the current year or its Jan 1 is within withinDays of referenceDate.
func ageApproachingMatch(dob time.Time, targetAge, withinDays int, referenceDate time.Time) (int, bool) {
	yearTurning := dob.Year() + targetAge

	if yearTurning < referenceDate.Year() {
		// Milestone year is in the past.
		return 0, false
	}

	if yearTurning == referenceDate.Year() {
		// We're in the milestone year — always within horizon.
		return yearTurning, true
	}

	// yearTurning > referenceDate.Year(): check if Jan 1 of that year is within horizon.
	yearStart := time.Date(yearTurning, 1, 1, 0, 0, 0, 0, time.UTC)
	daysUntil := int(yearStart.Sub(referenceDate).Hours() / 24)
	return yearTurning, daysUntil <= withinDays
}

// ageOn computes a person's age as of a given date.
func ageOn(dob time.Time, date time.Time) int {
	years := date.Year() - dob.Year()
	// If birthday hasn't occurred yet this year, subtract 1.
	birthdayThisYear := time.Date(date.Year(), dob.Month(), dob.Day(), 0, 0, 0, 0, time.UTC)
	if date.Before(birthdayThisYear) {
		years--
	}
	return years
}

// checkDeadlineWithRoom evaluates DEADLINE_WITH_ROOM rules.
// Groups accounts by (client_id, account_type) and evaluates once per client.
// Checks two tax years (current year - 1 and current year) to find approaching deadlines.
func (s *scanner) checkDeadlineWithRoom(ctx context.Context, rule TemporalRule, clients []Client, referenceDate time.Time) (emitted, checked int) {
	accountType := stringParam(rule.Params, "account_type")
	withinDays := intParam(rule.Params, "within_days")

	for _, c := range clients {
		checked++

		accounts, err := s.accounts.GetAccountsByClientID(ctx, c.ID)
		if err != nil {
			log.Printf("temporal: error fetching accounts for client %s: %v", c.ID, err)
			continue
		}

		// Check if client has any accounts of this type.
		hasType := false
		for _, a := range accounts {
			if a.AccountType == accountType {
				hasType = true
				break
			}
		}
		if !hasType {
			continue
		}

		// Check two tax years to find an approaching deadline.
		for _, taxYear := range []int{referenceDate.Year() - 1, referenceDate.Year()} {
			dl := deadline(accountType, taxYear)
			if dl == nil {
				continue
			}

			daysUntil := int(dl.Sub(referenceDate).Hours() / 24)
			if daysUntil < 0 || daysUntil > withinDays {
				continue
			}

			// Deadline is within window. Check room.
			room, err := s.contrib.GetRoom(ctx, c.ID, accountType, taxYear)
			if err != nil {
				log.Printf("temporal: error getting room for client %s %s year %d: %v", c.ID, accountType, taxYear, err)
				continue
			}
			if room <= 0 {
				continue
			}

			payload := map[string]any{
				"client_id":      c.ID,
				"account_type":   accountType,
				"deadline":       dl.Format("2006-01-02"),
				"days_until":     daysUntil,
				"room_remaining": room,
				"tax_year":       taxYear,
			}
			if err := s.publishEvent(ctx, rule.EventType, c.ID, EntityTypeClient, payload, referenceDate); err != nil {
				log.Printf("temporal: error publishing %s for client %s: %v", rule.EventType, c.ID, err)
			} else {
				emitted++
			}
			break // Only emit once per client per account type (first matching deadline)
		}
	}

	return emitted, checked
}

// checkDaysSince evaluates DAYS_SINCE rules against clients.
func (s *scanner) checkDaysSince(ctx context.Context, rule TemporalRule, clients []Client, referenceDate time.Time) (emitted, checked int) {
	field := stringParam(rule.Params, "field")
	threshold := intParam(rule.Params, "threshold")

	for _, c := range clients {
		checked++

		var fieldDate time.Time
		switch field {
		case "last_meeting_date":
			fieldDate = c.LastMeetingDate
		default:
			continue
		}

		if fieldDate.IsZero() {
			continue
		}

		daysSince := int(referenceDate.Sub(fieldDate).Hours() / 24)
		if daysSince > threshold {
			payload := map[string]any{
				"client_id":         c.ID,
				"last_meeting_date": fieldDate.Format("2006-01-02"),
				"days_since":        daysSince,
			}
			if err := s.publishEvent(ctx, rule.EventType, c.ID, EntityTypeClient, payload, referenceDate); err != nil {
				log.Printf("temporal: error publishing %s for client %s: %v", rule.EventType, c.ID, err)
			} else {
				emitted++
			}
		}
	}

	return emitted, checked
}

// checkBalanceIdle evaluates BALANCE_IDLE rules against accounts.
func (s *scanner) checkBalanceIdle(ctx context.Context, rule TemporalRule, clients []Client, referenceDate time.Time) (emitted, checked int) {
	minBalance := float64Param(rule.Params, "min_balance")
	idleDays := intParam(rule.Params, "idle_days")

	for _, c := range clients {
		accounts, err := s.accounts.GetAccountsByClientID(ctx, c.ID)
		if err != nil {
			log.Printf("temporal: error fetching accounts for client %s: %v", c.ID, err)
			continue
		}

		for _, a := range accounts {
			checked++

			if a.IsExternal {
				continue
			}

			if a.Balance < minBalance {
				continue
			}

			days := int(referenceDate.Sub(a.LastActivityDate).Hours() / 24)
			if days > idleDays {
				payload := map[string]any{
					"client_id":    c.ID,
					"account_id":   a.ID,
					"account_type": a.AccountType,
					"balance":      a.Balance,
					"idle_days":    days,
				}
				if err := s.publishEvent(ctx, rule.EventType, a.ID, EntityTypeAccount, payload, referenceDate); err != nil {
					log.Printf("temporal: error publishing %s for account %s: %v", rule.EventType, a.ID, err)
				} else {
					emitted++
				}
			}
		}
	}

	return emitted, checked
}

func (s *scanner) publishEvent(ctx context.Context, eventType, entityID, entityType string, payload any, referenceDate time.Time) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling event payload: %w", err)
	}
	return s.bus.Publish(ctx, EventEnvelope{
		ID:         fmt.Sprintf("evt_%s_%s_%d", eventType, entityID, referenceDate.UnixNano()),
		Type:       eventType,
		EntityID:   entityID,
		EntityType: entityType,
		Payload:    data,
		Source:     "TEMPORAL",
		Timestamp:  referenceDate,
	})
}

// intParam extracts an int parameter from the params map.
func intParam(params map[string]any, key string) int {
	v, ok := params[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	default:
		return 0
	}
}

// float64Param extracts a float64 parameter from the params map.
func float64Param(params map[string]any, key string) float64 {
	v, ok := params[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	default:
		return 0
	}
}

// stringParam extracts a string parameter from the params map.
func stringParam(params map[string]any, key string) string {
	v, ok := params[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}
