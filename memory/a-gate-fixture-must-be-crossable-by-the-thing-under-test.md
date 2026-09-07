---
name: a-gate-fixture-must-be-crossable-by-the-thing-under-test
description: A threshold fixture picked for the wrong depth passes with the code under test deleted — the gate has to be shallower than the effect being measured.
metadata:
  type: project
---

Testing that `Battle.grant` refills current health before each gate is read, I
reached for the fixture trait that was already there: `dug_in`, gated at
`below_health: 500`, against a health raise of `+200‰`. Without the refill the
holder reads as 2000 of a raised 2400 — five sixths — which is nowhere near half,
so the gate does not fire either way. **The test passed with the refill deleted.**
The mutation harness caught it, not the test.

The rule the fixture has to satisfy is arithmetic, not thematic: a threshold
fixture is only a test if the effect under measurement actually **crosses** the
threshold. Gate at nine tenths against a raise of a fifth and the same board now
discriminates — 83% is under 90%, so the bug fires and the mutation is red.

**Why:** a gate and a raise are two independent numbers in two different fixture
books, and nothing links them. Picking the trait that was already declared felt
like reuse; it was picking a number with no relationship to the one being tested.
The test still *looked* right — same board, same trait, same assertion — which is
the whole trap: nothing about it reads as inconclusive.

**How to apply:** before trusting a test that crosses a threshold, do the
arithmetic out loud — what is the ratio with the bug, what is the threshold, does
one sit on the far side of the other. If the fixture's threshold was chosen for
some other test, assume it is the wrong depth for this one and declare a new one.
And run the mutation before believing the green, which is the only reason this was
found at all. Same family as [[fixture-hidden-branch]] and
[[a-fixture-chosen-by-property-is-blind-to-other-properties]].
