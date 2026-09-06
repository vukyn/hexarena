---
name: goldens-cannot-be-green-on-two-platforms
description: "⚠️ internal/screen và cmd/hexarena-tui's screens.golden được accept trên WINDOWS: 12 dòng chứa `\\` ở chỗ filepath.Join viết `/`, nên hai test golden ĐỎ VĨNH VIỄN trên macOS/Linux và xanh trên Windows — `make check` KHÔNG xanh ở c8edde3; ai accept sau sẽ lật ngược lại"
metadata:
  type: project
---

⚠️ **`make check` không xanh trên macOS hay Linux, và nguyên nhân không phải code.**
Đo 2026-09-06 trên bản `git archive` sạch của `c8edde3`:

	internal/screen   TestEveryMovedScreenDrawsWhatTheGoldenHolds  ĐỎ
	  golden "đã sửa vine_whip trong ..\seed\data\skills.json"
	  drawn  "đã sửa vine_whip trong ../seed/data/skills.json"
	cmd/hexarena-tui  TestEveryScreenDrawsWhatTheGoldenHolds       ĐỎ
	  golden "… vào data\battles\phe-0-vs-phe-1-seed1.json"
	  drawn  "… vào data/battles/phe-0-vs-phe-1-seed1.json"

**Đúng 12 dòng**: 8 trong `cmd/hexarena-tui/testdata/screens.golden` (entry
`a saved battle`, hai ngôn ngữ × hai cỡ, mỗi render hai dòng) và 4 trong
`internal/screen/testdata/screens.golden` (entry `edited a skill`). Vào từ #319.
`cmd/hexforge-tui`'s golden **không có dòng nào** — fixture của nó không vẽ note
nào chứa đường dẫn.

**Why:** ghi chú của một lần ghi file nêu **đường dẫn thật** và đường dẫn đó do
`forge.Library` dựng bằng `filepath.Join`, tức `\` trên Windows và `/` ở mọi nơi
khác. Golden so **từng byte cả file**, nên nó ghim luôn cả hệ điều hành của người
accept. Ai accept lần sau sẽ lật 12 dòng đó về phía mình và làm đỏ phía kia — một
vòng lặp, không phải một lần.

**How to apply:**
- ⚠️ **Đừng `make golden` rồi commit luôn nếu bạn ở macOS/Linux.** Chạy xong,
  **đọc diff** và trả 12 dòng đó về dạng đã commit (`\`), không thì PR của bạn
  chở một thay đổi platform chẳng liên quan gì. Bước 5b làm đúng vậy: dùng script
  chỉ lật lại dòng nào mà **dạng backslash của nó có trong file đã commit** —
  đừng replace mù `/`→`\` trong đường dẫn, vì golden của `internal/screen` còn có
  dòng `../seed/data/...` khác vốn đã là `/` từ đầu.
- Hai test golden đỏ ≠ việc bạn làm sai. Kiểm bằng `git archive HEAD` ra thư mục
  tạm rồi chạy chúng ở đó — không đụng gì tới worktree, và nói ngay được cái đỏ
  là của ai.
- Ba đường ra, mỗi đường là một **quyết định** chứ không phải bản sửa: viết ghi
  chú bằng `path` thay vì `filepath` (là đổi wording, mà ghi chú đang nêu một file
  người đọc phải mở được), cho fixture chuẩn hoá dấu phân cách trước khi ghi (là
  scrub — comment của chính `bodyOf` phản đối: fixture âm thầm ngừng bỏ thứ gì là
  fixture âm thầm bắt đầu ghi sai dòng), hoặc **khai một platform** cho golden và
  ghi cạnh `make golden`. ⚠️ Wording nằm ở `i18n.NoteWrote`/`NoteBattleVerify`,
  đường dẫn nằm ở `forge.Library` — hai bản sửa ứng viên rơi vào hai package khác
  nhau.

Cùng họ với [[hexarena-crlf-data-digest]]: ở đó một checkout Windows làm **data
digest** lệch và hai peer không join được nhau; ở đây một accept Windows làm
**golden** lệch và hai platform không cùng xanh được. Cả hai đều là "byte trên
đĩa khác nhau giữa hai máy cùng commit", và cả hai đều không có test nào tự nói ra.
