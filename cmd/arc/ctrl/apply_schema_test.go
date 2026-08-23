//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package ctrl

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/schema/kernel"
	"github.com/fogfish/arcnet-cli/internal/bios"
	"github.com/fogfish/arcnet-cli/internal/core"
)

// newSchemaGraph builds a real, git-committed graph root via a real
// sut(NewInitCmd(), nil) call in a temp chdir'd directory — the same
// no-mock-VCS convention this package's own init_test.go already
// establishes — and returns the graph root directory.
func newSchemaGraph(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	chdir(t, dir)

	out, err := sut(NewInitCmd(), []string{})
	it.Then(t).ShouldNot(it.Error(out, err))
	return dir
}

func writeSchemaPatchFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	it.Then(t).Should(it.Nil(os.WriteFile(path, []byte(content), 0o644)))
	return path
}

func decodeApplySchemaJSON(t *testing.T, out string) kernel.ApplySchemaResult {
	t.Helper()
	var r kernel.ApplySchemaResult
	it.Then(t).Should(it.Nil(json.Unmarshal([]byte(out), &r)))
	return r
}

const propertyOnlySchemaPatch = `---
"@type": patch
document: acme-extension-schema
published: 2026-07-15
title: Acme extension vocabulary
---
# Property

## acmeWeight
` + "```yaml" + `
"@id": "acmeWeight"
"@type": Property
role: meta
merge: fillIfEmpty
` + "```" + `

The item's weight in kilograms.
`

const propertyOnlySchemaPatchUpdated = `---
"@type": patch
document: acme-extension-schema
published: 2026-07-15
title: Acme extension vocabulary
---
# Property

## acmeWeight
` + "```yaml" + `
"@id": "acmeWeight"
"@type": Property
role: meta
merge: fillIfEmpty
` + "```" + `

Measured in kilograms (SI).
`

const classOnlySchemaPatch = `---
"@type": patch
document: acme-extension-schema
published: 2026-07-15
title: Acme extension vocabulary
---
# Class

## Widget
` + "```yaml" + `
"@id": "Widget"
"@type": Class
merge: union
` + "```" + `

A physical item tracked by the Acme extension.
`

const classOnlySchemaPatchNoMerge = `---
"@type": patch
document: acme-extension-schema
published: 2026-07-15
title: Acme extension vocabulary
---
# Class

## Hypothesis
` + "```yaml" + `
"@id": "Hypothesis"
"@type": Class
` + "```" + `

A conclusion distilled from sources, carrying no whole-node merge field.
`

const mixedSchemaPatch = `---
"@type": patch
document: acme-extension-schema
published: 2026-07-15
title: Acme extension vocabulary
---
# Property

## acmeWeight
` + "```yaml" + `
"@id": "acmeWeight"
"@type": Property
role: meta
merge: fillIfEmpty
` + "```" + `

The item's weight in kilograms.

# Class

## Widget
` + "```yaml" + `
"@id": "Widget"
"@type": Class
merge: union
` + "```" + `

A physical item tracked by the Acme extension.

- required:: [[acmeWeight]]
`

const mixedValidInvalidSchemaPatch = `---
"@type": patch
document: acme-extension-schema
published: 2026-07-15
title: Acme extension vocabulary
---
# Property

## acmeWeight
` + "```yaml" + `
"@id": "acmeWeight"
"@type": Property
role: meta
merge: fillIfEmpty
` + "```" + `

The item's weight in kilograms.

# Class

## Widget
` + "```yaml" + `
"@id": "Widget"
"@type": Class
merge: union
` + "```" + `

A physical item tracked by the Acme extension.

# entity

## Acme Corp
` + "```yaml" + `
"@id": "Acme Corp"
"@type": Entity
category: [independent, social, continuant, collection]
` + "```" + `

The company behind the extension.
`

const entityOnlySchemaPatch = `---
"@type": patch
document: acme-corp-note
published: 2026-07-15
title: Acme Corp
---
# entity

## Acme Corp
` + "```yaml" + `
"@id": "Acme Corp"
"@type": Entity
category: [independent, social, continuant, collection]
` + "```" + `

The company behind the extension.
`

const timelineOnlySchemaPatch = `---
"@type": patch
document: acme-extension-schema
published: 2026-07-15
title: Acme extension vocabulary
---
# timeline

## 2026
` + "```yaml" + `
"@id": "2026"
"@type": Timeline
` + "```" + `

Yearly index.
`

