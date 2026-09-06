# The screens

The front-end subject matter: what `internal/screen` draws, how the two clients
frame it, what bubbletea v2 broke on the way in, and how a screen decides what to
give up when the window is too small. It moved out of `CLAUDE.md` § *The layer
rule* on 2026-09-07, whole, because it is **subject matter** rather than a rule
that binds every edit — the same split `docs/architecture.md` and
`docs/balance.md` were made by, and for the same measured reason: `CLAUDE.md` is
loaded in full at the start of every session, and this was 60KB of it.

⚠️ **The binding sentences did NOT move**, which is the point of the split rather
than a happy accident. `CLAUDE.md` § *The layer rule* still states the layer
contract, still says that `internal/forge` is the one part of the module allowed
to touch real files, and still says that **neither front-end may restate a rule,
the wording of a refusal included**. That last one is what the whole of this file
is downstream of. Nothing here was rewritten, shortened or dropped.

**Read this before touching `internal/screen`, `cmd/hexforge-tui` or
`cmd/hexarena-tui`.** For what a golden file records about these screens see
`docs/goldens.md`, and for the runtime pieces under them `docs/architecture.md`.

⚠️ **The `##` headings are new and the paragraphs under them are not.** The
block arrived as one run of bold-led paragraphs, which is unreadable at this
length, so a heading was put over each of the natural breaks — and the bold lead
it names was **kept**, because it is the sentence people quote and deleting it
would make "nothing was rewritten" false. The repetition is the price of that.

## What bubbletea v2 moved, and the five things it broke silently

The migration was made for one reason: **a terminal cannot deliver the Command
key over the classic escape sequences.** There is no encoding for it, and v1's
`tea.Key` carried only `Alt`, so ⌘S did not exist as far as a program was
concerned. The Kitty keyboard protocol does carry it, v2 parses that protocol,
and a Command key now arrives as `tea.ModSuper`. `internal/screen/savekey.go` is
the single declaration of which keystrokes save; all three forms ask `IsSaveKey`
rather than matching a string of their own. It moved there with the skill form,
and the origins form followed it — `cmd/hexforge-tui/savekey.go` is two one-line
forwarders now, because the character form has not moved and a copy on each side
of the package boundary would be the fourth spelling this exists to stop.

⚠️ **That does not make ⌘S universally available and nothing in this repo can.**
The terminal has to speak the protocol (kitty, Ghostty, WezTerm, foot, iTerm2
with CSI u on — Terminal.app never), it has to pass ⌘S through instead of opening
its own Save dialog, and on Linux a window manager may claim Super first. So
`ctrl+s` stays the binding that always works and `saveKeyLabel` keeps naming a
control-S on every platform. Do not "simplify" the footer to ⌘S alone on macOS.

⚠️ **The footer names `ctrl+s` and nothing else, on every platform.** Not an
oversight and not a downgrade of ⌘S, which still saves: ⌘ is East-Asian-Ambiguous
width, so `lipgloss.Width` counts one cell where many terminals draw two, and the
glyph is then drawn on top of the character after it — `⌘S` renders as two
overlapping characters. A program cannot detect which sort of terminal it has.
Spacing it apart needs a cell that is not there: the English character-form
footer is 73 cells without the label, the smallest window is 80, the last cell of
a row is left empty so writing it cannot wrap the line, and no ASCII spelling of
both keys fits in the six that leaves. ⌃/⇧/⌥ are ambiguous too, so they are not
the way out either. Guarded by `TestTheSaveLabelIsDrawableEverywhere` (label is
ASCII) and `TestEverySaveFooterFitsTheSmallestWindow` (the six cells). ⌘S is
announced in `MenuNote` instead, which `TestTheMenuFitsTheSmallestWindow` keeps
inside the window.

Four API changes matter, and three of them fail *quietly* rather than at compile
time — which is why they are written down:

- **`Model.View` returns `tea.View`, not `string`**, and the alternate screen is
  a field on it rather than `tea.WithAltScreen()` on the program. `model.View`
  wraps `model.screenContent`, which is the string the tests read.
- ⚠️ **A bare space stringifies as `"space"`, not `" "`.** `uv.Key.String`
  returns `Text` only when it is not a single space, so space falls through to
  `Keystroke()` and comes out named. Every `case " "` compiled fine and matched
  nothing.
- ⚠️ **Colour is the program's decision now, not the library's.** lipgloss v2
  writes escape codes unconditionally and the program downsamples for the
  terminal it is attached to, so a `textinput` on its own defaults keeps its
  colours under `NO_COLOR`. `newInput` / `newInputStyles` in `style.go` restore
  the palette's rule; under v1 the library detected the missing terminal and did
  it for free, which is exactly why it is named now.
  ⚠️ **And `plainTerminal` gets to decide it, so an unset `TERM` is "dumb" only
  away from Windows.** `TERM` is terminfo's convention: cmd.exe, PowerShell and
  Windows Terminal set none at all, so reading its absence as a dumb terminal
  drew **every native Windows terminal in plain text**, with no cursor in any
  text field — while macOS and Linux always set it and never reached the branch.
  The rule is `colorprofile`'s own, copied rather than guessed at
  (`isDumb := (!ok && runtime.GOOS != "windows") || term == dumbTerm`; it reports
  TrueColor for a Windows 10 build 14931 or later, and for any `WT_SESSION`), so
  the palette and the thing that writes the escape codes agree. `plainScreen`
  takes its three inputs as parameters because `runtime.GOOS` cannot be faked and
  both answers have to be assertable from either sort of machine
  (`TestAnUnsetTermIsOnlyDumbAwayFromWindows`).
  ⚠️ **Nothing in the suite could see it**, which is the part worth keeping: every
  other test here sets `NO_COLOR` and returns at the line above, so the branch was
  unreached on the machine it was written on and unreachable in CI. A rule that
  differs by platform needs its inputs handed in, not read.
- ⚠️ **A virtual cursor is drawn as reverse video**, which is an escape code, so
  `newInput` turns it off on a plain terminal. That is not a regression: v1's
  renderer stripped the attribute itself, so the plain path never had a cursor
  either.

Two mechanical ones: a key is `Code`/`Text`/`Mod` rather than `Type`/`Runes`
(see `numberKey`), and `textinput.Width` is now `SetWidth`.

## Two languages, one set of facts

**Two languages, one set of facts: `internal/i18n`.** `cmd/hexforge-tui` speaks
Vietnamese by default and English on `--lang en` / `HEXARENA_LANG=en` / `ctrl+l`;
`cmd/hexforge` stays English because scripts read it, and its output is a
contract pinned by `TestARefusalKeepsTheWordingTheCommandLinePrints`. That is why
`internal/forge` returns **values, not sentences**: `CheckCarry` gives a
`*CarryError` holding the affinity, the skill and the skill's element,
`PresetFacts` / `StageFacts` / `SaveNoteFacts` / `Report.Problems` give data, and
the string helpers (`DemandSummary`, `PresetSummary`, `StageSummary`,
`SaveNotes`) are thin wrappers **built on** those, so the command line's English
cannot drift from the facts. `internal/i18n` holds a `Lang` (parsed by name,
unknown is an error), a `Key` enum and one array per language; it may import
`internal/forge`, and **forge must never import it**.

