---
name: hexarena-v0-1-0-released
description: hexarena v0.1.0 tag đã push (d56e545); go install khai đúng tag; ⚠️ module path không /vN nên CHỈ tag được v0 hoặc v1
metadata: 
  node_type: memory
  type: project
  originSessionId: 7149af35-2760-4597-8d75-9779b86f6cfe
  modified: 2026-09-04T19:04:01.717Z
---

hexarena **`v0.1.0`** đã tag và push (2026-09-05), tại `d56e545`, digest **`4792c12397c5`**. Repo public. Cài thật từ proxy công khai, xác nhận cả hai binary khai đúng:

```
go install github.com/vukyn/hexarena/cmd/hexarena-host@v0.1.0
go install github.com/vukyn/hexarena/cmd/hexarena-tui@v0.1.0

hexarena-host v0.1.0
  protocol 1
  data     4792c12397c5
```

**How to apply:**

⚠️ **Module path `github.com/vukyn/hexarena` KHÔNG có hậu tố `/vN` → CHỈ tag được `v0.x.y` hoặc `v1.x.y`.** Đo được lúc thử: tag `v9.9.9` bị Go từ chối — `invalid version: should be v0 or v1, not v9`. Muốn `v2` phải đổi module path thành `.../v2` ở go.mod **và mọi import**.

⚠️ **Tag đẩy lên proxy.golang.org là BẤT BIẾN** — không cắt lại `v0.1.0` với nội dung khác được, xoá tag trên GitHub cũng không xoá bản proxy đã cache. Nên **chứng minh trước khi push**: dựng proxy file cục bộ (`GOPROXY=file://…`, layout `@v/<ver>.{info,mod,zip}`, zip **phải** `zip -qrD` không có entry thư mục, rooted `module@version/`) và cài từ một tag giả. Tôi làm vậy trước khi tag thật.

⚠️ **Chọn commit phải hỏi lại nếu main đã dời.** Tôi đề nghị tag `a758a3d`, user "ok", nhưng lúc chuẩn bị tag main đã đi **34 commit** và **digest đã khác** (`f7045f45141b` → `4792c12397c5`) — hai lựa chọn ship **dữ liệu khác nhau** cho bạn bè. Hỏi lại, user chọn mới nhất. Gate full trước khi tag.

**Điều kiện để bạn bè chơi được:** cả hai máy **cùng version** (room so digest 15 file data, khác là từ chối `data_mismatch`). `-version` in digest trên cả hai binary nên so được mà không cần mở phòng. → [[hexarena-crlf-data-digest]] cho ca digest khác vì CRLF.

Chỉ **3v3**; 5v5 chặn ở cờ host tới khi đọc lại cân bằng trên bàn này.

**Cơ chế memory của repo** (user cập nhật 2026-09-05): hai tầng, `memory: project` ở frontmatter agent → `<repo>/.claude/agent-memory/<agent>/` + `pet-platform/.claude/agent-memory/<agent>/`. hexarena: coder **22 note** (phần lớn `feedback_*`), security-scanner 1, explorer/onboarder/planner **0**. ⚠️ Ba chỗ lệch: `committer` là agent DUY NHẤT không có `memory:`; 5 note `project_hexarena_*` nằm ở tầng **platform** trong khi 22 `feedback_*` ở tầng **repo** (agent đọc kho repo không thấy 5 cái kia); `planner` không lưu bản thiết kế lobby dù nó là thứ đáng lưu nhất.

Liên quan: [[hexarena-pvp-plan]] [[hexarena-pvp-lobby]] [[hexarena-starter-squads]] [[hexarena-crlf-data-digest]]
