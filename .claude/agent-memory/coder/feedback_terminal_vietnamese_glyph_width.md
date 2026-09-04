---
name: terminal-vietnamese-glyph-width
description: The user's terminal draws some precomposed Vietnamese letters double-width; do not "fix" apparent column drift in code
metadata:
  type: feedback
---

When the user reports a stray space or a drifted column in Vietnamese TUI
output, suspect their terminal font before the code. They confirmed one case
themselves: `chịu được` looked like it had a stray space, and the source was
correctly precomposed UTF-8 (`ị` is U+1ECB, one rune), with `lipgloss.Width`,
`utf8.RuneCountInString` and `runewidth.StringWidth` all agreeing it is one
cell. Their terminal renders that glyph as two cells.

**Why:** the user explicitly ruled out a workaround — "Do not add a workaround
for it and do not change how widths are measured." A per-glyph fudge would put a
second, wrong width model next to the one every padding and clipping calculation
in the client already uses, to compensate for one machine's font.

**How to apply:** verify with the width functions first and say what they
measured. If they agree the string is one cell per letter, report it as a
rendering artifact, do not change the measurement. When a rename or rewording
happens to remove the glyph, that is a side effect, not the fix — say so. It is
worth reporting which other Vietnamese strings still sit in *padded* columns
(the ones where a double-width glyph actually misaligns something), since the
same font will likely do it to letters in the same Unicode block —
U+1EAD, U+1EC3, U+1EC7, U+1ED9 and friends are all over the labels. See
[[vietnamese-ui-copy-style]].
