//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import (
	"context"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/graph/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// Match enumerates every node file in the graph rooted at dir, keeps only
// nodes fully satisfying filter (core.Filter.Match), and reports every
// distinct fact on each kept node that satisfied at least one filter
// statement (core.Filter.MatchingFacts). filter MUST carry at least one
// statement — an empty/absent filter is rejected via ErrEmptyFilter before
// any node file is opened, since a vacuously-matching filter would carry
// no evidence for MatchingFacts to report (research.md D2, spec FR-005).
// A node file that cannot be opened or parsed is recorded in the result's
// Unreadable list and excluded from the scan (research.md D5).
func Match(ctx context.Context, mounter fsys.Mounter, filter core.Filter, dir string) (kernel.MatchResult, error) {
	store, err := mounter.Mount(dir)
	if err != nil {
		return kernel.MatchResult{}, err
	}

	if err := guardIsGraph(store, dir); err != nil {
		return kernel.MatchResult{}, err
	}

	if len(filter.Statements) == 0 {
		return kernel.MatchResult{}, ErrEmptyFilter.With(errNoCause)
	}

	paths, err := walkNodeFiles(store)
	if err != nil {
		return kernel.MatchResult{}, err
	}

	var matches []kernel.MatchEntry
	var unreadable []string

	for _, p := range paths {
		node, ok, err := readGrepNode(store, p)
		if err != nil || !ok {
			unreadable = append(unreadable, p)
			continue
		}
		if !filter.Match(node) {
			continue
		}
		for _, fact := range filter.MatchingFacts(node) {
			matches = append(matches, kernel.MatchEntry{ID: node.ID, Property: fact.Property, Value: fact.Value})
		}
	}

	return kernel.MatchResult{
		Root:       dir,
		Matches:    matches,
		Unreadable: unreadable,
	}, nil
}
