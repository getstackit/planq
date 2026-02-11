package cli

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
	"planq.dev/planq/internal/tui"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Launch split-pane TUI with two Claude instances",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTest()
	},
}

// runTest launches the dual-pane terminal TUI.
func runTest() error {
	cmd0 := exec.Command("claude")
	cmd1 := exec.Command("claude")

	if err := tui.Run(cmd0, cmd1); err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}
	return nil
}
