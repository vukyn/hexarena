# Memory

Distilled, one-fact-per-file notes about this repository — the layer between a
commit message and `CLAUDE.md`. Each file holds **one** thing that was hard to
learn, why it matters, and how to apply it; the index below is one line per file
and nothing else.

This exists **in the repository** rather than in a machine's own Claude memory
directory because that directory is workspace-scoped and machine-local: opening
this repo on another machine, or outside the workspace it was written in, arrived
with none of it. Almost all of `.claude/` here is gitignored (it holds worktrees
and per-agent scratch), so the tracked home is this file plus `memory/`.
⚠️ The exception is `/.claude/agent-memory`, which the `.gitignore` un-ignores by
name and which therefore **does** travel — see `CLAUDE.md` for why that matters at
commit time.

**Rules, and they are the same rules the notes were written under:**

- **One line per note in the index: a link and a hook.** Detail lives in the
  file, never here — an index that grows prose stops being skimmable, which is
  the only thing an index is for.
- **One fact per file.** A note that has to say "and also" is two notes.
- **Say why, not just what.** A rule with no reason gets "simplified" away by the
  next reader; the ones marked ⚠️ are the traps that already cost a session.
- **A `[[link]]` with no file yet is fine** — it marks a note worth writing, not
  an error. Some links here point at notes that stayed behind in the workspace
  (they are about other repositories on the same platform) and that is deliberate.
- **Delete a note that turns out to be wrong** rather than adding a second note
  that contradicts it. Several of these say, in as many words, that an earlier
  version of themselves was wrong — that is the format working.

⚠️ **This is a distillation, not the record.** `CLAUDE.md` (the design record),
`TODO.md` (what is done, open and decided against) and `README.md` remain the
authority; where a note and one of those disagree, the file in the repository
that owns the subject wins and the note is the thing to fix.

## This repository

