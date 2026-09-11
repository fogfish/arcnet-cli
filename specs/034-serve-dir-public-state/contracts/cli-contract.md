# CLI Contract: `arc serve` and `arc init`

**Feature**: `034-serve-dir-public-state` | **Date**: 2026-09-11

Only the two commands the scope directive names appear here. Every other command's
contract is unchanged.

---

## C1 — `arc serve [<dir>]`

```
Usage:
  arc serve [<dir>] [flags]

Flags:
  --http string   Serve over Streamable HTTP/SSE at [host]:port instead of stdio
                  (bare port/:port binds loopback only)
```

### Argument

| | |
|---|---|
| Name | `<dir>` |
| Cardinality | 0 or 1 (`cobra.MaximumNArgs(1)`) |
| Default when absent | the process working directory — **behaviour identical to today** (FR-002) |
| Relative input | resolved against the process working directory (FR-003) |
| Absolute input | used verbatim |
| 2+ arguments | Cobra usage error, exit non-zero, no transport opened (FR-007) |

### Preflight (before any transport is opened — FR-004)

Evaluated in order; the first failure returns, prints to stderr, and exits non-zero.

| Condition | Message shape |
|-----------|---------------|
| path does not exist | `<abs path> does not exist` |
| path exists, is not a directory | `<abs path> is not a directory` |
| path is not readable | `<abs path> cannot be read` |
| path is a directory with no `.arc/` | `<abs path> is not an initialized graph` |

All four name the absolute path, never the spelling the user typed, so a relative
argument is unambiguous in the message.

### Session guarantees

- Every tool answers from `<dir>` for the whole session lifetime (FR-005).
- No path outside `<dir>` is served; confinement is structural (`os.DirFS`).
- The argument applies identically to stdio and `--http` (FR-006).
- `serve` remains strictly read-only — unchanged.

### Examples (Cobra `Example` field)

```
arc serve
arc serve ~/graphs/notes
arc serve --http :8080 ~/graphs/notes
```

### MCP client configuration (the motivating case)

```json
{
  "mcpServers": {
    "arc": { "command": "arc", "args": ["serve", "/abs/path/to/graph"] }
  }
}
```

This must work regardless of the working directory the agent host chooses (SC-002).

### Stability

Additive. No existing invocation changes meaning, no flag is renamed, no `--json`
schema is touched. Not a breaking change under Constitution XIV.

---

## C2 — `arc init [<dir>]`

The command's arguments, flags, human output, and `--json` schema
(`kernel.InitResult`: `path`, `commit`, `foldersCreated`, `repository`) are
**unchanged**. Only what lands on disk and what the commit contains changes.

### Layout written

| Path | Content | In the initial commit |
|------|---------|-----------------------|
| `<8 canonical folders>/.gitkeep` | *(empty)* | yes |
| `_schema/**` seed documents | ARCNET-CORE vocabulary | yes |
| `.arc/.gitkeep` | *(empty)* | **yes** — was excluded |
| `.arc/.gitignore` | `cache/` | **yes** — was excluded, content was `*` |
| `.arc/cache/.gitignore` | `*` | **no** — new path, self-ignoring |

### Post-conditions

| # | Guarantee |
|---|-----------|
| P1 | `git status` reports a clean working tree (FR-012) |
| P2 | A clone of the repository is a usable graph with no setup step (FR-013) |
| P3 | Nothing under `.arc/cache/` is ever committed (FR-011) |
| P4 | No ignore rule outside `.arc/` is read, created, or modified (FR-016) |

P4 is unchanged from today and is what allows `--skip-git-init` into a project that
already has its own `.gitignore`.

### New refusal — `.arc/` excluded by the host repository (FR-017)

| | |
|---|---|
| Applies to | `--skip-git-init` only |
| Condition | the parent repository's ignore rules exclude `<graph>/.arc` |
| Timing | before any filesystem write — the target is untouched on refusal |
| Exit | non-zero |
| Message | `<dir>/.arc is excluded by the repository's ignore rules; a clone would not be a usable graph` |

Standalone mode cannot reach this: guard R1 already refuses a target inside an
existing repository.

### Unchanged refusals

`ErrAlreadyInitialized`, `ErrInsideRepository`, `ErrNoParentRepository`,
`ErrTargetIgnored`, `ErrLayoutCollision`, `ErrTargetNotEmpty`, `ErrGitUnavailable` —
all keep their conditions, messages, ordering, and rollback behaviour.

---

## C3 — Backward compatibility

| Situation | Behaviour |
|-----------|-----------|
| Graph created by an earlier version (`.arc/.gitignore` = `*`) | fully usable by every command; no warning, no detection, no conversion (FR-018, FR-020) |
| Existing `arc serve` invocations | byte-for-byte identical (FR-002, SC-004) |
| Existing `arc init` invocations | same flags, same output, same JSON schema; different commit contents |
| Migration | documented in `README.md`; user-initiated (FR-019) |

No code path inspects the *content* of `.arc/.gitignore` — `guardIsGraph` stats the
`.arc/` directory and nothing else — which is what makes legacy support free.

---

## C4 — Contracts explicitly NOT changed

- The MCP tool surface: all eight tools keep their names, input schemas, output
  shapes, descriptions, and session instructions (ADR 003 unaffected).
- `port.VCS` — no method added, removed, or re-signed.
- `fsys.Store` / `fsys.Mounter` — no method added, removed, or re-signed.
- `.arc/config.yml` schema — unchanged; only its version-control status changes.
- Every other command's arguments, flags, and output.
