//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package port declares secondary ports private to the graph use-case.
package port

import "context"

// BlameLine is one current line of a node file's git-blame attribution
// (research.md D7) — content is never needed, only which commit last
// touched it.
type BlameLine struct {
	Number int
	Commit string
}

// VCS is narrower than internal/app/ctrl/port.VCS — apply never
// initializes a repository or re-checks git availability, since a graph it
// operates on is already arc init-ed.
type VCS interface {
	IsTracked(ctx context.Context, dir, path string) (bool, error)
	StageAll(ctx context.Context, dir string) error
	Commit(ctx context.Context, dir, message string) (hash string, err error)

	// CommitsMatching, ChangedPaths, CommitsTouching, RevertCommit, Blame,
	// and ShowFile back arc revert's ingest-commit lookup, eligibility
	// tests, and per-node reconciliation (research.md D1/D3/D4/D7/D8,
	// contracts/vcs-port-contract.md).
	CommitsMatching(ctx context.Context, dir, needle string) ([]string, error)
	ChangedPaths(ctx context.Context, dir, hash string) ([]string, error)
	CommitsTouching(ctx context.Context, dir, path string) ([]string, error)
	RevertCommit(ctx context.Context, dir, hash string) (newHash string, err error)
	Blame(ctx context.Context, dir, path string) ([]BlameLine, error)
	ShowFile(ctx context.Context, dir, hash, path string) ([]byte, error)
}
