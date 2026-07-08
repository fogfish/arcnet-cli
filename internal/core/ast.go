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

// MergeOp is one of CORE §9.3's fixed, seven-value menu of merge
// operations a predicate's own schema document declares itself against.
type MergeOp string

const (
	MergeImmutable          MergeOp = "immutable"
	MergeUnion              MergeOp = "union"
	MergeFirstWriteWin      MergeOp = "firstWriteWin"
	MergeFillIfEmpty        MergeOp = "fillIfEmpty"
	MergeLastWriteWin       MergeOp = "lastWriteWin"
	MergeAppend             MergeOp = "append"
	MergeValidatedOverwrite MergeOp = "validatedOverwrite"
)

// Link is a single reference from one node to another (AST §6.3/§6.5), used
// for exactly one of two distinct purposes: HRefs or Edges — never both at
// once for the same occurrence (AST §3).
type Link struct {
	Predicate string `json:"predicate,omitempty"`
	Target    string `json:"target"`
	Alias     string `json:"alias,omitempty"`
}

// Predicate is one value contributed to an Attrs entry (AST §7): exactly
// one of Value/Target is set. Value holds a scalar as authored; Target
// holds a reference-valued predicate's target basename (informative only —
// never itself a source of a navigable edge, mirroring how Edges/HRefs
// already separate reference kinds). Alias is meaningful only alongside
// Target. This feature's parser only ever produces Value-set predicates;
// Target/Alias exist so a later schema-driven reference-attribute feature
// does not need another Node-shape change.
type Predicate struct {
	Value  any    `json:"value,omitempty"`
	Target string `json:"target,omitempty"`
	Alias  string `json:"alias,omitempty"`
}

// Node is the graph's domain-level unit (ARCNET-AST §4 "Node Object"). Its
// json tags (specs/007-arc-subgraph) expose this type through a --json
// contract (kernel.SubgraphResult.Patch.Nodes) — see
// contracts/subgraph-json-contract.md for this feature's breaking delta.
type Node struct {
	// ID is the basename, equal to the filename without ".md" (CORE §6, AST §4).
	ID string `json:"id"`
	// Type is mandatory, from "@type". Open vocabulary (AST §4 invariant 5):
	// this feature recognizes "source", "entity", "resource", "timeline",
	// plus whatever a graph's .arc/config.yml additionally registers.
	Type string `json:"type"`
	// Published is the source document's declared publication date,
	// propagated to every non-stub, non-schema node the patch creates
	// (spec 009 FR-001), and filled on a later merge that finds it
	// previously zero (spec 009 FR-010) — never overwritten once non-zero.
	// Zero value (IsZero()) means "not yet set" (a stub, a schema document,
	// or a node from before spec 009).
	Published time.Time `json:"published,omitempty"`
	// Attrs holds every front-matter key other than "@id"/"@type"/
	// "published" (AST §7); every present key's slice is non-empty.
	// Unrecognized keys are preserved verbatim (AST invariant 5, spec
	// FR-017).
	Attrs map[string][]Predicate `json:"attrs,omitempty"`
	// Texts holds every named prose field, bracket-stripped (research.md
	// D3/D3b): any [[Target]]/[[Target|alias]] markup originally embedded
	// in a value is removed and recorded in HRefs instead. Keys are
	// produced by textPredicateFor (research.md D4).
	Texts map[string]string `json:"texts,omitempty"`
	// HRefs are inline links originally embedded in Texts values, extracted
	// at parse time; never a source of navigable edges (AST invariant 3).
	HRefs []Link `json:"hrefs,omitempty"`
	// Edges are every outgoing structural link, in document order,
	// regardless of whether the source document wrote it as a flat bullet
	// or grouped under a heading/bold label (research.md D5) — grouping is
	// derived at render time, never stored (AST §3 invariant 4).
	Edges []Link `json:"edges,omitempty"`
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
