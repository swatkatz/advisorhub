# AdvisorHub — Architecture

## 1. Domain Entities

### Advisor

Single entry for prototype (Shruti K.). Has id, name, role, email.

### Household

Optional grouping for couples/families. Swati & Rohan are separate
clients in the "Gupta Family" household. James & Tanya are in "Williams
Family." Contribution limits are per-person. Some goals are
household-level (buying a home together).

### Client

Individual person. Key fields: name, email, date_of_birth (matters for
age milestones — RRIF at 71, CPP at 65, RESP strategy when child turns
17), household_id (nullable), last_meeting_date.

Health status (GREEN/YELLOW/RED) is **computed, not stored**. Derived
at query time from the client's most severe unresolved alert:
- RED: has unresolved CRITICAL alert
- YELLOW: highest unresolved alert is URGENT or ADVISORY
- GREEN: no active alerts, or INFO only

### Account

Belongs to a client. Fields: account_type (RRSP/TFSA/FHSA/RESP/NON_REG),
institution (Wealthsimple, RBC, TD, Desjardins...), balance, is_external
(WS accounts have real data, external are advisor-entered),
resp_beneficiary_id (nullable — only for RESP accounts),
fhsa_lifetime_contributions (stored running total for FHSA $40K lifetime
cap — only meaningful for FHSA accounts, 0 for others).

### ContributionRule (reference data — hardcoded)

| Account Type | Annual Limit                           | Lifetime Cap            | Deadline                                | Penalty            | CESG                |
| ------------ | -------------------------------------- | ----------------------- | --------------------------------------- | ------------------ | ------------------- |
| RRSP         | 18% earned income, max $32,490         | None                    | 60 days after year-end (Mar 3 for 2025) | 1%/month on excess | N/A                 |
| TFSA         | $7,000                                 | None (room accumulates) | Calendar year-end                       | 1%/month on excess | N/A                 |
| FHSA         | $8,000/year                            | $40,000                 | Calendar year-end                       | 1%/month on excess | N/A                 |
| RESP         | No annual limit (CESG on first $2,500) | $50,000/beneficiary     | None (CESG is calendar year)            | N/A                | 20% up to $500/year |

### TemporalRule (reference data — hardcoded)

Check types are a finite set of functions the scanner knows how to execute:

- `AGE_APPROACHING` — compares entity DOB against target age within a horizon
- `DEADLINE_WITH_ROOM` — checks if account deadline is within N days AND contribution room > 0
- `DAYS_SINCE` — checks if a date field on the entity exceeds a threshold
- `BALANCE_IDLE` — checks if cash balance above minimum has had no investment activity for N days

| Rule             | Check Type         | Entity          | Params                                       | Event Type          |
| ---------------- | ------------------ | --------------- | -------------------------------------------- | ------------------- |
| RRIF_CONVERSION  | AGE_APPROACHING    | Client          | age: 71, within_days: 365                    | AgeMilestone        |
| RESP_LAST_CESG   | AGE_APPROACHING    | RESPBeneficiary | age: 17, within_days: 365                    | AgeMilestone        |
| RRSP_DEADLINE    | DEADLINE_WITH_ROOM | Account         | account_type: RRSP, days_before: [30, 14, 7] | DeadlineApproaching |
| TFSA_DEADLINE    | DEADLINE_WITH_ROOM | Account         | account_type: TFSA, days_before: [30, 14, 7] | DeadlineApproaching |
| FHSA_DEADLINE    | DEADLINE_WITH_ROOM | Account         | account_type: FHSA, days_before: [30, 14, 7] | DeadlineApproaching |
| ENGAGEMENT_STALE | DAYS_SINCE         | Client          | field: last_meeting, threshold: 180          | EngagementStale     |
| CASH_UNINVESTED  | BALANCE_IDLE       | Account         | min_balance: 5000, idle_days: 30             | CashUninvested      |

The scanner is generic: iterate rules, fetch matching entities, dispatch to the appropriate check function, emit events for matches. Adding a new rule is adding a row, not writing new code (unless a new check type is needed).

### RESPBeneficiary

Per-child entity. Fields: client_id (the subscriber/owner — simplified
for prototype), name, date_of_birth, lifetime_contributions. Linked to
one or more RESP accounts via resp_beneficiary_id on Account.

### Contribution

Record of money into an account. Client, account, amount, date, tax_year.

### Transfer

Money moving between institutions. Source, destination (always WS),
account_type, amount, status pipeline (INITIATED → DOCUMENTS_SUBMITTED →
IN_REVIEW → IN_TRANSIT → RECEIVED → INVESTED), days_in_current_stage.

### Goal

Belongs to client or household. Name, target_amount, target_date,
progress_pct, status (ON_TRACK/BEHIND/AHEAD). Mocked for prototype.

### AdvisorNote

Append-only log per client. Date + text.

### Alert

Detailed in Section 2. Key fields: id, condition_key (dedup identity),
client_id, severity, category, status (OPEN/SNOOZED/ACTED/CLOSED),
snoozed_until, payload (JSON — mutable data), linked_action_item_ids,
draft_message.

