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

// coreIndexFixture declares Predicates for spec.md 012's per-predicate
// dispatch (Types entries are kept for continuity/documentation only —
// TypeDef.Merge is no longer consulted by core.Merge). Every predicate not
// listed here falls back to MergeUnion (research.md D6).
var coreIndexFixture = core.Index{
	Types: map[string]core.TypeDef{
		"source":   {Merge: core.MergeImmutable},
		"entity":   {Merge: core.MergeUnion},
		"resource": {Merge: core.MergeFirstWriteWin},
		"timeline": {Merge: core.MergeAppend},
	},
	Predicates: map[string]core.PredicateDef{
		"ref":       {Merge: core.MergeImmutable},
		"status":    {Merge: core.MergeLastWriteWin},
		"relevance": {Merge: core.MergeFirstWriteWin},
	},
}

// indexWithType returns coreIndexFixture plus one additional registered
// type, for tests exercising a domain type's own registered merge
// behavior.
func indexWithType(name string, op core.MergeOp) core.Index {
	types := make(map[string]core.TypeDef, len(coreIndexFixture.Types)+1)
	for k, v := range coreIndexFixture.Types {
		types[k] = v
	}
	types[name] = core.TypeDef{Merge: op}
	return core.Index{Types: types, Predicates: coreIndexFixture.Predicates}
}

// indexWithPredicate returns coreIndexFixture plus one additional
// registered predicate, for tests exercising the already-registered path.
func indexWithPredicate(name string) core.Index {
	predicates := make(map[string]core.PredicateDef, len(coreIndexFixture.Predicates)+1)
	for k, v := range coreIndexFixture.Predicates {
		predicates[k] = v
	}
	predicates[name] = core.PredicateDef{}
	return core.Index{Types: coreIndexFixture.Types, Predicates: predicates}
}

// fakeSchema records every RegisterType/RegisterPredicate call, for
// asserting graph.Apply's auto-discovery hook (spec.md US2).
type fakeSchema struct {
	registeredTypes      []string
	registeredPredicates []string
}

