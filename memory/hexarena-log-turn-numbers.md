---
name: hexarena-log-turn-numbers
description: "hexarena battle log — `A1 turn 5` then `E1 turn 4` is NOT out of order; event.Turn is the acting unit's own count, and the log is chronological (measured, At monotone). Left as-is 2026-08-31."
metadata:
  type: project
---

**Reported as "battle log thứ tự các turn hiển thị lộn xộn" (2026-08-31), and the log is right — INVESTIGATED, NO CHANGE MADE, by the user's call ("tạm giữ").** Do not re-diagnose this as an ordering bug.

`event.Turn` is **the acting unit's own turn number**, not a global round — `internal/core/battle/event.go:166` says so, and it is set from `turn.Number` / `b.queue.Turns(unit.ID)`. Statuses and cooldowns are counted in it, so `hồi 3` means three turns **of that unit**. A faster unit's count therefore runs ahead, and interleaving two units reads as numbers jumping about:

```
at 10000  venusaur   turn 1     ← time is straight
at 10989  charmeleon turn 1
at 20000  venusaur   turn 2
at 25641  wartortle  turn 1     ← the number is not
at 30000  venusaur   turn 3
at 31445  ivysaur    turn 2
```

**Measured, 3 seeds / 143 turns:** `At` (the action-value timeline) is monotone non-decreasing with **0** breaches, and each unit's own count increments by exactly 1 with **0** breaches. The screenshot the user sent proves it too: `A1 turn 5` applies poison to E1, then `E1 turn 4` takes the tick — cause before effect, so E1's 4th turn is after A1's 5th.

**There is no global "round" in this engine** (ATB action values, not rounds), but the nth-turn-taken-in-this-battle index is well defined and monotone. If this is ever revisited, the three options measured were: a global index **plus** the unit's own (`A1 turn 8 · its 5th` — count it in `Log`/`logRows`, the layer that already owns ordering, so `Line` keeps its "nothing but the event" contract); ownership spelled out only (`A1 its turn 4` — cheapest, gives the reader nothing to read order by); or the global index alone (loses the number cooldowns and status durations are measured in, so nothing on screen matches `hồi 3`). The header row is 12 cells against 79, so width is not the constraint.

⚠️ **How to check this class of report in one probe rather than by reading:** run `battle.New` → `Begin` → `RunToEnd` → `Drain`, then over the `TurnBegan` events assert `At` non-decreasing **and** each actor's own count stepping by 1. Two assertions separate "the log is out of order" from "the number is not what the reader thinks it is", and only the second was true here. Note `seed.Roster()` takes no argument and `battle.New` wants a `uint64` seed.

Related: [[hexarena-log-gloss]], [[hexarena-core-design]], [[hexarena-battle-screen-budget]].
