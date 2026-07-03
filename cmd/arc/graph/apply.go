//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package graph provides Cobra wiring for the graph (graph I/O) domain's
// commands.
package graph

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/adapter/git"
	appconfig "github.com/fogfish/arcnet-cli/internal/app/config"
	appgraph "github.com/fogfish/arcnet-cli/internal/app/graph"
	"github.com/fogfish/arcnet-cli/internal/app/graph/kernel"
	"github.com/fogfish/arcnet-cli/internal/bios"
	"github.com/fogfish/arcnet-cli/internal/core"
)

func pluralizeKind(kind core.Kind, count int) string {
	if count == 1 {
		return string(kind)
	}
	if kind == "entity" {
		return "entities"
	}
	s := string(kind)
	if strings.HasSuffix(s, "s") {
		return s
	}
	return s + "s"
}

func sortedKindsUnion(a, b map[core.Kind]int) []core.Kind {
	seen := map[core.Kind]bool{}
	kinds := make([]core.Kind, 0, len(a)+len(b))
	for k := range a {
		if !seen[k] {
			kinds = append(kinds, k)
			seen[k] = true
		}
	}
	for k := range b {
		if !seen[k] {
			kinds = append(kinds, k)
			seen[k] = true
		}
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })
	return kinds
}

type humanApplyPrinter struct{}

func (humanApplyPrinter) Show(r kernel.ApplyResult) ([]byte, error) {
	if r.Skipped {
		return []byte(fmt.Sprintf("%s%s is already tracked — nothing to do\n", bios.SCHEMA.IconOK, r.Document)), nil
	}

	kinds := sortedKindsUnion(r.Created, r.Merged)
	parts := make([]string, 0, len(kinds))
	for _, k := range kinds {
		created := r.Created[k]
		part := fmt.Sprintf("+%d %s", created, pluralizeKind(k, created))
		if merged := r.Merged[k]; merged > 0 {
			part += fmt.Sprintf(" (%d merged)", merged)
		}
		parts = append(parts, part)
	}

	text := fmt.Sprintf("%sApplied %s: %s (commit %s)\n", bios.SCHEMA.IconOK, r.Document, strings.Join(parts, ", "), r.CommitHash)
	return []byte(text), nil
}

var applyRenderers = bios.Registry[kernel.ApplyResult]{
	Human: humanApplyPrinter{},
}

// NewApplyCmd builds the `arc apply` command.
func NewApplyCmd() *cobra.Command {
	var result kernel.ApplyResult

	cmd := &cobra.Command{
		Use:   "apply <patch.md>",
		Short: "Apply a patch document patch to the graph.",
		Long: `
arc apply parses a document patch and creates or merges every node it
carries into the graph, deriving and appending timeline entries, and
producing exactly one commit. Re-applying an already-tracked document is a
safe no-op.

The patch format is specified at https://github.com/fogfish/arcnet-spec

See more info https://github.com/fogfish/arcnet-cli`,
		Example: `
	arc apply rescorla-2026-tls13.patch.md`,
		Args:          cobra.ExactArgs(1),
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

			rules, err := appconfig.Resolve(store)
			if err != nil {
				return err
			}

			// The git adapter's own internal Reporter stays silent
			// unconditionally — its "Committing empty graph" label is
			// specific to arc init and would be a misleading duplicate
			// of service.Apply's own "Committing" phase below (BUG-001).
			vcs := git.New(bios.NewReporter(true, true))

			// BUG-001: progress is opt-in via --verbose (silent by
			// default), matching cmd/arc/ctrl/init.go's convention
			// exactly; --quiet always wins regardless of --verbose.
			reporter := bios.NewReporter(bios.Quiet, !bios.Verbose)

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			patchPath, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}

			result, err = appgraph.Apply(ctx, fsys.Local{}, vcs, reporter, rules, dir, patchPath)
			if err != nil {
				return err
			}

			printer := applyRenderers.Resolve(bios.ResolveMode())
			out, err := printer.Show(result)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprint(os.Stdout, string(out)); err != nil {
				return err
			}

			if bios.ResolveMode() != bios.ModeJSON && !bios.Quiet {
				for _, w := range result.Warnings {
					fmt.Fprintln(os.Stderr, bios.SCHEMA.StatusWarn.Render(bios.SCHEMA.IconWarn+w))
				}
			}

			return nil
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			if bios.ResolveMode() == bios.ModeJSON || bios.Quiet {
				return nil
			}
			if len(result.Conflicts) == 0 {
				fmt.Fprintln(os.Stderr, bios.SCHEMA.Hint.Render(`(use "arc grep [<filter>] <pattern>" to fecth content from your graph, see arc help for other graph use-cases)`))
				return nil
			}
			fmt.Fprintln(os.Stderr, bios.SCHEMA.Hint.Render(fmt.Sprintf(
				`(a merge conflict was flagged in %s — resolve it manually before the next apply)`,
				strings.Join(result.Conflicts, ", "))))
			return nil
		},
	}

	return cmd
}
