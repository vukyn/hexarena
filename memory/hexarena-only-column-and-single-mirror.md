---
name: hexarena-only-column-and-single-mirror
description: hex.Place's 180° rotation makes `single` and `column` the ONLY shapes in patterns.json that are the same shape on both halves; any other shape on an ally-aimed skill is a different skill depending which side you stand
metadata:
  type: project
---

`hex.Place` maps an enemy formation through 180 degrees, and on the splash
directions that rotation reads **`up`↔`down`, `upper_right`↔`lower_left`,
`lower_right`↔`upper_left`**. Run every shape in `patterns.json` through it and
only two come back as themselves:

| shape | splash | closed under the rotation |
|---|---|---|
| `single` | — | **yes** (nothing to map) |
| **`column`** | `up`, `down` | **yes** (the pair swaps into itself) |
| `flank_up` / `flank_down` | `up` / `down` | no — each becomes the other |
| `wedge_right` / `wedge_left` | `upper_right`+`lower_right` / the mirror pair | no — each becomes the other |
| `pierce` | `upper_right`, `upper_right`×2 | no — it points down-left on the far half |
| `arc_up` / `arc_down` | `up`+`upper_right` / `down`+`lower_right` | no |

**Why it matters, and it is not symmetry-for-its-own-sake.** `forge.FightSquads`
fights every squad on **both** halves over the same seeds — that swap is the
measurement, not a refinement — so a shape that is a different shape on the enemy
half is a skill worth two different amounts in one report. All **14** ally-aimed
skills in the shipped book sit on `single` (9) or `column` (5), and that is very
likely the reason rather than a coincidence: a support skill has to be worth the
same thing in both arrangements or its carrier's rate is an average of two games.

⚠️ **It cancels on the five shipped squads and that hides the rule.** Their
formations are row-symmetric, so *every* shape reads the same on both halves of
`s01`–`s05` — measured, shape for shape, in `DAT-009`. The divergence is real and
visible only on `roster.json` (`arc_up` reaches 2 on one half and 3 on the other,
`wedge_right` 3 and 2), which `DAT-007` recorded. So a board test cannot catch
this, and neither can a squad rate: it is a property of the shape, not of the
data, and it has to be checked by reading the splash directions.

**How to apply.** Before putting an **ally-aimed** skill on any shape other than
`single` or `column`, check both halves explicitly — `hex.Place(hex.SideAlly,
slot)` and `hex.Place(hex.SideEnemy, slot)`, not the max of the two. The reduction
matters: `mostOccupiedCellsCaught` in `internal/seed/areaboard_test.go` takes the
**better** half, which is the right question for an enemy-aimed shape (a squad
only needs one arrangement where its area skill pays) and the wrong one for an
ally-aimed shape (a support skill is bounded by its worse half). That is why
`TestAnAllyAimedShapeIsCatchableFromItsOwnCarriersHalf` is a sibling rather than a
widening.

⚠️ Do not turn "the halves agree" into an assertion over shipped data: no shipped
squad formation can exercise its failing branch, which is
[[fixture-hidden-branch]] exactly. Log it.

→ [[hexarena-an-area-kit-has-no-finisher]] is the hostile half of the same
geometry; `DAT-009` in `TODO.md` is the friendly half and the measurement that
closed it.
