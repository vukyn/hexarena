---
name: a-golden-screen-shows-only-the-rows-that-fit
description: The spar golden in cmd/hexforge-tui is a 24-row WINDOW over a 27-row table, so it shows 9 opponents; 4 shipped characters carry a reply trait and only 1 of them is above the cut — "one row moved" is not "one matchup moved"
metadata:
  type: feedback
---

A screen golden records **what was drawn**, and a drawn screen is a window. The
spar screen in `cmd/hexforge-tui/testdata/screens.golden` says `27 rows` in its
own header and then prints nine of them before `… cut off; a taller window shows
the rest`. So the golden is a sample of the table, chosen by scroll position and
terminal height, not the table.

**Measured (2026-09-10, the per-strike reply).** Four shipped characters bring a
replying trait as their first learnset trait, which is what `forge.seedKit`
gives a spar duellist: `pokemon.bulbasaur` (venom_blood), `pokemon.happiny`
(ballast), `pokemon.pichu` (static) — plus onix/squirtle, which declare `thorns`
but not first, so a spar never fields it. Exactly **one** of those, bulbasaur,
sits above the cut. The golden therefore moved by one row (5.0% → 4.0%,
10-190-0 → 8-192-0, 46 → 44 turns) and said nothing whatever about the other
three.

**Why it matters:** the natural reading of a one-row golden diff is "the change
touched one matchup". Here the honest reading is "the change touched at least
one matchup, and the golden cannot see the rest of them". A balance report built
off the diff alone understates the blast radius, and the understatement is
invisible.

**How to apply:** before quoting a spar golden as a balance measurement, count
the rows the *table* claims against the rows the *screen* printed, and work out
which subjects fall below the cut. If the ones that fall below are the ones the
change is about, take the measurement with `hexforge spar` (or `forge.FightSquads`)
instead of reading the golden. Related: [[a-spar-golden-is-fought-from-the-cast]],
[[moving-a-call-into-a-loop-reorders-the-rolls]].
