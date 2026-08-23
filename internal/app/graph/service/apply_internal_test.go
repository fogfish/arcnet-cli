//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// package service (white-box, not service_test): nodeFolder and nodePath are
// unexported pure functions, and asserting the folder-derivation rule
// directly is the only way to assert it as a *string* rather than as a
// filesystem effect — which contract C7 requires, because os.Stat on the
// developer machine's case-insensitive APFS reports "Source/", "source/",
// and "SOURCE/" as the same path and would hide a case defect until CI.
// Mirrors this package's own revert_internal_test.go and
// batch_internal_test.go precedent.
package service

import (
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/schema/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// TestNodeFolderIsTheIdentityFunction is contract C1: a type folder's name
// equals its type name character for character, so the derivation applies no
// transform at all. Asserted over every seeded content type and over types
// the tool does not recognize — the rule holds identically for both, which
// is the whole reason it is the identity rather than a lookup with a
// fallback.
//
// Thought is the case that would have failed under the pre-v0.11 derivation:
// its kind+"s" fallback filed an unrecognized type under Thoughts/.
func TestNodeFolderIsTheIdentityFunction(t *testing.T) {
	for name := range kernel.CoreTypeBases {
		if name == "Timeline" {
			continue
		}
		it.Then(t).Should(it.Equal(name, nodeFolder(name)))
	}

	// Unrecognized domain types: no pluralizing suffix, no case change, and
	// no normalization in either direction for a name that is already
	// lowercase or already plural (spec.md Edge Cases).
	for _, name := range []string{"Thought", "Hypothesis", "Series", "thoughts", "aporia"} {
		it.Then(t).Should(it.Equal(name, nodeFolder(name)))
	}
}

// TestNodePathComposesTypeSlashIDDotMd is contract C4: the path a node is
// written to is its type folder, its own @id, and ".md" — asserted as the
// exact string the store is handed.
func TestNodePathComposesTypeSlashIDDotMd(t *testing.T) {
	for _, tt := range []struct {
		node core.Node
		want string
	}{
		{core.Node{ID: "doe-2026-probe", Type: "Source"}, "Source/doe-2026-probe.md"},
		{core.Node{ID: "Transport Layer Security", Type: "Entity"}, "Entity/Transport Layer Security.md"},
		{core.Node{ID: "probe-fragment", Type: "Resource"}, "Resource/probe-fragment.md"},
		{core.Node{ID: "rfc-8446", Type: "Reference"}, "Reference/rfc-8446.md"},
		{core.Node{ID: "a-passing-thought", Type: "Thought"}, "Thought/a-passing-thought.md"},
	} {
		it.Then(t).Should(it.Equal(tt.want, nodePath(tt.node)))
	}
}
