---
name: a-blanket-refusal-hides-where-its-legs-live
description: Narrowing one broad refusal means finding every book that owns a leg of it — one leg of the health rule was invisible from the file the refusal lived in.
metadata:
  type: project
---

`status.ParseBook` refused a health modifier outright, and its comment named two
reasons: a raise has to decide whether current health follows, and a drop has to
decide what happens to a unit already above the new maximum. Both are the same
question, and answering it ("current health never has to move") is only safe while
a maximum **cannot fall**. So the blanket became two narrower refusals in the same
file: permanent only, and positive only.

That is not the whole rule. A permanent status is normally the one thing nothing
takes back — `Set.Remove` refuses one, which is what stops a dispel turning a
trait off — but `Battle.reconsider` calls `Set.Release` when a **gated** trait's
gate closes. So a gated trait carrying a health term raises its holder's maximum
when the gate opens and drops it when it shuts, which is exactly the fall the
other two refusals were written to prevent. `status.ParseBook` cannot see a gate:
it parses statuses, and the gate is on the passive. The third leg had to live in
`passive.ParseBook`.

**Why:** a broad refusal is a single `if` that stands in for several distinct
rules, and each of those rules has its own natural home. Narrowing it in place
looks complete — the file compiles, the tests the refusal owned still pass — while
a leg that belongs in another book quietly ships as nothing.

**How to apply:** when narrowing or removing a blanket refusal, ask what else in
the codebase can produce the state it was preventing, and grep for the *mechanism*
rather than the value — here, every caller of `Release` rather than every mention
of health. Each answer that lives in a different book is a leg that needs its own
refusal and its own test there. See [[hexarena-strict-json-stops-at-a-custom-unmarshaller]]
for the other shape of this: a guard that reads as covering everything and covers
one layer.
