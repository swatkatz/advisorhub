package client

import (
	"context"
	"time"
)

// GoalStatus represents the status of a client goal.
type GoalStatus string

const (
	GoalStatusOnTrack GoalStatus = "ON_TRACK"
	GoalStatusBehind  GoalStatus = "BEHIND"
	GoalStatusAhead   GoalStatus = "AHEAD"
)

// Advisor represents a financial advisor.
type Advisor struct {
	ID    string
	Name  string
	Email string
	Role  string
}

// Household is an optional grouping for couples/families.
type Household struct {
	ID   string
	Name string
}

// Client represents an individual person managed by an advisor.
type Client struct {
	ID              string
	AdvisorID       string
	HouseholdID     *string // nullable
	Name            string
	Email           string
	DateOfBirth     time.Time
	LastMeetingDate time.Time
}

// Goal belongs to a client or household.
type Goal struct {
	ID           string
	ClientID     string
	HouseholdID  *string // nullable — for shared goals
	Name         string
	TargetAmount *float64
	TargetDate   *time.Time
	ProgressPct  int
	Status       GoalStatus
}

// AdvisorNote is an append-only log entry per client.
type AdvisorNote struct {
	ID        string
	ClientID  string
	AdvisorID string
	Date      time.Time
	Text      string
}

// ClientRepository provides access to client data.
type ClientRepository interface {
	GetClient(ctx context.Context, id string) (*Client, error)
	GetClients(ctx context.Context, advisorID string) ([]Client, error)
	GetClientsByHouseholdID(ctx context.Context, householdID string) ([]Client, error)
}

// HouseholdRepository provides access to household data.
type HouseholdRepository interface {
	GetHousehold(ctx context.Context, id string) (*Household, error)
	GetHouseholdByClientID(ctx context.Context, clientID string) (*Household, error)
}

// GoalRepository provides access to goal data.
type GoalRepository interface {
	GetGoalsByClientID(ctx context.Context, clientID string) ([]Goal, error)
}

// AdvisorNoteRepository provides access to advisor notes.
type AdvisorNoteRepository interface {
	GetNotes(ctx context.Context, clientID string, advisorID string) ([]AdvisorNote, error)
	AddNote(ctx context.Context, clientID string, advisorID string, text string) (*AdvisorNote, error)
}

// AdvisorRepository provides access to advisor data.
type AdvisorRepository interface {
	GetAdvisor(ctx context.Context, id string) (*Advisor, error)
}
