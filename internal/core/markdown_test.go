//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package core_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/core"
)

const patchFixture = `---
kind: patch
document: rescorla-2026-tls13
published: 2026-04-12
title: "TLS 1.3: Design and Rationale"
---
# Source

## rescorla-2026-tls13
` + "```yaml" + `
title: "TLS 1.3: Design and Rationale"
authors: [Eric Rescorla]
published: "2026-04-12"
url: https://example.org/tls13-design
` + "```" + `

A design retrospective on the TLS 1.3 handshake.

## Mentions
- mentions:: [[Transport Layer Security]]

# Entity

## Transport Layer Security
` + "```yaml" + `
category: [independent, abstract, occurrent, script]
` + "```" + `

A cryptographic protocol that establishes an authenticated, confidential channel.
`

func TestParsePatchManifestAndNodes(t *testing.T) {
	patch, err := core.ParsePatch(strings.NewReader(patchFixture))

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal("rescorla-2026-tls13", patch.Document)).
		Should(it.Equal("TLS 1.3: Design and Rationale", patch.Title)).
		Should(it.Equal(2026, patch.Published.Year())).
		Should(it.Equal(2, len(patch.Nodes)))

	source := patch.Nodes[0]
	it.Then(t).
		Should(it.Equal("rescorla-2026-tls13", source.ID)).
		Should(it.Equal(core.Kind("source"), source.Kind)).
		Should(it.Equal("A design retrospective on the TLS 1.3 handshake.", source.Text))

	mentions, ok := source.Links["mentions"]
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal(1, len(mentions.Seq))).
		Should(it.Equal("Transport Layer Security", mentions.Seq[0].Target))

	entity := patch.Nodes[1]
	it.Then(t).
		Should(it.Equal("Transport Layer Security", entity.ID)).
		Should(it.Equal(core.Kind("entity"), entity.Kind))

	categories, ok := entity.Attrs["category"].([]any)
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal(4, len(categories)))
}

func TestParsePatchManifestMissingDocument(t *testing.T) {
	fixture := `---
kind: patch
published: 2026-04-12
---
# Source

## x
` + "```yaml\n```\n" + `
text.
`
	_, err := core.ParsePatch(strings.NewReader(fixture))
	it.Then(t).Should(it.True(errors.Is(err, core.ErrManifestInvalid)))
}

func TestParsePatchManifestMissingPublished(t *testing.T) {
	fixture := `---
kind: patch
document: foo-2026-x
---
# Source

## x
` + "```yaml\n```\n" + `
text.
`
	_, err := core.ParsePatch(strings.NewReader(fixture))
	it.Then(t).Should(it.True(errors.Is(err, core.ErrManifestInvalid)))
}

func TestParsePatchBodyMalformedNoHeading(t *testing.T) {
	fixture := `---
kind: patch
document: foo-2026-x
published: 2026-04-12
---
Just some prose, no H1/H2 structure at all.
`
	_, err := core.ParsePatch(strings.NewReader(fixture))
	it.Then(t).Should(it.True(errors.Is(err, core.ErrPatchStructure)))
}

func TestParsePatchBodyMalformedMissingYAMLFence(t *testing.T) {
	fixture := `---
kind: patch
document: foo-2026-x
published: 2026-04-12
---
# Source

## x
No fenced yaml block here.
`
	_, err := core.ParsePatch(strings.NewReader(fixture))
	it.Then(t).Should(it.True(errors.Is(err, core.ErrPatchStructure)))
}

