CREATE TABLE action_items (
    id              TEXT PRIMARY KEY,
    client_id       TEXT NOT NULL REFERENCES clients(id),
    alert_id        TEXT,
    text            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'PENDING',
    due_date        DATE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at     TIMESTAMPTZ,
    resolution_note TEXT
);

CREATE INDEX idx_action_item_client_id ON action_items(client_id);
CREATE INDEX idx_action_item_alert_id ON action_items(alert_id);
