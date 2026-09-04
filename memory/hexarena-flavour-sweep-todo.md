---
name: hexarena-flavour-sweep-todo
description: "hexarena — the skill-flavour and biography sweep shipped in PR #111 (2026-08-28); the user said they will re-read the prose themselves later, so expect a correction batch"
metadata: 
  node_type: memory
  type: project
  modified: 2026-08-28T09:43:57.097Z
---

**Sweep is DONE** — PR #111, merged 2026-08-28 (`origin/main` 93aa484). 29 of 41 skill `flavour` clauses rewritten, 12 left alone deliberately, and all 4 biographies in `cast.json` rewritten.

**The user's review arrived and is applied** — PR #115, merged 2026-08-28. Fifteen corrections; the calibration list in [[hexarena-flavour-voice]] has them. Two patterns in what they rejected, worth having in hand before writing the next clause: **obscure imagery** ("cả một trời nắng", "không khí nứt ra thành từng lằn", "nuốt cả tiếng động", "đất dưới chân nảy theo", "bao nhiêu đời") and **the wrong word for the thing** ("khét lẹt" is a *burnt* smell above a mud attack; a bucket of water nobody has in a battle). Colour is wanted; colour nobody can picture is not.

**What the sweep established, worth not re-deriving:**

- Two clauses were promising mechanics the engine does not compute — `ingrain` ("bám chặt tại chỗ": nothing roots anything) and `substitution` ("đứng dậy ở chỗ khác": nothing moves the caster, `veil` is an evasion buff). That failure mode is the one to check first on any new clause.
- The 12 left alone: `aqua_ring`, `dragon_claw`, `outrage` (the user dictated their wording — do not touch without being asked), plus `kunai`, `rasengan`, `taunt`, `rapid_spin`, `shadow_clone`, `summon_toad`, `water_pulse`, `skull_bash`, `wide_guard`.
- Biographies had **no rule at all** before this. Two now exist in `internal/seed/cast_test.go`: `TestABiographyCarriesNoFigure` (digit ban — "Ivysaur từ cấp 16" goes stale the moment a `min_level` moves) and `TestABiographyNamesNoLaterForm` (no evolution list; the **first** form is free, being the creature's name). A bio must not restate the browser's own `giai đoạn` / `hp hiệu dụng` rows, nor summarise tactics — the tactics are the kit.

Related: [[hexarena-flavour-voice]], [[hexarena-descriptions-are-derived]], [[hexarena-shipping-a-character]].
