---
name: hexarena-tui-i18n
description: "hexforge-tui speaks vi (default) + en; facts live in forge as typed errors, wording in internal/i18n; CLI stays English"
metadata:
  type: project
---

`cmd/hexforge-tui` is localised vi/en with **Vietnamese as the default** (2026-08-25). `--lang en` or `HEXARENA_LANG=en`, flag beats env, unknown value is an error; `ctrl+l` toggles mid-form without losing typed text.

**The structural move: facts and wording came apart.** `internal/forge` used to format English inside its logic (`CheckCarry` returned a `fmt.Errorf`). It now carries **19 typed refusals** holding the values they complain about, plus structured summaries; the old string helpers are thin wrappers over them, so `cmd/hexforge`'s English cannot drift from the facts. `internal/i18n` imports `forge`; **`forge` never imports `i18n`**.

Catalog is **one fixed-size array per language** (131 keys) indexed by a `Key` const — a mistyped key is a build error, not a blank line. **No English fallback**: a missing Vietnamese string fails a test. Tests also assert both languages use the same format verbs (catches `%!d(MISSING)`).

**Deliberately NOT translated:** `cmd/hexforge` CLI (scripts/pipes use it), `internal/core` (untouched), element ids (`fire`, `metal`…) and stat labels (`hp atk def spd acc ddg`) — those are what the author *types* and what the data files store.

**Vietnamese glossary in use:** hệ (element) · chỉ số (stat) · bộ chiêu (kit) · chiêu (skill) · mẫu vai trò (archetype) · hạn mức (stat budget) · chịu được (absorbs/effective HP) · giai đoạn (stage) · trần cấp (level cap) · dàn nhân vật (cast) · nguồn / tác phẩm (origin) · mang được / không mang được (carry). Carry refusal reads `hệ fire không mang được chiêu "sever" (hệ metal)`.

**Two pre-existing bugs the i18n pass exposed** (neither caused by it):

1. The declared 72-column minimum was already too small — the old English browse footer measured 80 cells, so a 72-column terminal silently dropped `esc back · q quit`. Minimum is now **80x24** with a test holding every line under it.
2. The form's summary rows (`budget`, `carries`) assumed the detail panes' fixed label width while the field column is sized from the widest label — off by one **right in English and left in Vietnamese**, which is why it had gone unnoticed. Fixed by threading the computed width through.

**Data-name glosses (2026-08-25).** In vi, elements/archetypes/skills show `english <việt>` — `grass/electric <cỏ/điện>`, `skirmisher <du kích>` — English stays the name so the user can check the translation. 35 glosses in `internal/i18n/gloss.go`, three tables. **This is a lookup ALLOWED TO MISS and that is the design**: skill/archetype ids live in `internal/seed/data`, so an unglossed id renders as the bare English name, never a placeholder. Deliberately NOT in the strict catalog — folding data ids in would make every new skill in a JSON file a build-breaking gap in Go, the exact cost that keeping balance out of Go avoids. Elements are the one must-be-complete case (Go enum). Kit glosses go on a dimmed continuation line — 5 brackets don't fit 80 cols.

⚠️ **Angle brackets since PR #172 (`8b8f8d8`), because round ones NESTED.** The battle log names a status's source trait as a parenthetical of its own, so a round gloss inside it read `(virulence (độc lực))`. The gloss is the **inner** thing wherever the two meet, so the gloss is what changed shape — and it changed **everywhere**, because `glossBracket` is its one definition and a log spelling it differently from the reference screens is the second definition that const exists to prevent. `<x>` and `(x)` are both two cells, so **no width measurement moved**; no golden held a gloss so none moved; the whole change was one const plus five hardcoded literals in two test files.

**Label renames the user asked for:** dàn→danh sách · mượn từ→nguồn tham khảo (label only — three prose strings keep `mượn từ` as a *verb*; "được nguồn tham khảo" is ungrammatical) · dựa trên→lối chơi (+ English `tuned from`→`playstyle`, so both languages name one concept) · chịu được→máu quy đổi / `effective hp`.

**⚠️ Terminal renders some precomposed Vietnamese as 2 cells — NOT a code bug.** The user saw `chị u được`. Source is correct precomposed UTF-8 and `lipgloss.Width` = `utf8.RuneCountInString` = `runewidth.StringWidth` = 9, with `ị` U+1ECB at width 1. Their font/terminal draws it wide. **Do not work around it** — distorting the width measurement to suit one font misdraws every other terminal. 23 other precomposed letters sit in padded columns (`ậ` `ệ` `ộ` `ể` `ổ` `đ` `ả` `Ế`), so if those look off too the cause is confirmed as the font.

