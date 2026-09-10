---
name: a-refinement-beyond-the-spec-owes-its-own-test
description: The spec's test list covers the spec's rule — a narrowing you add on your own initiative is outside it and ships unguarded unless you notice; the tell is a ⚠️ comment stating a distinction no test names
metadata:
  type: feedback
---

A step spec listed four tests for a three-line rule. Implementing it I narrowed
one line further than asked — the `--data` fallback probe keys on
`fs.ErrNotExist` rather than on "the stat returned an error" — wrote the reason
into a ⚠️ doc comment, and wrote no test for it. All four spec tests passed. The
coordinator widened it to `err != nil`, ran the package, got `ok`, and sent it
back.

**Why:** a test list is derived from the rule as specified. The moment you make
the rule *stricter* than the spec, the list is one case short by construction,
and the missing case is invisible precisely because everything asked for is
green. The comment explaining the refinement reads as though it were enforced —
it is the most convincing form an unguarded claim can take.

**How to apply:** when writing a ⚠️ comment that says "X rather than Y, because
Y would…", stop and ask which test goes red if the code were Y. If the answer is
none, that comment is a specification and it needs a row. Then check the
discriminator is cheap before assuming it is not: this one needed no permission
tricks — a plain **file** where a path component should be a directory makes
every stat below it fail with **ENOTDIR**, which is not `fs.ErrNotExist`, so
`internal/seed` written as a file makes `internal/seed/data` neither present nor
absent.

Two details that made the resulting test worth having:

- **Assert the consequence, not only the mechanism.** The probe check was a
  `Fatal` at first, so under the mutation the test stopped at "the probe is
  wrong" and never reached "a library came back". Demoted to `Error`, all three
  halves report. → [[feedback_a_well_formed_measurement_can_measure_nothing]].
- **One arm, and say why the other is absent.** The `dataGiven: true` arm
  short-circuits before the probe, so it stays green under the mutation; adding
  it would have padded the suite with a row that cannot fail for the reason the
  test is named for. → [[feedback_a_mutation_must_hit_the_arm_you_claim]].
