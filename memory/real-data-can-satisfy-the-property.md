---
name: real-data-can-satisfy-the-property
description: "⚠️ Dữ liệu ship có thể TÌNH CỜ thoả tính chất đang kiểm, nên test dùng dữ liệu thật rỗng nghĩa — đo ở internal/draft: cast.json vốn đã sắp theo id, nên test 'giữ thứ tự khai báo' pass cả khi hàm sort"
metadata:
  node_type: memory
  type: feedback
---

⚠️ **Trước khi viết test cho một tính chất *thứ tự*, hỏi: bộ dữ liệu thật có
đang sẵn ở thứ tự đó không?** Nếu có thì test dùng dữ liệu thật **không phân biệt
được** cách làm đúng với cách làm sai, và nó pass — không đỏ, không cảnh báo, chỉ
là không đo gì.

**Đo được (2026-09-05, `internal/draft` bước 1 của ban/pick).**
`draft.NewPool` phải giữ **thứ tự khai báo** của `cast.Book` và **không** được
sort theo id. `internal/seed/data/cast.json` khai `naruto.naruto` rồi mọi
`pokemon.*` theo alphabet — **chính là thứ tự id**. Nên một test dựng pool từ
`seed.Cast()` rồi so với thứ tự của sách **pass y nguyên** trên một `NewPool` có
`slices.SortFunc(…, byID)` hàn vào trong. Xác nhận bằng mutation: cả `sort by id`
lẫn `reverse` **chỉ** bị bắt bởi test dùng một sách tổng hợp 5 nhân vật mà thứ tự
khai báo cố ý khác thứ tự id.

**Cách áp dụng.** Fixture tổng hợp, và **assertion đầu tiên là guard tự loại**:

```go
if slices.IsSorted(ids) {
    t.Fatal("thứ tự khai báo của fixture cũng là thứ tự id, nên test này không " +
        "phân biệt được thứ tự khai báo với một phép sort: nó không đo gì")
}
```

Guard đó là nửa quan trọng hơn: fixture trôi dần thành sorted là chuyện xảy ra
khi người sau thêm một hàng, và không có guard thì nó **rỗng nghĩa trong im lặng**
đúng như bản dùng dữ liệu thật.

⚠️ **Đây là thất bại NGƯỢC với [[fixture-hidden-branch]] và với "dùng dữ liệu
thật".** Ở đó fixture quá đơn giản nên nhánh không được chạy → chĩa vào dữ liệu
ship. Ở đây dữ liệu ship *tình cờ* là đáp án → phải dựng fixture. Câu hỏi không
phải "thật hay tổng hợp", mà là **"input này có phân biệt được đúng với sai
không"**. Hai lời khuyên trái nhau, và cái nào đúng phụ thuộc vào tính chất —
nên đừng nhớ lời khuyên, hãy hỏi câu hỏi.

Cùng họ với [[fixture-hidden-branch]] và với guard đếm case của
[[mutate-the-producer-not-just-the-logic]]: mọi claim vét cạn đều nợ một con số
nói nó đã chạy bao nhiêu ca, vì một vòng lặp chạy 0 lần cũng pass.
