---
name: goldens-see-different-screens
description: hexarena — 3 golden màn hình thấy 3 thứ khác nhau; đo được cả ba hình dạng mù; luật chung để biết một test có đo gì không
metadata:
  type: reference
---

Sau khi tách màn hình (xem [[hexarena-screen-extraction]]) repo có **ba** golden màn hình: `cmd/hexforge-tui` (~200 render), `internal/screen` (164), `cmd/hexarena-tui` (96). Chúng **không phải một tấm lưới đặt ba nơi.**

## Ba hình dạng mù, đo được cả ba

| PR | ai mù | vì sao |
|---|---|---|
| #205 | **package** | nới cột trong view đã dời → chỉ golden client bắt |
| #223 | **client** | fixture client **xoá `squads.json`** → không test nào của nó từng vẽ một danh mục có hàng |
| #225 | **cả hai** | không fixture nào dựng trạng thái rỗng / thành công / bị từ chối |

Hai cái đầu **bù nhau** — đó là biện minh cho golden của package (#206). Cái thứ ba **không cái nào bù**, và đó là giới hạn thật: **hai golden bao *màn* của nhau, không bao *trạng thái chưa ai với tới* của nhau.**

⚠️ Và một lỗ chưa đóng: nới cột trong màn **cả hai client vẽ** → **cả ba golden nhích, không test thuộc tính nào ở đâu đỏ** (hàng đó là *cột dữ liệu*, mọi width sweep miễn trừ).

## Luật chung rút ra — hỏi một test đo được GÌ

- **Golden bao render, KHÔNG bao transition.** Đổi navigation phải có driven test.
- **Test phủ đủ (walk-the-count) đếm CÓ MẶT, không đếm CÓ TÁC DỤNG.** #207 đo: applier entry có mặt nhưng no-op → walk **vẫn xanh**, 4 test khác đỏ. Hai lớp giữ hai nửa.
- **Một test kiểm một thứ bằng chính nó không thấy cái sai nhất quán.** #218: test filter đi hết vòng, kiểm hàng thuộc nhóm đang bật, kiểm tổng — **cả ba đúng** khi filter lệch một, vì hàng và nhãn đọc *cùng một* `Group()` sai.
- ⚠️ **Golden của detail pane chỉ ghi MỘT nhân vật — con dưới con trỏ — nên nó thấy gì là hàm của THỨ TỰ file data.** #243 (`WrappedIn` lấp cột cuối) đo được hai lần: trên cây trước #242 cùng bản vá đó làm **0/17 golden dịch**, sau #242 (sort `cast.json` → con trỏ rơi sang Naruto) làm **7 dòng dịch**. Cùng fix, cùng suite, cùng màn — khác đúng một commit data. Thực tế lỗi là **11/26 bio row vi và 6/26 en** lấp cột cuối. `aSquadOfSide` chọn `characters[i % len]` là gốc chung với entry TODO "`hexforge new` still churns `screens.golden`".
- **Fixture ngồi ở cuối danh sách không thấy lệch-một** (#223, clamp cứu). **Fixture một hàng không thấy chọn nhầm hàng** (#214, một squad thì id nào cũng xoá đúng).
- **`strings.Contains` không phải một layout** (#225: test lái đúng lời từ chối, thêm một dòng trước nó vẫn xanh).
- **Mirror phẳng làm phép đo mù** (#231: hai phe giống hệt thì hoán vị được).
- **Một bản vá có thể ship không có lưới** (#228: stack sâu-2 chữa lỗi thật, mutation mới phát hiện không gì giữ nó).

## Bẫy vận hành

- ⚠️ **`clean merge` của git KHÔNG nói gì về nội dung một golden** khi data nền đã dịch. Nổ **hai lần**: #212 (`8 trong 8` → `10 trong 10`) và #220 (`90 chiêu` → `91 chiêu`). **Chạy test trước khi merge, đừng tin nhãn GitHub.**
- ⚠️ **cwd âm thầm rơi về main checkout** khi worktree nó đang đứng bị xoá → mutate nhầm cây. Đường dẫn tuyệt đối + assert `pwd`.
- ⚠️ **Script mutation quên `open(p,'w')`** → chạy xanh mà không đo gì. Grep lại dòng **trên đĩa** trước khi tin.
- ⚠️ **Revert theo *chuỗi* khôi phục nhầm dòng** khi file có hai dòng giống hệt, và suite vẫn xanh. Nhắm **theo dòng**. File untracked thì `git diff` vô dụng — `diff` với backup.
- **Đừng gọi `go test` cả package hai lần trong một lệnh** (~55s/lần) — tôi tưởng mutation làm treo, hoá ra là tự mình.
