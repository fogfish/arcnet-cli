//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package ctrl

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/spf13/cobra"

	"github.com/fogfish/arcnet-cli/cmd/arc/lint"
	appschema "github.com/fogfish/arcnet-cli/internal/app/schema"
)

// ---------------------------------------------------------------------------
// specs/023-core-vocabulary-conformance — User Story 5
//
// An existing graph adopts the corrected vocabulary in one deliberate,
// reviewable step: arc upgrade replaces the built-in documents outright,
// in exactly one commit, leaving content untouched.
//
// Every scenario runs against testdata/legacy-graph — a byte-exact,
// hand-preserved snapshot of what a PREVIOUS release seeded, including
// merge: validatedOverwrite on scoreZ/scoreC. Asserting against a
// freshly-seeded graph instead would compare the current vocabulary with
// itself and pass vacuously.
// ---------------------------------------------------------------------------

// legacyFixtureRoot is resolved at package-variable initialization time —
// before any test calls chdir(t, ...) into a temporary graph, after which a
// relative path no longer resolves.
var legacyFixtureRoot = func() string {
	abs, err := filepath.Abs(filepath.Join("testdata", "legacy-graph"))
	if err != nil {
		panic(err)
	}
	return abs
}()

const legacySourceNode = `---
"@id": rescorla-2026-tls13
"@type": Source
title: "TLS 1.3: Design and Rationale"
published: "2026-04-12"
created: "2026-04-12"
---
# rescorla-2026-tls13

A design retrospective on the TLS 1.3 handshake.

## Mentions
- mentions:: [[Transport Layer Security]]
`

// legacyDriftedSource carries an abstract of TWO paragraphs — the shape
// only the previous append behaviour could have produced, and the one
// arc upgrade's prose-drift scan reports without repairing (research.md
// D12, contract C3.7).
const legacyDriftedSource = `---
"@id": chen-2026-pqkex
"@type": Source
title: "Post-Quantum Key Exchange in Practice"
published: "2026-04-28"
created: "2026-04-28"
---
# chen-2026-pqkex

A survey of post-quantum key exchange deployments observed in the wild.

Why hybrid key exchange was chosen, revisited some years after the first deployments shipped.

## Mentions
- mentions:: [[Transport Layer Security]]
`

const legacyEntityNode = `---
"@id": Transport Layer Security
"@type": Entity
category: [independent, abstract, occurrent, script]
published: "2026-04-12"
created: "2026-04-12"
---
# Transport Layer Security

A cryptographic protocol that establishes an authenticated channel.

## MentionedIn
- mentionedIn:: [[rescorla-2026-tls13]]
`

// buildLegacyGraph materializes a graph seeded by a PREVIOUS release: the
// canonical folders, .arc/, .gitignore, the legacy _schema/ snapshot, and
// three content nodes — all under one git commit.
func buildLegacyGraph(t *testing.T, dir string) {
	t.Helper()

	for _, folder := range []string{"Source", "Entity", "Resource", "Reference", "timeline/yearly", "timeline/monthly", "_schema/Class", "_schema/Property"} {
		it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Join(dir, filepath.FromSlash(folder)), 0o755)))
	}
	it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Join(dir, ".arc"), 0o755)))
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(dir, ".arc", ".gitkeep"), nil, 0o644)))
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".arc/\n"), 0o644)))

	copyLegacySchema(t, dir)

	writeGraphFile(t, dir, "Source/rescorla-2026-tls13.md", legacySourceNode)
	writeGraphFile(t, dir, "Source/chen-2026-pqkex.md", legacyDriftedSource)
	writeGraphFile(t, dir, "Entity/Transport Layer Security.md", legacyEntityNode)

	runGit(t, dir, "init")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "graph(init): empty knowledge graph")
}

func copyLegacySchema(t *testing.T, dir string) {
	t.Helper()
	err := filepath.WalkDir(legacyFixtureRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, rerr := filepath.Rel(legacyFixtureRoot, path)
		if rerr != nil {
			return rerr
		}
		if rel == "README.md" {
			return nil
		}
		raw, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		writeGraphFile(t, dir, filepath.ToSlash(rel), string(raw))
		return nil
	})
	it.Then(t).Should(it.Nil(err))
}

func writeGraphFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(relPath))
	it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Dir(full), 0o755)))
	it.Then(t).Should(it.Nil(os.WriteFile(full, []byte(content), 0o644)))
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	return gitOutput(t, dir, args...)
}

// schemaTree reads every _schema/ document in dir, keyed exactly as
// schema.Seed() keys its own output.
func schemaTree(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	root := filepath.Join(dir, "_schema")
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		rel, rerr := filepath.Rel(dir, path)
		if rerr != nil {
			return rerr
		}
		out[filepath.ToSlash(rel)] = string(raw)
		return nil
	})
	it.Then(t).Should(it.Nil(err))
	return out
}

func sortedMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func commitSubjects(t *testing.T, dir string) []string {
	t.Helper()
	raw := strings.TrimSpace(gitOutput(t, dir, "log", "--format=%s"))
	if raw == "" {
		return nil
	}
	return strings.Split(raw, "\n")
}

// arc upgrade
// Scenario 1 from spec.md US5 (023): every seeded definition afterwards
// matches the corresponding definition in a freshly initialized graph
// (FR-017, FR-018, SC-006).
func TestUpgradeReplacesBuiltInVocabularyWithCurrentSeed(t *testing.T) {
	dir := t.TempDir()
	buildLegacyGraph(t, dir)
	chdir(t, dir)

	_, err := sut(NewUpgradeCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	got := schemaTree(t, dir)
	want := map[string]string{}
	for path, raw := range appschema.Seed() {
		want[path] = string(raw)
	}

	it.Then(t).Should(it.Seq(sortedMapKeys(got)).Equal(sortedMapKeys(want)...))
	for path, expected := range want {
		it.Then(t).Should(it.Equal(expected, got[path]))
	}
}

// arc upgrade
// Scenario 2 from spec.md US5 (023): no content node outside the
// vocabulary folders is modified (FR-019, SC-006).
func TestUpgradeLeavesContentNodesByteIdentical(t *testing.T) {
	dir := t.TempDir()
	buildLegacyGraph(t, dir)
	chdir(t, dir)

	contentPaths := []string{
		"Source/rescorla-2026-tls13.md",
		"Source/chen-2026-pqkex.md",
		"Entity/Transport Layer Security.md",
	}
	before := map[string]string{}
	for _, p := range contentPaths {
		before[p] = readGraphFile(t, filepath.Join(dir, filepath.FromSlash(p)))
	}

	_, err := sut(NewUpgradeCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	for _, p := range contentPaths {
		it.Then(t).Should(it.Equal(before[p], readGraphFile(t, filepath.Join(dir, filepath.FromSlash(p)))))
	}
}

// arc upgrade
// Scenario 3 from spec.md US5 (023): the change is recorded as exactly one
// commit whose message identifies it as a vocabulary migration (FR-021;
// contract C3.6, CORE §13.3).
func TestUpgradeRecordsExactlyOneMigrationCommit(t *testing.T) {
	dir := t.TempDir()
	buildLegacyGraph(t, dir)
	chdir(t, dir)

	before := commitSubjects(t, dir)

	_, err := sut(NewUpgradeCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	after := commitSubjects(t, dir)

	it.Then(t).
		Should(it.Equal(len(before)+1, len(after))).
		Should(it.String(after[0]).Contain("graph(migrate):"))
}

// arc upgrade
// Scenario 4 from spec.md US5 (023): an author's own definitions are
// preserved and only the built-in ones are replaced (FR-020; contract C3.3
// row 3).
func TestUpgradePreservesAuthorExtendedVocabulary(t *testing.T) {
	dir := t.TempDir()
	buildLegacyGraph(t, dir)

	const authored = `---
"@id": aporia
"@type": Property
merge: append
role: text
---
# aporia

A predicate this graph's author added; not part of the built-in vocabulary.
`
	writeGraphFile(t, dir, "_schema/Property/aporia.md", authored)
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "schema: register aporia")
	chdir(t, dir)

	_, err := sut(NewUpgradeCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	it.Then(t).Should(it.Equal(authored, readGraphFile(t, filepath.Join(dir, "_schema", "Property", "aporia.md"))))
}

// arc upgrade (twice)
// Scenario 5 from spec.md US5 (023): a graph that has already adopted the
// corrected vocabulary changes nothing and creates no commit on a second
// run (FR-021, FR-024; contract C3.4).
func TestUpgradeSecondRunIsANoOp(t *testing.T) {
	dir := t.TempDir()
	buildLegacyGraph(t, dir)
	chdir(t, dir)

	_, err := sut(NewUpgradeCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	before := commitSubjects(t, dir)
	schemaBefore := schemaTree(t, dir)

	out, err := sut(NewUpgradeCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	it.Then(t).
		Should(it.Seq(commitSubjects(t, dir)).Equal(before...)).
		Should(it.String(out).Contain("already up to date"))

	after := schemaTree(t, dir)
	for path, content := range schemaBefore {
		it.Then(t).Should(it.Equal(content, after[path]))
	}
}

// arc upgrade
// Scenario 6 from spec.md US5 (023): documents whose summaries were
// already duplicated by the previous appending behaviour are listed for
// manual review, and none is silently rewritten (FR-023; contract C3.7).
func TestUpgradeReportsProseDriftWithoutRepairingIt(t *testing.T) {
	dir := t.TempDir()
	buildLegacyGraph(t, dir)
	chdir(t, dir)

	driftedPath := filepath.Join(dir, "Source", "chen-2026-pqkex.md")
	before := readGraphFile(t, driftedPath)

	out, err := sut(NewUpgradeCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	it.Then(t).
		Should(it.String(out).Contain("chen-2026-pqkex")).
		Should(it.Equal(before, readGraphFile(t, driftedPath)))
}

// arc upgrade
// Scenario 7 from spec.md US5 (023): the step completes on a graph whose
// vocabulary declares the retired seventh merge value — it is not blocked
// by the very declaration it exists to remove (FR-022; contract C3.2,
// research.md D10).
//
// This is the assertion that keeps the migration deadlock closed: steps
// 1-4 never decode a Property document, so merge: validatedOverwrite on
// disk cannot block the command that removes it.
func TestUpgradeSucceedsOnGraphDeclaringRetiredMergeValue(t *testing.T) {
	dir := t.TempDir()
	buildLegacyGraph(t, dir)
	chdir(t, dir)

	legacyScoreZ := readGraphFile(t, filepath.Join(dir, "_schema", "Property", "scoreZ.md"))
	it.Then(t).Should(it.String(legacyScoreZ).Contain("validatedOverwrite"))

	_, err := sut(NewUpgradeCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	upgraded := readGraphFile(t, filepath.Join(dir, "_schema", "Property", "scoreZ.md"))
	it.Then(t).ShouldNot(it.String(upgraded).Contain("validatedOverwrite"))
}

// arc lint on an un-upgraded graph, then arc upgrade, then arc lint again
// Scenario 4 from spec.md US2 (023), end to end: closing the merge menu at
// six makes every previously-seeded graph fail to load, and the error that
// reports it must carry the way out (FR-002, contract C1.2).
//
// This is the single most consequential user-visible moment in the whole
// feature — the first thing an existing user sees after upgrading the
// binary — so it is asserted as the round trip it actually is: break,
// remedy, recover.
func TestUpgradeIsTheRemedyNamedByTheRejection(t *testing.T) {
	dir := t.TempDir()
	buildLegacyGraph(t, dir)
	chdir(t, dir)

	_, lintErr := sut(lint.NewLintCmd(), nil)
	if lintErr == nil {
		t.Fatal("a graph declaring the retired seventh merge value must fail to load")
	}

	// Both scoreZ.md and scoreC.md declare the retired value, and Resolve
	// fails on whichever it reads first — either is a correct report, so
	// the assertion names the folder rather than one of the two files.
	message := lintErr.Error()
	it.Then(t).
		Should(it.String(message).Contain("_schema/Property/score")).
		Should(it.String(message).Contain("validatedOverwrite")).
		Should(it.String(message).Contain("arc upgrade"))
	for _, op := range conformantMergeOps {
		it.Then(t).Should(it.String(message).Contain(op))
	}

	_, err := sut(NewUpgradeCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	// The graph loads again. It may still report ordinary content
	// violations — this fixture's nodes were never ingested through arc
	// apply — but the schema no longer refuses to load at all, which is
	// the whole point of the remedy.
	_, afterErr := sut(lint.NewLintCmd(), nil)
	if afterErr != nil {
		it.Then(t).ShouldNot(it.String(afterErr.Error()).Contain("declares merge"))
	}
}

// dryRunUpgradeCmd sets --dry-run on the command directly. sut() invokes
// RunE without Cobra's flag parsing, so passing "--dry-run" as an argument
// would be silently ignored — the same reason revert_test.go's
// forcedRevertCmd exists.
func dryRunUpgradeCmd(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := NewUpgradeCmd()
	it.Then(t).Should(it.Nil(cmd.Flags().Set("dry-run", "true")))
	return cmd
}

// arc upgrade --dry-run
// Contract C3.5: --dry-run reports exactly what a real run would find, and
// writes nothing.
//
// The drift half of that is not incidental. An UN-UPGRADED graph cannot be
// resolved at all — its own scoreZ.md declares the retired seventh merge
// value — so a scan that only ever read the on-disk vocabulary would find
// no firstWriteWin predicates, report no candidates, and then the real run
// would report several. Which is exactly the surprise --dry-run exists to
// prevent, and exactly what running the quickstart against a real binary
// caught.
func TestUpgradeDryRunReportsSameDriftAsRealRun(t *testing.T) {
	dir := t.TempDir()
	buildLegacyGraph(t, dir)
	chdir(t, dir)

	before := commitSubjects(t, dir)
	schemaBefore := schemaTree(t, dir)

	dry, err := sut(dryRunUpgradeCmd(t), []string{})
	it.Then(t).Should(it.Nil(err))

	it.Then(t).
		Should(it.String(dry).Contain("dry run")).
		Should(it.String(dry).Contain("chen-2026-pqkex"))

	// Nothing written, nothing committed.
	it.Then(t).Should(it.Seq(commitSubjects(t, dir)).Equal(before...))
	for path, content := range schemaBefore {
		it.Then(t).Should(it.Equal(content, schemaTree(t, dir)[path]))
	}

	real, err := sut(NewUpgradeCmd(), []string{})
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.String(real).Contain("chen-2026-pqkex"))
}
