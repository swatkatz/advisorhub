# Spec: Alert

## Bounded Context

Owns: Alert entity, AlertSeverity/AlertStatus/AlertEventType enums, HealthStatus computation, AlertCategoryRule (hardcoded event→alert mapping table), condition_key construction, dedup logic, lifecycle state machine (OPEN/SNOOZED/ACTED/CLOSED transitions), cascade close, auto-snooze durations, Enhancer interface + Claude implementation (LLM summary + draft message generation). Database migration for the `alerts` table (starts empty — populated entirely by the pipeline at runtime).

Does not own: ActionItem entity or CRUD (action-item-service context — interacts via ActionItemService interface), event bus infrastructure (event-bus context — subscribes and publishes through it), client/account/note data (client and account contexts — reads via their repositories), domain event production (contribution-engine, transfer-monitor, temporal-scanner, seed-data contexts produce the events this context consumes).

Depends on:
- event-bus: subscribes to domain events via `EventBus.Subscribe()`, publishes AlertCreated/AlertUpdated/AlertClosed via `EventBus.Publish()`
- client: reads client name via `ClientRepository.GetClient()` (for enhancer LLM prompt context)
- client: reads recent advisor notes via `AdvisorNoteRepository.GetNotes()` (for enhancer LLM prompt context)
- action-item-service: creates ActionItems via `ActionItemService.CreateActionItem()` (on Send/Track), closes ActionItems via `ActionItemService.CloseActionItem()` (on cascade close)
- Anthropic Claude API (external): LLM calls for enhancement (summary + draft message)

Produces:
- Events (source: `SYSTEM`): `AlertCreated`, `AlertUpdated`, `AlertClosed`
- `AlertRepository` interface: FindByConditionKey, GetAlert, GetAlertsByClientID, GetAlertsByAdvisorID, CreateAlert, UpdateAlert
- `AlertService` interface: ProcessEvent (subscribe+map+lifecycle+enhance), Send, Track, Snooze, Acknowledge, Close, ComputeHealthStatus
- `Enhancer` interface: Enhance (behind interface for testability — stub writes deterministic string like `"enhanced:{alert_id}"` to verify wiring)

## Contracts

### Input

**Events consumed from the bus** (subscribes via `EventBus.Subscribe()`):

| Event Type | Source Context | Source Tag |
|---|---|---|
| OverContributionDetected | contribution-engine | REACTIVE |
| CESGGap | contribution-engine | REACTIVE |
| TransferStuck | transfer-monitor | REACTIVE |
| DeadlineApproaching | temporal-scanner | TEMPORAL |
| AgeMilestone | temporal-scanner | TEMPORAL |
| EngagementStale | temporal-scanner | TEMPORAL |
| CashUninvested | temporal-scanner | TEMPORAL |
| PortfolioDrift | seed-data (mocked) | ANALYTICAL |
| TaxLossOpportunity | seed-data (mocked) | ANALYTICAL |
| TransferCompleted | seed-data (mocked) | REACTIVE |
| ContributionProcessed | seed-data / contribution-engine | REACTIVE |
| DividendReceived | seed-data (mocked) | REACTIVE |

**Advisor actions from GraphQL mutations** (called by resolver layer):

- `AlertService.Send(ctx, alertID, message)` — from `sendAlert` mutation
- `AlertService.Track(ctx, alertID, actionItemText)` — from `trackAlert` mutation
- `AlertService.Snooze(ctx, alertID, until)` — from `snoozeAlert` mutation
- `AlertService.Acknowledge(ctx, alertID)` — from `acknowledgeAlert` mutation

**Data read from other contexts:**

- `ClientRepository.GetClient(ctx, clientID)` — client name for enhancer prompt
- `AdvisorNoteRepository.GetNotes(ctx, clientID, advisorID)` — recent notes for enhancer prompt

### Output

**Events emitted** (via `EventBus.Publish()`, source: `SYSTEM`):

