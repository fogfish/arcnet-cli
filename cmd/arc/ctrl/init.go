//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package ctrl provides Cobra wiring for the ctrl (graph management)
// domain's commands.
package ctrl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/adapter/git"
	appconfig "github.com/fogfish/arcnet-cli/internal/app/config"
	confighttp "github.com/fogfish/arcnet-cli/internal/app/config/adapter/http"
	configport "github.com/fogfish/arcnet-cli/internal/app/config/port"
	appctrl "github.com/fogfish/arcnet-cli/internal/app/ctrl"
	"github.com/fogfish/arcnet-cli/internal/app/ctrl/kernel"
	"github.com/fogfish/arcnet-cli/internal/bios"

	"gopkg.in/yaml.v3"
)

func resolveInitDir(args []string) (string, error) {
	dir := "."
	if len(args) == 1 {
		dir = args[0]
	}
	return filepath.Abs(dir)
}

// newConfigFetcher is indirected through a package-level var (rather than
// called as confighttp.New() directly) so unit tests can inject a mock
// Fetcher at the wiring layer (specs/002-arc-init/spec.md FR-017), matching
// internal/app/ctrl/service's resolveLocalRoot/removeLocalRoot precedent.
var newConfigFetcher = func() configport.Fetcher { return confighttp.New() }

// fetchConfigSeed resolves the content arc init seeds .arc/config.yml with:
// one best-effort fetch of github.com/fogfish/arcnet-spec's canonical
// config, falling back to the format's built-in merge rules on any failure
// (specs/002-arc-init/spec.md FR-017, research.md D5 revised). Reported
// under --verbose only, matching the existing progress convention.
func fetchConfigSeed(ctx context.Context, reporter bios.Reporter) []byte {
	const label = "Fetching default configuration"
	start := time.Now()

	cfg, usedFallback := appconfig.Default(ctx, newConfigFetcher())
	reporter.Done(label, time.Since(start))
	if usedFallback {
		reporter.Step("Using built-in configuration — offline or unreachable")
	}

	raw, err := yaml.Marshal(cfg)
	if err != nil {
		return nil
	}
	return raw
}

type humanInitPrinter struct{}

// Show renders the single default confirmation line (FR-016: default output
// is one concise line, no per-step progress). BUG-001: this line is plain
// text (no StatusOK green) — only the icon carries visual confirmation — and
// is built without an embedded newline, so it is never at risk of the
// lipgloss block-padding bug reporter.go's Done/Error also had to fix.
func (humanInitPrinter) Show(r kernel.InitResult) ([]byte, error) {
	text := fmt.Sprintf("%sInitialized empty knowledge graph at %s (commit %s)\n", bios.SCHEMA.IconOK, r.Root.Root, r.CommitHash)
	return []byte(text), nil
}

var initRenderers = bios.Registry[kernel.InitResult]{
	Human: humanInitPrinter{},
}

// NewInitCmd builds the `arc init` command.
func NewInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [<dir>]",
		Short: "Initialize a new, empty knowledge graph.",
		Long: `
arc init creates the canonical folder layout, the _meta/ registry stubs, the
.arc/ local state directory, a .gitignore excluding it, and a single initial
git commit — a ready-to-use empty knowledge graph.

See more info https://github.com/fogfish/arcnet-cli`,
		Example: `
	arc init
	arc init ./my-graph`,
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := resolveInitDir(args)
			if err != nil {
				return err
			}

			// BUG-001: progress is opt-in via --verbose (silent by
			// default); --quiet always wins regardless of --verbose.
			reporter := bios.NewReporter(bios.Quiet, !bios.Verbose)
			vcs := git.New(reporter)

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			configSeed := fetchConfigSeed(ctx, reporter)

			result, err := appctrl.Init(ctx, fsys.Local{}, vcs, dir, configSeed)
			if err != nil {
				return err
			}

			printer := initRenderers.Resolve(bios.ResolveMode())
			out, err := printer.Show(result)
			if err != nil {
				return err
			}

			_, err = fmt.Fprint(os.Stdout, string(out))
			return err
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			if bios.ResolveMode() == bios.ModeJSON || bios.Quiet {
				return nil
			}
			fmt.Fprintln(os.Stderr, bios.SCHEMA.Hint.Render(`(use "arc apply <patch.md>" to load content into your new graph)`))
			return nil
		},
	}

	return cmd
}
