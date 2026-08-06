# CLI Contract: `arc apply batch`

**Feature**: [../spec.md](../spec.md) | **Plan**: [../plan.md](../plan.md)

Governed by constitution Principle IX (CLIG/Cobra) and
[ADR 002](../../../adrs/002-ux-design-system.md).

---

## Command surface

```
arc apply batch <dir> [--fail-fast]
```

| Property | Value |
|---|---|
| Parent | `arc apply` (registered beside the existing `arc apply schema`) |
| `Use` | `batch <dir>` |
| `Args` | `cobra.ExactArgs(1)` — one positional subject argument, the patch directory |
| `Aliases` | none |
| `SilenceUsage` | `true` |
| `SilenceErrors` | `true` |

`Short`, `Long`, and `Example` are all populated (Principle XII).

### Positional argument

| Name | Required | Meaning |
|---|---|---|
| `<dir>` | yes | Local directory searched recursively for `*.md` patches (FR-001). Read-only — never modified, moved, or deleted (FR-021). |

Only local paths are accepted. A URL or scheme-prefixed path (`s3://…`,
`https://…`) is **not** supported by this feature; see
[plan.md § Deferred: remote patch sources](../plan.md#deferred-remote-patch-sources).

### Flags

| Flag | Short | Default | Meaning |
|---|---|---|---|
| `--fail-fast` | none | `false` | Halt at the first failing patch instead of continuing through the rest (FR-011). |

No shorthand is assigned: Principle IX reserves `-f` for file/force.

### Inherited persistent flags

`--json`, `--quiet`, `--verbose`, `--no-color` are inherited from the root
command and behave per their established meanings (research.md D9):

| Flag | Effect on this command |
|---|---|
| `--json` | Summary is emitted as [batch-result.schema.md](batch-result.schema.md) on stdout; per-patch progress and hints are suppressed. |
| `--quiet` | Per-patch progress suppressed (FR-015); summary still printed. |
| `--verbose` | Additionally enables `service.Apply`'s own per-node/per-predicate detail for each patch. |

---

## Behavioural contract

### Preflight, before any patch is read (FR-002)

Ordered; the first failure refuses the run with no filesystem or git change:

1. `<dir>` exists → otherwise refuse.
2. `<dir>` is a directory, not a file → otherwise refuse.
3. The working directory is an initialised graph → otherwise refuse.

### Discovery and planning

1. Walk `<dir>` recursively (FR-001).
2. Skip any directory whose name begins with `.` (FR-019).
3. Consider only regular files ending in `.md`; ignore all other files
   entirely — they do not appear in any count.
4. A `.md` file that does not declare a patch manifest is **passed over** and
   counted as `not_a_patch` (FR-003) — never a failure.
5. A `.md` file that declares a patch manifest but does not parse — including
   an absent or uninterpretable `published` date — is a **failed** patch
   (FR-020).
6. Order the remaining patches by `published` ascending, ties broken by
   relative path ascending (FR-004, FR-005). Patches that failed
   classification have no usable date and are appended **last**, ordered by
   path (research.md D5b). The plan is fixed before the first commit.

### Execution

Each planned patch is applied through the unchanged single-patch algorithm
(FR-006), producing exactly one commit (FR-007). An already-tracked document
is skipped with no change (FR-008), evaluated when the patch is reached so a
duplicate later in the same run is recognised (FR-009).

On failure the run continues by default (FR-010), or halts under
`--fail-fast` (FR-011). Either way the failing patch leaves no partial state
and no dangling commit, and earlier commits stand (FR-012).

### Output

**stdout** — the final summary only, so `--json` piping stays clean.

**stderr** — per-patch progress as the run proceeds (FR-015), aggregated
warnings (FR-017), and the closing hint.

Human-mode summary shape (styling via `bios.SCHEMA`, no raw ANSI):

```
✓ Applied 12 patches from ./patches (3 skipped, 1 failed, 2 not a patch)

  failed:
    2026/broken.patch.md — manifest is invalid

  conflicts flagged in: entities/tls.md
```

Nothing-to-apply case (FR-022) exits `0`:

```
✓ No patches to apply in ./patches
```

### Exit codes

| Code | Condition |
|---|---|
| `0` | No patch failed. Includes runs where every patch was skipped and runs where nothing applicable was found (FR-022). |
| `1` | At least one patch failed (FR-014), **or** the preflight refused the run (FR-002). |

A failed-patch exit prints the full summary first, then returns
`bios.ErrSilent` so no second, redundant error line is rendered
(research.md D8). A preflight refusal returns a real `faults`-annotated error
and is rendered by `cmd/arc/main.go` in the normal way.

---

## Use-case contract (`internal/app/graph`)

```go
func ApplyBatch(
    ctx context.Context,
    mounter fsys.Mounter,
    vcs port.VCS,
    reporter bios.Reporter,
    batchReporter bios.Reporter,
    index core.Index,
    schema port.SchemaRegistry,
    dir string,
    patchDir string,
    failFast bool,
) (kernel.BatchResult, error)
```

| Parameter | Role |
|---|---|
| `mounter` | Mounts both the graph root and the patch directory (`fsys.Local{}`). |
| `vcs`, `index`, `schema` | Passed straight through to `service.Apply`. |
| `reporter` | Per-patch **node-level** progress, `--verbose`-gated by the caller. |
| `batchReporter` | Per-patch **outcome** progress, on unless `--quiet` (FR-015). |
| `dir` | Absolute path of the graph root. |
| `patchDir` | Absolute path of the patch directory. |
| `failFast` | Halt at first failure (FR-011). |

Returns a populated `kernel.BatchResult` and a `nil` error for any run that
completed, including one with failed patches — per-patch failures are **data,
not errors** (research.md D7). A non-nil error is returned only for a
preflight refusal (FR-002).

Per ADR 001, this use-case does **not** format terminal output; rendering is
`cmd/arc/graph/batch.go`'s responsibility.

---

## Stability

Additive under Principle XIV: no existing command, flag, or output schema
changes. `arc apply <patch.md>` and `arc apply schema` are untouched.
