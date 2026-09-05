//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import (
	"bytes"
	"context"
	"io"
	"sort"
	"time"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/graph/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// classType/propertyType/timelineType are the declared "@type" values that
// separate a schema document and the derived chronological index from
// ordinary content. Stats reads them from each node's own front matter,
// never from the folder holding the file (spec FR-004a, research.md D3).
const (
	classType    = "Class"
	propertyType = "Property"
	timelineType = "Timeline"
)

// yearlyLayout/monthlyLayout are CORE §9.4's period-code shapes, the same
// two core.TimelinePeriods emits.
const (
	yearlyLayout  = "2006"
	monthlyLayout = "2006-01"
)

// topReferencedLimit caps the hub ranking (spec FR-015).
const topReferencedLimit = 10

// statsCorpus is one walk's yield: the two node populations arc stats
// reports over (research.md D1), plus the files belonging to neither.
//
// census is every file that parsed as a node, schema documents included —
// it answers "how big is this graph?". content is the census minus schema
// documents — it answers "is this graph's connective tissue intact?", and
// is the population whose basenames resolve links, which is what makes
// BrokenLinks agree with arc lint (contract C5).
type statsCorpus struct {
	census     []core.Node
	content    []core.Node
	unreadable []string
	foreign    []string
}

// Stats walks the graph rooted at dir once and reports its shape and
// health: total nodes and edges, the per-type breakdown, the broken-link
// count, and per-year ingestion coverage. detail gates COMPUTATION, not
// rendering (spec FR-018) — a default run never derives the hub ranking or
// the degree distribution at all.
//
// It is strictly read-only, structurally rather than by convention (spec
// FR-023): it takes no port.VCS and never reaches for Store.Create or
// Store.Remove, so it holds no capability to modify anything.
func Stats(ctx context.Context, mounter fsys.Mounter, index core.Index, dir string, detail bool) (kernel.StatsResult, error) {
	store, err := mounter.Mount(dir)
	if err != nil {
		return kernel.StatsResult{}, err
	}

	if err := guardIsGraph(store, dir); err != nil {
		return kernel.StatsResult{}, err
	}

	corpus, err := readStatsCorpus(store, index)
	if err != nil {
		return kernel.StatsResult{}, err
	}
	return statsReport(dir, corpus, detail), nil
}

// statsReport derives every reported figure from the corpus already in
// memory. The detail section is built only when asked for, so a default
// run genuinely does no work to produce figures it will not report.
func statsReport(dir string, corpus statsCorpus, detail bool) kernel.StatsResult {
	links := newLinkIndex(corpus.content)
	result := kernel.StatsResult{
		Root:        dir,
		Nodes:       len(corpus.census),
		Edges:       countEdges(corpus.content),
		ByType:      statsByType(corpus.census, detail),
		BrokenLinks: links.broken,
		Ingestion:   periodCounts(corpus.content, yearlyLayout),
		Unreadable:  corpus.unreadable,
		Foreign:     corpus.foreign,
	}

	if detail {
		result.Detail = statsDetail(corpus, links)
	}
	return result
}

// readStatsCorpus performs the feature's single physical traversal,
// splitting what it finds into the two populations. A file that cannot be
// opened or parsed is recorded as unreadable and the walk continues (spec
// FR-009, SC-006); markdown claiming no graph identity at all is the host
// project's own content, recorded as foreign and counted in no statistic
// (research.md D8).
func readStatsCorpus(store fsys.Store, index core.Index) (statsCorpus, error) {
	paths, err := walkGraphFiles(store)
	if err != nil {
		return statsCorpus{}, err
	}

	out := statsCorpus{unreadable: []string{}, foreign: []string{}}
	for _, path := range paths {
		out.classify(store, index, path)
	}

	sort.Strings(out.unreadable)
	sort.Strings(out.foreign)
	return out, nil
}

// classify files one walked path into the population it belongs to, or
// into neither.
func (c *statsCorpus) classify(store fsys.Store, index core.Index, path string) {
	raw, err := readStatsFile(store, path)
	if err != nil {
		c.unreadable = append(c.unreadable, path)
		return
	}
	if !core.LooksLikeNode(raw) {
		c.foreign = append(c.foreign, path)
		return
	}

	node, err := core.ParseNode(bytes.NewReader(raw), index)
	if err != nil {
		c.unreadable = append(c.unreadable, path)
		return
	}

	c.census = append(c.census, node)
	if !isSchemaNode(node) {
		c.content = append(c.content, node)
	}
}

