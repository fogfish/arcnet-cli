---
title: Reading notes
tags: [scratch]
---

# Reading notes

Front matter that declares no patch identity at all — neither the current
`"@type": patch` key nor the retired `kind` one. Passed over exactly like
README.md — `core.LooksLikePatch` returns false (spec 020 FR-003, spec 021
FR-008: a file with *no* identity is passed over; a file with a *retired*
identity is a named failure, never passed over).
