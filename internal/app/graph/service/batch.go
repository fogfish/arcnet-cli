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
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fogfish/faults"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/graph/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/graph/port"
	"github.com/fogfish/arcnet-cli/internal/bios"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// arc apply batch's own preflight sentinels (research.md D13). Per-patch
// failure reasons are NOT errors — they are carried as strings inside
// kernel.BatchResult, because the run continues past them by design (D7).
// These three are the only conditions that refuse the run outright (FR-002).
const (
	ErrPatchDirNotFound   = faults.Safe1[string]("patch directory %s does not exist")
	ErrPatchDirNotADir    = faults.Safe1[string]("%s is not a directory")
	ErrPatchDirUnreadable = faults.Safe1[string]("failed to read patch directory %s")
)

// candidate is one discovered .md file after classification (data-model.md).
// A file for which core.LooksLikePatch returns false produces no candidate
// at all — it only increments the plan's notAPatch counter (FR-003).
type candidate struct {
	// path is slash-separated, relative to the patch directory.
	path string
	// document and published come from the decoded manifest; both are
	// zero-valued when classification failed.
	document  string
	published time.Time
	// err is non-nil when the file declares kind: patch but does not parse
	// (FR-020).
	err error
}

// plan is the ordered candidate list plus the passed-over count — the
// complete input to execution, fixed before the first commit (research.md D5).
type plan struct {
	candidates []candidate
	notAPatch  int
}

// ApplyBatch walks patchDir recursively, classifies every Markdown file it
// finds, orders the applicable patches by their manifest publication date,
// and applies each one through the unchanged Apply — one commit per patch
// (FR-006, FR-007). Already-tracked documents are skipped (FR-008), non-patch
// Markdown is passed over (FR-003), and per-patch failures are collected as
// data rather than raised as errors, so the run continues (FR-010) unless
// failFast halts it at the first one (FR-011).
//
// reporter carries Apply's own per-node detail (--verbose-gated by the
// caller); batchReporter carries one line per patch outcome, on unless
// --quiet (FR-015, research.md D9).
//
// A non-nil error is returned only for a preflight refusal (FR-002): the
// patch directory is missing, is not a directory, or the working directory
// is not an initialized graph — all evaluated before any patch is read.
func ApplyBatch(ctx context.Context, mounter fsys.Mounter, vcs port.VCS, reporter bios.Reporter, batchReporter bios.Reporter, index core.Index, schema port.SchemaRegistry, dir, patchDir string, failFast bool) (kernel.BatchResult, error) {
	patchStore, err := guardPatchDir(mounter, patchDir)
	if err != nil {
		return kernel.BatchResult{}, err
	}

	graphStore, err := mounter.Mount(dir)
	if err != nil {
		return kernel.BatchResult{}, err
	}
	if err := guardIsGraph(graphStore, dir); err != nil {
		return kernel.BatchResult{}, err
	}

	paths, err := discoverPatchFiles(patchStore)
	if err != nil {
		return kernel.BatchResult{}, ErrPatchDirUnreadable.With(err, patchDir)
	}

	p := classifyPatchFiles(patchStore, paths, index)
	ordered := orderCandidates(p.candidates)

	result := kernel.BatchResult{
		Directory: patchDir,
		Patches:   make([]kernel.PatchOutcome, 0, len(ordered)),
		NotAPatch: p.notAPatch,
		Conflicts: []string{},
		Warnings:  []string{},
	}

	run := batchRun{
		ctx: ctx, mounter: mounter, vcs: vcs, reporter: reporter,
		batchReporter: batchReporter, index: index, schema: schema,
		dir: dir, patchDir: patchDir, failFast: failFast,
	}
	run.execute(&result, ordered)

	return result, nil
}

