# CLI Contract: `arc apply schema`

## Usage

```text
arc apply schema <patch.md> | <url> | arcnet:<name> [--timeout <duration>] [--quiet | -q] [--verbose | -v] [--json] [--color | -C] [--no-color]
```

- `<patch.md> | <url> | arcnet:<name>` — mandatory single positional argument (one positional slot, one semantic purpose, Principle IX): a local path to a patch document, an `http(s)://` URL to fetch one from, or an `arcnet:`-prefixed short reference into the official arcnet extensions catalog (research.md D1a) — `arcnet:<name>` resolves to `https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/schema/<name>` and is fetched exactly like a directly supplied URL. No default; running with no argument prints help and exits non-zero (Cobra `Args: cobra.ExactArgs(1)`).
- `--timeout` — command-local flag, default `30s`: the maximum time allowed to fetch a URL or `arcnet:`-resolved input (research.md D2). Ignored for a local-file input.
- All other flags are the existing persistent root flags inherited from `cmd/arc/root.go`.

## Help text (Principle XII shape)

- `Short`: "Import Property/Class schema definitions from a patch document."
- `Long`: a short paragraph on the schema-only restriction and the all-or-nothing validation guarantee, ending with `See more info https://github.com/fogfish/arcnet-cli`
- `Example`:
  ```text
  	arc apply schema arcnet-ext-media.schema.md
  	arc apply schema https://example.org/schemas/arcnet-ext-media.schema.md
  	arc apply schema arcnet:media.schema.md
  ```

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Every `Property`/`Class` definition in the patch was created or merged (or the patch was a no-op re-apply, per research.md D7) |
| `1` | Any failure: patch contains a disallowed node type (FR-005), patch fails to parse, target not an initialized graph, fetch failure/timeout, malformed `Property`/`Class` document |

## stdout / stderr contract

- **stdout (human mode, default)**, on success:
  ```text
  ✅ Applied arcnet-ext-media.schema.md: +3 predicates, +1 type (commit a1b2c3d)
  ```
  On a no-op re-apply:
  ```text
  ✅ arcnet-ext-media.schema.md introduced no schema changes — nothing to commit
  ```
- **stdout (`--json`)**:
  ```json
  {
    "source": "arcnet-ext-media.schema.md",
    "created": {"predicate": 3, "type": 1},
    "merged": {},
    "commit": "a1b2c3d"
  }
  ```
  `commit` is the short hash, empty string on a no-op. `created`/`merged` keys are always present (`"predicate"`, `"type"`), zero-valued when nothing of that kind changed.
- **stderr**: `Reporter` progress under `--verbose`/`-v` only (matching `arc apply`'s existing convention); the final human-readable error line on failure (Principle XII — never a raw Go error or stack trace).
- **stdin**: not read.

## Error messages (Principle XII: human-readable, no raw Go errors)

| Condition | stderr message (example) |
|---|---|
| Disallowed node type present (FR-005/FR-006) | `❌ patch node "acme-widget" has type "entity" — arc apply schema only accepts Property and Class nodes. Run \`arc help apply schema\` for guidance.` |
| Patch fails to parse | `❌ arcnet-ext-media.schema.md does not parse as a patch document. Run \`arc help apply schema\` for guidance.` |
| Target not an initialized graph | `❌ /path/to/target is not an initialized graph. Run \`arc init\` first, or \`arc help apply schema\` for guidance.` |
| URL fetch failure/timeout | `❌ failed to fetch https://example.org/schemas/arcnet-ext-media.schema.md: <cause>` |
| Empty `arcnet:` reference | `❌ "arcnet:" must be followed by a catalog path, e.g. arcnet:media.schema.md. Run \`arc help apply schema\` for guidance.` |
| Malformed `Property`/`Class` document | `❌ node "acme:size" has a missing or invalid role. Run \`arc help apply schema\` for guidance.` |

## Confirmation and destructiveness (Principle IX)

`arc apply schema` never deletes existing schema documents; creation is
additive and merges follow each definition's own declared merge behavior
(never a silent overwrite of a scalar value already set — same guarantee
`arc apply` already provides for content nodes). No `--yes`/`--force`
confirmation flag is needed.
