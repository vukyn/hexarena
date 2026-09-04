---
name: measure-which-guard-masks
description: When a defect is "hidden by a clamp", mutate on disk and measure WHICH guard masks it — and note that saturating arithmetic kills monotonicity-in-the-input downstream of a subtraction
metadata:
  type: feedback
---

When a write-up says a defect is unobservable "because of the clamp", do not accept
the named guard. Mutate the defect back in **on disk**, run the whole suite, and then
instrument the site to say which line actually swallows the difference.

**Why:** In hexarena's `internal/core/battle` the open TODO blamed `against`'s
`landed > target.HP` clamp for hiding a wrapped block-charge product. Measured, the
clamp was innocent and could never have done it: the two arithmetics disagree exactly
when the true product will not fit, which is exactly when the saturating helper pins
at `math.MaxInt64`, so the *correct* answer in every disagreement is nought — and a
clamp only ever pulls a figure DOWN. The real masker was a `damage <= 0` guard one
line below, catching a *second* wrap on the subtraction. Believing the write-up would
have produced a fix aimed at the wrong line.

**How to apply:** Three steps, in order.
1. Mutate on disk, confirm with `git diff` before trusting a green run (see
   [[feedback_scripted_revert_wrong_occurrence]] — this repo has a history of scripts
   that edited nothing and "passed").
2. Count reach, not just red/green. A branch entered ~18.9M times across one suite run
   with 0 wraps and a max input of 1,208 against a wrap threshold of 3.07e18 is a
   *reachability* answer, and reachability is this repo's stated bar for whether a
   board is honest. It also tells you a tuned fixture is the only board available,
   which is the thing to refuse.
3. Instrument both arithmetics side by side at the site and print pre-guard and
   post-guard figures. That is what names the guard.

⚠️ **Saturating arithmetic breaks monotonicity in the input once a subtraction is
downstream of it.** hexarena holds this whole family with *"never smaller for more
power"* (`TestNoFigureFallsAsPowerRises`), and it is **false on the correct code**
past a subtracted wall — measured, 6 of 24 parameter pairs, 10 falls — because the
blow saturates before the subtrahend does, so the difference shrinks while the input
climbs. Before reaching for that property, check it passes on the unmutated code. When
it does not, look for another axis of the same function: here it was the subtrahend's
own axis, *a deeper wall never lets more through*, which a wrap breaks and a
saturation cannot.
