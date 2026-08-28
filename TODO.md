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
  own wounds. Summons. Taunt.
- **Traits.** A character carries traits as well as a kit: permanent grants,
  gated grants that come and go, resistances, replies to whatever attacked,
  amplifiers, drains, and a permanent speed change.
- **Progression.** Learnsets as unlocks, a placement choosing four skills and one
  trait, evolution stages as an allowlist, and late-game builds as data with a
  screen of their own.
- **Authoring.** `hexforge` (CLI, for pipes) and `hexforge-tui` (full screen)
  over one `internal/forge`, so the two cannot disagree. Skill authoring and
  editing, art picker, kit and allowlist pickers, budget bounds, spar, `weigh`
  and `check`.
- **Reference screens.** Statuses, traits, elements, species, and the affinity
  chart drawn as closed ASCII loops in element colour.
- **Vietnamese.** The TUI is Vietnamese-first with an English toggle; every
  description is derived from the data, and only the flavour clause is authored.
- **The opponent.** `Suggest` prices statuses, buffs, guards, heals, cleanses,
  kills, summons and **tempo** in damage over capped horizons — tempo off the
  speed stat, never off the queue. Both halves of an all-sided skill, and a tie
  broken by what an option costs to have spent rather than by kit order. The roster
  was re-levelled once along the way, which is what makes every rate quoted anywhere
  comparable.

## Not done

- [x] **Every skill's range re-read under the rank rule. Done** — 21 of 31
      enemy-aimed ranges moved onto the depth tiers (1 contact, 2 over the line,
      3 the back line), and `maxRange` tightened to the three ranks a side has.
- [x] **`roster.json` re-levelled under blocking. Done** — each ace moved to its
      own **back** column behind the two young units, on both sides, and nothing
      else changed: 27.6% ally → **47.3%** over 4000 seeds, 12/40 → 24/40 on the
      smoke test. It was a placement answer as filed. ⚠️ Placement is purely
      defensive now, so the shape is guarded by
      `TestTheShippedFormationScreensItsAce` rather than left to whoever edits the
      file next. The levels were **not** touched: the 20..30 dial spans 40–82% on
      the screened board, and 30/30 is the bottom of it.

- [x] **An evolution line that forks. Done** — `Stage.After` names a stage's
      predecessor, so a line is a tree: two arms may share a threshold and a stage
      may sit past a fork on one arm. ⚠️ A line is read **by order or by name,
      never both**, and `Furthest` — the half that would have failed silently —
      refuses on a fork instead of picking an arm. Nothing shipped forks yet, on
      purpose. → `CLAUDE.md` § Open work.
- [ ] **Graphical client with ebiten.** A renderer over `[]Event` and nothing
      more — it must not read `*Battle`. Asset pipeline undecided.
      → `CLAUDE.md` § Open work.
- [ ] **Grow the cast.** Four ship across two origins, one per element (grass,
      fire, water, wind). This is content, and the constraints that bound it are
      written down. Read Squirtle first. → `CLAUDE.md` § Open work.
- [ ] **A deeper opponent — what is left is waiting.** Passing a turn because the
      next one is worth more needs a **lookahead**, and *where* in the order an extra
      turn falls needs the **queue**. Tempo, all-sided skills and not wasting a
      scarce skill are all priced. **Declining a losing turn is done** — an option
      priced below nought is no longer the fallback, so the opponent stops casting
      what it has itself priced as a loss; that is not waiting, which still needs
      to know what the next turn is worth. → `CLAUDE.md` § Open work.
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
- [x] **Vulnerability. Done** — a negative `Resists` share, composing through the
      same multiply. `Refused` stays one signed field; a negative is a share the
      target *added*. ⚠️ No shipped trait uses it yet: `reckless` ("liều mạng",
      which already keeps no guard back) is the natural first user and is a
      **balance** change, so it is data work rather than this. → `README.md`.
- [x] **`forge.PreviewDamage` reads the caster's own terms. Done** — the
      amplified figure is now the skill with everything it asks for holding
      (`requires`, `self_requires`, `self_gradient`), composed through the new
      `combat.Swung` so the preview and the battle share one expression.
      ⚠️ That extraction found the ordering — bonus first, share second — was
      guarded by **nothing**: swapping the two halves passed the whole suite.
      `TestSwungAddsTheBonusBeforeTakingTheShare` is the missing claim.
- [x] **`hexforge` can author a flavour clause. Done** — `--flavour` on both
      `skills add` and `skills edit`, and a prompt beside the name in the wizard.
      An empty string clears it, on the same terms as an allowlist. The parser's
      rules still apply through the flag rather than around it: a digit in a
      clause is refused at the write, which the end-to-end test asserts.
- [x] **`hexforge weigh` — an instrument that prices ONE field on ONE skill. Done.**
      The carrier fights a copy of itself whose only difference is that field, so
      the deviation from an even split is the field's whole worth: kit, stats,
      placement, level and turn order all cancel. Eight scalar fields; the sweep
      always inserts the skill's own value as a control, and **a control row that
      is not exactly 500‰ refuses the whole report** rather than printing a
      number. `worth` and `turns` are co-equal headline columns.
      ⚠️ **It replaced a measurement that was proven unsound**: the roster win
      rate is **non-monotone in ally damage** — +5% power on an ally-only skill
      *lowered* it 1.75pp, and the same change read +2.40pp before #136 and
      −2.14pp after. → `CLAUDE.md` § Pricing one number.
      **No balance data was authored**: `internal/core`, `internal/seed/data` and
      every golden are untouched.
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

- **`at_stage` on a learnset entry.** Unblocked and deliberately not built:
  `at_stage: "Ivysaur"` is exactly `stages: ["Ivysaur","Venusaur"]`, and two
  vocabularies for one idea is the cost. → `README.md`.
- **A character class.** An archetype's curve and kit already say what a class
  name would, and an archetype has **no** mechanical effect. Do not add one
  without answering what it does that a curve and a kit cannot.
- **A dependency ban.** Written, then removed on the author's instruction.
  `internal/core` importing nothing outside the standard library is the rule
  that matters; the rest of the tree may use what it needs.
