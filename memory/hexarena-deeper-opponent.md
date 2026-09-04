---
name: hexarena-deeper-opponent
description: "hexarena PR#117 — Suggest prices non-damage in damage; the 3 clamps ARE the design; permanent status has duration 0; expected() re-reads the board so a hypothetical dies there; roster 53.1%→79.0% needs a data retune"
metadata: 
  node_type: memory
  type: project
  modified: 2026-08-28T09:44:53.898Z
---

hexarena PR #117 (merged 2026-08-28, `86d148e`) closed the README roadmap item **A deeper opponent**: `battle.Suggest` now plays buffs, guards, heals, cleanses, statuses and kills, all priced **in damage** from the function that resolves each, over capped horizons. New `internal/core/battle/price.go`; `status.Set.With/Without/Pending`; `Battle.against`. Still one turn deep, no randomness, no mutation.

**Horizons** (`price.go`): `buffHorizon 3`, `guardHorizon 2`, `healHorizon 2`, `killHorizon 1`. Under-pricing costs a marginal cast; over-pricing costs a kill.

⚠️ **The detonate setup needs NO term.** Price the status and `poison_powder`/`fire_spin` become castable; the payoff is already right because `conditionTarget` is the one builder `Suggest` and `resolveAgainst` share. A "this unlocks that" term double-counts and needs an unbounded horizon.

⚠️ **THE THREE CLAMPS ARE THE DESIGN, not safety nets** — all three found by the rating misbehaving, none by reasoning:
1. **DoT clamped at target HP** (as `expected` clamps a strike). Unclamped it is the biggest number in the file → AI re-poisons a corpse-in-waiting → **19 points** of roster win rate.
2. **Heal clamped at `threat(target) × healHorizon`.** Without it a heal beats a kill *by construction*: damage is clamped at remaining HP (killing a 40-HP unit rates 40) while a full bar of room is not. Also gives "ally at full HP = 0" and "ally nothing can reach = 0" free.
3. **A kill is worth HP + `strike(victim) × killHorizon`** — everything defensive is paid over a horizon, so damage needed one too.

⚠️ **A permanent status carries duration 0**, so `min(kind.Duration, horizon)` prices EVERY permanent buff/debuff at nothing (the test's defence buff rated 0 while its own arithmetic said 72). Go through `turnsOf`. Same case `summonWorth` has for a summon that never leaves.

⚠️ **`expected` reads `b.occupant(cell)`**, so handing it a hypothetical unit gets the REAL one back. First buff term did that → **the whole defensive half of the pricing was dead**, and every test that only asserted *which skill* was chosen still passed. `Battle.against(actor, stats, declared, target, position, spent)` takes the unit; `expected` now goes through it.

⚠️ **`status.Set` shares slices and `Apply` writes through them.** `copied := *unit` + `Apply` refreshes the REAL unit's stack durations from inside `Suggest`. `Set.With` deep-copies (entries **and** each entry's stacks) and layers through `Apply` so the cap/refresh are the resolving ones. **No golden can see this** — a refreshed poison is identical to sustained pressure. `TestASetWithAnApplicationLeavesTheOriginalAlone` is the only guard.

Also: `Set.Pending()` sums `TickAmount × Remaining` **per stack** (`TickAmount(id)` totals stacks while `Remaining(id)` returns the longest — multiplying them over-counts).

⚠️ **BALANCE: shipped roster 53.1% → 79.0% ally** (4000 seeds, 0 stalls, mean 44→47 turns). Squads exchanged reads 82.5% for the same squad → **the rating is side-neutral and the swing is a CAST finding**: the roster's calibration rested on the opponent not playing statuses (ally owns the only applier→detonate pair `sludge_bomb`→`venoshock`; the enemy's fire unit throws burn at a water squad). **Follow-up DONE in PR #119 (`13d0e75`, data-only): Charmeleon 28→30, Ivysaur 16→30 → ally 49.1% at 20k seeds** (was 80.0% at 20k). ⚠️ **The ace level is not a dial** — Venusaur 60→50 alone drops ally 79.0%→**4.0%**, 60→45→0.4%; the young units are the resolution. Neighbours measured: 31/28 52.5% · 29/31 48.2% · 31/31 46.5% · 31/24 62.2%. Loadouts deliberately unchanged (Ivysaur@30 could take synthesis/venoshock/a trait and takes none) so the level was the only thing measured — and it buys a 3rd trait state: a unit that EARNED traits and declined. New standing rule in both docs: **levels are calibrated against how well the AI plays, so an AI change invalidates every rate; quote the seed count beside any figure.**

Build figures moved: Squirtle tank 517→**676** turns, semi 30→39; Bulbasaur parasite 17→**23** turns, healed 964→**2818**. Dragon-vs-fire 42.5%→**26.6%** (fire has a detonate, dragon has none) → `dragon_test` band widened to 150..850 with the reason written in; closing it is data.

**Method worth reusing:** gate the new terms off with a temporary const → `replay.golden` matched byte-for-byte and the sweep read the documented 53.1%, which proved the seam was a pure no-op before any term was judged. Mutation matrix as a throwaway python driver (patch → `go build` → unit tests → golden-alone) reported *which* test caught each mutant and whether a golden alone would have; 3 of 5 mutants moved **no** golden. One survivor reported not papered over: removing the buff cap shortcut is behaviour-equivalent (the cap lives in `Apply`).

Only `replay.golden` moved; `scenarios`/`opening`/`describe` unchanged — that is the check that nothing leaked into shared arithmetic.

**Still deliberately out**: the turn queue (so a speed buff, and `outrage`'s speed-only recoil, are worth nothing to the rating — `mire` is invisible), holding a skill for a later turn, all-sided skills (`expected` skips an ally rather than subtracting it), any lookahead.

Related: [[hexarena-builds-catalogue]], [[hexarena-summon-share]], [[hexarena-core-design]], [[hexarena-mechanics-log]]
