---
name: hexarena-a-redraw-may-not-read-the-mirrors-battle
description: "PlayScreen.View drew straight off the mirror's *battle.Battle with no lock — a real data race in shipped PvP code; the fix is a reading taken in Attach, where the lock is already held. ⚠️ The invariant was WRITTEN DOWN for Attach and lost for View. ⚠️ Sampling cannot see it (whole package -race clean, the one test alone clean at -count=10) — the overlap has to be FORCED. ⚠️ Moving the render to Attach is CHEAPER, not dearer: 130 readings against 202 draws in one battle."
metadata:
  node_type: memory
  type: project
---

A live `PlayScreen` holds the **mirror's** `*battle.Battle`, and `View` →
`drawings` read straight off it on the bubbletea goroutine while `Client.Play`
replayed turns into it on its own. `tui.Board`, `tui.Roster`, `tui.Order`,
`Fight.Finished`, `Outcome`/`Winner`, `Fight.Unit` and `Fight.Units` — all of it
with no lock at all. `make check` caught it once; it is not a flake, the access
is unsynchronised by construction and the load only decides whether the two
goroutines overlap.

**Why it survived: the rule was known for one caller and lost for the other.**
`socket.Sight`'s doc says a reading is "valid only for the duration of the call
it is handed to". `Attach`'s own comment on `Since` says *"a pure read, so this
is safe under the read lock the caller is holding"* — the author knew exactly
where the lock was. `View` is the same screen, the same pointer, one goroutine
away, and nothing said so there. **A comment that states an invariant at the
site that obeys it does not protect the site that does not.**

**How to apply.**

- **The fix is a reading, not a lock.** `readBattle(fight, tags) playReading` is
  taken in `Attach` — which already runs inside `socket.Mirror.Read`, under the
  lock — and the draw reads that. It carries the three sections **already
  rendered**, because rendering them *is* the read: a value carrying the units to
  be walked later moves the race rather than removing it. This is
  `socket.DraftSight`'s decision (a snapshot, never the pointer) arriving one
  screen later; `Sight.Fight` was named there as the one deliberate exception.
- ⚠️ **Local play is untouched and must stay untouched.** The same screen *owns*
  its battle in a hot-seat game and drives it (`Act`, `Pass`, `Advance`, `Drain`,
  `Suggest`, `Replay`), single-goroutine. So `read()` is three lines — the stored
  reading when `Live`, a fresh one off `p.Fight` otherwise — and the local one is
  **lazy** on purpose: a stored local reading would need refreshing at every site
  that steps the battle, and a site missed there draws a stale board with every
  test green.
- ⚠️ **Sampling cannot reproduce it.** The whole `cmd/hexarena-tui` package under
  `-race` was clean on main (0 races), and the one test that had failed was clean
  alone at `-race -count=10`. The reproducer **forces** the overlap: one goroutine
  redraws a copy of the model in a tight loop while the match applies turns —
  255 distinct `WARNING: DATA RACE` blocks on the first run, in 2.1s. A test that
  waits for a race is a test the next person re-runs.
- ⚠️ **Two vacuity nets, not one.** A run that drew nothing, or drew while nobody
  was stepping the battle, proves nothing and now fails: the test counts its own
  redraws *and* the turns the mirror applied during them (~317 against ~93).
- ⚠️ **A second, silent way to get this wrong**, which the race test cannot see:
  taking the reading only when the battle **pointer** changes. The board is then
  the opening board for the whole match while the event log underneath stays
  correct — no golden moves, no test reddens. `TestALiveRedrawDrawsTheBattleAsItStandsNow`
  asserts the drawn roster **is** `tui.Roster(fight, tags)` *now*, rather than
  that it changed.
- ⚠️ **The cost went DOWN, against the obvious expectation.** The three sections
  used to be rendered per **draw** and are now rendered per **reading** — and
  measured over one 3v3 bo1: **203 updates (= 203 views), 202 drawings, 130
  attaches, 28.8µs a reading**. Every message that attaches also draws, while
  every keystroke draws without attaching, so readings are structurally the
  smaller number: 3.7ms a battle against 5.8ms. Do not assume "per reading" is
  dearer than "per draw" in a client whose redraws outnumber its messages.

No golden moved, which is the answer to the question that mattered: a draw that
now reads a snapshot produces byte-identical screens.

Related: [[hexarena-socket-transport]] [[hexarena-pvp-lobby]]
[[hexarena-draft-and-spectator-plan]] [[fixture-hidden-branch]]
