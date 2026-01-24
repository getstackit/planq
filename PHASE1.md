# Phase 0: Scaffolding

**Goal:** Create `.planq/agent/` directory structure on workspace creation. No MCP, no hooks - just the files.

**Branch:** `phase-0/agent-scaffolding`

---

## Context

From codebase exploration:
- Workspace creation lives in `internal/cli/create.go`
- Directory initialization pattern exists in `internal/workspace/workspace.go` (`InitPlanqDir()`)
- Cleanup happens in `internal/cli/remove.go`
- Permissions: dirs `0755`, files `0644`
- Error wrapping: `fmt.Errorf("context: %w", err)`

---

## Deliverables

| Deliverable | Description |
|-------------|-------------|
| `AgentDir()` method | Returns path to `.planq/agent/` directory |
| `InitAgentDir()` method | Creates `.planq/agent/` with initial files |
| `CleanupAgentDir()` method | Removes `.planq/agent/` directory |
| `.gitignore` handling | Adds `.planq/agent/` to project's `.gitignore` |
| Integration in create flow | Called after `InitPlanqDir()` |
| Integration in remove flow | Called before worktree removal |

---

## File Changes

### 1. `internal/workspace/workspace.go`

**Add methods:**

```go
// AgentDir returns the path to the .planq/agent directory
func (w *Workspace) AgentDir() string {
    return filepath.Join(w.PlanqDir(), AgentSubdirName)
}

// InitAgentDir creates the .planq/agent directory structure with initial files
func (w *Workspace) InitAgentDir() error {
    agentDir := w.AgentDir()

    // Create directory
    if err := os.MkdirAll(agentDir, 0755); err != nil {
        return fmt.Errorf("failed to create agent directory %s: %w", agentDir, err)
    }

    // Create initial scratch.md
    scratchFile := filepath.Join(agentDir, "scratch.md")
    scratchContent := []byte("# Scratch\n\nWorking notes for this session.\n")
    if err := os.WriteFile(scratchFile, scratchContent, 0644); err != nil {
        return fmt.Errorf("failed to create scratch file: %w", err)
    }

    // Add .planq/agent to .gitignore
    if err := w.ensureGitignore(".planq/agent/"); err != nil {
        return fmt.Errorf("failed to update .gitignore: %w", err)
    }

    return nil
}

// CleanupAgentDir removes the .planq/agent directory
func (w *Workspace) CleanupAgentDir() error {
    agentDir := w.AgentDir()

    // Check if exists
    if _, err := os.Stat(agentDir); os.IsNotExist(err) {
        return nil // Nothing to clean up
    }

    if err := os.RemoveAll(agentDir); err != nil {
        return fmt.Errorf("failed to remove agent directory: %w", err)
    }

    return nil
}

// ensureGitignore adds an entry to .gitignore if not present
func (w *Workspace) ensureGitignore(entry string) error {
    gitignorePath := filepath.Join(w.WorktreePath, ".gitignore")

    // Read existing content
    content, err := os.ReadFile(gitignorePath)
    if err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("failed to read .gitignore: %w", err)
    }

    // Check if entry already exists
    lines := strings.Split(string(content), "\n")
    for _, line := range lines {
        if strings.TrimSpace(line) == entry {
            return nil // Already present
        }
    }

    // Append entry
    var newContent []byte
    if len(content) > 0 && !bytes.HasSuffix(content, []byte("\n")) {
        newContent = append(content, '\n')
    } else {
        newContent = content
    }
    newContent = append(newContent, []byte(entry+"\n")...)

    if err := os.WriteFile(gitignorePath, newContent, 0644); err != nil {
        return fmt.Errorf("failed to write .gitignore: %w", err)
    }

    return nil
}
```

**Add imports:**
```go
import (
    "bytes"
    "strings"
)
```

---

### 2. `internal/cli/create.go`

**After `ws.InitPlanqDir()` call (around line 137), add:**

```go
// Initialize agent directory
if err := ws.InitAgentDir(); err != nil {
    // Cleanup on failure
    if !isMain {
        _ = stackitClient.WorktreeRemove(name, false)
    }
    return fmt.Errorf("failed to initialize agent directory: %w", err)
}
```

---

### 3. `internal/cli/remove.go`

**Before worktree removal (around line 100), add:**

```go
// Clean up agent directory (best effort, don't fail on error)
ws := &workspace.Workspace{Name: name, WorktreePath: workdir}
if err := ws.CleanupAgentDir(); err != nil {
    fmt.Printf("  Warning: failed to clean up agent directory: %v\n", err)
}
```

---

## Test Plan

### Unit Tests (`internal/workspace/workspace_test.go`)