// BUG-006 (corrects BUG-005's over-broad fix): a real extraction pipeline
// intentionally emits a "# Timeline" section alongside a document's own
// "# Source" section — ParsePatch parses it as an ordinary H1-kind/H2-node
// section like any other; it is internal/app/graph/service.Apply's job
// (not ParsePatch's) to fold it into the tool's own derived timeline index
// rather than writing it as a generic node file (research.md D8b revised).
func TestParsePatchTimelineKindSectionParsesAsOrdinaryNode(t *testing.T) {
	fixture := `---
kind: patch
document: foo-2026-x
published: 2026-07-12
---
# Source

## foo-2026-x
` + "```yaml\n```\n" + `
text.

# Timeline

## 2026-07
` + "```yaml\ngranularity: monthly\n```" + `
- [[foo-2026-x]]
`
	patch, err := core.ParsePatch(strings.NewReader(fixture))
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(2, len(patch.Nodes)))

	timelineNode := patch.Nodes[1]
	it.Then(t).
		Should(it.Equal(core.Kind("timeline"), timelineNode.Kind)).
		Should(it.Equal("2026-07", timelineNode.ID))
}

func TestParsePatchInlineWikilinkStrippedIntoHRefs(t *testing.T) {
	fixture := `---
kind: patch
document: foo-2026-x
published: 2026-04-12
---
# Source

## foo-2026-x
` + "```yaml\ntitle: \"X\"\n```" + `

This document discusses [[Transport Layer Security]] in depth.
`
	patch, err := core.ParsePatch(strings.NewReader(fixture))
	it.Then(t).Should(it.Nil(err))

	node := patch.Nodes[0]
	it.Then(t).
		Should(it.Equal("This document discusses Transport Layer Security in depth.", node.Text)).
		Should(it.Equal(1, len(node.HRefs))).
		Should(it.Equal("Transport Layer Security", node.HRefs[0].Target))
}

const entityNodeFixture = `---
kind: entity
title: Transport Layer Security
category: [independent, abstract, occurrent, script]
aliases: [TLS, TLS 1.3]
---
# Transport Layer Security

A cryptographic protocol that establishes an authenticated channel.

- replaces:: [[SSL Protocol]]
- conformsTo:: [[RFC 8446]]

## mentionedIn
- mentionedIn:: [[rescorla-2026-tls13]]
`

func TestParseNode(t *testing.T) {
	node, err := core.ParseNode(strings.NewReader(entityNodeFixture))

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal("Transport Layer Security", node.ID)).
		Should(it.Equal(core.Kind("entity"), node.Kind)).
		Should(it.Equal("A cryptographic protocol that establishes an authenticated channel.", node.Text)).
		Should(it.Equal(2, len(node.Edges)))

	it.Then(t).
		Should(it.Equal("replaces", node.Edges[0].Predicate)).
		Should(it.Equal("SSL Protocol", node.Edges[0].Target))

	mentionedIn, ok := node.Links["mentionedIn"]
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal(1, len(mentionedIn.Seq))).
		Should(it.Equal("rescorla-2026-tls13", mentionedIn.Seq[0].Target))
}

func TestParseNodeMissingKind(t *testing.T) {
	fixture := `---
title: X
---
# X

text.
`
	_, err := core.ParseNode(strings.NewReader(fixture))
	it.Then(t).Should(it.True(errors.Is(err, core.ErrManifestInvalid)))
}

func TestRenderNodeAttrsSortedKindFirst(t *testing.T) {
	n := core.Node{
		ID:   "X",
		Kind: "entity",
		Attrs: map[string]any{
			"title": "X",
			"tags":  []any{"a", "b"},
		},
		Text: "Some text.",
	}

	out, err := core.RenderNode(n)
	it.Then(t).Should(it.Nil(err))

	rendered := string(out)
	kindIdx := strings.Index(rendered, "kind:")
	tagsIdx := strings.Index(rendered, "tags:")
	titleIdx := strings.Index(rendered, "title:")

	it.Then(t).
		Should(it.True(kindIdx < tagsIdx)).
		Should(it.True(tagsIdx < titleIdx))
}

