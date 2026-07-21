# CLI Contract Delta: `arc apply`, `arc init`, `arc lint`

This feature changes no command name, flag, or `--json`/`--plain` output
*schema* (field names/types are unchanged). It changes: (a) what `arc apply`
accepts/rejects, (b) the casing of values `arc init` seeds and `arc apply`
writes, (c) the set of violations `arc lint` can report. Per constitution
Principle XIV, this is a scriptable-behavior change (a previously-accepted
patch document can now be rejected; a previously-clean repository can now
lint dirty) and MUST be called out in release notes as a breaking change,
though no flag/command/JSON-field rename is involved.

## `arc apply` — new rejection path

**Usage**: unchanged — `arc apply <patch.md>`.

**New failure condition**: a patch document is rejected, whole-document, no
partial apply (spec Acceptance Scenario 1.3), when any H1 heading that opens
a type section does not begin with an uppercase letter, or when a node's
explicit `@type` (inside its yaml fence) does not begin with an uppercase
letter — independent of the heading (FR-005/FR-008).

| Exit code | Condition |
|---|---|
| `0` | success, unchanged |
| `1` (unchanged code, new cause) | patch rejected: casing violation (new), or any pre-existing rejection reason |

**stderr (human mode)**, example:
```text
class name "entity" must be CamelCase — start with an uppercase letter
```
This is `internal/core.ErrTypeCasing`'s own message (research.md D3),
surfaced unwrapped by `cmd/arc/graph/apply.go`'s existing "return err from
RunE" convention — no new per-command formatting is added, matching how
`ErrManifestInvalid`/`ErrPatchStructure` are already surfaced today.

**No longer happens**: a patch with H1 `# entity` used to succeed, silently
storing `@type: entity`. After this feature, that same input is rejected
(SC-002) — this is the intended, spec-mandated behavior change, not a
regression.

**Unaffected**: a patch with H1 `# Entity` (and no explicit `@type`, or an
explicit `@type: Entity` matching it) succeeds exactly as before, except the
stored `@type` is now `Entity` verbatim rather than the previous lowercased
`entity` (FR-004).

## `arc init` — seeded schema casing

**Usage**: unchanged.

**Output change**: seeded `_schema/types/*.md` filenames and `@id`/`@type`
values for the four built-in content types change from
`source.md`/`entity.md`/`resource.md`/`timeline.md`
(`@type: source`/`entity`/`resource`/`timeline`) to
`Source.md`/`Entity.md`/`Resource.md`/`Timeline.md`
(`@type: Source`/`Entity`/`Resource`/`Timeline`). `Node.md`/`Property.md`/
`Class.md` are unchanged (already CamelCase). Seeded predicate documents
(`_schema/predicates/*.md`) are entirely unaffected — predicate names keep
their existing camelCase convention (lowercase-first), which this feature
does not touch.

**Unaffected**: graph directory layout (`sources/`, `entities/`,
`resources/`, `timeline/` folder names) — research.md D5.

## `arc lint` — new violation rule

**New `Rule` value**: `"typeCase"` (`kernel.RuleTypeCase`), joining the
existing `Rule` enum (`internal/app/lint/kernel/lint.go`). Appears in
human, `--json`, and `--plain` lint output exactly like any other rule
value — no output schema change, only a new possible value in an
already-open string enum (mirrors how `RulePredicateCase` etc. were each
added without a schema version bump).

**Triggers**:
1. A `_schema/types/<name>.md` document whose `name` does not begin with
   an uppercase letter (FR-006) — `Path` is that document's path.
2. A node file whose own `@type` does not begin with an uppercase letter
   (FR-007) — `Path` is that node file's path.

**Example violation** (human mode):
```text
✗ _schema/types/entity.md  typeCase  type "entity" is not CamelCase
```

**Backward compatibility**: a repository that already contains
lowercase-typed schema/content (created before this feature) newly reports
these as ordinary `typeCase` violations the next time `arc lint` runs — no
automatic fix, no suppression (FR-009). `arc lint`'s own exit code
convention (non-zero when any violation exists) is unchanged.
