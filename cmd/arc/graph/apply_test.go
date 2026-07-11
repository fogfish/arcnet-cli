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
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/spf13/cobra"

	"github.com/fogfish/arcnet-cli/cmd/arc/lint"
	appschema "github.com/fogfish/arcnet-cli/internal/app/schema"
	"github.com/fogfish/arcnet-cli/internal/bios"
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
	for _, folder := range []string{"sources", "entities", "resources", "timeline/yearly", "timeline/monthly", "_schema/types", "_schema/predicates"} {
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

	runGit(t, dir, "init")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "graph(init): empty knowledge graph")
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

// seedSchemaNode writes _schema/types/<kind>.md directly, registering kind
// with merge behavior op — equivalent to a prior arc apply's auto-discovery
// or a hand-edit (spec.md US2/US3), without writing a git commit of its own.
func seedSchemaNode(t *testing.T, dir, kind, op string) {
	t.Helper()
	full := filepath.Join(dir, "_schema", "types", kind+".md")
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
kind: patch
document: rescorla-2026-tls13
published: 2026-04-12
title: "TLS 1.3: Design and Rationale"
---
# Source

## rescorla-2026-tls13
` + "```yaml" + `
"@id": "rescorla-2026-tls13"
"@type": source
title: "TLS 1.3: Design and Rationale"
authors: [Eric Rescorla]
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
"@type": entity
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
	assertIsFile(t, filepath.Join(dir, "sources", "rescorla-2026-tls13.md"))
	assertIsFile(t, filepath.Join(dir, "entities", "Transport Layer Security.md"))
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

	// spec.md User Story 2 (T022): timeline's entries list is the one
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
		Should(it.String(out).Contain("+1 source")).
		Should(it.String(out).Contain("+1 entity")).
		Should(it.String(out).Contain("rescorla-2026-tls13"))
}

const tlsEntitySeed = `---
"@id": "Transport Layer Security"
"@type": entity
title: Transport Layer Security
category: [independent, abstract, occurrent, script]
---
# Transport Layer Security

A cryptographic protocol.

- replaces:: [[SSL Protocol]]
`

const pqkexPatchMergingEntity = `---
kind: patch
document: chen-2026-pqkex
published: 2026-04-28
title: "Post-Quantum Key Exchange in Practice"
---
# Source

## chen-2026-pqkex
` + "```yaml" + `
"@id": "chen-2026-pqkex"
"@type": source
title: "Post-Quantum Key Exchange in Practice"
authors: [Lin Chen]
published: "2026-04-28"
` + "```" + `

Surveys post-quantum key exchange deployment.

# Entity

## Transport Layer Security
` + "```yaml" + `
"@id": "Transport Layer Security"
"@type": entity
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
	seedNode(t, dir, "entities/Transport Layer Security.md", tlsEntitySeed)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "pqkex.patch.md", pqkexPatchMergingEntity)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	entries, err := os.ReadDir(filepath.Join(dir, "entities"))
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(entries)))

	content := readFile(t, filepath.Join(dir, "entities", "Transport Layer Security.md"))
	it.Then(t).
		Should(it.String(content).Contain("replaces:: [[SSL Protocol]]")).
		Should(it.String(content).Contain("requires:: [[Forward Secrecy]]"))
}

const rfcResourceSeedEmptyStatus = `---
"@id": "RFC 8446"
"@type": resource
title: RFC 8446
ref: standard
status: ""
---
# RFC 8446

The normative specification of TLS 1.3.
`

const patchFillsResourceStatus = `---
kind: patch
document: chen-2026-pqkex
published: 2026-04-28
title: "Post-Quantum Key Exchange in Practice"
---
# Source

## chen-2026-pqkex
` + "```yaml" + `
"@id": "chen-2026-pqkex"
"@type": source
title: "Post-Quantum Key Exchange in Practice"
authors: [Lin Chen]
published: "2026-04-28"
` + "```" + `

Surveys post-quantum key exchange deployment.

# Resource

## RFC 8446
` + "```yaml" + `
"@id": "RFC 8446"
"@type": resource
ref: standard
status: read
` + "```" + `

The normative specification of TLS 1.3.
`

// arc apply pqkex.patch.md
// Scenario 2 from spec.md US2: a previously-empty resource field gets filled
func TestApplyMergeFillsEmptyResourceField(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedNode(t, dir, "resources/RFC 8446.md", rfcResourceSeedEmptyStatus)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "pqkex.patch.md", patchFillsResourceStatus)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	content := readFile(t, filepath.Join(dir, "resources", "RFC 8446.md"))
	it.Then(t).Should(it.String(content).Contain("status: read"))
}

const rfcResourceSeedSetStatus = `---
"@id": "RFC 8446"
"@type": resource
title: RFC 8446
ref: standard
status: read
category: normative
---
# RFC 8446

The normative specification of TLS 1.3.
`

const patchDivergesResourceStatus = `---
kind: patch
document: chen-2026-pqkex
published: 2026-04-28
title: "Post-Quantum Key Exchange in Practice"
---
# Source

