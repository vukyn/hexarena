---
name: the-fixture-that-makes-it-measurable-has-its-own-price
description: hexarena DAT-007 — adding the board that finally exercised the area axis cost that board ~250‰ of defence, twenty times what the axis returned (0‰→2‰); the repo had ALREADY priced that exact board change in docs/balance.md, under a different feature
metadata:
  type: feedback
---

Before building the fixture that makes a dead mechanism observable, **price the
fixture itself, and grep the repo for somebody who already did.**

`DAT-007` said the area axis was under-priced. The diagnosis was that on all five
shipped squad formations a `column` catches exactly one occupied cell, so
`splash_power` is never reached and no fought board exercises the axis at all. The
plan's fix was the honest one: add `s06`, a sixth squad that stacks — `s04`'s three
members at `roster.json`'s adjacent-pair formation, so the placement is the only
variable and `s04` vs `s06` is a pure control.

**It worked as a fixture and failed as an experiment.** On `s06` a `column` really
does catch 2 and an `arc_up` 3, and the area carrier's silent `dazzle` went 0 → 13
casts. The rate moved **0‰ → 2‰**, against a written-down gate of 150‰, while the
non-area control moved **−7‰** — *further than the subject*. And `s04` vs `s06`
read **752‰**: the identical three characters lost ~253‰ from three slot numbers,
because a stacked formation puts two units in the first occupied rank and drops the
ace from reach-depth 3 to 2.

⚠️ **That price was already in the repository, under a feature with a different
name.** `docs/balance.md`, in the composition-bonus section: *"a squad re-slotted
into one column reads 464‰ against `s01` where it reads 677‰ spread out — stacking
costs about 213‰."* Same trade, same order of magnitude, measured months earlier by
somebody pricing a bonus rather than a shape. Reading it first would not have
changed the decision to measure, but it would have set the expectation correctly
and it belonged in the plan.

**Why:** a fixture that changes the board changes *everything the board decides*.
Here reach is counted in occupied ranks, so a formation is simultaneously the
answer to "can this shape catch two" and the answer to "how many ranks shield the
ace" — one edit, two axes, and the second one is bigger. A gate on the subject
alone would have called `s06` a wash; the gate that caught it was the **control
row** (a squad with no area skill fighting the same new board) and the **isolation
row** (the new formation against the old one, both directions).

**How to apply.**

- When a plan says "add the placement/fixture that finally exercises X", ask what
  *else* that placement decides, and add a row that measures it on its own. Here:
  `s04` vs `s06` and `s06` vs `s04`, identical casts, three slots different.
- Grep the docs for the fixture change, not for the feature. "stack", "re-slot",
  "one column" found the prior 213‰ reading; "area", "splash_power" did not.
- Set the floor before the run and let it fail. The failure is publishable: the
  entry now says *why* the repricing cannot be taken yet, which is worth more than
  the repricing would have been.
- A failed gate still ships its record. What landed was zero data change, one
  derived-by-property guard test, a rewritten `TODO.md` premise, a corrected memory
  note and a new item — and **no golden moved**, which is exactly what the "refuse
  and measure" course was chosen for.

Related: [[measure_which_guard_masks]], [[a_well_formed_measurement_can_measure_nothing]],
[[a_refusal_can_be_right_for_the_wrong_reason]], [[fixture_decides_what_is_visible]].
