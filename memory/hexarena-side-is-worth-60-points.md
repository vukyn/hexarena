---
name: hexarena-side-is-worth-60-points
description: hexarena — thắng thế hoà = THỨ TỰ SLICE roster (caller quyết, không phải core); slot đáng −38..+62% ở 1v1 nhưng +8.5pp ở 2v2 và CHƯA ĐO ở 5v5
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

## Giá của slot — và nó KHÔNG chuyển sang trận nhiều người

`Matchup.Edge() = First.Rate() − Second.Rate()`, hiện ra cột `first move` của `hexforge spar`. **Đã có sẵn, đừng viết lại.** 500 seed mỗi slot, mirror **1v1**:

```
naruto.naruto      +62.0%      pokemon.charmander +19.6%
pokemon.bulbasaur  +24.8%      pokemon.machop      +6.4%
pokemon.poliwag    +24.4%      pokemon.cleffa     −38.0%   ← ÂM
pokemon.squirtle   +20.0%
```

Biên độ 100 điểm, **dấu không cố định** — đi trước là *bất lợi* cho cleffa. Nên phát hiện không phải "slot 1 luôn thắng", mà là **slot có giá rất to và hướng của nó là thuộc tính của kit**.

⚠️⚠️ **Nhưng đây là số DUEL và nó không chuyển.** Cùng cách đọc ở đội 2 người: **+8.5 điểm**, không phải sáu mươi — trận dài hơn thì loãng cái mà duel dồn hết vào mở màn. Và từ 2 người/phe thì **tỉ lệ một chiều không còn là phép đo** ([[one-way-mirror-not-a-measurement]]), nên slot đáng bao nhiêu ở 3v3/5v5 là **CHƯA ĐO**, không phải "nhỏ". Số để trích ở đó **chưa tồn tại**.

## Cùng speed thì dính nhau VĨNH VIỄN

`Next()` đặt `now = acting.next` rồi đẩy `acting.next += Wait`. Unit kia vẫn còn `next == now` → đi ngay sau, cùng thời điểm. Rồi cả hai lên `2W`. Nên bên thắng thế hoà không đi trước *một lần* — nó đi trước **mọi vòng**, tới khi buff/debuff đổi speed.

## Hệ quả khi xen thứ tự roster

`tui.Roster` (`tui.go:53`) duyệt `fight.Units()` **theo thứ tự enlist** → bảng roster hiện `A1 E1 A2 E2` thay vì nhóm hai phe. Sửa một dòng (sort theo side khi vẽ) nhưng phải nêu. `tui.Tags` **không** ảnh hưởng (đếm theo từng phe), và `TagsFromLog` đếm từ event `Started` cùng thứ tự đó nên hai bên vẫn khớp.

## ĐÃ LOẠI — đừng nêu lại

**Sửa `atb.Queue.order` để roll thế hoà.** Không cần (xem trên: thứ tự roster đủ), và nếu sửa `order` thật thì mất mọi con số đã đo — 47.3%, `Suggest` 81.3%, mọi control 500‰, mọi golden. `internal/core` không đổi vì feature mạng.

Liên quan: [[hexarena-pvp-plan]], [[hexarena-bout-and-waiting]], [[hexarena-speed-and-measurement]].
