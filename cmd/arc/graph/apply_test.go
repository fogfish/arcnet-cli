//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package graph

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/spf13/cobra"

	"github.com/fogfish/arcnet-cli/cmd/arc/lint"
	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	appschema "github.com/fogfish/arcnet-cli/internal/app/schema"
	schemakernel "github.com/fogfish/arcnet-cli/internal/app/schema/kernel"
	"github.com/fogfish/arcnet-cli/internal/bios"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// TestMain sets a fake git identity for the whole test binary, matching
// cmd/arc/ctrl's own precedent — arc apply shells out to a real `git
// commit`.
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "arc-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "arc-test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "arc-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "arc-test@example.com")
	os.Exit(m.Run())
}

func sut(cmd *cobra.Command, args []string) (string, error) {
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ch := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		ch <- buf.String()
	}()

	err := cmd.RunE(cmd, args)
	if err == nil && cmd.PostRunE != nil {
		_ = cmd.PostRunE(cmd, args)
	}

	w.Close()
	os.Stdout = stdout
	return <-ch, err
}

func sutCaptureStderr(t *testing.T, cmd *cobra.Command, args []string) (stdout, stderr string, err error) {
	t.Helper()
	origStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	ch := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		ch <- buf.String()
	}()

	stdout, err = sut(cmd, args)

	w.Close()
	os.Stderr = origStderr
	stderr = <-ch
	return stdout, stderr, err
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	original, err := os.Getwd()
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Nil(os.Chdir(dir)))
	t.Cleanup(func() { os.Chdir(original) })
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	it.Then(t).Should(it.Nil(err))
	return string(out)
}

// initGraph builds a minimal, real, git-committed graph root — equivalent
// to arc init's own layout — without depending on cmd/arc/ctrl (which
// would otherwise perform a real network config-seed fetch cmd/arc/graph's
// tests must not depend on). _schema/ is seeded with appschema.Seed()'s own
// real output — the full CORE vocabulary, matching arc init's actual
// behavior — so Resolve's fail-fast validation never rejects this fixture.
func initGraph(t *testing.T, dir string) {
	t.Helper()
	writeGraphLayout(t, dir)

	runGit(t, dir, "init")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "graph(init): empty knowledge graph")
}

// writeGraphLayout writes the canonical folder layout and seeded _schema/
// into dir without touching git — the part initGraph and initNestedGraph
// (revert_test.go, BUG-002) share.
func writeGraphLayout(t *testing.T, dir string) {
	t.Helper()
	for _, folder := range []string{"Source", "Entity", "Resource", "timeline/yearly", "timeline/monthly", "_schema/Class", "_schema/Property"} {
		it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Join(dir, folder), 0o755)))
	}
	it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Join(dir, ".arc"), 0o755)))
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(dir, ".arc", ".gitkeep"), nil, 0o644)))
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".arc/\n"), 0o644)))

	for path, raw := range appschema.Seed() {
		full := filepath.Join(dir, filepath.FromSlash(path))
		it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Dir(full), 0o755)))
		it.Then(t).Should(it.Nil(os.WriteFile(full, raw, 0o644)))
	}
}

// seedNode writes and commits a node file directly, for merge-scenario
// fixtures that need a pre-existing node before the patch under test is
// applied.
func seedNode(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(relPath))
	it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Dir(full), 0o755)))
	it.Then(t).Should(it.Nil(os.WriteFile(full, []byte(content), 0o644)))
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "seed: "+relPath)
}

// seedSchemaNode writes _schema/Class/<kind>.md directly, registering kind
// with merge behavior op — equivalent to a prior arc apply's auto-discovery
// or a hand-edit (spec.md US2/US3), without writing a git commit of its own.
func seedSchemaNode(t *testing.T, dir, kind, op string) {
	t.Helper()
	full := filepath.Join(dir, "_schema", "Class", kind+".md")
	it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Dir(full), 0o755)))
	content := "---\n\"@id\": " + kind + "\n\"@type\": Class\nmerge: " + op + "\n---\n# " + kind + "\n\nA domain type registered by this test fixture.\n"
	it.Then(t).Should(it.Nil(os.WriteFile(full, []byte(content), 0o644)))
}

func writePatchFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	it.Then(t).Should(it.Nil(os.WriteFile(path, []byte(content), 0o644)))
	return path
}

func assertIsFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.True(!info.IsDir()))
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	it.Then(t).Should(it.Nil(err))
	return string(content)
}

const tls13Patch = `---
"@type": patch
document: rescorla-2026-tls13
published: 2026-04-12
title: "TLS 1.3: Design and Rationale"
---
# Source

## rescorla-2026-tls13
` + "```yaml" + `
"@id": "rescorla-2026-tls13"
"@type": Source
title: "TLS 1.3: Design and Rationale"
author: [Eric Rescorla]
published: "2026-04-12"
url: https://example.org/tls13-design
` + "```" + `

A design retrospective on the TLS 1.3 handshake.

## Mentions
- mentions:: [[Transport Layer Security]]

# Entity

## Transport Layer Security
` + "```yaml" + `
"@id": "Transport Layer Security"
"@type": Entity
category: [independent, abstract, occurrent, script]
` + "```" + `

A cryptographic protocol that establishes an authenticated, confidential channel.
`

// arc apply tls13.patch.md
// Scenario 1 from spec.md US1: creates a new file for every patch-carried node
func TestApplyCreatesNodesForNewDocument(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	out, err := sut(NewApplyCmd(), []string{patch})

	it.Then(t).ShouldNot(it.Error(out, err))
	assertIsFile(t, filepath.Join(dir, "Source", "rescorla-2026-tls13.md"))
	assertIsFile(t, filepath.Join(dir, "Entity", "Transport Layer Security.md"))
}

// arc apply tls13.patch.md
// Scenario 2 from spec.md US1: yearly/monthly timeline entries, chronological order
func TestApplyCreatesTimelineEntriesChronologically(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	laterPatch := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(
		tls13Patch, "rescorla-2026-tls13", "chen-2026-pqkex"),
		"2026-04-12", "2026-04-28"),
		"TLS 1.3: Design and Rationale", "Post-Quantum Key Exchange in Practice")
	patch1 := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)
	patch2 := writePatchFile(t, dir, "pqkex.patch.md", laterPatch)

	_, err := sut(NewApplyCmd(), []string{patch2})
	it.Then(t).Should(it.Nil(err))
	_, err = sut(NewApplyCmd(), []string{patch1})
	it.Then(t).Should(it.Nil(err))

	yearly := readFile(t, filepath.Join(dir, "timeline", "yearly", "2026.md"))
	monthly := readFile(t, filepath.Join(dir, "timeline", "monthly", "2026-04.md"))

	it.Then(t).
		Should(it.String(yearly).Contain("rescorla-2026-tls13")).
		Should(it.String(yearly).Contain("chen-2026-pqkex")).
		Should(it.String(monthly).Contain("rescorla-2026-tls13")).
		Should(it.String(monthly).Contain("chen-2026-pqkex"))

	// chronological: rescorla (04-12) must appear before chen (04-28)
	it.Then(t).Should(it.True(strings.Index(monthly, "rescorla-2026-tls13") < strings.Index(monthly, "chen-2026-pqkex")))

	// spec.md User Story 2 (T022): Timeline's entries list is the one
	// production type this feature's single-link-role-predicate-body
	// omission rule governs today — a period file's entries list must
	// never carry a "## " heading anywhere.
	it.Then(t).
		ShouldNot(it.String(yearly).Contain("## ")).
		ShouldNot(it.String(monthly).Contain("## "))
}

// BUG-002 regression: a full arc init -> arc apply -> arc lint round trip
// (using the reporter's own dmitry-2026-graph scenario shape) must produce
// zero [typeRequires]/[typeOptional] violations against the timeline period
// files arc apply generates — before this fix, every such file failed both
// checks (missing "entries"/"cites", unregistered "period").
func TestApplyGeneratedTimelinePeriodFilesPassLintCleanly(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)
	_, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Nil(err))

	lintOut, _ := sut(lint.NewLintCmd(), nil)

	it.Then(t).
		ShouldNot(it.String(lintOut).Contain("timeline/yearly/2026.md")).
		ShouldNot(it.String(lintOut).Contain("timeline/monthly/2026-04.md"))
}

// arc apply tls13.patch.md
// Scenario 3 from spec.md US1: exactly one new commit, subject + stats
func TestApplyProducesExactlyOneCommitWithStats(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	before := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))
	beforeCount := len(strings.Split(before, "\n"))

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	after := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))
	afterCount := len(strings.Split(after, "\n"))
	it.Then(t).Should(it.Equal(beforeCount+1, afterCount))

	subject := runGit(t, dir, "log", "-1", "--pretty=%s")
	body := runGit(t, dir, "log", "-1", "--pretty=%b")
	it.Then(t).
		Should(it.String(subject).Contain("rescorla-2026-tls13")).
		Should(it.String(body).Contain("Nodes:")).
		Should(it.String(body).Contain("Source-Id: rescorla-2026-tls13"))
}

// arc apply tls13.patch.md
// Scenario 4 from spec.md US1: reports counts created by kind
func TestApplyReportsCreatedCounts(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	out, err := sut(NewApplyCmd(), []string{patch})

	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).
		Should(it.String(out).Contain("+1 Source")).
		Should(it.String(out).Contain("+1 Entity")).
		Should(it.String(out).Contain("rescorla-2026-tls13"))
}

const tlsEntitySeed = `---
"@id": "Transport Layer Security"
"@type": Entity
title: Transport Layer Security
category: [independent, abstract, occurrent, script]
---
# Transport Layer Security

A cryptographic protocol.

- replaces:: [[SSL Protocol]]
`

const pqkexPatchMergingEntity = `---
"@type": patch
document: chen-2026-pqkex
published: 2026-04-28
title: "Post-Quantum Key Exchange in Practice"
---
# Source

## chen-2026-pqkex
` + "```yaml" + `
"@id": "chen-2026-pqkex"
"@type": Source
title: "Post-Quantum Key Exchange in Practice"
author: [Lin Chen]
published: "2026-04-28"
` + "```" + `

Surveys post-quantum key exchange deployment.

# Entity

## Transport Layer Security
` + "```yaml" + `
"@id": "Transport Layer Security"
"@type": Entity
category: [independent, abstract, occurrent, script]
` + "```" + `

A cryptographic protocol.
- requires:: [[Forward Secrecy]]
`

// arc apply pqkex.patch.md
// Scenario 1 from spec.md US2: re-introducing an existing entity unions
// its relations with no duplicate file
func TestApplyMergesExistingEntityUnion(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedNode(t, dir, "Entity/Transport Layer Security.md", tlsEntitySeed)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "pqkex.patch.md", pqkexPatchMergingEntity)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	entries, err := os.ReadDir(filepath.Join(dir, "Entity"))
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(entries)))

	content := readFile(t, filepath.Join(dir, "Entity", "Transport Layer Security.md"))
	it.Then(t).
		Should(it.String(content).Contain("replaces:: [[SSL Protocol]]")).
		Should(it.String(content).Contain("requires:: [[Forward Secrecy]]"))
}

// "status" (retired outright, BUG-001/FR-034) is replaced by "scoreZ" —
// still a real registered MergeLastWriteWin predicate, exercising the exact
// same "silently takes the newest value" semantics these fixtures test.
const rfcResourceSeedEmptyStatus = `---
"@id": "RFC 8446"
"@type": Resource
title: RFC 8446
scoreZ: ""
---
# RFC 8446

The normative specification of TLS 1.3.
`

const patchFillsResourceStatus = `---
"@type": patch
document: chen-2026-pqkex
published: 2026-04-28
title: "Post-Quantum Key Exchange in Practice"
---
# Source

## chen-2026-pqkex
` + "```yaml" + `
"@id": "chen-2026-pqkex"
"@type": Source
title: "Post-Quantum Key Exchange in Practice"
author: [Lin Chen]
published: "2026-04-28"
` + "```" + `

Surveys post-quantum key exchange deployment.

# Resource

## RFC 8446
` + "```yaml" + `
"@id": "RFC 8446"
"@type": Resource
scoreZ: read
` + "```" + `

The normative specification of TLS 1.3.
`

// arc apply pqkex.patch.md
// Scenario 2 from spec.md US2: a previously-empty resource field gets filled
func TestApplyMergeFillsEmptyResourceField(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedNode(t, dir, "Resource/RFC 8446.md", rfcResourceSeedEmptyStatus)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "pqkex.patch.md", patchFillsResourceStatus)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	content := readFile(t, filepath.Join(dir, "Resource", "RFC 8446.md"))
	it.Then(t).Should(it.String(content).Contain("scoreZ: read"))
}

const rfcResourceSeedSetStatus = `---
"@id": "RFC 8446"
"@type": Resource
title: RFC 8446
scoreZ: read
heading: normative
---
# RFC 8446

The normative specification of TLS 1.3.
`

const patchDivergesResourceStatus = `---
"@type": patch
document: chen-2026-pqkex
published: 2026-04-28
title: "Post-Quantum Key Exchange in Practice"
---
# Source

## chen-2026-pqkex
` + "```yaml" + `
"@id": "chen-2026-pqkex"
"@type": Source
title: "Post-Quantum Key Exchange in Practice"
author: [Lin Chen]
published: "2026-04-28"
` + "```" + `

Surveys post-quantum key exchange deployment.

# Resource

## RFC 8446
` + "```yaml" + `
"@id": "RFC 8446"
"@type": Resource
scoreZ: backlog
heading: informative
` + "```" + `

A survey of TLS 1.3 adoption patterns.
`

const llmEntitySeed = `---
"@id": "LLM"
"@type": Entity
title: LLM
score-c: 0.13432835820895522
score-z: 2.2522964920476682
---
# LLM

Large Language Models are technological systems that have fundamentally transformed approaches to ontologies graph construction and knowledge management.
`

// karpathyPatchRegeneratesEntity re-contributes to the same entity id with
// recomputed score-c/score-z (as a graph-analytics pass would produce on
// every re-ingest run) and Text carrying one near-duplicate paraphrase
// paragraph (only its last word differs) plus one genuinely new paragraph.
const karpathyPatchRegeneratesEntity = `---
"@type": patch
document: karpathy-2026-agentic
published: 2026-05-01
title: "Agentic Coding Workflows"
---
# Source

## karpathy-2026-agentic
` + "```yaml" + `
"@id": "karpathy-2026-agentic"
"@type": Source
title: "Agentic Coding Workflows"
author: [Andrej Karpathy]
published: "2026-05-01"
` + "```" + `

Discusses agentic coding workflows and their effect on software development.

## Mentions
- mentions:: [[LLM]]

# Entity

## LLM
` + "```yaml" + `
"@id": "LLM"
"@type": Entity
score-c: 0.28125
score-z: 2.8783652519773235
` + "```" + `

Large Language Models are technological systems that have fundamentally transformed approaches to ontologies graph construction and knowledge organization.

Andrej Karpathy has publicly argued that agentic coding workflows will reshape how software is written and reviewed.
`

// arc apply karpathy.patch.md
// BUG-001 (FR-018): score-c/score-z are unregistered custom Attrs keys, so
// they fall back to union and genuinely list-accumulate on every re-ingest
// (research.md D5c/D6, a documented, intentional behavior change from the
// old arity-based dispatch, which silently kept only the existing value).
//
// BUG-001 (round 2, FR-030) SUPERSEDES the conflict-flagging half of this
// test's original premise: "definition" (LLM's own leading prose) is
// retired under CORE 0.12 and replaced by "text" — Resource's own
// predicate, deliberately kept MergeAppend rather than firstWriteWin to
// avoid a merge-scoping mechanism the algebra doesn't otherwise have
// (plan.md F7/Complexity Tracking). So a reworded near-duplicate paragraph
// no longer diverges-and-flags; it takes the same paragraph-level
// accumulate-and-dedupe path BUG-004 restored for every append-declared
// prose key: the near-duplicate reword is silently dropped (kept: the
// original wording) and the genuinely new paragraph is appended — no
// conflict marker, because append never flags.
func TestApplyEntityReContributionFlagsProseAndAccumulatesUnregisteredScalars(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedNode(t, dir, "Entity/LLM.md", llmEntitySeed)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "karpathy.patch.md", karpathyPatchRegeneratesEntity)

	stdout, stderr, err := sutCaptureStderr(t, NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(stdout, err))
	it.Then(t).ShouldNot(it.String(stderr).Contain("merge conflict"))

	content := readFile(t, filepath.Join(dir, "Entity", "LLM.md"))
	it.Then(t).
		ShouldNot(it.String(content).Contain("<<<<<<<")).
		Should(it.String(content).Contain("knowledge management")).
		ShouldNot(it.String(content).Contain("knowledge organization")).
		Should(it.String(content).Contain("Andrej Karpathy")).
		Should(it.String(content).Contain("0.13432835820895522")).
		Should(it.String(content).Contain("0.28125")).
		Should(it.String(content).Contain("2.2522964920476682")).
		Should(it.String(content).Contain("2.8783652519773235"))
}

