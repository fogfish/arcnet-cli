//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package ctrl

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/spf13/cobra"

	"github.com/fogfish/arcnet-cli/internal/app/schema/kernel"
	"github.com/fogfish/arcnet-cli/internal/bios"
)

// TestMain sets a fake git identity for the whole test binary. arc init
// shells out to a real `git commit`, which fails with "Author identity
// unknown" on any machine (including CI runners) that has no global
// user.name/user.email configured — the tool itself intentionally does not
// configure git identity (spec.md Assumptions), so the tests must supply
// their own, hermetically, rather than depend on the environment's global
// git config.
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

	w.Close()
	os.Stdout = stdout
	return <-ch, err
}

// sutCaptureStderr wraps sut, additionally capturing os.Stderr — needed to
// assert BUG-001's default-mode conciseness (no per-step git progress on
// stderr unless --verbose is set).
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

func assertIsDir(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.True(info.IsDir()))
}

func assertIsFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.True(!info.IsDir()))
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	it.Then(t).Should(it.Nil(err))
	return string(out)
}

// arc init
// Scenario 1 from specs/002-arc-init/spec.md, US1
func TestInitCurrentDirectoryCreatesLayout(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	out, err := sut(NewInitCmd(), []string{})

	it.Then(t).
		ShouldNot(it.Error(out, err)).
		Should(it.String(out).Contain(dir))

	for _, folder := range []string{"sources", "entities", "resources", filepath.Join("timeline", "yearly"), filepath.Join("timeline", "monthly"), filepath.Join("_schema", "types"), filepath.Join("_schema", "predicates")} {
		assertIsDir(t, filepath.Join(dir, folder))
	}
	assertIsFile(t, filepath.Join(dir, "_schema", "types", "entity.md"))
	assertIsFile(t, filepath.Join(dir, "_schema", "predicates", "related.md"))
	_, metaErr := os.Stat(filepath.Join(dir, "_meta"))
	it.Then(t).Should(it.True(os.IsNotExist(metaErr)))
	assertIsDir(t, filepath.Join(dir, ".arc"))

	gitignore, rerr := os.ReadFile(filepath.Join(dir, ".gitignore"))
	it.Then(t).
		Should(it.Nil(rerr)).
		Should(it.String(string(gitignore)).Contain(".arc/"))
}

// arc init
// Scenario 2 from specs/002-arc-init/spec.md, US1
func TestInitCurrentDirectorySingleCommit(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	log := strings.TrimSpace(gitOutput(t, dir, "log", "--oneline"))
	lines := strings.Split(log, "\n")

	it.Then(t).
		Should(it.Equal(1, len(lines))).
		Should(it.String(log).Contain("graph(init): empty knowledge graph"))
}

// arc init
// Scenario 3 from specs/002-arc-init/spec.md, US1
func TestInitCurrentDirectoryCleanWorkingTree(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	status := strings.TrimSpace(gitOutput(t, dir, "status", "--short"))
	it.Then(t).Should(it.Equal("", status))

	tracked := gitOutput(t, dir, "ls-files")
	it.Then(t).ShouldNot(it.String(tracked).Contain(".arc/"))
}

// arc init
// Scenario 4 from specs/002-arc-init/spec.md, US1
func TestInitCurrentDirectoryFoldersInHistory(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	tracked := gitOutput(t, dir, "ls-files")
	it.Then(t).
		Should(it.String(tracked).Contain("sources/.gitkeep")).
		Should(it.String(tracked).Contain("entities/.gitkeep")).
		Should(it.String(tracked).Contain("resources/.gitkeep")).
		Should(it.String(tracked).Contain("timeline/yearly/.gitkeep")).
		Should(it.String(tracked).Contain("timeline/monthly/.gitkeep")).
		Should(it.String(tracked).Contain("_schema/types/entity.md")).
		Should(it.String(tracked).Contain("_schema/predicates/related.md"))
}

