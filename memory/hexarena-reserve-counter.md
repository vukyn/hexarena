---
name: hexarena-reserve-counter
description: "hexarena PR#221 — reserve = charge phía MÌNH (sóng nhiệt/xum xuê/ẩm ướt, cap 999); ⚠️ phép chia đôi của charge SAI với reserve (456 stack dồn, 0 tiêu); stack_power clamp vào SỐ STACK không phải power; rung sâu thuộc về con NHANH"
metadata: 
  node_type: memory
  type: project
  modified: 2026-09-01T20:35:36.712Z
---

PR#221 (branch `feat/a-reserve-you-spend`, main lúc mở = `ebec7a0`). Xem [[hexarena-three-new-axes]], [[hexarena-absorb-guard]].

```
swelter   sóng nhiệt  bank_embers → pyre       (toàn bộ, 230/stack)
verdure   xum xuê     sprout      → bloom_burst(10)  · overgrow → đồng minh
moisture  ẩm ướt      soak        → deluge     (toàn bộ, 200) · tide_break(20) · drench → đồng minh
                      flare(1) · thorn_volley(5)
```

## 1. `reserve` = `charge` nhìn từ phía MÌNH

Mọi luật counter đã có sẵn (tally trơ, cap riêng `max_counter_stacks` 999, cấm modifier). Cái CHƯA có là **của ai**:
- charge đặt lên **ĐỊCH** → `Harmful` → phe địch tẩy được, đó là toàn bộ cái giá của nó.
- reserve nằm trên **CHỦ**, mua chiêu của chính chủ qua `self_requires` → KHÔNG harmful; cleanse debuff không đụng, chỉ dispel của địch mới gỡ.

⚠️ Gộp 2 cái là hỏng thấy ngay: `rinse` là cleanse chĩa vào ĐỒNG ĐỘI khai `dot, stat_debuff, charge` → nếu nhiên liệu cùng loại thì chiêu hồi máu đó rút cạn bình xăng.
Chung nhau đúng `Category.Counter()` (3 chỗ hỏi: cap, cấm modifier, định giá). Đổi tên `max_charge_stacks`→`max_counter_stacks` vì 1 field tên theo 1 category mà chặn 2 là lời nói dối ngầm.

## ⚠️⚠️ 2. Phép CHIA ĐÔI của charge SAI với reserve

Lý lẽ của charge: *"consumer tiêu 1 stack mỗi nhát, nên stack thứ 2 cần thêm 1 lượt nữa mới nổ"* → mỗi stack đáng nửa stack trước. **ĐÚNG với conduit, SAI với reserve** — reserve tiêu CẢ CỤM trong 1 nhát, stack thứ 2 nổ cùng lượt với stack đầu.

**Đo được trước khi viết**: để nguyên phép chia đôi, vòng lửa shipped **dồn 456 stack qua 40 trận và tiêu 0**. AI nạp đúng 1 lần rồi thôi; mọi bậc thang sâu là nhánh chết theo cấu trúc.
→ reserve định giá **PHẲNG** tới `pricing.capacity` (nhát tiêu sâu nhất mà KIT của chủ có), trên đó mới chia đôi.

## 3. `stack_power` — và clamp đặt ở đâu

`bonus_power` là số phẳng ⇒ "tiêu toàn bộ" viết bằng nó thì 20 stack trả bằng 2 stack. `stack_power` = anh em phía caster của `arc_power`, nhưng KHÁC arc: nó cộng vào power của chiêu nên **có nhắm, có roll, chia cho giáp, khiên ăn được** — chính vì nó không đọc bàn cờ nên mới được phép nằm phía caster.

⚠️ **`skill.MaxSpendPower` (4×scale.Base) clamp vào SỐ STACK, trong `Takes`, KHÔNG phải vào power.** Clamp vào power ⇒ caster nộp cả đống mà không xài ⇒ bình đầy tiêu ra ít giá trị/stack hơn bình cạn ⇒ rating thích tiêu đúng lúc phí nhất. Đúng cái bug `Skill.Cost` ở PR trước.
`Skill.SelfCeiling` phải có bên cạnh `Satisfying`: Satisfying là ca RẺ NHẤT, mà trả theo stack thì đáng nhất ở ca SÂU NHẤT.

## 4. Bug nó lôi ra

`Battle.spend` gọi `Set.Consume` (rỗng sạch) từ ngày có field ⇒ `consume_stacks` trên `self_requires` parse được, round-trip được, rồi **bị vứt** — tác giả xin bao nhiêu cũng lấy hết. Không ai thấy vì chưa chiêu shipped nào tiêu thứ của chính mình.

## Cân bằng — ⚠️ bậc sâu thuộc về con NHANH

| carrier | cặp reserve | spar | kit 4 đòn |
|---|---|---:|---:|
| Charizard | bank_embers+pyre (all,230) | 60.5% | 68.9% |
| Venusaur | sprout+bloom_burst (10) | **68.3%** | 65.5% |
| Blastoise | soak+deluge (all,200) | 59.7% | 63.4% |
| Poliwrath | soak+tide_break (20) | 59.4% | 58.6% |

20 stack = 4 lượt nạp, và 4 lượt là phần LỚN HƠN NHIỀU trong trận của con chậm: `tide_break` trên Blastoise đọc **7.4%** so với control 12.1%; chuyển sang Poliwrath thì hoà. **Độ sâu của một bậc là giá trả bằng tempo.**

## Bẫy test lại dính (lần 4 liên tiếp)

`TestADispelOfAReserveIsWorthTaking` **XANH khi xoá `unfuelled`** — vì `dispelled` còn đọc "đòn mạnh nhất của địch trước/sau khi tước", và trên bàn mà chiêu tiêu CHÍNH LÀ đòn mạnh nhất thì số hạng đó tự thấy nhiên liệu. Cùng họ với khiên ở PR#215. Sửa = dựng bàn cho chiêu tiêu THẤP HƠN đòn thường của địch ⇒ số hạng đòn-mạnh-nhất không nhúc nhích.

## Ghi chú

- Đổi tên chiêu shipped `heat_wave` "sóng nhiệt"→**"luồng nhiệt"**: chiêu và status trùng tên đọc y hệt nhau trong log ("dùng sóng nhiệt" vs "sóng nhiệt ×3").
- ⚠️ **5/5 PR trong 2 session này main đều nhích, và lần nào cũng có 1 golden màn hình MỚI DỜI CHỖ đếm số skill/status.** Rebase + `make golden` + đọc diff là bắt buộc.
- `hexforge spar --data <dir>` đọc thư mục khác ⇒ đo kit bằng bản sao trong scratchpad, KHÔNG bao giờ `git checkout` (xem [[git-checkout-discards-to-head]]).
- Learnset order là phép đo: spar lấy 4 chiêu + 1 trait ĐẦU TIÊN.
- `--stage` bắt buộc cho dòng RẼ NHÁNH (Poliwag → Poliwrath|Politoed).
