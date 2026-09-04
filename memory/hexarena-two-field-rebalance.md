---
name: hexarena-two-field-rebalance
description: hexarena — weigh chỉ đo MỘT field so với control trong data; đổi 2 field cùng lúc phải đo bằng spar overall trên bản data copy; weigh mới có --stage
metadata: 
  node_type: memory
  type: project
  modified: 2026-09-03T11:05:26.430Z
---

Đổi **hai số cùng lúc** trên một chiêu (vd pummel: strikes 3→5 **và** power 480→280) thì `hexforge weigh` KHÔNG đo được: control của weigh luôn là giá trị đang nằm trong data, nên nó chỉ trả lời "field X đáng bao nhiêu so với chính bản data này". Nối hai lần weigh lại cũng hỏng vì điểm giữa dễ **bão hoà** (pummel 480×5 đo được +47.9%, 480×6 bị từ chối vì saturated).

Cách đo đúng: copy `internal/seed/data` ra thư mục tạm cho MỖI ứng viên, sửa số trong bản copy, rồi chạy `hexforge spar <char> --stage <form> --data <dir>` và so **`overall`** giữa các bản. `overall` là số tuyệt đối nên so được **qua các trạng thái data khác nhau** — đúng thứ weigh không làm được. Nhiễu seed-to-seed của bench này ~±1pp ở 3000 seeds (base pummel đọc 50.7–51.3% qua các lần).

⚠️ Số học "giữ nguyên tổng sát thương" cho ra số SAI: 3×480=1440 → tưởng 288, nhưng thêm nhát tự nó có giá (+34.5% cho nhát 4, +47.9% cho nhát 5) nên phải hạ sâu hơn — 280 mới về đúng chỗ.

⚠️ Tổng bằng nhau KHÔNG có nghĩa cùng một chiêu: matchup đảo mạnh (squirtle 38→49%, gastly 64→57%). Đó là ý đồ, nhưng phải xem bảng từng đối thủ chứ đừng chỉ nhìn `overall`.

`hexforge weigh` giờ có **`--stage`** (PR#259). Trước đó `WeighRequest.Stage`/`CarriersRequest.Stage` đã tồn tại nhưng CLI không truyền → carrier rẽ nhánh (Poliwag rẽ ở lvl 32) luôn bị từ chối "name the one being fielded" mà không có cách nào gọi tên. Cùng họ với `spar --stage` vốn đã có.

Liên quan: [[hexarena-roster-cannot-price-damage]], [[hexarena-speed-and-measurement]], [[hexarena-stat-bounds-policy]].
