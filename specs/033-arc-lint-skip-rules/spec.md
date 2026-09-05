# Feature Specification: Lint Rule Skipping (`arc lint --skip`, `arc lint rules`)

**Feature Branch**: `033-arc-lint-skip-rules`

**Created**: 2026-09-05

**Status**: Draft

**Input**: User description: "`arc lint` takes `--skip`, it is a list of rule names to skip, it uses comma to separate multiple values. The flag requires a sibling command `arc lint rules` that shows a human readable definition for all rules implemented by the app."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Suppress a rule that doesn't apply to this graph (Priority: P1)

A user runs `arc lint` and finds it reports a rule they've deliberately decided not to follow in this graph — the report is otherwise useful, but this one rule's violations drown out everything else, or block a script that gates on lint's exit status. They re-run lint naming that rule to skip, and every other rule is still checked in full.

**Why this priority**: This is the entire point of the feature. Without it, a user stuck with one unwanted rule has no way to use lint at all — they either accept a permanently noisy or permanently failing report, or stop using the command. One rule's violations disappearing from both the report and the outcome, with everything else unaffected, is the complete value of this story.

**Independent Test**: Can be fully tested by running lint with `--skip` naming one rule against a graph with a known mix of violations spanning several rules, and confirming the named rule's violations are absent from the report and every other rule's violations are unchanged.

**Acceptance Scenarios**:

1. **Given** a graph with known violations of rule A and rule B, **When** the user runs lint with `--skip` naming rule A, **Then** the report shows rule B's violations exactly as before and shows none of rule A's.
2. **Given** a graph whose only violations belong to a rule the user names in `--skip`, **When** the user runs the command, **Then** the run reports zero violations and succeeds, rather than still failing because of the suppressed rule.
3. **Given** a user wants to skip more than one rule, **When** they pass `--skip` with several rule names separated by commas, **Then** violations of every named rule are absent and violations of every other rule are reported as usual.
4. **Given** a graph, **When** the user runs lint naming every rule the tool implements in `--skip`, **Then** the command still enumerates the graph's nodes and succeeds, reporting zero violations.

---

### User Story 2 - Discover the rules by name and meaning (Priority: P2)

Before a user can decide what to put in `--skip`, they need to know what rules exist, what each one actually checks, and the exact name to type. They run a command dedicated to listing every rule the tool implements, in plain language.

**Why this priority**: `--skip` is unusable without a trustworthy way to look up valid rule names and understand what each one means before turning it off. This is the reference a user consults once, then reuses `--skip` many times.

**Independent Test**: Can be fully tested by running the rule-listing command against any working directory (graph or not) and confirming it lists every rule the tool's lint checks implement, each with its exact name and a plain-language description, with no graph required.

**Acceptance Scenarios**:

1. **Given** any working directory, **When** the user runs the rule-listing command, **Then** it lists every rule lint implements, each paired with a human-readable explanation of what it checks — independent of whether the current directory holds a graph at all.
2. **Given** the user has just read the rule list, **When** they use one of the listed names verbatim in `arc lint --skip`, **Then** that exact name is accepted.
3. **Given** the user runs the rule-listing command twice with nothing else changed, **When** they compare the two outputs, **Then** they are identical, so the list is safe to rely on and script against.

---

### User Story 3 - Get an immediate, clear error on a mistyped rule name (Priority: P3)

A user misspells a rule name, or names a rule that no longer exists, in `--skip`. They need to find out immediately and unambiguously, rather than having lint silently run as if the flag had no effect, which would be especially dangerous in a script or CI job that trusts its own configuration.

**Why this priority**: Protects the value delivered by Story 1: a `--skip` that can silently fail to suppress anything (or silently accept garbage) would erode confidence in the feature every time someone reaches for it. This is a safety net around the core capability rather than a capability of its own.

**Independent Test**: Can be fully tested by running lint with `--skip` naming a rule that does not exist and confirming the command refuses with a message identifying exactly which name was invalid, before any part of the graph is checked.

**Acceptance Scenarios**:

1. **Given** a graph, **When** the user runs lint with `--skip` naming a rule that does not exist, **Then** the command refuses, names the invalid rule, and performs no lint checks.
2. **Given** `--skip` names both a valid and an invalid rule together, **When** the user runs the command, **Then** it still refuses on the invalid name rather than silently applying only the valid one.

---

### Edge Cases

- What happens when `--skip` contains the same rule name more than once? It is treated the same as naming it once; no error.
- What happens when `--skip`'s value has extra whitespace around a name, or an empty segment from a trailing/leading/doubled comma (e.g. `"ruleA, ,ruleB"`)? Whitespace around a name is ignored and empty segments are ignored; only genuinely unrecognized names are refused.
- What happens when `--skip` is passed with an empty value, or omitted entirely? Lint behaves exactly as it does today — nothing is skipped.
- What happens when a skipped rule is one that spans more than one file rather than belonging to a single node (e.g. a duplicate-basename check)? It is suppressed the same as a node-owned rule — entirely absent from the report, in every output mode.
- What happens when the rule-listing command is run outside any graph, or in a directory that isn't a graph at all? It succeeds and lists the same rules regardless, since the list describes the tool itself, not any particular graph.
- What happens when `--skip` is combined with the tool's machine-readable output mode? The suppressed rule's violations are absent from the structured output exactly as they are from the human-readable report — never present-but-flagged-skipped.
- What happens when `--skip` is combined with the tool's detailed/verbose output mode? The suppressed rule is equally absent from the per-node detail listing, not just the summary.

