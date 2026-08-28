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
  elemental chart. Verifiable logs, replay, undo. Draws for a battle nobody can
  act in and for a deadlock. Piercing, healing, draining, regeneration.
  Conditions read the target *or* the caster. Summons. Taunt.
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
  kills and summons in damage over capped horizons.

## Not done

- [ ] **An evolution line that forks.** A placement chooses how far along one
      path, never which path. ⚠️ The parse rule is the small half; `Furthest`
      is the large half and fails **silently**. → `CLAUDE.md` § Open work.
- [ ] **Graphical client with ebiten.** A renderer over `[]Event` and nothing
      more — it must not read `*Battle`. Asset pipeline undecided.
      → `CLAUDE.md` § Open work.
- [ ] **Grow the cast.** Three characters, one per element. This is content, and
      the constraints that bound it are written down. Read Squirtle first.
      → `CLAUDE.md` § Open work.
- [ ] **A deeper opponent — the three pieces left over.** The turn queue (so a
      speed buff is worth nothing to the rating), holding a skill for a later
      turn, and all-sided skills. → `CLAUDE.md` § Open work.
- [ ] **Re-level `roster.json`.** The deeper opponent moved the shipped roster
      from 53.1% to 79.0% ally. That is a cast finding, and the follow-up is a
      **data** change. → `CLAUDE.md` § Open work.
- [ ] **Vulnerability.** A target *easier* to poison is `Resists` with a negative
      share, which reuses the whole composition. Needs a decision about a
      negative `Refused` in the log — `resist` returns early when nothing is
      refused and would drop it. → `README.md` § Roadmap.
- [ ] **A damage gradient off the caster's own health.** `SelfRequires` is a
      *threshold* ("below 40%, +bonus"); a gradient that grows as the caster
      falls is a multiplier in `combat` reading the other unit. Separate feature.
- [ ] **`hexforge` cannot author a flavour clause.** The CLI stopped *wiping* it
      (#115) but still cannot set one — there is no `--flavour` flag, so a clause
      can only be written in the TUI or by hand in JSON.

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
