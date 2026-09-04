---
name: hexarena-protocol-wordings
description: "hexarena PR#255 chữ cho 10 refusal + 3 closure; ⚠️ tên key trùng identifier module = MIỄN TestNoKeyIsOrphaned (33/577 đang dính)"
metadata: 
  node_type: memory
  type: project
  modified: 2026-09-03T09:22:08.169Z
---

hexarena PR #255 (`c3c935e`, 2026-09-03): `internal/i18n/protocol.go` — `Lang.Refusal(name)` / `Lang.Closure(name)`, 13 key × 2 sách. Đóng một bậc của "ship chết" mà PR #253 vừa nới thêm (`ClosureStopped`).

**Why:** `socket.Client.Refusal.Error()` in `bad_password` thẳng ra màn hình. Chữ phải nói **chuyện gì xảy ra + làm gì**, không phải dịch enum — người đọc đang kẹt ở lobby, không có thông tin nào khác.

**How to apply:**

⚠️ **TÊN KEY i18n trùng identifier bất kỳ trong module = MIỄN `TestNoKeyIsOrphaned`.** Test walk **chữ identifier trần** toàn module (`ast.Ident`, `parser` mode `0` nên **comment không tính**). Đo được, cùng một key orphan thật: tên `ClosedStopped` → **FAIL**; tên `ClosureStopped` → **ok** (vì `wire/closed.go:73` khai identifier đó). Nên key đặt `Closed*` không phải `Closure*` — gánh việc, không phải thẩm mỹ.
**Bề rộng lỗ đo được: 33/577 key hiện có** trùng tên với identifier **không qualifier** ở nơi khác (`i18n.NoteEdited` ↔ `forge.NoteEdited`, họ `SkillField*`, `OriginField*`). Không cái nào chết hiện tại, nhưng mỗi cái xoá chỗ dùng cuối vẫn xanh. Cách siết: bắt ident phải là `Sel` của selector qualifier `i18n`. CHƯA LÀM.

**Thiết kế theo tiền lệ `Lang.StatusCategory`**: accessor nhận **`String()` của enum**, không nhận typed value → `internal/wire` KHÔNG vào import production của i18n. Test (test-only import wire) đi hết `CodeCount`/`ClosureCount` — đó là chỗ giá trị thêm sau mà thiếu chữ thành FAIL.

**Một họ chữ, không hai.** Status có 2 họ (predicate + noun) vì hai câu thật sự khác. Ở đây 1 consumer và nó chưa tồn tại → họ thứ hai chỉ nhân đôi mặt phẳng không ai đọc.

**4 test**: worded cả 2 tiếng · không phải spelling của CHÍNH nó · không phải spelling của BẤT KỲ enum nào (gộp cả 2 enum, chúng dùng chung `none`) · **không hai giá trị cùng một câu** (test tiền lệ không có — 10 từ chối mà 2 câu giống nhau là vô dụng, copy-paste là cách viết bug đó dễ nhất).

⚠️ **Chữ đủ nhưng CHƯA AI ĐỌC, và `TestNoKeyIsOrphaned` không thấy được** — key được chính accessor nhắc là "đã dùng". Màn pairing `cmd/hexarena-tui` mới làm hiện. Không test nào kiểm được **câu có đúng với code nó tả không** — hai hàng hoán vị qua cả 4 test; doc comment của wire là bản ghi.

**4 câu theo doc comment chứ không theo brief**: `room_unknown` phủ cả **host khởi động lại** (mã chở địa chỉ nên địa chỉ trả lời không chứng minh gì), không được khẳng định trận đã xong; `unknown_message` nói "hai bản khác nhau" chứ không "bản bạn cũ" — phía bị từ chối thì **phòng mới là bên đi sau**.

**GitGuardian SUCCESS** ở PR này — xác nhận filepath exclusion `**/*_test.go` gắn hexarena chạy. → [[gitguardian-dashboard-not-repo-file]]

Liên quan: [[hexarena-tui-i18n]] [[hexarena-wire-protocol]] [[hexarena-host-binary]] [[hexarena-pvp-plan]]
