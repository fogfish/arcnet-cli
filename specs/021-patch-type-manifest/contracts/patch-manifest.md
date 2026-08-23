# Contract: Patch Manifest Grammar

**Feature**: [../spec.md](../spec.md) | **Plan**: [../plan.md](../plan.md) | **Model**: [../data-model.md](../data-model.md)

Normative for the document-patch exchange format `arc` reads and writes, per ARCNET-CORE
§14.2.1. Supersedes the manifest shape documented in
[`specs/003-apply-patch/contracts/cli-contract.md`](../../003-apply-patch/contracts/cli-contract.md)
and [`specs/007-arc-subgraph/contracts/cli-contract.md`](../../007-arc-subgraph/contracts/cli-contract.md),
both of which are updated to match (research D7).

---

## 1. Grammar

```yaml
---
"@type": patch                      # MANDATORY — quoted key, literal lowercase value
document: shannon-1948-mathematical # MANDATORY
published: 2024-01-15               # MANDATORY — YYYY-MM-DD or RFC3339
title: "A Mathematical Theory..."   # OPTIONAL
stats: {sources: 1, entities: 4}    # OPTIONAL — flow style on emit
---
```

### `@type`

| Property | Value |
|---|---|
| Key form | **Quoted** — `"@type"` or `'@type'`. A bare `@type:` is a YAML error (§3.1). |
| Value | The literal lowercase string `patch`. `Patch`, `PATCH` are rejected. |
| Cardinality | Exactly one. Absence means the document is not a patch. |
| Emitted as | `"@type": patch`, double-quoted key, **first key** of the manifest. |

The lowercase value is deliberate and is *not* an exception this tool may normalize away.
Node `@type` values are CamelCase (spec 019) because they name classes in the graph; the
manifest's `@type` names a document kind, which ARCNET-CORE fixes as lowercase `patch`. A
patch is never a node and is never indexed.

### `kind` — retired

`kind` is not part of this grammar. It is recognized solely to produce an accurate refusal
(§3.3) and, when it accompanies an agreeing `@type`, to be ignored (§2, row 2). `arc` never
emits it.

---

## 2. Recognition

`T` = `@type` value, `K` = `kind` value; "present" = key exists with a non-empty string value.
First match wins.

| # | Condition | Result | Error |
|---|---|---|---|
| 1 | `T` present ∧ `K` present ∧ `T ≠ K` | reject | `ErrManifestTypeConflict(T, K)` |
| 2 | `T = "patch"` | **accept** | — |
| 3 | `T` present ∧ `T ≠ "patch"` | reject | `ErrManifestNotAPatch(T)` |
| 4 | `T` absent ∧ `K = "patch"` | reject | `ErrManifestLegacyKind` |
| 5 | otherwise | reject | `ErrManifestInvalid` |

Total: every (`T`, `K`) pair matches exactly one row.

Recognition (`LooksLikePatch`) is broader than acceptance and is **not** compatibility: it
returns true for rows 1, 2, 3 and 4 — anything that declares itself a patch under either key —
so that a refusal is reported against the file rather than the file being silently skipped.
Rows 1, 3, 4 still reject.

---

## 3. Error contract

Every message below is emitted to **stderr**, exits **1**, and leaves the graph and its commit
history unmodified (FR-007). Each is prefixed with the file path by the calling service
(`failed to read patch file <path>: …`) except where the caller has no path (in-memory or URL
sources, which carry the URL instead).

### 3.1 Unquoted identity key (FR-015)

```
$ arc apply bare.md
failed to read patch file bare.md: "@type" must be a quoted YAML string key, found it unquoted
```

Detected on the `parseDocument` error branch, before recognition. The message matches
`arc lint`'s `RuleIdentityQuoting` text verbatim.

### 3.2 Type conflict (row 1, FR-005)

```
$ arc apply conflicting.md
failed to read patch file conflicting.md: manifest declares "@type": patch and legacy kind: source — the two disagree
```

### 3.3 Retired key (row 4, FR-003)

```
$ arc apply legacy.md
failed to read patch file legacy.md: manifest uses the retired "kind: patch" key — rewrite it as "@type": patch
```

The message names the offending key **and** its replacement, so the file is correctable
without consulting ARCNET-CORE (SC-007).

### 3.4 Wrong value or casing (row 3, FR-016)

```
$ arc apply capitalized.md
failed to read patch file capitalized.md: manifest declares "@type": Patch, expected the literal lowercase "patch"
```

### 3.5 No identity at all (row 5) — unchanged

```
$ arc apply notapatch.md
failed to read patch file notapatch.md: manifest is missing a mandatory field or uses the pre-0.5 node format
```

Wording is unchanged from before this feature (FR-006).

---

## 4. Emission

`arc subgraph` (and every other `RenderPatch` caller — `arc serve`'s `subgraph_get`, the
schema seed writer) emits:

```yaml
---
"@type": patch
document: <citekey>
published: <YYYY-MM-DD>
title: <string>          # omitted when empty
stats: {k: v}            # omitted when empty
---
```

Guarantees:

- `"@type"` is the **first** key, double-quoted, value `patch`.
- No `kind` key appears anywhere in the output.
- Field order, date format, `title`/`stats` omission rules, and flow styling are unchanged
  from before this feature.
- Parse → render of a conformant patch is byte-identical except for the identity key
  (FR-010).

---

## 5. Stability

| Surface | Stable contract? | Affected |
|---|---|---|
| Patch Markdown manifest | yes — this document | **breaking change**, see [plan.md § Complexity Tracking](../plan.md#complexity-tracking) |
| `arc subgraph --json` | yes (spec 007) | **no** — `core.Patch` gains no field |
| `arc apply batch --json` | yes (spec 020) | counts only; no schema change |
| Human-readable stderr text | no (Principle X) | messages are new/changed as above |
