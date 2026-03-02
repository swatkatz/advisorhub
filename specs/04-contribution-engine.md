# Spec: Contribution Engine

## Bounded Context

Owns: Contribution entity, ContributionRule (hardcoded reference data), ClientContributionLimit (per-client per-year RRSP deduction limits — seeded). Core computation logic: contribution room calculation per account type, over-contribution detection with penalty computation, CESG gap detection for RESP accounts, FHSA lifetime tracking. Database migrations for the `contributions` and `client_contribution_limits` tables.

Does not own: Account data (account context — reads via AccountRepository), client data (client context — reads via ClientRepository), RESP beneficiary data (account context — reads via RESPBeneficiaryRepository), alert creation or state management (alert-generator and alert-lifecycle contexts), deadline monitoring (temporal-scanner context), event bus infrastructure (event-bus context — publishes through it).

Depends on:
- event-bus: publishes events via `EventBus.Publish()`
- account: reads accounts via `AccountRepository.GetAccountsByClientID()` (needs `account_type`, `is_external`, `fhsa_lifetime_contributions`, `resp_beneficiary_id`)
- account: reads RESP beneficiaries via `RESPBeneficiaryRepository.GetRESPBeneficiariesByClientID()` (needs `lifetime_contributions` for CESG gap detection)
- account: reads lifetime caps via `AccountType` (for FHSA $40K and RESP $50K caps)
- account: writes lifetime totals via `AccountRepository.UpdateFHSALifetimeContributions()` and `RESPBeneficiaryRepository.UpdateLifetimeContributions()`

