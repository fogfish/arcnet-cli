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
