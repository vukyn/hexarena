---
name: hexarena-cast-authoring
description: "hexarena hexforge: character = definition, roster = placement; cast/origin/archetype books; carry rule lives in skill.CanCarry"
metadata: 
  node_type: memory
  type: project
  modified: 2026-08-28T10:59:21.439Z
---

hexarena gained a second binary, `cmd/hexforge`, plus a pure package `internal/core/cast` (2026-08-25). Roadmap item (5) "A real cast" is now half done: the tooling and data shape exist, the actual cast does not.

**The split that drove the design: a character is a *definition*, a roster entry is a *placement*.** `roster.json` used to conflate them. Now:

- `origins.json` — the work a borrowed character comes from (id, title, medium, year). The user borrows characters from films/anime/games rather than inventing them, so origin is first-class catalog data, not a tag.
- `archetypes.json` — a preset per role (`bulwark` `vanguard` `sentinel` `duelist` `skirmisher`): a full `progression.Table` plus a suggested kit and column. Level-60 values equal the existing `roster.json` numbers, which is why adding all of this moved **no** existing golden.
- `cast.json` — the characters: origin, archetype, image, element, bio, evolution `Stages`, kit.
- `roster.json` — gains `{character, level, side, slot}`; the flat form still parses; mixing the two on one entry is rejected.

**`progression.Line` finally has a caller.** It supported staged evolution and nothing used it; a character's stats are a `Line` and one shipped example has two stages.

**Two rules worth not re-deriving:**

1. **Image path: shape in core, existence in cmd.** `internal/core` may not read the filesystem, so `cast.ValidateImagePath` checks only extension / relativeness / `..`; `hexforge check` is the single place that asks whether the file is really there.
2. **`skill.CanCarry` is the only statement of the carry rule** (a unit may use only a skill of an element it shares, neutral excepted). It was inline in `battle.enlist`; a character could be authored, validated, written and *then* rejected at battle load with no way for the author to know the rule existed. Called by engine, parser and tool. Archetype element demand is **derived from the kit**, never authored, so it cannot drift.

**Effective-HP budget is where presets bind:** bulwark absorbs 11214 and sentinel 11397 of the 11500 cap (hp×def multiply), leaving almost no room to buff defence; duelist 5383 and skirmisher 4036 have thousands spare. Not a balance error — damage roles spend their budget on atk/spd, which that cap does not cover.

**TUI (2026-08-25).** `cmd/hexforge-tui` is a bubbletea full-screen client over the same books: cast browser (level 1..60, names the stage owning that level), new-character form with the hp×def budget bar and the carry check recomputed per keystroke, origins catalog, check screen. It refuses to start when stdout is not a terminal.

**The split that made it safe: `internal/forge`.** The authoring logic (`Library` load/save, `Draft.Resolve`, `Inspect`, the budget and carry helpers, and the *wording* of every refusal) used to sit in `package main` under `cmd/hexforge` where no second front-end could reach it. It is now a package both clients call. `internal/forge` **may** touch the filesystem — it is not under `internal/core`. `cmd/hexforge` keeps its behaviour and stays the one that works with a pipe.

**Deps:** bubbletea v1.3.10 / bubbles v1.0.0 / lipgloss v1.1.0, 21 indirect, first `go.sum`. `x/sys` pinned to v0.44.0 — v0.38.0 carries GO-2026-5024 (Windows-only, unreachable, but free to dodge). A dependency ban was written and then **removed on the user's instruction** — do not reintroduce it.

**A pty smoke test earned its keep:** model tests all passed while long bio/path text wrapped and pushed the footer off the bottom of a real terminal. `View()` returning the right string is not the same as the screen being right.

**Class: dropped on purpose (2026-08-25).** A character *class* was in the original brief. Decision: archetype stands in for it, because with skills declared as data an archetype's curve + kit already say what a class name would. **Consequence worth knowing: an archetype has NO mechanical effect** — it never reaches the engine (`battle.Roster` = stats, skills, affinity, slot, nothing else) and nothing branches on one. Two units differ only by numbers and kit. Don't build on archetype thinking it does something; don't add a class without answering what it does that a curve and a kit cannot.

**Passive skills: SHIPPED — this paragraph is the original design note, kept for its reasoning, not a work list** (corrected 2026-08-28: it said "zero passives in the code", which stopped being true long ago). `internal/core/passive` is a package, `passives.json` ships ten traits, and resistance landed at `battle.inflict` rather than the `status.Set.Apply` this note proposed — see [[hexarena-mechanics-log]] for what each PR actually did. The rest below is why the shape is what it is. Of the four jobs a passive does, three REUSE existing homes rather than growing a parallel vocabulary — stat changes → `status.Kind.Modifiers` (which also inherits the saturation, so a passive can't compose with temporary buffs and explode), extra effects → `skill.Application`, conditions → `skill.Condition`. **Immunity is the only one with no home**: nothing can refuse an application, and `status.Set.Apply` is the choke point it belongs at (open: category vs status id, absolute vs saturating resistance — a hard cap on a continuous quantity has been the wrong answer everywhere else here). Two traps: a passive that changes a number **must emit an event** or the log can't explain its own figures; an enlist-time passive touching speed must apply **before the first wait is computed** (turn order = 1e6/speed; `retuneAll` exists because this was already got wrong with haste). Proposed shape: `passives.json` validated cross-book, `Passives []string` on `cast.Character`, archetype may suggest one — that would be archetype's first mechanical weight.

⚠️ **`forge.SkillAnswers` ⟷ `forge.resolveOnto` are hand-maintained inverses with no compile-time link, and a miss is a SILENT DATA WIPE** (PR #115, 2026-08-28). `resolveOnto` assigns a field on the base skill; `SkillAnswers` turns a skill back into the form answers. Any field `resolveOnto` **assigns** that `SkillAnswers` forgets to read back is overwritten with a zero on every edit — including an edit that never touched it. Fields `resolveOnto` leaves alone (`scaling`, `requires`, `self_requires`, `strips`, `summons`) ride through on the base skill and are safe.

This has cost two fields: `restrict.species` first, then `flavour` — every balance edit in either front-end deleted the authored clause, and nothing said so at the time. Both front-ends prefill from `SkillAnswers`, so the fix is one line there; **never** patch it in `cmd/hexforge-tui/skills.go` by reading the skill directly, which fixes the TUI and leaves the CLI wiping.

The guard is `TestEveryShippedSkillTakesABalanceEdit`: it now compares the **whole skill** with the edited field put back, rather than a chosen field. It watched only `.Restrict` before — written for the species loss — which is exactly why it sat green through the flavour loss. Add a field to `SkillDraft`, and check this test still passes; it is the only assertion that does not need extending each time.

**The trap has no second site — checked, 2026-08-28.** An earlier note here said the cast/archetype edit paths were unaudited for it; they do not exist. `EditSkill` is the only edit in `internal/forge` and `SkillAnswers` is the only inverse in it, so a skill is the only thing this tool can edit and the only thing this class of bug can reach. Do not go looking for the others.

**How to apply:** the game boots from the `go:embed` copy, so editing `internal/seed/data` needs a rebuild before it reaches a battle — `hexforge check` reads the directory, not the embed, and says so. Related: [[hexarena-core-design]].