- [tách màn hình XONG (12 PR)](memory/hexarena-screen-extraction.md) — hexforge-tui 10.144→3.315; chặn là 2 closure trên model
- [3 golden thấy 3 thứ khác nhau](memory/goldens-see-different-screens.md) — ⚠️ cả 3 hình dạng mù đo được; golden≠transition
- [⚠️ golden không xanh được ở 2 platform](memory/goldens-cannot-be-green-on-two-platforms.md) — TRIỆU CHỨNG hết (accept lại trên máy `/`); NGUYÊN NHÂN còn: golden ghi filepath.Join xanh ở platform accept sau cùng
- [hexarena PvP plan](memory/hexarena-pvp-plan.md) — mirror client; bo1|bo3 KHÔNG bo2; 3 số version, digest là cửa
- [thế hoà = thứ tự roster](memory/hexarena-side-is-worth-60-points.md) — seq = slice order, CALLER quyết; +62% ở 1v1, +8.5pp ở 2v2
- [⚠️ mirror một chiều: ĐÃ VÁ](memory/one-way-mirror-not-a-measurement.md) — aims duyệt ô TUYỆT ĐỐI mà Place xoay 180° → hai nửa ngược thứ tự
- [data digest = cửa so BẰNG](memory/hexarena-data-digest.md) — peer-equality KHÔNG phải version; concat mù BIÊN DỜI + RENAME
- [consumer thứ hai cần cursor thứ hai](memory/hexarena-a-second-consumer-needs-a-second-cursor.md) — log PvP phải đọc từ 0; ⚠️ cursor phòng đặt SAU bàn mở → 291 ghi vs 300 re-run
- [record + cursor thay Drain](memory/hexarena-cursor-record.md) — append-only + cursor mỗi consumer; ⚠️ view KHÔNG cap = hỏng 2 chiều
- ["phe của tôi" có thể là phe người khác](memory/a-reading-called-my-side-can-be-somebody-elses.md) — ⚠️ Mirror.side của spectator LÀ phe host → mọi lượt host bị đọc là lượt mình; chặn ở MỘT chỗ dẫn xuất
- [ban/pick + spectator](memory/hexarena-draft-and-spectator-plan.md) — draft 1…5c xong; spectator 1…5 xong (còn cờ host + xem draft); ⚠️ hai cách gọi một bước, giữ bằng HAI vòng duyệt; wire ko nói phòng đủ người, cũng ko báo sắp xếp đã tới
- [v0.1.0 đã release](memory/hexarena-v0-1-0-released.md) — go install @v0.1.0 khai đúng tag; ⚠️ path không /vN nên CHỈ tag được v0/v1; tag trên proxy BẤT BIẾN
- [4 đội starter, dùng được ngay](memory/hexarena-starter-squads.md) — ⚠️ Rate() BỎ trận Endless khỏi mẫu số: healer gặp healer hoà mãi mà vẫn khoe 85%
- [đếm ngược + allowlist đồng hồ](memory/hexarena-countdown-clock-allowlist.md) — ⚠️ danh sách import ≠ danh sách đồng hồ (context.WithTimeout)
- [CRLF phá data digest](memory/hexarena-crlf-data-digest.md) — ⚠️ join Mac→Win data_mismatch cùng commit; .gitattributes ko chữa checkout cũ
- [dán chữ gãy cả 2 client](memory/hexarena-paste-both-clients.md) — ⚠️ textinput.pasteMsg KHÔNG XUẤT; PasteMsg là đường duy nhất
- [lobby PvP chơi được](memory/hexarena-pvp-lobby.md) — ⚠️ RWMutex qua callback deadlock với WRITER; Live dùng Since không Drain
- [chữ cho refusal/closure](memory/hexarena-protocol-wordings.md) — ⚠️ tên key trùng identifier module = MIỄN orphan test (33/577)
- [host binary + cổng 13579](memory/hexarena-host-binary.md) — ⚠️ cổng cố định làm test thứ tự RỖNG; picker TỪ CHỐI thay vì đoán
- [internal/wire = protocol](memory/hexarena-wire-protocol.md) — ⚠️ envelope thiếu kind decode thành `hello` (enum zero); `time` bị AST walk chặn
- [room = state machine](memory/hexarena-room-state-machine.md) — no I/O/goroutine/clock, timeout là INPUT; ⚠️ leaf ≡ Furthest(cap)
- [registry = nhiều room](memory/hexarena-room-registry.md) — 1 goroutine/room, request là VALUE; mutex giữ MAP thôi
- [code nới 1 byte](memory/hexarena-room-code-widened.md) — 12 ký tự/256 room; ⚠️ bit thừa base32 = 16 code ra 1 room; LOGIC race -race ko thấy
- [socket = transport + mirror](memory/hexarena-socket-transport.md) — coder/websocket; ⚠️ wire ko nói lượt ai → client ko mỏng hơn mirror
- [⚠️ answer bị drain ăn](memory/hexarena-chooser-answer-routing.md) — Mirror.Asking bật TRƯỚC chooser → drain trần nuốt đáp án, treo 1 allowance
- [hexarena core design](memory/hexarena-core-design.md) — ATB wait=1e6/speed; 6x3 odd-q + 180° mirror; saturate-not-clamp
- [range = rank depth](memory/hexarena-range-is-rank-depth.md) — ⚠️ `range` = OCCUPIED enemy ranks, NOT hex distance; empty rank free
- [block vs the rider](memory/hexarena-block-cancels-the-rider.md) — ⚠️ #181 blocked strike lands `dot` ONLY; miss lands nothing
- [fester / heal cut](memory/hexarena-fester-heal-cut.md) — #190 anti-sustain; reduce BEFORE cap; ⚠️ "ko placement nào đo" ĐÃ VÁ — s05 field nó, build ko tính
- [hexarena cast authoring + TUI](memory/hexarena-cast-authoring.md) — character=definition vs roster=placement; CLASS DROPPED
- [hexarena TUI i18n + glosses](memory/hexarena-tui-i18n.md) — vi default; data glosses may miss; label widths MEASURED
- [hexarena log gloss](memory/hexarena-log-gloss.md) — #171 glosses skill/status/trait; ⚠️ skillGloss 0/43 shipped, names live in JSON
- [hexarena log turn numbers](memory/hexarena-log-turn-numbers.md) — ⚠️ `A1 turn 5` then `E1 turn 4` is NOT a bug; Turn = unit's OWN count
- [bonus xếp đội (ĐÃ SHIP bậc 1)](memory/hexarena-composition-bonuses.md) — ⚠️ buff vĩnh viễn phải > quickened 80‰; cặp bão hoà định giá bằng KHÔNG
- [hexarena squad builder](memory/hexarena-squad-builder.md) — TUI dựng đội→squads.json; ĐỔI PHE mới là phép đo (mirror=500‰)
- [hexarena battle-screen budget](memory/hexarena-battle-screen-budget.md) — #162 can't fit 80x24; drop board whole; #169 follow=state not offset
- [hexarena battle-screen summaries](memory/hexarena-battle-screen-summaries.md) — #160 1-line derived summary + ?; screen needs h>=32
- [hexarena TUI references](memory/hexarena-tui-references.md) — statuses/traits/elements/species screens + ring-drawn affinity chart
- [hexarena skill name filter](memory/hexarena-skill-name-filter.md) — #176 `/` filter; fold table NOT x/text; cursor indexes FILTERED view
- [chỗ cho cast listing](memory/hexarena-cast-listing-room.md) — browseRoom = skillsRoom song sinh; ⚠️ dự trữ là HẰNG SỐ vì đo mỗi lần vẽ tốn 13ms và tăng theo cast
- [hexarena TUI width rule](memory/hexarena-tui-width-rule.md) — prose→minWidth, data→usableWidth(); floor 120; footers only floor can widen
- [hexarena shipping chars #182/#187/#189](memory/hexarena-poliwag-bruiser.md) — ⚠️ spar KHÔNG đo được support (dùng squad); hạ GIÁP mới kết mirror
- [hexarena shipping a character](memory/hexarena-shipping-a-character.md) — 5 json + cast_test.go design table (hardcoded!); effHP<=11500
- [hai số gặp nhau ở MỘT tích](memory/hexarena-two-numbers-that-meet-in-one-product.md) — ENG-004: Swung là hạng-1 nên grid bị TỪ CHỐI; ship `self_bonus` thay vì mặt hai trục
- [kit toàn chiêu diện ko có đòn kết](memory/hexarena-an-area-kit-has-no-finisher.md) — igglybuff 0‰ vs mewtwo 1000‰ CÙNG GHẾ; ⚠️ spar 1v1 định giá column bằng MỘT mục tiêu → đo ra hình ngược
- [pricing a new element pool](memory/hexarena-pricing-a-new-element-pool.md) — đo, đừng đoán; dpt = pw×hits×acc÷(cd+1); ⚠️ cần control cùng wrapper
- [kit toàn hệ đu biên độ rộng](memory/hexarena-elemental-kits-swing-wider.md) — nhân hệ tính TỪNG chiêu; kit pha neutral né được khắc chế
- [pool đầy ≠ hệ được chơi](memory/hexarena-a-full-pool-is-not-a-played-element.md) — 10 chiêu electric đều xoay quanh `charge` của Magnemite
- [dark nằm ngoài bảng khắc](memory/hexarena-dark-is-off-the-chart.md) — không cycle nào; profile phẳng; ⚠️ Gastly/Mewtwo là control TỆ
- [trait đổi giá có HAI chỗ đọc](memory/hexarena-a-trait-that-changes-a-price-has-two-sites.md) — thiếu ở rating → cầm trait mà không bao giờ cast
- [clause mới cần MỌI chỗ đánh giá](memory/hexarena-a-new-clause-needs-every-site-that-evaluates-it.md) — gate có 2 chỗ: options + Act
- [strict JSON dừng ở custom unmarshaller](memory/hexarena-strict-json-stops-at-a-custom-unmarshaller.md) — DisallowUnknownFields mù bên trong modifier.Modifier
- [golden không giữ được luật phím](memory/hexarena-a-golden-cannot-hold-a-keystroke-rule.md) — mọi fixture gán p.Aiming=true; vòng lặp `&& Aiming` dung thứ cả hai đáp án
- [baseline mutation lấy từ git](memory/hexarena-a-mutation-baseline-must-come-from-git.md) — copy file đang dirty làm backup → chạy lỗi thành edit vĩnh viễn
- [fixture chọn theo tính chất vẫn mù](memory/hexarena-a-fixture-chosen-by-property-is-blind-to-other-properties.md) — battleCast sắp theo SỐ chiêu → kit toàn `single`, tính năng diện tích vô hình
- [referent cũ sống lâu hơn kết luận nó chặn](memory/hexarena-a-stale-referent-outlives-the-conclusion-it-blocked.md) — squirtle 93.0%→28.4%; chính test đó đã ghi "re-take before quoting"
- [rate KHÔNG nói build có chơi hết kit](memory/hexarena-a-rate-cannot-say-a-build-played-its-kit.md) — ⚠️ RAT-006 đổ tội nhầm: split=0 CẢ ở kit thắng; đếm cast bằng Census, bàn duel/mirror mù ngược nhau
- [lượt bị chặn ≠ lượt mất](memory/hexarena-a-denied-turn-is-not-a-lost-turn.md) — hidden đếm cửa sổ bằng lượt CHỦ + lấy cú mạnh nhất; sửa được 4× trên 30× cần
- [rating định giá 0 thì ngừng chơi — ĐÃ VÁ](memory/hexarena-a-rating-that-prices-nothing-stops-playing.md) — credit CHỈ cho guard permanent; ⚠️ credit phẳng làm 40/40 mirror ship Endless
- [nhánh summon bỏ qua giá](memory/hexarena-the-summon-branch-skips-the-price.md) — continue nên không tới prices.rate; ⚠️ probe trước khi kết luận
- [counter có hạn ≠ trần cả trận](memory/hexarena-a-timed-counter-is-not-a-lifetime-cap.md) — phải permanent + category không ai strip
- [hai cửa sổ không phủ hết list](memory/hexarena-two-windows-do-not-cover-a-growing-list.md) — vỡ 3 lần; duyệt hết con trỏ
- [mutation bị parser chặn = không chạy](memory/hexarena-a-mutation-the-parser-refuses-runs-nothing.md) — đổi ý nghĩa, đừng xoá field
- [lấy mẫu qua Chooser](memory/hexarena-sample-through-the-chooser.md) — event.Stacks là phần THÊM, không phải tổng đang giữ
- [ghim assertion, đừng ghim tiền đề](memory/hexarena-pin-the-assertion-not-the-premise.md) — 1 con số quyết định chỉ ghim ở ĐÚNG 1 test
- [cast.json phải đúng form tool](memory/hexarena-cast-json-must-be-in-tool-form.md) — Marshal sort theo id; chỉ cast+origins bị ràng
- [hexarena archetype must be glossed](memory/hexarena-archetype-must-be-glossed.md) — lối chơi NEVER bare id
- [hexarena descriptions derived](memory/hexarena-descriptions-are-derived.md) — 3 describers; only `flavour` authored; tables exact
- [hexarena flavour voice](memory/hexarena-flavour-voice.md) — write the fiction not the engine; never promise a mechanic
- [hexarena flavour sweep](memory/hexarena-flavour-sweep-todo.md) — #111 DONE 29/41 chiêu + 4 bio; bio digit/later-form ban
- [hexarena status naming](memory/hexarena-status-naming.md) — debuffs=verbs, buffs=nouns; status ≠ who receives it
- [hexarena mechanics log](memory/hexarena-mechanics-log.md) — MERGED per-PR log #43→#138: draw, roster, gates, learnsets, summons, builds
- [hexarena reckless gap](memory/hexarena-reckless-gap.md) — dragon build 22% vì reckless trả 2 stat mua 1; detonate KHÔNG phải nguyên nhân
- [hexarena builds catalogue](memory/hexarena-builds-catalogue.md) — builds.json=DATA + tên build; DoT tick unattributed
- [hexarena stat bounds policy](memory/hexarena-stat-bounds-policy.md) — ceilings+11500 bound AUTHORED, saturation bounds FOUGHT; đừng nâng ceiling
- [hexarena speed + measurement](memory/hexarena-speed-and-measurement.md) — ⚠️ WIN RATE cannot price speed (non-monotone); band TURN SHARE in ONE battle
- [hexarena reckless closed](memory/hexarena-reckless-closed.md) — #155/#156 cả 3 lever chết; ⚠️ stat BÃO HOÀ (−400‰ nền 400 → 290)
- [hexarena bout + waiting](memory/hexarena-bout-and-waiting.md) — #144 forge.Bout control 500‰ CHÍNH XÁC; deeper opponent ĐÓNG
- [hexarena deeper opponent](memory/hexarena-deeper-opponent.md) — #117 Suggest định giá non-damage; 3 clamp LÀ design; permanent duration=0
- [hexarena crit mechanic](memory/hexarena-crit-mechanic.md) — #135+#148 razor_leaf/wind_shuriken 200‰; ⚠️ THEME ≠ PRICE, kunai crit ra ÂM
- [roster cannot price damage](memory/hexarena-roster-cannot-price-damage.md) — ⚠️ ally damage ↑ → win rate ↓; dùng `hexforge weigh` + `--carriers all`
- [hexarena tempo + stalemate](memory/hexarena-tempo-and-stalemate.md) — #121 frozen() chỉ DoT giữ bàn; một lượt = turnWorth chứ ko bestStrike
- [hexarena roster placement](memory/hexarena-roster-placement.md) — #136 placement THUẦN PHÒNG THỦ 27.6→47.3%; level ko phải dial
- [Lỗ định giá của Suggest](memory/hexarena-rating-gaps.md) — audit XONG 11/11 category; ⚠️ bàn đo mù → 2 lần kết luận sai
- [reserve = charge phía mình](memory/hexarena-reserve-counter.md) — ⚠️ chia đôi SAI với reserve (456 dồn/0 tiêu); clamp vào SỐ STACK
- [3 trục mới: grant/convert/cost máu](memory/hexarena-three-new-axes.md) — ⚠️ chi phí trong nhánh ko chạy; đếm mà ko assert = ko đo gì
- [absorb = giáp ảo](memory/hexarena-absorb-guard.md) — pool trừ dần vs charge huỷ trọn nhát; checklist 9 bước
- [Gengar + dispel](memory/hexarena-dispel-and-gengar.md) — nhánh rating CHẾT; ⚠️ biên độ lấy từ MUTATION không phải số đo
- [Mew + Mewtwo](memory/hexarena-mew-and-mewtwo.md) — hệ trơ + dark; ⚠️ status 1 lượt ko làm mồi được; tốc độ là đồng tiền
- [hexarena all-sided + scarcity](memory/hexarena-allsided-and-scarcity.md) — #127 all-sided priced BOTH halves; đo AI đối xứng phải HEAD-TO-HEAD
- [đổi 2 số cùng lúc phải đo bằng spar](memory/hexarena-two-field-rebalance.md) — weigh chỉ đo 1 field vs control; copy data dir + spar overall
- [redraw không được đọc battle của mirror](memory/hexarena-a-redraw-may-not-read-the-mirrors-battle.md) — luật viết cho Attach, mất ở View; ⚠️ phải ÉP chồng lấn mới thấy; đọc theo reading RẺ hơn theo draw (130 vs 202)
- [guard suite mù với field nó không set](memory/a-guard-suite-is-blind-to-a-field-it-never-sets.md) — cả 5 test readonly XANH với `Authoring: len(m.player) > 0`; fixture chỉ chạy MỘT cấu hình
- [mã đội phải duy nhất vì raise gọi theo mã](memory/a-squad-id-must-be-unique-because-a-raise-names-one.md) — landSquad lấy khớp ĐẦU TIÊN → trỏ dòng 2 mà ra sân dòng 1; nối 2 danh sách đo được lỗi này

## General — engineering and workflow lessons that apply here

- [heading không phải là luật](memory/a-heading-is-not-a-rule.md) — CLAUDE.md 161KB→70KB; ⚠️ đo bytes TỪNG MỤC, cắt theo câu ràng buộc
- [gọi việc theo MÃ, không theo dòng](memory/todo-items-are-addressed-by-code.md) — TODO.md dùng AREA-NNN cố định; ⚠️ vùng theo CHỦ ĐỀ chứ không theo file bản vá rơi vào
- [Commits always via PR](memory/commits-always-via-pr.md) — never direct push to main
- [Stage explicit paths](memory/stage-explicit-paths-parallel-sessions.md) — parallel sessions on same repo; never `git add -A`
- [Verify committer staged files](memory/verify-committer-staged-files.md) — committer misreported 2×; verify show --stat + branch + log
- [git checkout discards to HEAD](memory/git-checkout-discards-to-head.md) — never revert a mutation with checkout; edit it back
- [Pin base to worktree HEAD](memory/pin-comparison-base-to-worktree-head.md) — before/after proof: archive HEAD, not origin/main
- [Mutate the producer](memory/mutate-the-producer-not-just-the-logic.md) — grep every write site; compile-failing mutations prove nothing
- [Comment style generic](memory/comment-style-generic.md) — comments high-level, no specific usecase detail
- [Test only what changed](memory/test-only-what-changed.md) — `./...` is minutes of CPU; ⚠️ `signal: killed` is a starved machine, not a red test
- [kill test ports after smoke](memory/kill-test-ports-after-smoke.md) — go-run child outlives parent → `lsof -ti :PORT|xargs kill -9`
- [gopls stale diagnostics multirepo](memory/gopls-stale-diagnostics-multirepo.md) — false BrokenImport; trust `go build`/`go vet`
- [Fixture hides a branch](memory/fixture-hidden-branch.md) — 5× in hexarena; test must fail when its own branch unexercised
- [Gate fixture must be crossable](memory/a-gate-fixture-must-be-crossable-by-the-thing-under-test.md) — gate 500 vs raise 200‰ never fires; passed with the code deleted
- [Blanket refusal hides its legs](memory/a-blanket-refusal-hides-where-its-legs-live.md) — narrowing one guard; the gated-trait leg lived in another book
- [Cast count ≠ squad count](memory/a-cast-count-is-not-a-squad-count.md) — saved squad may double a character; 1 carrier still reaches rung 2
- [Book walk misses departures](memory/a-shipped-book-walk-catches-arrivals-not-departures.md) — shipped⊆table says nothing about an orphan table entry
- [Assert on the clause, not the page](memory/assert-on-the-clause-not-the-whole-page.md) — Contains(page,x) matched the header, not the verdict
- [bỏ một default đã ship](memory/turning-a-shipped-default-into-an-opt-in.md) — 17 test cũ phải ĐỎ, không phải im; ⚠️ ban đã nới một lần sẽ tả sai phần còn lại
- [⚠️ test tự hoàn thành ko thấy đồng hồ tắt](memory/a-test-that-finishes-itself-cannot-see-a-stopped-clock.md) — allowance.set tắt clock ghế đang hỏi; treo, ship v0.1.0, suite xanh
- [⚠️ dữ liệu thật có thể thoả sẵn](memory/real-data-can-satisfy-the-property.md) — cast.json vốn sắp theo id → test thứ tự pass cả khi hàm sort
- [bubbletea v2 silent breaks](memory/bubbletea-v2-silent-breaks.md) — charm.land/…/v2; space="space"; NO_COLOR now yours
- [Windows sets no $TERM](memory/windows-sets-no-term.md) — `TERM==""` is dumb only off Windows; hand GOOS in as a param
- [Ambiguous-width glyphs in TUI](memory/terminal-ambiguous-width-glyphs.md) — ⌘⇧→⇄ measure 1 cell, draw 2; keep labels ASCII
- [Go tool toolchain floor](memory/govulncheck-toolchain-rebuild.md) — govulncheck+staticcheck built with Go >= repo directive
- [VN diacritic search](memory/vn-diacritic-search.md) — kuery text.FoldVN + denormalized *_search col; đ→d
- [gitleaks silence ≠ safety](memory/gitleaks-silence-is-not-safety.md) — default rules miss PG/Neon URIs + 1-line PEM
- [Test credential literals rule](memory/test-credential-literals-rule.md) — name fixture constants, never inline
- [GitGuardian = dashboard](memory/gitguardian-dashboard-not-repo-file.md) — .gitguardian.yaml chỉ cho ggshield; exclude ở dashboard TỪNG repo
