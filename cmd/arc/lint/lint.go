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
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/adapter/git"
	applint "github.com/fogfish/arcnet-cli/internal/app/lint"
	"github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
	appschema "github.com/fogfish/arcnet-cli/internal/app/schema"
	"github.com/fogfish/arcnet-cli/internal/bios"
)

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
// RuleTypeCase's schema-level occurrence (a "_schema/types/<name>.md" Path
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
the run never stops at the first one found. lint is strictly read-only.

See more info https://github.com/fogfish/arcnet-cli`,
		Example: `
	arc lint
	arc lint --verbose
	arc lint --json`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
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

	return cmd
}