Produces:
- Events (source: `REACTIVE`): `OverContributionDetected`, `CESGGap`, `ContributionProcessed`
- `ContributionRepository` interface: GetContributionsByClient, RecordContribution, GetClientContributionLimit, SaveClientContributionLimit
- `ContributionEngine` interface: AnalyzeClient (runs all detections, emits events), GetContributionSummary (for GraphQL resolver), GetRoom (for temporal-scanner's `DEADLINE_WITH_ROOM` check)

## Contracts

### Input

No events consumed from the bus. This context is a producer, not a consumer.

Data is written via:
- Seed data loader (bulk inserts contributions and client contribution limits on startup)
- `ContributionRepository.RecordContribution()` (general-purpose write path)

Computation is triggered via:
- `ContributionEngine.AnalyzeClient()` — called by seed data loader after setup to run all detections and emit events
- `ContributionEngine.GetContributionSummary()` — called by GraphQL resolver for `contributionSummary` query
- `ContributionEngine.GetRoom()` — called by temporal-scanner for `DEADLINE_WITH_ROOM` check

### Output

**Events emitted** (via `EventBus.Publish()`, source: `REACTIVE`):

All events use the `EventEnvelope` from the event-bus context. `EntityID` is the client ID, `EntityType` is `EntityTypeClient`. Event-specific data is in the `Payload`.

`OverContributionDetected`:
```json
{
  "client_id": "c1",
  "account_type": "RRSP",
  "reason": "annual_limit",
  "limit": 31560,
  "contributed": 33860,
  "excess": 2300,
  "penalty_per_month": 23,
  "institutions_involved": ["Wealthsimple", "RBC"]
}
```
`reason` is `"annual_limit"` or `"lifetime_cap"` (FHSA only).

`CESGGap`:
```json
{
  "client_id": "c1",
  "beneficiary_id": "resp_ben_1",
  "contributed_ytd": 1800,
  "cesg_eligible_max": 2500,
  "gap_amount": 700,
  "potential_grant_loss": 140
}
```

`ContributionProcessed`:
```json
{
  "client_id": "c1",
  "account_type": "RRSP",
  "remaining_room": 0
}
```

**Interfaces exposed to other contexts:**

```go
type ContributionRepository interface {
    GetContributionsByClient(ctx context.Context, clientID string, taxYear int) ([]Contribution, error)
    RecordContribution(ctx context.Context, contribution *Contribution) (*Contribution, error)
    GetClientContributionLimit(ctx context.Context, clientID string, taxYear int) (*ClientContributionLimit, error)
    SaveClientContributionLimit(ctx context.Context, limit *ClientContributionLimit) (*ClientContributionLimit, error)
}

type ContributionEngine interface {
    AnalyzeClient(ctx context.Context, clientID string, taxYear int) error
    GetContributionSummary(ctx context.Context, clientID string, taxYear int) (*ContributionSummary, error)
    GetRoom(ctx context.Context, clientID string, accountType string, taxYear int) (float64, error)
}
```

`GetContributionsByClient` returns all contributions for a client in a tax year. The engine groups and aggregates by account type in memory — the prototype dataset is small enough that this is simpler than specialized queries.

**Write dependencies on account context:**

After `RecordContribution`, the engine recomputes lifetime totals from its contribution records and updates the account context:
- FHSA contributions → calls `AccountRepository.UpdateFHSALifetimeContributions(ctx, accountID, total)`
- RESP contributions → calls `RESPBeneficiaryRepository.UpdateLifetimeContributions(ctx, beneficiaryID, total)`

### Data Model

**Contribution** (persisted)

| Field | Type | Constraints |
|-------|------|-------------|
| id | string (PK) | required |
| client_id | string (FK → Client) | required |
| account_id | string (FK → Account) | required |
| account_type | AccountType enum | required — denormalized from Account for aggregation without cross-context joins |
| amount | float64 | required, > 0 |
| date | date | required |
| tax_year | int | required |

Indexes: `idx_contribution_client_year` on `(client_id, tax_year)` — primary query path for room calculation and summary (engine groups by account_type in memory).

**ClientContributionLimit** (persisted, seeded)

| Field | Type | Constraints |
|-------|------|-------------|
| id | string (PK) | required |
| client_id | string (FK → Client) | required |
| tax_year | int | required |
| rrsp_deduction_limit | float64 | required — pre-calculated: min(18% x earned_income, $32,490) |

Indexes: `idx_ccl_client_year` unique on `(client_id, tax_year)` — single-row lookup for RRSP room calculation.

**ContributionRule** (hardcoded Go constants, not a database table)

| Account Type | Annual Limit | Penalty | CESG |
|---|---|---|---|
| RRSP | per-client via ClientContributionLimit | 1%/month on excess | N/A |
| TFSA | $7,000 | 1%/month on excess | N/A |
| FHSA | $8,000/year | 1%/month on excess | N/A |
| RESP | None (CESG on first $2,500) | N/A | 20% up to $500/year |

Lifetime caps are owned by the account context as a property of AccountType.

**ContributionSummary** (computed, not persisted — returned by `GetContributionSummary`)

| Field | Type |
|-------|------|
| client_id | string |
| tax_year | int |
| accounts | []AccountContribution |

**AccountContribution** (computed, not persisted)

| Field | Type |
|-------|------|
| account_type | AccountType |
| annual_limit | float64 |
| contributed | float64 |
| remaining | float64 |
| is_over_contributed | bool |
| over_amount | float64 |
| penalty_per_month | float64 |
| deadline | *time.Time (nullable — RESP has none) |
| days_until_deadline | *int (nullable) |

## State Machine

N/A — no state transitions in this context. Contributions are append-only records. The contribution engine computes derived results but does not manage entity state transitions.

## Behaviors (EARS syntax)

**Room calculation:**
- When `GetRoom(clientID, accountType, taxYear)` is called, the system shall sum all contributions for that client and account type in the given tax year across all institutions (internal + external) and return `annual_limit - contributed`.
- Where `accountType = RRSP`, the system shall use the client's `rrsp_deduction_limit` from `ClientContributionLimit` as the annual limit.
- Where `accountType = TFSA`, the system shall use $7,000 as the annual limit.
- Where `accountType = FHSA`, the system shall use $8,000 as the annual limit.
- Where `accountType = RESP` or `accountType = NON_REG`, the system shall return 0 (no annual room concept).

**Over-contribution detection:**
- When `AnalyzeClient(clientID, taxYear)` is called, the system shall check each account type (RRSP, TFSA, FHSA) for over-contributions.
- Where contributed > annual_limit for an account type, the system shall emit `OverContributionDetected` with excess = contributed - limit, penalty_per_month = excess x 0.01, and reason = "annual_limit".
- Where contributed > annual_limit, the system shall include `institutions_involved` by grouping contributions per institution across all accounts of that type.

**FHSA lifetime tracking:**
- When recording a contribution to an FHSA account, the system shall recompute the lifetime total from contribution records and update `Account.fhsa_lifetime_contributions` via `AccountRepository.UpdateFHSALifetimeContributions()`.
- Where FHSA lifetime total >= lifetime cap (read from account context's AccountType), the system shall emit `OverContributionDetected` with reason = "lifetime_cap" and excess = lifetime_total - lifetime_cap.

**CESG gap detection:**
- When `AnalyzeClient(clientID, taxYear)` is called, the system shall check each RESP beneficiary for CESG matching gaps.
- Where RESP contributions for a beneficiary in the tax year < $2,500 AND beneficiary lifetime contributions < $50,000 (read from account context), the system shall emit `CESGGap` with gap_amount = 2500 - contributed_ytd and potential_grant_loss = gap_amount x 0.20.
- Where RESP beneficiary lifetime contributions >= $50,000, the system shall not emit `CESGGap` (no further CESG eligibility).

**RESP lifetime tracking:**
- When recording a contribution to a RESP account, the system shall recompute the lifetime total for the linked beneficiary and update `RESPBeneficiary.lifetime_contributions` via `RESPBeneficiaryRepository.UpdateLifetimeContributions()`.

**Contribution summary:**
- When `GetContributionSummary(clientID, taxYear)` is called, the system shall return an `AccountContribution` entry for each account type the client holds.
- The system shall compute `deadline` as: RRSP = 60 days after tax year end, TFSA/FHSA = Dec 31 of tax year, RESP/NON_REG = nil.
- The system shall compute `days_until_deadline` relative to the current date.

**ContributionProcessed event:**
- When `AnalyzeClient` completes analysis for an account type with no over-contribution, the system shall emit `ContributionProcessed` with the remaining room.

## Decision Table

**Room calculation by account type:**

| Account Type | Annual Limit Source | Penalty | CESG Check | Deadline |
|---|---|---|---|---|
| RRSP | ClientContributionLimit.rrsp_deduction_limit | 1%/month on excess | No | 60 days after year-end |
| TFSA | $7,000 (hardcoded) | 1%/month on excess | No | Dec 31 |
| FHSA | $8,000 (hardcoded) | 1%/month on excess | No | Dec 31 |
| RESP | None | None | Yes — gap if ytd < $2,500 | None |
| NON_REG | None | None | No | None |

**Over-contribution decision:**

| Account Type | Contributed vs Annual Limit | Result |
|---|---|---|
| RRSP, TFSA, FHSA | contributed <= limit | No event |
| RRSP, TFSA, FHSA | contributed > limit | Emit OverContributionDetected (reason: "annual_limit") |
| RESP, NON_REG | — | No annual limit check |

**FHSA lifetime decision (additional check):**

| FHSA Lifetime Total vs Cap | Result |
|---|---|
| lifetime < cap | Annual limit applies normally ($8K) |
| lifetime >= cap | Any further contribution is excess → Emit OverContributionDetected (reason: "lifetime_cap") |

**CESG gap decision:**

| RESP YTD vs $2,500 | Beneficiary Lifetime vs $50K | Result |
|---|---|---|
| ytd >= $2,500 | any | No gap — full CESG match |
| ytd < $2,500 | lifetime < $50K | Emit CESGGap |
| ytd < $2,500 | lifetime >= $50K | No gap — lifetime cap reached, no further CESG |

## Test Anchors

1. Given a client with RRSP contributions of $20,000 across two institutions (WS $12,000 + RBC $8,000) and an rrsp_deduction_limit of $32,490, when `GetRoom(clientID, "RRSP", taxYear)` is called, then $12,490 is returned.
2. Given a client with RRSP contributions totaling $33,860 against an rrsp_deduction_limit of $31,560, when `AnalyzeClient` is called, then `OverContributionDetected` is emitted with excess $2,300, penalty_per_month $23, reason "annual_limit", and institutions_involved containing both institutions.
3. Given a client with TFSA contributions of $7,000 (exactly at limit), when `GetRoom(clientID, "TFSA", taxYear)` is called, then 0 is returned and no OverContributionDetected is emitted.
4. Given a client with TFSA contributions of $7,500, when `AnalyzeClient` is called, then `OverContributionDetected` is emitted with excess $500, reason "annual_limit".
5. Given a client with FHSA contributions of $6,000 this year and lifetime total of $38,000, when `AnalyzeClient` is called, then `OverContributionDetected` is emitted with reason "lifetime_cap", excess $4,000 (the $2,000 beyond the $40K cap, not the $2,000 remaining annual room).
6. Given a client with FHSA contributions of $9,000 this year and lifetime total of $25,000, when `AnalyzeClient` is called, then `OverContributionDetected` is emitted with reason "annual_limit", excess $1,000.
7. Given a client with RESP contributions of $1,800 for a beneficiary with lifetime contributions of $38,200, when `AnalyzeClient` is called, then `CESGGap` is emitted with gap_amount $700 and potential_grant_loss $140.
8. Given a client with RESP contributions of $2,500 for a beneficiary, when `AnalyzeClient` is called, then no `CESGGap` is emitted (full CESG match).
9. Given a client with RESP contributions of $1,000 for a beneficiary with lifetime contributions of $50,000, when `AnalyzeClient` is called, then no `CESGGap` is emitted (lifetime cap reached, no further CESG eligibility).
10. Given a client with contributions split across 3 institutions for RRSP, when `AnalyzeClient` detects over-contribution, then `institutions_involved` contains all 3 institutions with per-institution amounts.
11. Given a client with RRSP, TFSA, and FHSA accounts, when `GetContributionSummary(clientID, taxYear)` is called, then an AccountContribution entry is returned for each type with correct contributed, remaining, annual_limit, deadline, and days_until_deadline.
12. Given a contribution recorded to an FHSA account, when `RecordContribution` completes, then `Account.fhsa_lifetime_contributions` is updated via `AccountRepository.UpdateFHSALifetimeContributions()`.
13. Given a contribution recorded to a RESP account, when `RecordContribution` completes, then `RESPBeneficiary.lifetime_contributions` is updated via `RESPBeneficiaryRepository.UpdateLifetimeContributions()`.
14. Given a client with no contributions for the tax year, when `GetContributionSummary(clientID, taxYear)` is called, then AccountContribution entries show contributed = 0 and remaining = annual_limit for each type.
15. Given a client with no `ClientContributionLimit` for the tax year, when `GetRoom(clientID, "RRSP", taxYear)` is called, then the default $32,490 cap is used.
