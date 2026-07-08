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

	// ErrSchemaMissing is returned when _schema/predicates/ or
	// _schema/types/ is absent from an otherwise-initialized graph (spec
	// FR-014).
	ErrSchemaMissing = faults.Safe1[string]("schema folder %s is missing or unreadable")

	// ErrSchemaInvalid is returned when a predicate/type schema document
	// fails to parse, or is missing/has an invalid value for a mandatory
	// field (spec FR-014) — naming the offending file and field.
	ErrSchemaInvalid = faults.Safe2[string, string]("schema document %s has a missing or invalid %s")
)
