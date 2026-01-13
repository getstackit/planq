package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"planq.dev/planq/internal/stackit"
	"planq.dev/planq/internal/tmux"
)

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a workspace",
	Long:  `Remove a workspace by killing its tmux session and removing the git worktree.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return removeWorkspace(args[0])
	},
}

// removeWorkspace removes a workspace (tmux session + worktree).
func removeWorkspace(name string) error {
	sessionName := sessionPrefix + name

	fmt.Printf("Removing workspace %q...\n", name)

	// Kill tmux session
	tm, err := tmux.NewManager()
	if err != nil {
		fmt.Printf("  Warning: Could not initialize tmux: %v\n", err)
	} else {
		exists, _ := tm.SessionExists(sessionName)
		if exists {
			fmt.Printf("  Killing tmux session %q...\n", sessionName)
			if err := tm.KillSession(sessionName); err != nil {
				fmt.Printf("  Warning: Could not kill session: %v\n", err)
			} else {
				fmt.Println("  Session killed")
			}
		} else {
			fmt.Println("  No tmux session found")
		}
	}

	// Remove worktree
	fmt.Printf("  Removing worktree %q...\n", name)
	st := stackit.NewClient()
	if err := st.WorktreeRemove(name); err != nil {
		// Try force remove
		if err := st.WorktreeRemoveForce(name); err != nil {
			fmt.Printf("  Warning: Could not remove worktree: %v\n", err)
		} else {
			fmt.Println("  Worktree removed (forced)")
		}
	} else {
		fmt.Println("  Worktree removed")
	}

	fmt.Printf("Workspace %q removed\n", name)
	return nil
}
