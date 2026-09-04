---
name: hexarena-mew-and-mewtwo
description: "hexarena PR#211 — Mew (hệ trơ neutral) + Mewtwo (dark, nửa còn thiếu của cặp tương khắc); 3 phát hiện về engine: status 1 lượt không làm mồi được, tốc độ là đồng tiền, hạn mức EffHP không ghìm được ai"
metadata: 
  node_type: memory
  type: project
  modified: 2026-09-01T06:30:01.719Z
---

PR#211 (merged `0021235`, 2026-09-01) ship **2 nhân vật cùng lúc** — con thứ 9 và 10. Xem [[hexarena-shipping-a-character]] · [[hexarena-poliwag-bruiser]].

```
pokemon.mew     shifter "kẻ vạn biến"  neutral  mythic  1 dạng  3600/590/490/110/172/49  EffHP 9498  spar 56.4%
pokemon.mewtwo  breaker "kẻ phá giáp"  dark     mythic  1 dạng  3000/520/280/140/200/68  EffHP 5800  spar 68.0%
```
Cast sau: naruto 90.2 · machop 77.8 · **mewtwo 67.2** · charmander 60.6 · bulbasaur 59.8 · **mew 52.3** · poliwag 45.1 · magnemite 38.6 · squirtle 35.6 · cleffa 11.1.

Loài mới `mythic` "huyền thoại" — **loài ĐẦU TIÊN có 2 thành viên** (mọi loài khác gate cho đúng 1 con), nên đây là lần đầu trục species làm đúng việc nó sinh ra để làm. 11 chiêu mới. Cả hai đều là **dòng 1 DẠNG đầu tiên được ship** (`progression.Line` vốn đã resolve được, không sửa engine gì).

## ⚠️ 3 phát hiện về ENGINE (đáng nhớ hơn nhân vật)

**1. Status khống chế 1 lượt KHÔNG làm mồi cho chiêu có cooldown.** `stun` sống 1 lượt mà lượt đó là *của mục tiêu*, nó tiêu ngay bằng cách bị bỏ lượt → cửa sổ rộng đúng **một ô hàng đợi**, ai giữ ô đó do TỐC ĐỘ quyết. Đo 60 trận/hàng: Blastoise (spd 85) 239 choáng → **50** kích hoạt; Charizard (spd 140) 31 choáng → **0**.
⚠️ **Tăng tỉ lệ choáng làm TỆ hơn**: đổi nguồn sang `hypnosis` (choáng gấp đôi) → 9381 choáng đổi được **6** kích hoạt. Lượt đặt status chính là lượt đáng lẽ tiêu nó. Rút cooldown 3→2→1 gần như không đổi gì.
→ Muốn combo thật thì gate lên status sống ≥2 lượt (`expose`/`mire`/`weaken` — đúng cách `venoshock`/`dragon_drive` làm).

**2. Ở tốp trên, TỐC ĐỘ là đồng tiền, GIÁP gần như miễn phí.** Mewtwo bản đầu **98.5%**, vặn từng cần:
| cần | |
|---|---|
| pierce 700→400→0 | 98.4 → 97.0 → **90.1** |
| spd 150→130→110→90 | 98.4 → 91.4 → 79.7 → **69.3** |
| atk 740→620→520→440 | 98.4 → 89.7 → 79.4 → **71.3** |
| hp+def mỏng đi cùng lúc | 98.4 → **96.5** |
60 điểm tốc = **29 điểm win rate**. KHÔNG mâu thuẫn [[hexarena-speed-and-measurement]] — cái đó là mức *trait* (±50/150, nhiễu đội lốt thứ tự); ở đây dải 60 điểm, 4 số đo, đơn điệu sạch.

**3. ⚠️ Hạn mức EffHP KHÔNG ghìm được nhân vật.** Nó chỉ bound khả năng sống sót, mà đó là nửa RẺ. Mewtwo qua cửa ở 5800/11500 (mỏng nhất cast) trong khi đọc 98.5%. **Vặn spd + atk, đừng tin hạn mức.** Xem [[hexarena-stat-bounds-policy]].

