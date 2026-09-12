---
name: an-ast-walk-is-the-only-guard-against-a-literal
description: No arithmetic test can tell a derived constant from a hardcoded one that agrees with it today — the guard has to read the source, not the value
metadata:
  type: feedback
---

Twice in one session a value was supposed to be **derived** from data rather than
written down, both times with the reason in a ⚠️ comment, and both times nothing
held it:

- the aim list's matchup thresholds, read off `element.Chart` so a retune cannot
  leave stale marks — substituting the shipped `1500`/`667` left the whole package
  green;
- the wide roster's width threshold, computed from its own format string —
  replacing the call with `139` would have passed every arithmetic check.

**Why:** every test of the *value* compares it against the same data it was
derived from, so a literal that agrees today agrees in the test too. `threshold ==
ceiling+1`, `120 < threshold <= 160`, the boundary cases — all of them read
`139 == 139` whichever way the 139 arrived.

Two guards actually work, and they are different:

1. **Vary the input.** Retune the chart through the real parser and assert the
   *output* moves. Catches a literal, and also catches a wrong derivation.
2. **Walk the source.** Assert the declaration is a call and not a
   `*ast.BasicLit`. Catches a literal only, but catches it where varying the
   input is impossible — a format string's own width cannot be "retuned" from
   outside.

Use 1 where the data is a file you can rewrite; use 2 where the input is the code
itself. This repository has the idiom for both — `TestTheMarksFollowARetunedChart`
and `TestTheWideRosterThresholdIsComputedFromItsFormat`.

⚠️ The tell that a guard is needed at all: a comment saying a value is derived
*and why that matters*. A reason written down and not enforced is the shape this
session found three times — see [[a-guard-on-the-wrong-side-of-the-gate]].