// arc apply schema <patch.md>
// spec 018 US1 Acceptance Scenario 1: a Property-only patch creates a
// predicate definition for each Property node it carries.
func TestApplySchemaCreatesPredicateFromPropertyOnlyPatch(t *testing.T) {
	dir := newSchemaGraph(t)
	patch := writeSchemaPatchFile(t, dir, "ext.schema.md", propertyOnlySchemaPatch)

	out, err := sut(NewApplySchemaCmd(), []string{patch})

	it.Then(t).ShouldNot(it.Error(out, err))
	assertIsFile(t, filepath.Join(dir, "_schema", "Property", "acmeWeight.md"))
	it.Then(t).Should(it.String(out).Contain("+1 predicate"))
}

// arc apply schema <patch.md>
// spec 018 US1 Acceptance Scenario 2: a Class-only patch creates a type
// definition for each Class node it carries.
func TestApplySchemaCreatesTypeFromClassOnlyPatch(t *testing.T) {
	dir := newSchemaGraph(t)
	patch := writeSchemaPatchFile(t, dir, "ext.schema.md", classOnlySchemaPatch)

	out, err := sut(NewApplySchemaCmd(), []string{patch})

	it.Then(t).ShouldNot(it.Error(out, err))
	assertIsFile(t, filepath.Join(dir, "_schema", "Class", "Widget.md"))
	it.Then(t).Should(it.String(out).Contain("+1 type"))
}

// arc apply schema <patch.md>
// spec 018 US1 Acceptance Scenario 2 / SC-008 (Bugfix 018/BUG-001): a Class
// section carrying no whole-node "merge" field still succeeds and creates
// the type definition — the field has no effect on reconciliation (spec
// 012 FR-015/FR-020) and MUST NOT be required, unlike a Property's own
// "merge" field which remains mandatory.
func TestApplySchemaCreatesTypeFromClassOnlyPatchWithNoMergeField(t *testing.T) {
	dir := newSchemaGraph(t)
	patch := writeSchemaPatchFile(t, dir, "ext.schema.md", classOnlySchemaPatchNoMerge)

	out, err := sut(NewApplySchemaCmd(), []string{patch})

	it.Then(t).ShouldNot(it.Error(out, err))
	assertIsFile(t, filepath.Join(dir, "_schema", "Class", "Hypothesis.md"))
	it.Then(t).Should(it.String(out).Contain("+1 type"))
}

// arc apply schema <patch.md>
// spec 018 US1 Acceptance Scenario 3: a mixed patch creates both a
// predicate and a type definition in the same run.
func TestApplySchemaCreatesBothFromMixedPatch(t *testing.T) {
	dir := newSchemaGraph(t)
	patch := writeSchemaPatchFile(t, dir, "ext.schema.md", mixedSchemaPatch)

	out, err := sut(NewApplySchemaCmd(), []string{patch})

	it.Then(t).ShouldNot(it.Error(out, err))
	assertIsFile(t, filepath.Join(dir, "_schema", "Property", "acmeWeight.md"))
	assertIsFile(t, filepath.Join(dir, "_schema", "Class", "Widget.md"))

	log := gitOutput(t, dir, "log", "--oneline")
	it.Then(t).Should(it.String(log).Contain("schema(apply):"))
}

// arc apply schema <patch.md> --json
// spec 018 US1 Acceptance Scenario 4: the run reports how many predicate
// and type definitions were created, and how many were merged.
func TestApplySchemaReportsCreatedAndMergedSummary(t *testing.T) {
	dir := newSchemaGraph(t)
	patch := writeSchemaPatchFile(t, dir, "ext.schema.md", propertyOnlySchemaPatch)

	bios.JSON = true
	t.Cleanup(func() { bios.JSON = false })

	out, err := sut(NewApplySchemaCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	result := decodeApplySchemaJSON(t, out)
	it.Then(t).
		Should(it.Equal(1, result.Created["predicate"])).
		Should(it.Equal(0, result.Merged["predicate"]))
	it.Then(t).ShouldNot(it.Equal("", result.CommitHash))

	writeSchemaPatchFile(t, dir, "ext.schema.md", propertyOnlySchemaPatchUpdated)
	out, err = sut(NewApplySchemaCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	result = decodeApplySchemaJSON(t, out)
	it.Then(t).
		Should(it.Equal(0, result.Created["predicate"])).
		Should(it.Equal(1, result.Merged["predicate"]))
}

// arc apply schema arcnet:<name>
// spec 018 US1 Acceptance Scenario 5: an arcnet: reference resolves
// against the official catalog base and imports exactly as a directly
// supplied URL would (research.md D1a's httptest.Server test seam).
func TestApplySchemaArcnetShorthandResolvesAndImports(t *testing.T) {
	dir := newSchemaGraph(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		it.Then(t).Should(it.Equal("/media.schema.md", r.URL.Path))
		_, _ = w.Write([]byte(propertyOnlySchemaPatch))
	}))
	defer srv.Close()

	original := kernel.ArcnetCatalogBaseURL
	kernel.ArcnetCatalogBaseURL = srv.URL + "/"
	t.Cleanup(func() { kernel.ArcnetCatalogBaseURL = original })

	out, err := sut(NewApplySchemaCmd(), []string{"arcnet:media.schema.md"})

	it.Then(t).ShouldNot(it.Error(out, err))
	assertIsFile(t, filepath.Join(dir, "_schema", "Property", "acmeWeight.md"))
}

