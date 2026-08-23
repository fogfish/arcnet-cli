//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package graph

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/spf13/cobra"

	"github.com/fogfish/arcnet-cli/internal/bios"
)

// batchTestdataRoot is resolved at package-variable initialization time —
// before any test calls chdir(t, ...) into a temporary graph, after which a
// relative "testdata" no longer resolves.
var batchTestdataRoot = func() string {
	abs, err := filepath.Abs("testdata")
	if err != nil {
		panic(err)
	}
	return abs
}()

func batchFixture(name string) string { return filepath.Join(batchTestdataRoot, name) }

// copyTree duplicates a read-only fixture bundle into a writable directory,
// for the scenarios that add or repair a patch between two runs.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, raw, 0o644)
	})
	it.Then(t).Should(it.Nil(err))
}

// batchCmd builds the command under test; sut invokes RunE directly, so
// --fail-fast is set through Flags().Set, mirroring revert_test.go's own
// --force precedent.
func batchCmd(t *testing.T, failFast bool) *cobra.Command {
	t.Helper()
	cmd := NewApplyBatchCmd()
	if failFast {
		it.Then(t).Should(it.Nil(cmd.Flags().Set("fail-fast", "true")))
	}
	return cmd
}

// writeInto writes one patch file beneath dir, creating parents as needed.
func writeInto(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(rel))
	it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Dir(full), 0o755)))
	it.Then(t).Should(it.Nil(os.WriteFile(full, []byte(content), 0o644)))
}

// commitSubjects returns every commit subject, oldest first.
func commitSubjects(t *testing.T, dir string) []string {
	t.Helper()
	out := strings.TrimSpace(runGit(t, dir, "log", "--reverse", "--pretty=%s"))
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}

func commitCount(t *testing.T, dir string) int {
	t.Helper()
	return len(commitSubjects(t, dir))
}

// indexOfSubjectContaining reports where the commit naming document sits in
// the oldest-first history, or -1 when absent.
func indexOfSubjectContaining(subjects []string, document string) int {
	for i, s := range subjects {
		if strings.Contains(s, document) {
			return i
		}
	}
	return -1
}

// patchWithDocument builds a minimal well-formed single-Source patch.
func patchWithDocument(document, published string) string {
	return "---\n\"@type\": patch\ndocument: " + document + "\npublished: " + published +
		"\ntitle: \"" + document + "\"\n---\n# Source\n\n## " + document + "\n" +
		"```yaml\n\"@id\": \"" + document + "\"\n\"@type\": Source\ntitle: \"" + document +
		"\"\nauthors: [Test Author]\npublished: \"" + published + "\"\n```\n\nA test patch.\n"
}

// ---------------------------------------------------------------------------
// User Story 1 — Ingest a whole corpus in one command
// ---------------------------------------------------------------------------

// arc apply batch testdata/batch
// Scenario 1 from specs/020-apply-batch/spec.md US1: every well-formed patch
// in the tree is applied and its contribution lands in the graph.
func TestBatchAppliesEveryDiscoveredPatch(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	_, err := sut(batchCmd(t, false), []string{batchFixture("batch")})

	// the fixture carries one deliberately broken patch, so the run exits 1
	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))

	assertIsFile(t, filepath.Join(dir, "sources", "mccarthy-2023-legacy.md"))
	assertIsFile(t, filepath.Join(dir, "sources", "rescorla-2024-tls13.md"))
	assertIsFile(t, filepath.Join(dir, "sources", "chen-2026-pqkex.md"))
	assertIsFile(t, filepath.Join(dir, "sources", "karpathy-2026-notes.md"))
	assertIsFile(t, filepath.Join(dir, "entities", "Key Agreement.md"))
	assertIsFile(t, filepath.Join(dir, "entities", "Transport Layer Security.md"))
	assertIsFile(t, filepath.Join(dir, "entities", "Sequence Model.md"))
}

// arc apply batch testdata/batch
// Scenario 2 from spec.md US1: application order follows the manifest
// publication date, not the filename or filesystem enumeration order — the
// fixture's alphabetical order deliberately contradicts its date order.
func TestBatchAppliesInPublicationOrderNotFilenameOrder(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	_, err := sut(batchCmd(t, false), []string{batchFixture("batch")})
	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))

	subjects := commitSubjects(t, dir)
	legacy := indexOfSubjectContaining(subjects, "mccarthy-2023-legacy")
	tls13 := indexOfSubjectContaining(subjects, "rescorla-2024-tls13")
	pqkex := indexOfSubjectContaining(subjects, "chen-2026-pqkex")
	karpathy := indexOfSubjectContaining(subjects, "karpathy-2026-notes")

	it.Then(t).
		Should(it.True(legacy > 0)).
		Should(it.True(legacy < tls13)).
		Should(it.True(tls13 < pqkex)).
		Should(it.True(pqkex < karpathy))
}

