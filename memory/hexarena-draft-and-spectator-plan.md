---
name: hexarena-draft-and-spectator-plan
description: "hexarena ban/pick + spectator CHƯA LÀM; ban 2/bên ở 3v3 (vừa) 3/bên ở 5v5 (⚠️ thiếu 5 nhân vật); spectator KHÔNG được là chỗ ngồi thứ 3"
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

⚠️ **Pool là 11 chứ không phải 12** — `naruto.naruto` đã `hidden: true`. Với số ban đã chốt: **3v3 = 6 picks + 4 bans = 10/11 → VỪA**, dư 1, pick cuối thấy 2. **5v5 = 10 + 6 = 16/11 → cần THÊM 5 nhân vật**, điều kiện nội dung code không chữa được. (Đọc theo "tổng cả hai bên" thì 8/11 và 13/11 — đổi con số đích, không đổi kết luận.)
⚠️ **Ban TUỲ CHỌN biến thiếu hụt thành lỗi LÚC CHẠY**: phòng hợp lệ khi mở vẫn hết nhân vật giữa chừng draft. Nên bất kể con số, draft nợ một luật: **từ chối ban khi pool còn lại không đủ ngồi hai bên, xám ô đi** — và chính luật đó khiến phép tính này không phải kiểm lại mỗi lần thêm/ẩn nhân vật. ⚠️ **Pick cuối không phải quyết định khi `pool − picks − bans = 0`**.

⚠️ **SPECTATOR KHÔNG ĐƯỢC LÀ CHỖ NGỒI THỨ BA — nó đổi AI THẮNG.** Comment `seatCount` (series.go) nói chỗ ngồi là **mảng chứ không map** vì **thứ tự thăm hai chỗ đi vào roster, và thứ tự roster quyết ai thắng khi hoà tốc độ**. Xâu người xem qua cùng cấu trúc = dịch thứ tự đó. **Không gì trong suite bắt được**: roster vẫn hợp lệ, hai peer cùng có spectator vẫn khớp mọi digest — chỉ so với trận đánh KHÔNG có nó mới thấy. Spectator phải là **hạng công dân khác**: cursor riêng, `Deliver` từ chối luôn, không tồn tại với `seats`/roster/`other()`.
**Nửa đắt ĐÃ XONG**: `battle.Since` + bản ghi append-only (#236). Ba con số 2 chặn: `room/series.go seatCount`, `socket/table.go seatsPerTable`, `wire` chỉ `SeatHost`/`SeatGuest`.

**Draft là state machine thứ hai cùng hình dạng** với `internal/room` — chuỗi quyết định, timeout là **input** không phải đồng hồ → cùng lệnh cấm. **Mẹo mirror chuyển nguyên vẹn**; thứ draft sinh ra là **hai roster** = đúng thứ `wire.Start` đã chở → không gì phía sau draft đổi. ⚠️ Nhưng spectator xem draft cần **bản ghi mà draft chưa có**.

⚠️ **`Hidden` có hai nghĩa, đừng "lọc ở mọi nơi"**: `screen/squads.go` TÔN TRỌNG nó (chọn ai ra trận — draft cũng vậy), nhưng `screen/picker.go:416` **cố ý VẪN hiện** nhân vật ẩn (đó là danh sách `restrict.characters`; lọc sẽ làm một restriction đang tồn tại **không lưu lại được**). Comment của chính flag gọi nó "tiện lợi khi soạn, không phải tuyên bố thiết kế" — cổng draft biến nó thành tuyên bố thiết kế.

**5v5 ẩn**: `hexarena-host` từ chối `-format 5` — cờ đó là **chỗ DUY NHẤT** trong repo chọn format. ⚠️ **`wire.Format5v5` vẫn hợp lệ TRÊN WIRE cố ý** — xoá khỏi `Format.Valid` là **đổi giao thức** (hai peer phải đồng ý trường format chứa được gì dù bên nào đi trước). Test khẳng định **CẢ HAI nửa**; chỉ test lời từ chối thì xanh vào ngày ai xoá hằng số. Hai lý do gỡ ẩn: đọc lại cân bằng ở 3v3, **và** đủ dàn để draft 5v5.

Liên quan: [[hexarena-starter-squads]] [[hexarena-cursor-record]] [[hexarena-side-is-worth-60-points]] [[hexarena-builds-catalogue]] [[hexarena-room-state-machine]]
