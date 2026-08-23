//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package kernel_test

import (
	"regexp"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/schema/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

var camelCasePattern = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

var validRoles = map[string]bool{"meta": true, "text": true, "href": true, "edge": true, "link": true}

func TestCorePredicateDefsContainsFullCoreVocabulary(t *testing.T) {
	names := []string{
		"tags", "text",
		"published", "created", "updated", "indexed",
		"scoreZ", "scoreC",
		"mentions", "mentionedIn",
		"broader", "narrower", "isPartOf", "hasPart", "requires", "replaces", "isReplacedBy", "conformsTo", "related", "referencedBy",
		"cites", "citesAsEvidence", "citesAsAuthority", "supports", "confirms", "extends", "critiques", "disputes", "refutes", "isCitedBy",
		"title", "abstract", "authors", "url", "doi", "category", "aliases", "definition", "notes", "ref", "year", "status", "relevance", "granularity", "period", "heading",
		"role", "merge", "label", "aligned", "description", "required", "optional",
		"subClassOf",
	}

	it.Then(t).Should(it.Equal(len(names), len(kernel.CorePredicateDefs)))

	for _, name := range names {
		def, ok := kernel.CorePredicateDefs[name]
		it.Then(t).Should(it.True(ok))
		it.Then(t).
			Should(it.True(validRoles[def.Role])).
			ShouldNot(it.Equal("", string(def.Merge))).
			ShouldNot(it.Equal("", def.Description))
	}
}

func TestCorePredicateDefNamesAreCamelCase(t *testing.T) {
	for name := range kernel.CorePredicateDefs {
		it.Then(t).Should(it.True(camelCasePattern.MatchString(name)))
	}
}

func TestCoreTypeDefsContainsCoreTypesAndSchemaTypesThemselves(t *testing.T) {
	it.Then(t).Should(it.Equal(8, len(kernel.CoreTypeDefs)))

	for _, name := range []string{"Source", "Entity", "Resource", "Timeline", "Reference", "Node", "Property", "Class"} {
		def, ok := kernel.CoreTypeDefs[name]
		it.Then(t).Should(it.True(ok))
		it.Then(t).ShouldNot(it.Equal("", def.Description))
	}
}

// source/entity/resource/timeline's own directly declared Required lists
// (spec 017, data-model.md's reshaped-types table) — published/created now
// arrive only via the implicit Node base, never listed directly here.
func TestCoreTypeDefsRequiredListsMatchCoreSection11(t *testing.T) {
	source := kernel.CoreTypeDefs["Source"]
	it.Then(t).Should(it.Seq(source.Required).Equal("title", "abstract", "mentions"))

	entity := kernel.CoreTypeDefs["Entity"]
	it.Then(t).Should(it.Seq(entity.Required).Equal("category", "definition", "mentionedIn"))

	// CORE §11.4 v0.11: Resource is a fragment of an *ingested* document,
	// so it requires its own prose, its tag classification, and a backlink
	// to the document it was drawn from. The external-work predicates it
	// used to require moved to Reference (specs/022-reference-type-folders).
	resource := kernel.CoreTypeDefs["Resource"]
	it.Then(t).Should(it.Seq(resource.Required).Equal("text", "tags", "mentionedIn"))

	// CORE §11.6 v0.11: Reference records only enough to identify, locate,
	// and justify keeping a pointer to a work the graph has not ingested.
	reference := kernel.CoreTypeDefs["Reference"]
	it.Then(t).Should(it.Seq(reference.Required).Equal("title", "ref", "relevance"))

	// timeline deliberately diverges from CORE §11.5 here (BUG-002,
	// research.md D12): "entries" is replaced by "cites" (reusing the
	// existing citation predicate rather than the name CORE's own worked
	// example uses), and "period" is an arc-internal addition CORE never
	// documents (spec 003 BUG-007).
	timeline := kernel.CoreTypeDefs["Timeline"]
	it.Then(t).Should(it.Seq(timeline.Required).Equal("granularity", "cites", "period"))

	node := kernel.CoreTypeDefs["Node"]
	it.Then(t).Should(it.Seq(node.Required).Equal("published", "created"))
}

// BUG-001 / spec.md FR-014-FR-020, research.md D8: cross-cutting Structural
// (mentions, mentionedIn) and — for entity/resource — Semantic (§10.5)
// predicates MUST be listed under every relevant core type's Optional list,
// not just Required, so a real node using one of them is never falsely
// reported as not-permitted by checkTypeOptional. This is the closed test
// gap: TestCoreTypeDefsRequiredListsMatchCoreSection11 only ever asserted
// Required, never Optional. Content (tags, text) and Metadata/Control
// (published, created, updated, scoreZ, scoreC) predicates are no longer
// listed directly here (spec 017) — they arrive via the implicit Node base
// (TestCoreTypeDefsRequiredListsMatchCoreSection11's Node case, and
// internal/app/schema/service's resolver tests for the effective contract).
func TestCoreTypeDefsOptionalListsIncludeCrossCuttingPredicates(t *testing.T) {
	semantic := []string{"broader", "narrower", "isPartOf", "hasPart", "requires", "replaces", "isReplacedBy", "conformsTo", "related", "referencedBy"}

	tests := []struct {
		typ  string
		want []string
	}{
		{"Source", []string{"authors", "url", "cites", "doi", "indexed"}},
		{"Entity", append([]string{"aliases", "notes", "indexed", "mentions"}, semantic...)},
		// CORE §11.4 v0.11 lists notes as Resource's sole optional: the
		// structural and semantic optionals it used to carry are not
		// carried over, and §11.6 gives Reference none of them either
		// (specs/022-reference-type-folders, data-model.md §1.1). Both
		// keep "indexed", which is not a CORE predicate but the arc
		// extension arc apply stamps on every node it creates — every
		// other content type already carries it for the same reason.
		{"Resource", []string{"notes", "indexed"}},
		{"Timeline", []string{"heading", "indexed", "mentions", "mentionedIn"}},
		{"Reference", []string{"url", "authors", "year", "doi", "status", "isCitedBy", "notes", "indexed"}},
	}

	for _, tc := range tests {
		def := kernel.CoreTypeDefs[tc.typ]
		it.Then(t).Should(it.Seq(def.Optional).Equal(tc.want...))
	}
}

// Node's own Optional list (spec 017, data-model.md).
func TestCoreTypeDefsNodeOptionalList(t *testing.T) {
	node := kernel.CoreTypeDefs["Node"]
	it.Then(t).Should(it.Seq(node.Optional).Equal("tags", "text", "updated", "scoreZ", "scoreC"))
}

// Every content type declares an explicit rdfs:subClassOf base pointing at
// Node (spec 017, data-model.md) — redundant with the implicit rule but
// written for the seeded document's own self-description.
func TestCoreTypeBasesWireContentTypesToNode(t *testing.T) {
	for _, name := range []string{"Source", "Entity", "Resource", "Timeline", "Reference"} {
		it.Then(t).Should(it.Seq(kernel.CoreTypeBases[name]).Equal("Node"))
	}
	_, hasNode := kernel.CoreTypeBases["Node"]
	_, hasProperty := kernel.CoreTypeBases["Property"]
	_, hasClass := kernel.CoreTypeBases["Class"]
	it.Then(t).
		Should(it.True(!hasNode)).
		Should(it.True(!hasProperty)).
		Should(it.True(!hasClass))
}

// BUG-001 / spec.md FR-014-FR-020: every registered instance of the seed
// data's cross-cutting predicates is present in CorePredicateDefs itself
// (registration), not only referenced by a type's Optional list.
func TestCorePredicateDefsIndexedAndScorePredicatesAreRegistered(t *testing.T) {
	for _, name := range []string{"indexed", "scoreZ", "scoreC"} {
		def, ok := kernel.CorePredicateDefs[name]
		it.Then(t).Should(it.True(ok))
		it.Then(t).Should(it.Equal("meta", def.Role))
	}

	it.Then(t).Should(it.Equal(core.MergeImmutable, kernel.CorePredicateDefs["indexed"].Merge))
	it.Then(t).
		Should(it.Equal(core.MergeValidatedOverwrite, kernel.CorePredicateDefs["scoreZ"].Merge)).
		Should(it.Equal(core.MergeValidatedOverwrite, kernel.CorePredicateDefs["scoreC"].Merge))
}

// BUG-001 / spec.md FR-018: every role:"text" predicate in the built-in
// vocabulary seeds MergeAppend — role alone must be enough to predict
// dispatch, without reading each predicate's individual assignment.
func TestCorePredicateDefsTextRoleSeedsAppend(t *testing.T) {
	for _, def := range kernel.CorePredicateDefs {
		if def.Role != "text" {
			continue
		}
		it.Then(t).Should(it.Equal(core.MergeAppend, def.Merge))
	}
}

// ---------------------------------------------------------------------------
// specs/022-reference-type-folders — ARCNET-CORE v0.11
// ---------------------------------------------------------------------------

// TestReferencePredicatesAreAllRegistered enforces data-model.md §4 rule 5
// and spec.md FR-006: every predicate Reference names in either list is
// itself a key of CorePredicateDefs, so arc init seeds a Property document
// for each and a freshly initialized graph lints clean.
//
// This feature introduces no new predicate — Reference's whole vocabulary is
// the set displaced from Resource, plus title, all of which CORE §10 already
// documented. A type declaring a predicate nobody seeded is exactly the
// defect a new core type is most likely to introduce, and it surfaces at
// lint time on a graph the tool itself created.
func TestReferencePredicatesAreAllRegistered(t *testing.T) {
	reference := kernel.CoreTypeDefs["Reference"]

	for _, name := range append(append([]string{}, reference.Required...), reference.Optional...) {
		_, ok := kernel.CorePredicateDefs[name]
		it.Then(t).Should(it.True(ok))
	}
}

// TestEveryCoreTypePredicateIsRegistered generalizes the rule above to the
// whole seeded vocabulary: no type may name a predicate the seed does not
// also register, whichever type it is. Reference is the reason to write it,
// but the invariant was always meant to hold.
func TestEveryCoreTypePredicateIsRegistered(t *testing.T) {
	for _, def := range kernel.CoreTypeDefs {
		for _, name := range append(append([]string{}, def.Required...), def.Optional...) {
			_, ok := kernel.CorePredicateDefs[name]
			it.Then(t).Should(it.True(ok))
		}
	}
}

// TestResourceDeclaresNoExternalWorkPredicate pins contract C2's negative
// half: the eight predicates that moved to Reference appear in neither of
// Resource's lists. A Resource that still offered them would accept a node
// shaped like the retired definition without complaint, which is precisely
// the non-conformance this feature exists to end.
func TestResourceDeclaresNoExternalWorkPredicate(t *testing.T) {
	resource := kernel.CoreTypeDefs["Resource"]
	declared := append(append([]string{}, resource.Required...), resource.Optional...)

	for _, retired := range []string{"ref", "relevance", "url", "authors", "year", "doi", "status", "isCitedBy"} {
		for _, got := range declared {
			it.Then(t).Should(it.True(got != retired))
		}
	}
}
