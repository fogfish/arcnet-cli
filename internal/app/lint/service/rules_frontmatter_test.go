//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import (
	"fmt"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

var coreIndexFixture = core.Index{
	Types: map[string]core.TypeDef{
		"Source":   {Merge: core.MergeImmutable},
		"Entity":   {Merge: core.MergeUnion},
		"Resource": {Merge: core.MergeFirstWriteWin},
		"Timeline": {Merge: core.MergeAppend},
	},
}

func TestCheckUniqueBasenamesNoCollision(t *testing.T) {
	index := map[string][]string{
		"foo": {"Source/foo.md"},
		"bar": {"Entity/bar.md"},
	}
	out := checkUniqueBasenames(index)
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckUniqueBasenamesTwoWayCollision(t *testing.T) {
	index := map[string][]string{
		"rfc8446": {"Resource/rfc8446.md", "Entity/rfc8446.md"},
	}
	out := checkUniqueBasenames(index)
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).
		Should(it.Equal(kernel.RuleUniqueBasename, out[0].Rule)).
		Should(it.Equal("", out[0].Path)).
		Should(it.Equal(2, len(out[0].RelatedPaths))).
		Should(it.String(out[0].Message).Contain("rfc8446"))
}

func TestCheckUniqueBasenamesThreeWayCollisionNamesEveryFile(t *testing.T) {
	index := map[string][]string{
		"widget": {"a/widget.md", "b/widget.md", "c/widget.md"},
	}
	out := checkUniqueBasenames(index)
	it.Then(t).
		Should(it.Equal(1, len(out))).
		Should(it.Equal(3, len(out[0].RelatedPaths)))
}

func TestCheckUnrecognizedKindRecognized(t *testing.T) {
	node := core.Node{Type: "Source"}
	out := checkUnrecognizedKind(node, "Source/foo.md", coreIndexFixture)
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckUnrecognizedKindUnrecognized(t *testing.T) {
	node := core.Node{Type: "hypothesis"}
	out := checkUnrecognizedKind(node, "hypothesis/foo.md", coreIndexFixture)
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).
		Should(it.Equal(kernel.RuleUnrecognizedKind, out[0].Rule)).
		Should(it.String(out[0].Message).Contain("hypothesis"))
}

func TestCheckUnrecognizedKindConfigRegistered(t *testing.T) {
	index := core.Index{Types: map[string]core.TypeDef{
		"Source":     {Merge: core.MergeImmutable},
		"Entity":     {Merge: core.MergeUnion},
		"Resource":   {Merge: core.MergeFirstWriteWin},
		"Timeline":   {Merge: core.MergeAppend},
		"hypothesis": {Merge: core.MergeValidatedOverwrite},
	}}
	node := core.Node{Type: "hypothesis"}
	out := checkUnrecognizedKind(node, "hypothesis/foo.md", index)
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckIdentityKeyQuotingBothQuotedNoViolation(t *testing.T) {
	raw := []byte("---\n\"@id\": foo\n\"@type\": Source\n---\n")
	out := checkIdentityKeyQuoting(core.Node{}, "Source/foo.md", raw)
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckIdentityKeyQuotingUnquotedIdReportsViolation(t *testing.T) {
	raw := []byte("---\n@id: foo\n\"@type\": Source\n---\n")
	out := checkIdentityKeyQuoting(core.Node{}, "Source/foo.md", raw)
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).
		Should(it.Equal(kernel.RuleIdentityQuoting, out[0].Rule)).
		Should(it.Equal(2, out[0].Line)).
		Should(it.String(out[0].Message).Contain("@id"))
}

func TestCheckIdentityKeyQuotingUnquotedTypeReportsViolation(t *testing.T) {
	raw := []byte("---\n\"@id\": foo\n@type: Source\n---\n")
	out := checkIdentityKeyQuoting(core.Node{}, "Source/foo.md", raw)
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).
		Should(it.Equal(kernel.RuleIdentityQuoting, out[0].Rule)).
		Should(it.String(out[0].Message).Contain("@type"))
}

func TestCheckIdentityKeyQuotingMissingKeyDistinctMessage(t *testing.T) {
	raw := []byte("---\n\"@id\": foo\n---\n")
	out := checkIdentityKeyQuoting(core.Node{}, "Source/foo.md", raw)
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).
		Should(it.Equal(kernel.RuleIdentityQuoting, out[0].Rule)).
		Should(it.String(out[0].Message).Contain("missing")).
		Should(it.String(out[0].Message).Contain("@type"))
}

// TestCheckBareIdentityKeysWordingMatchesCoreConstant is the spec 021 D3
// pairing guard: arc lint and internal/core are independent implementations
// of one sentence about a bare identity key, and a user must meet identical
// wording whichever command surfaces the defect. core owns the format string
// (core.ErrIdentityQuoting); this asserts lint's own message is that string
// applied to the offending key.
func TestCheckBareIdentityKeysWordingMatchesCoreConstant(t *testing.T) {
	raw := []byte("---\n@id: foo\n@type: Source\n---\n")
	out := checkBareIdentityKeys("Source/foo.md", raw)

	it.Then(t).Must(it.Equal(2, len(out)))
	for i, key := range []string{"@id", "@type"} {
		it.Then(t).Should(it.Equal(
			fmt.Sprintf(string(core.ErrIdentityQuoting), key),
			out[i].Message,
		))
	}
}
