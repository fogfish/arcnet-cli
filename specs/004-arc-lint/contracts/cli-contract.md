# CLI Contract: `arc lint`

## Usage

```text
arc lint [--quiet | -q] [--verbose | -v] [--json] [--color | -C] [--no-color]
```

- No positional arguments — `arc lint` always operates on the current directory's graph (Cobra `Args: cobra.NoArgs`), the same convention `arc apply`/`arc init`'s no-argument form uses.
- All flags are the DS-03 persistent root flags, inherited from `cmd/arc/root.go`; `lint` introduces no command-local flags (research.md D14 — `--verbose`/`-v` is reused exactly as the user's instruction specifies, not a new flag).

## Help text (DS-11 shape)

- `Short`: one line, e.g. "Validate the graph against the CORE §14 conformance checklist."
- `Long`: blank line, then a short paragraph on what gets checked and that lint is read-only, ending with `See more info https://github.com/fogfish/arcnet-cli`
- `Example`:
  ```text
  	arc lint
  	arc lint --verbose
  	arc lint --json
  ```

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Zero violations found — the graph is fully conformant |
| `1` | One or more violations found (FR-016), **or** a refusal condition (target not an initialized graph) — DS-07: a single non-zero code, no distinct failure classes needed at this scope, matching `arc apply`'s existing convention |

## stdout / stderr contract

- **stdout (human mode, default — `Human` renderer, research.md D14)**: only nodes carrying a violation are listed, each with its rule(s), file, and line, followed by one summary line:
  ```text
  ❌ entities/Transport Layer Security.md:12 — [linkResolves] target "TLS 1.3" does not exist
  ❌ sources/rescorla-2026-tls13.md — [ingestCommit] 0 matching commits found for this document
  ❌ 2 nodes checked, 0 passing, 2 failing
  ```
  On a fully conformant graph:
  ```text
  ✅ 42 nodes checked, 42 passing, 0 failing
  ```
- **stdout (`--verbose`/`-v` — `Verbose` renderer)**: every enumerated node is listed, in walk order, with its individual pass/fail status, followed by the identical summary line:
  ```text
  ✅ sources/rescorla-2026-tls13.md
  ❌ entities/Transport Layer Security.md:12 — [linkResolves] target "TLS 1.3" does not exist
  ✅ resources/rfc8446.md
  ❌ 3 nodes checked, 1 passing, 2 failing
  ```
- **stdout (`--json`)**: the generic `jsonPrinter[kernel.LintResult]{}` `bios.Registry` already supplies (no bespoke wiring), regardless of `--verbose`:
  ```json
  {
    "root": "/path/to/graph",
    "nodes": [
      {
        "path": "sources/rescorla-2026-tls13.md",
        "id": "rescorla-2026-tls13",
        "kind": "source",
        "violations": [
          {"rule": "ingestCommit", "path": "sources/rescorla-2026-tls13.md", "line": 0, "message": "0 matching commits found for this document", "relatedPaths": []}
        ]
      }
    ],
    "violations": [
      {"rule": "ingestCommit", "path": "sources/rescorla-2026-tls13.md", "line": 0, "message": "0 matching commits found for this document", "relatedPaths": []}
    ],
    "passing": 0,
    "failing": 1
  }
  ```
- **stderr**: `Reporter` progress (research.md D14's four labels) shown ONLY under `--verbose`/`-v`, styled faint/gray, matching `arc init`/`arc apply`'s established convention exactly (never `SCHEMA.StatusOK` green); the final error line (DS-07, only on a refusal condition); no `PostRunE` hint is defined for this command in this iteration — a violation list is already the actionable next step, and DS-12 does not mandate a hint on every command.
- **stdin**: not read (DS-09 N/A — `lint` takes no file input, it reads the mounted graph directly).

## Error messages (DS-07/XII: human-readable, no raw Go errors)

| Condition | stderr message (example) |
|---|---|
| Target not an initialized graph | `❌ /path/to/target is not an initialized graph. Run \`arc init\` first, or \`arc help lint\` for guidance.` |
| `_meta/predicates.md` exists but is unreadable (real I/O failure, not "file absent") | `❌ failed to read _meta/predicates.md. Run \`arc help lint\` for guidance.` |

A per-node conformance defect (malformed front-matter, unresolved link, unregistered predicate, etc.) is **never** an error condition — it is a `Violation` entry in the result, reported as described above, never aborting the run (spec FR-013).

## Confirmation and destructiveness (constitution Principle IX)

`arc lint` never modifies the graph or its git history under any circumstance (spec FR-014) — it is the first purely read-only graph-inspecting command in this codebase. No `--yes`/`--force` confirmation flag is applicable.
