---
name: hexarena-draft-and-spectator-plan
description: "hexarena ban/pick + spectator: bước 1 (pool) + 2a (state machine) + 2b (arrange) XONG; ban 2/bên ở 3v3, 3/bên ở 5v5, pool 17 nên CẢ HAI format vừa (5v5 dư 1); ⚠️ pool KHÔNG thể cạn giữa draft — đã chứng minh; ⚠️ Done() giờ là CẢ draft, nghĩa cũ là Picked(); spectator KHÔNG được là chỗ ngồi thứ 3"
metadata: 
  node_type: memory
  type: project
  modified: 2026-09-03T17:44:11.025Z
---

hexarena PR #269 (`0736e54`, 2026-09-04): ghi TODO cho **ban/pick** + **spectator** (CHƯA LÀM), và **ẩn 5v5**.

**Quyết định của user (2026-09-04):**
1. Pick nhân vật → rồi chọn **build có sẵn trong `builds.json` HOẶC dựng tại chỗ**. `cast.ChooseLoadout` vẫn là luật hợp lệ duy nhất cho cả hai → draft thêm **bộ chọn**, không thêm luật.
2. **Ban theo FORMAT: 2/bên ở 3v3, 3/bên ở 5v5** — đối xứng, TUỲ CHỌN (được để trống hết).
3. Hết giờ pick → **huỷ cả phòng**, làm lại từ mã mới. Đây là chỗ DUY NHẤT không theo "timeout thông báo rồi bỏ lượt".
4. Ban theo **trận**, **bo1 trước**; bo3 là item riêng (pool reset hay không là quyết định thiết kế, không phải tham số).
5. Pool = mọi nhân vật **không** `cast.Character.Hidden`.

**How to apply:**

⚠️ **Pool là 17** (đo lại 2026-09-05 sau khi `pokemon.pichu` ship; ghi chú này từng viết 11, rồi 15, rồi 16 **trong cùng một ngày**, TODO.md từng viết 12 rồi 14) — 18 nhân vật ship, chỉ `naruto.naruto` có `hidden: true`. Với số ban đã chốt: **3v3 = 6 picks + 4 bans = 10/17 → VỪA**, dư 7. **5v5 = 10 + 6 = 16/17 → VỪA**, dư 1 — nên **pick cuối của 5v5 LẠI là quyết định** (thấy 2 lựa chọn); nó chỉ *không* là quyết định trong mấy giờ pool đứng ở 16. Con số này đổi mỗi lần ship nhân vật nên **phải suy ra chứ đừng nhớ**: `jq '[.characters[]|select(.hidden|not)]|length' internal/seed/data/cast.json`. ⚠️ Đọc theo "tổng cả hai bên" thì 8/17 và **13/17 → VỪA, dư 4** — ở pool 15 cách đọc từng đổi cả KẾT LUẬN chứ không chỉ con số đích. Đó là lý do thứ hai để (b) nói rõ *mỗi bên*.

⚠️ **Ban TUỲ CHỌN KHÔNG biến thiếu hụt thành lỗi lúc chạy — ghi chú này từng viết ngược, đã chứng minh sai 2026-09-05.** Mỗi ban và mỗi pick lấy ĐÚNG MỘT nhân vật khỏi pool, ban bỏ trống lấy 0, nên tuỳ chọn chỉ có thể làm pool ĐẦY HƠN. Tổng lấy ra tối đa là `2*picks + 2*bans` — đúng con số `draft.Fits` đã bắt pool phải chứa. Trước quyết định thứ k thì tối đa k-1 nhân vật đã đi, nên luôn còn ít nhất một, và pool không bao giờ tụt dưới số pick hai bên còn nợ. **Phòng mở hợp lệ thì draft chạy xong**, bất kể thứ tự quyết định và bất kể ban nào bị bỏ. Nên **KHÔNG có luật lúc chạy nào phải viết** — không từ chối ban, không xám ô — ở draft, room hay screen. Thay vào đó là `draft.Fits`: đo theo trường hợp xấu nhất (spent hết ban) nên **chặt hơn mức cần thiết** một cách có chủ ý, và một cấu hình bị từ chối lúc mở phòng vẫn hơn một draft chết giữa đường. Chứng minh vét cạn: `TestNoDraftThatFitsCanRunOutOfCharacters` (2 format × pool 0–40 × mọi chuỗi ban/skip × 3 thứ tự = 9840 cấu hình), và `TestADraftThatDoesNotFitCanRunOutOfCharacters` là nửa chứng minh phép mô phỏng NHÌN THẤY được cái thất bại nó bảo không xảy ra.

