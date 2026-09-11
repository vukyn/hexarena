---
name: both-aiming-goldens-compose-to-neutral
description: "internal/screen's `aiming` and `aiming at an area skill` states are matchup-blind: the fixture caster is neutral-on-neutral and the area caster throws water at water/ice, both composing to 1000 — an aim-row feature moved 0 golden lines in all three screens.golden."
metadata:
  type: feedback
---

Neither existing aim-list golden can picture an **elemental matchup**, so a
feature that puts one on an aim row is accepted with `make golden` moving
nothing at all.

**Why:** measured while adding the matchup mark (`internal/screen`, 2026-09-11).

- `aBattleAiming` runs on `battleCast`, which ranks the cast by trait count then
  kit width — that lands on `pokemon.mew` (**neutral**) throwing `psychic`,
  `body_slam`, `dream_eater` (all **neutral**) at a neutral, a grass and a ground
  defender. Every row composes to 1000.
- `aBattleAimingAnArea` runs on `anAreaCaster`, the first character with a
  multi-cell shape; on the fixture library that is a **water** caster against a
  **water/ice** half — water beats neither water nor ice, so it composes to 1000
  as well. A "dual-element defender" in the fixture is not automatically a
  non-neutral matchup.
- `cmd/hexarena-tui`'s `aiming in a live battle` is built from the same
  `atABattleOf`/`battleCast` fixture, so it is blind the same way.

Result: the whole suite, all three `screens.golden` included, stayed **green**
with the feature shipped and would have stayed green with it deleted. The fix
was a fourth aiming state built from a matchup found **by property** — the widest
set of distinct answers any one skill produces against the cast — which records
a `-` aim row, a `+` splash row, an empty splash cell and a neutral row in one
list (164 insertions, 0 deletions).

**How to apply:** before claiming "the goldens cover it", check the *values* the
fixture actually produces, not that the screen is drawn. For anything keyed on
the element chart, the fixture cast's affinities are the thing to read. A new
golden state is cheap and additive; editing the fixture cast is not (one skill
into a fixture kit once moved 656 golden lines).

Related: [[feedback_a_golden_screen_shows_only_the_rows_that_fit]],
[[feedback_two_screen_goldens]],
[[feedback_a_fixture_cast_edit_costs_goldens]],
[[feedback_the_fixture_decides_what_is_visible]].
