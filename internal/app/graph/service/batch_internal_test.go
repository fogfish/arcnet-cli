//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// package service (white-box, not service_test): discovery, classification,
// and ordering are the batch use-case's pure, unexported stages — the whole
// point of placing them in the service layer is that they are testable
// without Cobra and without git (Principle III, research.md D2). Exercising
// them directly is the only way to assert the plan before any commit is
// produced. Mirrors this package's own revert_internal_test.go precedent.
package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/graph/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

func mountTree(t *testing.T, files map[string]string) fsys.Store {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Dir(full), 0o755)))
		it.Then(t).Should(it.Nil(os.WriteFile(full, []byte(content), 0o644)))
	}
	store, err := (fsys.Local{}).Mount(root)
	it.Then(t).Should(it.Nil(err))
	return store
}

func patchSource(document, published string) string {
	manifest := "---\n\"@type\": patch\ndocument: " + document + "\n"
	if published != "" {
		manifest += "published: " + published + "\n"
	}
	manifest += "title: \"" + document + "\"\n---\n# Source\n\n## " + document + "\n" +
		"```yaml\n\"@id\": \"" + document + "\"\n\"@type\": Source\n```\n\nBody.\n"
	return manifest
}

// ---------------------------------------------------------------------------
// Discovery (T021, research.md D3)
// ---------------------------------------------------------------------------

// FR-001: discovery recurses into subdirectories to any depth.
func TestDiscoverRecursesIntoSubdirectories(t *testing.T) {
	store := mountTree(t, map[string]string{
		"top.md":               "top",
		"one/mid.md":           "mid",
		"one/two/three/low.md": "low",
	})

	paths, err := discoverPatchFiles(store)

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Seq(paths).Equal("one/mid.md", "one/two/three/low.md", "top.md"))
}

// FR-019: a directory whose name begins with a dot is never descended into.
func TestDiscoverSkipsDotDirectories(t *testing.T) {
	store := mountTree(t, map[string]string{
		"kept.md":            "kept",
		".git/config.md":     "vcs",
		".hidden/inside.md":  "hidden",
		"deep/.cache/hid.md": "nested hidden",
		"deep/kept.md":       "kept too",
	})

	paths, err := discoverPatchFiles(store)

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Seq(paths).Equal("deep/kept.md", "kept.md"))
}

// Only regular files ending in .md are considered; everything else is
// ignored entirely and appears in no count.
func TestDiscoverIgnoresNonMarkdownFiles(t *testing.T) {
	store := mountTree(t, map[string]string{
		"a.md":       "markdown",
		"b.markdown": "not .md",
		"c.txt":      "text",
		"d.MD":       "wrong case",
		"e":          "no extension",
	})

	paths, err := discoverPatchFiles(store)

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Seq(paths).Equal("a.md"))
}

// research.md D3: fs.WalkDir over os.DirFS reports a symlink with
// ModeSymlink, so IsDir() is false and the walk never descends into it —
// a link cycle cannot make discovery run forever.
func TestDiscoverDoesNotDescendSymlinkedDirectories(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(outside, "linked.md"), []byte("x"), 0o644)))
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(root, "real.md"), []byte("y"), 0o644)))
	it.Then(t).Should(it.Nil(os.Symlink(outside, filepath.Join(root, "link"))))
	it.Then(t).Should(it.Nil(os.Symlink(root, filepath.Join(root, "self"))))

	store, err := (fsys.Local{}).Mount(root)
	it.Then(t).Should(it.Nil(err))

	paths, err := discoverPatchFiles(store)

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Seq(paths).Equal("real.md"))
}

// ---------------------------------------------------------------------------
// Classification (T022, research.md D4, FR-003, FR-020)
// ---------------------------------------------------------------------------

// FR-003: a Markdown file that declares no patch manifest is passed over —
// it produces no candidate at all and is never a failure.
func TestClassifyPassesOverFilesWithoutAPatchManifest(t *testing.T) {
	store := mountTree(t, map[string]string{
		"README.md":     "# Just a readme\n\nNo front matter at all.\n",
		"notes.md":      "---\ntitle: notes\ntags: [x]\n---\n\n# Notes\n",
		"real.patch.md": patchSource("real-2025-doc", "2025-01-01"),
	})
	paths, err := discoverPatchFiles(store)
	it.Then(t).Should(it.Nil(err))

	p := classifyPatchFiles(store, paths, core.Index{})

	it.Then(t).
		Should(it.Equal(2, p.notAPatch)).
		Should(it.Equal(1, len(p.candidates))).
		Should(it.Equal("real.patch.md", p.candidates[0].path)).
		Should(it.Equal("real-2025-doc", p.candidates[0].document)).
		Should(it.Nil(p.candidates[0].err))
}

