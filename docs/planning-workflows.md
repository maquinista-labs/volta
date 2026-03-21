# Current State: Plan/Spec Generation Before Task Launch

> Goal: Document all existing ways a plan or spec is generated prior to task execution, covering both CLI and Telegram entry points, so we can identify duplication and simplification opportunities.

---

## Overview

There are **5 distinct workflows** for generating a plan/spec and transitioning tasks from "idea" to "ready for execution":

| # | Entry Point | Medium | Output | Status After |
|---|-------------|--------|--------|--------------|
| 1 | `/t_plan <description>` | Telegram | JSON batch of tasks | `ready` (immediate) |
| 2 | `/plan [project]` + `/plan release` | Telegram | Draft tasks via agent | `draft` → `ready` |
| 3 | `volta spec sync` | CLI | File-based task batch | `draft` → `ready` |
| 4 | Orchestrator `specScan()` | Automatic | Spec → planner agent | `draft` → `ready` |
| 5 | `volta add` (direct) | CLI | Single task | `ready` or `draft` |

---

## Workflow 1: `/t_plan` — Telegram Interactive Plan

**Entry point:** `internal/bot/plan_commands.go`

**Flow:**
```
User: /t_plan "add dark mode to settings"
  → handlePlanCommand()
  → buildPlanningPrompt() — wraps description in structured prompt
  → executePlan() — sends prompt to a tmux Claude window
  → Monitor watches output for "PLAN_JSON:" marker
  → HandlePlanFromMonitor() — parses JSON array of PlanTask structs
  → showPlanApproval() — renders inline keyboard with task list
  → User clicks [Approve]
  → handlePlanApprove() — calls bridge.AddWithDeps() for each task
  → Tasks created with status "ready" (or "pending" if deps unmet)
```

**Plan format (in-flight, not persisted as files):**
```go
type PlanTask struct {
    Title    string  // imperative description
    Body     string  // implementation details
    Priority int     // 1–5
    After    []int   // 0-based indices into same array (deps)
}
```

**Key characteristics:**
- Entirely in-band: plan lives only in the Claude output buffer until parsed
- Dependencies expressed as array indices, not task IDs
- Plan approval is a single Telegram UI interaction
- Tasks go directly to `ready` — no draft/release step
- No spec file produced or persisted

---

## Workflow 2: `/plan` — Telegram Planner Mode

**Entry points:**
- `internal/bot/planner_commands.go` — bot-side commands
- `cmd/volta/cmd_planner.go` — CLI equivalents (`volta planner start|stop|reopen|status`)

**Flow:**
```
User: /plan myproject
  → plannerStart()
  → Creates a Telegram forum topic: "Planner: myproject"
  → Spawns a Claude Code tmux window with planner-system-prompt.md
  → User converses with Claude to define tasks
  → Claude calls: volta add --status draft --project myproject [--after <ids>] [--priority N]
  → Tasks accumulate in DB as status="draft"

User: /plan release
  → plannerRelease()
  → Calls: minuano draft-release --all --project myproject
  → Transitions: draft → ready (or → pending if deps unmet)
  → Pending tasks auto-promote to ready as deps complete
```

**Planner system prompt:** `claude/planner-system-prompt.md`
- Instructs Claude to use `volta add --status draft` only
- Teaches dependency syntax (`--after <id>`)
- Warns against spawning executors or using `volta run`

**Key characteristics:**
- Interactive conversation loop — Claude is the planner agent
- Session tracked in DB (`planner_sessions` table)
- Tasks are created incrementally, not in a batch
- Explicit release step required before tasks are claimable
- No structured spec file produced

---

## Workflow 3: `volta spec sync` — File-Based Spec Sync

**Entry points:**
- `cmd/volta/cmd_spec.go` — `volta spec sync`, `volta spec validate`

**Flow:**
```
Developer writes: .specs/feature-dark-mode.md
  [YAML frontmatter]
  id: dark-mode-001
  title: "Add dark mode toggle"
  priority: 7
  depends_on: [setup-theme-engine]
  requires_approval: false
  ---
  [Markdown body — implementation details]

volta spec sync --dir .specs --project myproject [--release]
  → spec.ParseDir() — reads all *.md files
  → spec.Sync() — for each SpecFile:
      - If task missing: db.CreateTask() with status="draft"
      - If task exists: update title/body
      - AddDependency() for each depends_on entry
  → --release flag: runs DraftReleaseAll() → draft → ready/pending
```

**Spec file format** (`internal/spec/parser.go`):
```go
type SpecFile struct {
    ID               string
    Title            string
    Priority         int
    DependsOn        []string  // task IDs (not indices)
    TestCmd          string
    RequiresApproval bool
    Body             string    // markdown below frontmatter
    FilePath         string
}
```

**Key characteristics:**
- Specs are persisted files — can be version-controlled
- Dependencies use task IDs (not array indices like `/t_plan`)
- Idempotent: safe to re-run, updates existing tasks
- Manual release step (or `--release` flag)
- No AI involved — human-authored files

---

## Workflow 4: Orchestrator `specScan()` — Automatic Planner Spawning

**Entry point:** `internal/orchestrator/spec_scan.go`

