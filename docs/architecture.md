# Architecture

The runtime pieces of hexarena, in the order a reader meets them: the clients,
the room, the transport, the event stream, and how a battle ends.

⚠️ **This was `CLAUDE.md` until 2026-09-05 and it moved for size, not for
importance** — that file is loaded whole at the start of every session and this
is subject matter rather than a rule that binds every edit. The rules these
sections rest on stayed behind: `CLAUDE.md` § *The layer rule* is still where the
engine's purity and the renderer-cannot-disagree property are stated, and
§ *Invariants worth knowing before editing* is still what an edit may not break.
**Read this before touching `internal/room`, `internal/socket`, `internal/screen`
or the event stream.**

## Two full-screen clients over one `internal/screen`

`cmd/hexforge-tui` authors the cast and **`cmd/hexarena-tui`** plays the game,
and they draw the same screens out of the same package. That is what the eleven
extraction steps were for, and it is the same shape the authoring pair already
had one layer down — a CLI for pipes, a `-tui` for the screen, one `internal/`
package under both, so the two cannot disagree.

⚠️ **`cmd/hexarena` is not that client and must not become one.** It is the
**verification contract**: `--replay --verify` re-runs a log from its seed and
checks every event, and `--auto` / `--log` are the scriptable half. Nothing about
it changed, and nothing about it may.

**What the game client offers**: a menu, the seven catalogues a reader wants
(characters, skills, elements, traits, species, works, squads), a battle and a
**room to join** — the ninth entry, which is the lobby the PvP work landed
in — plus three screens reached by a keystroke rather than by the menu — the affinity chart
(`g` on the elements listing), the statuses reference (`?` on the traits listing)
and the description screen (`?` from three places). The build catalogue and the
art preview are the two `internal/screen` owns that it does not reach; both are
decisions filed in `TODO.md` rather than wiring nobody got to.

### One capability, because three shared screens author and one client cannot

`skills` has `a` and `e`, `origins` has `a`, and the squad catalogue's `n`,
`enter` and `d` reach the two depths that write `squads.json` and the deletion
that removes a side. A game client offers none of that.

`screen.Context.Authoring` is the whole answer, consulted **beside** the
keystroke it turns off so the keys and the footer are decided in one place.
`Context.Footer(authoring, reading)` is the footer half, and a read-only footer
is a **second wording** rather than the authoring one with a clause deleted by a
program: dropping `a thêm` out of a rendered line would leave the separators
either side of it, and nothing would measure what was left.

⚠️ **Nought is the read-only reading, and that is the load-bearing half.** The
safe answer has to be the one a forgotten declaration falls into, and here the
safe answer is *fewer* keys: a read-only client that quietly authored would write
the author's files off a key its footer never named, while a tool that quietly
stopped authoring loses `a`, `e`, `n`, `d` and `enter`. So the tool that can
author declares it (`model.ctx`, `newModel`, and both test fixtures in
`internal/screen`); the client that cannot has nothing to declare and therefore
nothing to forget.

⚠️ **Which suite catches a dropped declaration was measured, and it is not the
authoring client's.** Inverting one guard at a time and counting failing tests:
`skills.go`'s `a` reddens **internal/screen 4 · cmd/hexarena-tui 2 ·
cmd/hexforge-tui 0**; `squads.go`'s `n` reddens **internal/screen 10 ·
cmd/hexarena-tui 2** and cmd/hexforge-tui as well. The nought is that client's
`everyScreen` setting `SkillsScreen.Adding` **by hand** rather than pressing `a`,
so nothing in that package drives the key at all. The authoring half of these
guards is therefore held in `internal/screen` and the read-only half in
`cmd/hexarena-tui`, and neither client's suite alone covers both.

⚠️ **A key announced on a screen that ignores it is worse than one nobody was
told about** (`internal/screen/picker.go`), so the footers are measured rather
than reasoned about. `cmd/hexarena-tui/readonly_test.go` is four tests because
the claim has four independent halves: the keys do nothing (driven through the
real model), no footer the client draws names one, **the list of such keys is
derived** — every keystroke the suite can send, pressed under both readings, and
held equal to the written list in both directions — and the two footers differ,
because a read-only footer identical to the authoring one would satisfy the
second test by naming nothing at all.

### `draw.Fight` means two different things, and that is the PvP seam

`internal/screen/squads.go` raises `Action{Kind: Raise, Target: Fight}` about a
squad **by id**, and the catalogue is in the shared package — so both clients
receive it and each turns it into one of its own screens.
`TestEveryRaiseTargetNamesAScreenInThisClient` is held total **per client**,
which is what makes a Target sayable for a screen this package will never draw.

- In **cmd/hexforge-tui** it means *pick a second squad and measure the pairing*:
  two choosers, both arrangements over the same seeds, a win rate with a control
  that is exactly 500‰ by construction.
- In **cmd/hexarena-tui** it means *take this side into a battle*. The named
  squad is `Home` — the side the player fights on, since `Squad.Take` fields home
  as the ally half — and the opponent is `Away`.

⚠️ **The opponent WAS the seam, and the seam has now been crossed — but not
where that sentence predicted.** `cmd/hexarena-tui/pairing.go` still takes **the
next side on the file, wrapping**, which is one side against a copy of itself
when the catalogue holds one, the pairing every fixture in `internal/screen`
opens on and the one the authoring tool's fight calls its control. What is no
longer true is the claim that `pairing` is the only thing the server replaces:
three quarters of it was wrong and the file now says so. On the network path the
away side is **never a `placement.Squad` on this client** — it arrives already
resolved as `[]battle.Roster` on a `wire.Start`, because `Squad.Take` is the
*server's* call at the gate — `enter` does not hand two squads to `Open` (a live
battle is `Attach`ed), and the battle screen **does** learn the difference, in
one field: `draw.PlayScreen.Live`. What survived is that `landSquad` still
records which side the reader chose, and that answer now fills
`wire.Hello.Squad`.

⚠️ **A mirror was the obvious answer and is measurably worse.** Two identical
sides make the halves of a battle interchangeable — same board, same roster, same
order line whichever way round they are fielded — so *nothing* can see a client
that opens the battle with the sides swapped. The fixture therefore builds its
two sides around **different characters**, and
`TestABattleOpensOnTheSideTheRaiseNamed` walks **every** row of the catalogue and
names both halves, because home alone is satisfied by a client that fields the
named side twice and a cursor on the last row cannot tell `+1` from correct.

### The countdown, and the chooser's third arm: one clock, two features

**⚠️ The countdown needed no message, and the item that asked for one was
wrong.** `TODO.md` called for "a remaining duration **on the wire**"; it was
written before the mirror had its present shape, and the mirror makes it
unnecessary — both peers apply the same `wire.Turn` and open the same prompt, so
**both clients already know, locally, the moment a turn opened and whose it is**,
and `Welcome.Allowance` is already known to both. So each client counts down for
whichever seat is on turn: no new kind, no `KindCount` change, no byte of
`messages.golden` moved. The reason that item gave for a *duration rather than a
deadline* — two machines on a LAN have no reason to agree what time it is — is
exactly right, and it is **why** counting from a locally observed event is the
correct shape rather than a compromise. The two displays drift by the network hop
and by when each client processed the event, which is affordable because **the
display is advisory and the room's timer is authoritative**: a client whose
countdown is wrong still learns the real outcome, because a timeout arrives as a
pass event like any other.

- **`internal/screen` stays clockless and the seconds are handed in already
  counted** (`draw.PlayLive.Clock`, a `PlayClock` of two `int`s). That is the
  arrangement the room is already under one layer down — `Allowance` is a number
  the room *carries* and hands to its clients, and the transport counts it down —
  applied one layer further out. Seconds rather than a `time.Duration` for
  `wire.Welcome.Allowance`'s own reason.
- **`PlayClock.Waiting`'s nought is nobody**, so a screen handed a zero value
  draws no clock: between turns, before the first one, past the cap, and on every
  hot-seat battle there is.
