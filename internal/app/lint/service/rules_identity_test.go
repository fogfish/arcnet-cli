//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import (
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

func TestCheckIdentityCharsetLegalIdentityNoViolation(t *testing.T) {
	node := core.Node{Type: "Source", ID: "rescorla-2026-tls13"}
	out := checkIdentityCharset(node, "Source/rescorla-2026-tls13.md", []byte("---\n\"@id\": rescorla-2026-tls13\n---\n"))
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckIdentityCharsetSingleOffendingCharacter(t *testing.T) {
	node := core.Node{Type: "Entity", ID: "Handshake/Protocol"}
	out := checkIdentityCharset(node, "Entity/Handshake_Protocol.md", []byte("---\n\"@id\": \"Handshake/Protocol\"\n---\n"))

	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).
		Should(it.Equal(kernel.RuleIdentityCharset, out[0].Rule)).
		Should(it.String(out[0].Message).Contain(`identity "Handshake/Protocol"`)).
		Should(it.String(out[0].Message).Contain(`"/" at position 10`))
}

// data-model.md §4: multiple offending characters are all named, not only
// the first.
func TestCheckIdentityCharsetMultipleOffendingCharactersAllNamed(t *testing.T) {
	node := core.Node{Type: "Source", ID: "v1.3: Handshake/Protocol"}
	out := checkIdentityCharset(node, "Source/x.md", []byte("---\n\"@id\": \"v1.3: Handshake/Protocol\"\n---\n"))

	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).
		Should(it.String(out[0].Message).Contain(`"." at position 3`)).
		Should(it.String(out[0].Message).Contain(`":" at position 5`)).
		Should(it.String(out[0].Message).Contain(`"/" at position 16`))
}

func TestCheckSchemaIdentityCharsetLegalTypesAndPredicatesNoViolation(t *testing.T) {
	index := core.Index{
		Types:      map[string]core.TypeDef{"Entity": {}},
		Predicates: map[string]core.PredicateDef{"mentions": {}},
	}
	out := checkSchemaIdentityCharset(index)
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckSchemaIdentityCharsetTypeNameViolation(t *testing.T) {
	index := core.Index{
		Types: map[string]core.TypeDef{"Bad/Type": {}},
	}
	out := checkSchemaIdentityCharset(index)

	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).
		Should(it.Equal(kernel.RuleIdentityCharset, out[0].Rule)).
		Should(it.Equal("_schema/Class/Bad/Type.md", out[0].Path)).
		Should(it.Equal(0, out[0].Line)).
		Should(it.String(out[0].Message).Contain(`"/" at position 4`))
}

func TestCheckSchemaIdentityCharsetPredicateNameViolation(t *testing.T) {
	index := core.Index{
		Predicates: map[string]core.PredicateDef{"bad:predicate": {}},
	}
	out := checkSchemaIdentityCharset(index)

	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).
		Should(it.Equal(kernel.RuleIdentityCharset, out[0].Rule)).
		Should(it.Equal("_schema/Property/bad:predicate.md", out[0].Path)).
		Should(it.Equal(0, out[0].Line))
}
