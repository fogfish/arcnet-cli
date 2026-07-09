# Render-Shape Contract: `internal/core.RenderNode`/`RenderPatch`

Public surface every command that ever writes a node back to Markdown depends on (`arc apply`, `arc subgraph`
(stdout), `arc serve` (MCP tool replies), the schema use-case's `Seed`/`RegisterType`/`RegisterPredicate`).
Supersedes specs/010-predicate-node-model's `ast-contract.md` note that grouping is "derived at render time,
never stored" by fixing *what* render-time derivation means: a predicate's declared schema `Role`, not the
shape last observed at parse time.

## Signatures (breaking change within this repo; no external `--json` contract touched)

```go
func RenderNode(n Node, index Index) ([]byte, error)
func RenderPatch(p Patch, index Index) ([]byte, error)
```

Both gain one new required parameter, `index Index` (already an `internal/core` type — no new import). Every
existing caller (production and test) must be updated; there is no variadic/optional-parameter compatibility
shim, per this codebase's convention of correcting call sites directly rather than carrying a compatibility
shim for an internal (non-`--json`) function signature.

## Rendering algorithm (normative)

**⚠️ Amended — BUG-001 (2026-07-09)**: step 3's markup (below) originally applied identically to both
`RenderNode` and `RenderPatch`. It now branches by caller — see the "Link-role group markup by format" note
after step 3 — because ARCNET-CORE §14.2 reserves `##` exclusively for a patch document's own `@type`/`@id`
structure; a `## Label` heading inside a patch's per-node body corrupts that structure. Steps 1, 2, 4, 5, and
6 are unaffected by this amendment — the partition, ordering, and single-group-omission decisions are
identical for both `RenderNode` and `RenderPatch`; only the literal Markdown markup step 3 emits for a
link-role group differs by caller.

For a given `Node`/`Index` pair, `RenderNode`/`RenderPatch` MUST render `n.Edges` as follows:

1. Partition `n.Edges` by each occurrence's resolved role: `index.Predicates[l.Predicate].Role`, or `"edge"`
   when `l.Predicate` has no entry in `index.Predicates` (never an error, never a dropped occurrence).
2. Every `"edge"`-role occurrence renders as one flat bulleted list (`renderLinkBullet` per line, unchanged
   format), in original relative order across all edge-role predicates, with **no** heading.
3. Every `"link"`-role occurrence is grouped by its predicate name into one block per distinct name present,
   followed by that predicate's occurrences (`renderLinkBullet` per line, in original relative order within
   the group). `label` is `index.Predicates[name].Label` if non-empty, else the predicate name capitalized
   (`titleCaseType`). **Link-role group markup by format** (BUG-001): in `RenderNode`'s output, the block is
   `"## " + label + "\n"` (ARCNET-CORE §5); in `RenderPatch`'s output, the block is `"**" + label + "**\n"`
   — a bold-label paragraph, never a heading (ARCNET-CORE §14.2, "Markdown headings are reserved for type
   and identity; node bodies use bold labels, never headings").
4. Link-role blocks are ordered by their resolved `label`, ascending — this MAY differ from the original
   document's block order (permitted normalization, spec FR-010). Ordering is identical regardless of
   whether step 3 rendered a heading or a bold label.
5. **Exception**: if step 1 produces zero edge-role occurrences and step 3 produces occurrences of exactly one
   distinct link-role predicate name, that one group's heading/bold-label is omitted and its occurrences
   render as a bare bulleted list instead (same shape/position as step 2's flat list). This applies
   identically to both formats.
6. The rendered order within a node's body is: leading text (unchanged) → step 2's flat list, if non-empty →
   step 3/5's link block(s), if any → trailing text (unchanged). This exact ordering is required for the
   *existing, unchanged* parser (`walkNodeBody`) to read the output back into the same `Edges` shape — this
   contract does not change parsing (specs/010) or the parser's own grammar. `walkNodeBody`'s `blockTitle`
   helper already recognizes both an `## Label` heading and a `**Label**` bold-label paragraph as a valid
   group title (BUG-003 precedent), so this amendment requires no parser change in either format.

## Round-trip guarantees

- **Byte-stable on already-canonical input** (FR-008): `RenderNode(ParseNode(RenderNode(n, index)), index)` is
  byte-equal to `RenderNode(n, index)` for the same `index`, for any `n` this package's own renderer produced.
- **Normalizing on non-canonical input** (FR-009): a node parsed from a document whose original shape
  disagrees with its predicates' declared roles (e.g., a `link`-role predicate written as a flat bullet, or an
  `edge`-role predicate written grouped under a heading) re-renders in the canonical shape derived from
  `index` — never in the shape the original document happened to use.
- **Content preservation** (FR-010): normalization MAY reorder heading-block position and MAY reposition edge
  bullets relative to link groups, but MUST NOT alter, drop, or duplicate any `Link`'s `Predicate`/`Target`/
  `Alias`.
- **Unaffected by `index`'s completeness**: an `Index` missing an entry for some predicate present in `n.Edges`
  never causes an error — that predicate renders flat (edge-role fallback), per the algorithm above.
