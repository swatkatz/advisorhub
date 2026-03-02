# Spec: Account

## Bounded Context

Owns: Account, AccountType enum, RESPBeneficiary entities. Database migrations for these tables. CRUD operations and query interfaces for all owned entities.

Does not own: Client data (client context), contribution records or room calculations (contribution-engine context), transfer records (transfer-monitor context), alert data (alert-lifecycle context). Balance updates from real market data — balances are seeded and static for the prototype.

Depends on: Nothing — this is a foundational data context. References `client_id` via foreign key but does not import or depend on the client package.

Produces:
- `AccountType` enum: RRSP, TFSA, FHSA, RESP, NON_REG
- `AccountRepository` interface: GetAccount, GetAccountsByClientID
- `RESPBeneficiaryRepository` interface: GetRESPBeneficiary, GetRESPBeneficiariesByClientID

## Contracts

### Input

No events consumed. This is a foundational data context.

Data is written via:
- Seed data loader (bulk insert on startup)

### Output

Interfaces exposed to other contexts (by ID lookup only):

```go
type AccountRepository interface {
    GetAccount(ctx context.Context, id string) (*Account, error)
    GetAccountsByClientID(ctx context.Context, clientID string) ([]Account, error)
}

type RESPBeneficiaryRepository interface {
    GetRESPBeneficiary(ctx context.Context, id string) (*RESPBeneficiary, error)
    GetRESPBeneficiariesByClientID(ctx context.Context, clientID string) ([]RESPBeneficiary, error)
}
```

`GetAccountsByClientID` returns all accounts (internal + external). The GraphQL resolver filters by `is_external` to populate `Client.accounts` vs `Client.externalAccounts` — consistent with the convention that repos return by ID order and resolvers handle presentation logic.

### Data Model

**AccountType** (enum)

Values: `RRSP`, `TFSA`, `FHSA`, `RESP`, `NON_REG`

Owned by this context. Other contexts (contribution-engine, transfer-monitor, temporal-scanner) import this enum.

**Account**

| Field | Type | Constraints |
|-------|------|-------------|
| id | string (PK) | required |
| client_id | string (FK → Client) | required |
| account_type | AccountType enum | required |
| institution | string | required |
| balance | float64 | required |
| is_external | boolean | required, default false |
| resp_beneficiary_id | string (FK → RESPBeneficiary) | nullable — only populated for RESP accounts |
| fhsa_lifetime_contributions | float64 | required, default 0 — only meaningful for FHSA accounts |

Indexes: `idx_account_client_id` on `client_id` (primary query path — "all accounts for this client").

**RESPBeneficiary**

| Field | Type | Constraints |
|-------|------|-------------|
| id | string (PK) | required |
| client_id | string (FK → Client) | required — the subscriber/owner (simplified for prototype) |
| name | string | required |
| date_of_birth | date | required |
| lifetime_contributions | float64 | required, default 0 |

Indexes: `idx_resp_beneficiary_client_id` on `client_id` (beneficiaries for a client lookup).

## State Machine

N/A — no state transitions in this context.

## Behaviors (EARS syntax)

- When `GetAccount(id)` is called, the system shall return the account with that ID, or an error if not found.
- When `GetAccountsByClientID(clientID)` is called, the system shall return all accounts belonging to that client (both internal and external).
- When `GetRESPBeneficiary(id)` is called, the system shall return the RESP beneficiary with that ID, or an error if not found.
- When `GetRESPBeneficiariesByClientID(clientID)` is called, the system shall return all RESP beneficiaries linked to that client.
- Where an account has `account_type = RESP`, `resp_beneficiary_id` shall be non-null and reference a valid RESPBeneficiary.
- Where an account has `account_type != RESP`, `resp_beneficiary_id` shall be null.
- Where an account has `account_type = FHSA`, `fhsa_lifetime_contributions` shall reflect the cumulative total contributed to that account.
- Where an account has `account_type != FHSA`, `fhsa_lifetime_contributions` shall be 0.

## Decision Table

N/A

## Test Anchors

1. Given an account with a valid ID, when `GetAccount(id)` is called, then the correct account is returned with all fields populated.
2. Given an invalid account ID, when `GetAccount(id)` is called, then an error is returned.
3. Given a client with 3 accounts (2 internal, 1 external), when `GetAccountsByClientID(clientID)` is called, then all 3 accounts are returned.
4. Given a client with no accounts, when `GetAccountsByClientID(clientID)` is called, then an empty slice is returned without error.
5. Given a RESP account, when it is created, then `resp_beneficiary_id` is non-null and references a valid RESPBeneficiary.
6. Given a non-RESP account, when it is created, then `resp_beneficiary_id` is null.
7. Given a RESP beneficiary with a valid ID, when `GetRESPBeneficiary(id)` is called, then the correct beneficiary is returned with all fields populated.
8. Given an invalid beneficiary ID, when `GetRESPBeneficiary(id)` is called, then an error is returned.
9. Given a client with 2 RESP beneficiaries, when `GetRESPBeneficiariesByClientID(clientID)` is called, then both beneficiaries are returned.
10. Given a client with no RESP beneficiaries, when `GetRESPBeneficiariesByClientID(clientID)` is called, then an empty slice is returned without error.
11. Given an FHSA account with $15,000 lifetime contributions, when queried, then `fhsa_lifetime_contributions` reflects $15,000.
12. Given a non-FHSA account, when queried, then `fhsa_lifetime_contributions` is 0.
