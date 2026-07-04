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
	graphmock "github.com/fogfish/arcnet-cli/internal/app/graph/adapter/mock"
	"github.com/fogfish/arcnet-cli/internal/app/graph/service"
	"github.com/fogfish/arcnet-cli/internal/bios"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// fakeReporter records every Step call, for asserting per-node progress
// text (BUG-001, spec.md FR-021).
type fakeReporter struct {
	steps []string
}

func (r *fakeReporter) Start(string)               {}
func (r *fakeReporter) Step(label string)          { r.steps = append(r.steps, label) }
func (r *fakeReporter) Done(string, time.Duration) {}
func (r *fakeReporter) Error(string, error)        {}

var coreMergeRulesFixture = core.MergeRuleSet{
	"source":   core.MergeNone,
	"entity":   core.MergeUnion,
	"resource": core.MergeUnionFirstWriter,
	"timeline": core.MergeAppend,
}

var emptyPredicates = map[string]bool{}

// fakeSchema records every RegisterKind/RegisterPredicate call, for
// asserting graph.Apply's auto-discovery hook (spec.md US2).
type fakeSchema struct {
	registeredKinds      []core.Kind
	registeredPredicates []string
}

func (f *fakeSchema) RegisterKind(store fsys.Store, kind core.Kind) (bool, error) {
	f.registeredKinds = append(f.registeredKinds, kind)
	return true, nil
}

func (f *fakeSchema) RegisterPredicate(store fsys.Store, predicate string) (bool, error) {
	f.registeredPredicates = append(f.registeredPredicates, predicate)
	return true, nil
}

type memFileInfo struct{ name string }

func (i memFileInfo) Name() string       { return i.name }
func (i memFileInfo) Size() int64        { return 0 }
func (i memFileInfo) Mode() fs.FileMode  { return 0 }
func (i memFileInfo) ModTime() time.Time { return time.Time{} }
func (i memFileInfo) IsDir() bool        { return false }
func (i memFileInfo) Sys() any           { return nil }

type memOpenFile struct {
	*bytes.Reader
	name string
}

func (f memOpenFile) Close() error               { return nil }
func (f memOpenFile) Stat() (fs.FileInfo, error) { return memFileInfo{name: f.name}, nil }

type memFile struct {
	name  string
	buf   *bytes.Buffer
	store *memStore
}

func (f *memFile) Write(p []byte) (int, error) { return f.buf.Write(p) }
func (f *memFile) Close() error {
	f.store.files[f.name] = append([]byte(nil), f.buf.Bytes()...)
	return nil
}
func (f *memFile) Stat() (fs.FileInfo, error) { return memFileInfo{name: f.name}, nil }
func (f *memFile) Discard() error             { return nil }

type memStore struct {
	files   map[string][]byte
	dirs    map[string]bool
	removed []string
}

func newMemStore() *memStore {
	return &memStore{files: map[string][]byte{}, dirs: map[string]bool{}}
}

func newGraphStore() *memStore {
	s := newMemStore()
	s.dirs[".arc"] = true
	return s
}

func (s *memStore) Open(name string) (fs.File, error) {
	content, ok := s.files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return memOpenFile{bytes.NewReader(content), name}, nil
}

func (s *memStore) Stat(name string) (fs.FileInfo, error) {
	if s.dirs[name] {
		return memFileInfo{name: name}, nil
	}
	if _, ok := s.files[name]; ok {
		return memFileInfo{name: name}, nil
	}
	return nil, fs.ErrNotExist
}

func (s *memStore) ReadDir(name string) ([]fs.DirEntry, error) { return nil, nil }

func (s *memStore) Create(name string) (fsys.File, error) {
	return &memFile{name: name, buf: &bytes.Buffer{}, store: s}, nil
}

func (s *memStore) Remove(name string) error {
	delete(s.files, name)
	s.removed = append(s.removed, name)
	return nil
}

type memMounter struct{ store *memStore }

func (m memMounter) Mount(root string) (fsys.Store, error) { return m.store, nil }

const minimalSourcePatch = `---
kind: patch
document: foo-2026-x
published: 2026-04-12
title: "A Test Document"
---
# Source

## foo-2026-x
` + "```yaml" + `
title: "A Test Document"
authors: [Test Author]
published: "2026-04-12"
` + "```" + `

A test document.
`

const sourceEntityPatch = `---
kind: patch
document: foo-2026-x
published: 2026-04-12
title: "A Test Document"
---
# Source

## foo-2026-x
` + "```yaml" + `
title: "A Test Document"
authors: [Test Author]
published: "2026-04-12"
` + "```" + `

A test document.

# Entity

## Widget
` + "```yaml" + `
category: [independent, abstract, occurrent, script]
` + "```" + `

A test entity.
- replaces:: [[Old Widget]]
`

const existingWidgetEntity = `---
kind: entity
title: Widget
category: [independent, abstract, occurrent, script]
---
# Widget

A test entity.
`

