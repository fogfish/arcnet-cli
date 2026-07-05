//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package core_test

import (
	"strings"
	"testing"
	"time"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/core"
)

func TestMergeNoneNoOp(t *testing.T) {
	existing := core.Node{ID: "x", Kind: "source", Text: "original"}
	incoming := core.Node{ID: "x", Kind: "source", Text: "different"}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeNone, "incoming-doc")

	it.Then(t).
		Should(it.Nil(err)).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equal("original", merged.Text))
}

func TestMergeUnionEdgesAndMultiValuedAttrs(t *testing.T) {
	existing := core.Node{
		ID: "TLS", Kind: "entity", Text: "shared",
		Attrs: map[string]any{"tags": []any{"a", "b"}},
		Edges: []core.Link{{Predicate: "replaces", Target: "SSL"}},
	}
	incoming := core.Node{
		ID: "TLS", Kind: "entity", Text: "shared",
		Attrs: map[string]any{"tags": []any{"b", "c"}},
		Edges: []core.Link{{Predicate: "replaces", Target: "SSL"}, {Predicate: "conformsTo", Target: "RFC 8446"}},
	}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeUnion, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(0, len(conflicts)))
	it.Then(t).Should(it.Equal(2, len(merged.Edges)))

	tags, ok := merged.Attrs["tags"].([]any)
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal(3, len(tags)))
}

// BUG-004: MergeUnion's Attrs are never flagged as a conflict — a
// divergent value is kept at existing's first-written value, silently
// (spec.md FR-023). Superseded "conflict-marker embedding" behavior for
// MergeUnion is now covered by TestMergeUnionTextAppendsGenuinelyNewParagraph
// and TestMergeUnionTextDropsNearDuplicateParagraph below.
func TestMergeUnionScalarAttrsNeverFlagConflicts(t *testing.T) {
	existing := core.Node{ID: "x", Kind: "entity", Attrs: map[string]any{"score-c": "0.134"}}
	incoming := core.Node{ID: "x", Kind: "entity", Attrs: map[string]any{"score-c": "0.281"}}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeUnion, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equal("0.134", merged.Attrs["score-c"]))
}

// BUG-004: MergeUnion's Text is reconciled paragraph-by-paragraph, not
// compared as one scalar (spec.md FR-024) — a genuinely new incoming
// paragraph (no existing paragraph scores above the similarity threshold)
// is appended after the existing ones, never flagged.
func TestMergeUnionTextAppendsGenuinelyNewParagraph(t *testing.T) {
	existing := core.Node{ID: "x", Kind: "entity", Text: "Large Language Models are technological systems that have fundamentally transformed approaches to ontologies graph construction and knowledge management"}
	incoming := core.Node{ID: "x", Kind: "entity", Text: "Andrej Karpathy has publicly argued that agentic coding workflows will reshape how software is written and reviewed"}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeUnion, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.True(strings.Contains(merged.Text, "Large Language Models"))).
		Should(it.True(strings.Contains(merged.Text, "Andrej Karpathy"))).
		Should(it.True(strings.Index(merged.Text, "Large Language Models") < strings.Index(merged.Text, "Andrej Karpathy")))
}

// BUG-004: an incoming paragraph that is a near-duplicate paraphrase of an
// existing one (Jaccard similarity over 3-word shingles > 0.8) is treated
// as already-present and dropped, not duplicated or flagged.
func TestMergeUnionTextDropsNearDuplicateParagraph(t *testing.T) {
	existing := core.Node{ID: "x", Kind: "entity", Text: "Large Language Models are technological systems that have fundamentally transformed approaches to ontologies graph construction and knowledge management"}
	incoming := core.Node{ID: "x", Kind: "entity", Text: "Large Language Models are technological systems that have fundamentally transformed approaches to ontologies graph construction and knowledge organization"}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeUnion, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equal(existing.Text, merged.Text))
}

// BUG-004: multiple incoming paragraphs are each evaluated independently
// against every existing paragraph — a near-duplicate of one is dropped
// while a genuinely new one is still appended in the same merge.
func TestMergeUnionTextEvaluatesEachIncomingParagraphIndependently(t *testing.T) {
	existing := core.Node{ID: "x", Kind: "entity", Text: "Large Language Models are technological systems that have fundamentally transformed approaches to ontologies graph construction and knowledge management"}
	incoming := core.Node{ID: "x", Kind: "entity", Text: "Large Language Models are technological systems that have fundamentally transformed approaches to ontologies graph construction and knowledge organization\n\nAndrej Karpathy has publicly argued that agentic coding workflows will reshape how software is written and reviewed"}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeUnion, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equal(2, len(strings.Split(merged.Text, "\n\n")))).
		Should(it.True(strings.Contains(merged.Text, "knowledge management"))).
		Should(it.True(strings.Contains(merged.Text, "Andrej Karpathy")))
}

