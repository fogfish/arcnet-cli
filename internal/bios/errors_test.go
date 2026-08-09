//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package bios_test

import (
	"errors"
	"testing"

	"github.com/fogfish/faults"
	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/bios"
)

const errExample = faults.Safe1[string]("something failed for %s")

func TestHumanizeStripsFaultsSourceLocation(t *testing.T) {
	err := errExample.With(errors.New("root cause"), "widget")

	it.Then(t).Should(it.Equal("something failed for widget: root cause", bios.Humanize(err)))
}

// A guard with no underlying cause is wrapped with an empty error, so the
// rendered message must not keep the trailing ": " that leaves behind.
func TestHumanizeTrimsTrailingSeparatorOfACauselessGuard(t *testing.T) {
	err := errExample.With(errors.New(""), "widget")

	it.Then(t).Should(it.Equal("something failed for widget", bios.Humanize(err)))
}

// The prefix pattern must not eat a bracket pair belonging to the message
// itself (specs/006-arc-grep-content-search's regression).
func TestHumanizeKeepsBracketsBelongingToTheMessage(t *testing.T) {
	err := errExample.With(errors.New("missing closing ]"), "[a-z")

	it.Then(t).Should(it.String(bios.Humanize(err)).Contain("missing closing ]"))
}

func TestHumanizeOfNilIsEmpty(t *testing.T) {
	it.Then(t).Should(it.Equal("", bios.Humanize(nil)))
}
