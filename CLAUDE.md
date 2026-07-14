<!-- SPECKIT START -->
For additional context about technologies to be used, project structure,
shell commands, and other important information, read the current plan:
`specs/016-arc-revert/plan.md`
<!-- SPECKIT END -->

## Go file license header

Every `.go` file in this repository MUST start with this exact header,
before the `package` directive (and before any package-doc comment, which
must remain immediately adjacent to `package` with no blank line between
them):

```go
//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package foo
```

When a file has a package-doc comment, the header goes above it, separated
by exactly one blank line, with no blank line between the doc comment and
`package`:

```go
//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

// Package foo does X.
package foo
```

Add this header to every new `.go` file you create, and to any existing
`.go` file that is missing it if you touch that file.
