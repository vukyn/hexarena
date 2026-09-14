# Decisions

Every piece of work that is **finished**, with the reasoning and the measurements
that decided it. This is the *why* behind the code — a merged PR says what
changed, and this says what was learned paying for it.

⚠️ **These entries were in `TODO.md` until 2026-09-05, under headings that had
stopped being true.** `## Not done` held eleven completed items among its eleven
open ones, and a transitional section carried twenty-five more — forty-five per
cent of that file was finished work filed as unfinished. Nothing was deleted;
the headings now mean what they say and `TODO.md` is the open list again.

⚠️ **A second sweep, 2026-09-13, moved thirty more entries here — and the
paragraph above is the reason it was needed.** *"`TODO.md` is the open list
again"* held for eight days. By 2026-09-13 its `## Not done` carried **thirty
finished entries against twelve open ones**, 2,663 lines of them, because
"finished" was being recorded by changing `[ ]` to `[x]` in place. That is not a
lapse in discipline, it is a rule that decays on its own: a heading stays honest
only if closing an item *moves* it, so closing one now moves it. `DOC-001` in
`TODO.md` § *The codes* is this sweep's own entry, and its record is the last one
in this file.

⚠️ **The last thirty entries are in `TODO.md`'s order, not in landing order.**
The closing paragraph below describes the entries *above* them — written in the
order they landed. The thirty from the 2026-09-13 sweep kept the order they sat
in under `## Not done`, which is neither chronological nor sorted, and they are
worth nothing rearranged: the move was verified byte for byte against
`git show HEAD:TODO.md`, and preserving their sequence is part of what made that
check cheap. **They also carry their `AREA-NNN` codes**, in the bullet, right
after the `- [x]`; most of the older entries above predate the code index and
name themselves in prose instead. Search this file by code first, by title
second.

⚠️ **Two entries were DUPLICATED and are merged, not chosen between.**
*Grow the cast* and *Graphical client with ebiten* each existed twice — once in
`TODO.md` and once in `CLAUDE.md` § *Open work* — both current, both maintained,
worded differently. That is the *"two callers wording one choice"* mistake
`CLAUDE.md` § *Mistakes already made here* records, at the level of a document.
Both survivors live in `TODO.md` § *Not done*, since both are still open; each
kept every fact either copy had.

**Read `TODO.md` for what is open, `CLAUDE.md` for what an edit may not break,
`docs/architecture.md`, `docs/balance.md`, `docs/screens.md` and
`docs/goldens.md` for the subject matter, and this
file for why a finished thing is the way it is.** The order below is the order
the entries were written in, which is roughly the order they landed.

