//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import (
	"fmt"
	"sort"

	"github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// sortedLinkKeys returns links' keys in a deterministic order, for stable
// violation ordering.
func sortedLinkKeys(links map[string]core.LinkBlock) []string {
	keys := make([]string, 0, len(links))
	for k := range links {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// collectAllLinks flattens a node's HRefs, Edges, and every Links[*].Seq
// entry, in a deterministic order.
func collectAllLinks(node core.Node) []core.Link {
	var out []core.Link
	out = append(out, node.HRefs...)
	out = append(out, node.Edges...)
	for _, key := range sortedLinkKeys(node.Links) {
		out = append(out, node.Links[key].Seq...)
	}
	return out
}

// checkLinksResolve reports a RuleLinkResolves violation for every distinct
// unresolved link target (research.md D5), located via the raw-text
// locator.
func checkLinksResolve(node core.Node, path string, raw []byte, basenames map[string][]string) []kernel.Violation {
	var out []kernel.Violation
	seen := map[string]bool{}
	for _, l := range collectAllLinks(node) {
		if l.Target == "" || seen[l.Target] {
			continue
		}
		if _, ok := basenames[l.Target]; ok {
			continue
		}
		seen[l.Target] = true
		out = append(out, kernel.Violation{
			Rule:    kernel.RuleLinkResolves,
			Path:    path,
			Line:    locateLinkTarget(raw, l.Target),
			Message: fmt.Sprintf("target %q does not exist", l.Target),
		})
	}
	return out
}

// checkDerivedProvenance reports a RuleDerivedProvenance violation when a
// non-source node has no resolved link to any source-kind node
// (research.md D8) — no single line number applies (spec FR-015).
// timeline is also exempt (research.md D8 Bugfix, BUG-001): it is the
// tool's own chronological index over many documents, never content
// distilled from one document, the same way source itself is exempt.
func checkDerivedProvenance(node core.Node, path string, kindIndex map[string]core.Kind) []kernel.Violation {
	if node.Kind == "source" || node.Kind == "timeline" {
		return nil
	}

	for _, l := range collectAllLinks(node) {
		if l.Target == "" {
			continue
		}
		if kind, ok := kindIndex[l.Target]; ok && kind == "source" {
			return nil
		}
	}

	return []kernel.Violation{{
		Rule:    kernel.RuleDerivedProvenance,
		Path:    path,
		Line:    0,
		Message: "does not link to any source node",
	}}
}
