---
name: a-cast-count-is-not-a-squad-count
description: Counting the cast says what a DRAFTED squad can field, never a saved one — a saved squad may field the same character twice, so one carrier still reaches a rung of two.
metadata:
  type: project
---

`carriersByElement` counts characters in `cast.json`, and every element-bonus test
built on it read "one carrier" as "no squad can field a tribe of it". Light and
ice have one carrier each, so both were skipped as unreachable, and retiring
`same_element` was cleared on that basis.

The premise is only half true. `internal/draft` records both halves and they do
not contradict: the draft pool is **exclusive**, so a drafted side is six or ten
*different* characters by construction — but **a saved squad may field the same
character twice** (`draft/draft.go` Pick, `draft/arrange.go`). A saved squad of two
Lapras is two ice units, reaches `MinimumRung`, and after the retirement is paid
nothing at all.

**Why:** the cast is the *supply*, and a rung is a question about a *squad*. They
are the same number only under a rule — no duplicates — that applies to one of the
two ways a squad is built. A count taken over the wrong population reads as a
fact, produces no error, and quietly authorises the decision it was consulted for.

**How to apply:** before treating a cast count as a reachability bound, name which
squad it bounds. If a saved squad can reach the threshold and a drafted one
cannot, both are real and the gap has to be decided rather than skipped. Where the
gap is accepted, pin the set **in both directions** so it cannot grow quietly —
`TestTheElementsWithNoBonusAreExactlyTheOnesWithOneCarrier` fails on a third
element losing coverage *and* on a bonus authored for a tribe only a doubled-up
squad can field. Related: [[a-fixture-chosen-by-property-is-blind-to-other-properties]].
