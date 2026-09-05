---
name: normalised-upstream-cannot-discriminate
description: hexarena — a value normalised by its parser, or pinned by a constant, carries no variation downstream, so a guard that counts it counts everything: cast.Build.Stage is always set (28/28) and every shipped form is reached at LevelCap
metadata:
  type: feedback
---

Before writing a guard that counts a discriminating property, ask whether the
value still *varies* by the time you read it. Two measured instances, both from
the internal/draft state machine (step 2a):

- **`cast.Build.Stage` is the RESOLVED stage, not the authored one.**
  `builds.json` omits `stage` on a line that does not fork, and
  `cast.resolveBuild` fills it in with `stage.Name` before you ever see a
  `cast.Build`. So `if build.Stage != ""` counts **28 of 28** shipped builds and
  a vacuity guard written on it (`if forked == 0 { t.Fatal(...) }`) can never
  fire. What still varies is the *character's line*: ask
  `character.FurthestAt(progression.LevelCap)` and count `len(arms) > 1` → **3**
  (all three `pokemon.poliwag` builds).
- **At `progression.LevelCap` (60) every shipped form is reached.** The deepest
  `min_level` in `cast.json` is **48** (`pokemon.gible`), so for anything fielded
  at the cap — and every drafted unit is — `progression.Line.Resolve`'s *"stage
  %q begins at level %d, and this is level %d"* arm is **unreachable**. A test or
  a mutation aimed at "accepts a form the level does not reach" measures nothing
  there. The two arms that *are* reachable at the cap: the **unnamed fork**
  (`StageAt` refuses to choose) and the **unknown name** (a typo).

**Why:** both guards look sharp and both are green whatever the code does, which
is the failure mode this repository already has a list of — a fixture or a
constant that puts the interesting branch out of reach.
See [[fixture-hidden-branch]] and
[[a-well-formed-measurement-can-measure-nothing]].

**How to apply:** when a guard counts "how many of these are X", print the count
and check it against the total before trusting it — 28 of 28 is the tell. When a
level is pinned by a constant, work out which refusals that constant saturates
before choosing what to test or what to mutate.
