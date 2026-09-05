---
name: hexarena-a-mutation-the-parser-refuses-runs-nothing
description: hexarena — the data books validate hard, so the obvious mutation on a passive or skill is often refused at parse time; a mutation that cannot load proves nothing about the test
metadata:
  type: feedback
---

Checking `TestTheShippedSnowballReachesItsCapInARealBattle`, the obvious mutation
was to empty `renews` on the `quickening` trait. It does not redden the test. It
produces:

```
load the shipped books: passive "quickening": grants nothing, renews nothing,
resists nothing, adds nothing, answers nothing, drains nothing, converts nothing,
spares nothing and amplifies nothing, so holding it would change nothing
```

`passive.ParseBook` refuses a trait that does nothing, so the books never load and
every test in the package fails for the same reason. **That is a parse error
wearing a test failure's clothes** — it says nothing about whether the assertion
under examination can see the defect it names, which is the whole point of running
a mutation. Same family as [[mutate-the-producer-not-just-the-logic]]: a
compile-failing mutation proves nothing, and this repo's data books validate hard
enough that the same trap exists one layer out, in JSON.

**What to reach for instead — keep the shape legal and change the meaning:**

| instead of | do |
|---|---|
| emptying a passive's only effect | point it at a **different** status (`stoked` → `haste`) |
| deleting a field the parser requires | move it out of range *within* what parses, or off by one |
| removing a status | give it `duration: 1`, or one stack fewer than the claim needs |

Both replacements above ran and reddened exactly the assertion they were aimed at
— deepest stack **1 of 5** for the duration, **nought** for the redirect — which
is what let the comments in that test name a figure rather than a hope.

⚠️ **And say so in the comment when a mutation is unavailable.** The next reader
reaches for the same obvious one; a line saying it cannot be run, and why, saves
them the round trip and stops them concluding the test is weak.
