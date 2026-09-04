---
name: hexarena-bout-and-waiting
description: "hexarena #144: forge.Bout fights two ratings head to head with an EXACT 500‰ control; waiting was retired as arithmetically empty, not built"
metadata: 
  node_type: memory
  type: project
  modified: 2026-08-28T19:37:42.805Z
---

**How to measure a rating change: `forge.Bout` (#144, 2026-08-29).** `RunToEnd` used to call `b.Suggest` for both sides so two ratings could never meet; it now takes a `battle.Chooser`. `Bout` runs challenger-drives-ally then swapped, same seeds for both.

⚠️ **The control is EXACT, not statistical.** With the same rating on both sides the two arrangements are *the same battle* — same seed, roster and decisions — so it has one winner and the challenger takes exactly one of the two: **500‰ by construction**, and `Bout` refuses to print a figure otherwise. Consequences: **the board need not be symmetric** (the shipped roster's asymmetry cancels exactly, so measure on the real board — `CLAUDE.md`'s mirror-roster ban is about *data* changes and does not apply here); and **making the control fail is hard on purpose** — any Brain that is a function of *position* cancels exactly, even "surrender when playing the enemy"; breaking it needs state advancing *between* arrangements.

`forge.FirstUsable` is the **frozen ruler** (what `Suggest` was before pricing). It may never be improved — a moving ruler makes every past figure incomparable. Standing reading: **`Suggest` beats `FirstUsable` 780‰ over 10,000 seeds, band ±8‰, 45 median turns against the control's 48.** Quote future AI changes against this.

**Waiting is retired, not deferred — it is arithmetically empty.** `spendCooldowns` (`turn.go:307`) takes a turn off *every* cooldown and runs at the end of `Act` (`:541`), `Pass` (`:323`) and a stunned turn (`:92`) alike, while `Act` starts only the cast skill's own (`:542`). So act = `bestValue` + next turn's best, wait = `0` + next turn's best: the awaited skill comes off cooldown either way, and #131 already declines anything below nought. Two further bars: a simulating lookahead needs a `*Battle` that cannot be cloned and a resolution that either **rolls** or **restates the resolving arithmetic** — one broken rule each — and costs ≈×36 a turn's rating. Guarded by `TestAPassBuysNoCooldownAnActDoesNot` and `TestNothingWaitsOnPurpose`.

**The deeper-opponent item is CLOSED.** Both remaining halves are in *Decided against*. Waiting, above. And the **queue tie-break (#149): built, measured, thrown away.** A third key under `cooldown` (take the aim whose occupant acts soonest) moved **0 of 93,320 decisions**; new-vs-old read 500‰ as **10,000 W / 10,000 L** — the control signature, i.e. the identical battle every time — and no golden moved. ⚠️ The premise was wrong, not the code: one skill over several cells does share a cooldown, but a tie needs the **values** level too, and shipped units differ in health/defence/affinity so two aims almost never rate to the same integer (census: **0** tied prompts). Don't rebuild without first showing the tie exists on the board in hand. Two lessons kept: **absence must sit beside a queue reading, never inside it** (`Pending` returns 0 for an unknown id and 0 is *soonest*; a self-aim must be *declared* unread, not detected), and **`Suggest` may not call `Standings`** — it sorts the real slice, invisible to `describeBoard` but changing every later turn order. Reply pricing shipped in #147 (a wash on shipped data — only `ally.venusaur` answers at all).

⚠️ **The ruler figure is a reading of a BOARD, not of the rating.** It was 779‰ before the first crit chances and **813‰** after (#148), with nothing about the rating changed. Re-take it after any data change instead of quoting the last one.

Related: [[hexarena-roster-cannot-price-damage]], [[hexarena-deeper-opponent]], [[hexarena-crit-mechanic]].
