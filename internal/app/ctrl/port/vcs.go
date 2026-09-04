//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package port declares secondary ports private to the ctrl use-case.
package port

import "context"

// VCS is the narrow, capability-scoped interface exactly the Init use-case
// needs — nothing mirroring the full git CLI surface.
type VCS interface {
	IsAvailable(ctx context.Context) error
	Init(ctx context.Context, dir string) error
	// StagePaths stages exactly the given graph-relative paths, and
	// nothing else. Unlike StageAll it never sweeps in unrelated changes,
	// which is required when the graph root and the repository root
	// coincide (contracts/cli-contract.md C7).
	StagePaths(ctx context.Context, dir string, paths []string) error

	// CommitPaths commits exactly the given paths, ignoring whatever else
	// the index holds. Distinct from Commit, which commits the whole
	// index — including anything the user staged themselves before running
	// arc init (research.md D3, FR-014).
	CommitPaths(ctx context.Context, dir, message string, paths []string) (hash string, err error)
}
