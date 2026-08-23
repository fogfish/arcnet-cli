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
	"strings"

	"github.com/fogfish/faults"
)

// errNoCause is passed to a faults.SafeN.With for guard conditions that are
// not caused by an underlying Go error, so the rendered message has no
// trailing "%!s(<nil>)" artifact (mirrors internal/core's own precedent).
var errNoCause = errors.New("")

const (
	// ErrNotAGraph is returned when Resolve is called against a directory
	// with no ".arc/" marker (research.md D2) — distinct from
	// ErrSchemaMissing/ErrSchemaInvalid, which mean the directory is a
	// graph whose own _schema/ content is broken.
	ErrNotAGraph = faults.Type(`not an initialized graph, run "arc init" first`)

	// ErrSchemaWrite is returned when RegisterType/RegisterPredicate fails
	// to write a new schema document.
	ErrSchemaWrite = faults.Safe1[string]("failed to write schema document %s")

	// ErrSchemaMissing is returned when _schema/Property/ or
	// _schema/Class/ is absent from an otherwise-initialized graph (spec
	// FR-014).
	ErrSchemaMissing = faults.Safe1[string]("schema folder %s is missing or unreadable")

	// ErrSchemaInvalid is returned when a predicate/type schema document
	// fails to parse, or is missing/has an invalid value for a mandatory
	// field (spec FR-014) — naming the offending file and field.
	ErrSchemaInvalid = faults.Safe2[string, string]("schema document %s has a missing or invalid %s")

	// ErrSchemaCycle is returned when a type's rdfs:subClassOf declarations
	// form a cycle, of any length including direct self-reference — naming
	// the type where the cycle was detected (spec 017 FR-010).
	ErrSchemaCycle = faults.Safe1[string]("type %s participates in an rdfs:subClassOf cycle")

	// ErrSchemaUnresolvedBase is returned when a type's rdfs:subClassOf
	// declaration names a base type with no corresponding registered type —
	// naming the type and the unresolved base-type reference (spec 017
	// FR-011).
	ErrSchemaUnresolvedBase = faults.Safe2[string, string]("type %s declares rdfs:subClassOf an unresolved base type %s")

	// ErrDisallowedNodeType is returned by ApplyPatch when a patch node's
	// "@id"/"@type" is not Property/Class; names both. The entire operation
	// fails with zero writes (spec FR-005/FR-006).
	ErrDisallowedNodeType = faults.Safe2[string, string]("patch node %q has type %q — arc apply schema only accepts Property and Class nodes")

	// ErrPatchRead is returned by ApplyPatch when the patch source (local
	// path, URL, or resolved arcnet: reference) could not be read/fetched
	// or failed to parse; names the source.
	ErrPatchRead = faults.Safe1[string]("failed to read patch %s")

	// ErrEmptyArcnetReference is returned when the input is the bare prefix
	// "arcnet:" with nothing after it (spec FR-002a edge case) — rejected
	// before any fetch attempt.
	ErrEmptyArcnetReference = faults.Type(`"arcnet:" must be followed by a catalog path, e.g. arcnet:media.schema.md`)
)

// ErrSchemaMergeInvalid is returned when a predicate schema document
// declares a merge value outside CORE §9.3's closed set of six
// (specs/023-core-vocabulary-conformance FR-002, contract C1.2).
//
// Its message names four things, in order: the offending document, the
// offending value, the six legal values, and `arc upgrade` as the remedy.
// The fourth is MANDATORY, not decoration. A graph seeded by a release
// before spec 023 carries "validatedOverwrite" on its own scoreZ.md and
// scoreC.md, so this error is the FIRST thing an existing user sees after
// upgrading the binary — every other command hard-fails on it. Without the
// remedy the failure is a dead end whose only escape is hand-editing.
//
// The legal-value list is built from mergeMenu rather than written out, so
// the message can never advertise a menu the gate does not enforce.
var ErrSchemaMergeInvalid = faults.Safe2[string, string](
	"schema document %s declares merge %q; must be one of " + mergeMenuList() +
		". Run `arc upgrade` to bring this graph's built-in schema up to date",
)

// mergeMenuList renders mergeMenu as a comma-separated list.
func mergeMenuList() string {
	names := make([]string, 0, len(mergeMenu))
	for _, op := range mergeMenu {
		names = append(names, string(op))
	}
	return strings.Join(names, ", ")
}
