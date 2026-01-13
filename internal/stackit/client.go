// Package stackit provides a client for interacting with the stackit CLI.
package stackit

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Client wraps the stackit CLI for worktree operations.
type Client struct {
	binPath string
	cwd     string
}

// NewClient creates a new stackit client with default settings.
func NewClient() *Client {
	return &Client{
		binPath: "stackit",
	}
}

// NewClientWithPath creates a client with a custom stackit binary path.
func NewClientWithPath(binPath string) *Client {
	return &Client{
		binPath: binPath,
	}
}

// SetWorkingDirectory sets the working directory for stackit commands.
func (c *Client) SetWorkingDirectory(cwd string) {
	c.cwd = cwd
}

// WorktreeEntry represents a worktree from stackit worktree list.
type WorktreeEntry struct {
	Name   string
	Path   string
	Branch string
}

// WorktreeCreate creates a new worktree with the given name and optional scope.
func (c *Client) WorktreeCreate(name string, scope string) error {
	args := []string{"worktree", "create", name}
	if scope != "" {
		args = append(args, "--scope", scope)
	}

	_, err := c.run(args...)
	return err
}

// WorktreeOpen returns the path to a worktree by name.
func (c *Client) WorktreeOpen(name string) (string, error) {
	output, err := c.run("worktree", "open", name)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// WorktreeList returns all managed worktrees.
func (c *Client) WorktreeList() ([]WorktreeEntry, error) {
	output, err := c.run("worktree", "list")
	if err != nil {
		return nil, err
	}

	// Parse output - format is typically:
	// NAME     PATH                           BRANCH
	// feature  /path/to/worktree              branch-name
	var entries []WorktreeEntry
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		// Skip header line and empty lines
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			entry := WorktreeEntry{
				Name: fields[0],
				Path: fields[1],
			}
			if len(fields) >= 3 {
				entry.Branch = fields[2]
			}
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

// WorktreeRemove removes a worktree by name.
func (c *Client) WorktreeRemove(name string) error {
	_, err := c.run("worktree", "remove", name)
	return err
}

// WorktreeRemoveForce forcefully removes a worktree.
func (c *Client) WorktreeRemoveForce(name string) error {
	_, err := c.run("worktree", "remove", "--force", name)
	return err
}

// run executes a stackit command and returns the output.
func (c *Client) run(args ...string) (string, error) {
	cmd := exec.Command(c.binPath, args...)
	if c.cwd != "" {
		cmd.Dir = c.cwd
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("stackit %s failed: %w\nstderr: %s", args[0], err, stderr.String())
	}

	return stdout.String(), nil
}