## chen-2026-pqkex
` + "```yaml" + `
"@id": "chen-2026-pqkex"
"@type": source
title: "Post-Quantum Key Exchange in Practice"
authors: [Lin Chen]
published: "2026-04-28"
` + "```" + `

Surveys post-quantum key exchange deployment.

# Resource

## RFC 8446
` + "```yaml" + `
"@id": "RFC 8446"
"@type": resource
ref: standard
status: backlog
category: informative
` + "```" + `

A survey of TLS 1.3 adoption patterns.
`

const llmEntitySeed = `---
"@id": "LLM"
"@type": entity
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
kind: patch
document: karpathy-2026-agentic
published: 2026-05-01
title: "Agentic Coding Workflows"
---
# Source

## karpathy-2026-agentic
` + "```yaml" + `
"@id": "karpathy-2026-agentic"
"@type": source
title: "Agentic Coding Workflows"
authors: [Andrej Karpathy]
published: "2026-05-01"
` + "```" + `

Discusses agentic coding workflows and their effect on software development.

## Mentions
- mentions:: [[LLM]]

# Entity

## LLM
` + "```yaml" + `
"@id": "LLM"
"@type": entity
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
// "definition" (LLM's own leading prose, an "entity"'s own predicate,
// role: text) is now seeded append (FR-018) rather than firstWriteWin, so
// a near-duplicate paraphrase from a re-ingest pipeline is recognized and
// dropped (never flagged), while a genuinely new paragraph is appended —
// restoring the BUG-004 guarantee this feature's earlier per-name seed
// choice had regressed for role:text predicates.
func TestApplyEntityReContributionAppendsProseAndAccumulatesUnregisteredScalars(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedNode(t, dir, "entities/LLM.md", llmEntitySeed)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "karpathy.patch.md", karpathyPatchRegeneratesEntity)

	stdout, stderr, err := sutCaptureStderr(t, NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(stdout, err))
	it.Then(t).ShouldNot(it.String(stderr).Contain("merge conflict"))

	content := readFile(t, filepath.Join(dir, "entities", "LLM.md"))
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
// resource's firstWriteWin-declared "category" is preserved and flagged
// on genuine divergence (FR-013 conflict marker); its lastWriteWin-
// declared "status" diverges too but is never flagged (FR-012) — takes
// the newest applied value instead; its append-declared leading prose
// ("relevance", FR-018/BUG-001) diverges too but is appended, never
// flagged; commit still completes.
func TestApplyMergePreservesSetFieldOnDivergence(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedNode(t, dir, "resources/RFC 8446.md", rfcResourceSeedSetStatus)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "pqkex.patch.md", patchDivergesResourceStatus)

	before := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))
	beforeCount := len(strings.Split(before, "\n"))

	stdout, stderr, err := sutCaptureStderr(t, NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(stdout, err))

	after := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))
	afterCount := len(strings.Split(after, "\n"))
	it.Then(t).Should(it.Equal(beforeCount+1, afterCount))

	content := readFile(t, filepath.Join(dir, "resources", "RFC 8446.md"))
	it.Then(t).
		Should(it.String(content).Contain("<<<<<<< existing")).
		Should(it.String(content).Contain("normative")).
		Should(it.String(content).Contain("informative")).
		Should(it.String(content).Contain("The normative specification of TLS 1.3.")).
		Should(it.String(content).Contain("A survey of TLS 1.3 adoption patterns.")).
		Should(it.String(content).Contain("status: backlog"))

	it.Then(t).Should(it.String(stderr).Contain("RFC 8446"))
}

// arc apply pqkex.patch.md
// Scenario 4 from spec.md US2: commit stats distinguish merged from created
func TestApplyCommitStatsDistinguishMergedFromCreated(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedNode(t, dir, "entities/Transport Layer Security.md", tlsEntitySeed)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "pqkex.patch.md", pqkexPatchMergingEntity)

	out, err := sut(NewApplyCmd(), []string{patch})

	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).
		Should(it.String(out).Contain("+1 source")).
		Should(it.String(out).Contain("+0 entities")).
		Should(it.String(out).Contain("1 merged"))
}

const notePatchWithHypothesis = `---
kind: patch
document: kolesnikov-2026-note
published: 2026-05-01
title: "A Working Note"
---
# Source

## kolesnikov-2026-note
` + "```yaml" + `
"@id": "kolesnikov-2026-note"
"@type": source
title: "A Working Note"
authors: [Test Author]
published: "2026-05-01"
` + "```" + `

A short note.

# Hypothesis

## Forward Secrecy Requires Ephemeral Keys
` + "```yaml\n\"@id\": \"Forward Secrecy Requires Ephemeral Keys\"\n\"@type\": hypothesis\n```" + `

A conclusion distilled from sources.
`