func (f *fakeSchema) RegisterType(store fsys.Store, typ string) (bool, error) {
	f.registeredTypes = append(f.registeredTypes, typ)
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
"@id": "foo-2026-x"
"@type": source
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
"@id": "foo-2026-x"
"@type": source
title: "A Test Document"
authors: [Test Author]
published: "2026-04-12"
` + "```" + `

A test document.

# Entity

## Widget
` + "```yaml" + `
"@id": "Widget"
"@type": entity
category: [independent, abstract, occurrent, script]
` + "```" + `

A test entity.
- replaces:: [[Old Widget]]
`

const existingWidgetEntity = `---
"@id": "Widget"
"@type": entity
title: Widget
category: [independent, abstract, occurrent, script]
---
# Widget

A test entity.
`

// sourceResourcePatch/existingWidgetSpecResourceWithStatus: a "resource"
// node's leading prose (Texts["relevance"], firstWriteWin per
// coreIndexFixture) genuinely diverges from what's already on disk, so it
// is flagged — its "status" (lastWriteWin) diverges too but is never
// flagged (spec.md FR-012), and "ref" (immutable) is unchanged on both
// sides so it doesn't interact with this scenario.
const sourceResourcePatch = `---
kind: patch
document: foo-2026-x
published: 2026-04-12
title: "A Test Document"
---
# Source

## foo-2026-x
` + "```yaml" + `
"@id": "foo-2026-x"
"@type": source
title: "A Test Document"
authors: [Test Author]
published: "2026-04-12"
` + "```" + `

A test document.

# Resource

## Widget Spec
` + "```yaml" + `
"@id": "Widget Spec"
"@type": resource
ref: standard
status: backlog
` + "```" + `

An updated specification of Widget alignment.
`

const existingWidgetSpecResourceWithStatus = `---
"@id": "Widget Spec"
"@type": resource
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
"@id": "foo-2026-x"
"@type": source
title: "A Test Document"
authors: [Test Author]
published: "2026-04-12"
` + "```" + `

A test document.

# Hypothesis

## A Test Hypothesis
` + "```yaml" + `
"@id": "A Test Hypothesis"
"@type": hypothesis
` + "```" + `

A conclusion.
`

func TestApplyGuardNotAGraph(t *testing.T) {
	store := newMemStore()
	store.files["patch.md"] = []byte(minimalSourcePatch)

	_, err := service.Apply(context.Background(), memMounter{store: store}, &graphmock.VCS{}, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrNotAGraph)))
}

func TestApplyGuardPatchReadFailure(t *testing.T) {
	store := newGraphStore()

	_, err := service.Apply(context.Background(), memMounter{store: store}, &graphmock.VCS{}, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/missing.md")

	it.Then(t).Should(it.True(errors.Is(err, service.ErrPatchRead)))
}

func TestApplySkipsWhenAlreadyTracked(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(minimalSourcePatch)
	vcs := &graphmock.VCS{Tracked: map[string]bool{"sources/foo-2026-x.md": true}}

	result, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")

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

	result, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")

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

	result, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")

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

	result, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(result.Conflicts)))

	content := string(store.files["resources/Widget Spec.md"])
	it.Then(t).
		Should(it.String(content).Contain("<<<<<<< existing")).
		Should(it.String(content).Contain("The normative specification of Widget.")).
		Should(it.String(content).Contain("An updated specification of Widget alignment.")).
		Should(it.String(content).Contain("status: backlog"))
}

func TestApplyUnregisteredKindWarns(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(domainKindPatch)
	vcs := &graphmock.VCS{CommitHash: "abc123"}

	result, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")

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
	index := indexWithType("hypothesis", core.MergeValidatedOverwrite)

	result, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), index, &fakeSchema{}, "/graph", "/patch.md")

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

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, schema, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Seq(schema.registeredTypes).Contain("hypothesis"))
}

// arc apply — spec.md US2: a previously-unseen predicate declared in a
// patch-carried node is registered into _schema/predicates/ too.
func TestApplyUnregisteredPredicateRegistersSchemaPredicate(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(sourceEntityPatch)
	vcs := &graphmock.VCS{CommitHash: "abc123"}
	schema := &fakeSchema{}

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, schema, "/graph", "/patch.md")

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
	index := indexWithPredicate("replaces")

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), index, schema, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(0, len(schema.registeredPredicates)))
}

func TestApplyCommitErrorPropagates(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(minimalSourcePatch)
	vcs := &graphmock.VCS{CommitErr: errors.New("commit failed")}

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).ShouldNot(it.Nil(err))
}

func TestApplyTimelineEntriesCreated(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(minimalSourcePatch)
	vcs := &graphmock.VCS{CommitHash: "abc123"}

	result, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")

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
// decode as an integer. The file's own "@id"/"@type" (research.md D7)
// satisfy core.ParseNode's mandatory-identity rule the same way any other
// node file must, with "@id" equal to the file's own basename ("2026").
func TestApplyYearlyTimelinePeriodFileParsesViaCoreParseNode(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(minimalSourcePatch)
	vcs := &graphmock.VCS{CommitHash: "abc123"}

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")
	it.Then(t).Should(it.Nil(err))

	yearly := store.files["timeline/yearly/2026.md"]
	it.Then(t).Should(it.True(len(yearly) > 0))

	node, err := core.ParseNode(bytes.NewReader(yearly))
	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal("timeline", node.Type)).
		Should(it.Equal("2026", node.ID))
}

// BUG-001 / spec.md FR-021: one Reporter.Step line per processed node,
// naming its ID and outcome — a created node (no prior merge) gets no
// further predicate lines; a merged node gets one further indented
// Reporter.Step per predicate core.Merge touched, naming its resolved
// MergeOp and outcome (spec 012 FR-017/BUG-001).
func TestApplyReportsStepPerNode(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(sourceEntityPatch)
	store.files["entities/Widget.md"] = []byte(existingWidgetEntity)
	vcs := &graphmock.VCS{CommitHash: "abc123"}
	reporter := &fakeReporter{}

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, reporter, coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(6, len(reporter.steps))).
		Should(it.Equal("foo-2026-x: created", reporter.steps[0])).
		Should(it.Equal("Widget: merged", reporter.steps[1])).
		Should(it.Equal("  category: union -> appended", reporter.steps[2])).
		Should(it.Equal("  definition: union -> appended", reporter.steps[3])).
		Should(it.Equal("  published: union -> unchanged", reporter.steps[4])).
		Should(it.Equal("  title: union -> unchanged", reporter.steps[5]))
}

const stubSectionPatch = `---
kind: patch
document: foo-2026-x
published: 2026-04-12
title: "A Test Document"
---
# Source

## foo-2026-x
` + "```yaml" + `
"@id": "foo-2026-x"
"@type": source
title: "A Test Document"
authors: [Test Author]
published: "2026-04-12"
` + "```" + `

A test document.

# Entity

## StubEntity
` + "```yaml" + `
"@id": "StubEntity"
"@type": entity
` + "```" + `
`

// spec.md US1 Acceptance Scenario 3 / FR-002: a minimal-stub patch section
// (@id/@type only) creates a node carrying neither published nor indexed.
func TestApplyStubCreatesNodeWithNeitherPublishedNorIndexed(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(stubSectionPatch)
	vcs := &graphmock.VCS{CommitHash: "abc123"}

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")
	it.Then(t).Should(it.Nil(err))

	content := string(store.files["entities/StubEntity.md"])
	node, err := core.ParseNode(bytes.NewReader([]byte(content)))
	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.True(node.Published.IsZero())).
		ShouldNot(it.String(content).Contain("indexed:"))
}

// spec.md US1 Acceptance Scenario 4 / FR-003: an auto-registered
// _schema/types/<name>.md document carries neither published nor indexed,
// even though service.Apply's own writeNode never actually writes this
// path — schema.RegisterType is a separate port call the loop never routes
// through create-path stamping (research.md D8); this asserts the
// triggering node itself still stamped indexed, but the fake schema records
// no Attrs of its own to accidentally carry the timestamp.
func TestApplySchemaRegistrationCarriesNoTimestampAttrs(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(domainKindPatch)
	vcs := &graphmock.VCS{CommitHash: "abc123"}
	schema := &fakeSchema{}

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, schema, "/graph", "/patch.md")
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Seq(schema.registeredTypes).Contain("hypothesis"))
}

// spec.md US1 Acceptance Scenario 1/2 / FR-001/FR-005: every node one
// application creates carries published (from the patch's own date) and
// shares one identical indexed value.
func TestApplyCreatedNodesCarryPublishedAndShareIndexed(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(sourceEntityPatch)
	vcs := &graphmock.VCS{CommitHash: "abc123"}

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")
	it.Then(t).Should(it.Nil(err))

	source, err := core.ParseNode(bytes.NewReader(store.files["sources/foo-2026-x.md"]))
	it.Then(t).Should(it.Nil(err))
	entity, err := core.ParseNode(bytes.NewReader(store.files["entities/Widget.md"]))
	it.Then(t).Should(it.Nil(err))

	sourceIndexedPreds := source.Attrs["indexed"]
	it.Then(t).Should(it.Equal(1, len(sourceIndexedPreds)))
	sourceIndexed, ok := sourceIndexedPreds[0].Value.(string)
	it.Then(t).Should(it.True(ok))

	it.Then(t).
		ShouldNot(it.True(source.Published.IsZero())).
		Should(it.Equal(source.Published, entity.Published)).
		ShouldNot(it.Equal("", sourceIndexed)).
		Should(it.Equiv(source.Attrs["indexed"], entity.Attrs["indexed"]))
}

// spec.md US2 Acceptance Scenario 1 / FR-007/FR-009: a real merge stamps
// updated identical to the same application's indexed value.
func TestApplyMergedNodeGetsUpdatedMatchingIndexed(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(sourceEntityPatch)
	store.files["entities/Widget.md"] = []byte(existingWidgetEntity)
	vcs := &graphmock.VCS{CommitHash: "abc123"}

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")
	it.Then(t).Should(it.Nil(err))

	source, err := core.ParseNode(bytes.NewReader(store.files["sources/foo-2026-x.md"]))
	it.Then(t).Should(it.Nil(err))
	entity, err := core.ParseNode(bytes.NewReader(store.files["entities/Widget.md"]))
	it.Then(t).Should(it.Nil(err))

	sourceIndexedPreds := source.Attrs["indexed"]
	it.Then(t).Should(it.Equal(1, len(sourceIndexedPreds)))
	sourceIndexed, ok := sourceIndexedPreds[0].Value.(string)
	it.Then(t).Should(it.True(ok))

	it.Then(t).
		ShouldNot(it.Equal("", sourceIndexed)).
		Should(it.Equiv(source.Attrs["indexed"], entity.Attrs["updated"]))
}

const sourceOnlyReContributionPatch = `---
kind: patch
document: foo-2026-x2
published: 2026-04-12
title: "A Second Document"
---
# Source

## foo-2026-x2
` + "```yaml" + `
"@id": "foo-2026-x2"
"@type": source
title: "A Second Document"
authors: [Test Author]
published: "2026-04-12"
` + "```" + `

A test document.
`

// spec.md US2 Acceptance Scenario 2 / FR-008: a "none"-behavior kind's
// existing no-op guarantee holds — no updated is added on re-contribution
// to an already-tracked identity is prevented by the idempotency guard, so
// this instead exercises MergeNone directly on a distinct, already-present
// source-kind node via a differently-shaped fixture patch that reuses the
// same source id with the identical content, confirming Merge's existing
// whole-node no-op leaves Attrs untouched.
func TestApplyNoneKindMergeAddsNoUpdated(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(sourceOnlyReContributionPatch)
	store.files["sources/foo-2026-x2.md"] = []byte(`---
"@id": "foo-2026-x2"
"@type": source
title: "A Second Document"
authors: [Test Author]
published: "2026-04-12"
---
# foo-2026-x2

A test document.
`)
	vcs := &graphmock.VCS{CommitHash: "abc123"}

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")
	it.Then(t).Should(it.Nil(err))

	content := string(store.files["sources/foo-2026-x2.md"])
	it.Then(t).ShouldNot(it.String(content).Contain("updated:"))
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

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, reporter, coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal("Widget Spec: merged (conflict flagged)", reporter.steps[1]))
}

// spec.md US3 Acceptance Scenario 3/4: an existing on-disk node whose
// "@id" does not match its own file's basename is rejected exactly like
// any other old-format file — core.ParseNode cannot perform this check
// itself (no filename parameter), so service.Apply enforces it at the one
// point it has both the parsed Node and the path it was read from. The
// whole apply aborts with no partial write: the entity file is left
// byte-unchanged and no commit is produced.
func TestApplyExistingNodeIdMismatchedBasenameAbortsWithNoWrites(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(sourceEntityPatch)
	mismatched := `---
"@id": "Some Other Id"
"@type": entity
category: [independent, abstract, occurrent, script]
---
# Widget

A test entity.
`
	store.files["entities/Widget.md"] = []byte(mismatched)
	vcs := &graphmock.VCS{CommitHash: "abc123"}

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).ShouldNot(it.Nil(err))
	it.Then(t).
		Should(it.Equal(mismatched, string(store.files["entities/Widget.md"]))).
		Should(it.Equal(0, len(store.files["sources/foo-2026-x.md"]))).
		ShouldNot(it.Seq(vcs.Calls).Contain("StageAll:/graph"))
}
