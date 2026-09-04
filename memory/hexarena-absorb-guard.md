---
name: hexarena-absorb-guard
description: "hexarena PR#217 — category `absorb` (pool máu trừ dần) BÊN CẠNH `shield` (charge huỷ trọn nhát), cờ `unblockable`, pierce 1000; và bẫy 'test định giá 1 hàng không thấy clamp thiếu'"
metadata: 
  node_type: memory
  type: project
  modified: 2026-09-01T18:15:17.604Z
---

PR#217 (merged `ef09638`, 2026-09-01). Category thứ **10**: `status.Absorb`. Xem [[hexarena-dispel-and-gengar]] · [[hexarena-core-design]].

## ⚠️ `absorb` KHÔNG phải `shield` mượt hơn — hình dạng NGƯỢC

| | `shield` (`block`) | `absorb` (`aegis`) |
|---|---|---|
| stack là | **charge**, đếm | **pool**, tính bằng máu |
| chặn gì | **một nhát TRỌN** | **chừng ấy sát thương** |
| 1 cú nặng | huỷ sạch | ăn được bao nhiêu, **dư thì tràn** |
| 3 nhát nhỏ | mỗi nhát 1 charge | cùng tổng, cùng bào mòn |

Đo cùng tổng: tường 2 charge → nặng **0**, tách 3 → **100**. Giáp ảo → **100 / 100**.
Tường charge bị multi-hit khắc (định danh `warden` ↔ `bruiser`); pool thì **không quan tâm đòn tới kiểu gì**. Game chỉ có 1 trong 2 = chỉ có 1 kiểu tường.

**3 hệ quả, mỗi cái 1 test:**
- **Nhát bị pool ăn vẫn ĐÃ TRÚNG** → rider vẫn dính, drain vẫn hút phần vào máu, `Count(attempts, Struck)` vẫn đếm. Nhát bị `block` thì KHÔNG (nó không xảy ra). Đây là **phần bù** của `OutlastsAShield`, không phải trường hợp của nó.
- **Pool TRÀN**: giáp còn 1 điểm ăn 1 của cú 25 → lọt 24. Charge còn 1 thì xoá sạch cú đó.
- **`unblockable` đi QUA giáp, không tiêu giáp** — người sau vẫn gặp giáp đầy. Tiêu luôn thì 1 chiêu thành chiêu tước mạnh nhất game.

## Chỗ tách ra, và LÝ DO

- **`Stack.Pool` là field RIÊNG**, không dùng `TickAmount`: nó là đại lượng per-stack **duy nhất đi xuống**. `TickAmount` bị đóng băng rồi **nhân với số lượt còn lại** (`Tick` tính tiền, `Pending` cộng tổng) → dùng chung sẽ khiến giáp **tick lên chính người mang bằng độ dày của nó**. Cùng lý lẽ cho `Kind.PoolPower` (parser từ chối `tick_power` trên thứ không tick).
- **`combat.Absorb` là lượt quét SAU `Roll`, không phải nhánh trong**: charge tiêu *trong* roll vì nó quyết nhát đánh có xảy ra không; pool tiêu *sau* vì chỉ giảm thứ đã trúng, mà crit phải roll xong mới có số để ăn. ⇒ **chữ ký `Roll` không đổi** (1 caller prod + 18 caller test không phải sờ), `combat` vẫn thuần, replay vẫn chạy.
- **`Set.SpendPool` rút stack CŨ TRƯỚC** — thứ tự là hợp đồng replay, không phải sở thích.
- **`skill.Unblockable` là CÔNG TẮC còn `Pierce` là TỈ LỆ**: giáp liên tục nên tỉ lệ đi dọc mép, khiên **rời rạc** — "nửa cái block charge" không tồn tại. Viết ở `battle` thành *không có gì để tiêu* (charges = 0 + bỏ qua pool) ⇒ `combat.Roll` không cần cờ nào.

## ⚠️⚠️ Bẫy: TEST ĐỊNH GIÁ MỘT HÀNG KHÔNG THẤY ĐƯỢC CLAMP THIẾU

