---
name: hexarena-room-state-machine
description: internal/room = state machine, no I/O no goroutine no clock; timeout is an INPUT; splitmix64 cannot derive a per-battle seed; leaf==Furthest(cap) provably
metadata:
  type: project
---

`internal/room` (PR #240, merged `ea939f2`) — PvP match là **state machine trên
`internal/wire`**, không khai báo message riêng. 4 input, mỗi cái trả
`([]Outbound, error)`: `Join(hello)` (cổng, trả seat) · `Deliver(seat, body)` ·
`TimedOut(seat)` · `Left(seat)`. 2 fake client đánh hết 1 bo3 in-process **~40 ms**.
Không socket, không goroutine, không mutex.

⚠️ **Timeout là INPUT, không phải phép đọc.** Room không bao giờ hỏi mấy giờ;
ai giữ transport thì giữ đồng hồ và *nói cho nó biết*. Nhờ vậy `time` không được
import (AST walk giữ, giống `internal/wire`), và "3 timeout liên tiếp thì xử thua"
thành **đếm thuần**. `Skipped` prompt không khởi đồng hồ. **Một hành động thật
RESET bộ đếm** — bộ đếm không reset sẽ xử thua một người chơi chậm sau cả trận.

⚠️⚠️ **SPLITMIX64 KHÔNG DẪN ĐƯỢC seed-mỗi-trận từ seed-mỗi-match.** Bản hiển
nhiên — `rng.New(Seed+index).Next()`, tái dùng thay vì viết lại số học — là hàm
của **TỔNG**, vì splitmix64 tiến bằng cách CỘNG hằng số. Nên **trận 2 của seed 6
CHÍNH LÀ trận 1 của seed 7**, chính xác. Tôi đo riêng: **199 trùng / 199 cặp kề**
(`seed 6 b2 = seed 7 b1 = 11409396526365357622`). MỌI generator kiểu counter đều
có hình dạng đó. Dẫn từ 2 số phải là **hàm của 2 số** → `sha256(seed ‖ index)`.
Tìm ra bằng **sweep cặp kề**; 3 giá trị hardcode không thấy được.

⚠️ **`Leaves`/`IsLeaf` ≡ `Furthest(LevelCap)` — CHỨNG MINH ĐƯỢC, không phải may.**
`Line.Validate` từ chối stage có `MinLevel > LevelCap` (progression.go:466), nên
mọi stage của mọi line hợp lệ đều với tới được ở cap, và ở đó "ngọn mỗi nhánh" =
"xa nhất mà cap với tới". **Mutation của tôi thay `Furthest(LevelCap)` vào `IsLeaf`
→ PASS cả 21 test** của predicate + cổng, và KHÔNG test nào viết được để bắt.
Lý do viết ở **4 chỗ** ("sẽ sai cái ngày có stage authored trên cap") mô tả một
ngày KHÔNG THỂ ĐẾN — đã sửa cả 4. Predicate vẫn giữ: nó mua **cái level không
còn nằm trong câu hỏi** (2 đáp án lệch nhau ở MỌI level dưới cap; cổng chỉ hỏi ở
60), + **error** thay vì false khi tên không có trong line.
⚠️ Và spec của tôi sai: **poliwag CÓ fork** (`Poliwrath`/`Politoed` cùng
`after: Poliwhirl`) — tôi lấy từ note `CLAUDE.md` đã cũ.

⚠️ **Không thêm gì vào `battle.Outcome`.** Forfeit/disconnect/refused-join là kết
quả của **MATCH** → `room.Verdict`/`room.Forfeit`, cố ý không gọi là outcome để
không ai viết `battle.Outcome(result.Verdict)`. `battle.OutcomeCount` assert với
**literal 4** — đọc hằng số rồi so với chính nó thì đúng với mọi số.
⚠️ **Trận bị CAP không bị đóng dấu outcome**: giữ `Undecided` +
`BattleResult.Capped`. Dấu `Stalemate` = room viết một outcome engine chưa từng
sinh ra → log sẽ FAIL `--verify` của chính nó.

**Cổng: THỨ TỰ là phần của câu trả lời.** `Version.Check` (protocol trước digest)
→ password (constant-time) → seat → squad (5 luật dưới 1 code: `Validate` → cỡ
format → level 60 → **leaf** → `Take`). Test cho peer sai **2 thứ KỀ NHAU** mỗi
case — cổng không đo thứ tự thì báo lỗi nào nó tình cờ thấy trước.

**1 squad ĐƯỢC dùng cùng nhân vật 2 lần**: `Squad.Validate` cho phép, builder sẽ
ghi ra, và từ chối = từ chối squad người chơi tự lưu vì lý do không màn nào nói.

**`start` chở MỘT roster slice, không phải 2 field** — `seq` theo thứ tự slice
([[hexarena-side-is-worth-60-points]]).

⚠️ `TestNothingHereDrainsTheBattle` tìm **selector** chứ không tìm string: `Drain`
là thứ 261 site khác gọi, nên với tới nó chỉ 1 phím, và nó sẽ lặng lẽ lấy event
của consumer khác ([[hexarena-cursor-record]]).

Mutation của tôi: nói cả 2 seat rằng chúng đánh `SideAlly` → **5 test đỏ** (fixture
dùng nhân vật KHÁC nhau mỗi phe, nên swap thấy được — bài học #231).

Chưa làm: registry + 1 goroutine/room · WebSocket · ghi match ra `battle.Log` ·
spectator · alternation của contested speed group. → [[hexarena-pvp-plan]]
