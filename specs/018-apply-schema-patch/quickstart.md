# Quickstart: `arc apply schema`

Validates the feature end-to-end against a real graph on disk. See
[contracts/cli-contract.md](contracts/cli-contract.md) for the full
flag/output contract and [data-model.md](data-model.md) for the result
shape.

## Prerequisites

- An initialized graph: `arc init ./demo && cd ./demo`
- A schema-only patch document, e.g. `extension.schema.md`:

  ```markdown
  ---
  kind: patch
  document: acme-extension-schema
  published: 2026-07-15
  title: Acme extension vocabulary
  ---

  # Property

  ## acme:weight

  role:: meta
  merge:: fillIfEmpty

  The item's weight in kilograms.

  # Class

  ## Widget

  merge:: union
  required:: acme:weight

  A physical item tracked by the Acme extension.
  ```

## Scenario 1 — import a schema-only patch (spec User Story 1)

```sh
arc apply schema extension.schema.md
```

**Expected**: exits `0`; stdout reports `+1 predicates, +1 type` (or
equivalent human phrasing) and a commit hash; `_schema/predicates/acme_weight.md`
and `_schema/types/Widget.md` now exist; `git log -1` shows exactly one new
commit.

## Scenario 2 — reject a mixed patch (spec User Story 2)

```sh
cat >> extension.schema.md <<'EOF'

# entity

## Acme Corp

category:: organization
definition:: The company behind the extension.
EOF

arc apply schema extension.schema.md; echo "exit: $?"
```

**Expected**: exits `1`; stderr names `Acme Corp`/`entity` as the disallowed
node; `_schema/predicates/`/`_schema/types/` are unchanged from Scenario 1 —
no partial write of `acme:weight`/`Widget` happens on this second, rejected
run either, and `git status` shows no pending changes.

## Scenario 3 — re-apply an updated patch (spec User Story 3)

```sh
git checkout -- extension.schema.md   # undo the entity section from Scenario 2
sed -i '' 's/kilograms/kilograms (SI)/' extension.schema.md   # or manual edit
arc apply schema extension.schema.md
```

**Expected**: exits `0`; stdout reports `+0 predicates (1 merged), +0 types`;
`_schema/predicates/acme_weight.md`'s description reflects the updated text;
`_schema/types/Widget.md` is untouched (byte-identical, no commit content
for it).

## Scenario 4 — URL input

```sh
arc apply schema https://example.org/schemas/acme-extension.schema.md
```

**Expected**: identical output shape to Scenario 1, sourced over HTTP(S)
instead of the local filesystem; a deliberately unreachable host exits `1`
with a fetch-failure message (see cli-contract.md's error table).

## Scenario 5 — `arcnet:` catalog shorthand

```sh
arc apply schema arcnet:media.schema.md
```

**Expected**: identical output shape to Scenario 4, fetched from
`https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/schema/media.schema.md`
without the caller ever writing that URL out. A bare `arc apply schema arcnet:`
(nothing after the prefix) exits `1` immediately, with no network call made.

## Cleanup

```sh
cd .. && rm -rf demo
```
