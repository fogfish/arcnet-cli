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

// VCS is narrower than internal/app/ctrl/port.VCS — apply never
// initializes a repository or re-checks git availability, since a graph it
// operates on is already arc init-ed.
type VCS interface {
	IsTracked(ctx context.Context, dir, path string) (bool, error)
	StageAll(ctx context.Context, dir string) error
	Commit(ctx context.Context, dir, message string) (hash string, err error)
}
