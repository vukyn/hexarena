---
name: a-golden-can-be-red-before-you-touch-it
description: Golden lines in your diff may be the PREVIOUS commit's unaccepted debt; prove authorship with a git archive of HEAD, not a stash
metadata:
  type: feedback
---

**A golden that moves in your diff is not necessarily yours.** Before explaining
it, prove whether it was already red at HEAD.

**Why:** in `DAT-013` (hexarena), `make golden` moved 24 lines across
`internal/screen/testdata/screens.golden` and
`cmd/hexforge-tui/testdata/screens.golden`. None of them mentioned the build I
had changed — the diff was `pokemon.happiny` gaining two rows and `pokemon.mew`
losing two off the bottom of a **scrolling viewport**. The previous commit
(`#395`, DAT-011) had added `happiny.tend`/`happiny.decoy` to `builds.json`
without re-accepting the goldens, so `make check` was **already broken on main**.
Attributing those lines to my change would have been a false claim in the PR body
about what a balance edit did, and the repo's own rule is that *a golden that
moves where you did not expect is a finding, not noise* — the finding here was
somebody else's debt.

The inverse mattered just as much: my actual change moved **no golden line at
all**, because the builds screen draws a build's id and name and I changed its
skills and its intent. So no golden guards the change, and the hardcoded design
record (`hexBuild` in `newbuilds_test.go`) is the only thing that does. "The
goldens moved" and "the goldens saw my change" are independent facts.

**How to apply:** to attribute a golden line, run the golden test against a clean
tree at HEAD:

```
git archive HEAD | tar -x -C /tmp/<name> && cd /tmp/<name> && go test ./<pkg> -count=1
```

Use `git archive`, **not `git stash`** — a stash is a repo-level stack shared by
every worktree, and this platform runs parallel sessions on the same repo, so a
sibling session can pop yours. The archive is read-only and touches nothing.
Then say in the report which lines are yours and which you inherited.

Related: [[pin-comparison-base-to-worktree-head]], [[two-test-binaries-beat-stashing]].
