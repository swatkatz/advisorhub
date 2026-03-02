# AdvisorHub — Build Workflow

## Project

**AdvisorHub**: AI-powered dashboard for Wealthsimple financial advisors. Surfaces prioritized alerts across their client book so advisors focus on relationships, not spreadsheet arithmetic.

**Submission**: Wealthsimple AI Builders. Deadline March 2, 2026 11:59pm PT. Deliverable: GitHub repo + video.

**Dual thesis**: The product works. The process used to build it is replicable across WS.

---

## Three-Loop Model

```
Outer Loop (You):
  Write specs → Assign risk tiers → Run autodev.sh →
  Review HIGH-risk plans/PRs → Check architectural coherence →
  Course-correct specs if needed → Record video

Middle Loop (autodev.sh):
  Read manifest → Process contexts in dependency order →
  Dispatch Claude Code with spec → Enforce risk-tier gates →
  Log results to WORK_LEDGER.md → Commit

Inner Loop (Claude Code agent):
  Read spec → Write plan → (await approval if HIGH risk) →
  [Write failing test → Implement → Tests pass] × N →
  Log decisions → Push branch
```

---

## Risk Tiers

| Tier   | Verification                                                    | When to use                                                                                               |
| ------ | --------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| HIGH   | Agent writes plan, pauses for human approval. Human reviews PR. | Business logic with real-world consequences (financial calculations, state machines with cascade effects) |
| MEDIUM | Agent runs autonomously. Human skims PR.                        | Infrastructure, integrations, schema definitions                                                          |
| LOW    | Agent runs autonomously. Auto-merge if tests pass.              | Seed data, CRUD, frontend components                                                                      |

---

## Tech Stack

- **Backend:** Go + gqlgen (GraphQL)
- **Database:** PostgreSQL (Railway managed instance)
- **Event bus:** Go channels (structured as if Kafka/NATS)
- **LLM:** Anthropic API (Claude) for natural-language alert generation
- **Frontend:** React + TypeScript
- **Real-time:** GraphQL subscriptions over SSE (gqlgen native `transport.SSE{}`)

---

## Spec Format Template

Each bounded context gets a spec file in `specs/`. Use this template:

```markdown
# Spec: {Context Name}

## Bounded Context

Owns: {what this context is responsible for}
Does not own: {explicit exclusions — prevents agent from reaching outside its domain}
Depends on: {other contexts this reads from or calls}
Produces: {events emitted, interfaces exposed to other contexts}

## Contracts

### Input

{Events consumed, function signatures, data read}

### Output

{Events emitted, interfaces exposed, data written}

### Data Model

{Entities owned by this context — fields, types, constraints}

## State Machine (if applicable)

{ASCII diagram of states, transitions, and guards}

## Behaviors (EARS syntax)

Use these five patterns:

- When {trigger}, the system shall {response}. ← event-driven
- While {state}, the system shall {behavior}. ← state-dependent
- Where {condition}, the system shall {behavior}. ← conditional
- If {condition} then {behavior} else {alternative}. ← branching
- The system shall {behavior}. ← universal

## Decision Table (if applicable)

| Input A | Input B | Output |
| ------- | ------- | ------ |
| ...     | ...     | ...    |

## Test Anchors

{Explicit scenarios that MUST pass. These become TDD seeds.}

1. Given {precondition}, when {action}, then {expected result}.
2. ...
```

---

## Build Manifest Template

File: `manifest.yaml`

```yaml
contexts:
  - name: context-name
    spec: specs/01-context-name.md
    risk: HIGH | MEDIUM | LOW
    depends_on: []
    description: "One-line summary"
```

Order contexts so dependencies come first. The middle loop processes them sequentially in manifest order.

---

## autodev.sh Skeleton

```bash
#!/bin/bash
set -e

LEDGER="WORK_LEDGER.md"

# Initialize ledger
cat > $LEDGER << 'EOF'
# Work Ledger — AdvisorHub Build

| Context | Risk | Duration | Tests | Review | Status |
|---------|------|----------|-------|--------|--------|
EOF

echo ""
echo "Build started: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo ""

# Define contexts in dependency order
# Format: name|spec|risk
CONTEXTS=(
  # Fill these in as you write specs
  # "domain-entities|specs/01-domain-entities.md|LOW"
  # "event-bus|specs/02-event-bus.md|MEDIUM"
  # ...
)

for entry in "${CONTEXTS[@]}"; do
  IFS='|' read -r name spec risk <<< "$entry"

  echo "═══════════════════════════════════════"
  echo "  Context: $name"
  echo "  Risk:    $risk"
  echo "  Spec:    $spec"
  echo "═══════════════════════════════════════"

  start_time=$(date +%s)

  # Create branch
  git checkout -b "feat/$name" main 2>/dev/null || git checkout "feat/$name"

  # HIGH risk gate
  if [ "$risk" = "HIGH" ]; then
    echo ""
    echo "⚠️  HIGH RISK — Review agent plan before proceeding"
    echo "Press enter to continue after reviewing plan, or Ctrl+C to abort"
    read -r
  fi

  # TODO: Dispatch to Claude Code
  # claude --spec "$spec" --context "$name"

  # TODO: Run tests
  # go test ./internal/$name/...

  end_time=$(date +%s)
  duration=$((end_time - start_time))

  # Log to ledger
  echo "| $name | $risk | ${duration}s | TODO | $([ "$risk" = "HIGH" ] && echo "Human" || echo "Auto") | ✅ |" >> $LEDGER

  # Merge back
  git checkout main
  git merge "feat/$name" --no-ff -m "feat($name): implement $name [risk:$risk]"

  echo ""
done

echo "" >> $LEDGER
echo "Build completed: $(date -u +%Y-%m-%dT%H:%M:%SZ)" >> $LEDGER
echo ""
echo "✅ All contexts processed. See $LEDGER for build record."
```

---

## Work Ledger Output Format

The ledger is your evidence artifact. It should capture:

```markdown
# Work Ledger — AdvisorHub Build

| Context             | Risk   | Duration | Tests | Review         | Status |
| ------------------- | ------ | -------- | ----- | -------------- | ------ |
| domain-entities     | LOW    | 12m      | 8/8   | Auto           | ✅     |
| event-bus           | MEDIUM | 18m      | 12/12 | Skimmed        | ✅     |
| contribution-engine | HIGH   | 38m      | 24/24 | Human-reviewed | ✅     |
| ...                 | ...    | ...      | ...   | ...            | ...    |

Build started: 2026-03-01T10:00:00Z
Build completed: 2026-03-01T13:42:00Z
Total contexts: 8
Human reviews: 3
Auto-approved: 5
```
