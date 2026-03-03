# CLAUDE.md

## Project

AdvisorHub — AI-powered dashboard for financial advisors. Surfaces prioritized alerts across a client book so advisors focus on relationships, not manual tracking.

## Architecture

- `docs/ARCHITECTURE.md` — domain model, entity relationships, alert system, event pipeline, GraphQL schema, dashboard design, seed data
- `docs/BUILD_WORKFLOW.md` — spec-driven development process, three-loop model, risk tiers, autodev.sh, work ledger format
- `docs/mocks/advisor-dashboard.jsx` — visual reference for dashboard UI

### Tech Stack

- **Backend:** Go + gqlgen (GraphQL) + PostgreSQL
- **Event bus:** Go channels (in-memory, interfaces designed for Kafka/NATS swap)
- **LLM:** Anthropic Claude API for alert enhancement (summary + draft message)
- **Real-time:** GraphQL subscriptions over SSE (`transport.SSE{}` in gqlgen)
- **Frontend:** React + TypeScript
- **Deployment:** Railway (Postgres managed instance, Go service, React static)

### Deployment (Railway)

- **Project:** robust-happiness (Railway auto-name)
- **Backend service:** `https://backend-production-d58b.up.railway.app`
  - Root directory: `backend/`
  - Port: 8080
  - Env vars: `DATABASE_URL` (references Postgres service), `ANTHROPIC_API_KEY`, `CORS_ALLOWED_ORIGIN=https://advisorhub-production.up.railway.app`
- **Frontend service:** `https://advisorhub-production.up.railway.app`
  - Root directory: `frontend/`
  - Port: 8080 (serve)
  - Build arg: `VITE_API_URL=https://backend-production-d58b.up.railway.app`
- **Postgres:** managed instance, connected via `DATABASE_URL`
- **Auto-deploy:** both services deploy on push to main via GitHub integration
- **Migrations:** run from autodev.sh via `cat backend/migrations/*.sql | railway connect Postgres` (not from inside the container)

### Pipeline

```
Event Producers → Event Bus → Alert System → Dashboard
                               (map → dedup/state → enhance)
```

- **Event Producers:** Contribution Engine, Transfer Monitor, Temporal Scanner, Seed Data Loader
- **Alert System:** Single package combining event→alert mapping, lifecycle (dedup, state machine, cascade close), and LLM enhancement. Subscribes to domain events, maps to proto-alerts, runs dedup/state transitions, and conditionally enhances (LLM summary + draft message on CREATED/REOPENED). Enhancer is behind an interface for testability.

### Directory Structure

```
advisorhub/
├── CLAUDE.md                 ← you are here
├── docs/
│   ├── ARCHITECTURE.md       ← domain model, alert system, event pipeline, schema
│   ├── BUILD_WORKFLOW.md     ← development process, risk tiers, autodev.sh
│   ├── WORK_LEDGER.md        ← build log (written by autodev.sh, do not edit)
│   └── mocks/
│       └── advisor-dashboard.jsx  ← visual reference for frontend
├── specs/                    ← bounded context specs (your assignment is one of these)
├── schema.graphql            ← shared schema (consumed by backend + frontend)
├── autodev.sh
├── manifest.yaml
├── backend/
│   ├── server.go
│   ├── internal/
│   │   ├── eventbus/         ← event envelope, pub/sub
│   │   ├── contribution/     ← contribution engine (CRA rules, room calc)
│   │   ├── transfer/         ← transfer monitor (stuck detection)
│   │   ├── temporal/         ← temporal scanner (rule-driven sweep)
│   │   ├── alert/            ← alert system (mapping, lifecycle, enhancement)
│   │   ├── actionitem/       ← action item CRUD
│   │   ├── seed/             ← seed data loader
│   │   └── graph/            ← gqlgen generated + resolvers
│   │       ├── resolver.go
│   │       └── model/
│   ├── migrations/           ← SQL migrations (numbered)
│   ├── go.mod
│   └── go.sum
└── frontend/
    ├── src/
    ├── package.json
    └── tsconfig.json
```

Each bounded context lives in its own package under `backend/internal/`. Each context owns its own types. Cross-context references use primitive types (strings for IDs). Contexts communicate through the event bus or through interfaces — never by importing each other's packages directly.

## How to Work

1. **Read your spec first.** Your assigned spec is in `specs/`. It defines what you own, what you don't own, your contracts, and test anchors.
2. **Read `docs/ARCHITECTURE.md`** for domain model and architectural context if your spec references entities or patterns defined there.
3. **Write tests first.** Use the test anchors from your spec as starting points. Write a failing test, then implement. No exceptions.
4. **Stay in your bounded context.** Only modify files in your assigned package. Each context owns its own types. Use primitive types (strings) for cross-context references like IDs.
5. **Log non-obvious decisions.** If you make a choice not dictated by the spec (data structure selection, error handling approach, etc.), add a comment or note in your commit message explaining why.
6. **Don't touch other bounded contexts.** If your spec says "Depends on: event-bus", you import and use its public interface. You do not modify it.

## Conventions

### Go

- Go 1.25+
- Use `context.Context` on all public functions
- Errors: return `error`, don't panic. Wrap with `fmt.Errorf("doing x: %w", err)`
- Tests: `*_test.go` in the same package. Use table-driven tests where there are multiple scenarios.
- Naming: packages are lowercase single words matching the directory name

### Database

- PostgreSQL. Migrations in `backend/migrations/` numbered sequentially: `001_create_clients.sql`, `002_create_accounts.sql`, etc.
- Use `sqlx` for queries (not raw `database/sql`)
- Use `timestamptz` for all timestamp columns

### Timestamps & Timezone

