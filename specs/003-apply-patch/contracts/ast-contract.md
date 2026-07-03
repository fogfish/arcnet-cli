# AST Contract: `internal/core`

Public surface any future graph-reading command (`lint`, `retract`, `index build`) can depend on. goldmark types never appear here (research.md D2, D3).

## Parsing

```go
func ParsePatch(r io.Reader) (Patch, error)
func ParseNode(r io.Reader) (Node, error)
```

- `ParsePatch` — CORE §12.2/§12.3. Returns `ErrManifestInvalid` if `kind`/`document`/`published` are missing from the front-matter manifest; `ErrPatchStructure` if the body does not resolve to H1-kind/H2-node sections with a fenced ` ```yaml ` block per node.
- `ParseNode` — CORE §4/§9. Parses one on-disk graph node file (front-matter + body) into a `Node`, used both to read an existing node before merging (research.md D6) and to read an existing timeline period file before inserting an entry (research.md D8).
- Both preserve unrecognized front-matter attributes verbatim in `Attrs` (AST invariant 5).
- Both **strip** `[[Target]]`/`[[Target|alias]]`/`[predicate:: [[Target]]]` bracket markup found inline inside `text`/`notes` prose out of the returned `Node.Text`/`.Notes`, recording each occurrence as a `Link` appended to `HRefs` instead, in the order encountered (research.md D3/D3b). Standalone list-item edges (`- predicate:: [[Target]]`, not embedded in a prose sentence) are unaffected — those populate `Edges`/`Links`, never `HRefs`.
- A predicate-grouped body block (`Links`) is recognized in **either** of two forms (research.md D3c, BUG-003): a `**Label**` paragraph — CORE §12.2's canonical convention ("node bodies use bold labels, never headings") — or a `## Label` H2 heading (this feature's own, non-canonical but still-supported convention), each immediately followed by a list. Every such block, in either form, MUST be captured with no data loss, regardless of how many a node's body contains (spec FR-004).

## Serialization

```go
func RenderNode(n Node) ([]byte, error)
```

- Round-trips losslessly with `ParseNode` for any `Node` produced by `Merge` or constructed fresh from a patch's node section (ARCNET-AST §3.6 "lossless conversion" invariant, with the documented best-effort exception below for repeated-mention `HRefs` reconstruction): front-matter attribute order is stable (insertion order from the source `Attrs` map is not guaranteed — keys are sorted for determinism, since Go maps have no inherent order and CORE does not mandate a specific attribute order), `Edges`/`HRefs` item order and every `Links[predicate].Seq` item order are preserved exactly, `Links` blocks are rendered `edges` first then `links` blocks sorted by `Title` (AST §3.4 rendering rule — block order is not stored, only item order within a block is).
- **Inline wikilink reconstruction (research.md D3b)**: before writing `Text`/`Notes`, `RenderNode` walks `HRefs` in order and re-inserts bracket markup around the first eligible occurrence of each href's display substring (`Alias` if set, else `Target`). An occurrence is eligible only if **(i)** it is not already inside brackets produced by an earlier href in the same pass or already present in the plain text, and **(ii)** it starts at the beginning of the text or is immediately preceded by whitespace, **and** ends at the end of the text or is immediately followed by whitespace/punctuation (the symmetric trailing-boundary check research.md D3b derives from "never mid-word"). A href with no eligible occurrence is left unlinked in the rendered prose — it is not dropped from the `HRefs` the caller already has.
- This reconstruction is **best-effort, not guaranteed byte-exact**, when a target's display substring occurs more than once in the same `Text`/`Notes` with only one occurrence originally meant as a link — flagged explicitly in research.md D3b, not silently claimed as fully lossless.

## Merge (CORE §10, research.md D6)

```go
func Merge(existing, incoming Node, op MergeOp, sourceID string) (merged Node, conflicts []string, err error)
```

- `existing` is the zero `Node` (`ID == ""`) when no node with `incoming`'s identity exists yet — the caller (`internal/app/graph/service.Apply`) treats that case as a plain create, never calling `Merge`.
- Returns `conflicts` — the list of `Attrs` keys whose value was flagged per research.md D6/D7, plus the literal `"notes"` when `Notes` diverges; empty when nothing diverged. **Narrowed by BUG-004**: for a `union` kind, an `Attrs` key is only ever flagged via `MergeUnionFirstWriter`'s already-populated-scalar case (spec.md FR-013) — `MergeUnion`'s own scalar `Attrs` are first-writer-wins with no flag (spec.md FR-023), and the literal `"text"` is never returned at all, since `MergeUnion`'s `Text` field is reconciled paragraph-by-paragraph (spec.md FR-024, research.md D6 Bugfix — BUG-004) rather than compared as one scalar.
- `Merge` performs no I/O; `service.Apply` is responsible for reading `existing` via `ParseNode` and writing `merged` via `RenderNode`.
- `Merge` handles all five `MergeOp` values, including `MergeAppend` (spec.md FR-022, research.md D6 Bugfix — BUG-002): a domain/extension kind registered with `append` merges identically to `union`, but never flags a scalar conflict. `Merge` returns `ErrUnknownMergeOp` only for a genuinely unrecognized `MergeOp` string (e.g. a typo in `.arc/config.yml`), never for one of the five documented values.

## Timeline (CORE §9.4, research.md D8)

```go
func TimelinePeriods(published time.Time) (yearly, monthly string)
func TimelineEntry(id, title string, authors []string, published time.Time) string
```

## Merge-rule vocabulary (CORE §9/§10, research.md D5)

```go
var CoreMergeRules = MergeRuleSet{
    "source":   MergeNone,
    "entity":   MergeUnion,
    "resource": MergeUnionFirstWriter,
    "timeline": MergeAppend,
}

var KnownProfileMergeRules = MergeRuleSet{
    "hypothesis": MergeValidatedOverwrite,
    "aporia":     MergeValidatedOverwrite,
    "thought":    MergeUnion,
}

const ConfigPath = ".arc/config.yml"

func (MergeRuleSet) MarshalYAML() (any, error)
func (*MergeRuleSet) UnmarshalYAML(*yaml.Node) error
```

`MergeRuleSet` is `map[Kind]MergeOp`, with two methods beyond the marshal pair:
- `Union(other MergeRuleSet) MergeRuleSet` — a pure, non-mutating merge of two rule sets, `self` authoritative on conflict (`internal/app/config.Resolve` calls `core.CoreMergeRules.Union(loadedFromFile)`).
- `Lookup(kind Kind) (op MergeOp, ok bool)` — `ok=false` when `kind` is absent (research.md D5 revised); `internal/app/graph/service.Apply` uses this, not a raw map index, to decide between the kind's registered behavior and the safe `union` default-plus-warning fallback.
