// Package port declares secondary ports private to the ctrl use-case.
package port

import "context"

// VCS is the narrow, capability-scoped interface exactly the Init use-case
// needs — nothing mirroring the full git CLI surface.
type VCS interface {
	IsAvailable(ctx context.Context) error
	Init(ctx context.Context, dir string) error
	StageAll(ctx context.Context, dir string) error
	Commit(ctx context.Context, dir, message string) (hash string, err error)
}
