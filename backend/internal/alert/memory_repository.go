package alert

import (
	"context"
	"sort"
	"sync"
)

// MemoryAlertRepository is an in-memory AlertRepository for testing.
type MemoryAlertRepository struct {
	mu     sync.RWMutex
	alerts map[string]*Alert
}

func NewMemoryAlertRepository() *MemoryAlertRepository {
	return &MemoryAlertRepository{
		alerts: make(map[string]*Alert),
	}
}

func (r *MemoryAlertRepository) FindByConditionKey(_ context.Context, conditionKey string) (*Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var match *Alert
	for _, a := range r.alerts {
		if a.ConditionKey == conditionKey && a.Status != StatusClosed {
			if match == nil || a.CreatedAt.After(match.CreatedAt) {
				match = a
			}
		}
	}
	return match, nil
}

func (r *MemoryAlertRepository) GetAlert(_ context.Context, id string) (*Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	a, ok := r.alerts[id]
	if !ok {
		return nil, ErrAlertNotFound
	}
	return a, nil
}

func (r *MemoryAlertRepository) GetAlertsByClientID(_ context.Context, clientID string) ([]Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Alert
	for _, a := range r.alerts {
		if a.ClientID == clientID {
			result = append(result, *a)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (r *MemoryAlertRepository) GetAlertsByAdvisorID(_ context.Context, _ string) ([]Alert, error) {
	// Not used in unit tests — requires join through client table.
	return nil, nil
}

func (r *MemoryAlertRepository) CreateAlert(_ context.Context, alert *Alert) (*Alert, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.alerts[alert.ID] = alert
	return alert, nil
}

func (r *MemoryAlertRepository) UpdateAlert(_ context.Context, alert *Alert) (*Alert, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.alerts[alert.ID] = alert
	return alert, nil
}
