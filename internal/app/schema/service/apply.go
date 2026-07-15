//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/schema/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/schema/port"
	"github.com/fogfish/arcnet-cli/internal/bios"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// metaIndex is the fixed schema every Property/Class document's own
// predicates (role, merge, label, aligned, description, required,
// optional, subClassOf) merge against — CORE's meta-vocabulary, seeded
// into every graph's _schema/predicates/ by arc init and never varying per
// graph, so it is never resolved from a graph's own dynamic content schema
// (research.md D5/D6, mirroring service/schema.go's Seed).
var metaIndex = core.Index{Predicates: kernel.CorePredicateDefs, Types: kernel.CoreTypeDefs}

const arcnetPrefix = "arcnet:"

// Reporter phase labels (research.md, mirroring graph/service/apply.go's
// own labels).
const (
	labelReadingSchemaPatch = "Reading patch"
	labelApplyingSchema     = "Applying schema nodes"
	labelCommittingSchema   = "Committing"
)

// classifySource resolves source into the URL/local-path it should
// actually be read from (research.md D1/D1a): a literal "arcnet:" prefix
// resolves against kernel.ArcnetCatalogBaseURL; an http(s) URL is used
// as-is; anything else is treated as a local filesystem path.
func classifySource(source string) (resolved string, isURL bool, err error) {
	if suffix, ok := strings.CutPrefix(source, arcnetPrefix); ok {
		if suffix == "" {
			return "", false, ErrEmptyArcnetReference.With(errNoCause)
		}
		return kernel.ArcnetCatalogBaseURL + suffix, true, nil
	}

	if u, perr := url.Parse(source); perr == nil && (u.Scheme == "http" || u.Scheme == "https") {
		return source, true, nil
	}

	return source, false, nil
}

// readPatchSource reads and parses the patch at source, dispatching on
// classifySource's classification to either a local-file open (mirroring
// graph/service/apply.go's readPatch) or a port.Fetcher call. The reported
// source string is always the caller's own original input — never the
// resolved arcnet: URL (research.md D1a, quickstart.md Scenario 5).
func readPatchSource(ctx context.Context, mounter fsys.Mounter, fetcher port.Fetcher, source string) (core.Patch, error) {
	resolved, isURL, err := classifySource(source)
	if err != nil {
		return core.Patch{}, err
	}

	if isURL {
		body, err := fetcher.Fetch(ctx, resolved)
		if err != nil {
			return core.Patch{}, err
		}
		defer body.Close()

		patch, err := core.ParsePatch(body)
		if err != nil {
			return core.Patch{}, ErrPatchRead.With(err, source)
		}
		return patch, nil
	}

	store, err := mounter.Mount(filepath.Dir(resolved))
	if err != nil {
		return core.Patch{}, ErrPatchRead.With(err, source)
	}

	f, err := store.Open(filepath.Base(resolved))
	if err != nil {
		return core.Patch{}, ErrPatchRead.With(err, source)
	}
	defer f.Close()

	patch, err := core.ParsePatch(f)
	if err != nil {
		return core.Patch{}, ErrPatchRead.With(err, source)
	}
	return patch, nil
}

// classifyNodes fails the whole operation the moment any node's type is
// not Property/Class — naming the first offending node — before any
// _schema/ write begins (research.md D4, spec FR-005/FR-006). No rollback
// bookkeeping is needed: this pass performs no I/O of its own.
func classifyNodes(nodes []core.Node) error {
	for _, node := range nodes {
		if node.Type != propertyType && node.Type != classType {
			return ErrDisallowedNodeType.With(errNoCause, node.ID, node.Type)
		}
	}
	return nil
}

// schemaNodePlan is one Property/Class node's fully computed, not-yet-
// written outcome: planSchemaNode does every fallible step (decode
// validation, existing-document read, merge, render) up front so
// ApplyPatch can write every plan only after every node in the patch has
// planned successfully — the same "validate everything, write nothing
// until all succeeds" guarantee D4 established for node-type
// classification, extended to per-node decode/merge validation so a
// failure discovered on node N never leaves nodes 1..N-1 written (spec
// FR-012).
type schemaNodePlan struct {
	kind    string // "predicate" or "type"
	path    string
	raw     []byte
	created bool
	changed bool
}

func planSchemaNode(store fsys.Store, node core.Node, sourceID string) (schemaNodePlan, error) {
	var kind, dir string
	switch node.Type {
	case propertyType:
		kind, dir = "predicate", kernel.PredicatesDir
		if _, invalid := decodePredicateDef(node); invalid != "" {
			return schemaNodePlan{}, ErrSchemaInvalid.With(errNoCause, node.ID, invalid)
		}
	case classType:
		kind, dir = "type", kernel.TypesDir
		if _, invalid := decodeTypeDef(node); invalid != "" {
			return schemaNodePlan{}, ErrSchemaInvalid.With(errNoCause, node.ID, invalid)
		}
	}
	path := dir + "/" + node.ID + ".md"

	existing, existed, err := readExistingSchemaNode(store, path)
	if err != nil {
		return schemaNodePlan{}, err
	}

	final := node
	if existed {
		merged, _, _, merr := core.Merge(existing, node, metaIndex, sourceID)
		if merr != nil {
			return schemaNodePlan{}, ErrSchemaWrite.With(merr, path)
		}
		final = merged
	}

	raw, err := core.RenderNode(final, metaIndex)
	if err != nil {
		return schemaNodePlan{}, ErrSchemaWrite.With(err, path)
	}

	changed := !existed
	if existed {
		existingRaw, err := core.RenderNode(existing, metaIndex)
		if err != nil {
			return schemaNodePlan{}, ErrSchemaWrite.With(err, path)
		}
		changed = !bytes.Equal(existingRaw, raw)
	}

	return schemaNodePlan{kind: kind, path: path, raw: raw, created: !existed, changed: changed}, nil
}

