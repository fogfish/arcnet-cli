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

func TestMergeNoneNoOp(t *testing.T) {
	existing := core.Node{ID: "x", Type: "source", Texts: map[string]string{"abstract": "original"}}
	incoming := core.Node{ID: "x", Type: "source", Texts: map[string]string{"abstract": "different"}}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeNone, "incoming-doc")

	it.Then(t).
		Should(it.Nil(err)).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equal("original", merged.Texts["abstract"]))
}

// data-model.md/ast-contract.md "Merge": Edges is now the single unioned
// Link collection (what used to be Edges+Links) — every documented MergeOp
// unions it identically, except MergeNone which leaves existing untouched.
func TestMergeEdgesUnionAcrossOps(t *testing.T) {
	existingEdges := []core.Link{{Predicate: "replaces", Target: "SSL"}}
	incomingEdges := []core.Link{{Predicate: "replaces", Target: "SSL"}, {Predicate: "conformsTo", Target: "RFC 8446"}}

	tests := []struct {
		name    string
		op      core.MergeOp
		wantLen int
	}{
		{"MergeNone", core.MergeNone, 1},
		{"MergeUnion", core.MergeUnion, 2},
		{"MergeUnionFirstWriter", core.MergeUnionFirstWriter, 2},
		{"MergeAppend", core.MergeAppend, 2},
		{"MergeValidatedOverwrite", core.MergeValidatedOverwrite, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existing := core.Node{ID: "TLS", Type: "entity", Edges: existingEdges}
			incoming := core.Node{ID: "TLS", Type: "entity", Edges: incomingEdges}

			merged, _, err := core.Merge(existing, incoming, tt.op, "incoming-doc")

			it.Then(t).
				Should(it.Nil(err)).
				Should(it.Equal(tt.wantLen, len(merged.Edges)))
		})
	}
}

// data-model.md "Merge": Attrs merges key-by-key over the union of keys.
// A single-valued key (exactly one Predicate per side) is merged through
// the same scalar policy a bare scalar attribute used before this feature
// (mergeScalarPredicate) — under MergeUnion, a genuinely diverging
// single-valued key is never flagged (BUG-004), but it is NOT unioned into
// a multi-element list either: existing wins silently, exactly as the old
// scalar-Attrs path did (unlike a multi-valued key, which is always a
// plain list union — see TestMergeAttrsMultiValuedKeyUnionDedup).
func TestMergeAttrsSingleValuedKeyUnion(t *testing.T) {
	existing := core.Node{ID: "x", Type: "entity", Attrs: map[string][]core.Predicate{"score-c": {{Value: "0.134"}}}}
	incoming := core.Node{ID: "x", Type: "entity", Attrs: map[string][]core.Predicate{"score-c": {{Value: "0.281"}}}}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeUnion, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equiv(merged.Attrs["score-c"], []core.Predicate{{Value: "0.134"}}))
}

// Multi-valued keys union and dedup by Value's string representation,
// mirroring the prior unionScalarSlice behavior, now applied per-Predicate.
func TestMergeAttrsMultiValuedKeyUnionDedup(t *testing.T) {
	existing := core.Node{ID: "TLS", Type: "entity", Attrs: map[string][]core.Predicate{"tags": {{Value: "a"}, {Value: "b"}}}}
	incoming := core.Node{ID: "TLS", Type: "entity", Attrs: map[string][]core.Predicate{"tags": {{Value: "b"}, {Value: "c"}}}}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeUnion, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equiv(merged.Attrs["tags"], []core.Predicate{{Value: "a"}, {Value: "b"}, {Value: "c"}}))
}

// A key present on only one side of the merge is taken unchanged.
func TestMergeAttrsKeyPresentOnOnlyOneSideTakenUnchanged(t *testing.T) {
	existing := core.Node{ID: "x", Type: "entity", Attrs: map[string][]core.Predicate{"only-existing": {{Value: "a"}}}}
	incoming := core.Node{ID: "x", Type: "entity", Attrs: map[string][]core.Predicate{"only-incoming": {{Value: "b"}}}}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeUnion, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equiv(merged.Attrs["only-existing"], []core.Predicate{{Value: "a"}})).
		Should(it.Equiv(merged.Attrs["only-incoming"], []core.Predicate{{Value: "b"}}))
}

