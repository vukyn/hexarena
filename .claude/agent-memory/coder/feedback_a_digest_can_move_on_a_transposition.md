---
name: a-digest-can-move-on-a-transposition
description: hexarena — the seed-11 whole-battle digest moved with all 255 events byte-identical and only two of them swapped; dump and diff the event stream before writing a balance sentence about a new digest
metadata:
  type: feedback
---

`internal/seed`'s `theWholeBattleDigest` is `wire.DigestEvents` over the seed 11
event slice, so it is a hash of a **sequence**. An adjacent transposition is a
new digest and no change to the game whatsoever — and the failure message
("if the balance data changed, update the constant") reads exactly the same in
both cases.

**Measured (2026-09-11, the per-strike drain).** Moving the drain from after the
strike loop to inside it puts a `healed` ahead of the riders that used to
precede it. Seed 11 still runs to **255 events**, every one byte-identical, and
the whole diff is one `status_resisted` moving one slot later. The screen golden
said the same thing in two lines (vi + en), amount `112`, share `70%`, `920 hp
left` — all unchanged, only the line's position.

**The cheap way to tell a transposition from a balance move**, and it takes two
minutes: drop a throwaway `_test.go` into `internal/seed` on the dirty tree and
on a `git archive HEAD` copy, have each dump `fight.Drain()` as
`json.MarshalIndent` to a path from an env var, then

```
diff <(python3 -c "...print one sorted-keys json line per event...") <(same for after)
```

A reordering is `N` deleted and `N` inserted lines **with identical content**; a
balance move is different `amount` / `remaining` figures. Say which in the
report — "the digest moved" is not "the battle moved", and only one of them is
worth a paragraph.

Write the finding into the constant's doc comment, which is where this
repository keeps its measurements: the comment already carried the previous
occasion (the per-strike reply, which *did* move the battle), so the two
readings now sit side by side and the next reader can tell them apart.

Related: [[a-golden-can-be-red-before-you-touch-it]] (prove the golden was green
at HEAD first — both of these were), [[pin-comparison-base-to-worktree-head]].
