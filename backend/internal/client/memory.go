package client

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// MemoryClientRepo is an in-memory implementation of ClientRepository for testing.
type MemoryClientRepo struct {
	clients map[string]Client
}

func newMemoryClientRepo() *MemoryClientRepo {
	return &MemoryClientRepo{clients: make(map[string]Client)}
}

func (r *MemoryClientRepo) GetClient(_ context.Context, id string) (*Client, error) {
	c, ok := r.clients[id]
	if !ok {
		return nil, fmt.Errorf("getting client: not found: %s", id)
	}
	return &c, nil
}

func (r *MemoryClientRepo) GetClients(_ context.Context, advisorID string) ([]Client, error) {
	var result []Client
	for _, c := range r.clients {
		if c.AdvisorID == advisorID {
			result = append(result, c)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (r *MemoryClientRepo) GetClientsByHouseholdID(_ context.Context, householdID string) ([]Client, error) {
	var result []Client
	for _, c := range r.clients {
		if c.HouseholdID != nil && *c.HouseholdID == householdID {
			result = append(result, c)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

// MemoryHouseholdRepo is an in-memory implementation of HouseholdRepository for testing.
type MemoryHouseholdRepo struct {
	households map[string]Household
	clientRepo *MemoryClientRepo
}

func newMemoryHouseholdRepo() *MemoryHouseholdRepo {
	return &MemoryHouseholdRepo{
		households: make(map[string]Household),
	}
}

func newMemoryHouseholdRepoWithClients(clientRepo *MemoryClientRepo) *MemoryHouseholdRepo {
	return &MemoryHouseholdRepo{
		households: make(map[string]Household),
		clientRepo: clientRepo,
	}
}

func (r *MemoryHouseholdRepo) GetHousehold(_ context.Context, id string) (*Household, error) {
	h, ok := r.households[id]
	if !ok {
		return nil, fmt.Errorf("getting household: not found: %s", id)
	}
	return &h, nil
}

func (r *MemoryHouseholdRepo) GetHouseholdByClientID(ctx context.Context, clientID string) (*Household, error) {
	if r.clientRepo == nil {
		return nil, fmt.Errorf("getting household by client: no client repo configured")
	}
	c, err := r.clientRepo.GetClient(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("getting household by client: %w", err)
	}
	if c.HouseholdID == nil {
		return nil, nil
	}
	h, ok := r.households[*c.HouseholdID]
	if !ok {
		return nil, fmt.Errorf("getting household by client: household not found: %s", *c.HouseholdID)
	}
	return &h, nil
}

// MemoryGoalRepo is an in-memory implementation of GoalRepository for testing.
type MemoryGoalRepo struct {
	goals      map[string]Goal
	clientRepo *MemoryClientRepo
}

func newMemoryGoalRepo() *MemoryGoalRepo {
	return &MemoryGoalRepo{goals: make(map[string]Goal)}
}

func (r *MemoryGoalRepo) GetGoalsByClientID(ctx context.Context, clientID string) ([]Goal, error) {
	var result []Goal

	// Get the client's household ID so we can include household-level goals
	var householdID *string
	if r.clientRepo != nil {
		c, err := r.clientRepo.GetClient(ctx, clientID)
		if err == nil && c.HouseholdID != nil {
			householdID = c.HouseholdID
		}
	}

	for _, g := range r.goals {
		if g.ClientID == clientID {
			result = append(result, g)
		} else if householdID != nil && g.HouseholdID != nil && *g.HouseholdID == *householdID {
			result = append(result, g)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

// MemoryNoteRepo is an in-memory implementation of AdvisorNoteRepository for testing.
type MemoryNoteRepo struct {
	notes  []AdvisorNote
	nextID int
}

func newMemoryNoteRepo() *MemoryNoteRepo {
	return &MemoryNoteRepo{nextID: 1}
}

func (r *MemoryNoteRepo) GetNotes(_ context.Context, clientID string, advisorID string) ([]AdvisorNote, error) {
	var result []AdvisorNote
	for _, n := range r.notes {
		if n.ClientID == clientID && n.AdvisorID == advisorID {
			result = append(result, n)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Date.After(result[j].Date) })
	if result == nil {
		result = []AdvisorNote{}
	}
	return result, nil
}

func (r *MemoryNoteRepo) AddNote(_ context.Context, clientID string, advisorID string, text string) (*AdvisorNote, error) {
	note := AdvisorNote{
		ID:        fmt.Sprintf("n%d", r.nextID),
		ClientID:  clientID,
		AdvisorID: advisorID,
		Date:      time.Now(),
		Text:      text,
	}
	r.nextID++
	r.notes = append(r.notes, note)
	return &note, nil
}

// MemoryAdvisorRepo is an in-memory implementation of AdvisorRepository for testing.
type MemoryAdvisorRepo struct {
	advisors map[string]Advisor
}

func newMemoryAdvisorRepo() *MemoryAdvisorRepo {
	return &MemoryAdvisorRepo{advisors: make(map[string]Advisor)}
}

func (r *MemoryAdvisorRepo) GetAdvisor(_ context.Context, id string) (*Advisor, error) {
	a, ok := r.advisors[id]
	if !ok {
		return nil, fmt.Errorf("getting advisor: not found: %s", id)
	}
	return &a, nil
}
