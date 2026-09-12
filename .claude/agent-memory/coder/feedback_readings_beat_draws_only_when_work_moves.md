---
name: readings-beat-draws-only-when-work-moves
description: "hexarena's '130 readings against 203 views' argues for MOVING work out of the draw, not for rendering a second thing in the reading — adding a render is 260 against 203. Measured on PlayScreen: 4.7µs a narrow roster, 5.5µs a wide one, ~0.2ms over a whole battle, so the cost side decides nothing and the contract does"
metadata:
  type: feedback
---

`internal/screen`'s reading is taken per **message** and the draw runs per
**frame** — about 130 readings against 203 views over one 3v3 bo1 — which is why
`readBattle` renders the board, roster and order rather than carrying units for
the draw to walk. Reaching for that figure to justify rendering **two** roster
tables in the reading is the figure used backwards.

- *Moving* one render from draw to reading: 203 → 130. Cheaper.
- *Adding* a second render to the reading: 203 → **260**. Dearer.

**Why it still shipped that way:** measured rather than argued. On this package's
3v3 fixture a narrow `tui.Roster` is **4.7µs** and `tui.RosterWide` **5.5µs**, so
carrying both costs about **0.2ms more over a whole battle** than formatting one
at the draw — against a battle that lasts minutes. Neither side is a cost, so the
cost side decides nothing, and the decision fell to the contract: `playReading`
holds its sections **already rendered**, and carrying rows for one of four would
need a row type exported out of `internal/tui` plus a second entry point to feed
it back, only so the format stays in the package that owns it.

**How to apply:** when a brief hands you a ratio as the cost argument, check
whether the change *moves* the work or *adds* it — the ratio only prices a move.
Then take the measurement anyway: at these magnitudes both answers are free, and
saying so out loud is what lets the design be decided on ownership instead of on
a microbenchmark nobody can feel. Related:
[[feedback_measure_the_term_before_optimising_it]],
[[feedback_measure_the_thing_a_bound_bounds]].
