//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package core_test

import (
	"regexp"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/core"
)

// typeStatement mirrors --type's lowering (research.md D4): one statement,
// Target OR'd across every value.
func typeStatement(values ...string) core.Statement {
	return core.Statement{
		Predicate: core.Matcher{Values: []string{"type"}},
		Target:    core.Matcher{Values: values},
	}
}

// tagStatement mirrors one --tag repeat's lowering (research.md D4).
func tagStatement(value string) core.Statement {
	return core.Statement{
		Predicate: core.Matcher{Values: []string{"tags"}},
		Target:    core.Matcher{Values: []string{value}},
	}
}

// attrStatement mirrors one --attr name=value repeat's lowering.
func attrStatement(name, value string) core.Statement {
	return core.Statement{
		Predicate: core.Matcher{Values: []string{name}},
		Target:    core.Matcher{Values: []string{value}},
	}
}

// attrPatternStatement mirrors one --attr name~=pattern repeat's lowering.
func attrPatternStatement(name string, pattern *regexp.Regexp) core.Statement {
	return core.Statement{
		Predicate: core.Matcher{Values: []string{name}},
		Target:    core.Matcher{Patterns: []*regexp.Regexp{pattern}},
	}
}

// predicateStatement mirrors --predicate's lowering (research.md D10): a
// traversal constraint, Source/Target both wildcard.
func predicateStatement(values ...string) core.Statement {
	return core.Statement{Predicate: core.Matcher{Values: values}}
}

func TestFilterZeroValueMatchesEveryNode(t *testing.T) {
	node := core.Node{Type: "Entity", Attrs: map[string][]core.Predicate{"status": {{Value: "mature"}}}}

	it.Then(t).Should(it.True(core.Filter{}.Match(node)))
}

func TestFilterTypeStatementIsOR(t *testing.T) {
	source := core.Node{Type: "Source"}
	entity := core.Node{Type: "Entity"}
	resource := core.Node{Type: "Resource"}
	f := core.Filter{Statements: []core.Statement{typeStatement("Source", "Entity")}}

	it.Then(t).
		Should(it.True(f.Match(source))).
		Should(it.True(f.Match(entity))).
		Should(it.True(!f.Match(resource)))
}

func TestFilterTagStatementsAreAND(t *testing.T) {
	node := core.Node{Attrs: map[string][]core.Predicate{"tags": {{Value: "cryptography"}, {Value: "protocols"}}}}
	f := core.Filter{Statements: []core.Statement{tagStatement("cryptography"), tagStatement("protocols")}}
	fMissing := core.Filter{Statements: []core.Statement{tagStatement("cryptography"), tagStatement("unrelated")}}

	it.Then(t).
		Should(it.True(f.Match(node))).
		Should(it.True(!fMissing.Match(node)))
}

func TestFilterAttrExactMatchCaseInsensitiveScalar(t *testing.T) {
	node := core.Node{Attrs: map[string][]core.Predicate{"status": {{Value: "Mature"}}}}
	f := core.Filter{Statements: []core.Statement{attrStatement("status", "mature")}}
	fMismatch := core.Filter{Statements: []core.Statement{attrStatement("status", "backlog")}}

	it.Then(t).
		Should(it.True(f.Match(node))).
		Should(it.True(!fMismatch.Match(node)))
}

func TestFilterAttrExactMatchArrayMembership(t *testing.T) {
	node := core.Node{Attrs: map[string][]core.Predicate{"category": {{Value: "independent"}, {Value: "abstract"}}}}
	f := core.Filter{Statements: []core.Statement{attrStatement("category", "abstract")}}
	fMismatch := core.Filter{Statements: []core.Statement{attrStatement("category", "relative")}}

	it.Then(t).
		Should(it.True(f.Match(node))).
		Should(it.True(!fMismatch.Match(node)))
}

func TestFilterAttrPatternMatchScalarAndArray(t *testing.T) {
	scalarNode := core.Node{Attrs: map[string][]core.Predicate{"title": {{Value: "TLS 1.3: Design and Rationale"}}}}
	arrayNode := core.Node{Attrs: map[string][]core.Predicate{"category": {{Value: "independent"}, {Value: "abstract"}}}}
	f := core.Filter{Statements: []core.Statement{attrPatternStatement("title", regexp.MustCompile(`^TLS 1\.3`))}}
	fArray := core.Filter{Statements: []core.Statement{attrPatternStatement("category", regexp.MustCompile(`^abs`))}}
	fMismatch := core.Filter{Statements: []core.Statement{attrPatternStatement("title", regexp.MustCompile(`^SSL`))}}

	it.Then(t).
		Should(it.True(f.Match(scalarNode))).
		Should(it.True(fArray.Match(arrayNode))).
		Should(it.True(!fMismatch.Match(scalarNode)))
}

func TestFilterCombinedStatementsAreANDed(t *testing.T) {
	node := core.Node{
		Type: "Entity",
		Attrs: map[string][]core.Predicate{
			"tags":   {{Value: "cryptography"}},
			"status": {{Value: "mature"}},
		},
	}
	f := core.Filter{Statements: []core.Statement{
		typeStatement("Entity"),
		tagStatement("cryptography"),
		attrStatement("status", "mature"),
		attrPatternStatement("status", regexp.MustCompile(`^mat`)),
	}}
	fWrongType := core.Filter{Statements: []core.Statement{typeStatement("Resource"), tagStatement("cryptography")}}

	it.Then(t).
		Should(it.True(f.Match(node))).
		Should(it.True(!fWrongType.Match(node)))
}

func TestFilterMatchingZeroNodes(t *testing.T) {
	node := core.Node{Type: "Source"}
	f := core.Filter{Statements: []core.Statement{typeStatement("Resource")}}

	it.Then(t).Should(it.True(!f.Match(node)))
}

