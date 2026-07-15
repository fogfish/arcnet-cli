//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package kernel

// ArcnetCatalogBaseURL is the base an "arcnet:<suffix>" input resolves
// against (research.md D1a). A var, not a const, purely so an E2E test can
// temporarily repoint it at an httptest.Server for the duration of one
// test, mirroring internal/app/ctrl/service.go's resolveLocalRoot/
// removeLocalRoot precedent — production code never reassigns it.
var ArcnetCatalogBaseURL = "https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/schema/"

// ApplySchemaResult is the value service.ApplyPatch returns, rendered by
// bios.Registry[ApplySchemaResult] in cmd/arc/ctrl. Unlike
// internal/app/graph/kernel.ApplyResult, it carries no Skipped field — a
// no-op re-apply is expressed as zero-valued Created/Merged and an empty
// CommitHash (research.md D7), not a distinct boolean state.
type ApplySchemaResult struct {
	// Source is the resolved local path or URL the patch was read from.
	Source string `json:"source"`
	// Created holds counts of newly created definitions, keyed
	// "predicate"/"type".
	Created map[string]int `json:"created"`
	// Merged holds counts of definitions merged into an existing document,
	// same keys.
	Merged map[string]int `json:"merged"`
	// CommitHash is the short hash of the resulting commit; empty when
	// nothing changed.
	CommitHash string `json:"commit"`
}
