# Phase 1 Data Model: `arc apply batch`

**Feature**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Research**: [research.md](research.md)

This feature introduces **no new domain types** in `internal/core`. It adds
one result type and one outcome type to the `graph` use-case's kernel, plus
two internal, non-exported planning structures in the service.

---

## Kernel types (`internal/app/graph/kernel/batch.go`)

These are the use-case's output contract. They are rendered by
`cmd/arc/graph/batch.go` and serialised directly as `--json` (FR-018), so
their JSON tags are a stable, scriptable contract under Principle XIV.

### `Outcome`

The terminal state of one planned patch.

| Value | Meaning | Spec |
|---|---|---|
| `applied` | Patch applied; a commit was produced | FR-006, FR-007 |
| `skipped` | Document already tracked; no filesystem or git change | FR-008 |
| `failed` | Patch could not be applied; reason recorded | FR-010, FR-020 |
| `unprocessed` | Never reached because `--fail-fast` halted the run earlier | FR-011 |

Represented as a string type so the `--json` form is self-describing.

Files that are not patches at all (FR-003) are **not** `Outcome` values — they
never enter the plan. They are counted separately as `not_a_patch` in the
summary, because they are neither an applied unit of work nor a failure.

### `PatchOutcome`

One entry per patch in the plan, in application order.

| Field | JSON | Type | Notes |
|---|---|---|---|
| `Path` | `path` | `string` | Slash-separated path relative to the patch directory. Stable across platforms; the ordering tie-break key (FR-005). |
| `Document` | `document` | `string` | Source citekey from the patch manifest. Empty when the patch failed before its manifest decoded. |
| `Published` | `published` | `time.Time` | Manifest publication date; the primary sort key (FR-004). Zero when classification failed before the manifest decoded — such an entry is ordered after every classified patch (research.md D5b). |
| `Outcome` | `outcome` | `Outcome` | Terminal state, above. |
| `Reason` | `reason` | `string`, omitempty | Human-readable failure explanation. Populated only when `Outcome` is `failed` (FR-013). |
| `CommitHash` | `commit` | `string`, omitempty | Short hash of the commit this patch produced. Populated only when `Outcome` is `applied`. |
| `Created` | `created` | `map[string]int`, omitempty | Node counts by type, newly created. Carried through from `ApplyResult`. |
| `Merged` | `merged` | `map[string]int`, omitempty | Node counts by type, merged into existing nodes. Carried through from `ApplyResult`. |

**Invariants**

- `Reason` non-empty ⟺ `Outcome == failed`.
- `CommitHash` non-empty ⟺ `Outcome == applied`.
- `Created`/`Merged` are empty for every outcome other than `applied`.
- Entries appear in application order, so the slice itself documents the plan
  (FR-004, FR-005): publication date ascending, ties by `Path`, with
  classification failures (zero `Published`) appended last (research.md D5b).

### `BatchResult`

The run-level result returned by `appgraph.ApplyBatch`.

| Field | JSON | Type | Notes |
|---|---|---|---|
| `Directory` | `directory` | `string` | The patch directory, resolved to an absolute path. `ApplyBatch` receives only the resolved `patchDir` (see [contracts/cli-contract.md § Use-case contract](contracts/cli-contract.md)), so echoing the user's literal argument would require a second parameter carrying nothing the absolute form does not already say unambiguously. |
| `Patches` | `patches` | `[]PatchOutcome` | Every planned patch, in application order. |
| `Applied` | `applied` | `int` | Count of `applied` outcomes. |
| `Skipped` | `skipped` | `int` | Count of `skipped` outcomes (FR-008). |
| `Failed` | `failed` | `int` | Count of `failed` outcomes (FR-010). |
| `Unprocessed` | `unprocessed` | `int` | Count of `unprocessed` outcomes; always `0` unless `--fail-fast` halted the run (FR-011). |
| `NotAPatch` | `not_a_patch` | `int` | Markdown files passed over because they do not declare a patch manifest (FR-003). |
| `Conflicts` | `conflicts` | `[]string` | Union of every applied patch's flagged conflict paths, de-duplicated, sorted (FR-016). |
| `Warnings` | `warnings` | `[]string` | Union of every applied patch's warnings, de-duplicated, in first-seen order (FR-017). |

**Invariants**

- `Applied + Skipped + Failed + Unprocessed == len(Patches)`.
- `NotAPatch` counts files that never entered `Patches`, so it is deliberately
  **not** part of that sum.
- `Failed > 0` ⟺ the command returns `bios.ErrSilent` and the process exits
  non-zero (FR-014, research.md D8).
- `len(Patches) == 0 && NotAPatch >= 0` is the FR-022 "nothing to apply"
  case — a success.
- All slice fields are non-nil (empty, not `null`) so the `--json` shape is
  stable for consumers.

---

## Service-internal types (`internal/app/graph/service/batch.go`)

Unexported. Not part of any contract; listed so the implementation's shape is
agreed before coding.

### `candidate`

One discovered `.md` file after classification (research.md D4).

| Field | Type | Notes |
|---|---|---|
| `path` | `string` | Slash-separated, relative to the patch directory. |
| `document` | `string` | From the decoded manifest; empty when classification failed. |
| `published` | `time.Time` | From the decoded manifest; zero when classification failed. |
| `err` | `error` | Non-nil when the file declares `kind: patch` but does not parse (FR-020). |

A file for which `core.LooksLikePatch` returns `false` produces **no**
`candidate` — it only increments the `not_a_patch` counter.

### `plan`

The ordered candidate list plus the passed-over count — the complete input to
execution, fixed before the first commit (research.md D5).

| Field | Type | Notes |
|---|---|---|
| `candidates` | `[]candidate` | Sorted by `published` ascending, then `path` ascending; candidates with a non-nil `err` (and therefore a zero `published`) are appended last, ordered by `path` (research.md D5b). |
| `notAPatch` | `int` | Files passed over during discovery. |

---

## Relationship to existing types

- **`core.Patch`** ([ast.go:111](../../internal/core/ast.go#L111)) — read
  during classification for its `Document` and `Published` fields. Unchanged.
- **`kernel.ApplyResult`** ([apply.go:14](../../internal/app/graph/kernel/apply.go#L14))
  — returned by each `service.Apply` call and projected into one
  `PatchOutcome`: `Skipped` selects the `skipped` outcome, `CommitHash`,
  `Created`, and `Merged` are carried across, and `Conflicts`/`Warnings` are
  folded into the run-level unions. Unchanged.
- **`fsys.Store`** ([types.go:32](../../internal/adapter/fsys/types.go#L32))
  — only its read side (`fs.FS`, `fs.ReadDirFS`) is used, over the patch
  directory (research.md D11). Unchanged.

No migration, no persisted state, no state machine beyond the single
`Outcome` assignment per patch.
