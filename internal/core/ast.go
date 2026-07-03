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
	Predicate string
	Target    string
	Alias     string
}

// LinkBlock is one predicate-grouped body block (AST §6.5). Title is the
// display heading (e.g. "Mentions"), derived and never independently
// re-derived by a consumer once parsed.
type LinkBlock struct {
	Title string
	Seq   []Link
}

// Node is the graph's domain-level unit (ARCNET-AST §4 "Node Object").
type Node struct {
	// ID is the basename, equal to the filename without ".md" (CORE §6, AST §4).
	ID string
	// Kind is mandatory.
	Kind Kind
	// Attrs holds front-matter scalars, excluding kind (AST §4); unrecognized
	// keys are preserved verbatim (AST invariant 5, spec FR-017).
	Attrs map[string]any
	// Text is the leading prose block, bracket-stripped (research.md D3/D3b):
	// any [[Target]]/[[Target|alias]] markup originally embedded in the prose
	// is removed and recorded in HRefs instead.
	Text string
	// Notes is the trailing prose block, rendered after Edges/Links;
	// bracket-stripped exactly like Text (D3b).
	Notes string
	// HRefs are inline links originally embedded in Text/Notes, extracted at
	// parse time; never a source of navigable edges (AST invariant 3).
	HRefs []Link
	// Edges are ungrouped structural edges, order-preserving.
	Edges []Link
	// Links are predicate-grouped structural edges.
	Links map[string]LinkBlock
}

// Patch is one CORE §12.2 document patch: a manifest plus every H1/H2 node
// section it carries, in document order.
type Patch struct {
	// Document is the source citekey this patch contributes (mandatory).
	Document string
	// Published drives timeline derivation (D8); mandatory.
	Published time.Time
	// Title is recommended.
	Title string
	// Stats is recommended; carried through, not independently validated
	// against actual counts.
	Stats map[string]any
	Nodes []Node
}
