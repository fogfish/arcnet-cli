//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import "github.com/fogfish/faults"

const (
	ErrConfigMalformed = faults.Safe1[string]("%s is not valid YAML")
	ErrConfigConflict  = faults.Safe1[string]("%s already registers a different merge rule for this kind")
)
