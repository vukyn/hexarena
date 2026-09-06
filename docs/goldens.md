# The data files and the goldens that record them

What is in `internal/seed/data`, what each of the golden files under `testdata`
is a record *of*, and what an authoring write is allowed to do to a committed
file. It moved out of `CLAUDE.md` § *Data and golden files* on 2026-09-07 for the
reason `docs/screens.md` did: it is subject matter, `CLAUDE.md` is loaded in full
at the start of every session, and this was 31KB of it. Nothing was rewritten,
shortened or dropped.

⚠️ **Three binding sentences stayed behind and are NOT repeated here** — that
balance lives in `internal/seed/data`; that there are **sixteen** embedded files
and the name of each is spelled in **three** independent places, of which the
`dataFiles` slice is the silent one to miss; and that `make golden` is followed by
**reading the diff**, because a number that moved unexpectedly is a finding rather
than noise. Read `CLAUDE.md` § *Data and golden files* first; this file is the
reasoning and the measurements under it.

⚠️ **The `##` headings are new and the paragraphs under them are not.** The
block arrived as one run of bold-led paragraphs, which is unreadable at this
length, so a heading was put over each of the natural breaks — and the bold lead
it names was **kept**, because it is the sentence people quote and deleting it
would make "nothing was rewritten" false. The repetition is the price of that.

## What the goldens record, and what moved them

**Two goldens grew with the lobby, and both are named here so the next `make
golden` reader knows what moved.** `internal/screen/testdata/screens.golden`
gained **two** entries — `a live battle` and `a live battle waiting`, the two
states `draw.PlayScreen.Live` adds — and
`cmd/hexarena-tui/testdata/screens.golden` gained **nine**: three join-screen
states, the waiting screen, three live battle states and two result screens. The
only *existing* block that moved in either is that client's **menu**, which grew
its ninth row and, in English, widened its measured label column from eight cells
to eleven because "join a room" is the longest label there now.
`cmd/hexforge-tui`'s golden is **unchanged**, which is the check that the local
`PlayScreen` drawing did not move: that client draws it in local mode only, so
any diff there would have been the regression.

`skills.json` is the exception that is **balance and tool-written at once**:
`hexforge skills add` and the full-screen client's skill form both append to it,
and `hexforge skills edit` and the same form (opened with `e` on the listing)
both change what is already in it. That is why it is committed in the form
`Book.Marshal` writes, on the same terms
as `cast.json` below, and why a save says **the golden files have moved** rather
than only that it wrote — a power reaches `skills.golden` and `describe.golden`,
so `make golden` and reading the diff is the next step and not an afterthought.

⚠️ It does **not** reach `scenarios.golden` or `progression.golden`, and this
paragraph said it did until 2026-08-31. Each golden moves for what its generator
is handed and nothing else: `scenarioReport` takes the rules, the chart, the
modifier bounds, the ceilings and the pattern book, and reads the **status** book
for its poison ladder — no skill book, no cast. `progressionReport` takes the
limits and the rules alone. The four skills #182 added moved neither.

Two things about that write are worth knowing before touching it:

- `skill.Book.Marshal` keeps **declaration order** where `cast.Book.Marshal`
  sorts by id. A cast is a set looked up by id; a skill book's order is authored
  information (basic attacks, then the elemental ones, then utility, which is the
  order `skills.golden`'s table reads in), so sorting would shuffle a design
  record to buy the one-block diff that appending already gives. `skill.Book.Replace`
  is the same fact for an edit and is why it keeps a skill's **position**:
  reordering on a one-field change would rewrite the whole file and the whole
  golden table.
- `Skill.MarshalJSON` builds the **parse shape** (`skillFile`) rather than
  carrying tags of its own, so the only fields that can be written are the fields
  the parser reads. That is what makes the rewrite lossless for the four blocks
  the authoring form does not ask about — `requires`, `strips`, `scaling`,
  `self_applies` — and `TestTheShippedSkillBookSurvivesBeingWritten` measures it
  on the real data rather than on a fixture.
  ⚠️ **This said "a field added to one struct is a compile error in the other
  until it is added there too, which is the point", and that is FALSE** —
  measured 2026-09-05, from step 3 of the draft, which was about to be designed
  around it. `Skill.file()` (`internal/core/skill/skill.go:1115`) is a **keyed**
  composite literal, so a field added to either side compiles fine and is
  silently dropped by the writer; only an *unkeyed* literal would give the
  compile error the sentence describes. What actually catches it is the round
  trip over real data named above, which is why that test is the load-bearing
  half of this bullet rather than the reassurance at the end of it. **Do not
  reach for this as the precedent for "two structs the compiler keeps in step"**
  — there is no such precedent in this repository; the shapes that must not
  drift are either one struct embedded in two (`wire.DraftDecision`) or held by
  a round trip.
  Note `Scaling`'s zero value is **not** its default (the zero stat is health,
  and a skill scaling off health is refused), so a `skill.Skill` built in Go must
  set `skill.DefaultScaling()`; the refusal is loud rather than silent on purpose.

