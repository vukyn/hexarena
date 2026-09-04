---
name: hexarena-roster-cannot-price-damage
description: "hexarena's roster win rate is NON-MONOTONE in ally damage — buffing the ally lowers its win rate, so a win rate cannot price crit/power/pierce"
metadata: 
  node_type: memory
  type: project
  modified: 2026-08-28T20:28:04.542Z
---

**The roster win rate cannot price a damage change.** Measured 2026-08-28, 20,000 seeds each, post-#136 roster, baseline **47.55% ally**:

| change | ally rate | Δ |
|---|---:|---:|
| `razor_leaf` crit 200 — ally lands it 3.2:1 | 45.41% | −2.14pp |
| `sludge_bomb` crit 200 — **ally-only**, enemy never casts it | 45.15% | −2.40pp |
| `sludge_bomb` power **+5%, no crit** | 45.80% | −1.75pp |

The last row is the control: **giving the ally more damage lowers the ally's win rate**, by about as much as the crit did, with no crit involved. Suspected cause (unconfirmed): the ally composition wins by attrition — `venom_blood` replies, `leech_seed`, `endurance`, DoT — so killing faster cuts short the thing it wins with.

Two corroborations. The same change (`razor_leaf` crit 200) read **+2.40pp before #136** and **−2.14pp after** — the sign flipped on a *placement* change unrelated to crit. And the mechanism was verified sound meanwhile: 20.46% measured against 20.00% declared over 3,670 landing strikes.

**Why:** this is the same class of failure [[hexarena-speed-and-measurement]] records for speed — a win rate cannot price a non-monotone quantity — and it silently produces a plausible number instead of an error. It applies to `pierce`, `power`, `accuracy`, `strikes` and `self_gradient` too, not just crit.

**How to apply:** never price a skill field by roster win rate — use **`hexforge weigh <character> <skill> --field <name> --values a,b,c`**, shipped in **#139** (`internal/forge/weigh.go`). It twin-duels the carrier against a copy of itself with only that field changed, so stats, kit, placement, level, matchup and turn order cancel by construction. Three things about reading it: the **control row** (the skill's shipped value) must read exactly 500‰ or the whole report is refused, so the instrument self-checks on every run; **`worth` and `turns` are co-equal** — a mirror-duel rate once failed to *order* the speed amounts because a rate is lumpy at the kill threshold, and median turns is the reading taken inside one battle; and a row that landed nothing is **refused**, because *worth nothing* and *not rated* are different answers. Known-answer cross-check that must pass before trusting any figure: `--field power` above the shipped value reads above the control by more than the band **and** turns fall. Also check the carrier is even reachable first: measured per-side landing strikes per 2,000 battles showed `fire_fang` at **ally 0 / enemy 2400** (a nominally two-sided skill that is in practice enemy-only, because `ally.charmander` is level 8 beside level 30/60 units), while `razor_leaf` runs ally 37380 / enemy 11702.

**`--carriers all` (#152)** prices a field once per character that brings the skill, finding the carriers itself. ⚠️ **No headline number and no average, deliberately** — a weighing is a price against *a copy of that carrier*, so two rows were fought against two different opponents and are not two readings of one quantity; an average of them has no referent. A row compares only to *itself* at another value. Same reasoning fixes the sort: by worth at **one named column**, never by a scalar over the sweep. ⚠️ **On the shipped cast every table has exactly one row** — Bulbasaur, Charmander, Squirtle and Naruto have four **disjoint** learnsets, so no skill is brought by two characters and "skipped 3 of 4" is the main content. Still useful at one row (it finds the carrier and surfaces refusals inline); built for the cast the roadmap wants.

Related: [[hexarena-crit-mechanic]], [[pin-comparison-base-to-worktree-head]].
