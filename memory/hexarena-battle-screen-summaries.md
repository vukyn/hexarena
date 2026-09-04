---
name: hexarena-battle-screen-summaries
description: "hexarena PR #160 — a one-line derived summary beside every battle option plus ? for the full description; why it is a 4th describer, and the battle screen does not fit 80x24"
metadata: 
  node_type: memory
  type: project
  modified: 2026-08-29T18:00:44.702Z
---

PR **#160** (`51cfacd`, 2026-08-29). `cmd/hexforge-tui`'s battle screen listed the turn's options as bare skill ids. Now each row carries a **one-line derived summary** and `?` raises the full description of the option under the cursor.

```
> poison_powder  trúng độc 80% · tầm 2 · 2 ô · hồi 3
  venoshock      90% công · trúng độc→180% · tầm 1 · hồi 2
```

**`Lang.SummariseSkill` is a fourth describer, and the reason is the language.** `describeOpening` **fuses the authored flavour into the same sentence as the damage figure** in Vietnamese, so the flavour is not a separable clause there; English has none and separates cleanly — a rule that works in one language and not the other is not the rule. Price of a fourth reading: `TestTheOneLineSummaryQuotesNoFigureTheDescriptionDoesNot` holds every **digit run** it prints against `Describe`'s, every shipped skill × both languages. Wordings differ deliberately; figures may not.

**Zero rows** was the constraint that chose the shape: id column measured over the turn's own options (not the book), summary **clipped never wrapped** (a wrapped line pushes the footer off the bottom — what `frame` exists to prevent). An unavailable option keeps its `Reason` and drops its summary: one slot, and why it cannot be cast is the live question.

⚠️ **A cleanse cannot be summarised by enumerating what it strips.** The fixture's `purify` (3 categories) read **79 cells in Vietnamese** — the whole row budget before the aim and cooldown. So the summary **counts** where `Describe` **enumerates**, and says *harmful* only when every category the skill names is `Harmful()` (fixture `unmake` strips `buff`+`shield`, so the claim is not free). Harmful = `dot`, `stat_debuff`, `control`, `taunt`; benign = `buff`, `shield`, `regen`.

⚠️ **The battle screen does not fit its declared 80x24 minimum** — filed, not fixed. Body is **28 lines** against the **20** that survive (`frame` gives `height−2` minus 2 header rows); the option list first appears at **h ≥ 30**, un-truncated from **h ≥ 32**, and **h ≥ 38** once `recent` fills its 8 lines. `TestTheBattleScreenIsNoTallerThanItAlreadyWas` is a **tripwire at today's count, not a bound** — asserting it fits would ship red, the mistake [[hexarena-reckless-closed]] made once. Which rows to give up is its own layout item in `TODO.md`.

⚠️ **`Describe` prints raw category ids in English** (`Strips 1 stat_debuff and dot`) and it is **not** the lookup it looks like: `Lang.StatusCategory` is complete in both languages but its wordings are **predicates** for the statuses reference's column, so substituting gives *"Strips 1 stack of lowers a stat and damages every turn."* The sentence needs **noun phrases**, which English has none of — seven new keys, a family of its own. In `TODO.md`.

Both play footers were **over budget and unmeasured** (82/83 cells vs 79); now 77/78 with `?` added and no key dropped. That and two more blind branches are the transferable half — see [[fixture-hidden-branch]].

Related: [[hexarena-tui-references]], [[hexarena-tui-i18n]], [[hexarena-descriptions-are-derived]].