func TestRenderNodeEdgesBeforeLinksBlocksSortedByTitle(t *testing.T) {
	n := core.Node{
		ID:   "X",
		Kind: "entity",
		Text: "Some text.",
		Edges: []core.Link{
			{Predicate: "replaces", Target: "SSL Protocol"},
		},
		Links: map[string]core.LinkBlock{
			"mentions":    {Title: "Mentions", Seq: []core.Link{{Predicate: "mentions", Target: "A"}}},
			"mentionedIn": {Title: "AlreadyMentioned", Seq: []core.Link{{Predicate: "mentionedIn", Target: "B"}}},
		},
	}

	out, err := core.RenderNode(n)
	it.Then(t).Should(it.Nil(err))

	rendered := string(out)
	edgesIdx := strings.Index(rendered, "replaces:: [[SSL Protocol]]")
	alreadyIdx := strings.Index(rendered, "## AlreadyMentioned")
	mentionsIdx := strings.Index(rendered, "## Mentions")

	it.Then(t).
		Should(it.True(edgesIdx >= 0 && alreadyIdx >= 0 && mentionsIdx >= 0)).
		Should(it.True(edgesIdx < alreadyIdx)).
		Should(it.True(alreadyIdx < mentionsIdx))
}

func TestRenderNodeWikilinkRepeatedTargetOnlyOneLinked(t *testing.T) {
	n := core.Node{
		ID:   "X",
		Kind: "entity",
		Text: "Transport Layer Security is great. Transport Layer Security is a protocol.",
		HRefs: []core.Link{
			{Target: "Transport Layer Security"},
		},
	}

	out, err := core.RenderNode(n)
	it.Then(t).Should(it.Nil(err))

	rendered := string(out)
	it.Then(t).
		Should(it.Equal(1, strings.Count(rendered, "[[Transport Layer Security]]"))).
		Should(it.Equal(2, strings.Count(rendered, "Transport Layer Security")))
}

func TestRenderNodeWikilinkMidWordNotLinked(t *testing.T) {
	n := core.Node{
		ID:   "X",
		Kind: "entity",
		Text: "Insecurity is high here.",
		HRefs: []core.Link{
			{Target: "Security"},
		},
	}

	out, err := core.RenderNode(n)
	it.Then(t).Should(it.Nil(err))

	rendered := string(out)
	it.Then(t).
		Should(it.Equal(0, strings.Count(rendered, "[["))).
		Should(it.String(rendered).Contain("Insecurity is high here."))
}

func TestRenderNodeWikilinkPrecededByWhitespaceLinked(t *testing.T) {
	n := core.Node{
		ID:   "X",
		Kind: "entity",
		Text: "We discussed Security today.",
		HRefs: []core.Link{
			{Target: "Security"},
		},
	}

	out, err := core.RenderNode(n)
	it.Then(t).Should(it.Nil(err))

	rendered := string(out)
	it.Then(t).Should(it.String(rendered).Contain("We discussed [[Security]] today."))
}

func TestRenderNodeWikilinkAlreadyBracketedNotDoubleWrapped(t *testing.T) {
	n := core.Node{
		ID:   "X",
		Kind: "entity",
		Text: "See [[Security]] for details, not Security.",
		HRefs: []core.Link{
			{Target: "Security"},
		},
	}

	out, err := core.RenderNode(n)
	it.Then(t).Should(it.Nil(err))

	rendered := string(out)
	it.Then(t).
		Should(it.Equal(2, strings.Count(rendered, "[[Security]]"))).
		Should(it.Equal(0, strings.Count(rendered, "[[[[")))
}

