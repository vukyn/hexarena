---
name: hexarena-log-gloss
description: "hexarena PR #171 — the battle log glosses every skill/status/trait id it prints; why Lang.Gloss could not do it (skillGloss is fixture-only now) and why the gloss map is caller-supplied"
metadata:
  type: project
---

PR **#171** (`13a6db0`, 2026-08-31). The battle log printed data ids bare, so in Vietnamese it said nothing. Every **skill, status and trait** id now carries its Vietnamese name in angle brackets (`razor_leaf <phi diệp>` — see the nesting note below), **at every occurrence** — a log is read in pieces and scrolls since #169, so "first mention only" is a row the reader may never have on screen.

⚠️⚠️ **`Lang.Gloss(id)` glosses NOTHING in a real battle, and the table looks fine.** Measured: skill **43 shipped / 0 in `skillGloss` / 43 with an authored `name`**; status 21/21/0; passive 11/0/11. `skillGloss`'s 19 ids (`strike`, `ember_lance`, `creeping_rot`, `purify`, `unmake`…) intersect `internal/seed/data/skills.json` in **zero** places — they are the pre-`name`-field skills and today only `internal/testfixture` reaches them. Shipped skill and passive names live in the JSON, reachable only through `Lang.SkillName` / `PassiveName`. **Do not delete `skillGloss`** (fixtures), and **do not trust CLAUDE.md's old claim** that the table answers for "all nineteen shipped skills" — corrected in #171.

So this is [[fixture-hidden-branch]] with the fixture *ahead* of the shipped data: a test written on fixture skills passes through the id table and never exercises the path all 43 shipped skills take. The decisive mutation is building the map off the id table instead of the authored names — **every status line still reads correctly**, and only the shipped-skill assertions redden.

**The gloss map is caller-supplied**, not a `Lang` and not books: `Line(event, tags, glosses)` / `Log(events, tags, glosses)`, `map[string]string` of id → name, exactly as `tags` already is and `Summary`'s `names` already is. `internal/tui`'s package doc is the reason — *"the event lines are built from the event alone, never from the battle"* — which is what lets a replay draw a log handed to it as a file. **A nil map reproduces the line byte for byte**: that is what English is, what a bookless replay is, and the property the goldens rest on. `Lang.LogGlosses(skills, kinds, passives)` builds one and answers **nil** for English (the nil is the answer, not an absence). Bracket keeps one definition — `i18n.GlossBracket` over the existing `glossBracket`.

⚠️ **A colliding id is left OUT of the map, never picked between**, and the decision is read off the **id** rather than off which kinds offered a name — so it cannot go live the day somebody authors the second name. Zero collisions today; `taunt` is a shipped skill id and the status is `taunting`, a near miss. This is PR #161's lesson made structural (that guard looked *within* a table, not *across* them).

⚠️ **Glossing costs cells and one arm could not pay.** `amplified` is the only arm printing two glossed ids in one clause: `dragon_drive (long xung) is amplified by expose (phá giáp) x2, power 2300` = **82** of 79 — a row that fit at 59 before and would not after, on the screen [[hexarena-battle-screen-budget]] proved has no spare rows. The arm gave up the word `is` (3 cells) — the one register that can spare it, since this log is verb-terse everywhere else (`hits`, `misses`, `holds`, `lets go of`). Widest reachable row is now **79 of 79**, and nought margin is deliberate: the bound is recomputed from the books every run over *reachable* pairings (a skill beside **its own** condition's status, a trait beside the statuses it actually names), so a longer name authored later goes red rather than the log wrapping. `knownWide` is kept but **empty** as the shape for a breach somebody later accepts.

⚠️ **An over-wide log row WRAPS, it does not clip.** The option rows clip (`MaxWidth`); the log rows carry none. So a breach costs a row of a budget the screen has not got, rather than losing the digits at the end — a different failure from the one #160 designed the clip for.

⚠️ **The `Started` line is the widest row in the log in BOTH languages at 1/3/5 a side (81/87 vs 79) and always was.** Nothing on it is glossed — not the side, not the element note. Its 86–87 is a **fixture** figure (long names, dual affinities); the shipped roster's is 73. Left alone deliberately.

**How to prove "English unchanged" cheaply:** render all `battle.KindCount` kinds with a nil map on `main` and on the branch, diff the two. One line differed, the one intended. Faster and stronger than reading the diff.

⚠️ **The gloss is `<name>`, not `(name)`, and PR #172 (`8b8f8d8`) is why.** `status_applied` names the trait a status came from as a parenthetical of its own, so the round gloss inside it read `(virulence (độc lực))` — a bracket inside a bracket. The gloss is the **inner** thing wherever the two meet, so the gloss changed shape, and it changed **everywhere** because `glossBracket` is its one definition. `<x>` and `(x)` are both two cells, so **nothing measured moved** (the 79-of-79 `amplified` bound included) and **no golden held a gloss** so none moved either — the whole change was one const plus five hardcoded literals in two test files.

⚠️ **The nesting guard counts bracket DEPTH, it does not look for `((`** — the nesting it exists to catch does not contain `((`, the two brackets sat five words apart. And it fails when **no arm draws a gloss inside a parenthetical any more**: a sweep finding no nesting because the shape disappeared measures nothing, and would go quiet the day somebody reworded the one arm it was written for. Self-clearing in both directions.

Related: [[hexarena-tui-i18n]], [[hexarena-descriptions-are-derived]], [[fixture-hidden-branch]], [[hexarena-battle-screen-budget]].
