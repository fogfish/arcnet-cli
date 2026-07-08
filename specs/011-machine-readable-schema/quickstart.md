# Quickstart: Machine-Readable Predicate & Type Schema

Three runnable scenarios, one per spec.md user story. Run from a scratch directory; requires the `arc` binary built from this branch (`go build ./cmd/arc`, or `go run ./cmd/arc`).

## Scenario 1 — A freshly initialized graph fully describes its own vocabulary (US1)

```sh
mkdir demo-graph && cd demo-graph
arc init

cat _schema/predicates/isPartOf.md
# ---
# "@id": isPartOf
# "@type": Property
# role: edge
# merge: union
# aligned: "dcterms:isPartOf"
# ---
# # isPartOf
#
# Asserts that the subject is a component or member of the whole named by
# the target ...

cat _schema/types/entity.md
# ---
# "@id": entity
# "@type": Class
# merge: union
# ---
# # entity
#
# A node for a subject occurring in sources, typed by Sowa category.
#
# - required:: [[category]]
# - required:: [[definition]]
# - required:: [[mentionedIn]]
# - optional:: [[aliases]]
# - optional:: [[tags]]

ls _schema/nodes 2>&1
# ls: _schema/nodes: No such file or directory
```

**Expected outcome**: every predicate document declares `role`/`merge` (plus `label`/`aligned` where CORE provides one); every type document declares `## Requires`/`## Optional`-equivalent bullets (spec.md Acceptance Scenarios 1-2); no `_schema/nodes/` folder exists (Acceptance Scenario 3).

## Scenario 2 — Applying content auto-registers new vocabulary in full (US2)

```sh
cat > note.patch.md <<'EOF'
---
kind: patch
document: acme-2026-widget
published: 2026-07-07
---
# Entity

## Acme Widget

```yaml
"@id": Acme Widget
"@type": entity
category: [independent, physical, continuant, object]
definition: A load-bearing example widget.
supersedes:: [[Legacy Widget]]
```
EOF

arc apply note.patch.md

cat _schema/predicates/supersedes.md
# ---
# "@id": supersedes
# "@type": Property
# role: edge
# merge: union
# ---
# # supersedes
#
# Auto-registered by arc apply; describe this predicate's meaning here.
```

**Expected outcome**: `supersedes` (previously unseen) is registered as a full `Property` node with `role`/`merge` populated — not a bare existence stub — and the write lands in the same commit `arc apply` produces (Acceptance Scenarios 1, 4; `git show --stat HEAD` lists `_schema/predicates/supersedes.md` alongside `entities/Acme Widget.md`).

Now corrupt the schema and confirm the fail-fast contract. Deleting a
single, still-well-formed document merely makes that one type/predicate
unrecognized again — the same "auto-register with a safe default and warn"
path an entirely new type/predicate takes (Acceptance Scenario 2 above), not
a failure. FR-014's fail-fast contract is about a document that exists but
is broken, or a `_schema/` subfolder that is entirely unreadable:

```sh
sed -i '' 's/merge: union/merge: bogus/' _schema/types/entity.md
arc apply note.patch.md
echo $?
# non-zero; error names the invalid _schema/types/entity.md document and its "merge" field
```

**Expected outcome**: the command fails before writing anything else (spec Acceptance Scenario 5, FR-014).

## Scenario 3 — The schema is a reusable index, not tool-internal knowledge (US3)

```sh
arc lint
# both arc apply (Scenario 2) and arc lint recognize the identical
# predicate/type set, since both load the same _schema/ documents through
# the same core.Index — restore _schema/types/entity.md first if Scenario 2's
# corruption step was run.
```

Edit a predicate's declared role and confirm the change is what the next load reports:

```sh
sed -i '' 's/role: edge/role: link/' _schema/predicates/isPartOf.md
arc lint --verbose
```

**Expected outcome**: no code change is needed to alter a predicate's recognized role — editing its `_schema/predicates/` document is sufficient, and every command loads the same, edited value (Acceptance Scenario 4).
