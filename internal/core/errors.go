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
	// parse: a missing mandatory field (patch: "@type", document, published;
	// node: "@id", "@type"), or old-format detection (research.md D7) — a
	// legacy "kind" field present at all, "@id"/"@type" absent or empty, or
	// "@id" not matching the file's basename. .With's wrapped error always
	// names the specific file and the specific problem. For a patch manifest
	// it is row 5 of the recognition table (spec 021 data-model.md §2): the
	// fall-through reached when neither identity key declares a patch. Its
	// wording is deliberately unchanged by spec 021 (FR-006).
	ErrManifestInvalid = faults.Type("manifest is missing a mandatory field or uses the pre-0.5 node format")
	ErrPatchStructure  = faults.Type("patch body does not follow the H1-kind/H2-node section structure")

	// ErrManifestTypeConflict is row 1 of the patch-manifest recognition
	// table: the manifest declares both the current "@type" key and the
	// retired "kind" key with disagreeing values, so it contradicts itself.
	// Checked before every other row, so a self-contradictory manifest is
	// reported as a contradiction rather than as whichever half was read
	// first (spec 021 FR-005, research.md D4).
	ErrManifestTypeConflict = faults.Safe2[string, string]("manifest declares \"@type\": %s and legacy kind: %s — the two disagree")

	// ErrManifestLegacyKind is row 4: the manifest still declares itself with
	// the pre-0.5 "kind: patch" key and carries no "@type". The retired key is
	// recognized solely to produce this refusal — it is never accepted (spec
	// 021 FR-003). The message names both the offending key and its
	// replacement, so the file is correctable without consulting ARCNET-CORE.
	ErrManifestLegacyKind = faults.Type("manifest uses the retired \"kind: patch\" key — rewrite it as \"@type\": patch")

	// ErrManifestNotAPatch is row 3: the manifest declares an "@type" whose
	// value is not the literal lowercase "patch" — a different document kind,
	// or the right kind in the wrong casing ("Patch"). The offending value is
	// carried in the message so a casing error is shown, not merely asserted
	// (spec 021 FR-016).
	ErrManifestNotAPatch = faults.Safe1[string]("manifest declares \"@type\": %s, expected the literal lowercase \"patch\"")

	// ErrIdentityQuoting is returned when a patch's front matter writes an
	// identity key bare ("@type: patch" rather than "\"@type\": patch"). A
	// leading "@" is a reserved YAML plain-scalar indicator, so the document
	// is not merely mis-styled — it fails to parse outright, before manifest
	// recognition is ever reached (spec 021 FR-015, research.md D1). The
	// wording matches arc lint's own RuleIdentityQuoting message verbatim, so
	// a user meets the same sentence whichever command surfaces the problem.
	ErrIdentityQuoting = faults.Safe1[string]("%q must be a quoted YAML string key, found it unquoted")

	// ErrTypeCasing is returned when a patch's class-defining H1 heading or
	// explicit "@type" value does not begin with an uppercase letter (spec
	// 019 FR-004/FR-005/FR-008), naming the offending value.
	ErrTypeCasing = faults.Safe1[string]("class name %q must be CamelCase — start with an uppercase letter")
)
