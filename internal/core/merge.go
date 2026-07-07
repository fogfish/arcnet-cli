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

// mergePublished fills Published from incoming only when existing is not
// yet set — first-writer-wins forever after, never flagged as a conflict
// regardless of the calling op's own fillEmpty/flagConflicts rules
// (research.md D3).
func mergePublished(existing, incoming time.Time) time.Time {
	if !existing.IsZero() {
		return existing
	}
	return incoming
}

// Merge reconciles incoming into existing per CORE §10's fixed menu of
// merge operations (research.md D6). existing is the zero Node (ID == "")
// only when no node with incoming's identity exists yet — the caller
// (internal/app/graph/service.Apply) treats that case as a plain create,
// never calling Merge. conflicts is the list of Texts/Attrs keys
// (research.md D6/D7) whose value was flagged; empty when nothing
// diverged. For MergeUnion, a Texts key other than "notes" is never
// flagged and instead reconciled paragraph-by-paragraph through mergeText
// (BUG-004, spec.md FR-023/FR-024); "notes" always goes through the scalar
// path regardless of the op. Attrs — now list-valued per key — merges a
// single-valued key (exactly one Predicate on both sides) through the same
// scalar fillEmpty/flagConflicts policy a bare scalar used before this
// feature; a key that is multi-valued on either side is always merged as
// a plain list union, never flagged, exactly as before this feature. Merge
// performs no I/O.
func Merge(existing, incoming Node, op MergeOp, sourceID string) (Node, []string, error) {
	switch op {
	case MergeNone:
		return existing, nil, nil
	case MergeUnion:
		// A union kind's scalar Texts/Attrs are never flagged (BUG-004,
		// spec.md FR-023/FR-024): unlike a fact a human sets once, a union
		// kind's scalar content is understood to be recomputable/regenerable
		// by its own producing pipeline on every contribution (e.g. a
		// derived score, or a resynthesized paragraph of prose), so a
		// divergence there is the expected steady state, not a genuine
		// conflict between two writers.
		merged, conflicts := mergeCore(existing, incoming, sourceID, false, true, true)
		return merged, conflicts, nil
	case MergeUnionFirstWriter:
		merged, conflicts := mergeCore(existing, incoming, sourceID, true, true, false)
		return merged, conflicts, nil
	case MergeAppend:
		// Identical to MergeUnion's multi-valued field/edge union (BUG-002,
		// spec.md FR-022): a domain/extension kind registered with append
		// is ordinary patch-carried content, not the format's own excluded
		// timeline index (CORE §12.2), so it must be mergeable. CORE §10
		// describes append purely as ordered-list insertion ("keyed for
		// uniqueness so re-insertion is a no-op") with no concept of a
		// single authoritative scalar value, so — unlike MergeUnion — no
		// scalar field is ever flagged as a conflict.
		merged, _ := mergeCore(existing, incoming, sourceID, false, false, false)
		return merged, nil, nil
	case MergeValidatedOverwrite:
		// Multi-valued fields union exactly as MergeUnion; every scalar
		// field is first-writer-wins with conflicts never flagged, since
		// CORE §10 reserves overwriting a validation-owned scalar for an
		// optional validation pass this feature does not implement
		// (research.md D6) — no domain-profile schema exists yet to say
		// which fields are validation-owned.
		merged, _ := mergeCore(existing, incoming, sourceID, true, false, false)
		return merged, nil, nil
	default:
		return Node{}, nil, ErrUnknownMergeOp.With(errNoCause, string(op))
	}
}

