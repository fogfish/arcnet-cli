# Data Model: Full ARCNET-CORE §16 Conformance Checks for `arc lint`

**Feature**: [spec.md](spec.md) | **Research**: [research.md](research.md)

This feature introduces zero new persisted or wire-format types. It adds four new `Rule` enum values to
an existing type (`internal/app/lint/kernel.Rule`) and cross-references two value types that already
exist (`core.TypeDef`, `core.PredicateDef`, both defined in `internal/core/rules.go` since spec 011)
against a third (`core.Node`, `internal/core/ast.go`). This document names the exact relationships the
five new checks evaluate — no schema/struct change accompanies it.

## Existing entities this feature reads (no changes)

### `core.Node` (`internal/core/ast.go`)

The parsed in-memory shape of one node file. This feature reads four of its five fields to enumerate
every predicate occurrence a node carries, per D5:

| Field | Shape | Role this feature treats it as |
|---|---|---|
| `ID`/`Type` | `string` | The two universal identity predicates (`"@id"`/`"@type"`) — never subject to Requires/Optional/role checks |
| `Attrs` | `map[string][]Predicate` | `meta`-role occurrences |
| `Texts` | `map[string]string` | `text`-role occurrences |
| `Edges` | `[]Link` (`.Predicate` non-empty) | `edge`-or-`link`-role occurrences |
| `HRefs` | `[]Link` (`.Predicate` empty) | `href`-role occurrences |
| `HRefs` | `[]Link` (`.Predicate` non-empty) | Citation-tagged inline occurrence — role check exempt (D4), still subject to the citation-predicate check |

### `core.TypeDef` (`internal/core/rules.go`)

```go
type TypeDef struct {
    Merge       MergeOp
    Required    []string  // <- this feature's Requires check (FR-001)
    Optional    []string  // <- this feature's Optional check (FR-002), alongside Required
    Description string
}
```

Already populated by `internal/app/schema/service.Resolve` from each `_schema/types/<name>.md`
document's `## Requires`/`## Optional` edge lists. This feature adds no field to this type; it is a new
*consumer* of `Required`/`Optional`, which no lint check previously read.

### `core.PredicateDef` (`internal/core/rules.go`)

```go
type PredicateDef struct {
    Role        string    // <- this feature's role-conformance check (FR-008) and citation check (FR-006/FR-007)
    Merge       MergeOp
    Label       string
    Aligned     string    // <- this feature's citation check (FR-006/FR-007): valid iff prefix "cito:"
    Description string
}
```

Already populated the same way, from `_schema/predicates/<name>.md`. This feature adds no field; it
reads `Role` and `Aligned` for the first time in a lint check (previously read only by
`internal/core/markdown.go`'s renderer).

## New value: `kernel.Rule` additions (`internal/app/lint/kernel/lint.go`)

Four new constants, following the existing naming/string convention (`RuleXxx Rule = "camelCaseName"`):

| Constant | String value | Fires when (User Story) |
|---|---|---|
| `RuleTypeRequires` | `"typeRequires"` | A node's type is registered, and a predicate that type's `## Requires` lists is absent from the node (US1) |
| `RuleTypeOptional` | `"typeOptional"` | A node's type is registered, and a predicate the node carries is in neither that type's `## Requires` nor `## Optional` (US2) |
| `RuleIdentityQuoting` | `"identityQuoting"` | `"@id"` or `"@type"` is missing, or present but written as a bare (unquoted) YAML key (US3) |
| `RulePredicateRole` | `"predicateRole"` | A registered predicate's occurrence position (per D5) does not match its schema-declared `Role` (US5) |

No new constant is added for User Story 4 (schema-driven citation predicates) — it reuses the existing
`RuleCitationPredicate` constant; only the internal *source* of the valid-predicate set changes (D3), not
the reported rule identity, so existing consumers of that rule name (docs, any downstream tooling) are
unaffected.

## Relationships this feature validates (no new relationship types — existing `kernel.Violation` carries every result, unchanged)

```
Node.Type ──registered-as──> Index.Types[Type] (TypeDef)
                                    │
                    ┌───────────────┼────────────────┐
                    ▼                                 ▼
         TypeDef.Required                   TypeDef.Optional
   "every listed predicate            "every node predicate not
    MUST be present on Node"           in Required MUST be here,
    (RuleTypeRequires)                  or be @id/@type"
                                        (RuleTypeOptional)

Node predicate occurrence ──registered-as──> Index.Predicates[name] (PredicateDef)
                                                     │
                                        ┌────────────┴─────────────┐
                                        ▼                           ▼
                              PredicateDef.Role              PredicateDef.Aligned
                    "occurrence's structural       "if occurrence is a citation,
                     position MUST match Role"       Aligned MUST have prefix
                     (RulePredicateRole,              cito: for the predicate
                      HRefs-with-Predicate exempt)     to be valid"
                                                       (RuleCitationPredicate,
                                                        dynamic per D3)

Node front matter ──raw-text-inspected-for──> "@id"/"@type" key presence + quoting style
                                                       (RuleIdentityQuoting)
```

No entity in this model is created, updated, or deleted by this feature — every arrow above is a
read-only comparison performed once per node per `arc lint` run, consistent with the existing
`kernel.Violation`/`kernel.NodeStatus`/`kernel.LintResult` result shape (`internal/app/lint/kernel/lint.go`),
which is otherwise unchanged by this feature.
