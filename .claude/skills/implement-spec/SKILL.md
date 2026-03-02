---
name: implement-spec
description: Implement a bounded context from its spec. Use when the user asks to build, implement, or code a bounded context.
---

# Implement Spec

You are implementing a bounded context for AdvisorHub from its spec. Follow these instructions exactly.

This skill is the **inner loop** — it only handles implementation. Git operations (branching, committing, PRs) are handled by the caller (autodev.sh or the user).

## Before you start

Read these files in order:
1. `CLAUDE.md` — project overview, conventions, current state
2. The assigned spec from `specs/` (identified by argument or next unbuilt context from CLAUDE.md)
3. `docs/ARCHITECTURE.md` — for domain context referenced by the spec

Check CLAUDE.md "Specs written" and "Built and merged" sections to know what's available and what's already built.

## Argument

`$ARGUMENTS` — the bounded context name or number (e.g., "client", "01", "01-client"). If empty, pick the next unbuilt spec from CLAUDE.md.

## File placement

- Package code: `backend/internal/{context}/` (e.g., `backend/internal/client/`)
- Package name: lowercase single word matching directory (e.g., `client`, `eventbus`, `contribution`)
- Tests: `*_test.go` in the same package
- Migrations: `backend/migrations/` numbered sequentially (check existing migrations to pick the next number)

## TDD workflow

For each behavior or test anchor in the spec:

1. **Write the failing test first.** Translate the test anchor into a Go test. Use table-driven tests where there are multiple scenarios.
2. **Write the minimum implementation to make it pass.**
3. **Verify the test passes.**
4. **Repeat** for the next test anchor.

Do NOT write all tests at once then implement. Do NOT write implementation without a test.

## Testing approach

- **Unit tests only.** No real database, no Docker, no testcontainers.
- **Mock dependencies** using interfaces. If your context depends on another (e.g., temporal scanner depends on contribution engine), define the interface your context needs and create a mock implementation in the test file.
- **In-memory implementations** for repositories. Create a simple map-backed implementation in a `_test.go` or `testutil_test.go` file for testing.
- Test business logic thoroughly. Don't test trivial getters/setters.

## Implementation conventions

- **Go 1.25+**
- `context.Context` on all public functions
- Errors: return `error`, don't panic. Wrap with `fmt.Errorf("doing x: %w", err)`
- Each context owns its own types. Use primitive types (strings) for cross-context references.
- Never import another context's package. If you depend on another context, define the interface you need in your own package.
- Repositories return data ordered by `id` by default. Sorting for display is the GraphQL resolver's job.
- Use `sqlx` for database queries (not raw `database/sql`). All timestamps UTC.
- Migrations: numbered SQL files (`001_create_clients.sql`, `002_create_accounts.sql`, etc.)

## Codegen

If your context modifies `schema.graphql` (only the graphql-api context should), run both codegen steps:

- **Backend**: `cd backend && go run github.com/99designs/gqlgen generate`
- **Frontend**: `cd frontend && npx graphql-codegen`

If your context does NOT touch the schema, skip this.

## What NOT to do

- Don't create a `domain/` package or shared types package
- Don't modify other bounded contexts' code
- Don't add features beyond what the spec defines
- Don't add Docker, CI, or integration test infrastructure
- Don't create README or documentation files
- Don't over-engineer — this is a prototype
- Don't do any git operations (no branching, committing, or pushing)

## When you're done

1. **Run all tests** for your context: `go test ./backend/internal/{context}/...`
2. **Run go vet**: `go vet ./backend/internal/{context}/...`
3. **Update `CLAUDE.md`**: Move the context from "Specs written" to "In progress" or "Built and merged".
4. **Stop.** The caller handles git operations from here.
