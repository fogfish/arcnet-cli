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
	"strings"
	"testing"
	"time"
	"unicode"

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
		node, err := core.ParseNode(bytes.NewReader(raw), core.Index{})
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

// spec 017 US1: every seeded content type's document carries an explicit
// subClassOf:: [[Node]] edge (data-model.md's reshaped-types table, aligned
// to rdfs:subClassOf via the predicate's own Aligned field), and Node.md
// itself carries none.
func TestSeedContentTypesCarrySubClassOfNodeEdge(t *testing.T) {
	seed := service.Seed()

	for _, name := range []string{"Source", "Entity", "Resource", "Timeline"} {
		raw, ok := seed[kernel.TypesDir+"/"+name+".md"]
		it.Then(t).Should(it.True(ok))
		it.Then(t).Should(it.String(string(raw)).Contain("subClassOf:: [[Node]]"))
	}

	nodeRaw, ok := seed[kernel.TypesDir+"/Node.md"]
	it.Then(t).Should(it.True(ok))
	it.Then(t).ShouldNot(it.String(string(nodeRaw)).Contain("- subClassOf::"))
}

// spec 019 FR-002/FR-003: every key Seed() produces under _schema/types/
// begins with an uppercase letter — a regression guard against a future
// built-in type being added with a lowercase-first-letter name.
func TestSeedTypeKeysAreAllCamelCase(t *testing.T) {
	seed := service.Seed()

	for path := range seed {
		if !strings.HasPrefix(path, kernel.TypesDir+"/") {
			continue
		}
		name := strings.TrimSuffix(strings.TrimPrefix(path, kernel.TypesDir+"/"), ".md")
		it.Then(t).Should(it.True(unicode.IsUpper([]rune(name)[0])))
	}
}

func TestResolveRoundTripsSeedOutput(t *testing.T) {
	store := newSeededStore()

	index, err := service.Resolve(store)

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(len(kernel.CorePredicateDefs), len(index.Predicates))).
		Should(it.Equal(len(kernel.CoreTypeDefs), len(index.Types)))

	entity, ok := index.Types["Entity"]
	it.Then(t).Should(it.True(ok))
	it.Then(t).
		Should(it.Equal(core.MergeUnion, entity.Merge)).
		Should(it.Seq(entity.Required).Equal("category", "definition", "mentionedIn", "published", "created")).
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

// spec 012 FR-020, Bugfix 018/BUG-001: a Class document with no merge
// field resolves successfully — the whole-node merge field is no longer
// consulted by reconciliation and is not validated as mandatory.
func TestResolveClassDocumentMissingMergeSucceeds(t *testing.T) {
	store := newSeededStore()
	store.files[kernel.TypesDir+"/Hypothesis.md"] = "---\n\"@id\": Hypothesis\n\"@type\": Class\n---\n# Hypothesis\n\nA conclusion distilled from sources.\n"

	index, err := service.Resolve(store)

	it.Then(t).Should(it.Nil(err))
	_, ok := index.Types["Hypothesis"]
	it.Then(t).Should(it.True(ok))
}

// spec 012 FR-020, Bugfix 018/BUG-001: a Property document's merge field
// remains mandatory — only Class-level validation is narrowed.
func TestResolvePropertyDocumentMissingMergeFails(t *testing.T) {
	store := newSeededStore()
	store.files[kernel.PredicatesDir+"/broken.md"] = "---\n\"@id\": broken\n\"@type\": Property\nrole: edge\n---\n# broken\n\nSome text.\n"

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

	created, err := service.RegisterPredicate(store, "relatesTo", "edge", "")
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.True(created))

	content := store.written[kernel.PredicatesDir+"/relatesTo.md"]
	it.Then(t).
		Should(it.String(content).Contain(`"@type": Property`)).
		Should(it.String(content).Contain("role: edge")).
		Should(it.String(content).Contain("merge: union"))

	created, err = service.RegisterPredicate(store, "relatesTo", "edge", "")
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.True(!created))
	it.Then(t).Should(it.Equal(1, len(store.written)))
}

// TestRegisterPredicateTextObservedDefaultsToAppend (Bugfix BUG-002, spec
// 010 FR-019): a predicate first observed as non-wikilink body content
// auto-registers as role: text, merge: append instead of the edge/union
// default, so its content merges correctly (rather than being coerced into
// edge shape) on a later re-apply.
func TestRegisterPredicateTextObservedDefaultsToAppend(t *testing.T) {
	store := newFakeStore(nil)

	created, err := service.RegisterPredicate(store, "assumptions", "text", "")
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.True(created))

	content := store.written[kernel.PredicatesDir+"/assumptions.md"]
	it.Then(t).
		Should(it.String(content).Contain(`"@type": Property`)).
		Should(it.String(content).Contain("role: text")).
		Should(it.String(content).Contain("merge: append"))
}