All events use the `EventEnvelope` from the event-bus context. `EntityID` is the alert ID, `EntityType` is `EntityTypeClient` (since alerts are client-scoped — the client_id is in the payload for routing).

`AlertCreated`:
```json
{
  "alert_id": "a1",
  "client_id": "c1",
  "severity": "CRITICAL",
  "category": "over_contribution",
  "condition_key": "overcontrib:c1:RRSP",
  "summary": "...",
  "draft_message": "..."
}
```

`AlertUpdated`:
```json
{
  "alert_id": "a1",
  "client_id": "c1",
  "update_type": "PAYLOAD_UPDATED|STATUS_CHANGED|REOPENED",
  "status": "OPEN",
  "severity": "CRITICAL",
  "category": "over_contribution"
}
```

`AlertClosed`:
```json
{
  "alert_id": "a1",
  "client_id": "c1",
  "category": "over_contribution",
  "summary": "Resolved: ..."
}
```

**Interfaces exposed to other contexts:**

```go
type AlertRepository interface {
    FindByConditionKey(ctx context.Context, conditionKey string) (*Alert, error) // most recent WHERE status ≠ CLOSED
    GetAlert(ctx context.Context, id string) (*Alert, error)
    GetAlertsByClientID(ctx context.Context, clientID string) ([]Alert, error)
    GetAlertsByAdvisorID(ctx context.Context, advisorID string) ([]Alert, error)
    CreateAlert(ctx context.Context, alert *Alert) (*Alert, error)
    UpdateAlert(ctx context.Context, alert *Alert) (*Alert, error)
}

type AlertService interface {
    ProcessEvent(ctx context.Context, envelope eventbus.EventEnvelope) error
    Send(ctx context.Context, alertID string, message *string) (*Alert, error)
    Track(ctx context.Context, alertID string, actionItemText string) (*Alert, error)
    Snooze(ctx context.Context, alertID string, until *time.Time) (*Alert, error)
    Acknowledge(ctx context.Context, alertID string) (*Alert, error)
    Close(ctx context.Context, alertID string) (*Alert, error)
    ComputeHealthStatus(ctx context.Context, clientID string) (HealthStatus, error)
}

type Enhancer interface {
    Enhance(ctx context.Context, alert *Alert) error // writes summary + draft_message onto alert
}
```

`FindByConditionKey` returns the most recent alert matching the condition_key where status ≠ CLOSED, or nil if no match. This is the core dedup query.

`GetAlertsByAdvisorID` joins through client.advisor_id — needed for the `alerts(advisorId)` GraphQL query and the alert feed.

`ComputeHealthStatus` queries the client's most severe non-CLOSED alert and returns RED (CRITICAL), YELLOW (URGENT/ADVISORY), or GREEN (no alerts or INFO only).

### Data Model

**Alert** (persisted)

| Field | Type | Constraints |
|---|---|---|
| id | string (PK) | required |
| condition_key | string | required — dedup identity |
| client_id | string (FK → Client) | required |
| severity | AlertSeverity enum | required (CRITICAL, URGENT, ADVISORY, INFO) |
| category | string | required (e.g. "over_contribution", "transfer_stuck") |
| status | AlertStatus enum | required (OPEN, SNOOZED, ACTED, CLOSED) |
| snoozed_until | timestamp | nullable — set on snooze, cleared on reopen |
| payload | jsonb | required — mutable event data (amounts, dates, institutions) |
| summary | string | default empty — written by enhancer |
| draft_message | string | nullable — written by enhancer, only for categories with needs_draft=true |
| linked_action_item_ids | text[] | default empty — appended on Send/Track |
| created_at | timestamp | required, UTC |
| updated_at | timestamp | required, UTC |
| resolved_at | timestamp | nullable — set on CLOSED |

