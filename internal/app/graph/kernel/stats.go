//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package kernel

// StatsResult is arc stats' whole-graph report: the summary figures every
// run produces, plus a detail section computed only when verbosity asked
// for it (specs/032-arc-stats research.md D4).
//
// Two node populations feed it, deliberately (research.md D1). Nodes and
// ByType describe the CENSUS — every node file the graph holds, schema
// documents included — answering "how big is this graph?". Edges,
// BrokenLinks and every Detail figure but Schema describe the CONTENT
// population — the census minus Class/Property schema documents —
// answering "is this graph's connective tissue intact?". The asymmetry is
// part of the --json contract, not an artifact: it is what makes
// BrokenLinks agree with arc lint, which excludes _schema/ from its own
// basename universe.
type StatsResult struct {
	// Root is the graph root that was scanned.
	Root string `json:"root"`
	// Nodes is the census population size.
	Nodes int `json:"nodes"`
	// Edges counts structural link OCCURRENCES over the content
	// population, so a node linking twice to one target contributes two
	// (spec FR-003). Deliberately not the rule BrokenLinks uses.
	Edges int `json:"edges"`
	// ByType is the census grouped by each node's own declared "@type",
	// never by the folder holding it (spec FR-004a). Counts sum to Nodes.
	ByType []TypeCount `json:"byType"`
	// BrokenLinks counts each DISTINCT unresolved target once per node
	// referencing it, across structural Edges and inline HRefs alike
	// (spec FR-006) — the rule arc lint's linkResolves applies.
	BrokenLinks int `json:"brokenLinks"`
	// Ingestion is the yearly period breakdown, chronological, read from
	// the graph's own Timeline index rather than re-derived from source
	// nodes (spec FR-007, research.md D5).
	Ingestion []PeriodCount `json:"ingestion"`
	// Unreadable lists node files found but not parseable (spec FR-009).
	Unreadable []string `json:"unreadable"`
	// Foreign lists markdown under the graph root that claims no graph
	// identity at all — the host project's own README, ADRs, design notes
	// (spec 031 FR-032, research.md D8). Not a defect list: these files
	// are counted in no statistic.
	Foreign []string `json:"foreign"`
	// Detail is nil unless verbose output was requested. A nil pointer
	// with omitempty marshals to an ABSENT key, which is exactly spec
	// FR-020a's "absent, not null" distinction.
	Detail *StatsDetail `json:"detail,omitempty"`
}

// StatsDetail carries every verbose-only figure (spec FR-012..FR-017). It
// is computed only when asked for: verbosity gates what arc stats computes,
// not merely what it renders (spec FR-018).
type StatsDetail struct {
	// ByPredicate groups the content population's edges by predicate;
	// counts sum to StatsResult.Edges (spec FR-012).
	ByPredicate []PredicateCount `json:"byPredicate"`
	// IngestionMonthly is the monthly period breakdown; the months of a
	// year sum to that year's Ingestion figure (spec FR-013).
	IngestionMonthly []PeriodCount `json:"ingestionMonthly"`
	// Orphans counts content nodes with neither an outgoing nor an
	// incoming reference (spec FR-014).
	Orphans int `json:"orphans"`
	// Stubs counts content nodes carrying nothing beyond identity and
	// type — service.isStub's shape (spec FR-014).
	Stubs int `json:"stubs"`
	// UnresolvedTargets names each distinct unresolved target with the
	// number of nodes referencing it; Refs sum to
	// StatsResult.BrokenLinks (spec FR-014). It carries a count rather
	// than being a bare list precisely because BrokenLinks counts a
	// target once per referencing node.
	UnresolvedTargets []TargetCount `json:"unresolvedTargets"`
	// AvgOutDegree averages outgoing edges over the content population,
	// zero-degree nodes included in the denominator (spec FR-015).
	AvgOutDegree float64 `json:"avgOutDegree"`
	// MedianOutDegree is the median outgoing-edge count; an even
	// population takes the mean of the two central values (spec FR-015).
	MedianOutDegree float64 `json:"medianOutDegree"`
	// TopReferenced ranks the ten most-referenced nodes, refs descending
	// then id ascending (spec FR-015, FR-019).
	TopReferenced []RefRank `json:"topReferenced"`
	// Schema reports declared-vs-used vocabulary coverage (spec FR-016).
	Schema SchemaCoverage `json:"schema"`
	// Content reports content volume (spec FR-017).
	Content ContentVolume `json:"content"`
}

// TypeCount is one entry of the per-type node breakdown. Declared answers
// "does a Class schema node declare this type?" — distinct from
// SchemaCoverage.Classes, which counts the schema nodes themselves. It is
// meaningful only in verbose output.
type TypeCount struct {
	Type     string `json:"type"`
	Count    int    `json:"count"`
	Declared bool   `json:"declared,omitempty"`
}

// PredicateCount is one entry of the per-predicate edge breakdown.
type PredicateCount struct {
	Predicate string `json:"predicate"`
	Count     int    `json:"count"`
	Declared  bool   `json:"declared,omitempty"`
}

// PeriodCount is one ingestion period — a year ("2026") or a month
// ("2026-01") — with the number of entries its Timeline node holds.
type PeriodCount struct {
	Period string `json:"period"`
	Count  int    `json:"count"`
}

// TargetCount is one unresolved link target with the number of distinct
// nodes referencing it.
type TargetCount struct {
	Target string `json:"target"`
	Refs   int    `json:"refs"`
}

// RefRank is one node's incoming-reference count, for the hub ranking.
type RefRank struct {
	ID   string `json:"id"`
	Refs int    `json:"refs"`
}

// SchemaCoverage reports how much of the declared vocabulary the graph
// actually uses, and which predicates it uses without declaring.
type SchemaCoverage struct {
	Classes              int      `json:"classes"`
	ClassesUsed          int      `json:"classesUsed"`
	Properties           int      `json:"properties"`
	PropertiesUsed       int      `json:"propertiesUsed"`
	UndeclaredPredicates []string `json:"undeclaredPredicates"`
}

// ContentVolume reports the content population's bulk. InlineRefs is
// reported separately from StatsResult.Edges because an inline prose
// reference is not a structural edge (AST invariant 3).
type ContentVolume struct {
	InlineRefs            int `json:"inlineRefs"`
	AttributeValues       int `json:"attributeValues"`
	NodesWithoutPublished int `json:"nodesWithoutPublished"`
}
