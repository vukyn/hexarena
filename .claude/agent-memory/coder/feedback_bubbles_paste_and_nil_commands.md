---
name: bubbles-paste-and-nil-commands
description: The paste fix's three measured traps — textinput.Paste's message is UNEXPORTED so ctrl+v cannot use it; PasteInto's command is nil on a plain terminal so it is not a success signal; and testing a sanitiser is not testing that its caller calls it
metadata:
  type: feedback
---

Three things measured while making paste work in both TUI clients. Each looked
settled from reading and was wrong or incomplete until run.

## ⚠️ `textinput.Paste` cannot be wired from outside `package textinput`

`bubbles/v2`'s default key map **already binds `ctrl+v`**, and a `ctrl+v` that
reaches a focused field returns `textinput.Paste` as its command — which really
does shell out to `pbpaste`. But that command's message is **`textinput.pasteMsg`,
an unexported type**. A model in another package cannot name it in a type switch,
so it arrives at `Update`, matches nothing, and dies; the field never sees it.
Measured end to end: `ctrl+v` on the join screen read the clipboard and inserted
nothing.

**Why:** so "wire ctrl+v to `textinput.Paste()`" is not achievable as stated. The
only way to route it would be forwarding every unnamed message into whatever
field is focused, which is a far wider door than the feature needs.

**How to apply:** read the clipboard yourself (`github.com/atotto/clipboard`,
already in the tree via bubbles) and return **`tea.PasteMsg`** — the same message
a terminal's bracketed paste delivers — so both routes converge on one insert.
`ctrl+v` must also be answered **before** the screen switch, or the field fires
`textinput.Paste` as a second, wasted clipboard read.

## ⚠️ A bubbles command is not a signal that anything happened

`field.Update(...)`'s returned `tea.Cmd` is the **cursor's blink**. `NewInput`
turns the virtual cursor off on a plain terminal, and every test here runs under
`NO_COLOR` — so a paste that landed perfectly hands back **nil** on every machine
the suite runs on. Using `command == nil` as "refused" shipped for one commit: the
level field took `"42"` and the member stayed at sixty, and every assertion about
the *field* passed.

**Why:** the state and the command are independent, and the command is the one
that is environment-dependent.

**How to apply:** decide "did it land" by comparing the field's **value** either
side. Never by the command. If a test wants to assert on a command, dress the
field with `NewInput(false)` — under `NewInput(true)` a command assertion is
vacuous in both directions.

## ⚠️ Holding a helper equal to a library says nothing about the caller using it

`PasteText` (the sanitiser for the one text target that is a plain `string`, the
skill filter) was pinned equal to a real `textinput`'s answer over a dozen inputs.
Replacing `PasteText(text)` with a bare `text` in `pasteFilter` left the **whole
suite green**: the equality test exercises the helper directly, and every other
filter test pasted a string with nothing to sanitise. A query holding a real
newline draws as two rows on a screen that budgets one.

**Why:** two claims — "the rule is right" and "the caller applies the rule" — and
only the first was tested. Found by mutation, not by reading.

**How to apply:** when a rule lives in a helper, one test per **caller** must
paste something the rule actually changes. It also generalises: a fixture that
only ever feeds clean input cannot see a sanitiser being skipped.

Related: [[a-well-formed-measurement-can-measure-nothing]],
[[fixture-decides-what-is-visible]], [[pty-smoke-test-for-hexarena-tui]].
