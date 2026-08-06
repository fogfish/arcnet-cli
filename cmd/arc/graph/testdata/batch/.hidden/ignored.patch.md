---
kind: patch
document: hidden-2025-ignored
published: 2025-05-05
title: "A patch inside a hidden directory"
---
# Source

## hidden-2025-ignored
```yaml
"@id": "hidden-2025-ignored"
"@type": Source
title: "A patch inside a hidden directory"
authors: [Nobody]
published: "2025-05-05"
```

A perfectly well-formed patch that discovery must never reach, because its
directory name begins with a dot (spec 020 FR-019).
