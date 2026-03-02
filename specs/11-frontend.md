# Spec: Frontend

## Bounded Context

Owns: React + TypeScript dashboard application. All UI components, pages, routing, local state management, and styling. Apollo Client configuration (cache, links, SSE subscription transport). GraphQL code generation setup (graphql-codegen) producing typed hooks and TypeScript types from `schema.graphql`. Vite build configuration.

Does not own: Business logic — all computation happens server-side. GraphQL schema definition (`schema.graphql` is shared, owned by graphql-api context). Backend API or database. Alert state machine transitions, contribution calculations, transfer stuck detection — the frontend displays results, never computes them. Dockerfile (already exists at `frontend/Dockerfile` — multi-stage build with `serve`).

Depends on:
- graphql-api: all data fetched via `/graphql` endpoint (queries + mutations over HTTP, subscriptions over SSE)
- schema.graphql: shared schema used as input to graphql-codegen to generate TypeScript types and typed Apollo hooks

Produces:
- Static build output (`frontend/dist/`) served by `serve` on Railway
- No events emitted — this is a pure presentation layer

## Contracts

### Input

**GraphQL queries consumed (data fetched from backend):**

| Query | Used By | Fields Selected |
|---|---|---|
| `advisor(id)` | Sidebar — advisor info | `id, name, email, role` |
| `clients(advisorId)` | Client Book — client list table | `id, name, email, dateOfBirth, aum, lastMeeting, health, accounts { accountType }, alerts { id }` |
| `client(id)` | Client Book — detail panel | `id, name, email, dateOfBirth, aum, lastMeeting, health, household { id, name, members { id, name } }, accounts { id, accountType, institution, balance, isExternal }, externalAccounts { id, accountType, institution, balance }, alerts { id, severity, category, summary, status }, actionItems { id, text, status, dueDate, createdAt, resolvedAt, resolutionNote }, goals { id, name, targetAmount, targetDate, progressPct, status }, notes { id, date, text }` |
| `alerts(advisorId, filter)` | Alert Feed — alert card list | `id, conditionKey, client { id, name }, severity, category, status, snoozedUntil, summary, draftMessage, linkedActionItems { id, text, status }, createdAt, updatedAt` |
| `contributionSummary(clientId, taxYear)` | Client Book — contribution bars in detail panel | `clientId, taxYear, accounts { accountType, annualLimit, lifetimeCap, contributed, remaining, isOverContributed, overAmount, penaltyPerMonth, deadline, daysUntilDeadline }` |
| `transfers(advisorId)` | Transfer Tracking — kanban board | `id, client { id, name }, sourceInstitution, accountType, amount, status, initiatedAt, daysInCurrentStage, isStuck` |
| `actionItems(clientId)` | Client Book — action items in detail panel | `id, client { id, name }, alert { id }, text, status, dueDate, createdAt, resolvedAt, resolutionNote` |

**GraphQL mutations invoked (user actions):**

| Mutation | Triggered By | UI Effect |
|---|---|---|
| `sendAlert(alertId, message)` | "Send" button on alert card | Alert card shows "Sent ✓", transitions to dimmed state |
| `trackAlert(alertId, actionItemText)` | "Track" button on alert card | Alert card shows "Tracked ✓", action item appears in client detail |
| `snoozeAlert(alertId, until)` | "Snooze" button on alert card | Alert card removed from active feed (or shown as snoozed) |
| `acknowledgeAlert(alertId)` | Checkmark button on INFO alert | INFO alert transitions to CLOSED |
| `createActionItem(input)` | "Add action item" in client detail (stretch goal) | New action item appears in list |
| `updateActionItem(id, input)` | Status change on action item (stretch goal) | Action item status updates |
| `addNote(clientId, text)` | "Add note" button in client detail | Note appends to timeline |
| `runMorningSweep(advisorId)` | "Run Morning Sweep" button in alert feed header | Loading indicator, then new alerts stream in via SSE |

**GraphQL subscription consumed (real-time updates):**

| Subscription | Transport | Used By |
|---|---|---|
| `alertFeed(advisorId)` | SSE via `graphql-sse` library over Apollo link | Alert Feed — real-time alert card updates |

