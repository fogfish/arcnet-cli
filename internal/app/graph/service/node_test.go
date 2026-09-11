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
	"io/fs"
	"syscall"
	"testing"
	"testing/fstest"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/graph/service"
)

func TestNodeGetReturnsMatchingNodeFullContent(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/TLS.md": entityNode("TLS", "SSL"),
	})

	node, err := service.NodeGet(context.Background(), mounter, "/graph", "TLS")

	it.Then(t).
		Should(it.Nil(err)).
		Should(it.Equal("TLS", node.ID))
}

func TestNodeGetUnknownIDReturnsErrSeedNotFound(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/TLS.md": entityNode("TLS"),
	})

	_, err := service.NodeGet(context.Background(), mounter, "/graph", "No Such Node")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrSeedNotFound)))
}

func TestNodeGetNotAGraphReturnsErrNotAGraphBeforeLookup(t *testing.T) {
	mounter := grepMounter{store: grepStore{fstest.MapFS{}}}

	_, err := service.NodeGet(context.Background(), mounter, "/graph", "TLS")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrNotAGraph)))
}

func TestEnsureGraphValidGraphReturnsNil(t *testing.T) {
	mounter := newGrepGraph(map[string]string{
		"Entity/TLS.md": entityNode("TLS"),
	})

	err := service.EnsureGraph(context.Background(), mounter, "/graph")

	it.Then(t).Should(it.Nil(err))
}

func TestEnsureGraphNotAGraphReturnsErrNotAGraph(t *testing.T) {
	mounter := grepMounter{store: grepStore{fstest.MapFS{}}}

	err := service.EnsureGraph(context.Background(), mounter, "/graph")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrNotAGraph)))
}

// failingStore is an fsys.Store whose Stat always fails with statErr — the
// one thing EnsureGraph's root classification reads (specs/034-serve-dir-
// public-state, research D2). fstest.MapFS cannot express "this root is
// absent/unreadable/not a directory", because it synthesizes "." as a
// directory unconditionally.
type failingStore struct {
	fstest.MapFS
	statErr error
}

func (s failingStore) Stat(name string) (fs.FileInfo, error) { return nil, s.statErr }
func (failingStore) Create(name string) (fsys.File, error)   { return nil, errors.New("read-only store") }
func (failingStore) Remove(name string) error                { return errors.New("read-only store") }

type failingMounter struct{ store failingStore }

func (m failingMounter) Mount(root string) (fsys.Store, error) { return m.store, nil }

func mounterFailingWith(err error) failingMounter {
	return failingMounter{store: failingStore{MapFS: fstest.MapFS{}, statErr: err}}
}

// specs/034-serve-dir-public-state FR-004, research D2: a root that does
// not exist is named for that, not reported as "not an initialized graph".
func TestEnsureGraphMissingRootReturnsErrGraphDirNotFound(t *testing.T) {
	err := service.EnsureGraph(context.Background(), mounterFailingWith(fs.ErrNotExist), "/graph")

	it.Then(t).
		Should(it.True(errors.Is(err, service.ErrGraphDirNotFound))).
		Should(it.True(!errors.Is(err, service.ErrNotAGraph))).
		Should(it.String(err.Error()).Contain("/graph does not exist"))
}

// specs/034-serve-dir-public-state FR-004, research D2: an unreadable root
// gets a permission-specific message.
func TestEnsureGraphUnreadableRootReturnsErrGraphDirUnreadable(t *testing.T) {
	err := service.EnsureGraph(context.Background(), mounterFailingWith(fs.ErrPermission), "/graph")

	it.Then(t).
		Should(it.True(errors.Is(err, service.ErrGraphDirUnreadable))).
		Should(it.True(!errors.Is(err, service.ErrNotAGraph))).
		Should(it.String(err.Error()).Contain("/graph cannot be read"))
}

// specs/034-serve-dir-public-state FR-004, research D2: anything else —
// ENOTDIR foremost, which is what os.DirFS answers for a path that is a
// file — is reported as "not a directory".
func TestEnsureGraphFileRootReturnsErrGraphDirNotDirectory(t *testing.T) {
	err := service.EnsureGraph(context.Background(), mounterFailingWith(syscall.ENOTDIR), "/graph")

	it.Then(t).
		Should(it.True(errors.Is(err, service.ErrGraphDirNotDirectory))).
		Should(it.True(!errors.Is(err, service.ErrNotAGraph))).
		Should(it.String(err.Error()).Contain("/graph is not a directory"))
}

// specs/034-serve-dir-public-state: the unchanged path — a root that
// exists, is readable, and simply has no .arc/ — still answers
// ErrNotAGraph, and none of the three new classes.
func TestEnsureGraphReadableRootWithoutArcStillReturnsErrNotAGraph(t *testing.T) {
	mounter := grepMounter{store: grepStore{fstest.MapFS{"README.md": &fstest.MapFile{}}}}

	err := service.EnsureGraph(context.Background(), mounter, "/graph")

	it.Then(t).
		Should(it.True(errors.Is(err, service.ErrNotAGraph))).
		Should(it.True(!errors.Is(err, service.ErrGraphDirNotFound))).
		Should(it.True(!errors.Is(err, service.ErrGraphDirUnreadable))).
		Should(it.True(!errors.Is(err, service.ErrGraphDirNotDirectory)))
}
