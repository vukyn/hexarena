---
name: third-lan-flake
description: hexarena — TestTheCountdownReachesTheScreenOverASocket is a THIRD intermittent cmd/hexarena-tui test under -race, unrecorded in TODO.md; seen 1 of 3 make-check runs on 2026-09-04, unreproducible in 18 targeted runs
metadata:
  type: project
---

`TODO.md` records **two** LAN tests that fail in a loaded suite and pass alone
(`TestShutdownGivesUpAndNamesWhatItWasWaitingFor`,
`TestAJoinedMatchPlaysToItsEndOverALoopbackListener`). There is a **third**, and
it is not in that entry:

**`cmd/hexarena-tui`'s `TestTheCountdownReachesTheScreenOverASocket`** failed
inside `make check`'s `-race` line on 2026-09-04 with *both* a
`WARNING: DATA RACE` and its own assertion — `clock_test.go:338: the board drew 2
with 90 and 90 seconds`. Racing frames were `session.begin`'s Play goroutine
(`session.go:254`) against `session.dial`'s closure (`session.go:214`), reached
from `clock_test.go:296`.

**Why it matters:** it looks like a regression when it fires beside unrelated
work. Measured that day — 1 of 3 full `make check` runs red; **0 of 3** isolated
`-race` runs of the whole package on the working tree; **0 of 3** on a pristine
copy of HEAD with the branch's nine files restored from git; **0 of 12** loaded
runs of just the two socket-driven clock tests. So it is load-dependent, it
predates the `feat/say-which-version` work, and no amount of targeted running
brings it back.

**How to apply:** if this test reddens under `make check` while you are working
somewhere else in the repo, do not start bisecting your diff. Reproduce the way
that settles it cheaply: build a **git-metadata-free copy** of the tree
(`tar --exclude=.git`), restore your modified files from `git show HEAD:<path>`
into the copy, delete your new untracked test files, and run the package there —
never `git checkout` in the live worktree, and never a second worktree, since the
user runs parallel sessions. Then report the counts rather than a verdict.
The subject of the race is still uncaptured; catching a full report under load is
the open piece.

Related: [[measure-the-thing-a-bound-bounds]] for `-race`'s habit of catching a
test's own assumption rather than the code's.