### ActionItem

Tracked task shared between advisor and client. Fields: id, client_id,
alert_id (nullable — some are manually created), text, status
(PENDING/IN_PROGRESS/DONE/CLOSED), due_date, resolution_note.

### Event

Event bus envelope. Fields: id, event_type, entity_id, entity_type,
payload (JSON), source (REACTIVE/TEMPORAL/ANALYTICAL/SYSTEM),
timestamp. EntityID and EntityType are generic entity references
kept on the envelope for routing convenience — not domain-specific.

### Entity Relationships

```
Advisor ──(1:N)──▶ Client
Client  ──(N:1)──▶ Household (optional)
Client  ──(1:N)──▶ Account
Client  ──(1:N)──▶ Contribution
Client  ──(1:N)──▶ Transfer
Client  ──(1:N)──▶ Goal
Client  ──(1:N)──▶ AdvisorNote
Client  ──(1:N)──▶ Alert
Client  ──(1:N)──▶ ActionItem
Alert   ──(1:N)──▶ ActionItem (via linked_action_item_ids)
Account ──(N:1)──▶ ContributionRule (by account_type)
RESP Account ──(N:1)──▶ RESPBeneficiary
```

### Bounded Contexts

Each bounded context owns its entities, enums, and migrations. Cross-context references are by ID only. Contexts communicate through the event bus or through interfaces — never by importing each other's types directly.

| # | Context | Owns | Risk |
|---|---|---|---|
| 01 | client | Client, Household, Advisor, AdvisorNote, Goal. Migrations for these tables. | LOW |
| 02 | account | Account, AccountType, RESPBeneficiary. Migrations for these tables. | LOW |
| 03 | event-bus | EventEnvelope, EventSource, EntityType, pub/sub. | MEDIUM |
| 04 | contribution-engine | Contribution, ContributionRule, room calc, over-contribution detection, CESG gap detection. | HIGH |
| 05 | transfer-monitor | Transfer, TransferStatus, stage thresholds, stuck detection. | HIGH |
| 06 | temporal-scanner | TemporalRule, check functions (AGE_APPROACHING, DEADLINE_WITH_ROOM, DAYS_SINCE, BALANCE_IDLE), sweep orchestration. | HIGH |
| 07 | alert-generator | Event→alert mapping logic, AlertCategoryRule. Constructs `CreateAlertRequest` (type owned by alert-lifecycle). | MEDIUM |
| 08 | alert-lifecycle | Alert, AlertSeverity, AlertStatus, AlertEventType, HealthStatus (computed), `CreateAlertRequest`, `UpdateAlertWithSummaryAndDraft`, dedup, state machine, cascade close. | HIGH |
| 09 | alert-enhancer | LLM prompt/response logic. Constructs `UpdateAlertWithSummaryAndDraft` (type owned by alert-lifecycle). | MEDIUM |
| 10 | action-item-service | ActionItem, ActionItemStatus, CRUD, status transitions. | LOW |
| 11 | graphql-api | Resolvers, SSE subscriptions. Transport layer — no business logic. | MEDIUM |
| 12 | seed-data | Seed loader, pre-computed events. | LOW |
| 13 | frontend | React dashboard. | LOW |

Dependencies flow top to bottom. Each context only depends on contexts above it.

## 2. Alert System

### Severity Tiers

| Tier     | Meaning                                       | Examples                                                                      |
| -------- | --------------------------------------------- | ----------------------------------------------------------------------------- |
| CRITICAL | Client is actively losing money or blocked    | Over-contribution accruing penalties, transfer stuck 14+ days                 |
| URGENT   | Time-sensitive opportunity or required action | RRSP deadline approaching with room, CESG matching gap, RRIF conversion year  |
| ADVISORY | Worth discussing, advisor judgment needed     | Large uninvested cash, portfolio drift, stale engagement, tax-loss harvesting |
| INFO     | Status updates, no action needed              | Transfer completed, contribution processed, dividend received                 |

### Alert Categories

**Critical:** Over-contribution detected (RRSP, TFSA, FHSA), Transfer stuck

**Urgent:** Deadline approaching with room, CESG matching gap (RESP),
Age milestone requiring action (RRIF conversion, RESP last CESG year)

**Advisory:** Cash uninvested (30+ days), Portfolio drift (mocked),
Engagement stale (180+ days), Tax-loss harvesting opportunity (mocked),
Mortgage renewal approaching (mocked)

**Informational:** Transfer completed, Contribution processed,
Dividend received

### Condition Key

Every alert has a condition_key used for deduplication. Format:
`{alert_type}:{client_id}:{specifics}`

Examples:

- `overcontrib:c1:RRSP`
- `transfer_stuck:t1`
- `engagement_stale:c4`
- `deadline_approaching:c3:RRSP:2026`
- `cesg_gap:c1:resp_ben_1:2026`

### Lifecycle State Machine

