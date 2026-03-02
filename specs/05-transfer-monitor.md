# Spec: Transfer Monitor

## Bounded Context

Owns: Transfer entity, TransferStatus enum, stage threshold reference data (hardcoded), stuck detection logic. Database migrations for the `transfers` table.

Does not own: Client data (client context — references `client_id` via FK but does not import the client package), account data (account context), alert creation or lifecycle (alert-generator and alert-lifecycle contexts), event bus infrastructure (event-bus context — publishes through it), morning sweep orchestration (temporal-scanner context triggers the monitor).

Depends on:
- event-bus: publishes events via `EventBus.Publish()`

Produces:
- Events (source: `REACTIVE`): `TransferStuck`, `TransferCompleted`
- `TransferRepository` interface: GetTransfer, GetTransfersByClientID, GetActiveTransfers, CreateTransfer, UpdateTransferStatus
- `TransferMonitor` interface: CheckStuckTransfers (called by seed data loader on startup and by `runMorningSweep` — iterates active transfers, emits `TransferStuck` for stuck ones)

## Contracts

### Input

No events consumed from the bus. This context is a producer, not a consumer.

Data is written via:
- Seed data loader (bulk inserts transfers on startup)

Computation is triggered via:
- `TransferMonitor.CheckStuckTransfers()` — called by seed data loader after setup, and by `runMorningSweep` GraphQL resolver during sweep

### Output

**Events emitted** (via `EventBus.Publish()`, source: `REACTIVE`):

All events use the `EventEnvelope` from the event-bus context. `EntityID` is the transfer ID, `EntityType` is `EntityTypeTransfer`. Event-specific data is in the `Payload`.

`TransferStuck`:
```json
{
  "transfer_id": "t1",
  "client_id": "c8",
  "source_institution": "TD",
  "account_type": "RRSP",
  "amount": 67400,
  "status": "DOCUMENTS_SUBMITTED",
  "days_in_stage": 18,
  "stuck_threshold": 10
}
```

`TransferCompleted`:
```json
{
  "transfer_id": "t2",
  "client_id": "c6",
  "source_institution": "Scotia",
  "account_type": "NON_REG",
  "amount": 185000
}
```
In production, emitted by `UpdateTransferStatus` when status reaches INVESTED. In prototype, James's TransferCompleted is a pre-computed event from the seed data loader.

`TransferStatusChanged`:
```json
{
  "transfer_id": "t1",
  "client_id": "c8",
  "previous_status": "INITIATED",
  "new_status": "DOCUMENTS_SUBMITTED",
  "source_institution": "TD",
  "account_type": "RRSP",
  "amount": 67400
}
```
Not emitted in prototype (no real status transitions). Would come from `UpdateTransferStatus` in production.

**Interfaces exposed to other contexts:**

```go
type TransferRepository interface {
    GetTransfer(ctx context.Context, id string) (*Transfer, error)
    GetTransfersByClientID(ctx context.Context, clientID string) ([]Transfer, error)
    GetActiveTransfers(ctx context.Context) ([]Transfer, error)
    CreateTransfer(ctx context.Context, transfer *Transfer) (*Transfer, error)
    UpdateTransferStatus(ctx context.Context, id string, newStatus TransferStatus) (*Transfer, error) // TODO: in production, called by institution webhooks
}

type TransferMonitor interface {
    CheckStuckTransfers(ctx context.Context) ([]TransferCheckResult, error)
}
```

`GetActiveTransfers` returns all transfers WHERE status != INVESTED. Used by `CheckStuckTransfers` internally and available for other callers.

`CheckStuckTransfers` returns a `[]TransferCheckResult` so the caller (`runMorningSweep`) can build a `SweepResult` summary. Each result indicates what happened: `STUCK_DETECTED` or `NO_CHANGE`.

### Data Model

**TransferStatus** (enum)

Values: `INITIATED`, `DOCUMENTS_SUBMITTED`, `IN_REVIEW`, `IN_TRANSIT`, `RECEIVED`, `INVESTED`

Ordered pipeline. Forward-only transitions (see State Machine section).

**StageThreshold** (hardcoded Go constants, not a database table)