// arc apply batch testdata/batch
// Scenario 3 from spec.md US1: exactly one commit per applied patch, each
// carrying the same subject, stats, and Source-Id trailer a single-patch
// application produces, and no commit spanning two documents.
func TestBatchProducesOneCommitPerPatchWithSubjectAndTrailer(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	before := commitCount(t, dir)

	_, err := sut(batchCmd(t, false), []string{batchFixture("batch")})
	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))

	it.Then(t).Should(it.Equal(before+4, commitCount(t, dir)))

	documents := []string{
		"mccarthy-2023-legacy",
		"rescorla-2024-tls13",
		"chen-2026-pqkex",
		"karpathy-2026-notes",
	}
	trailers := runGit(t, dir, "log", "--pretty=%b")
	for _, doc := range documents {
		it.Then(t).Should(it.Equal(1, strings.Count(trailers, "Source-Id: "+doc)))
	}

	subjects := commitSubjects(t, dir)
	for _, subject := range subjects[1:] { // skip the graph(init) commit
		named := 0
		for _, doc := range documents {
			if strings.Contains(subject, doc) {
				named++
			}
		}
		it.Then(t).Should(it.Equal(1, named))
	}

	body := runGit(t, dir, "log", "-1", "--pretty=%b")
	it.Then(t).Should(it.String(body).Contain("Nodes:"))
}

// arc apply batch testdata/batch
// Scenario 4 from spec.md US1: patches nested in subdirectories are
// discovered too, and ordered globally rather than grouped per directory.
func TestBatchDiscoversNestedPatchesOrderedGlobally(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	_, err := sut(batchCmd(t, false), []string{batchFixture("batch")})
	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))

	// nested/deep/legacy.patch.md was found at all ...
	assertIsFile(t, filepath.Join(dir, "sources", "mccarthy-2023-legacy.md"))

	// ... and, despite living deepest in the tree, was applied first,
	// because its 2023 publication date is the oldest in the whole bundle.
	subjects := commitSubjects(t, dir)
	it.Then(t).Should(it.Equal(1, indexOfSubjectContaining(subjects, "mccarthy-2023-legacy")))
}

// arc apply batch testdata/batch
// Scenario 5 from spec.md US1: the closing summary reports applied, skipped,
// failed, and passed-over counts without a separate inspection step.
func TestBatchReportsSummaryCounts(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	out, err := sut(batchCmd(t, false), []string{batchFixture("batch")})
	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))

	// spec 021 T044: legacy-kind.patch.md adds one candidate that fails
	// recognition — failed +1, not_a_patch deliberately unchanged (FR-008).
	it.Then(t).
		Should(it.String(out).Contain("Applied 4 patches")).
		Should(it.String(out).Contain("0 skipped")).
		Should(it.String(out).Contain("2 failed")).
		Should(it.String(out).Contain("2 not a patch"))
}

// arc apply batch testdata/batch
// spec.md FR-015: per-patch progress is emitted to stderr as the run
// proceeds, and suppressed by --quiet while the summary still prints.
func TestBatchReportsPerPatchProgressUnlessQuiet(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	_, stderr, err := sutCaptureStderr(t, batchCmd(t, false), []string{batchFixture("batch")})
	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))
	it.Then(t).
		Should(it.String(stderr).Contain("mccarthy-2023-legacy")).
		Should(it.String(stderr).Contain("karpathy-2026-notes")).
		Should(it.String(stderr).Contain("applied"))

	quiet := t.TempDir()
	initGraph(t, quiet)
	chdir(t, quiet)
	bios.Quiet = true
	t.Cleanup(func() { bios.Quiet = false })

	out, stderrQuiet, err := sutCaptureStderr(t, batchCmd(t, false), []string{batchFixture("batch")})
	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))
	it.Then(t).
		ShouldNot(it.String(stderrQuiet).Contain("mccarthy-2023-legacy")).
		Should(it.String(out).Contain("Applied 4 patches"))
}

// ---------------------------------------------------------------------------
// User Story 2 — Re-run safely over a partly-ingested directory
// ---------------------------------------------------------------------------

// arc apply batch testdata/batch  (twice)
// Scenario 1 from spec.md US2: a fully-ingested directory re-runs as a
// no-op — no new commit, every patch reported as already tracked.
func TestBatchRerunOverIngestedDirectoryIsNoOp(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	_, err := sut(batchCmd(t, false), []string{batchFixture("batch")})
	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))
	afterFirst := commitCount(t, dir)

	out, err := sut(batchCmd(t, false), []string{batchFixture("batch")})
	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))

	it.Then(t).
		Should(it.Equal(afterFirst, commitCount(t, dir))).
		Should(it.String(out).Contain("Applied 0 patches")).
		Should(it.String(out).Contain("4 skipped")).
		Should(it.Equal("", strings.TrimSpace(runGit(t, dir, "status", "--porcelain"))))
}

// arc apply batch <copy of testdata/batch>  (twice, one patch added between)
// Scenario 2 from spec.md US2: only the not-yet-tracked document is applied
// on the second run, as its own commit.
func TestBatchRerunAppliesOnlyTheNewPatch(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	patches := t.TempDir()
	copyTree(t, batchFixture("batch"), patches)
	chdir(t, dir)

	_, err := sut(batchCmd(t, false), []string{patches})
	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))
	afterFirst := commitCount(t, dir)

	writeInto(t, patches, "2025/newcomer.patch.md", patchWithDocument("newcomer-2025-doc", "2025-09-09"))

	out, err := sut(batchCmd(t, false), []string{patches})
	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))

	it.Then(t).
		Should(it.Equal(afterFirst+1, commitCount(t, dir))).
		Should(it.String(out).Contain("Applied 1 patch")).
		Should(it.String(out).Contain("4 skipped"))
	assertIsFile(t, filepath.Join(dir, "sources", "newcomer-2025-doc.md"))
}

