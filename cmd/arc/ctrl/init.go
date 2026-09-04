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

	"github.com/spf13/cobra"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/adapter/git"
	appctrl "github.com/fogfish/arcnet-cli/internal/app/ctrl"
	"github.com/fogfish/arcnet-cli/internal/app/ctrl/kernel"
	appschema "github.com/fogfish/arcnet-cli/internal/app/schema"
	"github.com/fogfish/arcnet-cli/internal/bios"
)

func resolveInitDir(args []string) (string, error) {
	dir := "."
	if len(args) == 1 {
		dir = args[0]
	}
	return filepath.Abs(dir)
}

// repoProbe is the slice of the git adapter the repository-context
// resolution below needs. Detection is deliberately NOT on
// internal/app/ctrl/port.VCS: it is an environment probe the command layer
// performs before calling the use-case, not a decision (research.md D6).
// Every judgement made from these answers still belongs to the service.
type repoProbe interface {
	IsAvailable(ctx context.Context) error
	RepoRoot(ctx context.Context, dir string) (string, error)
	IsIgnored(ctx context.Context, dir, path string) (bool, error)
}

// resolveRepoContext answers the two questions service.Init cannot ask for
// itself: which repository, if any, encloses dir, and whether that
// repository's ignore rules exclude it.
//
// Both MUST be answered before the target directory exists, so that a
// refusal never leaves a half-created folder behind (FR-005) — which is why
// they are resolved here rather than probed from inside the service
// (research.md D2). `git -C` needs a directory that exists, so the probe
// runs at dir's nearest existing ancestor; git's own upward search then
// finds the same innermost repository it would have found from dir itself
// (FR-003).
func resolveRepoContext(ctx context.Context, probe repoProbe, mounter fsys.Mounter, dir string) (kernel.InitOpts, error) {
	var opts kernel.InitOpts

	ancestor, err := nearestExistingDir(mounter, dir)
	if err != nil || ancestor == "" {
		return opts, err
	}

	repo, err := probe.RepoRoot(ctx, ancestor)
	if err != nil || repo == "" {
		return opts, err
	}
	opts.ParentRepo = repo

	// The repository root is the one path its own ignore rules can never
	// exclude, and check-ignore rejects "." — so the question only arises
	// for a graph below the root.
	rel, err := filepath.Rel(repo, dir)
	if err != nil || rel == "." {
		return opts, nil
	}

	opts.TargetIgnored, err = probe.IsIgnored(ctx, repo, filepath.ToSlash(rel))
	return opts, err
}

// nearestExistingDir walks up from dir to the first directory that exists,
// returning "" only at a filesystem root that somehow does not. The walk
// goes through the fsys adapter rather than os.Stat, since fsys is the only
// package permitted to call os's filesystem functions (Constitution VII).
func nearestExistingDir(mounter fsys.Mounter, dir string) (string, error) {
	for {
		store, err := mounter.Mount(dir)
		if err != nil {
			return "", err
		}
		if _, err := store.Stat("."); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

// schemaSeed converts appschema.Seed()'s []byte content into the
// map[string]string shape appctrl.Init's schemaSeed parameter expects.
func schemaSeed() map[string]string {
	seed := appschema.Seed()
	out := make(map[string]string, len(seed))
	for path, content := range seed {
		out[path] = string(content)
	}
	return out
}

// humanInitPrinter carries the mode explicitly rather than inferring it
// from InitResult, because the two are not equivalent: adding a graph at a
// repository's own root reports Repository == Root, which is
// indistinguishable from a standalone init by comparison alone. The flag is
// the fact FR-026 wants stated.
type humanInitPrinter struct{ skipGitInit bool }

// Show renders the single default confirmation line (FR-016: default output
// is one concise line, no per-step progress). BUG-001: this line is plain
// text (no StatusOK green) — only the icon carries visual confirmation — and
// is built without an embedded newline, so it is never at risk of the
// lipgloss block-padding bug reporter.go's Done/Error also had to fix.
func (p humanInitPrinter) Show(r kernel.InitResult) ([]byte, error) {
	text := fmt.Sprintf("%sInitialized empty knowledge graph at %s (commit %s)", bios.SCHEMA.IconOK, r.Root.Root, r.CommitHash)

	// FR-026: when the graph was added to a repository that already
	// existed, say so and name it — otherwise "commit <hash>" reads as if
	// arc had created a repository of its own. Still one line, still one
	// trailing newline (BUG-001).
	if p.skipGitInit {
		text += fmt.Sprintf(" in existing repository %s", r.Repository)
	}

	return []byte(text + "\n"), nil
}

func initRenderers(skipGitInit bool) bios.Registry[kernel.InitResult] {
	return bios.Registry[kernel.InitResult]{
		Human: humanInitPrinter{skipGitInit: skipGitInit},
	}
}

// NewInitCmd builds the `arc init` command.
func NewInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [<dir>]",
		Short: "Initialize a new, empty knowledge graph.",
		Long: `
arc init creates the canonical folder layout, the _meta/ registry stubs, the
.arc/ local state directory with its own rule excluding it from version
control, and a single initial git commit — a ready-to-use empty knowledge
graph.

By default the target must not be inside an existing git repository: arc
refuses rather than nesting a new repository inside yours. Pass
--skip-git-init to add the graph to that repository instead — arc creates no
repository of its own, commits only the files initialization itself wrote,
and never reads or modifies the project's own .gitignore. If a canonical
folder name is already taken, initialize into a subfolder.

See more info https://github.com/fogfish/arcnet-cli`,
		Example: `
	arc init
	arc init ./my-graph
	arc init --skip-git-init ./notes`,
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := resolveInitDir(args)
			if err != nil {
				return err
			}

			skipGitInit, err := cmd.Flags().GetBool("skip-git-init")
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

			opts := kernel.InitOpts{SkipGitInit: skipGitInit}

			// Repository context is only resolvable when git runs at all.
			// When it does not, leave it unresolved and let service.Init's
			// own ErrGitUnavailable guard produce the canonical message
			// before anything is written — the probe reports nothing of
			// its own, so --verbose still shows one availability step.
			probe := git.New(bios.NewReporter(true, true))
			if probe.IsAvailable(ctx) == nil {
				resolved, err := resolveRepoContext(ctx, probe, fsys.Local{}, dir)
				if err != nil {
					return err
				}
				resolved.SkipGitInit = skipGitInit
				opts = resolved
			}

			result, err := appctrl.Init(ctx, fsys.Local{}, vcs, dir, schemaSeed(), opts)
			if err != nil {
				return err
			}

			printer := initRenderers(skipGitInit).Resolve(bios.ResolveMode())
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

	// Constitution IX: long form only. --skip-git-init is advanced,
	// infrequent usage, and shorthands are reserved for frequent flags.
	cmd.Flags().Bool("skip-git-init", false,
		"Create the graph inside the existing repository instead of starting a new one (advanced)")

	return cmd
}
