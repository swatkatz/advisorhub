package client

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"
)

// --- In-memory implementations for testing ---

type memClientRepo struct {
	clients []Client
}

func (r *memClientRepo) GetClient(_ context.Context, id string) (*Client, error) {
	for i := range r.clients {
		if r.clients[i].ID == id {
			return &r.clients[i], nil
		}
	}
	return nil, fmt.Errorf("getting client: not found: %s", id)
}

func (r *memClientRepo) GetClients(_ context.Context, advisorID string) ([]Client, error) {
	var result []Client
	for _, c := range r.clients {
		if c.AdvisorID == advisorID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (r *memClientRepo) GetClientsByHouseholdID(_ context.Context, householdID string) ([]Client, error) {
	var result []Client
	for _, c := range r.clients {
		if c.HouseholdID != nil && *c.HouseholdID == householdID {
			result = append(result, c)
		}
	}
	return result, nil
}

type memHouseholdRepo struct {
	households []Household
	clients    []Client // needed to look up client -> household
}

func (r *memHouseholdRepo) GetHousehold(_ context.Context, id string) (*Household, error) {
	for i := range r.households {
		if r.households[i].ID == id {
			return &r.households[i], nil
		}
	}
	return nil, fmt.Errorf("getting household: not found: %s", id)
}

func (r *memHouseholdRepo) GetHouseholdByClientID(_ context.Context, clientID string) (*Household, error) {
	// Find the client
	var householdID *string
	for _, c := range r.clients {
		if c.ID == clientID {
			householdID = c.HouseholdID
			break
		}
	}
	if householdID == nil {
		return nil, nil
	}
	for i := range r.households {
		if r.households[i].ID == *householdID {
			return &r.households[i], nil
		}
	}
	return nil, nil
}

type memGoalRepo struct {
	goals   []Goal
	clients []Client // needed to resolve household membership
}

func (r *memGoalRepo) GetGoalsByClientID(_ context.Context, clientID string) ([]Goal, error) {
	// Find the client's household ID
	var householdID *string
	for _, c := range r.clients {
		if c.ID == clientID {
			householdID = c.HouseholdID
			break
		}
	}

	var result []Goal
	for _, g := range r.goals {
		if g.ClientID == clientID {
			result = append(result, g)
		} else if householdID != nil && g.HouseholdID != nil && *g.HouseholdID == *householdID {
			// Include household-level goals for other members
			result = append(result, g)
		}
	}
	return result, nil
}

type memAdvisorNoteRepo struct {
	mu    sync.Mutex
	notes []AdvisorNote
	now   func() time.Time
}

func (r *memAdvisorNoteRepo) GetNotes(_ context.Context, clientID string, advisorID string) ([]AdvisorNote, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := []AdvisorNote{}
	for _, n := range r.notes {
		if n.ClientID == clientID && n.AdvisorID == advisorID {
			result = append(result, n)
		}
	}
	// Sort by date descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].Date.After(result[j].Date)
	})
	return result, nil
}

func (r *memAdvisorNoteRepo) AddNote(_ context.Context, clientID string, advisorID string, text string) (*AdvisorNote, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if r.now != nil {
		now = r.now()
	}
	note := AdvisorNote{
		ID:        fmt.Sprintf("note_%d", len(r.notes)+1),
		ClientID:  clientID,
		AdvisorID: advisorID,
		Date:      now,
		Text:      text,
	}
	r.notes = append(r.notes, note)
	return &note, nil
}

type memAdvisorRepo struct {
	advisors []Advisor
}

func (r *memAdvisorRepo) GetAdvisor(_ context.Context, id string) (*Advisor, error) {
	for i := range r.advisors {
		if r.advisors[i].ID == id {
			return &r.advisors[i], nil
		}
	}
	return nil, fmt.Errorf("getting advisor: not found: %s", id)
}

// --- Tests ---

func TestGetClients(t *testing.T) {
	repo := &memClientRepo{
		clients: []Client{
			{ID: "c1", AdvisorID: "a1", Name: "Priya Sharma"},
			{ID: "c2", AdvisorID: "a1", Name: "Marcus Chen"},
			{ID: "c3", AdvisorID: "a1", Name: "Swati Gupta"},
			{ID: "c4", AdvisorID: "a2", Name: "Other Advisor Client"},
		},
	}

	clients, err := repo.GetClients(context.Background(), "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clients) != 3 {
		t.Fatalf("expected 3 clients, got %d", len(clients))
	}
	for _, c := range clients {
		if c.AdvisorID != "a1" {
			t.Errorf("expected advisor_id a1, got %s", c.AdvisorID)
		}
	}
}

