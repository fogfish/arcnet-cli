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
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/core"
)

// testIndex covers every predicate this file's fixtures reference by a
// literal core.Link{Predicate: ...} or an inline "predicate::" markdown
// occurrence, so every RenderNode/RenderPatch call site below resolves a
// deterministic role rather than silently falling back to "edge" (research.md
// D3/D6). mentions/mentionedIn are link-role (grouped-heading candidates,
// Phase 3+); replaces is edge-role (stays a flat bullet); every other
// predicate literal present in a fixture (conformsTo, isCitedBy, cites,
// referencedBy, related, derivedFrom, assumes, addresses) is edge-role too,
// matching this feature's conservative default for a predicate that has no
// declared shape opinion of its own.
var testIndex = core.Index{
	Predicates: map[string]core.PredicateDef{
		"mentions":     {Role: "link"},
		"mentionedIn":  {Role: "link"},
		"replaces":     {Role: "edge"},
		"conformsTo":   {Role: "edge"},
		"isCitedBy":    {Role: "edge"},
		"cites":        {Role: "edge"},
		"referencedBy": {Role: "edge"},
		"related":      {Role: "edge"},
		"derivedFrom":  {Role: "edge"},
		"assumes":      {Role: "edge"},
		"addresses":    {Role: "edge"},
		// required mirrors CorePredicateDefs's own real entry exactly (an
		// explicit Label, link-role) — used by
		// TestRenderNodeLinkRolePredicateUsesCustomLabel.
		"required": {Role: "link", Label: "Requires"},
		// entries mirrors CorePredicateDefs's own real entry (link-role,
		// no explicit Label) — timeline's only edge-bearing predicate,
		// used by the single-link-role-predicate-body omission tests.
		"entries": {Role: "link"},
	},
}

const patchFixture = `---
kind: patch
document: rescorla-2026-tls13
published: 2026-04-12
title: "TLS 1.3: Design and Rationale"
---
# Source

## rescorla-2026-tls13
` + "```yaml" + `
"@id": rescorla-2026-tls13
"@type": source
title: "TLS 1.3: Design and Rationale"
authors: [Eric Rescorla]
published: "2026-04-12"
url: https://example.org/tls13-design
` + "```" + `

A design retrospective on the TLS 1.3 handshake.

- mentions:: [[Transport Layer Security]]

# Entity

## Transport Layer Security
` + "```yaml" + `
"@id": Transport Layer Security
"@type": entity
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
		Should(it.Equal("source", source.Type)).
		Should(it.Equal("A design retrospective on the TLS 1.3 handshake.", source.Texts["abstract"])).
		Should(it.Equal(1, len(source.Edges))).
		Should(it.Equal("Transport Layer Security", source.Edges[0].Target))

	entity := patch.Nodes[1]
	it.Then(t).
		Should(it.Equal("Transport Layer Security", entity.ID)).
		Should(it.Equal("entity", entity.Type))

	categories := entity.Attrs["category"]
	it.Then(t).Should(it.Equal(4, len(categories)))
}

func TestParsePatchManifestMissingDocument(t *testing.T) {
	fixture := `---
kind: patch
published: 2026-04-12
---
# Source

## x
` + "```yaml\n\"@id\": x\n\"@type\": source\n```\n" + `
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
` + "```yaml\n\"@id\": x\n\"@type\": source\n```\n" + `
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
// "# Source" section — ParsePatch parses it as an ordinary H1-type/H2-node
// section like any other; it is internal/app/graph/service.Apply's job
// (not ParsePatch's) to fold it into the tool's own derived timeline index
// rather than writing it as a generic node file (research.md D8b revised).
func TestParsePatchTimelineTypeSectionParsesAsOrdinaryNode(t *testing.T) {
	fixture := `---
kind: patch
document: foo-2026-x
published: 2026-07-12
---
# Source

## foo-2026-x
` + "```yaml\n\"@id\": foo-2026-x\n\"@type\": source\n```\n" + `
text.

# Timeline

## 2026-07
` + "```yaml\n\"@id\": \"2026-07\"\n\"@type\": timeline\ngranularity: monthly\n```" + `
- [[foo-2026-x]]
`
	patch, err := core.ParsePatch(strings.NewReader(fixture))
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(2, len(patch.Nodes)))

	timelineNode := patch.Nodes[1]
	it.Then(t).
		Should(it.Equal("timeline", timelineNode.Type)).
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
` + "```yaml\n\"@id\": foo-2026-x\n\"@type\": source\ntitle: \"X\"\n```" + `

This document discusses [[Transport Layer Security]] in depth.
`
	patch, err := core.ParsePatch(strings.NewReader(fixture))
	it.Then(t).Should(it.Nil(err))

	node := patch.Nodes[0]
	it.Then(t).
		Should(it.Equal("This document discusses Transport Layer Security in depth.", node.Texts["abstract"])).
		Should(it.Equal(1, len(node.HRefs))).
		Should(it.Equal("Transport Layer Security", node.HRefs[0].Target))
}

const entityNodeFixture = `---
"@id": Transport Layer Security
"@type": entity
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
		Should(it.Equal("entity", node.Type)).
		Should(it.Equal("A cryptographic protocol that establishes an authenticated channel.", node.Texts["definition"])).
		Should(it.Equal(3, len(node.Edges)))

	it.Then(t).
		Should(it.Equal("replaces", node.Edges[0].Predicate)).
		Should(it.Equal("SSL Protocol", node.Edges[0].Target)).
		Should(it.Equal("conformsTo", node.Edges[1].Predicate)).
		Should(it.Equal("RFC 8446", node.Edges[1].Target)).
		Should(it.Equal("mentionedIn", node.Edges[2].Predicate)).
		Should(it.Equal("rescorla-2026-tls13", node.Edges[2].Target))
}

func TestParseNodeLegacyKindFieldRejected(t *testing.T) {
	fixture := `---
