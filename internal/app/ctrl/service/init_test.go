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
	"github.com/fogfish/arcnet-cli/internal/app/ctrl/kernel"
)

// fakeFileInfo/fakeDirEntry carry an explicit dir flag: the FR-015 folder
// collision guard distinguishes an existing FILE from an existing FOLDER
// through fs.FileInfo.IsDir, so a fake that hardcoded false could not
// exercise it at all.
type fakeFileInfo struct {
	name string
	dir  bool
}

func (i fakeFileInfo) Name() string       { return i.name }
func (i fakeFileInfo) Size() int64        { return 0 }
func (i fakeFileInfo) Mode() fs.FileMode  { return 0 }
func (i fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (i fakeFileInfo) IsDir() bool        { return i.dir }
func (i fakeFileInfo) Sys() any           { return nil }

type fakeDirEntry struct {
	name string
	dir  bool
}

func (e fakeDirEntry) Name() string               { return e.name }
func (e fakeDirEntry) IsDir() bool                { return e.dir }
func (e fakeDirEntry) Type() fs.FileMode          { return 0 }
func (e fakeDirEntry) Info() (fs.FileInfo, error) { return fakeFileInfo{name: e.name, dir: e.dir}, nil }

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
	// existing maps a pre-existing path to whether it is a directory. A
	// name given to newFakeStore with a trailing "/" is a directory.
	existing map[string]bool

	// absent marks a root that does not exist at all — the shape
	// ResolveLocalRoot would have to create, where every target guard is
	// skipped because there is nothing yet to inspect.
	absent bool

	written   map[string]bool
	content   map[string][]byte
	removed   []string
	createErr error
}

func newFakeStore(existing ...string) *fakeStore {
	s := &fakeStore{existing: map[string]bool{}, written: map[string]bool{}, content: map[string][]byte{}}
	for _, e := range existing {
		s.existing[strings.TrimSuffix(e, "/")] = strings.HasSuffix(e, "/")
	}
	return s
}

// newAbsentFakeStore is the fixture for a target that does not exist yet.
func newAbsentFakeStore() *fakeStore {
	s := newFakeStore()
	s.absent = true
	return s
}

func (s *fakeStore) Open(name string) (fs.File, error) { return nil, fs.ErrNotExist }

func (s *fakeStore) Stat(name string) (fs.FileInfo, error) {
	if s.absent {
		return nil, fs.ErrNotExist
	}
	if name == "." {
		return fakeFileInfo{name: name, dir: true}, nil
	}
	if dir, ok := s.existing[name]; ok {
		return fakeFileInfo{name: name, dir: dir}, nil
	}
	// a recorded path implies its ancestor directories exist, exactly as a
	// real filesystem reports them.
	for w := range s.existing {
		if strings.HasPrefix(w, name+"/") {
			return fakeFileInfo{name: name, dir: true}, nil
		}
	}
	for w := range s.written {
		if w == name {
			return fakeFileInfo{name: name}, nil
		}
		if strings.HasPrefix(w, name+"/") {
			return fakeFileInfo{name: name, dir: true}, nil
		}
	}
	return nil, fs.ErrNotExist
}