Indexes:
- `idx_alert_condition_key_status` on `(condition_key, status)` — dedup query: find most recent non-CLOSED alert by condition_key
- `idx_alert_client_id` on `client_id` — client detail view, health status computation
- `idx_alert_client_id_status_severity` on `(client_id, status, severity)` — health status computation (find most severe non-CLOSED alert)

**AlertCategoryRule** (hardcoded Go map, not a database table)

| Event Type | Category | Severity | Needs Draft | Auto-Snooze Duration |
|---|---|---|---|---|
| OverContributionDetected | over_contribution | CRITICAL | true | 7 days |
| TransferStuck | transfer_stuck | CRITICAL | true | 5 days |
| DeadlineApproaching | deadline_approaching | URGENT | true | 3 days |
| AgeMilestone | age_milestone | URGENT | true | 14 days |
| CESGGap | cesg_gap | URGENT | true | 14 days |
| EngagementStale | engagement_stale | ADVISORY | true | 14 days |
| CashUninvested | cash_uninvested | ADVISORY | true | 14 days |
| PortfolioDrift | portfolio_drift | ADVISORY | true | 14 days |
| TaxLossOpportunity | tax_loss_opportunity | ADVISORY | true | 14 days |
| TransferCompleted | transfer_completed | INFO | false | — |
| ContributionProcessed | contribution_processed | INFO | false | — |
| DividendReceived | dividend_received | INFO | false | — |

Each rule also has a condition_key builder function. Condition key formats:

| Category | Condition Key Format | Example |
|---|---|---|
| over_contribution | `overcontrib:{client_id}:{account_type}` | `overcontrib:c1:RRSP` |
| transfer_stuck | `transfer_stuck:{transfer_id}` | `transfer_stuck:t1` |
| deadline_approaching | `deadline_approaching:{client_id}:{account_type}:{tax_year}` | `deadline_approaching:c3:RRSP:2026` |
| age_milestone | `age_milestone:{entity_id}:{age}` | `age_milestone:c2:71` |
| cesg_gap | `cesg_gap:{client_id}:{beneficiary_id}:{tax_year}` | `cesg_gap:c1:resp_ben_1:2026` |
| engagement_stale | `engagement_stale:{client_id}` | `engagement_stale:c5` |
| cash_uninvested | `cash_uninvested:{account_id}` | `cash_uninvested:a15` |
| portfolio_drift | `portfolio_drift:{client_id}` | `portfolio_drift:c9` |
| tax_loss_opportunity | `tax_loss:{client_id}:{holding}` | `tax_loss:c8:canadian_energy_etf` |
| transfer_completed | `transfer_completed:{transfer_id}` | `transfer_completed:t2` |
| contribution_processed | `contribution_processed:{client_id}:{account_type}` | `contribution_processed:c6:NON_REG` |
| dividend_received | `dividend_received:{client_id}` | `dividend_received:c9` |

**HealthStatus** (computed, not persisted)

| Most Severe Non-CLOSED Alert | Health |
|---|---|
| CRITICAL | RED |
| URGENT or ADVISORY | YELLOW |
| INFO only, or no alerts | GREEN |

## State Machine

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
  OPEN ──(acknowledge, INFO only)──▶ CLOSED

  CLOSED is terminal. Same condition_key recurring = new alert.
  On transition to CLOSED: cascade close all linked ActionItems.