// sourceResourcePatch/existingWidgetSpecResourceWithStatus (BUG-004): a
// "resource" node uses MergeUnionFirstWriter, untouched by BUG-004's
// MergeUnion-only fix, so an already-populated scalar field it diverges on
// is still flagged — unlike an "entity" (MergeUnion) node's Text/Attrs,
// which no longer conflict at all (see internal/core/merge_test.go).
const sourceResourcePatch = `---
kind: patch
document: foo-2026-x
published: 2026-04-12
title: "A Test Document"
---
# Source

## foo-2026-x
` + "```yaml" + `
title: "A Test Document"
authors: [Test Author]
published: "2026-04-12"
` + "```" + `

A test document.

# Resource

## Widget Spec
` + "```yaml" + `
ref: standard
status: backlog
` + "```" + `

The normative specification of Widget.
`

const existingWidgetSpecResourceWithStatus = `---
kind: resource
title: Widget Spec
ref: standard
status: read
---
# Widget Spec

The normative specification of Widget.
`

const domainKindPatch = `---
kind: patch
document: foo-2026-x
published: 2026-04-12
title: "A Test Document"
---
# Source

## foo-2026-x
` + "```yaml" + `
title: "A Test Document"
authors: [Test Author]
published: "2026-04-12"
` + "```" + `

A test document.

# Hypothesis

## A Test Hypothesis
` + "```yaml\n```" + `

A conclusion.
`

func TestApplyGuardNotAGraph(t *testing.T) {
	store := newMemStore()
	store.files["patch.md"] = []byte(minimalSourcePatch)

	_, err := service.Apply(context.Background(), memMounter{store: store}, &graphmock.VCS{}, bios.NewReporter(true, true), coreMergeRulesFixture, emptyPredicates, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrNotAGraph)))
}

func TestApplyGuardPatchReadFailure(t *testing.T) {
	store := newGraphStore()

	_, err := service.Apply(context.Background(), memMounter{store: store}, &graphmock.VCS{}, bios.NewReporter(true, true), coreMergeRulesFixture, emptyPredicates, &fakeSchema{}, "/graph", "/missing.md")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrPatchRead)))
}

func TestApplySkipsWhenAlreadyTracked(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(minimalSourcePatch)
	vcs := &graphmock.VCS{Tracked: map[string]bool{"sources/foo-2026-x.md": true}}

	result, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreMergeRulesFixture, emptyPredicates, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.True(result.Skipped)).
		Should(it.Equal("foo-2026-x", result.Document)).
		Should(it.Equal(1, len(store.files))) // only the patch file itself, no new writes
	it.Then(t).ShouldNot(it.Seq(vcs.Calls).Contain("StageAll:/graph"))
}

func TestApplyCreatesNewNode(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(minimalSourcePatch)
	vcs := &graphmock.VCS{CommitHash: "abc123"}

	result, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreMergeRulesFixture, emptyPredicates, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, result.Created["source"])).
		Should(it.Equal("abc123", result.CommitHash))
	it.Then(t).Should(it.True(len(store.files["sources/foo-2026-x.md"]) > 0))
	it.Then(t).
		Should(it.Seq(vcs.Calls).Contain("StageAll:/graph"))
}

func TestApplyMergesExistingNode(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(sourceEntityPatch)
	store.files["entities/Widget.md"] = []byte(existingWidgetEntity)
	vcs := &graphmock.VCS{CommitHash: "abc123"}

	result, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreMergeRulesFixture, emptyPredicates, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, result.Created["source"])).
		Should(it.Equal(1, result.Merged["entity"])).
		Should(it.Equal(0, len(result.Conflicts)))

	content := string(store.files["entities/Widget.md"])
	it.Then(t).Should(it.String(content).Contain("replaces:: [[Old Widget]]"))
}

// BUG-004: a "resource" node (MergeUnionFirstWriter) is unaffected by this
// bugfix — its already-populated scalar field is still flagged as a
// conflict on divergence, exactly as before.
func TestApplyFlagsConflict(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(sourceResourcePatch)
	store.files["resources/Widget Spec.md"] = []byte(existingWidgetSpecResourceWithStatus)
	vcs := &graphmock.VCS{CommitHash: "abc123"}

	result, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreMergeRulesFixture, emptyPredicates, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(result.Conflicts)))

	content := string(store.files["resources/Widget Spec.md"])
	it.Then(t).
		Should(it.String(content).Contain("<<<<<<< existing")).
		Should(it.String(content).Contain("read")).
		Should(it.String(content).Contain("backlog"))
}

func TestApplyUnregisteredKindWarns(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(domainKindPatch)
	vcs := &graphmock.VCS{CommitHash: "abc123"}

	result, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreMergeRulesFixture, emptyPredicates, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, result.Created["hypothesis"])).
		Should(it.Equal(1, len(result.Warnings))).
		Should(it.String(result.Warnings[0]).Contain("hypothesis"))
}