// arc apply batch testdata/batch
// Scenario 3 from spec.md US2: a run interrupted partway through resumes —
// the documents committed before the interruption are skipped and the rest
// are applied. The interruption is modelled by first batching a subtree.
func TestBatchResumesAfterAnInterruptedRun(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	// "the run got as far as nested/deep before it was killed"
	_, err := sut(batchCmd(t, false), []string{filepath.Join(batchFixture("batch"), "nested")})
	it.Then(t).Should(it.Nil(err))
	afterPartial := commitCount(t, dir)
	it.Then(t).Should(it.Equal(2, afterPartial))

	out, err := sut(batchCmd(t, false), []string{batchFixture("batch")})
	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))

	it.Then(t).
		Should(it.Equal(afterPartial+3, commitCount(t, dir))).
		Should(it.String(out).Contain("Applied 3 patches")).
		Should(it.String(out).Contain("1 skipped"))
}

// arc apply batch <two patches, one document>
// Scenario 4 from spec.md US2: the already-tracked check is evaluated when
// each patch is reached, so a duplicate later in the same run is skipped and
// the document ends up with exactly one commit (FR-009).
func TestBatchSkipsDuplicateDocumentWithinOneRun(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	patches := t.TempDir()
	writeInto(t, patches, "first.patch.md", patchWithDocument("dup-2025-doc", "2025-01-01"))
	writeInto(t, patches, "second.patch.md", patchWithDocument("dup-2025-doc", "2025-02-02"))
	chdir(t, dir)

	before := commitCount(t, dir)
	out, err := sut(batchCmd(t, false), []string{patches})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(before+1, commitCount(t, dir))).
		Should(it.String(out).Contain("Applied 1 patch")).
		Should(it.String(out).Contain("1 skipped")).
		Should(it.Equal(1, strings.Count(runGit(t, dir, "log", "--pretty=%b"), "Source-Id: dup-2025-doc")))
}

// ---------------------------------------------------------------------------
// User Story 3 — One bad patch does not abandon the rest
// ---------------------------------------------------------------------------

// arc apply batch testdata/batch
// Scenario 1 from spec.md US3: every valid patch is applied and committed
// despite the malformed one, which leaves no partial graph state and no
// dangling commit.
func TestBatchAppliesValidPatchesDespiteMalformedOne(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	before := commitCount(t, dir)
	_, err := sut(batchCmd(t, false), []string{batchFixture("batch")})
	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))

	it.Then(t).
		Should(it.Equal(before+4, commitCount(t, dir))).
		Should(it.Equal("", strings.TrimSpace(runGit(t, dir, "status", "--porcelain"))))

	_, statErr := os.Stat(filepath.Join(dir, "sources", "broken-2026-truncated.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

// arc apply batch testdata/batch
// Scenario 2 from spec.md US3: each failure is reported by file path with
// the reason, alongside the applied and skipped counts (FR-013).
func TestBatchReportsFailureByPathWithReason(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	out, err := sut(batchCmd(t, false), []string{batchFixture("batch")})
	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))

	it.Then(t).
		Should(it.String(out).Contain("failed:")).
		Should(it.String(out).Contain("broken/truncated.patch.md")).
		Should(it.String(out).Contain("manifest"))
}

// arc apply batch testdata/batch
// Scenario 3 from spec.md US3: a run with at least one failed patch exits
// non-zero, distinguishably from success (FR-014, research.md D8) — and the
// summary has already been printed, so no second error line is rendered.
func TestBatchExitsNonZeroWhenAnyPatchFailed(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	out, err := sut(batchCmd(t, false), []string{batchFixture("batch")})

	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))
	it.Then(t).
		Should(it.Equal("", err.Error())).
		Should(it.String(out).Contain("Applied 4 patches"))
}

// arc apply batch <directory of nothing but broken patches>
// Scenario 4 from spec.md US3: an all-failed run reports every failure and
// produces zero commits.
func TestBatchAllFailedRunProducesNoCommits(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	patches := t.TempDir()
	writeInto(t, patches, "one.patch.md", "---\n\"@type\": patch\ndocument: broken-one\n---\n# Source\n")
	writeInto(t, patches, "two.patch.md", "---\n\"@type\": patch\ndocument: broken-two\n---\n# Source\n")
	chdir(t, dir)

	before := commitCount(t, dir)
	out, err := sut(batchCmd(t, false), []string{patches})

	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))
	it.Then(t).
		Should(it.Equal(before, commitCount(t, dir))).
		Should(it.String(out).Contain("2 failed")).
		Should(it.String(out).Contain("one.patch.md")).
		Should(it.String(out).Contain("two.patch.md"))
}

// ---------------------------------------------------------------------------
// User Story 4 — Halt the batch at the first failure
// ---------------------------------------------------------------------------

