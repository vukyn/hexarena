---
name: fixture-hidden-branch
description: An early return that every test fixture takes makes everything below it dead code to the suite — demonstrated three times in hexarena within one week; the fix is a test that fails when its own branch went unexercised
metadata: 
  node_type: memory
  type: reference
  modified: 2026-08-29T18:00:24.806Z
---

**An early return that every fixture takes makes everything below it dead code to the suite.** Cross-compiling, `make check`, coverage and a golden all pass; the branch simply never runs. Seven instances in hexarena inside one week, every one found by measuring the screen or the output by hand rather than by a failing test:

1. **`plainTerminal`** (PR #158). `startIn` sets `NO_COLOR` for **every** test in `cmd/hexforge-tui` — deliberately, so assertions match a word rather than a word wrapped in escape codes — and `NO_COLOR` is asked *first* and returns. The `TERM` branch below it was unreachable, so *no native Windows terminal drew any colour* and nothing said so. See [[windows-sets-no-term]].
2. **`everyScreen`'s battle models** (PR #160). All three (`"a battle"`, `"aiming"`, `"a battle over"`) drew the **game-over** footer, because `fight.enter(screenPlay)` yields a battle for which `Finished()` is already true and `view` returns above the footer, the option rows and the aim block. `aiming.play.aiming = true` was set and never read. So `PlayFooter` shipped at **82 cells against a 79-cell window** and the width test rendered it every run without seeing it.
3. **The clip in the same PR.** The fixture's four options are at most 44 cells against 65 of room, so nothing clips from `atTheBattle` — a clip mutated to `minWidth-2` **passed the entire suite**.
4. **`l.join` between every pair** (PR #163). Three items read `a and b and c`, in both languages, for as long as the function existed. **Every list the shipped books produce has one item or two**, so `describe.golden` held zero three-item lists and holds zero after the fix — a golden could never have caught it. Reachable only through the fixture's `purify` (three stripped categories), which is the **same fixture skill** that exposed the strips-clause width in #160.
5. **A key alias** (PR #170). `[`/`]` alias `pgup`/`pgdown`, and an alias that never fires is **indistinguishable from one that does** in every test that does not press it. Worse, the natural test for it — press both, demand the same landing — is satisfied by a `key` helper that sends `KeyPgUp` under **both** names: **the vacuity lives in the fixture, where an assertion cannot see it.** The fix is to read the helper's own table (hoisted to package level for exactly that) and measure what the name sends, then assert the reference key **moved** the frame before comparing. And renaming a test's key constants onto the new alias silently drops the *old* key's coverage — mutate the old key, not the new one.
6. **A gloss table the shipped data left behind** (PR #171), the inverse of every case above — the **fixture** is ahead of the shipped data, not behind it. `skillGloss` holds 19 skill ids and `skills.json` holds 43; they intersect in **zero** places, because the shipped skills moved to an authored `name` field and only `internal/testfixture` still reaches the table. So a log gloss built on `Lang.Gloss` glosses nothing in a real battle, every fixture-based test passes, and CLAUDE.md's own claim that the table answers for "all nineteen shipped skills" had been false for some time. ⚠️ **The decisive mutation leaves most of the output correct:** building the map off the id table reddens only the shipped-skill assertions — all 21 status lines still read perfectly, because statuses *are* covered 21/21. Ask of a lookup table not "is it complete" but **"what fraction of the shipped ids does it reach"**, and take the extreme case from `seed.Books()`, never from the fixture.
7. **The battle screen's own height** (PR #162), the inverse shape: `frame` cut from the bottom and the option list was the last thing written, so at 80x24 the player's own moves were the first thing discarded — and the tripwire that existed asserted a *ceiling on rows*, which a golden and a width sweep both pass.

**The fix is not "add a case".** A table that happens to cover one branch today reverts to vacuous the next time the data moves. Make the test **fail when its own branch went unexercised**:

- count what each branch actually reached and assert both are non-zero, with a message naming the wording/behaviour that went unmeasured (`TestAStripIsCalledHarmfulOnlyWhenEveryCategoryItNamesIs`);
- build the extreme case rather than hoping the fixture is extreme — look the widest/longest input up, never name it (`TestAClippedRowKeepsTheLongestPrefixItHasRoomFor` pairs the widest summary with the widest id);
- when a language/platform legitimately cannot reach it, `t.Skip` **with the measurement**, and have the *parent* fail if **no** subtest reached it — self-clearing in both directions;
- hand a platform-varying rule its inputs as parameters (`plainScreen(noColour, term, goos)`); `runtime.GOOS` cannot be faked.

⚠️ **A golden records text, not width** — so a row that is clipped, truncated or overflowing looks identical in it. Width, height and line-count need their own tripwires, measured over what the **library** holds (fixtures included), not the shipped book: hexarena's `purify` is a fixture skill, which is exactly how a 79-cell cleanse clause hid from a golden covering all 43 shipped skills.

⚠️ `t.Skip` inside a `range` over cases is **`runtime.Goexit`** — it abandons the whole test, not the iteration, so every later case goes unmeasured in silence. Use subtests.

⚠️ **One fixture entity can be load-bearing for several latent branches.** hexarena's `purify` — a *fixture* cleanse naming three categories, which no shipped skill does — is the only reader of both the strips-width guard and the list-comma guard. When a fixture is the sole route to a branch, say so in the test's comment: the day somebody trims the fixture for being unrealistic, two guards go quiet at once.

Related: [[windows-sets-no-term]], [[mutate-the-producer-not-just-the-logic]], [[hexarena-battle-screen-summaries]], [[hexarena-battle-screen-budget]].
