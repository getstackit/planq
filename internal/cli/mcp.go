package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"planq.dev/planq/internal/git"
	"planq.dev/planq/internal/queue"
)

var mcpCmd = &cobra.Command{
	Use:    "mcp",
	Short:  "Start MCP server (stdio transport)",
	Hidden: true, // Not meant for direct user invocation
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMCPServer()
	},
}

func runMCPServer() error {
	// Create a new MCP server
	s := server.NewMCPServer(
		"planq",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Define the queue tool
	queueTool := mcp.NewTool("planq_queue",
		mcp.WithDescription("Save work for later. Queue a plan, bug, or idea to revisit."),
		mcp.WithString("text",
			mcp.Required(),
			mcp.Description("The text to queue (plan, bug, idea, etc.)"),
		),
	)
	s.AddTool(queueTool, queueHandler)

	// Define the list tool
	listTool := mcp.NewTool("planq_list",
		mcp.WithDescription("List all queued items. Returns items sorted oldest first."),
	)
	s.AddTool(listTool, listHandler)

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// getProjectRoot returns the project root from env or git.
func getProjectRoot() (string, error) {
	if root := os.Getenv("PLANQ_PROJECT_ROOT"); root != "" {
		return root, nil
	}
	return git.GetRepoRoot()
}

func queueHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	text, err := request.RequireString("text")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	projectRoot, err := getProjectRoot()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to find project root: %v (set PLANQ_PROJECT_ROOT to override)", err)), nil
	}

	filePath, err := queue.Add(projectRoot, text)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to queue: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Queued to %s", filePath)), nil
}

func listHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectRoot, err := getProjectRoot()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to find project root: %v (set PLANQ_PROJECT_ROOT to override)", err)), nil
	}

	items, err := queue.List(projectRoot)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list queue: %v", err)), nil
	}

	if len(items) == 0 {
		return mcp.NewToolResultText("No items in queue"), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d item(s) in queue:\n\n", len(items)))
	for _, item := range items {
		sb.WriteString(fmt.Sprintf("## %s\n%s\n\n", item.Filename, item.Content))
	}

	return mcp.NewToolResultText(sb.String()), nil
}
