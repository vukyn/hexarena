---
name: fmt-cannot-pad-an-inked-cell
description: "%-8s pads by RUNES, and escape codes are runes — so a coloured cell gets no padding and the whole column shifts, invisibly, because every golden is NO_COLOR"
metadata:
  type: feedback
---

A `fmt` width verb pads to a **minimum rune count**. A styled string carries
escape sequences, which are runes a terminal never draws, so `%-8s` handed
`\x1b[32mgra\x1b[0m` sees 13 runes, decides the field is already over eight and
pads **nothing** — the cell draws three visible cells where the column is eight,
and every row after it on that line slides five cells left.

**Why:** this is the defect that ships. The goldens in this repository are taken
under `NO_COLOR` with `Palette.Plain`, so the recorded table is the *uninked*
one and is perfectly aligned; the misalignment exists only in the drawing a
reader with a colour terminal gets, which is every reader and no test. Two
separate guards were already in place for the column's width
(`TestEveryAffinityTheRosterCanDrawFitsItsColumn`, the ceiling derivation) and
both measure plain strings, so both stay green.

**How to apply.** When a table cell is styled by the caller:

- **Pad the plain content in the renderer, not in the format verb.** Build the
  cell, count the *codes* (never the returned string), append the spaces
  yourself, to the column width **plus its gap** — padding to the column alone
  leaves the verb to add the gap, and the verb will not.
- Leave the verb in the format anyway. It is what the bound tests read the
  column width off, and it still pads a plain cell that arrived short.
- **Ink the text and never the padding**, or the plain width stops being
  knowable without asking the ink what it did.
- The test has to strip the escapes and compare the remainder against the plain
  cell, and it must first assert an escape is actually there — a nil ink makes
  every such claim vacuously true. See
  `TestAnInkedElementCellIsTheSameWidthAsAPlainOne` in `internal/tui`.

Related: [[feedback_go_fmt_pads_a_string_verb_by_runes]] is the same arithmetic
seen from the other side (a Vietnamese label is fewer cells than bytes);
[[feedback_two_screen_goldens]] for why NO_COLOR makes the coloured drawing
unrecorded everywhere.
