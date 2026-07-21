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
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/graph/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/graph/port"
	"github.com/fogfish/arcnet-cli/internal/bios"
	"github.com/fogfish/arcnet-cli/internal/core"
)

const (
	labelLocatingIngestCommit  = "Locating ingest commit"
	labelEvaluatingEligibility = "Evaluating revert eligibility"
	labelReverting             = "Reverting"
	labelReconciling           = "Reconciling nodes"
)

// Revert locates sourceID's ingest commit (research.md D1), and — unless
// the source node no longer exists (D2, FR-003) — retracts that patch's
// contribution from the graph via whichever of the two reconciliation
// approaches (D3/D4 whole-commit, or D5-D9 per-node) the ingest commit's
// current eligibility calls for, producing exactly one commit. Any
// failure before that commit leaves no removed node file and no rewritten
// node content behind (FR-016).
func Revert(ctx context.Context, mounter fsys.Mounter, vcs port.VCS, reporter bios.Reporter, index core.Index, dir, sourceID string) (kernel.RevertResult, error) {
	store, err := mounter.Mount(dir)
	if err != nil {
		return kernel.RevertResult{}, err
	}

	if err := guardIsGraph(store, dir); err != nil {
		return kernel.RevertResult{}, err
	}

	start := time.Now()
	hashes, err := vcs.CommitsMatching(ctx, dir, "Source-Id: "+sourceID)
	if err != nil {
		reporter.Error(labelLocatingIngestCommit, err)
		return kernel.RevertResult{}, err
	}
	if len(hashes) == 0 {
		err := ErrNoIngestCommit.With(errNoCause, sourceID)
		reporter.Error(labelLocatingIngestCommit, err)
		return kernel.RevertResult{}, err
	}
	// More than one match is the expected result of a prior
	// retract-then-reapply cycle for sourceID, not an integrity anomaly
	// (research.md D1, corrected — BUG-001): arc apply's own idempotency
	// check guarantees at most one ingest commit's contribution can ever
	// be active at a time, so every match older than the newest must
	// already have been fully retracted before the newer one was
	// created. CommitsMatching's own git log --all ordering is
	// newest-first, so hashes[0] is always the currently active ingest
	// commit to act on.
	ingestHash := hashes[0]
	reporter.Done(labelLocatingIngestCommit, time.Since(start))

	start = time.Now()
	sourcePath := nodeFolder("Source") + "/" + sourceID + ".md"
	tracked, err := vcs.IsTracked(ctx, dir, sourcePath)
	if err != nil {
		reporter.Error(labelIdempotency, err)
		return kernel.RevertResult{}, err
	}
	reporter.Done(labelIdempotency, time.Since(start))
	if !tracked {
		return kernel.RevertResult{Document: sourceID, Skipped: true}, nil
	}

	start = time.Now()
	paths, err := vcs.ChangedPaths(ctx, dir, ingestHash)
	if err != nil {
		reporter.Error(labelEvaluatingEligibility, err)
		return kernel.RevertResult{}, err
	}

	eligible := true
	touchingByPath := make(map[string][]string, len(paths))
	for _, p := range paths {
		touching, err := vcs.CommitsTouching(ctx, dir, p)
		if err != nil {
			reporter.Error(labelEvaluatingEligibility, err)
			return kernel.RevertResult{}, err
		}
		touchingByPath[p] = touching
		if len(touching) == 0 || touching[0] != ingestHash {
			eligible = false
		}
	}
	reporter.Done(labelEvaluatingEligibility, time.Since(start))

	if eligible {
		start = time.Now()
		newHash, err := vcs.RevertCommit(ctx, dir, ingestHash)
		if err != nil {
			reporter.Error(labelReverting, err)
			return kernel.RevertResult{}, err
		}
		reporter.Done(labelReverting, time.Since(start))
		return kernel.RevertResult{
			Document:   sourceID,
			Approach:   "whole-commit",
			CommitHash: newHash,
		}, nil
	}

	start = time.Now()
	result, err := revertPerNode(ctx, store, vcs, reporter, index, dir, sourceID, ingestHash, paths, touchingByPath)
	if err != nil {
		reporter.Error(labelReconciling, err)
		return kernel.RevertResult{}, err
	}
	reporter.Done(labelReconciling, time.Since(start))

	start = time.Now()
	if err := vcs.StageAll(ctx, dir); err != nil {
		reporter.Error(labelCommitting, err)
		return kernel.RevertResult{}, err
	}
	hash, err := vcs.Commit(ctx, dir, buildRevertCommitMessage(sourceID, result))
	if err != nil {
		reporter.Error(labelCommitting, err)
		return kernel.RevertResult{}, err
	}
	reporter.Done(labelCommitting, time.Since(start))
	result.CommitHash = hash

	return result, nil
}

