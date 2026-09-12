---
name: the-screen-cast-carries-no-dual-affinity
description: "internal/screen's battle fixture fields ZERO dual-element units at every squad size 1..5 — any per-half claim made there walks singles and reports success"
metadata:
  type: feedback
---

`atABattleOf(t, c, n)` in `internal/screen` builds its sides through
`battleCast`, which picks characters by the size of their kit. Measured at
**every** size from 1 to 5: **no unit carries a dual affinity**. `internal/tui`'s
`opening(t)` fixture does (`ground/metal`, `fire/metal`, `water/ice`,
`grass/electric`).

**Why:** a test asserting that each half of a dual affinity is drawn — coloured,
ordered, joined — written against the screen fixture walks six single-element
units, finds nothing to disagree with, and passes. The premise guard
(`if duals == 0 { t.Fatal }`) is what turned that into a visible failure rather
than a green test about nothing; without it the claim was never exercised.

**How to apply:**

- Make dual-affinity and per-element-half claims in `internal/tui`, where an
  `element.Affinity` can be **built** (`element.Dual`) and every legal pair
  walked, rather than fielded out of a cast.
- If a claim must be made against a fielded bench, assert the count of the case
  you are about (`duals`, `carried`, `elided`) before asserting anything about
  it, and say in the failure what the fixture would have to gain.
- ⚠️ This is a fact about today's fixture cast and today's `battleCast` ordering.
  A `t.Logf` was left in `TestTheRostersElementColoursAreThePalettesOwn` that
  fires if the fixture ever gains one, so the note dates itself.

Related: [[feedback_the_fixture_decides_what_is_visible]],
[[feedback_a_mono_element_mirror_cannot_resolve]],
[[feedback_spar_cannot_see_a_trait_the_first_slot_hides]].
