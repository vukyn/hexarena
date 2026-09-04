# Balance

How the game is priced and tuned: what `Suggest` pays for, what a character is
made of, each of the mechanic categories and what it cost to add, and the two
instruments that measure a number — `hexforge weigh` and `forge.Bout`.

⚠️ **This was `CLAUDE.md` until 2026-09-05 and it moved for size, not for
importance** — that file is loaded whole at the start of every session and this
is subject matter rather than a rule that binds every edit. What stayed there is
what bounds an edit whatever it touches: § *Data and golden files* (goldens are
the design record, accepted only through `make golden`), § *Invariants worth
knowing before editing*, and § *Saturate continuous values, cap discrete ones*.
**Read this before authoring a character, a skill or a status, and before
re-pricing anything.**

## Rating an action: how Suggest prices what is not damage

`battle.Suggest` rates every option in one unit — damage — and everything that is
not damage is priced *in* damage, from the function that resolves it, over a capped
horizon (`price.go`). Four horizons: `buffHorizon` 3, `guardHorizon` 2,
`healHorizon` 2, `killHorizon` 1. Under-pricing costs a marginal cast; over-pricing
costs a kill.

Rules for anything added to that file:

- **Read the resolving function, never a second copy of its arithmetic.** A status
  tick comes from `inflict`'s own expression, origin and all; a restore from
  `combat.Rules.Restore`; a stat change from `modifier.Set.Stat` through
  `Battle.Stats`; a stack cap from the status book through `Set.With`. A price built
  from a second reading lets the opponent prefer a skill for something the skill
  does not do, and nothing reports the disagreement.
- **Weight a chance, never roll one.** `Suggest` may not touch `b.source`. Compose
  `amplify` (the *actor*) with `resist` (the *target*) and clamp last, exactly as
  `inflict` does. ⚠️ Those two take the same Go type: swapping them compiles,
  changes every price, and only `TestPricingAStatusUsesTheChanceThatWouldBeRolled`
  says so.
- **Worth nothing means not rated, never rated at nought.** A rating of nought is
  still a rating and would beat "found nothing", taking the turn ahead of whatever
  the fallback would have chosen.
- ⚠️ **Worth *less* than nothing is not the same as worth nothing, and may not be
  the fallback.** `rate` subtracts friendly fire and the recoil a skill puts on its
  own caster, so a negative total is reachable and means something exact: taking
  this turn leaves the board worse than not taking it. The two used to share a
  bucket — an option was skipped for scoring `<= 0` and then picked straight back
  up by the fallback — so the opponent cast skills it had just priced as a loss.
  A loss is now dropped from the fallback, and a unit with nothing but losses
  declines; `Suggest` returning false is the existing route to `Pass`, which every
  caller already drives. An option worth exactly nought is still a fine way to
  spend a turn nobody else wants.
- **Clamp every term against what the board can actually deliver.** A
  damage-over-time at the target's remaining health, a heal at the room *and* at
  what an enemy could take off, a charge at the strikes there are to eat. The
  unclamped versions are not conservative — they are the largest numbers in the
  file, and one of them was worth 19 points of the shipped roster's win rate.
- ⚠️ **A permanent status carries zero duration.** `min(duration, horizon)` prices
  it at nothing. Go through `turnsOf`.
- ⚠️ **`expected` reads the occupant of a cell**, so handing it a hypothetical unit
  silently gets the real one back. `Battle.against` takes the unit; use it.
- ⚠️ **Build a hypothetical with `status.Set.With`, never by copying a unit and
  applying to it.** A `Set`'s entries and each entry's stacks are slices, and
  `Apply` writes through them — a shallow copy refreshes the real unit's durations
  from inside the rating, and a refreshed status is indistinguishable from sustained
  pressure in every golden.
- **Tempo is priced from the stat, never the queue.** A wait is `atb.Scale / speed`,
  so a share on the stat is that share of the unit's turns; `tempo` reads that and
  asks nothing about who acts next. ⚠️ A turn is worth `turnWorth` — the **mean** over
  what a unit could point at somebody — and **not** `bestStrike`: charged at the best
  attack, `outrage`'s recoil made the dragon build avoid its own heaviest skill and
  its duel rate fell 26.6% → 20.0%, a rating playing worse while believing it had
  improved. The same figure prices both directions, so a haste and a slow cost the
  same.
