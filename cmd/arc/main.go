//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/fogfish/arcnet-cli/internal/bios"
)

// bracketPrefix strips the "[pkg.Func line]" source-location prefix that
// github.com/fogfish/faults injects at every wrapping layer — useful for
// debugging, not for the single human-readable line this is the sole site
// (DS-07) responsible for printing.
var bracketPrefix = regexp.MustCompile(`\[[^\]]+\]\s*`)

func humanize(err error) string {
	msg := bracketPrefix.ReplaceAllString(err.Error(), "")
	return strings.TrimSuffix(msg, ": ")
}

func main() {
	cmd, err := newRootCmd().ExecuteC()
	if err != nil {
		// BUG-001: render only the single-line message through lipgloss —
		// passing the whole multi-line block (blank lines + hint) into one
		// Render() call hits the same block-padding bug fixed in
		// internal/bios/reporter.go: a styled multi-line string gets every
		// line padded to equal width instead of preserving line breaks.
		message := bios.SCHEMA.StatusFail.Render(bios.SCHEMA.IconFail + humanize(err))
		fmt.Fprintf(os.Stderr, "\n %s\n   Run `arc help %s` for guidance.\n\n", message, cmd.Name())
		os.Exit(1)
	}
}