// arc apply pqkex.patch.md
// Scenario 3 from spec.md US2 / spec.md US2 Acceptance Scenario 1: a
// resource's firstWriteWin-declared "heading" is preserved and flagged
// on genuine divergence (FR-013 conflict marker); its lastWriteWin-declared
// "scoreZ" (BUG-001/FR-034 replaces the retired "status" fixture predicate,
// same LastWriteWin semantics) diverges too but is never flagged (FR-012)
// — takes the newest applied value instead; its append-declared leading
// prose ("text", FR-018/BUG-001) diverges too but is appended, never
// flagged; commit still completes.
//
// "heading" replaces "category" as this fixture's firstWriteWin exemplar:
// category now declares immutable (schema.go), which freezes the
// established value SILENTLY — the freeze class never flags — so it can no
// longer stand for a divergence that reaches human review.
func TestApplyMergePreservesSetFieldOnDivergence(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedNode(t, dir, "Resource/RFC 8446.md", rfcResourceSeedSetStatus)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "pqkex.patch.md", patchDivergesResourceStatus)

	before := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))
	beforeCount := len(strings.Split(before, "\n"))

	stdout, stderr, err := sutCaptureStderr(t, NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(stdout, err))

	after := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))
	afterCount := len(strings.Split(after, "\n"))
	it.Then(t).Should(it.Equal(beforeCount+1, afterCount))

	content := readFile(t, filepath.Join(dir, "Resource", "RFC 8446.md"))
	it.Then(t).
		Should(it.String(content).Contain("<<<<<<< existing")).
		Should(it.String(content).Contain("normative")).
		Should(it.String(content).Contain("informative")).
		Should(it.String(content).Contain("The normative specification of TLS 1.3.")).
		Should(it.String(content).Contain("A survey of TLS 1.3 adoption patterns.")).
		Should(it.String(content).Contain("scoreZ: backlog"))

	it.Then(t).Should(it.String(stderr).Contain("RFC 8446"))
}

// arc apply pqkex.patch.md
// Scenario 4 from spec.md US2: commit stats distinguish merged from created
func TestApplyCommitStatsDistinguishMergedFromCreated(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedNode(t, dir, "Entity/Transport Layer Security.md", tlsEntitySeed)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "pqkex.patch.md", pqkexPatchMergingEntity)

	out, err := sut(NewApplyCmd(), []string{patch})

	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).
		Should(it.String(out).Contain("+1 Source")).
		Should(it.String(out).Contain("+0 entities")).
		Should(it.String(out).Contain("1 merged"))
}

const notePatchWithHypothesis = `---
"@type": patch
document: kolesnikov-2026-note
published: 2026-05-01
title: "A Working Note"
---
# Source

## kolesnikov-2026-note
` + "```yaml" + `
"@id": "kolesnikov-2026-note"
"@type": Source
title: "A Working Note"
author: [Test Author]
published: "2026-05-01"
` + "```" + `

A short note.

# Hypothesis

## Forward Secrecy Requires Ephemeral Keys
` + "```yaml\n\"@id\": \"Forward Secrecy Requires Ephemeral Keys\"\n\"@type\": Hypothesis\n```" + `

A conclusion distilled from sources.
`

// arc apply note.patch.md
// Scenario 1 from spec.md US3: a registered domain kind is applied using
// its registered behavior — no warning
func TestApplyRegisteredKindUsesRegisteredBehaviorNoWarning(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedSchemaNode(t, dir, "Hypothesis", "validatedOverwrite")
	chdir(t, dir)
	patch := writePatchFile(t, dir, "note.patch.md", notePatchWithHypothesis)

	stdout, stderr, err := sutCaptureStderr(t, NewApplyCmd(), []string{patch})

	it.Then(t).ShouldNot(it.Error(stdout, err))
	it.Then(t).ShouldNot(it.String(stderr).Contain("not a recognized node type"))
	assertIsFile(t, filepath.Join(dir, "Hypothesis", "Forward Secrecy Requires Ephemeral Keys.md"))
}

// arc apply note.patch.md
// Scenario 2 from spec.md US3: an unregistered kind still applies (union
// default) with a warning
func TestApplyUnregisteredKindWarnsAndDefaultsUnion(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "note.patch.md", notePatchWithHypothesis)

	stdout, stderr, err := sutCaptureStderr(t, NewApplyCmd(), []string{patch})

	it.Then(t).ShouldNot(it.Error(stdout, err))
	it.Then(t).Should(it.String(stderr).Contain("Hypothesis"))
	assertIsFile(t, filepath.Join(dir, "Hypothesis", "Forward Secrecy Requires Ephemeral Keys.md"))
}

// arc apply note.patch.md
// Scenario 1 from spec.md US2: a patch introducing an unregistered kind
// creates its schema document, applies successfully using the union
// default, and the new schema document lands in the same commit as the
// triggering patch.
func TestApplyUnregisteredKindCreatesSchemaDocumentInSameCommit(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "note.patch.md", notePatchWithHypothesis)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	assertIsFile(t, filepath.Join(dir, "_schema", "Class", "Hypothesis.md"))
	content := readFile(t, filepath.Join(dir, "_schema", "Class", "Hypothesis.md"))
	it.Then(t).
		Should(it.String(content).Contain(`"@type": Class`)).
		// specs/023 FR-005: auto-registration no longer stamps a
		// type-level merge onto the document it creates.
		ShouldNot(it.String(content).Contain("merge:"))

	stat := runGit(t, dir, "show", "--stat", "HEAD")
	it.Then(t).
		Should(it.String(stat).Contain("_schema/Class/Hypothesis.md")).
		Should(it.String(stat).Contain("Forward Secrecy Requires Ephemeral Keys.md"))
}

// arc apply tls13.patch.md
// Scenario 2 from spec.md US2: a patch introducing an unregistered
// predicate creates its schema document, in the same commit. "supersedes"
// (quickstart.md Scenario 2) is used rather than a CORE §10 predicate like
// "mentions", since every one of those is already registered by
// initGraph's full appschema.Seed() output, matching a real arc init.
func TestApplyUnregisteredPredicateCreatesSchemaDocumentInSameCommit(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patchWithNovelPredicate := strings.ReplaceAll(tls13Patch, "mentions:: [[Transport Layer Security]]", "supersedes:: [[Legacy Protocol]]")
	patch := writePatchFile(t, dir, "tls13.patch.md", patchWithNovelPredicate)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	assertIsFile(t, filepath.Join(dir, "_schema", "Property", "supersedes.md"))

	stat := runGit(t, dir, "show", "--stat", "HEAD")
	it.Then(t).
		Should(it.String(stat).Contain("_schema/Property/supersedes.md")).
		Should(it.String(stat).Contain("Source/rescorla-2026-tls13.md"))
}

const hypothesisWithLabeledBlocksPatch = `---
"@type": patch
document: dmitry-2026-article
published: 2026-01-01
title: "A Test Article"
---
# Hypothesis

## Ontology-Driven Multi-Purpose Knowledge Representation
` + "```yaml\n\"@id\": \"Ontology-Driven Multi-Purpose Knowledge Representation\"\n\"@type\": Hypothesis\n```" + `

A hypothesis about ontology-driven representation.

**Assumptions**
- Ontologies are static once published
- Users prefer YAML front matter over JSON

**References**
- [RFC 8259](https://tools.ietf.org/html/rfc8259) - JSON specification
- [CommonMark](https://commonmark.org/) - Markdown specification

**Related Aporias**
- [[Some Aporia]]
`

// BUG-002 (spec 010 FR-019): end-to-end reproduction of the reported bug —
// applying a patch whose Hypothesis node carries "**Assumptions**" (plain
// prose) and "**References**" (standard markdown links, not wikilinks)
// blocks must no longer silently drop that content. Both survive in the
// written node file, and each label auto-registers a schema predicate with
// the role matching its actually-observed content shape.
func TestApplyLabeledBlockNonWikilinkContentSurvives(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "article.patch.md", hypothesisWithLabeledBlocksPatch)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	nodePath := filepath.Join(dir, "Hypothesis", "Ontology-Driven Multi-Purpose Knowledge Representation.md")
	assertIsFile(t, nodePath)
	content := readFile(t, nodePath)
	it.Then(t).
		Should(it.String(content).Contain("Ontologies are static once published")).
		Should(it.String(content).Contain("Users prefer YAML front matter over JSON")).
		Should(it.String(content).Contain("RFC 8259")).
		Should(it.String(content).Contain("CommonMark")).
		Should(it.String(content).Contain("[[Some Aporia]]"))

	assumptionsSchema := readFile(t, filepath.Join(dir, "_schema", "Property", "assumptions.md"))
	it.Then(t).
		Should(it.String(assumptionsSchema).Contain("role: text")).
		Should(it.String(assumptionsSchema).Contain("merge: append"))

	referencesSchema := readFile(t, filepath.Join(dir, "_schema", "Property", "references.md"))
	it.Then(t).
		Should(it.String(referencesSchema).Contain("role: text")).
		Should(it.String(referencesSchema).Contain("merge: append"))
}

const hypothesisWithFullShapePatch = `---
"@type": patch
document: dmitry-2026-article2
published: 2026-01-01
title: "A Second Test Article"
---
# Hypothesis

## Ontology-Driven Multi-Purpose Knowledge Representation 2
` + "```yaml\n\"@id\": \"Ontology-Driven Multi-Purpose Knowledge Representation 2\"\n\"@type\": Hypothesis\n```" + `

A hypothesis about ontology-driven representation.

**Assumptions**
- Core graph structure can be meaningfully separated from domain-specific semantics
- [[LLM]]s can be effectively trained on regulated node structures to maintain semantic consistency

**Related Aporias**
- [[Domain Overspecialization Limits Generalization]]

**Assumes**
- assumes:: [[LLM]]

**Derived From**
- derivedFrom:: [[dmitry-2026-article2]]
`

// BUG-003 (spec 010 FR-020/FR-021/FR-022): end-to-end reproduction of the
// full formatting regression report — applying a patch whose Hypothesis
// node carries a text-role "**Assumptions**" list (with a wikilink
// immediately followed by an inflectional suffix, "[[LLM]]s") and three
// distinctly labeled edge blocks ("**Related Aporias**", "**Assumes**",
// "**Derived From**") must survive with its *shape* intact on write: list
// markers and literal wikilink brackets preserved verbatim, each block's
// own heading recovered, and the three edge blocks staying separate
// distinct groups rather than collapsing into one flat list.
func TestApplyLabeledBlockShapeSurvivesWikilinksListMarkersHeadingsAndGrouping(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "article2.patch.md", hypothesisWithFullShapePatch)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	nodePath := filepath.Join(dir, "Hypothesis", "Ontology-Driven Multi-Purpose Knowledge Representation 2.md")
	assertIsFile(t, nodePath)
	content := readFile(t, nodePath)
	it.Then(t).
		Should(it.String(content).Contain("## Assumptions")).
		Should(it.String(content).Contain("- Core graph structure can be meaningfully separated from domain-specific semantics")).
		Should(it.String(content).Contain("- [[LLM]]s can be effectively trained on regulated node structures to maintain semantic consistency")).
		Should(it.String(content).Contain("## Related Aporias")).
		Should(it.String(content).Contain("[[Domain Overspecialization Limits Generalization]]")).
		Should(it.String(content).Contain("## Assumes")).
		Should(it.String(content).Contain("assumes:: [[LLM]]")).
		Should(it.String(content).Contain("## Derived From")).
		Should(it.String(content).Contain("derivedFrom:: [[dmitry-2026-article2]]"))

	assumptionsSchema := readFile(t, filepath.Join(dir, "_schema", "Property", "assumptions.md"))
	it.Then(t).Should(it.String(assumptionsSchema).Contain("role: text"))

	relatedAporiasSchema := readFile(t, filepath.Join(dir, "_schema", "Property", "relatedAporias.md"))
	it.Then(t).
		Should(it.String(relatedAporiasSchema).Contain("role: link")).
		Should(it.String(relatedAporiasSchema).Contain("label: Related Aporias"))
}

// arc apply note.patch.md, then note2.patch.md
// Scenario 3 from spec.md US2: an already-registered kind is left
// unchanged, not duplicated, on a second apply that reuses it.
func TestApplyRegisteredKindContentNotDuplicated(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch1 := writePatchFile(t, dir, "note.patch.md", notePatchWithHypothesis)
	_, err := sut(NewApplyCmd(), []string{patch1})
	it.Then(t).Should(it.Nil(err))

	before := readFile(t, filepath.Join(dir, "_schema", "Class", "Hypothesis.md"))

	secondPatch := strings.ReplaceAll(strings.ReplaceAll(notePatchWithHypothesis,
		"kolesnikov-2026-note", "kolesnikov-2026-note2"),
		"Forward Secrecy Requires Ephemeral Keys", "Handshake Latency Bound By RTT")
	patch2 := writePatchFile(t, dir, "note2.patch.md", secondPatch)
	_, err = sut(NewApplyCmd(), []string{patch2})
	it.Then(t).Should(it.Nil(err))

	after := readFile(t, filepath.Join(dir, "_schema", "Class", "Hypothesis.md"))
	it.Then(t).Should(it.Equal(before, after))
}

// arc apply note.patch.md (re-registered)
// Scenario 3 from spec.md US3: registering a kind removes the warning on
// the next apply (of a different document, since the same document is
// idempotent per US4)
func TestApplyRegisteringKindRemovesWarningOnNextApply(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch1 := writePatchFile(t, dir, "note.patch.md", notePatchWithHypothesis)

	_, stderr1, err := sutCaptureStderr(t, NewApplyCmd(), []string{patch1})
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.String(stderr1).Contain("Hypothesis"))

	seedSchemaNode(t, dir, "Hypothesis", "validatedOverwrite")
	secondPatch := strings.ReplaceAll(strings.ReplaceAll(notePatchWithHypothesis,
		"kolesnikov-2026-note", "kolesnikov-2026-note2"),
		"Forward Secrecy Requires Ephemeral Keys", "Handshake Latency Bound By RTT")
	patch2 := writePatchFile(t, dir, "note2.patch.md", secondPatch)

	_, stderr2, err := sutCaptureStderr(t, NewApplyCmd(), []string{patch2})
	it.Then(t).Should(it.Nil(err))
	it.Then(t).ShouldNot(it.String(stderr2).Contain("not a recognized node type"))
}

// "status" (retired outright, BUG-001/FR-034) is replaced by "scoreZ" — a
// still-registered, arc-init-seeded MergeLastWriteWin predicate, so this
// test's hand-edit-the-schema-document premise (below) still has a real
// _schema/Property/scoreZ.md to hand-edit.
const hypothesisSeedConfirmed = `---
"@id": "A Test Hypothesis"
"@type": Hypothesis
title: A Test Hypothesis
scoreZ: confirmed
---
# A Test Hypothesis

A conclusion distilled from sources.
`

const patchDivergesHypothesisStatusTemplate = `---
"@type": patch
document: %s
published: 2026-05-02
title: "%s"
---
# Source

## %s
` + "```yaml" + `
"@id": "%s"
"@type": Source
title: "%s"
published: "2026-05-02"
` + "```" + `

A short note.

# Hypothesis

## A Test Hypothesis
` + "```yaml\n\"@id\": \"A Test Hypothesis\"\n\"@type\": Hypothesis\nscoreZ: draft\n```" + `

A conclusion distilled from sources.
`

