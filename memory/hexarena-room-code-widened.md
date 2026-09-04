---
name: hexarena-room-code-widened
description: room code 7 byte/12 ký tự, 1 listener; base32 bit thừa = nhiều code ra 1 room (4→16); logic race không phải data race nên -race không thấy
metadata:
  type: project
---

PR #248 (merged `920504a`) + #246 (`1038262`). Chốt câu hỏi cuối trước WebSocket.

**Code = 4 addr + 2 port + 1 room = 7 byte = 12 ký tự.** `RoomCodeLength` 12,
`RoomsPerProcess` 256, **một listener** mỗi process. Lời hứa "10 ký tự" **rút
chính thức**.

⚠️ **Nới code KHÔNG phải đổi wire format**: không body message nào chở
`RoomCode`, nên `messages.golden` **không dời một dòng**. Code là thứ *người ta
dán để nối*, không phải field trong protocol.

**Vì sao 1 listener chứ không 1 port/room** (ghi để không tranh lại): port là tài
nguyên OS hữu hạn cần lỗ firewall; mỗi room chết **rò 1 port**; nó trộn "room"
(khái niệm app) với "listener" (OS) → registry key theo code bị registry thứ hai
key theo port che; và nó **kéo lifetime socket vào lifetime room**, mà registry
không có I/O đúng để test được. Port-mỗi-"room" chỉ xuất hiện ở nơi mỗi room là
**cả một process** (Quake, CS, Agones) — kiến trúc ngược lại.

⚠️⚠️ **BIT THỪA CỦA BASE32 = NHIỀU CODE RA CÙNG MỘT ROOM.** 5 bit/ký tự, nên
7 byte = 56 bit trong 12 ký tự chở 60 → **4 bit thừa** (6 byte có 2).
`encoding/base32` **KHÔNG có `Strict()`** (base64 có) nên bit đó bị bỏ qua im
lặng. Đo `192.168.1.50:9000`:

| bytes | canonical | số string decode ra nó |
|---|---|---|
| 6 (cũ) | `YCUACMRDFA` | **4** |
| 7 (chưa chặn) | `YCUACMRDFADQ` | **16** |
| 7 (đã chặn) | `YCUACMRDFADQ` | **1 nhận, 15 từ chối** |

Quan trọng vì **`Registry` key map bằng chính STRING**: dán biến thể → tra key
không có → bị bảo "room unknown" trong khi room ngồi ngay đó. Chữa: `Decode`
**encode lại và từ chối nếu lệch** → code thành canonical key. Test lấy số kỳ
vọng từ `RoomCodeLength*5 - roomCodeBytes*8` chứ không viết 16.
⚠️ **Đúng MỘT test giữ phép từ chối đó** — xoá nó chỉ
`TestANonCanonicalRoomCodeIsRefused` đỏ, không gì khác cả repo.

⚠️⚠️ **LOGIC RACE KHÔNG PHẢI DATA RACE → `-race` KHÔNG THẤY.** `enrol` phải tìm
byte trống VÀ chiếm nó **dưới MỘT lần giữ mutex**. Chia thành hai lần
(tìm → nhả → chiếm) **sống qua cả suite + `-race` 3 lần**: hai lần giữ đều khoá
đúng nên detector không có gì báo, và **mọi `Open` khác trong file đều tuần tự**
nên chưa bao giờ có hai cái đâm vào cửa sổ. Tôi viết lưới
`TestConcurrentOpensAtOneAddressNeverShareAByte`: 8 goroutine thả cùng lúc, 20
round. Khi trúng cửa sổ thì **chính xác**: 2 caller tìm cùng byte, lần
`g.rooms[code] = entry` thứ hai **GHI ĐÈ** cái đầu → map 1 entry mà `live` đếm 2,
1 goroutine mồ côi, 2 caller giữ 1 code. Dưới mutation: *"8 callers … hold 7
distinct codes, want 8"*. **Cửa sổ là xác suất, assertion thì không** → không báo
oan, bắt nhanh chứ không chắc ngay lần đầu.

**`Open(at, config, deps) (RoomCode, error)`** — registry tự cấp byte trống thấp
nhất. Nhận code từ caller trước đây là **phân rã bị bắt buộc bởi câu hỏi mở**
(cấp *port* là I/O); `AddrPort` chỉ là giá trị nên chỗ vụng mất theo câu hỏi.
Trùng code thành **bất khả theo cấu trúc** → `TestADuplicateCodeIsRefused` xoá
tường minh. Room thứ 257 → **Go error**, không phải `wire.Code`: host không mở
được là vấn đề của host, không phải phép từ chối với người join.

⚠️ **Một test pass CẢ HAI CHIỀU** (tệ hơn fail): in-flight test mở room thứ hai
*sau khi* room đầu xong → registry cấp lại đúng byte đó → hai code bằng hay khác
tuỳ retirement chạy tới đâu, và **pass kiểu nào cũng pass**. Có lần nó chỉ có 1
subject thay vì 2. Đã dời `Open` lên và assert 2 code khác nhau.

⚠️ **`net/netip` vào `internal/room` nơi `net` bị ban — qua vì MAY.** Ban theo
đường dẫn chính xác. Giờ có ghi lý do: netip là package của **giá trị**, không mở
gì; ngày nào cần `net.Listen` thì chính `net` xuất hiện và walk từ chối.

Việc kế: **WebSocket**, và nó không còn câu hỏi mở nào.
→ [[hexarena-room-registry]] [[hexarena-pvp-plan]]
