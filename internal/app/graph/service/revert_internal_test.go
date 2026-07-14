//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// package service (white-box, not service_test): research.md D8's
// conflict-marker provenance resolution is exercised directly against
// resolveConflictMarker/parseConflictMarker here, bypassing
// core.RenderNode/core.ParseNode entirely — a conflict-marker-shaped
// Texts value does not currently survive that round trip (its
// "=======" delimiter is parsed as a Markdown setext-heading underline,
// silently dropping the "existing" side), a pre-existing internal/core
// gap independent of this feature and out of scope to fix here. Every
// other revert_test.go case stays black-box, matching this package's own
// established convention; this file is a narrow, justified exception for
// the one code path black-box testing cannot currently reach.
package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fogfish/it/v2"

	graphmock "github.com/fogfish/arcnet-cli/internal/app/graph/adapter/mock"
	"github.com/fogfish/arcnet-cli/internal/app/graph/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

func TestParseConflictMarkerRoundTripsExactShape(t *testing.T) {
	value := "<<<<<<< existing\nOld text.\n=======\nNew text.\n>>>>>>> foo-2026-x"

	existingVal, incomingVal, incomingSourceID, ok := parseConflictMarker(value)

	it.Then(t).Should(it.True(ok))
	it.Then(t).
		Should(it.Equal("Old text.", existingVal)).
		Should(it.Equal("New text.", incomingVal)).
		Should(it.Equal("foo-2026-x", incomingSourceID))
}

func TestParseConflictMarkerRejectsOrdinaryText(t *testing.T) {
	_, _, _, ok := parseConflictMarker("Just an ordinary paragraph.")

	it.Then(t).ShouldNot(it.True(ok))
}

// research.md D8(a): the reverted patch is the marker's own
// self-documented incoming side — resolved by plain-text comparison, no
// git call needed.
func TestResolveConflictMarkerWhenRevertedPatchIsIncomingSide(t *testing.T) {
	vcs := &graphmock.VCS{}

	resolved, matched, err := resolveConflictMarker(context.Background(), vcs, "/graph", "entities/Widget.md", "notes",
		"Old text.", "New text.", "foo-2026-x", "ingest123", "foo-2026-x")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.True(matched)).
		Should(it.Equal("Old text.", resolved))
	it.Then(t).Should(it.Equal(0, len(vcs.Calls)))
}

// research.md D8(b): the reverted patch is the marker's frozen "existing"
// side — resolved by walking CommitsTouching oldest-first via
// ShowFile+core.ParseNode to find this predicate's true first writer.
func TestResolveConflictMarkerWhenRevertedPatchIsFrozenExistingSide(t *testing.T) {
	// walkNodeBody only recognizes a solo paragraph as "trailing" prose
	// (vs. "leading") when it follows a heading-or-bold-label-plus-list
	// block — a placeholder Edge forces "Old text." to round-trip back
	// into the trailing "notes" key rather than entity's leading
	// "definition" key.
	c1Bytes, err := core.RenderNode(core.Node{ID: "Widget", Type: "entity"}, core.Index{})
	it.Then(t).Should(it.Nil(err))
	c2Bytes, err := core.RenderNode(core.Node{
		ID: "Widget", Type: "entity",
		Texts: map[string]string{"notes": "Old text."},
		Edges: []core.Link{{Predicate: "relatesTo", Target: "Other"}},
	}, core.Index{})
	it.Then(t).Should(it.Nil(err))

	vcs := &graphmock.VCS{
		CommitsTouchingFn: func(dir, path string) ([]string, error) {
			// newest-first, per contracts/vcs-port-contract.md.
			return []string{"c3", "c2", "c1"}, nil
		},
		ShowFileFn: func(dir, hash, path string) ([]byte, error) {
			switch hash {
			case "c1":
				return c1Bytes, nil // no "notes" value yet
			case "c2":
				return c2Bytes, nil // the first commit to set it
			default:
				return nil, errors.New("unexpected historical revision requested")
			}
		},
	}

	resolved, matched, err := resolveConflictMarker(context.Background(), vcs, "/graph", "entities/Widget.md", "notes",
		"Old text.", "New text.", "other-2026-y", "c2", "foo-2026-x")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.True(matched)).
		Should(it.Equal("New text.", resolved))
}

// research.md D8 "neither": the reverted patch made no contribution to
// this specific predicate — left untouched, not an error.
func TestResolveConflictMarkerWhenRevertedPatchIsNeitherSide(t *testing.T) {
	c1Bytes, err := core.RenderNode(core.Node{ID: "Widget", Type: "entity"}, core.Index{})
	it.Then(t).Should(it.Nil(err))
	c2Bytes, err := core.RenderNode(core.Node{
		ID: "Widget", Type: "entity",
		Texts: map[string]string{"notes": "Old text."},
		Edges: []core.Link{{Predicate: "relatesTo", Target: "Other"}},
	}, core.Index{})
	it.Then(t).Should(it.Nil(err))

	vcs := &graphmock.VCS{
		CommitsTouchingFn: func(dir, path string) ([]string, error) {
			return []string{"c3", "c2", "c1"}, nil
		},
		ShowFileFn: func(dir, hash, path string) ([]byte, error) {
			switch hash {
			case "c1":
				return c1Bytes, nil
			case "c2":
				return c2Bytes, nil // first writer is "c2", not the reverted patch's own ingest commit
			default:
				return nil, errors.New("unexpected historical revision requested")
			}
		},
	}

	resolved, matched, err := resolveConflictMarker(context.Background(), vcs, "/graph", "entities/Widget.md", "notes",
		"Old text.", "New text.", "other-2026-y", "ingest-of-someone-else", "foo-2026-x")

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		ShouldNot(it.True(matched)).
		Should(it.Equal("", resolved))
}

func TestMapTextParagraphsLocatesEachParagraphsLineRange(t *testing.T) {
	node := core.Node{
		ID:   "Widget",
		Type: "entity",
		Texts: map[string]string{
			"notes": "First paragraph.\n\nSecond paragraph.",
		},
	}
	rendered, err := core.RenderNode(node, core.Index{})
	it.Then(t).Should(it.Nil(err))

	ranges := mapTextParagraphs(node, revertLeadingKey(node.Type), revertTrailingKey, rendered)

	it.Then(t).Should(it.Equal(2, len(ranges["notes"])))
	it.Then(t).
		Should(it.Equal("First paragraph.", ranges["notes"][0].text)).
		Should(it.Equal("Second paragraph.", ranges["notes"][1].text))
	it.Then(t).Should(it.True(ranges["notes"][0].endLine < ranges["notes"][1].startLine))
}

func TestSplitParagraphsLocal(t *testing.T) {
	got := splitParagraphsLocal("one\n\ntwo\n\n\nthree")

	it.Then(t).Should(it.Equal(3, len(got)))
	it.Then(t).
		Should(it.Equal("one", got[0])).
		Should(it.Equal("two", got[1])).
		Should(it.Equal("three", got[2]))
}

func TestBuildRevertCommitMessageCarriesRevertedDocumentTrailerNotSourceId(t *testing.T) {
	result := kernel.RevertResult{
		Removed:      map[string]int{"entity": 1},
		Reconciled:   map[string]int{"resource": 2},
		LinksRemoved: 3,
	}

	msg := buildRevertCommitMessage("foo-2026-x", result)

	it.Then(t).
		Should(it.String(msg).Contain("Reverted-Document: foo-2026-x")).
		ShouldNot(it.String(msg).Contain("Source-Id:"))
}