**Flow:**
```
volta orchestrate --project myproject (or embedded in volta start)
  → Each tick: specScan()
  → Reads .specs/ directory
  → For each spec file:
      - Checks if corresponding task exists in DB
      - If not: spawnPlannerForSpec()
          → Creates Telegram topic: "Plan: <spec-title>"
          → Spawns planner agent (role="planner") in tmux
          → Sends spec content as bootstrap prompt to agent
          → Planner decomposes into tasks via volta add --status draft
  → Tracks dispatched specs to avoid duplicates (in-memory set)
```

**Key characteristics:**
- Fully automated — no human trigger beyond placing a file
- Bridges Workflows 2 and 3: reads spec files but spawns a planner agent
- Creates a Telegram topic per spec for visibility
- Relies on planner agent to interpret the spec and create subtasks
- Still requires an explicit release step (planner uses `--status draft`)

---

## Workflow 5: `volta add` / `/p_add` — Direct Task Creation

**Entry points:**
- `cmd/volta/cmd_add.go` — `volta add <title>`
- `internal/bot/add_task.go` — `/p_add` Telegram wizard

**Flow (CLI):**
```
volta add "Optimize DB query" --priority 8 --after task-abc --requires-approval
  → db.CreateTask() — status="ready" by default, or "draft" with --status draft
  → db.AddDependency() for each --after
```

**Flow (Telegram wizard):**
```
User: /p_add "Optimize DB query"
  → Multi-step wizard:
      1. Shows priority inline keyboard (1–10)
      2. Prompts for optional body (user replies to message)
      3. createTask() → bridge.Add()
  → Task created immediately as "ready"
```

**Key characteristics:**
- Single-task creation only — no batch or dependency graph
- No planning AI involved
- CLI default is `ready`; wizard default is also `ready`
- Simplest and most direct path

---

## Current Inconsistencies

### 1. Status at creation differs by workflow

| Workflow | Default status after creation |
|----------|-------------------------------|
| `/t_plan` | `ready` (immediate) |
| `/plan` mode | `draft` (requires release) |
| `volta spec sync` | `draft` (requires release or `--release` flag) |
| Orchestrator scan | `draft` (planner uses `--status draft`) |
| `volta add` / `/p_add` | `ready` (immediate) |

### 2. Dependency format is inconsistent

| Workflow | Dependency format |
|----------|-------------------|
| `/t_plan` PlanTask | Array indices (`After: [0, 1]`) |
| `volta add` | Task ID strings (`--after abc-123`) |
| Spec files | Task ID strings (`depends_on: [abc-123]`) |
| `/plan` agent | Task ID strings (agent uses `--after <id>`) |

### 3. Approval/review gates differ

| Workflow | Human review point |
|----------|--------------------|
| `/t_plan` | Pre-creation: Telegram approve/cancel |
| `/plan` mode | Post-creation: `/plan release` |
| `volta spec sync` | Post-creation: `--release` flag or separate command |
| Orchestrator scan | None explicit (planner creates then release needed) |
| `volta add` | Optional: `--requires-approval` per task |

### 4. Spec/plan persistence

| Workflow | Plan persisted? |
|----------|-----------------|
| `/t_plan` | No — JSON only lives in Claude output buffer |
| `/plan` mode | No — no spec file produced |
| `volta spec sync` | Yes — `.specs/*.md` files |
| Orchestrator scan | Partially — reads spec files, no new artifact |
| `volta add` | No — single task, no plan artifact |

### 5. Planner agent vs direct task creation

`/t_plan` and `/plan` both use AI to generate tasks, but:
- `/t_plan` has Claude output a JSON batch in one shot, then parses it
- `/plan` has Claude run imperatively (calling `volta add` multiple times)

These two approaches are architecturally very different despite solving the same problem.

---

## Key Files Reference

| File | Role |
|------|------|
| `internal/bot/plan_commands.go` | `/t_plan` flow, PLAN_JSON parsing |
| `internal/bot/planner_commands.go` | `/plan` bot commands |
| `cmd/volta/cmd_planner.go` | CLI planner commands |
| `internal/bot/add_task.go` | `/p_add` wizard |
| `cmd/volta/cmd_add.go` | `volta add` CLI |
| `cmd/volta/cmd_spec.go` | `volta spec sync/validate` |
| `internal/spec/parser.go` | Spec file parsing (SpecFile struct) |
| `internal/spec/sync.go` | Spec-to-DB sync logic |
| `internal/orchestrator/spec_scan.go` | Auto spec detection + planner spawn |
| `internal/orchestrator/orchestrator.go` | Main orchestrator loop |
| `internal/db/queries.go` | CreateTask, DraftRelease, AddDependency |
| `internal/bridge/bridge.go` | Bridge to minuano CLI |
| `claude/planner-system-prompt.md` | Planner agent instructions |

---

## Open Questions for Simplification

1. Should all AI-driven planning produce spec files, making the workflow file-first?
2. Should `/t_plan` adopt the draft/release pattern instead of creating tasks immediately?
3. Can the two "AI planner" approaches (`/t_plan` one-shot JSON vs `/plan` interactive) be unified?
4. Should dependency format be standardized to task IDs everywhere (dropping array indices)?
5. Is `specScan()` (auto planner spawning) serving a real need distinct from `volta spec sync`?
