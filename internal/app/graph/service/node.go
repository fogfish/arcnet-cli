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
	"errors"
	"io/fs"

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
//
// The root is classified before guardIsGraph is consulted, so a path that
// does not exist, is a file, or cannot be read is named for what it is
// rather than as "not an initialized graph" (specs/034-serve-dir-public-
// state, research D2). arc stats and arc apply reach this through
// filepath.Abs("."), which by construction exists and is a directory, so
// the new branches are unreachable for them.
func EnsureGraph(ctx context.Context, mounter fsys.Mounter, dir string) error {
	store, err := mounter.Mount(dir)
	if err != nil {
		return err
	}

	if err := guardGraphRoot(store, dir); err != nil {
		return err
	}

	return guardIsGraph(store, dir)
}

// guardGraphRoot answers whether dir is usable as a graph root at all, by
// the one question fsys.Store already answers truthfully about it
// (research D2). fsys.ResolveLocalRoot is deliberately NOT used here: it
// CREATES the directory when absent, which would turn a typo'd path handed
// to a read-only command into a new empty directory.
func guardGraphRoot(store fsys.Store, dir string) error {
	_, err := store.Stat(".")
	switch {
	case err == nil:
		return nil
	case errors.Is(err, fs.ErrNotExist):
		return ErrGraphDirNotFound.With(err, dir)
	case errors.Is(err, fs.ErrPermission):
		return ErrGraphDirUnreadable.With(err, dir)
	default:
		// ENOTDIR and anything else: the path resolves to something, but
		// not to a directory a graph could live in.
		return ErrGraphDirNotDirectory.With(err, dir)
	}
}
