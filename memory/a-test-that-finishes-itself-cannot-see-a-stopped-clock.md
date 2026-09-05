---
name: a-test-that-finishes-itself-cannot-see-a-stopped-clock
description: "⚠️ Test tự cho subject hoàn thành thì KHÔNG thấy được cơ chế nó vừa phá — đo ở internal/socket: allowance.set tắt đồng hồ của ghế đang được hỏi, cả suite xanh, vì mọi test đều close(resume) rồi client tự trả lời"
metadata:
  node_type: memory
  type: feedback
---

⚠️ **Một test kết thúc bằng cách để subject tự hoàn thành thì không thể thấy cơ
chế cứu-hộ mà nó vừa phá.** Timeout, retry, watchdog, circuit breaker — mọi thứ
tồn tại cho ca *không* xảy ra đều vô hình với một test mà ca đó không xảy ra.

**Đo được (2026-09-06, `internal/socket`).** `allowance.set` làm
`generation++` và `timer.Stop()` **trước** nhánh quyết định "không có gì để arm",
nên một timeout tới muộn cho ghế **đã trả lời** lại **tắt đồng hồ của ghế đang
được hỏi**. `timedOut` ở đường bị từ chối là caller duy nhất truyền seat để thu
hẹp, và nó **không** gọi `settled` tiếp, nên không gì arm lại. Ghế đó đánh tiếp
mà không có allowance nào; người chơi bỏ đi là trận **treo mãi** — đúng thứ duy
nhất mà cơ chế "timeout là input" tồn tại để chặn. Bug đã **ship trong v0.1.0**.

⚠️ **Cả package xanh, và test đúng chỗ đó cũng xanh.**
`TestALateTimeoutIsRefusedWithoutDroppingAnybody` bắn đúng cái timeout muộn ấy —
rồi `close(resume)` để hai client tự trả lời tới hết trận. Đồng hồ chưa bao giờ
được nhờ tới việc gì, nên nó tắt hay không **không quan sát được**. Mọi test khác
trong package cùng một hình dạng.

**Cách áp dụng.** Đọc **trạng thái** của cơ chế trực tiếp, đừng chờ hệ quả — và
đọc **đủ cả** trạng thái, vì từng nửa một đều đúng với một cơ chế đã chết:

```go
func (a *allowance) armed() (bool, uint64) { … }   // timer còn + generation nào
```

- timer còn mà **generation đã dời** → callback bắn rồi return, không báo gì:
  cùng cái treo, dịch xuống một dòng
- generation đứng yên mà **không có timer** → đơn giản là chẳng có gì được arm

Nên bản sửa phải `return` **trên** cả `mu.Lock()`, và test ghim **cả hai** giá
trị. Mutation xác nhận: dựng lại bug → nửa "timer còn" đỏ; sửa thành "return sau
`generation++`" → nửa "generation đứng yên" đỏ. Cộng một vacuity guard đọc trạng
thái **trước** khi bắn, vì không có nó thì assertion pass trên một bàn chưa từng
arm đồng hồ nào.

⚠️ Đây là họ hàng của [[fixture-hidden-branch]] nhưng **không cùng một lỗi**: ở
đó fixture chạm early return nên nhánh không được chạy; ở đây nhánh **được chạy
đúng**, và chính *đường dọn dẹp* của test làm hệ quả biến mất. Câu hỏi phải hỏi
là **"nếu cơ chế này chết, test của tôi có còn xanh không"** — không phải "test
có đi qua dòng đó không".

Cùng bài học với [[hexarena-three-new-axes]] (đếm mà không assert = không đo gì)
và với ghi chú của repo rằng mọi bound end-to-end ở đây là **hang detector**, chứ
không phải ngân sách hiệu năng.
