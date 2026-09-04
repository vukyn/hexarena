---
name: hexarena-pvp-plan
description: hexarena PvP LAN 3v3/5v5 — mirror client, series bo1|bo3 (KHÔNG bo2), mã room chứa địa chỉ, tách màn hình trước; 3 số version, 3 invariant
metadata:
  type: project
---

PR #193 (`dbb0270`) + **#196 (`c576d7c`) sửa hai chỗ #193 ghi sai**, 2026-08-31 — **thiết kế, chưa có code**. `README.md` § PvP over a LAN giữ lý do; `TODO.md` § Not done giữ danh sách 5 nhóm theo thứ tự phụ thuộc.

## Bốn quyết định (user chốt, đừng nêu lại)

1. **Client = mirror**, tự chạy engine. `battle.New(books, seed, rosters)` + `Replay([]Decision{d}, 1, nil)` mỗi decision server gửi → tự có events + prompt kế. Wire ~5 message, **không sinh kiểu State nào**. Server kèm event digest → lệch là kêu NGAY. Giá: fog of war mất vĩnh viễn; data hai bên phải khớp tuyệt đối.
2. **Match = một SERIES room tự cấu hình: `bo1` hoặc `bo3`. KHÔNG có bo2.**
   ⚠️ Chỉ series **CHẴN** triệt được phe, và chỉ series chẵn **phải bịa** luật cho tỉ số cân. Không có series nào có cả hai:

   | | trận | triệt phe | phải bịa luật |
   |---|---|---|---|
   | bo1 | 1 | không | không |
   | bo2 | 2 | **có** | **có** |
   | bo3 | 3 | không (dư 1) | không |

   ⚠️ **bo3 tệ hơn nó trông**: ở 1-1 trận 3 quyết cả match → series lẻ **dồn** lợi thế phe vào trận quan trọng nhất chứ không xoá. Và 1-1 sẽ thường xuyên (slot áp đảo → mỗi bên thắng trận mình làm ally).

   Bỏ bo2 xoá luôn metric "tổng tỉ lệ máu còn lại" mà #193 đề xuất → **không còn metric bịa nào trong toàn bộ thiết kế**. Trận thứ 3 không cần biện minh gì; một số chưa đo định đoạt cả match thì cần.

   **bo1 và trận 3 của bo3 là CÙNG một bài toán** → một luật: seed chọn phe + luân phiên dẫn thế hoà theo nhóm tốc độ (miễn phí, xem [[hexarena-side-is-worth-60-points]]). Nói thẳng là không triệt được.

   ⚠️ **Kiến trúc, chỗ tốn tiền nếu bỏ sót:** room giữ `battles: N` + luật kết thúc series. **bo1 KHÔNG phải ca đặc biệt, nó là `N = 1`.** Build "bo2" rồi tổng quát hoá sau là thứ tự đắt.

   **Độ dài đo được** (3v3 ship, AI hai bên, 8 seed): **34–55 lượt quyết định/trận**, 17–28 mỗi người. @15s ≈ 11 ph/trận, bo3 ≈ 34 ph. @đủ 90s = 68 ph/trận, **bo3 = 3.5 tiếng** → lập luận về **90 giây**, không phải về bo3. Hạn thời gian nên vào cfg room cạnh thể thức.
3. **Mã room = base32(4-byte IP + 2-byte port)**, 10 ký tự. Dán mã là vào được, không cần discovery. Password là cửa riêng.
4. **Binary mới**, tách màn hình tham chiếu ra khỏi `cmd/hexforge-tui` **trước**. ⚠️ 10k dòng production dưới **13.7k dòng test** — test là nửa khó, `everyScreen` là harness.

## Ba số version, ba câu hỏi khác nhau

| Số | Trả lời | Lệch |
|---|---|---|
| protocol (int) | nói chuyện được không | refuse |
| build (string) | bảo người ta cập nhật gì | chỉ in |
| **data digest** | **cùng một trận không** | **refuse lúc join** |

