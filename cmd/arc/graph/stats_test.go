//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package graph

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/graph/kernel"
	"github.com/fogfish/arcnet-cli/internal/bios"
)

// statsRaw runs `arc stats` in dir and returns its raw stdout.
func statsRaw(t *testing.T, dir string, jsonMode, verbose bool) (string, error) {
	t.Helper()
	chdir(t, dir)

	bios.JSON = jsonMode
	bios.Verbose = verbose
	t.Cleanup(func() { bios.JSON, bios.Verbose = false, false })

	return sut(NewStatsCmd(), nil)
}

// statsOf runs `arc stats --json` in dir and decodes the report.
func statsOf(t *testing.T, dir string, verbose bool) kernel.StatsResult {
	t.Helper()
	out, err := statsRaw(t, dir, true, verbose)
	it.Then(t).ShouldNot(it.Error(out, err))

	var result kernel.StatsResult
	it.Then(t).Must(it.Nil(json.Unmarshal([]byte(out), &result)))
	return result
}

func typeCount(r kernel.StatsResult, typ string) int {
	for _, e := range r.ByType {
		if e.Type == typ {
			return e.Count
		}
	}
	return -1
}

func periodCount(periods []kernel.PeriodCount, period string) int {
	for _, e := range periods {
		if e.Period == period {
			return e.Count
		}
	}
	return -1
}

// US1 scenario 1.1 — every summary figure equals known/'s hand-counted
// value (stats_fixture_test.go composition table).
func TestStatsReportsSummary(t *testing.T) {
	r := statsOf(t, fixtureKnown(t), false)

	it.Then(t).
		Should(it.Equal(17, r.Nodes)).
		Should(it.Equal(9, r.Edges)).
		Should(it.Equal(0, r.BrokenLinks)).
		Should(it.Equal(5, len(r.ByType))).
		Should(it.Equal(4, typeCount(r, "Class"))).
		Should(it.Equal(4, typeCount(r, "Property"))).
		Should(it.Equal(4, typeCount(r, "Timeline"))).
		Should(it.Equal(3, typeCount(r, "Entity"))).
		Should(it.Equal(2, typeCount(r, "Source"))).
		Should(it.Equal(2, len(r.Ingestion))).
		Should(it.Equal(1, periodCount(r.Ingestion, "2024"))).
		Should(it.Equal(1, periodCount(r.Ingestion, "2025"))).
		Should(it.Equal(0, len(r.Unreadable))).
		Should(it.Equal(0, len(r.Foreign)))
}

// US1 scenario 1.1 — the per-type breakdown is ordered count descending
// then name ascending, and ingestion chronologically (spec FR-005, FR-008).
func TestStatsOrdersSummaryDeterministically(t *testing.T) {
	r := statsOf(t, fixtureKnown(t), false)

	names := make([]string, 0, len(r.ByType))
	for _, e := range r.ByType {
		names = append(names, e.Type)
	}
	periods := make([]string, 0, len(r.Ingestion))
	for _, e := range r.Ingestion {
		periods = append(periods, e.Period)
	}

	it.Then(t).
		Should(it.Seq(names).Equal("Class", "Property", "Timeline", "Entity", "Source")).
		Should(it.Seq(periods).Equal("2024", "2025"))
}

// US1 scenario 1.2 — contract C1.
func TestStatsTypeCountsSumToTotal(t *testing.T) {
	r := statsOf(t, fixtureKnown(t), false)

	sum := 0
	for _, e := range r.ByType {
		sum += e.Count
	}
	it.Then(t).Should(it.Equal(r.Nodes, sum))
}

