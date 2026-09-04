---
name: name-strip-hides-a-clip
description: hexarena — the width sweep strips authored names (freeNames/withoutNames) out of a line before measuring it, so a NEW row that draws a stage/character name can overflow the terminal and the sweep stays green; the golden's ellipsis is what catches it
metadata:
  type: feedback
---

`TestEveryWordingFitsTheMinimumWidth` (both clients) does **not** measure the
line the terminal draws. It measures `withoutNames(line, freeNames(lib))` — the
line with every authored `character.Name`, `stage.Name`, `origin.Title`,
`built.Name`, `squad.Name`/`squad.ID` cut out of it, longest first. That is
deliberate and right (a name has no promised length), but it means:

**A row whose wording fits the floor only *without* its names passes the sweep
and is clipped on screen.** Measured while adding the squad builder's fork note
(TODO (c) of the forked-line entry): the sweep saw **~104 cells** of wording
after `Poliwrath / Politoed` was stripped, against a 119 budget — green. The
terminal drew **~124** and the frame cut the last clause off with `…`. Nothing in
either client's suite said a word. What said it was reading the **golden diff**
after `make golden`.

So the exemption trap has two halves, not one. The half already written down —
"if your new wording is long, the sweep will say so; fix the wording, do not
exempt it" — only holds for a row of pure program wording. The moment the row
interpolates a data value, the sweep is blind by construction and the check is
manual.

**How to apply.**

- After `make golden`, grep the new golden entries for `Ellipsis` (`…`) before
  accepting them. A clip in a *record* is a finding, not a formatting detail.
  One line of python over the golden does it: split on `=====`, look at the block.
- If a new row carries a name, an id or any value out of the data, **wrap it,
  do not clip it** — and wrap at `MinWidth-3`, not at `UsableWidth()`, because a
  note is prose (§ the width rule: prose wraps at the floor, a data cell spends
  the window). A single clip takes the *end* of a sentence, which is where the
  actionable half usually is.
- A wrapped prose site should be registered as an entry in
  `cmd/hexforge-tui/width_rule_test.go`'s `TestAWideWindowStillWrapsProseAtTheFloor`
  — it has an anti-vacuity guard (`len(floor) < 2` is fatal), so it also tells you
  the day the wording stops being long enough to prove anything. A site whose
  line carries a *value* is the sturdiest entry that table can have, because the
  value and not a shortenable wording is what decides that it wraps.

Related: [[fixture-decides-what-is-visible]], [[two-screen-goldens]],
[[terminal-ambiguous-width-glyphs]].