func TestFilterAttrListValuedSingleValue(t *testing.T) {
	node := core.Node{Attrs: map[string][]core.Predicate{"status": {{Value: "mature"}}}}
	f := core.Filter{Statements: []core.Statement{attrStatement("status", "mature")}}

	it.Then(t).Should(it.True(f.Match(node)))
}

func TestFilterAttrListValuedMultipleValuesMatchesAny(t *testing.T) {
	node := core.Node{Attrs: map[string][]core.Predicate{"category": {{Value: "independent"}, {Value: "abstract"}, {Value: "protocol"}}}}
	f := core.Filter{Statements: []core.Statement{attrStatement("category", "protocol")}}

	it.Then(t).Should(it.True(f.Match(node)))
}

func TestFilterAttrListValuedNoMatch(t *testing.T) {
	node := core.Node{Attrs: map[string][]core.Predicate{"category": {{Value: "independent"}, {Value: "abstract"}}}}
	f := core.Filter{Statements: []core.Statement{attrStatement("category", "relative")}}

	it.Then(t).Should(it.True(!f.Match(node)))
}

// --- User Story 2: edge-fact matching ---

func TestFilterMatchesViaEdgeFactAlone(t *testing.T) {
	node := core.Node{ID: "paper-a", Edges: []core.Link{{Predicate: "cites", Target: "paper-b"}}}
	f := core.Filter{Statements: []core.Statement{{
		Predicate: core.Matcher{Values: []string{"cites"}},
		Target:    core.Matcher{Values: []string{"paper-b"}},
	}}}
	fMismatch := core.Filter{Statements: []core.Statement{{
		Predicate: core.Matcher{Values: []string{"cites"}},
		Target:    core.Matcher{Values: []string{"paper-c"}},
	}}}

	it.Then(t).
		Should(it.True(f.Match(node))).
		Should(it.True(!fMismatch.Match(node)))
}

func TestFilterMatchesViaAttributeFactAlone(t *testing.T) {
	node := core.Node{ID: "paper-b", Attrs: map[string][]core.Predicate{"status": {{Value: "mature"}}}}
	f := core.Filter{Statements: []core.Statement{attrStatement("status", "mature")}}

	it.Then(t).Should(it.True(f.Match(node)))
}

// spec FR-005 wording: a combined filter where different statements are
// satisfied by different facts on the same node.
func TestFilterCombinedStatementsSatisfiedByDifferentFactKinds(t *testing.T) {
	node := core.Node{
		ID:    "paper-a",
		Attrs: map[string][]core.Predicate{"status": {{Value: "mature"}}},
		Edges: []core.Link{{Predicate: "cites", Target: "paper-b"}},
	}
	f := core.Filter{Statements: []core.Statement{
		attrStatement("status", "mature"),
		{Predicate: core.Matcher{Values: []string{"cites"}}, Target: core.Matcher{Values: []string{"paper-b"}}},
	}}
	fUnsatisfiedEdge := core.Filter{Statements: []core.Statement{
		attrStatement("status", "mature"),
		{Predicate: core.Matcher{Values: []string{"cites"}}, Target: core.Matcher{Values: []string{"paper-z"}}},
	}}

	it.Then(t).
		Should(it.True(f.Match(node))).
		Should(it.True(!fUnsatisfiedEdge.Match(node)))
}

// T037: --type/--tag/--attr-derived statements always constrain Target
// (research.md D2/D4), so a node's unrelated edges (different predicate,
// different target) never accidentally satisfy them — the edge-fact
// addition (T034) only ever adds new ways to match, never a spurious one
// for a statement whose Target the edge doesn't also satisfy.
func TestFilterTypeTagAttrStatementsUnaffectedByUnrelatedEdge(t *testing.T) {
	node := core.Node{
		ID:    "paper-a",
		Type:  "Source",
		Edges: []core.Link{{Predicate: "cites", Target: "paper-b"}},
	}

	it.Then(t).
		Should(it.True(core.Filter{Statements: []core.Statement{typeStatement("Source")}}.Match(node))). // sanity: still matches via the synthesized type fact
		ShouldNot(it.True(core.Filter{Statements: []core.Statement{attrStatement("status", "mature")}}.Match(node)))
}

// --- User Story 1: Traversal()/Narrowing() partition ---

func TestStatementIsTraversalConstraintOnlyWhenSourceAndTargetWildcard(t *testing.T) {
	it.Then(t).
		Should(it.True(predicateStatement("cites").IsTraversalConstraint())).
		Should(it.True(!attrStatement("status", "mature").IsTraversalConstraint())).
		Should(it.True(!typeStatement("Source").IsTraversalConstraint()))
}

func TestFilterTraversalNarrowingPartitionIsExhaustiveAndDisjoint(t *testing.T) {
	f := core.Filter{Statements: []core.Statement{
		predicateStatement("cites"),
		typeStatement("Source"),
		tagStatement("cryptography"),
	}}

	traversal := f.Traversal()
	narrowing := f.Narrowing()

	it.Then(t).
		Should(it.Equal(1, len(traversal.Statements))).
		Should(it.Equal(2, len(narrowing.Statements))).
		Should(it.Equal(len(f.Statements), len(traversal.Statements)+len(narrowing.Statements)))
}

// Narrowing() of a filter containing only a traversal constraint is
// vacuously true for every node — same posture as core.Filter{} today.
func TestFilterNarrowingOfPureTraversalFilterIsVacuouslyTrue(t *testing.T) {
	f := core.Filter{Statements: []core.Statement{predicateStatement("cites")}}

	it.Then(t).
		Should(it.Equal(0, len(f.Narrowing().Statements))).
		Should(it.True(f.Narrowing().Match(core.Node{Type: "Anything"})))
}
