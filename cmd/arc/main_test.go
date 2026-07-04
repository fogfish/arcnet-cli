//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package main

import (
	"errors"
	"testing"

	"github.com/fogfish/it/v2"
)

func TestHumanizeStripsDebugPrefix(t *testing.T) {
	err := errors.New("[github.com/fogfish/arcnet-cli/cmd/arc/graph.NewGrepCmd.func1 123] something failed: root cause")

	it.Then(t).Should(it.Equal("something failed: root cause", humanize(err)))
}

// BUG discovered by specs/006-arc-grep-content-search: a wrapped error
// whose own message legitimately contains a bracket pair (e.g. an invalid
// regexp's "missing closing ]" message, which echoes the offending pattern
// in brackets) must keep that content — only the "[pkg.Func <digits>]"
// debug prefix faults injects is stripped, never an arbitrary bracket span.
func TestHumanizePreservesBracketsInsideMessage(t *testing.T) {
	err := errors.New("[github.com/fogfish/arcnet-cli/internal/app/graph/service_test.Test 13] [TLS is not a valid pattern: error parsing regexp: missing closing ]: `[TLS`")

	it.Then(t).Should(it.String(humanize(err)).Contain("is not a valid pattern"))
}