## Writing to a data file the game ships

**Editing a skill can break what adding one cannot**, and that is the whole shape
of `forge.Library.EditSkill`. Nobody carries a new skill; shipped units carry an
edited one, so changing an element or narrowing a `restrict` can leave an authored
character — or an archetype preset's kit — no longer allowed to hold it. So an edit
re-parses `archetypes.json` and `cast.json` **off the disk** against the edited
book *before* writing anything, and a refusal from either is the thing to prevent
rather than to discover: a written file that then fails to load is a data
directory the game does not boot from. The re-parse is the authority; naming *who*
would break is a classification after the fact (`brokenPreset` / `brokenCharacter`
walking `CheckKit` and `CheckPresetKit`), exactly like `checkAffinity` classifying
after the element chart has already said no — nothing there can turn a no into a
yes, and a refusal neither walk recognises names nobody and keeps the parser's own
words. `forge.CheckPresetKit` is the preset half of that, bringing forward
`cast.resolveArchetype`'s three restriction rules the way `CheckSkill` brings
forward `resolveCharacter`'s five. Two other things an edit must keep: an
**absent** field and a field set to **zero** are different answers (hence
`SkillEdit`'s pointers and `FlagSet.Visit` in `cmd/hexforge`, and an explicitly
empty list is how a restriction is cleared), and a skill's **id is not editable** —
a rename has to cascade through every kit and every `restrict.characters` list, so
`SkillRenameError` says so rather than half-doing it.

Three of those files are the cast rather than the balance, and `cmd/hexforge` both
reads and **writes** them:

- `origins.json` — the works characters are borrowed from. Hand-editable;
  `hexforge origins add` appends to it.
- `archetypes.json` — the role presets: a suggested stat curve and kit per role.
  This is what stands in for a character **class**: a class was in the original
  design and was dropped deliberately, because with skills declared as data the
  curve and the kit already carry what a class name would have. So an archetype
  has **no mechanical effect** — it never reaches the engine, `battle.Roster`
  carries no archetype, and nothing branches on one. Do not add a class without
  deciding what it would do that a stat curve and a kit cannot; the live proposal
  is to give an archetype its first mechanical weight through a passive instead.
  Hand-authored only; the tool never writes it. Every preset's curve must pass
  `progression.Limits.CheckTable`, and the ids match the roles `roster.json`
  already uses.
- `cast.json` — the authored characters, each an evolution line
  (`progression.Line`) plus an origin, an archetype it was tuned from, an
  affinity, a kit and a path to its art. `hexforge new` appends to it.

`cast.json` and `origins.json` are **meant** to be committed in exactly the form
`Book.Marshal` writes — two-space indented, sorted by id. The reason is that the
tool rewrites the whole file on every addition, so a committed form that has
drifted from the written one makes the next `hexforge new` produce a diff of the
entire file instead of one block. Marshal is also the one place in `cast` that
*imposes* an order rather than preserving the authored one; everything else keeps
declaration order, because a map range would randomise it.

⚠️ **Neither file is actually in that form today, and the test that was supposed
to say so cannot see it.** Measured 2026-08-31: `cast.json`'s characters are in
declaration order, not sorted — `naruto.naruto` sits fourth where Marshal would
put it first — and `origins.json` reads `pokemon` then `naruto`. So the next real
`hexforge new` **will** reshuffle both files, which is precisely the churn this
paragraph exists to prevent. `TestWrittenCastIsStableAndReloads` passes anyway
because it reads the file out of the **scratch** directory, which
`testfixture.Inject` has already rewritten through `SaveCharacter`/`Marshal` — so
it compares Marshal's output against Marshal's own output and would pass on any
committed file whatsoever. → `TODO.md`.

