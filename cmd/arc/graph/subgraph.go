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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	appconfig "github.com/fogfish/arcnet-cli/internal/app/config"
	appgraph "github.com/fogfish/arcnet-cli/internal/app/graph"
	"github.com/fogfish/arcnet-cli/internal/app/graph/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/graph/service"
	"github.com/fogfish/arcnet-cli/internal/bios"
	"github.com/fogfish/arcnet-cli/internal/core"
)

const (
	defaultSubgraphDirectCap   = 4096
	defaultSubgraphBacklinkCap = 1024
)

// errNoCause is passed to a faults.SafeN.With for guard conditions that
// are not caused by an underlying Go error, so the rendered message has
// no trailing "%!s(<nil>)" artifact (mirrors internal/app/graph/service's
// own precedent).
var errNoCause = errors.New("")

type humanSubgraphPrinter struct{}

// Show writes core.RenderPatch's bytes verbatim to stdout — no
// bios.SCHEMA styling of any kind (research.md D10): this command's
// entire stdout contract is a structured, machine/LLM-consumable document
// (round-trip target: arc apply), not a colorized human table.
func (humanSubgraphPrinter) Show(r kernel.SubgraphResult) ([]byte, error) {
	return core.RenderPatch(r.Patch)
}

var subgraphRenderers = bios.Registry[kernel.SubgraphResult]{
	Human: humanSubgraphPrinter{},
}

// truncationNotice renders the plain, unstyled stderr diagnostic line for
// a truncated pool (research.md D10, spec FR-015) — no color, matching
// this command's own no-styling convention.
func truncationNotice(r kernel.SubgraphResult) string {
	var parts []string
	if r.DirectTruncated {
		parts = append(parts, fmt.Sprintf("direct-reachable set truncated to %d of %d nodes (most-connected kept)", r.DirectIncluded, r.DirectReachable))
	}
	if r.BacklinkTruncated {
		parts = append(parts, fmt.Sprintf("backlink-reachable set truncated to %d of %d nodes (most-connected kept)", r.BacklinkIncluded, r.BacklinkReachable))
	}
	return "subgraph: " + strings.Join(parts, "; ")
}

// NewSubgraphCmd builds the `arc subgraph` command.
func NewSubgraphCmd() *cobra.Command {
	opts := &optsFilter{}
	var depth int
	var stubs bool

	cmd := &cobra.Command{
		Use:   "subgraph <basename>",
		Short: "Extract a self-contained subgraph around a node.",
		Long: `
arc subgraph extracts the seed node named by <basename> plus everything
reachable from it within --depth hops — following both a node's own
outgoing structural connections and any other node's connection that
targets it — optionally narrowed by a --kind/--tag/--attr filter (see
Filtering; the filter never excludes the seed itself). The result is
serialized as one patch-exchange document, grouped by kind, ready to
re-ingest via arc apply or paste into an LLM prompt. subgraph is strictly
read-only.

See more info https://github.com/fogfish/arcnet-cli`,
		Example: `
	arc subgraph TLS
	arc subgraph TLS --depth 2
	arc subgraph TLS --kind source
	arc subgraph TLS --json
	arc subgraph TLS --stubs`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			basename := args[0]

			if depth < 0 {
				return service.ErrInvalidDepth.With(errNoCause, strconv.Itoa(depth))
			}

			dir, err := filepath.Abs(".")
			if err != nil {
				return err
			}

			store, err := (fsys.Local{}).Mount(dir)
			if err != nil {
				return err
			}

			cfgFile, err := appconfig.Load(store)
			if err != nil {
				return err
			}

			cfg := cfgFile.Subgraph
			if cfg.DirectCap <= 0 {
				cfg.DirectCap = defaultSubgraphDirectCap
			}
			if cfg.BacklinkCap <= 0 {
				cfg.BacklinkCap = defaultSubgraphBacklinkCap
			}

			filter, err := opts.build()
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			result, err := appgraph.Subgraph(ctx, fsys.Local{}, filter, basename, depth, cfg, dir, stubs)
			if err != nil {
				return err
			}

			printer := subgraphRenderers.Resolve(bios.ResolveMode())
			out, err := printer.Show(result)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprint(os.Stdout, string(out)); err != nil {
				return err
			}

			// research.md D11: extraction always succeeds once it runs at
			// all (the seed is always present) — there is no "ran, found
			// nothing" outcome to signal via bios.ErrSilent, unlike arc
			// grep/arc lint. A genuine refusal returned above, before
			// anything was printed.
			if (result.DirectTruncated || result.BacklinkTruncated) && bios.ResolveMode() != bios.ModeJSON {
				fmt.Fprintln(os.Stderr, truncationNotice(result))
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&depth, "depth", 1, "Number of hops to traverse from the seed (both directions)")
	cmd.Flags().BoolVar(&stubs, "stubs", false, "Emit a minimal placeholder node (kind and id only) for every extraction-boundary link target, so the output has no dangling reference when applied elsewhere")
	opts.apply(cmd)

	return cmd
}
