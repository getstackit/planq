package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"planq.dev/planq/internal/git"
	"planq.dev/planq/internal/queue"
)

var queueCmd = &cobra.Command{
	Use:   "queue <text>",
	Short: "Queue work for later",
	Long: `Queue a plan, bug, or idea to revisit later.

Items are saved to .planq/queue/ as timestamped markdown files.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		text := strings.Join(args, " ")
		return runQueue(text)
	},
}

func runQueue(text string) error {
	// Get project root
	projectRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	// Add to queue
	filePath, err := queue.Add(projectRoot, text)
	if err != nil {
		return err
	}

	fmt.Printf("Queued: %s\n", filePath)
	return nil
}
