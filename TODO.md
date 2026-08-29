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
  refusing on a fork rather than picking an arm.
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

## Not done

- [ ] **Graphical client with ebiten.** A renderer over `[]Event` and nothing
      more — it must not read `*Battle`. Asset pipeline undecided.
      → `CLAUDE.md` § Open work.
- [ ] **Grow the cast.** Four ship across two origins, one per element (grass,
      fire, water, wind). This is content, and the constraints that bound it are
      written down. Read Squirtle first. → `CLAUDE.md` § Open work.
- [ ] **The rest of the cast still crits at nothing.** Two skills carry a chance
      now, chosen by what `weigh` priced rather than by what the names suggest.
      The other twenty-six damaging skills are open, and each is its own reading:
      the price is a fact about the **carrier's whole line**, not about the
      skill's shape. ⚠️ Do not author one from a theme. `bite` and `kunai` are as
      pointed a blow as `razor_leaf` is and priced +0.2% and **−1.7%**; a cheap
      skill given a crit gets cast *more* and crowds out a better one, which is
      the `outrage` lesson wearing a different hat. → `CLAUDE.md` § Pricing one
      number.
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
- [ ] **The battle screen does not fit the window the tool declares.** Its body
      is **28 lines** and `frame` gives the body `m.height - 2` less the two the
      header takes, so at 80x24 only twenty survive and the whole option list is
      cut with the `Truncated` marker. Measured in both languages: the list first
      appears at **h ≥ 30**, the screen is un-truncated from **h ≥ 32**, and from
      **h ≥ 38** once the log tail has filled to its eight lines.
      ⚠️ `TestTheBattleScreenIsNoTallerThanItAlreadyWas` is a **tripwire and not a
      bound** — it asserts today's number as a ceiling and says so, because a test
      claiming the screen fits would ship red, which is the mistake the `reckless`
      work already made once. What is open is which rows the screen gives up:
      `playLogLines` is eight of them, and the board, the roster and the order
      line are `internal/tui`'s drawing rather than this screen's, so trimming any
      of them changes what a played battle looks like against what the game client
      plays. That is a layout redesign and its own piece of work.
      → `CLAUDE.md` § *Where a form beats a prompt* → the played battle.
- [ ] **A list of three or more reads `X and Y and Z`.** `l.join` puts the
      language's conjunction between every pair and no comma anywhere, so
      `purify` strips `damage over time and stat reduction and turn loss`. It is
      `join`'s shape for **every** list in `describe.go`, not this clause's, so
      what is open is one decision about all of them — and the two languages do
      not necessarily want the same answer.
- [ ] **An English trait row is a bare id with an empty column after it.** A
      trait's name is authored in `passives.json` in Vietnamese only, so the
      picker's detail cell comes back empty in English and the row draws its id
      and then padding. Pre-existing and wider than the picker — the trait
      listing drops the column outright in English, which is the house rule a
      table follows — so what is open is which of the two the picker should be,
      and whether an English name belongs in the data at all.

## Decided against — do not re-raise

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
