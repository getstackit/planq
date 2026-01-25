package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentDir(t *testing.T) {
	ws := &Workspace{
		Name:         "test-workspace",
		WorktreePath: "/path/to/worktree",
	}

	expected := "/path/to/worktree/.planq/agent"
	if got := ws.AgentDir(); got != expected {
		t.Errorf("AgentDir() = %q, want %q", got, expected)
	}
}

func TestInitAgentDir(t *testing.T) {
	tmpDir := t.TempDir()

	ws := &Workspace{
		Name:         "test-workspace",
		WorktreePath: tmpDir,
	}

	// InitAgentDir requires .planq/ and .claude/ to exist (created by InitPlanqDir in normal flow)
	if err := os.MkdirAll(filepath.Join(tmpDir, ".planq"), 0755); err != nil {
		t.Fatalf("Failed to create .planq: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, ".claude"), 0755); err != nil {
		t.Fatalf("Failed to create .claude: %v", err)
	}

	if err := ws.InitAgentDir(); err != nil {
		t.Fatalf("InitAgentDir() failed: %v", err)
	}

	// Verify .planq/agent/ exists
	agentDir := filepath.Join(tmpDir, ".planq", "agent")
	if _, err := os.Stat(agentDir); os.IsNotExist(err) {
		t.Error(".planq/agent directory not created")
	}

	// Verify .planq/agent/plans/ exists
	plansDir := filepath.Join(agentDir, "plans")
	if _, err := os.Stat(plansDir); os.IsNotExist(err) {
		t.Error(".planq/agent/plans directory not created")
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

	// Verify .claude/settings.json created with plansDirectory
	settingsFile := filepath.Join(tmpDir, ".claude", "settings.json")
	content, err = os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("Failed to read .claude/settings.json: %v", err)
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(content, &settings); err != nil {
		t.Fatalf("Failed to parse settings.json: %v", err)
	}
	if settings["plansDirectory"] != ".planq/agent/plans" {
		t.Errorf("plansDirectory = %q, want %q", settings["plansDirectory"], ".planq/agent/plans")
	}
}

func TestInitAgentDir_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()

	ws := &Workspace{
		Name:         "test-workspace",
		WorktreePath: tmpDir,
	}

	// InitAgentDir requires .planq/ and .claude/ to exist
	if err := os.MkdirAll(filepath.Join(tmpDir, ".planq"), 0755); err != nil {
		t.Fatalf("Failed to create .planq: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, ".claude"), 0755); err != nil {
		t.Fatalf("Failed to create .claude: %v", err)
	}

	// Call twice - should not fail
	if err := ws.InitAgentDir(); err != nil {
		t.Fatalf("First InitAgentDir() failed: %v", err)
	}
	if err := ws.InitAgentDir(); err != nil {
		t.Fatalf("Second InitAgentDir() failed: %v", err)
	}

	// Verify .gitignore doesn't have duplicate entries
	gitignore := filepath.Join(tmpDir, ".gitignore")
	content, err := os.ReadFile(gitignore)
	if err != nil {
		t.Fatalf("Failed to read .gitignore: %v", err)
	}
	count := strings.Count(string(content), ".planq/agent/")
	if count != 1 {
		t.Errorf(".gitignore has %d entries for .planq/agent/, want 1", count)
	}
}

func TestCleanupAgentDir(t *testing.T) {
	tmpDir := t.TempDir()

	ws := &Workspace{
		Name:         "test-workspace",
		WorktreePath: tmpDir,
	}

	// InitAgentDir requires .planq/ and .claude/ to exist
	if err := os.MkdirAll(filepath.Join(tmpDir, ".planq"), 0755); err != nil {
		t.Fatalf("Failed to create .planq: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, ".claude"), 0755); err != nil {
		t.Fatalf("Failed to create .claude: %v", err)
	}

	// Initialize first
	if err := ws.InitAgentDir(); err != nil {
		t.Fatalf("InitAgentDir() failed: %v", err)
	}

	// Verify it exists
	agentDir := filepath.Join(tmpDir, ".planq", "agent")
	if _, err := os.Stat(agentDir); os.IsNotExist(err) {
		t.Fatal(".planq/agent directory should exist before cleanup")
	}

	// Cleanup
	if err := ws.CleanupAgentDir(); err != nil {
		t.Fatalf("CleanupAgentDir() failed: %v", err)
	}

	// Verify .planq/agent/ is gone
	if _, err := os.Stat(agentDir); !os.IsNotExist(err) {
		t.Error(".planq/agent directory still exists after cleanup")
	}
}

func TestCleanupAgentDir_NotExists(t *testing.T) {
	tmpDir := t.TempDir()

	ws := &Workspace{
		Name:         "test-workspace",
		WorktreePath: tmpDir,
	}

	// Cleanup without init - should not fail
	if err := ws.CleanupAgentDir(); err != nil {
		t.Fatalf("CleanupAgentDir() failed on non-existent dir: %v", err)
	}
}

func TestConfigureClaudeSettings(t *testing.T) {
	tmpDir := t.TempDir()

	ws := &Workspace{
		Name:         "test-workspace",
		WorktreePath: tmpDir,
	}

	// Create .claude directory
	if err := os.MkdirAll(filepath.Join(tmpDir, ".claude"), 0755); err != nil {
		t.Fatalf("Failed to create .claude: %v", err)
	}

	if err := ws.ConfigureClaudeSettings(); err != nil {
		t.Fatalf("ConfigureClaudeSettings() failed: %v", err)
	}

	// Verify settings file created
	settingsFile := filepath.Join(tmpDir, ".claude", "settings.json")
	content, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("Failed to read settings.json: %v", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(content, &settings); err != nil {
		t.Fatalf("Failed to parse settings.json: %v", err)
	}

	if settings["plansDirectory"] != ".planq/agent/plans" {
		t.Errorf("plansDirectory = %q, want %q", settings["plansDirectory"], ".planq/agent/plans")
	}
}

func TestConfigureClaudeSettings_PreservesExisting(t *testing.T) {
	tmpDir := t.TempDir()

	ws := &Workspace{
		Name:         "test-workspace",
		WorktreePath: tmpDir,
	}

	// Create .claude directory with existing settings
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude: %v", err)
	}

	// Write existing settings with other fields (simulating stackit hook copy)
	existingSettings := `{"apiKey": "secret", "model": "claude-3", "someOtherSetting": "value"}`
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(existingSettings), 0644); err != nil {
		t.Fatalf("Failed to write existing settings: %v", err)
	}

	if err := ws.ConfigureClaudeSettings(); err != nil {
		t.Fatalf("ConfigureClaudeSettings() failed: %v", err)
	}

	// Verify settings
	content, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	if err != nil {
		t.Fatalf("Failed to read settings.json: %v", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(content, &settings); err != nil {
		t.Fatalf("Failed to parse settings.json: %v", err)
	}

	// Verify plansDirectory is set
	if settings["plansDirectory"] != ".planq/agent/plans" {
		t.Errorf("plansDirectory = %q, want %q", settings["plansDirectory"], ".planq/agent/plans")
	}

	// Verify existing settings are preserved
	if settings["apiKey"] != "secret" {
		t.Errorf("apiKey was not preserved, got %v", settings["apiKey"])
	}
	if settings["model"] != "claude-3" {
		t.Errorf("model was not preserved, got %v", settings["model"])
	}
	if settings["someOtherSetting"] != "value" {
		t.Errorf("someOtherSetting was not preserved, got %v", settings["someOtherSetting"])
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
		{
			name:     "no file exists",
			existing: "",
			entry:    ".planq/agent/",
			expected: ".planq/agent/\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			gitignore := filepath.Join(tmpDir, ".gitignore")

			// Write existing content (skip for "no file exists" case)
			if tt.name != "no file exists" && tt.existing != "" {
				if err := os.WriteFile(gitignore, []byte(tt.existing), 0644); err != nil {
					t.Fatalf("Failed to write existing .gitignore: %v", err)
				}
			} else if tt.name != "no file exists" {
				// Create empty file for "empty file" case
				if err := os.WriteFile(gitignore, []byte{}, 0644); err != nil {
					t.Fatalf("Failed to create empty .gitignore: %v", err)
				}
			}

			ws := &Workspace{WorktreePath: tmpDir}
			if err := ws.ensureGitignore(tt.entry); err != nil {
				t.Fatalf("ensureGitignore() failed: %v", err)
			}

			content, err := os.ReadFile(gitignore)
			if err != nil {
				t.Fatalf("Failed to read .gitignore: %v", err)
			}
			if string(content) != tt.expected {
				t.Errorf("got %q, want %q", string(content), tt.expected)
			}
		})
	}
}
