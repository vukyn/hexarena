---
name: hexarena-a-summon-needs-a-gap-between-format-and-cap
description: hexarena — hex.MaxTeamSize is both the widest team and the largest format, so a full 5v5 side leaves summonPlaces no room and every summoning skill is a dead slot; measured 400 casts at 3 and 4 a side, 0 at 5, and a probe at cap 7 restores it
metadata:
  type: feedback
---

`hex.MaxTeamSize` is **5** and it does two jobs: the widest team the board admits,
and the size of the largest format. So at five a side `battle.summonPlaces`
computes `room = MaxTeamSize - 5 = 0`, `summonWorth` prices every summoning skill
at nought, and `Suggest` never casts one.

**Measured, mirrored, 100 seeds** — `split` cast / copies arrived:

| 3v3 | 4v4 | 5v5 | 5v5, cap probed to 7 |
|---|---|---|---|
| 400 / 400 | 400 / 400 | **0 / 0** | 400 / 400 |

The cliff is exactly at the cap, and the probe is what says the **cap** is the knob
rather than the format. `split`, `shadow_clone`, `summon_toad` and the
`diglett.three` build are dead slots at five a side. → `ENG-013`.

⚠️ **Everything else about five a side is fine, which is the part that was
assumed backwards.** The mirror is exactly 500‰ with 0 endless over 200 battles,
and the screened formation is worth **more** there — 935‰ against 850‰ at three,
priced with the same bodies on both sides and one side's aces moved forward. So
the arrange phase gets more decisive as the board fills.

**How to apply.**

- ⚠️ **A constant that is both a board bound and a format size will collide with
  any mechanic that uses the spare room.** Ask what the *gap* is for before
  raising or reusing such a number.
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
