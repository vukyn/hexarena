---
name: a-cap-rederived-from-what-is-left-spends-the-gift
description: Narrowing a column buys a table nothing while a neighbouring column's cap is derived as "the row's budget minus everything else" — elided immediately redraws into the freed cells; measured on tui.Roster, 18 freed cells bought 0 of row bound
metadata:
  type: feedback
---

`internal/tui`'s `effectsRoom` was documented as *"the row spends 59 cells before
it and the floor is 120 with one held back, so what is left is this"* — a cap
**re-derived from the leftovers**. Two width savings were asked for on that row
and only one of them could possibly land:

- **Merging the tag and unit columns** (26 cells → 17) freed 9. Re-deriving
  `effectsRoom` from the new prefix would have given it 69 instead of 60 and the
  row's ceiling would have stayed at 119 — the merge would have bought *nothing*.
- **Dropping ` (always)`** freed 18 cells of *content* on the worst row and **0
  cells of row bound**, because `elided` fills up to the cap: the 4 rows that
  carried a `+1` marker spent the freed cells on drawing the effect they had been
  eliding. Measured: 749 of 753 changed golden lines were the two transformations
  exactly; the other 4 were that re-pack.

**Why:** a brief priced this change at "28 cells freed". 28 is the *content*
arithmetic. The row's ceiling went 119 → **110**, so 9 were bought, and the
measured widest row in the goldens went 119 → **100**. Content arithmetic and
bound arithmetic are different numbers whenever a column is capped, and only the
bound is what a later column can spend.

**How to apply:** before promising width to a future column, ask which cap the
freed cells fall under. If a neighbouring cap is written as *"whatever is left"*,
freeze it at its current number and say so in its doc comment, or the gift is
consumed the moment it is given. What the cheaper wording *does* buy is **entry
cost** — a permanent effect fell 19 cells → 10, so the same 60-cell column holds
about twice as many — which is what makes that cap cheap to cut later. Say that
instead of claiming width. Related: [[feedback_measure_the_term_before_optimising_it]],
[[feedback_measure_the_thing_a_bound_bounds]].
