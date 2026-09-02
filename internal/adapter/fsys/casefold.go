//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package fsys

import (
	"errors"
	"io/fs"
	"path"
	"strings"
	"unicode"
)

// CaseFolder is an optional capability a Store may implement to report how
// the location it is mounted on treats letter case (spec 003 FR-026,
// research.md D11). It is deliberately NOT part of Store: case behavior is
// a property of a real mounted volume, so an in-memory or fake Store has no
// honest answer to give. Callers use FoldsCase below, which treats a Store
// that does not implement this as case-sensitive — exact identity
// comparison, the behavior every caller had before this capability existed.
//
// Case behavior belongs to the mounted volume, never to the operating
// system: macOS hosts case-sensitive APFS volumes, Linux mounts exFAT/CIFS/
// "ext4 casefold" case-insensitively, and a graph may live on an external
// or network volume that differs from the boot disk. An implementation MUST
// therefore probe the location itself and MUST NOT answer from GOOS or any
// build-time constant.
type CaseFolder interface {
	// FoldsCase reports whether this location resolves two names differing
	// only in letter case to the same file.
	FoldsCase() bool
}

// FoldsCase reports whether store's location folds letter case. A store
// that cannot answer (it does not implement CaseFolder) is reported as
// case-sensitive: identities are then compared exactly, which is what every
// caller did before FR-026 and what an in-memory fake actually does.
func FoldsCase(store fs.FS) bool {
	folder, ok := store.(CaseFolder)
	return ok && folder.FoldsCase()
}

// ResolveName reports the spelling that name actually carries on store,
// which is not always the spelling asked for: on a location that folds
// case, Open("Entity/Lightstep.md") happily returns the bytes of an on-disk
// "Entity/LightStep.md". found is false when nothing there matches name
// under the store's own case rules.
//
// This MUST be read via ReadDir, never Stat — os's fs.FileInfo.Name() is
// derived from the path string handed in, so Stat("Entity/Lightstep.md")
// succeeds on a case-insensitive volume and reports "Lightstep.md",
// repeating the caller's own spelling straight back and hiding the very
// disagreement this function exists to surface (research.md D11's "Stat
// trap"). ReadDir is the only source of a real directory entry's name.
//
// Note the division of labor with a Stat-based probe: asking "does this
// location fold case?" is an existence question, which Stat answers
// correctly; asking "what is this file actually called?" is a name
// question, which only ReadDir answers.
func ResolveName(store fs.FS, name string) (actual string, found bool, err error) {
	dir, base := path.Split(name)
	dir = strings.TrimSuffix(dir, "/")
	if dir == "" {
		dir = "."
	}

	// A listing is the only source of a real directory entry's name, but not
	// every Store can produce one (an in-memory fake may not enumerate at
	// all). Where it cannot, fall through to an exact existence check below
	// rather than reporting a file that plainly exists as missing.
	entries, err := fs.ReadDir(store, dir)
	if err != nil {
		entries = nil
	}

	// An exact match always wins, even where the location folds case: it is
	// the file the caller literally named.
	for _, entry := range entries {
		if entry.Name() == base {
			return name, true, nil
		}
	}

	if FoldsCase(store) {
		for _, entry := range entries {
			if strings.EqualFold(entry.Name(), base) {
				if dir == "." {
					return entry.Name(), true, nil
				}
				return dir + "/" + entry.Name(), true, nil
			}
		}
	}

	if len(entries) > 0 {
		// The listing was usable and named no match — the file is absent.
		return "", false, nil
	}

	f, err := store.Open(name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}
	f.Close()

	return name, true, nil
}

// flipCase returns name with the case of every cased letter inverted, and
// reports whether name had any cased letter to invert. A name with none
// (all digits, say) is useless as a case probe, since the flipped spelling
// would be the original and every location would appear to fold.
func flipCase(name string) (string, bool) {
	var flipped strings.Builder
	flipped.Grow(len(name))

	cased := false
	for _, r := range name {
		switch {
		case unicode.IsUpper(r):
			cased = true
			flipped.WriteRune(unicode.ToLower(r))
		case unicode.IsLower(r):
			cased = true
			flipped.WriteRune(unicode.ToUpper(r))
		default:
			flipped.WriteRune(r)
		}
	}

	return flipped.String(), cased
}
