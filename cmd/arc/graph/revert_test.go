//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package graph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/spf13/cobra"

	"github.com/fogfish/arcnet-cli/internal/bios"
)

// forcedRevertCmd builds arc revert with --force pre-set — sut() invokes
// RunE directly (bypassing Cobra's own flag-parsing pass), so a
// command-local flag (unlike the bios package-level Quiet/Verbose/JSON
// globals apply_test.go's own fixtures toggle directly) must be set via
// cmd.Flags().Set, mirroring subgraph_test.go's own --depth precedent.
func forcedRevertCmd(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := NewRevertCmd()
	it.Then(t).Should(it.Nil(cmd.Flags().Set("force", "true")))
	return cmd
}

// arc apply tls13.patch.md; arc revert rescorla-2026-tls13 --force
// spec.md US1 Acceptance Scenarios 1-3 / quickstart.md Scenario A: nothing
// has touched the ingest commit's own files since — takes the
// whole-commit path, removing every node the patch created and producing
// exactly one new commit.
func TestRevertWholeCommitRemovesJustAppliedPatch(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	_, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Nil(err))

	before := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))
	beforeCount := len(strings.Split(before, "\n"))

	out, err := sut(forcedRevertCmd(t), []string{"rescorla-2026-tls13"})
	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).Should(it.String(out).Contain("whole-commit"))

	after := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))
	afterCount := len(strings.Split(after, "\n"))
	it.Then(t).Should(it.Equal(beforeCount+1, afterCount))

	_, statErr := os.Stat(filepath.Join(dir, "sources", "rescorla-2026-tls13.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
	_, statErr = os.Stat(filepath.Join(dir, "entities", "Transport Layer Security.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

// arc apply tls13.patch.md; arc revert rescorla-2026-tls13 --force;
// arc apply tls13.patch.md; arc revert rescorla-2026-tls13 --force
// spec.md FR-020/SC-009 (Bugfix BUG-001, 2026-07-12): a document reverted
// once and then re-applied has two ingest commits in the graph's history
// carrying the same Source-Id trailer — the second revert must locate
// and act on the newer one rather than refusing with "more than one
// ingest commit found".
func TestRevertSucceedsAfterRetractReapplyCycle(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	_, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Nil(err))
	out, err := sut(forcedRevertCmd(t), []string{"rescorla-2026-tls13"})
	it.Then(t).ShouldNot(it.Error(out, err))

	_, statErr := os.Stat(filepath.Join(dir, "sources", "rescorla-2026-tls13.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))

	// The whole-commit revert (`git revert`) undid the entire ingest
	// commit, including tls13.patch.md itself (it was staged alongside
	// the node files by arc apply's own `git add -A`) — write it again
	// before re-applying it.
	patch = writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	// Re-apply the identical patch — its source node no longer exists,
	// so arc apply's own idempotency check does not block it, producing
	// a second, independent ingest commit with the same Source-Id
	// trailer.
	_, err = sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Nil(err))
	assertIsFile(t, filepath.Join(dir, "sources", "rescorla-2026-tls13.md"))

	before := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))
	beforeCount := len(strings.Split(before, "\n"))

	out, err = sut(forcedRevertCmd(t), []string{"rescorla-2026-tls13"})
	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).
		ShouldNot(it.String(out).Contain("more than one ingest commit")).
		Should(it.String(out).Contain("whole-commit"))

	after := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))
	afterCount := len(strings.Split(after, "\n"))
	it.Then(t).Should(it.Equal(beforeCount+1, afterCount))

	_, statErr = os.Stat(filepath.Join(dir, "sources", "rescorla-2026-tls13.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

const unrelatedNotePatchDifferentYear = `---
kind: patch
document: kolesnikov-2020-note
published: 2020-05-01
title: "A Working Note"
---
# Source

## kolesnikov-2020-note
` + "```yaml" + `
"@id": "kolesnikov-2020-note"
"@type": Source
title: "A Working Note"
authors: [Test Author]
published: "2020-05-01"
` + "```" + `

A short note.

# Hypothesis

## Forward Secrecy Requires Ephemeral Keys
` + "```yaml\n\"@id\": \"Forward Secrecy Requires Ephemeral Keys\"\n\"@type\": Hypothesis\n```" + `

A conclusion distilled from sources.
`

// arc apply tls13.patch.md; arc apply note.patch.md; arc revert
// rescorla-2026-tls13 --force
// spec.md US2 Acceptance Scenarios 1-2 / quickstart.md Scenario B: an
// unrelated later patch (no shared files — a different year, so it
// doesn't even share tls13's own timeline/yearly period file) does not
// disqualify the whole-commit path, and its own contribution is left
// byte-for-byte unchanged.
func TestRevertOlderNonOverlappingPatchStillTakesWholeCommitPath(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	// Each patch file is written just before its own apply, never both
	// upfront — a patch file still sitting in the graph root when a
	// prior apply's own `git add -A` runs would otherwise be swept into
	// that unrelated commit, corrupting its own ChangedPaths.
	patch1 := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)
	_, err := sut(NewApplyCmd(), []string{patch1})
	it.Then(t).Should(it.Nil(err))

	patch2 := writePatchFile(t, dir, "note.patch.md", unrelatedNotePatchDifferentYear)
	_, err = sut(NewApplyCmd(), []string{patch2})
	it.Then(t).Should(it.Nil(err))

	before := readFile(t, filepath.Join(dir, "Hypothesis", "Forward Secrecy Requires Ephemeral Keys.md"))

	out, err := sut(forcedRevertCmd(t), []string{"rescorla-2026-tls13"})
	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).Should(it.String(out).Contain("whole-commit"))

	_, statErr := os.Stat(filepath.Join(dir, "sources", "rescorla-2026-tls13.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))

	after := readFile(t, filepath.Join(dir, "Hypothesis", "Forward Secrecy Requires Ephemeral Keys.md"))
	it.Then(t).Should(it.Equal(before, after))
}

const tlsEntityDocAPatch = `---
kind: patch
document: doc-2026-a
published: 2026-04-01
title: "Document A"
---
# Source

## doc-2026-a
` + "```yaml" + `
"@id": "doc-2026-a"
"@type": Source
title: "Document A"
authors: [Author A]
published: "2026-04-01"
` + "```" + `

First document.

## Mentions
- mentions:: [[TLS 1.3]]

# Entity

## TLS 1.3
` + "```yaml" + `
"@id": "TLS 1.3"
"@type": Entity
category: [independent, abstract, occurrent, script]
` + "```" + `

Introduced in RFC 8446.
`

const tlsEntityDocBPatch = `---
kind: patch
document: doc-2026-b
published: 2026-04-02
title: "Document B"
---
# Source

## doc-2026-b
` + "```yaml" + `
"@id": "doc-2026-b"
"@type": Source
title: "Document B"
authors: [Author B]
published: "2026-04-02"
` + "```" + `

Second document.

## Mentions
- mentions:: [[TLS 1.3]]

# Entity

## TLS 1.3
` + "```yaml" + `
"@id": "TLS 1.3"
"@type": Entity
category: [independent, abstract, occurrent, script]
tags: [deployed]
` + "```" + `

Widely deployed by 2026.
`

// arc apply docA.patch.md; arc apply docB.patch.md; arc revert doc-2026-a
// --force
// spec.md US3 Acceptance Scenario 2 / quickstart.md Scenario C (the crux
// case): docB enriched the same entity docA created — reverting docA
// takes the per-node path, the entity survives, docA's own paragraph is
// gone, and docB's own paragraph and tags value are untouched.
func TestRevertNodeEnrichedByLaterPatchTakesPerNodePath(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patchA := writePatchFile(t, dir, "docA.patch.md", tlsEntityDocAPatch)
	_, err := sut(NewApplyCmd(), []string{patchA})
	it.Then(t).Should(it.Nil(err))

	patchB := writePatchFile(t, dir, "docB.patch.md", tlsEntityDocBPatch)
	_, err = sut(NewApplyCmd(), []string{patchB})
	it.Then(t).Should(it.Nil(err))

	before := readFile(t, filepath.Join(dir, "entities", "TLS 1.3.md"))
	it.Then(t).
		Should(it.String(before).Contain("Introduced in RFC 8446.")).
		Should(it.String(before).Contain("Widely deployed by 2026."))

	out, err := sut(forcedRevertCmd(t), []string{"doc-2026-a"})
	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).Should(it.String(out).Contain("per-node"))

	assertIsFile(t, filepath.Join(dir, "entities", "TLS 1.3.md"))
	after := readFile(t, filepath.Join(dir, "entities", "TLS 1.3.md"))
	it.Then(t).
		ShouldNot(it.String(after).Contain("Introduced in RFC 8446.")).
		Should(it.String(after).Contain("Widely deployed by 2026.")).
		Should(it.String(after).Contain("deployed"))

	_, statErr := os.Stat(filepath.Join(dir, "sources", "doc-2026-a.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

// arc apply docA.patch.md; arc apply docB.patch.md; arc revert
// doc-2026-a --verbose --force
// spec.md FR-019: --verbose reports a per-node reconciliation detail
// line.
func TestRevertVerboseReportsPerNodeReconciliationDetail(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patchA := writePatchFile(t, dir, "docA.patch.md", tlsEntityDocAPatch)
	_, err := sut(NewApplyCmd(), []string{patchA})
	it.Then(t).Should(it.Nil(err))

	patchB := writePatchFile(t, dir, "docB.patch.md", tlsEntityDocBPatch)
	_, err = sut(NewApplyCmd(), []string{patchB})
	it.Then(t).Should(it.Nil(err))

	bios.Verbose = true
	t.Cleanup(func() { bios.Verbose = false })

	_, stderr, err := sutCaptureStderr(t, forcedRevertCmd(t), []string{"doc-2026-a"})
	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.String(stderr).Contain("entities/TLS 1.3.md")).
		Should(it.String(stderr).Contain("reconciled")).
		Should(it.String(stderr).Contain("paragraph"))
}

const resourcePatchForRevert = `---
kind: patch
document: doc-2026-r
published: 2026-05-01
title: "Resource Document"
---
# Source

## doc-2026-r
` + "```yaml" + `
"@id": "doc-2026-r"
"@type": Source
title: "Resource Document"
authors: [Author R]
published: "2026-05-01"
` + "```" + `

A resource-contributing document.

## Cites
- cites:: [[RFC 9999]]

# Resource

## RFC 9999
` + "```yaml\n\"@id\": \"RFC 9999\"\n\"@type\": Resource\nref: standard\n```" + `

An exclusively-owned resource.
`

const referrerPatchForRevert = `---
kind: patch
document: doc-2026-s
published: 2026-05-02
title: "Referrer Document"
---
# Source

## doc-2026-s
` + "```yaml" + `
"@id": "doc-2026-s"
"@type": Source
title: "Referrer Document"
authors: [Author S]
published: "2026-05-02"
` + "```" + `

A document whose entity references the other patch's resource.

## Mentions
- mentions:: [[Widget X]]

# Entity

## Widget X
` + "```yaml\n\"@id\": \"Widget X\"\n\"@type\": Entity\ncategory: [independent]\n```" + `

A widget.
- relatesTo:: [[RFC 9999]]
`

// arc apply docR.patch.md; arc apply docS.patch.md; arc revert doc-2026-r
// --force
// spec.md US3 Acceptance Scenario 1 / research.md D6: removing an
// exclusively-owned node sweeps every referrer's backlink, including one
// contributed by an entirely different patch.
func TestRevertExclusiveNodeRemovalSweepsCrossPatchBacklink(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patchR := writePatchFile(t, dir, "docR.patch.md", resourcePatchForRevert)
	_, err := sut(NewApplyCmd(), []string{patchR})
	it.Then(t).Should(it.Nil(err))

	patchS := writePatchFile(t, dir, "docS.patch.md", referrerPatchForRevert)
	_, err = sut(NewApplyCmd(), []string{patchS})
	it.Then(t).Should(it.Nil(err))

	widgetBefore := readFile(t, filepath.Join(dir, "entities", "Widget X.md"))
	it.Then(t).Should(it.String(widgetBefore).Contain("RFC 9999"))

	out, err := sut(forcedRevertCmd(t), []string{"doc-2026-r"})
	it.Then(t).ShouldNot(it.Error(out, err))

	_, statErr := os.Stat(filepath.Join(dir, "resources", "RFC 9999.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))

	widgetAfter := readFile(t, filepath.Join(dir, "entities", "Widget X.md"))
	it.Then(t).ShouldNot(it.String(widgetAfter).Contain("RFC 9999"))
}

// arc revert unknown-2026-x --force
// spec.md FR-002 / Edge Cases: an unrecognized source-id refuses cleanly.
func TestRevertUnknownSourceIdRefuses(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	out, err := sut(forcedRevertCmd(t), []string{"unknown-2026-x"})

	it.Then(t).Should(it.Error(out, err).Contain("unknown-2026-x"))
}

// arc revert rescorla-2026-tls13 --force (run twice)
// spec.md FR-003 / SC-008, Clarifications Session 2026-07-12: an
// already-retracted document is a safe no-op on a second invocation.
func TestRevertAlreadyRetractedIsSafeNoOp(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	_, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Nil(err))
	_, err = sut(forcedRevertCmd(t), []string{"rescorla-2026-tls13"})
	it.Then(t).Should(it.Nil(err))

	before := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))

	out, err := sut(forcedRevertCmd(t), []string{"rescorla-2026-tls13"})
	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).Should(it.String(out).Contain("already retracted"))

	after := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))
	it.Then(t).Should(it.Equal(before, after))
}