// US1 scenario 1.4 — contract C8: an initialized but empty graph reports
// zeros, emits every key, and exits 0 (spec FR-010).
func TestStatsEmptyGraph(t *testing.T) {
	out, err := statsRaw(t, fixtureEmpty(t), true, false)
	it.Then(t).ShouldNot(it.Error(out, err))

	var r kernel.StatsResult
	it.Then(t).Must(it.Nil(json.Unmarshal([]byte(out), &r)))
	it.Then(t).
		Should(it.Equal(0, r.Nodes)).
		Should(it.Equal(0, r.Edges)).
		Should(it.Equal(0, r.BrokenLinks)).
		Should(it.Equal(0, len(r.ByType))).
		Should(it.Equal(0, len(r.Ingestion)))

	for _, key := range []string{`"byType": []`, `"ingestion": []`, `"unreadable": []`, `"foreign": []`} {
		it.Then(t).Should(it.True(strings.Contains(out, key)))
	}
}

// US1 scenario 1.5 — spec FR-011: refuse a directory that is not a graph,
// with the shared ErrNotAGraph message, and exit non-zero.
func TestStatsRefusesNonGraph(t *testing.T) {
	out, err := statsRaw(t, fixtureNotGraph(t), false, false)

	it.Then(t).Must(it.True(err != nil))
	it.Then(t).
		Should(it.True(strings.Contains(err.Error(), "is not an initialized graph"))).
		Should(it.Equal("", out))
}

// spec FR-024 — broken links are a reported figure, not a failure: the
// command still exits 0.
func TestStatsExitsZeroDespiteBrokenLinks(t *testing.T) {
	r := statsOf(t, fixtureBroken(t), false)
	it.Then(t).Should(it.Equal(4, r.BrokenLinks))
}

// spec FR-006a — the human report must state both counting rules, since
// they deliberately differ.
func TestStatsHumanReportStatesCountingRules(t *testing.T) {
	out, err := statsRaw(t, fixtureKnown(t), false, false)
	it.Then(t).ShouldNot(it.Error(out, err))

	it.Then(t).
		Should(it.True(strings.Contains(out, "occurrence"))).
		Should(it.True(strings.Contains(out, "distinct unresolved target"))).
		Should(it.True(strings.Contains(out, "17"))).
		Should(it.True(strings.Contains(out, "Timeline")))
}

// US2 scenario 2.1 — contract C2: per-predicate counts sum to Edges.
func TestStatsVerboseByPredicate(t *testing.T) {
	r := statsOf(t, fixtureKnown(t), true)
	it.Then(t).Must(it.True(r.Detail != nil))

	sum := 0
	names := make([]string, 0, len(r.Detail.ByPredicate))
	counts := make([]int, 0, len(r.Detail.ByPredicate))
	for _, e := range r.Detail.ByPredicate {
		sum += e.Count
		names = append(names, e.Predicate)
		counts = append(counts, e.Count)
	}

	it.Then(t).
		Should(it.Seq(names).Equal("cites", "mentions")).
		Should(it.Seq(counts).Equal(7, 2)).
		Should(it.Equal(r.Edges, sum))
}

// US2 scenario 2.2 — contract C3: a year's months sum to that year.
func TestStatsVerboseMonthlyIngestion(t *testing.T) {
	r := statsOf(t, fixtureKnown(t), true)
	it.Then(t).Must(it.True(r.Detail != nil))

	it.Then(t).
		Should(it.Equal(2, len(r.Detail.IngestionMonthly))).
		Should(it.Equal(1, periodCount(r.Detail.IngestionMonthly, "2024-03"))).
		Should(it.Equal(1, periodCount(r.Detail.IngestionMonthly, "2025-06")))

	for _, year := range r.Ingestion {
		months := 0
		for _, m := range r.Detail.IngestionMonthly {
			if strings.HasPrefix(m.Period, year.Period) {
				months += m.Count
			}
		}
		it.Then(t).Should(it.Equal(year.Count, months))
	}
}

