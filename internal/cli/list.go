package cli

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"planq.dev/planq/internal/stackit"
	"planq.dev/planq/internal/state"
	"planq.dev/planq/internal/tmux"
	"planq.dev/planq/internal/workspace"
)

// Color palette (Catppuccin Mocha inspired)
var (
	colorActive   = lipgloss.Color("#a6e3a1") // green
	colorInactive = lipgloss.Color("#6c7086") // gray
	colorOrphaned = lipgloss.Color("#f38ba8") // red
	colorMain     = lipgloss.Color("#89b4fa") // blue
	colorReview   = lipgloss.Color("#f9e2af") // yellow
	colorText     = lipgloss.Color("#cdd6f4") // light text
	colorMuted    = lipgloss.Color("#6c7086") // muted text
	colorBorder   = lipgloss.Color("#45475a") // border
)

// Styles for workspace cards
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText)

	cardStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1).
			MarginBottom(1)

	cardActiveStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorActive).
			Padding(0, 1).
			MarginBottom(1)

	cardOrphanedStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(colorOrphaned).
				Padding(0, 1).
				MarginBottom(1)

	labelStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(10)

	valueStyle = lipgloss.NewStyle().
			Foreground(colorText)

	statusActiveStyle = lipgloss.NewStyle().
				Foreground(colorActive).
				Bold(true)

	statusInactiveStyle = lipgloss.NewStyle().
				Foreground(colorInactive)

	statusOrphanedStyle = lipgloss.NewStyle().
				Foreground(colorOrphaned).
				Bold(true)

	mainBadgeStyle = lipgloss.NewStyle().
			Foreground(colorMain).
			Bold(true)

	reviewBadgeStyle = lipgloss.NewStyle().
				Foreground(colorReview).
				Bold(true)

	summaryStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginTop(1)

	emptyStyle = lipgloss.NewStyle().
			Foreground(colorMuted)
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all workspaces",
	Long:  `List all planq workspaces (tmux sessions and git worktrees).`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return listWorkspaces()
	},
}

// workspaceEntry represents a combined workspace entry for display.
type workspaceEntry struct {
	Name        string
	Branch      string
	Dir         string
	Status      string
	Mode        string
	IsMain      bool
	NeedsReview bool
}

// listWorkspaces lists all planq workspaces with styled cards.
func listWorkspaces() error {
	// Collect worktrees
	worktreeMap := make(map[string]stackit.WorktreeEntry)
	st := stackit.NewClient()
	worktrees, err := st.WorktreeList()
	if err == nil {
		for _, wt := range worktrees {
			worktreeMap[wt.Name] = wt
		}
	}

	// Collect tmux sessions
	sessionMap := make(map[string]bool)
	tm, err := tmux.NewManager()
	if err == nil {
		sessions, err := tm.ListSessions(sessionPrefix)
		if err == nil {
			for _, s := range sessions {
				// Strip prefix for the name
				name := s.Name
				if len(s.Name) > len(sessionPrefix) {
					name = s.Name[len(sessionPrefix):]
				}
				sessionMap[name] = true
			}
		}
	}

	// Load main workspace info
	mainWorkspaceNames := make(map[string]bool)
	if globalState, err := state.Load(); err == nil {
		mainWorkspaceNames = globalState.GetMainWorkspaceNames()
	}

	// Build unified list
	entries := buildWorkspaceEntries(worktreeMap, sessionMap, mainWorkspaceNames)

	if len(entries) == 0 {
		fmt.Println(emptyStyle.Render("No workspaces found"))
		fmt.Println()
		fmt.Println(emptyStyle.Render("Create one with: planq create <name>"))
		return nil
	}

	// Render header
	header := titleStyle.Render("planq workspaces")
	fmt.Println(header)
	fmt.Println()

	// Render each workspace as a card
	var activeCount, inactiveCount, orphanedCount, reviewCount int
	for _, entry := range entries {
		card := renderWorkspaceCard(entry)
		fmt.Println(card)

		switch entry.Status {
		case "active":
			activeCount++
		case "inactive":
			inactiveCount++
		case "orphaned":
			orphanedCount++
		}
		if entry.NeedsReview {
			reviewCount++
		}
	}

	// Render summary
	summary := renderSummary(len(entries), activeCount, inactiveCount, orphanedCount, reviewCount)
	fmt.Println(summary)

	return nil
}