**Label columns must be measured, never constants.** `detailLabelWidth` was 11 and could hold neither `nguồn tham khảo` (15) nor `effective hp` (12); now computed (vi 16 / en 13), like the form's. A constant is what caused the form's summary rows to sit a cell out of line.

**How to apply:** Vietnamese runs 20–30% longer than English, so any new screen needs its width re-measured, not eyeballed. Diacritics measure one cell each and there is a test banning `Mn` combining marks, so a decomposed `ế` pasted in later fails rather than drifting a column. Related: [[hexarena-cast-authoring]].

## PR #161 — category viết HAI LẦN: vị ngữ vs danh từ (merged 6f71281)

`rapid_spin` tiếng Anh in ra `Strips 1 stack of stat_debuff and dot.` — enum Go trên
dòng người chơi đọc. Nguyên nhân: câu strips gọi `l.glossed(category)`, mà `Gloss()`
**cố ý** chỉ trả lời tiếng Việt (`gloss.go`: tiếng Anh thì id hiện đúng như data
viết). Luật đó đúng với **data id** và sai với category — category là **enum Go**.
Tiếng Việt chạy được là NHỜ MAY: bảng `categoryGloss` tình cờ tồn tại.

⚠️ **Không phải lỗi thiếu lookup.** `Lang.StatusCategory` đã đủ 7 chữ cả 2 thứ tiếng
nhưng là **VỊ NGỮ** (`lowers a stat`) cho cột màn tra cứu; nhét vào ra "Strips 1 stack
of lowers a stat". Cần họ thứ hai: `StatusCategoryNoun` = **danh từ**. Một câu và một
cột cần hai TỪ LOẠI khác nhau — đặt cạnh nhau, không nhét cờ vào một hàm.
EN dạng danh từ không đếm được, không mạo từ: stat reduction · damage over time ·
turn loss · stat increase · shielding · healing over time · forced targeting
(không bê `hiệu ứng X` sang vì "an X effect" cần mạo từ mà khung `stack of %s` không
có chỗ đặt). Chuỗi VI **DỜI** khỏi `categoryGloss` (xoá bảng), không copy.

⚠️ **Xoá bảng làm lộ bóng xuyên-bảng:** `Gloss` duyệt MỌI bảng cho MỌI id, mà
`skills.json` có chiêu id `taunt` trùng tên category → `Gloss("taunt")` lâu nay trả
tên CATEGORY cho một CHIÊU. Vô hình vì chiêu có tên authored và `SkillName` ưu tiên
nó. `TestNoIDIsGlossedTwice` không thấy: nó soi trùng TRONG bảng, không soi XUYÊN bảng.

⚠️ Test regression phải khớp **từ trần**, không phải substring — `shielding` chứa
`shield`, chỉ một trong hai là lỗi.

`BlurbStrips` viết số ít `"Strips %d stack of %s."` trên đúng key chỉ nhận ≥2.
Không golden nào đỡ vì chưa chiêu ship nào gỡ >1 cộng dồn — fixture mới chạm tới.

Còn mở sau PR: `X and Y and Z` không dấu phẩy (hình dạng `l.join` cho MỌI danh sách
trong describe.go). `cmd/hexarena` `tui.Extras` liệt kê enum trần → **decided against**:
hàm không nhận `Lang`, mọi trường cạnh nó là id/số trần; dịch nó = dịch CẢ DÒNG.

## PR #164 — bảng rỗng cột thì BỎ cột (merged 6871867)

Dòng nội tại tiếng Anh = id đệm tới cột rộng nhất + dấu ngăn + rỗng; thanh chọn phủ
cả dải trống. Nguyên nhân là **TÊN**, không phải lookup: `PassiveName`/`SpeciesName`/
`BuildName` trả rỗng cho EN **cố ý** — tên nằm cạnh id được authored một lần bằng
tiếng Việt, hiện nó lên màn EN là **rò rỉ chứ không phải bản dịch**; người đọc EN
nhận id, vốn là tên data tự đặt. `detailColumn(m, rows)` trả 0 khi không dòng nào có
chi tiết (0 = cách `speciesRow`/`passiveRow` đã dùng cho cột bị bỏ).

⚠️ Điều kiện phải là **"các dòng TRƯỚC MẶT rỗng cột"**, tính **một lần mỗi lần vẽ**.
Kiểm tra ngôn ngữ = proxy (sai ngày list VI rỗng / list EN có chi tiết); kiểm tra
từng dòng = bảng răng cưa. Lỗi rộng hơn tưởng: `pickElements`/`pickArchetypes`/
`pickCharacters` cũng rơi xuống `Gloss` (rỗng cho EN **by construction**) → cả 3
cùng vẽ cột trống, 2 trong 3 đã nằm trong `everyScreen` suốt.

