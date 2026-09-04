---
name: hexarena-skill-name-filter
description: "hexarena PR #176 — typed name filter on the skills listing: the `/` mode idiom, the diacritic fold table (explicit, NOT x/text, đ is not a combining mark), and the cursor-indexes-the-filtered-view trap with its fourth site outside the file"
metadata:
  type: project
---

PR **#176** (`88b0fba`, 2026-08-31). `cmd/hexforge-tui`'s skills listing gained a **typed name filter**: `/` opens it, typing narrows live, `enter` keeps the query and hands the listing back its keys, `esc` clears and closes in one stroke. Arrows move the cursor while the field has focus — **every letter on that screen is already a command** (`q a e j k`), which is the whole reason a typed filter needs a mode.

**Two kinds of filter now live in this client, and they are not the same idiom.** The pre-existing ones (`browseScreen.filter` by origin, `pickState.filter` by group) are **categorical, an `int` index cycled by a key, where 0 means "hides nothing"**. This one is typed text. An empty query hides nothing, which is the same value 0 carries. `i18n.Matches(query, fields...)` is generic and a picker could adopt it — **deliberately not wired to any picker**; extending typed filtering there is its own decision.

⚠️⚠️ **The bug the feature exists to introduce: the cursor indexes the FILTERED view while a read indexes the FULL slice, so `e` edits the wrong skill.** Everything funnels through `skillsScreen.rows()` (the one filter) and `selected() (skill.Skill, bool)` (the one indexed read) — the shape `pickState.visible` already had. **The fourth read site is outside the file the feature lives in:** `cmd/hexforge-tui/blurb.go` is the other half of `?` and walked the unfiltered slice for both the row *and* the `1 / 66` position beside the name. When adding a filter, grep for reads of the backing slice across the *package*, not the screen's file.

⚠️ **A filter test passes vacuously unless the first match is not index 0.** The guard: query `"long"` matches five skills **by their Vietnamese names only** (`long nộ`, `long vũ`, `long trảo`, `cuồng nộ long`, `long xung`) and its first hit is the 17th skill declared, and the test asserts `rows()[0].ID != skills[0].ID` **before** anything else.

**The diacritic fold is an explicit table in `internal/i18n/fold.go`, not `golang.org/x/text`.** ⚠️ `đ` is **not a combining mark**, so NFD-plus-strip-`Mn` needs a hand-written entry anyway — a table reaches the same answer without promoting an indirect dependency to direct. Table is 7 ASCII bases × 67 accented lower-case letters; the upper-case half is **derived** with `unicode.ToUpper` on both sides so one case cannot go missing from the other. Matching is on **id OR Vietnamese name**, case- and diacritic-insensitive: typing `diep` finds `phi diệp`, which is what makes the name half usable on a terminal with no Vietnamese IME.

⚠️ **This lookup is NOT allowed to miss, unlike the gloss tables** ([[hexarena-log-gloss]] — those render a bare id on a miss by design; a fold that misses silently stops matching). So three tests, and the third is the one that survives the data moving: the 67 letters enumerated longhand **with a `counted != 67` check** so a lost row reddens rather than going unchecked; a letter with no fold returned **unchanged** (`z Z 7 _ · ↑ ß ж` — it is a lookup, not a filter, or a footer would come apart); and a sweep **driven from the shipped data** — every wording in both catalogs plus every shipped skill, status and trait name, **3347 accented letters across 1154 strings** — which fails naming `foldGroups` and fatals on nought.

**The match reads `Vi.SkillName` in BOTH languages.** `ctrl+l` works from every screen and keeps what is typed, so a query silently finding fewer rows after a language toggle would be the screen mutating behind the author. Stated cost: an English reader can be handed a row whose id does not hold what they typed.

Footer: naming `/` cost 7 cells (vi) / 11 (en) that were not there, so the **words** after `↑/↓`, `esc` and `q` are dropped — `BrowseFooter`'s own squeeze, for keys whose meaning the screen shows. **No key given up.** 65/74 of 79; filter footer 63/70. The filter row is **reserved in every state** (`skillsRoom` = `height - 4 - 10`), so pressing `/` narrows the list without also shifting every row under it up by one.

Three `everyScreen` entries registered — field open, a query matching several, a query matching none — and `everyScreen` itself now fatals if `/` stops opening the field or the queries stop discriminating. Per [[hexarena-tui-i18n]], a state not registered there has no width, translation or leak sweep.

Related: [[hexarena-tui-i18n]], [[hexarena-tui-references]], [[vn-diacritic-search]], [[fixture-hidden-branch]].
