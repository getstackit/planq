# Agent State Repository Architecture

## Overview

A git-based system for managing agent context, memory, and inter-agent communication that lives **separately** from the code repository. Unlike Tasuku/Beads which embed state in `.tasuku/` or `.beads/` within the project, this approach treats agent state as orthogonal to code.

**Key Insight from Research**: Tasuku uses pull-based Markdown+YAML in `.tasuku/` with per-file locking for parallel safety. Beads uses JSONL dependency graphs in `.beads/` with hash-based IDs and semantic memory decay. Both embed in the code repo. Our approach externalizes state entirely.

## Design Principles

1. **Isolation from code history** - Agent artifacts don't pollute project commits, blame, or PR diffs
2. **Cross-project memory** - Learnings can flow between projects from a central store
3. **Lifecycle independence** - Agent state can be reset, archived, or forked independently of code branches
4. **Remote-first** - Designed for team sharing with push/pull semantics
5. **Human-readable** - Markdown + YAML frontmatter for git-diff friendliness

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| State repo scope | Per-project | Enables cross-session learning accumulation |
| Project identification | Explicit project name | User configures in `.planq/config`, avoids git remote edge cases |
| Branch collision handling | Fail if exists | Prevents accidental overwrites, simple semantics |
| Cleanup strategy | Manual consolidation + gc | Promote learnings to global, then prune agent branches |
| Agent interface | **MCP Server** | Structured tools with schema validation, like Tasuku |
| Auto-context | **Hooks** | Inject context at key moments (session start, pre-compact) |
| Sharing model | Remote-first | Team coordination is primary use case |
| Data format | Markdown + YAML frontmatter | Human-readable, git-diff friendly |
| Session lifecycle | Ends on `planq remove` | Clean mapping to workspace lifecycle |
| Mounting mechanism | **Nested git worktree** | Full git experience, cross-platform, no privileges |
| Branch context | **Files in state repo** (not git notes) | Git notes don't survive rebases |

---

## Architecture Components

### 1. State Repository Structure

Per-project state repos using **bare git repos with worktrees** (shared across sessions):

```
~/.planq/
├── state.json                        # Existing global planq state
└── state-repos/
    └── {project-name}.git/           # Bare repo per project (shared across sessions)
        ├── refs/heads/
        │   ├── global                # Shared learnings branch
        │   ├── agent-workspace1      # Per-agent branch
        │   └── agent-workspace2
        └── ...

# Each agent gets a worktree checked out at:
/path/to/code-worktree/.planq/agent/        # → branch agent-{name}
```

