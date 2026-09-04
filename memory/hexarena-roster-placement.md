---
name: hexarena-roster-placement
description: hexarena roster is placed ace-at-back behind an adjacent screen; placement is purely defensive under rank reach and is worth 27.6%→47.3%
metadata:
  type: project
---

**PR #136 (2026-08-28): the seed roster's aces moved from the FRONT column to their own BACK column (`2,1` → `0,1`), on both sides. 27.6% → 47.3% ally over 4000 seeds; smoke 12/40 → 24/40.** Two numbers in `roster.json`, nothing else touched.

**Under rank reach, placement is PURELY DEFENSIVE** — where a unit stands decides what can be aimed *at* it and nothing about what it can hit — so *ace at the back* is the dominant shape, and the roster gives it to **both** sides rather than to one. The old formation had each ace in front, authored for the board where reach was distance.

Three load-bearing parts, each measured separately:
- **screen** (pair at `1,0`/`1,1` is the first occupied rank) = the whole 27.6 → 47.3
- **adjacency**: splitting the pair to `1,0`+`1,2` reads **31.1%** — an area shape catching both is most of what the young units do
- **empty front column**: an empty rank costs no range, so the aces sit at depth **2** not 3 → the shipped board is the demonstration of that rule

Guarded by `TestTheShippedFormationScreensItsAce` (mutation-checked both ways). ⚠️ Flattening the formation is a balance change that reads as a tidy-up.

**Levels were NOT the answer and were not touched.** Floor is 20 on each young enemy (`dragon_rage` learned at 20; "two earned traits" is a roster contract), and the 20..30 grid on the screened board spans **40%–82%** ally with the shipped 30/30 at the bottom.

⚠️ **Every balance figure older than #133 predates ranks.** README/CLAUDE.md keep them as the shape of a lesson, not as numbers that carry.

Doc debt from #133 paid here: README's **dead range ladder** removed, `hex.ReachNeeded` deleted (dead outside tests, asserted the dead ladder), duel-slot claim moved off the slot onto the kits (`TestNoKitIsUnaimableInADuel`).

See [[hexarena-core-design]], [[hexarena-stat-bounds-policy]], [[hexarena-speed-and-measurement]].
