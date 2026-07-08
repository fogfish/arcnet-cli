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
	dirs      map[string]bool
	written   map[string]string
	createErr error
}

func newFakeStore(files map[string]string) *fakeStore {
	if files == nil {
		files = map[string]string{}
	}
	return &fakeStore{files: files, dirs: map[string]bool{".arc": true}, written: map[string]string{}}
}

func (s *fakeStore) Open(name string) (fs.File, error) {
	content, ok := s.files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return fakeOpenFile{bytes.NewReader([]byte(content))}, nil
}

func (s *fakeStore) Stat(name string) (fs.FileInfo, error) {
	if s.dirs[name] {
		return fakeFileInfo{name: name}, nil
	}
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
	if len(out) == 0 && !s.dirs[name] {
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

// newSeededStore builds a fake graph root whose _schema/ folders already
// carry every built-in document from service.Seed(), the baseline every
// Resolve-focused test case starts from.
func newSeededStore() *fakeStore {
	store := newFakeStore(nil)
	store.dirs[kernel.PredicatesDir] = true
	store.dirs[kernel.TypesDir] = true
	for path, raw := range service.Seed() {
		f, _ := store.Create(path)
		_, _ = f.Write(raw)
		_ = f.Close()
	}
	return store
}

func TestSeedReturnsOneEntryPerPredicateAndType(t *testing.T) {
	seed := service.Seed()
	it.Then(t).Should(it.Equal(len(kernel.CorePredicateDefs)+len(kernel.CoreTypeDefs), len(seed)))
}

func TestSeedEntriesRoundTripThroughParseNode(t *testing.T) {
	seed := service.Seed()

	for path, raw := range seed {
		node, err := core.ParseNode(bytes.NewReader(raw))
		it.Then(t).Should(it.Nil(err))

		if def, ok := kernel.CorePredicateDefs[node.ID]; ok {
			it.Then(t).
				Should(it.Equal(kernel.PredicatesDir+"/"+node.ID+".md", path)).
				Should(it.Equal("Property", node.Type))
			role, _ := node.Attrs["role"][0].Value.(string)
			it.Then(t).Should(it.Equal(def.Role, role))
			continue
		}

		if def, ok := kernel.CoreTypeDefs[node.ID]; ok {
			it.Then(t).
				Should(it.Equal(kernel.TypesDir+"/"+node.ID+".md", path)).
				Should(it.Equal("Class", node.Type))
			merge, _ := node.Attrs["merge"][0].Value.(string)
			it.Then(t).Should(it.Equal(string(def.Merge), merge))
		}
	}
}

func TestResolveRoundTripsSeedOutput(t *testing.T) {
	store := newSeededStore()

	index, err := service.Resolve(store)

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(len(kernel.CorePredicateDefs), len(index.Predicates))).
		Should(it.Equal(len(kernel.CoreTypeDefs), len(index.Types)))

	entity, ok := index.Types["entity"]
	it.Then(t).Should(it.True(ok))
	it.Then(t).
		Should(it.Equal(core.MergeUnion, entity.Merge)).
		Should(it.Seq(entity.Required).Equal("category", "definition", "mentionedIn")).
		ShouldNot(it.Equal("", entity.Description))

	isPartOf, ok := index.Predicates["isPartOf"]
	it.Then(t).Should(it.True(ok))
	it.Then(t).
		Should(it.Equal("edge", isPartOf.Role)).
		Should(it.Equal(core.MergeUnion, isPartOf.Merge)).
		ShouldNot(it.Equal("", isPartOf.Description))
}

func TestResolveNotAGraph(t *testing.T) {
	store := newFakeStore(nil)
	delete(store.dirs, ".arc")

	_, err := service.Resolve(store)

	it.Then(t).Should(it.True(errors.Is(err, service.ErrNotAGraph)))
}

func TestResolveMissingSchemaFolderFails(t *testing.T) {
	store := newFakeStore(nil)

	_, err := service.Resolve(store)

	it.Then(t).Should(it.True(errors.Is(err, service.ErrSchemaMissing)))
}

func TestResolveMalformedDocumentFails(t *testing.T) {
	store := newSeededStore()
	store.files[kernel.PredicatesDir+"/broken.md"] = "not valid front matter at all"

	_, err := service.Resolve(store)

	it.Then(t).Should(it.True(errors.Is(err, service.ErrSchemaInvalid)))
}

func TestResolveDocumentMissingRoleFails(t *testing.T) {
	store := newSeededStore()
	store.files[kernel.PredicatesDir+"/broken.md"] = "---\n\"@id\": broken\n\"@type\": Property\nmerge: union\n---\n# broken\n\nSome text.\n"

	_, err := service.Resolve(store)

	it.Then(t).Should(it.True(errors.Is(err, service.ErrSchemaInvalid)))
}

func TestResolveDocumentMissingDescriptionFails(t *testing.T) {
	store := newSeededStore()
	store.files[kernel.PredicatesDir+"/broken.md"] = "---\n\"@id\": broken\n\"@type\": Property\nrole: edge\nmerge: union\n---\n# broken\n"

	_, err := service.Resolve(store)

	it.Then(t).Should(it.True(errors.Is(err, service.ErrSchemaInvalid)))
}

func TestRegisterTypeCreatesFileOnce(t *testing.T) {
	store := newFakeStore(nil)

	created, err := service.RegisterType(store, "hypothesis")
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.True(created))

	content := store.written[kernel.TypesDir+"/hypothesis.md"]
	it.Then(t).
		Should(it.String(content).Contain(`"@type": Class`)).
		Should(it.String(content).Contain("merge: union"))

	created, err = service.RegisterType(store, "hypothesis")
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

	content := store.written[kernel.PredicatesDir+"/relatesTo.md"]
	it.Then(t).
		Should(it.String(content).Contain(`"@type": Property`)).
		Should(it.String(content).Contain("role: edge")).
		Should(it.String(content).Contain("merge: union"))

	created, err = service.RegisterPredicate(store, "relatesTo")
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.True(!created))
	it.Then(t).Should(it.Equal(1, len(store.written)))
}

func TestRegisterTypeWriteFailure(t *testing.T) {
	store := newFakeStore(nil)
	store.createErr = errors.New("disk full")

	_, err := service.RegisterType(store, "hypothesis")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrSchemaWrite)))
}

func TestRegisterPredicateWriteFailure(t *testing.T) {
	store := newFakeStore(nil)
	store.createErr = errors.New("disk full")

	_, err := service.RegisterPredicate(store, "relatesTo")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrSchemaWrite)))
}

func TestResolveReflectsHandEditedRoleValue(t *testing.T) {
	store := newSeededStore()
	store.files[kernel.PredicatesDir+"/isPartOf.md"] = "---\n\"@id\": isPartOf\n\"@type\": Property\nrole: edge\nmerge: union\n---\n# isPartOf\n\nComposition.\n"

	index, err := service.Resolve(store)
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal("edge", index.Predicates["isPartOf"].Role))

	store.files[kernel.PredicatesDir+"/isPartOf.md"] = "---\n\"@id\": isPartOf\n\"@type\": Property\nrole: link\nmerge: union\n---\n# isPartOf\n\nComposition.\n"

	index, err = service.Resolve(store)
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal("link", index.Predicates["isPartOf"].Role))
}
