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
	"regexp"
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
	defaultGrepWorkers      = 8
	defaultGrepMaxLineWidth = 512
)

// optsFilter is arc grep's own local --kind/--tag/--attr options struct
// (ADR 002 DS-02, research.md D14) — not yet promoted to a shared location,
// since arc grep is the first command in this codebase to implement
// VISION.md's Filtering section.
type optsFilter struct {
	kind []string
	tag  []string
	attr []string
}

func (o *optsFilter) apply(cmd *cobra.Command) {
	cmd.Flags().StringArrayVar(&o.kind, "kind", nil, "Restrict to nodes of this kind (repeatable, OR)")
	cmd.Flags().StringArrayVar(&o.tag, "tag", nil, "Restrict to nodes carrying this tag (repeatable, AND)")
	cmd.Flags().StringArrayVar(&o.attr, "attr", nil, "Restrict to nodes matching name=value or name~=pattern (repeatable, AND)")
}

// build assembles a core.Filter from the parsed flag values, per VISION.md's
// Filtering section (research.md D8): --kind is OR'd, --tag/--attr are
// AND'd, all three groups are ANDed together.
func (o optsFilter) build() (core.Filter, error) {
	f := core.Filter{}

	for _, k := range o.kind {
		f.Kinds = append(f.Kinds, core.Kind(k))
	}
	f.Tags = append(f.Tags, o.tag...)

	for _, a := range o.attr {
		if idx := strings.Index(a, "~="); idx >= 0 {
			name, pattern := a[:idx], a[idx+2:]
			re, err := regexp.Compile(pattern)
			if err != nil {
				return core.Filter{}, service.ErrInvalidAttrFlag.With(err, a)
			}
			if f.AttrPatterns == nil {
				f.AttrPatterns = map[string]*regexp.Regexp{}
			}
			f.AttrPatterns[name] = re
			continue
		}
		if idx := strings.Index(a, "="); idx >= 0 {
			name, value := a[:idx], a[idx+1:]
			if f.Attrs == nil {
				f.Attrs = map[string]string{}
			}
			f.Attrs[name] = value
			continue
		}
		return core.Filter{}, service.ErrInvalidAttrFlag.With(nil, a)
	}

	return f, nil
}

// colorEnabled reports whether the resolved bios.SCHEMA is the color
// schema — the exact same signal ADR 002 DS-05's SelectSchema already
// resolves once, at startup (research.md D11). IconOK is empty in
// SCHEMA_PLAIN and non-empty in SCHEMA_COLOR, so it doubles as that signal
// without a second, divergent TTY check.
func colorEnabled() bool {
	return bios.SCHEMA.IconOK != ""
}

// highlightSpan wraps text[start:end) in SCHEMA.Match, leaving text
// unchanged when the span is out of range.
func highlightSpan(text string, start, end int) string {
	if start < 0 || end > len(text) || start >= end {
		return text
	}
	return text[:start] + bios.SCHEMA.Match.Render(text[start:end]) + text[end:]
}

// fitWindow ellipsis-fits text to at most maxWidth bytes, keeping a window
// centered on [start:end) always visible, and returns the adjusted
// start/end offsets within the fitted string (research.md D11). text
// shorter than maxWidth is returned unchanged.
func fitWindow(text string, start, end, maxWidth int) (fitted string, newStart, newEnd int) {
	if len(text) <= maxWidth {
		return text, start, end
	}

	const ellipsis = "…"
	matchLen := end - start
	avail := maxWidth - matchLen
	if avail < 0 {
		avail = 0
	}
	left := avail / 2
	right := avail - left

	winStart := start - left
	winEnd := end + right
	if winStart < 0 {
		winEnd += -winStart
		winStart = 0
	}
	if winEnd > len(text) {
		winStart -= winEnd - len(text)
		if winStart < 0 {
			winStart = 0
		}
		winEnd = len(text)
	}

	prefix, suffix := "", ""
	if winStart > 0 {
		prefix = ellipsis
	}
	if winEnd < len(text) {
		suffix = ellipsis
	}

	fitted = prefix + text[winStart:winEnd] + suffix
	return fitted, start - winStart + len(prefix), end - winStart + len(prefix)
}