```

**Transitions:**

| From | To | Trigger | Guard | Side Effects |
|---|---|---|---|---|
| OPEN | SNOOZED | Advisor calls Snooze | severity ≠ INFO | Set `snoozed_until` (explicit or category default). Emit AlertUpdated. |
| OPEN | ACTED | Advisor calls Send or Track | severity ≠ INFO | Create linked ActionItem (via ActionItemService). Append ActionItem ID to `linked_action_item_ids`. Emit AlertUpdated. |
| ACTED | SNOOZED | Automatic (immediate after ACTED) | — | Set `snoozed_until` = now + category auto-snooze duration. Emit AlertUpdated. |
| SNOOZED | OPEN | ProcessEvent receives matching condition_key | `snoozed_until` has expired | Update payload. Emit AlertUpdated. Signal = REOPENED → trigger Enhancer. |
| SNOOZED | SNOOZED | ProcessEvent receives matching condition_key | `snoozed_until` has NOT expired | Update payload silently. No event emitted. Signal = UPDATED. |
| ACTED | ACTED | ProcessEvent receives matching condition_key | — | Update payload silently. No event emitted. Signal = UPDATED. |
| OPEN | OPEN | ProcessEvent receives matching condition_key | — | Update payload. Emit AlertUpdated. Signal = UPDATED. |
| OPEN | CLOSED | Close called (condition resolves) | — | Set `resolved_at`. Cascade close linked ActionItems. Emit AlertClosed. |
| SNOOZED | CLOSED | Close called (condition resolves) | — | Same as above. |
| ACTED | CLOSED | Close called (condition resolves) | — | Same as above. |
| OPEN | CLOSED | Advisor calls Acknowledge | severity = INFO | Set `resolved_at`. Emit AlertClosed. |
| (none) | OPEN | ProcessEvent, no existing non-CLOSED alert | — | Create new alert. Emit AlertCreated. Signal = CREATED → trigger Enhancer. |

**ProcessEvent signal values:**

| Scenario | Signal | Enhancer Called? |
|---|---|---|
| New alert created | CREATED | Yes |
| Snoozed alert reopened (expired) | REOPENED | Yes |
| Existing alert payload updated (any non-CLOSED status) | UPDATED | No |
| Duplicate, nothing changed | NO_CHANGE | No |

**INFO alert special case:** INFO alerts (TransferCompleted, ContributionProcessed, DividendReceived) are created as OPEN but have no Send/Track/Snooze actions. Their only transition is OPEN → CLOSED via Acknowledge (tick mark in UI). They display dimmed with "Sent ✓" badge.

## Behaviors (EARS syntax)

**Event→alert mapping:**

- When `ProcessEvent` receives an event, the system shall look up the `AlertCategoryRule` by event type to determine category, severity, needs_draft, and auto-snooze duration.
- When `ProcessEvent` receives an event with no matching `AlertCategoryRule`, the system shall log a warning and discard the event.
- When `ProcessEvent` receives an event with a matching rule, the system shall construct the `condition_key` using the rule's condition_key builder function and fields from the event payload.
- When `ProcessEvent` receives an event, the system shall extract `client_id` from the event payload. Where the event payload does not contain `client_id` explicitly (e.g. TransferStuck uses `transfer_id`), the system shall derive it from the `EntityID` on the envelope when `EntityType = Client`, or from the event payload's domain-specific fields.

**Dedup:**

- When `ProcessEvent` constructs a proto-alert, the system shall query `AlertRepository.FindByConditionKey` for the most recent alert with matching condition_key and status ≠ CLOSED.
- Where no existing alert is found, the system shall create a new OPEN alert, persist it, emit `AlertCreated`, and return signal CREATED.
- Where an existing alert is found with status OPEN, the system shall update the alert's payload and `updated_at`, emit `AlertUpdated`, and return signal UPDATED.
- Where an existing alert is found with status SNOOZED and `snoozed_until` has expired, the system shall transition to OPEN, update payload, emit `AlertUpdated`, and return signal REOPENED.
- Where an existing alert is found with status SNOOZED and `snoozed_until` has not expired, the system shall update payload and `updated_at` silently (no event emitted) and return signal UPDATED.
- Where an existing alert is found with status ACTED, the system shall update payload and `updated_at` silently (no event emitted) and return signal UPDATED.

**Enhancement:**

- When `ProcessEvent` returns signal CREATED or REOPENED, the system shall call `Enhancer.Enhance` with the alert.
- When `Enhancer.Enhance` is called, the system shall read the client's name via `ClientRepository.GetClient` and recent advisor notes via `AdvisorNoteRepository.GetNotes` to build LLM prompt context.
- When `Enhancer.Enhance` completes, the system shall write `summary` and (where `AlertCategoryRule.needs_draft = true`) `draft_message` onto the alert and persist the update.
- If `Enhancer.Enhance` fails (LLM error, timeout), the system shall log the error and leave `summary` empty and `draft_message` nil. The alert is still valid — the advisor sees the payload data without a natural language summary.
- When `ProcessEvent` returns signal UPDATED or NO_CHANGE, the system shall not call `Enhancer.Enhance`.

**Advisor actions — Send:**

- When `Send(alertID, message)` is called, the system shall transition the alert to ACTED.
- Where `message` is nil, the system shall use the alert's existing `draft_message`.
- When transitioning to ACTED via Send, the system shall create a linked ActionItem via `ActionItemService.CreateActionItem` with the message as text, append the ActionItem ID to `linked_action_item_ids`, and persist.
- When transitioning to ACTED, the system shall immediately transition to SNOOZED with `snoozed_until` = now + category auto-snooze duration.
- When Send completes, the system shall emit `AlertUpdated`.
- Where the alert status is CLOSED, the system shall return an error.
- Where the alert severity is INFO, the system shall return an error.

**Advisor actions — Track:**

- When `Track(alertID, actionItemText)` is called, the system shall transition the alert to ACTED.
- When transitioning to ACTED via Track, the system shall create a linked ActionItem via `ActionItemService.CreateActionItem` with the provided text, append the ActionItem ID to `linked_action_item_ids`, and persist.
- When transitioning to ACTED, the system shall immediately transition to SNOOZED with `snoozed_until` = now + category auto-snooze duration.
- When Track completes, the system shall emit `AlertUpdated`.
- Where the alert status is CLOSED, the system shall return an error.
- Where the alert severity is INFO, the system shall return an error.

**Advisor actions — Snooze:**

- When `Snooze(alertID, until)` is called, the system shall transition the alert to SNOOZED.
- Where `until` is nil, the system shall use the category's auto-snooze duration from `AlertCategoryRule`.
- When Snooze completes, the system shall emit `AlertUpdated`.
- Where the alert status is CLOSED, the system shall return an error.
- Where the alert severity is INFO, the system shall return an error.

**Advisor actions — Acknowledge:**

- When `Acknowledge(alertID)` is called, the system shall transition the alert to CLOSED and set `resolved_at`.
- Where the alert severity is not INFO, the system shall return an error.
- When Acknowledge completes, the system shall emit `AlertClosed`.

**Close (condition resolves):**

- When `Close(alertID)` is called, the system shall transition the alert to CLOSED and set `resolved_at`.
- When transitioning to CLOSED, the system shall cascade close all linked ActionItems by calling `ActionItemService.CloseActionItem` for each ID in `linked_action_item_ids` with resolution_note = `"Auto-closed: {category} condition resolved on {date}"`.
- When Close completes, the system shall emit `AlertClosed`.
- Where the alert is already CLOSED, the system shall return without error (idempotent).

**Health status:**

- When `ComputeHealthStatus(clientID)` is called, the system shall query the client's alerts where status ≠ CLOSED.
- Where any non-CLOSED alert has severity CRITICAL, the system shall return RED.
- Where the most severe non-CLOSED alert is URGENT or ADVISORY, the system shall return YELLOW.
- Where all alerts are CLOSED or INFO, or the client has no alerts, the system shall return GREEN.

## Decision Table

**Dedup decision on ProcessEvent:**

| Existing Alert Found? | Existing Status | Snooze Expired? | Result | Signal | Event Emitted | Enhancer? |
|---|---|---|---|---|---|---|
| No | — | — | Create new OPEN alert | CREATED | AlertCreated | Yes |
| Yes | OPEN | — | Update payload | UPDATED | AlertUpdated | No |
| Yes | SNOOZED | Yes | Transition to OPEN, update payload | REOPENED | AlertUpdated | Yes |
| Yes | SNOOZED | No | Update payload silently | UPDATED | — | No |
| Yes | ACTED | — | Update payload silently | UPDATED | — | No |
| Yes | CLOSED | — | Create new OPEN alert (new occurrence) | CREATED | AlertCreated | Yes |

**Advisor action guards:**

| Action | Severity = INFO? | Status = CLOSED? | Result |
|---|---|---|---|
| Send | Yes | — | Error |
| Send | No | Yes | Error |
| Send | No | No | ACTED → SNOOZED |
| Track | Yes | — | Error |
| Track | No | Yes | Error |
| Track | No | No | ACTED → SNOOZED |
| Snooze | Yes | — | Error |
| Snooze | No | Yes | Error |
| Snooze | No | No | SNOOZED |
| Acknowledge | No | — | Error |
| Acknowledge | Yes | Yes | Error |
| Acknowledge | Yes | No | CLOSED |

**Health status derivation:**

| Most Severe Non-CLOSED Alert | HealthStatus |
|---|---|
| CRITICAL | RED |
| URGENT | YELLOW |
| ADVISORY | YELLOW |
| INFO only | GREEN |
| No alerts | GREEN |

## Test Anchors

**Event→alert mapping:**

1. Given an `OverContributionDetected` event for client c1 with account_type RRSP, when `ProcessEvent` is called, then a new alert is created with severity CRITICAL, category "over_contribution", condition_key "overcontrib:c1:RRSP", and status OPEN.
2. Given a `DividendReceived` event for client c9, when `ProcessEvent` is called, then a new alert is created with severity INFO and category "dividend_received".
3. Given an event with an unrecognized event type, when `ProcessEvent` is called, then no alert is created and no error is returned.

**Dedup:**

4. Given no existing alert for condition_key "overcontrib:c1:RRSP", when `ProcessEvent` receives `OverContributionDetected` for c1 RRSP, then a new OPEN alert is created and signal is CREATED.
5. Given an existing OPEN alert for condition_key "overcontrib:c1:RRSP", when `ProcessEvent` receives a new `OverContributionDetected` for c1 RRSP with updated excess amount, then the existing alert's payload is updated, `updated_at` changes, and signal is UPDATED.
6. Given an existing SNOOZED alert for condition_key "transfer_stuck:t1" with `snoozed_until` in the past, when `ProcessEvent` receives `TransferStuck` for t1, then the alert transitions to OPEN, payload is updated, and signal is REOPENED.
7. Given an existing SNOOZED alert for condition_key "transfer_stuck:t1" with `snoozed_until` in the future, when `ProcessEvent` receives `TransferStuck` for t1, then the alert remains SNOOZED, payload is updated silently, and signal is UPDATED.
8. Given an existing ACTED alert for condition_key "cesg_gap:c1:resp_ben_1:2026", when `ProcessEvent` receives `CESGGap` for c1, then payload is updated silently and signal is UPDATED.
9. Given a CLOSED alert for condition_key "engagement_stale:c5", when `ProcessEvent` receives `EngagementStale` for c5, then a new OPEN alert is created (separate from the closed one) and signal is CREATED.

**Enhancement:**

10. Given `ProcessEvent` returns signal CREATED, when the enhancer is called, then `summary` and `draft_message` are written onto the alert (stub enhancer writes `"enhanced:{alert_id}"` for summary).
11. Given `ProcessEvent` returns signal REOPENED, when the enhancer is called, then `summary` and `draft_message` are updated on the alert.
12. Given `ProcessEvent` returns signal UPDATED, then the enhancer is not called.
13. Given an INFO alert (e.g. DividendReceived) with signal CREATED, when the enhancer is called, then `summary` is written but `draft_message` remains nil (needs_draft = false).
14. Given the enhancer fails (returns error), when `ProcessEvent` completes, then the alert is still persisted with empty summary and nil draft_message, and no error is returned to the caller.

**Advisor actions — Send:**

15. Given an OPEN alert with severity CRITICAL, when `Send(alertID, nil)` is called, then the alert transitions to ACTED then SNOOZED, a linked ActionItem is created with the alert's draft_message as text, the ActionItem ID is appended to `linked_action_item_ids`, and `snoozed_until` is set to now + 7 days (over_contribution auto-snooze).
16. Given an OPEN alert, when `Send(alertID, "custom message")` is called, then the ActionItem is created with "custom message" as text (not the draft_message).
17. Given a CLOSED alert, when `Send` is called, then an error is returned.
18. Given an INFO alert, when `Send` is called, then an error is returned.

**Advisor actions — Track:**

19. Given an OPEN alert with category "deadline_approaching", when `Track(alertID, "Follow up on RRSP contribution")` is called, then the alert transitions to ACTED then SNOOZED with `snoozed_until` = now + 3 days, and a linked ActionItem is created with the provided text.
20. Given a SNOOZED alert (not expired), when `Track` is called, then the alert transitions to ACTED then SNOOZED (snooze duration resets to category default).

**Advisor actions — Snooze:**

21. Given an OPEN alert with category "transfer_stuck", when `Snooze(alertID, nil)` is called, then the alert transitions to SNOOZED with `snoozed_until` = now + 5 days (category default).
22. Given an OPEN alert, when `Snooze(alertID, specificTime)` is called, then `snoozed_until` is set to the provided time.
23. Given a CLOSED alert, when `Snooze` is called, then an error is returned.

**Advisor actions — Acknowledge:**

24. Given an OPEN INFO alert, when `Acknowledge(alertID)` is called, then the alert transitions to CLOSED with `resolved_at` set, and `AlertClosed` is emitted.
25. Given a CRITICAL alert, when `Acknowledge` is called, then an error is returned.
26. Given an already-CLOSED INFO alert, when `Acknowledge` is called, then an error is returned.

**Close (cascade):**

27. Given an OPEN alert with 2 linked ActionItem IDs, when `Close(alertID)` is called, then the alert transitions to CLOSED, `resolved_at` is set, `ActionItemService.CloseActionItem` is called for both IDs with resolution_note containing the category and date, and `AlertClosed` is emitted.
28. Given an OPEN alert with no linked ActionItems, when `Close(alertID)` is called, then the alert transitions to CLOSED without cascade (no ActionItemService calls).
29. Given an already-CLOSED alert, when `Close` is called, then no error is returned (idempotent) and no event is emitted.

**Health status:**

30. Given client c1 has one CRITICAL non-CLOSED alert and one ADVISORY non-CLOSED alert, when `ComputeHealthStatus(c1)` is called, then RED is returned.
31. Given client c5 has one ADVISORY non-CLOSED alert and no CRITICAL/URGENT alerts, when `ComputeHealthStatus(c5)` is called, then YELLOW is returned.
32. Given client c6 has only CLOSED alerts, when `ComputeHealthStatus(c6)` is called, then GREEN is returned.
33. Given client c6 has one INFO non-CLOSED alert and no other alerts, when `ComputeHealthStatus(c6)` is called, then GREEN is returned.
34. Given a client with no alerts at all, when `ComputeHealthStatus` is called, then GREEN is returned.

**Events emitted:**

35. Given a new alert is created via `ProcessEvent`, then `AlertCreated` is published to the event bus with source SYSTEM, EntityID = alert ID, and payload containing alert_id, client_id, severity, category, and condition_key.
36. Given an alert transitions via Send/Track/Snooze, then `AlertUpdated` is published to the event bus.
37. Given an alert is closed via `Close` or `Acknowledge`, then `AlertClosed` is published to the event bus.