Three rules hold that shape, each with a test: no user-visible literal may live
in `cmd/hexforge-tui`, **in `cmd/hexarena-tui`, or in `internal/screen`**
(`TestNoScreenHoldsItsOwnWording`
greps its own AST — ⚠️ it reads `os.ReadDir(".")`, so it is **per package** and
there are **three** copies of it, one in each; a package that grows a screen and no
walker silently stops being held to the two-language rule, and the golden cannot
stand in for it, because a literal moved out of a package renders identically).
⚠️ **A fourth rule joined them at step 5b and it is about IMPORTS rather than
wording**: `internal/screen` may not see `internal/wire`, `internal/draft`,
`internal/socket` or `internal/room`, which is what makes `i18n.Lang.Refusal` and
`i18n.Lang.Seat` take a *name* and what keeps the lobby's three screens in
`cmd/hexarena-tui`. It was stated in three doc comments and checked by nothing
until `TestNoScreenImportsTheProtocol`. Also:
every
key is worded in both languages and no key is orphaned, and every wording
measures one cell per letter — write Vietnamese **composed**, or a combining mark
measures zero and every fixed-width column on that screen drifts. What is
deliberately *not* translated: ids of every kind, the six stat labels
`hp atk def spd acc ddg` (they are the `--hp` flag names and the data files' own
keys — see `forge.ShortStat`), and diagnostics from `internal/core`, which get a
lead-in in the reader's language in front of the parser's own English.

The minimum window is **120x24**. It was 72 while the client was English only and
80 once Vietnamese arrived — that runs 20–30% longer for the same sentence and the
busiest footers landed just past 72 — and it is 120 because a third of the client
had been trimmed flat against the old ceiling. Measured over every screen in both
languages at 200x60, on the widest line the sweep constrains: **34 of the 92
screen/language pairs sat at 76–79 cells** of the 79 there were, 29 more at 70–75.
What is pinned is almost entirely **footers**, which are catalog wording and
therefore cannot be given room any other way — the floor is the only lever for
that class, which is why widening the data cells in #173 and #175 moved the count
by one (35 → 34). The stated cost is that a terminal narrower than 120 does not
draw this front-end at all; `hexforge` needs no room and does everything it does.
`TestEveryWordingFitsTheMinimumWidth` renders every screen in both languages and
holds every line inside the floor, minus one column so a full-width line cannot
wrap.

## Cutting a line to the window, and saying so

**`frame` cuts every line to the window, and the cut says so.** It clips rather
than wraps because a wrapped row pushes every row under it down by one, which is
how the footer leaves the bottom of the screen — that part is settled and must
not be reopened. What changed is that the cut **marks**: it used to be
`lipgloss.MaxWidth`, which cuts safely and **silently**, so twenty-three of the
twenty-four sites rendering `m.lang.Error(...)` lost the tail of a sentence with
nothing saying they had, and a truncated explanation that does not say it is
truncated is worse than one that does. It is the horizontal twin of `frame`'s own
`Truncated` marker, which has said so since it was written.

- **One cutting rule for the package: `clip`, in `model.go` beside `pad` and
  `labelAt`.** It was the picker's private helper for one refusal row; `frame`
  calls it on every line of every screen now, so it moved for the reason
  `fieldValueRoom` lives there. `viewTooSmall` cuts through it too.
- ⚠️ **It has to be escape-aware and marking at once, and neither of the two
  tools it replaced could do both.** `MaxWidth` steps over an escape sequence and
  cannot mark. `clip`'s old body appended the mark but sliced `[]rune`, which on
  a styled line peels the terminating `\x1b[m` off the end one rune at a time —
  measured, not argued: a bold red ten-cell line cut to nine came back
  `"\x1b[1;31mabcdefgh…"`, right width, right letters, **no reset**, colour
  bleeding down the rest of the screen. Every caller then passed unstyled text so
  nothing showed it, and `frame`'s lines are styled, so wiring the old body into
  `frame` would have shipped the bleed on the first cut header. It is
  `ansi.Truncate` (`github.com/charmbracelet/x/ansi`, already a direct dependency)
  now, which re-closes what the cut left open.
- ⚠️ **The mark is added only when the line is genuinely longer than the room** —
  a line that exactly fills the window comes back byte for byte unchanged. That is
  the whole off-by-one risk of marking: an ellipsis on a line that fitted claims a
  tail that was never there and spends a cell of content to claim it.
  `TestALineThatExactlyFillsTheWindowIsNotMarked` crosses that boundary rather
  than approaching it (the same header at `w` and at `w-1`) and is the **only**
  test in the package that catches the mutation.
- **A marked line is exactly as wide as the unmarked cut would have been**, so
  `frame`'s row arithmetic is untouched — the mark is a cell *of* the window, not
  one past it. Asserted against `MaxWidth` itself over every room from 1 up, so
  what is held is that the widths did not move.
- ⚠️ **What still reaches that cut is all text, and that was measured before
  marking every line was chosen.** At the 120 floor, over every screen and state
  `everyScreen` registers, in both languages: the header naming the library
  directory (122 cells), the check screen's summary line, which also names it, and
  the form's archetype row — a preset id and its whole kit — at **128 (vi) / 131
  (en)**. At 160 and at 200, **nothing**. A path, a sentence, a list of ids.
- ⚠️ **No drawing can reach it**, which is why a blanket mark is safe: `tui.Board`
  is **19** cells wide against a floor of 120, `tui.Roster` likewise, and the
  preview's art is `usableWidth() - 2` **by construction**. An ellipsis on the end
  of a sentence says a tail was taken off; an ellipsis on the end of ten rows of
  hex art says something nobody can act on. `frame` is handed a joined string and
  cannot tell one from the other, so the claim is that the case never arises, and
  `TestNoDrawingIsEverWideEnoughToBeMarked` is what says so the day a wider
  drawing is added — widening the preview to the full window turns it red.

⚠️ **Every width figure quoted below against "79" or "of the 79 there are" was
measured at the old floor and is kept as the reading it was**, not restated: they
are records of why a wording was trimmed, and the trim is still in the catalog.
The live budget is 119. Raising the floor also **loosens** that sweep — every
existing line now passes trivially — which is the promise changing rather than a
test going vacuous; what it does not loosen is the vertical side, because prose
wraps at the floor and screens budget rows around the wrap.
`TestEveryFloorWrappedBlockTakesTheRowsItTakes` pins those row counts so the next
floor move cannot change one in silence.

`TestTheFormProducesTheCharacterTheCommandLineProduces` is what holds this: the
same answers as flags and as keystrokes must resolve to the same
`cast.Character`. A full-screen program cannot run with stdin as a pipe, so
`cmd/hexforge` is not going away — it is what a script uses, and the TUI refuses
to start when stdout is not a terminal rather than painting escape codes into
one.