- ⚠️ **It is drawn on the heading row rather than on a row of its own**, and that
  is the budget rather than taste. The log takes every row nothing above it
  claimed, so a new row costs a line of history *and* moves the frame the history
  is read through: measured, three lines of every live render instead of one. The
  heading is where a free reading goes on this screen, which is what
  `logPosition` is already there under. → `TODO.md`, *re-take `playFit`'s budget*.
- ⚠️ **`socket.Sight` gained `Capped`** because the countdown reads the open
  prompt off the battle rather than off `Asking` — `Asking` is nil on the other
  player's turn, which is the turn this feature exists to draw — and a capped
  battle is the one state where the battle still holds a prompt **nobody is being
  asked about**.

**The chooser's third arm is the same clock, which is why they landed together.**
A peer that dies while this client is being asked cannot unblock the chooser:
`Play` is inside `Decide` at that moment rather than inside `conn.read`. The arm
is a timer of `Welcome.Allowance` plus `chooserGrace`, after which the chooser
**passes**.
- ⚠️ **The grace exists so this client is the SECOND to give up.** It starts
  counting a hop after the room does, so the grace only covers clock-rate drift
  and a coarse timer; two seconds is three orders of magnitude over the drift and
  ~2% of the default allowance. The race is already designed for — `Room.TimedOut`
  and `Room.Deliver` refuse a seat they are not asking.
- ⚠️ **It closes a second hole nobody had written down**: a player who simply
  never answers stranded their own client, because the room's pass for that seat
  arrived at a socket nobody was reading.
- **The whole of this client's clock is `cmd/hexarena-tui/clock.go`**, one file,
  which is the entry the module-wide allowlist names.

## The room: a state machine with no I/O, and no clock either

`internal/room` is a PvP match as a state machine over `internal/wire`: messages
and prompts in, messages and decisions out. It declares **no message of its own**
— the protocol is `wire`'s and the room speaks it. Four inputs, each answering
`([]Outbound, error)`:

| input | what it is |
|---|---|
| `Join(hello)` | the gate; also returns the seat, zero on a refusal |
| `Deliver(seat, body)` | a `wire.Act` or a `wire.Pass` from a seated peer |
| `TimedOut(seat)` | the transport reporting that an allowance ran out |
| `Left(seat)` | the transport reporting that a peer went away |

and three readings a transport needs: `Awaiting()` (the seat whose answer is due,
which is what an allowance is started on), `Result()` / `Finished()`, and
`Played()`.

**⚠️ A timeout is an INPUT, not a reading, and that is the load-bearing shape.**
The room never asks what time it is; whoever owns the transport owns the
countdown and *tells* it. So `time` is unimported and
`TestTheRoomReadsNoClock` holds that with an AST walk over the package's own
directory (the shape `internal/wire` and both TUI clients already use). It also
means a whole bo3 plays out in-process in 40 ms rather than in real seconds, and
a PvP log stays exactly as verifiable as one from a battle nobody was waiting on.

**⚠️ NOBODY FORFEITS: leaving and timing out both only ANNOUNCE.** `TimeoutLimit`,
the per-seat tally of consecutive misses and the three-strike branch are **gone**,
and so is the whole concept — `room.Forfeit` and `VerdictForfeited` with it. What
replaced the verdict is **`VerdictAbandoned`**: a match nobody played out, not a
win, not a draw and not a forfeit, with `Result.Departed` naming who went away
(the field was called `Loser`, and a Loser on a verdict where nobody lost is
exactly the stale wording this file keeps a list of).
- **`TimedOut` stays and still passes the turn.** Only the counting died. Without
  the input a room waits forever on somebody who never answers, and the pass is
  what makes the match progress.
- **The refusal on `TimedOut` matters MORE now, for a different reason.** With no
  tally to protect through the back door, what a refusal protects is the **turn**:
  a report naming the seat that is not on turn would otherwise spend the other
  player's answer for them, entering a real decision into the battle and into the
  log. `TestATimeoutOnNothingIsRefusedAndSpendsNobodysTurn` drives both shapes.
- **The board already carried what the forfeit was pricing**, measured both ways:
  a seat that answers nothing **loses on the board** (the opponent kills the
  passing units — a bo3 ended `won` after 56 timeouts), and if both walk away the
  **turn cap** draws it. → `README.md` § *Nobody forfeits*, which also states the
  accepted cost: a losing player can leave for free, and on a LAN the enforcement
  is social.
- ⚠️ **Two good tests were DELETED rather than rewritten** —
  `TestThreeConsecutiveTimeoutsForfeitAndAFourthIsNotNeeded` and
  `TestARealActionResetsTheTimeoutCount` measured a mechanism that no longer
  exists, and a note at the head of `timeout_test.go` says so, because a rewritten
  one would have been a test whose name described a rule and whose body measured
  nothing.

**⚠️ A timeout needs NO message and a departure DOES, and the asymmetry is the
whole design.** A timeout is already on the wire: the pass carries
`room.TimeoutReason`, that is part of the `battle.Decision`, and the decision
rides on `wire.Turn` — so the mirror is told by the one declaration of it that
travels. `TestATimeoutTellsTheMirrorWithNoMessageOfItsOwn` **encodes and decodes**
the room's answer and reads the reason out of the far end, because
`Decision.Reason` is tagged `json:"reason,omitempty"` and that is precisely the
sort of declaration such a claim can be wrong about. A **departure** is the one
ending a mirror cannot reach: no `Ended` for the battle in progress and no further
`Start`, so a peer handed nothing hangs on its own open prompt. Hence
**`wire.Closed{Reason: wire.Closure}`** — the eighth kind — addressed to the seat
**still there** and to nothing else (the transport has already decided there is
nobody at the other one). One value today, `ClosureLeft`, with `ClosureNone` at
zero for the reason `CodeNone` is; a second reason is an **entry** in that enum
rather than a new kind, and `ClosureCount` + `TestEveryClosureHasANameAndTravels`
hold it the way `Kind` and `Code` are held.

⚠️ `internal/wire/clock_test.go`'s own comment says a room "does need a clock"
and that a copy of the ban there "would be exactly wrong". **That expectation was
wrong**; the comment is stale and the ban is inherited rather than escaped.

**⚠️ `Drain` is never called here, and a walk says so.** `TestNothingHereDrainsTheBattle`
looks for the *selector* rather than the string, because `Drain` is what every
other consumer of a battle in this repository calls — 261 sites — so reaching for
it is one keystroke, and it would silently take the events another consumer was
about to read. The room holds one cursor into `Battle.Since`; the point of
reading it that way is that a log writer, a spectator and a reconnect need no
change here.

**The order in `resolved` is what keeps the mirror's digest equal to the room's.**
The room advances to the next open turn **before** reading its cursor, because
that is exactly what a mirror does — `Replay` with one decision and a nil
fallback applies it, walks through whatever is forced after it, and stops on the
prompt it cannot decide. Both event runs therefore hold the decision, then every
skipped turn, then the next turn's opening. Reading the cursor a step earlier
makes **every** digest disagree while both peers are fighting the same battle
perfectly; that mutation reddens the headline test on turn 1.

**The turn cap is checked where the room would otherwise ASK somebody** — after
the skipped test, never before it — for the same reason. A mirror only stops at a
turn it is asked to decide. Skipped turns still count towards the cap; they
cannot be the turn it bites on.

