// Package tmux provides a wrapper around gotmux for workspace session management.
package tmux

import (
	"fmt"
	"os/exec"

	"github.com/GianlucaP106/gotmux/gotmux"
)

// Manager handles tmux session and pane operations.
type Manager struct {
	tmux *gotmux.Tmux
}

// NewManager creates a new tmux manager.
func NewManager() (*Manager, error) {
	t, err := gotmux.DefaultTmux()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tmux: %w", err)
	}
	return &Manager{tmux: t}, nil
}

// Layout defines the pane arrangement for a workspace.
type Layout struct {
	Name        string
	Description string
	Panes       []PaneSpec
}

// PaneSpec defines a single pane in a layout.
type PaneSpec struct {
	Name    string
	Size    int    // Percentage (0 = auto)
	Command string // Command to run in the pane
}

// DefaultLayout returns the default agent-artifact layout.
func DefaultLayout() Layout {
	return Layout{
		Name:        "agent-artifact",
		Description: "Main pane for agent, side pane for artifacts",
		Panes: []PaneSpec{
			{Name: "agent", Size: 70, Command: ""},
			{Name: "artifacts", Size: 30, Command: ""},
		},
	}
}

// SessionExists checks if a tmux session with the given name exists.
func (m *Manager) SessionExists(name string) (bool, error) {
	sessions, err := m.tmux.ListSessions()
	if err != nil {
		// If tmux server isn't running, no sessions exist
		return false, nil
	}

	for _, s := range sessions {
		if s.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// CreateSession creates a new tmux session with the given layout.
// Layout with 3 panes creates:
//
//	+----------------+--------+
//	|                |  pane1 |
//	|    pane0       +--------+
//	|                |  pane2 |
//	+----------------+--------+
func (m *Manager) CreateSession(name string, workdir string, layout Layout) (*gotmux.Session, error) {
	// Create the session
	session, err := m.tmux.NewSession(&gotmux.SessionOptions{
		Name:           name,
		StartDirectory: workdir,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session %q: %w", name, err)
	}

	// Enable mouse support
	if err := session.SetOption("mouse", "on"); err != nil {
		// Non-fatal, continue without mouse support
		fmt.Printf("Warning: could not enable mouse support: %v\n", err)
	}

	// Get the first window
	windows, err := session.ListWindows()
	if err != nil {
		return nil, fmt.Errorf("failed to list windows: %w", err)
	}
	if len(windows) == 0 {
		return nil, fmt.Errorf("session created but has no windows")
	}
	window := windows[0]

	// Get the main pane
	panes, err := window.ListPanes()
	if err != nil {
		return nil, fmt.Errorf("failed to list panes: %w", err)
	}
	if len(panes) == 0 {
		return nil, fmt.Errorf("window has no panes")
	}

	// Create additional panes based on layout
	if len(layout.Panes) > 1 {
		mainPane := panes[0]

		// Split horizontally (side by side) for the right column
		err = mainPane.SplitWindow(&gotmux.SplitWindowOptions{
			SplitDirection: gotmux.PaneSplitDirectionHorizontal,
			StartDirectory: workdir,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to split pane horizontally: %w", err)
		}
	}

	if len(layout.Panes) > 2 {
		// Get the right pane (pane 1) and split it vertically (top/bottom)
		panes, err = window.ListPanes()
		if err != nil {
			return nil, fmt.Errorf("failed to list panes after first split: %w", err)
		}
		if len(panes) > 1 {
			rightPane := panes[1]
			err = rightPane.SplitWindow(&gotmux.SplitWindowOptions{
				SplitDirection: gotmux.PaneSplitDirectionVertical,
				StartDirectory: workdir,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to split pane vertically: %w", err)
			}
		}
	}

	// Get final list of panes
	panes, err = window.ListPanes()
	if err != nil {
		return nil, fmt.Errorf("failed to list final panes: %w", err)
	}

	// Send commands to each pane
	for i, paneSpec := range layout.Panes {
		if i >= len(panes) {
			break
		}
		if paneSpec.Command != "" {
			if err = panes[i].SendKeys(paneSpec.Command); err != nil {
				return nil, fmt.Errorf("failed to send command to pane %d: %w", i, err)
			}
			if err = panes[i].SendKeys("Enter"); err != nil {
				return nil, fmt.Errorf("failed to send Enter to pane %d: %w", i, err)
			}
		}
	}

	return session, nil
}

// KillSession terminates a tmux session by name.
func (m *Manager) KillSession(name string) error {
	sessions, err := m.tmux.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	for _, s := range sessions {
		if s.Name == name {
			return s.Kill()
		}
	}

	return fmt.Errorf("session %q not found", name)
}

// AttachSession attaches to an existing tmux session.
func (m *Manager) AttachSession(name string) error {
	sessions, err := m.tmux.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	for _, s := range sessions {
		if s.Name == name {
			return s.Attach()
		}
	}

	return fmt.Errorf("session %q not found", name)
}

// ListSessions returns all planq-prefixed sessions.
func (m *Manager) ListSessions(prefix string) ([]*gotmux.Session, error) {
	sessions, err := m.tmux.ListSessions()
	if err != nil {
		return nil, err
	}

	var result []*gotmux.Session
	for _, s := range sessions {
		if prefix == "" || len(s.Name) >= len(prefix) && s.Name[:len(prefix)] == prefix {
			result = append(result, s)
		}
	}
	return result, nil
}

// GetSession returns a session by name.
func (m *Manager) GetSession(name string) (*gotmux.Session, error) {
	sessions, err := m.tmux.ListSessions()
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	for _, s := range sessions {
		if s.Name == name {
			return s, nil
		}
	}

	return nil, fmt.Errorf("session %q not found", name)
}

// paneInfo holds information about a pane's current state.
type paneInfo struct {
	index   int
	command string
}

// getPaneInfo returns information about all panes in a session.
func (m *Manager) getPaneInfo(sessionName string) ([]paneInfo, error) {
	cmd := exec.Command("tmux", "list-panes", "-t", sessionName, "-F", "#{pane_index}:#{pane_current_command}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var panes []paneInfo
	for _, line := range splitLines(string(output)) {
		if line == "" {
			continue
		}
		var idx int
		var command string
		if _, err := fmt.Sscanf(line, "%d:%s", &idx, &command); err != nil {
			continue
		}
		panes = append(panes, paneInfo{index: idx, command: command})
	}
	return panes, nil
}

// paneHasRunningProcess checks if a pane has a running foreground process (not just a shell).
func (m *Manager) paneHasRunningProcess(sessionName string, paneIndex int) bool {
	panes, err := m.getPaneInfo(sessionName)
	if err != nil {
		return false
	}

	for _, p := range panes {
		if p.index == paneIndex {
			return !isShellCommand(p.command)
		}
	}
	return false
}

// splitLines splits a string into lines.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// isShellCommand returns true if the command is a shell.
func isShellCommand(cmd string) bool {
	shells := []string{"bash", "zsh", "sh", "fish", "dash", "ksh", "tcsh", "csh"}
	for _, shell := range shells {
		if cmd == shell || cmd == "-"+shell {
			return true
		}
	}
	return false
}

// layoutMatches checks if the current pane layout matches the target layout.
// Returns true if no reconfiguration is needed.
func (m *Manager) layoutMatches(sessionName string, layout Layout) bool {
	panes, err := m.getPaneInfo(sessionName)
	if err != nil {
		return false
	}

	// Check pane count matches
	if len(panes) != len(layout.Panes) {
		return false
	}

	// Build a map of pane index -> command for easier lookup
	paneByIndex := make(map[int]string)
	for _, p := range panes {
		paneByIndex[p.index] = p.command
	}

	// Check each pane has the expected process running
	for i, spec := range layout.Panes {
		cmd, exists := paneByIndex[i]
		if !exists {
			return false
		}

		// Check if pane matches expectations based on its type
		switch spec.Name {
		case "agent":
			// Agent pane should have a non-shell process running
			// (claude shows up as version number like "2.1.7" or as "node" or "claude")
			if isShellCommand(cmd) {
				return false
			}
		case "plan":
			// Plan pane should be running glow
			if cmd != "glow" {
				return false
			}
		case "diff":
			// Diff pane runs a while loop in a shell, so accept any shell
			// (the loop runs git diff | delta continuously)
		case "terminal":
			// Terminal can be any shell - no specific requirement
		default:
			// Unknown pane type - if it has a command, check it's running something
			if spec.Command != "" && isShellCommand(cmd) {
				return false
			}
		}
	}

	return true
}

// ReconfigureSession reconfigures the session to match the target layout.
// This is idempotent - if the layout already matches, no changes are made.
// If pane 0 has a running process (like claude), it will not be restarted.
// Returns true if changes were made, false if layout already matched.
func (m *Manager) ReconfigureSession(name string, workdir string, layout Layout) (bool, error) {
	// Check if layout already matches - if so, no reconfiguration needed
	if m.layoutMatches(name, layout) {
		return false, nil
	}

	session, err := m.GetSession(name)
	if err != nil {
		return false, err
	}

	// Check if pane 0 has a running process before we do anything
	pane0HasProcess := m.paneHasRunningProcess(name, 0)

	// Get the window
	windows, err := session.ListWindows()
	if err != nil {
		return false, fmt.Errorf("failed to list windows: %w", err)
	}
	if len(windows) == 0 {
		return false, fmt.Errorf("session has no windows")
	}
	window := windows[0]

	// Kill all panes except pane 0
	panes, err := window.ListPanes()
	if err != nil {
		return false, fmt.Errorf("failed to list panes: %w", err)
	}

	// Kill panes in reverse order to avoid index shifting issues
	for i := len(panes) - 1; i > 0; i-- {
		if err := panes[i].Kill(); err != nil {
			return false, fmt.Errorf("failed to kill pane %d: %w", i, err)
		}
	}

	// Now we have a single pane (pane 0). Create the layout from scratch.
	panes, err = window.ListPanes()
	if err != nil {
		return false, fmt.Errorf("failed to list panes after cleanup: %w", err)
	}
	if len(panes) == 0 {
		return false, fmt.Errorf("no panes remaining after cleanup")
	}

	// Create additional panes based on layout
	mainPane := panes[0]

	if len(layout.Panes) > 1 {
		// Split horizontally for the right column
		err = mainPane.SplitWindow(&gotmux.SplitWindowOptions{
			SplitDirection: gotmux.PaneSplitDirectionHorizontal,
			StartDirectory: workdir,
		})
		if err != nil {
			return false, fmt.Errorf("failed to split pane horizontally: %w", err)
		}
	}

	if len(layout.Panes) > 2 {
		// Get the right pane and split it vertically
		panes, err = window.ListPanes()
		if err != nil {
			return false, fmt.Errorf("failed to list panes after first split: %w", err)
		}
		if len(panes) > 1 {
			rightPane := panes[1]
			err = rightPane.SplitWindow(&gotmux.SplitWindowOptions{
				SplitDirection: gotmux.PaneSplitDirectionVertical,
				StartDirectory: workdir,
			})
			if err != nil {
				return false, fmt.Errorf("failed to split pane vertically: %w", err)
			}
		}
	}

	// Get final list of panes and send commands
	panes, err = window.ListPanes()
	if err != nil {
		return false, fmt.Errorf("failed to list final panes: %w", err)
	}

	for i, paneSpec := range layout.Panes {
		if i >= len(panes) {
			break
		}
		// Skip pane 0 if it already has a running process (like claude)
		if i == 0 && pane0HasProcess {
			continue
		}

		// Always cd to workdir first to ensure correct working directory
		// This is more reliable than StartDirectory alone
		if i > 0 {
			cdCmd := fmt.Sprintf("cd '%s'", workdir)
			if err = panes[i].SendKeys(cdCmd); err != nil {
				return false, fmt.Errorf("failed to send cd to pane %d: %w", i, err)
			}
			if err = panes[i].SendKeys("Enter"); err != nil {
				return false, fmt.Errorf("failed to send Enter after cd to pane %d: %w", i, err)
			}
		}

		if paneSpec.Command != "" {
			if err = panes[i].SendKeys(paneSpec.Command); err != nil {
				return false, fmt.Errorf("failed to send command to pane %d: %w", i, err)
			}
			if err = panes[i].SendKeys("Enter"); err != nil {
				return false, fmt.Errorf("failed to send Enter to pane %d: %w", i, err)
			}
		}
	}

	return true, nil
}

// BindModeToggle adds a keybinding (prefix + m) to toggle workspace mode.
func (m *Manager) BindModeToggle(sessionName, workspaceName, worktreePath string) error {
	// Bind 'm' key in this session to run planq mode toggle
	// Quote the worktree path to handle spaces
	cmd := exec.Command("tmux", "bind-key", "-t", sessionName, "m",
		"run-shell", fmt.Sprintf("planq mode toggle --workspace '%s' --worktree '%s'", workspaceName, worktreePath))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bind mode toggle key: %w (output: %s)", err, string(output))
	}
	return nil
}

// SetEnvironment sets an environment variable in the tmux session.
func (m *Manager) SetEnvironment(sessionName, key, value string) error {
	cmd := exec.Command("tmux", "set-environment", "-t", sessionName, key, value)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set environment %s: %w (output: %s)", key, err, string(output))
	}
	return nil
}
