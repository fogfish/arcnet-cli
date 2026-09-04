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

	_, statErr := os.Stat(filepath.Join(dir, "Source", "rescorla-2026-tls13.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
	_, statErr = os.Stat(filepath.Join(dir, "Entity", "Transport Layer Security.md"))
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

	_, statErr := os.Stat(filepath.Join(dir, "Source", "rescorla-2026-tls13.md"))
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
	assertIsFile(t, filepath.Join(dir, "Source", "rescorla-2026-tls13.md"))

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

	_, statErr = os.Stat(filepath.Join(dir, "Source", "rescorla-2026-tls13.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

const unrelatedNotePatchDifferentYear = `---
"@type": patch
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

	_, statErr := os.Stat(filepath.Join(dir, "Source", "rescorla-2026-tls13.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))

	after := readFile(t, filepath.Join(dir, "Hypothesis", "Forward Secrecy Requires Ephemeral Keys.md"))
	it.Then(t).Should(it.Equal(before, after))
}

const tlsEntityDocAPatch = `---
"@type": patch
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
- mentions:: [[TLS13]]

# Entity

## TLS13
` + "```yaml" + `
"@id": "TLS13"
"@type": Entity
category: [independent, abstract, occurrent, script]
` + "```" + `

A cryptographic protocol securing traffic between two endpoints.

## MentionedIn
- mentionedIn:: [[doc-2026-a]]

Introduced in RFC 8446.
`

const tlsEntityDocBPatch = `---
"@type": patch
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
- mentions:: [[TLS13]]

# Entity

## TLS13
` + "```yaml" + `
"@id": "TLS13"
"@type": Entity
category: [independent, abstract, occurrent, script]
tags: [deployed]
` + "```" + `

A cryptographic protocol securing traffic between two endpoints.

## MentionedIn
- mentionedIn:: [[doc-2026-b]]

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

	before := readFile(t, filepath.Join(dir, "Entity", "TLS13.md"))
	it.Then(t).
		Should(it.String(before).Contain("Introduced in RFC 8446.")).
		Should(it.String(before).Contain("Widely deployed by 2026."))

	out, err := sut(forcedRevertCmd(t), []string{"doc-2026-a"})
	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).Should(it.String(out).Contain("per-node"))

	assertIsFile(t, filepath.Join(dir, "Entity", "TLS13.md"))
	after := readFile(t, filepath.Join(dir, "Entity", "TLS13.md"))
	it.Then(t).
		ShouldNot(it.String(after).Contain("Introduced in RFC 8446.")).
		Should(it.String(after).Contain("Widely deployed by 2026.")).
		Should(it.String(after).Contain("deployed"))

	_, statErr := os.Stat(filepath.Join(dir, "Source", "doc-2026-a.md"))
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
		Should(it.String(stderr).Contain("Entity/TLS13.md")).
		Should(it.String(stderr).Contain("reconciled")).
		Should(it.String(stderr).Contain("paragraph"))
}

const resourcePatchForRevert = `---
"@type": patch
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
"@type": patch
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

	widgetBefore := readFile(t, filepath.Join(dir, "Entity", "Widget X.md"))
	it.Then(t).Should(it.String(widgetBefore).Contain("RFC 9999"))

	out, err := sut(forcedRevertCmd(t), []string{"doc-2026-r"})
	it.Then(t).ShouldNot(it.Error(out, err))

	_, statErr := os.Stat(filepath.Join(dir, "Resource", "RFC 9999.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))

	widgetAfter := readFile(t, filepath.Join(dir, "Entity", "Widget X.md"))
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

	assertIsFile(t, filepath.Join(dir, "Source", "rescorla-2026-tls13.md"))
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

// ---------------------------------------------------------------------------
// spec 021 — patch manifest identity ("@type": patch)
// ---------------------------------------------------------------------------

// arc apply docA.patch.md; arc apply docB.patch.md; arc revert doc-2026-a
// --force
// spec 021 T046/T047 (FR-009, research.md D6): docA's own patch file was
// picked up by its ingest commit's `git add -A`, so the per-node walk sees it
// among the reverted commit's touched paths. It is an exchange document, not
// a node — revert must skip it rather than remove or text-strip it — and it
// must do so under either identity key, because isPatchDocument now asks the
// single shared recognition rule (core.LooksLikePatch) instead of "ParsePatch
// succeeds". A retired-key file is precisely the case the old rule got wrong:
// ParsePatch rejects it, so it stopped being recognized as an exchange file
// and became eligible for node handling by a destructive command.
func TestRevertSkipsPatchDocumentLeftInGraphTree(t *testing.T) {
	for _, tt := range []struct {
		name     string
		retireIt bool
	}{
		{name: "current identity key", retireIt: false},
		{name: "retired identity key", retireIt: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			initGraph(t, dir)
			chdir(t, dir)

			patchA := writePatchFile(t, dir, "docA.patch.md", tlsEntityDocAPatch)
			_, err := sut(NewApplyCmd(), []string{patchA})
			it.Then(t).Must(it.Nil(err))

			patchB := writePatchFile(t, dir, "docB.patch.md", tlsEntityDocBPatch)
			_, err = sut(NewApplyCmd(), []string{patchB})
			it.Then(t).Must(it.Nil(err))

			want := tlsEntityDocAPatch
			if tt.retireIt {
				// the state a graph reaches when a pre-0.5 patch was left
				// behind in-tree by an older release
				want = strings.Replace(tlsEntityDocAPatch, `"@type": patch`, "kind: patch", 1)
				seedNode(t, dir, "docA.patch.md", want)
			}

			out, err := sut(forcedRevertCmd(t), []string{"doc-2026-a"})

			it.Then(t).ShouldNot(it.Error(out, err))
			it.Then(t).Should(it.String(out).Contain("per-node"))

			// the exchange file survives, byte-for-byte — revert never removed
			// or rewrote it as if it were a node
			assertIsFile(t, filepath.Join(dir, "docA.patch.md"))
			it.Then(t).Should(it.Equal(want, readFile(t, filepath.Join(dir, "docA.patch.md"))))

			// while the nodes it contributed are gone
			_, statErr := os.Stat(filepath.Join(dir, "Source", "doc-2026-a.md"))
			it.Then(t).Should(it.True(os.IsNotExist(statErr)))
		})
	}
}

// ---------------------------------------------------------------------------
// specs/022-reference-type-folders — ARCNET-CORE v0.11
// ---------------------------------------------------------------------------

// v011RevertDocA and v011RevertDocB both contribute prose to the same
// Resource and the same Reference. Reverting A therefore cannot take the
// whole-commit path: B has since touched both nodes, so revert must
// reconstruct each node's prose from the predicate the ingest wrote it to —
// text for the Resource, relevance for the Reference (contract C4).
const v011RevertDocA = `---
"@type": patch
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

# Resource

## handshake-fragment
` + "```yaml" + `
"@id": "handshake-fragment"
"@type": Resource
tags: [handshake]
` + "```" + `

Doc A's account of the handshake fragment.

- mentionedIn:: [[doc-2026-a]]

# Reference

## rfc-8446
` + "```yaml" + `
"@id": "rfc-8446"
"@type": Reference
title: "The TLS 1.3 Protocol"
ref: RFC 8446
` + "```" + `

Doc A's reason for keeping this pointer.
`

const v011RevertDocB = `---
"@type": patch
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

# Resource

## handshake-fragment
` + "```yaml" + `
"@id": "handshake-fragment"
"@type": Resource
tags: [retrospective]
` + "```" + `

Doc B's account of the handshake fragment.

- mentionedIn:: [[doc-2026-b]]

# Reference

## rfc-8446
` + "```yaml" + `
"@id": "rfc-8446"
"@type": Reference
title: "The TLS 1.3 Protocol"
ref: RFC 8446
` + "```" + `

## IsCitedBy
- isCitedBy:: [[doc-2026-b]]
`

// arc apply docA; arc apply docB; arc revert doc-2026-a --force
// spec.md US2 Acceptance Scenario 3: reverting the ingesting source
// reconstructs each node's prose from the same key the ingest wrote it
// with, and leaves nothing orphaned. Read-side (revertLeadingKey) and
// write-side (core.TextPredicateFor) must agree, or A's paragraph survives
// the revert under a key revert never looked at.
//
// The two halves are deliberately asymmetric in what each document
// contributes, not in how the keys merge — "text" (Resource's leading key)
// and "relevance" (Reference's leading key) both declare append. Both
// documents contribute an account of the Resource, so those accumulate and
// revert must strip exactly A's; docA is the sole contributor of the
// Reference's relevance note, so the assertion there is that reverting
// docA removes the note entirely instead of leaving it behind under a key
// revert never read.
func TestRevertReconstructsResourceAndReferenceProseFromTheirOwnKeys(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	patchA := writePatchFile(t, t.TempDir(), "docA.patch.md", v011RevertDocA)
	_, err := sut(NewApplyCmd(), []string{patchA})
	it.Then(t).Should(it.Nil(err))

	patchB := writePatchFile(t, t.TempDir(), "docB.patch.md", v011RevertDocB)
	_, err = sut(NewApplyCmd(), []string{patchB})
	it.Then(t).Should(it.Nil(err))

	resourcePath := filepath.Join(dir, "Resource", "handshake-fragment.md")
	referencePath := filepath.Join(dir, "Reference", "rfc-8446.md")

	beforeResource := readFile(t, resourcePath)
	beforeReference := readFile(t, referencePath)
	it.Then(t).
		Should(it.String(beforeResource).Contain("Doc A's account of the handshake fragment.")).
		Should(it.String(beforeResource).Contain("Doc B's account of the handshake fragment.")).
		Should(it.String(beforeReference).Contain("Doc A's reason for keeping this pointer."))

	out, err := sut(forcedRevertCmd(t), []string{"doc-2026-a"})
	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).Should(it.String(out).Contain("per-node"))

	afterResource := readFile(t, resourcePath)
	afterReference := readFile(t, referencePath)
	it.Then(t).
		ShouldNot(it.String(afterResource).Contain("Doc A's account of the handshake fragment.")).
		Should(it.String(afterResource).Contain("Doc B's account of the handshake fragment.")).
		ShouldNot(it.String(afterReference).Contain("Doc A's reason for keeping this pointer.")).
		Should(it.String(afterReference).Contain("doc-2026-b"))
}

// arc apply docA; arc apply docB; arc revert doc-2026-a --force
// spec.md US3 Acceptance Scenario 6: revert locates every node the patch
// created inside its type-named folder, removes the ones it exclusively
// owns, and rewrites the surviving referrers in place.
func TestRevertLocatesAndRewritesNodesInTypeNamedFolders(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	patchA := writePatchFile(t, t.TempDir(), "docA.patch.md", v011RevertDocA)
	_, err := sut(NewApplyCmd(), []string{patchA})
	it.Then(t).Should(it.Nil(err))

	patchB := writePatchFile(t, t.TempDir(), "docB.patch.md", v011RevertDocB)
	_, err = sut(NewApplyCmd(), []string{patchB})
	it.Then(t).Should(it.Nil(err))

	assertNodeAt(t, dir, "Source", "doc-2026-a")

	out, err := sut(forcedRevertCmd(t), []string{"doc-2026-a"})
	it.Then(t).ShouldNot(it.Error(out, err))

	// The exclusively-owned Source is gone from Source/, and the shared
	// nodes survive in their own type folders with doc-2026-a's backlink
	// swept out of each.
	sourceNames := dirEntryNames(t, filepath.Join(dir, "Source"))
	it.Then(t).
		ShouldNot(it.Seq(sourceNames).Contain("doc-2026-a.md")).
		Should(it.Seq(sourceNames).Contain("doc-2026-b.md"))

	assertNodeAt(t, dir, "Resource", "handshake-fragment")
	assertNodeAt(t, dir, "Reference", "rfc-8446")

	resource := readFile(t, filepath.Join(dir, "Resource", "handshake-fragment.md"))
	it.Then(t).
		ShouldNot(it.String(resource).Contain("[[doc-2026-a]]")).
		Should(it.String(resource).Contain("[[doc-2026-b]]"))
}

// --- BUG-002 / spec.md SC-010: revert inside a nested repository -----------

// initNestedGraph builds a graph at <repo>/graph inside a git repository
// rooted at repo, with a sibling <repo>/other that no graph-scoped command
// may ever reach. Unlike initGraph, the graph directory has no .git of its
// own — the whole point of the fixture (BUG-002, T050).
func initNestedGraph(t *testing.T, repo string) (graph, other string) {
	t.Helper()
	graph = filepath.Join(repo, "graph")
	other = filepath.Join(repo, "other")
	it.Then(t).Should(it.Nil(os.MkdirAll(graph, 0o755)))
	it.Then(t).Should(it.Nil(os.MkdirAll(other, 0o755)))

	writeGraphLayout(t, graph)
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(other, "unrelated.md"), []byte("unrelated"), 0o644)))

	runGit(t, repo, "init")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-m", "graph(init): empty knowledge graph")
	return graph, other
}

// spec.md SC-010 / FR-021: the per-node reconciliation path (FR-012/FR-013)
// must behave identically for a graph nested inside a larger repository, and
// the revert's own commit must contain no file from outside the graph. The
// fixture shares two nodes across two documents, so unpicking docA's
// contribution runs the D8(b) historical walk — the only path that calls
// ShowFile. Before BUG-002's fix every read there failed with "fatal: path
// 'graph/...' exists, but not '...'", which resolveConflictMarker
// propagates — aborting the revert outright.
func TestRevertReconcilesSharedNodeInNestedRepository(t *testing.T) {
	repo := t.TempDir()
	graph, other := initNestedGraph(t, repo)
	chdir(t, graph)

	patchA := writePatchFile(t, t.TempDir(), "docA.patch.md", v011RevertDocA)
	_, err := sut(NewApplyCmd(), []string{patchA})
	it.Then(t).Should(it.Nil(err))

	patchB := writePatchFile(t, t.TempDir(), "docB.patch.md", v011RevertDocB)
	_, err = sut(NewApplyCmd(), []string{patchB})
	it.Then(t).Should(it.Nil(err))

	// dirty the sibling: no graph-scoped command may sweep it in
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(other, "unrelated.md"), []byte("edited"), 0o644)))

	resourcePath := filepath.Join(graph, "Resource", "handshake-fragment.md")
	referencePath := filepath.Join(graph, "Reference", "rfc-8446.md")

	out, err := sut(forcedRevertCmd(t), []string{"doc-2026-a"})
	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).Should(it.String(out).Contain("per-node"))

	// identical outcome to the repository-root case
	afterResource := readFile(t, resourcePath)
	afterReference := readFile(t, referencePath)
	it.Then(t).
		ShouldNot(it.String(afterResource).Contain("Doc A's account of the handshake fragment.")).
		Should(it.String(afterResource).Contain("Doc B's account of the handshake fragment.")).
		ShouldNot(it.String(afterReference).Contain("Doc A's reason for keeping this pointer.")).
		Should(it.String(afterReference).Contain("doc-2026-b"))

	// SC-010: the revert's commit carries nothing from outside the graph
	changed := runGit(t, repo, "show", "--pretty=", "--name-only", "HEAD")
	for _, line := range strings.Split(strings.TrimSpace(changed), "\n") {
		if line == "" {
			continue
		}
		it.Then(t).Should(it.True(strings.HasPrefix(line, "graph/")))
	}
	it.Then(t).Should(it.True(!strings.Contains(changed, "other/")))
}
