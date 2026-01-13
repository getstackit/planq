# Planq

A Go CLI for orchestrating parallel AI agent workspaces using git worktrees and tmux.

## Concept

**Planq** enables running multiple Claude Code (or other AI agent) sessions in parallel, each in its own isolated workspace. It combines:

- **Git worktrees** (via [stackit](https://github.com/getstackit/stackit)) for isolated code workspaces
- **tmux sessions** for multi-pane terminal layouts
- **Claude Code agents** running in plan mode with artifact monitoring

Each workspace is a combination of:
```
Workspace = git worktree + tmux session + AI agent process
```

## Workflow

```bash
# Create a new workspace for a feature
planq new "add-auth"

# This will:
# 1. Create a git worktree via stackit
# 2. Create a tmux session with layout:
#    ├── Main pane (70%): Claude Code in plan mode
#    └── Side pane (30%): Plan file viewer (auto-refresh)
# 3. Attach you to the tmux session

# List all workspaces
planq list
# NAME          STATE     BRANCH                    PATH
# add-auth      running   jonnii/add-auth-wt        ~/stacks/add-auth
# fix-bug       paused    jonnii/fix-bug-wt         ~/stacks/fix-bug

# Reattach to a workspace
planq attach add-auth

# Remove a workspace (cleans up tmux + worktree)
planq remove add-auth
```

## Architecture

```
planq/
├── cmd/planq/main.go             # Entry point
├── internal/
│   ├── cli/                      # Cobra commands (new, list, attach, remove)
│   ├── workspace/                # Core workspace abstraction
│   ├── tmux/                     # tmux integration via gotmux
│   ├── stackit/                  # stackit CLI integration
│   └── config/                   # Configuration management
├── go.mod
└── justfile
```

### Key Abstractions

**Workspace** - The central abstraction combining:
- Git worktree path
- tmux session name (`planq-<name>`)
- Agent process tracking
- State machine (creating → ready → running → paused → stopped)

**Layout** - Defines tmux pane arrangements:
- Default: 70% agent pane + 30% artifacts pane
- Customizable via config

**Stackit Client** - Shells out to stackit CLI for:
- `stackit worktree create <name>` - Create isolated worktree
- `stackit worktree open <name>` - Get worktree path
- `stackit worktree list` - List existing worktrees
- `stackit worktree remove <name>` - Cleanup worktree

## Dependencies

- [gotmux](https://github.com/GianlucaP106/gotmux) - tmux management from Go
- [cobra](https://github.com/spf13/cobra) - CLI framework
- [stackit](https://github.com/getstackit/stackit) - Stacked git branches (external CLI)

## Installation

```bash
go install planq.dev/planq/cmd/planq@latest
```

## Requirements

- Go 1.25+
- tmux
- stackit CLI (for worktree management)
- Claude Code CLI (or other agent)

## Development

```bash
# Build
just build

# Run exploration script
go run cmd/planq/main.go

# Run tests
just test
```

## License

MIT