// guardPatchDir implements the FR-002 preflight over the target path, in the
// contract's order: exists, then is a directory. It mounts the *parent* and
// stats the base name, because that is the only way through io/fs to tell
// "does not exist" apart from "exists but is a file" — os.DirFS over a file
// path reports the latter as an opaque read error.
func guardPatchDir(mounter fsys.Mounter, patchDir string) (fsys.Store, error) {
	parent, base := filepath.Dir(patchDir), filepath.Base(patchDir)
	if parent == patchDir || base == "." || base == string(filepath.Separator) {
		// patchDir is a filesystem root: it exists and is a directory by
		// construction, so there is nothing left to check.
		return mounter.Mount(patchDir)
	}

	probe, err := mounter.Mount(parent)
	if err != nil {
		return nil, ErrPatchDirUnreadable.With(err, patchDir)
	}

	info, err := probe.Stat(base)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, ErrPatchDirNotFound.With(err, patchDir)
	case err != nil:
		return nil, ErrPatchDirUnreadable.With(err, patchDir)
	case !info.IsDir():
		return nil, ErrPatchDirNotADir.With(errNoCause, patchDir)
	}

	return mounter.Mount(patchDir)
}

// discoverPatchFiles walks the mounted patch store, returning every regular
// .md file's slash-separated relative path in lexical order (FR-001).
// Directories whose name begins with a dot are never descended into
// (FR-019); a symlinked directory reports ModeSymlink rather than IsDir, so
// fs.WalkDir does not follow it either (research.md D3).
func discoverPatchFiles(store fsys.Store) ([]string, error) {
	out := []string{}
	err := fs.WalkDir(store, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != "." && strings.HasPrefix(entry.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if entry.Type().IsRegular() && strings.HasSuffix(entry.Name(), ".md") {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// classifyPatchFiles splits the discovered files into applicable patches and
// passed-over Markdown, reusing core.LooksLikePatch and core.ParsePatch so
// batch can never disagree with what arc apply itself accepts (research.md
// D4). A file that declares kind: patch but does not parse — including one
// whose published date is absent or uninterpretable — becomes a failed
// candidate, never a passed-over file (FR-003, FR-020).
func classifyPatchFiles(store fsys.Store, paths []string, index core.Index) plan {
	out := plan{candidates: make([]candidate, 0, len(paths))}

	for _, path := range paths {
		raw, err := fs.ReadFile(store, path)
		if err != nil {
			out.candidates = append(out.candidates, candidate{path: path, err: err})
			continue
		}

		if !core.LooksLikePatch(raw) {
			out.notAPatch++
			continue
		}

		patch, err := core.ParsePatch(bytes.NewReader(raw), index)
		if err != nil {
			out.candidates = append(out.candidates, candidate{path: path, err: err})
			continue
		}

		out.candidates = append(out.candidates, candidate{
			path:      path,
			document:  patch.Document,
			published: patch.Published,
		})
	}

	return out
}

// orderCandidates fixes the application order: publication date ascending,
// ties broken by relative path (FR-004, FR-005). Classification failures
// carry the zero time.Time and would otherwise sort to the very front as an
// artefact of Go's zero value, so they are appended last instead, ordered
// among themselves by path (research.md D5b). The input slice is not
// mutated.
func orderCandidates(candidates []candidate) []candidate {
	ordered := make([]candidate, len(candidates))
	copy(ordered, candidates)

	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if (a.err != nil) != (b.err != nil) {
			return b.err != nil
		}
		if a.err == nil && !a.published.Equal(b.published) {
			return a.published.Before(b.published)
		}
		return a.path < b.path
	})

	return ordered
}

// batchRun carries the invariant arguments of one run, so the execution loop
// below stays readable (Principle IV).
type batchRun struct {
	ctx           context.Context
	mounter       fsys.Mounter
	vcs           port.VCS
	reporter      bios.Reporter
	batchReporter bios.Reporter
	index         core.Index
	schema        port.SchemaRegistry
	dir           string
	patchDir      string
	failFast      bool
}

// execute walks the fixed plan, applying each candidate in turn. Under
// failFast the loop stops assigning real outcomes at the first failure and
// marks every remaining planned candidate unprocessed (FR-011); otherwise
// every candidate reaches a terminal outcome (FR-010).
func (run batchRun) execute(result *kernel.BatchResult, ordered []candidate) {
	halted := false

	for _, item := range ordered {
		if halted {
			record(result, kernel.PatchOutcome{
				Path:      item.path,
				Document:  item.document,
				Published: item.published,
				Outcome:   kernel.OutcomeUnprocessed,
			}, nil, nil)
			continue
		}

		outcome, conflicts, warnings := run.apply(item)
		record(result, outcome, conflicts, warnings)
		run.batchReporter.Step(progressLine(outcome))

		if outcome.Outcome == kernel.OutcomeFailed && run.failFast {
			halted = true
		}
	}
}

// apply reconstructs the candidate's absolute path and hands it to the
// unchanged Apply (FR-006, research.md D6), projecting the ApplyResult into
// one PatchOutcome. A per-patch error is returned as a failed outcome
// carrying its reason, never as a Go error (research.md D7).
func (run batchRun) apply(item candidate) (kernel.PatchOutcome, []string, []string) {
	outcome := kernel.PatchOutcome{
		Path:      item.path,
		Document:  item.document,
		Published: item.published,
	}

	if item.err != nil {
		outcome.Outcome = kernel.OutcomeFailed
		outcome.Reason = bios.Humanize(item.err)
		return outcome, nil, nil
	}

	patchPath := filepath.Join(run.patchDir, filepath.FromSlash(item.path))
	applied, err := Apply(run.ctx, run.mounter, run.vcs, run.reporter, run.index, run.schema, run.dir, patchPath)
	if err != nil {
		outcome.Outcome = kernel.OutcomeFailed
		outcome.Reason = bios.Humanize(err)
		return outcome, nil, nil
	}

	if applied.Skipped {
		outcome.Outcome = kernel.OutcomeSkipped
		return outcome, nil, nil
	}

	outcome.Outcome = kernel.OutcomeApplied
	outcome.CommitHash = applied.CommitHash
	outcome.Created = applied.Created
	outcome.Merged = applied.Merged
	return outcome, applied.Conflicts, applied.Warnings
}

// record appends one outcome and folds its counters, conflicts (FR-016) and
// warnings (FR-017) into the run-level result, preserving data-model.md's
// invariants: the four counters sum to len(Patches), and NotAPatch stays
// outside that sum.
func record(result *kernel.BatchResult, outcome kernel.PatchOutcome, conflicts, warnings []string) {
	result.Patches = append(result.Patches, outcome)

	switch outcome.Outcome {
	case kernel.OutcomeApplied:
		result.Applied++
	case kernel.OutcomeSkipped:
		result.Skipped++
	case kernel.OutcomeFailed:
		result.Failed++
	case kernel.OutcomeUnprocessed:
		result.Unprocessed++
	}

	result.Conflicts = unionSorted(result.Conflicts, conflicts)
	result.Warnings = unionFirstSeen(result.Warnings, warnings)
}

// progressLine is the one line the batch reporter emits per patch as it
// reaches its outcome (FR-015). A failed patch is named by path, because a
// classification failure has no document identity to name it with.
func progressLine(outcome kernel.PatchOutcome) string {
	switch outcome.Outcome {
	case kernel.OutcomeApplied:
		return fmt.Sprintf("%s: applied (commit %s)", outcome.Document, outcome.CommitHash)
	case kernel.OutcomeSkipped:
		return fmt.Sprintf("%s: skipped, already tracked", outcome.Document)
	case kernel.OutcomeFailed:
		return fmt.Sprintf("%s: failed — %s", outcome.Path, outcome.Reason)
	default:
		return fmt.Sprintf("%s: unprocessed", outcome.Path)
	}
}

// unionSorted folds add into base, de-duplicated and sorted — the shape
// FR-016 wants for conflicted file paths.
func unionSorted(base, add []string) []string {
	out := unionFirstSeen(base, add)
	sort.Strings(out)
	return out
}

// unionFirstSeen folds add into base, de-duplicated, preserving the order
// each value was first seen — the shape FR-017 wants for warnings, whose
// order carries the run's own narrative.
func unionFirstSeen(base, add []string) []string {
	if len(add) == 0 {
		return base
	}

	seen := make(map[string]bool, len(base)+len(add))
	for _, v := range base {
		seen[v] = true
	}

	out := base
	for _, v := range add {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}

	return out
}
