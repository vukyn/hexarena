---
name: hexarena-block-cancels-the-rider
description: "hexarena — a blocked strike now lands its `dot` riders and NOTHING else (PR #181); a missed strike still lands nothing; 0-power skills bypass block entirely; letting stat_debuff through was measured and rejected; and Drain() empties the buffer, which makes naive probes lie"
metadata:
  type: reference
---

⚠️ **This rule CHANGED on 2026-08-31 (PR #181, `343affc`). Earlier notes saying a blocked strike delivers nothing are out of date for `dot`.**

## The rule now

`internal/core/battle/resolveAgainst` carries **`blocked` beside `connected`**. A strike that landed sets `connected`; one a shield ate sets `blocked`. The rider block runs on `connected || blocked`, and when nothing landed (`throughAShield := !connected`) it keeps only riders whose category satisfies `status.Category.OutlastsAShield()` — **true for `Dot` and nothing else.**

**The reading:** *a shield stops the blow and the wear, but not the contamination.* Fire burns you through a shield and poison gets on you, because both are something **left on** the target rather than something **done to** it. A stat the blow never bent and a turn it never took are stopped with the strike — nothing is left over for them to be about.

⚠️ **A MISSED strike still delivers nothing, dot included.** That is the whole justification: a block means the blow arrived and was stopped, a miss means nothing touched the target. `blocked` is a second flag rather than a widened `connected` for exactly this reason.

A **multi-strike** skill with one strike eaten and one through has `connected`, so it gets the full rider set. (No shipped skill is both multi-strike and an applier, so that branch is latent.)

Trait riders under the same gate take the **same** filter — a trait's rider surviving a block on a different rule would be a difference no reader could find on either. **0 of 11 shipped traits declare `applies`**, so that half is latent too.

Measured, 60 seeds against a shielded squirtle, counting only casts where every strike was blocked: `flamethrower` (dot) 497 casts → **252 applied / 245 resisted** (it *rolls*); `whirlpool` (stat_debuff) 272 → **0/0**; `water_pulse` (control) 218 → **0/0** — no `StatusApplied` **and** no `StatusResisted`, so the roll never happens.

Reaches **5 of 43** shipped skills: `sludge_bomb` `ember` `flamethrower` `fire_spin` `heat_wave`.

## ⚠️ stat_debuff was measured and REJECTED — do not "complete" the predicate

With `mire` unstoppable (−25% speed a stack, 2 stacks), `pokemon.squirtle` against itself **stops resolving**: 0 of 20 duels finish inside spar's 4000-turn limit (Endless 40 of 40 across both arrangements) against 20 of 20 finishing with a kill; mire applications go **373 → 12875**. That breaks `TestABothWaysMirrorIsExactlyEven`, which is a **fairness invariant** (a character duelling an identical copy of itself comes to exactly 500‰), not a balance number. Also **never fold it into `Harmful()`** — that is `Dot|StatDebuff|Control|Taunt` and answers what a *cleanse* may strip.

⚠️ A one-case switch is exactly what a later reader tidies away, so the rejection lives in the predicate's own doc comment, not only here.

## ⚠️ 0-power skills bypass a shield entirely

The whole block/roll branch sits inside `if power > 0`. Five shipped skills go straight through a shield regardless of this rule: `poison_powder`, `sleep_powder`, `smokescreen`, `rally`, `wide_guard`.

`block` itself: category `shield`, `max_stacks` 3, `duration` 2; one charge eats one strike, spent charges removed before the strikes resolve. Only `wide_guard` (ally, ×2) and `withdraw` (`self_applies`, ×2, also restores 500) grant it. Visible to the player on the **shield category** in the statuses reference, not on any skill — the rule is global and per-skill text is derived.

## ⚠️ dot is NOT a real counter to a shield build — measured

The obvious conclusion ("dot now counters shield stacking") does **not** survive measurement:

| matchup, 300 seeds | baseline | with the rule |
|---|---|---|
| burners vs a 2-unit **shield wall** (`wide_guard`+`withdraw` screening a carrier) | wall loses 279/300 | wall loses **290/300** |
| the same pair **brawling** instead of shielding (control) | 280/300 | **280/300, byte-identical** |
| 1v1 burn vs a passive shield build carrying `rapid_spin` | burn wins 4/300 | **89/300** |
| the same shield build with `rapid_spin` swapped for an attack | burn wins **0/300** | **0/300** |

Three findings, and the third is the one that matters:

1. The shield wall was **already losing**; the rule takes ~4 points off it and *reduces* stalemates (endless 17 → 9). It shortens a way of grinding rather than opening a way of winning.
2. **`rapid_spin` — the only shipped cleanse — is a LIABILITY.** Swapping it for an attack makes the shield build win 300/300 under both rules. Spending a turn to strip 1 stack is worse than spending it on damage. So a shield build's answer to dot is **not** to cleanse, and dot only "counters" a *passive* shield build, i.e. one that was already bad.
3. **A shield build's survivability is mostly HEALING, not BLOCKING** — `withdraw` restores 500 a cast, `aqua_ring` grants regrowth, `endurance` grants toughened. Dot pierces the block and not the healing. So "counter to shield" really means "counter to block", and block is not the load-bearing half. Making dot a genuine counter would need a **heal-blocking status** (none of the 21 shipped statuses does this) or a dot tick that ignores regen — new mechanics, not a tweak to this rule.

## ⚠️ HOW TO PROBE THIS ENGINE WITHOUT LYING TO YOURSELF

**`Battle.Drain()` EMPTIES the buffer.** Two of my three measurements of this mechanic were wrong and both wrong answers looked entirely plausible:

- calling `Drain()` twice around one `Act` (once to count, once to read) makes the second call see **nothing** — reported `damaged=0`, which happened to agree with the answer I expected;
- not draining on the turns you are *not* measuring lets other units' events pile into the next batch — reported 1341 of 3535 fully-blocked targets "rolled for poison", which happened to *contradict* it.

Drain after `Begin()`, after every `Advance()`, and after every `Act`/`Pass`, so the batch you read holds one cast and nothing else. Driving API: `Advance() (*Prompt, error)` → `Act(skillID, aim)` / `Pass(reason)`. There is no `Next`/`Take`.

⚠️ **What caught both bugs was the CONTROL, not review.** Running the block-off arm beside the block-on arm exposed them, and a guard that fatals when either arm never occurred is what stopped a one-armed run being read as an answer. Same for the counter-measurement above: the "brawling" control coming back byte-identical is what proved the shift was the rule.

⚠️ Also beware the cap: a probe with `RunToEnd(400)` reported both rules as 40/40 unfinished and hid the whole effect. Spar's own limit is **`sparTurnLimit = 4000`**; use the harness's limit, not a round number.

Related: [[hexarena-mechanics-log]], [[hexarena-core-design]], [[hexarena-range-is-rank-depth]], [[fixture-hidden-branch]], [[hexarena-log-gloss]].
