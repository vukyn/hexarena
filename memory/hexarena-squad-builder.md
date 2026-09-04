---
name: hexarena-squad-builder
description: hexarena's TUI squad builder writes squads.json (the author's own data, not the game's); a squad carries no side and Take() prefixes ids
metadata:
  type: project
---

**PR #142 (2026-08-29): `hexforge-tui` gained a squad builder** — menu entry *đội hình* / *squads*. First screen that writes the **author's own** data; every other file the client edits is the game's.

- **`internal/core/placement`** (new, pure): `Placement` = character, level, form, cell, 4 skills, 1 trait — the same six facts a roster entry carries. `Squad` = named list, **max 5**. `Parse`/`Marshal`/`Validate`/`Take`/`Clone`.
- ⚠️ **A squad carries NO side.** `Squad.Take(side, cast)` fields it as either half and **prefixes unit ids with the side** — that is the only thing that makes a squad-vs-copy-of-itself readable in a log.
- **Shape vs loadout are checked at different moments on purpose**: `Validate` (ids/count/slots) at read, the loadout only at `Take`. A squad being built is half-finished, and a file that refused to hold one couldn't be saved and returned to.
- `squads.json` embedded in seed + loaded optionally by forge (like `builds.json`); **ships empty**. `Library.SaveSquad` **replaces** same-id (opposite of `SaveSkill`, deliberate: a working document's edit loop *is* saving again) and validates through `Take`.
- UI rules that matter: changing the character **empties the kit**; the form chooser offers *furthest* + every form by name (needed by [[hexarena-mechanics-log]]'s forking lines); the slot chooser **steps over** occupied cells; the 3×3 grid is ASCII only.

⚠️ **One loadout rule, which had become two.** "Which four of nine may this unit bring" existed as `seed.chooseFrom` AND `cast.chosenFor` (both unexported, worded differently); the builder needed a third. Now **`cast.ChooseLoadout` / `cast.ChooseFrom`**, all three call it, subject is a worded noun phrase (`unit "x"` / `the build "x"`). This is what lets the builder show the refusal *as the kit is chosen* and still match the write.

**PR #143: the fight shipped.** `f` on a squad → `forge.Library.FightSquads(home, away, seeds)`; `←/→` opponent, `+/-` battles (10…1000). Reuses spar's `fight`/`Tally`/`Result`/`median`.

⚠️ **BOTH WAYS ROUND IS THE MEASUREMENT.** Roster order decides the turn-queue tie-break, so one arrangement reports the first slot's advantage as the squad's (a mirror read **58.8%** measured that way once). Both halves run the **same** seeds. Consequence: **a squad vs a copy of itself reads exactly 500‰** — the control, and the first thing to break if the swap stops cancelling.

The halves are reported **apart** too: their difference is what *standing on a side* is worth — **18 points** on the fixture pairing.

⚠️ A squad rate is **NOT** the roster's win rate; the screen says so in prose, wrapped against **minWidth** (not the live window) or the width sweep has nothing to hold.

**PR #145: play it yourself.** `p` on the fight screen → one battle, you vs `battle.Suggest`. `↑/↓` skill (unavailable ones shown with the reason, cursor steps over), `enter` takes it and asks *where* **only when >1 cell**, `a` hands the turn to the engine, `p` pass, `u` undo, `n` next seed.

- Draws with **`internal/tui`** (`Board`/`Roster`/`Order`/`Line`) — the game client's own drawing, never a second one.
- **Undo = shorter script replayed**, not an unwinding (battle = pure fn of seed + decisions). So **the engine's turns are recorded in the script too** — a half not written down replays as a different battle.
- ⚠️ **Only screen holding something the model does NOT copy** (`*battle.Battle` is a pointer). Step the battle in `update`, **never** in `view`, or a redraw plays a turn.
- `take`/`skip` are two methods, not one taking a `Decision`, so "skill with no aim" is unrepresentable (this also removed 2 prose error strings the AST test flags).

**PR #146: `ctrl+s` writes the battle out** — `Library.SaveBattleLog` → `<data>/battles/<home>-vs-<away>-seed<n>.json`; pairing+seed identify a battle so a re-save overwrites. Saveable mid-battle (a half-played battle replays as that half). `.gitignore`s `internal/seed/data/battles/`.

⚠️ **`--verify` re-runs against the EMBEDDED copy, not the edited dir** → a log written after an unbuilt edit will not verify, and the mismatch is the edit. That is `NoteBattleVerify` (its own note kind, because "rebuild first" is part of the instruction on a log).

⚠️ `replaceFile` now MkdirAlls the target folder and puts the temp file **in it** (rename across folders isn't atomic). The "failed write leaves the old file" test had to switch from a *missing* folder to one under a **file**.

⚠️ File names come from author-typed squad ids → `forge.fileToken` sanitises (a squad named `../../escaped` still lands in `battles/`). Rule kept in forge, not in `placement.Squad`: what a file name may hold ≠ what an id may be.

⚠️ Found + fixed: `WarningShortReach` had **4 `%` verbs against 3 args** since #133 → printed `%!d(MISSING)`. `TestTheSameBlanksInEveryLanguage` can't catch that class — it compares the two languages with **each other**, never with the call site.

TUI conventions worth remembering: no goldens cover `cmd/hexforge-tui` (assertion tests only); every new screen **and each of its states** must be registered in `everyScreen` (language_test.go) or the width/both-languages sweeps skip it; i18n = 3 edits (keys/english/vietnamese) and an orphan key fails the suite; label widths **measured**, never constant.

See [[hexarena-tui-i18n]], [[hexarena-cast-authoring]], [[hexarena-mechanics-log]].

**PR #150: `?` reads a row while choosing it** — on the kit and trait pickers, `?` shows the row's derived description (`Describe` / `DescribePassive`), `?`/`esc` returns with the chosen set intact, `↑/↓` walks to the next description, `pgdn/pgup` scrolls.

⚠️ It is a **state on `pickState`, not a screen** — forced: the picker is drawn over whichever screen raised it and `model.key` routes to `m.picker` **before** any screen, so switching `m.screen` would leave the picker still eating the keys. Offered only on the 2 kinds with a describer; statuses deliberately excluded (their rows already print the facts, and the keys under that list belong to its chance field).

⚠️ **Two shipped bugs it uncovered, both from the fixture cast being empty:**
- `openSquadPassives` was raised with `kind: pickSkills` while holding **trait** ids → `detail` looked each up in the skill book → every row drew `unknown skill "venom_blood"` **in red**. Uncaught because the **testfixture cast learns no traits**, so every test that opened that picker opened an *empty* one. Fixed with a `pickPassives` kind; the regression test finds the trait-holder in the book rather than naming one (fixture rule: look it up, don't name it).
- Both squad pickers inherited the **form's** hint, which names the `!` mark. `squadOptions` sets **no refusal at all** (the options are already the character's learnset), so no row there can ever carry one — and on the trait list the vi hint said "chiêu" and `TraitSlots = 1` made "số là thứ tự" order a list of one. Now `SquadKitHint` / `SquadTraitHint`; the trait one says the slot **may be left empty** (`cast.Optional`), which nothing had said.

⚠️ **The test that keeps a hint true is not the one that reads it.** Beside asserting each picker draws its own wording, assert **no option on those lists holds a refusal** — a sentence can be made true by editing it; only the data behind it keeps it true.

Measured, and it cut a premise: a picker describes **one** row, and the longest shipped description is **3 lines** in either language across every skill and trait — so the scroll is a guard, not a live path (`blurbScreen` scrolls because it draws all 5 traits at once, ~30 lines against 17).

**PR #153: the formation picture draws live.** `←/→` stepped `s.unit.Slot` while the 3x3 was drawn from `s.editing.Units`; the two met only at `commit()` (run on leaving the member or opening a picker), so the mark **stayed on the old cell for the whole of the choosing** and jumped once you left. Fixed by `unitsDrawn(editing)` — the committed members with the one under edit **substituted** (not appended, so the cell being left empties in the same draw), into a **fresh slice** because `s.editing.Units` is shared with every model copied off the screen.

⚠️ **Do not fix this class by committing on a keypress.** `commit()` also latches `unsaved`, and the discard guard hangs off it — a cursor passing over a cell would leave a squad claiming changes nobody made. The drawing reads and writes nothing.

⚠️ **A liveness test must assert on the RENDER.** `s.unit.Slot` changing was already true on the broken build; the test that discriminates finds the mark's (line, column) in `m.screenContent()`. Mutation: `unitsDrawn` → `return s.editing.Units` must fail it.

Front rank: nothing was backwards — `hex.Place` rotates an enemy formation 180°, so authoring **col 2 is depth 0 on both halves**, and the old caption/ordering were right. What was missing is the *picture* never said so. Now `^^^` + "hàng này chạm địch trước" under the front column, and the column order is read off `hex.Ranks`/`hex.Place` rather than counted down from `FormationCols` (right today only by coincidence of one side's numbering). Slot row reads both: `< 1,0  hàng giữa >` — coordinate ties the screen to `squads.json`, rank is the half a coordinate cannot say.

**PR #154 paid that debt: the unsaved guard is a COMPARISON, not a flag.** It was `unsaved bool`, latched from **8** places and cleared from 4; `commit()` writes a member back on the way out whether or not a key moved anything, so opening a member and pressing `esc` claimed a change. Now `dirty() = !s.editing.Equal(s.baseline)`, baseline **cloned** at `begin`/`open`/`save` (an alias compares equal to itself forever). Discard restores `editing` from the baseline.

⚠️ **`placement.Squad.Equal`/`Placement.Equal` live in core beside `Clone`** — a second copy of that field list fails **silently**: a missed field compares equal, the question is never asked, the edit is thrown away with nothing on screen looking wrong. Order-sensitive (kit order *is* the kit).

⚠️ **Test it in two layers.** Keystroke table covers what a person can press; a **reflection walk** (`TestSquadEqualityReadsEveryField`, nudging each field in turn + a kit reorder) covers *every* field — `Placement.ID` is derived from the character and unreachable from the keyboard, so a screen-only table can never claim completeness. Mutation: drop `p.Slot` from `Equal` → both layers go red.

⚠️ Fixture trap: `withASquadSaved` wrote through `lib.SaveSquad` instead of the screen's `save()`, so the baseline stayed empty and 4 unrelated tests hit the discard prompt. A fixture that says "this is now on the file" must move the baseline too.

Known, left: `open()` loads id/name **untrimmed** while the edit branch writes `TrimSpace` back, so a hand-edited `squads.json` with padded id/name reads as changed on the first keystroke. Harmless — the trim would be saved anyway, and a guard erring toward *asking* is the safe direction.

## PR #157 — picker chặn tại slot, đếm theo slot (merged d806322)

Giới hạn loadout (`cast.SkillSlots=4`, `TraitSlots=1`) trước đây chỉ hỏi SAU khi enter
(`squadScreen.refuse` → `cast.ChooseFrom`), nên chọn 6 chiêu mới biết sai ở dòng lỗi
dưới form. Giờ `pickState.slots` (0 = không giới hạn — mọi picker khác giữ nguyên);
`toggle()` từ chối **sau** nhánh gỡ ra, vì loadout sửa tay quá 4 trong `squads.json`
đúng là trạng thái tác giả cần sửa được. **Chặn chứ không swap**, kể cả ở 1 ô: một
phím hai nghĩa sẽ đọc khác nhau trên hai picker cùng màn hình. Từ chối im lặng được
vì heading đổi sang `ChoiceSlots` (`"%d of %d slots"` / `"%d/%d ô"`), style `bad` khi
đã quá slot. `refuse`/`save` giữ nguyên — luật lúc ghi vẫn bắt data không qua picker.

⚠️ **"Vẫn gỡ được khi đầy" KHÔNG phân biệt việc gỡ mất cap** — không có cap thì gỡ
vẫn chạy, test vẫn xanh. Mutation thật của nó là **đẩy cap lên trước nhánh gỡ**. Bài
chung: một test về THỨ TỰ NHÁNH phải mutation bằng cách đảo thứ tự, không phải bằng
cách xoá nhánh.

⚠️ Chuỗi slot **chứa** chuỗi position (`"4 of 4 slots"` ⊃ `"4 of 4"`), nên test
"có ChoiceSlots, không có ChoicePosition" chỉ phân biệt được khi hai toán hạng thứ
hai khác nhau (13 option vs 4 slot) — test tự `t.Fatal` ở khâu dựng nếu picker liệt
kê đúng `SkillSlots` dòng.

Bẫy fixture lần 3: cast fixture học đúng 4 chiêu → không có dòng thứ 5 để thử. Helper
`aDeepLearner` **dò** nhân vật học nhiều nhất, cùng luật `aTraitHolder` — trong repo
này helper luôn TÌM chứ không GỌI TÊN.

## PR #159 — cửa menu vào trận + squads.json CÓ ship (merged 82253df)

Người chơi dựng xong đội không tìm ra chỗ đánh: menu 11 mục toàn soạn/tra cứu, cửa
duy nhất là `f` trên catalogue, còn `p` (tự đánh) chỉ hiện ở footer màn Fight — chưa
tới đó thì không bao giờ thấy. Fight không có mục menu vì **không tự có chủ thể**:
`sides()` đọc phe nhà từ `m.squad.cursor`, con trỏ của màn KHÁC. Giờ `fightScreen`
mang `home` cạnh `away` (↑/↓ phe nhà, ←/→ đối thủ); catalogue `f` **gieo** `home` từ
con trỏ → đường cũ y nguyên. Lý lẽ "trận cần chủ thể trước" tan, cửa menu thành thật.

⚠️ **`internal/seed/data/squads.json` CÓ SHIP** (user chốt). `seed.Squads` doc đã tự
nói: go:embed → đội dựng trong tool tới được trận ở lần build kế, cùng luật mọi data
file. `TestTheShippedCatalogueIsEmptyAndParses` khẳng định file rỗng → **đổ đúng ngày
có người dùng công cụ**; đã bỏ mệnh đề rỗng, giữ mệnh đề parse. Câu "ships empty" nằm
rải **7 chỗ** (roster.go, library.go, model.go, 2 test, README, CLAUDE.md) — sửa hết,
và giữ vết ⚠️ câu cũ trong CLAUDE.md để người sau đọc bản cũ chỗ khác không revert.
TUI fixture (`scratchData`) phải **loại squads.json khỏi bản copy**: fixture tự làm
chủ catalogue của nó, không đo đội tác giả vừa lưu.

⚠️ Trạng thái rỗng trước đây mượn nhầm chữ: Fight rỗng vẽ `SquadsEmpty` "bấm **n**"
— phím màn Fight KHÔNG có. Bài chung: chuỗi mượn từ màn khác mang theo phím của màn
khác. Giờ là `FightNoSquads`.

⚠️ **Brief tôi sai: `m.squad.saved` KHÔNG rỗng lúc khởi động** — `newModel` gọi
`refresh(lib)` lúc dựng, hai chỗ ghi duy nhất đều tự refresh. Dòng refresh thêm vào
`enter(screenFight)` **không test nào phân biệt** (mutation xoá nó suite vẫn xanh);
giữ lại nhưng comment nói thẳng nó không phải lá chắn cho trạng thái từng thấy.

Mutation học được: cho `sides()` đọc lại `m.squad.cursor` thì test tương thích VẪN
XANH — vì con trỏ đang trỏ đúng đội đó. Test tương thích chỉ đổ khi bỏ dòng gieo.
`esc` từ Fight vẫn về catalogue (user chọn để vậy).
