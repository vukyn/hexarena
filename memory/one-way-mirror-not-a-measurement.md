---
name: one-way-mirror-not-a-measurement
description: "hexarena — HAI nguyên nhân, cả hai ĐÃ VÁ: aim order (2026-09-07) + splash pattern tuyệt đối (ENG-012, 2026-09-08); giờ 5/5 squad ship cộng đúng 1000‰"
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

## ⚠️ NGUYÊN NHÂN THỨ HAI — ĐÃ VÁ 2026-09-08 (ENG-012)

Vá trên đo trên fixture **tổng hợp** (kit `strike`+`sweep`, không splash). Chạy lại
control arm trên **squad ship**, 2000 seed mỗi arm:

| squad | ally trước | enemy trước | tổng |
|---|---|---|---|
| s01 | 629.5‰ | 370.5‰ | **1000‰ — ĐÚNG CHẰN** |
| s02 | 742.4‰ | 257.6‰ | **1000‰ — ĐÚNG CHẰN** |
| s03 | 614.0‰ | 505.0‰ | 1119‰ ✗ |
| s04 | 697.5‰ | 455.0‰ | 1153‰ ✗ |
| s05 | 582.5‰ | 474.5‰ | 1057‰ ✗ |

**Thủ phạm 2: `pattern.targets` duyệt splash bằng bước cube TUYỆT ĐỐI.**
`arc_up` = `[["up"],["upper_right"]]`, mà `hex.Place` xoay 180° → *up* ánh xạ thành
*down*. Nên cùng một slot tác giả, hai nửa bàn splash trúng hàng xóm **khác nhau**.
s01/s02 đúng chằn chỉ vì kit của chúng không có splash chạm được unit thứ hai.

Cách khoanh vùng (làm lại được): chạy hai arm **từng prompt một**, so `Prompt.Options`
sau khi ánh xạ mirror. Options **giống hệt, cùng thứ tự** — cái khác là **giá trị
rating**: s03 lượt 1 `razor_leaf` được 273 ở arm này, 443 ở arm kia.

⚠️ **Bài học riêng của lần này: vá đúng một nguyên nhân KHÔNG đóng được control arm.**
Note này từng ghi headline "ĐÃ TÌM RA VÀ VÁ" và số 1000‰ — đúng, nhưng chỉ trên fixture
mình tự dựng. Fixture tổng hợp không chạm tới splash, nên nó xác nhận bản vá mà **không**
xác nhận tính chất. Muốn kết luận "control arm đã đóng" thì phải đo trên **dữ liệu ship**.

**Bản vá**: `pattern.targets` đi bằng **khung của người ra chiêu**, bọc qua `hex.Place`
(tự nghịch đảo): xoay aim vào khung caster → đi bước tuyệt đối → xoay ra. Phe ally ra
chiêu thì **y hệt như cũ từng byte** (`TestAnAllysShapeIsUnchangedByTheFrame` giữ điều đó
so với bản chép lại của walk cũ), phe enemy là ảnh phản chiếu **do cấu trúc**, không phải
do một bảng tên hướng đảo mà ai đó phải giữ cho đúng. Sau vá: **cả 5 squad ship cộng đúng
1000‰**.

⚠️ Chỉ **16/151 chiêu** bị đổi: `arc_up` 8, `flank_up` 6, `wedge_right` 1, `arc_down` 1.
`single` (114) và `column` (21) bất biến dưới xoay 180° — `column` = {up, down}, xoay đổi
chỗ hai cái, ra cùng tập hợp.

⚠️ **Test tính chất phải TỰ SINH hình, đừng đọc book ship.** Book chỉ dùng 4/6 hướng, nên
test trên nó không bao giờ đi `upper_left`/`lower_left` — đúng hai hướng mà `pierce` của
phe enemy rơi vào sau khi vá.

⚠️ Vẫn giữ: **mọi số phải đọc là KHOẢNG CÁCH giữa hai arm**, không phải tỉ lệ một arm —
đó là phương pháp, không phải cách vá lỗi. `forge.FightSquads` cộng hai chiều.
Xem [[hexarena-side-is-worth-60-points]].

## ⚠️ Giá phải trả: 3 test balance đỏ, cả 3 là LỖI PHÉP ĐO chứ không phải lỗi thiết kế

| test | vỡ thế nào | hoá ra là gì |
|---|---|---|
| shape earns its power | biên +56 → +7 | **độ sâu**: +42 và +41 ở 2400 và 6000 trận |
| strip earns its slot | tường hồi máu NHIỀU hơn khi bị strip | con số là **TỔNG**, không chuẩn hoá theo số lượt |
| cleanser earns its slot | 485/460 → 429/430 | thật, và đã tụt ở mọi lần sửa dụng cụ → DAT-011 |

⚠️ **Biên (margin) cần nhiều trận hơn mức (level).** Tỉ lệ đọc trên 600 trận xê dịch ±20‰,
nên đòi "đáng ít nhất 30" ở độ sâu đó là đọc nhiễu. +56 trước khi vá là may, không phải
bằng chứng.

⚠️ **Regeneration là máu MỖI LƯỢT, nên so tổng giữa hai arm là so độ dài trận.** Arm có
strip chạy 48.003 lượt so với 32.646 → tổng hồi máu cao hơn trong khi mỗi lượt thấp hơn
1/5. Đọc theo lượt: **25 so với 35**. "Ít nhất một nửa" chưa bao giờ được chứng minh.
Và hàng `withdraw` giờ phải **bằng nhau** — chỉ bằng nhau khi tính theo lượt (11 vs 12,
trong khi tổng là 343.981 vs 420.984) — đó là assertion khiến việc chuẩn hoá có thể chứng
minh bằng mutation.

⚠️ **Giả thuyết KIT ở dưới là SAI.** Đội 2 người ship bù nhau đúng chằn không phải vì kit, mà vì kit đó không tạo hoà giá ở chỗ thứ tự khác nhau — cùng một lỗi, chỉ là fixture không chạm tới. Giữ lại đoạn dưới vì nó ghi lại hai giả thuyết đã loại đúng (bàn cờ, cấu trúc).

⚠️ **Giá phải trả, đo được**: fixture định giá slot cleanser đọc 526→485‰ và 478→460‰ (sàn 450), và 1/600 trận chạm turn cap. Quét 600 seed hai chiều: engine đứng **4 lần có vá, 1 lần không vá** — và lần đó ở seed **307**, ngoài cửa sổ 300 seed test chạy. Vạch "không được đứng trận nào" là tính chất của **cửa sổ seed**, không phải của engine → giờ là `stallShare` 10‰, khai báo một chỗ cho cả 5 fixture.

### Ghi chép cũ, giữ lại vì hai giả thuyết bị loại vẫn đúng

## Bài học rộng

**Control arm bắt được, review không.** Tôi đã sắp công bố số 3v3/5v5 là kết luận; chỉ có việc hai arm không cộng thành 1000‰ mới chặn lại. Cùng họ với [[hexarena-fester-heal-cut]] (cặp đấu bão hoà không định giá được gì) và [[hexarena-speed-and-measurement]] (số ở một dàn không tự chuyển sang dàn khác) — nhưng đây là cấp trên: **phép đo chưa tự chứng minh thì không kết luận được gì**, kể cả khi cơ chế chạy đúng.

Và: **số duel không chuyển sang trận nhiều người.** Slot đáng +19.6%..+62% ở 1v1; cùng cách đọc ở đội 2 người là **+8.5 điểm**. Xem [[hexarena-side-is-worth-60-points]].

Liên quan: [[hexarena-pvp-plan]], [[fixture-hidden-branch]].