```go
func TestInitAgentDir(t *testing.T) {
    // Create temp directory
    tmpDir := t.TempDir()

    ws := &Workspace{
        Name:         "test-workspace",
        WorktreePath: tmpDir,
    }

    // Test initialization
    err := ws.InitAgentDir()
    if err != nil {
        t.Fatalf("InitAgentDir failed: %v", err)
    }

    // Verify .planq/agent/ exists
    agentDir := filepath.Join(tmpDir, ".planq", "agent")
    if _, err := os.Stat(agentDir); os.IsNotExist(err) {
        t.Error(".planq/agent directory not created")
    }

    // Verify scratch.md exists and has content
    scratchFile := filepath.Join(agentDir, "scratch.md")
    content, err := os.ReadFile(scratchFile)
    if err != nil {
        t.Fatalf("Failed to read scratch.md: %v", err)
    }
    if !strings.Contains(string(content), "# Scratch") {
        t.Error("scratch.md missing expected content")
    }

    // Verify .gitignore updated
    gitignore := filepath.Join(tmpDir, ".gitignore")
    content, err = os.ReadFile(gitignore)
    if err != nil {
        t.Fatalf("Failed to read .gitignore: %v", err)
    }
    if !strings.Contains(string(content), ".planq/agent/") {
        t.Error(".gitignore missing .planq/agent/ entry")
    }
}

func TestCleanupAgentDir(t *testing.T) {
    tmpDir := t.TempDir()

    ws := &Workspace{
        Name:         "test-workspace",
        WorktreePath: tmpDir,
    }

    // Initialize first
    if err := ws.InitAgentDir(); err != nil {
        t.Fatalf("InitAgentDir failed: %v", err)
    }

    // Cleanup
    if err := ws.CleanupAgentDir(); err != nil {
        t.Fatalf("CleanupAgentDir failed: %v", err)
    }

    // Verify .planq/agent/ is gone
    agentDir := filepath.Join(tmpDir, ".planq", "agent")
    if _, err := os.Stat(agentDir); !os.IsNotExist(err) {
        t.Error(".planq/agent directory still exists after cleanup")
    }
}

func TestEnsureGitignore(t *testing.T) {
    tests := []struct {
        name     string
        existing string
        entry    string
        expected string
    }{
        {
            name:     "empty file",
            existing: "",
            entry:    ".planq/agent/",
            expected: ".planq/agent/\n",
        },
        {
            name:     "existing entries",
            existing: "node_modules/\n.env\n",
            entry:    ".planq/agent/",
            expected: "node_modules/\n.env\n.planq/agent/\n",
        },
        {
            name:     "already present",
            existing: ".planq/agent/\nnode_modules/\n",
            entry:    ".planq/agent/",
            expected: ".planq/agent/\nnode_modules/\n",
        },
        {
            name:     "no trailing newline",
            existing: "node_modules/",
            entry:    ".planq/agent/",
            expected: "node_modules/\n.planq/agent/\n",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tmpDir := t.TempDir()
            gitignore := filepath.Join(tmpDir, ".gitignore")

            // Write existing content
            if tt.existing != "" {
                os.WriteFile(gitignore, []byte(tt.existing), 0644)
            }

            ws := &Workspace{WorktreePath: tmpDir}
            if err := ws.ensureGitignore(tt.entry); err != nil {
                t.Fatalf("ensureGitignore failed: %v", err)
            }

            content, _ := os.ReadFile(gitignore)
            if string(content) != tt.expected {
                t.Errorf("got %q, want %q", string(content), tt.expected)
            }
        })
    }
}
```

---

## Manual Verification

```bash
# 1. Build
just build

# 2. Create a workspace
planq create test-phase0

# 3. Verify .planq/agent/ exists
ls -la .planq/agent/
# Should show: scratch.md

# 4. Verify scratch.md content
cat .planq/agent/scratch.md
# Should show: "# Scratch\n\nWorking notes..."

# 5. Verify .gitignore updated
cat .gitignore | grep ".planq/agent"
# Should show: .planq/agent/

# 6. Remove workspace
planq remove test-phase0

# 7. Verify cleanup (if worktree still accessible)
# .planq/agent/ should be gone
```

---

## Evaluation Questions

After implementing and testing:

1. **Is the directory structure right?**
   - Should we add more initial files (changelog.md, context.md)?
   - Or keep minimal and add in later phases?

2. **Any issues with the location?**
   - Is `.planq/agent/` the right name?
   - Any conflicts with existing tools/conventions?

3. **Gitignore behavior correct?**
   - Should we warn if .gitignore is not tracked?
   - Handle case where .gitignore doesn't exist?

4. **Cleanup behavior correct?**
   - Should cleanup be more aggressive (remove from .gitignore)?
   - Or leave .gitignore entry (harmless)?

---

## Definition of Done

- [ ] `just check` passes (fmt, lint, tests)
- [ ] Unit tests for InitAgentDir, CleanupAgentDir, ensureGitignore
- [ ] Manual verification complete
- [ ] Evaluation questions answered
- [ ] Ready for PR

---

## Next Phase

After Phase 0 is complete and evaluated, proceed to **Phase 1: First MCP Tool - Scratch** which will:
- Create `internal/mcp` package
- Implement `planq_scratch` tool to read/write `.planq/agent/scratch.md`
- Add `planq mcp serve` command for testing
