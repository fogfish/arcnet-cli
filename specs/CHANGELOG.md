# Changelog

# 2026-07-04

/speckit-specify `arc grep [<filter>] <pattern>` — scan nodes matching the filter (see Filtering) for lines matching the regexp `<pattern>`; print `<kind>  <id>  <line-number>  <matched line>`, one match per output line; without a filter, scans every node file; suitable for piping to standard tools.

/speckit-plan implement grap grep as part of `internal/app/graph` domain, also maintain same hierarchy in `cmd/arc/graph`. UX implementation and usability MUST BE according to ADR 002 UX Design System (002-ux-design-system.md). Use colors to highite matched text if color mode is enabled. If macthed line longer that 80 chars (configurable via `.arc/config`) the ellipse before and after to fit the match roughtly to one terminal line. 

However, make a file-system grep utility as a reusable, performance optimized packaged at `internal/pkg/grep`. The utility must:
* Use parallel walker of directory traversal.
* Use parallel file processing.
* Use a bounded worker pool (number of workes configurable via `.arc/config` default is 8) and close files after processed.
* Use buffered reads (bufio.Reader).
* Buffer reuse with sync.Pool (minimize memory allocation).
* Be configured for particular file extension (*.md by default).
* Literal search with bytes.Contains when possible.
* Regex only when the query actually requires it.
* Treat files as plain text within the lib.


---

/speckit-specify Make a schema as a first class citizen of the graph. Instead of `_meta` and `.arc/config` a new folder `_schema` is defined. The folder contains subfolders: (a) `nodes/` contains a document per node kind (e.g. entity.md) and `predicates/` contains a documents per predicate (e.g. related.md). Each of them has `id` equal to file base name (equal to name of this entity) and `kind: schema`. The nodes document also contains a `merge` attribute. It substitude `.arc/config` behaviour. The schema is created by `arc init` for core specification (see https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/ARCNET-CORE.md). The schema is extended by `arc apply` when new node kind or predicate is discovered in the graph. 

/speckit-plan schema as own domain `internal/app/schema`. Remove "merge" configurability from `.arc/config` but keep the config infrastructure alive, just remove the github downloader, it is not relevant anymore. Integrate `schema` domain with `apply` and `init`. Isolate ALL ARCNET-CORE abstractions, definitions, const and invariants within `schema` domain. It MUST BE a single entity in the app that has dependencies to https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/ARCNET-CORE.md specification. 


# 2026-07-03

/speckit-specify `arc lint` — run the full CORE §14 checklist across every node and report violations with file path and line number for each: valid YAML front-matter and `kind` field; unique basenames (CORE §3.2); every `[[link]]` resolves to an existing basename; `source` citekey `id` equals its basename (CORE §6.2); `entity` four-word decoded Sowa `category` (CORE §9.2.1); derived nodes link back to at least one `source` (CORE §3.4); predicates are camelCase and registered in `_meta/predicates.md` (CORE §7.3); citations use a registered `cito:`-aligned predicate (CORE §8); each document is exactly one `graph(ingest):` commit (CORE §11.1); extension kind conformance per the kind's profile checklist and graph nodes does not have any active merge conflicts.

/speckit-plan liner is own domain `internal/app/lint`. Also maintain same hierarchy in `cmd/arc/lint`. UX implementation and usability MUST BE according to ADR 002 UX Design System (002-ux-design-system.md). `arc lint` in the normal verbosity mode shows only nodes with issues. `arc lint -v` in the verbose node shows status for each node. In the end it shows the overall graph status.

# 2026-07-02

/speckit-specify `arc apply <patch.md>` — apply a patch file to the graph (CORE §12.3): parse the patch manifest (`kind: patch`, `document`, `published`, `stats`); check idempotency and skip with a clear message if `sources/<id>.md` is already tracked (CORE §11.2); for each H1/H2 node section reconstruct the node object (ARCNET-AST §4 ); **create** new node files when the basename does not exist; **merge** into existing files per the kind's declared merge operation — `none` for `source`, `union` for `entity`, `union first-writer` for `resource`, and per-profile operation for domain/extension kinds (CORE §10); derive and append timeline entries from the source's `published` date (CORE §9.4); produce exactly one git commit with the mandatory subject, stats, and `Source-Id:` trailer (CORE §11.3); update the local index cache (Phase 4) atomically within the same filesystem transaction

See specifications:
* CORE https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/ARCNET-CORE.md
* ARCNET-AST https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/ARCNET-AST.md

/speckit-plan graph I/O is own domain `internal/app/graph`. Also maintain same hierarchy in `cmd/arc/graph`. Define the graph AST https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/ARCNET-AST.md as core domain (`internal/core`). Parse the patch using `github.com/yuin/goldmark` markdown parser into AST and then use AST to patch the graph itself. UX implementation and usability MUST BE according to ADR 002 UX Design System (002-ux-design-system.md). Establish YAML based config at `.arc/config.yml` (as part of `arc init`), this version of the config defines only merge rules per kind, the config management is implemented at `internal/app/config`. The default merge rules for all supported profiles is defined at `github.com/fogfish/arcnet-spec`. 

---

/speckit-specify `arc init [<dir>]` — initialize a new knowledge graph: create the canonical folder layout (`sources/`, `entities/`, `resources/`, `timeline/yearly/`, `timeline/monthly/`, `_meta/`); write stub files `_meta/predicates.md` and `_meta/aliases.md`; create `.arc/` for arc-managed state (see Graph Root from VISION.md); run `git init` and create `.gitkeep` for empty folders; write `.gitignore` excluding `.arc/`; stage everything and produce the initial commit `graph(init): empty knowledge graph` (CORE §11 https://raw.githubusercontent.com/fogfish/arcnet-spec/refs/heads/main/ARCNET-CORE.md)

/speckit-plan graph management (control plane) is own domain `internal/app/ctrl`. Also maintain same hierarchy in `cmd/arc/ctrl`. Integrate `git` as first class concept via invocation of command-line tool, informing user of `arc` about the git tool progress. UX implementation and usability MUST BE according to ADR 002 UX Design System (002-ux0design-system.md).       

# 2026-07-01

/speckit-specify setup the infrastructure for development of cli called `arc`. The infrastructure includes (1) an empty cobra application; (2) github actions to test, check and release application; (3) goreleaser configuration and github actions integrations.

/speckit-plan setup the infrastructure following the mandatory libraries defined by the constitution.md. Use https://github.com/fogfish/iq/tree/main as an example on how to setup GitHub Action and GoReleasing for testing https://raw.githubusercontent.com/fogfish/iq/refs/heads/main/.github/workflows/check-test.yml, linting (staticcheck) https://raw.githubusercontent.com/fogfish/iq/refs/heads/main/.github/workflows/check-code.yml and releasing https://raw.githubusercontent.com/fogfish/iq/refs/heads/main/.github/workflows/build.yml https://raw.githubusercontent.com/fogfish/iq/refs/heads/main/.goreleaser.yaml