// BUG-004 regression guard: Notes divergence under MergeUnion is still
// flagged exactly as before this fix — only Attrs/Text are narrowed.
func TestMergeUnionNotesStillFlagsConflicts(t *testing.T) {
	existing := core.Node{ID: "x", Kind: "entity", Notes: "existing notes"}
	incoming := core.Node{ID: "x", Kind: "entity", Notes: "incoming notes"}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeUnion, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, len(conflicts))).
		Should(it.Equal("notes", conflicts[0])).
		Should(it.True(strings.Contains(merged.Notes, "<<<<<<< existing")))
}

func TestMergeUnionLeavesEmptyScalarUnfilled(t *testing.T) {
	existing := core.Node{ID: "x", Kind: "entity", Attrs: map[string]any{"url": ""}}
	incoming := core.Node{ID: "x", Kind: "entity", Attrs: map[string]any{"url": "https://example.org"}}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeUnion, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equal("", merged.Attrs["url"]))
}

func TestMergeUnionFirstWriterFillsEmptyScalar(t *testing.T) {
	existing := core.Node{ID: "x", Kind: "resource", Attrs: map[string]any{"status": ""}}
	incoming := core.Node{ID: "x", Kind: "resource", Attrs: map[string]any{"status": "read"}}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeUnionFirstWriter, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equal("read", merged.Attrs["status"]))
}

func TestMergeUnionFirstWriterPreservesAlreadySetScalar(t *testing.T) {
	existing := core.Node{ID: "x", Kind: "resource", Attrs: map[string]any{"status": "read"}}
	incoming := core.Node{ID: "x", Kind: "resource", Attrs: map[string]any{"status": "backlog"}}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeUnionFirstWriter, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, len(conflicts))).
		Should(it.True(strings.Contains(merged.Attrs["status"].(string), "<<<<<<< existing")))
}

func TestMergeValidatedOverwriteNeverFlagsConflicts(t *testing.T) {
	existing := core.Node{ID: "x", Kind: "hypothesis", Attrs: map[string]any{"rank": "8.5"}}
	incoming := core.Node{ID: "x", Kind: "hypothesis", Attrs: map[string]any{"rank": "9.0"}}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeValidatedOverwrite, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equal("8.5", merged.Attrs["rank"]))
}

func TestMergeUnknownOpReturnsError(t *testing.T) {
	existing := core.Node{ID: "x", Kind: "entity"}
	incoming := core.Node{ID: "x", Kind: "entity"}

	_, _, err := core.Merge(existing, incoming, core.MergeOp("bogus"), "incoming-doc")

	it.Then(t).ShouldNot(it.Nil(err))
}

// BUG-002: a domain/extension kind registered with append (spec.md FR-022)
// must merge like union, never crash with ErrUnknownMergeOp.
func TestMergeAppendUnionsEdgesAndMultiValuedAttrs(t *testing.T) {
	existing := core.Node{
		ID: "Widget", Kind: "logEntry", Text: "shared",
		Attrs: map[string]any{"tags": []any{"a", "b"}},
		Edges: []core.Link{{Predicate: "replaces", Target: "SSL"}},
	}
	incoming := core.Node{
		ID: "Widget", Kind: "logEntry", Text: "shared",
		Attrs: map[string]any{"tags": []any{"b", "c"}},
		Edges: []core.Link{{Predicate: "replaces", Target: "SSL"}, {Predicate: "conformsTo", Target: "RFC 8446"}},
	}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeAppend, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(0, len(conflicts)))
	it.Then(t).Should(it.Equal(2, len(merged.Edges)))

	tags, ok := merged.Attrs["tags"].([]any)
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal(3, len(tags)))
}

func TestMergeAppendNeverFlagsScalarConflicts(t *testing.T) {
	existing := core.Node{ID: "x", Kind: "logEntry", Text: "existing text"}
	incoming := core.Node{ID: "x", Kind: "logEntry", Text: "incoming text"}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeAppend, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(0, len(conflicts))).
		Should(it.Equal("existing text", merged.Text))
}

// BUG-002: confirm ErrUnknownMergeOp still fires only for a genuinely
// unrecognized MergeOp, not for any of the five documented values.
func TestMergeAllDocumentedOpsNeverReturnErrUnknownMergeOp(t *testing.T) {
	existing := core.Node{ID: "x", Kind: "entity"}
	incoming := core.Node{ID: "x", Kind: "entity"}

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
		existing := core.Node{ID: "x", Kind: "entity"}
		incoming := core.Node{ID: "x", Kind: "entity", Published: incomingPublished}

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
		existing := core.Node{ID: "x", Kind: "entity", Published: existingPublished}
		incoming := core.Node{ID: "x", Kind: "entity", Published: incomingPublished}

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
	existing := core.Node{ID: "x", Kind: "source", Published: existingPublished}
	incoming := core.Node{ID: "x", Kind: "source", Published: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)}

	merged, conflicts, err := core.Merge(existing, incoming, core.MergeNone, "incoming-doc")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(existingPublished, merged.Published)).
		Should(it.Equal(0, len(conflicts)))
}
