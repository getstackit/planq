package workspace

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed templates/planq-mode.md
var planqModeSkill string

const (
	// PlanqDirName is the name of the planq directory within a worktree.
	PlanqDirName = ".planq"
	// ClaudeDirName is the name of the Claude configuration directory.
	ClaudeDirName = ".claude"
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