func TestApplyRegisteredKindNoWarning(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(domainKindPatch)
	vcs := &graphmock.VCS{CommitHash: "abc123"}
	rules := coreMergeRulesFixture.Union(core.MergeRuleSet{"hypothesis": core.MergeValidatedOverwrite})

	result, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), rules, emptyPredicates, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(0, len(result.Warnings)))
}

// arc apply — spec.md US2: an unregistered kind is also registered into
// _schema/ via the SchemaRegistry port, in the same call as the triggering
// patch (research.md D3).
func TestApplyUnregisteredKindRegistersSchemaKind(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(domainKindPatch)
	vcs := &graphmock.VCS{CommitHash: "abc123"}
	schema := &fakeSchema{}

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreMergeRulesFixture, emptyPredicates, schema, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Seq(schema.registeredKinds).Contain(core.Kind("hypothesis")))
}

// arc apply — spec.md US2: a previously-unseen predicate declared in a
// patch-carried node is registered into _schema/predicates/ too.
func TestApplyUnregisteredPredicateRegistersSchemaPredicate(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(sourceEntityPatch)
	vcs := &graphmock.VCS{CommitHash: "abc123"}
	schema := &fakeSchema{}

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreMergeRulesFixture, emptyPredicates, schema, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Seq(schema.registeredPredicates).Contain("replaces"))
}

// arc apply — spec.md US2 Acceptance Scenario 3: an already-registered
// predicate is not re-registered.
func TestApplyRegisteredPredicateNotReRegistered(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(sourceEntityPatch)
	vcs := &graphmock.VCS{CommitHash: "abc123"}
	schema := &fakeSchema{}
	predicates := map[string]bool{"replaces": true}

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreMergeRulesFixture, predicates, schema, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(0, len(schema.registeredPredicates)))
}

func TestApplyCommitErrorPropagates(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(minimalSourcePatch)
	vcs := &graphmock.VCS{CommitErr: errors.New("commit failed")}

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreMergeRulesFixture, emptyPredicates, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).ShouldNot(it.Nil(err))
}

func TestApplyTimelineEntriesCreated(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(minimalSourcePatch)
	vcs := &graphmock.VCS{CommitHash: "abc123"}

	result, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreMergeRulesFixture, emptyPredicates, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Seq(result.Timeline).Equal("2026", "2026-04"))
	it.Then(t).
		Should(it.True(len(store.files["timeline/yearly/2026.md"]) > 0)).
		Should(it.True(len(store.files["timeline/monthly/2026-04.md"]) > 0))
	it.Then(t).Should(it.String(string(store.files["timeline/monthly/2026-04.md"])).Contain("foo-2026-x"))
}

// BUG-007: a yearly timeline period file's bare, numeric-looking `period`
// value (e.g. "2026") must round-trip through the generic
// core.ParseNode, not just this feature's own bespoke
// parseTimelineEntries scan — an unquoted YAML scalar would otherwise
// decode as an integer, breaking core.deriveNodeID's period fallback.
func TestApplyYearlyTimelinePeriodFileParsesViaCoreParseNode(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(minimalSourcePatch)
	vcs := &graphmock.VCS{CommitHash: "abc123"}

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreMergeRulesFixture, emptyPredicates, &fakeSchema{}, "/graph", "/patch.md")
	it.Then(t).Should(it.Nil(err))

	yearly := store.files["timeline/yearly/2026.md"]
	it.Then(t).Should(it.True(len(yearly) > 0))

	node, err := core.ParseNode(bytes.NewReader(yearly))
	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(core.Kind("timeline"), node.Kind)).
		Should(it.Equal("2026", node.ID))
}

// BUG-001 / spec.md FR-021: one Reporter.Step line per processed node,
// naming its ID and outcome.
func TestApplyReportsStepPerNode(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(sourceEntityPatch)
	store.files["entities/Widget.md"] = []byte(existingWidgetEntity)
	vcs := &graphmock.VCS{CommitHash: "abc123"}
	reporter := &fakeReporter{}

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, reporter, coreMergeRulesFixture, emptyPredicates, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(2, len(reporter.steps))).
		Should(it.Equal("foo-2026-x: created", reporter.steps[0])).
		Should(it.Equal("Widget: merged", reporter.steps[1]))
}

// BUG-004: uses the same resource-kind conflict fixture as
// TestApplyFlagsConflict above, since an "entity" (MergeUnion) node no
// longer ever flags a conflict.
func TestApplyReportsStepConflictFlagged(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(sourceResourcePatch)
	store.files["resources/Widget Spec.md"] = []byte(existingWidgetSpecResourceWithStatus)
	vcs := &graphmock.VCS{CommitHash: "abc123"}
	reporter := &fakeReporter{}

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, reporter, coreMergeRulesFixture, emptyPredicates, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal("Widget Spec: merged (conflict flagged)", reporter.steps[1]))
}
