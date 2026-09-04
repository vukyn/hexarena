---
name: hexarena-poliwag-bruiser
description: "hexarena — shipping characters #182 poliwag/bruiser, #187 machop/slugger, #189 cleffa/mender; the spar-tuning loop, why a SUPPORT cannot be measured by spar, and the gates the 2026-08-26 checklist gets wrong today"
metadata: 
  node_type: memory
  type: project
  modified: 2026-08-31T09:30:13.410Z
---

PR#182 (merged `71a13fd`, 2026-08-31) ships the **5th character + 5th preset**: `pokemon.poliwag` (Poliwag→Poliwhirl→Poliwrath, water, species `amphibian`) on archetype **`bruiser`** — cột 0, cap hp 3700 / atk 660 / def 560 / spd 90 / acc 140 / ddg 40, EffHP **10632**/11500. Gloss `"bruiser": "kẻ áp sát"`.

Chiêu mới: `pummel` "đánh dồn dập" (**3 nhát** ×48% — chiêu 3-nhát ĐẦU TIÊN, trước đó max x2) · `body_slam` "lấy thịt đè người" (120% + stun 25%) · `submission` "khoá siết" (200% + expose 40%) · `bubble_beam` "tia bong bóng" (mire 40%). **Không đẻ status/passive mới.**

**Định danh preset = đếm charge, không phải sát thương:** một charge `block` xoá TRỌN một đòn chứ không xoá một phần → 3 nhát nhỏ ăn của khiên đúng bằng 1 nhát to. Đó là thứ khắc `warden`. Xem [[hexarena-core-design]].

## ⚠️ Vòng chỉnh số — `spar` chỉ đo 4 CHIÊU HỌC SỚM NHẤT

`forge.seedKit` lấy 4 skill + 1 trait **đầu tiên learnset khai**, nên **thứ tự learnset LÀ bài đo**. Bản đầu 14.5% chỉ vì `pummel` nằm ở level 24:

| chỉnh | overall |
|---|---|
| bản đầu | 14.5% |
| `pummel` → lv8 | 22.5% |
| `endurance` trước `swiftness` | 23.2% |
| hp 3400→3700, atk 620→660 | **45.5%** |

Bài học: **đừng kết luận "nhân vật yếu" trước khi kiểm tra 4 chiêu đầu learnset**. Và trait tốc độ vô nghĩa với con chậm.

Cast sau PR: naruto 83.1 · bulbasaur 54.1 · **poliwag 45.5** · squirtle 38.7 · charmander 28.5. Thêm 1 đối thủ kéo mọi hàng cũ xuống ~10 điểm. Poliwag thắng Venusaur 48% **dù thua hệ** vì đồng hồ 29 lượt kết trước đồng hồ độc — cùng lý do Charizard thắng đứt Venusaur, không phải ngoại lệ mới.

## Chỗ [[hexarena-shipping-a-character]] SAI so với code hôm nay

- **`species` đã ship** (không còn "planned"); `restrict.species` mới là trục cho chiêu dòng máu, `restrict.characters` chỉ còn là 1 trong 5 key.
- **Archetype cap ≠ top-stage stats KHÔNG có test nào ép** — chỉ là quy ước (4 con shipped đều theo). Nhiều nhân vật DÙNG CHUNG 1 archetype được; nhân vật mới KHÔNG bắt buộc đẻ archetype mới.
- **`scenarios.golden` + `replay.golden` KHÔNG đổi** khi thêm nhân vật (CLAUDE.md nói đổi — sai). Chúng đọc `roster.json`/patterns. Golden thật sự đổi: `cast` · `species` · `origins` (+ `archetypes` nếu thêm preset, `skills` + `describe` nếu thêm chiêu).
- Golden dùng **id**, không dùng `name` → **đổi gloss tiếng Việt không làm golden nhúc nhích**.
- Species **không cần bảng gloss Go** — `Lang.SpeciesName` đọc thẳng field `name`.
- Archetype **BẮT BUỘC** thêm vào `archetypeGloss` (`internal/i18n/gloss.go`) + 1 hàng vào bảng design hardcode `TestShippedArchetypesMatchTheReferenceProfiles`.