// arc apply
// spec.md FR-013/FR-015/FR-016: hand-editing a registered kind's own
// _schema/Class/<kind>.md merge value has no effect on reconciliation
// (D4 — it's vestigial); hand-editing the touched PREDICATE's own
// _schema/Property/<name>.md merge value is what changes the behavior a
// later arc apply invocation actually uses. "scoreZ" starts out
// arc-init-seeded lastWriteWin — a diverging contribution is silently
// applied (no conflict) — but after a hand-edit to firstWriteWin the
// identical shape of divergence is flagged.
func TestApplyHandEditedMergeValueChangesLaterApplyBehavior(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedSchemaNode(t, dir, "Hypothesis", "union")
	seedNode(t, dir, "Hypothesis/A Test Hypothesis.md", hypothesisSeedConfirmed)
	chdir(t, dir)

	lastWriteWinPatch := fmt.Sprintf(patchDivergesHypothesisStatusTemplate, "kolesnikov-2026-first", "First Note", "kolesnikov-2026-first", "kolesnikov-2026-first", "First Note")
	patch1 := writePatchFile(t, dir, "first.patch.md", lastWriteWinPatch)

	out1, err := sut(NewApplyCmd(), []string{patch1})
	it.Then(t).ShouldNot(it.Error(out1, err))

	content := readFile(t, filepath.Join(dir, "Hypothesis", "A Test Hypothesis.md"))
	it.Then(t).
		ShouldNot(it.String(content).Contain("<<<<<<<")).
		Should(it.String(content).Contain("scoreZ: draft"))

	statusDoc := readFile(t, filepath.Join(dir, "_schema", "Property", "scoreZ.md"))
	it.Then(t).Should(it.Nil(os.WriteFile(
		filepath.Join(dir, "_schema", "Property", "scoreZ.md"),
		[]byte(strings.ReplaceAll(statusDoc, "merge: lastWriteWin", "merge: firstWriteWin")), 0o644)))

	firstWriteWinPatch := fmt.Sprintf(patchDivergesHypothesisStatusTemplate, "kolesnikov-2026-second", "Second Note", "kolesnikov-2026-second", "kolesnikov-2026-second", "Second Note")
	firstWriteWinPatch = strings.ReplaceAll(firstWriteWinPatch, "scoreZ: draft", "scoreZ: shelved")
	patch2 := writePatchFile(t, dir, "second.patch.md", firstWriteWinPatch)

	out2, err := sut(NewApplyCmd(), []string{patch2})
	it.Then(t).ShouldNot(it.Error(out2, err))

	content = readFile(t, filepath.Join(dir, "Hypothesis", "A Test Hypothesis.md"))
	it.Then(t).
		Should(it.String(content).Contain("<<<<<<< existing")).
		Should(it.String(content).Contain("draft")).
		Should(it.String(content).Contain("shelved"))
}

// arc apply tls13.patch.md (twice)
// Scenario 1 from spec.md US4: re-applying an already-tracked document is
// a safe no-op
func TestApplyReapplyIsNoOp(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	_, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Nil(err))

	before := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).
		Should(it.String(out).Contain("already tracked")).
		Should(it.String(out).Contain("rescorla-2026-tls13"))

	after := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))
	it.Then(t).Should(it.Equal(before, after))
}

// arc apply broken.patch.md
// Edge case: manifest missing a mandatory field
func TestApplyMissingManifestFieldRefuses(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	broken := `---
"@type": patch
published: 2026-04-12
---
# Source

## x
` + "```yaml\n```\n" + `
text.
`
	patch := writePatchFile(t, dir, "broken.patch.md", broken)

	before := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Error(out, err))

	after := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))
	it.Then(t).Should(it.Equal(before, after))

	entries, rerr := os.ReadDir(filepath.Join(dir, "Source"))
	it.Then(t).
		Should(it.Nil(rerr)).
		Should(it.Equal(0, len(entries)))
}

// arc apply broken.patch.md
// Edge case: malformed patch body structure
func TestApplyMalformedPatchBodyRefuses(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	broken := `---
"@type": patch
document: foo-2026-x
published: 2026-04-12
---
Just prose, no H1/H2 structure.
`
	patch := writePatchFile(t, dir, "broken.patch.md", broken)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Error(out, err))
}

// arc apply tls13.patch.md
// RE-SCOPED — spec 031 BUG-001 (T068). This test asserted spec 010 US3
// Acceptance Scenario 4: an unrelated pre-0.5 node anywhere in the graph
// aborted the whole apply, via guardNoOldFormatNodes' whole-graph walk.
// That walk is gone (spec 031 FR-033) — it could not tell a legacy node from
// a host project's README once a graph root could be shared (FR-010,
// FR-032), and apply now reads only what its patch targets.
//
// What remains true, and is asserted here, is the part that was always the
// point: apply does not silently reinterpret a file it does not understand.
// An unrelated file is not misread — it is not read at all. Detection of a
// malformed node still happens on every path apply genuinely touches, which
// is what spec 010 FR-012's own read-triggered wording asks for; see
// TestRootModeApplyTargetedForeignFileFailsAsRead in nested_repo_test.go for
// the targeted case.
func TestApplyUnrelatedOldFormatNodeIsLeftUntouched(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	legacyNode := "---\nkind: entity\nid: old-node\ncategory: [independent]\n---\n# old-node\n\nAn entity written before this feature shipped.\n"
	seedNode(t, dir, "Entity/old-node.md", legacyNode)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	// the patch applied, and the file the patch never named is byte-for-byte
	// as it was — neither rewritten nor reinterpreted (spec 010 FR-013)
	assertIsFile(t, filepath.Join(dir, "Source", "rescorla-2026-tls13.md"))
	it.Then(t).Should(it.Equal(legacyNode, readFile(t, filepath.Join(dir, "Entity", "old-node.md"))))
}

// The three tests below cover spec 010 US3 Acceptance Scenarios 1-3: the
// legacy "kind" field, a missing "@id", and an "@id" that disagrees with its
// own basename. RE-SCOPED — spec 031 BUG-001 (T068): each used to seed its
// malformed file at a path the patch never named and assert the whole apply
// aborted, which only held because guardNoOldFormatNodes walked the entire
// graph. Each now seeds the SAME malformation at a path the patch DOES
// target, which is where spec 010 FR-012's read-triggered detection still
// applies and still must fire. The scenarios are unchanged; only the file
// arc has to actually read to meet them has moved.
//
// "Entity/Transport Layer Security.md" is one of tls13Patch's own two nodes.
// The source path would not do: it is committed by seedNode, so apply's
// idempotency check (FR-003) would report "already tracked" and return
// before reading anything.

// arc apply tls13.patch.md
// spec 010 US3 Acceptance Scenario 1: a node file using the legacy "kind"
// field with no "@id"/"@type" is rejected.
func TestApplyOldFormatKindFieldRefuses(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	legacyNode := "---\nkind: entity\nid: old-node\n---\n# old-node\n\nLegacy shaped.\n"
	seedNode(t, dir, "Entity/Transport Layer Security.md", legacyNode)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Error(out, err).Contain("Transport Layer Security"))
	it.Then(t).Should(it.String(err.Error()).Contain(`legacy "kind" field present`))
}

// arc apply tls13.patch.md
// spec 010 US3 Acceptance Scenario 2: a node file with "@type" but missing
// "@id" is rejected, with no fallback to any other field.
func TestApplyMissingIdRefuses(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	legacyNode := "---\n\"@type\": Entity\ntitle: No Id\n---\n# No Id\n\nMissing @id.\n"
	seedNode(t, dir, "Entity/Transport Layer Security.md", legacyNode)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Error(out, err).Contain("Transport Layer Security"))
	it.Then(t).Should(it.String(err.Error()).Contain(`missing mandatory "@id" field`))
}

// arc apply tls13.patch.md
// spec 010 US3 Acceptance Scenario 3: a node file whose "@id" does not
// equal its own file's basename is rejected rather than accepted.
func TestApplyIdMismatchedBasenameRefuses(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	mismatched := "---\n\"@id\": Something Else\n\"@type\": Entity\n---\n# Mismatched\n\nWrong id.\n"
	seedNode(t, dir, "Entity/Transport Layer Security.md", mismatched)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Error(out, err).Contain("Transport Layer Security"))
	it.Then(t).Should(it.String(err.Error()).Contain("does not match this file's basename"))
}

const patchWithExplicitTimelineSection = `---
"@type": patch
document: foo-2026-x
published: 2026-07-12
title: "A Test Document"
---
# Source

## foo-2026-x
` + "```yaml" + `
"@id": "foo-2026-x"
"@type": Source
title: "A Test Document"
published: "2026-07-12"
` + "```" + `

A test document.

# Timeline

## 2026-07
` + "```yaml\n\"@id\": \"2026-07\"\n\"@type\": Timeline\ngranularity: monthly\n```" + `
- [[foo-2026-x]]
`

// arc apply timeline.patch.md
// BUG-006 (corrects BUG-005's over-broad "refuse the whole patch" fix): a
// real extraction pipeline intentionally emits a "# Timeline" section
// alongside a document's own "# Source" section — the tool must apply
// such a patch successfully, folding the declared period into its own
// derived timeline index (correctly named timeline/monthly|yearly/*.md,
// never the generic per-kind "timelines/" folder that previously
// collided with it).
func TestApplyPatchCarriedTimelineSectionFoldedIntoIndex(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "timeline.patch.md", patchWithExplicitTimelineSection)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	_, statErr := os.Stat(filepath.Join(dir, "timelines"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))

	monthly := readFile(t, filepath.Join(dir, "timeline", "monthly", "2026-07.md"))
	yearly := readFile(t, filepath.Join(dir, "timeline", "yearly", "2026.md"))
	it.Then(t).
		Should(it.String(monthly).Contain("foo-2026-x")).
		Should(it.String(yearly).Contain("foo-2026-x"))
}

// arc apply timeline.patch.md
// BUG-006: a patch's explicitly-declared Timeline period may differ from
// the month patch.Published itself derives (e.g. a later correction);
// both monthly periods must end up populated, and the shared yearly
// rollup must contain the entry exactly once (not duplicated).
func TestApplyPatchCarriedTimelineSectionCascadesToYearlyForDifferentMonth(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	differentMonth := strings.ReplaceAll(patchWithExplicitTimelineSection, "2026-07-12", "2026-08-12")
	patch := writePatchFile(t, dir, "timeline.patch.md", differentMonth)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	augustMonthly := readFile(t, filepath.Join(dir, "timeline", "monthly", "2026-08.md"))
	julyMonthly := readFile(t, filepath.Join(dir, "timeline", "monthly", "2026-07.md"))
	yearly := readFile(t, filepath.Join(dir, "timeline", "yearly", "2026.md"))
	it.Then(t).
		Should(it.String(augustMonthly).Contain("foo-2026-x")).
		Should(it.String(julyMonthly).Contain("foo-2026-x")).
		Should(it.String(yearly).Contain("foo-2026-x")).
		Should(it.Equal(1, strings.Count(yearly, "foo-2026-x")))
}

// arc apply tls13.patch.md
// Edge case: target directory is not an initialized graph
func TestApplyTargetNotAGraphRefuses(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	out, err := sut(NewApplyCmd(), []string{patch})

	it.Then(t).Should(it.Error(out, err).Contain("initialized graph"))
}

// arc apply tls13.patch.md
// spec.md US2 Acceptance Scenario 5 / FR-014, quickstart.md Scenario 2: a
// malformed _schema/ document aborts arc apply before any other write.
func TestApplyMalformedSchemaDocumentAbortsWithZeroWrites(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	// spec 012 FR-020 (Bugfix 018/BUG-001): a Class document's own "merge"
	// field is no longer validated as mandatory, so corrupting it (as this
	// test previously did via "merge: bogus") no longer makes the document
	// malformed — its mandatory descriptive body is corrupted instead.
	entityDoc := readFile(t, filepath.Join(dir, "_schema", "Class", "Entity.md"))
	it.Then(t).Should(it.Nil(os.WriteFile(
		filepath.Join(dir, "_schema", "Class", "Entity.md"),
		[]byte(strings.ReplaceAll(entityDoc, "A node for a subject occurring in sources, typed by Sowa category.", "")), 0o644)))
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	before := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Error(out, err).Contain("_schema/Class/Entity.md"))

	after := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))
	it.Then(t).Should(it.Equal(before, after))

	_, statErr := os.Stat(filepath.Join(dir, "Source", "rescorla-2026-tls13.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

// arc apply tls13.patch.md
// spec.md US2 Acceptance Scenario 5 / FR-014: a missing _schema/ subfolder
// aborts arc apply before any other write.
func TestApplyMissingSchemaFolderAbortsWithZeroWrites(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	it.Then(t).Should(it.Nil(os.RemoveAll(filepath.Join(dir, "_schema", "Class"))))
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Error(out, err).Contain("_schema/Class"))

	_, statErr := os.Stat(filepath.Join(dir, "Source", "rescorla-2026-tls13.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

// arc apply pqkex.patch.md
// Edge case: a merge conflict marker is written while the commit still
// completes (FR-013); PostRunE prints a hint naming the conflicted file
func TestApplyConflictHintPrinted(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedNode(t, dir, "Resource/RFC 8446.md", rfcResourceSeedSetStatus)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "pqkex.patch.md", patchDivergesResourceStatus)

	_, stderr, err := sutCaptureStderr(t, NewApplyCmd(), []string{patch})

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.String(stderr).Contain("merge conflict"))
}

// arc apply --json tls13.patch.md
// --json output contract from contracts/cli-contract.md
func TestApplyJSONOutput(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)
	bios.JSON = true
	t.Cleanup(func() { bios.JSON = false })

	out, err := sut(NewApplyCmd(), []string{patch})

	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).
		Should(it.String(out).Contain(`"document"`)).
		Should(it.String(out).Contain(`"commit"`)).
		Should(it.String(out).Contain(`"warnings"`))
}

// arc apply --verbose tls13.patch.md
// BUG-001 / spec.md FR-021: --verbose reports one line per processed node
// naming its title and status; default mode is unaffected.
func TestApplyVerboseModeShowsPerNodeProgress(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)
	bios.Verbose = true
	t.Cleanup(func() { bios.Verbose = false })

	stdout, stderr, err := sutCaptureStderr(t, NewApplyCmd(), []string{patch})

	it.Then(t).ShouldNot(it.Error(stdout, err))
	it.Then(t).
		Should(it.String(stderr).Contain("rescorla-2026-tls13: created")).
		Should(it.String(stderr).Contain("Transport Layer Security: created")).
		Should(it.String(stderr).Contain("Reading patch file")).
		Should(it.String(stderr).Contain("Applying node contributions")).
		Should(it.String(stderr).Contain("Committing"))
}

const duplicateDefinitionPatchA = `---
"@type": patch
document: doc-2026-dup-a
published: 2026-06-01
title: "Document A"
---
# Source

## doc-2026-dup-a
` + "```yaml" + `
"@id": "doc-2026-dup-a"
"@type": Source
title: "Document A"
published: "2026-06-01"
` + "```" + `

First document.

# Entity

## Widget Dup
` + "```yaml" + `
"@id": "Widget Dup"
"@type": Entity
category: [independent]
` + "```" + `

A duplicate-tested widget definition.
`

const duplicateDefinitionPatchB = `---
"@type": patch
document: doc-2026-dup-b
published: 2026-06-02
title: "Document B"
---
# Source

## doc-2026-dup-b
` + "```yaml" + `
"@id": "doc-2026-dup-b"
"@type": Source
title: "Document B"
published: "2026-06-02"
` + "```" + `

Second document.

# Entity

## Widget Dup
` + "```yaml" + `
"@id": "Widget Dup"
"@type": Entity
category: [independent]
` + "```" + `

A duplicate-tested widget definition.
`

// arc apply docDupA.patch.md; arc apply --verbose docDupB.patch.md
// BUG-002 (spec.md FR-019): a second patch contributing a byte-identical
// definition to an entity another patch already created is a genuine
// no-op for that union/append predicate — --verbose must report
// "unchanged", never "appended", when nothing actually changed. The
// predicate itself is "text" (BUG-001/FR-030 — was "definition").
func TestApplyVerboseReportsUnchangedNotAppendedForDuplicateContribution(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patchA := writePatchFile(t, dir, "docDupA.patch.md", duplicateDefinitionPatchA)
	_, err := sut(NewApplyCmd(), []string{patchA})
	it.Then(t).Should(it.Nil(err))

	patchB := writePatchFile(t, dir, "docDupB.patch.md", duplicateDefinitionPatchB)
	bios.Verbose = true
	t.Cleanup(func() { bios.Verbose = false })

	stdout, stderr, err := sutCaptureStderr(t, NewApplyCmd(), []string{patchB})

	it.Then(t).ShouldNot(it.Error(stdout, err))
	it.Then(t).
		Should(it.String(stderr).Contain("text: append -> unchanged")).
		ShouldNot(it.String(stderr).Contain("text: append -> appended"))

	content := readFile(t, filepath.Join(dir, "Entity", "Widget Dup.md"))
	it.Then(t).Should(it.Equal(1, strings.Count(content, "A duplicate-tested widget definition.")))
}

// arc apply tls13.patch.md (default mode)
// Confirms BUG-001's fix did not regress default-mode conciseness.
func TestApplyDefaultModeShowsNoPerNodeProgress(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	stdout, stderr, err := sutCaptureStderr(t, NewApplyCmd(), []string{patch})

	it.Then(t).ShouldNot(it.Error(stdout, err))
	it.Then(t).
		ShouldNot(it.String(stderr).Contain("created")).
		ShouldNot(it.String(stderr).Contain("Reading patch file")).
		ShouldNot(it.String(stderr).Contain("Applying node contributions")).
		Should(it.String(stdout).Contain("rescorla-2026-tls13"))
}

