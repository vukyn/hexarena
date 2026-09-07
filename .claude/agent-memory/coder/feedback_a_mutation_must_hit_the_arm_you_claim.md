---
name: a-mutation-must-hit-the-arm-you-claim
description: A red test does not prove the arm you named is the one that measured it — a blunt mutation fired on the first table row and a realistic one walked past a whole-state snapshot; aim the mutation at the claimed arm
metadata:
  type: feedback
---

**A mutation that reddens a test proves only that *something* in it fired.** If
the test's doc names a particular arm as "the half that matters", the mutation
has to be shaped so **that arm alone** goes red. Both halves of this bit in one
session (hexarena `internal/room`, spectator step 3).

**Why:** the arm a test claims to measure is the reason the test was written, and
it is the one a later reader will delete as redundant. A green→red transition on
some *other* row is exactly the evidence that lets the claimed row rot.

**How to apply:**

- **Aim the mutation past the earlier rows.** A table swept `no squad at all` /
  `a legal squad` / `an illegal squad`, and the doc says the illegal one is what
  proves a validator is not being called. The obvious mutation — run the
  validator unconditionally — failed on **row one**, because the empty squad is
  illegal too, and the run never reached the row under test. The mutation that
  measures the claim is the *realistic* bug: validate only what was actually
  brought (`broughtASquad(x) && !fieldable(x)`), which leaves rows one and two
  green and reddens row three by name.
- **A "nothing moved" snapshot can be blind to the mutation it was written
  against.** A before/after comparison of every reading a room exposes — seat on
  turn, waiting, finished, result, battles played, prompts skipped, record
  length, and what a third player is answered — did **not** see a watcher that
  stole a seat, because the room was *full*: `freeSeat()` returns index 0 with
  `false`, so the overwrite lands on an already-taken seat and nothing observable
  changes until the *next* battle tries to field the emptied squad. The snapshot
  catches a different class (a join that records a body, spends a turn or moves
  the prompt) and the seat theft is caught by the whole-match comparison and by
  the "one seat plus N watchers is still waiting" test. Report which mutation
  each test actually caught rather than assuming the pair covers both.
- **Prefer a mutation that compiles and is a plausible edit.** A blunt one
  (delete the branch, change the arity) either fails to compile — proving
  nothing, see [[mutate-the-producer-not-just-the-logic]] — or reddens so much of
  the suite that no single test's claim is isolated.
- **When a mutation cannot be aimed at the method under test, add a decoy with
  the same shape.** An AST/reflect walk over "every exported method" could not be
  mutated through `Registry.Since` without breaking nine call sites (an arity
  change is a compile error, not a measurement), so a throwaway `SinceLeaking`
  with Since's signature plus a `*Room` went in; the walk named it, which is the
  proof the walk covers a method of that shape. The method count in its own log
  line (12 → 13) is the cheap half of the same check.

Related: [[feedback_a_well_formed_measurement_can_measure_nothing]],
[[feedback_fixture_decides_what_is_visible]], [[feedback_measure_which_guard_masks]].
