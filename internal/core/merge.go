//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package core

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// PredicateOutcome records how one predicate present on either side of a
// merge reconciled: its name, the MergeOp it resolved to (via
// index.Predicates, per FR-013), and a short Outcome label (one of the
// Outcome* constants, spec.md FR-017/BUG-001). Surfaced by
// internal/app/graph/service.Apply's --verbose report as one additional
// line per predicate, alongside the existing per-node summary — never
// rendered into a Node itself. Scoped to Attrs/Texts/Published, the
// predicates whose reconciliation genuinely varies by declared MergeOp;
// Edges/HRefs always union unconditionally regardless of op (research.md
// D5d), so there is no per-predicate algorithm choice there to report.
type PredicateOutcome struct {
	Name    string
	Op      MergeOp
	Outcome string
}

// Outcome labels a PredicateOutcome.Outcome MUST be one of (spec.md
// FR-017).
const (
	OutcomeCreated     = "created"     // key absent from existing, contributed for the first time by this merge
	OutcomeUnchanged   = "unchanged"   // existing value (or its absence) carried through untouched
	OutcomeFilled      = "filled"      // existing was empty/zero; incoming supplied its first value
	OutcomeOverwritten = "overwritten" // lastWriteWin took a differing incoming value
	OutcomeAppended    = "appended"    // union/append list or paragraph dispatch
	OutcomeFlagged     = "flagged"     // a genuine divergence was wrapped in a conflict marker
)

// Merge reconciles incoming into existing per CORE §9.3's per-predicate
// merge model (contracts/merge-behavior-contract.md): every predicate
// present in Attrs, Texts, Edges, or HRefs on either side is reconciled
// individually according to its own MergeOp, looked up in
// index.Predicates — never by one behavior applied to the whole node. A
// predicate absent from index.Predicates falls back to MergeUnion
// (research.md D6). Published is reconciled the same way, keyed by
// index.Predicates["published"].Merge. existing is the zero Node
// (ID == "") only when no node with incoming's identity exists yet — the
// caller (internal/app/graph/service.Apply) treats that case as a plain
// create, never calling Merge. conflicts lists every predicate name whose
// value was flagged (firstWriteWin/fillIfEmpty divergence only); empty
// when nothing diverged. outcomes is one PredicateOutcome per predicate
// present in Attrs/Texts on either side plus Published, sorted by name
// (spec.md FR-017/BUG-001) — additive relative to conflicts, which it
// subsumes. Merge performs no I/O and is commutative and idempotent for
// every MergeOp except lastWriteWin, which is intentionally
// application-order-sensitive (research.md D5a).
func Merge(existing, incoming Node, index Index, sourceID string) (Node, []string, []PredicateOutcome, error) {
	merged := existing

	var conflicts []string
	var outcomes []PredicateOutcome

	publishedOp := resolveMergeOp(index, "published")
	published, diverges, publishedOutcome := mergeScalar(existing.Published, incoming.Published, time.Time{}, publishedOp)
	merged.Published = published
	if diverges {
		conflicts = append(conflicts, "published")
	}
	outcomes = append(outcomes, PredicateOutcome{Name: "published", Op: publishedOp, Outcome: publishedOutcome})

	var textConflicts []string
	var textOutcomes []PredicateOutcome
	merged.Texts, textConflicts, textOutcomes = mergeTexts(existing.Texts, incoming.Texts, index, sourceID)
	conflicts = append(conflicts, textConflicts...)
	outcomes = append(outcomes, textOutcomes...)

	var attrConflicts []string
	var attrOutcomes []PredicateOutcome
	merged.Attrs, attrConflicts, attrOutcomes = mergeAttrs(existing.Attrs, incoming.Attrs, index, sourceID)
	conflicts = append(conflicts, attrConflicts...)
	outcomes = append(outcomes, attrOutcomes...)

	merged.Edges = unionLinks(existing.Edges, incoming.Edges)
	merged.HRefs = unionLinks(existing.HRefs, incoming.HRefs)

	sort.Slice(outcomes, func(i, j int) bool { return outcomes[i].Name < outcomes[j].Name })

	return merged, conflicts, outcomes, nil
}

