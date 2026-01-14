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
// 2-pane layout: agent (left, 50%) + git diff viewer (right, 50%)
func ExecuteLayout(agentCmd string) Layout {
	return Layout{
		Name:        "execute",
		Description: "Execution mode: agent + git diff",
		Panes: []PaneSpec{
			{Name: "agent", Size: 50, Command: agentCmd},
			{Name: "diff", Size: 50, Command: "while true; do clear; git diff --color=always | delta --paging=never; sleep 2; done"},
		},
	}
}
