//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import "github.com/fogfish/faults"

const ErrNotAGraph = faults.Safe1[string]("%s is not an initialized graph")

// ErrUnknownSkipRule rejects an arc lint --skip value naming one or more
// rules kernel.RuleDefinitions does not list (spec 033 FR-007).
const ErrUnknownSkipRule = faults.Safe1[string]("unknown rule(s) in --skip: %s — run `arc lint rules` to see valid rule names")
