---
name: hexarena-refusal-wordings
description: hexarena PR3/3 (2026-09-04) — battle.Block wordings + screen.OptionRefusal + the gated-condition opening; traps that made a sweep measure nothing
metadata:
  type: project
---

hexarena branch `feat/the-refusal-a-reader-can-act-on`, commit `f6cdc9d` on top of `bf370f3`. Third of three PRs: #286 added `Condition.Gates` + `battle.Option.Blocked/Turns/Status/Need/Held`, #292 shipped `heft`/`brace`/`wrecking_swing` on machop, this one moved the screens.

**Why:** `internal/screen/play.go` drew `option.Reason` raw — English, built in `internal/core` (which may not import `internal/i18n`) — on a Vietnamese screen. Fine while every refusal was a cooldown; fatal once a cooldown-0 skill can be greyed out, because it then explains itself with nothing.

**How to apply:**

- ⚠️ **A distinctness-only sweep can measure nothing by accident.** "Every block draws a different sentence" passed under the mutation that routed the cooldown through the fuel wording — the fuel arm *glosses* the status and the mutated arm did not, so the two rows differed by an accident of the gloss. Fixed by comparing each row against the wording its own block owns (values from the catalog, so a swapped/doubled arm still fails) while still **walking the enum** so a new unworded block fails.
- **Keep a new English wording byte-identical to the engine sentence it replaces** when you want a small golden diff: only the Vietnamese rows moved in all four screen goldens (`internal/screen`, `cmd/hexarena-tui`, `cmd/hexforge-tui`), the English half stayed put.
- ⚠️ Only a **caster-side** condition can gate — `skill.resolveCondition` refuses `requires.gates` outright — so the gated openings are self-facing only; keep the target opening reachable rather than mapping every gate onto the self wording.
- The compact summary's shape wording for a self condition ends in "spreads"/"lan" (a chain), which a caster's own condition is forbidden to do — a gate with no bonus figure fell straight into it.
- `Lang.Glossed(id)` renders `id <gloss>`, so it is **useless** for "names it by gloss and not by id" — use `Gloss(id)` with an id fallback (the pattern `browse.go`/`picker.go` already use).
- Left reading English on purpose, each with a comment: `internal/tui/tui.go` (no `Lang`, and a replay is drawn with no library) and `cmd/hexarena/main.go` (no `--lang`, whole client is English).

Related: [[hexarena-tui-i18n]], [[hexarena-cast-gate]]
