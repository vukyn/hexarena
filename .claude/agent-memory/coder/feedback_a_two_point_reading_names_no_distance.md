---
name: a-two-point-reading-names-no-distance
description: RAT-008 — "the term must fall 30x" was read off /4 and /64; sweeping every rung put the recovery at /5-/6, and the response was a CLIFF, so a correct half-sized term measured identically to no term at all
metadata:
  type: feedback
---

Sweep **every** rung before quoting how far a factor has to move, and before
building a term to move it that far.

**Why.** `hexarena` `RAT-008` said `pricing.hidden` "has to fall by roughly thirty
times" before the attacks win the turn. That number came from two samples — the
term at `/4` (175‰) and at `/64` (775‰) — with nothing taken in between. Sampled
at every rung it reads: `/1` 110‰, `/2` 145‰, `/3` 105‰, `/4` 175‰, `/5` **555‰**,
`/6` **775‰**, then a flat 775‰ at `/8`, `/16`, `/32` and `/64` alike. **`/64` was
a reading of the plateau, not of the distance.** The real figure is five or six.
The sweep cost one build per divisor, about a second a point.

⚠️ **Two consequences, and the second is the one that wasted the session.**

1. **A win rate is non-monotone in a price** (110, 145, 105, 175 below the cliff),
   so two samples name a distance only if you already know the curve between them
   is monotone — and here the repo's own docs say it is not, about
   `swiftness` and about the column bonus.
2. **Where the response is a cliff, a partial correction measures identically to
   no correction.** The term I built (`pricing.kept`) was *right* and *correctly
   sized*: it priced 550 against a denial of 1293, exactly what the derivation
   predicted, a 43% cut. It moved **no win rate in twelve measured rows**, because
   743 is still on the wrong side of a cliff that needs ~259. The nought was the
   instrument being unable to see a half-sized term — not the term being wrong.

**How to apply.**

- Before writing a term to close a measured gap, **sweep the gap first** and find
  where the response actually changes. Then check on paper whether the term you
  are about to write can clear that point. If it cannot, the measurement afterwards
  cannot distinguish it from nothing, and building it is a day spent to read a
  nought you could have predicted.
- When a quoted "N×" comes from a prior session, look at **how many points it was
  read from**. Two is not a curve. This is the same failure shape as
  [[feedback_measure_which_guard_masks]] — a plausible number attached to the
  wrong object — and the fix is the same: measure the thing rather than infer it.
- Keep the reverted term's figures. A refusal carrying `kept(machop)=550`, the
  cliff at `/5`–`/6` and *why* the expressible half is structurally too small
  (`selfSpendable`'s gated arm prices fuel at `strike/5`) is worth more than the
  term would have been. The repo's standing rule — an argument that sounds right is
  not a measurement, so revert rather than keep it as insurance — is what decided it.
