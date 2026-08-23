---
"@id": Source
"@type": Class
merge: immutable
---
# Source

A node for one ingested document — the provenance origin other nodes derive from.

- subClassOf:: [[Node]]

## Optional
- optional:: [[authors]]
- optional:: [[url]]
- optional:: [[cites]]
- optional:: [[doi]]
- optional:: [[indexed]]

## Requires
- required:: [[title]]
- required:: [[abstract]]
- required:: [[mentions]]
