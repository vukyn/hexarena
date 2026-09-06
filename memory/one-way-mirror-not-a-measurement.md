---
name: one-way-mirror-not-a-measurement
description: "hexarena — ĐÃ TÌM RA VÀ VÁ 2026-09-07: aims duyệt theo thứ tự ô TUYỆT ĐỐI, mà Place xoay 180° nên hai nửa bàn nhận ứng viên NGƯỢC nhau; giờ duyệt theo slot"
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

## ĐÃ TÌM RA (2026-09-07) — và không phải KIT

⚠️ **Thủ phạm là THỨ TỰ NHẮM.** `battle.aims` duyệt ứng viên bằng `hex.Cells()` — column-major trên **cả bàn**, toạ độ tuyệt đối — trong khi `hex.Place` đặt phe địch bằng **xoay 180°**. Xoay đảo hàng, vòng duyệt column-major thì không, nên hai nửa bàn được mời ứng viên theo thứ tự **ngược nhau**. `Suggest.take` giữ **cái đầu tiên** đạt giá trị cao nhất → hoà giá giữa hai mục tiêu y hệt rơi vào unit khác nhau tuỳ nửa nào đang hỏi.

Ví dụ đo được: slot `{1,1}` và `{1,0}` → phe ta ở ô `1,0` rồi `1,1`; phe địch ở `4,1` rồi `4,2`. Duyệt gặp `1,0` trước ở nửa này và `4,1` trước ở nửa kia — tức **u1 trước** một bên, **u0 trước** bên kia.

**Vá**: `battle.mirroredOrder` duyệt theo **slot tác giả**, nửa đối phương trước. ⚠️ Nửa đối phương đi trước chứ không phải nửa mình: chỉ chiêu all-sided phân biệt được, và để nửa mình trước là dời hoà giá của mọi chiêu đó về phía mình — đổi cân bằng khoác áo sửa determinism.

**Sau khi vá**: 1000‰ ở **cả 1, 2 và 3 unit/phe**, và **200/200 seed lật** kết quả khi đổi thứ tự liệt kê (trước: 200/200, 176/200, 134/200).

⚠️ **Giả thuyết KIT ở dưới là SAI.** Đội 2 người ship bù nhau đúng chằn không phải vì kit, mà vì kit đó không tạo hoà giá ở chỗ thứ tự khác nhau — cùng một lỗi, chỉ là fixture không chạm tới. Giữ lại đoạn dưới vì nó ghi lại hai giả thuyết đã loại đúng (bàn cờ, cấu trúc).

⚠️ **Giá phải trả, đo được**: fixture định giá slot cleanser đọc 526→485‰ và 478→460‰ (sàn 450), và 1/600 trận chạm turn cap. Quét 600 seed hai chiều: engine đứng **4 lần có vá, 1 lần không vá** — và lần đó ở seed **307**, ngoài cửa sổ 300 seed test chạy. Vạch "không được đứng trận nào" là tính chất của **cửa sổ seed**, không phải của engine → giờ là `stallShare` 10‰, khai báo một chỗ cho cả 5 fixture.

### Ghi chép cũ, giữ lại vì hai giả thuyết bị loại vẫn đúng

## Bài học rộng

**Control arm bắt được, review không.** Tôi đã sắp công bố số 3v3/5v5 là kết luận; chỉ có việc hai arm không cộng thành 1000‰ mới chặn lại. Cùng họ với [[hexarena-fester-heal-cut]] (cặp đấu bão hoà không định giá được gì) và [[hexarena-speed-and-measurement]] (số ở một dàn không tự chuyển sang dàn khác) — nhưng đây là cấp trên: **phép đo chưa tự chứng minh thì không kết luận được gì**, kể cả khi cơ chế chạy đúng.

Và: **số duel không chuyển sang trận nhiều người.** Slot đáng +19.6%..+62% ở 1v1; cùng cách đọc ở đội 2 người là **+8.5 điểm**. Xem [[hexarena-side-is-worth-60-points]].

Liên quan: [[hexarena-pvp-plan]], [[fixture-hidden-branch]].
