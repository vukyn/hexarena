# Memory

Distilled, one-fact-per-file notes about this repository — the layer between a
commit message and `CLAUDE.md`. Each file holds **one** thing that was hard to
learn, why it matters, and how to apply it; the index below is one line per file
and nothing else.

This exists **in the repository** rather than in a machine's own Claude memory
directory because that directory is workspace-scoped and machine-local: opening
this repo on another machine, or outside the workspace it was written in, arrived
with none of it. `.claude/` here is gitignored (it holds worktrees and per-agent
scratch), so the tracked home is this file plus `memory/`.

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
- [hexarena PvP plan](memory/hexarena-pvp-plan.md) — mirror client; bo1|bo3 KHÔNG bo2; 3 số version, digest là cửa
- [thế hoà = thứ tự roster](memory/hexarena-side-is-worth-60-points.md) — seq = slice order, CALLER quyết; +62% ở 1v1, +8.5pp ở 2v2
- [⚠️ mirror một chiều ≠ phép đo](memory/one-way-mirror-not-a-measurement.md) — bù nhau CHỈ đúng ở 1 unit/phe; phụ thuộc KIT
- [data digest = cửa so BẰNG](memory/hexarena-data-digest.md) — peer-equality KHÔNG phải version; concat mù BIÊN DỜI + RENAME
- [record + cursor thay Drain](memory/hexarena-cursor-record.md) — append-only + cursor mỗi consumer; ⚠️ view KHÔNG cap = hỏng 2 chiều
- [ban/pick + spectator (CHƯA LÀM)](memory/hexarena-draft-and-spectator-plan.md) — ban 2/bên 3v3 vừa; 5v5 ⚠️ thiếu nhân vật; spectator đổi ai thắng
- [v0.1.0 đã release](memory/hexarena-v0-1-0-released.md) — go install @v0.1.0 khai đúng tag; ⚠️ path không /vN nên CHỈ tag được v0/v1; tag trên proxy BẤT BIẾN
- [3 đội starter](memory/hexarena-starter-squads.md) — ⚠️ 3 bản cùng nhân vật là đội YẾU NHẤT (~11%), ngược TODO
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
- [fester / heal cut](memory/hexarena-fester-heal-cut.md) — #190 anti-sustain; reduce BEFORE cap; ⚠️ no shipped placement measures it
- [hexarena cast authoring + TUI](memory/hexarena-cast-authoring.md) — character=definition vs roster=placement; CLASS DROPPED
- [hexarena TUI i18n + glosses](memory/hexarena-tui-i18n.md) — vi default; data glosses may miss; label widths MEASURED
- [hexarena log gloss](memory/hexarena-log-gloss.md) — #171 glosses skill/status/trait; ⚠️ skillGloss 0/43 shipped, names live in JSON
- [hexarena log turn numbers](memory/hexarena-log-turn-numbers.md) — ⚠️ `A1 turn 5` then `E1 turn 4` is NOT a bug; Turn = unit's OWN count
- [bonus xếp đội (CHƯA LÀM)](memory/hexarena-composition-bonuses.md) — 7 quyết định chốt; ⚠️ origin miễn phí 14/15; đếm 1 lần vào trận
- [hexarena squad builder](memory/hexarena-squad-builder.md) — TUI dựng đội→squads.json; ĐỔI PHE mới là phép đo (mirror=500‰)
- [hexarena battle-screen budget](memory/hexarena-battle-screen-budget.md) — #162 can't fit 80x24; drop board whole; #169 follow=state not offset
- [hexarena battle-screen summaries](memory/hexarena-battle-screen-summaries.md) — #160 1-line derived summary + ?; screen needs h>=32
- [hexarena TUI references](memory/hexarena-tui-references.md) — statuses/traits/elements/species screens + ring-drawn affinity chart
- [hexarena skill name filter](memory/hexarena-skill-name-filter.md) — #176 `/` filter; fold table NOT x/text; cursor indexes FILTERED view
- [hexarena TUI width rule](memory/hexarena-tui-width-rule.md) — prose→minWidth, data→usableWidth(); floor 120; footers only floor can widen
- [hexarena shipping chars #182/#187/#189](memory/hexarena-poliwag-bruiser.md) — ⚠️ spar KHÔNG đo được support (dùng squad); hạ GIÁP mới kết mirror
- [hexarena shipping a character](memory/hexarena-shipping-a-character.md) — 5 json + cast_test.go design table (hardcoded!); effHP<=11500
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

## General — engineering and workflow lessons that apply here

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
- [bubbletea v2 silent breaks](memory/bubbletea-v2-silent-breaks.md) — charm.land/…/v2; space="space"; NO_COLOR now yours
- [Windows sets no $TERM](memory/windows-sets-no-term.md) — `TERM==""` is dumb only off Windows; hand GOOS in as a param
- [Ambiguous-width glyphs in TUI](memory/terminal-ambiguous-width-glyphs.md) — ⌘⇧→⇄ measure 1 cell, draw 2; keep labels ASCII
- [Go tool toolchain floor](memory/govulncheck-toolchain-rebuild.md) — govulncheck+staticcheck built with Go >= repo directive
- [VN diacritic search](memory/vn-diacritic-search.md) — kuery text.FoldVN + denormalized *_search col; đ→d
- [gitleaks silence ≠ safety](memory/gitleaks-silence-is-not-safety.md) — default rules miss PG/Neon URIs + 1-line PEM
- [Test credential literals rule](memory/test-credential-literals-rule.md) — name fixture constants, never inline
- [GitGuardian = dashboard](memory/gitguardian-dashboard-not-repo-file.md) — .gitguardian.yaml chỉ cho ggshield; exclude ở dashboard TỪNG repo
