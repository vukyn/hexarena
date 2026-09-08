---
name: measure-the-term-before-optimising-it
description: SCR-011 blamed a 17 MB per-test copy; on APFS one copy is 23ms sharing vs 22ms copying — the byte win was 71x and the clock win was 4.7%
metadata:
  type: feedback
---

Before optimising the cost a work item names, **time that one operation both
ways in isolation** — mutate the new mechanism off and run the same harness —
and **count how often it really happens at runtime**, not how many call sites
grep finds.

**Why:** `SCR-011` said `cmd/hexforge-tui` spent its ~1300s copying 17 MB of SVG
per `scratchData`, "essentially all of it". Measured on macOS/APFS: one copy of
the data directory is **23 ms sharing the art and 22 ms copying it** — seventeen
megabytes of page-cached SVG costs nothing on that filesystem. A whole
`scratchData` is ~215 ms and `testfixture.Inject` (re-parsing and re-saving the
books through `forge`) is ~180 ms of it. The change was still right — 13.5 GB of
writes per suite run down to 189 MB, and it removes a cost that grew with every
picture added — but the package moved **111.8s → 106.5s**, not to a tenth. The
item's big numbers were a **Windows** reading; the entry said so in one clause
and its headline did not.

The runtime count was the other surprise: the entry grepped **155** call sites
and the package really makes **301** copies (784 across the five packages that
had the same helper). A site count inside table-driven tests is about half.

**How to apply:** on any perf item here, produce three numbers before the diff is
judged — the operation A/B'd in a benchmark, the runtime invocation count
(instrument the function with an append-to-file behind an env var, run once,
revert), and the package's `go test` time before and after with `=== RUN` counts
either side so it is visible that no test stopped running. Say which platform
each number is from; this repo's suites are read on macOS and on Windows and the
two disagree about file I/O by orders of magnitude. Related:
[[a-well-formed-measurement-can-measure-nothing]],
[[measure-the-thing-a-bound-bounds]], [[measure-which-guard-masks]].
