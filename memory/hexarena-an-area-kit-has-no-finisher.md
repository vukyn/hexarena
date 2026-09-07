---
name: hexarena-an-area-kit-has-no-finisher
description: "hexarena — chiêu diện ở power chuẩn sách đáng ÍT hơn hẳn burst đơn mục tiêu, nên kit toàn chiêu diện không có đòn kết: igglybuff 0‰ vs mewtwo 1000‰ cùng ghế; ⚠️ spar 1v1 định giá column bằng MỘT mục tiêu nên đo ra hình dạng ngược"
metadata:
  node_type: memory
  type: project
---

Một kit làm **toàn** bằng chiêu diện (`column`, `arc_up`) và đa nhát, ở đúng mức
power sách đã đặt cho các chiêu diện khác, **không đủ sức kết trận**. Đo 2026-09-08
khi ship `pokemon.igglybuff` / preset `crooner` — light's area dealer.

**Số đo, `forge.FightSquads`, 200 seed mỗi chiều, mọi mirror control đọc đúng 500‰.**
Ghế giữ cố định: `s05` = machamp + gengar + ghế sau; `c05` = y hệt hai đồng đội đó
với Mewtwo ở ghế sau. Đối đầu cùng đội: **`c05` 1000‰ — 400/400 trận**.

| một biến so với bản ship | vs `s01` | vs `s04` |
|---|---|---|
| bản ship | 5‰ | 0‰ |
| tốc 95→140 | 32‰ | 5‰ |
| tốc **và** chính xác = 140/200 của Mewtwo | 32‰ | 7‰ |
| `plunge` 1900→2400 (trần tier chiêu-đổi-máu) | 7‰ | 0‰ |
| `endurance`→`berserk` | 10‰ | 2‰ |
| thêm `solar_beam` (2400) vào 4 chiêu ra sân | 2‰ | 47‰ |
| **bỏ cả 3 chiêu diện → 3 chiêu đơn nặng + `berserk`** | **172‰** | **237‰** |
| đối chứng: Mewtwo cùng ghế | 975‰ | 660‰ |

⚠️ **Không biến đơn nào dịch được, và KHÔNG phải stat line.** Trên giấy Wigglytuff
vừa dày vừa mạnh hơn cái đánh nó 400/400 — effHP 7927 vs 5802, atk 560 vs 520 —
và cho nó đúng tốc *lẫn* chính xác của Mewtwo vẫn chỉ 32‰. Chỉ khi thay **cả nửa
diện** của kit mới dịch. `discharge` (column 900) đứng được trên Magnezone vì có
`zap_cannon` bên cạnh, không phải vì một column đáng 900 tự thân.

⚠️ **spar (1v1) đo ra hình dạng NGƯỢC, đừng dùng nó cho carrier chiêu diện.**
Duel định giá một `column` bằng **một** mục tiêu, nên nó đang đo một chiêu khác
với chiêu sẽ ship. Ở đó: overall 4.5% (cleffa 26.8%, oddish 59.4%, happiny 0.0%),
probe đơn stat **phẳng hết** — acc 4.5% · atk 4.7% · def 4.6% · spd 8.2% — mà
nhân đôi power mọi chiêu mới thì đọc 47.4%. Tức là duel *có* nhạy với power,
nó chỉ không thấy diện tích. Cùng giới hạn mà [[hexarena-poliwag-bruiser]] ghi
cho support: **đọc carrier chiêu diện bằng squad, không bằng spar.**

⚠️ **`Rate()` bỏ trận Endless khỏi mẫu số**, nên tỉ lệ đẹp lên trong khi mẫu co
lại: một probe ở 1.5× power đọc 15‰ → 176‰ mà endless đi 202 → 301 của 400. Luôn
trích số endless cạnh mọi rate lấy trên đội khó kết trận. Cùng bẫy
[[hexarena-starter-squads]] đã ghi.

⚠️ **Một số tôi báo lúc đầu SAI, và sửa tại chỗ chứ không xoá.** Probe thay
Magnezone khỏi `s02` đọc 386/400 endless ở mirror của nó và tôi đã báo đó là
crooner làm đông bàn. Không phải: **mirror của `s02` tự nó 166 endless** vì
Blastoise mang bốn chiêu power 0, và bỏ nguồn sát thương duy nhất của đội đó mới
là thứ làm đông. Mirror `s05` đã ship đọc **endless 0**. Bài học: trước khi gán
một stalemate cho unit vừa thêm, **đo mirror của đội gốc**.

**Why:** tiền đề "định giá chiêu diện theo peer trong sách rồi ship" nghe đúng và
sai — từng chiêu đúng chuẩn (`dazzle` 900 = `discharge` 900; `refrain` 1200 giữa
`hyper_voice` 900 và `air_slash` 1400; `patter` 1140 tổng vs `pummel` 1400) mà cả
kit thì không đứng được. Dựng lại chuỗi 8 probe này tốn cả một session.

**How to apply:** đừng cho một nhân vật kit toàn chiêu diện. Cho nó một đòn kết
đơn mục tiêu và để diện làm phần bổ trợ — đúng cách Magnezone dùng `discharge`.
Nếu muốn sửa **nguyên nhân** thì cần định giá lại `patterns.json` →
`splash_power` (500‰ hôm nay) hoặc power chiêu diện toàn sách, và việc đó đụng
mọi hệ + mọi golden cân bằng → PR riêng, ghi ở `TODO.md` § *Not done* mã
`DAT-007`. Nhân vật đã ship nguyên trạng (quyết định của user) — mọi gate xanh,
`heal_cut` lần đầu được đo, light hết lệch carrier.

Related: [[hexarena-roster-cannot-price-damage]], [[hexarena-allsided-and-scarcity]],
[[hexarena-fester-heal-cut]], [[hexarena-speed-and-measurement]],
[[hexarena-two-field-rebalance]].
