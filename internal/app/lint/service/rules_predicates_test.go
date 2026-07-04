//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service

import (
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/adapter/fsys"
	"github.com/fogfish/arcnet-cli/internal/app/lint/kernel"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// predicatesMemStore is a minimal fsys.Store fake, local to this package's
// (internal, non-_test-suffixed) tests, so unexported functions like
// parsePredicateRegistry can be exercised directly.
type predicatesMemStore struct {
	files    map[string]string
	openErrs map[string]error
}

func newMemStoreForPredicates() *predicatesMemStore {
	return &predicatesMemStore{files: map[string]string{}, openErrs: map[string]error{}}
}

type predicatesMemFile struct{ *strings.Reader }

func (f predicatesMemFile) Close() error               { return nil }
func (f predicatesMemFile) Stat() (fs.FileInfo, error) { return nil, nil }

func (s *predicatesMemStore) Open(name string) (fs.File, error) {
	if err, ok := s.openErrs[name]; ok {
		return nil, err
	}
	content, ok := s.files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return predicatesMemFile{strings.NewReader(content)}, nil
}

func (s *predicatesMemStore) Stat(name string) (fs.FileInfo, error)      { return nil, fs.ErrNotExist }
func (s *predicatesMemStore) ReadDir(name string) ([]fs.DirEntry, error) { return nil, nil }
func (s *predicatesMemStore) Create(name string) (fsys.File, error) {
	return nil, errors.New("read-only fake")
}
func (s *predicatesMemStore) Remove(name string) error { return errors.New("read-only fake") }

func TestCheckPredicateCaseValid(t *testing.T) {
	node := core.Node{Edges: []core.Link{{Predicate: "mentions", Target: "X"}}}
	out := checkPredicateCase(node, "x.md", []byte("- mentions:: [[X]]\n"))
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckPredicateCaseInvalid(t *testing.T) {
	node := core.Node{Edges: []core.Link{{Predicate: "Mentions-Bad", Target: "X"}}}
	out := checkPredicateCase(node, "x.md", []byte("- Mentions-Bad:: [[X]]\n"))
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).Should(it.Equal(kernel.RulePredicateCase, out[0].Rule))
}

func TestCheckPredicateCaseDedupSamePredicateTwice(t *testing.T) {
	node := core.Node{Edges: []core.Link{
		{Predicate: "BadOne", Target: "X"},
		{Predicate: "BadOne", Target: "Y"},
	}}
	out := checkPredicateCase(node, "x.md", []byte("- BadOne:: [[X]]\n- BadOne:: [[Y]]\n"))
	it.Then(t).Should(it.Equal(1, len(out)))
}

func TestCheckPredicateRegisteredPresent(t *testing.T) {
	node := core.Node{Edges: []core.Link{{Predicate: "mentions", Target: "X"}}}
	registry := map[string]bool{"mentions": true}
	out := checkPredicateRegistered(node, "x.md", []byte("- mentions:: [[X]]\n"), registry)
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckPredicateRegisteredAbsent(t *testing.T) {
	node := core.Node{Edges: []core.Link{{Predicate: "unregisteredPred", Target: "X"}}}
	out := checkPredicateRegistered(node, "x.md", []byte("- unregisteredPred:: [[X]]\n"), map[string]bool{})
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).Should(it.Equal(kernel.RulePredicateRegistered, out[0].Rule))
}

func TestCheckPredicateFromLinksBlockKey(t *testing.T) {
	node := core.Node{Links: map[string]core.LinkBlock{
		"mentions": {Title: "Mentions", Seq: []core.Link{{Predicate: "mentions", Target: "X"}}},
	}}
	raw := []byte("## Mentions\n- mentions:: [[X]]\n")
	out := checkPredicateRegistered(node, "x.md", raw, map[string]bool{"mentions": true})
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckCitationPredicateValid(t *testing.T) {
	node := core.Node{HRefs: []core.Link{{Predicate: "cites", Target: "RFC 8446"}}}
	out := checkCitationPredicate(node, "x.md", []byte("[cites:: [[RFC 8446]]]\n"))
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestCheckCitationPredicateInvalid(t *testing.T) {
	node := core.Node{HRefs: []core.Link{{Predicate: "randomPredicate", Target: "RFC 8446"}}}
	out := checkCitationPredicate(node, "x.md", []byte("[randomPredicate:: [[RFC 8446]]]\n"))
	it.Then(t).Should(it.Equal(1, len(out)))
	it.Then(t).Should(it.Equal(kernel.RuleCitationPredicate, out[0].Rule))
}

func TestCheckCitationPredicateBareLinkExempt(t *testing.T) {
	node := core.Node{HRefs: []core.Link{{Target: "Widget"}}}
	out := checkCitationPredicate(node, "x.md", []byte("[[Widget]]\n"))
	it.Then(t).Should(it.Equal(0, len(out)))
}

func TestParsePredicateRegistryWellFormed(t *testing.T) {
	s := newMemStoreForPredicates()
	s.files[predicatesPath] = "# Predicates\n\n- `mentions` — a document mentions an entity\n- `cites` — a citation\n"

	registry, err := parsePredicateRegistry(s)

	it.Then(t).Should(it.Nil(err))
	it.Then(t).
		Should(it.True(registry["mentions"])).
		Should(it.True(registry["cites"])).
		Should(it.Equal(2, len(registry)))
}

func TestParsePredicateRegistryAbsentFile(t *testing.T) {
	s := newMemStoreForPredicates()

	registry, err := parsePredicateRegistry(s)

	it.Then(t).
		Should(it.Nil(err)).
		Should(it.Equal(0, len(registry)))
}

func TestParsePredicateRegistryReadFailure(t *testing.T) {
	s := newMemStoreForPredicates()
	s.openErrs[predicatesPath] = errors.New("boom")

	_, err := parsePredicateRegistry(s)

	it.Then(t).ShouldNot(it.Nil(err))
}
