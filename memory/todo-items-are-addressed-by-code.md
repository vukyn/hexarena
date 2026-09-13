---
name: todo-items-are-addressed-by-code
description: "hexarena — mọi mục TODO.md có mã AREA-NNN cố định; gọi việc theo MÃ, không theo số dòng; ⚠️ đóng một mục là DỜI entry sang docs/decisions.md, và bảng chỉ mục là gõ tay chứ không sinh ra"
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
| `DOC` | **doc**umentation | `TODO.md`, `docs/`, `CLAUDE.md`, `README.md`, `memory/` |

⚠️ **`RAT` là *rating*** — thứ `Suggest` quyết — không phải con chuột, không phải
tỉ lệ. Là vùng duy nhất mà ba chữ đọc lên thành một từ tiếng Anh khác hẳn, nên cột
"viết tắt của" phải có chứ không hiển nhiên.

**Why:** số dòng là dữ kiện về *lượng chữ nằm phía trên* một mục — mà đó chính là thứ
file này lớn lên. Một entry mở ở đây dài tới mười hai kilobyte; chỉ cần ai đó viết
thêm một đoạn là mọi số dòng đã trích trong commit, PR hay `docs/decisions.md` thành
sai, và không có gì báo. Mã thì không đổi.

**How to apply:**

- **Mã cấp một lần, không đổi, không tái dùng.** Xong việc thì mã giữ nguyên nhưng
  **entry phải DỜI**: cả mục, nguyên văn, sang `docs/decisions.md` dưới đúng mã đó, và
  § *Not done* chỉ còn `- [ ]`. Chuyển mục, từ chối mục cũng không đổi mã. Mục mới lấy
  số trống kế tiếp trong vùng.
- ⚠️ **Luật cũ chép ở đây — *"xong thì entry vẫn nằm trong § Not done kèm số đo, chỉ
  đổi `[ ]`→`[x]`"* — đã BỎ (`DOC-001`), và đây là số đo giết nó:** ngày 2026-09-13
  § *Not done* mang **30 mục đã xong so với 12 mục còn mở**, 2.663 dòng việc đã xong
  nằm dưới đúng cái tiêu đề nói là chưa xong. Không phải ai lười: tick tại chỗ tốn một
  ký tự, dời tốn một lần cắt-dán hai chục kilobyte, và cái sai thì không ai nhìn thấy
  — một luật như thế tự mục. Nay giữ được là nhờ một lệnh chứ không nhờ trí nhớ:
  ```sh
  awk '/^## Not done/{f=1;next} /^## /{f=0} f && /^- \[x\]/' TODO.md
  ```
  Nó phải không in gì. Trên commit ngay trước `DOC-001` nó in đúng **30 dòng** — và
  nửa đó mới là nửa đáng ghi: một cái chốt chưa ai từng nhìn thấy lúc nó BÁO thì chưa
  phải bằng chứng rằng nó chạy, mới chỉ là một lệnh người ta tin.
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
- ⚠️ **Bảng chỉ mục ở đầu `TODO.md` (mã → trạng thái → tiêu đề) là GÕ TAY.** Ghi chú
  này từng nói nó *"sinh từ chính file chứ không gõ tay"* — **sai**: trong repo không
  có bộ sinh nào hết, không `scripts/`, không target Makefile, không gì khớp
  `AREA-NNN` ngoài chính mấy file tài liệu. `DOC-001` sửa câu này chứ không dựng bộ
  sinh. Nên đổi tiêu đề hay đóng một mục là phải vá **hai chỗ trong cùng một commit**:
  entry (nay nằm ở `docs/decisions.md`) và dòng chỉ mục ở `TODO.md`.

Trạng thái: `open` (còn mở, entry ở § *Not done*) · `done` (xong — entry kèm số đo nằm
ở `docs/decisions.md` dưới cùng mã, **không** còn ở § *Not done*) · `shipped` (xong, và
§ *Done* có một đoạn văn xuôi về năng lực đó) · `refused` (§ *Decided against*).
