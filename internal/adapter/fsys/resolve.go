package fsys

import (
	"errors"
	"os"
)

// errNoCause is passed to faults.SafeN.With for guard conditions that are
// not caused by an underlying Go error (e.g. a successful stat proving a
// path already exists), so the rendered message has no trailing
// "%!s(<nil>)" artifact.
var errNoCause = errors.New("")

// ResolveLocalRoot ensures root exists as a local directory. It creates
// root if missing and reports whether it did so, so a caller can undo
// exactly that via RemoveLocalRoot on a later failure (FR-013). It also
// owns the FR-010 "target is a file, not a directory" check, since this is
// the one step that inspects the raw path before any Store exists.
func ResolveLocalRoot(root string) (created bool, err error) {
	info, statErr := os.Stat(root)

	switch {
	case statErr == nil:
		if !info.IsDir() {
			return false, ErrRootNotDirectory.With(errNoCause, root)
		}
		return false, nil
	case os.IsNotExist(statErr):
		if err := os.MkdirAll(root, 0o755); err != nil {
			return false, ErrRootCreate.With(err, root)
		}
		return true, nil
	default:
		return false, ErrRootCreate.With(statErr, root)
	}
}

// RemoveLocalRoot undoes a root ResolveLocalRoot created. It MUST only ever
// be called when the immediately-preceding ResolveLocalRoot call for the
// same root returned created=true.
func RemoveLocalRoot(root string) error {
	if err := os.RemoveAll(root); err != nil {
		return ErrRemove.With(err, root)
	}
	return nil
}
