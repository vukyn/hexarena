---
name: hexarena-starter-squads
description: "hexarena 4 đội starter 3 người, tất cả SẴN SÀNG DÙNG; ⚠️ 3 bản CÙNG nhân vật là đội YẾU NHẤT (~11%) — ngược giả thuyết TODO; mọi chiêu điện chỉ Magnezone mang được"
metadata: 
  node_type: memory
  type: project
  modified: 2026-09-05T00:00:00.000Z
---

`internal/seed/data/squads.json` — **4 đội 3 người**, 10/12 nhân vật, 11 lối build khác nhau (`machop.gamble` dùng ở hai đội), mọi loadout lấy NGUYÊN từ `builds.json`. **Cả 4 đều dùng được ngay**: mỗi đội qua `Squad.Validate` + `Take`, đúng cỡ 3v3, `stage` là leaf ở cấp 60. PR #268 (`f378cc5`, 2026-09-03) dựng s01–s03; s04 thêm sau (2026-09-05).

| đội | trước → sau |
|---|---|
s01 áp sát | machamp `gamble` · poliwrath `flurry` · clefable `mend` |
s02 rải độc | blastoise `fortress` · gengar `miasma` · magnezone `surge` |
s03 nhiễm điện | venusaur `parasite` · magnezone `trickle` (spark) · charizard `scorch` |
s04 xuyên phá | machamp `gamble` · mew `wither` · **mewtwo `breach`** |

**Why:** file cũ có 2 đội × **2 unit** → phòng chỉ 3v3/5v5 nên **join luôn bị `squad_refused`** trước khi ai dựng gì. Hai lý do bị từ chối: sai số người **và** thiếu `stage` (không phải leaf ở cấp 60).

**How to apply:**

⚠️ **3 bản CÙNG một nhân vật là đội YẾU NHẤT, không phải mạnh nhất.** `TODO.md` ghi "một đội được xếp cùng nhân vật 2 lần" là quyết **yes**, kèm ghi chú phép đo phản biện *"hai bản của một nhân vật là đội mạnh nhất"* **chưa ai lấy**. Lấy rồi: 3 Magnezone trên 3 build magnemite = **~11%**, thua s01 **200/200**. Ngược hẳn.

⚠️ **Mọi chiêu điện giới hạn "electric units, out of pokemon"** → **Magnezone là con DUY NHẤT** ship mang được. Bộ điện có đúng một carrier; không dựng được đội điện nhiều người khác nhau.

**Số đo (200 seed/cặp, CẢ HAI chiều — phe đáng rất nhiều):**

| | vs s01 | vs s02 | vs s03 | vs s04 |
|---|---|---|---|---|
s01 nhà | — | 34.5% | 6.0% | 9.5% |
s02 nhà | 80.5% | — | 10.0% | 39.5% |
s03 nhà | 94.5% | **45.0%** | — | 48.7% |
s04 nhà | 90.5% | 60.4% | **51.2%** | — |

s03 mạnh nhất, s01 yếu nhất; **s04 là đội cân nhất bộ** — 51.2% với s03 và 60.4% với s02, tức hai cặp sát nhau nhất giờ đều là cặp của nó. Mirror `s04 vs s04` ra **đúng 500‰** (control).

⚠️ **Cột `vs s04` được ĐO, không lấy `1000 − rate`.** Ở 3v3 hai arm không bảo đảm cộng thành 1000‰ (đo được 962‰) → [[one-way-mirror-not-a-measurement]]. Cặp này khớp trong 1‰ — **là quan sát của đúng cặp này, không phải giấy phép trừ** cho cặp sau.

⚠️ **KHÔNG có endless mới là tiêu chí chọn s04, không phải win rate.** `Tally.Rate()` **bỏ trận `Endless` khỏi mẫu số** (`spar.go:109`), nên một đội hoà mãi vẫn khoe tỉ lệ đẹp trên vài trận phân định — đúng cái bẫy đã suýt ship: bản đầu (`machamp charge + mew feed + breach`) đọc 84.7% với s02 mà **229/400 trận không kết**, một arm 191/200. Luôn in `Total().Endless` bên cạnh `Rate()`.

