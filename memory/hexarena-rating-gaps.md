---
name: hexarena-rating-gaps
description: "hexarena Suggest: TOÀN BỘ audit XONG (#224 #226 #227 #229 #230 #232 #234 #235), 11/11 category; ⚠️ bài học: dụng cụ đo mù (bàn không xong trận / roster không có giáp) làm 2 lần kết luận sai"
metadata: 
  node_type: memory
  type: project
  modified: 2026-09-01T21:07:46.509Z
---

PR#222 (merged `8ba9228`, 2026-09-02) — **chỉ ghi TODO.md, chưa sửa code**. Đo bằng probe dựng-rồi-xoá, mỗi dòng là lựa chọn `Suggest` THẬT trên bàn fixture, không phải đọc source.

## 4 lỗ định giá (TODO.md § Not done)

| cơ chế | field | rating làm gì |
|---|---|---|
| xuyên khiên | `Skill.Unblockable` | trước 3 `block`: chiêu chặn-được 700 thắng chiêu xuyên 600 |
## Giáp: XONG CẢ HAI NỬA (pool #229 `533503c`, charges #235 `e0058c8`)

`pastAPool` = min(pool, damage). `pastAWall` = charge huỷ **NGUYÊN MỘT NHÁT** (trade của `warden`: tường trả lời đòn nặng đơn, đa-nhát trả lời tường). `unblockable` bỏ qua cả hai.

### ⚠️⚠️ Bài học lớn nhất phiên: **DỤNG CỤ ĐO HỎNG, KHÔNG PHẢI MÔ HÌNH**

Hai lần tôi kết luận "charges không đáng merge" — cả hai lần đo trên bàn **mù**:
- roster shipped **không có mảnh giáp nào** ⇒ ruler đọc y hệt;
- bàn dày tường tôi dựng **không kết thúc nổi trận** ⇒ `Bout` từ chối cả control.

Bàn có tường **MÀ VẪN XONG TRẬN** (mỗi phe 1 `withdraw` + 2 kẻ đánh thật, 900 seed): **889‰ → 917‰**, biên ±24. Ngoài nhiễu.
⇒ **Dựng bàn KẾT THÚC ĐƯỢC trước khi trích bất kỳ con số nào về giáp.**

