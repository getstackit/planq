package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Mode represents the workspace mode.
type Mode string

const (
	// ModePlan is the planning mode where Claude creates implementation plans.
	ModePlan Mode = "plan"
	// ModeExecute is the execution mode where Claude implements the plan.
	ModeExecute Mode = "execute"
)

// ModeState tracks the current mode and when it was set.
type ModeState struct {
	Mode       Mode      `json:"mode"`
	SwitchedAt time.Time `json:"switched_at"`
}

// ModeFile returns the path to the mode state file.
func (w *Workspace) ModeFile() string {
	return filepath.Join(w.PlanqDir(), "mode.json")
}

// GetMode returns the current workspace mode.
func (w *Workspace) GetMode() (Mode, error) {
	data, err := os.ReadFile(w.ModeFile())
	if err != nil {
		if os.IsNotExist(err) {
			return ModePlan, nil // default to plan mode
		}
		return "", fmt.Errorf("failed to read mode file: %w", err)
	}

	var state ModeState
	if err := json.Unmarshal(data, &state); err != nil {
		return "", fmt.Errorf("failed to parse mode file: %w", err)
	}

	return state.Mode, nil
}

// SetMode updates the workspace mode.
func (w *Workspace) SetMode(mode Mode) error {
	state := ModeState{
		Mode:       mode,
		SwitchedAt: time.Now(),
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal mode state: %w", err)
	}

	if err := os.WriteFile(w.ModeFile(), data, 0644); err != nil {
		return fmt.Errorf("failed to write mode file: %w", err)
	}

	return nil
}

// ToggleMode switches between plan and execute modes.
func (w *Workspace) ToggleMode() (Mode, error) {
	current, err := w.GetMode()
	if err != nil {
		return "", err
	}

	var newMode Mode
	if current == ModePlan {
		newMode = ModeExecute
	} else {
		newMode = ModePlan
	}

	if err := w.SetMode(newMode); err != nil {
		return "", err
	}

	return newMode, nil
}
