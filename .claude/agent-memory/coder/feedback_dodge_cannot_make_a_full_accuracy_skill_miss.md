---
name: dodge-cannot-make-a-full-accuracy-skill-miss
description: combat.Rules.Chance returns scale.Base outright at accuracy >= 1000 BEFORE it looks at dodge, so no dodge stat makes an accuracy-1000 skill miss — and every multi-strike skill in the battle fixture book is accuracy 1000
metadata:
  type: feedback
---

`combat.Rules.Chance`'s first line is `if h.SkillAccuracy >= scale.Base { return
scale.Base }`. Dodge is read **after** that. So a fixture that raises the
target's `progression.Dodge` to force a miss is a fixture that measures nothing
when the skill declares accuracy 1000 — and it fails silently, as a volley where
every strike lands.

**Why it costs a session:** the shared `books(t)` fixture in
`internal/core/battle/battle_test.go` has **no** multi-strike skill that can
miss. Every one of `triple`, `gnaw`, `maul`, `volley`, `patter`, `flurry` is
accuracy 1000; the only sub-1000 entries (`flicker` 400, `feint` 1) strike once.
So "a miss standing between two blocks" is unreachable from the shipped fixture
by any stat, and the dodge route looks plausible right up to the point it
returns four landed strikes.

**How to apply:** author the low-accuracy multi-strike skill **in your own test
file**, never in `books(t)` — several fixtures in that package pick kits by
*property* (strike count, area, pattern) rather than by name, so a new entry in
the shared book quietly changes what they measure. Build it with
`skill.ParseBook` for the small JSON and hand the result to
`base.Skills.Append(deps, extra.Skills()...)`:

- ⚠️ `json.Unmarshal` straight into a `skill.Skill` **fails** — `element.Element`
  has no string decoder; `ParseBook` is the only path that resolves the names.
- `skill.Deps{Patterns: base.Patterns, Statuses: base.Statuses}` comes off the
  `battle.Books` you already have.

Then sweep seeds and **pin the outcome shape** (`"BMMB"`) as a premise before
reading any figure, so a seed that stops producing the arrangement reddens
instead of drifting. See [[a-fixture-cast-edit-costs-goldens]],
[[fixture-hidden-branch]].
