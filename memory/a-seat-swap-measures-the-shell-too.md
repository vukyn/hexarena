---
name: a-seat-swap-measures-the-shell-too
description: "hexarena — đổi một ghế trong đội rồi đọc rate là đo CẢ cái đội: hai shell cho hai kết luận ngược nhau trên cùng một nhân vật, cùng một dữ liệu; ⚠️ dấu hiệu nằm ở cột endless chứ không ở rate"
metadata:
  node_type: memory
  type: project
---

Đo một nhân vật mới bằng cách thay nó vào một ghế của đội đã ship rồi đọc
`forge.FightSquads` — **kết quả phụ thuộc vào hai thành viên còn lại nhiều như
phụ thuộc vào cái ghế.** Đo 2026-09-08 khi ship `pokemon.magikarp`
(Magikarp → Gyarados): hai shell, cùng dữ liệu, **kết luận ngược nhau**.

200 seed mỗi chiều, mọi mirror control đọc **đúng 500‰**.

**Shell A — `s02` (Blastoise + Gengar), Gyarados thay Magnezone:** 1000‰ vs `s01`,
896‰ vs `s03`, 813‰ vs `s04`, đối chứng đọc 717 / 120 / 378‰ → "quá mạnh, phải
cắt một nửa".

**Shell B — `s04` (Machamp + Mew), Gyarados thay Mewtwo, control chạy CÙNG lần:**

| đối thủ | Gyarados ghế đó | Mewtwo ghế đó |
|---|---|---|
| `s01` | 632‰ | 907‰ |
| `s02` | 463‰ | 621‰ |
| `s03` | **95‰** | 452‰ |
| `s05` | 1000‰ | 1000‰ |
| đối đầu | 387‰ | — |

→ "yếu hơn đều đặn 158–357‰".

⚠️ **Shell A là cái sai, và thứ nói ra điều đó là cột `endless`, không phải
rate.** Blastoise mang **bốn chiêu power 0**, nên toàn bộ sát thương của `s02`
nằm ở đúng cái ghế đang đo — vừa làm đóng góp của ghế thành cả phần công của đội,
vừa làm shell **đông cứng** khi hạ ghế xuống. Đo trong lúc tune ghế giảm dần:
mirror của shell A đi **endless 50 → 88 → 126** trên 400, matchup `s03` đi
**92 → 144 → 222**, mà `Rate()` **bỏ trận endless khỏi mẫu số** — nên mọi con số
đẹp lên trên một cái mẫu đang co lại. Shell B chạy **endless 0** suốt.

**Why:** một shell không kết được trận thì đang đo chính cái shell. Và một shell
mà hai thành viên kia không tự thắng được thì cái ghế không phải một biến, nó là
cả đội. Cùng cái bẫy `Rate()` mà [[hexarena-starter-squads]] ghi, và đây là lần
**thứ hai** nó quyết định sai một kết luận ở repo này — lần đầu là
[[hexarena-an-area-kit-has-no-finisher]], nơi tôi gán 386/400 endless cho unit vừa
thêm trong khi đó là tính chất của `s02`.

**How to apply — bốn thứ, đều rẻ:**

1. Chọn shell mà **hai thành viên còn lại tự thắng được** khi không có ghế đang đo.
   `s04` (machamp + mew) được; `s02` (blastoise + gengar) **không**.
2. Control phải chạy **cùng một lần**, trên **cùng bộ đối thủ**. Đừng ghép số
   control từ một lần chạy trước — [[hexarena-a-stale-referent-outlives-the-conclusion-it-blocked]].
3. **Trích `endless` cạnh mọi rate.** Rate một mình không phân biệt được
   "thắng 95‰" với "chỉ 178 trận kết thúc".
4. Đo **hai** shell trước khi kết luận. Nếu hai shell lệch nhau về dấu thì chưa
   có kết luận nào, chỉ có một cái shell hỏng cần tìm.

⚠️ **Một số đo còn treo, không dọn đi:** Gyarados là water/wind đánh `s03`
(grass + electric/metal + fire), mà bảng khắc chế cho water > metal và wind >
fire — vậy mà đọc **95‰**, còn Mewtwo hệ **dark** (nằm ngoài bảng, profile phẳng)
đọc 452‰ cùng ghế. Trên bàn đó, lợi thế hệ hai chiều đáng ít hơn khoảng cách stat
và tốc mà nó được mang theo. Đó là câu hỏi "một affinity đáng bao nhiêu", ghi ở
`TODO.md` `DAT-012`, và không phép đo nào ở đây trả lời nó.

Related: [[hexarena-starter-squads]], [[hexarena-an-area-kit-has-no-finisher]],
[[hexarena-roster-cannot-price-damage]], [[hexarena-squad-builder]],
[[hexarena-two-field-rebalance]], [[measurement-faults-not-design-faults]].
