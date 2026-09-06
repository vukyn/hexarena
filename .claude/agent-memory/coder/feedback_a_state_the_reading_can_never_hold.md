---
name: a-state-the-reading-can-never-hold
description: A screen state built by hand out of a reading can be one the reading can NEVER hold — derive "has my own decision landed?" from the producer, not from the field that looks like it says so
metadata:
  type: feedback
---

Before registering a screen state that means *"my own decision has gone"*, prove
the reading can reach it. Derive it from the **producer** — what the room records
and when — not from the field that looks like it says so.

**Why:** step 5c's arrange screen first had `Sent() = Live.Arranging &&
!Live.Yours`, which reads exactly right: the phase is open and this seat is no
longer being asked. It is unreachable. `draft.Arrange` records the **first**
arrangement nowhere (appending it would show it to the opponent) and both go in
at once, so a client that has arranged reads what it read before, and the moment
the record moves the phase is closed and `Arranging` is false. The state would
have shipped with a golden entry, a floor test and a translation test — all
passing, none of them ever drawn by anything a player can do. This is
[[fixture-decides-what-is-visible]] with the fixture being *the shape of the
reading itself*: hand-building a value proves the drawing, never the reachability.

Two consequences worth carrying:

- **A decision that moves no reading needs local state or the screen looks
  hung.** Here `ArrangeScreen.Sent` is set by the send. It is display only and
  **gates nothing** — the keys stay live, because a decision the room refuses is
  one to send again, and that retry loop is the only thing that gets a
  turned-away decision through (the memo step 5a refused was a *gate*).
- **The one frame you did not think of is usually the reachable one.** With the
  local flag in, the derived state came back as something else and real: the gap
  between the record closing the phase and the `wire.Start` behind it, where the
  reader is still on the screen with nothing left to do. Two states, two footers,
  two golden entries.

**How to apply:** when a state is *"the other end has taken what I sent"*, open
the producer and find the line that records it. If nothing records it before the
next state, the reading cannot say it. See
[[a-well-formed-measurement-can-measure-nothing]] and
[[fixture-hidden-branch]].