// renderWorkspaceCard creates a styled card for a workspace entry.
func renderWorkspaceCard(e workspaceEntry) string {
	// Status indicator and styles
	var statusIcon, statusText string
	var nameStyle lipgloss.Style
	var baseCardStyle lipgloss.Style

	switch e.Status {
	case "active":
		statusIcon = "●"
		statusText = statusActiveStyle.Render("active")
		nameStyle = lipgloss.NewStyle().Bold(true).Foreground(colorActive)
		baseCardStyle = cardActiveStyle
	case "orphaned":
		statusIcon = "⚠"
		statusText = statusOrphanedStyle.Render("orphaned")
		nameStyle = lipgloss.NewStyle().Bold(true).Foreground(colorOrphaned)
		baseCardStyle = cardOrphanedStyle
	default:
		statusIcon = "○"
		statusText = statusInactiveStyle.Render("inactive")
		nameStyle = lipgloss.NewStyle().Bold(true).Foreground(colorText)
		baseCardStyle = cardStyle
	}

	// Header line with name and optional badges
	headerLine := fmt.Sprintf("%s  %s", statusIcon, nameStyle.Render(e.Name))
	if e.IsMain {
		headerLine += "  " + mainBadgeStyle.Render("[main]")
	}
	if e.NeedsReview {
		headerLine += "  " + reviewBadgeStyle.Render("[review]")
	}

	// Detail lines
	lines := []string{
		headerLine,
		fmt.Sprintf("    %s %s", labelStyle.Render("Branch:"), valueStyle.Render(e.Branch)),
		fmt.Sprintf("    %s %s", labelStyle.Render("Dir:"), valueStyle.Render(e.Dir)),
		fmt.Sprintf("    %s %s", labelStyle.Render("Mode:"), valueStyle.Render(e.Mode)),
		fmt.Sprintf("    %s %s", labelStyle.Render("Status:"), statusText),
	}

	content := strings.Join(lines, "\n")
	return baseCardStyle.Render(content)
}

// renderSummary creates the summary line.
func renderSummary(total, active, inactive, orphaned, review int) string {
	word := "workspace"
	if total != 1 {
		word = "workspaces"
	}

	var details []string
	if active > 0 {
		details = append(details, statusActiveStyle.Render(fmt.Sprintf("%d active", active)))
	}
	if inactive > 0 {
		details = append(details, statusInactiveStyle.Render(fmt.Sprintf("%d inactive", inactive)))
	}
	if orphaned > 0 {
		details = append(details, statusOrphanedStyle.Render(fmt.Sprintf("%d orphaned", orphaned)))
	}
	if review > 0 {
		details = append(details, reviewBadgeStyle.Render(fmt.Sprintf("%d needs review", review)))
	}

	summary := fmt.Sprintf("%d %s", total, word)
	if len(details) > 0 {
		summary += " (" + strings.Join(details, ", ") + ")"
	}

	return summaryStyle.Render(summary)
}

// buildWorkspaceEntries combines worktrees and sessions into workspace entries.
func buildWorkspaceEntries(worktrees map[string]stackit.WorktreeEntry, sessions map[string]bool, mainWorkspaces map[string]bool) []workspaceEntry {
	seen := make(map[string]bool)
	var entries []workspaceEntry

	// Add all worktrees
	for name, wt := range worktrees {
		status := "inactive"
		if sessions[name] {
			status = "active"
		}

		// Get mode and review state from workspace
		mode := "-"
		needsReview := false
		ws := &workspace.Workspace{Name: name, WorktreePath: wt.Path}
		if m, err := ws.GetMode(); err == nil {
			mode = string(m)
		}
		if rs, err := ws.GetReviewState(); err == nil {
			needsReview = rs.NeedsReview
		}

		entries = append(entries, workspaceEntry{
			Name:        name,
			Branch:      wt.Branch,
			Dir:         filepath.Base(wt.Path),
			Status:      status,
			Mode:        mode,
			IsMain:      mainWorkspaces[name],
			NeedsReview: needsReview,
		})
		seen[name] = true
	}

	// Add orphaned sessions (sessions without worktrees)
	for name := range sessions {
		if !seen[name] {
			entries = append(entries, workspaceEntry{
				Name:   name,
				Branch: "-",
				Dir:    "-",
				Status: "orphaned",
				Mode:   "-",
				IsMain: mainWorkspaces[name],
			})
		}
	}

	// Sort by name
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	return entries
}
