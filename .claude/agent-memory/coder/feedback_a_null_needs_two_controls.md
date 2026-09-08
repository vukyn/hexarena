---
name: a-null-needs-two-controls
description: A subject arm that moves nothing is only reportable beside BOTH an exact-null control and a payload calibration arm — otherwise "no effect" and "the instrument is blind to this knob" are the same reading
metadata:
  type: feedback
---

When a measured change comes back flat, **flat is not yet a finding**. Two
different controls have to run beside it, and they answer different questions:

- **An exact-null arm** — a change derived to be a no-op — says the harness is
  deterministic and the edit path works. In `DAT-009` this was reshaping `rally`
  onto `arc_up`, derived to catch exactly what `column` catches on `s01`. It
  reproduced the baseline **battle for battle** (same W/L/D, same `Endless`, same
  median turns, same `AsAlly`/`AsEnemy` split, every row). Had it differed by one
  battle, something other than the variable was moving and no subject reading
  would have been interpretable.
- **A payload calibration arm** — a *different* knob on the *same* subject, big
  enough that the harness must see it — says the harness is not simply blind to
  the subject. In `DAT-009` this was leaving `rally` on its shipped shape and
  doubling its `fury` stacks: aggregate −27‰, one board's `AsEnemy` 220‰ → 5‰,
  the mirror's median 78 → 111 turns. So the board plainly prices what `rally`
  *delivers* — and the subject (a second *recipient*) moved one battle in twelve
  hundred.

Only with both does "the second recipient is worth nothing here" beat "this
harness cannot see `rally` at all".

**Why:** without the calibration arm the whole measurement collapses into the
failure mode this repo has hit repeatedly — a green, well-formed run that
measured nothing (`[[a_well_formed_measurement_can_measure_nothing]]`,
`[[normalised_upstream_cannot_discriminate]]`). A null result is the *expensive*
kind of result to publish, because the next reader will re-take it unless the
write-up can show the instrument had its eyes open.

**How to apply:** when a plan hands you a subject arm and a gate, budget a third
and fourth arm before running anything, and write the gate down first. Also state
in the write-up that the calibration arm was **discarded** and never reached the
shipped data — a calibration knob is an instrument, not a candidate. And note the
direction: the calibration arm here moved *downward*, which is a second reading
nobody asked for; report the number and refuse to explain the mechanism you did
not measure.

Related: `[[a_refusal_can_be_right_for_the_wrong_reason]]` — separate the verdict
from its evidence. `DAT-009`'s premise ("move a support column onto `arc_up`")
was wrong and its conclusion ("support area is not worth it here") was right; the
`arc_up` arm is what let both be said in the same entry.
