---
name: hexarena-range-is-rank-depth
description: "hexarena ⚠️ a skill's `range` is DEPTH into the enemy formation counted in OCCUPIED ranks, not hex distance from the caster — so standing at the back costs nothing, a front rank shields what is behind it, and killing the front opens it"
metadata:
  type: reference
---

⚠️⚠️ **A skill's `range` is not a distance.** `battle.reachableRanks` (`internal/core/battle/turn.go:433`) walks the **opposing** side's ranks via `hex.Ranks(side.Opposing())` — their frontline inwards — and spends **one point of range per OCCUPIED rank**. The caster's own cell is never consulted. I reasoned about it as hex distance and got three conclusions wrong in a row; the measurement that settles it in one line:

```
ally.charmander  at 1,1 uses ember        aimed 3,1   ← ember is range 1, hex d=2
enemy.charmander at 4,1 uses outrage      aimed 2,1   ← outrage is range 1, hex d=2
ally.charmander  at 1,1 uses flamethrower aimed 4,1   ← range 2, hex d=3
```

Four consequences, all load-bearing for placement:

1. **Depth costs nothing offensively.** A unit at the back rank strikes exactly as far into the enemy as one at the front. There is no "get closer" and **no movement in this engine at all**, so placement is permanent — but it is permanent *defence*, not permanent reach.
2. **A rank in front shields what is behind it from short range.** Measured over 40 battles of the b01/b02 squads: **493 range-1 casts, every one aimed at the front rank, none at depth 1.**
3. **Longer range goes over the top** — the backdoor. Range 2 aims the second occupied rank, so a range-2 kit kills a backline the range-1 kit cannot touch.
4. **An empty rank is skipped without spending range** (the `len(held) == 0 → continue` sits *before* `spent++`), so a front line falling really does open the rank behind it to a range-1 skill. This is the classic tactics-game shape and the engine implements it exactly. ⚠️ It went **unexercised** in 40 battles because the backline dies to the backdoor first — so it is a code-read fact with no test behind it.

This is what makes [[hexarena-roster-placement]]'s "placement is purely defensive" finding mechanical rather than empirical: depth buys protection from short range and gives up nothing in return.

⚠️ **A tank in front is a real mechanic, not flavour.** The user's model — "kill the tank first, then you can reach the damage behind it, unless a long-range skill backdoors" — is exactly right, and it was mine that was wrong.

⚠️ **Swapping which squad is fielded as "ally" is NOT a side-swap control.** `hex.Place(side, author)` mirrors 180° and every squad authors from its own point of view, so fielding b01-as-ally-vs-b02 and b02-as-ally-vs-b01 is the **same battle relabelled**: the skill-use counts come out byte-identical and only the winner's name changes. `forge.SquadFight`'s `AsAlly`/`AsEnemy` pair is therefore not measuring side bias for a *symmetric* pair of squads. To price anything, vary the squads. Related: [[hexarena-speed-and-measurement]], [[hexarena-allsided-and-scarcity]].

**Squad slot bookkeeping** (`placement.Placement.Slot`, `squads.json`): authored in that side's own 3x3, `col` 0..2 with **`col 2` = that side's frontline** (`hex.AllyFrontCol = 2`, `AllyBackCol = 0`), `row` 0..2, mapped by `hex.Place`. `roster.json` uses `"slot": [col, row]` in the same author coordinates. Two units at the same author slot in opposing squads are **not** adjacent (ally `(1,1)` → board `(1,1)`, enemy `(1,1)` → board `(4,1)`).

Related: [[hexarena-core-design]], [[hexarena-roster-placement]], [[hexarena-shipping-a-character]].