// arc apply note.patch.md
// Scenario 1 from spec.md US3: a registered domain kind is applied using
// its registered behavior — no warning
func TestApplyRegisteredKindUsesRegisteredBehaviorNoWarning(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedSchemaNode(t, dir, "hypothesis", "validatedOverwrite")
	chdir(t, dir)
	patch := writePatchFile(t, dir, "note.patch.md", notePatchWithHypothesis)

	stdout, stderr, err := sutCaptureStderr(t, NewApplyCmd(), []string{patch})

	it.Then(t).ShouldNot(it.Error(stdout, err))
	it.Then(t).ShouldNot(it.String(stderr).Contain("not a recognized node type"))
	assertIsFile(t, filepath.Join(dir, "hypothesis", "Forward Secrecy Requires Ephemeral Keys.md"))
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
	it.Then(t).Should(it.String(stderr).Contain("hypothesis"))
	assertIsFile(t, filepath.Join(dir, "hypothesis", "Forward Secrecy Requires Ephemeral Keys.md"))
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

	assertIsFile(t, filepath.Join(dir, "_schema", "types", "hypothesis.md"))
	content := readFile(t, filepath.Join(dir, "_schema", "types", "hypothesis.md"))
	it.Then(t).Should(it.String(content).Contain("merge: union"))

	stat := runGit(t, dir, "show", "--stat", "HEAD")
	it.Then(t).
		Should(it.String(stat).Contain("_schema/types/hypothesis.md")).
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

	assertIsFile(t, filepath.Join(dir, "_schema", "predicates", "supersedes.md"))

	stat := runGit(t, dir, "show", "--stat", "HEAD")
	it.Then(t).
		Should(it.String(stat).Contain("_schema/predicates/supersedes.md")).
		Should(it.String(stat).Contain("sources/rescorla-2026-tls13.md"))
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

	before := readFile(t, filepath.Join(dir, "_schema", "types", "hypothesis.md"))

	secondPatch := strings.ReplaceAll(strings.ReplaceAll(notePatchWithHypothesis,
		"kolesnikov-2026-note", "kolesnikov-2026-note2"),
		"Forward Secrecy Requires Ephemeral Keys", "Handshake Latency Bound By RTT")
	patch2 := writePatchFile(t, dir, "note2.patch.md", secondPatch)
	_, err = sut(NewApplyCmd(), []string{patch2})
	it.Then(t).Should(it.Nil(err))

	after := readFile(t, filepath.Join(dir, "_schema", "types", "hypothesis.md"))
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
	it.Then(t).Should(it.String(stderr1).Contain("hypothesis"))

	seedSchemaNode(t, dir, "hypothesis", "validatedOverwrite")
	secondPatch := strings.ReplaceAll(strings.ReplaceAll(notePatchWithHypothesis,
		"kolesnikov-2026-note", "kolesnikov-2026-note2"),
		"Forward Secrecy Requires Ephemeral Keys", "Handshake Latency Bound By RTT")
	patch2 := writePatchFile(t, dir, "note2.patch.md", secondPatch)

	_, stderr2, err := sutCaptureStderr(t, NewApplyCmd(), []string{patch2})
	it.Then(t).Should(it.Nil(err))
	it.Then(t).ShouldNot(it.String(stderr2).Contain("not a recognized node type"))
}

const hypothesisSeedConfirmed = `---
"@id": "A Test Hypothesis"
"@type": hypothesis
title: A Test Hypothesis
status: confirmed
---
# A Test Hypothesis

A conclusion distilled from sources.
`

const patchDivergesHypothesisStatusTemplate = `---
kind: patch
document: %s
published: 2026-05-02
title: "%s"
---
# Source

## %s
` + "```yaml" + `
"@id": "%s"
"@type": source
title: "%s"
published: "2026-05-02"
` + "```" + `

A short note.

# Hypothesis

## A Test Hypothesis
` + "```yaml\n\"@id\": \"A Test Hypothesis\"\n\"@type\": hypothesis\nstatus: draft\n```" + `

A conclusion distilled from sources.
`

