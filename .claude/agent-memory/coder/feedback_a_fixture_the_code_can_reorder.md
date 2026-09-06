---
name: a-fixture-the-code-can-reorder
description: A mutation that sorts a slice IN PLACE also sorts the test's expectation, so the order test passes — copy the expected order out before the call, and remember the shipped data may already satisfy the property
metadata:
  type: feedback
---

A test that asserts an **order** must copy the expected order out **before**
calling the code, not read it back off the same slice afterwards.

**Why:** measured in hexarena step 5b. The claim was "the draft screen draws the
pool in the order it was handed in, never sorted". The test handed in a reversed
pool and read the drawn rows back against `pool`. The mutation —
`slices.SortFunc(live.Pool, …)` inside the drawing — sorts **in place**, and
`live.Pool` is the same backing array the test handed in, so by the time the test
looked at its own expectation that expectation had been sorted too. **Zero tests
failed.** Neither golden could see it either, because the shipped cast happens to
be in id order, so sorting it is a no-op there — which `draft.NewPool`'s own
comment predicted in as many words.

**How to apply:** before writing an order/identity test, ask what the code under
test could do to the value you are comparing against. A slice, a map or a pointer
handed in is a value the callee can edit; a copy taken first is not. Two sibling
traps in the same shape:

- **The shipped data may already satisfy the property.** A sort test taken
  against data that is already sorted measures nothing, whichever direction the
  code goes. Build a fixture whose order is *not* the property's order — and
  assert the fixture really is that way before asserting anything else.
- **The mutation must be run and the count read.** "Nothing failed" is the
  finding here, not a pass; a mutation that reddens nothing means the net is
  missing, and the second-cheapest thing after writing the net is writing down
  which test now catches it.

See [[fixture-decides-what-is-visible]], [[real-data-can-satisfy-the-property]]
and [[mutate-the-producer-not-just-the-logic]].
