---
name: a-fixture-near-a-cap-reddens-on-the-cap
description: A test whose subject sits near a capacity bound reddens on its own premise guard when the mutation makes the subject BIGGER — picked the first unit holding a permanent effect, got one holding three, and the restore-the-wording mutation overflowed the column instead of failing the claim
metadata:
  type: feedback
---

Testing that a permanent effect draws no duration, the fixture took *the first
unit holding a permanent effect*. On the bench that is a unit holding **three**
of them. The test passed. Then the mutation — restoring the ` (always)` wording —
pushed that unit's effects column from 32 cells to 71 against a 60-cell cap, so
`elided` dropped an entry and the test died on its own premise guard
(*"the column elided something, so this reads only part of it"*) before reaching
the assertion about the permanent entry.

**Why:** a red test is not proof the guard works. That mutation would have gone
red with the permanent/timed assertion **deleted**, so the guard it was supposed
to demonstrate was never exercised. Every mutation that *lengthens* output walks
a near-the-cap fixture into the cap first.

**How to apply:** when the thing under test feeds a capped column, a wrapped
line, a row budget or a screen, choose the fixture by **how much headroom it
leaves under the mutation**, not by "the first one that has the property".
Here: *exactly one* permanent status, not *at least one* — two entries fit the
cap whichever way they are drawn, and the mutation then failed on the claim,
naming `phalanx x2 (always)`. Same family as
[[feedback_a_refusal_can_be_right_for_the_wrong_reason]] and
[[feedback_a_mutation_must_hit_the_arm_you_claim]].
