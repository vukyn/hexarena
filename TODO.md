# TODO

A short index of what is done and what is not. It is deliberately **thin**:
nothing here explains a design, because the explanations already live in
`CLAUDE.md` (the constraint each piece has to respect) and
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
            visibly disagree. **`CLAUDE.md` § Open work (now § *From CLAUDE.md § Open work* below) still says "nothing
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
            2026-09-04: fifteen characters ship** (this said twelve) and **ten**
            of them have an authored build (`builds.json`, unchanged) — the four
            without one inside the draftable pool are Happiny, Lapras, Oddish and
            Riolu. Ten picks is still the whole of the **built** cast, so a 5v5
            draft on builds alone has room for **nought bans**, and the build
            coverage rather than the cast size is now what binds. 3v3 is
            comfortable: six picks leaves four bans on the built cast, eight if
            the pool is the full fourteen.
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
            characters**, and § *Twenty-two traced Pokemon* holds eight lines of
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

- [ ] **Graphical client with ebiten.** A renderer over `[]Event` and nothing
      more — it must not read `*Battle`. Asset pipeline undecided.
      → § *From CLAUDE.md § Open work* below.
- [ ] **Grow the cast.** **Fifteen** ship across two origins (fourteen Pokemon,
      one Naruto) over **thirty-nine** authored stages, and **every one of the
      eleven elements is carried**: water ×3 (Lapras, Poliwag, Squirtle), grass ×2
      (Bulbasaur, Oddish), metal ×2 (Magnemite, Riolu), dark ×2 (Gastly, Mewtwo),
      neutral ×2 (Happiny, Mew), and one each of fire, ground, ice, wind, electric
      and light. Two duals: Magnemite (electric/metal) and Lapras (water/ice).
      This is content, and the constraints that bound it are written down. A
      character moves `cast.golden`, `species.golden` and `origins.golden` —
      **not** `scenarios.golden` or `replay.golden`, which this line claimed until
      2026-08-31. Read Squirtle first. → § *From CLAUDE.md § Open work* below.
      ⚠️ **The count in this line has been wrong four times** — it said three,
      then four, then five, then eight — so it is now derived rather than
      remembered: `jq '.characters|length' internal/seed/data/cast.json` and the
      element sweep below it. Do not hand-edit the number without re-running it.
      ⚠️ **No element is left to claim, so "one per element" is finished as a
      guide.** What a new character is bought for now is a **way of playing**, and
      the queue below is the art waiting for one.
      ⚠️ **On the archetype, which is the closest thing to a way of playing this
      data has a field for.** Fifteen characters against fifteen archetypes, one
      each — but that is where the counting has landed, **not a rule, and nothing
      enforces it**: `cast.ParseArchetypes` refuses an archetype *declared* twice
      and says nothing about two characters *tuned from* one, and no test in
      `internal/seed` asks. So a second character on a shipped archetype is
      **fine** and needs no permission. The aim is only that each character ends
      up unique in how it plays, and the archetype is one lever of several — the
      kit, the element, the stat table and the traits are the rest, and two
      characters off one preset with different kits are two ways of playing. The
      thing to avoid is a *duplicate*, not a shared preset. → § *Pricing one
      number*: what says two characters play differently is a measurement, not
      the field they were tuned from.
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
- [ ] **Twenty-two traced Pokemon are waiting for a character — eight complete
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

      Twenty-two files, eight lines, **no orphan form** — every line below is
      complete, and nothing `cast.json` names is missing from `assets/`.
      By line, as they would be authored:
      **pichu → pikachu → raichu** · **igglybuff → jigglypuff → wigglytuff** ·
      **mareep → flaaffy → ampharos** · **dratini → dragonair → dragonite** ·
      **abra → kadabra → alakazam** · **gible → gabite → garchomp** ·
      **magikarp → gyarados** · **onix → steelix**.
      ⚠️ **Nothing references any of these**, which is why they moved no golden
      and why `TestTheShippedArtIsCutOutRatherThanFramed` does **not** cover them —
      that test walks the art shipped characters name, so the day one of these is
      authored is the day its picture is first measured. The sources were checked
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
      four obvious axes outright.** Multiplicity across the fifteen shipped
      characters:

      | axis | most units that can share one value | rungs reachable |
      |---|---:|---|
      | element | 3 (water: Lapras, Poliwag, Squirtle) | 2, 3 |
      | origin | **14** (`pokemon`) | 2, 3, 4, 5 — but see below |
      | species | 2 (`plant`, `mythic`; every other species is 1) | 2, and only on two species |
      | archetype | 1 (fifteen characters, fifteen presets) | **none** |
      | archetype `column` | 6 / 5 / 4 for columns 0 / 1 / 2 | 2, 3, 4, 5 on all three |

      ⚠️ **An origin threshold is FREE at every rung today**, and that is the
      sharpest thing here: fourteen of fifteen characters are `pokemon`, so *any*
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
         knowing why this is not free: § *Grow the cast* below measures the
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
      where it is produced — is **answered by this file's own § Decided against**:
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
      → **The same root has an open entry below**, "a `hexforge new` still churns
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

