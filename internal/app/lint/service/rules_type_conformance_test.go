//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
	schemaservice "github.com/fogfish/arcnet-cli/internal/app/schema/service"
	"github.com/fogfish/arcnet-cli/internal/core"
)

var typeConformanceIndexFixture = core.Index{
	Types: map[string]core.TypeDef{
		"Source": {Required: []string{"title", "abstract"}, Optional: []string{"tags"}},
		"loose":  {},
	},
	Predicates: map[string]core.PredicateDef{
		"title":    {Role: "meta"},
		"abstract": {Role: "text"},
		"tags":     {Role: "meta"},
		"mentions": {Role: "link"},
		"noRole":   {},
	},
}

func TestCheckTypeRequiresPresentNoViolation(t *testing.T) {
	node := core.Node{Type: "Source", Attrs: map[string][]core.Predicate{"title": {{Value: "T"}}}, Texts: map[string]string{"abstract": "A"}}
	out := checkTypeRequires(node, "Source/x.md", []byte("---\ntitle: T\n---\n"), typeConformanceIndexFixture)
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckTypeRequiresAbsentReportsViolation(t *testing.T) {
	node := core.Node{Type: "Source", Attrs: map[string][]core.Predicate{"title": {{Value: "T"}}}}
	out := checkTypeRequires(node, "Source/x.md", []byte("---\ntitle: T\n---\n"), typeConformanceIndexFixture)
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).
		Should(it.Equal(kernel.RuleTypeRequires, out[0].Rule)).
		Should(it.String(out[0].Message).Contain("abstract")).
		Should(it.String(out[0].Message).Contain("Source"))
}

func TestCheckTypeRequiresUnregisteredTypeSkipped(t *testing.T) {
	node := core.Node{Type: "hypothesis"}
	out := checkTypeRequires(node, "hypothesis/x.md", []byte("---\n---\n"), typeConformanceIndexFixture)
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckTypeRequiresEmptyRequiredNeverViolates(t *testing.T) {
	node := core.Node{Type: "loose"}
	out := checkTypeRequires(node, "x.md", []byte("---\n---\n"), typeConformanceIndexFixture)
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckTypeOptionalListedPredicateNoViolation(t *testing.T) {
	node := core.Node{Type: "Source", Attrs: map[string][]core.Predicate{
		"title": {{Value: "T"}}, "tags": {{Value: "x"}},
	}, Texts: map[string]string{"abstract": "A"}}
	out := checkTypeOptional(node, "Source/x.md", []byte("---\ntitle: T\ntags: [x]\n---\n"), typeConformanceIndexFixture)
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckTypeOptionalUnlistedPredicateReportsViolation(t *testing.T) {
	node := core.Node{Type: "Source", Attrs: map[string][]core.Predicate{
		"title": {{Value: "T"}}, "extra": {{Value: "x"}},
	}, Texts: map[string]string{"abstract": "A"}}
	out := checkTypeOptional(node, "Source/x.md", []byte("---\ntitle: T\nextra: x\n---\n"), typeConformanceIndexFixture)
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).
		Should(it.Equal(kernel.RuleTypeOptional, out[0].Rule)).
		Should(it.String(out[0].Message).Contain("extra")).
		Should(it.String(out[0].Message).Contain("Source"))
}

func TestCheckTypeOptionalEmptyOptionalPermitsNothingExtra(t *testing.T) {
	node := core.Node{Type: "loose", Attrs: map[string][]core.Predicate{"whatever": {{Value: "x"}}}}
	out := checkTypeOptional(node, "x.md", []byte("---\nwhatever: x\n---\n"), typeConformanceIndexFixture)
	it.Then(t).Should(it.Equal(1, len(out)))
}

func TestCheckTypeOptionalPredicateListedUnderBothRequiredAndOptionalTolerated(t *testing.T) {
	index := core.Index{Types: map[string]core.TypeDef{
		"dup": {Required: []string{"tags"}, Optional: []string{"tags"}},
	}}
	node := core.Node{Type: "dup", Attrs: map[string][]core.Predicate{"tags": {{Value: "x"}}}}
	out := checkTypeOptional(node, "x.md", []byte("---\ntags: [x]\n---\n"), index)
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckPredicateRoleMatchingNoViolation(t *testing.T) {
	node := core.Node{Attrs: map[string][]core.Predicate{"title": {{Value: "T"}}}, Texts: map[string]string{"abstract": "A"}}
	out := checkPredicateRole(node, "x.md", []byte("---\ntitle: T\n---\n"), typeConformanceIndexFixture)
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckPredicateRoleTextPredicateAsEdgeReportsViolation(t *testing.T) {
	node := core.Node{Edges: []core.Link{{Predicate: "abstract", Target: "X"}}}
	out := checkPredicateRole(node, "x.md", []byte("- abstract:: [[X]]\n"), typeConformanceIndexFixture)
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).
		Should(it.Equal(kernel.RulePredicateRole, out[0].Rule)).
		Should(it.String(out[0].Message).Contain("abstract")).
		Should(it.String(out[0].Message).Contain("text")).
		Should(it.String(out[0].Message).Contain("edge"))
}

func TestCheckPredicateRoleLinkPredicateAsTextReportsViolation(t *testing.T) {
	node := core.Node{Texts: map[string]string{"mentions": "prose"}}
	out := checkPredicateRole(node, "x.md", []byte("mentions prose\n"), typeConformanceIndexFixture)
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).Should(it.Equal(kernel.RulePredicateRole, out[0].Rule))
}

func TestCheckPredicateRoleUnregisteredPredicateSkipped(t *testing.T) {
	node := core.Node{Edges: []core.Link{{Predicate: "unregisteredPred", Target: "X"}}}
	out := checkPredicateRole(node, "x.md", []byte("- unregisteredPred:: [[X]]\n"), typeConformanceIndexFixture)
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckPredicateRoleEmptyRoleSkipped(t *testing.T) {
	node := core.Node{Edges: []core.Link{{Predicate: "noRole", Target: "X"}}}
	out := checkPredicateRole(node, "x.md", []byte("- noRole:: [[X]]\n"), typeConformanceIndexFixture)
	it.Then(t).Should(it.Equal(0, len(out)))
}

// research.md D4: an inline citation-tagged HRefs occurrence of an edge-role
// predicate (e.g. citesAsEvidence) is exempt from the role check.
func TestCheckPredicateRoleCitationTaggedHRefExempt(t *testing.T) {
	index := core.Index{Predicates: map[string]core.PredicateDef{"citesAsEvidence": {Role: "edge"}}}
	node := core.Node{HRefs: []core.Link{{Predicate: "citesAsEvidence", Target: "X"}}}
	out := checkPredicateRole(node, "x.md", []byte("[citesAsEvidence:: [[X]]]\n"), index)
	it.Then(t).Should(it.Equal(0, len(out)))
}

// ---------------------------------------------------------------------------
// specs/023-core-vocabulary-conformance — User Story 3
//
// The fixtures above are synthetic: they declare their own type contracts,
// so they agree with themselves by construction and can say nothing about
// whether the SEEDED vocabulary is conformant. The assertions below close
// that gap by running the real checks over the hand-built v0.11 fixture
// graph, against the effective index Seed() -> Resolve() actually produces.
//
// CORE-FIX.md §5.7: neither half may come from the tool's own output. The
// nodes are hand-written (testdata/v011-graph); the contract they are
// checked against is the tool's — which is exactly where a disagreement
// would show up.
// ---------------------------------------------------------------------------

// v011Index resolves the built-in vocabulary the way a real graph does:
// Seed() rendered to disk, then read back through schema/service.Resolve,
// so the effective (inheritance-flattened) contract under test is the one
// arc lint would use.
func v011Index(t *testing.T) core.Index {
	t.Helper()
	dir := t.TempDir()

	for path, raw := range schemaservice.Seed() {
		full := filepath.Join(dir, filepath.FromSlash(path))
		it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Dir(full), 0o755)))
		it.Then(t).Should(it.Nil(os.WriteFile(full, raw, 0o644)))
	}
	it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Join(dir, ".arc"), 0o755)))

	store, err := (fsys.Local{}).Mount(dir)
	it.Then(t).Should(it.Nil(err))

	index, err := schemaservice.Resolve(store)
	it.Then(t).Should(it.Nil(err))
	return index
}

