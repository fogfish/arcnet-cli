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
// (DS-07) responsible for printing. Anchored to end in " <digits>]" (the
// line number faults always appends) rather than matching any "[...]"
// span — otherwise a wrapped error whose own text contains a bracket pair
// (e.g. an invalid regexp's "missing closing ]" message echoing the
// offending pattern) gets its real message silently eaten instead of just
// the debug prefix (BUG discovered by specs/006-arc-grep-content-search).
var bracketPrefix = regexp.MustCompile(`\[[^\[\]]*\s\d+\]\s*`)

func humanize(err error) string {
	msg := bracketPrefix.ReplaceAllString(err.Error(), "")
	return strings.TrimSuffix(msg, ": ")
}

func main() {
	cmd, err := newRootCmd().ExecuteC()
	if err != nil {
		// bios.ErrSilent (DS-07): the command already printed its complete
		// result (e.g. arc lint's violation list) — exit non-zero with no
		// second, redundant error line.
		if err == bios.ErrSilent {
			os.Exit(1)
		}

		// BUG-001: render only the single-line message through lipgloss —
		// passing the whole multi-line block (blank lines + hint) into one
		// Render() call hits the same block-padding bug fixed in
		// internal/bios/reporter.go: a styled multi-line string gets every
		// line padded to equal width instead of preserving line breaks.
		message := bios.SCHEMA.StatusFail.Render(bios.SCHEMA.IconFail + humanize(err))
		// cmd.Name() is only the leaf command's own Use word ("schema" for
		// "arc apply schema") — CommandPath() minus the root's own name
		// gives the full subcommand path a nested command needs for its
		// "arc help <path>" hint to actually resolve.
		helpPath := strings.TrimPrefix(cmd.CommandPath(), cmd.Root().Name()+" ")
		fmt.Fprintf(os.Stderr, "\n %s\n   Run `arc help %s` for guidance.\n\n", message, helpPath)
		os.Exit(1)
	}
}
