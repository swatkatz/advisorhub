package actionitem

import (
	"context"
	"fmt"
	"sort"
)

// MemoryActionItemRepo is an in-memory implementation of ActionItemRepository for testing.
type MemoryActionItemRepo struct {
	items map[string]ActionItem
}

func newMemoryActionItemRepo() *MemoryActionItemRepo {
	return &MemoryActionItemRepo{items: make(map[string]ActionItem)}
}

func (r *MemoryActionItemRepo) CreateActionItem(_ context.Context, item *ActionItem) (*ActionItem, error) {
	r.items[item.ID] = *item
	result := *item
	return &result, nil
}

func (r *MemoryActionItemRepo) GetActionItem(_ context.Context, id string) (*ActionItem, error) {
	item, ok := r.items[id]
	if !ok {
		return nil, fmt.Errorf("getting action item: not found: %s", id)
	}
	result := item
	return &result, nil
}

func (r *MemoryActionItemRepo) GetActionItemsByClientID(_ context.Context, clientID string) ([]ActionItem, error) {
	var result []ActionItem
	for _, item := range r.items {
		if item.ClientID == clientID {
			result = append(result, item)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	if result == nil {
		result = []ActionItem{}
	}
	return result, nil
}

func (r *MemoryActionItemRepo) GetActionItemsByAlertID(_ context.Context, alertID string) ([]ActionItem, error) {
	var result []ActionItem
	for _, item := range r.items {
		if item.AlertID != nil && *item.AlertID == alertID {
			result = append(result, item)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	if result == nil {
		result = []ActionItem{}
	}
	return result, nil
}

func (r *MemoryActionItemRepo) UpdateActionItem(_ context.Context, item *ActionItem) (*ActionItem, error) {
	if _, ok := r.items[item.ID]; !ok {
		return nil, fmt.Errorf("updating action item: not found: %s", item.ID)
	}
	r.items[item.ID] = *item
	result := *item
	return &result, nil
}
