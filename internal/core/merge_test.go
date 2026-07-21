//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package core_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/core"
)

// indexWith builds a minimal core.Index declaring a single Attrs/Texts-role
// predicate against op — every test below only ever needs one declared
// predicate at a time (data-model.md's per-shape × per-op reconciliation
// table is the authoritative source for every case here).
func indexWith(predicate string, op core.MergeOp) core.Index {
	return core.Index{Predicates: map[string]core.PredicateDef{predicate: {Merge: op}}}
}

// --- FR-001/FR-013: per-predicate independence, not by node type ---

// spec.md US1 Acceptance Scenario 4: a single merge simultaneously
// dispatches an immutable, a lastWriteWin, and a union predicate on the
// same node, each producing its own rule's outcome independently.
func TestMergeThreePredicatesEachFollowOwnRuleInOneApplication(t *testing.T) {
	index := core.Index{Predicates: map[string]core.PredicateDef{
		"ref":    {Merge: core.MergeImmutable},
		"status": {Merge: core.MergeLastWriteWin},
		"tags":   {Merge: core.MergeUnion},
	}}
	existing := core.Node{ID: "x", Type: "Resource", Attrs: map[string][]core.Predicate{
		"ref":    {{Value: "book"}},
		"status": {{Value: "backlog"}},
		"tags":   {{Value: "ai"}},
	}}
	incoming := core.Node{ID: "x", Type: "Resource", Attrs: map[string][]core.Predicate{
		"ref":    {{Value: "article"}},
		"status": {{Value: "read"}},
		"tags":   {{Value: "ml"}},
	}}

	merged, conflicts, _, err := core.Merge(existing, incoming, index, "patch-2")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equiv(merged.Attrs["ref"], []core.Predicate{{Value: "book"}})).
		Should(it.Equiv(merged.Attrs["status"], []core.Predicate{{Value: "read"}})).
		Should(it.Equiv(merged.Attrs["tags"], []core.Predicate{{Value: "ai"}, {Value: "ml"}}))
}

// --- immutable (freeze class) ---

func TestMergeImmutableRejectsLaterDivergingValueNoConflict(t *testing.T) {
	index := indexWith("ref", core.MergeImmutable)
	existing := core.Node{ID: "x", Type: "Resource", Attrs: map[string][]core.Predicate{"ref": {{Value: "book"}}}}
	incoming := core.Node{ID: "x", Type: "Resource", Attrs: map[string][]core.Predicate{"ref": {{Value: "article"}}}}

	merged, conflicts, _, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equiv(merged.Attrs["ref"], []core.Predicate{{Value: "book"}}))
}

func TestMergeImmutableAcceptsFirstValueWhenExistingEmpty(t *testing.T) {
	index := indexWith("ref", core.MergeImmutable)
	existing := core.Node{ID: "x", Type: "Resource"}
	incoming := core.Node{ID: "x", Type: "Resource", Attrs: map[string][]core.Predicate{"ref": {{Value: "book"}}}}

	merged, conflicts, _, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equiv(merged.Attrs["ref"], []core.Predicate{{Value: "book"}}))
}

func TestMergeImmutableTextsBehavesSameAsAttrs(t *testing.T) {
	index := indexWith("definition", core.MergeImmutable)
	existing := core.Node{ID: "x", Type: "Entity", Texts: map[string]string{"definition": "original"}}
	incoming := core.Node{ID: "x", Type: "Entity", Texts: map[string]string{"definition": "different"}}

	merged, conflicts, _, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equal("original", merged.Texts["definition"]))
}

// --- union (list class) ---

// spec.md US1 Acceptance Scenario 3 / US2 Acceptance Scenario 3.
func TestMergeUnionAccumulatesDistinctValuesNoConflict(t *testing.T) {
	index := indexWith("tags", core.MergeUnion)
	existing := core.Node{ID: "x", Type: "Entity", Attrs: map[string][]core.Predicate{"tags": {{Value: "a"}, {Value: "b"}}}}
	incoming := core.Node{ID: "x", Type: "Entity", Attrs: map[string][]core.Predicate{"tags": {{Value: "b"}, {Value: "c"}}}}

	merged, conflicts, _, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equiv(merged.Attrs["tags"], []core.Predicate{{Value: "a"}, {Value: "b"}, {Value: "c"}}))
}

