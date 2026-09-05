//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package service_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/schema/kernel"
	"github.com/fogfish/arcnet-cli/internal/app/schema/service"
)

// update regenerates the golden tree instead of asserting against it:
//
//	go test ./internal/app/schema/service -run TestSeedGolden -update
//
// A diff in testdata/golden/schema/ is a reviewable change to what every new
// graph contains (contract C2.1) and MUST be read as such in review, never
// regenerated reflexively.
var update = flag.Bool("update", false, "regenerate testdata/golden/schema/ from Seed() output")

// goldenRoot is the package-relative root of the snapshot. Every file under
// it is named by the very path Seed() keys it with, so the tree on disk is
// the tree arc init writes.
const goldenRoot = "testdata/golden/schema"

// TestSeedGolden fixes Seed()'s output byte-for-byte (contract C2.1). It is
// the regression net for the whole built-in vocabulary: CORE-FIX.md §7
// records that two findings survived four spec revisions because the seeded
// vocabulary was only ever reviewed as diffs of Go map literals.
func TestSeedGolden(t *testing.T) {
	seed := service.Seed()

	if *update {
		it.Then(t).Should(it.Nil(os.RemoveAll(goldenRoot)))
		for path, content := range seed {
			file := filepath.Join(goldenRoot, filepath.FromSlash(path))
			it.Then(t).Should(it.Nil(os.MkdirAll(filepath.Dir(file), 0o755)))
			it.Then(t).Should(it.Nil(os.WriteFile(file, content, 0o644)))
		}
		t.Log("regenerated", goldenRoot, "with", len(seed), "documents")
		return
	}

	golden, err := readGoldenTree()
	it.Then(t).Should(it.Nil(err))

	it.Then(t).Should(it.Seq(sortedKeys(seed)).Equal(sortedKeys(golden)...))

	for _, path := range sortedKeys(seed) {
		want, ok := golden[path]
		if !ok {
			continue
		}
		if !bytes.Equal(want, seed[path]) {
			t.Errorf("%s differs from its golden snapshot\n--- golden\n%s\n--- seed\n%s", path, want, seed[path])
		}
	}
}

// readGoldenTree reads the snapshot back keyed exactly as Seed() keys it.
func readGoldenTree() (map[string][]byte, error) {
	out := map[string][]byte{}
	err := filepath.Walk(goldenRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(goldenRoot, path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = content
		return nil
	})
	return out, err
}

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ---------------------------------------------------------------------------
// Contract C2.2 — invariants over the seeded tree
//
// Asserted directly against Seed() output, independent of the golden files,
// so a careless `-update` cannot mask a regression.
// ---------------------------------------------------------------------------

// TestSeedTreeIsClosedUnderPredicateReference is contract C2.2c: every
// predicate named in any Class document's required::/optional::/subClassOf::
// bullets has its own Property document.
//
// This is the invariant that makes SC-003 hold — a freshly initialized graph
// lints clean — and it is not self-evident: before this feature the tool
// wrote subClassOf:: bullets onto five seeded Class documents while Class
// itself never declared the predicate, a typeOptional violation the seeded
// tree inflicted on itself.
func TestSeedTreeIsClosedUnderPredicateReference(t *testing.T) {
	seed := service.Seed()

	properties := map[string]bool{}
	types := map[string]bool{}
	for path := range seed {
		switch {
		case strings.HasPrefix(path, kernel.PredicatesDir+"/"):
			properties[basename(path, kernel.PredicatesDir)] = true
		case strings.HasPrefix(path, kernel.TypesDir+"/"):
			types[basename(path, kernel.TypesDir)] = true
		}
	}

	bullet := regexp.MustCompile(`(?m)^- (required|optional|subClassOf):: \[\[([^\]]+)\]\]`)

	for path, raw := range seed {
		if !strings.HasPrefix(path, kernel.TypesDir+"/") {
			continue
		}
		for _, m := range bullet.FindAllStringSubmatch(string(raw), -1) {
			predicate, target := m[1], m[2]

			// A subClassOf:: bullet names a TYPE, not a predicate — the
			// predicate it references is "subClassOf" itself.
			if predicate == "subClassOf" {
				if !types[target] {
					t.Errorf("%s declares subClassOf [[%s]], which has no _schema/Class/ document", path, target)
				}
				if !properties["subClassOf"] {
					t.Errorf("%s uses subClassOf, which has no _schema/Property/ document", path)
				}
				continue
			}

			if !properties[target] {
				t.Errorf("%s declares %s [[%s]], which has no _schema/Property/ document", path, predicate, target)
			}
			if !properties[predicate] {
				t.Errorf("%s uses the %q predicate, which has no _schema/Property/ document", path, predicate)
			}
		}
	}
}

func basename(path, dir string) string {
	return strings.TrimSuffix(strings.TrimPrefix(path, dir+"/"), ".md")
}

// TestSeedRegistersMetadataPredicates is contract C2.2d / FR-011: the three
// §10.2 metadata predicates the seeded vocabulary omitted entirely, each
// with the role and merge the specification assigns it.
func TestSeedRegistersMetadataPredicates(t *testing.T) {
	seed := service.Seed()

	for _, name := range []string{"author", "about", "genre"} {
		raw, ok := seed[kernel.PredicatesDir+"/"+name+".md"]
		if !ok {
			t.Errorf("_schema/Property/%s.md is not seeded", name)
			continue
		}
		content := string(raw)
		it.Then(t).
			Should(it.String(content).Contain("role: meta")).
			Should(it.String(content).Contain("merge: union"))
	}
}

