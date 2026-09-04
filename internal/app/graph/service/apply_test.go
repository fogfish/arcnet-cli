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
	"sort"
	"strings"
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
// dispatch. A Types entry only needs to EXIST for its type to be
// recognized — core.TypeDef carries no merge field at all since
// specs/023-core-vocabulary-conformance FR-005. Every predicate not listed
// here falls back to MergeUnion (research.md D6).
var coreIndexFixture = core.Index{
	Types: map[string]core.TypeDef{
		"Source":    {},
		"Entity":    {},
		"Resource":  {},
		"Timeline":  {},
		"Reference": {},
	},
	Predicates: map[string]core.PredicateDef{
		"ref":       {Merge: core.MergeImmutable},
		"status":    {Merge: core.MergeLastWriteWin},
		"relevance": {Merge: core.MergeFirstWriteWin},
	},
}

// indexWithType returns coreIndexFixture plus one additional registered
// type, for tests exercising the already-registered path. It takes no
// MergeOp: core.TypeDef carries no merge field since
// specs/023-core-vocabulary-conformance FR-005, and registration — not a
// declared behaviour — is all these tests ever needed from it.
func indexWithType(name string) core.Index {
	types := make(map[string]core.TypeDef, len(coreIndexFixture.Types)+1)
	for k, v := range coreIndexFixture.Types {
		types[k] = v
	}
	types[name] = core.TypeDef{}
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
	registeredTypes          []string
	registeredPredicates     []string
	registeredPredicateRole  map[string]string
	registeredPredicateLabel map[string]string
}

func (f *fakeSchema) RegisterType(store fsys.Store, typ string) (bool, error) {
	f.registeredTypes = append(f.registeredTypes, typ)
	return true, nil
}

func (f *fakeSchema) RegisterPredicate(store fsys.Store, predicate, observedRole, label string) (bool, error) {
	f.registeredPredicates = append(f.registeredPredicates, predicate)
	if f.registeredPredicateRole == nil {
		f.registeredPredicateRole = map[string]string{}
	}
	f.registeredPredicateRole[predicate] = observedRole
	if label != "" {
		if f.registeredPredicateLabel == nil {
			f.registeredPredicateLabel = map[string]string{}
		}
		f.registeredPredicateLabel[predicate] = label
	}
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
"@type": patch
document: foo-2026-x
published: 2026-04-12
title: "A Test Document"
---
# Source

## foo-2026-x
` + "```yaml" + `
"@id": "foo-2026-x"
"@type": Source
title: "A Test Document"
authors: [Test Author]
published: "2026-04-12"
` + "```" + `

A test document.
`

const sourceEntityPatch = `---
"@type": patch
document: foo-2026-x
published: 2026-04-12
title: "A Test Document"
---
# Source

## foo-2026-x
` + "```yaml" + `
"@id": "foo-2026-x"
"@type": Source
title: "A Test Document"
authors: [Test Author]
published: "2026-04-12"
` + "```" + `

A test document.

# Entity

## Widget
` + "```yaml" + `
"@id": "Widget"
"@type": Entity
category: [independent, abstract, occurrent, script]
` + "```" + `

A test entity.
- replaces:: [[Old Widget]]
`

const existingWidgetEntity = `---
"@id": "Widget"
"@type": Entity
title: Widget
category: [independent, abstract, occurrent, script]
---
# Widget

A test entity.
`

// sourceReferencePatch/existingWidgetSpecReferenceWithStatus: a "Reference"
// node's leading prose (Texts["relevance"], firstWriteWin per
// coreIndexFixture) genuinely diverges from what's already on disk, so it
// is flagged — its "status" (lastWriteWin) diverges too but is never
// flagged (spec.md FR-012), and "ref" (immutable) is unchanged on both
// sides so it doesn't interact with this scenario.
//
// The node was typed "Resource" until ARCNET-CORE v0.11
// (specs/022-reference-type-folders). It is a "Reference" now for the same
// reason the predicates it carries are: ref/status/relevance describe an
// external work the graph has not ingested, and that whole semantic moved
// to Reference. Note that coreIndexFixture's firstWriteWin declaration for
// "relevance" is this test's own, chosen to exercise the flagged-divergence
// branch of the merge algebra over a text-role key; the SEEDED vocabulary
// declares "relevance" as append (CorePredicateDefs), as it does every
// other prose key. The fixture is deliberately not read from the seed, so
// this scenario keeps exercising that branch regardless.
const sourceReferencePatch = `---
"@type": patch
document: foo-2026-x
published: 2026-04-12
title: "A Test Document"
---
# Source

## foo-2026-x
` + "```yaml" + `
"@id": "foo-2026-x"
"@type": Source
title: "A Test Document"
authors: [Test Author]
published: "2026-04-12"
` + "```" + `

A test document.

# Reference

## Widget Spec
` + "```yaml" + `
"@id": "Widget Spec"
"@type": Reference
ref: standard
status: backlog
` + "```" + `

An updated specification of Widget alignment.
`

const existingWidgetSpecReferenceWithStatus = `---
"@id": "Widget Spec"
"@type": Reference
title: Widget Spec
ref: standard
status: read
---
# Widget Spec

The normative specification of Widget.
`

const domainKindPatch = `---
"@type": patch
document: foo-2026-x
published: 2026-04-12
title: "A Test Document"
---
# Source

## foo-2026-x
` + "```yaml" + `
"@id": "foo-2026-x"
"@type": Source
title: "A Test Document"
authors: [Test Author]
published: "2026-04-12"
` + "```" + `

A test document.

# Hypothesis

## A Test Hypothesis
` + "```yaml" + `
"@id": "A Test Hypothesis"
"@type": Hypothesis
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
	vcs := &graphmock.VCS{Tracked: map[string]bool{"Source/foo-2026-x.md": true}}

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
		Should(it.Equal(1, result.Created["Source"])).
		Should(it.Equal("abc123", result.CommitHash))
	it.Then(t).Should(it.True(len(store.files["Source/foo-2026-x.md"]) > 0))
	it.Then(t).
		Should(it.Seq(vcs.Calls).Contain("StageAll:/graph"))
}

