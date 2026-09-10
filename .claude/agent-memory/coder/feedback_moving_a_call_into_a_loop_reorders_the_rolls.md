---
name: moving-a-call-into-a-loop-reorders-the-rolls
description: Moving b.answer from after the cell walk into the strike loop also moved it ahead of the skill's riders and its drain, so even a ONE-strike cast draws the RNG in a different order — "the single case is unchanged" held only for a fixture with no riders and no drain
metadata:
  type: feedback
---

A call moved **into** a loop does not only run more often. It also runs at a
different point relative to everything that used to sit between the loop and
its old site — and in an engine where every status roll pulls from one
`*rng.Source`, a different point is a different number.

**Measured (hexarena, 2026-09-10, the per-strike reply).** `b.answer` moved from
after the cell walk in `Act` into the strike loop of `resolveAgainst`. The old
per-target order was `strikes → riders → restore → drain → kill → (next cell) →
answer`; the new one is `strike → reply → … → riders → restore → drain → kill`.
So for a **single-strike** skill, where the count of replies did not change at
all:

- the reply's own status roll now precedes the skill's rider roll, and in the
  shipped seed-11 battle a 40‰ poison that was **resisted** before now **lands**
  — the same roll, a different draw. 245 events → 255, 50 turns → 52, one
  status_applied where a status_resisted used to be.
- the drain now heals the caster **after** it has taken the answer instead of
  before, which is the owner's own note that step 2 alone is more lethal than
  step 2 plus step 3 will be.

**Why it costs a session:** the brief asked for a control saying "the one-hit
case did not move", the control was written with a fixture skill carrying no
riders and no drain, and it passed — twice, including under the revert
mutation. That is a true statement about *what a reply is worth* for one strike
and **not** a statement that a one-strike cast produces the same battle. Both
are worth saying; only one of them is what the test measured.

**How to apply:** when a call moves inside a loop, write down the two orders in
full and ask which *other* calls it crossed. Then say in the report which sense
of "unchanged" the control proves. A control fixture with no riders and no drain
is the right fixture for the amount — pair the claim with the goldens, which are
where the reordering shows. See [[a-green-expected-mutation-proves-nothing-by-itself]],
[[a-golden-screen-shows-only-the-rows-that-fit]].
