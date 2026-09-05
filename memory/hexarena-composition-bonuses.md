---
name: hexarena-composition-bonuses
description: "hexarena — squad composition bonus: MECHANISM + first bonus SHIPPED 2026-09-06 (same_element, rungs 2/3); ⚠️ buff vĩnh viễn phải > quickened 80‰ nên 50‰ bị luật từ chối"
metadata: 
  node_type: memory
  type: project
  modified: 2026-09-06T00:00:00.000Z
---

Squad **composition bonuses**: fielding several units that share something grants a bonus at a threshold (rung). User's idea, filed in `TODO.md` across PRs #289/#290/#291, **built 2026-09-06**.

**Đã ship:** `internal/core/composition` (đếm, ladder, 2 scope, `Book.Without`) · `data/bonuses.json` (file data thứ **16**) · `battle.Books.Bonuses` award **trước `queue.Add`** · event `bonus_held` (mang bonus + giá trị chia + số người) · bonus `same_element` sharers-only, bậc 2/3, cấp status vĩnh viễn `kinship` 1 và 2 stack (attack +100‰/stack).

⚠️ **`kinship` 50‰/stack đo êm hơn (+54‰ / +203‰) nhưng KHÔNG hợp lệ.** `TestASpeedTraitIsPricedBelowTheOtherPermanentBuffs` bắt mọi buff vĩnh viễn phải **lớn hơn** `quickened` = 80‰: một điểm tốc độ đáng hơn một điểm bất cứ gì, đặt ngang là biến trait kia thành bẫy. Sàn cho mọi grant vĩnh viễn là **81**, giá nhà là 150, nên bonus ngồi ở **100**.

**Số đo ở 100‰/stack** (cùng đội, cùng seed, `FightSquads(..., "same_element")` so với bật): bậc 2 **+111‰**, bậc 3 **+448‰** (lật hẳn một cặp 177→625‰); mirror 500‰ đúng chằn. ⚠️ **Cặp bão hoà định giá bằng KHÔNG** — rate đã 1000‰ hay 0‰ thì bonus đo ra +0, nên chỉ 2 trong 8 cặp chạy là trích được.

⚠️ **Không đội ship nào bắn bonus**: s01–s04 mỗi đội 3 hệ khác nhau. Đây là thứ người chơi *dựng đội để lấy*, không phải thứ có sẵn.

⚠️ **Hệ trơ KHÔNG thành bộ tộc**, và luật đọc từ `element.Chart.Inert()` chứ không viết chữ `neutral`: chia nhau cái hệ không khắc chế ai là chia nhau sự vắng mặt của một hệ. See [[hexarena-shipping-a-character]], [[hexarena-roster-cannot-price-damage]], [[hexarena-tui-references]].

**Why:** the design was settled *before* code, and four of the decisions were made against measurements off the shipped data rather than taste. Rebuilding that reasoning from the JSON costs a full session.

**How to apply — the 7 settled answers (2026-09-04):**

1. **Count taken ONCE on entering the battle.** Not recounted, summons don't count. → it's a drafting decision, never a tactic; needs no `tickStatuses` hook, no death recount, and **no map walk** (roster is still a slice before turn 1).
2. **Dual affinity counts toward BOTH halves** (Lapras water/ice, Magnemite electric/metal). ⚠️ First thing in the game that pays a dual for being one — CLAUDE.md measures a dual's *defensive* half as ≈nothing.
3. **Two kinds ship**: whole-squad, and sharers-only. A bonus declares which.
4. **Bonuses STACK** — several of each kind live at once is normal.
5. **One bonus at a time, each must do something no other does.** A new bonus's PR states what no shipped bonus already does; "nothing" → doesn't ship.
6. **Rungs 2 and 3 only.** 4/5 authored later *with* 5v5 — not declared-and-quiet. A rung that can't fire isn't declared.
7. **Its own JSON file** (a rule about squads, not an axis of how one unit fights).

**⚠️ Reachability measured — 2 of 4 obvious axes are dead:**

| axis | max sharing one value | rungs |
|---|---:|---|
| element | 3 (water) | 2, 3 |
| origin | 14 (`pokemon`) | free at every rung |
| species | 2 (`plant`, `mythic`) | rung 2, twice |
| archetype | 1 each | **none** |
| archetype `column` | 6/5/4 | 2,3,4,5 |

