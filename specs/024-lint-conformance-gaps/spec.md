# Feature Specification: Lint Conformance Gaps

**Feature Branch**: `024-lint-conformance-gaps`

**Created**: 2026-08-23

**Status**: Draft

**Input**: User description: "Two checks in ARCNET-CORE's conformance model are either under-enforced or absent from `arc lint`. First, §10.2 enumerates exactly twelve legal four-word Sowa category combinations that an `Entity` may carry, but the linter validates each of the four positions against its own word set independently, which admits one hundred and forty-four combinations. Eleven out of twelve accepted categories are therefore meaningless — for example `[independent, physical, continuant, purpose]` passes today and denotes nothing in Sowa's taxonomy. The check must validate the whole four-word tuple against the twelve-row table, and its error message must name the closest legal combination so the fix is obvious. Second, §7.1 states that a node's identity must not contain any of the characters `/ \ : * ? \" < > | .` — a rule that exists because identities are also filenames and links resolve by basename — and no rule enforces it. A node whose `@id` contains a path separator or a colon is unfilable, unlinkable, or silently truncated depending on the host filesystem, and nothing currently catches it. A new rule must flag any identity containing a forbidden character, naming the character and its position. Both rules are detection only: no file is rewritten and no format changes. However, the second rule identity has to be enforced by `arc apply` as well to prevent corruption of the graph."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Block unsafe identities before they enter the graph (Priority: P1)

A user runs `arc apply` on a patch that introduces or edits a node whose identity contains a character that cannot safely live in a filename (a slash, colon, or similar). Today the tool accepts it silently, producing a node that is unfilable, unlinkable, or silently mangled depending on the operating system it was written on. Instead, `arc apply` must refuse the operation, tell the user exactly which character is the problem and where in the identity it occurs, and leave the graph exactly as it was before the command ran.

**Why this priority**: This is the only one of the three checks that prevents irreversible damage — an unsafe identity written to disk can corrupt the graph (unfilable nodes, broken links, platform-dependent truncation) in a way `arc lint` can only report on after the fact. Stopping it at the point of ingestion is the highest-value fix in this feature.

**Independent Test**: Can be fully tested by running `arc apply` on a patch containing one node with a forbidden character in its identity and confirming the command exits with an error naming the character and position, and that no file is written or modified.

**Acceptance Scenarios**:

1. **Given** a patch that introduces a node whose identity is `Handshake/Protocol`, **When** the user runs `arc apply`, **Then** the command exits with an error naming `/` and its position in the identity, and the graph directory is left byte-for-byte unchanged.
2. **Given** a patch that introduces a node whose identity is `Handshake Protocol (TLS)` (no forbidden characters), **When** the user runs `arc apply`, **Then** the node is written normally and no identity error is reported.
3. **Given** a patch containing several nodes, only one of which has an unsafe identity, **When** the user runs `arc apply`, **Then** the entire operation is rejected — none of the patch's nodes are written, including the ones that were otherwise valid.
4. **Given** a patch that introduces a schema node (a predicate or type definition) whose identity contains a forbidden character, **When** the user runs `arc apply`, **Then** the same rejection applies as for a content node.

---

### User Story 2 - Surface existing unsafe identities already in the graph (Priority: P2)

A user runs `arc lint` on a graph that already contains nodes written before this feature existed, or written by hand outside of `arc apply`. Some of those nodes may carry identities with forbidden characters that were never caught. The user needs `arc lint` to find and report every one of them, precisely enough that fixing each is a matter of renaming the identity, not guessing.

**Why this priority**: Second because it is detection-only cleanup of an existing, already-shipped gap, valuable for graph hygiene but not preventing new damage the way User Story 1 does. It is independent of User Story 1: the two can be delivered and tested separately, and this one primarily protects graphs the tool did not create.

**Independent Test**: Can be fully tested by running `arc lint` against a graph containing a node with a known forbidden character in its identity and confirming the report names the character and its 1-indexed position, without any file in the graph being changed.

**Acceptance Scenarios**:

