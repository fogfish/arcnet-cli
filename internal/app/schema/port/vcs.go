//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package port declares secondary ports private to the schema use-case's
// apply-schema operation.
package port

import "context"

// VCS is narrower than internal/app/graph/port.VCS — apply schema never
// checks source-document idempotency and never initializes a repository,
// since the graph it operates on is already arc init-ed (research.md D3).
// Satisfied structurally by the existing internal/adapter/git.VCS concrete
// type (ADR 001 port isolation rule 1) — no new adapter code.
type VCS interface {
	StageAll(ctx context.Context, dir string) error
	Commit(ctx context.Context, dir, message string) (hash string, err error)
}
