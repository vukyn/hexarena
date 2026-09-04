---
name: hexarena-tempo-and-stalemate
description: "hexarena PR#121 — frozen() narrowed (only DoT holds a board open; a self-aim is not an action; a summon is); tempo priced from the STAT not the queue; a turn = turnWorth (mean) not bestStrike"
metadata: 
  node_type: memory
  type: project
  modified: 2026-08-28T10:47:12.710Z
---

hexarena PR #121 (merged 2026-08-28, `abea6d9`) closed two open items in one branch, one commit each.

## The stalemate hole, closed

`frozen()` asked two questions that were each slightly wrong and covered for each other:
- it refused a deadlock while **anything** timed was on the board — true of a poison (kills → empties a side), **false** of regen/buff/shield;
- it counted **any** legal aim as "still has something to do", **including a self-aim**, which every support-holding unit always has.

⇒ a unit tending its own regen held a frozen board open **for ever**: no draw declared, battle ran the 4000-turn limit, and **the log says nothing about what happened**. (The slot `1,2` draft, 5/4000 seeds.)

Now: `status.Set.TimedIn([]Category)` — **only `status.Dot`** keeps a board open (`var outcomeChanging`); and `canAimAtAnEnemy` replaces `canAimAtAnyone` in `frozen` (kept for `New`'s placement check, which asks a different question). ⚠️ **A summon counts as reach wherever it is aimed** — it puts a unit on the board that may reach what its summoner cannot; leaving it out made a caster holding only a summon read as deadlocked when its escort fell (`TestARosterUnitsSlotIsNotFreeWhenItFalls` caught it).

⚠️ **A taunt does NOT belong in `outcomeChanging`, and only a mutation established that**: `aims` offers a taunted unit its taunter **whether or not the taunter is in reach**, so a taunted unit always has an aim and is never frozen. The category was a claim no test could reach (mutation removing it → NOTHING failed) → deleted. The board it matters on (reachable enemy + taunting enemy out of reach) is still tested, and that test pins the **`aims`** behaviour from outside: the day an unreachable taunter stops being offered, that board becomes a wrongly declared draw.

## Tempo priced — from the stat, never the queue

`wait = atb.Scale / speed`, so a share added to the speed stat is that share added to the unit's turns: `H × (now − was) / was` extra turns over horizon H. **No queue read**, so purity holds. What stays out of reach: *where* the extra turn falls in the order (that would need the queue).

⚠️ **A turn is worth `turnWorth` — the MEAN over what a unit could point at somebody — not `bestStrike`.** Every other term in `price.go` uses the best attack; tempo must not. Charged at `bestStrike`, `outrage`'s recoil made the dragon build **avoid its own heaviest skill** and its duel rate fell **26.6% → 20.0%** — a rating playing worse while believing it improved. An extra turn is not another cast of the skill that is on cooldown 2/3 of the time. Same figure both directions (a haste and a slow cost the same).

**Consequence that is an answer, not an accident:** a buff is worth `horizon × share` of a turn → a 30% haste over 3 turns = 0.9 turn ⇒ **can never beat the unit's best attack**; it wins exactly while that attack is recharging (that is the test).

**Measured:** roster ally 49.1% → **49.4%** at 20k seeds (inside σ≈0.35 → **no second re-level needed**), 0 stalls. Dragon-vs-fire duel 26.6% → **22.1%** (second cast finding in that number: no detonate in the line, plus a heavy skill that really does charge a price). Squirtle 676/39 and Bulbasaur 23 turns/2818 healed unchanged. Only `replay.golden` moved.

⚠️ Process note that paid off twice here: **measure the build after pricing a cost.** A correctly-priced cost cannot make the AI perform worse; when it does, the price is too big — that is how the `bestStrike`→`turnWorth` correction was found.

Related: [[hexarena-deeper-opponent]], [[hexarena-draw-and-sidenone]], [[hexarena-taunt]], [[hexarena-core-design]]
