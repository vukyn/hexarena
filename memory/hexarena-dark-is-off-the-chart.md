---
name: hexarena-dark-is-off-the-chart
description: hexarena — dark sits in no elemental cycle, only in mutual with light, so a dark kit has a flat matchup profile by construction where every other element swings
metadata:
  type: project
---

`elements.json` puts every element in a cycle — organic, industrial, cross — and
those cycles are what make a kit swing. **Dark is in none of them.** Its only
entry anywhere is `mutual: [["light", "dark"]]`, so against nine of the eleven
elements a dark blow is exactly neutral, both ways.

**What that does to a character.** Measured 2026-09-05 in one 3v3 wrapper, same
seat, four opponents:

| unit | vs fire seat | vs s01 | vs s02 | vs s03 |
|---|---:|---:|---:|---:|
| Alakazam (`dark`) | 391‰ | 241‰ | 512‰ | 93‰ |
| Garchomp (`ground`) | 350‰ | 862‰ | — | 50‰ |
| Dratini (`wind`) | 516‰ | 225‰ | — | 16‰ |
| Raichu (`electric`) | 145‰ | 495‰ | — | 0‰ |

The elemental characters run from near-nought to near-total on the same four
squads; the dark one does not. That is the chart, not the numbers — see
[[hexarena-elemental-kits-swing-wider]] for the other half of the same fact, that
a kit's *element coverage* is what decides its spread.

**How to apply.** A dark character is bought for **consistency**, and it should be
tuned to sit a little below the elemental characters' *good* matchups because it
never has their bad ones. Do not read a flat mid figure as "undertuned" and raise
it — that is how a dark unit becomes the strongest thing in the cast without any
single measurement looking wrong. The reverse trap is live too: `pokemon.gastly`
and `pokemon.mewtwo` read **25‰ and 16‰** against the fire seat, the two weakest
in the cast, so "the shipped dark characters" is a **bad** control to size a new
one against. Use the whole cast.

⚠️ There is no `psychic` element — dark carries it. `psystrike`, `psycho_cut` and
`psychic` are all dark or neutral, which is why Mewtwo occupies most of what an
Alakazam would otherwise have been, and why `pokemon.abra` had to be bought for
something else entirely.
