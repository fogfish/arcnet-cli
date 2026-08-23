# Feature Specification: Patch Manifest Identity (`@type: patch`)

**Feature Branch**: `021-patch-type-manifest`

**Created**: 2026-08-23

**Status**: Draft

**Input**: User description: "The document patch exchange format still identifies itself with a legacy `kind: patch` front-matter key. ARCNET-CORE §14.2.1 mandates `@type: patch` as the manifest's only mandatory type declaration, and has done since CORE v0.5 renamed `kind`/`id`/`title` to the JSON-LD `@id`/`@type` pair. The node parser was migrated at that time and now actively rejects a legacy `kind` field on a node, but the patch manifest parser and renderer were not, so the two halves of the tool implement different revisions of the same spec. Today `arc apply` rejects every spec-conformant patch with a manifest-invalid error, and every patch `arc subgraph` emits is unreadable by any other conformant tool. Going forward the patch manifest MUST be recognized and emitted as `@type: patch`. The compatibility with a legacy `kind: patch` manifest IS NOT REQUIRED. A manifest carrying both keys with conflicting values MUST be rejected. `arc lint` is unaffected — patches are never nodes and are never indexed into the graph."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Apply a spec-conformant patch (Priority: P1)

An author receives a document patch produced by another ARCNET-conformant tool — or writes one by hand from the worked example in ARCNET-CORE §14.2.2 — and applies it to their graph with `arc apply`. The patch's manifest declares `"@type": patch`. The tool recognizes it, ingests the nodes it carries, and records the ingest commit.

Today this fails for every such patch: `arc apply` looks for the retired `kind` key, does not find it, and reports the manifest as invalid. This story is the point of the feature — until it works, `arc` cannot consume a single patch written to the current specification.

**Why this priority**: This is the import half of the interoperability break. With only this story shipped, `arc` becomes able to ingest documents from the wider ecosystem, which it cannot do at all today.

**Independent Test**: Transcribe a patch by hand from ARCNET-CORE §14.2.2, run `arc apply` against a fresh graph, and confirm the nodes land and the ingest commit is recorded — without changing the emitting side of the tool.

**Acceptance Scenarios**:

1. **Given** a patch file whose manifest declares `"@type": patch`, **When** the user runs `arc apply`, **Then** the patch is applied, its nodes are written, and exactly one ingest commit is produced.
2. **Given** the same patch already applied, **When** the user runs `arc apply` on it again, **Then** the existing idempotency behaviour holds and no second commit is produced.
3. **Given** a directory of `"@type": patch` files, **When** the user runs `arc apply batch`, **Then** every file is recognized as a patch and applied in the established date-then-path order.
4. **Given** a schema profile patch declaring `"@type": patch`, supplied as a local file, a URL, or an `arcnet:<name>` profile, **When** the user runs `arc apply schema`, **Then** it is recognized and applied.

---

### User Story 2 - Emit patches other tools can read (Priority: P2)

An author runs `arc subgraph` to export a slice of their graph for a collaborator or for publication. The emitted manifest declares `"@type": patch`, so any conformant tool — including `arc` itself — can read it back.

Today every emitted patch is stamped `kind: patch` and is unreadable by any other conformant tool; once Story 1 ships it would be unreadable by `arc` too. The two stories are each independently valuable, but both must land before the tool is self-consistent.

**Why this priority**: Emission is worthless without recognition, so it follows P1 — but it closes the export half of the break and makes the round trip work.

**Independent Test**: Run `arc subgraph`, inspect the emitted manifest, and confirm it carries `"@type": patch` and no `kind` key.

**Acceptance Scenarios**:

1. **Given** a populated graph, **When** the user runs `arc subgraph`, **Then** the emitted manifest's first key is the quoted `"@type"` with value `patch`, and no `kind` key appears anywhere in the manifest.
2. **Given** a patch emitted by `arc subgraph`, **When** that file is applied to a fresh graph with `arc apply`, **Then** it is accepted and applied (round-trip closure).
3. **Given** a graph, **When** the user runs `arc subgraph --json`, **Then** the output is byte-identical to what it produced before this feature — the manifest key is a Markdown-serialization detail, not part of the `--json` contract.

---

### User Story 3 - Unambiguous rejection of non-conformant manifests (Priority: P3)

An author holds a patch written against the retired revision of the format, or a hand-edited patch carrying both `@type` and `kind` with disagreeing values. `arc` refuses it and says exactly what is wrong and what to write instead — rather than failing with a generic manifest-invalid message, misreporting the file as a malformed graph node, or silently passing it over during a batch run.

