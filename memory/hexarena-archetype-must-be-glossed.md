---
name: hexarena-archetype-must-be-glossed
description: hexarena — every shipped archetype MUST have a Vietnamese gloss in i18n/gloss.go; the character browser shows it bare otherwise
metadata: 
  node_type: memory
  type: feedback
  modified: 2026-08-26T12:10:47.293Z
---

User instruction 2026-08-26: on the character-list screen, the **lối chơi** (archetype) field must always carry the Vietnamese name in brackets — never a bare English id. Applies to every new lối chơi, not just the ones that existed.

**Why:** `cmd/hexforge-tui/browse.go` renders it with `m.lang.Glossed(character.Archetype)`, which falls back to the bare id when `archetypeGloss` in `internal/i18n/gloss.go` has no entry. Every other field on that row is a figure or an element and reads identically in both languages, so the preset is the only *word* on the screen — a miss leaves one English word alone in a Vietnamese screen. `scorcher` and `warden` both shipped that way and nothing complained; see [[hexarena-shipping-a-character]] and [[hexarena-tui-i18n]].

**How to apply:** adding an archetype to `archetypes.json` is not done until `archetypeGloss` has its entry. `TestEveryShippedArchetypeIsGlossed` (in `internal/i18n/gloss_test.go`, added by PR #45) now enforces it against the shipped book — an unshipped preset may still miss, so the table is not a second registry. Statuses/skills stay the lenient kind of table, but fill them anyway: a shipped id with no gloss is the same defect wearing a different name.

Shipped glosses added at the same time: `scorcher` = kẻ thiêu đốt · `warden` = người gác cổng · `regrowth` = tái sinh · `toughened` = cứng đòn · `kindled` = rực lửa.

⚠️ `TestAnIDWithNoGlossIsNormal` uses made-up ids as stand-ins for "not in any data file". It used **`"warden"`** — which became real. Now `"tidebinder"`. Check that list whenever a new id ships.