// revertPerNode implements research.md D5-D9's per-node reconciliation: it
// removes every exclusively-owned node the ingest commit touched (D5/D6),
// sweeps every referrer's backlinks against that whole removed set in one
// pass (research.md D6's load-bearing detail — never once per removed
// node, so an earlier removal's edge drop can never be resurrected by a
// later removal's own read against the same pre-revert snapshot), then
// reconciles every shared node (D7-D9). It leaves the commit itself to its
// caller.
func revertPerNode(ctx context.Context, store fsys.Store, vcs port.VCS, reporter bios.Reporter, index core.Index, dir, sourceID, ingestHash string, paths []string, touchingByPath map[string][]string) (kernel.RevertResult, error) {
	result := kernel.RevertResult{
		Document:   sourceID,
		Approach:   "per-node",
		Removed:    map[string]int{},
		Reconciled: map[string]int{},
	}

	preIndex, err := enumerateNodes(store)
	if err != nil {
		return kernel.RevertResult{}, err
	}
	rev := buildReverseIndex(preIndex)

	sortedPaths := append([]string(nil), paths...)
	sort.Strings(sortedPaths)

	var exclusivePaths, sharedPaths []string
	for _, p := range sortedPaths {
		// A patch/exchange document left sitting inside the graph tree
		// (e.g. still touched by its own ingest commit's `git add -A`) is
		// a distinct, valid concept, never itself a graph node — skipped
		// here exactly as apply.go's own guardNoOldFormatNodes already
		// tolerates it.
		if isPatchDocument(store, p) {
			continue
		}
		if len(touchingByPath[p]) == 1 {
			exclusivePaths = append(exclusivePaths, p)
		} else {
			sharedPaths = append(sharedPaths, p)
		}
	}

	removedIDs := map[string]bool{}
	removedByPath := map[string]core.Node{}
	for _, p := range exclusivePaths {
		node, ok, err := readExistingNode(store, p, index)
		if err != nil {
			return kernel.RevertResult{}, err
		}
		if !ok {
			continue
		}
		if err := store.Remove(p); err != nil {
			return kernel.RevertResult{}, ErrNodeWrite.With(err, p)
		}
		result.Removed[node.Type]++
		removedIDs[node.ID] = true
		removedByPath[p] = node
	}

	swept, err := sweepBacklinks(store, index, preIndex, rev, removedIDs)
	if err != nil {
		return kernel.RevertResult{}, err
	}
	for _, n := range swept {
		result.LinksRemoved += n
	}

	var nodeOutcomes []kernel.NodeOutcome
	for _, p := range exclusivePaths {
		node, ok := removedByPath[p]
		if !ok {
			continue
		}
		n := len(rev[node.ID])
		detail := fmt.Sprintf("%d link%s removed", n, pluralSuffix(n))
		reporter.Step(fmt.Sprintf("%s: removed (%s)", p, detail))
		nodeOutcomes = append(nodeOutcomes, kernel.NodeOutcome{Path: p, Kind: "removed", Detail: detail})
	}

	for _, p := range sharedPaths {
		node, ok, err := readExistingNode(store, p, index)
		if err != nil {
			return kernel.RevertResult{}, err
		}
		if !ok {
			continue
		}

		updated, changed, detail, err := reconcileShared(ctx, vcs, index, dir, p, node, ingestHash, sourceID)
		if err != nil {
			return kernel.RevertResult{}, err
		}

		// A path can be both a shared node in its own right (attributable
		// text to strip) and a referrer whose backlink to some other
		// removed node in this same revert was just swept — the two
		// outcomes are merged into one NodeOutcome/Reporter.Step per path,
		// never reported separately (the second would otherwise read
		// "unchanged" for a file that was, in fact, just rewritten).
		linksDropped := swept[p]
		if !changed && linksDropped == 0 {
			reporter.Step(fmt.Sprintf("%s: unchanged (no attributable content)", p))
			nodeOutcomes = append(nodeOutcomes, kernel.NodeOutcome{Path: p, Kind: "unchanged"})
			continue
		}

		if changed {
			if err := writeNode(store, p, updated, index); err != nil {
				return kernel.RevertResult{}, err
			}
			result.Reconciled[updated.Type]++
		} else if linksDropped > 0 {
			result.Reconciled[node.Type]++
		}

		if linksDropped > 0 {
			linkDetail := fmt.Sprintf("%d link%s removed", linksDropped, pluralSuffix(linksDropped))
			if detail == "" {
				detail = linkDetail
			} else {
				detail = linkDetail + ", " + detail
			}
		}
		reporter.Step(fmt.Sprintf("%s: reconciled (%s)", p, detail))
		nodeOutcomes = append(nodeOutcomes, kernel.NodeOutcome{Path: p, Kind: "reconciled", Detail: detail})
	}

	sort.Slice(nodeOutcomes, func(i, j int) bool { return nodeOutcomes[i].Path < nodeOutcomes[j].Path })
	result.Nodes = nodeOutcomes

	return result, nil
}

