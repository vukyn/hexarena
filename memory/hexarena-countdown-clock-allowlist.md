---
name: hexarena-countdown-clock-allowlist
description: "hexarena PR#267 đếm ngược + nhánh select thứ 3; ⚠️ danh sách import ≠ danh sách đồng hồ; 3 clock ban đều mù package thứ 4"
metadata: 
  node_type: memory
  type: project
  modified: 2026-09-03T13:06:54.531Z
---

hexarena PR #267 (`4695849`, 2026-09-03): đồng hồ đếm ngược cả hai bên trên màn trận + **nhánh select thứ ba** của chooser (đóng tồn dư lobby để lại). Đi chung vì dùng chung một đồng hồ.

**Why:** người chơi không biết còn 5 giây hay 50; và peer chết đúng lúc mình bị hỏi thì goroutine kẹt hết trận (`Play` đang trong `Decide`, không trong `conn.read`).

**How to apply:**

⚠️ **DANH SÁCH IMPORT ≠ DANH SÁCH ĐỒNG HỒ.** Tôi đưa coder 5 file dựng từ `import "time"`. `internal/socket/connection.go` lấy write deadline + close handshake qua **`context.WithTimeout`** trên `Timings` dựng chỗ khác — **không import `time` một chữ**. Walk phải soi cả **lời gọi**: `context.WithTimeout/WithDeadline`, `tea.Tick/Every`, helper riêng. Danh sách thật là 6, không phải 5.

⚠️ **3 lệnh cấm đồng hồ đều đọc `os.ReadDir(".")` — thư mục CỦA CHÍNH NÓ.** `internal/room/clock_test.go:157`, `internal/wire/clock_test.go:130`, `internal/socket/clock_test.go:119`. Đồng hồ mọc ở package thứ tư thì **cả ba đều mù**. Đóng bằng `internal/socket/allowlist_test.go` — walk toàn module, giữ tập file đọc đồng hồ **bằng** một allowlist có lý do từng dòng (6 mục / 132 file), + guard `walked N files` để không pass rỗng. Đặt ở socket vì đó là package **sở hữu** đồng hồ.

**KHÔNG thêm message nào — TODO đòi một cái không cần tồn tại.** Bản ghi viết "a remaining duration **on the wire**", viết trước khi mirror thành hình này. Hai peer cùng áp một `wire.Turn`, cùng mở một prompt → **mỗi client tự biết tại chỗ** lượt mở lúc nào và của ai; `Welcome.Allowance` cả hai đã có. Lý do TODO đưa ("hai máy LAN không có lý do đồng ý mấy giờ") chính là lý do đếm-tại-chỗ ĐÚNG chứ không phải nhượng bộ. Màn hình chỉ **tham khảo**; đồng hồ server có thẩm quyền, timeout về dưới dạng pass event như mọi event.

**`internal/screen` vẫn không đọc được đồng hồ** — nhận **hai con số giây**, vẽ. Song song với phòng: allowance là số phòng mang theo và đưa đi.

⚠️ **Không có DÒNG đồng hồ — lên heading cạnh `logPosition`.** Một dòng riêng tốn **3 dòng mỗi lần vẽ chứ không phải 1**: log ăn mọi dòng phía trên không đòi → mất 1 dòng lịch sử + dời khung đọc + đổi khoảng trên heading. Golden: 0 thêm 0 xoá, **đúng 1 dòng mỗi block** (8 và 12).

⚠️ **Nhánh thứ ba làm một câu chữ đã ship thành SAI.** `RefusalNotYourTurn` (#255) viết "lỗi của chương trình chứ không phải của bạn" — đúng khi cách duy nhất gây ra là peer hỏng. Nhánh 3 gửi **pass muộn thật** → người chỉ hết giờ bị bảo chương trình hỏng. Client-sent late pass đi qua `deliver` chứ không `timedOut`, nên `refusedAlone` **không nuốt**. Tìm ra lúc **smoke, không phải bằng suy luận**. Sửa chữ, không nuốt refusal (nuốt = client đoán mò một refusal chính nó gây ra).

**Grace = 2s.** Client đã là bên thứ hai theo cấu trúc (phòng lên giờ khi **sinh ra** lượt, client đếm khi **nhận được**), nên grace chỉ phủ drift nhịp đồng hồ (~10ms/90s) + timer thô trên máy tải. Test chặn **cả hai biên** — bỏ nhánh và bắn tức thì đều đỏ.

Liên quan: [[hexarena-pvp-lobby]] [[hexarena-protocol-wordings]] [[hexarena-socket-transport]] [[hexarena-room-state-machine]]