// seedBlockerDirectory replaces the on-disk node file entities/Blocker.md
// with a directory, so testdata/batch-failfast's b-blocker.patch.md — which
// parses perfectly and therefore holds its real position in the publication
// order (research.md D5b) — fails while being applied rather than while
// being classified.
func seedBlockerDirectory(t *testing.T, dir string) {
	t.Helper()
	blocker := filepath.Join(dir, "entities", "Blocker.md")
	it.Then(t).Should(it.Nil(os.MkdirAll(blocker, 0o755)))
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(blocker, ".gitkeep"), nil, 0o644)))
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "seed: blocker")
}

// arc apply batch --fail-fast testdata/batch-failfast
// Scenario 1 from spec.md US4: the run stops at the failing patch — every
// patch ordered before it is committed, none ordered after it is applied.
func TestBatchFailFastHaltsAtFirstFailure(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedBlockerDirectory(t, dir)
	chdir(t, dir)

	before := commitCount(t, dir)
	_, err := sut(batchCmd(t, true), []string{batchFixture("batch-failfast")})
	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))

	it.Then(t).Should(it.Equal(before+1, commitCount(t, dir)))
	assertIsFile(t, filepath.Join(dir, "sources", "alpha-2025-early.md"))

	for _, absent := range []string{"beta-2025-blocker.md", "gamma-2025-late.md"} {
		_, statErr := os.Stat(filepath.Join(dir, "sources", absent))
		it.Then(t).Should(it.True(os.IsNotExist(statErr)))
	}
}

// arc apply batch --fail-fast testdata/batch-failfast
// Scenario 2 from spec.md US4: the halted run names the failing patch,
// reports how many patches were left unprocessed, and exits non-zero.
func TestBatchFailFastReportsUnprocessedCountAndExitsNonZero(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedBlockerDirectory(t, dir)
	chdir(t, dir)

	out, err := sut(batchCmd(t, true), []string{batchFixture("batch-failfast")})

	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))
	it.Then(t).
		Should(it.String(out).Contain("Applied 1 patch")).
		Should(it.String(out).Contain("1 failed")).
		Should(it.String(out).Contain("1 unprocessed")).
		Should(it.String(out).Contain("b-blocker.patch.md"))
}

// arc apply batch --fail-fast <copy of testdata/batch-failfast>
// Scenario 3 from spec.md US4: after the offending patch is repaired, the
// re-run skips the already-committed document and resumes from there.
func TestBatchFailFastResumesAfterRepair(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedBlockerDirectory(t, dir)
	chdir(t, dir)

	_, err := sut(batchCmd(t, true), []string{batchFixture("batch-failfast")})
	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))

	// repair: the node file is a real file again
	it.Then(t).Should(it.Nil(os.RemoveAll(filepath.Join(dir, "entities", "Blocker.md"))))
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "fix: blocker")
	afterRepair := commitCount(t, dir)

	out, err := sut(batchCmd(t, true), []string{batchFixture("batch-failfast")})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(afterRepair+2, commitCount(t, dir))).
		Should(it.String(out).Contain("Applied 2 patches")).
		Should(it.String(out).Contain("1 skipped")).
		Should(it.String(out).Contain("0 failed"))
	assertIsFile(t, filepath.Join(dir, "sources", "beta-2025-blocker.md"))
	assertIsFile(t, filepath.Join(dir, "sources", "gamma-2025-late.md"))
}

// arc apply batch testdata/batch-failfast   (no --fail-fast)
// FR-010 contrast to the scenario above: without the flag the same failing
// patch does not stop the run — the later patch is still applied.
func TestBatchWithoutFailFastContinuesPastTheSameFailure(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedBlockerDirectory(t, dir)
	chdir(t, dir)

	out, err := sut(batchCmd(t, false), []string{batchFixture("batch-failfast")})

	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))
	it.Then(t).
		Should(it.String(out).Contain("Applied 2 patches")).
		Should(it.String(out).Contain("1 failed"))
	assertIsFile(t, filepath.Join(dir, "sources", "gamma-2025-late.md"))
}

// ---------------------------------------------------------------------------
// Edge cases from spec.md
// ---------------------------------------------------------------------------

// arc apply batch testdata/batch
// Edge case: non-patch Markdown is passed over and counted, never applied
// and never a failure (FR-003).
func TestBatchPassesOverNonPatchMarkdown(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	out, err := sut(batchCmd(t, false), []string{batchFixture("batch")})
	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))

	it.Then(t).
		Should(it.String(out).Contain("2 not a patch")).
		ShouldNot(it.String(out).Contain("README.md")).
		ShouldNot(it.String(out).Contain("notes.md"))
}

// arc apply batch <directory holding one .bib and one patch>
// Edge case: a file that is not Markdown at all is ignored entirely — it
// appears in no count, not even not_a_patch.
func TestBatchIgnoresNonMarkdownFilesEntirely(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	patches := t.TempDir()
	writeInto(t, patches, "only.patch.md", patchWithDocument("only-2025-doc", "2025-03-03"))
	writeInto(t, patches, "refs.bib", "@article{x, title={y}}\n")
	writeInto(t, patches, "table.csv", "a,b\n1,2\n")
	chdir(t, dir)

	out, err := sut(batchCmd(t, false), []string{patches})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.String(out).Contain("Applied 1 patch")).
		Should(it.String(out).Contain("0 not a patch"))
}

