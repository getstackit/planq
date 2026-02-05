package cli

import (
	"context"
	"fmt"

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

	// Add tool handler
	s.AddTool(queueTool, queueHandler)

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func queueHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	text, err := request.RequireString("text")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Get project root
	projectRoot, err := git.GetRepoRoot()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to find project root: %v", err)), nil
	}

	// Add to queue
	filePath, err := queue.Add(projectRoot, text)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to queue: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Queued to %s", filePath)), nil
}
