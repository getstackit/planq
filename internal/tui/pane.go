// Package tui provides a dual-pane terminal TUI using charmbracelet/x/vt
// for terminal emulation and bubbletea v2 for rendering.
package tui

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync/atomic"

	"github.com/charmbracelet/x/vt"
	"github.com/charmbracelet/x/xpty"
)

// Pane wraps a single PTY + terminal emulator pair.
type Pane struct {
	pty  xpty.Pty
	emu  *vt.SafeEmulator
	cmd  *exec.Cmd
	done atomic.Bool
}

// NewPane creates a PTY, starts the command, creates the emulator,
// and launches goroutines to pipe output between them.
func NewPane(w, h int, cmd *exec.Cmd) (*Pane, error) {
	pty, err := xpty.NewPty(w, h)
	if err != nil {
		return nil, fmt.Errorf("creating pty: %w", err)
	}

	if err := pty.Start(cmd); err != nil {
		pty.Close()
		return nil, fmt.Errorf("starting command: %w", err)
	}

	emu := vt.NewSafeEmulator(w, h)

	p := &Pane{
		pty: pty,
		emu: emu,
		cmd: cmd,
	}

	// Pipe PTY output → emulator (terminal state updates).
	go func() {
		io.Copy(emu, pty) //nolint:errcheck
		p.done.Store(true)
	}()

	// Pipe emulator responses → PTY (e.g. DA responses, cursor position reports).
	go func() {
		io.Copy(pty, emu) //nolint:errcheck
	}()

	// Wait for the process to exit.
	go func() {
		xpty.WaitProcess(context.Background(), cmd) //nolint:errcheck
		p.done.Store(true)
	}()

	return p, nil
}

// Emulator returns the thread-safe terminal emulator.
func (p *Pane) Emulator() *vt.SafeEmulator {
	return p.emu
}

// Exited returns true if the child process has exited.
func (p *Pane) Exited() bool {
	return p.done.Load()
}

// Resize updates both the PTY and emulator dimensions.
func (p *Pane) Resize(w, h int) error {
	if err := p.pty.Resize(w, h); err != nil {
		return fmt.Errorf("resizing pty: %w", err)
	}
	p.emu.Resize(w, h)
	return nil
}

// Close shuts down the PTY and emulator.
func (p *Pane) Close() error {
	p.emu.Close()
	return p.pty.Close()
}
