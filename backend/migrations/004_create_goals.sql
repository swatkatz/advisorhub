CREATE TABLE IF NOT EXISTS goals (
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
