---
name: draft-state-machine
description: hexarena internal/draft — steps 2a, 2b, 3, 4 and 5a built; the Done()/Picked() split, the arrange phase's own accessors, the record's SHAPE living in internal/wire, the room-side arrange-phase clock (NOT one allowance), and the protocol gap 5a found — NOTHING tells the host the room is full
metadata:
  type: project
---

⚠️ **The full record lives in the repository** — `hexarena/TODO.md` § "Ban and
pick" / "The arrange phase" and `hexarena/memory/hexarena-draft-and-spectator-plan.md`
— because that travels to any machine and this directory does not. This note
keeps only what a coder needs *before* opening those.

`internal/draft` **step 2a** (branch `feat/draft-machine`, PR #308): `New`,
`Turn`, `Ban`/`SkipBan`, `Pick`/`Loadout`, `Candidates`, `TimedOut`,
`Cancelled`, `Picks`, `Since`. Bans-first-then-picks, alternating from
`Config.First`; a pick is two decisions; the timeout is an input and cancels the
whole draft; the record is `battle.Since`'s shape.

**Step 2b** (branch `feat/draft-arrange`, 2026-09-05): `internal/draft/arrange.go`
— `Arrange(seat, slots)` (one seat's **whole** arrangement in one call),
`Arranging`, `AwaitingArrangement`, `Squads`, plus `StepArrange` and
`Entry.Slots`.

- ⚠️ **`Turn()` did not widen and must not.** Two arrangements are pending at
  once; every refusal in the package assumes exactly one open decision, so the
  phase got its own accessors and `Turn` answers `false` from the moment picking
  closes.
- ⚠️ **`Done()` means the WHOLE draft now (picking *and* arrangement); the old
  meaning is `Picked()`.** Seven sites plus the doc comment moved. The reason to
  redefine rather than add a third name: the dangerous reading is the one a
  caller reaches for before fielding a squad, so that name has to be the safe
  one. `due()`'s *"this draft is finished"* refusal had to **split into two
  sentences**, because "the picking is over" is now two states.
- ⚠️ **The two arrangements are buffered and recorded together, in `seats`
  order.** Consequence that reaches a test: the record cannot say who arranged
  first, so a replay reproduces the phase's *end* and not its middle — the
  entry-by-entry replay comparison has to defer exactly one comparison, and
  count the deferral.

⚠️ **The reason not to return slot-less squads is a REFUSAL, not silent
stacking**, and a brief once said the opposite. `hex.Offset{}` is a real cell
(col 0 row 0) and passes every *per-unit* check in `placement.Squad.Validate` —
but Validate also keys a `map[hex.Offset]string` and refuses *"unit %q stands at
0,0, where %q already is"*. So `Squads()` answers two squads with **nobody in
them** until `Done()`, which Validate refuses by its own first line.

**Why:** the difference decides what a doc comment may claim, and "passes Take
and stacks the side on one cell" is checkable and false.

**How to apply:** when a brief explains *why* a shape is dangerous, re-derive
the mechanism against the code before repeating it in a comment. See
[[a-refusal-can-be-right-for-the-wrong-reason]]. What a `Pick` needs to become a
`placement.Placement` is a `Slot` and an `ID` — and the `ID` is the **character
id**, licensed by the pool being exclusive.

**Step 3** (branch `feat/draft-wire`, 2026-09-05) put the record on the wire, and
the one thing to know before editing this package: ⚠️ **`draft.Entry` and
`draft.Step` no longer exist.** The record's shape is `wire.DraftEntry` (a
`Seat` plus an embedded `wire.DraftDecision`) and the vocabulary is
`wire.DraftStep` with `wire.StepBan`… — declared **once, in `internal/wire`**,
with no local alias, because `internal/draft` imports `internal/wire` (for
`Format` and `Seat`) so wire may not import back. `Since`'s answer is therefore
already what `wire.Drafted` carries. The mirror's apply loop is
`Draft.Apply(wire.DraftEntry)`, moved out of `record_test.go`'s local `apply`
exactly as that function's comment predicted. Nothing about the state machine
moved — `Turn` still answers one seat and one step, `Done()` is still the whole
draft.

**Step 4** (branch `feat/draft-room`, 2026-09-06) made a `room.Room` host one:
`Config.Drafts`, the draft built in `room.New`, `internal/room/draft.go` for the
whole room side, `begin()` called **unchanged** on `Done()`. Three things worth
holding before touching it:

- ⚠️ **THE ARRANGE PHASE'S CLOCK IS NOT "ONE ALLOWANCE", AND THE BRIEF SAID IT
  WAS.** `room.Reading{Awaiting, Waiting}` holds **one** seat, `internal/socket`
  arms one timer off it, and the phase has **both** seats pending. It is
  serialised — `Awaiting` answers `AwaitingArrangement()[0]` — and the brief's
  stated consequence was "one allowance covers both sides, so a side that
  arranged promptly gets no fresh clock". **Measured in the consumer: the
  opposite.** `socket.Server.settled` re-arms off the reading after **every**
  batch, so the first arrangement to arrive *is* an exchange and the seat still
  owed gets a **fresh full allowance** from that moment. Worst case ≈ **2×** the
  allowance; a prompt side hands its opponent *more* time. And the brief's second
  consequence ("the seat blamed is the first still to arrange, not the slower
  one") is narrower than stated: `AwaitingArrangement` holds only seats that have
  **not** arranged, so once one side answers the name is exact — the inexactness
  is only the both-silent case, where the host is named by seats order.
- ⚠️ **`draftOpen()` must ask `Finished()` as well as `Cancelled()`.** A draft
  cancels itself only on its own timeout; a **departure** ends the match through
  `abandon` and leaves the draft neither done nor cancelled, so a
  Done-or-Cancelled test alone keeps reporting an open decision after the match
  is over — and `Awaiting` is documented false once it is, with a transport
  arming a countdown on it. It must also ask **both seats taken**: the draft
  exists from `New` and is already due its first ban while the room holds one
  player.
- ⚠️ **`Config.Validate` refuses `Drafts` with `Battles != 1`.** Not in the
  brief: decision (d) is "a ban lasts the match, the first cut is bo1 only", and
  accepting a drafting bo3 would silently pick one of the three games the bo3
  item lists.

**Why:** the clock paragraph is the one a reader will re-derive from the producer
and get backwards, and the two `draftOpen()` clauses are branches whose absence
leaves the suite green in every test that does not end a draft the odd way.

**How to apply:** before writing down what a room value *costs*, read the
consumer that acts on it (`internal/socket/table.go`'s `allowance.set` and
`server.go`'s `settled`) rather than reasoning from the room. See
[[measure-the-thing-a-bound-bounds]].


**Step 5a** (branch `feat/draft-mirror`, 2026-09-06) made a drafting room
joinable by a real `socket.Client`: `internal/socket/draft.go` holds the client's
own `*draft.Draft`, `Sight.Draft` is a **`DraftSight` value snapshot**, and
`ClientOptions` gained `Characters` (the cast the pool is `NewPool`'d from) and
`Draft` (a `DraftChooser`).

- ⚠️ **NOTHING ON THE WIRE TELLS THE HOST THE ROOM IS FULL, and that is the gap
  to know before touching this again.** A client's own draft is due its first
  decision the moment its **welcome** arrives, and a welcome arrives when *that*
  seat is seated — but the room refuses a decision until **both** are
  (`draftOpen`), and a drafting room sends **nothing at all** when it fills
  (`bothTaken` returns `nil, nil`; an empty `wire.Drafted` is refused by design).
  So the host bans into a one-player room, is refused `CodeNotYourTurn`, and —
  because a refusal leaves the decision open — re-sends. **Measured: five
  refusals** before the second client arrived.
- ⚠️ **`Play` therefore has to answer BEFORE its first read** (deleting that hangs
  the socket draft test at its 60s bound), and **the retry-on-refusal loop is the
  only thing that would ever get the host's first ban through** — so a memo of
  "already answered this decision" turns the spin into a **stall** and must not be
  added on its own. The same loop is an unbounded hot loop for a genuinely illegal
  decision, and the battle path has carried that shape since `wire.Refused`.
- **Two fixes, neither a client fix**: the room announces its draft opening (a new
  server→client kind), or the first decision belongs to **the seat that filled the
  room** — the guest, which is the only event a client can hang "the draft has
  begun" on. Guest-first in the mirror *alone* hangs (7 tests), so both ends move
  together. Not taken: host-first is settled decision (f).
- ⚠️ **In-process fakes cannot see a transport.** Step 4's whole draft suite was
  green while `Mirror.Receive` had **no `Drafted` arm** — a `wire.Drafted` errored
  the client and closed the connection, and a drafting room was unjoinable by
  anything real. The fixture-hides-a-branch lesson at the level of a **package
  boundary**. See [[fixture-hidden-branch]], [[fixture-decides-what-is-visible]].
- The stale-answer discriminator is `DraftAnswer.For` = `DraftDue{Seat, Step,
  Character, **Recorded**}`. ⚠️ The **step alone is not enough**: a seat's two ban
  slots have the other seat's between them, so an answer for the first delivered
  at the second carries the same step and the same absent character — everything
  the wire has. `Recorded` (the record length when the decision was raised) is
  local and never travels.

**Why:** the gap looks like a client bug and is a protocol one, and the obvious
"fix" (don't re-answer a decision already answered) deadlocks the host.

**How to apply:** before adding any guard to `Client.answer`, check what still
triggers the host's first ban. → `hexarena/TODO.md` § step 5a, which carries all
of it. Steps **5b** (draft screen) and **5c** (arrange screen) are what is left.