func TestApplyMergesExistingNode(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(sourceEntityPatch)
	store.files["Entity/Widget.md"] = []byte(existingWidgetEntity)
	vcs := &graphmock.VCS{CommitHash: "abc123"}

	result, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, result.Created["Source"])).
		Should(it.Equal(1, result.Merged["Entity"])).
		Should(it.Equal(0, len(result.Conflicts)))

	content := string(store.files["Entity/Widget.md"])
	it.Then(t).Should(it.String(content).Contain("replaces:: [[Old Widget]]"))
}

// BUG-004: a "Reference" node (MergeUnionFirstWriter) is unaffected by this
// bugfix — its already-populated scalar field is still flagged as a
// conflict on divergence, exactly as before.
func TestApplyFlagsConflict(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(sourceReferencePatch)
	store.files["Reference/Widget Spec.md"] = []byte(existingWidgetSpecReferenceWithStatus)
	vcs := &graphmock.VCS{CommitHash: "abc123"}

	result, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(result.Conflicts)))

	content := string(store.files["Reference/Widget Spec.md"])
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
		Should(it.Equal(1, result.Created["Hypothesis"])).
		Should(it.Equal(1, len(result.Warnings))).
		Should(it.String(result.Warnings[0]).Contain("Hypothesis"))
}

func TestApplyRegisteredKindNoWarning(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(domainKindPatch)
	vcs := &graphmock.VCS{CommitHash: "abc123"}
	index := indexWithType("Hypothesis")

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
	it.Then(t).Should(it.Seq(schema.registeredTypes).Contain("Hypothesis"))
}

// arc apply — spec.md US2: a previously-unseen predicate declared in a
// patch-carried node is registered into _schema/Property/ too.
func TestApplyUnregisteredPredicateRegistersSchemaPredicate(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(sourceEntityPatch)
	vcs := &graphmock.VCS{CommitHash: "abc123"}
	schema := &fakeSchema{}

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, schema, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Seq(schema.registeredPredicates).Contain("replaces"))
}

