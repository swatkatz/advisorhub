---
name: write-spec
description: Write a bounded context spec for AdvisorHub. Use when the user asks to write, create, or work on a spec for a bounded context.
---

# Write Spec

You are writing a bounded context spec for AdvisorHub. Follow these instructions exactly.

## Before you start

Read these files in order:
1. `CLAUDE.md` — project overview, current state, conventions
2. `docs/ARCHITECTURE.md` — domain model, entity relationships, alert system, event pipeline
3. `docs/BUILD_WORKFLOW.md` — spec template format
4. `specs/01-client.md` — reference spec (follow its format, depth, and style exactly)

Check CLAUDE.md "Specs written" and "Specs not started" sections to know which spec to write next, unless the user specifies one.

## Argument

`$ARGUMENTS` — the bounded context name or number (e.g., "account", "02", "02-account"). If empty, pick the next unwritten spec from CLAUDE.md.

## Process

Write the spec **one section at a time**. Show each section to the user and wait for approval before moving to the next. Do NOT write the whole spec at once.

### Section order:

1. **Bounded Context** — Owns, Does not own, Depends on, Produces
2. **Contracts** — Input, Output, Data Model (show together, they're tightly coupled)
3. **State Machine** — if applicable, otherwise write "N/A" and skip
4. **Behaviors (EARS syntax)** — use the five EARS patterns
5. **Decision Table** — if applicable, otherwise write "N/A" and skip
6. **Test Anchors** — Given/When/Then format

After each section, pause and ask the user for feedback. Only proceed when they approve.

## Conventions (established during spec #1)

Follow these conventions in every spec:

- **Data Model indexes**: Always describe indexes for every table. Name them explicitly (e.g., `idx_table_column`). Explain the query path each index supports.
- **Sorting**: Repositories return data ordered by `id` by default. All user-facing sorting (by name, date, severity, etc.) is the responsibility of the GraphQL resolver layer, not repositories. This is documented in `docs/ARCHITECTURE.md` under "GraphQL Schema > Sorting".
- **Repository interfaces**: Define Go interface signatures with `context.Context` as first param. Return `(*Entity, error)` for single, `([]Entity, error)` for lists.
- **Cross-context references**: By ID only. Never import another context's package.
- **Query paths**: Design interfaces around actual query patterns from the frontend/resolvers, not abstract CRUD.
- **FK references**: Note FK relationships in constraints column but remember the context does not own the referenced entity — it just stores the ID.

## After the spec is complete

Once all sections are approved and the spec is written to `specs/`:

1. **Update `CLAUDE.md`**: Move the context from "Specs not started" to "Specs written" with the spec file path.
2. **Update `docs/ARCHITECTURE.md`**: If any decisions made during spec writing affect the architecture (new conventions, clarifications, cross-cutting concerns), add them to the relevant section.

Always ask the user if there's anything to update in ARCHITECTURE.md — don't silently skip this step.