// arc apply
// spec.md FR-013/FR-015/FR-016: hand-editing a registered kind's own
// _schema/types/<kind>.md merge value has no effect on reconciliation
// (D4 — it's vestigial); hand-editing the touched PREDICATE's own
// _schema/predicates/<name>.md merge value is what changes the behavior a
// later arc apply invocation actually uses. "status" starts out
// arc-init-seeded lastWriteWin — a diverging contribution is silently
// applied (no conflict) — but after a hand-edit to firstWriteWin the
// identical shape of divergence is flagged.
func TestApplyHandEditedMergeValueChangesLaterApplyBehavior(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedSchemaNode(t, dir, "hypothesis", "union")
	seedNode(t, dir, "hypothesis/A Test Hypothesis.md", hypothesisSeedConfirmed)
	chdir(t, dir)

	lastWriteWinPatch := fmt.Sprintf(patchDivergesHypothesisStatusTemplate, "kolesnikov-2026-first", "First Note", "kolesnikov-2026-first", "kolesnikov-2026-first", "First Note")
	patch1 := writePatchFile(t, dir, "first.patch.md", lastWriteWinPatch)

	out1, err := sut(NewApplyCmd(), []string{patch1})
	it.Then(t).ShouldNot(it.Error(out1, err))

	content := readFile(t, filepath.Join(dir, "hypothesis", "A Test Hypothesis.md"))
	it.Then(t).
		ShouldNot(it.String(content).Contain("<<<<<<<")).
		Should(it.String(content).Contain("status: draft"))

	statusDoc := readFile(t, filepath.Join(dir, "_schema", "predicates", "status.md"))
	it.Then(t).Should(it.Nil(os.WriteFile(
		filepath.Join(dir, "_schema", "predicates", "status.md"),
		[]byte(strings.ReplaceAll(statusDoc, "merge: lastWriteWin", "merge: firstWriteWin")), 0o644)))

	firstWriteWinPatch := fmt.Sprintf(patchDivergesHypothesisStatusTemplate, "kolesnikov-2026-second", "Second Note", "kolesnikov-2026-second", "kolesnikov-2026-second", "Second Note")
	firstWriteWinPatch = strings.ReplaceAll(firstWriteWinPatch, "status: draft", "status: shelved")
	patch2 := writePatchFile(t, dir, "second.patch.md", firstWriteWinPatch)

	out2, err := sut(NewApplyCmd(), []string{patch2})
	it.Then(t).ShouldNot(it.Error(out2, err))

	content = readFile(t, filepath.Join(dir, "hypothesis", "A Test Hypothesis.md"))
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
kind: patch
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

	entries, rerr := os.ReadDir(filepath.Join(dir, "sources"))
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
kind: patch
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
// spec.md US3 Acceptance Scenario 4 / quickstart.md Scenario 3: an
// unrelated pre-0.5 node file sitting anywhere in the graph — not one the
// incoming patch even references — aborts the whole apply, with zero
// writes and zero commits, rather than being silently skipped.
func TestApplyUnrelatedOldFormatNodeAbortsWithZeroWrites(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	legacyNode := "---\nkind: entity\nid: old-node\ncategory: [independent]\n---\n# old-node\n\nAn entity written before this feature shipped.\n"
	seedNode(t, dir, "entities/old-node.md", legacyNode)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	before := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Error(out, err).Contain("old-node"))

	after := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))
	it.Then(t).Should(it.Equal(before, after))

	_, statErr := os.Stat(filepath.Join(dir, "sources", "rescorla-2026-tls13.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

// arc apply tls13.patch.md
// spec.md US3 Acceptance Scenario 1: a node file using the legacy "kind"
// field with no "@id"/"@type" is rejected.
func TestApplyOldFormatKindFieldRefuses(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	legacyNode := "---\nkind: entity\nid: old-node\n---\n# old-node\n\nLegacy shaped.\n"
	seedNode(t, dir, "entities/old-node.md", legacyNode)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Error(out, err).Contain("old-node"))
}

// arc apply tls13.patch.md
// spec.md US3 Acceptance Scenario 2: a node file with "@type" but missing
// "@id" is rejected, with no fallback to any other field.
func TestApplyMissingIdRefuses(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	legacyNode := "---\n\"@type\": entity\ntitle: No Id\n---\n# No Id\n\nMissing @id.\n"
	seedNode(t, dir, "entities/No Id.md", legacyNode)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Error(out, err).Contain("No Id"))
}

// arc apply tls13.patch.md
// spec.md US3 Acceptance Scenario 3: a node file whose "@id" does not
// equal its own file's basename is rejected rather than accepted.
func TestApplyIdMismatchedBasenameRefuses(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	mismatched := "---\n\"@id\": Something Else\n\"@type\": entity\n---\n# Mismatched\n\nWrong id.\n"
	seedNode(t, dir, "entities/Mismatched.md", mismatched)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Error(out, err).Contain("Mismatched"))
}

