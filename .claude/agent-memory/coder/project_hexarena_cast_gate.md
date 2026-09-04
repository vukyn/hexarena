---
name: hexarena-cast-gate
description: hexarena `gates` cast-gate (PR1 engine, 2026-09-04) — 3 traps that make a hexarena test pass while measuring nothing, plus the bench-vs-golden conflict
metadata:
  type: project
---

`skill.Condition.Gates` shipped 2026-09-04 on branch `feat/charge-and-release`
(commit "feat(core): let a condition gate its own cast", PR 1 of 3 — engine only;
data is PR 2, the glossed refusal PR 3). A gating condition is read *before* the
skill is offered, in `battle.options()` and again in `Battle.Act`, and
deliberately **not** in `aims()`.

**Why:** hexarena's tests are written to fail under a named mutation. Three
shapes here passed while measuring nothing, and none of them is visible by
reading the test.

**How to apply:** when writing a hexarena test that claims to hold a rating or a
status rule, check it against these before believing it.

- **Suggest's fallback arm makes a two-skill kit prove nothing about pricing.**
  A loop test on a kit of `[charger, gated spender]` charged 3 times and released
  even with the fuel priced at **0** — with the spender gated off, nothing else is
  on offer, so the fallback casts the cooldownless charger for want of anything to
  do. Adding a filler blow worth taking flipped it to 0 charges / 12 jabs / 0
  releases under the same mutation. **Any test asserting "the rating chose X"
  needs a rated alternative on the board.**
- **A status fixture whose `duration` >= the depth it banks hides
  `status.Apply`'s refresh loop.** With duration 5 and a 5-deep tank, deleting the
  refresh loop changed nothing. Duration must be *shorter* than the run being
  measured.
- **A round-trip mutation caught by the re-parse is not a `DeepEqual` test.**
  Dropping `Gates` from the `conditionFile` literal made the written skill
  *illegal*, so the test died at re-parse and any assertion would have passed. A
  second case where the field is legal either way (gate **+** `bonus_power`) is
  what makes `reflect.DeepEqual` the thing that catches it.

⚠️ **Adding one skill to `internal/testfixture.Skills` moves 64 golden lines.**
`testfixture.Inject` appends the bench onto the shipped book, and both TUI screen
goldens print that combined count. So "add a `TestTheBenchCoversTheMechanics`
row" and "no golden may move" are **in conflict** — every moved line is just the
count (e.g. 155→156), no prose. Prove that mechanically before accepting.

Related: [[hexarena-golden-workflow]] (`make golden`, never `go test ./... -update`).
