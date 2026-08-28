//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	configkernel "github.com/fogfish/arcnet-cli/internal/app/config/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/graph/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
	"github.com/fogfish/arcnet-cli/internal/pkg/grep"
)

// matchesAttrs reports whether node carries any Attrs Predicate.Value whose
// stringified form contains query, case-insensitively (research.md D4, spec
// FR-002b). Reference-valued Predicates (nil Value) are skipped, matching
// core.Filter's own attribute-matching convention.
func matchesAttrs(node core.Node, query string) bool {
	q := strings.ToLower(query)
	for _, preds := range node.Attrs {
		for _, p := range preds {
			if p.Value == nil {
				continue
			}
			if strings.Contains(strings.ToLower(fmt.Sprint(p.Value)), q) {
				return true
			}
		}
	}
	return false
}

// ContextRetrieve mounts dir, validates limit up front, then assembles the
// candidate pool from three passes — content match (grep.Search, escaped
// and case-insensitive), attribute match (matchesAttrs), and one-hop
// neighbor expansion (bfs, both directions, pooled across every direct-
// match seed and capped via cfgSubgraph, each direction scoped to
// filter.Traversal() exactly like Subgraph's own BFS passes —
// specs/027-triple-filter-model research.md D3/D5) — the upfront
// direct-match universe stays narrowed to the full, unsplit filter, while
// pooled neighbor-only candidates are narrowed to filter.Narrowing()
// instead (research.md D3), deduplicated by id, ranked (direct matches
// before neighbor-only, then degree descending, then id ascending), and
// truncated to limit (research.md D2-D8, spec.md FR-001 through FR-013).
func ContextRetrieve(ctx context.Context, mounter fsys.Mounter, filter core.Filter, query string, limit int, cfgGrep configkernel.GrepConfig, cfgSubgraph configkernel.SubgraphConfig, dir string) (kernel.ContextRetrieveResult, error) {
	store, err := mounter.Mount(dir)
	if err != nil {
		return kernel.ContextRetrieveResult{}, err
	}

	if err := guardIsGraph(store, dir); err != nil {
		return kernel.ContextRetrieveResult{}, err
	}

	if limit <= 0 {
		return kernel.ContextRetrieveResult{}, ErrInvalidLimit.With(errNoCause, strconv.Itoa(limit))
	}

	index, err := enumerateNodes(store)
	if err != nil {
		return kernel.ContextRetrieveResult{}, err
	}

	filterIncluded := map[string]bool{}
	for id, n := range index {
		if filter.Match(n) {
			filterIncluded[id] = true
		}
	}

	// Content-match pass (research.md D3): scan the same walkNodeFiles-
	// enumerated .md set Grep uses, narrowed to filter-passing nodes, with
	// query always escaped so it can never fail as an invalid pattern
	// (spec FR-003).
	paths, err := walkNodeFiles(store)
	if err != nil {
		return kernel.ContextRetrieveResult{}, err
	}

	pathIndex := map[string]core.Node{}
	pathIncluded := map[string]bool{}
	for _, p := range paths {
		node, ok, err := readGrepNode(store, p)
		if err != nil || !ok {
			continue
		}
		pathIndex[p] = node
		if filter.Match(node) {
			pathIncluded[p] = true
		}
	}

	contentMatch := map[string]bool{}
	if strings.TrimSpace(query) != "" {
		workers := cfgGrep.Workers
		if workers <= 0 {
			workers = defaultGrepWorkers
		}

		scanned, err := grep.Search(ctx, store, "(?i)"+regexp.QuoteMeta(query), grep.Options{
			Extension: ".md",
			Workers:   workers,
			Include:   func(p string) bool { return pathIncluded[p] },
		})
		if err != nil {
			return kernel.ContextRetrieveResult{}, err
		}
		for _, m := range scanned.Matches {
			contentMatch[pathIndex[m.Path].ID] = true
		}
	}

	// Attribute-match pass (research.md D4, spec FR-002b), narrowed to
	// filter-passing nodes.
	attrMatch := map[string]bool{}
	for id := range filterIncluded {
		if matchesAttrs(index[id], query) {
			attrMatch[id] = true
		}
	}

	directMatch := map[string]bool{}
	for id := range contentMatch {
		directMatch[id] = true
	}
	for id := range attrMatch {
		directMatch[id] = true
	}

	// Neighbor-expansion pass (research.md D5): one hop, both directions,
	// from every direct-match seed, pooled before capping once per
	// direction with the same safeguards subgraph_get applies (spec
	// FR-009). research.md D3: neighbor expansion is scoped by filter's
	// traversal-constraint statements only, mirroring Subgraph's own split.
	rev := buildReverseIndex(index)
	scope := filter.Traversal()

	directNeighbors := func(id string) []string {
		var out []string
		for _, e := range admittedEdges(index[id], scope) {
			out = append(out, e.Target)
		}
		return out
	}
	backlinkNeighbors := func(id string) []string {
		var out []string
		for _, e := range admittedBacklinks(id, rev, scope) {
			out = append(out, e.Source)
		}
		return out
	}

	directPool := map[string]bool{}
	backlinkPool := map[string]bool{}
	for id := range directMatch {
		for _, n := range bfs(index, directNeighbors, id, 1) {
			directPool[n] = true
		}
		for _, n := range bfs(index, backlinkNeighbors, id, 1) {
			backlinkPool[n] = true
		}
	}

	neighborReachable := map[string]bool{}
	for id := range directPool {
		neighborReachable[id] = true
	}
	for id := range backlinkPool {
		neighborReachable[id] = true
	}

	directKept, directTruncated := capPool(index, rev, mapKeys(directPool), cfgSubgraph.DirectCap)
	backlinkKept, backlinkTruncated := capPool(index, rev, mapKeys(backlinkPool), cfgSubgraph.BacklinkCap)

	neighborKept := map[string]bool{}
	for _, id := range directKept {
		neighborKept[id] = true
	}
	for _, id := range backlinkKept {
		neighborKept[id] = true
	}

	// Assemble, dedup, and filter the final candidate pool (spec FR-004,
	// FR-005) — mirrors Subgraph's own addCandidate posture: caps operate
	// on the unfiltered pool, filter narrows what actually survives into
	// the result.
	included := map[string]bool{}
	isDirect := map[string]bool{}
	var nodes []core.Node

	narrowing := filter.Narrowing()
	addCandidate := func(id string, direct bool) {
		if included[id] {
			return
		}
		n, ok := index[id]
		if !ok {
			return
		}
		// research.md D3: a direct match was already vetted by the full
		// filter (filterIncluded/pathIncluded, above); a neighbor-only
		// candidate is narrowed by filter.Narrowing() alone, so a bare
		// traversal-constraint filter does not re-exclude the very node it
		// scoped expansion to reach.
		if direct {
			if !filter.Match(n) {
				return
			}
		} else if !narrowing.Match(n) {
			return
		}
		included[id] = true
		isDirect[id] = direct
		nodes = append(nodes, n)
	}

	for id := range directMatch {
		addCandidate(id, true)
	}
	for id := range neighborKept {
		addCandidate(id, false)
	}

	sort.Slice(nodes, func(i, j int) bool {
		di, dj := isDirect[nodes[i].ID], isDirect[nodes[j].ID]
		if di != dj {
			return di
		}
		degI, degJ := degree(index, rev, nodes[i].ID), degree(index, rev, nodes[j].ID)
		if degI != degJ {
			return degI > degJ
		}
		return nodes[i].ID < nodes[j].ID
	})

	truncated := len(nodes) > limit
	if truncated {
		nodes = nodes[:limit]
	}

	now := time.Now().UTC()
	published := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	neighborTruncated := directTruncated || backlinkTruncated

	patch := core.Patch{
		Document:  "context:" + slugify(query) + "@" + now.Format(time.RFC3339),
		Published: published,
		Title:     "Context: " + query,
		Stats: map[string]any{
			"nodes":             len(nodes),
			"contentMatched":    len(contentMatch),
			"attrMatched":       len(attrMatch),
			"neighborReachable": len(neighborReachable),
			"neighborIncluded":  len(neighborKept),
			"neighborTruncated": neighborTruncated,
			"truncated":         truncated,
		},
		Nodes: nodes,
	}

	return kernel.ContextRetrieveResult{
		Root:              dir,
		Query:             query,
		Limit:             limit,
		Patch:             patch,
		ContentMatched:    len(contentMatch),
		AttrMatched:       len(attrMatch),
		NeighborReachable: len(neighborReachable),
		NeighborIncluded:  len(neighborKept),
		NeighborTruncated: neighborTruncated,
		Truncated:         truncated,
	}, nil
}

// mapKeys returns the keys of a string-set map — used to hand capPool's
// []string-shaped parameter the pooled id sets built above.
func mapKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for id := range m {
		out = append(out, id)
	}
	return out
}
