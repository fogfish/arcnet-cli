# `arc subgraph --json` Contract Delta: `Node` Shape

Delta over `specs/007-arc-subgraph/contracts/cli-contract.md`, whose
`"nodes"` array is documented there as "`/* core.Node values */`" — this
document is the concrete before/after for what a `core.Node` value
serializes to, since `internal/core.Node`'s `json` tags are the only
`--json` contract this feature changes (`kernel.SubgraphResult`'s other
fields — `directReachable`, `backlinkTruncated`, etc. — are untouched).

**This is a breaking, non-additive change** (plan.md Constraints/Complexity
Tracking, research.md D8) — accepted pre-1.0, called out explicitly per
constitution Principle XIV rather than hidden.

## Before (specs/007-arc-subgraph, pre-existing)

```json
{
  "id": "Transport Layer Security",
  "kind": "entity",
  "attrs": { "category": ["independent", "abstract"], "tags": ["cryptography"] },
  "text": "A cryptographic protocol establishing authenticated, confidential channels.",
  "notes": "",
  "hrefs": [],
  "edges": [{ "predicate": "replaces", "target": "SSL Protocol" }],
  "links": {
    "mentionedIn": {
      "title": "Mentions",
      "seq": [{ "predicate": "mentionedIn", "target": "rescorla-2026-tls13" }]
    }
  }
}
```

## After (this feature)

```json
{
  "id": "Transport Layer Security",
  "type": "entity",
  "attrs": {
    "category": [{ "value": "independent" }, { "value": "abstract" }],
    "tags": [{ "value": "cryptography" }]
  },
  "texts": {
    "definition": "A cryptographic protocol establishing authenticated, confidential channels."
  },
  "hrefs": [],
  "edges": [
    { "predicate": "replaces", "target": "SSL Protocol" },
    { "predicate": "mentionedIn", "target": "rescorla-2026-tls13" }
  ]
}
```

## Field-by-field delta

| Field | Before | After | Notes |
|---|---|---|---|
| `kind` | `string` | *(removed)* | Replaced by `type` (research.md D1). |
| `type` | *(absent)* | `string` | New; mirrors `"@type"`. |
| `attrs` | `map[string]any` (raw scalars/arrays) | `map[string][]{value?, target?, alias?}` | Every value is now an array of `Predicate` objects, even for a single value (data-model.md `Predicate`, research.md D2/D3). A consumer reading `attrs.category[0].value` instead of `attrs.category[0]` must update. |
| `text` | `string` | *(removed)* | Folded into `texts`, keyed by the type-appropriate predicate name (research.md D4) — e.g. an `entity`'s old `text` becomes `texts.definition`. |
| `notes` | `string` | *(removed)* | Folded into `texts` under a `notes`-suffixed key when present (research.md D4); omitted entirely from `texts` when empty (no more empty-string `notes` field). |
| `texts` | *(absent)* | `map[string]string` | New. Omitted (or `null`) when the node has no prose. |
| `hrefs` | `[]Link` | `[]Link` | Unchanged shape and semantics. |
| `edges` | `[]Link` (structural, ungrouped-only) | `[]Link` (every structural link, grouped or not) | Now the single source of every outgoing link — a consumer previously reading only `edges` and ignoring `links` was already incomplete; that gap is closed by this change (spec FR-017), not introduced by it. |
| `links` | `map[string]{title, seq}` | *(removed)* | Grouping is no longer stored (research.md D5); every entry that was in `links[*].seq` now appears in `edges` instead, in document order interleaved with what used to be plain `edges` entries. A consumer that rendered `links` blocks separately (e.g. `arc serve`'s node view) must instead derive any grouping it still wants to show at render time from each `Link.Predicate` — spec 013's concern, not this feature's; until spec 013 lands, `arc serve` renders every edge in one flat list. |

## Consumer migration notes

- A script parsing `attrs.<key>` as a bare value MUST change to
  `attrs.<key>[0].value` (or iterate the array for a multi-valued
  attribute).
- A script reading `text`/`notes` MUST change to look up the appropriate
  key in `texts` — there is no single fixed key name across all node
  types; `source` nodes use `texts.abstract`/`texts.notes`, `entity` nodes
  use `texts.definition`/`texts.notes`, etc. (research.md D4's table).
- A script reading `links` for grouped display MUST change to read
  `edges` and, if grouped display is still desired, group client-side by
  `Predicate` — the server-side grouped-heading concept `links` used to
  carry no longer exists as stored data.
- No `--json` schema-version flag is introduced by this feature
  (research.md D8's rejected alternative) — a consumer pinned to the old
  shape must pin the `arc` binary version instead, until the project
  reaches 1.0 and documents a stability contract.
