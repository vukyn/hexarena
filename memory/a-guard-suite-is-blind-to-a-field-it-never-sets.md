---
name: a-guard-suite-is-blind-to-a-field-it-never-sets
description: cmd/hexarena-tui/readonly_test.go's five tests all build the client with no player squad file, so `Authoring: len(m.player) > 0` in model.ctx left every one of them green — a guard suite cannot see a capability keyed on a field its own fixture leaves at the zero
metadata:
  node_type: memory
  type: reference
---

**A suite that holds "this client cannot author" is blind to authoring keyed on
a field its fixture never sets.** `readonly_test.go` is five tests, all four of
the ones its header describes plus `TestNoScreenInThisClientAsksOrPicks`, and
every one of them starts from a model built with **no player squad file**. So
the mutation

	Authoring: len(m.player) > 0,   // in model.ctx

— a client that starts authoring the moment a player has a file of their own,
which is the single most plausible accident while wiring one up — leaves **all
five green**. Measured, not reasoned about: `go test -run` over the four named
tests answered `ok` with the mutation in place.

**Why it matters:** those five tests are the whole of the evidence that a game
client does not write the game's data, and the reason they are trusted is that
they press real keys through the real model. That is true and still not enough —
what they press keys on is one *configuration* of the client, and a capability
flag derived from a **new** field is a second configuration nothing in the file
reaches. The suite's strength (it drives the real thing) is exactly what hides
this: nothing is stubbed, so nothing looks suspicious.

**How to apply:** when a client gains a field that a capability could be
derived from, the guard suite owes a case that *has* the field. Here that is
`TestAPlayerFileTurnsNoAuthoringKeyOn` — the same walk over `authoringKeys`,
plus every key `everyKeyPressed()` can send, on a client that carries a player
file, and the file's bytes compared before and after. Do not widen the existing
five to take the field as a parameter: they are about the ordinary client, and
a table that swept both would make a failure ambiguous about which
configuration broke.

This is the [[fixture-hidden-branch]] shape with the branch on the *outside* of
the fixture rather than under an early return: nothing is unreachable, the whole
suite simply runs one arm of a fork it does not know exists.
