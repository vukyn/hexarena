---
name: a-mono-element-mirror-cannot-resolve
description: A probe squad built to fire an ELEMENT bonus mirrors into its own resisting affinity and goes Endless, so the 500‰ control reads 0‰ — kit off the counted element, because the bonus counts affinity not kits
metadata:
  type: feedback
---

**A probe squad built to fire a per-element composition bonus cannot be kitted in
that element, because its own mirror is every attack into a resisting affinity
and the control stops resolving.**

**Why:** `DAT-002` sub-item 1 (2026-09-09) needed `probe_water` — four water
carriers — and gate 1 was "every mirror reads exactly **500‰**". Two failures, in
this order, and the first one hides the second:

1. **`builds.json` `squirtle.fortress` carries no damaging skill at all** (taunt,
   withdraw, wide_guard, aqua_ring). Two of them taunt-and-heal past spar's
   4000-turn limit: the mirror came back **10 of 10 Endless**.
2. Giving Blastoise water damage did **not** fix it — still 10 of 10. The cause
   underneath is the matchup: an all-water side against an all-water side is
   every blow into the element that resists it, so nothing on either half kills.

⚠️ **The symptom is not a red control, it is an unreadable one.** `Tally.Rate()`
drops Endless from the denominator, so an all-Endless cell returns **0‰** — which
looks like a catastrophic gate failure rather than "no battle finished". Read the
endless column before believing a rate; that is the same trap as
`[[hexarena-starter-squads]]`'s healer-versus-healer boasting 85%.

**How to apply:** kit the probe **off** the element it is counted for.
`composition` counts a unit's **affinity**, never its kit's element
(`Member{ID, Affinity, Column}` is the whole of what the counter sees), so every
seat stays a carrier and still receives the grant while fighting with neutral,
ice, wind or fighting skills. `probe_water` resolved the moment Gyarados took
`magikarp.gale` (wind) instead of `magikarp.surge` (water) and Poliwrath took
`poliwag.flurry` (fighting/neutral) instead of `poliwag.riptide` (water).

Two consequences worth carrying to the next probe:

- **Check the mirror at 5 seeds before spending a cell.** A 400-seed all-Endless
  mirror is 800 battles × 4000 turns — it ran **7 minutes** and printed nothing.
  Five seeds answers the same question in half a second.
- **The kits a probe can wear are constrained by its own control**, so the
  fixture is a cost of the instrument and belongs in the write-up beside the
  figure. → `[[feedback_the_fixture_that_makes_it_measurable_has_its_own_price]]`.

An **Endless-heavy mirror is still a valid control** provided it is not total: a
mirror is the same roster in both arrangements, so an Endless battle is Endless
in both halves and drops out symmetrically. `probe_water` read exactly 500‰ on
404/800 Endless. What it cannot survive is 800/800.