## ⚠️ 2 bẫy đã dính

1. **`skillGloss` là bảng fallback ĐÓNG BĂNG 19 id thời chưa có `name`** — id chiêu mới trùng 1 trong đó là fail (`TestTheLogGlossesNameEveryShippedID`). `flurry` nằm trong bảng → phải đổi thành `pummel`.
2. **`TestNoGlossedLogRowOutgrowsTheWindow` margin = 0** (79/79). Tên tiếng Việt dài là đỏ ngay. "lấy thịt đè người" (17) vẫn lọt; dài nhất đang ship là "phong ma thủ lý kiếm" (20).

Luật chữ: `bodyWords` khớp **substring** nhưng CHỈ soi chiêu **không có `restrict`** → chiêu có `restrict.origins` là thoát. `countWords` + `volleyWords` khớp **nguyên từ**, soi MỌI chiêu (volley chỉ khi strikes ≤ 1).

**Tên chiêu/trait viết thường hết** (user chốt 2026-08-31): 47/47 + 11/11. Một tên viết hoa đọc như danh từ riêng giữa câu tiếng Việt.

Liên quan: [[hexarena-shipping-a-character]] · [[hexarena-flavour-voice]] · [[hexarena-archetype-must-be-glossed]] · [[hexarena-roster-cannot-price-damage]]


---

## PR#187 — `pokemon.machop` + preset `slugger` (merged `1574d57`, 2026-08-31)

Machop→Machoke→Machamp. **Không có hệ fighting** → lấy `ground` (luyện núi, vác đá), nằm trên vòng organic **khắc cả 2 con water**, thua grass. Species mới `titan` "lực sĩ". Gloss `"slugger": "kẻ giáng đòn"`.
Cap: hp 3300 · **atk 760 (cao nhất cast)** · def 460 · spd 85 · **acc 210** · ddg 28 · EffHP 8375.

Chiêu mới: `vital_throw` "đòn vật" (180%, **acc 1000**) · `cross_chop` "đòn chéo" (**280%**, acc 620) · `seismic_toss` "quật đất" · `rock_throw` "ném đá" · `inner_focus` "tụ khí" (dùng `focus`).

⚠️ **`Rules.Chance` return `scale.Base` NGAY khi `SkillAccuracy >= 1000`, chưa đọc tới dodge** → chiêu acc 1000 **không trượt VÀ miễn dodge**. Mọi chiêu acc 1000 khác trong game đều 0 sát thương; `vital_throw` là chiêu đầu tiên có. Đó mới là định danh, không phải chỉ số acc.

⚠️ **Chỉ số accuracy đáng ÍT hơn con số của nó rất nhiều** — đừng định giá theo kích thước:
`landed = accuracy + Saturate(0, stat, 1000-accuracy, 0)` → đóng một PHẦN khoảng hở.
Trên `cross_chop` acc 620: Machamp (stat 210) ra **755**, Charizard (stat 150) ra **727**. 60 điểm stat = **<3 điểm** tỉ lệ trúng; `focus` chồng lên mua thêm <3 nữa. Stat acc ăn tiền ở chỗ **chống dodge**, không phải cứu acc thấp của chính chiêu.

**59.8% ngay bản đầu, KHÔNG cần vòng chỉnh** — vì learnset đặt `cross_chop`@8, `vital_throw`@12 nên spar bắt trúng chiêu định danh. Ngược hẳn poliwag. **Xác nhận lại bài học: thứ tự learnset LÀ bài đo.**

Cast sau: naruto 81.1 · machop 59.8 · bulbasaur 58.3 · charmander 37.4 · poliwag 36.8 · squirtle 26.6. Biên độ 54 (5 con trước là 55) → con thứ 6 không nới rộng. Machop là con thứ 2 (sau bulbasaur) lấy được game thật từ naruto (27%).

