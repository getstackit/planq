package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"planq.dev/planq/internal/stackit"
	"planq.dev/planq/internal/state"
	"planq.dev/planq/internal/tmux"
	"planq.dev/planq/internal/workspace"
)

var removeAll bool

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a workspace",
	Long:  `Remove a workspace by killing its tmux session and removing the git worktree.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if removeAll {
			if len(args) > 0 {
				return fmt.Errorf("cannot specify workspace name with --all")
			}
			return nil
		}
		if len(args) != 1 {
			return fmt.Errorf("requires exactly 1 argument (workspace name) or --all flag")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if removeAll {
			return removeAllWorkspaces()
		}
		return removeWorkspace(args[0])
	},
}

func init() {
	removeCmd.Flags().BoolVarP(&removeAll, "all", "a", false, "Remove all workspaces")
}

// removeAllWorkspaces removes all planq workspaces.
func removeAllWorkspaces() error {
	tm, err := tmux.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize tmux: %w", err)
	}

	sessions, err := tm.ListSessions(sessionPrefix)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No workspaces found")
		return nil
	}

	fmt.Printf("Removing %d workspace(s)...\n", len(sessions))
	for _, session := range sessions {
		// Extract workspace name from session name (remove prefix)
		name := strings.TrimPrefix(session.Name, sessionPrefix)
		if err := removeWorkspace(name); err != nil {
			fmt.Printf("  Warning: Failed to remove %q: %v\n", name, err)
		}
	}

	fmt.Println("Done")
	return nil
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

	// Check if this is a main workspace
	globalState, err := state.Load()
	if err != nil {
		fmt.Printf("  Warning: Could not load global state: %v\n", err)
	} else if repoPath, isMain := globalState.FindMainWorkspaceByName(name); isMain {
		// This is a main workspace - clean up .agent and remove state entry, but preserve worktree
		ws := &workspace.Workspace{Name: name, WorktreePath: repoPath}
		if err := ws.CleanupAgentDir(); err != nil {
			fmt.Printf("  Warning: Could not clean up .agent directory: %v\n", err)
		}
		fmt.Println("  Removing main workspace registration...")
		globalState.RemoveMainWorkspace(repoPath)
		if err := globalState.Save(); err != nil {
			fmt.Printf("  Warning: Could not save global state: %v\n", err)
		}
		fmt.Printf("Workspace %q removed (main worktree preserved)\n", name)
		return nil
	}

	// Not a main workspace - remove worktree via stackit
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
