package tui

import "image/color"

// Catppuccin Mocha color palette for TUI borders and status bar.
var (
	colorFocused = color.RGBA{R: 0xa6, G: 0xe3, B: 0xa1, A: 0xff} // #a6e3a1 green
	colorBlurred = color.RGBA{R: 0x45, G: 0x47, B: 0x5a, A: 0xff} // #45475a dark gray
	colorStatus  = color.RGBA{R: 0x6c, G: 0x70, B: 0x86, A: 0xff} // #6c7086 muted
)
