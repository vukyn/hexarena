---
name: a-green-expected-mutation-proves-nothing-by-itself
description: A mutation whose desired outcome is GREEN carries no evidence it landed — DAT-008's M3 was invisible to every test and golden in the repo, so it had to be probed at the field directly
metadata:
  type: feedback
---

**A mutation you expect to REDDEN proves itself; one you expect to stay GREEN
does not.** When the mutation reddens, the redness is the receipt that it
reached the code. When the whole point is that it stays green, a mutation that
never landed and a mutation the guard correctly ignores produce the identical
result — so a green arm needs a separate proof that the edit was observable at
all.

**Why:** DAT-008's mutation M3 gave `bedrock` a `"name"` field in
`statuses.json`; the guard (`fingerprint` zeroes `Kind.ID` and `Kind.Name` as
labels) must stay green, and it did. But running `internal/seed` *and*
`internal/i18n` with the mutation in place moved **nothing** — no golden, no
test — and swapping the value for one that is not the compiled i18n fallback
(`"nền đá"` → `"nền đá thử"`) still moved nothing. At that point "green because
`fingerprint` ignores labels" and "green because the field goes nowhere" were
indistinguishable, and the second would have made the arm worthless.

The proof was a throwaway test that loaded the book and `t.Fatalf`'d the field:
`Name="nền đá thử"`. The mutation really does reach `status.Kind.Name`, which is
exactly the field `fingerprint` zeroes, so the green is a reading of the guard.

⚠️ **The reason nothing else saw it is itself the finding, not noise.** No
fixture in the repository fires `ground_root`, so `bedrock` appears on **zero**
board goldens — only catalogue rows. A shipped field observed by nothing is the
same shape as *a mechanism no shipped placement fields is a mechanism nothing
measures*, and it is what became `DAT-010`.

**How to apply:**

- Run every mutation, red-expected or not. For a green-expected one, add a third
  step after "mutate" and "run the guard": **prove the mutated value reaches the
  code**, by asserting on the parsed value directly rather than on any test's
  verdict.
- The cheapest probe is a `t.Fatalf` printing the loaded field, deleted straight
  after. It costs one run and turns "green" into a reading.
- If nothing at all in the suite observes the mutated field, say so in the
  writeup — that is a coverage finding worth its own item, not a convenience.
- Revert data mutations by hand-editing the value back, never `git checkout`
  ([[git-checkout-discards-to-head]]), and check `git diff <data dir>` is empty
  before finishing.

Related: [[a-well-formed-measurement-can-measure-nothing]],
[[a-mutation-must-hit-the-arm-you-claim]], [[a-null-needs-two-controls]],
[[fixture-decides-what-is-visible]].
