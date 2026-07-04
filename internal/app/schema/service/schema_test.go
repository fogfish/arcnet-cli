//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service_test

import (
	"bytes"
	"errors"
	"io/fs"
	"testing"
	"time"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/schema/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/schema/service"
	"github.com/fogfish/arcnet-cli/internal/core"
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

type fakeOpenFile struct{ *bytes.Reader }

func (f fakeOpenFile) Close() error               { return nil }
func (f fakeOpenFile) Stat() (fs.FileInfo, error) { return fakeFileInfo{}, nil }

type fakeWriteFile struct {
	name string
	buf  *bytes.Buffer
	on   func(name string, content []byte)
}

func (f *fakeWriteFile) Write(p []byte) (int, error) { return f.buf.Write(p) }
func (f *fakeWriteFile) Close() error {
	f.on(f.name, f.buf.Bytes())
	return nil
}
func (f *fakeWriteFile) Stat() (fs.FileInfo, error) { return fakeFileInfo{name: f.name}, nil }
func (f *fakeWriteFile) Discard() error             { return nil }

type fakeStore struct {
	files     map[string]string
	written   map[string]string
	createErr error
}

func newFakeStore(files map[string]string) *fakeStore {
	if files == nil {
		files = map[string]string{}
	}
	return &fakeStore{files: files, written: map[string]string{}}
}

func (s *fakeStore) Open(name string) (fs.File, error) {
	content, ok := s.files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return fakeOpenFile{bytes.NewReader([]byte(content))}, nil
}

func (s *fakeStore) Stat(name string) (fs.FileInfo, error) {
	if _, ok := s.files[name]; ok {
		return fakeFileInfo{name: name}, nil
	}
	return nil, fs.ErrNotExist
}

func (s *fakeStore) ReadDir(name string) ([]fs.DirEntry, error) {
	prefix := name + "/"
	seen := map[string]bool{}
	var out []fs.DirEntry
	for path := range s.files {
		if len(path) <= len(prefix) || path[:len(prefix)] != prefix {
			continue
		}
		rest := path[len(prefix):]
		if seen[rest] {
			continue
		}
		seen[rest] = true
		out = append(out, fakeDirEntry{name: rest})
	}
	if len(out) == 0 {
		return nil, fs.ErrNotExist
	}
	return out, nil
}

func (s *fakeStore) Create(name string) (fsys.File, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	return &fakeWriteFile{name: name, buf: &bytes.Buffer{}, on: func(n string, c []byte) {
		s.written[n] = string(c)
		s.files[n] = string(c)
	}}, nil
}

func (s *fakeStore) Remove(name string) error { return nil }

func TestSeedReturnsSeventeenEntries(t *testing.T) {
	seed := service.Seed()
	it.Then(t).Should(it.Equal(17, len(seed)))
}

func TestSeedEntriesRoundTripThroughParseNode(t *testing.T) {
	seed := service.Seed()

	for path, raw := range seed {
		node, err := core.ParseNode(bytes.NewReader(raw))
		it.Then(t).Should(it.Nil(err))
		it.Then(t).Should(it.Equal(kernel.SchemaKind, node.Kind))

		if op, ok := kernel.CoreMergeRules.Lookup(core.Kind(node.ID)); ok {
			it.Then(t).Should(it.Equal(kernel.NodesDir+"/"+node.ID+".md", path))
			merge, _ := node.Attrs["merge"].(string)
			it.Then(t).Should(it.Equal(string(op), merge))
		}
	}
}

func TestResolveRoundTripsSeedOutput(t *testing.T) {
	store := newFakeStore(nil)
	for path, raw := range service.Seed() {
		f, err := store.Create(path)
		it.Then(t).Should(it.Nil(err))
		_, err = f.Write(raw)
		it.Then(t).Should(it.Nil(err))
		it.Then(t).Should(it.Nil(f.Close()))
	}

	rules, predicates, err := service.Resolve(store)

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(len(kernel.CoreMergeRules), len(rules)))
	it.Then(t).Should(it.Equal(len(kernel.CorePredicates), len(predicates)))

	op, ok := rules.Lookup("entity")
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal(core.MergeUnion, op))
}

func TestResolveSkipsMalformedDocument(t *testing.T) {
	store := newFakeStore(map[string]string{
		kernel.NodesDir + "/source.md": "---\nid: source\nkind: schema\nmerge: none\n---\n# source\n",
		kernel.NodesDir + "/broken.md": "not valid front matter at all",
	})

	rules, _, err := service.Resolve(store)

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(rules)))
	_, ok := rules.Lookup("broken")
	it.Then(t).Should(it.True(!ok))
}

func TestResolveAbsentFolderReturnsEmptyResults(t *testing.T) {
	store := newFakeStore(nil)

	rules, predicates, err := service.Resolve(store)

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(rules))).
		Should(it.Equal(0, len(predicates)))
}

func TestRegisterKindCreatesFileOnce(t *testing.T) {
	store := newFakeStore(nil)

	created, err := service.RegisterKind(store, "hypothesis")
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.True(created))

	content := store.written[kernel.NodesDir+"/hypothesis.md"]
	it.Then(t).Should(it.String(content).Contain("merge: union"))

	created, err = service.RegisterKind(store, "hypothesis")
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.True(!created))
	it.Then(t).Should(it.Equal(1, len(store.written)))
}

func TestRegisterPredicateCreatesFileOnce(t *testing.T) {
	store := newFakeStore(nil)

	created, err := service.RegisterPredicate(store, "relatesTo")
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.True(created))

	created, err = service.RegisterPredicate(store, "relatesTo")
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.True(!created))
	it.Then(t).Should(it.Equal(1, len(store.written)))
}

func TestRegisterKindWriteFailure(t *testing.T) {
	store := newFakeStore(nil)
	store.createErr = errors.New("disk full")

	_, err := service.RegisterKind(store, "hypothesis")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrSchemaWrite)))
}

func TestRegisterPredicateWriteFailure(t *testing.T) {
	store := newFakeStore(nil)
	store.createErr = errors.New("disk full")

	_, err := service.RegisterPredicate(store, "relatesTo")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrSchemaWrite)))
}

func TestResolveReflectsHandEditedMergeValue(t *testing.T) {
	store := newFakeStore(map[string]string{
		kernel.NodesDir + "/hypothesis.md": "---\nid: hypothesis\nkind: schema\nmerge: union\n---\n# hypothesis\n",
	})

	rules, _, err := service.Resolve(store)
	it.Then(t).Should(it.Nil(err))
	op, _ := rules.Lookup("hypothesis")
	it.Then(t).Should(it.Equal(core.MergeUnion, op))

	store.files[kernel.NodesDir+"/hypothesis.md"] = "---\nid: hypothesis\nkind: schema\nmerge: union-first-writer\n---\n# hypothesis\n"

	rules, _, err = service.Resolve(store)
	it.Then(t).Should(it.Nil(err))
	op, _ = rules.Lookup("hypothesis")
	it.Then(t).Should(it.Equal(core.MergeUnionFirstWriter, op))
}