// mergeCore merges existing/incoming per one MergeOp's (fillEmpty,
// flagConflicts) rule pair. unionText is true only for MergeUnion
// (BUG-004, spec.md FR-023/FR-024): it routes every Texts key other than
// "notes" through the paragraph-level mergeText instead of a scalar
// compare-or-flag — "notes" is unaffected either way.
func mergeCore(existing, incoming Node, sourceID string, fillEmpty, flagConflicts, unionText bool) (Node, []string) {
	merged := existing

	merged.Published = mergePublished(existing.Published, incoming.Published)

	var conflicts []string
	merged.Texts, conflicts = mergeTexts(existing.Texts, incoming.Texts, sourceID, fillEmpty, flagConflicts, unionText)

	// A union kind's Attrs are never flagged (BUG-004, spec.md FR-023/
	// FR-024), same as its Texts — attrsFlagConflicts mirrors the pre-
	// feature scalar-Attrs rule exactly (attrsFlagConflicts := flagConflicts
	// && !unionText).
	attrsFlagConflicts := flagConflicts && !unionText
	var attrConflicts []string
	merged.Attrs, attrConflicts = mergeAttrs(existing.Attrs, incoming.Attrs, sourceID, fillEmpty, attrsFlagConflicts)
	conflicts = append(conflicts, attrConflicts...)

	merged.Edges = unionLinks(existing.Edges, incoming.Edges)
	merged.HRefs = unionLinks(existing.HRefs, incoming.HRefs)

	return merged, conflicts
}

// paragraphSimilarityThreshold is the Jaccard-over-3-word-shingles score
// above which an incoming paragraph is considered a near-duplicate of an
// existing one (BUG-004, research.md D6 Bugfix) — a common, reasonable
// default for near-duplicate paragraph detection, not user-configurable.
const paragraphSimilarityThreshold = 0.8

// paragraphShingleSize is the shingle (word n-gram) width mergeText
// compares paragraphs by.
const paragraphShingleSize = 3

// mergeTexts merges the Texts map key-by-key over the union of both nodes'
// keys (data-model.md "Relationships / Lifecycle" > "Merge"). Every key
// literally named "notes" always goes through the scalar mergeScalarInto
// path, regardless of unionText; every other key goes through mergeText's
// paragraph-level reconciliation when unionText, or mergeScalarInto
// otherwise. A key present on only one side behaves like a scalar merge
// against "" on the other.
func mergeTexts(existing, incoming map[string]string, sourceID string, fillEmpty, flagConflicts, unionText bool) (map[string]string, []string) {
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
	for _, k := range sortedKeys {
		ev := existing[k]
		iv := incoming[k]

		if unionText && k != "notes" {
			merged[k] = mergeText(ev, iv)
			continue
		}

		merged[k], conflicts = mergeScalarInto(ev, iv, k, sourceID, fillEmpty, flagConflicts, conflicts)
	}

	return merged, conflicts
}

// mergeText reconciles a "union" kind's body prose paragraph-by-paragraph
// (spec.md FR-024, research.md D6 Bugfix — BUG-004) instead of as one
// scalar: entity-shaped Text is routinely regenerated wholesale by its
// producing pipeline, so comparing it as a single value guarantees a
// false-positive conflict on nearly every re-application. Splitting on
// paragraph boundaries and dropping any incoming paragraph that is a
// near-duplicate of one already present keeps the reconciliation additive
// and silent — never flagged, never lossy on either side.
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

func mergeScalarInto(existingVal, incomingVal, field, sourceID string, fillEmpty, flagConflicts bool, conflicts []string) (string, []string) {
	merged, diverges := mergeScalarString(existingVal, incomingVal, fillEmpty)
	if !diverges {
		return merged, conflicts
	}
	if !flagConflicts {
		return existingVal, conflicts
	}
	return conflictMarker(existingVal, incomingVal, sourceID), append(conflicts, field)
}

// mergeScalarString implements CORE §10's scalar rule: incoming identical
// or empty changes nothing; an empty existing is filled only when
// fillEmpty (MergeUnionFirstWriter/MergeValidatedOverwrite); any other
// divergence is reported via the second return value, for the caller to
// decide whether to flag it.
func mergeScalarString(existingVal, incomingVal string, fillEmpty bool) (merged string, diverges bool) {
	switch {
	case incomingVal == "" || incomingVal == existingVal:
		return existingVal, false
	case existingVal == "":
		if fillEmpty {
			return incomingVal, false
		}
		return existingVal, false
	default:
		return existingVal, true
	}
}

