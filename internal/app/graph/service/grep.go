//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	configkernel "github.com/fogfish/arcnet-cli/internal/app/config/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/graph/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
	"github.com/fogfish/arcnet-cli/internal/pkg/grep"
)

// defaultGrepWorkers matches internal/pkg/grep.Search's own default,
// applied here so a caller-resolved kernel.GrepConfig.Workers <= 0 always
// means "use the built-in default" (research.md D10).
const defaultGrepWorkers = 8

// walkFiles recursively walks store from its root, collecting every *.md
// file whose containing directory skipDir does not reject, in
// deterministic (sorted) order. The skip predicate is a parameter rather
// than a hard-coded set because two callers need two different universes:
// ordinary content walks exclude _schema/, while a whole-graph census must
// include it (specs/032-arc-stats research.md D2). A boolean flag was
// rejected — walkNodeFiles(store, true) communicates nothing at a call
// site, whereas the named wrappers below do.
func walkFiles(store fsys.Store, skipDir func(string) bool) ([]string, error) {
	var out []string
	var walk func(dir string) error
	walk = func(dir string) error {
		entries, err := store.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, e := range entries {
			full := e.Name()
			if dir != "." {
				full = dir + "/" + e.Name()
			}
			if e.IsDir() {
				if skipDir(full) {
					continue
				}
				if err := walk(full); err != nil {
					return err
				}
				continue
			}
			if !strings.HasSuffix(full, ".md") {
				continue
			}
			out = append(out, full)
		}
		return nil
	}

	if err := walk("."); err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

// walkNodeFiles collects every *.md file except anything under .arc/ or
// _schema/ (mirrors internal/app/lint/service.walkNodeFiles, research.md
// D9). Shared by Grep, Subgraph, Match and Context (research.md D7 in
// specs/007-arc-subgraph) — a second, copy-pasted walker in this package
// would be exactly the drift Principle V exists to prevent.
func walkNodeFiles(store fsys.Store) ([]string, error) {
	return walkFiles(store, func(dir string) bool {
		return dir == ".arc" || dir == "_schema"
	})
}

// walkGraphFiles collects every *.md file except anything under .arc/ —
// the whole-graph census universe, schema documents included, that
// arc stats counts as nodes (specs/032-arc-stats FR-002, research.md D1).
func walkGraphFiles(store fsys.Store) ([]string, error) {
	return walkFiles(store, func(dir string) bool {
		return dir == ".arc"
	})
}

// Grep enumerates every node file in the graph rooted at dir, narrows the
// content scan to nodes passing filter, and reports every matching line
// across every scanned node (research.md D9). A node file that cannot be
// opened, or that fails to parse as a node at all, is recorded in the
// result's Unreadable list and excluded from the scan (spec Edge Cases).
func Grep(ctx context.Context, mounter fsys.Mounter, filter core.Filter, pattern string, cfg configkernel.GrepConfig, dir string) (kernel.GrepResult, error) {
	store, err := mounter.Mount(dir)
	if err != nil {
		return kernel.GrepResult{}, err
	}

	if err := guardIsGraph(store, dir); err != nil {
		return kernel.GrepResult{}, err
	}

	// Validated up front, before any node file is opened, so an invalid
	// pattern never triggers even the enumeration pass below (spec FR-008).
	if _, err := regexp.Compile(pattern); err != nil {
		return kernel.GrepResult{}, ErrInvalidPattern.With(err, pattern)
	}

	paths, err := walkNodeFiles(store)
	if err != nil {
		return kernel.GrepResult{}, err
	}

	index := map[string]core.Node{}
	included := map[string]bool{}
	var unreadable []string

	for _, p := range paths {
		node, ok, err := readGrepNode(store, p)
		if err != nil || !ok {
			unreadable = append(unreadable, p)
			continue
		}
		index[p] = node
		if filter.Match(node) {
			included[p] = true
		}
	}

	workers := cfg.Workers
	if workers <= 0 {
		workers = defaultGrepWorkers
	}

	scanned, err := grep.Search(ctx, store, pattern, grep.Options{
		Extension: ".md",
		Workers:   workers,
		Include:   func(p string) bool { return included[p] },
	})
	if err != nil {
		return kernel.GrepResult{}, err
	}

	matches := make([]kernel.Match, 0, len(scanned.Matches))
	for _, m := range scanned.Matches {
		node := index[m.Path]
		matches = append(matches, kernel.Match{
			Type:  node.Type,
			ID:    node.ID,
			Path:  m.Path,
			Line:  m.Line,
			Text:  m.Text,
			Start: m.Start,
			End:   m.End,
		})
	}

	unreadable = append(unreadable, scanned.Unreadable...)
	sort.Strings(unreadable)

	return kernel.GrepResult{
		Root:       dir,
		Pattern:    pattern,
		Matches:    matches,
		Unreadable: unreadable,
	}, nil
}

func readGrepNode(store fsys.Store, path string) (core.Node, bool, error) {
	f, err := store.Open(path)
	if err != nil {
		return core.Node{}, false, err
	}
	defer f.Close()

	node, err := core.ParseNode(f, core.Index{})
	if err != nil {
		return core.Node{}, false, err
	}
	return node, true, nil
}
