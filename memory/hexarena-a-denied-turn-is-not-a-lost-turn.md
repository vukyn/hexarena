---
name: hexarena-a-denied-turn-is-not-a-lost-turn
description: hexarena — pricing.hidden charged a hide the enemy's heaviest blow once per turn of the HOLDER's window; both errors are fixed, and the rest is REFUSED with a measurement — the win rate's response to the term is a CLIFF at /5-/6, so the half that is expressible moved nought
metadata:
  type: feedback
---

`pricing.hidden` had two errors and neither was the size of the hole.

**Error one: the window was counted in the wrong unit's turns.** A hiding status
counts down on its **holder's** turns, so `turnsOf` answers about the holder — and
what a hide denies is the **enemy's**. Those are equal only at equal speed, and the
shipped Diglett is fast: it took the whole window before Machop moved once and was
charged Machop's blow twice for it. Fixed by `overTheWindow`, which converts by the
speed ratio, per enemy.

**Error two: a denied turn was priced at the heaviest blow in the enemy's kit.**
That is the same correction `turnWorth` is already the written statement of — the
heaviest skill is on cooldown most of the time, so a turn is worth the *mean*. It
cost the `outrage` measurement once and was still being made here. Fixed by capping
`against - elsewhere` at `turnWorth(other)`.

⚠️ **Both together are worth 4×.** Probed on the opening board: `bestAgainst` 2751,
`turnWorth` 1293, windowed **1362** — against a best strike of **352** for the
holder itself.

⚠️ **So the rest is not a factor, it is the wrong quantity.** Read the log: while
the hider is under, Machop casts `brace` three times and comes out with its defence
permanently up. The turn lost its **aim** and kept its **value**. What should be
priced is the enemy's best turn *with* the holder reachable minus its best turn
*without*.

**`RAT-008` is now CLOSED as REFUSED, and the refusal is the fact worth keeping.**
The second half was built (`pricing.kept`: the best of the enemy's *self-aimed*
casts, one ply through the rating's own `rate`, subtracted after the `turnWorth`
cap — the exact complement of the floor below, since `turnWorth` walks only
enemy-aimed skills). It priced **550** against a denial of 1293 — a 43% cut,
exactly what the derivation said — and moved **no win rate in twelve measured
rows**, plus ten `burrow` casts of 1586 in the one row whose opponent has a
self-aimed skill. Reverted, on the precedent of the `hidden` clamp that moved one
battle in four hundred.

⚠️ **"A needed 30×" was itself a two-point reading, and the sweep overturned it.**
Every divisor of the term, split build against Machop: /1 **110‰**, /2 **145‰**, /3
**105‰**, /4 **175‰**, /5 **555‰**, /6 **775‰**, and then 775‰ at /8, /16, /32 and
/64 alike. The hole is **five or six times**, not thirty — /64 was a reading of the
*plateau*. And below the cliff the rate is **non-monotone**. Consequence, and it is
the transferable one: **a partial correction to a term whose response is a cliff
cannot be measured by a rate at all.** `kept` at 550 leaves the denial at 743 where
the cliff needs ~259, and it cannot get there — `brace` prices through
`selfSpendable`'s gated arm at `strike/5`, so the expressible half is about half the
hole *by construction*. The remainder is **deferral** (a hide moves the blow rather
than removing it), which needs the horizon `RAT-007` refuses.

**How to apply.**

- **Ask what unit a horizon is counted in.** `turnsOf` is always the holder's turns.
  Any term about what an *enemy* does inside that window has to convert.
- **A denial term is a tempo term**, so it takes `turnWorth` and not `bestStrike` —
  the same rule the file already states for `tempo`.
- ⚠️ **A price cut that moves nothing is not evidence it was wrong to make**, but it
  is evidence it was not the cause: these two moved `diglett.whole` from 319 to 317
  wins in 440 and its census casts from 9 to 7. Say the size out loud; the earlier
  `hidden` clamp that moved one battle in four hundred was reverted for exactly this
  reason. See [[hexarena-a-rate-cannot-say-a-build-played-its-kit]].
- ⚠️ **Sweep a factor at EVERY rung before quoting how far it has to fall.** Two
  samples name a distance only if the curve between them is monotone, and a win
  rate in a price is not — `/4` and `/64` said thirty, and `/5` and `/6` said five.
  The cheap version of this is one build per divisor and it costs about a second a
  point; the expensive version is a term written, reviewed and reverted against a
  number that was never there.
- ⚠️ **Where the response is a cliff, build the term ONLY if it clears the cliff on
  paper first.** Otherwise the measurement cannot tell a correct half-sized term
  from no term, and both readings are the same nought — which is not the term
  failing, it is the instrument being unable to see it.
- **`elsewhere` is nought whenever the holder is the only target**, which is every
  duel and every board where a hide looks best. Do not floor it with `turnWorth`:
  that function counts only enemy-aimed skills, and against a hidden holder in a one
  on one the enemy cannot use any of them.
