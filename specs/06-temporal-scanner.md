# Spec: Temporal Scanner

## Bounded Context

Owns: TemporalRule (hardcoded reference data — not a database table), CheckType enum (`AGE_APPROACHING`, `DEADLINE_WITH_ROOM`, `DAYS_SINCE`, `BALANCE_IDLE`), check function implementations for each CheckType, sweep orchestration logic. No database tables — all rule definitions are Go constants.

Does not own: Client data (client context — reads via ClientRepository), account data (account context — reads via AccountRepository), RESP beneficiary data (account context — reads via RESPBeneficiaryRepository), contribution room calculation (contribution-engine context — calls `ContributionEngine.GetRoom()`), event bus infrastructure (event-bus context — publishes through it), alert creation or lifecycle (alert-generator and alert-lifecycle contexts — downstream of emitted events), transfer monitoring (transfer-monitor context — separate producer triggered in parallel by `runMorningSweep`).

Depends on:
- event-bus: publishes events via `EventBus.Publish()`
- client: reads clients via `ClientRepository.GetClients()` (for `AGE_APPROACHING` on Client, `DAYS_SINCE` on Client)
- account: reads accounts via `AccountRepository.GetAccountsByClientID()` (for `DEADLINE_WITH_ROOM`, `BALANCE_IDLE`)
- account: reads RESP beneficiaries via `RESPBeneficiaryRepository.GetRESPBeneficiariesByClientID()` (for `AGE_APPROACHING` on RESPBeneficiary)
- contribution-engine: calls `ContributionEngine.GetRoom()` for `DEADLINE_WITH_ROOM` check

Produces:
- Events (source: `TEMPORAL`): `DeadlineApproaching`, `AgeMilestone`, `EngagementStale`, `CashUninvested`
- `TemporalScanner` interface: `RunSweep` (triggered by `runMorningSweep` GraphQL mutation)

## Contracts

### Input

No events consumed from the bus. This context is a producer, not a consumer.

Computation is triggered via:
- `TemporalScanner.RunSweep()` — called by the `runMorningSweep` GraphQL mutation resolver

Data is read from other contexts (by interface, not import):
- `ClientRepository.GetClients(advisorID)` → all clients for the advisor (for `AGE_APPROACHING` on Client, `DAYS_SINCE` on Client)
- `AccountRepository.GetAccountsByClientID(clientID)` → all accounts per client (for `DEADLINE_WITH_ROOM`, `BALANCE_IDLE`)
- `RESPBeneficiaryRepository.GetRESPBeneficiariesByClientID(clientID)` → beneficiaries per client (for `AGE_APPROACHING` on RESPBeneficiary)
- `ContributionEngine.GetRoom(clientID, accountType, taxYear)` → remaining room (for `DEADLINE_WITH_ROOM`)

### Output

**Events emitted** (via `EventBus.Publish()`, source: `TEMPORAL`):

All events use `EventEnvelope` from event-bus. Event-specific data is in `Payload`.

`DeadlineApproaching` — Envelope: EntityID = client_id, EntityType = `EntityTypeClient`
```json
{
  "client_id": "c3",
  "account_type": "RRSP",
  "deadline": "2026-03-03",
  "days_until": 12,
  "room_remaining": 8200.00,
  "tax_year": 2026
}
```

`AgeMilestone` (Client) — Envelope: EntityID = client_id, EntityType = `EntityTypeClient`
```json
{
  "client_id": "c2",
  "name": "Marcus Chen",
  "rule": "RRIF_CONVERSION",
  "target_age": 71,
  "current_age": 70,
  "year_turning": 2026
}
```

`AgeMilestone` (RESPBeneficiary) — Envelope: EntityID = beneficiary_id, EntityType = `EntityTypeRESPBeneficiary`
```json
{
  "client_id": "c1",
  "beneficiary_id": "resp_ben_1",
  "name": "Arjun Sharma",
  "rule": "RESP_LAST_CESG",
  "target_age": 17,
  "current_age": 16,
  "year_turning": 2027
}
```

`EngagementStale` — Envelope: EntityID = client_id, EntityType = `EntityTypeClient`
```json
{
  "client_id": "c5",
  "last_meeting_date": "2025-09-05",
  "days_since": 178
}
```

`CashUninvested` — Envelope: EntityID = account_id, EntityType = `EntityTypeAccount`
```json
{
  "client_id": "c4",
  "account_id": "acc_xyz",
  "account_type": "NON_REG",
  "balance": 45200,
  "idle_days": 34
}
```

**Interface exposed:**

```go
type TemporalScanner interface {
    RunSweep(ctx context.Context, advisorID string, referenceDate time.Time) (*ScannerResult, error)
}
```

`referenceDate` replaces `time.Now()` for all date comparisons, enabling deterministic tests. The GraphQL resolver passes `time.Now()` in production.

### Data Model

**CheckType** (enum, not persisted)

Values: `AGE_APPROACHING`, `DEADLINE_WITH_ROOM`, `DAYS_SINCE`, `BALANCE_IDLE`

**TemporalRule** (hardcoded Go struct, not a database table)