// arc revert rescorla-2026-tls13 --force
// spec.md Edge Cases / FR-004: the target directory is not an initialized
// graph.
func TestRevertTargetNotAGraphRefuses(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	out, err := sut(forcedRevertCmd(t), []string{"rescorla-2026-tls13"})

	it.Then(t).Should(it.Error(out, err).Contain("initialized graph"))
}

// arc revert rescorla-2026-tls13 (no --force, non-interactive stdin)
// research.md D10 / quickstart.md Scenario F: the destructive-operation
// confirmation gate refuses rather than hanging or silently proceeding
// when stdin is not a terminal and --force is absent.
func TestRevertWithoutForceRefusesNonInteractively(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	_, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Nil(err))

	before := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))

	out, err := sut(NewRevertCmd(), []string{"rescorla-2026-tls13"})
	it.Then(t).Should(it.Error(out, err))

	after := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))
	it.Then(t).Should(it.Equal(before, after))

	assertIsFile(t, filepath.Join(dir, "sources", "rescorla-2026-tls13.md"))
}

// arc revert --json rescorla-2026-tls13 --force
// --json output contract (mirrors TestApplyJSONOutput's own precedent).
func TestRevertJSONOutput(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)
	_, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Nil(err))

	bios.JSON = true
	t.Cleanup(func() { bios.JSON = false })

	out, err := sut(forcedRevertCmd(t), []string{"rescorla-2026-tls13"})

	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).
		Should(it.String(out).Contain(`"document"`)).
		Should(it.String(out).Contain(`"approach"`)).
		Should(it.String(out).Contain(`"commit"`))
}