kind: entity
title: X
---
# X

text.
`
	node, err := core.ParseNode(strings.NewReader(fixture))
	it.Then(t).
		Should(it.True(errors.Is(err, core.ErrManifestInvalid))).
		Should(it.Equal("", node.ID)).
		Should(it.Equal(0, len(node.Attrs)))
}

func TestParseNodeMissingIDRejected(t *testing.T) {
	fixture := `---
"@type": entity
---
# X

text.
`
	node, err := core.ParseNode(strings.NewReader(fixture))
	it.Then(t).
		Should(it.True(errors.Is(err, core.ErrManifestInvalid))).
		Should(it.Equal("", node.ID)).
		Should(it.Equal(0, len(node.Attrs)))
}

func TestParseNodeMissingTypeRejected(t *testing.T) {
	fixture := `---
"@id": X
---
# X

text.
`
	node, err := core.ParseNode(strings.NewReader(fixture))
	it.Then(t).
		Should(it.True(errors.Is(err, core.ErrManifestInvalid))).
		Should(it.Equal("", node.ID)).
		Should(it.Equal(0, len(node.Attrs)))
}

func TestParsePatchNodeLegacyKindFieldInFenceRejected(t *testing.T) {
	fixture := `---
kind: patch
document: foo-2026-x
published: 2026-04-12
---
# Entity

## X
` + "```yaml\nkind: entity\n\"@id\": X\n\"@type\": entity\n```" + `

text.
`
	patch, err := core.ParsePatch(strings.NewReader(fixture))
	it.Then(t).
		Should(it.True(errors.Is(err, core.ErrManifestInvalid))).
		Should(it.Equal(0, len(patch.Nodes)))
}

// BUG-001: a patch node section following CORE §12.2's own canonical
// convention — no "@id"/"@type" duplicated inside the yaml fence at all —
// is accepted: "@id" comes from the "## <ID>" heading, "@type" from the
// enclosing "# <Type>" heading (lowercased). This is the shape every
// pre-existing patch fixture, and real external patch producers (e.g.
// fogfish/bots), already use.
func TestParsePatchNodeHeadingOnlyNoExplicitIdentitySucceeds(t *testing.T) {
	fixture := `---
kind: patch
document: foo-2026-x
published: 2026-04-12
---
# Entity

## X
` + "```yaml\ntitle: X\n```" + `

text.
`
	patch, err := core.ParsePatch(strings.NewReader(fixture))
	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, len(patch.Nodes))).
		Should(it.Equal("X", patch.Nodes[0].ID)).
		Should(it.Equal("entity", patch.Nodes[0].Type))
}

// BUG-001: an explicit "@id" key that agrees with the section heading is
// accepted even with no explicit "@type" present — "@type" still derives
// from the "# <Type>" heading.
func TestParsePatchNodeExplicitIdAgreeingNoExplicitTypeSucceeds(t *testing.T) {
	fixture := `---
kind: patch
document: foo-2026-x
published: 2026-04-12
---
# Entity

## X
` + "```yaml\n\"@id\": X\n```" + `

text.
`
	patch, err := core.ParsePatch(strings.NewReader(fixture))
	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, len(patch.Nodes))).
		Should(it.Equal("X", patch.Nodes[0].ID)).
		Should(it.Equal("entity", patch.Nodes[0].Type))
}

// BUG-001: an explicit "@type" key that agrees with the enclosing heading
// (case-insensitively, matching RenderPatch's own title-cased "# <Type>"
// output) is accepted even with no explicit "@id" present.
func TestParsePatchNodeExplicitTypeAgreeingNoExplicitIdSucceeds(t *testing.T) {
	fixture := `---
kind: patch
document: foo-2026-x
published: 2026-04-12
---
# Entity

## X
` + "```yaml\n\"@type\": entity\n```" + `

text.
`
	patch, err := core.ParsePatch(strings.NewReader(fixture))
	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(1, len(patch.Nodes))).
		Should(it.Equal("X", patch.Nodes[0].ID)).
		Should(it.Equal("entity", patch.Nodes[0].Type))
}

// BUG-001: an explicit "@type" key that disagrees with the enclosing
// "# <Type>" heading is rejected as inconsistent, exactly like a
// disagreeing explicit "@id" already was.
func TestParsePatchNodeExplicitTypeDisagreeingHeadingRejected(t *testing.T) {
	fixture := `---
kind: patch
document: foo-2026-x
published: 2026-04-12
---
# Entity

## X
` + "```yaml\n\"@type\": resource\n```" + `

text.
`
	_, err := core.ParsePatch(strings.NewReader(fixture))
	it.Then(t).Should(it.True(errors.Is(err, core.ErrManifestInvalid)))
}

// spec FR-011/data-model.md: a patch node contribution's "@id" MUST equal
// its "## <ID>" section heading, with no fallback — the same rule a
// standalone file applies against its basename, verifiable here directly
// since the heading text lives in the same document being parsed.
func TestParsePatchNodeIDMismatchHeadingRejected(t *testing.T) {
	fixture := `---
kind: patch
document: foo-2026-x
published: 2026-04-12
---
# Entity

## X
` + "```yaml\n\"@id\": Y\n\"@type\": entity\n```" + `

text.
`
	_, err := core.ParsePatch(strings.NewReader(fixture))
	it.Then(t).Should(it.True(errors.Is(err, core.ErrManifestInvalid)))
}

