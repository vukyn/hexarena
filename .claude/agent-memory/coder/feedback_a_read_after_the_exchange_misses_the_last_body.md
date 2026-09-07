---
name: a-read-after-the-exchange-misses-the-last-body
description: In hexarena a room retires itself the moment its match ends, so a consumer that reads Registry.Since AFTER the exchange loses the final wire.Turn — 62 of 63, deterministically; the record read has to ride home on the answer to the input that recorded it
metadata:
  type: feedback
---

`internal/room`'s registry goroutine answers a request and then, if
`playing.Finished()`, **returns and retires its own entry**. The exchange that
records a match's last `wire.Turn` is the same exchange that finishes the room —
so a consumer that answers its players and *then* calls `Registry.Since(code,
cursor)` is asking a room that has already gone.

**Why:** it is not a race you can win. Measured with a throwaway probe (a whole
match through the registry, a `Since` after every `Deliver`): **62 of 63 turns,
`known=false` on the last read, 3 runs out of 3.** The room needs a few
instructions to retire; the transport needs a socket write first. The spec I was
handed said "after each exchange, the server reads `Registry.Since`" and that
design cannot see the killing blow of any match.

**How to apply:**

- **Make the answer carry it.** `room.Answer.Watched`/`Cursor` are now filled on
  *every* input, not on a `Since` alone — one line in `answerFrom`, taken inside
  the room's own goroutine before anything can retire. `Since` stays for the one
  read that has no input to ride on: a watcher's catch-up at cursor **0**.
- **The payoff is bigger than the bug.** With bodies pushed on the answer, the
  transport hands a room **no cursor but nought**, so `Room.Since`'s deliberate
  panic on an out-of-range cursor (on the room's goroutine — it takes the process
  down) is unreachable from there, and no range guard had to be added anywhere.
  A consumer's own cursor becomes a *check* against `Answer.Cursor`, and a
  consumer out of step is ended rather than caught up quietly.
- **Anything else that reads that record inherits this**: the log writer, a
  reconnect, a spectator of a draft. Ask "what produced the last body, and does
  that thing still exist when I ask?" before writing a read loop.
- **The probe was worth its five minutes.** A 60-line `zz_probe_test.go` that
  printed `steps / collected / lastReadKnown` settled a design argument I would
  otherwise have reasoned about wrongly in either direction. Write it, read it,
  delete it.

Related: [[a-state-the-reading-can-never-hold]],
[[three-guards-a-neighbour-already-covered]].