1. **Given** a node whose identity is `Handshake/Protocol`, **When** the user runs `arc lint`, **Then** a violation is reported naming `/` and its position (10) within the identity.
2. **Given** a node whose identity is `Handshake Protocol (TLS)`, **When** the user runs `arc lint`, **Then** no identity-charset violation is reported for that node.
3. **Given** a `Source` node whose identity is the citekey `rescorla-2026-tls13`, **When** the user runs `arc lint`, **Then** no identity-charset violation is reported.
4. **Given** a schema node (predicate or type definition) whose identity contains a `.`, **When** the user runs `arc lint`, **Then** the same violation is reported as for a content node.
5. **Given** a graph containing an identity with more than one forbidden character, **When** the user runs `arc lint`, **Then** the report names every offending character and its own position, not just the first.
6. **Given** any graph, **When** the user runs `arc lint` before and after the run, **Then** no file in the graph has changed.

---

### User Story 3 - Reject Sowa categories that mix incompatible words (Priority: P3)

A user runs `arc lint` on a graph containing an `Entity` node whose four-word category combines words from different rows of Sowa's taxonomy — a combination that is structurally well-formed (one word from each of the four expected word groups) but denotes nothing meaningful, because Sowa's taxonomy only recognizes twelve specific four-word combinations, not every cross-product of the four groups. The user needs `arc lint` to catch this and to suggest the closest legal combination so the fix is a small edit rather than a lookup in an external document.

**Why this priority**: Third because it is the most self-contained of the three checks — it touches one field on one node type — and, unlike User Story 1, a wrong-but-structurally-valid category is a data-quality problem, not a filesystem-corruption risk. It does not depend on either of the other two stories.

**Independent Test**: Can be fully tested by running `arc lint` against an `Entity` node whose category mixes words from two different legal combinations and confirming the report names the rejected combination and offers a legal one, without any file being changed.

**Acceptance Scenarios**:

1. **Given** an `Entity` whose category is `[independent, abstract, occurrent, script]` (a legal combination), **When** the user runs `arc lint`, **Then** no category violation is reported.
2. **Given** an `Entity` whose category is `[independent, physical, continuant, purpose]` (each word individually valid, but not a legal combination), **When** the user runs `arc lint`, **Then** a violation is reported naming the rejected combination and suggesting `[independent, physical, continuant, object]` as the closest legal combination.
3. **Given** an `Entity` whose category has the wrong number of words, **When** the user runs `arc lint`, **Then** the existing wrong-length message is reported, unchanged by this feature.
4. **Given** a graph, **When** the user runs `arc lint` before and after the run, **Then** no file in the graph has changed.

---

### Edge Cases

