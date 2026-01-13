// Package main provides the exploration entry point for planq.
// This is a Phase 0 exploration script to test tmux + stackit integration.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"planq.dev/planq/internal/stackit"
	"planq.dev/planq/internal/tmux"
)

const sessionPrefix = "planq-"

func main() {
	// Flags for exploration
	name := flag.String("name", "", "Workspace name")
	action := flag.String("action", "demo", "Action: demo, create, list, attach, remove")
	scope := flag.String("scope", "", "Scope for worktree (optional)")
	agentCmd := flag.String("agent-cmd", "", "Command to run in agent pane (default: claude)")
	flag.Parse()

	switch *action {
	case "demo":
		runDemo()
	case "create":
		if *name == "" {
			fmt.Fprintln(os.Stderr, "Error: --name is required for create")
			os.Exit(1)
		}
		if err := createWorkspace(*name, *scope, *agentCmd); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "list":
		if err := listWorkspaces(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "attach":
		if *name == "" {
			fmt.Fprintln(os.Stderr, "Error: --name is required for attach")
			os.Exit(1)
		}
		if err := attachWorkspace(*name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "remove":
		if *name == "" {
			fmt.Fprintln(os.Stderr, "Error: --name is required for remove")
			os.Exit(1)
		}
		if err := removeWorkspace(*name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown action: %s\n", *action)
		os.Exit(1)
	}
}

// runDemo demonstrates the tmux + stackit integration without creating worktrees.
func runDemo() {
	fmt.Println("=== Planq Exploration Demo ===")
	fmt.Println()

	// Test tmux manager
	fmt.Println("1. Testing tmux manager...")
	tm, err := tmux.NewManager()
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
		fmt.Println("   Make sure tmux is installed and running")
	} else {
		fmt.Println("   tmux manager initialized successfully")

		// List existing planq sessions
		sessions, err := tm.ListSessions(sessionPrefix)
		if err != nil {
			fmt.Printf("   Warning: Could not list sessions: %v\n", err)
		} else {
			fmt.Printf("   Found %d existing planq session(s)\n", len(sessions))
			for _, s := range sessions {
				fmt.Printf("   - %s\n", s.Name)
			}
		}
	}
	fmt.Println()

	// Test stackit client
	fmt.Println("2. Testing stackit client...")
	st := stackit.NewClient()
	worktrees, err := st.WorktreeList()
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
		fmt.Println("   Make sure stackit is installed and you're in a git repo")
	} else {
		fmt.Printf("   Found %d stackit worktree(s)\n", len(worktrees))
		for _, wt := range worktrees {
			fmt.Printf("   - %s: %s\n", wt.Name, wt.Path)
		}
	}
	fmt.Println()

	// Show usage
	fmt.Println("=== Usage ===")
	fmt.Println()
	fmt.Println("Create a workspace:")
	fmt.Println("  go run ./cmd/planq --action=create --name=my-feature")
	fmt.Println()
	fmt.Println("Create with scope:")
	fmt.Println("  go run ./cmd/planq --action=create --name=my-feature --scope=auth")
	fmt.Println()
	fmt.Println("List workspaces:")
	fmt.Println("  go run ./cmd/planq --action=list")
	fmt.Println()
	fmt.Println("Attach to workspace:")
	fmt.Println("  go run ./cmd/planq --action=attach --name=my-feature")
	fmt.Println()
	fmt.Println("Remove workspace:")
	fmt.Println("  go run ./cmd/planq --action=remove --name=my-feature")
}

// createWorkspace creates a new workspace with worktree + tmux session.
func createWorkspace(name, scope, agentCmd string) error {
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
		return fmt.Errorf("session %q already exists, use --action=attach to attach", sessionName)
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

	// Determine agent command
	if agentCmd == "" {
		agentCmd = "claude" // Default to claude
	}

	// Create tmux session with layout
	fmt.Printf("  Creating tmux session %q...\n", sessionName)
	layout := tmux.Layout{
		Name: "agent-artifact",
		Panes: []tmux.PaneSpec{
			{Name: "agent", Size: 70, Command: agentCmd},
			{Name: "artifacts", Size: 30, Command: fmt.Sprintf("watch -n 2 'ls -la %s'", workdir)},
		},
	}

	session, err := tm.CreateSession(sessionName, workdir, layout)
	if err != nil {
		// Cleanup worktree if session creation fails
		_ = st.WorktreeRemove(name)
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	fmt.Printf("  Session created: %s\n", session.Name)
	fmt.Println()
	fmt.Printf("Workspace %q created successfully!\n", name)
	fmt.Println()
	fmt.Printf("To attach: tmux attach -t %s\n", sessionName)
	fmt.Printf("Or run: go run ./cmd/planq --action=attach --name=%s\n", name)

	return nil
}

// listWorkspaces lists all planq workspaces.
func listWorkspaces() error {
	fmt.Println("=== Planq Workspaces ===")
	fmt.Println()

	// List tmux sessions
	tm, err := tmux.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize tmux: %w", err)
	}

	sessions, err := tm.ListSessions(sessionPrefix)
	if err != nil {
		fmt.Println("No tmux sessions found (tmux server may not be running)")
	} else if len(sessions) == 0 {
		fmt.Println("No planq sessions found")
	} else {
		fmt.Printf("%-20s %-10s\n", "NAME", "WINDOWS")
		fmt.Printf("%-20s %-10s\n", "----", "-------")
		for _, s := range sessions {
			// Strip prefix for display
			displayName := s.Name
			if len(s.Name) > len(sessionPrefix) {
				displayName = s.Name[len(sessionPrefix):]
			}
			fmt.Printf("%-20s %-10d\n", displayName, s.Windows)
		}
	}
	fmt.Println()

	// List stackit worktrees
	fmt.Println("=== Stackit Worktrees ===")
	fmt.Println()
	st := stackit.NewClient()
	worktrees, err := st.WorktreeList()
	if err != nil {
		fmt.Printf("Could not list worktrees: %v\n", err)
	} else if len(worktrees) == 0 {
		fmt.Println("No worktrees found")
	} else {
		fmt.Printf("%-20s %-50s %-30s\n", "NAME", "PATH", "BRANCH")
		fmt.Printf("%-20s %-50s %-30s\n", "----", "----", "------")
		for _, wt := range worktrees {
			fmt.Printf("%-20s %-50s %-30s\n", wt.Name, wt.Path, wt.Branch)
		}
	}

	return nil
}

// attachWorkspace attaches to an existing workspace's tmux session.
func attachWorkspace(name string) error {
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
		return fmt.Errorf("session %q does not exist", sessionName)
	}

	fmt.Printf("Attaching to session %q...\n", sessionName)

	// Use exec to replace current process with tmux attach
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux not found: %w", err)
	}

	return execCommand(tmuxPath, "attach", "-t", sessionName)
}