## `roster.json`, the instrument

**`roster.json` is an instrument, not a scenario, and it has four contracts.**
It is 3v3 by character reference — ally Venusaur 60 / Wartortle 16 / Charmander 8
against enemy Blastoise 60 / Charmeleon 30 / Ivysaur 30 — and each of those is
load-bearing:

- **No unit on both sides.** It used to be the same character three times per
  side, and a mirror cannot measure anything: a change helps both squads by
  exactly as much, so the win rate moves only by noise. That is what stopped
  `razor_leaf`'s pierce being judged by anything but its damage table.
  `TestTheShippedRosterIsNotAMirror` compares the **resolved** units — name and
  stat line — because a species and a level resolve to those, and two units
  agreeing on them are the same unit however they were authored.
- **Every unit reaches past the enemy's front rank.** `battle.New` refuses
  nothing on reach any more — it cannot, because a range of one always finds
  whoever is foremost — so the roster is held to the stricter rule by
  `TestEveryShippedUnitCanReachEveryEnemy`: every unit carries at least one skill
  of depth two or more. A squad whose whole back half is decoration until the
  enemy's front line dies is playable, and it is not what this roster is for.
  ⚠️ **This contract used to be about cells and no longer is.** Slot `1,2` was
  once **four** cells from the enemy's own `1,2` — past every range in the cast —
  and a draft that used it stalled 5 seeds in 4000, not even as a draw: a
  survivor kept refreshing a regeneration, so something was always pending and
  `frozen` correctly never fired. Distance cannot strand anybody now; **depth**
  can, which is why the check moved onto the kit.
- **Both trait states and all three stages are in play.** Charmander at 8 is below
  `blaze`'s unlock level, Wartortle at 16 sits exactly on `endurance`'s, and Ivysaur
  at 30 has earned two traits and fields neither — so a battle exercises a unit with
  its trait, one that has not earned one, and one that declined. Since `blaze`
  became gated it carries a third state as well: Charmeleon holds it from the
  opening board and only comes *into* it partway down, which is what a shipped log
  now shows a `passive_held` for mid-battle.
- ⚠️ **The formation is what makes the figure, and it is placed to be screened.**
  Every unit stands where it does for a reason: each ace holds its side's **back**
  column at `0,1`, the two young units share the middle column at `1,0` and `1,1`,
  and the **front column is empty on both sides**.
  ⚠️ **Placement is purely defensive now** — a unit's reach does not depend on
  where it stands, only on what it can be aimed past — so "ace at the back" is the
  dominant placement, and the roster ships it on **both** sides rather than handing
  it to one. The shipped roster had the aces in **front**, authored for a board
  where reach was distance, and it read **27.6%** ally over 4000 seeds once ranks
  landed. Moving the two aces to the back column and changing nothing else — not a
  level, not a loadout — reads **47.3%**, and the 40-seed smoke test moved 12/40 to
  24/40 with it. Three separate things ride on that shape, and all three were
  measured:
  - **The ace is behind a screen.** Reach is counted in occupied ranks, so the
    pair in the middle is the first rank an attacker meets and the ace is the
    second — out of reach of every depth-one skill until the screen dies, which
    is the blocking rule doing the thing it exists for.
  - **The empty front column is deliberate.** An empty rank costs no range, so the
    ace sits at depth **two** rather than three and both sides can still be fought
    to a finish. The shipped board is therefore the standing demonstration of that
    rule as well as of blocking.
  - **The screen is adjacent.** `1,0` and `1,1` touch; splitting the pair to
    `1,0` and `1,2` reads **31.1%** against the adjacent pair's 47.3% over the
    same 4000 seeds, because an area shape that catches both of them is most of
    what the young units are for.

  `TestTheShippedFormationScreensItsAce` holds the first and the third of those,
  because a flattened formation is a balance change that reads as a tidy-up.
