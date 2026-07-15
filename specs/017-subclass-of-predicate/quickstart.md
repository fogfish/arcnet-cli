# Quickstart: Validating `rdfs:subClassOf` Type Inheritance

Prerequisites: a built `arc` binary from this branch; an empty working directory.

## 1. A freshly initialized graph seeds `Node` and wires the four content types to it

```sh
mkdir demo-graph && cd demo-graph
arc init
cat _schema/types/Node.md
cat _schema/types/source.md
```

**Expected**: `_schema/types/Node.md` exists, declaring `## Requires` containing `published`/`created` and `## Optional` containing `tags`/`text`/`updated`/`scoreZ`/`scoreC`. `_schema/types/source.md` carries a `subClassOf:: [[Node]]` edge (predicate name `subClassOf`, `Aligned: "rdfs:subClassOf"` — internal/core's bullet parser only accepts `\w+` predicate names, so the colon-bearing RDFS term lives in `Aligned`, not the bullet key itself) and no longer lists `tags`/`created`/`updated`/`scoreZ`/`scoreC` directly under its own `## Optional` (data-model.md's reshaped-types table).

## 2. Lint enforces the inherited contract (User Story 1)

Apply or hand-author a `source` node missing `published` (now required only via `Node`, not directly on `source`):

```sh
arc apply <a patch introducing a source node with title/abstract/mentions but no published>
arc lint
```

**Expected**: lint reports a missing-required-predicate violation naming `published` against the `source` node — proving the inherited requirement is enforced exactly as a directly declared one would be (spec Acceptance Scenario US1.3, SC-002).

## 3. Multiple base types compose (User Story 2)

```sh
mkdir -p _schema/types
cat > _schema/types/citable.md <<'EOF'
---
merge: union
---
citable work base.

## Requires

- required:: [[doi]]
EOF

cat > _schema/types/timestamped.md <<'EOF'
---
merge: union
---
timestamped record base.

## Requires

- required:: [[updated]]
EOF

cat > _schema/types/dataset.md <<'EOF'
---
merge: union
---
A dataset, both citable and timestamped.

- subClassOf:: [[citable]]
- subClassOf:: [[timestamped]]
EOF

arc lint
```

**Expected**: a `dataset` node is now required to carry both `doi` (from `citable`) and `updated` (from `timestamped`, overriding `Node`'s own `updated: optional` — required wins, spec FR-007) — confirmed by authoring a `dataset` node missing one of the two and observing the corresponding lint violation.

## 4. Multi-level chains resolve transitively (User Story 3)

Extend `timestamped.md` with `- subClassOf:: [[Node]]` (redundant with the implicit rule, but exercises an explicit 2-level chain: `dataset → timestamped → Node`) and confirm `dataset` still ends up requiring `Node`'s `published`/`created` — it does, via the *implicit* rule alone (`dataset` is not `Node`/`Property`/`Class`), independent of any explicit chain.

## 5. Cycles and unresolved bases fail schema loading (User Story 4)

```sh
cat >> _schema/types/citable.md <<'EOF'

- subClassOf:: [[dataset]]
EOF

arc lint
```

**Expected**: `arc lint` (and every other schema-dependent command) fails immediately with a clear error naming the cycle (`citable → dataset → citable`), not a hang or a silent pass — restore `citable.md` afterward and confirm a clean `arc lint` run again.

```sh
sed -i '' 's/\[\[Node\]\]/[[NoSuchType]]/' _schema/types/timestamped.md
arc lint
```

**Expected**: a clear error naming `timestamped` and the unresolved `NoSuchType` reference.

## Cleanup

```sh
cd .. && rm -rf demo-graph
```