func (s *fakeStore) ReadDir(name string) ([]fs.DirEntry, error) {
	entries := make([]fs.DirEntry, 0, len(s.existing))
	for e, dir := range s.existing {
		entries = append(entries, fakeDirEntry{name: e, dir: dir})
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

	_, err := Init(context.Background(), fakeMounter{store: newFakeStore()}, &mock.VCS{}, "/target", nil, kernel.InitOpts{})

	it.Then(t).Should(it.True(errors.Is(err, fsys.ErrRootNotDirectory)))
}

func TestInitGuardGitUnavailable(t *testing.T) {
	withStubbedResolve(t, false, nil)
	vcs := &mock.VCS{IsAvailableErr: errors.New("no git")}

	_, err := Init(context.Background(), fakeMounter{store: newFakeStore()}, vcs, "/target", nil, kernel.InitOpts{})

	it.Then(t).Should(it.True(errors.Is(err, ErrGitUnavailable)))
}

func TestInitGuardAlreadyInitialized(t *testing.T) {
	withStubbedResolve(t, false, nil)
	store := newFakeStore(".arc")

	_, err := Init(context.Background(), fakeMounter{store: store}, &mock.VCS{}, "/target", nil, kernel.InitOpts{})

	it.Then(t).Should(it.True(errors.Is(err, ErrAlreadyInitialized)))
	it.Then(t).Should(it.Equal(0, len(store.written)))
}

func TestInitGuardTargetNotEmpty(t *testing.T) {
	withStubbedResolve(t, false, nil)
	store := newFakeStore("unrelated.txt")

	_, err := Init(context.Background(), fakeMounter{store: store}, &mock.VCS{}, "/target", nil, kernel.InitOpts{})

	it.Then(t).Should(it.True(errors.Is(err, ErrTargetNotEmpty)))
	it.Then(t).Should(it.Equal(0, len(store.written)))
}

func TestInitSuccessWritesLayoutAndCommits(t *testing.T) {
	withStubbedResolve(t, false, nil)
	store := newFakeStore()
	vcs := &mock.VCS{CommitHash: "abc123"}

	result, err := Init(context.Background(), fakeMounter{store: store}, vcs, "/target", nil, kernel.InitOpts{})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal("abc123", result.CommitHash))
	it.Then(t).Should(it.Equal("/target", result.Root.Root))
	it.Then(t).Should(it.True(store.written["Source/.gitkeep"]))
	it.Then(t).Should(it.True(store.written["_schema/Class/.gitkeep"]))
	it.Then(t).Should(it.True(store.written[".arc/.gitkeep"]))
	// research.md D4: the exclusion rule lives inside .arc/, not at the
	// graph root, in BOTH modes.
	it.Then(t).
		Should(it.True(store.written[".arc/.gitignore"])).
		Should(it.Equal("*\n", string(store.content[".arc/.gitignore"]))).
		Should(it.True(!store.written[".gitignore"]))
	it.Then(t).Should(it.Equal("/target", result.Repository))
	it.Then(t).
		Should(it.Seq(vcs.Calls).Contain("IsAvailable", "Init:/target")).
		// FR-013/FR-014: staged and committed by explicit pathspec, never
		// "everything under the graph root".
		Should(it.True(strings.HasPrefix(vcs.Calls[2], "StagePaths:/target:Source/.gitkeep,"))).
		Should(it.True(strings.HasPrefix(vcs.Calls[3], "CommitPaths:/target:graph(init): empty knowledge graph:Source/.gitkeep,")))
}

func TestInitRollsBackOnCommitFailureWithoutCreatedRoot(t *testing.T) {
	withStubbedResolve(t, false, nil)
	store := newFakeStore()
	vcs := &mock.VCS{CommitErr: errors.New("commit failed")}

	_, err := Init(context.Background(), fakeMounter{store: store}, vcs, "/target", nil, kernel.InitOpts{})

	it.Then(t).ShouldNot(it.Nil(err))
	it.Then(t).Should(it.Seq(store.removed).Contain(
		"Source/.gitkeep", "_schema/Class/.gitkeep", ".arc/.gitkeep", ".arc/.gitignore",
	))
	it.Then(t).ShouldNot(it.Seq(store.removed).Contain(".gitignore"))
}

func TestInitWritesSchemaSeedVerbatim(t *testing.T) {
	withStubbedResolve(t, false, nil)
	store := newFakeStore()
	vcs := &mock.VCS{CommitHash: "abc123"}
	seed := map[string]string{"_schema/Class/source.md": "---\nid: source\nkind: schema\nmerge: none\n---\n# source\n"}

	_, err := Init(context.Background(), fakeMounter{store: store}, vcs, "/target", seed, kernel.InitOpts{})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.True(store.written["_schema/Class/source.md"])).
		Should(it.Equal(seed["_schema/Class/source.md"], string(store.content["_schema/Class/source.md"])))
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

	// The guards no longer reach rollback at all — they all precede root
	// creation now (FR-005) — so the failure has to come from a step that
	// runs after the layout is written.
	vcs := &mock.VCS{CommitErr: errors.New("commit failed")}
	store := newAbsentFakeStore()

	_, err := Init(context.Background(), fakeMounter{store: store}, vcs, "/fresh-target", nil, kernel.InitOpts{})

	it.Then(t).ShouldNot(it.Nil(err))
	it.Then(t).Should(it.Equal("/fresh-target", removedRoot))
}

// ---------------------------------------------------------------------------
// specs/031-init-existing-git-repo — guards, mode branch and footprint.
// ---------------------------------------------------------------------------

