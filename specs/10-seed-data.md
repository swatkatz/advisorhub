# Spec: Seed Data

## Bounded Context

Owns: Seed data loader function. Populates the database on startup with advisor, households, clients, accounts, RESP beneficiaries, contributions, transfers, goals, and advisor notes. Emits pre-computed events for mocked scenarios (PortfolioDrift, TaxLossOpportunity, DividendReceived, ContributionProcessed, TransferCompleted). Does not own any database tables — it writes into tables owned by other contexts.

Does not own: Any database tables or migrations (those belong to client, account, contribution-engine, transfer-monitor contexts). Business logic (contribution room calculation, stuck detection, alert mapping, etc.). The seed loader is a data pump — it inserts rows and fires events, nothing more.

Depends on:
- client: writes via `ClientRepository`, `HouseholdRepository`, `GoalRepository`, `AdvisorRepository`, `AdvisorNoteRepository`
- account: writes via `AccountRepository`, `RESPBeneficiaryRepository`
- contribution-engine: writes via `ContributionRepository`
- transfer-monitor: writes via `TransferRepository`
- event-bus: publishes pre-computed events via `EventBus.Publish()`

Produces:
- Events (source: `REACTIVE`): `ContributionProcessed`, `DividendReceived`, `TransferCompleted`
- Events (source: `ANALYTICAL`): `PortfolioDrift`, `TaxLossOpportunity`
- `SeedLoader` interface: `Load(ctx) error`

## Contracts

### Input

No events consumed. No external input. The seed loader is called once on startup by `server.go`.

Triggered via:
- `SeedLoader.Load(ctx)` — called during server initialization, before the event bus starts delivering to consumers

### Output

**Data written to other contexts' repositories:**

| Repository | Method | What's Seeded |
|---|---|---|
| `AdvisorRepository` | Create | 1 advisor (Shruti K.) |
| `HouseholdRepository` | Create | 2 households (Gupta Family, Williams Family) |
| `ClientRepository` | Create | 10 clients (c1–c10) |
| `AccountRepository` | Create | All accounts (internal WS + external) per client |
| `RESPBeneficiaryRepository` | Create | Priya's son, David's children |
| `ContributionRepository` | Create | Contribution records for current tax year |
| `TransferRepository` | Create | 6 transfers at various stages |
| `GoalRepository` | Create | 9 goals across clients/households |
| `AdvisorNoteRepository` | AddNote | 2–3 notes per client |

**Events emitted** (via `EventBus.Publish()`):

| Event Type | Source | EntityID | EntityType | Payload |
|---|---|---|---|---|
| `PortfolioDrift` | ANALYTICAL | c9 | Client | `{ client_id: "c9", drift_pct: 12, current_allocation: {tech: 42, ...}, target_allocation: {tech: 30, ...} }` |
| `TaxLossOpportunity` | ANALYTICAL | c8 | Client | `{ client_id: "c8", holding: "Canadian Energy ETF", unrealized_loss: 3200 }` |
| `DividendReceived` | REACTIVE | c9 | Client | `{ client_id: "c9", amount: 1240 }` |
| `ContributionProcessed` | REACTIVE | c6 | Client | `{ client_id: "c6", account_type: "NON_REG", amount: 185000 }` |
| `TransferCompleted` | REACTIVE | t2 | Transfer | `{ transfer_id: "t2", client_id: "c6", source_institution: "Scotia", account_type: "NON_REG", amount: 185000 }` |

**Interface exposed:**

```go
type SeedLoader interface {
    Load(ctx context.Context) error
}
```

### Data Model

No owned tables. All seed data is hardcoded in Go structs within the `seed` package. The loader calls repository interfaces to persist data into tables owned by other contexts.

**Seed data reference** (values from `docs/ARCHITECTURE.md` sections 5–7):

**Advisor**

| ID | Name | Email | Role |
|---|---|---|---|
| adv1 | Shruti K. | shruti@wealthsimple.com | Financial Advisor |

**Households**

| ID | Name |
|---|---|
| h1 | Gupta Family |
| h2 | Williams Family |

**Clients**

| ID | Name | Household | DOB | Last Meeting |
|---|---|---|---|---|
| c1 | Priya Sharma | — | 1988-03-15 | 2025-12-14 |
| c2 | Marcus Chen | — | 1955-11-08 | 2026-01-22 |
| c3 | Swati Gupta | h1 | 1990-07-22 | 2026-02-10 |
| c4 | Rohan Gupta | h1 | 1989-01-30 | 2026-02-10 |
| c5 | Elena Vasquez | — | 1982-09-11 | 2025-09-05 |
| c6 | James Williams | h2 | 1975-04-18 | 2026-02-25 |
| c7 | Tanya Williams | h2 | 1977-08-03 | 2026-02-25 |
| c8 | Amir Patel | — | 1993-06-27 | 2025-11-18 |
| c9 | Sophie Tremblay | — | 1970-12-01 | 2026-01-30 |
| c10 | David Kim | — | 1980-05-14 | 2025-08-12 |