// research.md D5c: a union-declared key list-unions even when both sides
// currently carry exactly one value — a documented, intentional behavior
// change from the old arity-based dispatch.
func TestMergeUnionSingleValuedBothSidesStillListUnions(t *testing.T) {
	index := indexWith("authors", core.MergeUnion)
	existing := core.Node{ID: "x", Type: "Resource", Attrs: map[string][]core.Predicate{"authors": {{Value: "Alice"}}}}
	incoming := core.Node{ID: "x", Type: "Resource", Attrs: map[string][]core.Predicate{"authors": {{Value: "Bob"}}}}

	merged, conflicts, _, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equiv(merged.Attrs["authors"], []core.Predicate{{Value: "Alice"}, {Value: "Bob"}}))
}

// research.md D5b: union on a Texts key falls back to append's paragraph
// merge (no scalar comparison, so no false-positive conflict on a
// regenerated paragraph).
func TestMergeUnionTextAppendsGenuinelyNewParagraph(t *testing.T) {
	index := indexWith("abstract", core.MergeUnion)
	existing := core.Node{ID: "x", Type: "Entity", Texts: map[string]string{"abstract": "Large Language Models are technological systems that have fundamentally transformed approaches to ontologies graph construction and knowledge management"}}
	incoming := core.Node{ID: "x", Type: "Entity", Texts: map[string]string{"abstract": "Andrej Karpathy has publicly argued that agentic coding workflows will reshape how software is written and reviewed"}}

	merged, conflicts, _, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.True(strings.Contains(merged.Texts["abstract"], "Large Language Models"))).
		Should(it.True(strings.Contains(merged.Texts["abstract"], "Andrej Karpathy")))
}

func TestMergeUnionTextDropsNearDuplicateParagraph(t *testing.T) {
	index := indexWith("abstract", core.MergeUnion)
	existing := core.Node{ID: "x", Type: "Entity", Texts: map[string]string{"abstract": "Large Language Models are technological systems that have fundamentally transformed approaches to ontologies graph construction and knowledge management"}}
	incoming := core.Node{ID: "x", Type: "Entity", Texts: map[string]string{"abstract": "Large Language Models are technological systems that have fundamentally transformed approaches to ontologies graph construction and knowledge organization"}}

	merged, conflicts, _, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equal(existing.Texts["abstract"], merged.Texts["abstract"]))
}

// BUG-002 (spec.md FR-019): a contribution that is a full duplicate of
// existing prose adds nothing — mergeText correctly drops it, and the
// reported outcome must say so (unchanged), not appended.
func TestMergeUnionTextExactDuplicateReportsUnchanged(t *testing.T) {
	index := indexWith("definition", core.MergeAppend)
	same := "Large Language Models; technological systems that have fundamentally transformed approaches to ontologies, graph construction, and knowledge management."
	existing := core.Node{ID: "x", Type: "Entity", Texts: map[string]string{"definition": same}}
	incoming := core.Node{ID: "x", Type: "Entity", Texts: map[string]string{"definition": same}}

	merged, _, outcomes, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(same, merged.Texts["definition"]))
	got := outcomeFor(t, outcomes, "definition")
	it.Then(t).
		Should(it.Equal(core.MergeAppend, got.Op)).
		Should(it.Equal(core.OutcomeUnchanged, got.Outcome))
}

// BUG-002 (spec.md FR-019): a near-duplicate paragraph (Jaccard >
// threshold, not byte-identical) is also correctly dropped by mergeText,
// and must likewise report unchanged rather than appended.
func TestMergeUnionTextNearDuplicateReportsUnchanged(t *testing.T) {
	index := indexWith("abstract", core.MergeUnion)
	existing := core.Node{ID: "x", Type: "Entity", Texts: map[string]string{"abstract": "Large Language Models are technological systems that have fundamentally transformed approaches to ontologies graph construction and knowledge management"}}
	incoming := core.Node{ID: "x", Type: "Entity", Texts: map[string]string{"abstract": "Large Language Models are technological systems that have fundamentally transformed approaches to ontologies graph construction and knowledge organization"}}

	_, _, outcomes, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	got := outcomeFor(t, outcomes, "abstract")
	it.Then(t).Should(it.Equal(core.OutcomeUnchanged, got.Outcome))
}

// --- firstWriteWin (flagOnDiverge class) ---

