---
name: hexarena-screen-extraction
description: hexarena — tách màn hình tham chiếu khỏi cmd/hexforge-tui thành 6 bước; ⚠️ cmd/hexforge-tui KHÔNG có golden nào; cách chụp byte và 2 nguồn bẩn của nó
metadata:
  type: project
---

Groundwork của [[hexarena-pvp-plan]]. **6 bước, không làm một cục.** Đã xong 3:

✅ **XONG TOÀN BỘ** (2026-09-02). 6 bước dự kiến thành **12 PR** — mỗi lần đo lại lộ ra chặn thật, và tách ra thì mỗi PR tự verify được. `cmd/hexforge-tui` **10.144 → 3.315 dòng**; `internal/screen` **9.154**; client thứ hai `cmd/hexarena-tui` **1.156**.

| bước | PR | commit |
|---|---|---|
| 1 — context vẽ dùng chung | #199 | `f3e6676` |
| 1b — golden client (200 render) | #201 | `b840b00` |
| 2 — navigation thành `Action` | #203 | `4c2ceb5` |
| 3 — dời 6 màn | #205 | `127c06f` |
| 3b — golden package (32 render) | #206 | `3078569` |
| 4a — describer nhận subject | #207 | `086a9bb` |
| 4b — dời 2 describer | #210 | `415168e` |
| 5a — dời browse | #212 | `bef64c6` |
| 5b — guard thành dữ liệu | #214 | `c131d96` |
| 5c — picker `apply` thành dữ liệu | #216 | `aaa5ad3` |
| 5d-i — dời picker | #218 | `6678ccd` |
| 5d-ii — dời skills | #220 | `ebec7a0` |
| 5d-iii — dời squads | #223 | `8a8ff8b` |
| 6a — dời origins | #225 | `96e2fcb` |
| 6b — dời play | #228 | `f5bdaa0` |
| **6c — client thứ hai** | **#231** | **`6fbf194`** |

## ⚠️ Chặn thật của bước 5 KHÔNG phải kích thước — là hai closure trên `model`

`pickState.apply func(model, pickAnswer) model` và `ask(…, confirm func(model) model)`. Closure không đi theo package. Chữa: biến thành **dữ liệu** (token/đích đến mà **package không bao giờ diễn giải**), rồi dời mới là dời thuần.

## Khuôn mẫu đã dùng lại nhiều lần

- **forwarder một dòng** giữ ~200 call site không đổi (bước 1)
- **embed, không alias** khi màn có field kiểu enum của client → `everyScreen` diff **rỗng** (4b)
- **`(self, Action)`** là hình dạng dùng cho `Update`/`Confirmed`/`Picked` — nên bước sau không convert lại
- **giá trị mờ** (`Into any`, `About any`, destination) khi payload là từ vựng của client
- ⚠️ **`Action` KHÔNG nhận được**: một tập id (picker → `PickResult`), và `tea.Cmd` (làm `Action` mất comparable → `Update` trả **ba** giá trị)

## Bước 6c — hai quyết định đo được

**`Context.Authoring bool`, nought = read-only.** Không phải mặc định tiện tay: **nửa an toàn phải là nửa mà một lần quên khai rơi vào**. Và **footer phải theo phím** — `picker.go:311`: *"một phím được thông báo trên màn bỏ qua nó thì tệ hơn phím không ai được kể."*

**`Target.Fight` = mối nối PvP.** `Target` là *yêu cầu* client biến thành màn của **chính nó** — nên `pairing()` là một hàm, và đó là thứ duy nhất đổi khi server gửi phe away.

⚠️ **Mirror phẳng làm phép đo mù**: hai phe giống hệt thì hoán vị được, nên không gì (kể cả golden) thấy được một lần đổi chỗ.

## Bước 4b — mẹo EMBED, và `everyScreen` không đổi một dòng

`from` là enum `screen` của riêng binary → **không đi theo được**. Thay vì dời `from` lên model (phải sửa 3 entry `everyScreen`), client **bọc**:

```go
type blurbScreen struct { draw.BlurbScreen; from screen }
```

Không thêm state, promote `View`, `language_test.go` diff **RỖNG**. Bước 3 và 4a đều phải tốn field-case churn ở đó; 4b tốn 0. **Dùng lại mẹo này ở bước 5.**