func readExistingSchemaNode(store fsys.Store, path string) (core.Node, bool, error) {
	f, err := store.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return core.Node{}, false, nil
		}
		return core.Node{}, false, ErrSchemaInvalid.With(err, path, "document")
	}
	defer f.Close()

	node, err := core.ParseNode(f)
	if err != nil {
		return core.Node{}, false, ErrSchemaInvalid.With(err, path, "document")
	}
	return node, true, nil
}

func writeSchemaNode(store fsys.Store, plan schemaNodePlan) error {
	f, err := store.Create(plan.path)
	if err != nil {
		return ErrSchemaWrite.With(err, plan.path)
	}
	if _, err := f.Write(plan.raw); err != nil {
		_ = f.Discard()
		return ErrSchemaWrite.With(err, plan.path)
	}
	if err := f.Close(); err != nil {
		return ErrSchemaWrite.With(err, plan.path)
	}
	return nil
}

// buildApplySchemaCommitMessage mirrors graph/service/apply.go's
// buildCommitMessage shape, scoped to a "<n> predicate(s), <m> type(s)"
// subject (research.md, tasks.md T026).
func buildApplySchemaCommitMessage(result kernel.ApplySchemaResult) string {
	var parts []string
	for _, kind := range []string{"predicate", "type"} {
		n := result.Created[kind] + result.Merged[kind]
		if n == 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%d %s(s)", n, kind))
	}
	return fmt.Sprintf("schema(apply): %s", strings.Join(parts, ", "))
}

// ApplyPatch mounts dir, reads the patch at source (a local path, URL, or
// arcnet: reference, research.md D1/D1a), validates every node section is
// Property/Class (spec FR-004/FR-005/FR-006), then creates or merges each
// one into _schema/predicates/<name>.md or _schema/types/<name>.md,
// producing exactly one commit — or none at all when the patch is a no-op
// re-apply (research.md D7). Any failure — a disallowed node type, a
// malformed Property/Class document, a fetch/read failure — leaves
// _schema/ byte-for-byte unchanged (spec FR-012): every node is validated
// and rendered (schemaNodePlan) before any write begins.
func ApplyPatch(ctx context.Context, mounter fsys.Mounter, vcs port.VCS, fetcher port.Fetcher, reporter bios.Reporter, dir, source string) (kernel.ApplySchemaResult, error) {
	store, err := mounter.Mount(dir)
	if err != nil {
		return kernel.ApplySchemaResult{}, err
	}
	if err := guardIsGraph(store, dir); err != nil {
		return kernel.ApplySchemaResult{}, err
	}

	start := time.Now()
	patch, err := readPatchSource(ctx, mounter, fetcher, source)
	if err != nil {
		reporter.Error(labelReadingSchemaPatch, err)
		return kernel.ApplySchemaResult{}, err
	}
	reporter.Done(labelReadingSchemaPatch, time.Since(start))

	if err := classifyNodes(patch.Nodes); err != nil {
		reporter.Error(labelApplyingSchema, err)
		return kernel.ApplySchemaResult{}, err
	}

	start = time.Now()
	plans := make([]schemaNodePlan, 0, len(patch.Nodes))
	for _, node := range patch.Nodes {
		plan, err := planSchemaNode(store, node, patch.Document)
		if err != nil {
			reporter.Error(labelApplyingSchema, err)
			return kernel.ApplySchemaResult{}, err
		}
		plans = append(plans, plan)
	}

	result := kernel.ApplySchemaResult{
		Source:  source,
		Created: map[string]int{},
		Merged:  map[string]int{},
	}

	anyChanged := false
	for i, plan := range plans {
		outcome := "unchanged"
		switch {
		case plan.created:
			result.Created[plan.kind]++
			outcome = "created"
		case plan.changed:
			result.Merged[plan.kind]++
			outcome = "merged"
		}

		if plan.created || plan.changed {
			anyChanged = true
			if err := writeSchemaNode(store, plan); err != nil {
				reporter.Error(labelApplyingSchema, err)
				return kernel.ApplySchemaResult{}, err
			}
		}

		reporter.Step(fmt.Sprintf("%s: %s", patch.Nodes[i].ID, outcome))
	}
	reporter.Done(labelApplyingSchema, time.Since(start))

	if !anyChanged {
		return result, nil
	}

	start = time.Now()
	if err := vcs.StageAll(ctx, dir); err != nil {
		reporter.Error(labelCommittingSchema, err)
		return kernel.ApplySchemaResult{}, err
	}

	hash, err := vcs.Commit(ctx, dir, buildApplySchemaCommitMessage(result))
	if err != nil {
		reporter.Error(labelCommittingSchema, err)
		return kernel.ApplySchemaResult{}, err
	}
	reporter.Done(labelCommittingSchema, time.Since(start))
	result.CommitHash = hash

	return result, nil
}

func guardIsGraph(store fsys.Store, dir string) error {
	if _, err := store.Stat(".arc"); err != nil {
		return ErrNotAGraph.With(err)
	}
	return nil
}