// conflictMarker reproduces VISION.md's documented git-style conflict
// marker (research.md D7). existing's own original writer cannot be
// determined from a Node value alone (no per-field provenance is stored),
// so the left side always falls back to the literal token "existing".
func conflictMarker(existingVal, incomingVal, incomingSourceID string) string {
	return fmt.Sprintf("<<<<<<< existing\n%s\n=======\n%s\n>>>>>>> %s", existingVal, incomingVal, incomingSourceID)
}

// mergeAttrs merges the Attrs map key-by-key over the union of keys
// (data-model.md "Relationships / Lifecycle" > "Merge", ast-contract.md
// "Merge"): a key present on only one side is taken unchanged. A key
// present on both sides that is multi-valued on either side (more than one
// Predicate) is merged as one list union, deduplicated per Predicate
// (research.md — this feature does not change per-predicate merge policy,
// only the shape it applies to) — a list union is inherently non-
// conflicting, matching this feature's own multi-valued attrs behavior
// unchanged from before it. A key that is single-valued on both sides
// (exactly one Predicate each) is merged through the same scalar
// fillEmpty/flagConflicts policy a bare scalar attribute used before this
// feature, so e.g. MergeUnionFirstWriter still flags a genuinely diverging
// single-valued attribute as a conflict, exactly as it did pre-feature.
func mergeAttrs(existing, incoming map[string][]Predicate, sourceID string, fillEmpty, flagConflicts bool) (map[string][]Predicate, []string) {
	merged := make(map[string][]Predicate, len(existing)+len(incoming))
	for k, v := range existing {
		merged[k] = v
	}

	keys := make(map[string]bool, len(incoming))
	for k := range incoming {
		keys[k] = true
	}
	sortedKeys := make([]string, 0, len(keys))
	for k := range keys {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	var conflicts []string
	for _, k := range sortedKeys {
		iv := incoming[k]
		ev, existed := existing[k]

		if !existed {
			merged[k] = iv
			continue
		}

		if len(ev) != 1 || len(iv) != 1 {
			merged[k] = unionPredicates(ev, iv)
			continue
		}

		mergedPred, diverges := mergeScalarPredicate(ev[0], iv[0], fillEmpty)
		if !diverges {
			merged[k] = []Predicate{mergedPred}
			continue
		}
		if !flagConflicts {
			merged[k] = ev
			continue
		}
		merged[k] = []Predicate{{Value: conflictMarker(fmt.Sprint(ev[0].Value), fmt.Sprint(iv[0].Value), sourceID)}}
		conflicts = append(conflicts, k)
	}

	return merged, conflicts
}

// mergeScalarPredicate implements CORE §10's scalar rule (pre-feature
// mergeAnyScalar, generalized to Predicate.Value): incoming nil or
// identical to existing changes nothing; an empty/nil existing is filled
// only when fillEmpty; any other divergence is reported via the second
// return value, for the caller to decide whether to flag it.
func mergeScalarPredicate(existing, incoming Predicate, fillEmpty bool) (merged Predicate, diverges bool) {
	if incoming.Value == nil || fmt.Sprint(incoming.Value) == fmt.Sprint(existing.Value) {
		return existing, false
	}
	if existing.Value == nil || fmt.Sprint(existing.Value) == "" {
		if fillEmpty {
			return incoming, false
		}
		return existing, false
	}
	return existing, true
}

// unionPredicates unions two Predicate lists, deduplicated by Value's
// fmt.Sprint representation, or by Target when Value is nil (mirrors the
// prior scalar-slice dedup-by-string-representation approach, applied
// per-Predicate), existing entries first.
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
// (AST §6.5), deduplicated, order-preserving, existing entries first.
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