`describe.go` **KHÔNG dời** (`m.browse`×22, `model`×14) — applier + xử lý phím đi bộ trên màn gọi, thuộc client.

## Đo thật, khác cách #193 mô tả

`cmd/hexforge-tui` = **10k dòng production + 13.7k test** (không phải "23k dòng gắn chặt"). `model` là container của struct per-screen, mỗi màn một file, mỗi màn `newXScreen(lib)`. Ghép chặt chỉ **3 loại**:

1. helper vẽ + palette — mọi màn đọc, không màn nào quyết định → **bước 1**
2. `m.screen = X` — navigation → **bước 2** (thay bằng giá trị trả về, binary tự map enum)
3. **đọc state màn khác** — chỉ 2 màn nặng

## Phát hiện thiết kế: `blurb`/`preview` không phải màn hình

Chúng là **bộ mô tả một chủ thể**, hiện thò tay vào màn đã gọi: `blurb` đọc `m.browse`(15) + `m.play`(9) + `m.skills`(7); `preview` đọc `m.browse`(10). Tách = **truyền subject vào** — cải thiện thật, và chỉ 637 dòng. Bước 4.

6 màn **gần như miễn phí** (chart/elements/statuses/species/builds/passives, ~1400 dòng): ghép duy nhất là `m.screen`. Bước 3.
`browse`/`skills`/`squads` là nhóm nặng, cần dời cả `picker` (997 dòng). Bước 5.
Bước 6 = `cmd/hexarena` full-screen trên package đó.

## ⚠️⚠️ KHÔNG có golden nào cho màn hình

`cmd/hexforge-tui` **không có `testdata/`**. 10k dòng vẽ chỉ được bao bởi test **thuộc tính** (độ rộng, dịch, lọt gloss). Nên **"không golden nào nhích" không nói gì về render** — muốn khẳng định trung tính thì phải **tự chụp byte**.

## Cách chụp byte (đã dùng, đã validate)

`startIn(t, lang, dir)` trong `tui_test.go` nhận đường dẫn **cố định** — đó là cửa. Dựng worktree gốc trên `origin/main`, chụp cả hai, diff. Kết quả bước 1: **88 render (11 màn × 2 lang × 4 cỡ), 3091 dòng, byte-identical.**

⚠️ **Phải kiểm instrument HAI CHIỀU trước khi tin số:** (a) deterministic với chính nó, (b) **có thể fail** — đổi 1 glyph `ellipsis` dịch 52 dòng.

⚠️ **Lần chụp đầu KHÔNG deterministic.** Hai nguồn bẩn:
1. đường dẫn `t.TempDir()` **được vẽ vào header** → điểm cắt dịch theo độ dài tên temp.
2. **2 entry của `everyScreen` tự dựng model trên scratch riêng** → đội hình do fixture ghi lọt vào bytes.
→ Cách chữa: **một** model trên data dir cố định, `m.enter(screenX)` cho danh sách màn tự chọn, **đừng** duyệt `everyScreen`.

Chưa bao byte: `builds`, `preview`, `squads`, `form`, `spar`, `fight`, `play`.

## Bước 2 — vốn từ ĐO được, 3 thứ, không dư

`internal/screen/action.go`: `Action{Kind, Target, Focus}`; `Kind` = Stay/Back/Quit/Raise; `Target` = NoTarget/Chart/Statuses + **`TargetCount`**. 6 màn ký `update(c Context, msg) (<own>, Action)` — **thôi nhận `model`, thôi trả `tea.Cmd`**. `internal/screen` **không** import bubbletea (key type ở lại signature của client) → `Action` sạch dep.

⚠️ **`Target` không có entry trong map của client = `Raise` âm thầm không làm gì.** Chữa bằng `TargetCount` khai như `element.Count`/`status.CategoryCount`, và test **walk theo count** chứ không walk theo entry ai đó nhớ liệt kê. Đo hai chiều: bỏ entry → đỏ; **thêm** một `Target` không map → đỏ và **gọi tên**.