func TestRenderNodeAttrsQuotedIDTypeFirstThenSorted(t *testing.T) {
	n := core.Node{
		ID:   "X",
		Type: "entity",
		Attrs: map[string][]core.Predicate{
			"title": {{Value: "X"}},
			"tags":  {{Value: "a"}, {Value: "b"}},
		},
		Texts: map[string]string{"definition": "Some text."},
	}

	out, err := core.RenderNode(n, testIndex)
	it.Then(t).Should(it.Nil(err))

	rendered := string(out)
	it.Then(t).
		Should(it.String(rendered).Contain(`"@id": X`)).
		Should(it.String(rendered).Contain(`"@type": entity`))

	idIdx := strings.Index(rendered, `"@id"`)
	typeIdx := strings.Index(rendered, `"@type"`)
	tagsIdx := strings.Index(rendered, "tags:")
	titleIdx := strings.Index(rendered, "title:")

	it.Then(t).
		Should(it.True(idIdx >= 0 && idIdx < typeIdx)).
		Should(it.True(typeIdx < tagsIdx)).
		Should(it.True(tagsIdx < titleIdx))
}

// TestRenderNodeSchemaDrivenFlatAndGroupedMixOnOneNode (research.md D8):
// replaces is edge-role in testIndex and renders as a flat bullet with no
// heading; mentions/mentionedIn are link-role and each render grouped under
// their own heading, default-capitalized since neither declares an explicit
// Label in testIndex — the same fixture TestRenderNodeEdgesFlatBulletedList
// NoGroupedHeadings used to assert always-flat, now asserting the
// schema-driven mixed shape instead.
func TestRenderNodeSchemaDrivenFlatAndGroupedMixOnOneNode(t *testing.T) {
	n := core.Node{
		ID:    "X",
		Type:  "entity",
		Texts: map[string]string{"definition": "Some text."},
		Edges: []core.Link{
			{Predicate: "replaces", Target: "SSL Protocol"},
			{Predicate: "mentions", Target: "A"},
			{Predicate: "mentionedIn", Target: "B"},
		},
	}

	out, err := core.RenderNode(n, testIndex)
	it.Then(t).Should(it.Nil(err))

	rendered := string(out)
	it.Then(t).
		Should(it.String(rendered).Contain("replaces:: [[SSL Protocol]]")).
		Should(it.String(rendered).Contain("## Mentions")).
		Should(it.String(rendered).Contain("mentions:: [[A]]")).
		Should(it.String(rendered).Contain("## MentionedIn")).
		Should(it.String(rendered).Contain("mentionedIn:: [[B]]"))

	replacesIdx := strings.Index(rendered, "replaces::")
	replacesHeadingIdx := strings.Index(rendered, "## ")
	it.Then(t).
		ShouldNot(it.True(replacesHeadingIdx >= 0 && replacesHeadingIdx < replacesIdx))

	mentionsHeadingIdx := strings.Index(rendered, "## Mentions")
	mentionsBulletIdx := strings.Index(rendered, "mentions:: [[A]]")
	mentionedInHeadingIdx := strings.Index(rendered, "## MentionedIn")
	mentionedInBulletIdx := strings.Index(rendered, "mentionedIn:: [[B]]")
	it.Then(t).
		Should(it.True(replacesIdx < mentionedInHeadingIdx)).
		Should(it.True(mentionedInHeadingIdx < mentionedInBulletIdx)).
		Should(it.True(mentionedInBulletIdx < mentionsHeadingIdx)).
		Should(it.True(mentionsHeadingIdx < mentionsBulletIdx))
}

// TestRenderNodeLinkRolePredicateUsesCustomLabel (research.md D4): a
// link-role predicate whose PredicateDef.Label is non-empty (testIndex's
// "required", mirroring CorePredicateDefs's own real entry) renders its
// heading using that label, not the default-capitalized predicate name.
func TestRenderNodeLinkRolePredicateUsesCustomLabel(t *testing.T) {
	// A second, edge-role predicate is present alongside "required" so the
	// single-link-role-predicate-body omission rule (spec FR-006/FR-007)
	// does not itself suppress "required"'s heading — this test is about
	// label resolution, not the omission rule (covered separately by
	// TestRenderNodeSingleLinkRolePredicateBodyOmitsHeading).
	n := core.Node{
		ID:    "X",
		Type:  "entity",
		Texts: map[string]string{"definition": "Some text."},
		Edges: []core.Link{
			{Predicate: "required", Target: "title"},
			{Predicate: "replaces", Target: "Y"},
		},
	}

	out, err := core.RenderNode(n, testIndex)
	it.Then(t).Should(it.Nil(err))

	rendered := string(out)
	it.Then(t).
		Should(it.String(rendered).Contain("## Requires")).
		ShouldNot(it.String(rendered).Contain("## Required"))
}

// TestRenderNodeUnregisteredPredicateDefaultsToFlatEdge (spec FR-013,
// research.md D3): a predicate absent from the index entirely renders as a
// flat bullet with no heading — the conservative default.
func TestRenderNodeUnregisteredPredicateDefaultsToFlatEdge(t *testing.T) {
	n := core.Node{
		ID:    "X",
		Type:  "entity",
		Texts: map[string]string{"definition": "Some text."},
		Edges: []core.Link{
			{Predicate: "unregisteredPredicate", Target: "Y"},
		},
	}

	out, err := core.RenderNode(n, testIndex)
	it.Then(t).Should(it.Nil(err))

	rendered := string(out)
	it.Then(t).
		ShouldNot(it.String(rendered).Contain("## ")).
		Should(it.String(rendered).Contain("unregisteredPredicate:: [[Y]]"))
}

// TestRenderNodeSingleLinkRolePredicateBodyOmitsHeading (spec Acceptance
// Scenario 1, research.md D5): a timeline-typed node whose only Edges are
// entries occurrences (link-role, testIndex's own single edge-bearing
// predicate for this type) renders as a bare bulleted list with no
// "## Entries" heading — the redundant-heading case this feature's own
// single-group omission rule exists to eliminate.
func TestRenderNodeSingleLinkRolePredicateBodyOmitsHeading(t *testing.T) {
	n := core.Node{
		ID:   "2026",
		Type: "timeline",
		Edges: []core.Link{
			{Predicate: "entries", Target: "foo-2026-x"},
			{Predicate: "entries", Target: "bar-2026-y"},
		},
	}

	out, err := core.RenderNode(n, testIndex)
	it.Then(t).Should(it.Nil(err))

	rendered := string(out)
	it.Then(t).
		ShouldNot(it.String(rendered).Contain("## ")).
		Should(it.String(rendered).Contain("entries:: [[foo-2026-x]]")).
		Should(it.String(rendered).Contain("entries:: [[bar-2026-y]]"))
}

