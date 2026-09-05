---
"@id": Node
"@type": Class
---
# Node

The graph's implicit universal base type: every content type (Source, Entity, Resource, Timeline, Reference) inherits its Required/Optional contract via rdfs:subClassOf, whether declared explicitly or not. Never itself a node's own @type — it exists only to be inherited from (spec 017).

- optional:: [[published]]
- optional:: [[created]]
- optional:: [[indexed]]
- optional:: [[tags]]
- optional:: [[text]]
- optional:: [[updated]]
- optional:: [[scoreZ]]
- optional:: [[scoreC]]
