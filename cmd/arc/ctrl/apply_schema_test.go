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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/fogfish/it/v2"

	"github.com/fogfish/arcnet-cli/internal/app/schema/kernel"
	"github.com/fogfish/arcnet-cli/internal/bios"
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
kind: patch
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
kind: patch
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
kind: patch
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

const mixedSchemaPatch = `---
kind: patch
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
kind: patch
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
"@type": entity
category: [independent, social, continuant, collection]
` + "```" + `

The company behind the extension.
`

const entityOnlySchemaPatch = `---
kind: patch
document: acme-corp-note
published: 2026-07-15
title: Acme Corp
---
# entity

## Acme Corp
` + "```yaml" + `
"@id": "Acme Corp"
"@type": entity
category: [independent, social, continuant, collection]
` + "```" + `

The company behind the extension.
`

const timelineOnlySchemaPatch = `---
kind: patch
document: acme-extension-schema
published: 2026-07-15
title: Acme extension vocabulary
---
# timeline

## 2026
` + "```yaml" + `
"@id": "2026"
"@type": timeline
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
	assertIsFile(t, filepath.Join(dir, "_schema", "predicates", "acmeWeight.md"))
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
	assertIsFile(t, filepath.Join(dir, "_schema", "types", "Widget.md"))
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
	assertIsFile(t, filepath.Join(dir, "_schema", "predicates", "acmeWeight.md"))
	assertIsFile(t, filepath.Join(dir, "_schema", "types", "Widget.md"))

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
	assertIsFile(t, filepath.Join(dir, "_schema", "predicates", "acmeWeight.md"))
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
		Should(it.Error(out, err).Contain("entity"))
	_, statErr := os.Stat(filepath.Join(dir, "_schema", "predicates", "Acme Corp.md"))
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
	_, weightErr := os.Stat(filepath.Join(dir, "_schema", "predicates", "acmeWeight.md"))
	it.Then(t).Should(it.True(os.IsNotExist(weightErr)))
	_, widgetErr := os.Stat(filepath.Join(dir, "_schema", "types", "Widget.md"))
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
		Should(it.Error(out, err).Contain("timeline"))
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

	content, rerr := os.ReadFile(filepath.Join(dir, "_schema", "predicates", "acmeWeight.md"))
	it.Then(t).Should(it.Nil(rerr))
	it.Then(t).
		Should(it.String(string(content)).Contain("kilograms.")).
		Should(it.String(string(content)).Contain("Measured in kilograms (SI)."))
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