// TestRenderNodeSingleLinkRolePredicateHeadingReappearsWithOtherContent
// (spec Acceptance Scenario 2, Edge Case "two-or-more distinct link-role
// predicates"): the same entries-only fixture plus one additional
// predicate's occurrence present in Edges causes "## Entries" to reappear —
// the omission is presence-based, not permission-based (research.md D5).
func TestRenderNodeSingleLinkRolePredicateHeadingReappearsWithOtherContent(t *testing.T) {
	n := core.Node{
		ID:   "2026",
		Type: "timeline",
		Edges: []core.Link{
			{Predicate: "entries", Target: "foo-2026-x"},
			{Predicate: "mentions", Target: "Something Else"},
		},
	}

	out, err := core.RenderNode(n, testIndex)
	it.Then(t).Should(it.Nil(err))

	rendered := string(out)
	it.Then(t).
		Should(it.String(rendered).Contain("## Entries")).
		Should(it.String(rendered).Contain("entries:: [[foo-2026-x]]")).
		Should(it.String(rendered).Contain("## Mentions")).
		Should(it.String(rendered).Contain("mentions:: [[Something Else]]"))
}

func TestRenderNodeWikilinkRepeatedTargetOnlyOneLinked(t *testing.T) {
	n := core.Node{
		ID:   "X",
		Type: "entity",
		Texts: map[string]string{
			"definition": "Transport Layer Security is great. Transport Layer Security is a protocol.",
		},
		HRefs: []core.Link{
			{Target: "Transport Layer Security"},
		},
	}

	out, err := core.RenderNode(n, testIndex)
	it.Then(t).Should(it.Nil(err))

	rendered := string(out)
	it.Then(t).
		Should(it.Equal(1, strings.Count(rendered, "[[Transport Layer Security]]"))).
		Should(it.Equal(2, strings.Count(rendered, "Transport Layer Security")))
}

func TestRenderNodeWikilinkMidWordNotLinked(t *testing.T) {
	n := core.Node{
		ID:    "X",
		Type:  "entity",
		Texts: map[string]string{"definition": "Insecurity is high here."},
		HRefs: []core.Link{
			{Target: "Security"},
		},
	}

	out, err := core.RenderNode(n, testIndex)
	it.Then(t).Should(it.Nil(err))

	rendered := string(out)
	it.Then(t).
		Should(it.Equal(0, strings.Count(rendered, "[["))).
		Should(it.String(rendered).Contain("Insecurity is high here."))
}

func TestRenderNodeWikilinkPrecededByWhitespaceLinked(t *testing.T) {
	n := core.Node{
		ID:    "X",
		Type:  "entity",
		Texts: map[string]string{"definition": "We discussed Security today."},
		HRefs: []core.Link{
			{Target: "Security"},
		},
	}

	out, err := core.RenderNode(n, testIndex)
	it.Then(t).Should(it.Nil(err))

	rendered := string(out)
	it.Then(t).Should(it.String(rendered).Contain("We discussed [[Security]] today."))
}

func TestRenderNodeWikilinkAlreadyBracketedNotDoubleWrapped(t *testing.T) {
	n := core.Node{
		ID:    "X",
		Type:  "entity",
		Texts: map[string]string{"definition": "See [[Security]] for details, not Security."},
		HRefs: []core.Link{
			{Target: "Security"},
		},
	}

	out, err := core.RenderNode(n, testIndex)
	it.Then(t).Should(it.Nil(err))

	rendered := string(out)
	it.Then(t).
		Should(it.Equal(2, strings.Count(rendered, "[[Security]]"))).
		Should(it.Equal(0, strings.Count(rendered, "[[[[")))
}

// TestRoundTripCoreWorkedExamples covers ARCNET-CORE §11's source/entity/
// resource/timeline worked examples, plus one DOMAIN-ARTICLE-style
// hypothesis example using edge predicates derivedFrom/assumes/addresses
// (tasks.md T029).
func TestRoundTripCoreWorkedExamples(t *testing.T) {
	fixtures := map[string]string{
		"source": `---
"@id": rescorla-2026-tls13
"@type": source
title: "TLS 1.3: Design and Rationale"
published: "2026-04-12"
authors: [Eric Rescorla]
url: https://example.org/tls13-design
---
# rescorla-2026-tls13

A design retrospective on the TLS 1.3 handshake.

- mentions:: [[Transport Layer Security]]
`,
		"entity": entityNodeFixture,
		"resource": `---
"@id": RFC 8446
"@type": resource
title: RFC 8446
ref: standard
authors: [Eric Rescorla]
year: 2018
url: https://www.rfc-editor.org/rfc/rfc8446
status: read
---
# RFC 8446

The normative specification of TLS 1.3.

- isCitedBy:: [[rescorla-2026-tls13]]
`,
		"timeline": `---
"@id": "2026-04"
"@type": timeline
granularity: monthly
---
# 2026-04

- [[rescorla-2026-tls13]]
`,
		"hypothesis": `---
"@id": tls13-forward-secrecy
"@type": hypothesis
status: open
---
# tls13-forward-secrecy

TLS 1.3 handshakes provide forward secrecy by default.

- derivedFrom:: [[rescorla-2026-tls13]]
- assumes:: [[Transport Layer Security]]
- addresses:: [[Key Compromise]]
`,
	}

	for typ, fixture := range fixtures {
		t.Run(typ, func(t *testing.T) {
			first, err := core.ParseNode(strings.NewReader(fixture))
			it.Then(t).Should(it.Nil(err))
			it.Then(t).Should(it.Equal(typ, first.Type))

			rendered, err := core.RenderNode(first, testIndex)
			it.Then(t).Should(it.Nil(err))

			second, err := core.ParseNode(strings.NewReader(string(rendered)))
			it.Then(t).Should(it.Nil(err))

			it.Then(t).
				Should(it.Equal(first.ID, second.ID)).
				Should(it.Equal(first.Type, second.Type)).
				Should(it.Equal(len(first.Texts), len(second.Texts))).
				Should(it.Equal(len(first.Edges), len(second.Edges)))

			for k, v := range first.Texts {
				it.Then(t).Should(it.Equal(v, second.Texts[k]))
			}
		})
	}
}

