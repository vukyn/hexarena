---
name: hexarena-side-is-worth-60-points
description: hexarena — thắng thế hoà = THỨ TỰ SLICE roster (caller quyết, không phải core); ĐÃ KÉO CẦN GẠT 2026-09-08 trong room.alternateContested, gap −45% ở 3v3
metadata:
  type: reference
---

## Cần gạt nằm ở caller, KHÔNG ở core

```
battle.New       for _, entry := range roster { enlist(entry) }   ← THỨ TỰ SLICE
battle.go:319    queue.Add(unit.ID, speed)
atb.go:105       q.joined++ ; seq: q.joined
atb.go:178       ... return left.seq < right.seq
```

`seq` = thứ tự phần tử trong slice `roster` mà **caller** truyền vào `battle.New`. Nên ai thắng thế hoà là **quyết định của caller**: sửa được mà **không chạm `internal/core`, không golden nào nhích** (data hiện có giữ nguyên thứ tự của nó), và `Log.Roster` ghi lại thứ tự nên `--verify` vẫn chạy.

Ally thắng mọi thế hoà hôm nay **chỉ vì** `forge.FightSquads` viết `append(ally, enemy...)`.

⚠️ PR #193 ghi ngược lại ("phải sửa `atb.Queue.order`, sẽ mất mọi con số cân bằng") — **SAI**, đã sửa ở #196.

Đo mirror 2 unit, 2000 seed: ally-first **54.2%**, enemy-first 45.7%, một xu cho cả phe **50.2%**, luân phiên từng cặp **49.6%**. Cần gạt chạy và miễn phí.

## ĐÃ KÉO CẦN GẠT (2026-09-08) — `room.alternateContested`

Luân phiên lead của mỗi **nhóm speed có tranh chấp** (nhóm = unit của **cả hai phe**
cùng một speed). Nằm trong `internal/room`, `begin()` gọi sau `append(home, away...)`.
Không chạm `internal/core`, không golden nào nhích.

Đo bằng **khoảng cách giữa hai arm** (không phải tỉ lệ một arm — xem
[[one-way-mirror-not-a-measurement]]), 2000 seed mỗi squad:

| squad | home nguyên khối | luân phiên |
|---|---|---|
| s01 | ±129.5‰ | ±76.4‰ |
| s02 | ±242.4‰ | ±89.4‰ |
| s03 | ±54.5‰ | ±42.8‰ |
| s04 | ±121.3‰ | ±93.3‰ |
| s05 | ±54.0‰ | ±26.5‰ |

Nhỏ hơn ở **cả năm**, trung bình **−45%**. 5v5 (squad tự ghép vì squad ship chỉ 3 người):
±46.0‰ → ±2.5‰, và ±125.7‰ → ±69.3‰.

⚠️ **Luân phiên theo CẶP, chạy xuyên các nhóm** — không phải mỗi nhóm một lần. Hai cách
đọc chỉ khác nhau khi một nhóm có hơn một unit mỗi phe; mirror toàn speed khác nhau
không bao giờ tạo ra ca đó, nên phải dựng fixture riêng
(`TestTheLeadChangesHandsInsideOneSpeedGroupToo` — machop 85 + squirtle 85 cả hai phe).
Bản "mỗi nhóm một lần" **xanh** ở mọi test khác trong file.

⚠️ **Nhóm theo speed LÚC ENLIST, không phải speed tác giả.** `enlist` áp composition
bonus + passive **trước** khi `queue.Add` đọc speed: Magnezone tác giả 110 → enlist 117,
và → **123** nếu cùng phe có unit electric thứ hai, trong khi đối thủ vẫn 117. Đọc
`Roster.Stats` sẽ gọi hai con đó là một cặp tranh chấp — cặp không tồn tại — và lệch pha
mọi nhóm phía sau. Nên phải dựng **một `battle.New` để đọc speed rồi vứt đi**; tính chất
giữ cho việc đó lương thiện là `TestTheSpeedsDoNotDependOnTheOrderTheRosterIsIn`.

⚠️ **Nhóm CHỈ MỘT PHE không tiêu lượt luân phiên nào.** Không ai thắng thế hoà ở đó.
Bỏ guard này trông như đơn giản hoá và làm lệch pha mọi nhóm sau nó.

⚠️ **KHÔNG thay cho đánh hai chiều.** `Config.HomeFor` mới là cái triệt tiêu phần dư;
cái này cộng thêm lên trên, từng trận.

## Giá của slot — và nó KHÔNG chuyển sang trận nhiều người

`Matchup.Edge() = First.Rate() − Second.Rate()`, hiện ra cột `first move` của `hexforge spar`. **Đã có sẵn, đừng viết lại.** 500 seed mỗi slot, mirror **1v1**:

```
naruto.naruto      +62.0%      pokemon.charmander +19.6%
pokemon.bulbasaur  +24.8%      pokemon.machop      +6.4%
pokemon.poliwag    +24.4%      pokemon.cleffa     −38.0%   ← ÂM
pokemon.squirtle   +20.0%
```

Biên độ 100 điểm, **dấu không cố định** — đi trước là *bất lợi* cho cleffa. Nên phát hiện không phải "slot 1 luôn thắng", mà là **slot có giá rất to và hướng của nó là thuộc tính của kit**.

⚠️⚠️ **Nhưng đây là số DUEL và nó không chuyển.** Cùng cách đọc ở đội 2 người: **+8.5 điểm**, không phải sáu mươi — trận dài hơn thì loãng cái mà duel dồn hết vào mở màn. Và từ 2 người/phe thì **tỉ lệ một chiều không còn là phép đo** ([[one-way-mirror-not-a-measurement]]), nên slot đáng bao nhiêu ở 3v3/5v5 phải đọc bằng **khoảng cách hai arm**. Đã đo 2026-09-08 — xem bảng ở trên.

## Cùng speed thì dính nhau VĨNH VIỄN

`Next()` đặt `now = acting.next` rồi đẩy `acting.next += Wait`. Unit kia vẫn còn `next == now` → đi ngay sau, cùng thời điểm. Rồi cả hai lên `2W`. Nên bên thắng thế hoà không đi trước *một lần* — nó đi trước **mọi vòng**, tới khi buff/debuff đổi speed.

## Hệ quả khi xen thứ tự roster

`tui.Roster` (`tui.go:53`) duyệt `fight.Units()` **theo thứ tự enlist** → bảng roster hiện `A1 E1 A2 E2` thay vì nhóm hai phe. Sửa một dòng (sort theo side khi vẽ) nhưng phải nêu. `tui.Tags` **không** ảnh hưởng (đếm theo từng phe), và `TagsFromLog` đếm từ event `Started` cùng thứ tự đó nên hai bên vẫn khớp.

## ĐÃ LOẠI — đừng nêu lại

**Sửa `atb.Queue.order` để roll thế hoà.** Không cần (xem trên: thứ tự roster đủ), và nếu sửa `order` thật thì mất mọi con số đã đo — 47.3%, `Suggest` 81.3%, mọi control 500‰, mọi golden. `internal/core` không đổi vì feature mạng.

Liên quan: [[hexarena-pvp-plan]], [[hexarena-bout-and-waiting]], [[hexarena-speed-and-measurement]].
