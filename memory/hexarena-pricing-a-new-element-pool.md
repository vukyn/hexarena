---
name: hexarena-pricing-a-new-element-pool
description: hexarena — a brand-new element pool has to be priced by measurement against an existing pool, and a squad figure means nothing without a shipped-character control in the same wrapper
metadata:
  type: project
---

Shipping `pokemon.dratini` (Dratini → Dragonair → Dragonite, PR for the `tempest`
preset) needed **the first draftable wind carrier**, and wind turned out to be a
dead element: both wind skills in the book were `restrict.origins: [naruto]`, so
the pool had to be authored from nothing. See [[hexarena-shipping-a-character]]
and [[hexarena-squad-builder]].

**Hand-picked numbers for a new pool were badly wrong and nothing said so.** The
first six wind skills were written by eye against the fire and electric rows and
read **0–225‰** in a 3v3 — a rout, not a tuning gap — while the same character on
the existing neutral dragon kit read 450–833‰. The data parsed, every gate in
`internal/seed` was green, and the only thing that knew was `FightSquads`.

**What actually calibrates: sustained damage per turn, not power.** `power ×
strikes × accuracy ÷ (cooldown + 1)` puts every skill on one axis, and the shipped
pools cluster on it — a starter's cd-1 filler sits near 475, a cd-2 heavy hitter
near 480, and a kit's best two summed is what a squad fight is deciding. The
original wind kit summed 626 against fire's 955; repricing the pool so its best
two summed ~890 moved the same matchup from 225‰ to **516‰** against a fire
control, with the mirror still reading exactly 500‰.

⚠️ **A squad-vs-squad figure is unreadable without a baseline third unit.** The
measurement wrapper was `[X, Machamp, Clefable]` vs a shipped squad, and the wind
build read **0‰ against s02** — which looked like a broken character until a
shipped **Magnezone** dropped into the same wrapper read 0‰ too. The wrapper loses
to s02 whoever the third unit is. The same control settled the opposite worry:
the dragon kit's 833‰ against s01 belonged to *the kit*, because Charizard on the
identical four skills read 817‰, and Dragonite lost the head-to-head 433‰.

⚠️ **A pool that has an elemental counter will read near zero into it, and that is
the chart working.** The repriced wind kit still reads 16‰ into s03, which is
grass (grass > wind on the cross cycle) fronted by a `leech_seed`/`synthesis`
wall: 0.667× damage cannot out-pace a heal. Do not "fix" that by raising the pool
— the character's answer is its other kit, and having one is why it carries
neutral lineage skills beside an elemental pool.

**How to apply:** author the pool, then fight it in the wrapper against (a) a
mirror — must read 500‰, (b) a shipped character in the same wrapper, and (c) the
same character on a different kit. Only then accept goldens. A temporary
`internal/forge` test that appends squads to a copied data dir and calls
`FightSquads` is the whole instrument; delete it before committing.
