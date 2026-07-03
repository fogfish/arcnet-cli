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
	"context"
	"errors"
	"io/fs"
	"testing"
	"time"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	configmock "github.com/fogfish/arcnet-cli/internal/app/config/adapter/mock"
	"github.com/fogfish/arcnet-cli/internal/app/config/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/config/service"
	"github.com/fogfish/arcnet-cli/internal/core"
)

type fakeFileInfo struct{ name string }

func (i fakeFileInfo) Name() string       { return i.name }
func (i fakeFileInfo) Size() int64        { return 0 }
func (i fakeFileInfo) Mode() fs.FileMode  { return 0 }
func (i fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (i fakeFileInfo) IsDir() bool        { return false }
func (i fakeFileInfo) Sys() any           { return nil }

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
	files   map[string]string
	written map[string]string
}

func newFakeStore(files map[string]string) *fakeStore {
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

func (s *fakeStore) ReadDir(name string) ([]fs.DirEntry, error) { return nil, nil }

func (s *fakeStore) Create(name string) (fsys.File, error) {
	return &fakeWriteFile{name: name, buf: &bytes.Buffer{}, on: func(n string, c []byte) {
		s.written[n] = string(c)
	}}, nil
}

func (s *fakeStore) Remove(name string) error { return nil }

func TestResolveAbsentFileReturnsCoreRulesOnly(t *testing.T) {
	store := newFakeStore(nil)

	rules, err := service.Resolve(store)

	it.Then(t).Should(it.Nil(err))
	op, ok := rules.Lookup("source")
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal(core.MergeNone, op))
	_, ok = rules.Lookup("hypothesis")
	it.Then(t).Should(it.True(!ok))
}

func TestResolveMalformedFile(t *testing.T) {
	store := newFakeStore(map[string]string{core.ConfigPath: "not: [valid: yaml"})

	_, err := service.Resolve(store)

	it.Then(t).Should(it.True(errors.Is(err, service.ErrConfigMalformed)))
}

func TestResolveUnionsRegisteredKind(t *testing.T) {
	store := newFakeStore(map[string]string{core.ConfigPath: "mergeRules:\n  hypothesis: validated-overwrite\n"})

	rules, err := service.Resolve(store)

	it.Then(t).Should(it.Nil(err))
	op, ok := rules.Lookup("hypothesis")
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal(core.MergeValidatedOverwrite, op))
	sourceOp, _ := rules.Lookup("source")
	it.Then(t).Should(it.Equal(core.MergeNone, sourceOp))
}

func TestSaveWritesYAML(t *testing.T) {
	store := newFakeStore(nil)
	cfg := kernel.Config{MergeRules: core.MergeRuleSet{"hypothesis": core.MergeValidatedOverwrite}}

	err := service.Save(store, cfg)

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.String(store.written[core.ConfigPath]).Contain("hypothesis"))
}

func TestDefaultFetchSucceeds(t *testing.T) {
	fetcher := configmock.Fetcher{Body: []byte("mergeRules:\n  source: none\n  entity: union\n")}

	cfg, usedFallback := service.Default(context.Background(), fetcher)

	it.Then(t).Should(it.True(!usedFallback))
	op, ok := cfg.MergeRules.Lookup("entity")
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal(core.MergeUnion, op))
}

func TestDefaultFetchNetworkError(t *testing.T) {
	fetcher := configmock.Fetcher{Err: errors.New("network unreachable")}

	cfg, usedFallback := service.Default(context.Background(), fetcher)

	it.Then(t).Should(it.True(usedFallback))
	op, _ := cfg.MergeRules.Lookup("source")
	it.Then(t).Should(it.Equal(core.MergeNone, op))
}

func TestDefaultFetchMalformedYAML(t *testing.T) {
	fetcher := configmock.Fetcher{Body: []byte("not: [valid: yaml")}

	cfg, usedFallback := service.Default(context.Background(), fetcher)

	it.Then(t).Should(it.True(usedFallback))
	it.Then(t).Should(it.True(cfg.MergeRules != nil))
}

func TestDefaultFetchEmptyMergeRules(t *testing.T) {
	fetcher := configmock.Fetcher{Body: []byte("title: unrelated\n")}

	cfg, usedFallback := service.Default(context.Background(), fetcher)

	it.Then(t).Should(it.True(usedFallback))
	op, ok := cfg.MergeRules.Lookup("source")
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal(core.MergeNone, op))
}
