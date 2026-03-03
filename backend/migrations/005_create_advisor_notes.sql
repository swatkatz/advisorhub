CREATE TABLE IF NOT EXISTS advisor_notes (
    id TEXT PRIMARY KEY,
    client_id TEXT NOT NULL REFERENCES clients(id),
    advisor_id TEXT NOT NULL REFERENCES advisors(id),
    date DATE NOT NULL,
    text TEXT NOT NULL
);

CREATE INDEX idx_note_client_advisor_date ON advisor_notes(client_id, advisor_id, date DESC);
