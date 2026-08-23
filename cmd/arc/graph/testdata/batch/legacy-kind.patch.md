---
kind: patch
document: turing-2025-legacy-kind
published: 2025-06-18
title: "A patch still keyed with the retired manifest identity"
---
# Source

## turing-2025-legacy-kind
```yaml
"@id": "turing-2025-legacy-kind"
"@type": Source
title: "A patch still keyed with the retired manifest identity"
authors: [Test Author]
published: "2025-06-18"
```

This document is otherwise well-formed: only its manifest identity is wrong.
It declares the pre-0.5 `kind: patch` key and no `"@type"`, so `arc apply
batch` must report it **by name** under `failed` — never pass it over as
ordinary Markdown (spec 021 FR-008, SC-005). It is the only fixture in this
tree that can demonstrate that distinction.
