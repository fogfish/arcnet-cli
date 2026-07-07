//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package service implements the graph use-case's business logic.
package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/graph/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/graph/port"
	"github.com/fogfish/arcnet-cli/internal/bios"
	"github.com/fogfish/arcnet-cli/internal/core"
)

var coreKindFolders = map[string]string{
	"source":   "sources",
	"entity":   "entities",
	"resource": "resources",
}

// nodeFolder is never called with kind == "timeline": Apply's per-node loop
// intercepts a patch-carried "timeline"-kind section before this function
// is ever consulted, folding it into applyTimeline's own two-tier
// timeline/yearly|monthly layout instead (research.md D8b revised,
// BUG-005/BUG-006) — timeline period files use applyTimeline's own
// specialized bullet-list rendering, not core.RenderNode's generic
// serialization, so this generic per-kind folder derivation could never
// produce a compatible file for that kind even if it recognized it.
func nodeFolder(kind string) string {
	if folder, ok := coreKindFolders[kind]; ok {
		return folder
	}
	if strings.HasSuffix(kind, "s") {
		return kind
	}
	return kind + "s"
}

func nodePath(node core.Node) string {
	return nodeFolder(node.Type) + "/" + node.ID + ".md"
}

// distinctPredicates collects every distinct predicate name node declares:
// every non-empty Link.Predicate in Edges (research.md D4, D5 — Edges is
// now the single unioned collection, what used to be Edges+Links). HRefs
// are excluded — those are citation-type predicates, a separate vocabulary
// this feature does not seed.
func distinctPredicates(node core.Node) []string {
	seen := map[string]bool{}
	var out []string
	add := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		out = append(out, name)
	}

	for _, l := range node.Edges {
		add(l.Predicate)
	}
	return out
}

// isStub reports whether node carries no content beyond ID/Type — the
// exact zero-beyond-ID/Type shape service/subgraph.go's --stubs flag
// already emits (spec 007 FR-017, research.md D7). A stub-shaped node
// created by service.Apply gets neither Published nor an "indexed" stamp.
func isStub(node core.Node) bool {
	return len(node.Attrs) == 0 && len(node.Texts) == 0 &&
		len(node.HRefs) == 0 && len(node.Edges) == 0
}

// nodeContentChanged reports whether merged's rendered content actually
// differs, byte-for-byte, from existing's (research.md D6) — the single
// mechanism deciding whether a merge earns an "updated" stamp, correct for
// every core.MergeOp uniformly including MergeNone's already-a-no-op case.
func nodeContentChanged(existing, merged core.Node) (bool, error) {
	existingRaw, err := core.RenderNode(existing)
	if err != nil {
		return false, err
	}
	mergedRaw, err := core.RenderNode(merged)
	if err != nil {
		return false, err
	}
	return !bytes.Equal(existingRaw, mergedRaw), nil
}

// setAttr nil-safely sets key on attrs to a single-valued Predicate,
// allocating it if nil, and returns it — used to stamp both "indexed" and
// "updated".
func setAttr(attrs map[string][]core.Predicate, key string, value any) map[string][]core.Predicate {
	if attrs == nil {
		attrs = map[string][]core.Predicate{}
	}
	attrs[key] = []core.Predicate{{Value: value}}
	return attrs
}

// attrString returns node's single-valued key attribute as a string, or ""
// when absent/not a string — used to read back "title"/"authors"-shaped
// scalars applyTimeline needs.
func attrString(node core.Node, key string) string {
	preds := node.Attrs[key]
	if len(preds) == 0 {
		return ""
	}
	s, _ := preds[0].Value.(string)
	return s
}

// Reporter phase labels (data-model.md Reporter events, research.md D9,
// BUG-001 revised).
const (
	labelReadingPatch  = "Reading patch file"
	labelIdempotency   = "Checking idempotency"
	labelApplyingNodes = "Applying node contributions"
	labelUpdatingTL    = "Updating timeline"
	labelCommitting    = "Committing"
)

