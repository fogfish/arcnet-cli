# Contract: Predicate & Type Schema Document Shape

Supersedes the existence-only `_schema/nodes/<kind>.md`/`_schema/predicates/<name>.md` shape spec 005 established. Governs what `arc init`'s `Seed()` writes, what `arc apply`'s `RegisterPredicate`/`RegisterType` write for a newly discovered predicate/type, and what `internal/app/schema/service.Resolve` requires on read.

## `_schema/predicates/<name>.md`

```markdown
---
"@id": isPartOf
"@type": Property
role: edge
merge: union
aligned: "dcterms:isPartOf"
---
# isPartOf

Asserts that the subject is a component or member of the whole named by the
target — composition (part-whole), not generalization.
```

- `"@id"` (mandatory): equal to the file's basename.
- `"@type"` (mandatory): the literal `Property`.
- `role` (mandatory): one of `meta`/`text`/`href`/`edge`/`link`.
- `merge` (mandatory): one of the recognized `core.MergeOp` values.
- `label` (recommended, optional): human-readable title; a `link`-role predicate's default heading (capitalized name) applies when absent.
- `aligned` (recommended, optional): a standard-vocabulary term (e.g. `dcterms:isPartOf`) or `arc:<name>` if graph-native.
- Body (mandatory): one to a few sentences of descriptive prose — decoded into `Texts["description"]`.

**Read contract**: `Resolve` rejects the entire load (spec FR-014) if any registered predicate document is missing `role` or `merge`, if `role` is outside the five recognized values, if `merge` is not a recognized `core.MergeOp` value, or if the body carries no description prose.

## `_schema/types/<name>.md` (replacing `_schema/nodes/<kind>.md`)

```markdown
---
"@id": entity
"@type": Class
merge: union
---
# entity

A node for a subject occurring in sources, typed by Sowa category.

- required:: [[category]]
- required:: [[definition]]
- required:: [[mentionedIn]]
- optional:: [[aliases]]
- optional:: [[tags]]
```

- `"@id"` (mandatory): equal to the file's basename.
- `"@type"` (mandatory): the literal `Class`.
- `merge` (mandatory, arcnet-cli-specific bridge field beyond CORE's own documented `Class` shape — spec FR-015): one of the recognized `core.MergeOp` values, read by `arc apply`'s existing whole-node merge dispatch exactly as `_schema/nodes/<kind>.md`'s `merge` field is read today.
- Body (mandatory): descriptive prose — decoded into `Texts["description"]` — followed by zero or more `required::`-prefixed `[[predicate]]` bullets, then zero or more `optional::`-prefixed `[[predicate]]` bullets.
- **Known presentational gap** (research.md D4): the `required`/`optional` bullets above render as one flat list, not grouped under `## Requires`/`## Optional` headings as CORE §9.2's own worked example shows — a future feature (deferred by spec 010) owns heading-grouped rendering. The bullets round-trip correctly regardless.

**Read contract**: `Resolve` rejects the entire load (spec FR-014) if any registered type document is missing `merge`, if `merge` is not a recognized value, or if the body carries no description prose. `Required`/`Optional` MAY both be empty.

## Auto-registered documents (`arc apply` discovery)

A predicate discovered mid-`arc apply` (no existing `_schema/predicates/<name>.md`) is registered as:

```markdown
---
"@id": <name>
"@type": Property
role: edge
merge: union
---
# <name>

Auto-registered by arc apply; describe this predicate's meaning here.
```

A type discovered mid-`arc apply` (no existing `_schema/types/<name>.md`) is registered as:

```markdown
---
"@id": <name>
"@type": Class
merge: union
---
# <name>

Auto-registered by arc apply; describe this type's meaning here.
```

Both satisfy the read contract above in full (no missing mandatory field), so a graph never accumulates a document `Resolve` would reject — the fail-fast contract and the auto-registration contract are drawn from the same set of mandatory fields.
