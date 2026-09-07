---
name: hexarena-a-denied-turn-is-not-a-lost-turn
description: hexarena — pricing.hidden charged a hide the enemy's heaviest blow once per turn of the HOLDER's window; both errors are fixed and are worth 4x of a needed 30x, because the enemy keeps the turn and spends it on a self-buff
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

⚠️ **Both together are worth 4× and the hole is 30×.** Probed on the opening board:
`bestAgainst` 2751, `turnWorth` 1293, windowed **1362** — against a best strike of
**352** for the holder itself. With `hidden` returning nought the build reads 780‰,
exactly its slot-empty control; at /64, 775‰; at /4, 175‰; as shipped, 110‰.

⚠️ **So the rest is not a factor, it is the wrong quantity.** Read the log: while
the hider is under, Machop casts `brace` three times and comes out with its defence
permanently up. The turn lost its **aim** and kept its **value**. What should be
priced is the enemy's best turn *with* the holder reachable minus its best turn
*without*, and the second half is one ply of the opponent's own rating — a search,
which this file refuses on purpose. Filed as `RAT-008`.

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
- **`elsewhere` is nought whenever the holder is the only target**, which is every
  duel and every board where a hide looks best. Do not floor it with `turnWorth`:
  that function counts only enemy-aimed skills, and against a hidden holder in a one
  on one the enemy cannot use any of them.
