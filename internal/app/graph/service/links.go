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

// nodeRelations returns every outgoing relation n carries — its own Edges
// followed by its own HRefs (spec 030 Assumptions: node_links/
// node_backlinks deliberately treat inline prose references as relations,
// unlike nodeTargets, which stays Edges-only for Subgraph's traversal).
func nodeRelations(n core.Node) []core.Link {
	return append(append([]core.Link(nil), n.Edges...), n.HRefs...)
}

// buildRelationReverseIndex mirrors buildReverseIndex (subgraph.go), but
// iterates nodeRelations(n) instead of n.Edges alone, so node_backlinks
// reports HRefs-sourced relations too (spec 030 FR-004/FR-008). Reuses
// the existing reverseIndex/backlinkEdge types unchanged; Subgraph's own
// buildReverseIndex call is untouched.
func buildRelationReverseIndex(index nodeIndex) reverseIndex {
	rev := reverseIndex{}
	for id, n := range index {
		for _, e := range nodeRelations(n) {
			rev[e.Target] = append(rev[e.Target], backlinkEdge{Source: id, Predicate: e.Predicate})
		}
	}
	return rev
}

// NodeLinks mounts dir, enumerates and indexes every node (reusing
// enumerateNodes/guardIsGraph, unchanged from NodeGet's own precedent),
// looks up id, returning ErrSeedNotFound on a miss, and reports every
// outgoing relation on that node — its own Edges followed by its own
// HRefs (nodeRelations; spec 030 FR-002/FR-008) — as []core.Link.
func NodeLinks(ctx context.Context, mounter fsys.Mounter, dir, id string) ([]core.Link, error) {
	store, err := mounter.Mount(dir)
	if err != nil {
		return nil, err
	}

	if err := guardIsGraph(store, dir); err != nil {
		return nil, err
	}

	index, err := enumerateNodes(store)
	if err != nil {
		return nil, err
	}

	node, ok := index[id]
	if !ok {
		return nil, ErrSeedNotFound.With(errNoCause, id)
	}
	return nodeRelations(node), nil
}

// NodeBacklinks mounts dir, enumerates and indexes every node, looks up
// id (ErrSeedNotFound on a miss — an unknown id is distinct from a valid
// id with zero backlinks, spec 030 FR-005/FR-006), builds the Edges+HRefs
// reverse index (buildRelationReverseIndex — NOT Subgraph's Edges-only
// buildReverseIndex), and reports every relation elsewhere in the graph
// whose target is id, as []kernel.BacklinkEntry.
func NodeBacklinks(ctx context.Context, mounter fsys.Mounter, dir, id string) ([]kernel.BacklinkEntry, error) {
	store, err := mounter.Mount(dir)
	if err != nil {
		return nil, err
	}

	if err := guardIsGraph(store, dir); err != nil {
		return nil, err
	}

	index, err := enumerateNodes(store)
	if err != nil {
		return nil, err
	}

	if _, ok := index[id]; !ok {
		return nil, ErrSeedNotFound.With(errNoCause, id)
	}

	rev := buildRelationReverseIndex(index)
	var out []kernel.BacklinkEntry
	for _, e := range rev[id] {
		out = append(out, kernel.BacklinkEntry{Source: e.Source, Predicate: e.Predicate})
	}
	return out, nil
}
