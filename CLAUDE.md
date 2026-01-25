# Planq

Go CLI for orchestrating parallel AI agent workspaces using git worktrees and tmux.

**Tech stack:** Go 1.25, Cobra, gotmux, mise

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
| `rg` (ripgrep) | `grep` | Fast text search, respects .gitignore |
| `fd` | `find` | Fast file search, intuitive syntax |
| `ast-grep` | `sed` for code | AST-based code search and refactoring |
| `jq` | - | JSON processing |
| `yq` | - | YAML processing |
| `tokei` | `wc -l` | Code statistics and language breakdown |

**Examples:**
```bash
rg "func.*Create" --type go          # Search for Create functions in Go files
fd "\.go$" internal/                  # Find all Go files in internal/
ast-grep -p 'fmt.Errorf($$$)' .       # Find all fmt.Errorf calls
tokei                                 # Get codebase statistics
jq '.dependencies' package.json       # Parse JSON
```

## Requirements

**All changes must pass tests and lint before committing:**

```bash
mise run check         # Runs fmt, lint, and tests
mise run test          # Run all tests
mise run lint          # Run linter
mise run build         # Build binary
```

**Workflow:** Run `mise run check` during development for quick feedback.

## Build

```bash
mise run build   # Builds ./bin/planq binary
mise run deps    # Install dependencies
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
