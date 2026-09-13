---
name: a-heading-held-by-prose-decays
description: A document rule asserted in prose cannot notice the document changing under it — hold it with a command, and show the command's positive case
metadata:
  type: feedback
---

**A rule about a file, written as a sentence in that file, is not a rule. It is a
claim, and nothing re-checks it.**

`DOC-001`. On 2026-09-05 a sweep moved finished work out of `TODO.md` and wrote in
the header: *"`## Not done` is only what is not done. Finished work and its
reasoning moved to `docs/decisions.md`."* True when written. **False eight days
later**, and by 2026-09-13 § *Not done* held **thirty finished entries against
twelve open ones** — 2,663 lines of shipped work filed as unfinished.

Nothing went wrong in anybody's discipline. The rule decayed because the *cheap*
way to close an item was to change `[ ]` to `[x]` where it sat, and the sentence
had no way to object. Three places in the repository then stated three different
rules at once: this header, § *The codes* (`done` = kept in place, with its
measurements) and `memory/hexarena-battle-screen-budget.md` (leaving a `[x]` there
is a mistake, *"made twice"* — it was thirty).

**The fix is not a better sentence.** Closing an item now MOVES it, and the
heading is held by one command:

```sh
awk '/^## Not done/{f=1;next} /^## /{f=0} f && /^- \[x\]/' TODO.md
```

⚠️ **The load-bearing half is the positive case.** It prints nothing on the swept
tree and **thirty** on the commit before — `git show HEAD~1:TODO.md` piped into
the same `awk`. A guard whose failing case is never shown is not evidence that it
works, only a command somebody believes in. Same rule as
[[a-search-that-returns-zero-needs-a-known-positive]] and
[[a-guard-on-the-wrong-side-of-the-gate]]: a green check with no demonstrated red
is decoration.

**How to apply.**

- A documentation invariant that can be expressed as a grep or an `awk` should be,
  and the check belongs in the file it guards so a reader meets it.
- State the count the check returns today **and** on a tree known to violate it.
- When a rule has to change, grep for every place that states it. This one was
  written in three; a rule deleted in one file and left standing in another is how
  the defect returns.
- ⚠️ Do not accept "it will be distilled later" as an arrangement. A staging area
  with no deadline is never emptied — that is what these eight days measured.

Xem thêm [[a-heading-is-not-a-rule]], [[todo-items-are-addressed-by-code]].
