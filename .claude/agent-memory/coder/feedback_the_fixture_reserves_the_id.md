---
name: the-fixture-reserves-the-id
description: internal/testfixture APPENDS 5 archetypes to the shipped book, so bulwark/vanguard/sentinel/duelist/skirmisher are reserved names — and all five are already in archetypeGloss, which reads as "that step is done"
metadata:
  type: feedback
---

Before naming a new **shipped** id in `internal/seed/data`, check whether
`internal/testfixture` already declares one — it **appends** its books to the
shipped ones in every scratch data directory, so the two id spaces are one.

**Why:** `internal/testfixture/fixture.go` declares five archetype presets
(`bulwark`, `vanguard`, `sentinel`, `duelist`, `skirmisher`) and injects them with
`appendTo(dir/archetypes.json, "archetypes", Archetypes)`. Shipping a preset named
`bulwark` (2026-09-08, the onix line) made `cast.ParseArchetypes` refuse every
scratch directory with **`archetype "bulwark" is declared twice`** — about forty
tests red across `cmd/hexarena-tui`, `cmd/hexforge-tui` and `internal/screen`,
none of them in the package or the file that was edited, all of them failing in
`buildScratchData` before a single assertion ran. The five names were chosen
precisely because they were *not* shipped, so they are reserved rather than
available.

**How to apply:**

- Grep the fixture before naming anything: `grep -n '"id"' internal/testfixture/fixture.go`.
  It carries skills, statuses, archetypes, origins and characters, so the same
  trap exists on every one of those axes.
- ⚠️ **A gloss already being present is a FALSE all-clear.** All five fixture
  presets sit in `archetypeGloss` (`internal/i18n/gloss.go`) so that
  fixture-driven screens read in Vietnamese. A plan that says "add the id to
  `archetypeGloss`" and a grep that finds it already there means **the id belongs
  to the fixture**, not that the step is finished. That is what made the collision
  survive planning.
- When the collision bites, rename the **shipped** side. The fixture's names reach
  three golden files and a dozen test bodies; a brand-new shipped id reaches its
  own data plus one design-table row and one gloss line.
- The failure names a file you did not touch, so read the *message* rather than
  the location: `reload before fixture-anime: archetype "X" is declared twice`
  points at the injection, not at the client.

Related: [[fixture-decides-what-is-visible]], [[fixture-cast-edit-costs-goldens]],
[[a-green-expected-mutation-proves-nothing-by-itself]].
