---
name: assert-on-the-clause-not-the-whole-page
description: strings.Contains over a whole rendered page passes when some other part of that page holds the substring — assert on the clause under test, not the output.
metadata:
  type: project
---

`hexforge census` draws a header, one table per board, then a verdict. The verdict
is the only part the front-end can get wrong on its own: `CensusWalk` stops at the
first board that plays every slot, so the last table is *always* a clean board and
a verdict read off it says "plays every slot" in identical words whether the build
needed one board or four.

The test for it asserted `strings.Contains(page, "3 board")`. **It passed with the
verdict collapsed to the one-board wording**, because the *header* line reads
`3 board(s) walked`. The mutation said so; reading it did not.

Fixed by slicing the verdict off the page first —
`page[strings.LastIndex(page, "against s03"):]` — and asserting there, plus a
negative assertion that the one-board wording is absent.

**Why:** a rendered page is many clauses concatenated, and `Contains` cannot say
which one matched. The more thorough the renderer, the likelier some other clause
already carries the substring — so a *richer* output makes the test weaker, which
is the opposite of the intuition.

**How to apply:** when asserting on rendered output, name the region under test
before asserting — slice it, or render only that piece by calling the smaller
function. Add the negative assertion too: the wrong wording must be *absent*, not
merely the right one present. And run the mutation, because this failure is
invisible to review — the test reads exactly like a correct one. Related:
[[a-gate-fixture-must-be-crossable-by-the-thing-under-test]] and
[[fixture-hidden-branch]].
