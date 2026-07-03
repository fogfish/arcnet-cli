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
	"strings"
	"testing"
	"time"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/ctrl/adapter/mock"
)

type fakeFileInfo struct{ name string }

func (i fakeFileInfo) Name() string       { return i.name }
func (i fakeFileInfo) Size() int64        { return 0 }
func (i fakeFileInfo) Mode() fs.FileMode  { return 0 }
func (i fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (i fakeFileInfo) IsDir() bool        { return false }
func (i fakeFileInfo) Sys() any           { return nil }

type fakeDirEntry struct{ name string }

func (e fakeDirEntry) Name() string               { return e.name }
func (e fakeDirEntry) IsDir() bool                { return false }
func (e fakeDirEntry) Type() fs.FileMode          { return 0 }
func (e fakeDirEntry) Info() (fs.FileInfo, error) { return fakeFileInfo(e), nil }

type fakeFile struct {
	name    string
	store   *fakeStore
	content []byte
}

func (f *fakeFile) Write(p []byte) (int, error) {
	f.content = append(f.content, p...)
	f.store.content[f.name] = f.content
	return len(p), nil
}
func (f *fakeFile) Close() error               { return nil }
func (f *fakeFile) Stat() (fs.FileInfo, error) { return fakeFileInfo{name: f.name}, nil }
func (f *fakeFile) Discard() error             { return nil }

type fakeStore struct {
	existing  map[string]bool
	written   map[string]bool
	content   map[string][]byte
	removed   []string
	createErr error
}

func newFakeStore(existing ...string) *fakeStore {
	s := &fakeStore{existing: map[string]bool{}, written: map[string]bool{}, content: map[string][]byte{}}
	for _, e := range existing {
		s.existing[e] = true
	}
	return s
}

func (s *fakeStore) Open(name string) (fs.File, error) { return nil, fs.ErrNotExist }

func (s *fakeStore) Stat(name string) (fs.FileInfo, error) {
	if s.existing[name] {
		return fakeFileInfo{name: name}, nil
	}
	for w := range s.written {
		if w == name || strings.HasPrefix(w, name+"/") {
			return fakeFileInfo{name: name}, nil
		}
	}
	return nil, fs.ErrNotExist
}

func (s *fakeStore) ReadDir(name string) ([]fs.DirEntry, error) {
	entries := make([]fs.DirEntry, 0, len(s.existing))
	for e := range s.existing {
		entries = append(entries, fakeDirEntry{name: e})
	}
	return entries, nil
}

func (s *fakeStore) Create(name string) (fsys.File, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	s.written[name] = true
	return &fakeFile{name: name, store: s}, nil
}

func (s *fakeStore) Remove(name string) error {
	s.removed = append(s.removed, name)
	delete(s.written, name)
	return nil
}

type fakeMounter struct {
	store    *fakeStore
	mountErr error
}

func (m fakeMounter) Mount(root string) (fsys.Store, error) {
	if m.mountErr != nil {
		return nil, m.mountErr
	}
	return m.store, nil
}

func withStubbedResolve(t *testing.T, created bool, err error) {
	t.Helper()
	originalResolve := resolveLocalRoot
	originalRemove := removeLocalRoot
	resolveLocalRoot = func(string) (bool, error) { return created, err }
	removeLocalRoot = func(string) error { return nil }
	t.Cleanup(func() {
		resolveLocalRoot = originalResolve
		removeLocalRoot = originalRemove
	})
}

func TestInitGuardTargetIsFile(t *testing.T) {
	withStubbedResolve(t, false, fsys.ErrRootNotDirectory.With(nil, "/target"))

	_, err := Init(context.Background(), fakeMounter{store: newFakeStore()}, &mock.VCS{}, "/target", nil)

	it.Then(t).Should(it.True(errors.Is(err, fsys.ErrRootNotDirectory)))
}

func TestInitGuardGitUnavailable(t *testing.T) {
	withStubbedResolve(t, false, nil)
	vcs := &mock.VCS{IsAvailableErr: errors.New("no git")}

	_, err := Init(context.Background(), fakeMounter{store: newFakeStore()}, vcs, "/target", nil)

	it.Then(t).Should(it.True(errors.Is(err, ErrGitUnavailable)))
}

func TestInitGuardAlreadyInitialized(t *testing.T) {
	withStubbedResolve(t, false, nil)
	store := newFakeStore(".arc")

	_, err := Init(context.Background(), fakeMounter{store: store}, &mock.VCS{}, "/target", nil)

	it.Then(t).Should(it.True(errors.Is(err, ErrAlreadyInitialized)))
	it.Then(t).Should(it.Equal(0, len(store.written)))
}

func TestInitGuardTargetNotEmpty(t *testing.T) {
	withStubbedResolve(t, false, nil)
	store := newFakeStore("unrelated.txt")

	_, err := Init(context.Background(), fakeMounter{store: store}, &mock.VCS{}, "/target", nil)

	it.Then(t).Should(it.True(errors.Is(err, ErrTargetNotEmpty)))
	it.Then(t).Should(it.Equal(0, len(store.written)))
}

func TestInitSuccessWritesLayoutAndCommits(t *testing.T) {
	withStubbedResolve(t, false, nil)
	store := newFakeStore()
	vcs := &mock.VCS{CommitHash: "abc123"}

	result, err := Init(context.Background(), fakeMounter{store: store}, vcs, "/target", nil)

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal("abc123", result.CommitHash))
	it.Then(t).Should(it.Equal("/target", result.Root.Root))
	it.Then(t).Should(it.True(store.written["sources/.gitkeep"]))
	it.Then(t).Should(it.True(store.written["_meta/predicates.md"]))
	it.Then(t).Should(it.True(store.written[".arc/.gitkeep"]))
	it.Then(t).Should(it.True(store.written[".gitignore"]))
	it.Then(t).
		Should(it.Seq(vcs.Calls).Contain(
			"IsAvailable", "Init:/target", "StageAll:/target",
			"Commit:/target:graph(init): empty knowledge graph",
		))
}

