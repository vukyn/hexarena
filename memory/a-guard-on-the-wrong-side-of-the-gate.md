---
name: a-guard-on-the-wrong-side-of-the-gate
description: A guard that fires only where the existing test already fires detects nothing — check which side of the test RUN the gap is on before designing an assertion
metadata:
  type: feedback
---

`ENG-014`. A stale golden reached `main` and sat there red. My first proposal was a
new assertion — a guard that a golden derived from `internal/seed/data` must be
younger than the last change to that data. **It is vacuous, and provably so:** it
would fire only where `go test` runs, and where `go test` runs, the golden test
**already** fires, with a better message that names the differing line. A second
red line in the one place already red detects nothing.

The gap was not a missing assertion. It was that **nothing runs the suite on a
pull request** — no `.github/workflows/`, and `gh pr view --json statusCheckRollup`
returns exactly one check, a secret scanner.

**Why:** "the break was not caught" and "the break was not detectable" are
different sentences, and the fix for each lives on a different side of the test
run. Reaching for an assertion is the reflex; it is the right move only when the
run happens and says nothing.

**How to apply:** before designing a guard, ask *where would this fire, and what
already fires there?* If the answer is "the same place as the test that already
fails", the work is to make the run happen — CI, a hook, a required check — not to
add an assertion to a run nobody performs. Same discipline as
[[fixture-hidden-branch]] pointed the other way: there the assertion existed and
no data reached it; here the assertion reaches the defect and no *run* reaches the
assertion.

⚠️ Paired lesson, from the same item: I raised this as "the fourth occurrence of
one pattern" and **the count was one.** I had conflated *main was red* with *a
golden went stale* across four events sharing only the first half — measured one
at a time, `#307`'s goldens are green at its own commit (its redness was a
digest), `#319` changed three golden files and no data, `#326` changed one golden
file and no data. Only `#395` is the shape. And the wide number is worse than
useless: 88 of 129 commits touching `internal/seed/data` re-accepted no golden,
which is **not** 88 breaks, because the goldens cover only some screens. See
[[a-number-without-a-counter-check-is-just-a-number]].
