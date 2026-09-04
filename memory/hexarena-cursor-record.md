---
name: hexarena-cursor-record
description: append-only event record + Since(cursor); uncapped view corrupts BOTH ways; one test in the whole repo catches it; nil-on-empty read by nobody
metadata:
  type: project
---

PR #236 (merged `9b0a594`). `Drain` **xoá buffer** → battle chỉ có 1 consumer.
Room PvP có 2 người + spectator + log, nên log event giờ **giữ suốt đời battle**:

```go
func (b *Battle) Drain() []Event                 // hành vi KHÔNG đổi, 261 call site không động
func (b *Battle) Since(cursor int) ([]Event, int)
func (b *Battle) Recorded() int
```

`Drain` giờ **được hiện thực NHƯ MỘT** `Since` consumer mà battle giữ cursor hộ →
1 phễu, nên luật cap và luật nil-on-empty mỗi cái có đúng 1 khai báo, không lệch
nhau được. Cursor ngoài phạm vi → **panic** (cách `rng.Intn` đọc bound sai; 2 panic
đó là 2 panic DUY NHẤT trong `internal/core`). Trả slice rỗng sẽ làm consumer đã đi
*trước* battle trông y hệt consumer đúng nhịp — chính cái desync âm thầm mà cursor
tồn tại để chặn.

⚠️ **Mọi view là slice 3 CHỈ SỐ `b.events[c:n:n]`, và đó không phải sự cẩn thận
suông.** View không cap dùng chung spare capacity → **hỏng CẢ HAI chiều**, đo được:

| | caller giữ | record giữ |
|---|---|---|
| `b.events[from:]` | `[1 2 3 42]` — số 999 nó vừa append BỊ XOÁ | `[1 2 3 42]` |
| `b.events[from:n:n]` | `[1 2 3 999]` | `[1 2 3 42]` |

`append` của caller ghi vào ô mà `emit` kế sẽ dùng, rồi `emit` **ghi đè lên giá
trị caller vừa append**. **Cả 2 client đều nằm trên đúng đường đó**
(`internal/screen/play.go:341`, `cmd/hexarena/main.go:193` — gán slice đã drain rồi
append vào nó sau).

⚠️ **Bỏ cap → CẢ REPO chỉ 1 test đỏ**: `TestAViewAndTheRecordSurviveEachOthersAppends`.
Test đó là toàn bộ lưới cho một lỗi cả 2 client đều đứng lên. Nó cũng chỉ **với
tới được** khi `cap > len` — mà caller không hỏi được capacity của người khác — nên
nó assert capacity trực tiếp VÀ quét 10 độ dài record với 1 biến đếm `t.Fatal` nếu
sweep không quan sát được gì (append growth khấu hao → đọc 1 lần là tung xúc xắc
đeo mặt assertion).

⚠️ **`Drain` trả `nil` (không phải slice rỗng) khi không có gì mới được giữ vì GIỮ
LÀ MIỄN PHÍ, KHÔNG phải vì có ai đọc.** Đổi thành `[]Event{}` → đỏ **3 test, cả 3
đều mới**, và **0** trong vài trăm caller cũ (họ đều lấy len hoặc range). Biết điều
này trước khi ai coi nó là chịu lực.

⚠️ **Một test panic thì che phần còn lại của lưới.** Mutation "`Drain` không tiến
cursor" thoạt đầu chỉ hiện 1 failure, vì test `Errorf` chênh độ dài rồi index quá
cuối slice ngắn → panic, huỷ cả package. Sửa thành `Fatalf`: lưới thật là **10+
test**, phần lớn là test CŨ drain trong vòng lặp và sẽ thấy event lặp lại.

Follow-up chưa làm: cả 2 client + `cmd/hexarena` vẫn tự tích slice event riêng để
dựng `Log` → record giờ là **bản copy thứ 2** của byte process đã giữ. Consumer của
room nên giữ cursor và đọc `Since` chứ đừng tích. Engine không cần cơ chế gì (296 B
/event, ~16 KB cho 1v1 xong). → [[hexarena-pvp-plan]] [[hexarena-wire-protocol]]
