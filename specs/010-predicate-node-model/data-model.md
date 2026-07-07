# Phase 1 Data Model: Predicate-First Graph Node Model

Scope: `internal/core.Node`, its new `Predicate` member type, and the
`Patch` container that carries `Node`s across `arc apply`'s exchange
format. `Link` is unchanged from its current shape. This document
supersedes the `Node`/`Link`/`LinkBlock` section of
`specs/003-apply-patch/contracts/ast-contract.md` and
`specs/009-node-timestamp-attrs`'s additive `Published` note; `Published`
itself is untouched by this feature (still `time.Time`, still merged the
same way) and is simply carried over below for completeness.

## Node

The graph's addressable unit — one Markdown file on disk, or one `## <ID>`
section inside a patch document.

| Field | Type | Required | Notes |
|---|---|---|---|
| `ID` | `string` | Yes | From `"@id"`. MUST equal the file's basename (extension stripped) for a standalone node file. For a patch-document node contribution, satisfied by the `"## <ID>"` section heading text itself — an explicit `"@id"` key inside the node's own yaml fence is optional, and if present MUST agree with the heading (BUG-001). No fallback to `title`/`period`/legacy `id`. |
| `Type` | `string` | Yes | From `"@type"` for a standalone node file. For a patch-document node contribution, established from the enclosing `"# <Type>"` section heading when no explicit `"@type"` key is present in the node's own yaml fence — mirroring `ID`'s own patch-heading carve-out above and the pre-existing CORE §12.2 convention; when both the heading and an explicit `"@type"` key are present, they MUST agree (BUG-001). Open vocabulary (`source`, `entity`, `resource`, `timeline`, plus graph-registered custom types). Replaces the old `Kind Kind` field; the `Kind` named type is removed project-wide (research.md D1). |
| `Published` | `time.Time` | No | Unchanged from spec 009 — zero value means unset; filled on create, never overwritten once non-zero. Not part of `Attrs` (dedicated top-level field, same as `"@id"`/`"@type"`). |
| `Attrs` | `map[string][]Predicate` | No | Every front-matter key other than `"@id"`/`"@type"`/`"published"`. Every present key's slice is non-empty. A key absent from the map means the attribute was not declared at all (distinct from an explicitly empty list, which AST forbids — see Predicate below). |
| `Texts` | `map[string]string` | No | Every named prose field. Keys populated by this feature: at most two per node, `textPredicateFor(Type, leading)`'s result for the leading prose block and for the trailing prose block (research.md D4) — e.g. `{"abstract": "...", "notes": "..."}` for a `source`. A node with no prose has an empty/nil map. |
| `HRefs` | `[]Link` | No | Unchanged — inline `[[Target]]`/`[[Target\|alias]]` mentions extracted from `Texts` values at parse time, reconstructed into `Texts` at render time. Never a source of navigable edges (AST invariant, unchanged from today). |
| `Edges` | `[]Link` | No | Every outgoing structural link, in document order, regardless of whether the source document wrote it as a flat bullet or grouped under a heading/bold label. Replaces the old `Edges []Link` + `Links map[string]LinkBlock` pair; `LinkBlock` is deleted (research.md D5). |

Removed from today's shape: `Kind Kind` (→ `Type string`), `Text string` +
`Notes string` (→ `Texts map[string]string`), `Links map[string]LinkBlock`
(→ folded into `Edges`).

**Validation rules** (enforced by `ParseNode`/`parsePatchBody`, all
resulting in `ErrManifestInvalid` on failure, per research.md D7):

- `"@id"` MUST be present and non-empty.
- `"@type"` MUST be present and non-empty.
- For a standalone node file, `"@id"` MUST equal the file's basename
  (extension stripped) exactly.
- **BUG-001**: For a patch-document node contribution specifically,
  `"@id"`/`"@type"` being "present" is satisfied by the `"## <ID>"`/
  `"# <Type>"` section headings alone — an explicit yaml-fence key is
  optional, not mandatory, and (when present) is cross-checked for
  agreement with the heading rather than required as the sole source.
  This carve-out does not apply to a standalone node file, which has no
  section heading to derive from and must declare both explicitly in its
  own front matter.
- A front-matter `kind` key (the old identity field) present at all is
  treated as an old-format file and rejected, even if `"@id"`/`"@type"`
  are also present (ambiguous/mixed-format files are not given the benefit
  of the doubt).
