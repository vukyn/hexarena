---
name: no-arithmetic-test-can-see-a-literal-that-agrees
description: "A test asserting derived == measured cannot tell a derivation from a constant that equals it today — both sides are the same number until a column moves. The only guard that reddens the DAY the literal is written is an AST walk over the declaration; measured on tui.rosterWideWidth (139), where the hardcode mutation left every arithmetic test green"
metadata:
  type: feedback
---

Asked for "a test that goes red if the code stops computing the threshold from
the format", the obvious four tests all pass against `var rosterWideWidth = 139`:

- `threshold == ceilingMeasuredAnotherWay + 1` — both sides are 139.
- `threshold` lands between the two golden windows — 120 < 139 ≤ 160 either way.
- the boundary is inclusive at `threshold` and exclusive at `threshold-1`.
- the helper `rosterWidth` really reads its format argument (fed synthetic
  formats) — true of the helper whether or not the shipped value calls it.

Every one of those compares two numbers that are equal **today**. They only
redden once somebody widens a column, which is a drift detector, not the guard
that was asked for.

**Why:** there is no runtime way to distinguish `f(x)` from a constant equal to
`f(x)` when `x` cannot be varied, and a format string in a `const` cannot be
varied. The distinguishing fact is *syntactic*, so the test has to read syntax.

**How to apply:** `go/parser` the file, find the `ast.ValueSpec` for the name,
and require the value to be an `*ast.CallExpr` on the deriving function whose
first argument is the identifier of the thing it derives from. Measured: the
hardcode mutation gives `rosterWideWidth is assigned a *ast.BasicLit rather than
a call` while every arithmetic test stays green; widening `%-16s` to `%-20s` with
the literal left alone reddens the ceiling test as well (139 against 143). Keep
**both** — the walk catches the literal the day it is written, the arithmetic
catches a derivation fed the wrong format. The repo already does AST-level guards
(`internal/wire`'s clock ban, worldbit's `TestDeterminismLint`), so this is in
style rather than exotic. Related:
[[feedback_an_empty_path_field_is_a_relative_path]],
[[feedback_a_green_expected_mutation_proves_nothing_by_itself]].
