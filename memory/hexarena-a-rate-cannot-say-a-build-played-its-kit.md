---
name: hexarena-a-rate-cannot-say-a-build-played-its-kit
description: hexarena — a build's win rate reads fine with a dead slot in it, so RAT-006 blamed a skill the rating never cast for a collapse a different skill caused; count the casts with forge.Library.Census, and pick the board carefully because a duel and a mirror are blind in opposite ways
metadata:
  type: feedback
---

⚠️ **This note replaces one that said the opposite**, and the earlier version was
wrong in the way that matters: it recorded "two turn-spending self-casts in one kit
is a decision `Suggest` cannot make — the loser is the skill whose payoff arrives
later". Re-measured, that is not what happened.

**The reading, re-taken.** Diglett against Machop, 200 seeds each way:

| kit | rate | casts |
|---|---|---|
| shipped `diglett.three` | **725‰** | `split` **0** |
| `burrow` for `rock_throw` | **110‰** | `split` **0**, burrow **1258** |
| that slot empty | **780‰** | `split` **0** |
| `split` dropped entirely | **725‰** | W290 L110, the shipped tally to the battle |

`split` is uncast in the **winning** kit too, and removing it changes nothing at
all — so "the rating never cast the split" was never the mechanism. `burrow` is
cast about three times a battle and each of those turns is the collapse.

**The fact.** A rate reads perfectly well with a dead slot in it. Nothing in a win
rate says which of four slots was used, so any conclusion of the form "this build
does not get to do X" has to be taken off the **cast counts**, not off the rate.
`forge.Library.Census(build, squad, seeds)` is that instrument and
`TestEveryShippedBuildPlaysItsOwnKit` is the rule over the catalogue: a build may
not name a skill it never casts.

**How to apply.**

- **Count the casts before naming a cause.** Two kits with the same silent slot and
  a 615-point gap between them is the shape to expect, not the exception.
- ⚠️ **Pick the census board on purpose — it is blind two ways.** A **duel** prices
  a taunt and a hide at nothing *correctly* (nobody had a choice of aim), so
  `squirtle.fortress` casts `taunt` 0 times in 132 duels and 252 in as many squad
  battles. A **mirror** lets a skill be dominated by its own kit-mate: seven shipped
  builds read a dead slot against a copy of themselves and play it against anybody
  else. What works is standing the subject **in** an authored squad in place of one
  member, against that squad intact — the other five units cancel.
- **One board is one matchup**, so the rule walks several and asks only that a slot
  fire somewhere.
- ⚠️ **A gated slot needs board TIME, not more boards.** `wrecking_swing` wants five
  stacks of `heft`, so it reads silent at two seeds and fires at six.
- **The slot-empty control is still the right control** — it separates "this skill
  is bad here" from "this skill is displacing something" — and it is what said the
  fourth slot was worth *less than nothing*, which is what pointed at the price. See
  [[hexarena-a-denied-turn-is-not-a-lost-turn]].