const patchWithExplicitTimelineSection = `---
kind: patch
document: foo-2026-x
published: 2026-07-12
title: "A Test Document"
---
# Source

## foo-2026-x
` + "```yaml" + `
"@id": "foo-2026-x"
"@type": source
title: "A Test Document"
published: "2026-07-12"
` + "```" + `

A test document.

# Timeline

## 2026-07
` + "```yaml\n\"@id\": \"2026-07\"\n\"@type\": timeline\ngranularity: monthly\n```" + `
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
	entityDoc := readFile(t, filepath.Join(dir, "_schema", "types", "entity.md"))
	it.Then(t).Should(it.Nil(os.WriteFile(
		filepath.Join(dir, "_schema", "types", "entity.md"),
		[]byte(strings.ReplaceAll(entityDoc, "merge: union", "merge: bogus")), 0o644)))
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	before := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Error(out, err).Contain("_schema/types/entity.md"))

	after := strings.TrimSpace(runGit(t, dir, "log", "--oneline"))
	it.Then(t).Should(it.Equal(before, after))

	_, statErr := os.Stat(filepath.Join(dir, "sources", "rescorla-2026-tls13.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

// arc apply tls13.patch.md
// spec.md US2 Acceptance Scenario 5 / FR-014: a missing _schema/ subfolder
// aborts arc apply before any other write.
func TestApplyMissingSchemaFolderAbortsWithZeroWrites(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	it.Then(t).Should(it.Nil(os.RemoveAll(filepath.Join(dir, "_schema", "types"))))
	chdir(t, dir)
	patch := writePatchFile(t, dir, "tls13.patch.md", tls13Patch)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Error(out, err).Contain("_schema/types"))

	_, statErr := os.Stat(filepath.Join(dir, "sources", "rescorla-2026-tls13.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

// arc apply pqkex.patch.md
// Edge case: a merge conflict marker is written while the commit still
// completes (FR-013); PostRunE prints a hint naming the conflicted file
func TestApplyConflictHintPrinted(t *testing.T) {
	dir := t.TempDir()
	initGraph(t, dir)
	seedNode(t, dir, "resources/RFC 8446.md", rfcResourceSeedSetStatus)
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
kind: patch
document: acme-2026-deploy1
published: 2026-06-01
title: "First Deploy"
---
# Source

## acme-2026-deploy1
` + "```yaml" + `
"@id": "acme-2026-deploy1"
"@type": source
title: "First Deploy"
published: "2026-06-01"
` + "```" + `

A deployment record.

# LogEntry

## Deploy Event
` + "```yaml\n\"@id\": \"Deploy Event\"\n\"@type\": logentry\n```" + `

An event log.
- relatesTo:: [[Service A]]
`

const deployEvent2Patch = `---
kind: patch
document: acme-2026-deploy2
published: 2026-06-02
title: "Second Deploy"
---
# Source

## acme-2026-deploy2
` + "```yaml" + `
"@id": "acme-2026-deploy2"
"@type": source
title: "Second Deploy"
published: "2026-06-02"
` + "```" + `

Another deployment record.

# LogEntry

## Deploy Event
` + "```yaml\n\"@id\": \"Deploy Event\"\n\"@type\": logentry\n```" + `

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
	seedSchemaNode(t, dir, "logentry", "append")
	chdir(t, dir)
	patch1 := writePatchFile(t, dir, "deploy1.patch.md", deployEvent1Patch)
	patch2 := writePatchFile(t, dir, "deploy2.patch.md", deployEvent2Patch)

	out1, err := sut(NewApplyCmd(), []string{patch1})
	it.Then(t).ShouldNot(it.Error(out1, err))

	out2, err := sut(NewApplyCmd(), []string{patch2})
	it.Then(t).ShouldNot(it.Error(out2, err))
	it.Then(t).
		Should(it.String(out2).Contain("+0 logentrys")).
		Should(it.String(out2).Contain("1 merged"))

	entries, rerr := os.ReadDir(filepath.Join(dir, "logentrys"))
	it.Then(t).
		Should(it.Nil(rerr)).
		Should(it.Equal(1, len(entries)))

	content := readFile(t, filepath.Join(dir, "logentrys", "Deploy Event.md"))
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
kind: patch
document: rescorla-2026-tls13
published: 2026-04-12
title: "TLS 1.3: Design and Rationale"
---
# Source

## rescorla-2026-tls13
` + "```yaml" + `
"@id": "rescorla-2026-tls13"
"@type": source
title: "TLS 1.3: Design and Rationale"
authors: [Eric Rescorla]
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
"@type": entity
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

	source := readFile(t, filepath.Join(dir, "sources", "rescorla-2026-tls13.md"))
	entity := readFile(t, filepath.Join(dir, "entities", "Transport Layer Security.md"))
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

	source := readFile(t, filepath.Join(dir, "sources", "rescorla-2026-tls13.md"))
	entity := readFile(t, filepath.Join(dir, "entities", "Transport Layer Security.md"))

	sourceIndexed := extractQuotedAttr(source, "indexed")
	entityIndexed := extractQuotedAttr(entity, "indexed")
	it.Then(t).
		ShouldNot(it.Equal("", sourceIndexed)).
		Should(it.Equal(sourceIndexed, entityIndexed))
}

const stubEntityPatch = `---
kind: patch
document: foo-2026-stub
published: 2026-04-12
title: "Stub Test Document"
---
# Source

## foo-2026-stub
` + "```yaml" + `
"@id": "foo-2026-stub"
"@type": source
title: "Stub Test Document"
published: "2026-04-12"
` + "```" + `

A stub test document.

# Entity

