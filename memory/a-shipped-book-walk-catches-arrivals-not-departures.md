---
name: a-shipped-book-walk-catches-arrivals-not-departures
description: A guard that walks the shipped book and asks a table about each id sees a thing arriving without an entry, never an entry left behind by a thing that left.
metadata:
  type: project
---

`TestEveryShippedStatusIsGlossed` loads `seed.StatusBook()` and asks
`statusGloss` for each id. It exists because three statuses once shipped bare
after a rename, and it catches that. It cannot catch the other direction, and its
own comment licenses the blind spot on purpose — "an unshipped id is still free to
miss, so this does not become a second place a status has to be registered".

Free to **miss** is not free to **linger**. Retiring `same_element` took `kinship`
out of `statuses.json`; had its gloss stayed, nothing anywhere would have said so.
The table would go on translating an effect no unit in the game can hold, and the
next reader would take the entry as evidence the status still exists.

**Why:** the walk's direction is its coverage. Iterating the *shipped* side and
looking up the *table* asserts `shipped ⊆ table` and says nothing about `table \
shipped`. Every registry checked this way has the same shape, and the gap only
opens when something is **removed** — which is rarer than adding, so the guard
looks complete for years.

**How to apply:** when you delete shipped data, grep for every table keyed by its
id before assuming the deletion is complete. When you write a coverage guard over
a registry, ask what the reverse walk would catch and whether anything else does.
Where the reverse walk needs an allowance, derive it from the data — the column
values were allowed for through `composition.ColumnValue` and `hex.FormationCols`,
not through a list of the three that exist today. Same family as
[[a-blanket-refusal-hides-where-its-legs-live]].
