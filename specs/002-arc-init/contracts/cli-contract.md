# CLI Contract: `arc init`

## Usage

```text
arc init [<dir>] [--quiet | -q] [--verbose | -v] [--json] [--color | -C] [--no-color]
```

- `<dir>` — optional single positional argument (DS-02/DS-09: one positional slot, one semantic purpose). Defaults to the current working directory when omitted.
- All flags are the DS-03 persistent root flags, inherited from `cmd/arc/root.go`; `init` introduces no command-local flags.

## Help text (DS-11 shape)

- `Short`: one line, e.g. "Initialize a new, empty knowledge graph."
- `Long`: blank line, then a short paragraph on what gets created and why, ending with `See more info https://github.com/fogfish/arcnet-cli`
- `Example`:
  ```text
  	arc init
  	arc init ./my-graph
  ```

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Graph initialized successfully |
| `1` | Any failure: bad target path, missing `git`, already-initialized graph, non-empty target directory, or a mid-run I/O error (DS-07: single non-zero code, no distinct failure classes needed at this scope) |

## stdout / stderr contract

- **stdout (human mode, default)**: on success, a brief statement of what changed — resolved path and commit hash (constitution Principle X: "successful operations that change state MUST briefly explain what changed"). Example:
  ```text
  ✅ Initialized empty knowledge graph at /Users/dev/my-graph (commit a1b2c3d)
  ```
- **stdout (`--json`)**:
  ```json
  {
    "path": "/Users/dev/my-graph",
    "commit": "a1b2c3d",
    "foldersCreated": ["sources", "entities", "resources", "timeline/yearly", "timeline/monthly", "_meta"]
  }
  ```
  The `commit` field, like the human-mode confirmation line, is always the **short** hash (`git rev-parse --short HEAD`) — never the full 40-character SHA (research.md D2 Bugfix, BUG-001).
- **stderr**: git progress (`Reporter` Start/Done lines) is shown ONLY under `--verbose`/`-v`, styled faint/gray (`SCHEMA.Hint`-equivalent) — not by default, and not in `SCHEMA.StatusOK` green (research.md D2 Bugfix, BUG-001); the final error line (DS-07, only on failure); and the `PostRunE` next-step hint (DS-12, suppressed under `--json`/`--quiet`):
  ```text
  (use "arc apply <patch.md>" to load content into your new graph)
  ```
  `--quiet` suppresses progress and the hint regardless of `--verbose`.
- **stdin**: not read; `init` takes no piped input (DS-09 N/A for this command).

## Error messages (DS-07/XII: human-readable, no raw Go errors)

| Condition | stderr message (example) |
|---|---|
| Target exists and is a file | `❌ /path/to/target is a file, not a directory. Run \`arc help init\` for guidance.` |
| `git` not on PATH | `❌ git is required but was not found on PATH. Install git and try again. Run \`arc help init\` for guidance.` |
| Target already a graph (`.arc/` present) | `❌ /path/to/target is already an initialized graph. Run \`arc help init\` for guidance.` |
| Target non-empty, not a graph | `❌ /path/to/target is not empty. \`arc init\` requires an empty or non-existent directory. Run \`arc help init\` for guidance.` |

## Confirmation and destructiveness (constitution Principle IX)

`arc init` performs no destructive operation on pre-existing content (FR-014/FR-015 refuse rather than overwrite), so no `--yes`/`--force` confirmation flag is needed — this is a pure creation command.

**Bugfix**: 2026-07-02 — BUG-001 revised the stdout/stderr contract: git progress is now `--verbose`-gated (previously shown by default), the commit hash is always short in both human and `--json` output (previously full-length in `--json`), and the `PostRunE` hint now suggests `arc apply <patch.md>` instead of `arc list`.