| Stage | Expected Days | Stuck After |
|---|---|---|
| INITIATED | 1-3 | 5 |
| DOCUMENTS_SUBMITTED | 3-7 | 10 |
| IN_REVIEW | 3-10 | 14 |
| IN_TRANSIT | 5-10 | 14 |
| RECEIVED | 1-3 | 5 |

INVESTED is terminal — no threshold.

**Transfer** (persisted)

| Field | Type | Constraints |
|---|---|---|
| id | string (PK) | required |
| client_id | string (FK → Client) | required |
| source_institution | string | required |
| account_type | AccountType enum | required — denormalized for display without cross-context joins |
| amount | float64 | required, > 0 |
| status | TransferStatus enum | required, default INITIATED |
| initiated_at | date | required |
| last_status_change | timestamp (UTC) | required — set on every status transition, used to compute days_in_current_stage |

Computed fields (not stored):
- `days_in_current_stage`: `floor(now - last_status_change)` in days
- `is_stuck`: `days_in_current_stage > StageThreshold[status].StuckAfter`

Indexes: `idx_transfer_client_id` on `client_id` (primary query path — "all transfers for this client"), `idx_transfer_status` on `status` (active transfer lookup for stuck detection sweep).

**TransferCheckResult** (computed, not persisted — returned by `CheckStuckTransfers`)

| Field | Type |
|---|---|
| transfer_id | string |
| signal | CheckSignal enum: STUCK_DETECTED, NO_CHANGE |

## State Machine

```
[INITIATED] ──▶ [DOCUMENTS_SUBMITTED] ──▶ [IN_REVIEW] ──▶ [IN_TRANSIT] ──▶ [RECEIVED] ──▶ [INVESTED]
```

**Transitions:**

- Each transition is forward-only, one step at a time. No skipping stages, no backwards movement.
- INVESTED is terminal.
- In production, transitions are triggered by institution webhooks via `UpdateTransferStatus`. For prototype, transfers are seeded in their current stage — no transitions occur at runtime.

**Stuck detection overlay** (not a state transition — a computed property):

```
For each non-INVESTED transfer:
  days_in_stage = floor(now - last_status_change) in days
  threshold = StageThreshold[status].StuckAfter

  If days_in_stage > threshold → is_stuck = true, emit TransferStuck
  Else → is_stuck = false, no event
```

| Stage | Stuck After (days) |
|---|---|
| INITIATED | 5 |
| DOCUMENTS_SUBMITTED | 10 |
| IN_REVIEW | 14 |
| IN_TRANSIT | 14 |
| RECEIVED | 5 |
| INVESTED | — (terminal) |

## Behaviors (EARS syntax)

**Stuck detection:**
- When `CheckStuckTransfers()` is called, the system shall iterate all transfers where status != INVESTED.
- Where `days_in_current_stage > StageThreshold[status].StuckAfter`, the system shall emit `TransferStuck` with the transfer details, days_in_stage, and stuck_threshold.
- Where `days_in_current_stage <= StageThreshold[status].StuckAfter`, the system shall skip the transfer and return `NO_CHANGE`.

**Computed fields:**
- When a transfer is queried, the system shall compute `days_in_current_stage` as `floor(now - last_status_change)` in days.
- When a transfer is queried, the system shall compute `is_stuck` as `days_in_current_stage > StageThreshold[status].StuckAfter`.
- Where status = INVESTED, `is_stuck` shall be false.

**Repository operations:**
- When `GetTransfer(id)` is called, the system shall return the transfer with that ID, or an error if not found.
- When `GetTransfersByClientID(clientID)` is called, the system shall return all transfers belonging to that client.
- When `GetActiveTransfers()` is called, the system shall return all transfers where status != INVESTED.
- When `CreateTransfer(transfer)` is called, the system shall persist the transfer and set `last_status_change` to `initiated_at`.

**Status transitions (production stub):**
- When `UpdateTransferStatus(id, newStatus)` is called, the system shall validate that `newStatus` is exactly one stage forward from the current status.
- If the transition is invalid, the system shall return an error.
- The system shall update `status` and set `last_status_change` to now (UTC).

**Check results:**
- When `CheckStuckTransfers()` completes, the system shall return a `[]TransferCheckResult` with one entry per active transfer indicating `STUCK_DETECTED` or `NO_CHANGE`.