// US2 scenario 2.3 — Entity Three is the graph's only node with neither an
// incoming nor an outgoing reference, and its only stub.
func TestStatsVerboseOrphans(t *testing.T) {
	r := statsOf(t, fixtureKnown(t), true)
	it.Then(t).Must(it.True(r.Detail != nil))

	it.Then(t).
		Should(it.Equal(1, r.Detail.Orphans)).
		Should(it.Equal(1, r.Detail.Stubs))
}

// US2 scenario 2.4 — contract C4: per-target refs sum to BrokenLinks.
func TestStatsVerboseUnresolvedTargets(t *testing.T) {
	r := statsOf(t, fixtureBroken(t), true)
	it.Then(t).Must(it.True(r.Detail != nil))

	targets := make([]string, 0, len(r.Detail.UnresolvedTargets))
	refs := make([]int, 0, len(r.Detail.UnresolvedTargets))
	sum := 0
	for _, e := range r.Detail.UnresolvedTargets {
		targets = append(targets, e.Target)
		refs = append(refs, e.Refs)
		sum += e.Refs
	}

	it.Then(t).
		Should(it.Seq(targets).Equal("Ghost", "Missing One")).
		Should(it.Seq(refs).Equal(2, 2)).
		Should(it.Equal(r.BrokenLinks, sum))
}

// US2 scenario 2.5 — degree includes zero-degree nodes in the denominator,
// and the hub ranking breaks ties by ascending id (spec FR-015, FR-019).
func TestStatsVerboseDegreeAndHubs(t *testing.T) {
	r := statsOf(t, fixtureKnown(t), true)
	it.Then(t).Must(it.True(r.Detail != nil))

	ids := make([]string, 0, len(r.Detail.TopReferenced))
	refs := make([]int, 0, len(r.Detail.TopReferenced))
	for _, e := range r.Detail.TopReferenced {
		ids = append(ids, e.ID)
		refs = append(refs, e.Refs)
	}

	it.Then(t).
		Should(it.Equal(1.0, r.Detail.AvgOutDegree)).
		Should(it.Equal(1.0, r.Detail.MedianOutDegree)).
		Should(it.Seq(ids).Equal("Entity One", "Entity Two", "alpha-2024", "beta-2025")).
		Should(it.Seq(refs).Equal(5, 2, 2, 2))
}

// US2 scenario 2.6 — spec FR-016.
func TestStatsVerboseSchemaCoverage(t *testing.T) {
	r := statsOf(t, fixtureKnown(t), true)
	it.Then(t).Must(it.True(r.Detail != nil))

	it.Then(t).
		Should(it.Equal(4, r.Detail.Schema.Classes)).
		Should(it.Equal(3, r.Detail.Schema.ClassesUsed)).
		Should(it.Equal(4, r.Detail.Schema.Properties)).
		Should(it.Equal(4, r.Detail.Schema.PropertiesUsed)).
		Should(it.Seq(r.Detail.Schema.UndeclaredPredicates).Equal("status"))
}

// US2 scenario 2.6 — TypeCount.Declared distinguishes a type a Class node
// declares from one only observed on nodes.
func TestStatsVerboseMarksDeclaredTypes(t *testing.T) {
	r := statsOf(t, fixtureKnown(t), true)

	declared := map[string]bool{}
	for _, e := range r.ByType {
		declared[e.Type] = e.Declared
	}

	it.Then(t).
		Should(it.True(declared["Source"])).
		Should(it.True(declared["Entity"])).
		Should(it.True(declared["Timeline"])).
		Should(it.True(!declared["Class"])).
		Should(it.True(!declared["Property"]))
}

// US2 scenario 2.7 — inline references are reported separately from
// structural edges, which they are not (spec FR-017).
func TestStatsVerboseContentVolume(t *testing.T) {
	r := statsOf(t, fixtureKnown(t), true)
	it.Then(t).Must(it.True(r.Detail != nil))

	it.Then(t).
		Should(it.Equal(2, r.Detail.Content.InlineRefs)).
		Should(it.Equal(8, r.Detail.Content.AttributeValues)).
		Should(it.Equal(7, r.Detail.Content.NodesWithoutPublished)).
		Should(it.Equal(9, r.Edges))
}

