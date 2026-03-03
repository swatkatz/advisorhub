package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/swatkatz/advisorhub/backend/internal/actionitem"
	"github.com/swatkatz/advisorhub/backend/internal/eventbus"
)

var alertIDCounter uint64

func generateAlertID() string {
	return fmt.Sprintf("alert_%d", atomic.AddUint64(&alertIDCounter, 1))
}

type alertService struct {
	repo        AlertRepository
	bus         eventbus.EventBus
	actionItems actionitem.ActionItemService
	enhancer    Enhancer
	now         func() time.Time
}

// NewAlertService creates a new AlertService.
func NewAlertService(
	repo AlertRepository,
	bus eventbus.EventBus,
	actionItems actionitem.ActionItemService,
	enhancer Enhancer,
	now func() time.Time,
) AlertService {
	return &alertService{
		repo:        repo,
		bus:         bus,
		actionItems: actionItems,
		enhancer:    enhancer,
		now:         now,
	}
}

func (s *alertService) ProcessEvent(ctx context.Context, envelope eventbus.EventEnvelope) error {
	rule, ok := CategoryRules[envelope.Type]
	if !ok {
		log.Printf("alert: no category rule for event type %s, discarding", envelope.Type)
		return nil
	}

	var payload map[string]any
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return fmt.Errorf("parsing event payload: %w", err)
	}

	conditionKey := rule.BuildConditionKey(payload, envelope)
	clientID := rule.ExtractClientID(payload, envelope)

	existing, err := s.repo.FindByConditionKey(ctx, conditionKey)
	if err != nil {
		return fmt.Errorf("finding alert by condition key: %w", err)
	}

	now := s.now()
	var alert *Alert
	var signal ProcessSignal

	if existing == nil {
		alert = &Alert{
			ID:                  generateAlertID(),
			ConditionKey:        conditionKey,
			ClientID:            clientID,
			Severity:            rule.Severity,
			Category:            rule.Category,
			Status:              StatusOpen,
			Payload:             envelope.Payload,
			LinkedActionItemIDs: []string{},
			CreatedAt:           now,
			UpdatedAt:           now,
		}
		alert, err = s.repo.CreateAlert(ctx, alert)
		if err != nil {
			return fmt.Errorf("creating alert: %w", err)
		}
		signal = SignalCreated
	} else {
		alert = existing
		alert.Payload = envelope.Payload
		alert.UpdatedAt = now

		switch alert.Status {
		case StatusOpen:
			signal = SignalUpdated
		case StatusSnoozed:
			if alert.SnoozedUntil != nil && now.After(*alert.SnoozedUntil) {
				alert.Status = StatusOpen
				alert.SnoozedUntil = nil
				signal = SignalReopened
			} else {
				signal = SignalUpdated
			}
		case StatusActed:
			signal = SignalUpdated
		}

		alert, err = s.repo.UpdateAlert(ctx, alert)
		if err != nil {
			return fmt.Errorf("updating alert: %w", err)
		}
	}

	// Enhance on CREATED or REOPENED
	if signal == SignalCreated || signal == SignalReopened {
		if enhanceErr := s.enhancer.Enhance(ctx, alert); enhanceErr != nil {
			log.Printf("alert: enhancer failed for alert %s: %v", alert.ID, enhanceErr)
		} else {
			if _, updateErr := s.repo.UpdateAlert(ctx, alert); updateErr != nil {
				log.Printf("alert: failed to persist enhancement for alert %s: %v", alert.ID, updateErr)
			}
		}
	}

	// Emit events
	switch signal {
	case SignalCreated:
		s.emitAlertCreated(ctx, alert)
	case SignalReopened:
		s.emitAlertUpdated(ctx, alert, "REOPENED")
	case SignalUpdated:
		if alert.Status == StatusOpen {
			s.emitAlertUpdated(ctx, alert, "PAYLOAD_UPDATED")
		}
	}

	return nil
}

