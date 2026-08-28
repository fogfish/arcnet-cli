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

	configkernel "github.com/fogfish/arcnet-cli/internal/app/config/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/graph/service"
	"github.com/fogfish/arcnet-cli/internal/core"
)

const contextSourceMatch = `---
"@id": tls-source
"@type": Source
---
# tls-source

TLS 1.3 is the latest version.
`

const contextSourceMatchB = `---
"@id": tls-source-b
"@type": Source
---
# tls-source-b

TLS 1.3 mentioned again.

- [[tls-source]]
`

// contextEntityNeighbor's body never mentions "TLS 1.3" itself — it is only
// reachable via the backlink neighbor expansion pass, from its own edge to
// tls-source.
const contextEntityNeighbor = `---
"@id": tls-entity
"@type": Entity
---
# tls-entity

Unrelated prose.

- [[tls-source]]
`

const contextEntityAttrMatch = `---
"@id": attr-entity
"@type": Entity
category: TLS handshake protocol process
---
# attr-entity

No matching prose here.
`

func TestContextRetrieveGuardNotAGraph(t *testing.T) {
	mounter := grepMounter{store: grepStore{fstest.MapFS{}}}

	_, err := service.ContextRetrieve(context.Background(), mounter, core.Filter{}, "TLS", 10, configkernel.GrepConfig{}, configkernel.SubgraphConfig{}, "/graph")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrNotAGraph)))
}

func TestContextRetrieveContentMatchHit(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Source/tls-source.md": contextSourceMatch,
	})

	result, err := service.ContextRetrieve(context.Background(), mounter, core.Filter{}, "TLS 1.3", 10, configkernel.GrepConfig{}, configkernel.SubgraphConfig{}, "/graph")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, len(result.Patch.Nodes))).
		Should(it.Equal("tls-source", result.Patch.Nodes[0].ID)).
		Should(it.Equal(1, result.ContentMatched))
}

func TestContextRetrieveAttributeMatchHit(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/attr-entity.md": contextEntityAttrMatch,
	})

	result, err := service.ContextRetrieve(context.Background(), mounter, core.Filter{}, "TLS handshake", 10, configkernel.GrepConfig{}, configkernel.SubgraphConfig{}, "/graph")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, len(result.Patch.Nodes))).
		Should(it.Equal("attr-entity", result.Patch.Nodes[0].ID)).
		Should(it.Equal(1, result.AttrMatched))
}

func TestContextRetrieveNeighborExpansionHit(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Source/tls-source.md": contextSourceMatch,
		"Entity/tls-entity.md": contextEntityNeighbor,
	})

	result, err := service.ContextRetrieve(context.Background(), mounter, core.Filter{}, "TLS 1.3", 10, configkernel.GrepConfig{}, configkernel.SubgraphConfig{}, "/graph")

	it.Then(t).Should(it.Nil(err))
	ids := nodeIDSet(result.Patch.Nodes)
	it.Then(t).
		Should(it.Equal(2, len(result.Patch.Nodes))).
		Should(it.True(ids["tls-source"])).
		Should(it.True(ids["tls-entity"])).
		Should(it.Equal(1, result.NeighborIncluded))
}

// tls-source is both a direct content match and a backlink neighbor of
// tls-source-b (which also directly matches) — it must appear exactly
// once.
func TestContextRetrieveDedupNodeReachableTwoWays(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Source/tls-source.md":   contextSourceMatch,
		"Source/tls-source-b.md": contextSourceMatchB,
	})

	result, err := service.ContextRetrieve(context.Background(), mounter, core.Filter{}, "TLS 1.3", 10, configkernel.GrepConfig{}, configkernel.SubgraphConfig{}, "/graph")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(2, len(result.Patch.Nodes)))
}

func TestContextRetrieveNoMatchReturnsEmptyNotError(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Source/tls-source.md": contextSourceMatch,
	})

	result, err := service.ContextRetrieve(context.Background(), mounter, core.Filter{}, "quantum key distribution", 10, configkernel.GrepConfig{}, configkernel.SubgraphConfig{}, "/graph")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(0, len(result.Patch.Nodes)))
}

func TestContextRetrieveFilterExcludesReachableNeighbor(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Source/tls-source.md": contextSourceMatch,
		"Entity/tls-entity.md": contextEntityNeighbor,
	})
	filter := filterType("Source")

	result, err := service.ContextRetrieve(context.Background(), mounter, filter, "TLS 1.3", 10, configkernel.GrepConfig{}, configkernel.SubgraphConfig{}, "/graph")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, len(result.Patch.Nodes))).
		Should(it.Equal("tls-source", result.Patch.Nodes[0].ID))
}

func TestContextRetrieveFullyExcludingFilterReturnsEmptyNotError(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Source/tls-source.md": contextSourceMatch,
	})
	filter := filterType("Resource")

	result, err := service.ContextRetrieve(context.Background(), mounter, filter, "TLS 1.3", 10, configkernel.GrepConfig{}, configkernel.SubgraphConfig{}, "/graph")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(0, len(result.Patch.Nodes)))
}

func TestContextRetrieveFewerCandidatesThanLimitReturnsAllNoPadding(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Source/tls-source.md": contextSourceMatch,
		"Entity/tls-entity.md": contextEntityNeighbor,
	})

	result, err := service.ContextRetrieve(context.Background(), mounter, core.Filter{}, "TLS 1.3", 10, configkernel.GrepConfig{}, configkernel.SubgraphConfig{}, "/graph")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(2, len(result.Patch.Nodes))).
		ShouldNot(it.True(result.Truncated))
}

// Direct matches (content or attribute) rank ahead of neighbor-only
// matches (Clarifications 2026-08-24), so both direct-tier candidates
// survive a limit that only leaves room for two out of three.
func TestContextRetrieveMoreCandidatesThanLimitTruncatesToHighestRanked(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Source/tls-source.md":   contextSourceMatch,
		"Source/tls-source-b.md": contextSourceMatchB,
		"Entity/tls-entity.md":   contextEntityNeighbor,
	})

	result, err := service.ContextRetrieve(context.Background(), mounter, core.Filter{}, "TLS 1.3", 2, configkernel.GrepConfig{}, configkernel.SubgraphConfig{}, "/graph")

	it.Then(t).Should(it.Nil(err))
	ids := nodeIDSet(result.Patch.Nodes)
	it.Then(t).
		Should(it.Equal(2, len(result.Patch.Nodes))).
		Should(it.True(result.Truncated)).
		Should(it.True(ids["tls-source"])).
		Should(it.True(ids["tls-source-b"]))
}

func TestContextRetrieveInvalidLimitReturnsErrInvalidLimit(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Source/tls-source.md": contextSourceMatch,
	})

	_, err := service.ContextRetrieve(context.Background(), mounter, core.Filter{}, "TLS 1.3", 0, configkernel.GrepConfig{}, configkernel.SubgraphConfig{}, "/graph")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrInvalidLimit)))
}
