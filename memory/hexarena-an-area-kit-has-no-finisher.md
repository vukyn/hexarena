---
name: hexarena-an-area-kit-has-no-finisher
description: "hexarena — một `column` bắt đúng MỘT ô có người trên cả năm đội hình đã ship, nên splash_power không bao giờ tới và nửa diện của kit chưa từng được đánh: igglybuff 0‰ vs mewtwo 1000‰ cùng ghế; ⚠️ chồng đội cho diện tích tốn ~250‰ phòng thủ, đắt gấp đôi thứ nó mua"
metadata:
  node_type: memory
  type: project
---

⚠️ **Bản đầu của ghi chú này nói sai ở câu chính, và sửa tại chỗ chứ không thêm
ghi chú thứ hai.** Nó viết *"chiêu diện ở power chuẩn sách đáng ÍT hơn hẳn burst
đơn mục tiêu"*. Không phải. Câu đúng là:

**Trên cả năm đội hình `squads.json` đã ship, một `column` bắt đúng MỘT ô có
người.** `pattern.Book.Power` trả full power ở index 0 và `splash_power` từ
index 1 trở đi, nên `splash_power` **không bao giờ tới** với một `column` trên
bất kỳ bàn nào `forge.FightSquads` đánh. `dazzle` (column 900) là một chiêu
**đơn mục tiêu 900** trong mọi con số đã trích — và `discharge`, `night_shade`,
`magnetise` cũng vậy. Nửa diện của kit chưa từng được đánh; nó không bị định giá
thấp, nó **không tồn tại trên bàn**.

Đo 2026-09-08 bằng cách replay `hex.Place` + `pattern.Targets`, chỉ nhắm vào ô có
người (đúng những ô `battle.aims` cho phép):

| hình | tốt nhất trên bàn `FightSquads` đánh | trên `roster.json` |
|---|---|---|
| `column` | **1** | 2 |
| `flank_up` | **1** | 2 |
| `arc_up` | 2 (ở `s02`) | 3 |
| `arc_down` | 2 (hình học; ko đội nào mang) | 3 |

Nguyên nhân một câu: **`s02`–`s05` đứng `(2,1) (1,1) (0,1)`, `s01` đứng
`(2,0) (2,2) (0,1)` — mỗi unit một cột riêng** — mà `column` toé `up`/`down`, tức
là *trong* cột. Các đội đã ship chỉ kề nhau theo `upper_right`, nên hai `arc` là
hình duy nhất chạm được ô thứ hai.

**Số đo gốc vẫn đúng như số, chỉ đọc lại ý nghĩa.** `forge.FightSquads`, 200 seed
mỗi chiều, mọi mirror control đọc đúng 500‰. Ghế giữ cố định: `s05` = machamp +
gengar + ghế sau; `c05` = y hệt hai đồng đội đó với Mewtwo ở ghế sau. Đối đầu
cùng đội: **`c05` 1000‰ — 400/400 trận**.

| một biến so với bản ship | vs `s01` | vs `s04` |
|---|---|---|
| bản ship | 5‰ | 0‰ |
| tốc 95→140 | 32‰ | 5‰ |
| tốc **và** chính xác = 140/200 của Mewtwo | 32‰ | 7‰ |
| `plunge` 1900→2400 (trần tier chiêu-đổi-máu) | 7‰ | 0‰ |
| `endurance`→`berserk` | 10‰ | 2‰ |
| thêm `solar_beam` (2400) vào 4 chiêu ra sân | 2‰ | 47‰ |
| **bỏ cả 3 chiêu diện → 3 chiêu đơn nặng + `berserk`** | **172‰** | **237‰** |
| đối chứng: Mewtwo cùng ghế | 975‰ | 660‰ |

⚠️ **Dòng quyết định của bảng là so POWER, không phải so DIỆN TÍCH.** Nó thay
900 / 1200 / 380×3 bằng những chiêu ~2400 — và **cả sáu chiêu, trước lẫn sau, đều
đánh trúng đúng một mục tiêu** trên bàn đó. Không có gì trong dòng đó nói về diện
tích cả.

⚠️ **Sửa được không? ĐÃ THỬ, ĐÃ ĐO, ĐÃ BỎ.** Thêm `s06` copy nguyên ba thành viên
của `s04` (cùng nhân vật, level, stage, kit, passive) sang đội hình chồng của
`roster.json` — `(1,0) (1,1)` cặp kề nhau, `(0,1)` ace phía sau — nên **vị trí là
biến duy nhất**. Nó làm trục diện tích có thật: `column` bắt **2**, `arc_up` bắt
**3**, và `dazzle` từ **im (0 cast)** thành **13 cast**. Rate thì đi từ **0‰ sang
2‰**, còn đối chứng không mang chiêu diện (`s01`) đi **−7‰** — tức là dịch xa
hơn cả đối tượng. Ngưỡng đặt trước khi chạy là 150‰. Rớt, nên `s06` bị bỏ, không
byte nào của `squads.json` đổi.

