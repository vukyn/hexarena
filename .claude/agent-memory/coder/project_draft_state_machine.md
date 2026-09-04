---
name: draft-state-machine
description: hexarena step 2a — internal/draft's state machine, Picks() not Squads(); and the correction that a slot-less squad is REFUSED by Squad.Validate rather than silently stacking every unit on one cell
metadata:
  type: project
---

`internal/draft` step 2a (branch `feat/draft-machine`, 2026-09-05): `New`,
`Turn`, `Ban`/`SkipBan`, `Pick`/`Loadout`, `Candidates`, `TimedOut`,
`Done`/`Cancelled`, `Picks`, `Since`. Bans-first-then-picks, alternating from
`Config.First`; a pick is two decisions; the timeout is an input and cancels the
whole draft; the record is `battle.Since`'s shape (panic on a bad cursor,
three-index view). The output is `Picks() [2][]Pick` — the slot is step 2b's
decision (TODO.md § "Ban and pick" (g)).

⚠️ **The reason not to return slot-less squads is a REFUSAL, not silent
stacking**, and the brief I was handed said the opposite. `hex.Offset{}` is a
real cell (col 0 row 0, the ally back corner) and passes every *per-unit* check
in `placement.Squad.Validate` — but Validate also keys a `map[hex.Offset]string`
and refuses *"unit %q stands at 0,0, where %q already is"*, and `Squad.Take`
calls Validate. Every format is ≥3 units, so a squad built with `Slot` left at
its zero is **always turned away at the moment it is fought**, naming a cell
nobody chose. The conclusion (don't write `Squads()`) survives; the mechanism to
quote is the late refusal.

**Why:** the difference decides what a doc comment and a design note may claim,
and "passes Take and stacks the side on one cell" is checkable and false.

**How to apply:** when a brief explains *why* a shape is dangerous, re-derive the
mechanism against the code before repeating it in a comment — the verdict can be
right for the wrong reason. See
[[a-refusal-can-be-right-for-the-wrong-reason]]. For step 2b: what a Pick still
needs to become a `placement.Placement` is a `Slot` and an `ID`; Level and Stage
are already resolved.
