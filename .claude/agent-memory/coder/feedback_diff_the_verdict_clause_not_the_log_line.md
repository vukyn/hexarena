---
name: diff-the-verdict-clause-not-the-log-line
description: When a t.Logf guard is the evidence a data addition is neutral, diff its VERDICT CLAUSE against a git-archive run — the fielded-by list grows on every addition, so a whole-line diff always "changed"
metadata:
  type: feedback
---

When a **logging** guard (`t.Logf`, asserts only that it measured something) is
the evidence that a data addition did not move a property, compare the guard's
**verdict clause** against a `git archive HEAD` run of the same test — not the
whole log line.

**Why:** `s06` (`DAT-015`, 2026-09-09) had one deciding authoring choice — rows
`{0,2}` rather than `{0,1}` in both stacked columns, so `splash_power` stays
unreachable and `DAT-007`'s premise does not move. The only signal that the
choice was right is one clause inside a line that changes on *any* addition:

```
column  catches 2 on roster/ally, and 1 on every board a rate is read off
        (fielded by s01/clefable, …, s06/garchomp, s06/dugtrio, s06/blissey, …)
```

The fielded-by list gained three names by construction — a five-unit squad
carrying `column` casts *must* appear there. Read as a whole line that is
"changed", which is indistinguishable from the failure case (`{0,1}` would have
rewritten the clause to say the shape catches 2 on a fought board). Read as a
clause it is unchanged, which is the proof. `TestAShippedFormationIsCatchable…`
and `TestEveryBonusIsMeasuredAgainstEveryShippedFormation` are both this shape.

**Corollary — the reporting branch shows up as a LINE NUMBER.** The bonus guard
reports through two `t.Logf` sites, so the move it was run to detect reads as
`bonusboard_test.go:264 → :256`:

```
:264  ground_root  rungs 2/3  reached by NO shipped formation, of the 7 walked
:256  ground_root  rungs 2/3  reached by s06 (ground ×3)
```

That is the cheap confirmation that the *other* branch actually ran, and it is
the same trap as [[a-hand-built-arm-can-miss-the-reporting-branch]] seen from
the reading end rather than the writing end.

**How to apply:** before running the guard on the dirty tree, `git archive HEAD |
tar -x -C $(mktemp -d)` and run it there (never a shared stash —
[[a-golden-can-be-red-before-you-touch-it]]). Then diff, and name in the report
*which clause* did not move and *which list* did. Also: a pure `internal/seed/data`
JSON change scores `risk 0.00 / 0 test gaps` in `detect_changes_tool` — the graph
does not index data files, so that zero means "not seen", not "not risky", and the
guard clauses are the real self-review. → [[graph-rename-blind-spots]]