// TestRegisterPredicateLinkObservedGetsRoleLinkAndLabel (Bugfix BUG-003,
// spec 010 FR-021/FR-022): an edge occurrence carried with its own
// "**Label**" block auto-registers as role: link (not the flat role: edge
// default), with its `label` attribute set to the block's own literal
// label text, so the block's original grouping and heading survive a
// write.
func TestRegisterPredicateLinkObservedGetsRoleLinkAndLabel(t *testing.T) {
	store := newFakeStore(nil)

	created, err := service.RegisterPredicate(store, "relatedAporias", "link", "Related Aporias")
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.True(created))

	content := store.written[kernel.PredicatesDir+"/relatedAporias.md"]
	it.Then(t).
		Should(it.String(content).Contain(`"@type": Property`)).
		Should(it.String(content).Contain("role: link")).
		Should(it.String(content).Contain("merge: union")).
		Should(it.String(content).Contain("label: Related Aporias"))
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

	_, err := service.RegisterPredicate(store, "relatesTo", "edge", "")

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

// --- subClassOf (rdfs:subClassOf-aligned) resolution (spec 017) -----------

// nodeStub is a no-op Node type document: every custom type below implicitly
// inherits it (research.md D5's universal-base rule applies to any type
// named anything other than Node/Property/Class), and since it contributes
// no Required/Optional of its own, these tests observe only the explicit
// hierarchy under exercise, not the real seeded Node contract (already
// covered by TestSeedContentTypesCarrySubClassOfNodeEdge and the E2E tests).
const nodeStub = "---\n\"@id\": Node\n\"@type\": Class\nmerge: union\n---\n# Node\n\nStub base for resolver unit tests.\n"

// typeDoc renders a minimal Class-typed type schema document for resolver
// unit tests: a description paragraph, flat subClassOf bullets (no heading —
// Role \"edge\", per research.md D1), then headed Requires/Optional sections.
func typeDoc(id string, required, optional, subClassOf []string) string {
	var body strings.Builder
	body.WriteString("---\n\"@id\": " + id + "\n\"@type\": Class\nmerge: union\n---\n# " + id + "\n\n" + id + " under test.\n")
	for _, base := range subClassOf {
		body.WriteString("\n- subClassOf:: [[" + base + "]]\n")
	}
	if len(required) > 0 {
		body.WriteString("\n## Requires\n")
		for _, r := range required {
			body.WriteString("- required:: [[" + r + "]]\n")
		}
	}
	if len(optional) > 0 {
		body.WriteString("\n## Optional\n")
		for _, o := range optional {
			body.WriteString("- optional:: [[" + o + "]]\n")
		}
	}
	return body.String()
}

// newTypesStore builds a fake graph root whose _schema/types/ carries
// exactly the given documents, plus a no-op Node.md (unless the caller
// supplies its own), and an empty _schema/predicates/ dir — Resolve never
// cross-checks a type's Required/Optional/subClassOf targets against
// registered predicates.
func newTypesStore(types map[string]string) *fakeStore {
	store := newFakeStore(nil)
	store.dirs[kernel.PredicatesDir] = true
	store.dirs[kernel.TypesDir] = true
	if _, ok := types["Node"]; !ok {
		store.files[kernel.TypesDir+"/Node.md"] = nodeStub
	}
	for name, raw := range types {
		store.files[kernel.TypesDir+"/"+name+".md"] = raw
	}
	return store
}

func TestResolveSingleInheritanceNoOwnPredicates(t *testing.T) {
	store := newTypesStore(map[string]string{
		"C2": typeDoc("C2", []string{"p"}, []string{"q"}, nil),
		"C1": typeDoc("C1", nil, nil, []string{"C2"}),
	})

	index, err := service.Resolve(store)
	it.Then(t).Should(it.Nil(err))

	c1 := index.Types["C1"]
	it.Then(t).
		Should(it.Seq(c1.Required).Equal("p")).
		Should(it.Seq(c1.Optional).Equal("q"))
}

func TestResolveOwnPlusInheritedCombine(t *testing.T) {
	store := newTypesStore(map[string]string{
		"C2": typeDoc("C2", []string{"p"}, nil, nil),
		"C1": typeDoc("C1", []string{"r"}, nil, []string{"C2"}),
	})

	index, err := service.Resolve(store)
	it.Then(t).Should(it.Nil(err))

	c1 := index.Types["C1"]
	it.Then(t).Should(it.Seq(c1.Required).Equal("r", "p"))
}

func TestResolveMultipleBasesWithOverlappingPredicatesDedup(t *testing.T) {
	store := newTypesStore(map[string]string{
		"C2": typeDoc("C2", []string{"p"}, nil, nil),
		"C3": typeDoc("C3", []string{"p", "s"}, nil, nil),
		"C1": typeDoc("C1", nil, nil, []string{"C2", "C3"}),
	})

	index, err := service.Resolve(store)
	it.Then(t).Should(it.Nil(err))

	c1 := index.Types["C1"]
	it.Then(t).Should(it.Seq(c1.Required).Equal("p", "s"))
}