// US2 scenario 2.8 — contract C6: no detail figure appears by default, in
// any form (spec FR-018, FR-020a).
func TestStatsDefaultOmitsDetail(t *testing.T) {
	out, err := statsRaw(t, fixtureKnown(t), true, false)
	it.Then(t).ShouldNot(it.Error(out, err))

	var r kernel.StatsResult
	it.Then(t).Must(it.Nil(json.Unmarshal([]byte(out), &r)))

	it.Then(t).
		Should(it.True(r.Detail == nil)).
		Should(it.True(!strings.Contains(out, `"detail"`))).
		Should(it.True(!strings.Contains(out, `"declared"`)))
}

// US3 scenario 3.1 — the summary document matches the documented schema,
// and nothing but the document reaches stdout (spec FR-020).
func TestStatsJSONSummary(t *testing.T) {
	out, err := statsRaw(t, fixtureKnown(t), true, false)
	it.Then(t).ShouldNot(it.Error(out, err))

	var doc map[string]any
	it.Then(t).Must(it.Nil(json.Unmarshal([]byte(out), &doc)))

	keys := []string{"root", "nodes", "edges", "byType", "brokenLinks", "ingestion", "unreadable", "foreign"}
	for _, key := range keys {
		_, ok := doc[key]
		it.Then(t).Should(it.True(ok))
	}
	_, hasDetail := doc["detail"]
	it.Then(t).
		Should(it.Equal(len(keys), len(doc))).
		Should(it.True(!hasDetail))
}

// US3 scenario 3.2 — verbose adds a populated detail section carrying every
// documented key.
func TestStatsJSONVerbose(t *testing.T) {
	out, err := statsRaw(t, fixtureKnown(t), true, true)
	it.Then(t).ShouldNot(it.Error(out, err))

	var doc struct {
		Detail map[string]any `json:"detail"`
	}
	it.Then(t).Must(it.Nil(json.Unmarshal([]byte(out), &doc)))

	keys := []string{
		"byPredicate", "ingestionMonthly", "orphans", "stubs", "unresolvedTargets",
		"avgOutDegree", "medianOutDegree", "topReferenced", "schema", "content",
	}
	for _, key := range keys {
		_, ok := doc.Detail[key]
		it.Then(t).Should(it.True(ok))
	}
	it.Then(t).Should(it.Equal(len(keys), len(doc.Detail)))
}

// US3 scenario 3.3 — contract C7: two runs over an unchanged graph emit
// byte-identical output, which only service-level sorting can guarantee
// (Go randomizes map iteration per run).
func TestStatsJSONDeterministic(t *testing.T) {
	dir := fixtureKnown(t)

	first, err := statsRaw(t, dir, true, true)
	it.Then(t).ShouldNot(it.Error(first, err))
	second, err := statsRaw(t, dir, true, true)
	it.Then(t).ShouldNot(it.Error(second, err))

	it.Then(t).Should(it.Equal(first, second))
}

// Contract C9 / spec FR-004a — moved/ holds known/'s exact node set in
// folders arc never creates, so every figure must be identical. Any
// path-derived logic fails here.
func TestStatsIgnoresFolderLayout(t *testing.T) {
	known := statsOf(t, fixtureKnown(t), true)
	moved := statsOf(t, fixtureMoved(t), true)

	known.Root, moved.Root = "", ""
	a, err := json.Marshal(known)
	it.Then(t).Must(it.Nil(err))
	b, err := json.Marshal(moved)
	it.Then(t).Must(it.Nil(err))

	it.Then(t).Should(it.Equal(string(a), string(b)))
}

