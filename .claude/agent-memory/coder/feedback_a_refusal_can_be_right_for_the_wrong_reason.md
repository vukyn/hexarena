---
name: a-refusal-can-be-right-for-the-wrong-reason
description: hexforge weigh — a guard whose verdict is right can still hold the wrong evidence; TODO entries name examples that do not reproduce, so re-measure the example before designing the fix
metadata:
  type: feedback
---

When a TODO/CLAUDE.md entry says a guard is wrong, separate **the verdict** from
**the evidence it holds**, and re-measure the entry's own named example before
designing anything.

**Why:** `weigh`'s `refuseUnreadable` refused every support skill. The entry
said the refusal itself was right (*worth nothing* ≠ *not rated*) and it was —
what was mis-specified was its *evidence*: a count of landed **damaging**
strikes, which proves nothing about a skill of nought power. Fixing the evidence
(`forge.Mechanism`: striking / applying / restoring / cleansing / summoning, read
off the variant **as fought**, counted off `[]battle.Event`) unblocked every
buff, debuff, heal, cleanse and summon without touching the verdict.
But the entry's named example, `poison_powder` on `pokemon.bulbasaur`, **still
refuses after the fix, and never failed for the stated reason**: the refusal said
*cast **0** times*, not "landed none". Bulbasaur is its only carrier, its fielded
trait `venom_blood` resists `poison` at 1000‰, and a weighing fights a carrier
against a copy of *itself* — so the only target on the board is immune to the
skill's only mechanism and `Suggest` never picks it. **Power 0 was never the
obstacle.** Two design premises in the same entry were false; the second
(`self_gradient` "is two numbers") was false too — `skill.Gradient` has one
field and has since #132, and its real blocker is that its *off* state is a nil
pointer rather than a nought.

**How to apply:** before implementing from a `TODO.md` item in this repo, run its
example and read the refusal's exact words. If the words do not match the entry,
the entry's diagnosis is stale and the scope changes — say so and re-scope rather
than implementing the described fix. Related: [[measure-which-guard-masks]],
[[fixture-decides-what-is-visible]].

**A refusal-table row that asserts only `err != nil` can pass for the wrong
reason, and mutation is the only thing that says so.** `cmd/hexforge`'s
`TestWeighRefusesAnArgumentListItCannotAnswer` is fourteen argument lists, each
asserted merely to be refused. A row added there for "a gradient on a skill that
declares none" **passed under the mutation it was written to kill** (seat (b):
nought mapped to nil inside `WeighField.set`) — because the sweep then ran, and
one of its rows was refused as *saturated* at `--seeds 2`. Verdict right, evidence
unrelated. The fix is a test of its own that asserts the parser's **sentence**
(`want a share in parts per thousand`), not the presence of an error. Any table
row whose only claim is "something was refused" is worth this check.

**Two durable facts this turned up, both about reading `[]battle.Event`:**
- A skill's `restores` emits a `Healed` carrying **neither the skill nor the
  caster** (`Actor` is whoever's health went up). Attribute it to the **cast in
  progress** — exact, because a cast resolves whole before the next begins. The
  other two `Healed` are told apart by what they *do* carry: a regeneration names
  the status that ticked, a drain always carries its share (a share of nought
  heals nothing and emits nothing). If `Skill` is ever added to that event, delete
  the rule rather than keeping a second reading of one fact.
- A `weigh` reading costs `2 × seeds × rows` battles, but **battle length
  dominates the wall clock**: `pokemon.cleffa` (118 median turns) fought 80,000
  battles in 115 s, `naruto.naruto` (273 turns) 20,000 in 135 s — 5× per battle.
  A row budget in battles is not a row budget in minutes.