**⚠️ `TurnCap` is ON THE WIRE, on `wire.Welcome`, and that is what makes a capped
battle visible to a mirror.** It sits beside `Allowance` and the argument is that
field's own: a cap is **room configuration**, not part of the battle. The
allowance is there so a client can count down; the cap is there so a client can
**stop on the same turn**. No message and no `Ended` is needed with it, because
the client is a mirror and reaches the cap by the same arithmetic — every opened
turn emits exactly one `battle.TurnBegan`, so counting those *including the
opening* (the event cursor starts after the opening board, so a client counting
only what arrives on a `Turn` sits a turn behind for the whole battle) gives the
room's own count. `TestTheTurnCapEndsABattleAsADrawTheOutcomeAlreadyHas` asserts
the two counts are **equal** and that each client stopped, and the fixture mirror
fails if the room ever asks past its own cap.
⚠️ **Three alternatives were refused — do not re-raise them**: a constant both
peers read (the host loses the setting and a version skew desyncs silently where
a config field is checked at the handshake), a "battle was capped" message (a
protocol bump *and* a second declaration of how a battle ends), and letting the
engine emit `Ended` at the cap (a cap is a **policy**, not a way a battle can end
— `Stalemate` is real, "somebody decided to stop" is not, and adding it makes
every renderer and `--verify` learn a room's policy). → `README.md` § *The turn
cap travels on `welcome`*.
⚠️ **Measured, not deduced: a capped battle's log DOES verify.** It holds no
`Ended` event at all, so the question was real. Replicating the room's stopping
rule at a cap of 6 on the shipped roster: 44 events, 6 choices, **0** `Ended`, the
last event a `turn_began` — and `--verify`'s own procedure reproduced all 44
exactly. ⚠️ The trap for the log writer is *where it stops*: the record must
include the capped turn's own `turn_began`, because the room advanced into that
turn before deciding not to ask about it and the re-run does too. One event
earlier is 43 recorded against 44 re-run, and `--verify` fails on the count.

**⚠️ Nothing is added to `battle.Outcome`, and a capped battle is not stamped
with one.** A departure, a dropped socket and a refused join are results of the
**match**, so they live in `room.Verdict` — deliberately not called an outcome, so
nobody writes `battle.Outcome(result.Verdict)`. A battle the cap stopped keeps
`Undecided` and sets `BattleResult.Capped`: the engine concluded nothing about it,
a room writing an outcome the engine never produced would be a second reading of
how a battle ends, and the eventual log would fail its own `--verify`. The
standing counts it as the draw it is, which is all "the outcome already carries
the draws" needs to buy.
`TestADepartureAddsNothingToTheBattlesOutcomes` holds `battle.OutcomeCount`
against a **literal 4** — reading the constant and comparing it to itself would
agree with any number at all. ⚠️ It used to have **two** cases, one per route to a
forfeit; the timeout route moved out to
`TestASeatThatNeverAnswersLosesOnTheBoardRatherThanByForfeit`, where the claim is
that the match ends the *ordinary* way.

**The gate's order is part of its answer**: version (protocol before digest,
which `wire.Version.Check` owns) → password → seat → squad. A gate whose order is
untested reports whichever fault it happened to notice first, so
`TestTheGateRefusesInItsOwnOrder` gives it a peer wrong about **two adjacent
things** per case. The squad half is five rules under one `wire.CodeSquadRefused`
— `Squad.Validate`, the format's size, level 60, a **leaf** of the line, then
`Take`, which is already the loadout check.

⚠️ **A leaf is not `Furthest` and not `StageAt` — and the reason first written
down for that was wrong.** `progression.Line.Leaves` / `IsLeaf` were added for
this gate: a leaf is a fact about the *line*, `Furthest` about a *level*. The
claim was that a gate written on `Furthest` would start accepting an unfinished
form the day a stage was authored above the cap. **That day cannot come**:
`Line.Validate` refuses a stage whose `MinLevel` is past the cap, so every stage
of every legal line is reachable there and `Furthest(LevelCap)` **is** the tip of
each arm, by construction. Measured — substituting it inside `IsLeaf` passes all
twenty-one tests over the predicate and its gate, and no test can be written that
it fails.
So what the predicate buys is the **level that is no longer in the question**: the
two answers diverge at every level below the cap, and a caller reaching for
`Furthest` has to supply one. This gate asks only at 60 (it insists on level 60
first), so it is the *next* caller the name protects. The difference that is real
today is that `IsLeaf` **errors** on a name the line does not have rather than
answering false — a typo and a form with something after it are different
mistakes, and this gate's job is to say which rule was broken.

⚠️ **One squad MAY field the same character twice**, decided rather than
overlooked, with the reasoning at `squadIsFieldable`.

**The seed derivation is `sha256(seed ‖ index)` and the obvious reuse was
measured wrong.** `rng.New(Seed + index).Next()` looks like the right move under
this file's rule about not restating arithmetic, and splitmix64 advances by
**adding a constant** — so one round of it is a function of the sum, and battle
two of a match seeded 6 *is* battle one of a match seeded 7. Exactly, for every
adjacent pair. Every counter-based generator has that shape: a derivation from
two numbers needs a function of two numbers.

**⚠️ The two things the shipped protocol could not say are both closed**, and
the entry for each is above: a departure now sends `wire.Closed`, and a capped
battle is reached by both peers off `wire.Welcome.TurnCap`. What is left unsaid is
deliberate — a match played out to its end sends nothing, because the client
computes that ending from its own `Ended` and `Welcome.Battles`.

### Many rooms: `room.Registry`, and the two things holding the invariant