## StubEntity
` + "```yaml\n\"@id\": \"StubEntity\"\n\"@type\": entity\n```" + `
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

	content := readFile(t, filepath.Join(dir, "entities", "StubEntity.md"))
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

	schemaDoc := readFile(t, filepath.Join(dir, "_schema", "types", "hypothesis.md"))
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
	seedNode(t, dir, "entities/Transport Layer Security.md", tlsEntitySeed)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "pqkex.patch.md", pqkexPatchMergingEntity)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	source := readFile(t, filepath.Join(dir, "sources", "chen-2026-pqkex.md"))
	entity := readFile(t, filepath.Join(dir, "entities", "Transport Layer Security.md"))

	sourceIndexed := extractQuotedAttr(source, "indexed")
	entityUpdated := extractQuotedAttr(entity, "updated")
	it.Then(t).
		ShouldNot(it.Equal("", sourceIndexed)).
		Should(it.Equal(sourceIndexed, entityUpdated))
}

const memoNoneSeed = `---
"@id": Widget
"@type": memo
title: Widget
---
# Widget

Original text.
`

const memoNonePatch = `---
kind: patch
document: foo-2026-memo
published: 2026-05-01
title: "Memo Patch"
---
# Source

## foo-2026-memo
` + "```yaml" + `
"@id": "foo-2026-memo"
"@type": source
title: "Memo Patch"
published: "2026-05-01"
` + "```" + `

A memo patch.

# Memo

## Widget
` + "```yaml\n\"@id\": \"Widget\"\n\"@type\": memo\n```" + `

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
	seedSchemaNode(t, dir, "memo", "immutable")
	seedNode(t, dir, "memos/Widget.md", memoNoneSeed)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "memo.patch.md", memoNonePatch)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	content := readFile(t, filepath.Join(dir, "memos", "Widget.md"))
	it.Then(t).
		Should(it.String(content).Contain("Original text.")).
		Should(it.String(content).Contain("Changed text.")).
		Should(it.String(content).Contain("updated:"))
}

const stubbedThingSeed = `---
"@id": "StubbedThing"
"@type": entity
---
# StubbedThing
`

const patchFillsStubWithRealContent = `---
kind: patch
document: foo-2026-fill
published: 2026-06-01
title: "Fill Patch"
---
# Source

## foo-2026-fill
` + "```yaml" + `
"@id": "foo-2026-fill"
"@type": source
title: "Fill Patch"
published: "2026-06-01"
` + "```" + `

A fill patch.

# Entity

## StubbedThing
` + "```yaml\n\"@id\": \"StubbedThing\"\n\"@type\": entity\npublished: \"2026-05-02\"\n```" + `

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
	seedNode(t, dir, "entities/StubbedThing.md", stubbedThingSeed)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "fill.patch.md", patchFillsStubWithRealContent)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	content := readFile(t, filepath.Join(dir, "entities", "StubbedThing.md"))
	it.Then(t).
		Should(it.String(content).Contain(`published: "2026-05-02"`)).
		ShouldNot(it.String(content).Contain("indexed:"))

	updated := extractQuotedAttr(content, "updated")
	it.Then(t).ShouldNot(it.Equal("", updated))
}

const noOpUnionEntitySeed = `---
"@id": "Widget"
"@type": entity
title: Widget
category: [independent, abstract, occurrent, script]
---
# Widget

A test entity.
- replaces:: [[Old Widget]]
`

const noOpUnionPatch = `---
kind: patch
document: foo-2026-noop
published: 2026-05-03
title: "No-op Patch"
---
# Source

## foo-2026-noop
` + "```yaml" + `
"@id": "foo-2026-noop"
"@type": source
title: "No-op Patch"
published: "2026-05-03"
` + "```" + `

A no-op patch.

# Entity

## Widget
` + "```yaml" + `
"@id": "Widget"
"@type": entity
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
	seedNode(t, dir, "entities/Widget.md", noOpUnionEntitySeed)
	chdir(t, dir)
	patch := writePatchFile(t, dir, "noop.patch.md", noOpUnionPatch)

	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	content := readFile(t, filepath.Join(dir, "entities", "Widget.md"))
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

	source := readFile(t, filepath.Join(dir, "sources", "rescorla-2026-tls13.md"))
	it.Then(t).
		Should(it.String(source).Contain("mentions:: [[Transport Layer Security]]")).
		Should(it.String(source).Contain("cites:: [[RFC 8446]]"))

	entity := readFile(t, filepath.Join(dir, "entities", "Transport Layer Security.md"))
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
kind: patch
document: dmitry-2026-graph
published: 2026-07-03
title: "Ontologies, Graph Structures, and LLM-Based Knowledge Management"
---
# Source

## dmitry-2026-graph
` + "```yaml" + `
authors:
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

	assertIsFile(t, filepath.Join(dir, "sources", "dmitry-2026-graph.md"))
	assertIsFile(t, filepath.Join(dir, "entities", "LLM.md"))

	entity := readFile(t, filepath.Join(dir, "entities", "LLM.md"))
	it.Then(t).
		Should(it.String(entity).Contain(`"@id": LLM`)).
		Should(it.String(entity).Contain(`"@type": entity`))
}