const deployEvent1Patch = `---
"@type": patch
document: acme-2026-deploy1
published: 2026-06-01
title: "First Deploy"
---
# Source

## acme-2026-deploy1
` + "```yaml" + `
"@id": "acme-2026-deploy1"
"@type": Source
title: "First Deploy"
published: "2026-06-01"
` + "```" + `

A deployment record.

# LogEntry

## Deploy Event
` + "```yaml\n\"@id\": \"Deploy Event\"\n\"@type\": LogEntry\n```" + `

An event log.
- relatesTo:: [[Service A]]
`

const deployEvent2Patch = `---
"@type": patch
document: acme-2026-deploy2
published: 2026-06-02
title: "Second Deploy"
---
# Source

## acme-2026-deploy2
` + "```yaml" + `
"@id": "acme-2026-deploy2"
"@type": Source
title: "Second Deploy"
published: "2026-06-02"
` + "```" + `

Another deployment record.

# LogEntry

## Deploy Event
` + "```yaml\n\"@id\": \"Deploy Event\"\n\"@type\": LogEntry\n```" + `

An event log.
- relatesTo:: [[Service B]]
`

// arc apply deploy1.patch.md, then deploy2.patch.md
// BUG-002 / spec.md FR-022: a domain kind registered with the "append"
// merge behavior applies and re-merges successfully, unioning relations
// like "union" does, with no crash and no conflict.
func TestApplyAppendRegisteredKindUnionsAcrossPatches(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedSchemaNode(t, dir, "LogEntry", "append")
	chdir(t, dir)
	patch1 := writePatchFile(t, dir, "deploy1.patch.md", deployEvent1Patch)
	patch2 := writePatchFile(t, dir, "deploy2.patch.md", deployEvent2Patch)

	out1, err := sut(NewApplyCmd(), []string{patch1})
	it.Then(t).ShouldNot(it.Error(out1, err))

	out2, err := sut(NewApplyCmd(), []string{patch2})
	it.Then(t).ShouldNot(it.Error(out2, err))
	it.Then(t).
		Should(it.String(out2).Contain("+0 LogEntrys")).
		Should(it.String(out2).Contain("1 merged"))

	// The summary above still reads "LogEntrys" while the folder below is
	// "LogEntry": pluralizeKind formats a display count and nodeFolder
	// derives a path, and specs/022-reference-type-folders made only the
	// second one the identity function (research.md D4). This pair is the
	// regression guard against "fixing" the display string to match.

	entries, rerr := os.ReadDir(filepath.Join(dir, "LogEntry"))
	it.Then(t).
		Should(it.Nil(rerr)).
		Should(it.Equal(1, len(entries)))

	content := readFile(t, filepath.Join(dir, "LogEntry", "Deploy Event.md"))
	it.Then(t).
		Should(it.String(content).Contain("relatesTo:: [[Service A]]")).
		Should(it.String(content).Contain("relatesTo:: [[Service B]]")).
		ShouldNot(it.String(content).Contain("<<<<<<<"))
}

// A patch shaped like github.com/fogfish/arcnet-spec's own canonical
// example patch (examples/patches/rescorla-2026-tls13.md): predicate-
// grouped body blocks use CORE §12.2's bold-label convention, never
// headings, and each node carries multiple such blocks.
const boldLabelCanonicalPatch = `---
"@type": patch
document: rescorla-2026-tls13
published: 2026-04-12
title: "TLS 1.3: Design and Rationale"
---
# Source

## rescorla-2026-tls13
` + "```yaml" + `
"@id": "rescorla-2026-tls13"
"@type": Source
title: "TLS 1.3: Design and Rationale"
author: [Eric Rescorla]
published: "2026-04-12"
` + "```" + `

A design retrospective on the TLS 1.3 handshake.

**Mentions**
- mentions:: [[Transport Layer Security]]

**Cites**
- cites:: [[RFC 8446]]

# Entity

## Transport Layer Security
` + "```yaml" + `
"@id": "Transport Layer Security"
"@type": Entity
category: [independent, abstract, occurrent, script]
` + "```" + `

A cryptographic protocol.

**Mentioned In**
- mentionedIn:: [[rescorla-2026-tls13]]

**Related**
- related:: [[Forward Secrecy]]
`

// extractQuotedAttr returns the value of a `key: "value"` front-matter line
// within content, or "" if absent — used to compare timestamp attributes
// (`indexed`/`updated`) written by two different node files from the same
// arc apply invocation (spec.md FR-005/FR-009).
func extractQuotedAttr(content, key string) string {
	m := regexp.MustCompile(key + `: "([^"]+)"`).FindStringSubmatch(content)
	if m == nil {
		return ""
	}
	return m[1]
}

// arc apply tls13.patch.md
// Scenario 1 from spec.md US1: a created ordinary node's published equals
// the patch's own declared date.
func TestApplyCreatedNodeCarriesPublishedFromPatch(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	source := readFile(t, filepath.Join(dir, "Source", "rescorla-2026-tls13.md"))
	entity := readFile(t, filepath.Join(dir, "Entity", "Transport Layer Security.md"))
	it.Then(t).
		Should(it.String(source).Contain(`published: "2026-04-12"`)).
		Should(it.String(entity).Contain(`published: "2026-04-12"`))
}

// arc apply tls13.patch.md
// Scenario 2 from spec.md US1: every node one application creates shares
// an identical indexed value (FR-005).
func TestApplyCreatedNodesShareIdenticalIndexedValue(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	source := readFile(t, filepath.Join(dir, "Source", "rescorla-2026-tls13.md"))
	entity := readFile(t, filepath.Join(dir, "Entity", "Transport Layer Security.md"))

	sourceIndexed := extractQuotedAttr(source, "indexed")
	entityIndexed := extractQuotedAttr(entity, "indexed")
	it.Then(t).
		ShouldNot(it.Equal("", sourceIndexed)).
		Should(it.Equal(sourceIndexed, entityIndexed))
}

const stubEntityPatch = `---
"@type": patch
document: foo-2026-stub
published: 2026-04-12
title: "Stub Test Document"
---
# Source

## foo-2026-stub
` + "```yaml" + `
"@id": "foo-2026-stub"
"@type": Source
title: "Stub Test Document"
published: "2026-04-12"
` + "```" + `

A stub test document.

# Entity

## StubEntity
` + "```yaml\n\"@id\": \"StubEntity\"\n\"@type\": Entity\n```" + `
`

// arc apply stub.patch.md
// Scenario 3 from spec.md US1: a minimal-stub patch section (kind/id only)
// creates a node carrying neither published nor indexed.
func TestApplyStubShapedSectionCreatesNodeWithNoPublishedOrIndexed(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "stub.patch.md", stubEntityPatch)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	content := readFile(t, filepath.Join(dir, "Entity", "StubEntity.md"))
	it.Then(t).
		ShouldNot(it.String(content).Contain("published:")).
		ShouldNot(it.String(content).Contain("indexed:"))
}

// arc apply note.patch.md
// Scenario 4 from spec.md US1: an auto-registered _schema/ document carries
// neither published nor indexed, exactly like a stub node.
func TestApplyAutoRegisteredSchemaDocumentCarriesNoTimestamps(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "note.patch.md", notePatchWithHypothesis)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	schemaDoc := readFile(t, filepath.Join(dir, "_schema", "Class", "Hypothesis.md"))
	it.Then(t).
		ShouldNot(it.String(schemaDoc).Contain("published:")).
		ShouldNot(it.String(schemaDoc).Contain("indexed:"))
}

// arc apply pqkex.patch.md
// Scenario 1 from spec.md US2: a real merge stamps updated identical to
// the same application's indexed value on a newly created node.
func TestApplyRealMergeStampsUpdatedIdenticalToIndexed(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedNode(t, dir, "Entity/Transport Layer Security.md", tlsEntitySeed)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "pqkex.patch.md", pqkexPatchMergingEntity)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	source := readFile(t, filepath.Join(dir, "Source", "chen-2026-pqkex.md"))
	entity := readFile(t, filepath.Join(dir, "Entity", "Transport Layer Security.md"))

	sourceIndexed := extractQuotedAttr(source, "indexed")
	entityUpdated := extractQuotedAttr(entity, "updated")
	it.Then(t).
		ShouldNot(it.Equal("", sourceIndexed)).
		Should(it.Equal(sourceIndexed, entityUpdated))
}

const memoNoneSeed = `---
"@id": Widget
"@type": Memo
title: Widget
---
# Widget

Original text.
`

const memoNonePatch = `---
"@type": patch
document: foo-2026-memo
published: 2026-05-01
title: "Memo Patch"
---
# Source

## foo-2026-memo
` + "```yaml" + `
"@id": "foo-2026-memo"
"@type": Source
title: "Memo Patch"
published: "2026-05-01"
` + "```" + `

A memo patch.

# Memo

## Widget
` + "```yaml\n\"@id\": \"Widget\"\n\"@type\": Memo\n```" + `

Changed text.
`

// arc apply memo.patch.md
// spec.md FR-015: a custom type's own registered whole-node "merge" value
// no longer determines how any of its predicates reconcile — "memo" is
// hand-registered "immutable" here, but that's now inert data (D4); its
// leading prose reconciles via the generic "text" predicate (seeded
// append), so a second, genuinely new contribution still grows its
// content and stamps updated, regardless of the type's own vestigial
// merge value.
func TestApplyTypeLevelMergeValueNoLongerGovernsReconciliation(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedSchemaNode(t, dir, "Memo", "immutable")
	seedNode(t, dir, "Memo/Widget.md", memoNoneSeed)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "memo.patch.md", memoNonePatch)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	content := readFile(t, filepath.Join(dir, "Memo", "Widget.md"))
	it.Then(t).
		Should(it.String(content).Contain("Original text.")).
		Should(it.String(content).Contain("Changed text.")).
		Should(it.String(content).Contain("updated:"))
}

const stubbedThingSeed = `---
"@id": "StubbedThing"
"@type": Entity
---
# StubbedThing
`

const patchFillsStubWithRealContent = `---
"@type": patch
document: foo-2026-fill
published: 2026-06-01
title: "Fill Patch"
---
# Source

## foo-2026-fill
` + "```yaml" + `
"@id": "foo-2026-fill"
"@type": Source
title: "Fill Patch"
published: "2026-06-01"
` + "```" + `

A fill patch.

# Entity

## StubbedThing
` + "```yaml\n\"@id\": \"StubbedThing\"\n\"@type\": Entity\npublished: \"2026-05-02\"\n```" + `

Now has real content.
`

// arc apply fill.patch.md
// Scenario 3 from spec.md US2: a stub later merged with real content fills
// published (per that node's own merge rules, from the contribution's own
// value — distinct from the patch manifest's own published) and gains
// updated, but never gains indexed (only ever assigned at non-stub
// creation, and this node's creation was a stub).
func TestApplyStubMergedWithRealContentFillsPublishedAndUpdatedNeverIndexed(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedNode(t, dir, "Entity/StubbedThing.md", stubbedThingSeed)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "fill.patch.md", patchFillsStubWithRealContent)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	content := readFile(t, filepath.Join(dir, "Entity", "StubbedThing.md"))
	it.Then(t).
		Should(it.String(content).Contain(`published: "2026-05-02"`)).
		ShouldNot(it.String(content).Contain("indexed:"))

	updated := extractQuotedAttr(content, "updated")
	it.Then(t).ShouldNot(it.Equal("", updated))
}

const noOpUnionEntitySeed = `---
"@id": "Widget"
"@type": Entity
title: Widget
category: [independent, abstract, occurrent, script]
---
# Widget

A test entity.
- replaces:: [[Old Widget]]
`

const noOpUnionPatch = `---
"@type": patch
document: foo-2026-noop
published: 2026-05-03
title: "No-op Patch"
---
# Source

## foo-2026-noop
` + "```yaml" + `
"@id": "foo-2026-noop"
"@type": Source
title: "No-op Patch"
published: "2026-05-03"
` + "```" + `

A no-op patch.

# Entity

## Widget
` + "```yaml" + `
"@id": "Widget"
"@type": Entity
category: [independent, abstract, occurrent, script]
` + "```" + `

A test entity.
- replaces:: [[Old Widget]]
`

// arc apply noop.patch.md
// Scenario 4 from spec.md US2: a non-"none" (union) re-contribution that
// nets out identical to the file's prior content adds no updated.
func TestApplyNoOpUnionReContributionAddsNoUpdated(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedNode(t, dir, "Entity/Widget.md", noOpUnionEntitySeed)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "noop.patch.md", noOpUnionPatch)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	content := readFile(t, filepath.Join(dir, "Entity", "Widget.md"))
	it.Then(t).ShouldNot(it.String(content).Contain("updated:"))
}

// arc apply tls13.patch.md
// BUG-003 / spec.md FR-004: a patch using CORE §12.2's canonical
// bold-label body convention, with multiple predicate-grouped blocks per
// node, must have every declared edge survive into the written node file
// — none silently dropped.
func TestApplyBoldLabelPatchNoEdgeLoss(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", boldLabelCanonicalPatch)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	source := readFile(t, filepath.Join(dir, "Source", "rescorla-2026-tls13.md"))
	it.Then(t).
		Should(it.String(source).Contain("mentions:: [[Transport Layer Security]]")).
		Should(it.String(source).Contain("cites:: [[RFC 8446]]"))

	entity := readFile(t, filepath.Join(dir, "Entity", "Transport Layer Security.md"))
	it.Then(t).
		Should(it.String(entity).Contain("mentionedIn:: [[rescorla-2026-tls13]]")).
		Should(it.String(entity).Contain("related:: [[Forward Secrecy]]"))
}

// headingOnlyCanonicalPatch mirrors CORE §12.2's own canonical patch
// convention exactly as real external patch-generating tools (e.g.
// fogfish/bots) produce it: no "@id"/"@type" duplicated inside any node's
// own yaml fence at all — identity/type come solely from the "## <ID>"/
// "# <Type>" section headings (BUG-001).
const headingOnlyCanonicalPatch = `---
"@type": patch
document: dmitry-2026-graph
published: 2026-07-03
title: "Ontologies, Graph Structures, and LLM-Based Knowledge Management"
---
# Source

## dmitry-2026-graph
` + "```yaml" + `
author:
    - Dmitry
published: "2026-07-03"
title: Ontologies, Graph Structures, and LLM-Based Knowledge Management
` + "```" + `

Structured markdown documents with explicit ontology can serve as practical graph nodes.

**Mentions**
- mentions:: [[LLM]]

# Entity

## LLM
` + "```yaml" + `
category: [independent, abstract, occurrent, script]
` + "```" + `

Large Language Models.
`

// arc apply bots.patch.md
// BUG-001: a patch shaped exactly like a real external patch-generating
// tool's output (no "@id"/"@type" duplicated inside any node's yaml
// fence) is accepted end-to-end, deriving identity/type from the section
// headings alone — not rejected as an unsupported/old-format file.
func TestApplyHeadingOnlyCanonicalPatchAcceptedEndToEnd(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "bots.patch.md", headingOnlyCanonicalPatch)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	assertIsFile(t, filepath.Join(dir, "Source", "dmitry-2026-graph.md"))
	assertIsFile(t, filepath.Join(dir, "Entity", "LLM.md"))

	entity := readFile(t, filepath.Join(dir, "Entity", "LLM.md"))
	it.Then(t).
		Should(it.String(entity).Contain(`"@id": LLM`)).
		Should(it.String(entity).Contain(`"@type": Entity`))
}

// --- spec 012: Per-Predicate Merge Reconciliation for arc apply ---
//
// The fixtures below use arc init's own real seeded schema
// (appschema.Seed(), via initGraph) so every predicate's declared merge
// behavior is exactly what a real graph would use: created/immutable,
// scoreZ/lastWriteWin, tags/union, heading/firstWriteWin, url/
// fillIfEmpty (internal/app/schema/kernel/schema.go). "heading" (role:
// meta) stands in for a firstWriteWin exemplar here rather than
// "abstract" (role: text), since BUG-001/FR-018 repoints every role:text
// predicate — abstract included — to append, and rather than "category",
// which now declares immutable and so freezes without ever flagging. "created"/"scoreZ" replace
// this block's original "ref"/"status" exemplars, both retired outright
// under CORE 0.12 (BUG-001/FR-034) — same immutable/lastWriteWin
// semantics, real registered predicates instead of retired ones.