**Accounts** — seeded per client to support alert-triggering scenarios. Includes both internal (Wealthsimple) and external accounts. Exact account IDs and balances are implementation details — the key constraints are:

- Priya (c1): WS RRSP ($18,860 contributed), external RBC RRSP ($15,000 contributed, balance $45K), WS TFSA, external RBC TFSA ($12K), WS FHSA, WS RESP (beneficiary: son, $1,800 ytd, $38,200 lifetime)
- Marcus (c2): WS RRSP ($620K balance) — RRIF conversion scenario
- Swati (c3): WS RRSP (needs $8,200 room remaining for deadline scenario), WS TFSA, WS FHSA
- Rohan (c4): WS NON_REG ($45,200 cash, idle 34 days — cash uninvested scenario), WS TFSA
- Elena (c5): WS RRSP, WS TFSA
- James (c6): WS NON_REG (received $185K transfer from Scotia)
- Tanya (c7): WS RRSP, WS TFSA — portfolio drift scenario (tech 42% vs 30% target)
- Amir (c8): WS RRSP (pending $67,400 transfer from TD), WS TFSA — tax loss scenario
- Sophie (c9): WS RRSP, WS TFSA, WS NON_REG — dividend scenario
- David (c10): WS RESP (children as beneficiaries, oldest turns 17 next year)

**Transfers**

| ID | Client | Source | Type | Amount | Status | Days in Stage |
|---|---|---|---|---|---|---|
| t1 | c8 (Amir) | TD | RRSP | $67,400 | DOCUMENTS_SUBMITTED | 18 |
| t2 | c6 (James) | Scotia | NON_REG | $185,000 | INVESTED | 0 |
| t3 | c1 (Priya) | RBC | RRSP | $42,000 | IN_TRANSIT | 3 |
| t4 | c10 (David) | BMO | TFSA | $28,500 | IN_REVIEW | 5 |
| t5 | c5 (Elena) | Desjardins | RRSP | $55,000 | INITIATED | 2 |
| t6 | c9 (Sophie) | National Bank | NON_REG | $120,000 | IN_TRANSIT | 6 |

To seed `days_in_stage`, the loader sets `last_status_change` to `now - N days` where N is the target days_in_stage. For INVESTED transfers, `last_status_change` is set to today.

**Goals**

| Client(s) | Goal | Target | Progress | Status |
|---|---|---|---|---|
| c1 (Priya) | First home (FHSA) | $120K | 28% | BEHIND |
| c2 (Marcus) | Retirement at 65 | $2M | 85% | ON_TRACK |
| c3 + c4 (household h1) | First home | $200K | 45% | ON_TRACK |
| c3 (Swati) | Mat leave savings | $30K | 90% | AHEAD |
| c5 (Elena) | Retirement at 60 | $800K | 42% | BEHIND |
| c6 + c7 (household h2) | Kids education (RESP) | $150K | 68% | ON_TRACK |
| c8 (Amir) | Emergency fund | $25K | 60% | ON_TRACK |
| c9 (Sophie) | Early retirement at 58 | $1.5M | 72% | ON_TRACK |
| c10 (David) | Son's university | $80K | 55% | BEHIND |

Household-level goals use `household_id` and set `client_id` to one member (the primary — Swati for h1, James for h2).

**Advisor Notes** — 2–3 per client. Realistic free-text notes. Example content from architecture doc; exact text is an implementation detail.

## State Machine

N/A — no state transitions in this context.

## Behaviors (EARS syntax)

- When `Load(ctx)` is called, the system shall insert seed data in dependency order: advisor → households → clients → accounts → RESP beneficiaries → contributions → transfers → goals → advisor notes.
- When `Load(ctx)` is called, the system shall emit pre-computed events (PortfolioDrift, TaxLossOpportunity, DividendReceived, ContributionProcessed, TransferCompleted) after all database rows are inserted.
- When `Load(ctx)` is called and the database already contains seed data (e.g. advisor with id "adv1" exists), the system shall skip seeding and return without error (idempotent).
- If any repository write fails during `Load(ctx)`, then the system shall return an error and stop seeding (no partial state — caller should run in a transaction or treat as fatal startup failure).
- The system shall set transfer `last_status_change` to `now - days_in_stage` for each seeded transfer, so that computed fields (`days_in_current_stage`, `is_stuck`) produce correct values immediately.
- The system shall set all seeded timestamps in UTC.

## Decision Table

N/A

## Test Anchors

N/A — seed data is static. Correctness is verified by the downstream contexts' tests and by visual inspection on the dashboard.
