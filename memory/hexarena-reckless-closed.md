---
name: hexarena-reckless-closed
description: "hexarena reckless CLOSED (#155/#156) — all three levers measured dead; a stat SATURATES so a -400‰ term on base 400 fights at 290"
metadata: 
  node_type: memory
  type: project
  modified: 2026-08-29T07:51:32.116Z
---

**`reckless` is kept as it stands and the item is closed into *Decided against*.** It grants `unleashed` (+300‰ attack) and `bare` (−400‰ defence *and* −400‰ dodge); the dragon build reads **22.1%** against the fire line and **96.6% / 93.0%** against the rest of the cast. That is an extreme trade — a shape, not a bug.

All three levers the item named are measured dead (build duel in `internal/seed/dragon_test.go`, both arrangements, 3000 battles a row; referents `blaze` 38.9% floor / `blood_thirst` 55.1% ceiling):

- **drop `bare`'s dodge clause** → +2.8 only (22.0→24.8). Decomposed, the **defence term is 88%** of the cost (dodge-only reads 43.4%, no `bare` 46.3%), and the clauses **add** — the "dodge is superlinear against a burn-and-detonate opponent" argument is disproved.
- **+ a `vulnerability`** (six harmful statuses at −200‰) → **−3.3**, *below* where it started. So `vulnerability` **still has no shipped user**.
- **soften the defence magnitude** → cannot land: **both gates flip at the same two-point rung** (duel clears the floor between −95 and −90; the cast-wide pair saturates squirtle at 100% across exactly that step) because both are one event — a strike crossing a kill threshold.

⚠️ **Two engine facts worth more than the fix.** **A stat SATURATES rather than applies** — `modifier.Set.Stat` calls `scale.Saturate(base, raw−base, limit, floor)`, so a −400‰ term on a base of 400 fights at **290, not 240**, and that lever's whole reachable range is 290–391. A grid computed as though a share applied straight sits entirely under the floor (mine did). And **the dial moves in steps with plateaus, not a curve** — 14 amounts gave 9 rates, four returning the identical figure — so it cannot be tuned to a target.

⚠️ **A vulnerability costs more than its arithmetic, because the opponent steers**: `pricing.landed` calls `fight.resist` on the *target*, so the rating sees a unit inviting a status and aims one at it deliberately.

**How to price a trait:** `TestRecklessSpendsNoMoreThanItBuys` (#155) reads damage **dealt and taken off the event log**, with and without the trait, and refuses a cost over twice what it bought — a win rate cannot price a stat. A **stat-counting** shape rule was written twice and dropped both times: it counts stats, never magnitudes, so it passes a `bare` at −900 and refuses one at −25. ⚠️ **A `weigh`-shaped instrument for a trait field is the missing tool** — every reading above needed shipped JSON hand-edited, a duel run, and the file put back.

Unmeasured and open: whether the 22% belongs to `inferno` and the fire line's detonate rather than to this trait.

Related: [[hexarena-roster-cannot-price-damage]], [[hexarena-bout-and-waiting]], [[hexarena-builds-catalogue]].
