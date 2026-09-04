---
name: hexarena-descriptions-are-derived
description: "hexarena — skill/trait/status descriptions are generated from the data; only a flavour clause is authored, and a trait's ban is stricter; shares never truncate"
metadata: 
  node_type: memory
  type: feedback
  modified: 2026-08-27T06:45:00.000Z
---

The user asked for LoL-style skill blurbs (2026-08-26, PR #59) and chose **derived from data** over authored text, plus **percent-of-stat numbers** over damage figures.

**Why:** an authored line drifts the moment a value moves — "gấp đôi" outlives `bonus_power` falling 1000 → 700 and nothing catches it. The repo already refused that trade in `Archetype.Demands` (computed from the kit so it cannot describe a kit it no longer has). A share ("100% công") is true wherever it is read; a damage figure is true for one caster against one target and false at the next buff. Also: authored text would be Vietnamese-only, the way `Skill.Name` already is, so English would get nothing.

**How to apply:**
- Code lives in **`internal/i18n/describe.go`** — `Lang.Describe`, `Lang.DescribePassive`, in **both languages** (moved out of `internal/tui`, which reads `[]Event` and had no business holding it). `internal/tui` keeps only the block framing (`Detail`, `DetailPassives`), which take a `Lang`. ⚠️ **Never add a `description` field to `skills.json`.** A new skill field that changes what a skill does means extending the generator, not writing prose.
- Two entrances: `?N` / `?TAG` at the battle prompt (a **numbered prompt, not a cursor UI** — that is why it is `?N`), and `?` on the hexforge-tui skill listing → `screenBlurb`. ⚠️ **Not under the forge form**: 19 fields already show 13 in an 80x24 window, so a 3-line block costs a quarter of them; a screen costs nothing until asked for.
- ⚠️ In **English a bare id IS the name** (`poison`), so the no-id-leak assertion is Vietnamese-only. English needs singular wordings Vietnamese does not (`BlurbCostCooldownOne`, `BlurbStripsOne`) — two keys, never a plural rule.
- ⚠️ A **gated** trait says "Mang"/"Carries", not "Luôn mang"/"Always carries" — "always" followed by "only while under a third" contradicts itself.
- Borrow `i18n.Vi.Gloss` for every data name; never grow a second Vietnamese vocabulary. Status **categories** got glosses in this PR for the same reason.
- `internal/i18n/testdata/describe.golden` covers **every** shipped skill and trait in **both languages** (and `internal/i18n` had to be added to `make golden` by hand): a balance change moves a line there, and that diff is how the change reads to a player. `skills.golden` = what a skill is worth; `describe.golden` = what it sounds like.
- `TestEverySkillDescriptionSaysWhatItDoes` holds what a golden cannot: every skill must state an aim, a cost line with a range or `bản thân`, and leak no status id into the prose (a leak = missing gloss).

⚠️ **The battle prompt still asks for Vietnamese on an otherwise English screen** — the user's explicit choice, recorded as a stated cost. It is now one word (`i18n.Vi` in `cmd/hexarena/main.go`) rather than a rewrite. Do not "fix" the inconsistency piecemeal: translate the whole battle screen in one piece or leave it.

**Three describers, one house** (`internal/i18n/describe.go`): `Describe` (skill — "should I use this"), `DescribePassive` (trait — "what is this unit carrying"), `DescribeStatus` (status — "what has just happened to me", PR #75).

**The one authored thing is `flavour`, and it exists on BOTH `skills.json` and `passives.json`** (PR #72, #77). Safe only because `ParseBook` refuses a **digit**, and a seed test refuses a **spelled numeral** (`hai`…`mười`; `một` exempt — it is the article far more often than a count).
- ⚠️ **A trait's body-word ban is UNCONDITIONAL and permanently so.** A skill free for anybody may not name a body; a **restricted** one may (`ingrain` says rễ, only `plant` takes it). **`passive.Passive` has no restriction mechanism at all** — no element/archetype/species/character — so nothing can ever guarantee the body. Not a rule awaiting a future field: the field would have to be built first. `TestATraitFlavourNamesNoBody`.
- ⚠️ On a **skill** the clause **replaces** the derived opening; on a **trait** it is a **lead line**. A trait has 1–6 lines and no opening among them, so a replacement would replace whichever sorted first.
- The clause is also what fixes the bare `nó` opening 6 of 11 trait wordings — it gives the pronoun an antecedent, so **no wording had to change**. (The pronoun is deliberate: a data name is authored lowercase and cannot head a sentence.)

**Trait sentence order is NARRATIVE, not field order** (PR #77): what the holder **is** (grants, resists) → what its **own attacks** do (applies, amplifies, drains) → what attacking **it** costs (replies) → **when** (while). Field order read backwards on `venom_blood`, which replied to an attacker before saying it was immune.

⚠️ **TWO share renderers, on purpose** (PR #91). `forge.Percent` keeps a tenth (`2.5%`) for **hexforge's tables** — an author is tuning the number and the tenth is what is being tuned. `i18n.share` **rounds to a whole percent, half away from zero** for **sentences** — a tenth is precision a player cannot act on. History: share *truncated* first (2.5→**2**, a fifth of the value gone; skills are priced in hundreds of ‰ and lost nothing, traits in tens), was replaced by the exact figure, and the tenth is now rounded away again. 25→**3**, not 2 — rounding is not the old truncation.

⚠️ **USER RULE: no description ever tunes a figure by less than 1%.** That rule — not the renderer — is what makes rounding safe: data never lands where it would print `0%`. `TestNoShippedShareIsUnderOnePercent` in `internal/seed` holds it over every shipped **skill, trait and status** (fixtures in unit tests are exempt — it is a rule about authored content). Floor is **1%**, not the 5‰ rounding needs: a value in that gap is a zero too many. Never re-add a decimal to a sentence to survive data the rule forbids.

**Status reference** (PR #75): `?mire` / `?*` at the prompt, `hexforge statuses`, `screenStatuses` in the tui. Print a **life** not a tick (poison 150/450 vs burn 160/320 — the tick alone ranks them backwards), **one stack's life and the cap's, NOT the ramp** `skills.golden` prints (that needs a skill off cooldown every turn; no kit has one). Permanent = "always"; a 1-stack status gets no "per stack". Grouping is `status.Book.Grouped` **in core**, so three front-ends cannot disagree. The caveat is printed **once**, as the **last** line — and `frame` cuts from the bottom, so there is a test at 80x24.

**Where each describer is read** (#80, #81 closed the last gaps): skill → `?N` at the prompt, `?` on the skill listing → `screenBlurb`. Trait → `?TAG` at the prompt, `?` on the **cast browser** → the same `screenBlurb` branching on `blurb.from`, **and the `nội tại` menu** (`screenPassives`) which lists *every declared* trait. Status → `?mire`/`?*`, `hexforge statuses`, `screenStatuses`. Plus `hexforge passives` (9 columns; the reply is **one** cell on purpose).

⚠️ **The browser's `?` and the `nội tại` menu answer DIFFERENT questions** — don't collapse them. `?` is "what is this character carrying", filtered by level, so a trait nobody has learned is reachable from nowhere; the menu is "what traits are there". The menu's column is **who carries it, not who may**: a trait has **no restriction mechanism at all**, so "may" is everybody. `Library.TraitCarriers` walks the **cast**, not the trait book — the edge lives on the character's learnset.

⚠️ **Adding a screen — or a distinct *state* of one — to hexforge-tui means adding it to `everyScreen` in `language_test.go`**, or every width and translation test skips it in silence. That test renders at width **200** and measures against **79**, and excuses only authored free text — so a derived sentence must **wrap to `minWidth`, never to `m.usableWidth()`**. `screenPreview` is still not in `everyScreen`.

Related: [[hexarena-archetype-must-be-glossed]], [[hexarena-shipping-a-character]], [[hexarena-mechanics-log]].
