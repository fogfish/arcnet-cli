//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package bios

import (
	"regexp"
	"strings"
)

// bracketPrefix strips the "[pkg.Func line]" source-location prefix that
// github.com/fogfish/faults injects at every wrapping layer — useful for
// debugging, never for text a user reads. Anchored to end in " <digits>]"
// (the line number faults always appends) rather than matching any "[...]"
// span — otherwise a wrapped error whose own text contains a bracket pair
// (e.g. an invalid regexp's "missing closing ]" message echoing the
// offending pattern) gets its real message silently eaten instead of just
// the debug prefix (BUG discovered by specs/006-arc-grep-content-search).
var bracketPrefix = regexp.MustCompile(`\[[^\[\]]*\s\d+\]\s*`)

// Humanize renders err as the single human-readable sentence a user should
// see, with every faults debug prefix removed and the trailing ": " an
// errNoCause-wrapped guard leaves behind trimmed off.
//
// cmd/arc/main.go is the sole site that prints a *failing command's* error
// (ADR 002 DS-07) and uses this for that line. It lives here in the shared
// kernel because a command may also carry an error forward as data rather
// than raising it — arc apply batch records each failed patch's reason in
// kernel.BatchResult, where it is both rendered in the summary and
// serialized into the --json contract (specs/020-apply-batch FR-013,
// FR-018), and neither audience wants a source location in it.
func Humanize(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimSuffix(bracketPrefix.ReplaceAllString(err.Error(), ""), ": ")
}