// BUG-004: MergeUnion's Texts key other than "notes" is reconciled
// paragraph-by-paragraph, not compared as one scalar (spec.md FR-024) — a
// genuinely new incoming paragraph (no existing paragraph scores above the
// similarity threshold) is appended after the existing ones, never flagged.
func TestMergeUnionTextAppendsGenuinelyNewParagraph(t *testing.T) {
	existing := core.Node{ID: "x", Type: "entity", Texts: map[string]string{"abstract": "Large Language Models are technological systems that have fundamentally transformed approaches to ontologies graph construction and knowledge management"}}
	incoming := core.Node{ID: "x", Type: "entity", Texts: map[string]string{"abstract": "Andrej Karpathy has publicly argued that agentic coding workflows will reshape how software is written and reviewed"}}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeUnion, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.True(strings.Contains(merged.Texts["abstract"], "Large Language Models"))).
		Should(it.True(strings.Contains(merged.Texts["abstract"], "Andrej Karpathy"))).
		Should(it.True(strings.Index(merged.Texts["abstract"], "Large Language Models") < strings.Index(merged.Texts["abstract"], "Andrej Karpathy")))
}

// BUG-004: an incoming paragraph that is a near-duplicate paraphrase of an
// existing one (Jaccard similarity over 3-word shingles > 0.8) is treated
// as already-present and dropped, not duplicated or flagged.
func TestMergeUnionTextDropsNearDuplicateParagraph(t *testing.T) {
	existing := core.Node{ID: "x", Type: "entity", Texts: map[string]string{"abstract": "Large Language Models are technological systems that have fundamentally transformed approaches to ontologies graph construction and knowledge management"}}
	incoming := core.Node{ID: "x", Type: "entity", Texts: map[string]string{"abstract": "Large Language Models are technological systems that have fundamentally transformed approaches to ontologies graph construction and knowledge organization"}}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeUnion, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equal(existing.Texts["abstract"], merged.Texts["abstract"]))
}

// BUG-004: multiple incoming paragraphs are each evaluated independently
// against every existing paragraph — a near-duplicate of one is dropped
// while a genuinely new one is still appended in the same merge.
func TestMergeUnionTextEvaluatesEachIncomingParagraphIndependently(t *testing.T) {
	existing := core.Node{ID: "x", Type: "entity", Texts: map[string]string{"abstract": "Large Language Models are technological systems that have fundamentally transformed approaches to ontologies graph construction and knowledge management"}}
	incoming := core.Node{ID: "x", Type: "entity", Texts: map[string]string{"abstract": "Large Language Models are technological systems that have fundamentally transformed approaches to ontologies graph construction and knowledge organization\n\nAndrej Karpathy has publicly argued that agentic coding workflows will reshape how software is written and reviewed"}}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeUnion, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equal(2, len(strings.Split(merged.Texts["abstract"], "\n\n")))).
		Should(it.True(strings.Contains(merged.Texts["abstract"], "knowledge management"))).
		Should(it.True(strings.Contains(merged.Texts["abstract"], "Andrej Karpathy")))
}

// BUG-004 regression guard: a Texts key literally named "notes" is still
// flagged exactly as before this fix — only other Texts keys/Attrs are
// narrowed to never-flagged/paragraph-merged.
func TestMergeUnionNotesStillFlagsConflicts(t *testing.T) {
	existing := core.Node{ID: "x", Type: "entity", Texts: map[string]string{"notes": "existing notes"}}
	incoming := core.Node{ID: "x", Type: "entity", Texts: map[string]string{"notes": "incoming notes"}}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeUnion, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, len(conflicts))).
		Should(it.Equal("notes", conflicts[0])).
		Should(it.True(strings.Contains(merged.Texts["notes"], "<<<<<<< existing")))
}

// Under MergeUnion, "notes" goes through the scalar path with fillEmpty
// false: an empty existing "notes" is left unfilled, unlike a non-"notes"
// Texts key (which mergeText fills unconditionally, see below).
func TestMergeUnionNotesLeavesEmptyUnfilled(t *testing.T) {
	existing := core.Node{ID: "x", Type: "entity", Texts: map[string]string{"notes": ""}}
	incoming := core.Node{ID: "x", Type: "entity", Texts: map[string]string{"notes": "incoming notes"}}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeUnion, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equal("", merged.Texts["notes"]))
}

// mergeText fills an empty side unconditionally, regardless of fillEmpty:
// a non-"notes" Texts key under MergeUnion with an empty existing value is
// filled from incoming.
func TestMergeUnionNonNotesTextKeyFillsEmptyViaMergeText(t *testing.T) {
	existing := core.Node{ID: "x", Type: "entity", Texts: map[string]string{"claim": ""}}
	incoming := core.Node{ID: "x", Type: "entity", Texts: map[string]string{"claim": "the claim"}}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeUnion, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equal("the claim", merged.Texts["claim"]))
}

