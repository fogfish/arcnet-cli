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
	"strings"

	"github.com/spf13/cobra"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/adapter/git"
	appgraph "github.com/fogfish/arcnet-cli/internal/app/graph"
	"github.com/fogfish/arcnet-cli/internal/app/graph/kernel"
	appschema "github.com/fogfish/arcnet-cli/internal/app/schema"
	"github.com/fogfish/arcnet-cli/internal/bios"
)

func pluralizePatches(count int) string {
	if count == 1 {
		return "patch"
	}
	return "patches"
}

type humanBatchPrinter struct{}

// Show renders the closing batch summary (FR-013): one headline carrying
// every count, then — only when there is something to say — the per-failure
// block and the run-wide conflict list (FR-016), so a conflict flagged early
// in a long run is read at the end rather than lost in scrollback.
func (humanBatchPrinter) Show(r kernel.BatchResult) ([]byte, error) {
	if len(r.Patches) == 0 {
		return []byte(fmt.Sprintf("%sNo patches to apply in %s\n", bios.SCHEMA.IconOK, r.Directory)), nil
	}

	counts := []string{
		fmt.Sprintf("%d skipped", r.Skipped),
		fmt.Sprintf("%d failed", r.Failed),
	}
	if r.Unprocessed > 0 {
		counts = append(counts, fmt.Sprintf("%d unprocessed", r.Unprocessed))
	}
	counts = append(counts, fmt.Sprintf("%d not a patch", r.NotAPatch))

	var out strings.Builder
	fmt.Fprintf(&out, "%sApplied %d %s from %s (%s)\n",
		bios.SCHEMA.IconOK, r.Applied, pluralizePatches(r.Applied), r.Directory, strings.Join(counts, ", "))

	if r.Failed > 0 {
		fmt.Fprint(&out, "\n  failed:\n")
		for _, p := range r.Patches {
			if p.Outcome == kernel.OutcomeFailed {
				fmt.Fprintf(&out, "    %s — %s\n", p.Path, p.Reason)
			}
		}
	}

	if len(r.Conflicts) > 0 {
		fmt.Fprintf(&out, "\n  conflicts flagged in: %s\n", strings.Join(r.Conflicts, ", "))
	}

	return []byte(out.String()), nil
}

var batchRenderers = bios.Registry[kernel.BatchResult]{
	Human: humanBatchPrinter{},
}

// NewApplyBatchCmd builds the `arc apply batch` command.
func NewApplyBatchCmd() *cobra.Command {
	var failFast bool

	cmd := &cobra.Command{
		Use:   "batch <dir>",
		Short: "Apply every patch found in a directory, in publication order.",
		Long: `
arc apply batch walks <dir> recursively, classifies every *.md file it
finds, and applies each applicable patch through the same algorithm arc
apply uses — oldest publication date first, one commit per patch.

Markdown that declares no patch manifest is passed over rather than failed,
hidden directories are never descended into, and a document already tracked
in the graph is skipped, so re-running over a growing directory is safe.

By default a patch that cannot be applied is recorded and the run continues;
--fail-fast halts at the first failure instead, leaving the remainder
unprocessed. Either way the command exits non-zero if any patch failed.

See more info https://github.com/fogfish/arcnet-cli`,
		Example: `
	arc apply batch ./patches
	arc apply batch --fail-fast ./patches
	arc apply batch --json ./patches`,
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

			index, err := appschema.Resolve(store)
			if err != nil {
				return err
			}

			patchDir, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}

			// The git adapter's own internal Reporter stays silent, and
			// service.Apply's per-node detail remains --verbose-gated
			// exactly as arc apply leaves it; the batch's own per-patch
			// progress is on unless --quiet (FR-015, research.md D9).
			vcs := git.New(bios.NewReporter(true, true))
			reporter := bios.NewReporter(bios.Quiet, !bios.Verbose)
			batchReporter := bios.NewReporter(bios.Quiet, bios.ResolveMode() == bios.ModeJSON)

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			result, err := appgraph.ApplyBatch(ctx, fsys.Local{}, vcs, reporter, batchReporter,
				index, appschema.Component{}, dir, patchDir, failFast)
			if err != nil {
				return err
			}

			printer := batchRenderers.Resolve(bios.ResolveMode())
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
				printBatchHint(result)
			}

			// The summary is already printed, so exit 1 through the silent
			// sentinel rather than a second, redundant error line (FR-014,
			// research.md D8).
			if result.Failed > 0 {
				return bios.ErrSilent
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&failFast, "fail-fast", false,
		"Halt at the first failing patch instead of continuing through the rest")

	return cmd
}

func printBatchHint(r kernel.BatchResult) {
	switch {
	case len(r.Conflicts) > 0:
		fmt.Fprintln(os.Stderr, bios.SCHEMA.Hint.Render(fmt.Sprintf(
			`(a merge conflict was flagged in %s — resolve it manually before the next apply)`,
			strings.Join(r.Conflicts, ", "))))
	case r.Failed > 0:
		fmt.Fprintln(os.Stderr, bios.SCHEMA.Hint.Render(
			`(fix the patches listed above and re-run — already-applied documents are skipped)`))
	case r.Applied > 0:
		fmt.Fprintln(os.Stderr, bios.SCHEMA.Hint.Render(
			`(use "arc grep [<filter>] <pattern>" to fecth content from your graph, see arc help for other graph use-cases)`))
	}
}
