//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service_test

import (
	"context"
	"errors"
	"testing"
	"testing/fstest"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/graph/service"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// specs/028-node-match-filter data-model.md D2: an empty/missing filter is
// rejected before any node file is opened.
func TestMatchEmptyFilterReturnsErrEmptyFilter(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Source/a.md": grepSourceNodeA,
	})

	_, err := service.Match(context.Background(), mounter, core.Filter{}, "/graph")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrEmptyFilter)))
}

func TestMatchGuardNotAGraph(t *testing.T) {
	mounter := grepMounter{store: grepStore{fstest.MapFS{}}}
	filter := filterType("Source")

	_, err := service.Match(context.Background(), mounter, filter, "/graph")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrNotAGraph)))
}

func TestMatchFilterMatchingZeroNodesReturnsEmptyMatches(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Source/a.md": grepSourceNodeA,
	})
	filter := filterType("Resource")

	result, err := service.Match(context.Background(), mounter, filter, "/graph")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(0, len(result.Matches)))
}

func TestMatchUnreadableNodeExcludedAndReported(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Source/a.md":      grepSourceNodeA,
		"Source/broken.md": "not a valid node front matter",
	})
	filter := filterType("Source")

	result, err := service.Match(context.Background(), mounter, filter, "/graph")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, len(result.Unreadable))).
		Should(it.Equal("Source/broken.md", result.Unreadable[0]))
}

func TestMatchReturnsOneMatchEntryPerFact(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Source/a.md": grepSourceNodeA,
	})
	filter := filterType("Source")

	result, err := service.Match(context.Background(), mounter, filter, "/graph")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(result.Matches)))
	it.Then(t).
		Should(it.Equal("a", result.Matches[0].ID)).
		Should(it.Equal("type", result.Matches[0].Property)).
		Should(it.Equal("Source", result.Matches[0].Value))
}

func TestMatchResultRootIsDir(t *testing.T) {
	mounter := newGrepGraph(map[string]string{"Source/a.md": grepSourceNodeA})
	filter := filterType("Source")

	result, err := service.Match(context.Background(), mounter, filter, "/graph")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal("/graph", result.Root))
}
