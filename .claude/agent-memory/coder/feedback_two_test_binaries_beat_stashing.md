---
name: two-test-binaries-beat-stashing
description: For an interleaved before/after wall-clock reading, compile `go test -c` binaries for both sides once and run them from the package dir — no stash toggling, and the RUN counts come free
metadata:
  type: feedback
---

To measure a change's effect on package wall clock, build **both sides as test
binaries up front** — `go test -c -o /tmp/before-<pkg> ./<pkg>` on the clean tree,
the same after the edit — then interleave `before, after, before, after…` by
running the binaries from inside the package directory (`cd <pkg> && ./before-…
-test.count=1 -test.v`).

**Why:** the alternative is `git stash` toggling six times, which is slow, risky
on a dirty tree, and easy to get wrong. Binaries also make the reading fair: both
sides get identical flags, and `grep -c '^=== RUN'` on each log answers "did the
suite stop testing something" in the same pass. Used for `SCR-011` (hexarena,
2026-09-08): three rounds a side over five packages, ~26 minutes, and no range
overlapped.

⚠️ **Two traps.** The binary must be run **from the package directory** — these
suites use relative paths like `../../internal/seed/data`. And a temporary timing
harness added between the two compiles lands in only one binary and shows up as a
phantom `=== RUN` delta; either build both sides after adding it, or say so.

**How to apply:** any perf claim in this repo. One pair of runs is not a
measurement — the first half of `SCR-011` got the *direction* wrong from a single
pair, because main's fastest run lined up against the branch's median. Report the
spread. Related: [[feedback_measure_the_term_before_optimising_it]],
[[feedback_measure_the_thing_a_bound_bounds]].
