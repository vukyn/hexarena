---
name: hexarena-a-mutation-baseline-must-come-from-git
description: hexarena — a mutation harness that copies the working file as its backup turns a crashed run into a silent permanent edit; take the baseline from `git show HEAD:<file>` and restore with `git checkout --`
metadata:
  type: feedback
---

A mutation sweep over eight keystroke rules was written like this:

```python
shutil.copy(SRC, BAK)          # baseline = whatever is on disk right now
...
shutil.copy(BAK, SRC)          # restore, at the end
```

The first run crashed part way through — after a mutation had been written to
disk and before the restore. The fix to the crash was a one-line edit and a
re-run, and **the re-run copied the mutated file as its new baseline**. From then
on every measurement was taken against a `play.go` that was quietly missing a
guard, and the final "restore" put that file back for good. It was caught only by
noticing that one mutation reported `pattern not found` — the pattern it looked
for was the code the previous run had deleted.

**How to apply.** A mutation harness must take its baseline from the object
store, never from the working tree, and restore the same way:

```python
base = subprocess.run(['git','show','HEAD:'+SRC], capture_output=True, text=True).stdout
...
subprocess.run(['git','checkout','--',SRC], check=True)
```

Two properties fall out that a file copy cannot give: a crash leaves a *dirty
tree* rather than a new baseline, so `git status` shows it; and the restore is
idempotent, so re-running the harness is always safe. Check `git status` after
any mutation run — a clean tree is the proof the sweep put everything back.

⚠️ This is the mirror of [[git-checkout-discards-to-head]], not a contradiction
of it: `git checkout --` is right here because the file is supposed to be
untouched work, and wrong there because the file held real edits. What decides it
is whether the working copy is a deliberate change or a harness artefact — so
never point it at a file the session has been editing, which is why the harness
should mutate one file and the tests should live in another.
