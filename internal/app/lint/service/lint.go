//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package service implements the lint use-case's business logic.
package service

import (
	"bytes"
	"context"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/lint/port"
	"github.com/fogfish/arcnet-cli/internal/bios"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// Reporter phase labels (data-model.md Reporter events).
const (
	labelReadingGraph       = "Reading graph"
	labelCheckingBasenames  = "Checking basenames and links"
	labelCheckingPredicates = "Checking predicates and citations"
	labelCheckingHistory    = "Checking commit history"
)

// parsedNode is one successfully-parsed node file, carrying both its
// structural core.Node and its raw bytes (for line-locating violations).
type parsedNode struct {
	Path     string
	Basename string
	Node     core.Node
	Raw      []byte
}

// Lint mounts dir, walks every node file, and checks it against the full
// CORE §14 conformance checklist, never stopping at the first violation
// found (spec FR-013). It never writes to the graph (spec FR-014).
func Lint(ctx context.Context, mounter fsys.Mounter, vcs port.VCS, reporter bios.Reporter, index core.Index, dir string) (kernel.LintResult, error) {
	store, err := mounter.Mount(dir)
	if err != nil {
		return kernel.LintResult{}, err
	}

	if err := guardIsGraph(store, dir); err != nil {
		return kernel.LintResult{}, err
	}

	start := time.Now()
	paths, err := walkNodeFiles(store)
	if err != nil {
		reporter.Error(labelReadingGraph, err)
		return kernel.LintResult{}, err
	}

	fileViolations := map[string][]kernel.Violation{}
	basenameIndex := map[string][]string{}
	var parsed []parsedNode

	for _, path := range paths {
		raw, err := readRaw(store, path)
		if err != nil {
			reporter.Error(labelReadingGraph, err)
			return kernel.LintResult{}, err
		}

		basename := basenameOf(path)
		basenameIndex[basename] = append(basenameIndex[basename], path)

		if line := locateConflictMarker(raw); line > 0 {
			fileViolations[path] = append(fileViolations[path], kernel.Violation{
				Rule: kernel.RuleMergeConflict, Path: path, Line: line,
				Message: "unresolved git merge-conflict marker found",
			})
			continue
		}

		node, err := core.ParseNode(bytes.NewReader(raw), index)
		if err != nil {
			if quoting := checkBareIdentityKeys(path, raw); len(quoting) > 0 {
				fileViolations[path] = append(fileViolations[path], quoting...)
				continue
			}
			fileViolations[path] = append(fileViolations[path], kernel.Violation{
				Rule: kernel.RuleFrontMatter, Path: path, Line: locateFrontMatterDelimiter(raw),
				Message: err.Error(),
			})
			continue
		}

		// ParseNode has no filename parameter (contracts/ast-contract.md), so
		// the "@id" == basename rule (spec FR-002, US3 Acceptance Scenario 3)
		// is enforced here, by the one caller that knows the file's actual
		// path — universally, for every node, not just "source"-kind ones
		// (checkSourceCitekey's own "@id"==basename check is a narrower,
		// pre-existing CORE §11 rule specific to a source's citekey).
		if node.ID != basename {
			fileViolations[path] = append(fileViolations[path], kernel.Violation{
				Rule: kernel.RuleFrontMatter, Path: path, Line: locateFrontMatterField(raw, `"@id"`),
				Message: `"@id" ` + node.ID + ` does not match this file's basename ` + basename,
			})
			continue
		}

		parsed = append(parsed, parsedNode{Path: path, Basename: basename, Node: node, Raw: raw})
	}
	reporter.Done(labelReadingGraph, time.Since(start))

	kindIndex := map[string]string{}
	for _, p := range parsed {
		if _, ok := kindIndex[p.Basename]; !ok {
			kindIndex[p.Basename] = p.Node.Type
		}
	}

	start = time.Now()
	graphSpanning := checkUniqueBasenames(basenameIndex)
	graphSpanning = append(graphSpanning, checkSchemaTypeCase(index)...)
	for _, p := range parsed {
		fileViolations[p.Path] = append(fileViolations[p.Path], checkUnrecognizedKind(p.Node, p.Path, index)...)
		fileViolations[p.Path] = append(fileViolations[p.Path], checkLinksResolve(p.Node, p.Path, p.Raw, basenameIndex)...)
		fileViolations[p.Path] = append(fileViolations[p.Path], checkDerivedProvenance(p.Node, p.Path, kindIndex)...)
		fileViolations[p.Path] = append(fileViolations[p.Path], checkSourceCitekey(p.Node, p.Path, p.Basename, p.Raw)...)
		fileViolations[p.Path] = append(fileViolations[p.Path], checkEntityCategory(p.Node, p.Path, p.Raw)...)
	}
	reporter.Done(labelCheckingBasenames, time.Since(start))

	start = time.Now()
	for _, p := range parsed {
		fileViolations[p.Path] = append(fileViolations[p.Path], checkPredicateCase(p.Node, p.Path, p.Raw)...)
		fileViolations[p.Path] = append(fileViolations[p.Path], checkNodeTypeCase(p.Node, p.Path)...)
		fileViolations[p.Path] = append(fileViolations[p.Path], checkPredicateRegistered(p.Node, p.Path, p.Raw, index.Predicates)...)
		fileViolations[p.Path] = append(fileViolations[p.Path], checkCitationPredicate(p.Node, p.Path, p.Raw, index.Predicates)...)
		fileViolations[p.Path] = append(fileViolations[p.Path], checkTypeRequires(p.Node, p.Path, p.Raw, index)...)
		fileViolations[p.Path] = append(fileViolations[p.Path], checkTypeOptional(p.Node, p.Path, p.Raw, index)...)
		fileViolations[p.Path] = append(fileViolations[p.Path], checkIdentityKeyQuoting(p.Node, p.Path, p.Raw)...)
		fileViolations[p.Path] = append(fileViolations[p.Path], checkPredicateRole(p.Node, p.Path, p.Raw, index)...)
	}
	reporter.Done(labelCheckingPredicates, time.Since(start))

	start = time.Now()
	for _, p := range parsed {
		violations, err := checkIngestCommit(ctx, vcs, dir, p.Node, p.Path)
		if err != nil {
			reporter.Error(labelCheckingHistory, err)
			return kernel.LintResult{}, err
		}
		fileViolations[p.Path] = append(fileViolations[p.Path], violations...)
	}
	reporter.Done(labelCheckingHistory, time.Since(start))

	nodes := make([]kernel.NodeStatus, 0, len(paths))
	for _, path := range paths {
		status := kernel.NodeStatus{Path: path, Violations: fileViolations[path]}
		for _, p := range parsed {
			if p.Path == path {
				status.ID = p.Node.ID
				status.Type = p.Node.Type
				break
			}
		}
		nodes = append(nodes, status)
	}

	return kernel.NewLintResult(dir, nodes, graphSpanning...), nil
}

func guardIsGraph(store fsys.Store, dir string) error {
	if _, err := store.Stat(".arc"); err != nil {
		return ErrNotAGraph.With(err, dir)
	}
	return nil
}

func readRaw(store fsys.Store, path string) ([]byte, error) {
	f, err := store.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func basenameOf(path string) string {
	name := path
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		name = path[idx+1:]
	}
	return strings.TrimSuffix(name, ".md")
}

// walkNodeFiles recursively walks store from its root, collecting every
// *.md file except anything under .arc/ or _schema/ (research.md D6,
// spec.md FR-015, Clarifications Q1/Q3 — schema documents are exempt from
// ordinary content rules and never enter the basename-uniqueness index), in
// deterministic (sorted) order.
func walkNodeFiles(store fsys.Store) ([]string, error) {
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
				if full == ".arc" || full == "_schema" {
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