- ⚠️ **The levels are calibrated against how well the opponent plays, so an AI
  change invalidates them.** The two young enemies were 28 and 16 until `Suggest`
  learned to price statuses; the roster then read **80.0% ally over 20,000 seeds**
  and had to be re-levelled to **Charmeleon 30 / Ivysaur 30**, which reads 49.1%.
  ⚠️ **The ace level is not a dial** — Venusaur 60 → 50 alone takes the ally side
  from 79.0% to 4.0% at 4000 seeds. Tune the young units, and change one thing at a
  time: the loadouts were deliberately left alone in that pass so the level was the
  only thing measured.
  ⚠️ **They were left alone in the blocking pass too, and the room is narrower
  than it looks.** Charmeleon cannot go below **20** (`dragon_rage` is learned
  there) and Ivysaur cannot go below **20** either (two traits earned is the
  contract above), so the whole dial is 20..30 on each. Swept over that grid on
  the screened formation it spans **40% to 82%** ally, and the shipped 30/30 sits
  at the bottom of it — which is why the placement was the answer and the levels
  were left alone.

⚠️ **The 40-seed sweep in `TestSeedBattlesFinishFromEverySeed` is a smoke test, not
a measurement.** It read 45 per cent on a draft whose true rate over 4000 seeds
was 55. Tune levels against a few thousand seeds and quote that figure; the test's
job is only that the battle finishes and that neither side is a scripted defeat.

## `builds.json`, the late-game catalogue

**`builds.json` is the late-game catalogue: which four skills and which trait a
character is *for*.** A learnset of nine skills and five traits offers more
combinations than anybody would field, and before the file existed the only kit
the repository could name was "the first four declared" — the order the file
happens to list, which is not a decision. `cast.ParseBuilds` checks every entry
against the cast book at `progression.LevelCap` on the furthest form, so a build
naming a move only an earlier form knows (`sleep_powder`) is refused with the list
of what that form does know.

- A build adds exactly two things over the loadout it names — a `name` and a
  one-clause `intent` — and **nothing numeric**: everything it does is already
  described by its skills and its trait, so a figure in either field is refused at
  parse time (the same rule skill `flavour` lives under).
- **A character listed there has at least two builds.** One build is not a build,
  it is that character's kit, and a screen offering a single option tells a player
  they have a decision they do not have. A character with none is the honest case —
  Naruto today — and `TestABuildIsACatalogueOfChoicesRatherThanOfKits` is the claim.
- **Three is the shape a mechanism with a middle needs.** Magnemite was the first
  character with three, and the third is what says its question has more than a yes
  and a no in it: `trickle` converts the counter as fast as it lays it down,
  `surge` ignores the mechanism entirely, and `hoard` waits and takes the pile.
  ⚠️ **`surge` is a direction rather than an absence** — the same character built
  as though the counter were not there — and
  `TestTheThreeMagnemiteBuildsAnswerTheCounterDifferently` asserts it spends
  **nothing**, because "three answers" is otherwise two answers and a duplicate.
  ⚠️ **None of the three wins a duel** (0 or 1 of sixty against Charizard, the
  heaviest attacker in the cast against the thinnest frame in it), exactly as the
  mender's two did not. What a duel prices here is what a build spends its turns
  on, and the ordering is what is held: blows, size of each, and stacks a
  discharge (drip **1.00**, hoard **3.85**).
- **Mew has three for the opposite reason**, and the pair is worth reading
  together. Magnemite's three are three answers to *one* question; Mew has no
  question of its own — no counter to spend, no element to lean on — so its three
  are three different characters, and what is held is that no two of them spend a
  turn on the same thing: `feed` is the only one that heals at all, `wither`
  inflicts several times the statuses, and `borrowed` deals the most damage and
  misses by far the most doing it. Four columns, three builds, each leading one.
  ⚠️ **`borrowed` carries nothing written for Mew** — every skill in it is already
  carried by somebody else — which is `surge`'s move made about a whole character
  rather than about a mechanism, and `TestOneMewBuildCarriesNothingOfItsOwn`
  counts rather than spot-checks, because a build that quietly picked one of Mew's
  own back up would still work and would no longer be saying anything.
- **`mewtwo.origin` is that build's mirror**, and the pair of them is why both are
  in the catalogue: one carries nothing of its own and the other carries nothing
  but the original's. It is also a *measurement* rather than only a theme — see
  `CLAUDE.md` § *Piercing is a ratio, and it stops at the strike* for what
  putting one loadout on two bodies said.
