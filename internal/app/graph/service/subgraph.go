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
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	configkernel "github.com/fogfish/arcnet-cli/internal/app/config/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/graph/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// nodeIndex maps a node's ID to its parsed value, built by enumerateNodes
// (research.md D7).
type nodeIndex map[string]core.Node

// reverseIndex maps a target ID to the IDs of every node carrying a
// structural connection (Edges/Links) to it — the backlink adjacency
// (research.md D4).
type reverseIndex map[string][]string

// enumerateNodes walks every node file in the graph (reusing walkNodeFiles,
// research.md D7), parsing each into the id -> core.Node index; a file that
// cannot be opened or parsed is silently excluded (mirrors Grep's own
// enumeration).
func enumerateNodes(store fsys.Store) (nodeIndex, error) {
	paths, err := walkNodeFiles(store)
	if err != nil {
		return nil, err
	}

	index := nodeIndex{}
	for _, p := range paths {
		node, ok, err := readGrepNode(store, p)
		if err != nil || !ok {
			continue
		}
		index[node.ID] = node
	}
	return index, nil
}

// nodeTargets returns every structural target n points at: its own Edges
// plus every Links block's Seq (research.md D3/D4) — HRefs are never
// navigable structural connections.
func nodeTargets(n core.Node) []string {
	var out []string
	for _, e := range n.Edges {
		out = append(out, e.Target)
	}
	for _, block := range n.Links {
		for _, l := range block.Seq {
			out = append(out, l.Target)
		}
	}
	return out
}

// buildReverseIndex builds the backlink adjacency for every node in index
// (research.md D4).
func buildReverseIndex(index nodeIndex) reverseIndex {
	rev := reverseIndex{}
	for id, n := range index {
		for _, target := range nodeTargets(n) {
			rev[target] = append(rev[target], id)
		}
	}
	return rev
}

// outDegree is n's own outgoing structural edge count: len(Edges) plus the
// sum of every Links block's Seq length (research.md D4).
func outDegree(n core.Node) int {
	d := len(n.Edges)
	for _, block := range n.Links {
		d += len(block.Seq)
	}
	return d
}

// degree is id's total structural connectivity across the whole graph —
// its own out-degree plus its in-degree (len(rev[id])) — used to rank
// cap-truncation candidates (research.md D4/D5). A degree computed for an
// id absent from index is 0 (out-degree) plus whatever in-degree the
// reverse index recorded.
func degree(index nodeIndex, rev reverseIndex, id string) int {
	return outDegree(index[id]) + len(rev[id])
}

// bfs runs one breadth-first traversal from seed over index, following
// neighbors(id) up to depth hops, returning every discovered node id
// (excluding seed itself), each included exactly once at its shortest hop
// distance (research.md D3; spec FR-004). A neighbor absent from index (a
// dangling link target) is silently skipped (spec FR-006). depth <= 0
// yields no reached nodes (spec FR-013).
func bfs(index nodeIndex, neighbors func(id string) []string, seed string, depth int) []string {
	visited := map[string]bool{seed: true}
	frontier := []string{seed}
	var reached []string

	for hop := 0; hop < depth && len(frontier) > 0; hop++ {
		var next []string
		for _, id := range frontier {
			for _, target := range neighbors(id) {
				if visited[target] {
					continue
				}
				if _, ok := index[target]; !ok {
					continue
				}
				visited[target] = true
				reached = append(reached, target)
				next = append(next, target)
			}
		}
		frontier = next
	}

	return reached
}

// capPool sorts ids by degree descending (ties broken by ID ascending for
// determinism) and retains only the top cap entries when cap > 0 and the
// pool exceeds it; cap <= 0 means uncapped (research.md D4/D5; spec
// FR-014/015). Neither cap ever causes a refusal — the caller only learns
// whether truncation occurred.
func capPool(index nodeIndex, rev reverseIndex, ids []string, cap int) (kept []string, truncated bool) {
	if cap <= 0 || len(ids) <= cap {
		return ids, false
	}

	sorted := append([]string(nil), ids...)
	sort.Slice(sorted, func(i, j int) bool {
		di, dj := degree(index, rev, sorted[i]), degree(index, rev, sorted[j])
		if di != dj {
			return di > dj
		}
		return sorted[i] < sorted[j]
	})
	return sorted[:cap], true
}

