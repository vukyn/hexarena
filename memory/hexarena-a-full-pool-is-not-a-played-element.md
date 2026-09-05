---
name: hexarena-a-full-pool-is-not-a-played-element
description: hexarena — an element with a full skill book can still be unplayable for a second character, because the pool may be built entirely around the first carrier's mechanic
metadata:
  type: project
---

Wind was obviously empty when `pokemon.dratini` needed it — two skills, both
`restrict.origins: [naruto]` — so the work was visible. **Electric looked
finished and was not.** It shipped **ten** skills, none restricted beyond
`origins: [pokemon]`, and a second electric character could carry every one of
them. The trap: all ten are built around `charge` — `charge_beam` and
`magnetise` put stacks on, `electro_ball`, `spark` and `overload` spend them —
which is `pokemon.magnemite`'s whole identity. Re-carrying the pool would have
shipped a second Magnemite with different stats.

**Measured, not assumed.** Raichu on Magnemite's chain kit read 8‰ against a fire
seat, 33‰ into s01 and 0‰ into s03 — a kit that is weak in the abstract and only
looked passable on the character it was designed around (Raichu still beat
Magnemite 606‰ *on that same kit*, which is the reading that says the kit is the
problem and not the body). It was drafted as a shipped `builds.json` entry and
pulled before merge for exactly that.

**What a second carrier of a full element needs instead:** its own signature,
gated so the first carrier cannot take it. `pokemon.pichu` got the species
`rodent` and two skills restricted to it — `volt_tackle` (the second shipped user
of `skill.Cost`, health paid up front) and `nuzzle` — plus the `static` passive,
which replies `stun` the way `venom_blood` replies `poison`. Same element, same
shared pool, a different thing to do with a turn.

⚠️ **Check the shape of a pool, not its size.** `jq '[.skills[]|select(.element
=="electric")]|length'` says ten and tells you nothing. What to read is whether
the pool has a mechanic in it — a status the skills pass between each other — and
whose mechanic it is. Related: [[hexarena-pricing-a-new-element-pool]] for the
empty case and [[hexarena-elemental-kits-swing-wider]] for what an all-elemental
kit costs.