// spec.md US2 Acceptance Scenario 1.
func TestMergeFirstWriteWinFlagsGenuineDivergence(t *testing.T) {
	index := indexWith("abstract", core.MergeFirstWriteWin)
	existing := core.Node{ID: "x", Type: "Resource", Texts: map[string]string{"abstract": "First summary."}}
	incoming := core.Node{ID: "x", Type: "Resource", Texts: map[string]string{"abstract": "A different summary."}}

	merged, conflicts, _, err := core.Merge(existing, incoming, index, "doc-42")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, len(conflicts))).
		Should(it.Equal("abstract", conflicts[0])).
		Should(it.True(strings.Contains(merged.Texts["abstract"], "<<<<<<< existing"))).
		Should(it.True(strings.Contains(merged.Texts["abstract"], "First summary."))).
		Should(it.True(strings.Contains(merged.Texts["abstract"], "A different summary."))).
		Should(it.True(strings.Contains(merged.Texts["abstract"], "doc-42")))
}

func TestMergeFirstWriteWinAttrsFlagsGenuineDivergence(t *testing.T) {
	index := indexWith("category", core.MergeFirstWriteWin)
	existing := core.Node{ID: "x", Type: "Entity", Attrs: map[string][]core.Predicate{"category": {{Value: "physical"}}}}
	incoming := core.Node{ID: "x", Type: "Entity", Attrs: map[string][]core.Predicate{"category": {{Value: "abstract"}}}}

	merged, conflicts, _, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, len(conflicts))).
		Should(it.Equal("category", conflicts[0])).
		Should(it.True(strings.Contains(fmt.Sprint(merged.Attrs["category"][0].Value), "<<<<<<< existing")))
}

func TestMergeFirstWriteWinFillsEmptyWithoutFlag(t *testing.T) {
	index := indexWith("category", core.MergeFirstWriteWin)
	existing := core.Node{ID: "x", Type: "Entity", Attrs: map[string][]core.Predicate{"category": {{Value: ""}}}}
	incoming := core.Node{ID: "x", Type: "Entity", Attrs: map[string][]core.Predicate{"category": {{Value: "abstract"}}}}

	merged, conflicts, _, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equiv(merged.Attrs["category"], []core.Predicate{{Value: "abstract"}}))
}

func TestMergeFirstWriteWinIdenticalValueNeverFlags(t *testing.T) {
	index := indexWith("category", core.MergeFirstWriteWin)
	existing := core.Node{ID: "x", Type: "Entity", Attrs: map[string][]core.Predicate{"category": {{Value: "abstract"}}}}
	incoming := core.Node{ID: "x", Type: "Entity", Attrs: map[string][]core.Predicate{"category": {{Value: "abstract"}}}}

	_, conflicts, _, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(0, len(conflicts)))
}

// --- fillIfEmpty (flagOnDiverge class, spec FR-006: identical to firstWriteWin once set) ---

// spec.md US2 Acceptance Scenario 4 / Edge Case: fillIfEmpty never flags
// the first contribution, but a later genuine divergence is flagged
// exactly like firstWriteWin.
func TestMergeFillIfEmptyAcceptsFirstValueNoFlag(t *testing.T) {
	index := indexWith("url", core.MergeFillIfEmpty)
	existing := core.Node{ID: "x", Type: "Resource"}
	incoming := core.Node{ID: "x", Type: "Resource", Attrs: map[string][]core.Predicate{"url": {{Value: "https://example.org/a"}}}}

	merged, conflicts, _, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equiv(merged.Attrs["url"], []core.Predicate{{Value: "https://example.org/a"}}))
}

func TestMergeFillIfEmptyFlagsDivergenceAfterFirstWrite(t *testing.T) {
	index := indexWith("url", core.MergeFillIfEmpty)
	existing := core.Node{ID: "x", Type: "Resource", Attrs: map[string][]core.Predicate{"url": {{Value: "https://example.org/a"}}}}
	incoming := core.Node{ID: "x", Type: "Resource", Attrs: map[string][]core.Predicate{"url": {{Value: "https://example.org/b"}}}}

	merged, conflicts, _, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, len(conflicts))).
		Should(it.True(strings.Contains(fmt.Sprint(merged.Attrs["url"][0].Value), "<<<<<<< existing")))
}

func TestMergeFillIfEmptyIdenticalRepeatedValueNeverFlags(t *testing.T) {
	index := indexWith("url", core.MergeFillIfEmpty)
	existing := core.Node{ID: "x", Type: "Resource", Attrs: map[string][]core.Predicate{"url": {{Value: "https://example.org/a"}}}}
	incoming := core.Node{ID: "x", Type: "Resource", Attrs: map[string][]core.Predicate{"url": {{Value: "https://example.org/a"}}}}

	_, conflicts, _, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(0, len(conflicts)))
}

// --- lastWriteWin (alwaysOverwrite class) ---

