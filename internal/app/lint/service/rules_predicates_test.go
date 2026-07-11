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

func TestCheckPredicateCaseValid(t *testing.T) {
	node := core.Node{Edges: []core.Link{{Predicate: "mentions", Target: "X"}}}
	out := checkPredicateCase(node, "x.md", []byte("- mentions:: [[X]]\n"))
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckPredicateCaseInvalid(t *testing.T) {
	node := core.Node{Edges: []core.Link{{Predicate: "Mentions-Bad", Target: "X"}}}
	out := checkPredicateCase(node, "x.md", []byte("- Mentions-Bad:: [[X]]\n"))
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).Should(it.Equal(kernel.RulePredicateCase, out[0].Rule))
}

func TestCheckPredicateCaseDedupSamePredicateTwice(t *testing.T) {
	node := core.Node{Edges: []core.Link{
		{Predicate: "BadOne", Target: "X"},
		{Predicate: "BadOne", Target: "Y"},
	}}
	out := checkPredicateCase(node, "x.md", []byte("- BadOne:: [[X]]\n- BadOne:: [[Y]]\n"))
	it.Then(t).Should(it.Equal(1, len(out)))
}

func TestCheckPredicateRegisteredPresent(t *testing.T) {
	node := core.Node{Edges: []core.Link{{Predicate: "mentions", Target: "X"}}}
	registry := map[string]core.PredicateDef{"mentions": {}}
	out := checkPredicateRegistered(node, "x.md", []byte("- mentions:: [[X]]\n"), registry)
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckPredicateRegisteredAbsent(t *testing.T) {
	node := core.Node{Edges: []core.Link{{Predicate: "unregisteredPred", Target: "X"}}}
	out := checkPredicateRegistered(node, "x.md", []byte("- unregisteredPred:: [[X]]\n"), map[string]core.PredicateDef{})
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).Should(it.Equal(kernel.RulePredicateRegistered, out[0].Rule))
}

// Edges is the single flat collection: predicates that formerly lived
// under distinct grouped-link headings (e.g. "mentions" and
// "citesAsEvidence") now interleave as plain Edges entries, and the
// registered-predicate check still flags whichever one is unregistered.
func TestCheckPredicateRegisteredFromFormerlyDistinctGroups(t *testing.T) {
	node := core.Node{Edges: []core.Link{
		{Predicate: "mentions", Target: "X"},
		{Predicate: "citesAsEvidence", Target: "Y"},
	}}
	raw := []byte("- mentions:: [[X]]\n- citesAsEvidence:: [[Y]]\n")
	out := checkPredicateRegistered(node, "x.md", raw, map[string]core.PredicateDef{"mentions": {}})
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).
		Should(it.Equal(kernel.RulePredicateRegistered, out[0].Rule)).
		Should(it.String(out[0].Message).Contain("citesAsEvidence"))
}

var citationRegistryFixture = map[string]core.PredicateDef{
	"cites":           {Aligned: "cito:cites"},
	"randomPredicate": {Aligned: "schema:randomPredicate"},
}

func TestCheckCitationPredicateValid(t *testing.T) {
	node := core.Node{HRefs: []core.Link{{Predicate: "cites", Target: "RFC 8446"}}}
	out := checkCitationPredicate(node, "x.md", []byte("[cites:: [[RFC 8446]]]\n"), citationRegistryFixture)
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckCitationPredicateInvalid(t *testing.T) {
	node := core.Node{HRefs: []core.Link{{Predicate: "randomPredicate", Target: "RFC 8446"}}}
	out := checkCitationPredicate(node, "x.md", []byte("[randomPredicate:: [[RFC 8446]]]\n"), citationRegistryFixture)
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).Should(it.Equal(kernel.RuleCitationPredicate, out[0].Rule))
}

func TestCheckCitationPredicateBareLinkExempt(t *testing.T) {
	node := core.Node{HRefs: []core.Link{{Target: "Widget"}}}
	out := checkCitationPredicate(node, "x.md", []byte("[[Widget]]\n"), citationRegistryFixture)
	it.Then(t).Should(it.Equal(0, len(out)))
}

// spec User Story 4 Acceptance Scenario 1: a domain-registered cito:-aligned
// predicate not in the old hardcoded list is accepted, since the vocabulary
// is now sourced from the graph's own schema, not a fixed Go list.
func TestCheckCitationPredicateDomainRegisteredCitoAlignedAccepted(t *testing.T) {
	node := core.Node{HRefs: []core.Link{{Predicate: "citesAsExample", Target: "Widget"}}}
	registry := map[string]core.PredicateDef{"citesAsExample": {Aligned: "cito:citesAsExample"}}
	out := checkCitationPredicate(node, "x.md", []byte("[citesAsExample:: [[Widget]]]\n"), registry)
	it.Then(t).Should(it.Equal(0, len(out)))
}

// spec User Story 4 Acceptance Scenario 2: a registered but non-cito:-
// aligned predicate is still rejected.
func TestCheckCitationPredicateRegisteredNonCitoAlignedRejected(t *testing.T) {
	node := core.Node{HRefs: []core.Link{{Predicate: "mentions", Target: "Widget"}}}
	registry := map[string]core.PredicateDef{"mentions": {Aligned: "schema:mentions"}}
	out := checkCitationPredicate(node, "x.md", []byte("[mentions:: [[Widget]]]\n"), registry)
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).Should(it.Equal(kernel.RuleCitationPredicate, out[0].Rule))
}

// spec User Story 4 Acceptance Scenario 3: a graph with zero cito:-aligned
// predicates rejects every citation usage — no built-in fallback vocabulary.
func TestCheckCitationPredicateEmptyRegistryRejectsEverything(t *testing.T) {
	node := core.Node{HRefs: []core.Link{{Predicate: "cites", Target: "Widget"}}}
	out := checkCitationPredicate(node, "x.md", []byte("[cites:: [[Widget]]]\n"), map[string]core.PredicateDef{})
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).Should(it.Equal(kernel.RuleCitationPredicate, out[0].Rule))
}
