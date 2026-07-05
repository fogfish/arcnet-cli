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
	"github.com/fogfish/arcnet-cli/internal/core"
)

// NodeGet mounts dir, enumerates and indexes every node (reusing
// enumerateNodes/guardIsGraph, unchanged from Subgraph's own precedent), and
// looks up id in that index, returning ErrSeedNotFound on a miss (research.md
// D3 in specs/008-arc-serve-mcp).
func NodeGet(ctx context.Context, mounter fsys.Mounter, dir, id string) (core.Node, error) {
	store, err := mounter.Mount(dir)
	if err != nil {
		return core.Node{}, err
	}

	if err := guardIsGraph(store, dir); err != nil {
		return core.Node{}, err
	}

	index, err := enumerateNodes(store)
	if err != nil {
		return core.Node{}, err
	}

	node, ok := index[id]
	if !ok {
		return core.Node{}, ErrSeedNotFound.With(errNoCause, id)
	}
	return node, nil
}

// EnsureGraph mounts dir and confirms it is an initialized graph, without
// reading or parsing any node — the preflight arc serve's RunE calls before
// starting any transport (spec FR-004, research.md D3/D6).
func EnsureGraph(ctx context.Context, mounter fsys.Mounter, dir string) error {
	store, err := mounter.Mount(dir)
	if err != nil {
		return err
	}
	return guardIsGraph(store, dir)
}
