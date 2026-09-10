---
name: a-bulk-spend-needs-a-wall-deeper-than-the-volley
description: A per-strike report over a resource spent in bulk is only measurable when the pool is DEEPER than the volley spends; at 1 charge, and on the LAST value when charges == strikes, the buggy and correct readings agree
metadata:
  type: feedback
---

When a resource is spent **in bulk before** a loop and then **reported inside**
it, the bug is "every iteration claims the post-loop figure". The fixture that
can see it has to satisfy two things at once, and the obvious ones satisfy
neither:

- **pool > what the volley spends**, or the correct sequence ends at the same
  nought the buggy one prints, and a test that asserts only the last value is
  green on both.
- **more than one consuming iteration**, or there is no sequence to compare.

Measured on `battle` block charges (2026-09-10, `turn.go` `resolveAgainst`):

| wall | connecting strikes | buggy | correct | discriminates? |
|---|---|---|---|---|
| 1 | 1 | `[0]` | `[0]` | no |
| 3 | 1 | `[2]` | `[2]` | no — single strike was never wrong |
| 3 | 3 | `[0 0 0]` | `[2 1 0]` | yes, but the LAST value agrees |
| **3** | **2** | `[1 1]` | `[2 1]` | **yes, every value** |

**Why:** the shipped defect had lived through a full golden suite because the two
golden occurrences were both 2-strike volleys into a 2-charge wall reported as
`0, 0` — the *last* figure was right, and a reader skims the last figure.

**How to apply:** assert the **sequence**, and pin the last element against the
resource's real post-volley state (`Stacks(...)` here) rather than a literal —
that is the claim worth making. And pair it with a control that the **spend
itself** did not move (`left == before - blockedCount`), or a "fix" that also
started removing one per line passes everything. ⚠️ That control, and the
single-strike test, stay **green** under the revert mutation on purpose: they
say what did *not* change. Only 2 of 4 tests may bite — say so rather than
padding them until all four do. See [[a-mutation-must-hit-the-arm-you-claim]],
[[mutate-the-producer-not-just-the-logic]].