// BUG-003: CORE §12.2's canonical bold-label convention ("node bodies use
// bold labels, never headings") must be recognized for predicate-grouped
// blocks, with no data loss across multiple blocks — now flattened into one
// Edges slice (research.md D5), not a Links map. Fixture is this bug's own
// reported example.
const boldLabelThreeBlocksPatch = `---
kind: patch
document: dmitry-2026-graph
published: 2026-01-01
---
# Entity

## Arcnet-spec
` + "```yaml\n\"@id\": Arcnet-spec\n\"@type\": entity\ncategory: [independent, abstract, occurrent, script]\n```" + `

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
		ShouldNot(it.String(node.Texts["definition"]).Contain("**")).
		Should(it.Equal(3, len(node.Edges)))

	it.Then(t).
		Should(it.Equal("mentionedIn", node.Edges[0].Predicate)).
		Should(it.Equal("dmitry-2026-graph", node.Edges[0].Target)).
		Should(it.Equal("referencedBy", node.Edges[1].Predicate)).
		Should(it.Equal("Core Thoughts Extension", node.Edges[1].Target)).
		Should(it.Equal("related", node.Edges[2].Predicate)).
		Should(it.Equal("Article Extension", node.Edges[2].Target))
}

const mixedBoldAndHeadingPatch = `---
kind: patch
document: foo-2026-x
published: 2026-01-01
---
# Entity

