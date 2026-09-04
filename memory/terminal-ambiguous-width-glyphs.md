---
name: terminal-ambiguous-width-glyphs
description: "TUI trap — ⌘ ⌃ ⇧ ⌥ are East-Asian-Ambiguous width: measured 1 cell, drawn 2, glyph overlaps the next char"
metadata: 
  node_type: memory
  type: reference
  modified: 2026-08-26T16:24:52.522Z
---

⌘ (U+2318), ⌃ (U+2303), ⇧ (U+21E7), ⌥ (U+2325) are **East-Asian-Ambiguous width**. `lipgloss.Width`/uniseg count them as **1 cell**; many terminals draw them **2 cells wide with a 1-cell advance**, so the glyph is painted **on top of the character after it**. `⌘S` renders as ⌘ and S overlapping — not "crowded", genuinely unreadable.

**A program cannot detect which behaviour its terminal has.** No escape query for it.

Adding a space (`⌘ S`) fixes the overlap but costs a cell you usually don't have, and the drawn line is then 1 cell wider than every measurement says — which can wrap the row.

**Rule: keep key labels ASCII in a TUI footer.** Say `ctrl+s` / `cmd+S`, not the symbols. Put the symbol in prose (README, a note line) if at all.

hexarena hit this twice: PR #55 added a space thinking it was crowding (wrong diagnosis, ⚠️ shipped without testing the visual), PR #57 (2026-08-26) dropped the glyph. Footer now says `ctrl+s` on every platform; ⌘S still saves and is announced in `MenuNote`. Guarded by `TestTheSaveLabelIsDrawableEverywhere` (per-letter `> 127` check, so the next tempting symbol fails too).

⚠️ Lesson beyond the glyph: **a rendering complaint cannot be verified from a test suite or a width calculation** — both said the label was fine. Ask for a screenshot of the fix before calling it done.

Related: [[bubbletea-v2-silent-breaks]], [[hexarena-tui-i18n]]
