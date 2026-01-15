package cli

import (
	"os"

	"github.com/spf13/cobra"
	"planq.dev/planq/internal/tmux"
	"planq.dev/planq/internal/workspace"
)

var notifyCmd = &cobra.Command{
	Use:    "notify",
	Short:  "Notification commands for hooks",
	Hidden: true, // Hidden since this is for internal/hook use
}

var notifyStoppedCmd = &cobra.Command{
	Use:   "stopped",
	Short: "Notify that the agent has stopped",
	Long: `Called by Claude Code hook when the agent stops.
If the workspace is not currently attached, marks it as needing review.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleAgentStopped()
	},
}

func init() {
	notifyCmd.AddCommand(notifyStoppedCmd)
}

// handleAgentStopped marks the workspace as needing review if not attached.
func handleAgentStopped() error {
	// Get workspace name from environment
	name := os.Getenv("PLANQ_WORKSPACE")
	if name == "" {
		// Not in a planq workspace, silently exit
		return nil
	}

	// Get worktree path from environment
	workdir := os.Getenv("PLANQ_WORKTREE_PATH")
	if workdir == "" {
		// Try current directory
		var err error
		workdir, err = os.Getwd()
		if err != nil {
			return nil // Silently fail
		}
	}

	// Check if session is attached
	tm, err := tmux.NewManager()
	if err != nil {
		return nil // Silently fail
	}

	sessionName := sessionPrefix + name
	attached, err := tm.IsSessionAttached(sessionName)
	if err != nil {
		return nil // Silently fail
	}

	// If attached, no need to flag for review
	if attached {
		return nil
	}

	// Mark workspace as needing review
	ws := &workspace.Workspace{
		Name:         name,
		WorktreePath: workdir,
	}

	return ws.SetNeedsReview()
}
