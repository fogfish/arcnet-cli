//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package bios

import (
	"errors"
	"testing"

	"github.com/fogfish/it/v2"
)

// go test's own stdin is never a terminal (research.md D10, quickstart.md
// Scenario F's "piped stdin, no --force" case) — Confirm must refuse
// rather than hang or silently proceed.
func TestConfirmRefusesWhenStdinNotATerminal(t *testing.T) {
	ok, err := Confirm("remove this node?")

	it.Then(t).
		Should(it.True(!ok)).
		Should(it.True(errors.Is(err, ErrConfirmRefused)))
}

func TestParseConfirmAnswer(t *testing.T) {
	for _, tc := range []struct {
		line string
		want bool
	}{
		{"y", true},
		{"Y", true},
		{"yes", true},
		{"YES", true},
		{"  y  ", true},
		{"n", false},
		{"N", false},
		{"no", false},
		{"", false},
		{"maybe", false},
	} {
		it.Then(t).Should(it.Equal(tc.want, parseConfirmAnswer(tc.line)))
	}
}
