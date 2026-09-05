//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package lint provides Cobra wiring for the lint (graph conformance
// validation) domain's commands.
package lint

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/adapter/git"
	applint "github.com/fogfish/arcnet-cli/internal/app/lint"
	"github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/lint/service"
	appschema "github.com/fogfish/arcnet-cli/internal/app/schema"
	"github.com/fogfish/arcnet-cli/internal/bios"
)

// errNoCause is passed to a faults.SafeN.With for guard conditions that are
// not caused by an underlying Go error, so the rendered message has no
// trailing "%!s(<nil>)" artifact (mirrors cmd/arc/graph's own precedent).
var errNoCause = errors.New("")

// knownRules is the set of every rule kernel.RuleDefinitions declares,
// computed once and shared by every parseSkip call.
var knownRules = func() map[kernel.Rule]bool {
	m := make(map[kernel.Rule]bool, len(kernel.RuleDefinitions))
	for _, def := range kernel.RuleDefinitions {
		m[def.Rule] = true
	}
	return m
}()

// parseSkip splits csv on "," trimming whitespace and dropping empty
// segments, then validates each remaining name against knownRules
// (research.md D7). unknown preserves first-occurrence order and names each
// bad value exactly once (FR-007); skip collapses duplicates by
// construction, being a map.
func parseSkip(csv string) (skip map[kernel.Rule]bool, unknown []string) {
	skip = map[kernel.Rule]bool{}
	seenUnknown := map[string]bool{}

	for _, segment := range strings.Split(csv, ",") {
		name := strings.TrimSpace(segment)
		if name == "" {
			continue
		}

		rule := kernel.Rule(name)
		if !knownRules[rule] {
			if !seenUnknown[name] {
				seenUnknown[name] = true
				unknown = append(unknown, name)
			}
			continue
		}
		skip[rule] = true
	}

	return skip, unknown
}

// filterViolations drops every entry whose Rule is in skip, preserving
// order.
func filterViolations(violations []kernel.Violation, skip map[kernel.Rule]bool) []kernel.Violation {
	var out []kernel.Violation
	for _, v := range violations {
		if skip[v.Rule] {
			continue
		}
		out = append(out, v)
	}
	return out
}

// applySkip rebuilds r with every skipped rule's violations removed, from
// both node-owned and graph-spanning violations, via the same
// NewLintResultWithForeign constructor service.Lint itself uses — so
// Passing/Failing are recomputed consistently rather than adjusted by hand
// (data-model.md, Invariant C/D).
func applySkip(r kernel.LintResult, skip map[kernel.Rule]bool) kernel.LintResult {
	filteredNodes := make([]kernel.NodeStatus, len(r.Nodes))
	for i, n := range r.Nodes {
		filteredNodes[i] = n
		filteredNodes[i].Violations = filterViolations(n.Violations, skip)
	}

	filteredGraphSpanning := filterViolations(graphSpanningViolations(r), skip)

	return kernel.NewLintResultWithForeign(r.Root, filteredNodes, r.Foreign, filteredGraphSpanning...)
}

func formatOwnedViolation(v kernel.Violation) string {
	loc := v.Path
	if v.Line > 0 {
		loc = fmt.Sprintf("%s:%d", v.Path, v.Line)
	}
	return fmt.Sprintf("%s%s — [%s] %s\n", bios.SCHEMA.IconFail, loc, v.Rule, v.Message)
}

func formatUnownedViolation(v kernel.Violation) string {
	return fmt.Sprintf("%s[%s] %s\n", bios.SCHEMA.IconFail, v.Rule, v.Message)
}

// graphSpanningViolations returns r.Violations entries with no single
// owning node (research.md D14) — RuleUniqueBasename (Path == "") and
// RuleTypeCase's schema-level occurrence (a "_schema/Class/<name>.md" Path
// that never corresponds to one of r.Nodes, since schema documents are
// excluded from that walk, spec.md Clarifications Q1/Q3).
func graphSpanningViolations(r kernel.LintResult) []kernel.Violation {
	nodePaths := make(map[string]bool, len(r.Nodes))
	for _, n := range r.Nodes {
		nodePaths[n.Path] = true
	}

	var out []kernel.Violation
	for _, v := range r.Violations {
		if !nodePaths[v.Path] {
			out = append(out, v)
		}
	}
	return out
}

func formatGraphSpanningViolation(v kernel.Violation) string {
	if v.Path == "" {
		return formatUnownedViolation(v)
	}
	return formatOwnedViolation(v)
}

// foreignNote renders the foreign-file index as one non-failing line, or
// nothing when the graph root holds no host content (spec 031 FR-035).
// It is deliberately a count rather than a list: on a graph sharing a root
// with a large project the list is long, unchanging and uninteresting, and
// burying the violations under it would defeat the purpose of the report.
// --verbose names them; --json always carries the full list.
func foreignNote(r kernel.LintResult) string {
	if len(r.Foreign) == 0 {
		return ""
	}
	return fmt.Sprintf("%s%d file(s) skipped: not graph nodes\n", bios.SCHEMA.IconWarn, len(r.Foreign))
}

