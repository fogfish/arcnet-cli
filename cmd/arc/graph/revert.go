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

func sumMapCounts(m map[string]int) int {
	total := 0
	for _, n := range m {
		total += n
	}
	return total
}

type humanRevertPrinter struct{}

func (humanRevertPrinter) Show(r kernel.RevertResult) ([]byte, error) {
	if r.Skipped {
		return []byte(fmt.Sprintf("%s%s is already retracted — nothing to do\n", bios.SCHEMA.IconOK, r.Document)), nil
	}

	var parts []string
	if r.Approach == "whole-commit" {
		parts = append(parts, "whole-commit revert")
	} else {
		if n := sumMapCounts(r.Removed); n > 0 {
			parts = append(parts, fmt.Sprintf("%d removed", n))
		}
		if n := sumMapCounts(r.Reconciled); n > 0 {
			parts = append(parts, fmt.Sprintf("%d reconciled", n))
		}
		if r.LinksRemoved > 0 {
			parts = append(parts, fmt.Sprintf("%d links removed", r.LinksRemoved))
		}
		if len(parts) == 0 {
			parts = append(parts, "no nodes touched")
		}
	}

	text := fmt.Sprintf("%sReverted %s: %s (%s, commit %s)\n", bios.SCHEMA.IconOK, r.Document, strings.Join(parts, ", "), r.Approach, r.CommitHash)
	return []byte(text), nil
}

var revertRenderers = bios.Registry[kernel.RevertResult]{
	Human: humanRevertPrinter{},
}

// NewRevertCmd builds the `arc revert` command.
func NewRevertCmd() *cobra.Command {
	var result kernel.RevertResult
	var force bool

	cmd := &cobra.Command{
		Use:   "revert <source-id>",
		Short: "Retract a patch document's contribution from the graph.",
		Long: `
arc revert locates the ingest commit for <source-id> (its "Source-Id:"
trailer) and retracts that patch's contribution from the graph: a
whole-commit git revert when nothing has since touched any file it
changed, or a per-node reconciliation otherwise — removing a node
outright when the reverted patch was its sole author, or stripping only
the reverted patch's own text contribution from a node another patch has
since enriched. Re-reverting an already-retracted document is a safe
no-op.

Removing graph content is a destructive operation: arc revert asks for
confirmation unless --force/-f is given.

See more info https://github.com/fogfish/arcnet-cli`,
		Example: `
	arc revert rescorla-2026-tls13
	arc revert rescorla-2026-tls13 --force`,
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

			// The git adapter's own internal Reporter stays silent
			// unconditionally, mirroring apply.go's own precedent
			// (BUG-001): its labels are specific to arc init and would be
			// a misleading duplicate of service.Revert's own phases.
			vcs := git.New(bios.NewReporter(true, true))

			reporter := bios.NewReporter(bios.Quiet, !bios.Verbose)

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			sourceID := args[0]

			if !force {
				ok, err := bios.Confirm(fmt.Sprintf("Retract %s's contribution from the graph?", sourceID))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(os.Stderr, bios.SCHEMA.Hint.Render("aborted"))
					return bios.ErrSilent
				}
			}

			result, err = appgraph.Revert(ctx, fsys.Local{}, vcs, reporter, index, dir, sourceID)
			if err != nil {
				return err
			}

			printer := revertRenderers.Resolve(bios.ResolveMode())
			out, err := printer.Show(result)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprint(os.Stdout, string(out)); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip the confirmation prompt (required for non-interactive use)")

	return cmd
}