// arc apply batch <empty dir>
// Edge case / FR-022: nothing to apply is a success that makes no change
// and says so plainly.
func TestBatchEmptyDirectorySucceeds(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	empty := t.TempDir()
	chdir(t, dir)

	before := commitCount(t, dir)
	out, err := sut(batchCmd(t, false), []string{empty})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(before, commitCount(t, dir))).
		Should(it.String(out).Contain("No patches to apply")).
		Should(it.Equal("", strings.TrimSpace(runGit(t, dir, "status", "--porcelain"))))
}

// arc apply batch <dir of nothing but non-patch markdown>
// FR-022 with a non-zero passed-over count and zero failures.
func TestBatchDirectoryOfOnlyNonPatchMarkdownSucceeds(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	patches := t.TempDir()
	writeInto(t, patches, "a.md", "# Just notes\n")
	writeInto(t, patches, "b.md", "---\ntitle: notes\n---\n# More notes\n")
	chdir(t, dir)

	out, err := sut(batchCmd(t, false), []string{patches})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.String(out).Contain("No patches to apply"))
}

// arc apply batch /does/not/exist
// Edge case / FR-002: a missing target path refuses the run before any
// patch is read, and makes no change.
func TestBatchRefusesMissingPath(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	before := commitCount(t, dir)
	out, err := sut(batchCmd(t, false), []string{filepath.Join(dir, "no-such-directory")})

	it.Then(t).Should(it.Error(out, err).Contain("no-such-directory"))
	it.Then(t).
		Should(it.Equal(before, commitCount(t, dir))).
		Should(it.Equal("", strings.TrimSpace(runGit(t, dir, "status", "--porcelain"))))
}

// arc apply batch ./some-file.md
// Edge case / FR-002: a file rather than a directory refuses the run.
func TestBatchRefusesFileInsteadOfDirectory(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	file := filepath.Join(batchFixture("batch"), "2024", "tls13.patch.md")

	before := commitCount(t, dir)
	out, err := sut(batchCmd(t, false), []string{file})

	it.Then(t).Should(it.Error(out, err).Contain("not a directory"))
	it.Then(t).Should(it.Equal(before, commitCount(t, dir)))
}

// cd /tmp/not-a-graph && arc apply batch testdata/batch
// Edge case / FR-002: an uninitialized graph refuses the run before any
// patch is read.
func TestBatchRefusesUninitializedGraph(t *testing.T) {
	notAGraph := t.TempDir()
	chdir(t, notAGraph)

	out, err := sut(batchCmd(t, false), []string{batchFixture("batch")})

	it.Then(t).Should(it.Error(out, err).Contain("not an initialized graph"))
}

// arc apply batch <two patches sharing one publication date>
// Edge case / FR-005: equal dates are broken by relative path, so two runs
// into two fresh graphs produce the identical commit sequence.
func TestBatchSameDateTieBreakIsDeterministic(t *testing.T) {
	patches := t.TempDir()
	writeInto(t, patches, "zulu.patch.md", patchWithDocument("zulu-2025-doc", "2025-07-07"))
	writeInto(t, patches, "alpha.patch.md", patchWithDocument("alpha-2025-doc", "2025-07-07"))
	writeInto(t, patches, "nested/mike.patch.md", patchWithDocument("mike-2025-doc", "2025-07-07"))

	var histories [][]string
	for range 2 {
		dir := t.TempDir()
		initGraph(t, dir)
		chdir(t, dir)

		_, err := sut(batchCmd(t, false), []string{patches})
		it.Then(t).Should(it.Nil(err))
		histories = append(histories, commitSubjects(t, dir))
	}

	it.Then(t).Should(it.Seq(histories[0]).Equal(histories[1]...))

	// alphabetical by relative path: alpha.patch.md, nested/mike.patch.md,
	// zulu.patch.md
	subjects := histories[0]
	it.Then(t).
		Should(it.True(indexOfSubjectContaining(subjects, "alpha-2025-doc") <
			indexOfSubjectContaining(subjects, "mike-2025-doc"))).
		Should(it.True(indexOfSubjectContaining(subjects, "mike-2025-doc") <
			indexOfSubjectContaining(subjects, "zulu-2025-doc")))
}

