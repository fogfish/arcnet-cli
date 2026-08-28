//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package graph

import (
	"errors"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/graph/service"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// statementFor finds the one f.Statements entry whose Predicate matches
// name, failing the test if there is none or more than one — every
// --type/--tag/--attr repeat-name combination this file exercises lowers to
// exactly one statement per distinct predicate name (research.md D4).
func statementFor(t *testing.T, f core.Filter, name string) core.Statement {
	t.Helper()
	var found []core.Statement
	for _, s := range f.Statements {
		if s.Predicate.Match(name) {
			found = append(found, s)
		}
	}
	it.Then(t).Should(it.Equal(1, len(found)))
	return found[0]
}

func TestOptsFilterBuildParsesExactAttrValue(t *testing.T) {
	opts := optsFilter{attr: []string{"status=mature"}}

	f, err := opts.build()

	it.Then(t).Should(it.Nil(err))
	s := statementFor(t, f, "status")
	it.Then(t).Should(it.True(s.Target.Match("mature")))
}

func TestOptsFilterBuildParsesPatternAttrValue(t *testing.T) {
	opts := optsFilter{attr: []string{`title~=^TLS`}}

	f, err := opts.build()

	it.Then(t).Should(it.Nil(err))
	s := statementFor(t, f, "title")
	it.Then(t).Should(it.True(s.Target.Match("TLS 1.3")))
}

func TestOptsFilterBuildRejectsMalformedAttrValue(t *testing.T) {
	opts := optsFilter{attr: []string{"status"}}

	_, err := opts.build()

	it.Then(t).Should(it.True(errors.Is(err, service.ErrInvalidAttrFlag)))
}

func TestOptsFilterBuildComposesTypeTagAttr(t *testing.T) {
	opts := optsFilter{
		typ:  []string{"Entity", "Source"},
		tag:  []string{"cryptography"},
		attr: []string{"status=mature"},
	}

	f, err := opts.build()

	it.Then(t).Should(it.Nil(err))
	typeStmt := statementFor(t, f, "type")
	it.Then(t).
		Should(it.True(typeStmt.Target.Match("Entity"))).
		Should(it.True(typeStmt.Target.Match("Source"))).
		ShouldNot(it.True(typeStmt.Target.Match("Resource")))

	tagStmt := statementFor(t, f, "tags")
	it.Then(t).Should(it.True(tagStmt.Target.Match("cryptography")))

	statusStmt := statementFor(t, f, "status")
	it.Then(t).Should(it.True(statusStmt.Target.Match("mature")))
}

// research.md D10: --predicate lowers to one traversal-constraint
// statement (Source/Target both wildcard), OR'd across every repeat.
func TestOptsFilterBuildPredicateLowersToTraversalConstraint(t *testing.T) {
	opts := optsFilter{predicate: []string{"cites", "mentions"}}

	f, err := opts.build()

	it.Then(t).Should(it.Nil(err))
	it.Then(t).Should(it.Equal(1, len(f.Statements)))
	s := f.Statements[0]
	it.Then(t).
		Should(it.True(s.IsTraversalConstraint())).
		Should(it.True(s.Predicate.Match("cites"))).
		Should(it.True(s.Predicate.Match("mentions"))).
		ShouldNot(it.True(s.Predicate.Match("other")))
}
