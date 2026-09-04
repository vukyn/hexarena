# TODO

A short index of what is done and what is not. It is deliberately **thin**:
nothing here explains a design, because the explanations already live in
`CLAUDE.md` (the constraint each piece has to respect) and
`README.md` (§ *What each question cost to answer* — the detail behind each one).
This file exists so
that "what is left" can be read in a minute instead of found in 300KB of prose.

⚠️ **This file goes stale and the repository is built not to tolerate that.**
Everything else here is derived — descriptions come from the data, the affinity
chart is drawn from the chart, goldens are regenerated. A hand-kept list is the
one thing that can quietly become a lie.
**When you finish something, tick it here in the same commit.**

⚠️ **"One line per item, and a pointer" is what this said until 2026-09-05, and
it had not been true for a long time** — the open entries below run to twelve
kilobytes each because they carry the measurements that settled them, and that
is the file working rather than failing. What *was* failing is now fixed: the
headings say what they hold. **`## Not done` is only what is not done.** Finished
work and its reasoning moved to `docs/decisions.md`, which is where to look for
why a shipped thing is the way it is.

## Done

The honest record is the git history — every entry below is a merged PR and
`git log --oneline` is the full list, in order, and cannot go stale. The grouping
is only so the shape is readable.

- **Engine.** Turn-based battle on an odd-q hex grid, action-value turn order,
  elemental chart. Verifiable logs — a cell and an aim are written only where
  there is one — replay, undo. Draws for a battle nobody can act in and for a
  deadlock. Piercing, healing, draining, regeneration. Conditions read the target
  *or* the caster, as a threshold or as a gradient that grows with the caster's
  own wounds, and paying in power or in health — a reserve buys a heal as well as
  a blow, and a caster holding no fuel heals nothing without a cast gate being
  written. Reach counted in ranks from the far side rather than in cells from
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
  and not the term; → `docs/balance.md` § Rating an action.
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
  → `docs/balance.md` § Pricing one number.

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
  → `docs/architecture.md` § *The event log is the contract* → the description rules.

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
      with no I/O and **no clock** in it — **and so is the registry**, which is
      one goroutine per room around it and reads no clock either. The two things
      the protocol could not say are both **closed**: a departure sends
      `wire.Closed` and a capped battle is reached off `wire.Welcome.TurnCap`. So
      what is left of *The room* is writing a finished match out as a
      `battle.Log`. The next item to pick up is the **WebSocket** under *The
      wire*. ⚠️ The one-listener-per-room question that item used to carry is
      **decided** — one listener per process, and the room code gained a byte for
      the room — so what is left of it is the socket.

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
            ⚠️ One catalogue is left out on purpose and it is a gap rather than a
            half-finished screen: **`BuildsScreen`** is the eighth listing
            `internal/screen` owns and this client's menu is the seven the step
            asked for. (The **art preview** was the other one and is registered in
            every sweep now, on a linear character *and* on a forking one.)
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
            unimported and `TestTheRoomReadsNoClock` holds it with an AST walk
            over the package's own directory. ⚠️ **Nothing is counted**: a
            timeout announces and passes the turn, and the three-strike forfeit
            that used to be "pure counting" is gone with the whole concept — see
            the clock item and the abandonment item below.
            `internal/wire/clock_test.go`'s comment
            says a room "does need a clock" and that a copy of the ban here
            "would be exactly wrong"; that expectation was wrong and the comment
            is now stale.
            ⚠️ `TestNothingHereDrainsTheBattle` is the same walk pointed at the
            **selector** `Drain`: 261 call sites elsewhere make reaching for it
            one keystroke, and it would silently take the events another
            consumer was about to read.
      - [x] Many rooms per process. A room owns its battle in **one goroutine**
            and shares it with nothing; the registry takes a mutex, a battle
            never does. **Done** — `room.Registry`, keyed by `wire.RoomCode`:
            `Open` · `Join` · `Deliver` · `TimedOut` · `Left` each answering
            `(Answer, error)`, plus `Read` · `Close` · `CloseAll` · `Wait` ·
            `Count` · `Running` · `Codes`. `wire.CodeRoomUnknown` is what an
            unknown code answers, which **closes a code that shipped dead**:
            nothing in the repository sent it before this.
            ⚠️ **It is in `internal/room` rather than a package of its own**, and
            both reasons are mechanical: `TestTheRoomReadsNoClock` walks the
            package's own directory, so a registry beside the room inherits the
            clock ban rather than needing a second copy of it, and the `*Room`
            never has to be exported out of the package at all.
            ⚠️ **A request on the channel is a VALUE, never a `func(*Room)`.** A
            closure lets the caller capture the pointer and keep it, which
            defeats the whole invariant while reading as the tidier design. So a
            small discriminated `request` travels and `answerFrom` is the one
            place a `*Room` is called.
            ⚠️ **The mutex guards the map and NOTHING else** — `lookup` is the one
            place a request path locks and it releases before `ask` sends, or
            every room in the process would serialise through one lock: the letter
            of the rule kept with its point lost, and invisible to every test.
            `TestNoLockingFunctionSendsOnAChannel` is an AST walk holding it, and
            it is a **reachability** analysis rather than a per-function one
            because the mutation that hides from that is a locking function that
            merely *calls* the sender. Measured: holding the mutex across the send
            reddens it in half a second by name, and **deadlocks** the in-flight
            test (a retiring room needs the mutex the blocked sender is holding) —
            two catches, and the walk is the one that says why.
            ⚠️ **`sync` came OFF that package's import ban and the entry's stated
            reason is why**: it read "the registry takes the mutex", written when
            the registry was expected to live elsewhere, so it would have refused
            the one file it was written to accommodate. The claim survives
            *sharper* as `TestNoRoomMethodTouchesTheMutex`, which refuses a mutex,
            a channel and a goroutine on any method of `Room` **by receiver** —
            an import ban cannot say which type may lock. `TestTheRegistryHandsOutNoRoom`
            is the other guard: no exported method's type graph reaches a
            `*room.Room`, in or out.
            ⚠️ **The registry reads no clock either.** `TimedOut` is *forwarded*
            and nothing starts a timer: whoever owns the transport owns the
            countdown, and the transport is the next item.
            ⚠️ **A room retires its own entry the moment its match ends**, which
            forced the API: the protocol has no "the match is over" message, so a
            transport asking afterwards would be asking about a room that had
            already gone. Hence the `Reading` rides on the `Answer` to the input
            that ended the match. `Wait` closes nothing, deliberately — that is
            what makes "no goroutine is left behind" a measurement rather than a
            tidy-up.
            ⚠️ `make check` now runs this package a second time with **-race**:
            3.9s against a gate of about a minute, and a race test nobody runs is
            not a net.
            **Still not in it**: the WebSocket, the clock, writing a finished
            match out as a `battle.Log`, spectators.
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
            visibly disagree. **`docs/decisions.md` § *An evolution line that forks* still says "nothing
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
            constant, and **nothing counts**.
            ⚠️ **Never a timestamp into the battle**, and now never a *reading*
            either: the room is told. A `Skipped` prompt starts no clock because
            the room walks past one itself and never leaves it open —
            `TestASkippedPromptStartsNoClock` asserts that over a whole match and
            holds it against `Room.Skipped()`, a count exposed precisely because
            a skipped turn produces no decision and therefore no message, so
            without it the claim would be held by nothing.
            ⚠️ **`room.TimeoutLimit` is GONE**, with the per-seat tally and the
            three-strike branch. A timeout announces and passes the turn; the
            pass is what makes the match progress, and it is all the input buys.
            ⚠️ A `TimedOut` on a seat nobody is asking is still **refused**, and
            the refusal now protects something better: with no tally to reach
            through the back door, what a spurious report would otherwise do is
            **spend the other player's turn for them** — a real decision into the
            battle and into the log
            (`TestATimeoutOnNothingIsRefusedAndSpendsNobodysTurn`, both shapes:
            nobody on turn, and the wrong seat).
            ⚠️ **A timeout needs no message**, measured rather than assumed: the
            pass carries `room.TimeoutReason`, the reason is part of the
            `battle.Decision`, and the decision rides on `wire.Turn` where
            `Decision.Reason` is `json:"reason,omitempty"`.
            `TestATimeoutTellsTheMirrorWithNoMessageOfItsOwn` encodes and decodes
            the room's own answer and reads the reason off the far end.
            ⚠️ A voluntary pass leaves `Decision.Reason` **empty** and lets
            `battle.Pass` supply "passed", so the room adds no second spelling of
            it. An **illegal** act leaves the turn open, because it is not an
            answer.
            ⚠️ The timeout reason is **not glossed** — `tui.Line` prints
            `event.Note` raw, so today it reads `timeout` in both languages. That
            is the wordings item under *The client*.
      - [x] A **departure**, a disconnect and a refused join are results of the
            **match**. **Done** — `room.Verdict` (`unfinished` · `won` · `drawn` ·
            `abandoned`), deliberately **not** called an outcome so nobody writes
            `battle.Outcome(result.Verdict)`, with `Result.Departed` naming the
            seat that went away.
            `TestADepartureAddsNothingToTheBattlesOutcomes` holds
            `battle.OutcomeCount` against a **literal 4** — reading the constant
            and comparing it to itself would agree with any number at all.
            ⚠️ **NOBODY FORFEITS.** `room.Forfeit` (`none` · `timed_out` ·
            `left`) and `VerdictForfeited` are gone, and so is the concept:
            leaving and timing out both only announce. `VerdictAbandoned` is what
            a departure leaves behind — not a win, not a draw, not a forfeit.
            ⚠️ **The board carries what the forfeit was pricing**, measured both
            ways: a seat that answers nothing loses on the board
            (`TestASeatThatNeverAnswersLosesOnTheBoardRatherThanByForfeit` — a
            bo3 ending `won` after 56 of that seat's allowances ran out), and if
            both walk away the turn cap draws it
            (`TestWhenNobodyAnswersTheTurnCapDrawsIt`).
            ⚠️ **The stated cost: a losing player can leave for free.** Accepted
            rather than overlooked — on a LAN between friends the enforcement is
            social. → `README.md` § *Nobody forfeits*. Do not "fix" it by
            reinstating a forfeit.
            ⚠️ **A departure DOES need a message and now has one**:
            `wire.Closed{Reason: wire.ClosureLeft}`, the eighth kind, to the seat
            **still there** and to nothing else. It is the one ending a mirror
            cannot reach — no `Ended` for the battle in progress and no further
            `Start` — where every other ending the client computes. One closure
            today, `ClosureNone` at zero for the reason `CodeNone` is, and a
            second reason is an **entry** rather than a new kind
            (`ClosureCount`, `TestEveryClosureHasANameAndTravels`).
            ⚠️ Two good tests were **deleted** with the mechanism they measured
            (`TestThreeConsecutiveTimeoutsForfeitAndAFourthIsNotNeeded`,
            `TestARealActionResetsTheTimeoutCount`) and a note at the head of
            `timeout_test.go` says so.
            ⚠️ `Left` before the first battle **frees the seat** instead of
            ending anything: there is no match yet. A reconnect window therefore
            sits in front of `Left` rather than inside it. → the seat token item
            under *The wire*.
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
      - [x] A capped battle **was invisible to a mirror**. **Done** — `TurnCap`
            rides on `wire.Welcome`, beside `Allowance` and by that field's own
            argument: a cap is *room configuration*, not part of the battle. The
            allowance is there so a client can count down; the cap is there so a
            client can **stop on the same turn**.
            ⚠️ **No new message and no `Ended`**, which is the whole point. The
            client is a mirror, so given the cap it reaches the cap by the same
            arithmetic: every opened turn emits exactly one `battle.TurnBegan`,
            and counting those **including the opening** — the event cursor
            starts *after* the opening board, so a client counting only what
            arrives on a `Turn` sits a turn behind for a whole battle — gives the
            room's own count. `TestTheTurnCapEndsABattleAsADrawTheOutcomeAlreadyHas`
            asserts the two counts are **equal** and that each client stopped;
            the fixture mirror fails if the room ever asks past its own cap.
            After the cap fires both sides hold the same honest state: the room
            has `Undecided` + `BattleResult.Capped`, the mirror's own battle is
            stopped with no `Ended` and nothing decided.
            ⚠️ **Three alternatives were refused — do not re-raise them.** A
            **constant both peers read** costs the host the setting and is still
            a number both sides must agree on, except a version skew then desyncs
            silently where a config field is checked at the handshake. A
            **"battle was capped" message** is a protocol bump *and* a second
            declaration of how a battle ends. **Letting the engine emit `Ended`
            at the cap** is wrong for the reason the room may not stamp
            `Stalemate`: a cap is a **policy**, not a way a battle can end, and
            adding it to `battle.Outcome` makes every renderer and `--verify`
            learn a room's policy.
            ⚠️ **Measured, not deduced: a capped log verifies.** It carries no
            `Ended` at all, so the question was real. Replicating the room's
            stopping rule at a cap of 6 on the shipped roster: 44 events, 6
            choices, **0** `Ended`, last event a `turn_began` — and `--verify`'s
            own procedure reproduced all 44 exactly. ⚠️ The trap belongs to the
            log-writer item below: the record must include the capped turn's
            **own** `turn_began`, because the room advanced into that turn before
            deciding not to ask about it and the re-run advances into it too. One
            event earlier is 43 recorded against 44 re-run and `--verify` fails
            on the count.
      - [ ] Write each finished match out as a `battle.Log`, which makes every
            PvP match `--replay --verify`-able for nothing. ⚠️ The room holds no
            second copy of the events for it — a log writer is another cursor
            over `Battle.Since`, which is exactly why the room reads it that way.
            ⚠️ **Where it stops is load-bearing for a capped battle, and it has
            been measured.** A capped log has **no `Ended` event** and it does
            verify — but only when the record includes the capped turn's own
            `turn_began`, which is where the room's own cursor already stands,
            because `settle` advanced into that turn before deciding not to ask
            about it and `Replay` advances into it too. Stopping one event
            earlier reads 43 recorded against 44 re-run and fails on the count.
            The room's cursor is therefore the right place to write from and a
            "tidier" stop is not.

      **The wire**
      - [x] WebSocket transport, the dependency confined to one boundary.
            **Done** — `internal/socket`, and the dependency is
            **`github.com/coder/websocket`**.
            ⚠️ **The library was measured rather than remembered, and it is not
            gorilla.** `gorilla/websocket` is not archived and reads as the
            default choice, and it has **0 commits since 2025-09** with its last
            release **v1.5.3 from June 2024** — 27 months — and it pulls
            `golang.org/x/net`. `coder/websocket`: **11 commits in the last year,
            v1.8.15 released 2026-06-15, zero dependencies** (its `go.mod` has no
            `require` block at all), first-class `context.Context` on Read and
            Write, concurrent writes supported, passes autobahn. So the module
            gained **one line** in `go.mod` and two in `go.sum`, which was checked
            rather than hoped for.
            ⚠️ **It is the continuation of `nhooyr.io/websocket` and the version
            numbers go BACKWARDS across the rename**: the old path's last release
            is v1.8.17 (2024-08) and the new path is at v1.8.15. Do not "upgrade"
            to the old path; `nhooyr.io/websocket` is a dead end.
            ⚠️ **The two import bans name the library that is actually used
            now.** `internal/room/clock_test.go` and `internal/wire/clock_test.go`
            both banned `github.com/gorilla/websocket`, written when the transport
            was unbuilt — a ban on a library the module does not depend on, which
            could never fire. Both name `github.com/coder/websocket`.
            ⚠️ **The collision this item used to carry was DECIDED before it and
            is no longer a question: one listener per process.** The record's "a
            code carries its own address" and "one process runs many rooms" could
            not both hold with one listener while a code carried only an address,
            and the answer is that `wire.RoomCode` carries a **seventh byte naming
            the room** — twelve characters, 256 rooms behind one socket. The other
            way out, a listener per room, was refused: a port is a finite OS
            resource wanting a firewall hole, one leaks per crashed room, and it
            conflates a room (an application idea) with a listener (an OS one), so
            the registry keyed by code would be shadowed by a second one keyed by
            port and socket lifetime would become room lifetime. What it cost is
            written down — **ten characters became twelve, and the ten-character
            claim is retired** — and `messages.golden` did not move, because no
            message carries a `RoomCode`. **It did not move for this PR either.**
            → `README.md` § *A room, and getting into one*, and § *The transport*.
            ⚠️ **The code rides in the URL PATH and in no message**, which is the
            other half of that: a code is what a person pastes to connect, so it
            is addressing rather than protocol content. `socket.RoomPath` is the
            one spelling of it, `/room/{code}`.
            ⚠️ **A pasted code is decoded and RE-ENCODED before it is used as a
            key** (`socket.roomOf`), and that is not ceremony. `RoomCode.Decode`
            upper-cases first because the alphabet is upper-case only and the fold
            is total — so a lower-case code is a perfectly good code — but every
            key in the registry's map came out of `wire.EncodeRoom`, so without
            the re-encoding a player who typed theirs in lower case would be told
            the room is unknown *while the room sat right there*. An **undecodable**
            code is deliberately **not** refused here: it is handed to the registry
            as it stands, where it is the key of no room and answers
            `wire.CodeRoomUnknown` — the registry's own refusal, and the one
            declaration of it.
            ⚠️ **This package owns the clock and is the only place `time`
            appears.** `internal/room` and `internal/wire` both refuse to import
            it, so the ban's counterpart is a **positive** claim here:
            `TestTheTransportOwnsTheClockAndPrintsNothing` fails if no file in
            `internal/socket` reads a clock, because otherwise somebody could move
            the countdown into a fourth package and both existing bans would still
            pass. The whole conversion is one function, `socket.Allowance`, turning
            `Reading.Config.Allowance` (seconds as an int) into a duration.
            ✅ **The fourth package arrived, and the answer to it is
            `TestEveryClockInTheModuleIsOnTheAllowlist`** — a module-root walk in
            this package holding every non-test file that can read a clock against
            a written list with a reason each. ⚠️ It looks for the *calls* as well
            as the import, because `internal/socket/connection.go` reads a clock
            (`context.WithTimeout` over `Timings`) **without importing `time`**,
            and an import-only walk called that file clockless. `socket.Allowance`
            is on the call list for the same reason: a caller of it is doing clock
            arithmetic by definition, which is also why it is exported rather than
            copied into the client.
            ⚠️ **A timer that fires while an answer is in flight is NORMAL.** The
            room refuses a timeout for a seat it is not asking, so a late report is
            already harmless — and the transport must not read that refusal as a
            reason to close anything, or it drops a player for answering *quickly*.
            The refusal is not forwarded either: the transport owns the timeout, so
            it owns the answer to it. Measured — making it fatal reddens exactly
            `TestALateTimeoutIsRefusedWithoutDroppingAnybody` and nothing else in
            the repository, which is the whole net.
            ⚠️ **The close threshold is `socket.DefaultCloseThreshold` = 60s**, and
            what it guards is the item below. → its own note in `socket.go`.
            ⚠️ **`socket.DefaultMessageLimit` was set from a guess and the guess
            was wrong.** The reasoning was that a 5v5 `wire.Start` carries the whole
            resolved roster and would approach the library's own 32 KiB default, so
            a megabyte was the safe answer. Measured: the largest start a legal room
            can send is **2,911 bytes** over ten units. The library's default would
            have done, and a megabyte was 360 times more allocation than a peer
            should be able to ask for. It is **64 KiB** now, and
            `TestTheLargestStartFitsTheMessageLimit` holds both ends — no headroom
            and nothing but headroom are both worth failing on.
            ⚠️ **`ended()` did not know `net.ErrClosed`**, and the departure test
            is what found it: a client that closed its **own** connection reported
            `use of closed network connection` as a failure of the match it had
            just left. `context.DeadlineExceeded` is deliberately *not* on that
            list — the only deadline here is the write timeout, so exceeding one is
            a peer that has stopped reading and is exactly what the error sink is
            for.
            ⚠️ **`internal/socket` is run a second time under `-race` in
            `make check`**, beside `internal/room`, because those two are the whole
            of the concurrency in the repository. Measured: 4.7s plain, 6.1s under
            the detector, so it costs about 1.4s — and the detector is what caught
            the end-to-end test asserting the server had let its tables go the
            instant the *client* returned, which it has not.
            ⚠️ **Out of scope on purpose, and each is its own item below**: any TUI
            screen, the lobby/waiting/countdown drawing, the wordings, the seat
            token and the rejoin, and **the host binary** — which is built now, and
            took `Server.Shutdown` back into this package with it, because
            `http.Server.Shutdown` does not wait for hijacked connections and only
            this package holds the sockets that were hijacked. → the item below.
      - [x] **The host binary.** **Done** — `cmd/hexarena-host`. It opens one
            room, prints the code, serves the match, prints the result and exits;
            it plays nothing, because both players are clients.
            ⚠️ **The Server's missing shutdown is built and it is FOUR steps, not
            two.** `http.Server.Shutdown` does not wait for **hijacked**
            connections and a WebSocket is hijacked, so `socket.Server.Shutdown`
            is: tell every peer (below), `Registry.CloseAll`, `Registry.Wait`
            bounded by the context, then poll until `Tables()` **and**
            `Running()` are both nought. The last two are separate readings on
            purpose — a table outlives its match by however long two sockets take
            to close — so a shutdown checking one would return with the other
            still going.
            ⚠️ **`CloseAll` runs even on a context that is already done.** Only
            the *waiting* is bounded; a shutdown that skipped the closing because
            it was out of time would leave behind exactly what it was asked to
            stop. The refusal names both counts, because "context deadline
            exceeded" alone tells a host nothing it can act on.
            ⚠️ **The peers are told with a new closure, `wire.ClosureStopped`,
            and that is the only change to `internal/wire` in this PR.**
            `ClosureLeft` would have been a lie — a departure is a judgement about
            a *peer*, and this is the thing that owns the connection deciding to
            stop — and sending nothing leaves a player staring at a dead socket.
            `ClosureCount` moved to 3, no exhaustive switch over `Closure` exists
            anywhere to update, and **`messages.golden` did not move**: no fixture
            carries the new value, so `make golden` had nothing to accept.
            ⚠️ **The notify uses `drop` and not `bye`, and it is worth five
            seconds a socket.** `bye` is the close *handshake* and waits for the
            peer's answer, which a peer that is not reading never sends — and a
            peer that is not reading is exactly what a shutdown has to survive.
            The `wire.Closed` has already said why at the application level and is
            flushed before the socket goes. Measured on the four-connection
            shutdown test: **20.0s with `bye`, 0.01s with `drop`**.
            ⚠️ **`socket.Options.Joined` was added**, because a join leaves no
            other trace a caller can reach: `room.Reading` carries no seat
            occupancy, so a host wanting a line per player could otherwise only
            poll for the *match* starting, which is one line for two people.
            ⚠️ **Which address goes in the code is the sharp part, and it is
            probe → walk → refuse.** A code carries four address bytes, so it is
            IPv4 and it has to be an address the *other* machine can dial. The
            route probe (`net.Dial("udp4", "192.0.2.1:9")`, TEST-NET-1, no packet
            sent — a connected UDP socket only picks a route) answers exactly the
            right question and **fails on a machine with no default route**, which
            a LAN behind a bare switch can be. The interface walk is the fallback
            and is genuinely ambiguous: `docker0` is `172.17.0.1`, is up, is
            private, and is unreachable from the other player's laptop.
            ⚠️ **The docker tie is REFUSED rather than broken, and the reason is
            measured.** "Prefer 192.168/16 over 172.16/12" is the tempting rule
            and it is wrong *on this machine*, whose real LAN address is
            **172.16.32.222** — inside the same RFC 1918 block docker's default
            bridge sits in. "Prefer the lowest" is a coin toss with a tidy
            implementation. The interface *name* would work and is not an address,
            so the pure picker cannot see it. So more than one survivor is an
            error naming every candidate and asking for `-advertise`: guessing
            wrong prints twelve characters that simply do not work, with nothing on
            screen to explain why. The picker is pure (`pick([]netip.Addr)`) and
            table-tested; the two gatherers are the impure half.
            ⚠️ **`-advertise` is deliberately MORE permissive than the picker.**
            It allows loopback and link-local, with a note on the banner saying
            what they mean, because the flag exists to overrule the picker and
            `-advertise 127.0.0.1` is how somebody tries the thing out with two
            clients on one machine. What it still refuses is an address nothing can
            dial at all.
            ⚠️ **The default port is 13579 and it is FIXED**, so a room is on the
            same port every run and a firewall rule has something to name. Free in
            IANA and `/etc/services`; below both ephemeral floors (measured:
            darwin `net.inet.ip.portrange.first` = 49152, Linux's default range
            starts at 32768), so the OS never hands it out underneath us. **31337
            was a candidate and is rejected**: it is Back Orifice's port and IDS
            rules flag it. `-port 0` is still supported and documented as "take any
            free port".
            ⚠️ **The cost of a fixed port is that "address already in use" becomes
            ordinary**, so it is caught and rewritten to name the port and the
            flag rather than passing the syscall's words through.
            ⚠️ **Listen first, THEN open — and the test for it must use `-port 0`.**
            A code carries the port and `Registry.Open` takes the address the code
            will name, so the port has to be bound before the room exists. At the
            fixed default the wrong order still produces a code carrying 13579,
            which still works — so an ordering test run at the default passes
            either way and measures nothing. `TestTheCodeCarriesThePortThatWasActuallyBound`
            drives `-port 0` and asserts the decoded port equals the listener's and
            is non-zero; reversing the two calls reddens it, verified.
            ⚠️ **The clipboard question is ANSWERED: no.** There is no clipboard in
            the standard library, so it means shelling out to `pbcopy`, `xclip`/
            `xsel` or `wl-copy` — three external binaries, a per-platform branch and
            a silent failure on a machine with none of them. The code is twelve
            characters from an alphabet that already excludes 0, 1, 8 and 9 because
            people mishear those; it is meant to be read out loud.
            ⚠️ **`wire.Version.Build` gets a real value here**, from `-ldflags
            -X main.build=…`, else `debug.ReadBuildInfo`'s `vcs.revision` (twelve
            characters, `+dirty` when the tree was modified), else `"devel"`. The
            ordering is a pure function so the test needs no linker.
            ⚠️ **A password given as a flag is visible in `ps`**, which is said in
            `-h` and in the doc comment; `HEXARENA_ROOM_PASSWORD` is read when the
            flag is empty and is one fewer place it is written down, not a fix.
            ⚠️ **`wire.Password`'s redaction did NOT cover this binary's own
            settings struct, and that was measured rather than guessed.** `fmt`
            reaches a field's `String` through `reflect.Value.Interface`, and an
            **unexported** field cannot be interfaced — so `%v` of a struct with an
            unexported `wire.Password` prints the password in full, while every
            other test in the repository stays green. `main.settings` restates the
            redaction with its own `String`/`GoString`. → the note on the type.
            ⚠️ **`cmd/hexarena-host` is on the `-race` line of `make check`, and it
            earned the place.** The transport calls `Options.Joined` and
            `Options.Report` on **a connection's own goroutine** while main is
            printing the banner and the result, so this binary prints from three
            goroutines at once — and the detector caught exactly that on the join
            test's first run, before `main.screen` (a lock around the writer)
            existed. Measured: 1.3s plain, 2.0s under the detector.
            ⚠️ **Out of scope on purpose:** it runs **one** room (the registry
            holds 256, but naming which one finished is a tool for a different
            job), and it writes **no battle log** — that waits on the room writing
            one out, which is its own item.
      - [x] Room code: base32 of a four-byte address, a two-byte port and a
            **one-byte room**, twelve characters, with a round-trip test over
            addresses *and* room bytes.
            ⚠️ **A non-canonical code is refused**, and that is about a map key
            rather than pedantry: twelve characters carry sixty bits and seven
            bytes are fifty-six, so four bits are spare and **sixteen** strings
            decode to any one room's bytes (measured; it was four at six bytes).
            `encoding/base32` has no `Strict()`, so it ignores the trailing bits —
            and the registry keys its map on the string, so a joiner pasting a
            variant would be told the room is unknown while the room sat right
            there. `RoomCode.Decode` re-encodes and refuses a mismatch, naming the
            code that does work.
      - [x] Room password: constant-time comparison, never logged. Documented as
            what it is — a gate against strangers on the network, **not**
            security. **Done** — the comparison has been `wire.Password.Equal`
            since the protocol landed and the redaction has been that type's
            `String`/`GoString`; what was missing was anything that *carried* one
            over a wire, so "never logged" had nothing to be true of.
            ⚠️ **The transport's share is a rule with no exceptions: it never
            reports the bytes of a message it could not read.** A hello that
            **decodes** is safe by the type — `fmt` calls a field's own `String`,
            which is what `TestARoomPasswordIsNeverPrinted` pins by reflection —
            but a hello that **does not** decode is bytes with no type left to do
            the redacting, and `encoding/json`'s own errors quote what they choked
            on. So `socket.errUnreadable` is a sentinel carrying a **byte count**
            and never the decoder's error, and the package can reach no logger at
            all: `log`, `log/slog` and `os` are import-banned and `fmt`'s printing
            verbs are refused by selector, so the only output is the caller's own
            `Options.Report`, which takes an `error`.
            ⚠️ **Two tests, because neither half is enough.**
            `TestTheTransportOwnsTheClockAndPrintsNothing` is the structural guard
            and cannot see a password handed to `Report`;
            `TestAWrongPasswordIsRefusedAndNeverPrinted` drives a wrong password
            **and a malformed hello whose bytes hold the password** over real
            connections and greps the sink for the characters, and cannot see a
            print nothing happened to reach on the day it ran.
      - [ ] A seat token and a rejoin. ⚠️ **The ground this item used to give was
            wrong** — it was filed as cheap because the cursor makes catching up
            cheap, which is true and is not the reason it matters. The real one:
            **the transport cannot tell a wifi hiccup from a departure.** A socket
            closing is a socket closing. With no rejoin, every network blip kills
            a match, and the transport's close threshold becomes the only thing
            standing between two seconds of lag and losing a whole bo3.
            ⚠️ **Until rejoin exists that threshold is a real setting**, not a
            few-second ping timeout to be picked by whoever writes the socket. It
            is the only dial there is, and it is currently guarding a whole match.
            ⚠️ **It exists now and it is 60 seconds**:
            `socket.DefaultCloseThreshold`, configurable through
            `socket.Timings.CloseThreshold`, with what it is guarding written on
            the constant. Two bounds picked it, and neither is taste:
            **generous against a hiccup** — a LAN wifi roam or a switch
            reconvergence is seconds and TCP retransmission rides out tens of
            seconds without the socket noticing, so 60s is several times the worst
            plausible blip; and **under the turn allowance** (90s,
            `room.DefaultAllowance`), so a machine that dies mid-turn is noticed as
            a departure *before* its allowance runs out and the match ends as
            abandoned rather than grinding out one timeout per turn until the board
            kills the passing units.
            ⚠️ **What it does NOT govern is most departures.** A peer whose process
            exits sends a FIN, the read fails at once, and that is a real departure
            with nothing to threshold. The number is only ever spent on a peer that
            has gone **silent and unresponsive** — which is exactly the case it has
            to be forgiving about, and why liveness is a **ping**
            (`socket.DefaultKeepalive`, 15s) rather than a read deadline: a player
            thinking about a turn sends nothing for up to the whole allowance, so a
            deadline on a read would drop somebody for concentrating.
            Two more things already known, so the next reader does not hunt for
            them:
            ⚠️ **`Reading` deliberately does not hold `Pending`**
            (`registry.go:193`), because `Room.Pending` hands back a
            `*battle.Prompt` — a pointer into the room's own state — and passing
            one out of the room's goroutine is exactly the sharing the registry
            exists to prevent. What a rejoining client needs *is* the open prompt,
            so a rejoin wants a **copy whose slices are copied too**.
            ⚠️ **`Left` before the first battle frees the seat** rather than
            closing the room, so the reconnect window sits **in front of** `Left`
            and not inside it.
      - [x] One end-to-end test over a loopback listener, two real clients.
            **Done** —
            `TestTwoRealClientsFightAWholeBo3OverALoopbackListener`, a whole bo3
            over `httptest`, 145 turns checked by each client, 30 ms.
            ⚠️ **It asserts more than "it finished"**, which is the failure mode a
            test of this shape falls into. Four claims, each of which could be
            quietly wrong while a match still ran to completion: each client was
            told **its own seat and its own side** (two facts, two messages — and
            the sides are asserted to have *swapped* between battles, or a bo1
            wearing a bo3's name would satisfy it); the **per-turn digests agreed
            on every turn**, with `Mirror.Compared` as the vacuity guard against a
            run that checked a handful; the **verdict** re-derived from what each
            client's own engine settled rather than read off the room's word for
            it; and **nothing was reported**, the error sink being the transport's
            only output.
            ⚠️ **The two squads are different characters**, and that is the
            measurement rather than variety: a mirror makes the halves of a battle
            interchangeable, so nothing could see a transport that handed one
            client the other's side.

      **The client**
      - [x] The mirror driver: `battle.New` off the seed and the two rosters,
            then `Replay` one decision at a time with a nil fallback. Compare the
            client's own event digest against the server's every turn, so a
            divergence is loud on the turn it happens. **Done** —
            `socket.Mirror`, and it is **production code in the transport's own
            package** rather than the test fixture it was filed as.
            ⚠️ **The reason is one observation about the protocol: nothing on the
            wire says whose turn it is.** `wire.Turn` carries a decision and a
            digest, and `Mirror.Asking` — the prompt this client's own battle
            stopped on, naming a unit on the side this client plays — is the only
            derivation there is. That is deliberate (a "your turn" message would be
            a second declaration of state the mirror already computes), and it
            means **no client can be thinner than a mirror**, so an end-to-end test
            could not exist without one. Writing it as test code and promoting it
            later would have been writing it twice.
            ⚠️ **The same argument reaches the END of a match.** There is no
            series-standing message, so `Mirror.Over` re-derives the series rule the
            room also has — its own `Ended` events against `wire.Welcome.Battles`.
            Two peers agreeing because they compute the same thing from the same
            configuration **is** the mirror contract, the same shape
            `wire.Welcome.TurnCap` already takes; it is not a duplication to be
            tidied away.
            ⚠️ **`Mirror.Decide` applies nothing.** A mirror steps its battle from
            the `wire.Turn` that comes back rather than from its own input, which is
            why the room sends every turn to both clients including the one that
            asked — deciding *and* applying locally would be two paths into one
            battle.
            ⚠️ **A divergence is a typed `*Divergence` naming the turn**, and
            `TestADivergenceIsLoudOnTheTurnItHappens` makes it a **real** one
            rather than a doctored hash: one client is handed a decision naming the
            same unit and a **different legal skill** read off its own open prompt,
            so its engine resolves a different turn and genuinely parts company.
            Flipping a byte of the digest would measure the comparison and say
            nothing about whether two battles that had diverged would be noticed.
            The divergence is forced on the **third** turn, because the claim is
            *on the turn it happens* and clean turns in front of it are what make
            the failure distinguishable from "this never worked".
            ⚠️ `Divergence.Turn` is `battle.Decision.Turn`, which is the **unit's
            own** count of its turns and not a position in the battle — a reader
            who takes it for the latter will see `A1 turn 5` before `E1 turn 4` and
            think the report is wrong.
      - [ ] A player squad file under `os.UserConfigDir()`, separate from
            `internal/seed/data/squads.json` — that one is the game's own data,
            edited by the authoring tool, and a player has no business in it.
      - [x] Lobby, room and waiting screens — **registered in `everyScreen` in
            the same commit that adds them**, for the reason at the top of this
            list. Done: `cmd/hexarena-tui/lobby.go` holds `joinScreen`,
            `waitingScreen` and `resultScreen`, and all nine of their states went
            into `everyScreen` and this client's golden in the same commit.
            ⚠️ **They live in `cmd/hexarena-tui` rather than in
            `internal/screen`**, and the reason is the import graph: a lobby
            holding a `wire.RoomCode` or a `wire.Code` would pull the protocol
            into the package two clients share, and `i18n.Lang.Refusal(name)`
            takes a **name** precisely so that never has to happen. Stated cost:
            `internal/screen`'s golden cannot see them, so a layout regression
            there is caught by one golden rather than two.
      - [x] Undo **off** in PvP, along with another seed, the save key and the
            "let it pick" key — six keys guarded on `draw.PlayScreen.Live`, one
            guard each, at the key.
      - [x] The battle screen driven by `socket.Mirror` rather than by itself:
            `PlayScreen.Attach` and a cursor of its own, because `Drain` writes
            `b.drained` and a live battle is read under a lock that admits
            several readers.
      - [x] The countdown: both clocks drawn on a live battle, so a player can
            see the other one thinking. **Done** — `cmd/hexarena-tui/clock.go`
            counts, `draw.PlayLive.Clock` carries the answer, and the battle
            screen's heading row draws it.
            ⚠️ **This item asked for "a remaining duration ON THE WIRE" and that
            was wrong — nothing was added to the protocol.** It was written
            before the mirror had the shape it has now, and the mirror makes the
            message unnecessary: both peers apply the same `wire.Turn` and open
            the same prompt, so **both clients already know, locally, the moment
            a turn opened and whose it is**, and `Welcome.Allowance` is already
            known to both. So each client counts down for whichever seat is on
            turn — no new kind, no `KindCount` change, no golden moved in
            `internal/wire`. The reasoning the item gave for a *duration rather
            than a deadline* — two machines on a LAN have no reason to agree what
            time it is — is exactly right and is **why** a count from a locally
            observed event is the correct shape rather than a compromise.
            The cost is that the two displays drift by the network hop and by
            when each client processed the event: tens of milliseconds against a
            ninety-second allowance, affordable because **the display is advisory
            and the room's timer is authoritative** — a client whose countdown is
            wrong still learns the real outcome, because a timeout arrives as a
            pass event like any other.
            ⚠️ **`internal/screen` stays clockless**: the remaining seconds are
            **handed in already counted** on `draw.PlayLive`, which is the
            arrangement `internal/room` is already under one layer down — the
            room carries `Allowance` as a number and hands it to its clients
            rather than ever reading one. The screen draws two numbers and does
            not know what a second is.
            ⚠️ **The clock is on the heading row and not on a row of its own**,
            which is the budget item below rather than a layout preference: the
            log pays for every row anything else takes, so a row of its own moves
            three lines per render (the clock, a line of history, and the log's
            own position) instead of one. The heading is where a free reading
            goes on this screen, which is the argument `logPosition` is already
            there under.
      - [x] **The residual the lobby left open, closed with it** — they share a
            clock, which is why they landed together. A peer that dies *while
            this client is being asked* did not unblock the chooser: `Play` is
            inside `Decide` at that moment rather than inside `conn.read`, so
            neither the read failing nor the keepalive giving up (which cancels
            `Play`'s own internal context, not the session's) could reach it, and
            the goroutine sat until the player pressed `esc`. **Done**: a third
            select arm, a timer of `Welcome.Allowance` plus `chooserGrace`, after
            which the chooser passes. The grace is what makes this client the
            *second* to give up — this client starts counting a network hop after
            the room does, so the grace only has to cover clock-rate drift and a
            coarse timer.
            ⚠️ **It closes a second hole the original note did not see**: a
            player who simply never answers stranded their own client just as
            thoroughly, because the room's pass for that seat arrived at a socket
            nobody was reading.
      - [ ] Re-take `playFit`'s budget. ⚠️ A 5v5 body already measures 28 rows
            against the 24 the floor gives it, and PvP adds a waiting row on top.
            ⚠️ **The clock row it also predicted was never spent**: the countdown
            went on the heading row instead, precisely because there was no row to
            give it — measured, a row of its own moves three lines of every live
            render rather than one. A row for it is affordable the day this item
            is done, and not before.
      - [x] The wordings, in both books, Vietnamese composed: room, lobby,
            waiting. **Done**: the menu's ninth entry, the join screen (heading,
            two field labels and their placeholders, the squad chooser, the hint,
            the no-squad line, the wrong-length refusal, the `--data` notice, the
            dialling line, the refusal lead-in and the footer), the waiting
            screen, the result screen, the two seats, and the battle screen's
            live waiting line and three live footers.
            ⚠️ **Still open: the timed-out pass reason**, which is the last item
            in this group — `tui.Line` prints `event.Note` raw, so a turn lost to
            the clock still reads `loses the turn (timeout)` in both languages.
            ✅ **The two protocol enums are done**: all ten `wire.Code`s and all
            three `wire.Closure`s are worded in both books, via
            `i18n.Lang.Refusal(name)` and `i18n.Lang.Closure(name)` in
            `internal/i18n/protocol.go` — one sentence per value and no second
            family, because the status categories have two only where two
            sentences genuinely needed two shapes. They take the enum's **name**
            rather than the typed value, the shape `Lang.StatusCategory` already
            has, which keeps `internal/wire` out of `internal/i18n`'s production
            imports entirely; the four walks over `wire.CodeCount` and
            `wire.ClosureCount` live in `internal/i18n/protocol_test.go`, where
            the import is test-only.
            ✅ **They are worded AND read now, which they were not.** The gap
            this paragraph recorded is closed by the lobby: the six refusals a
            **gate** answers are drawn on the join screen, the three only
            reachable **during** a match are drawn on the battle screen in the
            slot a save note takes locally, and both closures are drawn on the
            result screen. `TestEveryRefusalIsShownAndEveryClosureIsShown` in
            `cmd/hexarena-tui` walks `wire.CodeCount` and `wire.ClosureCount` and
            asserts the sentence is on the drawn body in both languages.
            ⚠️ **Which of the two screens a refusal belongs to is DERIVED
            TWICE**, out of a real `room.Room` — six from `Join` and three from
            `Deliver` — and held disjoint and total, because both screens will
            word any code handed to them and a declared table could be permuted
            freely.
            ⚠️ **`TestNoKeyIsOrphaned` could not see that gap and must not be
            quoted as if it could.** It counts an identifier named anywhere in
            the module, and each of the thirteen keys is named in `protocol.go`'s
            own lookup, so it passed with nothing on any screen showing the words.
            The same blindness holds for `socket.Refusal.Error()`, which still
            prints the raw id — deliberately: it is the developer-facing error,
            and the screen is what words it in the player's language.
            ⚠️ And gloss the new pass reason — `tui.Line` prints `event.Note`
            **raw**, so today a timeout would read `loses the turn (timeout)` in
            both languages.

      **Later, deliberately**
      - [ ] **Spectators**, which the cursor above makes nearly free — the
            *reading* half is already built and the *seating* half is not.
            **Done already, and it is the expensive half**: `battle.Since` over
            an append-only record, so the two players, a spectator who joined
            halfway and the log writer each read the whole battle at their own
            pace. `Drain` used to empty the buffer, which is why only one
            consumer could exist; replacing it was the prerequisite and it
            landed in #236.
            **Not built**: anything that seats one. Three twos are in the way,
            and each is written down where it is —
            `internal/room/series.go` `seatCount = 2`,
            `internal/socket/table.go` `seatsPerTable = 2`, and
            `internal/wire` offering `SeatHost` and `SeatGuest` and nothing
            else. `internal/room/registry.go` already says which layer owns the
            change: *"a third seat is a room change, not a registry one"*.
            ⚠️ **A spectator must not be a third seat, and the reason is not
            tidiness — a third seat would change who wins.** `seatCount`'s own
            comment says why the seats are an array rather than a map: **the
            order the two seats are visited in reaches the roster, and the
            roster's order decides which side wins a speed tie.** So a watcher
            threaded through the same structure as the players shifts that
            order, and the battle it is watching is not the battle that would
            have been fought. Nothing in the suite would catch it: the roster
            would still be legal, the digests would still agree between two
            peers who both had the spectator, and only a comparison against a
            match played *without* one would show it. A spectator therefore has
            to be a different kind of citizen — it takes `wire.Start` and every
            `wire.Turn` through a cursor of its own, its `Deliver` is refused
            outright, and it does not exist to `seats`, to the roster or to
            `other()`. `internal/room/result.go` already flags the last of
            those: `other()` stops being "the other one" the day a room holds
            spectators.
            ⚠️ It also needs a **decision about the room code**: the code a
            spectator pastes is the same twelve characters a player pastes, so
            either the room hands out a second kind of code or a joiner says
            which it means. The second is cheaper and is a wire change, not a
            code change.
      - [ ] **Ban and pick, and a spectator watching it.** Before a match, the
            two sides take turns banning a character and picking one, out of a
            **shared pool**, so a 3v3 fields six different characters and a 5v5
            ten. A pick carries the character **and its skills**. A spectator
            watches the draft as well as the battle.
            ⚠️ **The cast is too small for a 5v5 draft, measured: this is a
            content prerequisite and no amount of code fixes it.** **Re-measured
            2026-09-05: sixteen characters ship** (this said twelve, then
            fifteen) and **eleven** of them have an authored build
            (`builds.json`) — the four without one inside the draftable pool are
            Happiny, Lapras, Oddish and Riolu. Eleven picks leaves **one** ban on
            the built cast at 5v5, so build coverage rather than cast size is
            still what binds. 3v3 is comfortable: six picks leaves five bans on
            the built cast, nine if the pool is the full fifteen.
            So either 5v5 drafting waits for more builds, or the draft is 3v3-only
            and says so.
            ⚠️ **It contradicts a decision this file records as settled, and the
            contradiction is fine but must be written down rather than
            discovered.** *"One squad may field the same character twice"* is
            decided **yes** above. A shared exclusive pool forbids it by
            construction. Both can hold — a *saved* squad may double up, a
            *drafted* one cannot, because the pool is what a draft is — but the
            rule then has a scope where today it has none, and
            `squadIsFieldable` is where that scope has to be legible. Note that
            the measurement taken in #268 weakens the case for doubling up
            anyway: three copies of one character is the **weakest** squad
            available, about 11% across both arrangements.
            ⚠️ **The draft is a second state machine with the same shape as the
            first, and the temptation is to write it differently.** It is a
            sequence of decisions over messages, with a timeout that is an
            **input** rather than a clock, exactly like `internal/room` — so it
            belongs beside it under the same bans, and a draft that read a clock
            or iterated a map would break the same contract for the same reason.
            The good news is that the mirror trick transfers whole: a draft is a
            pure function of the decisions taken, so a client can replay it and
            the server needs to send only the decisions. And what a finished
            draft produces is **two rosters**, which is what `wire.Start`
            already carries — so nothing downstream of the draft changes.
            ⚠️ **A spectator watching a draft needs a record the draft does not
            have.** The battle's append-only record is what lets a late joiner
            catch up; a draft has no `battle.Battle`, so it owes its own — the
            same append-only-plus-cursor shape, or a spectator can only ever
            join a draft at the start.
            **Settled by the author, 2026-09-04:**
            **(a) A pick takes the character, then its loadout — either a build
            already in `builds.json` or one made on the spot.** Both paths, and
            `cast.ChooseLoadout` stays the single loadout rule for both, so the
            draft adds a *chooser* rather than a second legality rule. The
            on-the-spot path is the squad builder's own screen reached from
            inside the draft, which is why the pick is two decisions and not one:
            the character leaves the pool the moment it is taken, and the loadout
            follows.
            **(b) Bans are per format: two a side at 3v3, three a side at
            5v5.** Mirrored, and optional — a side may leave every slot unspent.
            ⚠️ Read as **a side**, which is what "three bans, mirrored" meant
            when it was first settled; the totals below say what the other
            reading would cost, because the difference decides how much cast the
            draft needs and it is not worth discovering later.
            **(c) A pick that runs out of time cancels the whole room** — no
            auto-pick, no default. The match starts over from a new code. This is
            the one place the design does *not* follow "a timeout announces and
            passes"; it is a draft, and a side that never picked has no squad to
            fight with.
            **(d) A ban lasts the match, and the first cut is bo1 only.** Ban and
            pick for a bo3 is its own item below, because "per match" in a series
            is a different game: three drafts, or one draft and two rematches on
            it, is a design decision and not a parameter.
            **(e) The pool is every character that is not held back** —
            `cast.Character.Hidden`, which already exists and which
            `internal/screen/squads.go` already honours *for exactly this
            reason*: it is choosing who fights. ⚠️ Note the flag's own comment
            calls it "an authoring convenience rather than a design statement",
            and a draft gate makes it a design statement; and note
            `internal/screen/picker.go`'s warning that one other list offers held
            back characters **on purpose**, so "filter Hidden everywhere" is
            wrong.
            **Bans are optional** — a side may leave all three slots unspent.

            ⚠️ **3v3 FITS AND 5v5 DOES NOT, MEASURED.** The pool is the cast
            minus the hidden: **fourteen** today, because `naruto.naruto` is the
            only character carrying `hidden: true`. Against fourteen, with the
            counts settled above:

                3v3, 2 a side:  6 picks +  4 bans = 10 of 14 — fits, four to spare
                5v5, 3 a side: 10 picks +  6 bans = 16 of 14 — needs 16 in the pool

            So the 3v3 draft is buildable **today** with four characters to spare,
            and the 5v5 draft needs **two more** than the cast now holds — or one
            more plus unhiding naruto. That is a content prerequisite and no
            amount of code changes it, which is one of the two reasons five a side
            is held back at `hexarena-host`'s flag.
            ⚠️ **This arithmetic said "eleven, not twelve" and "needs five more"
            until 2026-09-04**, and it moves every time a character ships, so it
            is derived rather than remembered:
            `jq '[.characters[]|select(.hidden|not)]|length'
            internal/seed/data/cast.json`. The gap that is left is now **two
            characters**, and § *Nineteen traced Pokemon* holds seven lines of
            art already waiting for them.
            For the other reading of the same sentence, bans as a **total** across
            both sides rather than each: 3v3 becomes 8 of 11 and the last pick
            sees four, and 5v5 becomes 13 of 11 and still needs two more
            characters than exist. Either reading leaves 3v3 comfortable and 5v5
            short, so the reading changes the target and not the conclusion.
            ⚠️ **Bans being optional is what makes a shortfall a runtime failure
            rather than a refused configuration.** A room that is legal when it
            opens can still run out of characters partway through a draft, so
            whatever the counts are, the draft owes a rule for the moment the
            pool would no longer seat both sides: refuse the ban, and grey the
            slot. That rule is cheap and it is what stops the arithmetic above
            from having to be re-checked every time cast is added or a character
            is hidden.
            ⚠️ **The last pick is not a decision whenever slack is nought**, and
            slack is `pool - picks - bans`. Worth drawing on the screen: a draft
            whose final pick has one candidate should say so rather than present
            a list of one.
      - [ ] **Ban and pick for a bo3.** Deliberately after the bo1 draft, because
            "a ban lasts the match" is ambiguous in a series and the ambiguity is
            a design decision rather than a parameter: three drafts, one draft
            carried across all three battles, or a draft per battle with the
            previous winner banning first. Each is a different game. ⚠️ And the
            arithmetic above gets worse per repetition if the pool does not
            reset.
      - [ ] mDNS room browsing, so a client can list rooms with no code at all.
      - [ ] A chess clock — a budget per player rather than per turn.
      - [ ] Prove the mirror across architectures: the same seed and the same
            digest on amd64 and arm64. Friends are not all on one machine, and
            this is the assumption the whole design rests on.
      - [ ] Read the balance again at 3v3. The screened formation was tuned at
            five a side, and a shorter board leaves a summon more free slots.
            ⚠️ **Five a side is held back until this is done** — `hexarena-host`
            refuses `-format 5`, which is the only place in the repository a
            format is chosen. `wire.Format5v5` stays valid **on the wire** on
            purpose: taking it out of `Format.Valid` would be a protocol change,
            and two peers have to keep agreeing about what the field can hold
            whichever is a version ahead. `TestFiveASideIsHeldBackHereAndNowhereElse`
            asserts both halves, because a test for the refusal alone would go
            green the day somebody deleted the constant.
            The second reason is the draft: at three bans a side a 5v5 needs
            **sixteen** in the pool and there are eleven, so a 5v5 ban-and-pick
            cannot be seated either. Lifting the hold-back wants both — the
            numbers read on this board, and five more characters to draft on it.