// --- spec 012: Per-Predicate Merge Reconciliation for arc apply ---
//
// The fixtures below use arc init's own real seeded schema
// (appschema.Seed(), via initGraph) so every predicate's declared merge
// behavior is exactly what a real graph would use: ref/immutable,
// status/lastWriteWin, tags/union, category/firstWriteWin, url/
// fillIfEmpty (internal/app/schema/kernel/schema.go). "category" (role:
// meta) stands in for a firstWriteWin exemplar here rather than
// "abstract" (role: text), since BUG-001/FR-018 repoints every role:text
// predicate — abstract included — to append.

const resourcePatch012First = `---
kind: patch
document: doc-2026-a
published: 2026-07-01
title: "Doc A"
---
# Source

## doc-2026-a
` + "```yaml" + `
"@id": "doc-2026-a"
"@type": source
title: "Doc A"
published: "2026-07-01"
` + "```" + `

A source document.

# Resource

## example-book
` + "```yaml" + `
"@id": "example-book"
"@type": resource
ref: book
status: backlog
tags: [ai]
category: "First summary."
` + "```" + `

A tracked reading item.
`

const resourcePatch012Second = `---
kind: patch
document: doc-2026-b
published: 2026-07-02
title: "Doc B"
---
# Source

## doc-2026-b
` + "```yaml" + `
"@id": "doc-2026-b"
"@type": source
title: "Doc B"
published: "2026-07-02"
` + "```" + `

Another source document.

# Resource

## example-book
` + "```yaml" + `
"@id": "example-book"
"@type": resource
ref: article
status: read
tags: [ml]
category: "A different summary."
` + "```" + `

A tracked reading item.
`

// arc apply doc-a.patch.md, then doc-b.patch.md
// spec 012 User Story 1, Acceptance Scenarios 1-4: a single second
// contribution touching three predicates at once resolves each by its own
// rule within that one application — ref (immutable) rejects the
// divergence, status (lastWriteWin) takes the newest value, tags (union)
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

	content := readFile(t, filepath.Join(dir, "resources", "example-book.md"))
	it.Then(t).
		Should(it.String(content).Contain("ref: book")).
		Should(it.String(content).Contain("status: read")).
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
		Should(it.String(stderr).Contain("ref: immutable -> unchanged")).
		Should(it.String(stderr).Contain("status: lastWriteWin -> overwritten")).
		Should(it.String(stderr).Contain("tags: union -> appended"))
}

// spec 012 User Story 2, Acceptance Scenario 1/3: within the same
// combined application, category (firstWriteWin) is flagged for human
// review on genuine divergence, but tags (union) and status
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

	content := readFile(t, filepath.Join(dir, "resources", "example-book.md"))
	it.Then(t).
		Should(it.String(content).Contain("<<<<<<< existing")).
		Should(it.String(content).Contain("First summary.")).
		Should(it.String(content).Contain("A different summary."))

	// tags/status diverge too but must never be wrapped in a marker
	tagsAndStatusLines := []string{"status: read", "- ai", "- ml"}
	for _, line := range tagsAndStatusLines {
		it.Then(t).Should(it.String(content).Contain(line))
	}
}

const resourceFillIfEmptySeed = `---
"@id": "RFC 9999"
"@type": resource
title: RFC 9999
ref: standard
url: ""
---
# RFC 9999

A draft specification.
`

const patchFillsUrlFirstTime = `---
kind: patch
document: doc-2026-c
published: 2026-07-03
title: "Doc C"
---
# Source

## doc-2026-c
` + "```yaml" + `
"@id": "doc-2026-c"
"@type": source
title: "Doc C"
published: "2026-07-03"
` + "```" + `

A source document.

# Resource

## RFC 9999
` + "```yaml" + `
"@id": "RFC 9999"
"@type": resource
ref: standard
url: https://example.org/rfc9999-v1
` + "```" + `

A draft specification.
`

const patchDivergesUrlAfterFirstWrite = `---
kind: patch
document: doc-2026-d
published: 2026-07-04
title: "Doc D"
---
# Source

## doc-2026-d
` + "```yaml" + `
"@id": "doc-2026-d"
"@type": source
title: "Doc D"
published: "2026-07-04"
` + "```" + `

Another source document.

# Resource

## RFC 9999
` + "```yaml" + `
"@id": "RFC 9999"
"@type": resource
ref: standard
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
	seedNode(t, dir, "resources/RFC 9999.md", resourceFillIfEmptySeed)
	chdir(t, dir)
	patch1 := writePatchFile(t, dir, "doc-c.patch.md", patchFillsUrlFirstTime)

	_, stderr1, err := sutCaptureStderr(t, NewApplyCmd(), []string{patch1})
	it.Then(t).Should(it.Nil(err))
	it.Then(t).ShouldNot(it.String(stderr1).Contain("merge conflict"))

	content := readFile(t, filepath.Join(dir, "resources", "RFC 9999.md"))
	it.Then(t).Should(it.String(content).Contain("https://example.org/rfc9999-v1"))

	patch2 := writePatchFile(t, dir, "doc-d.patch.md", patchDivergesUrlAfterFirstWrite)
	_, stderr2, err := sutCaptureStderr(t, NewApplyCmd(), []string{patch2})
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.String(stderr2).Contain("merge conflict"))

	content = readFile(t, filepath.Join(dir, "resources", "RFC 9999.md"))
	it.Then(t).
		Should(it.String(content).Contain("<<<<<<< existing")).
		Should(it.String(content).Contain("rfc9999-v1")).
		Should(it.String(content).Contain("rfc9999-v2"))
}

const resourceIndependentPredicatesSeed = `---
"@id": "example-topic"
"@type": resource
title: example-topic
ref: topic
---
# example-topic

