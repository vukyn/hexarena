# Decisions

Every piece of work that is **finished**, with the reasoning and the measurements
that decided it. This is the *why* behind the code — a merged PR says what
changed, and this says what was learned paying for it.

⚠️ **These entries were in `TODO.md` until 2026-09-05, under headings that had
stopped being true.** `## Not done` held eleven completed items among its eleven
open ones, and a transitional section carried twenty-five more — forty-five per
cent of that file was finished work filed as unfinished. Nothing was deleted;
the headings now mean what they say and `TODO.md` is the open list again.

⚠️ **Two entries were DUPLICATED and are merged, not chosen between.**
*Grow the cast* and *Graphical client with ebiten* each existed twice — once in
`TODO.md` and once in `CLAUDE.md` § *Open work* — both current, both maintained,
worded differently. That is the *"two callers wording one choice"* mistake
`CLAUDE.md` § *Mistakes already made here* records, at the level of a document.
Both survivors live in `TODO.md` § *Not done*, since both are still open; each
kept every fact either copy had.

**Read `TODO.md` for what is open, `CLAUDE.md` for what an edit may not break,
`docs/architecture.md` and `docs/balance.md` for the subject matter, and this
file for why a finished thing is the way it is.** The order below is the order
the entries were written in, which is roughly the order they landed.

- [x] **`weigh` can price a skill that deals none. DONE.** The refusal was
      right and its **evidence** was mis-specified. *Worth nothing* and *not
      rated* are still different answers and a row that did nothing is still
      refused — but the proof that the mechanism fired was a count of landed
      *damaging* strikes, which is no proof at all about a skill whose mechanism
      is not damage. `forge.Mechanism` is that evidence now: **striking,
      applying, restoring, cleansing, summoning**, read off the skill **as the
      row fought it** (a `power` sweep can add or remove striking) and counted
      off `[]battle.Event`, which stays the only contract a reader may use.
      `Weighing.Worth` never needed the change — the challenger's balanced share
      over a duel against its own twin does not read damage — so the ruler was
      always there and only the guard in front of it moved.
      **Measured, all at level 60, `--seeds 10000` (20,000 battles a row, band
      **±0.8%**), each carrier against a copy of itself.** ⚠️ A weigh figure is
      not a win rate and does not carry across a data change; quote it with its
      carrier and level or do not quote it.

      | carrier | skill (mechanism) | field | 2 | 3 = shipped | 4 | 5 | ordered? |
      |---|---|---|---:|---:|---:|---:|---|
      | `pokemon.cleffa` | `charm` — nought power, applies `weaken`×2 | `cooldown` | **+7.8%** | **+0.0%** | **−18.7%** | **−23.4%** | worth ✅ · turns ❌ |
      | `pokemon.cleffa` | `moonlight` — nought power, restores 400 | `cooldown` | **+25.5%** | **+0.0%** | −0.1% | **−14.3%** | worth ✅ · turns ✅ |

      Median turns beside them: charm 120 / 118 / 119 / 109, moonlight 127 / 118
      / 117 / 109. Both **controls read 10,000–10,000 exactly** — the whole
      instrument still rests on `refuseUnevenControl`, and it holds on a skill
      that never strikes.
      A third shape, `naruto.naruto` `shadow_clone` (a **summon**), `cooldown`,
      `--seeds 2000` (band ±1.6%): 3 → **−8.6%**, 4 → −5.2%, **5 = shipped →
      +0.0%**, 6 → −4.5%, 7 → **−28.5%**, at 272 / 276 / 273 / 246 / 223 turns
      and 94,589 clones off 49,887 casts on the control row. ⚠️ **That sweep is
      NOT monotone in worth and the report refuses to price it**, which is the
      honest outcome rather than a failure: the shipped cooldown of 5 is a local
      *maximum*, so a shorter one makes Naruto worse — the `kunai` lesson again,
      a cheaper skill gets reached for more and crowds out the one that should
      have been cast. Nothing was tuned.
      ⚠️ **This entry's own example does not work, and not for the reason stated
      here.** A `cooldown` weighing on `poison_powder` still refuses after the
      widening, and the refusal now says why: *cast **0** time(s) and applied no
      status*. `pokemon.bulbasaur` is its only carrier (`--carriers all` is one
      row and ten skips), its fielded trait at the cap is **`venom_blood`, which
      resists `poison` at 1000‰ — totally** — and a weighing fights a carrier
      against a copy of *itself*, so the only target on the board is immune to
      the skill's only mechanism and `Suggest` never picks it. Power 0 was never
      the obstacle. A skill can be unweighable because **its carrier's twin is
      its counter**, and no widening reaches that.
      ⚠️ **A restore is the one effect the log cannot attribute by itself.** The
      `Healed` a skill's `restores` produces carries neither the skill nor the
      caster — `Actor` on it is whoever's health went up — so it is credited to
      **the cast in progress**, which is exact because a cast resolves whole
      before the next begins. The other two `Healed` are told apart by what they
      *do* carry: a regeneration names the status that ticked, and a drain always
      carries the share it took, because a share of nought heals nothing and
      emits no event at all. If a `Skill` field is ever added to that event, this
      rule becomes a second reading of one fact and should be deleted.
      ⚠️ **The two-mechanism case has no shipped instance that can be priced.**
      `pokemon.squirtle` `withdraw` restores 500 *and* puts two `block` charges
      on, which is the row that would show two mechanism columns at once — and
      the squirtle mirror leaves **200 of 200** battles undecided at 100 seeds,
      so the endless refusal takes it first. The columns are exercised by the
      fixture instead.
      → `internal/forge/weigh.go` (`Mechanism`, `Mechanisms`, `mechanismsOver`,
      `refuseUnreadable`), `internal/forge/spar.go` (`Effects`, `Matchup.fold`,
      `restored`), `cmd/hexforge/weigh.go`.
      ⚠️ **`docs/balance.md` § *Pricing one number* was stale in two places and both
      have since been corrected**: its refusal list now reads "a row on which the
      skill did none of its own work" rather than "landed the skill zero times",
      and its `⚠️ Only scalars are weighable` paragraph now seats
      `self_gradient` rather than giving a reason for leaving it out. See the
      entry below.
- [x] **`self_gradient` is ONE number, and it is now the ninth `WeighField`.
      DONE.** The line above it used to read "a bonus *and* a share, so sweeping
      it is a surface"; that was measured false, and what actually kept it out —
      an off state that is not a number — has been seated rather than argued
      about any longer.
      **The decision taken: seat (a).** `of` hands back `AtEmpty`, or nought when
      the skill declares none — nil-safe the way `Gradient.Share` already is —
      and `set` assigns straight through, so a nought falls to the parser, which
      refuses `at_empty` below one in **its own words, unwrapped**. The sweep
      stays one-dimensional, so `MonotoneWorth` keeps exactly the meaning it has
      on the other eight fields: one field, one line, one answer. This field
      makes no surface; the surface is the entry below, and it is a different
      pair of numbers.
      **Why seat (b) lost, put on disk rather than reasoned about.** Mapping the
      nought to nil inside `set` buys the one row seat (a) cannot have, and pays
      a second copy of a bound `skill.resolve` owns — the one thing `set`'s own
      doc says it exists not to do. With that mapping actually in the file,
      `TestASweptGradientOfNoughtComesBackInTheParsersOwnWords`,
      `TestAGradientPricesHowMuchAndNeverWhether` and the CLI's
      `TestWeighingAGradientOnASkillWithoutOneSaysWhatTheParserSays` all fail,
      the last of them with the sweep refused for being *saturated* — an
      unrelated sentence about a run that should never have started. That is
      what holds the seat.
      ⚠️ **The honest limit, written into the code comment and standing here: the
      field prices HOW MUCH a gradient is worth and never WHETHER to have one.**
      A sweep may not contain the control row of a skill that declares none,
      because that control is a nought and the parser refuses the whole report
      before a battle — so no report anywhere has a row for "this skill with no
      gradient at all", and none may be read as one. Stated rather than worked
      around.
      **The reading**, on `desperate` at level 60, 10,000 seeds from each slot,
      80,000 battles, band **±0.8pp**, carrier `fixture-anime.gradient` — the
      bench adept with `desperate` in its fielded four, kit `strike riptide
      guard_wall desperate`:

      | `at_empty` | worth | turns |
      |---:|---:|---:|
      | 500 | **−20.8%** | 68 |
      | 1000 (control, what the bench declares) | +0.0% | 64 |
      | 1500 | **+20.0%** | 62 |
      | 2000 | **+34.6%** | 60 |

      **`MonotoneWorth` holds and so does `MonotoneTurns`** — the report says
      "only ever moves one way" for both columns — and every swept row clears the
      band by twenty-five times or more. It is a price on a fixture carrier
      against a copy of itself, so it says nothing about the shipped book: what
      it establishes is that the field measures at all, and that a gradient is
      worth a great deal to a carrier that leans on one.
      ⚠️ **Shipped carriers: zero, and the book was not touched to change that.**
      `hexforge weigh --carriers all comeback --field self_gradient` still
      answers *"no character brings comeback at level 60, so there is nothing to
      price: all 11 in the book were skipped"*. A gradient will be priced on a
      shipped character the day one fields `comeback`, and not before.
      ⚠️ **The carrier is built by the test rather than added to the fixture
      cast, and the cost of the other way was measured.** `desperate` is in
      neither fixture character's kit; putting it in the adept's displaces
      `purify` from the fielded four, and the adept fights in the goldens — with
      it in the fixture, `make golden` moves **656 lines** across
      `cmd/hexarena-tui/testdata/screens.golden` and
      `cmd/hexforge-tui/testdata/screens.golden`. The goldens are the design
      record, and none of that would be a design decision. `bringsTheGradient`
      saves a copy of the carrier instead, which is what `forkedTwin` does one
      file over and for the same stated reason: a weighing needs carriers the
      fixture cast does not have. **No golden moved.**
      → `internal/forge/weigh.go` (`WeighSelfGradient`, `of`, `set`),
      `internal/forge/weigh_test.go` (`bringsTheGradient`, `declaringSkill`),
      `cmd/hexforge/weigh_test.go`, `docs/balance.md` § Pricing one number.
