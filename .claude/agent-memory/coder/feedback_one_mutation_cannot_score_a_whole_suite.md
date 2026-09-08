---
name: one-mutation-cannot-score-a-whole-suite
description: A change carrying several independent decisions needs one mutation PER decision — reverting hexarena's stage-element fallback reddened 8 of 13 new tests and left 5 looking blind that were guarding different decisions entirely
metadata:
  type: feedback
---

**"Confirm each new test goes red individually" is a claim about a mutation, not
about a suite.** If the change carries more than one independent decision, one
mutation scores only the tests that guard *that* decision, and every other test
comes back green — which reads as "measures nothing" and is the evidence a later
reader deletes it on.

**Why:** measured on hexarena `CAST-003` (a stage may declare its own element).
The plan named one mutation — revert the fallback to `return c.Element`. It
reddened 8 of the 13 new tests. The other five were fine: they guard three
*different* decisions the same PR made, and each needed a mutation of its own.

| mutation | what it is | went red |
|---|---|---|
| the fallback returns the character's | the mechanism does nothing | 8 tests, across 5 packages |
| the carry loop accepts if **any** form can carry | the quantifier | 1 subtest, the ungated arm |
| drop the per-stage `Chart.ValidateAffinity` | the new validation | 1 subtest of 5 in a table |
| `castReport` prints the character's element | the golden's guard | **nothing — correctly** |

Three findings worth carrying, in the order they cost time:

- **Mutate the ONE shared home, not the exported door.** The fallback lived in an
  unexported `elementAt(affinity, stage)` with the exported `Character.ElementAt`
  a one-line delegate, because the parser has to ask before there is a
  `Character`. Reverting only the exported method would have left the parser
  green and understated the blast radius by a whole package. One home is worth
  having partly *because* it gives the mutation one site.
- **A green-and-expected mutation is a result, and must be reported as one.** The
  golden line that will guard a mistyped `"elemnt"` key is provably insensitive
  **today**: no shipped stage declares an element, so printing the character's
  instead moves no byte and `TestCastGolden` stays green. That is the correct
  answer for this PR and prospective for the next — say so, rather than quietly
  omitting the row. → [[feedback_a_green_expected_mutation_proves_nothing_by_itself]]
- **A shared fixture helper reddens tests through its own `t.Fatalf`.** Two of
  the eight were red because the helper that builds the carrier could no longer
  parse its book, not because their own assertion fired. Still individually red,
  still honest — but say which, or the count overstates the coverage.

**How to apply:** before running the mutation the plan names, list the *decisions*
the diff makes (a fallback, a quantifier, a new refusal, a new record line) and
write one mutation per decision. Restore each by editing back and check the file
back to a `shasum` taken before the first mutation —
[[git-checkout-discards-to-head]] and
[[hexarena-a-mutation-baseline-must-come-from-git]] both bite here, and a hash is
the cheap proof the restore was exact.

Related: [[feedback_a_mutation_must_hit_the_arm_you_claim]],
[[feedback_a_well_formed_measurement_can_measure_nothing]],
[[fixture-cast-edit-costs-goldens]].
