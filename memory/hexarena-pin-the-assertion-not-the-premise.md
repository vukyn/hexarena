---
name: hexarena-pin-the-assertion-not-the-premise
description: A test whose assertion is general but whose premise is pinned to today's figure reddens on every content change while the thing it guards has not moved
metadata:
  type: feedback
---

`TestTheLastPickOfAFiveASideChoosesFromOne` asserted `atLastPick == slack + 1` —
general, correct at any pool size, written that way from the first line. Above it
sat a guard:

```go
if slack != 0 {
    t.Fatalf("a pool of %d seats a 5v5 with %d to spare, and this test is about
        the pool that seats one exactly", pool.Len(), slack)
}
```

Shipping `pokemon.pichu` took slack from nought to one and **the premise fired
where nothing was wrong**. The behaviour it measures had not changed; only the
number it refused to run at had.

**Why it happened, which is the reusable half:** the repo pins figures
deliberately — a figure that changes a decision is *meant* to redden, and
`TestFiveASideFitsTheShippedCastWithRoomToChoose` is where the slack figure is
pinned on purpose. The mistake was pinning it **twice**. The second copy lived in
a premise, where it looks like a safety check rather than an assertion, so it
never got the "is this figure still the decision?" reading the first copy got.

**How to apply:** when a test needs the world to be in some state, ask whether the
state is the *claim* or merely the *setup*. Setup guards belong at the boundary
that makes the walk meaningless — here `slack < 0`, a pool that cannot seat the
format and so has no last pick at all — not at today's value. And a figure that
changes a design decision gets pinned in exactly one test, named for the decision.
Related: [[fixture-hidden-branch]], [[real-data-can-satisfy-the-property]].
