# TODO Tracking with Beads

Shared reference for creating and managing TODOs via beads. Referenced by `/ship` (Step 5.5) and `/plan-ceo-review` to ensure consistent issue creation.

---

## Creating Issues

Use `bd create` for all TODO tracking. Never create TODOS.md files.

```bash
bd create \
  --title="Short summary of the work" \
  --description="Why this issue exists and what needs to be done" \
  --type=task|bug|feature \
  --priority=0|1|2|3|4
```

**Priority values:** 0=critical, 1=high, 2=medium, 3=low, 4=backlog (P0–P4). Do NOT use "high"/"medium"/"low" strings.

---

## Required Fields

| Field | Flag | Notes |
|-------|------|-------|
| Title | `--title` | One-line summary of the work |
| Description | `--description` | Why it exists + what needs to be done + where to start |
| Priority | `--priority` | 0–4 |
| Type | `--type` | `task`, `bug`, or `feature` |

---

## Priority Definitions

- **P0 (0)** — Blocking: must be done before next release
- **P1 (1)** — Critical: should be done this cycle
- **P2 (2)** — Important: do when P0/P1 are clear
- **P3 (3)** — Nice-to-have: revisit after adoption/usage data
- **P4 (4)** — Someday: good idea, no urgency

---

## Creating Multiple Issues Efficiently

When creating several issues at once, use parallel subagents:

```bash
# Run these in parallel via subagents
bd create --title="Implement feature X" --description="..." --type=feature --priority=2
bd create --title="Write tests for X" --description="..." --type=task --priority=2
bd create --title="Update docs for X" --description="..." --type=task --priority=3
```

Then add dependencies if needed:

```bash
bd dep add beads-yyy beads-xxx  # yyy depends on xxx (xxx blocks yyy)
```

---

## Completing Issues

```bash
bd close <id>                       # Mark complete
bd close <id1> <id2> ...            # Close multiple at once
bd close <id> --reason="explanation" # Close with reason
```

---

## Viewing Work

```bash
bd ready                    # Issues with no blockers
bd list --status=open       # All open issues
bd show <id>                # Full detail with dependencies
bd search <keyword>         # Search by keyword
```
