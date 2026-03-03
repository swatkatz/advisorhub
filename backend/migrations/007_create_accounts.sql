CREATE TABLE accounts (
    id TEXT PRIMARY KEY,
    client_id TEXT NOT NULL REFERENCES clients(id),
    account_type TEXT NOT NULL CHECK (account_type IN ('RRSP', 'TFSA', 'FHSA', 'RESP', 'NON_REG')),
    institution TEXT NOT NULL,
    balance DOUBLE PRECISION NOT NULL,
    is_external BOOLEAN NOT NULL DEFAULT false,
    resp_beneficiary_id TEXT REFERENCES resp_beneficiaries(id),
    fhsa_lifetime_contributions DOUBLE PRECISION NOT NULL DEFAULT 0
);

CREATE INDEX idx_account_client_id ON accounts(client_id);
