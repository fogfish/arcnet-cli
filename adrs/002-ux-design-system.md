# ADR 002 - CLI UX Design System

**Status**: Accepted
**Date**: 2026-06-30

## Context

[ADR 001](CLI-ADR-001.md) fixes the structural architecture of a Cobra-based CLI (hexagonal layering, `cmd/` as the sole primary adapter, ports/adapters for external systems). It does not say what the command surface should *look and feel like* to the person or script invoking it. The project constitution, in turn, states the user-facing rules ([clig.dev](https://clig.dev) compliance, Principles IX–XII) but a rule like "output MUST adapt to its audience" or "destructive operations MUST require confirmation" is not, by itself, enough for an autonomous coding agent to produce a *consistent* command without re-deriving the same micro-decisions — which flag shorthand to reuse, how a `--json` mode is wired up, what an error message looks like — every single time a new command is added.

This ADR exists to close that gap: it is a **design system**, not a restatement of clig.dev. Each numbered decision below (`DS-01`, `DS-02`, …) fixes one recurring UX decision to a single concrete pattern, with the Go shape an agent should reproduce and the clig.dev rule it implements. This ADR does not ask contributors to read clig.dev independently — the [CLIG Compliance Checklist](#clig-compliance-checklist) at the end of this document translates the full clig.dev rule set (108 rules) into verifiable checkboxes, each annotated with the DS or constitution principle that implements it. An agent MUST work through that checklist when generating or reviewing any command. Where this ADR is explicit, follow it; where only a checklist entry exists, that entry is the complete rule.

The patterns below were not invented for this document. They are extracted, verified line-by-line against two real, public Cobra CLI codebases — one a single-binary AI workflow tool, the other an AWS RDS diagnostic tool — so that "the design system" means "the pattern these two tools already converged on independently," not a hypothetical ideal.

## Decision

### Vocabulary

| Term                   | Meaning                                                                                                    |
| ---------------------- | ---------------------------------------------------------------------------------------------------------- |
| **Leaf command**       | A `cobra.Command` with a `RunE`, reachable by a full path (e.g. `tool agent batch`)                        |
| **Options struct**     | A package-level `fXxx` value grouping one concern's flags, with `apply(cmd)` and `build(...)` methods      |
| **Renderer / Printer** | A port that turns a domain value into output bytes for one output mode (human, `--json`, silent)           |
| **Schema / Theme**     | A struct of `lipgloss.Style`/icon values selected once, by color-on/off, never branched on inline          |
| **Reporter**           | A port for progress/status events during a long-running command, with a no-op (Null Object) implementation |
| **Hint**               | A short, dimmed suggestion for the next command to run, printed to stderr after the main result            |

### DS-01: Command Grammar & Naming

**Status**: Accepted

**Context**: clig.dev requires consistent subcommand naming and discourages ambiguity (constitution Principle IX). Left unstated, an agent will alternate between `tool create resource` and `tool resource-create` across features.

**Decision**: Use **noun → verb** ordering for every subcommand path: `tool <resource> <action>` (e.g. `agent batch`, `agent serve`, `config`). A bare top-level verb is permitted only when the tool has a single, obvious subject (e.g. `check`, `list`, `show` in a tool whose entire domain is one resource type). Once a project picks one of these two shapes for its first command, every subsequent command MUST follow it — this is decided once, in this ADR or its project-specific successor, never re-decided per feature.

```go
// noun → verb, nested under a resource command
rootCmd.AddCommand(agentCmd)
agentCmd.AddCommand(agentBatchCmd)
agentCmd.AddCommand(agentServeCmd)
```

**Consequences**: `tool <resource> --help` shows every action on that resource in one place; tab-completion groups naturally; an agent extending the tool always knows where a new action attaches.

### DS-02: Flag Architecture — Options Structs

**Status**: Accepted

**Context**: A growing command tree accumulates flags. Declaring `cmd.Flags().StringVar(...)` inline inside each command's `init()` scatters related flags across files and invites copy-paste drift (two commands quietly using different defaults for "the same" flag).

**Decision**: Group every related set of flags into one package-level **options struct** with two methods:
- `apply(cmd *cobra.Command)` — registers the flags (on `cmd.PersistentFlags()` if shared by subcommands, `cmd.Flags()` if local to one command) and their defaults/help text
- `build(...) (T, error)` — validates the collected flag values and constructs the domain/service value the command needs; returns an error for invalid combinations instead of letting invalid state reach business logic

```go
var fagent optsAgent

type optsAgent struct {
    file     string
    splitter string
    array    bool
    merge    bool
}

func (opts *optsAgent) apply(cmd *cobra.Command) {
    f := cmd.PersistentFlags()
    f.StringVarP(&opts.file, "file", "f", "", "Path to definition file")
    f.StringVar(&opts.splitter, "splitter", "none", "Split input into sentence, paragraph, or chunk")
    f.BoolVar(&opts.array, "array", false, "Pass inputs as an array")
    f.BoolVar(&opts.merge, "merge", false, "Combine inputs into a single document")
}

// validate rejects flag combinations that don't make sense together
func (opts *optsAgent) validate() error {
    if opts.array && opts.merge {
        return fmt.Errorf("--array and --merge are mutually exclusive (--array collects inputs as array, --merge combines as single document)")
    }
    return nil
}

func (opts *optsAgent) build(/* deps */) (*service.Worker, error) {
    if err := opts.validate(); err != nil {
        return nil, err
    }
    return service.New().Splitter(opts.splitter).Array(opts.array).Build()
}
```

A single options struct MAY be `apply`'d to more than one command (e.g. global output flags applied to the root command and inherited by every subcommand via `PersistentFlags`); a command MAY combine several options structs (e.g. a command that needs both input and output options calls `finput.apply(cmd)` and `foutput.apply(cmd)`).

**Consequences**: Flag validation has exactly one place to live (`validate()`), reusable across every command that shares the struct. An agent adding a new flag to an existing concern edits one struct, not N command files. This directly implements clig.dev's "be consistent across subcommands" rule (constitution Principle IX) by construction rather than by convention alone.

### DS-03: Global Persistent Flags & Their Shorthands

**Status**: Accepted

**Context**: clig.dev names a small set of conventional single-letter flags. Left to chance, two commands in the same tool will assign `-o` to two different meanings.

**Decision**: Reserve these shorthands project-wide, applied once on the root command as persistent flags, inherited by every subcommand:

| Flag           | Shorthand | Meaning                                                     |
| -------------- | --------- | ----------------------------------------------------------- |
| `--help`       | `-h`      | Cobra-provided, never reassigned                            |
| `--version`    | (none)    | Cobra-provided via `rootCmd.Version`                        |
| `--quiet`      | `-q`      | Suppress progress output; errors still shown                |
| `--verbose`    | `-v`      | Show additional diagnostic detail                           |
| `--json`       | (none)    | Machine-readable structured output                          |
| `--color`      | `-C`      | Force-enable color (auto-detected otherwise)                |
| `--profile`    | `-p`      | Named configuration profile                                 |
| `--output`     | `-o`      | Output file path                                            |
| `--output-dir` | `-O`      | Output directory/bucket                                     |
| `--input-dir`  | `-I`      | Input directory/bucket                                      |
| `--name`       | `-n`      | A primary resource name, when the command needs exactly one |

A project MAY add command-local shorthands beyond this table, but MUST NOT reassign a shorthand already listed here to a different meaning anywhere in the command tree (Principle IX: "consistent flag naming").

**Consequences**: A user who has learned `-q`/`-v`/`-o` on one command of the tool never has to relearn them on another. New commands inherit this vocabulary for free instead of inventing it.

### DS-04: Output Renderer Pattern

**Status**: Accepted

**Context**: Every command must support both a human-readable default and a `--json` mode (constitution Principle X). Branching on `if outJSON { ... } else { ... }` inline inside business logic, repeated per command, violates Principle III (formatting is presentation, not domain logic) and drifts: one command's switch lists `silent → json → verbose` in that priority order, another lists them differently, and a third forgets to wire `--json` at all. Worse, adding a new global output mode later (e.g. `--plain`) means hunting down and editing every command's switch by hand — exactly the "manual, per-call-site" maintenance burden a port/adapter design is supposed to eliminate.

**Decision**: Resolve *which* mode is active in exactly one shared function, and let each command **register** its renderers into a generic registry rather than branch on flags itself. A command supplies only the human-facing renderer(s) it actually needs custom formatting for; `--json` and `--silent` are answered automatically by the registry for every command, with zero per-command wiring.

```go
// Mode is the single resolved output mode for this invocation.
type Mode int

const (
    ModeHuman Mode = iota
    ModeVerbose
    ModeSilent
    ModeJSON
)

// ResolveMode is the ONLY place flag-to-mode priority is decided.
// Every command calls this; no command re-derives the priority order.
func ResolveMode() Mode {
    switch {
    case outSilent:
        return ModeSilent
    case outJSON:
        return ModeJSON
    case outVerbose:
        return ModeVerbose
    default:
        return ModeHuman
    }
}

// Renders a domain value T to output bytes for exactly one output mode.
type Printer[T any] interface {
    Show(T) ([]byte, error)
}

// Registry binds a domain type T's bespoke human-mode renderers.
// JSON and Silent need no entry — the registry supplies them for
// every T automatically, so a new global mode added here is
// instantly available to every command without touching cmd/*.go.
type Registry[T any] struct {
    Human   Printer[T]
    Verbose Printer[T] // optional; falls back to Human if nil
}

func (r Registry[T]) Resolve(mode Mode) Printer[T] {
    switch mode {
    case ModeJSON:
        return jsonPrinter[T]{} // generic, works for any T, never registered by hand
    case ModeSilent:
        return nonePrinter[T]{}
    case ModeVerbose:
        if r.Verbose != nil {
            return r.Verbose
        }
        return r.Human
    default:
        return r.Human
    }
}
```

```go
// cmd/list.go — the ONLY thing a command author writes:
// the bespoke human-readable renderer(s), registered once.
var listRenderers = output.Registry[domain.Resource]{
    Human:   humanList,
    Verbose: verboseList,
}

func runList(cmd *cobra.Command, args []string, api Service) error {
    result, err := api.List(cmd.Context())
    if err != nil {
        return err
    }
    out := listRenderers.Resolve(output.ResolveMode())
    return stdout(out.Show(result))
}
```

Every command that produces structured output MUST construct one `Registry[T]` for its domain type and resolve through `Registry.Resolve(ResolveMode())`. A command MUST NOT write its own `switch`/`if` chain over `outVerbose`/`outSilent`/`outJSON` to pick a renderer — that logic lives exactly once, in `ResolveMode()` and `Registry.Resolve`. `jsonPrinter[T]`/`nonePrinter[T]` (the generic JSON/silent renderers) MUST be implemented once, in the shared output package, and MUST NOT be re-registered or reimplemented per command.

**Consequences**: Adding a new global output mode (e.g. `--plain`) is a single change to `Mode`, `ResolveMode()`, and `Registry.Resolve` in the shared package — every existing command picks it up automatically, with no per-command edits and no risk of one command's switch drifting out of sync with another's. A new command's author writes only the renderer(s) that are actually bespoke (human, optionally verbose); `--json` and `--silent` are correct by construction. This is the direct implementation of constitution Principle X's "`--json`/`--plain` are the stable, scriptable contract" — the contract is enforced by the registry, not by each command remembering to honor it.

### DS-05: Color & Theme Schema

**Status**: Accepted

**Context**: clig.dev requires color to be disabled automatically for non-TTY output, `NO_COLOR`, `TERM=dumb`, or `--no-color`/absence of `--color` (constitution Principle X). Raw ANSI escape sequences (`"\033[32m%s\033[0m"`) sprinkled through formatting code make that rule impossible to enforce in one place, impossible to read, and impossible to extend (bold, underline, adaptive light/dark-terminal colors all mean hand-rolling more escape sequences).

**Decision**: **`github.com/charmbracelet/lipgloss` MUST be used for all colored/styled output** — it is a mandatory dependency (constitution: [Mandatory Libraries & Tooling](CLI-DRAFT.md#mandatory-libraries--tooling)), not an optional convenience. Define a single **Schema** (theme) struct holding every `lipgloss.Style` the tool needs, with exactly two named instances — `SCHEMA_PLAIN` and `SCHEMA_COLOR` — and one package-level variable selected once at startup. `SCHEMA_PLAIN`'s styles carry no color/decoration attributes, so `.Render()` returns unstyled text even if accidentally invoked outside the startup check — there is no separate "is color on" branch to forget inside a formatter.

```go
import "github.com/charmbracelet/lipgloss"

type Schema struct {
    StatusOK   lipgloss.Style
    StatusWarn lipgloss.Style
    StatusFail lipgloss.Style
    Hint       lipgloss.Style // for PostRunE next-step suggestions (DS-12)
    IconOK     string // e.g. "✅ " or ""
    IconWarn   string
    IconFail   string
}

var SCHEMA_PLAIN = Schema{
    StatusOK:   lipgloss.NewStyle(),
    StatusWarn: lipgloss.NewStyle(),
    StatusFail: lipgloss.NewStyle(),
    Hint:       lipgloss.NewStyle(),
}

var SCHEMA_COLOR = Schema{
    StatusOK:   lipgloss.NewStyle().Foreground(lipgloss.Color("2")), // green
    StatusWarn: lipgloss.NewStyle().Foreground(lipgloss.Color("3")), // yellow
    StatusFail: lipgloss.NewStyle().Foreground(lipgloss.Color("1")), // red
    Hint:       lipgloss.NewStyle().Faint(true),
    IconOK:     "✅ ", IconWarn: "🟧 ", IconFail: "❌ ",
}

var SCHEMA = SCHEMA_PLAIN // default; flipped once in PersistentPreRun

func setup(cmd *cobra.Command, args []string) {
    if outColor && isTTY(os.Stdout) && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb" {
        SCHEMA = SCHEMA_COLOR
    }
}
```

Formatting code calls `SCHEMA.StatusOK.Render(text)` — it never constructs a `lipgloss.Style` of its own, never contains a raw `\033[` sequence, and never re-checks TTY/`NO_COLOR` state; that check happens exactly once, in the root command's `PersistentPreRun`, per constitution Principle X (`NO_COLOR`, `TERM=dumb`, `--no-color`, non-TTY all force `SCHEMA_PLAIN`).

**Consequences**: Disabling color is a one-line change (`SCHEMA = SCHEMA_PLAIN`) with no risk of a missed `if` somewhere deep in a formatter. Adding a new themed element means adding one field to `Schema` and both instances, not hunting down every print site. Using `lipgloss.Style` instead of hand-written ANSI also gets correct behavior for layout-affecting styling (padding, borders, adaptive colors) for free if a command's output ever grows beyond simple inline color.

### DS-06: Progress & Status Reporting Port

**Status**: Accepted

**Context**: Long-running commands need to report progress to `stderr` (constitution Principle X: responsiveness, animated indicators only on a TTY). If progress calls are scattered as ad hoc `fmt.Fprintln(os.Stderr, ...)` through service code, `--quiet`/`--silent` requires threading a boolean through every call site, and unit-testing the service requires capturing real stderr output.

**Decision**: Define progress reporting as a **port interface** implemented by domain/service code's dependencies, with two implementations: a real one that writes to `stderr`, and a **Null Object** that does nothing. Select the implementation once, at command setup, based on `--quiet`/`--silent` — never branch on those flags again inside service code.

```go
// Port: service code depends on this interface, never on os.Stderr directly.
type Reporter interface {
    Start(label string)
    Step(label string)
    Done(label string, elapsed time.Duration)
    Error(label string, err error)
}

// Real implementation: writes icons + timing to stderr.
type stderrReporter struct{ w io.Writer }

func (r *stderrReporter) Start(label string) { fmt.Fprintf(r.w, "▶ %s\n", label) }
// ...

// Null Object: every method is a no-op, same interface, zero behavior.
type silentReporter struct{}

func (silentReporter) Start(string)                       {}
func (silentReporter) Step(string)                        {}
func (silentReporter) Done(string, time.Duration)          {}
func (silentReporter) Error(string, error)                 {}

func newReporter(quiet, silent bool) Reporter {
    if silent {
        return silentReporter{}
    }
    return &stderrReporter{w: os.Stderr}
}
```

For CLIs whose primary work is a multi-step pipeline with meaningful phase-by-phase progress (an agent runner, a batch processor), apply this port through a richer task-tree renderer rather than flat `Start`/`Step`/`Done` lines — see `DS-08` below.

**Consequences**: Service code is unit-testable with the Null Object and no stderr capture needed (it's already exercised this way by the `sut()` pattern in the testing principle of the constitution). `--quiet`/`--silent` becomes a single constructor choice, not a flag threaded through every call site.

### DS-07: Error Handling, Exit Codes & the Single Error-Formatting Site

**Status**: Accepted

**Context**: If every `RunE` formats its own errors, error presentation drifts (some commands print a raw Go error, others a friendly sentence) and Cobra's default behavior — printing the error *and* full usage text on every failure — buries the actual problem under a wall of flag documentation.

**Decision**:
- Every leaf command sets **`SilenceUsage: true`** and **`SilenceErrors: true`**. Cobra then does not auto-print usage or the error; the command's `RunE` returns a plain `error`.
- **Exactly one site** — the top-level `Execute()` function called from `main.go` — formats and prints the final error, to `stderr`, and sets the process exit code:

```go
func Execute(vsn string) {
    rootCmd.Version = vsn
    if err := rootCmd.Execute(); err != nil {
        msg := err.Error()
        fmt.Fprintf(os.Stderr,
            "\n ❌ %s\n   Run `tool help` for guidance.\n\n",
            strings.ToUpper(msg[:1])+msg[1:])
        os.Exit(1)
    }
}
```

- A command MAY exit with a non-default non-zero code for a meaningfully distinct failure class (e.g. "health check failed" vs "usage error") by setting it explicitly in a `PostRunE`, never by calling `os.Exit` from inside business logic.
- Errors returned from `RunE` MUST already be human-readable sentences by the time they reach `Execute()` (constitution Principle XII: expected errors are rewritten before reaching the user) — wrap with `fmt.Errorf("stage: %w", err)` at each layer so the final message has a causal chain without a Go stack trace leaking through.

**Consequences**: A failing command shows exactly one focused error line plus a pointer to help — never a multi-paragraph usage dump the user has to scroll past to find what went wrong. Exit-code semantics stay centralized and auditable in one function.

### DS-08: Long-Running / Multi-Step Task Output

**Status**: Accepted

**Context**: A command that runs a multi-phase pipeline (load → validate → process → write) needs output that proves it is alive and shows which phase is running, without flooding the terminal (constitution Principle X: responsiveness, suppressed animation off-TTY).

**Decision**: For simple single-phase commands, `DS-06`'s flat `Reporter` port is sufficient. For commands whose primary UX *is* a multi-step task tree — an agent runner, a batch/pipeline tool — apply the deeper task-tree conventions documented in a project's console-UX design system (see [`github.com/fogfish/chalk`](https://github.com/fogfish/chalk)'s `DESIGN_SYSTEM.md` for the canonical, fully worked set of rules: gerund task labels, two-level nesting maximum, a one-metric `Done` suffix, actionable-only notes, a single end-of-run separator, and the iteration/timing thresholds for when to decompose a task into sub-tasks). Where a project depends on `github.com/fogfish/chalk` directly, that document's rules apply verbatim and take precedence over ad hoc progress formatting. Where a project does not take that dependency, replicate only the rules that matter at smaller scale:
- task labels are gerund ("Loading dataset", not "Load dataset" or "Dataset loaded")
- a completed step gets at most one parenthetical outcome metric, never a paragraph
- sub-task nesting never exceeds two levels
- a task expected to run longer than ~5 seconds either decomposes into sub-tasks or emits one progress note partway through — never runs silently

**Consequences**: A project doesn't have to choose between "no progress UX at all" and "reinvent task-tree rendering from scratch" — it picks up a known-good, previously verified rule set sized to how much progress surface the command actually needs.

### DS-09: Input Handling — Files, stdin, and Mounted Directories

**Status**: Accepted

**Context**: clig.dev requires `-` to mean stdin/stdout, and recommends multiple positional arguments of the *same kind* to support globbing (constitution Principle IX). CLIs that process documents or records additionally need a consistent way to accept "many inputs from a directory" without one-off flag names per command.

**Decision**:
- A command that can read piped input MUST check whether stdin actually has data before blocking on it: `fi, _ := os.Stdin.Stat(); if fi.Size() == 0 { /* no piped input, e.g. show usage or fall back to default */ }`.
- Multiple positional file arguments of the same kind are accepted as `args []string` directly (`tool agent FILE1 FILE2 ...`); a command MUST NOT use positional arguments for two semantically different things.
- A directory- or bucket-shaped input (`-I`/`--input-dir`) is its own option, layered on top of the same input-building port, so the same business logic accepts "explicit file list", "piped stdin", or "mounted directory" through one `Source` port without three parallel code paths in the command.

**Consequences**: A command's input story is predictable across the whole tool: positional files, `-I` for a directory, stdin when nothing else is given. An agent adding a new ingest-style command reuses the existing `Source` port instead of inventing a fourth input convention.

### DS-10: Local Configuration & Secrets

**Status**: Accepted

**Context**: clig.dev recommends XDG-style config locations and forbids accepting secrets directly as flag values (constitution Principle XI). A CLI that talks to external providers (cloud accounts, model APIs) typically needs named profiles holding credentials.

**Decision**: Provide a dedicated **`config`** leaf command that writes to a single per-user rc file (e.g. `~/.<tool>rc`, using a `netrc`-style `machine <profile> ... key value ...` block format), keyed by named profile, never accepting a raw secret as a bare positional flag value without an explicit opt-in:

```go
var configCmd = &cobra.Command{
    Use:   "config",
    Short: "configure connection profiles",
    Example: `
  tool config --provider-a              configure provider A
  tool config --provider-b <secret>     configure provider B with a secret
    `,
    SilenceUsage:  true,
    SilenceErrors: true,
    RunE:          runConfig,
}
```

On success, print one short, friendly confirmation — never a silent success: `"\n ✅ All good — you're set up and ready to go!\n"`. If a profile is already configured, `config` MUST be idempotent (detect the existing profile and confirm rather than duplicate the entry).

**Consequences**: Every provider/credential concern funnels through one discoverable command and one file format, instead of each integration inventing its own `~/.toolrc.providerX.json`. This satisfies Principle XI's "ask for consent before modifying configuration… prefer creating over silently appending" by construction (`config` is the explicit, named action that touches the file).

### DS-11: Help Text & Examples Formatting

**Status**: Accepted

**Context**: clig.dev requires `--help` to lead with examples and a description (constitution Principle XII). Cobra's `Long`/`Example` fields are free-form strings; without a fixed shape, every command formats them differently.

**Decision**: For every command's `Long` field:
- Start with a blank line (Cobra renders it directly under the `Usage:` line; a leading blank line avoids the description butting up against it)
- A short paragraph explaining *why* the command exists and what it's for, in plain prose — not a restatement of `Short`
- End with a `See more info <repository-url>` pointer line (constitution Principle XII: top-level help MUST link to web documentation)

For every command's `Example` field: each example is a real, runnable invocation, indented with a leading tab, one per line, ordered from the simplest/most common to the most advanced — never a description of what a flag does (that belongs in the flag's own help text, not in `Example`).

```go
var agentCmd = &cobra.Command{
    Use:   "agent",
    Short: "Run a workflow against the configured backend.",
    Long: `
The agent command executes a defined workflow, processing input through
a configured backend and producing output according to the specified
workflow configuration.

See more info https://github.com/<org>/<tool>
    `,
    Example: `
	tool agent -f <yml>
	tool agent -f <yml> FILE1 FILE2 ...
	`,
}
```

**Consequences**: `--help` output is uniform across every command in the tool, which is what lets a user (or an agent reading its own `--help` output to self-correct) predict where to look for the example they need.

### DS-12: Next-Step Hints

**Status**: Accepted

**Context**: clig.dev rule 25 — "Suggest commands users should run next in your output." After a command completes, the user often does not know what to do next; a single contextual line on stderr removes that friction without polluting stdout or breaking scripts. This is a distinct concern from error messages (DS-07) and from help text (DS-11): hints describe a natural *continuation*, not a failure or a reference.

**Decision**: Place next-step hints in **`PostRunE`** — never in `RunE`. `RunE` is responsible for producing the result or returning an error; `PostRunE` is responsible for the conversational layer that follows success. Hints MUST be suppressed in `--json`, `--plain`, and `--silent` modes (check `output.ResolveMode()` before emitting). Render hints using `SCHEMA.Hint` (DS-05: faint in color mode, plain in plain mode) and write to `stderr`.

```go
func listPost(cmd *cobra.Command, args []string) error {
    switch output.ResolveMode() {
    case output.ModeJSON, output.ModeSilent:
        return nil // no hints for machine-readable or silent modes
    }

    if !outVerbose {
        stderr(SCHEMA.Hint.Render(`(use "tool list --verbose" to see full details)`) + "\n")
    }
    return nil
}
```

When the hint can incorporate a specific value from the current invocation (a flag value, a resource name, an interval), embed it so the suggestion is copy-pasteable without any edits:

```go
func checkPost(cmd *cobra.Command, args []string) error {
    switch output.ResolveMode() {
    case output.ModeJSON, output.ModeSilent:
        return nil
    }

    if rootName != "" && !outVerbose {
        stderr(SCHEMA.Hint.Render(
            fmt.Sprintf(`(use "tool check -v -n %s" to see the full report)`, rootName),
        ) + "\n")
    }
    if rootName == "" && !outVerbose {
        stderr(SCHEMA.Hint.Render(`(use "tool check -v" to see details)`) + "\n")
    }
    return nil
}
```

**Rules**:
- Hints MUST live in `PostRunE`; `RunE` MUST NOT emit next-step suggestions
- Hints MUST be suppressed when `output.ResolveMode()` returns `ModeJSON`, `ModeSilent`, or `ModePlain`
- Hints MUST be rendered with `SCHEMA.Hint.Render(...)` and written to `stderr` (never `stdout`)
- Hint format: `(use "tool subcommand --flag" to <action>)` — lowercase, parenthetical, a real runnable invocation
- At most two distinct conditional hints per `PostRunE`; do not enumerate every possible next action
- Hint phrasing MUST be conditional on the flags actually used in this invocation, never unconditional boilerplate

**Consequences**: A user who sees `(use "tool check -v -n mydb" to see the full report)` knows exactly what to run next, with the exact name already filled in. Scripts and CI jobs invoking `--json` never see hint text mixed into the output stream.

## Accepting

Fixing these twelve decisions trades a small amount of one-time flexibility (a future command can't silently invent its own flag shorthand or error format) for the thing this ADR exists to produce: an autonomous coding agent extending this CLI can implement a *new* command's UX correctly by pattern-matching against `DS-01`–`DS-12` instead of re-deriving CLIG from first principles, or worse, copying the shape of whichever existing command happens to be open in context. The cost is symmetrical with any style guide: it is occasionally more verbose than the minimal one-off solution for a single command, and it requires this document (not the agent's judgment) to be the place new UX decisions get made and recorded.

The [CLIG Compliance Checklist](#clig-compliance-checklist) that follows operationalises all 108 clig.dev rules into a single audit list — agents and reviewers MUST work through it for every new or changed command.

## CLIG Compliance Checklist

An agent MUST verify every applicable item below when generating or reviewing a command or command change. Items are annotated with the DS decision or constitution principle (Const.) that governs the implementation; items with no annotation are direct clig.dev requirements verifiable from source alone.

### Help

- [ ] `-h`, `--help`, and the `help` subcommand all display identical help text and short-circuit all other flag processing (DS-03, DS-11)
- [ ] A command run with missing required arguments prints concise help (description + 1-2 examples + flag summary), never a raw error or panic (DS-07)
- [ ] Every command's `Short`, `Long`, and `Example` Cobra fields are populated — none left empty (DS-11)
- [ ] `Long` starts with a blank line and ends with `See more info <repo-url>` (DS-11)
- [ ] `Example` contains real, runnable invocations in ascending complexity order, each indented with a tab (DS-11)
- [ ] Top-level `--help` includes a support / issue-reporting URL (DS-11)
- [ ] The most common flags and subcommands appear first in help output (DS-11)
- [ ] When a command expects piped input but stdin is an interactive TTY with no data, it displays help and exits cleanly without blocking (DS-09)
- [ ] Typo'd subcommand names or unrecognised flag values suggest a correction in the error message where a correction can be inferred (DS-07)

### Output

- [ ] Primary result output goes to `stdout`; errors, progress, and hints go to `stderr` (DS-06, DS-07, DS-12)
- [ ] `--json` flag emits machine-readable JSON on `stdout` (DS-04)
- [ ] `--plain` flag emits script-friendly tabular text on `stdout` (DS-04)
- [ ] `--quiet` / `-q` suppresses non-essential output; errors are still shown (DS-03, DS-06)
- [ ] `--verbose` / `-v` reveals additional diagnostic detail (DS-03)
- [ ] A successful state-changing operation prints a brief explanation of what changed — never silent success (Const. X)
- [ ] Next-step hints are printed to `stderr` after successful operations that naturally lead to a follow-on command (DS-12)
- [ ] Next-step hints are suppressed in `--json`, `--plain`, and `--silent` modes (DS-12)
- [ ] Large text output to an interactive TTY is either paged (e.g. via `less`) or the user is instructed to pipe to a pager
- [ ] Debug-only output is hidden by default and revealed only under `--verbose` or `--debug` (DS-03, DS-06)

### Color & Terminal

- [ ] All colored/styled output uses `github.com/charmbracelet/lipgloss`; no raw `\033[` ANSI sequences anywhere in source (DS-05, Const. X)
- [ ] Color is automatically disabled when stdout is not a TTY (DS-05)
- [ ] Color is automatically disabled when `NO_COLOR` environment variable is set to any value (DS-05)
- [ ] Color is automatically disabled when `TERM=dumb` (DS-05)
- [ ] `SCHEMA_PLAIN` / `SCHEMA_COLOR` are selected exactly once in `PersistentPreRun` — no per-formatter re-check of TTY or `NO_COLOR` state (DS-05)
- [ ] Color is never the sole carrier of information — every color signal is paired with text or a symbol (DS-05)
- [ ] Animated progress indicators render only to a TTY and are automatically suppressed when output is piped or redirected (DS-06, DS-08)

### Arguments & Flags

- [ ] Flags preferred over positional arguments; at most one positional-argument slot used for a single semantic purpose (DS-02, DS-09)
- [ ] Every flag has a long form (`--flag`); single-letter shorthands reserved only for the most frequently used flags (DS-02, DS-03)
- [ ] Single-letter shorthands follow the reserved table in DS-03 and are not reassigned to a different meaning in any command (DS-03)
- [ ] Standard cross-tool flag names used where conventions exist (`--json`, `--quiet`, `--verbose`, `--output`, etc.) (DS-03)
- [ ] Secrets are NOT accepted as direct flag values visible in `ps` output or shell history (DS-03, DS-10, Const. XI)
- [ ] `-` is accepted as a file argument to mean stdin/stdout wherever the command accepts a file path (DS-09)
- [ ] Mutually exclusive flag combinations are validated in `opts.validate()` and produce a clear error message (DS-02)
- [ ] Dangerous or irreversible operations require explicit confirmation or a `--yes`/`--force` flag for non-interactive use (Const. IX)
- [ ] `--no-input` flag exists for every command that could otherwise prompt for input (DS-03, Const. IX)
- [ ] Flag defaults are appropriate for the most common usage without any extra flags
- [ ] Arguments, flags, and subcommands are order-independent where possible

### Subcommands

- [ ] Every subcommand follows the project-wide noun-verb (or verb-noun) ordering documented in DS-01 for this project (DS-01)
- [ ] No catch-all or implicit subcommands exist (DS-01)
- [ ] No arbitrary subcommand abbreviations; only explicit `Aliases` declared on the `cobra.Command` (DS-01)
- [ ] Flag names and semantics are consistent with sibling subcommands — the same flag name means the same thing everywhere in the tree (DS-02, DS-03)

### Errors & Exit Codes

- [ ] Every leaf command sets `SilenceUsage: true` and `SilenceErrors: true` (DS-07)
- [ ] Error formatting and exit-code setting happen in exactly one place — the top-level `Execute()` function (DS-07)
- [ ] Expected/anticipated errors are rewritten into human-readable sentences before reaching `Execute()`, wrapped with `fmt.Errorf("context: %w", err)` at each layer (DS-07, Const. XII)
- [ ] Unexpected/internal errors include enough detail to file a useful bug report, and optionally a pre-populated issue URL (DS-07, Const. XII)
- [ ] Exit code `0` on success; non-zero on any failure (DS-07)
- [ ] Distinct non-zero exit codes for distinct failure classes are documented if used (DS-07)
- [ ] `os.Exit` is NEVER called from `RunE` or domain logic — only from `Execute()` or explicitly in `PostRunE` (DS-07)

### Interactivity & Signals

- [ ] Prompts and interactive elements appear ONLY when `stdin` is an interactive TTY (Const. IX)
- [ ] Password and secret input hides the typed characters (echo disabled) when prompting (Const. IX)
- [ ] Ctrl-C (SIGINT) exits promptly; if mid-flight, the tool announces what it is cleaning up before exiting (Const. X)
- [ ] A second Ctrl-C force-exits immediately without waiting for cleanup (Const. X)

### Robustness

- [ ] All flag and argument validation happens in `opts.validate()` / `opts.build()` before domain logic runs (DS-02)
- [ ] The command produces some visible output within ~100 ms of invocation for any long operation, confirming it has started (DS-06)
- [ ] Progress indicators are shown for operations taking more than ~1 second (DS-06, DS-08)
- [ ] Network/external calls have sensible default timeouts, overridable by a flag or config value (Const. VII)

### Configuration & Environment Variables

- [ ] Configuration precedence: flags → environment variables → project config → user config → system config (Const. XI)
- [ ] User-facing config file locations follow the XDG Base Directory Specification (or documented OS equivalents for macOS/Windows) (DS-10, Const. XI)
- [ ] The tool asks for confirmation before modifying a config file it does not own; prefers creating a new file over silently appending (DS-10, Const. XI)
- [ ] Tool-specific environment variables use uppercase letters, digits, and underscores with a `TOOLNAME_` prefix (Const. XI)
- [ ] Standard cross-tool environment variables (`NO_COLOR`, `DEBUG`, `EDITOR`, `HTTP_PROXY`, `HTTPS_PROXY`, `ALL_PROXY`, `NO_PROXY`, `TERM`, `PAGER`, `SHELL`, `TMPDIR`) are honored by their canonical names and never reinvented under a project-specific prefix (Const. XI)
- [ ] `.env` file is supported for local development convenience but is not the only supported config mechanism in CI or production (Const. XI)
- [ ] Secrets are NOT stored in plain environment variables when an alternative (secret file, OS keychain, credential provider) exists (Const. XI)

### Naming & Distribution

- [ ] Binary name is lowercase, short, memorable, and specific enough to be unambiguous in a shell PATH (DS-01)
- [ ] Binary name uses only lowercase letters and optionally dashes — no uppercase, no underscores, no leading digits (DS-01)
- [ ] Releases produce a single statically-linked binary per OS/arch target (Const. XIII)
- [ ] Checksum file published alongside every release's binaries (Const. XIII)
- [ ] Uninstallation instructions documented (remove binary + config directory at minimum) (Const. XIII)

### Analytics & Telemetry

- [ ] No usage analytics, telemetry, or crash reporting is collected without explicit, documented opt-in consent (Const. XIV)
- [ ] If telemetry exists: what is collected, why, and how to disable it is documented in the README (Const. XIV)

## Implementation notes

These patterns sit in the architecture defined by [ADR 001](CLI-ADR-001.md):
- `Printer[T]`, `Mode`, `ResolveMode()`, and `Registry[T]` (`DS-04`), and `Reporter`/`Schema` (`DS-05`, `DS-06`), are **primary-port-adjacent**: they are selected from `cmd/`, but the types themselves SHOULD live in one small shared package (e.g. `internal/bios` or `internal/pkg/output`) so use-case code can depend on the `Printer[T]`/`Reporter` interface without importing Cobra, and so every command resolves modes through the same `ResolveMode()`/`Registry[T]` instead of each command importing its own copy.
- Options structs (`DS-02`, `DS-03`) live in `cmd/` only — they are Cobra wiring, not domain logic, per constitution Principle III.
- `ResolveMode()`, the generic JSON/Silent renderers, and any shared `Schema` instances MUST be implemented once and imported everywhere they're needed; a second, slightly different `ResolveMode()` or renderer appearing in a second package is a violation of this ADR and of constitution Principle V (no duplicate, divergent implementations of the same capability).
