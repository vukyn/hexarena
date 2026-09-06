---
name: hexarena-cast-listing-room
description: "browseRoom là skillsRoom song sinh; dự trữ pane là HẰNG SỐ vì đo mỗi lần vẽ tốn 13ms/redraw và tăng theo cast"
metadata:
  node_type: memory
  type: project
---

**`screen.browseRoom`** (PR sau #328) — cast listing giờ đo số hàng **từ cửa sổ** và cuộn quanh con trỏ bằng `Window`, y như `skillsRoom`. Trước đó nó vẽ **mọi** hàng, nên mỗi nhân vật ship ra là detail pane mất một dòng ở **mọi** cỡ cửa sổ; dòng mất trước tiên trên dòng tiến hoá **rẽ nhánh** là **form row** — thứ duy nhất nói stat bên dưới thuộc nhánh nào.

⚠️ **Dự trữ pane là HẰNG SỐ `browseDetailRows = 24`, không phải phép đo — và đó là kết luận từ SỐ ĐO chứ không phải đi tắt.**
`speciesRoom` đo được pane biến thiên của nó (`longestNote`) vì đó chỉ là một lần wrap một chuỗi. Làm y hệt ở đây nghĩa là **render pane của từng nhân vật** để đếm dòng:

| thành phần | chi phí |
|---|---|
| cả pass (23 nhân vật) | **13ms mỗi lần vẽ** |
| `os.Stat` sau art row | **53µs/nhân vật** (Windows) |
| `Context.Wrapped` | **15µs/lần**, ~10 lần mỗi pane |
| `Context.Label` | 6.7µs/lần |
| `lipgloss Render` | 0.9µs/lần |

Đó là chi phí **mỗi phím bấm** và **tăng tuyến tính theo cast** — đúng thứ mà dự trữ này sinh ra để khỏi phải bận tâm. Nên con số được viết ra và `TestTheDetailReserveIsTheTallestPaneInTheBook` giữ nó **CHÍNH XÁC, cả hai chiều**: vượt = pane bị frame cắt, thiếu = listing nhường dòng cho không.

⚠️ **Một số cho cả hai ngôn ngữ, tiếng Anh chịu thiệt.** Đo ở `MinWidth` (nơi mọi dòng wrap gắt nhất): pane cao nhất là `pokemon.mew` **cấp 16** — **24 dòng vi / 17 dòng en**, vì mỗi dòng gloss không vẽ gì khi không có tên. Screen **không được rẽ nhánh theo ngôn ngữ**, và thứ đo được ở đây là *số dòng* chứ không phải bề rộng một nhãn có thể hỏi ra — nên dự trữ là mức cao nhất pane từng đạt, người đọc tiếng Anh thấy dòng trống bên dưới. Đúng cái giá `speciesRoom` đã trả.

⚠️ **Ở sàn 120x24 pane vẫn bị cắt và không cách chia nào cứu được**: riêng dự trữ đã 24 dòng so với 20 dòng frame chừa cho thân. Sàn 3 hàng listing là **giữ điều hướng**, không phải một phần của ngân sách cân được. Trục việc này chữa là **độ dài cast**, không phải chiều cao cửa sổ.

⚠️ **Nó lộ ra một flake nó KHÔNG gây ra.** `TestABracketScrollsWhereverAPageKeyDoes` so **cả màn hình vẽ ra**, mà mỗi model trong đó tự gọi `start` → ba scratch dir khác nhau **một chữ số**, và header đặt tên thư mục data. Chữ số đó có hiện hay không phụ thuộc path có vượt clip ở `MinWidth` — do phần ngẫu nhiên của temp dir quyết định. Nên nó đỏ khoảng **một nửa số lần chạy**, ở site và ngôn ngữ nào rơi trúng biên. Giờ so **dưới header**, đúng lý do cả hai golden đã bỏ dòng đó.

Xem thêm [[hexarena-tui-width-rule]], [[hexarena-battle-screen-budget]], [[goldens-see-different-screens]].
