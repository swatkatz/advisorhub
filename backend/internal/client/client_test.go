package client

import (
	"context"
	"testing"
	"time"
)

func TestGetClients_ReturnsAllClientsForAdvisor(t *testing.T) {
	repo := newMemoryClientRepo()
	ctx := context.Background()
	advisorID := "a1"

	repo.clients["c1"] = Client{ID: "c1", AdvisorID: advisorID, Name: "Alice"}
	repo.clients["c2"] = Client{ID: "c2", AdvisorID: advisorID, Name: "Bob"}
	repo.clients["c3"] = Client{ID: "c3", AdvisorID: advisorID, Name: "Carol"}
	repo.clients["c4"] = Client{ID: "c4", AdvisorID: "a2", Name: "Other"}

	clients, err := repo.GetClients(ctx, advisorID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clients) != 3 {
		t.Fatalf("expected 3 clients, got %d", len(clients))
	}
}

func TestGetClient_ReturnsCorrectClient(t *testing.T) {
	repo := newMemoryClientRepo()
	ctx := context.Background()

	dob := time.Date(1990, 1, 15, 0, 0, 0, 0, time.UTC)
	lastMeeting := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	householdID := "h1"

	repo.clients["c1"] = Client{
		ID:              "c1",
		AdvisorID:       "a1",
		HouseholdID:     &householdID,
		Name:            "Alice",
		Email:           "alice@example.com",
		DateOfBirth:     dob,
		LastMeetingDate: lastMeeting,
	}

	c, err := repo.GetClient(ctx, "c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ID != "c1" {
		t.Errorf("expected ID c1, got %s", c.ID)
	}
	if c.Name != "Alice" {
		t.Errorf("expected name Alice, got %s", c.Name)
	}
	if c.Email != "alice@example.com" {
		t.Errorf("expected email alice@example.com, got %s", c.Email)
	}
	if !c.DateOfBirth.Equal(dob) {
		t.Errorf("expected DOB %v, got %v", dob, c.DateOfBirth)
	}
	if !c.LastMeetingDate.Equal(lastMeeting) {
		t.Errorf("expected last meeting %v, got %v", lastMeeting, c.LastMeetingDate)
	}
	if c.HouseholdID == nil || *c.HouseholdID != "h1" {
		t.Errorf("expected household_id h1, got %v", c.HouseholdID)
	}
}

func TestGetClient_InvalidID_ReturnsError(t *testing.T) {
	repo := newMemoryClientRepo()
	ctx := context.Background()

	_, err := repo.GetClient(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for invalid client ID, got nil")
	}
}

func TestGetClientsByHouseholdID_ReturnsBothMembers(t *testing.T) {
	repo := newMemoryClientRepo()
	ctx := context.Background()
	householdID := "h1"

	repo.clients["c1"] = Client{ID: "c1", AdvisorID: "a1", HouseholdID: &householdID, Name: "Swati"}
	repo.clients["c2"] = Client{ID: "c2", AdvisorID: "a1", HouseholdID: &householdID, Name: "Rohan"}
	repo.clients["c3"] = Client{ID: "c3", AdvisorID: "a1", Name: "Solo"}

	clients, err := repo.GetClientsByHouseholdID(ctx, householdID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clients) != 2 {
		t.Fatalf("expected 2 clients in household, got %d", len(clients))
	}
}

func TestGetHousehold_ReturnsHousehold(t *testing.T) {
	repo := newMemoryHouseholdRepo()
	ctx := context.Background()

	repo.households["h1"] = Household{ID: "h1", Name: "Gupta Family"}

	h, err := repo.GetHousehold(ctx, "h1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.Name != "Gupta Family" {
		t.Errorf("expected Gupta Family, got %s", h.Name)
	}
}

func TestGetHouseholdByClientID_WithHousehold(t *testing.T) {
	clientRepo := newMemoryClientRepo()
	householdRepo := newMemoryHouseholdRepoWithClients(clientRepo)
	ctx := context.Background()

	householdID := "h1"
	clientRepo.clients["c1"] = Client{ID: "c1", HouseholdID: &householdID}
	householdRepo.households["h1"] = Household{ID: "h1", Name: "Gupta Family"}

	h, err := householdRepo.GetHouseholdByClientID(ctx, "c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h == nil {
		t.Fatal("expected household, got nil")
	}
	if h.Name != "Gupta Family" {
		t.Errorf("expected Gupta Family, got %s", h.Name)
	}
}

