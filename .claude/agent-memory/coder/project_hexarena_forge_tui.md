---
name: hexarena-forge-tui
description: hexarena's first third-party dep (bubbletea) confined to cmd/hexforge-tui by an import-graph test; plus bubbletea/pty gotchas that cost debug cycles
metadata:
  type: project
---

hexarena gained `internal/forge` (shared authoring logic) plus `cmd/hexforge-tui`
(bubbletea) on 2026-08-25, breaking its "standard library only" rule for the
first time. `cmd/hexforge` stayed as-is.

**Why:** a full-screen client cannot run with stdin as a pipe, so the CLI is
still what scripts and the end-to-end tests use; and two front-ends restating
the same rule is the exact failure this repo has already hit twice (see its
CLAUDE.md "One source for a recorded string"). The dependency is confined to one
directory and that confinement is mechanical:
`TestOnlyTheFullScreenClientHasThirdPartyImports` in `internal/forge` parses
every `.go` file with `go/parser` (build tags cannot hide an import) and fails if
a non-stdlib, non-module import appears outside `cmd/hexforge-tui`. It also fails
if the TUI stops importing bubbletea, so it can't go vacuous.

**How to apply:** before adding any import to hexarena outside the TUI, expect
that test to fail — that is intended, not a broken test. Verify a boundary test
like this by adding a probe file that *compiles* (`var _ = pkg.Symbol`), not by
an import that fails to build.

Gotchas that cost real time, none obvious from the code:

- **bubbletea v1.3.x blocks at startup querying the terminal** — it emits OSC 11
  (`\x1b]11;?`) and DSR (`\x1b[6n`) and waits. A headless pty smoke test must
  write back `\x1b]11;rgb:0000/0000/0000\x1b\\` and `\x1b[1;1R` or the program
  hangs before drawing anything and the pty read dies with EIO.
- **`textinput.Update` returns a non-nil blink command**, so "did this key quit?"
  cannot be `cmd != nil`. Run the command and check for `tea.QuitMsg` — but only
  on screens with no text field, since a blink cmd is a timer that would sleep.
- **`textinput.View()` pads to `input.Width`**, so the field's Width — not the
  value — sets the row width and therefore the minimum terminal width.
- A multi-rune `tea.KeyMsg` stringifies to that word, so typing "up" in one
  message routes as the up arrow. Test helpers must send one rune per message.

Related: [[hexarena-onboarding-fixes]].
