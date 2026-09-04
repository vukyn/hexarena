---
name: hexarena-shipping-a-character
description: hexarena — the checklist and the hard gates for adding a character + archetype to internal/seed/data (15 shipped 2026-09-04; counts derived not remembered; sharing an archetype is fine)
metadata: 
  node_type: memory
  type: project
  modified: 2026-09-04T00:00:00.000Z
---

Adding a shipped character to `hexarena/internal/seed/data` touches **5 JSON files + 1 Go test**, and half of it is not discoverable from the data alone. See [[hexarena-core-design]] and [[hexarena-cast-authoring]].

**Cast size is DERIVED, never remembered** (2026-09-04): `jq '.characters|length' internal/seed/data/cast.json` = **15** over 39 stages, `jq '[.characters[]|select(.hidden|not)]|length'` = **14** draftable (naruto is the only `hidden`). Both TODO.md and CLAUDE.md had this number wrong five times running; they now carry the jq instead of a figure. **All 11 elements are carried**, so "one per element" is finished as a guide — a new character is bought for a *way of playing*. Art waiting for an author: 22 files / 8 complete lines (pichu, igglybuff, mareep, dratini, abra, gible, magikarp, onix), derived by `comm -23` over `assets/*.svg` against the `"assets/….svg"` strings in `cast.json`.

**Sharing an archetype is FINE — user, 2026-09-04.** Fifteen characters on fifteen presets is where the counting landed, **not a rule**: `cast.ParseArchetypes` refuses a preset *declared* twice and says nothing about two characters *tuned from* one, and no test in `internal/seed` asks. The user's words: *"sau này tạo nhân vật trùng lối chơi ko sao, đừng quá quan trọng vấn đề này, tuy nhiên sẽ cố gắng build làm sao cho mỗi nhân vật sẽ độc nhất lối chơi"* — so do **not** stall an author on inventing a 16th preset. The aim is a distinct way of playing, and the preset is one lever beside the kit, the affinity, the stat table and the traits; what says two characters play differently is a **measurement**, not the field they were tuned from.

⚠️ **This note was written 2026-08-26 and four of its claims went stale.** Each is corrected in place below and repeated here so a skim catches them: `species` **ships** (it is no longer "planned", and `restrict.species` — not `restrict.characters` — is the lineage axis); archetype cap == top-stage stats is **convention, not a gate**; a new character does **not** need a new archetype; and adding a character moves **`cast`/`species`/`origins`**, never `scenarios.golden` or `replay.golden`. The repo's own `CLAUDE.md`, `README.md` and `TODO.md` carried the golden error too and were corrected the same day.

**Why:** the gates below are enforced by tests that fail *after* the data looks fine, and two of them cost a round trip each time they are rediscovered.

**How to apply — the order that works:**

