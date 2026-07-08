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
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	lintmock "github.com/fogfish/arcnet-cli/internal/app/lint/adapter/mock"
	"github.com/fogfish/arcnet-cli/internal/app/lint/service"
	"github.com/fogfish/arcnet-cli/internal/bios"
	"github.com/fogfish/arcnet-cli/internal/core"
)

type memFileInfo struct {
	name  string
	isDir bool
}

func (i memFileInfo) Name() string       { return i.name }
func (i memFileInfo) Size() int64        { return 0 }
func (i memFileInfo) Mode() fs.FileMode  { return 0 }
func (i memFileInfo) ModTime() time.Time { return time.Time{} }
func (i memFileInfo) IsDir() bool        { return i.isDir }
func (i memFileInfo) Sys() any           { return nil }

func (i memFileInfo) Type() fs.FileMode {
	if i.isDir {
		return fs.ModeDir
	}
	return 0
}
func (i memFileInfo) Info() (fs.FileInfo, error) { return i, nil }

type memOpenFile struct {
	*strings.Reader
	name string
}

func (f memOpenFile) Close() error               { return nil }
func (f memOpenFile) Stat() (fs.FileInfo, error) { return memFileInfo{name: f.name}, nil }

// memStore is a minimal, read-only fs.FS-backed fake of fsys.Store, for
// internal/app/lint/service unit tests — lint never calls Create/Remove, so
// both are stubbed to fail loudly if ever invoked.
type memStore struct {
	files    map[string]string
	dirs     map[string]bool
	openErrs map[string]error
}

func newMemStore() *memStore {
	return &memStore{files: map[string]string{}, dirs: map[string]bool{".arc": true}, openErrs: map[string]error{}}
}

func (s *memStore) Open(name string) (fs.File, error) {
	if err, ok := s.openErrs[name]; ok {
		return nil, err
	}
	content, ok := s.files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return memOpenFile{strings.NewReader(content), name}, nil
}

func (s *memStore) Stat(name string) (fs.FileInfo, error) {
	if s.dirs[name] {
		return memFileInfo{name: name, isDir: true}, nil
	}
	if _, ok := s.files[name]; ok {
		return memFileInfo{name: name}, nil
	}
	return nil, fs.ErrNotExist
}

func (s *memStore) ReadDir(dir string) ([]fs.DirEntry, error) {
	prefix := dir + "/"
	if dir == "." {
		prefix = ""
	}

	seen := map[string]bool{}
	var entries []fs.DirEntry

	add := func(name string, isDir bool) {
		if seen[name] {
			return
		}
		seen[name] = true
		entries = append(entries, memFileInfo{name: name, isDir: isDir})
	}

	for path := range s.files {
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		rest := strings.TrimPrefix(path, prefix)
		parts := strings.SplitN(rest, "/", 2)
		add(parts[0], len(parts) > 1)
	}
	for d := range s.dirs {
		if !strings.HasPrefix(d, prefix) {
			continue
		}
		rest := strings.TrimPrefix(d, prefix)
		if rest == "" {
			continue
		}
		parts := strings.SplitN(rest, "/", 2)
		add(parts[0], true)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

func (s *memStore) Create(name string) (fsys.File, error) { return nil, errors.New("read-only fake") }
func (s *memStore) Remove(name string) error              { return errors.New("read-only fake") }

type memMounter struct{ store *memStore }

func (m memMounter) Mount(root string) (fsys.Store, error) { return m.store, nil }

const conformantSourceFixture = `---
"@id": "foo-2026-x"
"@type": source
title: "A Test Document"
authors: [Test Author]
published: "2026-04-12"
---
# foo-2026-x

A test document.
- mentions:: [[Widget]]
`

const conformantEntityFixture = `---
"@id": "Widget"
"@type": entity
title: Widget
category: [independent, abstract, occurrent, script]
---
# Widget

A test entity.
- mentions:: [[foo-2026-x]]
`

var coreIndexFixtureLint = core.Index{
	Types: map[string]core.TypeDef{
		"source":   {Merge: core.MergeNone},
		"entity":   {Merge: core.MergeUnion},
		"resource": {Merge: core.MergeUnionFirstWriter},
		"timeline": {Merge: core.MergeAppend},
	},
	Predicates: map[string]core.PredicateDef{"mentions": {}},
}

func newConformantStore() *memStore {
	s := newMemStore()
	s.files["sources/foo-2026-x.md"] = conformantSourceFixture
	s.files["entities/Widget.md"] = conformantEntityFixture
	return s
}

func TestLintGuardNotAGraph(t *testing.T) {
	s := newMemStore()
	delete(s.dirs, ".arc")

	_, err := service.Lint(context.Background(), memMounter{s}, &lintmock.VCS{}, bios.NewReporter(true, true), coreIndexFixtureLint, "/graph")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrNotAGraph)))
}

