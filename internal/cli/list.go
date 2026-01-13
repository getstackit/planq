package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"planq.dev/planq/internal/stackit"
	"planq.dev/planq/internal/tmux"
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

// listWorkspaces lists all planq workspaces.
func listWorkspaces() error {
	fmt.Println("=== Planq Workspaces ===")
	fmt.Println()

	// List tmux sessions
	tm, err := tmux.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize tmux: %w", err)
	}

	sessions, err := tm.ListSessions(sessionPrefix)
	if err != nil {
		fmt.Println("No tmux sessions found (tmux server may not be running)")
	} else if len(sessions) == 0 {
		fmt.Println("No planq sessions found")
	} else {
		fmt.Printf("%-20s %-10s\n", "NAME", "WINDOWS")
		fmt.Printf("%-20s %-10s\n", "----", "-------")
		for _, s := range sessions {
			// Strip prefix for display
			displayName := s.Name
			if len(s.Name) > len(sessionPrefix) {
				displayName = s.Name[len(sessionPrefix):]
			}
			fmt.Printf("%-20s %-10d\n", displayName, s.Windows)
		}
	}
	fmt.Println()

	// List stackit worktrees
	fmt.Println("=== Stackit Worktrees ===")
	fmt.Println()
	st := stackit.NewClient()
	worktrees, err := st.WorktreeList()
	if err != nil {
		fmt.Printf("Could not list worktrees: %v\n", err)
	} else if len(worktrees) == 0 {
		fmt.Println("No worktrees found")
	} else {
		fmt.Printf("%-20s %-50s %-30s\n", "NAME", "PATH", "BRANCH")
		fmt.Printf("%-20s %-50s %-30s\n", "----", "----", "------")
		for _, wt := range worktrees {
			fmt.Printf("%-20s %-50s %-30s\n", wt.Name, wt.Path, wt.Branch)
		}
	}

	return nil
}
