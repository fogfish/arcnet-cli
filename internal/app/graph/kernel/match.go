//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package kernel

// MatchEntry is one reported fact justifying a node's inclusion in
// node_match's result — the row rendered as "| id | property | value |".
type MatchEntry struct {
	ID       string `json:"id"`
	Property string `json:"property"`
	Value    string `json:"value"`
}

// MatchResult is the domain value component.go's Match returns to
// cmd/arc/graph, rendered by the MCP handler as a markdown table.
type MatchResult struct {
	// Root is the graph root that was searched.
	Root string `json:"root"`
	// Matches holds every reported fact, across every node satisfying the
	// filter (empty when nothing matched).
	Matches []MatchEntry `json:"matches"`
	// Unreadable holds node files that could not be read or parsed and
	// were excluded from the scan.
	Unreadable []string `json:"unreadable"`
}
