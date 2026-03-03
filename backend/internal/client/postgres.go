package client

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// clientRow maps to the clients table.
type clientRow struct {
	ID              string     `db:"id"`
	AdvisorID       string     `db:"advisor_id"`
	HouseholdID     *string    `db:"household_id"`
	Name            string     `db:"name"`
	Email           string     `db:"email"`
	DateOfBirth     time.Time  `db:"date_of_birth"`
	LastMeetingDate time.Time  `db:"last_meeting_date"`
}

func clientFromRow(r clientRow) Client {
	return Client{
		ID:              r.ID,
		AdvisorID:       r.AdvisorID,
		HouseholdID:     r.HouseholdID,
		Name:            r.Name,
		Email:           r.Email,
		DateOfBirth:     r.DateOfBirth,
		LastMeetingDate: r.LastMeetingDate,
	}
}

// PostgresClientRepo implements ClientRepository using PostgreSQL.
type PostgresClientRepo struct {
	db *sqlx.DB
}

// NewPostgresClientRepo creates a new PostgresClientRepo.
func NewPostgresClientRepo(db *sqlx.DB) *PostgresClientRepo {
	return &PostgresClientRepo{db: db}
}

func (r *PostgresClientRepo) GetClient(ctx context.Context, id string) (*Client, error) {
	var row clientRow
	err := r.db.GetContext(ctx, &row, "SELECT id, advisor_id, household_id, name, email, date_of_birth, last_meeting_date FROM clients WHERE id = $1", id)
	if err != nil {
		return nil, fmt.Errorf("getting client %s: %w", id, err)
	}
	c := clientFromRow(row)
	return &c, nil
}

func (r *PostgresClientRepo) GetClients(ctx context.Context, advisorID string) ([]Client, error) {
	var rows []clientRow
	err := r.db.SelectContext(ctx, &rows, "SELECT id, advisor_id, household_id, name, email, date_of_birth, last_meeting_date FROM clients WHERE advisor_id = $1 ORDER BY id", advisorID)
	if err != nil {
		return nil, fmt.Errorf("getting clients for advisor %s: %w", advisorID, err)
	}
	clients := make([]Client, len(rows))
	for i, row := range rows {
		clients[i] = clientFromRow(row)
	}
	return clients, nil
}

func (r *PostgresClientRepo) GetClientsByHouseholdID(ctx context.Context, householdID string) ([]Client, error) {
	var rows []clientRow
	err := r.db.SelectContext(ctx, &rows, "SELECT id, advisor_id, household_id, name, email, date_of_birth, last_meeting_date FROM clients WHERE household_id = $1 ORDER BY id", householdID)
	if err != nil {
		return nil, fmt.Errorf("getting clients for household %s: %w", householdID, err)
	}
	clients := make([]Client, len(rows))
	for i, row := range rows {
		clients[i] = clientFromRow(row)
	}
	return clients, nil
}

// householdRow maps to the households table.
type householdRow struct {
	ID   string `db:"id"`
	Name string `db:"name"`
}

// PostgresHouseholdRepo implements HouseholdRepository using PostgreSQL.
type PostgresHouseholdRepo struct {
	db *sqlx.DB
}

// NewPostgresHouseholdRepo creates a new PostgresHouseholdRepo.
func NewPostgresHouseholdRepo(db *sqlx.DB) *PostgresHouseholdRepo {
	return &PostgresHouseholdRepo{db: db}
}

func (r *PostgresHouseholdRepo) GetHousehold(ctx context.Context, id string) (*Household, error) {
	var row householdRow
	err := r.db.GetContext(ctx, &row, "SELECT id, name FROM households WHERE id = $1", id)
	if err != nil {
		return nil, fmt.Errorf("getting household %s: %w", id, err)
	}
	h := Household{ID: row.ID, Name: row.Name}
	return &h, nil
}

func (r *PostgresHouseholdRepo) GetHouseholdByClientID(ctx context.Context, clientID string) (*Household, error) {
	var householdID *string
	err := r.db.GetContext(ctx, &householdID, "SELECT household_id FROM clients WHERE id = $1", clientID)
	if err != nil {
		return nil, fmt.Errorf("getting household for client %s: %w", clientID, err)
	}
	if householdID == nil {
		return nil, nil
	}
	return r.GetHousehold(ctx, *householdID)
}

// goalRow maps to the goals table.
type goalRow struct {
	ID           string     `db:"id"`
	ClientID     string     `db:"client_id"`
	HouseholdID  *string    `db:"household_id"`
	Name         string     `db:"name"`
	TargetAmount *float64   `db:"target_amount"`
	TargetDate   *time.Time `db:"target_date"`
	ProgressPct  int        `db:"progress_pct"`
	Status       GoalStatus `db:"status"`
}