// resolveMergeOp looks up predicate's declared MergeOp in index, falling
// back to MergeUnion (research.md D6) when the predicate has no schema
// document yet — mirroring apply.go's own unrecognized-type fallback
// precedent.
func resolveMergeOp(index Index, predicate string) MergeOp {
	if def, ok := index.Predicates[predicate]; ok {
		return def.Merge
	}
	return MergeUnion
}

// isListMerge reports whether op reconciles a predicate as a deduplicated
// list rather than a single scalar value (research.md D5): every seeded
// link/edge-role predicate declares one of these two, and Edges/HRefs
// always union regardless of op (D5d).
func isListMerge(op MergeOp) bool {
	return op == MergeUnion || op == MergeAppend
}

// mergeScalar implements research.md D5's three scalar dispatch classes,
// chosen by op: freeze (immutable, validatedOverwrite) — existing, once
// non-empty, is permanent, never flagged; flagOnDiverge (firstWriteWin,
// fillIfEmpty) — same freeze rule, except a later genuine divergence is
// reported via diverges; alwaysOverwrite (lastWriteWin) — incoming,
// whenever non-empty, always replaces existing, never flagged. Every
// class accepts incoming unconditionally, without flagging, when existing
// is still the zero value — "first write" for freeze/flagOnDiverge,
// simply "no prior value to overwrite" for lastWriteWin. outcome is one
// of the Outcome* constants (spec.md FR-017/BUG-001), derived from the
// same branch that decided merged/diverges — never computed separately.
func mergeScalar[T comparable](existing, incoming, zero T, op MergeOp) (merged T, diverges bool, outcome string) {
	if incoming == zero || incoming == existing {
		return existing, false, OutcomeUnchanged
	}
	if existing == zero {
		return incoming, false, OutcomeFilled
	}
	if op == MergeLastWriteWin {
		return incoming, false, OutcomeOverwritten
	}
	if op == MergeFirstWriteWin || op == MergeFillIfEmpty {
		return existing, true, OutcomeFlagged
	}
	return existing, false, OutcomeUnchanged
}

// paragraphSimilarityThreshold is the Jaccard-over-3-word-shingles score
// above which an incoming paragraph is considered a near-duplicate of an
// existing one (BUG-004, research.md D6 Bugfix) — a common, reasonable
// default for near-duplicate paragraph detection, not user-configurable.
const paragraphSimilarityThreshold = 0.8

// paragraphShingleSize is the shingle (word n-gram) width mergeText
// compares paragraphs by.
const paragraphShingleSize = 3

