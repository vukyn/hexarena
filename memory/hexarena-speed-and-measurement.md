---
name: hexarena-speed-and-measurement
description: "hexarena PR#118 — speed is the dominant stat and cannot be priced by a win rate; band the TURN SHARE inside one battle. Naruto's 3 forms + the evolution-fork note"
metadata:
  type: project
---

hexarena PR #118 (merged 2026-08-28, `b66ad13`). `swiftness` ("thần tốc") grants `quickened`, permanent **+80 speed**, on Naruto@24 — its second trait.

## ⚠️ A WIN RATE CANNOT PRICE A SPEED TRAIT

Over 300 mirror duels (both ways) the rate **does not even order the amounts**:

```
+30 59.6%   +40 63.3%   +50 74.0%   +60 63.0%
+80 73.0%   +100 57.0%  +150 59.0%
```

+150 reads **below** +50. The **turn queue is discrete**: a few points of speed buy whether one more turn lands before the opponent acts, and which side of that line a seed falls on is lumpy — worse now that `Suggest` casts no-power skills and puts a summon in the queue.

**Band the TURN SHARE instead** — what the trait does, not what comes of it. Monotone across the whole sweep:

```
+30 2.6%  +40 3.5%  +50 4.4%  +60 5.6%  +80 7.9%  +100 9.9%  +150 14.8%
```

⚠️ **At 150 the win-rate band PASSES and the turn-share band FAILS.** That is how the bad measurement was caught.

⚠️ **Count both sides of ONE battle.** The first version compared totals of two separate sweeps = measuring battle **length**, and the faster unit ends battles sooner. It passed by luck, then reported swiftness with **fewer** turns while being exactly as fast. Same battle → length is shared → cancels.

## ⚠️ The house figure 150 does not transfer to speed

Every other permanent buff a trait grants is **150** (`toughened`, `kindled`, `unleashed`). Speed is worth far more per point — **speed is turns, and a turn is every other stat applied again**. Corroborated elsewhere: 10% off Squirtle's speed hurt more than 25% onto its defence.

Naming: `haste` already glosses "nhanh nhẹn" → `quickened` = "thoăn thoắt". Trait was "nhanh chân" until **`chân` turned out to be a `bodyWord`** → "thần tốc".

## Naruto's three forms

`Naruto@1 → Shippuden@16 → Tiên nhân@32` against `naruto.svg` / `naruto-shippuden.svg` / `naruto-sage-mode.svg` — before the 2-year training, after it, after learning sage art.

⚠️ **The count was right; both names sat ONE FORM AHEAD of their own art** (middle called "Tiên nhân" showing Shippuden art; last called "Vĩ thú hoá" showing sage-mode art). Renamed only — **no stat moved**. Vĩ thú hoá will be a **separate character**: a stage is the same unit later, that is a different unit.

## Roadmap note: an evolution line that forks

Choosing **how far** exists (PR#73 allowlist); choosing **which path** does not.
- Small half: `Line.Validate` refuses `MinLevel <= previous`.
- ⚠️ **Large half is `Furthest`, and it fails SILENTLY** — `Line.Allowed` returns a **prefix**, `StageAt` the last reached; with a fork the browser, `hexforge check`'s budget row and `fielded` each take whichever arm the file lists last.
- ⚠️ A prefix cannot express a stage **after** a fork → the line must become a tree. Design that first.
- Budget already fine (`Validate` prices each stage separately).

⚠️ Naruto is still the **only character with no build** in `builds.json`, and now has 2 traits.

Related: [[hexarena-mechanics-log]], [[hexarena-builds-catalogue]], [[hexarena-stat-bounds-policy]], [[hexarena-core-design]]