// isPatchDocument reports whether path parses as a core.Patch exchange
// document rather than a graph node — mirroring apply.go's own
// guardNoOldFormatNodes tolerance (a patch/exchange file left sitting
// inside the graph tree, e.g. still touched by its own ingest commit, is
// a distinct, valid concept, never itself a node revert should try to
// remove or reconcile). A path that no longer exists or fails to read is
// treated as false (not a patch), letting the caller's own existing
// not-found handling apply.
func isPatchDocument(store fsys.Store, path string) bool {
	f, err := store.Open(path)
	if err != nil {
		return false
	}
	raw, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		return false
	}
	_, perr := core.ParsePatch(bytes.NewReader(raw), core.Index{})
	return perr == nil
}

// referrerPath is nodePath's own generic per-kind folder derivation,
// except for a @type: Timeline referrer — nodePath's fallback
// pluralization would derive "timelines/<id>.md", but a timeline period
// file actually lives under periodGranularity's own two-tier
// "timeline/yearly|monthly/<id>.md" layout (apply.go's own
// upsertTimelinePeriod writer). Used only to key sweepBacklinks' returned
// map under the path a caller's own ChangedPaths-derived path (`p`) will
// actually match.
func referrerPath(node core.Node) string {
	if node.Type == "Timeline" {
		if path, _, _, ok := periodGranularity(node.ID); ok {
			return path
		}
	}
	return nodePath(node)
}

