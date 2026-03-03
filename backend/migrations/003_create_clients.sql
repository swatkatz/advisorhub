CREATE TABLE IF NOT EXISTS clients (
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