1. `statuses.json` — only if a new permanent status is needed.
2. `passives.json` — a passive may grant **`permanent: true` statuses only**; a timed one is refused at parse ("it would wear off on the holder's own turns"). `toughened` (def) and `kindled` (atk, added for `blaze`) are the only two. **Reuse before inventing**: `warden` took `endurance` rather than a third name for the same idea.
3. `skills.json` — element must be the character's or `neutral`; a kit demanding >2 non-neutral elements can never be carried.
4. `archetypes.json` — `stats` = the **top stage's** stats by convention; `column` 0..2; passives are `[{id, at_level}]`. ⚠️ **No test enforces that equality** (grep'd 2026-08-31) — and a new character is **not** required to bring a new preset: several characters may share one, the only uniqueness rule is inside `archetypes.json`. Adding a preset *does* force a row in the hardcoded design table, step 6.
5. `cast.json` — id `origin.name`, one dot; **`image` REQUIRED at character level**, optional per stage (absent = character's own); level-1 base ≈ **⅓ of cap** (a test wants an early battle legible). A missing SVG is **not** a parse error — `internal/core` may not touch the filesystem — it is caught by `TestInspectPassesOnTheShippedData` / `hexforge check`. `bio` may carry **no digit** and may **not name a later form**. `species` is optional but every shipped character declares one; ids must exist in `species.json`, no duplicates, no cap on count.
6. **`internal/seed/cast_test.go` — `TestShippedArchetypesMatchTheReferenceProfiles` hardcodes every preset's cap stats and column.** It does not read the JSON. A new preset that is not added there fails with "the shipped data declares N presets, the design table lists N-1". This is the single most surprising step.
7. `make golden` (never `go test ./... -update`), then `go test ./...`.

⚠️ **Which goldens actually move** (each generator is handed exactly what it reads):
a **character** → `cast.golden` · `species.golden` · `origins.golden`; a **preset** →
`archetypes.golden`; a **skill/status** → `skills.golden` + `describe.golden`.
`scenarios.golden` reads rules/chart/bounds/ceilings/patterns **plus the status book**
(`writeStatusScenario` calls `seed.StatusBook`) — no skill book, no cast.
`progressionReport` takes limits + rules alone. `replay.golden` renders **`roster.json`**,
so a cast-only character never reaches it. Goldens key rows by **id**, so changing a
Vietnamese `name` moves nothing.

**The budget gates (all measured at the cap, worst case):**
- ceilings hp 4800 / atk 800 / def 800 / spd 200 / acc 300 / ddg 150
- `EffectiveHP = hp*1000/DefenseReduction(def)`, `DefenseReduction = 300*1000/(300+def)`, must be **≤ 11500**. Shipped: scorcher 7242 · blighter 10410 · warden 11285 (215 to spare — deliberate, the wall preset should sit against the bound the budget exists to draw).

**`TestADetonateIsWorthLessThanItsBreakEven`:** a skill with `requires.consume: true` must burst for **more** than (forgone ticks + one plain attack) and **at most 2×** it. `inferno` at power 900 + bonus 1100 came out 0.76× and was refused; 1000 + 2500 lands ~1.3×. A non-consuming `requires` (venoshock, brine) skips the test entirely.

⚠️ **Rebase before merging a data PR.** #40 and #39 edited `archetypes.json`/`cast.json` on different lines, so git merged both clean and the result did not decode — `passives` had become `[{id, at_level}]` under it. Broke `main` on 14 tests; fixed by #41. Textual non-conflict is not schema agreement.

⚠️ **`TestTheSkillListNamesSkillsInVietnamese` broke on correct data.** The TUI skill list is a window around the cursor and the test named two *fixture* skills that sort after every shipped one; nine new skills pushed `sever` off screen. Fixed in #42 by rendering top **and** bottom. Expect list/table tests to be length-fragile whenever the shipped book grows.

Art: trace with `python3 img2svg/cli/img2svg.py <png> -o <svg> -q faithful`, delete the raster. `TestTheShippedArtIsCutOutRatherThanFramed` only measures art the **cast references**, so an unused asset is untested until a character points at it.

**A skill named after a body may not be free to carry** (user's rule 2026-08-26, PR #48). The engine checks elements and allowlists and has **no opinion about anatomy**, so `withdraw` glossed "thu mai" (pulling into a shell) would have landed on a shell-less character unrefused. Two rules, not a preference: a **general effect gets a general name** (`withdraw` → "thủ thế", numbers untouched), while **identity keeps its name and carries a `restrict`** (`ingrain`/`synthesis` → `species: [plant]`; `dragon_rage`/`dragon_dance` → `species: [dragon]`). ⚠️ Those two moved **off** `characters: [pokemon.charmander]` once the species axis shipped — `characters` is now one of five keys (`elements`, `archetypes`, `characters`, `species`, `origins`), and **species is the right axis for a lineage**. `TestABodyBoundSkillIsRestricted` is the hand-written list — add to it whenever a new skill's name names a body part or a lineage.

⚠️ **Do not restrict a lineage skill by archetype.** An archetype says how a unit fights, not what it is; that coupling is what the planned **species** axis exists to remove (roadmap entry in `README.md` → *What a unit is, not only how it fights* + `CLAUDE.md`). It also breaks `TestEitherOrderWorksBetweenTheKitAndTheElement`, because the new-character form has an archetype defaulted while identity is still blank. A **character**-restricted skill cannot sit in a preset's kit at all, so `scorcher` suggests 7 while Charmander carries 9 — that is the engine agreeing, not a bug.

`skills.golden` now has a **who may carry** section listing only restricted skills.

## Two more gates the original note missed

**`skillGloss` is a frozen 19-id fallback** (`internal/i18n/gloss.go`) from before skills carried a `name`. A shipped skill id appearing in it fails `TestTheLogGlossesNameEveryShippedID` — `flurry` is one of the nineteen, which is why #182's three-strike skill shipped as `pummel`. Do not add to that table; do not reuse its ids.

**`TestNoGlossedLogRowOutgrowsTheWindow` has zero margin** (79 of 79 cells, bound recomputed from the books every run). A long Vietnamese skill/status name turns it red. Longest shipped name is "phong ma thủ lý kiếm" (20).

**A new archetype needs a gloss**: `archetypeGloss` in `internal/i18n/gloss.go` — `TestEveryShippedArchetypeIsGlossed`. A **species** needs none: `Lang.SpeciesName` reads the authored `name` field. Never touch `keys.go`/`vietnamese.go`/`english.go` for a data id — those hold the program's own wording.

**Word bans, exactly:** `bodyWords` matches **substring** but only on skills with **no `restrict`** (so anything carrying `restrict.origins` is exempt). `countWords` and `volleyWords` match **whole words** on every skill (volley only when `strikes <= 1`).

**Skill/trait names are all lower case** (user, 2026-08-31): a name is printed inside a Vietnamese sentence, so a title-cased one reads as a proper noun mid-clause.
