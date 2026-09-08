---
name: a-hand-built-arm-can-miss-the-reporting-branch
description: A hand-built exerciser that calls the helper directly proves the DERIVATION and leaves the test's own reporting branches unexercised — only a real-data mutation reached the other log line
metadata:
  type: feedback
---

A hand-built control arm added to cover "the branch no shipped data can reach"
covers the branch **in the helper it calls**, not the branches in the test that
reports the helper's answer. Those are two different sets of lines, and the arm
never touches the second set.

**Why:** DAT-010 shipped
`TestEveryBonusIsMeasuredAgainstEveryShippedFormation` (a log-only walk) beside
`TestTheBonusReachWalkSeesAFormationThatReachesARung` (four hand-built arms).
The four arms call `bonusesReached` directly, so they proved the reduction over
`composition.Book.Awards` works in both directions — mutating the helper to
return everything reddened arms 2 and 4, mutating it to return nothing reddened
arms 1 and 3. But the walk has **three** `t.Logf` branches (fires on a fought
board / fires only off-board / fires nowhere) and shipped data reaches exactly
one of them: every line came out of `bonusboard_test.go:264`. The hand-built
arms cannot reach the other two, because they never run the walk.

The only thing that did was the optional data mutation: re-slotting `s01` into
one column flipped `same_column` to `bonusboard_test.go:256` — a **different
source line** — which is what proved the reporting half works at all. Reading
`-v` and noticing the line number moved is the check; a line-number-blind
comparison of the two tables would have said "one row changed" and missed that a
whole branch had just been executed for the first time.

**How to apply:** when a log-only walk carries branches, count which source
lines the `-v` output actually names. If every logged line comes from one
`file:line`, the other branches are unexercised whatever the hand-built
exerciser next door does — and the mutation that exercises them is one on the
**real data the walk reads**, not one on the helper. Run it even when it is
marked optional; it is the only arm that looks at the reporter.

Related: [[feedback_a_null_needs_two_controls]],
[[feedback_a_green_expected_mutation_proves_nothing_by_itself]],
[[feedback_a_mutation_must_hit_the_arm_you_claim]].
