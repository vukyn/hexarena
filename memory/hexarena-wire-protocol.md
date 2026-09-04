---
name: hexarena-wire-protocol
description: internal/wire = PvP protocol, no I/O; 4 messages carry nothing on purpose; time cannot enter the package; zero-value enum decoded as the wrong message
metadata:
  type: project
---

`internal/wire` (PR #237, merged `3a6cd17`) — PvP protocol as **one package with no
I/O**: envelope, 7 kinds, 10 refusal codes, 3 version numbers, room code, event
digest. Room + WebSocket là 2 item sau. **Groundwork PvP XONG 4/4.**

Gần như không khai báo lại gì: `placement.Squad` đã là squad wire format,
`battle.Roster` đã có json tag, `hex.Cell` đã có absence riêng, `seed.Digest` đã
là data digest → cổng so 2 giá trị cùng type, compiler kiểm.

**4 message cố ý chở ÍT hơn có thể**, mỗi cái là 1 luật đã có:

| | chở | KHÔNG chở |
|---|---|---|
| `pass` | **không gì cả** | reason — `battle.NoActionReason` là khai báo duy nhất |
| `act` | skill + aim | unit; server biết tới lượt ai |
| `start` | **1** roster slice | 2 field — `seq` theo thứ tự slice, quyết tie tốc độ, tới 60 điểm ([[hexarena-side-is-worth-60-points]]) |
| — | — | **series-standing** — client là mirror, tự đọc `Ended` |

Kind + Code serialise **BY NAME** (luật của event kind/side/outcome). `Format` là
ngoại lệ CÓ LÝ: giá trị *chính là* số unit mỗi phe → không có thứ tự khai báo để
insertion phá; format 4 bị từ chối chứ không đọc sai.

⚠️ **`time` không được import vào package, và đó là phép đo chứ không phải lời
hứa.** Turn allowance = **giây, kiểu int** (Duration JSON ra nanosecond, golden
không đọc được). Test là AST walk trên chính thư mục package — không phải "không
field nào là timestamp" mà **"khái niệm không vào được"**. Đo: thêm import `time`
vào room.go → `TestTheProtocolCannotReadAClock` đỏ, gọi tên file VÀ mang số file
đã quét (không pass bằng cách quét 0 file).

⚠️ **Envelope không tên kind DECODE THÀNH `hello`.** `KindHello` = 0 và
`Kind.UnmarshalJSON` không bao giờ chạy cho field vắng → `{"body":{}}` parse sạch
thành SAI message. Chữa bằng `Envelope.UnmarshalJSON` bắt buộc field, **KHÔNG**
bằng cách thêm kind ở 0: đó là câu trả lời của `hex.SideNone` và ở đây ngược lại
mới đúng — một phe thật sự có "không ai", một envelope không kind thì không phải
message của format này.

⚠️ **Test "hai lần chạy giống nhau thì bằng nhau" KHÔNG thấy digest đọc thiếu
field.** Cho `DigestEvents` marshal 4 field thay vì cả event → reflection walk đỏ
**25/29** subtest + golden `turn` đỏ, còn
`TestTwoIdenticalRunsOfEventsAgreeAndOrderMatters` **XANH**. Vì vậy test độ nhạy
phải là walk mọi field, không phải bảng viết tay. Cùng hình dạng với
[[goldens-see-different-screens]].

**Framing viết lại chứ không share với `seed.digest`, và NGẮN HƠN**: seed đóng
khung **tên file** vì tên không nằm trong file; `kind` của event là thứ marshal
đầu tiên → prefix tên là bản copy thứ 2 của thứ đã ở trong khung. Share sẽ cần
package **thứ ba** mà cả seed lẫn wire import, để giữ đúng `sha256` +
`binary.BigEndian` ([[hexarena-data-digest]]).

**Golden: mọi body là fixture VIẾT TAY, không có gì từ data đã ship** —
`start` chở roster thật sẽ dời theo mọi commit balance. Giữ bằng test
(`TestTheGoldenIsBuiltFromNothingShipped` từ chối mọi prefix id đã ship); tôi kiểm
riêng: digest trong `hello` là `000102…1f` (đếm lên), không phải digest thật
`5e96a5a8…`.

⚠️ Golden ghi **password dạng rõ**, cố ý (design record nói password đi qua dạng
rõ). Đã quét: `gitleaks detect --no-git --source internal/wire` → **no leaks
found**. Go side dùng const `fixturePassword`.

⚠️ **10 refusal code CHƯA có wording nào** — wire không được import `internal/i18n`
nên test wording không sống ở đây. Count giữ trước, gap ghi vào `TODO.md`; walk
2 quyển sách sẽ về cùng item wording của client. → [[hexarena-pvp-plan]]