SSE events received:
- `CREATED` → new alert card animates into feed at correct sort position
- `UPDATED` → existing alert card updates in place (payload/status changed)
- `CLOSED` → alert card transitions to resolved state with "Resolved" badge

### Output

**No backend output.** The frontend is a pure presentation layer. User actions invoke mutations which return updated objects — the frontend updates its local Apollo cache from mutation responses.

### Data Model

**No database tables.** All data lives in Apollo Client's normalized in-memory cache, keyed by `__typename:id`.

**TypeScript types and hooks are generated — not hand-written.** graphql-codegen reads `schema.graphql` and `.graphql` operation files to produce:

- `frontend/src/generated/graphql.ts` — all TypeScript types (enums, input types, query/mutation result types) and typed Apollo hooks (`useClientsQuery`, `useAlertsQuery`, `useSendAlertMutation`, etc.)

**graphql-codegen configuration (`frontend/codegen.ts`):**

| Setting | Value |
|---|---|
| Schema source | `../schema.graphql` (shared schema at project root) |
| Documents | `src/**/*.graphql` (operation files co-located with components) |
| Output | `src/generated/graphql.ts` |
| Plugins | `typescript`, `typescript-operations`, `typescript-react-apollo` |
| Config | `withHooks: true`, `enumsAsTypes: true` |

**GraphQL operation files (`.graphql` files co-located with features):**

| File | Operations Defined |
|---|---|
| `src/features/alerts/operations.graphql` | `GetAlerts` query, `SendAlert` / `TrackAlert` / `SnoozeAlert` / `AcknowledgeAlert` mutations, `AlertFeed` subscription |
| `src/features/transfers/operations.graphql` | `GetTransfers` query |
| `src/features/clients/operations.graphql` | `GetClients` query, `GetClient` query, `GetContributionSummary` query, `GetActionItems` query, `AddNote` mutation, `GetAdvisor` query |
| `src/features/sweep/operations.graphql` | `RunMorningSweep` mutation |

**Apollo Client setup (`src/lib/apollo.ts`):**

- HTTP link: points to `VITE_API_URL` env var (e.g., `https://<backend>.up.railway.app/graphql`)
- SSE subscription link: uses `graphql-sse` library's `createClient` + a custom Apollo `Link` that bridges SSE events to Apollo's subscription observable
- Link split: subscriptions route through SSE link, queries/mutations through HTTP link
- Cache: `InMemoryCache` with default type policies. Normalized by `id` field on all types.

**Environment variables:**

| Variable | Purpose | Example |
|---|---|---|
| `VITE_API_URL` | Backend GraphQL endpoint URL | `http://localhost:8080/graphql` (dev), `https://<backend>.up.railway.app/graphql` (prod) |

**Directory structure:**

```
frontend/
├── Dockerfile                    ← already exists
├── package.json
├── tsconfig.json
├── vite.config.ts
├── vitest.config.ts
├── codegen.ts                    ← graphql-codegen config
├── index.html
├── src/
│   ├── main.tsx                  ← entry point, ApolloProvider
│   ├── App.tsx                   ← layout + routing (sidebar + main content)
│   ├── lib/
│   │   └── apollo.ts             ← Apollo Client setup (HTTP + SSE links)
│   ├── generated/
│   │   └── graphql.ts            ← codegen output (types + hooks) — DO NOT EDIT
│   ├── features/
│   │   ├── alerts/
│   │   │   ├── operations.graphql
│   │   │   ├── AlertFeed.tsx
│   │   │   └── AlertCard.tsx
│   │   ├── transfers/
│   │   │   ├── operations.graphql
│   │   │   ├── TransferTracking.tsx
│   │   │   └── TransferCard.tsx
│   │   └── clients/
│   │       ├── operations.graphql
│   │       ├── ClientBook.tsx
│   │       ├── ClientDetail.tsx
│   │       ├── ContributionBar.tsx
│   │       └── GoalBar.tsx
│   ├── components/               ← shared UI primitives
│   │   ├── Badge.tsx
│   │   ├── Avatar.tsx
│   │   ├── AccountTag.tsx
│   │   ├── SeverityDot.tsx
│   │   └── IconButton.tsx
│   ├── styles/
│   │   └── theme.ts              ← color tokens, severity/health color maps
│   └── test/
│       ├── server.ts             ← MSW setupServer()
│       ├── handlers.ts           ← default GraphQL mock handlers
│       ├── fixtures.ts           ← typed mock data matching generated types
│       └── render.tsx            ← custom render() with ApolloProvider
```

