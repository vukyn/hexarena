---
name: hexarena-paste-both-clients
description: "hexarena PR#274 dán chữ gãy ở CẢ HAI client; ⚠️ textinput.pasteMsg KHÔNG XUẤT nên ctrl+v đọc clipboard rồi vứt; walk toàn module bắt type có field mà thiếu Paste"
metadata: 
  node_type: memory
  type: project
  modified: 2026-09-03T19:33:27.218Z
---

hexarena PR #274 (`8856447`, 2026-09-04): ⌘V và ctrl+v không dán được vào **bất kỳ ô nhập nào** ở cả `cmd/hexarena-tui` (màn join PvP) lẫn `cmd/hexforge-tui` (**29 chỗ**). Không gì báo lỗi — chỉ lặng lẽ chưa bao giờ chạy.

**Why:** bubbletea v2 bật **bracketed paste mặc định** và giao paste dưới dạng **`tea.PasteMsg`**, không phải chuỗi `KeyPressMsg`. `bubbles/textinput` **xử lý đúng** message đó — nhưng `Update` của cả hai client chỉ switch `tea.KeyPressMsg` + message riêng, nên message **chết ở model**, không tới field. `PasteMsg` xuất hiện **0 lần** trong repo. → [[bubbletea-v2-silent-breaks]] (giờ là thứ **thứ năm** gãy im lặng, `CLAUDE.md` đã sửa)

**How to apply:**

⚠️ **`ctrl+v` TỆ HƠN "không xử lý" — nó đọc clipboard rồi VỨT.** `bubbles` **bind sẵn** `ctrl+v` → `textinput.Paste` (KeyMap mặc định, `textinput.go:82`), hàm đó **thật sự gọi `pbpaste`**, rồi phát ra **`textinput.pasteMsg` KHÔNG XUẤT** (`textinput.go:22`) — model ngoài package **không gọi tên được, không forward được**. Nên "nối ctrl+v vào `textinput.Paste()`" **KHÔNG CÀI ĐẶT ĐƯỢC**. Cách đúng: tự đọc clipboard, trả về **`tea.PasteMsg`** để hai đường (terminal bracketed paste + phím của chương trình) gặp nhau ở **một chỗ chèn**.

`github.com/atotto/clipboard` **đã có trong go.mod (indirect)** → chỉ thành direct, không thêm module. darwin dùng `pbpaste` (luôn có); unix cần `xsel`/`xclip`/`wl-paste` (có thể thiếu → trả nil, bubbletea bỏ qua message nil, **không có gì xảy ra**).
⚠️ **Không mâu thuẫn với quyết định "no clipboard" của `cmd/hexarena-host`** — cái đó là **CLI không tương tác gọi ra ngoài để COPY mã ĐI**; đây là **ô nhập TUI có phím dán** với phụ thuộc đã nằm sẵn trong cây. Phân biệt ghi trong package comment của `internal/clipboard`.

⚠️ **Guard `Focused()` của màn hình là BACKSTOP, không phải luật** — `textinput.Update` tự thoát khi field blur, nên bỏ guard **không đổi gì suite thấy được** (tôi mutate 2 lần mới nhận ra comment đã nói trước). **Luật thật là KIỂM CHẾ ĐỘ** (`o.Adding`, `s.FormInFront()`, `s.Mode`): client đang đứng ở một danh sách vẫn có field focus mà **người đọc không thấy** — paste nhắm "field đang focus" sẽ lặng lẽ đổ chữ vào đó.

**Xuống dòng — đo được**: `textinput` sanitise mỗi `\n` và `\r` thành **MỘT DẤU CÁCH** (`\t` cũng vậy; control khác bị bỏ). Mã dán kèm newline để lại field **13 ký tự**; `TrimSpace` sẵn có ở `submit()` làm nó vô hại. Assert **cả hai nửa** nên một cái trim "cho tử tế" lúc nhập sẽ fail test.

**Class guard**: `TestEveryTypeThatOwnsATextFieldTakesAPaste` — AST walk **mọi file không-test toàn module**, tìm type khai báo `textinput.Model`, giữ tập đó **bằng** tập có `Paste`. 6 field-holder / 135 file + allowlist 2 mục (model của mỗi client: route nhưng không giữ field). Test-mỗi-màn là 4 test mà màn thứ năm lọt — mẫu repo này ghi **5 lần**.

Liên quan: [[hexarena-pvp-lobby]] [[hexarena-tui-i18n]] [[hexarena-host-binary]]
