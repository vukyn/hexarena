---
name: hexarena-flavour-voice
description: "hexarena — how to write authored prose (skill/trait flavour, species note): verb must match the action, write the fiction not the engine, RPG colour welcome but never promise a mechanic that does not exist"
metadata: 
  node_type: memory
  type: feedback
  modified: 2026-08-28T10:30:48.636Z
---

The user's standing rule for seeding data (2026-08-26 PR #72, 2026-08-27 PR #77): a `flavour` clause must **fit the context of the thing it names and read plainly**. Prose quality is part of the deliverable, not decoration — the user reviews the wording line by line and has sent back batches of corrections twice.

**Why:** the clause is the only authored sentence in a description; everything around it is derived. So it is the only place the writing can be *wrong about the fiction* rather than wrong about a number — and a clause that reads as nonsense on somebody's screen costs more than the few words rewording it would have cost. It also has to sit flush against the derived half, which appends figures to it in the same sentence.

**How to apply — checklist before shipping a clause:**

1. **The verb must be the action that physically happens.** `búng` is a flick of a finger — wrong for leaves and for a fireball; `bắn` (shoot) is right for both. `rắc` (sprinkle, of seasoning) → `rải` (scatter) for a powder. `đâm` (stab) → `cắm` (plant, of a root). `cuộn` (roll up) → `tạo` for making a vortex.
2. **No cute or quirky verb for a mechanical effect.** The user rejected `tự vá` (patch yourself up) twice — for `synthesis` and `aqua_ring` — as *nghe vô duyên*. Use the plain word: `tự hồi`. Same for `tự nhận` → `nhận`, `lấy chỗ đứng` → `tại chỗ`.
3. **Never restate what the derived half already says.** `fire_fang`'s first clause opened *Ngoạm hai nhát* and the derived half then printed *2 nhát*: the same fact twice, one of which stops being true when the strike count moves. This is what `TestAFlavourClauseSpellsOutNoNumber` now catches, but write it right first.
4. **Do not describe an action the carrier may not be able to do.** `flamethrower`'s *há miệng* was cut. This is the softer, judgement half of the hard body-word ban — see [[hexarena-descriptions-are-derived]] for the ban itself and for why a **trait's** version of it is unconditional.
5. **Use the domain's own word.** `giống` → `loài` for a species. When in doubt grep a sibling entry in the same JSON and match its register.
6. ⚠️ **Drop "nó" / "của nó" wherever the meaning survives** (2026-08-27, PR #91). A description is about **one** unit, so the pronoun carries nothing and only lengthens the line — this holds for the derived wordings too, not just flavour: `Nó gây %s mạnh thêm %s` → `Hiệu quả %s mạnh hơn %s`, `Mọi đòn của nó hút lại` → `Mọi đòn hút lại`. Where a sentence really has **two** subjects, name them rather than using two pronouns: a reply is *"ai đánh trúng thì bị phản lại … công **của người bị đánh**"*.
7. **Prefer the word the rest of the data already uses.** `bồi lại sức` → `hồi lại sức`, because `hồi` is what every other healing line says. Grep the sibling JSON before inventing a synonym.
8. **Keep it one clause, present tense, no figures.** The derived half appends numbers to this sentence, so a clause that ends on its own full thought reads as two sentences jammed together.
9. ⚠️ **Write the fiction, not the engine — RPG/fantasy colour is wanted** (2026-08-28). The authored prose exists to make the game *varied and interesting*; it must not read as a restatement of the rule it sits above. A note explaining a carry rule (*"nên chất rồng không riêng một kẻ mang được"*, *"thứ mang được cùng lúc với dòng máu rồng"*) is engine commentary wearing prose clothes — the screen already prints the derived lines that say it. Say what the thing **is in the world** instead: what it looks like, what it does, what it feels like to meet. Room for imagery, sensory detail and a second clause where the field allows it (a species `note` has no digit ban and no derived twin, so it has the most room; a skill `flavour` still owes rules 3 and 8). **This does not reopen rule 2:** what the user rejected there was *undignified* (`tự vá`, `lấy chỗ đứng` — *vô duyên*), not *fantastical*. High register is welcome; silly is not.

   ⚠️ **But it must not promise what the engine cannot deliver** (the user's immediate follow-up). Colour is free; a *mechanic* is not. Do not write imagery a reader would reasonably cash in — "cái mai chặn mọi đòn" when a turtle has no defence bonus, "rễ hút sinh lực kẻ đứng cạnh" when nothing drains, "đợi cho qua" when nothing waits out a status. This is rule 4 widened from *the body* to *the rules*: rule 4 bans an action the carrier may not be able to perform, and this bans an outcome the engine does not compute. Test: for every effect the sentence implies, name the field that implements it — no field, reword.

   Engine rationale that gets displaced by this has to survive somewhere — check for an existing home before deleting it (plant's species gate is recorded in `internal/seed/skills_test.go`), and say so if the only remaining copy would be git history.

⚠️ **`shadow_clone` compares its copies to the caster ON PURPOSE** (PR #120, the author's own wording): *"…yếu hơn bản thể gốc nhưng tấn công được"*, sitting above the derived *"mỗi bên mang 40% chỉ số người gọi"*. That is the duplication `casterWords` / `TestASummonsFlavourClaimsNothingAboutItsCaster` exists to refuse, and it passes only because the banned strings are `bản gốc` / `người gọi` rather than `bản thể gốc`. It was raised with the author and they kept it. **Do not widen the banned list to catch it, and do not "fix" the clause.** It is a decision, not a leak.

**2026-08-28 batch, second pass** (PR #120). The author re-read the first batch and sent nine more. What was still wrong: imagery that needs a beat to decode — `chỉ qua một nhịp là mầm đã bén` · `trả lại bằng một cột lửa trắng` · `tiếng dội đi trước rồi sức mới tới` · `đúng lúc đòn tới`. Plus: drop a simile once the noun is strong enough (`như vừa mở cửa lò`, `như roi`), prefer the bigger word for a big action (`thấp và nặng`→`vang trời`), name the thing fully (`một điệu`→`một điệu nhảy`), and cut a softening `cũng`. **The through-line across both passes: a clause must land on the first read.**

**2026-08-28 batch** (PR #115, on the #111 sweep). Rejected as **tối nghĩa / không hiểu**: `cả một trời nắng` · `không khí nứt ra thành từng lằn` · `nuốt cả tiếng động` · `đất dưới chân nảy theo` · `đủ đặc để chắn đường và đủ sức để ném` · `và tắt là hết một đời` · `lửa lụi` · `bao nhiêu đời` · `đánh ai cũng một nhịp như nhau`. Rejected as **the wrong word for the thing**: `khét lẹt` (a *burnt* smell, above a mud attack) · a `chậu nước` nobody has in a battle · `rung rinh` (too gentle for a wave of heat). Plainer word wanted: thả ra→phóng ra · từng thớ→cơ thể · Phun→Tạo · đặc→dày · theo một nhịp→theo nhịp · dòng nước→tấn nước · đợt đòn→đợt tấn công · gọi đòn→thu hút sát thương · đứng lâu hơn→trụ lâu hơn · trì nặng→trụ vững · thuỷ ba→sóng nước. **The lesson under the batch: colour is wanted, colour nobody can picture is not.** A clause has to name something a reader can see.

**Corrections the user has actually sent** (keep this list; it is the calibration): búng→bắn · rắc→rải · đâm→cắm · cuộn→tạo · giống→loài · bồi→hồi · `tự vá`/`tự nhận`/`lấy chỗ đứng` rejected as *vô duyên* · `há miệng` cut · `Miễn hoàn toàn`→`miễn nhiễm` · `càng bốc dữ`→`càng dữ dội` · `càng trỗi lên dữ`→`càng trỗi dậy` · `nuôi mình` rejected · an opaque clause (*vết thương nó xé ra lại quay về nuôi chính nó*) rejected outright as **tối nghĩa**.

⚠️ **A trait's clause has a different job from a skill's.** A skill's clause replaces the derived opening and says *what the attack looks like*. A trait's is a **lead line** above the derived lines and says *what the creature is* — `venom_blood` → *Máu chảy trong người vốn là nọc; ai cắn phải thì tự chuốc lấy*. (It was once justified as giving the derived lines' bare `nó` an antecedent; PR #91 removed those pronouns instead, so the clause now stands on saying what the trait *is* and nothing else.)

⚠️ **English gets no clause.** It is authored once, in Vietnamese, exactly as `Name` is; an English reader falls back to the derived opening. Do not add a translations table.

Golden to check the result against: `internal/i18n/testdata/describe.golden` — read the whole entry, not the clause alone, because the clause and the appended figures have to read as one sentence.

Related: [[hexarena-descriptions-are-derived]], [[hexarena-shipping-a-character]], [[comment-style-generic]].
