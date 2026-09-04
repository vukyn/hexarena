---
name: hexarena-dispel-and-gengar
description: "hexarena PR#215 — sửa `dispelled` (nhánh rating CHẾT: tước 3 thứ chỉ định giá 1) + ship Gengar; và bài học 'biên độ phải lấy từ mutation, không phải từ số đo'"
metadata: 
  node_type: memory
  type: project
  modified: 2026-09-01T17:20:02.229Z
---

PR#215 (merged `f79e62b`, 2026-09-01). Con thứ 11: **`pokemon.gastly`** (Gastly→Haunter→Gengar), hệ `dark` (carrier **thứ 2**, sau [[hexarena-mew-and-mewtwo]]), loài mới `shade` "bóng ma", preset mới `hexer` "kẻ phá phép". Cap 3000/620/340/130/175/80, EffHP 6400, spar **48.4%**.

## ⚠️ `dispelled` là nhánh rating CHẾT — tước được 3 thứ, định giá đúng 1

`battle.Suggest.dispelled` có từ lâu và **chưa lần nào chạy trên data ship**: 2 chiêu tước đang có (`rapid_spin`, `rinse`) đều chĩa vào phe mình → đi qua `cleansed`. Chưa ai chĩa vào địch.

Nó dựng hypothetical rồi so công/thủ → **buff chỉ số** đọc được, **`shield`** và **`regen`** dịch 0 chỉ số nên ra **0**. Đo: con cầm chiêu tước + `jab` chọn `jab` kể cả khi địch có 3 charge khiên / 3 stack hồi / cả hai.

Sửa = thêm 2 số hạng **nghịch đảo của thứ định giá lúc ĐẶT vào**, đọc từ chính hàm đó: `unguarded` = `shielded` ngược (charges × `strikeThreat`, clamp `guardHorizon`); `undone` = chênh lệch `Pending()` chạy qua **`worthHealing`** (3 clamp y như một chiêu hồi).

- **Hồi phục trên con ĐẦY MÁU = tước không đáng gì** — rơi ra từ clamp room, không viết tay.
- `undone` **từ chối chạy nếu strip gọi tên category `Harmful()`**: `Pending()` cộng mọi tick không phân biệt lành/hại, nên tước `dot` khỏi địch sẽ bị đọc thành *lợi*.
- **KHÔNG có gì đang ship nhúc nhích** sau khi sửa (suite + mọi golden + mọi tỉ lệ). Đó là dấu hiệu nhánh *chết* chứ không *sai*.

⚠️ **`buff` KHÔNG được nằm trong danh sách tước.** `status.Set.Cleanse` không bỏ qua status vĩnh viễn, và chỉ trait **có cổng** (`While != nil`) mới cấp lại (`Battle.hold` chạy 1 lần lúc enlist). Tước `buff` = gỡ VĨNH VIỄN `toughened` của `endurance`, `evasive` của `elusive`, `quickened` của `swiftness`. Đó là quyết định thiết kế riêng, đừng nhét vào một nhân vật. `spite` chỉ gọi `shield` + `regen`.

## ⚠️⚠️ BIÊN ĐỘ PHẢI LẤY TỪ MUTATION, KHÔNG PHẢI TỪ SỐ ĐO

Dính **2 lần trong cùng một PR**:

1. Test pricing viết với địch **nửa máu** hết. Xoá số hạng khiên đi → **VẪN XANH**: trên con bị thương rating chọn chiêu tước sẵn vì lý do khác. Phải đo trên con **đầy máu** mới đỏ. (Lý do "khác" đó chưa truy ra — ghi lại chứ không đuổi theo.)
2. Test build: 2 kit làm cho **GIỐNG HỆT**, chỉ khác trait, vẫn ra 1300 vs 1178 và 153 vs 81 → `miasma.dealt > unbind.dealt` xanh trên 2 bản sao của cùng 1 build. Vì `contagion` gây weaken → trận dài hơn → đánh nhiều hơn. Sửa = đòi **≥1.8×** (số thật 2.07×, số mutation 1.10×) + check 2 kit **rời nhau**.

