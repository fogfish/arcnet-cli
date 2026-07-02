// Package bios is the shared kernel (ADR 002 DS-04, DS-05, DS-06) reused by
// every command: output-mode resolution, the color schema, and progress
// reporting.
package bios

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

// Schema is the set of styles and icons every command renders through.
type Schema struct {
	StatusOK   lipgloss.Style
	StatusWarn lipgloss.Style
	StatusFail lipgloss.Style
	Hint       lipgloss.Style
	IconOK     string
	IconWarn   string
	IconFail   string
}

var SCHEMA_PLAIN = Schema{
	StatusOK:   lipgloss.NewStyle(),
	StatusWarn: lipgloss.NewStyle(),
	StatusFail: lipgloss.NewStyle(),
	Hint:       lipgloss.NewStyle(),
}

var SCHEMA_COLOR = Schema{
	StatusOK:   lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
	StatusWarn: lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
	StatusFail: lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
	Hint:       lipgloss.NewStyle().Faint(true),
	IconOK:     "✅ ",
	IconWarn:   "🟧 ",
	IconFail:   "❌ ",
}

var SCHEMA = SCHEMA_PLAIN

// SelectSchema resolves SCHEMA exactly once, at command setup, per the
// TTY/NO_COLOR/TERM=dumb/--color rules (constitution Principle X, ADR 002
// DS-05). forceColor is the --color/-C flag; out is the stream color would
// be rendered to (os.Stdout).
func SelectSchema(forceColor bool, out *os.File) {
	noColor := os.Getenv("NO_COLOR") != ""
	dumbTerm := os.Getenv("TERM") == "dumb"
	isTTY := isatty.IsTerminal(out.Fd()) || isatty.IsCygwinTerminal(out.Fd())

	if !noColor && !dumbTerm && (forceColor || isTTY) {
		SCHEMA = SCHEMA_COLOR
		return
	}
	SCHEMA = SCHEMA_PLAIN
}