// A single-valued Attrs key (exactly one Predicate per side) behaves like a
// bare scalar attribute did before this feature: MergeUnionFirstWriter
// fills an empty existing value from incoming, with no conflict flagged.
func TestMergeUnionFirstWriterFillsEmptySingleValuedAttr(t *testing.T) {
	existing := core.Node{ID: "x", Type: "resource", Attrs: map[string][]core.Predicate{"status": {{Value: ""}}}}
	incoming := core.Node{ID: "x", Type: "resource", Attrs: map[string][]core.Predicate{"status": {{Value: "read"}}}}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeUnionFirstWriter, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equiv(merged.Attrs["status"], []core.Predicate{{Value: "read"}}))
}

// A single-valued Attrs key that is already set on both sides and
// genuinely diverges IS flagged as a conflict under MergeUnionFirstWriter
// (unlike a multi-valued key, which is always list-unioned) — this is the
// pre-feature scalar-Attrs conflict behavior, carried over unchanged, only
// the shape ([]Predicate of length 1) is new.
func TestMergeUnionFirstWriterFlagsDivergingSingleValuedAttr(t *testing.T) {
	existing := core.Node{ID: "x", Type: "resource", Attrs: map[string][]core.Predicate{"status": {{Value: "read"}}}}
	incoming := core.Node{ID: "x", Type: "resource", Attrs: map[string][]core.Predicate{"status": {{Value: "backlog"}}}}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeUnionFirstWriter, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, len(conflicts))).
		Should(it.Equal("status", conflicts[0])).
		Should(it.Equal(1, len(merged.Attrs["status"]))).
		Should(it.True(strings.Contains(fmt.Sprint(merged.Attrs["status"][0].Value), "<<<<<<< existing")))
}

// MergeValidatedOverwrite never flags a single-valued Attrs conflict:
// existing wins silently, exactly as a bare scalar attribute did pre-feature.
func TestMergeValidatedOverwriteNeverFlagsSingleValuedAttrConflict(t *testing.T) {
	existing := core.Node{ID: "x", Type: "hypothesis", Attrs: map[string][]core.Predicate{"rank": {{Value: "8.5"}}}}
	incoming := core.Node{ID: "x", Type: "hypothesis", Attrs: map[string][]core.Predicate{"rank": {{Value: "9.0"}}}}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeValidatedOverwrite, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equiv(merged.Attrs["rank"], []core.Predicate{{Value: "8.5"}}))
}

// MergeAppend never flags a single-valued Attrs conflict either — existing
// wins silently, mirroring MergeAppend's never-flag Text behavior.
func TestMergeAppendNeverFlagsSingleValuedAttrConflict(t *testing.T) {
	existing := core.Node{ID: "x", Type: "logEntry", Attrs: map[string][]core.Predicate{"status": {{Value: "existing"}}}}
	incoming := core.Node{ID: "x", Type: "logEntry", Attrs: map[string][]core.Predicate{"status": {{Value: "incoming"}}}}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeAppend, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equiv(merged.Attrs["status"], []core.Predicate{{Value: "existing"}}))
}

func TestMergeUnionFirstWriterFillsEmptyTextKey(t *testing.T) {
	existing := core.Node{ID: "x", Type: "resource", Texts: map[string]string{"status": ""}}
	incoming := core.Node{ID: "x", Type: "resource", Texts: map[string]string{"status": "read"}}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeUnionFirstWriter, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equal("read", merged.Texts["status"]))
}

// Also verifies the conflict marker embeds the caller-supplied sourceID
// (research.md D7) and both diverging values.
func TestMergeUnionFirstWriterPreservesAlreadySetTextKey(t *testing.T) {
	existing := core.Node{ID: "x", Type: "resource", Texts: map[string]string{"claim": "existing claim"}}
	incoming := core.Node{ID: "x", Type: "resource", Texts: map[string]string{"claim": "incoming claim"}}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeUnionFirstWriter, "doc-42")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, len(conflicts))).
		Should(it.Equal("claim", conflicts[0])).
		Should(it.True(strings.Contains(merged.Texts["claim"], "<<<<<<< existing"))).
		Should(it.True(strings.Contains(merged.Texts["claim"], "existing claim"))).
		Should(it.True(strings.Contains(merged.Texts["claim"], "incoming claim"))).
		Should(it.True(strings.Contains(merged.Texts["claim"], "doc-42")))
}

func TestMergeValidatedOverwriteNeverFlagsConflicts(t *testing.T) {
	existing := core.Node{ID: "x", Type: "hypothesis", Texts: map[string]string{"claim": "8.5"}}
	incoming := core.Node{ID: "x", Type: "hypothesis", Texts: map[string]string{"claim": "9.0"}}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeValidatedOverwrite, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equal("8.5", merged.Texts["claim"]))
}

