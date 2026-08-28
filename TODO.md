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
  be improved. `Suggest` beats it **77.9%** over 10,000 seeds, band ±0.8pp, and
  finishes sooner (45 turns against the control's 48).
- **Balance.** Every enemy-aimed range re-read under the rank rule, and each ace
  moved to its own back column behind a screen — 27.6% ally → **47.3%**. Both were
  data answers, and the formation is guarded by a test rather than by whoever
  edits the file next. ⚠️ The levels were deliberately **not** touched: the 20..30
  dial spans 40–82% on the screened board, so it is not the lever it looks like.
  A field is priced with `weigh` against a copy of its own carrier, because the
  roster win rate is **non-monotone in ally damage** and prices nothing.

- **Squads.** A side is built in the TUI — who is in it, what each brings, where
  each stands — saved to `squads.json`, and fought against another over N seeds
  **both ways round**, which is what makes a mirror read exactly even instead of
  reporting the first slot's advantage as the squad's — or played by hand against
  the engine, one battle at a time, with undo — and written out as a log the
  game client replays and verifies.

## Not done

- [ ] **Graphical client with ebiten.** A renderer over `[]Event` and nothing
      more — it must not read `*Battle`. Asset pipeline undecided.
      → `CLAUDE.md` § Open work.
- [ ] **Grow the cast.** Four ship across two origins, one per element (grass,
      fire, water, wind). This is content, and the constraints that bound it are
      written down. Read Squirtle first. → `CLAUDE.md` § Open work.
- [ ] **A deeper opponent — what is left is the queue.** *Where* in the order an
      extra turn falls is the one thing `Suggest` still cannot see, and it lands as
      a **tie-break** rather than a term: a queue reading may be **compared**, never
      added or multiplied, because a value that reaches an arithmetic expression is
      tempo and tempo is priced from the speed stat. Everything else is priced —
      tempo, all-sided skills, not wasting a scarce skill, and declining a losing
      turn. Waiting is decided against; see below. → `CLAUDE.md` § Rating an action.
- [ ] **`reckless` is the dragon build's 22.1%, and a detonate does not fix it.**
      The trait grants `unleashed` (+300‰ attack) **and** `bare` (−400‰ defence
      *and* −400‰ dodge), all permanent: two stats paid for one, into a matchup
      whose `inferno` amplifies ×3.5 off a status. Swapping it for `blood_thirst`
      reads 55.1% and for `blaze` 38.9%; the missing detonate is worth −0.8 by
      comparison, so **the theory this item was filed under is disproved and
      should not be re-raised**. The levers are all data — soften `bare`, drop its
      dodge clause, or raise `unleashed`.
      ⚠️ **Whatever lands has to land with `vulnerability`, not before it.** A
      negative `Resists` share is built and `reckless` is its natural first user;
      adding it alone sinks the build further, so the two are one change.
      ⚠️ `TestRecklessIsATradeAndNotAGift` asks whether *something* is given up
      and cannot ask whether **too much** is, which is why nothing caught this.
      → `dragon_test.go` § `TestTheDragonLineCanSpendWhatItApplies` for the table.
- [ ] **The crit balance data itself is still unwritten.** #135 shipped the
      mechanic inert — **every shipped skill declares `crit: 0`** — and the
      instrument to price one now exists, so what is left is the authoring: which
      skills crit, at what chance, and what each is worth to its carrier.
      ⚠️ A weigh figure is a price on **one carrier at one level against itself**.
      It is not a win rate and **it does not carry across a data change**, so a
      number taken now has to be re-taken after the roster or a placement moves.
      Read `worth` and `turns` together — a row inside the band whose turns moved
      is a real effect. → `CLAUDE.md` § Pricing one number.
- [ ] **A `--carriers all` sweep for `hexforge weigh`.** Unblocked now that
      `Weigh` exists and deliberately not built with it: one carrier is one
      question, and a table over the whole cast is a different one that needs to
      decide what an average of prices taken against different opponents means.
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
- **`at_stage` on a learnset entry.** Unblocked and deliberately not built:
  `at_stage: "Ivysaur"` is exactly `stages: ["Ivysaur","Venusaur"]`, and two
  vocabularies for one idea is the cost. → `README.md`.
- **A character class.** An archetype's curve and kit already say what a class
  name would, and an archetype has **no** mechanical effect. Do not add one
  without answering what it does that a curve and a kit cannot.
- **A dependency ban.** Written, then removed on the author's instruction.
  `internal/core` importing nothing outside the standard library is the rule
  that matters; the rest of the tree may use what it needs.