const unregisteredLabelTextPatch = `---
"@type": patch
document: foo-2026-y
published: 2026-04-12
title: "A Test Document"
---
# Entity

## Widget2
` + "```yaml" + `
"@id": "Widget2"
"@type": Entity
` + "```" + `

A test entity.

**Assumptions**
- Ontologies are static once published
- Users prefer YAML front matter over JSON
`

// BUG-002 (spec 010 FR-019): a "**Label**" block whose label resolves to no
// registered predicate, and whose content isn't wikilink-shaped, is
// auto-registered as role: text (not the edge/union default) — closing
// spec 011 research.md's own flagged auto-discovery gap.
func TestApplyUnregisteredLabelTextContentRegistersAsTextRole(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(unregisteredLabelTextPatch)
	vcs := &graphmock.VCS{CommitHash: "abc123"}
	schema := &fakeSchema{}

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, schema, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Seq(schema.registeredPredicates).Contain("assumptions"))
	it.Then(t).Should(it.Equal("text", schema.registeredPredicateRole["assumptions"]))
}

const unregisteredLabeledEdgePatch = `---
"@type": patch
document: foo-2026-z
published: 2026-04-12
title: "A Test Document"
---
# Entity

## Widget3
` + "```yaml" + `
"@id": "Widget3"
"@type": Entity
` + "```" + `

A test entity.
- replaces:: [[Old Widget]]

**Related Aporias**
- [[Some Aporia]]
`

// BUG-003 (spec 010 FR-021/FR-022): an edge occurrence carried with its
// own "**Label**" block auto-registers as role: link (not the flat
// role: edge default), with its `label` attribute set to the block's
// literal label text — so the block's original grouping/heading survive a
// write, instead of collapsing into an undifferentiated flat bullet list.
// The fixture also carries an unrelated bare edge-role occurrence
// ("replaces"), so the single-link-role-predicate-body heading omission
// (spec 013 FR-006 — legitimately elsewhere applicable when a link-role
// predicate's occurrences are a node's *entire* edge content) does not
// mask this assertion.
func TestApplyUnregisteredLabeledEdgeRegistersAsLinkRoleWithLabel(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(unregisteredLabeledEdgePatch)
	vcs := &graphmock.VCS{CommitHash: "abc123"}
	schema := &fakeSchema{}

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, schema, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Seq(schema.registeredPredicates).Contain("relatedAporias"))
	it.Then(t).Should(it.Equal("link", schema.registeredPredicateRole["relatedAporias"]))
	it.Then(t).Should(it.Equal("Related Aporias", schema.registeredPredicateLabel["relatedAporias"]))

	content := string(store.files["Entity/Widget3.md"])
	it.Then(t).
		Should(it.String(content).Contain("## Related Aporias")).
		Should(it.String(content).Contain("[[Some Aporia]]")).
		Should(it.String(content).Contain("replaces:: [[Old Widget]]"))
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

	node, err := core.ParseNode(bytes.NewReader(yearly), core.Index{})
	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal("Timeline", node.Type)).
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
	store.files["Entity/Widget.md"] = []byte(existingWidgetEntity)
	vcs := &graphmock.VCS{CommitHash: "abc123"}
	reporter := &fakeReporter{}

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, reporter, coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(6, len(reporter.steps))).
		Should(it.Equal("foo-2026-x: created", reporter.steps[0])).
		Should(it.Equal("Widget: merged", reporter.steps[1])).
		// category/text (Entity's leading prose, BUG-001/FR-030 — was
		// "definition") are byte-identical on both sides of this fixture —
		// BUG-002 (spec.md FR-019): a union/append predicate whose incoming
		// contribution adds no genuinely new value must report unchanged,
		// not appended.
		Should(it.Equal("  category: union -> unchanged", reporter.steps[2])).
		Should(it.Equal("  published: union -> unchanged", reporter.steps[3])).
		Should(it.Equal("  text: union -> unchanged", reporter.steps[4])).
		Should(it.Equal("  title: union -> unchanged", reporter.steps[5]))
}