// spec.md US1 Acceptance Scenario 2 / US2 Acceptance Scenario 2.
func TestMergeLastWriteWinAlwaysTakesIncomingNoFlag(t *testing.T) {
	index := indexWith("status", core.MergeLastWriteWin)
	existing := core.Node{ID: "x", Type: "Resource", Attrs: map[string][]core.Predicate{"status": {{Value: "backlog"}}}}
	incoming := core.Node{ID: "x", Type: "Resource", Attrs: map[string][]core.Predicate{"status": {{Value: "read"}}}}

	merged, conflicts, _, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equiv(merged.Attrs["status"], []core.Predicate{{Value: "read"}}))
}

// research.md D5a: lastWriteWin never leaves a conflict marker even when
// the reverse order is applied — it is order-sensitive, not conflict-prone.
func TestMergeLastWriteWinReverseOrderTakesWhicheverAppliedLast(t *testing.T) {
	index := indexWith("status", core.MergeLastWriteWin)
	n := core.Node{ID: "x", Type: "Resource", Attrs: map[string][]core.Predicate{"status": {{Value: "backlog"}}}}
	a := core.Node{ID: "x", Type: "Resource", Attrs: map[string][]core.Predicate{"status": {{Value: "read"}}}}
	b := core.Node{ID: "x", Type: "Resource", Attrs: map[string][]core.Predicate{"status": {{Value: "archived"}}}}

	abFirst, _, _, err := core.Merge(n, a, index, "doc-a")
	it.Then(t).Should(it.Nil(err))
	abFirst, _, _, err = core.Merge(abFirst, b, index, "doc-b")
	it.Then(t).Should(it.Nil(err))

	baFirst, _, _, err := core.Merge(n, b, index, "doc-b")
	it.Then(t).Should(it.Nil(err))
	baFirst, _, _, err = core.Merge(baFirst, a, index, "doc-a")
	it.Then(t).Should(it.Nil(err))

	it.Then(t).
		Should(it.Equiv(abFirst.Attrs["status"], []core.Predicate{{Value: "archived"}})).
		Should(it.Equiv(baFirst.Attrs["status"], []core.Predicate{{Value: "read"}}))
}

func TestMergeLastWriteWinTextsBehavesSameAsAttrs(t *testing.T) {
	index := indexWith("status", core.MergeLastWriteWin)
	existing := core.Node{ID: "x", Type: "Resource", Texts: map[string]string{"status": "backlog"}}
	incoming := core.Node{ID: "x", Type: "Resource", Texts: map[string]string{"status": "read"}}

	merged, conflicts, _, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equal("read", merged.Texts["status"]))
}

// --- append (list class) ---

func TestMergeAppendAttrsUnionsLikeUnion(t *testing.T) {
	index := indexWith("entries", core.MergeAppend)
	existing := core.Node{ID: "x", Type: "Timeline", Attrs: map[string][]core.Predicate{"entries": {{Value: "a"}}}}
	incoming := core.Node{ID: "x", Type: "Timeline", Attrs: map[string][]core.Predicate{"entries": {{Value: "b"}}}}

	merged, conflicts, _, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equiv(merged.Attrs["entries"], []core.Predicate{{Value: "a"}, {Value: "b"}}))
}

func TestMergeAppendTextAppendsParagraphsNeverFlags(t *testing.T) {
	index := indexWith("text", core.MergeAppend)
	existing := core.Node{ID: "x", Type: "Entity", Texts: map[string]string{"text": "Existing paragraph about a topic worth remembering."}}
	incoming := core.Node{ID: "x", Type: "Entity", Texts: map[string]string{"text": "A genuinely new paragraph introducing another topic entirely."}}

	merged, conflicts, _, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.True(strings.Contains(merged.Texts["text"], "Existing paragraph"))).
		Should(it.True(strings.Contains(merged.Texts["text"], "genuinely new paragraph")))
}

// --- validatedOverwrite (freeze class, identical to immutable for now) ---

// Edge Case: an ordinary patch contribution never overwrites nor flags a
// validatedOverwrite predicate's already-set value.
func TestMergeValidatedOverwriteNeverOverwritesOrFlags(t *testing.T) {
	index := indexWith("rank", core.MergeValidatedOverwrite)
	existing := core.Node{ID: "x", Type: "hypothesis", Attrs: map[string][]core.Predicate{"rank": {{Value: "8.5"}}}}
	incoming := core.Node{ID: "x", Type: "hypothesis", Attrs: map[string][]core.Predicate{"rank": {{Value: "9.0"}}}}

	merged, conflicts, _, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equiv(merged.Attrs["rank"], []core.Predicate{{Value: "8.5"}}))
}

