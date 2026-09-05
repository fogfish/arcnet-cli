//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package graph

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	appgraph "github.com/fogfish/arcnet-cli/internal/app/graph"
	"github.com/fogfish/arcnet-cli/internal/app/graph/kernel"
	appschema "github.com/fogfish/arcnet-cli/internal/app/schema"
	"github.com/fogfish/arcnet-cli/internal/bios"
)

// labelScanningGraph is arc stats' single reporter phase — the command has
// exactly one, because it makes exactly one pass over the graph.
const labelScanningGraph = "Scanning graph"

// statsRow renders one "label ..... value" line at a fixed width, so a
// column of figures lines up whatever the labels are.
func statsRow(label string, value string) string {
	return fmt.Sprintf("  %-28s %8s\n", label, value)
}

func statsCount(label string, value int) string {
	return statsRow(label, strconv.Itoa(value))
}

// statsSection renders a titled block, or nothing at all when the block
// would be empty — an empty graph should report zeros, not a page of
// headings with nothing under them.
func statsSection(title string, rows []string) string {
	if len(rows) == 0 {
		return ""
	}
	return "\n  " + title + "\n" + strings.Join(rows, "")
}

func typeRows(entries []kernel.TypeCount) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, statsCount("  "+e.Type, e.Count))
	}
	return out
}

func predicateRows(entries []kernel.PredicateCount) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, statsCount("  "+e.Predicate, e.Count))
	}
	return out
}

func periodRows(entries []kernel.PeriodCount) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, statsCount("  "+e.Period, e.Count))
	}
	return out
}

func targetRows(entries []kernel.TargetCount) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, statsCount("  "+e.Target, e.Refs))
	}
	return out
}

func rankRows(entries []kernel.RefRank) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, statsCount("  "+e.ID, e.Refs))
	}
	return out
}

func nameRows(names []string) []string {
	out := make([]string, 0, len(names))
	for _, name := range names {
		out = append(out, "    "+name+"\n")
	}
	return out
}

// skippedNote reports the files the scan set aside, keeping the two kinds
// apart: a file that failed to parse is graph damage, an ordinary host
// README beside the graph is not (spec FR-009, research.md D8).
func skippedNote(r kernel.StatsResult) string {
	if len(r.Unreadable) == 0 && len(r.Foreign) == 0 {
		return ""
	}

	buf := strings.Builder{}
	buf.WriteString("\n")
	if len(r.Unreadable) > 0 {
		buf.WriteString(fmt.Sprintf("%s%d file(s) unreadable: could not be parsed as nodes\n", bios.SCHEMA.IconFail, len(r.Unreadable)))
	}
	if len(r.Foreign) > 0 {
		buf.WriteString(fmt.Sprintf("%s%d file(s) skipped: not graph nodes\n", bios.SCHEMA.IconWarn, len(r.Foreign)))
	}
	return buf.String()
}

// countingRules states, in the report itself, the two rules that
// deliberately differ, so a reader never has to infer why a graph can
// report few edges and many broken links (spec FR-006a).
func countingRules() string {
	lines := []string{
		"edges count structural link occurrences: one node linking twice to one target counts twice.",
		"broken links count each distinct unresolved target once per node that references it,",
		"across structural edges and inline references alike.",
	}
	var buf strings.Builder
	buf.WriteString("\n")
	for _, line := range lines {
		buf.WriteString(bios.SCHEMA.Hint.Render("  "+line) + "\n")
	}
	return buf.String()
}

func statsSummary(r kernel.StatsResult) string {
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("graph %s\n\n", r.Root))
	buf.WriteString(statsCount("nodes", r.Nodes))
	buf.WriteString(statsCount("edges", r.Edges))
	buf.WriteString(statsCount("broken links", r.BrokenLinks))
	buf.WriteString(statsSection("nodes by type", typeRows(r.ByType)))
	buf.WriteString(statsSection("ingestion by year", periodRows(r.Ingestion)))
	buf.WriteString(skippedNote(r))
	return buf.String()
}

func connectivityRows(d kernel.StatsDetail) []string {
	return []string{
		statsCount("  disconnected nodes", d.Orphans),
		statsCount("  stub nodes", d.Stubs),
		statsRow("  average out-degree", strconv.FormatFloat(d.AvgOutDegree, 'f', 2, 64)),
		statsRow("  median out-degree", strconv.FormatFloat(d.MedianOutDegree, 'f', 2, 64)),
	}
}