const resourcePatch012First = `---
"@type": patch
document: doc-2026-a
published: 2026-07-01
title: "Doc A"
---
# Source

## doc-2026-a
` + "```yaml" + `
"@id": "doc-2026-a"
"@type": Source
title: "Doc A"
published: "2026-07-01"
` + "```" + `

A source document.

# Resource

## example-book
` + "```yaml" + `
"@id": "example-book"
"@type": Resource
created: book
scoreZ: backlog
tags: [ai]
heading: "First summary."
` + "```" + `

A tracked reading item.
`

const resourcePatch012Second = `---
"@type": patch
document: doc-2026-b
published: 2026-07-02
title: "Doc B"
---
# Source

## doc-2026-b
` + "```yaml" + `
"@id": "doc-2026-b"
"@type": Source
title: "Doc B"
published: "2026-07-02"
` + "```" + `

Another source document.

# Resource

## example-book
` + "```yaml" + `
"@id": "example-book"
"@type": Resource
created: article
scoreZ: read
tags: [ml]
heading: "A different summary."
` + "```" + `

A tracked reading item.
`

// arc apply doc-a.patch.md, then doc-b.patch.md
// spec 012 User Story 1, Acceptance Scenarios 1-4: a single second
// contribution touching three predicates at once resolves each by its own
// rule within that one application — created (immutable) rejects the
// divergence, scoreZ (lastWriteWin) takes the newest value, tags (union)
// accumulates every distinct value, and none of the three affects any
// other's outcome.
func TestApply012US1PerPredicateReconciliation(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch1 := writePatchFile(t, dir, "doc-a.patch.md", resourcePatch012First)
	patch2 := writePatchFile(t, dir, "doc-b.patch.md", resourcePatch012Second)

	_, err := sut(NewApplyCmd(), []string{patch1})
	it.Then(t).Should(it.Nil(err))
	out, err := sut(NewApplyCmd(), []string{patch2})
	it.Then(t).ShouldNot(it.Error(out, err))

	content := readFile(t, filepath.Join(dir, "Resource", "example-book.md"))
	it.Then(t).
		Should(it.String(content).Contain("created: book")).
		Should(it.String(content).Contain("scoreZ: read")).
		Should(it.String(content).Contain("- ai")).
		Should(it.String(content).Contain("- ml"))
}

// BUG-001 / spec.md FR-017: --verbose reports one indented line per
// predicate a merge actually touched, naming its resolved MergeOp and
// outcome — not only the node-level summary line.
func TestApply012BUG001VerboseReportsPerPredicateOutcomes(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch1 := writePatchFile(t, dir, "doc-a.patch.md", resourcePatch012First)
	patch2 := writePatchFile(t, dir, "doc-b.patch.md", resourcePatch012Second)

	_, err := sut(NewApplyCmd(), []string{patch1})
	it.Then(t).Should(it.Nil(err))

	bios.Verbose = true
	t.Cleanup(func() { bios.Verbose = false })
	stdout, stderr, err := sutCaptureStderr(t, NewApplyCmd(), []string{patch2})
	it.Then(t).ShouldNot(it.Error(stdout, err))

	it.Then(t).
		Should(it.String(stderr).Contain("example-book: merged")).
		Should(it.String(stderr).Contain("created: immutable -> unchanged")).
		Should(it.String(stderr).Contain("scoreZ: lastWriteWin -> overwritten")).
		Should(it.String(stderr).Contain("tags: union -> appended"))
}

// spec 012 User Story 2, Acceptance Scenario 1/3: within the same
// combined application, heading (firstWriteWin) is flagged for human
// review on genuine divergence, but tags (union) and scoreZ
// (lastWriteWin) — which diverge too — are never flagged.
func TestApply012US2ConflictFlaggingScopedToFirstWriteWin(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch1 := writePatchFile(t, dir, "doc-a.patch.md", resourcePatch012First)
	patch2 := writePatchFile(t, dir, "doc-b.patch.md", resourcePatch012Second)

	_, err := sut(NewApplyCmd(), []string{patch1})
	it.Then(t).Should(it.Nil(err))
	_, stderr, err := sutCaptureStderr(t, NewApplyCmd(), []string{patch2})
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.String(stderr).Contain("merge conflict"))

	content := readFile(t, filepath.Join(dir, "Resource", "example-book.md"))
	it.Then(t).
		Should(it.String(content).Contain("<<<<<<< existing")).
		Should(it.String(content).Contain("First summary.")).
		Should(it.String(content).Contain("A different summary."))

	// tags/scoreZ diverge too but must never be wrapped in a marker
	tagsAndStatusLines := []string{"scoreZ: read", "- ai", "- ml"}
	for _, line := range tagsAndStatusLines {
		it.Then(t).Should(it.String(content).Contain(line))
	}
}

const resourceFillIfEmptySeed = `---
"@id": "RFC 9999"
"@type": Resource
title: RFC 9999
url: ""
---
# RFC 9999

A draft specification.
`

const patchFillsUrlFirstTime = `---
"@type": patch
document: doc-2026-c
published: 2026-07-03
title: "Doc C"
---
# Source

## doc-2026-c
` + "```yaml" + `
"@id": "doc-2026-c"
"@type": Source
title: "Doc C"
published: "2026-07-03"
` + "```" + `

A source document.

# Resource

## RFC 9999
` + "```yaml" + `
"@id": "RFC 9999"
"@type": Resource
url: https://example.org/rfc9999-v1
` + "```" + `

A draft specification.
`

const patchDivergesUrlAfterFirstWrite = `---
"@type": patch
document: doc-2026-d
published: 2026-07-04
title: "Doc D"
---
# Source

## doc-2026-d
` + "```yaml" + `
"@id": "doc-2026-d"
"@type": Source
title: "Doc D"
published: "2026-07-04"
` + "```" + `

Another source document.

# Resource

## RFC 9999
` + "```yaml" + `
"@id": "RFC 9999"
"@type": Resource
url: https://example.org/rfc9999-v2
` + "```" + `

A draft specification.
`

// spec 012 User Story 2, Acceptance Scenario 4 / Edge Case: a
// fillIfEmpty predicate's first contribution is never flagged, but a
// later, genuinely diverging contribution is.
func TestApply012US2FillIfEmptyFlagsOnlyAfterFirstWrite(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedNode(t, dir, "Resource/RFC 9999.md", resourceFillIfEmptySeed)
	chdir(t, dir)
	patch1 := writePatchFile(t, dir, "doc-c.patch.md", patchFillsUrlFirstTime)

	_, stderr1, err := sutCaptureStderr(t, NewApplyCmd(), []string{patch1})
	it.Then(t).Should(it.Nil(err))
	it.Then(t).ShouldNot(it.String(stderr1).Contain("merge conflict"))

	content := readFile(t, filepath.Join(dir, "Resource", "RFC 9999.md"))
	it.Then(t).Should(it.String(content).Contain("https://example.org/rfc9999-v1"))

	patch2 := writePatchFile(t, dir, "doc-d.patch.md", patchDivergesUrlAfterFirstWrite)
	_, stderr2, err := sutCaptureStderr(t, NewApplyCmd(), []string{patch2})
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.String(stderr2).Contain("merge conflict"))

	content = readFile(t, filepath.Join(dir, "Resource", "RFC 9999.md"))
	it.Then(t).
		Should(it.String(content).Contain("<<<<<<< existing")).
		Should(it.String(content).Contain("rfc9999-v1")).
		Should(it.String(content).Contain("rfc9999-v2"))
}

const resourceIndependentPredicatesSeed = `---
"@id": "example-topic"
"@type": Resource
title: example-topic
created: topic
---
# example-topic

An ongoing research topic.
`

const patchContributesTags = `---
"@type": patch
document: doc-2026-e
published: 2026-07-05
title: "Doc E"
---
# Source

## doc-2026-e
` + "```yaml" + `
"@id": "doc-2026-e"
"@type": Source
title: "Doc E"
published: "2026-07-05"
` + "```" + `

A source document.

# Resource

## example-topic
` + "```yaml" + `
"@id": "example-topic"
"@type": Resource
created: topic
tags: [ai]
` + "```" + `

An ongoing research topic.
`

const patchContributesStatus = `---
"@type": patch
document: doc-2026-f
published: 2026-07-06
title: "Doc F"
---
# Source

## doc-2026-f
` + "```yaml" + `
"@id": "doc-2026-f"
"@type": Source
title: "Doc F"
published: "2026-07-06"
` + "```" + `

Another source document.

# Resource

## example-topic
` + "```yaml" + `
"@id": "example-topic"
"@type": Resource
created: topic
scoreZ: read
` + "```" + `

An ongoing research topic.
`

// spec 012 User Story 3, Acceptance Scenario 2: two patches contributing
// to independent predicates (tags/scoreZ) converge on an identical result
// regardless of which order they're applied in.
func TestApply012US3IndependentPredicatesConvergeInEitherOrder(t *testing.T) {
	forward := t.TempDir()
	initGraph(t, forward)
	seedNode(t, forward, "Resource/example-topic.md", resourceIndependentPredicatesSeed)

	reverse := t.TempDir()
	initGraph(t, reverse)
	seedNode(t, reverse, "Resource/example-topic.md", resourceIndependentPredicatesSeed)

	func() {
		chdir(t, forward)
		tagsPatch := writePatchFile(t, forward, "e.patch.md", patchContributesTags)
		_, err := sut(NewApplyCmd(), []string{tagsPatch})
		it.Then(t).Should(it.Nil(err))
		statusPatch := writePatchFile(t, forward, "f.patch.md", patchContributesStatus)
		_, err = sut(NewApplyCmd(), []string{statusPatch})
		it.Then(t).Should(it.Nil(err))
	}()

	func() {
		chdir(t, reverse)
		statusPatch := writePatchFile(t, reverse, "f.patch.md", patchContributesStatus)
		_, err := sut(NewApplyCmd(), []string{statusPatch})
		it.Then(t).Should(it.Nil(err))
		tagsPatch := writePatchFile(t, reverse, "e.patch.md", patchContributesTags)
		_, err = sut(NewApplyCmd(), []string{tagsPatch})
		it.Then(t).Should(it.Nil(err))
	}()

	forwardContent := readFile(t, filepath.Join(forward, "Resource", "example-topic.md"))
	reverseContent := readFile(t, filepath.Join(reverse, "Resource", "example-topic.md"))

	stripUpdated := func(s string) string {
		return regexp.MustCompile(`updated: "[^"]+"\n`).ReplaceAllString(s, "")
	}
	it.Then(t).Should(it.Equal(stripUpdated(forwardContent), stripUpdated(reverseContent)))
}

// spec 012 User Story 3, Acceptance Scenario 3: lastWriteWin is the
// documented, deliberate exception to order-independence — applying the
// same two status-diverging contributions in reverse order changes which
// value wins.
func TestApply012US3LastWriteWinIsOrderSensitive(t *testing.T) {
	forward := t.TempDir()
	initGraph(t, forward)
	seedNode(t, forward, "Resource/example-book.md", `---
"@id": "example-book"
"@type": Resource
title: example-book
created: book
---
# example-book

A tracked reading item.
`)

	reverse := t.TempDir()
	initGraph(t, reverse)
	seedNode(t, reverse, "Resource/example-book.md", `---
"@id": "example-book"
"@type": Resource
title: example-book
created: book
---
# example-book

A tracked reading item.
`)

	readPatch := `---
"@type": patch
document: doc-2026-read
published: 2026-07-07
title: "Doc Read"
---
# Source

## doc-2026-read
` + "```yaml" + `
"@id": "doc-2026-read"
"@type": Source
title: "Doc Read"
published: "2026-07-07"
` + "```" + `

A source document.

# Resource

## example-book
` + "```yaml" + `
"@id": "example-book"
"@type": Resource
created: book
scoreZ: read
` + "```" + `

A tracked reading item.
`
	archivedPatch := `---
"@type": patch
document: doc-2026-archived
published: 2026-07-08
title: "Doc Archived"
---
# Source

## doc-2026-archived
` + "```yaml" + `
"@id": "doc-2026-archived"
"@type": Source
title: "Doc Archived"
published: "2026-07-08"
` + "```" + `

Another source document.

# Resource

## example-book
` + "```yaml" + `
"@id": "example-book"
"@type": Resource
created: book
scoreZ: archived
` + "```" + `

A tracked reading item.
`

	func() {
		chdir(t, forward)
		p1 := writePatchFile(t, forward, "read.patch.md", readPatch)
		_, err := sut(NewApplyCmd(), []string{p1})
		it.Then(t).Should(it.Nil(err))
		p2 := writePatchFile(t, forward, "archived.patch.md", archivedPatch)
		_, err = sut(NewApplyCmd(), []string{p2})
		it.Then(t).Should(it.Nil(err))
	}()

	func() {
		chdir(t, reverse)
		p1 := writePatchFile(t, reverse, "archived.patch.md", archivedPatch)
		_, err := sut(NewApplyCmd(), []string{p1})
		it.Then(t).Should(it.Nil(err))
		p2 := writePatchFile(t, reverse, "read.patch.md", readPatch)
		_, err = sut(NewApplyCmd(), []string{p2})
		it.Then(t).Should(it.Nil(err))
	}()

	forwardContent := readFile(t, filepath.Join(forward, "Resource", "example-book.md"))
	reverseContent := readFile(t, filepath.Join(reverse, "Resource", "example-book.md"))

	it.Then(t).
		Should(it.String(forwardContent).Contain("scoreZ: archived")).
		Should(it.String(reverseContent).Contain("scoreZ: read"))
}

// spec 012 User Story 3, Acceptance Scenario 4: a conflict already marked
// for human review is not re-wrapped when an equivalent later
// contribution repeats the same divergence.
func TestApply012US3ReplayDoesNotRewrapConflictMarker(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch1 := writePatchFile(t, dir, "doc-a.patch.md", resourcePatch012First)
	patch2 := writePatchFile(t, dir, "doc-b.patch.md", resourcePatch012Second)

	_, err := sut(NewApplyCmd(), []string{patch1})
	it.Then(t).Should(it.Nil(err))
	_, err = sut(NewApplyCmd(), []string{patch2})
	it.Then(t).Should(it.Nil(err))

	firstConflict := readFile(t, filepath.Join(dir, "Resource", "example-book.md"))
	it.Then(t).Should(it.Equal(1, strings.Count(firstConflict, "<<<<<<<")))

	replayPatch := strings.ReplaceAll(resourcePatch012Second, "doc-2026-b", "doc-2026-b-replay")
	patch3 := writePatchFile(t, dir, "doc-b-replay.patch.md", replayPatch)
	_, err = sut(NewApplyCmd(), []string{patch3})
	it.Then(t).Should(it.Nil(err))

	secondConflict := readFile(t, filepath.Join(dir, "Resource", "example-book.md"))
	it.Then(t).
		Should(it.Equal(1, strings.Count(secondConflict, "<<<<<<<"))).
		Should(it.Equal(firstConflict, secondConflict))
}

const lowercaseH1Patch = `---
"@type": patch
document: rescorla-2026-tls13
published: 2026-04-12
title: "TLS 1.3: Design and Rationale"
---
# entity

## Transport Layer Security
` + "```yaml" + `
"@id": "Transport Layer Security"
"@type": entity
category: [independent, abstract, occurrent, script]
` + "```" + `

A cryptographic protocol.
`

// arc apply entity.patch.md
// spec.md US1 Acceptance Scenario 1: a patch whose class-defining H1
// heading begins with a lowercase letter is rejected — non-zero exit, the
// graph left unmodified (no node file written, no new commit).
func TestApplyLowercaseH1HeadingRejected(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, t.TempDir(), "entity.patch.md", lowercaseH1Patch)

	before := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Error(out, err).Contain("entity")).Should(it.Error(out, err).Contain("CamelCase"))

	after := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))
	it.Then(t).Should(it.Equal(before, after))

	_, statErr := os.Stat(filepath.Join(dir, "Entity", "Transport Layer Security.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

const uppercaseH1Patch = `---
"@type": patch
document: rescorla-2026-tls13
published: 2026-04-12
title: "TLS 1.3: Design and Rationale"
---
# Entity

## Transport Layer Security
` + "```yaml" + `
"@id": "Transport Layer Security"
category: [independent, abstract, occurrent, script]
` + "```" + `

A cryptographic protocol.
`

// arc apply entity.patch.md
// spec.md US1 Acceptance Scenario 2: a patch whose class-defining H1
// heading begins with an uppercase letter succeeds, and the node's class
// is stored using the heading's exact casing, with no lowercasing.
func TestApplyUppercaseH1HeadingStoresExactCasing(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "entity.patch.md", uppercaseH1Patch)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	content := readFile(t, filepath.Join(dir, "Entity", "Transport Layer Security.md"))
	it.Then(t).Should(it.String(content).Contain(`"@type": Entity`))
}

const multiH1OneNonCompliantPatch = `---
"@type": patch
document: rescorla-2026-tls13
published: 2026-04-12
title: "TLS 1.3: Design and Rationale"
---
# Source

## rescorla-2026-tls13
` + "```yaml" + `
"@id": "rescorla-2026-tls13"
title: "TLS 1.3: Design and Rationale"
author: [Eric Rescorla]
published: "2026-04-12"
` + "```" + `

A design retrospective on the TLS 1.3 handshake.

# entity

## Transport Layer Security
` + "```yaml" + `
"@id": "Transport Layer Security"
category: [independent, abstract, occurrent, script]
` + "```" + `

A cryptographic protocol.
`

// arc apply tls13.patch.md
// spec.md US1 Acceptance Scenario 3: a patch with multiple H1 sections
// where at least one begins with a lowercase letter rejects the entire
// document — no partial apply, not even for the well-formed section.
func TestApplyMultiH1OneNonCompliantRejectsWholeDocument(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, t.TempDir(), "tls13.patch.md", multiH1OneNonCompliantPatch)

	before := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Error(out, err).Contain("entity"))

	after := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))
	it.Then(t).Should(it.Equal(before, after))

	_, sourceStatErr := os.Stat(filepath.Join(dir, "Source", "rescorla-2026-tls13.md"))
	it.Then(t).Should(it.True(os.IsNotExist(sourceStatErr)))
	_, entityStatErr := os.Stat(filepath.Join(dir, "Entity", "Transport Layer Security.md"))
	it.Then(t).Should(it.True(os.IsNotExist(entityStatErr)))
}

