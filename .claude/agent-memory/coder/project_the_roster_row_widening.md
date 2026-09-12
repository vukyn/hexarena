---
name: the-roster-row-widening
description: "Both steps DONE. Step 1 bought 9 cells (ceiling 119→110). Step 2 did NOT spend them and did NOT cut effectsRoom — option A: a second table (tui.RosterWide, ceiling 138) offered only where the window reaches a derived threshold of 139, so 120 keeps the shipped narrow table byte-for-byte and 160 gains element/atk/def. ⚠️ the note's own 'step 2 has to cut effectsRoom' premise was FALSE"
metadata:
  type: project
---

The owner wanted three more roster columns — **element, attack, defence**. Two
branches, both now written:

**Step 1** (`feat/the-roster-row-pays-for-itself`) bought room: a permanent
effect stopped drawing ` (always)` and the 5-cell tag and 21-cell name columns
merged into 17. The row's **ceiling** went 119 → **110**, so **9 cells**, not the
28 the brief's content arithmetic promised — `effectsRoom` is a frozen 60 and
`elided` spends content, not bound. → [[feedback_a_cap_rederived_from_what_is_left_spends_the_gift]]

**Step 2** (`feat/a-wide-roster-says-more`, worktree `hexarena-wide`) is **option
A**, and ⚠️ **this note's earlier "step 2 has to take the rest out of
`effectsRoom`" was wrong.** Nothing was cut. `internal/tui` gained a **second**
table:

- `tui.Roster` — unchanged, ceiling 110, what every window at the floor draws.
- `tui.RosterWide` — the same row with `%-16s%4d %4d   ` spliced between the
  tempo and the effects, ceiling **138**.
- `tui.RosterIsWide(width)` — the one declaration of the rule; `tui.RosterFor`
  is the convenience that uses it.

**Why:** the three columns need ~28 in the worst case and the floor only ever
had 9, so at the floor either something is cut or the columns do not appear. The
owner chose "they do not appear", which makes the narrow table a **byte**
promise — held against a verbatim copy of main's code
(`TestTheNarrowRosterDrawsWhatMainDrewToTheByte`), not against a golden the same
change accepts.

**How to apply.** Four facts worth having before touching this again:

- **The threshold is 139 and it is derived, never typed** —
  `rosterWidth(rosterWideRow, …)`, the widest line the format can draw plus the
  cell every line leaves empty. 120 < 139 ≤ 160, so exactly the goldens' roomy
  sections move. → [[feedback_no_arithmetic_test_can_see_a_literal_that_agrees]]
- **The pick happens at DRAW time.** `readBattle` runs inside `Attach`, which is
  per turn, not per frame; `playReading` carries **both** rendered tables and
  `playReading.rosterTable(c)` chooses. A test that re-attaches per size is blind
  to getting this wrong — demonstrated. → [[feedback_readings_beat_draws_only_when_work_moves]]
- **Attack and defence DO have a bound**, and it is not the progression ceiling:
  `modifier.Set.Stat` saturates towards `ceiling × headroom / 1000` and
  `scale.Saturate` never reaches its limit, so the widest figure is `limit − 1` =
  **2399** under the shipped books (ceiling 800, headroom 3000‰) — four digits,
  where the ceiling alone would have said three. Past the column a `%4d` pushes
  its own row right rather than clipping a digit.
- **The element column is sized to the element BOOK, not the cast** — 15 cells
  (`electric/ground`), because `element.Dual` refuses neutral and refuses a
  repeat. The widest shipped affinity is `electric/metal` at 14, which is an
  observation rather than a bound.
