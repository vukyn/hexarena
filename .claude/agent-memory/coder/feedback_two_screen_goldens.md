---
name: two-screen-goldens
description: hexarena has THREE screens.golden — internal/screen's (the drawing), cmd/hexforge-tui's and cmd/hexarena-tui's (each client's framing) — each catches what the others cannot; their blind spots are the CAST under the cursor and the PALETTE (all three run NO_COLOR)
metadata:
  type: feedback
---

There are **three** `testdata/screens.golden` files over the same screens, and
**none may be dropped or treated as a subset of another**.

- `internal/screen/testdata/screens.golden` — **164 renders, 3851 lines** since
  step 6b: the six moved listings + the two hand-built states + the description
  screen in both readings (`skill blurb`, `trait blurb`) + **five picker states**
  (`kit picker`, `allowlist picker`, `filtered picker`, `status picker`,
  `reading a skill`) + the **skill listing's seven states** + the **squad
  builder's seven** (`squads`, `a squad`, `a squad member`, `a deep member`,
  `a held-back member`, `a squad kit`, `a squad trait`), both languages,
  `MinWidth x MinHeight` and 160x60, **body and footer recorded apart** (a screen
  here answers with `View(c) (body, footer)`).
  ⚠️ The picker is **handed** its list, so it has no one shape: its entries are a
  decision, and the five chosen are the paths through `View` that share no line.
  They are hand-built here and *raised* in the client, so the two records of the
  same name are the drawing and the raising of it.
  ⚠️ **The art preview is in all three now** (2026-09-03, branch
  `test/preview-sweep`) — it was the fifth and last screen outside every sweep.
  +164 lines / ~21.2 KB in each golden, a pure insertion. See the palette note
  below for what those entries can and cannot see. What is still unmeasured is
  another architecture (`rasterx` calls Sin/Cos/Atan2/Tan), so the shape record
  says in its own comment that it is same-machine.
- `cmd/hexforge-tui/testdata/screens.golden` — 200 renders, 8200 lines: every
  `everyScreen` entry **as the application draws it**, header dropped, inside
  `frame`.
- `cmd/hexarena-tui/testdata/screens.golden` — **96 renders, 3936 lines** since
  step 6c: the game client's framing, 24 entries over 12 of its 13 views, both
  languages, floor and 160x60. It holds **three things neither of the others
  can**: the **read-only footers** (`i18n.SkillsReadFooter` /
  `OriginsReadFooter` / `SquadsReadFooter` — nought hits in both other goldens
  before it existed), a **squad catalogue with rows on it** (both other fixtures
  delete `squads.json`), and a **battle at three a side** (a one-a-side board +
  roster + order + options is exactly the twenty rows the floor leaves, so
  nothing is ever dropped and the squeeze notice is drawn by nothing).
  Measured in 6c: widening `browseIDWidth` by one cell reddens **all three**
  goldens and **no property test in any package** — the cast browser's list row
  is a data column, exempt from every width sweep.

**Why:** measured, not argued. Three mutations, run against both:

| mutation | package golden | client golden |
|---|---|---|
| status category column `column+1` → `column+2` | **red** (`vi \| 120x24 \| statuses`) | red |
| `SpeciesScreen.View` keeps its trailing `\n` | **red** | **green** — `frame` pads the body with blanks to `room`, so a trailing empty row is absorbed |
| client `frame`'s `room := m.height - 2` → `- 3` | **green** | **red** — the *statuses* screen loses its caveat line to the `Truncated` marker |
| `screen.Ellipsis` widened by a glyph | red | red — the traits listing clips its own carrier row |

So: the package golden holds **what is drawn**, the client's holds **the framing
of it**. The `Ellipsis` row is the one worth remembering as a near miss — an
earlier note assumed only `frame`'s clip reaches it on these screens, and the
traits carrier row clips itself.

⚠️ **A golden can be the ONLY net, and that is not always a gap worth leaving.**
Measured in 5d-i: making the picker's filter select the wrong group
(`Groups[Filter-1]` → `Groups[Filter%len]`) reddened **both goldens and nothing
else** — the hand-written filter test walked the cycle, checked every visible row
belonged to the group in force, and checked the counts added up, all of which
still hold when the *labelling* is wrong in the same direction as the filtering.
Pin **which** group a press lands on, not merely that some group is in force.
Contrast the slot cap (`>=` → `>`), which reddened the **package tests alone** —
no golden moves, because a cap is a keystroke and a golden is a render.

