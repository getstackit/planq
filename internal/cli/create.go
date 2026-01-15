package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"planq.dev/planq/internal/deps"
	"planq.dev/planq/internal/git"
	"planq.dev/planq/internal/stackit"
	"planq.dev/planq/internal/state"
	"planq.dev/planq/internal/tmux"
	"planq.dev/planq/internal/workspace"
)

var (
	createScope    string
	createAgentCmd string
	createDetach   bool
	createMain     bool
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new workspace",
	Long:  `Create a new workspace with a git worktree and tmux session.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return createWorkspace(args[0], createScope, createAgentCmd, createDetach, createMain)
	},
}

func init() {
	createCmd.Flags().StringVarP(&createScope, "scope", "s", "", "Scope for worktree (optional)")
	createCmd.Flags().StringVarP(&createAgentCmd, "agent-cmd", "a", "", "Command to run in agent pane (default: claude)")
	createCmd.Flags().BoolVarP(&createDetach, "detach", "d", false, "Create workspace without opening it")
	createCmd.Flags().BoolVar(&createMain, "main", false, "Use main worktree instead of creating a new one (for testing)")
}

// createWorkspace creates a new workspace with worktree + tmux session.
func createWorkspace(name, scope, agentCmd string, detach, useMain bool) error {
	sessionName := sessionPrefix + name

	// Validate dependencies before proceeding
	validation := deps.Validate()
	if !validation.AllRequiredMet {
		fmt.Print(deps.FormatValidationResult(validation))
		return fmt.Errorf("cannot create workspace: missing required dependencies")
	}
	if len(validation.MissingOptional) > 0 {
		fmt.Print(deps.FormatValidationResult(validation))
		fmt.Println("Continuing with limited functionality...")
		fmt.Println()
	}

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

	var workdir string
	var isMainWorkspace bool
	st := stackit.NewClient()

	if useMain {
		// Create workspace using main worktree
		repoRoot, err := git.GetRepoRoot()
		if err != nil {
			return fmt.Errorf("failed to get repository root: %w", err)
		}

		// Check if a main workspace already exists for this repo
		globalState, err := state.Load()
		if err != nil {
			return fmt.Errorf("failed to load global state: %w", err)
		}

		if existing, exists := globalState.GetMainWorkspace(repoRoot); exists {
			return fmt.Errorf("main workspace %q already exists for this repository; remove it first with 'planq remove %s'", existing.Name, existing.Name)
		}

		workdir = repoRoot
		isMainWorkspace = true

		fmt.Printf("  Using main worktree at: %s\n", workdir)

		// Record in global state
		globalState.SetMainWorkspace(repoRoot, name)
		if err := globalState.Save(); err != nil {
			return fmt.Errorf("failed to save global state: %w", err)
		}
	} else {
		// Create worktree via stackit
		fmt.Printf("  Creating worktree via stackit...\n")
		if err := st.WorktreeCreate(name, scope); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}

		// Get worktree path
		var err error
		workdir, err = st.WorktreeOpen(name)
		if err != nil {
			return fmt.Errorf("failed to get worktree path: %w", err)
		}
		fmt.Printf("  Worktree created at: %s\n", workdir)
	}

	// Create workspace and initialize .planq directory
	ws := &workspace.Workspace{
		Name:         name,
		WorktreePath: workdir,
	}

	fmt.Printf("  Initializing .planq directory...\n")
	if err := ws.InitPlanqDir(); err != nil {
		// Cleanup on failure
		if !isMainWorkspace {
			_ = st.WorktreeRemove(name)
		} else {
			// Remove state entry for main workspace
			if globalState, err := state.Load(); err == nil {
				globalState.RemoveMainWorkspace(workdir)
				_ = globalState.Save()
			}
		}
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

	_, err = tm.CreateSession(sessionName, workdir, layout)
	if err != nil {
		// Cleanup on failure
		if !isMainWorkspace {
			_ = st.WorktreeRemove(name)
		} else {
			// Remove state entry for main workspace
			if globalState, err := state.Load(); err == nil {
				globalState.RemoveMainWorkspace(workdir)
				_ = globalState.Save()
			}
		}
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Set PLANQ_WORKSPACE environment variable in the session
	if err := tm.SetEnvironment(sessionName, "PLANQ_WORKSPACE", name); err != nil {
		fmt.Printf("  Warning: failed to set PLANQ_WORKSPACE: %v\n", err)
	}

	// Set PLANQ_WORKTREE_PATH environment variable in the session
	if err := tm.SetEnvironment(sessionName, "PLANQ_WORKTREE_PATH", workdir); err != nil {
		fmt.Printf("  Warning: failed to set PLANQ_WORKTREE_PATH: %v\n", err)
	}

	// Bind mode toggle keybinding (Ctrl-B m)
	if err := tm.BindModeToggle(sessionName, name, workdir); err != nil {
		fmt.Printf("  Warning: failed to bind mode toggle key: %v\n", err)
	}

	// Bind workspace navigation keybindings (Ctrl-B w, n, p)
	if err := tm.BindWorkspaceNavigation(sessionName); err != nil {
		fmt.Printf("  Warning: failed to bind workspace navigation keys: %v\n", err)
	}

	// Configure status bar with initial mode (plan)
	if err := tm.ConfigureStatusBar(sessionName, name, "plan"); err != nil {
		fmt.Printf("  Warning: failed to configure status bar: %v\n", err)
	}

	// Configure pane borders with titles
	if err := tm.ConfigurePaneBorders(sessionName); err != nil {
		fmt.Printf("  Warning: failed to configure pane borders: %v\n", err)
	}

	// Set pane titles for plan mode layout
	paneTitles := []string{"Agent", "Plan", "Terminal"}
	for i, title := range paneTitles {
		if err := tm.SetPaneTitle(sessionName, i, title); err != nil {
			fmt.Printf("  Warning: failed to set pane %d title: %v\n", i, err)
		}
	}

	if detach {
		fmt.Println()
		fmt.Printf("To open: planq open %s\n", name)
		return nil
	}

	// Attach to the session
	return tm.AttachSession(sessionName)
}