**Why this priority**: Legacy compatibility is explicitly not required, so correctness here is about the *quality of the refusal*. It is the migration experience for anyone holding older files, and it is what stops a batch run from quietly ingesting less than the user intended.

**Independent Test**: Feed `arc apply` a `kind: patch` file and a self-contradictory file; confirm each is refused with a message naming the file, the offending key, and the required replacement, and that the graph is unmodified.

**Acceptance Scenarios**:

1. **Given** a patch whose manifest declares `kind: patch` and carries no `@type`, **When** the user runs `arc apply`, **Then** the command exits non-zero, the error names the file, the retired `kind` key, and `"@type": patch` as the replacement, and the graph is unmodified.
2. **Given** a patch whose manifest declares `"@type": patch` and `kind: source`, **When** the user runs `arc apply`, **Then** the command exits non-zero, the error identifies the conflict and both values, and the graph is unmodified.
3. **Given** a patch whose manifest declares `"@type": patch` and `kind: patch`, **When** the user runs `arc apply`, **Then** the patch is applied normally.
4. **Given** a directory containing a `kind: patch` file alongside conformant ones, **When** the user runs `arc apply batch`, **Then** the retired file is reported by name as a failed candidate — never counted among the files passed over as ordinary Markdown.
5. **Given** a patch whose manifest declares neither `@type` nor `kind`, **When** the user runs `arc apply`, **Then** the pre-existing manifest-invalid error is reported unchanged.

---

### Edge Cases

- **Unquoted `@type: patch`**: YAML reserves a leading `@` as an indicator character, so the unquoted form is a hard front-matter parse failure that occurs before any patch-identity logic runs. It MUST surface as an error naming the quoting requirement — mirroring the guidance `arc lint` already gives for node identity keys — not as a generic manifest-invalid error.
- **`"@type": Patch`** (capitalized): rejected. ARCNET-CORE §14.2.1 fixes the literal lowercase value for the manifest, deliberately unlike node types, which are CamelCase (spec 019).
- **`kind` with a non-`patch` value and no `@type`**: rejected as a manifest declaring no patch identity at all, not as a legacy patch.
- **Node-level `@type` inside the patch body**: the manifest's `@type` and the per-node `@type` values in the body's YAML fences occupy different scopes and MUST NOT interfere.
- **A patch file left sitting inside the graph tree**: `arc apply` and `arc revert` must continue to tell such a file apart from a graph node when scanning the tree, using the new recognition rule. A `kind: patch` file left in the tree is no longer recognized as a patch and will be reported by the tree scan; the message must remain traceable to the file.
- **A patch that declares `"@type": patch` correctly but fails to parse for an unrelated reason** (e.g. a spec 019 CamelCase violation in its body): the real parse failure must remain the reported reason, not the identity check.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `arc` MUST recognize a front-matter manifest as a document patch when, and only when, it carries the key `@type` with the value `patch`.
- **FR-002**: Every patch `arc` writes MUST carry `"@type"` as the manifest's first key, in quoted form, with the value `patch`, and MUST NOT carry a `kind` key.
- **FR-003**: A manifest carrying `kind: patch` and no `@type` MUST be rejected. The rejection MUST name the offending file, the retired `kind` key, and `"@type": patch` as the replacement.
- **FR-004**: A manifest carrying both `@type` and `kind` MUST be accepted when both name `patch`; `kind` MUST be ignored and MUST NOT be carried into any patch `arc` subsequently writes.
- **FR-005**: A manifest carrying both `@type` and `kind` with disagreeing values MUST be rejected with an error distinct from FR-003's, identifying the conflict and naming both values.
- **FR-006**: A manifest carrying neither `@type` nor `kind` MUST be rejected with the pre-existing manifest-invalid error, unchanged in wording and behaviour.
- **FR-007**: Every rejection under FR-003, FR-005, and FR-006 MUST leave the graph and its commit history completely unmodified.
- **FR-008**: `arc apply batch` MUST classify a file carrying `kind: patch` as a named failed candidate, never as a file passed over for not being a patch. No file a user intended as a patch may be silently skipped.
- **FR-009**: Every command that reads a patch — `arc apply`, `arc apply batch`, `arc apply schema` (local file, URL, and `arcnet:<name>` profile), and `arc revert`'s patch-versus-node discrimination — MUST apply the identical recognition rule, so no two commands can disagree about whether a given file is a patch.
- **FR-010**: Parsing a conformant patch and re-emitting it MUST preserve the `document`, `published`, `title`, and `stats` manifest fields and the entire node body byte-for-byte, changing only the identity key.
- **FR-011**: `arc subgraph --json` output MUST be unaffected by this change.
- **FR-012**: Existing idempotency, ingest-commit, and merge behaviour MUST be unchanged; this feature changes only how a patch declares its identity.
- **FR-013**: `arc lint` behaviour MUST be unchanged. Patches are not nodes, are never indexed into the graph, and are outside the linter's remit.
- **FR-014**: Every patch example shipped in the repository — test fixtures, `specs/**/quickstart.md`, `specs/**/contracts/`, and project documentation — MUST declare `"@type": patch`; none may present `kind: patch` as a current, valid example.
- **FR-015**: An unquoted `@type` key in a patch manifest MUST produce an error naming the quoting requirement rather than a generic manifest-invalid error.
- **FR-016**: An `@type` value equal to `patch` in any casing other than all-lowercase MUST be rejected.

