---
name: fixture-golden-cannot-see-producer
description: hexarena's messages.golden is built from hand-written fixtures, so it pins the wire FORMAT and can never see the room failing to fill a field in — measured; a new message field needs a golden entry AND an assertion in the producer
metadata:
  type: feedback
---

`internal/wire/testdata/messages.golden` records one hand-written body per
message kind, deliberately: a `start` carrying a real roster would move on every
balance commit. The cost of that safety is a **blind spot on the producer**, and
it is measured rather than argued.

**Measured 2026-09-03** (PR *nobody forfeits*): taking one line out of
`internal/room/gate.go` so the room stops setting `TurnCap` on the
`wire.Welcome` it builds leaves **`internal/wire` entirely green — the golden
included** — because the golden's welcome is a *fixture's*, not a room's. Two
tests in `internal/room` catch it by name, and both were written in that same PR:
one compares every welcome field against the room's own `Config`, one asserts the
client stopped on the turn the room stopped on. Without them **nothing at all**
noticed.

**Why:** a golden over hand-written fixtures answers *does this field travel, and
in what bytes* and cannot answer *does anybody fill it in*. Same family as
[[two-screen-goldens]] — which suite can see which mutation — and the same shape
as the platform rule about mutating the producer rather than the logic.

**How to apply:** adding a field to a `wire` message is **two** obligations, not
one — a golden entry saying it travels, and an assertion in `internal/room` (or
whatever builds the message) saying it is populated from the configuration it
comes from. When a mutation to a producer leaves a format golden green, that is
the golden working as designed, not a reason to widen it: put the assertion where
the producer is. A compile-failing mutation (deleting the struct field) measures
nothing here — drop the *assignment* instead.
