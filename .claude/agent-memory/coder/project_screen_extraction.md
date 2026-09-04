---
name: screen-extraction
description: hexarena's reference-screen extraction is FINISHED — steps 1…6c landed and cmd/hexarena-tui is the second client; what is left in cmd/hexforge-tui is the character form, the check/spar and the fight; a client fixture cannot follow a screen across the package boundary, and the wording guard is per-package (three copies now)
metadata:
  type: project
---

hexarena is moving its reference screens out of `cmd/hexforge-tui` into
`internal/screen` in six PRs, so `cmd/hexarena` (the PvP client) can draw the
same ones. Landed so far:

- **step 1** (#199, `refactor/a-context-two-clients-can-share`) — the palette and
  the drawing helpers; measured **byte-for-byte neutral** after the fact.
- **step 1b** (#201) — `cmd/hexforge-tui/testdata/screens.golden`, the net every
  later step is read against.
- **step 2** (#203) — navigation. Six screens return a `screen.Action`
  (`Stay`/`Back`/`Quit`/`Raise` + `Target` + `Focus`) and the client owns Back in
  a one-slot `model.raisedFrom`.
- **step 3** (#205, `refactor/six-listings-leave-the-authoring-tool`) — the **move**:
  `chart`, `elements`, `statuses`, `species`, `builds`, `passives` are
  `draw.ChartScreen` … `draw.PassivesScreen` in `internal/screen`, `view(m model)`
  became `View(c draw.Context)`, and `Clamp`/`Window`/`Marked`/`TraitIndent` went
  with them (the client keeps one-line forwarders, the house pattern `pad`/`clip`
  already used). Golden unmoved.

- **step 3b** (`test/the-moved-screens-get-their-own-golden`) — the moved
  screens get a golden **in the package that owns them**:
  `internal/screen/testdata/screens.golden`, plus `./internal/screen` in
  `make golden` and in the four `-update` sites in `CLAUDE.md`. It closed a
  measured gap — a column width in a moved `View` was held by the *client's*
  golden alone. See [[two-screen-goldens]] for what each of the two catches.

- **step 4a** (`refactor/a-describer-is-handed-its-subject`) — the two
  **describers** stop reaching. `blurb` and `preview` described whatever raised
  them by reading `m.browse` / `m.skills` / `m.play`; the raiser now pushes a
  `screen.Subject` (a tagged struct: `Kind`, `ID`, `Level`, `At`/`Of`) and both
  answer `View(c draw.Context)`. `Action.Focus` is gone — it was the same
  mechanism with one undeclared case — and `SubjectKindCount` mirrors
  `TargetCount` so the client's applier map (`cmd/hexforge-tui/describe.go`) can
  be proved total. Targets `Blurb` and `Preview` are declared ahead of their
  raisers. Both goldens unmoved; `everyScreen` grew exactly 3 lines.
  ⚠️ **Four raise sites collapsed to THREE subject kinds, measured not assumed.**
  A listed skill and a battle option are one subject (same id, same position, same
  paragraph — `At`/`Of` carry the only difference), and the traits blurb and the
  art preview are two describers of one subject (a character at a level), so the
  applier for that kind writes **both** screens.
  ⚠️ **A describer that must not read three screens cannot keep its own
  `update`.** The raiser-walking (arrows moving the listing behind) moved into a
  new client file; only `View` and its helpers stayed. `from screen` stays on the
  blurb but is read by the client alone now.

- **step 4b** (`refactor/the-describers-follow-the-screens`) — the **move**:
  `blurb.go` and `preview.go` are `draw.BlurbScreen` / `draw.PreviewScreen`, with
  `SkillLines`, `TraitSentences` and `TraitRoom` exported for the kit picker and
  `Ramp` for the client's own picture-row test. `describe.go` did **not** move —
  it is the applier and the raiser-walking, and it writes the model.

- **step 5a** (`refactor/the-cast-browser-follows`) — the **cast browser**:
  `browse.go` is `draw.BrowseScreen`, a **plain alias**. Its whole client
  coupling was three `m.screen` writes, and all three became actions.
  `budgetLine` moved with it as `draw.BudgetLine`; `m.wrappedIn` was the one
  forwarder left dead and was deleted.
  ⚠️ **The first raise of a describer through `navigate` needed two client-side
  fixes** — `model.raise` fills `blurbScreen.from` for `target == screenBlurb`,
  and both describers clear `raisedFrom` in their own `esc`.

- **step 5b** (`refactor/the-guard-stops-holding-a-closure`) — `model.ask`'s
  `confirm func(model) model` becomes data: `guardState{question, asked, about}`,
  a `Confirmed(draw.Context, guardSubject)` per asking screen, and a
  `confirmedBy map[screen]…` held total by a declared `guardAskers` list.
  ⚠️ **`draw.Subject` does NOT fit as the carrier** — its Kind is counted and the
  raise applier is held total over that count. A client-local `guardSubject` is
  the answer.
  ⚠️ **Only ONE behaviour test answered `y` before this.** A count test proves
  presence, not effect.

- **step 5c** (`refactor/the-picker-answers-with-data`) — `pickState.apply`'s ten
  closures become **one `into pickDest` field**, a dense enum walked by
  `TestEveryPickDestinationLandsSomewhere`.
  ⚠️ **Keyed by DESTINATION, not by screen.** A destination is what can go
  unhandled.
  ⚠️ **A count beats a declared list**: a constant added above the count enters
  the walk for free.

- **step 5d-i** (`refactor/the-picker-follows`) — the **picker move**.
  `picker.go` is `internal/screen/picker.go`: `draw.PickState` / `PickOption` /
  `PickAnswer` / `PickKind` and friends. The client kept `pick.go`.
  ⚠️ **`(itself, draw.Action)` does NOT fit and a result type was needed** —
  `Update` answers `(*PickState, PickResult{Answered, Into, Answer, Cmd})`.
  ⚠️ **`Into any` is the whole trick.**
  ⚠️ **A moved screen forces exporting whatever the client's fixtures touch —
  including a *method*.**

- **step 5d-ii** (`refactor/the-skill-listing-follows`) — the **skill listing and
  its form**. `skills.go` is `internal/screen/skills.go` as `draw.SkillsScreen`
  (a **plain alias** in the client), 1571 lines. Also moved: `savekey.go` →
  `internal/screen/savekey.go` (`IsSaveKey` / `SaveKeyLabel`, client keeps two
  forwarders because the character and origins forms still ask), `choiceFormat` →
  `draw.ChoiceFormat`, `fieldValueRoom` → `draw.FieldValueRoom`. Package golden
  **+828 lines, additions only** (96 renders / 2401 lines); client golden
  **unmoved**; `language_test.go` +53/−120, all field-case plus two tests moved
  out. `model.continued` was the one forwarder left dead and was deleted.
  ⚠️ **`Action` grew two Kinds and two fields**: `Ask{Question i18n.Key}` and
  `Pick{Picker *PickState}`, with `KindCount` + `Kind.String()`. An Ask is **not**
  a Raise (nothing is navigated to; the question is drawn over what is in front
  and there is no Target to invent) and a Pick is **not** one either (a picker is
  not a screen a client names, it is a value the screen built).
  ⚠️ **A tea.Cmd may NOT go on `Action`** — a func field makes it non-comparable,
  and its own doc says comparability is why `Subject` is a tagged struct rather
  than an interface. So `SkillsScreen.Update` has **three returns**
  `(SkillsScreen, Action, tea.Cmd)` and the client has `navigateWith`. A result
  type of exactly `{Action, Cmd}` is a tuple with a name; `PickResult` earned its
  type by carrying four things.
  ⚠️ **Six of the ten pick destinations FOLLOWED the screen** as
  `draw.SkillsPick` (+`SkillsPickCount`), because `Into any`'s own reason — "a
  destination names a field of a *client's* screen" — stopped applying the moment
  the skill form stopped being a client's. `pickedInto` is `map[any]pickLanding`
  now, all three `Picked` methods take `into any`, and the totality walk walks
  **both** counts. The client is the one thing that knows both vocabularies.
  ⚠️ **`skillsScreen.inputs` really is shared between copies** and was moved
  unchanged; `TestTheSkillFormsFieldsAreSharedBetweenCopies` pins it in three
  directions (Inputs shared + same address, `Field` not shared, `ResetForm` hands
  back fresh storage).
  ⚠️ **`OpenAllowlist`/`OpenStatuses` had to be exported** even though production
  calls them from inside the package: `everyScreen` registers three pickers and a
  client fixture cannot follow a screen across the boundary. Same for
  `SkillLabelWidth`, `DamageRow`, `FieldValue`, `SkillFieldLabel`,
  `SkillFieldHelp`, `IndexOf`, `FilterLimit` and every `SkillField*`.
  ⚠️ **`NewSkillsScreen`/`Refresh`/`ResetForm`/`Prefill` take a `Context`, not a
  library** — the form dresses text fields and `NewInput` needs `Palette.Plain`.
  `newModel` builds a Context locally to do it.
  ⚠️ **Space on the "on itself" status field writes the *inflicts* field.** That
  is pre-existing behaviour, preserved deliberately; the move is not the change
  that gets to decide whether it is right.

- **step 5d-iii** (`refactor/the-squad-builder-follows`) — the **squad builder**,
  the last of the group. `squads.go` (1175 lines) is `internal/screen/squads.go`
  as `draw.SquadsScreen` (a **plain alias** in the client). `offeredCharacters`,
  `formationSlots` → `draw.FormationSlots`, `rankOf`, `rankNames` and
  `m.rankLabel` → `draw.RankLabel(c, slot)` went with it; `squadOptions` was
  deleted in favour of `draw.IDOptions`. Package golden **+492 lines, additions
  only** (124 renders / 2921 lines), client golden **unmoved**,
  `language_test.go` **+36/−36** and every line of it mechanical.
  ⚠️ **`Action` grew `About any` and `Target` gained `Fight`.** `About` is the
  opaque carrier an Ask needs — the builder asks **two** questions and the delete
  needs the id under the catalogue's cursor — declared as
  `draw.SquadsAsk{Kind, ID}` beside the screen, exactly the `PickState.Into`
  idiom. The client's `guardSubject` enum was **deleted**: `guardState.about` and
  all five `Confirmed` methods take `any` now.
  ⚠️ **`Target` may name a screen that is NOT moving**, and that is what a Target
  is for: the fight stays in the client, and the builder's `f` returns
  `Raise{Target: Fight}`.
  ⚠️ **`SubjectKind` grew a FIFTH kind, `SquadSubject`, and that is not the
  mistake 5b refused.** 5b refused putting a *guard* subject in `Subject`,
  because no raise carries one; the fight raise **is** a raise, so naming the
  squad by **id** in a Subject is the idiom working. The client's `landSquad`
  turns the id into the row index `fightScreen.home` keeps, and declines when the
  catalogue has no such id.
  ⚠️ **`raise` had to grow one special case**: `screenFight` goes through
  `model.enter` (its run cache must be dropped) while the other four do not (a
  refresh would put back the cursor `applySubject` just moved).
  ⚠️ **`pickDest` did NOT collapse.** The prediction that the builder's two were
  the last besides `pickNowhere` was wrong: the **character form** still holds
  `pickIntoKit` / `pickIntoSpecies`, so `pickedInto` is keyed by `any` over
  **three** vocabularies now (`pickDest`, `draw.SkillsPick`, `draw.SquadsPick`)
  and the totality walk walks three counts.
  ⚠️ **Four forwarders went dead and were deleted**: `numberField`, `numberKey`
  (last caller was the level field), `window` and `traitSentences` (one caller
  each, and it was this screen). `skillLines` and `traitIndent` had lost their
  production callers back in 5d-i and were being kept alive by tests alone — also
  deleted, with the four test sites pointed at `draw.SkillLines` /
  `draw.TraitIndent`.

- **step 6a** (#225) — the **works catalogue**, `internal/screen/origins.go`.

- **step 6b** (`refactor/the-played-battle-follows`) — the **played battle**,
  the last big one. `play.go` (1244 lines) is `internal/screen/play.go` as
  `draw.PlayScreen` (a **plain alias** in the client). **Two returns sufficed**
  (`Update(c, msg) (PlayScreen, Action)`) — verified rather than assumed: no
  `textinput` anywhere in the file, so there is no blink command to carry.
  Package golden **+754, additions only** (164 renders / 3851 lines), client
  golden **unmoved**, `language_test.go` **+8/−8** and every line mechanical.
  ⚠️ **The cross-screen read was the design work.** `m.fight.sides(m)` was read
  twice — to field the roster and to name the log file — reaching into a screen
  that is *not moving*. Both are "which two squads is this battle between", a
  parameter of opening, so `Open(c, home, away placement.Squad)` takes them and
  the screen keeps both (`Home`, `Away`); the client answers once in
  `enter(screenPlay)` and drops `sides`'s bool, because an empty catalogue hands
  over two squads with nobody in them and `Open` refuses exactly that.
  ⚠️ **`blurbScreen.from` is GONE** — the debt #212/#220/#223 each deferred.
  All three raisers return `Raise{Blurb}` through `navigate` now, and its three
  reads became `m.raisedFrom`. `Subject.Kind` could not have served: #207
  collapsed a listed skill and a battle option into one `SkillSubject`.
  ⚠️⚠️ **One slot was NOT enough and that is the trap of this step.** `raise`
  overwrites `raisedFrom`, so the moment play's `?` went through `navigate` the
  battle lost its own way back: read a description, esc, esc, and you land on the
  **menu** instead of the fight. Play is the first screen in this client that is
  **both raised and a raiser**. Fixed with a second field, `raisedOver` (a
  depth-2 stack: `raisedBy` pushes, `goBack` pops and promotes) — chosen over a
  `[]screen` because it left **all twelve** existing `raisedFrom` test sites and
  every `everyScreen` entry untouched. The fight's `p` is what pushes
  `screenFight`, because the raiser is what records the door.
  ⚠️⚠️ **The fix shipped with NOTHING holding it and that was caught in review.**
  Every existing way-back test walks a chain **one** raise deep, which a single
  slot answers perfectly — `TestTheFightRaisesABattleYouPlay` presses
  `fight → p → esc` and passes with the push collapsed to `m.raisedFrom = from`.
  The defect needs the raise **in between**: `fight → p → ? → esc → esc`. Closed
  by `TestAWayBackSurvivesTheScreenItRaised`, which is the only thing in the
  client suite that reddens under that collapse. **A depth-2 field needs a
  depth-2 test; a hand trace in a doc comment is not a net.**
  ⚠️ **And two slots are enough for TODAY, not by design — for a subtler reason
  than "no chain is three deep".** The chain *is* three pushes
  (catalogue → fight → battle → description), so the third displaces the
  catalogue; it is sound only because `fight.update`'s esc writes `screenSquads`
  **itself** rather than following a way back, so the lost entry is never read.
  Convert that esc to a `draw.Back`, or give an already-two-deep screen a raise,
  and two slots answer with the wrong door in silence. The test's last leg walks
  out to the catalogue so that day is a failure rather than a wrong `esc`; a real
  `[]screen` is the answer then.
  ⚠️ **`i18n.NoteWrote`/`NoteBattleVerify` on this screen measured NOUGHT hits
  in both goldens** (8201 + 3098 lines, both languages). The client cannot draw a
  saved battle — its fixture writes into a temp dir, so the note's path is not
  stable between two runs — but the package can, because the note is a **value**
  with a relative path. `a saved battle` and `a battle with no pairing` are the
  two new entries neither sweep could reach.
  ⚠️ **The option cursor's marker was held by the client and both goldens and by
  nothing in the package** — measured under mutation (`index == p.Option+1`).
  Closed with an assertion inside the ported row test.
  ⚠️ **Fixture rule, documented in the file:** every state builds its **own**
  battle. `PlayScreen` holds a `*battle.Battle`, the third field on which "a
  screen is a value" is false (after `Inputs []textinput.Model` and
  `Typed *textinput.Model`).
  **Split**: 24 tests moved to `internal/screen/play_test.go` (the budget
  arithmetic, the drop order, the roster clip, the notice, the whole log-frame
  block, the option rows, the turn/undo/seed/aiming/ending, the footers) plus 1
  new (the `?` Action's Subject); 9 stayed in the client (the frame's
  `Truncated`, the option list ending on the framed footer, the raise *landing*,
  three filesystem tests, the clipped row, which needs `appendSkills`).

- **step 6c** (`feat/a-second-client-draws-the-same-screens`) — **the second
  client**: `cmd/hexarena-tui`, a menu, the seven catalogues
  (cast/skills/elements/traits/species/works/squads) and a battle, plus
  statuses/chart/blurb/preview reached by a keystroke. 5 production files
  (`main.go`, `style.go`, `model.go`, `subject.go`, `pairing.go`), 6 test files,
  its own golden (**96 renders, 3936 lines**), its own wording walker (the
  **third** copy).
  ⚠️ **The binary is `cmd/hexarena-tui`, NOT `cmd/hexarena`** — which TODO.md's
  Groundwork item said for months. `cmd/hexarena` is the verification contract
  (`--replay --verify`, `--auto`, `--log`) and is untouched; the pair follows the
  `hexforge` / `hexforge-tui` house rule.
  ⚠️ **The three shared screens that author are answered by ONE capability**:
  `screen.Context.Authoring`, consulted beside the keystroke it turns off, with
  `Context.Footer(authoring, reading)` for the footer half and a **second
  wording** per footer (deleting a clause out of a rendered line leaves the
  separators). **Nought is the read-only reading** so a forgotten declaration
  loses keys rather than gaining writes.
  ⚠️ **`draw.Fight` is the PvP seam** and `cmd/hexarena-tui/pairing.go` is the
  one function that changes when a server arrives: home is the squad the raise
  named, away is *the next side on the file, wrapping* (a copy of itself when the
  catalogue holds one). See [[fixture-decides-what-is-visible]] for why a mirror
  fixture could not have measured it.
  ⚠️ **`screenCount` + a `notSwept` map is the new client's answer to TODO.md's
  five slipped screens** — a walk over the enum, not over the sweep map, so a
  view added without an entry *or a written exclusion* is red. The art preview is
  the one exclusion (rasterised art); `BuildsScreen` is drawn by this client at
  all and is filed as a gap.
  ⚠️ **Ask and Pick are unreachable here** (both are the authoring half of the
  vocabulary), so `navigate` does nothing with them and
  `TestNoScreenInThisClientAsksOrPicks` presses every key on all three
  author-capable screens to say so.
  **No golden moved**, neither existing one, and `internal/core`/`cmd/hexarena`
  were not touched.

**The move is finished.** What is left in `cmd/hexforge-tui` is `form` (the
character form), `savekey.go` (two forwarders), `check`, `spar`, `fight`,
`describe.go` (the applier), `pick.go`, `style.go` and `model.go`. The fight is
deliberately staying: it is what hands the two squads in.

What step 6c inherited and paid (kept because the next client-side change
touches the same seams):

- `TargetCount` and `SubjectKindCount` are held **total per client**, so
  `draw.Fight` and `draw.SquadSubject` are a bill the second client inherits: it
  will need a `raiseTargets` entry for the fight and an applier for the squad, or
  a reason to widen the walk instead.
- The squad builder is the one screen in the package that **writes** the author's
  own file, through `internal/forge`. Its golden entries and all of its package
  tests are built as **values** (`Saved` is a field, `Open` takes its baseline off
  whatever it is handed), and nothing in `internal/screen` opens a file.
- `blurbScreen.from` is **closed** (6b). What replaced it is `model.raisedFrom`
  beside `raisedOver`; a *third* screen that is both raised and a raiser wants a
  real stack rather than a third field.
- `cmd/hexarena` owes: a `raiseTargets` entry for `draw.Fight`, appliers for
  `draw.SquadSubject` and `draw.SkillSubject`, and its own answer to what a
  battle's `esc` means. It also has to hand `PlayScreen.Open` two
  `placement.Squad` values — that is the whole of what the battle screen needs
  from a client, and it is why the fight staying put costs nothing.

Learned in 5c/5d and still true:

- ⚠️ **`m.picker` is a `*pickState` and the pointer IS the presence flag.**
- ⚠️ **`(*pickState).update` is a POINTER receiver** while every screen here is a
  value.

**Why:** `README.md` § PvP over a LAN and `TODO.md` § Groundwork.

**How to apply** — what bites in the remaining PR:

- ⚠️ **A moved screen holding a `screen`-typed field cannot be a bare alias, and
  the answer is to EMBED rather than to relocate the field** (`blurbScreen.from`).
  ⚠️ `from` is set as a **client special case** in `model.raise`, and it stays
  until the **third** raiser (`play`) is converted — `browse` went in #212,
  `skills` in 5d-ii.
- ⚠️ **A moved screen that reads the machine needs the answer carried in, and
  `Palette` is where it goes** (`Palette.Plain`).
- ⚠️ **The import must be aliased `draw`.**
- ⚠️ **A client fixture cannot follow a screen across the boundary, so moving a
  screen FORCES exporting whatever `everyScreen`'s hand-built states touch** —
  field-case, methods, and whole helpers. Budget for it and say so up front.
- ⚠️ **`TestNoScreenHoldsItsOwnWording` scans `os.ReadDir(".")`, so it is
  per-package and there are TWO copies now.** Prove it still bites with a planted
  sentence.
- ⚠️ **A moved screen now needs entries in TWO goldens.**
- **What each net catches, measured under mutation.** A column width in a moved
  `View` → **both** goldens. A filter matching one character short → two package
  tests **and** both goldens. A raise carrying the wrong row → one package test
  (the Subject) and **three** client tests (the landing). `esc` not returning
  `Back` → **the client alone**, nothing in `internal/screen`. A guard asked with
  the wrong question key → one test each side.
- **The split rule that worked:** a test asserting *where you land* or that a key
  reaches something is a client test and stays; a test asserting *what a screen
  draws* or *how its cursor moves* goes with the screen.
  ⚠️ **Neither net in `internal/screen` can see a raise carrying the wrong
  row** — the package can only assert the `Subject` an `Action` carries.
- ⚠️ **A moved "fits the smallest window" test cannot see the frame.** Those stay
  in the client (they read `m.screenContent()`).
- ⚠️ **`everyScreen` and the golden cannot test navigation at all.**

See [[screen-neutrality-capture]] for how to prove a step changed nothing.
