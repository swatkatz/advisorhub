# Spec: Client

## Bounded Context

Owns: Client, Household, Advisor, AdvisorNote, Goal entities. Database migrations for these tables. CRUD operations and query interfaces for all owned entities.

Does not own: Account data (account context), alert computation or health status derivation (alert-lifecycle context), contribution data (contribution-engine context), transfer data (transfer-monitor context). Health status is computed at query time by the GraphQL resolver layer using alert data — this context only exposes the raw client data.

Depends on: nothing — this is a foundational context with no upstream dependencies.

Produces:
- `ClientRepository` interface: GetClient, GetClients, GetClientsByHouseholdID
- `HouseholdRepository` interface: GetHousehold, GetHouseholdByClientID
- `GoalRepository` interface: GetGoalsByClientID
- `AdvisorNoteRepository` interface: GetNotes, AddNote
- `AdvisorRepository` interface: GetAdvisor

## Contracts

### Input

No events consumed. This is a foundational data context.

Data is written via:
- Seed data loader (bulk insert on startup)
- `addNote` GraphQL mutation (routed through AdvisorNoteRepository)

### Output

Interfaces exposed to other contexts (by ID lookup only):

```go
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
```

### Data Model

**Advisor**
| Field | Type | Constraints |
|-------|------|-------------|
| id | string (PK) | required |
| name | string | required |
| email | string | required, unique |
| role | string | required |

Indexes: unique on `email`.

**Household**
| Field | Type | Constraints |
|-------|------|-------------|
| id | string (PK) | required |
| name | string | required |

Indexes: none beyond PK.

**Client**
| Field | Type | Constraints |
|-------|------|-------------|
| id | string (PK) | required |
| advisor_id | string (FK → Advisor) | required |
| household_id | string (FK → Household) | nullable |
| name | string | required |
| email | string | required, unique |
| date_of_birth | date | required |
| last_meeting_date | date | required |

Indexes: `idx_client_advisor_id` on `advisor_id` (primary query path — "all clients for this advisor"), `idx_client_household_id` on `household_id` (household member lookups), unique on `email`.

**Goal**
| Field | Type | Constraints |
|-------|------|-------------|
| id | string (PK) | required |
| client_id | string (FK → Client) | required |
| household_id | string (FK → Household) | nullable — for shared goals |
| name | string | required |
| target_amount | float | nullable |
| target_date | date | nullable |
| progress_pct | int | required, 0-100 |
| status | GoalStatus enum | required (ON_TRACK, BEHIND, AHEAD) |

Indexes: `idx_goal_client_id` on `client_id` (client detail view), `idx_goal_household_id` on `household_id` (household-level goals).

**AdvisorNote**
| Field | Type | Constraints |
|-------|------|-------------|
| id | string (PK) | required |
| client_id | string (FK → Client) | required |
| advisor_id | string (FK → Advisor) | required |
| date | date | required |
| text | string | required |

Indexes: `idx_note_client_advisor_date` on `(client_id, advisor_id, date DESC)`.

## State Machine

N/A — no state transitions in this context.

## Behaviors (EARS syntax)

- When `GetClients(advisorID)` is called, the system shall return all clients belonging to that advisor.
- When `GetClient(id)` is called, the system shall return the client with that ID, or an error if not found.
- When `GetClientsByHouseholdID(householdID)` is called, the system shall return all clients in that household.
- When `GetHouseholdByClientID(clientID)` is called, the system shall return the household for that client, or nil if the client has no household.
- When `GetGoalsByClientID(clientID)` is called, the system shall return all goals for that client, including household-level goals where the client is a member.
- When `GetNotes(clientID, advisorID)` is called, the system shall return all notes for that client by that advisor, ordered by date descending.
- When `AddNote(clientID, advisorID, text)` is called, the system shall create a new AdvisorNote with the current date and return it.
- Where a client has no household, `household_id` shall be null and `GetHouseholdByClientID` shall return nil without error.

## Decision Table

N/A

## Test Anchors

1. Given an advisor with 3 clients, when `GetClients(advisorID)` is called, then all 3 clients are returned.
2. Given a valid client ID, when `GetClient(id)` is called, then the correct client is returned with all fields populated.
3. Given an invalid client ID, when `GetClient(id)` is called, then an error is returned.
4. Given two clients in the same household, when `GetClientsByHouseholdID(householdID)` is called, then both clients are returned.
5. Given a client with a household, when `GetHouseholdByClientID(clientID)` is called, then the household is returned.
6. Given a client with no household, when `GetHouseholdByClientID(clientID)` is called, then nil is returned without error.
7. Given a client with 2 individual goals and 1 household-level goal, when `GetGoalsByClientID(clientID)` is called, then all 3 goals are returned.
8. Given a client with 3 notes, when `GetNotes(clientID, advisorID)` is called, then notes are returned ordered by date descending.
9. Given a valid client ID, advisor ID, and note text, when `AddNote(clientID, advisorID, text)` is called, then a new AdvisorNote is created with today's date, the given advisor_id, and the given text.
10. Given a client with no notes, when `GetNotes(clientID, advisorID)` is called, then an empty slice is returned without error.