- ⚠️ **The catalogue and the design tables in the tests must agree.**
  `poisonBuild`/`sustainBuild` (`bulbasaur_test.go`), `fireBuild`/`dragonBuild`
  (`dragon_test.go`) and `tankBuild`/`semiBuild` (`squirtle_test.go`) are hardcoded
  on purpose — they are what was measured — and
  `TestTheShippedBuildsAreTheOnesTheTestsMeasure` fails if the data drifts from
  them, kit **or** trait. Shipping a new build means measuring it in a test first,
  then adding the row.

Art lives under `internal/seed/data/assets/` and is **not embedded** — the embed
directive names the JSON files one by one. The two placeholder SVGs there exist so
`hexforge check` passes out of the box; replace them, do not delete them without
also replacing the example characters.

The golden files under `testdata` are **the design record**, not fixtures to be
regenerated on autopilot:

- `internal/seed/testdata/scenarios.golden` is the largest one and the most
  useful: it holds the measured behaviour of buffs, debuffs, elemental
  effectiveness, accuracy, dodge, block, multi-strike, area coverage, the turn
  economy and the damage-over-time ramp, with the numbers the design decisions
  were made from.
- `replay.golden` is a whole battle rendered from its log.
- `skills.golden`, `elements.golden`, `progression.golden`, `combat.golden` are
  the tables each book produces.
- `origins.golden`, `archetypes.golden`, `cast.golden` are the same for the cast:
  which works are catalogued and who was borrowed from each, every preset's curve
  with what it spends of the effective-health budget, and every character
  resolved at each of its stage boundaries and at the cap.
- `cmd/hexforge-tui/testdata/screens.golden` is the **rendered client**: every
  entry `everyScreen` registers, in both languages, at the 120x24 floor and at
  160x60 — 200 renders, 8200 lines. It was the only golden in a `cmd` package
  until `cmd/hexarena-tui` grew one, and
  it exists because the screens had *property* tests (width, translation, no
  leaked wording) and no byte-level one, so a misplaced space or a moved clip
  point passed everything. ⚠️ **The header line is deliberately not recorded** —
  it names the data directory, so it names the machine; the fixture also hands
  `forge.Load` a **relative** directory, because the check screen's count line
  prints one in the body too. See the doc comment on
  `TestEveryScreenDrawsWhatTheGoldenHolds` for both, and for what a golden
  written today does and does not prove about the step before it.
