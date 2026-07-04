//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package git_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/adapter/git"
	"github.com/fogfish/arcnet-cli/internal/bios"
)

func setGitIdentity(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_AUTHOR_NAME", "Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
}

func TestVCSIsAvailable(t *testing.T) {
	vcs := git.New(bios.NewReporter(true, true))

	err := vcs.IsAvailable(context.Background())

	it.Then(t).Should(it.Nil(err))
}

func TestVCSInitStageCommit(t *testing.T) {
	setGitIdentity(t)
	dir := t.TempDir()
	vcs := git.New(bios.NewReporter(true, true))
	ctx := context.Background()

	it.Then(t).Should(it.Nil(vcs.Init(ctx, dir)))

	writeFile(t, dir, "file.md", "content")

	it.Then(t).Should(it.Nil(vcs.StageAll(ctx, dir)))

	hash, err := vcs.Commit(ctx, dir, "graph(init): empty knowledge graph")
	it.Then(t).
		Should(it.Nil(err)).
		ShouldNot(it.Equal("", hash))
}

func TestVCSIsTrackedTrue(t *testing.T) {
	setGitIdentity(t)
	dir := t.TempDir()
	vcs := git.New(bios.NewReporter(true, true))
	ctx := context.Background()

	it.Then(t).Should(it.Nil(vcs.Init(ctx, dir)))
	writeFile(t, dir, "tracked.md", "content")
	it.Then(t).Should(it.Nil(vcs.StageAll(ctx, dir)))
	_, err := vcs.Commit(ctx, dir, "commit")
	it.Then(t).Should(it.Nil(err))

	tracked, err := vcs.IsTracked(ctx, dir, "tracked.md")
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.True(tracked))
}

func TestVCSIsTrackedFalse(t *testing.T) {
	setGitIdentity(t)
	dir := t.TempDir()
	vcs := git.New(bios.NewReporter(true, true))
	ctx := context.Background()

	it.Then(t).Should(it.Nil(vcs.Init(ctx, dir)))
	writeFile(t, dir, "untracked.md", "content")

	tracked, err := vcs.IsTracked(ctx, dir, "untracked.md")
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.True(!tracked))
}

func TestVCSIsTrackedError(t *testing.T) {
	dir := t.TempDir()
	vcs := git.New(bios.NewReporter(true, true))
	ctx := context.Background()

	_, err := vcs.IsTracked(ctx, dir, "whatever.md")
	it.Then(t).ShouldNot(it.Nil(err))
}

func TestVCSCommitsMatchingZeroMatches(t *testing.T) {
	setGitIdentity(t)
	dir := t.TempDir()
	vcs := git.New(bios.NewReporter(true, true))
	ctx := context.Background()

	it.Then(t).Should(it.Nil(vcs.Init(ctx, dir)))
	writeFile(t, dir, "file.md", "content")
	it.Then(t).Should(it.Nil(vcs.StageAll(ctx, dir)))
	_, err := vcs.Commit(ctx, dir, "graph(ingest): foo-2026-x — A Test Document\n\nSource-Id: foo-2026-x\n")
	it.Then(t).Should(it.Nil(err))

	hashes, err := vcs.CommitsMatching(ctx, dir, "Source-Id: bar-2026-y")
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.Equal(0, len(hashes)))
}

func TestVCSCommitsMatchingExactlyOneMatch(t *testing.T) {
	setGitIdentity(t)
	dir := t.TempDir()
	vcs := git.New(bios.NewReporter(true, true))
	ctx := context.Background()

	it.Then(t).Should(it.Nil(vcs.Init(ctx, dir)))
	writeFile(t, dir, "file.md", "content")
	it.Then(t).Should(it.Nil(vcs.StageAll(ctx, dir)))
	hash, err := vcs.Commit(ctx, dir, "graph(ingest): foo-2026-x — A Test Document\n\nSource-Id: foo-2026-x\n")
	it.Then(t).Should(it.Nil(err))

	hashes, err := vcs.CommitsMatching(ctx, dir, "Source-Id: foo-2026-x")
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.Equal(1, len(hashes)))
	it.Then(t).Should(it.True(strings.HasPrefix(hashes[0], hash) || strings.HasPrefix(hash, hashes[0])))
}

func TestVCSCommitsMatchingMoreThanOneMatch(t *testing.T) {
	setGitIdentity(t)
	dir := t.TempDir()
	vcs := git.New(bios.NewReporter(true, true))
	ctx := context.Background()

	it.Then(t).Should(it.Nil(vcs.Init(ctx, dir)))
	writeFile(t, dir, "file1.md", "content1")
	it.Then(t).Should(it.Nil(vcs.StageAll(ctx, dir)))
	_, err := vcs.Commit(ctx, dir, "graph(ingest): foo-2026-x — A Test Document\n\nSource-Id: foo-2026-x\n")
	it.Then(t).Should(it.Nil(err))

	writeFile(t, dir, "file2.md", "content2")
	it.Then(t).Should(it.Nil(vcs.StageAll(ctx, dir)))
	_, err = vcs.Commit(ctx, dir, "graph(ingest): foo-2026-x — A Test Document (re-ingest)\n\nSource-Id: foo-2026-x\n")
	it.Then(t).Should(it.Nil(err))

	hashes, err := vcs.CommitsMatching(ctx, dir, "Source-Id: foo-2026-x")
	it.Then(t).
		Should(it.Nil(err)).
		Should(it.Equal(2, len(hashes)))
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
	it.Then(t).Should(it.Nil(err))
}