- An identity is `rescorla-2026-tls13.md`-adjacent in spirit but the check applies to the identity value itself, not the on-disk filename — so a citekey identity like `rescorla-2026-tls13` (no dot) is legal, while an identity like `v1.3 Handshake` (a dot inside the value) is not, even though every node file ends in `.md` regardless.
- A `Timeline` node's identity is a period code such as `2026-04` or `2026` — neither contains a forbidden character and both must pass.
- An identity most likely to carry a real violation is a full document title used verbatim (e.g. a `Reference`'s identity), especially one containing a colon from a subtitle — this is exactly the case both the `arc apply` rejection and the `arc lint` report exist to catch.
- A patch that only modifies an already-existing node (rather than introducing a new one) still has its target identity checked by `arc apply`; an unsafe identity already sitting in the graph from before this feature shipped is not retroactively blocked by `arc apply` merely because a later patch happens to touch that node's other attributes — it is surfaced by `arc lint` (User Story 2) instead.
- A category tuple that is one word short or one word long keeps the existing, distinct "wrong length" message rather than being run through the twelve-row comparison.
- An identity containing a Unicode character that visually resembles a forbidden ASCII character (e.g. a fullwidth colon) is out of scope for detection.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `arc apply` MUST reject any patch that introduces or modifies a node — content or schema — whose identity contains any of the forbidden characters `/ \ : * ? " < > | .`.
- **FR-002**: A rejection under FR-001 MUST name every forbidden character present in the identity and each character's 1-indexed position within the identity value.
- **FR-003**: A rejection under FR-001 MUST leave the graph completely unmodified — no node from the patch is written, including any other node in the same patch that would otherwise have been valid.
- **FR-004**: `arc lint` MUST check every node's identity — content and schema — for the same forbidden characters as FR-001, independent of whether the node was written by `arc apply` or by hand.
- **FR-005**: A violation reported under FR-004 MUST name every forbidden character found in the identity and each character's 1-indexed position, matching the level of detail required by FR-002.
- **FR-006**: Neither the FR-004 lint check nor the FR-008 category check MAY modify, rewrite, or reformat any file in the graph; both are detection-only.
- **FR-007**: `arc lint` MUST validate an `Entity` node's four-word category as a single combination against the twelve legal combinations defined by Sowa's taxonomy, rather than validating each of the four words independently against its own word group.
- **FR-008**: A category that fails FR-007 MUST produce a violation that names the rejected combination and identifies at least one legal combination that shares the most leading words with it, so the suggested fix is unambiguous.
- **FR-009**: A category with a number of words other than four MUST continue to produce the existing wrong-length violation message, unchanged by FR-007/FR-008.
- **FR-010**: The set of twelve legal Sowa combinations and the set of ten forbidden identity characters are each fixed, closed lists; neither check may accept a combination or character outside its respective list without a corresponding update to that list.

### Key Entities

- **Entity category**: A four-word attribute on an `Entity` node, each word drawn from a fixed vocabulary; only twelve specific four-word combinations are meaningful under Sowa's taxonomy, though all 144 positional cross-products currently pass validation.
- **Node identity**: The `@id` value carried by every node (content or schema); doubles as the basis for the node's filename and for how other nodes link to it by reference.
- **Patch**: The unit of change `arc apply` ingests; may introduce or modify one or more node identities in a single operation, which either all succeed or all fail together.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: No node with a forbidden character in its identity can ever be written to a graph by `arc apply`, regardless of how many other nodes are in the same patch.
- **SC-002**: A user who runs `arc apply` on a patch with an unsafe identity learns exactly which character and position caused the rejection from the command's own output, without needing to inspect the file system or consult external documentation.
- **SC-003**: Of the 144 positional Entity-category combinations previously accepted by `arc lint`, exactly the 12 combinations that are meaningful under Sowa's taxonomy continue to pass; the other 132 are now rejected.
- **SC-004**: A user whose `Entity` category is rejected receives a suggested legal combination in the same report, closing the loop without a second lookup.
- **SC-005**: Running `arc lint` on any graph never changes a single file in that graph, before or after this feature.

## Assumptions

- The twelve legal Sowa combinations and the ten forbidden identity characters (`/ \ : * ? " < > | .`) are exactly the sets specified in ARCNET-CORE §10.2 and §7.1 respectively; this feature does not revisit or extend either list.
- "Closest legal combination" (FR-008) means the legal combination sharing the longest matching prefix, read left to right, with the rejected combination — the simplest rule that satisfies "obvious fix," not a general similarity search.
- The identity charset check applies to the identity value only, not to the derived on-disk filename — the trailing `.md` file extension every node file carries is not part of the identity being checked, so a `.`-free identity remains legal even though its file ends in `.md`.
- `arc apply`'s FR-001 check covers only the node identities present in the patch being applied at that moment; it does not scan or retroactively re-validate identities already committed to the graph from before this feature shipped. Those are covered by `arc lint` (User Story 2) instead.
- Schema documents (predicate and type definitions) are in scope for both the `arc apply` rejection and the `arc lint` report, since a schema node's identity is equally a filename.
- Both `arc lint` checks remain detection-only, consistent with `arc lint --fix` being out of scope for the tool today; no auto-fix or rewrite behavior is introduced by this feature.
- Unicode characters that visually resemble a forbidden ASCII character are not treated as forbidden by either check.