// spec FR-009, research.md D8 — a host project's README is expected, not
// graph damage, so it is reported apart from a file that failed to parse.
func TestStatsSeparatesUnreadableFromForeign(t *testing.T) {
	r := statsOf(t, fixtureMessy(t), false)

	it.Then(t).
		Should(it.Equal(4, r.Nodes)).
		Should(it.Seq(r.Unreadable).Equal("Entity/Broken.md")).
		Should(it.Seq(r.Foreign).Equal("README.md", "docs/design.md"))
}

// spec FR-023, SC-007 — the graph is byte-for-byte unchanged after a run.
func TestStatsDoesNotModifyGraph(t *testing.T) {
	dir := fixtureKnown(t)
	before := snapshotContent(t, dir)

	_ = statsOf(t, dir, true)

	it.Then(t).Should(it.Seq(flatten(snapshotContent(t, dir))).Equal(flatten(before)...))
}

// SC-004 — 10,000 nodes in under 5 seconds.
func TestStatsPerformance(t *testing.T) {
	dir := statsGraphRoot(t)
	writeStatsFiles(t, dir, knownSchema())
	seedLargeGraph(t, dir, 10000)

	start := time.Now()
	r := statsOf(t, dir, true)
	elapsed := time.Since(start)

	it.Then(t).
		Should(it.Equal(10008, r.Nodes)).
		Should(it.Less(elapsed, 5*time.Second))
}

// seedLargeGraph writes n Entity nodes, each citing its predecessor, in a
// fan of subdirectories so the walk is not one flat listing.
func seedLargeGraph(t *testing.T, dir string, n int) {
	t.Helper()
	files := make(map[string]string, n)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("node-%05d", i)
		body := fmt.Sprintf("---\n\"@id\": \"%s\"\n\"@type\": Entity\n---\n# %s\n\nGenerated node.\n", id, id)
		if i > 0 {
			body += fmt.Sprintf("- cites:: [[node-%05d]]\n", i-1)
		}
		files[fmt.Sprintf("Entity/%02d/%s.md", i%50, id)] = body
	}
	writeStatsFiles(t, dir, files)
}

// flatten renders a tree snapshot as one sorted, comparable slice.
func flatten(tree map[string]string) []string {
	out := make([]string, 0, len(tree))
	for path, content := range tree {
		out = append(out, path+"\x00"+content)
	}
	sort.Strings(out)
	return out
}

// snapshotContent records every file beneath dir by relative path and content,
// so a run's read-only guarantee can be checked byte-for-byte.
func snapshotContent(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = string(raw)
		return nil
	})
	it.Then(t).Must(it.Nil(err))
	return out
}

// spec FR-022 — --quiet suppresses progress while the report itself is
// still emitted, and --quiet wins over --verbose.
func TestStatsQuietSuppressesProgressNotReport(t *testing.T) {
	chdir(t, fixtureKnown(t))

	bios.Quiet, bios.Verbose = true, true
	t.Cleanup(func() { bios.Quiet, bios.Verbose = false, false })

	stdout, stderr, err := sutCaptureStderr(t, NewStatsCmd(), nil)

	it.Then(t).ShouldNot(it.Error(stdout, err))
	it.Then(t).
		Should(it.True(strings.Contains(stdout, "nodes"))).
		Should(it.True(strings.Contains(stdout, "17"))).
		Should(it.Equal("", stderr))
}

// spec FR-020, Principle X — in --json mode nothing but the document
// reaches stdout; progress goes to stderr.
func TestStatsJSONWritesOnlyDocumentToStdout(t *testing.T) {
	chdir(t, fixtureKnown(t))

	bios.JSON, bios.Verbose = true, true
	t.Cleanup(func() { bios.JSON, bios.Verbose = false, false })

	stdout, stderr, err := sutCaptureStderr(t, NewStatsCmd(), nil)
	it.Then(t).ShouldNot(it.Error(stdout, err))

	var doc map[string]any
	it.Then(t).
		Should(it.Nil(json.Unmarshal([]byte(stdout), &doc))).
		Should(it.True(strings.Contains(stderr, "Scanning graph")))
}
