---
name: measure-in-the-slot-it-competes-for
description: A shared squad shell prices ONE slot; putting a second wall in the flex slot priced the composition (381‰) not the unit (491‰) — and a win rate pinned at 0 in both arms measures nothing, where a per-blow figure reads the multiplier itself
metadata:
  type: feedback
---

A borrowed measurement shell answers the question **its** variable slot asks. Ask
what slot the new thing competes for, and vary that one.

**Why:** `internal/seed`'s `aSquadOf` holds a striker and a wall and varies the
third member — right for the mender it was written for. Dropping a second **wall**
into that slot makes the home squad two walls and one striker against two strikers
and one wall, so what comes back is a composition reading wearing the unit's name:
**381‰ / 360‰**, under the shared 450‰ floor, with the character behaving exactly
as designed. Swapping the *wall* instead — striker and flex held — read **491‰**
against the incumbent, and the shell's own control (the incumbent against a copy of
itself) came to exactly **500‰**, which is the only thing that proves the shell is
not reporting its own bias.

**How to apply:**

- Write a second shell rather than parameterising the first. A shell that can vary
  either slot is a shell whose reading has to be read twice to know what it priced.
- **Always fight the control.** `fightSquads` runs every seed from both slots, so
  a squad against a copy of itself must be exactly 500‰; assert it in the same
  test, before the reading, as a `Fatal`.
- ⚠️ **A win rate can be structurally unable to move.** Pricing a second element
  by win rate against the unit's two counters would have been nought-against-nought
  in both arms — 0 of 80 either way — so the mechanism could work perfectly and the
  reading would be flat. The instrument that worked was **damage taken per blow**
  (`total taken / total blows received`): it *is* the multiplier, it has no floor to
  pin against, and it came back at 143 vs 211 (ratio 0.68 where the chart says 0.67)
  and 255 vs 170 (1.50 where the chart says 1.50). Pick a quantity the mechanism
  moves *directly* over a quantity it moves *eventually*.
- Both halves must be totals divided by totals. A per-seed average truncates to
  nought on a twelve-turn duel, which is the Machop row of noughts again.
- A ratio is also what makes two readings of different **lengths** comparable:
  60 duels of one build ran 3776 turns and the other 1681, so a bare total of
  damage said the passive build was more aggressive. Per hundred turns it says the
  opposite, which is the truth.

Related: [[a-well-formed-measurement-can-measure-nothing]],
[[a-null-needs-two-controls]], [[a-two-point-reading-names-no-distance]],
[[fixture-decides-what-is-visible]].
