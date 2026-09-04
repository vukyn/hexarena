---
name: rebase-onto-a-moved-main
description: hexarena's origin/main moves during almost every task — and mid-task, by a background pull on the shared worktree; rebase by patch (never git stash), exclude the goldens, and pin `git rev-parse HEAD` either side of any before/after golden measurement
metadata:
  type: feedback
---

Re-fetch `origin/main` before finishing any hexarena task, and expect it to have
moved.

**Why:** it has now caught three steps in a row. #212 (a data PR adding two
characters to a list a golden counted), #220 (a data PR adding a skill, so
`90 chiêu` became `91 chiêu`) and #221, which landed *during* 5d-iii and moved
**both** `screens.golden` files, `describe.golden`, `skills.golden` and
`cast.golden`. A clean `git merge` says nothing about a golden's content: the
files merge and the recorded numbers are stale.

**How to apply** — the recipe that worked, on a branch with **no local commits**
and a dirty tree:

1. `git diff -- . ':(exclude)<the golden you regenerate>' > work.patch`, and copy
   any **untracked** files (a moved screen's new home is `??`) to the scratchpad.
   Leaving the golden out of the patch is the whole trick — it is the one file
   guaranteed to conflict, and it is the one file you can rebuild.
2. Delete the untracked copies, `git checkout -- .`, `git merge --ff-only
   origin/main`.
3. `git apply --3way work.patch`, restore the untracked files.
4. `make golden`, then re-check `git diff --numstat` on **both** goldens against
   the new base: the one you did not mean to move must be **absent** from the
   diff, and the one you did must still be additions-only.
5. `git reset` (mixed) afterwards — `git apply --3way` **stages** everything, and
   this repo's convention is that the committer stages explicit paths.

⚠️ **`git apply` printed "Applied patch to … cleanly" for twenty files and then
landed nothing.** It is atomic, so one failure rolls the whole thing back and the
per-file lines are already on stdout. Running it a second time worked. **Check
`git status --short` after applying**, never the log lines.

⚠️ **Do not use `git stash`.** The stash ref is repo-global and other sessions
work in sibling worktrees of the same repository; a patch file in the scratchpad
is nobody else's business.

⚠️ **It also moves DURING a task, without anybody telling you.** Measured
2026-09-03: the worktree was fast-forwarded by a background `pull` twice inside
one session (`c98d218` → `eeee515`, the cast-file reformat), and the symptom was
a golden that failed, passed after `make golden`, and failed again an hour later
with the identical diff — plus a `TODO.md` whose entries had quietly changed
under a `sed -n` I had already read. **A golden measurement taken across that
boundary measures the pull, not the change.** So: `git rev-parse --short HEAD`
immediately before and after any before/after golden run, and if it moved,
throw the reading away and re-take it. `git status` is not enough — the tree
reads *clean* the whole time, because the mover is a fast-forward and not an
edit.

Repo context in [[screen-extraction]]; what each golden catches is
[[two-screen-goldens]].
