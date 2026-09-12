---
name: go-fmt-pads-a-string-verb-by-runes
description: "%-21s pads by RUNES not bytes, so the Vietnamese summon name `phân thân 1` (11 runes / 13 bytes) is already aligned in the roster golden — a width bound on a name must use utf8.RuneCountInString, and elided's len() is the odd one out"
metadata:
  type: feedback
---

`fmt.Sprintf("%-21s", s)` pads to 21 **runes**, not 21 bytes. Verified against
the committed golden rather than assumed: `cmd/hexforge-tui/testdata/screens.golden`
draws a summoned unit called `phân thân 1` — 11 runes, 13 bytes — and the health
bar's `[` lands at **rune index 26 / byte index 28**, i.e. the table is aligned
by runes and the byte offset is the one that looks wrong.

**Why:** it matters the moment a column is sized to the longest name it can hold
with no slack. Counting a bound in bytes would refuse `phân thân 1` (13 against a
13-cell budget it actually fits in 11), and there are two independent Vietnamese
sources of names here — a skill's `summons.name`, and any character or stage name
an author writes.

**How to apply:** measure a name bound with `utf8.RuneCountInString`, never
`len`. ⚠️ `internal/tui`'s `elided` uses `len(part)` (bytes) for the effects
column — that is not a bug today only because every status id is ASCII, and it is
the first thing to change if a status id ever carries a diacritic. Separately,
rune count is still not *display* width: see
[[feedback_terminal_vietnamese_glyph_width]] for glyphs a terminal draws wider
than the one cell they measure.
