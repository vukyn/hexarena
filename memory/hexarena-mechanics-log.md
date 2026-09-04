---
name: hexarena-mechanics-log
description: "hexarena — the merged per-PR mechanics log: every shipped battle/data change from PR#43 to PR#109 with the gotcha each one cost. Search here before touching combat, traits, summons, learnsets or a build."
metadata: 
  node_type: memory
  type: project
  modified: 2026-08-28T12:05:43.681Z
---

Several memories merged into one topic file to keep the index readable. Each section below is the original entry, unedited.

## hexarena-draw-and-sidenone

hexarena PR #43 (merged 2026-08-26) closed roadmap item "a battle nobody can act in has no outcome".

**`battle.Outcome`** on the closing `Ended` event: `undecided|victory|annihilation|stalemate`, serialised by name. `Battle.Outcome()` for callers — `Winner()` cannot tell the two draws apart.

**`frozen()`** = pure predicate, NOT a turn counter: every living unit has nothing timed on it (`status.Set.Timed()`, ignores permanent trait statuses) AND no skill with a legal aim, **cooldowns ignored**. Cooldowns/control deliberately unread — that is what stops an ordinary skipped turn reading as a deadlock. Poisoned deadlock ≠ deadlock.

⚠️ **`checkEnd` vs `settle` must stay separate.** `checkEnd` = "is a side empty", called from `kill()` mid-skill. `settle` = checkEnd + deadlock test, only where a turn finished (end of `Act`, end of `Pass`, Advance's 2 skipped-turn returns). Asking the deadlock question from `kill` reads a board nobody will act from.

⚠️ **A self-target skill always has an aim** → a unit that can buff itself is never frozen (correct — it can act). Two of those = genuine runaway, which is what the turn limit is for. `RunToEnd` still returns nil at the limit.

Authoring half: `battle.New` refuses a unit that can aim at nobody from its slot; `hexforge check` **warns** (`forge.Warning`, not `forge.Problem` — check still passes) when a character's longest range < `hex.ReachNeeded(archetype.Column)` (1/2/3 front/middle/back, measured through `hex.Place`).

⚠️ **BREAKING: `hex.SideNone` is now the zero value** (`SideNone, SideAlly, SideEnemy`). Side is `json:"side,omitempty"`, so with SideAlly at 0 every ally event wrote NO side and readers recovered it from a missing field — identical bytes to "no winner". **Logs written before #43 fail `--verify`.** Never put a real value back at zero. `Side.Fights()` is the guard; `battle.New` refuses an entry on no side.

Unfixed wart: every event writes `"cell":{"col":0,"row":0}` — `omitempty` has no effect on nested structs, and `omitzero` is wrong because `{0,0}` is a real cell. Would need an off-board zero for `Offset`.

Related: [[hexarena-core-design]], [[hexarena-shipping-a-character]], [[commits-always-via-pr]]

## hexarena-roster-instrument

hexarena PR #53 (merged 2026-08-26) closed the roadmap item "a real cast — and an asymmetric roster".

`roster.json` = 3v3 by character reference, **no unit on both sides**: ally Venusaur 60 / Wartortle 16 / Charmander 8 vs enemy Blastoise 60 / Charmeleon 28 / Ivysaur 16. Formation `2,1` / `1,0` / `1,1` both sides.

**Measured 4000 seeds: ally 48.5%.** Removing `razor_leaf`'s pierce 400 → 46.5%. Old mirror moved 23/40→25 = noise, which is why pierce was unjudgeable. Answer: pierce 400 is a modest dial.

⚠️ **The 40-seed `TestSeedBattlesFinishFromEverySeed` is a smoke test, not a measurement** — it read 45% on a draft whose true rate was 55%. Tune against ≥600 seeds; a throwaway `tools/sens` main.go over `seed.NewBattle` is the way (delete it before committing).

⚠️ **Slot `1,2` is 4 cells from enemy `1,2`** (measured via `hex.Place`) — past every range in the cast (max 3, hydro_pump 4). A draft using it stalled **5/4000 seeds**: 2 survivors unreachable, and NOT a draw because one kept refreshing a self regen → `frozen()` correctly never fired → ran the turn limit out. **This is the hole in the stalemate predicate**: a self-refreshing regen/buff holds a frozen board open forever. Not fixed.

⚠️ **Element triangle is NOT closed**: water>fire (1500), fire>grass (1500), **grass vs water = neutral (1000)**. Grass has no elemental answer; it pays in neutral-element skills (`sludge_bomb`, `venoshock`). And **squirtle cannot carry an ace slot** — strongest element but lowest attack/speed curves in the cast, loses the ace duel to Venusaur on stats.

New tests (both mutation-tested): `TestTheShippedRosterIsNotAMirror` (compares **resolved** units — name + stat line, not the authoring form), `TestEveryShippedUnitCanReachEveryEnemy` (stricter than `battle.New`'s "can reach somebody", which stays right for a game). `TestShippedRosterIsUsable` now needs ≥2 affinities.

`scenarios.golden` does NOT move with the roster — it runs on the testfixture bench. Only `replay.golden` does.

Related: [[hexarena-mechanics-log]], [[hexarena-shipping-a-character]], [[hexarena-core-design]]

## hexarena-gated-grant

hexarena PR #62 (merged 2026-08-27, `0f7ba1b`) built the gated grant — a trait that is a stat change *and* a condition. `blaze` = `{"grants":[{"status":"kindled"}],"while":{"below_health":333}}`.

**A gate covers the WHOLE trait** (grants + resists + applies together). Burn immunity therefore split out of `blaze` into a new `heatproof`. **A trait wanting one gated half is two traits.**

Mechanism: `status.Set.Hold`/`Release` (engine-only door; `Remove` still refuses permanents so no cleanse dispels a trait; each half refuses the other's kind) · new event kind `PassiveReleased`, `PassiveHeld` now also fires mid-battle · `battle.reconsider`.

⚠️ **THE TRAP: health does not move in `wound` and `heal` only — there are THREE sites.** The strike loop in `battle/turn.go resolveAgainst` does `target.HP -= attempt.Damage` directly and never calls `wound`; that is where nearly all damage is dealt. README/CLAUDE.md both claimed two. Read the gate **per strike**, not per skill. And guard `unit.HP <= 0` as well as `unit.Dead` — the strike loop leaves a target at zero and kills it *after* the skill, so a flag-only guard fires on a corpse.

⚠️ **A gated trait is nearly a ONE-WAY DOOR on autopilot** — `Suggest` never heals. Only healing in 60 bench battles is a drain to its own caster (~1/40 bar vs damage ~1/10). Swept 4000 battle-seeds × every arrangement (bigger drain, smaller holder, gates 200–950): came back off **once** (gate 200). So `passive_released` is proved by a **hand-played** battle inside `TestEveryEventKindIsReachable`, not by widening the sweep.

Balance: ally 48.5% → **49.2%** over 4000 seeds; pierce dial unchanged (46.5% without). `replay.golden` now shows `passive_held` mid-battle at 25641.

⚠️ **Mutation lesson**: the first retune test passed under its own mutation — a turn's own `retuneAll` sweep covered for it. The retune in `reconsider` is *not* what keeps the queue right; it is what puts `speed_changed` **adjacent** to the trait. Test that adjacency, with a **multi-strike** attacker so other events can intervene.

Also: `tui.DescribePassive` said "Luôn mang X" (always carries) — contradicts the gate line. Dropped "Luôn" when `While != nil`.

Related: [[hexarena-shipping-a-character]], [[hexarena-core-design]], [[mutate-the-producer-not-just-the-logic]]

## hexarena-answering-back

hexarena PR #67 (merged 2026-08-27, `a187b811`) built *Answering back*: `venom_blood` costs whatever bit into it.

`"replies": {"power": 40, "applies": [{"status":"poison","chance":25}]}` on `passive.Passive`. Damage via `combat.Rules.Damage`, statuses via `battle.inflict` — **not a second damage path**. Logged as ordinary `damaged`/`status_applied` with `Passive` set and `Skill` empty; **never both** (guarded).

`battle.inflict` no longer takes a `skill.Skill` but an **`origin{Skill|Passive, Element, Scaling}`** — the three things it wanted from one. A reply is **neutral element, no accuracy roll** (the chart prices what was *thrown*; contact already happened).

`b.answer(actor, bitten, turn)` sits at the end of `Act`, after the target loop. Three of the four rules ARE that placement: once per skill use · every target takes the whole skill first · a reply never triggers a reply (the answer list is built from the skill's targets, so a reply is not in it — **not** a depth counter).

⚠️ **A reply's worth is set by how often its holder is attacked and how long it survives — no number on the trait can express that.** Both Bulbasaurs hold `venom_blood`, but ally Venusaur@60 vs enemy Ivysaur@16 → ally makes 69% of replies, deals **86%** of reply damage. Roster asymmetry, not trait asymmetry. Sweep over 4000 seeds: power40 alone 49.5% · **40+25 shipped 51.9%** · 40+200 73.5% · 250+500 **98.3%**. The roster stops being an instrument long before the trait stops being small.

⚠️ **It flipped the sign of the pierce figure.** Removing `razor_leaf`'s pierce used to cost the ally 2.7pp; now it *gains* 1.1pp (51.9→53.0), at every reply size tried. Piercing helps the attacker; this is the first thing that charges for attacking. **No balance figure measured before #67 carries forward.**

⚠️ **Two reply tests passed for the wrong reason** — found only by mutation: (1) killing the *only* ally makes `Act` return early on `b.finished`, so the reply path was never reached — keep a bystander alive; (2) a skill aimed at an **ally** is never a reply candidate under any rule, so it proves nothing — use a 0-power skill that lands on the holder across the midline.

Descriptions now live in `internal/i18n` (PR #64) and are **bilingual**: ⚠️ `TestTheSameBlanksInEveryLanguage` forbids `%[n]` indexed verbs, so both languages must take blanks in the SAME order — write whole sentences per case, not clauses joined.

Related: [[hexarena-mechanics-log]], [[hexarena-mechanics-log]], [[mutate-the-producer-not-just-the-logic]]

## hexarena-learnset-slots

hexarena PR #70 (merged 2026-08-27, `b51ad08`) built the learnset and the slots — step 3 of the roadmap item. Chosen evolution is still open, deliberately last.

`cast.Character.Skills` is now **`[]cast.Unlock`** — the same type the traits already used. One `{id, at_level}` shape, one validator, one `UnlockedIDs`, one renderer (`forge.UnlockSummary` draws a kit and a trait list alike). `SkillsAt(level)` mirrors `PassivesAt`. `cast.Learn([]string)` wraps a preset's kit at level 1; `cast.LearnedIDs` is "everything it ever knows" (what restriction checks want).

`internal/seed/roster.go`: `SkillSlots = 4`, `TraitSlots = 1`. ⚠️ **`skills`/`passives` on a REFERENCE entry changed meaning** — they used to be a refused restatement, they are now the loadout. Refused: naming nothing, over the cap, twice, or unlearned at that level.

⚠️ **The kit is required, the trait slot is optional** (`required`/`optional` named consts). A unit with no skills cannot act; a unit with no trait is ordinary.

⚠️ **`battle.Log.Roster`** carries the RESOLVED placement, `--verify` rebuilds from it (`battle.New(books, seed, log.Roster)`), NOT `seed.NewBattle`. `Log.Replayable()` false ⇒ old log renders but refuses to verify. `battle.Roster` gained json tags (log must be snake case).

**Measured 4000 seeds: ally 49.5%** (was 51.9), 0 stalls, longest 63, **2.2% idle turns** — not the 30% the design note predicted, because that was a level-1 two-skill unit and the youngest shipped is level 8 with three.

⚠️ **Pierce figure has now flipped sign twice**: 49.2→46.5 (before replies) · 51.9→53.0 (with replies) · **49.5→46.0** (with slots). A dial is worth more to a kit that cannot dilute it. **No balance figure carries across a feature — re-measure every time.**

⚠️ Mutation found an untested rule: **a character must learn something at level 1** (else it cannot be fielded there, and every other level would be legal — the author finds out by fielding it).

⚠️ Rebase lesson: a test asserting "level unlocked it ⇒ the unit carries it" is FALSE once a slot exists. The surviving property is "nothing brought that the level has not unlocked".

Related: [[hexarena-mechanics-log]], [[hexarena-mechanics-log]], [[hexarena-mechanics-log]]

## hexarena-chosen-evolution

hexarena PR #73 (merged 2026-08-27, `7373366`) closed *Learnsets, slots, and choosing to evolve*.

`progression.Line.Resolve(level)` → **`Resolve(level, stage)`**, plus `Line.Allowed(level)` and **`progression.Furthest`** (the empty string, named) for callers that are not choosing — a browse screen describes, it does not field. Roster entry gained optional `"stage"`; absent = furthest, so older rosters are unchanged.

⚠️ **A form ahead of the level is REFUSED, never clamped** (a clamp fields a different unit from the one written down). Two refusals told apart: unknown name = typo; known-but-ahead = not grown into it.

⚠️ **The stage gate on a learnset entry is `Unlock.Stages`, an ALLOWLIST — not `at_stage`, not a threshold.** A threshold only says "from this form on", which a level already says, so everything an early form knew a grown one knew too and giving up an evolution bought nothing. Only a list can say `["Bulbasaur","Ivysaur"]` — a move Venusaur never gets. `at_stage: "Ivysaur"` *is* `stages: ["Ivysaur","Venusaur"]`; **`at_stage` was unblocked and deliberately still not built** (two vocabularies for one idea). Same reasoning as `skill.Restriction`.

Refused: form not in the line · named twice · **every** form (= what naming none means) · a stage list on a preset (no line) · a character that learns nothing its **first form at level 1** can use.

`Unlock.Held(stage)` / `Available(level, stage)`; `UnlockedIDs(entries, level, stage)`; `Character.SkillsAt/PassivesAt(level, stage)`; `Character.form()` resolves Furthest once so both lists ask about the same form. `forge.UnlockSummary` prints `sleep_powder@12[Bulbasaur,Ivysaur]` — the mark shows at **every** level, unlike a level gate, because it never stops being true.

⚠️ **The shipped roster does NOT take the trade** — every unit is fielded as its furthest form, balance unchanged at **49.5%**, `replay.golden` did not move. No character is close enough for one kept skill to pay for a stage of stats: a cast-tuning question, not a mechanism one.

⚠️ Mutation lesson repeated: two mutations **failed to compile** (unused variable) and proved nothing until rewritten with `_ = x`.

⚠️ Test lesson: a layout test that took "the row immediately under the kit" broke because the new mark wrapped the row. Search the rows below a label, never index the next one — a tenth skill breaks it just as surely.

Related: [[hexarena-mechanics-log]], [[mutate-the-producer-not-just-the-logic]]

## hexarena-regen-heals

hexarena PR#79 (2026-08-27, `aad3f0f`) fixed `regrowth` being **inert**: `battle.inflict` computed a tick only for `status.Dot`, so `aqua_ring` and `ingrain` did *nothing whatsoever* when cast (both are power 0 with no `restores`). Note: `synthesis` was never affected — it heals via `restores`.

**Why:** three traps that cost real time and would recur.

**How to apply:**
- A regen tick uses `Rules.Restore`, **not** `Rules.Damage` — no defence curve, no elemental chart. Only the actor's scaling stat and the freeze survive.
- ⚠️ **`wound` emits no event.** So any test asserting "healing resolves before damage" by comparing *event order* is vacuous — a mutation swapping them survives. Assert **survival** (HP after the turn) instead.
- ⚠️ `heal` emits its own event and `wound` does not, so tick healing must be applied **per status entry** (each naming itself) rather than from `Set.Tick`'s total, or every tick logs two `Healed`. `Set.Tick`'s healing return is now deliberately discarded.
- No golden moved: **no roster unit fields either skill**, and `Suggest` never picks a self-cast regen. That pair is why a shipped skill stayed dead with every test passing — reach for a hand-played test on the shipped books, like `aHandPlayedGateCrossing`.
- An amplifier still refuses a regen, now on wording grounds: the share reads "ticks harder" / "mạnh thêm" in both languages. See [[hexarena-descriptions-are-derived]]. ⚠️ The **old** reason ("a multiplication of zero") was written in three places and expired with this fix — the README bullet, the `passive/amplify_test.go` case comment and CLAUDE.md were corrected in PR #80. Check all three if the refusal is ever revisited.

Related: [[hexarena-core-design]], [[hexarena-shipping-a-character]], [[mutate-the-producer-not-just-the-logic]].

## hexarena-spar

hexarena PR #78 (merged 2026-08-27, `4f18acc`) added **`forge.Library.Spar(id, level, seeds)`** — duels a character against **every** character in the book, itself included, and reports a rate per opponent. Two front-ends: `hexforge spar <id> [--level N] [--seeds N]` and **`s` from the TUI's check screen** (not the browser — the browse footer was already 78 of its 79 cells, and check→spar is "legal → belongs").

⚠️ **A spar rate is NOT the roster's win rate. Never compare them.** Roster = 5v5, authored slots, authored loadouts (49.5%). Spar = 1v1, front column, auto four-skill kit, no ally. Bulbasaur reads 50.0% in both and that is a coincidence.

⚠️ **Every pairing is fought BOTH WAYS and that is the measurement, not thoroughness.** `atb.Queue.order` breaks a tie by **enlistment seq**, so the unit placed first acts first all battle — worth **72/100** to Bulbasaur in a mirror. `Matchup.First`/`Second` stay apart so the **control row** (character vs itself) reports what the slot alone was worth: +44.0% Bulbasaur, +23.0% Charmander, +9.0% Squirtle — small where duels run long. Control is **excluded from the headline** (even by construction).

⚠️ **A per-row `Failure` was built then DELETED as unreachable.** Front column → `hex.ReachNeeded`==1, every non-self skill needs range>=1, a self-aimed skill hits an always-occupied cell ⇒ `battle.New` can't refuse one pairing and accept another. Refusal is an error naming the pairing; `TestTheDuelSlotAsksTheLeastOfAKit` pins it. **An ally-only kit does NOT trigger it** — ally targets reach own-side cells incl. the caster's.

⚠️ `Endless` (turn limit) is in **neither half** of a rate's fraction.

Loadout = **first `cast.SkillSlots` skills + first trait the learnset declares**, printed above the figures. A roster refuses to auto-choose; a spar chooses and states it — *a roster IS the conditions, a spar is a measurement*. Gives learnset declaration order a meaning (first = preference).

Moved: **`SkillSlots`/`TraitSlots` → `internal/core/cast`** (were in `internal/seed`); `forge.Library` now reads `modifiers.json`.

**`forge.PercentInColumn`** = `forge.Percent` with the tenth always printed (a column of 50%, 100.0%, 39% can't be read down the page). ⚠️ Vietnamese decimal **comma deliberately NOT used** — every other percentage in the tool prints a point.

**Found:** Squirtle loses to Charmander **30.5/69.5** at the cap, with water on the chart against fire. Cast tuning, not a mechanism bug.

⚠️ 15 mutations; **two tests existed only after a mutation exposed the gap** — nothing pinned *which half was which* (fixed: two characters agree about each other's halves, not just totals) and nothing pinned an **edge's sign** (fixed: hand-built row).

⚠️ **Process trap hit here: `git commit --amend` during a CONFLICTED rebase amends the upstream base commit and silently swallows it.** It ate `88d02ac`; content survived, history did not. Fix was `git reset --soft origin/main` + recommit. Verify with `git merge-base --is-ancestor origin/main HEAD`.

Related: [[hexarena-mechanics-log]], [[hexarena-mechanics-log]], [[mutate-the-producer-not-just-the-logic]], [[commits-always-via-pr]]

## hexarena-ally-targeting

hexarena PR #86 (merged 2026-08-27, `bf25830`) added **`rally`** — the shipped book's **first `target: ally` skill**. Before it: 22 enemy, 6 self, **0 ally**.

```json
{ "id": "rally", "name": "hiệu triệu", "element": "neutral", "target": "ally",
  "range": 1, "pattern": "column", "power": 0, "strikes": 0,
  "accuracy": 1000, "cooldown": 4,
  "applies": [{ "status": "fury", "chance": 1000, "stacks": 1 }] }
```

**Nothing learns it** — deliberate (a learnset entry is a balance decision); `solar_beam` is the precedent for an unlearned shipped skill. So `replay.golden`/`scenarios.golden` did not move.

⚠️ **`accuracy: 1000` = cannot miss and cannot be dodged** — `combat.Rules.Chance` returns `scale.Base` immediately. That is the explicit way to write an effect that must land, and it is why every zero-power support skill in the book uses it: otherwise an ally's *dodge* would make you miss your own buff.

⚠️ **An `ally`-target skill CAN be aimed at the caster's own cell** — `Side.Reaches(from, at)` for Ally is `at == from`, and `aims()` does not exclude the caster's cell (only `Self` is special-cased earlier). So aiming a column at yourself buffs you + the units above/below.

⚠️ **A `column` pattern can NEVER cross the midline** — up/down don't change column, and the midline runs *between* columns. So a test placing the enemy *opposite* the caller cannot tell `target: ally` from `target: all`; it asserts the **geometry**, not the rule. Fix: put the enemy at a formation slot whose `hex.Place` lands **adjacent to the caller and inside a column** (enemy slot `{2,2}` → board `{3,0}`, adjacent to ally `{2,0}`), then assert the aim is **not offered**.

⚠️ **Wording bug this exposed:** `i18n` described every `applies` as `BlurbInflicts` ("Inflicts…" / "Gây … cho mục tiêu"), so the first ally skill said it *inflicts* a buff on a friend. Now `BlurbGives` when `declared.Target != skill.Enemy` — **Enemy is the one side callable hostile without asking anything else**; self/ally plainly aren't, and `all` lands on your own squad too.

⚠️ **`hexforge skills add` has no `--flavour` flag** — author with the tool, then add `flavour` by hand.

⚠️ **Python trap that destroyed skills.json:** `open(P,'w').write(build_from_file())` — Python evaluates `open(...,'w')` FIRST, truncating the file before the builder reads it. **Read the original into a variable before the loop, compute the text, then open for write.** Recovery was `git checkout` the data file + re-author (safe here: the file was destroyed, not mutated).

Related: [[hexarena-status-naming]], [[hexarena-core-design]], [[mutate-the-producer-not-just-the-logic]]

## hexarena-dragon-build

hexarena PR #93 (merged 2026-08-27, `3ee18b9`) gave Charmander a **second late-game direction**: `dragon_claw`@40 (neutral, r2, 1100, cd1, `expose` 40%), `outrage`@44 (neutral, r2, 2800, cd3, **`mire` on itself**), `blood_thirst`@40 (existing), `reckless`@44 (new trait: atk +30%, def −40%, dodge −40%, permanent).

⚠️ **There is NO dragon element.** Dragon is a **species** — that is why `dragon_rage`/`dragon_dance` are `neutral` + `restrict.species:["dragon"]`. The line cannot buy chart advantage; it buys **evenness** (neutral is inert: 1.0× into everything vs fire's 1.5× grass / 0.667× water). **Fire = spiky, dragon = flat.** Measured: spread in turns-to-kill **fire 15, dragon 1**.

⚠️ **A TRAIT CAN NEVER BE SPECIES-FLAVOURED.** `passive.Passive` has **no restriction mechanism at all** (no element/archetype/species/character), so nothing can guarantee what its sentence describes. `TestATraitFlavourNamesNoBody` enforces it on `bodyWords` (mai vỏ cánh chân rễ vảy sừng lông) — but the *principle* is wider: an id/name like "huyết cổ long" is equally wrong. **The dragon identity must live entirely in the skills, which CAN restrict by species.**

⚠️ **A status cannot move max HP.** `Unit.MaxHP()` reads `Base[HP]`; nothing reads the modified value. `status.ParseBook` **already refuses** an `hp` modifier. Not a gap: effective HP is a **joint hp+def budget**, so **defence IS bulk** — Charizard 7242 = 3100 × (300+400)/300; `reckless` takes it to 6096.

⚠️ **`TestADetonateIsWorthLessThanItsBreakEven` prices a burst against the TICKS it consumes**, so a detonate on a non-DOT (`expose`) looks free and the cap collapses to 2× a plain hit. Deeper truth the rule hinted at: consuming `expose` removes the armour break that makes following hits land harder — the opposite of a detonate. So `outrage` is **not** a detonate.

⚠️ **`hexforge spar` CANNOT measure Charmander** — it fields the **first four declared** skills, which for Charmander are its four weakest; the real fire build (flamethrower/inferno/ember/fire_spin) is ~99.7% vs the cast. The earlier 84.7% figure was an artifact. Compare two builds **head to head** instead (`internal/seed/dragon_test.go`).

⚠️ **The 43.8% dragon-vs-fire figure is a FLOOR, not a balance point** — `battle.Suggest` never buffs, so `dragon_dance` is never cast and the dragon build fights with 3 of 4 slots.

⚠️ **Mutation lesson: 4 of 6 mutations were caught ONLY by a golden**, which proves a number moved, never that the design was wrong. Two real holes found: removing `reckless`'s cost (a pure gift) passed everything; giving a dragon skill an element was caught by no claim. Fixed with `TestRecklessIsATradeAndNotAGift` + `TestEveryDragonSkillIsNeutral`.

Next (user's plan): **D = a `skill.Condition` that reads the CASTER** (today it reads the target — so "spend your own fury" is unwritable), then build Squirtle as a counterweight.

Related: [[hexarena-status-naming]], [[hexarena-mechanics-log]], [[hexarena-mechanics-log]], [[hexarena-shipping-a-character]]

## hexarena-self-condition

hexarena PR #95 (merged 2026-08-27, `a8e150c`) added **`skill.Skill.SelfRequires`** — the same `*Condition` as `Requires`, asked of the **caster** instead of the target. Before it, "hits harder while I am furied / cornered" had **no spelling at all**.

```json
{ "id": "outrage", "power": 2200, "self_requires": { "below_health": 400, "bonus_power": 1200 } }
```

**Two fields, not one field with a `whose` flag** — the book already spells this as `applies` / `self_applies`. The *type* is shared: everything in a Condition is the shape of a question, nothing is about who is asked.

⚠️ **Read ONCE PER USE, in `Act` (`Battle.spend`), never in `resolveAgainst`** — that runs once per cell a shape covers, so a consumed condition would charge a column 3× and a single-target skill 1×. `spend` sits **before `applyToSelf`** so a skill that grants and spends the same status cannot pay itself. It also `retuneAll` after a consume (a consumed stat change is a stat change gone).

⚠️ **The bonus joins power BEFORE the splash reduction.** This mutation *survived* the first test: a bonus added **after** still makes every target take more, which is all the obvious test checks — the edge quietly takes a full share. **Assert the RATIO between the aim cell and the edge, not the rise.**

⚠️ **`conditionCaster` is a SECOND builder beside `conditionTarget`, not a parameter on one** — a skill may read one status of its target and another of itself, so one builder would have to be *told* which condition it reads = the exact reading-vs-resolution mismatch `conditionTarget` exists to prevent. `Suggest` reads it once **outside** its per-cell loop.

⚠️ **`resolveCondition` is now ONE validator for both fields**, carrying the field name in every message (two copies = two rule sets, and the looser is the one an author finds). New refusal: a `bonus_power` on a **self-aimed** skill, which deals no damage for it to land on.

⚠️ **`skills.golden`'s amplifier table read only `Requires`** — it would have told an author their skill has no amplifier while the engine amplified it. Gained a **whose** column (target/caster).

⚠️ **`outrage` reads HEALTH, not `fury`, on purpose** — `battle.Suggest` never buffs, so a fury gate would never fire under autopilot and the feature would ship hand-measurable only. A cornered gate fires on its own and pairs with `reckless`'s frailty. Dragon-vs-fire unmoved at **42.5%**.

7 mutations, all caught. ⚠️ One broke the build (unused var) and proved nothing until rewritten with `_ = spent`.

Still absent: **the gradient** (smoothly harder the further the caster fell) — a multiplier in `combat`, not a condition. `SelfRequires` is the threshold version.

Related: [[hexarena-mechanics-log]], [[hexarena-core-design]], [[mutate-the-producer-not-just-the-logic]]

## hexarena-reply-scaling

hexarena PR #97 (merged 2026-08-27, `fe12b50`) gave **`passive.Reply.Scaling`** (a whole `skill.Scaling` — stat *and* base-or-current).

```json
{ "id": "thorns", "replies": { "power": 80, "scaling": { "stat": "defense" } } }
```

⚠️ **Attack was the WRONG DEFAULT, not a missing field.** A trait that answers whoever hit it belongs to a unit **built to be hit** — armoured, not sharp — so pricing every reply off attack made thorns worth least to exactly the character thorns are for. **Blastoise 640 def / 460 atk → the same share off the wrong stat is a third less.**

Reading the **current** value by default is what makes a stat trade work: a trait trading speed for defence buys **thicker armour AND harder thorns with one number**.

⚠️ **Latent bug fixed:** `origin.Scaling` was a bare `progression.Kind` and all 3 readers took the current value — so a skill with `"source":"base"` would have had its **damage read base** and its **DoT tick read current**. Unreachable (no shipped skill declares scaling) and would have arrived the day one did. All 3 now go through `origin.stat` → `combat.PickScaling`.

⚠️ **The word "attack" was hardcoded in THREE places**: the engine, the i18n reply blurb, and **`hexforge passives` (`cmd/hexforge/list.go`)**. The listing is in **no golden** and nothing else read it — the mutation putting the literal back **passed the entire suite**. Now covered by `TestThePassiveListingNamesTheStatAReplyIsPricedOff`.

⚠️ **Two more gaps only a mutation found:** the passive book's **round trip** (a dropped `scaling` survived) and the parse assignment. Grep for hardcoded stat words whenever a stat becomes authorable.

`skill.ParseScaling` is **exported** so a trait and a skill are read by one parser. Health refused everywhere (damage growing as its owner is healed). New refusal: a reply with **no damage** naming a stat.

Shipped `thorns` (8% defence) on Squirtle@32. Duel = a reply's best case (attacked every turn it lives): **14.2% → 15.3%** overall, Charmander 28.5% → **30.7%**. ⚠️ **Squirtle still loses to Bulbasaur 0% either way** — a cast problem, untouched.

**Squirtle tank/taunt build plan** (user's spec, in order): ① reply scaling **DONE** · ② **taunt — NOTHING EXISTS**, needs a status + `aims()` filter, and the rule "taunted unit that cannot reach the taunter" must be decided (touches `battle.New`'s reach check and Stalemate) · ③ the skill/trait set (data-only). ⚠️ **A tank nobody attacks is furniture — `Suggest` picks by expected damage, so it hits the SQUISHIEST, never the tank. Taunt is the precondition for the whole build, not one skill in it.**

Already writable for ③: block for an ally (`target: ally` + `applies: block`), **`skill.Scaling` off defence** (field exists, **no shipped skill uses it**; def 640 / atk 460 → 1.39× per point of power), `stun`, heal/strips. ⚠️ **A `block` stack eats one WHOLE STRIKE** — great vs `outrage` 3400, weak vs 2-strike skills.

Related: [[hexarena-mechanics-log]], [[hexarena-mechanics-log]], [[hexarena-status-naming]]

## hexarena-taunt

hexarena PR #99 (merged 2026-08-27, `df1d57e`) added **`taunting`** — a taunted unit still acts, it just may not pick.

```json
{ "id": "taunt", "target": "self", "power": 0,
  "self_applies": [{ "status": "taunting", "chance": 1000 }] }
```

`Battle.aims` narrows an **enemy-aimed** skill to whoever is taunting. ⚠️ **`Suggest` obeys with ZERO AI change** — it reads the aims it is offered and nothing else.

⚠️ **The status sits on the TAUNTER, not on the taunted.** A taunt on the victim would need to remember *who* taunted it, and a `status.Stack` **deliberately does not remember its applier** (that keeps a stack worth the same after its author dies). On the taunter it needs no memory: "who must I attack" is read off the board, and a corpse is not on it — **no cleanup path at all**.

⚠️ **Range is not read.** Nothing moves on this board, so a range-respecting taunt would be ignored by exactly the long-ranged attackers a tank needs to pull. A range-1 skill hits a taunter 4 cells away — nothing past the *legality* of the aim ever read distance.

⚠️ **A new `status.Taunt` category, NOT a second `Control`.** Stun = "you don't act", taunt = "you act, may not pick" — opposite. Categories exist so a cleanse can name a class; "strips a control" eating a taunt too is unaimable. It also stopped the reference printing *"the holder loses its turn"* under `taunting`. Declared **last** (name-serialised, but `CategoryCount` + grouped listing order are declaration order).

⚠️ **`Category.Harmful()` MUST include it** — that gates what a trait may **resist**, so a taunt outside it makes "cannot be provoked" unwritable. **The mutation moving it passed every other test in the repo.** Pinned by `TestATauntIsSomethingItsHolderWantsGone`.

⚠️ **A taunt is spent on the TAUNTER's own turns** (a status is timed in its holder's turns), so a taunter faster than its victim burns it before the victim acts. **The slowest unit makes the best taunter** — Blastoise at speed 85, exactly. Bit me twice while writing the fixture.

A taunt can only ever **unfreeze** a stalemate (adds an aim for a unit that had none; the taunter is alive by definition), so `frozen()` needs no change — it already asks through `aims`.

7 mutations, all caught. Shipped `taunt` on Squirtle@40, cd 3.

**Squirtle tank build: ① reply scaling DONE (#97) · ② taunt DONE (#99) · ③ the skill/trait set — NEXT.** Already writable for ③: block for an ally (`target: ally` + `applies: block`), **`skill.Scaling` off defence** (field exists, still **no shipped skill uses it**; def 640/atk 460 → 1.39× per point of power), `stun`, heal/strips, a speed→defence trade trait (mirror of `reckless`, and with #97 it buys thorns damage too). ⚠️ A `block` stack eats one WHOLE STRIKE — great vs `outrage` 3400, weak vs 2-strike skills.

Related: [[hexarena-mechanics-log]], [[hexarena-status-naming]], [[hexarena-core-design]]

## hexarena-summon-share

hexarena PR #102 (merged 2026-08-28, `4f97659`) fixed three Naruto wordings that said what the numbers under them did not.

⚠️ **`describeSummon` now prints the SHARE and still not the fixed stat line.** They are not the same kind of number: a share is **one figure meaning the same thing wherever read** (40% of whoever made it is half of 80%, whichever character holds the skill) — the same argument the file makes for printing power as a share of a stat, not as damage. A fixed line is six figures nobody compares without the caster.

⚠️ **The share was left out because "the listing beside this carries it" — NO LISTING DOES.** Neither `hexforge` nor `hexforge-tui` mentions a summon **at all** (`grep -rn summon internal/forge/ cmd/` → nothing). The description is the ONLY place a summon is described. 2 copies at 10% and at 80% read identically.

```
Gọi ra 2 phân thân, mỗi bên mang 40% chỉ số người gọi, trụ lại 4 lượt.
```

⚠️ `share` vs `share_of_base` must read differently (one pays for pre-cast buffing, one ignores every buff). ⚠️ One copy vs several need **different wordings** — a pair handed the singular reads as a pair carrying 40% *between them*. Comparing the two descriptions is **vacuous** (they differ in count either way): the test splits the wording on its `%s` blanks and asks for the literals between them.

⚠️ **A creature is not a copy.** English never prints an authored VN name and falls back to a word — which was `"copy"` for everything, so the shipped toad read *"Calls up a copy"*. Told apart by **the stat spelling** (share = copy, own stat line = creature), the line the engine already draws.

Two data rules born here (both in `internal/seed/skills_test.go`):
- **`casterWords`** — a summon's flavour may claim nothing about its caster. **Both** shipped summons did: the clone restated its share, and the toad's *"to hơn cả người gọi"* is **contradicted by its own stat line** (toad hp 2400 atk 420 vs Naruto@44 2814/493; only def is higher).
- **`volleyWords`** — a ≤1-strike flavour may describe no volley. The half of the number ban neither check saw: `ParseBook` bans digits, `TestAFlavourClauseSpellsOutNoNumber` bans spelled numbers, *"một nhúm"* walks past both. ⚠️ `chùm` is deliberately OFF the list (same judgement that kept teeth out of `bodyWords`): `bubble` blows a cluster that lands as one hit.

`kunai` renamed **`phi đao`** — `phi tiêu` is the word a reader of the origin uses for the *other* skill's weapon (`wind_shuriken`, in the same kit). Every hexarena skill name is VN/Sino-VN; no loanword names.

⚠️ **8 mutations all caught, but 6 were re-run against the DEDICATED tests alone** — every one also moves the shipped golden, which proves nothing about the new tests. The dropped-"each" mutant **survived that second pass**. Always re-run golden-adjacent mutants with `-run` naming only the new tests.

⚠️ `gh pr merge` fails locally with `fatal: 'main' is already used by worktree at …` when another session holds a worktree on main — **the remote merge still succeeded**; verify with `gh pr view --json state` and delete the branch by hand.

**Squirtle tank build: ① reply scaling (#97) · ② taunt (#99) · ③ the skill/trait set — still NEXT.**

Related: [[hexarena-mechanics-log]], [[hexarena-descriptions-are-derived]], [[hexarena-flavour-voice]], [[hexarena-tui-i18n]]

## hexarena-squirtle-builds

hexarena PR #109 (merged 2026-08-28, `129fc90`) gave Squirtle **two builds out of one learnset** — 3 skills + 1 trait + 2 statuses, **no engine change at all**.

| | thuần tank | semi tank |
|---|---|---|
| skills | `taunt` `withdraw` `wide_guard` `aqua_ring` | `skull_bash` `water_gun` `whirlpool` `withdraw` |
| trait | `thorns` | `ballast` |
| measured | 517 turns, 5 dmg/turn | 30 turns, 100 dmg/turn |

Every mechanism was already there and unused: `rally` proved `target: ally`; `resolveAgainst` always ran `strips`/`restores` against whatever cell `aims` yielded; `skill.Scaling` was parsed/marshalled/described with **no shipped user**.

⚠️ **The stat line CANNOT differ between builds** — Squirtle absorbs 11285 of the 11500 effHP budget (215 spare). A split that had to live in the numbers is impossible here; **the 4-skill/1-trait slots are the whole mechanism** for build variety.

**`skull_bash`** (neutral, r1, power 850, cd2, `scaling.stat: defense`) is the **first shipped skill scaling off anything but attack**. def 640 / atk 460 = **1.39×** per point of power → a def-scaled skill wants ~**0.72×** the power of an atk-scaled one.

⚠️ **`ballast` is the ATTACKING build's trait, not the tank's** — measured, not intended. Survival is gated on how often `withdraw` is cast, so **10% off speed hurts more than 25% onto defence**: tank survives **1/30** with ballast vs **29/30** with endurance. Semi: 15/30 ballast vs 12/30 endurance. Final tune `fortified` +250 def / `encumber` −100 speed / reply power 50 off defence (swept +200/−300, +250/−150, +250/−100, +400/−250).

⚠️ **A TEST MAY NOT RAISE A STAT** — `battle.New` checks the roster stat line against the effHP budget, so +200 def fails before the battle starts ("over the budget of 11500"). Prove scaling by **halving attack**: `skull_bash 29526 → 29526` (exactly equal) while `water_gun 51625 → 25725`. A stat falling and damage falling with it proves **nothing** (a weaker unit dies sooner and swings fewer times) — the *unchanged* figure is the proof.

⚠️ **A trait's permanent statuses are OUTSIDE the budget** — `CheckValues` takes a resolved stat line and never sees a passive. `endurance` was already through that gap.

⚠️ **`battle.Suggest` takes a 0-power skill ONLY when it can find nothing to hit**, then the *first available* one. So: a support-only kit is exercised only because it carries no weapon; a kit with any weapon almost never guards. `hexforge spar` cannot measure either build (it fields the first four declared skills). Test `wide_guard` by hand via `fight.Advance()` → `fight.Act()`, **not** `fight.Pending()` (Pending alone never yields a prompt).

⚠️ Tank + `endurance` is **unkillable in a duel** — 29/30 hit the 4000-turn cap dealing nothing. Left as-is: a squad puts 5× the damage in.

⚠️ **New skill needs `restrict.origins` OR a `sharedPool` entry** (`TestEverySkillSaysWhichWorkItIsFrom`, from PR#104). `wide_guard` went in sharedPool.

⚠️ `bulwark` was rejected as a trait name — **already an `archetypeGloss` key**. Also: a permanent debuff needs a **verb** (`encumber`), a permanent buff a **participle** (`fortified`).

## hexarena-suggest-declines-a-loss (PR #131, 2026-08-28)

`Suggest` skipped an option scoring `value <= 0` and then the **fallback picked the same skill straight back up** — "worth nothing" and "worth less than nothing" were one bucket. So a unit whose only reachable skill was an all-sided shape covering more of its own squad cast it every turn, *after* computing the cost. A loss is now dropped from the fallback; exactly-nought is still taken (a shield at cap, a taunt with nobody near, is a fine way to spend a turn nobody wants).

⚠️ **`Pass()` has existed since #121** — engine, `cmd/hexarena`, `log.go` replay, seed tests. Adding a "decline" needs **no new engine action**: `Suggest` returning `false` is the existing route to it. I mis-scoped this as a new feature by grepping `battle.go` and `Act` only; `Pass` lives in `turn.go`.

⚠️ **`Queue.Next()` charges the wait when the turn is TAKEN** (`atb.go:193`, in `Advance`), before the unit chooses — so passing can never loop the queue. Any "should it cost time" worry about a pass is already answered.

⚠️ **The obvious loss case is wrong.** Recoil into a target on a sliver is *not* negative: the kill is priced and a kill dwarfs the recoil. The reachable loss is **friendly fire** — `crossfire` + `quake` across the midline with the caster and the unit behind both nearly dead. Wrote the wrong test first and it failed honestly.

**Still open after this**: waiting proper (pass because the *next* turn is worth more) needs a lookahead; where an extra turn falls needs the queue. Do not let a "declines a loss" commit tick that item.

## hexarena-preview-reads-the-caster (PR #137, 2026-08-28)

`forge.PreviewDamage` read only `requires` (the TARGET's condition), so `outrage` (self_requires) and `comeback` (self_gradient) — the two shipped skills whose whole design is a caster-side term — previewed at their plain power. The amplified figure is now **everything the skill asks for holding**: `requires` + `self_requires` + `self_gradient` at the bottom.

**Composition lives in `combat.Swung(power, bonus, share)`**, beside `combat.Gradient`. `battle`'s `swing.applied` keeps the *reading* (which unit, once per USE not per target) and delegates the arithmetic — one expression for engine and preview both.

⚠️ **The ordering — bonus added first, share taken of the sum — was guarded by NOTHING.** Swapping the two halves passed the whole suite; only the new preview test caught it. `TestSwungAddsTheBonusBeforeTakingTheShare` holds it now, and every discriminating case needs **both** terms (with either at nought the two orders agree).

⚠️ Caster asked at **health 0 out of 1** — same shape as `Condition.Satisfying()`, deliberately not a measurement of a real unit.

`Amplified` suppressed when equal to the plain figure, so the line means "this skill has a ceiling worth naming".

## hexarena-evolution-forks (PR #138, 2026-08-28)

`progression.Stage.After` names the stage a stage grows out of → a line is a **tree**, not an ordered list. Arms share a threshold; a stage may sit past the fork on one arm.

⚠️ **A line is read by ORDER or by NAME, never both.** No `after` anywhere → read by order (what every line meant before; no shipped file moved). Any `after` → every stage but the root must name one, and it must be **declared before it** (cycles unwritable, not detected). Mixing = refused.

⚠️ **`Furthest` refuses on a fork instead of picking.** `Line.Furthest(level)` = tip of EVERY arm; `StageAt` errors naming the arms. Whole change compile-clean because every `progression.Furthest` caller already had an error path — that is what made this safe.

Front-end halves that would otherwise lie: `hexforge check` prices **one row per arm** (budget bites per grown end; art on first row only), and `StageSummary` brackets arms — `Eevee@1 → (Vaporeon@32 | Jolteon@32 → Tempest@48)`. `i18n.Lang.StageSummary` now **delegates** to `forge.StageSummary` (they were byte-identical).

⚠️ Round-trip is a real risk: hexforge rewrites the whole cast file on save, so a dropped `after` turns a tree silently back into a list — asserted by marshal→re-parse in the cast test.

**Nothing shipped forks yet**, deliberately (mechanism without a balance move, like crit): no golden moved.

Related: [[hexarena-mechanics-log]], [[hexarena-status-naming]], [[hexarena-core-design]]

## PR #168 — chiêu TỰ NHẮM không trả `restores` (merged 55c2101)

User báo: bulbasaur cast `synthesis` mà không thấy hồi máu. **Đúng.** Đo trước khi
sửa: `hp 900 -> 900, healed events 0`.

Payout của `restores` nằm INLINE trong `resolveAgainst`, mà `Act` **return TRƯỚC đó**
với `Target: Self` (đúng — chiêu tự nhắm không có hình dạng để đi). Nên restore nằm
bên kia một cái return mà **đúng những chiêu khai báo nó không bao giờ tới**.
`withdraw` cùng hình dạng: `self_applies` block chạy (qua `applyToSelf`), 500 hồi máu
chết. Đó là **2 chiêu ship DUY NHẤT** khai `restores` → trường này chết toàn game.
Fix: `Battle.restore` thành method, thêm caller thứ hai ở nhánh self (position 0 —
tự nhắm không bao giờ bị splash).

⚠️ **AI THẤY, engine KHÔNG.** `pricing.restored` định giá restore; đo được
`Suggest → skill="synthesis"` trên caster gần chết. Đúng hình dạng `CLAUDE.md` cấm
("giá dựng từ bản đọc thứ hai khiến đối thủ ưa chiêu vì thứ nó không làm") — **chỉ có
điều bản đọc thứ hai mới là bản THẬT, còn resolving function mới là bản đã biến mất**.

⚠️ **Không golden nào dịch — ĐÓ LÀ PHÁT HIỆN.** Không unit roster nào mang 2 chiêu đó.
Cộng return sớm = toàn bộ lý do trường khai báo chết lâu vậy mà test xanh. **Cùng cặp
lý do với bug regen (`aqua_ring`/`ingrain`).** Bài: **cơ chế không placement ship nào
fielding = cơ chế không gì đo được.**

Test nhắm HÌNH DẠNG chứ không nhắm 1 chiêu: cặp fixture chỉ khác `target` so **hai nửa
BẰNG NHAU** (không so với con số — con số là của `combat.Rules.Restore`, viết ra là
bản sao thứ 2 của phép tính); + duyệt MỌI chiêu ship có `restores` trong `internal/seed`.
⚠️ Cặp fixture khác cả `range` (0 vs 1) vì parser ép — ghi vào comment thay vì nói dối
"chỉ khác một field".

⚠️ **CLAUDE.md có 2 câu sai, sửa chứ không xoá lặng lẽ:** (1) mục regen viết
"`synthesis` chưa bao giờ bị ảnh hưởng, nó hồi qua `restores`, thứ luôn chạy" — cả HAI
nửa hồi máu ship đều chết cùng lúc bằng 2 đường, và note sửa nửa này khẳng định nửa kia
ổn; (2) "`Suggest` không bao giờ chọn heal" — `pricing.restored` bác bỏ.

⚠️ **Số balance có trước:** "khả năng sống Squirtle phụ thuộc tần suất cast `withdraw`"
đo khi restore của withdraw ĐÃ CHẾT → đọc mỗi mệnh đề block. Cắm cảnh báo tại chỗ số
và trong CLAUDE.md; phải đo lại trước khi trích.

---

## PR#178 — 2 con số trong log là số THÔ của engine

**`power 3500` → `power x3.5`.** Mọi tỉ lệ = permille trên `scale.Base`; log in raw
nên `power` trông như một con số sát thương nằm cạnh sát thương thật. `tui.multiple`
trim số 0, KHÔNG làm tròn (1‰ vẫn `0.001` — log phải tái lập được theo rules).
ASCII `x`, không `×` (xem [[terminal-ambiguous-width-glyphs]]).

**`giving up` = một tick → cả phần còn nợ.** `Set.Remove` cũ cộng `TickAmount`, giờ
cộng `Stack.worth() = TickAmount × Remaining` — **đúng biểu thức `Pending()` đã dùng
để định giá cleanse**; 2 chỗ nói 2 kiểu suốt. Không đổi trận: `forgone` chỉ chảy vào
`Event.Amount`, 2 caller `Remove` còn lại vứt giá trị. `scenarios_test` tự nhân
`Duration` → bỏ, golden **y hệt byte** = bằng chứng con số mới mới là con số vẫn muốn.

⚠️ **Bẫy tautology (tôi dính):** khẳng định `consumed.Amount == Statuses.Pending()`
**PASS cả khi mutate** — hai bên cùng một biểu thức. Phải đọc độc lập:
`TickAmount(id) × Remaining(id)`. Kèm chốt `Remaining >= 2`: ở lượt cuối
`tick` và `tick × còn lại` TRÙNG NHAU → fixture tụt về đó là test rỗng
(cùng họ với [[fixture-hidden-branch]]; mọi fixture cũ đều consume ngay lượt dán).

**Sort "nặng nhất trước" đổi sang `worth()` mà không đổi hành vi**: `Apply` refresh
`Remaining` của MỌI stack nên các stack cùng status luôn cùng duration.

**Số đo (engine, không tính tay)** — Charizard lv60 atk 700, inferno vào Blastoise lv60
def 640→**786** (`fortified` từ `ballast`), fire→water 667‰: không bỏng **128**, tiêu
bỏng **451**, burn tick **103**. Khớp CHÍNH XÁC log người dùng đưa → đó là thứ ghim
cách đọc. ⚠️ `bonus_power` PHẲNG theo stack còn `Consume` lấy HẾT stack → **bỏng x2 vừa
dán thì tiêu là LỖ**; hoà vốn ở `323/103 ≈ 3.13` lượt-cháy còn lại.

---

## PR#179 — unit ra sân dưới tên DẠNG, không phải tên nhân vật

Level **cho phép** một dạng chứ không quy định; stat line đánh nhau là của **dạng**.
Nhưng cả **3** chỗ resolve character → `battle.Roster` đều đặt `character.Name`:
`placement.Placement.resolve` (màn squad + mọi squad fight), `seed.resolveRosterEntry`
(roster.json), `forge.place` (spar → `who.Stage`). Cả 3 đều **đã resolve ra `form`** rồi
chỉ dùng để gate learnset, xong vứt tên. → "Bulbasaur" đứng cạnh hp 3800 của Venusaur.

⚠️ **Cách phát hiện = đọc hp max/spd chứ không đọc tên.** lv60: Venusaur 3800,
Charizard 3100 (spd 140 CHỈ Charizard có: 118/130/140), Blastoise 3600.
`squads.json` không khai `stage` → luôn lấy dạng xa nhất.

Định danh nhân vật KHÔNG mất: vẫn nằm ở placement id + tag A1/A2. `forge.Duellist`
giữ RIÊNG `Name` và `Stage` vì report spar có 2 cột — chỉ `Stage` được phép tới battle.

**Chốt chống test rỗng (dùng ở cả 3 test):** dòng tiến hoá nào có dạng cuối trùng tên
nhân vật thì 2 lựa chọn là CÙNG MỘT CHUỖI → test tự `Fatal` thay vì pass giả.
Cùng họ [[fixture-hidden-branch]]. Test roster còn khẳng định tên **đi xuống theo dòng**
(cùng nhân vật ở 2 level = 2 tên) chứ không chỉ đúng ở cap.

Golden không đổi: fixture replay dùng **dạng flat** tự khai `name`, không qua resolve.