func goalFromRow(r goalRow) Goal {
	return Goal{
		ID:           r.ID,
		ClientID:     r.ClientID,
		HouseholdID:  r.HouseholdID,
		Name:         r.Name,
		TargetAmount: r.TargetAmount,
		TargetDate:   r.TargetDate,
		ProgressPct:  r.ProgressPct,
		Status:       r.Status,
	}
}

// PostgresGoalRepo implements GoalRepository using PostgreSQL.
type PostgresGoalRepo struct {
	db *sqlx.DB
}

// NewPostgresGoalRepo creates a new PostgresGoalRepo.
func NewPostgresGoalRepo(db *sqlx.DB) *PostgresGoalRepo {
	return &PostgresGoalRepo{db: db}
}

func (r *PostgresGoalRepo) GetGoalsByClientID(ctx context.Context, clientID string) ([]Goal, error) {
	// Get goals directly owned by the client, plus household-level goals
	// where the client is a member of that household.
	query := `
		SELECT g.id, g.client_id, g.household_id, g.name, g.target_amount, g.target_date, g.progress_pct, g.status
		FROM goals g
		WHERE g.client_id = $1
		   OR (g.household_id IS NOT NULL AND g.household_id = (SELECT household_id FROM clients WHERE id = $1))
		ORDER BY g.id`
	var rows []goalRow
	err := r.db.SelectContext(ctx, &rows, query, clientID)
	if err != nil {
		return nil, fmt.Errorf("getting goals for client %s: %w", clientID, err)
	}
	goals := make([]Goal, len(rows))
	for i, row := range rows {
		goals[i] = goalFromRow(row)
	}
	return goals, nil
}

// noteRow maps to the advisor_notes table.
type noteRow struct {
	ID        string    `db:"id"`
	ClientID  string    `db:"client_id"`
	AdvisorID string    `db:"advisor_id"`
	Date      time.Time `db:"date"`
	Text      string    `db:"text"`
}

// PostgresNoteRepo implements AdvisorNoteRepository using PostgreSQL.
type PostgresNoteRepo struct {
	db *sqlx.DB
}

// NewPostgresNoteRepo creates a new PostgresNoteRepo.
func NewPostgresNoteRepo(db *sqlx.DB) *PostgresNoteRepo {
	return &PostgresNoteRepo{db: db}
}

func (r *PostgresNoteRepo) GetNotes(ctx context.Context, clientID string, advisorID string) ([]AdvisorNote, error) {
	var rows []noteRow
	err := r.db.SelectContext(ctx, &rows, "SELECT id, client_id, advisor_id, date, text FROM advisor_notes WHERE client_id = $1 AND advisor_id = $2 ORDER BY date DESC", clientID, advisorID)
	if err != nil {
		return nil, fmt.Errorf("getting notes for client %s: %w", clientID, err)
	}
	notes := make([]AdvisorNote, len(rows))
	for i, row := range rows {
		notes[i] = AdvisorNote{ID: row.ID, ClientID: row.ClientID, AdvisorID: row.AdvisorID, Date: row.Date, Text: row.Text}
	}
	return notes, nil
}

func (r *PostgresNoteRepo) AddNote(ctx context.Context, clientID string, advisorID string, text string) (*AdvisorNote, error) {
	id := fmt.Sprintf("note_%d", time.Now().UnixNano())
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO advisor_notes (id, client_id, advisor_id, date, text) VALUES ($1, $2, $3, $4, $5)",
		id, clientID, advisorID, now, text)
	if err != nil {
		return nil, fmt.Errorf("adding note for client %s: %w", clientID, err)
	}
	return &AdvisorNote{
		ID:        id,
		ClientID:  clientID,
		AdvisorID: advisorID,
		Date:      now,
		Text:      text,
	}, nil
}

// advisorRow maps to the advisors table.
type advisorRow struct {
	ID    string `db:"id"`
	Name  string `db:"name"`
	Email string `db:"email"`
	Role  string `db:"role"`
}

// PostgresAdvisorRepo implements AdvisorRepository using PostgreSQL.
type PostgresAdvisorRepo struct {
	db *sqlx.DB
}

// NewPostgresAdvisorRepo creates a new PostgresAdvisorRepo.
func NewPostgresAdvisorRepo(db *sqlx.DB) *PostgresAdvisorRepo {
	return &PostgresAdvisorRepo{db: db}
}

func (r *PostgresAdvisorRepo) GetAdvisor(ctx context.Context, id string) (*Advisor, error) {
	var row advisorRow
	err := r.db.GetContext(ctx, &row, "SELECT id, name, email, role FROM advisors WHERE id = $1", id)
	if err != nil {
		return nil, fmt.Errorf("getting advisor %s: %w", id, err)
	}
	a := Advisor{ID: row.ID, Name: row.Name, Email: row.Email, Role: row.Role}
	return &a, nil
}
