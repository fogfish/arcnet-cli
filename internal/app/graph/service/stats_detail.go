//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import (
	"sort"

	"github.com/fogfish/arcnet-cli/internal/app/graph/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// statsDetail derives every verbose-only figure from the corpus already in
// memory and the reference index already built — no second pass over disk,
// which is what keeps the whole command linear in graph size (SC-004).
func statsDetail(corpus statsCorpus, links linkIndex) *kernel.StatsDetail {
	degrees := outDegrees(corpus.content)
	return &kernel.StatsDetail{
		ByPredicate:       byPredicate(corpus.content, declaredIDs(corpus.census, propertyType)),
		IngestionMonthly:  periodCounts(corpus.content, monthlyLayout),
		Orphans:           countOrphans(corpus.content, links),
		Stubs:             countStubs(corpus.content),
		UnresolvedTargets: unresolvedTargets(links),
		AvgOutDegree:      meanDegree(degrees),
		MedianOutDegree:   medianDegree(degrees),
		TopReferenced:     topReferenced(links),
		Schema:            schemaCoverage(corpus),
		Content:           contentVolume(corpus.content),
	}
}

// byPredicate groups the content population's structural edges by
// predicate. Counts sum to the reported edge total because both count the
// same occurrences (spec FR-012, contract C2).
func byPredicate(content []core.Node, declared map[string]bool) []kernel.PredicateCount {
	counts := map[string]int{}
	for _, node := range content {
		for _, link := range node.Edges {
			counts[link.Predicate]++
		}
	}

	out := make([]kernel.PredicateCount, 0, len(counts))
	for name, count := range counts {
		out = append(out, kernel.PredicateCount{Predicate: name, Count: count, Declared: declared[name]})
	}

	sortByCountThenName(out,
		func(e kernel.PredicateCount) int { return e.Count },
		func(e kernel.PredicateCount) string { return e.Predicate },
	)
	return out
}

// countOrphans counts content nodes that neither reference anything nor are
// referenced by anything (spec FR-014). An unresolved outgoing link still
// counts as reaching outward — the node is broken, not disconnected.
func countOrphans(content []core.Node, links linkIndex) int {
	total := 0
	for _, node := range content {
		if !links.outgoing[node.ID] && links.incoming[node.ID] == 0 {
			total++
		}
	}
	return total
}

// countStubs counts content nodes carrying nothing beyond identity and
// type, reusing isStub — the same shape arc apply and arc subgraph --stubs
// already recognize, rather than a second definition free to drift from it.
func countStubs(content []core.Node) int {
	total := 0
	for _, node := range content {
		if isStub(node) {
			total++
		}
	}
	return total
}

// unresolvedTargets names each distinct unresolved target with the number
// of nodes referencing it, so the entries sum exactly to the broken-link
// total they explain (spec FR-014, contract C4).
func unresolvedTargets(links linkIndex) []kernel.TargetCount {
	out := make([]kernel.TargetCount, 0, len(links.unresolved))
	for target, refs := range links.unresolved {
		out = append(out, kernel.TargetCount{Target: target, Refs: refs})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Target < out[j].Target })
	return out
}

// outDegrees is every content node's outgoing structural edge count,
// zero-degree nodes included — they are part of the population the average
// describes, and dropping them would flatter the graph (spec FR-015).
func outDegrees(content []core.Node) []int {
	out := make([]int, 0, len(content))
	for _, node := range content {
		out = append(out, len(node.Edges))
	}
	return out
}

func meanDegree(degrees []int) float64 {
	if len(degrees) == 0 {
		return 0
	}
	total := 0
	for _, degree := range degrees {
		total += degree
	}
	return float64(total) / float64(len(degrees))
}

// medianDegree takes the mean of the two central values on an even
// population (research.md D9).
func medianDegree(degrees []int) float64 {
	if len(degrees) == 0 {
		return 0
	}
	sorted := append([]int(nil), degrees...)
	sort.Ints(sorted)

	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return float64(sorted[mid])
	}
	return float64(sorted[mid-1]+sorted[mid]) / 2
}

// topReferenced ranks the most-referenced nodes by incoming structural and
// inline references alike, ties broken by ascending id, truncated to ten
// (spec FR-015, FR-019). Only nodes something actually points at are
// ranked: a hub list padded with zero-reference nodes answers no question.
func topReferenced(links linkIndex) []kernel.RefRank {
	out := make([]kernel.RefRank, 0, len(links.incoming))
	for id, refs := range links.incoming {
		out = append(out, kernel.RefRank{ID: id, Refs: refs})
	}

	sortByCountThenName(out,
		func(e kernel.RefRank) int { return e.Refs },
		func(e kernel.RefRank) string { return e.ID },
	)

	if len(out) > topReferencedLimit {
		out = out[:topReferencedLimit]
	}
	return out
}

// schemaCoverage reports how much of the declared vocabulary the graph
// actually uses, and which predicates it uses without declaring (spec
// FR-016). It is the one detail figure derived from the census rather than
// the content population — the schema documents themselves are what it
// counts.
func schemaCoverage(corpus statsCorpus) kernel.SchemaCoverage {
	classes := declaredIDs(corpus.census, classType)
	properties := declaredIDs(corpus.census, propertyType)

	usedTypes := map[string]bool{}
	usedPredicates := map[string]bool{}
	for _, node := range corpus.content {
		usedTypes[node.Type] = true
		for _, name := range predicatesOf(node) {
			usedPredicates[name] = true
		}
	}

	return kernel.SchemaCoverage{
		Classes:              len(classes),
		ClassesUsed:          countUsed(classes, usedTypes),
		Properties:           len(properties),
		PropertiesUsed:       countUsed(properties, usedPredicates),
		UndeclaredPredicates: undeclared(usedPredicates, properties),
	}
}

// predicatesOf collects every predicate name node uses in an edge, an
// inline reference, or an attribute — the three ways spec FR-016 counts a
// Property as used. The prose slot keys core.ParseNode always synthesizes
// are excluded: they are a structural convention, not vocabulary a graph
// declares, exactly as apply's own distinctPredicates already treats them.
func predicatesOf(node core.Node) []string {
	out := make([]string, 0, len(node.Edges)+len(node.HRefs)+len(node.Attrs))
	for _, link := range allLinks(node) {
		if link.Predicate != "" {
			out = append(out, link.Predicate)
		}
	}
	for name := range node.Attrs {
		out = append(out, name)
	}
	return out
}

func countUsed(declared map[string]bool, used map[string]bool) int {
	total := 0
	for name := range declared {
		if used[name] {
			total++
		}
	}
	return total
}

func undeclared(used map[string]bool, declared map[string]bool) []string {
	out := []string{}
	for name := range used {
		if !declared[name] {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// contentVolume reports the content population's bulk, counting inline
// references apart from structural edges because an inline prose reference
// is not an edge (AST invariant 3, spec FR-017).
func contentVolume(content []core.Node) kernel.ContentVolume {
	var out kernel.ContentVolume
	for _, node := range content {
		out.InlineRefs += len(node.HRefs)
		for _, values := range node.Attrs {
			out.AttributeValues += len(values)
		}
		if node.Published.IsZero() {
			out.NodesWithoutPublished++
		}
	}
	return out
}
