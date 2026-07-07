//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package kernel

// Match is one reported line, in one node's file, that matched arc grep's
// pattern — the row cmd/arc/graph renders as "<kind>  <id>  <line>  <text>".
type Match struct {
	// Type is the owning node's kind.
	Type string `json:"type"`
	// ID is the owning node's parsed identity.
	ID string `json:"id"`
	// Path is the node file path, relative to the graph root.
	Path string `json:"path"`
	// Line is the 1-based line number within Path.
	Line int `json:"line"`
	// Text is the full, untruncated, unstyled matched line — presentation
	// (highlighting/truncation) is applied only in cmd/arc/graph/grep.go,
	// never here.
	Text string `json:"text"`
	// Start is the byte offset within Text where the match begins.
	Start int `json:"start"`
	// End is the byte offset within Text where the match ends.
	End int `json:"end"`
}

// GrepResult is the domain value component.go's Grep returns to
// cmd/arc/graph, rendered by bios.Registry[GrepResult].
type GrepResult struct {
	// Root is the graph root that was searched.
	Root string `json:"root"`
	// Pattern is the regexp pattern searched for.
	Pattern string `json:"pattern"`
	// Matches holds every match found, across every node passing Filter
	// (empty when nothing matched).
	Matches []Match `json:"matches"`
	// Unreadable holds node files that could not be read or parsed and were
	// excluded from the scan.
	Unreadable []string `json:"unreadable"`
}