func TestResolveDuplicateBaseDeclarationCoalesces(t *testing.T) {
	store := newTypesStore(map[string]string{
		"C2": typeDoc("C2", []string{"p"}, nil, nil),
		"C1": typeDoc("C1", nil, nil, []string{"C2", "C2"}),
	})

	index, err := service.Resolve(store)
	it.Then(t).Should(it.Nil(err))

	c1 := index.Types["C1"]
	it.Then(t).Should(it.Seq(c1.Required).Equal("p"))
}

func TestResolveThreeLevelChainResolvesTransitively(t *testing.T) {
	store := newTypesStore(map[string]string{
		"C3": typeDoc("C3", []string{"t"}, nil, nil),
		"C2": typeDoc("C2", nil, nil, []string{"C3"}),
		"C1": typeDoc("C1", nil, nil, []string{"C2"}),
	})

	index, err := service.Resolve(store)
	it.Then(t).Should(it.Nil(err))

	c1 := index.Types["C1"]
	it.Then(t).Should(it.Seq(c1.Required).Equal("t"))
}

// Diamond hierarchy: A -> B -> D and A -> C -> D, D's own predicate must
// appear exactly once in A's effective contract (spec US3.2).
func TestResolveDiamondHierarchyDedupsCommonAncestor(t *testing.T) {
	store := newTypesStore(map[string]string{
		"D": typeDoc("D", []string{"d"}, nil, nil),
		"B": typeDoc("B", nil, nil, []string{"D"}),
		"C": typeDoc("C", nil, nil, []string{"D"}),
		"A": typeDoc("A", nil, nil, []string{"B", "C"}),
	})

	index, err := service.Resolve(store)
	it.Then(t).Should(it.Nil(err))

	a := index.Types["A"]
	it.Then(t).Should(it.Seq(a.Required).Equal("d"))
}

func TestResolveDirectSelfReferenceCycleFails(t *testing.T) {
	store := newTypesStore(map[string]string{
		"X": typeDoc("X", nil, nil, []string{"X"}),
	})

	_, err := service.Resolve(store)
	it.Then(t).Should(it.True(errors.Is(err, service.ErrSchemaCycle)))
}

func TestResolveLongerCycleFails(t *testing.T) {
	store := newTypesStore(map[string]string{
		"Cyc1": typeDoc("Cyc1", nil, nil, []string{"Cyc2"}),
		"Cyc2": typeDoc("Cyc2", nil, nil, []string{"Cyc1"}),
	})

	_, err := service.Resolve(store)
	it.Then(t).Should(it.True(errors.Is(err, service.ErrSchemaCycle)))
}

func TestResolveUnresolvedBaseTypeReferenceFails(t *testing.T) {
	store := newTypesStore(map[string]string{
		"W": typeDoc("W", nil, nil, []string{"NoSuchType"}),
	})

	_, err := service.Resolve(store)
	it.Then(t).Should(it.True(errors.Is(err, service.ErrSchemaUnresolvedBase)))
}

// A type whose _schema/types/ carries no Node.md of its own also fails
// unresolved-base — the implicit Node reference (research.md D5) is exactly
// as much a schema reference as an explicit one (data-model.md's Errors
// section, contracts/type-schema-document.md).
func TestResolveMissingImplicitNodeBaseFails(t *testing.T) {
	store := newFakeStore(nil)
	store.dirs[kernel.PredicatesDir] = true
	store.dirs[kernel.TypesDir] = true
	store.files[kernel.TypesDir+"/W.md"] = typeDoc("W", nil, nil, nil)

	_, err := service.Resolve(store)
	it.Then(t).Should(it.True(errors.Is(err, service.ErrSchemaUnresolvedBase)))
}

// A predicate required by an ancestor stays required in the descendant's
// effective contract even though the descendant's own declaration would
// otherwise leave it merely optional (spec FR-007).
func TestResolveRequiredWinsOverOptional(t *testing.T) {
	store := newTypesStore(map[string]string{
		"Base": typeDoc("Base", []string{"m"}, nil, nil),
		"Sub":  typeDoc("Sub", nil, []string{"m"}, []string{"Base"}),
	})

	index, err := service.Resolve(store)
	it.Then(t).Should(it.Nil(err))

	sub := index.Types["Sub"]
	it.Then(t).
		Should(it.Seq(sub.Required).Equal("m")).
		Should(it.Seq(sub.Optional).BeEmpty())
}

// A type declaring no rdfs:subClassOf relationship of its own still
// implicitly inherits Node (research.md D5) — its effective contract is not
// merely its own direct declaration.
func TestResolveNoExplicitBaseStillGetsImplicitNode(t *testing.T) {
	store := newTypesStore(map[string]string{
		"Node": typeDoc("Node", []string{"n"}, nil, nil),
		"Solo": typeDoc("Solo", []string{"r"}, nil, nil),
	})

	index, err := service.Resolve(store)
	it.Then(t).Should(it.Nil(err))

	solo := index.Types["Solo"]
	it.Then(t).Should(it.Seq(solo.Required).Equal("r", "n"))
}
