# CLI Contract: `arc grep` / `arc subgraph` filter flags (delta)

This is an amendment to `specs/006-arc-grep-content-search/contracts/cli-contract.md` and `specs/007-arc-subgraph/contracts/cli-contract.md` — it does not restate their full contract, only what this feature changes or adds.

## Unchanged (spec FR-010 through FR-014, SC-002)

`--type <type>` (repeatable, OR), `--tag <tag>` (repeatable, AND), `--attr <name>=<value>` (repeatable, AND, case-insensitive equality), `--attr <name>~=<pattern>` (repeatable, AND, regexp) — flag names, `Cobra` registration (`StringArrayVar`), help text, repetition/AND/OR semantics, and every observable success/error message are **byte-identical** to before this feature on both `arc grep` and `arc subgraph`. Internally, `opts.build()` now returns a `core.Filter{Statements: [...]}` instead of the old four-field struct (research.md D4), but this is invisible to a CLI caller.

## New: `--predicate <name>` on `arc subgraph` only (research.md D10)

```text
arc subgraph <basename> [--depth <n>] [--predicate <name>]... [--type <type>]... [--tag <tag>]... [--attr <name>=<value>|<name>~=<pattern>]... [--stubs] [--quiet | -q] [--verbose | -v] [--json]
```

- `--predicate <name>` is a new, repeatable `StringArrayVar` flag, OR semantics across repeats (same convention as `--type`) — see `research.md` D4/D10 for its exact lowering.
- **Not registered on `arc grep`** (`opts.apply` gains a second variant, or a boolean toggle, so `arc grep`'s command construction does not add a flag with no traversal to scope) and **not registered on `arc lint`** (which registers no filter flags today and gains none here).
- Effect: restricts which structural connections `arc subgraph`'s BFS follows, in both the outgoing and incoming (backlink) direction, to only those whose `Link.Predicate` is one of the given names. Omitting `--predicate` entirely preserves today's default: every connection is followed, unchanged.
- Composes with `--type`/`--tag`/`--attr`: `--predicate` controls *what gets reached*; `--type`/`--tag`/`--attr` continue to control *which of the reached (non-seed) nodes survive into the result* (spec FR-006/FR-009). Both may be given together in one invocation.
- The seed node is never excluded by `--predicate` — it is always present in the output (spec FR-009, unchanged from today's `--type`/`--tag`/`--attr` seed-exemption).

### Example

```text
arc subgraph TLS --predicate cites
arc subgraph TLS --predicate cites --predicate mentions --depth 2
arc subgraph TLS --predicate cites --type Source
```

### Help text delta

- `Long` gains one sentence: "`--predicate <name>` restricts which structural connections are followed during expansion to the named relation(s), independent of `--type`/`--tag`/`--attr`, which narrow the result instead."
- `Example` gains: `\tarc subgraph TLS --predicate cites`

### Error messages

`--predicate` accepts any string value (open vocabulary, matching how `--type`/`--tag`'s values are never validated against a fixed enum) — no new error condition is introduced. A `--predicate` value matching no edge on any reached node is not an error: the expansion simply reaches nothing further from that node (spec Edge Cases), exactly as an unmatched `--type`/`--tag`/`--attr` filter produces an empty-but-successful result today.

## Exit codes, confirmation, destructiveness

Unchanged from `specs/006`/`specs/007`'s existing contracts — `--predicate` introduces no new refusal condition, no new confirmation prompt, and no change to `arc grep`'s or `arc subgraph`'s strictly-read-only guarantee.