func TestMergeUnknownOpReturnsError(t *testing.T) {
	existing := core.Node{ID: "x", Type: "entity"}
	incoming := core.Node{ID: "x", Type: "entity"}

	_, _, err := core.Merge(existing, incoming, core.MergeOp("bogus"), "incoming-doc")

	it.Then(t).ShouldNot(it.Nil(err))
}

// BUG-002: a domain/extension kind registered with append (spec.md FR-022)
// must merge like union, never crash with ErrUnknownMergeOp.
func TestMergeAppendUnionsEdgesAndAttrs(t *testing.T) {
	existing := core.Node{
		ID: "Widget", Type: "logEntry",
		Attrs: map[string][]core.Predicate{"tags": {{Value: "a"}, {Value: "b"}}},
		Edges: []core.Link{{Predicate: "replaces", Target: "SSL"}},
	}
	incoming := core.Node{
		ID: "Widget", Type: "logEntry",
		Attrs: map[string][]core.Predicate{"tags": {{Value: "b"}, {Value: "c"}}},
		Edges: []core.Link{{Predicate: "replaces", Target: "SSL"}, {Predicate: "conformsTo", Target: "RFC 8446"}},
	}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeAppend, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(0, len(conflicts)))
	it.Then(t).Should(it.Equal(2, len(merged.Edges)))
	it.Then(t).Should(it.Equiv(merged.Attrs["tags"], []core.Predicate{{Value: "a"}, {Value: "b"}, {Value: "c"}}))
}

func TestMergeAppendNeverFlagsTextConflicts(t *testing.T) {
	existing := core.Node{ID: "x", Type: "logEntry", Texts: map[string]string{"claim": "existing text"}}
	incoming := core.Node{ID: "x", Type: "logEntry", Texts: map[string]string{"claim": "incoming text"}}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeAppend, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equal("existing text", merged.Texts["claim"]))
}

// BUG-002: confirm ErrUnknownMergeOp still fires only for a genuinely
// unrecognized MergeOp, not for any of the five documented values.
func TestMergeAllDocumentedOpsNeverReturnErrUnknownMergeOp(t *testing.T) {
	existing := core.Node{ID: "x", Type: "entity"}
	incoming := core.Node{ID: "x", Type: "entity"}

	for _, op := range []core.MergeOp{
		core.MergeNone, core.MergeUnion, core.MergeUnionFirstWriter,
		core.MergeAppend, core.MergeValidatedOverwrite,
	} {
		_, _, err := core.Merge(existing, incoming, op, "incoming-doc")
		it.Then(t).Should(it.Nil(err))
	}
}

// research.md D3: Published fills once from incoming when existing.Published
// is zero, for every non-"none" op, never flagged as a conflict.
func TestMergePublishedFillsFromIncomingWhenExistingZero(t *testing.T) {
	incomingPublished := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)

	for _, op := range []core.MergeOp{
		core.MergeUnion, core.MergeUnionFirstWriter, core.MergeAppend, core.MergeValidatedOverwrite,
	} {
		existing := core.Node{ID: "x", Type: "entity"}
		incoming := core.Node{ID: "x", Type: "entity", Published: incomingPublished}

		merged, conflicts, err := core.Merge(existing, incoming, op, "incoming-doc")

		it.Then(t).Should(it.Nil(err))
		it.Then(t).
			Should(it.Equal(incomingPublished, merged.Published)).
			Should(it.Equal(0, len(conflicts)))
	}
}

// research.md D3: Published, once non-zero on existing, is preserved
// unchanged by a later merge even when incoming declares a different
// value — first-writer-wins, never flagged.
func TestMergePublishedPreservedWhenExistingAlreadySet(t *testing.T) {
	existingPublished := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)
	incomingPublished := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	for _, op := range []core.MergeOp{
		core.MergeUnion, core.MergeUnionFirstWriter, core.MergeAppend, core.MergeValidatedOverwrite,
	} {
		existing := core.Node{ID: "x", Type: "entity", Published: existingPublished}
		incoming := core.Node{ID: "x", Type: "entity", Published: incomingPublished}

		merged, conflicts, err := core.Merge(existing, incoming, op, "incoming-doc")

		it.Then(t).Should(it.Nil(err))
		it.Then(t).
			Should(it.Equal(existingPublished, merged.Published)).
			Should(it.Equal(0, len(conflicts)))
	}
}

// research.md D3: MergeNone's existing whole-node no-op leaves Published
// untouched, matching every other field.
func TestMergeNoneLeavesPublishedUntouched(t *testing.T) {
	existingPublished := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)
	existing := core.Node{ID: "x", Type: "source", Published: existingPublished}
	incoming := core.Node{ID: "x", Type: "source", Published: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeNone, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(existingPublished, merged.Published)).
		Should(it.Equal(0, len(conflicts)))
}
