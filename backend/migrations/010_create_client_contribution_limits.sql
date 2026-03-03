CREATE TABLE client_contribution_limits (
    id TEXT PRIMARY KEY,
    client_id TEXT NOT NULL REFERENCES clients(id),
    tax_year INTEGER NOT NULL,
    rrsp_deduction_limit DOUBLE PRECISION NOT NULL
);

CREATE UNIQUE INDEX idx_ccl_client_year ON client_contribution_limits (client_id, tax_year);
