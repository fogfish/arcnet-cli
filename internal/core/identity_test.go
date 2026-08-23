//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package core_test

import (
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/core"
)

func TestScanIdentityCharsetLegalControlIdentity(t *testing.T) {
	out := core.ScanIdentityCharset("rescorla-2026-tls13")
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestScanIdentityCharsetEachForbiddenCharacter(t *testing.T) {
	cases := []struct {
		char     rune
		id       string
		position int
	}{
		{'/', "Handshake/Protocol", 10},
		{'\\', "Handshake\\Protocol", 10},
		{':', "Handshake:Protocol", 10},
		{'*', "Handshake*Protocol", 10},
		{'?', "Handshake?Protocol", 10},
		{'"', "Handshake\"Protocol", 10},
		{'<', "Handshake<Protocol", 10},
		{'>', "Handshake>Protocol", 10},
		{'|', "Handshake|Protocol", 10},
		{'.', "Handshake.Protocol", 10},
	}

	for _, c := range cases {
		out := core.ScanIdentityCharset(c.id)
		it.Then(t).Should(it.Equal(1, len(out)))
		it.Then(t).
			Should(it.Equal(c.char, out[0].Char)).
			Should(it.Equal(c.position, out[0].Position))
	}
}

func TestScanIdentityCharsetMultipleOffendingCharactersAllNamed(t *testing.T) {
	out := core.ScanIdentityCharset("v1.3: Handshake/Protocol")

	it.Then(t).Should(it.Equal(3, len(out)))
	it.Then(t).
		Should(it.Equal('.', out[0].Char)).
		Should(it.Equal(3, out[0].Position)).
		Should(it.Equal(':', out[1].Char)).
		Should(it.Equal(5, out[1].Position)).
		Should(it.Equal('/', out[2].Char)).
		Should(it.Equal(16, out[2].Position))
}

func TestFormatIdentityCharsetViolationSingle(t *testing.T) {
	pairs := core.ScanIdentityCharset("Handshake/Protocol")
	msg := core.FormatIdentityCharsetViolation(pairs)

	it.Then(t).Should(it.Equal(`contains forbidden character "/" at position 10`, msg))
}

func TestFormatIdentityCharsetViolationMultiple(t *testing.T) {
	pairs := core.ScanIdentityCharset("v1.3: Handshake/Protocol")
	msg := core.FormatIdentityCharsetViolation(pairs)

	it.Then(t).Should(it.Equal(`contains forbidden characters "." at position 3, ":" at position 5, "/" at position 16`, msg))
}