// renderMatchRow formats one match row, applying truncate-and-highlight
// only when the color schema is active (research.md D11) — piped/plain
// output is always the full, untruncated, unstyled line.
func renderMatchRow(m kernel.Match, maxWidth int, truncate bool) string {
	text := m.Text
	if colorEnabled() {
		start, end := m.Start, m.End
		if truncate {
			text, start, end = fitWindow(text, start, end, maxWidth)
		}
		text = highlightSpan(text, start, end)
	}
	return fmt.Sprintf("%s  %s  %d  %s\n", m.Kind, m.ID, m.Line, text)
}

type humanGrepPrinter struct{ maxWidth int }

// Show lists one row per match, no header/footer/summary line (spec
// FR-006/FR-007) — a long line is ellipsis-fit around the match on a color
// terminal.
func (p humanGrepPrinter) Show(r kernel.GrepResult) ([]byte, error) {
	var buf []byte
	for _, m := range r.Matches {
		buf = append(buf, renderMatchRow(m, p.maxWidth, true)...)
	}
	return buf, nil
}

type verboseGrepPrinter struct{ maxWidth int }

// Show is identical to humanGrepPrinter, except truncation is disabled —
// the full line is always shown (still colorized when applicable).
func (p verboseGrepPrinter) Show(r kernel.GrepResult) ([]byte, error) {
	var buf []byte
	for _, m := range r.Matches {
		buf = append(buf, renderMatchRow(m, p.maxWidth, false)...)
	}
	return buf, nil
}

// NewGrepCmd builds the `arc grep` command.
func NewGrepCmd() *cobra.Command {
	opts := &optsFilter{}

	cmd := &cobra.Command{
		Use:   "grep <pattern>",
		Short: "Search node content for lines matching a pattern.",
		Long: `
arc grep scans every node file's content (not just front-matter) in the
graph for lines matching a regexp <pattern>, optionally narrowed by a
--kind/--tag/--attr filter (see Filtering). One line is printed per match:
<kind>  <id>  <line>  <text> — suitable for piping to standard tools. grep
is strictly read-only.

See more info https://github.com/fogfish/arcnet-cli`,
		Example: `
	arc grep TLS
	arc grep --kind source TLS
	arc grep --tag cryptography --attr status=mature "TLS 1\.3"
	arc grep --json TLS`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			pattern := args[0]

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

			cfg := cfgFile.Grep
			if cfg.Workers <= 0 {
				cfg.Workers = defaultGrepWorkers
			}
			if cfg.MaxLineWidth <= 0 {
				cfg.MaxLineWidth = defaultGrepMaxLineWidth
			}

			filter, err := opts.build()
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			result, err := appgraph.Grep(ctx, fsys.Local{}, filter, pattern, cfg, dir)
			if err != nil {
				return err
			}

			renderers := bios.Registry[kernel.GrepResult]{
				Human:   humanGrepPrinter{maxWidth: cfg.MaxLineWidth},
				Verbose: verboseGrepPrinter{maxWidth: cfg.MaxLineWidth},
			}
			printer := renderers.Resolve(bios.ResolveMode())
			out, err := printer.Show(result)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprint(os.Stdout, string(out)); err != nil {
				return err
			}

			// DS-07: zero matches is a finding, not a refusal — signaled
			// via bios.ErrSilent after the (empty) result has already been
			// printed, exactly like arc lint's own convention
			// (research.md D12). A genuine refusal (invalid pattern, not a
			// graph) returns above, before anything is printed.
			if len(result.Matches) == 0 {
				return bios.ErrSilent
			}
			return nil
		},
	}

	opts.apply(cmd)

	return cmd
}
