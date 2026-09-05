---
name: draft-state-machine
description: hexarena internal/draft — steps 2a, 2b, 3 and 4 built; the Done()/Picked() split, the arrange phase's own accessors, the record's SHAPE living in internal/wire, and the room-side arrange-phase clock (which is NOT one allowance)
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