**Luật rút ra: một test "hai thứ này khác nhau" mà dùng `>` trần trụi thì gần như luôn xanh sai. Chạy mutation TRƯỚC, lấy biên độ từ khoảng cách giữa số thật và số mutation.**

## Đo chiêu tước: `spar` mù, phải dùng ĐỘI

`spite` tước khiên + hồi phục của địch — trận đơn không ai mang 2 thứ đó → spar đọc nó thành chiêu 700 power. Cùng điểm mù với mender ([[hexarena-poliwag-bruiser]]). Dụng cụ = khuôn của mender: 1 striker + 1 tường + ô đang đo, 2 chiều, 300 trận/hàng, 3 đối thủ chỉ khác **chiêu thứ 4 của tường**:

| tường địch | kit | tước | đòn bị chặn | tường hồi |
|---|---|---:|---:|---:|
| có khiên (`withdraw`) | có `spite` | 508 | **1541** | 270722 |
| | không | 0 | **2434** | 276988 |
| chỉ đánh | có `spite` | **0** | 0 | 0 |
| có hồi (`aqua_ring`) | có `spite` | 589 | 0 | **297579** |
| | không | 0 | 0 | **899834** |

⚠️ **Tỉ lệ thắng KHÔNG phải thứ đọc** (500→540‰). Thứ đội đo chính xác là chiêu ấy *làm gì*.
⚠️ **Hàng giữa làm 2 hàng kia có nghĩa** — không có gì để tước thì tước 0, tỉ lệ nhích NGƯỢC lại.
⚠️ **`restores` ≠ `regen`**: `withdraw` hồi ngay lúc cast → tước không lại được; chỉ tick mới tước được. Đó là lý do 2 số hồi ở hàng khiên bằng nhau.
⚠️ Event `StatusStripped` **không mang status id** → không phân biệt được tước khiên hay tước hồi từ log; phải đọc gián tiếp qua `Blocked` / `Healed`.
⚠️ Unit id trong squad có tiền tố phe: `ally.wall` / `enemy.third` → lọc theo `strings.HasPrefix`.

## ⚠️ Tốc độ là đồng tiền — xác nhận lần 2

Gengar bản đầu **0.0% trước TẤT CẢ 11 đối thủ**. Vặn từng cần:
| cần | |
|---|---|
| thứ tự learnset (spar đọc 4 chiêu đầu) | 0.0 → 18.9 |
| atk 480→720 | 8.7 → 34.9 |
| **spd 125→140→155** | 16.2 → 33.8 → **44.1** |
| **trait `elusive`→`blood_thirst`** | 22.0 → **48.1** |

30 điểm tốc = 28 điểm tỉ lệ (khớp [[hexarena-mew-and-mewtwo]]). **`blood_thirst` đáng 26 điểm ở đây** vs ~8 trên Machop — hút máu đáng tiền nhất trên khung MỎNG đánh NHIỀU. Gengar cố ý **không nhanh nhất** (130 vs Mewtwo/Charizard 140), mua tỉ lệ bằng hút máu.

## Carrier thứ 2 của một hệ

Gengar lấy nguyên `shadow_ball`/`dark_pulse`/`nasty_plot` của Mewtwo + 4 chiêu riêng (`shadow_claw`/`spite`/`night_shade`/`curse`); Mewtwo giữ `psystrike`/`psycho_cut`. `TestTheSecondCarrierOfAnElementSharesItsLine` đòi **vừa có chiêu chung vừa có chiêu riêng** — một hệ mà mọi chiêu thuộc về 1 người là "một nhân vật có từ riêng cho kit của nó".

**Hai con dark = hai đáp án cho cùng một bài (bức tường)**: Mewtwo *xuyên* giáp (pierce 800), Gengar *gỡ* giáp.

Build: `gastly.unbind` "gỡ chỗ dựa" (spite curse shadow_claw night_shade + elusive) · `gastly.miasma` "chướng khí" (poison_powder sludge_bomb venoshock shadow_ball + contagion). 23 build / 10 trong 11 con.
