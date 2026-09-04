---
name: hexarena-elemental-kits-swing-wider
description: hexarena — how much of a kit is elemental rather than neutral decides how wide its matchup spread is, and a mostly-neutral kit quietly dodges its own character's counter
metadata:
  type: project
---

Two ground characters, measured in the same 3v3 wrapper against the same squad
(s03, which fronts a grass `leech_seed`/`synthesis` wall):

| unit | kit | vs s03 |
|---|---|---:|
| Machamp | `cross_chop`, `submission`, `seismic_toss`, `inner_focus` — **one** ground skill | **408‰** |
| Garchomp | `dig`, `stone_edge`, `earthquake`, `sand_tomb` — **four** ground skills | **50‰** |

Both are `ground`. Grass beats ground. Only one of them pays for it, because the
elemental multiplier is applied **per skill**, not per unit: Machamp's kit is
three-quarters `neutral`, so three-quarters of its damage never meets the chart at
all.

**Why it matters when authoring:** a character's affinity looks like the thing
that decides its matchups, and it is not — **the element coverage of its kit is**.
An all-elemental kit is a deliberate choice for a wide swing (Garchomp reads 862‰
into s01, where ground beats water, and 50‰ into s03), and a kit padded with
neutral skills is a deliberate choice for a flat one. Neither is wrong; picking
one by accident is.

⚠️ **Do not read a low figure into a counter as the kit being undertuned.** That
was the first reading of Garchomp's 50‰ and it was wrong — the same character on
its neutral lineage kit reads 491‰ into the same squad. A character whose
elemental kit has a hard counter needs a *second* kit, not bigger numbers; see
[[hexarena-pricing-a-new-element-pool]], where Dratini landed in exactly the same
shape and the fix was the same.

**How to apply:** when a new kit measures wildly across opponents, count its
non-neutral skills before touching a power. Then fight a **shipped** character of
the same element in the same seat — that is what separates "this element is
having a bad matchup" from "these numbers are wrong".