func TestRoundTripSourceEntityResource(t *testing.T) {
	fixtures := map[core.Kind]string{
		"source": `---
kind: source
id: rescorla-2026-tls13
title: "TLS 1.3: Design and Rationale"
published: "2026-04-12"
authors: [Eric Rescorla]
url: https://example.org/tls13-design
---
# TLS 1.3: Design and Rationale

A design retrospective on the TLS 1.3 handshake.

## Mentions
- mentions:: [[Transport Layer Security]]
`,
		"entity": entityNodeFixture,
		"resource": `---
kind: resource
title: RFC 8446
ref: standard
authors: [Eric Rescorla]
year: 2018
url: https://www.rfc-editor.org/rfc/rfc8446
status: read
---
# RFC 8446

The normative specification of TLS 1.3.

## isCitedBy
- isCitedBy:: [[rescorla-2026-tls13]]
`,
	}

	for kind, fixture := range fixtures {
		t.Run(string(kind), func(t *testing.T) {
			first, err := core.ParseNode(strings.NewReader(fixture))
			it.Then(t).Should(it.Nil(err))

			rendered, err := core.RenderNode(first)
			it.Then(t).Should(it.Nil(err))

			second, err := core.ParseNode(strings.NewReader(string(rendered)))
			it.Then(t).Should(it.Nil(err))

			it.Then(t).
				Should(it.Equal(first.ID, second.ID)).
				Should(it.Equal(first.Kind, second.Kind)).
				Should(it.Equal(first.Text, second.Text)).
				Should(it.Equal(first.Notes, second.Notes)).
				Should(it.Equal(len(first.Edges), len(second.Edges))).
				Should(it.Equal(len(first.Links), len(second.Links)))
		})
	}
}

// BUG-003: CORE §12.2's canonical bold-label convention ("node bodies use
// bold labels, never headings") must be recognized for predicate-grouped
// blocks, with no data loss across multiple blocks. Fixture is this bug's
// own reported example.
const boldLabelThreeBlocksPatch = `---
kind: patch
document: dmitry-2026-graph
published: 2026-01-01
---
# Entity

## Arcnet-spec
` + "```yaml" + `
category: [independent, abstract, occurrent, script]
` + "```" + `

A lightweight ontology specification developed by the sender defining core graph structures, [[Article Extension]]s, and thought extensions for knowledge management.

**Mentioned In**
- mentionedIn:: [[dmitry-2026-graph]]

**Referenced By**
- referencedBy:: [[Core Thoughts Extension]]

**Related**
- related:: [[Article Extension]]
`

func TestParsePatchBoldLabelBlocksNoDataLoss(t *testing.T) {
	patch, err := core.ParsePatch(strings.NewReader(boldLabelThreeBlocksPatch))
	it.Then(t).Should(it.Nil(err))

	node := patch.Nodes[0]
	it.Then(t).
		Should(it.Equal("Arcnet-spec", node.ID)).
		ShouldNot(it.String(node.Text).Contain("**")).
		Should(it.Equal(3, len(node.Links)))

	mentionedIn, ok := node.Links["mentionedIn"]
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal("Mentioned In", mentionedIn.Title)).
		Should(it.Equal(1, len(mentionedIn.Seq))).
		Should(it.Equal("dmitry-2026-graph", mentionedIn.Seq[0].Target))

	referencedBy, ok := node.Links["referencedBy"]
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal("Core Thoughts Extension", referencedBy.Seq[0].Target))

	related, ok := node.Links["related"]
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal("Article Extension", related.Seq[0].Target))
}

const mixedBoldAndHeadingPatch = `---
kind: patch
document: foo-2026-x
published: 2026-01-01
---
# Entity

## Widget
` + "```yaml\n```" + `

A widget.

**Mentions**
- mentions:: [[A]]

## Cites
- cites:: [[B]]
`

func TestParsePatchMixedBoldLabelAndHeadingBlocks(t *testing.T) {
	patch, err := core.ParsePatch(strings.NewReader(mixedBoldAndHeadingPatch))
	it.Then(t).Should(it.Nil(err))

	node := patch.Nodes[0]
	it.Then(t).Should(it.Equal(2, len(node.Links)))

	mentions, ok := node.Links["mentions"]
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal("A", mentions.Seq[0].Target))

	cites, ok := node.Links["cites"]
	it.Then(t).
		Should(it.True(ok)).
		Should(it.Equal("B", cites.Seq[0].Target))
}
