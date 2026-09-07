---
name: hexarena-a-second-consumer-needs-a-second-cursor
description: "hexarena — log của phòng PvP phải có cursor RIÊNG bắt đầu từ 0; cursor của phòng đặt sau bàn mở nên log viết từ đó re-run ra NHIỀU sự kiện hơn số đã ghi"
metadata:
  node_type: memory
  type: project
---

**Một consumer thứ hai của `Battle.Since` cần một cursor thứ hai — kể cả khi chỗ
đọc trông giống hệt nhau.**

Phòng PvP giờ xuất mỗi trận ra `battle.Log` (`room.BattleResult.Log`), nên mọi trận
`hexarena --replay FILE --verify` được. Phòng **soạn** log và **không ghi gì** — nó
không có filesystem; log đi ra theo `Reading.Played`, `hexarena-host -logs DIR` mới
biến nó thành file.

⚠️ **Cursor của phòng KHÔNG bắt đầu từ 0.** Nó được đặt bằng `Recorded()` **sau**
bàn mở, vì không `wire.Turn` nào chở bàn mở — mirror tự gọi `Begin` sinh ra. Log
viết từ cursor đó sẽ re-run ra danh sách sự kiện **DÀI HƠN** số nó ghi:

| trận | log ghi | re-run |
|---|---:|---:|
| 1 | 291 | 300 |
| 2 | 345 | 354 |

Test nói **đúng nguyên nhân** (`TestALoggedBattleStartsAtTheOpeningBoard`: sự kiện
đầu tiên phải thuộc turn 0) chứ không chỉ "sự kiện khác nhau" — một test chỉ so số
đếm sẽ báo sai chỗ.

⚠️ **Trận bị cap không có `Ended` mà vẫn verify được** — với điều kiện bản ghi bao
gồm `turn_began` **của chính lượt bị cap**, đúng chỗ cursor phòng đang đứng, vì
`settle` đã tiến vào lượt đó rồi mới quyết định không hỏi, và `Replay` cũng tiến
vào. Dừng sớm một sự kiện: 43 ghi vs 44 re-run.

⚠️ **Trận bị bỏ dở thì không ai ghi log**, và điều đó rơi ra từ phòng chứ không phải
luật thứ hai: chỉ `close` dựng `BattleResult`, còn rời phòng đi qua `abandon`.

⚠️ **Tên file KHÔNG dùng của `internal/forge`.** Cái đó là
`<home>-vs-<away>-seed<N>.json`, hợp cho spar giữa hai squad thư viện giữ theo id.
Ghế PvP là `host`/`guest` ở **mọi** trận, nên mã phòng + số trận mới phân biệt được.

⚠️ **Nó làm đỏ `TestNoLockingFunctionSendsOnAChannel`, và đó là test đáng xem chứ
không đáng tắt.** AST walk đó khoá `functions` theo **tên trần**, nên
`r.fight.Since(...)` trong reader mới bị buộc vào `Registry.Since` (có gửi channel)
→ walk báo một method của room "với tới" channel nó chưa từng nghe. Giờ chỉ tính
selector call là cạnh khi **receiver là receiver của chính hàm đó**: `r.watch(...)`
là cạnh, `r.fight.Since(...)` thì không. Số hàm "reach a send" tụt **23 → 8**, và
mutation vẫn cắn (cho một hàm giữ lock gọi một hàm có gửi → đỏ) — luôn kiểm lại
điều đó sau khi siết một test, kẻo chỉ là nới lỏng.

Xem thêm [[hexarena-cursor-record]], [[hexarena-room-state-machine]].
