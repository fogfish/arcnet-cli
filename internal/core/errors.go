//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package core

import "github.com/fogfish/faults"

const (
	ErrManifestInvalid = faults.Type("patch manifest is missing a mandatory field (kind: patch, document, published)")
	ErrPatchStructure  = faults.Type("patch body does not follow the H1-kind/H2-node section structure")
	ErrUnknownMergeOp  = faults.Safe1[string]("%s is not a recognized merge operation")
)