// spec.md US3 Acceptance Scenario 4: replaying the exact same conflicting
// contribution does not duplicate or re-wrap the marker.
func TestMergeFirstWriteWinReplayDoesNotRewrapMarker(t *testing.T) {
	index := indexWith("abstract", core.MergeFirstWriteWin)
	existing := core.Node{ID: "x", Type: "Resource", Texts: map[string]string{"abstract": "First summary."}}
	incoming := core.Node{ID: "x", Type: "Resource", Texts: map[string]string{"abstract": "A different summary."}}

	once, _, _, err := core.Merge(existing, incoming, index, "doc-42")
	it.Then(t).Should(it.Nil(err))

	twice, conflicts, _, err := core.Merge(once, incoming, index, "doc-42")
	it.Then(t).Should(it.Nil(err))

	it.Then(t).
		Should(it.Equal(once.Texts["abstract"], twice.Texts["abstract"])).
		Should(it.Equal(1, len(conflicts))).
		Should(it.Equal(1, strings.Count(twice.Texts["abstract"], "<<<<<<<")))
}

// --- FR-012: never-flag negative assertions across every non-flagging op ---

// spec.md US2 Independent Test: a conflict marker appears only for
// firstWriteWin/fillIfEmpty; every other declared op never flags, even on
// a genuine divergence.
func TestMergeNeverFlagsExceptFirstWriteWinAndFillIfEmpty(t *testing.T) {
	neverFlag := []core.MergeOp{
		core.MergeUnion, core.MergeAppend, core.MergeLastWriteWin,
		core.MergeImmutable, core.MergeValidatedOverwrite,
	}

	for _, op := range neverFlag {
		t.Run(string(op), func(t *testing.T) {
			index := indexWith("field", op)
			existing := core.Node{ID: "x", Type: "Entity", Attrs: map[string][]core.Predicate{"field": {{Value: "existing-value"}}}}
			incoming := core.Node{ID: "x", Type: "Entity", Attrs: map[string][]core.Predicate{"field": {{Value: "incoming-value"}}}}

			_, conflicts, _, err := core.Merge(existing, incoming, index, "incoming-doc")

			it.Then(t).Should(it.Nil(err))
			it.Then(t).Should(it.Equal(0, len(conflicts)))
		})
	}
}

// --- FR-010: idempotency for every MergeOp ---

func TestMergeIsIdempotentForEveryOp(t *testing.T) {
	allOps := []core.MergeOp{
		core.MergeImmutable, core.MergeUnion, core.MergeFirstWriteWin,
		core.MergeFillIfEmpty, core.MergeLastWriteWin, core.MergeAppend,
		core.MergeValidatedOverwrite,
	}

	for _, op := range allOps {
		t.Run(string(op), func(t *testing.T) {
			index := indexWith("field", op)
			existing := core.Node{ID: "x", Type: "Entity", Attrs: map[string][]core.Predicate{"field": {{Value: "existing-value"}}}}
			incoming := core.Node{ID: "x", Type: "Entity", Attrs: map[string][]core.Predicate{"field": {{Value: "incoming-value"}}}}

			once, _, _, err := core.Merge(existing, incoming, index, "incoming-doc")
			it.Then(t).Should(it.Nil(err))
			twice, _, _, err := core.Merge(once, incoming, index, "incoming-doc")
			it.Then(t).Should(it.Nil(err))

			it.Then(t).Should(it.Equiv(once.Attrs["field"], twice.Attrs["field"]))
		})
	}
}

// --- FR-010: commutativity on independent predicates, except lastWriteWin ---

func TestMergeIsCommutativeOnIndependentPredicatesForEveryOpExceptLastWriteWin(t *testing.T) {
	commutativeOps := []core.MergeOp{
		core.MergeImmutable, core.MergeUnion, core.MergeFirstWriteWin,
		core.MergeFillIfEmpty, core.MergeAppend, core.MergeValidatedOverwrite,
	}

	for _, op := range commutativeOps {
		t.Run(string(op), func(t *testing.T) {
			index := core.Index{Predicates: map[string]core.PredicateDef{
				"fieldA": {Merge: op},
				"fieldB": {Merge: op},
			}}
			n := core.Node{ID: "x", Type: "Entity"}
			a := core.Node{ID: "x", Type: "Entity", Attrs: map[string][]core.Predicate{"fieldA": {{Value: "a-value"}}}}
			b := core.Node{ID: "x", Type: "Entity", Attrs: map[string][]core.Predicate{"fieldB": {{Value: "b-value"}}}}

			abOrder, _, _, err := core.Merge(n, a, index, "doc-a")
			it.Then(t).Should(it.Nil(err))
			abOrder, _, _, err = core.Merge(abOrder, b, index, "doc-b")
			it.Then(t).Should(it.Nil(err))

			baOrder, _, _, err := core.Merge(n, b, index, "doc-b")
			it.Then(t).Should(it.Nil(err))
			baOrder, _, _, err = core.Merge(baOrder, a, index, "doc-a")
			it.Then(t).Should(it.Nil(err))

			it.Then(t).
				Should(it.Equiv(abOrder.Attrs["fieldA"], baOrder.Attrs["fieldA"])).
				Should(it.Equiv(abOrder.Attrs["fieldB"], baOrder.Attrs["fieldB"]))
		})
	}
}

