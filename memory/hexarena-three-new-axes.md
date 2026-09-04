---
name: hexarena-three-new-axes
description: "hexarena PR#219 — Grant{Power,Scaling} (giáp ảo từ trait), Passive.Converts (quy đổi ≠ pierce), Skill.Cost (trả máu); ⚠️ 2 bug định giá: cost nằm trong nhánh không chạy, và clamp làm chiêu rẻ nhất lúc chí mạng nhất"
metadata: 
  node_type: memory
  type: project
  modified: 2026-09-01T19:25:51.591Z
---

PR#219 (merged `9e142bf`, 2026-09-01). Ba trục mới + 1 chiêu. Xem [[hexarena-absorb-guard]].

```
carapace     vỏ cứng   giáp ảo = 900‰ THỦ      -> Squirtle
projection   hộ thể    giáp ảo = 900‰ CÔNG     -> Charmander
rending      xé giáp   250‰ mỗi đòn đi thẳng   -> Machop
steel_beam   trụ thép  360% công, tốn 15% máu  -> Magnemite
```

## 1. `Grant` phải mang SỐ (`Power` + `Scaling`)

Mọi grant trước đó là **công tắc** (`toughened` = status có modifier tự nói nó làm gì). Giáp ảo là **pool** — sâu bao nhiêu chính là toàn bộ nội dung của trait.
⚠️ **`Set.Hold` luôn truyền `Apply(kind, 0)`** với lý lẽ "status vĩnh viễn không tick nên không có gì snapshot". Đúng với MỌI status vĩnh viễn từng có, và **hết đúng đúng lúc giáp ảo được phép vĩnh viễn** → giáp grant kiểu đó parse được, apply được, hiện trong log, và **chặn 0**.
⚠️ **Luật cấm permanent-absorb của tôi ở PR#217 đặt SAI CHỖ.** "Guard with no clock" sai về pool — đồng hồ của pool là lượng máu trong nó. Nguy hiểm thật là **CỔNG**: `hold`/`release` chạy lại grant mỗi lần cổng mở → giáp sau `below_health` đầy lại mỗi lần chủ băng qua vạch. Luật dời sang `passive.ParseBook`, chỗ nhìn thấy cổng.
- Pool đọc dòng **BASE**, không phải buffed: grant lên lần lượt từng trait lúc enlist, đọc buffed thì đáp án phụ thuộc **thứ tự khai trait**.
- Stat parse qua `skill.ParseScaling` (kèm luôn luật cấm scale theo máu).
- `Set.Hold(kind, amount, stacks)` — đổi chữ ký, 3 caller prod + 2 file test.

## ⚠️ 2. Pool đáng nhất ở KHUNG MỎNG → bản scale theo THỦ tự phản mình

Pool là lượng **phẳng**, nên nó đáng bao nhiêu = tỉ lệ so với thứ chủ nó vốn đã chịu được. Công **không** tương quan với khả năng sống; **thủ thì có** → giáp scale theo thủ trao pool sâu nhất cho kẻ cần ít nhất, và trên tường nó thua `endurance` (mà `endurance` đáng giá chính vì thủ đã cao).

| | trait | spar | trait nó thay |
|---|---|---|---|
| Charizard | `projection` (công) | **71.9%** | blaze 63.2% |
| Blastoise | `carapace` (thủ) | **29.2%** | endurance 32.1% |

Vẫn ship cả hai; **ghi LÝ DO** để lần cân bằng sau không đọc con số thành bug.

## 3. Quy đổi ≠ pierce

**Pierce hạ MẪU SỐ** (giáp nhỏ đi, đòn vẫn bị chia) → càng giáp cao càng lọt ít.
**Convert TÁCH ĐÒN** (phần quy đổi không chia cho gì) → tới nguyên vẹn bất kể giáp, đặt **SÀN CỨNG** giáp không kéo xuống được.

| giáp | pierce 400‰ | convert 400‰ |
|---|---:|---:|
| 0 | 600 | 600 |
| 900 | 214 | 330 |
| 2400 | 103 | **280** |

