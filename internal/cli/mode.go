package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"planq.dev/planq/internal/tmux"
	"planq.dev/planq/internal/workspace"
)

var modeWorkspace string
var modeWorktree string

var modeCmd = &cobra.Command{
	Use:   "mode [plan|execute|toggle]",
	Short: "Switch or show workspace mode",
	Long: `Switch between plan and execute modes, or show the current mode.

Without arguments, shows the current mode.
With 'plan' or 'execute', switches to that mode.
With 'toggle', switches to the opposite mode.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return showMode()
		}
		return switchMode(args[0])
	},
}

func init() {
	modeCmd.Flags().StringVarP(&modeWorkspace, "workspace", "w", "", "Workspace name (default: detect from environment)")
	modeCmd.Flags().StringVar(&modeWorktree, "worktree", "", "Worktree path (default: detect from environment or cwd)")
}

// getWorkspaceName returns the workspace name from flag or environment.
func getWorkspaceName() (string, error) {
	if modeWorkspace != "" {
		return modeWorkspace, nil
	}

	// Try to get from environment variable
	if name := os.Getenv("PLANQ_WORKSPACE"); name != "" {
		return name, nil
	}

	return "", fmt.Errorf("workspace name required: use --workspace flag or set PLANQ_WORKSPACE")
}

// getTmuxSessionEnv reads an environment variable from a tmux session.
func getTmuxSessionEnv(sessionName, varName string) (string, error) {
	cmd := exec.Command("tmux", "show-environment", "-t", sessionName, varName)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Output format is "VAR=value\n"
	line := strings.TrimSpace(string(output))
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("unexpected format: %s", line)
	}
	return parts[1], nil
}

// getWorktreePath returns the worktree path from flag, tmux session env, process env, or cwd.
func getWorktreePath(workspaceName string) (string, error) {
	// First, check the flag
	if modeWorktree != "" {
		return modeWorktree, nil
	}

	// Second, check tmux session environment variable
	sessionName := sessionPrefix + workspaceName
	if path, err := getTmuxSessionEnv(sessionName, "PLANQ_WORKTREE_PATH"); err == nil && path != "" {
		return path, nil
	}

	// Third, check process environment variable
	if path := os.Getenv("PLANQ_WORKTREE_PATH"); path != "" {
		return path, nil
	}

	// Finally, fall back to current working directory
	return os.Getwd()
}

// loadWorkspace loads a workspace by name.
func loadWorkspace(name string) (*workspace.Workspace, string, error) {
	workdir, err := getWorktreePath(name)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get worktree path: %w", err)
	}

	ws := &workspace.Workspace{
		Name:         name,
		WorktreePath: workdir,
	}

	return ws, workdir, nil
}

// showMode displays the current workspace mode and reapplies the layout.
func showMode() error {
	name, err := getWorkspaceName()
	if err != nil {
		return err
	}

	ws, workdir, err := loadWorkspace(name)
	if err != nil {
		return err
	}

	mode, err := ws.GetMode()
	if err != nil {
		return fmt.Errorf("failed to get mode: %w", err)
	}

	fmt.Printf("Workspace %q is in %s mode\n", name, mode)

	// Always reapply layout in case the view is messed up
	return reconfigureSession(name, workdir, ws, mode)
}

// switchMode switches to the specified mode or toggles.
func switchMode(target string) error {
	name, err := getWorkspaceName()
	if err != nil {
		return err
	}

	ws, workdir, err := loadWorkspace(name)
	if err != nil {
		return err
	}

	var newMode workspace.Mode

	switch target {
	case "plan":
		newMode = workspace.ModePlan
	case "execute":
		newMode = workspace.ModeExecute
	case "toggle":
		newMode, err = ws.ToggleMode()
		if err != nil {
			return fmt.Errorf("failed to toggle mode: %w", err)
		}
		fmt.Printf("Switched workspace %q to %s mode\n", name, newMode)
		return reconfigureSession(name, workdir, ws, newMode)
	default:
		return fmt.Errorf("invalid mode %q: use 'plan', 'execute', or 'toggle'", target)
	}

	// Check if already in target mode
	currentMode, err := ws.GetMode()
	if err != nil {
		return fmt.Errorf("failed to get current mode: %w", err)
	}

	if currentMode == newMode {
		fmt.Printf("Workspace %q is already in %s mode\n", name, newMode)
	} else {
		// Set the new mode
		if err := ws.SetMode(newMode); err != nil {
			return fmt.Errorf("failed to set mode: %w", err)
		}
		fmt.Printf("Switched workspace %q to %s mode\n", name, newMode)
	}

	// Always reapply layout in case the view is messed up
	return reconfigureSession(name, workdir, ws, newMode)
}

// reconfigureSession reconfigures the tmux session for the new mode.
func reconfigureSession(name, workdir string, ws *workspace.Workspace, mode workspace.Mode) error {
	sessionName := sessionPrefix + name

	tm, err := tmux.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize tmux: %w", err)
	}

	// Ensure the plan file exists before glow tries to display it
	planFile := ws.PlanFile()
	if _, err := os.Stat(planFile); os.IsNotExist(err) {
		// Create the planq directory if needed
		if err := os.MkdirAll(ws.PlanqDir(), 0755); err != nil {
			return fmt.Errorf("failed to create planq directory: %w", err)
		}
		// Create empty plan file
		if err := os.WriteFile(planFile, []byte{}, 0644); err != nil {
			return fmt.Errorf("failed to create plan file: %w", err)
		}
	}

	// Get the appropriate layout for the mode
	agentCmd := ws.AgentCommand()
	var layout tmux.Layout

	switch mode {
	case workspace.ModeExecute:
		layout = tmux.ExecuteLayout(agentCmd)
	default:
		layout = tmux.PlanLayout(agentCmd, planFile)
	}

	changed, err := tm.ReconfigureSession(sessionName, workdir, layout)
	if err != nil {
		return fmt.Errorf("failed to reconfigure session: %w", err)
	}

	// Update status bar with current mode
	if err := tm.ConfigureStatusBar(sessionName, name, string(mode)); err != nil {
		// Non-fatal, just warn
		fmt.Printf("Warning: could not update status bar: %v\n", err)
	}

	// Set pane titles based on mode
	var paneTitles []string
	if mode == workspace.ModeExecute {
		paneTitles = []string{"Agent"}
	} else {
		paneTitles = []string{"Agent", "Plan", "Terminal"}
	}
	for i, title := range paneTitles {
		if err := tm.SetPaneTitle(sessionName, i, title); err != nil {
			// Non-fatal, pane might not exist yet
			break
		}
	}

	if changed {
		fmt.Printf("Reconfigured tmux session for %s mode\n", mode)
	} else {
		fmt.Printf("Layout already matches %s mode, no changes needed\n", mode)
	}

	return nil
}