func (s *alertService) Send(ctx context.Context, alertID string, message *string) (*Alert, error) {
	alert, err := s.repo.GetAlert(ctx, alertID)
	if err != nil {
		return nil, err
	}
	if alert.Status == StatusClosed {
		return nil, ErrAlertClosed
	}
	if alert.Severity == SeverityInfo {
		return nil, ErrInfoAlert
	}

	// Determine message text
	msgText := ""
	if message != nil {
		msgText = *message
	} else if alert.DraftMessage != nil {
		msgText = *alert.DraftMessage
	}

	// Create linked ActionItem
	ai, err := s.actionItems.CreateActionItem(ctx, alert.ClientID, &alertID, msgText, nil)
	if err != nil {
		return nil, fmt.Errorf("creating action item: %w", err)
	}
	alert.LinkedActionItemIDs = append(alert.LinkedActionItemIDs, ai.ID)

	// Transition to SNOOZED (via ACTED → auto-snooze)
	alert.Status = StatusSnoozed
	snoozedUntil := s.now().Add(GetAutoSnoozeDuration(alert.Category))
	alert.SnoozedUntil = &snoozedUntil
	alert.UpdatedAt = s.now()

	alert, err = s.repo.UpdateAlert(ctx, alert)
	if err != nil {
		return nil, fmt.Errorf("updating alert: %w", err)
	}

	s.emitAlertUpdated(ctx, alert, "STATUS_CHANGED")
	return alert, nil
}

func (s *alertService) Track(ctx context.Context, alertID string, actionItemText string) (*Alert, error) {
	alert, err := s.repo.GetAlert(ctx, alertID)
	if err != nil {
		return nil, err
	}
	if alert.Status == StatusClosed {
		return nil, ErrAlertClosed
	}
	if alert.Severity == SeverityInfo {
		return nil, ErrInfoAlert
	}

	ai, err := s.actionItems.CreateActionItem(ctx, alert.ClientID, &alertID, actionItemText, nil)
	if err != nil {
		return nil, fmt.Errorf("creating action item: %w", err)
	}
	alert.LinkedActionItemIDs = append(alert.LinkedActionItemIDs, ai.ID)

	// Transition to SNOOZED (via ACTED → auto-snooze)
	alert.Status = StatusSnoozed
	snoozedUntil := s.now().Add(GetAutoSnoozeDuration(alert.Category))
	alert.SnoozedUntil = &snoozedUntil
	alert.UpdatedAt = s.now()

	alert, err = s.repo.UpdateAlert(ctx, alert)
	if err != nil {
		return nil, fmt.Errorf("updating alert: %w", err)
	}

	s.emitAlertUpdated(ctx, alert, "STATUS_CHANGED")
	return alert, nil
}

func (s *alertService) Snooze(ctx context.Context, alertID string, until *time.Time) (*Alert, error) {
	alert, err := s.repo.GetAlert(ctx, alertID)
	if err != nil {
		return nil, err
	}
	if alert.Status == StatusClosed {
		return nil, ErrAlertClosed
	}
	if alert.Severity == SeverityInfo {
		return nil, ErrInfoAlert
	}

	alert.Status = StatusSnoozed
	if until != nil {
		alert.SnoozedUntil = until
	} else {
		snoozedUntil := s.now().Add(GetAutoSnoozeDuration(alert.Category))
		alert.SnoozedUntil = &snoozedUntil
	}
	alert.UpdatedAt = s.now()

	alert, err = s.repo.UpdateAlert(ctx, alert)
	if err != nil {
		return nil, fmt.Errorf("updating alert: %w", err)
	}

	s.emitAlertUpdated(ctx, alert, "STATUS_CHANGED")
	return alert, nil
}

func (s *alertService) Acknowledge(ctx context.Context, alertID string) (*Alert, error) {
	alert, err := s.repo.GetAlert(ctx, alertID)
	if err != nil {
		return nil, err
	}
	if alert.Severity != SeverityInfo {
		return nil, ErrNotInfoAlert
	}
	if alert.Status == StatusClosed {
		return nil, ErrAlertClosed
	}

	alert.Status = StatusClosed
	now := s.now()
	alert.ResolvedAt = &now
	alert.UpdatedAt = now

	alert, err = s.repo.UpdateAlert(ctx, alert)
	if err != nil {
		return nil, fmt.Errorf("updating alert: %w", err)
	}

	s.emitAlertClosed(ctx, alert)
	return alert, nil
}

