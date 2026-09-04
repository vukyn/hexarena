---
name: pty-smoke-test-for-hexarena-tui
description: hexarena-tui refuses a pipe, so smoke-testing it needs python pty.fork + TIOCSWINSZ; two of them plus hexarena-host play a whole PvP match from a shell
metadata:
  type: feedback
---

`cmd/hexarena-tui` and `cmd/hexforge-tui` refuse to start unless stdout is a
character device, so `echo keys | ./bin/hexarena-tui` gets the "not a terminal"
refusal and nothing else. A real end-to-end smoke test needs a **pty**.

**Why:** the suite drives `model.Update` headlessly and never starts a program,
so anything between `main` and the model — flag parsing, the session's attach,
`run`'s defer, the real socket — is unmeasured by `go test`. A digest bug that
made a notice fire on every unedited data directory got through the whole gate
and showed up on the first pty run.

**How to apply** (macOS, python3, no extra deps):

- `pid, fd = pty.fork()`; child sets `TERM=xterm-256color` and `os.execv`s the
  binary. `LINES`/`COLUMNS` are **not** enough — bubbletea reads the pty, so set
  the size with `fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", rows,
  cols, 0, 0))` and give it at least 120x24 or every screen is the too-small one.
- Drain with `select.select([fd], [], [], 0.2)` in a loop; strip escapes with
  `re.sub(r'\x1b\[[0-9;?]*[a-zA-Z]|\x1b[()][B0]|\x1b[=>]|\x1b\][^\x07]*\x07','',raw)`.
  Arrow keys are `\x1b[B` / `\x1b[C`, enter `\r`, quit `\x03`.
- **Two ptys plus `./bin/hexarena-host` play a whole match**: spawn both, walk
  each to the menu's join entry, type the same code, then hammer `\r` at both in
  a loop — enter answers whichever turn is open and is dropped when neither is.
  Break on **both** clients showing the result, not the first: one seat finishes
  a beat before the other and killing early makes the host report an abandoned
  match that never happened.
- ⚠️ **The screen repaints CHANGED CELLS, not lines.** A ticking countdown comes
  through as `\x08\x088 \x08\x087 ` — backspaces and two digits — so a capture
  that splits the stripped stream into lines and greps sees the first full draw
  and then nothing, which reads exactly like "the feature does not update". To
  read a whole line again, force a repaint: `TIOCSWINSZ` to a different width
  then `SIGWINCH` to the child. To see it *move*, grep the raw stream for digits.
- The shipped `squads.json` sides are refused by a 3v3 room for **two**
  independent reasons — **two units**, and no `stage`, so they are not a leaf of
  the line at level 60. Both come back as one `squad_refused`. The join screen
  also opens on the **first** squad on the file, so appending a legal side is not
  enough: build the legal ones into a scratch `--data` copy and delete the rest.
- A scratch program that needs `internal/...` can live **outside** the repo,
  which leaves the worktree clean: a module named `github.com/vukyn/hexarena/…`
  with `replace github.com/vukyn/hexarena => <worktree>` satisfies the internal
  rule (it is checked on the import path prefix). The in-tree `_`-prefixed
  directory still works and needs deleting afterwards.
- Kill the port when done: `lsof -ti :13579 | xargs kill -9`.

Related: [[a-well-formed-measurement-can-measure-nothing]].
