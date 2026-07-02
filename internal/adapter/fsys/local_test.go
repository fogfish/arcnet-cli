//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package fsys_test

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
)

func TestLocalCreateWritesFile(t *testing.T) {
	root := t.TempDir()
	store, err := fsys.Local{}.Mount(root)
	it.Then(t).Should(it.Nil(err))

	f, err := store.Create("sub/dir/file.md")
	it.Then(t).Should(it.Nil(err))
	_, werr := f.Write([]byte("hello"))
	it.Then(t).Should(it.Nil(werr))
	it.Then(t).Should(it.Nil(f.Close()))

	content, rerr := os.ReadFile(filepath.Join(root, "sub", "dir", "file.md"))
	it.Then(t).
		Should(it.Nil(rerr)).
		Should(it.Equal("hello", string(content)))
}

func TestLocalStatAndReadDir(t *testing.T) {
	root := t.TempDir()
	store, err := fsys.Local{}.Mount(root)
	it.Then(t).Should(it.Nil(err))

	f, err := store.Create("sources/.gitkeep")
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Nil(f.Close()))

	info, statErr := store.Stat("sources")
	it.Then(t).
		Should(it.Nil(statErr)).
		Should(it.True(info.IsDir()))

	entries, rdErr := store.ReadDir(".")
	it.Then(t).
		Should(it.Nil(rdErr)).
		Should(it.Equal(1, len(entries)))
}

func TestLocalReadDirNotExist(t *testing.T) {
	root := t.TempDir()
	store, err := fsys.Local{}.Mount(root)
	it.Then(t).Should(it.Nil(err))

	_, statErr := store.Stat(".arc")
	it.Then(t).Should(it.True(fs.ValidPath(".arc")))
	it.Then(t).Should(it.True(os.IsNotExist(statErr) || statErr != nil))
}

func TestLocalRemove(t *testing.T) {
	root := t.TempDir()
	store, err := fsys.Local{}.Mount(root)
	it.Then(t).Should(it.Nil(err))

	f, err := store.Create("_meta/aliases.md")
	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Nil(f.Close()))

	it.Then(t).Should(it.Nil(store.Remove("_meta/aliases.md")))

	_, statErr := os.Stat(filepath.Join(root, "_meta", "aliases.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

func TestLocalDiscardRemovesHalfWrittenFile(t *testing.T) {
	root := t.TempDir()
	store, err := fsys.Local{}.Mount(root)
	it.Then(t).Should(it.Nil(err))

	f, err := store.Create("sources/broken.md")
	it.Then(t).Should(it.Nil(err))
	_, werr := f.Write([]byte("partial"))
	it.Then(t).Should(it.Nil(werr))

	it.Then(t).Should(it.Nil(f.Discard()))

	_, statErr := os.Stat(filepath.Join(root, "sources", "broken.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

func TestLocalOpenReadsFile(t *testing.T) {
	root := t.TempDir()
	store, err := fsys.Local{}.Mount(root)
	it.Then(t).Should(it.Nil(err))

	f, err := store.Create("_meta/predicates.md")
	it.Then(t).Should(it.Nil(err))
	_, werr := f.Write([]byte("content"))
	it.Then(t).Should(it.Nil(werr))
	it.Then(t).Should(it.Nil(f.Close()))

	opened, oerr := store.Open("_meta/predicates.md")
	it.Then(t).Should(it.Nil(oerr))
	data, rerr := io.ReadAll(opened)
	it.Then(t).
		Should(it.Nil(rerr)).
		Should(it.Equal("content", string(data)))
	it.Then(t).Should(it.Nil(opened.Close()))
}
