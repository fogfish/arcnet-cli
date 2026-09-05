# Quickstart & Validation Guide: `arc stats`

**Feature**: 032-arc-stats | **Plan**: [plan.md](./plan.md)

How to run the feature and how to prove it satisfies the spec. Implementation
belongs in `tasks.md`; this file is the validation contract.

---

## Prerequisites

```bash
cd /Users/kolesnik/devel/go/src/github.com/fogfish/arcnet-cli
go build ./...
```

No credentials, no network, no git history required — `arc stats` reads files
only ([research D10](./research.md)).

---

## Running it

```bash
cd <a graph directory>

arc stats                 # summary
arc stats --verbose       # summary + detail
arc stats --json          # machine-readable summary
arc stats --json -v       # machine-readable summary + detail
arc stats --quiet         # report without progress
```

Against a directory that is not a graph, it refuses and exits non-zero; against
an empty graph it reports zeros and exits 0.

---

## Test fixtures

Under `cmd/arc/graph/testdata/stats/`, each graph has a **known, hand-counted
composition** so every assertion compares against an independently known value
rather than against the implementation's own output (SC-002):

| Fixture | Purpose |
|---|---|
| `known/` | Fixed counts across all types, predicates, periods; the primary assertion target |
| `empty/` | Initialized, no nodes — FR-010 |
| `broken/` | Contains unresolved links, some repeated across nodes — FR-006, C4, C5 |
| `moved/` | Same node set as `known/`, reorganized into non-default folders — FR-004a, C9 |
| `messy/` | Malformed front matter plus an ordinary `README.md` — FR-009, D8 |
| `notgraph/` | No `.arc/` — FR-011 |

`moved/` is the fixture that makes FR-004a testable rather than aspirational:
its expected figures are **identical** to `known/`'s, so any path-derived logic
fails the test.

---

## Acceptance scenario → test map (Principle VIII, 1:1)

Every test is in `cmd/arc/graph/stats_test.go` unless noted, driven through
`RunE` via the existing `sut()` helper.

### User Story 1 — summary

| Scenario | Test | Asserts |
|---|---|---|
| 1.1 reports all five summary figures | `TestStatsReportsSummary` | Each figure equals `known/`'s hand-counted value |
| 1.2 type counts sum to total | `TestStatsTypeCountsSumToTotal` | Invariant C1 |
| 1.3 broken-link count matches lint | `TestStatsBrokenLinksMatchLint` *(in `stats_agreement_test.go`)* | **C5** — runs `Lint` and `Stats` over `broken/`, compares the two numbers |
| 1.4 empty graph reports zeros | `TestStatsEmptyGraph` | C8, exit 0 |
| 1.5 refuses a non-graph | `TestStatsRefusesNonGraph` | `ErrNotAGraph`, non-zero exit |

### User Story 2 — verbose

| Scenario | Test | Asserts |
|---|---|---|
| 2.1 edges by predicate | `TestStatsVerboseByPredicate` | C2 |
| 2.2 monthly ingestion | `TestStatsVerboseMonthlyIngestion` | C3 |
| 2.3 disconnected nodes | `TestStatsVerboseOrphans` | Known orphan count |
| 2.4 names unresolved targets | `TestStatsVerboseUnresolvedTargets` | C4 |
| 2.5 degree and hubs | `TestStatsVerboseDegreeAndHubs` | Avg/median incl. zero-degree nodes; top-10 ordering and tie-break |
| 2.6 schema coverage | `TestStatsVerboseSchemaCoverage` | Declared vs used, undeclared predicates |
| 2.7 content volume | `TestStatsVerboseContentVolume` | Inline refs separate from edges |
| 2.8 default omits detail | `TestStatsDefaultOmitsDetail` | C6 |

### User Story 3 — machine-readable

| Scenario | Test | Asserts |
|---|---|---|
| 3.1 well-formed, stdout only | `TestStatsJSONSummary` | Parses; matches [the contract](./contracts/stats-json-contract.md) |
| 3.2 verbose adds detail | `TestStatsJSONVerbose` | `detail` present and populated |
| 3.3 byte-identical across runs | `TestStatsJSONDeterministic` | **C7** — two runs, `it.Equal` on raw bytes |

### Cross-cutting

| Requirement | Test |
|---|---|
| FR-004a / C9 — folder independence | `TestStatsIgnoresFolderLayout` — asserts `moved/` and `known/` produce identical results |
| FR-009 / D8 — unreadable vs foreign | `TestStatsSeparatesUnreadableFromForeign` |
| FR-023 / SC-007 — read-only | `TestStatsDoesNotModifyGraph` — snapshots the fixture tree before and after |
| D2 — walker refactor is behavior-preserving | The existing Grep/Subgraph/Match tests must pass **untouched** |

Unit tests for the derivation rules (median with even population, tie-breaks,
period-code parsing) live in `internal/app/graph/service/stats_test.go` and
`internal/app/graph/kernel/stats_test.go` against a fake `fsys.Store`.

---

## Validation commands

```bash
go test ./...                                    # full suite
go test ./cmd/arc/graph/ -run TestStats -v       # this feature's E2E
go test ./internal/app/graph/... -cover          # domain coverage
go test ./internal/app/graph/service/ -run 'TestGrep|TestSubgraph|TestMatch'   # D2 refactor safety
go vet ./...
```

### Performance check (SC-004)

```bash
go test ./cmd/arc/graph/ -run TestStatsPerformance -timeout 60s
```

Generates a 10,000-node graph in a temp directory and asserts completion under
5 seconds.

---

## Definition of done

- [ ] All 16 scenario tests green, each asserting a hand-counted value
- [ ] C1–C9 each covered by a named test
- [ ] Existing Grep/Subgraph/Match tests pass with **no edits** (D2 proof)
- [ ] `go vet ./...` clean
- [ ] Every new `.go` file carries the license header from `CLAUDE.md`
- [ ] `Short`, `Long`, `Example` populated on the command (Principle XII)
- [ ] No inline comments outside GoDoc; no function over 25 lines (Principle IV)
