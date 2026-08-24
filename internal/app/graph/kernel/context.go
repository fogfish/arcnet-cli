//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package kernel

import "github.com/fogfish/arcnet-cli/internal/core"

// ContextRetrieveResult is the domain value component.go's ContextRetrieve
// returns to cmd/arc/graph, mirroring SubgraphResult's shape
// (data-model.md).
type ContextRetrieveResult struct {
	// Root is the graph root that was searched.
	Root string `json:"root"`
	// Query is the query text, as given.
	Query string `json:"query"`
	// Limit is the resolved limit (post-default).
	Limit int `json:"limit"`
	// Patch is the ranked, truncated candidates as a synthesized
	// patch-exchange document, ready for core.RenderPatch.
	Patch core.Patch `json:"patch"`
	// ContentMatched is the count of nodes found by the content-match pass,
	// before ranking/truncation.
	ContentMatched int `json:"contentMatched"`
	// AttrMatched is the count of nodes found by the attribute-match pass,
	// before ranking/truncation.
	AttrMatched int `json:"attrMatched"`
	// NeighborReachable is the count of nodes found by neighbor expansion,
	// before capping.
	NeighborReachable int `json:"neighborReachable"`
	// NeighborIncluded is the count of neighbor-expansion nodes retained
	// after capPool.
	NeighborIncluded int `json:"neighborIncluded"`
	// NeighborTruncated is true when NeighborReachable > NeighborIncluded.
	NeighborTruncated bool `json:"neighborTruncated"`
	// Truncated is true when the combined, deduplicated candidate pool
	// exceeded Limit before truncation.
	Truncated bool `json:"truncated"`
}