## State Machine

N/A — no state transitions owned by this context. All state machines live server-side:
- Alert states (OPEN/SNOOZED/ACTED/CLOSED) — alert context
- Action item states (PENDING/IN_PROGRESS/DONE/CLOSED) — action-item-service context
- Transfer status pipeline — transfer-monitor context

The frontend reflects server state via query responses and subscription updates. It never computes or transitions state locally.

## Behaviors (EARS syntax)

### App Initialization

- The system shall wrap the app in `ApolloProvider` with the configured Apollo Client instance.
- The system shall configure Apollo Client with an HTTP link pointing to `VITE_API_URL` for queries and mutations, and an SSE link using `graphql-sse` for subscriptions.
- The system shall split links so that subscription operations route through the SSE link and all other operations route through the HTTP link.
- The system shall use `InMemoryCache` with default type policies, normalizing all types by `id`.

### Codegen

- The system shall use `graphql-codegen` with plugins `typescript`, `typescript-operations`, and `typescript-react-apollo` to generate typed hooks and TypeScript types from `schema.graphql` and `.graphql` operation files.
- The system shall output generated code to `src/generated/graphql.ts`. This file shall not be hand-edited.
- When `schema.graphql` changes, the system shall regenerate via `npx graphql-codegen` (per CLAUDE.md Codegen convention).

### Layout & Navigation

- When the app loads, the system shall execute `GetAdvisor(id: "adv1")` and render the advisor's name and role in the sidebar.
- The system shall render a fixed sidebar (220px) with: AdvisorHub logo, advisor info, and navigation for three sections: Alert Feed, Transfers, Client Book.
- The system shall display a badge count of unresolved alerts (status != CLOSED) on the Alert Feed nav item.
- When a nav item is clicked, the system shall switch the main content area to the corresponding section.
- The system shall default to the Alert Feed section on load.

### Alert Feed

- When the Alert Feed section is active, the system shall execute `GetAlerts(advisorId: "adv1")` and render alert cards sorted by severity (CRITICAL first) then recency.
- The system shall display filter buttons: All, Needs Attention (status = OPEN), Critical, Urgent, Advisory.
- When a filter is selected, the system shall re-query with the appropriate `AlertFilter` input or filter the cached results client-side.
- The system shall render each alert card with: severity indicator (colored left border + dot), client name, category badge, timestamp, summary text.
- Where an alert has a `draftMessage`, the system shall show a "Preview draft" toggle. When expanded, the draft is displayed in an editable text area.
- Where an alert has severity != INFO and status != CLOSED, the system shall show three action buttons: Send, Track, Snooze. These are mutually exclusive — once one is chosen, the alert transitions to ACTED and buttons disappear.
- When "Send" is clicked, the system shall call `sendAlert(alertId, message)` where `message` is the draft text (edited or original). On success, the alert card shall transition to a dimmed "Sent ✓" state.
- When "Track" is clicked, the system shall call `trackAlert(alertId, actionItemText)` with the alert summary as default text. On success, the alert card shall show "Tracked ✓".
- When "Snooze" is clicked, the system shall call `snoozeAlert(alertId, null)` (category default duration). On success, the alert card shall be removed from the active feed or shown as snoozed.
- Where an alert has severity = INFO, the system shall render the card dimmed with a "Sent ✓" badge and a single action: Acknowledge (checkmark icon).
- When "Acknowledge" is clicked on an INFO alert, the system shall call `acknowledgeAlert(alertId)`. On success, the card transitions to CLOSED state.
- The system shall display a "Run Morning Sweep" button in the feed header.
- When "Run Morning Sweep" is clicked, the system shall call `runMorningSweep(advisorId: "adv1")` and display a loading indicator with count of alerts found.
- While the sweep is running, the "Run Morning Sweep" button shall be disabled and show a spinner with "Scanning... (N found)" text.
- When the sweep mutation returns, the system shall display the `SweepResult` summary (alerts generated, updated, skipped, duration).

