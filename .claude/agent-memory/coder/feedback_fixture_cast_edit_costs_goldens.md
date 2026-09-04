---
name: fixture-cast-edit-costs-goldens
description: hexarena — putting one skill into a testfixture character's kit moved 656 golden lines across two clients; the repo's idiom is a carrier the TEST builds (twinOf / forkedTwin / bringsTheGradient), not a fixture-cast edit
metadata:
  type: feedback
---

When a test needs a character the fixture cast does not have, **build it in the
test**; do not edit `internal/testfixture`'s `Characters`.

**Why:** measured while seating `self_gradient` as a `WeighField`. `desperate`
(the bench's only gradient) is in neither fixture character's kit, and a kit is
the **first `cast.SkillSlots` learnset entries** (`forge.seedKit`) — so a fifth,
appended entry is never fielded and the only way to field it is to displace one
of the four. Adding it to `fixture-anime.adept` displaced `purify`, and `make
golden` then moved **656 lines** across `cmd/hexarena-tui/testdata/screens.golden`
and `cmd/hexforge-tui/testdata/screens.golden` — the adept fights in those
goldens, so its kit change rewrites whole battle screens. The goldens are the
design record; none of those 656 lines would have been a design decision, and a
cast change also silently moves what each golden can see ([[two-screen-goldens]]).

The repo already says this in its own words: `forkedTwin`'s doc — *"It is the
refusal a test can ask for. Everything a battle refuses a row for … needs a
carrier the fixture cast does not have"* — and `twinOf` beside it, both in
`internal/forge/carriers_test.go`, both saving a modified copy of a fixture
character into the scratch data dir through `lib.SaveCharacter`.
`bringsTheGradient` in `weigh_test.go` is the third.

**How to apply:** before touching the fixture cast, make the edit, run `make
golden`, read `git diff --stat`, then revert — the line count decides. A
test-built carrier costs one helper and moves nothing. Give the helper a
self-check that the carrier actually *fields* what it was built for (saved is not
fielded), or it silently becomes a carrier that knows the skill and never brings
it. Related: [[fixture-decides-what-is-visible]], [[fixture-hidden-branch]].