⚠️⚠️ **GOLDEN KHÔNG BAO TRANSITION.** 200 render, 0 chuyển màn — mọi consumer của `everyScreen` **chỉ vẽ**, và 2 entry chạm mấy màn này thì gán `m.screen` trực tiếp. Nên đổi navigation chỉ được giữ bởi **driven test**: `chart_test.go`, `trait_status_test.go`, `builds_test.go`, `reference_test.go`, 6 assert `m.screen` trong `tui_test.go`. Hai guard **bù nhau, không thay nhau**.

Hai coupling tự rụng ở bước 2: `chart` bỏ `screenElements` hardcode → `Back` (kiểm lại: `elements` vẫn là raiser duy nhất, và **không fixture nào bấm key trên chart**); `passives` thôi đọc `m.statuses` → trả `Raise{Statuses, Focus: id}`, client áp focus. `statusesScreen.from` xoá, client giữ **một ô** — đúng nghĩa cũ vì `screenMenu` **là giá trị 0** của `screen` và ô xoá trong nhánh `Back` (đúng chỗ field cũ tự xoá).

## Bước 4a — describer nhận subject, và món nợ #203 trả xong

`blurb` đọc `m.browse`×8 + `m.skills`×4 + `m.play`×4; `preview` đọc `m.browse`×10. **Màn kéo từ ba màn khác thì không dời được.** Đảo chiều: raiser **đẩy** một `draw.Subject` qua một applier của client.

Cùng cơ chế với món nợ: `Action.Focus string` + `model.focus` chỉ trả lời `screenStatuses` → xoá cả hai, thay bằng `Subject` + `SubjectKindCount` (khai như `TargetCount`).

**3 kind, không phải 4.** Chiêu-trong-listing và lựa-chọn-trong-trận đo ra là hai **thật ra là một**: cùng id, cùng vị trí, cùng đoạn, cùng heading, cùng footer — khác biệt duy nhất đã nằm trong `At`/`Of`. Gộp `viewSkill`+`viewOption` **cho nhánh listing cái error branch `Lookup` mà nó vốn thiếu**. Đúng luật đã ghi ở *đã loại* `at_stage`: hai vốn từ cho một ý là cái giá.

Subject mang **id, không mang giá trị đã resolve** — raise gọi tên cái nó muốn, không hỏi ai cái đó ở đâu. Đưa `skill.Skill` đã resolve = đẩy lookup **và lời từ chối** sang raiser.

⚠️ Spec tôi không lường: **describer không được đọc màn gọi nó thì cũng không giữ được `update`** — phím mũi tên ở đó đi bộ trên màn kia. Hai `update` phải dời sang `describe.go` cạnh applier.

## ⚠️⚠️ BỨC TRANH CỦA PREVIEW KHÔNG ĐƯỢC ĐO BỞI GÌ CẢ

Đo ở dạng mạnh nhất: **đảo ngược toàn bộ dải vẽ** (`const Ramp`) — mọi điểm sáng thành tối và ngược lại → **23 package XANH, không gì đỏ**. Cũng vậy với đảo trọng số độ sáng, và hoán vị nửa khối `▀`↔`▄`. Thứ duy nhất bắt được là mutation biến **mực thành trống**.

→ Trả lời câu `TODO.md` để ngỏ ("decide what the entry asserts before adding it"): hôm nay nó khẳng định **không gì cả**. Nên **chưa đăng ký `previewScreen` vào `everyScreen`** — đăng ký mà chưa chốt = thêm một golden đồng ý với mọi bức vẽ.

Dữ liệu để chốt sau:
- **Tái lập trong máy: CÓ** — digest byte-for-byte qua nhiều process; rasteriser không đọc clock/map.
- **Qua kiến trúc: CHƯA chứng minh** — `rasterx` gọi `Sin`/`Cos`/`Atan2`/`Tan`, họ hàm stdlib Go có assembly **theo kiến trúc**. Với câu chữ thì hiển nhiên; với bức vẽ thì không.
- **Bản màu không kham nổi**: cùng 55 dòng = **8.4 KB** plain vs **128 KB** màu (mỗi ô một escape). Mọi golden chụp dưới `NO_COLOR` → entry sẽ ghi dải ký tự và để `blockCell` không ai đo.

## ⚠️⚠️ Test phủ đủ đếm CÓ MẶT, không đếm CÓ TÁC DỤNG

