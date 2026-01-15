// Package cli provides the Cobra command-line interface for planq.
package cli

import (
	"os"

	"github.com/spf13/cobra"
)

// sessionPrefix is the prefix for all planq tmux sessions.
const sessionPrefix = "planq-"

// rootCmd is the base command for planq.
var rootCmd = &cobra.Command{
	Use:   "planq",
	Short: "Orchestrate parallel AI agent workspaces",
	Long:  `Planq manages parallel AI agent workspaces using git worktrees and tmux.`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(modeCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(helpCmd)
	rootCmd.AddCommand(notifyCmd)
}