func TestGetClient(t *testing.T) {
	dob := time.Date(1988, 3, 15, 0, 0, 0, 0, time.UTC)
	lastMeeting := time.Date(2025, 12, 14, 0, 0, 0, 0, time.UTC)

	repo := &memClientRepo{
		clients: []Client{
			{
				ID:              "c1",
				AdvisorID:       "a1",
				HouseholdID:     nil,
				Name:            "Priya Sharma",
				Email:           "priya@example.com",
				DateOfBirth:     dob,
				LastMeetingDate: lastMeeting,
			},
		},
	}

	t.Run("valid ID returns client with all fields", func(t *testing.T) {
		c, err := repo.GetClient(context.Background(), "c1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.ID != "c1" {
			t.Errorf("expected ID c1, got %s", c.ID)
		}
		if c.Name != "Priya Sharma" {
			t.Errorf("expected name Priya Sharma, got %s", c.Name)
		}
		if c.Email != "priya@example.com" {
			t.Errorf("expected email priya@example.com, got %s", c.Email)
		}
		if !c.DateOfBirth.Equal(dob) {
			t.Errorf("expected DOB %v, got %v", dob, c.DateOfBirth)
		}
		if !c.LastMeetingDate.Equal(lastMeeting) {
			t.Errorf("expected last meeting %v, got %v", lastMeeting, c.LastMeetingDate)
		}
		if c.HouseholdID != nil {
			t.Errorf("expected nil household_id, got %v", c.HouseholdID)
		}
	})

	t.Run("invalid ID returns error", func(t *testing.T) {
		_, err := repo.GetClient(context.Background(), "nonexistent")
		if err == nil {
			t.Fatal("expected error for invalid ID, got nil")
		}
	})
}

func TestGetClientsByHouseholdID(t *testing.T) {
	hhID := "hh1"
	repo := &memClientRepo{
		clients: []Client{
			{ID: "c3", AdvisorID: "a1", HouseholdID: &hhID, Name: "Swati Gupta"},
			{ID: "c4", AdvisorID: "a1", HouseholdID: &hhID, Name: "Rohan Gupta"},
			{ID: "c1", AdvisorID: "a1", HouseholdID: nil, Name: "Priya Sharma"},
		},
	}

	clients, err := repo.GetClientsByHouseholdID(context.Background(), "hh1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clients) != 2 {
		t.Fatalf("expected 2 clients in household, got %d", len(clients))
	}
}

func TestGetHousehold(t *testing.T) {
	hhID := "hh1"
	households := []Household{
		{ID: "hh1", Name: "Gupta Family"},
	}
	clients := []Client{
		{ID: "c3", AdvisorID: "a1", HouseholdID: &hhID, Name: "Swati Gupta"},
		{ID: "c1", AdvisorID: "a1", HouseholdID: nil, Name: "Priya Sharma"},
	}
	repo := &memHouseholdRepo{households: households, clients: clients}

	t.Run("client with household returns household", func(t *testing.T) {
		hh, err := repo.GetHouseholdByClientID(context.Background(), "c3")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hh == nil {
			t.Fatal("expected household, got nil")
		}
		if hh.ID != "hh1" {
			t.Errorf("expected household ID hh1, got %s", hh.ID)
		}
		if hh.Name != "Gupta Family" {
			t.Errorf("expected household name Gupta Family, got %s", hh.Name)
		}
	})

	t.Run("client without household returns nil without error", func(t *testing.T) {
		hh, err := repo.GetHouseholdByClientID(context.Background(), "c1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hh != nil {
			t.Errorf("expected nil household, got %+v", hh)
		}
	})

	t.Run("GetHousehold by ID", func(t *testing.T) {
		hh, err := repo.GetHousehold(context.Background(), "hh1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hh.Name != "Gupta Family" {
			t.Errorf("expected Gupta Family, got %s", hh.Name)
		}
	})
}

