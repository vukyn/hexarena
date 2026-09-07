---
name: a-reading-called-my-side-can-be-somebody-elses
description: "⚠️ socket.Mirror.side is the HOST's half for a spectator, so `unit.Side != m.side` read every host turn as the watcher's own — guard the ONE derivation (asking), never the readers; plus a bo3 ends at 2-0 and an \"off\" wording nested in the \"on\" wording makes a both-states assertion self-contradict"
metadata:
  node_type: memory
  type: project
---

hexarena, spectator step 5 (2026-09-07). `Mirror.side` is documented as *"the
half of the board **this client** plays"*, and for a watcher it is the **host's**
half — `internal/room` records `wire.Start` with the host's side because a
spectator plays neither and a `Start` with no side would be a `hex.Side` zero
reading as `SideAlly` downstream. So `Mirror.asking`'s
`unit.Side != m.side` answered **true on every one of the host's turns**:
`Client.Play` called the chooser, sent a `wire.Act`, and the room refused it.
Measured with the guard deleted: **119 of a two-battle match's turns**, 138
`not_your_turn` refusals down one socket. Nothing was corrupted — the room's
refusal leaves the prompt open — a spectator was just a machine spamming
refusals at a match it came to watch.

**Why it matters:** a field whose *name* says "mine" is the field a derived
"is this mine?" reaches for, and it can be a fact about somebody else. The whole
suite was green: the watcher's digests all agreed, both peers played the same
match, and the room refused every act correctly.

**How to apply:**

- **Guard the one derivation, not the readers.** `asking()` is where "I am being
  asked" is computed and `Decide`, `Play.answer` and `Sight.Asking` all come
  through it — so one line there covers all three. ⚠️ Do **not** also nil the
  prompt in `screen.PlayScreen.Attach`: two floors for one invariant is the
  mistake on `battle.healingFor`, where deleting either reddened nothing. On the
  screen the consequence (`Pending == nil`) is what every decision key already
  falls past.
- **Ask the existing derivation.** `wire.Welcome.Watching()` (= `!Seat.Valid()`)
  is the single declaration; `Mirror.watching()` is `m.seated && …Watching()`
  and the seated half matters, because the zero `Welcome` names no seat either.
  Never `m.seat == SeatHost`.
- ⚠️ **The premise is the measurement.** "Never asked" is trivially true of a
  client that never saw the host's unit on turn, so the test asserts the count of
  readings where it *was* (76 of 173) before asserting the count of asks is nought.
- ⚠️ **A bo3 ends at 2-0.** `len(Reading.Played) == config.Battles` is wrong on
  every run — `Mirror.Over` stops the series the moment a side is past taking
  back. Assert `>= 2 && <= Battles`, or compare against the clients' own `Fought`.
- ⚠️ **A wording nested inside its opposite makes a both-states assertion
  self-contradict.** `JoinWatchOff` was `"không"` / `"no"` and `JoinWatchOn` was
  `"có — không ngồi ghế nào"` / `"yes — take no seat"`, so a screen showing only
  the ON row *contains* the OFF wording in **both** languages and the "on and off
  at once" check fired. Word the two answers as whole non-nesting phrases.
- ⚠️ **The client has a second, silent net at `session.answer`**: a nil prompt
  drops the keystroke, so a screen mutation that emits an `Answer` on a watching
  screen puts **nothing** on the wire. The discriminating sweep for keys is
  therefore the `internal/screen` one (it asserts the `Action` kind); the client's
  measures reachability and the refusal list.

Related: [[hexarena-draft-and-spectator-plan]] [[hexarena-socket-transport]]
[[hexarena-cursor-record]] [[fixture-hidden-branch]]
[[mutate-the-producer-not-just-the-logic]]
