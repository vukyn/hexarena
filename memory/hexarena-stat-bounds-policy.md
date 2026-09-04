---
name: hexarena-stat-bounds-policy
description: "hexarena PR#112 — SETTLED: ceilings+effHP budget bound the AUTHORED line only; saturation bounds the fought line; the stat floor and why speed can never reach 0"
metadata: 
  node_type: memory
  type: project
  modified: 2026-08-28T08:30:45.342Z
---

hexarena PR #112 (merged 2026-08-28, `0638de8`) settled the stat-bound policy and pinned the floor.

## The policy (user's decision, now the code's)

⚠️ **Ceilings + `max_effective_hp` bound the AUTHORED line only** — what a designer writes into a character at the level cap. `progression.Limits.CheckValues` takes **six numbers and nothing else**, so a trait/buff/rune is invisible to it.

⚠️ **Going past them in a battle is the DESIGN, not a leak.** A buff that could not take a stat past its ceiling would be a buff with a cliff in it. `battle.New` rejects a base line of 740 defence and then hands the same unit 786 through a trait, **in the same call** — intended.

⚠️ **What bounds the FOUGHT line is the saturation, not the budget**: `modifier.Set.Stat` rescales against `ceiling × headroom` (headroom 3000), so nothing reaches **3× a ceiling** however much is stacked. **A future rune/inventory system lands in a system that already bounds it — no new mechanism needed.**

Current numbers (`progression.json`): `level_cap 60`, ceilings `hp 4800 / atk 800 / def 800 / spd 200 / acc 300 / ddg 150`, `max_effective_hp 11500`; `defense_constant K = 300` (`combat.json`); `effHP = HP × (K+def)/K`.

⚠️ **Do NOT raise the ceilings** — nothing is near them (worst: Charmander atk 700/800 = 87%). The binding constraint is `max_effective_hp` (Squirtle 215 left), and raising a *ceiling* gives Squirtle nothing because the full thing is the **hp×def product**. Also **the ceiling does two jobs** — raising it silently raises the in-battle saturation limit (×3). Raising 11500 lengthens every fight and revalues every DoT/regen/sustain.

`hexforge check` prints a **second table**: the fought line per (character × trait), via `forge.Library.Held`. Only **unconditional permanent** grants count (`status.Set.Hold` refuses timed; a `While` gate is skipped — `blaze` reads a health no character has outside a battle). `TestNoTraitCarriesACharacterFarPastTheBudget` is a **tripwire, not a bound**: 120%, shipped worst Squirtle/`ballast` 113.4%.

Found the moment the table existed: **Bulbasaur/`endurance` has 157 left**; `reckless` is *under* the bound (`bare` costs more than `unleashed` buys).

## The floor

⚠️ **Speed can never reach 0** — four guards: the floor at `base × floor_fraction/1000` (10%), which `scale.Saturate` **approaches without arriving**; `Stat`'s `if value < 1 { return 1 }`; `atb.Wait`'s clamp; `atb.Queue.Add`/`Reschedule`'s clamp. `TestNoShippedDebuffCanFreezeAUnit` stacks every harmful status 50 deep on every character and asks whether the queue still turns.

⚠️ **`max_stacks` binds LONG before the floor.** `expose` caps at 2 stacks → Squirtle def bottoms at **410 of 640**; the floor is 64, ~6× further down; `expose ×100` still lands 2 stacks. **To strip armour harder the levers are the status amount and `max_stacks`, NOT the floor** — or `pierce`, which is the counter armour was given (`EffectiveHPAgainst(…, scale.Base)` = raw HP). A per-stat floor of 0 would break the deliberate global `floor_fraction` that makes different-magnitude stats saturate at the same rate.

⚠️ **`TestStatNeverDropsBelowOne` was passing for the wrong reason** — it crushed base 3, where saturation lands ≥1 by arithmetic and the branch is **dead** (deleting it left the test green). **Only base 0 reaches the guard** (Saturate gets gap 0 and returns the base) — and that is real: a summon is authored with a fixed stat line and `"dodge": 0` is ordinary. Fixed in this PR.

⚠️ TUI guards that catch new warning strings: **79-cell width**, and **"absorbs" is banned in English** (screen renamed to "effective hp" / "máu quy đổi").

Related: [[hexarena-mechanics-log]], [[hexarena-core-design]], [[hexarena-builds-catalogue]]