func TestGetGoalsByClientID(t *testing.T) {
	hhID := "hh1"
	repo := &memGoalRepo{
		goals: []Goal{
			{ID: "g1", ClientID: "c3", HouseholdID: nil, Name: "Mat leave savings", ProgressPct: 90, Status: GoalStatusAhead},
			{ID: "g2", ClientID: "c3", HouseholdID: &hhID, Name: "First home", ProgressPct: 45, Status: GoalStatusOnTrack},
			{ID: "g3", ClientID: "c4", HouseholdID: &hhID, Name: "Shared goal from Rohan", ProgressPct: 30, Status: GoalStatusBehind},
			{ID: "g4", ClientID: "c1", HouseholdID: nil, Name: "Other client goal", ProgressPct: 28, Status: GoalStatusBehind},
		},
		clients: []Client{
			{ID: "c3", HouseholdID: &hhID},
			{ID: "c4", HouseholdID: &hhID},
			{ID: "c1", HouseholdID: nil},
		},
	}

	t.Run("returns individual and household goals", func(t *testing.T) {
		goals, err := repo.GetGoalsByClientID(context.Background(), "c3")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should include g1 (own), g2 (own+household), g3 (household via c4)
		if len(goals) != 3 {
			t.Fatalf("expected 3 goals, got %d", len(goals))
		}
	})

	t.Run("client without household gets only own goals", func(t *testing.T) {
		goals, err := repo.GetGoalsByClientID(context.Background(), "c1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(goals) != 1 {
			t.Fatalf("expected 1 goal, got %d", len(goals))
		}
		if goals[0].ID != "g4" {
			t.Errorf("expected goal g4, got %s", goals[0].ID)
		}
	})
}

func TestGetNotes(t *testing.T) {
	d1 := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)

	repo := &memAdvisorNoteRepo{
		notes: []AdvisorNote{
			{ID: "n1", ClientID: "c1", AdvisorID: "a1", Date: d1, Text: "First note"},
			{ID: "n2", ClientID: "c1", AdvisorID: "a1", Date: d3, Text: "Third note"},
			{ID: "n3", ClientID: "c1", AdvisorID: "a1", Date: d2, Text: "Second note"},
		},
	}

	t.Run("notes returned ordered by date descending", func(t *testing.T) {
		notes, err := repo.GetNotes(context.Background(), "c1", "a1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(notes) != 3 {
			t.Fatalf("expected 3 notes, got %d", len(notes))
		}
		if notes[0].ID != "n2" {
			t.Errorf("expected first note n2 (most recent), got %s", notes[0].ID)
		}
		if notes[1].ID != "n3" {
			t.Errorf("expected second note n3, got %s", notes[1].ID)
		}
		if notes[2].ID != "n1" {
			t.Errorf("expected third note n1 (oldest), got %s", notes[2].ID)
		}
	})

	t.Run("empty slice for client with no notes", func(t *testing.T) {
		notes, err := repo.GetNotes(context.Background(), "c99", "a1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if notes == nil {
			t.Fatal("expected empty slice, got nil")
		}
		if len(notes) != 0 {
			t.Errorf("expected 0 notes, got %d", len(notes))
		}
	})
}

func TestAddNote(t *testing.T) {
	fixedTime := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC)
	repo := &memAdvisorNoteRepo{
		now: func() time.Time { return fixedTime },
	}

	note, err := repo.AddNote(context.Background(), "c1", "a1", "Discussed RRSP strategy")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if note.ClientID != "c1" {
		t.Errorf("expected client_id c1, got %s", note.ClientID)
	}
	if note.AdvisorID != "a1" {
		t.Errorf("expected advisor_id a1, got %s", note.AdvisorID)
	}
	if note.Text != "Discussed RRSP strategy" {
		t.Errorf("expected text 'Discussed RRSP strategy', got '%s'", note.Text)
	}
	if !note.Date.Equal(fixedTime) {
		t.Errorf("expected date %v, got %v", fixedTime, note.Date)
	}
	if note.ID == "" {
		t.Error("expected non-empty ID")
	}

	// Verify note is retrievable
	notes, err := repo.GetNotes(context.Background(), "c1", "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note after add, got %d", len(notes))
	}
}

func TestGetAdvisor(t *testing.T) {
	repo := &memAdvisorRepo{
		advisors: []Advisor{
			{ID: "a1", Name: "Shruti K.", Email: "shruti@example.com", Role: "Financial Advisor"},
		},
	}

	t.Run("valid ID returns advisor", func(t *testing.T) {
		a, err := repo.GetAdvisor(context.Background(), "a1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.Name != "Shruti K." {
			t.Errorf("expected name Shruti K., got %s", a.Name)
		}
	})

	t.Run("invalid ID returns error", func(t *testing.T) {
		_, err := repo.GetAdvisor(context.Background(), "nonexistent")
		if err == nil {
			t.Fatal("expected error for invalid ID, got nil")
		}
	})
}
