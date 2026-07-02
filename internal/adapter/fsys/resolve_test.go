package fsys_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
)

func TestResolveLocalRootCreatesMissingDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "graph")

	created, err := fsys.ResolveLocalRoot(root)

	it.Then(t).
		Should(it.Nil(err)).
		Should(it.True(created))

	info, statErr := os.Stat(root)
	it.Then(t).
		Should(it.Nil(statErr)).
		Should(it.True(info.IsDir()))
}

func TestResolveLocalRootExistingDirectory(t *testing.T) {
	root := t.TempDir()

	created, err := fsys.ResolveLocalRoot(root)

	it.Then(t).
		Should(it.Nil(err)).
		Should(it.True(!created))
}

func TestResolveLocalRootTargetIsFile(t *testing.T) {
	root := filepath.Join(t.TempDir(), "target")
	it.Then(t).Should(it.Nil(os.WriteFile(root, []byte("x"), 0o644)))

	_, err := fsys.ResolveLocalRoot(root)

	it.Then(t).Should(it.True(errors.Is(err, fsys.ErrRootNotDirectory)))
}

func TestRemoveLocalRootRemovesCreatedRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "graph")
	_, err := fsys.ResolveLocalRoot(root)
	it.Then(t).Should(it.Nil(err))

	it.Then(t).Should(it.Nil(fsys.RemoveLocalRoot(root)))

	_, statErr := os.Stat(root)
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}