## Where a form beats a prompt

**Where a form beats a prompt: the kit, and a skill's damage.** Two things the
full-screen client does that the command line cannot, and both are `internal/forge`
answers rather than screen logic:

- The kit is a **multi-select over the skill book** (`internal/screen/picker.go`),
  not a typed list. Every skill is listed, including the ones this character may
  not take, each marked and captioned with who it *is* for — a hidden skill reads
  as a skill that does not exist. The availability of a row is
  `forge.CheckSkill`'s answer, the same value the write refuses on, so the mark
  and the refusal cannot disagree. Nineteen rows do not fit beside a form (the
  form is nineteen body lines of the twenty it has in a 120x24 window), so the
  list is a **sub-screen that scrolls**, and `(*draw.PickState).Room` counts what the screen
  spends — including the empty string a trailing newline leaves when `frame`
  splits the body, which was miscounted first time and truncated the list.
- The new-skill form shows **expected damage as the power is typed**, from
  `forge.Library.PreviewDamage`, which is `combat.Rules.Damage` against the
  attack ceiling and *half* the defence ceiling. Those two are not a tasteful
  guess: they are the pair `skills.golden`'s own damage column is measured from
  (800 and 400), so the figure before a write is the figure the golden shows
  after one. It truncates **per strike** rather than once over the total, as a
  battle does — three strikes of 600 are 615, not 617. The amplified figure beside
  it is the skill with **everything it asks for holding**: the target's
  `requires`, the caster's `self_requires`, and `self_gradient` at the bottom of
  the caster's health, composed through `combat.Swung` in the order the battle
  composes them. ⚠️ It read only `requires` until then, which showed `outrage` and
  `comeback` — the two skills whose whole design is a caster-side term — at their
  plain power. **The row always draws its whole reading — the two figures and the
  reference pair they are measured against — and that is now a bound rather than
  a hope.** It used to drop the pair when the line would not fit
  (`Lang.DamageWithin`, with `damageRowRoom` computing the room), and PR #177's
  floor of 120 made that branch unreachable at every window the program draws:
  the line is four numbers in fixed wording, two of them the stat ceilings (three
  digits, always) and two `int64` at nineteen digits, so its **arithmetic**
  ceiling is 89 cells in Vietnamese and 87 in English against a narrowest room of
  97 and 98. Both were deleted rather than kept as dead weight, because each way
  the branch could have been reached again already has a stronger *build-time*
  guard: a **wording** that grew is caught by
  `TestEveryWordingFitsTheMinimumWidth`, which measures this row (it is program
  wording around figures, so none of the free-text exemptions reach it); **figures**
  that grew are capped by the type; and the **floor** going back down is caught by
  `TestTheDamageRowKeepsItsReferencePairAtEveryWindow`, which derives both the
  ceiling and the room rather than writing either down and names the arithmetic
  when it fails. ⚠️ Deleting `damageRowRoom` is also what fixed its off-by-one:
  it spent `width - 2 - labelWidth - 1`, so the row could fill the window's last
  cell — the one column every other row leaves empty because a line filling it
  wraps on some terminals. The surviving test measures the room off the rendered
  row against `minWidth - 1`, which is where that cell is now accounted for.
- The **squad builder** (`internal/screen/squads.go`) is the one screen that
  writes the author's own file rather than the game's: every other file here is
  data somebody wrote for the game, and `squads.json` is a side built to be
  fought with. It **ships like the rest of them** — the `go:embed` copy means a
  squad saved here reaches a battle at the next build, so nothing treats a squad
  in that file as a mistake. ⚠️ This used to read "writes something the game does
  not ship", and `seed.Squads` had a test asserting the file held none; both were
  written before anybody had built one, and the test failed the day somebody did.
  Three modes in one screen (`squadList`/`squadEdit`/`squadUnit`), because they
  are one decision at three depths and splitting them would put the half-built
  squad somewhere two screens could reach.
  - A member is **character, level, form, cell, four skills, one trait** — the
    same six facts a roster entry carries. Everything under the character is read
    against it, so changing the character **empties the kit**; the form chooser
    offers the empty stage plus every form by name, which is what a **forking**
    line needs; the slot chooser **steps over** an occupied cell rather than
    letting the save refuse it later.
    ⚠️ **The empty stage is two different things, and the screen says which.** On
    a line that does not fork it is the furthest form the level reaches, worded
    `SquadFurthest` — unchanged, in the field and in the member's row in the
    squad. On a line that **forks** it names nothing: the level has two ends,
    `StageAt` refuses to choose, and the placement is not fieldable at all —
    `placement.Squad.Take`, which is the one call `SaveSquad` and `FightSquads`
    both make, refuses the member. So it is worded `SquadForkUnnamed` instead, and
    a line under the fields (`SquadForkArms`, prose, wrapped at the floor because
    it carries authored stage names) names the arms, the key that picks one, and
    the two costs of not picking: the save, and the **silently shortened
    loadout lists**. `SquadsScreen.Form` hands `cast.SkillsAt`/`PassivesAt` an
    empty form in that state, which holds no gate, so both pickers offer only what
    every arm learns — measured on the one shipped fork at level 60: **13 skills
    and 4 traits unnamed, against 14 and 4 as Poliwrath and 15 and 5 as
    Politoed**. A list cannot say why a row is not on it, so the screen has to.
    ⚠️ **This chooser is the prior art the three read-only views were given a
    smaller copy of.** They describe a character rather than field one, so their
    question is narrower — which grown form does this level resolve to — and their
    arms come from `cast.Character.FurthestAt` rather than `StagesAt`: one form on
    a line that does not fork, one per arm on a line that does. The choice lives on
    `BrowseScreen.Form`, is settled on every read by `screen.ChosenForm` (the
    cursor and the level both move under a chosen name), rides to the two
    describers on `Subject.Stage`, and is walked with `s`. ⚠️ **A line that does
    not fork draws no row and answers no key**, which is what keeps every other
    character's record byte for byte what it was.
  - **A character can be held back: `cast.Character.Hidden`, and the squad
    builder is the only SCREEN that reads it.** `"hidden": true` in `cast.json`,
    absent meaning offered, so the flag is written only where it is set. A hidden
    character still ships, still loads, still fights, and a squad or a roster
    naming one is as valid as any other. Naruto is the one shipped example.
    ⚠️ **This said "the ONLY thing that reads it" and "an authoring convenience
    an author flips back, not a design statement" until 2026-09-05, and
    `internal/draft` made both false.** `draft.NewPool` gates the ban-and-pick
    pool on the flag, so a held-back character cannot be banned, cannot be picked
    and cannot be fielded in a drafted match at all — which is a rule of the game
    rather than a convenience. What survives untouched is the half about the
    **engine**: nothing in `internal/core` and nothing in `battle` reads it, and
    a replay has no use for who an author was choosing between.
    ⚠️ There are now **two** callers filtering it and they filter *different*
    rules — plain Hidden for a draft, Hidden-minus-a-`keep` for this screen — so
    a `cast.Book.Offered()` accessor was **considered and refused**, not left
    open. `offeredCharacters`' own comment carries the argument; do not go and
    write the accessor its earlier wording predicted.
    - **It round-trips or it is deleted.** `hexforge new` rewrites the whole file
      on every append, so the field is on the parse shape (`characterFile`) as
      well as on `Character`, exactly as `Skill.MarshalJSON` builds the parse
      shape rather than carrying tags of its own.
      `TestWrittenCastIsStableAndReloads` authors a held-back character and is
      what catches the omission.
    - ⚠️ **`squadScreen.characters` stays the WHOLE cast** and the filter is
      `offeredCharacters`, asked at the two sites that *choose* — `addUnit` and
      `cycle`. Filtering the held slice instead looks equivalent and is not:
      `character()` looks a member's character up in it to read the forms, the
      learnset and the traits, so a squad on the file naming a since-hidden
      character would lose its forms and its kit picker would refuse to open,
      with the row still printing the id.
    - ⚠️ **The one already chosen stays offered**, keyed on
      `squadScreen.unitOpenedAs` — what the member named when it was **opened**,
      not what is chosen right now. Hidden means *not offered for a new choice*,
      never *taken away from a choice already made*: a chooser that dropped the
      row would step off it on the first arrow press and write somebody else into
      a member nobody asked to change, in the author's own saved file. Keying it
      on the live answer is the near miss — the list then changes shape while it
      is being walked, so `right` then `left` lands one row short and the
      character is unreachable for the rest of the edit. That was found by the
      round-trip assertion and by nothing else.
    - The screen says why a character nothing else offers is on the list
      (`i18n.SquadHeldBack`), and the state is registered in `everyScreen` as
      `a held-back member` — which **asserts it draws that line**, because a
      registered state that renders nothing passes every sweep.
    - ⚠️ **`pickCharacters` is deliberately NOT filtered** (see the comment on
      `characterOptions`). It answers *which characters is this skill kept for*,
      and hiding a row there would make an existing `restrict.characters` naming
      that character unauthorable — the field is a picker and nothing else writes
      it. `TestASkillRestrictionMayStillNameAHeldBackCharacter` is what stops the
      job being "finished". The cast browser, the builds screen, `hexforge list`,
      the spar and the roster are unfiltered for the same reason: none of them is
      choosing a side to fight with.
    - `cast.golden` prints one line for a held-back character and nothing for the
      rest — the record has to be able to show that a character was taken out of
      the authoring lists, and "offered" on every other row is noise.
  - ⚠️ **A squad carries no side.** `placement.Squad.Take(side, cast)` fields it
    as either half and **prefixes the unit ids with the side**, so a squad fought
    against a copy of itself has two halves a log can tell apart.
  - `Library.SaveSquad` **replaces** the squad of the same id rather than
    refusing it, which is the opposite of `SaveSkill` and deliberate: a skill
    already in the book is something units carry, while a squad is a working
    document whose whole edit loop is saving it again. It validates through
    `Take`, so nothing is written that could not be fielded.