**Hierarchical Content Structure** (within each agent's worktree):

```
.planq/agent/                               # Git worktree of state repo
├── scratch.md                        # Agent's working notes
├── changelog.md                      # Agent-specific actions
├── context.md                        # Current focus/state
├── artifacts/                        # Generated outputs
├── notes/                            # Observations
└── checkpoints/                      # Named save points

# Session-level (on 'global' branch):
├── plan.md                           # Driving plan for session
├── changelog.md                      # Aggregated session log
├── learnings.md                      # Accumulated knowledge
├── decisions.md                      # Architectural decisions
└── branches/
    └── {branch-name}.md              # Branch-attached context
```

### 2. Branch Context (Why Not Git Notes)

**Git Notes Limitations Discovered**:
- Notes attach to **commits**, not branches - when you rebase, notes are orphaned
- `git config notes.rewriteRef` helps but is fragile
- Notes don't push/pull by default (need explicit `refs/notes/*` configuration)
- Multiple authors adding notes causes conflicts

**Recommended Approach**: Store branch context as **files in the state repo**:

```
.planq/agent/branches/
└── {branch-name}.md                  # Context for this branch
```

```markdown
---
branch: jonnii/add-auth
created_at: 2026-01-20T10:00:00Z
agent: add-auth
status: in_progress
---

## Context

This branch implements JWT authentication for the CLI.

## Plan

1. Create auth package with token generation
2. Add login/logout commands
3. Protect workspace commands with auth

## Key Decisions

- Using RS256 for JWT signing (see decisions.md#jwt-signing)
```

**Advantages over git notes**:
- Survives rebases (tracked in state repo, not code repo)
- Natural git workflow (add, commit, push)
- Human-readable and editable
- No special configuration needed

**Optional**: Use git notes for CI/CD metadata (build status, approvals) since those attach to specific commits intentionally.

### 3. Mounted Scratch Pad (Nested Git Worktree)

**Evaluated Options**:

| Option | Verdict | Issue |
|--------|---------|-------|
| Symlink | Partial | Agent can't use git from `.planq/agent/` |
| Git submodule | Not viable | Tracked in parent repo, defeats purpose |
| Bind mount | Not viable | Requires privileges, poor cross-platform |
| **Nested worktree** | **Recommended** | Full git experience, cross-platform |

**How Nested Worktree Works**:

```bash
# One-time: create bare state repo for project
git init --bare ~/.planq/state-repos/{project-name}.git

# Per-workspace: create worktree at .planq/agent/
cd ~/.planq/state-repos/{project-name}.git
git worktree add /path/to/code-worktree/.planq/agent agent-{workspace-name}
```

**Result**:
```
/path/to/code-worktree/
├── .planq/agent/                    # This IS a git repo (worktree)
│   ├── .git                   # Worktree link file
│   ├── scratch.md
│   ├── changelog.md
│   └── artifacts/
├── .planq/                    # Existing planq metadata
└── [project files]
```

**Agent Experience** (full git capability):
```bash
cd .planq/agent
git add .
git commit -m "Checkpoint: finished auth middleware"
git log --oneline
git diff HEAD~1
```

**Why This Works**:
- Git correctly handles nested repos when parent gitignores the inner
- Each agent gets its own branch (`agent-{name}`) in the state repo
- Cross-platform: works identically on macOS, Linux, Windows
- No special privileges needed
- Natural cleanup: `git worktree remove .planq/agent`

### 4. Changelog Format for Inter-Agent Communication

**Design**: Append-only Markdown with YAML frontmatter, inspired by Beads' hash-based IDs.

**Entry Schema**:

```yaml
---
# Required
id: bd-a1b2c3d4              # Unique ID (hash-based like Beads)
timestamp: 2026-01-20T10:30:00Z
agent: add-auth               # Workspace name
type: implement               # Entry type (see below)

# Relationships
parent: bd-9f8e7d6c           # Previous entry by same agent
depends_on: [bd-1a2b3c4d]     # Entries this depends on

# Decision tracking (optional)
decision: Use JWT with RS256
alternatives: [HS256, Session-based, OAuth2]
rationale: RS256 allows public key verification

# Progress
status: completed             # pending|in_progress|completed|blocked
---

## Implemented JWT token validation

Added `ValidateToken()` function and integrated with session middleware.
```

**Entry Types**:

| Type | Use Case |
|------|----------|
| `start` | Beginning of agent session |
| `plan` | Design decisions, approach selection |
| `implement` | Active coding work |
| `decision` | Recording why approach X was chosen |
| `handoff` | Context for another agent to continue |
| `end` | Completion of work segment |

**Session Aggregation**: Main `changelog.md` aggregates from agent changelogs:

```markdown
---
generated_at: 2026-01-20T15:00:00Z
project: my-project
agents: [add-auth, fix-cli-help]
---

# Session Changelog

## Active Agents

| Agent | Status | Last Activity |
|-------|--------|---------------|
| add-auth | completed | 14:00 |
| fix-cli-help | in_progress | 14:45 |
```

### 5. Hierarchical Scoping

```
global branch    ← Cross-session learnings (shared across agents)
    ↓ inherits
agent-{name}     ← Individual agent scratch space (per-worktree)
```

**Implementation via git branches**:
- `global` branch: Session-wide learnings, decisions, aggregated changelog
- `agent-{name}` branch: Per-agent scratch pad, individual changelog

**Read path**: Agent reads own branch, can cherry-pick from global
**Write path**: Agent writes to own branch, can merge to global for sharing

---

## Integration with Planq

### Modified Workspace Creation Flow

```
planq create <name>
    ↓
    [existing steps: validate, create worktree via stackit]
    ↓
    ├─→ Get project name from .planq/config (or prompt to initialize)
    │
    ├─→ Initialize state repo if not exists
    │   └─→ git init --bare ~/.planq/state-repos/{project-name}.git
    │   └─→ Create orphan 'global' branch with initial structure
    │
    ├─→ Check if agent-{name} branch exists → FAIL if yes
    │
    ├─→ Create agent worktree at .planq/agent/
    │   └─→ git worktree add /path/to/.planq/agent agent-{name}
    │   └─→ Initialize scratch.md, changelog.md, artifacts/
    │
    ├─→ Add .planq/agent to .gitignore if needed
    │
    └─→ Set environment variables in tmux session
        └─→ PLANQ_PROJECT_NAME, PLANQ_STATE_REPO, PLANQ_AGENT_BRANCH
```

### New Commands

```bash
# State repo management
planq state init          # Initialize state repo for current session
planq state push          # Push state repo to remote
planq state pull          # Pull state repo from remote
planq state gc            # Garbage collect old state repos

# Changelog operations
planq log                 # View current agent's changelog
planq log --all           # View aggregated session changelog
planq log add "message"   # Add entry to changelog

# Context management
planq context show        # Show current agent's context
planq context promote     # Merge learnings to global branch
```

### Environment Variables (set in tmux session)

```bash
PLANQ_PROJECT_NAME=my-project           # From .planq/config
PLANQ_STATE_REPO=~/.planq/state-repos/my-project.git
PLANQ_AGENT_BRANCH=agent-feature
PLANQ_AGENT_DIR=/path/to/worktree/.planq/agent
```

### Consolidation Workflow

Agent work is promoted to the global branch before cleanup:

1. **Review agent changelog**: `planq log agent-auth`
2. **Cherry-pick learnings**: Manually or via `planq context promote`
3. **Mark for cleanup**: Agent branch flagged as consolidated
4. **Run GC**: `planq state gc` removes consolidated branches

This workflow enables accumulated learnings across sessions while keeping the state repo clean.

### MCP Server

An MCP server provides structured tools for agent interaction with the state repo. This is the **primary interface** for AI agents.

**Installation:**
```bash
planq mcp install        # Add to Claude Code's MCP config
planq mcp serve          # Run standalone (for debugging)
```

**Core Tools:**

| Tool | Purpose | Example |
|------|---------|---------|
| `planq_context` | Get current agent context | Returns scratch, recent changelog, focus |
| `planq_scratch` | Update working notes | `planq_scratch("Investigating auth bug...")` |
| `planq_log` | Add changelog entry | `planq_log("Implemented JWT validation", type="implement")` |
| `planq_learn` | Capture a learning | `planq_learn("Always check token expiry before validation")` |
| `planq_decide` | Record a decision | `planq_decide("Use RS256", over=["HS256"], because="...")` |
| `planq_checkpoint` | Create named save point | `planq_checkpoint("before-refactor")` |
| `planq_handoff` | Prepare context for another agent | `planq_handoff("Continue with API tests")` |

**Tool Schemas:**

```yaml
planq_log:
  message: string (required)    # What happened
  type: enum [start, plan, implement, decision, handoff, end]
  status: enum [pending, in_progress, completed, blocked]

planq_learn:
  insight: string (required)    # The learning
  context: string (optional)    # What prompted this

planq_decide:
  decision: string (required)   # What was chosen
  over: array[string]           # Alternatives considered
  because: string               # Rationale
```

### Hooks

Hooks automatically inject context at key moments, reducing agent cognitive load.

| Hook | Trigger | Action |
|------|---------|--------|
| `SessionStart` | Agent session begins | Surface `.planq/agent/context.md` and recent changelog |
| `PreCompact` | Before context compression | Prompt: "Capture any learnings before compaction" |
| `PostCommit` | After git commit | Prompt: "Log this work to changelog?" |

**Hook Configuration** (in `.planq/config`):
```yaml
hooks:
  session_start: true
  pre_compact: true
  post_commit: false           # Optional, can be noisy
```

---

## Implementation Phases

### Phase 1: Foundation (Core Infrastructure)
- [ ] Create `internal/staterepo` package
- [ ] Implement bare repo initialization
- [ ] Implement agent worktree creation at `.planq/agent/`
- [ ] Update `planq create` to initialize agent directory
- [ ] Update `planq remove` to cleanup agent worktree
- [ ] Add `.planq/agent` to gitignore handling

### Phase 2: MCP Server
- [ ] Create `internal/mcp` package
- [ ] Implement MCP server with stdio transport
- [ ] Implement `planq_context` tool
- [ ] Implement `planq_scratch` tool
- [ ] Implement `planq_log` tool
- [ ] Implement `planq_learn` and `planq_decide` tools
- [ ] Implement `planq_checkpoint` and `planq_handoff` tools
- [ ] Add `planq mcp install` command
- [ ] Add `planq mcp serve` command (for debugging)

### Phase 3: Hooks
- [ ] Create `internal/hooks` package
- [ ] Implement `SessionStart` hook
- [ ] Implement `PreCompact` hook
- [ ] Implement `PostCommit` hook (optional)
- [ ] Add hook configuration to `.planq/config`

### Phase 4: CLI Commands
- [ ] Add `planq log` command family (CLI fallback)
- [ ] Add `planq context` command family
- [ ] Add `planq state` commands (init, push, pull, gc)

### Phase 5: Context & Sharing
- [ ] Implement branch context files (`branches/{name}.md`)
- [ ] Implement global branch for shared learnings
- [ ] Add promote/merge workflow

### Phase 6: Remote Sharing
- [ ] Add remote configuration to state repo
- [ ] Implement `planq state push/pull`
- [ ] Handle merge conflicts in changelogs
- [ ] Team visibility dashboard (optional)

---

## Files to Create/Modify

| File | Change |
|------|--------|
| `internal/staterepo/staterepo.go` | **New** - State repo management (bare repo, worktrees) |
| `internal/staterepo/worktree.go` | **New** - Worktree add/remove operations |
| `internal/mcp/server.go` | **New** - MCP server with stdio transport |
| `internal/mcp/tools.go` | **New** - Tool implementations (context, scratch, log, etc.) |
| `internal/mcp/schemas.go` | **New** - Tool input/output schemas |
| `internal/hooks/hooks.go` | **New** - Hook definitions and triggers |
| `internal/hooks/session.go` | **New** - SessionStart, PreCompact hooks |
| `internal/changelog/entry.go` | **New** - Entry struct and types |
| `internal/changelog/changelog.go` | **New** - Append/parse operations |
| `internal/changelog/aggregator.go` | **New** - Session-level aggregation |
| `internal/workspace/workspace.go` | **Modify** - Add AgentDir(), InitAgentDir() |
| `internal/cli/create.go` | **Modify** - Call InitAgentDir() |
| `internal/cli/remove.go` | **Modify** - Call CleanupAgentDir() |
| `internal/cli/mcp.go` | **New** - `planq mcp` subcommands (install, serve) |
| `internal/cli/state.go` | **New** - `planq state` subcommands |
| `internal/cli/log.go` | **New** - `planq log` subcommands |
| `internal/cli/context.go` | **New** - `planq context` subcommands |

---

## Verification Plan

1. **Unit tests**: State repo creation, worktree add/remove, changelog parse/write, MCP tool handlers
2. **Integration test**: Full `planq create` → write to `.planq/agent/` → `planq remove` flow
3. **MCP testing**:
   - Run `planq mcp serve` and test tools via MCP inspector
   - Verify `planq mcp install` adds correct config to Claude Code
   - Test each tool writes correct format to `.planq/agent/`
4. **Hook testing**:
   - Verify `SessionStart` surfaces context on new session
   - Verify `PreCompact` prompts for learning capture
5. **Manual verification**:
   - Create workspace, verify `.planq/agent/` is a git repo
   - Run `cd .planq/agent && git log` from within worktree
   - Use MCP tools from Claude Code, verify files updated
   - Remove workspace, verify state repo cleaned up

---

## Open Questions & Future Improvements

### Agent Adoption Concerns

| Concern | Problem | Potential Fix |
|---------|---------|---------------|
| Agent forgets to log | MCP tools require active choice to call | Add `PostToolCall` hook: "You edited 5 files, log this?" |
| Manual promotion | "Promote to global" relies on human discipline | `SessionEnd` hook prompts: "Promote these learnings?" with auto-suggestions |
| When to capture learnings | "Capture learnings" is vague guidance | Provide examples, or detect patterns (e.g., after fixing a bug, after making a decision) |

### Read Path Needs Work

The current design has many *write* tools but underspecified *read* behavior:

- **`planq_context`** - What exactly does it return? Needs to be a structured summary:
  ```yaml
  scratch: "Current focus: investigating auth bug..."
  recent_log: [last 5 entries]
  active_decisions: [relevant to current branch]
  session_learnings: [captured this session]
  ```

- **Historical search** - How does an agent find a decision from 3 sessions ago? Consider adding:
  ```
  planq_search(query: "JWT signing", type: "decision")
  ```

### Format Simplification

The changelog entry format may be overengineered. Consider starting simpler:

**Current (heavy):**
```yaml
---
id: bd-a1b2c3d4
timestamp: 2026-01-20T10:30:00Z
agent: add-auth
type: implement
parent: bd-9f8e7d6c
depends_on: [bd-1a2b3c4d]
status: completed
---
```

**Simpler alternative:**
```markdown
## 2026-01-21T10:30:00Z - implement

Implemented JWT validation. Chose RS256 over HS256 for public key verification.
```

Add structure (IDs, relationships) later if needed for cross-agent coordination.

### CLAUDE.md Integration

Belt and suspenders: hooks inject context, but CLAUDE.md should also mention `.planq/agent/`:

```markdown
## Agent State

Your working memory lives in `.planq/agent/`. Use planq MCP tools to:
- Update scratch notes: `planq_scratch`
- Log significant work: `planq_log`
- Capture learnings: `planq_learn`
- Record decisions: `planq_decide`

Context is automatically surfaced at session start.
```

### Additional Hooks to Consider

| Hook | Trigger | Action |
|------|---------|--------|
| `PostToolCall` | After N file edits | "Log this batch of changes?" |
| `SessionEnd` | Session terminating | "Promote learnings? Handoff context?" |
| `OnError` | After agent encounters error | "Capture this as a learning?" |
| `BranchSwitch` | Switching code branches | Surface branch-specific context from `branches/{name}.md` |

### Learning Examples

Help agents know what's worth capturing:

**Good learnings:**
- "Always run `just check` before committing - pre-commit hooks catch issues"
- "The `workspace.go` file is the central abstraction - start there when debugging"
- "JWT tokens need both expiry AND signature validation - learned from auth bug"

**Not learnings (just work):**
- "Added function X" (that's a log entry)
- "Fixed typo" (too trivial)

### Metrics & Observability (Future)

Track usage to understand what's valuable:
- Which tools are called most?
- Are learnings actually read in future sessions?
- How often does `planq_search` find relevant history?

---

## Incremental Implementation Plan

Each phase is a **stacked PR** that can be tested and evaluated before proceeding. Course corrections are expected between phases.

### Phase 0: Scaffolding
**Goal:** Minimal infrastructure to enable iteration.

| Task | Deliverable |
|------|-------------|
| Create `.planq/agent/` on workspace create | Directory exists in worktree |
| Add `.planq/agent/` to `.gitignore` | Not tracked in code repo |
| Initialize with empty `scratch.md` | File exists |

**Test:** Create workspace, verify `.planq/agent/scratch.md` exists.
**Evaluate:** Is the directory structure right? Any issues with nested paths?

---

### Phase 1: First MCP Tool - Scratch
**Goal:** Prove MCP integration works with simplest possible tool.

| Task | Deliverable |
|------|-------------|
| Create `internal/mcp` package | Basic MCP server skeleton |
| Implement `planq_scratch` tool | Read/write `.planq/agent/scratch.md` |
| Add `planq mcp serve` command | Run server for testing |

**Test:**
- Run `planq mcp serve`, call `planq_scratch` via MCP inspector
- Install in Claude Code, use from conversation

**Evaluate:**
- Does MCP integration work smoothly?
- Is scratch useful? Do we actually use it?
- What's missing?

---

### Phase 2: Context Tool
**Goal:** Surface agent state back to the agent.

| Task | Deliverable |
|------|-------------|
| Implement `planq_context` tool | Returns current scratch + metadata |
| Define context response format | JSON with scratch, timestamps, etc. |

**Test:** Call `planq_context`, verify it returns useful info.
**Evaluate:**
- What does the agent actually need to know?
- Is the format right?
- Should we add more to context?

---

### Phase 3: Simple Changelog
**Goal:** Append-only log of agent work.

| Task | Deliverable |
|------|-------------|
| Implement `planq_log` tool | Append entry to `.planq/agent/changelog.md` |
| Simple format (timestamp + message) | No complex YAML yet |
| Update `planq_context` to include recent log | Last N entries |

**Test:** Log several entries, verify file format, verify context includes them.
**Evaluate:**
- Does logging feel useful or burdensome?
- Is the format right?
- Do we need structure (types, IDs) yet?

---

### Phase 4: SessionStart Hook
**Goal:** Automatically surface context when agent starts.

| Task | Deliverable |
|------|-------------|
| Create `internal/hooks` package | Hook infrastructure |
| Implement `SessionStart` hook | Calls `planq_context` on session start |
| Add `planq mcp install` command | Register MCP + hooks with Claude Code |

**Test:** Start new session, verify context appears automatically.
**Evaluate:**
- Does auto-context help?
- Is it too noisy?
- What should be included/excluded?

---

### Phase 5: Learn & Decide Tools
**Goal:** Structured capture of insights and decisions.

| Task | Deliverable |
|------|-------------|
| Implement `planq_learn` tool | Append to learnings section |
| Implement `planq_decide` tool | Append to decisions section |
| Update `planq_context` | Include recent learnings/decisions |

**Test:** Capture learnings and decisions, verify they appear in context.
**Evaluate:**
- Are these distinct from log entries?
- Are we actually using them?
- What triggers a learning vs. a log?

---

### Phase 6: State Repo (Git Backend)
**Goal:** Persist agent state in separate git repo.

| Task | Deliverable |
|------|-------------|
| Create `internal/staterepo` package | Bare repo management |
| Initialize state repo on first workspace | `~/.planq/state-repos/{project}.git` |
| Mount `.planq/agent/` as worktree | Full git in `.planq/agent/` |
| Migrate from simple files to worktree | Transparent to MCP tools |

**Test:**
- Create workspace, verify `.planq/agent/` is a git worktree
- Run `cd .planq/agent && git log`
- Commit from within `.planq/agent/`

**Evaluate:**
- Is git backend worth the complexity?
- Any issues with nested repos?
- Ready for multi-agent?

---

### Future Phases (After Evaluation)

| Phase | Goal | Depends On |
|-------|------|------------|
| Multi-agent | Multiple workspaces share state repo | Phase 6 |
| Global branch | Cross-session learnings | Phase 6 |
| PreCompact hook | Prompt before context compression | Phase 4 |
| Handoff tool | Structured handoff to another agent | Phase 5 |
| Search tool | Query historical learnings/decisions | Phase 5 |
| Remote sharing | Push/pull state repo | Phase 6 |

---

### Stacked PR Workflow

Each phase maps to a branch in a stack:

```
main
 └── phase-0/scaffolding
      └── phase-1/mcp-scratch
           └── phase-2/context-tool
                └── phase-3/changelog
                     └── ...
```

**Workflow:**
1. Implement phase N on branch `phase-N/description`
2. Test and evaluate
3. If changes needed, update in place
4. If phase design was wrong, fold changes into parent or restructure
5. Submit PR, get review
6. Merge and continue to phase N+1

**Course Correction Points:**
- After Phase 1: Is MCP the right approach? Any friction?
- After Phase 3: Is logging useful? Simplify or enhance format?
- After Phase 4: Are hooks helping or annoying?
- After Phase 6: Is git backend worth it? Continue or simplify?
