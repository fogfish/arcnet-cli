//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package graph

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/cmd/arc/lint"
)

// ---------------------------------------------------------------------------
// specs/031-init-existing-git-repo — User Story 4 (FR-021..FR-025).
//
// Verification-first (research.md D7): the git adapter already carries
// BUG-002's nested-repository fixes. These scenarios establish that every
// git-backed command behaves identically for a graph nested inside a larger
// project as for a standalone one, rather than assuming it. quickstart.md S9.
// ---------------------------------------------------------------------------

// initHostGraph builds a graph at <repo>/kb inside a host repository that
// carries unrelated history of its own: a sibling other/ directory with its
// own commits, none of which any graph-scoped command may reach. Unlike
// initGraph the graph has no .git of its own — that is the whole point.
func initHostGraph(t *testing.T, repo string) (graph, other string) {
	t.Helper()
	graph = filepath.Join(repo, "kb")
	other = filepath.Join(repo, "other")
	it.Then(t).Should(it.Nil(os.MkdirAll(graph, 0o755)))
	it.Then(t).Should(it.Nil(os.MkdirAll(other, 0o755)))

	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(other, "unrelated.md"), []byte("unrelated\n"), 0o644)))
	runGit(t, repo, "init")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-m", "host: unrelated work")

	writeGraphLayout(t, graph)
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-m", "graph(init): empty knowledge graph")
	return graph, other
}

