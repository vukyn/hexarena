---
name: hexarena-a-summon-needs-a-gap-between-format-and-cap
description: hexarena — one constant was both the widest team and the largest format, so a full 5v5 side left summonPlaces no room and every summoning skill was a dead slot; CLOSED by splitting it into hex.BoardSlots (9) and hex.MaxSquadSize (5), and 5v5 is open
metadata:
  type: feedback
---

⚠️ **CLOSED 2026-09-09 — the finding stands, the constant does not.**
`hex.MaxTeamSize` was **5** and did two jobs: the widest team the board admits,
and the size of the largest format. So at five a side `battle.summonPlaces`
computed `room = MaxTeamSize - 5 = 0`, `summonWorth` priced every summoning skill
at nought, and `Suggest` never cast one. It is now **two** constants —
`hex.BoardSlots = FormationCols*FormationRows` (9) and `hex.MaxSquadSize` (5) —
and `cmd/hexarena-host` opens `-format 5`.

**Measured, mirrored, 100 seeds** — `split` cast / copies arrived:

| 3v3 | 4v4 | 5v5 | 5v5, cap probed to 7 |
|---|---|---|---|
| 400 / 400 | 400 / 400 | **0 / 0** | 400 / 400 |

The cliff is exactly at the cap, and the probe is what says the **cap** is the knob
rather than the format. `split`, `shadow_clone`, `summon_toad` and the
`diglett.three` build were dead slots at five a side. → `ENG-013`.

⚠️ **The 400 was the FIXTURE, not the engine — and the harness was never
committed, so it could not be reproduced.** Re-run in 2026-09 over three
independent fixtures on the same protocol, the *shape* reproduced every time (a
cliff to nought at five and nowhere else) and the *magnitude* never did: 462 /
464 / 410 at three a side. Quote a cast count with the squad it came off, and
**commit the harness or expect to rebuild it**. The split then took 5v5 from 0 to
752 / 738 / 778 with the mirror at 500‰ and 0 endless in all eighteen readings.

⚠️ **Everything else about five a side is fine, which is the part that was
assumed backwards.** The mirror is exactly 500‰ with 0 endless over 200 battles,
and the screened formation is worth **more** there — 935‰ against 850‰ at three,
priced with the same bodies on both sides and one side's aces moved forward. So
the arrange phase gets more decisive as the board fills.

**How to apply.**

- ⚠️ **A constant that is both a board bound and a format size will collide with
  any mechanic that uses the spare room.** Ask what the *gap* is for before
  raising or reusing such a number. The fix is to split it, not to raise it: two
  names, each read by the layer that means it, and the correspondence checked one
  layer up. ⚠️ Both halves are `int`, so **the compiler cannot tell a missed
  conversion from a correct one** — retire the old name entirely and let the build
  enumerate all 61 sites, rather than find-and-replacing.
- ⚠️ **A predicted regression can fail to appear because a different bound was
  already binding.** 3v3 was expected to move (a repeated summon could now reach
  further) and came back bit-identical: `split` gates itself on `sundered` below
  two stacks, so it casts twice a battle whatever the board allows. Measure the
  gate you did not think about.
- ⚠️ **"No golden moves" was wrong, and the miss was about which fixture a screen
  fights.** The estimate reasoned over `roster.json`, which fields no summoner.
  `cmd/hexforge-tui`'s spar screen fights the whole **cast**, and `naruto.naruto`
  carries two summons — its median duel moved 58 → 60. Ask which *book* a golden
  is drawn from, not only which data file changed.
- **`roster.json` was never five a side.** Three places claimed "the shipped
  balance was read at five a side" — `TODO.md`, `wire.Format5v5`'s doc comment,
  `cmd/hexarena-host`'s refusal — and all three were wrong from the day they were
  written; the roster had six units when the first one was authored. Check a
  premise about which board a figure came from before re-measuring against it.
  See [[hexarena-a-stale-referent-outlives-the-conclusion-it-blocked]].
- **Price a formation with the same bodies on both sides**, moving only the
  placement. Two different squads at two sizes measures the squads.
- **`same_column` stops being a choice at five a side**: its rung is three, and
  five units with an empty front column have two columns to stand in, so one holds
  three of them. ⚠️ A second bonus looked format-sensitive in the same run and was
  the **fixture** — the fourth and fifth bodies were the electric ones, so the
  count arrived with them. Hold the bodies fixed or the reading is about the squad.
