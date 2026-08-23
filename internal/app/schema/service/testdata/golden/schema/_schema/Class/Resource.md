---
"@id": Resource
"@type": Class
---
# Resource

A fragment of an ingested document's content that is relevant to the graph but does not warrant its own dedicated type; tag-classified so a recurring pattern can later be promoted into a proper domain type.

- subClassOf:: [[Node]]

## Optional
- optional:: [[notes]]
- optional:: [[indexed]]

## Requires
- required:: [[text]]
- required:: [[tags]]
- required:: [[mentionedIn]]
