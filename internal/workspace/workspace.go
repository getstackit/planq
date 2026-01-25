package workspace

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed templates/planq-mode.md
var planqModeSkill string

const (
	// PlanqDirName is the name of the planq directory within a worktree.
	PlanqDirName = ".planq"
	// ClaudeDirName is the name of the Claude configuration directory.
	ClaudeDirName = ".claude"
	// AgentSubdirName is the name of the agent state subdirectory within .planq.
	AgentSubdirName = "agent"
)

// Workspace represents a planq workspace with its configuration.
type Workspace struct {
	Name         string
	WorktreePath string
}

// PlanqDir returns the path to the .planq directory.
func (w *Workspace) PlanqDir() string {
	return filepath.Join(w.WorktreePath, PlanqDirName)
}

// ClaudeCommandsDir returns the path to the .claude/commands directory.
func (w *Workspace) ClaudeCommandsDir() string {
	return filepath.Join(w.WorktreePath, ClaudeDirName, "commands")
}

// PlanFile returns the path to the plan file (named after the workspace).
func (w *Workspace) PlanFile() string {
	return filepath.Join(w.PlanqDir(), w.Name+".md")
}

// InitPlanqDir creates the .planq directory structure and empty plan file.
func (w *Workspace) InitPlanqDir() error {
	dirs := []string{
		w.PlanqDir(),
		filepath.Join(w.PlanqDir(), "artifacts"),
		w.ClaudeCommandsDir(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create empty plan file so glow has something to display
	planFile := w.PlanFile()
	if err := os.WriteFile(planFile, []byte{}, 0644); err != nil {
		return fmt.Errorf("failed to create plan file %s: %w", planFile, err)
	}

	// Create planq-mode skill for Claude
	skillFile := filepath.Join(w.ClaudeCommandsDir(), "planq-mode.md")
	if err := os.WriteFile(skillFile, []byte(planqModeSkill), 0644); err != nil {
		return fmt.Errorf("failed to create skill file %s: %w", skillFile, err)
	}

	// Initialize mode to plan
	if err := w.SetMode(ModePlan); err != nil {
		return fmt.Errorf("failed to initialize mode: %w", err)
	}

	return nil
}

// AgentCommand returns the Claude command configured for the current mode.
func (w *Workspace) AgentCommand() string {
	mode, err := w.GetMode()
	if err != nil {
		mode = ModePlan // default to plan mode on error
	}

	switch mode {
	case ModeExecute:
		return w.executeAgentCommand()
	default:
		return w.planAgentCommand()
	}
}

// planAgentCommand returns the Claude command for plan mode.
func (w *Workspace) planAgentCommand() string {
	planFile := w.PlanFile()
	systemPrompt := fmt.Sprintf(
		"You are in planning mode for the planq workspace %q. "+
			"You MUST write your implementation plan to %s. This is a REQUIREMENT. "+
			"Do NOT make any code changes. Do NOT use any other file for planning. "+
			"Read from and write to ONLY this plan file. "+
			"This file will be displayed in the artifacts pane for user review. "+
			"Wait for explicit user approval before proceeding with any implementation.",
		w.Name,
		planFile,
	)
	return fmt.Sprintf("claude --append-system-prompt %q", systemPrompt)
}

// executeAgentCommand returns the Claude command for execute mode.
func (w *Workspace) executeAgentCommand() string {
	planFile := w.PlanFile()
	systemPrompt := fmt.Sprintf(
		"You are in execution mode for the planq workspace %q. "+
			"Follow the implementation plan at %s. "+
			"Implement each step carefully.",
		w.Name,
		planFile,
	)
	return fmt.Sprintf("claude --append-system-prompt %q", systemPrompt)
}

// AgentDir returns the path to the .planq/agent directory.
func (w *Workspace) AgentDir() string {
	return filepath.Join(w.PlanqDir(), AgentSubdirName)
}

// InitAgentDir creates the .planq/agent directory structure with initial files.
func (w *Workspace) InitAgentDir() error {
	agentDir := w.AgentDir()

	// Create agent directory and plans subdirectory
	dirs := []string{
		agentDir,
		w.AgentPlansDir(),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
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

	// Configure Claude to use agent plans directory
	if err := w.ConfigureClaudeSettings(); err != nil {
		return fmt.Errorf("failed to configure Claude settings: %w", err)
	}

	return nil
}

// CleanupAgentDir removes the .planq/agent directory.
func (w *Workspace) CleanupAgentDir() error {
	agentDir := w.AgentDir()

	if _, err := os.Stat(agentDir); os.IsNotExist(err) {
		return nil // Nothing to clean up
	}

	if err := os.RemoveAll(agentDir); err != nil {
		return fmt.Errorf("failed to remove agent directory: %w", err)
	}

	return nil
}

// AgentPlansDir returns the path to the .planq/agent/plans directory.
func (w *Workspace) AgentPlansDir() string {
	return filepath.Join(w.AgentDir(), "plans")
}

// ClaudeSettingsFile returns the path to the .claude/settings.json file.
func (w *Workspace) ClaudeSettingsFile() string {
	return filepath.Join(w.WorktreePath, ClaudeDirName, "settings.json")
}

// ConfigureClaudeSettings creates or updates .claude/settings.json with planq-specific settings.
// It merges with existing settings to preserve any configuration copied from the main repo
// (e.g., by stackit hooks).
func (w *Workspace) ConfigureClaudeSettings() error {
	settingsFile := w.ClaudeSettingsFile()

	// Read existing settings if present, using map to preserve unknown fields
	settings := make(map[string]any)
	if data, err := os.ReadFile(settingsFile); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("failed to parse existing settings: %w", err)
		}
	}

	// Merge in plansDirectory (overwrites if already set)
	settings["plansDirectory"] = ".planq/agent/plans"

	// Write settings
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(settingsFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	return nil
}

// ensureGitignore adds an entry to .gitignore if not present.
func (w *Workspace) ensureGitignore(entry string) error {
	gitignorePath := filepath.Join(w.WorktreePath, ".gitignore")

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
