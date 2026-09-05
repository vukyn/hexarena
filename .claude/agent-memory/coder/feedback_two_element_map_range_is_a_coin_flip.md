---
name: two-element-map-range-is-a-coin-flip
description: a map-range mutation over a TWO-element set is caught in only 17 of 20 runs, so one test run cannot prove the ordering rule — read the same state N times; and separately, a "release" that no accessor can observe is caught 0 of 10 and must be labelled as such
metadata:
  type: feedback
---

Two readings out of one mutation run on `internal/draft`'s arrange phase, both
about a mutation that *looked* well covered.

**A map range over two keys is right half the time, so a single test run is not
a measurement.** The rule under test was "derive this slice from the `seats`
array, never from a map — the order reaches an output". Mutating the loop into a
`map[wire.Seat]int` range reddened the package in **17 of 20 runs** and passed in
3, even though four separate assertions read that order: Go randomises map order,
but with two keys it picks the right one half the time, and each assertion is an
independent coin flip. A test that has to be re-run to be believed is the shape
`CLAUDE.md` § *Mistakes already made here* keeps a note about.
The fix is not more assertions in more tests — it is **N readings of one
unchanging state** in a single test (64 was plenty, and free, because the state
is built once). That took the same mutation to **10 of 10**.

**A release nothing can observe is caught 0 of 10, and the honest move is to say
so in the comment.** Clearing the arrange-phase buffer inside `abandon` (the
timeout path) looked like a guard against leaking a buffered arrangement.
Deleting the clear left the **whole suite green**: every accessor that could
answer from the buffer — `Squads`, `Arranging`, `AwaitingArrangement`, `Done` —
gates on `Cancelled`/`Picked` first, so the retained data is unreachable. The
*behavioural* claim ("no arrangement entry reaches a cancelled draft's record")
was already held by a different line — nothing ever appends it — so the clear is
a **release**, not a chokepoint.

**Why:** both readings change what a report may claim. "Eight mutations, eight
caught" would have been false twice over: once probabilistically, once
absolutely. And a reader who later finds the unobservable line will go looking
for the test that holds it and waste a cycle.

**How to apply:** before quoting a mutation as caught, run it **≥10 times** if
the thing it perturbs is an ordering over a small set. And when a mutation is
*not* caught, decide between the two endings rather than adding a test to make
the number look better: either it is unobservable (keep it if it is a release,
and write "measured unobservable" into the comment so nobody hunts for its test)
or it is a real gap. See [[a-well-formed-measurement-can-measure-nothing]] and
[[fixture-decides-what-is-visible]].
