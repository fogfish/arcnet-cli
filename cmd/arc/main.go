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
	"strings"

	"github.com/fogfish/arcnet-cli/internal/bios"
)

// humanize renders err as the single human-readable line this is the sole
// site (DS-07) responsible for printing. The stripping rules themselves now
// live in internal/bios, because arc apply batch also needs them for the
// per-patch failure reasons it carries as data rather than raising
// (specs/020-apply-batch FR-013).
func humanize(err error) string { return bios.Humanize(err) }

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
