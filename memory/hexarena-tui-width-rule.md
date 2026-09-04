---
name: hexarena-tui-width-rule
description: "hexforge-tui width rule — prose wraps at minWidth, data spends usableWidth(); minHeight is dead for layout; floor raised 80→120 off a measurement"
metadata: 
  node_type: memory
  type: project
  modified: 2026-08-31T08:07:15.046Z
---

**Luật bề rộng của `cmd/hexforge-tui`** (PR #173 + #175, giữ bởi `cmd/hexforge-tui/width_rule_test.go`, chép vào `CLAUDE.md`):

> **Văn xuôi wrap tại sàn (`minWidth`). Ô dữ liệu tiêu cửa sổ (`m.usableWidth()`).**

`minWidth` là bề rộng chương trình **hứa vẽ được**, không phải trần được tiêu.

- **Văn xuôi tại sàn** — đo theo terminal thật thì một câu có hai hình dạng, `TestEveryWordingFitsTheMinimumWidth` không còn gì để bám, và đoạn văn chạy ngang 200 cột là dòng người đọc lạc chỗ.
- **Dữ liệu theo cửa sổ** — gloss/id/path/allowlist bị cắt trên terminal rộng là vứt nội dung không đổi lại gì.
- `width_rule_test.go` giữ **cả hai chiều**: nới văn xuôi cũng đỏ như cắt dữ liệu.

⚠️ **`minHeight` KHÔNG có tác dụng gì lên layout.** Nó chỉ xuất hiện 3 chỗ: khai báo, `tooSmall()`, câu báo màn hình nhỏ. Mọi screen tính chỗ từ `m.height` thật. Nâng `minHeight` hiện thêm **số không** — chỉ từ chối thêm terminal. Muốn thêm chiều dọc phải sửa chi phí chrome / thứ tự bỏ khối từng screen (xem [[hexarena-battle-screen-budget]]).

⚠️ **Nâng sàn là đòn bẩy DUY NHẤT cho footer.** Footer là wording trong catalog; cho nó dùng `usableWidth()` là để nó bị cắt lại trên terminal 80 cột — đúng lỗi sweep sinh ra để chặn (72 cột từng cắt mất phím thoát). Nới ô dữ liệu không đụng được nhóm này: đo trước #173 là 35/92 cặp screen ép sát 79, sau #175 vẫn 34/92.

**Cách đo lại** (worktree throwaway, test tạm rồi xoá): render mọi `everyScreen` ở 200×60 cả hai ngôn ngữ, bỏ dòng khớp `carriesFreeText(line, free + pickerDetails(m))`, lấy dòng rộng nhất mỗi screen. Dòng dừng đúng 78–79 là **dấu vết chữ bị gọt cho vừa**, không phải chữ tự nhiên kết thúc ở đó.

**Sàn 80 → 120 SHIPPED** (PR #177, main `4d5d719`). `minHeight` giữ 24. Giá: terminal dưới 120 cột không chạy `hexforge-tui` nữa; CLI `hexforge` không cần chỗ và màn too-small đã trỏ sang.

⚠️ **Sàn được gọi tên ở 6 file, không chỉ `model.go`** — `CLAUDE.md`, `README.md` ×2, `TODO.md`, `i18n/keys.go`, `i18n/gloss.go`, cộng comment `skillsRoom`. Đổi sàn phải `grep -rn "80x24\|120x24\|N columns"` cả repo. Phần lớn là khẳng định tính từ **chiều cao** ("hai mươi dòng thân") nên số học không đổi, chỉ sai tên — trừ `GlossedKit`, nơi "năm cặp ngoặc không lọt 80 cột" là phép ĐO có thể đã hết đúng; để nguyên, đánh dấu là bản đọc ở sàn cũ.

⚠️ Rủi ro thật của việc nâng sàn là **chiều dọc, không phải chiều ngang**: văn xuôi wrap tại sàn nên nâng sàn **đổi số dòng nó chiếm**. Đo ra 3 khối đổi: caution `fight.go` **2→1**, note loài `plant`/`turtle` **2→1**, note thứ hai của save **3→2**. Mọi row budget đã rà — không cái nào hỏng; `species.go` `longestNote()` tự bám vì nó đo ở đúng bề rộng lúc render và lấy max (`dragon`/`lizard` vẫn wrap). Giờ ghim bởi `TestEveryFloorWrappedBlockTakesTheRowsItTakes`.
"Sweep lỏng ra 40 ô" **KHÔNG phải lỗi** — hứa 120 thì footer 119 ô là hợp lệ. Đừng dựng máy móc chống trôi cho nó.

⚠️ **Nâng sàn làm 8 fixture cỡ theo sàn cũ thành RỖNG** — và cả 8 đều tự báo bằng lời của chúng chứ không lặng lẽ xanh (thiết kế đó chạy đúng). Rederive mọi fixture từ `minWidth`, đừng hardcode số ô.

⚠️ **Nâng sàn GIẾT một nhánh code**: `Lang.DamageWithin` bỏ cặp tham chiếu khi chật, mà trần số học của dòng đó là **89 ô (vi) / 87 (en)** so với chỗ hẹp nhất **98/99** → không cửa sổ nào chạm tới. Xem [[hexarena-mechanics-log]].

Xem thêm [[hexarena-tui-i18n]].
