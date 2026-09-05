//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package kernel_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/graph/kernel"
)

// sampleStatsResult is a report whose figures satisfy every documented
// invariant, so a test that breaks one is testing the invariant rather
// than the fixture.
func sampleStatsResult() kernel.StatsResult {
	return kernel.StatsResult{
		Root:  "/graph",
		Nodes: 6,
		Edges: 5,
		ByType: []kernel.TypeCount{
			{Type: "Entity", Count: 4},
			{Type: "Source", Count: 2},
		},
		BrokenLinks: 3,
		Ingestion:   []kernel.PeriodCount{{Period: "2025", Count: 2}, {Period: "2026", Count: 1}},
		Unreadable:  []string{},
		Foreign:     []string{},
		Detail: &kernel.StatsDetail{
			ByPredicate: []kernel.PredicateCount{
				{Predicate: "cites", Count: 3, Declared: true},
				{Predicate: "mentions", Count: 2, Declared: true},
			},
			IngestionMonthly: []kernel.PeriodCount{
				{Period: "2025-01", Count: 2},
				{Period: "2026-04", Count: 1},
			},
			UnresolvedTargets: []kernel.TargetCount{
				{Target: "Ghost", Refs: 2},
				{Target: "Missing", Refs: 1},
			},
		},
	}
}

// Contract C1 — the per-type breakdown accounts for every node.
func TestStatsTypeCountsAccountForEveryNode(t *testing.T) {
	r := sampleStatsResult()

	sum := 0
	for _, e := range r.ByType {
		sum += e.Count
	}
	it.Then(t).Should(it.Equal(r.Nodes, sum))
}

// Contract C2 — the per-predicate breakdown accounts for every edge.
func TestStatsPredicateCountsAccountForEveryEdge(t *testing.T) {
	r := sampleStatsResult()

	sum := 0
	for _, e := range r.Detail.ByPredicate {
		sum += e.Count
	}
	it.Then(t).Should(it.Equal(r.Edges, sum))
}

// Contract C3 — a year's months sum to that year.
func TestStatsMonthlyIngestionSumsToYearly(t *testing.T) {
	r := sampleStatsResult()

	for _, year := range r.Ingestion {
		months := 0
		for _, month := range r.Detail.IngestionMonthly {
			if strings.HasPrefix(month.Period, year.Period) {
				months += month.Count
			}
		}
		it.Then(t).Should(it.Equal(year.Count, months))
	}
}

// Contract C4 — the per-target refs sum to the broken-link total they
// explain. TargetCount carries a count rather than being a bare list
// precisely so this can hold.
func TestStatsUnresolvedTargetRefsSumToBrokenLinks(t *testing.T) {
	r := sampleStatsResult()

	sum := 0
	for _, e := range r.Detail.UnresolvedTargets {
		sum += e.Refs
	}
	it.Then(t).Should(it.Equal(r.BrokenLinks, sum))
}

// Contract C6 — a nil Detail marshals to an ABSENT key, never to null, so
// a consumer distinguishes "not requested" from "requested and empty" by
// key presence alone (spec FR-020a).
func TestStatsNilDetailMarshalsToAbsentKey(t *testing.T) {
	r := sampleStatsResult()
	r.Detail = nil

	raw, err := json.Marshal(r)
	it.Then(t).Must(it.Nil(err))

	var doc map[string]any
	it.Then(t).Must(it.Nil(json.Unmarshal(raw, &doc)))

	_, present := doc["detail"]
	it.Then(t).
		Should(it.True(!present)).
		Should(it.True(!strings.Contains(string(raw), "null")))
}

// The summary document carries exactly the v1 contract's keys — a renamed
// or newly-promoted key is a breaking change (Principle XIV).
func TestStatsSummaryCarriesDocumentedKeys(t *testing.T) {
	r := sampleStatsResult()
	r.Detail = nil

	raw, err := json.Marshal(r)
	it.Then(t).Must(it.Nil(err))

	var doc map[string]any
	it.Then(t).Must(it.Nil(json.Unmarshal(raw, &doc)))

	for _, key := range []string{"root", "nodes", "edges", "byType", "brokenLinks", "ingestion", "unreadable", "foreign"} {
		_, ok := doc[key]
		it.Then(t).Should(it.True(ok))
	}
	it.Then(t).Should(it.Equal(8, len(doc)))
}

// byType[].declared is a verbose-only figure and stays out of the summary
// document when false, exactly as detail does.
func TestStatsUndeclaredTypeOmitsDeclaredKey(t *testing.T) {
	raw, err := json.Marshal(kernel.TypeCount{Type: "Entity", Count: 1})
	it.Then(t).Must(it.Nil(err))

	it.Then(t).Should(it.Equal(`{"type":"Entity","count":1}`, string(raw)))
}