**Groundwork XONG 4/4**: màn hình chung ([[hexarena-screen-extraction]]) · data digest ([[hexarena-data-digest]]) · cursor thay `Drain` ([[hexarena-cursor-record]]) · `internal/wire` ([[hexarena-wire-protocol]]). **Room XONG** ([[hexarena-room-state-machine]]) — state machine, không I/O, không goroutine, KHÔNG CẢ ĐỒNG HỒ (timeout là input). **Registry + SOCKET + HOST BINARY XONG** ([[hexarena-room-registry]] · [[hexarena-socket-transport]]). 3 câu hỏi trước socket đã chốt hết ([[hexarena-room-code-widened]]): 1 listener + code 12 ký tự/256 room; rời phòng & timeout **chỉ thông báo, không tính thắng thua**; `TurnCap` lên `wire.Welcome`. Host binary XONG ([[hexarena-host-binary]]) — cổng mặc định **13579** (⚠️ cổng cố định làm test thứ tự hoá RỖNG, phải lái qua `-port 0`), `socket.Server.Shutdown`, `wire.ClosureStopped` (`ClosureCount` 2→3). Wordings XONG ([[hexarena-protocol-wordings]]) — 10 refusal + 3 closure, 2 sách, nhưng **CHƯA AI ĐỌC**. **PvP CHƠI ĐƯỢC XONG** ([[hexarena-pvp-lobby]]) — lobby + waiting + result + battle screen chạy trên mirror; trận 2 người qua socket thật 0 phân kỳ. Ban/pick + spectator ĐÃ CHỐT THIẾT KẾ, chưa làm ([[hexarena-draft-and-spectator-plan]]); 5v5 ẩn. 3 đội starter XONG ([[hexarena-starter-squads]]) — join mặc định KHÔNG còn bị từ chối. Đếm ngược + nhánh select thứ 3 XONG ([[hexarena-countdown-clock-allowlist]]). Việc kế: **TUI** (`pairing()` là hàm DUY NHẤT đổi), — màn pairing là thứ làm chữ hiện lên. Còn sau: ghi match ra `battle.Log` (⚠️ bản ghi phải gồm `turn_began` của lượt bị cap, dừng sớm 1 event thì `--verify` trượt ở phép so số lượng), rejoin cho wifi hiccup (⚠️ tới lúc đó `DefaultCloseThreshold`=60s là dial DUY NHẤT gác cả trận), spectator, alternation của contested speed group (cần phép đo, phe đáng tới 60 điểm).

⚠️ **Một `version` không đủ** — data là 15 file JSON `go:embed` (`internal/seed/seed.go:20`), sửa `power` trong `skills.json` đổi mọi trận mà **không nhích semver**. Hash bytes theo đúng thứ tự khai báo, **không parse** (parse = đọc data lần hai). `assets/` KHÔNG tính: ảnh không với tới mô phỏng.

## Ba invariant ghi trước khi có gì phá được

- **Đồng hồ không vào trận.** Chỉ *quyết định* vào — `Pass` với **một hằng số duy nhất** (`Decision` tự ghi: hai caller dùng hai chữ khác nhau cho cùng một lựa chọn làm replay lệch). Prompt `Skipped` không đếm giờ. Thời gian còn lại đi qua wire là **duration, không phải deadline** (hai máy LAN không có lý do đồng hồ giống nhau).
- **Không có văn bản trên wire** — server gửi **error code**, client dịch. Không thì server quyết định client đọc tiếng gì.
- ⚠️ **`Drain()` xoá buffer, một consumer.** Room có 2 người + spectator + log. Server drain **một lần** vào record append-only, mỗi consumer giữ cursor. Reconnect và spectator giữa trận đều thành "mọi thứ sau index n" — cursor là thứ khiến cả hai gần như miễn phí.

Xử thua / đứt mạng là kết quả **match**, không phải trận → **không thêm gì vào `battle.Outcome`**. Socket đứt không phải một cách trận đấu kết thúc, và đó là kiểu trong `internal/core`.

## Đã có sẵn, đừng build lại

`placement.Squad` = wire format (tham chiếu, **không có stat** → không gian lận được). `Squad.Take` → `cast.ChooseLoadout` = validator loadout. `Advance→Prompt/Act/Pass` = API server cần. `battle.Log` = match record → mỗi trận PvP `--replay --verify` miễn phí. `playScreen` đã giữ đúng state mirror cần (`seed`/`roster`/`script`/`events`/`pending`), 21 chỗ đọc `.fight`.

## Còn mở (danh sách nêu mà chưa trả lời)

- Một đội có được xếp 2 con cùng nhân vật? `Squad.Validate` hiện **cho phép** (chỉ kiểm id + slot).
- Cân bằng ở 3v3 chưa đọc lại (đội hình có màn chắn cân ở 5 người; bàn ngắn hơn cho summon nhiều ô trống hơn).
- Stage phải là **lá** của nhánh, KHÔNG dùng `Furthest` — nó refuse trên fork, và politoed đang xếp hàng làm fork đầu tiên.
- ⚠️ `tui.Line` in `event.Note` **thô** (`tui.go:318`) → `loses the turn (timeout)` cả hai ngôn ngữ. Cần gloss.
- 5v5 body đã đo **28 dòng** trên sàn 24; PvP thêm dòng đồng hồ + dòng chờ → phải đo lại `playFit`.

- ⚠️ **Chưa tìm ra chiêu nào giải quyết theo thứ tự không mirror** — phải xong việc này TRƯỚC khi trích số nào ở 3v3/5v5. Xem [[one-way-mirror-not-a-measurement]].
- `tui.Roster` duyệt theo thứ tự enlist → xen roster làm bảng hiện `A1 E1 A2 E2`; sort theo side khi vẽ.

Đã loại: fog of war, draft/ban, bo2, TLS, NAT/relay. Xem [[hexarena-side-is-worth-60-points]] cho cần gạt thế hoà, [[one-way-mirror-not-a-measurement]] cho giới hạn của phép đo.