- **An all-sided skill is rated by both halves.** `expected` gives the enemy half
  (it skips a unit on the caster's own side) and `friendlyFire` subtracts the own
  half — damage plus the turns lost with anything it kills, the caster **included**,
  because a shape can cover the cell it is cast from. ⚠️ The old refusal was not an
  oversight: the guard and "expected does not subtract an ally" were two halves of
  one decision, so relaxing either alone is the opponent that bombs its own squad.
  An all-sided attack also counts as an attack in `bestStrike`, `turnWorth`,
  `bestAgainst` and `worstStrikes`, through `aimedAtAnEnemy` so the four cannot
  disagree.
- **A tie is broken by what an option costs to have spent**, never by kit order:
  `take` compares value first and `declared.Cooldown` only on a tie. That is all
  *holding a skill for a later turn* honestly comes to here — damage is clamped at
  the target's health, so a nuke and a filler are worth the same against a sliver
  and the nuke was being burnt on it. ⚠️ A **discount** rather than a tie-break
  would price scarcity by guessing at turns, which is the mistake tempo was
  corrected for. Measure a change like this **head to head** (the tie-break on one
  side at a time): both sides use `Suggest`, so the roster rate shows which squad's
  kit gained, not whether the rating improved.
  ⚠️ **The FALLBACK obeys the same tie-break**, and did not until 2026-09-02.
  `Suggest` keeps one option worth nought in case nothing at all is worth doing,
  and that arm kept "the first such skill in kit order" long after `take` stopped
  doing so — so kit `[scour, wipe]` cast the three-turn cleanse and `[wipe, scour]`
  cast the free one, with kit order the whole of the decision. Options worth
  *nothing* are the sharpest case of the rule rather than an exception to it: both
  buy nought, so what casting one costs is all there is to separate them.
  ⚠️ **And when the cheapest of them still costs a cooldown, the turn is
  DECLINED.** `Suggest` returns no choice, which `RunToEndWith` turns into
  `Battle.Pass` — the mechanism already existed — and the pass carries
  `DeclinedReason` rather than `NoActionReason`, because a unit with no move and a
  unit that had one and would not take it are different facts about the board. A
  cooldownless option is still cast: what this file cannot see is always something
  rather than nothing, so refusing a free cast could throw away a real effect while
  refusing a priced one only declines a turn nobody had a use for. It waited for
  all six pricing gaps to close, and that was the right order.
  ⚠️ Not the *waiting* TODO.md decided against: that is passing to get a skill
  back sooner, which stays empty because `spendCooldowns` runs on a pass and an act
  alike. This is not starting a cooldown, and the same arithmetic is what makes it
  sound.
- ⚠️ **A repeating count is read as its EXPECTATION, never as either end.**
  `Rules.Expected` goes through `Hit.ExpectedStrikes` (parts per thousand, so the
  division lives in `Expected` and not at the call site) and `hitAgainst` carries
  `Repeat` and `MaxStrikes` onto the `Hit`. Both halves or neither: with the
  fields dropped, `ExpectedStrikes` answers the plain count and fixing `Expected`
  alone moves nothing. `pricing.worstStrikes` reads the same figure and returns it
  in per mille — a charge cancels ONE strike, so it is worth *less* against an
  attacker that keeps going, and a whole number cannot carry a count of 3,120
  without inflating every guard priced against it.
  ⚠️ **`Rules.Total` deliberately stays on `StrikeCount`.** It is the
  deterministic figure `skills.golden`'s damage column is written from, and a
  number that moved with an expected value would stop being what an author
  compares two skills by. The two answer different questions.
  Measured: a Magnemite kit built on `spark` read **3.6%** with the floor and
  **25.0%** with the expectation — the rating would not cast its own best skill.
- **Health a skill gives back is priced as healing, from both sources.**
  `pricing.drained` reads `drainShare(declared.Drains + lifesteal(actor))` — the
  expression `resolveAgainst` pays it with, sum first and then bound — over the
  damage `expected` says will land, and runs it through `worthHealing`. Reading
  only the skill's own field prices `leech_seed` and leaves `blood_thirst` at
  nought, which is half a fix; skipping `worthHealing` makes it a flat bonus
  wearing a health check.
- **A restore is priced from both its halves too, through `swingOf`.**
  `pricing.restored` reads `declared.Restores + swingOf(...).Restore`, which is
  the expression `Battle.restore` pays with — one function, so the price and the
  payout cannot drift. A reserve-paid heal carries no flat `restores` at all, so a
  price read off that field alone rates every one of them at nought and `Suggest`
  never casts one. The cost side is `pricing.selfSpendable`, which has a health
  arm beside its damage arm: a per-stack restore needs no share-of-a-cast because
  it *is* the per-stack figure, and it clamps through `worthHealing` against the
  holder — exact rather than a guess, because the skill is aimed at its caster.
- **A blow is discounted by an absorbing POOL on its target, and an unblockable
  one is not.** `Battle.pastAPool`, read in `against` so every damage figure in
  the file inherits it. What a pool takes over a volley is the smaller of the pool
  and the damage, which is what `combat.Absorb` comes to across one — not a second
  copy of it. **A wall of block CHARGES is `Battle.pastAWall`**, and it cancels a
  STRIKE rather than damage — that is `warden`'s whole trade, so a wall answers one
  heavy blow and multi-strike answers a wall.
  ⚠️ **It is charged on every cast and a charge is only paid for once**, which is a
  real over-count and is accepted: measured, every discount small enough to leave
  the balance claims standing reads inside the measurement's own band against the
  frozen ruler, and every discount large enough to clear that band moves them.
  Monotone both ways, no setting in between. → `TODO.md` for the sweep.
  ⚠️ **It took two dead hypotheses and a broken instrument to land.** The boards
  used to judge it were blind — the shipped roster carries no guard, and the
  wall-heavy board built for it does not resolve — so the gain read as nothing.
  Build a wall board that FINISHES before quoting a figure about a guard.
- **A discharge is priced on the turn that fires it, not only on the turn that
  charges.** `pricing.discharged` walks `chainFrom` — the same function the
  resolution walks, so the aim gates it identically — and prices each carrier at
  the resolving `Rules.Damage` expression with `ArcPower * Takes(held)`, over the
  expected strikes weighted by the chance to connect (an arc rides a strike that
  did not MISS, so a blocked one still fires) and capped at the rounds the carrier
  has stacks for. ⚠️ Before this, `ArcPower` appeared in one place in the file —
  `spendable`, which decides whether *laying a charge down* is worth a turn — so a
  conduit's whole payload was a free rider on the skill's own power. The aimed
  carrier's arc is clamped at what the blow itself leaves of it, or a conduit
  aimed at a sliver is worth that health twice.
- **Every status category has an arm.** `granted` takes the ones a holder wants
  — Regen, Shield, Absorb, Buff, Reserve, **Taunt** — and `inflictedOn` the ones
  it does not — Dot, Control, StatDebuff, Charge, **HealCut**. ⚠️ A taunt is a
  `granted` case even though `Category.Harmful` says otherwise, because
  `tauntStatus` **sits on the unit doing the taunting**: pricing it as harm bills
  its own caster its best strike three times over for casting it. Worth the aim it
  takes off every enemy at once, and nought against an enemy the holder was
  already the best target of — which is why it reads as exactly nothing in a duel
  and has to be measured in a squad. A heal cut is priced through `healingFor`,
  the expression a heal is paid with, over `Set.PendingIn(Regen)` — the
  regeneration visibly owed, deliberately less than the truth, because every other
  source of healing needs a lookahead this file does not have.
- ⚠️ **The queue may break a tie; it may never set a price.** A queue reading is
  **compared**, never added or multiplied. A value that reaches an arithmetic
  expression is tempo, and tempo is priced from the **speed stat** — see the tempo
  bullet above. The reason is not taste: the queue is discrete and lumpy, so a few
  points of speed buy whether one more turn lands before the other unit acts, and a
  mirror-duel win rate could not even *order* the shipped speed amounts (`+150` read
  59.0% while `+50` read 74.0%; see `swiftness` under § *Open work*). A number that
  lumpy may decide which of two equals to take. It may not say what either is worth.
  The rule stands even though nothing currently uses it, because it is the line
  between a tie-break and a second tempo.
  ⚠️ **A third key under `cooldown` was built, measured and thrown away.** It took
  the aim whose occupant acted soonest, off the non-mutating `atb.Queue.Pending`.
  Head to head against the rating without it (`forge.Bout`, 10,000 seeds, control
  exactly 500‰): **500‰ exactly, 10,000 wins and 10,000 losses.** Not "inside the
  band" — the *control signature*, meaning the two ratings played the identical
  battle every time. It moved **0 of 93,320 decisions** over 2,000 shipped battles,
  and a census found **0 prompts** where two options came out level on both value
  and cooldown. **No golden moved.**
  The premise was wrong rather than the code. "One skill pointed at several cells
  has the same cooldown on every call, so the winner is whichever cell `hex.Cells`
  lists first" is true, but a tie needs the **values** level too, and shipped units
  differ in health, defence and affinity, so two aims almost never rate to the same
  integer. Reaching the key at all needed a fixture tying two enemies at identical
  health on purpose. **Do not rebuild it without first showing the tie exists on the
  board in hand.**
  ⚠️ Two things learned building it are worth more than the key was. **Absence must
  be carried beside a queue reading, never encoded into it** — `Queue.Pending`
  answers 0 for a unit it has never heard of and 0 is *soonest*, so an aim with
  nobody to read would have outranked every aim with somebody; a self-aim has to be
  *declared* unread rather than detected, since the cell is occupied by the actor
  and a fast unit would otherwise have every self-aimed skill outrank every attack.
  And **`Suggest` may not call `Standings`**: it orders the real slice in place, so
  a rating that called it would move nothing on the board — `describeBoard` could
  not see it — while changing the order units act in afterwards, arriving in every
  golden at once with nothing naming it. Read `Pending` or `Preview`.
- **Waiting is arithmetically empty in this engine**, and is therefore *decided
  against* rather than unbuilt. `spendCooldowns` brings **every** cooldown down by
  the turn just served, and it runs at the end of `Act` (`turn.go:541`), at the end
  of `Pass` (`:323`) and on a turn control took (`:92`); `Act` then starts a
  cooldown on the one skill it cast and on nothing else. So the skill being "waited
  for" comes back on the same turn whichever the unit does, and across its next two
  turns *act now* is worth `bestValue` plus next turn's best while *wait now* is
  worth nought plus that same next turn's best. **Acting dominates by exactly
  `bestValue`**, and an option priced below nought is already declined — so there
  is no residue for a waiting rule to collect.
  `TestAPassBuysNoCooldownAnActDoesNot` holds the premise (a pass buys no cooldown
  an act does not) and `TestNothingWaitsOnPurpose` holds the consequence (a skipped
  turn carries one of three forced reasons and never a fourth).
  ⚠️ **Two further bars, if it is ever re-raised.** A *simulating* lookahead has to
  clone a `*Battle` — an unexported queue, `[]*Unit`, a `status.Set` whose entries
  are slices, and the `*rng.Source` — and then resolve into the clone. That is
  either `resolveAgainst`, which **rolls**, breaking *weight a chance, never roll
  one*; or a weighted twin of it, breaking *read the resolving function, never a
  second copy of its arithmetic*. Two available implementations, one broken rule
  each. And it costs roughly **×36 a turn** on the shipped kits, taking a
  20,000-battle sweep from ~7.5s to ~4.5 minutes.
- **An attack is charged for the reply it provokes.** `friendlyFire` is what a
  skill costs its own side; `replied` is what the units it hurts cost it back, and
  the two are subtracted side by side in `rate`. Before it, `price.go` did not
  mention `passive.Replies` at all, so attacking a `venom_blood` or a `thorns`
  holder was free — the same blind spot `friendlyFire` exists to close, pointed
  the other way. It reads `Battle.answer` rather than a second copy of it, and
  every decision that function makes is a decision here: **once per cast, never
  per strike** (a reply answers a *use*, so a three-strike skill pays one answer,
  and charging three declines it outright); **every unit the shape actually
  damages, whichever side they are on**, because `answer` never asks; not the
  caster itself; not a holder the blow kills; not a holder whose gate is shut. The
  damage is charged at face value clamped at what the caster has left and over
  **no horizon at all** — a reply is health taken off *now* — while the statuses
  go through `inflictedOn`, which already prices an inflicted status over the
  horizons above, rather than through a second copy of them. Both halves are
  weighted by the chance the attack connects, read off the one `combat.Hit`
  `Battle.hitAgainst` builds, which is the same Hit `against` weights the damage
  with.
  ⚠️ **A dead attacker is priced, and both halves of it.** A reply may kill —
  `Battle.reply` gives its damage no exemption for arriving out of turn — so the
  caster's health is spent down the walk the way `reply` spends it, a lethal
  answer is charged the turns the caster will now never take at `killHorizon`
  (exactly as `friendlyFire` charges an ally it kills), and **nothing after it is
  charged at all**, because `reply` returns before its own statuses land and
  `answer` returns before the next holder answers. Backwards in one direction the
  opponent is suicidal, in the other it is paralysed.
  ⚠️ **The charge is clamped at nought per answer, and that is the sign guard.**
  `rate` *subtracts* it, so a negative charge would make an attack **more**
  attractive for being answered — and "the opponent hunts the `venom_blood`
  holder" reads as a plausible strategy rather than as a sign pointing the wrong
  way. `TestAReplyRepelsRatherThanAttracts` asserts the direction from both sides
  of the board, because one side alone cannot tell a cost from a tie.
  ⚠️ **The gate is read on the holder as the board stands**, while `answer` reads
  it on the holder as the blow left it. That can only err one way, and it is a
  property of the type rather than a hope: `passive.Condition` carries nothing but
  `BelowHealth`, so a gate can turn *on* as its holder is hurt and can never turn
  off — this therefore misses a trait the attack itself wakes up and over-charges
  for none, which is the direction every cap in this file errs in, and it costs no
  hypothetical unit to be exact about.
  **Measured, and it is a wash — reported rather than papered over.** Head to head
  against the same rating with the term added back (`forge.Bout`, 10,000 seeds,
  control exactly 500‰): **499‰, band ±8‰, median 48 turns**. Inside the band, so
  not a finding. Against the frozen ruler it reads **779‰ (band ±8‰, median 45)**
  where the term-less rating reads **780‰** on the same seeds — the standing
  figure, reproduced exactly, which is what says the comparison is honest. The
  reason is the **data** and not the term: the shipped roster's only answering
  trait is `venom_blood` on `ally.venusaur`, a poison at a **4%** chance, so the
  charge is a rounding error and the term changes a decision in **22 of 500**
  shipped battles. Given every unit a shipped reply instead, all three are still
  a wash: `thorns` 497‰, `ballast` 499‰, `venom_blood` on all six 500‰ exactly —
  not one decision moved, because a charge both aims carry cancels.
  **And the term does win, on a board where a reply is worth something.** The same
  head to head with every unit answering off its defence: at 400‰ **503‰** (median
  38 turns against 48), at 800‰ **513‰ — clear of the ±8‰ band**, median 34, and
  at 200‰ *plus a certain poison* **607‰**, median 34. So the null on the shipped
  roster is a reading of the **data** and not of the term: the shipped replies are 80‰ and 50‰ of defence and a 4% poison, which is
  too small to change a decision worth changing. It is kept for the reason
  `friendlyFire` was — a rating that cannot see a cost prefers the option carrying
  it — with the honest note that it buys nothing today and starts paying the
  moment a reply worth authoring is authored.
- **Still out of scope:** *where* an extra turn falls in the order, as a **term**.
  As a tie-break it was built and thrown away for measuring a null (above); as a
  term it is what the rule forbids — a queue reading that priced how many turns
  something buys is tempo, and tempo is priced from the speed stat. The detonate
  setup needs neither: price the status and the payoff rates itself.

⚠️ **No balance figure carries across this change.** The shipped roster read 53.1%
ally before and 79.0% after, side-neutral (82.5% for the same squad with the sides
exchanged), 0 stalls, 4000 seeds. That is a cast finding — the roster's calibration
rested on the opponent not playing statuses. It has been re-levelled since, as a
separate data change: **Charmeleon 30 / Ivysaur 30, reading 49.1% over 20,000
seeds**. Any rate quoted anywhere in this file predating that pass is a reading of a
different instrument.

## Species: what a unit is

Shipped. An element says what a unit is made of and an archetype says how it
fights; `cast.Species` says **what it is** — a shell, roots, a lineage — and it is
deliberately the thinnest axis in the repository: an id, a word for a screen, an
optional note, and no stats, no kit and no rule of its own.

- **A fourth allowlist, not a fourth concept.** `skill.Restriction.Species` sits
  beside the others and composes the same way, so `dragon_rage` and
  `dragon_dance` moved off `characters: [pokemon.charmander]` — the wrong axis, on
  purpose, for as long as there was no right one — and are now carryable by *every*
  dragon. A character declares `species` as a list because a unit may be several
  things at once; Charmander is a `lizard` and a `dragon`.
- **Nothing in `battle` branches on it**, exactly as nothing branches on an
  archetype. It is a carry rule settled while a character is authored plus a word
  a browser prints — which is what keeps it cheap, and why `scenarios.golden` and
  `replay.golden` did not move when it landed.
- **An empty list is a real answer**, not a hole: "nothing in particular" is what
  most of a cast is, and it is what a lineage skill refuses. The one place that
  reading is relaxed is `forge.Carrier`, where an empty list is a question nobody
  has reached yet, so a lineage skill picked *before* a species is settled is
  refused at the write rather than at the keystroke. That is documented on the
  field rather than fixed, because a half-filled form cannot tell the two apart.
- `⚠️` **Every axis has to be read back into the draft.** `forge.SkillAnswers`
  held three allowlists, and species arrived without the fourth line — so every
  balance edit to a lineage skill silently rewrote it as free to anybody, with the
  file still loading and every carrier still carrying.
  `TestEveryShippedSkillTakesABalanceEdit` now compares the restriction across the
  edit, which is the general form of that check rather than a species-shaped one.

## Origins: which story a skill is out of

Shipped. `restrict.origins` is the fifth allowlist and the **broadest**: a
species says what a holder must be and an origin says which fiction it must come
out of, so `rasengan` is closed to every Pokémon and open to every Naruto
character, including the ones nobody has authored yet. It is a list because a
crossover is a thing that happens, and because every other axis here is one.

- **The axis exists because `characters` was the wrong one.** A work outlives
  every character in it, so gating `rasengan` on `naruto.naruto` would say "only
  this one may carry it" and would have to be edited the next time that work lent
  the cast anybody — the same argument `species` makes about a lineage.
- **Enforced in `cast.resolveCharacter` against `declared.Origin`**, beside the
  species check, and brought forward to a half-filled form by
  `forge.CheckSkill`. ⚠️ On that form the origin is a **chooser**, so unlike the
  element and the species list it is answered from the moment the form opens —
  there is no keystroke at which it is unanswered, and a skill kept for another
  work is correctly greyed out straight away. A test that read "nothing answered
  ⇒ nothing unavailable" as covering every axis was wrong about this one.
- **Nearly the whole book is gated now**, which inverts a reading the golden
  report was built on: "free to anybody" used to be the majority and is now six
  skills. That list is `internal/seed`'s `sharedPool`, **written down with a
  reason each**, and `TestEverySkillSaysWhichWorkItIsFrom` fails a skill that
  declares neither. The gate is an allowlist, so an omitted one is silent — a
  Naruto skill authored without it is carried by a Pokémon and the book still
  loads — and that test is the only thing standing between the rule and the
  omission. **A pattern-based exemption would not do**: "every neutral skill" would
  have exempted `rasengan` on a technicality.
- ⚠️ **A column missing from the golden report is a restriction the design
  record cannot show.** Origins arrived and thirty-two rows appeared in
  `skills.golden`'s "who may carry" table with nothing but dashes across them,
  because the report had four columns and the book had five.

## The inert element, and what a character with no type actually buys

Shipped with `pokemon.mew`, the first character to declare `neutral`. The chart
has listed it as inert since the beginning and nothing had ever stood on it, so
this is a value the engine already supported being used rather than a feature.

- **It is a real affinity, not an absence.** `element.Single(Neutral)` builds,
  `Chart.ValidateAffinity` accepts, and `MultiplierAgainst` reads a flat thousand
  in both directions against every shipped affinity —
  `TestMewNeitherGainsNorLosesAgainstAnyShippedAffinity`, which also refuses to
  pass unless *some* other pairing in the cast moves, or it would be measuring a
  chart with no edges rather than an element with none.
  ⚠️ `Dual` still refuses to pair with it, and rightly: an inert second element
  adds no line to the multiplier. It adds a line to the *kit*, which is the next
  bullet, and that is a different question.
- **What it actually costs is the kit, and the cost is the identity.** A unit may
  carry a skill of an element it shares or a neutral one, so an inert character
  can carry **only** the neutral skills — which is the largest single pool in the
  book and, by construction, nobody's signature: everybody's plain moves and no
  element's line. That inverts into the widest learnset in the cast (23 skills
  against Magnemite's 17), and it is why Mew's own five had to be authored neutral
  and gated by `restrict.species` rather than by an element.
- ⚠️ **It does not flatten a matchup profile, and the expectation that it would
  was wrong.** Mew's spar rates run from nought to a hundred across the cast,
  exactly as polarised as every other character's — every shipped character spans
  roughly ninety points. What decides a pairing here is tempo and sustain; the
  chart is a term inside a strike, not the shape of a fight. **Removing the
  elemental term removes the elemental term.**

### A one-form line, and a stat line that is a median rather than a share

- **The two mythics are the first shipped characters that do not evolve**, and
  nothing in the engine moved for them: `progression.Line` has always resolved a
  single form
  (`TestALineWithOneFormStillResolves` predates this by a long way). A level still
  means what it meant — the curve runs base to max, the learnset still opens over
  it — so what a one-form line gives up is the threshold and the fork, not
  progression. `TestTheCastHasALineThatDoesNotEvolve` holds each of those and
  that the pair is both of them and nobody else — a third one-form line would be a
  decision rather than an accident, and this is where it would be noticed.
- ⚠️ **"The same share of every ceiling" is not the same idea as "the middle of
  the cast", and the first one is a trap.** The first draft put Mew at seventy per
  cent of all six declared ceilings, which reads like the fiction's "a hundred in
  everything". The cast uses the ceilings very unevenly — attack tops out at
  nineteen twentieths of its ceiling and **dodge at less than half of its** — so an
  even share handed Mew the cast's best dodge by half again and its best speed
  outright. It sparred **72.4%**. The midpoint of what the cast actually fields,
  axis by axis, reads **56.4%** with hp and attack *higher* than the first draft.
  `TestMewHoldsNoExtremeOnAnyStat` is the claim, and it is stated against the
  shipped top forms rather than against `progression.json`.

### ⚠️ A one-turn control status cannot be a setup for a skill on a cooldown

The most useful thing measured here, and it is about the engine rather than about
Mew. `dream_eater` reads a condition — hit harder into a stunned target — and the
character carrying it is also the one applying the stun. That looks like a two-card
combo and is not one.

`stun` lasts a single turn and the turn it lasts is the **target's**, which the
target then spends being skipped. So the whole window in which the condition holds
is one slot of the turn queue, and who owns that slot is decided by speed alone.
Sixty duels each, same kit, same stun source:

| opponent | speed | stuns landed | amplified |
|---|---:|---:|---:|
| Blastoise | 85 | 239 | **50** |
| Venusaur | 100 | 66 | 2 |
| Sennin | 134 | 77 | 2 |
| Charizard | 140 | 31 | **0** |

Two things follow, and both were measured rather than reasoned:

- **Raising the chance makes it worse.** Swapping the source for `hypnosis`, which
  lands more than twice as many stuns, took the conversion *down* — 9381 stuns to
  6 amplifications against Blastoise. The turn that lays the status down is the
  same turn that could have spent it, so a kit that stuns more attacks less and
  reaches the window less often. That is why `hypnosis` is not among the four the
  character brings by default; it lives in a build.
- **Shortening the skill's cooldown moves almost nothing** (3 → 2 → 1 changed the
  Charizard column not at all), which is the tell that the bound is the queue and
  not the availability.

