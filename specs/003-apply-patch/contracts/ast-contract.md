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

## Serialization

```go
func RenderNode(n Node) ([]byte, error)
```

- Round-trips losslessly with `ParseNode` for any `Node` produced by `Merge` or constructed fresh from a patch's node section (ARCNET-AST §3.6 "lossless conversion" invariant): front-matter attribute order is stable (insertion order from the source `Attrs` map is not guaranteed — keys are sorted for determinism, since Go maps have no inherent order and CORE does not mandate a specific attribute order), `Edges`/`HRefs` item order and every `Links[predicate].Seq` item order are preserved exactly, `Links` blocks are rendered `edges` first then `links` blocks sorted by `Title` (AST §3.4 rendering rule — block order is not stored, only item order within a block is).

## Merge (CORE §10, research.md D6)

```go
func Merge(existing, incoming Node, op MergeOp, sourceID string) (merged Node, conflicts []string, err error)
```

- `existing` is the zero `Node` (`ID == ""`) when no node with `incoming`'s identity exists yet — the caller (`internal/app/graph/service.Apply`) treats that case as a plain create, never calling `Merge`.
- Returns `conflicts` — the list of `Attrs` keys (or the literal `"text"`) whose value was flagged per research.md D6/D7; empty when nothing diverged.
- `Merge` performs no I/O; `service.Apply` is responsible for reading `existing` via `ParseNode` and writing `merged` via `RenderNode`.

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

`MergeRuleSet` is `map[Kind]MergeOp`; `Union(other MergeRuleSet) MergeRuleSet` (a pure, non-mutating merge of two rule sets — `internal/app/config.Resolve` calls `core.CoreMergeRules.Union(loadedFromFile)`) is the one small method beyond the marshal pair.