- [ ] **Graphical client with ebiten.** A renderer over `[]Event`, nothing more.
      It must not read `*Battle`, and it must not need the engine to know how long
      an animation takes. Asset pipeline is undecided: SVG has to be baked to PNG
      at build time or rasterised at load, because ebiten draws neither.
- [ ] **Grow the cast.** **Sixteen** ship across two origins (fifteen Pokemon,
      one Naruto) over **forty-two** authored stages, and **every one of the
      eleven elements is carried**: water ×3 (Lapras, Poliwag, Squirtle), grass ×2
      (Bulbasaur, Oddish), metal ×2 (Magnemite, Riolu), dark ×2 (Gastly, Mewtwo),
      neutral ×2 (Happiny, Mew), wind ×2 (Dratini, Naruto), and one each of fire,
      ground, ice, electric and light. Two duals: Magnemite (electric/metal) and
      Lapras (water/ice). ⚠️ **Wind was carried only by the hidden Naruto until
      Dratini**, so both of the book's wind skills were `restrict.origins:
      [naruto]` and no draftable unit could field one: an element being "carried"
      is not the same as its pool being reachable — see
      `memory/hexarena-pricing-a-new-element-pool.md`.
      This is content, and the constraints that bound it are written down.
      ⚠️ **This entry existed TWICE until 2026-09-05** — once here and once in
      `CLAUDE.md` § *Open work*, both current, both maintained, worded
      differently. That is the *"two callers wording one choice"* mistake
      `CLAUDE.md` § *Mistakes already made here* records, at the level of a
      document; the two are merged below and neither had anything the other did
      not need.
      ⚠️ **The count in this line has been wrong five times.** It said "three, one
      per element" until 2026-08-28, "four, one per element" until 2026-08-31,
      "five" until Magnemite, and "eight" until 2026-09-04; #98 landed Naruto and
      #182 landed Poliwag, which is where one-per-element stopped being true, and
      **every element has been claimed since**, which is where it stopped being a
      guide at all. Derive the number rather than remember it —
      `jq '.characters|length' internal/seed/data/cast.json` — and do not edit it
      by hand.
      ⚠️ **No element is left to claim, so "one per element" is finished as a
      guide.** What a new character is bought for now is a **way of playing**, and
      § *Nineteen traced Pokemon* below is the art waiting for one.
      ⚠️ **On the archetype, which is the closest thing to a way of playing this
      data has a field for.** Sixteen characters against sixteen archetypes, one
      each — but that is where the counting has landed, **not a rule, and nothing
      enforces it**: `cast.ParseArchetypes` refuses an archetype *declared* twice
      and says nothing about two characters *tuned from* one, and no test in
      `internal/seed` asks. So a second character on a shipped archetype is
      **fine** and needs no permission. The aim is only that each character ends
      up unique in how it plays, and the archetype is one lever of several — the
      kit, the element, the stat table and the traits are the rest, and two
      characters off one preset with different kits are two ways of playing. The
      thing to avoid is a *duplicate*, not a shared preset. → `docs/balance.md`
      § *Pricing one number*: what says two characters play differently is a
      measurement, not the field they were tuned from.
      ⚠️ **Magnemite is the first character to declare two elements**
      (`"element": ["electric", "metal"]`), and the field has accepted an array
      since the chart was written — `element.Dual`, `ValidateAffinity` and
      `MultiplierAgainst` were all shipped with no user. What the pairing turned
      out to be worth is written down in `internal/seed/dual_test.go`: a pair
      whose halves are unrelated mostly *cancels* — electric/metal is countered by
      four elements as two singles and by **one** as a pair — so the defensive half
      of a dual is close to nothing, and what is bought is the second skill book.
      **Three constraints bound the content**: an archetype's kit constrains a
      character's affinity (`skill.CanCarry` enforces it while authoring,
      `Archetype.Demands` reports it), `progression.Limits` bounds health and
      defence **together** because those two multiply, and — softer than the other
      two — a skill kept for a lineage asks the character to *be* one, so adding a
      dragon is two lines: the kind in `species.json` and the claim on the
      character. **A new skill also has to say which story it is out of** —
      `restrict.origins`, or a line in `sharedPool` arguing it belongs to nobody;
      `TestEverySkillSaysWhichWorkItIsFrom` refuses the omission.
      ⚠️ **A character moves `cast.golden`, `species.golden` and `origins.golden`
      — not `scenarios.golden` and not `replay.golden`.** This line claimed the
      opposite until 2026-08-31, and the entries saying a cast-only addition moves
      no golden were right all along: `replay.golden` renders the **roster**, so a
      character reaches it by being placed in `roster.json` and by nothing else.
      Add a preset and `archetypes.golden` moves too; add skills and
      `skills.golden` and `describe.golden` move. #182 added a character, a preset
      and four skills, and moved exactly those six.
      **Read Squirtle first** — water is the strongest of the elements and
      Blastoise still cannot carry an ace slot, because its attack and speed
      curves are the lowest in the cast.
      ⚠️ **Lapras landed after this line was written** and is the **second** dual
      affinity (water/ice), the first `glacier`, the first `leviathan`, and the
      first character to bring the `ice` book — five skills authored with it
      (`ice_shard`, `ice_beam`, `blizzard`, `ice_wall`, `hail`). Budget: effHP
      **10,132** bare and **10,926** under `endurance`, against the 11,500 bound;
      `hexforge check` reports no problems.
      ⚠️ **A spar reads a column carrier low, and the learnset ORDER is a lever
      worth 6.5 points of it.** `forge.seedKit` fields the first four learnset
      entries, so a kit led by `blizzard` (a column) and `withdraw` (self) spends
      half of itself on a 1v1 board: **15.4%** overall at 300 seeds. Ordered
      `ice_shard, ice_beam, water_gun, withdraw` — the duel kit first, the area
      skills behind it — the same character reads **21.9%**, which is the order
      shipped. Bracketed by `pokemon.cleffa` at **17.9%** and `pokemon.squirtle`
      at **40.4%** on the same measurement, so it is not an outlier; and per
      § *Pricing one number* a spar rate prices nothing, it is only the gate that
      says a character is not degenerate.
      ⚠️ **The area slow was priced on a SQUAD, because a duel cannot see one.**
      `blizzard` (column, enemy-only) carries `mire` — speed −25%, 2 stacks, 2
      turns. A 1v1 spar reads the chance at nothing (15.4 / 15.2 / 16.6% at 30 /
      50 / 70%), which is the board being wrong rather than the skill being
      worthless: one target is the case a column skill is not for. Measured
      instead with `Library.FightSquads` over a 3v3, Lapras + Squirtle + Machop
      against **the same squad with `water_gun` in blizzard's slot**, 500 seeds
      each way round — the swap is the measurement, and the mirror control read
      **50.0% exactly**:

      | mire chance | home rate |
      |---:|---:|
      | none | 47.9% |
      | 30% (was shipped) | 51.7% |
      | **50% (shipped now)** | **54.4%** |
      | 70% | 56.2% |
      | 100% | 60.5% |

      Monotone over five points, 1,000 battles a row (2σ ≈ ±3.2pp). **Without the
      slow, blizzard is worth less than `water_gun`** — 47.9% against an even
      50.0 — so the slow is the whole of why it is brought, and 30% left it
      barely above the control. 50% sits under `whirlpool`'s 60% while blizzard
      hits harder (950 against 800), which is the ladder the other five mire
      appliers make.
      ⚠️ **`hail` was the other candidate and it fails, measurably.** It is
      `target: all`, so its damage lands on its own side too: weighing its power
      saturates in BOTH directions — 800 wins 0.0% of what it decides, 450 wins
      ~100% — which makes *cutting* its damage a buff rather than compensation.
      And a global slow at 50% or more saturates the mirror outright (100%/0%),
      the first mover locking the other out. Left alone.
      ⚠️ **Neither `blizzard` nor `hail` is in the fielded four**, so both are
      squad picks: `seedKit` takes the first four learnset entries and the duel
      kit is what is shipped there. That is why the squad measurement above had
      to name the kit rather than let the seed pick it.
      ⚠️ **Open, and it is an authoring judgement rather than a bug**: the ice
      half costs Lapras the water matchup outright. `pokemon.squirtle` takes
      **62.0%** off `pokemon.charmander` and Lapras takes **0.0%** of 400, because
      fire answers ice on the cross chain while water answers fire on the organic
      one, and the pair reads the worse half. Whether a dual should lose a
      favourable matchup whole is a decision for whoever authors next — it is
      written down rather than tuned away.
- [ ] **Nineteen traced Pokemon are waiting for a character — seven complete
      lines.** The art lands first because `cast.ParseBook` refuses a character
      that declares no image, so the order is forced: trace, then author.
      ⚠️ **This entry said "thirty-one" and was stale in BOTH directions**, which
      is why the number is now measured rather than carried: seven of the lines it
      listed have **shipped** since (Cleffa, Happiny, Gastly, Magnemite, Riolu,
      Mew, Mewtwo), and three lines it never mentioned have been **traced** since
      (Abra `7df76e6`, and Gible + Magikarp in `190768e`). A hand-kept list of
      what is waiting goes wrong every time either end moves. The measurement:

          comm -23 \
            <(ls internal/seed/data/assets/*.svg | xargs -n1 basename | sed 's/\.svg$//' | sort) \
            <(grep -o '"assets/[^"]*\.svg"' internal/seed/data/cast.json \
                | sed 's|"assets/||; s|\.svg"||' | sort -u)

      Nineteen files, seven lines, **no orphan form** — every line below is
      complete, and nothing `cast.json` names is missing from `assets/`.
      By line, as they would be authored:
      **pichu → pikachu → raichu** · **igglybuff → jigglypuff → wigglytuff** ·
      **mareep → flaaffy → ampharos** · **abra → kadabra → alakazam** ·
      **gible → gabite → garchomp** ·
      **magikarp → gyarados** · **onix → steelix**.
      ⚠️ **Nothing references any of these**, which is why they moved no golden
      and why `TestTheShippedArtIsCutOutRatherThanFramed` does **not** cover them —
      that test walks the art shipped characters name, so the day one of these is
      authored is the day its picture is first measured. **That day came for the
      dratini line on 2026-09-05** and all three passed first time, which is one
      piece of evidence for the hand-check below rather than a guarantee for the
      rest. The sources were checked
      by hand instead: real `tRNS` transparency, 46–71% of the canvas clear, the
      inked box well inside the frame — no baked chequer, so no `--decheck` was
      needed.
      ⚠️ Each of these needs more than a picture: an origin (`pokemon` exists), a
      species claim if any skill it wants is a lineage skill, an archetype whose
      kit its affinity can carry, and a stat table inside `progression.Limits`'
      joint health-and-defence bound. **A new skill also has to say which story it
      is out of.** ⚠️ And **no element is left to claim** — all eleven are carried
      — so what a new one is bought for is a way of playing; a shipped archetype
      may be reused, and § *Grow the cast* above says why that is not a problem.
      Authoring one is the *Grow the cast* item above, not a separate task — this
      entry is the queue, not the work.
- [ ] **Squad composition bonuses: a threshold at 2/3/4/5 for a shared element
      or a shared origin, plus whatever other axis is worth one.** The idea, as
      asked for: fielding several units that share something grants the squad a
      bonus, stronger bonuses sit at higher thresholds, and **not every bonus
      needs four rungs** — one or two is fine where the effect is worth it.
      Nothing is built; this entry is the idea plus what the shipped data already
      says about it, so the design starts from measurements rather than from
      taste.

      ⚠️ **Reachability is not a detail — it is measured, and it kills two of the
      four obvious axes outright.** Multiplicity across the sixteen shipped
      characters:

      | axis | most units that can share one value | rungs reachable |
      |---|---:|---|
      | element | 3 (water: Lapras, Poliwag, Squirtle) | 2, 3 |
      | origin | **15** (`pokemon`) | 2, 3, 4, 5 — but see below |
      | species | 2 (`plant`, `mythic`, `dragon`; every other species is 1) | 2, and only on three species |
      | archetype | 1 (sixteen characters, sixteen presets) | **none** |
      | archetype `column` | 6 / 5 / 5 for columns 0 / 1 / 2 | 2, 3, 4, 5 on all three |

      ⚠️ **An origin threshold is FREE at every rung today**, and that is the
      sharpest thing here: fifteen of sixteen characters are `pokemon`, so *any*
      squad that is not built around Naruto satisfies a 5-of-one-origin bonus by
      accident. A bonus nobody has to build for is not a bonus, it is a stat
      change with extra words. The same trap sits one step further out if traits
      were the axis: `endurance` is on **nine** characters' presets. Origin only
      becomes a real axis when a third origin ships with enough characters to
      field, and until then the honest options are (a) don't do origin, (b) do it
      at rungs the *smaller* origins can reach and accept that `pokemon` gets it
      free, or (c) key it on something scarcer than the origin itself.
      ⚠️ **Decision 4 below makes this worse rather than better** — with bonuses
      stacking, no 3v3 squad can fail the origin axis *at all*, so the settled
      answer is: element first, origin held until a second origin can field two
      or three.

      ⚠️ **A 4- or 5-rung bonus does not exist in the format that is playable
      today.** 3v3 caps every count at three, and 5v5 is still behind
      `hexarena-host`'s flag — and per § *PvP over a LAN* the 5v5 **draft** needs
      two more characters before its pool even fits. So a four-rung table would
      ship with its top half unreachable, which is the "fixture hides a branch"
      shape this repository has paid for five times. Either the top rungs wait
      for 5v5, or a bonus's rungs are chosen per format, or the tables are
      declared with the top rungs and a test asserts they are **currently
      unreachable on purpose** rather than silently dead.

      ⚠️ **The duplicate loophole, and it is already decided in the other
      direction.** `Squad.Validate` refuses a repeated unit **id** and a repeated
      **slot** and says **nothing** about the same character twice — that is
      settled above as *"it MAY"*, with `TestOneSquadMayFieldTheSameCharacterTwice`
      holding it. So a 2-of-an-element rung is satisfiable by fielding one
      character twice, and a 3-rung by three copies. #268 measured three copies of
      one character as the **weakest** squad available, about 11% across both
      arrangements — which is exactly why this matters: a composition bonus is the
      one thing that would make the degenerate shape worth building, and the
      measurement that currently argues against it was taken **without** one. The
      draft is the opposite case: a shared exclusive pool forbids doubling by
      construction, so the same bonus means two different things in the two modes,
      and `squadIsFieldable` is where that scope has to be legible.

      **Axes beyond element and origin, with what the data says about each:**
      - **`column`** (0..2 on the archetype preset) — the best-shaped candidate:
        every rung is reachable on all three columns, and it means *how far
        forward this unit wants to stand*, so a threshold on it is a **formation**
        bonus rather than a tribe bonus. It also reads as a real decision, because
        stacking one column is a squad with a hole in it.
      - **Stage depth** — all five at the tip of their line, or all at an interior
        stage (an "unevolved" bonus). ⚠️ Asymmetric today: Lapras, Mew and Mewtwo
        have exactly one stage, so they are permanently final-form and can never
        satisfy an interior-stage rung.
      - **Formation geometry** — all in one rank, one per rank, and so on. The
        board already makes shape matter (`range` counts **occupied enemy ranks**,
        not distance), so a geometry bonus stacks with a rule that is already
        load-bearing — which is a reason to price it carefully, not a reason to
        skip it.
      - **The KIT's element rather than the character's affinity** — a squad whose
        *skills* are all one element is a different claim from one whose carriers
        are, and it is buildable today where the affinity version is not.
      - **All-distinct ("rainbow")** — the one shape whose reachability *improves*
        as the cast grows instead of needing the cast to grow first, and the
        natural counterweight to every threshold above: it rewards not stacking.
      - **Level spread** — all at the cap, or a squad deliberately carrying an
        under-levelled unit. Cheap to read, and it interacts with `progression`
        rather than with the chart.

      ⚠️ **Where it can live, and the two core rules it walks into.** Counting a
      roster by element or origin is a **map walk**, and `internal/core` forbids a
      map's iteration order reaching an output — so the count is a sorted or
      fixed-order walk, the same rule `Registry.Codes` and `Server.held` already
      obey. And the shape of the grant decides what comes free: a **permanent
      `status` applied at battle start** gets the log line, the drawing and the
      describers for free — and then `dispel` becomes a question nobody has asked
      (may a strip take a composition bonus off? if yes, the bonus is a target; if
      no, `strips` needs a category it may not name, which § *What a dispel may
      not name* already has a precedent for). A grant **baked into `Take`/`New`**
      is invisible, undispellable and cheaper, and no screen can explain it.

      ⚠️ **The per-character budget stops bounding what is fielded.** The gate is
      `EffectiveHP = hp*(300+def)/300 ≤ 11500` **per character at the cap**; a
      squad-wide defence grant breaches it collectively while every member still
      passes. Decide whether the bound is checked before or after the bonus — and
      note that ceilings **saturate rather than clamp**
      (`CLAUDE.md` § *Saturate continuous values, cap discrete ones*), so a bonus
      on a stat already near its ceiling buys much less than the number says —
      the `reckless` sweep measured a −400‰ term on a base of 400 fighting at
      **290** rather than 240, with the lever's whole reachable range 290..391.

      ⚠️ **Nothing can be priced until `forge` can turn the bonus OFF, and that is
      the prerequisite rather than a later step.** Both obvious controls are
      already measured wrong: swapping a member measures **the member**
      (`docs/balance.md` § *Pricing one number: `hexforge weigh`, and why a roster win
      rate could not* — ally damage up read ally win rate *down*), and
      putting the same bonus on both squads **cancels it** (the Oddish/Bulbasaur
      pairing read −29‰ that way, and the event log said why: `enemy.partner:
      105`). The only control that measures the bonus is **the same squad, same
      members, same seeds, bonus on against bonus off** — with the mirror control
      reading 500‰ exactly. ⚠️ **And it is a set of bonuses to disable rather than
      a boolean**, because bonuses stack: a global off switch would measure *the
      system* and could never price one rung. → decision 4. Build the switch
      first; every number quoted before it exists is about something else.

      ⚠️ **PvP, and it is cheaper than it looks.** A squad crosses the wire whole
      in `wire.Hello`, so the room and the client's mirror each derive the bonus
      from the same bytes — no new message, and a disagreement surfaces as a
      per-turn **digest mismatch** rather than as two different boards. It does
      join what the data digest gates, so two peers on different balance data stop
      being able to play, which is already the contract.

      **All seven settled 2026-09-04 (author's call). Each one closes a branch,
      and two of them close it by ruling something OUT rather than in:**
      1. **The count is taken once, on entering the battle. It is NOT recounted,
         and a summon does NOT count.** So a composition bonus is a **drafting**
         decision and never a tactic: focusing the odd unit out cannot take it
         away, and `summonAffinity` handing a summon the caster's affinity is now
         irrelevant to it. ⚠️ This is the cheap answer in the right way — it needs
         no hook in `tickStatuses`, no recount on a death, and nothing in
         `internal/core/battle` has to know a unit left. It also means the grant
         can be resolved **before the first turn**, where the roster is still a
         slice and no map walk is needed at all.
      2. **A dual affinity counts toward BOTH halves.** Lapras is water/ice and
         Magnemite electric/metal, so each of them is glue on two axes. ⚠️ Worth
         knowing why this is not free: § *Grow the cast* above measures the
         **defensive** half of a dual as close to nothing — a pair whose halves
         are unrelated mostly cancels — so counting both halves here is the first
         thing in the game that pays a dual for being one. That is a reason to
         watch the two duals in the first measurements, not a reason to change it.
      3. **There are TWO KINDS of bonus: one the whole squad receives, and one
         only the units that share the thing receive.** Both ship; a bonus
         declares which kind it is. ⚠️ The second kind is the one that needs a
         drawing decision — a screen has to be able to say *whose* bonus it is,
         and a per-unit grant on some units and not others is a thing no existing
         status display shows.
      4. **Bonuses STACK.** Entering a battle with several squad-wide bonuses and
         several sharers-only bonuses live at once is the ordinary case, not an
         edge. The correlation objection this entry raised is **dead as raised**,
         because it was an objection to a *`column`* bonus and there is no column
         bonus: water came bundled with a free column rung where grass did not,
         and with no column axis there is nothing to bundle.
         ⚠️ **But stacking makes the ORIGIN axis worse, not better, and this is
         the one thing to read before authoring one.** With rungs 2 and 3 at 3v3,
         **no 3v3 squad can fail the origin axis at all**: fourteen of the fifteen
         shipped characters are `pokemon`, and the fifteenth is one character, so
         the worst case a squad can reach is Naruto plus two Pokemon — which is
         still **2 of one origin**, still a rung. (`Hidden` does not save it:
         `cast.Character.Hidden` is an authoring convenience by its own doc — a
         hidden character still ships, still loads, still fights, and a squad
         naming one is as valid as any other.) So an origin bonus shipped today
         would fire for **every squad in the game, unconditionally** — that is not
         a bonus, it is a change to the base numbers wearing a threshold. Under
         the *one-at-a-time* rule it would at least have competed for a slot and
         lost; under stacking it is simply added to everything.
         **Therefore: element bonuses first, and hold the origin axis until a
         second origin has two or three characters to field.** The axis is not
         wrong, it is only empty — same finding as § *Grow the cast*'s, one layer
         on.
         ⚠️ **Stacking does NOT break the pricing instrument, provided the switch
         is PER BONUS.** The control is still the same squad, same members, same
         seeds, with **one** bonus toggled — the others may stay on, because what
         is being measured is the difference the toggled one makes on the board it
         is actually played on. A single global on/off would measure *the system*
         and could never price a rung. So `forge.FightSquads` takes a set of
         bonuses to disable, not a boolean.
      5. **Bonuses are built ONE AT A TIME, and each must do something no other
         bonus does.** Not a batch of them behind one mechanism: the kind of grant
         is settled per bonus when that bonus is built, and the rule is
         **distinctness** — two bonuses that come to the same thing with different
         words are the *"two callers wording one choice"* mistake
         `CLAUDE.md` § *Mistakes already made here* already records, at the level
         of a feature instead of a string. ⚠️ So a new bonus's PR has to state
         **what no shipped bonus already does**, the way a new character states
         which archetype and which element it is first at. When the answer is
         "nothing", the bonus does not ship.

      6. **Rungs 2 and 3 only, for now.** 3v3 is the format that exists, so a
         bonus ships with the rungs its format can reach and nothing else — rungs
         **4 and 5 are authored later**, as part of opening 5v5, and are tested
         then. ⚠️ Read the difference: this is *not* option (A) with the top rungs
         left quiet. A rung that cannot fire is not declared at all, so there is
         no row for a test to pass vacuously over, and the day 5v5 opens the new
         rungs arrive **with** their measurements rather than inheriting a claim
         nobody checked.
      7. **Its own file.** A bonus is a rule about squads rather than an axis of
         how one unit fights, so it does not belong beside the presets in
         `archetypes.json`. ⚠️ **A new data file is a sixteenth name in three
         independent places**, and they are exactly:
         `internal/seed/seed.go`'s single-line `//go:embed` directive · the
         `dataFiles` slice in `internal/seed/digest.go` (whose own comment says it
         mirrors the directive, and whose count is the *fifteen* in that comment) ·
         and one `XxxFile()` accessor beside the other fifteen. Missing the second
         is the silent one: the file loads and the **data digest stops covering
         it**, so two peers on different bonus data would pass the digest gate and
         then diverge — which is the failure the digest exists to prevent.
         It also earns a golden of its own, on the rule that each generator is
         handed exactly what it reads.

      - [ ] **A reference screen for the bonuses, on the menu.** A player has to
            be able to look up what a threshold gives before building a squad, the
            way `screenStatuses`, `screenElements`, `screenTraits` and
            `screenSpecies` already work — read-only, drawn by `internal/screen`,
            wording out of `internal/i18n`, golden-held, and offered by **both**
            clients because a reference screen is not authoring.
            ⚠️ **It is the TENTH menu entry.** `menuItems` holds nine today (seven
            catalogues, a battle and the join), and the entry belongs after
            Origins where the other catalogues are.
            ⚠️ **The sweep is the thing that gets forgotten**, and `model.go`
            records **five separate occasions** where a screen slipped the
            authoring tool's sweep and silently lost its width, translation and
            leak tests. `screenCount` plus `TestEveryScreenThisClientDrawsIsSwept`
            is what catches it — a new screen either goes in the sweep or carries a
            written reason for staying out.
            ⚠️ **It is a DATA screen, not prose**, so it spends
            `Context.UsableWidth()` rather than `draw.MinWidth`, against the
            **120x24** floor — and its **footer** is the exception to that split:
            a key-chord row is catalog wording, so it is measured against the
            floor and **the floor is the only lever it has**
            (`internal/screen/screen.go`, the note on `MinWidth`, measured: 35
            pairs packed against the ceiling before #173/#175 and 34 after).
            `TestEveryWordingFitsTheMinimumWidth` is what holds it.
            ⚠️ **It has to draw the two KINDS apart, and draw that several fire
            at once** (→ decisions 3 and 4): a reader building a squad needs to
            see which grants land on everybody and which land only on the units
            that share the thing, and that stacking is normal rather than an
            edge. That is one more column than the existing catalogues carry, and
            it is the part that will fight the width.
            ⚠️ **Nothing to draw yet.** This screen cannot be built before the
            first bonus exists, because a catalogue of nothing is a screen no test
            can hold — the same reason `TestTheShippedArtIsCutOutRatherThanFramed`
            does not reach unused art. Rungs 4 and 5 do not exist either
            (decision 6), so the first version of this screen draws **two** rungs
            and has to not look broken when it later draws four.

- [ ] **The two-number surface is a different pair, and no skill in the book can
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
- [ ] ⚠️ **A `hexforge new` still churns `screens.golden`, and sorting the cast
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

- [ ] **The battle screen says how many cells a shape catches and never which.**
      A skill's blurb reaches the aim list, but the list prints one cell and its
      occupant, and the summary beside it prints a count — `3 ô`. Neither says who
      an area skill is about to reach, so the one thing worth knowing before
      spending a turn is the one thing a player has to hold in their head.
      ⚠️ **The drawing already exists and cannot be reused as it stands.**
      `shapeBoard` in `internal/screen/skills.go` renders a footprint from
      `forge.ShapeCoverage`, and it is anchored to `ShapeDiagramCell` — the fixed
      `{4, 1}` chosen so eight of the nine shipped shapes draw in full. That is
      right for the authoring screen, which is describing a *shape*, and wrong for
      a battle screen, which is asking about *this aim on this board*: coverage has
      to be resolved through `pattern.Targets` from the cell under the cursor, then
      the units standing in the result named. A footprint drawn from the diagram
      cell would be a picture of a different board.
      Worth doing with the item below — they are one complaint, that the screen
      will not show what a turn is about to do before it is spent.

- [ ] **A skill with one legal target fires without asking.** The picker opens
      only on `len(option.Aims) > 1`, so a single-target skill with one enemy left
      commits the turn on the keystroke that chose the skill. The author wants the
      picker every time.
      ⚠️ **This is a decision being reversed, not a bug being fixed**, and the
      reasoning is written down at `cmd/hexarena/main.go` — *a question with one
      answer is not a decision*. It reads well until the answer is the one thing
      the player wanted to look at before committing, which is why it belongs with
      the coverage item: with a footprint on screen, the single-aim stop is where
      it gets read.
      Three sites hold the rule, and a change to one of them is a change to the
      keystroke count of the other two: `internal/screen/play.go` twice — the
      answer path and the live-take path — and the CLI's `chooseAim` once. Whether
      the CLI follows is the author's call; it is a different kind of screen.
      ⚠️ The screens are golden-held, so the extra keystroke moves every play
      fixture that casts a single-aim skill. That is the change being visible
      rather than a fixture problem, but it is the bulk of the diff.

- [ ] **`reckless` is the dragon build's 22.1%.** The roadmap said the gap was the
      missing detonate. It was tested: the line was given one (`dragon_drive`, off
      the `expose` its own `dragon_claw` applies, and it does fire — 19 amplified
      casts in 60 battles), and fielding it reads **21.2%** against 22.0%, which is
      nothing and slightly the wrong way. ⚠️ **Measured one change at a time over
      3000 battles**: the detonate **−0.8**, `reckless → blood_thirst` **+33.1**,
      `reckless → blaze` **+16.9**, fire losing *its* detonate **+10.9**. The trait
      grants `unleashed` **and** `bare` — 30% of attack for 40% of defence *and*
      40% of dodge — into a build whose opponent amplifies its heaviest skill three
      and a half times off a status.
      ⚠️ **`TestRecklessIsATradeAndNotAGift` cannot see this.** It asks whether
      something is given up and passes; whether *too much* is given up is a
      different question, and the same shape of gap a win rate had against
      `swiftness`.
      ⚠️ **Do not just swap the build's trait.** `blood_thirst` beats `reckless` in
      the mirror **and** against the rest of the cast (100%/100% against 98.6%/96.0%),
      so the swap is a power increase rather than a rebalance. Softening the cost is
      the other lever and it is uncoupled — `bare` is granted by `reckless` and by
      nothing else, and no skill applies it — but there is **no test holding its
      magnitude**, so whatever it becomes has to be measured and written down the
      same way. Decide what the trait is *for* first.
      ⚠️ **Dropping `bare`'s dodge clause was tried and it is not the lever.** The
      argument was good — the clause is what makes the trait two-stats-for-one, and
      dodge gates whether an attack connects at all, so removing it should compound
      against a burn-and-detonate opponent. Measured on the same instrument, same
      3000 battles both ways round: **R0 shipped 22.0% · R1 dodge dropped 24.8%
      (+2.8) · R2 vulnerability alone 16.1% (−5.9) · R3 both 18.7% (−3.3)**, against
      referents re-taken on the same run of **A `blood_thirst` 55.1%** and **B
      `blaze` 38.9%**. R3 is below where the trait started and far below B, so the
      candidate was **not shipped**. The interaction is clean — `R3−R0 = −3.3`
      against `(R1−R0)+(R2−R0) = −3.1` — so the two terms **add**, and the
      superlinearity the lever was chosen for does not exist.
      ⚠️ **`bare`'s cost is 88% defence and 12% dodge**, which is why. Decomposed:
      dodge-only reads **43.4%**, no `bare` at all **46.3%**, and those add too
      (2.8 + 21.4 = 24.2 against 24.3). So the *only* lever that can move the figure
      is softening the defence magnitude — the one with no natural stopping point,
      and the one this item has always been reluctant to pull. Dropping `bare`
      outright lands at 46.3%, inside (B, A) and on the 45–50% target, but that is a
      trait with no cost and `TestRecklessIsATradeAndNotAGift` refuses it.
      ⚠️ **Half the missing guard now exists.**
      `TestRecklessSpendsNoMoreThanItBuys` prices the trait in **damage off the
      event log** rather than in wins — a win rate cannot price a stat, which is the
      `swiftness` finding — and asserts the trait buys damage and spends no more
      than twice what it buys (a declared design constant borrowed from the detonate
      rule, the only invented number in it). Shipped data reads 30956 bought for
      50084 spent, ratio 1.62. It is what caught R3: `bought` went **negative**
      (−7898) where the 18.7% win rate sat comfortably inside every existing band.
      The other half — a cast-wide *no trait may lower more distinct stats than it
      raises* (`ballast` keeps it non-vacuous; necessary and **not** sufficient,
      since it would pass a `bare` at −900) — is now **dropped rather than held**.
      No fix landed for it to land with, and the sweep below is the reason it is
      the wrong test: it counts stats and never magnitudes, so it would pass a
      `bare` at −900 and refuse a `bare` at −25, and what turned out to be worth
      holding is exactly the magnitude. The ledger already holds that, in the
      currency the trait is denominated in.
      ⚠️ **The third lever is dead too, and all three are now measured.** Softening
      `bare`'s defence magnitude was swept on the same instrument — dodge left at
      −400, `unleashed` untouched, 3000 battles a row, referents re-taken on the
      same run and identical (**A `blood_thirst` 55.1%**, **B `blaze` 38.9%**,
      **R0 22.0%**). The four amounts the item prescribed **all sit under the
      floor**: −300 **24.4%** · −250 **24.7%** · −200 **27.8%** · −150 **37.0%**.
      Swept finer: −125/−100/−95 **37.4%**, −90/−85/−80/−75 **39.9%**, −50
      **41.2%**, −25 **42.9%**, and a term of 0 is refused by the parser.
      ⚠️ **The dial is a quarter as long as it reads, because the stat saturates.**
      `modifier.Set.Stat` saturates a change towards a floor rather than applying
      it, so a −400‰ term on a base of 400 fights at **290**, not 240, and the whole
      reachable range of the lever is **290..391**. The rate also moves in *steps*:
      fourteen amounts give nine rates, with plateaus (−90..−75 are all exactly
      1197/3000) and cliffs, because a two-point defence change is worth nothing
      until it moves a strike across a kill threshold. **A dial like this cannot be
      tuned to a target** — the nearest rung to the 45–50% wanted is 42.9%, at a
      cost of nine points of a stat.
      ⚠️ **Both gates flip at the same rung, which is why nothing shipped.** The
      duel clears **B** between −95 (37.4%) and −90 (39.9%); the cast-wide pair
      saturates squirtle at **100%** between −95 (99.0%/98.6%) and −90
      (99.0%/**100.0%**). Same two points of defence, 366 → 368, both directions at
      once — because both are one event, a strike crossing a kill threshold. So
      **every amount that makes the dragon build a real matchup also makes
      `reckless` the 100%-against-the-cast trait `blood_thirst` was refused for
      being**, and there is no value that passes both. Reported and **not shipped**;
      the data is unchanged and the floor in
      `TestTheDragonBuildIsASidegradeAndNotAnUpgrade` therefore **stays at 150**,
      since the duel still reads 22.1%. What the trait needs next is a *different
      kind* of cost — one the duel prices and the cast-wide matchups do not — or the
      acceptance that the 22% is a statement about `inferno` and belongs to the fire
      line's detonate. All three of `reckless`'s own dials are spent.
      ⚠️ **A `weigh`-shaped instrument for a trait is the tool that is missing**,
      and it is its own piece of work. Everything above was measured by editing
      shipped JSON, running a duel, and putting it back: there is no way to sweep a
      trait's field the way `weigh` sweeps a skill's, so every reading costs a
      hand-managed data mutation and cannot be reproduced from the repo. A trait
      field table would have priced all four readings and both decompositions in one
      pass. The magnitude sweep found a third of it: patch the parsed
      `statuses.json` **in memory** and rebuild the status, passive and skill books
      around it, which is exact and reproducible and needs no file put back. What is
      still missing is somewhere for the answer to live that is not a test log.
      → `TODO.md` for the `weigh` field-coverage item this joins.
      → `README.md` § *What the dragon line's detonate was worth*, § *What
      `bare`'s dodge clause was worth* and § *What softening `bare`'s defence was
      worth*.

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
  → `docs/balance.md` § Rating an action; `TestAPassBuysNoCooldownAnActDoesNot`,
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
  → `docs/balance.md` § Rating an action.
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