func TestLintConformantGraphAllPass(t *testing.T) {
	s := newConformantStore()
	vcs := &lintmock.VCS{Commits: map[string][]string{"Source-Id: foo-2026-x": {"abc123"}}}

	result, err := service.Lint(context.Background(), memMounter{s}, vcs, bios.NewReporter(true, true), coreIndexFixtureLint, "/graph")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(2, len(result.Nodes))).
		Should(it.Equal(2, result.Passing)).
		Should(it.Equal(0, result.Failing)).
		Should(it.Equal(0, len(result.Violations)))
}

func TestLintExcludesArcAndSchema(t *testing.T) {
	s := newConformantStore()
	s.files[".arc/config.yml"] = ""
	s.files["_schema/types/entity.md"] = "---\n\"@id\": entity\n\"@type\": Class\nmerge: union\n---\n# entity\n\nA node for a subject occurring in sources.\n"
	vcs := &lintmock.VCS{Commits: map[string][]string{"Source-Id: foo-2026-x": {"abc123"}}}

	result, err := service.Lint(context.Background(), memMounter{s}, vcs, bios.NewReporter(true, true), coreIndexFixtureLint, "/graph")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(2, len(result.Nodes)))
}

func TestLintIncludesNodeInNonStandardFolder(t *testing.T) {
	s := newConformantStore()
	s.files["hypothesis/A Test Hypothesis.md"] = "---\n\"@id\": \"A Test Hypothesis\"\n\"@type\": hypothesis\ntitle: A Test Hypothesis\n---\n# A Test Hypothesis\n\nA conclusion.\n- mentions:: [[foo-2026-x]]\n"
	index := core.Index{
		Types: map[string]core.TypeDef{
			"source":     {Merge: core.MergeNone},
			"entity":     {Merge: core.MergeUnion},
			"resource":   {Merge: core.MergeUnionFirstWriter},
			"timeline":   {Merge: core.MergeAppend},
			"hypothesis": {Merge: core.MergeValidatedOverwrite},
		},
		Predicates: coreIndexFixtureLint.Predicates,
	}
	vcs := &lintmock.VCS{Commits: map[string][]string{"Source-Id: foo-2026-x": {"abc123"}}}

	result, err := service.Lint(context.Background(), memMounter{s}, vcs, bios.NewReporter(true, true), index, "/graph")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(3, len(result.Nodes)))
}

func TestLintRecordsFrontMatterViolationWithoutAbortingWalk(t *testing.T) {
	s := newConformantStore()
	s.files["entities/Broken.md"] = "not valid front matter at all\n"
	vcs := &lintmock.VCS{Commits: map[string][]string{"Source-Id: foo-2026-x": {"abc123"}}}

	result, err := service.Lint(context.Background(), memMounter{s}, vcs, bios.NewReporter(true, true), coreIndexFixtureLint, "/graph")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(3, len(result.Nodes)))

	found := false
	for _, n := range result.Nodes {
		if n.Path == "entities/Broken.md" {
			found = true
			it.Then(t).Should(it.Equal(1, len(n.Violations)))
		}
	}
	it.Then(t).Should(it.True(found))
}

func TestLintMergeConflictExcludesFileFromOtherChecks(t *testing.T) {
	s := newConformantStore()
	s.files["entities/Broken.md"] = "<<<<<<< HEAD\nkind: entity\n=======\nkind: entity\n>>>>>>> feature\n"
	vcs := &lintmock.VCS{Commits: map[string][]string{"Source-Id: foo-2026-x": {"abc123"}}}

	result, err := service.Lint(context.Background(), memMounter{s}, vcs, bios.NewReporter(true, true), coreIndexFixtureLint, "/graph")

	it.Then(t).Should(it.Nil(err))
	for _, n := range result.Nodes {
		if n.Path == "entities/Broken.md" {
			it.Then(t).Should(it.Equal(1, len(n.Violations)))
			it.Then(t).Should(it.Equal("mergeConflict", string(n.Violations[0].Rule)))
		}
	}
}
