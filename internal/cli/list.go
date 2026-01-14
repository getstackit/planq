package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/charmbracelet/lipgloss/v2/table"
	"github.com/spf13/cobra"
	"planq.dev/planq/internal/stackit"
	"planq.dev/planq/internal/tmux"
	"planq.dev/planq/internal/workspace"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all workspaces",
	Long:  `List all planq workspaces (tmux sessions and git worktrees).`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return listWorkspaces()
	},
}

// workspaceEntry represents a combined workspace entry for display.
type workspaceEntry struct {
	Name   string
	Branch string
	Dir    string
	Status string
	Mode   string
}

// listWorkspaces lists all planq workspaces in a unified table.
func listWorkspaces() error {
	// Collect worktrees
	worktreeMap := make(map[string]stackit.WorktreeEntry)
	st := stackit.NewClient()
	worktrees, err := st.WorktreeList()
	if err == nil {
		for _, wt := range worktrees {
			worktreeMap[wt.Name] = wt
		}
	}

	// Collect tmux sessions
	sessionMap := make(map[string]bool)
	tm, err := tmux.NewManager()
	if err == nil {
		sessions, err := tm.ListSessions(sessionPrefix)
		if err == nil {
			for _, s := range sessions {
				// Strip prefix for the name
				name := s.Name
				if len(s.Name) > len(sessionPrefix) {
					name = s.Name[len(sessionPrefix):]
				}
				sessionMap[name] = true
			}
		}
	}

	// Build unified list
	entries := buildWorkspaceEntries(worktreeMap, sessionMap)

	if len(entries) == 0 {
		fmt.Println("No workspaces found")
		return nil
	}

	// Build and render table
	t := table.New().
		Headers("NAME", "BRANCH", "DIR", "STATUS", "MODE").
		Rows(entriesToRows(entries)...)

	fmt.Fprintln(os.Stdout, t)

	return nil
}

// buildWorkspaceEntries combines worktrees and sessions into workspace entries.
func buildWorkspaceEntries(worktrees map[string]stackit.WorktreeEntry, sessions map[string]bool) []workspaceEntry {
	seen := make(map[string]bool)
	var entries []workspaceEntry

	// Add all worktrees
	for name, wt := range worktrees {
		status := "inactive"
		if sessions[name] {
			status = "active"
		}

		// Get mode from workspace
		mode := "-"
		ws := &workspace.Workspace{Name: name, WorktreePath: wt.Path}
		if m, err := ws.GetMode(); err == nil {
			mode = string(m)
		}

		entries = append(entries, workspaceEntry{
			Name:   name,
			Branch: wt.Branch,
			Dir:    filepath.Base(wt.Path),
			Status: status,
			Mode:   mode,
		})
		seen[name] = true
	}

	// Add orphaned sessions (sessions without worktrees)
	for name := range sessions {
		if !seen[name] {
			entries = append(entries, workspaceEntry{
				Name:   name,
				Branch: "-",
				Dir:    "-",
				Status: "orphaned",
				Mode:   "-",
			})
		}
	}

	// Sort by name
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	return entries
}

// entriesToRows converts workspace entries to table rows.
func entriesToRows(entries []workspaceEntry) [][]string {
	rows := make([][]string, len(entries))
	for i, e := range entries {
		rows[i] = []string{e.Name, e.Branch, e.Dir, e.Status, e.Mode}
	}
	return rows
}
