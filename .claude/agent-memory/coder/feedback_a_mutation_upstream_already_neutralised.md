---
name: a-mutation-upstream-already-neutralised
description: A mutation can compile, run and be unobservable because something UPSTREAM already made its line moot — the room's cursor was already at the end, and Deliver's second guard already refused; measured twice in one PR
metadata:
  type: feedback
---

Mutate at the moment the state is actually wrong, not at the API the bug is
*named* at. A mutation that compiles and runs still proves nothing if something
upstream has already made its effect unreachable in the fixture — and both times
this happened, the green run looked exactly like a passing test protecting me.

**Why:** measured twice in the `feat/room-spectator-read` PR (2026-09-07),
`internal/room`.

1. **The spec's own mutation was a no-op.** The brief said a watcher read written
   against `r.cursor` would starve the players, so redden the test by making
   `Room.Since` advance `r.cursor`. It did nothing: `resolved()` and `begin()`
   both leave `r.cursor` at `fight.Recorded()` before they return, so **whenever
   control is outside the room the cursor is already at the end** and a second
   read of the battle takes nothing off anybody. The starvation is real but it
   lives *inside* `resolved`, between `settle()` and the players' own
   `r.fight.Since(r.cursor)`. Moving the mutation there reddened at once, with
   digest `e3b0c44298fc` — the SHA-256 of **no bytes**, which is what a digest of
   an empty event run is and is the tell that the players got nothing.
   (`r.cursor = 0` inside `Since` also reddens, which is what proves the test is
   a live net for "a watcher's read disturbs the room's read position" rather
   than only for the resolved-side shape.)
2. **A second guard masked the first.** `Deliver` refuses an unseated sender,
   then `answerFrom` refuses a seat that is not the one being asked. Deleting the
   *first* guard changed nothing, because in the fixture the guest happened to be
   on turn and the second guard refused the impersonated host anyway. The table
   had to be driven **once with each seat on turn**, with a count that fails if
   either arrangement never came up, before the mutation went red.

**How to apply:** before believing a mutation, ask what state the mutated line
reads and whether the fixture ever reaches it *in that state*. Two cheap probes:
if the mutated value is a cursor or an index, print where it stands at the moment
of the call; if the mutated line is a guard, check whether a later guard answers
the same case. And when a table sweeps a role (a seat, a side, a phase), count
the roles observed and `t.Fatal` when one never came up — that is the
[[fixture-hidden-branch]] rule applied to the mutation rather than to the code.

Related: [[feedback_measure_which_guard_masks]] — the same shape read from the
other end, where the guard I blamed turned out to be innocent.
