package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"planq.dev/planq/internal/tmux"
)

var openCmd = &cobra.Command{
	Use:   "open <name>",
	Short: "Open an existing workspace",
	Long:  `Open an existing workspace by attaching to its tmux session.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return openWorkspace(args[0])
	},
}

// openWorkspace opens an existing workspace's tmux session.
func openWorkspace(name string) error {
	sessionName := sessionPrefix + name

	tm, err := tmux.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize tmux: %w", err)
	}

	exists, err := tm.SessionExists(sessionName)
	if err != nil {
		return fmt.Errorf("failed to check session: %w", err)
	}
	if !exists {
		return fmt.Errorf("workspace %q does not exist", name)
	}

	fmt.Printf("Opening workspace %q...\n", name)

	// Use exec to replace current process with tmux attach
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux not found: %w", err)
	}

	return execCommand(tmuxPath, "attach", "-t", sessionName)
}

// execCommand replaces the current process with the given command.
// This is used for tmux attach so the user gets a proper terminal experience.
func execCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
