---
name: client-runs-without-a-checkout
description: The 3-step "hexarena-tui runs from a clean go install" work — all three done (LoadEmbedded, the caller switch, the art wording); the state a library with NO directory puts every screen in, and what each now says
metadata:
  type: project
---

Three steps. The defect: `forge.DefaultDataDir` is the **relative** path
`internal/seed/data`, so an installed binary died with `read
internal/seed/data/combat.json: no such file or directory`.

- **Step 1 (merged, `009b200`)** — `forge.Load(dir)` gained a sibling
  `forge.LoadEmbedded()` over one `loadBooks`, and `dataHome` became a *type* so
  `filepath.Join("", …)` cannot silently read the player's working directory.
  → [[an-empty-path-field-is-a-relative-path]].
- **Step 2 (merged, `b4dcc60`)** — `cmd/hexarena-tui` only. `options.dataGiven`
  off `flag.FlagSet.Visit`, and `loadLibrary`: given → `Load` always; not given
  + directory there → `Load`; not given + `fs.ErrNotExist` → `LoadEmbedded`.
  **Why the fallback keys on the probe and not the error:** "load, then embed if
  that errored" swallows an author's trailing comma — they play the baked-in
  books and are told nothing.
- **Step 3 (branch `feat/art-that-was-never-shipped`, 2026-09-10)** — the art,
  below.

**The state, and what step 3 made each screen say.** The art is **not**
embedded (66 files, 16 MB) and is staying out, so a library with no directory
has every book and no picture. That is a *different* state from a directory
whose picture has not been drawn yet, and the two now say different things,
told apart by a new `forge.Library.HasDataDirectory()` rather than by
`Lib.Dir() == ""` — a display string's emptiness is a formatting accident, and
reaching `Library.home` is what puts the question inside
`TestEveryLibraryAccessorThatReachesTheDataDirectoryDecidesWhatNoDirectoryMeans`,
which then demanded its `homeDecisions` row (the walk working, as designed).

| where | a directory, no picture | no directory at all |
|---|---|---|
| `preview.go` | `ArtMissing` / `THIẾU`, bad | `PreviewArtNotShipped`, dim |
| `browse.go` `artLine` | `ArtMissing`, bad | `ArtNotShipped` / `không kèm theo`, dim |

Both preview states are recorded in `internal/screen/testdata/screens.golden` as
an adjacent pair; the browser's two are deliberately not, and that reasoning is
a comment at the golden entry rather than only in a report → [[two-screen-goldens]].

⚠️ **Four sites, not the three the spec listed** — `artLine` was called the
authoring tool's own and is drawn by both clients →
[[a-shared-screen-has-no-tool-only-wording]].

Three more effects of the same state, all now handled:

- the header printed `programName + "  " + lib.Dir()`, so with no directory it
  welded two trailing spaces onto every screen; the separator moved inside the
  `dir != ""` branch. `cmd/hexforge-tui`'s identical line was **left alone** —
  `forge.Load` refuses an empty directory, so an authoring tool always has one;
- the battle's save key answered `forge.ErrNoDataDirectory` into
  `PlayScreen.Err`. The offer is **withdrawn** instead: two second footer
  wordings (`PlayNoSaveFooter`, `PlayOverNoSaveFooter` — never the local ones
  with the clause deleted) and one extra condition on the existing `p.Live`
  guard, so the footer and the keyboard agree;
- `play.go`'s `filepath.Rel(c.Lib.Dir(), path)` turned out to be **unreachable**
  in this state → [[a-named-site-can-be-unreachable]].

The battle itself is unaffected either way — `model.go` builds its mirror from
`seed.Books()` whatever `--data` says.