⚠️ Và **2 giả thuyết về nguyên nhân đều SAI**: `spendable` đọc `strike(mate)` chiết khấu; `ArcPower` chưa định giá lúc xả. Cái sau là fix thật (#230) nhưng **không liên quan**. Tôi đã viết cả hai vào TODO như thể đã biết nguyên nhân.

### ⚠️ Không có núm vặn

Charge huỷ 1 nhát **MỘT LẦN DUY NHẤT**, nên trừ nguyên tường vào **mỗi** cast là tính lặp cùng một mất mát. Over-count đó **có thật và được chấp nhận**: mọi mức chiết khấu đủ nhỏ để giữ claim đều **trong biên độ**, mọi mức đủ lớn để vượt biên độ đều làm vỡ claim. Đơn điệu hai chiều.

### 3 claim ĐỌC LẠI (không bump số)

- **Conduit thiếu vũ khí cho đúng bàn nó sinh ra để đánh**: arc là thứ duy nhất giáp không chặn, nhưng khi rating chưa thấy giáp thì arc chưa bao giờ phải *thật sự* là câu trả lời. `electro_ball` 285→430, `spark` 190→285, `overload` 180→270 ⇒ accumulating **193 → 426‰** (sàn 354).
- ⚠️ **Tường trong cột = câu trả lời cho chiêu diện**, và test cũ **CHE MẤT**: đối thủ của nó mang `withdraw`, rating mù báo chiêu diện vẫn thắng. Đo cả 2 bàn: **456/400 khi không có gì cản (+56)**, **286/485 khi có tường (−199)**. Test giờ giữ **cả hai hàng** — claim mạnh hơn cái nó thay.
- Claim strip tự quay lại (665 nhát bị chặn vs 1298).

### ⚠️ Mệnh đề chết tự viết ra

Tôi clamp số charge xuống số nhát *sẽ trúng* ("Roll không tiêu charge khi trượt"). Đúng lý lẽ, **chết về số học**: trừ quá tay → âm → guard bên dưới đã trả 0. **Không mutation nào phá được** ⇒ sweep báo GREEN và đó là cách phát hiện. Cú trượt tính ở đúng chỗ: `connecting`.

## `Repeat`: ĐÃ SỬA (PR#226, `f70f034`)

Luật đã viết sẵn trong `combat.go` (*"ExpectedStrikes is the figure everything outside the roll reads"*) mà rating là caller **duy nhất** không theo. Phải sửa **CẢ HAI NỬA**, mỗi nửa một mình không đổi gì:
- `Rules.Expected` nhân `h.StrikeCount()` → đổi sang `h.ExpectedStrikes()`, **chia PermilleBase NGAY TRONG `Expected`** (đơn vị là phần nghìn) để không caller nào giữ bản sao số học.
- `hitAgainst` **không hề gán** `Repeat`/`MaxStrikes` vào `Hit` ⇒ `ExpectedStrikes` không có gì để kỳ vọng.

`pricing.worstStrikes` đọc cùng cái sàn đó, giờ trả **phần nghìn**: khiên chặn NGUYÊN một nhát nên đáng **ÍT hơn** trước kẻ đánh nhiều nhát; làm tròn 3120→3 sẽ thổi phồng mọi khiên, và nó là **MẪU SỐ** nên sai số chạy về phía nguy hiểm (định giá khiên cao quá = mất mạng).

⚠️ **`Rules.Total` CỐ Ý giữ `StrikeCount`** — nó là con số tất định mà cột damage của `skills.golden` viết ra. Test assert hai hàm **đối nhau**, kèm mutation bắt `Total` cũng đọc phân phối.

📊 Magnemite với kit dựng quanh `spark`: **3.6% → 25.0%**. Rating không chịu dùng chiêu tốt nhất của chính nó. ⚠️ Không golden nào nhúc nhích, không spar mặc định nào đổi — vì `spark` ở **level 22**, ngoài 4 chiêu đầu learnset. **Đó là lý do 3.6% nằm đó không ai thấy**: `spar` chỉ đọc 4 chiêu đầu, nên lỗ nào nằm ngoài đó là lỗ vô hình với mọi phép đo mặc định.


⚠️ **Nửa giáp là MÂU THUẪN NỘI BỘ chứ không phải thiếu sót**: `shielded`+`guarded` trả tiền để DỰNG giáp, không gì trừ tiền khi ĐÁNH VÀO giáp ⇒ rating mua khiên rồi coi khiên địch như không có. Giao kèo của `warden` (tường chặn nguyên nhát ⇄ đa đòn phá tường) vô hình; `shadow_punch` mang `unblockable` miễn phí.

## `drains`: ĐÃ SỬA (PR#227, `d25683b`)

`pricing.drained` = `drainShare(skill + traits)` trên damage `expected`, rồi qua **`worthHealing`**.
- ⚠️ Clamp là TOÀN BỘ thiết kế — không có nó thì thành cộng thẳng đội lốt điều kiện máu. Test assert **cả hai chiều**: caster đầy máu thì `cleave` (xuyên 60% giáp) vẫn phải thắng `drink`.
- ⚠️ Phải cộng **CẢ trait**: đọc mỗi `declared.Drains` chỉ sửa 2/4 (`blood_thirst`, `last_gasp` là passive) — nửa fix.
- 📊 Replay golden dịch: Venusaur giờ dùng `leech_seed`, **3 lần hồi máu trong trận trước đó không hồi lần nào**.

## `taunt` + `heal_cut`: ĐÃ SỬA (PR#232, `566aaad`) — đủ 11/11 category

⚠️⚠️ **Bản đầu tôi đặt taunt vào `inflictedOn` và SAI HẲN HƯỚNG.** `Category.Harmful` bảo taunt harmful → tưởng là `inflictedOn`. Nhưng **`tauntStatus` nằm trên kẻ ĐI khiêu khích** (comment `battle.go` ghi đúng câu đó; chiêu `taunt` shipped là chiêu tự nhắm, thân toàn `self_applies`). `rate` trừ tiền cho status harmful tự áp ⇒ bản đầu **tính phí đơn vị 3× đòn mạnh nhất của chính nó** vì dám khiêu khích — **tệ hơn con số 0 cũ**.
⇒ Đúng chỗ là **`granted`**, đáng = cái AIM nó tước của MỌI địch cùng lúc; = 0 với địch mà chủ vốn đã là mục tiêu tốt nhất.
⚠️ **Fixture của tôi áp `taunting` lên ĐỊCH nên test xanh** — nó khớp mô hình trong đầu tôi, không khớp engine. Luôn kiểm status nằm trên AI.

`uncured` (heal_cut): đọc qua `healingFor` (đúng hàm trả heal, kèm sàn) trên `Set.PendingIn(Regen)` — **accessor mới**, vì `Pending()` cộng mọi stack tick KHÔNG phân dấu ⇒ định giá theo nó là được trả tiền cho POISON kẻ khác đặt. Chỉ đếm regen ĐANG THẤY, cố ý ít hơn sự thật.

📊 Taunt đáng **đúng 0 trong 1v1** theo cấu trúc (kẻ khiêu khích là mục tiêu duy nhất) ⇒ **phải đo bằng squad**: wall mang `taunt` thay `withdraw` đọc **395‰ → 783‰**, control (`withdraw`) 646‰ cả hai. Trước fix, đổi sang taunt làm đội **TỆ HƠN** control.

## Pass: ĐÃ SỬA (PR#234, `837759e`)

`if hasFallback && fallbackCooldown == 0 { cast } else { pass }`. Cơ chế pass có sẵn — `(Choice{}, false)` → `RunToEndWith` gọi `Battle.Pass`.
- Pass mang **`DeclinedReason`** riêng, không dùng `NoActionReason`: "không có nước" vs "có mà không đi" là 2 sự thật khác nhau.
- ⚠️ **`TestNothingWaitsOnPurpose` tự dự đoán chính nó** trong comment ("lý do thứ tư… test này gọi tên nó vào ngày nó xuất hiện") và **đỏ ngay** khi ghi chú mới ra đời. Viết lại thành `TestATurnIsGivenUpOnlyForAReasonThatIsWrittenDown`, giờ BẮT BUỘC ghi chú mới phải xuất hiện (4 lần / 200 trận shipped).
- ⚠️ Chiêu **cd 0 vẫn cast**: thứ rating không thấy luôn là *cái gì đó*.
- ⚠️ **KHÔNG phải "waiting" đã decided-against** (bỏ lượt để LẤY LẠI chiêu sớm — vẫn rỗng vì `spendCooldowns` chạy cả 2 đường). Cái này là **KHÔNG NẠP** cooldown. Đã ghi vào chính mục đó.
- Chờ đủ 6 lỗ định giá đóng mới làm, vì cả 6 đều under-price.

## ⚠️ Bẫy trong chính bộ đo mutation

`return worthHealing(denied, target, p.threat(target))` **không duy nhất** (`undone` cũng kết thúc bằng dòng đó) ⇒ harness đột biến **nhầm hàm**, báo GREEN giả. Giờ assert anchor xuất hiện **đúng 1 lần** trước khi chạy. Cùng luật với `Edit`.

Còn lại: **giáp** (xem trên). Kèm cái đầu: test **cấu trúc** — mọi `status.Category` phải có nhánh trong `granted` HOẶC `inflictedOn`, cộng bảng tay mọi field của `Skill` đánh dấu *đã định giá / cố ý không + lý do*. Đó là thứ lẽ ra bắt cả 4 cùng lúc.

## Fallback: ĐÃ SỬA (PR#224, `d9c9c10`)

Nhánh fallback giờ đi qua **cùng tie-break cooldown** như mọi option có điểm (`<` chứ không `<=`, thứ tự kit vẫn là tie-break cuối). Trước đó: `[scour(cd3), wipe(cd0)]` → **scour**; `[wipe, scour]` → **wipe** — thứ tự kit quyết định tất.
⚠️ **Không đổi con số nào trên cast hiện tại** (Blastoise 32.1 / Venusaur 61.1 / Clefable 10.7, không golden nào nhúc nhích) — vì **mọi chiêu power 0 trong sách đều cd ≥ 2** và cặp cùng xuất hiện thường bằng cooldown. Chỉ có tác dụng khi kit mang chiêu phụ RẺ cạnh chiêu phụ HIẾM.
→ Chính con số "cd ≥ 2 khắp nơi" làm **nửa pass quan trọng hơn tưởng**: gần như MỌI lần rơi vào fallback đều là đốt một cooldown.

## Pass: CÒN MỞ

`Suggest` **đã pass được**: trả `(Choice{}, false)` → `RunToEndWith` gọi `Battle.Pass`. Chỉ là không bao giờ dùng khi còn option đáng 0.

⚠️ **TODO § *Waiting* ("chờ là rỗng về số học") KHÔNG phủ nửa pass.** Lập luận cũ: *act = bestValue + tốt-nhất-lượt-sau; wait = 0 + cùng tốt-nhất-lượt-sau*. Hổng ở chỗ **"tốt nhất lượt sau" KHÔNG giữ nguyên** — act nạp cooldown lên đúng chiêu vừa dùng, và `TestAPassBuysNoCooldownAnActDoesNot` khẳng định đó là khác biệt DUY NHẤT act tạo ra. Khi `bestValue = 0` thì act vượt trội đúng 0, cooldown là mất trắng: `rapid_spin` (power 0, cd 3, gỡ 1 stack) quăng ra lúc không có debuff = 3 lượt cleanse đổi lấy không gì.
`frozen()` là hàm thuần của TRẠNG THÁI chứ không đếm lượt im lặng ⇒ pass không mở bàn vô hạn mới.

⚠️ **Thứ tự bắt buộc ngược lại**: cả 4 lỗ trên đều là **under-price**, nên "đáng 0 với rating" chưa bằng "đáng 0". Cho pass trước khi vá định giá = dạy AI bỏ lượt vì thứ nó chưa biết đọc.

## Đã kiểm, KHÔNG phải lỗ

- `restored` (heal) **có** định giá và Suggest **có** chọn heal — câu "Suggest never chooses a heal" trong README là prose CŨ. Bàn đo đầu của tôi sai: `p.threat(ally)` = đòn nặng nhất đánh ĐƯỢC VÀO ally (không phải đòn của ally), nên ally đứng ngoài tầm mọi thứ ⇒ threat 0 ⇒ heal đáng 0, **đúng như comment nói**.
- crit, pierce, accuracy, self_gradient, summons, replies, resists, amplifies, strips, applies, converts, cost, reserve — đều đã định giá.
