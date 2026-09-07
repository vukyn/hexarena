---
name: hexarena-a-fixture-chosen-by-property-is-blind-to-other-properties
description: hexarena — battleCast picks its cast by trait and skill COUNT, which happened to yield an all-single kit, so a whole feature about area shapes was invisible to every screen test and golden on the day it landed
metadata:
  type: feedback
---

`battleCast` was rewritten (#328) to pick the battle fixture's cast **by property
rather than by position** — most traits, then most skills, then by id — which is
the right instinct and fixed a real staleness. It is still blind, and the
blindness is structural: the property it sorts on is *how many* skills a
character has, and nothing about *what shape* those skills cover.

The cast it lands on brings four `single` skills. So when the aim list gained
rows naming the cells an area skill also catches, **every existing test and every
golden drove a screen on which those rows are correctly absent** — the whole
suite passed before the feature was written and would have gone on passing with
it deleted. Twenty-one area skills ship in `builds.json`; this is a state a real
battle reaches on most turns of a real kit.

**How to apply.** "Chosen by property" is only as good as the property. Before
trusting a shared fixture with a new feature, ask what property the feature needs
and whether the fixture selects on it — and if it does not, give the feature its
own fixture that selects on the right one, rather than widening the shared cast
and moving every golden in the package.

Two things make that fixture worth writing rather than skipping:

- Pick by the property, never by name. A named character goes stale the day its
  learnset is rebalanced, and it goes stale **silently** — a kit of `single`
  skills draws a correct screen with nothing on it and no line to notice missing.
- Give the state its own golden entry. Without one the feature has a behaviour
  test and no picture, which is the same gap as
  [[hexarena-a-golden-cannot-hold-a-keystroke-rule]] seen from the other side:
  there a picture could not hold a rule, here there was no picture at all.

⚠️ Fail loudly when the search finds nothing. Every fixture here ends in
`t.Fatal` when no shipped character carries the shape, or no aim this turn
catches a second cell — otherwise the day the data changes, the test goes green
by measuring nothing, which is exactly the failure it was written to prevent.
