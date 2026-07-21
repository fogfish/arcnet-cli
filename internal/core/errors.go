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
	// ErrManifestInvalid covers every reason a node/patch manifest fails to
	// parse: a missing mandatory field (patch: kind, document, published;
	// node: "@id", "@type"), or old-format detection (research.md D7) — a
	// legacy "kind" field present at all, "@id"/"@type" absent or empty, or
	// "@id" not matching the file's basename. .With's wrapped error always
	// names the specific file and the specific problem.
	ErrManifestInvalid = faults.Type("manifest is missing a mandatory field or uses the pre-0.5 node format")
	ErrPatchStructure  = faults.Type("patch body does not follow the H1-kind/H2-node section structure")

	// ErrTypeCasing is returned when a patch's class-defining H1 heading or
	// explicit "@type" value does not begin with an uppercase letter (spec
	// 019 FR-004/FR-005/FR-008), naming the offending value.
	ErrTypeCasing = faults.Safe1[string]("class name %q must be CamelCase — start with an uppercase letter")
)
