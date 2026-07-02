# CLI Contract: `arc apply`

## Usage

```text
arc apply <patch.md> [--quiet | -q] [--verbose | -v] [--json] [--color | -C] [--no-color]
```

- `<patch.md>` — mandatory single positional argument (DS-02/DS-09: one positional slot, one semantic purpose), the path to the patch file to apply. No default; `arc apply` with no argument prints help and exits non-zero (Cobra `Args: cobra.ExactArgs(1)`).
- All flags are the DS-03 persistent root flags, inherited from `cmd/arc/root.go`; `apply` introduces no command-local flags (research.md D9 — `--dry-run`/`--batch` are separate, out-of-scope VISION.md commands).

## Help text (DS-11 shape)

- `Short`: one line, e.g. "Apply a document patch to the graph."
- `Long`: blank line, then a short paragraph on what gets created/merged and the one-commit guarantee, ending with `See more info https://github.com/fogfish/arcnet-cli`
- `Example`:
  ```text
  	arc apply rescorla-2026-tls13.patch.md
  ```

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Applied successfully, or skipped because the document was already tracked (both are non-error outcomes) |
| `1` | Any failure: malformed patch, target not an initialized graph, unrecognized node kind, malformed `.arc/config.yml`, mid-run I/O error (DS-07: single non-zero code, no distinct failure classes needed at this scope) |

## stdout / stderr contract

- **stdout (human mode, default)**, on success:
  ```text
  ✅ Applied rescorla-2026-tls13: +1 source, +2 entities, +1 resource (commit a1b2c3d)
  ```
  On idempotent skip (FR-003):
  ```text
  ✅ rescorla-2026-tls13 is already tracked — nothing to do
  ```
  (constitution Principle X: "successful operations that change state MUST briefly explain what changed" — the skip line explains why nothing changed, the same spirit for a no-op success.)
- **stdout (`--json`)**:
  ```json
  {
    "document": "rescorla-2026-tls13",
    "skipped": false,
    "created": {"source": 1, "entity": 2, "resource": 1},
    "merged": {"entity": 1},
    "conflicts": [],
    "commit": "a1b2c3d",
    "timeline": ["2026", "2026-04"]
  }
  ```
  `commit` is always the **short** hash (`git rev-parse --short HEAD`), matching `arc init`'s established convention (`specs/002-arc-init/research.md` D2 Bugfix).
- **stderr**: `Reporter` progress (research.md D9 labels) shown ONLY under `--verbose`/`-v`, styled faint/gray, matching `arc init`'s BUG-001-fixed convention exactly (never `SCHEMA.StatusOK` green); the final error line (DS-07, only on failure); and a conditional `PostRunE` hint (DS-12) naming any conflicted file(s), e.g.:
  ```text
  (a merge conflict was flagged in entities/Transport Layer Security.md — resolve it manually before the next apply)
  ```
  No hint is printed when there are no conflicts. `--quiet` suppresses progress and the hint regardless of `--verbose`; hints are suppressed under `--json` (DS-12).
- **stdin**: not read; the patch is always a named file argument, never piped (DS-09 N/A — CORE §12 patches are shareable files, not a stream).

## Error messages (DS-07/XII: human-readable, no raw Go errors)

| Condition | stderr message (example) |
|---|---|
| Patch manifest missing a mandatory field | `❌ patch manifest is missing a mandatory field (kind: patch, document, published). Run \`arc help apply\` for guidance.` |
| Patch body malformed | `❌ patch body does not follow the H1-kind/H2-node section structure. Run \`arc help apply\` for guidance.` |
| Target not an initialized graph | `❌ /path/to/target is not an initialized graph. Run \`arc init\` first, or \`arc help apply\` for guidance.` |
| Unrecognized node kind | `❌ hypothesis is not a recognized node kind for this graph. Register it in .arc/config.yml first. Run \`arc help apply\` for guidance.` |
| Malformed `.arc/config.yml` | `❌ .arc/config.yml is not valid YAML. Run \`arc help apply\` for guidance.` |

## Confirmation and destructiveness (constitution Principle IX)

`arc apply` never deletes or silently overwrites existing content — creation is additive, merges are commutative/idempotent (CORE §10), and a scalar conflict is flagged rather than overwritten (FR-013). No `--yes`/`--force` confirmation flag is needed.