So an amplifier on a duration-one control is a **speed check**, and should be
priced as a rare doubling rather than as a combo. A condition meant to be set up
on purpose wants a status that outlives the setter's next turn — `expose`, `mire`
and `weaken` all last two or three — and `venoshock`/`dragon_drive` are the shipped
skills that do it that way. `TestAOneTurnSetupIsAQueueRaceRatherThanACombo` holds
the direction rather than the figures.

### The other half of the pair: `dark`, `pierce`, and what a stat line is for

`pokemon.mewtwo` is the first character to carry **dark**, which is the other half
of the chart's only mutual pair. Light had a carrier — Cleffa — and dark had
none, so for as long as that was true Cleffa's element was, in play,
indistinguishable from an inert one: strong against nobody, weak to nobody.
`TestTheMutualPairFinallyHasBothHalves` is the claim that the pair is now
*fielded* rather than declared, and it checks both directions, because both ways
is what mutual means and what a cycle never gives.

- **What Mewtwo is for is `pierce`**, an axis three shipped skills touched and no
  character was about. ⚠️ **The obvious measurement of it cannot say anything**:
  three attackers of different elements against three targets of different armour
  is three different chart readings, and the elemental term swamps the one being
  read. What works is **one attacker carrying two skills** — `psystrike` pierces
  800 and `body_slam` pierces nothing — thrown at the same targets, so the
  elemental term is a constant that divides out. Across the whole armour range the
  cast fields (Blastoise 640 down to Magnezone 340), armour costs the piercing
  blow **175 per mille** and the plain one **381**.