// arc apply schema <patch.md>
// spec 018 US2 Acceptance Scenario 1: a disallowed node section (entity)
// fails the command and identifies the node's id/type.
func TestApplySchemaRejectsDisallowedNodeType(t *testing.T) {
	dir := newSchemaGraph(t)
	patch := writeSchemaPatchFile(t, dir, "ext.schema.md", entityOnlySchemaPatch)

	out, err := sut(NewApplySchemaCmd(), []string{patch})

	it.Then(t).
		Should(it.Error(out, err).Contain("Acme Corp")).
		Should(it.Error(out, err).Contain("Entity"))
	_, statErr := os.Stat(filepath.Join(dir, "_schema", "Property", "Acme Corp.md"))
	it.Then(t).Should(it.True(os.IsNotExist(statErr)))
}

// arc apply schema <patch.md>
// spec 018 US2 Acceptance Scenario 2: a patch mixing valid Property/Class
// sections with one disallowed section writes none of the patch's
// definitions — not even the otherwise-valid ones.
func TestApplySchemaRejectsMixedValidAndInvalidPatchWritesNothing(t *testing.T) {
	dir := newSchemaGraph(t)
	patch := writeSchemaPatchFile(t, dir, "ext.schema.md", mixedValidInvalidSchemaPatch)

	out, err := sut(NewApplySchemaCmd(), []string{patch})

	it.Then(t).Should(it.Error(out, err).Contain("Acme Corp"))
	_, weightErr := os.Stat(filepath.Join(dir, "_schema", "Property", "acmeWeight.md"))
	it.Then(t).Should(it.True(os.IsNotExist(weightErr)))
	_, widgetErr := os.Stat(filepath.Join(dir, "_schema", "Class", "Widget.md"))
	it.Then(t).Should(it.True(os.IsNotExist(widgetErr)))

	status := gitOutput(t, dir, "status", "--short", "--", "_schema")
	it.Then(t).Should(it.Equal("", status))
}

// arc apply schema <patch.md>
// spec 018 US2 Acceptance Scenario 3: a reserved graph-structure kind
// (timeline) is treated as disallowed with no special-casing.
func TestApplySchemaRejectsTimelineKind(t *testing.T) {
	dir := newSchemaGraph(t)
	patch := writeSchemaPatchFile(t, dir, "ext.schema.md", timelineOnlySchemaPatch)

	out, err := sut(NewApplySchemaCmd(), []string{patch})

	it.Then(t).
		Should(it.Error(out, err).Contain("2026")).
		Should(it.Error(out, err).Contain("Timeline"))
}