// arc init
// spec.md US1 Acceptance Scenarios 1-2: every core predicate/type is
// seeded as a real, machine-readable document — role/merge (plus
// label/aligned where declared) and a description for every predicate;
// required/optional and a description for every type — not an
// existence-only stub.
func TestInitSeedsAllCoreKindsAndPredicates(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	for name, def := range kernel.CoreTypeDefs {
		path := filepath.Join(dir, "_schema", "types", name+".md")
		assertIsFile(t, path)

		content, rerr := os.ReadFile(path)
		it.Then(t).Should(it.Nil(rerr))
		it.Then(t).
			Should(it.String(string(content)).Contain(`"@type": Class`)).
			Should(it.String(string(content)).Contain("merge: " + string(def.Merge)))
		for _, required := range def.Required {
			it.Then(t).Should(it.String(string(content)).Contain("required:: [[" + required + "]]"))
		}
	}

	for name, def := range kernel.CorePredicateDefs {
		path := filepath.Join(dir, "_schema", "predicates", name+".md")
		assertIsFile(t, path)

		content, rerr := os.ReadFile(path)
		it.Then(t).Should(it.Nil(rerr))
		it.Then(t).
			Should(it.String(string(content)).Contain(`"@type": Property`)).
			Should(it.String(string(content)).Contain("role: " + def.Role)).
			Should(it.String(string(content)).Contain("merge: " + string(def.Merge)))
	}
}

