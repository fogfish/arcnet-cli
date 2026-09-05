# Phase 1 Data Model: Lint Rule Skipping (`arc lint --skip`, `arc lint rules`)

**Feature**: 033-arc-lint-skip-rules | **Spec**: [spec.md](./spec.md) | **Research**: [research.md](./research.md)

---

## `kernel.RuleDefinition` — new, in `internal/app/lint/kernel/lint.go`

| Field | JSON | Type | Meaning |
|---|---|---|---|
| `Rule` | `rule` | `Rule` (existing type) | The canonical identifier — the exact value `--skip` accepts and existing `Violation.Rule` already carries |
| `Description` | `description` | `string` | Human-readable explanation of what the rule checks |

## `kernel.RuleDefinitions` — new, in `internal/app/lint/kernel/lint.go`

A single package-level `[]RuleDefinition`, declared in the same order as the
existing `Rule` const block, one entry per rule ([research D3](./research.md)):

| Rule | Description |
|---|---|
| `frontMatter` | Front matter parses as well-formed YAML and declares a recognized identity (`@id`/`@type` or legacy `kind`). |
| `uniqueBasename` | No two node files anywhere in the graph share the same basename. |
| `linkResolves` | Every `[[link]]` target, both structural edges and inline prose references, resolves to a node that exists in the graph. |
| `sourceCitekey` | A source node's identity matches the citekey convention its own content implies. |
| `entityCategory` | An entity node declares one of the twelve legal Sowa category combinations. |
| `derivedProvenance` | A derived node records the node(s) it was derived from. |
| `predicateCase` | Predicates are written in their registered camelCase form, not an unregistered casing variant. |
| `predicateRegistered` | Every predicate a node uses is registered in the graph's schema. |
| `citationPredicate` | Citation predicates align with the schema's cito-aligned citation vocabulary. |
| `unrecognizedKind` | A node's kind is one the graph recognizes, not an unrecognized extension kind. |
| `ingestCommit` | A source node's content is backed by exactly one `graph(ingest):` commit in git history. |
| `mergeConflict` | No unresolved git merge-conflict markers remain in a node file. |
| `typeRequires` | A node satisfies every predicate its declared type's schema marks Required. |
| `typeOptional` | A node's use of predicates its type's schema marks Optional is well-formed. |
| `identityQuoting` | `@id`/`@type` front-matter keys and values are quoted, not left as bare YAML. |
| `predicateRole` | A predicate's structural occurrence (bullet vs. heading, etc.) matches the role its schema declares. |
| `typeCase` | A node's or schema document's declared type is written in its registered camelCase form. |
| `identityCharset` | A node's or schema's identity uses only the graph's permitted identifier character set. |

This is prose-form documentation of the literal Go slice; the slice itself is
the single source of truth (FR-014) — this table MUST be kept in sync with it
by inspection, not treated as an independent definition.

**Invariant A**: `len(RuleDefinitions)` equals the number of declared `Rule`
constants — no rule is missing from the catalog, and no catalog entry names
a rule that does not exist. Unit-tested directly.

**Invariant B**: `RuleDefinitions`' order is fixed and never derived from a
map — repeated reads within and across processes return identical order
(FR-012).

---

## Skip set — new, unexported, in `cmd/arc/lint`

A `map[kernel.Rule]bool`, built by a small pure function from the raw
`--skip` flag value:

```go
func parseSkip(csv string) (skip map[kernel.Rule]bool, unknown []string)
```

| Input | `skip` | `unknown` |
|---|---|---|
| `""` (flag omitted or empty) | empty map | `nil` |
| `"ruleA,ruleB"` | `{ruleA: true, ruleB: true}` | `nil` |
| `"ruleA, ,ruleA"` | `{ruleA: true}` (whitespace and empty segments ignored, duplicate collapsed) | `nil` |
| `"ruleA,bogus"` | `{ruleA: true}` | `["bogus"]` |

`unknown` is deduplicated and preserves first-occurrence order, so the error
message names each bad value exactly once (FR-007).

---

## Filtered `kernel.LintResult` — no new type, existing type reused

`cmd/arc/lint`'s `RunE` transforms the `kernel.LintResult` returned by
`applint.Lint` before it reaches the printer, whenever `skip` is non-empty:

1. For each `NodeStatus` in `result.Nodes`, drop every `Violation` whose
   `Rule` is in `skip` — same node, filtered `Violations` slice.
2. From `result.Violations`, take the subset with no owning node (the same
   set `graphSpanningViolations` already isolates today) and drop every
   entry whose `Rule` is in `skip`.
3. Rebuild via `kernel.NewLintResultWithForeign(result.Root, filteredNodes,
   result.Foreign, filteredGraphSpanning...)` — this recomputes `Violations`,
   `Passing`, and `Failing` from the filtered inputs ([research D2](./research.md)).

**Invariant C**: For every `r` in `skip`, no `Violation` with `Rule == r`
appears anywhere in the rebuilt result — not in any `NodeStatus.Violations`,
not in the flattened `Violations`, in any output mode (FR-004, FR-005).

**Invariant D**: A `NodeStatus` whose `Violations` became empty after
filtering counts toward `Passing`, never `Failing` — a direct consequence of
reusing the existing constructor rather of than adjusting counts by hand
(FR-005).

**Invariant E**: `result.Foreign` is never touched by filtering — foreign
files are not violations and are unaffected by `--skip` (edge case:
foreign-file reporting is orthogonal to rule skipping).

---

## Glossary additions

Staged here for `ARCHITECTURE.md` when it is created (Principle II
pre-existing gap, same handling as spec 032's plan):

- **Lint rule catalog** — the fixed, ordered list of every conformance check
  `arc lint` implements, each paired with its canonical identifier and a
  human-readable description; the single source both `arc lint --skip`
  validates against and `arc lint rules` lists (`kernel.RuleDefinitions`).