```
         ┌──────────────────────────────────────┐
         │                                      │
         ▼                                      │
      [OPEN] ──(snooze)──▶ [SNOOZED] ──(expiry)─┘
         │                   │     │
         │(send/track)       │     │(condition resolves)
         ▼                   │     ▼
      [ACTED] ──(auto)──▶ [SNOOZED]  [CLOSED]
         │                              ▲
         │(condition resolves)          │
         └──────────────────────────────┘

  OPEN ──(condition resolves)──▶ CLOSED  (also applies)

  CLOSED is terminal. Same condition_key recurring = new alert.
  On transition to CLOSED: cascade close all linked ActionItems.
```

**Transitions:**

- OPEN → SNOOZED: advisor manually snoozes
- OPEN → ACTED: advisor sends message or creates ActionItem
- ACTED → SNOOZED: automatic, with category-specific snooze duration
- SNOOZED → OPEN: snooze expires AND condition still detected on next sweep
- Any non-CLOSED → CLOSED: underlying condition resolves
- CLOSED → (never reopened): new occurrence creates new alert

### Advisor Actions

Three actions available on non-INFO, non-CLOSED alerts:

**Send & Track:** Sends pre-drafted or custom message to client,
creates linked ActionItem, transitions alert to ACTED,
sets auto-snooze per category duration.

**Track Only:** Creates linked ActionItem without sending message.
Same state transition as Send & Track.

**Snooze:** Transitions to SNOOZED. For prototype, snoozed_until =
next sweep. In production, configurable duration.

INFO alerts are auto-sent. No advisor action needed. Displayed
with "Sent ✓" badge, dimmed in feed.

### Auto-Snooze Durations (after advisor action)

| Category             | Duration |
| -------------------- | -------- |
| Over-contribution    | 7 days   |
| Deadline approaching | 3 days   |
| Transfer stuck       | 5 days   |
| CESG matching gap    | 14 days  |
| Engagement stale     | 14 days  |
| Portfolio drift      | 14 days  |
| Cash uninvested      | 14 days  |
| All other            | 14 days  |

### Deduplication Logic

On new condition detected:

1. Query: find most recent alert WHERE condition_key matches
   AND status ≠ CLOSED
2. No match → create new OPEN alert
3. Match found OPEN → update payload (amounts, dates), no new alert
4. Match found SNOOZED + expired → transition to OPEN, update payload
5. Match found SNOOZED + not expired → update payload silently,
   keep snoozed
6. Match found ACTED → update payload silently, don't resurface
7. Match found CLOSED → create NEW alert (new occurrence)

### Cascade Close

When alert transitions to CLOSED:

1. Set alert.resolved_at = now
2. For each id in linked_action_item_ids:
   - Set ActionItem.status = CLOSED
   - Set ActionItem.resolved_at = now
   - Set ActionItem.resolution_note = "Auto-closed: {category}
     condition resolved on {date}"
3. Emit AlertClosed event → shows as INFO in feed:
   "Resolved: {summary}"

### Event Bus: Producers & Consumers

#### Producers

| Producer                   | Event Types Emitted                                                | Source Tag |
| -------------------------- | ------------------------------------------------------------------ | ---------- |
| Contribution Engine        | ContributionRecorded, OverContributionDetected, CESGGap            | REACTIVE   |
| Transfer Monitor           | TransferStatusChanged, TransferStuck, TransferCompleted            | REACTIVE   |
| Temporal Scanner           | DeadlineApproaching, AgeMilestone, EngagementStale, CashUninvested | TEMPORAL   |
| Analytical Engine (mocked) | PortfolioDrift, TaxLossOpportunity                                 | ANALYTICAL |
| Alert Lifecycle            | AlertCreated, AlertClosed, AlertUpdated                            | SYSTEM     |
| Seed Data Loader           | ContributionProcessed, DividendReceived                            | REACTIVE   |

#### Consumers

| Consumer         | Subscribes To                                                                                                                                                                                                        | What It Does                                                                                     |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| Alert Generator  | OverContributionDetected, TransferStuck, DeadlineApproaching, AgeMilestone, CESGGap, EngagementStale, CashUninvested, PortfolioDrift, TaxLossOpportunity, TransferCompleted, ContributionProcessed, DividendReceived | Maps event → proto-alert, forwards to Alert Lifecycle                                            |
| Alert Lifecycle  | (receives proto-alerts from Alert Generator, not directly from bus)                                                                                                                                                  | Dedup, state transitions, cascade close. Emits AlertCreated/AlertClosed/AlertUpdated back to bus |
| Alert Enhancer   | AlertCreated, AlertUpdated (where signal = REOPENED)                                                                                                                                                                 | Calls LLM, writes summary + draft_message onto the alert                                         |
| SSE Subscription | AlertCreated, AlertClosed, AlertUpdated                                                                                                                                                                              | Pushes to frontend via GraphQL subscription                                                      |

#### Event Flow Diagram

