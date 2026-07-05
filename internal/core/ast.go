//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package core is the graph's shared, use-case-independent core domain: the
// AST (ARCNET-AST §4-6), a goldmark-backed Markdown↔AST codec, the CORE §10
// merge algebra, CORE §9.4 timeline derivation, and the CORE §9/§10
// kind/merge-rule vocabulary. No dependency on any internal/app/<use-case>.
package core

import "time"

// Kind is a node's front-matter kind. Open vocabulary (AST §4 invariant 5):
// this feature recognizes "source", "entity", "resource", "timeline", plus
// whatever a graph's .arc/config.yml additionally registers.
type Kind string

// MergeOp is one of CORE §10's fixed menu of merge operations.
type MergeOp string

const (
	MergeNone               MergeOp = "none"
	MergeUnion              MergeOp = "union"
	MergeUnionFirstWriter   MergeOp = "union-first-writer"
	MergeAppend             MergeOp = "append"
	MergeValidatedOverwrite MergeOp = "validated-overwrite"
)

// Link is a single reference from one node to another (AST §6.3/§6.5), used
// for exactly one of three distinct purposes: HRefs, Edges, or a
// LinkBlock's Seq — never two at once for the same purpose (AST §3).
type Link struct {
	Predicate string `json:"predicate,omitempty"`
	Target    string `json:"target"`
	Alias     string `json:"alias,omitempty"`
}

// LinkBlock is one predicate-grouped body block (AST §6.5). Title is the
// display heading (e.g. "Mentions"), derived and never independently
// re-derived by a consumer once parsed.
type LinkBlock struct {
	Title string `json:"title"`
	Seq   []Link `json:"seq"`
}

// Node is the graph's domain-level unit (ARCNET-AST §4 "Node Object"). Its
// json tags (specs/007-arc-subgraph) are the first exposure of this type
// through a --json contract (kernel.SubgraphResult.Patch.Nodes) — no
// existing --json contract embeds Node, so this is purely additive.
type Node struct {
	// ID is the basename, equal to the filename without ".md" (CORE §6, AST §4).
	ID string `json:"id"`
	// Kind is mandatory.
	Kind Kind `json:"kind"`
	// Published is the source document's declared publication date,
	// propagated to every non-stub, non-schema node the patch creates
	// (spec 009 FR-001), and filled on a later merge that finds it
	// previously zero (spec 009 FR-010) — never overwritten once non-zero.
	// Zero value (IsZero()) means "not yet set" (a stub, a schema document,
	// or a node from before spec 009).
	Published time.Time `json:"published,omitempty"`
	// Attrs holds front-matter scalars, excluding kind (AST §4); unrecognized
	// keys are preserved verbatim (AST invariant 5, spec FR-017).
	Attrs map[string]any `json:"attrs,omitempty"`
	// Text is the leading prose block, bracket-stripped (research.md D3/D3b):
	// any [[Target]]/[[Target|alias]] markup originally embedded in the prose
	// is removed and recorded in HRefs instead.
	Text string `json:"text,omitempty"`
	// Notes is the trailing prose block, rendered after Edges/Links;
	// bracket-stripped exactly like Text (D3b).
	Notes string `json:"notes,omitempty"`
	// HRefs are inline links originally embedded in Text/Notes, extracted at
	// parse time; never a source of navigable edges (AST invariant 3).
	HRefs []Link `json:"hrefs,omitempty"`
	// Edges are ungrouped structural edges, order-preserving.
	Edges []Link `json:"edges,omitempty"`
	// Links are predicate-grouped structural edges.
	Links map[string]LinkBlock `json:"links,omitempty"`
}

// Patch is one CORE §12.2 document patch: a manifest plus every H1/H2 node
// section it carries, in document order. Its json tags (specs/007-arc-
// subgraph) are the first exposure of this type through a --json contract
// (kernel.SubgraphResult.Patch) — purely additive, mirroring Node's own.
type Patch struct {
	// Document is the source citekey this patch contributes (mandatory).
	Document string `json:"document"`
	// Published drives timeline derivation (D8); mandatory.
	Published time.Time `json:"published"`
	// Title is recommended.
	Title string `json:"title,omitempty"`
	// Stats is recommended; carried through, not independently validated
	// against actual counts.
	Stats map[string]any `json:"stats,omitempty"`
	Nodes []Node         `json:"nodes"`
}
