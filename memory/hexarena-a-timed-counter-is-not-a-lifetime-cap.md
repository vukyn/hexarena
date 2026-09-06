---
name: hexarena-a-timed-counter-is-not-a-lifetime-cap
description: hexarena — Condition.BelowStacks caps a skill only while its counter survives, and a timed status caps six turns rather than a battle; the counter has to be permanent
metadata:
  type: project
---

`Condition.BelowStacks` reads how many stacks the caster holds, so the cap lasts
exactly as long as the stacks do. `status.Set.Apply` refreshes **every** stack's
duration when a new one lands, which hides the problem while the skill is still
being cast — and then the last cast is the last renewal, the stacks run out
`duration` turns later, and the allowance quietly comes back.

`max_duration` is **6** and a duel here runs 40–90 turns, so "twice a battle"
spelled with a timed counter is really "twice every seven turns". Measured on
`pokemon.diglett` with the counter timed instead of permanent: **70 of 120 duels
went over the allowance and 21 reached five splits**, against a designed cap of
two.

**The fix is `"permanent": true` on the counter.** `Set.Tick` skips a permanent
entry whole, so the tally never expires. Nothing refuses a *skill* from applying
a permanent status — the Apply/Remove versus Hold/Release split is about
consuming, and a cap consumes nothing.

⚠️ **Choose the category so nothing can strip it.** A cap somebody can cleanse is
not a cap: `rinse` strips `charge`, `rapid_spin`/`heal_bell` strip
`dot`/`stat_debuff`/`heal_cut`, `spite` strips `shield`/`regen`. `buff` is
untouched by every shipped cleanse, and a counter carrying no modifiers is inert
in the rating besides. `reserve` looks apt and is not — `price.go` reads that
category in five places.

⚠️ **An engine test cannot catch this and the one that existed did not.** It runs
a handful of turns, which a timed counter survives. The claim is about a **whole
battle**, so the test belongs in `internal/seed` over real duels, with the
counter's permanence held as a premise — see
`TestTheShippedSplitIsCappedForAWholeBattle`, which fails loudly rather than
quietly measuring a window.