- `internal/screen/testdata/screens.golden` is the **moved screens, in the
  package that owns them**: the six listings — `chart`, `elements`, `species`,
  `statuses`, `traits`, `builds` — plus the two states nothing shipped can draw
  (`unclaimed kind`, `traitless build`), the **description screen in both of
  its readings** (`skill blurb`, `trait blurb`), the **five states of the
  picker** (`kit picker`, `allowlist picker`, `filtered picker`, `status picker`,
  `reading a skill`) and the **skill listing with the seven states of it** the
  client's own sweep registers (`skills`, `add a skill`, `edit a skill`,
  `edited a skill`, `filtering skills`, `filtered skills`,
  `skills filtered to none`, `shape diagram`), the **squad builder** at each of
  its three depths with the two member states and the two pickers that go with
  them, the **works catalogue** with the `add a work` form over it plus the
  two states **neither** sweep could draw before it moved (`an empty works
  catalogue`, `a refused work` — `i18n.OriginsEmpty` and `i18n.AddRefused` each
  measured at **nought hits in both goldens** beforehand; the third gap,
  `i18n.OriginAdded`, stays open because it prints `Lib.OriginsPath()` and
  `noAbsolutePath` walks the recorded body), and the **played battle** in the
  six states of it that share no line (`a battle`, `aiming`, `a battle over`,
  `a scrolled battle log`, plus two more **neither** sweep could draw —
  `a saved battle`, whose note measured **nought hits in both goldens** and whose
  path is a *relative* value here, and `a battle with no pairing`, which the
  client's fight guards its `p` against) — in both languages at the 120x24
  floor and at 160x60, plus the **ban and pick** in the twelve states of it that
  draw a line no other state draws (`a draft`, `a draft waiting for a peer`,
  `a draft ban thrown away`, `a draft on the other side`, `a draft pick`,
  `a draft part way through`, `a draft with one candidate`, `a draft loadout`,
  `a forked draft loadout`, `a refused draft loadout`, `a draft arranging`,
  `a cancelled draft`) — **248 renders, 6270 lines**, body and footer recorded apart
  because a screen here answers with the two separately and every wording squeeze
  in this file is a footer.
  ⚠️ **The draft's twelve are built as VALUES and there is no other way to build
  one**, which is what declaring `draw.DraftLive` bought: a `DraftScreen` is
  handed a reading, so no room, no listener and no goroutine goes anywhere near
  either golden. Three of the twelve are states `cmd/hexarena-tui`'s own sweep
  cannot reach at all — a decision with **one candidate** (arithmetic the shipped
  pool does not produce), a **refused** loadout, and a **fork with no arm named**
  — and a fourth is step 5a's finding: a draft with nothing recorded, where a ban
  is thrown away. ⚠️ **It exists because the layout of code in
  `internal/screen` was held by a file in another package.** Measured after #205:
  widening the status category column by one cell
  (`Pad(row.Category.String(), column+1)` → `column+2`) left **every test in
  `internal/screen` green** and was caught by
  `cmd/hexforge-tui/testdata/screens.golden` alone. ⚠️ **And it now catches what
  the client cannot.** Measured after #222: widening the squad catalogue's id
  column by one cell leaves the **whole client suite green**, because
  `scratchData` deletes `squads.json` and so no test in `cmd/hexforge-tui` ever
  draws a catalogue with a row in it. The two goldens are not one net in two
  places; each sees a screen the other is blind to.
  ⚠️ **The skill listing's entries are driven with keys where the client's are**,
  and each hand-built state asserts it drew the line it exists for. The three
  filter states are what the query decides, so a field set by hand would record a
  test's idea of the filter; the reported edit is built as a `forge.SkillChange`
  value rather than written, because nothing in this package touches the data
  directory.
  ⚠️ **The picker is handed its list, so it has no one shape and its entries are
  a decision rather than a screen each.** The five are the paths through `View`
  that share no line with one another: rows carrying a refusal and a detail
  column, rows with a filter line over them, that filter narrowed, a field and
  its percentage under the list, and the reading pane, which replaces the list
  outright. They are **hand-built** where the client's five of the same name are
  raised through a form — two of the three screens that raise a picker are in
  `cmd/hexforge-tui` still, and the skill form's five are raised here now — so the
  two records are the drawing and the raising of it, which is this pair of
  goldens' whole arrangement.
  ⚠️ **The blurb gets two entries for three subject kinds.** A listed skill and a
  battle option are one `SubjectKind` — same id, same paragraph, same footer, only
  `At`/`Of` differ — so a third entry would record the same render twice;
  `NoSubject` is the arm a raise cannot reach, and a client's applier is what
  proves that. The **art preview** has an entry here as well now, and it is the
  one that records a *drawing* rather than a sentence.
  ⚠️ **Three more entries record a line that FORKS** — `a forked cast row`,
  `a forked art preview`, `a forked trait blurb`, over `pokemon.poliwag` at level
  46. Every other entry in all three records is a line that does not fork, which
  is what let a user meet `level 46 reaches [Poliwrath Politoed], which are
  alternatives: name the one being fielded` drawn in red where a picture should
  be. The three views fail differently and that is why there are three: the
  preview drew the refusal, the detail pane stopped at the row with it in, and the
  blurb drew **neither arm's traits and said nothing at all**, which is the shape a
  record is the only thing that can catch.
  ⚠️ **Neither golden is a subset of the other and neither may be dropped**, which
  was measured both ways rather than assumed. A trailing newline left on
  `SpeciesScreen.View`'s body reddens this one and is **absorbed by the frame's
  blank padding** in the client's; the client's `frame` budgeting one row fewer
  leaves this one green and takes the caveat line off the *statuses* screen in
  that one. What the package golden cannot see is everything the client composes —
  the header, the blank, the vertical cut and its `Truncated` marker, the
  horizontal clip. `screen.Ellipsis` reddens **both**, because the traits listing
  clips its own carrier row.
  ⚠️ Unlike the client's it drops **nothing** and needs no relative directory
  trick: no file in `internal/screen` calls `.Dir()` and `check` did not move, so
  the books load straight from `../seed/data`. `noAbsolutePath` asserts that
  anyway — a property that holds by construction is one a later change breaks
  quietly.
- `cmd/hexarena-tui/testdata/screens.golden` is the **game client's** framing of
  the same screens: every entry its own `everyScreen` registers — 46 over all 18
  of its views — in both languages at the 120x24 floor and at 160x60,
  **184 renders, 7544 lines**. It is the third record of one set of screens and
  none of the three replaces another: `internal/screen`'s holds the drawing, the
  authoring tool's holds *its* framing of it, and this one holds **four things
  neither of the others can draw**.
  ⚠️ **The three read-only footers.** `i18n.SkillsReadFooter`,
  `i18n.OriginsReadFooter` and `i18n.SquadsReadFooter` are what a screen draws
  when `screen.Context.Authoring` is nought, and measured before this golden
  existed they came back at **nought hits in both** of the others, in both
  languages, because there was no client to draw them.
  ⚠️ **A squad catalogue with rows on it.** Both other fixtures delete
  `squads.json` — it is the author's own working document — and this one writes
  two sides into it, so the id column that #222 measured as invisible to the
  whole authoring suite is finally recorded somewhere.
  ⚠️ **A battle at three a side.** A one-a-side board, roster, order line and
  option list come to exactly the twenty rows the floor leaves, so nothing is
  ever dropped and the notice naming what the window was too short for is drawn
  by nothing; three a side is 24 rows against 20, which is where the budget
  starts deciding.
  ⚠️ **A `draw.PickState` in front of a screen.** Every picker in the vocabulary
  was the *authoring* half of it until a drafted loadout arrived, so this client
  had no picker field at all and `navigate`'s `draw.Pick` arm was a no-op that
  said so in as many words. `a draft kit` and `a draft trait` are the first two
  ever recorded here.
  ⚠️ It drops its header line and hands `forge.Load` a **relative** directory,
  for the reason the authoring tool's does: `frame` names the data directory and
  a saved battle's own note names the file it wrote, both of which would
  otherwise be a machine in a committed file. The drop is **asserted** rather
  than scrubbed, and `noAbsolutePath` walks every recorded line.
  ⚠️ The **art preview** is in every sweep and in all three goldens now. Its
  picture is exempt from the width sweep and nothing else about it is — see the
  `hexforge-tui` entry below for the arithmetic. Both clients also carry
  `a forked art preview` and `a forked trait blurb`; those two are the only
  entries in either client's record drawn from **shipped** art rather than the
  fixture's flat rectangle, so they inherit `internal/screen`'s same-machine
  caveat and are worth reading as a finding rather than a gate if they ever move
  on another architecture.
- `internal/wire/testdata/messages.golden` is the **PvP protocol**: one entry per
  message kind — the exact bytes `wire.Encode` produces, indented so a diff points
  at a field rather than at a four-hundred-character line — so a **wire change
  shows up in a diff**. A field renamed, a field dropped, an `omitempty` added or
  taken away, a kind or a code renamed, or the event digest's framing changed each
  moves a line here and nothing else in the suite shows a reader what moved.
  ⚠️ **Every body is a hand-written fixture and none of it comes from the shipped
  data**, which is the whole reason this file is safe to have. A `start` body
  carrying a real roster would move on every balance commit — a stat curve retuned,
  a skill's level moved, a character added — while measuring nothing about the
  protocol, and a golden that moves for reasons unrelated to what it measures is a
  merge-conflict generator. It is the same argument that gives `internal/seed`
  **no** golden on the data digest at all, and it is held rather than promised:
  `TestTheGoldenIsBuiltFromNothingShipped` refuses a record holding any shipped id
  prefix. The `turn` entry's digest **is** computed, off the fixture events, which
  is what pins `DigestEvents`' framing in the same diff.
  ⚠️ **The other side of that safety is a blind spot, and it was measured.** A
  hand-written fixture pins the **format** and can never see the **producer**:
  taking one line out of `gate.go` so the room stops filling in `TurnCap` on the
  `wire.Welcome` it builds leaves `internal/wire` — this golden included —
  **entirely green**, because the golden's welcome is a fixture's and not a
  room's. What catches it is `internal/room`, by name, in two tests
  (`TestTwoFakeClientsFightAWholeBo3InProcess` compares every welcome field
  against the room's own `Config`, and
  `TestTheTurnCapEndsABattleAsADrawTheOutcomeAlreadyHas` asserts each client
  stopped on the turn the room stopped on). So a field added to a message needs
  **two** things: an entry here saying it travels, and an assertion in the room
  saying it is filled in.