// Apply mounts dir, parses the patch at patchPath, and — unless the
// document is already tracked (FR-003) — creates or merges every node
// section it carries per rules, updates the derived timeline, and produces
// exactly one commit (CORE §11.3). Any failure before the commit leaves no
// newly-created node file behind (FR-015, bounded to genuinely new files —
// a pre-existing node overwritten by an in-progress merge is left at its
// last-written state, recoverable via git, mirroring internal/app/ctrl's
// own bounded rollback precedent). Progress is reported through reporter
// (silent by default, --verbose-gated by the caller — BUG-001): one
// Start/Done pair per phase, plus one Reporter.Step per node processed
// inside "Applying node contributions", naming the node's ID and outcome
// (spec.md FR-021).
func Apply(ctx context.Context, mounter fsys.Mounter, vcs port.VCS, reporter bios.Reporter, rules core.MergeRuleSet, predicates map[string]bool, schema port.SchemaRegistry, dir, patchPath string) (kernel.ApplyResult, error) {
	store, err := mounter.Mount(dir)
	if err != nil {
		return kernel.ApplyResult{}, err
	}

	if err := guardIsGraph(store, dir); err != nil {
		return kernel.ApplyResult{}, err
	}

	if err := guardNoOldFormatNodes(store); err != nil {
		return kernel.ApplyResult{}, err
	}

	start := time.Now()
	appliedAt := time.Now().UTC()
	stamp := appliedAt.Format(time.RFC3339)
	patch, err := readPatch(mounter, patchPath)
	if err != nil {
		reporter.Error(labelReadingPatch, err)
		return kernel.ApplyResult{}, err
	}
	reporter.Done(labelReadingPatch, time.Since(start))

	start = time.Now()
	sourcePath := nodeFolder("source") + "/" + patch.Document + ".md"
	tracked, err := vcs.IsTracked(ctx, dir, sourcePath)
	if err != nil {
		reporter.Error(labelIdempotency, err)
		return kernel.ApplyResult{}, err
	}
	reporter.Done(labelIdempotency, time.Since(start))
	if tracked {
		return kernel.ApplyResult{Document: patch.Document, Skipped: true}, nil
	}

	result := kernel.ApplyResult{
		Document:  patch.Document,
		Created:   map[string]int{},
		Merged:    map[string]int{},
		Conflicts: []string{},
		Warnings:  []string{},
		Timeline:  []string{},
	}

	var createdPaths []string
	var sourceNode core.Node
	var timelinePeriodsFromPatch []string

	start = time.Now()
	for _, node := range patch.Nodes {
		// timeline is a format-reserved index kind (CORE §12.2); a real
		// extraction pipeline may still emit one explicitly alongside a
		// document's own contribution (e.g. as a self-describing period
		// annotation) — it is never written as a generic node file, both
		// because applyTimeline's period files use its own specialized
		// bullet-list rendering (incompatible with core.RenderNode's
		// generic serialization) and because the plain per-kind folder
		// derivation below has no way to reproduce applyTimeline's
		// yearly/monthly layout. Its declared period id is instead folded
		// into applyTimeline's own derivation further below (research.md
		// D8b revised, BUG-005/BUG-006).
		if node.Type == "timeline" {
			timelinePeriodsFromPatch = append(timelinePeriodsFromPatch, node.ID)
			reporter.Step(fmt.Sprintf("%s: folded into timeline index", node.ID))
			continue
		}

		op, ok := rules.Lookup(node.Type)
		if !ok {
			op = core.MergeUnion
			result.Warnings = append(result.Warnings, fmt.Sprintf(
				"%s is not a recognized node kind for this graph — applied using the default \"union\" merge behavior", node.Type))
			if _, err := schema.RegisterKind(store, node.Type); err != nil {
				reporter.Error(labelApplyingNodes, err)
				rollback(store, createdPaths)
				return kernel.ApplyResult{}, err
			}
		}

		path := nodePath(node)

		existing, existed, err := readExistingNode(store, path)
		if err != nil {
			reporter.Error(labelApplyingNodes, err)
			rollback(store, createdPaths)
			return kernel.ApplyResult{}, err
		}

		merged := node
		outcome := "created"
		if existed {
			var conflicts []string
			merged, conflicts, err = core.Merge(existing, node, op, patch.Document)
			if err != nil {
				reporter.Error(labelApplyingNodes, err)
				rollback(store, createdPaths)
				return kernel.ApplyResult{}, err
			}
			result.Merged[node.Type]++
			outcome = "merged"
			if len(conflicts) > 0 {
				result.Conflicts = append(result.Conflicts, path)
				outcome = "merged (conflict flagged)"
			}

			changed, err := nodeContentChanged(existing, merged)
			if err != nil {
				reporter.Error(labelApplyingNodes, err)
				rollback(store, createdPaths)
				return kernel.ApplyResult{}, err
			}
			if changed {
				merged.Attrs = setAttr(merged.Attrs, "updated", stamp)
			}
		} else {
			result.Created[node.Type]++
			if !isStub(node) {
				if merged.Published.IsZero() {
					merged.Published = patch.Published
				}
				merged.Attrs = setAttr(merged.Attrs, "indexed", stamp)
			}
		}

		for _, name := range distinctPredicates(merged) {
			if predicates[name] {
				continue
			}
			if _, err := schema.RegisterPredicate(store, name); err != nil {
				reporter.Error(labelApplyingNodes, err)
				rollback(store, createdPaths)
				return kernel.ApplyResult{}, err
			}
		}

		if err := writeNode(store, path, merged); err != nil {
			reporter.Error(labelApplyingNodes, err)
			rollback(store, createdPaths)
			return kernel.ApplyResult{}, err
		}
		if !existed {
			createdPaths = append(createdPaths, path)
		}

		reporter.Step(fmt.Sprintf("%s: %s", node.ID, outcome))

		if node.Type == "source" && node.ID == patch.Document {
			sourceNode = node
		}
	}
	reporter.Done(labelApplyingNodes, time.Since(start))

	start = time.Now()
	timeline, err := applyTimeline(store, patch, sourceNode, timelinePeriodsFromPatch)
	if err != nil {
		reporter.Error(labelUpdatingTL, err)
		rollback(store, createdPaths)
		return kernel.ApplyResult{}, err
	}
	reporter.Done(labelUpdatingTL, time.Since(start))
	result.Timeline = timeline

	start = time.Now()
	if err := vcs.StageAll(ctx, dir); err != nil {
		reporter.Error(labelCommitting, err)
		rollback(store, createdPaths)
		return kernel.ApplyResult{}, err
	}

	hash, err := vcs.Commit(ctx, dir, buildCommitMessage(patch, result))
	if err != nil {
		reporter.Error(labelCommitting, err)
		rollback(store, createdPaths)
		return kernel.ApplyResult{}, err
	}
	reporter.Done(labelCommitting, time.Since(start))
	result.CommitHash = hash

	return result, nil
}

