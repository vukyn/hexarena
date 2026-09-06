---
name: a-heading-is-not-a-rule
description: "CLAUDE.md nạp mỗi phiên: tách theo câu RÀNG BUỘC chứ không theo tên heading; đo bytes TỪNG MỤC trước khi tách"
metadata:
  node_type: memory
  type: feedback
---

**Tách tài liệu nạp-mỗi-phiên (`CLAUDE.md`) phải đo theo mục, và cắt theo câu ràng buộc chứ không theo tên heading.**

Đợt tách 2026-09-05 dời 3 mục ra `docs/` và để lại **149.851 bytes**. Hai ngày sau: **161.139**. Đo lại từng mục:

| mục | bytes |
|---|---:|
| `## The layer rule` | **64.273** |
| `## Invariants worth knowing before editing` | 40.698 |
| `## Data and golden files` | 39.216 |
| 6 mục còn lại cộng lại | ~15.000 |

⚠️ **Luật thật trong "The layer rule" chỉ ~30 dòng** (no float / no clock / no rng / no map-iteration / no filesystem). 860 dòng còn lại là chất liệu front-end — bubbletea v2, i18n, `frame`, picker, squad builder, budget màn battle, bộ lọc skill. Riêng **một** khối bullet, *Where a form beats a prompt*, là **42.640 bytes = 27% CẢ FILE**.

**Vì sao nó sống sót đợt tách đầu: nó nằm dưới một heading nghe có vẻ ràng buộc.** Đợt đầu duyệt theo tên mục, thấy "The layer rule" là luật nên bỏ qua — mà thứ ràng buộc là **câu**, không phải cái tiêu đề trên nó.

**Cách làm (2026-09-07, `CLAUDE.md` 161KB → 70KB, không xoá chữ nào):**

1. Đo bytes từng `## ` (script 10 dòng), rồi đo tiếp từng đoạn bold-led bên trong mục lớn nhất.
2. Cắt: câu ràng buộc **ở lại**, lý lẽ + số đo **đi**. Ở đây là `docs/screens.md` (60KB) và `docs/goldens.md` (31KB).
3. Để lại con trỏ ⚠️ nói **cái gì** đi đâu và **khi nào** phải đọc.
4. Cập nhật bảng "moved to" trong § *Where the rest of the record lives*.
5. ⚠️ **Sửa tham chiếu ngược**: `grep -rn "CLAUDE.md" README.md TODO.md docs/ memory/ | grep "§"`. Lần này có **9 chỗ** trỏ tới mục vừa dời (TODO.md ×5, docs/decisions.md ×4), cộng 2 chỗ "below"/"above" vắt qua biên.
6. **Chứng minh không mất chữ**: mọi dòng của bản cũ phải có mặt trong hợp 3 file mới. Còn sót đúng 13 dòng — là 13 dòng cố ý viết lại. Đây là bước không được bỏ.

⚠️ **Heading mới thêm thì đoạn bold-led cũ GIỮ NGUYÊN** (dù lặp ý), vì xoá nó là làm câu "không viết lại gì" thành sai. Ghi rõ điều đó trong header file mới.

Xem thêm [[hexarena-core-design]].
