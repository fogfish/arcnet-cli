# Phase 0 Research: Predicate-First Graph Node Model

No `[NEEDS CLARIFICATION]` markers remain in the Technical Context — the user's
own plan input fully specified the technical approach. This document records
the design decisions that approach implies, so Phase 1 has a settled basis to
draft `data-model.md`/`contracts/` from, and so later readers see the
rationale, not just the diff.

## D1 — Drop the `Kind` named type in favor of plain `string`

**Decision**: `internal/core.Kind` (`type Kind string`) is removed entirely.
`Node.Type` (renamed from `Node.Kind`) is `string`. Every package that
currently keys a map or types a parameter by `core.Kind`
(`MergeRuleSet map[Kind]MergeOp`, `coreKindFolders`, `Created`/`Merged` result
maps, `SchemaKind`, `coreKindDescriptions`, `RegisterKind`'s parameter, the
grep kernel's `Kind` JSON field) is mechanically updated to `string`.

**Rationale**: ARCNET-AST v0.6 §4 treats `@type` as an open-vocabulary
string, not a closed enum — the codebase's own comments already say as much
("Open vocabulary"). A dedicated named type added no compile-time safety
(the value still comes from unchecked YAML/JSON at the boundary) and this
feature's own field rename (`Kind`→`Type`) is the natural point to drop the
indirection rather than carry it forward under a new name.

**Alternatives considered**: Keep `type Kind = string` as an alias so
existing signatures need no further edit — rejected because a bare alias
buys nothing (call sites still touch every file to rename `Kind`→`Type` on
`Node` itself) while leaving a redundant type name lingering past its
usefulness, contrary to Principle V (YAGNI).

## D2 — `Predicate` struct shape for `Attrs`

**Decision**: `type Predicate struct { Value any; Target string; Alias string }`,
exactly one of `Value`/`Target` populated per AST §7. `Attrs` becomes
`map[string][]Predicate`; every key's slice is non-empty when present.

**Rationale**: Matches AST §7 verbatim ("every `attrs` entry is a
non-empty, ordered array of `Predicate`, each with exactly one of
`value`/`target`"). Reusing the existing `Link`-adjacent shape (`Target`,
`Alias`) for the reference-valued case keeps the vocabulary consistent with
how `Edges`/`HRefs` already describe a reference.

**Alternatives considered**: A tagged union / interface (`AttrValue`
interface with `ScalarAttr`/`RefAttr` implementations) was considered for
stronger compile-time exclusivity — rejected as unnecessary ceremony for a
value that is only ever constructed by the parser and consumed by
`RenderNode`/merge code within the same package; a plain struct with an
"exactly one of" GoDoc contract, mirroring how `Link` already documents its
own "used for exactly one of three purposes" invariant, is consistent with
existing style (Principle IV: composition over complexity, no unneeded
abstraction).

## D3 — Front-matter scalar → list normalization happens at parse time, list → scalar collapse happens at render time

**Decision**: `ParseNode`/`parsePatchBody` wrap every front-matter value
(after removing `"@id"`/`"@type"`) into `[]Predicate`: a YAML scalar becomes
a one-element list; a YAML sequence becomes one element per item.
`renderAttrYAML`'s replacement renders a one-element `[]Predicate` back as a
bare scalar and a multi-element list as a YAML sequence.

**Rationale**: This is what makes FR-014/FR-015 (round-trip and idempotent
round-trip) hold against files a maintainer already hand-wrote with plain
scalars (`year: 2018`) — CORE's own §11 worked examples show single-valued
predicates as bare scalars on disk even though AST's in-memory model always
treats them as a sequence (AST §7's cardinality-erasure is a modeling
statement, not a serialization mandate). Collapsing at render time, not at
parse time, keeps the internal representation uniform (every consumer
always deals with `[]Predicate`, never a scalar-or-list union) while still
producing scalar-shaped YAML on disk for the common single-value case.

**Alternatives considered**: Always rendering as a flow-style array
(`tags: [cryptography]` even for one value) was rejected — it breaks
round-trip fidelity against every existing hand-written single-valued
attribute in the current graph corpus, which is exactly the kind of loss
spec.md's Assumptions section rules out.

## D4 — `Texts` keyed by an explicit, temporary `@type`→predicate lookup table

**Decision**: `walkNodeBody` keeps its existing structural parse (leading
paragraphs / optional bare list / heading-or-bold-label blocks / trailing
paragraphs) unchanged in *shape-recognition* terms, but its two prose
outputs are now labelled via `textPredicateFor(nodeType string, leading
bool) string`, a small hardcoded table: `source`→`abstract`/`notes`,
`entity`→`definition`/`notes`, `resource`→`relevance`/`notes`,
`hypothesis`→`claim`/`notes`, `aporia`→`tension`/`notes`,
`thought`→`claim`/`notes`, and a generic fallback `text`/`notes` for any
other/unregistered `@type`.

**Rationale**: This is the smallest change that (a) satisfies AST §6.1's
requirement that `Texts` be an open, name-keyed map (a future node type or a
future spec 011 Schema Index can add table entries or replace the table
outright without touching `Node`'s shape again) and (b) produces
domain-appropriate predicate names (a `source`'s leading prose really is its
`abstract`) without inventing schema-role parsing this feature explicitly
excludes.

**Known limitation, carried into Constraints/Complexity Tracking**: because
`walkNodeBody` still only recognizes two prose *positions* per node
(leading, trailing), a node cannot yet declare a third independently named
prose section by writing, say, a `## Relevance` heading followed by a
paragraph — that would require teaching the heading/bold-label matcher to
also claim paragraph-followed blocks (today it only claims list-followed
blocks, for links), which is deliberately deferred to avoid pre-empting
spec 011's Schema Index-driven answer to "how is a heading's role known
without parsing schema."

**Alternatives considered**: Teaching `walkNodeBody` to recognize
`"## <Name>"` + paragraph as an arbitrary named text block now (fully
satisfying FR-005's "several sections" scenario immediately) — considered
and set aside for this spec specifically because the plan's own scoping
statement treats the lookup table as the complete text-labelling mechanism
for this increment; revisit alongside spec 011 rather than shipping a
heuristic likely to be replaced within one or two specs.

## D5 — `Edges` unifies `Edges`+`Links`; `LinkBlock` type is deleted

**Decision**: `Node.Links map[string]LinkBlock` and the `LinkBlock` type are
removed. Every link previously captured in either container — a bare
bulleted list or a heading/bold-label-grouped list — becomes one `Link`
entry appended, in document order, to a single `Node.Edges []Link`.

**Rationale**: AST §3 invariant 4 states grouping is derived at render
time, never stored — keeping a `Links` container at all would preserve the
exact old-shape distinction this feature exists to remove (FR-007/FR-008).

**Alternatives considered**: None seriously — this is the feature's
explicit, named requirement (spec.md FR-007), not a design choice with
competing options.

## D6 — Render-time flat rendering only; grouped-heading rendering deferred to spec 013

**Decision**: `RenderNode`/`RenderPatch`'s replacement renders every
`Node.Edges` entry as a flat `- predicate:: [[Target]]` bullet (or bare
`[[Target]]` when `Predicate` is empty), in `Edges`' stored order. No
`"## <Label>"` grouped rendering is produced by this feature.

**Rationale**: AST §10's conformance checklist explicitly permits
normalizing cosmetic edges-grouping order/layout on round-trip ("cosmetic
`edges`-grouping order **MAY** be normalized") — so flat-only rendering does
not violate FR-014's round-trip requirement even for a node originally
authored with a grouped heading block; content and connectivity survive
identically, only the on-disk grouping cosmetic changes. Deferring the
role-driven grouped/flat decision (FR-008) to spec 013 avoids building a
second, temporary role-heuristic here that spec 013's real Schema
Index-driven renderer would then have to replace.

**Alternatives considered**: Preserving each link's *original* grouped/flat
form as parsed (effectively keeping a per-link "was grouped" bit) was
rejected — it is exactly the "fixed by how the source document happened to
write it" behavior spec.md's FR-008 explicitly forbids; better to render
uniformly flat now (a real, spec-permitted interim state) than to fake
role-awareness with parse-order memory.

## D7 — Old-format detection and rejection

**Decision**: A node/patch file is rejected (existing `ErrManifestInvalid`
fault, extended with a specific guidance message) whenever front matter
contains a legacy `kind` key, or is missing `"@id"`, or is missing
`"@type"`, or `"@id"` does not equal the file's basename (extension
stripped). No fallback parsing path for any of these cases is implemented.

**Rationale**: Directly implements spec.md FR-012/FR-013 and US3's four
acceptance scenarios; reusing the existing `ErrManifestInvalid` fault
(rather than inventing a parallel error type) keeps every caller's existing
error-handling path (`service.Apply`, lint, subgraph) unchanged — they
already propagate this fault today, so no new call-site branching is
needed, only a clearer message.

**Alternatives considered**: A best-effort "detect old format, offer a
migration hint" mode was considered and explicitly rejected per the user's
own plan input ("support of old-format... MUST NOT BE implemented, the
compatibility or migration is not a concern at this phase") and spec.md's
Assumptions (no migration tool in this feature).

## D8 — `arc subgraph --json` breaking change is accepted, not hidden

**Decision**: `kernel.SubgraphResult.Patch.Nodes`' JSON shape changes
non-additively (`kind`→`type`, `attrs` values become arrays of
`{value|target,alias}`, `text`/`notes` collapse into a `texts` map,
`edges`+`links` collapse into one `edges` array). No parallel
backward-compatible field is kept alongside the new shape.

**Rationale**: The old and new `Node` shapes are not simultaneously
representable without either duplicating every field (violating Principle V
YAGNI for a pre-1.0 tool with no documented `--json` stability commitment
yet) or serializing lossy/ambiguous data. The project's release train is
still `0.0.x` (per recent tags/CHANGELOG entries), so Principle XIV's
"SHOULD precede a breaking `--json` change with a deprecation warning" is
weighed against there being no external consumer this warning could
meaningfully protect yet.

**Alternatives considered**: A `--json` schema version field / opt-in flag
to select old vs. new shape — rejected as speculative infrastructure for a
compatibility guarantee the project has not made; revisit if/when `arc`
reaches 1.0 and commits to `--json` stability.
