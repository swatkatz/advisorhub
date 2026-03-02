# Spec: GraphQL API

## Bounded Context

Owns: GraphQL resolvers for all queries, mutations, and subscriptions defined in `schema.graphql`. gqlgen configuration (`backend/gqlgen.yml`) and generated code (`backend/graph/`). SSE subscription infrastructure — subscribes to `AlertCreated`/`AlertUpdated`/`AlertClosed` on the event bus and pushes to connected frontends. HTTP server setup (`backend/server.go`) with dependency injection and route wiring. Computed field resolution: `Client.health` (delegates to AlertService), `Client.aum` (sums account balances), `Client.accounts`/`Client.externalAccounts` (filters by `is_external`), `Transfer.daysInCurrentStage`/`Transfer.isStuck` (delegates to Transfer entity's computed fields). All user-facing sorting (alerts by severity then recency, clients by health/name/AUM per architecture convention). Sweep orchestration — coordinates `TemporalScanner.RunSweep`, `TransferMonitor.CheckStuckTransfers`, and `ContributionEngine.AnalyzeClient` for the `runMorningSweep` mutation.

Does not own: Business logic — all mutations delegate to service interfaces. Data persistence — no database tables. Event production — does not emit domain events directly (delegates to AlertService, which emits alert lifecycle events). Alert state machine logic (alert context). Contribution room computation (contribution-engine context). Stuck detection logic (transfer-monitor context). Temporal rule evaluation (temporal-scanner context). Action item status validation (action-item-service context).

Depends on:
- event-bus: subscribes to `AlertCreated`, `AlertUpdated`, `AlertClosed` via `EventBus.Subscribe()` for SSE subscription push
- client: `ClientRepository.GetClient`, `GetClients`, `GetClientsByHouseholdID`; `HouseholdRepository.GetHousehold`, `GetHouseholdByClientID`; `GoalRepository.GetGoalsByClientID`; `AdvisorNoteRepository.GetNotes`, `AddNote`; `AdvisorRepository.GetAdvisor`
- account: `AccountRepository.GetAccount`, `GetAccountsByClientID`
- alert: `AlertService.Send`, `Track`, `Snooze`, `Acknowledge`, `ComputeHealthStatus`; `AlertRepository.GetAlert`, `GetAlertsByClientID`, `GetAlertsByAdvisorID`
- action-item-service: `ActionItemService.CreateActionItem`, `GetActionItem`, `GetActionItemsByClientID`, `GetActionItemsByAlertID`, `UpdateActionItem`
- contribution-engine: `ContributionEngine.GetContributionSummary`, `AnalyzeClient`
- transfer-monitor: `TransferRepository.GetTransfer`, `GetTransfersByClientID`; `TransferMonitor.CheckStuckTransfers`
- temporal-scanner: `TemporalScanner.RunSweep`

Produces:
- HTTP server on `/graphql` endpoint (queries + mutations over HTTP, subscriptions over SSE)
- No events emitted — this is a pure transport layer

Schema changes required (to existing `schema.graphql`):
1. Add `scalar Date` and `scalar DateTime` declarations (used throughout but not declared)
2. Add `acknowledgeAlert(alertId: ID!): Alert!` to Mutation type (documented in architecture, present in alert spec, missing from schema)

### Deployment: Railway

The server is deployed on Railway (Go service). Railway provides managed PostgreSQL, automatic HTTPS, and a reverse proxy. SSE subscriptions require specific configuration to work through Railway's proxy layer:

| Concern | Configuration |
|---|---|
| Proxy buffering | Set `X-Accel-Buffering: no` response header to disable Railway's nginx buffering for SSE streams |
| Caching | Set `Cache-Control: no-cache, no-transform` to prevent proxy caching/transformation of the event stream |
| Connection | Set `Connection: keep-alive` to maintain the long-lived SSE connection |
| Content-Type | Must be `text/event-stream; charset=UTF-8` for SSE recognition |
| Keepalive | Send heartbeat comments (`: keepalive\n\n`) every 15 seconds to avoid TCP idle timeout at Railway's proxy |
| Platform timeout | Railway enforces a 15-minute max HTTP request duration. Frontend must implement auto-reconnect with `Last-Event-ID` for seamless resumption after timeout |
| gqlgen transport order | `transport.SSE{}` must be registered **first**, before `transport.Options{}`, `transport.GET{}`, `transport.POST{}` |
| gqlgen keepalive | Configure `transport.SSE{KeepAlivePingInterval: 15 * time.Second}` — this handles the heartbeat automatically via gqlgen's built-in ping mechanism |

### Smoke-testing on Railway (curl)

```bash
# 1. Test a query
curl -s -X POST https://<app>.up.railway.app/graphql \
  -H "content-type: application/json" \
  -d '{"query":"{ clients(advisorId: \"a1\") { id name health } }"}'

# 2. Open an SSE subscription (terminal 1 — streams until Ctrl-C)
curl -N -X POST https://<app>.up.railway.app/graphql \
  -H "accept: text/event-stream" \
  -H "content-type: application/json" \
  -d '{"query":"subscription { alertFeed(advisorId: \"a1\") { type alert { id severity summary } } }"}'

# 3. Trigger events (terminal 2 — alerts stream into terminal 1)
curl -s -X POST https://<app>.up.railway.app/graphql \
  -H "content-type: application/json" \
  -d '{"query":"mutation { runMorningSweep(advisorId: \"a1\") { alertsGenerated duration } }"}'
```

## Contracts

### Input

**Queries — each resolver reads from context interfaces, applies sorting, and returns data:**

| Query | Resolver Logic |
|---|---|
| `clients(advisorId)` | `ClientRepository.GetClients(advisorID)`. Sort by health rank (RED=0 → YELLOW=1 → GREEN=2), then name ascending. Nested fields resolved lazily per Computed Fields table below. |
| `client(id)` | `ClientRepository.GetClient(id)`. Return error if not found. |
| `alerts(advisorId, filter)` | `AlertRepository.GetAlertsByAdvisorID(advisorID)`. Apply optional filter in-memory: match `severity`, `status`, `clientId` if provided. Sort by severity rank (CRITICAL=0 → URGENT=1 → ADVISORY=2 → INFO=3), then `created_at` descending. |
| `alert(id)` | `AlertRepository.GetAlert(id)`. Return error if not found. |
| `contributionSummary(clientId, taxYear)` | `ContributionEngine.GetContributionSummary(clientID, taxYear)`. Resolver enriches each `AccountContribution` with `lifetimeCap` from `AccountType.LifetimeCap()`. |
| `transfers(advisorId)` | `ClientRepository.GetClients(advisorID)`, then for each client `TransferRepository.GetTransfersByClientID(clientID)`. Aggregate and return all transfers (including INVESTED — kanban board needs all columns). |
| `actionItems(clientId)` | If `clientId` provided: `ActionItemService.GetActionItemsByClientID(clientID)`. If nil: iterate all advisor clients and aggregate their action items. |

**Mutations — each delegates to a service interface. No business logic in the resolver:**

| Mutation | Delegates To | Notes |
|---|---|---|
| `sendAlert(alertId, message)` | `AlertService.Send(ctx, alertID, message)` | Returns updated alert. Error if CLOSED or INFO. |
| `trackAlert(alertId, actionItemText)` | `AlertService.Track(ctx, alertID, actionItemText)` | Returns updated alert. Error if CLOSED or INFO. |
| `snoozeAlert(alertId, until)` | `AlertService.Snooze(ctx, alertID, until)` | Returns updated alert. Error if CLOSED or INFO. |
| `acknowledgeAlert(alertId)` | `AlertService.Acknowledge(ctx, alertID)` | Returns updated alert. Error if not INFO or already CLOSED. |
| `createActionItem(input)` | `ActionItemService.CreateActionItem(ctx, input.ClientID, input.AlertID, input.Text, input.DueDate)` | Returns created item with status PENDING. |
| `updateActionItem(id, input)` | `ActionItemService.UpdateActionItem(ctx, id, input.Text, input.Status, input.DueDate)` | Returns updated item. Error on invalid status transition. |
| `addNote(clientId, text)` | `AdvisorNoteRepository.AddNote(ctx, clientID, advisorID, text)` | `advisorID` pulled from context (hardcoded to `"a1"` for prototype). Returns created note. |
| `runMorningSweep(advisorId)` | Orchestrates three producers, aggregates results. See Behaviors section for full flow. | Returns `SweepResult`. |

**Subscription:**

| Subscription | Mechanism |
|---|---|
| `alertFeed(advisorId)` | Resolver subscribes to `AlertCreated`, `AlertUpdated`, `AlertClosed` on `EventBus.Subscribe()`. For each event received: (1) extract `alert_id` from payload, (2) fetch full alert via `AlertRepository.GetAlert(alertID)`, (3) verify alert's client belongs to the advisor via `ClientRepository.GetClient(clientID)`, (4) construct `AlertEvent{type, alert}` and push to the gqlgen subscription channel. On context cancellation (client disconnects), stop reading from bus channels. |

**Event bus events consumed (for subscription only):**

| Event Type | Used For |
|---|---|
| `AlertCreated` | Push new alert card to frontend — `AlertEvent{type: CREATED}` |
| `AlertUpdated` | Update existing alert card — `AlertEvent{type: UPDATED}` |
| `AlertClosed` | Transition alert to resolved — `AlertEvent{type: CLOSED}` |

### Output

**HTTP responses:** Standard GraphQL JSON envelope (`{"data": {...}, "errors": [...]}`) for queries and mutations.

**SSE stream:** For `alertFeed` subscription. Client connects via POST to `/graphql` with `Accept: text/event-stream`. Server responds with `Content-Type: text/event-stream; charset=UTF-8` and streams `data:` frames per SSE spec. gqlgen's `transport.SSE{}` handles framing automatically.

**No events emitted.** The resolver layer does not publish to the event bus. All event emission is delegated to service contexts (AlertService emits AlertCreated/AlertUpdated/AlertClosed).

### Data Model

**No database tables owned.** The GraphQL API is a transport layer — all persistence is delegated to context repositories/services.

**Schema changes to `schema.graphql`:**

Add at the top of the file:
```graphql
scalar Date
scalar DateTime
```

Add to Mutation type:
```graphql
acknowledgeAlert(alertId: ID!): Alert!
```

No other schema changes. All existing types, enums, and inputs are correct per the architecture.

**Scalar type mappings (in `gqlgen.yml`):**

| GraphQL Scalar | Go Type | Format | Timezone |
|---|---|---|---|
| `Date` | Custom `graph.Date` wrapping `time.Time` | Marshals as `"2006-01-02"`, unmarshals from `"YYYY-MM-DD"` string | N/A (date only) |
| `DateTime` | Custom `graph.DateTime` wrapping `time.Time` | RFC3339 with offset: `"2006-01-02T15:04:05-05:00"` | EST (`America/New_York`) |

Both custom scalars live in `backend/graph/scalars.go` (~40 lines total).

**Resolver struct (dependency injection container in `backend/graph/resolver.go`):**

```go
type Resolver struct {
    ClientRepo        client.ClientRepository
    HouseholdRepo     client.HouseholdRepository
    GoalRepo          client.GoalRepository
    NoteRepo          client.AdvisorNoteRepository
    AdvisorRepo       client.AdvisorRepository
    AccountRepo       account.AccountRepository
    AlertService      alert.AlertService
    AlertRepo         alert.AlertRepository
    ActionItemService actionitem.ActionItemService
    ContribEngine     contribution.ContributionEngine
    TransferRepo      transfer.TransferRepository
    TransferMonitor   transfer.TransferMonitor
    TemporalScanner   temporal.TemporalScanner
    EventBus          eventbus.EventBus
}
```

`server.go` constructs this with concrete implementations and passes it to `handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: &resolver}))`.

**Computed field resolution (nested resolvers — gqlgen generates these):**

| GraphQL Field | Resolver Logic | Source Interface |
|---|---|---|
| `Client.household` | `HouseholdRepository.GetHouseholdByClientID(client.ID)` — returns nil if no household | client |
| `Client.accounts` | `AccountRepository.GetAccountsByClientID(client.ID)` → filter `is_external = false` | account |
| `Client.externalAccounts` | `AccountRepository.GetAccountsByClientID(client.ID)` → filter `is_external = true` | account |
| `Client.aum` | `AccountRepository.GetAccountsByClientID(client.ID)` → sum all `balance` (internal + external) | account |
| `Client.health` | `AlertService.ComputeHealthStatus(client.ID)` | alert |
| `Client.alerts` | `AlertRepository.GetAlertsByClientID(client.ID)` — sort by severity rank then `created_at` DESC | alert |
| `Client.actionItems` | `ActionItemService.GetActionItemsByClientID(client.ID)` | action-item |
| `Client.goals` | `GoalRepository.GetGoalsByClientID(client.ID)` | client |
| `Client.notes` | `AdvisorNoteRepository.GetNotes(client.ID, advisorID)` — already sorted by date DESC per spec 01 | client |
| `Household.members` | `ClientRepository.GetClientsByHouseholdID(household.ID)` | client |
| `Alert.client` | `ClientRepository.GetClient(alert.ClientID)` | client |
| `Alert.linkedActionItems` | For each ID in `alert.LinkedActionItemIDs`: `ActionItemService.GetActionItem(id)` | action-item |
| `ActionItem.client` | `ClientRepository.GetClient(actionItem.ClientID)` | client |
| `ActionItem.alert` | If `actionItem.AlertID != nil`: `AlertRepository.GetAlert(alertID)`. Else nil. | alert |
| `Transfer.client` | `ClientRepository.GetClient(transfer.ClientID)` | client |

**Sorting rules (resolver responsibility per architecture convention):**

| Query / Field | Sort Order |
|---|---|
| `clients(advisorId)` | Health rank ASC (RED=0, YELLOW=1, GREEN=2), then name ASC |
| `alerts(advisorId)` | Severity rank ASC (CRITICAL=0, URGENT=1, ADVISORY=2, INFO=3), then `created_at` DESC |
| `Client.alerts` | Same as `alerts` query |
| `Client.notes` | Date DESC (handled by repository per spec 01) |
| `actionItems` | Status rank (PENDING=0, IN_PROGRESS=1, DONE=2, CLOSED=3), then `due_date` ASC nulls last |

All other lists return in repository default order (by `id`).

## State Machine

N/A — no state transitions in this context. All state machines are delegated to service contexts (AlertService for alert states, ActionItemService for action item states, TransferRepository for transfer status pipeline).

## Behaviors (EARS syntax)

**Server setup:**

- The system shall initialize a `time.Location` for `America/New_York` once at startup and use it for all timestamp generation.
- The system shall register gqlgen transports in order: `transport.SSE{KeepAlivePingInterval: 15 * time.Second}` first, then `transport.Options{}`, `transport.GET{}`, `transport.POST{}`.
- The system shall set response headers `X-Accel-Buffering: no`, `Cache-Control: no-cache, no-transform` on SSE connections for Railway proxy compatibility.
- The system shall serve a single HTTP endpoint at `/graphql` for queries, mutations, and subscriptions.

**Queries:**

- When `clients(advisorId)` is called, the system shall return all clients for that advisor via `ClientRepository.GetClients(advisorID)`, sorted by health rank ascending (RED=0, YELLOW=1, GREEN=2) then name ascending.
- When `client(id)` is called, the system shall return the client via `ClientRepository.GetClient(id)`. Where the client is not found, the system shall return a GraphQL error.
- When `alerts(advisorId, filter)` is called, the system shall return alerts via `AlertRepository.GetAlertsByAdvisorID(advisorID)`, sorted by severity rank ascending (CRITICAL=0, URGENT=1, ADVISORY=2, INFO=3) then `created_at` descending.
- Where `filter` is provided on the `alerts` query, the system shall apply matching in-memory: include only alerts where `severity` matches `filter.severity` (if set), `status` matches `filter.status` (if set), and `client_id` matches `filter.clientId` (if set). Filters are AND-combined.
- When `alert(id)` is called, the system shall return the alert via `AlertRepository.GetAlert(id)`. Where not found, the system shall return a GraphQL error.
- When `contributionSummary(clientId, taxYear)` is called, the system shall call `ContributionEngine.GetContributionSummary(clientID, taxYear)` and enrich each `AccountContribution` with `lifetimeCap` from `AccountType.LifetimeCap()`.
- When `transfers(advisorId)` is called, the system shall get all clients via `ClientRepository.GetClients(advisorID)`, then for each client call `TransferRepository.GetTransfersByClientID(clientID)`, and return the aggregated list (including INVESTED transfers — the kanban board needs all columns).
- When `actionItems(clientId)` is called and `clientId` is provided, the system shall return action items via `ActionItemService.GetActionItemsByClientID(clientID)`, sorted by status rank ascending (PENDING=0, IN_PROGRESS=1, DONE=2, CLOSED=3) then `due_date` ascending with nulls last.
- When `actionItems(clientId)` is called and `clientId` is nil, the system shall iterate all advisor clients and aggregate their action items with the same sort order.

**Mutations:**

- When `sendAlert(alertId, message)` is called, the system shall delegate to `AlertService.Send(ctx, alertID, message)` and return the updated alert.
- When `trackAlert(alertId, actionItemText)` is called, the system shall delegate to `AlertService.Track(ctx, alertID, actionItemText)` and return the updated alert.
- When `snoozeAlert(alertId, until)` is called, the system shall delegate to `AlertService.Snooze(ctx, alertID, until)` and return the updated alert.
- When `acknowledgeAlert(alertId)` is called, the system shall delegate to `AlertService.Acknowledge(ctx, alertID)` and return the updated alert.
- When `createActionItem(input)` is called, the system shall delegate to `ActionItemService.CreateActionItem(ctx, input.ClientID, input.AlertID, input.Text, input.DueDate)` and return the created action item.
- When `updateActionItem(id, input)` is called, the system shall delegate to `ActionItemService.UpdateActionItem(ctx, id, input.Text, input.Status, input.DueDate)` and return the updated action item.
- When `addNote(clientId, text)` is called, the system shall delegate to `AdvisorNoteRepository.AddNote(ctx, clientID, advisorID, text)` and return the created note. The `advisorID` is hardcoded to `"a1"` for the prototype (single advisor).
- When any mutation service call returns an error, the system shall return it as a GraphQL error. The resolver shall not swallow or transform service errors — they propagate as-is (e.g., "cannot send INFO alert", "invalid status transition").

**Sweep orchestration (`runMorningSweep`):**

- When `runMorningSweep(advisorId)` is called, the system shall:
  1. Record start time (`time.Now().In(est)`)
  2. Subscribe to `AlertCreated`, `AlertUpdated`, `AlertClosed` on the event bus (for counting outcomes)
  3. Call `ContributionEngine.AnalyzeClient(ctx, clientID, taxYear)` for each advisor client (emits events to bus)
  4. Call `TransferMonitor.CheckStuckTransfers(ctx)` (emits events to bus)
  5. Call `TemporalScanner.RunSweep(ctx, advisorID, time.Now().In(est))` (emits events to bus)
  6. Wait for event processing to settle (implementation detail — the in-memory bus delivers to buffered channels immediately, so a short drain period with timeout suffices)
  7. Count `AlertCreated` events received → `alertsGenerated`, `AlertUpdated` → `alertsUpdated`
  8. Compute `alertsSkipped` as total producer events emitted minus (`alertsGenerated` + `alertsUpdated`)
  9. Return `SweepResult{alertsGenerated, alertsUpdated, alertsSkipped, duration}`
- Where any individual producer call fails, the system shall log the error and continue with remaining producers — a single failure shall not abort the sweep.
- Where no events are produced (all clients fully analyzed, no stuck transfers, no temporal matches), the system shall return `SweepResult{0, 0, 0, duration}`.

**Subscription (`alertFeed`):**

- When `alertFeed(advisorId)` is subscribed, the system shall subscribe to `AlertCreated`, `AlertUpdated`, `AlertClosed` on the event bus and return a Go channel that gqlgen streams over SSE.
- When an event is received on any of the three bus channels, the system shall extract `alert_id` from the event payload, fetch the full alert via `AlertRepository.GetAlert(alertID)`, verify the alert's client belongs to the advisor (via `ClientRepository.GetClient(clientID)` and checking `advisorID`), and if matched, send an `AlertEvent{type, alert}` to the subscription channel.
- Where the alert's client does not belong to the subscribed advisor, the system shall drop the event silently.
- When the subscription context is cancelled (client disconnects), the system shall stop reading from the bus channels and close the subscription channel. The goroutine must not leak.
- The system shall map event types to `AlertEventType`: `AlertCreated` → `CREATED`, `AlertUpdated` → `UPDATED`, `AlertClosed` → `CLOSED`.

**Computed field resolution:**

- When `Client.health` is resolved, the system shall call `AlertService.ComputeHealthStatus(client.ID)` and return the result.
- When `Client.aum` is resolved, the system shall call `AccountRepository.GetAccountsByClientID(client.ID)` and return the sum of all `balance` values (internal + external accounts).
- When `Client.accounts` is resolved, the system shall call `AccountRepository.GetAccountsByClientID(client.ID)` and return only accounts where `is_external = false`.
- When `Client.externalAccounts` is resolved, the system shall call `AccountRepository.GetAccountsByClientID(client.ID)` and return only accounts where `is_external = true`.
- When `Client.household` is resolved, the system shall call `HouseholdRepository.GetHouseholdByClientID(client.ID)`. Where the client has no household, the system shall return nil.
- When `Client.alerts` is resolved, the system shall call `AlertRepository.GetAlertsByClientID(client.ID)` and sort by severity rank ascending then `created_at` descending.
- When `Client.actionItems` is resolved, the system shall call `ActionItemService.GetActionItemsByClientID(client.ID)`.
- When `Client.goals` is resolved, the system shall call `GoalRepository.GetGoalsByClientID(client.ID)`.
- When `Client.notes` is resolved, the system shall call `AdvisorNoteRepository.GetNotes(client.ID, advisorID)` (already sorted by date DESC per spec 01).
- When `Household.members` is resolved, the system shall call `ClientRepository.GetClientsByHouseholdID(household.ID)`.
- When `Alert.client` is resolved, the system shall call `ClientRepository.GetClient(alert.ClientID)`.
- When `Alert.linkedActionItems` is resolved, the system shall call `ActionItemService.GetActionItem(id)` for each ID in `alert.LinkedActionItemIDs` and return the results. Where `LinkedActionItemIDs` is empty, the system shall return an empty list.
- When `ActionItem.client` is resolved, the system shall call `ClientRepository.GetClient(actionItem.ClientID)`.
- When `ActionItem.alert` is resolved: if `actionItem.AlertID` is non-nil, the system shall call `AlertRepository.GetAlert(alertID)`. If nil, the system shall return nil.
- When `Transfer.client` is resolved, the system shall call `ClientRepository.GetClient(transfer.ClientID)`.

## Decision Table

**Alert filter application (in-memory, AND-combined):**

| `filter.severity` set? | `filter.status` set? | `filter.clientId` set? | Result |
|---|---|---|---|
| No | No | No | Return all alerts for advisor |
| Yes | No | No | Include only alerts matching severity |
| No | Yes | No | Include only alerts matching status |
| No | No | Yes | Include only alerts matching clientId |
| Yes | Yes | No | Include only alerts matching severity AND status |
| Yes | No | Yes | Include only alerts matching severity AND clientId |
| No | Yes | Yes | Include only alerts matching status AND clientId |
| Yes | Yes | Yes | Include only alerts matching all three |

**Sorting rank maps:**

| AlertSeverity | Rank (ascending) |
|---|---|
| CRITICAL | 0 |
| URGENT | 1 |
| ADVISORY | 2 |
| INFO | 3 |

| HealthStatus | Rank (ascending) |
|---|---|
| RED | 0 |
| YELLOW | 1 |
| GREEN | 2 |

| ActionItemStatus | Rank (ascending) |
|---|---|
| PENDING | 0 |
| IN_PROGRESS | 1 |
| DONE | 2 |
| CLOSED | 3 |

**Subscription event type mapping:**

| Bus Event Type | AlertEventType | Frontend Behavior |
|---|---|---|
| `AlertCreated` | `CREATED` | New alert card appears in feed |
| `AlertUpdated` | `UPDATED` | Existing alert card updates in place |
| `AlertClosed` | `CLOSED` | Alert card shows "Resolved" badge |

**Sweep result derivation:**

| Counted From | SweepResult Field |
|---|---|
| `AlertCreated` events received on bus during sweep | `alertsGenerated` |
| `AlertUpdated` events received on bus during sweep | `alertsUpdated` |
| Total producer events emitted − (generated + updated) | `alertsSkipped` |
| `time.Since(start).String()` | `duration` |

**Computed field source routing:**

| GraphQL Field | Computed From | Returns |
|---|---|---|
| `Client.aum` | Sum of all account balances (internal + external) | `Float!` |
| `Client.health` | `AlertService.ComputeHealthStatus` | `HealthStatus!` |
| `Client.accounts` | `GetAccountsByClientID` filtered `is_external = false` | `[Account!]!` |
| `Client.externalAccounts` | `GetAccountsByClientID` filtered `is_external = true` | `[Account!]!` |
| `ContributionSummary.accounts[].lifetimeCap` | `AccountType.LifetimeCap()` | `Float` (nil for RRSP/TFSA/NON_REG) |
| `Transfer.daysInCurrentStage` | Transfer entity computed field | `Int!` |
| `Transfer.isStuck` | Transfer entity computed field | `Boolean!` |

## Test Anchors

**Queries:**

1. Given an advisor with 3 clients (one RED, one YELLOW, one GREEN), when `clients(advisorId)` is called, then all 3 clients are returned sorted RED first, then YELLOW, then GREEN.
2. Given an advisor with 2 clients both GREEN, when `clients(advisorId)` is called, then they are returned sorted alphabetically by name.
3. Given a valid client ID, when `client(id)` is called, then the client is returned with all scalar fields populated.
4. Given an invalid client ID, when `client(id)` is called, then a GraphQL error is returned.
5. Given an advisor with 4 alerts (1 CRITICAL, 1 URGENT, 1 ADVISORY, 1 INFO), when `alerts(advisorId)` is called with no filter, then all 4 are returned sorted CRITICAL first, then URGENT, ADVISORY, INFO.
6. Given 2 CRITICAL alerts created at different times, when `alerts(advisorId)` is called, then within the CRITICAL group, the more recent alert appears first.
7. Given 4 alerts (2 CRITICAL, 1 URGENT, 1 ADVISORY), when `alerts(advisorId, filter: {severity: CRITICAL})` is called, then only the 2 CRITICAL alerts are returned.
8. Given 4 alerts across 2 clients, when `alerts(advisorId, filter: {clientId: "c1"})` is called, then only alerts for client c1 are returned.
9. Given alerts in mixed statuses, when `alerts(advisorId, filter: {severity: URGENT, status: OPEN})` is called, then only alerts matching both URGENT severity AND OPEN status are returned.
10. Given a valid alert ID, when `alert(id)` is called, then the alert is returned with all fields populated.
11. Given an invalid alert ID, when `alert(id)` is called, then a GraphQL error is returned.
12. Given client c1 with RRSP and FHSA accounts, when `contributionSummary(clientId: "c1", taxYear: 2026)` is called, then the result includes an `AccountContribution` entry for each account type, and the FHSA entry has `lifetimeCap = 40000`.
13. Given the `contributionSummary` result for an RRSP account, then `lifetimeCap` is null (no lifetime cap for RRSP).
14. Given an advisor with 3 clients, client c1 has 2 transfers and client c2 has 1 transfer (INVESTED), when `transfers(advisorId)` is called, then all 3 transfers are returned (including the INVESTED one).
15. Given a client with 3 action items (1 PENDING, 1 IN_PROGRESS, 1 DONE), when `actionItems(clientId)` is called, then they are returned sorted PENDING first, then IN_PROGRESS, then DONE.
16. Given 2 PENDING action items, one with `dueDate: 2026-03-10` and one with `dueDate: nil`, when `actionItems(clientId)` is called, then the one with a due date appears first (nulls last).

**Mutations:**

17. Given an OPEN CRITICAL alert, when `sendAlert(alertId, "custom message")` is called, then `AlertService.Send` is called with the alert ID and message, and the updated alert is returned.
18. Given an OPEN alert, when `trackAlert(alertId, "Follow up")` is called, then `AlertService.Track` is called and the updated alert is returned.
19. Given an OPEN alert, when `snoozeAlert(alertId, nil)` is called, then `AlertService.Snooze` is called with nil `until` (category default), and the updated alert is returned.
20. Given an OPEN INFO alert, when `acknowledgeAlert(alertId)` is called, then `AlertService.Acknowledge` is called and the CLOSED alert is returned.
21. Given `AlertService.Send` returns an error (e.g., "cannot send INFO alert"), when `sendAlert` is called, then the error is propagated as a GraphQL error.
22. Given valid input, when `createActionItem(input)` is called, then `ActionItemService.CreateActionItem` is called with `input.ClientID`, `input.AlertID`, `input.Text`, `input.DueDate` and the created action item is returned.
23. Given valid input, when `updateActionItem(id, input)` is called, then `ActionItemService.UpdateActionItem` is called with the ID and non-nil fields from input.
24. Given `ActionItemService.UpdateActionItem` returns an error (e.g., "invalid status transition"), when `updateActionItem` is called, then the error is propagated as a GraphQL error.
25. Given a valid client ID and text, when `addNote(clientId, text)` is called, then `AdvisorNoteRepository.AddNote` is called with `advisorID = "a1"` (prototype hardcoded) and the created note is returned.

**Sweep orchestration:**

26. Given an advisor with 2 clients, when `runMorningSweep(advisorId)` is called, then `ContributionEngine.AnalyzeClient` is called for each client, `TransferMonitor.CheckStuckTransfers` is called once, and `TemporalScanner.RunSweep` is called once.
27. Given the sweep produces 3 `AlertCreated` events and 1 `AlertUpdated` event on the bus, when `runMorningSweep` returns, then `SweepResult` has `alertsGenerated = 3` and `alertsUpdated = 1`.
28. Given no events are produced during the sweep, when `runMorningSweep` returns, then `SweepResult` has `alertsGenerated = 0`, `alertsUpdated = 0`, `alertsSkipped = 0`.
29. Given `ContributionEngine.AnalyzeClient` fails for one client but succeeds for another, when `runMorningSweep` is called, then the sweep continues for the remaining client and returns a result (partial success, not an error).
30. Given `runMorningSweep` is called, then `SweepResult.duration` is a non-empty string representing elapsed time (e.g., `"1.2s"`).

**Subscription:**

31. Given an active `alertFeed(advisorId: "a1")` subscription, when an `AlertCreated` event is published on the bus for a client belonging to advisor "a1", then the subscriber receives an `AlertEvent` with `type = CREATED` and the full alert object.
32. Given an active `alertFeed(advisorId: "a1")` subscription, when an `AlertUpdated` event is published, then the subscriber receives an `AlertEvent` with `type = UPDATED`.
33. Given an active `alertFeed(advisorId: "a1")` subscription, when an `AlertClosed` event is published, then the subscriber receives an `AlertEvent` with `type = CLOSED`.
34. Given an active `alertFeed(advisorId: "a1")` subscription, when an `AlertCreated` event is published for a client belonging to advisor "a2", then the subscriber does not receive the event (filtered out by advisor).
35. Given an active `alertFeed` subscription, when the client disconnects (context cancelled), then the subscription goroutine stops and does not leak.

**Computed fields:**

36. Given a client with 3 accounts (internal: $100K, $200K; external: $50K), when `Client.aum` is resolved, then $350,000 is returned (sum of all, including external).
37. Given a client with 2 internal accounts and 1 external account, when `Client.accounts` is resolved, then only the 2 internal accounts are returned.
38. Given a client with 2 internal accounts and 1 external account, when `Client.externalAccounts` is resolved, then only the 1 external account is returned.
39. Given a client with one non-CLOSED CRITICAL alert, when `Client.health` is resolved, then `RED` is returned.
40. Given a client with no household, when `Client.household` is resolved, then nil is returned.
41. Given an alert with 2 linked action item IDs, when `Alert.linkedActionItems` is resolved, then both action items are returned.
42. Given an alert with an empty `LinkedActionItemIDs`, when `Alert.linkedActionItems` is resolved, then an empty list is returned.
43. Given an action item with `alertID = nil`, when `ActionItem.alert` is resolved, then nil is returned.
44. Given an action item with a valid `alertID`, when `ActionItem.alert` is resolved, then the linked alert is returned.

**Timestamps:**

45. Given the server is running with `America/New_York` timezone, when a mutation creates a new entity (e.g., `addNote`), then the returned `DateTime` field includes the EST offset (e.g., `"2026-03-02T09:15:00-05:00"`).
46. Given a `Date` field on any entity, when it is serialized, then it marshals as `"YYYY-MM-DD"` with no time or timezone component.
