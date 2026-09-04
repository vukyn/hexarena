---
name: hexarena-art-picker
description: hexforge-tui art field became a picker over forge.ArtFiles (2026-08-25); bubbles SetValue cursor trap + fixed-floor width elision
metadata:
  type: project
---

hexarena's new-character form takes its art from a chooser over `forge.ArtFiles`
(a recursive, sorted walk of `<data dir>/assets`) instead of a typed path, with a
text-field fallback when nothing is on disk. `cmd/hexforge` keeps `--image` as
free text.

**Why:** a typed path could name a file that was not there, so `hexforge check`
was the only thing that found out. The fallback exists because an empty assets
folder must not be able to block authoring — and it is the only path that can
still produce the "art is missing" write warning, so that warning's TUI coverage
lives in the fallback test now, not in the save test.

**How to apply:**
- Two traps if you touch `cmd/hexforge-tui` again:
  1. **bubbles v1 `textinput.SetValue` does not move the cursor** unless the
     field was empty or the cursor was past the new end (`setValueInternal`).
     Refilling a field a letter at a time (as the art path follows the id)
     strands the cursor where the *first* value ended, and the test helper
     `retype` then deletes only part of it. Call `CursorEnd()` after `SetValue`.
  2. A chooser row showing free-form text must be measured against the
     **`minWidth` floor, never the live window** — measuring the real terminal
     gives one row two lengths and `TestEveryWordingFitsTheMinimumWidth` (which
     renders at width 200 and asserts <= 79) has nothing to hold.
- Measured worst case with elision: exactly 79 cells in both languages; label
  column is 12 vi / 10 en, leaving ~54 cells for the path.
- Related: [[comment-style-generic]] does **not** apply here — this repo wants
  long "why" comments naming the mistake each rule prevents.
