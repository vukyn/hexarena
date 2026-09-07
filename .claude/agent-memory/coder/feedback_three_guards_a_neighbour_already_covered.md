---
name: three-guards-a-neighbour-already-covered
description: Three guards in one change were deleted by a mutation with the suite still green, because a neighbouring path already covered each — the fixes are one declaration, a different reading (the clock generation), and saying so in the comment
metadata:
  type: feedback
---

Mutating my own new code found **three** guards that no test could see, in one
change. Each looked load-bearing and each was already covered by the path beside
it. Run the mutation on every guard you add, not only on the interesting ones.

**Why:** a guard a mutation deletes for free is a guard the next reader deletes
for free, and the suite says nothing either time. This repository already had the
lesson once (`healingFor`'s floor, where the caller's `amount == 0` swallowed the
negative); it recurs whenever two layers can both refuse the same thing.

**How to apply — the three shapes, and what each needed:**

- **A bound stated twice.** The watcher cap was checked in `Server.watched`
  (which refuses) *and* in `table.admit` (which returns false). Rewriting `admit`
  to evict the oldest watcher instead left **the whole suite green**, because the
  caller's refusal meant the mutated line never ran. Fix: **one declaration** —
  `table.full()` is the bound, `admit` only appends. The bound has to be readable
  *before* the welcome goes out (a welcome then a refusal is two answers to one
  hello), so the refusal cannot move into the mutator.
- **A branch whose wrongness the layer below tolerates.** Routing a watcher's
  departure into `Server.left` — reporting a *seat* leaving — changed nothing,
  because `Room.Left` returns `nil, nil` for a seat that was never taken. What it
  *did* change was the clock: the transport re-arms the allowance after every
  departure it reports, so the fix was to assert **`allowance.armed()`'s
  generation** across the departure. Find the one reading a wrong path moves;
  here every other reading was identical.
- **A prompt action a slower path also performs.** Deleting `endWatching` on the
  match-finished path reddened nothing: both players leave the moment a match
  ends and their departures reach the same call milliseconds later. Making the
  whole function a no-op *did* redden. So the claim (something ends a watcher)
  holds; the promptness does not, and the comment now says so, with the one case
  where it matters (a match ended by a timeout with both peers still connected).

⚠️ **A mutation that does not compile proves nothing** — `admit(peer, 0, stop)`
left `cursor` unused and the build failed; `cursor*0` is the same mutation that
runs. And **snapshot the file, not `git checkout`**: the baseline here was
uncommitted working-tree code, so restoring from HEAD would have thrown the whole
change away. Copy to the scratchpad, restore, and re-check the md5 every time.

Related: [[measure-which-guard-masks]],
[[a-mutation-must-hit-the-arm-you-claim]],
[[a-read-after-the-exchange-misses-the-last-body]].
