---
name: hexarena-room-registry
description: room.Registry = 1 goroutine/room; request là VALUE không closure; mutex giữ map thôi; reachability walk > per-function; -race vào make check; sync rời import ban
metadata:
  type: project
---

`room.Registry` (PR #244, merged `b63ed8b`) — nửa "many rooms", key theo room
code. Là **concurrency quanh room và không gì khác**: không socket, không đồng hồ,
không log writer, không spectator. **1 goroutine mỗi room** đọc 1 channel.

Sống **trong `internal/room`** chứ không package riêng, và cả 2 lý do đều máy móc:
`TestTheRoomReadsNoClock` đi bộ đúng thư mục package → registry nằm cạnh thì
thừa hưởng ban đồng hồ miễn phí; và `*Room` không cần export → invariant được
**cưỡng chế** thay vì chỉ nhờ vả. Registry **cũng không đọc đồng hồ**: nó forward
`TimedOut`, không bao giờ khởi timer ([[hexarena-room-state-machine]]).

**"Chia sẻ với KHÔNG GÌ" được giữ 3 cách:**

1. ⚠️ **Request trên channel là VALUE, không phải closure.** `func(*Room)` là
   thiết kế trông gọn gàng mà **phá nát chính invariant** — nó cho caller giữ
   con trỏ nó vừa được đưa. Goroutine switch trên kind; `*Room` là **tham số của
   `serve`**.
2. **Không method nào của `Room` được chạm mutex / channel / goroutine** — kiểm
   **theo RECEIVER**, nên đúng bất kể method viết ở file nào.
3. **Không hàm nào chạm mutex của registry được VỚI TỚI một channel send.** Mutex
   giữ **map và không gì khác**: tra code → **NHẢ** → rồi mới gửi. Giữ khoá qua
   lúc gửi thì **mọi room trong process xếp hàng qua 1 khoá** — giữ đúng chữ của
   luật mà mất hết ý.

⚠️ **(3) là walk theo REACHABILITY, không phải per-function, và khác biệt ĐO
ĐƯỢC.** Mutation của tôi: cho `Deliver` lock rồi chỉ **gọi** `ask` → walk bắt và
nêu lý do; **per-function check thấy lock, không thấy send, và bỏ lọt hoàn toàn**.
Mutation kia (giữ mutex qua send trong `ask`): walk bắt trong 0.5s, VÀ behaviour
suite **deadlock 90s** — cùng một lỗi với chẩn đoán tệ hơn nhiều.

⚠️ **`sync` RỜI import ban — đó là NỚI một guard, nên đọc lý do.** Nó ở trong ban
với ghi chú "registry giữ mutex" — viết khi registry còn tưởng sẽ ở package khác.
Nó về **chính đây**, nên ban sẽ từ chối đúng cái file nó được viết ra để nhường
chỗ. Claim sống sót **SẮC HƠN** qua (2), vì import ban phạm-vi-package **không
nói được TYPE NÀO được lock**. Tôi đo chứ không tin: đặt mutex lên `Room` + lock
trong `Room.Skipped` → `TestNoRoomMethodTouchesTheMutex` đỏ, nêu file/dòng/method.
Dòng log của ban từng ghi "no clock and no goroutine" — nửa sau hết đúng, đã sửa.

**`-race` VÀO `make check`** (dòng thứ 4, chỉ `./internal/room`). Đo: **~3.4s** so
với gate ~60s. Đây là chỗ DUY NHẤT có concurrency trong repo, và race ở đây =
battle thôi tái tạo từ seed → mất log format + `--verify` + undo. **Test race
không ai chạy thì không phải lưới.**

**`wire.CodeRoomUnknown` thôi ship chết**: khai báo rồi, `gate.go` gọi nó là "của
registry", và **không gì gửi** cho tới PR này.

**2 thứ thiết kế bắt buộc mà brief không nêu:**
- **Mọi input trả kèm `Reading` của room**, vì room **rút entry ngay khi match
  xong** → transport hỏi sau là hỏi về room đã biến mất, và **kết quả mọi match
  sẽ không với tới được**. `Read` chỉ cho room còn chạy. ⚠️ `Pending` CỐ Ý không
  có trên reading: `*battle.Prompt` là con trỏ vào state của room — lỗ thật cho
  rejoin, ghi trên type.
- **`Wait` chờ MỌI room** → deadlock trên room mở mà không đánh xong (bản test
  đầu treo 600s). Đó là `Wait` đúng như tài liệu, và test sai.

⚠️ **XUNG ĐỘT ghi lại chứ KHÔNG quyết:** "code chở địa chỉ của chính nó" +
"1 process nhiều room" **không cùng đúng với 1 listener** — mọi room encode cùng
addr:port, 10 ký tự sẽ đặt tên cho PROCESS chứ không phải room. Đường rẻ hơn:
**1 listener mỗi room** (wire format nguyên, `messages.golden` không dời, code vẫn
10 ký tự). Đổi `wire.RoomCode` thì dời cả 3. Ghi là **một CÁCH ĐỌC, không phải
quyết định** — cấp port là I/O, registry không có, socket mới là nơi câu trả lời
đáp xuống. `Open` nhận code được đưa, từ chối trùng.
