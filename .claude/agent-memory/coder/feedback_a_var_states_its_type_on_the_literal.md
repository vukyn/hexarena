---
name: a-var-states-its-type-on-the-literal
description: "`var x = [element.Count]string{…}` leaves ValueSpec.Type NIL — an AST walk reading only the spec found 0 declarations and reported a clean package"
metadata:
  type: feedback
---

In Go's AST, `var name Type = …` puts the type on the `*ast.ValueSpec.Type`, but
`var name = [N]T{…}` puts it on the `*ast.CompositeLit` inside `ValueSpec.Values`
and leaves `ValueSpec.Type` **nil**. Both are ordinary, and the second is the
form a table is usually written in.

**Why:** a guard written to fail if a *second* table of element colours ever
appears walked every top-level `ValueSpec`, tested `value.Type != nil &&
namesAnElement(value.Type)`, found **zero** declarations across the whole
package — including the one it was written around — and passed. A survey that
finds nothing looks exactly like a survey that finds nothing wrong.

**How to apply:**

- Read **both** places: `value.Type`, and `literal.Type` for every
  `*ast.CompositeLit` in `value.Values`.
- Always assert the count of what the walk found, not only that nothing
  offended: `if len(found) != 1 { t.Fatalf(…) }` is what turned this from green
  to red. A walk with no positive control is a walk that measures nothing.
- The same trap applies to `const` blocks with implicit types and to any
  "is anything else declared like X" sweep.

Related: [[feedback_no_arithmetic_test_can_see_a_literal_that_agrees]] (why the
AST walk is needed at all), [[feedback_a_site_survey_keyed_on_one_name_misses_the_wrapper]]
(a survey keyed too narrowly), and
[[feedback_a_well_formed_measurement_can_measure_nothing]].
