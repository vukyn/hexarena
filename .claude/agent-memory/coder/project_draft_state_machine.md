---
name: draft-state-machine
description: hexarena internal/draft — steps 2a and 2b built; the Done()/Picked() split, the arrange phase's own accessors, and the correction that a slot-less squad is REFUSED by Squad.Validate rather than silently stacking
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