### SSE Subscription (Real-time Updates)

- When the app loads, the system shall establish an `alertFeed(advisorId: "adv1")` subscription over SSE.
- When an `AlertEvent` with type `CREATED` is received, the system shall insert the new alert card into the feed at the correct sort position (by severity then recency), with an entry animation.
- When an `AlertEvent` with type `UPDATED` is received, the system shall update the existing alert card in place (matching by alert ID).
- When an `AlertEvent` with type `CLOSED` is received, the system shall transition the alert card to a resolved state with a "Resolved" badge.
- If the SSE connection drops, the system shall attempt to reconnect automatically (graphql-sse handles this by default).

### Transfer Tracking

- When the Transfer Tracking section is active, the system shall execute `GetTransfers(advisorId: "adv1")` and render a kanban board with 6 columns matching `TransferStatus` enum: Initiated, Documents Submitted, In Review, In Transit, Received, Invested.
- The system shall render each transfer card with: client name, source institution, account type tag, amount, days in current stage, and initiated date.
- Where `isStuck = true`, the system shall highlight the card with a warning border and show "Stuck Nd" in red.
- The system shall show a count badge on each column header indicating the number of transfers in that stage.

### Client Book

- When the Client Book section is active, the system shall execute `GetClients(advisorId: "adv1")` and render a searchable table with columns: client name + health dot, account type tags, AUM, last meeting date, alert count badge.
- The system shall display total client count and aggregate AUM in the section header.
- When text is entered in the search input, the system shall filter the client list by name (case-insensitive, client-side).
- When a client row is clicked, the system shall execute `GetClient(id)` and `GetContributionSummary(clientId, taxYear: 2026)` and open a detail panel on the right.
- If the same client row is clicked again, the system shall close the detail panel.

### Client Detail Panel

- The system shall display the client header: avatar (initials), name, AUM, last meeting date.
- The system shall display a "Contribution Room — 2026" section with progress bars for each account type from `contributionSummary`. Each bar shows: account type tag, contributed vs limit, remaining room or "Maxed ✓" or "Over by $X" in red.
- Where `isOverContributed = true` for an account, the system shall render the progress bar in red and show the penalty amount.
- Where the client has external accounts, the system shall display an "External Accounts" section listing institution, account type, and balance.
- The system shall display a "Goals" section with progress bars showing name, percentage, and status color (green for ON_TRACK/AHEAD, yellow for BEHIND).
- The system shall display an "Action Items" section with status dot, text, status label, and due date for each item.
- The system shall display an "Advisor Notes" section with date and text for each note, ordered by date descending.
- The system shall display an "Add note" button. When clicked, the system shall show a text input. On submit, the system shall call `addNote(clientId, text)` and append the new note to the timeline.

### Error Handling

- When any GraphQL query or mutation returns an error, the system shall display the error message to the user (inline near the relevant component, not a global toast).
- While a query is loading, the system shall display a loading indicator in the relevant content area.
- Where a mutation is in-flight, the system shall disable the triggering button to prevent double-submission.

### Styling

- The system shall use the dark theme color palette from `docs/mocks/advisor-dashboard.jsx` (background: `#0A0E17`, surface: `#111827`, accent: `#22D3EE`, etc.) stored in `src/styles/theme.ts`.
- The system shall use inline styles or Tailwind CSS for styling (per CLAUDE.md convention).
- The system shall use the `DM Sans` font family.

## Decision Table

**Alert card rendering by severity and status:**

| Severity | Status | Card Opacity | Action Buttons | Badge |
|---|---|---|---|---|
| CRITICAL | OPEN | 1.0 | Send, Track, Snooze | — |
| URGENT | OPEN | 1.0 | Send, Track, Snooze | — |
| ADVISORY | OPEN | 1.0 | Send, Track, Snooze | — |
| INFO | OPEN | 0.65 | Acknowledge | "Sent ✓" |
| Any | ACTED | 0.65 | — | "Sent ✓" or "Tracked ✓" |
| Any | SNOOZED | hidden (filtered out) | — | — |
| Any | CLOSED | 0.65 | — | "Resolved" |

