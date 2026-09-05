---
name: hexarena-a-new-clause-needs-every-site-that-evaluates-it
description: hexarena — adding a clause to skill.Condition is not done when the type parses it; every place that evaluates the condition by hand has to learn it, and the gate has two
metadata:
  type: feedback
---

`Condition.BelowStacks` (a lifetime cap on casts, added 2026-09-05) parsed,
round-tripped, and was read by `Condition.Holds` — and did **nothing**. Both
places that decide whether a gated skill may be cast compared the stacks
themselves:

```go
if held := unit.Statuses.Stacks(spends.Status); held < spends.MinStacks {
```

`battle.options` words the refusal for the offer and `battle.Act` re-checks it
because `internal/room` drives a PvP turn straight through `Act` off a decision
that arrived over the wire. Neither asked the condition, so a clause added to the
condition reached neither.

**How it was caught, which is the part worth copying.** The first test passed with
the cap disabled entirely — it read whatever prompt `Advance` returned, and by the
third cast that prompt belonged to a *copy* the split had put on the board. A copy
knows one skill, so "the split is not offered" was true of the copy and said
nothing about the caster. Rewriting it to advance until the **caster** is up made
it fail, and the failure was the real bug. See [[fixture-hidden-branch]].

**The fix's shape:** one `gateCloses(spends, held)` that both sites call, building
the whole `Option` so the counts travel with the reason. Same shape as
[[hexarena-a-trait-that-changes-a-price-has-two-sites]] one layer out — that one
was a trait read at two price sites, this is a clause read at two gate sites, and
in both cases nothing in the type system says the second site exists.

⚠️ **A cap also needs its own refusal, not the fuel one.** They tell a reader to
do opposite things: short of fuel you wait and fill the tank; spent out you are
finished with the skill for the battle. Reusing `BlockFuel` would have drawn
"needs 2 sundered, holding 2" — telling somebody to wait for something that is
never coming. New `Block` value, new key, new arm in `screen.OptionRefusal`.

**How to apply:** after adding a clause to `Condition`, grep for the fields it
sits beside — `MinStacks`, `Gates`, `BelowHealth` — and read **every** hit outside
the type. The parser and `Holds` are the two that are easy to remember and the two
that are not enough.