// arc apply schema <patch.md>
// spec 018 US3 Acceptance Scenario 1: re-applying a patch with a changed
// field merges it into the existing predicate per its declared merge
// behavior, rather than duplicating a new document.
func TestApplySchemaReapplyMergesChangedField(t *testing.T) {
	dir := newSchemaGraph(t)
	patch := writeSchemaPatchFile(t, dir, "ext.schema.md", propertyOnlySchemaPatch)

	out, err := sut(NewApplySchemaCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	writeSchemaPatchFile(t, dir, "ext.schema.md", propertyOnlySchemaPatchUpdated)
	out, err = sut(NewApplySchemaCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	content, rerr := os.ReadFile(filepath.Join(dir, "_schema", "Property", "acmeWeight.md"))
	it.Then(t).Should(it.Nil(rerr))
	it.Then(t).
		Should(it.String(string(content)).Contain("kilograms.")).
		Should(it.String(string(content)).Contain("Measured in kilograms (SI)."))
}

const sourceOptionalOnlySchemaPatch = `---
"@type": patch
document: acme-extension-schema
published: 2026-07-15
title: Acme extension vocabulary
---
# Class

## Source
` + "```yaml" + `
"@id": "Source"
"@type": Class
` + "```" + `

**Optional**
- optional:: [[proposes]]
- optional:: [[raises]]
`

// arc apply schema <patch.md>
// spec 018 US3 Acceptance Scenario 3 (Bugfix BUG-002): re-declaring the
// already-registered built-in Source type with only a new Optional
// predicate, and no description/merge in the section, merges successfully
// instead of being rejected as missing a mandatory field the existing
// document already supplies.
func TestApplySchemaMergesOptionalPredicateIntoExistingTypeOmittingDescription(t *testing.T) {
	dir := newSchemaGraph(t)
	patch := writeSchemaPatchFile(t, dir, "ext.schema.md", sourceOptionalOnlySchemaPatch)

	out, err := sut(NewApplySchemaCmd(), []string{patch})

	it.Then(t).ShouldNot(it.Error(out, err))

	content, rerr := os.ReadFile(filepath.Join(dir, "_schema", "Class", "Source.md"))
	it.Then(t).Should(it.Nil(rerr))
	it.Then(t).
		Should(it.String(string(content)).Contain("optional:: [[proposes]]")).
		Should(it.String(string(content)).Contain("optional:: [[raises]]")).
		Should(it.String(string(content)).Contain("provenance origin"))
}

// arc apply schema <patch.md>
// spec 018 US3 Acceptance Scenario 2: re-applying an unchanged patch
// completes without reporting any created/merged changes, and produces no
// commit.
func TestApplySchemaReapplyWithNoChangesReportsZero(t *testing.T) {
	dir := newSchemaGraph(t)
	patch := writeSchemaPatchFile(t, dir, "ext.schema.md", classOnlySchemaPatch)

	out, err := sut(NewApplySchemaCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	before := gitOutput(t, dir, "log", "--oneline")

	bios.JSON = true
	t.Cleanup(func() { bios.JSON = false })
	out, err = sut(NewApplySchemaCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	result := decodeApplySchemaJSON(t, out)
	it.Then(t).
		Should(it.Equal(0, result.Created["type"])).
		Should(it.Equal(0, result.Merged["type"])).
		Should(it.Equal("", result.CommitHash))

	after := gitOutput(t, dir, "log", "--oneline")
	it.Then(t).Should(it.Equal(before, after))
}

// ---------------------------------------------------------------------------
// spec 021 — patch manifest identity ("@type": patch)
// ---------------------------------------------------------------------------

// arc apply schema <patch.md> | <url> | arcnet:<name>
// spec 021 US1 Acceptance Scenario 4: a schema patch declaring itself with
// the quoted "@type": patch key is recognized identically from all three
// source forms. Remote ARCNET profiles are published as conformant patches,
// so this is the change that makes them loadable at all.
func TestApplySchemaRecognizesTypeKeyFromEverySource(t *testing.T) {
	t.Run("local file", func(t *testing.T) {
		dir := newSchemaGraph(t)
		patch := writeSchemaPatchFile(t, dir, "ext.schema.md", propertyOnlySchemaPatch)

		out, err := sut(NewApplySchemaCmd(), []string{patch})

		it.Then(t).ShouldNot(it.Error(out, err))
		assertIsFile(t, filepath.Join(dir, "_schema", "Property", "acmeWeight.md"))
	})

	t.Run("url", func(t *testing.T) {
		dir := newSchemaGraph(t)

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(propertyOnlySchemaPatch))
		}))
		defer srv.Close()

		out, err := sut(NewApplySchemaCmd(), []string{srv.URL + "/media.schema.md"})

		it.Then(t).ShouldNot(it.Error(out, err))
		assertIsFile(t, filepath.Join(dir, "_schema", "Property", "acmeWeight.md"))
	})

	t.Run("arcnet catalog reference", func(t *testing.T) {
		dir := newSchemaGraph(t)

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(propertyOnlySchemaPatch))
		}))
		defer srv.Close()

		original := kernel.ArcnetCatalogBaseURL
		kernel.ArcnetCatalogBaseURL = srv.URL + "/"
		t.Cleanup(func() { kernel.ArcnetCatalogBaseURL = original })

		out, err := sut(NewApplySchemaCmd(), []string{"arcnet:media.schema.md"})

		it.Then(t).ShouldNot(it.Error(out, err))
		assertIsFile(t, filepath.Join(dir, "_schema", "Property", "acmeWeight.md"))
	})
}

