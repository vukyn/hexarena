---
name: bubbletea-v2-silent-breaks
description: "bubbletea v1→v2 migration — module moved to charm.land, and 3 changes that break silently (space, colour, cursor)"
metadata: 
  node_type: memory
  type: reference
  modified: 2026-08-26T13:07:46.067Z
---

hexarena PR #52 (merged 2026-08-26) migrated `cmd/hexforge-tui` to bubbletea v2, to read ⌘S.

**Modules moved house**: `github.com/charmbracelet/{bubbletea,bubbles,lipgloss}` → **`charm.land/{bubbletea,bubbles,lipgloss}/v2`**. `go get github.com/charmbracelet/bubbletea/v2` fails with "module declares its path as charm.land/bubbletea/v2".

**Why v2 at all**: Command (⌘) is not a modifier the classic terminal escape sequences encode — it never reaches the program, and v1's `tea.Key` had only `Alt`. Kitty keyboard protocol reports it; v2 parses that protocol and requests basic disambiguation **by default**, so ⌘ arrives as `tea.ModSuper` / `"super+s"`.
⚠️ Still not universal: terminal must speak the protocol (kitty, Ghostty, WezTerm, foot, iTerm2+CSI u — **Terminal.app never**), must pass ⌘S instead of opening its own Save dialog, and Linux WMs may claim Super first. Always keep ctrl+s as the binding that works.

Compile-time breaks:
- `Model.View() string` → `View() tea.View`; alt screen is `view.AltScreen = true`, **not** `tea.WithAltScreen()` (removed from options).
- Key is `Code rune` / `Text string` / `Mod KeyMod`, not `Type`/`Runes`. `tea.KeyMsg` → `tea.KeyPressMsg`. No `KeyType`, no `KeyRunes`, no `KeyCtrlS`.
- `textinput.Width = n` → `SetWidth(n)`.
- `lipgloss.Color` is `func(string) color.Color`, not a type.

⚠️ **Silent breaks — compile fine, behave wrong:**
1. **A bare space stringifies as `"space"`, not `" "`.** `uv.Key.String` returns `Text` only when `Text != " "`, so space falls to `Keystroke()` and comes out named. Every `case " "` matched nothing.
2. **Colour is the program's job now.** lipgloss v2 writes escapes unconditionally; the *program* downsamples. A `textinput` on default styles **keeps its colours under NO_COLOR**. Fix: build unstyled `textinput.Styles` yourself and `SetStyles`. v1's renderer did this for free via TTY detection.
3. **Virtual cursor = reverse video** (hardcoded in `cursor.Model.View`), i.e. an escape code. If "no escapes on a plain terminal" is a contract, `SetVirtualCursor(false)` when plain — v1 stripped the attribute itself so the plain path never had a cursor either.

Related: [[hexarena-cast-authoring]], [[hexarena-tui-i18n]]

⚠️ **Thứ NĂM (2026-09-04)**: paste giao bằng **`tea.PasteMsg`**, không phải chuỗi `KeyPressMsg` — model chỉ switch `KeyPressMsg` sẽ nuốt im lặng mọi lần dán. → [[hexarena-paste-both-clients]]
