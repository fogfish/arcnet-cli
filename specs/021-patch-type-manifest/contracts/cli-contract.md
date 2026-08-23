# CLI Contract: commands affected by the manifest identity change

**Feature**: [../spec.md](../spec.md) | **Plan**: [../plan.md](../plan.md) | **Manifest grammar**: [patch-manifest.md](./patch-manifest.md)

Governed by constitution Principles VIII (spec traceability), IX (CLIG/Cobra) and X (output).

**No command, subcommand, flag, alias, or exit-code surface changes.** This contract records
what each affected command must do with the *content* it reads and writes.

---

## 1. Command surface — unchanged

| Command | Role for patches | Change |
|---|---|---|
| `arc apply <patch.md>` | reads one patch | recognition rule (§2 of the manifest contract) |
| `arc apply batch <dir>` | reads many | recognition + classification (FR-008) |
| `arc apply schema <patch.md>\|<url>\|arcnet:<name>` | reads one, three sources | recognition, all three sources |
| `arc subgraph <basename>` | writes one | emission (FR-002) |
| `arc revert <source-id>` | must *skip* patches in-tree | recognition (FR-009, D6) |
| `arc serve` (`subgraph_get`) | writes one | emission, via the same `RenderPatch` |
| `arc lint` | — | **none** (FR-013) |

No `Use`, `Args`, `Aliases`, `Short`, `Long`, or `Example` field is edited by this feature.

---

## 2. Per-command behaviour

### `arc apply <patch.md>`

| Input | stdout | stderr | Exit |
|---|---|---|---|
| `"@type": patch` | ingest summary | — | 0 |
| `"@type": patch`, already applied (FR-012 — idempotency unchanged) | skip message | — | 0 |
| `"@type": patch` + `kind: patch` | ingest summary | — | 0 |
| `"@type": patch` + `kind: source` | — | conflict (3.2) | 1 |
| `kind: patch` | — | retired key (3.3) | 1 |
| `"@type": Patch` | — | casing (3.4) | 1 |
| neither key | — | manifest invalid (3.5) | 1 |
| bare `@type:` | — | quoting (3.1) | 1 |

Every non-zero path leaves the graph and git history untouched (FR-007). Error messages carry
the file path (D5).

### `arc apply batch <dir>`

Classification per file:

| File | Classified as | Counted in |
|---|---|---|
| `"@type": patch`, parses | candidate → applied/skipped | `applied` / `skipped` |
| `"@type": patch`, body malformed | **failed candidate**, named | `failed` |
| `kind: patch` | **failed candidate**, named | `failed` |
| `"@type": patch` + conflicting `kind` | **failed candidate**, named | `failed` |
| no identity key at all | passed over | `not_a_patch` |

A retired-key file **MUST NOT** land in `not_a_patch` (FR-008, SC-005). A non-zero `failed`
count exits 1, as today. `--json` schema is unchanged — only the counts move.

### `arc apply schema`

Identical recognition for all three source forms. The URL and `arcnet:<name>` forms already
wrap parse errors with the source string; the local-file form does too. Remote ARCNET profiles
are published as conformant `"@type": patch` documents, so this is the change that makes them
loadable at all.

### `arc subgraph <basename>`

Emits per [patch-manifest.md §4](./patch-manifest.md#4-emission). `--json` output is
byte-identical to before this feature (FR-011).

### `arc revert <source-id>`

Uses the shared recognition rule to identify exchange files left in the graph tree and skip
them (FR-009, D6). No user-facing output change for any well-formed graph.

### `arc lint`

Unchanged. Patches are not nodes and are never indexed; no rule is added, removed, or
retimed (FR-013). A regression test asserts identical output on a fixture graph.

---

## 3. Scenario → test map (Principle VIII, 1:1)

| Scenario | Assertion | Test file |
|---|---|---|
| **US1.1** `"@type": patch` applies, one ingest commit | nodes written, 1 commit | `cmd/arc/graph/apply_test.go` |
| **US1.2** re-apply is idempotent, no second commit | commit count stable | `cmd/arc/graph/apply_test.go` |
| **US1.3** batch applies all in date-then-path order | order + outcomes | `cmd/arc/graph/batch_test.go` |
| **US1.4** schema patch from file / URL / `arcnet:` | applied, all three | `cmd/arc/ctrl/apply_schema_test.go` |
| **US2.1** emitted manifest's first key is `"@type": patch`, no `kind` | byte inspection of output | `cmd/arc/graph/subgraph_test.go` |
| **US2.2** emitted patch re-applies to a fresh graph | round trip closes | `cmd/arc/graph/subgraph_test.go` |
| **US2.3** `--json` unchanged | schema + values | `cmd/arc/graph/subgraph_test.go` |
| **US3.1** `kind: patch` rejected, names file + replacement | exit 1, message, graph unmodified | `cmd/arc/graph/apply_test.go` |
| **US3.2** conflicting keys rejected, names both values | exit 1, message, graph unmodified | `cmd/arc/graph/apply_test.go` |
| **US3.3** both keys agreeing applies normally | applied | `cmd/arc/graph/apply_test.go` |
| **US3.4** batch names the retired file, never passes it over | `failed` +1, `not_a_patch` unchanged | `cmd/arc/graph/batch_test.go` |
| **US3.5** neither key → pre-existing error verbatim | exact message match | `cmd/arc/graph/apply_test.go` |

### Edge cases → tests

| Edge case | Test file |
|---|---|
| bare unquoted `@type:` → quoting error | `internal/core/markdown_test.go` + `cmd/arc/graph/apply_test.go` |
| `"@type": Patch` capitalized → rejected | `internal/core/markdown_test.go` |
| `kind: source` alone → manifest-invalid, not legacy | `internal/core/markdown_test.go` |
| manifest `@type` vs per-node `@type` in body fences don't interfere | `internal/core/markdown_test.go` |
| retired patch left in graph tree → traceable error, not a node error | `internal/app/graph/service/apply_test.go` |
| `"@type": patch` with a spec-019 casing violation in the body → the *body* error is reported | `cmd/arc/graph/apply_test.go` |

### Unit coverage

| Test | Covers |
|---|---|
| five-case recognition table (`@type` only / `kind` only / both agreeing / both conflicting / neither) | FR-001, FR-003..FR-006, data-model §2 — the table is total, so this is complete |
| `RenderPatch` → `ParsePatch` round trip, extended | FR-002, FR-010 |
| `RenderPatch` output has no `kind` key | FR-002, SC-003 |
| `LooksLikePatch` true for both key forms | FR-008 |
| `ErrIdentityQuoting` text matches `lint`'s `RuleIdentityQuoting` | D3 |

All assertions use `github.com/fogfish/it/v2`; E2E tests drive `RunE` through the existing
`sut()` helper in each command package. Failure-path assertions use `errors.Is` against the
`core` constants, never string matching.
