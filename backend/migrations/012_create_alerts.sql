CREATE TABLE IF NOT EXISTS alerts (
    id                    TEXT PRIMARY KEY,
    condition_key         TEXT NOT NULL,
    client_id             TEXT NOT NULL REFERENCES clients(id),
    severity              TEXT NOT NULL,
    category              TEXT NOT NULL,
    status                TEXT NOT NULL DEFAULT 'OPEN',
    snoozed_until         TIMESTAMPTZ,
    payload               JSONB NOT NULL DEFAULT '{}',
    summary               TEXT NOT NULL DEFAULT '',
    draft_message         TEXT,
    linked_action_item_ids TEXT[] NOT NULL DEFAULT '{}',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at           TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_alert_condition_key_status ON alerts (condition_key, status);
CREATE INDEX IF NOT EXISTS idx_alert_client_id ON alerts (client_id);
CREATE INDEX IF NOT EXISTS idx_alert_client_id_status_severity ON alerts (client_id, status, severity);
