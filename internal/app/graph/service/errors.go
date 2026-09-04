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
	ErrNotAGraph = faults.Safe1[string]("%s is not an initialized graph")
	ErrPatchRead = faults.Safe1[string]("failed to read patch file %s")
	ErrNodeWrite = faults.Safe1[string]("failed to write %s")
	// ErrNodeRead reports a rejection raised while READING an existing node
	// file — a parse failure, or an "@id" that drifted from its own
	// basename. It exists because the write sentinel above was doing both
	// jobs, so a file arc had only ever read was reported as one it had
	// failed to write (spec 031 FR-034, BUG-001). The distinction is not
	// cosmetic: "failed to write README.md" sends the reader looking for a
	// permissions or disk fault that does not exist, which is precisely
	// what Constitution X's actionable-output rule (spec 031 FR-029) forbids.
	ErrNodeRead        = faults.Safe1[string]("failed to read %s")
	ErrInvalidPattern  = faults.Safe1[string]("%s is not a valid pattern")
	ErrInvalidAttrFlag = faults.Safe1[string]("--attr %s must be name=value or name~=pattern")
	ErrSeedNotFound    = faults.Safe1[string]("no node found with basename %s")
	ErrInvalidDepth    = faults.Safe1[string]("--depth %s must be a non-negative integer")
	ErrInvalidLimit    = faults.Safe1[string]("limit %s must be a positive integer")

	// ErrHTTPAddr and ErrInvalidFilterPattern are arc serve's own sentinels
	// (specs/008-arc-serve-mcp): an invalid/in-use --http address, and an
	// MCP filter object's attrPatterns value that is not a valid regexp.
	ErrHTTPAddr             = faults.Safe1[string]("invalid or unavailable --http address %s")
	ErrInvalidFilterPattern = faults.Safe1[string]("%s is not a valid pattern")

	// ErrNoIngestCommit is arc revert's own sentinel (research.md D1) for
	// the "no commit at all" case. More than one match is not an error —
	// see D1's corrected rationale (BUG-001): it is the expected result
	// of a retract-then-reapply cycle, resolved by acting on the newest
	// match rather than refusing.
	ErrNoIngestCommit = faults.Safe1[string]("no ingest commit found for %s")

	// ErrIdentityCharset rejects a patch that would introduce or modify a
	// node — content or schema — whose identity contains an ARCNET-CORE
	// §7.1 forbidden filesystem character (spec.md FR-001-FR-003,
	// data-model.md §5). Its second argument is the same detail fragment
	// core.FormatIdentityCharsetViolation renders for arc lint's
	// RuleIdentityCharset violation, so the two surfaces word the same
	// underlying rule identically (research.md D8).
	ErrIdentityCharset = faults.Safe2[string, string]("identity %q %s")

	// ErrEmptyFilter is node_match's own sentinel (specs/028-node-match-
	// filter, research.md D2): a missing or zero-statement filter is
	// rejected outright rather than silently matching (and reporting
	// facts for) every node.
	ErrEmptyFilter = faults.Type("filter must contain at least one statement")
)
