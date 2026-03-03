CREATE TABLE resp_beneficiaries (
    id TEXT PRIMARY KEY,
    client_id TEXT NOT NULL REFERENCES clients(id),
    name TEXT NOT NULL,
    date_of_birth DATE NOT NULL,
    lifetime_contributions DOUBLE PRECISION NOT NULL DEFAULT 0
);

CREATE INDEX idx_resp_beneficiary_client_id ON resp_beneficiaries(client_id);