```
Contribution Engine ──▶ OverContributionDetected ──▶ Alert Generator ──▶ Alert Lifecycle ──▶ AlertCreated ──▶ Alert Enhancer
                        CESGGap                                                              AlertClosed      SSE → Frontend
                        ContributionRecorded                                                 AlertUpdated

Transfer Monitor ────▶ TransferStuck ──────────────▶ Alert Generator ──▶ (same)
                       TransferCompleted
                       TransferStatusChanged

Temporal Scanner ────▶ DeadlineApproaching ─────────▶ Alert Generator ──▶ (same)
                       AgeMilestone
                       EngagementStale
                       CashUninvested

Analytical Engine ───▶ PortfolioDrift ──────────────▶ Alert Generator ──▶ (same)
(mocked)               TaxLossOpportunity

Seed Data Loader ────▶ ContributionProcessed ───────▶ Alert Generator ──▶ (same)
                       DividendReceived
```

#### Events Sent to Frontend (via SSE)

Only three event types push to the frontend through the GraphQL subscription:

| Event Type   | Frontend Behavior                                              |
| ------------ | -------------------------------------------------------------- |
| AlertCreated | New alert card appears in feed                                 |
| AlertUpdated | Existing alert card updates (payload change, state transition) |
| AlertClosed  | Alert card transitions to resolved state with "Resolved" badge |

All other events are internal. The frontend never sees raw events like OverContributionDetected or DeadlineApproaching — it only sees the finished alert objects.

## 4. Real Components

These components contain actual business logic, not mocked data.

### Contribution Engine

Owns: Given a client's accounts and contribution history, compute
remaining room per account type, detect over-contributions, detect
CESG matching gaps, and calculate penalties.

**Inputs:**

- Client's accounts (including external)
- Contribution records for the tax year
- ContributionRule reference data

**Outputs (events emitted):**

- OverContributionDetected { client_id, account_type, limit,
  contributed, excess, penalty_per_month, institutions_involved[] }
- CESGGap { client_id, beneficiary_id, contributed_ytd,
  cesg_eligible_max, gap_amount, potential_grant_loss }
- ContributionProcessed { client_id, account_type, remaining_room }

**Core computations:**

1. **Room calculation** — sums all contributions for a client +
   account_type + tax_year across ALL institutions (internal + external),
   subtracts from annual limit. For RRSP, limit is the lesser of 18%
   of earned income or $32,490. For TFSA/FHSA, limit is flat.

2. **Over-contribution detection** — if contributed > limit, compute
   excess and penalty. Penalty = excess × 0.01 per month the excess
   persists. The engine emits the event with the full breakdown so
   the Alert Enhancer can generate something like: "Priya has
   over-contributed $2,300 to her RRSP across Wealthsimple ($18,860)
   and RBC ($15,000) against a $31,560 limit."

3. **CESG gap detection** — for RESP accounts, checks if
   contributed_ytd < $2,500 (CESG-eligible max). If gap exists,
   calculates: gap_amount = 2500 - contributed_ytd, potential grant
   loss = gap_amount × 0.20. Also checks lifetime cap ($50,000)
   to avoid recommending contributions that would exceed it.

4. **FHSA lifetime tracking** — checks cumulative contributions
   against $40,000 lifetime cap in addition to $8,000 annual limit.

5. **Deadline awareness** — doesn't emit deadline events directly
   (that's the Temporal Scanner's job via DEADLINE_WITH_ROOM check
   type), but provides the room data the scanner needs. The scanner
   calls the engine: "does client X have room in account type Y?"

**What it does NOT own:**