func guardIsGraph(store fsys.Store, dir string) error {
	if _, err := store.Stat(".arc"); err != nil {
		return ErrNotAGraph.With(err, dir)
	}
	return nil
}

// validateNodeBasename enforces spec FR-002/US3 Acceptance Scenario 3:
// core.ParseNode has no filename parameter (contracts/ast-contract.md), so
// the "@id" == basename rule is checked here, by callers that know path.
func validateNodeBasename(node core.Node, path string) error {
	basename := strings.TrimSuffix(filepath.Base(path), ".md")
	if node.ID != basename {
		return fmt.Errorf("%q does not match this file's basename %q", node.ID, basename)
	}
	return nil
}

// guardNoOldFormatNodes scans every existing node file in the graph before
// any write begins (spec FR-012/FR-013, US3 Acceptance Scenario 4,
// quickstart.md Scenario 3): a single pre-0.5 file anywhere in the graph —
// not just one the incoming patch happens to target — aborts the whole
// apply with no partial write, mirroring arc lint's own whole-graph walk
// (internal/app/lint/service.Lint). A well-formed patch/exchange document
// (e.g. one written into the graph root before being applied, this
// package's own writePatchFile-style convention) is not a graph node and
// is skipped — its own "kind: patch" manifest is a distinct, still-valid
// concept unaffected by this feature (data-model.md's Patch section).
func guardNoOldFormatNodes(store fsys.Store) error {
	paths, err := walkNodeFiles(store)
	if err != nil {
		return err
	}
	for _, path := range paths {
		f, err := store.Open(path)
		if err != nil {
			continue
		}
		raw, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			continue
		}

		if _, perr := core.ParsePatch(bytes.NewReader(raw)); perr == nil {
			continue
		}

		node, parseErr := core.ParseNode(bytes.NewReader(raw))
		if parseErr != nil {
			return ErrNodeWrite.With(parseErr, path)
		}
		if err := validateNodeBasename(node, path); err != nil {
			return ErrNodeWrite.With(err, path)
		}
	}
	return nil
}

