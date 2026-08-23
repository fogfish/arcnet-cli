//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package core

import (
	"fmt"
	"strings"
)

// forbiddenIdentityChars is ARCNET-CORE §7.1's closed set of characters no
// node identity ("@id"/map-key) may contain (data-model.md §3, research.md
// D6) — the sole definition both arc lint and arc apply consume, so the two
// commands can never drift onto two different character sets.
var forbiddenIdentityChars = []rune{'/', '\\', ':', '*', '?', '"', '<', '>', '|', '.'}

func isForbiddenIdentityChar(r rune) bool {
	for _, f := range forbiddenIdentityChars {
		if r == f {
			return true
		}
	}
	return false
}

// IdentityCharPosition is one forbidden character found in a node identity,
// paired with its 1-indexed rune position within that identity (research.md
// D5).
type IdentityCharPosition struct {
	Char     rune
	Position int
}

// ScanIdentityCharset returns every forbiddenIdentityChars occurrence in id,
// in order, 1-indexed by rune position — not byte offset, so a multi-byte
// UTF-8 character preceding a forbidden ASCII one never skews the reported
// position (research.md D5). Returns nil when id is legal.
func ScanIdentityCharset(id string) []IdentityCharPosition {
	var out []IdentityCharPosition
	for i, r := range []rune(id) {
		if isForbiddenIdentityChar(r) {
			out = append(out, IdentityCharPosition{Char: r, Position: i + 1})
		}
	}
	return out
}

// FormatIdentityCharsetViolation renders pairs (as returned by
// ScanIdentityCharset) into the detail fragment arc lint's RuleIdentityCharset
// violation and arc apply's ErrIdentityCharset rejection share verbatim
// (data-model.md §4, research.md D8) — every offending character named, not
// only the first.
func FormatIdentityCharsetViolation(pairs []IdentityCharPosition) string {
	word := "character"
	if len(pairs) > 1 {
		word = "characters"
	}
	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = fmt.Sprintf("%q at position %d", string(p.Char), p.Position)
	}
	return fmt.Sprintf("contains forbidden %s %s", word, strings.Join(parts, ", "))
}
