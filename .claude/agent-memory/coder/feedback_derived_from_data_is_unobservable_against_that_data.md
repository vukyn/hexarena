---
name: derived-from-data-is-unobservable-against-that-data
description: "\"The thresholds are read off the chart, not hardcoded\" stayed green with 1500/667 written in, because every test asked the shipped chart and both spellings agree there. Retune the data — two tunings, one per half of the switch."
metadata:
  type: feedback
---

A claim of the form *"this value is derived from the data rather than written
down"* cannot be tested against the data it was derived from. Retune the data.

**Why:** `internal/screen/matchup.go` maps an element multiplier to a mark by
asking the chart — `multiplier > levels.Advantage` rather than `> 1500` — and
the doc comment gave the reason: a second copy of a balance number draws the
wrong mark the day somebody retunes `elements.json`. A reviewer replaced the
four lookups with the shipped constants and `go test ./internal/screen -count=1`
came back **ok**. Every test in the package, including the one that builds its
expectations *from* `chart.Multipliers()`, then asks that same chart — where the
two spellings agree by construction. The comment read as though it were
enforced; it was decoration, the same shape as a ⚠️ note no test names.

What closed it: `element.ParseChart` takes raw JSON, so the fixture is the
shipped `elements.json` with three numbers swapped and everything else carried
through as `json.RawMessage` — real cycles, real mutual pair, `Chart.Validate`
really run. **Two tunings, because each catches half the switch:**

- **wider** (2000/1000/500) — a *single* advantage is 2000 and a single
  disadvantage 500, which hardcoded 1500/667 read as *doubled*: catches `+`, `-`
- **narrower** (1200/1000/850) — a *doubled* advantage is 1440 and a doubled
  disadvantage 722, which hardcoded read as *single*: catches `++`, `--`

Two more rules that made it real. **Assert the marks, never the multipliers** — a
check that `Multipliers()` came back different proves the fixture retuned and
says nothing about what the code consulted. And keep a **discrimination net**: a
row where the shipped numbers would have answered the same thing is green
without measuring, so the test records which rows *disagree* and fails unless all
four arms are among them. The oracle for that net is the mutation itself, kept
in the test and labelled as such — the arrangement `narrowSwung` already uses in
`internal/core/combat/swung_test.go`.

**How to apply:** whenever a comment says "read off X rather than written down",
the test has to hand the code a *different* X. If X is a data file the repo
parses, build the variant through the real parser rather than mocking. And check
what actually goes red: here, under the mutation the new test was the **only**
failure in the whole package — which is the measure of how unheld the property
had been.

Related: [[feedback_a_probe_whose_fallback_coincides]],
[[feedback_a_refinement_beyond_the_spec_owes_its_own_test]],
[[feedback_a_green_expected_mutation_proves_nothing_by_itself]],
[[feedback_a_null_needs_two_controls]].
