# Planq

> **Note:** This project is a work in progress. APIs and features may change.

A Go CLI for orchestrating parallel AI agent workspaces using git worktrees and tmux.

## Concept

**Planq** enables running multiple Claude Code (or other AI agent) sessions in parallel, each in its own isolated workspace. It combines:

- **Git worktrees** (via [stackit](https://github.com/getstackit/stackit)) for isolated code workspaces
- **tmux sessions** for multi-pane terminal layouts
- **Plan/Execute modes** for structured agent workflows
- **Agent state** for persistent scratch pads and context

Each workspace is a combination of:
```
Workspace = git worktree + tmux session + agent state + AI agent process
```

## Workflow

```bash
# Create a new workspace for a feature
planq create add-auth

# This will:
# 1. Create a git worktree via stackit
# 2. Create .planq/ directory with plan file and agent state
# 3. Create a tmux session with 3-pane layout:
#    ├── Agent pane (60%): Claude Code in plan mode
#    ├── Plan pane (20%): Plan file viewer (glow)
#    └── Terminal pane (20%): Shell for manual commands
# 4. Attach you to the tmux session

# List all workspaces
planq list

# Switch between plan and execute modes (Ctrl-B m in tmux)
planq mode toggle

# Reopen a workspace
planq open add-auth

# Remove a workspace (cleans up tmux + worktree)
planq remove add-auth

# Clean up orphaned workspaces
planq clean
```

## Workspace Structure

Each workspace creates:

```
your-project/
├── .planq/
│   ├── {name}.md           # Plan file (reviewed in plan pane)
│   ├── mode.json           # Current mode (plan/execute)
│   ├── artifacts/          # Generated artifacts
│   └── agent/              # Agent state (gitignored)
│       └── scratch.md      # Agent's working notes
└── [project files]
```

## Architecture

```
planq/
├── cmd/planq/              # Entry point
├── internal/
│   ├── cli/                # Cobra commands (create, list, open, remove, mode)
│   ├── workspace/          # Core workspace abstraction
│   ├── tmux/               # tmux integration via gotmux
│   ├── stackit/            # stackit CLI integration
│   ├── state/              # Global state management
│   ├── git/                # Git utilities
│   └── deps/               # Dependency validation
├── go.mod
└── mise.toml
```

### Key Abstractions

**Workspace** - The central abstraction combining:
- Git worktree path
- tmux session name (`planq-<name>`)
- Plan file and agent state
- Mode (plan or execute)

**Modes**:
- **Plan mode**: Agent writes to plan file, no code changes
- **Execute mode**: Agent implements the plan

**Agent State** - Persistent context in `.planq/agent/`:
- `scratch.md`: Working notes that survive session restarts

## Dependencies

- [gotmux](https://github.com/GianlucaP106/gotmux) - tmux management from Go
- [cobra](https://github.com/spf13/cobra) - CLI framework
- [stackit](https://github.com/getstackit/stackit) - Stacked git branches (external CLI)
- [glow](https://github.com/charmbracelet/glow) - Markdown viewer (optional, for plan pane)

## Installation

```bash
go install planq.dev/planq/cmd/planq@latest
```

## Requirements

- Go 1.25+
- tmux
- stackit CLI (for worktree management)
- Claude Code CLI (or other agent)
- glow (optional, for plan viewing)

## MCP Server

Planq includes an MCP (Model Context Protocol) server that exposes tools to Claude Code. This enables Claude to queue work items (plans, bugs, ideas) directly through the MCP protocol.

### Available Tools

| Tool | Description |
|------|-------------|
| `planq_queue` | Save work for later. Queue a plan, bug, or idea to revisit. |
| `planq_list` | List all queued items, sorted oldest first. |

### Setup

**Option 1: Project-level configuration (recommended)**

Create a `.mcp.json` file in your project root:

```json
{
  "mcpServers": {
    "planq": {
      "type": "stdio",
      "command": "/path/to/planq",
      "args": ["mcp"]
    }
  }
}
```

If running from a non-git directory or a parent monorepo, set `PLANQ_PROJECT_ROOT`:

```json
{
  "mcpServers": {
    "planq": {
      "type": "stdio",
      "command": "/path/to/planq",
      "args": ["mcp"],
      "env": {
        "PLANQ_PROJECT_ROOT": "/path/to/your/project"
      }
    }
  }
}
```

**Option 2: Using Claude CLI**

```bash
# Add to current project (stored in ~/.claude.json)
claude mcp add --transport stdio planq -- planq mcp

# Add globally (available in all projects)
claude mcp add --transport stdio planq --scope user -- planq mcp

# Add to project (creates .mcp.json, can be checked into git)
claude mcp add --transport stdio planq --scope project -- planq mcp
```

### Verify Setup

```bash
# List configured MCP servers
claude mcp list

# Check status in Claude Code
/mcp
```

### Development

To dogfood the MCP server locally:

```bash
# Build the binary
mise run build

# The .mcp.json is pre-configured to use ./bin/planq
```

## Development

```bash
# Install mise and project tools
brew install mise  # or: curl https://mise.run | sh
mise install

# Build
mise run build

# Run tests
mise run test

# Run all checks (fmt, lint, test)
mise run check
```

## License

MIT