## Mew — hệ trơ `neutral`

Hệ trơ có trong chart từ commit đầu, **chưa ai đứng lên**. Nhân 1000 CẢ HAI CHIỀU với mọi hệ. Giá phải trả là **bộ chiêu**: chỉ mang được chiêu trung tính — mà đó là kho lớn nhất và không của riêng ai → **learnset RỘNG NHẤT cast (23 chiêu, magnemite 17)**. `Dual` vẫn từ chối ghép với neutral (đúng).
⚠️ **KHÔNG làm phẳng profile matchup** — Mew vẫn 0%..100%, mọi con trong cast đều trải ~90 điểm. Cái quyết định trận đấu là tempo + hồi máu, chart chỉ là một số hạng trong một cú đánh.

⚠️ **"Cùng tỉ lệ trên mọi TRẦN" ≠ "trung điểm của CAST".** Bản đầu đặt 70% của cả 6 trần (nghe như "toàn 100" trong truyện) → **72.4%**, vì cast dùng trần rất lệch: atk chạm 19/20 trần, **né chưa tới nửa trần**. Trung điểm cast (đo từ các dạng cuối đang ship) → **56.4%**, mà máu + công còn *cao hơn* bản cũ.

## Mewtwo — `dark`, và pierce

`dark` là nửa còn thiếu của **cặp tương khắc DUY NHẤT** (light/dark). Cleffa cầm light một mình → hệ light thực chiến **không khác gì hệ trơ**. Giờ mới fielded.

⚠️ **Cách đo pierce đúng: MỘT người đánh, HAI chiêu.** Đo "3 người đánh 3 mức giáp" là 3 lần đọc chart khác nhau, số hạng hệ nuốt mất thứ đang đo. Cho Mewtwo cầm `psystrike` (pierce 800) + `body_slam` (pierce 0) đánh cùng mục tiêu → số hạng hệ là hằng số, ước đi. Kết quả: qua cả dải giáp cast có (640→340), giáp lấy của đòn xuyên **175‰**, của đòn thường **381‰**.

**`mewtwo.origin` cầm đúng 4 chiêu của Mew trên khung Mewtwo** — phép đo DUY NHẤT trong repo đặt *một loadout lên hai thân*. ⚠️ Bản sao **thua ở mọi cột** (16 lượt/1042/hồi 701 vs Mew 26/1352/1106): restore đọc atk mà atk nó thấp hơn, và kit đó không có gì xuyên giáp. **Chỉ số không phải "nhiều hơn", nó CÓ HƯỚNG.**

## Build (21 build / 9 trong 10 con)

`mew.feed` `mew.borrowed` `mew.wither` · `mewtwo.breach` `mewtwo.origin`. Mew có **3 vì lý do NGƯỢC với magnemite**: magnemite = 3 đáp án cho 1 câu hỏi, Mew không có câu hỏi riêng nên 3 build = 3 nhân vật khác nhau, giữ điều kiện "không 2 build nào tiêu lượt vào cùng một việc" (feed dẫn cột hồi máu, wither dẫn cột status, borrowed dẫn cột sát thương + trượt).
`mew.borrowed` **không cầm gì của Mew** ↔ `mewtwo.origin` **chỉ cầm đồ của Mew** — soi gương nhau.

## Gate mới dính phải

- Preset kit **KHÔNG được chứa chiêu restrict theo `species` hoặc `characters`** (chỉ `origins` là được) → `hexforge check` từ chối thẳng.
- `TestAFlavourClauseSpellsOutNoNumber` bắt chữ **"hai"** trong flavour của `psycho_cut` (countWords soi MỌI chiêu, khớp nguyên từ).
- Fixture `aTraitHolder` (cả `cmd/hexforge-tui` lẫn `internal/screen`) lấy **con giữ NHIỀU TRAIT NHẤT** → cho Mew 7 trait là golden dịch. Đây là fixture tự dựng điều kiện đúng cách, không phải bug.
