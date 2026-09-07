---
name: hexarena-a-greedy-rating-cannot-hold-two-self-casts
description: hexarena — Suggest prices one turn at a time, so a kit holding two turn-spending self-casts loses the one with the smaller immediate number forever; burrow beside split cost the Diglett build 61 points and cast the split zero times
metadata:
  type: feedback
---

`burrow` was put into both Diglett builds and measured. In the **whole** build it
is worth **+21.5** points in the matchup that was a script. In the **split** build
it is worth **−61.5** — 11.0% against Machop where the shipped kit reads 72.5%.

The slot-empty control is what makes that legible: the same split build with the
fourth slot simply **empty** reads **78.0%**, better than shipped. So burrow is not
costing a slot, it is costing the *mechanism* — over a whole duel the rating cast
the split **not once**.

**Nothing is mis-priced.** `hidden` is bounded by what the enemy would actually
have thrown; `summonWorth` by the body it buys. `Suggest` is a **greedy one-turn
evaluator**: it prices what a turn does now and has no term for "this compounds". A
summon and a hide are both turns that deal no damage, so whichever has the larger
*immediate* number takes the slot every time both are off cooldown, forever.

**How to apply.**

- **One turn-spending self-cast per kit** until the rating can see a second turn.
  Two of them is a decision `Suggest` cannot make, and the loser is not the weaker
  skill — it is the one whose payoff arrives later.
- When a skill measures badly in a build, add the **slot-empty control** before
  concluding anything. It separates "this skill is bad here" from "this skill is
  displacing something", and those need opposite fixes. Here it was the difference
  between blaming the skill and finding that the build never used its own engine.
- The same skill's worth is a property of the **kit**, not of the skill. Quoting one
  number for it is quoting a matchup.

⚠️ Chasing this, `hidden` was given a clamp that sounded obviously right — you can
only avoid dying once — and it moved **one battle in four hundred** across 3,200. It
was reverted. An argument that sounds right is not a measurement, and an unmeasured
change is one nothing holds. See [[hexarena-a-mutation-baseline-must-come-from-git]]
for the harness rule this session broke while measuring it: the mutation was applied
to the same file that held the real edit, and `git checkout --` took both back.