- **Prototype uses EST (`America/New_York`) everywhere.** This is a demo-only simplification so timestamps display correctly without frontend timezone conversion.
- All `time.Now()` calls must use `time.Now().In(est)` where `est, _ = time.LoadLocation("America/New_York")`. This location is initialized once in `server.go` and passed to contexts that generate timestamps.
- Seed data timestamps are constructed in EST: `time.Date(2026, 2, 25, 14, 30, 0, 0, est)`
- The `DateTime` GraphQL scalar marshals as RFC3339 with offset: `"2026-03-02T09:15:00-05:00"`
- The `Date` GraphQL scalar is timezone-agnostic: `"2026-03-02"`
- **In production**, the server would use UTC and the frontend would handle locale conversion. For the prototype, skip that complexity.

### Event Bus

- Import from `internal/eventbus`
- Events use the `EventEnvelope` type: `{ID, Type, EntityID, EntityType, Payload, Source, Timestamp}`
- EntityID/EntityType are generic entity references (not domain-specific) — kept on the envelope for query convenience, should eventually move to Payload
- Publish with `bus.Publish(ctx, envelope)`
- Subscribe with `bus.Subscribe(eventType)` which returns a channel
- Source is one of: `REACTIVE`, `TEMPORAL`, `ANALYTICAL`, `SYSTEM`

### GraphQL

- Schema in `schema.graphql` at project root (shared by backend + frontend)
- gqlgen config in `backend/gqlgen.yml`
- Resolvers in `backend/internal/graph/resolver.go` (or split by type)
- SSE transport enabled for subscriptions
- Mutations: `sendAlert`, `trackAlert`, `snoozeAlert`, `acknowledgeAlert`, `createActionItem`, `updateActionItem`, `addNote`, `runMorningSweep`

### Frontend

- React 18+ with TypeScript
- Vite for build tooling
- Visual reference: `docs/mocks/advisor-dashboard.jsx`
- Tailwind for styling is fine, or inline styles
- GraphQL client: urql or Apollo — pick one and stick with it
- SSE subscription via `graphql-sse` client library
- CORS for cross-origin requests to backend (`rs/cors` on backend side)
- Static serving via `serve` in production (no nginx)

### Codegen

When `schema.graphql` changes, both sides must regenerate:

- **Backend**: `cd backend && go run github.com/99designs/gqlgen generate` — produces Go types in `graph/model/` and resolver stubs in `graph/`
- **Frontend**: `cd frontend && npx graphql-codegen` — produces TypeScript types and typed hooks from the schema

Run both after any schema change. The graphql-api context owns the schema; other contexts should not modify it.

## Current State

<!-- Update this section as contexts are built and merged -->

### Built and merged

- client (Client, Household, Advisor, AdvisorNote, Goal) — `specs/01-client.md`

### In progress

- (none yet)

### Specs written

- client (Client, Household, Advisor, AdvisorNote, Goal) — `specs/01-client.md`
- account (Account, AccountType, RESPBeneficiary) — `specs/02-account.md`
- event-bus (EventEnvelope, EventSource, EntityType, pub/sub) — `specs/03-event-bus.md`
- contribution-engine (Contribution, ContributionRule, room calc, CESG) — `specs/04-contribution-engine.md`
- transfer-monitor (Transfer, TransferStatus, stage thresholds, stuck detection) — `specs/05-transfer-monitor.md`
- temporal-scanner (TemporalRule, check functions, sweep) — `specs/06-temporal-scanner.md`
- alert (Alert, AlertCategoryRule, dedup, state machine, cascade close, LLM enhancement) — `specs/07-alert.md`
- action-item-service (ActionItem, ActionItemStatus, CRUD, status transitions) — `specs/08-action-item-service.md`
- graphql-api (resolvers, SSE subscriptions, Railway deployment) — `specs/09-graphql-api.md`
- seed-data (seed loader, pre-computed events) — `specs/10-seed-data.md`
- frontend (React dashboard, Apollo Client, graphql-codegen) — `specs/11-frontend.md`

### Specs not started

- (none)

## Key Domain Rules (quick reference)

These are detailed in `docs/ARCHITECTURE.md` but repeated here for fast access:

- **Pipeline:** Event Producers → Event Bus → Alert System (map → dedup/state → enhance on CREATED/REOPENED) → Dashboard
- **Alert dedup:** by `condition_key`. Find most recent WHERE status ≠ CLOSED. If CLOSED, create new alert.
- **Alert states:** OPEN → SNOOZED → OPEN (on expiry) | OPEN → ACTED → SNOOZED (auto) | Any non-CLOSED → CLOSED (on condition resolve)
- **Cascade close:** when alert → CLOSED, all linked ActionItems → CLOSED with auto resolution note
- **Advisor actions:** Send (sends draftMessage + creates ActionItem), Track (creates ActionItem only), Snooze
- **Contribution rules:** RRSP ($32,490 or 18% earned income), TFSA ($7,000), FHSA ($8,000/$40K lifetime), RESP ($2,500 CESG match/$50K lifetime)
- **Penalty:** 1%/month on excess for RRSP, TFSA, FHSA. No penalty for RESP (just lose CESG match).
- **Event producers:** Contribution Engine (REACTIVE), Transfer Monitor (REACTIVE), Temporal Scanner (TEMPORAL), Analytical Engine (ANALYTICAL, mocked), Seed Data Loader (REACTIVE, pre-computed)
- **Temporal scanner:** Rule-driven. Iterates TemporalRules table, dispatches to check functions (AGE_APPROACHING, DEADLINE_WITH_ROOM, DAYS_SINCE, BALANCE_IDLE). Triggered by `runMorningSweep` mutation.
- **SSE events to frontend:** AlertCreated, AlertUpdated, AlertClosed only. All other events are internal.
