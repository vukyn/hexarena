---
name: hexarena-a-rating-that-prices-nothing-stops-playing
description: "hexarena — ĐÃ VÁ: blow bị guard nuốt trọn định giá 0 nên Suggest bỏ lượt mãi; credit chỉ cho guard PERMANENT (credit phẳng làm 40/40 mirror ship Endless), share 100‰ trong cửa sổ đo được 50..999"
metadata:
  type: feedback
---

A mirror of two units carrying `carapace` never finishes. Measured to the turn:
over eight turns **neither unit acts once**, and after six hundred turns both
stand at **3600/3600 with 576 of pool left**. The pool is never spent because no
blow is ever thrown at it.

The chain is short. `pastAPool` returns **0** when the pool covers the blow →
`expected` is nought → every option rates nought → the pass rule reads "nothing
worth doing" → both sides pass, for ever.

**Why it hid for a release.** A rating that mis-prices an option picks the wrong
skill, and somebody eventually notices the skill. A rating where *every* option
reads nought **stops playing**, and two sides standing still produce a long battle
rather than an obviously broken one — the turn limit catches it and reports
nothing about what happened. The recorded diagnosis blamed a "wall-heavy roster"
and a pass rule; both were wrong, and a plain **1v1** reproduces it.

**How to apply.**

- When a board does not resolve, ask **what the rating chose**, not what the units
  are. One probe printing the chosen skill per turn answered in a single run what
  three sessions of win-rate readings had not.
- A defensive term in `price.go` prices what a guard is worth **to its holder**
  (`shielded`, `guarded`). There is no term anywhere for what **destroying** one is
  worth to the attacker, and that asymmetry is the defect: progress that is not
  health lost is invisible to the whole file. Compare
  [[hexarena-a-denied-turn-is-not-a-lost-turn]] — the same blind spot from the
  other end: what an enemy KEEPS when its aim is denied is invisible too.
- **A guard board must be asymmetric to resolve.** Only one side may carry the
  guard. That is also the right shape for measuring what a guard is worth, so the
  board that works and the board that answers the question are the same board.

⚠️ **ĐÃ VÁ, và không phải bằng cách entry đề xuất.** `battle.guardCredit` = 100‰,
tính công cho phần một guard **PERMANENT** nuốt — `Set.PermanentPoolIn` là phép đọc
mới cho việc đó.

⚠️ **Credit phẳng (mọi guard) KHÔNG ship được, đo ra chứ không suy ra.** Bản viết
trước làm đúng như entry đề xuất: `pokemon.happiny` và `pokemon.squirtle` đấu bản
sao của chính nó chuyển từ kết thúc mọi seed sang **40/40 Endless** →
`TestABothWaysMirrorIsExactlyEven` đỏ, đúng bất biến công bằng đã từng bác bỏ vụ cho
`stat_debuff` xuyên shield. **Không share nào cứu được**: mirror roster ship cần
**≤10**, fixture guard cần **≥50** — giao rỗng.

**Thứ phân biệt là guard có QUAY LẠI hay không.** `withdraw`→`block` (2 lượt, cast
lại), `brace`→`heft`; chỉ `carapace`→`bastion` là permanent — cấp một lần, `Apply`
từ chối stack thứ hai, cạn là mất hẳn. Gặm guard tái tạo chỉ mua được đúng số lượt nó
quay lại; gặm guard permanent mới là tiến triển. ⚠️ Ba build guard đang ship
(`squirtle.fortress`, `squirtle.ram`, `machop.charge`) đều mang guard **có hạn** →
bán kính ảnh hưởng bằng **không**.

⚠️ **Share ĐO ĐƯỢC, và ghi chú cũ ở đây nói ngược.** Sweep cũ chỉ hỏi *bàn có kết
thúc không* — mọi giá trị từ 10% lên đều "có", giống hệt nhau: một **bậc thang**, mà
bậc thang thì không đọc ra được cực đại. Hỏi thêm một bàn thứ hai là đóng được cửa sổ.
Quét bước 5‰ sau khi thu hẹp về guard permanent: **≤45** mirror đứng im lại (credit là
MỘT phép chia truncate, 21×45÷1000 = 0 — sweep chỉ-hỏi-kết-thúc sẽ chấm 45 là đạt);
**50…999** mọi bàn xanh; **1000** một điểm guard bằng một điểm máu, mục tiêu có guard
hoà với mục tiêu trần và `take` giữ cái aim walk gặp trước.

⚠️ **Hai trong ba test bàn ban đầu VÔ NGHĨA**, và fixture ghi lại: đặt kẻ có guard vào
ô aim walk duyệt **sau**, thế hoà rơi đúng chiều một cách tình cờ và cả hai đều xanh
kể cả khi xoá luật. Đổi ô mới thấy.