- Alert creation (Alert Generator's job)
- Deadline monitoring (Temporal Scanner's job)
- Client data or account balances (reads from domain, doesn't write)

**Edge cases to test:**

- Contributions split across 3+ institutions
- RRSP room based on earned income (not flat cap)
- FHSA annual limit hit before lifetime limit
- FHSA lifetime limit hit mid-year
- RESP contribution above CESG threshold but below lifetime cap
  (legal, just no additional grant)
- Zero room remaining (not over, just full)

### Transfer Monitor

Owns: Given transfer records, detect stuck transfers based on
days_in_current_stage exceeding expected thresholds per stage.

**Inputs:**

- Transfer records with status and last_status_change

**Outputs (events emitted):**

- TransferStuck { transfer_id, client_id, source_institution,
  account_type, amount, status, days_in_stage }
- TransferCompleted { transfer_id, client_id, amount }

**Thresholds (reference data):**

| Stage               | Expected Days | Stuck After |
| ------------------- | ------------- | ----------- |
| INITIATED           | 1-3           | 5           |
| DOCUMENTS_SUBMITTED | 3-7           | 10          |
| IN_REVIEW           | 3-10          | 14          |
| IN_TRANSIT          | 5-10          | 14          |
| RECEIVED            | 1-3           | 5           |

Logic is simple: for each non-INVESTED transfer, if
days_in_current_stage > stuck_after threshold, emit TransferStuck.
When status = INVESTED, emit TransferCompleted.

### Temporal Scanner

Owns: Iterates TemporalRules (from Section 1), evaluates check
functions against entities, emits events for matches.

**Inputs:**

- TemporalRule reference data
- Clients, accounts, RESP beneficiaries (for entity lookups)
- Contribution Engine (called for DEADLINE_WITH_ROOM check —
  "does this client have room?")

**Outputs:** Emits the event_type specified in the matching
TemporalRule (DeadlineApproaching, AgeMilestone, EngagementStale,
CashUninvested).

**Check functions:**

```go
AGE_APPROACHING(entity, params):
  age_at_year_end = year_end - entity.date_of_birth
  return age_at_year_end >= params.age
    AND days_until_year_end <= params.within_days

DEADLINE_WITH_ROOM(entity, params):
  deadline = compute_deadline(entity.account_type)
  days_until = deadline - today
  room = contribution_engine.GetRoom(entity.client_id, entity.account_type)
  return room > 0
    AND days_until IN params.days_before  // fires at 30, 14, 7

DAYS_SINCE(entity, params):
  days = today - entity[params.field]
  return days > params.threshold

BALANCE_IDLE(entity, params):
  return entity.balance >= params.min_balance
    AND days_since_last_activity(entity) > params.idle_days
```

**Triggered by:** `runMorningSweep` GraphQL mutation (button click
in dashboard). In production, would also run on a cron schedule.

### Alert Generator

Owns: Mapping raw domain events to proto-alerts. Pure lookup, no
business logic.

**Inputs:** Events from the bus (subscribes to all domain event types)
**Outputs:** Proto-alerts passed to Alert Lifecycle

For each event:

1. Look up AlertCategoryRule by event_type → get category, severity,
   draft_message flag
2. Construct condition_key from event payload (condition_key is
   NOT on the EventEnvelope — the Alert Generator builds it)
3. Extract client_id from event payload (or derive from EntityID
   when EntityType = Client)
4. Build alert payload from event payload
5. Create proto-alert: { condition_key, client_id, severity,
   category, status: OPEN, payload }
5. Pass to Alert Lifecycle

That's it. No computation, no rules, no LLM calls.

### Alert Lifecycle

Owns: Alert state machine, deduplication, cascade close.

**Inputs:** Proto-alerts from Alert Generator, advisor actions
from GraphQL mutations
**Outputs:** Persisted alerts, AlertCreated/AlertUpdated/AlertClosed
events, cascade close of ActionItems

Logic covered in Section 2. Returns a signal per operation:
CREATED, REOPENED, UPDATED, NO_CHANGE.

### Alert Enhancer

Owns: Natural language generation via LLM.

**Inputs:** Alerts where signal = CREATED or REOPENED
**Outputs:** Updates alert with summary and draft_message fields

**LLM prompt includes:**

- Alert category and severity
- Payload data (amounts, dates, institutions)
- Client name and context (recent advisor notes, account summary)

**Produces:**

- summary: 1-2 sentence advisor-facing description
- draft_message: suggested client email (only when
  AlertCategoryRule.draft_message = true)

### Event Bus

Owns: Pub/sub, event envelope, EntityType constants.

**Implementation:** Go channels. Map of event_type → subscriber
channels. Non-blocking publish via buffered channels.

```go
type EntityType string

const (
    EntityTypeClient          EntityType = "Client"
    EntityTypeAccount         EntityType = "Account"
    EntityTypeTransfer        EntityType = "Transfer"
    EntityTypeRESPBeneficiary EntityType = "RESPBeneficiary"
)

type EventEnvelope struct {
    ID         string
    Type       string
    EntityID   string          // ID of the domain entity this event concerns
    EntityType EntityType      // generic entity reference, not domain-specific
    Payload    json.RawMessage
    Source     EventSource     // REACTIVE | TEMPORAL | ANALYTICAL | SYSTEM
    Timestamp  time.Time
}
```

EntityID and EntityType are kept on the envelope for routing
convenience. Technically these belong in Payload — kept here for
ease of development, to be refactored later.

Note: condition_key is NOT on the envelope. It is constructed by
the Alert Generator when mapping events to proto-alerts.

In production, the bus would be backed by Kafka/NATS.

### ActionItem Service

Owns: CRUD for action items, status transitions.

**Inputs:** Create/update requests from GraphQL mutations,
cascade close commands from Alert Lifecycle
**Outputs:** Persisted ActionItems

Simple CRUD. Status transitions: PENDING → IN_PROGRESS → DONE.
Plus CLOSED (set only by cascade close from alert resolution).

## 5. Mocked Components

These components emit pre-computed events or serve static data.
They enter the same pipeline as real components — the Alert
Generator, Alert Lifecycle, and Alert Enhancer don't know the
difference.

### Seed Data Loader

Loads on startup. Populates the database with clients, accounts,
contribution histories, transfers, goals, notes, and RESP
beneficiaries. Also emits pre-computed events for scenarios that
would normally come from engines we haven't built.

**Pre-computed events emitted on startup:**

- PortfolioDrift { client_id: c9 (Sophie), drift_pct: 12,
  current_allocation: {tech: 42, ...}, target_allocation: {tech: 30, ...} }
- TaxLossOpportunity { client_id: c8 (Amir), holding: "Canadian Energy ETF",
  unrealized_loss: 3200 }
- DividendReceived { client_id: c9 (Sophie), amount: 1240 }
- ContributionProcessed { client_id: c6 (James), account_type: NON_REG,
  amount: 185000 }

These are fire-and-forget. They go through Alert Generator →
Alert Lifecycle → Alert Enhancer like anything else.

### Client Profiles (seeded)

| ID  | Name            | Household       | DOB        | AUM    | Last Meeting |
| --- | --------------- | --------------- | ---------- | ------ | ------------ |
| c1  | Priya Sharma    | —               | 1988-03-15 | $485K  | 2025-12-14   |
| c2  | Marcus Chen     | —               | 1955-11-08 | $1.25M | 2026-01-22   |
| c3  | Swati Gupta     | Gupta Family    | 1990-07-22 | $480K  | 2026-02-10   |
| c4  | Rohan Gupta     | Gupta Family    | 1989-01-30 | $240K  | 2026-02-10   |
| c5  | Elena Vasquez   | —               | 1982-09-11 | $310K  | 2025-09-05   |
| c6  | James Williams  | Williams Family | 1975-04-18 | $1.4M  | 2026-02-25   |
| c7  | Tanya Williams  | Williams Family | 1977-08-03 | $700K  | 2026-02-25   |
| c8  | Amir Patel      | —               | 1993-06-27 | $195K  | 2025-11-18   |
| c9  | Sophie Tremblay | —               | 1970-12-01 | $890K  | 2026-01-30   |
| c10 | David Kim       | —               | 1980-05-14 | $540K  | 2025-08-12   |

### Alert-Triggering Scenarios

Each client is designed to trigger specific alerts through
the pipeline:

**Priya (c1) — RED, 2 real + 0 mocked:**

- OverContributionDetected: RRSP contributed $33,860 (WS $18,860 +
  RBC $15,000) vs $31,560 limit. Excess $2,300, penalty $23/month.
- CESGGap: RESP contributed $1,800 ytd, needs $700 more for full
  $500 CESG match. Son's lifetime: $38,200 of $50K cap.
- External accounts: RBC RRSP $45K, RBC TFSA $12K

**Marcus (c2) — GREEN, 1 real:**

- AgeMilestone: Turns 71 in November 2026. RRIF conversion required
  by Dec 31. RRSP balance: $620K.

**Swati (c3) — YELLOW, 1 real + 1 mocked:**

- DeadlineApproaching: RRSP deadline 12 days away, $8,200 room
  remaining. On mat leave, mentioned wanting to maximize before
  returning.
- CashUninvested (via Rohan c4): Non-reg has $45,200 cash sitting
  uninvested for 34 days.

**Rohan (c4) — YELLOW:**

- Cash uninvested alert surfaces under household context.

**Elena (c5) — YELLOW, 1 real:**

- EngagementStale: Last meeting 178 days ago. Advisor note mentions
  mortgage renewal in April.

**James (c6) — GREEN, 1 mocked:**

- TransferCompleted (pre-computed): Non-reg transfer from Scotia
  ($185K) just completed. INFO alert.

**Tanya (c7) — GREEN, 1 mocked:**

- PortfolioDrift (pre-computed): Tech at 42% vs 30% target.

**Amir (c8) — RED, 1 real + 1 mocked:**

- TransferStuck: RRSP transfer from TD ($67,400) stuck at
  DOCUMENTS_SUBMITTED for 18 days (threshold: 10).
- TaxLossOpportunity (pre-computed): Canadian Energy ETF with
  $3,200 unrealized loss.

**Sophie (c9) — GREEN, 1 mocked:**

- DividendReceived (pre-computed): Quarterly dividend $1,240. INFO.

**David (c10) — YELLOW, 1 real:**

- EngagementStale: Last meeting 201 days ago.
- Note: Oldest child turns 17 next year — RESP strategy needed
  (this would also trigger AgeMilestone from temporal scanner on
  the RESPBeneficiary).

### Transfers (seeded)

| Client | Source        | Type    | Amount   | Status              | Days in Stage |
| ------ | ------------- | ------- | -------- | ------------------- | ------------- |
| Amir   | TD            | RRSP    | $67,400  | DOCUMENTS_SUBMITTED | 18            |
| James  | Scotia        | Non-Reg | $185,000 | INVESTED            | 0             |
| Priya  | RBC           | RRSP    | $42,000  | IN_TRANSIT          | 3             |
| David  | BMO           | TFSA    | $28,500  | IN_REVIEW           | 5             |
| Elena  | Desjardins    | RRSP    | $55,000  | INITIATED           | 2             |
| Sophie | National Bank | Non-Reg | $120,000 | IN_TRANSIT          | 6             |

### Goals (seeded, all mocked values)

| Client        | Goal                   | Target | Progress | Status   |
| ------------- | ---------------------- | ------ | -------- | -------- |
| Priya         | First home (FHSA)      | $120K  | 28%      | BEHIND   |
| Marcus        | Retirement at 65       | $2M    | 85%      | ON_TRACK |
| Swati + Rohan | First home             | $200K  | 45%      | ON_TRACK |
| Swati         | Mat leave savings      | $30K   | 90%      | AHEAD    |
| Elena         | Retirement at 60       | $800K  | 42%      | BEHIND   |
| James + Tanya | Kids education (RESP)  | $150K  | 68%      | ON_TRACK |
| Amir          | Emergency fund         | $25K   | 60%      | ON_TRACK |
| Sophie        | Early retirement at 58 | $1.5M  | 72%      | ON_TRACK |
| David         | Son's university       | $80K   | 55%      | BEHIND   |

### Advisor Notes (seeded, 2-3 per client)

Not specifying exact text here — seed loader generates
realistic notes like:

- "Discussed RRSP strategy. Swati wants to maximize before
  returning from mat leave."
- "Priya wasn't aware of employer RRSP contribution at RBC.
  Need to follow up on withdrawal options."
- "Marcus wants to discuss RRIF conversion options. Prefers
  gradual drawdown strategy."

## 6. GraphQL Schema

### Transport

- HTTP for queries and mutations
- SSE for subscriptions (`transport.SSE{}` in gqlgen)
- Single endpoint: `/graphql`

### Sorting

Repositories return data ordered by `id` by default. All user-facing sorting (clients by health/name/AUM, alerts by severity/recency, notes by date) is the responsibility of the GraphQL resolver layer, not the underlying repositories. This keeps repository interfaces simple and pushes presentation logic to the transport boundary.

### Schema

```graphql
type Query {
  clients(advisorId: ID!): [Client!]!
  client(id: ID!): Client!
  alerts(advisorId: ID!, filter: AlertFilter): [Alert!]!
  alert(id: ID!): Alert!
  contributionSummary(clientId: ID!, taxYear: Int!): ContributionSummary!
  transfers(advisorId: ID!): [Transfer!]!
  actionItems(clientId: ID): [ActionItem!]!
}

type Mutation {
  sendAlert(alertId: ID!, message: String): Alert!
  trackAlert(alertId: ID!, actionItemText: String!): Alert!
  snoozeAlert(alertId: ID!, until: DateTime): Alert!
  createActionItem(input: CreateActionItemInput!): ActionItem!
  updateActionItem(id: ID!, input: UpdateActionItemInput!): ActionItem!
  addNote(clientId: ID!, text: String!): AdvisorNote!
  runMorningSweep(advisorId: ID!): SweepResult!
}

type Subscription {
  alertFeed(advisorId: ID!): AlertEvent!
}

# --- Core Types ---

type Client {
  id: ID!
  name: String!
  email: String!
  dateOfBirth: Date!
  household: Household
  accounts: [Account!]!
  externalAccounts: [Account!]!
  aum: Float!
  lastMeeting: Date!
  health: HealthStatus!
  alerts: [Alert!]!
  actionItems: [ActionItem!]!
  goals: [Goal!]!
  notes: [AdvisorNote!]!
}

type Household {
  id: ID!
  name: String!
  members: [Client!]!
}

type Account {
  id: ID!
  accountType: AccountType!
  institution: String!
  balance: Float!
  isExternal: Boolean!
}

type Alert {
  id: ID!
  conditionKey: String!
  client: Client!
  severity: AlertSeverity!
  category: String!
  status: AlertStatus!
  snoozedUntil: DateTime
  summary: String!
  draftMessage: String
  linkedActionItems: [ActionItem!]!
  createdAt: DateTime!
  updatedAt: DateTime!
}

type ActionItem {
  id: ID!
  client: Client!
  alert: Alert
  text: String!
  status: ActionItemStatus!
  dueDate: Date
  createdAt: DateTime!
  resolvedAt: DateTime
  resolutionNote: String
}

type ContributionSummary {
  clientId: ID!
  taxYear: Int!
  accounts: [AccountContribution!]!
}

type AccountContribution {
  accountType: AccountType!
  annualLimit: Float!
  lifetimeCap: Float
  contributed: Float!
  remaining: Float!
  isOverContributed: Boolean!
  overAmount: Float
  penaltyPerMonth: Float
  deadline: Date
  daysUntilDeadline: Int
}

type Transfer {
  id: ID!
  client: Client!
  sourceInstitution: String!
  accountType: AccountType!
  amount: Float!
  status: TransferStatus!
  initiatedAt: Date!
  daysInCurrentStage: Int!
  isStuck: Boolean!
}

type Goal {
  id: ID!
  name: String!
  targetAmount: Float
  targetDate: Date
  progressPct: Int!
  status: GoalStatus!
}

type AdvisorNote {
  id: ID!
  date: Date!
  text: String!
}

type SweepResult {
  alertsGenerated: Int!
  alertsUpdated: Int!
  alertsSkipped: Int!
  duration: String!
}

type AlertEvent {
  type: AlertEventType!
  alert: Alert!
}

# --- Enums ---

enum AccountType {
  RRSP
  TFSA
  FHSA
  RESP
  NON_REG
}
enum AlertSeverity {
  CRITICAL
  URGENT
  ADVISORY
  INFO
}
enum AlertStatus {
  OPEN
  SNOOZED
  ACTED
  CLOSED
}
enum ActionItemStatus {
  PENDING
  IN_PROGRESS
  DONE
  CLOSED
}
enum TransferStatus {
  INITIATED
  DOCUMENTS_SUBMITTED
  IN_REVIEW
  IN_TRANSIT
  RECEIVED
  INVESTED
}
enum HealthStatus {
  GREEN
  YELLOW
  RED
}
enum GoalStatus {
  ON_TRACK
  BEHIND
  AHEAD
}
enum AlertEventType {
  CREATED
  UPDATED
  CLOSED
}

# --- Inputs ---

input AlertFilter {
  severity: AlertSeverity
  status: AlertStatus
  clientId: ID
}

input CreateActionItemInput {
  clientId: ID!
  alertId: ID
  text: String!
  dueDate: Date
}

input UpdateActionItemInput {
  text: String
  status: ActionItemStatus
  dueDate: Date
}
```

### Mutation Behavior

**sendAlert:** Transitions alert to ACTED, creates linked
ActionItem with the alert summary as text, sends message to
client (for prototype: just records that it was "sent").
Optional custom message overrides the draft.

**trackAlert:** Transitions alert to ACTED, creates linked
ActionItem with provided text. No message sent.

**snoozeAlert:** Transitions alert to SNOOZED. If `until` is
null, defaults to category-specific auto-snooze duration.

**runMorningSweep:** Triggers the Temporal Scanner + Transfer
Monitor. Events flow through the pipeline. New/updated alerts
push to the frontend via the alertFeed subscription. Returns
a summary of what happened.

### Subscription Behavior

**alertFeed:** SSE stream. Client connects with
`Accept: text/event-stream`. Receives AlertEvent objects
as alerts are created, updated, or closed. The AlertEventType
tells the frontend whether to add a new card, update an
existing one, or mark one as resolved.

## 7. Dashboard

Visual reference: `docs/mocks/advisor-dashboard.jsx`

### Layout

Sidebar (fixed) + main content area. Sidebar contains: advisor
name, navigation for three sections, and a badge count of
unresolved alerts.

### Sections

**Alert Feed** (primary view)

- Displays alerts sorted by severity (CRITICAL first), then
  by recency
- Filters: All, Needs Attention, Critical, Urgent, Advisory
- Each alert card shows: severity indicator, client name,
  category, timestamp, summary text
- Expandable draft message (editable text box)
- Three action buttons: Send, Track, Snooze
- INFO alerts shown dimmed with "Sent ✓" badge, no actions
- "Run Morning Sweep" button in header triggers
  `runMorningSweep` mutation

**Transfer Tracking**

- Kanban board with 6 columns matching TransferStatus enum:
  Initiated → Documents Submitted → In Review → In Transit →
  Received → Invested
- Each card shows: client name, source institution, account
  type, amount, days in stage
- Stuck transfers (isStuck = true) highlighted with warning
  indicator

**Client Book**

- Left panel: searchable table of clients. Columns: name,
  account type tags, AUM, last meeting, health dot, alert
  count badge
- Right panel (opens on row click): client detail with:
  - Contribution summary (from `contributionSummary` query,
    real computed data — progress bars, over-contribution in red)
  - External accounts list
  - Goals with progress bars
  - Action items with status
  - Advisor notes (chronological) with "Add note" input

### Key Interactions

| User Action             | Mutation          | UI Effect                                                             |
| ----------------------- | ----------------- | --------------------------------------------------------------------- |
| Click Send on alert     | `sendAlert`       | Alert card shows "Sent ✓", transitions to dimmed state                |
| Click Track on alert    | `trackAlert`      | Alert card shows "Tracked ✓", ActionItem appears in client detail     |
| Click Snooze on alert   | `snoozeAlert`     | Alert card removed from feed (or shown as snoozed)                    |
| Click Run Morning Sweep | `runMorningSweep` | Loading indicator, then new alerts stream in via SSE                  |
| Click client row        | (query)           | Right panel opens with `client`, `contributionSummary`, `actionItems` |
| Add note                | `addNote`         | Note appends to timeline                                              |
| Edit draft message      | (local state)     | Text box updates, sent on next `sendAlert` call                       |

### SSE Subscription Behavior

Frontend connects to `alertFeed` subscription on page load.
When AlertEvent arrives:

- CREATED → new alert card animates into feed at correct
  sort position
- UPDATED → existing card updates in place (payload changed)
- CLOSED → card transitions to resolved state with
  "Resolved" badge, then fades or moves to bottom

The "Run Morning Sweep" demo flow: advisor clicks button →
mutation triggers temporal scanner + transfer monitor →
events flow through pipeline → alerts are created/updated →
AlertEvents push via SSE → cards appear one by one in the
feed with a slight stagger.