// research.md D5a: lastWriteWin is the sole documented exception —
// reordering which of two contributions is applied last changes the result.
func TestMergeLastWriteWinIsNotCommutative(t *testing.T) {
	index := indexWith("status", core.MergeLastWriteWin)
	n := core.Node{ID: "x", Type: "Resource"}
	a := core.Node{ID: "x", Type: "Resource", Attrs: map[string][]core.Predicate{"status": {{Value: "read"}}}}
	b := core.Node{ID: "x", Type: "Resource", Attrs: map[string][]core.Predicate{"status": {{Value: "archived"}}}}

	abOrder, _, _, err := core.Merge(n, a, index, "doc-a")
	it.Then(t).Should(it.Nil(err))
	abOrder, _, _, err = core.Merge(abOrder, b, index, "doc-b")
	it.Then(t).Should(it.Nil(err))

	baOrder, _, _, err := core.Merge(n, b, index, "doc-b")
	it.Then(t).Should(it.Nil(err))
	baOrder, _, _, err = core.Merge(baOrder, a, index, "doc-a")
	it.Then(t).Should(it.Nil(err))

	it.Then(t).ShouldNot(it.Equiv(abOrder.Attrs["status"], baOrder.Attrs["status"]))
}

// --- Edge Cases: list dedup / paragraph dedup no-op on replay ---

func TestMergeUnionReapplyingSameEntryDoesNotDuplicate(t *testing.T) {
	index := indexWith("tags", core.MergeUnion)
	existing := core.Node{ID: "x", Type: "Entity", Attrs: map[string][]core.Predicate{"tags": {{Value: "ai"}}}}
	incoming := core.Node{ID: "x", Type: "Entity", Attrs: map[string][]core.Predicate{"tags": {{Value: "ai"}}}}

	merged, _, _, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equiv(merged.Attrs["tags"], []core.Predicate{{Value: "ai"}}))
}

// BUG-002 (spec.md FR-019): re-contributing an already-present Attrs
// value is a no-op for a union/append-declared predicate — the reported
// outcome must say unchanged, not appended.
func TestMergeUnionAttrsReapplyingSameEntryReportsUnchanged(t *testing.T) {
	index := indexWith("tags", core.MergeUnion)
	existing := core.Node{ID: "x", Type: "Entity", Attrs: map[string][]core.Predicate{"tags": {{Value: "ai"}}}}
	incoming := core.Node{ID: "x", Type: "Entity", Attrs: map[string][]core.Predicate{"tags": {{Value: "ai"}}}}

	_, _, outcomes, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	got := outcomeFor(t, outcomes, "tags")
	it.Then(t).
		Should(it.Equal(core.MergeUnion, got.Op)).
		Should(it.Equal(core.OutcomeUnchanged, got.Outcome))
}

// --- Edges/HRefs: unconditional union regardless of declared op (research.md D5d) ---

func TestMergeEdgesUnionAcrossEveryOp(t *testing.T) {
	existingEdges := []core.Link{{Predicate: "replaces", Target: "SSL"}}
	incomingEdges := []core.Link{{Predicate: "replaces", Target: "SSL"}, {Predicate: "conformsTo", Target: "RFC 8446"}}

	for _, op := range []core.MergeOp{
		core.MergeImmutable, core.MergeUnion, core.MergeFirstWriteWin,
		core.MergeFillIfEmpty, core.MergeLastWriteWin, core.MergeAppend,
		core.MergeValidatedOverwrite,
	} {
		t.Run(string(op), func(t *testing.T) {
			index := indexWith("replaces", op)
			existing := core.Node{ID: "TLS", Type: "Entity", Edges: existingEdges}
			incoming := core.Node{ID: "TLS", Type: "Entity", Edges: incomingEdges}

			merged, _, _, err := core.Merge(existing, incoming, index, "incoming-doc")

			it.Then(t).Should(it.Nil(err))
			it.Then(t).Should(it.Equal(2, len(merged.Edges)))
		})
	}
}