// ---------------------------------------------------------------------------
// spec 021 — patch manifest identity ("@type": patch)
// ---------------------------------------------------------------------------

// gitLog returns the graph's whole oneline history, for before/after
// assertions that a rejection left git untouched (spec 021 FR-007).
func gitLog(t *testing.T, dir string) string {
	t.Helper()
	return strings.TrimSpace(runGit(t, dir, "log", "--oneline"))
}

// withIdentity rewrites tls13Patch's manifest identity line, the single
// variable across every spec 021 scenario below.
func withIdentity(identity string) string {
	return strings.Replace(tls13Patch, `"@type": patch`, identity, 1)
}

// arc apply shannon.patch.md
// spec 021 US1 Acceptance Scenario 1: a patch transcribed straight from
// ARCNET-CORE §14.2.2 — declaring itself with the quoted "@type": patch key
// — applies, writes every carried node, and records exactly one ingest
// commit. This is the whole defect: before this feature the success rate for
// a spec-conformant patch was 0% (SC-001).
func TestApplyTypeKeyPatchAppliesWithOneIngestCommit(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	before := commitCount(t, dir)

	out, err := sut(NewApplyCmd(), []string{patch})

	it.Then(t).ShouldNot(it.Error(out, err))
	assertIsFile(t, filepath.Join(dir, "Source", "rescorla-2026-tls13.md"))
	assertIsFile(t, filepath.Join(dir, "Entity", "Transport Layer Security.md"))
	it.Then(t).Should(it.Equal(before+1, commitCount(t, dir)))
}

// arc apply shannon.patch.md (twice)
// spec 021 US1 Acceptance Scenario 2: idempotency survives the key change —
// a second apply is a skip, with no second commit (FR-012, SC-006).
func TestApplyTypeKeyPatchReapplyRecordsNoSecondCommit(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	_, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Nil(err))
	afterFirst := commitCount(t, dir)

	out, err := sut(NewApplyCmd(), []string{patch})

	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).
		Should(it.String(out).Contain("already tracked")).
		Should(it.Equal(afterFirst, commitCount(t, dir)))
}

// arc apply legacy.patch.md
// Every rejection fixture below is written *outside* the graph tree, matching
// this file's own TestApplyMultiH1OneNonCompliantRejectsWholeDocument
// precedent: a patch left sitting inside the graph is a separate concern —
// and since spec 031 BUG-001 removed apply's whole-graph walk, no concern of
// apply's at all unless the patch names it (see
// TestApplyRetiredKeyPatchInGraphTreeIsLeftUntouched).
//
// spec 021 US3 Acceptance Scenario 1: the retired key is refused, naming the
// file, the offending key and its replacement, leaving git untouched
// (FR-003, FR-007, SC-007).
func TestApplyLegacyKindPatchRefusedNamingReplacement(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, t.TempDir(), "legacy.patch.md", withIdentity("kind: patch"))

	before := gitLog(t, dir)

	out, err := sut(NewApplyCmd(), []string{patch})

	it.Then(t).Must(it.True(errors.Is(err, core.ErrManifestLegacyKind)))
	it.Then(t).
		Should(it.String(err.Error()).Contain("retired")).
		Should(it.String(err.Error()).Contain("legacy.patch.md")).
		Should(it.String(err.Error()).Contain(`"@type": patch`)).
		Should(it.Equal(before, gitLog(t, dir))).
		Should(it.Equal("", strings.TrimSpace(out)))

	_, statErr := os.Stat(filepath.Join(dir, "Source", "rescorla-2026-tls13.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

// arc apply conflict.patch.md
// spec 021 US3 Acceptance Scenario 2: a self-contradictory manifest is
// refused with an error distinct from the retired-key one, naming both
// disagreeing values (FR-005).
func TestApplyConflictingIdentityKeysRefusedNamingBothValues(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, t.TempDir(), "conflict.patch.md", withIdentity("\"@type\": patch\nkind: source"))

	before := gitLog(t, dir)

	_, err := sut(NewApplyCmd(), []string{patch})

	it.Then(t).Must(it.True(errors.Is(err, core.ErrManifestTypeConflict)))
	it.Then(t).
		Should(it.String(err.Error()).Contain("conflict.patch.md")).
		Should(it.String(err.Error()).Contain("patch")).
		Should(it.String(err.Error()).Contain("source")).
		Should(it.Equal(before, gitLog(t, dir)))
	it.Then(t).ShouldNot(it.True(errors.Is(err, core.ErrManifestLegacyKind)))
}

// arc apply both.patch.md
// spec 021 US3 Acceptance Scenario 3: a redundant retired key alongside an
// agreeing "@type" is not a contradiction — it is ignored and the patch
// applies normally (FR-004).
func TestApplyBothIdentityKeysAgreeingApplies(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "both.patch.md", withIdentity("\"@type\": patch\nkind: patch"))

	out, err := sut(NewApplyCmd(), []string{patch})

	it.Then(t).ShouldNot(it.Error(out, err))
	assertIsFile(t, filepath.Join(dir, "Source", "rescorla-2026-tls13.md"))
}

// arc apply notapatch.md
// spec 021 US3 Acceptance Scenario 5: a manifest declaring neither identity
// key still produces the pre-existing manifest-invalid message, unchanged in
// wording (FR-006).
func TestApplyNoIdentityKeyProducesPreexistingMessage(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, t.TempDir(), "notapatch.md", withIdentity("title: Reading notes"))

	before := gitLog(t, dir)

	_, err := sut(NewApplyCmd(), []string{patch})

	it.Then(t).Must(it.True(errors.Is(err, core.ErrManifestInvalid)))
	it.Then(t).
		Should(it.String(err.Error()).Contain("manifest is missing a mandatory field or uses the pre-0.5 node format")).
		Should(it.Equal(before, gitLog(t, dir)))
	it.Then(t).ShouldNot(it.True(errors.Is(err, core.ErrManifestLegacyKind)))
	it.Then(t).ShouldNot(it.True(errors.Is(err, core.ErrManifestNotAPatch)))
}

// arc apply cased.patch.md
// spec 021 edge case (FR-016): "@type": Patch is the right kind in the wrong
// casing — rejected with a message that shows the offending value.
func TestApplyCapitalizedTypeValueRefused(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, t.TempDir(), "cased.patch.md", withIdentity(`"@type": Patch`))

	_, err := sut(NewApplyCmd(), []string{patch})

	it.Then(t).Must(it.True(errors.Is(err, core.ErrManifestNotAPatch)))
	it.Then(t).Should(it.String(err.Error()).Contain("Patch"))
}

// arc apply bare.patch.md
// spec 021 edge case (FR-015): a bare "@type:" is a hard YAML syntax error —
// the user must meet the quoting sentence, not the raw lexer error.
func TestApplyBareIdentityKeyReportsQuoting(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, t.TempDir(), "bare.patch.md", withIdentity(`@type: patch`))

	_, err := sut(NewApplyCmd(), []string{patch})

	it.Then(t).Must(it.True(errors.Is(err, core.ErrIdentityQuoting)))
	it.Then(t).
		Should(it.String(err.Error()).Contain("bare.patch.md")).
		Should(it.String(err.Error()).Contain("must be a quoted YAML string key, found it unquoted"))
	it.Then(t).ShouldNot(it.String(err.Error()).Contain("cannot start any token"))
}

// arc apply tls13.patch.md, with a retired-key patch sitting in the graph
// RE-SCOPED — spec 031 BUG-001 (T068). This asserted spec 021 T048: a
// retired-key patch left inside the graph tree made apply abort with
// ErrManifestLegacyKind, because guardNoOldFormatNodes walked the whole
// graph and met it. The walk is gone (spec 031 FR-033) — it could not tell a
// stray exchange file from a host project's own markdown, and apply now
// reads only what its patch targets.
//
// The retired key is still refused wherever it is actually read: as the
// applied patch itself (TestApplyLegacyKindPatchRefusedNamingReplacement,
// which is spec 021 US3 Acceptance Scenario 1 and the case users hit), and
// by arc lint's own walk. What is asserted here is that a file the patch
// never names is simply not apply's business: the apply succeeds and the
// stray file is left exactly as it was.
func TestApplyRetiredKeyPatchInGraphTreeIsLeftUntouched(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	stale := withIdentity("kind: patch")
	seedNode(t, dir, "Entity/stale.patch.md", stale)
	chdir(t, dir)
	patch := writePatchFile(t, t.TempDir(), "tls13.patch.md", tls13Patch)

	out, err := sut(NewApplyCmd(), []string{patch})

	it.Then(t).ShouldNot(it.Error(out, err))
	assertIsFile(t, filepath.Join(dir, "Source", "rescorla-2026-tls13.md"))
	it.Then(t).Should(it.Equal(stale, readFile(t, filepath.Join(dir, "Entity", "stale.patch.md"))))
}

// arc apply tls13.patch.md, with a conformant patch sitting in the graph
// spec 021: the same walk passes over a well-formed exchange document
// entirely — a patch in the tree is a valid, distinct concept, never a node.
func TestApplyTypeKeyPatchInGraphTreeIsPassedOver(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedNode(t, dir, "Entity/exchange.patch.md", tlsEntityDocBPatch)
	chdir(t, dir)
	patch := writePatchFile(t, t.TempDir(), "tls13.patch.md", tls13Patch)

	out, err := sut(NewApplyCmd(), []string{patch})

	it.Then(t).ShouldNot(it.Error(out, err))
	assertIsFile(t, filepath.Join(dir, "Source", "rescorla-2026-tls13.md"))
}

// arc apply cased-body.patch.md
// spec 021 T049 / contracts/cli-contract.md edge cases: when a document
// declares a valid '"@type": patch' manifest but its body carries a spec 019
// CamelCase violation, the *body* error must remain the reported reason. The
// identity check runs first and passes, so it must not shadow the real
// defect.
func TestApplyTypeKeyPatchWithBodyCasingViolationReportsBodyError(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	body := strings.Replace(tls13Patch, `"@type": Entity`, `"@type": entity`, 1)
	patch := writePatchFile(t, t.TempDir(), "cased-body.patch.md", body)

	_, err := sut(NewApplyCmd(), []string{patch})

	it.Then(t).Must(it.True(errors.Is(err, core.ErrTypeCasing)))
	it.Then(t).Should(it.String(err.Error()).Contain("entity"))
	// ErrManifestInvalid stays in the chain — parsePatchBody has wrapped body
	// failures in it since before this feature — but no *identity* error does,
	// which is what "the identity check did not shadow the body error" means.
	it.Then(t).
		ShouldNot(it.True(errors.Is(err, core.ErrManifestLegacyKind))).
		ShouldNot(it.True(errors.Is(err, core.ErrManifestNotAPatch))).
		ShouldNot(it.True(errors.Is(err, core.ErrManifestTypeConflict)))
}

// ---------------------------------------------------------------------------
// specs/022-reference-type-folders — ARCNET-CORE v0.11
// ---------------------------------------------------------------------------

// v011Patch carries one node of every core content type this feature
// touches, plus one domain-profile type the tool does not recognize. Each
// content node opens with body prose, so every node exercises the
// leading-prose derivation as well as the folder derivation.
const v011Patch = `---
"@type": patch
document: doe-2026-probe
published: 2026-04-12
title: A probe document
---
# Source

## doe-2026-probe
` + "```yaml" + `
"@id": "doe-2026-probe"
"@type": Source
title: A probe document
author: [Jane Doe]
published: "2026-04-12"
` + "```" + `

An abstract of the probe document.

## Mentions
- mentions:: [[Probe Concept]]

# Entity

## Probe Concept
` + "```yaml" + `
"@id": "Probe Concept"
"@type": Entity
category: [independent, abstract, occurrent, script]
` + "```" + `

A definition of the probe concept.

- mentionedIn:: [[doe-2026-probe]]

# Resource

## probe-fragment
` + "```yaml" + `
"@id": "probe-fragment"
"@type": Resource
tags: [probe, fragment]
` + "```" + `

A fragment of the probe document worth keeping around.

- mentionedIn:: [[doe-2026-probe]]

# Reference

## rescorla-2018-tls13
` + "```yaml" + `
"@id": "rescorla-2018-tls13"
"@type": Reference
title: "The Transport Layer Security (TLS) Protocol Version 1.3"
published: "2018-08-10"
` + "```" + `

The normative definition of the handshake this document analyses.

# Thought

## a-passing-thought
` + "```yaml" + `
"@id": "a-passing-thought"
"@type": Thought
tags: [speculation]
` + "```" + `

An unrecognized domain type's node, carrying leading prose of its own.
`

// coreIndex builds the schema index straight from the seeded vocabulary —
// the same values arc init writes to _schema/ — so a test can re-parse a
// written node file without mounting the graph's own store.
func coreIndex() core.Index {
	return core.Index{
		Predicates: schemakernel.CorePredicateDefs,
		Types:      schemakernel.CoreTypeDefs,
	}
}

// parseNodeFile re-parses a written node document back into a core.Node,
// exposing which prose key its body actually landed under. Reading the
// rendered bytes cannot answer that on its own: a leading-prose slot is
// rendered bare, without a heading naming its predicate.
func parseNodeFile(t *testing.T, path string) core.Node {
	t.Helper()
	raw, err := os.ReadFile(path)
	it.Then(t).Should(it.Nil(err))

	node, err := core.ParseNode(bytes.NewReader(raw), coreIndex())
	it.Then(t).Should(it.Nil(err))
	return node
}

// violationsFor returns the lint report lines owned by path. Lint prints one
// line per violation, prefixed with "<path>[:<line>] — [<rule>] <message>".
func violationsFor(t *testing.T, lintOut, path string) []string {
	t.Helper()
	var out []string
	for _, line := range strings.Split(lintOut, "\n") {
		if strings.Contains(line, path+" —") || strings.Contains(line, path+":") {
			out = append(out, line)
		}
	}
	return out
}

// rootDirNames reports the directory names the filesystem actually stores
// at the graph root. Contract C7: os.Stat cannot distinguish "Timeline"
// from "timeline" on APFS, so a case-sensitive assertion must compare the
// names ReadDir returns.
func rootDirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	it.Then(t).Should(it.Nil(err))

	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out
}

// assertNodeAt asserts that folder holds a node file named exactly
// "<id>.md", comparing against the basenames the filesystem reports rather
// than stat-ing a constructed path (contract C7) — and, unlike
// assertIsFile, reports a plain failure instead of panicking on a nil
// FileInfo when the node is missing, so one red test cannot take the whole
// test binary down with it.
func assertNodeAt(t *testing.T, dir, folder, id string) {
	t.Helper()
	it.Then(t).Should(it.Seq(dirEntryNames(t, filepath.Join(dir, folder))).Contain(id + ".md"))
}

// dirEntryNames reports the basenames the filesystem actually stores in
// dir, case included. A missing directory yields no names rather than a
// failure, so a caller asserting a node's absence works whether the whole
// type folder or only the node went away.
func dirEntryNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// applyV011Patch initializes a graph, applies v011Patch from outside the
// graph tree (so the patch document itself is never linted as a node), and
// returns the graph root.
func applyV011Patch(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, t.TempDir(), "probe.patch.md", v011Patch)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))
	return dir
}