⚠️ **Cỗ máy hoà là HEALER GẶP HEALER, và s02 là nguồn**: `blastoise fortress` (`aqua_ring` + `withdraw` + `taunt`) không giết ai và không chết. Đo endless/300 với s02: mọi biến thể mang healer đều tắc — `fortress+mend+breach` **254**, `parasite+mend+breach` **295**, `charge+mend+breach` **175**, `parasite+wither+breach` **57** — còn s04 (không healer) **0/900**. ⚠️ Baseline: **s02 vs s03 đã hoà sẵn 140/300** trước khi có s04, nên đây là tính chất của s02 chứ không phải lỗi đội mới.

⚠️ **s04 là mắt xích cân bằng, KHÔNG phải đội mạnh nhất — và đó là tiêu chí thứ hai.** Mewtwo `breach` gánh đội tới mức muốn bao nhiêu cũng được: trong 13 cấu hình đo quanh nó, `fortress + mend + breach` ra **mean 934‰** (thắng s01 và s03 **1000‰**) và `parasite + wither + breach` ra **1000/1000/900** — ship thẳng cái mạnh nhất là xoá cả dải của ba đội cũ.

⚠️ **`mewtwo.origin` CHẾT trên đội hình**: 0‰ trước cả s02 lẫn s03 ở hai cấu hình khác nhau (mean 175‰ và 281‰). Đúng ý note builds — origin là kit của Mew đặt lên cái thân mỏng hơn — nhưng số đo mới là thứ đóng cửa: đừng dựng đội quanh nó.

⚠️ **Đổi chỗ Mewtwo ra hàng giữa là hỏng đội**: cùng bộ ba, breach ở col 1 thay col 0 rơi 598 → 139‰. Mewtwo hp max 3000 / def max 280 (mỏng nhất cast) nên **phải đứng col 0 sau màn chắn**; ranks tính từ phe kia nên col 0 ở depth 3.

⚠️ **s04 dùng lại `machop.gamble` của s01 — cố ý.** Trùng build không phải trùng đội: cùng con Machamp ấy đứng trước `mew wither` + `mewtwo breach` đọc 90.5/60.4/51.2 trong khi s01 đọc 34.5/6.0/9.5. Biến thể `charge` thay `gamble` (kP) đo 88.0/96.5/31.3 — mạnh hơn với s02 và sụp với s03, lệch hơn bản đã chọn.

⚠️ **15 cấu hình đo, các cần KHÔNG cộng dồn**: `cleffa.hex` thay `mend` → tệ hơn · `machop.sure` thay `gamble` → s01 sụp 3.5% · **hạ pháo s02 (`magnemite.trickle` thay `surge`) → s02 thắng NHIỀU hơn** (không đơn điệu) · `mewtwo.breach` → 86% vượt hạng · `bulbasaur.parasite` (tuyến đầu tự hồi) là thứ GÁNH mọi biến thể chạy được. Cần duy nhất không lấy từ builds.json: **giãn cặp tiền tuyến s01 khỏi hàng giữa** (81%→73%).

**Cách đo đúng**: `battle.RunToEndWith(maxTurns, fight.Suggest)` là vòng chuẩn (`ai.go:676`) — tự viết vòng `Advance/Act/Pass` sẽ sai vì bỏ `prompt.Skipped` và điều kiện là `!b.finished` chứ không phải `prompt == nil`. Gate của phòng = `Squad.Validate` + kích cỡ format + `Take(hex.SideAlly, characters)`.

**Formation**: 3 cột × 3 hàng, **col 2 = TIỀN TUYẾN**, col 0 = hậu (hex.go:42–45 `AllyFrontCol=2`). `slot` là `{"col":N,"row":N}`; `stage` là **tên form** ("Machamp", "Poliwrath").

⚠️ **`data/squads.json` nằm trong 15 file của data digest** → sửa nó **dời digest**, hai máy phải cùng build không thì `data_mismatch`.

Còn 2 nhân vật chưa dùng: **lapras** (không có build nào trong `builds.json` — muốn dùng phải tự viết loadout, phá quy ước "loadout lấy nguyên từ builds.json") và **naruto** (`hidden: true`, cũng 0 build).

Liên quan: [[hexarena-builds-catalogue]] [[hexarena-squad-builder]] [[hexarena-pvp-lobby]] [[hexarena-data-digest]] [[hexarena-roster-placement]]
