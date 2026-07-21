# Research: CamelCase Node Class Names

## D1 — What "starts with an uppercase letter" means, precisely

**Decision**: A class/type name is CamelCase-compliant iff its first rune satisfies `unicode.IsUpper`. No further constraint on the remaining characters (digits, further case changes are unconstrained).

**Rationale**: Spec Assumptions section already fixes this scope; `unicode.IsUpper` is Unicode-aware or free (spec Edge Cases: accented capitals), matches the existing `internal/app/lint/service.camelCasePattern` precedent's spirit (`^[a-z][a-zA-Z0-9]*$` for predicates) without over-constraining type names to ASCII-only.

**Alternatives considered**: A regex `^[A-Z][a-zA-Z0-9]*$` mirroring `camelCasePattern` exactly — rejected because it silently excludes valid Unicode uppercase letters (Assumptions explicitly requires Unicode-awareness), and a first-rune check is simpler and needs no regex at all for the `arc apply` gate (a regex is still used for the lint per-node/schema check to reuse the existing file's established idiom — see D6).

## D2 — Where the `arc apply` CamelCase gate lives

**Decision**: `internal/core.patchNodeIdentity` (called from `parsePatchBody`, `internal/core/markdown.go:174-209`) stops lowercasing the H1 heading (`strings.ToLower(typeHeading)` at line 190 is deleted) and instead uses `typeHeading` verbatim as the default `typ`. A new unexported helper `isCamelCase(s string) bool` (first-rune `unicode.IsUpper`, false for empty string) gates both the heading text and, when present, the explicit `@type` value (FR-004/FR-005/FR-008). A non-compliant heading or explicit `@type` returns a new `ErrTypeCasing` (see D3) naming the offending value; `parsePatchBody`/`ParsePatch` already propagate node-construction errors up to `internal/app/graph/service.Apply`'s `readPatch` failure path unchanged, so no new plumbing is needed between `internal/core` and the CLI layer.

**Rationale**: `patchNodeIdentity` is already the single place that resolves the "H1 heading vs. explicit `@type`" precedence (existing `EqualFold` comparison at line 192); adding the casing gate here keeps that precedence logic in one function rather than duplicating the heading/explicit-value resolution in `cmd/arc/graph/apply.go`. Principle III (Hexagonal Architecture): `internal/core` is domain-level parsing logic with no Cobra dependency, so `cmd/` stays limited to formatting the returned error.

**Alternatives considered**: Validating in `cmd/arc/graph/apply.go` after `ParsePatch` returns — rejected: the command layer would need to re-parse the raw document to find H1 headings, duplicating `internal/core`'s own AST walk, and would violate Principle III's "no business logic in `cmd/`" rule.

## D3 — Error type for the rejection

**Decision**: Add to `internal/core/errors.go`:
```go
ErrTypeCasing = faults.Safe1[string]("class name %q must be CamelCase — start with an uppercase letter")
```
wrapped as `ErrTypeCasing.With(typeHeading)` (or `.With(explicitType)` for the FR-008 case), returned from `patchNodeIdentity`.

**Rationale**: Matches the constitution's Mandatory Libraries & Tooling `faults` pattern exactly (`ErrManifestInvalid`/`ErrPatchStructure` are the file's existing two `faults.Type` constants; `Safe1[string]` is the established shape for a message that names one offending value — 8+ existing precedents across the codebase, e.g. `internal/app/schema/service/errors.go:46` `ErrSchemaCycle`).

## D4 — Renaming the four built-in content types

**Decision**: In `internal/app/schema/kernel/schema.go`, rename `CoreTypeDefs`' four content-type map keys and `CoreTypeBases`' four keys: `"source"→"Source"`, `"entity"→"Entity"`, `"resource"→"Resource"`, `"timeline"→"Timeline"`. `"Node"`, `"Property"`, `"Class"` are already CamelCase and are unchanged. `CoreTypeBases`' values (`[]string{"Node"}`) are already correct.