## Requirements *(mandatory)*

### Functional Requirements

#### `arc lint --skip`

- **FR-001**: The tool MUST accept a `--skip` option on the lint command, whose value is one or more rule names separated by commas.
- **FR-002**: The tool MUST ignore leading/trailing whitespace around each named rule and MUST ignore empty segments produced by stray commas, so `"ruleA, ,ruleB"` behaves identically to `"ruleA,ruleB"`.
- **FR-003**: The tool MUST treat a rule named more than once in `--skip` the same as naming it once.
- **FR-004**: When every name in `--skip` matches a rule the tool implements, the tool MUST NOT report any violation of a named rule, in any output mode (default, detailed/verbose, machine-readable) and regardless of whether the violation belongs to a single node or spans the graph.
- **FR-005**: A node or graph condition whose only violations belong to skipped rules MUST be counted as passing, and the run's overall outcome (including exit status) MUST reflect only the rules that were not skipped.
- **FR-006**: Naming every rule the tool implements in `--skip` MUST be accepted; the run MUST still enumerate the graph's nodes and succeed, reporting zero violations.
- **FR-007**: When `--skip` names one or more rules the tool does not implement, the tool MUST refuse the run with a clear message identifying every unrecognized name, and MUST perform no lint checks in that run — including when the same `--skip` value also names valid rules.
- **FR-008**: The tool MUST validate `--skip`'s rule names before requiring or resolving a graph, so an invalid name is reported even against a target that is not an initialized graph.

#### `arc lint rules`

- **FR-009**: The tool MUST provide a sibling command to lint that lists every rule the lint checks implement.
- **FR-010**: Each listed rule MUST show its exact name — the same identifier `--skip` accepts for that rule — paired with a human-readable description of what the rule checks.
- **FR-011**: The rule-listing command MUST succeed regardless of whether the current directory holds an initialized graph, and MUST take no graph-specific input.
- **FR-012**: The rule-listing command's output MUST be deterministically ordered and MUST be identical across repeated runs with nothing else changed.
- **FR-013**: The rule-listing command MUST support the tool's existing machine-readable output mode, carrying the same name/description pairing as structured data.
- **FR-014**: The set of rules the rule-listing command lists MUST always match, one-to-one, the set of rule names `--skip` accepts — the two MUST NOT drift apart as rules are added, renamed, or removed.

### Key Entities

- **Lint Rule**: One named conformance check lint implements. Carries an exact identifier (the value both `--skip` and the rule-listing command key on) and a human-readable description of what it checks. The same catalog of rules backs both `--skip`'s validation and `arc lint rules`' listing, so they can never disagree about what exists.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can eliminate one specific rule's violations from both the lint report and its outcome by naming it in `--skip`, while every other rule's violations remain reported exactly as before — verified 100% correctly against a graph with a known mix of violations across multiple rules.
- **SC-002**: A user with no prior knowledge of the tool's rule set can find the exact name and plain-language meaning of every implemented rule within 10 seconds, by running one command, without reading any source code or specification document.
- **SC-003**: 100% of `--skip` values naming a rule the tool does not implement are rejected immediately, with the invalid name identified in the message, before any lint check runs — never silently accepted or silently ignored.
- **SC-004**: Skipping every rule the tool implements at once always completes successfully and reports zero violations, verified against a graph with a large, known violation count across every rule.
- **SC-005**: Repeated runs with the same `--skip` value against an unchanged graph produce identical output, in both human-readable and machine-readable form, 100% of the time.

## Assumptions

- The rule identifiers `--skip` accepts are the same stable names already used internally to distinguish one conformance check from another, presented and matched case-sensitively as exact values — a public vocabulary a user can depend on across releases, not free-text matching or fuzzy lookup.
- `--skip` is a single flag whose value is a comma-separated list, matching the user's own description ("uses comma to separate multiple values"), rather than a flag repeated once per rule.
- An unknown rule name in `--skip` is treated as a refusal — a non-zero exit with a clear message and no partial lint run — rather than a silent no-op or a warning, consistent with this codebase's existing convention of failing clearly on bad input rather than degrading silently. This protects a user or a CI script from a typo that quietly stops suppressing anything.
- `arc lint rules` is read-only reference data with no dependency on any particular graph; it takes no positional arguments and requires no initialized graph in the working directory.
- Skipping a rule applies uniformly to every violation that rule would otherwise produce, including violations that span the whole graph rather than belonging to one node (for example, a duplicate-identity check), not only to violations owned by a single node.
- `--skip` is a per-invocation flag with no persistent, project-level equivalent in this feature's scope; a user or script that wants the same rules skipped again must pass `--skip` again.
- The rule catalog `arc lint rules` lists is exactly the catalog `--skip` validates against, sourced from one place, so adding, renaming, or removing a rule can never leave the two out of sync.