func readStatsFile(store fsys.Store, path string) ([]byte, error) {
	f, err := store.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// isSchemaNode reports whether node is a vocabulary document rather than
// content. The test is the declared type, so a schema document a user has
// filed elsewhere is still recognized as one.
func isSchemaNode(node core.Node) bool {
	return node.Type == classType || node.Type == propertyType
}

// allLinks flattens node's inline HRefs and structural Edges into one
// slice, the same order and union arc lint's own collectAllLinks uses for
// its linkResolves rule (research.md D6).
func allLinks(node core.Node) []core.Link {
	out := make([]core.Link, 0, len(node.HRefs)+len(node.Edges))
	out = append(out, node.HRefs...)
	return append(out, node.Edges...)
}

// linkIndex is the content population's reference bookkeeping, derived in
// one pass and shared by every figure that needs it: the broken-link total,
// the per-target breakdown behind it, the hub ranking, and the orphan test.
//
// broken counts each DISTINCT unresolved target once per referencing node,
// which is why unresolved carries a count rather than being a bare set: a
// target four nodes each reference contributes four (spec FR-006, FR-014).
type linkIndex struct {
	incoming   map[string]int
	unresolved map[string]int
	outgoing   map[string]bool
	broken     int
}

func newLinkIndex(content []core.Node) linkIndex {
	known := make(map[string]bool, len(content))
	for _, node := range content {
		known[node.ID] = true
	}

	idx := linkIndex{
		incoming:   map[string]int{},
		unresolved: map[string]int{},
		outgoing:   map[string]bool{},
	}
	for _, node := range content {
		idx.add(node, known)
	}
	return idx
}

// add folds one node's references into the index. An unresolved target
// already seen on this node is skipped, which is the distinct-per-node
// rule; a resolved one is counted every time it occurs, since in-degree
// ranks by how often a node is reached.
func (idx *linkIndex) add(node core.Node, known map[string]bool) {
	seen := map[string]bool{}
	for _, link := range allLinks(node) {
		if link.Target == "" {
			continue
		}
		idx.outgoing[node.ID] = true
		if known[link.Target] {
			idx.incoming[link.Target]++
			continue
		}
		if seen[link.Target] {
			continue
		}
		seen[link.Target] = true
		idx.unresolved[link.Target]++
		idx.broken++
	}
}

// countEdges totals structural link OCCURRENCES, so a node linking twice
// to one target contributes two (spec FR-003) — deliberately not the
// distinct-per-node rule linkIndex.broken applies.
func countEdges(content []core.Node) int {
	total := 0
	for _, node := range content {
		total += len(node.Edges)
	}
	return total
}

// statsByType groups the census by each node's own declared type. Declared
// is a verbose-only figure, so it stays false on a default run rather than
// leaking a detail value into the summary document (spec FR-018).
func statsByType(census []core.Node, detail bool) []kernel.TypeCount {
	declared := declaredIDs(census, classType)
	counts := map[string]int{}
	for _, node := range census {
		counts[node.Type]++
	}

	out := make([]kernel.TypeCount, 0, len(counts))
	for typ, count := range counts {
		out = append(out, kernel.TypeCount{Type: typ, Count: count, Declared: detail && declared[typ]})
	}

	sortByCountThenName(out,
		func(e kernel.TypeCount) int { return e.Count },
		func(e kernel.TypeCount) string { return e.Type },
	)
	return out
}

// declaredIDs collects the identities of every census node of the given
// schema type — a Class document's id names a type, a Property document's
// names a predicate.
func declaredIDs(census []core.Node, schemaType string) map[string]bool {
	out := map[string]bool{}
	for _, node := range census {
		if node.Type == schemaType {
			out[node.ID] = true
		}
	}
	return out
}

// periodCounts reports one entry per Timeline node whose own id is a period
// code of the requested granularity, counting the entries that node holds
// (spec FR-007, research.md D5). A period with no entry is still reported,
// so a recorded year never silently disappears (spec FR-008).
func periodCounts(content []core.Node, layout string) []kernel.PeriodCount {
	out := []kernel.PeriodCount{}
	for _, node := range content {
		if node.Type != timelineType || !isPeriodCode(node.ID, layout) {
			continue
		}
		out = append(out, kernel.PeriodCount{Period: node.ID, Count: len(node.Edges)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Period < out[j].Period })
	return out
}

// isPeriodCode reports whether id is exactly a period code of layout's
// granularity. The round-trip through Format rejects an id that parses
// only loosely, so "2024-3" is not mistaken for "2024-03".
func isPeriodCode(id, layout string) bool {
	at, err := time.Parse(layout, id)
	return err == nil && at.Format(layout) == id
}

// sortByCountThenName applies the ordering every grouped list in this
// report shares: count descending, then name ascending. It sorts here, in
// the service, rather than in a printer, because --json must be
// deterministic too — Go randomizes map iteration per run, so an unsorted
// map-derived slice passes locally and flakes in CI (spec FR-021, SC-005).
func sortByCountThenName[T any](xs []T, count func(T) int, name func(T) string) {
	sort.Slice(xs, func(i, j int) bool {
		if count(xs[i]) != count(xs[j]) {
			return count(xs[i]) > count(xs[j])
		}
		return name(xs[i]) < name(xs[j])
	})
}
