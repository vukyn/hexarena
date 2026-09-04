---
name: hexarena-chooser-answer-routing
description: "hexarena — the chooser's bare drain ate a live answer and froze the match for a whole allowance (PR #287); answers are routed by turn now"
metadata: 
  node_type: memory
  type: project
  modified: 2026-09-04T11:35:10.103Z
---

`cmd/hexarena-tui`'s `session.choose` used to **drain** its one-slot answer channel on entry, on the premise that nothing could be in it for the turn now opening because the chooser had not yet sent `matchAskingMsg`. **That premise was wrong and it deadlocked the client.** Fixed in PR #287 (`087489f`… → `5f9f0a5`). See [[fixture-hidden-branch]], [[hexarena-pvp-lobby]].

**Why:** the screen does not learn whose turn it is from the chooser. It learns it from **`socket.Mirror.Asking`**, which is true the moment the room's batch is taken in — **one message and one redraw EARLIER** than `choose` is called. So a player answering off the board already in front of them lands in the slot *first*; the drain ate a real decision; `PlayScreen.Answered` meant the screen would not offer that turn again; and both ends stood still until the allowance ran out at the far end.

**How to apply:**

- An answer now carries the turn it was pressed for — `session.pressed{answer, unit, turn}`, a pair **beside** `draw.PlayAnswer` and not two more fields inside it, because that type is `battle.Chooser`'s return pair and its own doc refuses a second vocabulary for a decision.
- `choose(prompt *battle.Prompt)` **reads its prompt parameter** — it was unnamed and unread, and it was the whole answer. Slot matches this prompt → take it and return **without** `matchAskingMsg` (nobody to tell). Doesn't match → discard, which is the original stale-answer rule.
- `model.go` passes `m.battle.Pending` *after* the screen took the keystroke: `answering()` keeps `Pending` rather than clearing it, so that is the turn the decision was taken on.

**⚠️ Two general lessons, both now in `CLAUDE.md` § Mistakes already made here:**

1. **A window is not closed by the fact that you have not opened it.** When something buffered must be told apart from something stale, the discriminator is **the turn, never the moment**.
2. **A `select` over two ready arms is a coin flip — decide on a reading.** Same PR fixed `socket.Server.Shutdown`: `select { case <-settled; case <-ctx.Done() }` returned success or refusal *at random* whenever the last room ended near the bound, and the refusal it wrote read `0 room(s) and 0 connected room(s) still running` — a give-up naming nothing. Ask the thing (`rooms.Running()`), not the channel (a channel can be un-ready just because its goroutine hasn't been scheduled), and **hand the counts to the refusal as parameters** so the reading that decided and the reading reported cannot disagree.

**⚠️ A test that only reddens under load is a test the next person re-runs.** Both bugs failed inside `make check` and passed alone. Neither was a flake:
- the match test failed at **61.22s** against a 60s bound for work it does in 0.9s — one whole allowance, which is the fingerprint;
- the shutdown test failed in **0.00s** — its premise was wrong, not its timing. An already-done context proves the bound is *available*, not that anything is *waiting*. It now **holds a table open across the call** (a second `claim`, the way a connection does), so `Tables()` reads 1 on every path and the count is asserted by value.

**Reproduce the load condition:** 16 busy `while :; do :; done` shells + a temporarily shortened bound. Measured: without the shutdown wedge, red inside 60 runs at `GOMAXPROCS` 2 and 8, green at 1. Every end-to-end bound in this repo is a **hang detector**, not a performance budget — when one fires, what it was waiting on is the finding.
