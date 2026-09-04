---
name: hexarena-starter-squads
description: "hexarena PR#268 3 đội starter 3 người; ⚠️ 3 bản CÙNG nhân vật là đội YẾU NHẤT (~11%) — ngược giả thuyết TODO; mọi chiêu điện chỉ Magnezone mang được"
metadata: 
  node_type: memory
  type: project
  modified: 2026-09-03T16:36:54.270Z
---

hexarena PR #268 (`f378cc5`, 2026-09-03): `internal/seed/data/squads.json` — **3 đội 3 người**, 8/12 nhân vật, 9 lối build, mọi loadout lấy NGUYÊN từ `builds.json`.

| đội | trước → sau |
|---|---|
s01 áp sát | machamp `gamble` · poliwrath `flurry` · clefable `mend` |
s02 rải độc | blastoise `fortress` · gengar `miasma` · magnezone `surge` |
s03 nhiễm điện | venusaur `parasite` · magnezone `trickle` (spark) · charizard `scorch` |

**Why:** file cũ có 2 đội × **2 unit** → phòng chỉ 3v3/5v5 nên **join luôn bị `squad_refused`** trước khi ai dựng gì. Hai lý do bị từ chối: sai số người **và** thiếu `stage` (không phải leaf ở cấp 60).

**How to apply:**

⚠️ **3 bản CÙNG một nhân vật là đội YẾU NHẤT, không phải mạnh nhất.** `TODO.md` ghi "một đội được xếp cùng nhân vật 2 lần" là quyết **yes**, kèm ghi chú phép đo phản biện *"hai bản của một nhân vật là đội mạnh nhất"* **chưa ai lấy**. Lấy rồi: 3 Magnezone trên 3 build magnemite = **~11%**, thua s01 **200/200**. Ngược hẳn.

⚠️ **Mọi chiêu điện giới hạn "electric units, out of pokemon"** → **Magnezone là con DUY NHẤT** ship mang được. Bộ điện có đúng một carrier; không dựng được đội điện nhiều người khác nhau.

**Số đo (200 seed/cặp, CẢ HAI chiều — phe đáng rất nhiều):**

| | vs s01 | vs s02 | vs s03 |
|---|---|---|---|
s01 nhà | — | 34.5% | 6.0% |
s02 nhà | 80.5% | — | 10.0% |
s03 nhà | 94.5% | **45.0%** | — |

s03 mạnh nhất, s01 yếu nhất, **s03 vs s02 là cặp sát nhau nhất**.

⚠️ **15 cấu hình đo, các cần KHÔNG cộng dồn**: `cleffa.hex` thay `mend` → tệ hơn · `machop.sure` thay `gamble` → s01 sụp 3.5% · **hạ pháo s02 (`magnemite.trickle` thay `surge`) → s02 thắng NHIỀU hơn** (không đơn điệu) · `mewtwo.breach` → 86% vượt hạng · `bulbasaur.parasite` (tuyến đầu tự hồi) là thứ GÁNH mọi biến thể chạy được. Cần duy nhất không lấy từ builds.json: **giãn cặp tiền tuyến s01 khỏi hàng giữa** (81%→73%).

**Cách đo đúng**: `battle.RunToEndWith(maxTurns, fight.Suggest)` là vòng chuẩn (`ai.go:676`) — tự viết vòng `Advance/Act/Pass` sẽ sai vì bỏ `prompt.Skipped` và điều kiện là `!b.finished` chứ không phải `prompt == nil`. Gate của phòng = `Squad.Validate` + kích cỡ format + `Take(hex.SideAlly, characters)`.

**Formation**: 3 cột × 3 hàng, **col 2 = TIỀN TUYẾN**, col 0 = hậu (hex.go:42–45 `AllyFrontCol=2`). `slot` là `{"col":N,"row":N}`; `stage` là **tên form** ("Machamp", "Poliwrath").

⚠️ **`data/squads.json` nằm trong 15 file của data digest** → sửa nó **dời digest**, hai máy phải cùng build không thì `data_mismatch`.

Còn 4 nhân vật chưa dùng: lapras, mew, mewtwo, naruto.

Liên quan: [[hexarena-builds-catalogue]] [[hexarena-squad-builder]] [[hexarena-pvp-lobby]] [[hexarena-data-digest]] [[hexarena-roster-placement]]
