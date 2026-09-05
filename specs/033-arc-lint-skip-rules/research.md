# Phase 0 Research: Lint Rule Skipping (`arc lint --skip`, `arc lint rules`)

**Feature**: 033-arc-lint-skip-rules | **Date**: 2026-09-05 | **Spec**: [spec.md](./spec.md)

Every decision below is grounded in code and convention that already exists in
this repository, and in the user's own explicit direction for this feature
("implement the feature at `cmd/arc/lint` level only by apply post filtering
to the results from linter service. The rules id and human readable
definitions are implemented as constants in `internal/app/lint/kernel`").

---

## D1 — Filtering lives entirely in `cmd/arc/lint`, not the lint service

**Decision**: `--skip` parsing, validation, and violation filtering happen
entirely in `cmd/arc/lint`, applied as a post-processing step to the
`kernel.LintResult` that `internal/app/lint/service.Lint` already returns.
`service.Lint`'s signature and behavior are unchanged — every rule always
runs, against every node, regardless of `--skip`.

**Rationale**: This is the user's explicit direction, and it fits Principle
III on its own merits: filtering an already-complete result is "input
validation" and "formatting output" — the two things `cmd/` is explicitly
permitted to do — not a new conformance rule. `service.Lint`'s contract stays
unconditionally complete, which also means nothing about the domain's own
correctness needs re-verifying: the rules being skipped were already checked
correctly before this feature existed.

**Alternatives considered**:
- *Pass a skip set into `service.Lint`* — a larger, unnecessary change (a new
  parameter threaded through every one of the eight rule-check files) for no
  reuse benefit, since no other caller of `Lint` exists today.

---

## D2 — Reconstructing a consistent result after filtering

**Decision**: Node-owned violations and the graph-spanning subset are
filtered separately, reusing the exact split
[`graphSpanningViolations`](../../cmd/arc/lint/lint.go#L46-L59) already
performs, then the filtered result is rebuilt by calling the existing
exported `kernel.NewLintResultWithForeign(root, filteredNodes, foreign,
filteredGraphSpanning...)` — the same constructor `service.Lint` itself
already calls — rather than recomputing `Passing`/`Failing` by hand in
`cmd/`.

**Rationale**: Recounting passing/failing nodes is a small piece of domain
logic that already exists and is already exported; reusing it means the
counting rule can never disagree between an unfiltered and a `--skip`-ed
result, and `cmd/arc/lint` adds no new counting logic of its own — it only
decides *which* violations to keep, which is the filtering step D1
scopes to `cmd/`.

**Alternatives considered**:
- *Recompute `Passing`/`Failing` inline in `cmd/arc/lint`* — duplicates a
  rule kernel already expresses once; a future third counting site (this
  would be the second) is exactly the drift Principle V exists to prevent.

---

## D3 — Rule catalog as a single ordered slice in `kernel`

**Decision**: Add one new package-level value to
[`internal/app/lint/kernel/lint.go`](../../internal/app/lint/kernel/lint.go),
directly after the existing `Rule` const block:

```go
type RuleDefinition struct {
    Rule        Rule   `json:"rule"`
    Description string `json:"description"`
}

var RuleDefinitions = []RuleDefinition{ /* one entry per Rule constant, same order */ }
```

**Rationale**: FR-014 requires `--skip`'s validation and `arc lint rules`'
listing to never drift apart as rules are added, renamed, or removed. A
single slice both sides read makes drift a compile-time-adjacent
impossibility (one literal to update, in one file) rather than a discipline
two independent call sites have to maintain by hand. A slice — not a map —
also gives FR-012's determinism for free: Go randomizes map iteration order
per run, a slice literal's order never changes.

**Alternatives considered**:
- *A `map[Rule]string`* — breaks FR-012's determinism unless sorted at every
  read site; a slice needs no such step.
- *Descriptions embedded in each rule-check file's own `errors.go`* — spreads
  the catalog across eight files (`rules_*.go`), exactly what FR-014's
  single-catalog requirement exists to avoid.

---

## D4 — Error placement follows the existing `--attr`/`--depth` precedent

**Decision**: The "unknown rule in `--skip`" error is a new
`faults.Safe1[string]` constant added to
[`internal/app/lint/service/errors.go`](../../internal/app/lint/service/errors.go)
(alongside the existing `ErrNotAGraph`), returned directly from
`cmd/arc/lint`'s `RunE` — the same shape `cmd/arc/graph/grep.go` uses for
`service.ErrInvalidAttrFlag` and `cmd/arc/graph/subgraph.go` uses for
`service.ErrInvalidDepth` today.

**Rationale**: Principle XII requires `faults` for error annotation, and
every existing flag-*value*-validation error in this codebase already lives
in the domain's `service/errors.go` and is invoked from `cmd/` — never
declared inside a `cmd/` package. Matching this exactly is a smaller,
more consistent change than starting a second convention for the same kind
of error. The parsing and validation logic itself still lives in
`cmd/arc/lint` (D1) — only the error *message constant* follows the
existing `service/errors.go` location.

**Alternatives considered**:
- *Declare the error inside `cmd/arc/lint`* — no precedent anywhere in this
  codebase for a `cmd/` package minting its own `faults` constant; every
  existing one lives beside its domain's other errors.

---

## D5 — `arc lint rules` is a Cobra child of `arc lint`

**Decision**: `lint.NewLintRulesCmd()` is registered as a child via
`lintCmd.AddCommand(lint.NewLintRulesCmd())` in `root.go`, mirroring the
existing `apply` / `apply schema-patch` / `apply batch` parent-child wiring —
the only other place this repository nests a Cobra command today.

**Rationale**: The spec calls it a "sibling command" to lint; Cobra's own
parent-child relationship (`arc lint rules`) expresses exactly that,
consistent with the one precedent already in `root.go`. The alternative — a
second flat top-level command — would need a hyphenated name (`arc
lint-rules`) to avoid colliding with the existing `lint` command, and no
other top-level command in this tree is hyphenated.

**Alternatives considered**:
- *`arc lint-rules` as a flat top-level command* — inconsistent naming, no
  precedent, and less discoverable than a subcommand of the command it
  documents.

---

## D6 — `arc lint rules` output rendering

**Decision**: `bios.Registry[[]kernel.RuleDefinition]{Human: humanRulesPrinter{}}`
— the same registry/printer pattern every other command in this codebase
uses. `--json` is free via the generic `jsonPrinter[T]`. No bespoke
`Verbose` printer: the human-mode list already shows every rule's full
description, so `--verbose` has nothing further to reveal.

**Rationale**: Reuses `internal/bios` exactly as it exists today — no change
to that shared package, keeping this a zero-risk addition to it (Principle
V, YAGNI).

**Alternatives considered**:
- *A distinct verbose form (e.g. including which rules are graph-spanning)*
  — no requirement calls for it; would be speculative surface FR-010
  does not ask for.

---

## D7 — `--skip` parsing rules

**Decision**: Split on `,`, trim each segment, drop empty segments, dedupe by
exact string match, and validate each remaining name against
`kernel.RuleDefinitions` — all inside one small, directly unit-testable
function in `cmd/arc/lint` (no cobra, no I/O), e.g. `parseSkip(csv string)
(skip map[kernel.Rule]bool, unknown []string)`.

**Rationale**: Matches FR-002/FR-003 exactly, and keeps the parsing rule
itself trivially testable in isolation from Cobra wiring — satisfying
Principle VI's "domain logic testable without spinning up Cobra" in spirit
even though this particular function lives in `cmd/` by D1's own direction:
it takes and returns only plain values, no `*cobra.Command`.

**Alternatives considered**:
- *`cobra`'s built-in `StringSlice` flag type* — splits on comma already, but
  does not trim whitespace around each element and would require the same
  validation step afterward regardless; a plain `string` flag with a small
  hand-written parser keeps whitespace-trimming and validation in one place
  instead of two.

---

## Resolved unknowns

| Unknown from Technical Context | Resolution |
|---|---|
| Where does the rule catalog live | D3 — `kernel.RuleDefinitions`, one ordered slice |
| Where does filtering happen | D1 — entirely in `cmd/arc/lint`, post-processing the service result |
| How is a consistent, recounted result rebuilt | D2 — reuse `kernel.NewLintResultWithForeign` |
| Where does the new error constant live | D4 — `internal/app/lint/service/errors.go`, following the `--attr`/`--depth` precedent |
| How is `arc lint rules` wired into the command tree | D5 — Cobra child of `arc lint`, mirroring `apply`'s existing nesting |
| How is `arc lint rules` rendered | D6 — existing `bios.Registry`/`jsonPrinter` pattern, no new machinery |

No `NEEDS CLARIFICATION` markers remain.
