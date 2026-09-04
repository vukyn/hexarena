---
name: hexarena-allsided-and-scarcity
description: "hexarena PR#127 — all-sided skills priced by BOTH halves (friendlyFire, caster included, splash kills count); \"holding a skill\" = cooldown TIE-BREAK not discount; a symmetric AI change must be measured HEAD TO HEAD, never by the roster rate"
metadata: 
  node_type: memory
  type: project
  modified: 2026-08-28T11:43:19.148Z
---

hexarena PR #127 (merged 2026-08-28, `b616c12`) closed the last two pieces of *A deeper opponent*, one commit each. What is left after it: **waiting** (needs a lookahead) and *where* in the order an extra turn falls (needs the queue).

## All-sided skills (`skill.All`)

The old refusal in `rate` was **sound, not an oversight**: `expected` *skips* a unit on the caster's own side rather than subtracting it, so the guard and that fact were two halves of one decision — lifting the guard alone = the AI that bombs its own squad and scores it as a gain. Answered with `friendlyFire` (= `expected`+`finished` pointed at the caster's own side), kept a separate function because every other question those two are asked means "what could this unit do to *somebody else*".

- ⚠️ **The caster is not skipped** — a shape can cover the cell it is cast from, and `resolveAgainst` never asks whose side a target is on.
- ⚠️ **A kill on a SPLASH cell is priced here**, unlike `finished` (primary cell only). Not inconsistent: `finished` asks `expected` "if aimed at that cell", which over-states an edge; this loop already holds the *reduced* share.
- Status loop: skip **relaxed, not replaced** — the friendly/hostile branches were already right; the guard just kept one from running. ⚠️ Only a mutation found it: every damage-priced test passed with the enemy half of the *status* loop skipped, because `expected` does that half. Needed a **single-cell** all-sided fixture (`fume`) so an aim can be pure gain or pure cost.
- An all-sided attack now counts as an attack in **4** places via `aimedAtAnEnemy`: `bestStrike`, `turnWorth`, `bestAgainst`, `worstStrikes`. Before: a unit whose only attack was all-sided read as threatening **nobody** → heals/shields against it priced 0.
- Nothing shipped is all-sided ⇒ **no golden, no balance figure moved.**

## "Holding a skill for a later turn" = a cooldown TIE-BREAK

Waiting is out of reach (needs next turn's value). What is reachable is **waste**: damage clamps at the target's HP, so vs a sliver the nuke and the filler are worth **exactly** the same and the tie went to **kit order** — the nuke burnt on 10 HP for 3 turns of cooldown. `take` in `Suggest` now compares value, then `declared.Cooldown` only on a tie. ⚠️ A **discount** would price scarcity by guessing at turns = the mistake tempo was corrected for. ⚠️ The **summon branch** is the one place the comparison could be written twice — a mutant passing it cooldown 0 survived until a summon-tie test existed.

## ⚠️ The measurement rule this established

**A symmetric AI change cannot be judged by the roster win rate.** Both sides run `Suggest`, so an improvement to both leaves the rate alone; what the rate shows is *whose kit had more to gain*. Measure **head to head** — the change on ONE side at a time (scratch package var toggled per acting side), 20 000 seeds each way:

| | baseline | with it | Δ |
|---|---:|---:|---:|
| ally | 49.4% | 49.4% | 0.0 |
| enemy | 50.6% | 51.5% | **+0.9** |
| both (shipped figure) | 49.4% | **48.5%** | — |

Never worse either side, better on one — the same bar as [[hexarena-tempo-and-stalemate]]. Asymmetry is a **cast fact**: ally ace kit is cd 2/2/2/3 (no spread); enemy has `hydro_pump` cd4 beside `water_gun` cd1 — the nuke/filler pair. Roster left at 1.5 off even, **not re-levelled**: levels are coarse (Ivysaur 16→30 = tens of points). `replay.golden` moved by **one choice** on seed 11. Build duels unmoved (dragon 22.1%, Squirtle 676/39, Bulbasaur 23/2818).

⚠️ Other sessions work this repo in parallel — this branch had to rebase over #124/#125/#126 (dragon detonate) and re-confirm the 49.4% baseline on the rebased tree. Doc conflicts in README/TODO were "both sides added at the same anchor": keep both, main's first.

Related: [[hexarena-deeper-opponent]], [[hexarena-tempo-and-stalemate]], [[hexarena-core-design]]
