---
name: hexarena-two-numbers-that-meet-in-one-product
description: "hexarena ENG-004 — bonus × share gặp nhau ở ĐÚNG một tích trong combat.Swung nên bề mặt là hạng-1; grid hai trục bị TỪ CHỐI, thay bằng field self_bonus"
metadata:
  node_type: memory
  type: project
---

**Trước khi dựng báo cáo hai trục, hỏi hai số đó gặp nhau ở đâu.**

`ENG-004` xin một grid trên cặp *bonus của `self_requires`* × *share của
`self_gradient`*. Chúng gặp nhau ở **đúng một biểu thức**, `combat.Swung`:

    (power + bonus) × (1000 + share) ÷ 1000        // MỘT tích, MỘT lần truncate

Nên **mỗi nhát đánh, bề mặt là hạng một**: đáp số chỉ là hàm của cái tích, mọi ô
trên cùng một hyperbol là **một ô**. Ở power 1000: `bonus 0/share 1000`,
`bonus 1000/share 0`, `bonus 250/share 600`, `bonus 600/share 250` ra cùng một số
— và đẳng thức **sống sót qua truncate**, vì phép chia lấy một lần, của cái tích.

Hai chỗ đọc `Bonus`/`Share` còn lại là **trường event** (`turn.go` gắn share vào
`SkillUsed`, bonus vào `Amplified`) — renderer in ra, không tới trận. Không còn
chỗ nào cho tương tác nấp.

⚠️ **Giới hạn trung thực, phải nói ra:** `self_gradient` khai `at_empty`, share
**thực** là hàm của máu caster — nên ở mức **cả trận** hai số không hoán đổi được.
Grid vẫn có thể vẽ ra một hình, nhưng hình đó là **vòng phản hồi** (caster bị
thương đánh mạnh hơn → thắng sớm hơn → ít bị thương hơn), tức tính chất của **bàn
cờ**. Grid trên hai field kỹ năng sẽ quy nó cho hai field.

⚠️ **Cặp cũng không có chủ thể nào đang ship.** Đo lại 2026-09-08: **2** chiêu có
gradient (`comeback`, `reversal`), **12** có `self_requires`, trong đó **5** khai
bonus — giao **rỗng**. Cặp là **hợp lệ**: `resolveGradient` chỉ từ chối gradient
đứng cạnh điều kiện đọc **máu**; ngưỡng trên status thì hợp nhau bình thường.
⚠️ Con số trong entry cũ **sai cả hai chiều** (ghi 1 gradient / 7 self_requires) —
lại một lần nữa luật *"đo lại trước khi trích"*.

**Thứ cặp đó cần là cái ĐƯỜNG còn thiếu, không phải mặt.** Gradient vốn weigh
được, bonus thì không — nên ship `self_bonus`. ⚠️ `set` **sao chép** condition chứ
không dựng cái mới: đọc status nào, mấy stack, có gate không, có consume không đều
**là** kỹ năng, reset bất kỳ cái nào là định giá một kỹ năng khác mà báo cáo trông
y hệt.

⚠️ **Bench KHAI được cơ chế mà không CHẠM tới được** — đúng bẫy repo giữ danh sách.
`vent` là chiêu duy nhất của bench nhắm địch, có power, có điều kiện riêng; nó gate
trên 3 stack `swelter` và **không gì trong bench tạo ra swelter** → lần chạy
end-to-end đầu cast 0 lần, weighing từ chối row (đúng, và bằng lời của chính nó).
Nhiên liệu được author vào **scratch library** trong test chứ không thêm vào bench:
bench là thứ hàng trăm golden vẽ.
⚠️ Bonus bằng đúng power 2400 của `vent` làm **bão hoà** bàn và bị từ chối — *một
slot thắng 100.0%, slot kia 100.0%* — nên test dùng 400. Từ chối đó là dụng cụ chạy
đúng.

Xem thêm [[hexarena-pricing-a-new-element-pool]], [[fixture-hidden-branch]],
[[hexarena-a-stale-referent-outlives-the-conclusion-it-blocked]].