func TestGetHouseholdByClientID_NoHousehold_ReturnsNil(t *testing.T) {
	clientRepo := newMemoryClientRepo()
	householdRepo := newMemoryHouseholdRepoWithClients(clientRepo)
	ctx := context.Background()

	clientRepo.clients["c1"] = Client{ID: "c1", HouseholdID: nil}

	h, err := householdRepo.GetHouseholdByClientID(ctx, "c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h != nil {
		t.Errorf("expected nil household, got %v", h)
	}
}

func TestGetGoalsByClientID_IncludesHouseholdGoals(t *testing.T) {
	repo := newMemoryGoalRepo()
	ctx := context.Background()

	householdID := "h1"
	clientRepo := newMemoryClientRepo()
	clientRepo.clients["c1"] = Client{ID: "c1", HouseholdID: &householdID}
	repo.clientRepo = clientRepo

	repo.goals["g1"] = Goal{ID: "g1", ClientID: "c1", Name: "Individual Goal 1"}
	repo.goals["g2"] = Goal{ID: "g2", ClientID: "c1", Name: "Individual Goal 2"}
	repo.goals["g3"] = Goal{ID: "g3", ClientID: "c2", HouseholdID: &householdID, Name: "Household Goal"}

	goals, err := repo.GetGoalsByClientID(ctx, "c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(goals) != 3 {
		t.Fatalf("expected 3 goals (2 individual + 1 household), got %d", len(goals))
	}
}

func TestGetNotes_OrderedByDateDescending(t *testing.T) {
	repo := newMemoryNoteRepo()
	ctx := context.Background()

	repo.notes = append(repo.notes,
		AdvisorNote{ID: "n1", ClientID: "c1", AdvisorID: "a1", Date: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC), Text: "First note"},
		AdvisorNote{ID: "n2", ClientID: "c1", AdvisorID: "a1", Date: time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC), Text: "Second note"},
		AdvisorNote{ID: "n3", ClientID: "c1", AdvisorID: "a1", Date: time.Date(2026, 1, 25, 0, 0, 0, 0, time.UTC), Text: "Third note"},
	)

	notes, err := repo.GetNotes(ctx, "c1", "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 3 {
		t.Fatalf("expected 3 notes, got %d", len(notes))
	}
	if notes[0].Text != "Second note" {
		t.Errorf("expected first note to be 'Second note' (newest), got %s", notes[0].Text)
	}
	if notes[1].Text != "Third note" {
		t.Errorf("expected second note to be 'Third note', got %s", notes[1].Text)
	}
	if notes[2].Text != "First note" {
		t.Errorf("expected third note to be 'First note' (oldest), got %s", notes[2].Text)
	}
}

func TestAddNote_CreatesNoteWithCurrentDate(t *testing.T) {
	repo := newMemoryNoteRepo()
	ctx := context.Background()

	note, err := repo.AddNote(ctx, "c1", "a1", "New note text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if note.ClientID != "c1" {
		t.Errorf("expected client_id c1, got %s", note.ClientID)
	}
	if note.AdvisorID != "a1" {
		t.Errorf("expected advisor_id a1, got %s", note.AdvisorID)
	}
	if note.Text != "New note text" {
		t.Errorf("expected text 'New note text', got %s", note.Text)
	}
	if note.ID == "" {
		t.Error("expected non-empty ID")
	}
	today := time.Now().Truncate(24 * time.Hour)
	noteDate := note.Date.Truncate(24 * time.Hour)
	if !noteDate.Equal(today) {
		t.Errorf("expected date to be today (%v), got %v", today, noteDate)
	}

	// Verify it's retrievable
	notes, err := repo.GetNotes(ctx, "c1", "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note after add, got %d", len(notes))
	}
}

func TestGetNotes_Empty_ReturnsEmptySlice(t *testing.T) {
	repo := newMemoryNoteRepo()
	ctx := context.Background()

	notes, err := repo.GetNotes(ctx, "c1", "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notes == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(notes) != 0 {
		t.Fatalf("expected 0 notes, got %d", len(notes))
	}
}

func TestGetAdvisor_ReturnsAdvisor(t *testing.T) {
	repo := newMemoryAdvisorRepo()
	ctx := context.Background()

	repo.advisors["a1"] = Advisor{ID: "a1", Name: "Shruti K.", Email: "shruti@example.com", Role: "Senior Advisor"}

	a, err := repo.GetAdvisor(ctx, "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Name != "Shruti K." {
		t.Errorf("expected Shruti K., got %s", a.Name)
	}
}

func TestGetAdvisor_InvalidID_ReturnsError(t *testing.T) {
	repo := newMemoryAdvisorRepo()
	ctx := context.Background()

	_, err := repo.GetAdvisor(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for invalid advisor ID, got nil")
	}
}
