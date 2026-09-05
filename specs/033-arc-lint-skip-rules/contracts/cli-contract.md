# CLI Contract: `arc lint --skip`, `arc lint rules`

**Feature**: 033-arc-lint-skip-rules | **Stability**: human output is not a
contract (Principle XIV); `--json` is. This extends the existing
[004-arc-lint CLI contract](../../004-arc-lint/contracts/cli-contract.md)
rather than replacing it — everything not mentioned here (base exit codes,
the human/verbose report shape, the `--json` `LintResult` schema) is
unchanged.

## Surface

```text
arc lint [--skip <rule[,rule...]>] [--quiet | -q] [--verbose | -v] [--json] [--color | -C] [--no-color]
arc lint rules [--json]
```

- `arc lint` gains one new command-local flag, `--skip`, a string whose
  value is a comma-separated list of rule names (FR-001). No shorthand — not
  among the reserved single-letter conventions (Principle IX).
- `arc lint rules` is a new Cobra child of `arc lint`
  ([research D5](../research.md)), `Args: cobra.NoArgs`, taking no
  command-specific flag. It inherits the same persistent root flags as every
  command but only `--json` changes its output (there is no verbose form,
  [research D6](../research.md)).
- `arc lint rules` requires no initialized graph in the working directory and
  performs no filesystem walk of one (FR-011).

## Help text (DS-11 shape)

`arc lint`'s existing `Long` text gains one sentence describing `--skip`, and
its `Example` block gains one line:

```text
	arc lint
	arc lint --verbose
	arc lint --json
	arc lint --skip ingestCommit
```

`arc lint rules`:
- `Short`: one line, e.g. "List every conformance rule arc lint implements."
- `Long`: a short paragraph stating this is reference data — the same rule
  identifiers `arc lint --skip` accepts — and that it requires no graph.
- `Example`:
  ```text
  	arc lint rules
  	arc lint rules --json
  ```

## `arc lint --skip` behavior

| Condition | Result |
|---|---|
| `--skip` omitted, or `""` | Identical to today's `arc lint` — no filtering |
| `--skip` names one or more known rules | Every violation of a named rule is absent from the report, in every output mode; `Passing`/`Failing`/exit status reflect only the remaining rules (FR-004, FR-005) |
| `--skip` names every rule the tool implements | The run still enumerates nodes and succeeds, reporting `N nodes checked, N passing, 0 failing` (FR-006) |
| `--skip` names one or more unrecognized rules | The command refuses (see Error messages below), before resolving or requiring a graph (FR-007, FR-008) |

Validation runs before `fsys.Local{}.Mount`/`appschema.Resolve` are called, so
an invalid `--skip` value is caught even against a directory that is not a
graph at all (FR-008) — this is an intentional change to `RunE`'s existing
ordering.

## Error messages (DS-07/XII: human-readable, no raw Go errors)

| Condition | stderr message (example) |
|---|---|
| `--skip` names one or more rules `arc lint rules` does not list | `❌ unknown rule(s) in --skip: bogusRule — run \`arc lint rules\` to see valid rule names` |

Multiple unrecognized names are joined with `, ` into the single message
(FR-007's "identifying every unrecognized name").

## Exit codes

`arc lint`'s existing table (004's contract) gains one new non-zero case:

| Code | Meaning |
|---|---|
| `0` | Zero violations found among the rules that were not skipped |
| `1` | One or more (non-skipped) violations found, **or** a refusal condition — target not an initialized graph, **or `--skip` named an unrecognized rule** |

`arc lint rules` always exits `0` — it is static reference data with no
failure mode beyond a Cobra usage error (e.g. an unexpected positional
argument), which Cobra itself already handles.

## `--json` contract

- `arc lint --json --skip <rules>`: the existing `kernel.LintResult` schema
  (004's contract), unchanged in shape — skipped-rule violations are simply
  absent from `nodes[].violations` and `violations`, and `passing`/`failing`
  are recomputed accordingly. No new field is added to signal that filtering
  occurred; the absence of the violation *is* the signal (consistent with
  032's "absent, not null" convention for optional detail).
- `arc lint rules --json`: a new schema — a bare JSON array of
  `RuleDefinition`:
  ```json
  [
    {"rule": "frontMatter", "description": "Front matter parses as well-formed YAML and declares a recognized identity (@id/@type or legacy kind)."},
    {"rule": "uniqueBasename", "description": "No two node files anywhere in the graph share the same basename."}
  ]
  ```
  Ordering matches `kernel.RuleDefinitions`' declaration order, deterministic
  across runs (FR-012).

## Confirmation and destructiveness (constitution Principle IX)

Neither `--skip` nor `arc lint rules` changes `arc lint`'s read-only nature
(spec 033 has no FR touching graph mutation). No `--yes`/`--force`
confirmation flag is applicable.
