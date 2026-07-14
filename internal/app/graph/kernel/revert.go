//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package kernel

// RevertResult is the domain value component.go's Revert returns to
// cmd/arc/graph, rendered by bios.Registry[RevertResult].
type RevertResult struct {
	// Document is the retracted patch's source id.
	Document string `json:"document"`
	// Skipped is true when the source node no longer exists (Clarifications
	// Session 2026-07-12, FR-003); every other field is zero-valued then.
	Skipped bool `json:"skipped"`
	// Approach is "whole-commit" (D3/D4) or "per-node" (D5-D9) — FR-018.
	Approach string `json:"approach"`
	// Removed holds node counts by kind, deleted outright (FR-009).
	Removed map[string]int `json:"removed"`
	// Reconciled holds node counts by kind that had only the reverted
	// patch's own text content stripped, kept otherwise intact (FR-012).
	Reconciled map[string]int `json:"reconciled"`
	// LinksRemoved is the count of Edges dropped across every referrer
	// node touched by a removed node's backlink sweep (FR-010).
	LinksRemoved int `json:"linksRemoved"`
	// Nodes holds one NodeOutcome per node the revert touched, in
	// deterministic (path-sorted) order — populated always, surfaced only
	// under --verbose (FR-019, mirrors apply's per-predicate report
	// precedent, spec 012 FR-017).
	Nodes []NodeOutcome `json:"nodes"`
	// CommitHash is the short hash of the single resulting commit; empty
	// when Skipped.
	CommitHash string `json:"commit"`
}

// NodeOutcome records one node's fate within a revert (FR-019).
type NodeOutcome struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`             // "removed" | "reconciled" | "unchanged"
	Detail string `json:"detail,omitempty"` // e.g. "3 links removed", "1 paragraph stripped"
}
