# Phase 2 Design Preconditions — Sign-off

**Feature**: `021-patch-type-manifest` | **Tasks**: [tasks.md](./tasks.md) | **Date**: 2026-08-23

The constitution's Compliance Checklist PRECONDITIONS require a *recorded decision* for each
design gate, not working code. This file is that record for T006, T007, T008, T009 and T017.

---

## T006 — No new domain type (Principles II, V; research [D2](./research.md))

**Confirmed.** `core.Patch` (`internal/core/ast.go:111`) is unchanged:

```go
type Patch struct {
    Document  string         `json:"document"`
    Published time.Time      `json:"published"`
    Title     string         `json:"title,omitempty"`
    Stats     map[string]any `json:"stats,omitempty"`
    Nodes     []Node         `json:"nodes"`
}
```

No field is added, removed, or retagged, and no new type is introduced anywhere. The manifest
identity is a *recognition predicate* consumed during decode, never carried data — which is
what makes FR-011 (`arc subgraph --json` unaffected) true **by construction** rather than by a
`json:"-"` tag. The `Deprecations []string` carrier sketched in `CORE-FIX.md` §3.7 is rejected
(D2): with no transitional acceptance there is no notice to carry.

The only additions to `internal/core` are four package-level `faults` error constants in
`errors.go` and one unexported helper (`patchManifestType`) plus one unexported detector in
`markdown.go`. Errors already cross the hexagonal boundary freely, so Principle III's
"`internal/core` must not import `internal/bios`" constraint is satisfied without new plumbing.

## T007 — Command and flag surface provably unchanged (Principle IX)

**Confirmed.** No command, subcommand, flag, alias, `Args`, `Use`, `Short`, `Long`, `Example`,
or exit-code surface changes. `cmd/arc/root.go` and every `cmd/**/*.go` **production** file are
untouched by this feature; `cmd/` appears in the change set only as E2E test and `testdata/`
fixture surface.

Verified: no `cmd/**/*.go` non-test file references the patch manifest identity key at all —
the only `kind` occurrences under `cmd/` are unrelated local identifiers
(`pluralizeKind`/`pluralizeSchemaKind` parameters) and `arc lint`'s own help text about
front-matter/extension *node* kinds, neither of which names the patch manifest.

Exit codes are unchanged: every new rejection path returns an error through the existing
`RunE` → `bios` path, which already maps a non-nil error to stderr + exit `1`.

## T008 — Contract completeness sign-off (FR-001..FR-016)

**Confirmed complete.** [contracts/patch-manifest.md](./contracts/patch-manifest.md) is
normative for the format; [contracts/cli-contract.md](./contracts/cli-contract.md) is normative
for per-command behaviour and carries the 1:1 scenario→test map required by Principle VIII.

| FR | Where the contract pins it |
|---|---|
| FR-001 | patch-manifest §1 `@type`, §2 row 2 |
| FR-002 | patch-manifest §4 Emission (first key, quoted, no `kind`) |
| FR-003 | patch-manifest §2 row 4, §3.3; cli-contract §2 `arc apply` |
| FR-004 | patch-manifest §2 row 2 (`kind` ignored when agreeing) |
| FR-005 | patch-manifest §2 row 1, §3.2 |
| FR-006 | patch-manifest §2 row 5, §3.5 ("wording unchanged") |
| FR-007 | patch-manifest §3 preamble; cli-contract §2 `arc apply` |
| FR-008 | cli-contract §2 `arc apply batch` classification table |
| FR-009 | cli-contract §1 command table + §2 `arc revert`; patch-manifest §2 recognition-vs-acceptance |
| FR-010 | patch-manifest §4 guarantees |
| FR-011 | patch-manifest §5 Stability; cli-contract §2 `arc subgraph` |
| FR-012 | cli-contract §2 `arc apply` (already-applied row) |
| FR-013 | cli-contract §1 + §2 `arc lint` |
| FR-014 | research [D7](./research.md) migration classification; Polish tasks T051–T055 |
| FR-015 | patch-manifest §1 key form, §3.1 |
| FR-016 | patch-manifest §1 value, §2 row 3, §3.4 |

Missing FR labels on four contract sections were added during this sign-off; no substantive
gap was found.

## T009 — External integration & adapter design (Principle VII)

**N/A.** No external system is integrated, and no adapter is added or modified. All filesystem
I/O continues through the existing `internal/adapter/fsys` port; no `os.*` file call is
introduced anywhere by this feature. `internal/adapter/http` (used by `arc apply schema` for
URL and `arcnet:` sources) is untouched — only the *content* it fetches is now recognized.

## T017 — Configuration & secrets review (Principle XI)

**N/A.** No configuration value, environment variable, XDG path, or secret is introduced, read,
or logged by this feature. Precedence rules and `internal/app/config` are untouched.

---

# Phase N — Constitution Compliance Verification

Recorded 2026-08-23, after implementation. Each gate below was checked, not assumed.

## Design Phase

