---
name: hexarena-composition-bonuses
description: "hexarena — squad composition bonus (thresholds on a shared element/origin); 7 decisions settled 2026-09-04, NOTHING built"
metadata: 
  node_type: memory
  type: project
  modified: 2026-09-04T11:29:22.229Z
---

Squad **composition bonuses**: fielding several units that share something grants a bonus at a threshold (rung). User's idea, filed in `TODO.md` across PRs #289/#290/#291. **Nothing is built** — the entry is idea + measurements + settled decisions. See [[hexarena-shipping-a-character]], [[hexarena-roster-cannot-price-damage]], [[hexarena-tui-references]].

**Why:** the design was settled *before* code, and four of the decisions were made against measurements off the shipped data rather than taste. Rebuilding that reasoning from the JSON costs a full session.

**How to apply — the 7 settled answers (2026-09-04):**

1. **Count taken ONCE on entering the battle.** Not recounted, summons don't count. → it's a drafting decision, never a tactic; needs no `tickStatuses` hook, no death recount, and **no map walk** (roster is still a slice before turn 1).
2. **Dual affinity counts toward BOTH halves** (Lapras water/ice, Magnemite electric/metal). ⚠️ First thing in the game that pays a dual for being one — CLAUDE.md measures a dual's *defensive* half as ≈nothing.
3. **Two kinds ship**: whole-squad, and sharers-only. A bonus declares which.
4. **Bonuses STACK** — several of each kind live at once is normal.
5. **One bonus at a time, each must do something no other does.** A new bonus's PR states what no shipped bonus already does; "nothing" → doesn't ship.
6. **Rungs 2 and 3 only.** 4/5 authored later *with* 5v5 — not declared-and-quiet. A rung that can't fire isn't declared.
7. **Its own JSON file** (a rule about squads, not an axis of how one unit fights).

**⚠️ Reachability measured — 2 of 4 obvious axes are dead:**

| axis | max sharing one value | rungs |
|---|---:|---|
| element | 3 (water) | 2, 3 |
| origin | 14 (`pokemon`) | free at every rung |
| species | 2 (`plant`, `mythic`) | rung 2, twice |
| archetype | 1 each | **none** |
| archetype `column` | 6/5/4 | 2,3,4,5 |

⚠️ **ORIGIN IS UNUSABLE TODAY AND STACKING MAKES IT WORSE.** No 3v3 squad can *fail* the origin axis: **17/18** are `pokemon` and the 18th is one character, so worst case = Naruto + 2 Pokemon = still rung 2. ⚠️ Figure derived, not remembered — it read 14/15 and then 16/17 on 2026-09-05 alone, and every character that ships makes this objection **stronger**: `jq '[.characters[]|select(.id|startswith("pokemon."))]|length' internal/seed/data/cast.json`. `Hidden` doesn't save it, and saves it **less** than this used to say: this called the flag "an authoring convenience by its own doc" and ⚠️ that half of the doc is gone — `internal/draft.NewPool` gates the ban-and-pick pool on it, so it is a rule of the game for a drafted squad. Either way naruto is both the only hidden character **and** the only non-`pokemon` one, so honouring the flag removes the one character a squad could have failed the axis with and makes an origin bonus **more** unconditional. **Element first; hold origin until a 2nd origin can field 2–3.**

⚠️ **`column` was the best-shaped axis** (every rung reachable, means formation not tribe) — user decided **no column bonus**, which is what killed the correlation objection (water bundles a free column rung, grass doesn't: `blighter` col 1 vs `sapper` col 2).

**⚠️ Pricing — the prerequisite, not a later step.** Nothing is measurable until `forge.FightSquads` can disable **a set of bonuses** (not a boolean — a global switch measures *the system*, never one rung). Control = same squad, same members, same seeds, ONE bonus toggled, others left on; mirror control must read 500‰ exactly. Both obvious controls are already known wrong: swapping a member measures *the member*, and the same bonus on both squads **cancels** (Oddish pairing read −29‰).

**Other gotchas written into the entry:**
- `Squad.Validate` refuses a repeated unit **id** and **slot**, says nothing about the same **character** twice (settled "it MAY") → a rung-2 is one character fielded twice. #268 measured 3 copies as the *weakest* squad (~11%) — but that was measured with **no** composition bonus in play.
- Per-character `EffectiveHP ≤ 11500` stops bounding a **squad-wide** grant. Ceilings **saturate**, so a grant near a ceiling buys less than the number says.
- A bonus as a permanent **status** gets log/drawing/describers free — and makes `dispel` a question nobody has asked. Baked into `Take`/`New` = invisible, undispellable, unexplainable.
- PvP is cheap: the squad crosses the wire whole in `wire.Hello`, both ends derive it, a disagreement shows as a **per-turn digest mismatch**.
- ⚠️ A new data file = a **16th name in 3 independent places**: the `//go:embed` line in `internal/seed/seed.go`, the `dataFiles` slice in `internal/seed/digest.go`, and one `XxxFile()` accessor. **Missing `dataFiles` is silent** — the file loads and the data digest stops covering it, so two peers on different bonus data pass the gate then diverge.

**Sub-item — a reference screen on the menu** (user asked for it): read-only catalogue like statuses/elements/traits/species, both clients. ⚠️ It is the **10th** `menuItems` entry (9 today). ⚠️ Must go in the sweep — `model.go` records **five** screens that slipped it and silently lost width/translation/leak tests (`screenCount` + `TestEveryScreenThisClientDrawsIsSwept`). ⚠️ Data screen → `UsableWidth()`, but its **footer** is measured against the 120-col floor and the floor is a footer's only lever. Must draw the two *kinds* apart and show that several fire at once, and not look broken when 2 rungs become 4. **Cannot be built before the first bonus exists.**
