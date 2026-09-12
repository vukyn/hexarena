---
name: the-roster-row-widening
description: Step 1 of making room in tui.Roster for element/attack/defence columns — done on branch feat/the-roster-row-pays-for-itself; ceiling 119→110 (9 cells), measured widest 119→100, and the next step must cut effectsRoom because 9 is not the 28 the columns need
metadata:
  type: project
---

The owner wants three more roster columns — **element, attack, defence**. They do
not fit: the widest roster row recorded in the goldens was 119 cells of a 119
budget (a 120-column floor with the last cell left empty). Step 1 bought room and
deliberately did **not** add the columns.

What shipped (branch `feat/the-roster-row-pays-for-itself`, not committed by me):
a permanent effect no longer draws ` (always)` (1123 entries across the goldens
against 192 timed ones), and the 5-cell tag column and 21-cell name column merged
into one 17-cell column `%-2s ` + name. 753 golden lines moved across four files
plus one block in `README.md`, which `make golden` does **not** regenerate.

Two follow-ups landed in the same branch after review. **The heading is
localised** — `tui.Roster` now takes an `i18n.Lang` the way `tui.Detail` does,
because the heading carries the convention the change created (`no countdown
means permanent` / `không đếm ngược là vĩnh viễn`) and a rule left in one
language is a rule its reader does not have; `hp` and `spd` stay untranslated
under internal/i18n's own stat-label rule. And **`internal/tui/testdata/crowded.golden`**
puts the elided state back into the record: the change bought back enough width
that nothing on the bench overflowed any more, so the goldens went from four `+N`
rows to zero and what an elided row looks like was pictured nowhere.

**Why:** the arithmetic in the brief was content arithmetic. The row's *ceiling*
went 119 → **110** — 9 cells, all of them from the merge — because `effectsRoom`
was frozen at 60 rather than re-derived; the measured widest row went 119 → 100.
The three columns need roughly 28 in the worst case (a dual affinity renders as
`electric/metal`, 14–17 cells, plus two four-digit stat columns).

**How to apply:** step 2 has **9 cells** guaranteed, not 28, so it has to take the
rest out of `effectsRoom`. That is now cheap in a way it was not before — a
permanent entry fell from 19 cells to 10, so the same 60-cell column holds about
twice as many effects — but it is a real information cut and needs its own
measurement and its own golden read. `TestTheWidestRosterRowLeavesTheRoomTheNextColumnsNeed`
in `internal/tui/roster_bound_test.go` pins 110 and the 9, so that step cannot
spend the room twice; `TestEveryNameTheRosterCanDrawFitsItsColumn` pins the
13-cell name budget the merge left with no slack.
See [[a-cap-rederived-from-what-is-left-spends-the-gift]].