- Every `Attrs` value list is non-empty when the key is present (a
  zero-element list is never constructed; an attribute with no values is
  simply absent from the map).

## Predicate

One value contributed to an `Attrs` entry.

| Field | Type | Notes |
|---|---|---|
| `Value` | `any` | A JSON/YAML scalar (string, number, bool), as authored. Set when this predicate is not a reference. |
| `Target` | `string` | The target node's basename, when this predicate is reference-valued (e.g. a front-matter key whose value is a `[[Target]]`-shaped or otherwise identity-bearing reference). Informative only — never itself a source of a navigable edge (mirrors AST §7's `attrs`-vs-`edges` separation; a reference-shaped `Attrs` entry does not appear in `Edges`). |
| `Alias` | `string` | Optional display alias, meaningful only alongside `Target`. |

**Invariant**: exactly one of `Value`/`Target` is set per `Predicate`
(AST §7). This feature's parser only ever produces `Value`-set predicates
(front-matter scalars/arrays); the `Target`/`Alias` fields exist on the type
now so a later feature (schema-driven reference attributes) does not need
another `Node`-shape change — populating them is out of this feature's
scope.

## Link (unchanged)

| Field | Type | Notes |
|---|---|---|
| `Predicate` | `string` | Optional relationship name (e.g. `mentions`, `replaces`). Empty for a bare/untyped mention. |
| `Target` | `string` | The target node's basename. Required. |
| `Alias` | `string` | Optional display text, from `[[Target\|alias]]`. |

No shape change. Used for exactly one of `HRefs` or `Edges` per occurrence,
never both (unchanged invariant).

## Patch

The exchange document `arc apply` consumes.

| Field | Type | Notes |
|---|---|---|
| `Document` | `string` | Unchanged. |
| `Published` | `time.Time` | Unchanged. |
| `Title` | `string` | Unchanged. |
| `Stats` | `map[string]any` | Unchanged. |
| `Nodes` | `[]Node` | Each element follows `Node`'s new shape above; each node contribution's `"@id"` is established from its `"## <ID>"` section heading (an explicit yaml-fence `"@id"` key is optional, and if present MUST agree — BUG-001), with no fallback — same rule as a standalone file (spec FR-011). Likewise, `"@type"` is established from the enclosing `"# <Type>"` heading when no explicit `"@type"` key is present in the fence (spec FR-018, BUG-001). |

No field of `Patch` itself changes shape; only the `Node`s it carries do.

**Bugfix**: 2026-07-07 — BUG-001 Updated from bugfix patch: reconciled the `Node.ID`/`Node.Type` row asymmetry above (`Type` previously had no patch-heading carve-out, unlike `ID`) and clarified the `Nodes` row accordingly.

## Relationships / Lifecycle

- **Parse**: `[]byte` (Markdown + YAML front matter) → `Node`. Front-matter
  scalars/arrays become `Attrs` lists; structural body prose becomes
  `Texts` entries keyed via `textPredicateFor`; bare-list and
  heading/bold-label-grouped links both flatten into `Edges`, in the
  document's original left-to-right, top-to-bottom order across both
  sources; inline `[[...]]` mentions inside `Texts` values are extracted
  into `HRefs` and stripped from the stored `Texts` string.
- **Render**: `Node` → `[]byte`. Inverse of Parse for `Texts`/`HRefs`
  (brackets reconstructed), `Attrs` (single-element lists render as
  scalars, multi-element as sequences, research.md D3), and `Edges`
  (flat bullets only, research.md D6). Round-trip (Parse ∘ Render) MUST be
  the identity up to the permitted cosmetic edges-grouping normalization
  (spec FR-014); Render ∘ Parse ∘ Render MUST be strictly idempotent
  (spec FR-015).
- **Merge**: two `Node`s sharing the same `ID` combine per `MergeOp`
  (unchanged policy set — union/first-writer/append/etc., research.md
  Constraints). `Texts` merges key-by-key over the union of both nodes'
  keys, applying the existing scalar-merge policy per key. `Attrs` merges
  key-by-key over the union of keys, applying the existing policy to each
  key's `[]Predicate` list (list-level union/first-writer/append, not
  element-level). `Edges` merges as one unioned list (existing `Link`
  union behavior, now applied to a single collection instead of
  `Edges`+`Links` separately). `HRefs` unchanged.
