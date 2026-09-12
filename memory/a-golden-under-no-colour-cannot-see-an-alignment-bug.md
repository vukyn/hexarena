---
name: a-golden-under-no-colour-cannot-see-an-alignment-bug
description: fmt pads by runes and escape codes are runes, so a coloured cell gets no padding and every coloured row slides — and every golden stays green, because they are all taken under NO_COLOR
metadata:
  type: feedback
---

The roster's element column draws a code in that element's colour. Written the
obvious way — hand the styled string to `%-8s` — **every coloured row slides
left**, because `fmt` pads by rune count and an escape sequence is runes.

**And nothing in the design record can see it.** All three `screens.golden` are
taken under `NO_COLOR` with `Palette.Plain` true, so the recorded rendering has
no escapes in it and lines up perfectly. Measured: padding over the inked string
instead of the plain one reddens exactly **one** test —
`TestAnInkedElementCellIsTheSameWidthAsAPlainOne` — while `internal/screen` and
`cmd/hexarena-tui` both stay `ok`.

So the fix is that the cell pads **its own plain content** and the format verb
stays only because the bound tests read the column width off it.

**Why:** a golden is a picture of one rendering. Taking every golden under
`NO_COLOR` is right — it keeps the record readable and diffable — but it means
the record is blind to every defect that only exists when colour is on.
Alignment is the obvious one; anything measured in cells is another.

**How to apply:** when adding ink to a column, ask what the golden would look
like if the ink were wrong. If the answer is "the same", the golden is not the
guard and a unit test has to be — one that renders with ink and compares widths
against the plain rendering. Same shape as
[[a-guard-on-the-wrong-side-of-the-gate]]: the record fires where the defect
cannot reach it.

⚠️ Related but not the same as the matchup marks' rule, which is that meaning
must live in text because the record cannot see colour. This one is the reverse
direction: the *text* is fine and the **geometry** is broken, and the record
cannot see that either.