## Decision Table

**Stuck detection by stage:**

| Current Status | Days in Stage | Threshold | Result |
|---|---|---|---|
| INITIATED | <= 5 | 5 | NO_CHANGE |
| INITIATED | > 5 | 5 | STUCK_DETECTED → emit TransferStuck |
| DOCUMENTS_SUBMITTED | <= 10 | 10 | NO_CHANGE |
| DOCUMENTS_SUBMITTED | > 10 | 10 | STUCK_DETECTED → emit TransferStuck |
| IN_REVIEW | <= 14 | 14 | NO_CHANGE |
| IN_REVIEW | > 14 | 14 | STUCK_DETECTED → emit TransferStuck |
| IN_TRANSIT | <= 14 | 14 | NO_CHANGE |
| IN_TRANSIT | > 14 | 14 | STUCK_DETECTED → emit TransferStuck |
| RECEIVED | <= 5 | 5 | NO_CHANGE |
| RECEIVED | > 5 | 5 | STUCK_DETECTED → emit TransferStuck |
| INVESTED | — | — | Skipped (terminal) |

**Status transition validation (production):**

| Current Status | Allowed Next Status |
|---|---|
| INITIATED | DOCUMENTS_SUBMITTED |
| DOCUMENTS_SUBMITTED | IN_REVIEW |
| IN_REVIEW | IN_TRANSIT |
| IN_TRANSIT | RECEIVED |
| RECEIVED | INVESTED |
| INVESTED | — (terminal, no transition) |

## Test Anchors

1. Given a transfer with a valid ID, when `GetTransfer(id)` is called, then the correct transfer is returned with all fields populated.
2. Given an invalid transfer ID, when `GetTransfer(id)` is called, then an error is returned.
3. Given a client with 2 transfers, when `GetTransfersByClientID(clientID)` is called, then both transfers are returned.
4. Given a client with no transfers, when `GetTransfersByClientID(clientID)` is called, then an empty slice is returned without error.
5. Given 4 active transfers and 1 INVESTED transfer, when `GetActiveTransfers()` is called, then 4 transfers are returned (INVESTED excluded).
6. Given a transfer at DOCUMENTS_SUBMITTED for 18 days (threshold: 10), when `CheckStuckTransfers()` is called, then `TransferStuck` is emitted with days_in_stage=18 and stuck_threshold=10.
7. Given a transfer at IN_TRANSIT for 3 days (threshold: 14), when `CheckStuckTransfers()` is called, then no event is emitted and the result is NO_CHANGE.
8. Given a transfer at INITIATED for exactly 5 days (threshold: 5), when `CheckStuckTransfers()` is called, then no event is emitted (stuck requires strictly greater than threshold).
9. Given a transfer at INITIATED for 6 days (threshold: 5), when `CheckStuckTransfers()` is called, then `TransferStuck` is emitted.
10. Given an INVESTED transfer, when `CheckStuckTransfers()` is called, then the transfer is skipped entirely.
11. Given 3 active transfers where 1 is stuck and 2 are not, when `CheckStuckTransfers()` is called, then the result contains 1 STUCK_DETECTED and 2 NO_CHANGE entries, and exactly 1 `TransferStuck` event is emitted.
12. Given no active transfers, when `CheckStuckTransfers()` is called, then an empty result is returned and no events are emitted.
13. Given a transfer with `last_status_change` 10 days ago, when queried, then `days_in_current_stage` is 10.
14. Given a transfer at IN_REVIEW with `days_in_current_stage` of 15 (threshold: 14), when queried, then `is_stuck` is true.
15. Given a transfer at IN_REVIEW with `days_in_current_stage` of 8 (threshold: 14), when queried, then `is_stuck` is false.
16. Given a new transfer, when `CreateTransfer()` is called, then the transfer is persisted with `last_status_change` equal to `initiated_at`.
17. Given a transfer at INITIATED, when `UpdateTransferStatus(id, DOCUMENTS_SUBMITTED)` is called, then status is updated, `last_status_change` is set to now, and the previous status was INITIATED.
18. Given a transfer at INITIATED, when `UpdateTransferStatus(id, IN_TRANSIT)` is called (skipping stages), then an error is returned and the transfer is unchanged.
