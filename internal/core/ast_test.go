//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package core_test

import (
	"testing"
	"time"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/core"
)

func TestNodeZeroValue(t *testing.T) {
	var n core.Node

	it.Then(t).
		Should(it.Equal("", n.ID)).
		Should(it.Equal("", n.Type)).
		Should(it.True(n.Published.IsZero())).
		Should(it.Equal(0, len(n.Attrs))).
		Should(it.Equal(0, len(n.Texts))).
		Should(it.Equal(0, len(n.HRefs))).
		Should(it.Equal(0, len(n.Edges)))
}

// data-model.md: Node.Published is a typed field, mirroring Patch's own
// existing Published field — a Node literal that sets it retains it as an
// ordinary struct field, no different from any other typed field.
func TestNodePublishedRetainsSetValue(t *testing.T) {
	published := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)
	n := core.Node{ID: "x", Type: "source", Published: published}

	it.Then(t).
		ShouldNot(it.True(n.Published.IsZero())).
		Should(it.Equal(published, n.Published))
}

// data-model.md/research.md D2: a Predicate with only Value set represents
// a scalar attribute value; one with only Target set represents a
// reference-valued predicate (Value/Target are exactly-one-of, this
// feature's parser only ever populates Value).
func TestPredicateValueOnly(t *testing.T) {
	p := core.Predicate{Value: "cryptography"}

	it.Then(t).
		Should(it.Equal("cryptography", p.Value)).
		Should(it.Equal("", p.Target)).
		Should(it.Equal("", p.Alias))
}

func TestPredicateTargetOnly(t *testing.T) {
	p := core.Predicate{Target: "Transport Layer Security", Alias: "TLS"}

	it.Then(t).
		Should(it.Equal(nil, p.Value)).
		Should(it.Equal("Transport Layer Security", p.Target)).
		Should(it.Equal("TLS", p.Alias))
}

// data-model.md: an Attrs entry is a non-empty ordered list of Predicate —
// one element for a single-valued attribute, several for a multi-valued one.
func TestAttrsSingleAndMultiValued(t *testing.T) {
	n := core.Node{
		Attrs: map[string][]core.Predicate{
			"year": {{Value: 2018}},
			"tags": {{Value: "cryptography"}, {Value: "networking"}},
		},
	}

	it.Then(t).
		Should(it.Equal(1, len(n.Attrs["year"]))).
		Should(it.Equal(2, len(n.Attrs["tags"]))).
		Should(it.Equal("cryptography", n.Attrs["tags"][0].Value)).
		Should(it.Equal("networking", n.Attrs["tags"][1].Value))
}

// data-model.md: Texts is an open, name-keyed map — a node can carry
// several independently named prose fields simultaneously.
func TestTextsMultipleDistinctKeys(t *testing.T) {
	n := core.Node{
		Texts: map[string]string{
			"abstract": "A cryptographic protocol.",
			"notes":    "Superseded by TLS 1.3.",
		},
	}

	it.Then(t).
		Should(it.Equal(2, len(n.Texts))).
		Should(it.Equal("A cryptographic protocol.", n.Texts["abstract"])).
		Should(it.Equal("Superseded by TLS 1.3.", n.Texts["notes"]))
}

// research.md D5: a single Edges slice mixes what were previously bare and
// grouped links, in document order, with no per-block grouping retained.
func TestEdgesUnifiesBareAndGroupedLinks(t *testing.T) {
	n := core.Node{
		Edges: []core.Link{
			{Target: "SSL Protocol"},
			{Predicate: "mentionedIn", Target: "rescorla-2026-tls13"},
		},
	}

	it.Then(t).
		Should(it.Equal(2, len(n.Edges))).
		Should(it.Equal("SSL Protocol", n.Edges[0].Target)).
		Should(it.Equal("mentionedIn", n.Edges[1].Predicate))
}

func TestPatchZeroValue(t *testing.T) {
	var p core.Patch

	it.Then(t).
		Should(it.Equal("", p.Document)).
		Should(it.True(p.Published.IsZero())).
		Should(it.Equal(0, len(p.Nodes)))
}

func TestMergeOpConstants(t *testing.T) {
	it.Then(t).
		Should(it.Equal(core.MergeOp("none"), core.MergeNone)).
		Should(it.Equal(core.MergeOp("union"), core.MergeUnion)).
		Should(it.Equal(core.MergeOp("union-first-writer"), core.MergeUnionFirstWriter)).
		Should(it.Equal(core.MergeOp("append"), core.MergeAppend)).
		Should(it.Equal(core.MergeOp("validated-overwrite"), core.MergeValidatedOverwrite))
}
