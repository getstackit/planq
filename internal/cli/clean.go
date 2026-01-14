package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"planq.dev/planq/internal/stackit"
	"planq.dev/planq/internal/tmux"
)

var cleanDryRun bool

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up orphaned workspaces",
	Long:  `Remove tmux sessions that no longer have corresponding worktrees.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cleanOrphaned()
	},
}

func init() {
	cleanCmd.Flags().BoolVarP(&cleanDryRun, "dry-run", "n", false, "Show what would be cleaned without removing")
}

// cleanOrphaned removes orphaned tmux sessions.
func cleanOrphaned() error {
	// Get worktree names
	worktreeNames := make(map[string]bool)
	st := stackit.NewClient()
	worktrees, err := st.WorktreeList()
	if err == nil {
		for _, wt := range worktrees {
			worktreeNames[wt.Name] = true
		}
	}

	// Get tmux sessions
	tm, err := tmux.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize tmux: %w", err)
	}

	sessions, err := tm.ListSessions(sessionPrefix)
	if err != nil {
		fmt.Println("No tmux sessions found")
		return nil
	}

	// Find orphaned sessions
	var orphaned []string
	for _, s := range sessions {
		name := s.Name
		if len(s.Name) > len(sessionPrefix) {
			name = s.Name[len(sessionPrefix):]
		}
		if !worktreeNames[name] {
			orphaned = append(orphaned, s.Name)
		}
	}

	if len(orphaned) == 0 {
		fmt.Println("No orphaned sessions found")
		return nil
	}

	if cleanDryRun {
		fmt.Println("Would remove the following orphaned sessions:")
		for _, name := range orphaned {
			fmt.Printf("  - %s\n", name)
		}
		return nil
	}

	// Kill orphaned sessions
	for _, sessionName := range orphaned {
		fmt.Printf("Removing orphaned session: %s\n", sessionName)
		if err := tm.KillSession(sessionName); err != nil {
			fmt.Printf("  Warning: failed to kill session %s: %v\n", sessionName, err)
		}
	}

	fmt.Printf("Cleaned %d orphaned session(s)\n", len(orphaned))
	return nil
}