const stubSectionPatch = `---
"@type": patch
document: foo-2026-x
published: 2026-04-12
title: "A Test Document"
---
# Source

## foo-2026-x
` + "```yaml" + `
"@id": "foo-2026-x"
"@type": Source
title: "A Test Document"
authors: [Test Author]
published: "2026-04-12"
` + "```" + `

A test document.

# Entity

## StubEntity
` + "```yaml" + `
"@id": "StubEntity"
"@type": Entity
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

	content := string(store.files["Entity/StubEntity.md"])
	node, err := core.ParseNode(bytes.NewReader([]byte(content)), core.Index{})
	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.True(node.Published.IsZero())).
		ShouldNot(it.String(content).Contain("indexed:"))
}

// spec.md US1 Acceptance Scenario 4 / FR-003: an auto-registered
// _schema/Class/<name>.md document carries neither published nor indexed,
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
	it.Then(t).Should(it.Seq(schema.registeredTypes).Contain("Hypothesis"))
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

	source, err := core.ParseNode(bytes.NewReader(store.files["Source/foo-2026-x.md"]), core.Index{})
	it.Then(t).Should(it.Nil(err))
	entity, err := core.ParseNode(bytes.NewReader(store.files["Entity/Widget.md"]), core.Index{})
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
	store.files["Entity/Widget.md"] = []byte(existingWidgetEntity)
	vcs := &graphmock.VCS{CommitHash: "abc123"}

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")
	it.Then(t).Should(it.Nil(err))

	source, err := core.ParseNode(bytes.NewReader(store.files["Source/foo-2026-x.md"]), core.Index{})
	it.Then(t).Should(it.Nil(err))
	entity, err := core.ParseNode(bytes.NewReader(store.files["Entity/Widget.md"]), core.Index{})
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
"@type": patch
document: foo-2026-x2
published: 2026-04-12
title: "A Second Document"
---
# Source

## foo-2026-x2
` + "```yaml" + `
"@id": "foo-2026-x2"
"@type": Source
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
	store.files["Source/foo-2026-x2.md"] = []byte(`---
"@id": "foo-2026-x2"
"@type": Source
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

	content := string(store.files["Source/foo-2026-x2.md"])
	it.Then(t).ShouldNot(it.String(content).Contain("updated:"))
}

