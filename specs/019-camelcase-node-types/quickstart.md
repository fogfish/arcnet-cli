# Quickstart: CamelCase Node Class Names

Validates the feature end-to-end against a real graph on disk. See
[contracts/cli-contract.md](contracts/cli-contract.md) for the full
behavior contract and [data-model.md](data-model.md) for the validity rule.

## Prerequisites

- A built `arc` binary (`go build -o arc ./cmd/arc`)

## Scenario 1 — built-in schema is CamelCase (spec User Story 2)

```sh
./arc init ./demo && cd ./demo
ls _schema/types/
```

**Expected**: `Class.md Entity.md Node.md Property.md Resource.md Source.md Timeline.md`
— every filename begins with an uppercase letter; `grep '"@type"' _schema/types/Entity.md`
shows `"@type": "Class"` and `grep '"@id"' _schema/types/Entity.md` shows
`"@id": "Entity"`.

## Scenario 2 — `arc apply` rejects a lowercase H1 (spec User Story 1, Acceptance Scenario 1)

```sh
cat > bad.patch.md <<'EOF'
---
kind: patch
document: bad-entry
published: 2026-07-19
title: A lowercase-headed contribution
---

# entity

## acme-widget

category:: object
definition:: A thing that should have been rejected.
EOF

./arc apply bad.patch.md; echo "exit: $?"
```

**Expected**: exits `1`; stderr states `class name "entity" must be CamelCase`
(or equivalent); `entities/acme-widget.md` does not exist; `git status`
shows no pending changes; `git log -1 --oneline` is unchanged from before
this command ran.

## Scenario 3 — `arc apply` accepts a CamelCase H1 (spec User Story 1, Acceptance Scenario 2)

```sh
cat > good.patch.md <<'EOF'
---
kind: patch
document: good-entry
published: 2026-07-19
title: A CamelCase contribution
---

# Entity

## acme-widget

category:: object
definition:: A thing that should have been accepted.
EOF

./arc apply good.patch.md; echo "exit: $?"
grep '"@type"' entities/acme-widget.md
```

**Expected**: exits `0`; `entities/acme-widget.md` exists with
`"@type": "Entity"` (not lowercased); exactly one new commit.

## Scenario 4 — `arc lint` flags a hand-authored lowercase type (spec User Story 3)

```sh
mkdir -p _schema/types
cat > _schema/types/gadget.md <<'EOF'
---
"@id": "gadget"
"@type": "Class"
merge: union
---
# gadget

A hand-authored, non-conformant type definition.
EOF

./arc lint; echo "exit: $?"
```

**Expected**: exits non-zero; the report includes a `typeCase` violation for
`_schema/types/gadget.md` stating `type "gadget" is not CamelCase`.

## Cleanup

```sh
cd .. && rm -rf demo
```
