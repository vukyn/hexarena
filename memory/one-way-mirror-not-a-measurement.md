---
name: one-way-mirror-not-a-measurement
description: hexarena — tỉ lệ mirror MỘT CHIỀU chỉ đo được gì ở 1 unit/phe; từ 2 người là hỏng; không phải bàn cờ, không phải cấu trúc — phụ thuộc KIT, chưa tìm ra
metadata:
  type: reference
---

Đo 2026-08-31 (PR #196). Mirror đánh một chiều + chiều ngược **phải** cộng thành 1000‰ — cùng những trận đó, đổi vai. Hàng giữa, 1000 seed mỗi arm:

| cỡ đội | ally-first | enemy-first | tổng |
|---|---|---|---|
| **1 unit** | 660‰ | 340‰ | **1000‰ — ĐÚNG CHẰN** |
| 2 unit | 577‰ | 444‰ | 1021‰ ✗ |
| 3 unit | 487‰ | 475‰ | 962‰ ✗ |

⚠️ **Từ 2 người/phe là hỏng.** Nên `1 − rate` KHÔNG phải tỉ lệ của phe kia, và một số trích từ một slot ở cỡ đội >1 **không đo được gì**.

`TestABothWaysMirrorIsExactlyEven` giữ đúng ca exact — nhưng nó đánh ở `duelSlot` với **một** unit, nên **không với tới** mấy ca kia. `spar` vẫn đúng (1v1). `forge.FightSquads` cộng hai chiều — đúng phương pháp — nhưng triệt tiêu **chỉ được chứng minh ở 1 người/phe**, nên tỉ lệ squad-fight có sai lệch dư.

## Đã loại hai giả thuyết

- **KHÔNG phải bàn cờ.** `hex.Place` là isometry thật, cả chéo phe **lẫn trong cùng phe** — đo 0/81 cặp lệch. ⚠️ `TestPlaceMirrorsBothSides` chỉ kiểm profile **chéo phe**, nên nó *có* lỗ — mà lỗ đó rỗng.
- **KHÔNG phải cấu trúc.** Đội 2 người đang ship (b01) bù nhau **đúng chằn** (1085/915 ↔ 915/1085), còn đội 2 người tự dựng **cùng nhân vật, cùng ô** thì không. Khác duy nhất: **KIT** (b01 dùng 4 chiêu đã chọn; harness dùng `seedKit` = 4 chiêu đầu).

→ Có một chiêu giải quyết theo thứ tự **không mirror**. **Chưa tìm ra.** Đây là việc phải làm TRƯỚC khi trích bất kỳ số nào ở 3v3/5v5.

## Bài học rộng

**Control arm bắt được, review không.** Tôi đã sắp công bố số 3v3/5v5 là kết luận; chỉ có việc hai arm không cộng thành 1000‰ mới chặn lại. Cùng họ với [[hexarena-fester-heal-cut]] (cặp đấu bão hoà không định giá được gì) và [[hexarena-speed-and-measurement]] (số ở một dàn không tự chuyển sang dàn khác) — nhưng đây là cấp trên: **phép đo chưa tự chứng minh thì không kết luận được gì**, kể cả khi cơ chế chạy đúng.

Và: **số duel không chuyển sang trận nhiều người.** Slot đáng +19.6%..+62% ở 1v1; cùng cách đọc ở đội 2 người là **+8.5 điểm**. Xem [[hexarena-side-is-worth-60-points]].

Liên quan: [[hexarena-pvp-plan]], [[fixture-hidden-branch]].