| Field | Type | Description |
|-------|------|-------------|
| Name | string | Rule identifier, e.g. `"RRIF_CONVERSION"` |
| CheckType | CheckType | Which check function to dispatch to |
| EntityType | EntityType | Which entity type to iterate |
| Params | map[string]any | Check-specific parameters (documented per CheckType below) |
| EventType | string | Event type to emit on match |

**Params by CheckType:**

| CheckType | Expected Params |
|-----------|----------------|
| AGE_APPROACHING | `age` (int) — target age; `within_days` (int) — horizon window |
| DEADLINE_WITH_ROOM | `account_type` (string); `within_days` (int) — fire when deadline is within this many days |
| DAYS_SINCE | `field` (string) — entity date field to check; `threshold` (int) — days |
| BALANCE_IDLE | `min_balance` (float64); `idle_days` (int) |

**Hardcoded rules:**

| Name | CheckType | EntityType | Params | EventType |
|------|-----------|------------|--------|-----------|
| RRIF_CONVERSION | AGE_APPROACHING | Client | age: 71, within_days: 365 | AgeMilestone |
| RESP_LAST_CESG | AGE_APPROACHING | RESPBeneficiary | age: 17, within_days: 365 | AgeMilestone |
| RRSP_DEADLINE | DEADLINE_WITH_ROOM | Account | account_type: RRSP, within_days: 30 | DeadlineApproaching |
| TFSA_DEADLINE | DEADLINE_WITH_ROOM | Account | account_type: TFSA, within_days: 30 | DeadlineApproaching |
| FHSA_DEADLINE | DEADLINE_WITH_ROOM | Account | account_type: FHSA, within_days: 30 | DeadlineApproaching |
| ENGAGEMENT_STALE | DAYS_SINCE | Client | field: last_meeting_date, threshold: 180 | EngagementStale |
| CASH_UNINVESTED | BALANCE_IDLE | Account | min_balance: 5000, idle_days: 30 | CashUninvested |

Note: The architecture lists `days_before: [30, 14, 7]` for deadline rules. This spec simplifies to `within_days: 30` — the scanner fires when the deadline is within 30 days and room exists. The payload includes `days_until` so the alert-generator can map urgency. Dedup in alert-lifecycle prevents duplicate alerts on repeated sweeps.

**ScannerResult** (computed, not persisted)

| Field | Type |
|-------|------|
| EventsEmitted | int |
| RulesEvaluated | int |
| EntitiesChecked | int |
| Duration | time.Duration |

The GraphQL resolver aggregates `ScannerResult` with transfer-monitor and alert-lifecycle results to build the `SweepResult` GraphQL type.

## State Machine

N/A — the temporal scanner is stateless. Each sweep is a pure computation: read rules, fetch entities, evaluate check functions, emit events, return results. No persistent state or transitions.

## Behaviors (EARS syntax)

**Sweep orchestration:**
- When `RunSweep(advisorID, referenceDate)` is called, the system shall iterate all hardcoded TemporalRules, fetch the matching entities for each rule, dispatch to the appropriate check function, and emit events for matches.
- When `RunSweep` is called, the system shall fetch all clients via `ClientRepository.GetClients(advisorID)` once, then reuse the list across all rules.
- The system shall return a `ScannerResult` with counts of rules evaluated, entities checked, events emitted, and elapsed duration.

**AGE_APPROACHING check:**
- When evaluating an `AGE_APPROACHING` rule against an entity, the system shall compute the entity's age at the end of the reference year (`referenceDate.Year()`) using the entity's `date_of_birth`.
- Where the entity's age at year-end >= `params.age` AND the number of days from `referenceDate` to Dec 31 of the reference year <= `params.within_days`, the system shall emit the rule's `EventType` with the entity's details.
- Where the entity's age at year-end < `params.age`, the system shall skip that entity.

**DEADLINE_WITH_ROOM check:**
- When evaluating a `DEADLINE_WITH_ROOM` rule, the system shall group accounts by `(client_id, account_type)` and evaluate once per client — since contribution room is per-client, not per-account.
- The system shall compute the deadline as: RRSP = 60 days after tax year end, TFSA/FHSA = Dec 31 of tax year.
- Where `days_until_deadline <= params.within_days` AND `ContributionEngine.GetRoom(clientID, accountType, taxYear) > 0`, the system shall emit `DeadlineApproaching` with `days_until`, `room_remaining`, and `deadline` in the payload.
- Where room = 0 (fully contributed), the system shall skip that client even if the deadline is within range.

**DAYS_SINCE check:**
- When evaluating a `DAYS_SINCE` rule against a client, the system shall compute the number of days between the client's `params.field` value and `referenceDate`.
- Where `days_since > params.threshold`, the system shall emit the rule's `EventType`.

**BALANCE_IDLE check:**
- When evaluating a `BALANCE_IDLE` rule against an account, the system shall check the account's balance and `last_activity_date`.
- Where `account.balance >= params.min_balance` AND days since `account.last_activity_date` > `params.idle_days`, the system shall emit `CashUninvested`.
- Where the account is external (`is_external = true`), the system shall skip it (no activity tracking for external accounts).

