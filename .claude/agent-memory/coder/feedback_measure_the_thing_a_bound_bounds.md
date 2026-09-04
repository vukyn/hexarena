---
name: measure-the-thing-a-bound-bounds
description: A limit or threshold written from reasoning about a size is usually wrong by orders of magnitude in hexarena; encode the measurement in a test that fails at BOTH ends, and expect a -race run to catch test-order assumptions rather than product races
metadata:
  type: feedback
---

Two lessons from writing `internal/socket`, both of which cost a correction
after the code already looked finished.

## A bound written from reasoning about a size, not from the size

I set a WebSocket read limit to **1 MiB**, with a confident comment: a 5v5
`wire.Start` carries the whole resolved roster, so it would approach the
library's 32 KiB default. Then I wrote the test. The largest start a legal room
can send is **2,911 bytes** — the library's own default would have done, and the
megabyte was 360× more allocation than a peer should be able to ask for. The
comment was more wrong than the number.

**Why:** this is the same shape as [[i18n-key-gofmt-churn]] ("measure the column
before naming the key") and [[measure-which-guard-masks]] ("the blamed clamp was
innocent"). The repo's own record is full of it — `DefaultMessageLimit`,
`swiftness`'s +150, the roster win rate. Reasoning about a magnitude reads as
knowledge and is a guess.

**How to apply:** before writing any limit, threshold, cap or timeout, construct
the largest/slowest thing it bounds and print the figure. Then put the figure in
the constant's doc comment *and* make the test hold **both ends** — a limit under
what the system produces is a broken feature, a limit orders of magnitude over it
is the defect I actually shipped, and a one-sided assertion (`largest <= limit`)
passes for both. A threshold with two bounds and a stated reason for each
(`DefaultCloseThreshold`: generous against a wifi hiccup, under the turn
allowance) is defensible; one with a round number is not.

## `-race` in this repo catches test assumptions, not just product races

Adding `internal/socket` to `make check`'s `-race` line cost ~1.4s and its first
catch was **my end-to-end test**, not the transport: it asserted the server had
released its per-room table the instant the *client's* `Play` returned. A client
returning means that client is done; the server's own connection goroutine tears
down a moment later. Without the detector the two interleaved the harmless way on
every run.

**Why:** the same reason `internal/room` is in that line — the detector is the
only thing that reorders these. But the *finding* was a wrong claim in a test
rather than a race in the code, which is worth expecting: a concurrent
end-to-end test's assertions about "and now nothing is left" are the first
things it will redden.

**How to apply:** when a test asserts a teardown/cleanup state right after
waiting on one side of a connection, either poll with a bound or find something
to wait on. And if there is nothing to wait on, that absence is a real API gap
worth filing (a `socket.Server` has no shutdown; `http.Server.Shutdown` does not
wait for hijacked connections) rather than a reason to drop the assertion.
