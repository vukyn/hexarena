---
name: hexarena-sample-through-the-chooser
description: hexarena — to read a unit's live state during a battle, wrap Suggest and pass it to RunToEndWith; the event log carries what an application ADDED, never the running total
metadata:
  type: project
---

A claim like *"this trait's status builds to its cap over a fight"* cannot be read
off `[]Event`. `Event.Stacks` is **how many stacks an application added**, not the
depth the holder is standing on, and `RunToEnd` gives back only a turn count — by
the time it returns, the fight is over and the statuses with it.

**The seam that already exists:** `Battle.RunToEndWith(maxTurns, Chooser)` calls
the `Chooser` once per open turn with the prompt. Wrap `Suggest`:

```go
mine, _ := fight.Unit("mine")
best := 0
sampling := func(prompt *battle.Prompt) (battle.Choice, bool) {
    if prompt.Unit == mine.ID {
        if held := mine.Statuses.Stacks("stoked"); held > best {
            best = held
        }
    }
    return fight.Suggest(prompt)
}
fight.RunToEndWith(4000, sampling)
```

`Unit.Statuses.Stacks(id)` is the running total. The sample lands at the moment
the unit is due to act — **after** the tick that spends durations and after the
renewal that lands behind it, which is exactly the moment a per-turn trait's claim
is about.

**Why it is worth knowing:** the alternative reached for first is to drive the
battle by hand with `Advance`/`Act`, which duplicates `RunToEndWith`'s loop —
including the `prompt.Skipped` and `Pass` arms that are easy to get subtly wrong
and that decide whether a turn counts at all. Related:
[[hexarena-a-mutation-the-parser-refuses-runs-nothing]] for how the resulting
figure was then held.