// verboseForeignNote names every skipped file, so a user who expected one of
// them to be a node can see exactly which files lint declined to check and
// why (spec 031 FR-035).
func verboseForeignNote(r kernel.LintResult) string {
	var buf []byte
	for _, path := range r.Foreign {
		buf = append(buf, fmt.Sprintf("%s%s — skipped, not a graph node\n", bios.SCHEMA.IconWarn, path)...)
	}
	return string(buf)
}

func summaryLine(r kernel.LintResult) string {
	icon := bios.SCHEMA.IconOK
	if r.Failing > 0 {
		icon = bios.SCHEMA.IconFail
	}
	return fmt.Sprintf("%s%d nodes checked, %d passing, %d failing\n", icon, len(r.Nodes), r.Passing, r.Failing)
}

type humanLintPrinter struct{}

// Show lists only nodes carrying a violation, each with its rule(s), file,
// and line, followed by one overall graph-status summary line
// (research.md D14).
func (humanLintPrinter) Show(r kernel.LintResult) ([]byte, error) {
	var buf []byte
	for _, v := range graphSpanningViolations(r) {
		buf = append(buf, formatGraphSpanningViolation(v)...)
	}
	for _, n := range r.Nodes {
		for _, v := range n.Violations {
			buf = append(buf, formatOwnedViolation(v)...)
		}
	}
	buf = append(buf, foreignNote(r)...)
	buf = append(buf, summaryLine(r)...)
	return buf, nil
}

type verboseLintPrinter struct{}

// Show lists every enumerated node's individual pass/fail status, in walk
// order, followed by the identical overall summary line (research.md D14).
func (verboseLintPrinter) Show(r kernel.LintResult) ([]byte, error) {
	var buf []byte
	for _, v := range graphSpanningViolations(r) {
		buf = append(buf, formatGraphSpanningViolation(v)...)
	}
	for _, n := range r.Nodes {
		if len(n.Violations) == 0 {
			buf = append(buf, fmt.Sprintf("%s%s\n", bios.SCHEMA.IconOK, n.Path)...)
			continue
		}
		for _, v := range n.Violations {
			buf = append(buf, formatOwnedViolation(v)...)
		}
	}
	buf = append(buf, verboseForeignNote(r)...)
	buf = append(buf, summaryLine(r)...)
	return buf, nil
}

var lintRenderers = bios.Registry[kernel.LintResult]{
	Human:   humanLintPrinter{},
	Verbose: verboseLintPrinter{},
}

// NewLintCmd builds the `arc lint` command.
func NewLintCmd() *cobra.Command {
	var result kernel.LintResult
	var skipFlag string

	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Validate the graph against the CORE §14/§16 conformance checklist.",
		Long: `
arc lint walks every node file in the graph and checks it against the full
CORE §14/§16 conformance checklist: valid front-matter and kind, unique
basenames, resolvable [[link]]s, source citekey identity, entity Sowa
category, derived-node provenance, registered camelCase predicates,
schema-driven cito-aligned citation predicates, one graph(ingest): commit per
document, extension-kind recognition, absence of unresolved git
merge-conflict markers, a node's own type-declared Requires/Optional
predicate contract, "@id"/"@type" front-matter quoting, and predicate-role
structural conformance. Every violation is reported with its file and line;
the run never stops at the first one found. --skip excludes one or more
named rules (see `+"`arc lint rules`"+`) from the report entirely. lint is
strictly read-only.

See more info https://github.com/fogfish/arcnet-cli`,
		Example: `
	arc lint
	arc lint --verbose
	arc lint --json
	arc lint --skip ingestCommit`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			skip, unknown := parseSkip(skipFlag)
			if len(unknown) > 0 {
				return service.ErrUnknownSkipRule.With(errNoCause, strings.Join(unknown, ", "))
			}

			dir, err := filepath.Abs(".")
			if err != nil {
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

			// The git adapter's own internal Reporter stays silent
			// unconditionally — CommitsMatching has no progress label of
			// its own worth surfacing separately from service.Lint's own
			// "Checking commit history" phase.
			vcs := git.New(bios.NewReporter(true, true))

			// Progress is opt-in via --verbose (silent by default);
			// --quiet always wins regardless of --verbose.
			reporter := bios.NewReporter(bios.Quiet, !bios.Verbose)

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			result, err = applint.Lint(ctx, fsys.Local{}, vcs, reporter, index, dir)
			if err != nil {
				return err
			}

			if len(skip) > 0 {
				result = applySkip(result, skip)
			}

			printer := lintRenderers.Resolve(bios.ResolveMode())
			out, err := printer.Show(result)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprint(os.Stdout, string(out)); err != nil {
				return err
			}

			// DS-07: a distinct non-zero exit code when violations are
			// found, signaled via a sentinel error returned after the
			// result has already been printed — never a bare os.Exit
			// inside RunE, and never a second "error line" for what is a
			// finding, not a refusal (contracts/cli-contract.md).
			if len(result.Violations) > 0 {
				return bios.ErrSilent
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&skipFlag, "skip", "", "Comma-separated rule names to exclude from the report (see `arc lint rules`)")

	return cmd
}
