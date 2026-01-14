// Package tmux provides a wrapper around gotmux for workspace session management.
package tmux

import (
	"fmt"

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
