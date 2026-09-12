---
name: the-element-column-is-a-code
description: "The wide roster draws 3-letter element IDS inked per half; threshold fell 139 -> 131 (still above the 120 floor) and the permanence rule moved to the statuses catalogue"
metadata:
  type: project
---

Branch `feat/the-element-column-is-short-and-coloured`, worktree
`hexarena-elem`, off `main` at `8d96844`. Two changes the owner asked for:

**1. The roster heading lost its parenthesis.** `RosterHeadingEffects` is now
`effects` / `hiệu ứng`. The rule it carried — an effect with no countdown beside
it is permanent — moved to `i18n.StatusesNoCountdown`, a second dim foot line on
the **statuses catalogue** (`internal/screen/statuses.go`), above the existing
caveat. `statusesRoom`'s `below` went 6 → 7, so the floor draws eleven statuses
where it drew twelve.

**2. The element column is a three-letter code in the element's own colour.**
`tui.ElementCode` is the first three runes of the element's **id**, so it is an
id shortened and stays the same in both languages — which is the rule
`internal/i18n`'s doc comment already states for element ids, and which the gloss
table (`grass/electric <cỏ/điện>`, beside the id and never instead of it) exists
to preserve.

**How to apply.** Four facts worth having before touching this again:

- **The ink is a FUNCTION handed in, not a palette.** `tui.ElementInk` is
  `func(element.Element, string) string`; `Palette.ElementInk()` supplies it.
  `internal/tui` keeps the table and the caller keeps the ink — the same
  division `Detail` makes with a language. A nil ink draws plain.
- **The reading carries the inked table**, and `playReading`'s "unstyled" rule
  survives: a palette is built once at boot and cannot change, where a **width**
  changes between one turn and the next, which is the whole reason the width
  decision stayed at draw time. Nothing touches `p.Fight` at draw.
  → [[project_the_roster_row_widening]]
- **The numbers moved on their own, off the format.** `rosterElementRoom` 15 → 7
  (the bound is `gra/gro`, two codes and a slash, since `element.Dual` admits no
  third half); the wide row's ceiling 138 → **130**; the derived threshold 139 →
  **131**. ⚠️ Still **above the 120 floor**, so the narrow table remains what
  every floor window draws and the golden diff stays readable — at 119 the wide
  table would have become universal, which is a far bigger change than was asked.
- **Goldens that moved:** the battle sections of all three `screens.golden` (the
  narrow heading line, one per section; seven lines per wide 160x60 section),
  `internal/tui/{crowded,opening}.golden` (the narrow heading), and the statuses
  section of all three (one line: `stun` fell off the floor listing, the notation
  line arrived). Nothing else.

⚠️ Change 1 moves **narrow** goldens too, unavoidably — the brief expected "the
wide ones and the statuses screen", and the narrow heading is a third.

Related: [[feedback_fmt_cannot_pad_an_inked_cell]],
[[feedback_the_screen_cast_carries_no_dual_affinity]],
[[feedback_a_var_states_its_type_on_the_literal]].
