---
name: hexarena-socket-transport
description: internal/socket = coder/websocket boundary + mirror + đồng hồ; không gì trên wire nói tới lượt ai nên client không mỏng hơn mirror; lowercase code cần re-encode
metadata:
  type: project
---

`internal/socket` (PR #251, merged `bb289cf`) — biên WebSocket: server là
`http.Handler`, client `Dial`, **mirror** client chạy, và **đồng hồ**. Room +
wire vẫn từ chối import `time` (AST walk giữ) vì *ai giữ transport thì giữ đồng
hồ*; đây là chủ.

**Lib: `github.com/coder/websocket`** — đo 9/2026: gorilla **0 commit từ
2025-09**, release cuối **v1.5.3 6/2024**, kéo `golang.org/x/net`. coder: **11
commit năm qua, v1.8.15 (2026-06-15), go.mod KHÔNG có block `require`**,
`context.Context` hạng nhất, ghi song song được, qua autobahn.
⚠️ **Số version ĐI NGƯỢC qua lần đổi path**: `nhooyr.io/websocket` dừng ở
**v1.8.17 (2024-08)**, path mới đang **v1.8.15**. `nhooyr.io` là đường cụt.
⚠️ Trung thực: `golang.org/x/net` **vốn đã** là indirect qua bubbletea → chọn
coder mua được **không thêm cạnh direct**, chứ không phải vắng x/net.

⚠️⚠️ **KHÔNG GÌ TRÊN WIRE NÓI TỚI LƯỢT AI.** `wire.Turn` chở decision + digest;
client tự suy "đang hỏi tôi" từ battle nó replay. Cố ý — message "tới lượt bạn"
sẽ là khai báo thứ hai về state mà mirror vốn tự tính. Hệ quả: **không client nào
mỏng hơn mirror được**, nên **e2e test không tồn tại được mà thiếu mirror** →
mirror phải vào cùng PR transport dù nó nằm mục *The client* trong TODO.

**Mirror**: `battle.New` off seed+roster từ `wire.Start` → `Replay` MỘT decision
mỗi lần với fallback nil → `Since` → `wire.DigestEvents` → so. Lệch thì **kêu
ngay lượt đó**. Giới hạn lượt lấy từ `Welcome.TurnCap` (lý do field đó lên
welcome).

⚠️ **Code đi trong URL `/room/{code}`**, không trong message — code là **địa
chỉ**, không phải nội dung protocol (vì thế nới lên 12 ký tự không dời golden).
⚠️ **`roomOf` decode RỒI ENCODE LẠI, và đó là chịu lực**, đo được:

| dán | decode | là key của map |
|---|---|---|
| `YCUACMRDFADQ` | nhận | **có** |
| `ycuacmrdfadq` | **nhận** (fold toàn phần) | **KHÔNG** |
| `ycuacmrdfadb` | **từ chối**, nêu code đúng | — |

Thiếu re-encode → **người dán code chữ thường bị bảo "room không tồn tại"** trong
khi room ngồi ngay đó. Hai luật của #248 và #251 **không trái nhau**.
Code **không decode được** thì KHÔNG bị transport từ chối — đưa nguyên cho
registry, là key của không room nào, ăn `wire.CodeRoomUnknown` của chính registry.
Một khai báo, không hai.

**Đồng hồ**: `allowanceOf` là phép đổi duy nhất từ `Reading.Config.Allowance`
(giây, int) sang duration. `TestTheTransportOwnsTheClockAndPrintsNothing` là
**khẳng định DƯƠNG** — fail nếu KHÔNG file nào ở đây đọc đồng hồ, vì hai ban
per-package không thấy được đồng hồ dời sang package thứ tư.

**`DefaultCloseThreshold` = 60s**, cấu hình được, chọn theo hai biên: rộng với
hiccup (TCP chịu hàng chục giây mà socket không hay), và **DƯỚI** allowance 90s —
máy chết giữa lượt thì kết trận abandoned chứ không nghiền một timeout mỗi lượt
tới khi bàn giết hết con đang pass. Nó chỉ gác peer **im lặng** (process thoát thì
gửi FIN, read fail ngay) → liveness là ping 15s và **không có read deadline nào**,
vì người đang nghĩ thì không gửi gì suốt cả allowance.

⚠️ **Timer nổ khi câu trả lời đang bay là BÌNH THƯỜNG.** `Room.TimedOut` từ chối
seat không tới lượt; transport **không được** đọc đó là lý do đóng. Sai = **đá
người chơi vì họ trả lời NHANH**. Đúng **một** test cả repo bắt:
`TestALateTimeoutIsRefusedWithoutDroppingAnybody`.

⚠️ **Một bound phải bound cái ĐÃ ĐO.** Read limit từng biện minh "5v5 `wire.Start`
gần 32 KiB nên 1 MB an toàn" — framing đó do brief của tôi khuyến khích. Đo:
**2.911 byte**. 1 MB là **360×** nhiều hơn mức một peer được đòi. Giờ 64 KiB, test
giữ **cả hai đầu**.

⚠️ **`ended()` không biết `net.ErrClosed`** → client tự đóng lại báo lỗi cho trận
nó vừa rời. `context.DeadlineExceeded` **cố ý không** trong danh sách đó: deadline
duy nhất ở đây là write timeout, quá nó nghĩa là peer đã thôi đọc.

`-race` thêm cho `internal/socket` (~1.4s) cạnh dòng `internal/room`. E2e có
**lưới vacuity**: hai client phải so **cùng số** digest và **≥ 30** → phép so
không chạy bị bắt bằng đếm.
⚠️ Coder báo gate bước đầu **gofmt bẩn** và **không chạy lại** — tôi chạy lại cả
gate: 29 ok. Nhớ luôn tự chạy.

Còn: **binary host** (chỗ quyết flag + in code), TUI (`pairing()` là hàm duy nhất
đổi), wordings (`wire.CodeCount`=10 + `ClosureCount` chưa client nào chuyển thành
chữ). → [[hexarena-pvp-plan]] [[hexarena-room-registry]]