// v011FixtureNodes parses every content node in testdata/v011-graph.
func v011FixtureNodes(t *testing.T, index core.Index) map[string]core.Node {
	t.Helper()
	root := filepath.Join("testdata", "v011-graph")

	out := map[string]core.Node{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() == "README.md" {
			return err
		}
		raw, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		node, perr := core.ParseNode(bytes.NewReader(raw), index)
		if perr != nil {
			return perr
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return rerr
		}
		out[filepath.ToSlash(rel)] = node
		return nil
	})
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.True(len(out) == 5))
	return out
}

func v011FixtureRaw(t *testing.T, rel string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "v011-graph", filepath.FromSlash(rel)))
	it.Then(t).Should(it.Nil(err))
	return raw
}

// FR-007 through FR-009, FR-028, FR-029 / SC-004: a node carrying exactly
// the predicates ARCNET-CORE v0.11 requires of its type produces zero
// missing-required-predicate violations.
func TestCheckTypeRequiresV011FixtureIsClean(t *testing.T) {
	index := v011Index(t)

	for path, node := range v011FixtureNodes(t, index) {
		out := checkTypeRequires(node, path, v011FixtureRaw(t, path), index)
		for _, v := range out {
			t.Errorf("%s: unexpected typeRequires violation: %s", path, v.Message)
		}
	}
}

// FR-029 / contract C2.2c: every predicate a v0.11-shaped node carries is
// permitted by its own type — including a Timeline's optional granularity,
// period, and heading (US3 scenario 3).
func TestCheckTypeOptionalV011FixtureIsClean(t *testing.T) {
	index := v011Index(t)

	for path, node := range v011FixtureNodes(t, index) {
		out := checkTypeOptional(node, path, v011FixtureRaw(t, path), index)
		for _, v := range out {
			t.Errorf("%s: unexpected typeOptional violation: %s", path, v.Message)
		}
	}
}

// FR-007: the universal base requires NOTHING, so neither published nor
// created is inherited as a requirement by any type. Asserted over the
// effective contract directly — the inheritance flattening is where a
// reintroduced base requirement would hide.
func TestEffectiveContractsInheritNoProvenanceRequirement(t *testing.T) {
	index := v011Index(t)

	it.Then(t).Should(it.Equal(0, len(index.Types["Node"].Required)))

	for _, typ := range []string{"Entity", "Resource", "Timeline", "Reference"} {
		for _, predicate := range index.Types[typ].Required {
			if predicate == "published" || predicate == "created" {
				t.Errorf("type %s still requires %q by inheritance from Node", typ, predicate)
			}
		}
	}

	// FR-008: a Source requires published in its OWN right — the relaxation
	// must not become a blanket exemption (US3 scenario 4).
	requiresPublished := false
	for _, predicate := range index.Types["Source"].Required {
		if predicate == "published" {
			requiresPublished = true
		}
		if predicate == "created" {
			t.Errorf("type Source still requires \"created\"")
		}
	}
	it.Then(t).Should(it.True(requiresPublished))
}