// mergeTexts merges the Texts map key-by-key over the union of both
// nodes' keys: a key declared append (or union, D5b's documented
// fallback) is reconciled paragraph-by-paragraph through mergeText; every
// other key goes through the scalar dispatch classes (mergeScalar),
// keyed by that key's own declared MergeOp — no key is special-cased by
// name (research.md D5b: "notes" needs no carve-out anymore, since its own
// predicate already declares firstWriteWin). A key present on only one
// side behaves like a scalar/paragraph merge against "" on the other.
func mergeTexts(existing, incoming map[string]string, index Index, sourceID string) (map[string]string, []string, []PredicateOutcome) {
	merged := make(map[string]string, len(existing)+len(incoming))

	keys := make(map[string]bool, len(existing)+len(incoming))
	for k := range existing {
		keys[k] = true
	}
	for k := range incoming {
		keys[k] = true
	}
	sortedKeys := make([]string, 0, len(keys))
	for k := range keys {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	var conflicts []string
	var outcomes []PredicateOutcome
	for _, k := range sortedKeys {
		ev := existing[k]
		iv := incoming[k]
		op := resolveMergeOp(index, k)
		_, existed := existing[k]

		if isListMerge(op) {
			merged[k] = mergeText(ev, iv)
			outcome := OutcomeAppended
			switch {
			case !existed:
				outcome = OutcomeCreated
			case merged[k] == ev:
				// BUG-002: mergeText's own near-duplicate detection
				// (paragraphAlreadyPresent) can legitimately produce a
				// merged value identical to the existing one — that is
				// a no-op, not an append, and must be reported as such.
				outcome = OutcomeUnchanged
			}
			outcomes = append(outcomes, PredicateOutcome{Name: k, Op: op, Outcome: outcome})
			continue
		}

		var textOutcome string
		merged[k], conflicts, textOutcome = mergeScalarInto(ev, iv, k, sourceID, op, conflicts)
		outcomes = append(outcomes, PredicateOutcome{Name: k, Op: op, Outcome: textOutcome})
	}

	return merged, conflicts, outcomes
}

// mergeText reconciles an append/union-declared Texts key paragraph-by-
// paragraph (spec.md FR-008, research.md D5b) instead of as one scalar:
// prose is routinely regenerated wholesale by its producing pipeline, so
// comparing it as a single value guarantees a false-positive conflict on
// nearly every re-application. Splitting on paragraph boundaries and
// dropping any incoming paragraph that is a near-duplicate of one already
// present keeps the reconciliation additive and silent — never flagged,
// never lossy on either side.
func mergeText(existingText, incomingText string) string {
	if incomingText == "" {
		return existingText
	}
	if existingText == "" {
		return incomingText
	}
	merged := mergeParagraphs(splitParagraphs(existingText), splitParagraphs(incomingText))
	return strings.Join(merged, "\n\n")
}

// mergeParagraphs appends every incoming paragraph not already covered by
// an existing one to the existing paragraph sequence, existing first,
// preserving incoming's own order for the genuinely new material.
func mergeParagraphs(existing, incoming []string) []string {
	existingShingles := make([][]string, len(existing))
	for i, p := range existing {
		existingShingles[i] = shingles(p, paragraphShingleSize)
	}

	merged := append([]string(nil), existing...)
	for _, p := range incoming {
		if !paragraphAlreadyPresent(shingles(p, paragraphShingleSize), existingShingles) {
			merged = append(merged, p)
		}
	}
	return merged
}

func paragraphAlreadyPresent(incoming []string, existingShingles [][]string) bool {
	for _, es := range existingShingles {
		if jaccardSimilarity(es, incoming) > paragraphSimilarityThreshold {
			return true
		}
	}
	return false
}

// splitParagraphs recovers the paragraph granularity ParseNode/RenderNode
// already round-trip Text through: blank-line-delimited blocks.
func splitParagraphs(text string) []string {
	raw := strings.Split(text, "\n\n")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// shingles splits text into overlapping word n-grams; a paragraph shorter
// than n becomes its own single shingle.
func shingles(text string, n int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	if len(words) < n {
		return []string{strings.Join(words, " ")}
	}
	out := make([]string, 0, len(words)-n+1)
	for i := 0; i+n <= len(words); i++ {
		out = append(out, strings.Join(words[i:i+n], " "))
	}
	return out
}

// jaccardSimilarity is |A∩B| / |A∪B| over two shingle sets.
func jaccardSimilarity(a, b []string) float64 {
	setA := make(map[string]bool, len(a))
	for _, s := range a {
		setA[s] = true
	}
	setB := make(map[string]bool, len(b))
	for _, s := range b {
		setB[s] = true
	}

	intersection := 0
	for s := range setA {
		if setB[s] {
			intersection++
		}
	}
	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

// mergeScalarInto flags a genuine divergence via conflictMarker, except
// when existingVal is already the marker a previous merge produced for
// this exact incomingVal (spec.md US3 Acceptance Scenario 4/FR-010): that
// case is idempotent replay, not a fresh divergence, so the marker is
// re-flagged in conflicts without being wrapped again.
func mergeScalarInto(existingVal, incomingVal, field, sourceID string, op MergeOp, conflicts []string) (string, []string, string) {
	if embedded, ok := incomingFromConflictMarker(existingVal); ok && embedded == incomingVal {
		return existingVal, append(conflicts, field), OutcomeFlagged
	}

	merged, diverges, outcome := mergeScalar(existingVal, incomingVal, "", op)
	if !diverges {
		return merged, conflicts, outcome
	}
	return conflictMarker(existingVal, incomingVal, sourceID), append(conflicts, field), outcome
}

// conflictMarker reproduces VISION.md's documented git-style conflict
// marker (research.md D7). existing's own original writer cannot be
// determined from a Node value alone (no per-field provenance is stored),
// so the left side always falls back to the literal token "existing".
func conflictMarker(existingVal, incomingVal, incomingSourceID string) string {
	return fmt.Sprintf("<<<<<<< existing\n%s\n=======\n%s\n>>>>>>> %s", existingVal, incomingVal, incomingSourceID)
}

const (
	conflictMarkerOpen  = "<<<<<<< existing\n"
	conflictMarkerSep   = "\n=======\n"
	conflictMarkerClose = "\n>>>>>>> "
)

// incomingFromConflictMarker extracts the incoming-side value embedded in
// a previously-produced conflictMarker, so a replayed merge can recognize
// "this exact divergence was already flagged" instead of nesting a new
// marker around the old one.
func incomingFromConflictMarker(value string) (incoming string, ok bool) {
	if !strings.HasPrefix(value, conflictMarkerOpen) {
		return "", false
	}
	rest := value[len(conflictMarkerOpen):]
	sepIdx := strings.Index(rest, conflictMarkerSep)
	if sepIdx < 0 {
		return "", false
	}
	rest = rest[sepIdx+len(conflictMarkerSep):]
	closeIdx := strings.LastIndex(rest, conflictMarkerClose)
	if closeIdx < 0 {
		return "", false
	}
	return rest[:closeIdx], true
}

// mergeAttrs merges the Attrs map key-by-key over the union of keys: a
// key present on only one side is taken unchanged. A key declared union
// or append always reconciles as one deduplicated list union
// (unionPredicates), regardless of how many values happen to be present
// on either side right now (research.md D5c — a documented, intentional
// behavior change from arity-based dispatch). Every other key reconciles
// through mergeScalar's scalar dispatch classes (freeze/flagOnDiverge/
// alwaysOverwrite) on the key's whole value — a single Predicate for an
// ordinary scalar attribute, or the whole list verbatim for a seeded
// multi-valued-by-nature key that nonetheless declares a scalar-dispatch
// op (e.g. "category"), never truncated to its first element.
func mergeAttrs(existing, incoming map[string][]Predicate, index Index, sourceID string) (map[string][]Predicate, []string, []PredicateOutcome) {
	merged := make(map[string][]Predicate, len(existing)+len(incoming))
	for k, v := range existing {
		merged[k] = v
	}

	keys := make(map[string]bool, len(existing)+len(incoming))
	for k := range existing {
		keys[k] = true
	}
	for k := range incoming {
		keys[k] = true
	}
	sortedKeys := make([]string, 0, len(keys))
	for k := range keys {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	var conflicts []string
	var outcomes []PredicateOutcome
	for _, k := range sortedKeys {
		iv, incomingPresent := incoming[k]
		ev, existed := existing[k]
		op := resolveMergeOp(index, k)

		if !incomingPresent {
			// Present only in existing: carried through untouched by this
			// merge — nothing for this predicate to decide.
			outcomes = append(outcomes, PredicateOutcome{Name: k, Op: op, Outcome: OutcomeUnchanged})
			continue
		}

		if !existed {
			merged[k] = iv
			outcomes = append(outcomes, PredicateOutcome{Name: k, Op: op, Outcome: OutcomeCreated})
			continue
		}

		if isListMerge(op) {
			merged[k] = unionPredicates(ev, iv)
			outcome := OutcomeAppended
			if predicateSliceKey(merged[k]) == predicateSliceKey(ev) {
				// BUG-002: every incoming value was already present —
				// unionPredicates' own dedup produced a no-op, not an
				// append, and must be reported as such.
				outcome = OutcomeUnchanged
			}
			outcomes = append(outcomes, PredicateOutcome{Name: k, Op: op, Outcome: outcome})
			continue
		}

		mergedPreds, flagged, outcome := mergeScalarPredicate(ev, iv, sourceID, op)
		merged[k] = mergedPreds
		if flagged {
			conflicts = append(conflicts, k)
		}
		outcomes = append(outcomes, PredicateOutcome{Name: k, Op: op, Outcome: outcome})
	}

	return merged, conflicts, outcomes
}

func firstPredicate(preds []Predicate) Predicate {
	if len(preds) == 0 {
		return Predicate{}
	}
	return preds[0]
}

// mergeScalarPredicate dispatches a whole Attrs key's Predicate list
// through mergeScalar's scalar classes (freeze/flagOnDiverge/
// alwaysOverwrite). The common case — at most one Predicate per side, an
// ordinary scalar attribute — compares/merges the bare Value (via
// fmt.Sprint, mirroring how a scalar Attrs value already compares today)
// and, on divergence, wraps a conflict marker exactly as before. A key
// that is multi-valued on either side despite declaring a scalar op
// (research.md's "category" — a seeded word-bag predicate declared
// firstWriteWin) is compared/merged as one whole list instead, so a
// genuinely unchanged multi-valued list is recognized as equal rather
// than truncated to its first element. Mirrors mergeScalarInto's
// replay-idempotency guard: re-merging the exact incoming value a
// previous marker already recorded re-flags it without wrapping a second
// marker around the first (spec.md US3 Acceptance Scenario 4).
func mergeScalarPredicate(existing, incoming []Predicate, sourceID string, op MergeOp) (merged []Predicate, flagged bool, outcome string) {
	if len(existing) <= 1 && len(incoming) <= 1 {
		mergedPred, flag, out := mergeScalarPredicateValue(firstPredicate(existing), firstPredicate(incoming), sourceID, op)
		return []Predicate{mergedPred}, flag, out
	}

	evKey, ivKey := predicateSliceKey(existing), predicateSliceKey(incoming)

	if embedded, ok := incomingFromConflictMarker(evKey); ok && embedded == ivKey {
		return existing, true, OutcomeFlagged
	}

	mergedKey, diverges, mergeOutcome := mergeScalar(evKey, ivKey, "", op)
	if diverges {
		return []Predicate{{Value: conflictMarker(evKey, ivKey, sourceID)}}, true, mergeOutcome
	}
	if mergedKey == ivKey {
		return incoming, false, mergeOutcome
	}
	return existing, false, mergeOutcome
}

// mergeScalarPredicateValue implements mergeScalar's rule generalized to
// a single Predicate.Value: incoming nil or identical to existing changes
// nothing; an empty/nil existing is filled unconditionally; any other
// divergence is wrapped in a conflict marker and reported via flagged —
// never for lastWriteWin, which always overwrites.
func mergeScalarPredicateValue(existing, incoming Predicate, sourceID string, op MergeOp) (merged Predicate, flagged bool, outcome string) {
	ev, iv := fmt.Sprint(existing.Value), fmt.Sprint(incoming.Value)
	if existing.Value == nil {
		ev = ""
	}
	if incoming.Value == nil {
		iv = ""
	}

	if embedded, ok := incomingFromConflictMarker(ev); ok && embedded == iv {
		return existing, true, OutcomeFlagged
	}

	mergedVal, diverges, mergeOutcome := mergeScalar(ev, iv, "", op)
	if diverges {
		return Predicate{Value: conflictMarker(ev, iv, sourceID)}, true, mergeOutcome
	}
	if mergedVal == ev {
		return existing, false, mergeOutcome
	}
	return incoming, false, mergeOutcome
}

// predicateSliceKey builds a stable, order-sensitive comparison key for a
// whole Attrs Predicate list, reusing predicateDedupeKey per element —
// used only to detect whether two same-shaped multi-valued lists are
// equal or genuinely diverge, never rendered into a Node.
func predicateSliceKey(preds []Predicate) string {
	parts := make([]string, len(preds))
	for i, p := range preds {
		parts[i] = predicateDedupeKey(p)
	}
	return strings.Join(parts, "\x1f")
}

// unionPredicates unions two Predicate lists, deduplicated by Value's
// fmt.Sprint representation, or by Target when Value is nil, existing
// entries first.
func unionPredicates(existing, incoming []Predicate) []Predicate {
	var out []Predicate
	seen := map[string]bool{}
	add := func(p Predicate) {
		key := predicateDedupeKey(p)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, p)
	}
	for _, p := range existing {
		add(p)
	}
	for _, p := range incoming {
		add(p)
	}
	return out
}

func predicateDedupeKey(p Predicate) string {
	if p.Value != nil {
		return "v:" + fmt.Sprint(p.Value)
	}
	return "t:" + p.Target
}

// unionLinks unions two ordered Link lists by the (predicate, target) pair
// (AST §6.5), deduplicated, order-preserving, existing entries first —
// unconditional regardless of any predicate's declared MergeOp (research.md
// D5d): no seeded link/edge predicate uses a scalar-natured op, and a link
// has no single "slot" to freeze or overwrite.
func unionLinks(existing, incoming []Link) []Link {
	var out []Link
	seen := map[[2]string]bool{}
	add := func(l Link) {
		key := [2]string{l.Predicate, l.Target}
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, l)
	}
	for _, l := range existing {
		add(l)
	}
	for _, l := range incoming {
		add(l)
	}
	return out
}