// assertNoSideEffects is the shared shape of every refusal in this feature:
// nothing written, nothing removed, and git never asked to do anything at
// all (FR-005).
func assertNoSideEffects(t *testing.T, store *fakeStore, vcs *mock.VCS) {
	t.Helper()
	it.Then(t).
		Should(it.Equal(0, len(store.written))).
		Should(it.Equal(0, len(store.removed))).
		Should(it.Equal(0, len(vcs.Calls)))
}

// data-model.md R1 (FR-004, FR-005, FR-006): a target inside an existing
// repository is refused without the flag, before anything is created — and
// the refusal names both the target and the repository.
func TestInitGuardInsideRepository(t *testing.T) {
	withStubbedResolve(t, false, nil)
	store := newAbsentFakeStore()
	vcs := &mock.VCS{}

	_, err := Init(context.Background(), fakeMounter{store: store}, vcs, "/repo/sub", nil,
		kernel.InitOpts{ParentRepo: "/repo"})

	it.Then(t).Should(it.True(errors.Is(err, ErrInsideRepository)))
	it.Then(t).
		Should(it.String(err.Error()).Contain("/repo")).
		Should(it.String(err.Error()).Contain("--skip-git-init"))
	assertNoSideEffects(t, store, vcs)
}

// data-model.md R2 (FR-012): the flag with no enclosing repository is
// refused rather than silently falling back to creating one.
func TestInitGuardNoParentRepository(t *testing.T) {
	withStubbedResolve(t, false, nil)
	store := newAbsentFakeStore()
	vcs := &mock.VCS{}

	_, err := Init(context.Background(), fakeMounter{store: store}, vcs, "/elsewhere", nil,
		kernel.InitOpts{SkipGitInit: true})

	it.Then(t).Should(it.True(errors.Is(err, ErrNoParentRepository)))
	assertNoSideEffects(t, store, vcs)
}

// data-model.md R3 (FR-020): a target the host project's ignore rules
// exclude is refused upfront, not left to fail later at "nothing to commit".
func TestInitGuardTargetIgnored(t *testing.T) {
	withStubbedResolve(t, false, nil)
	store := newAbsentFakeStore()
	vcs := &mock.VCS{}

	_, err := Init(context.Background(), fakeMounter{store: store}, vcs, "/repo/private/g", nil,
		kernel.InitOpts{ParentRepo: "/repo", SkipGitInit: true, TargetIgnored: true})

	it.Then(t).Should(it.True(errors.Is(err, ErrTargetIgnored)))
	assertNoSideEffects(t, store, vcs)
}

// FR-005: every guard precedes root creation, so a refusal never reaches
// resolveLocalRoot at all — there is nothing left behind to roll back.
func TestInitGuardsRunBeforeRootIsResolved(t *testing.T) {
	resolved := false
	originalResolve := resolveLocalRoot
	originalRemove := removeLocalRoot
	resolveLocalRoot = func(string) (bool, error) { resolved = true; return false, nil }
	removeLocalRoot = func(string) error { return nil }
	t.Cleanup(func() {
		resolveLocalRoot = originalResolve
		removeLocalRoot = originalRemove
	})

	for _, opts := range []kernel.InitOpts{
		{ParentRepo: "/repo"},
		{SkipGitInit: true},
		{ParentRepo: "/repo", SkipGitInit: true, TargetIgnored: true},
	} {
		_, err := Init(context.Background(), fakeMounter{store: newAbsentFakeStore()}, &mock.VCS{}, "/repo/sub", nil, opts)
		it.Then(t).ShouldNot(it.Nil(err))
	}

	// git availability is also a guard, and it too precedes creation.
	_, err := Init(context.Background(), fakeMounter{store: newAbsentFakeStore()},
		&mock.VCS{IsAvailableErr: errors.New("no git")}, "/target", nil, kernel.InitOpts{})
	it.Then(t).Should(it.True(errors.Is(err, ErrGitUnavailable)))

	it.Then(t).Should(it.True(!resolved))
}

