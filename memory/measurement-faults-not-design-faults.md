---
name: measurement-faults-not-design-faults
description: "hexarena — sửa engine làm đỏ 3 test balance; cả 3 là LỖI PHÉP ĐO: biên cần độ sâu, tổng phải chuẩn hoá theo lượt, và fixture không chạm tới ca thì xác nhận bản vá chứ không xác nhận tính chất"
metadata:
  type: reference
---

Vá `ENG-012` (splash pattern đi theo khung caster, 2026-09-08) làm đỏ đúng 3 test balance
trong `internal/seed`. Phản xạ đầu tiên là "bản vá làm hỏng cân bằng". **Sai — 2/3 là lỗi
của chính phép đo, có từ trước, chỉ bị bản vá làm lộ ra.**

## 1. Biên (margin) cần nhiều trận hơn mức (level)

`TestAShapeEarnsItsPowerWhereASparCannotSeeIt` đòi "hình dạng đáng ít nhất **30**‰".
Đo trên 600 trận:

| độ sâu | biên |
|---|---|
| 300 seed (600 trận) trước vá | +56 |
| 300 seed sau vá | **+7** ✗ |
| 1200 seed | +42 ✓ |
| 3000 seed | +41 ✓ |

Tỉ lệ đọc trên 600 trận xê dịch ±20‰. **Nên +56 trước khi vá là MAY, không phải bằng
chứng** — cùng một claim đúng, chỉ là dụng cụ quá mỏng để thấy. Claim dạng "đáng ít nhất
N" phải chạy sâu hơn claim dạng "trên sàn X".

## 2. Tổng không so được giữa hai arm có độ dài khác nhau

`TestAStripEarnsItsSlot...` đòi "gỡ regeneration thì hồi máu phải giảm ít nhất một nửa",
đọc trên **TỔNG** máu hồi cả run. Regeneration hồi máu **mỗi lượt**, mà hai arm là hai kit
khác nhau nên trận dài ngắn khác nhau:

```
arm có strip:   1.756.857 máu / 48.003 lượt = 36/lượt
arm không:      1.491.828 máu / 32.646 lượt = 45/lượt
```

Tổng thì arm có strip **CAO HƠN**, mỗi lượt thì thấp hơn 1/5. Số cũ (0.24) đúng chỉ vì hai
độ dài tình cờ gần nhau.

⚠️ **Assertion nào chứng minh được việc chuẩn hoá?** Không phải hàng regeneration — bỏ
chuẩn hoá vẫn xanh ở đó. Phải là hàng **`withdraw`**, nơi doc đã nói "hai con số hồi máu
bằng nhau" suốt bao lâu mà **không ai assert**: bằng nhau chỉ đúng theo lượt (11 vs 12),
tổng là 343.981 vs 420.984. Mutation bỏ chuẩn hoá → hàng đó đỏ.

## 3. Fixture không chạm tới ca thì xác nhận BẢN VÁ, không xác nhận TÍNH CHẤT

Bản vá aim-order (#196) được chứng minh trên mirror **tổng hợp** với kit `strike`+`sweep`
— **không có chiêu nào splash**. Nó cộng đúng 1000‰ và mọi người tin control arm đã đóng.
Squad ship thì không, và không ai đo lại suốt một tuần.

→ Cùng họ với [[fixture-hidden-branch]]. Test tính chất nên **tự sinh** dữ liệu: book ship
chỉ dùng 4/6 hướng hex, test trên nó sẽ không bao giờ đi `upper_left`/`lower_left`.

## Cái thứ 3 ~~là lỗi thiết kế thật~~ — CŨNG là lỗi phép đo (sửa 2026-09-09)

`TestACleanserEarnsItsSlot...`: 526/478 → 485/460 (sau vá aim-order) → 429/430 (sau
ENG-012). **Tụt ở mọi lần sửa dụng cụ**, tức nó chưa bao giờ qua sàn trên một dụng cụ
không nịnh nó. Kit của Happiny không bị vá nào đụng (`column` + single-target), Machop đối
chứng cũng vậy — thứ đổi là `bubble` của con tường dùng chung.

⚠️ **Không hạ sàn.** Sàn 450 giữ nguyên trong code, in ra khoản thiếu mỗi lần chạy, và
assert một **vạch sụp đổ** 400 (bản đầu tiên đọc 133).

⚠️⚠️ **Câu "lỗi thiết kế thật" ở trên SAI, và `DAT-011` đóng lại bằng cách chứng minh
điều đó.** Sàn 450 được lấy từ mender (525/543 hồi đó, 601/518 hôm nay) — nhưng kit
mender trong fixture là **hybrid tự pha** hai build của nó, thứ `builds.json` không chở.
Build thật đọc ra:

| kit | slugger | blighter | bruiser |
|---|---|---|---|
| hybrid fixture | 601 | 508 | 518 |
| `cleffa.mend` | **340** | 615 | 864 |
| `cleffa.hex` | **205** | 197 | 180 |

Cleanser bị đo bằng **kit duy nhất nó có** so với con số **không build ship nào tạo ra**.
Trên kit đã catalogue, cleanser là support **ổn định nhất** (407/456/431, dải 49) còn
`cleffa.mend` dải 524 và `cleffa.hex` thua 4/5 → `DAT-013`.

→ **Sàn mà chỉ một kit tự pha vượt được thì không phải sàn.** Trước khi kết luận một nhân
vật dưới sàn, kiểm tra kit của thứ định ra cái sàn ấy có nằm trong catalogue hay không.
Xem thêm [[hexarena-effective-hp-buys-burst-and-loses-attrition]] cho phần đòn bẩy stat.

Liên quan: [[one-way-mirror-not-a-measurement]], [[hexarena-side-is-worth-60-points]].