// alignedTerms is every predicate ARCNET-CORE aligns to a standard
// vocabulary term, written out as a literal. Deriving it from
// CorePredicateDefs would make the assertion agree with the table by
// construction — the whole point of SC-008 is that six of these declared
// none, and nothing in the table's own shape revealed it.
// "authors" (plural) is retired under CORE 0.12 (BUG-001/FR-031) — "author"
// alone carries the schema:author alignment now.
var alignedTerms = map[string]string{
	"title":    "schema:title",
	"abstract": "schema:abstract",
	"url":      "schema:url",
	"doi":      "schema:doi",
	"aliases":  "skos:altLabel",
	"author":   "schema:author",
}

// TestSeedDeclaresEveryAssignedAlignment is SC-008: zero seeded predicates
// omit an alignment the specification assigns them.
func TestSeedDeclaresEveryAssignedAlignment(t *testing.T) {
	seed := service.Seed()

	for name, term := range alignedTerms {
		raw, ok := seed[kernel.PredicatesDir+"/"+name+".md"]
		if !ok {
			t.Errorf("_schema/Property/%s.md is not seeded", name)
			continue
		}
		if !strings.Contains(string(raw), "aligned: "+term) {
			t.Errorf("_schema/Property/%s.md does not declare aligned: %s", name, term)
		}
	}
}

// TestSeedProsePredicatesDeclareAppend is contract C2.2e, and
// TestSeedCitationPredicateDeclaresUnion is C2.2f — both asserted over the
// rendered documents, which is what a graph actually receives.
func TestSeedProsePredicatesDeclareAppend(t *testing.T) {
	seed := service.Seed()

	// "definition" is retired under CORE 0.12 (BUG-001/FR-030); Entity's
	// leading prose is now "text", which is deliberately MergeAppend, not
	// firstWriteWin (see schema.go's CorePredicateDefs comment). "abstract"
	// and "description" declare append for the same reason, so no prose
	// predicate is first-fixed any longer — the seeded documents are
	// asserted here so that stays true of what a graph receives, not only
	// of the in-memory table. "relevance"/"notes" are retired outright
	// (spec 023 BUG-002) and no longer seeded at all.
	for _, name := range []string{"abstract", "description", "text"} {
		raw, ok := seed[kernel.PredicatesDir+"/"+name+".md"]
		it.Then(t).Should(it.True(ok))
		it.Then(t).
			Should(it.String(string(raw)).Contain("merge: append")).
			ShouldNot(it.String(string(raw)).Contain("merge: firstWriteWin"))
	}
}

func TestSeedCitationPredicateDeclaresUnion(t *testing.T) {
	seed := service.Seed()

	raw, ok := seed[kernel.PredicatesDir+"/cites.md"]
	it.Then(t).Should(it.True(ok))
	it.Then(t).
		Should(it.String(string(raw)).Contain("merge: union")).
		ShouldNot(it.String(string(raw)).Contain("merge: append"))
}

// TestSeedRetiredPredicatesAreGoneEntirely is SC-009 ([BUG-001](../../../../specs/023-core-vocabulary-conformance/bugs/BUG-001.md)):
// CORE 0.12 retires six predicate names outright (five renamed to an
// already-registered replacement, one — "granularity" — dropped with no
// replacement at all). None of the six may be seeded as its own Property
// document, and none may appear in any seeded type's Required/Optional
// bullets — a plain breaking change (FR-037), matching
// TestAstExactlyMergeOpValues' exhaustive-set style for the merge
// vocabulary (contract C1.1). "notes" (the seventh predicate BUG-001
// touches) is deliberately excluded here: it stays registered and is still
// legitimately seeded on Reference — only Entity/Resource's Optional lists
// lose it (FR-033), asserted separately by
// TestSeedResourceDocumentCarriesNoExternalWorkPredicate and
// TestCoreTypeDefsOptionalListsIncludeCrossCuttingPredicates.
func TestSeedRetiredPredicatesAreGoneEntirely(t *testing.T) {
	seed := service.Seed()

	// "notes"/"relevance" were excluded here until spec 023 BUG-002: T072's
	// own note said including them would have been wrong, since at the
	// time "notes" was retired only from Entity/Resource's Optional lists
	// (not fully retired) and "relevance" wasn't retired at all yet. Both
	// are now retired outright — no type declares either, and neither is
	// seeded as a Property document — so both belong in this exhaustive
	// absence check.
	retired := []string{"definition", "authors", "year", "ref", "status", "granularity", "notes", "relevance"}

	for _, name := range retired {
		_, ok := seed[kernel.PredicatesDir+"/"+name+".md"]
		it.Then(t).Should(it.True(!ok))
	}

	for path, raw := range seed {
		if !strings.HasPrefix(path, kernel.TypesDir+"/") {
			continue
		}
		content := string(raw)
		for _, name := range retired {
			it.Then(t).
				ShouldNot(it.String(content).Contain("required:: [[" + name + "]]")).
				ShouldNot(it.String(content).Contain("optional:: [[" + name + "]]"))
		}
	}
}