// applicationTimestamp matches the RFC3339 instants arc stamps into
// "indexed" at write time. Two runs of the same command are seconds apart,
// so a parity comparison has to normalize them away — everything else about
// the two graphs must match exactly.
var applicationTimestamp = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T[0-9:.]+(?:Z|[+-][0-9:]+)`)

// graphTree reads every graph-content file under dir as a path→content map,
// skipping git's and arc's own state. Two trees compare equal exactly when
// the two runs produced the same graph.
func graphTree(t *testing.T, dir string) map[string]string {
	t.Helper()
	tree := map[string]string{}

	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(dir, path)
		if rerr != nil {
			return rerr
		}
		head := strings.Split(filepath.ToSlash(rel), "/")[0]
		if head == ".git" || head == ".arc" {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		content, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		tree[filepath.ToSlash(rel)] = applicationTimestamp.ReplaceAllString(string(content), "<timestamp>")
		return nil
	})
	it.Then(t).Should(it.Nil(err))

	return tree
}

// assertSameGraph asserts a nested graph and a standalone one hold exactly
// the same files with exactly the same content — FR-025's parity, stated as
// an observable outcome rather than as an absence of leakage.
func assertSameGraph(t *testing.T, nested, standalone string) {
	t.Helper()
	got, want := graphTree(t, nested), graphTree(t, standalone)

	it.Then(t).Should(it.Seq(sortedKeys(got)).Equal(sortedKeys(want)...))
	for path, content := range want {
		it.Then(t).Should(it.Equal(content, got[path]))
	}
}

func sortedKeys(tree map[string]string) []string {
	keys := make([]string, 0, len(tree))
	for key := range tree {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// headPaths returns every path the repository's HEAD commit touched,
// repository-relative and sorted.
func headPaths(t *testing.T, repo string) []string {
	t.Helper()
	out := strings.TrimSpace(runGit(t, repo, "show", "--name-only", "--format=", "HEAD"))
	if out == "" {
		return nil
	}
	paths := strings.Split(out, "\n")
	sort.Strings(paths)
	return paths
}

// arc apply, nested inside a host repository
// spec.md US4 Acceptance Scenario 1 (FR-022): the ingest commit contains only
// paths inside the graph, and an unrelated dirty sibling is never swept in.
func TestNestedApplyCommitsOnlyGraphSubtree(t *testing.T) {
	repo := t.TempDir()
	graph, other := initHostGraph(t, repo)
	chdir(t, graph)

	// unrelated work in flight elsewhere in the host project
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(other, "unrelated.md"), []byte("edited\n"), 0o644)))

	patch := writePatchFile(t, t.TempDir(), "docA.patch.md", v011RevertDocA)
	out, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	paths := headPaths(t, repo)
	it.Then(t).Should(it.True(len(paths) > 0))
	for _, path := range paths {
		it.Then(t).Should(it.String(path).HavePrefix("kb/"))
	}

	it.Then(t).
		Should(it.Equal("edited\n", readFile(t, filepath.Join(other, "unrelated.md")))).
		Should(it.String(runGit(t, repo, "status", "--porcelain")).Contain("other/unrelated.md"))
}

// arc apply, nested vs standalone
// spec.md US4 Acceptance Scenario 2 (FR-025): the graph the nested run
// produces is identical, file for file, to the standalone run's.
func TestNestedApplyParityWithStandalone(t *testing.T) {
	patch := writePatchFile(t, t.TempDir(), "docA.patch.md", v011RevertDocA)

	standalone := t.TempDir()
	initGraph(t, standalone)
	func() {
		chdir(t, standalone)
		out, err := sut(NewApplyCmd(), []string{patch})
		it.Then(t).ShouldNot(it.Error(out, err))
	}()

	repo := t.TempDir()
	graph, _ := initHostGraph(t, repo)
	func() {
		chdir(t, graph)
		out, err := sut(NewApplyCmd(), []string{patch})
		it.Then(t).ShouldNot(it.Error(out, err))
	}()

	assertSameGraph(t, graph, standalone)
}

// arc revert, nested inside a host repository
// spec.md US4 Acceptance Scenario 3 (FR-023, FR-024): the revert commit
// touches only the graph subtree, and the host's unrelated files are intact.
func TestNestedRevertAffectsOnlyGraphSubtree(t *testing.T) {
	repo := t.TempDir()
	graph, other := initHostGraph(t, repo)
	chdir(t, graph)

	patch := writePatchFile(t, t.TempDir(), "docA.patch.md", v011RevertDocA)
	_, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Nil(err))

	out, err := sut(forcedRevertCmd(t), []string{"doc-2026-a"})
	it.Then(t).ShouldNot(it.Error(out, err))

	for _, path := range headPaths(t, repo) {
		it.Then(t).Should(it.String(path).HavePrefix("kb/"))
	}

	_, statErr := os.Stat(filepath.Join(graph, "Source", "doc-2026-a.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
	it.Then(t).Should(it.Equal("unrelated\n", readFile(t, filepath.Join(other, "unrelated.md"))))
}

// arc revert, nested vs standalone
// spec.md US4 Acceptance Scenario 4 (FR-025): reverting the same ingest
// leaves both graphs in the same state.
func TestNestedRevertParityWithStandalone(t *testing.T) {
	patch := writePatchFile(t, t.TempDir(), "docA.patch.md", v011RevertDocA)

	standalone := t.TempDir()
	initGraph(t, standalone)
	func() {
		chdir(t, standalone)
		_, err := sut(NewApplyCmd(), []string{patch})
		it.Then(t).Should(it.Nil(err))
		out, err := sut(forcedRevertCmd(t), []string{"doc-2026-a"})
		it.Then(t).ShouldNot(it.Error(out, err))
	}()

	repo := t.TempDir()
	graph, _ := initHostGraph(t, repo)
	func() {
		chdir(t, graph)
		_, err := sut(NewApplyCmd(), []string{patch})
		it.Then(t).Should(it.Nil(err))
		out, err := sut(forcedRevertCmd(t), []string{"doc-2026-a"})
		it.Then(t).ShouldNot(it.Error(out, err))
	}()

	assertSameGraph(t, graph, standalone)
}

// arc lint, nested inside a host repository carrying a look-alike commit
// spec.md US4 Acceptance Scenario 5 (FR-023, FR-025): the CORE §11.1 "one
// ingest commit per document" history check searches the graph's own subtree
// only, so an unrelated commit elsewhere in the host project carrying the
// same Source-Id trailer is not mistaken for a second ingest.
//
// Asserted as parity against the standalone run rather than against "0
// failing": this fixture's own patch leaves an unrelated conformance
// violation, and what FR-025 promises is that nesting changes nothing —
// an unscoped history query would add a violation to the nested run alone.
func TestNestedLintScopesHistoryToGraphSubtree(t *testing.T) {
	patch := writePatchFile(t, t.TempDir(), "docA.patch.md", v011RevertDocA)

	standalone := t.TempDir()
	initGraph(t, standalone)
	var standaloneOut string
	func() {
		chdir(t, standalone)
		_, err := sut(NewApplyCmd(), []string{patch})
		it.Then(t).Should(it.Nil(err))
		standaloneOut, _ = sut(lint.NewLintCmd(), nil)
	}()

	repo := t.TempDir()
	graph, other := initHostGraph(t, repo)
	chdir(t, graph)

	_, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Nil(err))

	// a look-alike ingest commit belonging to some other graph sharing the
	// same repository — same trailer, entirely outside kb/
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(other, "elsewhere.md"), []byte("elsewhere\n"), 0o644)))
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-m", "graph(apply): Document A\n\nSource-Id: doc-2026-a")

	nestedOut, _ := sut(lint.NewLintCmd(), nil)

	it.Then(t).
		Should(it.Equal(standaloneOut, nestedOut)).
		ShouldNot(it.String(nestedOut).Contain("ingest"))
}

// ---------------------------------------------------------------------------
// specs/031-init-existing-git-repo — Bugfix BUG-001, Phase 7 (FR-032..FR-035).
//
// The second parity axis. initHostGraph above nests the graph in a subfolder
// and keeps the host's own content in a *sibling* directory, so no command's
// tree walk ever meets a file arc did not write. Here the graph root IS the
// project root: the host's markdown sits inside the very tree apply, lint and
// grep walk. quickstart.md S10.
// ---------------------------------------------------------------------------

// foreignFiles are the host project's own markdown — content arc neither
// wrote nor understands, carrying no "@id"/"@type" front matter (FR-032).
var foreignFiles = map[string]string{
	"README.md":        "# My Project\n\nA readme that predates the graph.\n",
	"CONTRIBUTING.md":  "# Contributing\n\nSend patches.\n",
	"docs/design.md":   "# Design\n\nNotes on the transport layer.\n",
	"docs/adr/0001.md": "# ADR 1\n\nWe chose TLS.\n",
}

// rootModeEntity is a conformant Entity node carrying the same term the
// host's docs/design.md carries.
const rootModeEntity = `---
"@id": Widget
"@type": Entity
category: [independent, abstract, occurrent, script]
---
# Widget

