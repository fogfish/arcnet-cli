//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import (
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
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
	out := checkTypeRequires(node, "sources/x.md", []byte("---\ntitle: T\n---\n"), typeConformanceIndexFixture)
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckTypeRequiresAbsentReportsViolation(t *testing.T) {
	node := core.Node{Type: "Source", Attrs: map[string][]core.Predicate{"title": {{Value: "T"}}}}
	out := checkTypeRequires(node, "sources/x.md", []byte("---\ntitle: T\n---\n"), typeConformanceIndexFixture)
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
	out := checkTypeOptional(node, "sources/x.md", []byte("---\ntitle: T\ntags: [x]\n---\n"), typeConformanceIndexFixture)
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckTypeOptionalUnlistedPredicateReportsViolation(t *testing.T) {
	node := core.Node{Type: "Source", Attrs: map[string][]core.Predicate{
		"title": {{Value: "T"}}, "extra": {{Value: "x"}},
	}, Texts: map[string]string{"abstract": "A"}}
	out := checkTypeOptional(node, "sources/x.md", []byte("---\ntitle: T\nextra: x\n---\n"), typeConformanceIndexFixture)
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