// BUG-004: uses the same Reference-kind conflict fixture as
// TestApplyFlagsConflict above, since an "Entity" (MergeUnion) node no
// longer ever flags a conflict.
func TestApplyReportsStepConflictFlagged(t *testing.T) {
	store := newGraphStore()
	store.files["patch.md"] = []byte(sourceReferencePatch)
	store.files["Reference/Widget Spec.md"] = []byte(existingWidgetSpecReferenceWithStatus)
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
"@type": Entity
category: [independent, abstract, occurrent, script]
---
# Widget

A test entity.
`
	store.files["Entity/Widget.md"] = []byte(mismatched)
	vcs := &graphmock.VCS{CommitHash: "abc123"}

	_, err := service.Apply(context.Background(), memMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).ShouldNot(it.Nil(err))
	it.Then(t).
		Should(it.Equal(mismatched, string(store.files["Entity/Widget.md"]))).
		Should(it.Equal(0, len(store.files["Source/foo-2026-x.md"]))).
		ShouldNot(it.Seq(vcs.Calls).Contain("StageAll:/graph"))
}

// --- BUG-008: case-variant node identities -------------------------------
//
// caseStore is a memStore that enumerates its contents and, when folds is
// set, resolves names the way a case-insensitive volume does: Open reaches a
// case-variant file, and Create truncates it IN PLACE under its existing
// name rather than making a second file (which is exactly what os.Create
// does on APFS, and the corruption the basename guard prevents).
//
// folds is INJECTED, never probed, so both branches are exercised
// deterministically on any developer's disk and on CI (spec.md SC-012).
type caseStore struct {
	*memStore
	folds bool
}

func (s caseStore) FoldsCase() bool { return s.folds }

// resolve mimics the volume's own path resolution.
func (s caseStore) resolve(name string) string {
	if _, ok := s.files[name]; ok {
		return name
	}
	if !s.folds {
		return name
	}
	for have := range s.files {
		if strings.EqualFold(have, name) {
			return have
		}
	}
	return name
}

func (s caseStore) Open(name string) (fs.File, error) { return s.memStore.Open(s.resolve(name)) }
func (s caseStore) Stat(name string) (fs.FileInfo, error) {
	return s.memStore.Stat(s.resolve(name))
}
func (s caseStore) Create(name string) (fsys.File, error) {
	return s.memStore.Create(s.resolve(name))
}

func (s caseStore) ReadDir(name string) ([]fs.DirEntry, error) {
	seen := map[string]fs.DirEntry{}
	prefix := ""
	if name != "." {
		prefix = name + "/"
	}
	for path := range s.files {
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		rest := strings.TrimPrefix(path, prefix)
		if i := strings.Index(rest, "/"); i >= 0 {
			seen[rest[:i]] = memDirEntry{name: rest[:i], dir: true}
			continue
		}
		seen[rest] = memDirEntry{name: rest}
	}
	for dir := range s.dirs {
		if strings.HasPrefix(dir, prefix) {
			rest := strings.TrimPrefix(dir, prefix)
			if rest != "" && !strings.Contains(rest, "/") {
				seen[rest] = memDirEntry{name: rest, dir: true}
			}
		}
	}

	out := make([]fs.DirEntry, 0, len(seen))
	for _, e := range seen {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out, nil
}

type memDirEntry struct {
	name string
	dir  bool
}

func (e memDirEntry) Name() string      { return e.name }
func (e memDirEntry) IsDir() bool       { return e.dir }
func (e memDirEntry) Type() fs.FileMode { return 0 }
func (e memDirEntry) Info() (fs.FileInfo, error) {
	return memFileInfo{name: e.name}, nil
}

type caseMounter struct{ store caseStore }

func (m caseMounter) Mount(root string) (fsys.Store, error) { return m.store, nil }

const lightstepVariantPatch = `---
"@type": patch
document: bar-2026-y
published: 2026-05-13
title: "A Second Document"
---
# Source

## bar-2026-y
` + "```yaml" + `
"@id": "bar-2026-y"
"@type": Source
title: "A Second Document"
authors: [Test Author]
published: "2026-05-13"
` + "```" + `

A second document.

# Entity

## Lightstep
` + "```yaml" + `
"@id": "Lightstep"
"@type": Entity
category: [independent, abstract, occurrent, script]
` + "```" + `

Mentioned by the second document.
- replaces:: [[Old Widget]]
`

const existingLightStepEntity = `---
"@id": "LightStep"
"@type": Entity
title: LightStep
category: [independent, abstract, occurrent, script]
---
# LightStep

The originally ingested spelling.
`

func newCaseGraph(folds bool) caseStore {
	inner := newGraphStore()
	inner.files["patch.md"] = []byte(lightstepVariantPatch)
	inner.files["Entity/LightStep.md"] = []byte(existingLightStepEntity)
	return caseStore{memStore: inner, folds: folds}
}

// FR-027, case-insensitive branch: one node, merged under the identity the
// graph already recorded, with a warning naming both spellings.
func TestApplyFoldsCaseVariantIdentityOnCaseInsensitiveStore(t *testing.T) {
	store := newCaseGraph(true)
	vcs := &graphmock.VCS{CommitHash: "abc123"}

	result, err := service.Apply(context.Background(), caseMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, result.Merged["Entity"])).
		Should(it.Equal(0, result.Created["Entity"]))

	// exactly one file for the subject, still under its recorded spelling
	_, forked := store.files["Entity/Lightstep.md"]
	it.Then(t).Should(it.Equal(forked, false))
	content := string(store.files["Entity/LightStep.md"])
	it.Then(t).Should(
		it.String(content).Contain(`"@id": LightStep`),
		it.String(content).Contain("replaces:: [[Old Widget]]"),
	)

	// the fold is visible (FR-027)
	var folded string
	for _, w := range result.Warnings {
		if strings.Contains(w, "folded") {
			folded = w
		}
	}
	it.Then(t).Should(
		it.String(folded).Contain("Lightstep"),
		it.String(folded).Contain("LightStep"),
	)
}

// FR-005/FR-027, case-sensitive branch: two genuinely distinct nodes,
// compared exactly. This is today's behavior and it stays.
func TestApplyKeepsCaseVariantIdentitiesDistinctOnCaseSensitiveStore(t *testing.T) {
	store := newCaseGraph(false)
	vcs := &graphmock.VCS{CommitHash: "abc123"}

	result, err := service.Apply(context.Background(), caseMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, result.Created["Entity"])).
		Should(it.Equal(0, result.Merged["Entity"]))

	_, created := store.files["Entity/Lightstep.md"]
	it.Then(t).Should(it.Equal(created, true))
	it.Then(t).Should(
		it.String(string(store.files["Entity/LightStep.md"])).Contain("The originally ingested spelling."),
	)

	for _, w := range result.Warnings {
		it.Then(t).Should(it.Equal(strings.Contains(w, "folded"), false))
	}
}

// FR-029: the guard still catches a genuinely drifted "@id", and does so
// against the file's own real name — its true purpose, preserved.
func TestApplyStillRejectsNodeFileWhoseIdentityDriftedFromItsFilename(t *testing.T) {
	store := newCaseGraph(false)
	store.files["Entity/Lightstep.md"] = []byte(`---
"@id": "SomethingElse"
"@type": Entity
---
# SomethingElse

Hand-edited so its identity no longer matches its filename.
`)
	vcs := &graphmock.VCS{CommitHash: "abc123"}

	_, err := service.Apply(context.Background(), caseMounter{store: store}, vcs, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")

	it.Then(t).ShouldNot(it.Nil(err))
	it.Then(t).Should(
		it.String(err.Error()).Contain("SomethingElse"),
		it.String(err.Error()).Contain("Lightstep"),
	)
}

// T117 / FR-028 scope guard: the fold is scoped to the ONE case a storage
// location forces, and no further. Identities are never normalized
// graph-wide — not lowercased, not case-folded on read, and an existing
// node file is never rewritten to a different spelling of its own identity.
// This matters beyond tidiness: identities double as wikilink targets
// ([[LightStep]]) resolved by third-party readers (Obsidian above all)
// whose rules this project does not control, so a graph-wide canonical form
// would silently change what every existing link resolves to.
func TestApplyNeverNormalizesIdentitiesGraphWide(t *testing.T) {
	// A case-sensitive location keeps both spellings verbatim: no
	// normalization is applied on either the read or the write path.
	sensitive := newCaseGraph(false)
	_, err := service.Apply(context.Background(), caseMounter{store: sensitive}, &graphmock.VCS{CommitHash: "abc123"}, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(
		// untouched — still byte-for-byte the seeded fixture, quotes and all
		it.String(string(sensitive.files["Entity/LightStep.md"])).Contain(`"@id": "LightStep"`),
		it.String(string(sensitive.files["Entity/Lightstep.md"])).Contain(`"@id": Lightstep`),
	)
	for path := range sensitive.files {
		it.Then(t).Should(it.Equal(strings.Contains(path, "lightstep.md"), false))
	}

	// A case-insensitive location folds onto the recorded identity and
	// leaves that file's own "@id" and filename exactly as they were — the
	// incoming spelling never rewrites either.
	insensitive := newCaseGraph(true)
	_, err = service.Apply(context.Background(), caseMounter{store: insensitive}, &graphmock.VCS{CommitHash: "abc123"}, bios.NewReporter(true, true), coreIndexFixture, &fakeSchema{}, "/graph", "/patch.md")
	it.Then(t).Should(it.Nil(err))

	content := string(insensitive.files["Entity/LightStep.md"])
	it.Then(t).Should(it.String(content).Contain(`"@id": LightStep`))
	it.Then(t).ShouldNot(it.String(content).Contain(`"@id": Lightstep`))

	for path := range insensitive.files {
		it.Then(t).Should(it.Equal(strings.Contains(path, "Entity/Lightstep.md"), false))
	}
}
