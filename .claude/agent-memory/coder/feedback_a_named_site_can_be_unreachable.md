---
name: a-named-site-can-be-unreachable
description: A file:line a report names as broken may not be reachable in the state the report describes — filepath.Rel("", p) at play.go:1066 sits behind an early return, so the "empty first argument" never happens
metadata:
  type: feedback
---

**A site a bug report names is a hypothesis about control flow, not a
measurement.** Trace the state to it before writing a guard there.

**Why:** the art step's spec listed `internal/screen/play.go:1066` —
`filepath.Rel(c.Lib.Dir(), path)` "with an empty first argument" — as a defect
to measure and fix. It is unreachable in that state. Twelve lines above it,
`c.Lib.SaveBattleLog(...)` refuses a library with no directory
(`ErrNoDataDirectory`) and `save` returns on the error, so the `Rel` is only
ever reached past a **successful** write, which needs a directory. Withdrawing
the save offer made it doubly unreachable. A guard added there would have been
dead code carrying a comment claiming it was load-bearing.

The measurement was worth taking anyway, and it says the call is safe even if
the reasoning about reachability is one refactor from being wrong:

| call | answer |
|---|---|
| `filepath.Rel("", "/tmp/x/battles/a.json")` | `""`, **error** |
| `filepath.Rel("", "battles/a.json")` | `"battles/a.json"`, nil |
| `filepath.Rel("", "")` | `"."`, nil |

So the call site's existing `if shortened, err := …; err == nil` already carries
it: a rooted path errors and the whole path is kept, a relative path comes back
unchanged which is already the answer wanted. Only `Rel("", "")` mangles
anything, and an empty path only exists on the error return.

**How to apply:** when a report names a line, (1) run the call in a scratch
program to see what it really answers — it took four lines and refuted the
premise; (2) read *upward* for the early return that decides whether the state
can arrive; (3) if it cannot, the deliverable is an **assertion of
unreachability** plus the measurement, not a guard. Assert both halves: a test
that only says "no note was produced" is a fact about today's control flow, and
the measured table is what makes being wrong about that safe.

Related: [[an-empty-path-field-is-a-relative-path]] (the sibling finding, where
the empty path really *was* reachable and really did read the CWD — same
expression, opposite verdict, which is why neither can be assumed from the
other), [[a-refusal-can-be-right-for-the-wrong-reason]].
