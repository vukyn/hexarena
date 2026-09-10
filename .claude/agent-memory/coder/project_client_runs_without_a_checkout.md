---
name: client-runs-without-a-checkout
description: The 3-step "hexarena-tui runs from a clean go install" work — step 1 (forge.LoadEmbedded) and step 2 (the caller switch, given/not-given rule) done; step 3 is art, and the preview already degrades to the MISSING line rather than crashing
metadata:
  type: project
---

Three steps, on branch `feat/client-runs-without-a-checkout` off `main` at
`009b200`. The defect: `forge.DefaultDataDir` is the **relative** path
`internal/seed/data`, so an installed binary died with `read
internal/seed/data/combat.json: no such file or directory`.

- **Step 1 (merged, `009b200`)** — `forge.Load(dir)` gained a sibling
  `forge.LoadEmbedded()` over one `loadBooks`, and `dataHome` became a *type* so
  `filepath.Join("", …)` cannot silently read the player's working directory.
  → [[feedback_an_empty_path_field_is_a_relative_path]].
- **Step 2 (this work)** — `cmd/hexarena-tui` only. `options.dataGiven` off
  `flag.FlagSet.Visit`, and `loadLibrary`: given → `Load` always; not given +
  directory there → `Load`; not given + `fs.ErrNotExist` → `LoadEmbedded`.
- **Step 3 (open)** — art.

**Why the fallback keys on the probe and not the error:** "load, then embed if
that errored" swallows an author's trailing comma — they play the baked-in books
and are told nothing. Same shape as `testfixture.RequireSharedArt`.

**How to apply, for step 3:** the art preview does **not** need a crash guard and
none was added. An embedded library answers `""` from `ImagePath`, `os.Stat("")`
fails, `ArtStamp` says not-present, and `PreviewScreen.View` draws the ordinary
`i18n.ArtMissing` line on an otherwise normal screen — verified by
`TestTheArtPreviewOverAnEmbeddedLibrarySaysThePictureIsMissing`, which step 3 is
free to move. Two other embedded-library effects worth knowing before step 3:
the header prints `programName + "  " + lib.Dir()`, so it carries **trailing
whitespace** with no directory; and the battle's save key answers
`forge.ErrNoDataDirectory` into `PlayScreen.Err` (a visible error line, not a
crash) because there is nowhere to write a log. The battle itself is unaffected
either way — `model.go` builds its mirror from `seed.Books()` whatever `--data`
says.