// arc init
// spec.md US1 Acceptance Scenario 3: no _schema/nodes/ folder exists.
func TestInitNoSchemaNodesFolderExists(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := sut(NewInitCmd(), []string{})
	it.Then(t).Should(it.Nil(err))

	_, statErr := os.Stat(filepath.Join(dir, "_schema", "nodes"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

// arc init
// Scenario 5 from spec.md US1: initialization succeeds with no network
// access at all — there is no fetch to fail (research.md D5)
func TestInitSucceedsWithNoNetworkAccess(t *testing.T) {
	dir := t.TempDir()
	emptyProxy := "http://localhost:1"
	t.Setenv("HTTP_PROXY", emptyProxy)
	t.Setenv("HTTPS_PROXY", emptyProxy)

	out, err := sut(NewInitCmd(), []string{dir})
	it.Then(t).ShouldNot(it.Error(out, err))
	assertIsFile(t, filepath.Join(dir, "_schema", "types", "entity.md"))
}

// arc init <target-file>
// FR-010 edge case from specs/002-arc-init/spec.md
func TestInitTargetIsFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "not-a-dir")
	it.Then(t).Should(it.Nil(os.WriteFile(target, []byte("x"), 0o644)))

	out, err := sut(NewInitCmd(), []string{target})

	it.Then(t).Should(it.Error(out, err).Contain("directory"))

	content, rerr := os.ReadFile(target)
	it.Then(t).
		Should(it.Nil(rerr)).
		Should(it.Equal("x", string(content)))
}

// PATH=<empty> arc init <dir>
// FR-011 edge case from specs/002-arc-init/spec.md
func TestInitGitUnavailable(t *testing.T) {
	dir := t.TempDir()
	emptyPath := t.TempDir()
	t.Setenv("PATH", emptyPath)

	out, err := sut(NewInitCmd(), []string{dir})

	it.Then(t).Should(it.Error(out, err).Contain("git"))

	entries, rerr := os.ReadDir(dir)
	it.Then(t).
		Should(it.Nil(rerr)).
		Should(it.Equal(0, len(entries)))
}

// arc init <non-empty-dir>
// FR-015 edge case from specs/002-arc-init/spec.md
func TestInitTargetNonEmpty(t *testing.T) {
	dir := t.TempDir()
	it.Then(t).Should(it.Nil(os.WriteFile(filepath.Join(dir, "unrelated.txt"), []byte("x"), 0o644)))

	out, err := sut(NewInitCmd(), []string{dir})

	it.Then(t).Should(it.Error(out, err).Contain("empty"))

	entries, rerr := os.ReadDir(dir)
	it.Then(t).
		Should(it.Nil(rerr)).
		Should(it.Equal(1, len(entries)))
}

// arc init --json <dir>
// --json output contract from specs/002-arc-init/contracts/cli-contract.md
func TestInitJSONOutput(t *testing.T) {
	dir := t.TempDir()
	bios.JSON = true
	t.Cleanup(func() { bios.JSON = false })

	out, err := sut(NewInitCmd(), []string{dir})

	it.Then(t).ShouldNot(it.Error(out, err))

	var payload struct {
		Path           string   `json:"path"`
		Commit         string   `json:"commit"`
		FoldersCreated []string `json:"foldersCreated"`
	}
	it.Then(t).Should(it.Nil(json.Unmarshal([]byte(out), &payload)))
	it.Then(t).
		Should(it.Equal(dir, payload.Path)).
		ShouldNot(it.Equal("", payload.Commit)).
		Should(it.LessOrEqual(len(payload.Commit), 12)).
		Should(it.Seq(payload.FoldersCreated).Contain("sources", "entities", "resources", "_schema/types", "_schema/predicates"))
}

// arc init <dir>
// FR-016 from specs/002-arc-init/spec.md — default output is a single
// concise line; per-step git progress is opt-in via --verbose (BUG-001)
func TestInitDefaultModeIsConciseSingleLine(t *testing.T) {
	dir := t.TempDir()

	stdout, stderr, err := sutCaptureStderr(t, NewInitCmd(), []string{dir})

	it.Then(t).ShouldNot(it.Error(stdout, err))
	it.Then(t).
		Should(it.Equal(1, strings.Count(stdout, "\n"))).
		Should(it.Equal("", stderr))

	commit := strings.TrimSpace(strings.Split(strings.Split(stdout, "commit ")[1], ")")[0])
	fullHash := strings.TrimSpace(gitOutput(t, dir, "rev-parse", "HEAD"))
	it.Then(t).
		Should(it.True(len(commit) <= 12)).
		Should(it.True(strings.HasPrefix(fullHash, commit)))
}

// arc init --verbose <dir>
// --verbose progress contract from specs/002-arc-init/contracts/cli-contract.md (BUG-001)
func TestInitVerboseModeShowsGitProgress(t *testing.T) {
	dir := t.TempDir()
	bios.Verbose = true
	t.Cleanup(func() { bios.Verbose = false })

	stdout, stderr, err := sutCaptureStderr(t, NewInitCmd(), []string{dir})

	it.Then(t).ShouldNot(it.Error(stdout, err))
	it.Then(t).
		Should(it.String(stderr).Contain("Checking git availability")).
		Should(it.String(stderr).Contain("Preparing git repository")).
		Should(it.String(stderr).Contain("Committing empty graph"))
}

// arc init <non-existent-dir>
// Scenario 1 from specs/002-arc-init/spec.md, US2
func TestInitNamedDirectoryCreatesLayout(t *testing.T) {
	base := t.TempDir()
	cwd := filepath.Join(base, "cwd")
	it.Then(t).Should(it.Nil(os.MkdirAll(cwd, 0o755)))
	chdir(t, cwd)

	target := filepath.Join(base, "graph")

	out, err := sut(NewInitCmd(), []string{target})

	it.Then(t).ShouldNot(it.Error(out, err))
	assertIsDir(t, filepath.Join(target, "sources"))
	assertIsDir(t, filepath.Join(target, ".arc"))

	entries, rerr := os.ReadDir(cwd)
	it.Then(t).
		Should(it.Nil(rerr)).
		Should(it.Equal(0, len(entries)))
}

// arc init <dir>
// Scenario 2 from specs/002-arc-init/spec.md, US2
func TestInitNamedDirectoryReportsPath(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "graph")

	out, err := sut(NewInitCmd(), []string{target})

	it.Then(t).
		ShouldNot(it.Error(out, err)).
		Should(it.String(out).Contain(target))
}

// arc init <already-a-graph>
// Scenario 1 from specs/002-arc-init/spec.md, US3 (FR-014)
func TestInitRefusesReInitialization(t *testing.T) {
	dir := t.TempDir()
	_, err := sut(NewInitCmd(), []string{dir})
	it.Then(t).Should(it.Nil(err))

	before := gitOutput(t, dir, "log", "--oneline")

	out, err := sut(NewInitCmd(), []string{dir})

	it.Then(t).Should(it.Error(out, err).Contain("already"))

	after := gitOutput(t, dir, "log", "--oneline")
	it.Then(t).Should(it.Equal(before, after))
}
