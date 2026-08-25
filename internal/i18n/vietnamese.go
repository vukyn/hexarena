package i18n

// vietnamese is the client in Vietnamese, and the language it opens in.
//
// It is written the way a person here would say it out loud, not translated
// word for word from the English array beside it: several lines are shorter
// than their English counterparts and a few say the thing in a different order,
// because the point is that it reads comfortably, not that it lines up with an
// original.
//
// What stays in English is deliberate and listed in the package comment: ids of
// every kind, the six stat labels, and diagnostics that come from
// internal/core, which describe the shape of a data file and get a Vietnamese
// lead-in rather than a Vietnamese rewrite.
var vietnamese = [keyCount]string{
	MeasuringTerminal: "đang đo màn hình…",
	// Every line here is short on purpose: this is the one screen drawn into a
	// window already known to be too small, so it has to survive being cut.
	TerminalTooSmall: `màn hình quá nhỏ

cần ít nhất %dx%d
đang có %dx%d

Kéo rộng cửa sổ, hoặc
dùng hexforge: cùng
danh sách nhân vật,
cùng các bước kiểm tra,
cỡ nào cũng chạy được.

q hoặc ctrl+c để thoát`,
	Truncated:   "… bị cắt bớt; cửa sổ cao hơn sẽ thấy hết",
	NoArguments: "không nhận tham số nào, mà nhận được %v",
	NotATerminal: "đầu ra không phải màn hình terminal, mà chương trình toàn màn hình sẽ đổ mã điều khiển vào đó.\n" +
		"Hãy dùng hexforge: cùng danh sách nhân vật, cùng các bước kiểm tra, nhận cờ dòng lệnh\n" +
		"và đọc được pipe; `hexforge check` in ra đúng những gì màn hình kiểm tra ở đây hiển thị",
	DataFlagUsage:     "thư mục dữ liệu để đọc và ghi",
	LanguageFlagUsage: "ngôn ngữ hiển thị: vi hoặc en",

	MenuHeading:            "bạn muốn làm gì?",
	MenuCast:               "danh sách nhân vật",
	MenuCastDetail:         "xem các nhân vật đã tạo, ở bất kỳ cấp nào",
	MenuNewCharacter:       "tạo nhân vật",
	MenuNewCharacterDetail: "tạo một nhân vật, xem hạn mức và hệ ngay khi gõ",
	MenuOrigins:            "nguồn",
	MenuOriginsDetail:      "các tác phẩm nhân vật được mượn từ, và thêm tác phẩm mới",
	MenuCheck:              "kiểm tra",
	MenuCheckDetail:        "xem ảnh có đủ chưa và hạn mức có bị vượt không",
	MenuNote: "Mọi thứ ghi ở đây đều qua đúng các bước kiểm tra của hexforge, còn game thì\n" +
		"chạy từ bản dữ liệu nhúng sẵn — build lại thì sửa đổi mới vào được trận.",
	MenuFooter: "↑/↓ chọn · enter mở · ctrl+l English · q thoát",

	ConfirmFooter:  "%s [y/N] · ctrl+c thoát",
	ArtPresent:     "có",
	ArtMissing:     "THIẾU",
	ChoicePosition: "%d/%d",

	FormHeading:  "nhân vật mới",
	FormSubtitle: "mọi thứ kiểm tra ở đây đúng bằng lúc ghi, tính ở cấp %d",
	FormFooter:   "↑/↓ ô · ←/→ chọn · ctrl+s ghi · esc quay lại · ctrl+l English · ctrl+c thoát",
	FormDiscard:  "bỏ nhân vật đang tạo?",
	// The stat rows are labelled hp atk def spd acc ddg in both languages; see
	// forge.ShortStat for why those six are not translated.
	FieldID:             "id",
	FieldName:           "tên",
	FieldOrigin:         "nguồn",
	FieldArchetype:      "mẫu vai trò",
	FieldArt:            "ảnh",
	FieldKit:            "bộ chiêu",
	FieldElement:        "hệ",
	FieldBiography:      "tiểu sử",
	NoneCatalogued:      "chưa có mục nào",
	NoArtToChoose:       "chưa thấy ảnh nào trong %s — cứ gõ đường dẫn vào đây",
	CurveAgainstCeiling: "%d → %d, trần %d",
	OverTheCeiling:      "VƯỢT TRẦN",
	LabelBudget:         "hạn mức",
	LabelCarries:        "mang chiêu",
	BudgetWithin:        "%s %d/%d, còn dư %d",
	BudgetOver:          "%s %d/%d, VƯỢT HẠN MỨC %d",
	CarryNoElementYet:   "chưa chọn hệ — %s",
	CarryRefused:        "KHÔNG — %s",
	CarryAccepted:       "ĐƯỢC — %s mang được mọi chiêu trong bộ",
	WriteRefused:        "chưa ghi được: %s",

	KitTakesAnyElement:    "bộ chiêu này toàn chiêu trung tính nên hệ nào cũng mang được",
	KitNeeds:              "bộ chiêu này cần hệ %s",
	PresetTakesAnyElement: "%s (hệ nào cũng được)",
	PresetNeeds:           "%s (cần %s)",
	ElementJoiner:         " và ",

	BrowseHeading:          "danh sách nhân vật",
	BrowseShowing:          "đang xem %s (%d trong %d nhân vật)",
	BrowseAllOrigins:       "tất cả nguồn",
	BrowseFooter:           "↑/↓ nhân vật · ←/→ cấp · f lọc · ctrl+l English · esc quay lại · q thoát",
	BrowseNothingHere:      "không có gì ở đây.",
	BrowseNothingAuthored:  "Chưa có nhân vật nào. Chọn \"tạo nhân vật\" ở menu.",
	BrowseNoneFromThisWork: "Chưa có nhân vật nào mượn từ tác phẩm này. Bấm f để đổi bộ lọc.",
	LabelFrom:              "nguồn tham khảo",
	LabelPlaystyle:         "lối chơi",
	LabelElement:           "hệ",
	LabelKit:               "bộ chiêu",
	LabelArt:               "ảnh",
	LabelStages:            "giai đoạn",
	LabelBiography:         "tiểu sử",
	LabelAtLevel:           "cấp %d",
	LabelEffectiveHP:       "máu quy đổi",
	StageInWords:           "giai đoạn %s",

	CheckHeading:        "kiểm tra",
	CheckFooter:         "↑/↓ chọn · r đọc lại · ctrl+l English · esc quay lại · q thoát",
	CheckPassed:         "ĐẠT — không có vấn đề gì",
	CheckFailed:         "KHÔNG ĐẠT — %d vấn đề",
	CheckCounts:         "%s: %d nguồn, %d mẫu vai trò, %d nhân vật",
	CheckNothingToCheck: "chưa có nhân vật nào để kiểm tra.",
	ColumnCharacter:     "nhân vật",
	ColumnArt:           "ảnh",
	ColumnEffectiveHP:   "máu quy đổi so với hạn mức, ở trần cấp",
	CheckDoesNotResolve: "không tính được chỉ số: %s",
	CheckOverBudget:     "VƯỢT",
	CheckProblem:        "vấn đề: %s",
	CheckNote: "phần này đọc file trên đĩa; game chạy từ bản nhúng sẵn, nên sửa xong phải\n" +
		"build lại thì mới vào được trận",

	OriginsHeading:     "nguồn",
	OriginsSubtitle:    "các tác phẩm mà nhân vật được mượn từ",
	OriginsFooter:      "↑/↓ chọn · a thêm tác phẩm · ctrl+l English · esc quay lại · q thoát",
	OriginsEmpty:       "danh mục chưa có tác phẩm nào. Bấm a để thêm.",
	OriginsCastCount:   "%2d nhân vật",
	OriginsTally:       "%d tác phẩm · loại: %s",
	OriginAdded:        "đã thêm %s (%s) vào %s",
	LabelNote:          "ghi chú",
	OriginFormHeading:  "thêm tác phẩm",
	OriginFormSubtitle: "nhân vật chỉ ghi được tên tác phẩm đã có trong danh mục",
	OriginFormFooter:   "↑/↓ ô · ←/→ loại · ctrl+s thêm · esc quay lại · ctrl+l English · ctrl+c thoát",
	OriginFormHint:     "năm để trống được nếu không rõ; ghi chú là chữ tự do",
	OriginFormDiscard:  "bỏ tác phẩm đang thêm?",
	OriginFieldID:      "id",
	OriginFieldTitle:   "tên",
	OriginFieldMedium:  "loại",
	OriginFieldYear:    "năm",
	OriginFieldNote:    "ghi chú",
	AddRefused:         "chưa thêm được: %s",

	ErrorIDTaken:            "nhân vật %q đã có trong danh sách rồi",
	ErrorMissingName:        "nhân vật cần có tên hiển thị",
	ErrorUnknownOrigin:      "không có nguồn %q; thêm bằng lệnh %q",
	ErrorUnknownArchetype:   "không có mẫu vai trò %q; đang có: %s",
	ErrorOriginTaken:        "nguồn %q đã có trong danh mục rồi",
	ErrorEmptyKit:           "nhân vật không có chiêu nào thì đến lượt cũng chẳng làm được gì",
	ErrorDuplicateSkill:     "chiêu %q bị ghi hai lần",
	ErrorUnknownSkill:       "không có chiêu nào tên %q",
	ErrorUnknownElement:     "không có hệ nào tên %q",
	ErrorMissingElement:     "chưa nhập hệ",
	ErrorAffinityCount:      "%q liệt kê %d hệ; chỉ nhận một hệ, hoặc hai hệ cách nhau bằng dấu /",
	ErrorAffinityCounters:   "%s ghép hai hệ vốn đã khắc nhau",
	ErrorAffinityUndeclared: "%s có hệ không nằm trong bảng hệ",
	ErrorAffinityRefused:    "bảng hệ không nhận %s: %v",
	ErrorCarry:              "hệ %s không mang được chiêu %q (hệ %s)",
	ErrorCurveShape:         "%q chưa đúng dạng; viết theo kiểu base:max",
	ErrorCurveNumber:        "%q có phần %s không phải là số",
	ErrorCurveNotPositive:   "%s bắt đầu ở %d; phải là số dương",
	ErrorCurveShrinks:       "%s kết thúc ở %d nhưng bắt đầu từ %d; chỉ số không tụt khi lên cấp",
	ErrorCurveRefused:       "đường chỉ số %s chưa hợp lệ: %v",
	ErrorStatField:          "%s: %s",
	ErrorFieldID:            "id này không dùng được: %v",
	ErrorFieldImage:         "đường dẫn ảnh không dùng được: %v",
	ErrorYear:               "năm %q không phải là số; để trống nếu không rõ",
	ErrorAsGiven:            "%v",

	ProblemMissingArt:     "nhân vật %s dùng ảnh %s, nhưng không thấy file ở %s",
	ProblemDoesNotResolve: "nhân vật %s không tính được chỉ số: %v",
	NoteWrote:             "đã ghi %s vào %s",
	NoteArtMissing:        "lưu ý: chưa có file %s; phần kiểm tra sẽ còn báo cho đến khi có",
	NoteRebuild:           "lưu ý: game chạy từ bản nhúng sẵn — build lại thì nhân vật này mới vào trận",
}
