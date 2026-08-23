//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package ctrl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/adapter/git"
	appctrl "github.com/fogfish/arcnet-cli/internal/app/ctrl"
	"github.com/fogfish/arcnet-cli/internal/app/ctrl/kernel"
	appschema "github.com/fogfish/arcnet-cli/internal/app/schema"
	schemakernel "github.com/fogfish/arcnet-cli/internal/app/schema/kernel"
	"github.com/fogfish/arcnet-cli/internal/bios"
)

type humanUpgradePrinter struct{}

// Show renders the one-line outcome, plus — when there is anything to
// review — the advisory prose-drift list (contract C3.7). The review block
// never affects the exit code: it reports nodes an EARLIER release's merge
// behaviour may have damaged, which this command deliberately does not
// repair (FR-023).
func (humanUpgradePrinter) Show(r kernel.UpgradeResult) ([]byte, error) {
	var out strings.Builder

	switch {
	case r.DryRun:
		fmt.Fprintf(&out, "%swould replace %d, add %d, remove %d schema documents (dry run — nothing written)\n",
			bios.SCHEMA.IconOK, len(r.Replaced), len(r.Added), len(r.Removed))
	case r.CommitHash == "":
		// Mirrors arc apply schema's existing "nothing to commit" phrasing
		// (contract C3.4).
		fmt.Fprintf(&out, "%sgraph schema is already up to date — nothing to commit\n", bios.SCHEMA.IconOK)
	default:
		fmt.Fprintf(&out, "%sUpgraded %s: %d replaced, %d added, %d removed (commit %s)\n",
			bios.SCHEMA.IconOK, r.Root.Root, len(r.Replaced), len(r.Added), len(r.Removed), r.CommitHash)
	}

	if len(r.NeedsReview) > 0 {
		fmt.Fprintf(&out, "%s%d node(s) may carry prose accumulated by the previous merge behaviour — review manually:\n",
			bios.SCHEMA.IconWarn, len(r.NeedsReview))
		for _, path := range r.NeedsReview {
			fmt.Fprintf(&out, "    %s\n", path)
		}
	}

	return []byte(out.String()), nil
}

var upgradeRenderers = bios.Registry[kernel.UpgradeResult]{
	Human: humanUpgradePrinter{},
}

// NewUpgradeCmd builds the `arc upgrade` command — a top-level sibling of
// `init` (research.md D11): the same operation over a graph's built-in
// vocabulary, one for a new graph and one for an existing one.
func NewUpgradeCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Bring an existing graph's built-in schema up to date with this release of arc.",
		Long: `
arc upgrade replaces a graph's built-in predicate and type definitions with
the ones this release seeds, in exactly one commit. Your content is never
touched: every file outside _schema/, and every _schema/ document you wrote
yourself, is left exactly as it is.

Built-in definitions are replaced outright rather than merged, because every
correction is a retraction — dropping a required predicate, changing a merge
behaviour — that the additive merge path cannot express. A built-in document
you have hand-edited is therefore replaced too; extend the vocabulary by
adding your own documents instead.

Running it on an already-current graph changes nothing and creates no commit.

Nodes whose prose looks like it accumulated under an older merge behaviour
are listed for you to review. They are never rewritten — the boundary
between the original text and what was appended to it is unrecoverable.

See more info https://github.com/fogfish/arcnet-cli`,
		Example: `
	arc upgrade
	arc upgrade --dry-run
	arc upgrade --json`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := filepath.Abs(".")
			if err != nil {
				return err
			}

			// BUG-001 convention (init.go, apply_schema.go): progress is
			// opt-in via --verbose (silent by default); --quiet always wins.
			reporter := bios.NewReporter(bios.Quiet, !bios.Verbose)
			vcs := git.New(reporter)

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			result, err := appctrl.Upgrade(ctx, fsys.Local{}, vcs, appschema.Component{},
				dir, appschema.Seed(), schemakernel.RetiredBuiltIns, dryRun)
			if err != nil {
				return err
			}

			printer := upgradeRenderers.Resolve(bios.ResolveMode())
			out, err := printer.Show(result)
			if err != nil {
				return err
			}

			_, err = fmt.Fprint(os.Stdout, string(out))
			return err
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Report what would change without writing anything")

	return cmd
}