`guarded` bản đầu **nhân pool với số stack** → rating thấy đáng dựng giáp lên con **đã có giáp**, mỗi lượt, mãi mãi (đúng lỗi mà clamp "2 con buff nhau vô tận" sinh ra để chặn). Sửa = đi qua **`Set.With`** như `shielded`/`standing`, đọc **HIỆU** — thế thì trần stack, refresh, stack thừa đều là thứ `Apply` xử lý thật.

Bắt được vì test hỏi rating **2 lần**: chưa có gì / **đã chạm trần**. Chỉ hàng 2 đỏ. Nối tiếp bài học [[hexarena-dispel-and-gengar]]: **giáp không ai định giá thì chỉ là data**.

## Ship kèm

`aegis` "lá chắn" (pool 1800‰ công, 2 stack, 3 lượt) · `light_screen` "màn sáng" cho **ĐỒNG ĐỘI** (Cleffa — nửa việc `withdraw` không làm được) · `shadow_punch` "quyền xuyên" (Gengar): **`pierce: 1000`** → `Pierced` trả về **đúng 0 giáp**, cộng `unblockable`.

⚠️ **`pierce: 1000` chính là công tắc mà comment của `Pierced` phản đối** (công tắc làm con giáp cao vô dụng trước chiêu này, miễn nhiễm chiêu kia). Ship có chủ đích, **trả giá chỗ khác**: 900 power / cd 4 = 1/3 `cross_chop` mỗi lượt.

⚠️ **Không chiêu nào vào build** — build là khẳng định ĐÃ ĐO về 4 chiêu + 1 trait; đây mới là công cụ trong learnset. Nhưng đã đo **nó chạy thật** trước (60 trận đội: 354 lần dựng giáp, ăn 128.329 sát thương qua 446 nhát).

## Việc phải làm khi thêm 1 CATEGORY (checklist)

1. `status.go`: enum **cuối cùng** + `CategoryCount` + `categoryNames`; `Harmful()`; validate `Kind` (field bắt buộc + cấm permanent).
2. `status_test.go`: `TestCategoryNames` giữ **danh sách theo thứ tự khai** — phải nối vào CUỐI.
3. i18n **4 chỗ**: `keys.go` + `english.go` + `vietnamese.go` + map trong `forge.go` — và **HAI họ**: `StatusCategory` (vị ngữ) VÀ `StatusCategoryNoun` (danh từ). Test bắt cả hai, và bắt cả "noun trùng enum spelling của BẤT KỲ category nào".
4. `describe.go`: thêm nhánh switch, không thì status tự mô tả thành **rỗng**.
5. `gloss.go` `statusGloss`: id status mới bắt buộc có tên Việt.
6. Event kind mới ⇒ `internal/tui/tui.go` phải có `case`, VÀ `TestEveryEventKindIsReachable` đòi **một trận trên bench thật sự phát ra nó** → phải viết màn chơi tay.
   ⚠️ Màn chơi tay đầu tiên hỏng vì **tốc độ**: giáp sống 3 lượt CỦA NGƯỜI MANG, nên con giáp NHANH (spd 200) dựng xong là hết hạn trước khi con chậm (spd 10) kịp đánh. Phải cho con mang giáp **CHẬM hơn**, và 2 tốc độ **gần nhau** (100 vs 120) không thì trận kết thúc trước.
7. `replay.golden` **CÓ dịch**: nó in bảng đếm mọi event kind → thêm 1 dòng `absorbed 0`. Không phải trận đấu đổi.
8. `scenarios.golden` **KHÔNG tự dịch** (phần status chỉ đo poison) — nhưng nên **tự thêm một mục** cho cơ chế mới; đó là chỗ ghi đường cong.
   ⚠️ Generator bác bỏ lời tôi viết: 2 cột lệch **2** chứ không bằng nhau — đó là **truncation của multi-hit mất TRƯỚC khi có giáp nào nhìn thấy** (3×205=615 vs 1×617), không phải giáp xử lý khác nhau.
9. Fixture book có **2 bản riêng**: `internal/testfixture/fixture.go` (cho `internal/seed`) và JSON inline trong `internal/core/battle/battle_test.go`. Thêm chiêu/status thử nghiệm phải sửa **cả hai**.
