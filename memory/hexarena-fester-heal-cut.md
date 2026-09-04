---
name: hexarena-fester-heal-cut
description: "hexarena PR #190 — `fester`/`heal_cut`: the anti-sustain debuff, its two hook points, reduce-before-cap, the floor that was dead code, the full cost of a NEW status category, and ⚠️ no shipped placement can measure it"
metadata:
  type: project
---

PR **#190** (`e4682ef`, 2026-08-31). `fester` cuts the healing its holder **receives**: 40% a stack, `max_stacks` 2 → 80%, **never fully off** (the engine's standing preference is to saturate rather than clamp; a hard shutdown is a deletion, not a debuff). Carried by `fire_fang` at 500‰; `rapid_spin` strips the category. Vietnamese gloss `lở loét`.

## The mechanic

**Five healing sources, TWO hook points.** `restores` (`synthesis` 900, `withdraw` 500) · a skill's `drains` (`leech_seed`) · the `regen` status (`regrowth`) · a trait's `drains` (`blood_thirst`, `last_gasp`) · `comeback`'s `at_empty` — but only `(*Battle).heal` and `(*Battle).drain` **raise health**, so `healingFor(unit, amount) (landed, reduced)` is called from both and the arithmetic is written once. `drain` is separate from `heal` only because its event carries `Drained`.

⚠️ **Reduce BEFORE the cap to the room left, never after.** Both callers cap afterwards; capping first hands the cut a number that was going to be thrown away anyway, so the debuff would be invisible **exactly where a sustain build lives** (near full health). Cap-then-reduce reddens precisely one test.

⚠️ **The floor was DEAD CODE as first written.** Both callers guarded `amount <= 0` *before* the cut, so deleting the clamp inside `healingFor` changed nothing observable — **two floors for one invariant, and a mutation deletes either for free.** Same shape as the reply drain's `damage > 0`. Fixed by making the post-cut guard `== 0` so the floor in `healingFor` is the only one. Look for this shape whenever a guard sits on both sides of a computation.

⚠️ **The log must say the cut happened**, or `heals 244` cannot be told from a 900 that was reduced. `Event.Reduced` carries the share; **not** `Event.Refused`, which already means a signed share of a *chance*.

## The full cost of a NEW status category — pay all of it

`heal_cut` is its own category and not a term-less `stat_debuff`, because `describeStatusEffect` is a switch on the category **plus** a loop over `Modifiers`, `StatDebuff` has **no arm in the switch** (a term-less one describes itself as nothing), and the reference prints the category as a **predicate** — it would read *"lowers a stat"* on screen for something that lowers no stat. That is [[hexarena-tui-i18n]]'s PR #161 bug class.

The checklist, all of it load-bearing:
- ⚠️ **Append LAST in the iota.** `CategoryCount` and every declaration-order table (the grouped reference's print order) move when one is slotted in; the enum's own doc says appending cannot reinterpret a saved book or log.
- `categoryNames`, and the stale "seven categories" counts scattered in comments.
- ⚠️ `Harmful()` — **must** include it (what a cleanse may strip).
- ⚠️ `OutlastsAShield()` — must **not** (it is not contamination; see [[hexarena-block-cancels-the-rider]]).
- ⚠️ **BOTH** wording families: `Lang.StatusCategory` (predicate, for the column) **and** `Lang.StatusCategoryNoun` (noun, for `strips %d stack of %s`), both languages. English noun uncountable and article-free.
- A `case` in `describeStatusEffect`; `i18n/forge.go`'s two category lookup maps; `statusGloss`.
- `ParseBook`: require the category's number and refuse it elsewhere, mirroring `tick_power`.
- The `skills.golden` report needs a **new column**, or the status reads as a row of noughts — "a column missing from the golden report is a restriction the design record cannot show".

## ⚠️ No shipped placement can measure it

**6260 festers over 20,000 shipped battles and ZERO heals cut.** The only unit that ever heals is `foe.ivysaur` (965 heals, all `leech_seed` drain), and fester only ever lands on the ally side — **0 of 965 heals came from a unit that had ever held one.** Ally 497‰ → 499‰ inside a ±7‰ band. Third recorded instance of *a mechanism no shipped placement fields is a mechanism nothing measures.*

⚠️ **A saturated matchup prices nothing.** The fire side reads **19‰ at level 60** and **1000‰ at 32**; the measurement lives at **44** (494‰). There it takes the sustain side **500‰ → 468‰** against a rider-off control holding at exactly 500‰, band ±13‰; a second kit moved 401 → 366‰. My own naive attempt used a pair already at 99% for the fire side and moved the outcome by **one battle in 600** while healing fell 6.4% — the mechanism worked and the harness could not see it. Level the pairing to a balance point **first**, then measure.

**`rapid_spin` still does not earn a slot** (it did not before #181 either): utility-for-utility it costs **99‰ rider-off and 102‰ rider-on**, so the gap the cleanse must make up is unchanged. It is not going uncast — 25,359 casts, 14,770 stacks stripped, halving the cut heals — it simply costs more than it saves. A cleanse in this engine spends a turn to remove one stack; an attack spends the same turn on damage.

`fire_fang` was chosen over `brine` because `brine` is **squirtle-only** and squirtle *is* the sustain build, so the anti-heal would only ever counter itself. And `fire_fang` **strikes twice**, which finally made #181's "one strike eaten, one through" branch reachable — it was latent and now has `TestOneStrikeEatenAndOneThroughDeliversOnce`.

## Open findings, not fixed

- **`price.go` rates a heal cut at nothing**, falling through as `taunt` does. `Suggest` never aims a fester at a healer on purpose and never discounts a heal it is about to have cut. Both errors run the direction every cap in that file errs in, so **every figure above is a floor** on what the status is worth.
- ⚠️ **Pre-existing log bug, found here, unrelated to the change:** every regeneration heal reports `Stacks == 0` — measured **15632 of 15632** on `main`, none with a positive count — so `heals 60 from regrowth x0` has been shipping. The `Healed` regen arm never carries the stack count.

Related: [[hexarena-block-cancels-the-rider]], [[hexarena-status-naming]], [[hexarena-tui-i18n]], [[hexarena-roster-cannot-price-damage]], [[fixture-hidden-branch]].
