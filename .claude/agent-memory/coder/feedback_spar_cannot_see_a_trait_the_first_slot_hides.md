---
name: spar-cannot-see-a-trait-the-first-slot-hides
description: hexforge spar fields a character's FIRST trait and its FIRST FOUR skills, so a rule about trait × multi-strike is unmeasurable there — 3 carriers, 2400 battles, byte-identical output; the squads are the instrument
metadata:
  type: feedback
---

`hexforge spar` is the reflex instrument in this repository and it has **two**
narrowings, not one. The known one is the trait: `forge.seedKit` takes
`firstOf(character.PassivesAt(...), cast.TraitSlots)` and `TraitSlots` is 1, so
a spar duellist brings the character's **first** trait. The one that cost this
session is the other half of the same line — it also takes
`firstOf(character.SkillsAt(...), cast.SkillSlots)`, so it brings the **first
four** skills.

A rule that needs *two* things on one unit is therefore unmeasurable by spar
unless both happen to be first.

**Measured (2026-09-11, the per-strike drain).** The change is only visible where
a drain meets a **multi-strike** skill. Eighteen shipped characters carry
`blood_thirst` or `last_gasp`; exactly three carry it **first** — gastly,
machop, riolu — and not one of those three has any of the nine multi-strike
skills in its first four. So:

```
pokemon.gastly  brings shadow_claw shadow_ball night_shade spite  and blood_thirst
pokemon.machop  brings brace wrecking_swing cross_chop rock_throw and blood_thirst
pokemon.riolu   brings metal_claw reversal aura_sphere force_palm and last_gasp
```

3 characters × 200 seeds × 2 slots × 25 opponents = **2400 battles a side**, and
`diff` on the two reports was **byte-identical**. That is a true null and it is
also a measurement of nothing: the instrument could not have shown a difference.

**The instrument that could** is `forge.FightSquads`, because a *squad* names its
units' traits and skills explicitly instead of taking the first of each. Two
shipped placements pair a drain with a multi-strike skill —
`s01/poliwrath` (`blood_thirst` + `pummel`) and `s03/venusaur` (`blood_thirst` +
`razor_leaf`) — and fighting those two against all six squads over 200 seeds
moved 2 of 12 matchups by 3‰ each, in opposite directions.

**How to apply:** before quoting a spar null, print the `brings …` line the
report's own header carries and check that **both** halves of your rule are in
it. If they are not, say "spar cannot field this pairing" rather than "spar
found no effect", and reach for `FightSquads` over the authored squads. See
[[a-golden-screen-shows-only-the-rows-that-fit]] for the same trap one layer up,
and [[a-well-formed-measurement-can-measure-nothing]].

⚠️ **And check your probe compiles.** The first `FightSquads` probe named
`Tally.Won/Lost/Drawn` (the fields are `Wins/Losses/Draws/Endless`), so both
sides produced an **empty** file and `diff` said "identical". Print the output,
never just the diff verdict — two empty files pass every equality test there is.
