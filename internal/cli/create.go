package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"planq.dev/planq/internal/stackit"
	"planq.dev/planq/internal/tmux"
)

var (
	createScope    string
	createAgentCmd string
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new workspace",
	Long:  `Create a new workspace with a git worktree and tmux session.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return createWorkspace(args[0], createScope, createAgentCmd)
	},
}

func init() {
	createCmd.Flags().StringVarP(&createScope, "scope", "s", "", "Scope for worktree (optional)")
	createCmd.Flags().StringVarP(&createAgentCmd, "agent-cmd", "a", "", "Command to run in agent pane (default: claude)")
}

// createWorkspace creates a new workspace with worktree + tmux session.
func createWorkspace(name, scope, agentCmd string) error {
	sessionName := sessionPrefix + name

	fmt.Printf("Creating workspace %q...\n", name)

	// Check if session already exists
	tm, err := tmux.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize tmux: %w", err)
	}

	exists, err := tm.SessionExists(sessionName)
	if err != nil {
		return fmt.Errorf("failed to check session: %w", err)
	}
	if exists {
		return fmt.Errorf("session %q already exists, use 'planq open %s' to open it", sessionName, name)
	}

	// Create worktree via stackit
	fmt.Printf("  Creating worktree via stackit...\n")
	st := stackit.NewClient()
	if err := st.WorktreeCreate(name, scope); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Get worktree path
	workdir, err := st.WorktreeOpen(name)
	if err != nil {
		return fmt.Errorf("failed to get worktree path: %w", err)
	}
	fmt.Printf("  Worktree created at: %s\n", workdir)

	// Determine agent command
	if agentCmd == "" {
		agentCmd = "claude" // Default to claude
	}

	// Create tmux session with layout
	fmt.Printf("  Creating tmux session %q...\n", sessionName)
	layout := tmux.Layout{
		Name: "agent-artifact",
		Panes: []tmux.PaneSpec{
			{Name: "agent", Size: 70, Command: agentCmd},
			{Name: "artifacts", Size: 30, Command: fmt.Sprintf("watch -n 2 'ls -la %s'", workdir)},
		},
	}

	session, err := tm.CreateSession(sessionName, workdir, layout)
	if err != nil {
		// Cleanup worktree if session creation fails
		_ = st.WorktreeRemove(name)
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	fmt.Printf("  Session created: %s\n", session.Name)
	fmt.Println()
	fmt.Printf("Workspace %q created successfully!\n", name)
	fmt.Println()
	fmt.Printf("To open: planq open %s\n", name)

	return nil
}
