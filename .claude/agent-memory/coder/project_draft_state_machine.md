---
name: draft-state-machine
description: hexarena internal/draft — steps 2a, 2b and 3 built; the Done()/Picked() split, the arrange phase's own accessors, and the step-3 decision that the record's SHAPE now lives in internal/wire
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
