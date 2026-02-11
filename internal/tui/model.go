package tui

import (
	"fmt"
	"image"
	"os/exec"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/vt"
)

// tickMsg triggers periodic checks (process exit, screen refresh).
type tickMsg time.Time

func doTick() tea.Cmd {
	return tea.Tick(33*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Run launches the dual-pane terminal TUI with the given commands.
func Run(cmd0, cmd1 *exec.Cmd) error {
	m := &model{cmd0: cmd0, cmd1: cmd1}
	p := tea.NewProgram(m)
	_, err := p.Run()
	// Ensure cleanup even if bubbletea exits without going through Update.
	m.cleanup()
	return err
}

// model is the bubbletea model for the dual-pane terminal TUI.
type model struct {
	cmd0, cmd1  *exec.Cmd
	panes       [2]*Pane
	focused     int
	metaActive  bool
	width       int
	height      int
	started     bool
	cleanupOnce sync.Once
}

// Init returns the initial command.
func (m *model) Init() tea.Cmd {
	return doTick()
}

// Update handles messages.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleResize(msg)

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tickMsg:
		if m.started && m.panes[0].Exited() && m.panes[1].Exited() {
			m.cleanup()
			return m, tea.Quit
		}
		return m, doTick()
	}

	return m, nil
}

// handleResize creates panes on first resize or resizes existing ones.
func (m *model) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	pw, ph := paneSize(m.width, m.height)
	if pw <= 0 || ph <= 0 {
		return m, nil
	}

	if !m.started {
		p0, err := NewPane(pw, ph, m.cmd0)
		if err != nil {
			return m, tea.Quit
		}
		p1, err := NewPane(pw, ph, m.cmd1)
		if err != nil {
			p0.Close()
			return m, tea.Quit
		}
		m.panes[0] = p0
		m.panes[1] = p1
		m.started = true
		return m, nil
	}

	for _, p := range m.panes {
		p.Resize(pw, ph) //nolint:errcheck
	}
	return m, nil
}

// handleKey processes key events with Ctrl+A meta prefix support.
func (m *model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if !m.started {
		return m, nil
	}

	key := tea.Key(msg)

	if m.metaActive {
		m.metaActive = false
		switch {
		case key.Code == tea.KeyTab:
			m.focused = 1 - m.focused
		case key.Code == 'q' && key.Mod == 0:
			m.cleanup()
			return m, tea.Quit
		case key.Code == 'a' && key.Mod == tea.ModCtrl:
			// Ctrl+A Ctrl+A → send literal Ctrl+A to focused pane
			m.sendKey(vt.KeyPressEvent{Code: 'a', Mod: vt.ModCtrl})
		}
		return m, nil
	}

	// Ctrl+A triggers meta mode
	if key.Code == 'a' && key.Mod == tea.ModCtrl {
		m.metaActive = true
		return m, nil
	}

	// Forward key to focused pane
	m.sendKey(vt.KeyPressEvent(msg))
	return m, nil
}

// sendKey forwards a key event to the focused pane if it's still running.
func (m *model) sendKey(key vt.KeyPressEvent) {
	p := m.panes[m.focused]
	if !p.Exited() {
		p.Emulator().SendKey(key)
	}
}

// cleanup closes both panes. Safe to call multiple times.
func (m *model) cleanup() {
	m.cleanupOnce.Do(func() {
		for _, p := range m.panes {
			if p != nil {
				p.Close()
			}
		}
	})
}

// View returns the tea.View with a composite layer that draws both panes.
func (m *model) View() tea.View {
	var v tea.View
	v.AltScreen = true

	if !m.started {
		v.SetContent("Waiting for terminal size...")
		return v
	}

	v.Content = &dualPaneLayer{
		panes:   m.panes,
		focused: m.focused,
		width:   m.width,
		height:  m.height,
	}

	// Show cursor from the focused emulator
	emu := m.panes[m.focused].Emulator()
	pos := emu.CursorPosition()
	pw, _ := paneSize(m.width, m.height)

	cursorX := pos.X + 1 // +1 for left border
	if m.focused == 1 {
		cursorX = pos.X + pw + 4 // left pane width + 2 borders + divider + right border
	}
	cursorY := pos.Y + 1 // +1 for top border
	v.Cursor = tea.NewCursor(cursorX, cursorY)

	return v
}

// dualPaneLayer implements tea.Layer to draw two terminal emulators side-by-side.
type dualPaneLayer struct {
	panes   [2]*Pane
	focused int
	width   int
	height  int
}