// readPatch mounts a store rooted at patchPath's own containing directory,
// rather than reading it through the graph-rooted store — a patch is a
// parallel exchange file, never part of the graph itself (CORE §12.1), so
// it may live anywhere on disk, including outside dir's tree, which an
// fs.FS scoped to dir could never reach (fs.FS forbids both absolute paths
// and ".." traversal).
func readPatch(mounter fsys.Mounter, patchPath string) (core.Patch, error) {
	store, err := mounter.Mount(filepath.Dir(patchPath))
	if err != nil {
		return core.Patch{}, ErrPatchRead.With(err, patchPath)
	}

	f, err := store.Open(filepath.Base(patchPath))
	if err != nil {
		return core.Patch{}, ErrPatchRead.With(err, patchPath)
	}
	defer f.Close()

	patch, err := core.ParsePatch(f)
	if err != nil {
		return core.Patch{}, err
	}
	return patch, nil
}

func readExistingNode(store fsys.Store, path string) (core.Node, bool, error) {
	f, err := store.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return core.Node{}, false, nil
		}
		return core.Node{}, false, ErrNodeWrite.With(err, path)
	}
	defer f.Close()

	node, err := core.ParseNode(f)
	if err != nil {
		return core.Node{}, false, ErrNodeWrite.With(err, path)
	}
	if err := validateNodeBasename(node, path); err != nil {
		return core.Node{}, false, ErrNodeWrite.With(err, path)
	}

	return node, true, nil
}

func writeNode(store fsys.Store, path string, node core.Node) error {
	raw, err := core.RenderNode(node)
	if err != nil {
		return ErrNodeWrite.With(err, path)
	}
	return writeRaw(store, path, raw)
}

func writeRaw(store fsys.Store, path string, content []byte) error {
	f, err := store.Create(path)
	if err != nil {
		return ErrNodeWrite.With(err, path)
	}
	if _, err := f.Write(content); err != nil {
		_ = f.Discard()
		return ErrNodeWrite.With(err, path)
	}
	if err := f.Close(); err != nil {
		return ErrNodeWrite.With(err, path)
	}
	return nil
}

func rollback(store fsys.Store, paths []string) {
	for _, p := range paths {
		_ = store.Remove(p)
	}
}

func buildCommitMessage(patch core.Patch, result kernel.ApplyResult) string {
	subject := fmt.Sprintf("graph(ingest): %s — %s", patch.Document, patch.Title)

	kinds := make([]string, 0, len(result.Created)+len(result.Merged))
	seen := map[string]bool{}
	for k := range result.Created {
		if !seen[k] {
			kinds = append(kinds, k)
			seen[k] = true
		}
	}
	for k := range result.Merged {
		if !seen[k] {
			kinds = append(kinds, k)
			seen[k] = true
		}
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })

	stats := make([]string, 0, len(kinds))
	for _, k := range kinds {
		stats = append(stats, fmt.Sprintf("%s: +%d created, +%d merged", k, result.Created[k], result.Merged[k]))
	}

	var buf strings.Builder
	buf.WriteString(subject)
	buf.WriteString("\n\n")
	buf.WriteString("Nodes: " + strings.Join(stats, ", ") + "\n")
	if len(result.Timeline) > 0 {
		buf.WriteString("Timeline: " + strings.Join(result.Timeline, ", ") + "\n")
	}
	buf.WriteString("Source-Id: " + patch.Document + "\n")

	return buf.String()
}

type timelineEntry struct {
	id        string
	published time.Time
	line      string
}

var timelineEntryPattern = regexp.MustCompile(`^- \[\[([^\]]+)\]\].* — (\d{4}-\d{2}-\d{2})$`)

func parseTimelineEntries(content string) []timelineEntry {
	var out []timelineEntry
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimRight(line, "\r")
		m := timelineEntryPattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		t, err := time.Parse("2006-01-02", m[2])
		if err != nil {
			continue
		}
		out = append(out, timelineEntry{id: m[1], published: t, line: line})
	}
	return out
}

// attrStringSlice returns every Predicate's Value in preds, stringified —
// used to read back a multi-valued scalar attribute like "authors".
func attrStringSlice(preds []core.Predicate) []string {
	if len(preds) == 0 {
		return nil
	}
	out := make([]string, 0, len(preds))
	for _, p := range preds {
		out = append(out, fmt.Sprint(p.Value))
	}
	return out
}

var (
	monthlyPeriodPattern = regexp.MustCompile(`^\d{4}-\d{2}$`)
	yearlyPeriodPattern  = regexp.MustCompile(`^\d{4}$`)
)

