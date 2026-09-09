---
name: a-prereg-floor-is-the-point-when-it-fails
description: Reading C came in at +48‰ against a floor of +50‰ — the pass is to ship the null, not to run a second board; and the mutation check is what separates an honest null from a blind instrument
metadata:
  type: feedback
---

**A floor written down before the run only does anything on the day it fails by
2‰. Ship the null; do not go looking for a board that clears it.**

**Why:** `DAT-002` sub-item 1 (2026-09-09) pre-registered *"reading C must move at
least one non-saturated pairing by ≥ +50‰"*, derived as just under 3σ (binomial σ
at p = 0.5 over 800 battles ≈ 17.7‰). It moved **+48‰**. The gate's own wording
says "at least one **pairing**", so adding a second opponent squad after seeing a
near-miss would have been *inside the letter* of the gate and is exactly the move
the gate exists to stop — the number to beat was fixed before the dice, so the
opponent set has to be too.

⚠️ **A null is only worth shipping if the instrument is shown to have been able
to see something.** Two checks did that here and both are cheap:

- **Reading B**, the whole bonus against `--without water_tide`, read **+240‰** on
  the same board. So the instrument was not dead; the *increment* was small.
- **The mutation:** a rung at 4 declaring the stacks rung 3 already grants read
  **342‰, identical in every field** to the shipped book. That is what says the
  +48‰ is the third stack and not the extra `bonus_held` row. Without it, "+48‰"
  and "an artefact of one more event" are indistinguishable, and a null on an
  artefact says nothing at all. → `[[feedback_a_null_needs_two_controls]]`.

**How to apply:** when a pre-registered figure lands just under, write the number,
the floor, its derivation, and *why the instrument could see*, then stop. Say what
the reading is a reading **of** — here a side with two-and-a-half healing sources,
so the honest next step is a differently-built board rather than a second look at
this one. The item stays open with its premise rewritten, which is the
`DAT-007`/`DAT-009`/`DAT-010` shape this repository already uses.

⚠️ **A null still ships everything that is not the number.** Step 3's parse guard
(a rung may not grant more stacks than the status holds) went in regardless: it is
validation, not a widening of the effect vocabulary, and it closes the hole the
measurement walked through — a rung declaring more stacks than the cap parses,
loads, draws, **fires**, and changes nothing.