- [x] ⚠️ **The art preview was outside every sweep there is. DONE.** It is
      registered now in **all three**: `everyScreen` in cmd/hexforge-tui
      (`language_test.go`), `everyScreen` in cmd/hexarena-tui (`sweep_test.go`)
      and `everyMovedScreen` in internal/screen — so the screen has the width,
      translation, leak and read-only sweeps every other screen has, and a golden
      entry in each of the three records. It was the **fifth and last** instance
      of the shape `CLAUDE.md` records having been made four times, and the one
      that had been *known* the whole time; cmd/hexarena-tui's `notSwept` is
      **empty** now, since this was the single excuse in it.

      **The decision: a plain-text golden, plus a property test for what a golden
      cannot see.** Goldens here run under `NO_COLOR`, so the record is `rampCell`
      — which is also the only affordable one: measured, one character in one
      language is 19 lines / 2.0 KB at 120x24 and 55 lines / 8.4 KB at 160x60
      plain, against **14 KB** and **128 KB** coloured, because every cell carries
      its own truecolor sequence. Both windows are taken, for the reason every
      other entry takes both: the floor is where `previewChrome` bites (`Height -
      8` leaves 16 rows, so the art is rasterised into a 32-pixel box) and the
      roomy size is the same picture with nothing taken away. **Measured cost:
      +164 lines / ~21.2 KB in each of the three goldens, 0 lines removed** — a
      pure insertion, nothing else moved.
      ⚠️ **The three records do not hold the same thing, and that is the point.**
      internal/screen's is over the **shipped** cast, so it records a shape
      (`naruto.naruto` at the cap, `assets/naruto-sage-mode.svg`); both clients'
      fixtures use `testfixture.Art`, a 16x16 solid rectangle, so theirs are a flat
      block of one ramp character and what a diff over them says is the
      **framing** — where the drawing starts, how many rows the budget gave it,
      how wide it is. A flat fill also carries none of the shape record's
      same-machine caveat: a rectangle is a fill rather than a curve.
      ⚠️ **The shape record is a same-machine record, said in the test rather
      than assumed.** The rasterisation is reproducible here — byte-identical
      across separate `go test` processes, in both drawings, and twice in one
      process off two separately loaded libraries — but `rasterx` calls
      `math.Sin` (15), `Cos` (10), `Atan2` (6) and `Tan` (4), which is the family
      Go has had per-architecture assembly for. So the entry's own **assertions**
      are structural — that the rows are drawn out of `Ramp` and nothing else, and
      that the widest is `UsableWidth() - 2` — and the pixel field is left to the
      record, where a diff on another machine is a finding to read rather than a
      gate that broke.

      ⚠️ **The width question, answered: the picture is exempt and the wording is
      not.** `CLAUDE.md` § the TUI width rule splits prose, which takes
      `MinWidth`, from data, which takes `UsableWidth()` — and the art is neither.
      `picture` asks for exactly `UsableWidth() - 4` cells and `cellRows` writes
      one cell a pixel column after a two-space indent, so **every row is
      `UsableWidth() - 2` wide by construction**, which at the sweep's own 200
      columns is 198: an assertion against the floor there would either fail on
      correct code or pass on nothing. So both clients' width sweeps skip the
      picture rows by the ramp's alphabet (`aPictureRow`, which is why `Ramp` is
      exported) and measure the heading, the art/level/stage line and the footer
      like any other sentence. What holds the arithmetic instead is the test that
      already existed for it, `TestNoDrawingIsEverWideEnoughToBeMarked`.

      ⚠️ **What the two mutations proved, applied on disk and read back through
      `git diff` rather than through a green run.** Both were `entirely green`
      before this item; the whole suite was run once per mutation.
      **Swapping `▀` for `▄`** in all three branches of `blockCell` reddens
      **exactly one test in the repository**, `TestEachPixelIsDrawnInItsOwnHalfOfTheCell`,
      with *"the top half alone: the cell's upper half is unpainted, want the top
      pixel's 200;40;40"* and five more like it — **all three goldens stayed
      green**, which is the entry's claim about `NO_COLOR` measured rather than
      argued. The property is that the half a pixel is drawn in is the half it
      came from: the cell is taken apart into (upper, lower) by reading the block
      character — `▀` hangs the foreground above and the background below, `▄` the
      other way round — so a swap moves a colour into the wrong half in every
      branch. It needs no environment: `blockCell` builds a bare lipgloss style
      rather than asking the Palette it is handed, so it writes truecolor whatever
      the terminal is, and the fixture checks a cell has a sequence in it before
      it reads one.
      **Swapping the red and green weights** in `luminance` reddens **four**:
      `TestTheRampWeighsGreenOverRedOverBlue` (*"the weights read green 76, red
      149, blue 29"* and *"the ramp draws green at 7, red at 4 and blue at 8"*)
      and all three goldens — including the flat-fill client ones, since a solid
      colour still has a luminance and the block went from `+` to `*`. The
      property is the ordering rather than the constants (green weighs most, blue
      least) and it is asserted **through `rampCell` as well**, because the ramp
      is inverted against the weight and an ordering that held on the number while
      the inversion turned over would be a picture drawn inside out.

      **Still not covered, stated rather than left to be discovered.** The
      coloured drawing's **exact bytes**: `blockCell`'s output is measured
      cell-by-cell as a property and by no record anywhere, so the *composition*
      of a coloured picture — 128 KB a render — is deliberately unrecorded. And
      **another architecture**: the shape record is same-machine, for the named
      reason above. Whether the shipped traced SVGs even reach an arc path is
      still unknown; they are `vtracer` output, which is beziers and polygons.
      → `internal/screen/preview_test.go`
      (`TestTheRampWeighsGreenOverRedOverBlue`,
      `TestEachPixelIsDrawnInItsOwnHalfOfTheCell`, `halvesOf`),
      `internal/screen/screens_golden_test.go` (`theArtPreview`, `aRampRow`),
      `cmd/hexforge-tui/language_test.go`, `cmd/hexarena-tui/sweep_test.go`
      (`aPictureRow`, `notSwept` emptied), the three `testdata/screens.golden`,
      `CLAUDE.md` § the description screen.

- [x] ⚠️ **The three read-only views were a dead end on a line that FORKS, and
      the sweeps could not see it. DONE.** Reported by a user: `p` on
      `pokemon.poliwag` at any level from 32 up drew
      `level 46 reaches [Poliwrath Politoed], which are alternatives: name the one
      being fielded` in red and no picture at all.

      **The refusal is right and was not touched.** `progression.Line.StageAt`
      refuses on purpose — taking whichever arm the file lists last hands a reader
      a wrong stat line, a wrong picture and a wrong trait list with nothing on
      screen saying so. What was missing was a way to *name* an arm, so a shipped
      character was unreachable in every read-only view.

      **The shape, which is the squad builder's own, one size smaller.** A
      placement may deliberately field an earlier form, so the builder's chooser
      is over `StagesAt`; these views ask the narrower question — which grown form
      does this level resolve to — so their arms come from `FurthestAt`, which is
      **one** stage on a line that does not fork. The choice lives on
      `BrowseScreen.Form`, is **settled on every read** by `screen.ChosenForm`
      (the cursor and the level both move under a chosen name), rides to the two
      describers on `Subject.Stage`, and is walked with `s` — answered in the
      browser and in each client's `updatePreview`/`updateBlurb`, exactly as the
      level already is, because a describer keeps no cursor of its own.
      ⚠️ **The arm a chooser opens on is a pick, and what makes it not the silent
      pick `StageAt` refuses is that `FormRow` draws it** — on all three views, in
      both languages, before anybody has pressed anything. `TestAForkIsNeverPicked-
      Silently` is that property.
      ⚠️ **`previewChrome` became a floor.** The fork row is a fourteenth line, so
      `View` counts what it wrote and the drawing gives the row back; a constant
      that did not would put the picture one row over budget on exactly the
      character the row was added for, which is that comment's own defect a second
      time. Measured in the record: the forked preview entry is the **same total
      height** as the linear one in all three goldens at both sizes.

      **Coverage, which was half the work.** The preview had only just entered the
      sweeps and was pointed at a **linear** fixture, so the forked case was
      covered by nothing. Added: `a forked art preview` and `a forked trait blurb`
      to both clients' `everyScreen`, plus `a forked cast row` in
      `everyMovedScreen`, all through a `theForkedBrowser` that **finds** the fork
      in the shipped books and is fatal when there is none.
      **Measured golden cost: +328 lines in each client record, +397 in
      internal/screen's, and 0 lines removed anywhere** — a pure insertion, which
      is the proof that a line that does not fork is byte for byte what it was.
      ⚠️ Red-before-green was verified by reverting the production change alone:
      `TestAForkingLineIsPreviewedRatherThanRefused` fails with *"the art preview
      of pokemon.poliwag at level 46 draws the refusal instead of a picture"*, and
      reverting the chrome arithmetic alone reddens
      `TestThePreviewFitsTheWindowItWasGiven` with *"on a line that forks at 24
      rows the preview is cut off"*.

      **Still not covered, stated rather than left to be discovered.**
      ~~**(a)** The **detail pane** on a forking character is in
      `internal/screen`'s golden but in **neither client's sweep**.~~ **DONE.**
      `a forked detail pane` is now an entry in both clients' `everyScreen`
      (`cmd/hexforge-tui/language_test.go`, `cmd/hexarena-tui/sweep_test.go`),
      registered off the same `theForkedBrowser` the preview and blurb entries are
      raised from — so the pane the fork actually decides (the chooser row, the
      two-ended stage summary, and the art, trait and stat rows that read the arm
      in front) now has a width test, a translation test and a leak test, and a
      record in both client goldens. What let it in is `kitGlosses`, the
      `whoMayCarry`/`traitCarriers` twin the entry above predicted, in both
      fixtures. **Measured golden cost: +164 lines in each client record and 0
      removed anywhere, `internal/screen`'s golden unmoved** — a pure insertion.
      ⚠️ **The exemption is tight, measured rather than argued.** Across the whole
      of both sweeps it newly skips exactly **two** lines, both of them the
      glossed-kit row itself (the forked pane's 199-cell one and the ordinary
      browser's 64-cell one), and **none at all in English**, where `GlossedKit`
      draws nothing. Its nearest sibling — `BudgetPierced`, the other dim
      `WrappedIn` row on the same pane — was lengthened to 198 cells as a mutation
      and both sweeps went red on *the a forked detail pane screen in vi draws a
      line 198 cells wide, over the 119 it has*.
      ~~**(a′)** ⚠️ **The claim this entry used to make about the form row is
      false, and the mutation is what disproved it.** … The fix is to narrow what
      a stage name may exempt (a name is a *cell*, not a line).~~ **DONE.**
      `freeText` is split in two in both clients' fixtures. **Prose** — a
      biography, an origin's or a species' note, a build's intent, the library
      directory — keeps the line exemption `carriesFreeText` always gave it,
      because a paragraph really is the whole row. A **name** — `character.Name`,
      `stage.Name`, `origin.Title`, `built.Name`, and the arena client's
      `squad.Name`/`squad.ID` — moves to `freeNames`, and `withoutNames` takes it
      *out of the line* so the wording either side of it is still measured.
      Longest first, because `Mew` is a shipped form name and a substring of
      `Mewtwo`, and stripping the short one first would leave `two` behind to be
      measured as though the program had written it.
      **Red-before-green, as four mutations on disk.** `i18n.FormChoice` at 155
      cells now reddens **both** clients in all three forked entries — *the a
      forked detail pane screen in vi draws a line 157 cells wide, over the 119 it
      has*, printing `dạng  <  >  …` with the form name gone, which is the row the
      old exemption swallowed whole. `BudgetPierced` lengthened is still red, at
      197 cells, so nothing regressed. A **116-cell form name** and a **99-cell
      character name** injected into the fixture leave both sweeps green — a name
      has no promised length, and that is the behaviour being kept.
      **The kit-ids row is now exempt by kind.** `kitIDs` is `kitGlosses`' twin
      beside it, covering both of the pane's `UnlockSummaryAt` rows — the kit and
      the traits — at every level either can change at, since the summary prints a
      gate only while it is still ahead and `carriesFreeText` matches on an
      opening. Dropping `kitIDs` reddens both clients in **both** languages (*156
      cells* in vi, *153* in en), so it and not a form name is what holds that row
      now; the traits row goes in beside it because it is the same call on the
      other learnset, not because it fits today.
      ⚠️ **The strip's cost is measured, not asserted.** Nothing knows *where* on
      a row a name was drawn, so a name occurring inside ordinary wording is taken
      out of that too — an error that only ever runs one way, since the remainder
      is shorter than what was drawn and can therefore hide a breach but never
      invent one. Over every line of every screen of both clients in both
      languages: `kitIDs` newly skips **3 lines** per language in `hexarena-tui`
      and **2** in `hexforge-tui`, every one of them an id row; the strip touches
      70–142 measured lines; the most any one line gives up is **33 cells** — the
      stage-summary row, four form names on one line — leaving 39 cells of program
      wording still measured; and it takes **0** lines anywhere from over the
      floor to under it, so it is at present hiding nothing. No minimum name
      length is imposed: one would cost more than it bought, because `Mew` left
      out of the strip means the Mewtwo pane is measured with its own name on the
      row, which is a failure *invented* out of authored data. The number to watch
      as the books grow is that rescued-line count, not a name's length.
      **No golden moved**, in any of the eight packages `make golden` covers,
      which is what a test-only change should cost. Nothing legitimate was hiding
      behind a name: closing the hole caught no over-long wording.
      ~~**(b)** `cmd/hexforge/new.go`'s `--levels` print~~ **DONE — a row per
      arm.** `renderCharacter` reads `FurthestAt(level)` and draws the stat line
      and the stage row **once per arm**; a one-shot print has nowhere to hold a
      choice, so it prints both rather than choosing. Measured before: `hexforge
      show pokemon.poliwag` ended on *"level 60 reaches [Poliwrath Politoed],
      which are alternatives"* where the stats belong, so **neither** arm's
      numbers were reachable from the command. After: Poliwrath 10,632 and
      Politoed 11,176 of the 11,500 budget, each with its own picture.
      `FurthestAt` answers exactly one stage on a line that does not fork, so an
      ordinary character costs nothing — held by
      `TestAnOrdinaryLineIsStillOneRow`, which counts the stage rows of every
      unforked shipped character. `TestAForkingLineIsShownAsARowPerArm` goes red
      on the old resolve with *"the page hands the reader the refusal where the
      stats belong"*, and asserts the **pair** — a page naming one arm and not
      the other would be the old defect wearing a stage name. No golden moved.
      ~~**(c)** `SquadsScreen.Form` still falls back to the empty stage on a fork
      the placement has not named, and `stageLabel`/`unitLine` still call that
      *furthest*.~~ **DONE — named and offered.** Option (1) of the two, and it
      was the cheap one: the form field is **already** a chooser and
      `StageChoices` has listed both arms by name since it was written, so
      "offer it" cost nothing and the whole of what was missing was the naming.
      `formLabel` — one function behind both the field and the member's row in the
      squad, because they are one fact at two depths — draws `SquadForkUnnamed`
      where the line forks and `SquadFurthest` everywhere else, and a
      `SquadForkArms` line under the fields names the arms and the key.
      **It was two defects with one cause, and the second was the invisible
      one.** `Form` returns the empty string on an unnamed fork, which
      `cast.SkillsAt`/`PassivesAt` read as "no gate is held", so both pickers
      silently offered only what every arm learns. Measured on `pokemon.poliwag`
      at level 60: **unnamed 13 skills / 4 traits · Poliwrath 14 / 4 · Politoed
      15 / 5** — `submission` missing against one arm, `rinse`, `chorus` and the
      trait `composure` against the other. `Form` still refuses rather than
      picking, because a picked arm is a wrong learnset written into the author's
      own file; what changed is that the screen says so.
      **Which bug it was, measured rather than assumed: it builds and then fails
      to *save*.** `placement.Squad.Take` — the one call `forge.SaveSquad` and
      `forge.FightSquads` both make — refuses the member with *level 60 reaches
      [Poliwrath Politoed], which are alternatives: name the one being fielded*,
      so an unnamed fork was never a wrong battle, it was a dead end whose first
      mention of a fork arrived under the save key. `forge.Load` accepts a
      hand-written one, so the refusal is at the two gates and not at the parse.
      ⚠️ **The note is wrapped, not clipped like the held-back line beside it.**
      It is the only note on the screen carrying a value out of the data — arm
      names have no promised length — so the wording that fits the floor alone
      does not fit it beside them, and one clip would take the consequences off
      the end. Wrapped at `MinWidth`, which is the prose half of the width rule,
      and registered as the **third** entry of
      `TestAWideWindowStillWrapsProseAtTheFloor` — the sturdiest of the three
      against that test's recorded loss of coverage, since a value is what decides
      whether it wraps rather than a wording somebody may shorten.
      ⚠️ **The width sweep could not have caught the clip, and did not.** A stage
      name is stripped from the line by `freeNames`, so the sweep measured ~104
      cells of wording where the terminal drew ~124 — the golden is what showed
      the ellipsis. A row carrying a name is measured *around* the name; it is not
      measured *with* it.
      **Coverage.** `a forked squad` and `a forked member` are new entries in
      `internal/screen`'s golden and in `cmd/hexforge-tui`'s `everyScreen` —
      raised off an `aForkedMember`/`aForkingSquad` pair that **finds** the fork in
      the shipped books and is fatal when there is none, which is
      `theForkedBrowser`'s rule. Two entries because the two depths draw it
      through different code: the field reads the member under edit, the row reads
      a member that is not open. Nothing was added to `cmd/hexarena-tui` — `n` and
      `enter` are gated on `Context.Authoring`, so a game client never reaches
      either depth. **Measured golden cost: +328 lines in `cmd/hexforge-tui`'s
      record, +148 in `internal/screen`'s, 0 removed anywhere, and
      `cmd/hexarena-tui`'s unmoved** — a pure insertion, which is the proof a line
      that does not fork is byte for byte what it was.
      ⚠️ Red-before-green verified by reverting the production change alone:
      `TestAnUnnamedForkIsNotCalledFurthest` fails with *the member's form row
      calls [Poliwrath Politoed] "dạng xa nhất", and that word names neither end of
      a line that forks*, `TestTheSquadRowSaysTheSameThingAsTheFormRow` and
      `TestAnUnnamedForkNarrowsBothLists` go red with it, and
      `TestALinearMemberIsStillCalledFurthest` stays green — which is what makes it
      the control rather than a fourth copy of the same assertion.
      ~~**(d)** The alternative UI shape, if `s`-cycles ever reads wrong~~
      **DECIDED — the sub-picker is not built, and the two spellings are a key
      budget rather than drift.** The trigger this was written against has not
      fired: nothing reads wrong, and the alternative costs a `PickState`, a
      `Target`, an entry in each client's `raiseTargets` and a fourth screen in
      the three sweeps, against a chooser row that is two keys and no navigation.
      ⚠️ **The question it was really holding open was why one fork is chosen two
      ways** — `< value >` walked with ←/→ in the squad builder, `s` on browse and
      preview — and that is answered rather than left to taste: **←/→ are already
      the level** on both of those screens (`describe.go`, twice, plus `home` and
      `end`), so a field walked with them cannot exist there without taking the
      level's keys. The builder has fields and a cursor to focus one; browse and
      preview have neither. Two spellings, one reason, written down here so the
      next reader does not "unify" them into a screen that can no longer change
      level. Revisit only with a reader who actually got it wrong.
      → `internal/screen/form.go`, `browse.go`, `preview.go`, `blurb.go`,
      `action.go` (`Subject.Stage`), `internal/i18n` (`FormChoice`),
      `cmd/hexforge-tui/describe.go`, `cmd/hexarena-tui/subject.go`,
      `internal/screen/form_test.go`, both clients' `fork_test.go`,
      `cmd/hexforge-tui/tui_test.go` (`ontoTheFork`), the three
      `testdata/screens.golden`, `CLAUDE.md` § the squad builder and § the
      description screen; and for (a), `cmd/hexforge-tui/language_test.go` and
      `cmd/hexarena-tui/{sweep,fixture}_test.go` (`kitGlosses`); and for (a′), the
      same three files (`freeText`/`freeNames`/`withoutNames`/`kitIDs`, and the
      width, language and gloss-leak sweeps in each); and for (c),
      `internal/screen/squads.go` (`formLabel`, `unnamedArms`, `characterOf`),
      `internal/i18n` (`SquadForkUnnamed`, `SquadForkArms`),
      `internal/screen/squadfork_test.go`, `internal/screen/screens_golden_test.go`,
      `cmd/hexforge-tui/{language,width_rule}_test.go`, two of the three
      `testdata/screens.golden`, and `CLAUDE.md` § the squad builder.
- [x] ⚠️ **A saturating multiplier is re-narrowed one line downstream. DONE.**
      The question this asked first — carry a saturated multiplier, or refuse it
      where it is produced — is **answered by `TODO.md` § *Decided against*:**
      a ceiling on `Skill.Power` is an implementation limit dressed as a design
      bound, so there is nothing to refuse with, and the figure is carried. Carrying
      it means the arithmetic that takes it is exact, which is the answer this
      package already gave twice.
      **The three sites were nine.** Beyond the two splash shares and the restore:
      the rating's own splash share, its `perStrike × connecting`, its wall of
      block charges, `Rules.Total`, `ExpectedStrike`'s weighted average,
      `Rules.Expected`, both halves of a converted strike **and their sum**,
      `Restore`, `Pierced`, `ExpectedStrikes`' own count, and the two attempt
      tallies. `combat.Scaled`, `combat.Repeated`, `wide.plus` and a saturating
      `summed` now carry all of them.
      ⚠️ **The worst of them was not a limit of the type at all**: a weighted
      average lies between the two figures it averages, so `ExpectedStrike` could
      never legitimately need more than an `int64` — and it wrapped to **−1**,
      arithmetic thrown away on the way to a number that was always representable.
      ⚠️ **The property is monotonicity, not a table.** *"Never smaller for more
      power"* is what a wrap always breaks and a saturation never does, so it
      catches a site nobody has written yet. Measured: no golden moves and no
      shipped figure changes — the largest landable multiplier in the book is
      3,500, twelve orders below where any of this begins.
      → `internal/core/combat/carry_test.go`, `internal/core/battle/carry_test.go`,
      `CLAUDE.md` § *Saturate continuous values, cap discrete ones*.
- [x] **`combat.ExpectedStrike` weights in a narrow `int64` product. DONE.**
      Folded into the item above and fixed with it: the weighted average is built
      in 128 bits through `wide.plus`, and `Rules.Expected`'s second product —
      added by #226 on the same path — goes through `combat.Scaled`. Both were
      rating-only and reached no golden, which is exactly why nothing reported
      them; `TestNoFigureFallsAsPowerRises` reports them now.

- [x] ⚠️ **The ninth narrow product had no board, and it was not the clamp
      hiding it. DONE.** The entry this replaces blamed `against`'s
      `landed > target.HP` clamp. Measured, that is wrong, and the correction is
      the useful part of the item.
      **What the mutation says.** The saturating call in `pastAWall` was put back
      to the raw `perStrike * charges` **on disk** (verified through `git diff`,
      not through a green run) and the whole suite run: **0 tests red**, in every
      package. Not one board, not one golden, not one arithmetic test — including
      `TestRepeatedIsTheNarrowProductWhereverThatHeld`, which measures
      `combat.Repeated` itself and never sees the call site.
      **Why no board.** A probe on the branch counted **at least 18.9 million**
      entries into the wall clause across one full suite run — ≥7.9M in
      `internal/seed`, ≥7.1M in `cmd/hexforge-tui`, ≥3.9M in `internal/forge` —
      with **0 wraps**, and the largest per-strike figure any of them put in front
      of the wall was **1,208**. The wrap begins at 3.07 × 10¹⁸. Nothing shipped
      is within fifteen orders of magnitude of it, which is this repository's own
      standard for *reachable* and settles the question: there is no honest board.
      ⚠️ **The clamp cannot hide this, and never could.** Swept over 200,431
      per-strike figures × every charge count: `combat.Repeated` and the narrow
      product disagree **233,105** times, and in **233,105 of 233,105** `Repeated`
      answered `math.MaxInt64` — necessarily, since the two disagree exactly when
      the product will not fit, which is exactly when `wide.over` pins. So the
      correct blow past the wall is `damage − math.MaxInt64` in every disagreement,
      which is **nought**. The clamp only ever pulls a figure DOWN to the target's
      health; nought is never clamped. Of the **299,548** (per-strike, charges,
      blow) triples where the two arithmetics land on different figures, the clamp
      makes them equal again in **0**. What hides it is the `damage <= 0` guard one
      line below the product, catching the second wrap — measured on the one board
      that reaches it, `perStrike` 3,750,000,000,000,000,000 × 3 charges gives a
      narrow product of **−7,196,744,073,709,551,616**, and `damage − narrow`
      overflows in its turn and lands back under nought.
      ⚠️ **`TestNoFigureFallsAsPowerRises` does not transfer to this site**, and
      that was measured before it was abandoned rather than assumed. On the
      **correct** saturating code the figure past a wall FALLS as power rises, in
      **6 of 24** (connecting, charges) pairs on the carry ladder, **ten falls** in
      all — the blow saturates at `math.MaxInt64` before the wall's product does,
      so the subtraction shrinks while the power climbs. A test asserting *"never
      smaller for more power"* here would fail on the code it protects.
      **So the property is the wall's own axis**: *a deeper wall never lets more
      through*. Non-increasing in charges holds for a saturating product by
      construction and is the first thing a narrow one loses — a wrapped charge
      product comes back smaller than the shallower wall's, so the deeper wall
      subtracts less. It fails on the mutation at **20** places on the same ladder,
      across four `connecting` values and four separate rungs, so it hangs on no
      single modular coincidence and on no number in a book.
      → `internal/core/battle/carry_wall_test.go` (`TestNoWallLetsMoreThroughAsIt
      Deepens`), the one white-box file in `battle` and the header says why.
      ⚠️ The general shape stays worth naming even though the diagnosis moved: **a
      guard downstream of a defect makes the defect unobservable**, and the
      instrument reads the same either way. Same family as the blind boards in the
      guard sweep — and the lesson added here is that it is worth measuring WHICH
      guard, because the obvious one was innocent.

- [x] **`m.wrapped` no longer fills the window's final column. DONE.**
      `Context.WrappedIn` (`internal/screen/screen.go`) spent
      `UsableWidth() - 2 - width - 1`, so `LabelAt` emitted exactly
      `UsableWidth()` cells and a wrapped row filled the one column every other
      row leaves empty — the fifth copy of the off-by-one `FieldValueRoom`'s own
      comment records fixing in four other places. It spends
      `UsableWidth() - 1 - marker - width - 1` now, written in that function's
      idiom (the window less its final column, less the marker, less the label
      column, less the gap after it), and the `room < 8` clip is measured off the
      corrected number too, so the narrow branch gives up the same cell.
      ⚠️ **Seven golden lines moved and they are three renderings of one row**:
      `browse`'s biography for `naruto.naruto`, in Vietnamese at both windows and
      in English at the floor, each losing a word off a line and taking it up on
      the next — three lines in the first rendering and two in each of the others,
      which is the whole of it.
      **No row count changed**, so no screen's vertical budget moved, and the
      other sixteen golden files are byte for byte what they were — including
      both clients', whose own `browse` entries draw a character whose bio does
      not sit on the boundary.
      ⚠️ **That is far less than the defect is, and the gap is the fixture rather
      than the fix.** A `browse` or `builds` record holds the detail pane of
      **one** character — the one under the cursor — so it can only ever see that
      character's wrap points. Over the whole cast the defect is ordinary:
      **11 of 26** biography rows in Vietnamese and **6 of 26** in English filled
      the final column before the fix (thirteen characters, at the 120 floor and
      at 160), `pokemon.gastly`, `pokemon.mew`, `pokemon.mewtwo`,
      `pokemon.machop` and both fixture characters among them.
      ⚠️ **And which character the record draws is itself data.** Measured on the
      tree as it stood before #242, which sorted `cast.json` by id and so moved
      the browse cursor onto Naruto: `make golden` under each arithmetic in turn
      came back **byte for byte identical, all seventeen files**. The same fix,
      the same suite, the same screens — and one data commit is the whole
      difference between a golden that reports it and a golden that cannot.
      → **The same root has an open entry in `TODO.md` § *Not done***, "a `hexforge new` still churns
      `screens.golden`": `aSquadOfSide` picks its squad as `characters[index %
      len(characters)]`, so the cast file's order decides which character every
      one of these records draws. That entry reads the churn half of it and this
      one the blindness half — one fixture, two symptoms, and the third option it
      lists (pick by a property the screen measures) answers both.
      ⚠️ **Nothing is newly cut by #186 either**, which is the reason this was
      left alone rather than folded in. A word longer than the room still takes a
      line of its own and overflows it — `WrapWords` says so on purpose — so
      narrowing the room by one can only newly mark a line carrying a word of
      exactly the *old* room. The longest whitespace-delimited token drawn on any
      recorded screen is **38** cells against a room of **99** at the floor, and
      the ellipsis count in the three `screens.golden` is unchanged at 25 / 13 /
      15. The bound is `WrapWords`' own exception rather than this function's, so
      it stays stated rather than closed.
      → `TestAWrappedRowLeavesTheWindowsLastColumnEmpty` holds the property, at
      three windows and in two word-length families, because the interesting
      value is the one that fills the row *exactly* and greedy packing reaches an
      odd length or an even one depending on the words it is given; the widest
      line emitted anywhere in the sweep is asserted to be `UsableWidth() - 1`,
      so it cannot pass by drawing everything comfortably short.
      `TestTheBiographyRowFitsTheWindowItIsDrawnIn` is the same claim at the call
      site the defect was measured on, over the cast in both languages. Both go
      red on the old arithmetic and the first names the offending row and width.

- [x] **The committed cast is not in the form the tool writes, and the test that
      was supposed to catch that is vacuous. DONE.** Both halves done apart, in
      that order: the two files reformatted in a commit carrying nothing else,
      then the property pointed at the committed file.
      ⚠️ **The old test could not fail and the new one is measured going red.**
      `TestWrittenCastIsStableAndReloads` read the *scratch* copy, which
      `scratchData` → `testfixture.Inject` had already rewritten through
      `SaveCharacter` — Marshal compared against Marshal, green whatever the
      repository held. `TestTheCommittedBooksAreInTheFormTheToolWrites` reads the
      committed file, and with the old data restored it fails on both books while
      the old test still passes. That pairing is the whole finding: **a property
      about a committed file cannot be held by a test that writes the file first.**
      The old check is kept and re-worded to the smaller claim it does hold — that
      a save puts Marshal's exact bytes on disk.

- [x] ⚠️ **Four mechanics `Suggest` resolved and did not price, measured 2026-09-02. DONE.**
      Every row below is a choice `Suggest` actually made on a fixture board, not a
      reading of the source. All four run the direction `price.go` errs in — a
      marginal cast rather than a kill — but three of them are large.

      | mechanic | field | what the rating did |
      |---|---|---|
      | ~~repeating strikes~~ | ~~`Repeat`, `MaxStrikes`~~ | **done** — `Rules.Expected` reads `ExpectedStrikes` and `hitAgainst` carries the fields; a Magnemite kit built on `spark` went **3.6% → 25.0%** |
      | ~~draining~~ | ~~`Skill.Drains`, `Passive.Drains`~~ | **done** — `pricing.drained` through `worthHealing`; the shipped replay now has Venusaur casting `leech_seed` and healing where it healed nothing |
      | unblockable | `Skill.Unblockable` | into three block charges, the blockable 700 beat the unblockable 600 |
      | attacking **into** a guard | `Shield` / `Absorb` on the target | preferred the softer target carrying a pool of 100,000 — in all three arrangements, including with no pool at all |

      ⚠️ **`Repeat` was the sharpest and is done.** It was a rule already written
      down that the rating was the one caller not to follow, and it took both
      halves: `Rules.Expected` multiplied by `h.StrikeCount()`, and `hitAgainst`
      never set `Repeat` or `MaxStrikes` on the `Hit` at all, so fixing either
      alone would have moved nothing. `worstStrikes` read the same floor and now
      reads the same count, in per mille, because a charge is worth *less* against
      a repeating attacker and the floor could not say so. `Total` deliberately
      stays on the floor — it is the deterministic column `skills.golden` is
      written from. → `TestExpectedReadsTheDistributionAndTotalReadsTheFloor`,
      `TestTheRatingReadsTheTailOfARepeatingSkill`,
      `TestAGuardIsWorthLessAgainstAnAttackerThatKeepsGoing`.

      **Drains is done too** — `pricing.drained` reads `drainShare(skill + traits)`
      over the damage `expected` says will land and runs it through the same
      `worthHealing` clamps a restore gets, so it is worth nothing on a caster
      with no room and nothing on a caster nothing can reach.

      ⚠️ **The GUARD is done.** `Battle.pastAWall` takes the block charges,
      `Battle.pastAPool` the absorbing pool, and an `unblockable` skill meets
      neither — the same three the resolution offers.

      ⚠️ **What blocked the charge half for two attempts was the INSTRUMENT, not
      the model.** Both boards used to judge it were blind: the shipped roster
      carries no guard at all, and the wall-heavy board built for it does not
      RESOLVE — `forge.Bout` refuses it, control and all, either way. A wall board
      built to actually finish, one `withdraw` carrier a side and two real
      attackers, 900 seeds:

      | | rate against the frozen ruler |
      |---|---|
      | without the charge clause | **889‰** ± 24 |
      | with it | **917‰** ± 24 |

      Outside the band. Two hypotheses about *why the balance moved* were measured
      and killed before that — `spendable` reading a guard-discounted
      `strike(mate)`, and `ArcPower` unpriced on the discharge — and the second was
      a real fix (#230) that did nothing for this. Neither was the cause.

      ⚠️ **Amortising is not a dial.** A charge cancels one strike EVER, so
      discounting a blow by the whole wall on every cast charges the same loss
      every turn, and the over-count is real. It is accepted, because every
      discount small enough to leave the balance claims standing reads INSIDE the
      band and every discount large enough to clear the band moves them — monotone
      both ways, no setting in between.

      So the balance moved, and the two claims that broke are re-derived rather
      than re-baselined:

      - **The conduit was under-armed for the board it is meant to answer.** Its
        arc is the one thing a guard does not stop, and once the rating could see a
        guard the arcs were not strong enough to be that answer: `electro_ball`
        285→430, `spark` 190→285, `overload` 180→270. The accumulating kit reads
        **426‰** against the bursting kit's 590 (floor 354), where it read 193.
      - **A wall standing in the column is the answer to a shape**, and
        `TestAShapeEarnsItsPowerWhereASparCannotSeeIt` used to hide it: its
        opposition carries `withdraw`, and a rating blind to the charge reported
        the shape winning anyway. Measured on both boards — 456/400 with nothing in
        the way, 286/485 with the wall — so the test now holds BOTH rows, which is
        a stronger claim than the one it replaced.
      - `TestAStripEarnsItsSlotOnlyAgainstSomethingToStrip` came back on its own:
        the strip now visibly reduces blocked blows, 665 against 1298.

      The gap it came from: `shielded` and `guarded` pay to *put* a guard up and
      nothing discounted a blow *into* one, so the rating bought walls and treated
      the enemy's as absent. Both halves are closed now, `warden`'s own trade
      included.

      `taunt` and `heal_cut` were the same class of omission and are **done** —
      `pricing.taunting` and `pricing.uncured`, so every one of the eleven status
      categories now has an arm in `granted` or `inflictedOn`. ⚠️ The taunt is in
      **granted**, not `inflictedOn`: the status sits on the unit DOING the
      taunting, so pricing it as harm charged its own caster for casting it.
      Measured in a squad, which is the only place a taunt can be measured at all
      — it is worth **exactly nothing** in a duel by construction, because the
      taunter is the only target there is — a wall carrying `taunt` in place of
      `withdraw` read **395‰ before and 783‰ after**, against 646‰ for the control
      that carries neither.

      Nothing is left of this item. Worth adding with the first of them: a structural test
      that every `status.Category` has an arm in `granted` or `inflictedOn`, and a
      hand-kept table of every `Skill` field marked *priced* or *deliberately not,
      with the reason* — the guard that would have caught all four at once.
      → `README.md` § *Cutting the healing* for the two already known.
- [x] **Two LAN tests fail under a loaded suite and pass alone, for two
      different reasons.** **Done** — and neither was a flake. Both were seen red
      inside `make check` and green on their own in the same working tree, they
      were recorded together because they were found together, and the difference
      between them was the useful part: one was a test wrong about its own
      premise, the other was a **product deadlock** the test was the only thing
      watching.
      ⚠️ **`TestShutdownGivesUpAndNamesWhatItWasWaitingFor` was never a timing
      flake**, which is what it looks like from the summary line: it failed in
      **0.00s** with *a shutdown on a context that was already done reported no
      error*. Its own doc claimed an already-done context left it "none of the
      timing that would make the test flaky", and that was the wrong claim — a
      done context guarantees the bound is *available*, not that anything is
      *waiting*. With one room and one socket, `stopping` drops the peer and
      `CloseAll` retires the room, so both counts can reach nought inside the
      call and there is nothing to give up on. The guard above it could not see
      that either: `Tables() == 1` says a room is open, and tables are rooms
      rather than connections.
      **The fix is a wedge, not a wider bound**: the test now takes a second
      reference on the table the way a connection does, so `Tables()` reads 1 for
      the whole call on every path and the connected count can be asserted **by
      value** — which it could not be before. Measured both ways: without the
      wedge the test goes red inside 60 runs at `GOMAXPROCS` 2 and 8 and passes
      at 1; with it, 60 runs green at all three.
      ⚠️ **It also found a real coin flip in `Server.Shutdown`.** `waited`'s
      select has two arms that are both ready whenever the last room ends around
      the moment the bound does, so the same shutdown of the same server returned
      nil or a refusal at random — and the refusal it wrote on that path read
      *"0 room(s) and 0 connected room(s) still running"*, a give-up naming
      nothing to act on, which is the exact message `gaveUp` exists to avoid.
      Both callers now decide on a **reading** rather than on the select's
      choice, and `gaveUp` **takes its counts as parameters** so the number that
      refused is the number reported.
      `TestAShutdownWithNothingLeftToWaitForDoesNotGiveUp` holds it, 200 times per
      run because a coin flip survives one.
      ⚠️ **`TestAJoinedMatchPlaysToItsEndOverALoopbackListener` was a DEADLOCK in
      the client, not a slow machine**, and widening the bound would have buried
      it. It failed at **61.22s** against a minute for work it does alone in 0.9s,
      and the instrumented run said why: the client had gone silent on the battle
      screen — live, prompt open, **already answered** — with nothing sent for the
      whole minute.
      **`session.choose` drained the answer slot before asking.** The premise was
      that nothing could be in the slot for the turn now opening, because the
      chooser had not sent `matchAskingMsg` yet. That is wrong: *"it is your
      turn"* is `socket.Mirror.Asking`, true the moment the room's batch is taken
      in — one message and one redraw **earlier** than the chooser is called — so
      a player answering off the board already in front of them lands in the slot
      first. The drain ate a real decision, `PlayScreen.Answered` meant the screen
      would not offer that turn again, and both ends stood still for a whole
      allowance. That is why it cost about a minute exactly.
      The answer now carries the turn it was pressed for (`session.pressed`, a
      pair beside `draw.PlayAnswer` rather than two more fields inside it — that
      type is `battle.Chooser`'s return pair and its own doc refuses a second
      vocabulary), and the chooser **asks what the slot is for** instead of
      emptying it. `choose`'s prompt parameter was unnamed and unread; it is the
      whole answer.
      ⚠️ **The end-to-end test alone is not enough to hold this** and never was:
      it only reddens under load, and a suite that goes green on the re-run
      teaches the next person to re-run it. Both halves are now deterministic unit
      tests — `TestAnAnswerPressedBeforeTheChooserAsksIsTakenRatherThanDropped`
      (reverting to the bare drain fails it in 5s) and
      `TestAStaleAnswerIsNotSpentOnTheNextTurn` rewritten onto two real turns
      (taking the slot unconditionally fails it in 0.00s).

- [x] **An evolution line that forks. Done** — a placement chooses **which path**
      as well as how far. `progression.Stage.After` names the stage a stage grows
      out of, so the line stops being an ordered list and becomes a **tree**; two
      arms may share a threshold, and a stage may sit past a fork on one arm only.
      ⚠️ **A line is read by order *or* by name, never both.** A line where no
      stage names an `after` is read by order — stage `i` grows out of `i-1`,
      which is what every line meant before this and why no shipped file moved.
      The moment any stage names one, every stage but the root has to, and each
      must name a predecessor **declared before it** (which is what makes a cycle
      unwritable rather than something to detect). Mixing is refused: the order of
      a file deciding parentage in a file that also states it is a wrong stat line
      rather than an error.
      ⚠️ **`Furthest` was the large half and its failure mode was silence, so it
      refuses now.** `Line.Furthest(level)` returns the tip of **every** arm;
      `StageAt` is the single-answer wrapper and errors — naming both arms — when
      there is more than one. Every caller that passes `progression.Furthest`
      already had an error path, so the whole change is compile-clean: a browser,
      a placement and a balance harness each get a refusal where they would have
      got whichever arm the file happened to list last.
      ⚠️ **`hexforge check` prices one row per arm.** The budget bites at the
      grown end of a line and a forking character has two, so `Library.Inspect`
      loops `Character.FurthestAt(LevelCap)` — art on the first row only, since
      art belongs to the character rather than to an arm.
      ⚠️ **`StageSummary` draws arms bracketed** — `Eevee@1 → (Vaporeon@32 |
      Jolteon@32)` — because joining them with the same arrow reads as a chain.
      `i18n.Lang.StageSummary` delegates to `forge.StageSummary` now instead of
      repeating the shape; the two were byte-identical.
      ⚠️ **This entry said "nothing shipped forks yet" until 2026-09-03 and that
      is stale.** `pokemon.poliwag` ships as
      `Poliwag → Poliwhirl → (Poliwrath | Politoed)` since `ed79a28`, so the
      mechanism has a shipped user and the fork's interesting cases are reachable
      from real data — which is what let the PvP gate's leaf rule be measured on
      both arms and on the interior stage rather than only on a fixture. The
      original intent (the mechanism lands without a balance move, the way crit
      did) held: `politoed` was authored as a data change of its own.
      ⚠️ **Not the same thing as the tailed-beast Naruto**, which is a separate
      *character* beside Naruto: a stage is the same unit later, and that is a
      different unit. The two want completely different mechanisms.
- [x] **A deeper opponent. Done** — see `docs/balance.md` § *Rating an action* for the rules and
      *A deeper opponent* in `README.md` for what moved. Statuses, buffs, guards,
      heals, cleanses and kills are all priced in damage now, over capped horizons,
      and the detonate setup came free with pricing the status. **Tempo followed**
      and is priced too — off the speed stat, so nothing reads the queue; see
      *Rating an action* for why a turn is worth `turnWorth` and not the best
      strike. **All-sided skills are rated too** (`friendlyFire`, the own half
      subtracted), and *holding a skill for a later turn* is answered as far as a
      one-turn-deep rating honestly can: a **tie-break on cooldown**, so a scarce
      skill is not spent on what a common one buys. **Waiting is no longer "still
      out" — it is decided against**: a pass buys no cooldown an act does not, so
      acting dominates by exactly what the action is worth, and the two available
      lookaheads each break a rule of the file. See `docs/balance.md` § *Rating an action* for
      the arithmetic. What is left is *where* in the order an extra turn falls,
      the only part that would need the queue.
      ⚠️ It cost a **balance answer, not a golden**, exactly as the summon did:
      the shipped roster went 53.1% → 79.0% ally, which is a cast finding.
      **Re-levelled since** (Charmeleon 30 / Ivysaur 30, 49.1%, and 49.4% once
      tempo was priced), so every rate quoted anywhere is on the same instrument.
- [x] **A gated grant: a stat change that comes and goes.** `blaze` is now what
      it is named after — `{"grants":[{"status":"kindled"}],"while":{"below_health":333}}`
      — and its burn immunity moved to `heatproof`, because **a gate covers the
      WHOLE trait** (grants, resists and applies together) and a trait wanting one
      gated half is two traits. Four pieces: `Set.Hold`/`Set.Release` are the
      engine-only door (each refuses what `Apply`/`Remove` handle, so neither
      becomes a second one of those, and `Remove` still refuses a permanent status
      so no cleanse can dispel a trait); `PassiveHeld` now fires mid-battle too and
      `PassiveReleased` is the way back; a **retune** at the crossing — not what
      keeps the queue right (a turn already ends with a sweep) but what puts the
      `speed_changed` NEXT TO the trait that caused it; and one re-evaluation point
      per unit whose health moved, in `battle.reconsider`.
      ⚠️ **The plan was wrong about where health moves.** Not `wound` and `heal`
      and nowhere else — **three** places: the strike loop in `resolveAgainst`
      subtracts from its target directly, and that is where nearly all the damage
      is dealt. Hooking only the two named functions opens a gate for a poison tick
      and never for a sword. It is read **per strike**, not per skill, or the same
      trait would be worth less against a multi-strike skill for a reason written
      on neither. Guard on `unit.HP <= 0` as well as `Dead`: the strike loop leaves
      a target at zero and kills it afterwards, so a flag-only guard announces a
      trait to something whose `died` line is the next event.
      ⚠️ **A gated trait is nearly a one-way door on autopilot** — `Suggest` never
      heals, the only healing in 60 bench battles is a drain to its own caster
      (~1/40 of a bar against damage worth ~1/10), and across 4000 battle-seeds of
      every arrangement tried a trait came back off **once**. So `passive_released`
      is proved by a **hand-played** battle in `TestEveryEventKindIsReachable`, not
      by widening the sweep until the rare case shows up.
- [x] **Two builds for one Bulbasaur — SHIPPED** (#114), as `bulbasaur.poison`
      ("rải độc") and `bulbasaur.parasite` ("ký sinh") in `builds.json`, measured
      by `TestTheTwoBulbasaurBuildsAreDifferentUnits` before they were listed.
      Two builds a character is the target and every Pokémon now has its pair;
      **Naruto is the one with none**, which the catalogue treats as the honest
      case rather than a gap to fill with a duplicate.
      ⚠️ **The build this entry originally described could not be built.** It
      asked build (1) to be *immune to poison* **and** *sharper* **and** *biting
      back* — but immunity and the reply are both `venom_blood` while the
      amplifier is `virulence`, and `TraitSlots = 1`. Two traits, one slot. What
      ships is the amplifier, so the poison build **hits harder and is not
      immune**; `venom_blood` is the reserve entry of that direction, the way
      `last_gasp` is of the other, and `TestBulbasaurCanBeBuiltTwoWays` checks
      only that neither direction loses its **last** trait.
      So `Resists` has a mechanism (✅, `heatproof` and `venom_blood`, `amount:
      1000` is total immunity) and **no shipped build carries one** — a third
      build is where that would go, not a second trait slot.
      Pieces, all ✅: `Resists` · *amplifying a status* (`virulence`) ·
      *answering back* (`venom_blood`) · **passive lifesteal** (`blood_thirst`,
      `last_gasp`) · `While` (`blaze` is gated).
- [x] **Passive lifesteal — SHIPPED.** `passive.Passive.Drains`, a share of the
      damage its holder deals, added to the skill's own drain and resolved where a
      drain already is. `blood_thirst` 250 · `last_gasp` 400 gated at 400.
      A gated share is the **cheaper** of the two gates (no door into a permanent
      status, no event either way, no retune), not the only legal one: #62 made a
      gated grant work too, so *Overgrow* and *Vladimir* are both writable and
      only the cost differs.
      ⚠️ Trait shares **add**, never compose — a share of damage dealt is not a
      chance — and the total is **capped at the base**. That cap is a
      *conservation* (cannot take back more than was dealt, the same invariant
      `skill.resolve` holds on one share), not the buff ceiling this engine
      rejects; saturating would pay 285 for a trait that says 400.
      ⚠️ **A reply drains too — and did not, until it was looked for.**
      `resolveAgainst` paid out and `reply` did not, so a trait holding both jobs
      promised a share of an answer it never gave. What said so was the
      description: *"mọi đòn của nó hút lại…"* / *"everything it does takes
      back…"*. `TraitSlots = 1` stops a unit carrying a replier **and** a drainer;
      it stops nothing about a trait that is **both**, and `passive.Passive` holds
      both fields. Nothing shipped is both, so **no golden moved** — the same
      shape as the regeneration bug, a job rendering in the sentences and not in
      the engine.
      The trait's share only (a reply has no skill), and **before the kill**:
      `resolveAgainst` drains whether or not the target fell, so draining after
      the return would make lethal damage the one blow worth nothing.
      ⚠️ **A `damage > 0` guard beside it was written and removed** — a mutation
      deleting it survived every test, because the branch it sits in has already
      said the reply has power and `drain` refuses a share of nothing anyway. The
      skill path's `dealt > 0` is **not** the same guard: there the sum can be
      nought after every strike missed.
      ⚠️ The `Healed` event carries **`Drained`**, the share, because `Amount`
      alone cannot say why once a trait can drain too — the `Pierce`/`Refused`
      trap again.
      Both are on Bulbasaur's learnset now (`blood_thirst`@20, `last_gasp`@40),
      which needed the trait slot first: before it a character brought everything,
      so a fourth trait made one unit better rather than two different. At the cap
      the **one** slot decides between five traits of four kinds — resist+reply,
      amplifier, stat change, drain, gated drain. `TestBulbasaurCanBeBuiltTwoWays`
      measures that the *choice* exists, and fails if `TraitSlots` grows or a
      direction loses its last entry.
      ⚠️ `Applies` is NOT retaliation, and is still not: it adds to what the
      holder's **own** attack inflicts (touch → poisoned). Retaliation is
      `Replies`, which is built.
      ⚠️ **Circular, so choose rather than discover it**: a character brings every
      trait it has, so all five pieces make ONE better unit, not two different
      ones — choosing needs the **trait slot** (*Learnsets and slots*, built), and that entry
      says a slot is only a decision once traits differ in **kind**. Suggested
      order: nothing. The gate, the reply, the drain, the amplifier **and** the
      trait slot are all built, so the five pieces are five things a placement now
      chooses **one** of — a resistance, a gated grant, a reply, a drain and an
      amplifier are five different sorts of thing, which is exactly the "differ in
      kind" the slot was waiting for. What is left is putting them on characters.
      See README → *Two builds for one character*.
      ⚠️ **`venom_blood` is gated at 24 now, and the roster figure moved with
      it.** It was the one trait on the learnset with no level, so a Bulbasaur
      had the strongest of the five from level 1. **4000 seeds: 49.5% → 53.2%
      ally.** Every other measurement in this file reading 49.5% predates it.
      The swing is one-sided by construction: `ally.venusaur` is level 60 and
      keeps the trait, while `foe.ivysaur` is 16 and loses it — a placement
      naming a trait above its level is a **hard parse error**
      (`chooseFrom`: "has not learned at level 16"), not a silent drop, so the
      roster had to change in the same commit or stop loading.
      ⚠️ **What replaced it on `foe.ivysaur` is worth almost nothing**: measured
      both ways, `endurance` gives 52.9% against 53.2% for no trait at all. The
      3.7 points are the loss of `venom_blood`, not the gap it left, so the
      choice between them is a design one — it fields none, because handing it
      another trait would swap one for another rather than show the gate, and
      `ally.charmander` already fields none.
      ⚠️ **`virulence` on `ally.venusaur` was measured and rejected: 56.3%.** It
      is the *stronger* trait of the two at the cap, so swapping the ally's
      build to compensate pushes the figure further out, not back.
- [x] **Naruto's three forms renamed to the three the story has.**
      `Naruto@1 → Shippuden@16 → Sennin@32`, against `naruto.svg`,
      `naruto-shippuden.svg` and `naruto-sage-mode.svg`: before the two years of
      training, after them, and after learning the sage art.
      ⚠️ **The count was right and both names were one form ahead of their own
      picture** — the middle was called *Tiên nhân* while showing Shippuden art,
      the last *Vĩ thú hoá* while showing sage-mode art. The art was right the
      whole way down; the labels were off by one.
      ⚠️ **The third form was authored `Tiên nhân` and is `Sennin` now, because a
      stage name is a KEY and may not be a translation of one.** `Line.Resolve`
      looks a stage up with `candidate.Name == stage`, `Stage.After` names a
      predecessor by name, a placement writes `"stage": "Ivysaur"` and a learnset
      gate lists stage names — four hand-typed spellings of one string — and
      `Line.Validate` refuses two stages sharing a name, which is a uniqueness
      constraint only a key has. *Tiên nhân* is the Vietnamese **translation** of
      仙人, so the romaji was the name that was missing rather than a new
      invention: it keeps the meaning, matches `Shippuden`'s convention exactly,
      and is right in both languages. **A stage name is still drawn raw** at
      `browse.go`, `preview.go`, `squads.go` and through `unit.Name` in
      `play.go` — that is the house rule for an id and there is deliberately **no
      `Lang.StageName`**; what was wrong was the data. `progression.ValidateStageName`
      is the refusal, at the parser rather than over the shipped data so it binds
      every line anybody writes and an authoring form can reject a name as it is
      typed — the reason `cast.ValidateID` is exported. Printable ASCII, at least
      one letter, no edge or doubled space: `Tiên` has two Unicode encodings that
      draw identically and `==` calls them different names, so a non-ASCII key
      silently misses. ⚠️ `TestTheScreensGlossEveryDataName` **still collects no
      stage name and should not** — it asserts that an id is shown *with its
      gloss*, and a stage name has none by decision; the parser is where the rule
      belongs.
      The tailed-beast form was never a stage: **a stage is the same unit later,
      and that is a different unit**, so it becomes its own character later.
      ⚠️ **No stat moved, nothing rebalanced** — a stage name is printed, never
      read, so every measurement taken against this line still holds.
      Naruto has **no build** in `builds.json` and now has two traits, so it is
      the character the "two builds each" target is waiting on.
- [x] **A permanent speed trait — `swiftness` on Naruto.** Grants `quickened`,
      permanent, **+80** speed. Naruto's second trait, so the only character with
      a default now has a choice.
      ⚠️ **The house figure for a permanent buff is 150** (`toughened`, `kindled`,
      `unleashed`) **and it does not transfer to speed** — a point of speed is
      worth more than a point of anything else, because speed is turns and a turn
      is every other stat applied again. Priced in the **share of the turn order**
      it buys: `+30` 2.6% · `+50` 4.4% · **`+80` 7.9%** · `+100` 9.9% · `+150`
      14.8% more turns than `endurance` gets in the same battles.
      ⚠️ **A win rate does not measure this and a band over one is a trap.** Over
      300 mirror duels the rate does not even *order* the amounts — `+150` comes
      back at 59.0% while `+50` reads 74.0% — because the turn queue is discrete,
      so a few points of speed buy whether one more turn lands before the other
      unit acts, and that is lumpy per seed. `Suggest` casting no-power skills put
      a summon in the queue too and made the lumps larger. **At 150 the win-rate
      band passes and the turn-share band fails**, which is how it was caught.
      ⚠️ The first turn test compared **two separate sweeps' totals**, which
      measures battle *length* — and the faster unit ends its battles sooner. It
      passed until `Suggest` changed, then reported swiftness with **fewer** turns
      while being exactly as fast. Both sides of **one** battle is the comparison
      that cannot say that.
      ⚠️ Naming: `haste` already glosses "nhanh nhẹn" so `quickened` is "gia
      tốc" (it read "thoăn thoắt" first); the trait was "nhanh chân" until
      **`chân` turned out to be a `bodyWord`**, and is "thần tốc".
- [x] **The budget bounds a line nobody fights on**, and `hexforge check` now
      says so: a second table of the line each character actually fights on, per
      trait, via `forge.Library.Held`.
      ⚠️ `progression.Limits.CheckValues` takes **six numbers and nothing else**,
      so the bound is on the paper line. A trait is named on a *placement* and its
      grants go on at enlistment — **`battle.New` rejects a base line of 740
      defence and then hands the same unit 786 through a trait, in the same
      call.**
      ⚠️ Only **permanent** grants count, which is all a trait can give
      (`status.Set.Hold` refuses a timed one). A gated trait (`blaze`) is skipped:
      its condition reads a health no character has outside a battle.
      ⚠️ **SETTLED: the bound is the PAPER line's.** A ceiling and the budget
      bound what an **author** writes at the cap; going past them in a battle is
      the point, not a leak — for a buff, a trait, and whatever a rune becomes.
      What holds the **fought** line is the **saturation**: `ceiling × headroom`,
      so nothing reaches 3× a ceiling however much is stacked.
      So this prints a figure and raises nothing. `TestNoTraitCarriesACharacterFarPastTheBudget`
      is a **tripwire not a bound** — 120%, shipped worst Squirtle/`ballast` 113.4%.
      Found the moment the table existed: **Bulbasaur/`endurance` has 157 left**;
      `reckless` is *under* the bound (`bare` costs more than `unleashed` buys).
      ⚠️ Three existing guards caught the first wording: the TUI's 79-cell width
      test, the renamed-label ban (**"absorbs" is banned in English**), and the
      short-reach test's `len(Warnings) != 0`.
- [x] **Speed cannot reach nought, and now something says so.** Four guards stood
      between the data and a unit that never acts again — the floor at a tenth of
      base (approached, never reached), `Stat`'s return of 1, `atb.Wait`'s clamp,
      and `atb.Queue.Add`/`Reschedule`'s — and none of them stated the invariant.
      `TestNoShippedDebuffCanFreezeAUnit` stacks every harmful shipped status 50
      deep on every character and asks whether the queue still turns.
      ⚠️ **`max_stacks` binds long before the floor.** `expose` caps at 2, so
      Squirtle's defence bottoms at **410 of 640** — the floor is 64, ~6× further.
      To strip armour harder the levers are the amount and `max_stacks`, **not the
      floor**, or `pierce` (the counter armour was given).
      ⚠️ **`TestStatNeverDropsBelowOne` was passing for the wrong reason** — it
      crushed a base of 3, where the saturation lands above nought by arithmetic
      and the branch is dead; deleting the branch left it green. **Only a base of
      nought reaches the guard** (Saturate gets a gap of nought and returns the
      base), which is real: a summon is authored with a fixed line and nought
      dodge is ordinary.
- [x] **Two Squirtle builds out of one learnset**, in data only — three skills
      (`skull_bash`, `wide_guard`, `water_pulse`), one trait (`ballast`), two
      permanent statuses (`fortified`, `encumber`), no engine change at all.
      A placement spends 4 skill slots and 1 trait slot, so two kits out of one
      learnset is what that system is for.
      ⚠️ **The stat line cannot differ**: Squirtle absorbs 11285 of the 11500
      budget. The split has to live in the slots because it cannot live anywhere
      else.
      `skull_bash` is the **first shipped skill to scale off anything but
      attack** — `skill.Scaling` had no shipped user until now. def 640 / atk 460
      = **1.39×** a point of power, so a defence-scaled skill wants ~0.72× the
      power of an attack-scaled one.
      ⚠️ **`ballast` is the attacking build's trait and not the tank's**, which
      is measured, not intended: survival is gated on how often `withdraw` can be
      cast, so **a tenth off speed is worth more than a quarter onto defence** —
      the tank build survives 1/30 with `ballast` against 29/30 with `endurance`.
      ⚠️ **A test may not raise a stat**: `battle.New` checks the roster line
      against the budget. `skull_bash`'s scaling is proved by *halving attack*
      (figure comes back **exactly equal**, while `water_gun` halves). A stat
      falling and damage falling with it proves nothing — the unit also dies
      sooner and swings fewer times.
      ⚠️ **A trait's permanent statuses are outside the budget** — `CheckValues`
      takes a resolved stat line and never sees a passive. `endurance` was
      already through that gap.
      ⚠️ Tank + `endurance` is **unkillable in a duel** (29/30 hit the 4000-turn
      cap, dealing nothing); a squad puts five times the damage into it.
      ⚠️ Both build figures are understatements: `Suggest` takes a no-power skill
      only when it can find nothing to hit, so the tank kit is exercised only
      because it carries no weapon. `hexforge spar` cannot measure either build.
- [x] **A summon says how strong it arrives.** `describeSummon` prints the
      **share** and still not the fixed stat line, because they are not the same
      kind of number: a share is one figure that means the same thing wherever it
      is read, and a fixed line is six nobody can compare without the caster.
      ⚠️ It was left out because "the listing beside this carries it" — **no
      listing does.** Neither `hexforge` nor `hexforge-tui` mentions a summon at
      all, so the sentence is the only place one is described.
      ⚠️ **`share` and `share_of_base` must read differently** — one rewards
      buffing before the cast and one ignores every buff — and **one copy and
      several need different wordings**, since a pair handed the singular reads as
      a pair carrying 40% between them. Comparing the two descriptions does not
      catch that: they differ in their count either way, so the test asks about
      the share clause by splitting the wording on its blanks.
      ⚠️ **A creature is not a copy.** English falls back to a word, and the word
      was "copy" for everything: the shipped toad read as a copy of the ninja who
      called it. Told apart by the stat spelling, which is the line the engine
      already draws.
      Data rules that came out of it: a summon's flavour may claim nothing about
      its caster (`casterWords` — both shipped Naruto summons did, and the toad's
      "to hơn cả người gọi" is contradicted by its own stat line), and a
      one-strike flavour may describe no volley (`volleyWords` — `kunai` said "một
      nhúm", which is the count said twice **and** the wrong weapon, since a
      handful thrown at once is the shuriken beside it; renamed `phi đao`).
      `chùm` is off that list on the judgement that kept teeth out of `bodyWords`.
- [x] **A taunt: the choice of enemy taken away, not the turn.** Status
      `taunting`, and `Battle.aims` narrows an **enemy-aimed** skill to whoever is
      taunting. `Suggest` obeys it **with no AI change at all**, because it reads
      the aims it is offered and nothing else.
      ⚠️ **The status sits on the TAUNTER, not on the taunted.** A taunt held by
      its victim would have to remember *who* taunted it, and a `status.Stack`
      **deliberately does not remember who applied it** (that is what keeps a
      stack worth the same after its author has died). Held by the taunter it
      needs no memory: "who must I attack" is read off the board, and a corpse is
      not on it — `TestATauntDiesWithItsTaunter` needs no cleanup path.
      ⚠️ **Range is not read.** Nothing on this board moves, so a taunt that could
      be answered by standing far enough away would be ignored by exactly the
      long-ranged attackers a tank most needs to pull. A range-1 skill aimed at a
      taunter four cells away lands: nothing past the *legality* of the aim has
      ever read distance.
      ⚠️ **A new category `Taunt`, not a second `Control`.** Stun means "you do not
      act"; taunt means "you act, and may not pick" — opposite things to a turn. A
      category exists so a cleanse can name a class, and "strips a control" taking
      a taunt off with a stun is a cleanse nobody could aim. It also stopped the
      reference printing *"the holder loses its turn"* under `taunting`, which the
      shared category had it saying. Declared **last** (serialises by name, but
      `CategoryCount` and the grouped listing order are declaration order).
      ⚠️ **`Category.Harmful()` must include it** — that gates what a trait may
      **resist**, so a taunt outside it makes "cannot be provoked" unwritable.
      Nothing else noticed: the mutation passed every other test in the repo.
      ⚠️ **A taunt is spent on the TAUNTER's own turns**, so a fast taunter wastes
      it before a slow victim ever acts. The slowest unit in a squad makes the best
      taunter — which is Squirtle (speed 85, slowest in the cast) exactly.
      A taunt only ever *unfreezes* a stalemate (it adds an aim for a unit that had
      none), so `frozen()` needs no change; it already goes through `aims`.
      Shipped `taunt` on Squirtle@40 (self-aimed, cd 3).
- [x] **A reply priced off the stat its holder actually has.** `passive.Reply`
      gained **`Scaling`** (a `skill.Scaling`, so stat *and* base-or-current),
      authored as `"replies": {"power": 80, "scaling": {"stat": "defense"}}`.
      ⚠️ **Attack was the wrong default, not a missing field.** A trait that
      answers whoever hit it belongs to a unit **built to be hit** — armoured, not
      sharp — so pricing every reply off attack made thorns worth least to exactly
      the character thorns are for. Blastoise is 640 defence / 460 attack: the
      same share off the wrong stat is a **third** less.
      ⚠️ **`origin.Scaling` is now the whole `skill.Scaling`, not a bare Kind**,
      and all three read sites go through `origin.stat` → `combat.PickScaling`.
      That fixed a latent bug nobody could reach: a skill declaring
      `"source":"base"` had its damage read the base line and its **DoT tick read
      the current one**. No shipped skill declares scaling, so it was unreachable —
      and would have arrived the day one did.
      ⚠️ **`skill.ParseScaling` is exported** so a trait and a skill are read by one
      parser. Health stays refused wherever it is asked for (damage that grows as
      its owner is healed). Refused too: a reply with **no damage** naming a stat.
      ⚠️ **The word "attack" was hardcoded in THREE places** — the engine, the
      i18n reply blurb, and `hexforge passives`. The listing's is in no golden and
      nothing else reads it: the mutation that put the literal back **passed the
      whole suite**, and an author would have tuned a thorns trait against a
      number a third out. `TestThePassiveListingNamesTheStatAReplyIsPricedOff`.
      Shipped `thorns` (8% defence) on Squirtle@32. Measured in a duel — thorns'
      best case, since the holder is attacked every turn it lives: **14.2% →
      15.3%** overall, Charmander matchup 28.5% → 30.7%. ⚠️ **Squirtle still loses
      to Bulbasaur 0% either way**, which is a cast problem this does not touch.
- [x] **Answering back — the fifth job.** `venom_blood` now costs whatever bit
      into it: `"replies": {"power": 40, "applies": [{"status":"poison","chance":25}]}`.
      ⚠️ **Not `applies` reworded** — that fires on a target the holder chose,
      during the holder's turn; a reply fires on the **attacker**, on somebody
      else's turn, from a unit that is not acting. **Not a second damage path**:
      damage through `combat.Rules.Damage`, statuses through `battle.inflict`, so
      the same resistances, the same rolls and the same event kinds. `inflict` no
      longer takes a `skill.Skill` but an **`origin`** (`{Skill|Passive, Element,
      Scaling}`) — the three things it ever wanted from one — which is what lets a
      reply share it rather than fork it. A reply has **no element and no
      accuracy**: it is neutral and it lands, because the chart prices what one
      creature *threw* and an accuracy roll asks whether contact happened, which
      it already has.
      Four rules, three of which are just where `b.answer` sits (after the whole
      skill, once, per holder): a reply **may kill** (a battle can end on a turn
      nobody took); a reply **never triggers a reply** — closed because the
      answer list is built from the skill's own targets and a reply is not one,
      *not* by a depth counter; **once per USE, not per strike**; **every target
      takes the whole skill first**. A holder the skill killed does not answer,
      and once one reply kills the attacker the holders behind it do not either.
      ⚠️ **A reply is priced by how often its holder is attacked and how long it
      survives, and no number on the trait can say that.** Both Bulbasaurs hold
      `venom_blood`, but the ally fields Venusaur at 60 and the enemy Ivysaur at
      16: over 4000 battles the ally makes 69% of the replies and deals **86%** of
      all reply damage. That is the roster's asymmetry, not the trait's — and it
      caps how big the trait may be. Measured: power 40 alone 49.5%, **40+25
      (shipped) 51.9%**, 40+200 73.5%, 250+500 **98.3%**. Tune over thousands of
      seeds; the 40-seed sweep cannot see a move this size.
      ⚠️ **The counter-damage was later sold to buy chance, and `venom_blood`
      answers with poison alone now** (`{"applies":[{"status":"poison",
      "chance":40}]}`, no `power`). 20 000 seeds: **25‰+power 40 → 53.0%**,
      **40‰+power 0 → 53.1%** — the same cost, and the poison lands **0.27 →
      0.44 per battle**, 63% more often. 50‰ alone was measured at **56.1%** and
      refused; 50‰+power 0 at 54.3%. ⚠️ **σ ≈ 0.35 points at 20 000 seeds**, so a
      gap under **0.7** is noise and the 4 000-seed sweeps used earlier cannot
      resolve one.
      ⚠️ **No shipped trait answers with damage any more** — `venom_blood` was
      the only replier in the cast, so a battle from the shipped roster emits no
      `Damaged` carrying a trait. `TestTheShippedRosterAnswersItsAttackers` was
      narrowed to the status half and logs the damage count instead of asserting
      it; the mechanism stays covered by `internal/core/battle/reply_test.go`,
      whose fixtures exist for it. The bench roster never covered it at all
      (`endurance` and `blaze` only).
      ⚠️ **It flipped the sign of a figure already in the README.** Removing
      `razor_leaf`'s pierce used to cost the ally 2.7pp; it now gains 1.1pp
      (51.9 → 53.0), at every reply size tried. Piercing helps whoever attacks,
      and this is the first thing that charges for attacking — so **no balance
      figure measured before this feature carries forward**.
- [x] **A condition a skill reads about ITSELF.** `skill.Skill.SelfRequires`, the
      same `*Condition` as `Requires` asked of the caster instead of the target —
      `Applies`/`SelfApplies` spelled again, two fields rather than one field with
      a "whose" flag. Until it existed, "hits harder while I am furied" and "hits
      harder while I am cornered" had **no spelling at all**.
      ⚠️ **Read ONCE PER USE, in `Act`, not in `resolveAgainst`** — that runs once
      per cell a shape covers, so a consumed condition would charge a column three
      times and a single-target skill once, for a difference written on neither
      skill. `Battle.spend` is that seam, and it sits **before `applyToSelf`** so a
      skill that grants and spends the same status cannot pay itself.
      ⚠️ The bonus is added **before** the splash share is taken, like the target's.
      A bonus added after still makes every target take more, which is why that
      mutation survived the first draft of the test — assert the **ratio between
      the aim and the edge**, not the rise.
      ⚠️ **`conditionCaster` is a second builder beside `conditionTarget`**, not a
      parameter on one: a skill may read one status of its target and another of
      itself, so a single builder would have to be *told* which condition it was
      reading — the exact reading-vs-resolution mismatch `conditionTarget` exists
      to prevent. `Suggest` reads it once outside its own loop, where it is read
      for real.
      ⚠️ **`resolveCondition` is now one validator for both fields**, carrying the
      field name through every message: two copies would be two sets of rules and
      the looser is the one an author finds. `TestBothConditionsAreRefusedTheSameWay`
      runs one table through both. Also refused: a bonus power on a **self-aimed**
      skill, which deals no damage for it to land on.
      ⚠️ **The reference table read only `Requires`** and would have told an author
      their skill has no amplifier while the engine amplified it — `skills.golden`
      gained a **whose** column. Shipped on `outrage`: 2200 → **3400 below 40%
      health**, a gate that fires under autopilot where a `fury` payoff never
      would, and one that pairs with the frailty `reckless` buys. Dragon-vs-fire
      unmoved at **42.5%**.
- [x] **A health threshold a *skill* can read.** `skill.Condition.BelowHealth`
      (permille) reads the **target**; `passive.Condition` reads its **holder**.
      `brine` moved onto it and is finally its canon move — 1000 power → 2000 at or
      below half health. ⚠️ **They share the arithmetic, not the type**:
      `scale.AtOrBelowShare` is the one comparison and both call it. One shared
      `Condition` could not say *whose* health it meant (and `passive` imports
      `skill`, so it was unwritable). A condition may read a status, health, or
      both — **both is AND**, so a clause narrows rather than widens. Refused, not
      defaulted: asks nothing; stacks with no status; share outside 1..1000;
      consumes a status it never names. The reading is a **`skill.Target`** struct
      (stacks, health, maximum) — not three params, because two int64 healths swap
      silently; `Amplified`/`PowerAgainst` take it, `Condition.Satisfying()` is the
      cheapest holding target for previews/reports. ⚠️ `battle.conditionTarget` is
      the **single** builder because `Suggest` and `resolveAgainst` must read
      identically, else the AI prefers a bonus it does not get. The **gradient** —
      smoothly harder the further the *caster* fell, rather than a threshold — is
      the separate feature below; `SelfRequires` stays the threshold version.
- [x] **A damage gradient off the caster's own health. Done** — `self_gradient`
      with one number, `at_empty`, the share it adds at the bottom of the bar.
      `combat.Gradient` is the arithmetic; `comeback` is the first user (900 power,
      1710 with nothing left). ⚠️ **A multiplier, not a bonus, which is why it is
      in `combat` and not a fourth field on `Condition`**: a bonus would have to be
      added to the *declared* power, which is not what a skill lands at once a
      detonate has amplified it; a share scales whatever power the skill arrived at,
      so the two compose instead of arguing about order. A condition could not
      express it at all — a condition answers yes or no.
      ⚠️ **`combat.Gradient` returns the share ADDED, not the multiplier**, so
      nought means "nothing happened" in the struct, in the log and in the tables —
      the shape `Pierce`/`Refused`/`Drained` already have, and the reason every log
      written before this is byte for byte what it was. `swing.applied` adds
      `PermilleBase`, once.
      ⚠️ **Read once per USE, and here that seam has teeth.** `Battle.spend` records
      the rule for a *cost*; a gradient has no cost to pay twice, so the rule looked
      like tidiness. It is not: **a draining skill heals its own caster inside the
      loop that walks a shape**, so a per-cell reading gives the second unit in a
      column a softer swing than the first, written on no skill.
      `TestTheGradientIsReadOncePerUseAndNotOncePerTarget` catches it as a **ratio** —
      edge and middle differ by design, but being hurt must multiply both the same,
      and a re-read hands the edge ~1180 per mille where the middle got 1500.
      ⚠️ **`swing{Bonus, Share}` replaced the bare `spent int`** that
      `resolveAgainst` and `against` both took. Two adjacent ints in every
      signature: handing the bonus where the share goes compiles, divides the power
      by a thousand, and reads as a balance change. `swingOf` is the single reading,
      for the reason `conditionTarget` is.
      **The share is on `skill_used`.** `Power` there is what the skill *declares*,
      so without it a hurt caster's strike lands for more than the log states with
      nothing to bridge them — worse than a pierce, which is at least the same on
      every cast.
      **One new refusal that is not arithmetic:** a gradient beside a
      `self_requires` reading **health**. Two curves off one number is unpriceable
      and unreadable; a threshold on a *status* composes fine, which is why the rule
      asks what the condition reads, not whether there is one. **No upper bound**,
      unlike `pierce` — piercing past all the armour is meaningless, a share added
      to power has no such ceiling.
      ⚠️ **A mirror duel that swaps SIDES rather than KITS measures itself.** The
      queue breaks a tie by enlistment, so leaving the roster order alone enlists
      the first-written kit first in *both* halves: a unit against an identical copy
      of itself read **58.8%**. Swap the kits.
      `TestTheMirrorIsFairBeforeAnythingIsMeasuredThroughIt` demands exactly even
      before anything else in that file is believed. ⚠️ And the swap that
      discriminates is against **rasengan**, not against the kit's filler — dropping
      `kunai` wins ~75% at *every* power from 500 to 1100, because a 700-power
      cooldown-0 skill makes the fourth slot nearly free.
      ⚠️ **The "blocks the form does not ask about" list was wrong in all three
      places it was written down** (`internal/screen/skills.go`,
      `internal/forge/skills.go`, `TestTheShippedSkillBookSurvivesBeingWritten`):
      each named `self_applies`, which the form *does* ask, and none named
      `self_requires` or `summons`, which it does not. Corrected, with
      `self_gradient` joining them.
      ⚠️ **`forge.PreviewDamage` used to read only the target's condition**, so
      neither `self_requires` nor `self_gradient` showed in the authoring preview.
      Fixed since — one change covered both, and the composition moved into
      `combat.Swung` so the preview and the battle share the expression.
- [x] **Learnsets and slots.** A character now holds a **learnset** —
      `Character.Skills` is `[]cast.Unlock`, the *same type as the traits* — and a
      placement **chooses** from it: `SkillSlots = 4`, `TraitSlots = 1`, in
      `internal/seed/roster.go`. One `{id, at_level}` shape, one validator, one
      `UnlockedIDs`, and even one renderer (`forge.UnlockSummary` draws a kit and a
      trait list alike, `razor_leaf poison_powder@8`).
      ⚠️ **`skills`/`passives` on a reference entry changed meaning**: they used to
      be a restatement of the character sheet and were REFUSED as one; they are now
      the loadout. Refused instead: naming nothing (a slot is a decision — no
      default), naming more than the slots hold, naming one twice, naming what the
      level has not learned. A refusal lists what *was* available.
      ⚠️ **The kit is required and the trait slot is not**, and that asymmetry is
      deliberate: a unit with no skills cannot act, so an empty kit is never a
      choice; a unit with no trait is ordinary, so insisting would make "the plain
      version" unwritable. `required`/`optional` are named constants at the call
      site for exactly this.
      ⚠️ **An archetype's kit stays `[]string`** — a preset has no level to gate
      against, so it is a suggestion for authoring; `cast.Learn` turns it into a
      learnset when hexforge builds a character from it, all at level 1.
      **`battle.Roster` is unchanged** and still takes a resolved kit: a learnset
      settles before a battle exactly as an evolution does.
      **The log now carries the placement.** `battle.Log.Roster` holds the resolved
      roster and `--verify` rebuilds from it, because once a placement picks four
      of nine, re-running the embedded data would compare two different battles and
      call the difference corruption. It carries the RESOLVED form, which makes a
      log readable across a data edit. `Log.Replayable()` is false for a log
      written before this: it still renders (that reads events only) and refuses to
      verify, saying why. `battle.Roster` gained json tags so the log is snake case
      like every other file here.
      **Measured, 4000 seeds:** roster **49.5%** ally (was 51.9 before slots), 0
      stalls, longest 63, and only **2.2%** of turns idle — not the 30% the design
      note predicted, because that was a level-1 two-skill unit and the youngest
      shipped unit is level 8 with three. ⚠️ Pierce flipped sign *again*: 49.2→46.5
      before replies, 51.9→53.0 with them, **49.5→46.0** with slots. **No balance
      figure carries across a feature.**
- [x] **Choosing to evolve.** A level **allows** a form; the placement names
      which it fielded. `Line.Resolve(level)` → `Resolve(level, stage)`, plus
      `Line.Allowed(level)` and `progression.Furthest` (the empty string, named)
      for every caller that has no placement behind it — a browse screen showing
      a character at level 30 is describing, not fielding. Roster entry gained
      `"stage"`, optional, absent = furthest, so an older roster still says what
      it said.
      ⚠️ **A form ahead of the level is REFUSED, never clamped** — a clamp fields
      a different unit from the one written down. Two refusals, told apart: a name
      the line does not answer to is a typo; a name merely ahead of the level is a
      placement that has not grown into it.
      ⚠️ **`at_stage` was never built and is not wanted.** The second gate is
      `Unlock.Stages`, an **allowlist**: a threshold could only say "from this
      form on", which a level already says, so everything an early form knew a
      grown one knew too and giving up an evolution bought nothing. A list says
      `["Bulbasaur","Ivysaur"]` — a move Venusaur never gets. Same reason
      `skill.Restriction` is an allowlist. `at_stage: "Ivysaur"` is just
      `stages: ["Ivysaur","Venusaur"]`; two fields would be two vocabularies.
      Refused: a form the line lacks · one named twice · **every** form (that is
      what naming none means) · a stage list on a preset (no line) · a character
      that learns nothing its **first form at level 1** can use.
      `forge.UnlockSummary` prints `sleep_powder@12[Bulbasaur,Ivysaur]` — the mark
      shows at every level, unlike a level gate, because it never stops being true.
      ⚠️ **The shipped roster does not take the trade.** Every unit is fielded as
      its furthest form, balance is unchanged at **49.5%** and `replay.golden` did
      not move. None of the three characters is close enough for one kept skill to
      pay for a stage of stats — a cast-tuning question, not a mechanism one.
      **Conditions beyond a level stay out of scope**: items, a friendship count,
      battles fought — each needs somewhere to persist between battles, and there is
      no meta layer, no inventory and no save. A level is what a character sheet knows.
- [x] **Looking a status up — SHIPPED.** `Lang.DescribeStatus` beside
      `Describe`/`DescribePassive`, derived, both languages, and three front-ends
      read the same sentences: `?mire` / `?*` at the battle prompt, `hexforge
      statuses` (the sixth listing), and `screenStatuses` in the tool.
      ⚠️ **A life, not a tick, and one stack's life — not the ramp.** Poison ticks
      50% to burn's 80%, so the tick alone ranks them backwards; over their lives
      it is 150/160 for one stack and **450/320** at their caps. `skills.golden`
      prints a *third* figure under the same words — the full ramp of a status
      reapplied every turn (600% for poison) — which needs a skill off cooldown
      every turn and no shipped kit has one. Two numbers, one phrase: the golden's
      is an author's ceiling, the reference's assumes nothing.
      ⚠️ **Permanent is "always"**, and a one-stack status gets **no rate**:
      `toughened` reads *tăng thủ 15%*, not *15% mỗi lớp*
      (`TestAPermanentStatusIsNeverGivenARate`).
      ⚠️ **Grouping lives in core** — `status.Book.Grouped` — because three
      front-ends working it out is three answers to "which category is this in".
      In the tool the headings are **rows**, since the listing scrolls and a
      heading drawn between rows falls off the top; the price is a cursor that can
      land on one, which `TestTheStatusCursorNeverLandsOnAHeading` refuses.
      ⚠️ The caveat ("these are the book's figures, an amplifier or a resistance
      changes what lands") is printed **once** per reference and is the **last**
      line, and `frame` cuts from the bottom —
      `TestTheStatusCaveatSurvivesTheSmallestWindow` measures it at 120x24 in both
      languages.
      ⚠️ Adding a screen to `hexforge-tui` means adding it to `everyScreen` in
      `language_test.go`, or every width and translation test silently skips it.
- [x] **Reading a trait — SHIPPED.** `?` on `screenBrowse` raises `screenBlurb`
      for the traits the character under the cursor holds **at the level it is
      walking**, and `hexforge passives` gained the `answers` and `drains` columns
      it never had — two of the six jobs the parser accepts had rendered **nowhere**
      in the tool, so `blood_thirst` printed a row blank after its name.
      ⚠️ **One screen, not two.** Which screen is behind is what `esc` had to
      answer anyway and used to answer with a constant, and it is **not a
      cursor**. A second screen would be a second copy of the framing, the footer
      and the escape.
      ⚠️ **It used to be the single field the screen kept, then it was read only
      by the client, and now it is gone** — the subject it was handed replaced it,
      because reading `m.browse`, `m.skills` and `m.play` is what made this screen
      unmovable, and `model.raisedFrom` replaced the way back once all three
      raisers returned a `draw.Raise`. The describer branches on `subject.Kind`.
      ⚠️ **It scrolls, and `scroll` is still not the refused cursor.** A cursor
      could point at a different character than the browser behind it; an offset
      selects nothing and every key that changes *what* is described resets it.
      Five traits at the cap wrap past 120x24 — the declared floor, not an odd case
      — so the frame would eat the last one.
      ⚠️ **Wrap to `minWidth`, NOT to `m.usableWidth()`** — the opposite of
      `m.wrapped`, which carries authored free text and takes whatever width there
      is, less the one column at the end of it that every row here leaves empty.
      ⚠️ **That last clause was missing from the code as well as from this line
      until it was measured.** `screen.WrappedIn` spent `- 2 - width - 1`, a cell
      more than every other row, so a wrapped value filled the window's final
      column — the column `frame` leaves empty precisely so a full-width line
      cannot wrap. It spends `UsableWidth() - 1 - marker - width - 1` now; the
      measurement, and why no golden moved for it, is in `TODO.md`.
      These are the program's own prose: `TestEveryWordingFitsTheMinimumWidth`
      renders at width 200 and measures against the floor less one (79 when that
      line was written, 119 now), and free text is excused while
      a derived sentence is not. Unwrapped, the reply line was cut mid-word at the
      floor ("…3% khả nă").
      ⚠️ **The `answers` column is ONE cell** — `DescribePassive` writes one
      sentence for a whole reply on purpose; a damage cell filed away from a status
      cell leaves a reader adding it up.
      ⚠️ Adding a screen (or a *state* of one) to `hexforge-tui` means adding it to
      `everyScreen` in `language_test.go`, or every width and translation test skips
      it in silence. Both blurb shapes are in it now, and `screenPreview` is too —
      it was the fifth and last screen outside the sweeps, and it went into both
      clients' `everyScreen` and into `everyMovedScreen` with a golden entry of its
      own. ⚠️ **It is the one entry whose picture is exempt from the width sweep**:
      a drawing is `usableWidth() - 2` wide by construction, so a floor has nothing
      to say about it, and the sweeps tell art from wording by the ramp's alphabet
      (`aPictureRow`). The wording around it — heading, the art/level/stage line,
      the footer — takes the floor like every other sentence. ⚠️ **A golden is taken
      under `NO_COLOR`, so it records `rampCell` and can never see `blockCell`**;
      the coloured half is held by `TestEachPixelIsDrawnInItsOwnHalfOfTheCell` and
      the ramp's weights by `TestTheRampWeighsGreenOverRedOverBlue`, both in
      `internal/screen/preview_test.go`.
      ⚠️ **A screen registered on a linear character measures nothing about a
      fork, and for a while that was the whole of the coverage.** The preview went
      into the sweeps pointed at the first row of the cast, which does not fork, so
      the one shipped character that could not be drawn at all stayed invisible to
      every record. Both clients now register `a forked art preview` and
      `a forked trait blurb` through `theForkedBrowser`, which **finds** the fork
      in the shipped books and is fatal when there is none — a helper that quietly
      settled for a linear character would turn "the data changed" into "these
      entries measure nothing". ⚠️ Their pictures also made the width sweeps skip
      **blank** rows: a drawing's transparent margin is a full-width run of spaces,
      and `aPictureRow` refuses a blank row on purpose because a count of painted
      rows reads the same predicate.
      **And the `nội tại` menu**, `screenPassives`: every *declared* trait with the
      description of the one under the cursor, which is the other question — `?`
      on the browser is filtered by a level, so a trait nobody has learned yet is
      reachable from nowhere.
      ⚠️ **The column is "who carries it", NOT "who may".** A trait has no
      restriction mechanism, so *may* is everybody. `Library.TraitCarriers` walks
      the **cast**, not the trait book: the edge lives on the character, and an
      index the other way round is a second place for it.
      ⚠️ **Not a column on `hexforge passives`** — nine columns already, and a
      carrier row is as long as the cast. A clippable row and a cursor are what
      make it affordable; the CLI has neither.
      **And the name in the sentence is a door.** `?` on `screenPassives` opens
      `screenStatuses` at the status the trait names, `esc` comes back, and the
      name is marked (bold, no colour) where it is printed so `?` has something
      visible to be about. `i18n.StatusesNamed(trait)` serves both halves: the ids
      a description will name, in the order it names them. Reading the sentences
      back instead would be substring matching against prose in two languages —
      it styles a name that happens to sit in a flavour clause and misses one the
      glossary lacks, since that one prints as a bare id.
      ⚠️ **It is a second reading of `DescribePassive` and drifts silently.**
      `TestATraitNamesEveryStatusItsDescriptionNames` holds them together; the
      rule the shipped book cannot show is pinned apart — **a reply names its
      first application and no more**, because one sentence has room for one
      status.
      ⚠️ **`blood_thirst` and `last_gasp` name nothing** (a drain names no
      status), so `?` must stay put rather than open whatever the status cursor
      sat on.
      ⚠️ **`statusesScreen.from` must be cleared where it is STORED.** The first
      version returned before the assignment that puts the screen back on the
      model, so a later visit through the menu inherited the earlier visit's way
      back. Caught by a test, not by reading.
      ⚠️ **Marking is ONE left-to-right pass, longest match wins** — a pass per
      name re-marks its own output in either order (`bỏng` inside the `bỏng nặng`
      just produced; or `bỏng nặng` never matching). And each **word** is marked
      whole, not the phrase: the sentences wrap afterwards and the wrap splits on
      spaces, so a style spanning two words breaks when they land on different
      lines.
- [x] **A regeneration that heals — SHIPPED.** `regrowth` was declared, glossed,
      described and **inert**: `inflict` computed a tick only for `status.Dot`, so
      a `Regen` stack went on carrying nought and every step below it was already
      correct and never reached. One branch:
      `tick = b.books.Rules.Restore(b.Stats(actor)[from.Scaling], kind.TickPower)`.
      ⚠️ **`Restore`, not `Damage` — it drops two terms deliberately.** No defence
      (`combat.Rules.Restore` says why: armour turns away what comes *at* a unit,
      so dividing lets a unit's own armour weaken its own regeneration) and no
      elemental multiplier (the chart prices what one creature threw at another; a
      grass unit healing a fire ally throws nothing). Kept: the actor's scaling
      stat and the **freeze** — the promise `status.Regen` already made and
      nothing honoured.
      ⚠️ **The old note misnamed the casualties, and the correction was wrong
      too.** The two skills fixed *here* were **`aqua_ring` and `ingrain`**, and
      neither had a working half: power 0, no `restores`, nothing but the
      regeneration, so casting either did *nothing*. What this note then claimed —
      "`synthesis` was never affected, it heals through `restores`, which always
      worked" — was **false**. A `restores` payout sat inside `resolveAgainst`,
      which `Act` returns before for a `Target: Self` skill, so `synthesis` healed
      nothing either, and `withdraw` paid out its block and dropped its five
      hundred. **Both** halves of the shipped healing were inert at once, by two
      different routes, and the note correcting one of them asserted the other was
      fine. → *Healing is not damage with a sign*.
      ⚠️ **A second bug was hiding behind the first**: `tickStatuses` named the
      status that healed and then healed again from the total, so every tick would
      have logged **two** `Healed` events. Healing is now applied per entry and
      `heal` carries the status id — one event saying what healed, how much landed
      and the health left. Damage stays on the total (`wound` emits nothing, and
      has no name to carry). Per entry is also the truthful arithmetic: `heal`
      stops at full health, so a second regeneration is clamped where one total
      would have hidden it.
      ⚠️ **The two decisions, made.** A regen scales off the applying skill's stat
      read live off the actor — the same expression the Dot branch uses. And an
      amplifier does **not** raise a regen: the refusal in `passive.Amplification`
      stays, but on the other ground, because its old reason ("a multiplication of
      zero") expired with this fix. The share reads *"its poison ticks 30% harder"*
      in both languages, and a share that heals under that sentence is a
      description that lies. Lifting it is a wording change first.
      ⚠️ **No golden moved, and that is the finding.** The plan expected a balance
      diff. **No roster unit fields either skill**, and `Suggest` never chooses a
      self-cast regeneration on a unit that can always reach somebody — those two
      facts together are why a shipped skill did nothing this long with every test
      passing. Proof is hand-played (`TestTheShippedRegenerationHeals` on the
      shipped books) plus eight tests in `internal/core/battle`. Fielding a regen
      is a **balance** decision and belongs with the cast work.
      ⚠️ **One of the eight was worthless as first written.** The order test
      asserted `healed` came before `status_ticked`, and a mutation putting damage
      first **survived**: `wound` emits nothing, so the events come out in the same
      order either way and only the survivor changes. It asserts survival now.
- [x] **A spar: measuring whether a character *belongs*.** `check` says a
      character is legal; nothing said whether it stands beside the ones already
      written. `forge.Library.Spar(id, level, seeds)` duels it against **every**
      character in the book, itself included, and reports a rate per opponent.
      Two front-ends, as always: `hexforge spar` and `s` from the TUI's **check**
      screen (raised there rather than from the browser — the two are halves of one
      question, and the browse footer was already 78 of its 79 cells).
      ⚠️ **A spar rate is NOT the roster's win rate and the two must never be
      compared.** The roster figure (49.5%) is five units a side in their authored
      slots with authored loadouts; a spar is 1v1, front column, an auto-chosen
      four-skill kit and no ally to heal or shield. Bulbasaur reads 50.0% in both
      and that is a coincidence.
      ⚠️ **Every pairing is fought BOTH WAYS, and that is the measurement rather
      than thoroughness.** `atb.Queue.order` breaks a tie by enlistment `seq`, so
      of two units with the same speed the one placed first acts first for the
      whole battle — worth **72/28** to Bulbasaur against an identical copy of
      itself. One-way, every rate would be that advantage plus the character with
      no way to tell them apart. `Matchup.First`/`Second` stay apart on the record
      precisely so the **control row** (the character against itself) can report
      what the slot alone was worth: +44.0% Bulbasaur, +23.0% Charmander, +9.0%
      Squirtle — small where duels run long, because a head start washes out.
      The control is **excluded from the headline**: it is even by construction, so
      counting it drags every answer to the middle.
      ⚠️ **A per-row `Failure` was built and then removed as unreachable.** Both
      duellists stand in the front column where `hex.ReachNeeded` is 1, every
      non-self skill must declare a range of at least 1, and a self-aimed skill
      aims at a cell that is always occupied — so `battle.New` cannot refuse one
      pairing and accept another, and what it *can* still refuse is a fault in the
      books that every row would share. A refusal is therefore an error out of
      `Spar`, and `TestTheDuelSlotAsksTheLeastOfAKit` is what says so if the board
      ever changes shape. **An ally-only kit does not trigger it** — an ally target
      reaches its own side's cells, which includes the caster's.
      ⚠️ **`Endless` is not a draw** and is in neither half of a rate's fraction: a
      pair that can never resolve would otherwise read as a pair that reliably
      loses.
      ⚠️ **A spar chooses a loadout where a roster refuses to**, and the two are
      not in conflict: a roster *is* the conditions of a battle so it has nobody to
      state them to; a spar is a measurement, and a measurement states its
      conditions — which is why `Duellist` carries the kit and both front-ends
      print it above the figures. It reads **declaration order**, which gives a
      learnset a meaning it did not have (first declared is first choice); every
      other rule would be `forge` inventing an opinion about what a character is
      for.
      Two things moved to make it possible: `SkillSlots`/`TraitSlots` are now in
      **`internal/core/cast`** (two callers read them, and a second copy of "four"
      is how a measurement stops measuring what gets fielded), and `forge.Library`
      now reads `modifiers.json`, which it never had to before — nothing an author
      writes is checked against the bounds, but a battle needs them.
      **Found immediately:** Squirtle loses to Charmander **30.5/69.5** at the cap,
      with water on the chart against fire. Cast tuning, not a mechanism bug — but
      nothing before this would have said it.
- [x] **Summoning — SHIPPED as an engine, nothing in the cast uses it yet.**
      `skill.Summon` + `battle.summon`: a skill that puts units on the caster's
      side. Two new event kinds, `Summoned` and `Left`.
      ⚠️ **Three stat spellings, exactly one per skill**: `share` (the caster's
      stats **as they stand**), `share_of_base` (ignores timed effects, for when
      a pre-buffed copy is an exploit), `stats` (a fixed line — a toad is its own
      animal). Either share is **frozen at the cast**, like a DoT tick.
      ⚠️ **Three ways off**: killed · `lasts` counts the summon's **own** turns
      (a cooldown counts the caster's, same reason) · `bound` goes with its
      summoner. `bound` is per-skill, not a rule: a clone is an extension, a
      called creature is not.
      ⚠️ **A summon COUNTS in `checkEnd`** — a side holding only a clone has not
      lost. And `dismissBound` runs **before** `checkEnd` in `kill`, or a battle
      is declared running that the next line ends.
      ⚠️ **Goes through `enlist`**, which is every rule about standing here. A
      summon building its own `Unit` is a second answer to all of them.
      ⚠️ **Nothing is in the log** — derived from caster + skill + board + a
      counter on the caster, so the id is built in the engine. An id a caller
      chose is a fact `--verify` would have to carry.
      ⚠️ **A fallen ROSTER unit keeps its slot; a departed SUMMON does not.** The
      formation is what the roster wrote down, and a summon was never in it — it
      borrowed an empty slot. Counting a departed summon kills a repeatable skill
      **silently**: shipped formations leave 2 free slots a side, so the third
      cast of a battle puts nothing down and says nothing. ⚠️ The cell is reusable
      and the **id is not** — `Unit.Summoned` never resets, because an id is what
      a log decision names. ⚠️ **The first version counted every corpse and the
      reason written down for it was WRONG** ("removing a corpse changes what
      everybody can aim at") — `Battle.occupant` already skips the dead, so a
      corpse is not a target and blocks nothing but a place to stand; the
      corpse-sensitive reach check is in `New`, over a roster. ⚠️ **Front column
      first** — `range hex.FormationCols` walks *backward* and drops every summon
      at the far edge where range 1 reaches nobody.
      ⚠️ **A summon may not summon**, checked in a **second pass** over the
      finished book: the summoned skill may be declared below the summoner.
      ⚠️ **`Suggest` never casts one** (power 0 ⇒ fallback only), so both kinds
      are proved by a hand-played battle — `aHandPlayedSummon`, beside
      `aHandPlayedGateCrossing`.
      ⚠️ **`unit.HP = 0` is NOT a death** — nothing reads health looking for a
      corpse, `kill` sets `Dead`. A test that "killed" a copy that way left the
      board with no vacated cell on it and a mutation freeing vacated cells
      survived. Drive a departure the engine performs.
- [x] **The first summoner, and the second origin — Naruto, cast only.** In the
      cast and **not the roster**, so the mechanism gets a real user and
      `replay.golden` does not move.
      New: origin `naruto` (the first non-Pokémon), preset **`summoner`** (column
      1 — it stands a row back and lets the copies spend the turns; no existing
      preset is that), species `human`, six skills, and one character with three
      stages.
      **Two summons, one of each kind**: `shadow_clone` is a share (2 copies at
      400‰, `bound`, `lasts: 4`), `summon_toad` is a fixed line with its own
      element — the case a share cannot write.
      ⚠️ **`Suggest` now casts both** — see `docs/balance.md` § *Pricing a summon*. It did not
      when this shipped (power 0 ⇒ fallback only), so every figure ever measured
      of the summoner was measured with its own mechanism idle.
      ⚠️ **A new preset needs a row in `cast_test.go`'s hardcoded design table**
      and a gloss in `archetypeGloss`, or two tests fail by name.
      ⚠️ **Art is REQUIRED** — `cast.ParseBook` refuses "declares no image". Three
      pictures, one per stage, traced with **`img2svg -q balanced`** (302–420 KB,
      in line with the 21 assets already there). `faithful` came out at 849 KB for
      the busiest of them.
      ⚠️ **A PNG-saved-as-JPEG carries the transparency chequer as PIXELS**, and
      tracing that wraps the character in a grey-and-white background.
      `TestTheShippedArtIsCutOutRatherThanFramed` catches it — it measures the
      corners of the **inked** rectangle, not of the canvas, so any background
      fails and so does a body ending in a straight wide line. Strip it with
      `img2svg --decheck`: erasing by colour holes an eye highlight, a white fur
      collar and a metal headband, and a border flood alone cannot reach a patch
      enclosed between an arm and a coat.
      ⚠️ **An authored summon name is Vietnamese**, so `describeSummon` prints it
      only in `Vi` and says "copy"/"copies" in English — the division `Gloss`
      makes, and a summon has no id to fall back on.
      Its six skills are now `restrict.origins: [naruto]` — see **Origins**
      above; the `summoner` preset keeps them, which is why that ban does not
      exist.