| Gate | Result |
|---|---|
| **TN01** ARCHITECTURE.md reflects architectural changes | **PASS** — the **Patch** glossary row (`ARCHITECTURE.md:244`) now names `"@type": patch` as the identity declaration and records the retired key's refuse-only status; the **Batch Plan** row (`:273`) restates candidate recognition in terms of `core.LooksLikePatch` and states that a retired-key file is always a failed candidate. No other architectural change exists to reflect. |
| **TN02** Domain concepts in the Glossary | **PASS** — no new domain concept; the two affected rows were corrected (T004/T005). |
| **TN03** Command/flag surface matches the Phase 2b design | **PASS** — `git status --porcelain cmd/` lists only `_test.go` files and `testdata/`. Not one `cmd/**` production file was touched, so no `Use`, `Args`, `Aliases`, `Short`, `Long`, `Example`, flag or exit-code changed. |

## Implementation Phase

| Gate | Result |
|---|---|
| **TN04** ADRs for new architectural patterns | **N/A — none introduced.** This feature implements an external specification (ARCNET-CORE §14.2.1); it decides no architecture. ADRs 001/002/003 were reviewed against the implementation and none is contradicted. |
| **TN05** Ports/interfaces; Cobra wiring and adapters separated | **PASS** — `go list -deps ./internal/core` resolves to `internal/core` alone: no `cobra`, no `internal/bios`, no `cmd`. The new failure modes cross the boundary as `faults` error values, and the file path is attached by the service-layer caller that knows it (`readPatch`). |
| **TN06** Unit tests written first, compiled, failed semantically | **PASS** — T027 gate. After the fixture migration and before any `markdown.go` edit, `go build ./...` succeeded and the suite failed with `manifest is missing a mandatory field or uses the pre-0.5 node format` on `"@type"`-keyed input — a recognition failure, not a compile error. |
| **TN07** `github.com/fogfish/it/v2` used exclusively | **PASS** — `grep -rn testify cmd/ internal/` is empty; every new assertion goes through `it.Then(t)`. Failure paths assert with `errors.Is` against the `internal/core` constants, never string matching (message-text assertions are additive, on top of an `errors.Is` gate). |
| **TN08** No Bash used for unit-level correctness validation | **PASS** — shell was used only for the quickstart walkthrough (T056), the `--json` golden (T002/T039) and repo-wide greps (T055). Every correctness claim is carried by a Go test. |
| **TN09** External integrations follow port/adapter | **N/A** — no external system integrated, no adapter added or modified, no `os.*` file call introduced. |
| **TN10** TTY/`NO_COLOR`/`--quiet`/`--verbose`/lipgloss | **PASS by non-change** — no rendering code was touched. New errors reach the user through the existing `bios` error path: stderr, exit `1`, unchanged banner. Confirmed in the T056 walkthrough. |
| **TN11** Configuration precedence, XDG, no secrets | **N/A** — no configuration value, environment variable or secret introduced or read. |
| **TN12** Help text populated for new/changed commands | **N/A** — no command is new or changed (TN03). |
| **TN13** Phase 2d E2E tests turned GREEN, changed minimally | **PASS** — the full suite is green. Exactly two pre-existing expectations changed, each forced by a deliverable: the batch counts (`failed` 1→2, `patches` 5→6) for the new FR-008 fixture, and one `strings.Count(raw, "kind:")` 1→0 for the emission change. No existing assertion was loosened or deleted. |
| **TN14** Every spec.md scenario has a passing, colocated E2E test | **PASS** — all 12 scenarios map 1:1 to named tests in `cmd/arc/graph/{apply,batch,subgraph}_test.go` and `cmd/arc/ctrl/apply_schema_test.go`, per [contracts/cli-contract.md §3](./contracts/cli-contract.md). Presence and pass verified by name. |
| **TN15** Release/versioning impact | **CONFIRMED BREAKING — deviation still valid.** This changes a script-consumed format without the Principle XIV deprecation window and without a major bump. Both deviations remain justified as recorded in [plan.md § Complexity Tracking](./plan.md#complexity-tracking), and the justification was re-checked at implementation time and strengthened by evidence: the `0.1.12` binary refuses the live published profile at `fogfish/arcnet-spec/schema/core.md` with `manifest is missing a mandatory field` (T056 Scenario 9), so there is no working ecosystem to deprecate out of. The two contracts the constitution names as stable are provably unaffected — `arc subgraph --json` diffs byte-identical against the pre-change golden modulo its own run-time timestamp (T039), and `arc apply batch --json`'s schema is unchanged (counts only). |

## Quality gates (T057)

- `go build ./...` — clean
- `go vet ./...` — clean
- `staticcheck ./...` — clean (no findings, matching the T003 baseline)
- `gofmt -l cmd internal` — empty
- `go test ./... -cover` — all packages pass. `internal/core` **88.5% → 89.0%**; `cmd/arc/graph` 85.9%, `cmd/arc/ctrl` 87.1% unchanged; `internal/app/graph/service` 77.2% → 77.1% (denominator shift from the added guard branch; no test removed). No material regression.
