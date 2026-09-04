---
name: hexarena-data-digest
description: seed.DataDigest = peer-equality gate; framing reason measured (concat blind to moved boundary + rename, NOT swap); walk beats self-referential table
metadata:
  type: project
---

`seed.DataDigest()` (PR #233, merged `6738c70`) — hash of the 15 embedded JSON
files in `go:embed` order, for the PvP room's join gate. **`Digest [32]byte`**,
`String()` full hex compares, `Short()` 12 hex for humans only.

**Nó là phép so BẰNG NHAU giữa 2 peer, KHÔNG phải số version.** Không cần ổn định
qua commit — mọi PR data *đáng lẽ* phải làm nó đổi.

**Từng file được ĐÓNG KHUNG: name → byte length (uint64 BE) → bytes.** Lý do tôi
viết trong spec là SAI, và coder đo ra: `sha256(concat(bytes))` **KHÔNG mù** với
"2 file đổi nội dung cho nhau" — đọc theo thứ tự list nên swap dời byte sang
offset khác, hash thấy ngay. Nó mù với:

| case | concat-only |
|---|---|
| nội dung swap | **thấy** |
| **biên dịch chuyển** (2 file kề, cùng tổng byte, chia khác) | **mù** |
| **đổi tên** file | **mù** |

Kết luận sống, lý do chết. Giữ case sai làm subtest có nhãn — nó là case cửa gặp
thật, và một lý lẽ sai nên nằm cạnh phép đo đã giết nó.

**Đo từng nửa khung** (bỏ từng phần rồi chạy): `name` làm gần hết việc (bắt
rename, và nằm giữa byte 2 file nên kiêm separator → bắt cả moved boundary).
`length` chỉ được GIỮ bởi đúng 1 assertion (biên dời qua đúng bản copy tên file
kế tiếp). Case nào đỏ dưới MỌI mutation là đang đo cả khung như một cục.

⚠️ **Test chịu lực là phép ĐI BỘ (walk) qua embed FS, không phải bảng sensitivity.**
15 cái tên giờ tồn tại **3 bản độc lập**: directive, 15 lời gọi `ReadFile`,
`dataFiles`. Bỏ 1 tên khỏi `dataFiles` → walk đỏ và gọi tên file; còn
`TestFlippingOneByteInAnyFileChangesTheDigest` vẫn **XANH** vì nó table-driven
off chính list đó. Cùng hình dạng với [[goldens-see-different-screens]]: test
kiểm 1 list bằng chính list đó không thấy được thiếu-nhất-quán.

**KHÔNG golden trên giá trị digest, cố ý** — nó sẽ dời theo mọi commit data
không liên quan (repo này parallel session ship data PR liên tục) mà không bắt
thêm gì so với property test. Máy sinh merge-conflict đo được số không.

File thiếu = **error**, không bao giờ digest một phần: 2 peer đồng ý trên một lần
đọc dở là kết quả tệ nhất cửa này tạo ra được. `assets/` ngoài vòng (55 SVG).
Không parse gì — 15 file JSON sai cú pháp vẫn digest được (đó là 1 test).

Groundwork PvP còn: `internal/wire`, và cursor thay `Drain`
([[hexarena-pvp-plan]]).
