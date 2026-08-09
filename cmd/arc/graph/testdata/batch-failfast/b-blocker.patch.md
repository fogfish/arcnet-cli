---
kind: patch
document: beta-2025-blocker
published: 2025-03-14
title: "Blocker — parses cleanly, fails during application"
---
# Source

## beta-2025-blocker
```yaml
"@id": "beta-2025-blocker"
"@type": Source
title: "Blocker — parses cleanly, fails during application"
authors: [Beta Author]
published: "2025-03-14"
```

This patch is perfectly well-formed, so it classifies successfully and holds
its real position in the publication order (research.md D5b). It fails while
being *applied*, because the test seeds `entities/Blocker.md` as a directory
in the graph — the merge read of that existing node cannot succeed.

## Mentions
- mentions:: [[Blocker]]

# Entity

## Blocker
```yaml
"@id": "Blocker"
"@type": Entity
category: [independent, abstract, occurrent, script]
```

The entity whose on-disk node file the test has replaced with a directory.
