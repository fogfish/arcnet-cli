# ADR 001 - Principles of System Architecture (CLI Edition)

**Status**: Accepted
**Date**: 2026-06-30

## Context

We are aiming for a composable and maintainable monorepo architecture to support organic evolution of a command-line application built in Go with `github.com/spf13/cobra`. The following principles define the system architecture and structure of the solution, adapting the project's hexagonal/onion architecture cornerstones to a single-binary CLI tool and to the [Command Line Interface Guidelines](https://clig.dev) (CLIG), as required by the project constitution (Principles III, VII, IX, XIII).

A CLI tool differs from a server-side system in one structural way that matters for architecture: there is exactly **one** deployable artifact (the binary), and exactly **one** kind of primary driver (the command line itself, invoked by a human or a script). Every use-case in the system is reached through the same Cobra command tree, not through independently deployable entry points. The principles below account for that.

## Decision

We apply three cornerstones for evolving system architecture, identical to the project's general architecture ADR:

1. [Screaming architecture](https://blog.cleancoder.com/uncle-bob/2011/09/30/Screaming-Architecture.html) philosophy that highlights use-cases over framework structure.
2. [Hexagonal architecture](https://alistair.cockburn.us/hexagonal-architecture/) supports a composable solution, equally drivable by a human at a terminal or by another program through a pipe, evolved independently from Cobra and from any specific external system (cloud SDK, REST API, filesystem).
3. [Onion architecture](https://herbertograca.com/2017/11/16/explicit-architecture-01-ddd-hexagonal-onion-clean-cqrs-how-i-put-it-all-together/) organizes business and DDD logic through composable abstraction of the hexagon, driving dependencies inward, toward the domain.

A fourth, CLI-specific constraint governs the outermost layer:

4. The command-line surface — the sole primary (driving) adapter — MUST conform to [clig.dev](https://clig.dev): noun/verb command grammar, `--help` on every command, `--json`/`--plain` as the stable scripting contract, TTY-aware output, and the other rules codified in the project constitution (Principle IX, X).

The solution organically consists of:

1. **Domain is the core** building blocks. It represents real-world concepts (entities and types) in the problem domain, independent of Cobra, of any specific cloud provider, and of the terminal. The core domain defines the common vocabulary of data structures interpretable by humans and machines in the context of the application. It guarantees an interoperability baseline for software components within the application, in the absence of strong content-negotiation techniques. Ideally, the compiler acts as a theorem prover for the application.
2. **Application Services** contain the logic to unfold features, stories, and use-cases. Use-cases are first-class citizens in the architecture (see screaming architecture). Each use-case is a cohesive module with its application logic, domain logic, ports, and adapters organized together within `/internal/app`.
3. **Ports & Adapters**: the architectural pattern defines **primary adapters**, used to tell the application to do something, and **secondary adapters**, told by the application to do something. For this CLI, there is exactly one family of primary adapters — Cobra commands under `/cmd` — and many possible families of secondary adapters (cloud SDK clients, REST clients, the local filesystem, a cache).

## Neglected

Standard package layouts do not provide additional clarity about fine-grained configuration:
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout?tab=readme-ov-file)
- [Standard Package Layout](https://medium.com/@benbjohnson/standard-package-layout-7cdbc8391fc1)

We also neglected flattening every subcommand directly into `/cmd` with no further structure (the pattern used by very small, single-file CLI tools). It does not scale once the tool grows beyond a handful of commands, and it tempts business logic to leak into Cobra `RunE` functions.

Our approach is a combination of multiple architecture patterns; none dominates in our solution:
- [Screaming architecture](https://blog.cleancoder.com/uncle-bob/2011/09/30/Screaming-Architecture.html)
- [Hexagonal architecture](https://alistair.cockburn.us/hexagonal-architecture/), [Hexagonal Architecture in Go](https://medium.com/@matiasvarela/hexagonal-architecture-in-go-cfd4e436faa3)
- [Onion architecture](https://herbertograca.com/2017/11/16/explicit-architecture-01-ddd-hexagonal-onion-clean-cqrs-how-i-put-it-all-together/)
- [Command Line Interface Guidelines](https://clig.dev), governing the outermost (primary adapter) layer specifically

## To Achieve

Architectural simplicity and maintainability, with a command surface that remains predictable and scriptable as the number of subcommands grows.

## Accepting

The downside of the pattern is extra complexity in the definition of interfaces, structs, and conversion of domain models (aka boilerplate) — and, specific to a CLI, an extra translation step between Cobra's `*cobra.Command`/flag types and the plain Go types a use-case actually needs. We accept this cost because it keeps `go test ./internal/...` runnable with no terminal, no network, and no live credentials.

## Implementation notes

### Domain

For us, domain is **the core** building blocks. It represents real-world concepts (entities and types) in the problem domain. The core domain defines the common vocabulary of data structures interpretable by humans and machines in the context of the application. It guarantees an interoperability baseline for software components within the application, in the absence of strong content-negotiation techniques. Ideally, the compiler acts as a theorem prover for the application.

We do not see the domain as a monolithic core; screaming architecture is also applicable there. We do not leak across bounded contexts. The domain logic is organized into libraries (aka domain services), each maintaining the finite algebra for a given type or problem domain.

The evolution of domain logic passes through a few phases of development:
1. `/internal/core`: initially, the domain is a solid part of the application — a collection of core types in the context of the application's problem domain. These types are allowed dependencies on themselves or on open-source modules only.
2. `/internal/pkg`: further evolution of core types materializes into a stricter definition of applicability boundaries (aka bounded context). The core types are overgrown with generic, reusable domain logic, which requires isolation for efficient maintainability. This logic forms a domain service, defined as a self-contained Go module (aka library). These modules are constrained to dependencies on open-source modules only.
3. `github.com`: the pinnacle of evolution is promotion of a domain service to a reusable open-source component, importable by other CLI tools or libraries. We recommend spinning it off into its own open-source project that conducts evolution following its own requirements and lifecycle — this is how a domain concept proven inside one CLI (e.g., a rule engine, a metrics aggregator, a manifest parser) becomes a general-purpose Go library.

### Application Services

The application **only implements** the solution because a person (aka "user") needs one — whether that person runs the command interactively or wrote the script that runs it unattended. In that way, it tells a story to people in various roles. Application services contain the logic to unfold these stories (aka use-cases). Use-cases are first-class citizens in the architecture (see screaming architecture). Each use-case is a cohesive module with its application logic, domain logic, ports, and adapters organized together within `/internal/app`.

Use-cases are strictly decoupled. A use-case has no direct knowledge of, or dependency on, any other use-case. It has no reference to any fine-grained code unit from another use-case — not even structs or interfaces.

We use Dependency Injection (no "foreign" services are instantiated inside the use-case) and Dependency Inversion (a use-case depends on abstractions, not concrete implementations) to decouple use-cases. Go interfaces give us effective Dependency Injection/Inversion within use-cases, so the architecture can evolve — for example, swapping a cloud SDK client for a fake in tests, or adding a second adapter for the same port (e.g., a REST client alongside a cloud SDK client) — without touching the use-case itself.

Only Go structs violate this principle — it is impossible to maintain the "no reference" rule unless the structure is defined via an interface or shape abstraction. We solve this with restrictions:
1. Only domain types are allowed within "public" use-case interfaces.
2. Alternatively, a shared kernel as application-core functionality decouples dependencies through definition of basic input/output specification objects and global ports (`/internal/bios`).

A use-case can be generalized as a process:
- use an adapter to retrieve one or several instances of domain types (from a cloud API, a file, a cache);
- tell those instances to do some domain logic;
- use an adapter to persist or emit the result (write a file, call a cloud API, print to stdout via the formatter).

Unlike the server-side variant of this architecture, a CLI use-case's output is not "persisted" in a database by default — its terminal step is almost always **formatting a result for the primary adapter to print**. The use-case returns a domain value; the Cobra command (primary adapter) is responsible for rendering it as a human table, `--json`, or `--plain` output, per Principle X of the constitution. Use-cases MUST NOT format terminal output themselves — that would couple domain logic to presentation and make `--json` and `--plain` impossible to keep in sync.

### Ports & Adapters

In the hexagonal architecture, when any driver wants to use the functionality at a port, it sends a request that is converted by an adapter, for the specific technology of the driver, into a usable procedure call or message, which passes further inward. The application is blissfully ignorant of the driver's technology. When the application has something to send out, it sends it out through a port to an adapter, which produces the appropriate signal for the receiving technology. The architectural pattern defines **primary adapters**, used to tell the application to do something, and **secondary adapters**, told by the application to do something.

For this CLI, there is exactly one primary adapter family: **Cobra commands under `/cmd`**. There is no second driver (no HTTP server, no message queue consumer) unless an ADR explicitly introduces one (e.g., a `serve` subcommand exposing the same use-cases over HTTP) — and if that happens, it is itself just another primary adapter calling the same use-case root package, never a parallel implementation of the use-case's logic.

In Go, ports are interfaces and adapters are their implementations, following the best practice that an interface belongs in the package that uses values of that type, not the package that implements those values. We adjust the realization of the pattern for this project: the secondary ports are defined by the use-case-specific `port` package (aligned with Go principles). The primary port, however, is defined within the use-case-specific root package (against general Go principles, but consistent across this codebase) — this is the `component.go` file that a single `cmd/` file calls into. We implement secondary adapters provided to a use-case within its `service` package and within its `adapter` package.

**Port isolation rules** (to prevent cross-use-case coupling):

1. **A use-case's `port/` package is private to that use-case.** Other use-cases MUST NOT import it directly. If use-case B needs a behaviour that use-case A's infrastructure provides, use-case B defines its own narrow port interface (Interface Segregation Principle) and the wiring layer (`cmd/`) connects the two. This keeps the hexagon boundaries firm.

2. **Ports must be as narrow as the use-case's actual need.** If a use-case only calls `List`, its port interface declares only `List` — not `Create`, `Update`, `Delete`. Copying a broader interface from another package, or from a vendor SDK's client interface, creates an implicit coupling and violates ISP (see constitution Principle VII).

3. **Ownership of a side-effecting, non-idempotent operation follows the service that performs it.** The service that calls a creating, deleting, or otherwise state-mutating operation against an external system is the sole owner of that call. Callers MUST NOT repeat it speculatively "to be safe" — invoking a non-idempotent operation a second time may fail outright or silently duplicate state. If a use-case needs to know whether the operation already happened, the owning service exposes a query for that, not an invitation to call the mutating operation again.

The evolution of adapters passes through a few phases of development:
1. `adapter`: initially, the adapter is bound to the context of one use-case.
2. `/internal/adapter`: further evolution of an adapter causes generalization toward application-level reusability. These adapters are grouped by technology dependency (e.g., one cloud provider's SDK, one REST API).
3. `github.com`: the pinnacle of evolution is promotion of an adapter to a reusable open-source module, usable across multiple CLI tools. This is feasible in the context of common technology adaptation based on generic principles (e.g., a generic pagination helper for a cloud SDK, a generic retry/backoff wrapper for HTTP clients).

### Distribution as Code

Unlike a serverless backend, where independently deployable components isolate blast radius per use-case, a CLI tool has exactly **one** deployable artifact: the binary itself. There is no infrastructure stack to provision, no container/lambda packaging, and no CDK app in this repository — `/cmd` contains Cobra command definitions, not infrastructure-as-code.

All use-cases under `/internal/app` are wired together into a single Cobra command tree, composed once in `main.go`, and shipped as one cross-platform, statically linked binary per OS/architecture via an automated release pipeline (e.g., GoReleaser), per the constitution's Distribution, Versioning, Security & Release Engineering principle (Principle XIII). The version reported by `--version` is injected at build time via linker flags, never hardcoded, and never depends on a runtime call to any provisioning system.

### Project Layout

```
root/
├── cmd/                            # entry points: the sole primary (driving) adapter
│   ├── root.go                     # cobra root command, persistent flags, Execute(vsn string)
│   ├── version.go                  # --version command, reports build-time injected version
│   ├── componentX/                 # cobra command wiring for one use-case
│   │   └── componentX.go           # flag/arg parsing, calls internal/app/componentX, formats output
│   └── ...
│
├── internal/                       # internal implementation
│   ├── core/                       # core domain types in the scope of the application's context
│   │   └── ...
│   ├── bios/                       # base input/output system, a shared kernel
│   │   └── ...
│   ├── pkg/                        # generic domain libraries
│   │   ├── libX/
│   │   │   ├── lib.go
│   │   │   └── lib_test.go
│   │   └── ...
│   ├── app/                        # application services, use-cases & business logic
│   │   ├── componentX/              # each use-case encapsulates all related layers
│   │   │   ├── kernel/             # domain objects and logic specific to this use-case
│   │   │   │   └── ...
│   │   │   ├── port/                # secondary ports used by this use-case only
│   │   │   │   └── ...
│   │   │   ├── adapter/             # implementations of ports (secondary adapters)
│   │   │   │   ├── adapterX/        # e.g. a cloud SDK client, a REST client, a file adapter
│   │   │   │   │   ├── client.go
│   │   │   │   │   ├── client_test.go  # unit and integration tests for adapter
│   │   │   │   │   └── ...
│   │   │   │   └── mock/            # fake/mock adapter for testing
│   │   │   │       └── ...
│   │   │   ├── service/             # implementations of use-case logic
│   │   │   │   ├── logic.go
│   │   │   │   ├── logic_test.go    # unit and integration tests for use-case
│   │   │   │   └── ...
│   │   │   ├── README.md            # documentation for this specific use-case
│   │   │   └── component.go         # defines the primary port implemented by the use-case
│   │   └── ...
│   ├── adapter/                    # generic, cross-use-case adapters
│   │   ├── technologyX/            # adapters grouped by technology dependency
│   │   │   ├── adapterX/
│   │   │   │   ├── client.go
│   │   │   │   └── client_test.go  # unit and integration tests for adapter
│   │   │   └── ...
│   │   └── ...
│   └── ...
├── docs/                           # documentation
│   └── ...
├── tests/                          # e2e tests, exercising the compiled binary / root command
│   └── e2e/
│       └── ...
├── .github/                        # CI/CD: build, test, govulncheck, release workflow
│   └── ...
├── .goreleaser.yml                 # cross-platform binary release pipeline, checksums
├── main.go                         # CLI entrypoint: calls cmd.Execute(version)
├── go.mod
└── go.sum
```
