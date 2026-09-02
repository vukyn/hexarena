# TODO

A short index of what is done and what is not. It is deliberately **thin**:
nothing here explains a design, because the explanations already live in
`CLAUDE.md` (§ Open work — the constraint each piece has to respect) and
`README.md` (§ Roadmap — the detail and the open questions). This file exists so
that "what is left" can be read in a minute instead of found in 300KB of prose.

⚠️ **This file goes stale and the repository is built not to tolerate that.**
Everything else here is derived — descriptions come from the data, the affinity
chart is drawn from the chart, goldens are regenerated. A hand-kept list is the
one thing that can quietly become a lie, so it carries as little as it can:
one line per item, and a pointer to the place that holds the real answer.
**When you finish something, tick it here in the same commit.**

## Done

The honest record is the git history — every entry below is a merged PR and
`git log --oneline` is the full list, in order, and cannot go stale. The grouping
is only so the shape is readable.

- **Engine.** Turn-based battle on an odd-q hex grid, action-value turn order,
  elemental chart. Verifiable logs — a cell and an aim are written only where
  there is one — replay, undo. Draws for a battle nobody can act in and for a
  deadlock. Piercing, healing, draining, regeneration. Conditions read the target
  *or* the caster, as a threshold or as a gradient that grows with the caster's
  own wounds. Reach counted in ranks from the far side rather than in cells from
  the caster. A resistance share may be negative, so a target can be made easier
  to afflict and not only harder. Summons. Taunt.
- **Traits.** A character carries traits as well as a kit: permanent grants,
  gated grants that come and go, resistances, replies to whatever attacked,
  amplifiers, drains, and a permanent speed change.
- **Progression.** Learnsets as unlocks, a placement choosing four skills and one
  trait, evolution stages as an allowlist, and late-game builds as data with a
  screen of their own. A line may fork — `Stage.After` names a predecessor, so it
  is a tree — and is read **by order or by name, never both**, with `Furthest`
  refusing on a fork rather than picking an arm. A stage **name** is an
  identifier and `progression.ValidateStageName` refuses one that could not be:
  it is the key an `after`, a placement and a learnset gate each spell by hand,
  so it is drawn raw in both languages and may not be a word out of one.
- **Authoring.** `hexforge` (CLI, for pipes) and `hexforge-tui` (full screen)
  over one `internal/forge`, so the two cannot disagree. Skill authoring and
  editing, art picker, kit and allowlist pickers, budget bounds, spar, `weigh`
  and `check`. A flavour clause is authorable from the flags as well as the
  wizard, and the damage preview reads the caster's own terms rather than only
  the target's.
- **Reference screens.** Statuses, traits, elements, species, and the affinity
  chart drawn as closed ASCII loops in element colour.
- **Vietnamese.** The TUI is Vietnamese-first with an English toggle; every
  description is derived from the data, and only the flavour clause is authored.
  A status **category** is worded twice, because a sentence and a column want
  different parts of speech: `StatusCategory` is the predicate the statuses
  reference explains an id with, `StatusCategoryNoun` is the noun a strips clause
  names. ⚠️ English cannot fall back on the id here — a category is a Go enum,
  not a data id, so the rule that an English name is whatever the data writes
  does not reach it, and both families are held complete by a test that refuses
  an enum spelling. → `internal/i18n/forge.go`.
  A **name** goes the other way: `PassiveName`, `SpeciesName` and `BuildName`
  answer nothing in English on purpose, because the word beside the id is
  authored once and in Vietnamese, so showing it would be a leak rather than a
  translation — the English reader gets the id, which is the data's own name for
  itself. A table whose only column was that name therefore **drops the column**
  rather than padding a row of blanks, and the picker asks whether the rows in
  front of it have anything in that column rather than which language or which
  kind it is drawing. → `internal/screen/picker.go` → `detailColumn`.
  The same reading reaches the **game client's** blocks: `tui.Detail`,
  `tui.DetailPassives` and `tui.DetailStatus` all word a heading through the
  accessor, so the three agree by inspection. ⚠️ `DetailPassives` read its field
  raw for as long as it existed, twelve lines under a caller that did not —
  **latent**, because its one caller hardcodes `i18n.Vi`, and a leak the day
  `cmd/hexarena` honours `--lang`. A rule with three call sites is only worth
  the one that got it wrong quietly.
  ⚠️ **A screen the sweep never renders has no leak test, no width test and no
  translation test at all.** The species picker read `kind.Name` raw and drew
  `dragon rồng` in English for as long as it existed;
  `TestTheScreensGlossEveryDataName` is built to catch exactly that and was green,
  because `everyScreen` registered no species picker — and the skill form's
  species allowlist raises the same kind, so the branch had **two** unmeasured
  entrances. Fixing a leak of this shape is therefore two things: the branch, and
  the entry that would have caught it. The same probe registered the origins
  picker, whose title line no other screen draws.
- **The opponent.** `Suggest` prices statuses, buffs, guards, heals, cleanses,
  kills, summons and **tempo** in damage over capped horizons — tempo off the
  speed stat, never off the queue. Both halves of an all-sided skill, a tie
  broken by what an option costs to have spent rather than by kit order, and the
  two costs of acting: what a skill does to its own side and **what the units it
  hurts answer with**. The roster was re-levelled once along the way, which is
  what makes every rate quoted anywhere comparable.
  ⚠️ The reply cost is the one term landed on a **null** measurement — 499‰
  against the same rating without it, 10,000 seeds, band ±8‰ — because the
  shipped board's only answering trait is a 4% poison on one unit. Given every
  unit a reply worth having it reads 513‰ and 607‰, so what is null is the data
  and not the term; → `CLAUDE.md` § Rating an action.
