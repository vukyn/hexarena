---
name: an-uncommitted-harness-leaves-only-a-shape
description: A recorded measurement whose harness was never committed cannot be reproduced body-for-body — rebuild the PROTOCOL over several fixtures and separate the shape (the engine's) from the magnitude (the squad's)
metadata:
  type: feedback
---

A balance figure in `docs/balance.md` is only reproducible if the fixture that
produced it is in the repository. ENG-013's baseline was recorded as `split` cast
**400 / 400 / 0** at three, four and five a side. `git show --stat` on the PR that
wrote it shipped docs, `TODO.md` and two tests — **no harness**, and no squad
named anywhere.

Rebuilt on the stated protocol (same bodies at every size, mirrored, 100 seeds
each way, larger squads = the small one with a body *added*) over **three**
independent fixtures, the readings came to 462/704/0, 464/756/0 and 410/614/0.
Every one reproduced the recorded **shape** — a cliff to nought at exactly the
cap and nowhere else, arrivals equal to casts, mirror 500‰, endless 0 of 200 —
and none reproduced the recorded **400**.

**Why:** the magnitude is a property of the squad (which bodies, which slots, how
long they live), the shape is a property of the engine. Only the second is what a
finding is about.

**How to apply.**

- ⚠️ **Do not stop on a magnitude mismatch when the instruction says "reproduce
  the baseline or stop".** Ask what the baseline is *evidence for*. Here it was
  "the cliff is at the cap", and three fixtures said so more strongly than one.
  Establishing your own baseline and running the control on the **same**
  instrument restores comparability, which is the reason the gate exists.
- **Run the control.** Re-taking the item's own cap-7 probe on the rebuilt
  instrument is what turns a reconstruction into a measurement: 5v5 went 0 → 752
  / 738 / 778 on the three fixtures with everything else unmoved. That, not the
  400, is what says the cap is the knob.
- **Several fixtures beat one**, and cost nothing here: one reading was ~7s, so
  three fixtures × three sizes × two caps was under a minute. A single fixture
  cannot tell you which of its numbers are the engine's.
- **Commit the harness or expect to rebuild it.** If it genuinely must not
  survive (this one was deleted before the PR), write the *fixture* into the doc
  beside the numbers — the protocol sentence alone was not enough to reproduce a
  figure a year-old PR treated as decisive.
- ⚠️ Quote the endless count beside every rate, **including the zeros**. Eighteen
  readings at 500‰ / 0-of-200 is what let "nothing stopped resolving" be a
  statement rather than an impression.

See [[a-two-point-reading-names-no-distance]] and
[[the-fixture-decides-what-a-suite-can-see]].