⚠️ **Test length-fragile MỚI, chưa từng ghi:** `TestAWideWindowWidensTheDataCells` (`cmd/hexforge-tui/width_rule_test.go`) — fixture `aSkillTheWholeCastIsNamedOn` dựng allowlist gọi tên **CẢ CAST**. Test đòi ô vừa **quá dài cho sàn** VỪA **đủ ngắn cho cửa sổ 200**; danh sách lớn theo cast chỉ thoả vế đầu → con thứ 6 vượt 200 cột, đỏ vì lý do không liên quan luật đang đo. Sửa **fixture** (cố định `namedCharacters = 5`), không update golden.


---

## PR#189 — `pokemon.cleffa` + preset `mender` (merged `20499dd`, 2026-08-31)

Cleffa→Clefairy→Clefable, hệ **`light`** (con light ĐẦU TIÊN), loài mới `fae` "tiên", gloss `"mender": "người chữa lành"`.
Cap: hp **4200 (cao nhất cast)** · atk **420 (thấp nhất)** · def 380 · spd 115 · acc 155 · ddg 55 · EffHP 9523.

⚠️ **`solar_beam` là chiêu `light` mà TRƯỚC ĐÓ KHÔNG AI CẦM ĐƯỢC** (bulbasaur là grass) — ship ra nằm không từ đầu. Cleffa là carrier đầu tiên. Bài học: **grep xem chiêu nào chưa ai mang trước khi đẻ chiêu mới**.

Chiêu mới: `moonlight` (**hồi 400‰ theo `scaling: defense`** — chiêu hồi đầu tiên không tính theo attack; `skull_bash` đã chứng minh field này chạy nhưng chưa ai dùng cho restore) · `charm` (**`weaken` ×2** — chiêu ĐẦU TIÊN gây `weaken`, status ship sẵn mà chưa gì chạm) · `moonblast` · `wish` (regrowth cho ĐỒNG ĐỘI) · `sing` (stun).

## ⚠️⚠️ SUPPORT THUẦN KHÔNG THỂ THẮNG 1v1 — và `spar` KHÔNG đo được nó

Cleffa hồi nhanh hơn tự đánh → **mọi trận `Endless: 40`** (chạm trần 4000 lượt), `TestABothWaysMirrorIsExactlyEven` ĐỎ. Đó là **bất biến công bằng**, không phải con số cân bằng — không được bỏ qua.

**Thứ chữa được KHÔNG phải hạ hồi máu mà là HẠ GIÁP.** def 560 → mirror không bao giờ kết; def 380 → mọi trận kết, mirror đúng 500‰. Tôi mất 3 vòng mới thấy vì cứ nhắm vào con số hồi máu.

| vòng | kết quả |
|---|---|
| def 560, heal 900 | Endless toàn bộ |
| heal 900→400, `wish` ra khỏi kit sớm | vẫn Endless |
| **def 560→380**, atk 340→420 | mọi trận kết, mirror 500‰ |
| hp 3300→4200 | matchup khác 0 đầu tiên (poliwag 75‰) |

**Spar cho 0‰ trước 5/6 con và con số đó ĐỊNH GIÁ ZERO.** Dụng cụ đúng là **đánh theo đội**: cùng 1 striker + 1 wall, 2 đội chỉ khác ô thứ ba, 2 chiều 300 seed → `TestAMenderEarnsItsSlotWhereASparCannotSeeIt`:
- vs slugger (machop) **525‰**
- vs bruiser (poliwag) **543‰**

Sàn để 450 — giữ **luận điểm**, không giữ con số. Đây là phép đo DUY NHẤT trong repo định giá được support. CLAUDE.md đã ghi sẵn điều tương tự cho build tank của Squirtle ("`hexforge spar` cannot measure either build") — đây là lần nó đúng với cả NHÂN VẬT chứ không chỉ một build.