// arc apply batch testdata/batch
// Edge case / FR-019: a directory whose name begins with a dot is never
// descended into, so its well-formed patch is never applied.
func TestBatchNeverDescendsHiddenDirectories(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	out, err := sut(batchCmd(t, false), []string{batchFixture("batch")})
	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))

	_, statErr := os.Stat(filepath.Join(dir, "sources", "hidden-2025-ignored.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
	it.Then(t).ShouldNot(it.String(out).Contain("hidden-2025-ignored"))

	// but the same directory named explicitly as the target IS walked
	explicit := t.TempDir()
	initGraph(t, explicit)
	chdir(t, explicit)

	_, err = sut(batchCmd(t, false), []string{filepath.Join(batchFixture("batch"), ".hidden")})
	it.Then(t).Should(it.Nil(err))
	assertIsFile(t, filepath.Join(explicit, "sources", "hidden-2025-ignored.md"))
}

// arc apply batch <patch with no published date>
// Edge case / FR-020: an absent or uninterpretable publication date makes
// the patch a failure — never applied at a guessed position.
func TestBatchAbsentPublicationDateIsAFailure(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	patches := t.TempDir()
	writeInto(t, patches, "dated.patch.md", patchWithDocument("dated-2025-doc", "2025-04-04"))
	writeInto(t, patches, "undated.patch.md",
		"---\n\"@type\": patch\ndocument: undated-doc\ntitle: \"No date\"\n---\n# Source\n\n## undated-doc\n"+
			"```yaml\n\"@id\": \"undated-doc\"\n\"@type\": Source\n```\n\nNo publication date at all.\n")
	writeInto(t, patches, "nonsense.patch.md",
		"---\n\"@type\": patch\ndocument: nonsense-doc\npublished: \"not-a-date\"\ntitle: \"Bad date\"\n---\n# Source\n\n## nonsense-doc\n"+
			"```yaml\n\"@id\": \"nonsense-doc\"\n\"@type\": Source\n```\n\nAn uninterpretable date.\n")
	chdir(t, dir)

	out, err := sut(batchCmd(t, false), []string{patches})

	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))
	it.Then(t).
		Should(it.String(out).Contain("Applied 1 patch")).
		Should(it.String(out).Contain("2 failed")).
		Should(it.String(out).Contain("undated.patch.md")).
		Should(it.String(out).Contain("nonsense.patch.md"))

	for _, absent := range []string{"undated-doc.md", "nonsense-doc.md"} {
		_, statErr := os.Stat(filepath.Join(dir, "sources", absent))
		it.Then(t).Should(it.True(os.IsNotExist(statErr)))
	}
}

// arc apply batch <patch diverging from a seeded resource>
// Edge case: a patch that flags a merge conflict still counts as applied —
// a flagged conflict is a recorded outcome, not a failure — and the
// conflicted file is surfaced once in the final summary (FR-016).
func TestBatchConflictCountsAsAppliedAndIsSurfacedInSummary(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedNode(t, dir, "resources/RFC 8446.md", rfcResourceSeedSetStatus)
	patches := t.TempDir()
	writeInto(t, patches, "conflicting.patch.md", patchDivergesResourceStatus)
	writeInto(t, patches, "clean.patch.md", patchWithDocument("clean-2026-doc", "2026-06-06"))
	chdir(t, dir)

	before := commitCount(t, dir)
	out, err := sut(batchCmd(t, false), []string{patches})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(before+2, commitCount(t, dir))).
		Should(it.String(out).Contain("Applied 2 patches")).
		Should(it.String(out).Contain("0 failed")).
		Should(it.String(out).Contain("conflicts")).
		Should(it.String(out).Contain("RFC 8446.md"))
}

// arc apply batch <patch carrying an unregistered node type>
// Edge case / FR-017: an unregistered node kind produces a warning and does
// NOT abort the run — the batch behaves exactly as the single-patch command
// does, and the warning is aggregated into the run-level result.
func TestBatchUnregisteredKindWarnsWithoutAbortingTheRun(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	patches := t.TempDir()
	writeInto(t, patches, "hypothesis.patch.md",
		"---\n\"@type\": patch\ndocument: warn-2025-doc\npublished: 2025-02-02\ntitle: \"Unregistered kind\"\n---\n"+
			"# Source\n\n## warn-2025-doc\n```yaml\n\"@id\": \"warn-2025-doc\"\n\"@type\": Source\n```\n\nBody.\n\n"+
			"# Hypothesis\n\n## Some Hypothesis\n```yaml\n\"@id\": \"Some Hypothesis\"\n\"@type\": Hypothesis\n```\n\nA hypothesis.\n")
	writeInto(t, patches, "plain.patch.md", patchWithDocument("plain-2025-doc", "2025-03-03"))
	chdir(t, dir)

	before := commitCount(t, dir)
	out, stderr, err := sutCaptureStderr(t, batchCmd(t, false), []string{patches})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.Equal(before+2, commitCount(t, dir))).
		Should(it.String(out).Contain("Applied 2 patches")).
		Should(it.String(out).Contain("0 failed")).
		Should(it.String(stderr).Contain("Hypothesis is not a recognized node type"))
	assertIsFile(t, filepath.Join(dir, "Hypothesis", "Some Hypothesis.md"))
	assertIsFile(t, filepath.Join(dir, "_schema", "types", "Hypothesis.md"))
}

// arc apply batch <copy of testdata/batch>
// FR-021: the patch directory is read-only input — byte-identical after
// the run, with nothing moved, renamed, deleted, or rewritten.
func TestBatchLeavesPatchDirectoryUntouched(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	patches := t.TempDir()
	copyTree(t, batchFixture("batch"), patches)
	chdir(t, dir)

	before := snapshotTree(t, patches)

	_, err := sut(batchCmd(t, false), []string{patches})
	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))

	after := snapshotTree(t, patches)
	it.Then(t).Should(it.Seq(after).Equal(before...))
}