// sweepBacklinks rewrites every referrer of any id in removedIDs, computed
// once against the pre-revert reverseIndex, in exactly one pass per
// referrer (research.md D6). An ordinary referrer is rewritten via
// core.RenderNode; a @type: Timeline referrer is diverted to
// removeTimelineEntry, the structural sibling to apply.go's own
// upsertTimelinePeriod (BUG-007's rendering boundary, preserved here
// rather than duplicated with a second, divergent writer). The returned
// map, keyed by referrer path, lets a caller merge this sweep's outcome
// with a separate per-path outcome (e.g. reconcileShared's own, for a
// path that is both a referrer and independently shared) instead of
// reporting the same path "touched" twice under different labels.
func sweepBacklinks(store fsys.Store, index core.Index, preIndex nodeIndex, rev reverseIndex, removedIDs map[string]bool) (map[string]int, error) {
	referrers := map[string]bool{}
	for id := range removedIDs {
		for _, r := range rev[id] {
			if removedIDs[r] {
				continue
			}
			referrers[r] = true
		}
	}

	sorted := make([]string, 0, len(referrers))
	for r := range referrers {
		sorted = append(sorted, r)
	}
	sort.Strings(sorted)

	swept := map[string]int{}
	for _, id := range sorted {
		referrer := preIndex[id]

		var kept []core.Link
		dropped := 0
		for _, l := range referrer.Edges {
			if removedIDs[l.Target] {
				dropped++
				continue
			}
			kept = append(kept, l)
		}
		if dropped == 0 {
			continue
		}
		swept[referrerPath(referrer)] = dropped

		if referrer.Type == "Timeline" {
			if err := removeTimelineEntry(store, referrer, removedIDs); err != nil {
				return nil, err
			}
			continue
		}

		referrer.Edges = kept
		if err := writeNode(store, nodePath(referrer), referrer, index); err != nil {
			return nil, err
		}
	}

	return swept, nil
}

