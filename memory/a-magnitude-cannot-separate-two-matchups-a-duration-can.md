---
name: a-magnitude-cannot-separate-two-matchups-a-duration-can
description: "hexarena ENG-006 step 3 — the SAME defence tier is worth 248 vs 216 ungated (1.15x) and 222 vs 38 behind above_health:700 (5.8x); the decoupling axis is WHERE ALONG THE BAR the fight is decided, not battle length, and not how long the gate is open"
metadata:
  node_type: memory
  type: project
---

`pristine` (2026-09-08, ENG-006 step 3) is the first shipped trait with a gate at
the **top** of the health bar and the first with a gate on a **grant**: `plated`
(+10% defence, always) beside the shipped `fortified` (+25%) behind
`above_health: 700`, on Magnezone.

**The number that justified it**, build duel, both arrangements, 500 seeds a side
= 1000 battles a cell, mirror control exactly 500‰ at every rung:

| opponent | the tier UNGATED | the same tier BEHIND THE GATE |
|---|---:|---:|
| Cleffa | +248‰ | +222‰ |
| Gastly | +216‰ | +38‰ |

**Why:** the ungated column is the *payload calibration* and it is the whole
argument. 248 against 216 is 15% apart — **a magnitude cannot tell those two
matchups apart, so no amount of a stat would ever have been a lever between
them**. Behind the gate the same tier is **5.8×** apart. That is the first
reading in this repository where a trait's worth separates two matchups by more
than a rounding error, and it is why `reckless`'s three magnitude dials all died
(→ [[hexarena-reckless-closed]]).

**How to apply:** three traps this measurement walked into first, all of which
would have produced a confident wrong answer.

- ⚠️ **The axis is NOT battle length.** Cleffa is the *longer* matchup (42 turns
  vs 20). An early sweep that ranked opponents by median turns found no monotone
  relation at all.
- ⚠️ **The axis is NOT how long the gate stays open.** Timeline-weighted, the gate
  is in force for 369‰ and 323‰ of the two fights — within a quarter of each other
  while the value differs six-fold. Damage-weighted it is 409‰ vs 325‰, and that
  statistic is nearly *determined* (a holder that spends its whole bar takes ~30%
  of it above a 700 gate by construction), so it discriminates almost nothing.
- ✅ **The axis is WHERE ALONG THE BAR the fight is decided.** Measure it by
  fighting the same tier gated at the *other* end (`below_health: 700`), so the
  two arms partition the bar: Cleffa +222 above / +228 below, Gastly +38 / +122.

**Two pricing rungs, both the last one before a cliff.** Gate swept
200…900: the share of the payload delivered is 100/90/90/89/89/**4**/3 (Cleffa)
against 75/67/44/29/17/10/6 (Gastly) — below 700 the arms converge into a plain
discount on a stat, at 800 the whole feature dies. Ungated tier swept 5/10/15%:
better than `endurance` against 2/10/12 of 21 characters, worse against 11/4/2 —
at 5% nobody takes it, at 15% it *is* `endurance` plus a bonus. **A two-tier
trait's floor must sit UNDER what the trait it competes with grants outright**,
because a placement brings one trait and a strict superset retires the other.

**Do not price a two-tier trait by adding its faces.** 340 bare → 373 → 452,
where 100‰+250‰ asks for 459: the gated tier buys 79 on top of the ungated one
against the 81 it buys alone. → [[hexarena-reckless-closed]] for the same
saturation at four times the size.

Related: [[hexarena-a-trait-that-changes-a-price-has-two-sites]],
[[a-null-needs-two-controls]], [[a-term-with-no-subject-is-invisible-to-every-walk]],
[[hexarena-speed-and-measurement]]