## Widget
` + "```yaml\n\"@id\": Widget\n\"@type\": entity\n```" + `

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
	it.Then(t).Should(it.Equal(2, len(node.Edges)))

	it.Then(t).
		Should(it.Equal("mentions", node.Edges[0].Predicate)).
		Should(it.Equal("A", node.Edges[0].Target)).
		Should(it.Equal("cites", node.Edges[1].Predicate)).
		Should(it.Equal("B", node.Edges[1].Target))
}

// sortedByTypeThenID mirrors RenderPatch's own deterministic ordering
// (research.md D9), so a round-trip test can compare the parsed-back node
// set against the input regardless of the input slice's original order.
func sortedByTypeThenID(nodes []core.Node) []core.Node {
	out := append([]core.Node(nil), nodes...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func TestRenderPatchRoundTripsSingleNode(t *testing.T) {
	p := core.Patch{
		Document:  "foo-2026-x",
		Published: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Title:     "Foo",
		Nodes: []core.Node{
			{
				ID:    "Widget",
				Type:  "entity",
				Attrs: map[string][]core.Predicate{"category": {{Value: "form"}}},
				Texts: map[string]string{"definition": "A widget."},
			},
		},
	}

	raw, err := core.RenderPatch(p, testIndex)
	it.Then(t).Should(it.Nil(err))

	back, err := core.ParsePatch(strings.NewReader(string(raw)))
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.Equal("foo-2026-x", back.Document)).
		Should(it.Equal(1, len(back.Nodes)))

	gotNode, wantNode := back.Nodes[0], p.Nodes[0]
	it.Then(t).
		Should(it.Equal(wantNode.ID, gotNode.ID)).
		Should(it.Equal(wantNode.Type, gotNode.Type)).
		Should(it.Equal(wantNode.Texts["definition"], gotNode.Texts["definition"])).
		Should(it.Equal(1, len(gotNode.Attrs["category"])))
}

// The per-node yaml fence always carries both "@id" and "@type" (both
// quoted keys), regardless of what Attrs itself holds — unlike the old
// shape, there is no "guarantee id survives in Attrs" fallback anymore
// since "@id"/"@type" are now dedicated top-level fields, never sourced
// from Attrs.
func TestRenderPatchFenceAlwaysHasQuotedIDAndType(t *testing.T) {
	p := core.Patch{
		Document:  "foo-2026-x2",
		Published: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Nodes: []core.Node{
			{ID: "Widget", Type: "entity", Attrs: map[string][]core.Predicate{"category": {{Value: "form"}}}, Texts: map[string]string{"definition": "A widget."}},
		},
	}

	raw, err := core.RenderPatch(p, testIndex)
	it.Then(t).Should(it.Nil(err))

	rendered := string(raw)
	it.Then(t).
		Should(it.String(rendered).Contain(`"@id": Widget`)).
		Should(it.String(rendered).Contain(`"@type": entity`))

	back, err := core.ParsePatch(strings.NewReader(rendered))
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(back.Nodes)))
	it.Then(t).Should(it.Equal("Widget", back.Nodes[0].ID))
}

func TestRenderPatchRoundTripsMultipleTypesSortedDeterministically(t *testing.T) {
	nodes := []core.Node{
		{ID: "z-source", Type: "source", Texts: map[string]string{"abstract": "z body."}},
		{ID: "Widget", Type: "entity", Attrs: map[string][]core.Predicate{"category": {{Value: "form"}}}, Texts: map[string]string{"definition": "widget body."}},
		{ID: "a-source", Type: "source", Texts: map[string]string{"abstract": "a body."}},
	}
	p := core.Patch{Document: "foo-2026-y", Published: time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC), Nodes: nodes}

	raw, err := core.RenderPatch(p, testIndex)
	it.Then(t).Should(it.Nil(err))

	// Types sorted alphabetically ("entity" before "source"); within
	// "source", IDs sorted alphabetically ("a-source" before "z-source") —
	// research.md D9.
	out := string(raw)
	entityIdx := strings.Index(out, "# Entity")
	sourceIdx := strings.Index(out, "# Source")
	aIdx := strings.Index(out, "## a-source")
	zIdx := strings.Index(out, "## z-source")
	it.Then(t).
		Should(it.True(entityIdx >= 0 && sourceIdx > entityIdx)).
		Should(it.True(aIdx >= 0 && zIdx > aIdx))

	back, err := core.ParsePatch(strings.NewReader(out))
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(len(nodes), len(back.Nodes)))

	gotSorted, wantSorted := sortedByTypeThenID(back.Nodes), sortedByTypeThenID(nodes)
	for i := range wantSorted {
		it.Then(t).
			Should(it.Equal(wantSorted[i].ID, gotSorted[i].ID)).
			Should(it.Equal(wantSorted[i].Type, gotSorted[i].Type))
	}
}

func TestRenderPatchRoundTripsNodeWithEdgesTextsHRefs(t *testing.T) {
	nodes := []core.Node{
		{
			ID:    "Transport Layer Security",
			Type:  "entity",
			Attrs: map[string][]core.Predicate{"category": {{Value: "form"}}},
			Texts: map[string]string{
				"definition": "TLS is the successor to SSL.",
				"notes":      "See also RFC 8446.",
			},
			Edges: []core.Link{
				{Target: "rescorla-2026-tls13"},
				{Predicate: "mentions", Target: "SSL"},
			},
		},
	}
	p := core.Patch{Document: "foo-2026-z", Published: time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC), Nodes: nodes}

	raw, err := core.RenderPatch(p, testIndex)
	it.Then(t).Should(it.Nil(err))

	back, err := core.ParsePatch(strings.NewReader(string(raw)))
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(back.Nodes)))

	got := back.Nodes[0]
	it.Then(t).
		Should(it.Equal("Transport Layer Security", got.ID)).
		Should(it.Equal("TLS is the successor to SSL.", got.Texts["definition"])).
		Should(it.Equal("See also RFC 8446.", got.Texts["notes"])).
		Should(it.Equal(2, len(got.Edges))).
		Should(it.Equal("rescorla-2026-tls13", got.Edges[0].Target)).
		Should(it.Equal("mentions", got.Edges[1].Predicate)).
		Should(it.Equal("SSL", got.Edges[1].Target))
}

// TestRenderPatchStableAcrossHeadingGroupReordering (spec FR-010,
// contracts/render-shape-contract.md round-trip guarantees): re-rendering
// never reorders anything beyond what the contract permits — the flat
// edge-role list always precedes any link-role group, groups are ordered by
// resolved label ascending (mentionedIn before mentions), and no Link's
// Predicate/Target/Alias is ever altered, dropped, or duplicated across a
// parse/render/parse/render cycle. Per BUG-001/research.md D10, a
// RenderPatch link-role group renders as a "**Label**" bold-label
// paragraph, never a "## Label" heading — ARCNET-CORE §14.2 reserves "##"
// exclusively for a patch document's own @type/@id structure.
func TestRenderPatchStableAcrossHeadingGroupReordering(t *testing.T) {
	nodes := []core.Node{
		{
			ID:   "Widget",
			Type: "entity",
			Edges: []core.Link{
				{Predicate: "mentionedIn", Target: "B"},
				{Predicate: "replaces", Target: "SSL Protocol"},
				{Predicate: "mentions", Target: "A"},
			},
		},
	}
	p := core.Patch{Document: "foo-2026-x", Published: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Nodes: nodes}

	first, err := core.RenderPatch(p, testIndex)
	it.Then(t).Should(it.Nil(err))

	out := string(first)
	it.Then(t).
		ShouldNot(it.String(out).Contain("## MentionedIn")).
		ShouldNot(it.String(out).Contain("## Mentions"))

	replacesIdx := strings.Index(out, "replaces::")
	mentionedInLabelIdx := strings.Index(out, "**MentionedIn**")
	mentionsLabelIdx := strings.Index(out, "**Mentions**")
	it.Then(t).
		Should(it.True(replacesIdx >= 0 && mentionedInLabelIdx >= 0 && mentionsLabelIdx >= 0)).
		Should(it.True(replacesIdx < mentionedInLabelIdx)).
		Should(it.True(mentionedInLabelIdx < mentionsLabelIdx))

	back, err := core.ParsePatch(strings.NewReader(out))
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(back.Nodes)))

	wantTargets := map[string]string{}
	for _, e := range nodes[0].Edges {
		wantTargets[e.Predicate] = e.Target
	}
	gotTargets := map[string]string{}
	for _, e := range back.Nodes[0].Edges {
		gotTargets[e.Predicate] = e.Target
	}
	it.Then(t).Should(it.Equal(len(wantTargets), len(gotTargets)))
	for k, v := range wantTargets {
		it.Then(t).Should(it.Equal(v, gotTargets[k]))
	}

	second, err := core.RenderPatch(core.Patch{Document: p.Document, Published: p.Published, Nodes: back.Nodes}, testIndex)
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(out, string(second)))
}

// TestRenderPatchBoldLabelRoundTripsWithoutParserChange (BUG-001, T035):
// confirms walkNodeBody's existing bold-label recognition (blockTitle/
// boldLabel, introduced for BUG-003) already parses RenderPatch's corrected
// "**Label**" bold-label block back into the same Edges shape, with no
// parser change needed — RenderPatch(ParsePatch(RenderPatch(n, testIndex)),
// testIndex) is byte-equal to RenderPatch(n, testIndex), mirroring
// TestIdempotentRoundTrip's RenderNode pattern but for the patch format's
// bold-label markup.
func TestRenderPatchBoldLabelRoundTripsWithoutParserChange(t *testing.T) {
	p := core.Patch{
		Document:  "foo-2026-bold",
		Published: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Nodes: []core.Node{
			{
				ID:   "BoldRoundTrip",
				Type: "entity",
				Edges: []core.Link{
					{Predicate: "replaces", Target: "SSL Protocol"},
					{Predicate: "mentions", Target: "A"},
					{Predicate: "mentionedIn", Target: "B"},
				},
			},
		},
	}

	first, err := core.RenderPatch(p, testIndex)
	it.Then(t).Should(it.Nil(err))

	out := string(first)
	it.Then(t).
		ShouldNot(it.String(out).Contain("## Mentions")).
		ShouldNot(it.String(out).Contain("## MentionedIn")).
		Should(it.String(out).Contain("## BoldRoundTrip")). // the node's own @id H2 heading is expected
		Should(it.String(out).Contain("**Mentions**")).
		Should(it.String(out).Contain("**MentionedIn**"))

	back, err := core.ParsePatch(strings.NewReader(out))
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(back.Nodes)))
	it.Then(t).Should(it.Equal(3, len(back.Nodes[0].Edges)))

	second, err := core.RenderPatch(core.Patch{Document: p.Document, Published: p.Published, Nodes: back.Nodes}, testIndex)
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(out, string(second)))
}

// research.md D2: a "published" front-matter key is decoded into
// Node.Published, not left in Attrs.
func TestParseNodeExtractsPublishedNeverLeftInAttrs(t *testing.T) {
	fixture := `---
