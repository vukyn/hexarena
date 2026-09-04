---
name: hexarena-builds-catalogue
description: hexarena builds.json + bulbasaur 2 builds (2026-08-28) — builds became DATA (cast.ParseBuilds) with names; DoT damage is unattributed; a mirror duel measures the twin
metadata: 
  node_type: memory
  type: project
  modified: 2026-08-28T08:25:22.586Z
---

**Shipped 2026-08-28 — PR #114, squash-merged `cda6199`.** README section + data-file row landed later in PR #117 (`86d148e`), so both records are complete. Late-game builds became **data**, and Bulbasaur got the measured-test treatment Charmander/Squirtle already had.

**New data contract** — `internal/seed/data/builds.json` (embedded) + `cast.ParseBuilds` in `internal/core/cast/build.go` (`Build`, `BuildBook.All/Of/Get/Count`) + `seed.Builds()` + `forge.Library.Builds()/BuildsOf()`. A build = `{id, character, name, intent, skills[≤4], passives[≤1]}`, validated **at `progression.LevelCap` on the furthest form** (that is what catches `sleep_powder`, stage-gated away from Venusaur). Parse refuses: unknown character, unlearned/duplicate entry, over-slot, duplicate id, duplicate **name within one character**, and **any digit in `name`/`intent`** (a build's numbers are its skills' numbers).

| nhân vật | build A | build B |
|---|---|---|
| Venusaur | **rải độc** poison_powder/sludge_bomb/venoshock/razor_leaf + `virulence` | **ký sinh** leech_seed/synthesis/ingrain/razor_leaf + `blood_thirst` |
| Charizard | **thiêu đốt** flamethrower/inferno/ember/fire_spin + `blaze` | **long tộc** dragon_claw/outrage/dragon_rage/dragon_dance + `reckless` |
| Blastoise | **cố thủ** taunt/withdraw/wide_guard/aqua_ring + `thorns` | **giáp kích** skull_bash/water_gun/whirlpool/withdraw + `ballast` |
| Naruto | **0 build** — deliberate; learnset has no second direction |

⚠️ **A DoT tick names no author.** Poison damage is `StatusTicked` with `Actor = the unit carrying it`; there is only ONE `Kind: Damaged` literal in `battle` (the passive reply). So "damage dealt by me" = `Damaged{Actor:me}` **+** `StatusTicked{Actor:them}` in a duel — and it cannot be attributed at all in a squad. Counting only `Damaged` hid a quarter of the poison build's output (106 vs 139/turn) — i.e. the metric punished exactly the build it was measuring. Also `Healed.Actor` is the unit **being healed** (not `Target`).

⚠️ **Two builds of one character cannot always be compared head to head, and the reason differs per character.** Squirtle: the tank kit has no power at all. Bulbasaur: both kits kill, but the mirror duel is decided by outlasting — sustain beats poison ~10.5% over 600 duels both ways, which measures the twin, not the build. Only Charmander's pair is a real sidegrade band (42.5%). Measure **shape against a fixed opponent** instead: poison 13 turns @139/turn healing 0 · sustain 17 turns @47/turn healing 964 (30 seeds vs shipped Charizard).

⚠️ **`virulence` looked WORSE than `endurance` on the poison kit** until the DoT ticks were counted (41740 → 54565 vs 53318). A trait that buys damage cannot be judged by a metric that misses where the damage lands.

**Design record stays hardcoded in tests**: `poisonBuild`/`sustainBuild` (`bulbasaur_test.go`), `fireBuild`/`dragonBuild`, `tankBuild`/`semiBuild`. `TestTheShippedBuildsAreTheOnesTheTestsMeasure` fails if builds.json drifts from them — kit **or** trait. New rule: **a character in the catalogue has ≥2 builds** (one build is a kit, not a choice); zero is honest.

**TUI**: `screenBuilds` in `cmd/hexforge-tui/builds.go` (menu row between species and check), grouped list, cursor skips character headings, ⚠️ the "no build yet" note rides **on the character's heading row** — as its own row it scrolled away from the character it meant. 9 new i18n keys. No golden moved (hexforge-tui owns none).

⚠️ **CLAUDE.md drift fixed in the same change**: platform root claimed hexarena is "standard library only" (false — bubbletea/lipgloss/bubbles v2, oksvg, rasterx; only `internal/core` is stdlib-only) and both CLAUDE.mds said `-update` lives in 3 packages (it is **4**: hex, i18n, seed, tui).

Related: [[hexarena-squirtle-builds]], [[hexarena-dragon-build]], [[hexarena-shipping-a-character]], [[hexarena-flavour-sweep-todo]], [[hexarena-reply-scaling]]
