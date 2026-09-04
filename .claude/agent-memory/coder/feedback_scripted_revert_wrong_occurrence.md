---
name: scripted-revert-wrong-occurrence
description: Reverting a mutation with a scripted find/replace can restore the WRONG occurrence and leave the suite green — always finish with git diff on the file, never on the test result
metadata:
  type: feedback
---

When a test-only task needs a production file mutated and then put back, the
revert must be verified with `git diff <file>`, not with a green test run.

**Why:** measured in hexarena while proving `internal/screen`'s new golden can
fail. `species.go` holds `return out.String(), footer` **twice** — an empty-book
early return and the real one — so a `perl -0pi -e 's/…/…/'` (no `/g`, slurp
mode) restore matched the *first* occurrence, which was the wrong line. The
suite went green anyway, because the empty-book branch is unreachable with real
data: no test and no golden could see it. Only `git status --short` showed the
file still dirty.

Two things follow. A revert-by-script is a second edit, not an undo — it needs
its own check. And a mutation landing in a branch **no fixture reaches** is
exactly the shape this repo keeps a list of (`plainTerminal`'s early return, the
finished-battle fixture, the `fire_fang` ally-0 row): green means nothing about
that branch.

**How to apply:** the tree is normally dirty with the task's own work, so
`git checkout --` is unavailable — edit the value back by hand and then read
`git diff` on the file until it is empty. Target the **line number**
(`sed -i '' '196s/…/…/'`) rather than the text, and print the line before and
after so the mutation and the restore are both visible.

⚠️ **The same hazard runs the other way: a scripted INSERT can swallow the line
it anchors on.** Measured 2026-09-03 adding the art preview to
`cmd/hexarena-tui`'s `everyScreen` — the anchor was two lines
(`screens["a battle with no pairing"] = …` + `return screens`) and the
replacement re-emitted only the second, silently deleting a sweep entry. The
symptom was **not** a compile error and not an obviously wrong test: the golden
diff simply landed under a banner nobody had touched (`a saved battle` where the
record held `a battle with no pairing`), which reads like an unrelated regression.
A golden diff at an unexpected banner means an entry was **added or removed**
before it, not that the drawing moved — check the edit before the code. Anchor on
one line, or `git diff` the edited file before running anything.

⚠️ **A file the step just created is UNTRACKED, so `git diff` says nothing about
it.** That is the case in every move-a-file step: the new copy in
`internal/screen` is `??`. Copy it to the scratchpad **before** the first
mutation and finish each revert with `diff <backup> <file>` — same discipline,
different instrument. Also prefer patterns anchored on enough surrounding text to
be unique, and check the match count first (`grep -c`).