- The **fight** (`cmd/hexforge-tui/fight.go`, `forge.Library.FightSquads`) is
  raised from the squad catalogue with `f`, the way the spar is raised from the
  check: that is where a squad is under a cursor. Home is read off that cursor,
  the opponent is this screen's own chooser, and the runs are cached by
  `home|away|seeds` because a value receiver throws away a field written while
  drawing.
  - ⚠️ **Both ways round is the measurement, not a refinement.** Roster order
    decides the turn-queue tie-break, so one arrangement reports the *first
    slot's* advantage as the squad's — a mirror read **58.8%** the last time one
    was measured without swapping. Both halves run the **same** seeds.
  - **A squad against itself is a control**: exactly 500 per mille, by
    construction. `TestASquadAgainstACopyOfItselfIsExactlyEven` is what breaks
    first if the swap stops cancelling. The halves are reported apart too,
    because their difference is what *standing on a side* is worth — 18 points on
    the fixture pairing.
  - ⚠️ **The slot template cancels a synergy, and it looks like an answer.**
    `TestAMenderEarnsItsSlotWhereASparCannotSeeIt` holds the two fixed carriers
    constant across **both** squads, which is right for pricing a slot and wrong
    for pricing a pairing: whatever the partner is worth to the third member, it
    is worth to the opponent's third member too, so it nets to nought. Measured —
    a sapper written to compound with the blighter read **−29‰** in that shape,
    and the event log showed the synergy plainly working on both sides at once
    (poison three deep and 338 applications with the blighter present, two deep
    and 155 without). Two nearby designs fail the same way for their own reasons:
    a **striker** as the control measures what the squad's other two slots were
    short of, and **two different partners** measures the partners. What isolates
    it is one skill — the same character in both squads, `poison_powder` against
    `leech_seed`, both powders of no power — which read **636‰**.
    → `internal/seed/sapper_test.go`, which carries the three failures in its own
    comment so the next attempt does not rediscover them.
  - ⚠️ A squad rate is **not** the roster's win rate, and the screen says so in
    prose under the figure. That line is wrapped against **minWidth**, not the
    window in hand, which is the prose half of the width rule:
    **prose wraps at the floor, a data cell spends the window.** `minWidth` is
    the width this program promises to draw in, not a ceiling on what it may
    spend, so a gloss or a list of ids takes `m.usableWidth()` — cutting one on
    a wide terminal throws away content for nothing. A *sentence* measured
    against the real terminal would have two shapes instead of one, leave
    `TestEveryWordingFitsTheMinimumWidth` nothing to hold, and run a paragraph
    across a hundred columns for a reader to lose their place in.
    `cmd/hexforge-tui/width_rule_test.go` holds **both** directions: widening
    prose is as much a failure as clipping data.
  - ⚠️ **That line used to cite the art chooser as its precedent, and no longer
    can.** `artRoom` clips a filesystem **path**, which is data, and it takes
    the window as of #173. The precedent for a sentence is this line and the
    save note in `play.go`; what holds them at the floor is
    `TestAWideWindowStillWrapsProseAtTheFloor`. The old argument — that
    widening would leave the width sweep nothing to hold — was answering the
    wrong half of a row: the sweep measures the **catalog's wording**, and the
    catalog parts of the art row are exactly what `artRoom` *subtracts*.
