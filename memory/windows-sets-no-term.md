---
name: windows-sets-no-term
description: "No native Windows terminal sets $TERM, so `TERM == \"\"` must mean dumb only when GOOS != windows — a colour check read it as dumb and drew every Windows session plain"
metadata: 
  node_type: memory
  type: reference
  modified: 2026-08-29T14:26:55.062Z
---

**No native Windows terminal sets `$TERM`.** It is terminfo's convention; cmd.exe, PowerShell and Windows Terminal have no terminfo, so an empty `TERM` there carries **no information at all**. Treating it as a dumb terminal — the correct reading on POSIX — silently downgrades every Windows session.

Shipped in hexarena `cmd/hexforge-tui/style.go` and fixed in **PR #158** (`b67ccc7`, 2026-08-29). The bug cost three things at once, not just the palette: every lipgloss style collapsed to the identity, the art preview drew its **monochrome ramp** instead of the colour blocks (the one screen where colour is information), and the virtual cursor was off in every text field.

**The right rule already exists in the dependency tree — copy it, don't guess.** `github.com/charmbracelet/colorprofile` (pulled in by bubbletea v2) writes it in `colorProfile`:

```go
isDumb := (!ok && runtime.GOOS != "windows") || term == dumbTerm
```

and `envColorProfile` explains itself: *"Use Windows API to detect color profile. Windows Terminal and cmd.exe don't define $TERM."* It reports **TrueColor** for a Windows 10 build ≥ 14931 and for any `WT_SESSION`. Consequence worth knowing before reaching for a fix: **bubbletea/lipgloss never strip colour on Windows**, so a program drawing plain there is doing it to itself.

**How to make it testable:** `runtime.GOOS` cannot be faked, so a rule whose answer differs by platform must take its inputs as **parameters** — `plainScreen(noColour, term, goos string) bool` — with the env-reading wrapper kept thin. Then both answers are assertable from either sort of machine.

⚠️ **Why no test could have caught it, which generalises past this bug.** Every test in that package sets `NO_COLOR` (deliberately — so assertions match a word rather than a word wrapped in escape codes), and `NO_COLOR` is asked **first** and returns. The `TERM` branch was unreached on the machine it was written on and unreachable in CI. **A guard whose early return is set by every test fixture makes everything below it dead code to the suite.** Cross-compiling (`GOOS=windows go build ./...`) proves it compiles and says nothing about which branch runs.

Related: [[bubbletea-v2-silent-breaks]], [[terminal-ambiguous-width-glyphs]].