// FR-020: a file that declares a patch identity ("@type": patch) but does
// not parse becomes a failed candidate — never a passed-over file.
func TestClassifyMakesUnparsablePatchAFailedCandidate(t *testing.T) {
	store := mountTree(t, map[string]string{
		"truncated.patch.md": "---\n\"@type\": patch\ndocument: broken-doc\npublished: 2025-01-01\n---\n# Source\n\n## broken-doc\n```yaml\n\"@id\": \"bro",
	})
	paths, err := discoverPatchFiles(store)
	it.Then(t).Should(it.Nil(err))

	p := classifyPatchFiles(store, paths, core.Index{})

	it.Then(t).
		Should(it.Equal(0, p.notAPatch)).
		Should(it.Equal(1, len(p.candidates))).
		Should(it.True(p.candidates[0].err != nil))
}

// FR-020: an absent publication date is a classification failure, so the
// patch is never applied at a guessed position in the order.
func TestClassifyMakesAbsentPublishedDateAFailedCandidate(t *testing.T) {
	store := mountTree(t, map[string]string{
		"undated.patch.md": patchSource("undated-doc", ""),
	})
	paths, err := discoverPatchFiles(store)
	it.Then(t).Should(it.Nil(err))

	p := classifyPatchFiles(store, paths, core.Index{})

	it.Then(t).
		Should(it.Equal(1, len(p.candidates))).
		Should(it.True(p.candidates[0].err != nil)).
		Should(it.True(p.candidates[0].published.IsZero()))
}

// FR-020: so is a published value that cannot be interpreted as a date.
func TestClassifyMakesUninterpretablePublishedDateAFailedCandidate(t *testing.T) {
	store := mountTree(t, map[string]string{
		"nonsense.patch.md": patchSource("nonsense-doc", "\"the third of never\""),
	})
	paths, err := discoverPatchFiles(store)
	it.Then(t).Should(it.Nil(err))

	p := classifyPatchFiles(store, paths, core.Index{})

	it.Then(t).
		Should(it.Equal(1, len(p.candidates))).
		Should(it.True(p.candidates[0].err != nil)).
		Should(it.True(p.candidates[0].published.IsZero()))
}

// ---------------------------------------------------------------------------
// Ordering (T023, research.md D5, D5b)
// ---------------------------------------------------------------------------

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func pathsOf(candidates []candidate) []string {
	out := make([]string, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, c.path)
	}
	return out
}

// FR-004: publication date ascending, oldest first, regardless of the order
// discovery happened to report the files in.
func TestOrderSortsByPublicationDateAscending(t *testing.T) {
	ordered := orderCandidates([]candidate{
		{path: "c.md", published: day("2026-04-02")},
		{path: "a.md", published: day("2023-08-05")},
		{path: "b.md", published: day("2024-03-11")},
	})

	it.Then(t).Should(it.Seq(pathsOf(ordered)).Equal("a.md", "b.md", "c.md"))
}

// FR-005: equal dates are broken deterministically by relative path, so
// repeated runs over unchanged input produce identical history.
func TestOrderBreaksEqualDatesByPath(t *testing.T) {
	same := day("2025-07-07")
	ordered := orderCandidates([]candidate{
		{path: "zulu.md", published: same},
		{path: "nested/mike.md", published: same},
		{path: "alpha.md", published: same},
	})

	it.Then(t).Should(it.Seq(pathsOf(ordered)).Equal("alpha.md", "nested/mike.md", "zulu.md"))
}

// research.md D5b: a classification failure carries the zero time.Time and
// would otherwise sort to the very front of the plan as an artefact of Go's
// zero value. It is appended last instead, ordered among its peers by path.
func TestOrderAppendsClassificationFailuresLast(t *testing.T) {
	ordered := orderCandidates([]candidate{
		{path: "zz-broken.md", err: errNoCause},
		{path: "dated.md", published: day("2025-01-01")},
		{path: "aa-broken.md", err: errNoCause},
		{path: "older.md", published: day("2023-01-01")},
	})

	it.Then(t).Should(it.Seq(pathsOf(ordered)).
		Equal("older.md", "dated.md", "aa-broken.md", "zz-broken.md"))
}

// ---------------------------------------------------------------------------
// Run-level aggregation and progress (FR-015, FR-016, FR-017)
// ---------------------------------------------------------------------------