**Rationale**: FR-002 requires every built-in class name to begin uppercase; these are the only four non-compliant built-in names (confirmed by reading `kernel/schema.go:102-164` — `Node`/`Property`/`Class` already comply). `Seed()` (`internal/app/schema/service/schema.go:56-77`) and `RegisterType`/`RegisterPredicate` key output filenames directly off these map keys/`typ` arguments, so this one rename automatically propagates to `arc init`'s seeded `_schema/types/*.md` filenames (FR-003) with no other kernel/service change needed.

**Alternatives considered**: Keeping the map keys lowercase and translating to CamelCase only at render time — rejected: `core.Node.ID`/`core.Node.Type` and the `_schema/types/<name>.md` filename are the same string throughout the codebase (`node.ID == basename` is an enforced invariant elsewhere); introducing a second "display casing" would violate that invariant and add a translation layer nothing else in the codebase has.

## D5 — Physical directory layout is unchanged

**Decision**: `internal/app/graph/service/apply.go`'s `coreKindFolders` map (line 32-36) is updated to key off the new CamelCase type names while keeping its lowercase-plural *values* unchanged: `{"Source": "sources", "Entity": "entities", "Resource": "resources"}`. `cmd/arc/graph/apply.go`'s `pluralizeKind`'s special case (`if kind == "entity" { return "entities" }`) becomes `if kind == "Entity" { return "entities" }`. No other path/folder-naming logic changes.

**Rationale**: The spec (FR-002/003/004/005) governs the `@type` *value* stored in front matter and the `_schema/types/*.md` filename, not the graph's own physical content-folder layout (`sources/`, `entities/`, `resources/`, `timeline/`). Leaving folder names lowercase-plural means an existing repository's directory structure is untouched by this feature — only newly-written nodes' `@type:` value and newly-seeded schema filenames change casing. Changing folder names too would be a second, unrelated breaking change to on-disk layout that neither the spec nor the user's plan direction ("built-in schema and arc init... classes/types as CamelCase"; "arc apply to accept only... H1 is CamelCase") asked for.

