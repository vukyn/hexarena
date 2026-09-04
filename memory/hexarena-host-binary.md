---
name: hexarena-host-binary
description: "hexarena PR#253 cmd/hexarena-host — cổng cố định 13579 làm test thứ tự hoá rỗng; picker từ chối thay vì đoán; redact không với tới field unexported"
metadata: 
  node_type: memory
  type: project
  modified: 2026-09-03T08:35:08.672Z
---

hexarena PR #253 (`6286383`, 2026-09-03): `cmd/hexarena-host` — mở 1 phòng, in code dán, phục vụ trận, thoát. Kèm `socket.Server.Shutdown` (lỗ `TODO.md` tự ghi) và `wire.ClosureStopped` (thay đổi protocol DUY NHẤT; `ClosureCount` 2→3, không golden nào dời).

**Why:** cổng cố định là user chốt ("đừng đi cổng 0"), và nó kéo theo một cái bẫy đo lường không hiển nhiên.

**How to apply:**

⚠️ **Cổng cố định làm test thứ tự hoá RỖNG.** Bind-trước-Open-sau là bắt buộc vì code chở cổng. Nhưng ở cổng cố định, mở phòng TRƯỚC vẫn ra code chở 13579, vẫn chạy — test chạy ở mặc định pass dù có bug hay không. Phải lái qua `-port 0` (giữ hỗ trợ, không phải mặc định) và assert cổng decode == `listener.Addr()` VÀ khác 0. Cùng họ với [[hexarena-room-registry]] và mục "test pass cả hai chiều" ở [[hexarena-room-code-widened]].

**13579 chọn có đo**: trống trong IANA (hàng xóm 13xxx có đăng ký: NetBackup 13720–13785, powwow 13223/4, i-zipqd 13160); **dưới cả hai sàn ephemeral** — darwin `net.inet.ip.portrange.first`=49152, Linux `ip_local_port_range` từ 32768 — nên OS không bao giờ phát nó cho socket khác. **31337 loại có chủ đích**: cổng Back Orifice, IDS gắn cờ.

⚠️ **Cổng cố định mua về "address already in use" thành lỗi thường ngày** (2 host 1 máy, hoặc tiến trình cũ). Với cổng 0 chuyện này không xảy ra được — nên đó là GIÁ, phải bắt theo tên + nói làm gì (`-port <khác>`). `syscall.EADDRINUSE` đúng trên unix, **sai trên Windows** (`WSAEADDRINUSE`) — rơi xuống message chung.

⚠️ **`wire.Password` redact KHÔNG với tới field không xuất.** `fmt` lấy `String` qua `reflect.Value.Interface`, mà field unexported từ chối → `%v` của struct chứa nó in nguyên mật khẩu, mọi test khác vẫn xanh. Struct nào ôm một `wire.Password` phải tự khai lại `String`/`GoString`.

**Địa chỉ nào vào code** (`EncodeRoom` chở 4 byte → IPv4 only): hỏi bảng định tuyến trước (`net.Dial("udp4", "192.0.2.1:9")` — không gửi gói, chỉ chọn route), walk interface dự phòng (máy không có default route), `pick()` thuần trên `[]netip.Addr` để test được không cần mạng. ⚠️ **>1 ứng viên = TỪ CHỐI kể tên, không đoán**: docker bridge `172.17.0.1` là private + up + máy kia không dial được, và không luật nào trên riêng địa chỉ phân biệt nó với LAN — "ưu tiên 192.168/16" sẽ từ chối chính địa chỉ thật của máy user (172.16.32.222, tức 172.16/12).

⚠️ **Đường LAN không test nào chạm.** Mọi test dial loopback; listener là IPv6 dual-stack `*:13579`. Tôi đo tay: TCP tới `172.16.32.222:13579` thông, `/room/<code>` → 426, `/` → 404.

⚠️ **`Registry.Wait` chặn KHÔNG đo được bằng hành vi** — bước settle hội tụ dù có Wait hay không. Giữ bằng AST walk. Mutation bỏ Wait chỉ đỏ test structural.

**GitGuardian đỏ trên PR test-fixture password** — #251 và #253 đều "Generic Password" trên hằng fixture (`fixturePassword = "the-cat-sat-on-the-mat"`). Không có gì để rotate, main không có branch protection (repo private, không Pro) nên không chặn. **Đã dứt 2026-09-03**: user thêm filepath exclusion `**/*_test.go` gắn riêng hexarena ở dashboard — chỗ DUY NHẤT tắt được, file trong repo vô tác dụng. → [[gitguardian-dashboard-not-repo-file]] [[test-credential-literals-rule]]

Liên quan: [[hexarena-socket-transport]] [[hexarena-pvp-plan]] [[hexarena-wire-protocol]]
