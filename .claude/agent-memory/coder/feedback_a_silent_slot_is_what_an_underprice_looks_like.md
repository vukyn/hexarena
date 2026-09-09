---
name: a-silent-slot-is-what-an-underprice-looks-like
description: To decide "is the build weak or is the rating under-pricing it", count CASTS — an under-price is silence, and a slot firing in the thousands refuses the repricing by measurement
metadata:
  type: feedback
---

To settle *is this build weak, or does the rating under-price what it holds*,
**count the casts**. An under-priced option shows up as **silence**: the rating
never chooses it, so the slot is dead weight. A slot that fires is a slot the
rating already wants.

**Why:** `DAT-013` (hexarena) suspected the rating under-priced control because
`cleffa.hex` lost four battles in five with four control slots. A repricing was
on the table, and its blast radius is enormous — `price.go`'s `inflictedOn` arms
are on the path of every status in the book — against a rekit, which moves one
JSON entry. The census settled it in one run: **76.0% / 75.1% / 75.5% of all
casts were control**, `charm` was the most-cast skill on every board, and not one
of the four slots was silent. The verdict was the kit, and the repricing was
**refused by measurement rather than deferred** — a much stronger outcome than
"we decided not to".

The contrast is what makes the number legible, and it arrived free in the same
sweep: `rapid_spin` as a candidate in that slot read **93 casts of 17591 (0.5%)**.
*That* is what an under-priced slot looks like. Quote a near-silent row beside
the live ones or the reader has no scale for "76% is a lot".

**How to apply:** before proposing any rating/pricing change to explain a weak
build, drain the event log and tally `battle.SkillUsed` per skill for the subject
seat. Two traps:

- **Take the census on the board the finding is about.** `forge.Census` stands
  the build in a *shipped squad's* first slot against that squad intact — a
  different board from a test fixture's shell, and RAT-006's "the board is the
  finding inside the finding" applies. The fixture-board census is the
  load-bearing one; agreeing with the shipped-squad one is a bonus, not a
  substitute.
- **The actor id is side-prefixed.** `placement.Take` builds `ally.third` /
  `enemy.third`, and both squads in a both-ways-round fixture carry a unit
  literally named `third`, so tallying on `event.Actor == "third"` counts
  nothing and tallying without the side counts **both squads**.

Related: [[a-null-needs-two-controls]] — the near-silent row is the calibration
arm here. [[measure-the-term-before-optimising-it]].