// arc apply probe.patch.md
// spec.md US1 Acceptance Scenario 4: a patch carrying a Reference node is
// accepted, written, and reported with no "unknown type" diagnostic —
// Reference is a seeded core type, not a type discovered at apply time.
func TestApplyAcceptsReferenceNodeWithoutUnknownTypeDiagnostic(t *testing.T) {
	dir := applyV011Patch(t)

	assertNodeAt(t, dir, "Reference", "rescorla-2018-tls13")

	// A type the tool already knows is never auto-registered, so applying
	// one must not rewrite its seeded schema document. Thought, which it
	// does not know, is the control: that one is registered on first sight.
	// specs/023 FR-028: §11.6 v0.11 requires "title" alone; "relevance" is
	// retained as Optional. "ref" is retired outright under CORE 0.12
	// (BUG-001/FR-034) — no longer part of Reference's conformant shape.
	referenceSchema := readFile(t, filepath.Join(dir, "_schema", "Class", "Reference.md"))
	it.Then(t).
		Should(it.String(referenceSchema).Contain("required:: [[title]]")).
		Should(it.String(referenceSchema).Contain("optional:: [[relevance]]")).
		ShouldNot(it.String(referenceSchema).Contain("optional:: [[ref]]"))
}

// arc apply probe.patch.md && arc lint
// spec.md US1 Acceptance Scenario 6: a Reference node carrying title,
// published, and leading prose draws no type-conformance diagnostic about a
// missing required predicate or an undeclared one.
//
// The assertion names the three required predicates rather than demanding
// the node be violation-free: arc apply writes no "created" timestamp, so
// every applied node of every type carries one pre-existing typeRequires
// violation for it. That gap predates this feature and is scoped to
// CORE-FIX's Feature 023 (research.md §7 follow-up 3).
func TestApplyReferenceNodeIsTypeConformant(t *testing.T) {
	dir := applyV011Patch(t)
	assertNodeAt(t, dir, "Reference", "rescorla-2018-tls13")

	lintOut, _ := sut(lint.NewLintCmd(), nil)
	got := violationsFor(t, lintOut, "Reference/rescorla-2018-tls13.md")

	for _, line := range got {
		it.Then(t).
			ShouldNot(it.String(line).Contain("not permitted by type")).
			ShouldNot(it.String(line).Contain(`requires predicate "title"`)).
			ShouldNot(it.String(line).Contain(`requires predicate "ref"`)).
			ShouldNot(it.String(line).Contain(`requires predicate "relevance"`))
	}
}

// arc apply probe.patch.md
// spec.md US2 Acceptance Scenarios 1 and 2: a Resource node's leading prose
// is stored under text and nothing under relevance; a Reference node's is
// stored under relevance (contract C4).
func TestApplyStoresLeadingProseUnderTheTypesOwnPredicate(t *testing.T) {
	dir := applyV011Patch(t)

	resource := parseNodeFile(t, filepath.Join(dir, "Resource", "probe-fragment.md"))
	it.Then(t).
		Should(it.String(resource.Texts["text"]).Contain("A fragment of the probe document")).
		Should(it.Equal("", resource.Texts["relevance"]))

	reference := parseNodeFile(t, filepath.Join(dir, "Reference", "rescorla-2018-tls13.md"))
	it.Then(t).
		Should(it.String(reference.Texts["relevance"]).Contain("The normative definition of the handshake")).
		Should(it.Equal("", reference.Texts["text"]))
}

// arc apply probe.patch.md
// spec.md US2 Acceptance Scenarios 5 and 6: the derivation is unchanged for
// every other type — Source keeps abstract, Entity keeps definition, and a
// type with no type-specific prose predicate keeps text.
func TestApplyLeavesOtherTypesLeadingProseDerivationUnchanged(t *testing.T) {
	dir := applyV011Patch(t)

	source := parseNodeFile(t, filepath.Join(dir, "Source", "doe-2026-probe.md"))
	it.Then(t).Should(it.String(source.Texts["abstract"]).Contain("An abstract of the probe document"))

	entity := parseNodeFile(t, filepath.Join(dir, "Entity", "Probe Concept.md"))
	// BUG-001/FR-030: Entity's leading-prose predicate is now "text", not
	// "definition".
	it.Then(t).Should(it.String(entity.Texts["text"]).Contain("A definition of the probe concept"))

	thought := parseNodeFile(t, filepath.Join(dir, "Thought", "a-passing-thought.md"))
	it.Then(t).Should(it.String(thought.Texts["text"]).Contain("An unrecognized domain type's node"))
}

// arc apply probe.patch.md && arc lint
// spec.md US2 Acceptance Scenario 4: the applied Resource draws no
// diagnostic for a missing required text or an undeclared relevance — the
// exact pair the pre-v0.11 derivation would have produced against the
// corrected type definition.
func TestApplyResourceNodeCarriesTextNotRelevance(t *testing.T) {
	dir := applyV011Patch(t)
	assertNodeAt(t, dir, "Resource", "probe-fragment")

	lintOut, _ := sut(lint.NewLintCmd(), nil)
	got := violationsFor(t, lintOut, "Resource/probe-fragment.md")

	for _, line := range got {
		it.Then(t).
			ShouldNot(it.String(line).Contain(`requires predicate "text"`)).
			ShouldNot(it.String(line).Contain(`predicate "relevance" is not permitted`))
	}
}

// arc apply probe.patch.md
// spec.md US3 Acceptance Scenario 2: every core content node lands at the
// exact path <TypeName>/<id>.md.
func TestApplyFilesCoreNodesInTypeNamedFolders(t *testing.T) {
	dir := applyV011Patch(t)

	for folder, id := range map[string]string{
		"Source":    "doe-2026-probe",
		"Entity":    "Probe Concept",
		"Resource":  "probe-fragment",
		"Reference": "rescorla-2018-tls13",
	} {
		names := rootDirNames(t, dir)
		it.Then(t).Should(it.Seq(names).Contain(folder))
		assertNodeAt(t, dir, folder, id)
	}
}

// arc apply probe.patch.md
// spec.md US3 Acceptance Scenario 3: a domain type the tool does not
// recognize is filed under a folder named exactly that type — no
// pluralizing suffix, no case change. Asserted against the directory names
// the filesystem reports, because os.Stat on APFS answers yes to
// "Thought", "Thoughts", and "thoughts" indiscriminately (contract C7).
func TestApplyFilesUnknownDomainTypeUnderItsVerbatimName(t *testing.T) {
	dir := applyV011Patch(t)

	names := rootDirNames(t, dir)
	it.Then(t).Should(it.Seq(names).Contain("Thought"))
	for _, wrong := range []string{"Thoughts", "thoughts", "thought"} {
		for _, got := range names {
			it.Then(t).Should(it.True(got != wrong))
		}
	}
	assertNodeAt(t, dir, "Thought", "a-passing-thought")
}

// arc apply probe.patch.md
// spec.md US3 Acceptance Scenario 4: timeline/ stays exempt. Timeline nodes
// remain bucketed by granularity and no flat Timeline/ folder is ever
// created for them.
func TestApplyKeepsTimelineNodesBucketedAndCreatesNoFlatFolder(t *testing.T) {
	dir := applyV011Patch(t)

	assertIsFile(t, filepath.Join(dir, "timeline", "yearly", "2026.md"))
	assertIsFile(t, filepath.Join(dir, "timeline", "monthly", "2026-04.md"))

	for _, got := range rootDirNames(t, dir) {
		it.Then(t).Should(it.True(got != "Timeline"))
	}
}

// -----------------------------------------------------------------------
// specs/023-core-vocabulary-conformance — User Story 1
//
// No type-specific prose predicate declares firstWriteWin any longer.
// "abstract" joined "text" (Entity's leading prose, BUG-001/FR-030) in
// declaring append, "definition" is retired outright, and "relevance"
// (Reference's leading prose) has since followed them — so a reworded
// Source abstract, Entity body, or Reference relevance note now
// accumulates as a second paragraph rather than flagging, exactly like
// any other append-declared prose key (schema.go).
//
// Two facts about arc apply shape every fixture below.
//
//  1. plan.md F2 / research.md D4 — mergeText already drops an incoming
//     paragraph whose Jaccard-over-3-word-shingles similarity to an
//     existing one exceeds 0.8, so a byte-identical re-contribution is
//     ALREADY a no-op on main. A test that must genuinely fail before the
//     fix therefore uses substantially REWORDED prose, which falls under
//     that threshold and is appended as a second paragraph by the old
//     append behaviour.
//
//  2. service.Apply's idempotency guard is keyed on Source/<document>.md
//     being git-tracked, so re-ingesting the SAME document id is skipped
//     outright and never reaches core.Merge at all. Prose drift in the
//     field arrives from a SECOND document whose patch also describes a
//     shared node — the ordinary cross-document merge — which is what
//     crossDocumentPatch below models.
// -----------------------------------------------------------------------

const tls13Abstract = "A design retrospective on the TLS 1.3 handshake and the reasoning behind its single round trip."
const tls13Definition = "A cryptographic protocol that establishes an authenticated and confidential channel between two peers."
const tls13Relevance = "Worth keeping because it is the normative text every implementation claim in this graph is measured against."

// The rewordings deliberately share almost no 3-word shingles with the
// originals, so mergeText's near-duplicate guard does NOT suppress them and
// the old append behaviour genuinely doubles each value.
const tls13AbstractReworded = "Why one round trip was chosen, looked at again several years after the protocol shipped."
const tls13DefinitionReworded = "An authenticated, private transport negotiated by two endpoints using standardized cryptography."
const tls13RelevanceReworded = "Kept as the authoritative wording against which claims recorded here are checked."

// conflictMarkerToken is internal/core.conflictMarker's opening line — the
// documented, user-visible shape of a flagged divergence (VISION.md). The
// established value is the marker's left-hand side, so it survives verbatim
// instead of being absorbed into a stack of near-synonymous paragraphs.
const conflictMarkerToken = "<<<<<<< existing"

func tls13ProsePatch(abstract, definition, relevance string) string {
	return `---
"@type": patch
document: rescorla-2026-tls13
published: 2026-04-12
title: "TLS 1.3: Design and Rationale"
---
# Source

## rescorla-2026-tls13
` + "```yaml" + `
"@id": "rescorla-2026-tls13"
"@type": Source
title: "TLS 1.3: Design and Rationale"
published: "2026-04-12"
` + "```" + `

` + abstract + `

## Mentions
- mentions:: [[Transport Layer Security]]

# Entity

## Transport Layer Security
` + "```yaml" + `
"@id": "Transport Layer Security"
"@type": Entity
category: [independent, abstract, occurrent, script]
` + "```" + `

` + definition + `

# Reference

## RFC 8446
` + "```yaml" + `
"@id": "RFC 8446"
"@type": Reference
title: "The Transport Layer Security Protocol Version 1.3"
` + "```" + `

` + relevance + `
`
}

// crossDocumentPatch is a SECOND document's patch that also describes the
// three nodes tls13ProsePatch established — the ordinary cross-document
// contribution through which a shared node's prose actually merges.
func crossDocumentPatch(abstract, definition, relevance, extraCites string) string {
	return `---
"@type": patch
document: chen-2026-pqkex
published: 2026-04-28
title: "Post-Quantum Key Exchange in Practice"
---
# Source

## chen-2026-pqkex
` + "```yaml" + `
"@id": "chen-2026-pqkex"
"@type": Source
title: "Post-Quantum Key Exchange in Practice"
published: "2026-04-28"
` + "```" + `

A survey of post-quantum key exchange deployments observed in the wild.

## Mentions
- mentions:: [[Transport Layer Security]]

## rescorla-2026-tls13
` + "```yaml" + `
"@id": "rescorla-2026-tls13"
"@type": Source
title: "TLS 1.3: Design and Rationale"
published: "2026-04-12"
` + "```" + `

` + abstract + `
` + extraCites + `

# Entity

## Transport Layer Security
` + "```yaml" + `
"@id": "Transport Layer Security"
"@type": Entity
category: [independent, abstract, occurrent, script]
` + "```" + `

` + definition + `

# Reference

## RFC 8446
` + "```yaml" + `
"@id": "RFC 8446"
"@type": Reference
title: "The Transport Layer Security Protocol Version 1.3"
` + "```" + `

` + relevance + `
`
}

// applyProseFixture applies the establishing patch, then a second
// document's patch carrying the given prose for the same three nodes.
func applyProseFixture(t *testing.T, dir, abstract, definition, relevance, extraCites string) string {
	t.Helper()
	first := writePatchFile(t, dir, "tls13.patch.md", tls13ProsePatch(tls13Abstract, tls13Definition, tls13Relevance))
	_, err := sut(NewApplyCmd(), []string{first})
	it.Then(t).Should(it.Nil(err))

	second := writePatchFile(t, dir, "pqkex.patch.md", crossDocumentPatch(abstract, definition, relevance, extraCites))
	_, stderr, err := sutCaptureStderr(t, NewApplyCmd(), []string{second})
	it.Then(t).Should(it.Nil(err))
	return stderr
}

// arc apply tls13.patch.md, then pqkex.patch.md
// Scenario 1 from spec.md US1 (023), amended: "abstract" declares append,
// not firstWriteWin (schema.go), so a substantially reworded summary — one
// that falls under mergeText's near-duplicate threshold — accumulates
// alongside the established wording instead of preserving it and flagging.
// The established value still survives verbatim; what changed is that the
// divergence is absorbed silently, because append never flags.
func TestApplyRewordedAbstractAppendsAlongsideFirstValue(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	stderr := applyProseFixture(t, dir, tls13AbstractReworded, tls13Definition, tls13Relevance, "")

	content := readFile(t, filepath.Join(dir, "Source", "rescorla-2026-tls13.md"))

	it.Then(t).
		Should(it.String(content).Contain("the reasoning behind its single round trip")).
		Should(it.String(content).Contain("Why one round trip was chosen")).
		ShouldNot(it.String(content).Contain(conflictMarkerToken)).
		ShouldNot(it.String(stderr).Contain("merge conflict was flagged"))
}

// arc apply tls13.patch.md, then pqkex.patch.md carrying identical prose
// Scenario 2 from spec.md US1 (023): applying the same prose twice leaves
// every first-fixed value byte-identical, with nothing flagged.
func TestApplyIdenticalProseTwiceIsByteIdentical(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	sourcePath := filepath.Join(dir, "Source", "rescorla-2026-tls13.md")
	entityPath := filepath.Join(dir, "Entity", "Transport Layer Security.md")
	referencePath := filepath.Join(dir, "Reference", "RFC 8446.md")

	first := writePatchFile(t, dir, "tls13.patch.md", tls13ProsePatch(tls13Abstract, tls13Definition, tls13Relevance))
	_, err := sut(NewApplyCmd(), []string{first})
	it.Then(t).Should(it.Nil(err))

	before := map[string]string{
		sourcePath:    prosePayload(readFile(t, sourcePath)),
		entityPath:    prosePayload(readFile(t, entityPath)),
		referencePath: prosePayload(readFile(t, referencePath)),
	}

	second := writePatchFile(t, dir, "pqkex.patch.md", crossDocumentPatch(tls13Abstract, tls13Definition, tls13Relevance, ""))
	_, err = sut(NewApplyCmd(), []string{second})
	it.Then(t).Should(it.Nil(err))

	for path, want := range before {
		got := prosePayload(readFile(t, path))
		it.Then(t).
			Should(it.Equal(want, got)).
			ShouldNot(it.String(got).Contain(conflictMarkerToken))
	}
}

// prosePayload extracts the node body below its front matter, so an
// assertion compares authored prose rather than the "updated" timestamp
// arc apply legitimately restamps on every merge.
func prosePayload(content string) string {
	parts := strings.SplitN(content, "\n---\n", 2)
	if len(parts) != 2 {
		return content
	}
	return strings.TrimSpace(parts[1])
}

// arc apply tls13.patch.md (three times, same file)
// Scenario 3 from spec.md US1 (023): a third apply reports no change to
// commit.
func TestApplyThirdApplyReportsNothingToCommit(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13ProsePatch(tls13Abstract, tls13Definition, tls13Relevance))

	_, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Nil(err))
	_, err = sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Nil(err))

	before := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).Should(it.String(out).Contain("already tracked"))

	it.Then(t).Should(it.Equal(before, strings.TrimSpace(runGit(t, dir, "log", "--oneline"))))
}

