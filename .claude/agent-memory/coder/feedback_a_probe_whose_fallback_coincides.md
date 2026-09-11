---
name: a-probe-whose-fallback-coincides
description: "Nil-ing PlayScreen.Fight could not detect splashUnder reading the battle at draw time — the hex.SideAlly fallback equals the answer on every board; -race was silent too because Side/Affinity are never written. Mutate the READING instead."
metadata:
  type: feedback
---

To prove a drawing reads the **reading** and not the battle, mutate the reading
— do not take the battle away, and do not lean on `-race`.

**Why:** `internal/screen`'s `splashUnder` resolved the caster's half with
`p.Fight.Unit(p.Pending.Unit)` *at draw time*, which on a live screen is the
exact race `readBattle`/`Attach` exist to remove (it had been there since the
splash rows shipped — the rule was written for `Attach` and lost one function
away). Two obvious probes both said "fine":

- **`p.Fight = nil`, then compare the two drawings.** The code's fallback when
  the battle is absent is `hex.SideAlly`, and the prompted unit is on the ally
  side in *every* battle this screen opens — so the defective code and the
  correct code agree on every reachable board. The probe passed with the defect
  in place. (The same probe *does* bite the matchup mark, because there the
  fallback is "no mark" and the correct answer is a mark — so a fallback that
  coincides is the discriminator, not the probe.)
- **`go test -race` on `cmd/hexarena-tui`'s forced-overlap redraw test.** Silent:
  `Unit.Side` and `Unit.Affinity` are written once at enlist and never again, and
  with no summon in the fixture the `units` slice header is not rewritten either.
  The detector sees memory races, not contract violations. It would fire on a
  draw-time read of `HP`, `Dead`, `Cooldowns` or `Statuses`; it cannot fire on
  the immutable fields, which are exactly the ones a drawing wants.

What discriminated: build a `playReading` with the prompted unit's `Side`
flipped and require the resolved footprint to **move**. Only code that consulted
the reading can respond to it. (`TestTheSplashFrameComesFromTheReading`; it needs
a vacuity net, because `column` and `single` mirror and catch the same cells in
either frame.)

**How to apply:** whenever the claim is "X is taken from the snapshot", ask what
the snapshot-free path would answer. If its fallback happens to equal the right
answer on every reachable input, the absence probe measures nothing — mutate the
snapshot's own contents and assert the output follows. Widening the `-race`
redraw test is still worth doing (it now opens the aim list on the renderer's
own copy), but treat it as a net for *mutable* fields only.

Related: [[feedback_a_well_formed_measurement_can_measure_nothing]],
[[feedback_a_mutation_must_hit_the_arm_you_claim]],
[[feedback_measure_which_guard_masks]].
