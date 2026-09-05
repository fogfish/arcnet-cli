# CLI Contract: `arc stats`

**Feature**: 032-arc-stats | **Stability**: human output is not a contract
(Principle XIV); `--json` is — see [stats-json-contract.md](./stats-json-contract.md).

## Surface

```text
arc stats
```

- `Args`: `cobra.NoArgs`. No positional argument, no command-specific flag
  ([research D10](../research.md)).
- Operates on the graph in the current working directory, like `arc lint`.
- Registered in `cmd/arc/root.go` as a flat top-level command.

### Inherited persistent flags

| Flag | Effect on `arc stats` |
|---|---|
| `-v, --verbose` | Computes and renders the detail section (FR-012..FR-017); also enables progress reporting, as in every other command |
| `--json` | Emits the structured document; mirrors verbosity (FR-020a) |
| `-q, --quiet` | Suppresses progress; the report itself is still emitted (FR-022) |
| `-C, --color` | Force-enables color; auto-detected otherwise |

`--plain` is not implemented — no command in this repository implements it
([research D10](../research.md)).

## Required Cobra fields (Principle XII)

`Short`, `Long`, and `Example` MUST all be populated. `Example` MUST show the
three modes that differ in output:

```text
	arc stats
	arc stats --verbose
	arc stats --json
```

## Output routing (Principle X)

- The report → **stdout**.
- Progress steps and errors → **stderr**, via `bios.NewReporter(bios.Quiet, !bios.Verbose)`.
- In `--json` mode nothing but the document reaches stdout (FR-020).

## Exit codes (FR-024, Principle IX)

| Code | Condition |
|---|---|
| `0` | A report was produced — **including** when the graph has broken links, unreadable files, or undeclared predicates |
| non-zero | No report could be produced: the target is not a graph (`service.ErrNotAGraph`), or the graph could not be read |

Broken links are a *reported figure*, not a failure. Gating on graph health
remains `arc lint`'s role (Clarification Q3).

## Read-only guarantee (FR-023, SC-007)

`graph.Stats` takes a `fsys.Mounter` and **no** `port.VCS`. It never calls
`Store.Create` or `Store.Remove`. Read-only is therefore structural — the
service has no writing capability to misuse — not merely a convention.

## Error contract

FR-011's "not a graph" refusal reuses the existing
`internal/app/graph/service.ErrNotAGraph` (`faults` type, Principle XII), so
`arc stats` and every other graph command refuse identically. No new error type
is minted for a condition that already has one.
