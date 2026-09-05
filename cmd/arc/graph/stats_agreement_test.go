//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package graph

import (
	"context"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/adapter/git"
	appgraph "github.com/fogfish/arcnet-cli/internal/app/graph"
	applint "github.com/fogfish/arcnet-cli/internal/app/lint"
	lintkernel "github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
	appschema "github.com/fogfish/arcnet-cli/internal/app/schema"
	"github.com/fogfish/arcnet-cli/internal/bios"
)

// TestStatsBrokenLinksMatchLint is contract C5 / SC-003: arc stats and
// arc lint MUST never disagree about how many broken links a graph has.
//
// It lives here, in the composition root, rather than being enforced by
// shared code: internal/app/graph must not import internal/app/lint
// (sibling use-cases, Principle V's acyclic rule), and a shared helper
// would silently keep the two in agreement even while both drifted away
// from the rule. Running both commands over one fixture and comparing the
// two numbers fails when EITHER side changes (research.md D6).
func TestStatsBrokenLinksMatchLint(t *testing.T) {
	dir := fixtureBroken(t)
	runGit(t, dir, "init")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "graph(init): broken-link fixture")

	store, err := (fsys.Local{}).Mount(dir)
	it.Then(t).Must(it.Nil(err))

	index, err := appschema.Resolve(store)
	it.Then(t).Must(it.Nil(err))

	silent := bios.NewReporter(true, true)
	ctx := context.Background()

	lintResult, err := applint.Lint(ctx, fsys.Local{}, git.New(silent), silent, index, dir)
	it.Then(t).Must(it.Nil(err))

	statsResult, err := appgraph.Stats(ctx, fsys.Local{}, index, dir, true)
	it.Then(t).Must(it.Nil(err))

	linkResolves := 0
	for _, v := range lintResult.Violations {
		if v.Rule == lintkernel.RuleLinkResolves {
			linkResolves++
		}
	}

	it.Then(t).
		Should(it.Equal(4, linkResolves)).
		Should(it.Equal(linkResolves, statsResult.BrokenLinks))
}
