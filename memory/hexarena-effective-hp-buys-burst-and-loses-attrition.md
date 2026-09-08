---
name: hexarena-effective-hp-buys-burst-and-loses-attrition
description: "hexarena — mọi cách mua sức sống cho một support đẩy hai matchup NGƯỢC CHIỀU: thắng thêm ở burst, mất ở attrition, và mua nhiều thì bàn ngừng giải"
metadata:
  type: project
---

**Trên một support không thêm sát thương, effective HP mua thắng ở matchup burst và
mất thắng ở matchup attrition — nên không con số nào vượt được một cái sàn assert
theo TỪNG matchup.**

Đo trên `DAT-011` (Happiny/Blissey, cleanser, 2026-09-09), fixture
`internal/seed/mender_test.go`, 400 seed mỗi phía trừ chỗ ghi khác:

| stat line | effHP | vs slugger | vs blighter | stall |
|---|---:|---:|---:|---:|
| ship, def 220 | 8 320 | 392 | 447 | 0 |
| def 280 | 9 280 | 367 | 485 | 5 |
| def 340 | 10 240 | **524** | 394 | 9 |
| def 400 | 11 200 | **622** | 352 | 16 |
| cả stat line của mender | 9 520 | 473 | 652 | **68/600** |

Cơ chế một câu: **trận dài hơn là thứ phe độc muốn**, và support không thêm sát thương
để kết trận sớm hơn. `effHP = hp × (K+def)/K`, `K = DefenseConstant = 300`.

⚠️ **Mua nhiều thì bàn NGỪNG GIẢI**, và đó là ngưỡng cứng chứ không phải cảm giác:
`stallShare` cho phép 10‰ trận không xong, def 400 ra 40‰ và stat line mender ra 113‰.
Đúng hình dạng `RAT-004` đã vá một lần (guarded mirror), lần này tới qua stat line chứ
không qua rating.

⚠️ **Trần khớp là trần thật, không phải gợi ý**: def 418 bị `progression.Limits` từ chối
tại chỗ — 4800 máu sau 418 giáp hấp thụ 11 510 so với `max_effective_hp` 11 500. Blissey
đã ở **trần hp 4800**, nên cách trần khớp 12 điểm ở def 400. Không có chỗ nào phía trên.

⚠️ **Blissey có thanh máu TO NHẤT mà effHP THẤP NHẤT bàn** — 8 320, dưới cả Machamp
8 360 — vì giáp 220 so với 380–520 của mọi người. Máu thô không phải sức sống; một engine
một-stat-giáp làm sụp "không giáp vật lý, giáp đặc biệt khổng lồ" thành "không giáp".

**Cách dùng.** Đừng sweep stat để cứu một support dưới sàn: đọc hai matchup cùng lúc và
xem dấu. Nếu ngược dấu thì đòn bẩy đóng, và câu hỏi tiếp theo là **cái sàn lấy từ đâu**
→ [[measurement-faults-not-design-faults]]. Đòn bẩy kit và trait cũng đóng theo cách
riêng: [[hexarena-a-cleanse-is-a-matchup-dependent-slot]].

Liên quan: [[hexarena-stat-bounds-policy]], [[hexarena-speed-and-measurement]],
[[hexarena-a-rating-that-prices-nothing-stops-playing]].
