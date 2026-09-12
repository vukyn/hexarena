---
name: a-bound-is-two-books-multiplied
description: A stat's widest possible figure is the ceiling TIMES the headroom, not the ceiling — 800 and 3000‰ give 2399, and the ceiling alone is already wrong against figures the game draws today
metadata:
  type: reference
---

Sizing the roster's attack and defence columns needed the widest figure either can
ever be. The obvious answers are both wrong:

- **The widest value observed** — 960 across every golden — is not a bound. It is
  a fact about the fixtures that happened to be recorded.
- **The ceiling** in `progression.json`, `ceilings.attack = 800`, is not a bound
  either. It bounds a unit's *base*. The roster already draws **945** defence.

The bound is **two books multiplied**. `Battle.Stats` runs the base through
`modifier.Set.Stat`, which saturates towards `limit = ceiling × Headroom / 1000`,
and `scale.Saturate` is `base + gap*delta/(gap+delta)` — strictly under
`base+gap` in integer arithmetic. So the answer is `limit − 1`:

```
ceiling 800 × headroom 3000‰ = 2400  →  widest figure 2399  →  four digits
```

`progression.Limits.CheckValues` in `battle.enlist` is what holds the base under
the ceiling, and every unit goes through it — **a summon included**, which is the
path a "bases come from the cast book" reading would miss.

**Why:** a ceiling and a multiplier live in different files, so a reader holding
one of them has a number that looks authoritative and is short by a factor of
three. Three digits would have been wrong on the day it was written, not at some
future retune.

**How to apply:** when a column, a buffer or a format needs a maximum, ask what
*else* scales it before sizing. Derive the figure from the books in a test that
reads both, so a retune of either moves the guard — see
[[a-guard-on-the-wrong-side-of-the-gate]] for the general form, and
⚠️ **do not size anything to the largest value you can find in a golden**, which
is [[a-search-that-returns-zero-needs-a-known-positive]] in the other direction:
the record shows what has happened, never what can.
