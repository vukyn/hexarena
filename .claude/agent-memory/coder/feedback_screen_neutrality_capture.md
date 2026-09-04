---
name: screen-neutrality-capture
description: Proving a hexforge-tui change is behaviour-neutral — there is a committed screens.golden now; a throwaway capture is only for archaeology, and its data dir must be RELATIVE, not just fixed-length
metadata:
  type: feedback
---

`cmd/hexforge-tui/testdata/screens.golden` exists now (200 renders, 8200 lines:
every `everyScreen` entry × both languages × 120x24 and 160x60), accepted only
through `make golden`. **So a neutral change is proved by the golden not moving,
not by a throwaway capture.** Write a capture only for *archaeology* — comparing
two commits that already exist — and then only with model-level API (`startIn`,
`everyScreen`, `screenContent`, `i18n.Lang`) so one file compiles on both trees.
Name it `zz_*_test.go`: that pattern is gitignored for exactly this.

**Why:** two traps, and my earlier note had the second one wrong.

- ⚠️ **The library directory reaches the BODY, not only the header.** The header
  (`programName` + `lib.Dir()`) is one line and can be dropped, but the check
  screen's count line (`i18n.CheckCounts`) prints `report.Dir` as ordinary
  content. A fixed-*length* temp path is therefore not enough and scrubbing is
  worse (the clip already happened at the real length). The answer is a
  **relative** dir: `scratchData(t)` for the fixture, copy it into
  `<t.TempDir()>/cmd/hexforge-tui/data`, put a pristine copy where
  `shippedDataDir` resolves from there, `t.Chdir` into that fake package dir, and
  `startIn(t, lang, "data")`. The fake tree is needed because two `everyScreen`
  entries call `start()` themselves and that reads `shippedDataDir` relative to
  the cwd. Resolve any golden path with `filepath.Abs` **before** the chdir.
- ⚠️ **Those two own-model entries need no scrub** — their directory only ever
  reaches the header, which is dropped. What replaces the scrub is an assertion
  that no line of the dump holds a rooted path at all (allowing for the bare `/`
  the skill filter's footer names as a key).

**How to apply:** check the instrument **both** ways before believing a reading.
Identical to itself over two runs on one tree says a difference means something;
**able to fail** says an identical pair means something — widen `screen.Ellipsis`
by a glyph and 8 lines move. And pick the base commit so that **no data change
sits in the interval**: measured, #198's one new character moves 42 of the 200
renders across 12 screens, which would drown any refactor signal.
`c576d7c`→`f3e6676` (#199, the drawing-context extraction) came back **byte for
byte identical** under this method.

Repo context in [[screen-extraction]].