- **`mewtwo.origin` carries Mew's own four on Mewtwo's frame**, which makes it the
  only reading in the package that puts one loadout on two bodies — and therefore
  the only way to say what the stat line actually changed. ⚠️ **It changed it for
  the worse on every column that kit is about**: fewer turns, less healing (a
  restore reads attack, and the clone's is lower) and less damage. A stat line is
  not "more"; it is **pointed**, and Mewtwo's points at skills that pierce —
  `mewtwo.breach` on the same body out-damages it better than two to one.
  `TestTheSameLoadoutIsWorseOnTheClone` holds all four of those.
- **`mythic` is the first species in the book with two members.** Every other one
  gates for exactly one character, so this is the first time the axis does the
  thing it was added for: five skills authored for Mew are carried by both,
  and `TestTheCloneKnowsWhatTheOriginalKnowsExceptHowToBeSomethingElse` refuses a
  species gate that one of its two members declines — that is a character gate
  written the long way. The exception is `transform`, gated on the character, so
  the one thing the clone cannot do is be anything else.

### ⚠️ At the top of the cast, speed is the dominant currency and armour is nearly free

Tuning Mewtwo took it from **98.5%** to **68.0%**, and almost none of that came
from where it was expected. Measured one dial at a time, everything else held:

| dial | reading |
|---|---|
| `psystrike` pierce 700 → 400 → 0 | 98.4 → 97.0 → **90.1** |
| speed 150 → 130 → 110 → 90 | 98.4 → 91.4 → 79.7 → **69.3** |
| attack 740 → 620 → 520 → 440 | 98.4 → 89.7 → 79.4 → **71.3** |
| hp 3200→2900 **and** defence 300→240 together | 98.4 → **96.5** |

So the eight-hundred-per-mille signature is worth about eight points, thinning
the frame past the thinnest in the cast is worth two, and **sixty points of speed
is worth twenty-nine**. A glass cannon is not paying for its glass: the enemy
needs turns to spend the opening, and a fast unit does not give them.

⚠️ **This is not the finding `swiftness` failed on and does not contradict it.**
That one is about *trait-sized* deltas — `+150` read 59.0% where `+50` read 74.0%,
which is noise wearing an ordering — and it still stands. What is ordered cleanly
here is a **sixty-point** range, four readings, monotone. A duel rate can separate
speeds that are far apart and cannot separate speeds that are close, which is the
same fact as "the queue is discrete and lumpy" seen from the other end.

The practical consequence for authoring: **the EffHP budget cannot hold a
character down.** It bounds survivability and nothing else, and survivability is
the cheap half. Mewtwo passes the budget at 5800 of 11500 — the thinnest line in
the cast — while reading 98.5%. Tune the speed and the attack; the budget will not
tell you.

## A strip pointed outward: what `dispelled` was missing, and how it was found

Shipped with `pokemon.gastly`, the first character to carry a strip aimed at an
**enemy**. `battle.Suggest` has had `dispelled` since strips were priced at all,
and it had never once fired on shipped data — both existing strips (`rapid_spin`,
`rinse`) point at the caster's own side and go through `cleansed`.

⚠️ **There are three things a strip can take and it priced one.** A stat buff
moves a number, so `dispelled`'s hypothetical reads it for free. A **shield** and
a **regeneration** move no stat at all — so both came back nought, and an
opponent handed a dispel declined it in favour of a ten-power poke. Measured
before anything was written: an actor holding a strip and a `jab` chose the `jab`
against an enemy carrying three block charges, three regeneration stacks, and both
at once.

The two terms added are the **inverses of the ones that price putting each on**,
read from those functions rather than written again — `unguarded` is `shielded`
backwards (charges times `strikeThreat`, clamped at `guardHorizon` for the same
reason), and `undone` runs the denied ticks through `worthHealing`, the same three
clamps a heal is priced by.

- ⚠️ **A regeneration on a unit at full health is worth nothing to take**, because
  the healing it owes cannot be banked. That falls out of `worthHealing`'s room
  clamp rather than being written, and it is a row in the test rather than a
  footnote.
- ⚠️ **`undone` refuses to fire when the strip names a harmful category.**
  `Pending` totals every stack's frozen tick without asking whether it heals or
  hurts, so a strip naming `dot` would be taking a poison *off* an enemy and the
  same difference would report that as a gain. Nothing shipped does that; the
  guard is there because the arithmetic cannot tell.
- **Nothing moved.** The whole suite, every golden, every measured rate: unchanged
  by the fix, because no shipped skill reached the arm. That is the tell that a
  priced branch was dead rather than wrong.

### ⚠️ Two rows of that test passed for the wrong reason, and only a mutation said so

`TestADispelIsPricedForEachOfTheThreeThingsAStripCanTake` was first written with
every enemy at half health. Deleting the shield term left it **green**: on a hurt
target the rating prefers the strip anyway, for a reason that is not the shield.
The block row now reads a target at **full** health, which is the only version
that goes red when the term it exists to hold is removed.

The same trap caught the build test one file over. Two Gengar kits made *identical*
and told apart only by their traits still read 1300 damage against 1178 and 153
statuses against 81 — `contagion` weakens what it hits and a weakened enemy takes
longer to win — so `miasma.dealt > unbind.dealt` passed on two copies of one build.
The margin asked for is now the one the two kits actually produce, plus a
disjointness check. **A separation test needs a margin taken from the mutation, not
from the reading.**

### A dispel cannot be measured by a duel either

`spite` takes a shield and a regeneration off an enemy, and neither is a thing a
lone unit brings to a duel — so `hexforge spar` reads it as a 700-power attack and
says nothing, exactly the blind spot that made Cleffa read 0 to 7 per mille. The
instrument is the mender's: one striker, one wall, and the slot under test, fought
both ways round. Three opponents differing only in their wall's fourth skill, two
Gengars differing only in `spite` against `bite`, 300 battles a row:

| opponent wall | kit | strips | my blows blocked | the wall healed |
|---|---|---:|---:|---:|
| shields (`withdraw`) | with `spite` | 508 | **1541** | 270722 |
| | without | 0 | **2434** | 276988 |
| only hits | with `spite` | **0** | 0 | 0 |
| | without | 0 | 0 | 0 |
| regenerates (`aqua_ring`) | with `spite` | 589 | 0 | **297579** |
| | without | 0 | 0 | **899834** |

⚠️ **The win rate is not what is being read** — it moves 500 to 540 per mille
against the shielding wall, because a shield is not the whole of a fight. What a
squad says exactly is what the skill *did*: a third of the blows the block would
have eaten no longer are, and the regeneration gives back barely a third.

⚠️ **The middle row is what makes the other two mean anything.** Against a squad
with nothing to take, the skill strips nought and is a plain attack, and the rate
goes very slightly the *other* way. A utility skill worth its slot everywhere
would not be a utility skill.

⚠️ **A restore is not a regeneration.** `withdraw` heals on the turn it is cast
and cannot be taken back, which is why the shielding wall's two healing figures
are level while the regenerating wall's differ by two thirds. Only the ongoing
tick is strippable.

### ⚠️ What a dispel may not name, and why `buff` is not in it

`status.Set.Cleanse` does not skip permanent statuses, and only a **gated** trait
(`While != nil`) re-applies its grant — `Battle.hold` runs once at enlistment for
everything else. So a strip naming `buff` would permanently take `toughened` off
an `endurance` holder, `evasive` off an `elusive` one, `quickened` off `swiftness`,
and never give them back. That is a whole design decision about whether a trait can
be removed, not a side effect to ship inside a character, so `spite` names `shield`
and `regen` and nothing else. Do not widen it without deciding that question first.

## Two shapes of guard: `absorb` beside `shield`, and the blow that ignores both

Shipped. `status.Absorb` is the tenth category and the second shape a guard comes
in. It is **not** a smoother `Shield`, and the difference is the whole reason it
exists:

| | `shield` (`block`) | `absorb` (`aegis`) |
|---|---|---|
| what a stack is | a **charge**, counted | a **pool**, in health |
| what it stops | one **strike**, whole | that much **damage** |
| against one heavy blow | cancels it entirely | eats what it can, the rest spills |
| against a split volley | one charge a strike | the same total, same drain |

So a wall of charges is the answer to a single heavy blow and multi-strike is the
answer to the wall — which is `warden`'s identity and `bruiser`'s counter to it —
while a pool is **indifferent to how the damage arrives**. Measured on equal
totals: against two charges a `heavy` dealt **0** and a three-strike `triple`
dealt **100**; against a barrier both dealt **100**.
`TestABarrierEatsDamageWhereAShieldEatsStrikes` holds exactly that, as the two
guards against each other rather than either against nothing.

Three things follow and each is a test:

- **A strike a pool ate still CONNECTED.** It arrived and was paid for; only its
  damage went elsewhere. So the riders it carries land, its drains drain what
  actually reached health, and `Count(attempts, Struck)` counts it — none of which
  is true of a blocked strike, which never happened. That is the complement of
  `OutlastsAShield`, not a case of it.
- **A pool spills.** A blow bigger than what is left is not cancelled; the
  remainder goes through, which a charge can never do.
- **An unblockable blow leaves the guard standing.** It goes *past* a barrier
  rather than through it, so the next attacker meets a full one. Spending what it
  ignores would make one skill quietly the strongest strip in the game.

### Where each piece lives, and why the split is not arbitrary

- ⚠️ **`Stack.Pool` is its own field, not `TickAmount`.** It is the only per-stack
  figure in the package that goes **down**. `TickAmount` is frozen and multiplied
  by the turns left — that is what `Tick` charges for and `Pending` totals — so
  sharing one field would have a barrier tick its holder for its own strength and
  `Pending` price a guard as though every point were owed once a turn. The same
  argument gives `Kind.PoolPower` its own field: the parser refuses a `tick_power`
  on anything that does not tick, and relaxing that for one category would lose
  the guarantee for every other.
- ⚠️ **`combat.Absorb` is a pass over the outcomes, not a branch inside `Roll`.**
  A charge is spent *during* the roll because it decides whether a strike happens;
  a pool is spent *after*, because it only reduces what landed and the critical
  has to be rolled before there is a figure to eat. Two mechanics, two moments.
  The pass touches no randomness and walks the attempts in the order `Roll`
  produced them, so `combat` stays a pure function of its arguments and a battle
  with a barrier on the board still replays from its seed. It also meant `Roll`'s
  signature never moved — one production caller and eighteen test callers left
  alone.
- **`Set.SpendPool` drains oldest stack first**, and the order is the replay
  contract rather than a preference — entries sit in a slice in application order
  for the same reason a tick resolves in one.
- **`skill.Unblockable` is a switch where `Pierce` is a ratio**, and that is not
  an inconsistency: defence is continuous, so a ratio walks an armoured unit's
  edge instead of jumping to the end of it, while a guard is **discrete**. "Half a
  block charge" is not a quantity this engine has. It is spelled in `battle` as
  *there is nothing to spend* — the charges are set to nought and the pool pass is
  skipped — so `combat.Roll` needs no flag at all.

### ⚠️ The pricing bug this shipped with, and the test that caught it

`guarded` first computed the pool and **multiplied it by the stack count**. That
rated putting a barrier on a unit that already had one, every turn, forever — the
same defect the "two units buffing each other" term exists to stop. The fix is the
one `shielded` and `standing` already use: build the hypothetical through
`Set.With`, so the cap, the refresh and the wasted stack are the ones `Apply`
resolves, and read the *difference*.

It was caught because the test asked the rating twice — once with nothing up and
once **at the cap** — and only the second row failed. A pricing test with one row
is a pricing test that cannot see a missing clamp.

And the reason the test exists at all is the previous PR: `dispelled` sat in the
rating for a long time pricing a shield at nothing, so an opponent handed a dispel
declined it. **A guard nobody rates is data.**

### What shipped on top of it

`aegis` (lá chắn), pool 1800 per mille of the caster's attack a stack, 2 stacks, 3
turns. `light_screen` puts it on an **ally** — Cleffa's, because a barrier a
mender hands somebody else is the half `withdraw` cannot do. `shadow_punch` is
Gengar's and is the counter to every form of mitigation at once: `pierce: 1000`,
which `Pierced` resolves to exactly nought defence, plus `unblockable`.

⚠️ **`pierce: 1000` is the switch `Pierced`'s own comment argues against**, and
shipping one is a deliberate exception rather than an oversight. The argument
stands — a switch makes an armoured unit worthless against one skill and
unaffected by the next — so the skill that takes it is paid for elsewhere: 900
power on a four-turn cooldown, which is a third of what `cross_chop` does per
turn. It is the answer to armour that costs you, not a default.

⚠️ **Neither new skill is in a build**, deliberately. A build is a measured claim
about four skills and a trait; these two are new tools in a learnset, and whether
either earns a slot is a separate question with its own measurement. Measured
firing in real battles first, though — 60 squad battles put 354 barriers up and
soaked 128,329 damage across 446 strikes — because a skill nothing casts is the
failure mode this section already has one example of.

## Three new axes: a grant with a number, a share past defence, a price in health

Shipped together because the first two are the wall and the third is what a
character pays to break one.

### A grant that carries a quantity

`passive.Grant` was `{Status, Stacks}` and every grant before this was a
**switch**: `toughened` is a status whose modifiers say what it does, so granting
it needs nothing but its name. A barrier is not a switch — it is a pool, and how
deep the pool is is the whole of what the trait says.

- ⚠️ **`Set.Hold` applied a nought amount**, on the argument that a permanent
  status can be neither a damage-over-time nor a regeneration and so has nothing
  to snapshot. That was true of every permanent status there was, and stopped
  being true the moment a guard could be permanent. A barrier granted that way
  parses, applies, shows in the log and **stops nothing at all**.
- ⚠️ **The permanent-absorb refusal was the wrong rule in the wrong place.**
  `status.ParseBook` refused it as "a guard with no clock", and that is wrong
  about a pool: a pool has a clock of its own, which is the damage in it. What is
  actually dangerous is a **gate** — `hold` and `release` run a grant again every
  time one reopens, so a barrier behind `below_health` comes back full each time
  its holder crosses the line. The rule now lives on `passive.ParseBook`, where
  the gate can be seen.
- **The pool is read off the holder's BASE line.** Grants go on one trait after
  another at enlistment, so a buffed reading would make the answer depend on which
  trait was applied first — a unit with `endurance` and a defence-scaled barrier
  would get a different barrier in one declaration order than in another, with
  nothing on either trait saying so.
- The stat comes through `skill.ParseScaling`, which is the sentence that function
  is exported under, and it brings the health refusal with it.

### ⚠️ A pool is worth most where the frame is thinnest, which makes one of the two traits self-defeating

A pool is a flat quantity, so what it is worth is a share of what its holder could
already take. Attack does not correlate with survivability, so an attack-scaled
barrier lands wherever it lands. **Defence does** — so a defence-scaled one hands
the deepest pool to the unit that needs it least, and on a wall it competes with a
permanent defence stat that is worth more precisely because the defence is already
high.

| carrier | trait | spar | the lead trait it replaces |
|---|---|---|---|
| Charizard | `projection` (attack) | **71.9%** | `blaze` 63.2% |
| Blastoise | `carapace` (defence) | **29.2%** | `endurance` 32.1% |

Both ship: a weaker option is still an option. What is written down is the
*reason*, so a later balance pass does not read the figures as a bug.

### Converting is not a bigger pierce

`passive.Converts` sends a share of every blow its holder throws past defence
entirely. It sits beside `Drains` for that field's own reason: a skill's drain
belongs to the skill and a trait's belongs to the unit.

**Piercing lowers the divisor** — armour gets smaller and the blow is still
divided by what is left, so what gets through keeps falling as armour rises.
**Converting splits the blow** — the converted share is divided by nothing, so it
arrives whole however deep the armour is and puts a floor under the blow that
armour cannot lower.

| defence | pierced 400‰ | converted 400‰ |
|---|---:|---:|
| 0 | 600 | 600 |
| 300 | 375 | 420 |
| 900 | 214 | 330 |
| 2400 | 103 | **280** |

So one is the answer to armour in general and the other is the answer to a **wall**
— which is what makes it the damage dealer's tool against a tank rather than a
second helping of the same thing. Measured on the shipped cast, Machamp with
`rending` against Blastoise: **150-0 over 30 turns**, against `blood_thirst`'s
138-12 over **71**. And it costs where there is nothing to bypass — against
Charizard it wins ten fewer duels than the drain it replaces.

- ⚠️ **Both halves are computed unfloored and the floor is applied to the sum.**
  Two `damage` calls would floor twice, handing a converting attacker a free
  minimum-damage point that grows with nothing.
- ⚠️ **A conversion of nought takes a separate branch** rather than falling
  through at a share of a thousand, which truncates differently. That is what let
  this ship without moving a single figure in a single golden —
  `TestConvertingNothingChangesNothing`.

### A price in health

`skill.Cost` is a share of the caster's **maximum** health, paid **up front** and
**whether or not anything lands**. A share taken out of the damage dealt would be
free on a turn the skill missed, and a skill that is free when it fails has no
decision in it. Of maximum rather than current, so the price is the same on the
first turn and the last.

⚠️ **It may never take the last point.** `Suggest` prices what a turn is worth to
the board and has no term at all for "and then I am not here", so a cast that
could be lethal would be rated as pure gain. The floor keeps the whole question
out of the rating.

#### ⚠️ The bug it shipped with: a cost filed in a branch that does not run

A health cost is a cost of *acting*, so it went into `friendlyFire` — which is
where every other cost of acting lives. But `rate` calls that arm only when
`declared.Target == skill.All`. So a single-target skill charging a quarter of its
caster's health was charged **nothing**: measured, a Magnezone holding one cast it
three times a battle, handed over seven tenths of itself and lost **120 of 120**
duels, against 69-51 for the same kit without it. It is subtracted in `rate` now,
for every skill.

#### ⚠️ And the clamp that made it cheapest when it was most fatal

The price was first read through the same expression that pays it — the floor
included — so the skill got *cheaper* exactly as casting it got more dangerous: a
unit at two hundred was charged a hundred and ninety nine instead of seven
hundred and fifty, and the rating therefore liked it best in the one position
where it should refuse.

`spentHealth` is now the one figure in `price.go` deliberately **not** the one the
resolving function pays. The rule the file is written under is about arithmetic
that could drift; this is the same arithmetic with one clamp left off, and that
clamp is *what a unit can afford* rather than *what the skill costs*. A unit that
cannot afford it should decline.

With both fixed, `steel_beam` is cast when the board is worth it and not
otherwise — against Blastoise (defence 640) 30 casts in 120 duels, against
Charizard (400) **129**.

## Pricing a summon, so the opponent casts one

Shipped. `battle.Suggest` rates a summoning skill in the one unit it counts in —
damage — as **the damage the copies would deal over the turns they are given**.
Before it, a summon had no power of its own, so it reached `Suggest` only as the
*fallback*: the option taken when nothing at all could be hurt. The shipped
summoner therefore never called anybody up while it had a kunai in reach.

- **A summon is the only thing in the book that buys turns rather than spending
  one**, so the turns are the price: `summonWorth` puts a hypothetical copy in
  the cell `summonPlaces` would give it, at the line `summonStats` would give it,
  with the elements `summonAffinity` would give it, and multiplies its best
  single-turn attack by the horizon. All four are the functions that do the real
  thing — ⚠️ **two of them were extracted for this**, because a rating built from
  its own reading of "where would the copies stand" is a rating that pays for a
  copy the board has no room for.
- ⚠️ **The horizon is capped, at `summonHorizon`.** The honest horizon for a
  summon that *stays* is the rest of the battle, which this rating cannot see and
  which would put such a skill above every attack in the book for ever. So a
  summon is priced for its own `lasts` when that is shorter and for the cap when
  it is longer or absent. `summon_toad` is the case that needs it: no `lasts` at
  all. The cap direction is deliberate — over-pricing costs a kill, under-pricing
  costs a cast that was marginal anyway.
- **It is an upper bound and says so**: the turns are what the skill promises,
  not what the board will grant. A copy can be killed on arrival, and a `bound`
  one leaves when its summoner does. A shallow rating knows neither.
- ⚠️ **A cast worth nothing falls through to the fallback rather than scoring
  nought.** A rating of nought is still a rating — it beats "no damaging option
  at all" — so on a full board it would take the turn ahead of a shield that
  would have done something.
- ⚠️ **No golden moved and that is not "no effect".** Naruto is cast-only, and
  nothing in the roster summons, so `scenarios.golden` and `replay.golden` cannot
  see this. What moved is the **balance answer**, at 2000 seeds a slot:

  | | before | after |
  |---|---|---|
  | naruto.naruto overall | 56.4% | **93.8%** |
  | · vs bulbasaur | 0.0% | 81.5% |
  | · vs charmander | 69.5% | 100.0% |
  | pokemon.bulbasaur overall | 66.6% | **39.5%** |
  | naruto mirror, turns | 36 | 303 |

  The summoner was **losing every single battle to Bulbasaur** and now wins 81.5%
  of them. Nothing about the character changed; the opponent started using its
  kit. So the shipped numbers were never measuring the character that ships, and
  **Naruto now needs a retune** — that is a data PR, not this one.
- The mirror's slot skew flipped from **+27.1% to −71.8%** and its battles got
  eight times longer. Not a limit being hit (4000, and no draws) — two summoners
  regenerating bodies at each other. Worth a look before the retune.

## Amplifying a status, which really is two features

Shipped. A trait may declare `amplifies`, per status, with two optional shares:
`effect` raises the damage-over-time tick and `chance` raises the roll that
decides whether the status lands. Either alone is a legal trait, because they are
worth different things — a tick pays over the life of a stack, a chance pays per
cast.

- **Both sides meet at one site and compose by multiplying.** `battle.amplify`
  raises the chance, `battle.resist` lowers it, both inside `inflict`, so the
  order they are applied in cannot matter and neither has to know the other
  exists. The cost is stated rather than hidden: resistances stacking *diminish*
  for free, amplifiers stacking *compound*.
- ⚠️ **`amplify` reads the unit that is *acting*; `resist`, a few lines above it,
  reads the unit being acted on.** Every other job a trait has is about its
  holder. The two take the same Go type, so passing the wrong one compiles and
  silently hands a target its attacker's amplifier —
  `TestAnAmplifierReadsTheUnitThatIsActing` is the only thing standing between
  that and a release.
- **The clamp is last, and only last.** A probability cannot exceed one, so a
  composed chance is clamped to `scale.Base` — but clamping *before* the target's
  share would make the order matter (a certainty amplified then halved is 500,
  halved then amplified is 600). Compose, then clamp.
- **The tick is amplified where it is frozen**, in the one multiplication
  `inflict` hands `Set.Apply`, so nothing later has to know the trait existed —
  which matters because a `Stack` deliberately does not remember who applied it.
- **The shares reach the event, and the record.** `AmplifiedChance` and
  `AmplifiedEffect` sit beside `Refused` on `status_applied` and
  `status_resisted`, and the replay record prints them (and the frozen tick) when
  there is one. This was the third time the same trap came up after `Pierce` and
  `Refused`, and the first time it was noticed before shipping rather than after.
- ⚠️ **A regeneration cannot have its effect amplified, and the reason is a bug
  older than this field.** A regen declares `tick_power` and heals from a frozen
  amount exactly as a poison damages from one, but `battle.inflict` computes a
  tick only for a `Dot` — so **an applied regeneration freezes nought and heals
  nothing**. `ingrain` and `aqua_ring` both self-apply `regrowth`; nothing has
  noticed because `battle.Suggest` never casts a non-damaging skill. The parse
  refuses `effect` on anything but a `Dot` rather than promising a multiplication
  of zero. Fixing the regen is its own change, and it moves balance.
- **It composes with a reply for free, and the shipped data proves it.**
  `venom_blood` answers an attacker with poison and `virulence` amplifies poison;
  both sit on the same Bulbasaur, and a reply inflicts through the same
  `battle.inflict` with the holder as the actor — so the reply's poison is
  amplified too, 25 per mille becoming 30 in the record. Nothing was written to
  make that happen, which is the argument for `origin` and for one inflict path
  rather than two.
- **Vulnerability** — the mirror where a target is *easier* to affect — is built
  and has **no shipped user**. It is `Resists` with a negative share and reuses
  the whole composition; the decision it needed was made (`Refused` is signed, and
  the early return in `resist` no longer treats "nothing refused" as "nothing to
  do"). What is missing is a trait the share is right for.
  - It was trialled on `reckless`, which is its natural first user, at −200‰
    against each of the six harmful statuses that can actually be inflicted, and
    **it made the build worse**: the dragon-vs-fire duel fell 22.0% → 16.1% on the
    vulnerability alone. Not shipped. → `README.md` § *What `bare`'s dodge clause
    was worth*.
  - ⚠️ **A vulnerability costs more than its arithmetic, because the opponent
    steers into it.** `pricing.landed` calls `fight.resist` on the *target*, so
    the rating sees that a unit invites a status and aims one at it on purpose. A
    negative share is not a multiplier applied to whatever would have happened
    anyway — it changes what the opponent chooses to do, so a share-times-uptime
    calculation under-prices it and a reading that looks too harsh is probably
    right. Do not sanity-check one against paper and conclude the harness broke.
  - The rendering half is proven: `BlurbTraitVulnerable` gets its own sentence
    rather than the resistance one with a minus in it, and a −200‰ share prints as
    `20%` in both languages — "Tăng 20% khả năng dính bỏng" / "Takes 20% more of
    any burn aimed at it" — with the sign carried by the verb.

## Pricing a detonate, so the burst is paid for

A detonate is `requires` with `consume: true`: the skill is amplified while the
target carries a status, and eats it on the way through. The rule is that it may
beat leaving the status alone, but **not by more than a factor of two** —
otherwise applying a status and immediately spending it is the only line worth
playing, and the status is a cast animation.

- **A detonate is only as big as what it spends.** The ceiling is set by what
  consuming the status throws away, so the fuel decides the skill, not the other
  way round. `burn` throws away **548**; `expose` throws away **102**. That is why
  `inferno` bursts for 1200 and `dragon_drive` for 788, and why a line whose only
  status is a stat debuff cannot have a big detonate without first being given a
  status worth detonating.
- ⚠️ **There are two currencies and the arithmetic is different.** A
  damage-over-time is worth its remaining **ticks**, which land whatever the
  attacker does next. A stat debuff ticks for nothing and is worth the **extra
  damage ordinary attacks land while it is up** — priced by hitting the lowered
  defence and the real one with the same plain attack and charging the difference
  for every turn the debuff had left. `forgoneBy` in `skills_test.go` is the one
  place that knows both, and the golden table names which one each skill spends.
- ⚠️ **The tick-only reading was shipped and was a wrong answer, not a missing
  feature.** `tick power × stacks × duration` is **nought** for a stat debuff, so
  a detonate off one was priced as giving up *nothing* and a burst of any size
  would have passed. It went unnoticed because every detonate was off a DoT.
  Both the test and the table now **refuse** a status they can price in neither
  currency, rather than returning nought — nought and "gives up nothing" are the
  same number and only one of them is true.
- **The consume happens before the damage is computed**, in `resolveAgainst`:
  `target.Statuses.Consume` runs, *then* `b.Stats(target)` is read. So a detonate
  off `expose` hits into the defence it just handed back — it pays for its own
  burst, including against itself. Nothing had to be written for that; it is the
  ordering, and it is worth not disturbing.
- **A detonate is a mechanism, so assert it as one.** A win rate cannot see
  whether the combo is played: `TestTheDragonLineCanSpendWhatItApplies` counts
  `StatusApplied`, `Amplified` and `StatusConsumed` off the log and requires the
  last two to be **equal**. A burst that amplifies without consuming is the
  strictly-better skill this whole rule exists to refuse, and no golden would
  notice.

## A consume can be paid in shape: `charge`, the counter category

`charge` is a status category that does **nothing** to its holder — no stat, no
tick, no turn. It is ammunition, and it is the ninth category, appended last for
the reason `taunt` and `heal_cut` were. Full reasoning in `README.md` §
*Counting instead of doing*; what matters when editing:

- **It is the one category `Book.MaxStacks` does not bound.** `max_counter_stacks`
  (999) bounds it instead, and a charge carrying a modifier or a tick is refused
  at parse — the ceiling was granted on the understanding that it multiplies
  nothing. Do not "tidy" that into one cap; the two numbers bound different
  things.
- **`reserve` is the same category on the caster's own side**, and the two are
  told apart by `Harmful` alone: a charge goes on an enemy so a cleanse strips it
  (`rinse` names it), a reserve is its holder's fuel so only an enemy's dispel
  does. `Category.Counter()` is what the cap and the modifier refusal ask, so
  appending a third counter means adding it there and nowhere else. Spent through
  `self_requires`, and a reserve is priced FLAT up to `pricing.capacity` — the
  deepest single spend the holder's kit owns — rather than halved like a charge,
  because a reserve spender cashes the whole run at once. Measured: with the
  halving the shipped loop banked 456 stacks over forty duels and spent none.
- **`self_requires` gained `stack_power`**, the caster-side answer to `arc_power`:
  power per stack consumed, so "spend everything" has arithmetic in it. Bounded by
  `skill.MaxSpendPower` applied in `Condition.Takes` — on the STACKS, never on the
  power, so the leftovers stay in the tank; clamping the bonus instead is the bug
  `Skill.Cost` shipped one field along. `Skill.SelfCeiling` is the reading a bound
  must use, because `Satisfying` is the cheapest case and a scaling payment is
  worth most at the deepest.
- **`requires` gained `applies`**, riders a target's condition pays out only
  where it holds. It is the condition's third currency — a bonus buys damage, a
  restore buys health, and these buy a status the skill does not otherwise
  inflict — and it is what lets a skill say *hit harder, and also weaken, but
  only against something already rotting*. They go out through the same
  `inflict`, the same roll and the same shield filter as the skill's own
  `applies`, because a rider surviving a block on a different rule from its
  neighbour would be a difference no reader could find on either.
  ⚠️ **The target's condition only.** The caster's own reads the caster, so a
  rider there would land on whoever cast it, which `self_applies` already writes
  without a condition standing in front of it.
  ⚠️ **The reading is taken before the consume, in `resolveAgainst`, and carried
  down to the rider block.** Asking again where the riders are paid would ask a
  set the consume above may already have emptied.
  ⚠️ **`Condition` stopped being comparable** the moment it carried a slice, so
  the round-trip tests that asserted `*back.SelfRequires != Condition{…}` compare
  through `reflect.DeepEqual` now. Whole-value still, on purpose: a field-by-field
  check stops covering the next field somebody adds.
- **`self_requires` also gained `stack_restore`**, the health twin of
  `stack_power`: health per stack consumed, in permille of the caster's scaling
  stat, which is the unit `restores` already counts in. It is what lets a reserve
  pay for a heal instead of a blow, and with a base `restores` of nought the gate
  falls out of the arithmetic — a caster holding no fuel heals nothing without a
  cast gate being written anywhere. Bounded by `skill.MaxSpendRestore` in
  `Condition.Takes`, on the STACKS for every word of the reason above, and
  `Skill.SelfRestoreCeiling` is its reading of a bound.
  ⚠️ **A condition may hold one rate or the other, never both** — two ceilings
  cannot answer one `Takes`, so `resolveCondition` refuses the pair rather than
  leaving `Takes` to choose.
  ⚠️ **Do not widen `Condition.Scales` to cover it.** `describe.go` branches on
  `Scales` to print what a stack adds *to the blow*, so a heal-only spender would
  print that clause with a power of nought — wrong prose, and no test would fail.
  `ScalesRestore` is a separate predicate for exactly that reason.
  ⚠️ **The figure is handed to `Battle.restore`, not read there.** `Act` spends
  before it pays out, so a reading taken inside `restore` asks an emptied tank and
  answers nought on every cast, silently. It travels on `swing`, the reading taken
  once per use — which is also why it cannot be read per target.
  ⚠️ **`pricing.selfSpendable` skipped every skill of no power**, which is the
  shape a reserve-paid heal has, so a unit whose only spender was a heal valued
  its whole tank at nought: it never banked, cashing in cost nothing, and a dispel
  against it was free. The guard belongs on the damage ARM, not on the search —
  a share of a cast is meaningless when the cast is not a blow.
- ⚠️ **`Battle.spend` called `Set.Consume` for as long as the field existed**, so
  `consume_stacks` on a caster's own condition parsed, round-tripped and was
  thrown away — the whole pile went whatever the author asked for. Nothing noticed
  because nothing shipped ever spent anything of its own.
- **Never permanent.** `Set.Remove` refuses a permanent status (that is what keeps
  a trait from being dispelled) and a consume goes through `Remove`, so a
  permanent counter is one nothing could ever spend.
- **`requires` gained `chains`, `damped`, `arc_power` and `consume_stacks`.** A
  condition carrying `arc_power` is a **conduit** and is a different animal from a
  detonate: it damps its own blow (`damped`) and fires the stored charge instead
  (`arc_power`), one stack **per strike**, along a chain of adjacent carriers. The
  old refusal "consumes for no bonus" is now "for neither a bonus nor a discharge",
  and a skill may take one payment, never both.
- ⚠️ **`charge` outlasts a shield, and is the SECOND category ever to.** The
  predicate's own comment warns against completing it, and that warning is about a
  *stat* getting through (mire broke the mirror invariant); a counter carries no
  stat. `TestOnlyContaminationOutlastsAShield` is a written-out table rather than a
  rule, because what Dot and Charge share is a sentence and no predicate spells it.
- ⚠️ **The arc is not the skill's damage and must not be routed through the skill's
  machinery.** Not aimed, not rolled against accuracy or dodge, **not blocked** —
  it is what was already sitting on the target. It is logged as `Damaged` with
  `Status` set, the way a reply is logged with `Passive` set, so a renderer that
  switches on kind still draws the health it took.
- ⚠️ **A miss delivers nothing, a block delivers everything.** `discharge` runs on
  every strike outcome except `Missed` — the same sentence `OutlastsAShield` is
  written on. Do not "tidy" that into one condition.
- ⚠️ **The chain stops at the midline**, and it is the one shape in the engine
  that had to be told. Every other is a pattern, and `pattern.Targets` already
  drops a splash cell on the far side — `Side.CrossesSides` is the only thing that
  lifts it. A chain reads the board instead, so it obeyed none of that: aimed at an
  enemy standing next to a charged teammate it walked straight back over the line,
  for 272 damage and two of that teammate's stacks. The *enemy's* own chargers
  arrange that for free. `TestTheChainStopsAtTheMidline`.
- **The chain is `Battle.chainFrom`: BFS from the aim over `NeighborsOnBoard`,
  through carriers only.** It is re-walked **every strike**, because it shrinks as
  it burns; and it returns nothing at all when the *aimed* unit is clean, which is
  the gate on the entire mechanism. Deterministic — the map is asked only for
  membership, and no map iteration reaches the result.
- ⚠️ **`hex` adjacency is not what a reader guesses.** `3,1`'s neighbours are
  `4,2 4,1 3,0 2,1 2,2 3,2` — **`4,0` is two cells away**, so a chain reaches it
  only through a charged `3,0`. Odd columns sit half a cell lower; check with
  `Offset.NeighborsOnBoard` rather than by eye.
- **A rolled strike count**: `repeat_chance` + `max_strikes` on the skill,
  `Hit.RollStrikes` in combat. ⚠️ **Wire BOTH into `combat.Hit` when building it in
  `resolveAgainst`** — the first cut set neither and every roll silently returned
  the floor, which no test caught because the floor is a legal count.
  `ExpectedStrikes` is the mean and is what everything outside the roll reads;
  the description quotes floor/odds/cap instead, because the mean describes no
  cast anybody will have.
- ⚠️⚠️ **A conduit's own figure is a CONSTANT — the arc is added, never traded
  for.** There was a `damped` field that cut the blow to pay for the discharge; it
  is gone, and the reasoning behind it was the mistake. A stack does not appear on
  a target by itself: the turn that put it there is the price, paid before the
  conduit was cast. A conduit is bought with **tempo** — charging turns and its own
  cooldown. Do not reintroduce the trade; the user rejected it twice.
- **One bound, carried on the skill's own face**: `arc_power < power`. A stack may
  top the blow up, never outweigh it — otherwise the skill is a delivery mechanism
  for somebody else's turn and its printed figure stops describing it.
- **`consume_stacks` has two useful values and no third.** 1 is a drip (`spark`,
  `electro_ball`); 0 is a **nuke** (`overload`) — it takes the whole pile and its
  arc multiplies by the count. A nuke may not `chain`, must need ≥ 2 stacks, and
  must sit on a longer cooldown than the drip.
- ⚠️⚠️ **The drip/nuke rule has been written both ways round and both were wrong**,
  because the answer depends on what the skills give up and that kept changing.
  While conduits damped, a nuke damped *and* waited and was owed the better rate
  (held to a poorer one it dealt 472 at a pile of six — a skill nobody brings). With
  nothing damping, a nuke's compensation for waiting is that it takes the whole
  pile, which is already the larger purchase — so **no better rate than the drip**,
  and a longer cooldown. `TestANukeGetsNoBetterRateThanADrip`.
- ⚠️ **`chainFrom` returns the head alone when `chains` is false.** Returning nil
  there is the bug that leaves a non-chaining conduit firing no charge at all.
- ⚠️ **`TestADetonateIsWorthLessThanItsBreakEven` cannot price a counter and skips
  one by category.** Its arithmetic is what leaving the status alone was worth, and
  a counter is worth neither ticks nor a defence term. The skip is on
  `kind.Category == status.Charge`, **not** on the shape of the payment.
  `TestAConduitPaysForWhatItDischarges` bounds it instead: the arc must beat the
  damped power and not beat it twice over.
- The playstyle is held by `TestAccumulatingIsAWayOfFightingRatherThanASlowerOne`
  — the damage must arrive in **more, smaller** pieces *and* at a rate within
  three fifths of the burst kit's. Shipped reading: 363‰ over 11398 blows of 97
  against 366‰ over 3114 of 366. Four skills charge (thunder_shock 1, thunderbolt
  2 as riders on damage; charge_beam 2, magnetise 3 as turns spent) and two spend
  — a conduit that bought every stack with a turn could never fire twice running.

## Pricing one number: `hexforge weigh`, and why a roster win rate could not

`hexforge weigh <character> <skill> --field F --values a,b,c` prices **one field
on one skill** by fighting the carrier against a copy of itself whose only
difference is that field. `forge.WeighReport` is the measurement, `internal/forge/weigh.go`;
`cmd/hexforge/weigh.go` only draws it. **It authors no balance data** — the
variant lives in memory, `skill.Book.Append` marshals and re-parses it, and
`internal/seed/data` never moves.

⚠️ **The roster win rate does not price anything, and the control experiment is
why.** Over all 20,000 seeds, post-#136 roster, baseline **47.55%** ally:

| change | ally rate | Δ |
|---|---:|---:|
| `razor_leaf` crit 200 (ally lands it 3.2:1) | 45.41% | **−2.14pp** |
| `sludge_bomb` crit 200 (**ally-only**, enemy never casts it) | 45.15% | **−2.40pp** |
| `sludge_bomb` power **+5%, no crit at all** | 45.80% | **−1.75pp** |

The last row is the control: **giving the ally more damage *lowers* its win
rate**, by about as much as the crit did, with no crit involved. The roster rate
is **non-monotone in ally damage** — a dial that is sometimes worth less the more
of it you have is not priced by that dial's reading. The same change also read
**+2.40pp before #136 and −2.14pp after**: the sign flipped on an unrelated
*placement* change. The engine was never the problem — the mechanism measured
**20.46%** against **20.00%** declared over 3,670 landing strikes.

**What a weigh figure is.** A price in **parts per thousand**, on a named
carrier, at a named level, against a copy of itself, with that carrier's kit and
stats and placement all cancelled. It is **not a win rate**, it says nothing
about the roster, and **it does not carry across a data change** — a placement
move can reverse its sign, which is exactly what happened above. Quote it with
the carrier and the level attached or do not quote it.

⚠️ **Swap the KITS, not the sides.** `forge.duel` already does. The queue breaks
a tie by enlistment, so leaving the roster order alone enlists the first-written
kit first in *both* halves and a unit against an identical copy of itself reads
**58.8%** — see the `self_gradient` entry above, where that trap was first paid
for. Reusing `duel` is the whole reason a weighing can claim its two halves
cancel. A `Damaged` event carries `Actor` and **no `Side` at all**, so the strike
tally follows the kit swap too: the challenger is `first` in one half and
`second` in the other, and folding on the wrong id reports the *opponent's*
strikes with nothing on screen looking wrong.
`TestAMatchupCountsTheChallengersStrikesAndNotTheOpponents` is what holds it.

⚠️ **The control is a row of every report, not a test somewhere else.** The
sweep always inserts the skill's own declared value, and that row **must read
exactly 500‰** or the whole report is refused before a figure is printed. A test
proves the code was right once; this proves the twenty thousand battles just
fought were the twenty thousand intended — a variant leaked into both kits, a
side read backwards or a perturbed rng each produce rows that look exactly like
good rows and a control that is not even.

⚠️ **`turns` is a headline column, read beside `worth`, not a footnote.** A
mirror-duel rate once failed to even *order* speed amounts (`swiftness`, above:
`+150` read 59.0% while `+50` read 74.0%) because the turn queue is discrete, and
the fix was measuring inside one battle. Damage is lumpy at the **kill
threshold** for the same reason — a little more of it buys whether the last
strike lands this turn or next. Median turns-to-finish is the inside-one-battle
reading: more damage kills sooner, monotonically, with no win/lose boundary to be
lumpy at. **A row whose `worth` is inside the band but whose `turns` moved is a
real effect**, and the footer says so. Monotonicity is reported for the two
columns **separately**, because they can disagree and the disagreement is the
finding.

**The band.** `2σ = 1000/√N` permille over `N = 2 × seeds` battles, integer
square root and ceiling division — no float anywhere, permille throughout, and
never zero. The default of **10,000 seeds** (20,000 battles, ~7.5s, band
**±0.8pp**) comes from that arithmetic and not from taste: the effects above were
1.75–2.40pp, two to three times the band.

**Refusals, each because the row would otherwise print as an ordinary number.**
Seeds below one; a level outside `1..LevelCap`; an unknown character or skill; a
skill the carrier does not bring at that level and form (it would be *measured* —
an even row priced at nought on a skill nobody cast); a variant id the book
already declares; a value the parser refuses (returned in **the parser's own
words, unwrapped**, because every bound is `skill.resolve`'s and a second copy
would be free to disagree); a row on which the skill **did none of its own work**
— worth nothing means *not rated*, never *rated at nought*, which is the
`fire_fang` ally-0 lesson made permanent, and *its own work* is whichever
mechanism the row's own variant declares (striking, applying, restoring,
cleansing, summoning), because a strike is the mechanism of a damaging skill and
the wrong evidence for one that deals none; a row where the mechanism never fired
(crit is the only field with a mark of its own on an event); a row whose
`Endless` exceeds a
fifth of its battles; and a row with a **saturated half** (`First.Rate() >= 990`
or `Second.Rate() <= 10`), which has no room left for the field to move.

⚠️ Only **scalars** are weighable, and there are nine: `power`, `accuracy`,
`pierce`, `crit`, `strikes`, `drains`, `cooldown`, `range` and `self_gradient`.
`applies`, `strips`, `summons`, `restriction`, `element`, `target` and `pattern`
are excluded because changing one **authors a different skill** rather than
moving a dial, and the deviation would be that other skill's worth.

⚠️ **`self_gradient` is the ninth, and the one whose *off* state is not a
number.** It is one number, not two — `skill.Gradient` has had exactly one field
since #132, "the top of the curve is not a choice" — so a sweep of it is a line
and `MonotoneWorth` keeps the meaning it has everywhere else here. What is
different is the bottom of that line: a skill declaring no gradient carries a
**nil pointer** where every other field is off at a nought a crit is legal at.
`of` reads that absence as nought, nil-safe the way `Gradient.Share` is, and
`set` hands the nought straight back to the parser, which refuses a share below
one. ⚠️ **So the field prices *how much* a gradient is worth and never *whether
to have one***: a sweep on a skill that declares none has a control row of nought
and the whole report is refused, in the parser's own words, before a battle — so
no report here has a row for "this skill with no gradient at all" and none may be
read as one. Mapping that nought back to nil inside `set` would buy the row and
would be this package holding a second copy of a bound `skill.resolve` owns,
which is the one thing `set` exists not to do.

⚠️ **Nothing shipped carries a gradient**, so the field ships measuring nothing
in the book: `comeback` is the only one declared and no character fields it, so
`hexforge weigh --carriers all comeback --field self_gradient` answers *"all 11
in the book were skipped"*. That is the state of the data, not a fault in the
field, and the shipped book is not to be edited to make a number appear. The
first reading was taken on a **fixture** carrier — the bench adept with
`desperate` in its fielded four — at level 60 over 10,000 seeds, band ±0.8pp,
against `desperate`'s declared 1000: **500 → −20.8%**, control **+0.0%**, **1500
→ +20.0%**, **2000 → +34.6%**, at 68 / 64 / 62 / 60 median turns, ordered in both
columns. It prices a fixture and therefore nothing in the game; what it says is
that the field measures, and that a gradient is worth a great deal to a carrier
that leans on one. ⚠️ The carrier is built by the test (`bringsTheGradient`,
beside `forkedTwin`) rather than added to the fixture cast: `desperate` in the
adept's kit displaces `purify`, and the adept fights in the goldens — measured at
**656 golden lines** moved, none of them a design decision.

⚠️ **Do not try to seed the two twins separately.** They share one `*rng.Source`,
so changing one side re-scrambles every draw after it — roll drift. It **biases
nothing** (both arrangements fight the same seeds) and fixing it would mean an
`internal/core` change to make a measurement prettier.

### Across the whole cast: `--carriers all`, and why there is no average

`hexforge weigh --carriers all <skill> --field F --values a,b,c` takes the same
price once per character whose **fielded kit** brings the skill and prints a
table. `forge.WeighCarriers` is the sweep, `internal/forge/carriers.go`;
`cmd/hexforge/carriers.go` only draws it. The character operand is dropped: the
carrier is discovered rather than named.

⚠️ **It has no headline figure and must never grow one.** A weighing is a price
against a copy of *the carrier* — that is the only reason it is a price, because
everything else has been made identical on both sides and cancelled — so two
rows were fought against **two different opponents** and are not two readings of
one quantity. An average of them has no opponent it was taken against and no
board it was taken on, which is the exact shape of number this repository has
twice been burnt by: the roster win rate (non-monotone in ally damage, sign
flipped by a placement change) and the mirror-duel speed reading (could not order
`swiftness` at all). **A carrier may be compared only to itself, at another
value, along its own row.** The footer says that in words and
`TestTheCarrierFooterRefusesTheCrossCarrierComparison` asserts the sentence is
still there.

**One row is one weighing**, not a second instrument: same control, same band,
same fold, and `TestOneCarrierWeighedAloneAndInTheTableAgreeRowForRow` holds the
row identical to `hexforge weigh` on that carrier alone. What the table adds is
who is in it and in what order:

- **Membership** is `forge.NotBroughtError` caught with `errors.As`, so exactly
  one place knows what "brings" means. A character that cannot bring the skill is
  **absent** rather than a row of noughts — the same distinction as refusing a
  row that landed nothing instead of pricing it at nought — and the footer counts
  the skips and says why.
- **Refusals are per row.** A refused carrier keeps its line with a dash where
  its figures would be, spelled out under the table, and the other rows stand.
  ⚠️ Except a **control that is not exactly 500‰**: it is marked `⚠️ HARNESS` and
  sorts above every priced row, because every other refusal says *this carrier is
  uninteresting* and that one says *the run leaked*. Drawn alike they would be
  the same dash in the same column.
- **A value is refused whole, and once.** It is a fact about the skill rather
  than about any carrier, so every variant is built before a die is rolled — one
  parser sentence, instead of the same sentence per row discovered after the
  first carrier's battles had been paid for.
- **Order** is worth at the largest swept value, descending, ties by character
  id, harness refusals first and other refusals last. It is stated in the footer
  and is never the order the cast was read in — `internal/core` bans a map
  iteration that reaches an output, and this is the same discipline one layer up.
- Each cell is **`worth/turns`**, because the two are co-equal columns here for
  the reason they are in `renderWeigh` and cannot be adjacent columns on a table
  this wide. A row whose own sweep is not monotone is marked `not ordered` on its
  own line: a dial that is not monotone is not priced, and on a table the figures
  are all a reader sees.

**Cost** is `2 × seeds × values × carriers`, printed above the table. `--seeds`
defaults to **2000** here (band **±1.6%**) rather than 10,000, because the table
multiplies — that is the count the first crit chances were actually authored at.
Re-take an interesting row on its own with `hexforge weigh` at the full ten
thousand.

⚠️ **No shipped character shares a skill with another**, so on
`internal/seed/data` today every `--carriers all` table has exactly one row and a
footer of three skips. That is the tool working rather than failing: the skips
are the part that was previously noticed by eye.

**Left at its default, deliberately:** a second field per row. That is a surface,
and a surface is not something a column of figures can report — `MonotoneWorth`
has one answer per row of a grid, one per column and none for the grid.
(`self_gradient` is no longer on this list: it is the ninth field, above. A
`--carriers all` on the book's only gradient is still an empty table, because no
character brings it.)

### What the first crit chances cost, and what the readings taught

The first two were authored off `weigh`, one carrier each, 2000 seeds, band
±1.6%, a chance of 200‰ throughout:

| carrier | skill | worth |
|---|---|---|
| `pokemon.bulbasaur` | `razor_leaf` | **+8.4%** |
| `naruto.naruto` | `wind_shuriken` | **+6.9%** |
| `pokemon.charmander` | `fire_fang` | +1.0% — inside the band |
| `pokemon.squirtle` | `bite` | +0.2% |
| `naruto.naruto` | `kunai` | **−1.7%** |

The first two shipped and the last three did not.

⚠️ **A theme is not a price.** The rule these were picked under first was that a
crit belongs to a blow that strikes a point rather than to a spray, and the
measurement disagreed: `bite` and `kunai` are as pointed as `razor_leaf` is and
are worth nothing and less than nothing. What the price actually reads is the
**carrier's whole line** — `razor_leaf` and `wind_shuriken` are what their
carriers spend their turns on, `bite` sits in a squirtle mirror that grinds for
141 turns where damage barely decides, and `kunai` is cheap enough that making it
better gets it **cast more often**, crowding out the skill that should have been
cast. That last one is `outrage` wearing a different hat: a rating that likes a
skill more will reach for it more, and *more often* is not the same as *better*.

⚠️ `kunai` reading **negative** is the most useful row in the table, because it
is the proof that the instrument can see a buff make its carrier worse. A win
rate on the roster could not; that is what § *Pricing one number* is about.

The roster moved **47.56% → 49.73% ally** over 20,000 seeds afterwards, and
**8.02%** of all damaging strikes are now critical against nought before. Both
are recorded as observations. The roster figure prices nothing — `razor_leaf` is
carried by a level 60 ace on one side and a level 30 unit on the other, so the
shift is mostly whose hand it is in — and the strike share is the mechanism
check, not a result.

**The rest of the cast crits at nothing, and that is authoring rather than a
backlog.** Twenty-six damaging skills carry no chance. They are added **one at a
time, each off its own `weigh` reading**, by whoever is authoring — there is no
sweep to run and no target number of skills to reach, so it was taken out of
`TODO.md` rather than left standing as a task nobody was going to pick up whole.
⚠️ What must not be lost with it is the rule above: **the price is a fact about
the carrier's whole line, not about the skill's shape**, so a crit may not be
authored from a theme. `bite` and `kunai` are as pointed a blow as `razor_leaf`
is and priced +0.2% and **−1.7%** — a cheap skill given a crit gets cast *more*
and crowds out a better one, which is the `outrage` lesson wearing a different
hat.

## Fighting two ratings: `forge.Bout`, and the exact control it rests on

A roster win rate cannot see a rating. Both sides use `Suggest`, so a change that
helps both leaves the rate where it was, and what moves is whichever squad's kit
had more to gain — which is why the tie-break was measured by hand with itself
switched on for one side at a time. `Bout` is that procedure made an instrument:
two `Brain`s, the shipped roster, and a share read from the challenger.

**The two arrangements.** Every seed is fought twice — challenger driving the ally
squad, then the two exchanged — over the same seeds `1..N`, and the challenger's
wins are counted over all `2N` battles.

⚠️ **The control is exact, not statistical, and is asserted at exactly 500‰.** When
both `Brain`s are the same, the two arrangements are the *same battle* — same seed,
same roster, same decisions — so they have the same winner and the challenger holds
the winning side in exactly one of the two. That is `N` wins and `N` losses **by
construction**; a draw lands in both arrangements and counts half to each side, so
it does not disturb it either. `Bout` runs that control before it measures anything
and **refuses to print a figure** unless it comes out exactly even. Never assert it
within a band: a band is precisely what a bookkeeping leak would hide behind.

**The board does not have to be symmetric**, because the roster's asymmetry is
fought from both ends and cancels exactly — that is the whole content of the
paragraph above. So a bout measures on the **shipped** roster, which is the board
every other figure in this file is quoted on. ⚠️ The ban on a mirror roster is
about a **data** change — the shipped cast may not be flattened into two copies of
itself to make a number easier to read — and does not apply here: nothing is
authored, the roster is a parameter, and the mirror is in the procedure.

⚠️ **`FirstUsable` is a ruler and may never be improved.** It is `Suggest` before
any pricing landed: first option in kit order that is available, aimed at the first
cell offered, nothing compared. Every claim about a rating is of the shape "beats
`FirstUsable` by X‰ over N seeds", and X is comparable across this repository's
history only while the thing it is measured against never moves. A sharper ruler
would silently re-scale every figure ever quoted against it and nothing would
report the disagreement. `TestTheRulerIsNotAnOpponent` pins it. If a second, harder
baseline is wanted, **add one beside it**.

**The figure.** The shipped rating against the ruler, on the shipped roster, over
**10,000 seeds (20,000 battles): 77.9%**, band ±0.8pp, median **45** turns. The
control on the same board and the same seeds is 500‰ exactly at a median of **48**
turns — so the priced rating both wins far more often *and* finishes sooner, which
is the second reading and the one a rate alone cannot give.

**It refuses**: seeds below one; a control that is not exactly even; and a run whose
`Endless` share passes `endlessShare`, because two better opponents stalling each
other is a real risk of any change to a rating and must not be filed as a draw.

⚠️ **Roll drift, accepted and not fixed.** The two arrangements share one
`*rng.Source` per battle, so the first decision the two ratings disagree on
re-scrambles every draw after it. It **biases nothing** — both arrangements fight
the same seeds — and fixing it would mean an `internal/core` change to make a
measurement prettier. Same note `Weigh` carries; see § *Pricing one number*.

`Bout` takes the roster as a **parameter**, so `internal/forge` keeps not importing
`internal/seed`: a tool that read the embedded copy would keep reporting on the cast
as it was at the last build.