// --- research.md D6: a predicate absent from the schema index falls back to union ---

func TestMergeUnregisteredPredicateFallsBackToUnion(t *testing.T) {
	index := core.Index{Predicates: map[string]core.PredicateDef{}}
	existing := core.Node{ID: "x", Type: "Entity", Attrs: map[string][]core.Predicate{"novel": {{Value: "a"}}}}
	incoming := core.Node{ID: "x", Type: "Entity", Attrs: map[string][]core.Predicate{"novel": {{Value: "b"}}}}

	merged, conflicts, _, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equiv(merged.Attrs["novel"], []core.Predicate{{Value: "a"}, {Value: "b"}}))
}

// --- Published (research.md D3): folded into the generic scalar dispatch ---

func TestMergePublishedFillsFromIncomingWhenExistingZero(t *testing.T) {
	incomingPublished := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)
	index := indexWith("published", core.MergeImmutable)
	existing := core.Node{ID: "x", Type: "Entity"}
	incoming := core.Node{ID: "x", Type: "Entity", Published: incomingPublished}

	merged, conflicts, _, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(incomingPublished, merged.Published)).
		Should(it.Equal(0, len(conflicts)))
}

func TestMergePublishedImmutableOnceSet(t *testing.T) {
	existingPublished := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)
	incomingPublished := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	index := indexWith("published", core.MergeImmutable)
	existing := core.Node{ID: "x", Type: "Entity", Published: existingPublished}
	incoming := core.Node{ID: "x", Type: "Entity", Published: incomingPublished}

	merged, conflicts, _, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(existingPublished, merged.Published)).
		Should(it.Equal(0, len(conflicts)))
}

func TestMergePublishedLastWriteWinOverwrites(t *testing.T) {
	existingPublished := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)
	incomingPublished := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	index := indexWith("published", core.MergeLastWriteWin)
	existing := core.Node{ID: "x", Type: "Entity", Published: existingPublished}
	incoming := core.Node{ID: "x", Type: "Entity", Published: incomingPublished}

	merged, conflicts, _, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(incomingPublished, merged.Published)).
		Should(it.Equal(0, len(conflicts)))
}

// --- BUG-001 / spec.md FR-017: per-predicate outcome trail ---

// outcomeFor returns the PredicateOutcome named name from outcomes, or
// fails the test if absent.
func outcomeFor(t *testing.T, outcomes []core.PredicateOutcome, name string) core.PredicateOutcome {
	t.Helper()
	for _, o := range outcomes {
		if o.Name == name {
			return o
		}
	}
	t.Fatalf("no PredicateOutcome named %q in %v", name, outcomes)
	return core.PredicateOutcome{}
}