**Alert card action behavior (mutually exclusive — one action per alert):**

| Action | Mutation Called | Result | Card State After |
|---|---|---|---|
| Send | `sendAlert(alertId, message)` | Message sent + ActionItem created, alert → ACTED | Dimmed, "Sent ✓" badge, no buttons |
| Track | `trackAlert(alertId, text)` | ActionItem created (no message), alert → ACTED | Dimmed, "Tracked ✓" badge, no buttons |
| Snooze | `snoozeAlert(alertId, null)` | Alert → SNOOZED | Card removed from active feed |
| Acknowledge (INFO only) | `acknowledgeAlert(alertId)` | Alert → CLOSED | Dimmed, "Resolved" badge, no buttons |

**Alert feed filter mapping:**

| Filter Button | Query Parameter |
|---|---|
| All | No filter |
| Needs Attention | `filter: { status: OPEN }` |
| Critical | `filter: { severity: CRITICAL }` |
| Urgent | `filter: { severity: URGENT }` |
| Advisory | `filter: { severity: ADVISORY }` |

**Contribution bar color by state:**

| Condition | Bar Color | Label |
|---|---|---|
| `isOverContributed = true` | Red (`#EF4444`) | "Over by $X" |
| `remaining = 0` | Green (`#34D399`) | "Maxed ✓" |
| `remaining > 0` | Accent (`#22D3EE`) | "$X room" |

**Health dot color:**

| HealthStatus | Color |
|---|---|
| RED | `#EF4444` |
| YELLOW | `#F59E0B` |
| GREEN | `#34D399` |

**Goal progress bar color:**

| GoalStatus | Color |
|---|---|
| BEHIND | Yellow (`#F59E0B`) |
| ON_TRACK | Green (`#34D399`) |
| AHEAD | Green (`#34D399`) |

**Transfer card stuck highlighting:**

| `isStuck` | Border Color | Days Label |
|---|---|---|
| `true` | Red border (`rgba(239,68,68,0.4)`) + glow | "Stuck Nd" in red |
| `false` | Default border (`#1E293B`) | "Nd in stage" in muted |

## Test Anchors

### Test Infrastructure

**MSW (Mock Service Worker) setup for GraphQL mocking:**

Tests use MSW to intercept GraphQL requests at the network level. MSW handlers are defined against the same operation names used in `.graphql` operation files, ensuring mocks stay in sync with the actual queries.

**Running tests:**

```bash
cd frontend && npm test          # runs vitest in watch mode
cd frontend && npm run test:ci   # runs vitest once (CI mode, no watch)
```

Corresponding `package.json` scripts:

```json
{
  "scripts": {
    "test": "vitest",
    "test:ci": "vitest run"
  }
}
```

**Setup files:**

| File | Purpose |
|---|---|
| `src/test/server.ts` | MSW `setupServer()` instance. Started in `beforeAll`, reset handlers in `afterEach`, closed in `afterAll`. |
| `src/test/handlers.ts` | Default MSW handlers for all GraphQL operations (`graphql.query()`, `graphql.mutation()`). Returns realistic fixture data. |
| `src/test/fixtures.ts` | Typed mock data matching generated types from `src/generated/graphql.ts`. One fixture per GraphQL type (e.g., `mockClient`, `mockAlert`, `mockTransfer`). Uses generated types for compile-time safety. |
| `src/test/render.tsx` | Custom `render()` wrapper that provides `ApolloProvider` with a test-configured Apollo Client (pointing to MSW-intercepted endpoint). |

**MSW handler pattern:**