// boundaryTargets returns every structural link target referenced by any
// node in nodes that is present in index (a real, parsed node — spec
// FR-006 already excludes targets absent from the whole graph) but not
// itself in included, deduplicated and sorted for deterministic stub
// ordering (BUG-001, spec FR-017). It is computed once, over the already-
// finalized included set, and never over a stub's own (always-empty)
// targets — so it cannot recurse.
func boundaryTargets(index nodeIndex, included map[string]bool, nodes []core.Node) []string {
	seen := map[string]bool{}
	var out []string
	for _, n := range nodes {
		for _, target := range nodeTargets(n) {
			if included[target] || seen[target] {
				continue
			}
			if _, ok := index[target]; !ok {
				continue
			}
			seen[target] = true
			out = append(out, target)
		}
	}
	sort.Strings(out)
	return out
}

var slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// slugify lowercases s and collapses every run of non-alphanumeric
// characters into a single hyphen, trimmed at both ends — used to derive
// the synthesized patch document id (research.md D2), e.g. "Transport
// Layer Security" -> "transport-layer-security".
func slugify(s string) string {
	slug := slugNonAlnum.ReplaceAllString(strings.ToLower(s), "-")
	return strings.Trim(slug, "-")
}

// Subgraph mounts dir, enumerates and indexes every node, resolves the
// seed by basename, runs two independent BFS passes (direct/backlink)
// bounded by depth, applies cfg's caps, restricts the surviving non-seed
// candidates to filter (the seed is always included, regardless of
// filter), and synthesizes a core.Patch document ready for
// core.RenderPatch or re-ingestion via arc apply (research.md D2-D5,
// spec.md FR-001 through FR-016). When stubs is true (spec FR-017,
// BUG-001), every structural link target referenced by an included node
// but excluded from the extraction boundary also gets a minimal node
// section (kind + id only, no other attributes, empty body, never itself
// expanded) — so every link in the output resolves to a real node section
// even when the result is applied into a graph that does not already
// contain the excluded targets.
func Subgraph(ctx context.Context, mounter fsys.Mounter, filter core.Filter, basename string, depth int, cfg configkernel.SubgraphConfig, dir string, stubs bool) (kernel.SubgraphResult, error) {
	store, err := mounter.Mount(dir)
	if err != nil {
		return kernel.SubgraphResult{}, err
	}

	if err := guardIsGraph(store, dir); err != nil {
		return kernel.SubgraphResult{}, err
	}

	index, err := enumerateNodes(store)
	if err != nil {
		return kernel.SubgraphResult{}, err
	}

	seed, ok := index[basename]
	if !ok {
		return kernel.SubgraphResult{}, ErrSeedNotFound.With(errNoCause, basename)
	}

	rev := buildReverseIndex(index)

	directReached := bfs(index, func(id string) []string { return nodeTargets(index[id]) }, seed.ID, depth)
	backlinkReached := bfs(index, func(id string) []string { return rev[id] }, seed.ID, depth)

	directKept, directTruncated := capPool(index, rev, directReached, cfg.DirectCap)
	backlinkKept, backlinkTruncated := capPool(index, rev, backlinkReached, cfg.BacklinkCap)

	included := map[string]bool{seed.ID: true}
	nodes := []core.Node{seed}

	addCandidate := func(id string) {
		if included[id] {
			return
		}
		n := index[id]
		if !filter.Match(n) {
			return
		}
		included[id] = true
		nodes = append(nodes, n)
	}
	for _, id := range directKept {
		addCandidate(id)
	}
	for _, id := range backlinkKept {
		addCandidate(id)
	}

	var stubCount int
	if stubs {
		for _, id := range boundaryTargets(index, included, nodes) {
			target := index[id]
			nodes = append(nodes, core.Node{ID: target.ID, Kind: target.Kind})
			included[id] = true
			stubCount++
		}
	}

	now := time.Now().UTC()
	published := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	patch := core.Patch{
		Document:  "subgraph:" + slugify(seed.ID) + "@" + now.Format(time.RFC3339),
		Published: published,
		Title:     "Subgraph: " + seed.ID,
		Stats: map[string]any{
			"nodes":             len(nodes),
			"directReachable":   len(directReached),
			"directIncluded":    len(directKept),
			"directTruncated":   directTruncated,
			"backlinkReachable": len(backlinkReached),
			"backlinkIncluded":  len(backlinkKept),
			"backlinkTruncated": backlinkTruncated,
			"stubs":             stubCount,
		},
		Nodes: nodes,
	}

	return kernel.SubgraphResult{
		Root:              dir,
		Seed:              basename,
		Depth:             depth,
		Patch:             patch,
		DirectReachable:   len(directReached),
		DirectIncluded:    len(directKept),
		DirectTruncated:   directTruncated,
		BacklinkReachable: len(backlinkReached),
		BacklinkIncluded:  len(backlinkKept),
		BacklinkTruncated: backlinkTruncated,
		Stubs:             stubCount,
	}, nil
}