**General:**
- Where a check function encounters an error reading data (e.g., `GetRoom` fails), the system shall log the error and continue to the next entity — a single failure shall not abort the sweep.
- The system shall use `referenceDate` for all date comparisons, never `time.Now()`.

## Decision Table

**AGE_APPROACHING:**

| Age at Year-End vs Target | Days to Year-End <= within_days | Result |
|---|---|---|
| age >= target | Yes | Emit event |
| age >= target | No | Skip (outside horizon) |
| age < target | — | Skip (not yet reaching milestone) |

**DEADLINE_WITH_ROOM:**

| Days Until Deadline <= within_days | Room > 0 | Result |
|---|---|---|
| Yes | Yes | Emit DeadlineApproaching |
| Yes | No | Skip (fully contributed) |
| No | — | Skip (deadline not imminent) |

**DAYS_SINCE:**

| Days Since Field > threshold | Result |
|---|---|
| Yes | Emit event |
| No | Skip |

**BALANCE_IDLE:**

| Balance >= min_balance | Idle Days > idle_days | is_external | Result |
|---|---|---|---|
| Yes | Yes | false | Emit CashUninvested |
| Yes | Yes | true | Skip (no activity tracking for external) |
| Yes | No | — | Skip (recent activity) |
| No | — | — | Skip (balance below threshold) |

## Test Anchors

1. Given an advisor with clients that match multiple rules, when `RunSweep(advisorID, referenceDate)` is called, then all hardcoded rules are evaluated and `ScannerResult.RulesEvaluated` equals the total number of rules (7).
2. Given a client born 1955-11-08 and referenceDate = 2026-03-02, when the `RRIF_CONVERSION` rule (age: 71, within_days: 365) is evaluated, then `AgeMilestone` is emitted with target_age 71, current_age 70, year_turning 2026 (turns 71 at year-end, within 365-day horizon).
3. Given a client born 1990-07-22 and referenceDate = 2026-03-02, when the `RRIF_CONVERSION` rule is evaluated, then no event is emitted (age at year-end = 36, below target 71).
4. Given a RESP beneficiary born 2009-05-10 and referenceDate = 2025-06-01, when the `RESP_LAST_CESG` rule (age: 17, within_days: 365) is evaluated, then `AgeMilestone` is emitted with target_age 17, year_turning 2026.
5. Given a RESP beneficiary born 2015-04-20 and referenceDate = 2026-03-02, when the `RESP_LAST_CESG` rule is evaluated, then no event is emitted (age at year-end = 11, below target 17).
6. Given a client with an RRSP account, referenceDate = 2026-02-19 (12 days before RRSP deadline Mar 3), and `GetRoom` returns $8,200, when the `RRSP_DEADLINE` rule (within_days: 30) is evaluated, then `DeadlineApproaching` is emitted with days_until 12 and room_remaining 8200.
7. Given a client with an RRSP account, referenceDate = 2026-02-19, and `GetRoom` returns 0, when the `RRSP_DEADLINE` rule is evaluated, then no event is emitted (fully contributed, no room).
8. Given a client with an RRSP account and referenceDate = 2026-01-15 (47 days before deadline), when the `RRSP_DEADLINE` rule (within_days: 30) is evaluated, then no event is emitted (deadline outside 30-day window).
9. Given a client with 2 RRSP accounts (WS + RBC) and `GetRoom` returns $5,000, when the `RRSP_DEADLINE` rule is evaluated, then exactly one `DeadlineApproaching` event is emitted for that client (grouped by client, not per-account).
10. Given a client with last_meeting_date = 2025-09-05 and referenceDate = 2026-03-02 (178 days), when the `ENGAGEMENT_STALE` rule (threshold: 180) is evaluated, then no event is emitted (178 < 180).
11. Given a client with last_meeting_date = 2025-08-12 and referenceDate = 2026-03-02 (202 days), when the `ENGAGEMENT_STALE` rule (threshold: 180) is evaluated, then `EngagementStale` is emitted with days_since 202.
12. Given an internal account with balance $45,200, last_activity_date 34 days before referenceDate, when the `CASH_UNINVESTED` rule (min_balance: 5000, idle_days: 30) is evaluated, then `CashUninvested` is emitted with idle_days 34.
13. Given an external account with balance $50,000 and last_activity_date 60 days ago, when the `CASH_UNINVESTED` rule is evaluated, then no event is emitted (external accounts are skipped).
14. Given an internal account with balance $3,000 and last_activity_date 45 days ago, when the `CASH_UNINVESTED` rule (min_balance: 5000) is evaluated, then no event is emitted (balance below threshold).
15. Given `GetRoom` returns an error for one client during a `DEADLINE_WITH_ROOM` check, when `RunSweep` is called with multiple clients, then the failing client is skipped, remaining clients are still evaluated, and `ScannerResult` reflects the successful checks.
16. Given two different reference dates, when the same `DAYS_SINCE` rule is evaluated for the same client, then `days_since` in the payload reflects the difference from each respective referenceDate (proving `time.Now()` is not used).