func (s *alertService) Close(ctx context.Context, alertID string) (*Alert, error) {
	alert, err := s.repo.GetAlert(ctx, alertID)
	if err != nil {
		return nil, err
	}

	// Idempotent: already closed → no-op
	if alert.Status == StatusClosed {
		return alert, nil
	}

	alert.Status = StatusClosed
	now := s.now()
	alert.ResolvedAt = &now
	alert.UpdatedAt = now

	// Cascade close linked ActionItems
	for _, aiID := range alert.LinkedActionItemIDs {
		resolutionNote := fmt.Sprintf("Auto-closed: %s condition resolved on %s",
			alert.Category, now.Format("2006-01-02"))
		if _, closeErr := s.actionItems.CloseActionItem(ctx, aiID, resolutionNote); closeErr != nil {
			log.Printf("alert: failed to cascade close action item %s: %v", aiID, closeErr)
		}
	}

	alert, err = s.repo.UpdateAlert(ctx, alert)
	if err != nil {
		return nil, fmt.Errorf("updating alert: %w", err)
	}

	s.emitAlertClosed(ctx, alert)
	return alert, nil
}

func (s *alertService) ComputeHealthStatus(ctx context.Context, clientID string) (HealthStatus, error) {
	alerts, err := s.repo.GetAlertsByClientID(ctx, clientID)
	if err != nil {
		return HealthGreen, fmt.Errorf("getting alerts for client: %w", err)
	}

	var mostSevere AlertSeverity
	for _, a := range alerts {
		if a.Status == StatusClosed {
			continue
		}
		if mostSevere == "" || severityRank(a.Severity) > severityRank(mostSevere) {
			mostSevere = a.Severity
		}
	}

	switch mostSevere {
	case SeverityCritical:
		return HealthRed, nil
	case SeverityUrgent, SeverityAdvisory:
		return HealthYellow, nil
	default:
		return HealthGreen, nil
	}
}

func severityRank(s AlertSeverity) int {
	switch s {
	case SeverityCritical:
		return 4
	case SeverityUrgent:
		return 3
	case SeverityAdvisory:
		return 2
	case SeverityInfo:
		return 1
	default:
		return 0
	}
}

func (s *alertService) emitAlertCreated(ctx context.Context, a *Alert) {
	p := map[string]any{
		"alert_id":      a.ID,
		"client_id":     a.ClientID,
		"severity":      string(a.Severity),
		"category":      a.Category,
		"condition_key": a.ConditionKey,
		"summary":       a.Summary,
	}
	if a.DraftMessage != nil {
		p["draft_message"] = *a.DraftMessage
	}
	payload, _ := json.Marshal(p)
	s.bus.Publish(ctx, eventbus.EventEnvelope{
		ID:         fmt.Sprintf("evt_%s_%s_%d", EventAlertCreated, a.ID, s.now().UnixNano()),
		Type:       EventAlertCreated,
		EntityID:   a.ID,
		EntityType: eventbus.EntityTypeClient,
		Payload:    payload,
		Source:     eventbus.SourceSystem,
		Timestamp:  s.now(),
	})
}

func (s *alertService) emitAlertUpdated(ctx context.Context, a *Alert, updateType string) {
	payload, _ := json.Marshal(map[string]any{
		"alert_id":    a.ID,
		"client_id":   a.ClientID,
		"update_type": updateType,
		"status":      string(a.Status),
		"severity":    string(a.Severity),
		"category":    a.Category,
	})
	s.bus.Publish(ctx, eventbus.EventEnvelope{
		ID:         fmt.Sprintf("evt_%s_%s_%d", EventAlertUpdated, a.ID, s.now().UnixNano()),
		Type:       EventAlertUpdated,
		EntityID:   a.ID,
		EntityType: eventbus.EntityTypeClient,
		Payload:    payload,
		Source:     eventbus.SourceSystem,
		Timestamp:  s.now(),
	})
}

func (s *alertService) emitAlertClosed(ctx context.Context, a *Alert) {
	payload, _ := json.Marshal(map[string]any{
		"alert_id":  a.ID,
		"client_id": a.ClientID,
		"category":  a.Category,
		"summary":   fmt.Sprintf("Resolved: %s", a.Summary),
	})
	s.bus.Publish(ctx, eventbus.EventEnvelope{
		ID:         fmt.Sprintf("evt_%s_%s_%d", EventAlertClosed, a.ID, s.now().UnixNano()),
		Type:       EventAlertClosed,
		EntityID:   a.ID,
		EntityType: eventbus.EntityTypeClient,
		Payload:    payload,
		Source:     eventbus.SourceSystem,
		Timestamp:  s.now(),
	})
}
