---
name: a-spar-golden-is-fought-from-the-cast
description: A "no golden moves" estimate reasoned over roster.json missed cmd/hexforge-tui's spar screen, which fights the whole CAST — ask which BOOK a golden is drawn from, not only which data file changed
metadata:
  type: feedback
---

ENG-013's blast-radius estimate said **no golden would move**, and argued it
carefully: `roster.json` and `squads.json` field no summoner (grepped, zero hits
on all three skill ids), `replay.golden` reads `summoned 0` / `split 0`, the
catalogue goldens quote skills as descriptions and read no cap, and the one
rendered *number* (`i18n.SquadsSubtitle`'s "tối đa 5") stayed 5 because that site
was classified as the fielding cap.

Every one of those was true, and a golden moved anyway.
`cmd/hexforge-tui/testdata/screens.golden`, the **spar** screen: a spar is a duel
against the whole **cast book**, not against `roster.json`, and `naruto.naruto` is
the one shipped character whose kit holds `shadow_clone` and `summon_toad`. On a
duel board its side could now hold clones and a toad at once where the old cap
allowed one of the two, so its median duel read **60 turns where it read 58** —
four identical cells, two window sizes × two languages. Rate, record and
initiative did not move at all.

**How to apply.**

- ⚠️ **Ask which BOOK each golden is fought from**, not only which data file you
  edited. `roster.json` (shipped roster), `squads.json` (saved squads),
  `builds.json` (the catalogue) and `cast.json` (every character) reach different
  goldens, and a screen that walks *the cast* sees carriers no roster fields.
- **Grepping the data for a mechanic's ids answers "is it fielded", not "is it
  fought".** The census/spar/catalogue screens field things nobody authored into
  a squad.
- **Prove causation before accepting, and prove it by pinning the one line
  back.** Setting `summonPlaces`'s `room` to the old cap turned the golden green
  again — one command, and it converts "presumably this" into a fact. Revert the
  probe by **editing the line back**, never `git checkout`.
- **A moved golden is a finding to write down, not noise to accept.** Four cells
  of one column, with rate and record unchanged, is a length change rather than a
  balance answer — and saying which it is, in the doc, is the whole value.
- `make golden` (never `go test ./... -update`) is the accept path; run it once
  everything else is green so the diff is only the file you expect.

See [[fixture-cast-edit-costs-goldens]] and
[[a-golden-can-be-red-before-you-touch-it]].