// snapshotTree records "<relative path>:<content length>:<content hash-ish>"
// for every file beneath root, sorted — enough to detect any move, rename,
// deletion, or rewrite.
func snapshotTree(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel)+":"+string(raw))
		return nil
	})
	it.Then(t).Should(it.Nil(err))
	return out
}

// ---------------------------------------------------------------------------
// --json output contract
// ---------------------------------------------------------------------------

// batchJSON mirrors contracts/batch-result.schema.md independently of the
// Go types under test, so the assertions below validate the documented wire
// shape rather than whatever the implementation happens to marshal.
type batchJSON struct {
	Directory string `json:"directory"`
	Patches   []struct {
		Path       string         `json:"path"`
		Document   string         `json:"document"`
		Published  string         `json:"published"`
		Outcome    string         `json:"outcome"`
		Reason     string         `json:"reason"`
		CommitHash string         `json:"commit"`
		Created    map[string]int `json:"created"`
		Merged     map[string]int `json:"merged"`
	} `json:"patches"`
	Applied     int      `json:"applied"`
	Skipped     int      `json:"skipped"`
	Failed      int      `json:"failed"`
	Unprocessed int      `json:"unprocessed"`
	NotAPatch   int      `json:"not_a_patch"`
	Conflicts   []string `json:"conflicts"`
	Warnings    []string `json:"warnings"`
}

// arc apply batch --json testdata/batch
// FR-018 / contracts/batch-result.schema.md: stdout carries only the JSON
// document, every documented field is present with the documented type, the
// counter invariants hold, and no slice serialises as null.
func TestBatchJSONMatchesDocumentedSchema(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	bios.JSON = true
	t.Cleanup(func() { bios.JSON = false })

	out, err := sut(batchCmd(t, false), []string{batchFixture("batch")})
	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))

	// stdout is only the JSON document — nothing else is piped alongside it
	var result batchJSON
	it.Then(t).Should(it.Nil(json.Unmarshal([]byte(out), &result)))

	it.Then(t).
		Should(it.Equal(batchFixture("batch"), result.Directory)).
		Should(it.Equal(4, result.Applied)).
		Should(it.Equal(0, result.Skipped)).
		Should(it.Equal(2, result.Failed)).
		Should(it.Equal(0, result.Unprocessed)).
		Should(it.Equal(2, result.NotAPatch)).
		Should(it.Equal(6, len(result.Patches)))

	// applied + skipped + failed + unprocessed == len(patches);
	// not_a_patch is deliberately outside that sum
	it.Then(t).Should(it.Equal(
		result.Applied+result.Skipped+result.Failed+result.Unprocessed,
		len(result.Patches)))

	// arrays are never null
	it.Then(t).
		Should(it.True(result.Conflicts != nil)).
		Should(it.True(result.Warnings != nil))

	// entries are in application order, oldest publication first, with the
	// classification failure appended last (research.md D5b)
	expectPaths := []string{
		"nested/deep/legacy.patch.md",
		"2024/tls13.patch.md",
		"2026/pqkex.patch.md",
		"2026/karpathy.patch.md",
		"broken/truncated.patch.md",
		// spec 021: a retired-key file is a classification failure too, so it
		// carries no usable date and is appended last, ordered by path.
		"legacy-kind.patch.md",
	}
	for i, want := range expectPaths {
		it.Then(t).Should(it.Equal(want, result.Patches[i].Path))
	}

	// per-outcome field invariants
	for _, p := range result.Patches {
		switch p.Outcome {
		case "applied":
			it.Then(t).
				Should(it.True(p.CommitHash != "")).
				Should(it.Equal("", p.Reason)).
				Should(it.True(p.Document != "")).
				Should(it.True(p.Published != "0001-01-01T00:00:00Z"))
		case "failed":
			it.Then(t).
				Should(it.True(p.Reason != "")).
				Should(it.Equal("", p.CommitHash)).
				Should(it.Equal(0, len(p.Created))).
				Should(it.Equal(0, len(p.Merged)))
		default:
			t.Errorf("unexpected outcome %q for %s", p.Outcome, p.Path)
		}
	}

	it.Then(t).
		Should(it.Equal("2023-08-05T00:00:00Z", result.Patches[0].Published)).
		Should(it.Equal("mccarthy-2023-legacy", result.Patches[0].Document)).
		Should(it.Equal("0001-01-01T00:00:00Z", result.Patches[4].Published))
}

// arc apply batch --json <empty dir>
// The FR-022 nothing-to-apply case still emits the full documented shape,
// with patches/conflicts/warnings as empty arrays rather than null.
func TestBatchJSONNothingToApplyEmitsEmptyArrays(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	empty := t.TempDir()
	chdir(t, dir)
	bios.JSON = true
	t.Cleanup(func() { bios.JSON = false })

	out, err := sut(batchCmd(t, false), []string{empty})
	it.Then(t).Should(it.Nil(err))

	it.Then(t).
		Should(it.String(out).Contain(`"patches": []`)).
		Should(it.String(out).Contain(`"conflicts": []`)).
		Should(it.String(out).Contain(`"warnings": []`))

	var result batchJSON
	it.Then(t).Should(it.Nil(json.Unmarshal([]byte(out), &result)))
	it.Then(t).
		Should(it.Equal(0, len(result.Patches))).
		Should(it.Equal(0, result.Failed))
}

