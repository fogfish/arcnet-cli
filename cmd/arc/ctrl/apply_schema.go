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
	"time"

	"github.com/spf13/cobra"

	nethttp "github.com/fogfish/arcnet-cli/internal/adapter/http"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/adapter/git"
	appschema "github.com/fogfish/arcnet-cli/internal/app/schema"
	"github.com/fogfish/arcnet-cli/internal/app/schema/kernel"
	"github.com/fogfish/arcnet-cli/internal/bios"
)

const defaultApplySchemaTimeout = 30 * time.Second

func pluralizeSchemaKind(kind string, count int) string {
	if count == 1 {
		return kind
	}
	return kind + "s"
}

type humanApplySchemaPrinter struct{}

func (humanApplySchemaPrinter) Show(r kernel.ApplySchemaResult) ([]byte, error) {
	if r.CommitHash == "" {
		return []byte(fmt.Sprintf("%s%s introduced no schema changes — nothing to commit\n", bios.SCHEMA.IconOK, r.Source)), nil
	}

	parts := make([]string, 0, 2)
	for _, kind := range []string{"predicate", "type"} {
		created := r.Created[kind]
		part := fmt.Sprintf("+%d %s", created, pluralizeSchemaKind(kind, created))
		if merged := r.Merged[kind]; merged > 0 {
			part += fmt.Sprintf(" (%d merged)", merged)
		}
		parts = append(parts, part)
	}

	text := fmt.Sprintf("%sApplied %s: %s (commit %s)\n", bios.SCHEMA.IconOK, r.Source, strings.Join(parts, ", "), r.CommitHash)
	return []byte(text), nil
}

var applySchemaRenderers = bios.Registry[kernel.ApplySchemaResult]{
	Human: humanApplySchemaPrinter{},
}

// NewApplySchemaCmd builds the `arc apply schema` command.
func NewApplySchemaCmd() *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "schema <patch.md> | <url> | arcnet:<name>",
		Short: "Import Property/Class schema definitions from a patch document.",
		Long: `
arc apply schema reads a patch document — a local file, a URL, or an
arcnet:<name> shorthand into the official arcnet extensions catalog —
restricted to Property/Class node sections, and creates or merges each one
into the graph's _schema/ documents. Any non-Property/Class node anywhere
in the patch fails the whole operation before any write happens.

See more info https://github.com/fogfish/arcnet-cli`,
		Example: `
	arc apply schema arcnet-ext-media.schema.md
	arc apply schema https://example.org/schemas/arcnet-ext-media.schema.md
	arc apply schema arcnet:media.schema.md`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := filepath.Abs(".")
			if err != nil {
				return err
			}

			vcs := git.New(bios.NewReporter(true, true))
			fetcher := nethttp.New(timeout)

			// BUG-001 convention (cmd/arc/ctrl/init.go, cmd/arc/graph/apply.go):
			// progress is opt-in via --verbose (silent by default); --quiet
			// always wins regardless of --verbose.
			reporter := bios.NewReporter(bios.Quiet, !bios.Verbose)

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			result, err := appschema.ApplyPatch(ctx, fsys.Local{}, vcs, fetcher, reporter, dir, args[0])
			if err != nil {
				return err
			}

			printer := applySchemaRenderers.Resolve(bios.ResolveMode())
			out, err := printer.Show(result)
			if err != nil {
				return err
			}

			_, err = fmt.Fprint(os.Stdout, string(out))
			return err
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", defaultApplySchemaTimeout, "Maximum time allowed to fetch a URL or arcnet:-resolved input")

	return cmd
}
