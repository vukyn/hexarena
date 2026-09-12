---
name: a-fixture-keyed-on-the-marker-it-tests
description: A fixture loop that stops when it sees the very marker under test makes that marker decide whether the test runs — deleting ` +N` stopped crowdedUntilElided ever reaching its state, so the golden was never compared and the red read "could not build the fixture"
metadata:
  type: feedback
---

A golden was supposed to picture a roster row whose effects column had to leave
something out. The fixture crowded a unit with real statuses and stopped when
`strings.Contains(tui.Effects(unit), "+")` — the ` +N` marker. Deleting that
marker from `elided` then reddened the test, but on the **wrong line**: the loop
never saw a "+", exhausted the status book and hit its own `t.Fatalf`, so the
golden bytes were never compared at all. The report read *"every timed status is
on Bulwark and the column still draws them all"* — a fixture complaint — instead
of a diff showing the row that had quietly stopped saying what it hid.

**Why:** the marker was both the thing under test and the thing gating whether
the test got to run. Any mutation to it takes the test out of service rather than
failing it, which is the same defect as skipping on a result instead of a probe:
the guard evaporates exactly when it is needed.

**How to apply:** build the fixture on a property the mutation cannot switch off.
Here that is *counting*: `len(snapshot) - len(strings.Split(drawn, ", "))` — how
many effects the column did not draw — which is true whether or not the row says
so. With that, deleting the marker let the fixture reach its state and the golden
failed on its bytes, naming the row. Close relatives:
[[a-fixture-near-a-cap-reddens-on-the-cap]] (the fixture reddens on a capacity
premise instead of the claim) and the repo's own
`memory/a-skip-keyed-on-the-result-deletes-the-guard.md`.
