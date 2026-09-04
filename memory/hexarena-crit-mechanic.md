---
name: hexarena-crit-mechanic
description: "hexarena crit: per-skill chance, game-wide x1.25, per-strike, event flag; #135 mechanic + #148 first two chances priced by weigh — a theme is not a price"
metadata: 
  node_type: memory
  type: project
  modified: 2026-08-28T19:01:35.330Z
---

Critical hits did not exist in hexarena before 2026-08-28 (grep: zero hits for crit/critical/chí mạng/bạo kích anywhere). Author's settled design:

- **chance per skill** (`skill.Crit`, permille, `omitempty`) — never a unit stat: `progression.Values` is a fixed-size array behind a six-required-pointer schema, so a 7th kind breaks the load of `cast.json`/`archetypes.json`/`roster.json` and compile-breaks `profile(...)` in `cast_test.go`.
- **multiplier is one game-wide constant ×1.25** in `combat.Rules.CriticalMultiplier` / `combat.json`, `Validate` requires it.
- **rolled per strike**, only in `Roll`'s `default:` (landed) arm — a missed or blocked strike never happened.
- **event carries a bool flag**, not a permille int — deliberate deviation from `Pierce`/`Refused`/`Drained`/`Amplified*`, because those shares vary per cast while this multiplier never does.
- Vietnamese term is **"chí mạng"** (composed). The skill description names **the chance only, never ×1.25** — no other `combat.Rules` constant appears in a skill sentence, and naming it would force a `rules` param on `Describe` and an `i18n → combat` import.

**Two levers make this cheap.** `rng.Source.Chance` returns *without drawing* at `permille <= 0`, so shipping every skill at zero leaves the stream bit-identical (the trick `Pierce` used). And the crit factor folds into `Damage`'s numerator with a matching `PermilleBase` in the denominator — `floor(1000a/1000b) == floor(a/b)` — so the ordinary path is *provably* unchanged rather than probably. ⚠️ Never `Strike(h) * 1250 / 1000`: that is the second truncation the package exists to avoid.

**Balance data MERGED as `21ee773` (#148):** `razor_leaf` and `wind_shuriken` crit at **200‰** and nothing else does. Priced one carrier at a time with `hexforge weigh` (2000 seeds, band ±1.6%) — `razor_leaf` **+8.4%**, `wind_shuriken` **+6.9%**, and the three that did **not** ship are the lesson: `fire_fang` +1.0% (inside the band), `bite` +0.2%, `kunai` **−1.7%**. ⚠️ **A theme is not a price.** The rule they were first picked under — a crit belongs to a blow that strikes a point, not a spray — was wrong: `bite` and `kunai` are as pointed as `razor_leaf`. The price reads the **carrier's whole line**: a cheap skill (`kunai`, cooldown 0) given a crit gets *cast more often* and crowds out a better one — `outrage` wearing a different hat. `kunai` reading negative is the proof the instrument can see a buff make its carrier worse. Roster afterwards 47.56% → 49.73% ally, 8.02% of damaging strikes critical (observations; the roster figure prices nothing). Remaining 26 damaging skills still at nought, each its own reading.

**State: the mechanic MERGED as `74e5075` (#135), 2026-08-28.** Exactly two goldens moved (`combat.golden`, `skills.golden` — 43 rows, every other column including damage identical, crit column all zero); logs byte-identical to the pre-change binary across 12 seeds; pre-change logs still `--verify`. **PR 2 — the balance data — is deliberately unstarted:** the moment a skill declares a rate the stream diverges and the roster's rates move while every test stays green, so it needs a multi-thousand-seed reading before and after, one skill at a time. A shipped rate must be 0 or ≥ 10 permille (`TestNoShippedShareIsUnderOnePercent`) and must sit on a skill with power.

Two findings from it worth keeping: **a repo bug** — `TestEveryWordingFitsTheMinimumWidth` never measured most skill-form wordings, because only the *focused* field renders and `TestTheFormScrollsToTheFieldTheCursorIsOn` checked three hand-picked fields; the loop was widened to all of `skillFieldCount`. And **the ratio `CriticalStrike/Strike` is not an invariant** — both are floored integers, so it wobbles a part per thousand; assert absolute figures instead.

Related: [[hexarena-core-design]], [[hexarena-mechanics-log]], [[pin-comparison-base-to-worktree-head]].
