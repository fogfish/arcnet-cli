//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package kernel

// BacklinkEntry is one reported relation from another node onto the
// queried node — the row rendered as "| source | predicate |" by
// node_backlinks (specs/030-mcp-node-links-backlinks).
type BacklinkEntry struct {
	Source    string `json:"source"`
	Predicate string `json:"predicate,omitempty"`
}