## From CLAUDE.md § Open work

⚠️ **Moved here on 2026-09-05, verbatim.** It lived in `CLAUDE.md`, which is
loaded whole at the start of every session, and it is bookkeeping — three open
items and twenty-five done — which is this file's job rather than that one's. It
is kept as its own section instead of being merged into § *Done* and § *Not done*
above **because a merge is twenty-eight hand edits and a verbatim move is
provable**: no line of it changed.

⚠️ It is **not** a duplicate of the sections above. Measured before the move:
of its twenty-eight item titles, three also appear in this file and fifteen in
`README.md` — **twelve appear in neither**, so this section is the only home for
them.

Detail and the open questions are in `README.md` under Roadmap. What matters here
is the constraint each piece has to respect.

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
- [ ] **Graphical client with ebiten.** A renderer over `[]Event`, nothing more.
      It must not read `*Battle`, and it must not need the engine to know how long
      an animation takes. Asset pipeline is undecided: SVG has to be baked to PNG
      at build time or rasterised at load, because ebiten draws neither.
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
- [ ] **Grow the cast.** **Fifteen** characters ship across two origins —
      fourteen out of Pokémon and Naruto out of his own — over **thirty-nine**
      authored stages, and **all eleven elements are carried**: water ×3 (Lapras,
      Poliwag, Squirtle), grass ×2 (Bulbasaur, Oddish), metal ×2 (Magnemite,
      Riolu), dark ×2 (Gastly, Mewtwo), neutral ×2 (Happiny, Mew), and one each of
      fire, ground, ice, wind, electric and light.
      ⚠️ **The count in this line has been wrong five times.** It said "three, one
      per element" until 2026-08-28, "four, one per element" until 2026-08-31,
      "five" until Magnemite, and "eight" until 2026-09-04; #98 landed Naruto and
      #182 landed Poliwag, which is where one-per-element stopped being true, and
      **every element has been claimed since**, which is where it stopped being a
      guide at all. Derive the number rather than remember it —
      `jq '.characters|length' internal/seed/data/cast.json` — and do not edit it
      by hand.
      ⚠️ **What a new character is bought for is now a WAY OF PLAYING**, since
      there is no element left to be first at. The closest field to that is the
      archetype, and the count there is fifteen characters on fifteen presets, one
      each — but that is where the counting landed, **not a rule, and nothing
      enforces it**: `cast.ParseArchetypes` refuses a preset *declared* twice and
      says nothing about two characters *tuned from* one, and no test in
      `internal/seed` asks. **A second character on a shipped archetype is fine.**
      The aim is that each character ends up unique in how it plays, and the
      preset is one lever of several — the kit, the affinity, the stat table and
      the traits are the rest, and two characters off one preset with different
      kits are two ways of playing. Avoid a duplicate, not a shared preset; and
      per § *Pricing one number*, what says two characters play differently is a
      measurement rather than the field they were tuned from.
      ⚠️ **Magnemite is the first character to declare two elements**
      (`"element": ["electric", "metal"]`), and the field has accepted an array
      since the chart was written — `element.Dual`, `ValidateAffinity` and
      `MultiplierAgainst` were all shipped with no user. What the pairing turned
      out to be worth is written down in `internal/seed/dual_test.go`: a pair
      whose halves are unrelated mostly *cancels* — electric/metal is countered by
      four elements as two singles and by **one** as a pair — so the defensive half
      of a dual is close to nothing, and what is bought is the second skill book.
      The seed roster is no longer a mirror — so the thing this item was
      blocking, a measurable balance figure, exists. What is left is content,
      under three constraints: an archetype's kit constrains a character's
      affinity (`skill.CanCarry` enforces it while authoring, `Archetype.Demands`
      reports it), `progression.Limits` bounds health and defence **together**
      because those two multiply, and — softer than the other two — a skill kept
      for a lineage asks the character to *be* one, so adding a dragon is two
      lines: the kind in `species.json` and the claim on the character. **A new
      skill also has to say which story it is out of** — `restrict.origins`, or a
      line in `sharedPool` arguing it belongs to nobody;
      `TestEverySkillSaysWhichWorkItIsFrom` refuses the omission.

      ⚠️ **A character moves `cast.golden`, `species.golden` and `origins.golden`
      — not `scenarios.golden` and not `replay.golden`.** This item claimed the
      opposite until 2026-08-31, and the two items above that say a cast-only
      addition moves no golden were right all along: `replay.golden` renders the
      **roster**, so a character reaches it by being placed in `roster.json` and
      by nothing else. Add a preset and `archetypes.golden` moves too; add skills
      and `skills.golden` and `describe.golden` move. #182 added a character, a
      preset and four skills, and moved exactly those six.

      Read squirtle first — water is the strongest of the elements and Blastoise
      still cannot carry an ace slot, because its attack and speed curves are the
      lowest in the cast.
