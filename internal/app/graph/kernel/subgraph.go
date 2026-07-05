//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package kernel

import "github.com/fogfish/arcnet-cli/internal/core"

// SubgraphResult is the domain value component.go's Subgraph returns to
// cmd/arc/graph, rendered by bios.Registry[SubgraphResult].
type SubgraphResult struct {
	// Root is the graph root that was extracted from.
	Root string `json:"root"`
	// Seed is the seed node's basename (<basename> as given).
	Seed string `json:"seed"`
	// Depth is the <n> hops requested (post-default-resolution).
	Depth int `json:"depth"`
	// Patch is the seed + reachable nodes, plus the synthesized manifest
	// (research.md D2) — passed directly to core.RenderPatch by the
	// Human/--json renderers.
	Patch core.Patch `json:"patch"`
	// DirectReachable is the count of nodes discovered by the "direct"
	// (outgoing) BFS pass before capping (research.md D3/D5).
	DirectReachable int `json:"directReachable"`
	// DirectIncluded is the count of "direct" nodes actually retained
	// after capping (= min(DirectReachable, DirectCap)).
	DirectIncluded int `json:"directIncluded"`
	// DirectTruncated is true when DirectReachable > DirectIncluded.
	DirectTruncated bool `json:"directTruncated"`
	// BacklinkReachable is the count of nodes discovered by the
	// "backlink" (incoming) BFS pass before capping.
	BacklinkReachable int `json:"backlinkReachable"`
	// BacklinkIncluded is the count of "backlink" nodes actually retained
	// after capping.
	BacklinkIncluded int `json:"backlinkIncluded"`
	// BacklinkTruncated is true when BacklinkReachable > BacklinkIncluded.
	BacklinkTruncated bool `json:"backlinkTruncated"`
	// Stubs is the count of minimal placeholder node sections emitted for
	// extraction-boundary link targets when --stubs was requested
	// (BUG-001); 0 when --stubs was not passed.
	Stubs int `json:"stubs"`
}
