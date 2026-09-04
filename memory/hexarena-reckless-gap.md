---
name: hexarena-reckless-gap
description: hexarena's dragon build sits at 22% because reckless costs two stats to buy one — decomposed, and the detonate theory is disproved
metadata:
  type: project
---

**The Charmander dragon build wins 22.1% head-to-head against the fire build** (300 seeds × both orders — must swap sides, the queue breaks a tie by enlistment and slot one is worth ~50 points). Test band is 15–85%, so it passes while sitting at the edge. Builds are meant to be **sidegrades**.

⚠️ **The written-down theory was wrong and is disproved — do not re-raise it.** "The fire line has a detonate and the dragon line has none" → the line was *given* one (`dragon_drive`, detonating the `expose` its own `dragon_claw` applies) and fielding it moves 22.0% → **21.2%**, i.e. **−0.8**, slightly the wrong way. That is the pricing rule working: a detonate may not beat leaving the status alone by more than 2×, and `expose` is cheap (worth 102) against `burn` (548), so the dragon detonate is capped near ⅓ of `inferno`'s burst.

**Decomposition, one change at a time over the same 3000 battles:**

| change | rate | delta |
|---|---|---|
| shipped | 22.0% | — |
| dragon fields the detonate | 21.2% | −0.8 |
| fire loses its detonate | 32.9% | +10.9 |
| dragon drops `reckless` for `blood_thirst` | **55.1%** | **+33.1** |
| dragon drops `reckless` for `blaze` | 38.9% | +16.9 |
| both | 53.4% | +31.4 |

**`reckless` ("liều mạng") is the gap.** It grants `unleashed` (+300‰ attack) **and** `bare` (−400‰ defence **and** −400‰ dodge), all permanent — two stats paid for one. Worse, its opponent is the fire build whose `inferno` amplifies ×3.5 off `burn`, so losing armour and dodge is exactly what that matchup punishes hardest.

`TestRecklessIsATradeAndNotAGift` only asks whether *something* is given up, never whether **too much** is. Nothing catches this.

**The remaining work is DATA** (`passives.json` / `statuses.json`): soften `bare`, drop its dodge clause, or raise `unleashed`. ⚠️ **Chain caution:** `vulnerability` (a negative `Resists` share) is built and `reckless` is its natural first user — adding it would sink the build *further*, so it can only land together with compensation, never on its own.

Sources in-repo: `internal/seed/dragon_test.go` § `TestTheDragonLineCanSpendWhatItApplies` (the table), `TODO.md`.

See [[hexarena-mechanics-log]], [[hexarena-builds-catalogue]], [[hexarena-stat-bounds-policy]].
