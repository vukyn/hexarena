---
name: hexarena-the-summon-branch-skips-the-price
description: hexarena — Suggest's summon arm takes summonWorth and continues, so a summoning skill never reaches prices.rate and nothing it costs its caster was ever subtracted
metadata:
  type: project
---

`Suggest` rates a summoning skill in its own branch:

```go
if value := b.summonWorth(actor, declared); value > 0 {
    take(Choice{...}, value, declared.Cooldown)
    continue          // ← never reaches prices.rate
}
```

`prices.rate` is where `total -= p.spentHealth(actor, declared)` lives. **A
summoning skill never gets there**, so anything it costs its caster was rated at
nought — `skill.Cost` on a summon, and `Summon.Splits`, which tears off a share of
the caster's maximum health.

**What it looked like from outside.** `pokemon.diglett` splits itself for three
tenths of its maximum, capped at twice a battle. Measured on autopilot: **178
casts over 90 duels** — every allowance spent, every time — and the kit holding it
read **434‰** against the same character with a plain attack in that slot. The
rating was buying a body and being charged nothing for it, so it always bought.
Subtracting the cost in `summonWorth` took it to **90 casts over 90 duels** (the
first split taken, the second declined) and the kit to **504‰**, which is a skill
slot's worth — what a signature should be.

⚠️ **A wrong diagnosis got as far as a merged comment.** The engine PR's
`aHandPlayedSplit` said *"the rating has no term for this, so Suggest never casts
a split"*. It casts them constantly. The hand-played helper was needed because the
bench skill sits in no bench roster's kit — nobody could cast it — not because the
rating refused. Probe before writing why something did not happen: thirty lines
counting `SkillUsed` answered it in one run.

**How to apply:** anything added to a **summon** needs its price added in
`summonWorth`, not in `price.go`, and the two files do not point at each other.
Related: [[hexarena-a-trait-that-changes-a-price-has-two-sites]] and
[[hexarena-a-new-clause-needs-every-site-that-evaluates-it]] — the same mistake
three times now, each one a new field whose second reader nothing named.
