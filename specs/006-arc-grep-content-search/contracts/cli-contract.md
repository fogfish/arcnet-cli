# CLI Contract: `arc grep`

## Usage

```text
arc grep [--kind <kind>]... [--tag <tag>]... [--attr <name>=<value>|<name>~=<pattern>]... [--quiet | -q] [--verbose | -v] [--json] [--color | -C] [--no-color] <pattern>
```

- Exactly one positional argument, `<pattern>` (Cobra `Args: cobra.ExactArgs(1)`) — a regexp, matched against every scanned line (DS-09: a single positional slot for a single semantic purpose).
- `--kind`/`--tag`/`--attr` are `arc grep`'s own local flags (research.md D14), repeatable, composed per VISION.md's Filtering section (research.md D8): `--kind` is OR'd, `--tag` and `--attr` are AND'd, all three groups are ANDed together. Omitting all three scans the whole graph (spec FR-002).
- All other flags are the DS-03 persistent root flags, inherited from `cmd/arc/root.go` — no new global flag is introduced.

## Help text (DS-11 shape)

- `Short`: one line, e.g. "Search node content for lines matching a pattern."
- `Long`: blank line, then a short paragraph explaining that `arc grep` scans node file content (not just front-matter) for a regexp, optionally narrowed by a filter, read-only, ending with `See more info https://github.com/fogfish/arcnet-cli`
- `Example`:
  ```text
  	arc grep TLS
  	arc grep --kind source TLS
  	arc grep --tag cryptography --attr status=mature "TLS 1\.3"
  	arc grep --json TLS
  ```

## Exit codes

| Code | Meaning |
|---|---|
| `0` | One or more matches found |
| `1` | Zero matches found (research.md D12 — `bios.ErrSilent`, result already printed, no second error line), **or** a refusal condition (invalid `<pattern>`, target not an initialized graph — printed with a human-readable banner) |

A script distinguishes "ran, found nothing" from "refused to run" by the presence of an error banner on `stderr`, not by a distinct exit-code value — matching `arc lint`/`arc apply`'s existing convention exactly (research.md D12).

## stdout / stderr contract

- **stdout (human mode, default — `Human` renderer)**: one line per match, `<kind>  <id>  <line>  <text>`, whitespace-delimited, no header/footer/summary line mixed in (spec FR-006/FR-007):
  ```text
  source  rescorla-2026-tls13  42  TLS 1.3 removes support for static RSA key exchange.
  entity  Transport Layer Security  3  TLS is the successor to SSL.
  ```
  When `bios.SCHEMA` is the color schema (an interactive TTY, no `NO_COLOR`/`TERM=dumb`, or `--color` forced — ADR 002 DS-05), the matched span within `<text>` is rendered with `SCHEMA.Match`, and a line longer than the configured `maxLineWidth` (default 80, `.arc/config.yml` `grep.maxLineWidth` — research.md D10/D11) is ellipsis-fitted around the match (`…` prepended/appended as needed). Neither transform changes the underlying data — piped or `SCHEMA_PLAIN` output is always the full, untruncated, unstyled line.
  Zero matches produces no stdout lines at all (spec FR-009) — no "0 matches found" banner on stdout (that would violate FR-007's "no header/footer/summary" rule); the empty result is signaled purely through the exit code (research.md D12).
- **stdout (`--verbose`/`-v` — `Verbose` renderer)**: identical row format to `Human`, but truncation (research.md D11) is disabled — the full line is always shown (still colorized when `SCHEMA` is the color schema), matching DS-03's "reveals additional diagnostic detail."
- **stdout (`--json`)**: the generic `jsonPrinter[kernel.GrepResult]{}` `bios.Registry` already supplies (no bespoke wiring), regardless of `--verbose`:
  ```json
  {
    "root": "/path/to/graph",
    "pattern": "TLS",
    "matches": [
      {"kind": "source", "id": "rescorla-2026-tls13", "path": "sources/rescorla-2026-tls13.md", "line": 42, "text": "TLS 1.3 removes support for static RSA key exchange.", "start": 0, "end": 3}
    ],
    "unreadable": []
  }
  ```
- **stderr**: no `bios.Reporter` progress output for this command (a single-pass scan has no meaningful multi-phase progress to narrate, unlike `arc apply`/`arc lint`); one optional `PostRunE` hint (DS-12) suggesting a `--kind` filter when a large, unfiltered result is returned; the refusal-condition error banner (DS-07).
- **stdin**: not read (DS-09 N/A — `arc grep` reads the mounted graph directly, not piped input).

## Error messages (DS-07/XII: human-readable, no raw Go errors)

| Condition | stderr message (example) |
|---|---|
| `<pattern>` is not a valid regexp | `❌ "[TLS" is not a valid pattern: missing closing ]. Run \`arc help grep\` for guidance.` |
| Target not an initialized graph | `❌ /path/to/target is not an initialized graph. Run \`arc init\` first, or \`arc help grep\` for guidance.` |
| Malformed `--attr` value (neither `name=value` nor `name~=pattern`) | `❌ --attr "status" must be name=value or name~=pattern. Run \`arc help grep\` for guidance.` |

A node file that cannot be read, or does not parse as a valid node, is **never** an error condition — it is recorded in `GrepResult.Unreadable` and the scan continues (spec FR-012); it is not printed to `stdout` (which is match rows only) and is surfaced only in `--json` output's `unreadable` array, consistent with FR-007's "no header/footer/summary" rule for the human/verbose renderers.

## Confirmation and destructiveness (constitution Principle IX)

`arc grep` never modifies the graph or its git history under any circumstance (spec FR-010) — strictly read-only, the same guarantee `arc lint` already provides. No `--yes`/`--force` confirmation flag is applicable.