// arc apply schema legacy.schema.md
// spec 021 FR-003/FR-009: the retired key is refused with the identical
// recognition rule arc apply uses, and the message carries the source it was
// read from.
func TestApplySchemaLegacyKindRefused(t *testing.T) {
	dir := newSchemaGraph(t)
	legacy := strings.Replace(propertyOnlySchemaPatch, `"@type": patch`, "kind: patch", 1)
	patch := writeSchemaPatchFile(t, dir, "legacy.schema.md", legacy)

	_, err := sut(NewApplySchemaCmd(), []string{patch})

	it.Then(t).Must(it.True(errors.Is(err, core.ErrManifestLegacyKind)))
	it.Then(t).Should(it.String(err.Error()).Contain("legacy.schema.md"))
}

// ---------------------------------------------------------------------------
// specs/022-reference-type-folders — ARCNET-CORE v0.11
// ---------------------------------------------------------------------------

// arc apply schema ext.schema.md
// spec.md US3 Acceptance Scenario 5: an applied schema patch files its
// predicate node under _schema/Property/ and its type node under
// _schema/Class/ — the same two type folders arc init seeds into, since
// both read the one pair of path constants (contract C2).
//
// Asserted against the basenames the filesystem reports rather than a
// per-path os.Stat: on APFS a stat would answer yes to "_schema/property"
// and "_schema/class" too (contract C7).
func TestApplySchemaWritesIntoTypeNamedSchemaFolders(t *testing.T) {
	dir := newSchemaGraph(t)
	patch := writeSchemaPatchFile(t, dir, "ext.schema.md", mixedSchemaPatch)

	out, err := sut(NewApplySchemaCmd(), []string{patch})
	it.Then(t).ShouldNot(it.Error(out, err))

	it.Then(t).
		Should(it.Seq(schemaDirEntryNames(t, dir, "Property")).Contain("acmeWeight.md")).
		Should(it.Seq(schemaDirEntryNames(t, dir, "Class")).Contain("Widget.md"))

	// The retired folders must not reappear alongside them.
	schemaChildren := schemaDirEntryNames(t, dir, "")
	for _, retired := range []string{"predicates", "types"} {
		for _, got := range schemaChildren {
			it.Then(t).Should(it.True(got != retired))
		}
	}
}

// schemaDirEntryNames lists the basenames stored under _schema/<child>, or
// under _schema/ itself when child is empty. A missing directory yields no
// names, so an assertion about absence reads the same either way.
func schemaDirEntryNames(t *testing.T, dir, child string) []string {
	t.Helper()
	path := filepath.Join(dir, "_schema")
	if child != "" {
		path = filepath.Join(path, child)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// arc apply schema acme.schema.md (twice)
// Scenario 5 from spec.md US1 (specs/023-core-vocabulary-conformance): a
// predicate or type definition's own descriptive prose is byte-identical
// after two applies (FR-013, FR-015).
//
// This scenario lives here rather than in cmd/arc/graph/apply_test.go with
// the rest of US1 because "a patch carrying a predicate or type definition"
// is precisely what `arc apply schema` ingests — the other five scenarios
// are about content nodes.
//
// Note which predicate is actually under test: a Property/Class document's
// description decodes into the Texts key "text", not "description" (the
// parser has no Property/Class case of its own — schema/service's own
// descriptionKey names that fact). "text" declares append, so the guarantee
// here comes from mergeText's near-duplicate guard rather than from
// firstWriteWin. The assertion is the same either way, and it is the one
// the user actually observes.
func TestApplySchemaTwiceLeavesDescriptionByteIdentical(t *testing.T) {
	dir := newSchemaGraph(t)

	first := writeSchemaPatchFile(t, dir, "acme.schema.md", propertyOnlySchemaPatch)
	out, err := sut(NewApplySchemaCmd(), []string{first})
	it.Then(t).ShouldNot(it.Error(out, err))

	path := filepath.Join(dir, "_schema", "Property", "acmeWeight.md")
	before := readGraphFile(t, path)
	it.Then(t).Should(it.True(strings.TrimSpace(before) != ""))

	// A second patch file carrying the same definition — a re-import of the
	// same extension vocabulary, which is how this reaches the merge path.
	second := writeSchemaPatchFile(t, dir, "acme-again.schema.md", propertyOnlySchemaPatch)
	out, err = sut(NewApplySchemaCmd(), []string{second})
	it.Then(t).ShouldNot(it.Error(out, err))

	it.Then(t).Should(it.Equal(before, readGraphFile(t, path)))
}
