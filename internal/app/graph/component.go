//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package graph is the graph-mutation (graph I/O) domain use-case: it
// applies a document patch into an already-initialized knowledge graph.
package graph

import (
	"context"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	configkernel "github.com/fogfish/arcnet-cli/internal/app/config/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/graph/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/graph/port"
	"github.com/fogfish/arcnet-cli/internal/app/graph/service"
	"github.com/fogfish/arcnet-cli/internal/bios"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// Apply ingests the patch at patchPath into the graph rooted at dir. It is
// a thin delegator into service.Apply.
func Apply(ctx context.Context, mounter fsys.Mounter, vcs port.VCS, reporter bios.Reporter, index core.Index, schema port.SchemaRegistry, dir, patchPath string) (kernel.ApplyResult, error) {
	return service.Apply(ctx, mounter, vcs, reporter, index, schema, dir, patchPath)
}

// Revert retracts sourceID's patch contribution from the graph rooted at
// dir. It is a thin delegator into service.Revert.
func Revert(ctx context.Context, mounter fsys.Mounter, vcs port.VCS, reporter bios.Reporter, index core.Index, dir, sourceID string) (kernel.RevertResult, error) {
	return service.Revert(ctx, mounter, vcs, reporter, index, dir, sourceID)
}

// Grep searches node file content across the graph rooted at dir for lines
// matching pattern, narrowed by filter. It is a thin delegator into
// service.Grep.
func Grep(ctx context.Context, mounter fsys.Mounter, filter core.Filter, pattern string, cfg configkernel.GrepConfig, dir string) (kernel.GrepResult, error) {
	return service.Grep(ctx, mounter, filter, pattern, cfg, dir)
}

// Subgraph extracts the node identified by basename plus every node
// reachable from it within depth hops (both directions), narrowed by
// filter, from the graph rooted at dir. When stubs is true (BUG-001, spec
// FR-017), a minimal placeholder node is also emitted for every
// extraction-boundary link target. It is a thin delegator into
// service.Subgraph.
func Subgraph(ctx context.Context, mounter fsys.Mounter, filter core.Filter, basename string, depth int, cfg configkernel.SubgraphConfig, dir string, stubs bool) (kernel.SubgraphResult, error) {
	return service.Subgraph(ctx, mounter, filter, basename, depth, cfg, dir, stubs)
}

// NodeGet fetches the single node identified by id from the graph rooted at
// dir. It is a thin delegator into service.NodeGet.
func NodeGet(ctx context.Context, mounter fsys.Mounter, dir, id string) (core.Node, error) {
	return service.NodeGet(ctx, mounter, dir, id)
}

// EnsureGraph confirms dir is an initialized graph, without reading any node
// — the preflight arc serve calls before starting its MCP transport. It is a
// thin delegator into service.EnsureGraph.
func EnsureGraph(ctx context.Context, mounter fsys.Mounter, dir string) error {
	return service.EnsureGraph(ctx, mounter, dir)
}
