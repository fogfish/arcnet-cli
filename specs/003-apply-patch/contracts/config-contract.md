# Config Contract: `internal/app/config`

## Primary port (`component.go`)

```go
func Resolve(store fsys.Store) (core.MergeRuleSet, error)
func Save(store fsys.Store, cfg kernel.Config) error
```

- `Resolve` — reads `.arc/config.yml` (`core.ConfigPath`) via `store.Open`. File absent → returns `core.CoreMergeRules` alone (no error; this is "no domain kinds registered", spec User Story 3 Acceptance Scenario 2). File present but not valid YAML → `ErrConfigMalformed`. File present and valid → `core.CoreMergeRules.Union(loaded)` (the format's three built-in kinds always win if a file somehow tries to redeclare one differently — `Union` is first-writer over the two rule sets being combined, with `CoreMergeRules` as the first/authoritative side).
- `Save` — writes `cfg` back as YAML via `store.Create`. Used by `internal/app/ctrl/kernel.DefaultLayout`'s seed-content computation indirectly (research.md D5: `ctrl` calls `core.CoreMergeRules.YAML()` directly, **not** `config.Save` — `ctrl` never imports `internal/app/config`). `Save`'s primary caller in this feature's scope is `internal/app/config`'s own unit tests round-tripping `Resolve`; no CLI command calls `Save` directly in this iteration (research.md D5 — no `arc config` mutation command shipped yet).

## On-disk shape (`.arc/config.yml`)

```yaml
mergeRules:
  source: none
  entity: union
  resource: union-first-writer
```

A user opts a domain kind in by hand-adding a line, e.g. `hypothesis: validated-overwrite` (copied from `core.KnownProfileMergeRules`, or an arbitrary value if defining an entirely new, project-local kind not documented by any known profile — `Resolve` does not restrict `mergeRules` keys/values to `KnownProfileMergeRules`, only to the fixed `MergeOp` vocabulary itself).

## Seeding at `arc init` (touches `internal/app/ctrl`, research.md D5)

`internal/app/ctrl/kernel.DefaultLayout.MetaStubs[core.ConfigPath]` = `core.CoreMergeRules.YAML()`'s bytes. `arc init`'s existing `writeLayout` loop (`specs/002-arc-init`) writes this file with no new code path — it is one more `MetaStubs` entry, exactly like `_meta/predicates.md`/`_meta/aliases.md` already are.

## Known limitation (research.md D5)

`.arc/` is `.gitignore`d — `.arc/config.yml` is local to one clone, not synced via git. Two collaborators on the same graph repository can have different locally-registered domain kinds. Flagged, not silently resolved; see research.md D5 for the full note.