Đo được (mutation của tôi): applier entry **có mặt nhưng no-op** → test phủ đủ **VẪN XANH**, 4 test khác đỏ (3 test hành vi có tên + golden client).

Nên hai lớp giữ **hai nửa khác nhau**: phủ đủ giữ *kind có được xử lý không*, test hành vi giữ *xử lý có đúng không*. Không cái nào thừa. Đừng tưởng `walk-the-count` là đủ.

## ⚠️ Bước 4 đã xử: `model.focus` chỉ trả lời `screenStatuses`

Một `Raise` mang `Focus` tới target khác **âm thầm từ chối cả chuyến** — yếu hơn một bậc so với test phủ đủ của map. Hôm nay không ai gửi. **Bước 4 chính là lúc đó**: `blurb`/`preview` được raise **kèm chủ thể**, switch thành nhiều entry → cần đúng cách walk-the-count.

## ⚠️ Bước 3 phải nới walker

`TestNoScreenHoldsItsOwnWording` quét **thư mục package của chính nó** (`os.ReadDir(".")`), nên luật "cmd không được giữ literal người dùng thấy" **không đi theo code** sang `internal/screen`. Bước đầu tiên dời một màn thật **phải** nới nó, không thì thứ nó dời âm thầm thoát luật hai ngôn ngữ. Đúng họ với 5 lần `TODO.md` ghi cho `everyScreen`.

## Đã làm ở bước 1 — và khuôn mẫu đáng lặp

`internal/screen`: `Palette`/`NewPalette`/`Element`/`Plain`/`NewInput`/`Bar` + `Context{Lib,Lang,Style,Width,Height}` với `Text`/`UsableWidth`/`Label`/`LabelAt`/`Wrapped`/`WrappedIn`/`Continued`.

**Khuôn mẫu:** mỗi helper dời đi để lại **forwarder một dòng** trên `model` → **~200 call site không đổi một chữ**, `language_test.go` không cần sửa, `everyScreen` byte-for-byte nguyên. Diff không thể đổi hành vi.

Kéo theo **bắt buộc** (body gọi chúng): `minWidth`/`minHeight`, `detailLabels`/`detailLabelWidth`, `pad`/`wrapWords`/`clip`/`ellipsis`. Và `Palette` **phải export field** (240 site) vì `LabelAt` render qua chúng qua ranh giới package.

`plainTerminal()` **ở lại binary** — đọc env là việc của binary, trả lời câu hỏi thì không. `Plain` giữ tham số `goos`: mutation bỏ `&& goos != "windows"` chỉ đỏ test **trong package mới**, test bên cmd **vẫn xanh** (trên darwin mutant trả cùng đáp án) — xem [[windows-sets-no-term]].

⚠️ `wrappedIn` off-by-one (`room := UsableWidth() - 2 - width - 1`, lấp cột cuối) dời **nguyên văn**. `TODO.md` ghi để nguyên: sửa nó **đổi cái gì vừa màn hình**, nên phải là PR đo riêng.

## Vụn nhưng đáng biết

- `.gitignore` chỉ ignore `bin/` → binary build từ trong một thư mục `cmd/` **commit được**.
- `passives` panic với sách trait rỗng: `clamp(0, 0, -1)` trả 0 rồi index slice rỗng. Có sẵn, không tới được với data đang ship.
- `check.go:81` in đường dẫn data như **body**, không phải header — xem phần golden.
- ⚠️ **`previewScreen` ngoài `everyScreen`** → không test độ rộng, không test dịch, **không entry golden nào**. Màn duy nhất không có lưới. Đăng ký ở 4b.
- ⚠️ **Hai bẫy khi làm mutation proof, cả hai đã dính thật:** (a) `species.go` có `return out.String(), footer` ở **hai** dòng — revert theo *chuỗi* khôi phục nhầm chỗ và **cả suite vẫn xanh**; nhắm theo **dòng**, kiểm bằng `git diff`. (b) **cwd âm thầm rơi về main checkout** khi worktree nó đang đứng bị xoá — tôi đã mutate nhầm cây; dùng **đường dẫn tuyệt đối + assert `pwd`**. File untracked thì `git diff` không kiểm được restore, phải `diff` với backup.

Liên quan: [[hexarena-tui-width-rule]], [[hexarena-battle-screen-budget]], [[fixture-hidden-branch]].
