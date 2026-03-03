package client

import (
	"context"
	"time"
)

// GoalStatus represents the progress status of a goal.
type GoalStatus string

const (
	GoalStatusOnTrack GoalStatus = "ON_TRACK"
	GoalStatusBehind  GoalStatus = "BEHIND"
	GoalStatusAhead   GoalStatus = "AHEAD"
)

type Advisor struct {
	ID    string
	Name  string
	Email string
	Role  string
}

type Household struct {
	ID   string
	Name string
}

type Client struct {
	ID              string
	AdvisorID       string
	HouseholdID     *string
	Name            string
	Email           string
	DateOfBirth     time.Time
	LastMeetingDate time.Time
}

type Goal struct {
	ID           string
	ClientID     string
	HouseholdID  *string
	Name         string
	TargetAmount *float64
	TargetDate   *time.Time
	ProgressPct  int
	Status       GoalStatus
}

type AdvisorNote struct {
	ID        string
	ClientID  string
	AdvisorID string
	Date      time.Time
	Text      string
}

type ClientRepository interface {
	GetClient(ctx context.Context, id string) (*Client, error)
	GetClients(ctx context.Context, advisorID string) ([]Client, error)
	GetClientsByHouseholdID(ctx context.Context, householdID string) ([]Client, error)
}

type HouseholdRepository interface {
	GetHousehold(ctx context.Context, id string) (*Household, error)
	GetHouseholdByClientID(ctx context.Context, clientID string) (*Household, error)
}

type GoalRepository interface {
	GetGoalsByClientID(ctx context.Context, clientID string) ([]Goal, error)
}

type AdvisorNoteRepository interface {
	GetNotes(ctx context.Context, clientID string, advisorID string) ([]AdvisorNote, error)
	AddNote(ctx context.Context, clientID string, advisorID string, text string) (*AdvisorNote, error)
}

type AdvisorRepository interface {
	GetAdvisor(ctx context.Context, id string) (*Advisor, error)
}
