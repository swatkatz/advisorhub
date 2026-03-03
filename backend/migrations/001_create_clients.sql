-- 001_create_clients.sql
-- Client bounded context: Advisor, Household, Client, Goal, AdvisorNote

CREATE TABLE advisors (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    role TEXT NOT NULL
);

CREATE TABLE households (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE clients (
    id TEXT PRIMARY KEY,
    advisor_id TEXT NOT NULL REFERENCES advisors(id),
    household_id TEXT REFERENCES households(id),
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    date_of_birth DATE NOT NULL,
    last_meeting_date DATE NOT NULL
);

CREATE INDEX idx_client_advisor_id ON clients(advisor_id);
CREATE INDEX idx_client_household_id ON clients(household_id);

CREATE TABLE goals (
    id TEXT PRIMARY KEY,
    client_id TEXT NOT NULL REFERENCES clients(id),
    household_id TEXT REFERENCES households(id),
    name TEXT NOT NULL,
    target_amount DOUBLE PRECISION,
    target_date DATE,
    progress_pct INTEGER NOT NULL CHECK (progress_pct >= 0 AND progress_pct <= 100),
    status TEXT NOT NULL CHECK (status IN ('ON_TRACK', 'BEHIND', 'AHEAD'))
);

CREATE INDEX idx_goal_client_id ON goals(client_id);
CREATE INDEX idx_goal_household_id ON goals(household_id);

CREATE TABLE advisor_notes (
    id TEXT PRIMARY KEY,
    client_id TEXT NOT NULL REFERENCES clients(id),
    advisor_id TEXT NOT NULL REFERENCES advisors(id),
    date DATE NOT NULL,
    text TEXT NOT NULL
);

CREATE INDEX idx_note_client_advisor_date ON advisor_notes(client_id, advisor_id, date DESC);