// spec.md FR-017: one representative case per MergeOp, confirming the
// outcome trail names the touched predicate, its resolved MergeOp, and
// the outcome label matching data-model.md's per-op reconciliation table.
func TestMergeOutcomeTrailPerOp(t *testing.T) {
	tests := []struct {
		name        string
		op          core.MergeOp
		existing    core.Node
		incoming    core.Node
		wantOutcome string
	}{
		{
			name:        "immutable diverge kept unchanged",
			op:          core.MergeImmutable,
			existing:    core.Node{ID: "x", Attrs: map[string][]core.Predicate{"ref": {{Value: "book"}}}},
			incoming:    core.Node{ID: "x", Attrs: map[string][]core.Predicate{"ref": {{Value: "article"}}}},
			wantOutcome: core.OutcomeUnchanged,
		},
		{
			name:        "union accumulates",
			op:          core.MergeUnion,
			existing:    core.Node{ID: "x", Attrs: map[string][]core.Predicate{"tags": {{Value: "a"}}}},
			incoming:    core.Node{ID: "x", Attrs: map[string][]core.Predicate{"tags": {{Value: "b"}}}},
			wantOutcome: core.OutcomeAppended,
		},
		{
			name:        "firstWriteWin flags divergence",
			op:          core.MergeFirstWriteWin,
			existing:    core.Node{ID: "x", Attrs: map[string][]core.Predicate{"category": {{Value: "physical"}}}},
			incoming:    core.Node{ID: "x", Attrs: map[string][]core.Predicate{"category": {{Value: "abstract"}}}},
			wantOutcome: core.OutcomeFlagged,
		},
		{
			name:        "fillIfEmpty fills first value",
			op:          core.MergeFillIfEmpty,
			existing:    core.Node{ID: "x", Attrs: map[string][]core.Predicate{"url": {{Value: ""}}}},
			incoming:    core.Node{ID: "x", Attrs: map[string][]core.Predicate{"url": {{Value: "https://example.org/a"}}}},
			wantOutcome: core.OutcomeFilled,
		},
		{
			name:        "lastWriteWin overwrites",
			op:          core.MergeLastWriteWin,
			existing:    core.Node{ID: "x", Attrs: map[string][]core.Predicate{"status": {{Value: "backlog"}}}},
			incoming:    core.Node{ID: "x", Attrs: map[string][]core.Predicate{"status": {{Value: "read"}}}},
			wantOutcome: core.OutcomeOverwritten,
		},
		{
			name:        "append appends",
			op:          core.MergeAppend,
			existing:    core.Node{ID: "x", Attrs: map[string][]core.Predicate{"entries": {{Value: "a"}}}},
			incoming:    core.Node{ID: "x", Attrs: map[string][]core.Predicate{"entries": {{Value: "b"}}}},
			wantOutcome: core.OutcomeAppended,
		},
		{
			name:        "validatedOverwrite diverge kept unchanged",
			op:          core.MergeValidatedOverwrite,
			existing:    core.Node{ID: "x", Attrs: map[string][]core.Predicate{"rank": {{Value: "8.5"}}}},
			incoming:    core.Node{ID: "x", Attrs: map[string][]core.Predicate{"rank": {{Value: "9.0"}}}},
			wantOutcome: core.OutcomeUnchanged,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var field string
			for k := range tt.incoming.Attrs {
				field = k
			}
			index := indexWith(field, tt.op)

			_, _, outcomes, err := core.Merge(tt.existing, tt.incoming, index, "incoming-doc")

			it.Then(t).Should(it.Nil(err))
			got := outcomeFor(t, outcomes, field)
			it.Then(t).
				Should(it.Equal(field, got.Name)).
				Should(it.Equal(tt.op, got.Op)).
				Should(it.Equal(tt.wantOutcome, got.Outcome))
		})
	}
}

// A predicate present only in incoming (didn't exist on the node before)
// is reported "created", regardless of dispatch class.
func TestMergeOutcomeTrailCreatedForNewPredicate(t *testing.T) {
	index := indexWith("tags", core.MergeUnion)
	existing := core.Node{ID: "x"}
	incoming := core.Node{ID: "x", Attrs: map[string][]core.Predicate{"tags": {{Value: "ai"}}}}

	_, _, outcomes, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	got := outcomeFor(t, outcomes, "tags")
	it.Then(t).Should(it.Equal(core.OutcomeCreated, got.Outcome))
}

// A predicate present only in existing (incoming didn't touch it) is
// still reported, "unchanged" — FR-017 covers every predicate present on
// either side, not only the ones the incoming contribution mentions.
func TestMergeOutcomeTrailUnchangedForExistingOnlyPredicate(t *testing.T) {
	index := indexWith("aliases", core.MergeUnion)
	existing := core.Node{ID: "x", Attrs: map[string][]core.Predicate{"aliases": {{Value: "AKA"}}}}
	incoming := core.Node{ID: "x"}

	_, _, outcomes, err := core.Merge(existing, incoming, index, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	got := outcomeFor(t, outcomes, "aliases")
	it.Then(t).
		Should(it.Equal(core.MergeUnion, got.Op)).
		Should(it.Equal(core.OutcomeUnchanged, got.Outcome))
}

// The outcome trail is sorted by predicate name for deterministic
// --verbose output.
func TestMergeOutcomeTrailSortedByName(t *testing.T) {
	index := core.Index{Predicates: map[string]core.PredicateDef{
		"zeta":  {Merge: core.MergeUnion},
		"alpha": {Merge: core.MergeUnion},
	}}
	existing := core.Node{ID: "x"}
	incoming := core.Node{ID: "x", Attrs: map[string][]core.Predicate{
		"zeta":  {{Value: "z"}},
		"alpha": {{Value: "a"}},
	}}

	_, _, outcomes, err := core.Merge(existing, incoming, index, "incoming-doc")
	it.Then(t).Should(it.Nil(err))

	var names []string
	for _, o := range outcomes {
		names = append(names, o.Name)
	}
	it.Then(t).Should(it.Seq(names).Equal("alpha", "published", "zeta"))
}
