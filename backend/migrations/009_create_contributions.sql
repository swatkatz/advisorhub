CREATE TABLE contributions (
    id TEXT PRIMARY KEY,
    client_id TEXT NOT NULL REFERENCES clients(id),
    account_id TEXT NOT NULL REFERENCES accounts(id),
    account_type TEXT NOT NULL,
    amount DOUBLE PRECISION NOT NULL CHECK (amount > 0),
    date DATE NOT NULL,
    tax_year INTEGER NOT NULL
);

CREATE INDEX idx_contribution_client_year ON contributions (client_id, tax_year);
