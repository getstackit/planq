package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var helpCmd = &cobra.Command{
	Use:   "help",
	Short: "Help about planq commands",
	Long:  `Display help information about planq commands and usage.`,
}

var helpTmuxCmd = &cobra.Command{
	Use:   "tmux",
	Short: "Quick reference for tmux keybindings in planq",
	Long:  `Display a quick reference guide for tmux keybindings used in planq workspaces.`,
	Run: func(cmd *cobra.Command, args []string) {
		printTmuxHelp()
	},
}

func init() {
	helpCmd.AddCommand(helpTmuxCmd)
}

func printTmuxHelp() {
	help := `
Planq tmux Quick Reference
==========================

WORKSPACE SWITCHING
  Ctrl+B w          Open workspace selector (popup with fzf)
  Ctrl+B s          Session switcher (tmux built-in tree view)
  Ctrl+B n          Switch to next workspace
  Ctrl+B p          Switch to previous workspace
  planq list        Show all workspaces

PANE NAVIGATION
  Ctrl+B ←/→/↑/↓    Move between panes
  Ctrl+B o          Cycle through panes
  Click             Select pane (mouse enabled)

MODE SWITCHING
  Ctrl+B m          Toggle plan/execute mode
  planq mode        Show current mode
  planq mode plan   Switch to plan mode
  planq mode exec   Switch to execute mode

PANE MANAGEMENT
  Ctrl+B z          Zoom current pane (toggle fullscreen)
  Ctrl+B {          Swap pane left
  Ctrl+B }          Swap pane right
  Drag border       Resize pane (mouse enabled)

SESSION
  Ctrl+B d          Detach (leaves session running)
  planq open <name> Reattach to session
  Ctrl+B ?          Show all tmux keybindings

COPY MODE (scroll/select)
  Ctrl+B [          Enter copy mode
  q                 Exit copy mode
  Arrow keys        Navigate in copy mode
  Space             Start selection
  Enter             Copy selection

SCROLLING
  Mouse wheel       Scroll in TUI apps (glow, vim, etc.)
  Ctrl+B [          Enter copy mode for terminal scroll
  Page Up/Down      Scroll in copy mode

For more: man tmux or https://tmuxcheatsheet.com
`
	fmt.Print(help)
}