Trên cast: Machamp vs Blastoise — `rending` **150-0 / 30 lượt**, `blood_thirst` 138-12 / **71 lượt**. Cả hai đều thắng gần hết ⇒ **tỉ lệ nói vô nghĩa, ĐỒNG HỒ nói tất cả**. Và nó **trả giá** chỗ không có giáp: vs Charizard thua 10 trận so với drain.
- Hai nửa tính **KHÔNG SÀN**, sàn áp lên TỔNG — hai lần `damage()` sẽ sàn 2 lần = tặng free 1 điểm.
- Convert = 0 đi **NHÁNH RIÊNG**, không rơi qua công thức với share 1000 (truncate khác) → **không golden nào nhúc nhích**.

## 4. `Skill.Cost` — trả máu (trước đó KHÔNG có gì)

Gần nhất: `Drains` (ngược), `self_applies` (status chứ không phải máu), `Replies` (kẻ TẤN CÔNG mất máu). `modifier.HP` bị từ chối thẳng.
Share của máu **TỐI ĐA**, trả **TRƯỚC**, trả **dù trúng hay trượt** (trừ theo sát thương gây ra thì lượt trượt là miễn phí → chiêu không có quyết định nào). **Không bao giờ lấy điểm cuối** — `Suggest` không có số hạng nào cho "và rồi tôi không còn ở đây".

### ⚠️⚠️ Hai bug định giá, số đo bắt được

**(a) Cost nằm trong `friendlyFire`, mà `rate` chỉ gọi nhánh đó khi `target == skill.All`.** Chiêu đơn mục tiêu tốn 25% máu bị tính **0 đồng**:
| | cast | máu đưa ra | thành tích |
|---|---:|---:|---|
| giá không trừ | **360** | **254.012** | **0-120** |
| kit không có chiêu | 0 | 0 | 69-51 |
→ **Chi phí nằm trong nhánh không chạy là chi phí không ai trả.** Cùng họ với `dispelled` chết ([[hexarena-dispel-and-gengar]]) và `guarded` bỏ trần stack ([[hexarena-absorb-guard]]).

**(b) Đọc giá qua ĐÚNG hàm trả tiền (kèm sàn) → chiêu RẺ NHẤT đúng lúc CHÍ MẠNG NHẤT.** Con 200 máu bị tính 199 thay vì 750 ⇒ rating thích nó nhất đúng chỗ phải từ chối.
`spentHealth` giờ là con số **DUY NHẤT trong price.go cố ý KHÁC** thứ hàm resolve trả. Luật "đọc hàm resolve, đừng chép lại" nói về **số học có thể lệch**; đây là cùng số học bỏ đi một clamp, mà clamp đó là *đủ tiền hay không* chứ không phải *giá bao nhiêu*.

Sửa xong: quyết định phụ thuộc bàn cờ — **30** cast vs Blastoise (giáp 640), **129** vs Charizard (400).

## Bẫy test lại dính (lần 3 trong 3 PR liên tiếp)

Subtest "paid on a miss" **xanh trong khi đo ngược**: đếm `misses` mà **không assert**, trên chiêu fixture acc 1000 — `Rules.Chance` trả `Base` NGAY khi acc ≥ 1000, chưa đọc tới dodge, nên volley trúng hết. Sửa = assert **cả** `misses > 0` **lẫn** `landed == 0`.
→ Luật: **đếm mà không assert = không đo gì.** Cộng với [[hexarena-dispel-and-gengar]] "biên độ lấy từ mutation".

## Ghi chú

- Mỗi lần rebase trong session này (4/4 PR) main đều đã nhích, và lần nào cũng có một golden màn hình **mới dời chỗ** đếm số skill/status. Rebase + `make golden` + đọc diff trước khi merge là bắt buộc.
- `gh pr merge` báo `Base branch was modified` ngay sau force-push — chỉ cần thử lại.
- ⚠️ **Mọi status VĨNH VIỄN hiện `0 lượt`** ở màn kit picker (`toughened`/`unleashed`/`quickened`/`evasive`/`bastion`). Lỗi CÓ SẴN, đúng cái comment của `Snapshot.Remaining` cảnh báo. Chưa sửa.