⚠️ **Mutation: bẻ điều kiện thành kiểm tra ngôn ngữ thì test tiếng Việt VẪN XANH.**
Một khẳng định chỉ-tiếng-Việt không thấy được proxy hình-dạng-tiếng-Việt; nửa tiếng
Anh của test "cả hai thứ tiếng" mới phân biệt. Bài chung: test chống proxy phải chạy
ở phía proxy ĐOÁN SAI, không phải phía nó đoán đúng.

⚠️ **Rò rỉ CÒN MỞ (TODO):** picker species đọc `kind.Name` trần cho cả 2 thứ tiếng →
`dragon rồng` trên màn EN, đi vòng qua `SpeciesName` (mà *listing* species đã dùng).
`TestTheScreensGlossEveryDataName` sinh ra để bắt đúng thứ này và **mù**, vì
`everyScreen` không đăng ký picker species → vá là HAI việc: nhánh code + entry.
`pickOrigins` đọc trần y hệt nhưng KHÔNG cùng lỗi (Naruto/Pokémon là danh từ riêng,
không có `Lang.OriginTitle` nào bị đi vòng).

## PR #165 — picker species hỏi Lang, và cho test QUÉT TỚI màn đó (merged b18c1bb)

Picker species vẽ `dragon rồng` trên màn EN: detail đọc `kind.Name` trần cả 2 thứ
tiếng, đi vòng `Lang.SpeciesName` (trả rỗng cho EN cố ý). *Listing* species đã dùng
accessor trên cùng cuốn sách → picker là chỗ duy nhất lệch. Sửa xong, EN rỗng cột →
`detailColumn` tự bỏ cột (miễn phí từ #164).

⚠️⚠️ **BÀI LỚN NHẤT: màn nào test-sweep không vẽ thì KHÔNG có test rò rỉ, KHÔNG có
test width, KHÔNG có test dịch.** `TestTheScreensGlossEveryDataName` sinh ra để bắt
đúng lỗi này, gom đủ mọi `kind.Name`, quét từng dòng — và **XANH**, vì nó duyệt
`everyScreen` mà `everyScreen` không đăng ký picker species. Nhánh có **HAI cửa chưa
đo** (form species + allowlist species trên form chiêu). Nên vá lỗi loại này luôn là
HAI việc: nhánh code + entry lẽ ra bắt được nó. (Lần 3 của cùng hình dạng: trước đó
`plainTerminal` — mọi test set NO_COLOR rồi return trên nhánh; và `playScreen` —
fixture chơi hết trận nên footer 82/83 cell không ai đo.)

⚠️ **Mutation phải tách HAI NỬA:** (a) nhánh hỏng + CÓ entry → sweep đỏ; (b) nhánh
hỏng + BỎ entry → **sweep XANH** (đúng đời thật); (c) fix + bỏ entry → chỉ test entry
đỏ. Test entry KHÔNG được đỏ ở mutation (a) — nó khẳng định màn ĐƯỢC DUYỆT, không
phải ô đó viết gì.
⚠️ Test entry tìm màn bằng **model nó LÀ** (`picker.kind == pickSpecies`) rồi so
**byte-for-byte** với picker mở độc lập, 2 thứ tiếng. Kiểm tra bằng KEY của map sẽ
xanh với entry giữ nhầm model.

Đăng ký thêm picker origins, **nhánh KHÔNG đổi** (Naruto/Pokémon = danh từ riêng,
không có `Lang.OriginTitle` bị đi vòng) — lý do là PHỦ SÓNG: dòng tiêu đề của nó
không màn nào khác vẽ, nên chữ đó chưa từng đo với mốc 79 cell. Allowlist species
không thêm dòng nào mới → cố ý không đăng ký.

Còn mở (2 mục TODO mới): `stage.Name` in trần 5 chỗ, dạng 3 của Naruto authored
**`Tiên nhân`** = tiếng Việt trên màn EN (không cùng hình dạng: không có
`Lang.StageName`; sweep cũng KHÔNG gom tên stage). Và `tui.DetailPassives`
(`internal/tui/describe.go:46`) đọc `one.Name` trần, cách dòng 30 dùng đúng
`lang.SkillName` 12 dòng — âm ỉ vì caller duy nhất hardcode `i18n.Vi`.

## PR #166 — tên stage là KHOÁ, refuse ở parser (merged 0a81b55)

`Tiên nhân` (dạng 3 của Naruto) in trần trên màn EN. **Vẽ trần là ĐÚNG — lỗi ở DATA.**
`stage.Name` là **identifier**, không phải display text; 4 bằng chứng: `Line.Resolve`
so `candidate.Name == stage`; `Stage.After` gọi cha **bằng tên**; roster/squad ghi
`"stage":"Ivysaur"` + learnset gate liệt kê tên stage (đều **gõ tay** vào file mà
line không thấy); `Line.Validate` từ chối 2 stage trùng tên → **ràng buộc khoá**.
KHÔNG có `Lang.StageName` — luật nhà: id hiện y như data viết, cả 2 thứ tiếng.
`Tiên nhân` = bản dịch VI của 仙人, mà line đã là romaji (Naruto/Shippuden) → `Sennin`
là **tên vốn phải ở đó**, không phải tên bịa.

⚠️ **"ASCII" chỉ là DẠNG CƠ HỌC, không phải luật.** Luật: chuỗi **gõ tay 4 chỗ + so
bằng `==`** phải **gõ được** và có **ĐÚNG MỘT cách viết** — `Tiên` có 2 encoding
Unicode vẽ y hệt (`ê` dựng sẵn vs `e`+U+0302 tổ hợp), `==` coi là 2 tên → **khoá nhìn
đúng vẫn trượt IM LẶNG**. Cấm space đầu/cuối/đôi = cùng lỗi đó trong ASCII. Không siết
hơn: space/digit/hyphen/apostrophe/period hợp lệ (`Mega Charizard X`, `Ho-Oh`,
`Porygon2`).

⚠️ **Luật loại này phải mutation HAI CHIỀU**: trả `Tiên nhân` về → hỏng cả khâu LOAD
(4 package đỏ; bảng refuse có cả bản **decomposed**, viết bằng escape vì mắt không
phân biệt và editor tự chuẩn hoá mất); và 4 tên form hợp lý (`Sage Mode`/`Ho-Oh`/
`Farfetch'd`/`Sennin2`) phải load sạch → **siết nhầm cũng bị bắt**, không chỉ nới nhầm.

⚠️ Parser vs test là HAI KHẲNG ĐỊNH KHÁC NHAU: test ràng cái đang ship; refusal ràng
**mọi line ai viết sau này** + dùng lại được cho form authoring lúc đang gõ (lý do
`cast.ValidateID`/`ValidateImagePath` được export). Seam là `Line.Validate` —
`progression.ParseLine` KHÔNG tồn tại.

⚠️ **KHÔNG thêm tên stage vào `TestTheScreensGlossEveryDataName`**: sweep đó khẳng
định *id đi kèm GLOSS của nó*, mà tên stage không có gloss → thêm vào là khẳng định
NGƯỢC LẠI. Sweep thứ 2 (`freeText`) cũng bất lực: nó gom `stage.Name` vào danh sách
**miễn trừ**, và kể cả bỏ miễn trừ thì marker VI trong list là `"nhân vật"` chứ không
phải `"nhân"` — spot-check theo từ cố định không bắt được. Thêm lý do đặt luật ở parser.

## PR #167 — block nội tại hỏi Lang, khớp 2 block cạnh nó (merged 721a3ff)

`tui.DetailPassives` dựng heading từ `passive.Passive.Name` TRẦN, **cách 12 dòng**
dưới `Detail` vốn dùng `lang.SkillName`. Hai đáp án cho một câu hỏi trong MỘT file.
**Âm ỉ**, không cháy: caller duy nhất (`cmd/hexarena:391`) hardcode `i18n.Vi` → raw
read trùng accessor cho giá trị duy nhất `lang` từng nhận. Không có gì người dùng
thấy đổi hôm nay; nó chỉ rò ngày `cmd/hexarena` nhận `--lang`.

Guard đổi sang dạng **cả ba block trong file** dùng (`!= "" && != ID`) — điểm của
mục này là **ba hàm khớp nhau bằng mắt thường**, không phải bản thân cái fix.

⚠️ Mutation trái dự đoán (theo hướng TỐT): tôi tưởng `Detail` → `declared.Name` sẽ
KHÔNG đỏ vì 19 chiêu ship đều rơi về bảng compiled. **Sai** — có chiêu mang `name`
authored trong `skills.json` (file này tool-written), nên test cross-block phân biệt
được raw-read ở CẢ HAI hàm.
⚠️ Bỏ vế `!= ID` của guard → **không test nào đỏ**: không chiêu/nội tại ship nào tự
đặt tên bằng chính id mình. Giữ vế đó vì NHẤT QUÁN, không vì có test đỡ — ghi thẳng
vào commit thay vì giấu.

Test cross-block so ba hàm **qua thứ chúng VẼ** (skill/trait/status khác type), tính
chất chung duy nhất: block tiếng Anh mang id trần và không chữ tiếng Việt nào.