// FR-016: conflicted paths are de-duplicated and sorted, so a path flagged
// by three patches is named once and the list reads the same every run.
func TestUnionSortedDeduplicatesAndSorts(t *testing.T) {
	out := unionSorted([]string{"Entity/b.md"}, []string{"Entity/a.md", "Entity/b.md"})
	out = unionSorted(out, []string{"Resource/c.md", "Entity/a.md"})

	it.Then(t).Should(it.Seq(out).Equal("Entity/a.md", "Entity/b.md", "Resource/c.md"))
}

// FR-017: warnings are de-duplicated but keep first-seen order, because
// their sequence carries the run's own narrative.
func TestUnionFirstSeenDeduplicatesKeepingOrder(t *testing.T) {
	out := unionFirstSeen(nil, []string{"zeta unregistered", "alpha unregistered"})
	out = unionFirstSeen(out, []string{"alpha unregistered", "mid unregistered"})

	it.Then(t).Should(it.Seq(out).
		Equal("zeta unregistered", "alpha unregistered", "mid unregistered"))
}

// An empty addition leaves the accumulator untouched — the common case for
// every patch that raises neither a conflict nor a warning.
func TestUnionFirstSeenOfNothingIsIdentity(t *testing.T) {
	base := []string{"only"}

	it.Then(t).Should(it.Seq(unionFirstSeen(base, nil)).Equal("only"))
}

// FR-015: one progress line per patch, naming the document where there is
// one and falling back to the path for a failure that never decoded a
// manifest to have a document identity.
func TestProgressLineNamesEachOutcome(t *testing.T) {
	it.Then(t).
		Should(it.Equal("doc-a: applied (commit abc1234)", progressLine(kernel.PatchOutcome{
			Outcome: kernel.OutcomeApplied, Document: "doc-a", CommitHash: "abc1234"}))).
		Should(it.Equal("doc-b: skipped, already tracked", progressLine(kernel.PatchOutcome{
			Outcome: kernel.OutcomeSkipped, Document: "doc-b"}))).
		Should(it.Equal("broken.patch.md: failed — bad manifest", progressLine(kernel.PatchOutcome{
			Outcome: kernel.OutcomeFailed, Path: "broken.patch.md", Reason: "bad manifest"}))).
		Should(it.Equal("later.patch.md: unprocessed", progressLine(kernel.PatchOutcome{
			Outcome: kernel.OutcomeUnprocessed, Path: "later.patch.md"})))
}

// data-model.md's counter invariant: the four outcome counters sum to
// len(Patches), and NotAPatch is never folded into that sum.
func TestRecordKeepsCounterInvariants(t *testing.T) {
	result := kernel.BatchResult{Conflicts: []string{}, Warnings: []string{}, NotAPatch: 7}

	record(&result, kernel.PatchOutcome{Outcome: kernel.OutcomeApplied}, []string{"a.md"}, []string{"warn"})
	record(&result, kernel.PatchOutcome{Outcome: kernel.OutcomeSkipped}, nil, nil)
	record(&result, kernel.PatchOutcome{Outcome: kernel.OutcomeFailed}, nil, nil)
	record(&result, kernel.PatchOutcome{Outcome: kernel.OutcomeUnprocessed}, nil, nil)

	it.Then(t).
		Should(it.Equal(1, result.Applied)).
		Should(it.Equal(1, result.Skipped)).
		Should(it.Equal(1, result.Failed)).
		Should(it.Equal(1, result.Unprocessed)).
		Should(it.Equal(7, result.NotAPatch)).
		Should(it.Equal(4, len(result.Patches))).
		Should(it.Equal(result.Applied+result.Skipped+result.Failed+result.Unprocessed, len(result.Patches))).
		Should(it.Seq(result.Conflicts).Equal("a.md")).
		Should(it.Seq(result.Warnings).Equal("warn"))
}

// The plan is a pure value: ordering the same input twice yields the same
// sequence, and the caller's slice is not mutated in place.
func TestOrderIsStableAndDoesNotMutateItsInput(t *testing.T) {
	input := []candidate{
		{path: "c.md", published: day("2026-04-02")},
		{path: "a.md", published: day("2023-08-05")},
	}

	first := orderCandidates(input)
	second := orderCandidates(input)

	it.Then(t).
		Should(it.Seq(pathsOf(first)).Equal(pathsOf(second)...)).
		Should(it.Seq(pathsOf(input)).Equal("c.md", "a.md"))
}
