---
name: hexarena-a-cleanse-is-a-matchup-dependent-slot
description: "hexarena — chiêu ĐÚNG ARCHETYPE của Happiny (heal_bell) là slot duy nhất bỏ đi làm TỐT HƠN một matchup: +84 vs phe không có gì để gỡ, −123 vs phe có"
metadata:
  type: project
---

**Slot mang chính archetype của nhân vật có thể là slot khiến nó không qua sàn.**

`DAT-011`, 2026-09-09, 400 seed mỗi phía. Mỗi hàng khác kit ship đúng **một** slot, thay
bằng `hyper_voice` (thứ duy nhất khác trong learnset có sát thương):

| slot bị bỏ | vs slugger | vs blighter |
|---|---:|---:|
| không bỏ (kit ship) | 407 | 456 |
| `safeguard` | **163** (−244) | 345 (−111) |
| `soft_boiled` | 401 (−6) | 363 (−93) |
| `heal_bell` | **491 (+84)** | 333 (−123) |

⚠️ `heal_bell` là **slot duy nhất mà bỏ đi làm một matchup TỐT HƠN**. Cleanse đáng
khoảng **+123** trước phe mang thứ để gỡ và **−84** trước phe không mang. Nên slot thứ tư
phụ thuộc matchup **theo cấu tạo**, và một cái sàn assert **theo từng matchup** thì không
kit nào như thế vượt được.

Đếm cách thứ hai cùng một sự thật: `heal_bell` cast **2,5 lần/trận ở CẢ HAI matchup**,
so với `safeguard` 14,1 và `soft_boiled` 13,2. Rating từ chối nó ~80% số lần nó sẵn sàng,
vì không có gì để gỡ.

⚠️ **`safeguard` mới là thứ chịu lực** (−244 nếu bỏ), không phải cleanse. Nhân vật kiếm
được slot bằng **absorb theo cột**, còn cái tên archetype của nó là phần yếu nhất.

⚠️ **Trait cũng đóng, và fixture đã đông cứng nó.** `aThirdMemberAs` hardcode
`endurance` cho *mọi* thành viên thứ ba của *mọi* squad. Sweep cả 6 trait của Happiny:
`endurance` 407/456 là tốt nhất, `carapace` 276/**648** (ngược chiều), và `ballast`
đọc **21** — mười bảy thắng trên tám trăm trận, vì nó grant `encumber` lên đơn vị **chậm
nhất bàn** (speed 90). Một slot bị fixture đông cứng là một slot không ai định giá được;
`aThirdMemberFrom` giờ field build đã catalogue **kèm trait của nó**.

**Cách dùng.** Trước khi sửa stat cho một support dưới sàn, thay từng slot bằng một chiêu
sát thương và đọc **dấu** của hai matchup. Nếu chính chiêu archetype ra dấu dương thì vấn
đề không phải con số mà là **hình dạng của cái sàn**
→ [[measurement-faults-not-design-faults]]. Đòn bẩy stat đóng theo cách khác:
[[hexarena-effective-hp-buys-burst-and-loses-attrition]].

Liên quan: [[hexarena-a-rate-cannot-say-a-build-played-its-kit]],
[[hexarena-fester-heal-cut]], [[hexarena-builds-catalogue]].
