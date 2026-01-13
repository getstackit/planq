# Planq

Go CLI for orchestrating parallel AI agent workspaces using git worktrees and tmux.

**Tech stack:** Go 1.25, Cobra, gotmux

## Architecture

- `cmd/planq`: CLI entry point
- `internal/cli`: Cobra command definitions
- `internal/workspace`: Core workspace abstraction (worktree + tmux + agent)
- `internal/tmux`: tmux session/pane management via gotmux
- `internal/stackit`: Shell out to stackit CLI for worktree operations
- `internal/config`: Configuration loading/saving

## CLI Tools

Use these tools instead of standard alternatives:

| Tool | Use Instead Of | Purpose |
|------|----------------|---------|
| `rg` (ripgrep) | `grep` | Fast text search |
| `fd` | `find` | Fast file search |
| `jq` | - | JSON processing |

## Requirements

**All changes must pass tests and lint before committing:**

```bash
just check             # Runs fmt, lint, and tests
just test              # Run tests
just lint              # Run linter
just build             # Build binary
```

## Commit Messages

Use **Conventional Commits**:

```
<type>[optional scope]: <description>
```

**Types:** `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`

**Examples:**
- `feat: add tmux session creation`
- `fix: handle missing worktree gracefully`
- `refactor: extract layout configuration`

## Code Style

- Prefer early returns over deep nesting
- Always handle errors explicitly; never ignore them with `_`
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Write comments for exported functions and types

## Key Patterns

### Shelling out to stackit

```go
// Use the stackit client to shell out to stackit CLI
client := stackit.NewClient()
err := client.WorktreeCreate("my-feature", "feature")
path, err := client.WorktreeOpen("my-feature")
```

### tmux session management

```go
// Use gotmux for tmux operations
server := gotmux.NewServer()
session, err := server.NewSession("planq-my-feature")
pane, err := session.SplitWindow(gotmux.Horizontal, 30)
pane.SendKeys("claude --plan")
```

## Testing

- Use table-driven tests for multiple cases
- Mock external dependencies (stackit CLI, tmux)
