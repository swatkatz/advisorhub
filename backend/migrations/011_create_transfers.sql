CREATE TABLE IF NOT EXISTS transfers (
    id TEXT PRIMARY KEY,
    client_id TEXT NOT NULL REFERENCES clients(id),
    source_institution TEXT NOT NULL,
    account_type TEXT NOT NULL CHECK (account_type IN ('RRSP', 'TFSA', 'FHSA', 'RESP', 'NON_REG')),
    amount DOUBLE PRECISION NOT NULL CHECK (amount > 0),
    status TEXT NOT NULL DEFAULT 'INITIATED' CHECK (status IN ('INITIATED', 'DOCUMENTS_SUBMITTED', 'IN_REVIEW', 'IN_TRANSIT', 'RECEIVED', 'INVESTED')),
    initiated_at DATE NOT NULL,
    last_status_change TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_transfer_client_id ON transfers(client_id);
CREATE INDEX IF NOT EXISTS idx_transfer_status ON transfers(status);