func pluralSuffix(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// revertLeadingKey mirrors internal/core's own private
// textPredicateFor(nodeType, true) lookup (internal/core/markdown.go) —
// duplicated here because internal/core stays untouched by this feature
// (plan.md Technical Context: "no new function signatures") and exposes no
// public equivalent. Both tables are explicitly documented as a temporary
// stopgap pending spec 011's Schema Index; keep them in sync if core's
// table ever changes.
func revertLeadingKey(nodeType string) string {
	switch nodeType {
	case "Source":
		return "abstract"
	case "Entity":
		return "definition"
	case "Resource":
		return "relevance"
	case "hypothesis":
		return "claim"
	case "aporia":
		return "tension"
	case "thought":
		return "claim"
	default:
		return "text"
	}
}

// revertTrailingKey mirrors internal/core's own private
// textPredicateFor(nodeType, false), which is unconditionally "notes"
// regardless of nodeType.
const revertTrailingKey = "notes"

const (
	conflictMarkerOpenLine    = "<<<<<<< existing\n"
	conflictMarkerSepLine     = "\n=======\n"
	conflictMarkerClosePrefix = "\n>>>>>>> "
)

// parseConflictMarker mirrors internal/core/merge.go's own private
// conflictMarker shape — reimplemented locally since revert.go cannot
// import core's unexported helpers and core itself stays untouched by
// this feature.
func parseConflictMarker(value string) (existingVal, incomingVal, incomingSourceID string, ok bool) {
	if !strings.HasPrefix(value, conflictMarkerOpenLine) {
		return "", "", "", false
	}
	rest := value[len(conflictMarkerOpenLine):]
	sepIdx := strings.Index(rest, conflictMarkerSepLine)
	if sepIdx < 0 {
		return "", "", "", false
	}
	existingVal = rest[:sepIdx]
	rest = rest[sepIdx+len(conflictMarkerSepLine):]
	closeIdx := strings.LastIndex(rest, conflictMarkerClosePrefix)
	if closeIdx < 0 {
		return "", "", "", false
	}
	incomingVal = rest[:closeIdx]
	incomingSourceID = rest[closeIdx+len(conflictMarkerClosePrefix):]
	return existingVal, incomingVal, incomingSourceID, true
}

// resolveConflictMarker implements research.md D8's two provenance
// sub-cases for one conflict-marker-shaped Texts value: (a) the reverted
// patch is the marker's own self-documented incoming side — a plain-text
// comparison, no git call needed; (b) the reverted patch is the marker's
// frozen "existing" side, resolved by walking CommitsTouching(path)
// oldest-first via ShowFile+core.ParseNode to find this predicate's true
// first writer (every scalar merge behavior a conflict marker can appear
// under freezes a predicate's value at its first writer forever). Neither
// case matching means the reverted patch made no contribution to this
// specific predicate — matched is false and value is left untouched.
func resolveConflictMarker(ctx context.Context, vcs port.VCS, dir, path, key, existingVal, incomingVal, incomingSourceID, ingestHash, revertedSourceID string) (resolved string, matched bool, err error) {
	if incomingSourceID == revertedSourceID {
		return existingVal, true, nil
	}

	commits, err := vcs.CommitsTouching(ctx, dir, path)
	if err != nil {
		return "", false, err
	}

	for i := len(commits) - 1; i >= 0; i-- {
		c := commits[i]
		raw, err := vcs.ShowFile(ctx, dir, c, path)
		if err != nil {
			return "", false, err
		}
		if raw == nil {
			continue
		}
		historical, err := core.ParseNode(bytes.NewReader(raw), core.Index{})
		if err != nil {
			continue
		}
		if historical.Texts[key] != "" {
			if c == ingestHash {
				return incomingVal, true, nil
			}
			break
		}
	}

	return "", false, nil
}

// textParagraphRange is one paragraph's located span within a node file's
// current rendered bytes — its own text (for D7's paragraph-drop
// rewrite) plus its 1-indexed, inclusive line range (for intersecting
// against Blame's ingestHash-attributed lines).
type textParagraphRange struct {
	text      string
	startLine int
	endLine   int
}

// splitParagraphsLocal mirrors internal/core/merge.go's own private
// splitParagraphs (blank-line-delimited blocks) — reimplemented locally
// since core stays untouched by this feature and exposes no equivalent.
func splitParagraphsLocal(text string) []string {
	raw := strings.Split(text, "\n\n")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// mapTextParagraphs locates every Texts-key paragraph within rendered
// (core.RenderNode(node, index)'s own current output for node — the bytes
// actually on disk and blamed) by a forward, cursor-advancing search:
// internal/core's private reconstructHRefs never inserts or removes a
// line break, only bracket markup within an existing line, so a raw
// paragraph's own first line matches verbatim in rendered whenever that
// line carries no inline [[mention]] — the one known gap this local
// approximation accepts (a paragraph whose very first line contains a
// mention is left unattributable to a blamed commit, which D9 already
// treats as a safe, successful no-op — never silent data loss in the
// other direction). Physical order (leading key, other keys
// alphabetically, trailing key last — internal/core/markdown.go:754-767)
// is walked in that same sequence so the monotonically-advancing cursor
// never revisits an earlier line, keeping each key's occurrences
// correctly ordered even across the intervening, unparsed Edges block.
func mapTextParagraphs(node core.Node, leadingKey, trailingKey string, rendered []byte) map[string][]textParagraphRange {
	lines := strings.Split(strings.TrimSuffix(string(rendered), "\n"), "\n")
	out := map[string][]textParagraphRange{}
	cursor := 0

	appendKey := func(key string) {
		value, ok := node.Texts[key]
		if !ok || value == "" {
			return
		}
		for _, p := range splitParagraphsLocal(value) {
			pLines := strings.Split(p, "\n")
			foundAt := -1
			for i := cursor; i < len(lines); i++ {
				if lines[i] == pLines[0] {
					foundAt = i
					break
				}
			}
			if foundAt < 0 {
				continue
			}
			endAt := foundAt + len(pLines) - 1
			out[key] = append(out[key], textParagraphRange{text: p, startLine: foundAt + 1, endLine: endAt + 1})
			cursor = endAt + 1
		}
	}

	if leadingKey != "" {
		appendKey(leadingKey)
	}
	var others []string
	for k := range node.Texts {
		if k == leadingKey || k == trailingKey {
			continue
		}
		others = append(others, k)
	}
	sort.Strings(others)
	for _, k := range others {
		appendKey(k)
	}
	if trailingKey != "" {
		appendKey(trailingKey)
	}

	return out
}

func allLinesBlamed(pr textParagraphRange, blamedSet map[int]bool) bool {
	for line := pr.startLine; line <= pr.endLine; line++ {
		if !blamedSet[line] {
			return false
		}
	}
	return true
}

// physicalTextKeyOrder returns node's Texts keys in renderNodeBody's own
// documented physical order (leading key, other keys alphabetically,
// trailing key last).
func physicalTextKeyOrder(node core.Node, leadingKey, trailingKey string) []string {
	var keys []string
	if _, ok := node.Texts[leadingKey]; ok {
		keys = append(keys, leadingKey)
	}
	var others []string
	for k := range node.Texts {
		if k == leadingKey || k == trailingKey {
			continue
		}
		others = append(others, k)
	}
	sort.Strings(others)
	keys = append(keys, others...)
	if _, ok := node.Texts[trailingKey]; ok {
		keys = append(keys, trailingKey)
	}
	return keys
}

// reconcileShared implements research.md D7-D9 for one shared node: walks
// every Texts key in physical order, resolving a conflict-marker-shaped
// value via resolveConflictMarker's D8 provenance rules or, for an
// ordinary value, stripping only the paragraph(s) whose every line git
// blame attributes to ingestHash (D7) — a paragraph blamed even partly to
// any other commit is left in place, by construction. Attrs/Edges/HRefs
// are never read or written here — FR-011's explicit scope guard. changed
// is false (a safe, successful D9 no-op) when nothing on this node was
// attributable to ingestHash.
func reconcileShared(ctx context.Context, vcs port.VCS, index core.Index, dir, path string, node core.Node, ingestHash, revertedSourceID string) (updated core.Node, changed bool, detail string, err error) {
	rendered, err := core.RenderNode(node, index)
	if err != nil {
		return node, false, "", err
	}

	leadingKey := revertLeadingKey(node.Type)
	keys := physicalTextKeyOrder(node, leadingKey, revertTrailingKey)

	newTexts := make(map[string]string, len(node.Texts))
	for k, v := range node.Texts {
		newTexts[k] = v
	}

	var conflictCount, paragraphCount int
	var blamedSet map[int]bool
	var paragraphRanges map[string][]textParagraphRange

	for _, key := range keys {
		value := node.Texts[key]
		if existingVal, incomingVal, incomingSourceID, ok := parseConflictMarker(value); ok {
			resolved, matched, err := resolveConflictMarker(ctx, vcs, dir, path, key, existingVal, incomingVal, incomingSourceID, ingestHash, revertedSourceID)
			if err != nil {
				return node, false, "", err
			}
			if matched {
				newTexts[key] = resolved
				conflictCount++
			}
			continue
		}

		if blamedSet == nil {
			blame, err := vcs.Blame(ctx, dir, path)
			if err != nil {
				return node, false, "", err
			}
			blamedSet = map[int]bool{}
			for _, bl := range blame {
				if bl.Commit == ingestHash {
					blamedSet[bl.Number] = true
				}
			}
			paragraphRanges = mapTextParagraphs(node, leadingKey, revertTrailingKey, rendered)
		}

		var kept []string
		for _, pr := range paragraphRanges[key] {
			if allLinesBlamed(pr, blamedSet) {
				paragraphCount++
				continue
			}
			kept = append(kept, pr.text)
		}
		if len(kept) != len(splitParagraphsLocal(value)) {
			newTexts[key] = strings.Join(kept, "\n\n")
		}
	}

	if conflictCount == 0 && paragraphCount == 0 {
		return node, false, "", nil
	}

	updated = node
	updated.Texts = newTexts

	var parts []string
	if paragraphCount > 0 {
		parts = append(parts, fmt.Sprintf("%d paragraph%s stripped", paragraphCount, pluralSuffix(paragraphCount)))
	}
	if conflictCount > 0 {
		parts = append(parts, fmt.Sprintf("%d conflict marker%s resolved", conflictCount, pluralSuffix(conflictCount)))
	}
	return updated, true, strings.Join(parts, ", "), nil
}

func sumCounts(m map[string]int) int {
	total := 0
	for _, n := range m {
		total += n
	}
	return total
}

func formatKindCounts(m map[string]int) string {
	kinds := make([]string, 0, len(m))
	for k := range m {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	parts := make([]string, 0, len(kinds))
	for _, k := range kinds {
		parts = append(parts, fmt.Sprintf("%s: %d", k, m[k]))
	}
	return strings.Join(parts, ", ")
}

// buildRevertCommitMessage mirrors apply.go's buildCommitMessage shape
// (data-model.md's per-node commit message section) — same overall
// structure, new subject, and a deliberately distinct trailer key
// (Reverted-Document:, never Source-Id:) so a retracted-then-re-applied
// document's own ingest-commit uniqueness invariant (spec 003 FR-012) is
// never corrupted by a same-prefixed trailer satisfying CommitsMatching's
// plain substring --grep test.
func buildRevertCommitMessage(sourceID string, result kernel.RevertResult) string {
	subject := fmt.Sprintf("graph(revert): %s — per-node reconciliation", sourceID)

	var buf strings.Builder
	buf.WriteString(subject)
	buf.WriteString("\n\n")
	buf.WriteString(fmt.Sprintf("Removed: %d nodes (%s)\n", sumCounts(result.Removed), formatKindCounts(result.Removed)))
	buf.WriteString(fmt.Sprintf("Reconciled: %d nodes (%s)\n", sumCounts(result.Reconciled), formatKindCounts(result.Reconciled)))
	buf.WriteString(fmt.Sprintf("Links removed: %d\n", result.LinksRemoved))
	buf.WriteString("Reverted-Document: " + sourceID + "\n")

	return buf.String()
}

// removeTimelineEntry is a structural sibling to apply.go's own
// upsertTimelinePeriod, reusing parseTimelineEntries/periodGranularity
// (research.md D6): it removes every entry in node's own timeline period
// file whose id is in removedIDs, re-serializing with the exact same
// front-matter/heading shape upsertTimelinePeriod already produces —
// preserving BUG-007's rendering boundary (a timeline period file is never
// written via the generic core.RenderNode path) rather than introducing a
// second, divergent writer for the same file shape.
func removeTimelineEntry(store fsys.Store, node core.Node, removedIDs map[string]bool) error {
	path, granularity, heading, ok := periodGranularity(node.ID)
	if !ok {
		return nil
	}

	existing, err := readFileIfExists(store, path)
	if err != nil {
		return err
	}

	entries := parseTimelineEntries(existing)
	var kept []timelineEntry
	removedAny := false
	for _, e := range entries {
		if removedIDs[e.id] {
			removedAny = true
			continue
		}
		kept = append(kept, e)
	}
	if !removedAny {
		return nil
	}

	period := node.ID
	created := attrString(node, "created")
	var buf strings.Builder
	buf.WriteString("---\n")
	buf.WriteString("\"@id\": \"" + period + "\"\n")
	buf.WriteString("\"@type\": Timeline\n")
	buf.WriteString("period: \"" + period + "\"\n")
	buf.WriteString("granularity: " + granularity + "\n")
	buf.WriteString("published: \"" + created + "\"\n")
	buf.WriteString("created: \"" + created + "\"\n")
	buf.WriteString("---\n")
	buf.WriteString("# " + heading + "\n\n")
	for _, e := range kept {
		buf.WriteString(e.line)
		buf.WriteString("\n")
	}

	return writeRaw(store, path, []byte(buf.String()))
}
