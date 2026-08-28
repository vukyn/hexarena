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
  editing, art picker, kit and allowlist pickers, budget bounds, spar, and
  `check`.
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
- [ ] **Re-level `roster.json` under blocking.** The range pass is what made
      holding a front rank decide fights — 14 of 31 skills now stop at the first
      rank — and the shipped roster was placed for a board where it did not. The
      instrument went 19/40 ally to 12/40 on the numbers alone, with both sides
      losing depth about equally, so this is a **placement** answer and not a
      skill one. → `CLAUDE.md` § Invariants.

- [ ] **An evolution line that forks.** A placement chooses how far along one
      path, never which path. ⚠️ The parse rule is the small half; `Furthest`
      is the large half and fails **silently**. → `CLAUDE.md` § Open work.
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
      The line has one now (`dragon_drive`, off its own `expose`) and fielding it
      moves 22.0% → **21.2%**. ⚠️ Measured one change at a time: the detonate is
      worth **−0.8**, the trait **+33.1**. Still a **data** answer, just a
      different one. → `dragon_test.go` § `TestTheDragonLineCanSpendWhatItApplies`.
- [x] **Vulnerability. Done** — a negative `Resists` share, composing through the
      same multiply. `Refused` stays one signed field; a negative is a share the
      target *added*. ⚠️ No shipped trait uses it yet: `reckless` ("liều mạng",
      which already keeps no guard back) is the natural first user and is a
      **balance** change, so it is data work rather than this. → `README.md`.
- [ ] **`forge.PreviewDamage` cannot see a caster-side term.** It reads only the
      target's condition, so neither `self_requires` nor `self_gradient` shows in
      the authoring preview — `outrage` and `comeback` both preview as their plain
      power. Pre-existing; one change covers both. → `CLAUDE.md` § Open work.
- [x] **`hexforge` can author a flavour clause. Done** — `--flavour` on both
      `skills add` and `skills edit`, and a prompt beside the name in the wizard.
      An empty string clears it, on the same terms as an allowlist. The parser's
      rules still apply through the flag rather than around it: a digit in a
      clause is refused at the write, which the end-to-end test asserts.

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
