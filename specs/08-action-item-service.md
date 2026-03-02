# Spec: Action Item Service

## Bounded Context

Owns: ActionItem entity. Database migration for the `action_items` table. CRUD operations, status transitions, and query interfaces for action items. ActionItemStatus enum.

Does not own: Alert data or alert state machine (alert context). Client data (client context). The GraphQL resolver layer that calls into this context. The decision of *when* to create an action item (that's driven by `sendAlert`/`trackAlert` mutations in the resolver layer, or cascade close from the alert context).

Depends on: nothing — this is a foundational data context. References `client_id` and `alert_id` as foreign key IDs only. Does not import the client or alert packages.

Produces:
- `ActionItemRepository` interface: CreateActionItem, GetActionItem, GetActionItemsByClientID, GetActionItemsByAlertID, UpdateActionItem
- `ActionItemService` interface: CreateActionItem, GetActionItem, GetActionItemsByClientID, GetActionItemsByAlertID, UpdateActionItem, CloseActionItem

## Contracts

### Input

No events consumed. This context does not subscribe to the event bus.

Data is written via:
- `ActionItemService.CreateActionItem()` — called by alert context (on Send/Track), or by `createActionItem` GraphQL mutation
- `ActionItemService.UpdateActionItem()` — called by `updateActionItem` GraphQL mutation
- `ActionItemService.CloseActionItem()` — called by alert context on cascade close
- Seed data loader (bulk insert on startup, if action items are pre-seeded)

### Output

Interfaces exposed to other contexts (alert context consumes the service interface):

```go
type ActionItemRepository interface {
    CreateActionItem(ctx context.Context, item *ActionItem) (*ActionItem, error)
    GetActionItem(ctx context.Context, id string) (*ActionItem, error)
    GetActionItemsByClientID(ctx context.Context, clientID string) ([]ActionItem, error)
    GetActionItemsByAlertID(ctx context.Context, alertID string) ([]ActionItem, error)
    UpdateActionItem(ctx context.Context, item *ActionItem) (*ActionItem, error)
}

type ActionItemService interface {
    CreateActionItem(ctx context.Context, clientID string, alertID *string, text string, dueDate *time.Time) (*ActionItem, error)
    GetActionItem(ctx context.Context, id string) (*ActionItem, error)
    GetActionItemsByClientID(ctx context.Context, clientID string) ([]ActionItem, error)
    GetActionItemsByAlertID(ctx context.Context, alertID string) ([]ActionItem, error)
    UpdateActionItem(ctx context.Context, id string, text *string, status *ActionItemStatus, dueDate *time.Time) (*ActionItem, error)
    CloseActionItem(ctx context.Context, id string, resolutionNote string) (*ActionItem, error)
}
```

`ActionItemService` wraps `ActionItemRepository` and adds status transition validation. The alert context depends on the service interface (not the repository directly).

`CloseActionItem` is a special-purpose method for cascade close: sets status = CLOSED, resolved_at = now, and resolution_note to the provided string. Separate from `UpdateActionItem` because cascade close bypasses normal status transition validation (CLOSED can only be set via this path).

### Data Model

**ActionItem**

| Field | Type | Constraints |
|---|---|---|
| id | string (PK) | required |
| client_id | string (FK → Client) | required |
| alert_id | string (FK → Alert) | nullable — nil for manually created action items |
| text | string | required |
| status | ActionItemStatus enum | required (PENDING, IN_PROGRESS, DONE, CLOSED) |
| due_date | date | nullable |
| created_at | timestamp | required, UTC |
| resolved_at | timestamp | nullable — set when status → DONE or CLOSED |
| resolution_note | string | nullable — set on cascade close or manual resolution |

Indexes:
- `idx_action_item_client_id` on `client_id` — client detail view query path (`actionItems(clientId)`)
- `idx_action_item_alert_id` on `alert_id` — cascade close query path (`GetActionItemsByAlertID`) and alert detail view

## State Machine

```
  [PENDING] ──(start)──▶ [IN_PROGRESS] ──(complete)──▶ [DONE]
      │                       │                           │
      │(cascade close)        │(cascade close)            │(cascade close)
      ▼                       ▼                           ▼
   [CLOSED]               [CLOSED]                    [CLOSED]

  CLOSED is terminal. Set only by cascade close from alert resolution.
  DONE is terminal for normal workflow. No transitions out of DONE except cascade close.
```

**Transitions:**

| From | To | Trigger | Side Effects |
|---|---|---|---|
| PENDING | IN_PROGRESS | `UpdateActionItem` with status=IN_PROGRESS | `updated_at` set |
| IN_PROGRESS | DONE | `UpdateActionItem` with status=DONE | `resolved_at` = now, `updated_at` set |
| PENDING | DONE | `UpdateActionItem` with status=DONE | `resolved_at` = now, `updated_at` set (skip IN_PROGRESS) |
| PENDING | CLOSED | `CloseActionItem` (cascade) | `resolved_at` = now, `resolution_note` set |
| IN_PROGRESS | CLOSED | `CloseActionItem` (cascade) | `resolved_at` = now, `resolution_note` set |
| DONE | CLOSED | `CloseActionItem` (cascade) | `resolved_at` = now, `resolution_note` set |

**Invalid transitions (return error):**

- DONE → PENDING, DONE → IN_PROGRESS (can't reopen completed items)
- IN_PROGRESS → PENDING (can't move backward)
- CLOSED → anything (terminal)
- Any status → CLOSED via `UpdateActionItem` (CLOSED only via `CloseActionItem`)

## Behaviors (EARS syntax)

**Create:**

- When `CreateActionItem(clientID, alertID, text, dueDate)` is called, the system shall create a new ActionItem with status PENDING, `created_at` = now (UTC), and return it.
- Where `alertID` is nil, the system shall create the ActionItem without an alert link (manually created action item).
- Where `dueDate` is nil, the system shall create the ActionItem with a null due_date.

**Read:**

- When `GetActionItem(id)` is called, the system shall return the action item with that ID, or an error if not found.
- When `GetActionItemsByClientID(clientID)` is called, the system shall return all action items for that client.
- When `GetActionItemsByAlertID(alertID)` is called, the system shall return all action items linked to that alert.
- Where a client has no action items, `GetActionItemsByClientID` shall return an empty slice without error.
- Where an alert has no linked action items, `GetActionItemsByAlertID` shall return an empty slice without error.

**Update:**

- When `UpdateActionItem(id, text, status, dueDate)` is called, the system shall apply only the non-nil fields to the existing action item.
- Where `status` is provided, the system shall validate the transition against the state machine before applying.
- Where the status transition is invalid, the system shall return an error without modifying the action item.
- When status transitions to DONE, the system shall set `resolved_at` = now (UTC).
- Where `status` is CLOSED, the system shall return an error (CLOSED only via `CloseActionItem`).
- Where the action item is already CLOSED, the system shall return an error for any update.

**Cascade close:**

- When `CloseActionItem(id, resolutionNote)` is called, the system shall set status = CLOSED, `resolved_at` = now (UTC), and `resolution_note` = the provided string.
- Where the action item is already CLOSED, the system shall return without error (idempotent).
- Where the action item is DONE, the system shall still transition to CLOSED (cascade overrides terminal DONE state).

## Decision Table

**Status transition validation on `UpdateActionItem`:**

| Current Status | Requested Status | Result |
|---|---|---|
| PENDING | IN_PROGRESS | Allowed |
| PENDING | DONE | Allowed (skip IN_PROGRESS) |
| PENDING | CLOSED | Error — use CloseActionItem |
| IN_PROGRESS | DONE | Allowed |
| IN_PROGRESS | PENDING | Error — can't move backward |
| IN_PROGRESS | CLOSED | Error — use CloseActionItem |
| DONE | PENDING | Error — terminal |
| DONE | IN_PROGRESS | Error — terminal |
| DONE | CLOSED | Error — use CloseActionItem |
| CLOSED | any | Error — terminal |

**`CloseActionItem` behavior by current status:**

| Current Status | Result | resolved_at | resolution_note |
|---|---|---|---|
| PENDING | → CLOSED | set to now | set |
| IN_PROGRESS | → CLOSED | set to now | set |
| DONE | → CLOSED | overwritten to now | set |
| CLOSED | no-op (idempotent) | unchanged | unchanged |

## Test Anchors

**Create:**

1. Given a valid clientID, alertID, text, and dueDate, when `CreateActionItem` is called, then an ActionItem is returned with status PENDING, `created_at` set, and all fields matching the input.
2. Given a nil alertID, when `CreateActionItem` is called, then an ActionItem is created with `alert_id` = nil.
3. Given a nil dueDate, when `CreateActionItem` is called, then an ActionItem is created with `due_date` = nil.

**Read:**

4. Given an existing action item, when `GetActionItem(id)` is called, then the correct action item is returned with all fields populated.
5. Given an invalid ID, when `GetActionItem(id)` is called, then an error is returned.
6. Given a client with 3 action items (2 linked to alerts, 1 manual), when `GetActionItemsByClientID(clientID)` is called, then all 3 are returned.
7. Given a client with no action items, when `GetActionItemsByClientID(clientID)` is called, then an empty slice is returned without error.
8. Given an alert with 2 linked action items, when `GetActionItemsByAlertID(alertID)` is called, then both are returned.
9. Given an alert with no linked action items, when `GetActionItemsByAlertID(alertID)` is called, then an empty slice is returned without error.

**Update — valid transitions:**

10. Given a PENDING action item, when `UpdateActionItem` is called with status=IN_PROGRESS, then status changes to IN_PROGRESS and `resolved_at` remains nil.
11. Given an IN_PROGRESS action item, when `UpdateActionItem` is called with status=DONE, then status changes to DONE and `resolved_at` is set to now.
12. Given a PENDING action item, when `UpdateActionItem` is called with status=DONE, then status changes to DONE and `resolved_at` is set (skip IN_PROGRESS allowed).
13. Given an existing action item, when `UpdateActionItem` is called with only text changed (status and dueDate nil), then only text is updated and status is unchanged.
14. Given an existing action item, when `UpdateActionItem` is called with only dueDate changed, then only due_date is updated.

**Update — invalid transitions:**

15. Given an IN_PROGRESS action item, when `UpdateActionItem` is called with status=PENDING, then an error is returned and no fields are modified.
16. Given a DONE action item, when `UpdateActionItem` is called with status=IN_PROGRESS, then an error is returned.
17. Given a DONE action item, when `UpdateActionItem` is called with status=PENDING, then an error is returned.
18. Given a CLOSED action item, when `UpdateActionItem` is called with any fields, then an error is returned.
19. Given a PENDING action item, when `UpdateActionItem` is called with status=CLOSED, then an error is returned (CLOSED only via CloseActionItem).

**Cascade close:**

20. Given a PENDING action item, when `CloseActionItem(id, "Auto-closed: over_contribution condition resolved on 2026-03-02")` is called, then status = CLOSED, `resolved_at` is set, and `resolution_note` matches the provided string.
21. Given an IN_PROGRESS action item, when `CloseActionItem` is called, then status = CLOSED and `resolved_at` is set.
22. Given a DONE action item with `resolved_at` already set, when `CloseActionItem` is called, then status = CLOSED and `resolved_at` is overwritten to now.
23. Given an already-CLOSED action item, when `CloseActionItem` is called, then no error is returned and no fields are modified (idempotent).
