---
name: hexarena-a-trait-that-changes-a-price-has-two-sites
description: hexarena — anything that changes what a skill costs must be read where the rating prices it as well as where the battle charges it, or the holder never casts the skill the trait exists for
metadata:
  type: project
---

`passive.Spares` (added for `magic_guard`, 2026-09-05) forgives a share of
`skill.Cost`. There are **two** places a cost is read and they answer different
questions:

- `Battle.spendHealth` (`turn.go`) — what the caster actually hands over.
- `pricing.spentHealth` (`price.go`) — what `Suggest` thinks it will hand over.

**Apply the trait to only the first and you build a unit that never casts the
skill it holds the trait for.** The rating still sees the full ask, still
declines, and the holder pays nothing for a cast it never makes — a bug that is
completely invisible to a test that reads health, because the health is correct.
Apply it to only the second and the unit bleeds while the rating believes it does
not.

**The shape that fixes it:** one helper both call. `healthCost(maxHP, known,
spared)` is shared; the *clamp* to what a unit can afford stays in `spendHealth`
alone, because `spentHealth`'s own note explains at length why the rating must
keep charging the full ask — a price that falls as the cast gets more fatal is
rated best exactly where it should refuse.

⚠️ **This generalises past `Cost`.** `price.go` is written under the rule *read
the resolving function, never a second copy*, and a trait is a second input to a
resolving function. `lifesteal` and `converts` already had this shape — the
pricing calls `p.fight.lifesteal(actor)` — so the pattern to copy exists; what is
easy to miss is that a **new** field needs the same treatment, and nothing in the
type system says so.

**The test that catches it is one test with two subtests**, deliberately:
`TestATraitSparesAHealthCostAtBothSites`. Verified by mutating each site
separately — sparing dropped at the rating fails *the rating*, sparing dropped at
the charge fails *the charge*. Two independent tests would let somebody delete
the awkward one. Related: [[mutate-the-producer-not-just-the-logic]].
