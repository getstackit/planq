package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"planq.dev/planq/internal/tmux"
	"planq.dev/planq/internal/workspace"
)

var modeWorkspace string

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

// loadWorkspace loads a workspace by name.
func loadWorkspace(name string) (*workspace.Workspace, string, error) {
	// The workspace path is the current working directory when running inside a planq session
	workdir, err := os.Getwd()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get working directory: %w", err)
	}

	ws := &workspace.Workspace{
		Name:         name,
		WorktreePath: workdir,
	}

	return ws, workdir, nil
}

// showMode displays the current workspace mode.
func showMode() error {
	name, err := getWorkspaceName()
	if err != nil {
		return err
	}

	ws, _, err := loadWorkspace(name)
	if err != nil {
		return err
	}

	mode, err := ws.GetMode()
	if err != nil {
		return fmt.Errorf("failed to get mode: %w", err)
	}

	fmt.Printf("Workspace %q is in %s mode\n", name, mode)
	return nil
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
		return nil
	}

	// Set the new mode
	if err := ws.SetMode(newMode); err != nil {
		return fmt.Errorf("failed to set mode: %w", err)
	}

	fmt.Printf("Switched workspace %q to %s mode\n", name, newMode)
	return reconfigureSession(name, workdir, ws, newMode)
}

// reconfigureSession reconfigures the tmux session for the new mode.
func reconfigureSession(name, workdir string, ws *workspace.Workspace, mode workspace.Mode) error {
	sessionName := sessionPrefix + name

	tm, err := tmux.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize tmux: %w", err)
	}

	// Get the appropriate layout for the mode
	agentCmd := ws.AgentCommand()
	var layout tmux.Layout

	switch mode {
	case workspace.ModeExecute:
		layout = tmux.ExecuteLayout(agentCmd)
	default:
		layout = tmux.PlanLayout(agentCmd, ws.PlanFile())
	}

	fmt.Printf("Reconfiguring tmux session for %s mode...\n", mode)
	if err := tm.ReconfigureSession(sessionName, workdir, layout); err != nil {
		return fmt.Errorf("failed to reconfigure session: %w", err)
	}

	return nil
}
