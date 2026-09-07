---
name: hexarena-a-rating-that-prices-nothing-stops-playing
description: hexarena — a blow a guard would absorb whole rates nought, so Suggest passes rather than spending the guard; two guarded units stand at full health for 600 turns without acting, and it looks like a long battle rather than a broken one
metadata:
  type: feedback
---

A mirror of two units carrying `carapace` never finishes. Measured to the turn:
over eight turns **neither unit acts once**, and after six hundred turns both
stand at **3600/3600 with 576 of pool left**. The pool is never spent because no
blow is ever thrown at it.

The chain is short. `pastAPool` returns **0** when the pool covers the blow →
`expected` is nought → every option rates nought → the pass rule reads "nothing
worth doing" → both sides pass, for ever.

**Why it hid for a release.** A rating that mis-prices an option picks the wrong
skill, and somebody eventually notices the skill. A rating where *every* option
reads nought **stops playing**, and two sides standing still produce a long battle
rather than an obviously broken one — the turn limit catches it and reports
nothing about what happened. The recorded diagnosis blamed a "wall-heavy roster"
and a pass rule; both were wrong, and a plain **1v1** reproduces it.

**How to apply.**

- When a board does not resolve, ask **what the rating chose**, not what the units
  are. One probe printing the chosen skill per turn answered in a single run what
  three sessions of win-rate readings had not.
- A defensive term in `price.go` prices what a guard is worth **to its holder**
  (`shielded`, `guarded`). There is no term anywhere for what **destroying** one is
  worth to the attacker, and that asymmetry is the defect: progress that is not
  health lost is invisible to the whole file. Compare
  [[hexarena-a-greedy-rating-cannot-hold-two-self-casts]] — same blind spot, one
  step less severe.
- **A guard board must be asymmetric to resolve.** Only one side may carry the
  guard. That is also the right shape for measuring what a guard is worth, so the
  board that works and the board that answers the question are the same board.

⚠️ The fix — crediting `whole - landed` at a share — fixes the pool mirror at any
share from 10% up, identically at 10/25/50/100%. **A step, not a curve**, which is
the signature of an option that was reading exactly nought: it only has to become
positive. A sweep like that cannot choose the share, so the number has to come from
an argument about what destroying a guard is worth.
