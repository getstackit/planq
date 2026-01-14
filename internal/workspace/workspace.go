package workspace

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// PlanqDirName is the name of the planq directory within a worktree.
	PlanqDirName = ".planq"
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

// PlanFile returns the path to the plan file (named after the workspace).
func (w *Workspace) PlanFile() string {
	return filepath.Join(w.PlanqDir(), w.Name+".md")
}

// InitPlanqDir creates the .planq directory structure and empty plan file.
func (w *Workspace) InitPlanqDir() error {
	dirs := []string{
		w.PlanqDir(),
		filepath.Join(w.PlanqDir(), "artifacts"),
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

	return nil
}

// AgentCommand returns the Claude command configured for plan mode.
func (w *Workspace) AgentCommand() string {
	planFile := w.PlanFile()
	systemPrompt := fmt.Sprintf(
		"You are in planning mode for the planq workspace %q. "+
			"Write your implementation plan to %s. "+
			"This file will be displayed in the artifacts pane.",
		w.Name,
		planFile,
	)
	return fmt.Sprintf("claude --plan --append-system-prompt %q", systemPrompt)
}
