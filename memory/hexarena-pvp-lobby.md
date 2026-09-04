---
name: hexarena-pvp-lobby
description: "hexarena PR#260 lobby TUI — PvP CHƠI ĐƯỢC; ⚠️ RWMutex giữ qua chooser deadlock với WRITER không phải reader (test 2 goroutine PASS); digest khung tên sai"
metadata: 
  node_type: memory
  type: project
  modified: 2026-09-03T11:39:16.554Z
---

hexarena PR #260 (`30ea714`, 2026-09-03): `cmd/hexarena-tui` vào phòng, đánh trận qua LAN. **PvP chơi được thật** — trận 2 người qua socket thật, 0 phân kỳ, 0 lỗi digest.

**Why:** đóng cả hai thứ "ship chết" — transport không có cửa vào, và 13 câu chữ chưa ai đọc.

**How to apply:**

⚠️ **`RWMutex` giữ qua callback KHÔNG deadlock với reader — nó deadlock với WRITER kế tiếp.** Go xếp writer đang chờ **trước** reader mới, nên `Receive` kẹt sau read lock đang giữ, và bộ vẽ kẹt sau writer. **Test 2 goroutine PASS**; phải 3 goroutine + chờ có biên mới thấy. Plan viết "deadlock ngay lượt đầu với chính nó" là SAI.

**Vòng chặn + message loop**: `Play` giữ NGUYÊN, chạy goroutine riêng; chooser chặn trên channel `Update` bơm. **2 nhánh select**, nhánh huỷ được `defer sess.leave()` trong `run()` bảo đảm — tiến trình không rời `run()` mà không cancel. Cấu trúc, không phải cẩn thận. ⚠️ **Channel sức chứa 1 chứ không 0**: chooser gửi "tới lượt" TRƯỚC khi vào select, channel không đệm sẽ rơi `default` và **mất phím thật**; drain đầu mỗi lượt chặn ô đó sống quá lượt.

**2 bộ lái 1 màn hình**: `PlayScreen` xuống làm renderer ở chế độ `Live`, giữ **cursor riêng**, đọc bằng `Since` chứ không `Drain` (Drain **ghi** `b.drained`, read lock không mang được write). 7 chỗ phím có guard, `Action` kind thứ 7 (`Answer`) chở quyết định ra. Đường hot-seat nguyên vẹn — **đo bằng `cmd/hexforge-tui` golden y byte**.

⚠️ **2 bug tìm bằng CHẠY, không phải bằng test**: (1) digest thư mục khung theo tên **đã cắt** tiền tố còn embed khung theo tên **đầy đủ** → mọi thư mục chưa sửa đọc thành **đã sửa**; hai digest đều đúng dạng, đều ổn định, cả suite xanh. Sửa bằng wrapper `fs.FS` chỉ trim khi `Open`. (2) `Stepped` **không** gọi cho welcome — welcome tới trong bắt tay `Dial`, trước khi `Play` tồn tại.

⚠️ **Ra khỏi hộp join BỊ TỪ CHỐI**: mọi đội trong `squads.json` là **2 unit**, phòng chỉ có 3v3/5v5 → **không format nào đội có sẵn vào được**. `squad_refused` là thứ người chơi thấy cho tới khi tự dựng đội. Không phải lỗi — là chữ mới làm đúng việc; `squads.json` là catalogue của người chơi nên ship nội dung vào đó là quyết định của user.

**Mũi tên KHÔNG bị cấm**: 43 dòng mỗi sách đã dùng `↑/↓ ←/→` (kể cả `%d → %d`). Luật trong [[terminal-ambiguous-width-glyphs]] là về lớp `⌘⇧⇄` và sơ đồ. Tôi từng ra chỉ thị "no arrows" — quá rộng, coder đẩy lại đúng.

⚠️ **Rebase: keep-both mù có thể nhét code QUA BIÊN HÀM.** 9 dòng `screens[...]` rơi vào giữa `theForkedBrowser` (hàm trả `model`) → `undefined: screens`. Đọc vùng xung đột trước khi gộp; golden thì **dựng lại bằng `make golden`**, không gộp tay.

Còn (PR 2): countdown + nhánh select thứ 3 (peer chết đúng lúc mình bị hỏi — `Play` đang ở trong `Decide` chứ không ở `conn.read`, không gì đánh thức được; `esc` vẫn thoát); `playFit` đo lại; gloss lý do pass timeout; rejoin; ghi match ra `battle.Log`.

⚠️ `-race` trên `cmd/hexarena-tui` tốn **+29s**, gần gấp đôi gate — do các sweep vẽ mọi màn 2 tiếng 2 cỡ, không phải 4 test trận (~4s). Giữ không lọc `-run` cố ý.

Liên quan: [[hexarena-protocol-wordings]] [[hexarena-host-binary]] [[hexarena-socket-transport]] [[hexarena-pvp-plan]] [[git-checkout-discards-to-head]]
