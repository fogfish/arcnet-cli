//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package kernel

import "time"

// Outcome is the terminal state of one planned patch in a batch run
// (specs/020-apply-batch/data-model.md). It is a string type so the --json
// form is self-describing.
//
// Files that are not patches at all (FR-003) are NOT Outcome values — they
// never enter the plan and are counted separately as BatchResult.NotAPatch.
type Outcome string

const (
	// OutcomeApplied — the patch applied and produced a commit (FR-006,
	// FR-007).
	OutcomeApplied Outcome = "applied"
	// OutcomeSkipped — the document was already tracked; no filesystem or
	// git change was made (FR-008).
	OutcomeSkipped Outcome = "skipped"
	// OutcomeFailed — the patch could not be applied; Reason records why
	// (FR-010, FR-020).
	OutcomeFailed Outcome = "failed"
	// OutcomeUnprocessed — the patch was never reached because --fail-fast
	// halted the run earlier (FR-011).
	OutcomeUnprocessed Outcome = "unprocessed"
)

// PatchOutcome is one entry per patch in the batch plan, in application
// order.
//
// Invariants: Reason is non-empty iff Outcome is OutcomeFailed; CommitHash
// is non-empty iff Outcome is OutcomeApplied; Created/Merged are empty for
// every outcome other than OutcomeApplied.
type PatchOutcome struct {
	// Path is the slash-separated path relative to the patch directory —
	// stable across platforms and the ordering tie-break key (FR-005).
	Path string `json:"path"`
	// Document is the source citekey from the patch manifest; empty when
	// the patch failed before its manifest decoded.
	Document string `json:"document"`
	// Published is the manifest publication date, the primary sort key
	// (FR-004). Zero when classification failed before the manifest
	// decoded — such an entry is ordered after every classified patch
	// (research.md D5b).
	Published time.Time `json:"published"`
	// Outcome is the patch's terminal state.
	Outcome Outcome `json:"outcome"`
	// Reason is the human-readable failure explanation, populated only for
	// OutcomeFailed (FR-013).
	Reason string `json:"reason,omitempty"`
	// CommitHash is the short hash of the commit this patch produced,
	// populated only for OutcomeApplied.
	CommitHash string `json:"commit,omitempty"`
	// Created holds node counts by type, newly created — carried through
	// from ApplyResult.
	Created map[string]int `json:"created,omitempty"`
	// Merged holds node counts by type, merged into existing nodes —
	// carried through from ApplyResult.
	Merged map[string]int `json:"merged,omitempty"`
}

// BatchResult is the run-level domain value component.go's ApplyBatch
// returns to cmd/arc/graph, rendered by bios.Registry[BatchResult].
//
// Invariants: Applied+Skipped+Failed+Unprocessed == len(Patches); NotAPatch
// counts files that never entered Patches and is deliberately not part of
// that sum; every slice field is non-nil so the --json shape is stable.
type BatchResult struct {
	// Directory is the patch directory exactly as the user supplied it.
	Directory string `json:"directory"`
	// Patches holds every planned patch, in application order.
	Patches []PatchOutcome `json:"patches"`
	// Applied counts OutcomeApplied entries.
	Applied int `json:"applied"`
	// Skipped counts OutcomeSkipped entries (FR-008).
	Skipped int `json:"skipped"`
	// Failed counts OutcomeFailed entries (FR-010).
	Failed int `json:"failed"`
	// Unprocessed counts OutcomeUnprocessed entries; always 0 unless
	// --fail-fast halted the run (FR-011).
	Unprocessed int `json:"unprocessed"`
	// NotAPatch counts Markdown files passed over because they do not
	// declare a patch manifest (FR-003).
	NotAPatch int `json:"not_a_patch"`
	// Conflicts is the union of every applied patch's flagged conflict
	// paths, de-duplicated and sorted (FR-016).
	Conflicts []string `json:"conflicts"`
	// Warnings is the union of every applied patch's warnings,
	// de-duplicated, in first-seen order (FR-017).
	Warnings []string `json:"warnings"`
}
