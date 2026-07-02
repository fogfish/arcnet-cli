//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package fsys is the shared, cross-use-case filesystem adapter (ADR 001's
// application-level adapter tier). It is the only package in the entire
// codebase permitted to call os's file/directory functions (constitution
// Principle VII, Mandatory Libraries & Tooling: "Filesystem Abstraction").
package fsys

import (
	"io"
	"io/fs"
)

// File is the writable counterpart to fs.File — io/fs itself is read-only
// by design (fs.FS only defines Open). Discard undoes an in-flight write
// instead of leaving a half-written file behind.
type File interface {
	io.Writer
	io.Closer
	Stat() (fs.FileInfo, error)
	Discard() error
}

// Store is the capability a use-case needs to read and write a mounted
// graph root.
type Store interface {
	fs.FS
	fs.StatFS
	fs.ReadDirFS
	Create(name string) (File, error)
	Remove(name string) error
}

// Mounter mounts an already-resolved root as a Store. It performs no
// existence checks and no root creation — that is ResolveLocalRoot's job.
type Mounter interface {
	Mount(root string) (Store, error)
}
