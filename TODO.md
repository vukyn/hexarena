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
  kind it is drawing. → `cmd/hexforge-tui/picker.go` → `detailColumn`.
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
      the ones nothing else can start without.

      **Groundwork**
      - [ ] Factor the reference screens out of `cmd/hexforge-tui` into a package
            both binaries draw, and stand up `cmd/hexarena` as a full-screen
            client over it. ⚠️ **The biggest single item and the one that gates
            everything**: 10k lines of tightly-coupled bubbletea under 13.7k
            of tests, and the tests are the hard half — `everyScreen` is the
            harness, and a screen that moves out without being re-registered
            silently loses its width, translation and leak tests, which is a
            shape this file already records five times.
      - [ ] `internal/wire`: the protocol as one stdlib-only package. The
            envelope, the three version numbers, and error **codes** rather than
            prose. A golden per message, so a wire change shows up in a diff.
      - [ ] The data digest — the fifteen embedded JSON files, in `go:embed`
            order, hashed as bytes, no parsing. `assets/` excluded: art cannot
            reach the simulation.
      - [ ] Replace `Drain` at the server with an append-only record and a cursor
            per consumer. ⚠️ `Drain` **empties the buffer** and a room has two
            players, spectators and a log; the cursor is also what reconnect and
            mid-battle spectating are made of.

      **The room, with no network in it**
      - [ ] The room as a state machine over messages with no I/O: two fake
            clients drive a whole match in-process. ⚠️ Build it the other way
            round and this becomes the least-tested code in the repository.
      - [ ] Many rooms per process. A room owns its battle in **one goroutine**
            and shares it with nothing; the registry takes a mutex, a battle
            never does.
      - [ ] Validate a squad at the gate: `Squad.Validate`, then `Take` (which is
            already the loadout check), then the format's size, level 60, and a
            stage that is a **leaf** of the line. ⚠️ Not `Furthest` — that refuses
            on a fork, and `politoed` is queued above as the first fork.
      - [ ] Decide whether one squad may field the same character twice.
            `Squad.Validate` allows it today and checks only ids and slots.
      - [ ] The clock: ninety seconds a prompt, a timeout passing with a single
            constant reason. ⚠️ Never a timestamp into the battle. A `Skipped`
            prompt starts no clock. Three consecutive timeouts forfeit.
      - [ ] A forfeit, a disconnect and a refused join are results of the
            **match**. ⚠️ Nothing is added to `battle.Outcome` — a dropped socket
            is not a way a battle can end, and that enum is a core type.
      - [ ] A **series**, not a bo2: `battles: N` plus a rule for what ends it,
            from the room's first line, because *this* is the part that hurts to
            add later. ⚠️ **bo1 is not a special case — it is N = 1.** The room
            offers **bo1 and bo3**; bo2 is deliberately not offered, because only
            an even series cancels the side and only an even series has to invent
            a rule for a 1–1. The aggregate-health tie-break an earlier draft
            proposed is **dropped**, so no invented metric ships anywhere here.
      - [ ] One rule for bo1 *and* for the third battle of a bo3, which are the
            same problem: the seed picks the side, and the lead of each contested
            speed group alternates. ⚠️ Honestly uncancelled — say so rather than
            dress a coin as fairness.
      - [ ] The per-turn allowance belongs in the room's configuration beside the
            format. Measured on the shipped 3v3: **34–55 decisions a battle**, so
            ninety seconds each is 68 minutes a battle and 3.5 hours for a bo3.
      - [ ] A turn cap per battle so a stalemate ends. `Outcome` already has the
            draws.
      - [ ] Write each finished match out as a `battle.Log`, which makes every
            PvP match `--replay --verify`-able for nothing.

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
      ⚠️ **`politoed` is the odd one and the interesting one: it is a FORK.** The
      poliwag line already ships as `poliwag → poliwhirl → poliwrath`, and
      politoed grows out of poliwhirl as a second arm. The forking mechanism is
      built and `CLAUDE.md` records that **nothing shipped forks yet,
      deliberately** — so this is one `after` field away from being that
      mechanism's first shipped user, and it should be authored knowingly rather
      than as one more evolution. → `CLAUDE.md` § Open work, the forking entry.
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

- [ ] **`screenPreview` is still outside `everyScreen`**
      (`cmd/hexforge-tui/language_test.go`), so it has **no width test, no
      translation test and no leak test at all**. ⚠️ This is the **fifth**
      instance of a shape `CLAUDE.md` records having been made four times — after
      `plainTerminal`, `playScreen`, the species picker and the skill filter's
      states — and it is the one instance that has been *known* the whole time:
      the `CLAUDE.md` line describing the trait blurb already admits it ("Both
      blurb shapes are in it now; `screenPreview` still is not") and nobody filed
      it. The screen draws art, so it is also the one screen where a width test
      would be measuring a drawing rather than a sentence; decide what the entry
      asserts before adding it, or it will pass on nothing.
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
  → `CLAUDE.md` § Rating an action; `TestAPassBuysNoCooldownAnActDoesNot` and
  `TestNothingWaitsOnPurpose`.
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