// arc apply tls13.patch.md, then pqkex.patch.md
// Scenario 4 from spec.md US1 (023), amended twice. An Entity's leading
// prose stopped being first-fixed under CORE 0.12 (BUG-001/FR-030):
// "definition" is retired outright and its replacement ("text") is
// deliberately MergeAppend, to avoid the merge-scoping mechanism a
// per-type override would require (plan.md F7/Complexity Tracking). A
// Reference's "relevance" now declares append for a reason of its own:
// several documents may each have their own reason for pointing at the
// same external work, and those reasons are contributions to accumulate,
// not a slot the first writer fixes. Both therefore append alongside the
// established wording, and neither is flagged.
//
// research.md D8 / plan.md F4 defects 2-3: these two used to be the
// leading-prose predicates for Entity and Reference (spec 022 keying), so
// this test still pins the key each one lands under — what changed is that
// both now accumulate rather than preserve-and-flag.
func TestApplyRewordedDefinitionAndRelevanceAppendAlongsideFirstValue(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	applyProseFixture(t, dir, tls13Abstract, tls13DefinitionReworded, tls13RelevanceReworded, "")

	entity := readFile(t, filepath.Join(dir, "Entity", "Transport Layer Security.md"))
	reference := readFile(t, filepath.Join(dir, "Reference", "RFC 8446.md"))

	it.Then(t).
		ShouldNot(it.String(entity).Contain(conflictMarkerToken)).
		Should(it.String(entity).Contain("authenticated and confidential channel between two peers")).
		Should(it.String(entity).Contain("An authenticated, private transport negotiated")).
		ShouldNot(it.String(reference).Contain(conflictMarkerToken)).
		Should(it.String(reference).Contain("normative text every implementation claim")).
		Should(it.String(reference).Contain("Kept as the authoritative wording"))
}

// arc apply tls13.patch.md, then pqkex.patch.md with overlapping citations
// Scenario 6 from spec.md US1 (023): citation targets already present are
// not duplicated — the citation predicate combines by union (FR-014).
func TestApplyOverlappingCitationsAppearExactlyOnce(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	establishing := tls13ProsePatch(tls13Abstract, tls13Definition, tls13Relevance)
	establishing = strings.Replace(establishing,
		"## Mentions\n- mentions:: [[Transport Layer Security]]",
		"## Mentions\n- mentions:: [[Transport Layer Security]]\n\n## Cites\n- cites:: [[RFC 8446]]\n- cites:: [[RFC 5246]]", 1)
	first := writePatchFile(t, dir, "tls13.patch.md", establishing)
	_, err := sut(NewApplyCmd(), []string{first})
	it.Then(t).Should(it.Nil(err))

	second := writePatchFile(t, dir, "pqkex.patch.md", crossDocumentPatch(
		tls13Abstract, tls13Definition, tls13Relevance,
		"\n## Cites\n- cites:: [[RFC 8446]]\n- cites:: [[RFC 8447]]\n"))
	_, err = sut(NewApplyCmd(), []string{second})
	it.Then(t).Should(it.Nil(err))

	content := readFile(t, filepath.Join(dir, "Source", "rescorla-2026-tls13.md"))
	it.Then(t).
		Should(it.Equal(1, strings.Count(content, "[[RFC 8446]]"))).
		Should(it.Equal(1, strings.Count(content, "[[RFC 5246]]"))).
		Should(it.Equal(1, strings.Count(content, "[[RFC 8447]]")))
}

// ---------------------------------------------------------------------------
// specs/023-core-vocabulary-conformance — User Story 4
//
// author, about, and genre are part of the seeded vocabulary (CORE §10.2),
// so a patch recording them resolves against it and lints clean.
//
// research.md D5 corrects spec.md's original premise: nothing is
// auto-registered today, because distinctPredicates walks node.Edges and
// node.Texts only and all three are role: meta (front-matter Attrs). The
// real symptom is two lint violations per occurrence — predicateRegistered
// and typeOptional.
// ---------------------------------------------------------------------------

const metadataPredicatePatch = `---
"@type": patch
document: rescorla-2026-tls13
published: 2026-04-12
title: "TLS 1.3: Design and Rationale"
---
# Source

## rescorla-2026-tls13
` + "```yaml" + `
"@id": "rescorla-2026-tls13"
"@type": Source
title: "TLS 1.3: Design and Rationale"
published: "2026-04-12"
author: Eric Rescorla
about: [technique, technology]
genre: paper
` + "```" + `

A design retrospective on the TLS 1.3 handshake.

## Mentions
- mentions:: [[Transport Layer Security]]

# Entity

## Transport Layer Security
` + "```yaml" + `
"@id": "Transport Layer Security"
"@type": Entity
category: [independent, abstract, occurrent, script]
` + "```" + `

A cryptographic protocol that establishes an authenticated and confidential channel.
`

// propertyDocuments lists every _schema/Property/ basename present in dir.
func propertyDocuments(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(dir, "_schema", "Property"))
	it.Then(t).Should(it.Nil(err))

	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// arc init (seeded vocabulary)
// Scenario 1 from spec.md US4 (023): definitions exist for the authorship,
// aboutness, and genre predicates, each declaring the role and merge CORE
// §10.2 assigns it (FR-011; contract C2.2d).
func TestApplySeededVocabularyRegistersMetadataPredicates(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)

	for _, name := range []string{"author", "about", "genre"} {
		// Read defensively rather than through assertIsFile/readFile: both
		// dereference a nil os.FileInfo when the document is absent, and a
		// panic in a red-phase assertion takes the whole test binary with
		// it instead of failing this one case.
		raw, rerr := os.ReadFile(filepath.Join(dir, "_schema", "Property", name+".md"))
		it.Then(t).Should(it.Nil(rerr))

		content := string(raw)
		it.Then(t).
			Should(it.String(content).Contain(`"@type": Property`)).
			Should(it.String(content).Contain("role: meta")).
			Should(it.String(content).Contain("merge: union"))
	}
}

// arc apply metadata.patch.md, then arc lint
// Scenario 2 from spec.md US4 (023): applying a patch using all three
// draws no unregistered-predicate and no undeclared-predicate diagnostic
// for any of them (FR-012).
func TestApplyMetadataPredicatesLintClean(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	patch := writePatchFile(t, dir, "metadata.patch.md", metadataPredicatePatch)
	_, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Nil(err))

	lintOut, _ := sut(lint.NewLintCmd(), nil)

	for _, name := range []string{"author", "about", "genre"} {
		it.Then(t).
			ShouldNot(it.String(lintOut).Contain(`predicate "` + name + `" is not registered`)).
			ShouldNot(it.String(lintOut).Contain(`predicate "` + name + `" is not permitted`))
	}
}

// arc apply metadata.patch.md
// Scenario 3 from spec.md US4 (023): zero predicates are created — the
// three resolve against the seeded vocabulary rather than being registered
// on the spot (SC-005).
func TestApplyMetadataPredicatesCreateNoSchemaDocuments(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	before := propertyDocuments(t, dir)

	patch := writePatchFile(t, dir, "metadata.patch.md", metadataPredicatePatch)
	_, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Nil(err))

	it.Then(t).Should(it.Seq(propertyDocuments(t, dir)).Equal(before...))
}

// arc apply metadata.patch.md, then a second document naming another author
// Scenario 4 from spec.md US4 (023): two authors recorded across two
// separate applies leave the document carrying both names, each exactly
// once — the authorship predicate combines by union (FR-011).
func TestApplyRepeatedAuthorsUnionExactlyOnce(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	first := writePatchFile(t, dir, "metadata.patch.md", metadataPredicatePatch)
	_, err := sut(NewApplyCmd(), []string{first})
	it.Then(t).Should(it.Nil(err))

	// A SECOND document that also describes the same Source — the only
	// route by which a shared node's metadata actually merges, since
	// service.Apply skips a document whose own Source node is tracked.
	crossDocument := strings.Replace(metadataPredicatePatch,
		"document: rescorla-2026-tls13", "document: chen-2026-pqkex", 1)
	crossDocument = strings.Replace(crossDocument,
		"author: Eric Rescorla", "author: [Eric Rescorla, Hugo Krawczyk]", 1)
	second := writePatchFile(t, dir, "pqkex.patch.md", crossDocument)
	_, err = sut(NewApplyCmd(), []string{second})
	it.Then(t).Should(it.Nil(err))

	content := readFile(t, filepath.Join(dir, "Source", "rescorla-2026-tls13.md"))
	it.Then(t).
		Should(it.Equal(1, strings.Count(content, "Eric Rescorla"))).
		Should(it.Equal(1, strings.Count(content, "Hugo Krawczyk")))
}

// arc init (seeded vocabulary)
// Scenario 5 from spec.md US4 (023): the citation predicate declares the
// merge behaviour that combines without duplication (union), not the one
// that appends (FR-014; contract C2.2f).
func TestApplySeededCitationPredicateDeclaresUnion(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)

	content := readFile(t, filepath.Join(dir, "_schema", "Property", "cites.md"))

	it.Then(t).
		Should(it.String(content).Contain("merge: union")).
		ShouldNot(it.String(content).Contain("merge: append"))
}

// ---------------------------------------------------------------------------
// specs/024-lint-conformance-gaps — User Story 1
//
// arc apply rejects the entire operation, before any file is written, when
// a patch would introduce or modify a node — content or schema-implied —
// whose identity contains an ARCNET-CORE §7.1 forbidden character.
// ---------------------------------------------------------------------------

// arc apply tls13.patch.md
// Scenario 1 from spec.md US1 (024): a patch introducing a node whose
// identity is "Handshake/Protocol" is rejected, naming "/" and its position
// (10), with the graph directory left byte-for-byte unchanged.
func TestApplyUnsafeIdentityRejectedNamingCharacterAndPosition(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	unsafe := strings.ReplaceAll(tls13Patch, "Transport Layer Security", "Handshake/Protocol")
	patch := writePatchFile(t, dir, "tls13.patch.md", unsafe)

	before := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).
		Should(it.Error(out, err).Contain(`"/"`)).
		Should(it.Error(out, err).Contain("position 10"))

	after := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))
	it.Then(t).Should(it.Equal(before, after))

	entries, rerr := os.ReadDir(filepath.Join(dir, "Entity"))
	it.Then(t).
		Should(it.Nil(rerr)).
		Should(it.Equal(0, len(entries)))
}

// arc apply tls13.patch.md
// Scenario 2 from spec.md US1 (024): a patch introducing a node whose
// identity is "Handshake Protocol (TLS)" (no forbidden characters) is
// written normally, with no identity error reported.
func TestApplySafeIdentityAppliesNormally(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	safe := strings.ReplaceAll(tls13Patch, "Transport Layer Security", "Handshake Protocol (TLS)")
	patch := writePatchFile(t, dir, "tls13.patch.md", safe)

	out, err := sut(NewApplyCmd(), []string{patch})

	it.Then(t).ShouldNot(it.Error(out, err))
	assertIsFile(t, filepath.Join(dir, "Entity", "Handshake Protocol (TLS).md"))
}

// arc apply tls13.patch.md
// Scenario 3 from spec.md US1 (024): a patch carrying several nodes, only
// one of which has an unsafe identity, is rejected in its entirety — none
// of the patch's nodes are written, including the otherwise-valid Source.
func TestApplyOneUnsafeNodeAmongSeveralRejectsWholePatch(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	unsafe := strings.ReplaceAll(tls13Patch, "Transport Layer Security", "Handshake/Protocol")
	patch := writePatchFile(t, dir, "tls13.patch.md", unsafe)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Error(out, err))

	_, sourceErr := os.Stat(filepath.Join(dir, "Source", "rescorla-2026-tls13.md"))
	it.Then(t).Should(it.True(os.IsNotExist(sourceErr)))
}

const patchWithUnsafeSchemaTypeIdentity = `---
"@type": patch
document: kolesnikov-2026-badtype
published: 2026-05-01
title: "A Test Note"
---
# Source

## kolesnikov-2026-badtype
` + "```yaml" + `
"@id": "kolesnikov-2026-badtype"
"@type": Source
title: "A Test Note"
published: "2026-05-01"
` + "```" + `

A short note.

# Bad/Type

## Something
` + "```yaml\n\"@id\": \"Something\"\n\"@type\": \"Bad/Type\"\n```" + `

A node whose own type name contains a forbidden character.
`

// arc apply badtype.patch.md
// Scenario 4 from spec.md US1 (024): a patch introducing a schema node — here
// a not-yet-registered type name, "Bad/Type" — whose identity contains a
// forbidden character is rejected the same way as a content node, before any
// file (including the otherwise-valid Source) is written.
func TestApplySchemaNodeImpliedUnsafeTypeIdentityRejected(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "badtype.patch.md", patchWithUnsafeSchemaTypeIdentity)

	out, err := sut(NewApplyCmd(), []string{patch})

	it.Then(t).
		Should(it.Error(out, err).Contain(`"/"`)).
		Should(it.Error(out, err).Contain("Bad/Type"))

	_, sourceErr := os.Stat(filepath.Join(dir, "Source", "kolesnikov-2026-badtype.md"))
	it.Then(t).Should(it.True(os.IsNotExist(sourceErr)))

	_, schemaErr := os.Stat(filepath.Join(dir, "_schema", "Class", "Bad"))
	it.Then(t).Should(it.True(os.IsNotExist(schemaErr)))
}

// BUG-008 regression, this bug's own reported reproduction: an extraction
// pipeline emits one entity under two spellings across two documents. On a
// case-insensitive volume (APFS, NTFS) the second apply used to abort the
// WHOLE application with a basename-mismatch error that named a file which
// does not exist; on a case-sensitive one it silently forked the subject
// into two node files. Both outcomes are now specified (spec.md
// FR-026/FR-027) and this test asserts whichever one the volume the test is
// running on actually calls for — so it is meaningful on a developer's
// APFS disk and on a case-sensitive CI volume, without skipping on either.
const lightstepPatchA = `---
"@type": patch
document: alpha-2026-lightstep
published: 2026-04-12
title: "First Document"
---
# Source

## alpha-2026-lightstep
` + "```yaml" + `
"@id": "alpha-2026-lightstep"
"@type": Source
title: "First Document"
author: [Test Author]
published: "2026-04-12"
` + "```" + `

The first document.

## Mentions
- mentions:: [[LightStep]]

# Entity

## LightStep
` + "```yaml" + `
"@id": "LightStep"
"@type": Entity
category: [independent, abstract, occurrent, script]
` + "```" + `

A distributed tracing system.
`

const lightstepPatchB = `---
"@type": patch
document: beta-2026-lightstep
published: 2026-05-13
title: "Second Document"
---
# Source

## beta-2026-lightstep
` + "```yaml" + `
"@id": "beta-2026-lightstep"
"@type": Source
title: "Second Document"
author: [Test Author]
published: "2026-05-13"
` + "```" + `

The second document, whose producer spelled the entity differently.

## Mentions
- mentions:: [[Lightstep]]

# Entity

## Lightstep
` + "```yaml" + `
"@id": "Lightstep"
"@type": Entity
category: [independent, abstract, occurrent, script]
` + "```" + `

Also observed in the second document.
`

func TestApplyCaseVariantIdentityAcrossTwoPatches(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	chdir(t, dir)

	patchA := writePatchFile(t, dir, "alpha.patch.md", lightstepPatchA)
	patchB := writePatchFile(t, dir, "beta.patch.md", lightstepPatchB)

	_, err := sut(NewApplyCmd(), []string{patchA})
	it.Then(t).Should(it.Nil(err))

	// The reported failure: this second apply aborted outright.
	stdout, stderr, err := sutCaptureStderr(t, NewApplyCmd(), []string{patchB})
	it.Then(t).Should(it.Nil(err))
	out := stdout + stderr

	// No diagnostic may name a file that does not exist (spec.md FR-029).
	it.Then(t).ShouldNot(it.String(out).Contain("does not match this file's basename"))

	store, err := fsys.Local{}.Mount(dir)
	it.Then(t).Should(it.Nil(err))

	original := readFile(t, filepath.Join(dir, "Entity", "LightStep.md"))
	_, variantErr := os.Stat(filepath.Join(dir, "Entity", "Lightstep.md"))

	if fsys.FoldsCase(store) {
		// FR-027: one node, still under the identity the graph recorded.
		it.Then(t).Should(
			it.String(original).Contain(`"@id": LightStep`),
			it.String(out).Contain("folded"),
		)
		it.Then(t).ShouldNot(it.String(original).Contain(`"@id": Lightstep`))

		entries, err := os.ReadDir(filepath.Join(dir, "Entity"))
		it.Then(t).Should(it.Nil(err))
		for _, entry := range entries {
			it.Then(t).Should(it.Equal(entry.Name() == "LightStep.md", true))
		}
	} else {
		// FR-005: two genuinely distinct nodes, compared exactly.
		it.Then(t).Should(
			it.Nil(variantErr),
			it.String(original).Contain(`"@id": LightStep`),
			it.String(readFile(t, filepath.Join(dir, "Entity", "Lightstep.md"))).Contain(`"@id": Lightstep`),
		)
	}
}