// removeWorkspace removes a workspace (tmux session + worktree).
func removeWorkspace(name string) error {
	sessionName := sessionPrefix + name

	fmt.Printf("Removing workspace %q...\n", name)

	// Kill tmux session
	tm, err := tmux.NewManager()
	if err != nil {
		fmt.Printf("  Warning: Could not initialize tmux: %v\n", err)
	} else {
		exists, _ := tm.SessionExists(sessionName)
		if exists {
			fmt.Printf("  Killing tmux session %q...\n", sessionName)
			if err := tm.KillSession(sessionName); err != nil {
				fmt.Printf("  Warning: Could not kill session: %v\n", err)
			} else {
				fmt.Println("  Session killed")
			}
		} else {
			fmt.Println("  No tmux session found")
		}
	}

	// Remove worktree
	fmt.Printf("  Removing worktree %q...\n", name)
	st := stackit.NewClient()
	if err := st.WorktreeRemove(name); err != nil {
		// Try force remove
		if err := st.WorktreeRemoveForce(name); err != nil {
			fmt.Printf("  Warning: Could not remove worktree: %v\n", err)
		} else {
			fmt.Println("  Worktree removed (forced)")
		}
	} else {
		fmt.Println("  Worktree removed")
	}

	fmt.Printf("Workspace %q removed\n", name)
	return nil
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