`room.Registry` is the concurrency around the room and **nothing else** — no
socket, no clock, no log writer, no spectator. Keyed by `wire.RoomCode`: `Open`,
which takes an **address** and returns the code it allocated, then
`Join` · `Deliver` · `TimedOut` · `Left` each answering `(Answer, error)`,
plus `Read` (the room's own accessors, taken inside its goroutine) and
`Close` / `CloseAll` / `Wait` / `Count` / `Running` / `Codes`.

**⚠️ It is in `internal/room` and that is not a free choice.** Two mechanical
reasons: `TestTheRoomReadsNoClock` walks the package's **own directory**, so a
registry beside the room inherits the clock ban rather than needing a second copy
of it; and the `*Room` never has to leave the package, which is what lets the
invariant below be enforced instead of asked for. **The registry reads no clock
either** — `TimedOut` is forwarded exactly as the room takes it, because whoever
owns the transport owns the countdown.

**⚠️ A request on the channel is a VALUE, never a `func(*Room)`.** A closure is
the tidier-looking design and it defeats the whole thing: the caller captures the
pointer and may keep it, so the battle is reachable from a goroutine that is not
its own and nothing about the code looks wrong. A small discriminated `request`
travels instead, and `answerFrom` is the **one** place a `*Room` is called.

**⚠️ The mutex guards the map and NOTHING else.** `lookup` is the one place a
request path locks and it releases **before** `ask` sends to the room it found. A
mutex held across the send keeps the letter of "one goroutine per room" while
making N rooms as slow as one — and that failure is invisible to every test *and*
to the race detector, so it is held mechanically:

- `TestNoLockingFunctionSendsOnAChannel` is an AST **reachability** walk (send
  marks propagated to callers to a fixed point), because the mutation that hides
  from a per-function check is a locker that merely *calls* the sender. A `go`
  statement is deliberately not an edge: starting a goroutine blocks nobody.
  Measured — holding the lock across the send reddens it by name in half a second
  **and** deadlocks the in-flight test, since a retiring room needs the mutex the
  blocked sender is holding.
- `TestNoRoomMethodTouchesTheMutex` refuses a mutex, a channel or a goroutine on
  any method of `Room`, **by receiver**. ⚠️ It replaced `sync` on the clock ban's
  import list, which had to come off: its stated reason was "the registry takes
  the mutex", written when the registry was expected to live in its own package,
  so the ban would have refused the one file it was written to accommodate. The
  receiver check is *stronger* — an import ban cannot say which type may lock.
- `TestTheRegistryHandsOutNoRoom` walks the type graph of every exported
  signature: no `*room.Room` is reachable in or out. It cannot see through an
  interface, which is why the request's doc comment argues the closure point too.

**⚠️ A room retires its own entry the moment its match ends**, so nothing sweeps
and a finished room stops being joinable — and that is what forced the API shape.
A transport asking *afterwards* would be asking about a room that had already gone
and every match's result would be unreachable. Hence `Answer` carries the
`Reading` taken after the input, and `Read` is for looking at a room that is still
running. ⚠️ `wire.Closed` does not change that: it is for the **peer**, on one
ending only, so the transport's own reading of any ending still has to ride on the
answer — the same division `Known` draws between a refusal for the peer and an
answer for the transport. `Wait` **closes nothing**, deliberately: that is what makes "no goroutine
is left behind" a measurement rather than a tidy-up — a leaked room hangs it where
a shutdown call would have cleaned the leak up and reported success. A shutdown is
`CloseAll` then `Wait`, two calls.

**⚠️ A send on a closed channel panics and a second close panics, and both are
reachable here.** So a room's inbox is **never closed** — the escape from a send
nobody will read is the room's `done` channel, closed by the goroutine itself as
its last act — and `quit` is closed only by the `Close` that won the removal from
the map, which makes exactly-once a property of the code rather than an agreement.

**`wire.CodeRoomUnknown` is what an unknown code answers, and this closed a code
that shipped dead**: `gate.go` documents it as *the registry's* refusal and says
no room ever sends one, so before this nothing in the repository sent it at all.
It names **no seat**, for the reason a refusal at the gate names none.

**`make check` runs this package a second time with `-race`** (3.9s against a gate
of about a minute), and **`internal/socket` beside it** (4.7s plain, 6.1s under the
detector, so ~1.4s more). Those two are the whole of the concurrency in the
repository — one goroutine per room here, a reader and a keepalive per connection
plus a timer per prompt there — and a race test nobody runs is not a net.

⚠️ **`Open` takes the ADDRESS and hands the code back**, and that is the
collision behind it being settled: **one listener per process, and
`wire.RoomCode` carries a seventh byte naming the room** — twelve characters, 256
rooms behind one socket. `Open(at netip.AddrPort, config, deps) (wire.RoomCode,
error)`. It used to take the code, which was a decomposition forced by the open
question (allocating a *port* is I/O and the registry has none); an address is a
value and `wire.EncodeRoom` is arithmetic, so the awkwardness went with the
question. A listener per room was refused: a port is a finite OS resource wanting
a firewall hole, one leaks per crashed room, and it conflates a **room** (an
application idea) with a **listener** (an OS one) — so a registry keyed by code
would be shadowed by a second one keyed by port, and socket lifetime would become
room lifetime in the one component that has no I/O so that it can be tested. The
stated cost: **ten characters became twelve and the ten-character claim is
retired**; `messages.golden` did not move, because no message carries a
`RoomCode`. → `README.md` § *A room, and getting into one*.

- **The byte is the LOWEST FREE one, allocated under the map's own mutex.** One
  hold, so picking a free byte and occupying it are one act — two calls would
  leave a window in which a second `Open` takes the same byte and the loser is
  refused a code that was free when it asked. Lowest-free rather than a counter
  is what makes it *nameable*: a closed room gives its byte back and a test can
  say which code the next `Open` will return. `enrol` touches the mutex and sends
  on nothing (the goroutine still starts in `Open`, after it returns), so
  `TestNoLockingFunctionSendsOnAChannel` is satisfied.
- ⚠️ **A duplicate code became impossible by construction**, so
  `TestADuplicateCodeIsRefused` was **deleted** rather than rewritten — the same
  judgement as the two forfeit tests. What replaced it,
  `TestTheRegistryAllocatesTheLowestFreeRoomByte`, keeps the half that mattered:
  a second room behind one address leaves the first **untouched and playable**.
- **A 257th room is an error, not a `wire.Code`.** A joiner is told a room is
  unknown; a host with 256 rooms running is not a joiner and there is no peer to
  tell. What is still refused at the address is an IPv6 or zero `AddrPort`, which
  is `EncodeRoom`'s refusal and not a second copy of it.
- ⚠️ **`net/netip` is imported by `registry.go` and is deliberately NOT on the
  clock ban's import list**, for the reason `crypto/sha256` is not: it is a
  package of *values* — parse, compare, print, open nothing — where `net` and
  `net/http` dial and listen. The day something here wants `net.Listen` it is
  `net` that appears, and `TestTheRoomReadsNoClock` refuses it.

⚠️ **A room code must be CANONICAL, and that is about the map key rather than
pedantry.** Twelve characters carry sixty bits and seven bytes are fifty-six, so
four bits are spare and **sixteen** strings decode to any one room's bytes;
`encoding/base32` has no `Strict()` (unlike `encoding/base64`) and simply ignores
the trailing bits. Measured before and after the widening: `192.168.1.50:9000` had
**4** such strings at six bytes and **16** at seven. Since the registry keys its
map on the string, a joiner pasting a variant would look up a key that is not
there and be told the room is unknown *while the room sat right there* — a
correct-looking refusal, the worst shape a bug has. So `RoomCode.Decode` decodes,
re-encodes and refuses a mismatch, naming the code that does work.
`RoomCode.AddrPort` is the address-half reader on top of it and adds no refusal of
its own. ⚠️ Dropping that check reddens **one test only**
(`TestANonCanonicalRoomCodeIsRefused`) — nothing else in the repository notices —
so it is the whole net for a map key.

## The transport: `internal/socket`, the one boundary the WebSocket crosses

`internal/socket` is the PvP transport: an `http.Handler` around
`room.Registry`, a dialling `Client`, and the **`Mirror`** that client needs in
order to be a client at all. The dependency is **`github.com/coder/websocket`**
and it is confined here — `internal/room/clock_test.go` and
`internal/wire/clock_test.go` each refuse it **by name**.

**⚠️ Not gorilla, and the choice was measured rather than remembered.**
`gorilla/websocket` is not archived and therefore reads as safe: **0 commits since
2025-09**, last release **v1.5.3, June 2024** (27 months), and it pulls
`golang.org/x/net`. `coder/websocket`: **11 commits in the last year, v1.8.15
released 2026-06-15, zero dependencies** (its `go.mod` has no `require` block at
all), `context.Context` on Read and Write, concurrent writes, autobahn. The module
gained **one line** in `go.mod`. ⚠️ It is the continuation of
`nhooyr.io/websocket` and **the version numbers go backwards across the rename**
(the old path's last release is v1.8.17) — that path is a dead end, not a newer
one.
⚠️ **Both ban lists used to name gorilla**, written while the transport was
unbuilt: a ban on a library the module does not depend on can never fire. They
name the library actually used now.

**⚠️ This package owns the clock the room's allowance is enforced by**, which is
the counterpart of the two bans rather than an exemption from them.
`TestTheTransportOwnsTheClockAndPrintsNothing` therefore makes a **positive**
claim — it fails if no file here reads a clock — because a per-package ban cannot
see the countdown being moved into a *fourth* package, and both existing bans
would still pass. `Allowance` is the whole conversion: `Reading.Config.Allowance`
(seconds as an int) into a `time.Duration`, in one place. It is **exported**
because the game client's countdown and its chooser's third arm are the same
number becoming a duration, and a client repeating that arithmetic would be a
second declaration of what the protocol's seconds mean.

**⚠️ The fourth package arrived, and `TestEveryClockInTheModuleIsOnTheAllowlist`
is what that positive claim grew into.** It is here for the reason the other one
is — this is the package that owns the clock, so "and here is everywhere else
that may have one" reads correctly beside it — and it walks the **whole module**
(the shape `internal/i18n`'s `TestNoKeyIsOrphaned` set, dot-directory skip
included, because `.claude/worktrees` holds other checkouts of this repository)
holding every non-test file that can read a clock against a written allowlist
with a reason each. Six entries today: this package's `socket.go`, `table.go`,
`connection.go` and `server.go`, `cmd/hexarena-host/main.go`, and
`cmd/hexarena-tui/clock.go`.
- ⚠️ **It looks for the calls as well as the import, because a file here reads a
  clock without importing one.** `connection.go` takes its write deadline and its
  close handshake through `context.WithTimeout` over a `Timings` somebody else
  built — an import-only walk calls that file clockless. `context.WithDeadline`,
  `tea.Tick`, `tea.Every` and `socket.Allowance` are on the same list.
- **The count is asserted as well as the set**, because a walk that found nothing
  agrees with any allowlist there is.
- `TestTheClocklessPackagesAreStillClockless` names the packages the list is kept
  *for* — `internal/screen` and `internal/core` above all — so a failure says what
  was lost rather than only that a number moved.
- The timer is armed off `room.Reading` and nothing else — the seat is
  `Reading.Awaiting`, the length is the config's — so there is no state here about
  *whose* turn it is that could disagree with the room's.
- **A `Skipped` prompt starts no clock**, and that is a property of the room's own
  loop rather than a rule here. Verified rather than assumed
  (`TestASkippedPromptStartsNoClockOverASocket`): a match at a one-second
  allowance with both clients answering at once loses no turn to the clock, and
  `Reading.Skipped` says the match had skipped turns in it.
- `allowance.generation` makes a stale fire *quiet*; it is **not** what makes a
  late timeout *safe*. That is `Room.TimedOut` refusing a seat it is not asking,
  and it has to be, because the answer and the fire genuinely race.

**⚠️ A LATE TIMEOUT IS NORMAL, AND TREATING IT AS FATAL DROPS A PLAYER FOR
ANSWERING QUICKLY.** The room answers a timeout for a seat it is not asking with
`wire.CodeNotYourTurn`. Two rules follow, and the second is easy to miss:
- **It is not a reason to close anything.** Measured: making it fatal reddens
  **exactly one test in the repository**,
  `TestALateTimeoutIsRefusedWithoutDroppingAnybody`, which is therefore the whole
  net.
- **The refusal is not forwarded.** The transport owns the timeout, so it owns the
  answer to it; a `wire.Refused` the client never provoked would be a refusal of a
  question it never asked. The same division is drawn twice more:
  `wire.CodeRoomUnknown` is forwarded to a **joiner** (it is the registry's refusal
  for one) and **not** to a seated peer whose room has since ended, because that
  peer's room existed and finished.
- `table.late` is a **count** for the reason `Room.Skipped` is one: the path leaves
  no other trace — nothing is sent and nothing is closed — so without it a test
  asserting it would pass on a run where no timer ever fired late.

**⚠️ `DefaultCloseThreshold` = 60s is a real setting guarding a whole match**, not
a ping timeout. There is no rejoin, so a socket closing is a *match* ending
(`VerdictAbandoned`), and this is the only dial. Two bounds picked it: **generous
against a hiccup** (a LAN wifi roam is seconds; TCP retransmission rides out tens
of seconds without the socket noticing) and **under the 90-second allowance**, so a
machine that dies mid-turn ends the match as abandoned rather than grinding out one
timeout a turn until the board kills the passing units.
⚠️ It governs **only a peer that has gone silent and unresponsive** — a process that
exits sends a FIN and the read fails at once. Which is why liveness is a **ping**
(`DefaultKeepalive`, 15s) and there is **no read deadline anywhere**: a player
thinking about a turn sends nothing for up to the whole allowance, so a deadline on
a read would drop somebody for concentrating.

**⚠️ Nothing here prints, and the reason is the password.** The first message on
every connection is a `wire.Hello`, which carries the room's password in the clear.
A hello that **decodes** is safe by the type (`fmt` calls a field's own `String`);
one that **does not** is bytes with no type left to redact, and json's errors quote
what they choked on. So `errUnreadable` is a sentinel carrying a **byte count** and
never the decoder's error, `log` / `log/slog` / `os` are import-banned, `fmt`'s
printing verbs are refused **by selector** (an import ban cannot tell `fmt.Errorf`
from `fmt.Println`), and the only output is the caller's `Options.Report`, which
takes an `error`. Two tests, because neither half suffices: the AST walk cannot see
a password handed to `Report`, and `TestAWrongPasswordIsRefusedAndNeverPrinted` —
which sends a **malformed hello whose bytes hold the password** — cannot see a
print nothing happened to reach on the day it ran.

**A connection finds its room in the URL path** (`/room/{code}`, one spelling in
`RoomPath`), and no message body carries a code: a code is what a person *pastes to
connect*, so it is addressing rather than protocol content — which is why widening
it to twelve characters moved no byte of `messages.golden`.
⚠️ **`roomOf` decodes and RE-ENCODES**, and that is about the map key. `Decode`
upper-cases first because the alphabet is upper-case only and the fold is total, so
a lower-case code is a good code — but every key in the registry's map came out of
`EncodeRoom`, so without the re-encoding a player who typed theirs in lower case
would be told the room is unknown *while the room sat right there*. An
**undecodable** code is deliberately **not** refused here: it goes to the registry
as it stands, is the key of no room, and answers `wire.CodeRoomUnknown` — the one
declaration of that refusal, which the map cannot get wrong because it can only
hold strings `EncodeRoom` produced.

**Two locks, two jobs, both stated.** `table.exchange` orders one whole exchange —
ask the room, then write what it answered — so the order messages reach a peer is
the order the room produced them in. ⚠️ **It is per room and must stay per room**,
or it undoes the registry's whole point (N rooms not serialising through one lock).
⚠️ **Without it the *client* would be what kept the order straight**, since a
mirror cannot answer a turn it has not received — an invariant a well-behaved peer
maintains is not an invariant. `connection.writing` is the other: it orders writes
on one connection, which is what makes a close from that connection's own goroutine
safe against a dispatch from somebody else's. Always taken in that order, so there
is no cycle.

**⚠️ The `Mirror` is safe to DRAW while `Play` is stepping it, and that is new.**
It used to say it was not safe for concurrent use and needed not to be — one
client, one connection, one goroutine reading it — and that stopped being true
the moment a terminal drew one. `cmd/hexarena-tui` runs `Play` on its own
goroutine and redraws on bubbletea's, so the battle `Play` is stepping is the
battle the screen is reading. So `Mirror` carries an `RWMutex`: `Receive` takes
the write lock for a whole message, every accessor takes the read lock, and
`Mirror.Read(func(Sight))` is the one safe way for a renderer to look at several
readings at once — a screen that asked for the battle, then the prompt, then the
side would otherwise get three readings of three different moments.
- ⚠️ **`Decide` releases the lock before it calls the chooser**, and *why* that
  matters was got wrong once. It is **not** "two readers cannot both be in" — an
  RWMutex admits several, and holding the lock across `choose` passes a test
  built on that prediction (measured). What breaks is a **writer arriving while
  the chooser waits**: Go queues a waiting writer ahead of new readers, so the
  next `Receive` blocks behind the held read lock and the renderer blocks behind
  the writer — and the renderer is what the player is waiting to see before they
  can answer. `TestDecideDoesNotHoldTheLockAcrossTheChooser` builds that three-way
  and is the only thing that sees it.
- ⚠️ **Nothing a `Sight` hands over may outlive the callback**, and that is held
  by the doc comment and by nothing else — a `*battle.Battle` is handed out on
  purpose, because a client computes the board by computing the battle, so no
  type walk can refuse one. Same rule as `room.request`'s closure argument.
- **`ClientOptions.Stepped`** is the redraw hook, called on the `Play` goroutine
  after every message that loop takes in, **with no lock held** and handed
  **nothing**. A chooser alone cannot drive a screen: it only fires on this
  client's turns, so a board driven by one would sit still for the whole of the
  opponent's — up to a ninety-second allowance of a screen that looks frozen.
  ⚠️ It is *not* called for the welcome, which arrives during the handshake
  before `Play` exists; a screen has the room's format the moment `Dial` returns.

**⚠️ The `Mirror` is production code in this package, and that is a decision about
the protocol.** It is filed in `TODO.md` under *the client*, and it is here because
**nothing on the wire says whose turn it is**: `wire.Turn` carries a decision and a
digest, and `Mirror.Asking` — the prompt this client's own battle stopped on,
naming a unit on the side this client plays — is the only derivation there is. So
**no client can be thinner than a mirror**, an end-to-end test cannot exist without
one, and writing it as a fixture to promote later would be writing it twice.
- **`Mirror.Decide` applies nothing.** A mirror steps its battle from the
  `wire.Turn` that comes back rather than from its own input, which is why the room
  sends every turn to both clients including the one that asked.
- **`Mirror.Over` re-derives the series rule the room also has**, and that is the
  mirror contract rather than a duplication: there is no series-standing message,
  so the client learns each battle's outcome from its own `Ended` and the length
  from `Welcome.Battles`. Same shape as `Welcome.TurnCap`.
- **`Mirror.limit` comes off `Welcome.TurnCap`** rather than being a number of this
  package's own — no battle can open more turns than the cap, so a limit of the cap
  cannot cut a run of skipped turns short and there is nothing for a caller to get
  wrong.
- ⚠️ **`Divergence.Turn` is `Decision.Turn`, the unit's OWN count of its turns**,
  not a position in the battle. A reader who takes it for the latter sees
  `A1 turn 5` before `E1 turn 4` and thinks the report is wrong.
- ⚠️ `Receive` takes **both** a body and a pointer to one, because the two
  producers hand over different things: `room.Outbound` carries values and
  `wire.Decode` hands back pointers. `Room.Deliver` does the same for the same
  reason.

**⚠️ Two things were wrong in the first draft, and the tests are what said so.**
- `DefaultMessageLimit` was a megabyte, on the reasoning that a 5v5 `wire.Start`
  carries the whole resolved roster and would approach the library's own 32 KiB
  default. **Measured: 2,911 bytes** over ten units — the default would have done,
  and a megabyte was 360× more allocation than a peer should be able to ask for.
  It is 64 KiB, and `TestTheLargestStartFitsTheMessageLimit` holds **both** ends,
  because no headroom and nothing but headroom are both worth failing on.
- `ended()` did not know `net.ErrClosed`, so a client that closed its **own**
  connection reported `use of closed network connection` as a failure of the match
  it had just left. ⚠️ `context.DeadlineExceeded` is deliberately **not** on that
  list: the only deadline here is the write timeout, so exceeding one is a peer
  that has stopped reading — exactly what the error sink is for.

**⚠️ Two fixture rules, one of which has been retired.** A test may now read a
client's `Mirror` from another goroutine — the lock is what changed — and the
wrapped chooser stays anyway, because it gives a test a **turn-ordered** signal
rather than merely a safe one: "read the mirror when the third decision is being
taken" is a thing a poll cannot say. The other rule stands: a client returning
from `Play` does **not** mean the server has torn its table down, which is why
the leak check polls.

**⚠️ The tests are IN `package socket`**, the opposite of `internal/room`'s choice,
and the reason is that two claims cannot be produced from outside: a **late
timeout** is a race, so driving it from outside means sleeping and hoping, and
**`table.late`** leaves no other trace. Everything else goes over a real loopback
listener through the exported surface. → the two fixture rules above.

**What is deliberately NOT in here**, so a reader does not go looking: any TUI
screen — the lobby, waiting and result screens are `cmd/hexarena-tui`'s own, and
so is the **countdown**, which is that client counting for itself off the moment
its own mirror opened the turn rather than anything this package tells it — the
wordings (a `wire.Code` and a
`wire.Closure` travel as ids precisely so the sentence lives at the client's far
end — `socket.Refusal` carries the code and words nothing), the seat token and the
rejoin, writing a finished match out as a `battle.Log`, spectators, TLS (→
`README.md` § *Not in the first version*), and **the host binary** — which is
`cmd/hexarena-host` and is built: nothing here opens a listener, picks a port,
decides which address a room code carries, reads a flag or prints a word.

**⚠️ `Server.Shutdown` is the one thing that crossed back, and it had to.**
`http.Server.Shutdown` waits for connections it can still see finish a *request*,
and a WebSocket is **hijacked** — `net/http` handed it over and stopped counting
it — so only this package holds the sockets a clean shutdown has to wait for. It
is **four steps**: tell every peer with `wire.ClosureStopped`, `Registry.CloseAll`,
`Registry.Wait` bounded by the context, then wait for `Tables()` **and**
`Running()` to reach nought.

- **Two calls rather than one**, because `Registry.Wait` *closes nothing* — that
  is what makes it a measurement rather than a tidy-up, and a goroutine left
  behind hangs it instead of being quietly collected.
- **Two readings rather than one**, because a table outlives its match by however
  long two sockets take to close. They are exposed for this.
- ⚠️ **`CloseAll` runs even on a context that is already done.** Only the
  *waiting* is bounded; the refusal names both counts, because "context deadline
  exceeded" alone tells a host nothing it can act on.
- ⚠️ **The notify uses `drop`, not `bye`.** `bye` is the close handshake and waits
  five seconds for an answer a peer that is not reading never sends — and a peer
  that is not reading is exactly what a shutdown must survive. The `wire.Closed`
  has already said why and is flushed first. Measured on the four-connection
  shutdown test: **20.0s with `bye`, 0.01s with `drop`**.
- ⚠️ **`Registry.Wait` blocking is held STRUCTURALLY, not behaviourally**, and the
  test says so in as many words. Step four's poll converges whether or not step
  three waited, so deleting the `Wait` leaves the behavioural test green —
  `TestShutdownClosesEveryRoomAndThenMeasuresThem` walks the AST for the four
  calls in order instead. ⚠️ And the behavioural test's **`CloseAll` coverage had
  to be built**: the first version could not see it, because with no rejoin a
  socket closing *ends a match*, so the notify's own dropped connections retired
  every room without anybody asking. It takes a room **nobody joined** to measure
  `CloseAll` at all.

## The event log is the contract

`battle` emits `[]Event`. Anything that draws a battle reads that and nothing
else.

`internal/tui` is the reference: `Line`, `Summary`, `Tallies`, `TagsFromLog` and
`NamesFromLog` all take events and never touch a `*Battle`. **If a renderer needs
something the log cannot supply, add it to the log.** Reaching into the battle to
fill the gap is how a renderer becomes a second copy of the rules that then drifts
from the first.

Two tests hold the line: `TestEveryEventKindIsReachable` fails if a declared kind
is never emitted by any real battle, and `TestEveryEventKindRenders` fails if a
kind falls through the renderer's default case.

**A name on a log line is a caller-supplied MAP, never a `Lang`.** `tui.Line` and
`tui.Log` take `glosses map[string]string` — data id → Vietnamese name — beside
`tags`, and put it in brackets after the id at **every** occurrence
(`uses venoshock (độc kích) at 3,1`). The map is the shape it is because of the
package doc above: an event line is built from the event alone, so this package may
not be handed books, a battle or a language, and `tags` and `Summary`'s `names` are
already the same shape. **A nil or empty map reproduces the line byte for byte** —
that is what English is, what a replay drawn without books is, and the property
`opening.golden` holds. The bracket has **one** definition, `i18n.GlossBracket`;
do not re-spell `"%s <%s>"` in `tui`.
⚠️ **The gloss is written `<>` because round brackets nested.** This log names the
trait a status came from as a parenthetical of its own, so a round gloss inside it
read `(virulence (độc lực))`. The gloss is the *inner* thing wherever the two meet,
so the gloss is what changed shape — everywhere, since it has one definition, so
every reference screen reads `razor_leaf <phi diệp>` too. `<x>` and `(x)` are both
two cells, so **no width measurement moved**, the 79-of-79 `amplified` bound
included. Five hardcoded literals in two test files spell the punctuation out and
are the only things that had to move with it; **no golden holds a gloss**, which
`git diff --stat -- '*.golden'` proves in one line.
⚠️ **`Lang.Gloss` cannot name a skill and that is the whole reason
`Lang.LogGlosses` exists.** Measured over the shipped books: **43** skills, **0**
of them in `skillGloss`, **43** carrying an authored `name`; **22** statuses, **22**
in `statusGloss`, none with a name field; **11** traits, **0** in a table, **11**
authored. `skillGloss`'s nineteen ids intersect `skills.json` **not at all** —
they are the pre-`name` skills and only `internal/testfixture` still reaches them.
So the three kinds go through three accessors (`SkillName` / `Gloss` /
`PassiveName`), which is also what stops a name authored through hexforge drifting
from the log. **Do not delete `skillGloss`** and **do not build the map off it**:
a test using a fixture skill takes the id-table path and leaves the authored-name
path all 43 shipped skills take completely unexercised, which is why
`TestAShippedSkillIsGlossedFromItsAuthoredName` asserts both halves — the name is
on the line, *and* `Gloss` does not know it.
⚠️ **A colliding id is left out, never picked between.** The map is one namespace
over three books that have none, so nothing can tell which kind an event meant.
`TestNoIDIsGlossedTwice` cannot see this — it holds the compiled tables disjoint
from *each other*, the same within-a-table blind spot the category gloss had, and
`taunt` is a shipped skill id with `taunting` a shipped status id. `LogGlosses`
therefore drops the name (a bare id is this package's declared normal, and a wrong
name is worse than a missing one) and `LogGlossCollisions` is the loud half, over
the shipped books, in `TestNoLogGlossCollidesAcrossKinds`. It is read off the
**id** and not off which kinds offered a name, so a collision cannot go live the
day somebody authors the second name.
⚠️ **The coverage check is a table of ARMS, not of kinds.** `status_resisted` has
four branches and `damaged` two, so a sweep over `battle.KindCount` renders one arm
of each and reports full coverage. `sweepArms` in `internal/tui/gloss_test.go`
tables **16 glossing arms across 12 kinds** with the exact count of ids each names,
plus 13 arms that must stay bare; `glossingArms` is written down rather than counted
off the table, so extending the table is not the same act as claiming the extension
was measured. Adding a branch that prints `event.Skill`, `event.Status` or
`event.Passive` means adding a row.
⚠️ **One arm gave up a word to keep its row on the window, and the margin is now
nought.** `amplified` is the only arm printing two glossed ids in one clause, so
`dragon_drive (long xung) is amplified by expose (phá giáp) x2, power 2300` measured
**82** of the 79 cells there are — a row that fit before glossing (59) and would not
after, on a screen *the budget* measured as having no spare rows. The arm reads
`%s amplified by %s x%d, power %d` now: three cells for the word `is`, in the one
register that can spare it (this log is verb-terse everywhere else — `hits`,
`misses`, `holds`, `lets go of`), which puts the widest reachable row at **79 of
79**. That is the **only** line of English this change touched, and it is why the
en output is otherwise byte-identical rather than entirely so.
⚠️ **Nought margin is deliberate, not lucky, and `knownWide` is empty on purpose.**
The bound is recomputed from the books every run, so the day a longer skill name is
authored the test goes red rather than the log quietly wrapping — which is the whole
value of it being measured rather than declared. `knownWide` stays as the shape for
a breach somebody later decides to accept; an entry in it must carry its measured
figure and its reason.
`TestNoGlossedLogRowOutgrowsTheWindow` measures **reachable** combinations (a skill
beside *its own* condition's status, a trait beside the statuses it actually names,
a reply only from a trait that replies with damage) rather than the longest name of
each kind in one event — that bound is 102 cells and describes no event any battle
emits, and a bound nothing can reach is a bound nobody will fix.
⚠️ **The `Started` line is out of scope and stays as it is.** It measures **86–87**
cells against 79 on the client's own fixture cast and is already over today; it is
exempt from `TestEveryWordingFitsTheMinimumWidth` because it names a unit, and a
unit name is free text. Nothing on it is glossed — not the side, not the element
note, not the unit name.

**Every FIGURE in a skill description is derived; exactly one clause is authored**
(`internal/i18n/describe.go`, in **both** languages). `Skill.Flavour` opens the
sentence and the numbers are appended. ⚠️ **A flavour clause may not contain a
digit** — `skill.ParseBook` refuses one that does, and that single rule is the
whole of why authored prose is safe here: no number in it, nothing a balance
change can make wrong. Digits rather than a percent sign, because "110" and "gấp
2" are the same mistake. English has none (like `Skill.Name`, it is authored once
in Vietnamese) and falls back to the derived opening.
⚠️ **A clause obeys the same body rule as the name** — `withdraw` was renamed off
"thu mai" because anybody may carry it, and the first clause written for it said
"rụt hết vào trong mai": the same defect through a field the name test does not
look at. `TestAFreeSkillsFlavourNamesNoBodyItMayNotHave` checks an **unrestricted**
skill's clause against a hand-written `bodyWords` list; a restricted skill is
exempt because its restriction guarantees the body. It cannot tell whose shell is
meant, so a body word about anything trips it — reword rather than argue.
`?N` at the battle prompt describes the Nth offered skill, `?TAG` a unit's traits;
both reprint the menu and cost no turn. Every number comes from the skill itself,
because an authored line survives its own numbers moving — "doubles" outlives a
bonus falling 1000 → 700 — and this repo already refused that trade in
`Archetype.Demands`. ⚠️ **Never add a `description` field to skills.json** — `flavour` is one clause,
not a description, and the difference is that it holds no figures.
⚠️ **`passives.json` has `flavour` too, and its ban is stricter.** A skill free for
anybody may not name a body and a *restricted* one may (`ingrain` says roots, only
a plant takes it); **a trait has no restriction mechanism at all** — no element,
archetype, species or character — so for a trait the ban is unconditional and no
future field relaxes it. It is a **lead line**, not a replacement: a skill has one
opening sentence to take over, a trait has one to six lines and no opening among
them.
⚠️ **Trait wordings name the thing, never `nó`.** Six of eleven used to lead with
a bare pronoun (`Nó gây %s mạnh thêm %s`); a description is about one unit, so the
pronoun carried nothing and only lengthened the line. Drop it wherever the meaning
survives — and where a sentence has two subjects, name them: a reply is "ai đánh
trúng thì bị phản lại … công **của người bị đánh**", not two `nó` in one clause.
⚠️ **Trait sentence order is narrative, not field order**: what the holder *is*
(grants, resists) → what its **own attacks** do (applies, amplifies, drains) →
what attacking **it** costs (replies) → **when** (while). Field order read
backwards on `venom_blood`, which replied before it said it was immune.
⚠️ **Two share renderers, on purpose.** `forge.Percent` keeps a tenth (`2.5%`) for
**hexforge's tables**, where an author tunes the number and the tenth is what is
being tuned. `i18n.share` **rounds to a whole percent, half away from zero**, for
**sentences**, because a tenth is precision a player cannot act on and the decimal
point is the only mark in the line that is not a comma. History matters: share
*truncated* first, which turned a trait's 2.5 into 2 (skills are priced in
hundreds of ‰ and lost nothing; traits are priced in tens). Rounding is not that
mistake again — 25 becomes 3.
⚠️ **Rounding is safe because of a rule on the DATA: nothing is ever tuned by
less than 1%** (`TestNoShippedShareIsUnderOnePercent`, over every shipped skill,
trait and status). Nobody feels a share that small across a battle, and a
description of one prints `0%` — a feature reading as broken. So the floor lives
in the data; carrying a tenth in the sentence to survive data the rule forbids
would be the renderer paying for a case that cannot happen.
Numbers are **shares of a stat** ("100% công"), never damage figures: a figure is
true for one caster against one target and false at the next buff. ⚠️ The block is
**Vietnamese on an otherwise English screen** — a stated cost, not an oversight;
translate the whole battle screen in one piece or not at all. It borrows
`Lang.Gloss` rather than growing a second vocabulary (two names for one status is
how they drift) — and in **English a bare id is the name**, so `poison` in an
English sentence is the reading working, not a missing gloss. Two entrances:
`?N` / `?TAG` at the battle prompt, and `?` on the hexforge-tui skill listing,
**cast browser or played battle**, all three of which raise `screenBlurb` — one
screen branching on the **kind of subject it was handed**, describing a skill from
the listing, a character's traits from the browser and the option under the cursor
from the battle. ⚠️ It used to branch on `blurb.from` and read those three screens'
state directly; the raiser pushes a `screen.Subject` now and the describer reaches
for nothing (`cmd/hexforge-tui/describe.go`). A listed skill and a battle option
turned out to be **one** subject rather than two — same id, same position, same
paragraph — so the kinds are three (`StatusSubject`, `SkillSubject`,
`CharacterSubject`) and not the four the raise sites suggest. Both describers
moved into `internal/screen` once they stopped reaching (`screen.BlurbScreen`,
`screen.PreviewScreen`); what stayed in the client is `describe.go`, the applier
that says which of *its* screens a subject lands on and which raiser an arrow key
walks. ⚠️ **`blurbScreen.from` is gone**: it was a `screen`, this binary's own
enum, so it could not travel with the describer, and it survived for as long as
one of the blurb's three raisers still wrote `m.screen` itself. All three return
a `draw.Raise` at `draw.Blurb` through `navigate` now, so `model.raisedFrom` —
the slot the client already keeps for a Back — records who raised it, and its
three readings (the `esc`, and the two branches saying which raiser an arrow key
walks) read that. ⚠️ **`Subject.Kind` could not have replaced it**, which is
worth being exact about: the collapse above measured a listed skill and a battle
option to be *one* subject, so the subject cannot tell the listing from the
battle. A **fourth** reading of a skill sits beside that one and is not it:
`Lang.SummariseSkill` is the compact line the played battle draws on every
option's own row, and *Where a form beats a prompt* says why it cannot be this
one with the prose dropped. ⚠️ **The forge form is not the place for it** — 19
fields already show 13 of themselves in a 120x24 window, so a three-line block
under the form costs a quarter of the fields; a screen costs nothing until asked
for. Statuses are the third description, and `Lang.DescribeStatus` is the same
shape from the same house — see *Looking a status up* under Open work.
`internal/i18n/testdata/describe.golden` covers **every** shipped skill, trait and
status in **both** languages, so a balance change moves a line there — that diff is
how a number change reads to a player. ⚠️ English needs singular wordings where
Vietnamese does not (`BlurbCostCooldownOne`, `BlurbStripsOne`): two keys rather
than a plural rule, because a rule would make Vietnamese pretend it has a
distinction it does not.
⚠️ **A list of three takes one conjunction and commas, and `listed` is the only
place that knows it.** The conjunction alone was put between *every* pair, so
three items read `a and b and c` — not a sentence in either language. The
**two-item case is what hid it**: every list the shipped books produce has one
item or two, so `describe.golden` held **zero** of them and both goldens still
hold zero after the fix, which means no golden could ever have caught this and
`TestAListOfThreeTakesCommasAndOneConjunction` is the whole guard. The only 3+ list
anywhere is the fixture's `purify`, three stripped categories — the same skill that
exposed the strips clause's width, and for the same reason: a fixture reaches what
the shipped data cannot.
**Both languages take the same shape and that was measured rather than assumed** —
`a, b and c` and `a, b và c`: conjunction before the final item, comma between the
rest, and no serial comma in either. So `ListComma` is **one key pair** and the
conjunction stays the caller's, which is what lets `join` (prose, `BlurbAnd`) and
`JoinIDs` (untranslated ids, `ElementJoiner`) keep saying which sort of list they
are while agreeing about the grammar. Two joiners with identical *shape* declared
twice is the drift this file keeps a list of; two joiners with their own
*vocabulary* is the distinction `JoinIDs`' doc comment exists for.
⚠️ **The caller owns the conjunction and `listed` owns the comma**, so joining
items that may themselves hold a comma would build an ambiguous list. Every caller
today joins short noun phrases. The two that join **clauses** pass a slice of at
most two — `ReadsStatus` and `ReadsHealth` are the only conditions there are — so
they never reach the comma at all; a third condition is the site to re-read, not
this function.

Event kinds, sides and outcomes serialise **by name**. Do not change that to a
number: a saved log would silently reinterpret itself the next time a constant was
inserted. Renaming one breaks existing logs, so treat the names as the wire
format.

**`hex.SideNone` is the zero value, and that is load-bearing rather than tidy.**
A side is written with `omitempty`, so whichever side is zero never reaches the
wire — with `SideAlly` at zero, every ally unit's `started` event wrote no side
at all and a reader recovered it from a missing field, correct only because ally
was declared first. A battle with no winner wrote the same thing and meant
something else entirely. Now nothing is at zero: an absent side means nobody, a
won battle names its winner, and a draw names none. `Side.Fights()` is the check
for "is this one of the two", and `battle.New` refuses a roster entry that never
said. ⚠️ **This broke the log format once** — logs written before it fail
`--verify`, because the re-run produces `ally` where the file has nothing. That
was worth doing once, on throwaway files, to stop the format depending on a
coincidence. Do not put a real value back at zero.

**`hex.Cell` is an `Offset` that may be nowhere, and `Event.Cell` /
`Decision.Aim` are the two fields that need it.** The opposite trap to the one
above: `{0,0}` is the ally back corner, so the coordinate has no spare value to
mean "no cell" — and `omitempty` does nothing to a struct field, so every kind
that places nobody wrote the back corner into the log, as did every passed turn's
aim. `omitzero` on the offset only inverts the error. So absence lives in a
second, unexported field, the tag is **`omitzero`** (the repo's first use; go.mod
is `go 1.27.0`), and readers unwrap with the comma-ok `Cell.Offset()`. ⚠️ **It is
a value, not a pointer, and must stay one** — `--verify` and
`internal/seed/battle_test.go` compare whole events with `==`, which a pointer
satisfies **by address** with nothing to compile-fail on, so every log would fail
verification and no test would say why. `Cell.String()` prints `"none"` when
absent, which is what keeps the TUI goldens byte-identical. ⚠️ **This broke the
log format a second time** — a log saved earlier fails `--verify` on its first
event. `Unit.Cell` and `Roster.Slot` stay plain `hex.Offset`: they are always
meaningful and they key nine `map[hex.Offset]` sites.

## How a battle ends, and why a draw is an outcome rather than an error

Every ending comes through one `Ended` event carrying a `battle.Outcome`:
`Victory` (its `Side` names the winner), `Annihilation` (both sides empty) or
`Stalemate`. **Do not read the ending out of a note or out of what stopped
happening** — a renderer and `--verify` both switch on it, and an ending only
English can tell apart is one neither can be trusted with. `Battle.Outcome()`
is the same fact for a caller holding the battle; `Winner()` cannot say which of
the two draws it was.

`Stalemate` exists because **nothing on this board moves**. Reach is fixed when a
unit is enlisted and the enemies worth reaching are not, because they die, so two
short-ranged survivors on opposite back columns is a battle that can never
finish — seed 18 once skipped 3955 of 4000 turns and came back as a battle that
never ended rather than as a result.

Three constraints hold it together, and each is a way it was nearly got wrong:

- **`checkEnd` and `settle` are separate on purpose.** `checkEnd` asks only "is a
  side empty" and is called from `kill`, which happens in the middle of a skill
  still choosing targets. `settle` adds the deadlock test and runs only where a
  turn has finished and the board is at rest — the end of `Act`, the end of
  `Pass`, and Advance's two skipped-turn returns. Asking the deadlock question
  from `kill` would read a board nobody will act from.
- **`frozen` is a pure predicate, not a counter.** For every living unit: nothing
  timed on it (`status.Set.Timed`, which ignores a trait's permanent status) and
  no skill with a legal aim, **cooldowns ignored**. Both halves are pessimistic,
  because a draw declared on a battle that would have resolved is worse than the
  turn limit catching a real runaway. A count of quiet turns would be one more
  thing two runs of the same seed could disagree about.
- **Cooldowns and control are deliberately not read.** They are what make a
  skipped turn ordinary, and not reading them is exactly why an ordinary skipped
  turn cannot be mistaken for a deadlock. A poisoned deadlock is not a deadlock:
  the poison ends the battle by emptying a side.

Note a **self-targeting skill always has an aim**, so a unit that can still buff
itself is never frozen — correctly, since it can still act. A battle of two such
units is a genuine runaway and is what the turn limit is now for; `RunToEnd`
still returns without an error there, and the caller reads `Finished()`.

The cheap half is caught earlier. `battle.New` refuses a roster holding a unit
that can aim at nobody from its slot, and `hexforge check` **warns** — a
`forge.Warning`, not a `forge.Problem`, so the check still passes — when a
character's longest range cannot reach anybody from its archetype's column. Both
are necessary and neither is sufficient: reach shrinks as units die.
`hex.ReachNeeded` is where the column-to-distance answer lives, measured through
`hex.Place` rather than written down.
