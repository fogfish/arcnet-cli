package fsys

import (
	"io/fs"
	"os"
	"path/filepath"
)

// Local is the sole Mounter/Store implementation, wrapping os.DirFS(root)
// for reads and os.Create/os.MkdirAll/os.Remove for writes. It requires
// ResolveLocalRoot(root) to have already succeeded — it performs no
// existence checks or creation of its own.
type Local struct{}

func (Local) Mount(root string) (Store, error) {
	return &local{root: root, fsys: os.DirFS(root)}, nil
}

type local struct {
	root string
	fsys fs.FS
}

func (l *local) Open(name string) (fs.File, error) { return l.fsys.Open(name) }

func (l *local) Stat(name string) (fs.FileInfo, error) { return fs.Stat(l.fsys, name) }

func (l *local) ReadDir(name string) ([]fs.DirEntry, error) { return fs.ReadDir(l.fsys, name) }

func (l *local) Create(name string) (File, error) {
	path := filepath.Join(l.root, filepath.FromSlash(name))

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, ErrCreate.With(err, name)
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, ErrCreate.With(err, name)
	}

	return &localFile{File: f, path: path}, nil
}

func (l *local) Remove(name string) error {
	path := filepath.Join(l.root, filepath.FromSlash(name))
	if err := os.Remove(path); err != nil {
		return ErrRemove.With(err, name)
	}
	return nil
}

type localFile struct {
	*os.File
	path string
}

func (f *localFile) Discard() error {
	f.File.Close()
	return os.Remove(f.path)
}
