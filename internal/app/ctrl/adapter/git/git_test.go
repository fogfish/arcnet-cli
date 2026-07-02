package git_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/ctrl/adapter/git"
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

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
	it.Then(t).Should(it.Nil(err))
}
