//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import "github.com/fogfish/faults"

// ErrSchemaWrite is returned when RegisterKind/RegisterPredicate fails to
// write a new schema document.
const ErrSchemaWrite = faults.Safe1[string]("failed to write schema document %s")
