---
name: unstable-values-in-a-record
description: hexarena — buildString() is "devel" under go test and a VCS hash in a build, so it must be substituted before a golden records it AND hardcoding it is undetectable by the suite; the data digest is the opposite and stays real
metadata:
  type: feedback
---

Two values sit side by side on the join screen and they are **opposite cases**.
Getting them the same way round is the whole of the decision.

- **`seed.DataDigest().Short()` is stable.** It is over the fifteen committed
  JSON files, so it is the same twelve characters on every machine
  (`df3bed25a5c5` on `8856447`). Put the real one in the golden: a diff over it
  is exactly the finding the record should carry.
- **`buildString()` is not.** Measured: `devel` under `go test` (test binaries
  get no VCS stamping) and `8856447e93c1+dirty` out of `go build` on the same
  tree. `TestThisBinaryKnowsWhatItIs` already refuses to assert its value and
  says why. So a golden entry must substitute a fixture value first — same
  hazard the client golden already handles for the data directory by naming it
  with a relative path.

**⚠️ And the corollary that cost a mutation round: a value the suite cannot
vary is a value no test in the suite can pin.** Replacing
`wire.Local(buildString())` with `wire.Local("devel")` in the production path
left every new test green, because `devel` is precisely what `buildString()`
answers under `go test`. Four other mutations on the same three lines
(hardcode the digest / draw an empty digest / swap the two arguments / never
draw the line) all reddened. So do not claim a stamped-value seam is covered —
name it unmeasurable and move on. The digest half has no such gap, because a
**fabricated** digest is a value the data cannot produce: hand the screen
`0xab…` and assert the shipped short digest is *absent*, which is the only thing
that catches a hardcoded literal.

**Why:** the same fixture that makes a record readable makes it lie if the
substitution goes the wrong way — a golden holding `devel` is a golden that
says how the suite was invoked, and a golden holding a substituted digest is a
golden blind to the one number a `data_mismatch` is an argument about. Size the
fixture build string at **18 characters** (`0123456789ab+dirty`) — the widest
`wire.BuildOf`'s own derivation produces — so the width sweep still measures the
worst case; an `-ldflags` stamp is any length and is nobody's floor to defend.

**Also from the same task, an ordering footgun:** `git add --renormalize .`
re-adds **every tracked file** matching the pathspec, so run it immediately
after writing `.gitattributes` and **before** touching any code. Run late it
stages the unrelated edits, which blurs a two-commit PR and collides with the
user's "stage explicit paths only" rule. On a tree with no CR anywhere it stages
**nothing**, and nothing is the expected answer.

Related: [[two-screen-goldens]] for what each of the three goldens can see,
[[a-well-formed-measurement-can-measure-nothing]],
[[fixture-decides-what-is-visible]].
