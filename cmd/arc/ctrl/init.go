// Package ctrl provides Cobra wiring for the ctrl (graph management)
// domain's commands.
package ctrl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	appctrl "github.com/fogfish/arcnet-cli/internal/app/ctrl"
	"github.com/fogfish/arcnet-cli/internal/app/ctrl/adapter/git"
	"github.com/fogfish/arcnet-cli/internal/app/ctrl/kernel"
	"github.com/fogfish/arcnet-cli/internal/bios"
)

func resolveInitDir(args []string) (string, error) {
	dir := "."
	if len(args) == 1 {
		dir = args[0]
	}
	return filepath.Abs(dir)
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

			result, err := appctrl.Init(ctx, fsys.Local{}, vcs, dir)
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
