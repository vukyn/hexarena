---
name: hexarena-battle-screen-budget
description: "hexarena PR #162 — the battle screen cannot fit 80x24 (28-row floor for a legal squad), so it budgets its own body and reserves the option list; what drops is non-monotone in the height"
metadata: 
  node_type: memory
  type: project
  modified: 2026-08-29T18:50:43.102Z
---

PR **#162** (`f44f807`, 2026-08-29). Closes the `TODO.md` height item — **by showing the item asked the wrong question.**

**It cannot fit 80x24.** `frame` leaves the body **20 rows** there (`m.height - 2` less two header rows). Sections: heading 1 · `tui.Board` **10 fixed** · `tui.Roster` 1+one a unit · `tui.Order` 1 · options 1+one an option.

| squad | roster | heading+board+roster+order+options |
|---|---:|---:|
| 1 a side | 3 | **20** (exactly, no blank, no log) |
| 3 a side | 7 | **24** |
| 5 a side | 11 | **28** |

`hex.MaxTeamSize = 5`, so **28 is the floor for a legal squad**, and a **summon** puts units on the board past the squad's five (up to nine slots a side → `board + roster = 29`). No arrangement fits.

**The defect was where the cut landed.** `frame` cuts from the **bottom** and `choices` was the last thing the body wrote — so the option list, the only thing the player acts on, was the first thing discarded. Fixable at any content height, and that is what shipped: `playFit` spends a purse of `m.height - 4` (derived from `frame`'s arithmetic, not copied), reserving heading + turn-in-front, then in order **save note → roster (clipped a row at a time) → board (dropped whole) → order line → log (remainder)**. One dim line names what is missing. Board goes whole because ten rows of art have no half, and `choices` already prints the occupant beside each cell when aiming; roster goes last of the three because it carries health and effects.

⚠️ **What disappears is NOT monotone in the height** — a test written from intuition will be wrong. Measured: `h=28..36` board+roster+order · `h=27..26` order gone, board still there · `h=25` board gone, **order back**. The board is ten-or-nothing, so where it still just fits it eats the rows the order line wanted. Only the **offering** order is monotone; assert that plus the existence of the steps, never a fixed sequence.

⚠️ **`playLogLines` capped events, not rendered rows** — it is 8 and the log measured **11 rows** mid-battle, because `tui.Line` opens a turn with a blank row of its own. The section with the loosest claim on the screen was the one whose budget did not hold.

**No per-screen floor:** `screenContent` already draws `m.tooSmall()` below 80x24, and at 24 the purse covers a 5-a-side roster whole under the option list. A second floor would live inside `tooSmall` and re-answer a settled question.

**The log's own two defects, fixed in #169** (`1dc59de`): `playLogLines` was a **ceiling that never yielded to the window** — `playFit` handed the log the budget's remainder and the remainder was clamped to 8, so the body grew 20→42 rows while the log stood at 8 (h=24 and h=80 both gave 8). It is `playLogWanted` now, gets the surplus nobody above it claimed, and reads 46 rows at h=80. And **275 of 283 rows were unreachable**: `collect` appends every event and never trims, so the whole history was in hand with no scroll and no key. `pgup`/`pgdown` scroll it (the pair that already scrolls the trait blurb and the picker pane), and the range sits on the **heading row** — no row cost, and it is what says a history exists at all.

⚠️ **Following the tail is a STATE, not an offset value, because the tail moves.** `logOffset` counts from the **start** (which is what `100-123/123` means) and `logFollow` sits beside it; nought is an ordinary offset (the top), so neither value can be a sentinel — an event arriving while the reader is at the tail must not silently change what is under them. Same rule as the queue reading (`Pending` answers 0 for an unknown unit and 0 means *soonest*). The test for it must append an event with **no turn behind it**: every real turn resets the frame through `record`, so a test that plays a turn measures the reset and passes against a stored offset. And clamp the offset where it is **read** — `undo` rebuilds the history shorter.

Drop heights were re-measured across #169 and are **unchanged**, which is the check that the log only ever spends rows nobody above it claimed.

`internal/tui`'s drawings untouched — clipping the roster's *rows* is not reformatting its drawing — and at any height where everything fits the body is byte-identical.

⚠️ **`TODO.md`'s "Done" section takes `- **Topic.** prose` bullets, not `- [x]`** — a `[x]` left in "Not done" is a mistake I have now made twice. And `TODO.md` conflicts on essentially every rebase in this repo.

Related: [[hexarena-battle-screen-summaries]], [[fixture-hidden-branch]].

**`[`/`]` alias the page keys, PR #170** (`b94c869`). The user's keyboard has no PgUp/PgDn, so every frame this client scrolls was unreachable for them and the footers naming those keys were unreachable advice. `[` back, `]` forward, at **all three** sites that scroll — the battle log, the trait blurb (`blurb.go`), the picker's reading pane (`picker.go`) — because one site aliased alone is the second-vocabulary-for-one-idea the log's own comment refuses.

The footers now name `[/]` **instead of** `pgdn/pgup` (an alias advertised in place of what it aliases; the page keys still work). Advertising both does not fit: `pgdn/pgup` is 9 cells against 3, and `en PlayAimFooter` was already 77 of 79 → 86. And `[/]` is the *more* universally actionable advertisement — every keyboard has brackets, page keys included. ⚠️ Do NOT leave the bare pair where the old wording carried a verb: `PlayFooter` had 11 spare cells, so `[/] cuộn` / `[/] scroll`, which is the one place the replacement would otherwise have said **less** than what it replaced.

⚠️ **A key alias is the shape that ships dead** — indistinguishable from a working one in any test that does not press it. Two guards, and the second is the one worth remembering: **the vacuity lives in the fixture, where an assertion cannot see it.** A `key` helper mapping `"["` to `KeyPgUp` makes an alias-equality table **pass completely** while proving nothing, so the helper's map is hoisted to package level and read directly by `TestABracketIsTheKeystrokeItLooksLike`. Same measurement the bare-space case once needed (`" "` stringifies as `"space"`, so every `case " "` compiled and matched nothing). The equality table itself must assert the page key **moved** the frame before comparing — two no-ops satisfy equality — and name the site that went unmeasured.

⚠️ **Renaming a test's key constants can silently drop the old key's coverage.** `play_test.go`'s `scrollBackKey`/`scrollOnKey` went `"pgup"/"pgdn"` → `"["/"]"`, which moved every existing press onto the alias; pgup/pgdown then hang entirely off the new table. The check is a mutation on the *old* key (drop `"pgup"` from the log's case and confirm red), not on the new one.

Safe to take a printable key only where no text field is fed: the picker returns into its reading pane before the typed field and gates typing on `numberKey`, the browse blurb has no input, `isSaveKey` runs ahead of the battle switch. `form.go`/`origins.go` have fields but never handled a page key.
