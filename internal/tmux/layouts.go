package tmux

import "fmt"

// PlanLayout returns the layout for plan mode.
// 3-pane layout: agent (left), plan viewer (top-right), terminal (bottom-right)
func PlanLayout(agentCmd, planFile string) Layout {
	return Layout{
		Name:        "plan",
		Description: "Planning mode: agent + plan viewer + terminal",
		Panes: []PaneSpec{
			{Name: "agent", Size: 60, Command: agentCmd},
			{Name: "plan", Size: 20, Command: fmt.Sprintf("glow %s --tui", planFile)},
			{Name: "terminal", Size: 20, Command: ""},
		},
	}
}

// ExecuteLayout returns the layout for execute mode.
// Single full-width pane for focused implementation.
func ExecuteLayout(agentCmd string) Layout {
	return Layout{
		Name:        "execute",
		Description: "Execution mode: full-width agent",
		Panes: []PaneSpec{
			{Name: "agent", Size: 100, Command: agentCmd},
		},
	}
}
