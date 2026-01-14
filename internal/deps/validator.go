// Package deps provides dependency validation for planq.
package deps

import (
	"fmt"
	"os/exec"
	"strings"
)

// Dependency represents a required or optional external tool.
type Dependency struct {
	Name        string
	Required    bool
	Description string
	InstallHint string
}

// DefaultDependencies returns the list of dependencies to check.
func DefaultDependencies() []Dependency {
	return []Dependency{
		{
			Name:        "tmux",
			Required:    true,
			Description: "terminal multiplexer for workspace sessions",
			InstallHint: "brew install tmux (macOS) or apt install tmux (Linux)",
		},
		{
			Name:        "stackit",
			Required:    true,
			Description: "git worktree management",
			InstallHint: "see https://github.com/getstackit/stackit",
		},
		{
			Name:        "claude",
			Required:    true,
			Description: "Claude AI assistant CLI",
			InstallHint: "npm install -g @anthropic-ai/claude-code",
		},
		{
			Name:        "glow",
			Required:    false,
			Description: "markdown renderer for plan viewer",
			InstallHint: "brew install glow (macOS) or go install github.com/charmbracelet/glow@latest",
		},
	}
}

// CheckResult represents the result of checking a dependency.
type CheckResult struct {
	Dependency Dependency
	Available  bool
	Version    string
	Error      error
}

// Check checks if a single dependency is available.
func Check(dep Dependency) CheckResult {
	result := CheckResult{Dependency: dep}

	// Check if command exists using which
	cmd := exec.Command("which", dep.Name)
	output, err := cmd.Output()
	if err != nil {
		result.Available = false
		result.Error = fmt.Errorf("not found in PATH")
		return result
	}

	result.Available = true

	// Try to get version (best effort)
	path := strings.TrimSpace(string(output))
	if path != "" {
		versionCmd := exec.Command(dep.Name, "--version")
		versionOutput, err := versionCmd.Output()
		if err == nil {
			// Take first line of version output
			lines := strings.Split(string(versionOutput), "\n")
			if len(lines) > 0 {
				result.Version = strings.TrimSpace(lines[0])
			}
		}
	}

	return result
}

// CheckAll checks all default dependencies and returns results.
func CheckAll() []CheckResult {
	deps := DefaultDependencies()
	results := make([]CheckResult, len(deps))

	for i, dep := range deps {
		results[i] = Check(dep)
	}

	return results
}

// ValidationResult contains the overall validation result.
type ValidationResult struct {
	Results         []CheckResult
	MissingRequired []CheckResult
	MissingOptional []CheckResult
	AllRequiredMet  bool
}

// Validate checks all dependencies and returns a validation result.
func Validate() ValidationResult {
	results := CheckAll()
	validation := ValidationResult{
		Results:        results,
		AllRequiredMet: true,
	}

	for _, r := range results {
		if !r.Available {
			if r.Dependency.Required {
				validation.MissingRequired = append(validation.MissingRequired, r)
				validation.AllRequiredMet = false
			} else {
				validation.MissingOptional = append(validation.MissingOptional, r)
			}
		}
	}

	return validation
}

// FormatValidationResult formats the validation result for display.
func FormatValidationResult(v ValidationResult) string {
	var sb strings.Builder

	// Show missing required dependencies
	if len(v.MissingRequired) > 0 {
		sb.WriteString("Missing required dependencies:\n")
		for _, r := range v.MissingRequired {
			fmt.Fprintf(&sb, "  ✗ %s - %s\n", r.Dependency.Name, r.Dependency.Description)
			fmt.Fprintf(&sb, "    Install: %s\n", r.Dependency.InstallHint)
		}
		sb.WriteString("\n")
	}

	// Show missing optional dependencies
	if len(v.MissingOptional) > 0 {
		sb.WriteString("Missing optional dependencies:\n")
		for _, r := range v.MissingOptional {
			fmt.Fprintf(&sb, "  ⚠ %s - %s\n", r.Dependency.Name, r.Dependency.Description)
			fmt.Fprintf(&sb, "    Install: %s\n", r.Dependency.InstallHint)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