⚠️ **Nửa còn lại của ghi chú slack vẫn đúng**: **pick cuối không phải quyết định khi `pool − 2*picks − 2*bans = 0`** — đó là `draft.Slack`, và với spent hết ban thì pick cuối thấy đúng `slack + 1` lựa chọn.

⚠️ **Thứ tự đã chốt (2026-09-05): BAN HẾT TRƯỚC, RỒI MỚI PICK** — không xen kẽ ban-pick, không snake. Câu văn của chính TODO.md (*"the two sides take turns banning a character and picking one"*) đọc ra xen kẽ và là ứng viên còn lại, nên nó được ghi lại chứ không sửa. Bước 1 không xếp thứ tự gì cả và phép tính độc lập với thứ tự, nên chốt thứ tự không tốn gì.

⚠️ **SPECTATOR KHÔNG ĐƯỢC LÀ CHỖ NGỒI THỨ BA — nó đổi AI THẮNG.** Comment `seatCount` (series.go) nói chỗ ngồi là **mảng chứ không map** vì **thứ tự thăm hai chỗ đi vào roster, và thứ tự roster quyết ai thắng khi hoà tốc độ**. Xâu người xem qua cùng cấu trúc = dịch thứ tự đó. **Không gì trong suite bắt được**: roster vẫn hợp lệ, hai peer cùng có spectator vẫn khớp mọi digest — chỉ so với trận đánh KHÔNG có nó mới thấy. Spectator phải là **hạng công dân khác**: cursor riêng, `Deliver` từ chối luôn, không tồn tại với `seats`/roster/`other()`.
**Nửa đắt ĐÃ XONG**: `battle.Since` + bản ghi append-only (#236). Ba con số 2 chặn: `room/series.go seatCount`, `socket/table.go seatsPerTable`, `wire` chỉ `SeatHost`/`SeatGuest`.

**Draft là state machine thứ hai cùng hình dạng** với `internal/room` — chuỗi quyết định, timeout là **input** không phải đồng hồ → cùng lệnh cấm. **Mẹo mirror chuyển nguyên vẹn**; thứ draft sinh ra là **hai roster** = đúng thứ `wire.Start` đã chở → không gì phía sau draft đổi. ⚠️ Nhưng spectator xem draft cần **bản ghi mà draft chưa có**.

⚠️ **`Hidden` có hai nghĩa, đừng "lọc ở mọi nơi"**: `screen/squads.go` TÔN TRỌNG nó (chọn ai ra trận — draft cũng vậy), nhưng `screen/picker.go`'s `CharacterOptions` **cố ý VẪN hiện** nhân vật ẩn (đó là danh sách `restrict.characters`; lọc sẽ làm một restriction đang tồn tại **không lưu lại được**). Cổng draft biến flag thành **tuyên bố thiết kế**, và comment của chính flag đã nói vậy từ 2026-09-05.
⚠️ **`cast.Book.Offered()` đã CÂN NHẮC VÀ TỪ CHỐI, không phải còn để mở.** Comment cũ của `offeredCharacters` dự đoán "ngày có screen thứ hai cần danh sách này thì accessor là thứ nên làm" — screen thứ hai đã tới và lập luận KHÔNG đảo: draft cần Hidden thuần, squad builder cần Hidden-trừ-một-`keep` (chuyện của đội đang sửa, không book nào biết được), nên accessor chỉ trả lời nửa của mỗi bên. Hai vòng lặp ba dòng, mỗi vòng nằm cạnh luật của nó.

**Bước 1 XONG** (`internal/draft/pool.go`, nhánh `feat/draft-pool`): `NewPool`/`Len`/`Has`/`All` + `PicksPerSide`/`BansPerSide`/`Slack`/`Fits`, kèm bản copy lệnh cấm import của `internal/room`. ⚠️ **Thứ tự pool là thứ tự KHAI BÁO của cast book, KHÔNG sort theo id** — đó là chuyện trình bày (5 danh sách khác trong repo cũng theo thứ tự khai báo), và ⚠️ **cast ship hôm nay tình cờ ĐANG sort theo id** nên test thứ tự phải dùng fixture riêng, không thì đo được cái gì cả.

**Bước 2a XONG** (`internal/draft/draft.go` + `record.go`, PR #308): `New`/`Turn`/`Ban`/`SkipBan`/`Pick`/`Loadout`/`Candidates`/`TimedOut`/`Cancelled`/`Picks`/`Since`. Sản phẩm là `Picks() [2][]Pick` — ô đứng là quyết định của bước 2b.

**Bước 2b XONG** (`internal/draft/arrange.go`, nhánh `feat/draft-arrange`, 2026-09-05): `Arrange(seat, slots)` — **cả sắp xếp của một bên trong MỘT lời gọi**, `Arranging()`, `AwaitingArrangement()`, `Squads()`, thêm `StepArrange` + `Entry.Slots`. Bốn điều đáng nhớ:
- ⚠️ **`Turn()` KHÔNG nới ra**: hai sắp xếp mở cùng lúc, mà `Turn` trả một chỗ ngồi một bước — nới nó thành *tập hợp* là đổi mọi lời từ chối trong package vì một pha. Nên pha này có accessor riêng, và `Turn` trả `false` ngay khi hết pick.
- ⚠️ **`Done()` ĐỔI NGHĨA, nghĩa cũ thành `Picked()`.** `Done()` = cả draft (pick **và** sắp xếp, đúng trạng thái `Squads()` trả lời); `Picked()` = mọi ban đã dùng/bỏ, mọi pick có loadout. Lý do: cách đọc NGUY HIỂM là cách một caller hỏi trước khi ra trận, nên cái tên nó với tay tới phải là cái an toàn. Đổi đủ 7 chỗ + doc comment; lời từ chối *"this draft is finished"* **tách thành hai câu** vì hết pick giờ là hai trạng thái khác nhau.
- ⚠️ **Bản ghi giữ hai entry lại rồi ghi CÙNG LÚC, theo thứ tự `seats`** chứ không theo thứ tự đến. Giá phải trả: bản ghi **không nói được ai sắp trước**, nên mirror dựng lại được *cuối* pha chứ không dựng lại được *giữa* pha — `TestARecordReplaysIntoTheSameDraft` hoãn đúng một phép so sánh và **đếm** phép hoãn đó.
- ⚠️ **`Squad.Units` sắp theo Ô, row-major (Row là vòng ngoài)** — thứ tự slice quyết ai thắng khi hoà tốc độ, tới 60 điểm ở trận mirror. Thứ tự pick bị TỪ CHỐI. Row là trục engine bỏ qua ở mọi nơi khác (reach đếm theo rank nên Col đã quyết ai bị nhắm trước), nên tie-break bám vào nó thì không cộng dồn lợi thế của độ sâu. ⚠️ Đây là thứ tự lưới **thứ ba** trong repo — `hex.Cells`/`SideCells` là column-major, `squads.json` viết cột trước-tiên.
- **`Placement.ID` = id NHÂN VẬT**, vì pool loại trừ nên không bên nào trùng — và cả hai bên cũng không trùng nhau. Tiền tố phe của `Squad.Take` vẫn giữ vì lý do riêng của nó.
- ⚠️ Timeout trong pha này **huỷ cả draft** và **bỏ** sắp xếp đang giữ (ghi nó ra là làm rò đúng thứ cái buffer che). ⚠️ Việc *xoá* buffer đo được là **không quan sát được** (xoá dòng đó suite vẫn xanh, vì mọi accessor chặn ở `Cancelled`/`Picked` trước) — nó là giải phóng chứ không phải chốt chặn.
- **Test đáng giá nhất của bước này là một TRẬN THẬT**: `TestADraftedPairFightsAWholeBattle` đưa cả hai squad qua `Take` vào `battle.New` rồi chơi hết — 3v3 ally thắng 124 lượt, 5v5 enemy thắng 140 lượt (seed 11, đo lại sau #309 — 5v5 từng là 139 ở pool 16). ⚠️ Con số này **log chứ không assert**, và #309 là bằng chứng đó là lựa chọn đúng: assert "139" sẽ đỏ vì nội dung đổi trong khi hành vi không đổi — xem [[hexarena-pin-the-assertion-not-the-premise]]. Trước bước này draft sinh ra thứ **không đánh được**, nên test hình dạng sẽ không biết là nó dùng được hay không.

Chưa có wire message, chưa có room change, chưa có screen (bước 3–5).

**5v5 ẩn**: `hexarena-host` từ chối `-format 5` — cờ đó là **chỗ DUY NHẤT** trong repo chọn format. ⚠️ **`wire.Format5v5` vẫn hợp lệ TRÊN WIRE cố ý** — xoá khỏi `Format.Valid` là **đổi giao thức** (hai peer phải đồng ý trường format chứa được gì dù bên nào đi trước). Test khẳng định **CẢ HAI nửa**; chỉ test lời từ chối thì xanh vào ngày ai xoá hằng số. Hai lý do gỡ ẩn: đọc lại cân bằng ở 3v3, **và** đủ dàn để draft 5v5.

Liên quan: [[hexarena-starter-squads]] [[hexarena-cursor-record]] [[hexarena-side-is-worth-60-points]] [[hexarena-builds-catalogue]] [[hexarena-room-state-machine]]
