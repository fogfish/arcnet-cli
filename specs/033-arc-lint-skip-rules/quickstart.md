# Quickstart & Validation Guide: `arc lint --skip`, `arc lint rules`

**Feature**: 033-arc-lint-skip-rules | **Plan**: [plan.md](./plan.md)

How to run the feature and how to prove it satisfies the spec. Implementation
belongs in `tasks.md`; this file is the validation contract.

---

## Prerequisites

```bash
cd /Users/kolesnik/devel/go/src/github.com/fogfish/arcnet-cli
go build ./...
```

Same prerequisites as `arc lint` itself: a real `git` binary on `PATH` (the
existing `TestMain` in `cmd/arc/lint/lint_test.go` sets a fake commit
identity for the whole test binary).

---

## Running it

```bash
cd <a graph directory>

arc lint rules                     # list every rule, name + description
arc lint rules --json              # machine-readable rule catalog

arc lint --skip ingestCommit       # suppress one rule's violations
arc lint --skip ingestCommit,typeOptional   # suppress several
arc lint --skip bogusRule          # refuses immediately, names "bogusRule"
```

`arc lint rules` works in any directory, graph or not.

---

## Test fixtures

No `testdata/` directory — `cmd/arc/lint/lint_test.go` already builds every
fixture inline via `t.TempDir()` plus the existing `buildConformantGraph`,
`writeNode`, `ingestSource` helpers. This feature reuses that convention and,
where possible, the exact broken fixtures those helpers already produce:

| Existing fixture (from `TestLintReportsEveryViolationAcrossRulesInSameFile`) | Rules it triggers | Reused for |
|---|---|---|
| `Widget.md` missing 4th Sowa word + linking to a nonexistent node | `entityCategory`, `linkResolves` | User Story 1 scenario 1 — skip one, confirm the other still reports |
| Two files sharing a basename (`TestLintBasenameCollisionNamesBothFiles`) | `uniqueBasename` (graph-spanning) | Edge case — skipping a graph-spanning rule |

---

## Acceptance scenario → test map (Principle VIII, 1:1)

All tests colocated in `cmd/arc/lint/`, driven through `RunE` via the
existing `sut()` helper, flags set via `cmd.Flags().Set(...)` before calling
`sut` (mirroring `cmd/arc/graph/subgraph_test.go`'s `--depth`/`--type`
pattern).

### User Story 1 — `--skip` suppresses one rule's violations

| Scenario | Test | Asserts |
|---|---|---|
| 1.1 skip one rule, other rule's violations unchanged | `TestLintSkipSuppressesNamedRuleOnly` | Output contains `[linkResolves]`, does not contain `[entityCategory]`, when `--skip entityCategory` |
| 1.2 all violations belong to the skipped rule → success | `TestLintSkipAllViolationsInSkippedRuleSucceeds` | `err` is `nil`, summary line shows `0 failing` |
| 1.3 multiple rules named, comma-separated | `TestLintSkipMultipleRulesCommaSeparated` | Both named rules' violations absent, others present |
| 1.4 every implemented rule skipped at once | `TestLintSkipEveryRuleStillEnumeratesNodes` | Summary line shows `N nodes checked, N passing, 0 failing`; `err` is `nil` |

### User Story 2 — `arc lint rules`

| Scenario | Test | Asserts |
|---|---|---|
| 2.1 lists every rule with a description | `TestLintRulesListsEveryRuleWithDescription` | Output contains every `kernel.RuleDefinitions` entry's rule name and description |
| 2.2 works with no graph present | `TestLintRulesWorksOutsideGraph` | Run in an empty `t.TempDir()`, `err` is `nil` |
| 2.3 listed name is accepted by `--skip` | `TestLintRulesNameAcceptedBySkip` | For each `kernel.RuleDefinitions` entry, `cmd.Flags().Set("skip", name)` on a fresh `arc lint` produces no "unknown rule" error |
| 2.4 deterministic across runs | `TestLintRulesDeterministicOutput` | Two runs produce byte-identical output |

### User Story 3 — clear refusal on a bad `--skip` value

| Scenario | Test | Asserts |
|---|---|---|
| 3.1 unknown rule name refuses, names it | `TestLintSkipUnknownRuleRefuses` | `err` non-nil, output/error contains `"bogusRule"` and `"arc lint rules"` |
| 3.2 mixed valid + invalid still refuses | `TestLintSkipMixedValidAndInvalidRefuses` | Refuses on the invalid name; no partial lint output printed |

### Cross-cutting / edge cases

| Requirement | Test |
|---|---|
| FR-002 — whitespace/empty segments ignored | `TestLintSkipIgnoresWhitespaceAndEmptySegments` — `--skip "entityCategory, ,entityCategory"` behaves like `--skip entityCategory` |
| FR-003 — duplicate rule name, no error | Covered by the same test above |
| FR-004 — graph-spanning rule skip | `TestLintSkipGraphSpanningRuleSuppressesBothFiles` — `--skip uniqueBasename` against the basename-collision fixture, confirms `[uniqueBasename]` absent for both files |
| FR-008 — validated before graph resolution | `TestLintSkipUnknownRuleRefusesBeforeGraphCheck` — run in a directory that is *not* an initialized graph, with an unknown `--skip` value, and assert the error names the bad rule, not "not an initialized graph" |
| `--json` + `--skip` | `TestLintSkipJSONOutputOmitsSkippedRuleViolations` — parses the JSON, asserts no violation object has `"rule": "<skipped>"`, and `passing`/`failing` match the filtered count |
| `arc lint rules --json` | `TestLintRulesJSONOutput` — parses as `[]struct{Rule, Description string}`, non-empty, matches `kernel.RuleDefinitions` length |

Unit tests for the pure parsing/catalog logic:

- `cmd/arc/lint/lint_test.go` (or a new colocated `skip_test.go`): table-driven
  tests for `parseSkip` covering every row in
  [data-model.md](./data-model.md)'s parsing table (Principle VI table-driven
  requirement).
- `internal/app/lint/kernel/lint_test.go`: `TestRuleDefinitionsCoverEveryRule`
  — asserts `len(RuleDefinitions) == ` the number of declared `Rule`
  constants and that every entry's `Description` is non-empty.

---

## Validation commands

```bash
go test ./...                                      # full suite
go test ./cmd/arc/lint/ -run TestLintSkip -v        # --skip scenarios
go test ./cmd/arc/lint/ -run TestLintRules -v       # arc lint rules scenarios
go test ./internal/app/lint/kernel/ -cover          # RuleDefinitions coverage
go vet ./...
```

---

## Definition of done

- [x] All scenario tests above green
- [x] `parseSkip` table-driven unit test covers every parsing-rule row in data-model.md
- [x] `RuleDefinitions` length matches the number of `Rule` constants, verified by test
- [x] Existing `arc lint` tests (no `--skip` involved) pass unchanged — proves this is additive
- [x] `go vet ./...` clean
- [x] Every new/touched `.go` file carries the license header from `CLAUDE.md`
- [x] `Short`, `Long`, `Example` populated on `arc lint rules`; `arc lint`'s updated to mention `--skip` (Principle XII)
- [x] No inline comments outside GoDoc; no function over 25 lines (Principle IV)