```typescript
// src/test/handlers.ts
import { graphql, HttpResponse } from 'msw'
import { mockAdvisor, mockClients, mockAlerts, mockTransfers } from './fixtures'

export const handlers = [
  graphql.query('GetAdvisor', () => {
    return HttpResponse.json({
      data: { advisor: mockAdvisor }
    })
  }),
  graphql.query('GetClients', () => {
    return HttpResponse.json({
      data: { clients: mockClients }
    })
  }),
  graphql.query('GetAlerts', ({ variables }) => {
    // Respects filter variables to test filtering behavior
    return HttpResponse.json({
      data: { alerts: filterAlerts(mockAlerts, variables.filter) }
    })
  }),
  graphql.mutation('SendAlert', ({ variables }) => {
    return HttpResponse.json({
      data: { sendAlert: { ...findAlert(variables.alertId), status: 'ACTED' } }
    })
  }),
  // ... handlers for all operations
]
```

**Fixture typing pattern:**

```typescript
// src/test/fixtures.ts
import type { Client, Alert, Transfer } from '../generated/graphql'

export const mockClient: Client = {
  __typename: 'Client',
  id: 'c1',
  name: 'Priya Sharma',
  // ... all fields match generated type
}
```

**Per-test handler overrides:**

Individual tests override default handlers for specific scenarios (error states, empty results, loading states):

```typescript
server.use(
  graphql.query('GetAlerts', () => {
    return HttpResponse.json({
      errors: [{ message: 'Internal server error' }]
    })
  })
)
```

**SSE subscription mocking:**

MSW does not natively mock SSE streams. Subscription tests use a mock Apollo Link that emits `AlertEvent` objects directly into the Apollo subscription observable, bypassing the network layer. This mock link is injected via the test `render()` wrapper.

**Test runner:** Vitest (ships with Vite). Config in `vitest.config.ts` with `jsdom` environment for DOM testing.

### Test Cases

**App.tsx**

1. Given the app renders, when the `GetAdvisor` query resolves, then the sidebar displays the advisor's name and role.
2. Given the app renders, when the Alert Feed nav item is clicked, then the Alert Feed section is displayed.
3. Given the app renders, when the Transfers nav item is clicked, then the Transfer Tracking section is displayed.
4. Given the app renders, when the Client Book nav item is clicked, then the Client Book section is displayed.
5. Given the app renders, then the Alert Feed section is displayed by default.
6. Given 3 unresolved alerts exist, then the Alert Feed nav item displays a badge with count "3".

**AlertFeed.tsx**

7. Given the `GetAlerts` query returns 4 alerts (1 CRITICAL, 1 URGENT, 1 ADVISORY, 1 INFO), then they are rendered in severity order: CRITICAL first, then URGENT, ADVISORY, INFO.
8. Given the "Critical" filter is selected, then only CRITICAL alerts are displayed.
9. Given the "Needs Attention" filter is selected, then only alerts with status OPEN are displayed.
10. Given the "Run Morning Sweep" button is clicked, when the `RunMorningSweep` mutation is in-flight, then the button is disabled and shows a spinner with "Scanning..." text.
11. Given the `RunMorningSweep` mutation resolves with `{ alertsGenerated: 3, alertsUpdated: 1, alertsSkipped: 0, duration: "1.2s" }`, then the sweep result summary is displayed.

**AlertCard.tsx**

12. Given an alert with severity CRITICAL and status OPEN, then the card renders with a red left border, severity dot, client name, category badge, summary text, and three action buttons (Send, Track, Snooze).
13. Given an alert with a `draftMessage`, when "Preview draft" is clicked, then the draft text is displayed in an expandable area.
14. Given an alert with a `draftMessage`, when "Preview draft" is clicked and the draft text is edited, then "Send" uses the edited text.
15. Given the "Send" button is clicked, when `sendAlert` mutation resolves, then the card transitions to dimmed state with "Sent ✓" badge and no action buttons.
16. Given the "Track" button is clicked, when `trackAlert` mutation resolves, then the card transitions to dimmed state with "Tracked ✓" badge and no action buttons.
17. Given the "Snooze" button is clicked, when `snoozeAlert` mutation resolves, then the card is removed from the feed.
18. Given an alert with severity INFO, then the card renders dimmed with "Sent ✓" badge and only an Acknowledge button.
19. Given the "Acknowledge" button is clicked on an INFO alert, when `acknowledgeAlert` mutation resolves, then the card shows "Resolved" badge.
20. Given a mutation is in-flight, then the action button that was clicked is disabled.
21. Given a mutation returns an error, then the error message is displayed on the card.