// arc apply batch --json --fail-fast testdata/batch-failfast
// The unprocessed counter is reachable only under --fail-fast (FR-011).
func TestBatchJSONUnprocessedOnlyUnderFailFast(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedBlockerDirectory(t, dir)
	chdir(t, dir)
	bios.JSON = true
	t.Cleanup(func() { bios.JSON = false })

	out, err := sut(batchCmd(t, true), []string{batchFixture("batch-failfast")})
	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))

	var result batchJSON
	it.Then(t).Should(it.Nil(json.Unmarshal([]byte(out), &result)))

	it.Then(t).
		Should(it.Equal(1, result.Applied)).
		Should(it.Equal(1, result.Failed)).
		Should(it.Equal(1, result.Unprocessed)).
		Should(it.Equal(3, len(result.Patches))).
		Should(it.Equal("applied", result.Patches[0].Outcome)).
		Should(it.Equal("failed", result.Patches[1].Outcome)).
		Should(it.Equal("unprocessed", result.Patches[2].Outcome)).
		Should(it.Equal("", result.Patches[2].Reason)).
		Should(it.Equal("", result.Patches[2].CommitHash))
}

// ---------------------------------------------------------------------------
// spec 021 — patch manifest identity ("@type": patch)
// ---------------------------------------------------------------------------

// arc apply batch <dir>
// spec 021 US1 Acceptance Scenario 3: a directory whose every patch declares
// itself with the quoted "@type": patch key applies in full — in publication
// date order, ties broken by relative path — with one ingest commit each and
// nothing counted as passed over.
func TestBatchAppliesEveryTypeKeyPatchInDateThenPathOrder(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	patches := t.TempDir()
	// Alphabetical order deliberately contradicts date order, and the two
	// 2026-05-01 documents tie on date so only their relative path can
	// order them.
	writeInto(t, patches, "z-oldest.patch.md", patchWithDocument("aaa-2024-oldest", "2024-01-05"))
	writeInto(t, patches, "nested/b-tie.patch.md", patchWithDocument("bbb-2026-tie-b", "2026-05-01"))
	writeInto(t, patches, "a-tie.patch.md", patchWithDocument("ccc-2026-tie-a", "2026-05-01"))

	before := commitCount(t, dir)

	out, err := sut(batchCmd(t, false), []string{patches})
	it.Then(t).ShouldNot(it.Error(out, err))

	it.Then(t).Should(it.Equal(before+3, commitCount(t, dir)))

	subjects := commitSubjects(t, dir)
	oldest := indexOfSubjectContaining(subjects, "aaa-2024-oldest")
	tieA := indexOfSubjectContaining(subjects, "ccc-2026-tie-a")
	tieB := indexOfSubjectContaining(subjects, "bbb-2026-tie-b")

	it.Then(t).
		Should(it.True(oldest > 0)).
		Should(it.True(oldest < tieA)).
		// "a-tie.patch.md" sorts before "nested/b-tie.patch.md"
		Should(it.True(tieA < tieB))

	assertIsFile(t, filepath.Join(dir, "sources", "aaa-2024-oldest.md"))
	assertIsFile(t, filepath.Join(dir, "sources", "bbb-2026-tie-b.md"))
	assertIsFile(t, filepath.Join(dir, "sources", "ccc-2026-tie-a.md"))
}

// arc apply batch --json testdata/batch
// spec 021 US3 Acceptance Scenario 4 (FR-008, SC-005): legacy-kind.patch.md
// declares itself a patch under the retired key. It must be reported **by
// name** under `failed`, with the retired-key reason attached — and
// `not_a_patch` must be unchanged, since only README.md and notes.md declare
// no patch identity at all. That count assertion is the SC-005 test: a file
// the user meant as a patch is never silently skipped.
func TestBatchNamesRetiredKeyFileAsFailedNotPassedOver(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	bios.JSON = true
	t.Cleanup(func() { bios.JSON = false })

	out, err := sut(batchCmd(t, false), []string{batchFixture("batch")})
	it.Then(t).Should(it.True(errors.Is(err, bios.ErrSilent)))

	var result batchJSON
	it.Then(t).Must(it.Nil(json.Unmarshal([]byte(out), &result)))

	// README.md and notes.md only — the retired-key file is not among them
	it.Then(t).Should(it.Equal(2, result.NotAPatch))

	found := false
	for _, p := range result.Patches {
		if p.Path != "legacy-kind.patch.md" {
			continue
		}
		found = true
		it.Then(t).
			Should(it.Equal("failed", p.Outcome)).
			Should(it.String(p.Reason).Contain("retired")).
			Should(it.String(p.Reason).Contain(`"@type": patch`)).
			Should(it.Equal("", p.CommitHash))
	}
	it.Then(t).Should(it.True(found))

	// its Source node was never written — a named failure, not a partial apply
	_, statErr := os.Stat(filepath.Join(dir, "sources", "turing-2025-legacy-kind.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}