- **Measuring the opponent.** `forge.Bout` fights two ratings head to head over the
  same seeds from both ends of the board, on an **exactly even** control it refuses
  to print a figure without, against a frozen ruler (`FirstUsable`) that may never
  be improved. `Suggest` beats it **81.3%** over 10,000 seeds, band ±0.8pp, and
  finishes sooner (45 turns against the control's 47). ⚠️ That figure is a reading
  of a **board**, not of the rating alone: it was 77.9% before the first crit
  chances landed, and the rating did not change. Re-take it after a data change
  rather than quoting the last one.
- **Balance.** Every enemy-aimed range re-read under the rank rule, and each ace
  moved to its own back column behind a screen — 27.6% ally → **47.3%**. Both were
  data answers, and the formation is guarded by a test rather than by whoever
  edits the file next. ⚠️ The levels were deliberately **not** touched: the 20..30
  dial spans 40–82% on the screened board, so it is not the lever it looks like.
  A field is priced with `weigh` against a copy of its own carrier, because the
  roster win rate is **non-monotone in ally damage** and prices nothing. The
  first crit chances were authored that way: `razor_leaf` and `wind_shuriken` at
  200‰, worth **+8.4%** and **+6.9%** to their carriers, while the same chance on
  `bite` was worth +0.2% and on `kunai` **−1.7%**.
  `weigh --carriers all` takes that table in one command, one row per carrier
  that brings the skill, each with its own control and its own refusal — and with
  **no headline figure**, because two rows are prices against two different
  opponents and an average of them has no referent.
  → `CLAUDE.md` § Pricing one number.

- **Squads.** A side is built in the TUI — who is in it, what each brings, where
  each stands — saved to `squads.json`, and fought against another over N seeds
  **both ways round**, which is what makes a mirror read exactly even instead of
  reporting the first slot's advantage as the squad's — or played by hand against
  the engine, one battle at a time, with undo — and written out as a log the
  game client replays and verifies.
  Standing somewhere is a **picture**: the 3x3 is drawn from the member under
  edit, so the mark moves with `←/→` in the same draw rather than jumping once
  the member is left; the front rank is marked on the grid itself, off
  `hex.Ranks` rather than off a column number; and the slot row reads the cell
  *and* the rank. ⚠️ The drawing commits nothing: `s.editing.Units` is shared
  with every model copied off this one, so a write from inside a drawing would
  reach all of them. The discard guard is a **comparison** against the squad as
  it was last written (`placement.Squad.Equal`) and not a flag — the flag was
  raised by `commit()`, which runs on the way out of every member, so merely
  opening one claimed a change. → `README.md` § Building a squad.

- **The battle screen budgets its own body**, because it never could fit the
  window the tool declares. At 120x24 the body has **twenty** rows, and heading +
  board (10) + roster (1 + one a unit) + order + option list (1 + one an option)
  comes to 20 at a 1v1, 24 at a 3v3 and **28** at a 5v5 — `hex.MaxTeamSize` is 5,
  so 28 is the floor for a legal squad before a single blank or log line, and a
  summon puts units on the board past the five a squad brought (`board + roster =
  29` on its own). So "which rows does it give up to fit 24" had no answer, and
  **the defect was where the cut landed**: `frame` cuts from the bottom and the
  option list was the last thing the body wrote, so the one thing a player has to
  see in order to act was the first thing thrown away. `playFit` reserves the
  heading and the turn in front and hands the rest what is left, in a stated
  order — save note, roster (clipped a row at a time), board (dropped whole),
  order line, log — with one dim line naming what is not shown. ⚠️ What
  disappears is **not monotone** in the height: the board is ten rows or none, so
  where it still just fits it takes the rows the order line would have had. Only
  the *offering* order is monotone, which is what the test asserts. ⚠️ A second
  defect found measuring it: `playLogLines` capped **events** and not lines, and
  `tui.Line` opens a turn with a blank row of its own, so eight events measured
  **eleven rows**. ⚠️ No per-screen floor was added — `minHeight` already is one,
  and at 24 the budget still holds a 5-a-side roster whole under the option list.
  → `CLAUDE.md` § *Where a form beats a prompt* → the played battle, *the
  budget*.

- **A list of three reads as a sentence.** `i18n.listed` is the one place that
  knows the shape — conjunction before the final item, comma between the rest —
  and `join` (prose) and `JoinIDs` (untranslated ids) both go through it while
  keeping their own conjunction key. ⚠️ **Both languages wanted the same shape**,
  which the item had left open: `a, b and c` and `a, b và c`, no serial comma in
  either. ⚠️ **No golden moved, before or after.** Every list the shipped books
  produce has one item or two, so the defect was reachable only through the
  fixture's `purify` (three stripped categories, the same skill that exposed the
  strips clause's width) — which makes
  `TestAListOfThreeTakesCommasAndOneConjunction` the entire guard, and the
  end-to-end one beside it fails if nothing in the books reads out a three at all.
  → `CLAUDE.md` § *The event log is the contract* → the description rules.

## Not done

- [ ] ⚠️ **A one-way mirror rate stops being a measurement above one unit a
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

- [ ] **PvP over a LAN — 3v3 or 5v5, one server and n clients.** Squads built and
      saved on the player's own machine, a room joined by a code and an optional
      password, the battle resolved on the server. The design is settled and
      written down: the client is a **mirror** that runs the engine off the
      decisions the server sends, a match is a **series the room configures —
      bo1 or bo3** — fought both ways round, a room code carries its own address,
      and the reference
      screens move out of the authoring tool into a package both binaries draw.
      → `README.md` § PvP over a LAN, which holds the reasoning and the
      measurements and is the place to argue with, not this list.

      The items below are in dependency order, and the four in **Groundwork** are
      the ones nothing else can start without. **All four are done, and so is the
      room** — `internal/room` is a state machine over `internal/wire` messages
      with no I/O and **no clock** in it. What is left of *The room* is the
      registry (one goroutine per room), writing a finished match out as a
      `battle.Log`, and two things the protocol cannot currently say: a capped
      battle and a forfeit. The next item to pick up is either the registry or
      the WebSocket under *The wire*.

      **Groundwork**
      - [x] Factor the reference screens out of `cmd/hexforge-tui` into a package
            both binaries draw, and stand up a full-screen client over it.
            **Done** — eleven screens are in `internal/screen` and
            **`cmd/hexarena-tui`** draws them: a menu, seven catalogues and a
            battle. ⚠️ **The binary is `cmd/hexarena-tui` and not `cmd/hexarena`**,
            which this item used to say: `cmd/hexarena` is the *verification*
            contract — `--replay --verify` re-runs a log from its seed and checks
            every event, and `--auto`/`--log` are the scriptable half — so it is
            untouched, and the pair follows the house rule the authoring tools
            already do (a CLI for pipes, a `-tui` for the screen, one package
            under both).
            ⚠️ The tests were the hard half exactly as predicted, and what the
            new client was born with is the answer: its own `everyScreen`, its
            own wording walker (the third copy — the walker reads its own package
            directory only, so the ban does not follow code into a new package),
            its own golden, and one thing the authoring tool's sweep has never
            had — `screenCount`, a **count** the sweep is held against, so a view
            added without an entry or a written exclusion is a red test rather
            than a screen nothing measures.
            ⚠️ Two catalogues are left out on purpose and both are gaps rather
            than half-finished screens: **`BuildsScreen`** is the eighth listing
            `internal/screen` owns and this client's menu is the seven the step
            asked for, and the **art preview** is registered in no sweep because
            it draws rasterised art and what an entry would assert about it is
            still open (the same decision the other two records take).
      - [x] `internal/wire`: the protocol as one stdlib-only package. The
            envelope, the three version numbers, and error **codes** rather than
            prose. A golden per message, so a wire change shows up in a diff.
            **Done** — seven kinds (`hello` `act` `pass` · `welcome` `refused`
            `start` `turn`), ten codes, an envelope of a named kind plus a raw
            body, `wire.Digest` carrying `seed.Digest`, `EventDigest` +
            `DigestEvents`, `RoomCode`, `Password`, and
            `internal/wire/testdata/messages.golden`.
            ⚠️ **It imports `internal/seed`**, deliberately: the gate has to
            compare two `seed.Digest` values so the *compiler* checks the
            comparison, and both peers already import seed (neither can compute
            its own digest without it), so the embedded data costs nothing that
            was not already paid. No cycle — seed is data over `internal/core`
            and this is a protocol over both.
            ⚠️ **`start` carries ONE roster slice, not an ally list and an enemy
            list.** `atb.Queue.Add` assigns `seq` in the order `battle.New` is
            handed its roster, so *the caller's slice order decides which side
            wins a speed tie* — worth up to sixty points in a mirror. Two fields
            would be a second statement of an order the slice already holds and
            the peer would have to re-derive the enlistment by a convention
            written down nowhere. It is exactly what `battle.Log.Roster` is.
            ⚠️ **An envelope naming no kind at all is refused** rather than
            represented, which is the opposite answer to `hex.SideNone`'s and
            deliberate: a side genuinely has a "nobody", and an envelope with no
            kind is not a message this format has. Without the refusal
            `{"body":{}}` reads as a `hello`, since that is the zero value —
            `Envelope.UnmarshalJSON` exists for that one case.
            ⚠️ The event digest frames each event as **length then bytes** and
            drops the *name* half of `seed.digest`'s framing, because an event's
            identity is *inside* its bytes (`kind` is a field) where a file's
            name is not, so a name prefix would be a second copy of something
            already in the frame. The length is kept as defence in depth and
            **no test isolates it** — `json.Marshal` escapes every quote, so no
            free-text `Note` can forge a `{"kind":"` boundary and the collision
            seed could write down cannot be built here. Not shared with
            `seed.digest`: two different framings, and a shared helper would need
            a third package both `internal/seed` and `internal/wire` import.
            ⚠️ The load-bearing test is
            `TestTheEventDigestReadsEveryFieldOfAnEvent`, which walks
            `battle.Event`'s 29 fields by reflection — a digest marshalling two
            or three of them passes any hand-written table, and would agree about
            two turns that differed in everything a reader cares about.
      - [x] The data digest — the fifteen embedded JSON files, in `go:embed`
            order, hashed as bytes, no parsing. `assets/` excluded: art cannot
            reach the simulation. **Done** — `seed.DataDigest`, and it is a
            *peer-equality* check rather than a version number: every data commit
            is supposed to move it.
            ⚠️ Each file is **framed** — name, then byte length, then bytes — and
            the reason written into the brief for it was **wrong**. A plain
            `sha256(concat(bytes))` is *not* blind to two files exchanging
            contents: the files are read in a fixed order, so a swap moves those
            bytes to different offsets and a content-only hash sees it. What a
            concatenation genuinely cannot see is a **boundary that moved** or a
            **rename**. Measured, not reviewed; the conclusion survived and the
            reason did not. → `README.md` § Three version numbers.
            ⚠️ The load-bearing test is the **walk**, not the sensitivity table:
            these fifteen names exist in three independent copies (the directive,
            the fifteen `ReadFile` calls, `dataFiles`), and dropping one name
            reddens the walk while the per-file byte-flip test stays **green** —
            it is table-driven off that same list and cannot see a file the list
            forgot. A test that checks a list against itself cannot see a
            consistent omission.
            ⚠️ No golden on the digest value, deliberately: it would move on every
            unrelated data commit while catching nothing the properties do, which
            here makes it a merge-conflict generator that measures nothing.
      - [x] Replace `Drain` at the server with an append-only record and a cursor
            per consumer. ⚠️ `Drain` **empties the buffer** and a room has two
            players, spectators and a log; the cursor is also what reconnect and
            mid-battle spectating are made of.
            **Done** — the record is kept for the battle's whole life and
            `Since(cursor) ([]Event, int)` hands out views into it, with
            `Recorded()` for a consumer that wants only what happens next.
            `Drain` is now *implemented as* a Since consumer whose cursor the
            battle keeps, so its behaviour is unchanged and its 261 existing call
            sites did not move. An out-of-range cursor **panics** — the reading
            `rng.Intn` takes of a bad bound, and the only other panics in
            `internal/core` are its two; answering with an empty slice would make
            a consumer that got *ahead* of the battle look identical to one that
            is up to date, which is the silent desync a cursor exists to prevent.
            ⚠️ Every view is a **three-index slice** (`b.events[c:n:n]`), and this
            is not defensive tidiness — an uncapped view corrupts **both**
            directions, measured: the caller's own `append` writes into the slot
            the next `emit` will use, and that `emit` then overwrites the value
            the caller appended. Both clients sit on that path
            (`internal/screen/play.go:341`+`203`, `cmd/hexarena/main.go:193`
            each assign a drained slice and later append to it).
            ⚠️ **One test in the entire repository catches it** —
            `TestAViewAndTheRecordSurviveEachOthersAppends`, confirmed by
            stripping the cap and running everything. It is also only *reachable*
            while `cap > len`, so it sweeps ten record lengths and fails if the
            sweep observed nothing, rather than asserting once and hoping.
            ⚠️ `Drain` returning **nil** rather than an empty slice when nothing
            is new is kept because keeping it is free, **not** because anything
            reads it: mutating it to an empty non-nil slice reddens three tests,
            all three new, and **none** of the several hundred existing callers.
            Worth knowing before somebody treats it as load-bearing.
            ⚠️ The record is now a **second copy** of bytes the process already
            held — both clients accumulate their own event slice and never trim.
            A room's consumers should hold a cursor and read `Since` rather than
            accumulate; the engine needs no mechanism (296 B an event, ~16 KB for
            a finished 1v1).

      **The room, with no network in it**
      - [x] The room as a state machine over messages with no I/O: two fake
            clients drive a whole match in-process. **Done** — `internal/room`,
            four inputs (`Join` · `Deliver` · `TimedOut` · `Left`) each answering
            `([]Outbound, error)`, and `TestTwoFakeClientsFightAWholeBo3InProcess`
            plays a whole bo3 in **40 ms** over 123 decisions with the two
            mirrors' digests compared on every one of them.
            ⚠️ **The room reads NO clock, which the brief did not ask for and is
            the shape everything else rests on.** A timeout is an **input** —
            the transport owns the countdown and calls `TimedOut` — so `time` is
            unimported, `TestTheRoomReadsNoClock` holds it with an AST walk over
            the package's own directory, and "three consecutive timeouts
            forfeit" is pure counting. `internal/wire/clock_test.go`'s comment
            says a room "does need a clock" and that a copy of the ban here
            "would be exactly wrong"; that expectation was wrong and the comment
            is now stale.
            ⚠️ `TestNothingHereDrainsTheBattle` is the same walk pointed at the
            **selector** `Drain`: 261 call sites elsewhere make reaching for it
            one keystroke, and it would silently take the events another
            consumer was about to read.
      - [ ] Many rooms per process. A room owns its battle in **one goroutine**
            and shares it with nothing; the registry takes a mutex, a battle
            never does. ⚠️ Deliberately **not** in the room's own commit:
            concurrency does not belong beside "this has no I/O". Nothing in
            `internal/room` is safe for concurrent use and nothing there needs to
            be; `sync` is on that package's own import ban. `wire.CodeRoomUnknown`
            is this item's refusal and no room sends it today.
      - [x] Validate a squad at the gate: `Squad.Validate`, then the format's
            size, level 60, a stage that is a **leaf** of the line, then `Take`
            (which is already the loadout check). **Done** — five rules under
            `wire.CodeSquadRefused`, and the whole gate's order is
            version → password → seat → squad, pinned by
            `TestTheGateRefusesInItsOwnOrder` with a peer wrong about two
            adjacent things per case.
            ⚠️ **Not `Furthest` and not `StageAt`** — `progression.Line.Leaves` /
            `IsLeaf` were added for it, because a leaf is a fact about the *line*
            where `Furthest` is a fact about a *level*.
            ⚠️ **The justification first written for it was WRONG**: it said a
            gate on `Furthest` would start accepting an unfinished form the day a
            stage was authored above the cap. `Line.Validate` refuses that stage,
            so the day cannot come and `Furthest(LevelCap)` **is** the tip of each
            arm by construction. Measured — substituting it inside `IsLeaf` passes
            all twenty-one tests over the predicate and its gate, and no test can
            be written that it fails. What the predicate buys is the **level that
            is no longer in the question** (the two diverge everywhere below the
            cap, and this gate asks only at 60), plus: `IsLeaf` **errors** on a
            name the line does not have rather than answering false — a typo and a
            form with something after it are different mistakes.
            ⚠️ **`politoed` is no longer "queued": it SHIPPED** (`ed79a28`), so
            `pokemon.poliwag` forks as
            `Poliwag → Poliwhirl → (Poliwrath | Politoed)` and the gate's own
            test measures both arms and the interior stage on real data. The
            fixture test in `internal/core/progression` stays because it reaches
            what shipped data cannot — an interior stage of a fork at a level
            below its child's threshold, which is where `Leaves` and `Furthest`
            visibly disagree. **`CLAUDE.md` § Open work still says "nothing
            shipped forks yet" and is stale.**
      - [x] Decide whether one squad may field the same character twice.
            **Decided: it MAY**, and the reasoning is written where the gate is
            (`squadIsFieldable`). `Squad.Validate` checks ids and slots and says
            nothing about characters, the squad builder will happily write two
            Charizards, and `Take` prefixes ids with the side so even a mirror of
            a mirror reads apart in a log. A gate that refused it would refuse a
            player their own saved squad for a reason no screen has ever told
            them, and the screen that would have to start telling them does not
            exist. ⚠️ The measurement that argues the other way — that two copies
            of one character is the strongest squad available — has **not** been
            taken; refusing a shape on a hunch is what this repository does not
            do. `TestOneSquadMayFieldTheSameCharacterTwice` stops it being
            "tightened" later.
      - [x] The clock: the allowance a prompt gets, a timeout passing with a
            single constant reason. **Done** — `room.TimeoutReason` is that one
            constant and `room.TimeoutLimit` is three.
            ⚠️ **Never a timestamp into the battle**, and now never a *reading*
            either: the room is told. A `Skipped` prompt starts no clock because
            the room walks past one itself and never leaves it open —
            `TestASkippedPromptStartsNoClock` asserts that over a whole match and
            holds it against `Room.Skipped()`, a count exposed precisely because
            a skipped turn produces no decision and therefore no message, so
            without it the claim would be held by nothing.
            ⚠️ A `TimedOut` on a seat nobody is asking is **refused and not
            counted** — otherwise a transport reporting a spurious timeout could
            forfeit a player through the back door
            (`TestATimeoutOnNothingIsRefusedAndCountsNothing`).
            ⚠️ A voluntary pass leaves `Decision.Reason` **empty** and lets
            `battle.Pass` supply "passed", so the room adds no second spelling of
            it. An **illegal** act does not reset the miss count: a peer that
            could clear its tally by sending nonsense would never be forfeited.
            ⚠️ The timeout reason is **not glossed** — `tui.Line` prints
            `event.Note` raw, so today it reads `timeout` in both languages. That
            is the wordings item under *The client*.
      - [x] A forfeit, a disconnect and a refused join are results of the
            **match**. **Done** — `room.Verdict` (`unfinished` · `won` · `drawn` ·
            `forfeited`) and `room.Forfeit` (`none` · `timed_out` · `left`),
            deliberately **not** called an outcome so nobody writes
            `battle.Outcome(result.Verdict)`. Both routes to a forfeit are
            reached by `TestAForfeitAndADisconnectAddNothingToTheBattlesOutcomes`,
            so neither is a dead branch, and that test holds
            `battle.OutcomeCount` against a **literal 4** — reading the constant
            and comparing it to itself would agree with any number at all.
            ⚠️ **The protocol has no message for a forfeit**, so the room sends
            *nothing* and the transport closes. That is a gap rather than a
            decision — see the wordings item, which already lists "opponent
            left" — and a message for it is a protocol bump.
            ⚠️ `Left` before the first battle **frees the seat** instead of
            forfeiting: there is nothing to give up yet. A reconnect window sits
            in front of `Left` rather than inside it.
      - [x] A **series**, not a bo2: `battles: N` plus a rule for what ends it,
            from the room's first line. **Done** — a seat holding more than
            `Battles/2` ends it, otherwise every battle is fought and a series
            with no leader is `VerdictDrawn`. **bo1 is not a special case — it is
            N = 1**, and `Config.Validate` refuses an even series **by name**,
            saying which tie it would have to invent a rule for. The
            aggregate-health tie-break is **dropped**: no invented metric ships
            anywhere in the package.
      - [x] One rule for bo1 *and* for the third battle of a bo3, which are the
            same problem: **the seed picks the side**. `Config.HomeFor` alternates
            every battle that pairs off (`2 * (N/2)` of them, which is none of a
            bo1's) and reads the low bit of the battle's own derived seed for the
            one that does not. Honestly uncancelled, and it says so.
            ⚠️ The sharpest test of it is that **a bo3 and a bo1 on the same seed
            disagree about battle one** — half of an alternating pair in one, the
            uncancelled battle in the other — which is what a bo1 written as its
            own branch would get wrong.
            ⚠️ **The seed derivation is `sha256(seed ‖ index)`, not one round of
            `rng`**, and the obvious version was written, measured and thrown
            away: splitmix64 advances by adding a constant, so
            `rng.New(Seed + index).Next()` is a function of the **sum** and
            battle two of a match seeded 6 *is* battle one of a match seeded 7 —
            two different matches sharing a fight, exactly. Every counter-based
            generator has that shape, so a derivation from two numbers needs a
            function of two numbers.
      - [ ] The lead of each contested speed group alternates, on top of the seed
            picking the side. ⚠️ **Deferred on purpose and not forgotten.** It
            needs the roster slice composed against the queue rather than as the
            squads were authored, the side is worth **up to sixty points** in a
            mirror, and what it is worth at 3v3 or 5v5 is **unmeasured** — the
            two-unit mirror reads 49.6% for alternating against 54.2% for
            ally-first, and above one unit a side a one-way rate is not a
            measurement at all. Not something to implement on a hunch. The room
            leaves the roster slice order as the squads were authored.
      - [x] The per-turn allowance belongs in the room's configuration beside the
            format. **Done** — `Config.Allowance`, seconds, handed to both clients
            on `wire.Welcome` and never counted down here.
            `room.DefaultAllowance` is 90 because that is what was decided, and
            it is *configuration* because the argument is not settled: at 34–55
            decisions a battle it is 68 minutes a battle and 3.5 hours for a bo3.
      - [x] A turn cap per battle so a stalemate ends. **Done** —
            `Config.TurnCap`, default `room.DefaultTurnCap` = 400, and it needs no
            new `battle.Outcome`.
            ⚠️ **The room does not stamp `Stalemate` on a battle it stopped.** The
            engine concluded nothing about that battle, and a room writing an
            outcome the engine never produced would be a second reading of how a
            battle ends — and the eventual log would fail its own `--verify`. So
            `BattleResult.Outcome` stays `Undecided`, `BattleResult.Capped` says
            what happened, and the standing counts it as the draw it is. That is
            all "the outcome already carries the draws" buys.
            ⚠️ **The cap is checked where the room would otherwise ASK somebody**
            — after the skipped test, never before it. A mirror only stops at a
            turn it is asked to decide, so capping mid-run of skipped turns would
            leave the room's event run one short of the mirror's and report a
            divergence that was not one. Skipped turns still count towards the
            cap; they just cannot be the turn it bites on.
      - [ ] A capped battle is **invisible to a mirror**. `TurnCap` is not on the
            wire, no `Ended` event is emitted, and the design record's rule is
            that a client learns each battle's outcome from its own `Ended`. So a
            client is left holding an open prompt on a battle the room has
            stopped, and the next thing it hears is a `start` (or nothing). Two
            ways out, both bigger than a fix: carry the cap on `wire.Welcome`
            (a protocol bump), or make it a **constant** both peers read, which
            costs the host the setting.
      - [ ] Write each finished match out as a `battle.Log`, which makes every
            PvP match `--replay --verify`-able for nothing. ⚠️ The room holds no
            second copy of the events for it — a log writer is another cursor
            over `Battle.Since`, which is exactly why the room reads it that way.

      **The wire**
      - [ ] WebSocket transport, the dependency confined to one boundary.
      - [ ] Room code: base32 of a four-byte address and a two-byte port, ten
            characters, with a round-trip test.
      - [ ] Room password: constant-time comparison, never logged. Documented as
            what it is — a gate against strangers on the network, **not**
            security.
      - [ ] A seat token and a rejoin, which the cursor makes cheap.
      - [ ] One end-to-end test over a loopback listener, two real clients.

      **The client**
      - [ ] The mirror driver: `battle.New` off the seed and the two rosters,
            then `Replay` one decision at a time with a nil fallback. Compare the
            client's own event digest against the server's every turn, so a
            divergence is loud on the turn it happens.
      - [ ] Undo **off** in PvP. ⚠️ It works by replaying a truncated script, and
            the opponent has already seen the events it would take back.
      - [ ] A player squad file under `os.UserConfigDir()`, separate from
            `internal/seed/data/squads.json` — that one is the game's own data,
            edited by the authoring tool, and a player has no business in it.
      - [ ] Lobby, room and waiting screens — **registered in `everyScreen` in
            the same commit that adds them**, for the reason at the top of this
            list.
      - [ ] The countdown: a remaining duration on the wire rather than a
            deadline, because two machines on a LAN have no reason to agree what
            time it is. Both clocks drawn, so a player can see the other one
            thinking.
      - [ ] Re-take `playFit`'s budget. ⚠️ A 5v5 body already measures 28 rows
            against the 24 the floor gives it, and PvP adds a clock row and a
            waiting row on top.
      - [ ] The wordings, in both books, Vietnamese composed: room, lobby,
            waiting, timed out, opponent left, squad refused, version mismatch.
            ⚠️ And gloss the new pass reason — `tui.Line` prints `event.Note`
            **raw**, so today a timeout would read `loses the turn (timeout)` in
            both languages.
            ⚠️ **`wire.CodeCount` is TEN and no client words any of them yet**,
            which is the "shipped dead" shape this repository has recorded
            several times — a refusal a player cannot be shown. It could not be
            held where the codes live: `internal/wire` must not import
            `internal/i18n`, because the whole point of sending a code is that
            the wording lives at the far end, so `TestEveryRefusalCodeHasANameAndTravels`
            holds the **count** and says in its own comment that it cannot hold
            the wording. The walk over `wire.CodeCount` against both books
            belongs **here**, in the commit that adds these lines, in the shape
            `TestEveryKeyIsWordedInBothLanguages` already has. The ten are
            `none` (never sent) · `protocol_mismatch` · `data_mismatch` ·
            `bad_password` · `room_unknown` · `room_full` · `squad_refused` ·
            `not_your_turn` · `illegal_action` · `unknown_message`.

      **Later, deliberately**
      - [ ] Spectators, which the cursor above makes nearly free.
      - [ ] mDNS room browsing, so a client can list rooms with no code at all.
      - [ ] A chess clock — a budget per player rather than per turn.
      - [ ] Prove the mirror across architectures: the same seed and the same
            digest on amd64 and arm64. Friends are not all on one machine, and
            this is the assumption the whole design rests on.
      - [ ] Read the balance again at 3v3. The screened formation was tuned at
            five a side, and a shorter board leaves a summon more free slots.

- [ ] **Graphical client with ebiten.** A renderer over `[]Event` and nothing
      more — it must not read `*Battle`. Asset pipeline undecided.
      → `CLAUDE.md` § Open work.
- [ ] **Grow the cast.** Eight ship across two origins, covering eight elements
      (grass, fire, water twice, wind, ground, light, and electric/metal on one
      character — Magnemite is the first dual affinity shipped). This is content,
      and the constraints that bound it are written down. A character moves
      `cast.golden`,
      `species.golden` and `origins.golden` — **not** `scenarios.golden` or
      `replay.golden`, which this line claimed until 2026-08-31. Read Squirtle
      first. → `CLAUDE.md` § Open work.
- [ ] **Thirty-one Pokemon are traced and waiting for a character.** The art
      landed first because `cast.ParseBook` refuses a character that declares no
      image, so the order is forced: trace, then author. Nothing references any
      of these yet, which is why they moved no golden and why
      `TestTheShippedArtIsCutOutRatherThanFramed` does **not** cover them — that
      test walks the art shipped characters name, so the day one of these is
      authored is the day its picture is first measured. The sources were checked
      by hand instead: every one carries real `tRNS` transparency, 46–71% of the
      canvas clear, the inked box well inside the frame — no baked chequer, so no
      `--decheck` was needed.
      By line, as they would be authored:
      **pichu → pikachu → raichu** · **cleffa → clefairy → clefable** ·
      **igglybuff → jigglypuff → wigglytuff** · **happiny → chansey → blissey** ·
      **mareep → flaaffy → ampharos** · **dratini → dragonair → dragonite** ·
      **gastly → haunter → gengar** · **magnemite → magneton → magnezone** ·
      **onix → steelix** · **riolu → lucario** · **mew** and **mewtwo**, which
      have no line at all.
      ⚠️ **`politoed` has SHIPPED and is off this list.** It landed in `ed79a28`
      as the forking mechanism's first user — `poliwag → poliwhirl →
      (poliwrath | politoed)` — so `CLAUDE.md` § Open work's "nothing shipped
      forks yet" is **stale**, and the PvP gate's leaf rule measures both arms
      and the interior stage on real data because of it.
      ⚠️ Each of these needs more than a picture: an origin (`pokemon` exists), a
      species claim if any skill it wants is a lineage skill, an archetype whose
      kit its affinity can carry, and a stat table inside
      `progression.Limits`' joint health-and-defence bound. **A new skill also has
      to say which story it is out of.** Authoring one is the *Grow the cast* item
      above, not a separate task — this entry is the queue, not the work.
- [ ] **`weigh` prices neither a field that is two numbers nor a skill that
      deals none.** `self_gradient` is a bonus *and* a share, so sweeping it is a
      surface where the tool answers curves — it is out of the field table for
      that reason rather than for being hard, and a second report is what it
      wants, not a ninth constant. Separately, a row that lands nothing is
      refused rather than reported as even, which is right — *worth nothing* and
      *not rated* are different answers — but it leaves every **support** skill
      unweighable: a `cooldown` weighing on `poison_powder` refuses, because
      power 0 lands nothing at all. Pricing a buff's cooldown needs a reading
      that is not a count of landings. → `CLAUDE.md` § Pricing one number.

- [ ] **The art preview is outside every sweep there is.** It is not in
      `everyScreen` (`cmd/hexforge-tui/language_test.go`) and not in
      `everyMovedScreen` (`internal/screen/screens_golden_test.go`), so it has
      **no width test, no translation test, no leak test and no golden entry**.
      ⚠️ This is the **fifth** instance of a shape `CLAUDE.md` records having been
      made four times — after `plainTerminal`, `playScreen`, the species picker
      and the skill filter's states — and it is the one instance that has been
      *known* the whole time: the `CLAUDE.md` line describing the trait blurb
      already admits it ("Both blurb shapes are in it now; `screenPreview` still
      is not") and nobody filed it. The screen draws art, so it is also the one
      screen where a width test would be measuring a drawing rather than a
      sentence; decide what the entry asserts before adding it, or it will pass on
      nothing. It moved to `screen.PreviewScreen` in the describer step and the
      question moved with it unchanged.

      **What is measured, so the decision has its input.** ⚠️ **The rasterised
      art IS reproducible here**: the whole shipped cast previewed at the level
      cap digests byte for byte identically across three separate `go test`
      processes, in both the monochrome and the coloured drawing, and twice inside
      one process off two separately loaded libraries. So a golden entry would be
      **stable on this machine** — it is `internal/forge.rasteriseSVG` and nothing
      in it reads a clock, a map or an environment.
      ⚠️ **What is NOT measured is another architecture**, and there is a named
      reason to doubt it rather than a general one: `rasterx` calls `math.Sin`
      (15), `math.Cos` (10), `math.Atan2` (6), `math.Tan` (4) and `math.Sqrt`
      (23). `Sqrt` is an IEEE-exact instruction, but the four transcendentals are
      the family where Go's standard library has had per-architecture assembly, so
      "same seed, same bytes, every machine" — the promise `internal/core` makes
      and this repository's goldens rest on — is **not** established for a
      drawing. Whether the shipped traced SVGs even reach an arc path is unknown;
      they are `vtracer` output, which is beziers and polygons.
      ⚠️ **And the size, measured before anybody argues about taste.** One
      character, one language, monochrome: **19 lines / 2.0 KB** at 120x24 and
      **55 lines / 8.4 KB** at 160x60. Coloured, the *same* 19 and 55 lines are
      **14 KB** and **128 KB**, because every cell carries its own escape
      sequence. So a plain-text entry is affordable and a coloured one is not —
      which also means a golden taken under `NO_COLOR`, as both of them are,
      would record the ramp and leave `blockCell` measured by nothing.
      ⚠️ Today `internal/screen/preview_test.go` is the whole of what any suite
      says about the drawing, and it only asserts *ink versus blank*: measured,
      swapping the red and green luminance weights and swapping `▀` for `▄` each
      left `go test ./...` **entirely green**.
- [ ] **A saturating multiplier is re-narrowed one line downstream.** #185 let
      `combat.Swung` hand back `math.MaxInt64`, and three sites take that value
      into plain `int` arithmetic: `internal/core/battle/turn.go` multiplies the
      power by the splash share (`power * SplashPower / scale.Base`), the same
      file does it to a restore, and `internal/core/battle/ai.go` does it to the
      rating's power. Same class as #180 and #185, and **strictly better than
      before either of them** — the value arriving there used to be a wrapped
      negative — so this is the tail of that work rather than a new defect.
      ⚠️ Fixing it is not four more widenings: the question worth answering first
      is whether a saturated multiplier should be *carried* at all, or refused
      where it is produced.
- [ ] **`combat.ExpectedStrike` weights in a narrow `int64` product**
      (`internal/core/combat/combat.go`): `ordinary*(PermilleBase-chance) +
      critical*chance`, with both `Strike` results now able to reach
      `math.MaxInt64` after #180. Rating-only — it never reaches a golden — which
      is exactly why nothing would report it.
      ⚠️ **#226 added a second product on the same path and it belongs to this
      item**: `Rules.Expected` is now `ExpectedStrike(h) * ExpectedStrikes() /
      PermilleBase`, and the multiplier there reaches the strike cap in per mille —
      about ten thousand — so a saturated `ExpectedStrike` wraps one line sooner
      than it did. Same class, same fix, and whatever answers the question above
      ("carry a saturated multiplier, or refuse it where it is produced") answers
      both.

- [ ] ⚠️ **A declined turn makes a slow board slower, on a wall-heavy roster.**
      Measured 2026-09-03, before the block-charge clause landed: `forge.Bout` on a
      board of two `carapace` walls carrying `withdraw` a side leaves **175 of
      800** battles undecided with #234's pass rule off and **308 of 800** with it
      on — and with it on the refusal comes from the CONTROL, `Suggest` against
      itself. The shipped roster is unaffected: four declines over two hundred
      battles, every one resolved.
      ⚠️ `frozen()` cannot call such a board a draw, because a unit holding a
      self-aimed utility can still aim at something, so the turn limit catches it
      instead — and a battle that ends on the limit reports nothing about what
      happened. Recorded rather than acted on: the board is a constructed extreme
      and it was already refusing before the rule. **It was written down once and
      lost in a rewrite of the entry above it**, which is the only reason it is
      dated to a session that had already finished with it.
- [ ] **`m.wrapped` fills the window's final column.** `wrappedIn`
      (`cmd/hexforge-tui/model.go`) computes `room = usableWidth() - 2 - width -
      1`, so `labelAt` emits exactly `usableWidth()` cells — measured at 120 on
      `browse`'s biography row: exactly 120. Every other row in this client
      deliberately leaves the last column empty, because a line filling a
      terminal's final cell wraps on some of them, and `fieldValueRoom`'s comment
      records fixing this same off-by-one in four other places. ⚠️ It changes
      what fits, so it would newly mark a line #186 does not cut today — which is
      why #186 left it alone rather than folding it in.

- [ ] **The committed cast is not in the form the tool writes, and the test that
      was supposed to catch that is vacuous.** `CLAUDE.md` says `cast.json` and
      `origins.json` are committed exactly as `Book.Marshal` writes them — sorted
      by id — so that `hexforge new`, which rewrites the whole file, produces a
      one-block diff instead of a whole-file one. Measured 2026-08-31: **neither
      is.** `cast.json` is in declaration order (`naruto.naruto` fourth where
      Marshal puts it first) and `origins.json` reads `pokemon` then `naruto`, so
      the next real `hexforge new` reshuffles both.
      ⚠️ **`TestWrittenCastIsStableAndReloads` cannot fail on this**, which is
      why it went unnoticed: it reads the file out of the **scratch** directory,
      and `scratchData` → `testfixture.Inject` has already rewritten that copy
      through `SaveCharacter`/`Marshal`. It compares Marshal's output against
      Marshal's own output and would pass whatever is committed. Two halves to do
      apart: **reformat the two files** (a large diff that says nothing, so it
      wants a commit of its own and no other change riding along), and **point
      the test at the committed file** so the property is held rather than
      asserted. Doing the reformat without the test fix buys one tidy day.

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

## Decided against — do not re-raise

- **Re-rolling the turn-order tie-break from the seed.** Kept, and the reason
  first written here was **wrong**: it claimed this needs `atb.Queue.order`
  changed and would invalidate every balance figure. It does not. `seq` comes off
  a counter in `atb.Queue.Add`, which `enlist` calls once per unit, which `New`
  calls **in roster-slice order** — so the tie is the *caller's* to decide, with
  no core change and no golden moved. `forge.FightSquads` writing
  `append(ally, enemy...)` is the whole reason ally wins every tie today.
  Measured on the two-unit mirror, 2000 seeds: 54.2% ally-first, 45.7%
  enemy-first, **50.2%** with one coin for the side, **49.6%** alternating pair by
  pair. The lever works and costs nothing. It is still not the answer, because
  the entry below says evening the ties does not make a battle even — a match
  fights both ways round, and the coin is worth having *on top of* that.
  → `README.md` § PvP over a LAN.

- **A ceiling on `Skill.Power`.** The arithmetic that looked like it demanded
  one is gone: `Rules.damage` builds its numerator in 128 bits now, so nothing
  wraps and nothing panics, and the guard that saturates at `math.MaxInt64` is a
  bound on the **type** rather than on the design — `max_effective_hp` is 11,500,
  so anything reaching it already kills whatever it touches. Measured at the worst
  reachable factors (attack 2399, affinity 2000, crit 1250, K 300): the old
  expression held a power up to **5,126,231**; the new one holds
  **1,537,869,451,747,357,366**; the largest in the shipped book is **2,400**
  (`solar_beam`). The book therefore sits about 4×10¹⁴ below the guard.
  `combat.Swung` was the one place left where a power was still multiplied in a
  narrow type, and it is widened on the same terms — measured at the worst swing
  the book declares (a bonus of 1,200 and a share of 900), the old expression
  held a power up to **4,854,406,335,185,524** and the new one holds
  **4,854,406,335,186,722,908**, against a largest landable multiplier of
  **3,500** (`inferno`). So the arithmetic has stopped asking everywhere a power
  is read, and not only in the damage formula.
  ⚠️ **So a ceiling now would be an implementation limit dressed as a design
  bound**, which is the distinction the settled stat-bounds policy turns on: a
  ceiling states what an **author** may write at the cap, and there is no design
  argument for any particular number here — only an arithmetic one, and the
  arithmetic stopped asking. It is deliberately *not* a typo guard either: that
  is a different feature with a different justification, and it should be raised
  as one if it is wanted, rather than smuggled in as overflow protection.
  → `CLAUDE.md` § *Saturate continuous values, cap discrete ones*; the 128-bit
  numerator and its guard in `internal/core/combat/combat.go`.

- **Waiting — passing a turn because the next one is worth more.** It is
  **arithmetically empty in this engine**, not under-built: `spendCooldowns`
  decrements *every* cooldown at the end of an act, a pass and a stunned turn
  alike, so the skill being waited for comes back on the same turn either way and
  acting dominates waiting by exactly what the action is worth. The two available
  lookaheads each break a rule of `price.go` — one rolls, the other is a second
  copy of the resolving arithmetic — and either costs about ×36 a turn.
  ⚠️ **This is NOT the pass that now exists.** `Suggest` declines a turn when
  every available option is worth nought and the cheapest of them would start a
  cooldown, and that is the opposite question: not "wait to get a skill back" but
  "do not spend one on nothing". The arithmetic above is what makes it sound —
  an act pays its own cooldown and a pass does not — so the two live together.
  → `CLAUDE.md` § Rating an action; `TestAPassBuysNoCooldownAnActDoesNot`,
  `TestATurnIsDeclinedRatherThanSpentOnACooldownForNothing` and
  `TestATurnIsGivenUpOnlyForAReasonThatIsWrittenDown`.
- **Rebalancing `reckless`.** All three levers the item named are measured and
  dead, and the trait is kept as it stands: an extreme trade that loses badly to
  one matchup and wins nearly everything else — **22.1%** against the fire line
  and **96.6% / 93.0%** against the rest of the cast — which is a legitimate
  shape rather than a bug. Dropping `bare`'s dodge clause is worth **+2.8**;
  pairing it with a `vulnerability` is worth **−3.3**, *below* where the trait
  started; and softening the defence magnitude cannot land, because **the two
  gates flip at the same two-point rung** — the duel clears the 38.9% floor
  between −95 and −90 and the cast-wide pair saturates squirtle at 100% across
  exactly that step. Every amount that makes the matchup real makes `reckless`
  the 100%-against-the-cast trait `blood_thirst` was refused for being.
  ⚠️ The dial is also a quarter as long as it reads, because a stat
  **saturates**: a −400‰ term on a base of 400 fights at **290**, not 240, so the
  whole reachable range of that lever is 290–391. Do not re-raise any of the
  three. What is left, if anything, is a *different kind* of cost — one the duel
  prices and the cast-wide matchups do not — or the possibility that the 22%
  belongs to `inferno` and the fire line's detonate rather than to this trait,
  which nobody has measured. ⚠️ `vulnerability` therefore **still has no shipped
  user**; the mechanism is proven end to end but a negative share on top of an
  unchanged defence term makes the build strictly worse.
  → `README.md` § *What `bare`'s dodge clause was worth* and § *What softening
  `bare`'s defence was worth*.
- **The queue as a third tie-break key.** Built, measured and **thrown away**.
  `take` was given a key under `cooldown` — when value and cooldown are both
  level, take the aim whose occupant acts soonest — and it moved **0 of 93,320
  decisions** over 2,000 shipped battles. Against the rating without it: **500‰
  with 10,000 wins and 10,000 losses**, which is not a narrow result but the
  *control signature* — the two ratings played the identical battle every time,
  and no golden moved. ⚠️ The premise was wrong, not the code: "one skill pointed
  at several cells has the same cooldown on every call, so the winner is whichever
  cell `hex.Cells` lists first" is true, but a tie needs the **values** level too,
  and shipped units differ in health, defence and affinity, so two aims almost
  never rate to the same integer. A census found **0 prompts** tied on both.
  Do not rebuild it without first showing the tie exists on the board in hand.
  The rule it was built under is worth keeping and is kept: **a queue reading may
  be compared, never added or multiplied** — a value that reaches an arithmetic
  expression is tempo, and tempo is priced from the speed stat.
  → `CLAUDE.md` § Rating an action.
- **Wording the ids on `cmd/hexarena`'s menu line.** `tui.Extras` prints
  `strips control/dot x1` beside `health <=50%` and `cd3`: it takes no
  `i18n.Lang` at all, and every field on that line is an id or a raw number. So
  the enum spellings there are **not** the defect the strips clause had — that
  one was an i18n'd sentence falling through to an id. Translating this line
  means translating the line, not patching a lookup.
- **`at_stage` on a learnset entry.** Unblocked and deliberately not built:
  `at_stage: "Ivysaur"` is exactly `stages: ["Ivysaur","Venusaur"]`, and two
  vocabularies for one idea is the cost. → `README.md`.
- **A character class.** An archetype's curve and kit already say what a class
  name would, and an archetype has **no** mechanical effect. Do not add one
  without answering what it does that a curve and a kit cannot.
- **A dependency ban.** Written, then removed on the author's instruction.
  `internal/core` importing nothing outside the standard library is the rule
  that matters; the rest of the tree may use what it needs.