**TransferTracking.tsx**

22. Given the `GetTransfers` query returns 6 transfers across various statuses, then the kanban board renders 6 columns with transfers in the correct columns.
23. Given a column has 2 transfers, then the column header shows a count badge of "2".
24. Given a column has 0 transfers, then the column renders with an empty placeholder.

**TransferCard.tsx**

25. Given a transfer with `isStuck = true`, then the card renders with a red warning border and "Stuck 18d" label in red.
26. Given a transfer with `isStuck = false`, then the card renders with default border and "3d in stage" label in muted color.
27. Given a transfer, then the card displays client name, source institution, account type tag, amount formatted with $ and commas, and initiated date.

**ClientBook.tsx**

28. Given the `GetClients` query returns 10 clients, then the table renders 10 rows with name, health dot, account type tags, AUM, last meeting, and alert count.
29. Given the section header, then it displays total client count and aggregate AUM.
30. Given "Priya" is typed in the search input, then only clients whose name contains "Priya" are shown (case-insensitive).
31. Given a client row is clicked, then `GetClient` and `GetContributionSummary` queries are fired and the detail panel opens on the right.
32. Given a client row is clicked while the detail panel is already open for that client, then the detail panel closes.

**ClientDetail.tsx**

33. Given the detail panel is open for a client, then it displays the client header with avatar initials, name, AUM, and last meeting date.
34. Given a client with 3 account contributions (1 over-contributed, 1 maxed, 1 with room), then the contribution section renders 3 progress bars with correct colors and labels ("Over by $2,300", "Maxed ✓", "$2,500 room").
35. Given a client with 2 external accounts, then the "External Accounts" section displays institution, account type tag, and balance for each.
36. Given a client with no external accounts, then the "External Accounts" section is not rendered.
37. Given a client with 2 goals (1 BEHIND, 1 ON_TRACK), then the goals section renders progress bars with yellow and green colors respectively.
38. Given a client with 3 action items, then each item displays status dot, text, status label, and due date.
39. Given a client with 2 advisor notes, then notes are displayed in date descending order with date and text.
40. Given the "Add note" button is clicked and text is entered, when submitted, then `addNote` mutation is called and the new note appears in the timeline.

**ContributionBar.tsx**

41. Given `isOverContributed = true` with excess $2,300, then the bar is red and label reads "Over by $2,300".
42. Given `remaining = 0`, then the bar is green and label reads "Maxed ✓".
43. Given `remaining = 2500` with `annualLimit = 7000`, then the bar is accent-colored and label reads "$2,500 room".

**GoalBar.tsx**

44. Given a goal with status BEHIND and progress 42%, then the bar is yellow and shows "42%".
45. Given a goal with status ON_TRACK and progress 85%, then the bar is green and shows "85%".

**Badge.tsx, Avatar.tsx, AccountTag.tsx, SeverityDot.tsx, IconButton.tsx**

46. Given `Badge` receives text "Over-contribution" with critical color, then it renders with the correct background and text color.
47. Given `Avatar` receives initials "PS", then it renders a circle with those initials and a deterministic background color.
48. Given `AccountTag` receives type "RRSP", then it renders with the purple color scheme. Given type "TFSA", then green. Given type "NON_REG", then muted.
49. Given `SeverityDot` receives severity "critical", then it renders a red dot with glow.
50. Given `IconButton` with variant "send", then it renders with accent color styling. When clicked, it calls the `onClick` handler.

**SSE Subscription (tested in AlertFeed.tsx)**

51. Given the `alertFeed` subscription emits an event with type `CREATED`, then a new alert card appears in the feed at the correct sort position.
52. Given the `alertFeed` subscription emits an event with type `UPDATED`, then the existing alert card updates in place (matched by alert ID).
53. Given the `alertFeed` subscription emits an event with type `CLOSED`, then the alert card transitions to show a "Resolved" badge.