⚠️ **Lý do rớt mới là phát hiện thật: CHỒNG ĐỘI TỐN NHIỀU HƠN THỨ DIỆN TÍCH MUA
VỀ.** `s04` vs `s06` = **752‰** — cùng ba nhân vật, chỉ khác ba con số ô, mất
khoảng **253‰** — vì đội hình chồng đặt hai unit vào hàng có người đầu tiên và
kéo ace từ độ sâu 3 xuống 2, mà **reach đếm theo HÀNG CÓ NGƯỜI**. `docs/balance.md`
đo độc lập cùng cái giá đó ở chỗ khác: *"đội dồn một cột đọc 464‰ vs `s01` trong
khi trải ra đọc 677‰ — chồng tốn khoảng 213‰"*. Hai dụng cụ, cùng một bậc.

⚠️ **Census nói rating KHÔNG phải chỗ hỏng.** `hexforge census igglybuff.vang
--against <squad> --seeds 40`, 80 trận mỗi bàn: `dazzle` 63 / 31 / **0 im** / 68
trên `s01` / `s02` / `s04` / `s05`, `refrain` 82 / 92 / 77 / 80. Kit **có** được
chơi, và slot đi im đúng ở bàn mà một column đáng ít nhất. Rating định giá đúng
thứ chiêu thật sự làm. `covers` cũng chỉ có MỘT khai báo cho cả hai bên
(`turn.go:742`) và dòng splash giống hệt nhau ở `turn.go:1278` với `ai.go:496` —
engine không thiếu gì.

⚠️ **spar (1v1) đo ra hình dạng NGƯỢC, đừng dùng nó cho carrier chiêu diện.**
Duel định giá một `column` bằng **một** mục tiêu. Ở đó: overall 4.5% (cleffa
26.8%, oddish 59.4%, happiny 0.0%), probe đơn stat **phẳng hết** — acc 4.5% · atk
4.7% · def 4.6% · spd 8.2% — mà nhân đôi power mọi chiêu mới thì đọc 47.4%. Duel
*có* nhạy với power, nó chỉ không thấy diện tích. Cùng giới hạn mà
[[hexarena-poliwag-bruiser]] ghi cho support.
⚠️ **Cảnh báo đó thiếu một bậc:** một đội 3 người đã ship cũng định giá `column`
bằng một mục tiêu, y như duel. Đổi sang squad **không** thoát được.

⚠️ **`Rate()` bỏ trận Endless khỏi mẫu số**, nên tỉ lệ đẹp lên trong khi mẫu co
lại: một probe ở 1.5× power đọc 15‰ → 176‰ mà endless đi 202 → 301 của 400. Luôn
trích số endless cạnh mọi rate lấy trên đội khó kết trận. Cùng bẫy
[[hexarena-starter-squads]] đã ghi.

⚠️ **Một số tôi báo lúc đầu SAI, và sửa tại chỗ chứ không xoá.** Probe thay
Magnezone khỏi `s02` đọc 386/400 endless ở mirror của nó và tôi đã báo đó là
crooner làm đông bàn. Không phải: **mirror của `s02` tự nó 166 endless** vì
Blastoise mang bốn chiêu power 0, và bỏ nguồn sát thương duy nhất của đội đó mới
là thứ làm đông. Mirror `s05` đã ship đọc **endless 0**. Bài học: trước khi gán
một stalemate cho unit vừa thêm, **đo mirror của đội gốc**.

**Why:** tiền đề "định giá chiêu diện theo peer trong sách rồi ship" nghe đúng và
sai, nhưng không phải vì lý do bản đầu nói. Từng chiêu đúng chuẩn peer
(`dazzle` 900 = `discharge` 900; `refrain` 1200 giữa `hyper_voice` 900 và
`air_slash` 1400) — và **các peer đó cũng đều là `column`, cũng đều đơn mục tiêu
trên mọi bàn được đánh**. Peer chưa bao giờ là bằng chứng.

**How to apply:** trước khi định giá hay đọc bất kỳ chiêu diện nào, **đo đội hình
trước, đừng đọc sách trước** — hỏi hình đó bắt được mấy ô CÓ NGƯỜI trên bàn sắp
đánh. Vẫn đừng cho một nhân vật kit toàn chiêu diện: cho nó một đòn kết đơn mục
tiêu, đúng cách Magnezone dùng `discharge`. Và đừng đụng `splash_power`: nó
không tới được với cả 21 chiêu `column` trên mọi đội đã ship, nên nâng nó là nâng
một số không ai đọc. `TestAShippedFormationIsCatchableByTheShapesItFields`
(`internal/seed/areaboard_test.go`) giữ sàn dưới chuyện này — nó suy ra hình từ
dữ liệu chứ không từ danh sách chiêu viết tay, và hôm nay chỉ `roster.json` đỡ
được nó. Câu hỏi còn mở ở `TODO.md` `DAT-007`.

Related: [[hexarena-roster-placement]], [[hexarena-roster-cannot-price-damage]],
[[hexarena-allsided-and-scarcity]], [[hexarena-fester-heal-cut]],
[[hexarena-speed-and-measurement]], [[hexarena-two-field-rebalance]],
[[hexarena-a-rate-cannot-say-a-build-played-its-kit]],
[[hexarena-a-fixture-chosen-by-property-is-blind-to-other-properties]].
