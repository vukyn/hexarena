---
name: hexarena-a-golden-cannot-hold-a-keystroke-rule
description: hexarena — every fixture reaches the aiming state by assigning p.Aiming, so no golden measures the path to a state; a rule about which keystroke does what is held by nothing unless a behaviour test says so
metadata:
  type: feedback
---

Reversing "a skill with one legal cell does not ask where" was predicted, in
TODO.md, to move *every play fixture that casts a single-aim skill* — "the bulk
of the diff". It moved **no golden at all**.

The reason is that every fixture drawing the aim list gets there by assignment:

```go
p.Aiming = true
```

`screens_golden_test.aBattleAiming`, twice in `cmd/hexforge-tui/language_test.go`,
twice in `cmd/hexarena-tui/sweep_test.go`. A golden therefore holds the aiming
**state** and has never pressed the key that opens it.

That is correct for a golden — a picture of a state is what it is for — but it
bounds what the suite can be trusted to catch: **no keystroke rule is held by a
golden.** A change to which press does what can be green across every golden in
the repository.

**The second half is the one that actually got through.** `live_test.go` presses
enter inside

```go
for action.Kind == Stay && struck.Aiming {
```

which tolerates either answer by construction: it asserts a decision arrives
eventually and says nothing about how many questions were asked first. Reverting
`PlayScreen.answer` alone — the live twin of `choose` — reddened **not one test**
until a test was written for it. The local twin *was* covered, and that is what
made the gap invisible: the rule looked measured because half of it was.

**How to apply.** When changing what a keystroke does, mutate it back and watch —
and mutate each path separately, because two paths implementing one rule fail
independently. A `for … && screen.<flag>` loop in a screen test is a loop written
to be robust against the rule it walks past; treat every one as a place where a
rule may be held by nothing. See [[fixture-hidden-branch]] for the same
failure arriving from the other direction.