func schemaRows(c kernel.SchemaCoverage) []string {
	return []string{
		statsRow("  classes (used / declared)", fmt.Sprintf("%d / %d", c.ClassesUsed, c.Classes)),
		statsRow("  properties (used / declared)", fmt.Sprintf("%d / %d", c.PropertiesUsed, c.Properties)),
	}
}

func contentRows(c kernel.ContentVolume) []string {
	return []string{
		statsCount("  inline references", c.InlineRefs),
		statsCount("  attribute values", c.AttributeValues),
		statsCount("  nodes without a date", c.NodesWithoutPublished),
	}
}

func statsDetailReport(d kernel.StatsDetail) string {
	var buf strings.Builder
	buf.WriteString(statsSection("edges by predicate", predicateRows(d.ByPredicate)))
	buf.WriteString(statsSection("ingestion by month", periodRows(d.IngestionMonthly)))
	buf.WriteString(statsSection("connectivity", connectivityRows(d)))
	buf.WriteString(statsSection("unresolved targets", targetRows(d.UnresolvedTargets)))
	buf.WriteString(statsSection("most referenced", rankRows(d.TopReferenced)))
	buf.WriteString(statsSection("schema coverage", schemaRows(d.Schema)))
	buf.WriteString(statsSection("undeclared predicates", nameRows(d.Schema.UndeclaredPredicates)))
	buf.WriteString(statsSection("content volume", contentRows(d.Content)))
	return buf.String()
}

type humanStatsPrinter struct{}

// Show renders the five summary figures plus the counting-rule note every
// mode carries (spec FR-006a).
func (humanStatsPrinter) Show(r kernel.StatsResult) ([]byte, error) {
	return []byte(statsSummary(r) + countingRules()), nil
}

type verboseStatsPrinter struct{}

// Show renders the summary, then every detail figure the service computed.
func (verboseStatsPrinter) Show(r kernel.StatsResult) ([]byte, error) {
	var buf strings.Builder
	buf.WriteString(statsSummary(r))
	if r.Detail != nil {
		buf.WriteString(statsDetailReport(*r.Detail))
	}
	buf.WriteString(countingRules())
	return []byte(buf.String()), nil
}

var statsRenderers = bios.Registry[kernel.StatsResult]{
	Human:   humanStatsPrinter{},
	Verbose: verboseStatsPrinter{},
}

// NewStatsCmd builds the `arc stats` command.
func NewStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Report the shape and health of the graph.",
		Long: `
arc stats reads the graph in the current directory and reports its shape
and health in a single pass: how many nodes it holds, how many structural
edges connect them, how those nodes break down by declared type, how many
links point at nothing, and how much was ingested in each year. --verbose
adds edges by predicate, monthly ingestion, connectivity health, degree
and hub ranking, schema coverage, and content volume.

Nodes are recognized by what each file declares, never by the folder
holding it, so reorganizing the graph changes no reported figure. Files the
graph does not own — a host project's own README or design notes — are
reported apart from files that genuinely failed to parse. stats exits
successfully whenever it produced a report, broken links included: gating
on graph health is arc lint's job. stats is strictly read-only.

See more info https://github.com/fogfish/arcnet-cli`,
		Example: `
	arc stats
	arc stats --verbose
	arc stats --json`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := filepath.Abs(".")
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			// Refuse a directory that is not a graph through the shared
			// graph sentinel, before the schema index is resolved, so
			// arc stats and every other graph command refuse identically
			// (spec FR-011).
			if err := appgraph.EnsureGraph(ctx, fsys.Local{}, dir); err != nil {
				return err
			}

			store, err := (fsys.Local{}).Mount(dir)
			if err != nil {
				return err
			}

			index, err := appschema.Resolve(store)
			if err != nil {
				return err
			}

			// Progress is opt-in via --verbose (silent by default);
			// --quiet always wins regardless of --verbose (spec FR-022).
			reporter := bios.NewReporter(bios.Quiet, !bios.Verbose)
			start := time.Now()

			result, err := appgraph.Stats(ctx, fsys.Local{}, index, dir, bios.Verbose)
			if err != nil {
				reporter.Error(labelScanningGraph, err)
				return err
			}
			reporter.Done(labelScanningGraph, time.Since(start))

			printer := statsRenderers.Resolve(bios.ResolveMode())
			out, err := printer.Show(result)
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(os.Stdout, string(out))
			return err
		},
	}

	return cmd
}