⚠️ **Three of the squad entries record a FULLER state than the client's of the
same name, and it is the client's fixture that makes that necessary.**
`cmd/hexforge-tui`'s `scratchData` **deletes `squads.json`** before it loads
anything (it is the author's own working document, so a suite reading it would be
measuring somebody's saved sides), so the client's `squads` entry is the **empty**
listing and its `a squad trait` is a picker over a fixture cast that learns no
traits. Measured under mutation in 5d-iii: widening the catalogue's id column
(`Pad(squad.ID, width)` → `width+1`) reddens the **package golden alone** and
leaves the whole client suite green — a listing with no rows measures no column.
So the package golden's squads are **built as values** (never read off
`squads.json`, which would make the record move when an author saves a side) and
they deliberately hold rows where the client's hold none.

⚠️ **A state one golden cannot reach at all is worth hunting for, and the way to
find it is to count a wording's hits in both files.** Three steps running it paid:
#223 found the client blind to a squad catalogue *with rows in it* (its fixture
deletes `squads.json`), #225 found **both** blind to three works states, and 6b
found both blind to the played battle's **save note** — `i18n.NoteWrote` and
`i18n.NoteBattleVerify` at **nought hits in 8201 + 3098 lines, both languages**.
The client cannot draw that one: its fixture writes into `t.TempDir()`, so the
note names a path whose length is not stable between two runs on one machine,
which is why `everyScreen` leaves the state out and says so. The **package** can,
because nothing there writes — the note is a `[]forge.Note` value carrying a
relative path, which is also what a real save in `../seed/data` produces, so
`noAbsolutePath` still holds.

⚠️ **6b also measured a mutation only the two goldens caught**, and it was worth
closing rather than accepting: drawing the battle's option cursor a row late
(`index == p.Option+1`) reddened both goldens and the client's option-list sweep
and **nothing in `internal/screen`** — the ported row test read the id column and
the summary slot, neither of which a wrong cursor changes. An assertion went into
that test. Same shape as the picker's filter group in 5d-i.

⚠️ **WHICH row a detail-pane golden records is DATA, so a golden's blind spot
moves when the data does.** `browse` and `builds` record the pane of **one**
character — whoever is under the cursor — so they see that character's wrap
points and nobody else's. Measured 2026-09-03 on the `WrappedIn` off-by-one fix:
the *identical* change moved **three lines** of `internal/screen`'s golden on
`eeee515` and **nothing at all, in all seventeen golden files**, on `c98d218`.
The only difference between the two trees is #242, which sorted `cast.json` by
id and so put a different character under the cursor. Both clients' goldens
stayed byte-identical either way, because their `browse` entries land on a
character whose bio does not sit on the boundary. So a green golden after a
layout change is **not** evidence the layout did not move — it is evidence about
one row of one character. Measure the property over the whole cast in a test and
let the golden hold the bytes.

⚠️ **A golden's other blind spot is its PALETTE, and it is total.** All three run
under `NO_COLOR`, so any branch that only exists in the coloured drawing is
outside every record there is. Measured 2026-09-03 on `internal/screen`'s
`blockCell` (the art preview's coloured half): swapping `▀` for `▄` in all three
of its branches reddened **exactly one test in the repository** and **no golden at
all**; swapping the red/green weights in `luminance` — which the monochrome ramp
does reach — reddened **four**: the property test plus all three goldens. So "add
a golden entry" answers half of a screen that draws two ways; the coloured half
needs a property test, and there is nowhere else it can live.

⚠️ **A client golden's picture records FRAMING, not shape**, because both client
fixtures' art is `internal/testfixture.Art`, a 16x16 solid rectangle — the entry
is a flat block of one ramp character, and what a diff over it says is where the
drawing starts, how many rows the budget gave it and how wide. Only
`internal/screen`'s golden, which loads the **shipped** cast, records a
silhouette. Same shape of finding as the detail-pane note above: what a golden
sees is decided by the fixture's data, not by the screen.

⚠️ **lipgloss v2 writes truecolor regardless of `NO_COLOR` and `TERM`.**
`lipgloss.NewStyle().Foreground(c).Render(s)` emits `\x1b[38;2;r;g;bm…` in a bare
`go test` with `NO_COLOR=1` set — downsampling is the *program's* job in v2, not
the style's. Two consequences: a test asserting on colour needs no environment
setup, and a screen that must behave differently on a plain terminal has to ask
its own `Palette.Plain` rather than render and notice nothing happened. Still
assert an escape is present before reading one, or the colour claims go vacuous.

**How to apply:** when a screen moves into `internal/screen`, add its entries to
*all three* goldens; when one reddens alone, read the table above before assuming
the others are broken; when a mutation reddens *only* a golden, ask what the
behaviour test was actually asserting; and before deciding a golden entry closes
a screen, ask what that screen draws that `NO_COLOR` and the fixture's own data
keep out of the record. `make golden` accepts both
(`./internal/screen` is in the target now). Never `go test ./... -update`.

Repo context in [[screen-extraction]]; the archaeology recipe is
[[screen-neutrality-capture]].
