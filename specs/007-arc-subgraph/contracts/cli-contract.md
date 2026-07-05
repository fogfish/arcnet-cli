# CLI Contract: `arc subgraph`

## Usage

```text
arc subgraph <basename> [--depth <n>] [--kind <kind>]... [--tag <tag>]... [--attr <name>=<value>|<name>~=<pattern>]... [--quiet | -q] [--verbose | -v] [--json]
```

- Exactly one positional argument, `<basename>` (Cobra `Args: cobra.ExactArgs(1)`) — the seed node's identity (DS-09: a single positional slot for a single semantic purpose).
- `--depth <n>` is `arc subgraph`'s own local flag, an `int` defaulting to `1`; a negative value is rejected by `opts.build()` (`ErrInvalidDepth`), a non-integer value is rejected by Cobra's own flag parsing before `RunE` ever runs.
- `--kind`/`--tag`/`--attr` are the existing `optsFilter` flags already defined for `arc grep` (research.md D6), reused verbatim — same repeatable, composed semantics (VISION.md Filtering): `--kind` OR'd, `--tag`/`--attr` AND'd, all three groups ANDed together. The filter narrows only the non-seed reachable nodes (spec FR-002/FR-005); omitting all three includes every reachable node within `<n>` hops.
- All other flags are the DS-03 persistent root flags, inherited from `cmd/arc/root.go` — no new global flag is introduced. **No `--color`/color-related behavior applies to this command's own output** (research.md D10) — the flag still exists globally (it affects other commands), but `arc subgraph`'s renderer never reads `bios.SCHEMA`'s styled fields.

## Help text (DS-11 shape)

- `Short`: one line, e.g. "Extract a self-contained subgraph around a node."
- `Long`: blank line, then a short paragraph explaining that `arc subgraph` extracts the seed node plus everything reachable from it within `<n>` hops (both outgoing and incoming structural connections), optionally narrowed by a filter, serialized as a patch-exchange document ready to re-ingest via `arc apply` or paste into an LLM prompt, read-only, ending with `See more info https://github.com/fogfish/arcnet-cli`
- `Example`:
  ```text
  	arc subgraph TLS
  	arc subgraph TLS --depth 2
  	arc subgraph TLS --kind source
  	arc subgraph TLS --json
  ```

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Extraction completed and a subgraph document was printed (the seed is always present — spec FR-002 — so there is no "nothing found" outcome, unlike `arc grep`/`arc lint`; research.md D11) |
| `1` | A refusal: `<basename>` does not identify any node (`ErrSeedNotFound`), `--depth` is invalid (non-integer, rejected by Cobra, or negative, `ErrInvalidDepth`), or the target is not an initialized graph (`ErrNotAGraph`) — printed with a human-readable banner (DS-07) |

Unlike `arc grep`/`arc lint`, `arc subgraph` has no `bios.ErrSilent` "ran, found nothing" path (research.md D11) — every successful run produces a non-empty document.

## stdout / stderr contract

- **stdout (human mode, default — `Human` renderer)**: the exact bytes of `core.RenderPatch(result.Patch)` — a `---`-delimited manifest, then `# <Kind>` / `## <basename>` sections with fenced `yaml` front-matter and verbatim body, no color/styling of any kind (research.md D10), suitable to redirect straight into a file for `arc apply` or paste into an LLM prompt:
  ```text
  ---
  kind: patch
  document: subgraph:transport-layer-security@2026-07-04T12:00:00Z
  published: 2026-07-04
  title: "Subgraph: Transport Layer Security"
  stats: {nodes: 3, directReachable: 2, directIncluded: 2, directTruncated: false, backlinkReachable: 1, backlinkIncluded: 1, backlinkTruncated: false}
  ---
  # Entity

  ## Transport Layer Security
  ```yaml
  id: Transport Layer Security
  category: form structure attribute process
  ```

  TLS is the successor to SSL.

  # Source

  ## rescorla-2026-tls13
  ```yaml
  id: rescorla-2026-tls13
  title: TLS 1.3
  ```

  TLS 1.3 removes support for static RSA key exchange.
  ```
- **stdout (`--verbose`/`-v`)**: identical to `Human` — there is no truncation/highlighting distinction to reveal (unlike `arc grep`'s D11), since this command never styles or shortens its output.
- **stdout (`--json`)**: the generic `jsonPrinter[kernel.SubgraphResult]{}` `bios.Registry` already supplies (no bespoke wiring), regardless of `--verbose`:
  ```json
  {
    "root": "/path/to/graph",
    "seed": "Transport Layer Security",
    "depth": 1,
    "patch": { "document": "subgraph:transport-layer-security@2026-07-04T12:00:00Z", "published": "2026-07-04T00:00:00Z", "title": "Subgraph: Transport Layer Security", "stats": {"...": "..."}, "nodes": [ /* core.Node values */ ] },
    "directReachable": 2, "directIncluded": 2, "directTruncated": false,
    "backlinkReachable": 1, "backlinkIncluded": 1, "backlinkTruncated": false
  }
  ```
- **stderr**: no `bios.Reporter` progress output (a single enumeration + two bounded BFS passes has no meaningful multi-phase progress to narrate, matching `arc grep`'s own precedent); one plain, unstyled diagnostic line when either cap truncated its pool (research.md D10), e.g. `"subgraph: backlink-reachable set truncated to 1024 of 1400 nodes (most-connected kept)\n"`; the refusal-condition error banner (DS-07).
- **stdin**: not read (DS-09 N/A).

## Error messages (DS-07/XII: human-readable, no raw Go errors)

| Condition | stderr message (example) |
|---|---|
| `<basename>` does not identify any node | `❌ no node found with basename "TLS 1.4". Run \`arc help subgraph\` for guidance.` |
| `--depth` is negative | `❌ --depth "-1" must be a non-negative integer. Run \`arc help subgraph\` for guidance.` |
| `--depth` is not an integer | Cobra's own flag-parsing error, e.g. `invalid argument "two" for "--depth" flag: strconv.ParseInt: parsing "two": invalid syntax` |
| Target not an initialized graph | `❌ /path/to/target is not an initialized graph. Run \`arc init\` first, or \`arc help subgraph\` for guidance.` |
| Malformed `--attr` value | `❌ --attr "status" must be name=value or name~=pattern. Run \`arc help subgraph\` for guidance.` (identical to `arc grep`'s existing message, same underlying `ErrInvalidAttrFlag`) |

A dangling link target (points to no existing node) is **never** an error condition — it is silently excluded from the traversal (spec FR-006, Edge Cases).

## Confirmation and destructiveness (constitution Principle IX)

`arc subgraph` never modifies the graph or its git history under any circumstance (spec FR-009) — strictly read-only, the same guarantee `arc lint`/`arc grep` already provide. No `--yes`/`--force` confirmation flag is applicable.