An ongoing research topic.
`

const patchContributesTags = `---
kind: patch
document: doc-2026-e
published: 2026-07-05
title: "Doc E"
---
# Source

## doc-2026-e
` + "```yaml" + `
"@id": "doc-2026-e"
"@type": source
title: "Doc E"
published: "2026-07-05"
` + "```" + `

A source document.

# Resource

## example-topic
` + "```yaml" + `
"@id": "example-topic"
"@type": resource
ref: topic
tags: [ai]
` + "```" + `

An ongoing research topic.
`

const patchContributesStatus = `---
kind: patch
document: doc-2026-f
published: 2026-07-06
title: "Doc F"
---
# Source

## doc-2026-f
` + "```yaml" + `
"@id": "doc-2026-f"
"@type": source
title: "Doc F"
published: "2026-07-06"
` + "```" + `

Another source document.

# Resource

## example-topic
` + "```yaml" + `
"@id": "example-topic"
"@type": resource
ref: topic
status: read
` + "```" + `

An ongoing research topic.
`

// spec 012 User Story 3, Acceptance Scenario 2: two patches contributing
// to independent predicates (tags/status) converge on an identical result
// regardless of which order they're applied in.
func TestApply012US3IndependentPredicatesConvergeInEitherOrder(t *testing.T) {
	forward := t.TempDir()
	initGraph(t, forward)
	seedNode(t, forward, "resources/example-topic.md", resourceIndependentPredicatesSeed)

	reverse := t.TempDir()
	initGraph(t, reverse)
	seedNode(t, reverse, "resources/example-topic.md", resourceIndependentPredicatesSeed)

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

	forwardContent := readFile(t, filepath.Join(forward, "resources", "example-topic.md"))
	reverseContent := readFile(t, filepath.Join(reverse, "resources", "example-topic.md"))

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
	seedNode(t, forward, "resources/example-book.md", `---
"@id": "example-book"
"@type": resource
title: example-book
ref: book
---
# example-book

A tracked reading item.
`)

	reverse := t.TempDir()
	initGraph(t, reverse)
	seedNode(t, reverse, "resources/example-book.md", `---
"@id": "example-book"
"@type": resource
title: example-book
ref: book
---
# example-book

A tracked reading item.
`)

	readPatch := `---
kind: patch
document: doc-2026-read
published: 2026-07-07
title: "Doc Read"
---
# Source

## doc-2026-read
` + "```yaml" + `
"@id": "doc-2026-read"
"@type": source
title: "Doc Read"
published: "2026-07-07"
` + "```" + `

A source document.

# Resource

## example-book
` + "```yaml" + `
"@id": "example-book"
"@type": resource
ref: book
status: read
` + "```" + `

A tracked reading item.
`
	archivedPatch := `---
kind: patch
document: doc-2026-archived
published: 2026-07-08
title: "Doc Archived"
---
# Source

## doc-2026-archived
` + "```yaml" + `
"@id": "doc-2026-archived"
"@type": source
title: "Doc Archived"
published: "2026-07-08"
` + "```" + `

Another source document.

# Resource

## example-book
` + "```yaml" + `
"@id": "example-book"
"@type": resource
ref: book
status: archived
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

	forwardContent := readFile(t, filepath.Join(forward, "resources", "example-book.md"))
	reverseContent := readFile(t, filepath.Join(reverse, "resources", "example-book.md"))

	it.Then(t).
		Should(it.String(forwardContent).Contain("status: archived")).
		Should(it.String(reverseContent).Contain("status: read"))
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

	firstConflict := readFile(t, filepath.Join(dir, "resources", "example-book.md"))
	it.Then(t).Should(it.Equal(1, strings.Count(firstConflict, "<<<<<<<")))

	replayPatch := strings.ReplaceAll(resourcePatch012Second, "doc-2026-b", "doc-2026-b-replay")
	patch3 := writePatchFile(t, dir, "doc-b-replay.patch.md", replayPatch)
	_, err = sut(NewApplyCmd(), []string{patch3})
	it.Then(t).Should(it.Nil(err))

	secondConflict := readFile(t, filepath.Join(dir, "resources", "example-book.md"))
	it.Then(t).
		Should(it.Equal(1, strings.Count(secondConflict, "<<<<<<<"))).
		Should(it.Equal(firstConflict, secondConflict))
}
