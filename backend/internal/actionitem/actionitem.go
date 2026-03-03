package actionitem

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// ActionItemStatus represents the status of an action item.
type ActionItemStatus string

const (
	ActionItemStatusPending    ActionItemStatus = "PENDING"
	ActionItemStatusInProgress ActionItemStatus = "IN_PROGRESS"
	ActionItemStatusDone       ActionItemStatus = "DONE"
	ActionItemStatusClosed     ActionItemStatus = "CLOSED"
)

// ActionItem is a tracked task shared between advisor and client.
type ActionItem struct {
	ID             string
	ClientID       string
	AlertID        *string // nullable — nil for manually created action items
	Text           string
	Status         ActionItemStatus
	DueDate        *time.Time
	CreatedAt      time.Time
	ResolvedAt     *time.Time
	ResolutionNote *string
}

// ActionItemRepository provides data access for action items.
type ActionItemRepository interface {
	CreateActionItem(ctx context.Context, item *ActionItem) (*ActionItem, error)
	GetActionItem(ctx context.Context, id string) (*ActionItem, error)
	GetActionItemsByClientID(ctx context.Context, clientID string) ([]ActionItem, error)
	GetActionItemsByAlertID(ctx context.Context, alertID string) ([]ActionItem, error)
	UpdateActionItem(ctx context.Context, item *ActionItem) (*ActionItem, error)
}

// ActionItemService wraps ActionItemRepository and adds status transition validation.
type ActionItemService interface {
	CreateActionItem(ctx context.Context, clientID string, alertID *string, text string, dueDate *time.Time) (*ActionItem, error)
	GetActionItem(ctx context.Context, id string) (*ActionItem, error)
	GetActionItemsByClientID(ctx context.Context, clientID string) ([]ActionItem, error)
	GetActionItemsByAlertID(ctx context.Context, alertID string) ([]ActionItem, error)
	UpdateActionItem(ctx context.Context, id string, text *string, status *ActionItemStatus, dueDate *time.Time) (*ActionItem, error)
	CloseActionItem(ctx context.Context, id string, resolutionNote string) (*ActionItem, error)
}

// validTransitions defines valid status transitions via UpdateActionItem.
// CLOSED is not a valid target via UpdateActionItem (only via CloseActionItem).
var validTransitions = map[ActionItemStatus]map[ActionItemStatus]bool{
	ActionItemStatusPending: {
		ActionItemStatusInProgress: true,
		ActionItemStatusDone:       true,
	},
	ActionItemStatusInProgress: {
		ActionItemStatusDone: true,
	},
	// DONE and CLOSED have no valid transitions via UpdateActionItem.
}

// idCounter is used to generate unique IDs.
var idCounter atomic.Int64

// Service implements ActionItemService.
type Service struct {
	repo  ActionItemRepository
	clock func() time.Time
}

// NewService creates a new ActionItem service.
func NewService(repo ActionItemRepository) *Service {
	return &Service{
		repo:  repo,
		clock: func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) CreateActionItem(ctx context.Context, clientID string, alertID *string, text string, dueDate *time.Time) (*ActionItem, error) {
	item := &ActionItem{
		ID:        fmt.Sprintf("ai_%d", idCounter.Add(1)),
		ClientID:  clientID,
		AlertID:   alertID,
		Text:      text,
		Status:    ActionItemStatusPending,
		DueDate:   dueDate,
		CreatedAt: s.clock(),
	}
	return s.repo.CreateActionItem(ctx, item)
}

func (s *Service) GetActionItem(ctx context.Context, id string) (*ActionItem, error) {
	return s.repo.GetActionItem(ctx, id)
}

func (s *Service) GetActionItemsByClientID(ctx context.Context, clientID string) ([]ActionItem, error) {
	return s.repo.GetActionItemsByClientID(ctx, clientID)
}

func (s *Service) GetActionItemsByAlertID(ctx context.Context, alertID string) ([]ActionItem, error) {
	return s.repo.GetActionItemsByAlertID(ctx, alertID)
}

func (s *Service) UpdateActionItem(ctx context.Context, id string, text *string, status *ActionItemStatus, dueDate *time.Time) (*ActionItem, error) {
	item, err := s.repo.GetActionItem(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("updating action item: %w", err)
	}

	// CLOSED items cannot be updated at all.
	if item.Status == ActionItemStatusClosed {
		return nil, fmt.Errorf("updating action item: item %s is CLOSED", id)
	}

	// Validate status transition if status is being changed.
	if status != nil {
		// CLOSED cannot be set via UpdateActionItem.
		if *status == ActionItemStatusClosed {
			return nil, fmt.Errorf("updating action item: CLOSED can only be set via CloseActionItem")
		}

		allowed, ok := validTransitions[item.Status]
		if !ok || !allowed[*status] {
			return nil, fmt.Errorf("updating action item: invalid transition from %s to %s", item.Status, *status)
		}

		item.Status = *status

		// Set resolved_at when transitioning to DONE.
		if *status == ActionItemStatusDone {
			now := s.clock()
			item.ResolvedAt = &now
		}
	}

	if text != nil {
		item.Text = *text
	}

	if dueDate != nil {
		item.DueDate = dueDate
	}

	return s.repo.UpdateActionItem(ctx, item)
}

func (s *Service) CloseActionItem(ctx context.Context, id string, resolutionNote string) (*ActionItem, error) {
	item, err := s.repo.GetActionItem(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("closing action item: %w", err)
	}

	// Idempotent: already CLOSED is a no-op.
	if item.Status == ActionItemStatusClosed {
		return item, nil
	}

	item.Status = ActionItemStatusClosed
	now := s.clock()
	item.ResolvedAt = &now
	item.ResolutionNote = &resolutionNote

	return s.repo.UpdateActionItem(ctx, item)
}