### Key Entities

- **Patch Manifest**: The YAML front-matter block at the head of a document patch (ARCNET-CORE §14.2.1). Declares the patch's type, the `document` it derives from, its `published` date, and an optional `title` and `stats`. This feature changes only the type declaration.
- **`@type`**: The manifest's single mandatory type declaration, written as a quoted key with the literal lowercase value `patch`. The sole means by which a file is recognized as a document patch.
- **`kind`**: The pre-0.5 identity key, already retired for nodes. No longer a valid way to declare a patch; recognized only well enough to refuse the file with an accurate, actionable message.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A patch transcribed by hand from ARCNET-CORE §14.2.2's worked example applies cleanly to a fresh graph. Today the success rate for such patches is 0%.
- **SC-002**: A patch emitted by `arc subgraph` and applied back to a fresh graph reproduces the exported nodes exactly — the export/import round trip closes with zero manual edits.
- **SC-003**: 100% of patch examples in the repository declare `"@type": patch`; zero present `kind: patch` as a valid current example.
- **SC-004**: All five manifest shapes — `@type` only, `kind` only, both agreeing, both disagreeing, neither — produce the outcome specified in FR-001 through FR-006, verified by test.
- **SC-005**: In a batch run over a directory mixing conformant and `kind`-keyed patches, 100% of the `kind`-keyed files are reported by name; zero are silently passed over.
- **SC-006**: Applying the same patch twice still produces exactly one commit.
- **SC-007**: A user can correct a rejected patch from the error message alone, without consulting ARCNET-CORE, because every rejection names the offending file and the required key in one line.

## Assumptions

- **No transitional acceptance.** The user explicitly stated that compatibility with a legacy `kind: patch` manifest is not required, so this feature ships no deprecation-warning grace period: such patches are refused outright from the first release. `kind` is still *detected*, but only to produce an accurate refusal (FR-003) — detection for error quality is not acceptance. This supersedes the transitional variant sketched in `CORE-FIX.md` §3.3 FR-003.
- **Both keys agreeing is accepted.** The stated constraint names *conflicting* values as the rejection case, which implies a non-conflicting pair is not a conflict. A manifest carrying both keys with both naming `patch` therefore parses, with `kind` ignored as an unrecognized extra manifest field — consistent with how other unrecognized manifest fields are already treated.
- **Loud batch failure over silent skip.** A `kind: patch` file in a batch directory could in principle be classified as ordinary non-patch Markdown and passed over. That would silently ingest less than the user intended, so FR-008 chooses a named failure instead. This is a deliberate default chosen here, not an ARCNET-CORE requirement.
- **Literal lowercase `patch`.** This feature implements ARCNET-CORE §14.2.1 as written. Whether the value should instead be CamelCase `Patch` to match node types is a question about the specification, not this tool, and is out of scope.
- **The `document`, `published`, `title`, and `stats` manifest fields** are unchanged by ARCNET-CORE v0.10 and are untouched here.
- **Node front-matter is already migrated** to `@id`/`@type` and is not revisited; the node parser's existing rejection of a legacy `kind` field on a node stands unchanged.
- **Repository fixtures are a deliverable, not incidental.** `kind: patch` appears across test fixtures and specification quickstarts; migrating them is in scope (FR-014) and is the bulk of this change's surface area.
- **The other ARCNET-CORE v0.10 conformance gaps** — the `Reference` type and `Resource` inversion, schema vocabulary conformance, and lint conformance gaps — are tracked as separate features (`CORE-FIX.md` §2) and are out of scope here.