"@id": X
"@type": entity
published: "2026-04-12"
---
# X

Some text.
`
	node, err := core.ParseNode(strings.NewReader(fixture))
	it.Then(t).Should(it.Nil(err))

	it.Then(t).
		ShouldNot(it.True(node.Published.IsZero())).
		Should(it.Equal(2026, node.Published.Year())).
		Should(it.Equal(4, int(node.Published.Month()))).
		Should(it.Equal(12, node.Published.Day()))

	_, hasPublished := node.Attrs["published"]
	it.Then(t).ShouldNot(it.True(hasPublished))
}

// research.md D2: a "published" yaml-fence key within a patch's per-node
// section is likewise decoded into Node.Published, not left in Attrs.
func TestParsePatchBodyExtractsPublishedNeverLeftInAttrs(t *testing.T) {
	fixture := `---
kind: patch
document: foo-2026-x
published: 2026-04-12
---
# Entity

## X
` + "```yaml\n\"@id\": X\n\"@type\": entity\npublished: \"2026-05-01\"\n```" + `

Some text.
`
	patch, err := core.ParsePatch(strings.NewReader(fixture))
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(patch.Nodes)))

	node := patch.Nodes[0]
	it.Then(t).
		ShouldNot(it.True(node.Published.IsZero())).
		Should(it.Equal(5, int(node.Published.Month())))

	_, hasPublished := node.Attrs["published"]
	it.Then(t).ShouldNot(it.True(hasPublished))
}

// data-model.md: RenderNode renders a non-zero Published back into front
// matter, at its sorted-attribute position, date-only formatted.
func TestRenderNodeRendersNonZeroPublished(t *testing.T) {
	n := core.Node{
		ID:        "X",
		Type:      "entity",
		Published: time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC),
		Attrs:     map[string][]core.Predicate{"title": {{Value: "X"}}},
		Texts:     map[string]string{"definition": "Some text."},
	}

	out, err := core.RenderNode(n, testIndex)
	it.Then(t).Should(it.Nil(err))

	rendered := string(out)
	it.Then(t).Should(it.String(rendered).Contain(`published: "2026-04-12"`))

	publishedIdx := strings.Index(rendered, "published:")
	titleIdx := strings.Index(rendered, "title:")
	it.Then(t).Should(it.True(publishedIdx >= 0 && titleIdx < publishedIdx))
}

// data-model.md: RenderNode omits Published entirely when zero (a stub or
// schema document never gains a "published:" line).
func TestRenderNodeOmitsZeroPublished(t *testing.T) {
	n := core.Node{ID: "X", Type: "entity", Attrs: map[string][]core.Predicate{"title": {{Value: "X"}}}}

	out, err := core.RenderNode(n, testIndex)
	it.Then(t).Should(it.Nil(err))
	it.Then(t).ShouldNot(it.String(string(out)).Contain("published:"))
}

// data-model.md: RenderPatch's per-node yaml fence likewise renders a
// non-zero Published.
func TestRenderPatchRendersNonZeroPublished(t *testing.T) {
	p := core.Patch{
		Document:  "foo-2026-x",
		Published: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Nodes: []core.Node{
			{ID: "Widget", Type: "entity", Published: time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC), Attrs: map[string][]core.Predicate{"category": {{Value: "form"}}}, Texts: map[string]string{"definition": "A widget."}},
		},
	}

	raw, err := core.RenderPatch(p, testIndex)
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.String(string(raw)).Contain(`published: "2026-04-12"`))
}

// AST contract: ParseNode(RenderNode(n)) round-trips Published exactly,
// the same lossless-conversion invariant every other field already holds.
func TestRoundTripPublished(t *testing.T) {
	n := core.Node{
		ID:        "X",
		Type:      "entity",
		Published: time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC),
		Attrs:     map[string][]core.Predicate{"title": {{Value: "X"}}},
		Texts:     map[string]string{"definition": "Some text."},
	}

	raw, err := core.RenderNode(n, testIndex)
	it.Then(t).Should(it.Nil(err))

	back, err := core.ParseNode(strings.NewReader(string(raw)))
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(n.Published, back.Published))
}

