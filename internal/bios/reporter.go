//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package bios

import (
	"fmt"
	"io"
	"os"
	"time"
)

// Reporter is the port for progress/status events during a long-running
// command (ADR 002 DS-06). Service code depends on this interface, never on
// os.Stderr directly.
type Reporter interface {
	Start(label string)
	Step(label string)
	Done(label string, elapsed time.Duration)
	Error(label string, err error)
}

type stderrReporter struct{ w io.Writer }

func (r stderrReporter) Start(label string) {
	fmt.Fprintf(r.w, "▶ %s\n", label)
}

func (r stderrReporter) Step(label string) {
	fmt.Fprintf(r.w, "  %s\n", label)
}

// Done renders one faint/gray line per completed step (BUG-001: --verbose
// progress is not a success confirmation, so it MUST NOT reuse StatusOK's
// green). The styled text is rendered WITHOUT an embedded newline — lipgloss
// treats multi-line input as a block and pads every line to the block's
// width, so a trailing "\n" inside Render(...) is replaced with padding
// spaces instead of a line break (BUG-001 root cause). The newline is
// written separately, outside the styled span.
func (r stderrReporter) Done(label string, elapsed time.Duration) {
	text := fmt.Sprintf("%s (%s)", label, elapsed.Round(time.Millisecond))
	fmt.Fprintln(r.w, SCHEMA.Hint.Render(text))
}

func (r stderrReporter) Error(label string, err error) {
	text := fmt.Sprintf("%s%s failed: %s", SCHEMA.IconFail, label, err)
	fmt.Fprintln(r.w, SCHEMA.StatusFail.Render(text))
}

// silentReporter is the Null Object: every method is a no-op.
type silentReporter struct{}

func (silentReporter) Start(string)               {}
func (silentReporter) Step(string)                {}
func (silentReporter) Done(string, time.Duration) {}
func (silentReporter) Error(string, error)        {}

// NewReporter selects the Reporter implementation once, at command setup.
// quiet always forces silence; silent forces it independent of quiet (used
// by callers to additionally gate visibility on --verbose, since progress
// is opt-in via --verbose, not shown by default — BUG-001).
func NewReporter(quiet, silent bool) Reporter {
	if quiet || silent {
		return silentReporter{}
	}
	return stderrReporter{w: os.Stderr}
}