- [x] **A stage may declare its own element — `CAST-003`, and the line that
      fields it. DONE, in two parts.** Until now `Character.Element` was
      character-level, so an evolution could be six numbers and a name and nothing
      else. It could not be a **matchup change**, which is the one fact about a
      unit the form could not own while the form already owned its name, its stat
      line, its learnset gate and its picture. That was an inconsistency rather
      than a design, and closing it completes a decision the repository had
      already made twice.

      **The schema is one optional field.** `progression.Stage.Element
      *element.Affinity`, beside `Image`, for `Image`'s own written reason — what
      a form is called, what it looks like and what it is made of are all facts
      about the form, and a parallel map keyed by stage name would be a second
      thing to keep in step, stale exactly when a stage is renamed. **A pointer,
      and that is load-bearing**: the zero `Affinity` is a *valid* single-neutral
      affinity, so a value field would have read "this form is neutral" on every
      stage that declared nothing — a silent wrong answer rather than an absence,
      the same trap `Values.UnmarshalJSON` guards with `*int64`. `progression`
      does not validate it, because it has no chart, which is the layer split
      `Stage.Image` already records; `cast.resolveCharacter` does, per stage,
      beside the image loop. **The fallback has one home**, `Character.ElementAt`,
      mirroring `StageArt`, and no caller reads `Stage.Element` directly.

      **The carry check went per form, with "all" semantics.** A book only *some*
      form could carry is a book `battle.enlist` refuses at the moment somebody
      plays it — the exact gap between the authoring layer and the engine that
      `CLAUDE.md` § *One rule, one declaration* records as already fixed once.
      `Unlock.Stages` is what makes a gated metal skill invisible to a
      ground root, and it already shipped, so the parser gained a quantifier and
      nothing else. ⚠️ **Reachability alone would sometimes have sufficed and that
      is a trap**: an ungated `metal_claw @40` on a line evolving at 32 is only
      ever held by the grown form, so a purely reachability-based rule accepts it
      — until somebody moves the evolution level and it silently becomes illegal.
      The parser accepts both and **the shipped data writes the explicit gate**.

      **It is a fielding-time property, decisively.** `internal/core/battle`
      never imports `internal/core/cast`, `battle.Roster` carries an already
      resolved `Affinity`, and no unit evolves mid-fight — so there are **zero**
      changes under `battle`, no replay can break and `replay.golden` cannot move.
      Three sites produce an affinity from a character (`Placement.resolve`,
      `seed.resolveReference`, `forge.Library.duellist`) and all three already
      held the resolved stage on the line above. `duellist` is the single funnel
      for spar, weigh and carriers, so the whole measurement stack inherited the
      fix from one line.

      **What it buys, in one measurement nothing here could take before.**
      `TestASecondElementIsWorthWhatTheChartSaysItIs` holds a body still and
      varies only the affinity its two forms resolve to: against grass the chart
      says `ground` takes 1500 and `ground/metal` 1000, against ice 1500 and 2250,
      **so the sign has to flip**. Measured over sixty duels an arm: 211 → 143 a
      blow against Bulbasaur (a ratio of 0.68 where the chart says 0.67) and
      170 → 255 against Lapras (1.50 where the chart says 1.50). ⚠️ **A win rate
      would have measured nothing there**: grass and ice are this character's two
      counters and it reads 0 of 80 against both, so a rate would be nought
      against nought in both arms while the mechanism worked perfectly.

      ⚠️ **One trap the golden is the only guard against.** Neither
      `cast.ParseBook` nor `progression` uses `DisallowUnknownFields`, so a
      mistyped `"elemnt"` on a stage is silently ignored and reads as "the
      character's" — and the nil-pointer guard cannot fire, because nil is legal.
      `castReport` therefore prints the **resolved** element on every stage line,
      which is what makes such a typo a diff in `cast.golden`. Do not remove that
      half of the change.

      **The mechanism ships with one user and one queued user.** `pokemon.onix`
      fields it; `magikarp → gyarados` is traced and is next (water →
      water/**wind**). `charmander → charizard` would want it and was deliberately
      **not** taken: it re-prices a shipped character and moves every Charmander
      golden. That is honest rather than ideal — it does not clear "two independent
      users on day one" — but it does clear the bar this repository actually
      keeps, that *a mechanism no shipped placement fields is a mechanism nothing
      measures*.

- [x] **The second wall — `pokemon.onix` and the `monolith` preset, `CAST-002`.
      DONE.** Twenty-third character, twenty-third preset, first `mineral`, and
      the first shipped stage to sit a stat **on** a ceiling rather than under
      one: Steelix's defence is 800, which is `progression.json`'s own ceiling,
      and 3100 health behind it absorbs 11,397 of the 11,500 joint budget —
      **103 to spare**, where the previous tightest line, Blastoise, had 215. What
      pays for it is the lowest speed (60) and the lowest dodge (20) in the cast
      and a health line under the warden's.

      **Attack is the one number that was tuned, and it was tuned against a
      measurement.** At the 500 the design started from, `hexforge spar --stage
      Steelix --seeds 40` read **198‰** — under the 250–450 target band, third
      weakest in the cast. Accuracy turned out not to be the lever (130 → 160 → 180
      moved the overall by 0.4 points at a fixed attack, because a hit chance
      saturates), so attack went 500 → 600, the lowest round value clearing the
      band: **264‰ overall, mirror exactly 500‰, 0 endless anywhere, longest row
      63 turns.** The root form at the cap is the control rather than a balance
      reading and comes back at **1‰** — 2 wins of 1920 — which is what giving up
      an evolution costs on this line.

      ⚠️ **The slot measurement swaps the WALL, not the flex member, and the
      difference between the two shells is a reading and a category error.**
      `aSquadOf` holds a striker and a wall and varies the third slot, which is
      right for a mender and wrong for a second wall: it makes the home squad two
      walls and a striker against two strikers and a wall, so what comes back
      prices the composition. Measured in that slot the wall build reads **381‰
      against a slugger and 360‰ against a bruiser**, both under the mender's
      floor, with the character behaving exactly as designed. In the slot it is
      actually chosen for — striker and flex held, wall varied — it reads **491‰
      against the warden's own squad**, with the shell's control (the warden
      against a copy of itself) coming to exactly 500‰.

      **Two builds, and the second is the first entry in the catalogue that a
      FORM rather than a character can hold.** `onix.wall` (`taunt`,
      `wide_guard`, `stone_edge`, `dig` + `thorns`) and `onix.forge` (`steel_beam`,
      `flash_cannon`, `metal_sound`, `metal_claw` + `ballast`) — every skill in
      the second is metal and gated on the grown stage, so the root form could not
      carry one of them.

      ⚠️ **The preset is `monolith` because `bulwark` is not available**, and the
      reason is worth writing down once: `internal/testfixture` **appends** its
      five presets to the shipped `archetypes.json` in every scratch data
      directory, so `bulwark`, `vanguard`, `sentinel`, `duelist` and `skirmisher`
      are reserved names that a shipped preset may not take. Taking one fails
      about forty tests across three packages with *"archetype "bulwark" is
      declared twice"*, and all five are **already glossed**, so finding the id in
      `archetypeGloss` is a false signal that the gloss step is finished.

      ⚠️ **One finding logged rather than fixed: `unyielding` cannot fire on
      anything the game ships.** It resists `taunting` at 1000, and `taunting` has
      exactly one source in the whole book — `taunt`, which applies it to **its
      own caster**. So the trait can only ever refuse its holder's own taunt, and
      on a unit not carrying `taunt` it is inert. `machop.sure` ships with it
      today. It is left in this character's learnset as a choice and it is not
      what either build spends its trait slot on.

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
      not.** `docs/screens.md` § the TUI width rule splits prose, which takes
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
      `docs/screens.md` § the description screen.

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
      `testdata/screens.golden`, `docs/screens.md` § the squad builder and § the
      description screen; and for (a), `cmd/hexforge-tui/language_test.go` and
      `cmd/hexarena-tui/{sweep,fixture}_test.go` (`kitGlosses`); and for (a′), the
      same three files (`freeText`/`freeNames`/`withoutNames`/`kitIDs`, and the
      width, language and gloss-leak sweeps in each); and for (c),
      `internal/screen/squads.go` (`formLabel`, `unnamedArms`, `characterOf`),
      `internal/i18n` (`SquadForkUnnamed`, `SquadForkArms`),
      `internal/screen/squadfork_test.go`, `internal/screen/screens_golden_test.go`,
      `cmd/hexforge-tui/{language,width_rule}_test.go`, two of the three
      `testdata/screens.golden`, and `docs/screens.md` § the squad builder.
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
      → **The same root has its own entry, `DAT-003` below**, "a `hexforge new` still churns
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
      ⚠️ **The third and fourth of those were REVERSED later**, by the owner: a
      reply now fires **once per damaging strike**, inside the strike loop, so
      reply damage multiplies by the strikes that connected — and a reply that
      kills the caster stops the volley and the rest of the shape with it. The
      two that survive are unchanged. The rest of this entry is the record of
      what was decided at the time and is left standing; `README.md` § *the four
      rules* carries the rule as it is now.
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

- [x] `SCR-015` ⚠️ **The local battle chose the opponent for you, and the menu
      described the wrong feature.** Raised and closed 2026-09-10, out of a
      question about which squad the menu takes — `#412` and this entry's own
      wording change.

      **What shipped.** The PvE entry opens a chooser: one cursor for the side
      you command, one for the side across the board, `enter` plays it. It used
      to open a battle on whichever row the reader last pointed at, against **the
      next row wrapping** — `pairing.go` called that a local stand-in for
      matchmaking and said so deliberately, which is why it lasted. The chooser
      is client-local rather than shared with `cmd/hexforge-tui`'s `fightScreen`,
      because that one *measures* a pairing — seed ladder, rate, by-side split,
      a squad against itself as the control — while this one settles who turns
      up, and `pairing.go` already records that the two clients owe `Open`
      different pairings. What is copied from it is one structural decision: the
      cursors hold **indices into the catalogue, not ids**.

      ⚠️ **The away cursor opens on row 1, not row 0**, and that is the whole
      reason the golden diff is a pure insertion. Nought is the obvious start and
      would have made every unchosen pairing a side against a copy of itself,
      moving all eight battle entries. Row 1 is exactly what "the next row,
      wrapping" already gave a reader sitting on the first row.

      ⚠️ **The mode was misread twice while planning this, in opposite
      directions, and both readings reached the owner before they were checked.**
      First it was called PvE with no evidence; then, on finding `pairing.go`'s
      phrase *"the hot-seat battle"* and `play.go`'s `a` key — *"the engine's own
      answer, taken as the player's"* — it was called hot-seat, meaning the
      reader takes both sides. **It is PvE**, and `run()` settles it in four
      lines: a turn whose `unit.Side == p.Side` stops and asks, and every other
      turn goes to `engineOrder`. `a` is an assist on *your own* turn, and
      "hot-seat" in that comment means **local rather than networked**. The tell
      that should have been followed the first time is that only `run()` sets
      `Pending` on the local path — lines 385–402 are the live PvP path and read
      `live.Asking`.

      ⚠️ **A second claim made and then measured false**: that the six shipped
      sides were never offered to this client, on the strength of `.Squads()`
      appearing nowhere under `cmd/hexarena-tui`. The call is
      `draw.SquadsScreen.Refresh` → `forge.SquadsOffered(c.Lib.Squads(),
      c.Player)`, in the shared screen the client draws. Measured on a real
      machine: `shipped=6 player=1 offered=7`. A planned step to "add the shipped
      sides" was therefore dropped as a null before anybody built it. ⚠️ The whole
      suite is blind to this because `scratchDataFrom` **deletes `squads.json`**
      from every scratch data directory, so no fixture anywhere has a shipped
      side on it — worth its own item if a shipped side ever needs asserting.

      ⚠️ **`pairing.go` carried a false sentence** and it is fixed in place rather
      than contradicted beside: it claimed `m.taking` "is now also what fills
      `wire.Hello.Squad` when a room is joined". It does not — `model.dialling`
      reads `joinScreen.Chosen()`, which exists because a drafting room needs a
      "bring none" position no catalogue row can express.
      `TestThePairingChooserDoesNotReachThePvPSquad` now holds that by behaviour
      **and** by source.

      **The wording.** `GameMenuBattleDetail` said *"play a battle yourself, with
      the first side on the list"*, which after the chooser was false twice over
      — the entry opens a chooser, and "the first side" describes a pairing whose
      both halves are choices. Neither it nor the chooser's own hint said the
      opponent is the machine, which is the fact a reader most needs and the one
      that started this. Both now say it, in both languages. Twelve golden lines
      move and every one of them is one of those two sentences.

      **Related, and already true before any of this**: `s06` is five units while
      the other shipped sides are three, so an asymmetric pairing is reachable
      from the chooser. It builds — `s06.Take` answers five rosters and a
      three-unit side answers three, both without error. Whether it is *fair* is
      unmeasured and is not claimed here.

- [x] `FRG-004` ⚠️ **`hexforge` could not run from a clean `go install` either,
      and most of it only reads.** Raised and closed 2026-09-10, out of the last
      paragraph of `SCR-014`, which had written the whole tool off as a writer.

      **What was wrong.** All fourteen subcommands opened on
      `forge.Load(forge.DefaultDataDir)`, and that default is
      `"internal/seed/data"` — relative to the working directory, so it exists
      only inside a checkout. Installed from the proxy and run anywhere else,
      every one of them died on the same line the game client used to:

      ```
      hexforge: read internal/seed/data/combat.json: open internal/seed/data/combat.json: no such file or directory
      ```

      **The decision is per CODE PATH, not per subcommand**, which is the one
      thing a table keyed on the fourteen names would have got wrong. `origins`
      lists the catalogue and `origins add` appends to it, under one name,
      dispatched by a single `args[0] == "add"` inside `runOrigins`; `species`
      and `skills` are the same shape. Eleven code paths reach a data directory:

      | path | reads or writes | how it gets there |
      |---|---|---|
      | `loadForListing` | reads | the shared helper behind `origins` `species` `statuses` `archetypes` `passives` `skills` `cast` `builds` — **8** subcommands, one function |
      | `runShow` | reads | resolving one character and pricing it |
      | `runCheck` | reads | ⚠️ through `forge.Inspect`, **not** `forge.Load` |
      | `runSpar` | reads | duels fought in memory |
      | `runCensus` | reads | the same |
      | `runWeigh` | reads | the same, both the single and the `--carriers all` form |
      | `runOriginsAdd` | writes | `origins.json` |
      | `runSpeciesAdd` | writes | `species.json` |
      | `runSkillsAdd` | writes | `skills.json` |
      | `runSkillsEdit` | writes | `skills.json` |
      | `runNew` | writes | `cast.json` |

      ⚠️ **`runCheck` is the row a `grep` for `forge.Load` cannot see**, and the
      survey this item was raised from listed ten sites rather than eleven for
      exactly that reason. `forge.Inspect(dir)` is `Load` and then `Inspect`, and
      it reaches a directory just as hard while being spelled nothing like it.
      `check` is now the two halves written out, so the reading rule sits between
      them. The walk below therefore matches on **registering `--data`** rather
      than on calling into `forge`: a subcommand that takes the flag has declared
      it touches a data directory whatever it then does with the string, and one
      that reached a directory *without* taking the flag would be a worse defect
      than the one this item is about.

      **The rule is shared with the game client rather than spelled twice.**
      `SCR-014`'s three lines moved into `forge.LoadForReading(dir, dataGiven)`,
      and `cmd/hexarena-tui`'s `loadLibrary` is now one line calling it. Two
      commands cannot import each other, and this is the tool whose own doc
      comment says neither authoring front-end may restate a rule the other has,
      **the wording of a refusal included**. The three ⚠️ notes on it — the
      fallback keys on the directory being ABSENT and never on the load failing;
      absent means `fs.ErrNotExist` and not "the stat failed"; a named directory
      is never second-guessed — travel with the function, so a second front-end
      cannot get two of the three. `forge.DataDirectoryIsAbsent` is exported
      beside it so a front-end can phrase its own refusal **before** attempting
      the load.

      **Writers refuse, and the refusal names the way out.** They cannot fall
      back for the reason `go:embed` is read-only, and the refusal happens on the
      absence probe rather than at the write: `forge.ErrNoDataDirectory` would
      otherwise arrive after eleven prompts, which is a worse program than one
      that says so first. As printed:

      ```
      hexforge: origins add writes to a data directory and there is none at "internal/seed/data": the copy of the books embedded in this binary cannot be written to, so pass --data <dir> naming a directory that exists, or run hexforge from the root of a hexarena checkout, where internal/seed/data is the game's own
      hexforge: show reads a data directory and there is none at "/nope": pass --data <dir> naming one that exists, or leave --data off to read the copy of the books embedded in this binary
      ```

      The second is the other half `SCR-014` left bare: an explicitly named
      `--data` that is not there is still never replaced, but `forge.Load` could
      only answer with the read error on whichever book it reached first, naming
      a file nobody typed.

      ⚠️ **`check` now makes a narrower claim and says so.** A library with no
      directory asks for no art at all — `forge.artToCheck` already returns
      nothing, which is what stops sixty-six false problems, and that function's
      comment used to say nothing inspected such a library. Something does now. So
      a clean `hexforge check` off the embedded books is *not* the same clean it
      is over a directory, and the closing note says the art was not looked for
      and how to look for it.

      ⚠️ **A `~/.hexforge` home default was asked for and REFUSED, and the refusal
      is a measurement rather than a taste.** Nothing in the game ever plays from
      the `--data` directory. ⚠️ **Re-measured, and the figures this item was
      raised with were two out.** `cmd/hexarena-tui/model.go:781` — not 774 —
      builds its mirror from `seed.Books()` whatever `--data` says, and
      `cmd/hexarena` calls `seed.Books()` at **four** production sites, not five
      (`main.go:112`, `:181`, `:593`, `:619`; the fifth in the survey was
      `cmd/hexarena-host/main.go:455`, a different binary). The direction is
      unchanged and in fact stronger: `cmd/hexarena` and `cmd/hexarena-host`
      declare **no `--data` flag at all** — `hexarena-tui` is the only one of the
      three game binaries that has one, and it uses it for the catalogues and
      never for a battle. The game's data is
      `internal/seed/data`, embedded and tracked in git. A home default would send
      an author to edit a copy that reaches **neither a battle nor the
      repository** — which is strictly worse than the error it replaces, because
      it looks like it worked. The one thing that legitimately lives under
      `os.UserConfigDir` is `forge.PlayerSquadsPath`, and that is the *player's
      own squads*, which the game really does read. Do not add one; this
      paragraph exists because the question will be asked again.

      **The guard.** `TestEveryHexforgeCodePathThatReachesTheDataDirectorySaysWhetherItReadsOrWrites`
      is `internal/forge`'s own idiom — the walk over
      `Library.home` — pointed at this package: it parses the twelve non-test
      sources, collects every function registering `--data`, and holds that set
      **equal** both ways to the decision table. It also reads which of
      `loadForReading` / `loadForWriting` each one calls, so a row that *claims*
      to read while the source writes is red, and it forbids anything but those
      two helpers from calling `forge.Load` / `LoadEmbedded` / `LoadForReading` /
      `Inspect`. A second bijection covers `loadForListing`'s eight subcommand
      names, since one function serving eight names would otherwise let a ninth
      arrive unrun. It logs **12 source files, 11 code paths, 6 reading and 5
      writing, 8 listing subcommands**, and fails on none — a walk whose predicate
      has stopped matching agrees with every claim there is. Measured to bite:
      a fifteenth subcommand with a flag set and no row fails naming it; declaring
      a writer and calling `loadForReading` fails twice, once on the source and
      once on the run, where the writer really does get handed the embedded books
      and renders a skill it can never save; a row for a function that no longer
      exists fails as stale.

      Beside it, `TestEveryListingSubcommandListsFromACleanInstall` runs all eight
      names for real, and
      `TestANamedDataDirectoryThatIsBrokenIsRefusedRatherThanQuietlyReplaced`
      holds the trap the reading rule is really about — the two-line "load, and
      take the embedded copy if that errored" version passes every clean-install
      test here and quietly plays the shipped books over an author's trailing
      comma. Its second arm builds a checkout-shaped working directory, so the
      **unnamed and present** line of the rule is measured and not only the named
      one. ⚠️ Its assertion matches `decode combat rules` rather than
      `combat.json`: a *read* error is wrapped with the path, a *decode* error is
      not, so matching the file name would pass only while the load was failing
      for the wrong reason.

      **What is deliberately not done.** `cmd/hexforge-tui` is untouched. It is an
      interactive editor whose every screen is one keystroke from a write through
      `form.go`, so opening it browse-only would mean gating those keys on
      `HasDataDirectory` at each screen, and `screen.Context.Authoring` is already
      the flag that decides whether a screen offers authoring at all — the honest
      shape is a third state (authoring / browsing / read-only) rather than a
      fallback, and it moves both clients' goldens. Raised as a reading, not
      built. No golden moved for this item.

- [x] `SCR-014` ⚠️ **The game client could not run from a clean `go install`, and
      then accused the player of a defect the binary caused.** Raised 2026-09-09
      from a real report — `hexforge-tui` first, and the client turned out to have
      the same fault — and closed 2026-09-10 in three steps: `#404`, `#405`,
      `#406`.

      **What was wrong.** `cmd/hexarena-tui` opened with
      `forge.Load(forge.DefaultDataDir)`, and `DefaultDataDir` is
      `"internal/seed/data"` — a path **relative to the working directory**, which
      exists only inside a checkout. Installed from the proxy and run anywhere
      else it died:

      ```
      hexarena-tui: read internal/seed/data/combat.json: open internal/seed/data/combat.json: no such file or directory
      ```

      ⚠️ **The battle never needed that directory**, which is what made this a
      startup bug rather than a design one: `model.go` already builds the mirror
      from the embedded books, deliberately, because the digest at the room's gate
      is over the embedded files and a client fighting on an edited directory
      would pass a promise it then breaks. Only the load stood in the way.

      **Why the obvious fix was refused.** "Create the data directory if it is not
      there" was the first request, and it cannot work: `go:embed` names **16 JSON
      files**, and the art — **16 MB across 66 files** under
      `internal/seed/data/assets` — is not embedded and is not going to be. A
      directory the program wrote for itself would hold the books and no pictures,
      and the program's own checker calls that broken: `hexforge check` over a
      JSON-only directory reports `MISSING 3/3` for all 25 characters and exits 1.
      So the answer is to read the embedded copy and own no directory at all.

      **Step 1 (`#404`) — `forge` learns the second way in.** `Load` splits into
      `Load(dir)` and `LoadEmbedded()` over one `loadBooks`; `internal/seed` grows
      `Data() (fs.FS, error)`. ⚠️ The real work was what a library with **no**
      directory does with the sixteen functions that reached `dir`, because
      leaving the field empty is not neutral: `filepath.Join("", "cast.json")` is
      `"cast.json"`, a relative path in whatever directory the player is standing
      in. `dir string` therefore became `home dataHome`, whose `join` is the only
      expression in the package that builds a path under the data directory —
      `filepath.Join(l.home, name)` **does not compile**, so a seventeenth
      consumer cannot write the join out by hand. Anything that can refuse does,
      with `ErrNoDataDirectory`; an accessor whose signature is a bare string
      answers `""`.

      **Step 2 (`#405`) — the caller switch**, three lines of rule: `--data` given
      → `Load`, never a fallback; not given and the directory absent →
      `LoadEmbedded`; not given and present → `Load`, which is what keeps
      `make play-tui` showing an author their own edits. ⚠️ **The fallback keys on
      the directory being ABSENT, never on the load failing** — "load, and use the
      embedded copy if that errored" would swallow an author's trailing comma and
      play the shipped data in silence. Same distinction `SCR-013` rests on: probe,
      do not read the result.

      **Step 3 (`#406`) — the sentence.** With two states where there was one, the
      program said the same thing about both. `MISSING` is right for an author and
      **false** for somebody who installed the binary: nothing is missing, their
      setup is not broken, and the error styling accuses them of a defect they did
      not cause and cannot fix. The states now read `THIẾU` / `MISSING` in the bad
      style against `không kèm theo` / `not shipped` dimmed, and in the preview,
      *"chương trình không kèm ảnh — ảnh nằm cùng mã nguồn chứ không nhúng vào file
      chạy"*. The screen asks `forge.Library.HasDataDirectory()` rather than
      comparing `Dir()` to `""`: `Dir()` is a display string, empty because there
      was no path to show, and a comparison written in `internal/screen` would sit
      outside every guard `internal/forge` has.

      ⚠️ **A fourth site the plan did not name.** `internal/screen/browse.go` was
      written off as the authoring tool's browser. It is **one screen both clients
      draw**, so a player met the bad-styled `MISSING` on the cast detail pane one
      keystroke *before* the preview. Grepping the key rather than reading the
      screen list is what found it — `grep -rn "i18n.ArtMissing"` gives exactly two
      production sites.

      ⚠️ **Two guards were missing and the reviews found them, not the tests.**
      First, step 2's probe narrows to `fs.ErrNotExist` rather than any stat
      failure, because a path blocked by a file gives ENOTDIR — measured, and
      `errors.Is(err, fs.ErrNotExist)` is false for it. That narrowing arrived with
      a ⚠️ comment stating it and **nothing enforcing it**: widening it to
      `err != nil` compiles and leaves the whole suite green.
      `TestADataDirectoryPathBlockedByAFileIsRefusedRatherThanQuietlyReplaced` is
      what closes it. Second, step 3 recorded only the **new** half of the new
      distinction in the goldens. Measured at `9db0119`, `MISSING` / `THIẾU`
      appeared **zero** times across the **516** recorded renders, because every
      fixture has both a directory and its art — so the older half, the one a
      regression silently deletes, was held by nothing. Both halves now sit
      adjacent in `screens.golden`, differing in exactly one line, so a diff over
      them is a diff about the sentence; the count is now 524 renders and 4
      sightings. ⚠️ The figure first written here was "~460", carried out of a
      report rather than counted — `grep -c '^===== '` over both golden files is
      where the number comes from.

      **What is deliberately not done.** The cast row gets no golden entry: one
      there would add layout the `browse` entry already records, while which
      verdict appears is pinned harder by a table that refuses each arm the other
      two words. The condition for revisiting it is written on the entry map — if
      `artLine` ever grows a column the verdict sits in, those entries should
      arrive. And `hexforge` / `hexforge-tui` still require a real directory,
      correctly: they **write** files, and `go:embed` is read-only. ⚠️ Their error
      message is still the bare relative path with no hint about `--data` or the
      module root; that is a separate item — raised and closed as `FRG-004`
      above, which also corrects the sentence before it: `hexforge` requires a
      real directory only where it **writes**, and most of it does not, and
      `hexforge-tui` is untouched and still requires one everywhere.

- [x] `ENG-014` ⚠️ **Nothing runs the test suite on a pull request.** A
      stale golden merged to `main` and sat there red for one commit, and the
      reason is not that the break was undetectable — `make check` names it
      exactly, in the words the golden test was written to say. Nobody ran it.
      Raised 2026-09-09 out of `SCR-013`'s merge, and the first shape it was
      raised in is refused below by its own measurement.

      **What was measured.** `#395` (`0c7ef64`, `DAT-011`) added
      `happiny.tend` and `happiny.decoy` to `builds.json` without re-accepting
      `internal/screen/testdata/screens.golden` or
      `cmd/hexforge-tui/testdata/screens.golden`, and both golden tests go red on
      a pristine `git archive` of it:

      ```
      line 2829, under ===== vi | 160x60 | builds =====:
        golden "  pokemon.happiny       chưa viết bản dựng nào cho nhân vật này"
        drawn  "  pokemon.happiny"
      ```

      It went green again at `15b9653` (`#396`), which re-accepted the goldens as
      a side effect of its own screen change — the break was **repaired by
      accident**, not found. There is no `.github/workflows/` in the repository,
      and `gh pr view --json statusCheckRollup` returns **one** check on a PR:
      `GitGuardian Security Checks`. A secret scanner is the whole gate.

      ⚠️ **This was first raised as "a fourth occurrence of one pattern", and
      that claim is FALSE — the count is one.** It came from conflating *main was
      red* with *a golden went stale*, across four events that only share the
      first half. Measured, one at a time: `#307` changed `squads.json` and its
      goldens are **green** at `49e3581` (its redness was a digest); `#319`
      changed **three** golden files and no data at all; `#326` changed one golden
      file and no data. Only `#395` is the shape. Across the whole history, 88 of
      129 commits touching `internal/seed/data` re-accepted no golden — and that
      number is **not** evidence of 88 breaks, because the goldens cover only some
      screens and most data never reaches them. It is the number that looks like
      evidence and is not.

      ⚠️ **The obvious fix is refused, and the reason is that it is vacuous.**
      The first shape proposed here was a guard asserting that a golden derived
      from `internal/seed/data` is younger than the last change to that data. It
      would fire only where `go test` runs — and where `go test` runs, the golden
      test **already** fires, with a better message that names the differing line.
      A second red line in the one place that is already red detects nothing new.
      Everything about this item is on the other side of the test run, so the work
      is to make the run happen, not to add an assertion to it.

      **What the work is.** A workflow on `pull_request` that runs the same
      `make check` the contract already rests on, and a branch rule that requires
      it. Two things to weigh before writing it, both measured this session and
      neither settled: `make check` is **expensive** — the package times of one
      full green run sum to **1229s, 20.5 minutes**, on 8 cores, with
      `internal/seed` at 346s and `internal/forge` at 257s between them carrying
      half of it. ⚠️ That sum is **not** the wall-clock — `go test` runs packages
      in parallel. **The wall-clock is 390s, 6.5 minutes**, measured on the
      `v0.2.0` release run at `e442ec3` by stamping `date +%s` either side of
      `make check`; the sum overstates it by 3.2×, which is the number to ignore
      when sizing a runner. Six and a half minutes is affordable per PR, so the
      fast-gate split is **not** needed for cost — decide it on feedback latency
      if at all. And `internal/room`'s loopback
      test has hit its 60s bound three times under parallel load while measuring
      ~0.4s in isolation — a **160×** margin — so a shared CI runner is exactly the machine that will make it
      flake. A gate that flakes is a gate people learn to re-run, which is a gate
      that is off.

      **DONE — a workflow on `pull_request` and on pushes to `main`.** It runs
      `make check` rather than spelling the steps out again: the Makefile is
      where the gate is defined, and a workflow listing `go vet` and four
      `-race` packages is a second copy that drifts the first time somebody adds
      a package to the gate. The Go version comes off `go.mod` for the same
      reason, and `#409` is the proof it was needed — the directive moved to
      1.27.1 between one branch's check and its merge.

      ⚠️ **It runs on `main` too, and that is the case this was raised for.** A
      pull request green against an older `main` can still break `main` when it
      lands, which is precisely what `#395` did. A gate guarding only the door a
      change came through does not notice the room changing behind it.

      **The runner's wall-clock, measured on the run this workflow's own pull
      request triggered: 731s, 12m11s** — against 357s on eight local cores, so
      **2.05×**. That is the figure this entry said was unmeasured. Twelve
      minutes is affordable per pull request, so the fast-gate split floated
      above is still not needed for cost. ⚠️ `internal/room`'s loopback test did
      **not** flake on that run, which is one sample and not a clearance — the
      60s bound has been hit three times under parallel load locally, and a
      hosted runner has fewer cores than the machine that did it.

      ⚠️ **The first line of the gate did not gate, and had not since it was
      written.** `make check` opened with `gofmt -l .`, and `gofmt -l` LISTS and
      exits 0 — measured: an unformatted file in the tree prints its name and the
      target passes. The formatting half of this repository's gate has never
      failed anything. It is now a shell that exits 1 on a non-empty list, which
      is what the bare form only looked like it did. ⚠️ Read that twice before
      trusting any other one-line step: a command whose *output* is the finding
      is not a command whose *status* is the finding, and `make` reads only the
      status.

- [x] `DAT-013` **`cleffa.hex` lost four battles in five because of the KIT, not
      because the rating underprices control — MEASURED, and the build was
      rekitted rather than the rating repriced. The mender's fixture now fields a
      catalogued build that clears its floor. Half the item closes; the other half
      is a fixture defect and is handed on as `DAT-014`.** Raised 2026-09-09 out
      of `DAT-011`'s sweep and closed the same day.

      **What shipped.** `cleffa.hex` gives up `smokescreen` for `moonblast` —
      `charm, sing, moonblast, solar_beam` + `elusive` — and its `intent` is
      rewritten as a **lean** rather than a purity, which is what
      `bulbasaur.parasite` and `squirtle.fortress` already are. Cleffa's learnset
      holds exactly two damaging skills, so every possible repair costs one of
      `charm`/`sing`/`smokescreen`; that was an authoring decision and it was
      taken. `TestAMenderEarnsItsSlotWhereASparCannotSeeIt` now fields
      `cleffa.hex` through `aThirdMemberFrom` — the fixture's hand-written hybrid
      is gone — and gained the mirror control `onix_test.go` already carried.

      **The question the item said had to be settled first: the build, or the
      pricing? THE KIT, and the census is the evidence.** Every control skill in
      the kit has a live, non-zero pricing term reading a real modifier: `charm`
      → `weaken` → `standingLost` over a 3-turn horizon, `sing` → `stun` →
      `p.strike(target) × turnsOf(1,3)`, `smokescreen` → `blind` → `standingLost`
      over 2, and `blind`'s accuracy term is genuinely priced because
      `bestStrike` → `expected` → `against` weights by `Rules.Chance(hit)`. None
      falls through the *worth nothing means not rated* arm where `taunt` and
      `heal_cut` sit. ⚠️ **And the census was re-taken on the FIXTURE board**, not
      on the shipped-squad board `forge.Census` walks, because RAT-006's *the
      board is the finding inside the finding* applies here. Over 600 battles a
      column, `cleffa.hex` in the mender fixture's third seat:

      | opponent | rate | endless | `charm` | `sing` | `smokescreen` | `solar_beam` | control share |
      | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
      | a slugger | 356‰ | 0/600 | 6857 | 5493 | 4759 | 5411 | **76.0%** |
      | a bruiser | 226‰ | 0/600 | 7640 | 6446 | 5889 | 6624 | **75.1%** |
      | a blighter | 281‰ | 0/600 | 3713 | 2893 | 2398 | 2915 | **75.5%** |

      **Not one silent slot on any board, and `charm` is the most-cast skill on
      all three.** An under-price shows up as SILENCE; this is the opposite of
      silence. Under the item's own stated rule — *if it casts them and still
      loses four in five, the answer is the kit* — the verdict is the kit, and a
      repricing is **refused by measurement rather than deferred**. That matters
      because the blast radii are not comparable: a rekit moves one catalogue
      entry, while `price.go`'s `inflictedOn` Control/StatDebuff arms are on the
      path of every status in the book and would move every balance band in
      `internal/seed`, `replay.golden`, `scenarios.golden`, `RAT-002`'s 81.3%
      figure, and `TestABothWaysMirrorIsExactlyEven` — the fairness invariant
      that has already refused two changes of this shape.

      ⚠️ **The item mis-cited `RAT-002` and the premise did not survive reading
      its own citation.** `RAT-002` is about `forge.Bout` fighting two ratings
      head to head and `Suggest` beating `FirstUsable` 81.3%; it records
      **nothing** about control being under-priced. The nearest true statement is
      the general one at `price.go:12-34` — every non-damage horizon is capped
      rather than honest, erring deliberately under — which binds buffs, guards
      and heals equally and is not a control finding. Corrected here and in
      `mender_test.go`. (It had propagated to **one** place, not the three that
      were suspected: `cleanser_test.go` carried the stale figure *table*, not
      the citation, and its table is corrected too.)

      **The sweep.** Mender fixture shell, `aSquadOf`/`fightSquadsOver`, 300 seeds
      a side both ways round = 600 battles a cell, `books.Bonuses = nil` on every
      row (the fixture's own comment measures 413‰ live against 698‰ without,
      which straddles the floor — pricing a bonus is `DAT-010`'s owed
      measurement). **Every row below read 0 endless of 600**; nothing was
      saturated at 0‰ or 1000‰, so nothing was dropped. One slot given up for
      `moonblast`, the only other damaging skill in the learnset:

      | kit | slugger | bruiser | blighter |
      | --- | ---: | ---: | ---: |
      | `charm sing smokescreen solar_beam` (was shipped) | 356‰ | 226‰ | 281‰ |
      | `moonblast sing smokescreen solar_beam` | 616‰ | 610‰ | 373‰ |
      | `charm moonblast smokescreen solar_beam` | 696‰ | 563‰ | 400‰ |
      | **`charm sing moonblast solar_beam`** | **826‰** | **725‰** | **546‰** |
      | `charm sing smokescreen moonblast` | 251‰ | 171‰ | 260‰ |

      Giving up `solar_beam` is the one row that makes the build *worse* — it is
      the only thing in the kit that reaches a back line through two held ranks,
      and `moonblast` at range 2 does not replace it. Then the non-damaging
      alternatives in the slot that read worst, `smokescreen`'s:

      | in smokescreen's slot | slugger | bruiser | blighter |
      | --- | ---: | ---: | ---: |
      | `taunt` | 606‰ | 395‰ | 606‰ (1 endless) |
      | `wide_guard` | 730‰ | 616‰ | 278‰ |
      | `light_screen` | 666‰ | 780‰ | 386‰ |
      | `rapid_spin` | 341‰ | 220‰ | 300‰ |

      **Not one of them clears 400 against all three**, so `moonblast` is the
      optimum rather than merely a candidate. ⚠️ `rapid_spin` is the one genuinely
      near-silent slot found anywhere in this item — **93 casts of 17591** against
      a slugger, 0.5% — which is what an under-priced slot actually looks like,
      and is the contrast that makes the control census above legible.
      ⚠️ `moonlight`, `wish` and `withdraw` were excluded from every candidate:
      all three give health back, and `TestTheTwoCleffaBuildsAreDifferentUnits`
      asserts `hexed.healed == 0` as the design record of what the direction is.

      **The trait, swept on the winning kit over Cleffa's five-trait cap
      learnset** (the fixture hardcodes `endurance` for hand-written members, so a
      fixture that fixes the slot cannot report it). All rows 0 endless:

      | trait | slugger | bruiser | blighter |
      | --- | ---: | ---: | ---: |
      | `endurance` | 733‰ | 638‰ | 440‰ |
      | `swiftness` | 865‰ | 808‰ | 590‰ |
      | `last_gasp` | 860‰ | 725‰ | 641‰ |
      | `composure` | 825‰ | 725‰ | 540‰ |
      | `elusive` (shipped, kept) | 826‰ | 725‰ | 546‰ |

      The trait is worth up to **201‰** here (`endurance` → `last_gasp` against a
      blighter), so it is a real lever and `endurance` is clearly the worst of the
      five. But `elusive` is **kept**: the two rows above it are +39/+83/+44 and
      +34/+0/+95, and `fightSquadsOver`'s own doc says a rate off 600 battles
      *moves tens of parts per thousand between instruments* — so `swiftness` and
      `last_gasp` are not separable from the shipped trait at this depth, and
      neither is separable from the other. Moving the trait and the kit together
      on an unseparable margin would make the reading un-attributable to either.
      A deeper reading is what would settle it; nothing here needs it settled.

      **The second shell (`DAT-012`), required before a rekit is committed.**
      Re-taken with `aSquadBesides` and the partner varied from the fixture's
      `withdraw`-carrying Squirtle to a Machop — which removes the heal mirror
      entirely. Its own mirror control read exactly 500‰ with 0 endless, and all
      six rows read 0 endless of 600:

      | kit | slugger | bruiser | blighter |
      | --- | ---: | ---: | ---: |
      | was shipped | 296‰ | 468‰ | 150‰ |
      | shipped now | 406‰ | 620‰ | 246‰ |
      | **the swap is worth** | **+110** | **+152** | **+96** |

      **Same sign on all three**, so there is no disagreement to stop on. The
      magnitudes are much smaller and the absolute levels quite different, which
      is `DAT-012`'s whole point restated: a seat-swap rate is a reading about the
      shell as much as about the seat. ⚠️ **So the 826/725/546 are IN-SAMPLE** —
      this shell is the one the sweep chose on — and the out-of-sample figure for
      the change is the +110/+152/+96. Both are in `mender_test.go`'s doc.

      **The gate, written down before the sweep ran and not moved afterwards.**
      Floor `450` unchanged; collapse line `400`; ship bar = clears 400‰ against
      all three opponents AND beats the shipped build by ≥100‰ on each;
      `stallShare` 10‰ unchanged; every rate quoted with its endless count
      including the zeros; any row at 0‰ or 1000‰ or endless >10‰ dropped as
      saturated and said so; and every run carrying the shell against a copy of
      itself, which had to read exactly 500‰ with 0 endless or the run stopped.
      **Cleared: 826/725/546 against 400, and +470/+499/+265 against 100.** Both
      mirror controls read 300-300, 0 endless.

      ⚠️ **A third Cleffa build was considered and REFUSED, by the catalogue's own
      standard.** The obvious candidate was the fixture's own hybrid —
      `moonblast, charm, moonlight, solar_beam` — but it is **dominated by the
      rekit on every opponent**: 601/518/508 with `endurance` (the trait the
      fixture actually fielded, so this is the 601/518 the old test read) and
      726/578/545 with its best trait, against the rekit's 826/725/546. It is
      neither the sweep optimum nor near it, its own slots were never swept, and
      it sits between two existing directions while beating neither.
      `TestABuildIsACatalogueOfChoicesRatherThanOfKits`'s doc names this exact
      failure mode — *inventing one to fill the row would be the lie this test is
      against* — and `happiny.tend` was catalogued because it **was** the optimum
      its sweep found. Same standard, opposite answer.

      **What the goldens say: nothing.** The rekit moved **no golden line at
      all** — the builds screen draws a build's id and name, and this change moved
      its skills and its intent — so `hexBuild` in `newbuilds_test.go` is the only
      design record guarding it, which is what that record is for. ⚠️ **The 24
      golden lines this commit does carry are `DAT-011`'s debt, not this
      item's**: `internal/screen` and `cmd/hexforge-tui` were **already red at
      `0c7ef64`** because #395 added `happiny.tend`/`happiny.decoy` to
      `builds.json` without re-accepting the two `screens.golden`. Verified by
      running the golden test against a clean `git archive` of HEAD. The whole
      diff is `pokemon.happiny` gaining two rows and `pokemon.mew` losing two off
      the bottom of a scrolling viewport.

      **What does NOT close, and why it is not this item's.**
      `cleffa.mend` still cannot be read on this shell at all, and the cause is
      the shell rather than the build — see `DAT-014`. Re-taken here at 300 seeds
      a side: **85 of 600** battles past the turn cap against a slugger (141‰,
      fourteen times the allowance) and **19 of 600** against a bruiser (32‰,
      three times it), so **both rows are dropped as unresolved rather than
      quoted**; only the blighter row resolves cleanly enough to read (4 of 600,
      548‰). ⚠️ **The item's original `cleffa.mend` row — 340/615/864 — may not be
      quoted by anything**, and neither may the 380/905 re-take above it: they are
      rates over a board an eighth of which never finishes. The figure has been
      withdrawn from `cleanser_test.go` rather than corrected.

- [x] `DAT-011` **Happiny's cleanser slot is twenty per mille under its design
      floor — EVERY LEVER IS NOW MEASURED AND CLOSED, and the floor turned out not to
      be a line any shipped build clears. What shipped is two catalogued builds and
      an instrument that reads one.** Split out of `ENG-012` on 2026-09-08 rather
      than fixed there: what to do about it is an authoring decision about a
      character, and a geometry fix is not the place to make one. Swept and closed
      2026-09-09.

      **What shipped.** Two builds in `builds.json` — `happiny.tend` (`egg_bomb`,
      `safeguard`, `heal_bell`, `soft_boiled` + `endurance`, the optimum the sweep
      found) and `happiny.decoy` (`egg_bomb`, `safeguard`, `taunt`, `soft_boiled` +
      `carapace`, the opposite shape) — plus `aThirdMemberFrom`, which fields a
      catalogued build's four skills **and its trait** in the fixture's third slot,
      and the cleanser test now uses it. ⚠️ **The reading did not move**: 429‰ and
      430‰ before and after, because the build is exactly the kit the test used to
      write out by hand. That is the point rather than a disappointment — the
      instrument changed and the number did not, so the number was never about the
      instrument. Happiny was one of five characters with no entry in the catalogue
      at all, and every character that has one has at least two, which is why two.

      `TestACleanserEarnsItsSlotWhereASparCannotSeeIt` holds that a slot a striker
      holds better is a slot the cleanser should not be in, at a floor of 450 per
      mille. The reading:

      | instrument | vs a slugger | vs a blighter |
      | --- | --- | --- |
      | as first written | 526‰ | 478‰ |
      | after the aim order was fixed (#196) | 485‰ | 460‰ |
      | after `ENG-012`, 1200 seeds | **429‰** | **430‰** |

      ⚠️ **Every step down was an instrument being repaired, not the game being
      changed.** Happiny's own kit is untouched by either fix — `safeguard` and
      `heal_bell` are `column`, which the rotation leaves alone, and `egg_bomb`
      and `soft_boiled` are single-target — and so is the Machop it is measured
      against, whose four skills are all single-target. What moved is the shared
      wall's `bubble`, the one `arc_down` in the book.

      **So the slot has never cleared its floor on an instrument that was not
      flattering it**, and the debt is 20 per mille rather than the 58 the
      shallow reading suggested. The test prints the shortfall against every run
      and asserts a **collapse line of 400** instead — the regression it was
      written against read 133 — so the floor stays named and visible rather than
      being moved to meet the number.

      **The sweep. Twenty-odd configurations, 400 seeds a side unless noted, and
      the numbers are here so nobody re-measures them.** The entry used to end
      "a second reading of the kit is the place to look before the stat line";
      the kit was read, and so was everything else.

      *The kit — seven alternatives, and the shipped four beat every one.* Each
      row differs from the base kit in exactly one slot, replaced by
      `hyper_voice`, which is the only other thing in the learnset that damages:

      | the slot given up | vs a slugger | vs a blighter |
      | --- | ---: | ---: |
      | none — the base kit | 407‰ | 456‰ |
      | `safeguard` | **163‰** (−244) | 345‰ (−111) |
      | `soft_boiled` | 401‰ (−6) | 363‰ (−93) |
      | `heal_bell` | **491‰ (+84)** | 333‰ (−123) |

      ⚠️ **`heal_bell` is the only slot in the kit whose REMOVAL improves a
      matchup**, and it is the character's own archetype. A cleanse is worth
      about +123 against a squad that brings something to strip and about −84
      against one that does not, so the fourth slot is matchup-dependent by
      construction — and a floor asserted **per matchup** is a floor no such kit
      can clear. It fires 2.5 casts a battle in both matchups against
      `safeguard`'s 14.1 and `soft_boiled`'s 13.2, which is the same fact counted
      a second way. The other three alternatives (`rally`, `taunt`, `rapid_spin`,
      `wide_guard`, and `hyper_voice` in the striking slot) all read lower still.

      *The trait — all six, and the fixture's hardcoded one is already the best.*
      `aThirdMemberAs` fixes `endurance` for every third member of every squad it
      builds, which nothing had questioned:

      | trait | vs a slugger | vs a blighter |
      | --- | ---: | ---: |
      | `endurance` | **407‰** | 456‰ |
      | `last_gasp` | 407‰ | 455‰ |
      | `convalescence` | 397‰ | 447‰ |
      | `composure` | 392‰ | 446‰ |
      | `carapace` | 276‰ | **648‰** |
      | `ballast` | **21‰** | 332‰ |

      ⚠️ **`ballast` reads 21 per mille — seventeen wins in eight hundred
      battles.** It grants `encumber` alongside `fortified`, and Happiny is the
      slowest unit on the board at 90 speed, so the trait's own cost lands on the
      one character least able to pay it. Worth knowing before anybody fields it
      elsewhere; it is not a Happiny finding so much as an `encumber` one.

      *The stat line — and this is where the shape of the whole problem shows.*
      Defence swept 220 to 400 over 200 seeds, then the mender's entire stat
      table put on the cleanser:

      | stat line | effective hp | vs a slugger | vs a blighter | stalls |
      | --- | ---: | ---: | ---: | ---: |
      | shipped, defence 220 | 8 320 | 392‰ | 447‰ | 0 |
      | defence 280 | 9 280 | 367‰ | 485‰ | 5 |
      | defence 340 | 10 240 | **524‰** | 394‰ | 9 |
      | defence 400 | 11 200 | **622‰** | 352‰ | 16 |
      | the mender's whole table | 9 520 | 473‰ | 652‰ | **68 of 600** |

      ⚠️ **Every way of buying survival moves the two matchups in OPPOSITE
      directions**, and the mechanism is one sentence: effective health buys wins
      against burst and loses them against attrition, because a longer battle is
      what a poison squad wants and the cleanser adds no damage to end one
      sooner. ⚠️ **And the two that buy the most push the board past the ten per
      mille of unresolved battles a reading here may carry** — 40‰ at defence
      400, 113‰ on the mender's table. That is the guarded-mirror shape `RAT-004`
      already fixed once, arriving through the stat line instead of the rating.
      ⚠️ **Defence 418 is refused outright** by the joint budget: 4800 health
      behind it absorbs 11 510 against a `max_effective_hp` of 11 500, so
      Blissey is twelve points from the ceiling at defence 400 and there is no
      room above it at all. Raising attack was already refused (660 needed).

      *Two things that are NOT the cause, checked so they are not re-checked.*
      The fixture's third slot is `{Col: 0, Row: 1}` and `AllyFrontCol` is 2, so
      the support already stands in the **back** column behind both carriers —
      the cell does not punish it. And the opponent is not it either: the mender
      clears 450 against all three of slugger, blighter and bruiser
      (601‰ / 508‰ / 518‰), so the instrument is not hostile to supports.

      ⚠️ **What the sweep found instead is that the comparison was never level,
      and that is `DAT-013`.** The mender's floor-clearing reading comes off a
      **hybrid of its two catalogued builds** that `builds.json` does not carry;
      the builds it does carry read 340‰ and 205‰ against the same slugger — both
      under the floor, one of them far under the cleanser. So the cleanser was
      being measured on its only kit against a number no shipped build produces.
      ⚠️ **`DAT-013` closed that on 2026-09-09 and two of the figures in this
      paragraph did not survive it**: `cleffa.hex` was rekitted and reads 826‰
      against the same slugger, and the 340‰ was withdrawn rather than corrected —
      that board puts 85 of 600 battles past the turn cap, so no rate may be
      quoted for `cleffa.mend` on this shell at all (→ `DAT-014`). The 205‰ and
      340‰ above are kept as the reading that was taken, not as figures to cite.

      **The conclusion, stated plainly: the 20 per mille stands and is not a
      defect in the character.** On catalogued kits Happiny is the **steadiest**
      support in the game — 407‰ / 456‰ / 431‰ across three opponents, a band of
      49 — against `cleffa.mend`'s 524-wide spread and `cleffa.hex`'s flat 180‰.
      A support whose worst matchup is 407 is worth more than one whose worst is
      340 and whose other direction loses four in five. The floor stays at 450,
      printed and unmet, because the number it is unmet by is smaller than the
      error in the thing it was being compared against.

- [x] `ENG-012` **A pattern's splash was walked in absolute board directions, so an
      authored formation did not play the same on the two halves.** Found on
      2026-09-08 while measuring the contested-speed alternation, by a control arm
      that would not close. **Done** — `pattern.Targets` and `TargetsAcross` take
      the caster's side and walk the shape in that frame.

      **The control, before the fix.** A squad against a copy of itself, fought
      one way and then with the halves exchanged, must sum to 1000‰: they are the
      same battles relabelled. Over 2000 seeds a squad:

      | squad | ally listed first | enemy first | sum before | sum after |
      | --- | --- | --- | --- | --- |
      | s01 | 629.5‰ | 370.5‰ | **1000‰** | **1000‰** |
      | s02 | 742.4‰ | 257.6‰ | **1000‰** | **1000‰** |
      | s03 | 614.0‰ | 505.0‰ | 1119‰ | **1000‰** |
      | s04 | 697.5‰ | 455.0‰ | 1153‰ | **1000‰** |
      | s05 | 582.5‰ | 474.5‰ | 1057‰ | **1000‰** |

      ⚠️ **This was NOT the aim-order bug and that one was genuinely fixed.**
      `battle.mirroredOrder` walks candidates by authoring slot and the synthetic
      mirror it was measured on summed to 1000‰ at one, two and three a side.
      That fixture's kits are `strike` and `sweep`, which carry **no splash** — so
      it confirmed the patch without confirming the property, and the shipped
      squads were not re-measured for a week.

      **The mechanism.** Driving both arms prompt by prompt: the two arms were
      offered the **same options in the same order** and the *ratings* differed —
      s03 turn 1, `razor_leaf` on Venusaur, 273 against 443 for the same aim.
      `pattern.targets` walked `Splash` as absolute cube steps, `arc_up` is
      `[["up"], ["upper_right"]]`, and `hex.Place` puts the enemy half down under
      a **180 degree rotation** — which maps *up* to *down*. So `pierce` spread
      back towards the midline in an enemy's hands instead of through the
      formation, and `wedge_right` pointed at its own backline.

      **The fix, and why it is this one.** The package's own doc already said the
      shapes were caster-relative — "an ally always faces east", "both cells
      deeper in", "how a shape can pierce into a formation" — so the walk was
      wrong against its own stated contract rather than open to an authoring
      decision. It is done by **conjugating the walk through `hex.Place`**, which
      is its own inverse: rotate the aim into the caster's frame, walk the
      absolute steps, rotate back. For an ally both calls are the identity, so
      **an ally's cast is byte-identical to what it always was** —
      `TestAnAllysShapeIsUnchangedByTheFrame` holds exactly that against a
      written-out copy of the old walk — and an enemy's is that walk reflected,
      **by construction** rather than by a table of mirrored direction names
      somebody has to keep true.
      ⚠️ Only 16 of the 151 shipped skills carry a shape this moves: `arc_up` (8),
      `flank_up` (6), `wedge_right` (1), `arc_down` (1). `single` (114) and
      `column` (21) are rotation-invariant — `column` is `up` and `down`, and the
      rotation exchanges them.
      ⚠️ The property test generates its shapes rather than reading the shipped
      book: the book uses four of the six directions, so a test over it would
      never walk `upper_left` or `lower_left` — which are exactly the two an
      enemy's `pierce` lands on now.

      **What it cost, and every one of the three was a MEASUREMENT fault rather
      than a design one.**

      1. `TestAShapeEarnsItsPowerWhereASparCannotSeeIt` — the claim survives and
         the depth was the problem. The margin read +56 before and +7 after over
         600 battles, and **+42 and +41 over 2400 and 6000**. A claim of the form
         "worth at least thirty" cannot be read off a rate that moves twenty
         either way, so that test has its own `shapeSeeds` now.
      2. `TestAStripEarnsItsSlotOnlyAgainstSomethingToStrip` — the regeneration
         row compared **totals** across two arms whose battle lengths were never
         held equal, and a regeneration gives back health *per turn*. The stripped
         arm ran 48,003 turns against 32,646 and its total came out **higher**
         while its per-turn healing was a fifth lower. It reads a rate now, at
         four times the depth, and the claim is what the rate says: the strip
         takes at least a fifth off, measured at 25 a turn against 35. **"Halves
         it at least" was never established.** The shielding row's healing is now
         asserted to be *level*, which the doc had claimed for as long as the test
         existed and nothing checked — and it is level only per turn, which is
         what makes the normalisation mutation-provable.
      3. `TestACleanserEarnsItsSlotWhereASparCannotSeeIt` — → `DAT-011`. The
         reading has fallen at every repair of the instrument and it is 20 under
         its floor now.

- [x] `SCR-012` **A draft's last pick can have one candidate, and the screen
      presents it as a choice — DONE, and the mechanism was already shipped: what
      was missing was the GAME CLIENT's record of it.** Split out of `NET-001`'s
      ban-and-pick item on 2026-09-08, which named it and shipped without it; that
      was the only thing keeping that box open.

      `draft.Slack` is `pool - 2*picks - 2*bans` — both sides, because both spend
      out of one pool — and the exhaustive walk holds the tight form: with every
      ban spent, the final pick sees exactly `slack + 1` candidates. So **slack of
      nought means the last picker has no decision to make**, and the screen used
      to draw it as a list to choose from anyway.

      ⚠️ **It is not reachable on the shipped cast today, and that was not a
      reason to skip it.** Slack was nought at sixteen characters
      (`pokemon.gible`), one at seventeen (`pokemon.pichu`) and is wider now — 22
      characters ship, 21 of them offered, so a 3v3's last pick chooses from twelve
      — so the state is behind the cast rather than gone, and it returns the moment
      a format changes the arithmetic or 5v5's pool is counted against a bigger ban
      allowance. A screen that reads correctly only because of a content figure is
      the shape this repository has been caught by before.

      **What it is, concretely:** when the offered list holds exactly one
      candidate, say it is the only one left rather than drawing a cursor over a
      single row.

      ⚠️ **This entry used to say the figure "is `draft.Slack`'s and has to be read
      from it rather than counted on the screen", and that half was WRONG** — the
      code argues the opposite and its argument is the stronger one, so the
      sentence is replaced rather than kept beside it. `draw.DraftLive.OnlyOne`'s
      own doc: that expression *"has answered 1, 2 and 4 within two days as the
      pool moved from sixteen characters to nineteen, so a screen holding the
      count, or a test asserting it, is a number that stops being true without
      anything saying so"*. **The rule is the list in hand — one entry is not a
      choice** — and it is right for every step and every format with nothing
      written down. `draft.Slack` stays where it belongs, in the *derivation* a
      test does (`cmd/hexarena-tui`'s `TestTheLastPickChoosesFromSlackPlusOne` and
      `pinchedDraftPoolIDs`), which is the one package that may name it at all.

      ⚠️ **Two goldens moved, not three, and it is not "both clients".**
      `cmd/hexforge-tui` draws no draft screen at all — nothing in that package
      names `DraftScreen` — so the two records that can hold this are
      `internal/screen`'s (the drawing) and `cmd/hexarena-tui`'s (the client's
      framing of it).

      ⚠️ **And a spectator cannot reach a populated draft today, so it is one
      entry rather than two.** Measured in
      `TestAWatcherOfADraftingRoomIsHandedADraftThatNeverAdvances`: a watching
      client *is* welcomed into a drafting room with `Welcome.Drafts` set, so it
      builds its own `*draft.Draft`, `Sight.Draft.Mirrored` is true and `model`
      really does put the draft screen in front of it — but `internal/room/draft.go`
      writes neither `wire.Drafted` nor `ClosureDraftExpired` to `Room.watched`, so
      its record stays empty until the draft closes. After the two players drafted a
      whole match the watcher was still holding **21 candidates out of 21, with 0
      decisions replayed**. Watching a draft is step 7 of *Ban and pick, and a
      spectator watching it*, and this state comes with it rather than before it.

      **What shipped before this item was opened** (step 5b, `internal/screen`):
      `DraftLive.OnlyOne`, the draw in `DraftScreen.head`, `i18n.DraftOnlyOne` in
      both books, the *a draft with one candidate* golden entry and
      `TestADecisionWithOneCandidateSaysThereIsNoChoice`, which holds the pair.

      **What this item added, all of it in `cmd/hexarena-tui`:** the wording had
      **nought hits** in that client's golden, and the client draws through a
      different path — `draftLiveOf` into a reading, rendered inside `frame` — so
      the package's pair could not stand in for it. Now: a *a draft with no choice
      left* entry in `everyScreen` (+164 golden lines, four renders, both languages
      at both sizes, a pure insertion),
      `TestTheClientDrawsAForcedPickAsNoChoiceInBothLanguages` for the pair, and
      `assertNamesNobodyAsTheOnlyOne`, which asks about **every** character in the
      pool rather than one — a check against a single candidate stays green on a
      screen wrongly naming any of the others.

      ⚠️ **The fixture CONSTRUCTS the pinched pool and is measured against the
      cast, which is the half worth reading.** `aPinchedPickScreen` draws from
      `pinchedDraftPoolIDs`: the **size** is derived
      (`2*PicksPerSide + 2*BansPerSide`, asserted to leave `Slack` at nought) and
      the **members** are the first n of the twelve `namedDraftPoolIDs` this
      package already names. Neither half alone is enough — a written-down ten
      stops being right when either design figure moves, and slicing the *shipped*
      pool is #328 exactly (a fixture picking by position moved 1,292 golden lines
      the day the cast was sorted, with no screen having changed). So
      `TestTheForcedPickEntryIsNotMovedByWhatTheCastShips` draws the entry twice —
      once over the shipped cast and once over a cast one character wider, the
      arrival inserted **in front**, because `cast.json` is in id order and only an
      arrival that sorts ahead of what a slice was taking can catch one — and
      demands the same bytes. Measured under mutation: replacing the named list
      with `draft.NewPool(all).All()[:tight]` reddens it twice over, on the pool it
      took and on the drawing (`pokemon.igglybuff` became `pokemon.happiny`).

      ⚠️ **The neighbour of one is where the negative half is measured**, not the
      whole pool: `2*picks - 2` picks taken leaves two candidates, and two is the
      only reading that tells `len(Candidates) != 1` from any `<= n` form. Under
      mutation `!= 1` → `< 1` reddens the pair test at two candidates and the
      golden entry's own assertion at seven. **A wording change is caught by the
      goldens and by nothing else**: a test that reads the book for its expectation
      cannot see the book move, so rewording `DraftOnlyOne` in `english.go`
      reddens both goldens on their English blocks and no assertion anywhere.

- [x] `SCR-011` ⚠️ **Every scratch data directory copied the whole 17 MB data
      directory AND re-ran the fixture injection — BOTH FIXED. But the ~1300s in
      this item's first draft was a WINDOWS reading, `make check` is green on
      macOS, and neither half of the cost was where the first draft put it.**

      **What shipped, first half: the art is shared.** `testfixture.CopyData`
      copies the books and *shares* the art: a hard link first, a symbolic link
      where a hard link is refused (another filesystem), a byte copy where
      neither is available (Windows
      without developer mode, across two volumes). All three keep the file's size
      and modification time, which is what the art preview's cache is keyed on —
      `TestThePreviewRasterisesOncePerFileAndSize` is green, and the copy
      fallback restamps deliberately so it is not a second behaviour under test.
      ⚠️ **Sharing means writing to art in a scratch directory writes to the
      committed file**, so `testfixture.PrivateArt` is how the one test that
      rewrites its art says so, and `CopyData` refuses on the next call if a
      shared picture has moved rather than letting a corrupted SVG go unnoticed.

      **FIVE packages had this, not one.** `cmd/hexforge-tui`, `cmd/hexarena-tui`,
      `cmd/hexforge`, `internal/forge` and `internal/screen` each declared a
      byte-identical `copyTree`. All five now call one function, which is what
      stops the sixth copy of the slow shape being written.

      One copy of the data directory writes **17,269,474 bytes before and 241,573
      after**, and a full run of the five packages makes **784 of them** — 13.5 GB
      of writes down to 189 MB. The *call sites* were counted at 155 in
      `cmd/hexforge-tui`; the copies it really makes are **301**, so a site count
      is about half the figure. (Re-measured with the staging in: a scratch
      directory writes **260,112 bytes** in 19 files and hard-links 66 pictures —
      the books plus the fixture's three — and 17,288,013 if the sharing is
      mutated out, which is what
      `TestAScratchDataDirectoryStillSharesRatherThanCopies` bounds at a
      megabyte.)

      ⚠️ **"Essentially all of it copying pictures" is FALSE on this filesystem,
      and that is the correction that found the second half.** Timed either way,
      one copy of the data directory takes **23 ms sharing and 22 ms copying** —
      seventeen megabytes of page-cached SVG costs nothing measurable on APFS. A
      whole `scratchData` ran **206–235 ms**, so the copy was under a tenth of it
      and `testfixture.Inject` — which re-parsed and re-saved the books through
      `forge` for every scratch directory — was about **180 ms** of the rest. The
      art was worth 5.3s of 112s. The lever was the injection.

      **What shipped, second half: the injection is staged once per process.**
      `testfixture.Data(target, shipped, load)` is the one entry point now, and
      all five `scratchData` helpers are shims over it. It injects the fixture
      **once a test binary**, holds the resulting **book files as bytes in
      memory**, and builds each scratch directory by writing those bytes out,
      linking the shipped pictures straight from the shipped directory, and
      writing the fixture's own three pictures fresh. `testfixture.Stagings()`
      counts the injections so a test can assert the arrangement instead of a
      stopwatch.

      ⚠️ **`Inject` reloads the library 37 times, not four.** Four *call sites*,
      but one for the skill dependencies and then one before every single write —
      31 skills, 3 origins, 2 characters — because each save rewrites a whole book
      and a library opened once would hold a stale copy of the file it is about to
      replace. That is the 180 ms, and it is why doing it once was worth the
      machinery.

      **Three questions this had to answer, and none of them went the obvious
      way:**

      1. ⚠️ **A stale stage would leave every test green** — every scratch
         directory built from books that had since moved, and nothing to say so.
         Two things stop it. Nothing is written to disk between runs (what
         survives an injection is bytes, not a directory), so **no cross-run cache
         exists** and the injector compiled into the binary is always the one that
         ran. And within a run the stage is held under a **SHA-256 of the shipped
         books byte for byte plus the fixture constants**, so books edited while
         the tests are running produce a new stage rather than a stale one.
         `TestTheStageKeyCoversTheBooksAndTheFixture` moves each input separately
         and `TestEditedBooksReachTheNextScratchDirectory` is the same claim end to
         end. The art is deliberately not in the key: no picture is staged, so
         none can go stale.
      2. **Nothing has to clean it up, because there is nothing left.** This item
         used to say the work "needs a `TestMain` in five packages that have
         none", and that is **wrong** — but the reasoning behind it was right for
         the design it assumed. A staged *directory* has no correct owner:
         `t.TempDir` and `t.Cleanup` are per test, so the first test to ask would
         delete it while the rest were still reading it, and "the OS, eventually"
         is the only honest answer left. Staging the **bytes** removes the
         question: `injectOnce` works in its own `os.MkdirTemp` and removes it
         before it returns, asserted on both arms
         (`TestAnInjectionLeavesNothingBehind`, `TestStagingLeavesNothingOnDisk`).
      3. ⚠️ **The fixture's own art stayed private, and that was a decision.**
         Staging it alongside the books would have been free and would have turned
         three pictures every suite writes near into files two tests could fight
         over — and worse, a picture shared *from a staged directory* is shared
         from somewhere under `os.TempDir()`, which `rememberArt` skips, so the
         write-through guard on the committed art would have gone quietly dead.
         So `WriteArt` is separate from `Inject` and runs per directory, the
         shipped pictures are still linked **straight from the shipped
         directory**, and `TestTheFixturesOwnArtIsPrivateToEachScratchDirectory`
         holds it. Checked while here: the row
         `TestThePreviewRasterisesOncePerFileAndSize` opens on is
         `fixture-anime.adept`, whose `assets/fixture/adept.svg` **has no
         committed counterpart at all** — so that test writes to a private file
         either way, and its `testfixture.PrivateArt` call is insurance against
         the ordering moving, exactly as its own comment says.

      **Measured 2026-09-08, macOS 26.6.2 on APFS, 8 cores.** Two `go test -c`
      binaries per package, run from the package directory with
      `-test.count=1 -test.v`, **interleaved before/after, three rounds** — a
      single pair is not a measurement here, and the first half of this item got
      the direction wrong from one. The machine was this session's own and not
      otherwise loaded, but it is not quiet: the before side of `cmd/hexforge-tui`
      is bimodal at 114s and 188s. **No range overlaps.**

      | package | before (3 runs) | after (3 runs) | `=== RUN` |
      |---|---|---|---|
      | `cmd/hexforge-tui` | 114.4 / 187.8 / 187.9s | 68.3 / 71.5 / 73.5s | 196 → 196 |
      | `internal/forge` | 114.1 / 117.2 / 148.4s | 94.0 / 104.7 / 111.1s | 172 → 178 |
      | `internal/screen` | 42.7 / 47.6 / 60.4s | 7.9 / 10.1 / 13.0s | 258 → 258 |
      | `cmd/hexarena-tui` | 21.2 / 22.7 / 36.1s | 8.3 / 9.4 / 10.9s | 90 → 90 |
      | `cmd/hexforge` | 8.6 / 9.2 / 12.3s | 5.9 / 6.4 / 6.9s | 55 → 55 |

      The counts are `=== RUN` lines: the same tests still run, and
      `internal/forge` is six higher only because that is where the six tests this
      owed had to live — `internal/testfixture` may not import a library to inject
      through, since that package's own tests import it, and it takes three more
      of its own (172 → 178 here, 6 → 9 there). ⚠️ The *measured* after-binary
      carried a temporary timing harness in place of one of the six, which is why
      its count matched; the six together run in about 1.5s and none of the
      figures above turns on them.

      **One `scratchData`, same binaries, interleaved, three rounds a side.**
      Before: **139–273 ms**, every call, eighteen of them. After: the process's
      **first** call is 151–186 ms — that is the injection, paid once — and every
      call after it is **18–30 ms**, fifteen of them. So a scratch directory costs
      about a **tenth** of what it did, and the whole 180 ms injection is paid
      once instead of ~300 times in `cmd/hexforge-tui` alone.

      ⚠️ **`internal/forge` is the package this did NOT rescue, and that is worth
      knowing before somebody optimises it next.** It drops about 20% and still
      runs ~100s: its cost is `spar`/`weigh` fighting thousands of battles, not
      its scratch directories. `internal/screen` is the other end — 4–6x, because
      almost all it did was build data directories and draw.

      ⚠️ **No before/after was taken for `make check` itself, deliberately.** It
      is green after, twice, at 5:12 and 3:36 — the spread is the machine, and a
      pair of runs on one side is not a reading, which is the whole lesson of the
      first half of this item. The per-package figures above are the measurement.
      What the gate adds is a ceiling: `internal/seed` runs 211s inside it and
      builds no scratch directory at all, so the gate can only ever move by less
      than the five packages do.

      ⚠️ **The gate is not red here and this item used to say it was.** `make
      check` was green at `d022cdf` before any of this, and `cmd/hexforge-tui`
      runs 106–112s alone (148s inside `make check`, where packages compete).
      719s, 1295s and the 600s timeouts were **Windows** readings — "one measured
      `start()` costs about 4–10s" was always qualified that way, and a scratch
      copy there is a filesystem and a virus scanner rather than a memcpy. So **no
      `-timeout` is owed on macOS**, the 16 MB term the Windows figure was blamed
      on is gone, and whether what is left still needs one there is unmeasured —
      it needs somebody on that platform, not another reading here.

- [x] `SCR-013` ⚠️ **`SCR-011`'s guard cannot pass in the environment `SCR-011` was
      measured in.** Five packages now assert that a scratch data directory
      **shared** the shipped art, and on Windows with the repository on one volume
      and `TMP` on another, both kinds of link are refused — so the byte-copy
      fallback is the normal path and the assertion is unconditionally red.

      ```
      --- FAIL: TestAScratchDataDirectorySharesTheShippedArt
          66 of 66 shipped pictures were copied rather than shared
      ```

      `internal/forge` (twice, with `…StillSharesRatherThanCopies`),
      `internal/screen`, `cmd/hexarena-tui` and `cmd/hexforge-tui`.

      **Both legs measured rather than assumed**, source on `H:` and the scratch
      directory under `C:\…\Temp`:

      | call | result |
      |---|---|
      | hard link | `The system cannot move the file to a different disk drive` |
      | symbolic link | `Administrator privilege required for this operation` |

      ⚠️ **The fix itself is fine and this is not a request to revert it.**
      `SCR-011` says so in as many words — *"a byte copy where neither is
      available (Windows without developer mode, across two volumes)"* — so the
      fallback is designed, and the entry's own numbers show the art was never the
      expensive half anyway (23 ms sharing against 22 ms copying; the injection was
      the lever). What is wrong is only that the **guard** for it is written as a
      property of the machine: its doc explains it is there to catch *"a fallback
      that had quietly become the normal path"*, which is exactly the state a
      cross-volume Windows box is in permanently and legitimately.

      Three ways out, each a decision rather than a patch: skip the assertion where
      neither link is available and say which (a `t.Skip` naming the refusal, so
      the test still fails on a box that *could* link and did not); assert the
      cheaper invariant the entry already bounds — bytes written, which
      `…StillSharesRatherThanCopies` caps at a megabyte and which a copy fails
      whatever the reason; or put the scratch directory on the repository's own
      volume so a hard link is available, which fixes the cost too and not just the
      test.

      ⚠️ Same family as `goldens-cannot-be-green-on-two-platforms`: a test that
      pins the machine of whoever ran it. That one pins the path separator, this
      one pins the filesystem's link support, and neither says so when it fails.

      **DONE — the first way out, and it is a PROBE rather than a reading.** The
      guard now attempts the two links itself, from a real shipped picture to a
      real name inside the very scratch directory the shares would have landed in,
      through the same `linkFile` / `symlinkFile` seams the sharing uses. Refused
      both ways it skips, quoting the two refusals; accepted, it asserts. ⚠️ **The
      distinction is the whole fix**: a skip keyed on *"was the art copied?"* —
      which is what the obvious reading of the report suggests — is true both on a
      box that cannot link and on a mechanism that stopped linking, so it would
      delete the guard on every machine forever.
      `TestTheArtGuardFailsWhereLinkingWorksAndTheArtWasCopiedAnyway` is what
      refuses that shape, and it goes red under it.

      ⚠️ **The second way out does not close this on its own**, which is worth
      recording because it reads as though it should: a byte cap is a better
      invariant than an inode comparison, but a copied picture is seventeen
      megabytes however the reading is taken, so `…StillSharesRatherThanCopies`
      is red on the Windows box for exactly the reason the inode one is. Measured
      under a simulated cross-volume box: `17300838 bytes were written for one
      scratch directory, want under 1048576`. It therefore skips through the same
      decision rather than growing one of its own. The third way out was refused:
      putting the scratch tree on the repository's volume buys a hard link by
      giving up `t.TempDir()`'s cleanup, would leave `git status` dirty, and would
      still pin the guard to a *property of the machine* — it makes today's
      Windows box link and says nothing about the next filesystem that will not.

      **The decision lives once**, in `internal/testfixture` beside `CopiedArt`,
      as `RequireSharedArt` and `SkipWithoutArtSharing` — the same reason
      `SCR-011` step 1 made one `CopyData` out of five `copyTree`s. Five copies of
      a skip rule drift, and a drifted skip is a guard that is off in the one
      package nobody looked at.

      ⚠️ **This entry undercounted the sites: it is SIX, in five packages.**
      `cmd/hexforge` has the same guard and is not named above — `callers_of`
      reports five callers of `CopiedArt`, plus `…StillSharesRatherThanCopies`.

- [x] `DAT-012` ⚠️ **A seat-swap rate is a reading about the SHELL as much as about
      the seat. Two shells answered the opposite question for one character, with
      identical data — and the tell was in a column that is not the rate.**

      Shipping `pokemon.magikarp` (Magikarp → Gyarados, `maelstrom`) was measured
      by putting it in a shipped squad's back seat and reading
      `forge.FightSquads`, 200 seeds each way, every mirror control exactly 500‰.
      Done twice, because the first answer was wrong.

      **Shell A — `s02` (Blastoise + Gengar), Gyarados for Magnezone:**
      1000‰ vs `s01`, 896‰ vs `s03`, 813‰ vs `s04`, against a control reading
      717‰ / 120‰ / 378‰. Read on its own that is a character to cut in half.

      **Shell B — `s04` (Machamp + Mew), Gyarados for Mewtwo, matched control in
      the same run:**

      | opponent | Gyarados in the seat | Mewtwo in the seat |
      |---|---|---|
      | `s01` | 632‰ | 907‰ |
      | `s02` | 463‰ | 621‰ |
      | `s03` | **95‰** | 452‰ |
      | `s05` | 1000‰ | 1000‰ |
      | head to head | 387‰ | — |

      Same character, same numbers, opposite verdict: strong in A, weak in B.

      ⚠️ **Shell A is the dishonest one and the `endless` column says so, not the
      rate.** Blastoise brings four skills of nought power, so `s02`'s damage is
      whatever sits in that back seat — which makes the seat's own contribution
      the squad's whole offence, and it makes the shell freeze when the seat is
      weakened. Measured while tuning the seat DOWN, shell A's own mirror went
      **endless 50 → 88 → 126** of 400 and its `s03` matchup **92 → 144 → 222**,
      while `Rate()` drops an endless battle from the denominator — so every
      figure improved on a shrinking sample. Shell B ran **endless 0** throughout.
      This is the same trap `hexarena-starter-squads` records and the second time
      it has decided a conclusion here.

      **The convention this fixes**, and it is cheap: a seat swap is read in a
      shell whose **other two members can win without the seat**, the control is
      fought **in the same run** over the **same opponents**, and the `endless`
      count is quoted beside every rate. A shell that cannot finish is measuring
      the shell.

      **Where the character landed.** Between two shipped dealers — clearly above
      Magnezone in shell A, consistently 158–357‰ below Mewtwo in shell B — and it
      shipped there rather than being tuned to match Mewtwo, which would have made
      water's first attacker top-tier on arrival. `hexforge check` reports no
      problems; effective health 6800 of the 11500 budget, attack 740 second only
      to `slugger`'s 760, defence third thinnest in the preset table.

      ⚠️ **One reading is left unexplained rather than tidied away.** Gyarados is
      water/wind against `s03`'s grass + electric/metal + fire, and the chart puts
      water over metal and wind over fire — yet it reads **95‰** there where
      **dark** Mewtwo, which is off the chart entirely, reads 452‰. So on that
      board a two-way elemental advantage is worth less than the stat and speed gap
      it is carried by. That is a question about how much an affinity is worth, not
      about this character, and nothing here measured it.

- [x] `ENG-003` ⚠️ **A one-way mirror rate stopped being a measurement above one unit a
      side — FOUND AND FIXED.** The skill that resolved in an order that does not
      mirror was the **aim walk**: `battle.aims` offered its candidates in
      absolute board order, `hex.Cells()` is column-major over the whole board,
      and `hex.Place` turns an enemy slot through 180 degrees — a rotation
      reverses rows where a column-major walk does not, so the two halves were
      offered their candidates in **opposite** orders. `Suggest` keeps the first
      aim that reaches the best value, so a tie between two identical targets
      fell to a different unit depending on which half was asking. The walk is by
      authoring slot now, the far half first (`battle.mirroredOrder`).

      **Measured, a squad against a copy of itself, 400 seeds each way.** Before:
      one a side **1000‰ exactly**, two **1035‰**, three **1330‰**; per seed the
      winner failed to swap in 0, **24** and **66** of 200. After: **1000‰ at
      every size** and **200 of 200** seeds swap.
      ⚠️ One unit a side could never show it — one enemy is no tie at all — which
      is why `TestABothWaysMirrorIsExactlyEven`, which fights at `duelSlot`, was
      green throughout and `spar` was sound. `forge.FightSquads` is now sound too.

      ⚠️ **It cost one thing and the cost is measured rather than argued.** The
      cleanser slot fixture reads 526→**485‰** against a slugger and 478→**460‰**
      against a blighter, both still a long way over its floor of 450, and one
      battle of its 600 now reaches the turn cap. Swept over 600 seeds both ways
      — 1200 battles against the 600 the test runs — the engine stalls **4 times
      with the fix and once without**, and the once is at seed **307**, outside
      the 300-seed window the fixture fights. ⚠️ **The zero-stall bar was a
      property of the seed window, not of the engine**, so it is a stated share
      now (`stallShare`, ten per thousand, one declaration for all five fixtures)
      and the count is logged whenever it is not nought. What a stall is here is
      the open item below: a wall against a healer, healing that nearly matches
      damage over a thousand blows, fourteen hundred declined turns.

      The original entry:

      ⚠️ **A one-way mirror rate stops being a measurement above one unit a
      side, and nothing in the suite says so.** A mirror fought one way and its
      reverse must sum to 1000‰. Measured, middle row, 1000 seeds each: one unit
      a side **1000‰ exactly**, two units **1021‰**, three units **962‰**.
      `TestABothWaysMirrorIsExactlyEven` holds the exact case and fights at
      `duelSlot` with one unit, so it does not reach the others; `spar` is
      therefore sound and `forge.FightSquads` has an unmeasured residual it sums
      over rather than cancels.
      ⚠️ Not the board: `Place` is a real isometry across the sides **and**
      within one — 0 asymmetric pairs of 81 — so `TestPlaceMirrorsBothSides`,
      which checks only the cross-side profile, was not hiding it. Not structural
      either: the shipped two-unit squad *is* exactly complementary while a
      synthetic two-unit mirror of the same characters on the same cells is not,
      and they differ only in **kit**. So a skill resolves in an order that does
      not mirror and it has not been found — which is what to look for first,
      before any figure at 3v3 or 5v5 is quoted. → `README.md` § PvP over a LAN.

- [x] `ENG-011` **`main` did not build, and no conflict was raised — found
      2026-09-08 by branching a worktree off it.** `guardcredit_test.go` (`RAT-004`,
      PR #358) calls `unit.MaxHP()`; another PR moved that reading to
      `Battle.MaxHP(unit)`, because a maximum is resolved against the modifiers in
      force and only the fight can answer it. The two changes touch **different
      files**, so the squash merge was textually clean and semantically broken.
      ⚠️ **`go build ./...` is not a gate for this**: it does not compile `_test.go`,
      so a test calling a moved API is invisible to it — `go build` passed while
      `go vet ./internal/core/battle` did not. `make check` runs `go vet ./...`,
      which does catch it; the gap is that neither was run between the merge and
      the next branch off it.
      Fixed in place: the call reads `fight.MaxHP(unit)` now.

- [x] `SCR-010` **Two client tests measured the machine rather than the code —
      found 2026-09-08 on the way through `ENG-004`.** Both are in
      `cmd/hexarena-tui/player_test.go` and both were red on this Windows box
      before anything in that branch was touched.

      **`TestAnAbsentPlayerFileIsSilentAndTheShippedSidesStillWork`** compared two
      **whole drawn screens** built by two calls to `startCarrying`, so the two
      stood in two scratch directories differing in one character — which the
      header line names. Whether that character was on screen came down to whether
      the path cleared the clip at `minWidth`, which the random part of a temp
      directory decides. It compares below the header now (`belowHeader`), for the
      reason both goldens already drop that line. ⚠️ **This is the third time this
      exact shape has been fixed** — `TestABracketScrollsWhereverAPageKeyDoes` was
      the second, earlier the same week — so the rule is worth stating plainly: a
      test comparing two whole drawn screens must drop the header, because the
      header names a directory the test itself made up.

      **`TestThePlayerSquadPathIsBuiltRatherThanRead`** wrote its directory as a
      POSIX literal and asserted the built path began with it. `PlayerSquadsPath`
      joins, and `filepath.Join` normalises to the platform separator, so a
      slash-shaped literal came back backslash-shaped on Windows and the prefix
      never matched. It builds the directory with `filepath.Join` now — the same
      shape as `memory/windows-sets-no-term.md`, one directory over.

- [x] `ENG-004` **The two-number surface is a different pair — CLOSED. The grid is
      REFUSED and the missing half of the pair shipped.** `self_bonus` is a
      weighable field now, so both of the two numbers can be priced along a line;
      the two-axis report is refused, and the reason is arithmetic rather than
      cost.

      ⚠️ **The pair has no interaction for a grid to show.** The two numbers meet
      in exactly one expression — `combat.Swung(power, bonus, share)`, which is
      `(power + bonus) × (1000 + share) ÷ 1000`: one product, one truncation. So
      per blow the surface is **rank one**, and every cell on a hyperbola is the
      same cell: `bonus 0 / share 1000`, `bonus 1000 / share 0`, `bonus 250 /
      share 600` and `bonus 600 / share 250` are one figure at a power of a
      thousand, and the identity survives the truncation because the division is
      taken once, of the product. Held by
      `TestSwungIsAProductAndNotASurface` and, under it,
      `TestSwungReadsBothTermsAtAll` — a row of equal products is also satisfied
      by an expression that ignores one of the two terms.

      **The two other reads of either number are event fields.** `turn.go` puts
      the share on `SkillUsed` as `Gradient` and the bonus on `Amplified` as
      `Power`; a renderer prints both and neither reaches the battle. So there is
      nowhere else for an interaction to hide.

      ⚠️ **The honest limit of that, stated rather than left for somebody to
      find.** `self_gradient` declares `at_empty` and the *realised* share is a
      function of the caster's health, so at BATTLE level the two are not
      interchangeable. A grid could therefore still draw a shape — but the shape
      would be the feedback loop (a wounded caster hits harder, so it wins
      sooner, so it is less wounded), which is a property of the board. A grid
      over two skill fields would be attributing it to the fields, which is the
      same shape of number this file has been burnt by twice.

      **What shipped instead: `self_bonus`.** The gradient was weighable and the
      bonus was not, so of the pair only one axis had a line at all. The field
      reads and writes `SelfRequires.BonusPower`, and ⚠️ `set` **copies the
      condition** rather than building one — which status it reads, how many
      stacks, whether it gates, whether it consumes are the skill, and a weighing
      that reset any of them would price a different skill and look identical
      doing it (`TestMovingASelfBonusLeavesTheRestOfTheConditionAlone`).

      ⚠️ **The bench could DECLARE the mechanic and not reach it**, which is the
      fixture trap this repository keeps a list of. `vent` is the bench's only
      enemy-aimed powered skill with a condition of its own, it gates on three
      stacks of `swelter`, and **nothing in the bench applies one** — so the
      first end-to-end run cast it nought times and the weighing refused the row,
      correctly and in its own words. The fuel is authored into the scratch
      library by the test (`stokesSwelter`) rather than added to the bench, on
      the same terms as `bringsTheGradient`: the bench is what a hundred goldens
      draw.
      ⚠️ A bonus the size of `vent`'s own 2400 power **saturates** the board and
      is refused — *one slot wins 100.0% of what it decides and the other 100.0%*
      — so the test prices 400. That refusal is the instrument working.

      ⚠️ **The counts in the original entry were stale in BOTH directions**, which
      is why they are re-taken here: **2** skills carry a gradient (`comeback`,
      `reversal`) and not 1, **12** carry a `self_requires` and not 7, of which
      **5** declare a bonus (`outrage` 1200, `flare` 550, `thorn_volley` 650,
      `bloom_burst` 2200, `tide_break` 3800). The intersection is still **empty**,
      and the pairing is legal — `resolveGradient` refuses a gradient only beside
      a condition that reads **health**, and says so.

      The original entry:

      ⚠️ **The two-number surface is a different pair, and no skill in the book can
      carry it.** `combat.Swung(power, bonus, share)` is the one place two of
      these compose: the **bonus** is `self_requires`', the **share** is
      `self_gradient`'s. `resolveGradient` already refuses a gradient beside a
      `self_requires` that reads *health* ("two curves off one number is a skill
      nobody can price"), so the only legal pairing is a **status** threshold
      with a gradient — and the intersection is **empty**: 1 skill has a gradient
      (`comeback`), 7 have a `self_requires` (`outrage`, `flare`, `pyre`,
      `thorn_volley`, `bloom_burst`, `tide_break`, `deluge`), none has both. So
      the two-axis report has no subject and would have to be **authored before
      it could be measured**, which is the wrong order. That is why this half is
      still open while the field above shipped.
      **What a grid costs, measured rather than reasoned.** `Battles()` is
      `2 × seeds × rows` and a grid squares the rows. Wall clock on this machine:
      `pokemon.cleffa` at 118 median turns fought **80,000 battles in 115 s**
      (≈1.4 ms a battle), `naruto.naruto` at 273 turns fought **20,000 in 135 s**
      (≈6.8 ms). So a **4×4** grid at the tool's default 10,000 seeds is
      **320,000 battles ≈ 7½ min** on a short pairing and **≈36 min** on a long
      one; a **5×5** is 500,000 ≈ 12 min and **≈57 min**. At `--carriers all`'s
      2,000-seed default a 4×4 is 64,000 ≈ 1½ and 7 min. ⚠️ **Battle length
      dominates the count** — the same number of battles costs five times as much
      on Naruto as on Cleffa — so a row budget stated in battles is not a row
      budget stated in minutes.
      ⚠️ **`MonotoneWorth` has to be answered before a grid is built, not after.**
      *"Is more of this sometimes worth less"* has a direction only along a line.
      On a surface there is one answer per row, one per column and **none for the
      surface**, so a grid printing a single ordered/not-ordered footer would be
      inventing a figure with no referent — the same shape of number this file
      has already been burnt by twice.
      → `internal/core/skill/skill.go` (`resolveGradient`),
      `internal/forge/weigh.go` (`WeighField`).

- [x] `RAT-003` ⚠️ **A declined turn makes a slow board slower — RE-TAKEN, and every
      statement in the old entry was wrong.** The figures were dated 2026-09-03,
      *before the block-charge clause landed*, and nobody re-took them. Measured
      2026-09-07, 200 seeds a row at a limit of 600 turns, mirror boards of
      Blastoise stat lines:

          board                     1v1 endless   2v2 endless   turns
          no guard                     0.0%          0.0%       66 / 115
          endurance                    0.0%          0.0%       72 / 126
          carapace                   100.0%        100.0%        —
          withdraw                   100.0%        100.0%        —
          both                       100.0%        100.0%        —

      - **Not "a wall-heavy roster".** A plain **1v1** stalls. Squad size is not a
        term in it at all.
      - **Not 175/800 and 308/800.** It is **100%**, and it is not the pass rule:
        the no-guard mirror and the `endurance` mirror both resolve every seed.
      - **Not `frozen()`.** That predicate is right to return false — both units
        *can* aim at each other. They choose not to.

      ⚠️ **There are TWO causes and only one of them is a defect.**
      - **`carapace` is a rating defect.** A blow a `bastion` pool would eat whole
        is priced at **nought**, the pass rule reads "nothing worth doing", and both
        sides pass for ever — so the pool is never spent because no blow is ever
        thrown at it. Measured to the turn: over eight turns neither unit acts once,
        and after 600 turns both stand at **3600/3600 with 576 pool left**. → the
        entry below.
      - **`withdraw` is arithmetic, and no rating change touches it.** The rating
        does *not* refuse here — a `water_gun` lands for 146 on turn 5 — but
        `withdraw` restores **500** every four turns. The heal outruns the damage
        and nobody can die. That is the data, on a mirror, and it is not a bug.

      ⚠️ **`docs/balance.md` asks for "a wall board that FINISHES" and no method
      was written under it. Here it is: only ONE side may carry the guard.**
      Measured: `withdraw` vs plain **0% endless, ~104 turns**; `withdraw` vs a
      heavier kit **0%, ~104**; `carapace` vs plain **0%, ~67**; either of them
      mirrored **100%**. That is also the right shape for the question, since what
      a guard is worth is what it buys against a side not carrying one.
      `TestAGuardBoardResolvesOnlyWhenOneSideCarriesIt` holds all three rows,
      including the negative one, so the day a mirror starts resolving the
      condition says so instead of guarding nothing.

- [x] `RAT-004` **A guarded mirror never resolves, because the rating will not spend a
      guard — DONE, and the fix this entry costed is NOT the one that shipped.**
      `battle.guardCredit` counts what a **permanent** guard ate as progress, at
      100 per mille. `Set.PermanentPoolIn` is the new reading it needs.

      ⚠️ **The flat credit this entry recommended is not shippable, measured.**
      Crediting every guard was written first: `pokemon.happiny` and
      `pokemon.squirtle` against copies of themselves go from resolving every seed
      to **40 of 40 endless**, which breaks `TestABothWaysMirrorIsExactlyEven` — a
      fairness invariant, and the same one that refused the `stat_debuff` shield
      change. No share buys a way out: the shipped mirrors need it at **10 or
      under** and the guarded fixture needs it at **50 or over**, an empty
      intersection. That is the "blast radius" this entry named, arriving as a red
      invariant rather than as moved figures.

      **What separates the two is whether the guard COMES BACK**, and the shipped
      book makes the split cleanly: `withdraw` puts up `block` (duration 2, cast
      again), `brace` puts up `heft`, and only `carapace`'s `bastion` is permanent —
      granted once, refused a second stack, gone for good once empty. A bite out of
      a renewable guard buys the turn it takes to return; a bite out of a permanent
      one is the only progress the rating can honestly call progress.
      ⚠️ So the three shipped guard builds this entry worried about —
      `squirtle.fortress`, `squirtle.ram`, `machop.charge` — are **untouched**: all
      three carry timed guards. The blast radius came out at nothing, and
      `./...` moved no golden and no balance test.

      ⚠️ **The share IS measurable, and this entry said it was not.** That sweep
      asked one question — does the guarded board resolve — and every value from a
      tenth up answered yes identically, which is a step and not a curve. Asking a
      second board closes it. Swept at five per mille either way, once the rule was
      narrowed to permanent guards: **≤ 45** the mirror stalls again (the credit is
      one truncating division and 21 × 45 ÷ 1000 is nought — the resolve-only sweep
      would have scored 45 a success); **50 … 999** every board holds; **1000** a
      point of guard is worth a point of health, the guarded and the bare target
      tie, and `take` keeps whichever the aim walk reached first. 100 errs low
      inside that, which is the direction every horizon in this file errs in.

      ⚠️ **A flat credit had a fourth edge that vanished with it**, kept here
      because it is the shape of the mistake: at 200 an `unblockable` blow stopped
      being the answer behind a barrier, because the ten-times-heavier blow the
      barrier ate whole out-rated it. Narrowing to permanent guards took that board
      off the share entirely — `aegis` is timed.

      Held by four tests: `TestAGuardedMirrorDoesNotStandStill` (the defect),
      `TestABlowThatTakesHealthOutratesOneThatOnlyTakesGuard` (the ceiling),
      `TestOnlyAGuardThatStaysSpentEarnsCredit` (the distinction) and
      `TestTheGuardCreditSitsInsideItsMeasuredWindow` (where the number came from).
      ⚠️ **Two of the three board tests were vacuous first and the fixtures record
      it**: with the guarded enemy in the slot the aim walk reaches second, the tie
      fell the right way by accident and both were green with the rule deleted.

      The original entry:

      ⚠️ **A guarded mirror never resolves, because the rating will not spend a
      guard.** Split out of the entry above, where it is measured. A blow a pool
      would absorb whole rates **nought**, so `Suggest` passes rather than throwing
      it, so the pool is never depleted — a unit with a `bastion` and an opponent
      with an attack stand at full health for six hundred turns without either one
      acting.

      **It is the same family as the burrow finding and one step worse.** There the
      rating mis-priced a new option; here every option reads nought and the rating
      stops **playing** — which hides better, because two sides standing still look
      like a long battle rather than a broken one.

      **A costed fix exists and is not shipped.** Crediting what a blow takes out of
      a guard — `whole - landed`, at a share, added back in `Battle.against` — fixes
      the `carapace` mirror at **any share from 10% upwards**: 100% endless → **0%
      endless**, ~77 turns at 1v1 and ~135 at 2v2, and 10%, 25%, 50% and 100% give
      the *same* answer. A step rather than a curve, which is the signature of an
      option that was reading exactly nought: it only has to be positive.
      ⚠️ It does **not** touch the `withdraw` mirror at any share, because that
      stall is arithmetic and not a refusal.
      ⚠️ **The blast radius is real and is why it is not shipped here.** Three
      shipped builds carry a guard — `squirtle.fortress` and `squirtle.ram`
      (`withdraw`), `machop.charge` (`brace`) — so this moves balance figures and is
      the author's call rather than a bug fix. What it needs before shipping is the
      one thing this file always asks: the figure re-taken on the shipped roster,
      one change at a time.
      ⚠️ And the share needs a reason, not a rung. Every value tested gives the same
      board outcome, so the sweep cannot choose one — the argument has to come from
      what destroying a guard is *worth*, the way `shielded` prices a charge from
      the defender's side.

- [x] `DAT-003` ⚠️ **A `hexforge new` churned `screens.golden` — DONE, by the third option
      this entry named and had not tried.** `aSquadOfSide` picks by a **property
      the screen measures** now, not by a position and not by a name: the most
      traits at the cap (the roster row draws statuses), the widest kit as the
      tie-break (the option list is one row a skill), the id last. Both readings
      are taken at the cap and the furthest form, which is what the fixture
      fields.
      ⚠️ **The file's order cannot reach any of that**, which is the whole point:
      `TestTheBattleFixtureIgnoresTheCastFileOrder` reverses the cast and demands
      the same picks — and asserts the first pick really is the property's top,
      because "survives a reversal" is satisfied by any constant. Adding a
      character moves this only if it is more extreme on the property than what
      is already picked, which is an authoring event a golden should move for.
      ⚠️ **Neither of the two obvious answers was taken**, and that was the
      decision: keeping the index churns on every unrelated edit, and naming a
      character breaks the rule the fixtures here are written under.
      ⚠️ **It cost a one-time move of 588 lines** — the fixture fields different
      characters now — and it surfaced a **latent hole in a helper**:
      `steppedByTheRoom` acted on a prompt whose turn was `Skipped`, which the
      engine refuses ("no unit is waiting to act"). No character the helper had
      ever been handed produced a skipped turn, so the branch had never been
      reached; a fixture deciding what a test can measure is the same shape as
      the entry above. The helper walks past it now, as a room does.

      The original entry:

      ⚠️ **A `hexforge new` still churns `screens.golden`, and sorting the cast
      did not fix that half.** Measured while reformatting the cast: the reorder
      moved **1,292 lines** of `internal/screen/testdata/screens.golden`, because
      `aSquadOfSide` (`internal/screen/play_test.go`) builds its squad as
      `characters[index % len(characters)]` — so which characters every battle
      screen shows is a function of the cast file's ORDER, and the next character
      whose id sorts before the third moves them all again.
      ⚠️ **The obvious fix is against a rule that fixture states outright**: it
      names no character on purpose, because "a test that names a character breaks
      the day somebody edits the cast for a reason that has nothing to do with it".
      Both rules are about not churning on an unrelated edit and they point
      opposite ways here, so this is a decision to take rather than a bug to fix —
      and it was deliberately not taken inside a commit about file formatting.
      ⚠️ **The churn is the loud half; the quiet half is measured.** See the
      `m.wrapped` entry above: the identical fix moved **no golden at all** on the
      tree before #242 and seven lines after it, because sorting the cast moved
      the browse cursor onto a character whose biography sits on the boundary.
      A record holding one character can only ever see that character, so this
      fixture decides what every detail pane's golden is able to report.
      A third option exists and is untried: pick the squad by a property that is
      neither the name nor the position — the first n that satisfy something the
      screen is actually measuring (a unit with a trait, a unit at an evolved
      stage), which would move only when that property moves.

- [x] `SCR-007` **The battle screen says how many cells a shape catches and never which —
      DONE.** The aim list now draws, under the aim the cursor is on, every other
      cell that aim catches and whoever is standing in it, marked with the same
      `..` the shape diagram uses for the same thing and explained on the heading
      only when a row carries it. At most two extra rows, because `max_targets`
      is three.
      The coverage comes from a second entry point rather than a second walk:
      `forge.AimCoverage(id, aim)` returns the same `ShapeCoverage` values from
      the cell under the cursor, and takes the crossing decision off the skill —
      `declared.Target.CrossesSides()`, the predicate `battle.covers` reads — so
      the cells drawn are the cells the resolution would walk. ⚠️ It answers about
      CELLS and not about damage: a burrowed unit standing in a splash cell takes
      nothing, so the screen says what an aim reaches and not what it will hit.
      ⚠️ **The feature was invisible to the whole suite when it landed, and that
      is the part worth keeping.** The cast `battleCast` picks brings four
      `single` skills, so every existing test and every golden drove a screen on
      which the new rows are correctly absent — and would go on passing with the
      rows deleted. Twenty-one area skills ship in builds.json. So the fixtures
      here choose their character **by property** (the first whose furthest kit
      carries a shape covering more than one cell) and there is a second golden
      entry, `aiming at an area skill`, which is what makes the mutation
      "the rows are not drawn at all" redden a picture as well as a test.
      ⚠️ One mutation is held by a test alone and not by the golden — reading
      `Aims[0]` instead of the cursor — because the golden's cursor sits on the
      first aim that catches anything, which is often index nought.

- [x] `DAT-004` **A golden holds a state and says nothing about the path to it — SWEPT,
      and the one rule the sweep found held by nothing is now held.** The first
      half stands as a fact rather than a task: every fixture that draws the aim
      list reaches it with `p.Aiming = true` (`screens_golden_test.aBattleAiming`,
      `cmd/hexforge-tui/language_test.go` twice, `cmd/hexarena-tui/sweep_test.go`
      twice), so a golden holds a *state* and **no keystroke rule in this
      repository is held by a golden**. That is correct for a golden — a picture
      of a state is what it is for — and it is why each such rule needs a
      behaviour test of its own.

      **The loop sweep, and what it found.** Every `for` in the three screen
      suites whose body presses a key was read. Three classes, and only one is a
      defect:
      - **Tolerant assertion — one, and it was the known one.**
        `live_test.go`'s `for action.Kind == Stay && struck.Aiming` reached an
        Answer whether the list opened once, twice or not at all. It is now
        exactly two presses with the first asserted to return `Stay` and leave
        the screen aiming.
      - **Waits — sound.** Every `… && time.Now().Before(deadline)` loop in
        `cmd/hexarena-tui` (draft, match, clock) **asserts the state after
        itself**, so a timeout fails rather than passing quietly.
      - **Drivers — sound, now labelled.** The two `for range 2 { … key("enter") }`
        loops in `match_test.go` and `draft_test.go` agree with any number of
        presses by construction; they exist to reach the result screen. Both now
        say out loud that they hold nothing and name what does. The rest
        (`for m.form.cursor != field`, `for m.squad.Field != field`, …) navigate
        to a known target and assert elsewhere.

      **The mutation sweep, which is what the loop sweep could not answer.** Eight
      keystroke rules on `PlayScreen` were reverted one at a time against a git
      baseline and the four screen suites run: **all eight reddened**, so none was
      unheld. But five of them redden only `TestTheLiveFootersNameNoKeyTheScreenIgnores`,
      whose own doc says it *cannot see whether a named key does the right thing*
      — it sees that a key did **something**. For four of those five that is the
      right holder, because "does nothing on a live screen" is the whole rule (the
      save key, `n`, `u`, `a`).
      ⚠️ **The fifth was a real hole and is the same shape as the original bug.**
      `Update`'s `p.Pending == nil || (p.Live && p.Answered)` guard is what stops a
      live screen answering twice, and its named assertion was
      `if action.Kind != Stay` — which a screen that had forgotten it answered
      satisfies by **opening the aim list**. Since #322 made the list open on every
      skill, that assertion had stopped distinguishing "dropped" from "started the
      turn again". Dropping the clause left the test green. It now measures what
      the press did *not* do — no action, not aiming, and a screen that did not
      move — and the mutation reddens it.
      ⚠️ **Not swept:** the mutation pass covered `PlayScreen` only. `DraftScreen`
      and `ArrangeScreen` were read for tolerant loops and have none, but no rule
      of theirs was reverted and watched.

- [x] `ENG-005` **A field inside a `modifier` was dropped silently in all three books —
      DONE, and the asymmetry it warned about was closed with it.**
      `Modifier.UnmarshalJSON` decodes strictly now, so the typo lands as a
      sentence naming the field in a status, a trait and a skill at once; and
      `skill.ParseBook` and `passive.ParseBook` refuse an unknown field at their
      own level too, which is what stops "a misspelt field on a status is refused
      and the same typo on a skill is not" from being the new hole.
      ⚠️ **Measured before either was turned on, which is the half a stricter
      decoder needs**: both shipped books decode strictly as they stand, so this
      refuses nothing that ships — the risk named below (a stricter decoder can
      refuse the data it was written for) was checked rather than hoped about.
      ⚠️ **It refused something anyway, and that is the finding.** The test
      fixture's electric skill carried `"damped": 400` inside its `requires`, a
      field from an **earlier design** that `skill.go` still names in a comment as
      abandoned — dead data every suite in the repository had been reading past
      since it was dropped. Nothing was measuring it and nothing could: the whole
      point of the hole is that the file looks like it worked. It is deleted, and
      it is the argument for the change rather than a cost of it.
      The tests are at the level that was missed: `internal/core/modifier`'s own,
      because a book asserting its strictness proves nothing about what its
      custom unmarshaller does, plus a near-miss typo per book (`accuarcy` on a
      skill, `grant` for `grants` on a trait) — a singular where the book wants a
      plural is the typo this is actually for, since a field nobody ever declared
      is usually caught by the author writing it. The original entry:

      **A field inside a `modifier` was dropped silently, in all three
      books.** Found 2026-09-06 while closing the same hole in `statuses.json`:
      `status.ParseBook` now decodes with `DisallowUnknownFields`, and that flag
      **stops at a custom unmarshaller**. `modifier.Modifier` has one, so
      `{"target":"attack","mode":"add","amount":100,"amout":100}` inside a status,
      a trait or a skill parses clean and the typo goes nowhere — the exact defect
      the status fix was written for, one level down.
      ⚠️ The fix belongs to `modifier` and not to any of its three readers, which
      is why it was not done in passing: `Modifier.UnmarshalJSON` decodes
      `modifierFile` with `json.Unmarshal`, so the change is four lines there and
      it lands in every book at once. What has to be checked with it is the other
      direction — `passive.ParseBook` and `skill.ParseBook` do **not** refuse
      unknown fields at their own level either, so a `flavour` on a status is
      refused today while a misspelt field on a *skill* is not, which is a new
      asymmetry in place of the one that was closed.
      ⚠️ And a stricter decoder is the change that can refuse the data it was
      written for. `TestShippedStatusBook` is what caught nothing this time
      because statuses.json was already clean; the two bigger books have not been
      through it.

- [x] `SCR-008` **The cast listing draws every row, so the detail pane shrinks as the cast
      grows — DONE.** `screen.browseRoom` is `skillsRoom`'s twin now: the listing
      measures its rows from the window in hand and windows them around the cursor,
      and the pane keeps `browseDetailRows` whatever the book's length.

      ⚠️ **The reserve is a written-down constant rather than a measurement, and
      that is a cost finding.** The honest version is `speciesRoom`'s — render every
      row's pane and take the tallest, which cannot drift. Measured on the shipped
      book at twenty-three characters it costs **13ms a redraw**: `os.Stat` behind
      the art row is **53µs a character** on Windows and `Context.Wrapped` is **15µs**
      a call against ten of them a pane. That is per keystroke and it **grows with
      the cast**, which is the very thing the reserve exists to stop mattering. So
      the number is written down and `TestTheDetailReserveIsTheTallestPaneInTheBook`
      holds it against the book **exactly, in both directions** — over is a pane the
      frame cuts, under is rows the listing gave up for nothing.

      ⚠️ **One number rather than one per language, and English pays for it.** The
      pane is **24 rows in Vietnamese and 17 in English** (`pokemon.mew`, at level 16
      and at MinWidth, where every wrapped row wraps hardest), because every glossed
      row draws nothing when there is no name to draw. A screen may not branch on
      which language is in front, and what is measured here is a count of rows rather
      than a width a label can be asked for — so the reserve is the tallest the pane
      ever gets and an English reader sees blank rows under it. Same price
      `speciesRoom` already pays for its note.

      ⚠️ **At the 120x24 floor the pane is cut anyway and no split can help**: the
      reserve alone is 24 rows against the 20 the frame leaves a body, so the listing
      could give up every row it has and still not fit. The floor of three is
      navigation kept rather than a share of a budget that balances. What this fixes
      is the other axis — a taller window used to lose the pane one row per character
      shipped and now loses none.

      The stopgap is gone: `theForkedBrowser` in both clients' sweeps stands in the
      shared window again rather than setting a taller one of its own.

      ⚠️ **It found a flake it did not cause.** `TestABracketScrollsWhereverAPageKeyDoes`
      compared two whole drawn screens, and every model in it is built by its own call
      to `start`, so the three stand in three scratch directories differing in one
      digit — which the header line names. Whether that digit was on screen came down
      to whether the path cleared the clip at `MinWidth`, which the random component of
      a temp directory decides, so it failed on about half the runs at whichever site
      and language landed on the boundary. It compares below the header now, for the
      reason both goldens already drop that line.

      The original entry:

      ⚠️ **The cast listing draws every row, so the detail pane shrinks as the
      cast grows.** Found 2026-09-05 while shipping `pokemon.torchic`, when a sweep
      entry that records a **forked** detail pane stopped drawing the form row.

      The skill listing already solves this: `screen.skillsRoom` measures how many
      rows the listing gets **from the window in hand**, so the pane below it
      keeps its rows whatever the book's length. The cast listing has no
      equivalent — it draws all of them. At the sweep's shared 120x44 and nineteen
      characters the browser's detail pane now runs out before the last row and
      the screen says so: *"… bị cắt bớt; cửa sổ cao hơn sẽ thấy hết"*.

      ⚠️ **The row it drops first is the one that matters most.** For a line that
      forks, the form row is the whole decision — `(Poliwrath@32 | Politoed@32)`
      is drawn, but the chooser under it is what says which arm is fielded. So the
      degradation is not "a bit less prose"; it is the pane losing its point,
      quietly, on a screen that already has a notice for it.

      ⚠️ **It gets worse on a schedule.** Five characters shipped on 2026-09-05
      alone. Nothing about the listing bounds it, so every character costs the
      detail pane one more row, on every window size.

      What was done instead, and why it is a stopgap: `theForkedBrowser` in
      `cmd/hexarena-tui/sweep_test.go` now sets its own taller window, because
      bounding the listing changes what **every** screen draws and belongs in its
      own change rather than riding in with a character. The comment there says so
      and points here.

      The fix is `skillsRoom`'s twin — measure the listing's rows from the window,
      window it around the cursor the way the skill list already is, and let the
      detail pane keep the rest. ⚠️ It moves every golden that draws a cast
      listing, which is most of them; that is the change being visible rather than
      a fixture problem.

- [x] `RAT-005` **The rating could not price hiding — DONE.** `pricing.hidden` is the term,
      and it is `taunting` pointed the other way: a taunt is priced by what it
      *forces* an enemy to do, hiding by what it *stops* them doing, and both ask
      about the enemy's options rather than about the holder's danger. That is the
      `brace` precedent the entry named, arrived at through the sibling that was
      already in the file rather than through `selfSpendable`.

      ⚠️ **What is denied is the DIFFERENCE, not the attack.** An enemy whose best
      blow was aimed at somebody else loses nothing when the holder vanishes — it
      hits that somebody else, exactly as it was going to. So each enemy
      contributes what its best blow on the holder beats its best blow on anyone
      else by, over the turns the status lasts, and an enemy that never wanted this
      target contributes nought. Pricing the whole attack instead had a rear-line
      unit burrowing to deny blows that were never coming at it, which is what
      `TestSuggestDeclinesABurrowThatDeniesNothing` holds.

      ⚠️ **It reads nothing off the holder's health**, which is the cheap wrong
      answer the entry warned about: "the damage I would take this turn" peaks
      exactly when the holder is one point from dead, the moment a turn spent
      hiding is worth least.

      ⚠️ **Two known inexactnesses, in opposite directions, neither computed.**
      Over: a burrowed unit is still caught by splash, so part of the denied blow
      arrives from the cell next door. Under: the status also refuses every
      application thrown at its holder, and that is not priced at all — exactly as
      `taunting` prices no status either. Both are written on the function.

      Four mutations, three red — the term deleted, the whole attack denied instead
      of the difference, and the horizon dropped. The fourth (deleting the side
      test) is **green and expected**: `aims` offers an enemy-aimed skill nothing
      but enemy cells, so `bestAgainst` between two units of one side is nought
      anyway. `taunting` carries the same redundant line for the same reason, and
      the function now says so.

      ⚠️ **The horizon needed a figure test, not a decision test.** Dropping
      `× turnsOf(kind, buffHorizon)` halves every price and reddens no decision on
      any board that is not tuned to the exact rung where the halving flips a
      comparison. `hiding_term_test.go` is in package `battle` for that reason and
      asks the term directly: one equality for the arithmetic, and one for twice
      the turns being worth twice the denial.

      ⚠️ **`burrow` is in no shipped build.** It is in `pokemon.diglett`'s learnset
      and nowhere else, so the auto-battle now knows what hiding is worth and no
      shipped roster carries the skill to use it. Putting it in a build is an
      authoring decision and deliberately not folded in here — but a player build
      that takes it is no longer a unit one skill short.

- [x] `RAT-006` **Two self-cast skills in one kit — RE-MEASURED, and the premise did
      not survive.** The reading this was raised on reproduces exactly and means
      something else. Diglett against Machop, 200 seeds each way, cast counts beside
      the rate:

      | kit | rate | casts |
      |---|---|---|
      | shipped `diglett.three` | **725‰** | `split` **0**, dig 1750, earthquake 1846, rock_throw 1610 |
      | `burrow` for `rock_throw` | **110‰** | `split` **0**, burrow **1258** |
      | `burrow` for `dig` | 45‰ | `split` **0**, burrow 1288 |
      | `burrow` for `split` | 550‰ | burrow 1214 |
      | that slot simply empty | **780‰** | `split` **0** |
      | `split` dropped entirely | **725‰** | W290 L110 — the shipped tally to the battle |

      ⚠️ **Three readings overturn the diagnosis, and the third is the decisive
      one.** `split` is uncast in the *winning* kit as well, so "the rating never
      cast the split" is not what changed. Dropping `split` altogether leaves the
      shipped figure bit for bit — same wins, same losses — so against this
      opponent that slot contributes exactly nought whether `burrow` is beside it
      or not. And `burrow` is not a slot the rating declined: it is cast about
      three times a battle, and each of those turns is what the collapse is made
      of. The entry blamed a silent skill for a loud one.

      ⚠️ **The authoring rule it asked for is REFUSED as written.** "One
      turn-spending self-cast per kit" is contradicted by the shipped catalogue:
      `bulbasaur.parasite` holds `synthesis` and `ingrain` and plays both, because
      a heal's price moves with the health it is aimed at and two such skills
      therefore take turns; `squirtle.fortress` holds three and plays all three. A
      count would refuse two builds that use every slot they name.

      **What shipped instead is the rule the reading actually wanted, measured.**
      `forge.Library.Census` counts how often a build casts each of its own skills,
      and `TestEveryShippedBuildPlaysItsOwnKit` holds *a build may not name a skill
      it never casts* over the whole catalogue.
      ⚠️ **The board is the finding inside the finding.** A duel is blind to
      whole categories: `squirtle.fortress` casts `taunt` **0** times in 132 duels
      and **252** times in as many squad battles, because in a one on one a taunt
      denies an aim nobody had a choice about. A mirror is blind differently —
      against a copy of itself a skill can be dominated by its own kit-mate, and
      seven shipped builds read a dead slot they play against anybody else. So the
      census stands the subject **in** an authored squad in place of its first
      member, against that squad intact, and the rule walks the four of them.
      `TestACensusSeesASlotThatCannotFire` is the other half: a kit whose
      `wrecking_swing` has no `brace` beside it to fuel it must read silent, or the
      green above is green for nothing.

      **And the real defect, which was in the rating.** `pricing.hidden` made two
      errors and both are now corrected: it counted the hiding window in the
      HOLDER's turns when what a hide denies is the ENEMY's (`overTheWindow`), and
      it priced a denied turn at the heaviest blow in the enemy's kit when a denied
      turn is an ORDINARY turn of that enemy's — the correction `turnWorth` has
      been the written statement of since the `outrage` measurement. On the board
      above that takes `burrow` from 5502 to 1362.
      → `RAT-008` for what is left, which is not a factor — and which was chased,
      measured and refused the same day, with the sweep that puts the size of the
      remaining hole at five or six times rather than the thirty two samples had
      suggested.

- [x] `RAT-008` **A denied enemy turn is not a lost enemy turn — REFUSED, and the
      refusal is a measurement.** Raised 2026-09-08 out of `RAT-006` and closed the
      same day. The term the entry asked for was **written, measured and thrown
      away**; what is left is this entry, the sweep below and one regression test.
      The precedent is the `hidden` clamp in `README.md`: an argument that sounds
      right is not a measurement, and an inert term is not kept as insurance.

      **First, the entry's own headline number is wrong, and the sweep is what said
      so.** `RAT-008` read "roughly **thirty times**" off two samples — `/4` and
      `/64` — with nothing taken in between. Sampled properly, on the Diglett split
      build against Machop, 200 seeds both ways, casts beside the rate:

      | `hidden` divided by | rate | `burrow` cast |
      |---|---|---|
      | 1, as shipped | 110‰ | 1258 |
      | 2 | **145‰** | 1258 |
      | 3 | **105‰** | 1200 |
      | 4 | 175‰ | 1200 |
      | 5 | **555‰** | 926 |
      | 6 | **775‰** | 814 |
      | 8 | 775‰ | 814 |
      | 16 | 775‰ | 814 |
      | 32 | 775‰ | 802 |
      | 64 | 775‰ | 802 |
      | the term returning nought | 780‰ | 0 |

      ⚠️ **The hole is five or six times, not thirty, and it is a CLIFF rather
      than a slope.** Everything from `/6` to `/64` reads the identical 775‰ —
      `/64` was never a reading of "how far it must fall", it was a reading of the
      plateau the collapse recovers onto at `/6`. And below the cliff the rate is
      **non-monotone**: 110, 145, 105, 175. That is the shape a win rate has in a
      price, which this file already says of `swiftness` and of the column bonus,
      and it has a consequence that decides this item — **no partial correction to
      this term can be seen in a rate at all.**

      **The term that was built.** `pricing.kept(unit)`: the best of an enemy's
      **self-aimed** casts, priced through the rating's own `rate`, subtracted from
      `lost` after the `turnWorth` cap. It is the exact complement of the floor the
      entry below refused — `turnWorth` walks only `aimedAtAnEnemy` skills, which is
      precisely the set a hide removes, and `kept` walks only
      `declared.Target == skill.Self`, which is the set `aims` still offers a unit
      whose target is underground. One ply, no aim walk, no board advanced, nothing
      rolled, memoised per unit.

      ⚠️ **It priced exactly what it was derived to price and moved nothing.**
      Probed on the opening board: `kept(machop)` = **550**, taking `lost` from
      1293 to **743** — a 43% cut, over half the distance to `/2`. Measured over
      twelve rows (the four split kits against Machop, and the whole build with and
      without `burrow` against Squirtle, Charmander, Machop and Gastly): **not one
      win rate moved, in any row.** The only figure that moved anywhere was ten
      `burrow` casts of 1586 in the Squirtle row — the one opponent carrying a
      self-aimed skill of its own, `withdraw`, which is what says the term was live
      and pointed at the right set rather than dead.

      ⚠️ **And the cliff says why, arithmetically.** 743 is on the wrong side of
      it: the recovery needs the term at about 1293/5, so `kept` would have to be
      worth roughly **1050** rather than 550. It cannot be. `brace` prices through
      `granted`'s Reserve arm into `selfSpendable`'s gated arm, which is
      `strike(machop)/5` — a divisor that deliberately under-prices a gated
      spender's fuel and is right to, for the term it belongs to. **The expressible
      half of "the enemy keeps its turn" is about half the size of the hole, by
      construction.** That is the finding: not that the term is wrong, but that the
      correct term is too small to reach, and a term built to reach it would be a
      number chosen to hit a target.

      ⚠️ **The other half is DEFERRAL and it is refused for `RAT-007`'s reason.** A
      hide does not remove the enemy's blow, it moves it later, and it removes it
      only if the battle ends first — which needs a **horizon this rating does not
      have**. `RAT-007` names the two available lookaheads and each breaks a rule of
      `price.go`: one rolls, the other is a second copy of the resolving arithmetic.
      `kept` was neither — same `rate`, same board, same turn — and that is exactly
      why it could only reach the half that fits in one turn.

      ⚠️ **`elsewhere` is not the way in either**, unchanged from the raising: it
      reads the enemy's best strike at the holder's other allies, so it is nought
      exactly when the holder is the only target — every duel, and every board
      where a hide looks best.

      **What is left in code is one test**,
      `TestBurrowStaysOutOfTheSplitBuild` (`internal/seed/diglett_test.go`), which
      holds the standing consequence: `burrow` in `diglett.three` reads below the
      shipped kit AND the slot-empty control reads at or above it, so the loss is
      hiding costing more than the slot is worth rather than a slot being spent.
      Its premise — that the kit casts `burrow` at all — is asserted, because a rate
      cannot say whether a build played its kit. Three mutations, three red: the
      term deleted (the cast count goes to nought and the premise fires), the term
      at `/6` (the shipped comparison fires), and the slot-empty control weakened
      (the data claim fires).

      `burrow` therefore stays over-priced in a duel and about right in a squad —
      `diglett.whole` plays it 7 times over the census boards and reads 317 of 440
      against the cast — so the shipped catalogue is not carrying the defect, and
      `TestEveryShippedBuildPlaysItsOwnKit` is what keeps that true.

- [x] `FRG-003` **The cast census has no command — DONE.** `hexforge census <build>
      [--against SQUAD] [--seeds N]` prints the kit in its own order with a count
      beside each slot and the silent ones marked, and it is on the dispatch table
      and in the usage.

      ⚠️ **The command runs the WALK by default, not one board, and that is the
      whole design decision.** The rule a build has to pass is not "this slot fired
      against this squad" — a slot silent against one squad is a fact about that
      matchup, and `split` is uncast against eight of the twenty-one characters and
      cast four hundred times across the rest. So with no `--against` it walks the
      authored squads and stops at the first board that leaves nothing silent, and
      `--against` is the opt-in to one named board with a note saying, in the
      output, that it is **not** the catalogue rule. A reader handed a single board
      would take a silent row as a failed rule, and it is not one.

      ⚠️ **The walk moved out of the test and into the package**, as
      `forge.CensusWalk` plus `Library.CensusWalk` and the exported `CensusSeeds`.
      It was a helper in `census_test.go`; the moment a command asked the same
      question, leaving it there would have meant the rule worded twice — the
      mistake `CLAUDE.md` § *Mistakes already made here* records. The test helper is
      now the call and a `t.Fatalf`, and the reasoning (why the walk, why six
      seeds) lives with the code that does it.

      `Library.Build(id)` is new and its doc answers `Library.Builds`', which rules
      a by-id lookup out **for a screen**: a screen has the row it drew, and a
      command line has a string a person typed and nothing else.

      **What it prints beside the numbers is the board** — `CastCensus.Against` on
      every table — because a count without the squad it was taken against is a
      figure nobody can re-take. The seeds and the battle count come off a census
      that was actually taken rather than off the flag, for the same reason, and a
      mutation printing the default instead is red.

      ⚠️ **A verdict read off the last board is the one thing this front-end could
      get wrong on its own**, because the walk stops early: the last table drawn is
      always the board that played everything, so a verdict taken from it reads
      "plays every slot" in identical words for a build that needed one board and
      one that needed four. `TestACensusVerdictIsAboutTheWalkAndNotTheLastBoard`
      holds it. ⚠️ Its first version asserted `Contains(page, "3 board")` and
      **passed with the verdict collapsed** — the head line says "3 board(s)
      walked" — so it now slices the verdict off the page and asserts there. Found
      by mutation, not by reading.

      Six mutations, all red: the verdict collapsed to the one-board wording, a
      silent slot unmarked, the board unnamed, the walk never stopping early, the
      seeds printed from the flag default, and the subcommand off the dispatch
      table.

      ⚠️ **A second caller wants a shape this does not have: a BUILD duelled
      against a character.** `hexforge spar` duels *characters* on their learnset
      kits and `Census` stands a build in a *squad*, so neither can field an
      arbitrary kit in a one on one — which is the whole instrument `RAT-008` was
      measured on, and it had to be written as a test helper
      (`theDiglettKitAgainst`) because no command could do it. Worth folding in
      here rather than raising separately: it is the same subcommand with the
      opponent named instead of the squad.

- [x] `ENG-006` **A gate at the top of the health bar — `passive.Condition.AboveHealth`.**
      Built and measured on 2026-09-07 while closing the `reckless` item, reverted
      because that trait is not the one that wants it, and kept here because the
      *mechanism* is sound and is the only cost shape this engine cannot currently
      express.

      **All three steps are SHIPPED (2026-09-08).** Steps 1 and 2 built the
      mechanism — the term exists, parses, round-trips, is worded, the pricing
      defect it exposed is fixed, and a grant may carry a gate of its own — and
      **step 3 shipped the first subject, `pristine`, priced by measurement.**
      Until it landed nothing shipped carried either the term or a gated grant,
      so every walk over the shipped books was blind to both; each now walks
      exactly one trait.

      What step 1 landed:
      - `scale.AtOrAboveShare`, the twin of `AtOrBelowShare`. The overlap at the
        threshold is stated in the doc comment and held by
        `TestTheTwoSharesOverlapAtExactlyOnePoint`, which sweeps a whole bar and
        counts its own iterations: **for every value at least one end answers yes**
        (no hole), and **exactly one value — the threshold — is answered by both**
        (no band). ⚠️ On integers the shared point only exists when the threshold
        divides cleanly, which is why the sweep names its maximum.
      - `passive.Condition.AboveHealth`, `conditionFile.above_health`, and two new
        refusals in `ParseBook`: a **band** (both ends in one clause) and an
        **empty clause** (neither). Both ends are `omitempty` in the file schema
        and both have to be, or a gate at one end writes a nought at the other and
        the write no longer re-parses.
      - ⚠️ **One existing refusal message changed, deliberately.** `below_health: 0`
        was `is in force below 0 health, want a share in parts per thousand` and is
        now `is gated on a while with no threshold in it: say below_health or
        above_health` — with two ends, nought at one of them means "not this end",
        so a clause with nought at both is asking about nothing rather than asking
        badly. `TestConditionRejections` carries the new expectation and a comment
        saying it was changed rather than discovered.
      - `Condition.AtTop()` and `Condition.Threshold()`, so the four places that
        render or word a gate ask *which end* and *what figure* in one place.
        **All four read `While.BelowHealth` directly before this** — two renderers
        (`internal/i18n/describe.go`, `cmd/hexforge/list.go`) and two shipped-data
        walks (`i18n.TestATraitsSharesAreRoundedAndNotTruncated`,
        `seed`'s flavour walk) — so each would have printed or demanded a nought
        for a gate at the top of the bar. The walks are latent breaks rather than
        current ones and were moved anyway, since nothing ships the term.
      - `Marshal` carries both ends, held by a parse → write → parse test that names
        the surviving gate rather than leaving it to `DeepEqual` — two books that
        both lost it compare equal.
      - Wording: `BlurbTraitWhileAbove` beside `BlurbTraitWhile` in **both** books,
        a branch in `internal/i18n/describe.go`, and the gate column in
        `cmd/hexforge/list.go`, whose word was the literal `"under"`.
      - **The pricing defect below is fixed**: `price.go`'s reply term reads the
        gate through the new `Battle.inForceAt` against `holder.HP - dealt`, which
        is where `answer` reads it, and the comment arguing the old imprecision was
        safe is gone.

      ⚠️ **What no test could hold at step 1, said plainly rather than implied.**
      No golden recorded the new wording, because a golden over the shipped books
      cannot see a term no shipped trait carries — `describe.golden` did not move
      and could not until step 3 shipped a subject, which it now has. The same reason makes three of step 1's tests
      **hand-built fixtures** rather than walks: the i18n wording test constructs
      its two traits, `cmd/hexforge`'s writes them into a scratch data directory,
      and `internal/core/battle`'s pricing case adds `fresh_spikes` /
      `sturdy_spikes` to that package's own fixture book. Mutating
      `Condition.Threshold` to answer the wrong end leaves `internal/seed` and every
      shipped walk in `internal/i18n` **green**; only those constructed cases redden.

      **Step 2 is SHIPPED (2026-09-08): `Grant.While`, the per-grant gate.** The
      two-tier shape is writable — an ungated grant carrying X beside a gated one
      carrying the difference, on **one** trait — and the whole-trait gate is
      unchanged for the case it was always for.

      What step 2 landed:
      - `passive.Grant.While`, `grantFile.while`, and one **effective-gate**
        expression, `Passive.GateOver(grant)`: the grant's own clause where it
        carries one and the trait's otherwise. ⚠️ **That is the whole correctness
        story.** A grant is gated if the trait is gated **or** the grant is, and
        the two refusals a gate owns are rules about the *grant* rather than about
        where the clause was written — so a version reading `Grant.While` alone
        waves through every health-raising grant and every absorbing pool sitting
        under a **trait-level** gate, silently, with both refusals gone quiet.
        Every refusal, every renderer, the parser and the battle ask `GateOver`;
        `gateOver(trait, own)` is the same expression for the parser, which has to
        answer it about a grant it has not finished building.
      - **One gate over a grant, never two.** A grant carrying a clause on a trait
        that is itself gated is refused at parse: that is a conjunction, which is a
        band wearing two clauses instead of one, and every screen words a gate as a
        single sentence. So a trait is gated *as a whole* or *on its grants*, never
        both, which is what makes `GateOver` a single answer.
      - **The band and empty-clause rules are read once**, in a new
        `readCondition`, called for the trait's clause and for each grant's. The
        refusals are phrased as clauses ("is gated …") so the caller says whose
        gate it is — a trait's reads `passive %q: is gated …` and a grant's
        `passive %q: grants %q, which is gated …`. **No trait-level message
        changed.**
      - `Battle.reconsider` re-reads a trait whose gate is only on a grant.
        ⚠️ **Two halves, and either alone ships the feature dead**: the early
        return became `!held.Gated()` (trait-level `while == nil` leaves a
        per-grant gate stuck at whatever enlistment put it at), and the single
        `wanted` for the whole trait became one gate per grant (a single answer
        moves the ungated tier with the gated one). `Battle.hold` and
        `Battle.Begin` are per grant for the same reason, and `Battle.grant`'s own
        gate check went away rather than being duplicated beside `hold`'s.
        `Battle.holds` is the one reading of a gate against a unit; `inForce` and
        `inForceAt` delegate to it.
      - `Marshal` carries a grant's gate, both ends `omitempty`, held by a
        parse → write → parse test that names the surviving gate. `Book.All`
        deep-copies it: cloning the grant slice copies the pointer, not the clause.
      - **Wording**: `BlurbTraitGrantsWhile` / `BlurbTraitGrantsWhileAbove` in both
        books and a third branch in `i18n.DescribePassive`. A grant behind its own
        gate says when **in its own sentence**, because the trailing
        `BlurbTraitWhile` line words a *trait's* gate and a trait carrying a gated
        grant has none — so a two-tier trait reads "Always carries X." above
        "Carries Z at or above 70% health.". A trait gated as a whole keeps exactly
        the arrangement it had. `cmd/hexforge`'s grants cell carries the same
        clause through a shared `gateClause`, since the while column words the
        trait's gate and would leave a two-tier trait looking always-on.
      - `forge.Library.Held` skips **per grant** rather than per trait, which is
        what its comment always meant: a whole-trait skip threw away the tier that
        never lapses as well. The behaviour for a trait gated as a whole is
        unchanged, and the understatement at the top of the bar is still a step-3
        measurement.

      ⚠️ **What no test could hold at step 2, again.** No golden moved and none
      could: `describe.golden` cannot see a term no shipped trait carries. Every test for
      step 2 is therefore a hand-built fixture — `internal/core/passive`'s own
      book, `internal/core/battle`'s golden-free fixture book (a `fortified`
      status and four traits), `internal/i18n`'s constructed traits, and scratch
      data directories in `internal/forge` and `cmd/hexforge`. The seed share-floor
      walk gained a grant-gate arm that walked **nothing** the day it was
      written, deliberately, the way the trait-level arm did before `blaze`; it
      walks `pristine` since step 3.

      ⚠️ **Two refusals had no test before this.** Mutating the absorb refusal to
      read the grant's own field reddened **nothing** in the repository until the
      new table arrived — "a pool is refilled every time a gate reopens" was
      written and never measured.

      **Step 3 is SHIPPED (2026-09-08): `pristine`, on Magnezone at level 48.**
      An ungated `plated` (a new permanent buff, +10% defence) beside the shipped
      `fortified` (+25%) behind `above_health: 700`. The gated tier reuses
      `fortified` rather than getting a number of its own, because that is what
      `ballast` pays `encumber` for — the gate and the encumbrance are two prices
      for the same goods, which is what makes the two traits a choice rather than
      a ladder. The two tiers may not be the same status: `ParseBook` refuses
      `grants %q twice; say the stack count instead`, and a gated re-grant of what
      the trait already grants ungated is exactly that shape.

      **The measurement, and it is the whole reason the item existed.** Build
      duels on the established instrument — both arrangements over the same seeds,
      500 seeds a side, 1000 battles a cell, fixture arms patched into the parsed
      passive book in memory — with the **mirror control reading exactly 500‰ at
      every rung of every sweep**.

      | opponent | floor | gated | payload | tier ungated | tier behind the gate |
      |---|---:|---:|---:|---:|---:|
      | Cleffa | 460 | 682 | 708 | **+248** | **+222** |
      | Gastly | 222 | 260 | 438 | **+216** | **+38** |

      **A magnitude cannot separate those two matchups and a duration does.**
      Ungated the tier is worth 248 and 216 — 15% apart, so no amount of it would
      ever have been a lever between them. Behind the gate it is worth 222 and 38,
      **5.8 times apart**. That is the item's thesis measured, and it is the first
      reading in the repository where a trait's worth separates two matchups by
      more than a rounding error.

      ⚠️ **The axis is not battle length and the naive reading is backwards.**
      Cleffa is the *longer* matchup (42 turns against 20), and the gate is open
      for a comparable share of both timelines (369 and 323 parts per thousand).
      What differs is **where along the bar the fight is decided** — decomposed by
      fighting the same tier gated at the other end, Cleffa's payload is +222
      above the gate and +228 below it, Gastly's +38 and +122. So the item's own
      phrase, "a rout that ends at full health", is the axis and "a short battle"
      is not.

      **Both numbers are the last rung before a cliff.** The gate swept at
      200/300/500/600/700/800/900 delivers 100/90/90/89/89/4/3 per cent of its
      payload against Cleffa and 75/67/44/29/17/10/6 against Gastly: below 700 the
      two arms converge into a discount on a stat, at 800 the whole thing dies.
      The ungated tier swept at 5/10/15 per cent is better than `endurance`
      against 2/10/12 of the other 21 characters and worse against 11/4/2 — at 5%
      nobody would take it and at 15% it is `endurance` plus a bonus, so **10% is
      the only rung of the three that is a trade**.

      **`forge.Library.Held` was left skipping a gated grant at both ends and its
      stated reason was replaced.** That reason — a raise that "lapses on the
      first blow" is worse to show than none — is false at the top of the bar: the
      gate holds until its holder has spent three tenths of its health and is in
      force for about a third of a battle. A third is what settles it: the
      function returns one number, its two available answers are nought and the
      whole tier, and nought is the nearer of the two. Reading the gate at full
      health instead would make the preview equal the opening board and overstate
      by two thirds where this understates by one (373 against 452).

      **The goldens moved and the diff is the record.** `describe.golden` gained
      seventeen lines and they are the first appearance of
      `BlurbTraitGrantsWhileAbove` anywhere — *"Always carries plated. / Carries
      fortified at or above 70% health."* — against the trailing whole-trait clause
      `blaze` prints instead. The three screen goldens gained a status row and a
      trait row (`pristine  nguyên vẹn  pokemon.magnemite@48`, since the traits
      pane lists who carries each), the status picker went 43 → 44, and the
      statuses reference pane pushed `bastion` off its visible window — a scroll
      rather than a cut. **No balance golden moved**: no shipped placement fields
      the trait, exactly as none fields `carapace`.

      → `README.md` § *What a guard that holds while its holder is fresh is worth*
      for the full tables, and `internal/seed/pristine_test.go` for the four tests
      that hold it.

      ⚠️ **`internal/core/skill`'s `Condition` was left alone, on purpose.** It
      carries its own `BelowHealth` and the two types share arithmetic rather than
      a type, so `AtOrAboveShare` is available to it the day it wants one. It should
      not follow automatically: a skill's condition is a **composable amplifier**
      (`Status`, `MinStacks`, `BelowHealth`, `BelowStacks`, all combinable, nought
      meaning unasked) where a trait's gate is **one exclusive clause** — which is
      the whole reason a band is refused here — and it comes in a pair,
      `Requires` and `SelfRequires`, so "above health" there is two different
      features (paying a skill off for hitting a healthy target, and for being
      healthy) each with its own pricing question. Neither is this item.

      **Why it is worth having.** Every dial a trait offers today is a **stat**,
      and a stat cannot separate two matchups: both gates it moves are the same
      event — a strike crossing a kill threshold — so every amount moves the long
      grind and the short rout together. That is the wall all three of `reckless`'s
      own dials hit. **Duration is not a stat.** A trait behind `below_health`
      arrives when its holder is losing; one behind `above_health` *leaves* then —
      so it costs a great deal in a battle that goes long and nothing at all in one
      that is over while the holder is healthy. Nothing else in the book can say
      that.

      ⚠️ **Do not resurrect it for `reckless`.** It was measured there and the size
      is not there: best rung **29.7%** against a floor of **36.9%**, and the curve
      is not monotone. → `README.md` § *What gating `reckless` was worth* for the
      table. What this needs is a trait authored *for* it — a big grant whose
      author wants it to lapse as its holder is worn down — rather than a trait
      being rescued with it.

      **What it costs to build**, from having done it once:
      - `scale.AtOrAboveShare`, the twin of `AtOrBelowShare`. The two overlap at
        exactly one point on purpose — a value on the threshold satisfies both —
        so a pair of traits either side of one number covers the whole bar with
        nobody falling through.
      - `passive.Condition` gains the term; the file schema gains `above_health`;
        `ParseBook` refuses **both ends at once** (a band is two rules wearing one
        clause, and every screen words a gate as one clause) and refuses a `while`
        holding no threshold at all. ⚠️ That second refusal **changes an existing
        message**: `below_health: 0` used to be "want a share in parts per
        thousand" and becomes "no threshold in it", which is a case in
        `TestConditionRejections`.
      - `Marshal` carries the new term, or a round-trip drops it.
      - Wording: a second key beside `BlurbTraitWhile` in both languages, a branch
        in `describe.go`, and the gate column in `cmd/hexforge/list.go`, which
        hard-codes the word "under".

      **The first subject, as asked for: a guard that holds while its holder is
      fresh.** Raise defence by X%, and while the holder is above Y% health raise
      it by Z% instead. It is the polarity `reckless` wanted turned around — a
      *benefit* that lapses as its holder is worn down rather than a *cost* that
      does — and it decouples the same way and for the same reason: worth a great
      deal in a rout that ends at full health, worth almost nothing in a grind. It
      is also the natural counterpart to `blaze`, which grows as its holder falls,
      so the two would read as a pair rather than as one idea twice.

      ⚠️ **The whole-trait gate could not express it, and that is what step 2
      settled.** `While` gates the *trait*, so what was expressible was one tier —
      "+Z% while above Y, nothing below" — and the two-tier shape needed a
      **per-grant** `while`: an ungated grant carrying X and a gated one carrying
      the difference. Splitting it across two traits was never the answer, because
      a character has trait slots and one idea may not cost two of them.
      `Grant.While` is that field and it ships; the trait-level clause is kept for
      the whole-trait case, the two may not be combined over one grant, and both
      refusals came down with the gate. **The pricing is above, and it closed the
      item.**

      ⚠️ **Two tiers do not add up to the sum of their faces.** `modifier.Set.Stat`
      saturates a change towards a floor rather than applying it — a −400‰ term on a
      base of 400 fights at 290, not 240 — so an author writing X and Z and reading
      them as percentages will be charged noticeably less than the arithmetic says,
      and the second tier is worth less than the first because it is applied into an
      already-moved stat. Price both tiers by measurement, not by addition; see
      § *What softening `bare`'s defence was worth* for how short that dial turned
      out to be.

      ⚠️ **And one real defect that only appears once the term exists.**
      `price.go`'s reply pricing reads the gate on the holder **as the board
      stands**, with a comment arguing the imprecision is safe because
      `passive.Condition` carries nothing but `BelowHealth` — so a gate can only
      turn *on* as its holder is hurt, and the miss falls in the direction every
      cap in that file errs in. **That reasoning dies the day this term lands**: a
      gate at the top of the bar turns **off** as its holder is hurt, so the same
      line would charge for a reply that will never be made — the one direction
      that file does not err in. The fix is one expression: read the gate against
      `holder.HP - dealt`, which is exactly where `answer` reads it, so the two
      agree exactly instead of approximately. No shipped trait carries a `while`
      and a `replies` at once (`blaze` and `last_gasp` gate; `venom_blood`,
      `thorns`, `ballast` and `static` reply), so nothing changes on today's data —
      but the *rule* would allow one, and this is the site that would be wrong.

- [x] `ENG-013` **A summon lives in the gap between the format and the team cap, and
      at five a side there is no gap.** Raised 2026-09-08 by reading the balance at
      five a side, which is the one thing `NET-001`'s last sub-item was waiting on.
      **Closed 2026-09-09 by splitting the constant, and five a side is open.**

      `hex.MaxTeamSize` was **five**, and it was doing two jobs: it was the widest
      team the board admits *and* it was the size of the largest format. So a full
      5v5 side computed `room = MaxTeamSize - 5 = 0` in `battle.summonPlaces`,
      `summonWorth` priced every summoning skill at nought, and `Suggest` never
      cast one. Measured, mirrored, 100 seeds: `split` cast **400** times at three
      a side, **400** at four, **0** at five, with copies arriving on the same
      counts. The cliff was exactly at the cap, and a probe at seven put the counts
      back, which is what said it was the cap and not the format.

      **The shape shipped is a fourth one, and it is the one none of the three
      considered was.** The three were: raise the cap (makes a seven- or nine-unit
      *roster* legal, and moves `composition`'s rung ceiling); make the cap a
      function of the format (a rule crossing the layer line, since
      `internal/core/battle` has no notion of a format on purpose); or declare
      summoning dead at 5v5 (a skill that works at one format and not another,
      which no screen has anywhere to say). The fourth is that **the constant was
      two constants all along**:

      - `hex.BoardSlots = FormationCols * FormationRows` — nine, what the board
        admits, derived from the grid rather than written out so it cannot drift
        from the formation it claims to be the whole of. Read by
        `battle.summonPlaces`, `battle.enlist` and two capacity hints.
      - `hex.MaxSquadSize = 5` — what a side is *fielded* with. Read by
        `placement.Squad.Validate`, the squad builder, the room gate, the draft
        and `composition`'s rung ceiling.

      `internal/core/battle` gains no notion of a format, so the layer line is not
      crossed; `hex` is the leaf all five readers already import, and
      `internal/wire`'s `TestTheLargestFormatIsTheSquadCap` holds the cap in step
      with the formats **from the layer that knows what a format is**.

      ⚠️ **The price, accepted out loud and pinned rather than commented:
      `battle.New` now admits a NINE-unit roster.** That is the "a board nobody has
      designed" cost, and what bounds it is that every path which *authors* or
      *accepts* a squad counts to `MaxSquadSize` instead — held by
      `TestANineUnitRosterIsLegalHereAndRefusedByEveryAuthoringPath`
      (`internal/core/battle`, the engine half and `Squad.Validate`),
      `TestANineUnitSquadReachesNoRoomAndNoDraft` (`internal/room`, the gate and
      the draft) and `TestTheBuilderStopsAtTheSquadCapAndNotAtTheBoard`
      (`internal/screen`). **Do not "tighten" `enlist`'s bound to the squad cap**:
      a summon reaches the board through the same `enlist` a roster does, so that
      would refuse every copy a full side calls up and rebuild this item one layer
      down.

      **What the re-measurement found** (→ `docs/balance.md` § *Five a side, read at
      last* for the tables): the original harness was never committed and its
      fixture is recorded nowhere, so the protocol was rebuilt over **three**
      independent fixtures instead of one. All three reproduce the *shape* — a
      cliff to nought at five and nowhere else — and none reproduces the recorded
      400, which says the finding was the engine's and the number was the squad's.
      At the split, 5v5 goes from **0** casts to 752 / 738 / 778, arrivals equal
      casts everywhere, the mirror is **500‰ exactly** and `Endless` is **0 of
      200** in all eighteen readings, and battles run about 2% longer.
      ⚠️ **3v3 does not move at all** — the predicted regression did not happen,
      because `split` gates itself on `sundered` below two stacks and that bound
      was already binding at three a side.

      ⚠️ **One golden moved and the estimate that said none would was wrong about
      which screens can see a summon.** `cmd/hexforge-tui/testdata/screens.golden`,
      the `spar` screen: `naruto.naruto`'s median duel is **60 turns where it was
      58**, and nothing else on the row moves. A spar is fought from the **cast**
      rather than from `roster.json`, and that character is the one shipped kit
      holding `shadow_clone` and `summon_toad`. Causation was proven by pinning
      `room` back to the squad cap and watching the golden go green.

      ⚠️ **Two follow-ups this leaves, neither of them done here:**

      - **`summonPlaces`'s two board bounds now agree by construction.**
        `BoardSlots` is `FormationCols*FormationRows` and `census` gives every
        counted unit a distinct cell, so `len(free)` is always
        `BoardSlots - perSide` and **no fixture can tell the two bounds apart any
        more**. `room` is kept because it is the sentence the rating prices off,
        but a mutation to either bound alone is now free. Whether to collapse them
        is a decision, not a cleanup.
      - **The battle screen's row budget gets worse.** `SCR`'s re-take found no
        board fits the floor already; a side that can now hold nine makes the
        roster block longer still. It is a floor-versus-screen problem the format
        does not create, and it is not a blocker here.

      **`DAT-002`'s rungs 4 and 5 are unblocked and deliberately NOT authored.**
      The rung ceiling is `MaxSquadSize`, numerically unchanged at five, so
      `bonuses.json` parses identically and no seed golden moved; rungs at four and
      five become reachable the moment a 5v5 is played. Decision 6 requires that
      they arrive **with their own measurements**, so authoring them is a separate
      item — unblocking them was this one's deliverable.

- [x] `DAT-005` **`reckless` is the dragon build's 22.1% — CLOSED. All four levers are
      measured, the lever that looked alive was killed by a better instrument, and
      what shipped is a guard rather than a number.**

      The three dials this item had already spent stand: the missing detonate is
      worth **−0.8**, dropping `bare`'s dodge clause **+2.8**, a vulnerability
      **−5.9**, and softening `bare`'s defence clears the duel floor only past a
      cliff two points of defence wide.

      ⚠️ **The fourth lever was built, measured and reverted.** The item's own
      conclusion asked for *a different kind of cost — one the duel prices and the
      cast-wide matchups do not*, and there is a reason no stat can be that: both
      gates a magnitude offers are the same event, a strike crossing a kill
      threshold. What is not a stat is **duration** — the duel is a grind and the
      cast-wide matchups are over while the holder is healthy — so
      `passive.Condition` was given an `AboveHealth` twin and `reckless` was gated
      at the top of the bar. Best rung **29.7%** against a floor of **36.9%**, and
      not even monotone. The term was reverted with the reading: a gate end no
      data declares is a field nothing reads.

      ⚠️ **A premise of the whole item was sixty-five points stale, and its own
      test had said so.** The cast-wide objection — that softening `bare` makes
      `reckless` the 100%-against-the-cast trait `blood_thirst` was refused for
      being — was measured against a Squirtle whose `withdraw` restored nothing,
      and `squirtle_test.go` carries the sentence *"Re-take it before quoting the
      amount."* Nobody did. Re-taken: squirtle against the dragon build **93.0% →
      28.4%**, `blood_thirst`'s cast-wide pair **100%/100% → 100%/88.0%**. The
      cast-wide gate is **no longer binding** — every softening stays under both
      referents — so the third lever was alive again.

      ⚠️ **And the ledger killed it properly.** Damage off the event log, which is
      the currency the trait is denominated in, says what no win rate could: from
      about −200 onwards `cost` goes **negative** — a unit holding `reckless`
      takes *less* than a unit holding no trait at all while dealing five times
      more, because `unleashed` ends the battle sooner and `burn` ticks fewer
      times. −250 costs 22632 and wins 25.1%; −200 costs **−1134** and wins 28.5%;
      the duel needs 36.9%. **The two gates do still flip together — they are just
      not the pair on record.** Duel against ledger, not duel against cast-wide,
      and every value that makes the matchup real makes the trait free.

      **Shipped:** `TestRecklessSpendsNoMoreThanItBuys` gains its lower bound. It
      had only *not a tax* and no *still a trade*, so it passed all eight gift
      values. Nought rather than a share, so nothing is invented; the shipped
      trait clears it by 50084. Mutated on disk, `bare` at −250 stays green and
      −200/−150 go red while `TestRecklessIsATradeAndNotAGift` stays green through
      all three — which is exactly why it was added: counting which stats a grant
      lowers cannot see how much.

      **Not shipped:** any change to `bare`. The data is unchanged at −400, the
      duel still reads 22.1%, and the floor in
      `TestTheDragonBuildIsASidegradeAndNotAnUpgrade` stays at 150. What is left
      is the acceptance this item has been circling for four levers: the 22% is a
      statement about `inferno` and belongs to the fire line's detonate.
      → `README.md` § *What gating `reckless` was worth* and § *What re-taking the
      whole thing on a working `withdraw` was worth*.
      → the `weigh`-shaped trait instrument is still missing and is still its own
      piece of work; this measured four more readings by hand.

- [x] `DAT-009` ⚠️ **Ally-aimed area support is single-target on every shipped board
      — DONE, REFUSED, with the measurement. The item's own cheap candidate was
      wrong, is corrected below rather than deleted, and the shape that does work
      was built, measured and thrown away.** Raised 2026-09-08 out of `DAT-007`,
      measured and closed the same day. **Nothing in `internal/seed/data` changed.**
      What shipped is this entry and one test.

      ⚠️ **`arc_up` catches ONE on `s01`, exactly as `column` does.** The item read
      *"moving one of the five onto `arc_up` would make the axis real on `s01`"*,
      and that sentence is false. `s01` stands `(2,0) (2,2) (0,1)` and **all three
      pairwise hex distances are 2**, so no shape whose splash is one step can
      catch two on it. `DAT-007`'s coverage table was right; what it did not say is
      that `arc_up`'s 2 was read on `s02`–`s05`, which field no ally-aimed skill.

      **The friendly-half table**, derived the same way `DAT-007`'s was — replaying
      `hex.Place` and `pattern.Pattern.targets` over `squads.json` /
      `patterns.json`, aiming only where `battle.aims` permits. Read *ally half /
      enemy half*:

      | shape | `s01` | `s02` | `s03` | `s04` | `s05` |
      |---|---|---|---|---|---|
      | `single` | 1 / 1 | 1 / 1 | 1 / 1 | 1 / 1 | 1 / 1 |
      | `flank_up` | 1 / 1 | 1 / 1 | 1 / 1 | 1 / 1 | 1 / 1 |
      | `flank_down` | 1 / 1 | 1 / 1 | 1 / 1 | 1 / 1 | 1 / 1 |
      | **`column`** (all 5 support skills) | **1 / 1** | 1 / 1 | 1 / 1 | 1 / 1 | 1 / 1 |
      | `wedge_right` | 1 / 1 | 2 / 2 | 2 / 2 | 2 / 2 | 2 / 2 |
      | `wedge_left` | 1 / 1 | 2 / 2 | 2 / 2 | 2 / 2 | 2 / 2 |
      | **`pierce`** (fielded by nobody) | **2 / 2** | 2 / 2 | 2 / 2 | 2 / 2 | 2 / 2 |
      | **`arc_up`** (the item's candidate) | **1 / 1** | 2 / 2 | 2 / 2 | 2 / 2 | 2 / 2 |
      | `arc_down` | 1 / 1 | 2 / 2 | 2 / 2 | 2 / 2 | 2 / 2 |

      **The friendly half is not a different board.** On all five squads it is
      identical to the hostile half, shape for shape. The 180° rotation only bites
      on `roster.json`, where `DAT-007` recorded the divergence (`arc_up` 2 against
      3, `wedge_right` 3 against 2) — and `roster.json` fields no ally-aimed skill
      at all.

      ⚠️ **`single` and `column` are the only two shapes in the book closed under
      that rotation.** `hex.Place` maps `up`↔`down`, `upper_right`↔`lower_left`,
      `lower_right`↔`upper_left`; every other shape is a *different* shape on the
      enemy half. That is very likely *why* all 14 ally-aimed skills sit on one of
      those two, and it is the thing to check before putting an ally-aimed skill on
      a new shape. It cancels on the five squads only because their formations are
      row-symmetric.

      ⚠️ **`splash_power` is not the friendly half's gate, so `DAT-007`'s framing
      does not transfer.** `resolveAgainst` scales only `power` at `position > 0`
      (`turn.go:1278`); a status application is not scaled and `restore` is. All
      five support skills declare `"power": 0` and deliver `applies`/`strips`, so a
      support skill catching two allies delivers **two full-strength buffs**, not
      one and a half. The rating already prices it that way: `pricing.rate` walks
      `covers` and sums `restored + granted + cleansed` per occupant with the
      `position` index deliberately unused (`price.go:136-179`) — no friendly path,
      no friendly discount.

      **What is fielded, which is the whole reason this is one measurement and not
      five.** Of the 14 ally-aimed skills, 9 are `single` and all 5 area ones are
      `column`:

      | skill | fielded in a squad | in a build | in a learnset |
      |---|---|---|---|
      | `rally` | **yes — `s01`/clefable, slot (0,1)** | `cleffa.mend`, `poliwag.chorus` | 9 characters |
      | `chorus` | no | `poliwag.chorus` | 1 |
      | `heal_bell` | no | none | 1 |
      | `safeguard` | no | none | 2 |
      | `slipstream` | no | none | 1 |

      **`rally` on `s01` is the only ally-aimed area skill any rate in this
      repository has ever been taken over.** The other four cannot be measured off
      a squad rate without first putting them on a board, and a placement change is
      the ~253‰ `DAT-007` priced on `s04` vs `s06`. They were not fielded to
      measure them.

      **The gate, written down before any subject arm ran.** Subject: `rally`
      `column` → `pierce`, coverage on `s01` 1 → 2 on both halves, nothing else
      touched. All four had to hold: (1) the aggregate over the included boards
      rises by **≥ 50‰** — two to three and a half sigma on a 400-battle binomial,
      deliberately far below `DAT-007`'s 150‰ because this changes one slot on one
      unit and only has to clear sampling; (2) **every included board moves the same
      way**; (3) **`AsAlly` and `AsEnemy` both move**, neither more than twice the
      other; (4) the null control reproduces the baseline exactly and every mirror
      reads exactly 500‰. Board inclusion, also written down first: baseline rate
      inside **50…950‰**, `Endless` under a fifth of the row. `s05` fell out at
      995‰; `s02`, `s03` and `s04` are the gate.

      **Four arms, `forge.FightSquads`, 200 seeds each way, `s01` as home, endless
      beside every rate and the halves kept apart.** Every row is `endless 0 of
      400` except the mirror, which is `2 of 400` in all four arms.

      | arm | `s02` | `s03` | `s04` | `s05` (excluded) | mirror `s01` | aggregate |
      |---|---|---|---|---|---|---|
      | **baseline `column`** | 282‰ | 60‰ | 92‰ | 995‰ | **500‰** | **145‰** |
      | **`arc_up` null** | 282‰ | 60‰ | 92‰ | 995‰ | **500‰** | **145‰** |
      | **`pierce` subject** | 285‰ | 75‰ | **77‰** | 1000‰ | **500‰** | **145‰** |
      | `fury` ×2 calibration | 192‰ | 75‰ | 87‰ | 1000‰ | **500‰** | 118‰ |

      The aggregate is the three included boards pooled: baseline 174 wins of 1200,
      `pierce` **175** of 1200. **One battle in twelve hundred, against a floor of
      fifty parts per thousand.** `s01` vs `s04` reads 92‰, 37 / 363 / 0, median 48
      on the baseline — `DAT-007`'s published row, reproduced, which is what says
      this is `DAT-007`'s harness.

      Halves, subject against baseline: `s02` `AsAlly` 345 → 360 and `AsEnemy`
      220 → 210 — **opposite directions**; `s03` 60 → 75 on both; `s04` 100 → 85
      and 85 → 70. Median turns: `s02` 77 → 77, `s03` 61 → 63, `s04` 48 → 48, the
      mirror **78 → 104**.

      **Verdict: three of the four conditions fail.** (1) +0‰ against ≥ 50‰. (2)
      `s04` moves −15‰ while `s02` and `s03` move +3‰ and +15‰. (3) `s02`'s two
      halves move opposite ways. Only (4) holds, and it holds completely.

      ⚠️ **The `arc_up` arm reproduced the baseline BATTLE FOR BATTLE** — same
      rate, same W/L/D, same `Endless`, same median turns, same `AsAlly`/`AsEnemy`
      split, every row. That is the exact-null control the derivation predicted
      (`covers` skips empty cells, `pricing.rate` prices per occupant, and nothing
      outside `covers` reads the shape name during a battle) and it is what retires
      the item's own candidate: **the change `DAT-009` asked for is a no-op.**

      ⚠️ **The calibration arm is what stops this being "the instrument is
      blind".** It leaves `rally` on `column` and raises its own application from
      one `fury` stack to two — twice as much buff on one ally, no geometry at all
      — and the board **sees it plainly**: the aggregate moves **−27‰**, `s02`'s
      `AsEnemy` collapses **220‰ → 5‰**, and the mirror's median goes 78 → 111
      turns. So the harness can price what `rally` *delivers*; it cannot price a
      second *recipient*. That is `DAT-007`'s finding repeating on the friendly
      half, and it is the refusal. (The calibration arm moving **downward** is a
      second reading nobody asked for and its mechanism is **not** measured here —
      `fury` is +300‰ attack and stats saturate, so a second stack may simply be
      worth less than the turn; do not quote it as a balance number.) The
      calibration arm was discarded and never went near `skills.json`.

      **The census says the slot is played**, which is the half a rate cannot
      answer — `hexforge census cleffa.mend --against <squad> --seeds 40`, 80
      battles each, `rally` casts:

      | board | `column` (shipped) | `pierce` (scratch) |
      |---|---|---|
      | `s01` | 369 | 290 |
      | `s02` | 232 | 233 |
      | `s04` | 80 | 80 |

      Never silent, on either shape, unlike `dazzle` on `s04` in `DAT-007`. So "the
      rating under-values support, so the carrier never plays the slot" is refused
      by measurement as well.

      ⚠️ **`hexforge census` substitutes the subject into `squad.Units[0].Slot`**
      (`census.go:118`), and `s01`'s `Units[0]` is the **machamp at (2,0)**, not the
      clefable at (0,1). So a census `--against s01` is **not** the `s01` board: its
      counts are evidence about the slot's liveness and never about coverage.

      ⚠️ **Do not quote a `hexforge spar` number against any of this.** Spar cannot
      see support at all — `memory/hexarena-poliwag-bruiser.md`, where Politoed
      reads 14.2% because spar takes the four earliest-learned skills and the
      support half sits at 32/36.

      **What shipped instead of the data change.**
      `TestAnAllyAimedShapeIsCatchableFromItsOwnCarriersHalf`
      (`internal/seed/areaboard_test.go`), a sibling of
      `TestAShippedFormationIsCatchableByTheShapesItFields` rather than a widening
      of it, because it needs a different reduction: that one takes the **better**
      of the two halves, which is right for an enemy-aimed shape and wrong for an
      ally-aimed one, since `forge.FightSquads` fights both arrangements and a
      support shape is bounded by its **worse** half. It derives its carriers from
      `seed.Squads()` × `seed.Roster()` × `seed.SkillBook()` — never a written-down
      list — filters to `Target == Ally` with `MaxTargets() > 1`, and logs the
      per-half count with a pointer here, exactly as its neighbour logs `DAT-007`.
      Mutation-checked three ways on `skills.json`, each reverted by hand: `rally`
      → `pierce` moves the log 1/1 → **2/2**; `rally` → `arc_up` leaves it at
      **1/1**, which is what proves it is not reading the neighbour's
      max-over-formations number (that one reads 2 for `arc_up`, on `s02`); `rally`
      → `single` empties the derived set and the "measured nothing" guard goes red.
      ⚠️ Whether the two halves AGREE is **logged and not asserted**, deliberately:
      every shipped squad formation is symmetric under the 180° rotation, so no
      shape can tell the halves apart on any board a rate is read off, and an
      assertion whose branch no shipped data can exercise is a fixture hiding a
      branch. `mostOccupiedCellsCaught` gained one comment saying it is
      carrier-blind and takes the better half, pointing here.

      **What stays open, and it is not this item.** The four unfielded support
      columns are unmeasurable without a placement change, and a placement change
      costs about 250‰ (`DAT-007`'s `s04` vs `s06`; `docs/balance.md` reads ~213‰
      independently) — an order of magnitude more than anything the shape returned
      here. **Do not follow this up by re-slotting `s01`**: `DAT-007` already
      answered that the axis can only be redeemed by paying a defensive price twice
      its size. Flag, do not act.
      → `pierce` remains declared in `patterns.json` and carried by zero skills.
      That is now a measured fact rather than an oversight: on the boards this game
      ships, a second buffed ally is worth one battle in twelve hundred.
      → If a support shape is ever wanted for its own sake, the open question is
      whether it is authored as a NEW skill on `pierce` rather than by reshaping
      `rally` — `rally` is carried by nine learnsets and two builds and its clause
      reads *"vang dọc trận địa"*, along the line, which `pierce` is not. That is a
      character-identity call for the author, not a measurement.
- [x] `DOC-001` **`## Not done` had become a list of finished work — thirty
      entries against twelve — SWEPT, and the heading is held by a command now.**
      On 2026-09-13 `TODO.md` § *Not done* carried **30 `- [x]` entries against 12
      `- [ ]` ones**, and the thirty ran to **2,663 lines**. The file's own header
      had claimed since the 2026-09-05 sweep that *"`## Not done` is only what is
      not done"* and that finished work *"moved to `docs/decisions.md`"*. That
      claim was true for **eight days**.
      **The rule was what failed, not the discipline.** § *The codes* defined
      `done` as *"finished, entry kept in § Not done with its measurements"* — so
      the compliant way to close an item was to change `[ ]` to `[x]` where it sat,
      which is precisely what fills a `## Not done` with done work. Two documents
      disagreed inside one repository and the cheaper one won: ticking costs a
      character, moving costs a cut and paste of up to twenty kilobytes, and the
      difference is invisible to every reader who does not count.
      ⚠️ `memory/hexarena-battle-screen-budget.md` had already written down *"a
      `[x]` left in Not done is a mistake I have now made twice"*. It was thirty by
      the time anybody counted, because that note recorded as a slip the thing the
      status vocabulary was prescribing.
      **What shipped.** The thirty entries moved here **verbatim**, in the order
      they sat in under `## Not done`. The move was mechanical and checked as one:
      every entry was extracted from `git show HEAD:TODO.md` and from this file and
      compared by SHA-256 — **30/30 identical** — and the diff is deletion-only on
      `TODO.md` (0 added lines) and addition-only here (0 removed lines).
      ⚠️ **`## Done` did NOT get a bullet per swept entry.** It stays what it is,
      prose paragraphs describing capabilities the game has, and a reader finds a
      finished item through its index row in § *The codes*, which already carries a
      one-line summary. A third wording of each summary would be the *"two callers
      wording one choice"* mistake this file's own header names.
      **`done` now means moved, and `shipped` was kept rather than collapsed into
      it.** `shipped` says the record is the § *Done* prose rather than an entry
      here — a fact about which file to open, not a quality ranking — and merging
      the two words would have made the status column stop answering that. ⚠️ One
      row does not fit the definition: `CAST-003` reads `shipped` and its paragraph
      is in this file. It was flagged in `TODO.md` and left alone rather than
      silently re-filed, because re-filing an item is a judgement about the item and
      an unannounced one is the same failure this entry exists to stop.
      **The claim is a command now, and it has to print nothing:**

      ```sh
      awk '/^## Not done/{f=1;next} /^## /{f=0} f && /^- \[x\]/' TODO.md
      ```

      ⚠️ **On the commit before this one it prints thirty lines** — pipe
      `git show HEAD~1:TODO.md` into the same `awk` and watch it fail. That half is
      the load-bearing half: a guard whose positive case is never shown is not
      evidence that it works, only a command somebody believes in. It exists as a
      command rather than as a paragraph for the reason the paragraph it replaced
      demonstrates — prose about a file cannot notice the file changing under it,
      and eight days of asserting the headings were honest changed nothing.
      **`DOC` is a new area**, documentation: this file, `TODO.md`, `docs/`,
      `CLAUDE.md`, `README.md` and `memory/`. The repository's records about itself
      had no area at all, so until now the sweep that repairs the records could not
      be filed under the rule it enforces.
      ⚠️ **The retired rule was written down in two further places and both are
      corrected here**, because a rule deleted in one file and left standing in
      another is how this defect returns:
      `memory/todo-items-are-addressed-by-code.md` stated it in its *How to apply*
      list and again in its closing status line. The same note also claimed the
      § *The codes* index table *"được sinh từ chính file chứ không gõ tay"* —
      **false.** There is no generator anywhere in the repository: no `scripts/`,
      no Makefile target, nothing matching `AREA-NNN` outside the documents
      themselves. The claim is corrected and a generator was deliberately **not**
      built, because a sweep is not the place to add a tool that would rewrite the
      file it is measuring.
      **Seven pointers named a moved entry by its section title and were
      re-pointed here, by code**: `README.md` (`ENG-006`), `docs/balance.md` and
      `internal/seed/guardboard_test.go` (`RAT-004`),
      `internal/seed/mender_test.go` (`RAT-003`), `internal/seed/diglett_test.go`
      and `internal/seed/battle_test.go` (`RAT-005`), and one inside this file
      (`DAT-003`, which it described as an *open* entry in `TODO.md` § *Not done*).
      Pointers that name a code were left exactly as they were — the index row in
      § *The codes* is what carries a reader from a code to wherever its entry now
      lives, and that is the whole reason the codes exist.
      ⚠️ **`README.md`'s *"`TODO.md` § *Not done* — twelve items, not one of them
      ticked"* needed no change, and it was FALSE before this sweep**: it read
      twelve where the section held forty-two, thirty of them ticked. It was
      describing the intended state rather than the actual one, and this change is
      what made it true. A pointer can be stale in the direction of being right.

      **Three more repairs, decided rather than deferred.** `CAST-003` was
      statused `shipped` while its paragraph sat in this file and not in § *Done*,
      which contradicted the status column's own definition the moment that column
      was written down; it is `done` now, because the word has to name the file a
      reader should open and the record is here. And **two pointers left over from
      the 2026-09-05 sweep were still naming `TODO.md` § *Not done* for entries
      that had left the file that day** — `CLAUDE.md`, for the wall-of-charges
      product (*The ninth narrow product had no board*), and
      `memory/hexarena-rating-gaps.md`'s own heading (*Four mechanics `Suggest`
      resolved and did not price*). ⚠️ **Both had been wrong for eight days and
      nothing noticed**, which is the same failure at the level of a reference
      rather than a heading: a pointer that names a SECTION is a bet that the
      section keeps its contents, and this file exists because that bet loses.
      Every pointer this sweep wrote names a code.

      ⚠️ **A fourth repair, and the only one that was not a pointer: `README.md`
      § *What softening `bare`'s defence was worth* claimed the duration-gate idea
      was "on the roadmap rather than in the *decided against* list", and neither
      half held.** The gate SHIPPED as `ENG-006` in the opposite polarity — a
      benefit that lapses, `pristine` on Magnezone — while the **cost** polarity
      that passage wants was filed nowhere at all; the nearest entry, `DAT-006`, is
      a refusal that names it as a possibility rather than a task. **It is filed
      now**, as `DAT-016`, in this same change: gating `reckless`'s `bare` grant on
      `While.AboveHealth` costs no Go, because `ENG-006` already shipped the
      per-grant gate it needs. ⚠️ Filing it is not evidence for it — the
      *duration* form of the same idea is measured and reverted, and a health gate
      is a different axis with no reading on it at all. A claim about
      where an item is filed is checkable against the index in one grep, which is
      the whole argument for keeping that index: this one went unchecked long
      enough to survive the item it described being finished.

- [x] `NET-003` ⚠️ **A peer's name reached the host's terminal as commands, and
      the room it came through needs no password.** `wire.Hello.Name` is the one
      free-text field in the protocol. `cmd/hexarena-host` cut it to 32 runes and
      stripped nothing, so whatever a stranger on the network typed was written
      to a terminal that reads an ESC as an instruction rather than a character.

      **What was measured.** 36 raw ESC bytes reached stdout. The payload that
      makes it a High is **OSC 52** — `ESC ] 52 ; c ; <base64> BEL`, 22 runes,
      comfortably inside the old allowance — which writes the reader's clipboard;
      they paste it into a shell some minutes later, themselves, believing it is
      what they copied. `OSC 0` renames the window and `CSI 2J` clears the screen,
      which together let a line the host has already read be replaced.

      ⚠️ **The bound was never a defence and the shape of that is worth keeping.**
      A length check refuses nothing when the whole attack fits inside the length,
      and `playerName`'s own doc said in as many words that "nothing in the
      transport checks it" — the gap was written down beside the code that needed
      it for as long as both existed.

      **What the attack costs.** Nothing. `-browse` advertises the room over mDNS
      (`_hexarena._tcp`), the listener binds every interface, and an empty
      password admits any hello past the version gate. The repository is public,
      so the client that sends a crafted `Hello` is this one with a string
      changed.

      ⚠️ **Fixing it at `Room.Join` alone would have left the bug exactly where
      it was, and the review that proposed it had the call site wrong.** The name
      on the host's screen does **not** come off the seat the room stored:
      `internal/socket` hands `Options.Joined` the hello's own field
      (`server.go`), and `room.peer.name` has **no reader at all**. So the fix is
      at the protocol — `wire.CleanName`, run by `Hello.UnmarshalJSON`, which
      every decoded hello passes through whichever door it came in by. `Room.Join`
      calls the **same function** for the other door, a hello built in Go that
      never met a decoder; that is one declaration used twice rather than a second
      rule.

      **Cleaned rather than refused**, because a refusal is a second way for a
      join to fail over something invisible — an old client sending a stray tab
      turned away from a room it belongs in — and because refusing tells the
      sender exactly which byte was noticed. The predicate is
      `unicode.IsControl`, which the repository already owned in
      `screen.PasteText`; that rule moved to `internal/plain` and `PasteText` is
      now one line over it, so a terminal and a text field cannot drift into two
      answers. ⚠️ `unicode.IsControl` is the complete answer rather than a lucky
      one: the **C1 controls** (U+0080–U+009F, U+009D being OSC itself) are
      category Cc, so one predicate covers both the seven-bit and the eight-bit
      spelling of an escape — a hand-written `== 0x1b` range does not, and the
      mutation proving that is in the matrix.

      **`internal/plain` imports the standard library only and nothing under
      `internal/core` imports it**, so the layer contract is untouched:
      `go list -deps ./internal/core/...` still names no third-party package.

- [x] `CLI-002` ⚠️ **`--replay` let a saved file write the verdict.** A log is a
      file somebody handed over — a host writes one per battle and tells both
      players to replay it — so an event's `name`, `note` and `target` are
      somebody else's writing, and they reached stdout verbatim. Measured: a
      crafted log rendered a **forged** `verified: re-running seed 11 reproduced
      all 255 events exactly` banner it had not earned.

      ⚠️ **The forgery needs no escape sequence at all, which is why the
      reordering is the half that matters.** `cmd/hexarena` prints that exact
      sentence after a successful `--verify`, and the `unverified:` notice — the
      only line saying nothing checked the file — was printed **last**, after the
      whole body and the summary. A log long enough scrolls it off the top of a
      terminal, so the file decided whether the notice was read. It is now printed
      **before** the body, which is the one position the file cannot reach. Held
      by an assertion on the byte **offset**, not on the substring: "the output
      contains the notice" was true before the fix too.

      ⚠️ **The finding's own field list was too short, and the miss is
      instructive.** It named `name`, `note` and `target` and judged `actor` safe
      "because it is matched against unit ids" — but nothing matches it here:
      `tui.Line`'s `tag` falls back to printing the **raw id** for anything the
      tags map does not hold, and `TagsFromLog` fills that map from `Started`
      records only. So an id on any other kind of event is printed as the log
      wrote it. Rather than audit nine fields against forty format strings and
      re-audit on every new branch, `tui.Line` cleans **all nine** at the top, and
      `TestEveryStringOnAnEventIsMadeInert` walks `battle.Event` by reflection so
      a tenth field is covered by having been declared. `Summary` is cleaned too —
      it reads a second copy of the names out of the log, and it is the part
      printed *after* the body, where a reader has stopped watching.

      ⚠️ **The cleaning is in the renderer and may NOT move into
      `battle.ParseLog`.** `internal/core` imports nothing outside the standard
      library, and an event the parser had edited would no longer equal the event
      `--verify` re-runs — a log with one tab in a note would then fail
      verification as a falsified record. The bytes are a rendering problem and
      are fixed where the rendering is.

      **No golden moved, and that is the check rather than a relief.**
      `plain.Text` returns its argument unchanged when there is nothing to take
      out, so every `testdata` file in `internal/tui` renders byte for byte as
      before — a security fix that silently moved the design record would be the
      worse outcome.

      ⚠️ **Still open, and not introduced here: `verify` itself has no test.**
      This change threaded a writer through it; nothing exercises the path where a
      log genuinely re-runs, so the real banner is unmeasured. The knowledge graph
      surfaced it and it is recorded here rather than quietly fixed, because it
      wants a replayable fixture rather than a line.
