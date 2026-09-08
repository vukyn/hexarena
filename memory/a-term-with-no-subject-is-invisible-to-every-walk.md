---
name: a-term-with-no-subject-is-invisible-to-every-walk
description: a schema term shipped before its first shipped user is unreachable by every golden and every shipped-book walk in this repo, so its only instruments are three hand-built carriers
metadata:
  type: feedback
---

⚠️ **A term added to a book's schema before any shipped entry uses it is invisible to the entire golden and shipped-walk apparatus.** Measured on `ENG-006` step 1 (`passive.Condition.AboveHealth`, 2026-09-08): mutating `Condition.Threshold()` to answer the wrong end of the health bar left **`internal/seed` green and every shipped walk in `internal/i18n` green**, `describe.golden` did not move, and only hand-built fixtures went red. `make check` would have shipped the mutation.

**Why:** the walks in this repo are deliberately built out of the shipped books (`seed.PassiveBook()`, `lib.Passives().All()`, `describe.golden`) — which is what makes them catch a *shipped* mistake without anybody writing a case. A term nothing ships is not in the book, so the loop over the book runs zero iterations that touch it and passes by walking past it. That is `[[fixture-hidden-branch]]` and `[[real-data-can-satisfy-the-property]]` in their sharpest form: not "the fixture does not exercise the branch" but "**no fixture in the repository can**".

**How to apply.** When a plan says "the term lands now, the first user lands later", accept that the whole normal apparatus is off and build the carrier yourself, in each layer that renders or prices the term. The three that were needed here, which are the three shapes this repo has:

- **A hand-built value** where the package takes one — `internal/i18n` takes a `passive.Passive` literal, so the test constructs the trait and calls `DescribePassive` on it. Cheapest; skips the parse.
- **A scratch data directory** where the package only ever sees a parsed book — `cmd/hexforge` renders off `forge.Load(dir)`, so the test appends the declaration to `passives.json` inside the per-test `t.TempDir()` copy. This is the shape that also exercises the parser. Do **not** put it in `internal/testfixture` instead: a fixture-cast edit is paid for in goldens (`[[hexarena-shipping-a-character]]`).
- **The package's own fixture book** where one already exists — `internal/core/battle`'s `books(t)` holds a JSON passive book that no golden reads, so two traits were added to it for a price of nothing.

And **say in the entry and in the test comment that no golden can record this yet**. "Covered by a golden" and "no golden can see it" look identical from a green suite; only the sentence tells them apart. When the first subject ships, the goldens moving *is* the design record (`[[hexarena-descriptions-are-derived]]`).

See `[[a-shipped-book-walk-catches-arrivals-not-departures]]` for the neighbouring failure — a walk that proves shipped ⊆ table says nothing about the other direction.