⚠️ **ORIGIN IS UNUSABLE TODAY AND STACKING MAKES IT WORSE.** No 3v3 squad can *fail* the origin axis: **17/18** are `pokemon` and the 18th is one character, so worst case = Naruto + 2 Pokemon = still rung 2. ⚠️ Figure derived, not remembered — it read 14/15 and then 16/17 on 2026-09-05 alone, and every character that ships makes this objection **stronger**: `jq '[.characters[]|select(.id|startswith("pokemon."))]|length' internal/seed/data/cast.json`. `Hidden` doesn't save it, and saves it **less** than this used to say: this called the flag "an authoring convenience by its own doc" and ⚠️ that half of the doc is gone — `internal/draft.NewPool` gates the ban-and-pick pool on it, so it is a rule of the game for a drafted squad. Either way naruto is both the only hidden character **and** the only non-`pokemon` one, so honouring the flag removes the one character a squad could have failed the axis with and makes an origin bonus **more** unconditional. **Element first; hold origin until a 2nd origin can field 2–3.**

⚠️ **`column` was the best-shaped axis** (every rung reachable, means formation not tribe) — user decided **no column bonus**, which is what killed the correlation objection (water bundles a free column rung, grass doesn't: `blighter` col 1 vs `sapper` col 2).

**⚠️ Pricing — the prerequisite, not a later step.** Nothing is measurable until `forge.FightSquads` can disable **a set of bonuses** (not a boolean — a global switch measures *the system*, never one rung). Control = same squad, same members, same seeds, ONE bonus toggled, others left on; mirror control must read 500‰ exactly. Both obvious controls are already known wrong: swapping a member measures *the member*, and the same bonus on both squads **cancels** (Oddish pairing read −29‰).

**Other gotchas written into the entry:**
- `Squad.Validate` refuses a repeated unit **id** and **slot**, says nothing about the same **character** twice (settled "it MAY") → a rung-2 is one character fielded twice. #268 measured 3 copies as the *weakest* squad (~11%) — but that was measured with **no** composition bonus in play.
- Per-character `EffectiveHP ≤ 11500` stops bounding a **squad-wide** grant. Ceilings **saturate**, so a grant near a ceiling buys less than the number says.
- A bonus as a permanent **status** gets log/drawing/describers free — and makes `dispel` a question nobody has asked. Baked into `Take`/`New` = invisible, undispellable, unexplainable.
- PvP is cheap: the squad crosses the wire whole in `wire.Hello`, both ends derive it, a disagreement shows as a **per-turn digest mismatch**.
- ⚠️ A new data file = a **16th name in 3 independent places**: the `//go:embed` line in `internal/seed/seed.go`, the `dataFiles` slice in `internal/seed/digest.go`, and one `XxxFile()` accessor. **Missing `dataFiles` is silent** — the file loads and the data digest stops covering it, so two peers on different bonus data pass the gate then diverge.

**Màn tra cứu — ĐÃ LÀM (2026-09-06):** `internal/screen/bonuses.go`, read-only, vào menu **cả hai client** ngay sau Origins (entry thứ **10** ở client game, thứ **13** ở client authoring), đăng ký ở cả hai sweep và cả **ba** golden.

⚠️ **Cột scope in TỪ CỦA DATA (`sharers`/`squad`), câu giải thích để ở phần mô tả.** Bản đầu in câu tiếng Việt ở cả hai chỗ → đúng một dòng lặp lại chính nó trên màn chỉ có một bonus. Đây là cách màn statuses vẫn làm với category: hàng in `stat_debuff`, mô tả mới nói nghĩa.

⚠️ **`bonusesRoom` đếm grant thay vì hằng số.** Rung là thứ *chắc chắn sẽ mọc* (2 nay, 4 khi mở 5v5), nên room tính từ bonus rộng nhất **trong sách**; hằng số ở đây là con số lặng lẽ hết đúng, và listing sẽ vẽ đè lên mô tả. `TestTheBonusRoomLeavesTheWidestDescriptionItsRows` so sách 4 rung với sách 1 rung.

**Cũ — ý định của sub-item, giữ lại vì màn tra cứu sau sẽ hỏi lại đúng mấy câu này:**

**Sub-item — a reference screen on the menu** (user asked for it): read-only catalogue like statuses/elements/traits/species, both clients. ⚠️ It is the **10th** `menuItems` entry (9 today). ⚠️ Must go in the sweep — `model.go` records **five** screens that slipped it and silently lost width/translation/leak tests (`screenCount` + `TestEveryScreenThisClientDrawsIsSwept`). ⚠️ Data screen → `UsableWidth()`, but its **footer** is measured against the 120-col floor and the floor is a footer's only lever. Must draw the two *kinds* apart and show that several fire at once, and not look broken when 2 rungs become 4. **Cannot be built before the first bonus exists.**
