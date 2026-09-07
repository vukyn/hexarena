---
name: todo-items-are-addressed-by-code
description: "hexarena — mọi mục TODO.md có mã AREA-NNN cố định; gọi việc theo MÃ, không theo số dòng"
metadata:
  node_type: memory
  type: feedback
---

**Gọi một mục việc bằng MÃ của nó, không bao giờ bằng số dòng.**

`TODO.md` § *The codes* giữ bảng vùng và luật. Dạng `AREA-NNN`:

| vùng | viết tắt của | phạm vi |
|---|---|---|
| `ENG` | **eng**ine | luật và số học của engine |
| `RAT` | **rat**ing | `Suggest` + `price.go` — đối thủ chọn gì, tính giá thế nào |
| `PRG` | **pr**o**g**ression | learnset, tiến hoá, placement, build |
| `DAT` | **dat**a | số balance, `internal/seed/data`, golden |
| `CAST` | **cast** | nội dung: nhân vật, tranh, flavour |
| `SCR` | **scr**eens | `internal/screen`, hai client TUI, `internal/i18n` |
| `CLI` | **cli**ent | client không phải hai TUI |
| `FRG` | **f**o**rg**e | `internal/forge` + công cụ `hexforge` |
| `NET` | **net**work | PvP: room, socket, wire, host |

⚠️ **`RAT` là *rating*** — thứ `Suggest` quyết — không phải con chuột, không phải
tỉ lệ. Là vùng duy nhất mà ba chữ đọc lên thành một từ tiếng Anh khác hẳn, nên cột
"viết tắt của" phải có chứ không hiển nhiên.

**Why:** số dòng là dữ kiện về *lượng chữ nằm phía trên* một mục — mà đó chính là thứ
file này lớn lên. Một entry mở ở đây dài tới mười hai kilobyte; chỉ cần ai đó viết
thêm một đoạn là mọi số dòng đã trích trong commit, PR hay `docs/decisions.md` thành
sai, và không có gì báo. Mã thì không đổi.

**How to apply:**

- **Mã cấp một lần, không đổi, không tái dùng.** Xong việc thì mã giữ nguyên (entry
  vẫn nằm trong § *Not done* kèm số đo, chỉ đổi `[ ]`→`[x]`); chuyển mục, từ chối mục
  cũng không đổi. Mục mới lấy số trống kế tiếp trong vùng.
- ⚠️ **Vùng theo CHỦ ĐỀ, không theo file mà bản vá rơi vào.** Lỗi định giá sửa trong
  `internal/core/battle` vẫn là `RAT`, vì thứ người đọc muốn tìm là mọi mục về cách
  đối thủ chọn. Mục vắt qua hai vùng thì lấy vùng của nửa **còn mở**.
- **Mọi thứ mới đều lấy mã, ngay lúc viết ra.** Tính năng và bug đánh số **cùng một
  dãy** — vùng đã nói chủ đề, cột trạng thái đã nói mở hay đóng, thêm tiền tố `BUG-`
  là gọi tên lần thứ ba cho dữ kiện mã và checkbox đã mang.
- ⚠️ **Bug sửa ngay trong buổi tìm ra vẫn phải có mã**, ghi thẳng vào là `done` kèm số
  đo. Mã tồn tại để commit, PR và ghi chú sau này gọi cùng một thứ — mục chưa từng mở
  ngày nào cần điều đó y hệt mục nằm đó một tháng.
- Số trống kế tiếp **đọc từ bảng chỉ mục**, đừng đếm entry: số đã nghỉ vẫn là số đã
  tiêu, chỉ bảng chỉ mục mới biết cái gì còn trống.
- Trích mã trong commit message, PR và `docs/decisions.md`.
- Bảng chỉ mục ở đầu `TODO.md` (mã → trạng thái → tiêu đề) được **sinh từ chính file**
  chứ không gõ tay; sửa tiêu đề một mục thì sinh lại, đừng vá hai chỗ.

Trạng thái: `open` · `done` (xong, entry giữ lại kèm số đo) · `shipped` (§ *Done*) ·
`refused` (§ *Decided against*).
