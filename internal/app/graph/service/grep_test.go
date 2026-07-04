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

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	configkernel "github.com/fogfish/arcnet-cli/internal/app/config/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/graph/service"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// grepStore is a read-only fsys.Store backed by fstest.MapFS — arc grep
// never writes, so Create/Remove are never exercised.
type grepStore struct{ fstest.MapFS }

func (grepStore) Create(name string) (fsys.File, error) { return nil, errors.New("read-only store") }
func (grepStore) Remove(name string) error               { return errors.New("read-only store") }

type grepMounter struct{ store grepStore }

func (m grepMounter) Mount(root string) (fsys.Store, error) { return m.store, nil }

func newGrepGraph(files map[string]string) grepMounter {
	m := fstest.MapFS{".arc/.gitkeep": &fstest.MapFile{}}
	for path, content := range files {
		m[path] = &fstest.MapFile{Data: []byte(content)}
	}
	return grepMounter{store: grepStore{m}}
}

const grepSourceNodeA = `---
kind: source
id: a
---
# a

TLS 1.3 is great.
`

const grepSourceNodeMultiLine = `---
kind: source
id: multi
---
# multi

TLS appears here.
Another line.
TLS appears again.
`

const grepEntityNodeB = `---
kind: entity
title: b
---
# b

No match here.
`

func TestGrepGuardNotAGraph(t *testing.T) {
	mounter := grepMounter{store: grepStore{fstest.MapFS{}}}

	_, err := service.Grep(context.Background(), mounter, core.Filter{}, "TLS", configkernel.GrepConfig{}, "/graph")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrNotAGraph)))
}

func TestGrepEmptyFilterScansEveryNode(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"sources/a.md":  grepSourceNodeA,
		"entities/b.md": grepEntityNodeB,
	})

	result, err := service.Grep(context.Background(), mounter, core.Filter{}, "TLS", configkernel.GrepConfig{}, "/graph")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(result.Matches)))
	it.Then(t).
		Should(it.Equal(core.Kind("source"), result.Matches[0].Kind)).
		Should(it.Equal("a", result.Matches[0].ID)).
		Should(it.Equal("sources/a.md", result.Matches[0].Path))
}

func TestGrepFilterExcludesNonMatchingNodesFromScan(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"sources/a.md":  grepSourceNodeA,
		"entities/b.md": grepEntityNodeB,
	})
	filter := core.Filter{Kinds: []core.Kind{"entity"}}

	result, err := service.Grep(context.Background(), mounter, filter, "TLS", configkernel.GrepConfig{}, "/graph")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(0, len(result.Matches)))
}

func TestGrepUnreadableNodeExcludedAndReported(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"sources/a.md":      grepSourceNodeA,
		"sources/broken.md": "not a valid node front matter",
	})

	result, err := service.Grep(context.Background(), mounter, core.Filter{}, "TLS", configkernel.GrepConfig{}, "/graph")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(result.Matches)))
	it.Then(t).
		Should(it.Equal(1, len(result.Unreadable))).
		Should(it.Equal("sources/broken.md", result.Unreadable[0]))
}

func TestGrepInvalidPatternReturnsError(t *testing.T) {
	mounter := newGrepGraph(map[string]string{"sources/a.md": grepSourceNodeA})

	_, err := service.Grep(context.Background(), mounter, core.Filter{}, "[TLS", configkernel.GrepConfig{}, "/graph")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrInvalidPattern)))
}

// arc grep TLS
// Scenario 1 & 3 from spec.md US1: every match is labeled with the correct
// kind/id, and a node matching on multiple lines produces one kernel.Match
// per line, in line order.
func TestGrepMultiLineNodeProducesOneMatchPerLineInOrder(t *testing.T) {
	mounter := newGrepGraph(map[string]string{"sources/multi.md": grepSourceNodeMultiLine})

	result, err := service.Grep(context.Background(), mounter, core.Filter{}, "TLS", configkernel.GrepConfig{}, "/graph")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(2, len(result.Matches)))
	it.Then(t).
		Should(it.Equal("source", string(result.Matches[0].Kind))).
		Should(it.Equal("multi", result.Matches[0].ID)).
		Should(it.True(result.Matches[0].Line < result.Matches[1].Line))
}

func TestGrepCombinedFilterMatchesZeroNodes(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"sources/a.md":  grepSourceNodeA,
		"entities/b.md": grepEntityNodeB,
	})
	filter := core.Filter{Kinds: []core.Kind{"resource"}}

	result, err := service.Grep(context.Background(), mounter, filter, "TLS", configkernel.GrepConfig{}, "/graph")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(0, len(result.Matches)))
}
