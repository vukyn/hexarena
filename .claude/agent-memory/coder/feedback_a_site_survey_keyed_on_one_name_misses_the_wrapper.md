---
name: a-site-survey-keyed-on-one-name-misses-the-wrapper
description: FRG-004 — "10 forge.Load call sites" was 11; check.go reached the data directory through forge.Inspect(dir), a wrapper spelled nothing like Load. Key an exhaustive walk on the widest observable marker (registering --data), not on the function you grepped
metadata:
  type: feedback
---

An exhaustive claim about "every site that does X" must be keyed on the widest
marker the source can show, not on the name of the function you happened to
grep for.

**Why:** FRG-004's handed-in survey said `cmd/hexforge` had **10**
`forge.Load(*dir)` call sites and listed them correctly. There were **11**
paths reaching the data directory: `check.go:22` called
`forge.Inspect(*dir)`, which is `Load(dir)` and then `lib.Inspect()` — a
wrapper that reaches the directory just as hard and is spelled nothing like
`Load`. A `grep forge.Load` cannot see it, and neither can an AST walk
matching that selector. The same session's other correction was of the same
family: `cmd/hexarena` was said to call `seed.Books()` at "all five of its
sites" — it has **four** production sites; the fifth was
`cmd/hexarena-host`, a different binary.

**How to apply:** when writing the bijective walk that guards the rule, pick
the predicate one level *wider* than the thing you are actually policing.
Here the walk matches **`dataFlag` being registered** rather than any call
into `forge`: a subcommand that takes `--data` has declared it touches a data
directory whatever it then does with the string, and one that reached a
directory *without* taking the flag would be a worse defect than the one being
fixed. The narrow check still exists, as a second assertion (nothing but the
two load helpers may call `forge.Load`/`LoadEmbedded`/`LoadForReading`/
`Inspect`) — but it is the *consequence*, not the census.

Corollary for the report: re-measure every figure a spec hands you, including
line numbers (`model.go:774` was `:781`). Three rounds running, the correction
was the most valuable thing handed back.

Related: [[feedback_identifier_regex_hits_field_keys]],
[[feedback_a_shared_screen_has_no_tool_only_wording]],
[[feedback_measure_the_term_before_optimising_it]]
