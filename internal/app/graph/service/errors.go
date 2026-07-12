//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import (
	"errors"

	"github.com/fogfish/faults"
)

// errNoCause is passed to a faults.SafeN.With for guard conditions that
// are not caused by an underlying Go error, so the rendered message has
// no trailing "%!s(<nil>)" artifact (mirrors internal/core's own
// precedent, errNoCause in internal/core/markdown.go).
var errNoCause = errors.New("")

const (
	ErrNotAGraph       = faults.Safe1[string]("%s is not an initialized graph")
	ErrPatchRead       = faults.Safe1[string]("failed to read patch file %s")
	ErrNodeWrite       = faults.Safe1[string]("failed to write %s")
	ErrInvalidPattern  = faults.Safe1[string]("%s is not a valid pattern")
	ErrInvalidAttrFlag = faults.Safe1[string]("--attr %s must be name=value or name~=pattern")
	ErrSeedNotFound    = faults.Safe1[string]("no node found with basename %s")
	ErrInvalidDepth    = faults.Safe1[string]("--depth %s must be a non-negative integer")

	// ErrHTTPAddr and ErrInvalidFilterPattern are arc serve's own sentinels
	// (specs/008-arc-serve-mcp): an invalid/in-use --http address, and an
	// MCP filter object's attrPatterns value that is not a valid regexp.
	ErrHTTPAddr             = faults.Safe1[string]("invalid or unavailable --http address %s")
	ErrInvalidFilterPattern = faults.Safe1[string]("%s is not a valid pattern")

	// ErrNoIngestCommit and ErrAmbiguousIngestCommit are arc revert's own
	// sentinels (research.md D1): no commit at all, or more than one —
	// the latter is an integrity anomaly the patch format guarantees
	// cannot happen (spec 003 FR-011), so it points at arc lint's own
	// RuleIngestCommit check rather than guessing which commit is "the"
	// one.
	ErrNoIngestCommit        = faults.Safe1[string]("no ingest commit found for %s")
	ErrAmbiguousIngestCommit = faults.Safe1[string]("more than one ingest commit found for %s — run `arc lint` to check the Source-Id: trailer invariant")
)