func TestRenderPatchNoLegacyKindFieldInsidePerNodeFence(t *testing.T) {
	p := core.Patch{
		Document:  "foo-2026-w",
		Published: time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC),
		Nodes:     []core.Node{{ID: "x", Type: "entity", Attrs: map[string][]core.Predicate{}}},
	}

	raw, err := core.RenderPatch(p, testIndex)
	it.Then(t).Should(it.Nil(err))

	// The per-node fence must never declare a bare "kind:" field (research.md
	// D1/D2) — a node's type is declared via its own quoted "@type" key; the
	// top-level manifest's own "kind: patch" line is the only permitted
	// "kind:" occurrence in the whole document.
	it.Then(t).Should(it.Equal(1, strings.Count(string(raw), "kind:")))
	it.Then(t).Should(it.Equal(1, strings.Count(string(raw), `"@type": entity`)))
}

// FR-014/FR-015/spec FR-008, ast-contract.md/render-shape-contract.md:
// RenderNode(ParseNode(RenderNode(n, testIndex)), testIndex) is byte-equal
// to RenderNode(n, testIndex) for any Node produced by this package's own
// parser — a second round-trip of already-rendered output is stable. This
// fixture mixes an edge-role predicate (replaces, stays a flat bullet) and
// a link-role predicate (mentionedIn, grouped under "## MentionedIn") on
// one node — the mixed-shape case, not merely the previously-all-flat one.
func TestIdempotentRoundTrip(t *testing.T) {
	n := core.Node{
		ID:    "Transport Layer Security",
		Type:  "entity",
		Attrs: map[string][]core.Predicate{"category": {{Value: "independent"}, {Value: "abstract"}}},
		Texts: map[string]string{
			"definition": "A cryptographic protocol that establishes an authenticated channel.",
			"notes":      "Superseded by later revisions.",
		},
		Edges: []core.Link{
			{Predicate: "replaces", Target: "SSL Protocol"},
			{Predicate: "mentionedIn", Target: "rescorla-2026-tls13"},
		},
	}

	first, err := core.RenderNode(n, testIndex)
	it.Then(t).Should(it.Nil(err))

	parsed, err := core.ParseNode(strings.NewReader(string(first)))
	it.Then(t).Should(it.Nil(err))

	second, err := core.RenderNode(parsed, testIndex)
	it.Then(t).Should(it.Nil(err))

	it.Then(t).Should(it.Equal(string(first), string(second)))
}

// research.md D6/AST §10: a node originally written with a "## <Label>"
// grouped link block round-trips to a flat bulleted list — content and
// connectivity (the same set of Link values) survive identically, only the
// on-disk grouping layout does not.
// TestNormalizationCorrectsShapeTowardPredicateRole (research.md D8, spec
// FR-009): a node parsed from a document whose original shape — three
// bold-label grouped blocks — disagrees with its predicates' declared
// roles re-renders in the canonical schema-driven shape derived from
// index, not the shape the original document happened to use.
// mentionedIn (link-role in testIndex) stays grouped under its own
// heading; referencedBy/related (edge-role) flatten to bare bullets. A
// second sub-case covers the opposite direction: a link-role predicate
// originally written as a flat bullet is corrected to grouped shape.
// Content (Predicate/Target) survives identically either way (FR-010) —
// only the on-disk grouping layout changes.
func TestNormalizationCorrectsShapeTowardPredicateRole(t *testing.T) {
	patch, err := core.ParsePatch(strings.NewReader(boldLabelThreeBlocksPatch))
	it.Then(t).Should(it.Nil(err))

	node := patch.Nodes[0]
	rendered, err := core.RenderNode(node, testIndex)
	it.Then(t).Should(it.Nil(err))

	out := string(rendered)
	it.Then(t).
		Should(it.String(out).Contain("## MentionedIn")).
		Should(it.String(out).Contain("mentionedIn:: [[dmitry-2026-graph]]")).
		Should(it.String(out).Contain("referencedBy:: [[Core Thoughts Extension]]")).
		Should(it.String(out).Contain("related:: [[Article Extension]]")).
		ShouldNot(it.String(out).Contain("## ReferencedBy")).
		ShouldNot(it.String(out).Contain("## Related"))

	back, err := core.ParseNode(strings.NewReader(out))
	it.Then(t).Should(it.Nil(err))

	wantTargets := map[string]string{}
	for _, e := range node.Edges {
		wantTargets[e.Predicate] = e.Target
	}
	gotTargets := map[string]string{}
	for _, e := range back.Edges {
		gotTargets[e.Predicate] = e.Target
	}
	it.Then(t).Should(it.Equal(len(wantTargets), len(gotTargets)))
	for k, v := range wantTargets {
		it.Then(t).Should(it.Equal(v, gotTargets[k]))
	}

	// Opposite direction: a link-role predicate (mentions) originally
	// written as a flat bullet, alongside an edge-role predicate
	// (replaces, so the single-group omission does not itself suppress
	// the heading), is corrected to grouped shape on re-render.
	flatFixture := `---
"@id": FlatMentionsEntity
"@type": entity
---
# FlatMentionsEntity

Some text.

- replaces:: [[SSL Protocol]]
- mentions:: [[A]]
`
	flatNode, err := core.ParseNode(strings.NewReader(flatFixture))
	it.Then(t).Should(it.Nil(err))

	flatRendered, err := core.RenderNode(flatNode, testIndex)
	it.Then(t).Should(it.Nil(err))

	flatOut := string(flatRendered)
	it.Then(t).
		Should(it.String(flatOut).Contain("replaces:: [[SSL Protocol]]")).
		Should(it.String(flatOut).Contain("## Mentions")).
		Should(it.String(flatOut).Contain("mentions:: [[A]]"))
}