// FR-009: --skip-git-init creates no repository of its own; the commit lands
// in the one that already encloses the target.
func TestInitSkipGitInitCreatesNoRepository(t *testing.T) {
	withStubbedResolve(t, false, nil)
	store := newFakeStore()
	vcs := &mock.VCS{CommitHash: "abc123"}

	result, err := Init(context.Background(), fakeMounter{store: store}, vcs, "/repo/notes", nil,
		kernel.InitOpts{ParentRepo: "/repo", SkipGitInit: true})

	it.Then(t).Should(it.Nil(err))
	for _, call := range vcs.Calls {
		it.Then(t).Should(it.True(!strings.HasPrefix(call, "Init:")))
	}
	// FR-027: the repository reported is the enclosing one, an ancestor of
	// the graph root — which is exactly how a consumer tells the two modes
	// apart, with no mode boolean emitted.
	it.Then(t).
		Should(it.Equal("/repo", result.Repository)).
		Should(it.Equal("/repo/notes", result.Root.Root))
}

// FR-010: the emptiness requirement is lifted for --skip-git-init, which
// exists precisely to add a graph beside files the user already has — and
// is unchanged for the standalone path (FR-007).
func TestInitSkipGitInitAllowsNonEmptyTarget(t *testing.T) {
	withStubbedResolve(t, false, nil)
	store := newFakeStore("keep.txt")
	vcs := &mock.VCS{CommitHash: "abc123"}

	_, err := Init(context.Background(), fakeMounter{store: store}, vcs, "/repo/notes", nil,
		kernel.InitOpts{ParentRepo: "/repo", SkipGitInit: true})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.True(store.written["Source/.gitkeep"]))
}

// FR-015 / clarification 2: an existing canonical FOLDER collides whether
// or not it holds anything, and the refusal names it and the recovery.
func TestInitGuardFolderCollision(t *testing.T) {
	for _, existing := range []string{"Source/", "timeline/monthly/"} {
		withStubbedResolve(t, false, nil)
		store := newFakeStore(existing)
		vcs := &mock.VCS{}

		_, err := Init(context.Background(), fakeMounter{store: store}, vcs, "/repo", nil,
			kernel.InitOpts{ParentRepo: "/repo", SkipGitInit: true})

		it.Then(t).Should(it.True(errors.Is(err, ErrLayoutCollision)))
		it.Then(t).
			Should(it.String(err.Error()).Contain(strings.TrimSuffix(existing, "/"))).
			Should(it.String(err.Error()).Contain("subfolder"))
		assertNoSideEffects(t, store, vcs)
	}
}

// FR-015: a canonical name taken by a FILE rather than a folder collides
// just the same — the layout never overwrites what the user owns.
func TestInitGuardFileCollision(t *testing.T) {
	withStubbedResolve(t, false, nil)
	store := newFakeStore("Source")
	vcs := &mock.VCS{}

	_, err := Init(context.Background(), fakeMounter{store: store}, vcs, "/repo", nil,
		kernel.InitOpts{ParentRepo: "/repo", SkipGitInit: true})

	it.Then(t).Should(it.True(errors.Is(err, ErrLayoutCollision)))
	it.Then(t).Should(it.String(err.Error()).Contain("Source"))
	assertNoSideEffects(t, store, vcs)

	// and a seeded schema document already present collides too, not just
	// the statically-known layout files.
	store = newFakeStore("_schema/Class/Entity.md")
	_, err = Init(context.Background(), fakeMounter{store: store}, &mock.VCS{}, "/repo",
		map[string]string{"_schema/Class/Entity.md": "seeded"},
		kernel.InitOpts{ParentRepo: "/repo", SkipGitInit: true})
	it.Then(t).Should(it.True(errors.Is(err, ErrLayoutCollision)))
}

// FR-018, FR-019: rollback removes exactly this run's footprint — every
// file it wrote and every directory it had to create — and nothing that
// pre-existed, in a target the tool does not exclusively own.
func TestInitRollbackIsScopedToTheFootprint(t *testing.T) {
	withStubbedResolve(t, false, nil)
	store := newFakeStore("keep.txt", "notes/")
	vcs := &mock.VCS{CommitErr: errors.New("commit failed")}

	_, err := Init(context.Background(), fakeMounter{store: store}, vcs, "/repo/g", nil,
		kernel.InitOpts{ParentRepo: "/repo", SkipGitInit: true})

	it.Then(t).ShouldNot(it.Nil(err))
	it.Then(t).Should(it.Seq(store.removed).Contain(
		"Source/.gitkeep", ".arc/.gitkeep", ".arc/.gitignore",
		// the directories this run created, removed deepest-first
		"_schema/Class", "_schema", ".arc",
	))
	for _, removed := range store.removed {
		it.Then(t).
			Should(it.True(removed != "keep.txt")).
			Should(it.True(removed != "notes"))
	}
}
