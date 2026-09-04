---
name: hexarena-status-naming
description: "hexarena status naming rule — debuffs are verbs, buffs are nouns/participles; rally→fury (PR#83); a status never says who receives it, the skill does"
metadata: 
  node_type: memory
  type: project
  modified: 2026-08-27T07:22:58.742Z
---

hexarena PR #83 (merged 2026-08-27, `69da20c`) renamed the attack buff **`rally` → `fury`**, gloss **"sung sức" → "cuồng nộ"**.

⚠️ **Naming rule the set already followed, now written down: every debuff is a VERB (`weaken` `expose` `blind` `mire` `stun`), every buff is a NOUN or PARTICIPLE (`haste` `focus` `veil` `block` `toughened` `kindled`).** `rally` was the one buff in verb form — written in the debuffs' voice. Check this before adding a status.

Mirror pairs by axis: attack `weaken`↔`fury`(+`kindled` permanent) · defense `expose`↔`toughened` · accuracy `blind`↔`focus` · speed `mire`↔`haste` · dodge —↔`veil`.

⚠️ Rejected `empowered`: it means "stronger" generically, and **attack is the one axis with two buffs** (`kindled` = permanent trait ember, `fury` = timed 3-stack), so a generic label is weakest exactly where distinction matters. Rejected `roused`: "rouse the troops" keeps the group sense that was the reason for renaming.

⚠️ **A status never says who receives it — the SKILL does.** `applies` → whatever the skill hit; `self_applies` → the caster. `dragon_dance` (target self, dragon-only, cd 4) carries only `self_applies`, which is why an attack buff read as team-wide. **A team buff is a skill, not a status** — fixture `war_cry` (`target: ally`, `pattern: column`, power 0) is the shape, and the shipped book has **0 ally-targeting skills** (22 enemy, 6 self). The word `rally` is now free for that skill.

⚠️ **`replay.golden`/`scenarios.golden` did NOT move** — `battle.Suggest` never buffs, so `dragon_dance` has never been played in an autopilot battle. Only the two reference listings (`seed/skills.golden`, `i18n/describe.golden`) shifted.

⚠️ **`statusGloss` is a LENIENT table** (a miss shows the bare id). Renaming a status while leaving the gloss under the old key = full table, green tests, English word in a Vietnamese sentence. **3 statuses had already shipped bare that way.** `TestEveryShippedStatusIsGlossed` now measures the shipped book, not the table's length (same shape as `TestEveryShippedArchetypeIsGlossed`).

⚠️ `grep rally` has false positives: "st**rally**"/"gene**rally**" — never blind-sed a status id.

Numbers worth keeping (measured here): `expose` −30%/stack **saturates**, so 2 stacks = −36% def not −60%, and it is worth MORE vs armoured targets (+25.9% dmg vs Charizard 400 def, +32.4% vs Blastoise 640) because `defense_constant` is only 300.

Related: [[hexarena-tui-i18n]], [[hexarena-core-design]], [[hexarena-mechanics-log]]