⚠️ **Lần thứ 3 fixture vỡ vì thêm nhân vật** (sau #187 width fixture): `TestASweepRefusesWhatItCannotAnswerAtAll` + bản CLI song sinh dùng `solar_beam` làm "chiêu không ai cầm" — chỉ đúng khi chưa có con light nào. Sửa = **tự author chiêu sau khi đọc cast**, không mượn dữ liệu ship. **Quy tắc chung: fixture phải TỰ DỰNG điều kiện, đừng mượn sự thật của cast.**

**Ảnh**: `img2svg -q balanced` (preset dòng Naruto), 63/115/117 KB, xoá PNG sau khi trace.

---

## PR#194 — 6 nội tại mới, mỗi cái mở một cơ chế CHƯA AI DÙNG (merged `8feb9ce`)

`composure` "vững tâm" (resist stun+taunting 500) · `elusive` "huyền ảo" (grants `evasive`, **permanent dodge MỚI** +150‰) · `contagion` "lây lan" (`applies` weaken 250 lên đòn của chính nó) · `spiteful` "nham hiểm" (`amplifies` **chance** stun/mire 300) · `berserk` "cuồng chiến" (`unleashed` + **vulnerability** −300 stun/mire) · `unyielding` "bất khuất" (miễn taunting).

**Cách tìm chỗ trống — chạy lại được:** đọc `passives.json`, đếm trường nào có ai dùng. Kết quả lúc đó: `applies` **0 user**, `resists` âm (vulnerability) **0 user**, `resists` chỉ chạm burn+poison, `amplifies` chỉ 1 user (virulence, poison, effect+chance), và **`accuracy` + dodge-dương là 2 trục stat không có permanent status**.

## ⚠️ `berserk` — vulnerability LÀ cái giá dùng được

Đo bằng cách đẩy trait lên **đầu learnset** (thứ `spar` đọc) rồi trả về. 40 seed/slot:

| carrier | | |
|---|---|---|
| machop | `blood_thirst` 66.6% | **`berserk` 58.5%** · `reckless` 44.1% |
| bulbasaur | `venom_blood` 68.3% | `contagion` 69.7% |
| poliwag | `endurance` 43.7% | `spiteful` 35.0% |

Cùng `unleashed` y hệt `reckless` mà 58.5 vs 44.1 → **trả giá bằng "dễ dính khống chế" rẻ hơn nhiều so với `bare` (giáp)**. Đó chính là "loại chi phí khác" mà CLAUDE.md nói `reckless` đang thiếu — 3 cần gạt của reckless chết vì đều cùng đồng tiền defence.

## ⚠️ 2 con số đo ra là của NGƯỜI MANG chứ không phải của trait

- `unyielding` 40.2% = **"Machamp không nội tại"**: không kit spar nào mang `taunt` (Squirtle học lv40, spar lấy 4 chiêu SỚM NHẤT).
- `spiteful` đo trên bulbasaur ra 65.4/68.3 vì **kit nó toàn poison** — trait này không khuếch đại poison. Dời sang **poliwag** (bubble/bubble_beam→mire, body_slam→stun) mới đúng nhà.

**Bài học: trước khi kết luận trait yếu, kiểm tra kit của người mang có sinh ra status mà trait đó tác động không.**

**Không dịch figure nào đang ship**: nối trait mới **SAU** cái carrier đã khai đầu tiên → kit + trait mặc định y nguyên.

⚠️ **`TestATraitsSharesAreRoundedAndNotTruncated` vỡ vì share ÂM**: nó làm tròn `(permille+5)/10`, Go cắt về 0 nên `-300` → `-29%`; renderer in `30%` với **dấu nằm ở động từ**. Test chưa từng gặp âm vì vulnerability chưa ai dùng. Sửa = làm tròn **độ lớn**.

**Đặt tên trait**: 11 trait cũ **không cái nào phán xét đạo đức** (bền bỉ, khát máu, liều mạng, trụ vững) → user bác `hiểm ác` ("ác" = verdict), chọn `nham hiểm`. Và **không cần dịch sát**: `thorns`→"phản đòn", `ballast`→"trụ vững" đều đặt theo THỨ NÓ LÀM.

---

## PR#197 — 2 build/con cho poliwag, machop, cleffa (merged `87e986b`, tổng 12 build)

⚠️ **1 build/con KHÔNG DỰNG ĐƯỢC** — `TestABuildIsACatalogueOfChoicesRatherThanOfKits` từ chối đích danh: "một build không phải build, nó là kit của nhân vật". Ai vào `builds.json` phải có **≥ 2**; không có build nào là trường hợp trung thực (naruto).

| build | kit | trait |
|---|---|---|
| `poliwag.flurry` "liên hoàn" | pummel body_slam submission water_gun | blood_thirst |
| `poliwag.riptide` "nghịch lưu" | bubble bubble_beam whirlpool hydro_pump | **spiteful** |
| `machop.gamble` "đánh cược" | cross_chop submission seismic_toss inner_focus | **berserk** |
| `machop.sure` "chắc đòn" | vital_throw body_slam rock_throw seismic_toss | **unyielding** |
| `cleffa.mend` "chở che" | moonlight wish rally moonblast | **composure** |
| `cleffa.hex` "ru ngủ" | charm sing smokescreen solar_beam | **elusive** |

Mỗi cặp là **nhà cho 1 trong 6 trait của #194** — build là chỗ trait mới được dùng thật.

**Ship build mới = 3 việc**: (1) hàng trong `builds.json`, (2) biến `[]string` design record + hàng trong `TestTheShippedBuildsAreTheOnesTheTestsMeasure` (so cả **kit lẫn trait**), (3) một test ĐO trước khi liệt kê.

Khuôn đo (copy `bulbasaurRun`): đánh 60 seed trước **một đối thủ giữ nguyên** (Charizard), đọc từ event log — `Damaged` actor=mine, `StatusTicked` actor=**theirs** (tick không ghi tác giả!), `Healed`, `Missed`, `StatusApplied`. **Không cho 2 build đánh nhau** — mirror duel chỉ đo con sinh đôi.

Số đo: flurry 19 lượt/3210/hồi 726/45 status · riptide 17/3238/hồi 0/**188 status** · gamble 13/2858/**41 trượt** · sure 14/2443/28 trượt · mend 27/989/**hồi 1430**/66 · hex 24/856/hồi 0/**331 status**.

⚠️ **CỘT ĐẾM PHẢI GIỮ TỔNG, ĐỪNG CHIA TRUNG BÌNH.** Bản Machop đầu ra **0 vs 0** vì `tổng/60` chia nguyên — trận ~13 lượt, vài chục lần trượt chia 60 = 0, test báo 2 kit y hệt trong khi một bên trượt nhiều hơn rưỡi. Cùng họ với các bẫy "test xanh mà đo rỗng" khác trong repo.

⚠️ Build của Cleffa **không thắng trận nào** trong phép đo — và đó không phải thứ đang đọc. Mender thua mọi duel dù cầm gì; thứ duel còn nói được là **nó tiêu lượt vào việc gì**.

---

## PR#198 — Politoed: **FORK ĐẦU TIÊN ĐƯỢC SHIP**, và cái giá của nó (merged `ed79a28`)

`Stage.After` có sẵn từ lâu, CLAUDE.md ghi "Nothing shipped forks yet, **deliberately**". Chữ *deliberately* mới là toàn bộ câu chuyện.

`Poliwhirl@16 → Poliwrath@32 (3700/660/560)` **hoặc** `→ Politoed@32 (3800/580/580)`.
Skill mới: `rinse` "gột sạch" (**chiêu ĐẦU TIÊN tẩy status cho ĐỒNG ĐỘI** — `rapid_spin` chỉ tẩy cho chính nó) · `chorus` "hợp xướng" (**chiêu đầu tiên trao `haste`**). Gate `[Poliwrath]` cho `submission`, `[Politoed]` cho `rinse`/`chorus`/`composure`.

## ⚠️⚠️ CÁI GIÁ: `Resolve` TỪ CHỐI khi 2 nhánh, nên mọi chỗ hỏi "dạng xa nhất" đều gãy

Dữ liệu chỉ 4 dòng; **26 file phải sửa**. Đây là thứ phải biết TRƯỚC khi hứa thêm fork:

**3 tool đổi chữ ký:**
- `cast.Build` + `stage` (build là loadout đầy đủ; trên dòng rẽ nhánh thì *nhánh nào* là một phần của nó)
- `Spar` + stage: **chủ thể** rẽ nhánh → TỪ CHỐI kèm tên 2 nhánh; **đối thủ** rẽ nhánh → fielded TỪNG NHÁNH (câu trả lời `Inspect` đã dùng). `--stage` cho CLI; TUI truyền `row.Stage` của màn check (màn đó vốn đã 1 hàng/nhánh).
- `WeighRequest`/`CarriersRequest` + stage, VÀ sweep phải **hỏi membership theo từng nhánh TRƯỚC khi cân** — thiếu bước đó thì nhân vật không mang chiêu lọt vào bảng thành "hàng bị từ chối" (= "không định giá được") thay vì "chưa bao giờ là người mang".

⚠️ **Cân từng nhánh trong `WeighCarriers` là NGÕ CỤT** — `forkedTwin` là fixture tồn tại CHỈ ĐỂ tạo hàng bị từ chối, và cơ chế của nó chính là fork. Cân theo nhánh xoá mất ví dụ đó. Đã lùi về 1 hàng/nhân vật + membership theo nhánh.

**8 test phải học lại**, mỗi cái viết khi một đáp án là khả năng duy nhất:
- 2 test đọc `row.Art[0]` — `Inspect` cố ý liệt kê art **1 lần/nhân vật**, nhánh 2 có `Art=nil` → một cái **panic**
- `TestGivingUpAnEvolution…` vớ phải `chorus`: **gate trỏ vào dạng mà cap VẪN CHẠM TỚI là chọn-nhánh, không phải đánh-đổi-tiến-hoá** — cùng allowlist, hai chuyện khác nhau. Scan phải bỏ qua loại đó.
- `TestTheSparFightsWhoever…` dùng chỉ số cast làm chỉ số hàng — hai list hết trùng nhau
- `TestEveryShippedCharacterResolves`, 2 walk trong floor_test, các sweep spar → phải duyệt nhánh

⚠️ **Politoed đọc 14.2% spar (Poliwrath 43.7%) — con số đó định giá KIT chứ không phải nhánh.** Spar lấy 4 chiêu học sớm nhất = mấy cú đấm của Poliwrath; đồ nghề Politoed (`rinse`/`chorus`) 0 sát thương, nằm ở 32/36. Cùng điểm mù với mender. Thứ định giá nó là **build** `poliwag.chorus` fielded *as Politoed*.

## PR#197 + #198 — catalogue build: 13 build / 6 trong 7 nhân vật

Poliwag có **3** (2 Poliwrath + 1 Politoed) — con duy nhất >2, vì nó rẽ nhánh. Naruto **0** (trường hợp trung thực).
Build order trong file = thứ tự hiển thị → build cùng nhân vật phải **kề nhau**, đừng append cuối file.

## PR#200 — `hexforge builds` (merged `a125463`)

Book cuối cùng chưa có listing. Cột `form` **riêng** (trên dòng rẽ nhánh nó là phần của loadout, không phải thuộc tính nhân vật). Đếm theo **độ phủ cast** ("13 builds across 6 of the 7"), không chỉ đếm build — vì nhân vật không build là trường hợp hợp lệ.