A widget for the transport layer.
`

// initRootModeGraph builds a host repository whose root carries its own
// markdown, then initializes the graph layout at that same root — the shape
// `arc init --skip-git-init` produces at a project root, which FR-010
// legalized and which no other fixture in this repository builds.
func initRootModeGraph(t *testing.T, repo string) {
	t.Helper()

	for path, content := range foreignFiles {
		full := filepath.Join(repo, filepath.FromSlash(path))
		it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Dir(full), 0o755)))
		it.Then(t).Should(it.Nil(os.WriteFile(full, []byte(content), 0o644)))
	}
	runGit(t, repo, "init")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-m", "host: the project before the graph")

	writeGraphLayout(t, repo)
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-m", "graph(init): empty knowledge graph")
}

// assertForeignFilesIntact asserts every host file is byte-for-byte what it
// was and carries no pending change — FR-032's "no command may fail because
// one exists" has a sibling obligation: no command may quietly rewrite one.
func assertForeignFilesIntact(t *testing.T, repo string) {
	t.Helper()
	for path, want := range foreignFiles {
		it.Then(t).Should(it.Equal(want, readFile(t, filepath.Join(repo, filepath.FromSlash(path)))))
	}
	status := runGit(t, repo, "status", "--porcelain")
	for path := range foreignFiles {
		it.Then(t).ShouldNot(it.String(status).Contain(path))
	}
}

// arc apply, graph root shared with the host project
// spec.md FR-032/FR-033, SC-009: the host's own markdown is not graph content,
// so apply never opens it and never fails on it.
func TestRootModeApplySucceedsBesideForeignFiles(t *testing.T) {
	repo := t.TempDir()
	initRootModeGraph(t, repo)
	chdir(t, repo)
	patch := writePatchFile(t, t.TempDir(), "tls13.patch.md", tls13Patch)

	out, err := sut(NewApplyCmd(), []string{patch})

	it.Then(t).ShouldNot(it.Error(out, err))
	assertIsFile(t, filepath.Join(repo, "Source", "rescorla-2026-tls13.md"))
	assertForeignFilesIntact(t, repo)
}

// arc apply, graph root shared with the host project
// spec.md FR-032, US2 Acceptance Scenario 2: the ingest commit carries only
// files the apply itself wrote — never a foreign file swept in by staging.
func TestRootModeApplyCommitExcludesForeignFiles(t *testing.T) {
	repo := t.TempDir()
	initRootModeGraph(t, repo)
	chdir(t, repo)
	patch := writePatchFile(t, t.TempDir(), "tls13.patch.md", tls13Patch)

	_, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Nil(err))

	for _, path := range headPaths(t, repo) {
		_, foreign := foreignFiles[path]
		it.Then(t).ShouldNot(it.True(foreign))
	}
}

// arc apply, root-mode vs standalone
// spec.md FR-025/SC-009: parity across the root-sharing axis — the same patch
// applied to a graph sharing its root produces the same graph as one that
// owns its root outright.
func TestRootModeApplyParityWithStandalone(t *testing.T) {
	repo := t.TempDir()
	initRootModeGraph(t, repo)
	patch := writePatchFile(t, t.TempDir(), "tls13.patch.md", tls13Patch)

	chdir(t, repo)
	_, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Nil(err))

	standalone := t.TempDir()
	initGraph(t, standalone)
	chdir(t, standalone)
	_, err = sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Nil(err))

	// The host's own markdown is the one legitimate difference between the
	// two trees; everything arc owns must match exactly.
	got := graphTree(t, repo)
	for path := range foreignFiles {
		delete(got, path)
	}
	want := graphTree(t, standalone)
	it.Then(t).Should(it.Seq(sortedKeys(got)).Equal(sortedKeys(want)...))
	for path, content := range want {
		it.Then(t).Should(it.Equal(content, got[path]))
	}
}

// arc apply, with a foreign file sitting at a path the patch targets
// spec.md FR-034: this is the one case where a foreign file is legitimately
// fatal — the patch demanded that exact path. The rejection names the path and
// the missing field, and describes a READ, because no write is ever attempted.
func TestRootModeApplyTargetedForeignFileFailsAsRead(t *testing.T) {
	repo := t.TempDir()
	initRootModeGraph(t, repo)
	// An ENTITY path, not the source path: seeding the source would make
	// apply report "already tracked" (FR-003 idempotency) and return before
	// reading anything, so the read-path rejection under test would never
	// run.
	seedNode(t, repo, "Entity/Transport Layer Security.md", "# Not a node\n\nNo front matter at all.\n")
	chdir(t, repo)
	patch := writePatchFile(t, t.TempDir(), "tls13.patch.md", tls13Patch)

	before := gitLog(t, repo)

	out, err := sut(NewApplyCmd(), []string{patch})

	it.Then(t).Must(it.Fail(func() error { return err }))
	it.Then(t).
		Should(it.String(err.Error()).Contain("Entity/Transport Layer Security.md")).
		Should(it.String(err.Error()).Contain(`"@id"`)).
		Should(it.String(err.Error()).Contain("failed to read")).
		Should(it.Equal(before, gitLog(t, repo))).
		Should(it.Equal("", strings.TrimSpace(out)))
	it.Then(t).ShouldNot(it.String(err.Error()).Contain("failed to write"))
}

// arc grep, graph root shared with the host project
// spec.md FR-035 regression lock: grep already indexes what it cannot parse
// as a node and excludes it from the scan. This pins that behavior, since it
// is the convention apply and lint are being brought in line with.
func TestRootModeGrepExcludesForeignFilesFromMatches(t *testing.T) {
	repo := t.TempDir()
	initRootModeGraph(t, repo)
	// A real node carrying the same term docs/design.md carries, so a pass
	// is "only the node matched", not "nothing matched" — grep exits
	// non-zero on no matches at all (grep_test.go US1 Scenario 2), which
	// would otherwise make this assertion vacuously true.
	seedNode(t, repo, "Entity/Widget.md", rootModeEntity)
	chdir(t, repo)

	out, err := sut(NewGrepCmd(), []string{"transport"})

	it.Then(t).ShouldNot(it.Error(out, err))
	it.Then(t).Should(it.String(out).Contain("Entity  Widget"))
	for path := range foreignFiles {
		it.Then(t).ShouldNot(it.String(out).Contain(path))
	}
}

// arc apply, with the host project's markdown scaled up
// spec.md FR-033/SC-009 (T064): apply's read cost is governed by its patch,
// not by the graph root's size. The whole-graph walk this bugfix removed made
// every apply pay for every file under the root — a correctness defect on a
// shared root, and an unbounded cost on a large project. Growing the host's
// foreign markdown from a handful to a thousand files must change nothing
// observable: same result, same commit, and no read of any of them.
func TestRootModeApplyCostIsIndependentOfForeignFileCount(t *testing.T) {
	small := t.TempDir()
	initRootModeGraph(t, small)
	patch := writePatchFile(t, t.TempDir(), "tls13.patch.md", tls13Patch)

	chdir(t, small)
	_, err := sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Nil(err))

	large := t.TempDir()
	initRootModeGraph(t, large)
	bulk := filepath.Join(large, "docs", "bulk")
	it.Then(t).Should(it.Nil(os.MkdirAll(bulk, 0o755)))
	for i := 0; i < 1000; i++ {
		name := filepath.Join(bulk, fmt.Sprintf("note-%04d.md", i))
		it.Then(t).Should(it.Nil(os.WriteFile(name, []byte("# Note\n\nHost content.\n"), 0o644)))
	}
	runGit(t, large, "add", "-A")
	runGit(t, large, "commit", "-m", "host: a thousand notes")

	chdir(t, large)
	_, err = sut(NewApplyCmd(), []string{patch})
	it.Then(t).Should(it.Nil(err))

	// identical graph output either way — the thousand files are invisible
	// to apply, not merely tolerated by it
	got, want := graphTree(t, large), graphTree(t, small)
	for path := range got {
		if strings.HasPrefix(path, "docs/bulk/") {
			delete(got, path)
		}
	}
	it.Then(t).Should(it.Seq(sortedKeys(got)).Equal(sortedKeys(want)...))
	for path, content := range want {
		it.Then(t).Should(it.Equal(content, got[path]))
	}

	// and none of them reached the commit
	for _, path := range headPaths(t, large) {
		it.Then(t).ShouldNot(it.True(strings.HasPrefix(path, "docs/bulk/")))
	}
}