func TestInitRollsBackOnCommitFailureWithoutCreatedRoot(t *testing.T) {
	withStubbedResolve(t, false, nil)
	store := newFakeStore()
	vcs := &mock.VCS{CommitErr: errors.New("commit failed")}

	_, err := Init(context.Background(), fakeMounter{store: store}, vcs, "/target", nil)

	it.Then(t).ShouldNot(it.Nil(err))
	it.Then(t).Should(it.Seq(store.removed).Contain(
		"sources/.gitkeep", "_meta/predicates.md", ".arc/.gitkeep", ".gitignore", ".arc/config.yml",
	))
}

func TestInitWritesConfigSeedVerbatim(t *testing.T) {
	withStubbedResolve(t, false, nil)
	store := newFakeStore()
	vcs := &mock.VCS{CommitHash: "abc123"}
	seed := []byte("mergeRules:\n  source: none\n")

	_, err := Init(context.Background(), fakeMounter{store: store}, vcs, "/target", seed)

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.True(store.written[".arc/config.yml"])).
		Should(it.Equal(string(seed), string(store.content[".arc/config.yml"])))
}

func TestInitRollsBackViaRemoveLocalRootWhenCreated(t *testing.T) {
	var removedRoot string
	originalResolve := resolveLocalRoot
	originalRemove := removeLocalRoot
	resolveLocalRoot = func(string) (bool, error) { return true, nil }
	removeLocalRoot = func(dir string) error { removedRoot = dir; return nil }
	t.Cleanup(func() {
		resolveLocalRoot = originalResolve
		removeLocalRoot = originalRemove
	})

	vcs := &mock.VCS{IsAvailableErr: errors.New("no git")}
	store := newFakeStore()

	_, err := Init(context.Background(), fakeMounter{store: store}, vcs, "/fresh-target", nil)

	it.Then(t).ShouldNot(it.Nil(err))
	it.Then(t).Should(it.Equal("/fresh-target", removedRoot))
}