// Draw renders both panes with borders and a status bar into the screen buffer.
//
// Layout (column positions):
//
//	0         : left pane left border
//	1..pw     : left pane content (pw columns)
//	pw+1      : left pane right border
//	pw+2      : divider
//	pw+3      : right pane left border
//	pw+4..2pw+3 : right pane content (pw columns)
//	2pw+4     : right pane right border
//
// Total width = 2*pw + 5
func (d *dualPaneLayer) Draw(s tea.Screen, r tea.Rectangle) {
	pw, ph := paneSize(d.width, d.height)
	if pw <= 0 || ph <= 0 {
		return
	}

	leftColor := colorBlurred
	rightColor := colorBlurred
	if d.focused == 0 {
		leftColor = colorFocused
	} else {
		rightColor = colorFocused
	}

	lBorder := uv.Style{Fg: leftColor}
	rBorder := uv.Style{Fg: rightColor}
	divStyle := uv.Style{Fg: colorBlurred}
	statStyle := uv.Style{Fg: colorStatus}

	ox := r.Min.X // origin x
	oy := r.Min.Y // origin y

	// Column offsets (relative to ox)
	lBorderL := 0
	lContent := 1
	lBorderR := lContent + pw
	divCol := lBorderR + 1
	rBorderL := divCol + 1
	rContent := rBorderL + 1
	rBorderR := rContent + pw

	// Top border row
	setCell(s, ox+lBorderL, oy, "╭", lBorder)
	for i := range pw {
		setCell(s, ox+lContent+i, oy, "─", lBorder)
	}
	setCell(s, ox+lBorderR, oy, "╮", lBorder)
	setCell(s, ox+divCol, oy, "│", divStyle)
	setCell(s, ox+rBorderL, oy, "╭", rBorder)
	for i := range pw {
		setCell(s, ox+rContent+i, oy, "─", rBorder)
	}
	setCell(s, ox+rBorderR, oy, "╮", rBorder)

	// Content rows — draw borders, then let emulator.Draw fill content
	for row := range ph {
		y := oy + 1 + row
		setCell(s, ox+lBorderL, y, "│", lBorder)
		setCell(s, ox+lBorderR, y, "│", lBorder)
		setCell(s, ox+divCol, y, "│", divStyle)
		setCell(s, ox+rBorderL, y, "│", rBorder)
		setCell(s, ox+rBorderR, y, "│", rBorder)
	}

	// Draw emulator content into the pane areas
	leftArea := image.Rect(ox+lContent, oy+1, ox+lContent+pw, oy+1+ph)
	rightArea := image.Rect(ox+rContent, oy+1, ox+rContent+pw, oy+1+ph)
	d.panes[0].Emulator().Draw(s, leftArea)
	d.panes[1].Emulator().Draw(s, rightArea)

	// Bottom border row
	botY := oy + 1 + ph
	setCell(s, ox+lBorderL, botY, "╰", lBorder)
	for i := range pw {
		setCell(s, ox+lContent+i, botY, "─", lBorder)
	}
	setCell(s, ox+lBorderR, botY, "╯", lBorder)
	setCell(s, ox+divCol, botY, "│", divStyle)
	setCell(s, ox+rBorderL, botY, "╰", rBorder)
	for i := range pw {
		setCell(s, ox+rContent+i, botY, "─", rBorder)
	}
	setCell(s, ox+rBorderR, botY, "╯", rBorder)

	// Status bar
	statusY := botY + 1
	focusLabel := "LEFT"
	if d.focused == 1 {
		focusLabel = "RIGHT"
	}
	statusText := fmt.Sprintf("  Focus: %s  │  Ctrl+A Tab: switch  │  Ctrl+A q: quit", focusLabel)
	for i, ch := range statusText {
		x := ox + i
		if x >= r.Max.X {
			break
		}
		setCell(s, x, statusY, string(ch), statStyle)
	}
}

// setCell writes a single character cell to the screen with bounds checking.
func setCell(s tea.Screen, x, y int, content string, style uv.Style) {
	bounds := s.Bounds()
	if x < bounds.Min.X || x >= bounds.Max.X || y < bounds.Min.Y || y >= bounds.Max.Y {
		return
	}
	s.SetCell(x, y, &uv.Cell{
		Content: content,
		Width:   1,
		Style:   style,
	})
}

// paneSize calculates the content dimensions for each pane.
// Layout: │content│ │content│ (+ status bar)
// Total width = 2*(pw + 2) + 1 = 2*pw + 5
func paneSize(termWidth, termHeight int) (int, int) {
	pw := (termWidth - 5) / 2
	ph := termHeight - 3 // top border + bottom border + status bar
	return pw, ph
}