**Alternatives considered**: Renaming folders to match (`Sources/`, `Entities/`, `Resources/`) — rejected as unrequested scope expansion with a much larger migration/compatibility footprint (every existing graph's directory tree), inconsistent with FR-009's "no automatic migration" decision for schema content.

## D6 — New `arc lint` rule: `RuleTypeCase`

**Decision**: Add `RuleTypeCase Rule = "typeCase"` to `internal/app/lint/kernel/lint.go`'s existing `Rule` const block (alongside `RulePredicateCase`). Add a new file `internal/app/lint/service/rules_types_case.go` (sibling to `rules_predicates.go`) with:
- `checkNodeTypeCase(node core.Node, path string) []kernel.Violation` — FR-007: one violation if `node.Type` fails `isCamelCase` (using the same regex idiom as `camelCasePattern`, `^[A-Z][a-zA-Z0-9]*$`, mirrored for types), wired into `lint.go`'s existing per-node predicate-checking loop (`internal/app/lint/service/lint.go:~137`) alongside `checkPredicateCase`.
- `checkSchemaTypeCase(index core.Index) []kernel.Violation` — FR-006: one graph-spanning violation (no single owning file — mirrors `checkUniqueBasenames`'s shape, `Path` set to `kernel.TypesDir+"/"+name+".md"`) per `index.Types` key that fails the same check, iterated in sorted-key order for deterministic output.

**Rationale**: `RulePredicateCase`/`camelCasePattern`/`checkPredicateCase` (`internal/app/lint/service/rules_predicates.go:20,46-66`) is the direct precedent for exactly this shape of rule, just inverted (uppercase-first instead of lowercase-first) and applied to the type axis instead of the predicate axis. Splitting into a node-level check (FR-007) and a schema-level check (FR-006) mirrors how the codebase already separates "does this file's own content conform" checks from graph-spanning checks like `checkUniqueBasenames`.

**Alternatives considered**: One combined function checking only nodes (relying on `checkUnrecognizedKind` to eventually catch schema-only issues) — rejected: a schema `Class` document itself (e.g. `_schema/types/entity.md`, ID `entity`) is never walked as a "kind-bearing content node" the same way `checkUnrecognizedKind` walks it (its own `@type` is `Class`, not `entity`) — the only way to catch a badly-cased *type definition itself* is to inspect `index.Types`' keys directly, which requires the separate schema-spanning check.

## D7 — Mechanical literal renames elsewhere

**Decision**: Every remaining `"source"`/`"entity"`/`"resource"`/`"timeline"` string literal that is a *content-type comparison/lookup* (not an unrelated JSON field name or comment) is renamed to match D4, confirmed by direct read of each file:
- `internal/core/markdown.go`'s `textPredicateFor` switch (`"source"`, `"entity"`, `"resource"` cases) → `"Source"`, `"Entity"`, `"Resource"`. The `"hypothesis"`/`"aporia"`/`"thought"` cases are **not** renamed — they are not built-in seeded types (absent from `CoreTypeDefs`) and are out of this feature's scope (spec Assumptions: only built-in types are governed here); a user who defines those types is free to name them CamelCase going forward, and `arc lint`'s new `RuleTypeCase` will flag them if they don't, independent of this switch.
- `internal/app/graph/service/apply.go`: `node.Type == "timeline"` (line 216), `node.Type == "source"` (line 304), `sourcePath := nodeFolder("source")` (line 179) → `"Timeline"`, `"Source"`, `nodeFolder("Source")`.
- `internal/app/lint/service/rules_identity.go`: `node.Type != "source"` (line 22), `node.Type != "entity"` (line 37) → `"Source"`, `"Entity"`.
- `internal/app/lint/service/rules_links.go`: `node.Type == "source" || node.Type == "timeline"` (line 57), `kind == "source"` (line 65) → `"Source"`/`"Timeline"`, `"Source"`.
- `internal/app/lint/service/rules_history.go`: `node.Type != "source"` (line 24) → `"Source"`.
- `cmd/arc/graph/apply.go`: `pluralizeKind`'s `kind == "entity"` (line 35) → `"Entity"` (see D5).

`internal/core/ast.go`'s reference (a doc comment, line 62) is updated for accuracy but carries no runtime behavior. `internal/app/schema/kernel/apply.go`'s `"source"` and `internal/app/graph/kernel/apply.go`'s `"timeline"` are unrelated JSON struct tags (`json:"source"`, `json:"timeline"` — API field names, not `@type` values) and are **not** touched.

**Rationale**: Confirmed via `grep -rln` across `internal`/`cmd` for the four literals, then read every non-test match individually to classify content-type comparisons vs. unrelated JSON tags/comments. All are direct string-equality/map-key comparisons against `core.Node.Type`, so a mechanical 1:1 rename is correct and sufficient — no logic restructuring needed.

## D8 — Test fixtures and existing test assertions

**Decision**: Per repository convention (confirmed by reading `internal/core/markdown_test.go`), most existing fixtures already write H1 headings in title case (`# Entity`, `# Source`, `# Timeline`) while their yaml-fence `"@type"` values are lowercase (`"@type": entity`). After D2's change, the H1 headings themselves need no edits — only: (a) explicit lowercase `"@type"` values in yaml fences, which now fail the new FR-008 gate and must become CamelCase to keep existing tests green where the test's intent is a valid patch, or must have their *expected* outcome changed to "rejected" where the test's intent was already probing an edge case; (b) Go test code asserting `Type: "entity"`/`== "entity"` on parsed results, which must assert `"Entity"` instead; (c) `internal/app/graph/service/apply_test.go` and other suites constructing `core.Node{Type: "source", ...}` literals by hand. This is a mechanical, per-file editing pass at task-generation time (`/speckit-tasks`), not a design decision — enumerated here so Phase 2 tasks can budget for it accurately (~28 files matched by `grep -rl` for the four literals across `*_test.go`).

**Rationale**: TDD (Principle VI/VIII) requires these tests to be updated *as part of* driving the implementation (red→green), not as a follow-up cleanup — the existing suite is the acceptance bar this feature must continue to satisfy under the new casing rule.