// periodGranularity reports the on-disk path, front-matter granularity,
// and heading for period (CORE §9.4: monthly periods are "YYYY-MM", yearly
// periods are "YYYY"). ok is false when period matches neither shape.
func periodGranularity(period string) (path, granularity, heading string, ok bool) {
	if monthlyPeriodPattern.MatchString(period) {
		t, err := time.Parse("2006-01", period)
		if err != nil {
			return "", "", "", false
		}
		return "timeline/monthly/" + period + ".md", "monthly", t.Format("January 2006"), true
	}
	if yearlyPeriodPattern.MatchString(period) {
		return "timeline/yearly/" + period + ".md", "yearly", period, true
	}
	return "", "", "", false
}

// applyTimeline derives the yearly/monthly period files a patch's
// published date touches, plus any additional period a patch-carried
// "timeline"-kind section itself declares (extraPeriods — research.md D8b
// revised, BUG-005/BUG-006), and inserts one chronologically-ordered entry
// into each, creating the period file if absent (CORE §9.4, research.md
// D8). A monthly-only extra period MUST also touch its yearly rollup
// (e.g. "2026-07" implies "2026"), so a partial declaration never leaves
// the yearly index out of sync with the monthly one. Re-inserting an
// already-present document id is a no-op (CORE §10 "append" — keyed for
// uniqueness), so declaring a period that coincides with the one already
// derived from patch.Published is harmless.
func applyTimeline(store fsys.Store, patch core.Patch, source core.Node, extraPeriods []string) ([]string, error) {
	yearly, monthly := core.TimelinePeriods(patch.Published)

	title := patch.Title
	if title == "" {
		title = attrString(source, "title")
	}
	authors := attrStringSlice(source.Attrs["authors"])

	entry := timelineEntry{
		id:        patch.Document,
		published: patch.Published,
		line:      core.TimelineEntry(patch.Document, title, authors, patch.Published),
	}

	touched := []string{yearly, monthly}
	seen := map[string]bool{yearly: true, monthly: true}

	var extras []string
	addExtra := func(p string) {
		if !seen[p] {
			extras = append(extras, p)
			seen[p] = true
		}
	}
	for _, p := range extraPeriods {
		switch {
		case monthlyPeriodPattern.MatchString(p):
			addExtra(p)
			addExtra(p[:4]) // cascade: a monthly-only declaration also touches its yearly rollup
		case yearlyPeriodPattern.MatchString(p):
			addExtra(p)
		}
	}
	sort.Strings(extras)
	touched = append(touched, extras...)

	for _, period := range touched {
		path, granularity, heading, ok := periodGranularity(period)
		if !ok {
			continue
		}
		if err := upsertTimelinePeriod(store, path, period, granularity, heading, entry); err != nil {
			return nil, err
		}
	}

	return touched, nil
}

func upsertTimelinePeriod(store fsys.Store, path, period, granularity, heading string, newEntry timelineEntry) error {
	existing, err := readFileIfExists(store, path)
	if err != nil {
		return err
	}

	entries := parseTimelineEntries(existing)
	for _, e := range entries {
		if e.id == newEntry.id {
			return nil
		}
	}

	insertAt := len(entries)
	for i, e := range entries {
		if newEntry.published.Before(e.published) {
			insertAt = i
			break
		}
	}
	entries = append(entries, timelineEntry{})
	copy(entries[insertAt+1:], entries[insertAt:])
	entries[insertAt] = newEntry

	var buf strings.Builder
	buf.WriteString("---\n")
	// "@id"/"@type" (not the old "kind" field) so this file satisfies
	// internal/core.ParseNode's mandatory-identity rule for any generic
	// reader (lint's own walk parses every *.md file, timeline period
	// files included) — "@id" is period itself, which is exactly this
	// file's own basename (research.md D7). period is additionally kept
	// as its own quoted Attrs entry so it always decodes as a YAML string
	// — a bare 4-digit yearly value (e.g. "2026") would otherwise decode
	// as an integer (research.md D8 Bugfix, BUG-007).
	buf.WriteString("\"@id\": \"" + period + "\"\n")
	buf.WriteString("\"@type\": timeline\n")
	buf.WriteString("period: \"" + period + "\"\n")
	buf.WriteString("granularity: " + granularity + "\n")
	buf.WriteString("---\n")
	buf.WriteString("# " + heading + "\n\n")
	for _, e := range entries {
		buf.WriteString(e.line)
		buf.WriteString("\n")
	}

	return writeRaw(store, path, []byte(buf.String()))
}

func readFileIfExists(store fsys.Store, path string) (string, error) {
	f, err := store.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	defer f.Close()

	raw, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