- The **played battle** (`internal/screen/play.go`) is raised from the fight
  with `p`: the same pairing, one battle, the opponent played by
  `battle.Suggest`. `↑/↓` a skill, `enter` takes it and asks *where* only when
  there is more than one cell, `?` describes the one under the cursor, `a` hands
  the turn to the engine, `p` passes, `u` undoes, `n` is another seed, and
  `[/]` scroll the log (`pgup`/`pgdown` do the same and are what the brackets
  alias) — see *the budget* for why that pair and why following the tail is a
  state rather than an offset.
  - **Every option carries a one-line summary beside its id**, from
    `i18n.Lang.SummariseSkill`, and `?` raises the full description of the one
    under the cursor. An id is a name rather than an answer — nothing in
    `venoshock` says it is the skill that doubles into a poison — so the list was
    a column of things to guess between.
    ⚠️ **This costs zero rows, and that is the whole reason it is a line beside
    each option rather than a block under the list.** The screen's body is **28
    lines** at a 1v1 and `frame` gives it `m.height - 2` less the two the header
    takes, so at the declared 120x24 minimum only twenty survive. A pane under the
    list would be a pane nobody in the smallest window ever sees. What used to
    follow from that — the option list cut with the `Truncated` marker — is fixed;
    see *the budget* below.
    ⚠️ **An unavailable option keeps its `Reason` and drops its summary.** The row
    has one slot and the two answer different questions: why it cannot be cast is
    the live question the moment the cursor steps over it, and what it does is one
    keystroke away. Do not "fix" this by drawing both — the second would be the
    half that got clipped.
    The id column is **measured over `p.pending.Options`**, not over the book, for
    the reason `menuLabelWidth` and every detail pane measure theirs: the widest
    shipped id is thirteen cells and this unit may be bringing four short ones.
    The row **clips and never wraps** (`MaxWidth`, against `minWidth - 1` rather
    than the window in hand, like the fight's caution and the trait sentences), so
    the clause order matters — reach and cooldown are last because the end is what
    goes.
    ⚠️ **`SummariseSkill` is a fourth describer, and the reason is not brevity.**
    A compact line cannot be `Describe` with the flavour dropped: in Vietnamese
    `describeOpening` builds `BlurbFlavoured` out of the authored clause **and**
    the damage figure together, so there is no seam to cut. English separates
    cleanly, because English has no flavour and falls back to the derived opening
    — and a rule that works in one language and not the other is not the rule. So
    it is a distinct composition, held to the other reading by
    `TestTheOneLineSummaryQuotesNoFigureTheDescriptionDoesNot`: every digit run
    the compact line prints must appear in `Describe`'s output for the same skill
    in the same language, over every shipped skill. One way only — the compact
    line leaves out accuracy, pierce and a critical chance on purpose, and
    **counts** a strip rather than enumerating it — `purify`'s three categories
    are 79 cells in Vietnamese before the aim and the cooldown, so an enumeration
    could only ever arrive trimmed, and the claim it makes is read off
    `status.Category.Harmful` per skill rather than assumed (5 of the 8 categories
    are harmful, so a cleanse may be called one and a dispel may not). The
    wordings differ deliberately; the numbers may not. Both readings sit one under
    the other in `describe.golden`, so a balance change has to move both.
  - `?` raises **`screenBlurb`**, which is now branched on three ways — the skill
    listing, the cast browser and this — through the **same** `skillLines` the
    listing draws, so what a player reads while choosing is the paragraph an
    author reads while tuning. ⚠️ **Nothing in that path touches the battle**: the
    option is read, the skill is looked up in the library, and `esc` puts
    `m.screen` back rather than going through `model.enter`, which would rebuild
    the battle from its seed and throw the played half away. It works while
    **aiming** too (the skill is chosen and the cell is not, so the question is
    still open) and does nothing with no prompt or no options.
  - ⚠️ **The width sweep could not see any of this, and the fixture is why.**
    `everyScreen` built one battle and copied the model three times for its three
    states — but `PlayScreen` holds a `*battle.Battle`, so playing the "over"
    state out to its end **stepped the battle the other two pointed at**. By the
    time anything drew them `p.fight.Finished()` was true, `view` returned at the
    game-over branch, and `PlayFooter`, `PlayAimFooter`, the option rows and the
    whole aim block were rendered by **nothing in the suite**. Both footers were
    over the window the entire time — **82 cells (vi) and 83 (en)** against the 79
    there are — and every width test passed. Each state enters the screen for
    itself now, and both footers were retrimmed to name `?` and fit: **77 (vi) and
    78 (en)**, with the word after `esc` dropped, which is what `BrowseFooter`
    already does at the same squeeze. No key was given up.
    `TestTheBattleFootersNameTheDescriptionKeyAndFit` holds the half a width sweep
    cannot: that the key the feature hangs on is still named after the next trim.
    This is the **second** fixture in this repository whose early return made the
    interesting branch unreachable; the first was `plainTerminal`, where every
    test set `NO_COLOR` and returned above the branch. That is the transferable
    part: **a fixture that reaches a screen's early exit measures the exit**, and
    a screen it never renders is a screen with no width and no translation test at
    all.
  - It draws with `internal/tui` — `Board`, `Roster`, `Order`, `Line` — rather
    than with drawings of its own: what is played here has to look like what the
    game client plays, and a second drawing of a battle is a second thing that
    can disagree about what happened.
  - **The budget: this screen cannot fit the window the tool declares, so it
    decides for itself what to give up.** The fit was not a hard problem, it was
    an impossible one, and the measurement is worth keeping rather than
    re-taking. At 120x24 `PlayBodyRoom` leaves the body **twenty** rows:

    | section | rows |
    |---|---:|
    | heading | 1 |
    | `tui.Board` | **10**, fixed |
    | `tui.Roster` | **1 + one a unit** |
    | `tui.Order` | 1 |
    | the log | `PlayLogWanted`, then **every row nobody else claimed** |
    | the option list | 1 + one an option |

    | squad | roster | heading + board + roster + order + options | vs 20 |
    |---|---:|---:|---:|
    | 1v1 | 3 | **20** | exactly, with no blank and no log |
    | 3v3 | 7 | **24** | over by 4 |
    | 5v5 | 11 | **28** | over by 8 |

    `hex.MaxTeamSize` is 5, so **28 rows is the floor for a legal squad** before a
    single blank or log line, and a **summon** puts units on the board past the
    five a squad brought — up to the nine formation slots a side, which is
    `board + roster = 29` on its own. No arrangement of these sections fits.
    ⚠️ **So the defect was never the height, it was where the cut landed.**
    `frame` cuts from the **bottom** and the option list was the last thing the
    body wrote, so the one thing a player has to see in order to act was the
    first thing thrown away. `playFit` reserves the **heading and the turn in
    front** — never dropped, never cut, because a battle screen that cannot show
    the moves is not a battle screen — and hands what is left to the rest in a
    stated order: the **save's own note** (the answer to a keystroke pressed a
    moment ago, naming the file; *not* reserved, because a pair of notes runs to
    four rows or more and reserving them could crowd out the list), then **`Roster`** clipped
    a row at a time (health and effects are what a turn is decided on, and it is
    the one section that compresses by degrees), then **`Board`** dropped whole
    (ten rows of drawing have no half, and what it says is recoverable — the aim
    list prints the occupant beside every cell), then **`Order`**, then the
    **log**. Measured on the fixture with the log filled to its eight rows, and
    the heights are the same in both languages because every one of these
    sections is a *count* rather than a sentence: the whole screen survives from
    **h ≥ 36** at a 1v1, **40** at a 3v3 and **44** at a 5v5; the log is gone at
    **29 / 33 / 37**, the order line at **27 / 31 / 35**, the board at
    **25 / 29 / 33**, and the roster is clipped only once the aim list is up as
    well — from **30** at a 5v5, **24** at a 3v3, and never at a 1v1, whose whole
    roster is three rows.
    ⚠️ **What disappears is not monotone in the height, and that is the priority
    working.** The board takes ten rows or none, so at the height where it still
    just fits it takes the rows the order line and the log would have had, and one
    row shorter it cannot fit at all and both come back. Only the *offering* order
    is monotone.
    ⚠️ **The screen says what it gave up**, in one dim line under the heading
    (`i18n.PlayHidden` and the five names beside it) — a screen silently missing
    its board reads as a broken screen. A **shorter log frame is not in it**: the
    log is a frame over a history that is nearly always longer than it, so two rows
    fewer is the section working, while no rows at all is not. *How much* of the
    history is off screen is a different statement and it is the one on the heading
    row.
    ⚠️ **No per-screen floor was introduced, and the reason is that `minHeight`
    already is one.** `screenContent` returns `m.tooSmall()` before any screen is
    drawn below 120x24, so this screen is never asked for a shorter window than it
    can degrade into: at 24 the body has twenty rows, the heading and a
    four-option list reserve seven and the notice one, and the twelve left hold a
    5-a-side roster whole. A second floor would be a second answer to a question
    the first one has already refused — and it would have to be per-screen inside
    `tooSmall`, which is asked in `key` as well as in `View`. The one branch this
    puts out of reach is the save note being dropped, which needs h ≤ 15; it is
    constructed deliberately in `TestTheSaveNoteOutranksTheBoard` and the comment
    there carries the arithmetic.
    ⚠️ **`PlayLogWanted` is a floor of intent and used to be a ceiling** (and
    before that it counted events, which is a third thing again — `tui.Line` opens
    a turn with a blank row of its own, so one event arrives as two rows and eight
    events measured **eleven** a few turns in). The ceiling was a defect on its
    own: `playFit` hands the log the remainder of the budget and the remainder was
    then clamped to eight, so between a window 24 rows tall and one 80 rows tall the body
    grew **20 → 42** rows and the log stood still. Measured on the fixture, 3 a
    side, mid-battle: **8 rows at h=24, h=40 and h=80 alike**. A tall terminal
    bought the history nothing.
    The log now asks for `PlayLogWanted` **first** and then takes every row still
    unspent, which is why the same fixture reads **0 / 6 / 46** rows at those three
    heights (1v1: 5 / 12 / 52; 5v5: 1 / 8 / 48) — and why *nothing above it moved*:
    ⚠️ **growing the log may only ever spend rows nobody else claimed**, because
    #162's order is save note → roster → board → order line → log and the log is
    last precisely for being history rather than state. The two-part answer is also
    what keeps "everything fits" a question with an answer: a window that gives the
    log its eight rows has nothing missing, and one that gives it forty is that
    same window with room to spare. **Re-measured after the change and every drop
    height is unchanged**: the whole screen survives from **36 / 40 / 44**, the log
    is gone at **29 / 33 / 37**, the order line at **27 / 31 / 35**, the board at
    **25 / 29 / 33**, the roster is clipped (aiming) from **30 / 24 / never**. A
    single one of those moving would be a change to the priority rather than a side
    effect, and `TestTheBattleScreenDropsInTheOrderItStates` is what says so.
    ⚠️ **A floor in the *priority* is not available and was not built.** Eight
    guaranteed rows would have to come off the roster, the board or the order line,
    which is every one of those heights moving. The constant is what the section
    *asks* for, never what it is owed.
    ⚠️ **The log is a frame over the whole history now, and it scrolls in place.**
    `p.events` always held every event (`collect` appends and never trims) and the
    view threw the rest away: 283 rows rendered, eight drawn, **275 unreachable by
    any means** — no key, and nothing on the screen saying a history existed.
    `PlayScreen.LogRows` renders all of it and `logFrame` is the window; `pgup`
    and `pgdown` walk it, which is the pair that already scrolls the trait
    description and the picker rather than a second vocabulary for one idea (`↑/↓`
    walk the options and could not be taken). They work **while aiming** and **on a
    finished battle**, which is why all three footers name a scroll key at all — the
    keys they name are now `[/]`, see below.
    ⚠️ **Following the tail is a STATE, not an offset value, because the tail
    moves.** This is the decision the whole feature hangs off. A reader is normally
    at the newest rows; store that as the offset which happens to be newest and
    every event arriving silently shifts what is under them. So `logOffset` counts
    from the **start** of the history — which is what a `145–160 / 283` reading
    means — and `logFollow` is carried **beside** it. Same rule the abandoned queue
    tie-break paid for: `Queue.Pending` answers 0 for a unit it never heard of and 0
    is *soonest*, so absence had to be declared rather than detected. A sentinel
    offset would be that mistake again, and it would read as working, because the
    sentinel is a legal offset on the turn it is written —
    `TestFollowingTheTailIsNotAnOffset` is the only test that can tell the two
    apart, and it appends an event **without a turn behind it** on purpose, since
    every real turn resets the frame and would make the test pass for the wrong
    reason. Scrolling back down to the bottom *asks to follow again* (and puts the
    offset back to nought, which is also an ordinary offset — the top — so nought is
    exactly as unusable as a sentinel).
    ⚠️ **Acting resets it**, in `record`, which is the one place every turn goes
    through: the player's, the engine's, the pass and the "let it pick". Somebody
    who scrolled back and then acted would be reading a frame from before their own
    decision. Undo and another seed reset it through `begin`.
    ⚠️ **Undo makes the history shorter**, so the offset is clamped against the
    current total **wherever it is read** and not only where it is written — `undo`
    rebuilds the battle from a cut script, so `p.events` is rebuilt too and an
    offset kept across it points past the end.
    ⚠️ **The position goes on the heading row**, not on a row of its own:
    `trận đấu  seed 1` is about seventeen cells of the seventy-nine, and a row of
    its own would cost exactly what #162 spent a whole PR proving this screen has
    not got. It is shown **whenever rows are hidden**, not only while scrolled back
    — the discoverability half of the report is that nothing said a history existed,
    and a reader who cannot see that there are 283 rows will not look for the key.
    It says nothing when the log is not drawn at all, because the notice above
    already names it as a section the window is too short for.
    ⚠️ **The footer had to be trimmed to name the new key, and no key was given
    up** — that is what was asked for. Measured, not counted (a hand-count of a
    candidate came back four cells over twice): the battle footer was **77 (vi) /
    78 (en)** of the 79 there are, and `pgdn/pgup` needs twelve. So the words after
    `↑/↓`, `enter` and `?` are dropped — the three keys whose meaning the screen
    itself shows, which is the same judgement `BrowseFooter` and this footer's own
    `esc` already took — and the battle footer is **74 / 74**, the aim footer
    **72 / 77** and the over footer **65 / 63**, with the word for scrolling kept on
    the two that had room for it. `TestTheBattleFootersNameTheDescriptionKeyAndFit`
    covers all three and logs each width.
    ⚠️ **The footers name `[/]` now, and the page keys still work — this is an
    ALIAS advertised in place of what it aliases, not a rebinding.** `[` is back
    and `]` is forward, at **all three** sites that scroll (this log, the trait
    description, the picker's reading pane), because a site aliased alone is
    exactly the second vocabulary the paragraph above refuses. The reason is a
    keyboard rather than a preference: a compact board has no PgUp and no PgDn,
    reaching them through a modifier or a layer, so a footer naming them was
    unreachable advice and the whole log below the frame was as unreachable as it
    was before #169. **Advertising both pairs does not fit** — `pgdn/pgup` is nine
    cells against the brackets' three, and the English aim footer would come to
    **86** of the 79 there are — which is why the wording is a replacement. The
    five reworded keys all came in under budget, measured with `%s` substituted
    (vi / en): battle **73 / 75**, aim **66 / 71**, over **59 / 57**,
    `PickerReadingFooter` **58 / 57**, `BlurbMore` **25 / 28**. Four of them
    dropped the full six cells; the battle footer spent five of them back on the
    verb (`[/] cuộn` / `[/] scroll`), because a bare pair of brackets is the one
    place the wording would have said less than what it replaced.
    ⚠️ **A key alias is the shape that ships dead**, so it is pressed rather than
    read: `TestABracketScrollsWhereverAPageKeyDoes` tables the three sites and both
    directions and demands the bracket and the page key land in the same state
    **after asserting the page key moved it** — two no-ops satisfy an equality.
    And the fixture's own vacuity is the half an assertion cannot see: a `key`
    helper sending `KeyPgUp` under the name `"["` passes that table completely,
    which is why `TestABracketIsTheKeystrokeItLooksLike` reads `keyPresses`
    itself. `[` is safe to take because it reaches no text field — the picker
    enters its reading pane before the typed field and `numberKey` admits only
    digits, the browse blurb has no input, and `isSaveKey` is asked ahead of the
    battle screen's switch — while `form.go` and
    `internal/screen/origins.go`, which do have fields, never handled a page key
    and are untouched.
    ⚠️ **`internal/tui` did not change and did not need a row-limited `Roster`.**
    Clipping the roster's *rows* is this screen splitting a drawing it was given;
    reformatting it would be the other thing. The old
    `TestTheBattleScreenIsNoTallerThanItAlreadyWas` tripwire is gone, replaced by
    the bound that is now true: the option list survives every window the tool
    draws, the aim list with it, and `frame`'s `Truncated` marker never appears on
    this screen at all.
  - **Undo is a shorter script replayed**, not an unwinding: the script is cut at
    the player's last decision and the battle rebuilt from the seed. The engine's
    turns are recorded too, because a half that was not written down would replay
    as a different battle.
  - ⚠️ **The only screen holding something the model does not copy.** Every other
    screen is a value; a `*battle.Battle` is a pointer, so a mutation reaches
    every copy of the model. The battle is stepped in `update` and **never**
    touched in `view`, which is what stops a redraw playing a turn.
  - **`ctrl+s` writes a `battle.Log`** through `Library.SaveBattleLog` into
    `<data>/battles/<home>-vs-<away>-seed<n>.json` — the pairing and the seed
    identify a battle, so saving twice overwrites rather than accumulating.
    Saveable mid-battle: a half-played battle replays as exactly that half.
    ⚠️ **`--verify` re-runs against the embedded copy**, not the directory being
    edited, so a log written after an unbuilt edit will not verify — which is
    what `NoteBattleVerify` says, and why it is a note of its own rather than the
    generic rebuild line.
    ⚠️ File names are built from author-typed squad ids and are made safe here
    (`fileToken`), not by tightening what an id may be: what a file name may hold
    is `forge`'s problem, and a data rule change deserves its own reason.
    ⚠️ `replaceFile` now `MkdirAll`s the target's folder and puts the temp file
    **in that folder** — a rename across folders is not an atomic swap. The test
    that proved a failed write leaves the old file alone had to change its
    mechanism: a missing folder is created now, so it fails on one under a *file*.
  - `take`/`skip` are two methods rather than one taking a `Decision`, so a
    decision with a skill and no aim — which the engine refuses and nothing here
    should be able to build — cannot be written at all.
- ⚠️ **One loadout rule, and it had quietly become two.** "Which four of the nine
  may this unit bring" existed as `seed.chooseFrom` *and* `cast.chosenFor`, both
  unexported, both worded slightly differently — and the builder needed it a
  third time. It is **`cast.ChooseLoadout` / `cast.ChooseFrom`** now and all three
  call it; the subject is a worded noun phrase (`unit "x"`, `the build "x"`) so
  each caller still says what it is talking about. The builder shows the refusal
  **as the kit is chosen**, which can only match the write because it is the same
  call.
- Every field carries a **help line describing the focused one**, and a shape
  chooser draws the cells it covers on a sub-screen built from `pattern.Targets`
  and `hex.Render`, so the drawing cannot disagree with what the engine catches.
  Both replaced a static footnote that stated the parts-per-thousand convention
  and was not read. A field that needs a *list* — the statuses a skill inflicts,
  and each of the five allowlists — opens the same picker as the kit; each picker names
  its own hint, because the kit's (order, and what this character cannot take)
  says nothing true about an allowlist.
- A skill's **Vietnamese name is data, not Go**: `skill.Skill.Name` is opaque
  display text and `internal/core` never learns what a language is. An authored
  name wins, the compiled table in `internal/i18n/gloss.go` answers when there is
  none, and the bare id when there is neither. Any table showing it drops the
  column rather than drawing it empty.
  ⚠️ **This used to read "which is still the case for all nineteen shipped
  skills", and that has been false for a while.** Re-measured: **43** shipped
  skills, **43** carrying an authored name, and `skillGloss`'s nineteen ids
  intersect `skills.json` **not at all**. The table is now reached only by
  `internal/testfixture`, which is exactly why a test built on a fixture skill
  measures the wrong path — see `docs/architecture.md` § *The event log is the contract*.
- The skill listing **filters by name**, typed, live, on `/`
  (`internal/screen/skills.go`). Forty-three skills is a screen and a half at
  the declared floor, and the only way to a row was the arrow keys.
  - **It is a mode, and that follows from the keyboard rather than from taste.**
    Every letter this screen has is already a command — `q`, `a`, `e`, `k`, `j`,
    `?` — so a field sharing the keyboard with them could take no query at all.
    While the field has it, only `esc` (clear and close), `enter` (keep and hand
    the rows back), `backspace` and the two **arrows** are keys; the vim pair is
    text like every other letter, which is why the arrows are the arrows here.
  - **A row matches on its id or on its Vietnamese name, case- and
    diacritic-insensitively** — typing `diep` finds `phi diệp` — through
    `i18n.Fold` / `i18n.Matches` / `i18n.MatchesSkill`. That is the point of the
    feature rather than a refinement of it: on a terminal with no Vietnamese
    input method the name half is otherwise unreachable, so the filter would be
    an id filter with a misleading name.
    ⚠️ **The fold is an explicit table and `golang.org/x/text` stays indirect.**
    NFD-plus-strip-`Mn` needs a hand-written entry anyway, because **`đ` is not a
    `d` with a mark on it**. And unlike every gloss table here, this one has to be
    **complete**: a missing letter does not fall back to an id, it silently stops
    matching. `TestEveryLetterAShippedNameUsesCanBeFolded` walks the shipped books
    and both catalogs, so a name authored later in a letter the table lacks is a
    red test rather than a row nobody can reach.
    ⚠️ **The match reads the *data* and asks nothing about the language in
    front**, even though the English listing draws no name column: `ctrl+l` works
    from every screen and keeps everything typed, so a query that found different
    rows after it would be the screen mutating behind the author. The stated cost
    is that an English reader can be handed a row whose id does not hold what
    they typed.
  - ⚠️ **`s.cursor` indexes the FILTERED view**, so every read of it goes through
    `draw.SkillsScreen.Rows` / `Selected` — the funnel `draw.PickState.Visible` already is.
    `e`, `?`, the damage row under the listing and the description screen's own
    `↑/↓` all used to index `s.skills` with it, and **the two lists are identical
    while nothing is typed**, so a wrong read would have gone on passing the whole
    suite while an author edited the wrong skill.
    `TestEveryKeyThatReadsARowReadsTheFilteredOne` filters to a query whose first
    match is the *seventeenth* skill declared and asserts the fixture is that
    discriminating before it asserts anything else.
  - `a` is deliberately **not** guarded on there being a row: it indexes nothing,
    and a filter that found nothing is exactly when an author wants to write the
    skill they were looking for. `e` and `?` decline.
  - **The filter row costs a listing row, paid unconditionally**: `skillsRoom`
    reserves **ten** now rather than nine. That is the same rule the other two
    conditional lines there follow — reserve for the busiest state — and it buys
    something visible: pressing `/` narrows the list without also shifting every
    row under it up by one. The footer had to be trimmed to name the key and no
    key was given up; the words after `↑/↓`, `esc` and `q` are dropped, which is
    `BrowseFooter`'s own squeeze. Measured (vi / en): the listing footer **77 / 78
    → 65 / 74**, and the filter footer **63 / 70**.
  - The three states are registered in `everyScreen` (`filtering skills`,
    `filtered skills`, `skills filtered to none`) and are **driven with the keys
    an author would press**. A state added without an entry there has no width
    test, no translation test and no leak test, which is a mistake this
    repository has now made four times.

## Every listing measures its rows from the window

**Every listing measures its rows from the window, and the cast browser was the
one that did not.** `browseRoom` is `skillsRoom`'s twin: the listing takes
`c.Height - 4` less the two rows above it, the blank below it and
`browseDetailRows`, and `Window` scrolls it around the cursor. It drew **every**
row until then, so each character shipped cost the detail pane a row on every
window size — and the row a forking line loses first is the **form row**, which
is the only thing saying which arm the stats under it belong to.

- ⚠️ **The reserve is a written-down constant and that is a COST finding, not a
  shortcut.** `speciesRoom` measures its variable pane (`longestNote`) because it
  can — a wrap over one authored string. Doing the same here means rendering
  every character's pane to count its lines, which measured **13ms a redraw** at
  twenty-three characters: `os.Stat` behind the art row is **53µs a character**
  on Windows and `Context.Wrapped` is **15µs** a call against ten of them a pane.
  That is per keystroke and it grows with the cast, which is the very thing the
  reserve exists to stop mattering. So the number is written down and
  `TestTheDetailReserveIsTheTallestPaneInTheBook` holds it **exactly, in both
  directions** — over is a pane the frame cuts, under is rows the listing gave up
  for nothing.
- ⚠️ **One number rather than one per language**, measured at `MinWidth` where
  every wrapped row wraps hardest: the tallest pane is `pokemon.mew`'s at level
  16, **24 rows in Vietnamese and 17 in English**, because a glossed row draws
  nothing when there is no name to draw. A screen may not branch on which
  language is in front, and what is measured is a *count of rows* rather than a
  width a label can be asked for — so the reserve is the tallest the pane ever
  gets and an English reader sees blank rows under it. Same price `speciesRoom`
  already pays.
- ⚠️ **At the 120x24 floor the pane is cut anyway and no split can help**: the
  reserve alone is 24 rows against the 20 the frame leaves a body. The floor of
  three is navigation kept rather than a share of a budget that balances — the
  axis this fixes is the cast's length, not the window's height.

## Authoring with nobody watching

**`hexforge new` must work with nobody watching.** A preset-supplied value is
not missing, so an unattended run takes every default and errors only on a field
that has none, naming its flag. Two traps live here. `os.Stdin.Stat` cannot tell
a terminal from `/dev/null` — both are character devices — so the mode check is
only a first guess and **EOF on a read is the authoritative signal**; hitting it
turns the rest of the session unattended rather than failing on a field whose
default was fine. And the kit is asked **before** the element, because the kit is
what decides which elements are legal: asking the other way round means either
validating against a preset the author is about to replace with `--skills`, or
accepting an answer the write then refuses.

**That prompt order is a prompt's problem, not a form's.** At a prompt an answer
once given is given, so one order has to be right. On the form both fields are on
screen and **either may be filled in first**: `forge.Carrier` says an unanswered
fact restricts nothing, so with no element yet the picker marks nothing, and with
an element settled it marks the skills that element cannot take. Going the other
way, the carry line under the form refuses an element the chosen kit cannot take
and names the skill. Neither direction mutates the other's answer — changing the
element does not empty the kit, it turns those rows into marked rows and the carry
line red, which is a state an author can see and fix rather than one that happened
behind them. Do not "tidy" that by dropping the offending skills: a silent
mutation is how a one-way order comes back, and it makes the live check a lie.

⚠️ **The fifth: pasting stopped working in both clients and nothing failed.**
v2 enables bracketed paste by default and delivers a paste as **`tea.PasteMsg`**,
not as a run of `tea.KeyPressMsg`. `bubbles`' `textinput` handles that message
correctly — but a model whose `Update` switches only on `tea.KeyPressMsg` drops it
before the field ever sees it, and `PasteMsg` appeared **nowhere** in this
repository until #274. Every text field in both clients ignored ⌘V, and the join
screen ignored the room code somebody had just copied.
⚠️ **`ctrl+v` was worse than unhandled: it read the clipboard and threw it away.**
`bubbles` binds `ctrl+v` to `textinput.Paste` by default, which shells out to
`pbpaste` — and then delivers `textinput.pasteMsg`, which is **unexported**, so a
model outside that package cannot name it and cannot forward it. The fix reads the
clipboard through `internal/clipboard` and returns a `tea.PasteMsg`, so the
terminal's own paste and the program's own key end at the same insert.
