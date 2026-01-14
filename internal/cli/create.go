package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"planq.dev/planq/internal/stackit"
	"planq.dev/planq/internal/tmux"
	"planq.dev/planq/internal/workspace"
)

var (
	createScope    string
	createAgentCmd string
	createDetach   bool
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new workspace",
	Long:  `Create a new workspace with a git worktree and tmux session.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return createWorkspace(args[0], createScope, createAgentCmd, createDetach)
	},
}

func init() {
	createCmd.Flags().StringVarP(&createScope, "scope", "s", "", "Scope for worktree (optional)")
	createCmd.Flags().StringVarP(&createAgentCmd, "agent-cmd", "a", "", "Command to run in agent pane (default: claude)")
	createCmd.Flags().BoolVarP(&createDetach, "detach", "d", false, "Create workspace without opening it")
}

// createWorkspace creates a new workspace with worktree + tmux session.
func createWorkspace(name, scope, agentCmd string, detach bool) error {
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

	// Create workspace and initialize .planq directory
	ws := &workspace.Workspace{
		Name:         name,
		WorktreePath: workdir,
	}

	fmt.Printf("  Initializing .planq directory...\n")
	if err := ws.InitPlanqDir(); err != nil {
		// Cleanup worktree if .planq creation fails
		_ = st.WorktreeRemove(name)
		return fmt.Errorf("failed to initialize .planq directory: %w", err)
	}
	fmt.Printf("  Plan file will be at: %s\n", ws.PlanFile())

	// Determine agent command (use workspace default unless overridden)
	finalAgentCmd := ws.AgentCommand()
	if agentCmd != "" {
		finalAgentCmd = agentCmd
	}

	// Create tmux session with layout
	// Layout: agent (left), plan viewer (top-right), terminal (bottom-right)
	fmt.Printf("  Creating tmux session %q...\n", sessionName)
	layout := tmux.Layout{
		Name: "agent-plan-terminal",
		Panes: []tmux.PaneSpec{
			{Name: "agent", Size: 60, Command: finalAgentCmd},
			{Name: "plan", Size: 20, Command: fmt.Sprintf("glow %s --tui", ws.PlanFile())},
			{Name: "terminal", Size: 20, Command: ""},
		},
	}

	session, err := tm.CreateSession(sessionName, workdir, layout)
	if err != nil {
		// Cleanup worktree if session creation fails
		_ = st.WorktreeRemove(name)
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Set PLANQ_WORKSPACE environment variable in the session
	if err := tm.SetEnvironment(sessionName, "PLANQ_WORKSPACE", name); err != nil {
		fmt.Printf("  Warning: failed to set PLANQ_WORKSPACE: %v\n", err)
	}

	// Bind mode toggle keybinding (Ctrl-B m)
	if err := tm.BindModeToggle(sessionName, name); err != nil {
		fmt.Printf("  Warning: failed to bind mode toggle key: %v\n", err)
	}

	fmt.Printf("  Session created: %s\n", session.Name)
	fmt.Println()
	fmt.Printf("Workspace %q created successfully!\n", name)

	if detach {
		fmt.Println()
		fmt.Printf("To open: planq open %s\n", name)
		return nil
	}

	// Attach to the session
	fmt.Println()
	fmt.Println("Opening workspace...")
	return tm.AttachSession(sessionName)
}
