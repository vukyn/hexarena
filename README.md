# hexarena

A turn-based PvE team battler: two squads of up to five on a shared hex grid,
acting in an action-value turn order, resolving skills against an elemental
matchup chart.

The engine is deterministic. A battle is a pure function of its seed and the
decisions taken, so it replays exactly — which is what lets the terminal client
and a future graphical one be the same battle rendered twice rather than two
implementations of it.

```
go run ./cmd/hexarena --seed 11 --side ally    # play a side
go run ./cmd/hexarena --auto --seed 11         # watch both sides play themselves
go run ./cmd/hexforge                          # author the cast: see the subcommands
go run ./cmd/hexforge spar pokemon.squirtle    # ...and fight one against the whole cast
go run ./cmd/hexforge-tui                      # the same authoring, full screen, in Vietnamese
go run ./cmd/hexforge-tui --lang en            # ...or in English; ctrl+l swaps them mid-session
go test ./...
```

The full-screen authoring client is built on bubbletea v2, with bubbles and
lipgloss beside it. Everything else needs only the standard library so far,
which is how it worked out rather than a constraint being kept.

## The battlefield

One shared odd-q offset hex grid, six columns by three rows. Each team authors
its own three by three formation — column 0 is its backline, column 2 its
frontline — and fills at most five of the nine slots.

```
 c0  c1  c2  c3  c4  c5
 BK  MD  FR  FR  MD  BK
 __    __    __
/  \__/A2\__/  \__
\__/  \__/E3\__/  \
/A5\__/A1\__/E4\__/
\__/A4\__/E1\__/E5\
/  \__/A3\__/  \__/
\__/  \__/E2\__/  \
   \__/  \__/  \__/
```

The enemy formation is placed by rotating it 180 degrees about the centre of the
board. Without that rotation the two halves would not mirror: odd columns sit
half a cell lower, so identically-authored slots would have different distance
profiles and the matchup would be quietly unbalanced.

Range is hex distance on that shared grid, which gives it a legible ladder:

| range | reaches |
| --- | --- |
| 1 | the opposing frontline |
| 2 | its middle column |
| 3 | its backline |
| 5 | anywhere on the board |

A hex has six neighbours and no diagonal to argue about, which is why the grid is
hex rather than square. On a square grid with Chebyshev distance the three rows
stop mattering as soon as the columns are two apart; with Manhattan distance a
melee unit cannot reach the enemy standing diagonally in front of it.

## Turn order

Every unit waits `1_000_000 / speed` action value between its turns, and the unit
with the smallest pending timestamp acts. A window of exactly one million action
value therefore gives a unit as many turns as it has speed: a hundred speed is a
hundred turns per cycle, and doubling it doubles them.

The scale is a million rather than ten thousand because at ten thousand the
integer wait collapses — fifty of the speeds from two to two hundred become
indistinguishable from the speed one point below, and the collisions are worst at
the top of the range where the difference matters most.

A speed change keeps the fraction of the wait already served. Restarting it would
hand a speed buff a partial turn for free, and would let a unit be stalled
indefinitely by alternating a buff and a debuff on it.

## Damage

```
damage = scaling * skillPower * affinity * K / (K + defence)
```

Every multiplier is an integer in parts per thousand and the whole expression
resolves with one division at the end, so truncation happens once. `K` is the
defence at which a hit is halved.

Damage is a ratio against defence rather than a subtraction because
`attack - defence` produces a breakpoint where a small stat change flips a hit
between full damage and none, and that breakpoint moves every time the level cap
changes.

## Piercing

A skill may declare `pierce`, the share of the target's defence it ignores, in
parts per thousand. It is the answer to armour, which was the one defence in the
game with nothing on the other side of it: dodge answers accuracy, a guaranteed
hit answers dodge, block answers the guaranteed hit, multi-strike answers block,
cleanse answers a poison — and armour answered all of them.

**It is a ratio rather than a switch**, and that is the design rather than an
implementation detail. `progression.Limits` caps health and defence together
because they multiply, so two very different builds sit at the same bound:

| | health | defence | absorbs |
| --- | ---: | ---: | ---: |
| sentinel | 3100 | 800 | 11397 |
| bulwark | 4800 | 400 | 11214 |

Equal to within two percent. Piercing is what separates them, and how far it
separates them is the dial:

| pierced | sentinel | bulwark | bulwark's edge |
| ---: | ---: | ---: | ---: |
| 0 | 11397 | 11214 | 0.98x |
| 200 | 9717 | 9937 | 1.02x |
| 400 | 8072 | 8648 | 1.07x |
| 600 | 6418 | 7361 | 1.15x |
| 1000 | 3100 | 4800 | 1.55x |

A switch — true damage, defence ignored outright — offers only the last row,
which makes an armour unit worthless against one skill and untouched by the next
with nothing in between. That is the same reason buffs saturate here instead of
being clamped: a hard cap on a continuous quantity is the wrong shape, and this
engine has chosen against it everywhere else.

Three things follow, and each of them was a decision:

- **It reaches the skill's own strikes and nothing else.** A status the skill
  applies ticks against full defence, because a tick's damage is computed once
  when the stack is applied and frozen for the stack's whole life — so a pierced
  tick would be worth as many pierced hits as the stack has turns left. Measured
  on the shipped poison that is 400 a turn for three turns against 171, which is
  a different skill from the one the author wrote.
- **A pierced hit says so in the log.** The `damaged` event carries the share.
  Without it a reader holding the attacker's stats, the power and the multiplier
  could reproduce every damage figure in the game except this one, and a log its
  reader cannot reproduce is the log lying. The terminal client reads it out as
  *through 40% of the armour*.
- **The effective-health figure now describes one case out of two.** It measures
  damage that does not pierce, which is what the budget is checked against, so
  `hexforge` shows the other end beside it: what a stat line absorbs when its
  armour is ignored outright, which is exactly its health. The gap between the
  two figures is how much of a unit's durability it bought with armour rather
  than with health.

`razor_leaf` is the one skill that carries it, at 400, and what that buys is the
shape the ratio was chosen for — nothing against a bare target, half again as
much against the defence ceiling:

| target's defence | plain | pierced 400 | gain |
| ---: | ---: | ---: | ---: |
| 0 | 960 | 960 | — |
| 100 | 720 | 800 | 11% |
| 400 | 410 | 532 | 29% |
| 800 | 260 | 368 | 41% |

It does not warp the kit it sits in: across forty auto-battles `razor_leaf` goes
from 34 percent of all casts to 37, and the other three attacks keep their share.
Whether 400 is the *right* number could not be judged at all while the shipped
roster was a mirror — both squads carried the same kit, so piercing helped each
by exactly as much and the win rate moved by noise. The roster is no longer a
mirror (see *The seed roster is the balance instrument*), and the answer is
modest — and it has now been measured three times, in three different games, and
it has not held still:

| the game it was measured in | with the pierce | without it |
| --- | ---: | ---: |
| before anything answered back | 49.2% | 46.5% |
| once `venom_blood` answered attackers | 51.9% | 53.0% |
| once a placement brought four skills of nine | **49.5%** | **46.0%** |

Piercing helps whoever is attacking, so the middle row is what happened when
attacking started to cost something: the same size of move, the opposite sign.
The bottom row is what happened when the ally's Venusaur had to bring
`razor_leaf` as one of *four* skills rather than one of nine — a dial is worth
more to a kit that cannot dilute it.

So the figure is not a property of the skill. It is a property of the skill, the
opposing traits and the loadout together, and the loadout is now a **choice**.
The damage table was telling the truth in all three: this is a dial that changes
who armour is good against, not one that decides battles.

### Why raw health has no floor of its own

`MaxEffectiveHP` bounds durability against damage that does not pierce, so the
question was whether an armour-heavy line now needs a floor under its health to
stop one skill deleting it. Measured, it does not, and the reason is worth
keeping.

Among lines that actually saturate the joint bound, raw health runs from 3128
(at the 800 defence ceiling) to 4800 (at 400 or less, where the health ceiling
binds first). Fully pierced, the health-heavy line's edge over the armour-heavy
one is 1.53x. The worst elemental matchup in the game is a 2.25x swing, and that
is a swing the design already accepts — so piercing moves durability by less
than the element chart does.

The other half of the answer is that a floor has nowhere to live. Expressed as a
ratio — raw health must be at least so much of effective health — it is
algebraically `DefenseReduction(defence)`, which depends on defence alone: the
ratio floor *is* the defence ceiling, already in place at 800. Expressed as an
absolute minimum it cannot work at all, because `CheckTable` walks every level
from one and every unit has small health at level one.

So there is no new knob to add. If 1.53x ever proves too much, the thing to turn
is `ceilings.defense`.

## Elements

Eleven elements. Eight sit in two four-cycles joined by a cross eight-cycle,
which gives every one of them exactly two strengths and two weaknesses:

```
organic     water > fire > grass > ground > (water)
industrial  ice > metal > wind > electric > (ice)
cross       water > metal > grass > wind > fire > ice > ground > electric > (water)
```

Light and dark are a mutual pair, strong against each other and neutral against
everything else. Neutral is inert in both directions.

A unit may carry two elements, and a defender's two multipliers stack: a hit both
halves are weak to lands at roughly 2.25x, one both resist at roughly 0.44x, and a
weakness and a resistance cancel back to neutral. A pair whose members counter
each other is rejected, which leaves twenty-eight legal combinations — and, as it
turns out, exactly the ones with a natural reading.

An attack carries a single element, the one its skill declares. A second element
therefore buys a second line of skills rather than a better multiplier.

## Landing a hit

Four things stand between a skill and its target, and each answers a different
one of the others.

- **Accuracy** starts at the skill's own and closes towards a certain hit as the
  attacker's accuracy stat rises, without ever reaching it.
- **Dodge** reopens that gap towards a floor. It is applied after accuracy rather
  than subtracted from it, so it bites even against an attack that was going to
  land ninety-nine times in a hundred.
- **A skill declaring full accuracy** cannot miss and cannot be dodged. That is
  what leaves a block charge a job to do.
- **A block charge** cancels one strike that would otherwise have landed,
  guaranteed hits included, and is only spent on a strike that connects. One
  charge erases a heavy single blow and is burned by a sixth of a six-strike one.

Which is a rough triangle: dodge answers ordinary accuracy, a guaranteed hit
answers dodge, block answers the guaranteed hit, and multi-strike answers block.

## Timed effects

Damage over time, stat debuffs, control, buffs and shields are all one mechanism.
Block charges are a shield status whose stack count is the charge count, which is
how they gained an expiry without a second system.

Durations are counted in the holder's own turns and a status ticks at the start of
one, which keeps the total damage of a poison independent of speed: a hasted
victim takes its three ticks sooner, not more often.

Every stack keeps the damage it was applied with. Two attackers stacking one
poison must each contribute what their own attack was worth, and the unit that
applied a stack may be dead by the time it resolves.

A tick is never rolled and never offered to a block charge. That is the whole role
damage over time plays against those two defences, and cleanse is what answers it.

What each declared status actually does is a listing of its own — see *Looking a
status up*, which is where `?mire` at the prompt and `hexforge statuses` both go.

## Buffs and debuffs

Terms of the same target sum, then saturate towards a limit they never reach:

```
result = base + gap * delta / (gap + delta)
```

Against six hundred and twenty attack, two fifty percent buffs land at 1.74x
rather than the 2.25x that composing them would give, and four land at 2.17x
rather than 5.06x. Composing looks harmless on two and explodes on four, and a
battle where several supports stack on one carry is exactly where four happens. A
hard clamp would do the same job at the edges and nothing in between, which is
what makes stacking feel either free or worthless with no middle ground.

Because a large term saturates rather than overflowing, a debuff can safely be
authored well past a hundred percent, and a modest buff still lands close to its
face value.

## Healing

Three mechanisms give health back, and they are separate things rather than one
feature: a skill's `restores` heals whoever it targets, a skill's `drains`
returns a share of the damage it actually dealt to its caster, and a `regen`
status returns health at the start of its holder's turn — the mirror of a poison,
running on the same machinery with the sign the other way.

Four rules hold all three together:

- **Healing does not go through the defence curve, and damage over time does.**
  That asymmetry is deliberate. Defence turns away what is coming *at* a unit and
  has nothing to do with what is helping it, so a well-armoured unit is no harder
  to heal than a bare one. `combat.Rules.Restore` is where the division is
  absent on purpose, and adding it for symmetry's sake would make armour quietly
  reduce its own side's support.
- **A drain reads the damage dealt, never the damage rolled.** A strike that
  missed or was blocked drains nothing, which is what keeps a drain a reward for
  connecting rather than for swinging.
- **`status.Set.Tick` returns two unsigned totals, not one signed one.** Damage
  and healing come back separately. A negative travelling down the damage path
  would subtract a negative, and `wound` calls `kill` the moment health reaches
  zero — so a single signed total is the one shape in which a tick could bring a
  corpse back.
- **A dead unit is not healable and health clamps at `MaxHP`.** The first keeps a
  battle able to end; the second is what stops a regeneration from being a shield
  with no cap. Overheal spills into nothing.

Every restore emits a `healed` event. None of the other kinds explains health
going *up*, so without one a renderer would draw a number changing with no cause
in the log.

Two consequences worth naming. `battle.Suggest` never chooses a heal, because it
maximises expected damage and a heal has none — the same gap recorded under *A
deeper opponent*. And a unit that heals is durable beyond its stat line, so the
joint health-and-defence budget becomes an understatement rather than a bound.

## Passives

A skill is spent on a turn. A **passive** is not: it is in force from the moment
a unit is enlisted, and nothing the unit or its opponents do turns it off.

A passive **grants statuses**, and that is the whole mechanism. Nothing about a
stat change is reimplemented: the terms belong to the status, `modifier.Set`
saturates every term of the same target together, and a trait therefore saturates
*alongside* a temporary buff rather than composing with it. A passive that
composed would be the one place in this game where stacking explodes, and reusing
the status is what makes that unwritable.

Every status a passive grants must be declared **permanent** — a new flag on a
status kind, not a duration of nought, because nought would make an absent or
mistyped duration silently permanent. Permanent means four things, and each is a
place something else would have ended it:

- it never counts down, so it never appears in `Set.Tick`'s expiry list;
- a dispel, a cleanse and a detonate all reach `Set.Remove`, and it refuses them
  there — one guard for all three;
- it cannot be a damage-over-time or a regeneration, because a permanent one of
  either ticks for the whole battle with nothing able to stop it;
- and a renderer is told, so it draws *always* rather than the `0t` that reading
  the countdown alone would give.

The reason those matter is that a trait is granted **once**. A dispel that took
one off would turn it off for the rest of the battle with no way back, which is a
far larger effect than stripping a buff somebody cast a moment ago.

**A trait is in force before the first wait is computed.** A wait is
`1_000_000 / speed` and the queue is built while a unit is being enlisted, so a
trait touching speed goes on before that — not corrected afterwards, because the
first turn has already been served by the time anything would notice. The test
for it makes the holder the *slower* unit at its base and faster only with the
trait counted; an earlier version used two equal speeds and passed on the
tie-break whether the trait was applied first or not.

**And it says so in the log.** Each granted status is a `passive_held` event
beside the unit's own `started`, naming the trait and the status. It is its own
kind rather than a `status_applied` with an empty skill, because the two are
different facts: one is something a unit did to another and rolled a chance for,
the other is what a unit simply is.

Traits are declared in `passives.json` and checked against the status book at
load, the way a skill is checked against the pattern and status books. A character
names the ones it holds; an **archetype may suggest** some, which is where a
preset finally gains a mechanical weight — until now a preset never reached the
engine and nothing branched on one. `battle.Roster` carries them because the test
of what belongs on a roster entry has always been "does a replay read it", not "is
it small": an archetype and an evolution stage are settled *before* a battle and
leave nothing behind but numbers, while a trait is in force during one.

### A trait can refuse a status

A trait may also **resist** one, which is the only thing a passive does that had
no home already. A stat change reuses `status.Kind.Modifiers`, so the trait grants
and the existing code does the rest; nothing in the engine could say *no* to an
application, and the roll that decided one belonged entirely to the skill making
it.

```json
{ "id": "venom_blood", "grants": [], "resists": [{ "status": "poison", "amount": 1000 }] }
```

**The choke point is `battle.inflict`, not `status.Set.Apply`.** Apply is where
every status passes through, which makes it the obvious place and the wrong one:
it has no dice, so a resistance living there could only refuse outright. That is a
hard cap on a continuous quantity, which this engine has rejected everywhere —
buffs saturate, piercing is a share, dodge approaches a floor it never reaches.
`inflict` is where the chance is rolled, so a resistance there is a share of a
probability.

**A full thousand is the explicit way to write immunity**, and it is available
only to whoever authors it. Sources compose by multiplying what each lets through
— two resistances of six hundred leave sixteen percent rather than none — so
stacking diminishes for free, needs no saturation helper, and can never reach the
absolute. That is the same division as a skill declaring full accuracy: certain if
you write it, never if you stack towards it. A single resistance is exact, which is
the case worth being exact in.

**By status, not by category.** A category would be tidier — `Cleanse` takes
categories, for good reason — but it cannot say the thing that is wanted. "Immune
to poison" as a category is "immune to every damage-over-time", which hands the
holder immunity to burn as well. An id can name a class by listing it; a category
can never name one member of one. Only a **harmful** category may be resisted:
`Harmful()` is the existing split, and a trait refusing a buff would be refusing
its own side's help.

**And the log had to learn to tell two refusals apart.** `status_resisted` is
emitted whether the roll failed or the target refused it, so the event now carries
the share refused, and the renderer says which:

```
A1  E1 resists poison (40%)                    the roll failed
A1  E1 shrugs off poison (40% chance, 60% refused)
A1  E1 is immune to poison
```

Without that a reader is handed the word "resisted" and no way to tell a piece of
luck from a property of the unit — which is exactly the confusion the feature had
to be explained out of.

`venom_blood` is on Bulbasaur, which is canon and was one line — but only after the
roster stopped being a mirror. Against the mirror it took the whole poison layer
out of the shipped fight: 183 applications became 0, along with every
damage-over-time tick and every `venoshock` amplifier, because both sides were the
same poison-immune character. That was the roster and not the trait, and the
measurement now says so. Across forty auto-battles the layer survives — 83
applications become 60, 183 ticks become 133, 119 amplifiers become 104 — because
one Bulbasaur stands on each side and four other units do not care. The ally win
rate moved from 20 of 40 to 17, which is **noise and not a reading**: forty
battles cannot resolve three, and the counts above are what the change is judged
on.

### A trait can add to what its holder does, and can wait until it is hurt

Two more of the four things a passive is asked for, and neither needed a new rule.

```json
{
  "id": "cornered",
  "grants": [],
  "while": { "below_health": 500 },
  "applies": [{ "status": "poison", "chance": 500 }]
}
```

**What it adds goes through the same list a skill's own applications do.**
`skill.Application` is reused rather than copied, and the trait's riders are
inflicted by the same `inflict` — so they take the same roll, the same
resistance, and the same event. A second pass of their own would have been a
second place for all three to be got wrong.

**A rider goes on a damaging skill and no other.** `resolveAgainst` deliberately
never asks which side a target is on, so "already dealing damage to it" is the
available way to say the skill is hostile — and it is the honest one, since a
damaging skill aimed across the midline is already an attack on whoever is
standing there. Without the rule a cleanse would poison the ally it was healing.
The test for it had to be rewritten: it first used a *self*-aimed shield, which
returns before `resolveAgainst` is reached at all, so it passed with the rule
deleted.

**`while` is not `skill.Condition`, and that is the point.** A skill's condition
asks what the *target* is carrying, because it exists to pay a skill off for
arriving after a debuff. A trait wants to ask about its *holder*, and the question
it wants most is one no status can answer: how hurt am I. Bending one type to both
jobs is how two vocabularies grow inside one. So `while` carries a single term, a
**share** of maximum health rather than a number of points — a threshold in points
would be a different fraction of the bar at every level, so a trait authored for a
level-eight unit would be permanently on at sixty.

The share is read **live**, at the moment the rider or the resistance is asked
for, so a trait that only protects a hurt unit stops protecting it the moment it
is healed back. It is *at or under*: half means half counts. And a share is not a
fraction — `333` of 3000 health is 999, so "a third" written that way admits
everything strictly under a third and not a third exactly.

**A gate covers the whole trait** — the grants, the resistances, the riders and
the reply come and go together, and a trait wanting one gated half and one
ungated half is two traits. A gated `grants` was refused for a long time, because a grant is put
on when the unit is enlisted and the status it puts on is permanent precisely so
nothing can take it off; gating one needs the engine to hold and release that
status as health crosses the line, which is a mechanism rather than a term. It is
built: see *A gated grant: a stat change that comes and goes*.

The fifth job a trait can do — answering whatever attacked its holder — is the
one that fires on somebody else's turn, and it is written up under *Answering
back* rather than here because it needed a hook rather than a field.

### A trait comes in at a level

A character does not simply hold its traits; it holds each **from a level
onwards**:

```json
"passives": [{ "id": "endurance", "at_level": 16 }]
```

`cast.Unlock` is that entry, and it is deliberately about an *id* rather than
about a trait: the kit is the same question — declare many, unlock by
progression, bring some — so when skills gain their levels they gain this type
rather than a second one beside it. `UnlockedIDs` is the one function that answers
"what is in force at level N", and it takes the list rather than reading a
character, so the kit can use it unchanged.

Four things about the shape are decisions rather than details:

- **An unstated level is one**, the way an unstated strike count is. The common
  case is a trait a character has always had, and `"at_level": 1` on every entry
  would be noise on the line that matters least. A parse normalises the absent
  case to one, so there is exactly one value in memory meaning "from the start"
  and no caller has to know which spelling it is holding — which is why `Unlock`
  needs a `MarshalJSON` that omits a level of one rather than an `omitempty` tag.
- **There is a second gate, and it is a list rather than a threshold.** `stages`
  names the forms that may hold the entry, and an empty one is every form. A
  threshold — `at_stage` — would only ever have said "from this form onwards",
  which is a thing a level already says; a list can say "the bulb forms only",
  which nothing else can. See *Choosing to evolve*.
- **Both gates are applied in exactly one place**, `seed.ParseRoster`, because
  that is the only place a character, a level and a chosen form meet. A flat
  roster entry writes out its own traits and has neither a level nor an evolution
  line to be measured against; the engine is handed a resolved list exactly as it
  is handed a resolved stat line.
- **Bringing every unlocked trait is not a choice**, which is what makes this
  half cheap: nothing has to be recorded in the log. The slot that turns several
  unlocked traits into a decision is the expensive half, and it is where the log
  work is paid for — once, for the kit as well.

`hexforge show` prints every gate, because a character sheet is read all at once.
The browser prints a gate only while it is **still ahead** — `endurance@16` at
level 8, a bare `endurance` at 16 — so the mark reads as "not yet" and the row
changes as the level is walked, which is the one thing a level slider is for.

Bulbasaur's `endurance` comes in at 16, its Ivysaur stage. The shipped roster
fields it at levels 60, 24 and 8, so the youngest of the three goes without —
which is what a level being more than a number looks like.

`hexforge passives` lists what is declared and what each grants.

### Amplifying a status, which really is two features

A trait that "makes its poison better" was two different things, and both are
built. `amplifies` is a list on a trait, one entry per status, with two optional
shares:

```json
{ "id": "virulence", "name": "độc lực", "grants": [],
  "amplifies": [{ "status": "poison", "effect": 300, "chance": 200 }] }
```

- **`effect`** raises the tick. The tick is computed once, in `battle.inflict`,
  from the applier's scaling stat, and frozen on the stack for its whole life —
  so the amplifier folds into that one multiplication and nothing later has to
  know about it. That is what keeps a stack worth the same after its author has
  died, which matters because a `Stack` deliberately does not remember who
  applied it.
- **`chance`** raises the roll. That is the same site a resistance bites, so the
  two meet there: one side's trait raises the chance and the other's lowers it,
  both by multiplication, so the order cannot matter.

Either share alone is a legal trait, because they read differently in play — a
stronger tick is worth more the longer a stack lives, a better chance the more
often the skill is cast.

Three things it had to do, and the third is the one that usually gets skipped:

1. A field on `passive.Passive`, per status, the way `Resists` is. ✅
2. The multiplication — into the tick for the effect, into the chance for the
   chance. ✅
3. **The amplification reaches the event.** `AmplifiedChance` and
   `AmplifiedEffect` sit beside `Refused`, because `amount` already carried the
   frozen tick and `chance` the rolled figure, and neither is reproducible from
   the skill book alone. The replay record prints them, and the frozen tick with
   them, so the design record explains its own numbers rather than showing 480
   where the skill says 400. ✅

**This one reads the *applier's* traits at a site that reads the target's.**
`battle.resist` walks the target's passives; `battle.amplify` walks the actor's,
a few lines away and taking the same type. Passing the wrong unit compiles.

The clamp is worth knowing about: a probability cannot exceed one, so the
composed chance is clamped — after both sides, never between them, because
clamping first would make their order matter.

It composes with a reply without anything being written to make it: `venom_blood`
answers whatever attacked its holder with poison, `virulence` amplifies poison,
and both sit on the same Bulbasaur — so the reply's poison is amplified as well,
25 per mille reading as 30 in the record. A reply inflicts through the same
`inflict` with the holder as the actor, which is the whole argument for one path
rather than two.

Two limits, both deliberate:

- **A regeneration's effect cannot be amplified.** A regen ticks in every sense
  that matters to a data file, and it is refused anyway — but ⚠️ **not for the
  reason the refusal was first written**. That reason was that `battle.inflict`
  computed a tick only for a damage-over-time, so an applied regeneration froze
  nought and a share would have promised a multiplication of zero; *A
  regeneration that heals* fixed that and the argument expired with it. What
  keeps the refusal is the wording: `passive.Amplifies` reads *"its poison ticks
  30% harder"* in both languages, and a share that healed under that sentence
  would be a description that lies. Lifting it is a wording change first.
- **Vulnerability is not built.** A target that is *easier* to poison is
  `Resists` with a negative share, which reuses the whole composition rather than
  adding a field — but it needs a decision about a negative `Refused` in the log,
  and `resist` returns early when nothing is refused, which would silently drop
  it.

One thing already true and worth stating, because it limits what can be
attributed after the fact: a `Stack` remembers its frozen amount and nothing else.
It does not know who applied it, deliberately — the applier may be dead by the
time the stack resolves, so keeping the id would be keeping a pointer to something
that no longer exists. So `status_ticked` names the unit *taking* the damage, not
the one that caused it, and the only place the source is recorded is the
`status_applied` event. Two units poisoning the same target leave two stacks that
the state cannot tell apart.

## How a battle ends

Three endings, and the closing event names which one it was rather than leaving a
reader to work it out from what stopped happening.

| outcome | what it means |
| --- | --- |
| `victory` | one side is empty, and the closing event names the other |
| `annihilation` | both sides are empty, which a simultaneous kill can produce |
| `stalemate` | units are alive on both sides and nobody can act again |

The third is the one that needs explaining, and it exists because **nothing on
this board moves**. A unit occupies the slot it was placed in for the whole
battle, so its reach is settled the moment it is enlisted — and the enemies worth
reaching are not, because they die. Measured through `hex.Place`, the nearest
enemy is one cell from the front column, two from the middle and three from the
back, so a short-ranged unit standing behind the front stops being able to act at
all the moment the enemies near it fall. Two of them, one on each side, is a
battle that can never finish. It is not hypothetical: seed 18 once spent 3955 of
its 4000 turns skipped, every one a unit with nothing usable, and came back as a
battle that never ended rather than as a result.

A stalemate is declared when, for every living unit, **nothing timed is on it**
and **no skill it knows has a legal aim, cooldowns ignored**. Both halves are
asked pessimistically, because declaring a draw on a battle that would have
resolved is worse than letting the turn limit catch a real runaway:

- A skipped turn is ordinary. Turns are lost to control and to cooldowns
  constantly and those resolve, so neither is read here — which is exactly why an
  ordinary skipped turn cannot be mistaken for this.
- A poisoned deadlock is not a deadlock. The poison will kill somebody, and that
  ends the battle by emptying a side. So will a stun wearing off or a shield
  running out. Anything with a duration left to spend is a promise the board is
  not final; a permanent status a trait granted is not, and `status.Set.Timed`
  is where that distinction lives.
- It is a pure function of the state, not a count of quiet turns. A battle that
  draws on one machine has to draw on every other from the same seed, and a
  counter is one more thing two runs could disagree about.

The common case is caught earlier and more cheaply. `battle.New` refuses a roster
holding a unit that can aim at nobody from the slot it was given, and `hexforge
check` **warns** — rather than fails — when a character's longest range cannot
reach anybody from the column its archetype puts it in. A warning, because the
squad in front of it is what does the reaching and that is a design an author may
well mean. Neither is sufficient on its own, which is the point: reach shrinks as
units die, so a roster that starts fine can still end deadlocked.

The turn limit stays what it always was, a backstop. It is no longer standing in
for an outcome the engine could not express, so reaching it now means something
genuinely endless is happening rather than a draw waiting to be recognised.

## What a skill does, in words

The menu is a table of figures, which answers "how much" and not "what happens".
At the prompt, `?2` asks about the second skill offered, `?A1` asks what a unit is
carrying, and `?mire` or `?*` asks about a status — see *Looking a status up*.
Each prints and then draws the menu again, because reading is part of deciding and
cannot cost a turn.

```
 2) razor_leaf      grass      rng 3   arc_up        pow 1200   acc 880   x2 cd2
> ?2

  razor_leaf · phi diệp
  ----------------------------------------------
  Đánh 3 ô đối phương, 2 nhát, mỗi nhát 60% công (tổng 120%), xuyên 40% giáp.
  Tầm 3 · 88% trúng · hồi 2 lượt.
```

**Every figure in it is derived from the skill, and exactly one clause is not.**
`Skill.Flavour` is the opening — *vung dây leo quật kẻ địch từ xa* — and the
numbers are appended to it. Everything else stays computed, because an authored
figure drifts the moment the figure it describes moves: a line reading "doubles"
survives a bonus dropping from 1000 to 700 with nothing to catch it, and this
engine already refused that trade once in `Archetype.Demands`.

⚠️ **A flavour clause may not contain a digit, and `ParseBook` refuses one that
does.** That single rule is what makes authored prose safe here: a clause with no
number in it cannot be made wrong by changing a number. The check is for digits
rather than for a percent sign, because "110" and "gấp 2" are the same mistake in
different clothes.

⚠️ **A clause is bound by the same rule the name is.** `withdraw` was renamed from
*thu mai* to *thủ thế* because anybody may carry it and a shell-less creature
reading "thu mai" is nonsense — and the first clause written for it said *rụt hết
vào trong mai*, which is the same defect through a field the name test does not
look at. `TestAFreeSkillsFlavourNamesNoBodyItMayNotHave` holds it now, against a
hand-written list of body words, and a restricted skill is exempt because its
restriction guarantees the body: `ingrain` may say roots, since only a plant may
take it. The check reads the whole clause and cannot tell *whose* shell is meant,
so a body word about something else trips it too — that bluntness is the right way
round, because rewording costs a few words and a miss costs a sentence that reads
as nonsense on somebody's screen.

Without a clause a skill opens with the derived one — *Đánh đối phương, 110%
công* — which is what every description read like before, and is what a skill
still being authored in the tool reads like. What may not happen is a skill
*shipping* that way, which `TestEveryShippedSkillHasAFlavourClause` holds.

`testdata/describe.golden` holds every shipped skill's description, so a number
moving in `skills.json` moves a line there and the diff says how the change reads
to a player. It is the balance record from the other end: `skills.golden` says
what a skill is worth, this says what it sounds like.

**Shares rather than damage figures.** "100% of attack" is true wherever it is
read; a damage number is true for one caster against one target and stops being
true at the next buff. A player comparing two skills is comparing the shares.

**Both languages, and two places that ask for it.** The sentences live in
`internal/i18n` beside the gloss tables they borrow every data name from, which
is what makes a second vocabulary unwritable and a second language cheap. The
authoring tool needed that: `?` on its skill listing raises the same description,
and that tool has a language toggle a Vietnamese-only block would have ignored.

In English a bare id **is** the name, so `poison` in an English sentence is the
reading working rather than a gloss missing. English also needs singular wordings
where Vietnamese does not — "1 turn to recharge" against "1 turns" — and that is
two keys rather than a plural rule, because a rule would make Vietnamese pretend
it has a distinction it does not.

The battle prompt still asks for Vietnamese while the screen around it is
English. That is a stated cost: the screen gets translated in one piece or not at
all, and this block is the one part read to *decide*, so it is worth being legible
before the rest catches up. It is now one word to change rather than a rewrite.

**Not under the authoring form.** That form has nineteen fields and shows
thirteen of them in an eighty-by-twenty-four window; a three-line block would cost
a quarter of the fields for something read occasionally. A screen costs nothing
until it is asked for, which is the same trade the art preview makes.

### And what a trait is, which needed the same clause

A trait's sentences were derived and nothing else, which is right and is also why
they read as a field dump:

```
  venom_blood · máu độc
    Ai đánh trúng nó thì chịu lại 4% công của nó, và dính trúng độc, 2% khả năng.
    Miễn hoàn toàn trúng độc.
```

Three things wrong with that, and they are three different kinds of wrong.

**It never says what the trait is.** Every line reports what a number does; the
authored name *máu độc* is rendered in the heading and never in the sentences, so
the mechanism arrives with nothing to hang it on. `Passive.Flavour` is the one
line allowed to say it, under the same digit ban a skill's clause obeys.

⚠️ **A trait's clause may name no body, and there is no exemption.** A skill free
for anybody may not say *mai* and a restricted one may, because the restriction
guarantees the body — `ingrain` names roots and only a plant may take it. **A
trait has no restriction mechanism at all**: no element, no archetype, no species,
no character. So the ban is unconditional, and it is not a rule waiting to be
relaxed by a future field — the field would have to be built first.
`TestATraitFlavourNamesNoBody` holds it, and `TestATraitFlavourSpellsOutNoNumber`
closes the spelled-numeral half the character check cannot see.

**The order was the order the fields were declared in**, which read backwards on
the one trait with two halves: `venom_blood` answered an attacker and *then* said
it was immune to poison, when the fiction runs the other way round. The order is
now what the holder **is** (grants, resists) → what its **own attacks** do
(applies, amplifies, drains) → what attacking **it** costs (replies) → **when**
any of it is true (while).

**And every sentence leaned on "nó".** Six of the eleven trait wordings led with
a bare pronoun — *Nó gây trúng độc mạnh thêm 30%*, *Mọi đòn của nó hút lại 25%* —
which was a deliberate choice at the time, to avoid capitalising a lowercase data
name at the head of a sentence. It was the wrong trade: a description is about one
unit and nothing else, so the pronoun carries no information and only makes the
line longer. The wordings drop it and name the thing instead — *Hiệu quả trúng độc
mạnh hơn 30%*, *Mọi đòn hút lại 25%* — and the reply says who is being paid back
rather than leaving two pronouns in one sentence to disagree.

```
  venom_blood (máu độc)
    Máu chảy trong người vốn là nọc; ai cắn phải thì tự chuốc lấy.
    Miễn nhiễm trúng độc.
    Ai đánh trúng thì bị phản lại 4% công của người bị đánh, và có 3% khả năng
    dính trúng độc.
```

⚠️ **3%, not 2.5% and not 2%, and the two renderers are the point.** A share was
first *truncated* to whole percent, on the argument that a fraction of a percent
is a tuning detail whose exact figure sits in the listing beside the sentence.
Both halves failed on traits: a skill is priced in **hundreds** of parts per
thousand and loses nothing, while a trait is priced in **tens**, so a reply chance
of 25 printed as 2% — a fifth of the value gone — and there was no listing beside
a trait to check against. Truncation was replaced by the exact figure, and the
tenth it brought is now rounded away again, because a tenth of a percent is a
precision a *player* cannot act on and a decimal point is the only mark in the
line that is not a comma.

So `forge.Percent` keeps the tenth for **hexforge's tables**, where an author is
tuning the number and the tenth is what is being tuned, and `i18n.share` rounds
for **sentences**. Half away from zero, so 25 becomes 3 rather than the 2 this
started at.

**What makes the rounding safe is a rule on the data, not a decimal place.**
Nothing is ever tuned by **less than a percent** — a share that small is one
nobody feels across a battle, so it is not a tuning anybody authors on purpose,
and a description of one would come out as *0%*, which reads as a feature that
does not work. `TestNoShippedShareIsUnderOnePercent` holds it over every shipped
skill, trait and status. Carrying a tenth in the sentence to survive data the
rule forbids would be the renderer paying for a case that cannot happen.

## Battle logs

A battle can be written out, read back, printed, and re-run from its seed to check
the file is a faithful record rather than a story about one:

```
go run ./cmd/hexarena --auto --seed 11 --log b11.json
go run ./cmd/hexarena --replay b11.json --verify
```

The log carries the seed, the decisions taken and the events produced. Seed and
decisions together reproduce the battle from nothing, so `--verify` re-runs it and
compares every event. That check found two real defects the first time it ran.

Undo works the same way and needs no snapshot: the client drops the last decision
that was its own and replays the rest. Because the engine is deterministic that
lands on exactly the position the battle was in, with nothing deep copied to get
there.

Kinds, sides and outcomes are written by name, not by number, so inserting a
constant later cannot silently reinterpret every log already saved.

A side is written only when there is one, which is why "no side" rather than
"ally" is the zero value. With a real side at zero, every ally unit's opening
event left the field out and a reader got the right answer only because ally
happened to be declared first — and a battle with no winner wrote exactly the
same thing while meaning the opposite. A won battle now names its winner and a
draw names nobody. Logs written before that change do not `--verify`.

## Layout

```
cmd/hexarena/          terminal client: all input and output, no rules
cmd/hexforge/          authoring tool: the only program that reads the data
                       directory rather than the embedded copy, and the only
                       one that asks whether a file exists
internal/tui/          rendering, pure functions over the event log
internal/seed/         the embedded data and the loaders that parse it
internal/core/
  scale/               proportional arithmetic and the saturation curve
  rng/                 the only source of randomness
  hex/                 board geometry
  pattern/             the shapes an area skill covers
  element/             the affinity chart and a unit's one or two elements
  combat/              the damage formula, accuracy, dodge and block
  progression/         stat curves, evolution and the stat budget
  modifier/            buffs and debuffs
  status/              timed effects
  atb/                 turn order
  skill/               skill declarations, validated against every other book
  cast/                who a character is: origins, archetype presets, characters
  battle/              the only package that holds state; emits the event log
```

Every package below `battle` is a pure function of its integer arguments.
`battle` owns the numbers that change, and `rng` is the only place randomness
comes from.

## Data

Balance lives in `internal/seed/data`, embedded at build time:

| file | what it sets |
| --- | --- |
| `elements.json` | the affinity chart and its multipliers |
| `combat.json` | the defence constant, the damage floor, the hit-chance floor, the charge cap |
| `progression.json` | the level cap, the stat ceilings, the effective-health budget |
| `modifiers.json` | how far a buff and a debuff saturate |
| `patterns.json` | the area shapes and the splash share |
| `statuses.json` | the timed effects, their tick power and their modifier terms |
| `skills.json` | the skills |
| `origins.json` | the works the cast is borrowed from |
| `species.json` | what a unit can be: a shell, roots, a lineage |
| `archetypes.json` | the role presets: a suggested stat curve and kit per role |
| `cast.json` | the authored characters, each with an evolution line |
| `roster.json` | a seed roster to exercise the engine with |

Changing a number there changes the game without touching Go. The tests will tell
you what moved: several of them freeze design figures deliberately, and the golden
files under `testdata` are a record of what the numbers currently produce. Run
`go test ./internal/core/hex ./internal/i18n ./internal/seed ./internal/tui -update` to accept a
change, and read the diff — that diff is the point of them.

### The seed roster is the balance instrument

`roster.json` is 3v3 by character reference, and **no unit appears on both
sides**:

| | ace | support | young |
| --- | --- | --- | --- |
| ally | Venusaur, 60 | Wartortle, 16 | Charmander, 8 |
| enemy | Blastoise, 60 | Charmeleon, 28 | Ivysaur, 16 |

That is a measuring instrument rather than a scenario, and it is the whole
reason the file is shaped that way. It used to be the same character three times
on each side, and **a mirror cannot measure anything**: a change to a number
helps both squads by exactly as much, so the win rate moves only by noise. That
is what stopped `razor_leaf`'s piercing value from being judged by anything but
its damage table — giving it 400 moved the ally win rate from 23 of 40 to 25,
which is nothing.

Now it measures. Over four thousand auto-battles the roster sits at **49.5 per
cent** to the ally, on a roster where the ally holds the only Venusaur and the
enemy the only Blastoise. It has moved four times, and every move was a feature
landing rather than a number being tuned: 48.5 before `blaze` was gated, 49.2
after, 51.9 once `venom_blood` began answering whatever attacked it, and 49.5
once a placement had to bring four skills out of nine. The reply moved it toward
the ally because the ally's Venusaur is the unit most attacked on the board; the
slots moved it back, because that same unit is the one with the deepest kit to
lose. Taking `razor_leaf`'s pierce off now moves it to 46.0 — see *Piercing* for
what that figure has done across the three games it has been measured in. The 40-seed sweep the test runs reads
20–20, and it is far too coarse to tune against: it read 45 per cent on a draft
whose true rate was 55.

Four properties earn their place, and each is a way the roster was wrong before:

- **Every number weighs differently on the two sides.** Each species appears at a
  different level on each, so touching bulbasaur's curve moves an ally ace and an
  enemy support rather than two identical units. `TestTheShippedRosterIsNotAMirror`
  holds that, and it compares the *resolved* units — a name, a stat line — because
  two units agreeing on those are the same unit however they were authored.
- **Every unit can reach every enemy.** `battle.New` only refuses a unit that can
  reach nobody, which is the right rule for a game; the seed roster is held to
  the stricter one, because a battle that cannot finish measures nothing. An
  earlier draft stood its third unit on slot 1,2, which is **four** cells from
  the enemy's own 1,2 and past every range in the cast — and five seeds in four
  thousand ended with two survivors unable to touch each other. It was not even a
  draw: one kept refreshing a regeneration, so something was always pending and
  the board was never final. `TestEveryShippedUnitCanReachEveryEnemy` is that
  lesson.
- **Every element appears on both sides**, so neither squad holds an answer the
  other cannot have. The matchups are not a closed triangle: water beats fire and
  fire beats grass, but **grass against water is neutral**, so grass has no
  elemental answer at all and pays for it in neutral-element skills —
  `sludge_bomb` and `venoshock` land at full value on anything.
- **All three forms and every trait state are in play.** The levels span the
  final, middle and base stage; Wartortle and Ivysaur sit exactly on
  `endurance`'s unlock level, and Charmander at 8 is below `blaze`'s — so a
  battle exercises a unit holding its trait and a unit that has not earned one.
  Since `blaze` became gated there is a third state: Charmeleon has earned it and
  is not yet in it, so a shipped log carries a `passive_held` partway down rather
  than only at the opening board.

Squirtle is the finding this produced, and it is the kind a mirror hides. Water
is the strongest of the three elements and Blastoise still cannot carry the ace
slot on its own: its attack and speed curves are the lowest in the cast, so an
element advantage does not compensate for a passive stat line. What balances the
two squads is the levels of the units behind the aces.

## Authoring a cast

A character is a **definition**; a roster entry is a **placement**. Keeping them
apart is what lets the same character stand in a dozen encounters at a dozen
levels while the engine only ever receives the flat stat line that falls out of
resolving one. Three things get authored:

- An **origin** is a work a character was borrowed from: an id, a title, a medium
  (`anime`, `film`, `series`, `game`, `comic`, `novel`) and optionally a year and
  a note. An origin nobody has borrowed from yet is allowed.
- An **archetype** is the preset a role starts from: a suggested stat curve for
  every stat, a suggested kit, and the formation column the role belongs in. It is
  what stands in for a character *class*: the original design called for one, and
  it was dropped on purpose, because with skills declared as data an archetype's
  curve and kit already say everything a class name would have said. Note the
  consequence — an archetype has **no mechanical effect at all**. It never reaches
  the engine (`battle.Roster` carries stats, skills, affinity and a slot, and
  nothing else), so two units differ only by their numbers and their kit. It is
  a starting point and not a constraint — a character records which preset it came
  from and is then free to differ, which is what stops two units of the same role
  being the same unit with two names. A preset that does not itself fit the stat
  budget is rejected at load, because it would hand every author a stat line that
  fails later.
- A **character** is the definition: an id, a name, an origin, the archetype it
  was tuned from, a path to its art, one or two elements, a kit, and an
  **evolution line** — one or more stages, each with its own stat curve, the level
  a stage declares being the first level it owns.
- A **stage may name its own art**, and most do not. A form that names none shows
  the character's picture, which is what keeps the field optional and every
  character authored before it existed valid — `cast.Character.StageArt` is the
  one place that fallback is decided, because two screens inventing it separately
  is how a character ends up with two pictures depending on which one is asking.
  A character therefore has a *set* of pictures rather than one, and `hexforge
  check` verifies every one of them: art that only a grown form uses is art
  nobody looks at until the character has grown, so a missing file there is
  exactly the one that would surface in front of a player.

A unit may only carry a skill of an element it shares, or a neutral one — that is
what makes a second element worth having, since it buys a second line of skills
rather than a better multiplier. So a kit *demands* elements, and the character
carrying it must have all of them. The demand is derived from the kit, never
authored, and `hexforge archetypes` shows it in the `needs` column; a preset whose
kit demanded three elements would be rejected, because no affinity can hold
three.

```
go run ./cmd/hexforge                  # list the subcommands
go run ./cmd/hexforge origins          # the catalog of works
go run ./cmd/hexforge origins add my-series --title "Some Series" --medium series --year 2024
go run ./cmd/hexforge species          # what a unit can be, and who is one
go run ./cmd/hexforge species add fox --name "cáo"
go run ./cmd/hexforge archetypes       # the presets, their curves and their kits
go run ./cmd/hexforge skills           # the declared skills and who may carry each
go run ./cmd/hexforge statuses         # the timed effects, grouped, and what each does
go run ./cmd/hexforge skills add oath --power 1200 --accuracy 900
go run ./cmd/hexforge skills edit oath --power 1100   # change one already in the book
go run ./cmd/hexforge cast             # the authored characters
go run ./cmd/hexforge new              # create a character
go run ./cmd/hexforge show some.id --level 30
go run ./cmd/hexforge check            # parse from disk, verify the art, report the budget
go run ./cmd/hexforge spar some.id     # fight it against the whole cast and report the rates
```

`hexforge new` prefills from flags and prompts only for what is still missing, so
`--id --name --origin --archetype --image --element --bio --species --skills` and
the per-stat `--hp --atk --def --spd --acc --ddg` overrides (written `base:max`)
turn it into a one-liner. Choosing an archetype fills every curve and the kit, and
each prompt shows the preset as its default. Two of the orderings are deliberate:
the kit is asked before the element, because the kit is what decides which
elements are legal, and what the character *is* is asked before the kit, because a
skill kept for a lineage asks for it. Before writing, it
prints the resolved level 1 and level 60 lines with how much of the
effective-health budget they spend; `--yes` skips the confirmation. Every answer
is checked as it is entered, against the same parsers the game loads through — the
tool knows no rules of its own, which is why a character it writes is a character
that loads.

`hexforge skills add` and `hexforge skills edit` take the same flag names, and
the difference between them is what an absent flag means. On `add` it is a
question the wizard asks or a default it takes; on `edit` it means leave the
field alone, so `--cooldown 0` sets a cooldown to zero, no `--cooldown` leaves
it, and `--restrict-elements ""` clears a list. All four allowlists take a flag —
`--restrict-elements`, `--restrict-archetypes`, `--restrict-characters`,
`--restrict-species` — and an edit that names none of them leaves every one of
them exactly as it was. A skill's id cannot be edited —
renaming one has to change every kit and every restriction that names it — and an
edit that would leave an authored character or an archetype preset unable to
carry the skill is refused before anything is written, naming who would break.
Editing is balance rather than content, so a successful one reports the damage
before and after and says the golden files have moved.

With nobody watching — a pipe, a script, CI — it takes the default for every field
that has one and fails only on a field that has none, naming the flag that would
have supplied it. Those are `--id --name --origin --archetype --element`; the art
path defaults to one derived from the id, and the kit and every curve come from
the archetype:

```
hexforge new --id my-series.lee --name "Lee" --origin my-series \
  --archetype duelist --element wind/ground --yes
```

### Whether a character belongs

`hexforge check` says a character is **legal**: the budget is not overspent, the
affinity carries every skill in the kit, the art is really on disk. None of that
says whether it *belongs* beside the ones already written, and that question has
only ever had one honest answer — fight it and count.

```
go run ./cmd/hexforge spar pokemon.squirtle --seeds 200
```

```
pokemon.squirtle — Squirtle at level 60 as Blastoise, water
brings water_gun bubble bite withdraw and endurance
3 rows, 200 seeds from each slot, 1200 battles in all

opponent                     rate  won  lost  drawn  turns  first move
pokemon.bulbasaur            0.0%    0   400      0     30  +0.0%
pokemon.charmander          30.5%  122   278      0     32  +0.0%
pokemon.squirtle (control)  50.0%  200   200      0     56  +9.0%

overall 15.2% against 2 opponent(s)
```

Water is supposed to answer fire, and Squirtle still loses to Charmander seven
times in ten. That is the sort of thing this exists to find, and four things
about *how* it finds it, because each one is the difference between a figure and
a number that merely looks like one.

**Every pairing is fought twice, and that is the measurement rather than
thoroughness.** The turn queue breaks a tie by enlistment, so of two units with
the same speed the one placed first acts first for the whole battle. Against an
identical copy of itself Bulbasaur wins **72 of 100** from that slot — so a
one-way rate would be that advantage plus the character, with no way to tell
which was which. Fighting both slots and adding them cancels it exactly. The two
halves stay on the record, which is what makes the **control row** — a character
against itself, even by construction — worth drawing at all: its `first move`
figure is what the slot alone was worth, and it is the number every other row on
the screen has to be read against. It is +44.0% for Bulbasaur, +23.0% for
Charmander and +9.0% for Squirtle, and the reason is on the same row: Squirtle's
duels run fifty-six turns and a head start washes out, Bulbasaur's run
thirty-four.

**A rate is over seeds, never over a battle.** One duel is a coin toss — the same
two units at two seeds can end either way — so a screen showing the result of a
single battle would be showing noise and calling it a finding. A hundred from
each slot by default; `--seeds` moves it.

**Both sides bring the first four skills and the first trait their learnset
declares**, and the report says which. A roster refuses to do this — a file that
picked four of nine on an author's behalf would never say which — and the two are
not in conflict: a roster *is* the conditions of a battle, so it has nobody to
state them to, while a spar is a measurement, and a measurement states its
conditions. It also gives learnset order a meaning it did not have, which is
deliberate — first declared is first choice, and every other rule (longest range,
highest power) would be the tool inventing an opinion about what a character is
for.

**Both stand in the front column.** Nothing on this board moves, so the slot
decides what a kit can reach, and that is the only column that asks nothing of
one: `hex.ReachNeeded` is 1 there, which is the shortest range a skill may
declare, and a skill aimed at its own caster is aimed at a cell that is always
occupied. So no legal kit can be unaimable from a duel slot, which is why a
refused pairing is an error rather than a row of zeroes — and
`TestTheDuelSlotAsksTheLeastOfAKit` is what keeps that true if the board ever
changes shape.

What the report deliberately does not do is choose an early form. A stage curve
only rises, so fielding one is a trade an author makes on purpose, and measuring
a trade nobody asked for would answer a question nobody asked.

`internal/forge` is the one package that touches the filesystem for anything
beyond reading a data file: it verifies that every picture a character names is
really there, its own and each of its forms'. `internal/core/cast` checks only the *shape* of an image path
(relative, no `..`, ending `.svg` or `.png`), because a core package may not read
the filesystem and only the caller knows what the path is relative to.

### The same authoring, full screen

```
go run ./cmd/hexforge-tui              # or: make forge-tui
```

`hexforge-tui` is a second front-end over the same `internal/forge`: a cast
browser that resolves a character at any level you walk to with the arrow keys, a
new-character form, the origin catalog with an add form, the status and trait
references, and the check rendered as a screen. Two things it can do that a sequence of prompts cannot — because
everything is visible at once and both are pure integer arithmetic recomputed on
every keystroke:

- a **live budget meter**, the effective health a stat line absorbs against the
  joint health-and-defence bound, so the limit is a number you are watching
  rather than a rejection at the end;
- a **live carry check**, which says whether the affinity carries every skill in
  the kit and names the first one it cannot, as the element or the kit is being
  typed;
- an **art preview**, `p` from the browser, which draws the picture of the form
  the level resolved to — so walking the level is how a character's forms are
  compared, which is the whole reason a form may have art of its own;
- a **spar**, `s` from the check screen, which fights the character under the
  cursor against the whole cast and draws the rates — the level walked with the
  arrow keys and the number of battles moved with `+` and `-`, both refought as
  they change. It is raised from the check rather than from the browser because
  the two are halves of one question, and because a measurement is worth nothing
  until the data behind it loads, which is the screen that says so.

Both answers come from `internal/forge` — the same functions the write goes
through, so neither can be a second opinion.
`TestTheFormProducesTheCharacterTheCommandLineProduces` asserts that the same
answers typed into the form and passed as flags resolve to the same character.

It speaks **Vietnamese and English, Vietnamese by default**.

```
go run ./cmd/hexforge-tui                 # tiếng Việt
go run ./cmd/hexforge-tui --lang en       # or: HEXARENA_LANG=en, or ctrl+l
```

`--lang vi|en` beats `HEXARENA_LANG`, an unrecognised value in either is an error
naming the two that work, and `ctrl+l` swaps the languages from any screen —
mid-form included, without losing a keystroke, because a field holds what was
typed and only the labels around it are redrawn.

#### Saving: ctrl+s, and ⌘S where a terminal will pass it

Every form writes on **ctrl+s**, which works everywhere, and also on **⌘S**,
which works where the terminal lets it. The footer names `ctrl+s` on every
platform, macOS included, and the menu note is where ⌘S is mentioned instead.

That is a drawing problem rather than a change of heart. ⌘ is East-Asian-
Ambiguous width: it is *measured* as one cell and a good many terminals *draw*
it as two, so the glyph lands on top of the character after it and `⌘S` comes
out as two characters overlapping. Nothing inside a program can find out which
sort of terminal is in front, and spacing it apart needs a cell the footer does
not have — the English character-form footer is 73 cells without the label, the
smallest window is 80, and the last cell of a row is left empty so that writing
it cannot wrap the line. Six cells, and no ASCII spelling of both keys fits in
six. ⌃, ⇧ and ⌥ are ambiguous in exactly the same way, so none of them is the
way out.

⌘S is not something a program can simply ask for. Command is not a modifier the
classic terminal escape sequences can encode — it does not reach the program at
all — so it takes the **Kitty keyboard protocol**, which reports it as Super and
which bubbletea v2 parses. Three things then have to be true at once:

| | |
| --- | --- |
| the terminal speaks the protocol | kitty, Ghostty, WezTerm, foot, and iTerm2 with CSI u enabled. **Terminal.app does not** |
| it passes ⌘S through | rather than opening its own *Save Text As…* |
| nothing upstream claims Super | on Linux a window manager may take it first |

Where any of those fails, ⌘S never arrives and ctrl+s is the answer — which is
the other reason the footer names a control-S and nothing else. A terminal that will not pass ⌘S can usually be
told to send the other one instead: iTerm2 and Ghostty can both map ⌘S to
`Send Text: ^S`, which reaches this program as the keystroke it always accepts.

That is possible because `internal/forge` hands over *facts* rather than
sentences: a refused kit arrives as a value carrying the affinity, the skill and
the skill's element, and `internal/i18n` turns it into
`hệ fire không mang được chiêu "sever" (hệ metal)` or into
`fire cannot carry the skill "sever", which is metal`. `cmd/hexforge-tui` holds
no wording of its own — a test greps its source to keep it that way. Element
ids, skill ids and the stat labels `hp atk def spd acc ddg` are the same in both
languages on purpose: they are what you type and what the data files store.

The preview is the one place in this client where colour carries information
rather than decorating it, and the monochrome path is therefore a different
drawing rather than the same one with the colour removed: with `NO_COLOR` it
draws a ramp of weights, which keeps the shading, instead of a silhouette in one
character, which would keep only the outline. `internal/forge.ArtImage`
rasterises — the only place in the repository that does — and hands back pixels,
because a terminal and a graphical client turn those into something to look at
very differently. Two pixel rows go into one cell as an upper half block, so a
cell is very nearly square and the picture keeps its proportions without anyone
correcting for the font. The drawing is cached against the file's size and
modification time rather than against its path, so redrawing an asset outside the
program updates the preview instead of being ignored until a restart: a tool
whose job is telling you the truth about a data directory should not be the last
thing to notice it changed.

`cmd/hexforge` stays English, and is not superseded. A full-screen program cannot
run with stdin as a pipe, so the flag-and-prompt tool is what a script, CI, and
this repository's own end-to-end tests use; `hexforge-tui` refuses to start when
stdout is not a terminal rather than writing control codes into a file. It takes
the same `--data <dir>`, quits on `q` or `ctrl+c`, backs out of a screen with
`esc`, asks before discarding an edited form, says so plainly when the window is
too small to draw one — it wants 80 by 24 — and encodes nothing in colour alone:
with `NO_COLOR` set it renders as plain text, and a character with no art still
reads `MISSING`, or `THIẾU`.

Every subcommand takes `--data <dir>`, defaulting to `internal/seed/data`. That is
also the caveat: the game boots from the copies baked in by `go:embed`, so an edit
needs a rebuild before it reaches a battle.

A roster entry may then place a character by reference instead of restating its
numbers:

```json
{ "id": "ally.lee", "character": "example-game.sprout", "level": 30, "side": "ally", "slot": [1, 1] }
```

The flat form keeps working. What is refused is the **mixture**: a reference that
also writes out `name`, `element`, `stats` or `skills` is rejected rather than
resolved by precedence, because two sources for one number is how the two drift
apart.

### What a unit is, not only how it fights

An element says what a unit is made of and an archetype says how it fights, and
for a long time nothing said whether it had a shell, roots or a lineage — so a
skill named after a body was free to land on a body it did not fit. `withdraw` is
the one that found it: Squirtle pulls into its shell, the name read perfectly on
the character it was written for, and nothing would have stopped a Machop taking
it and reading "thu mai" on a creature with no shell.

Two answers, and they are different rules rather than a preference. Where the
*effect* is general the **name** should be too, which is why `withdraw` is glossed
as a stance and not as a shell. Where the identity is the point the name stays and
the skill carries a restriction — and that restriction now has an axis to sit on.

`species.json` is a catalog like `origins.json`: an id, a word for a screen, an
optional note about where the line is drawn, and nothing else. A character names
what it is with `species`, a **list**, because a unit may be several things at
once — Charmander is a `lizard` and a `dragon` — and `skill.Restriction` has a
fourth allowlist beside the elements, the archetypes and the characters, which a
unit satisfies by being any one of the kinds named.

That is the whole of it. **Nothing in `battle` branches on a species**, exactly as
nothing branches on an archetype: it is a carry rule settled while a character is
authored plus a word a browser prints, which is what keeps it cheap and why
`scenarios.golden` and `replay.golden` did not move when it landed. What did move
is `dragon_rage` and `dragon_dance`, off `characters: [pokemon.charmander]` — the
wrong axis, chosen on purpose while there was no right one — and onto `species:
[dragon]`, where they say what they always meant.

Two things the axis is deliberately *not*:

- **Not a second place to say what a preset is for.** A preset may not hold a
  species-restricted skill, for the same reason it may not hold a
  character-restricted one: a preset says how a character fights and nothing about
  what it is, so every character built from it that is not one of those kinds would
  be refused, and the refusal would land on whoever wrote the character. That is
  why `scorcher` still suggests seven skills where Charmander carries nine — and
  why `blighter` suggests seven where Bulbasaur carries nine: `ingrain` and
  `synthesis` read as a body — roots and photosynthesis — so they are kept by the
  `plant` species and left the preset behind. They were on `elements` first,
  because that kept the kit whole, and grass was only ever a proxy for "something
  that grows": a grass-element construct with no roots could take both and nothing
  said no. A preset losing an entry is the smaller loss.
  `TestABodyBoundSkillIsRestricted` is where those judgements are recorded, and it
  names the *axis* each skill is kept by rather than only asking that it is kept:
  "anybody may carry it" is one failure and "the wrong list keeps it" is another,
  and the second reads as a pass.
- **Not optional-by-accident.** A character that claims nothing is nothing in
  particular, which is what most of a cast is and a real answer rather than a gap
  — it is exactly what a lineage skill refuses. The one place that reading is
  relaxed is a half-filled form, where an empty list is a question nobody has
  reached yet: `forge.Carrier` says so on the field, and a lineage skill picked
  before a species is settled is refused at the write instead of at the keystroke.

## Roadmap

### Graphical client with ebiten

The event log is the contract, and `--verify` proves it is a faithful one, so a
graphical client is a renderer over the same log rather than a second
implementation of the rules. `internal/tui` is the reference for what that means
in practice: it never reads the battle, only the events.

Open questions before starting:

- **Asset pipeline.** Unit art is SVG and ebiten cannot draw it, so it is either
  baked to PNG at build time or rasterised at load. The authoring tool has
  already answered this for itself — `forge.ArtImage` rasterises at load with
  `oksvg` and `rasterx` for the terminal preview — which narrows the question
  rather than settling it: a preview redraws once per look at a size bounded by
  `MaxArtPixels`, while a battle draws every frame, and tens of milliseconds a
  picture is affordable in the first case and not in the second. So the open
  question is whether the client rasterises once at load into a texture, or
  whether the build bakes the sizes it needs.
- **Animation against an instant log.** Events carry no duration. The renderer
  has to decide how long a strike takes to draw without the engine knowing or
  caring, and a skipped or fast-forwarded animation must not change what
  happened.
- **What the board shows.** The terminal client labels a cell with a two
  character tag and puts everything else in a table. A graphical one has room for
  health, statuses and the turn order on the board itself, which is a design
  question rather than a porting one.

### A deeper opponent

`battle.Suggest` picks the option with the highest expected damage and falls back
to the first usable non-damaging skill. It is enough to run a battle to its end
and it is fully deterministic, which is what the tests need, but it does not play
the game:

- It never buffs, never cleanses, never shields.
- It never sets a status up in order to detonate it. Across a full battle of five
  hundred events a detonate fires once and a cleanse never does.
- It does not read the elemental matchup when choosing a target beyond what the
  expected damage already implies, and it does not consider the turn order at all.

So the whole timed-effect layer is present, tested, and not actually being played.
That also caps how deep a hand-played battle feels, because an opponent that never
cleanses is one the player never has to think about.

Constraints any replacement must keep: `Suggest` reads no randomness and mutates
nothing, so a client may call it for a hint without disturbing the battle's own
sequence, and two identical battles must still produce identical logs.

### A gated grant: a stat change that comes and goes

Built. All four things a passive is normally asked for were already there — a
stat change, refusing a status, adding to what the holder does, and waiting until
it is hurt — and the one *combination* the canonical abilities want was refused
at parse rather than half-built. **Blaze** is now what it is named after: below a
third of its health, Charmander's kit hits harder.

```json
{ "id": "blaze", "grants": [{"status": "kindled"}], "while": {"below_health": 333} }
```

A gate covers the **whole** trait — its grants, its resistances and its riders
come and go together. That is why the burn immunity `blaze` used to carry moved
to a trait of its own: a fire creature that only resisted burns while it was
losing would be a rule nobody could state. **A trait wanting one gated half and
one ungated half is two traits**, and saying so is what keeps a gate from
becoming a per-field flag that has to be read out of the data.

Four things it needed, and they were worked out before any of it was written:

- **A door into a permanent status, for the engine only.** `Set.Remove` refuses a
  permanent status so that no cleanse can dispel a trait, which is right and
  which left a gate with no way to take its own grant back. `Set.Hold` and
  `Set.Release` are that way in and out, and each refuses what the other pair
  handles — Hold will not touch a timed status, Release will not either — so
  neither becomes a second `Apply`/`Remove` with the rules missing.
- **An event each way.** `PassiveHeld` was already the trait coming on; it now
  fires mid-battle as well as with the opening board, and `PassiveReleased` is
  the way back. A trait letting go takes a visible number down with it, and the
  log is the only contract a renderer has.
- **A retune each time**, since a gated trait touching speed reorders the queue.
  A turn already ends with a sweep, so this is not what keeps the queue correct —
  it is what puts the speed change **next to the trait that caused it** instead
  of several events later beside whatever happened to be resolving.
- **One re-evaluation point**, for the unit whose health moved rather than for
  everybody.

⚠️ That last one is where the plan was wrong about its own code. It said health
moves in `wound` and `heal` and nowhere else. It moves in **three** places: the
strike loop in `resolveAgainst` subtracts from its target directly rather than
calling `wound`, and that is where nearly all the damage in a battle is dealt. A
version hooked to the two named functions would have opened a gate for a poison
tick and never for a sword. The gate is read per *strike* rather than per skill,
so a trait that comes on after the first hit of three is in force for the other
two — otherwise the same trait would be worth less against a multi-strike skill
for a reason written on neither.

**A gated trait is very nearly a one-way door in an automatic battle, and that is
a fact about the opponent rather than about the gate.** A unit below a third of
its health is a unit that is losing, and `battle.Suggest` never heals anybody: the
only healing across sixty bench battles is what a drain returns to its own
caster, worth about a fortieth of a health bar against damage worth a tenth.
Across four thousand battle-seeds and every arrangement tried — a bigger drain, a
smaller holder, a higher line — a trait came back off **once**. So the release is
proved by a hand-played battle in `TestEveryEventKindIsReachable` rather than by
widening the sweep until the rare case turned up, because "a player heals and the
opponent does not" is the actual reason, and a test that passed at seed 3,197
would have hidden it.

### Two builds for one character, which is what the trait work is for

The roadmap entries below are each written as a mechanism. This is the thing they
add up to, recorded because the order to build them in only makes sense once the
target is stated: **Bulbasaur should be two different units depending on the trait
it brings.**

- **The poison specialist.** Immune to poison, its poison hurts more, and
  attacking it poisons the attacker.
- **The bloodsucker.** Heals from the damage it deals, and heals *more* the closer
  it is to dying.

Neither is one feature, and the pieces are already scattered across the entries
around this one:

| the build wants | where it lives | built |
| --- | --- | --- |
| immune to poison | `Resists`, on `venom_blood` | yes |
| its poison hurts more | `Amplifies`, on `virulence` | yes |
| attacking it poisons the attacker | `Replies`, on `venom_blood` | yes |
| heals from damage dealt | `passive.Passive.Drains` | yes |
| heals more when nearly dead | `While`, gating that share | yes |

`While` is the row that moved: it gates the whole of a trait, grants included,
and `blaze` is the first shipped trait to use it.

Four of the five are built. The poison specialist is most of the way there —
`venom_blood` refuses poison and answers whoever attacks it, which leaves only
"its poison hurts more" — and the bloodsucker is finished.

**When the first build arrives.** `venom_blood` was the one trait on Bulbasaur's
learnset with no level on it, so a Bulbasaur held the strongest of its five from
level 1. It is gated at **24** now: `endurance` from 16, the answer from 24, the
amplifier from 32 — the poison specialist assembles across the middle of the
climb rather than opening with its best piece.

⚠️ It is a **balance change and the figure moved**: 4000 seeds of the shipped
roster read **49.5% ally before and 53.2% after**. The swing is one-sided by
construction — `ally.venusaur` is level 60 and keeps the trait, `foe.ivysaur` is
16 and loses it. That second half is not optional: a placement naming a trait
above its level is a hard parse error rather than a silent drop, so the roster
had to change in the same commit or stop loading.

What `foe.ivysaur` takes instead is worth almost nothing — 52.9% with
`endurance` against 53.2% with none — so the 3.7 points are the loss of the
trait rather than the gap it left. It fields none, because handing it another
trait swaps one for another instead of showing the gate, and `ally.charmander`
already fields none. Compensating on the ally side was measured too and is
worse: `virulence` in place of `venom_blood` at the cap reads **56.3%**, because
it is the stronger of the two.

**The second build.** `passive.Passive.Drains` is a share of the damage its holder
deals, added to whatever the skill already drains and resolved at the site that
already resolves one. `blood_thirst` takes a quarter of everything; `last_gasp`
takes two fifths and is gated at four tenths of health.

A share is the cheaper of the two gates rather than the only legal one — a grant
can be gated too (see *A gated grant: a stat change that comes and goes*), so both
*Overgrow* and *Vladimir* are writable and the difference is what they cost. Read
fresh on every strike at a site that already resolves, a gated share needs no door
into a permanent status, no event in either direction, and no retune. `last_gasp`
is a gate that cost one comparison.

Two decisions inside it. Trait shares **add** rather than composing, because a
share of the damage dealt is not a chance: two resistances compose by what each
lets through, and two drains simply both drain. And the total is **capped at the
base**, which is not the hard cap this engine rejects elsewhere — a buff ceiling
bounds how good a number may get, where this bounds a *conservation*: health taken
back cannot exceed damage dealt, the same invariant `skill.resolve` enforces on a
single share. Saturating instead would have been worse than either, paying out 285
for a trait that says 400 on a skill that drains nothing.

⚠️ **A reply drains too, and did not until it was looked for.** `resolveAgainst`
paid out and `reply` did not, so a trait holding both jobs promised a share of an
answer it never gave — and the thing that said so was the description: *mọi đòn
của nó hút lại 25% sát thương gây ra*, **everything it does** takes back a share,
and an answer is one of the things it does. The placement's single trait slot
stops a unit carrying a replier and a drainer; it stops nothing about a trait that
is both, and `passive.Passive` holds both fields.

Nothing shipped does it yet — `venom_blood` answers and drains nought,
`blood_thirst` and `last_gasp` drain and answer nothing — so **no golden moved**.
That is the same shape the regeneration bug had: a job that renders in the
sentences and not in the engine, invisible because the data has not yet asked for
it.

The trait's own share only, since a reply has no skill to add one, and **before**
the kill rather than after: `resolveAgainst` drains from what it dealt whether or
not the target fell, so draining after the return would make lethal damage the
one blow worth nothing to take back.

⚠️ The heal carries `Drained`, the share it took, for the reason `Pierce` and
`Refused` exist: `Amount` alone cannot say why. Six hundred off a strike that
dealt three hundred and three hundred off one that dealt six hundred are the same
number by the time a reader sees them, and the skill's own figure stopped
accounting for it the moment a trait could drain too.

**The circular bit, resolved.** A character used to bring every trait it had, so
giving Bulbasaur all five rows above would have made it one unit that was better
rather than two that were different. A placement now brings **one** trait, so the
rows are a choice — which is what makes filling in the missing ones worth doing.
A slot is only a decision once traits differ in
**kind** rather than in number, which is why the slot was built last of the
three: the resistance, the gated grant and the reply are what it now picks
between. Either order would have worked; what would not have worked is building
neither and expecting the other to justify it.

Nothing is left on that order. The gate, *Answering back*, the drain, the
amplifier and the **trait slot** are all built, and the slot arrived last on
purpose: by the time it landed the traits differed in **kind** — a resistance, a
gated grant, a reply, a drain and an amplifier are five different sorts of thing,
and a placement brings exactly one of them.

**The scenario is playable.** `blood_thirst` and `last_gasp` are on Bulbasaur's
learnset at 20 and 40, which was the one line it was short of: both existed in the
book and the character learned neither, so the slot had nothing from the sustain
direction to offer. They were waiting for the slot rather than for a smaller
number, because before it a character brought everything it had and a fourth trait
made one unit better instead of two units different.

At the cap the slot decides between five traits of four kinds — a resistance that
also replies, an amplifier, a stat change, a drain and a gated drain — so bringing
the poison answer means not bringing the sustain, and the other way round.
`TestBulbasaurCanBeBuiltTwoWays` measures that the *choice* exists rather than that
any one trait does, and fails if the slot count grows or a direction loses its last
entry.

Nothing shipped uses `Applies` yet; `blaze` is the only trait using `While` on a
grant, and `venom_blood` the only one using `Replies`.

**And it answers with poison alone.** The reply's 40 per mille of counter-damage
was sold to buy chance, because at 25 per mille the poison landed **0.27 times a
battle** — once every four — which is a trait a player never sees work. Twenty
thousand seeds of the shipped roster:

| reply | ally | poison landed |
|---|---:|---:|
| 25‰ chance · power 40 | 53.0% | 0.27 per battle |
| **40‰ chance · no power** | **53.1%** | **0.44 per battle** |
| 50‰ chance · power 40 | 56.1% | 0.54 per battle |
| 50‰ chance · no power | 54.3% | 0.55 per battle |

So the trade is free and the trait is visible 63% more often for it. Raising the
chance without selling the damage costs **three points**, which is what the first
row and the third measure between them.

It reads better too: *máu độc* is blood that poisons whatever bites it, not blood
that punches back — and dropping `power` is what tells it apart from a thorns
trait rather than making it one with a rider.

⚠️ **σ is about 0.35 points at twenty thousand seeds**, so a gap under 0.7 is
noise. The four-thousand-seed sweeps used earlier in this file cannot resolve one,
which is why these four rows were re-measured at five times the count.

⚠️ **No shipped trait answers with damage any more.** `venom_blood` was the only
replier in the cast, so a battle from the shipped roster emits no `Damaged`
carrying a trait at all, and `TestTheShippedRosterAnswersItsAttackers` now asserts
only the status half. The mechanism is not untested for it — the fixtures in
`internal/core/battle/reply_test.go` exist for exactly that — but the shipped game
no longer *does* it, which is a cast fact and comes back the day a trait wants
counter-damage again.
⚠️ `Applies` is close to the third row above and is **not** it: it adds to what the holder's *own* attack inflicts —
touch something and it is poisoned — where the row wants the reverse, an answer to
being attacked. Writing the first and calling it the second would be the cheapest
way to close this out wrongly.

### Answering back

Built. `venom_blood` is *máu độc*, and blood that is poisonous now costs whatever
bit into it: whoever damages the holder takes a share of a stat back, and may be
poisoned by it.

```json
{ "id": "venom_blood",
  "replies": { "power": 40, "applies": [{ "status": "poison", "chance": 25 }] },
  "resists": [{ "status": "poison", "amount": 1000 }] }
```

#### The stat a reply is priced against

A reply names it, and until it did, every one was priced off attack.

```json
{ "id": "thorns", "replies": { "power": 80, "scaling": { "stat": "defense" } } }
```

⚠️ **Attack was the wrong default rather than a missing field**, and the reason is
the whole point of the feature. A trait that answers whoever hit it belongs to a
unit **built to be hit** — which is an armoured unit and not a sharp one — so
pricing every reply off attack made thorns worth least to exactly the character
thorns are for. Blastoise carries 640 defence and 460 attack: the same share off
the wrong stat is a **third** less.

It takes a whole `skill.Scaling`, so it says the stat *and* whether it reads the
base line or the modified one, and `skill.ParseScaling` is exported so that a
trait and a skill are read by one parser. Reading the *current* value by default
is what makes a stat trade work: a trait that gives up speed for defence buys
thicker armour and harder thorns with one number.

⚠️ Widening `origin` to carry the whole declaration fixed a latent bug nobody
could reach. A skill declaring `"source": "base"` had its **damage** read the base
line and its **damage-over-time tick** read the current one — two answers to one
question. No shipped skill declares scaling at all, so it was unreachable, and it
would have arrived on the day one did.

⚠️ **The word "attack" was hardcoded in three places**: the engine, the reply
sentence, and `hexforge passives`. The listing's copy is in no golden and nothing
else reads it, so the mutation that put the literal back **passed the entire
suite** — and an author tuning a thorns trait would have read "8% attack" off a
listing whose engine was multiplying defence, and written a number a third out.

`thorns` ships on Squirtle from 32, at 8% of defence. Measured in a duel, which is
a reply's best case because the holder is attacked every turn it survives: 14.2% →
**15.3%** overall, and 28.5% → **30.7%** against Charmander. ⚠️ Squirtle still
loses to Bulbasaur every time either way — that is a cast problem, and this does
not touch it.

It is not `applies` wearing a different hat. That one hands the holder's own
attack an extra application, so it fires on a target the holder chose, at a
moment `battle` is already resolving. A reply fires on the **attacker**, during
somebody else's turn, from a unit that is not acting — and inventing that hook
was most of the work.

**It is not a second damage path**, and the shape of the code is the guarantee
rather than a comment saying so. The damage goes through `combat.Rules.Damage`
and the statuses through `battle.inflict`, which is the function every skill's
applications already use — so a reply is refused by the same resistances, rolled
by the same source, and written to the log as `damaged`, `status_applied` and
`status_resisted` like anything else. What `inflict` used to take was a whole
`skill.Skill`; it now takes an `origin`, which is the three things it ever wanted
out of one — what to name in the log, which element to price a tick against, and
which stat that tick scales off. A reply names a trait instead of a skill and is
otherwise indistinguishable, which is exactly the point: a replay reads it
without knowing, and `--verify` re-runs it from the seed.

Two absences on the declaration, and they are the same decision. **A reply has no
element and no accuracy.** The elemental chart prices what one creature threw at
another, and a trait reading it would make a fire creature's blood weak to water
for a reason written nowhere on the trait; an accuracy roll asks whether contact
was made, and contact is the thing that has already happened.

Four rules, and three of them are simply where the call sits — after the whole
skill, once, per holder.

- **A reply may kill**, and a battle can end on a turn nobody took. A
  damage-over-time tick already ends battles, so the shape existed; what is new
  is that the unit dying is the one taking the turn.
- **A reply never triggers a reply.** Closed by rule and not by a depth counter,
  because a counter is a number somebody raises. The list of who may answer is
  built from the skill's own targets, and a reply is not in it — so there is
  nothing for a second one to answer. Two holders facing each other settle in one
  exchange.
- **A reply answers a use of a skill, not a strike.** Otherwise a trait's worth
  would scale with somebody else's strike count, and `fire_fang` would be quietly
  worse than `flamethrower` into one holder for a reason written on neither.
- **The holder takes every strike first and answers afterwards** — indeed, every
  *target* takes the whole skill before anybody answers it. A reply resolved
  inside the loop could kill the actor while it still had cells to hit, and "what
  happens to the rest of the skill" is a question with no good answer.

That last one leaves the holder's death as the only one that can land first, and
dead is dead: **a holder killed by the skill does not answer**, the way it cannot
be healed. Which hands retaliation a counter without anybody designing one —
killing the holder outright is how a reply is avoided, so a trait that punishes
attacking rewards hitting hard instead of taxing everybody equally. The same rule
runs the other way: once one holder's reply has killed the attacker, the holders
behind it are answering a corpse, and they do not.

#### What the sweep found, which is why the shipped numbers are small

A reply is the first thing in this game whose worth is set by **how often its
holder is attacked and how long it survives**, and no number on the trait can
express that. Both Bulbasaurs carry `venom_blood`; the ally fields Venusaur at 60
and the enemy Ivysaur at 16, so the ally's holder is hit far more, survives far
longer, and answers far harder. Over four thousand battles the ally deals 86 per
cent of all the reply damage in the game while making 69 per cent of the replies.

That asymmetry is not the trait's, it is the roster's — and it means the trait
cannot be big without the roster stopping being a measuring instrument:

| power | poison chance | ally win rate |
| ---: | ---: | ---: |
| — | — | 49.2% |
| 40 | — | 49.5% |
| **40** | **25** | **51.9%** |
| 40 | 200 | 73.5% |
| 80 | 100 | 75.4% |
| 250 | 500 | 98.3% |

So the shipped reply is a scratch and a two and a half per cent chance of poison.
⚠️ Every figure in the table above was measured **before** a placement brought
four skills of nine, which moved the same roster from 51.9 to 49.5 — so read them
as the shape of the curve rather than as today's numbers. Anything an author is tempted
to raise here should be measured over thousands of seeds first, because the
40-seed sweep in the tests cannot see a move of this size.

⚠️ **It also moved a measurement that was already in this file.** Removing
`razor_leaf`'s pierce used to cost the ally 2.7 points; against a roster where
being the attacker costs something, it *gained* the ally 1.1 instead. The sign
flipped, and it flipped at every reply size tried. Piercing helps whoever is
attacking, and this was the first thing in the game that charges for attacking —
so no figure measured before a feature can be carried across it without being
taken again. It flipped back when four slots landed; see *Piercing*, which now
carries all three measurements.

### Looking a status up

Built. `Lang.DescribeStatus` sits beside `Describe` and `DescribePassive`, and it
answers the third question: the first two are *should I use this* and *what is
this unit carrying*, and this one is *what has just happened to me* — which the
game asked most often and could not answer at all. The log said *resisted mire*,
the unit table said *mire*, and nothing anywhere said mire is a quarter off speed
for two turns.

Three places ask for it, all the same sentences:

```
> ?mire                        one status, by id or by the name it is printed under
> ?*                           the whole reference, grouped
hexforge statuses              the sixth listing, in English like the rest of it
hexforge-tui → hiệu ứng        a screen with the listing and the description
```

```
  mire · sa lầy
  ----------------------------------------------
  Giảm tốc 25% mỗi lớp.
  Đủ 2 lớp là 50%.
  Giảm chỉ số · 2 lượt · tối đa 2 lớp.

  Đây là số khai báo: nội tại khuếch đại hay kháng cự đổi thứ thật sự dính.
```

**A life, not a tick.** `poison` ticks for 50% and `burn` for 80%, so a reference
printing the tick alone has a reader rating burn the heavier of the two. Over
their lives poison is 150% to burn's 160% for one stack and **450% to 320% at
their caps**, which is the other way round at the only place it matters. Both
figures are stated, and the stacked total with them, because a percent per stack
against a cap of three is a different number from the percent.

⚠️ **One stack's life and the cap's, not the ramp.** `skills.golden` prints a
third figure under the same words — the full ramp of a status reapplied every turn
until it caps, 600% for poison — and that one is an author's ceiling rather than a
player's fact: reaching it needs a skill off cooldown every turn, which no shipped
kit has. What the reference states assumes nothing about how often it is applied.
Two numbers under one phrase is worth the note; they answer different questions.

⚠️ **Permanent is "always", never "0 turns"**, and a one-stack status is never
given a rate. `toughened` reads *tăng thủ 15%* rather than *15% mỗi lớp*: it caps
at one stack, so a rate is a promise of something `status.Set` refuses.
`TestAPermanentStatusIsNeverGivenARate` holds it.

**Grouped, by `status.Book.Grouped`.** The grouping is the half a flat list
cannot carry: `rapid_spin` strips a `stat_debuff` and a `dot`, and that is
unreadable to somebody who cannot see which statuses the two words cover. It is
one function in core rather than one per front-end, because a grouping worked out
three times is three answers to *which category is this in* waiting to disagree.
In the tool the headings are **rows**, not something drawn between them — the
listing scrolls, and a heading computed while rendering falls off the top of the
window and leaves the rows under it unlabelled. The price is a cursor that can
index one, which `TestTheStatusCursorNeverLandsOnAHeading` refuses.

⚠️ **It describes the declared kind, not what a unit will feel.** `virulence`
amplifies poison, a resistance refuses it, and a stacked stat term saturates
rather than adding up — so the figures are the book's and not the log's. The
caveat says so **once** per reference rather than under every status: a warning
repeated fifteen times is a warning nobody finishes reading. It is also the last
line of the screen, and the frame cuts from the bottom, so
`TestTheStatusCaveatSurvivesTheSmallestWindow` measures it at eighty by
twenty-four in both languages.

**And it is what found the regeneration bug.** `regrowth` was declared, glossed
and inert: `Battle.inflict` computed a tick only for `status.Dot`, so a `Regen`
stack went on carrying nought. The reference describes what the book *declares*,
which is the honest thing for it to do and is exactly why the gap became visible
— writing *hồi 40% mỗi lớp, một lớp là 120%* beside a status that healed nobody
is the sort of thing a reference is for. Fixed since, in *A regeneration that
heals*.

### Reading a trait

Built. `Lang.DescribePassive` had one caller — `?TAG` at the battle prompt — so
the tool where a trait is actually tuned printed none of it. An author moving
`virulence` from 300 to 200 watched the table's `+30%` become `+20%` and never
read the sentence a player gets, which is the exact drift the skill description
screen exists to prevent, reproduced one layer over.

Two holes, both closed.

**`?` on the cast browser** now raises the description screen for the traits the
character under the cursor carries **at the level it is sitting on**:

```
Bulbasaur  cấp 60

  venom_blood (máu độc)
    Máu chảy trong người vốn là nọc; ai cắn phải thì tự chuốc lấy.
    Miễn nhiễm trúng độc.
    Ai đánh trúng thì bị phản lại 4% công của người bị đánh, và có 3% khả năng
    dính trúng độc.

  endurance (bền bỉ)
    ...
  … 17/23 dòng, pgdn/pgup để cuộn
```

**One screen, not a sixth listing.** `blurbScreen` branches on `from`, the single
field it keeps — and `from` is not a cursor, it is which screen is behind, a
question `esc` had to answer anyway and used to answer with a constant because
there was only ever one answer. A second screen would have been a second copy of
the framing, the footer and the escape, differing only in which describer it
called.

**No cursor and no level of its own**, the same refusal `screenPreview` makes:
both are read off the browser, so walking either here walks it there and the two
cannot disagree about what is in front. The level matters more than it looks — a
trait comes in at a level, so a screen listing every *declared* trait would be
describing traits the character has not learned. `PassivesAt` at
`progression.Furthest`, which is what the detail pane behind it resolves with.

⚠️ **It scrolls, and a scroll offset is not the cursor that was refused.** The
difference is what the two can disagree about: a cursor could point at a
different character than the browser behind it, while an offset selects nothing —
it is which lines of the browser's own answer are visible, and every key that
changes *what* is described resets it. Five traits at the cap wrap past an
eighty-by-twenty-four window, which is the declared floor rather than an unusual
case, and letting the frame cut it would mean the one screen built for reading a
trait cannot finish reading one.

⚠️ **The sentences wrap to the floor, not to the window** — the opposite of what
`m.wrapped` does, and right for a different reason. Those rows carry authored
free text, which has to go somewhere and takes whatever width there is; these are
the program's own prose, and prose has a measure. Before wrapping, the reply
sentence was cut mid-word at the floor — *…3% khả nă* — which reads as the tool
being broken rather than as a terminal being narrow.

**And `hexforge passives` gained the two columns it never had.** `answers` and
`drains`: two of the six jobs the parser accepts rendered **nowhere** in the tool,
so `blood_thirst` printed a row blank after its name and `venom_blood`'s reply was
invisible. The reply is **one** cell, because `DescribePassive` writes one sentence
for the whole of it on purpose — what a reader wants is what attacking that unit
costs, and a damage column filed away from a status column leaves them adding it
up across the table.

**And the listing that answers the other question.** `?` on the browser answers
"what is this character carrying", filtered by a level — so a trait is reachable
only through a character that already has it, and a trait nobody has learned yet
is reachable from nowhere at all. The `nội tại` menu answers "what traits are
there", which is the question somebody has *before* they know which character to
look at.

```
nội tại  các nội tại đã khai báo, và ai mang

  id            tên tiếng Việt  ai mang
> endurance     bền bỉ          pokemon.bulbasaur@16 pokemon.squirtle@16
  blaze         bùng lửa        pokemon.charmander@16
  ...

  endurance (bền bỉ)
    Chịu đòn quen rồi, đau tới đâu cũng đứng vững tới đó.
    Luôn mang kiên cường.
```

⚠️ **The column that earns its place is "who carries it", not "who may".** A trait
has no restriction mechanism at all, so *may* is everybody and answers nothing.
Who actually **does** is the fact worth a column — and a trait nobody learns
cannot reach a battle, which is not an error (a catalog may be written before the
cast that fills it) but is exactly the sort of thing a listing exists to show. The
one under the cursor says so in words, because an empty cell in a column reads as
a column that failed to fill rather than as a fact.

`Library.TraitCarriers` walks the **cast**, not the trait book. A trait is
declared knowing nothing about who takes it and the edge lives on the character —
a learnset is the character's fact — so an index kept the other way round would be
a second place for the same edge to live.

Read-only, like the status reference and unlike the skills listing: a trait carries
modifier terms and shares of damage, which is balance rather than content, and
adding one changes what every character holding it does.

⚠️ **Not a column on `hexforge passives`.** That table is nine columns already and
a carrier row is as long as the cast makes it; a listing row that can be clipped
and a cursor that can select one are what make the column affordable, and the CLI
has neither.

**And the name in that sentence is now a door.** *Luôn mang **kiên cường*** left
the one word a reader does not know unexplained, and the reference that explains
it was a menu and two screens away. `?` on the traits listing opens the status
reference at the status the trait names, and `esc` comes back to the trait — one
keystroke each way.

The name is marked where it is printed, so `?` has something visible to be about.
Bold and no colour: it is standing in a sentence rather than in a column, and a
coloured word mid-paragraph reads as a link to somewhere the terminal cannot take
you. Nothing is carried by the marking alone — the sentence says the same thing
on a monochrome terminal, which is the palette's own rule.

**One function serves both halves.** `i18n.StatusesNamed(trait)` is the ids a
trait's description will name, in the order the sentences name them. The
alternative was reading the sentences back to find the names in them, which is
substring matching against prose in two languages: it styles a name that happens
to occur in a flavour clause, and misses one the glossary has no entry for,
because that one prints as a bare id.

⚠️ **It is a second reading of `DescribePassive`'s rules and drifts from them
silently.** A name the list has and the sentences do not is marked where nothing
is printed; a name the sentences have and the list does not is the one word on
screen `?` cannot open. `TestATraitNamesEveryStatusItsDescriptionNames` holds
the two together against the shipped book, and the rule that book cannot show is
pinned separately: **a reply names its first application and no more**, because
the one sentence a reply gets has room for one status — so a trait answering with
two holds two and names one.

⚠️ **Two shipped traits name nothing at all.** `blood_thirst` and `last_gasp`
only drain, and a drain names no status. `?` stays put rather than opening
whatever the status cursor happened to be on, which would answer a question
nobody asked.

⚠️ **`esc` had one answer and now has two.** The status listing is reachable from
the menu and from a trait, and a reader sent there by one keystroke expects the
next to undo it. `statusesScreen.from` is that, and it is **cleared as it is
used** — the first version returned before storing the cleared value, so a second
visit through the menu inherited the first visit's way back. A test caught it,
not a reading.

⚠️ **Marking is one left-to-right pass, not a replacement per name.** A pass per
name re-marks its own output whichever order the names are tried in: longest
first and `bỏng` matches inside the `bỏng nặng` just produced, shortest first and
`bỏng nặng` never matches at all. And each **word** is marked whole rather than
the name as a phrase, because the sentences are wrapped afterwards and the wrap
splits on spaces — a style spanning two words survives only until they land on
different lines, and then the first opens a sequence the second closes.

### A health threshold a skill can read

Built. `skill.Condition` reads how hurt the **target** is, where `passive.Condition`
reads its holder, and `brine` is finally the move it is named after: at or below
half health it doubles, 1000 power becoming 2000.

```json
{ "id": "brine", "power": 1000, "requires": { "below_health": 500, "bonus_power": 1000 } }
```

**The two conditions share their arithmetic and not their type**, which was the
thing worth deciding rather than discovering. `scale.AtOrBelowShare` is the one
comparison and both call it, so a rounding change lands in both at once. One
`Condition` serving both would have had a `Holds(health, maximum)` that could not
say *whose* health it meant, and the first mistake would be passing the other
unit's — and it was unwritable anyway, because `passive` imports `skill`.

A condition may now read a status, or health, or **both**, and both is *and*: a
second clause narrows a skill rather than widening it, because a clause that
widened would make every skill written under one reading wrong under the other.
Four ways of writing one that cannot mean anything are refused rather than
defaulted — asking nothing at all, counting stacks of no status, a share outside
parts per thousand, and consuming a status it never names.

The reading is a `skill.Target`: the stacks the target carries, its health, and its
maximum. A struct rather than three parameters because two of them are `int64`
health values that mean nothing apart, and a caller that swapped them would
compile. `battle` builds one in `conditionTarget`, which both `Suggest` and
`resolveAgainst` call — a rating built from a different reading than the resolution
would make the opponent prefer a skill for a bonus it does not get.

### A condition a skill reads about itself

Built, and it is the sentence above turned round: `requires` reads the target, and
**`self_requires` reads the caster**.

```json
{ "id": "outrage", "power": 2200,
  "self_requires": { "below_health": 400, "bonus_power": 1200 } }
```

Two fields rather than one field with a *whose* flag, because the book already
spells this distinction exactly that way: `applies` lands on whatever the skill
hit and `self_applies` lands on the caster. A reader who knows what the `self_`
prefix means there knows what it means here, and a flag would be a third thing to
get wrong. The type is the same `Condition` — everything about it is the shape of
a question and nothing about who is being asked.

Until it existed, two obvious skills had **no spelling at all**: *hits harder
while I am furied* and *hits harder while I am cornered*. `dragon_dance` had been
a setup with nothing to pay it off since it was written.

⚠️ **It is read once per use, in `Act`, and not in `resolveAgainst`.** That
function runs once per cell a shape covers, so a condition consumed there would
charge a column three times and a single-target skill once — a difference written
on neither skill. `Battle.spend` is that seam, and it sits *before* `applyToSelf`
so that a skill which both grants and spends a status cannot pay itself.

⚠️ **The bonus joins the power before the splash share is taken**, the same as the
target's. A shape's edge is worth less however the power was arrived at. This is
the one that nearly got away: a bonus added *after* the reduction still makes
every target take more, so the obvious test passes and the edge quietly takes a
full share. What catches it is the **ratio** between the aim and the edge, not the
rise.

⚠️ **`conditionCaster` is a second builder beside `conditionTarget`**, not a
parameter on one. A skill may read one status of its target and another of itself,
so a single builder would have to be told which condition it was reading — which
is the reading-versus-resolution mismatch `conditionTarget` was written to
prevent, arriving through the other door. `Suggest` reads it once outside its own
loop, which is where it is read for real.

One validator serves both fields. Every rule is about the shape of a condition and
none is about whose health it counts, so a second copy would be a second set of
rules nobody wrote down, and the looser of the two is the one an author would
find. `resolveCondition` carries the field name through every message, because
"asks nothing" on a skill with two conditions is a refusal nobody can act on. One
new refusal joins the four: a bonus power on a **self-aimed** skill, which never
reaches a target for the power to land on.

`skills.golden` grew a **whose** column, and that was not cosmetic — the table read
only `requires`, so it would have told an author their skill has no amplifier
while the engine amplified it.

`outrage` is the first user: 2200 power, and 3400 at or below forty per cent
health. A health gate rather than the `fury` payoff it was designed around, and
deliberately so — `battle.Suggest` never buffs, so a fury gate would never fire
under autopilot, while a cornered one fires on its own and pairs with the
frailty `reckless` buys. The dragon build's figure did not move: 42.5%.

What is still absent is the **gradient**: the move that hits harder in proportion
to how far the caster has fallen, rather than past a line. It is a multiplier on
power rather than a bonus to it and belongs in `combat` — `self_requires` is the
threshold version of the same idea, not a replacement for it.

### Learnsets and slots

Built, except for the half that could not be built first — see *Choosing to
evolve* below.

A character no longer simply holds a list of skills; it holds a **learnset**,
each entry from the level it is learned at. A placement then **chooses** four of
them, and one of the traits, and that choice is what is fielded.

```json
"skills": [
  { "id": "vine_whip" },
  { "id": "razor_leaf" },
  { "id": "poison_powder", "at_level": 8 },
  { "id": "sludge_bomb",   "at_level": 32 }
]
```

```json
{ "id": "ally.venusaur", "character": "pokemon.bulbasaur", "level": 60,
  "side": "ally", "slot": [2, 1],
  "skills": ["razor_leaf", "sludge_bomb", "venoshock", "leech_seed"],
  "passives": ["venom_blood"] }
```

**Skills and traits are one mechanism, and now demonstrably so.** Both are
`cast.Unlock` — one `{id, at_level}` shape, one validator, one "what is available
at level N" function — and the only thing that differs between the two lists is
how many slots there are. Even the authoring tool shares it: a learnset renders
through the same `UnlockSummary` a trait list does, so `razor_leaf
poison_powder@8` is one row and one function rather than two.

#### The four rules a placement is refused by

| refused | why |
| --- | --- |
| naming nothing | a slot is a decision, and a default would be this parser choosing four of nine on an author's behalf and never saying which |
| naming more than four (or more than one trait) | the slot count is the whole feature |
| naming the same thing twice | a wasted slot reads as a typo, not as a choice |
| naming what the level has not learned | the learnset is what makes a young unit different in *kind* rather than merely weaker |

A refusal lists what *was* available, because an author who has just been told
"no" wants the list rather than a second trip to `cast.json`.

**The trait slot is optional where the kit is not**, and the asymmetry has a
reason rather than being a convenience. A unit that brings no skills cannot act,
so an empty kit is never something anybody chose; a unit that brings no trait is
an ordinary unit, so an empty trait slot is a decision like any other. Insisting
would make "I want the plain version" unwritable.

**The engine learns none of this.** `battle.Roster` still takes a resolved kit
and a resolved stat line, because a learnset is settled *before* a battle exactly
as an evolution already is. What changed is the authoring layer and the
placement — and the log.

#### The log carries the placement

`battle.Log` now records the roster it was fought with, and `--verify` rebuilds
from that rather than from the embedded data.

It had to. While a roster came out of the data and nothing about it was decided,
re-running a log meant loading that data again and the two were the same battle
by construction. Now that a placement picks four of nine, a log that did not say
which four could not be re-run at all — `--verify` would compare two different
battles and report the difference as corruption.

It carries the **resolved** form rather than the reference, which buys something
the reference never could: a log is now readable across a data edit. Retuning a
stat curve or moving a skill's learn level no longer invalidates every log
written before it.

⚠️ A log written before this **renders exactly as it always did** — that reads
the events and nothing else — and refuses to verify, saying why. Re-running
today's roster against it and calling the mismatch corruption would be a verifier
lying about which of the two was wrong.

#### What four slots actually cost, measured

The design note predicted a level-one unit idling about a third of its turns, and
that is right for the unit it described. Across four thousand battles of the
*shipped* roster the figure is **2.2 per cent** of turns spent with nothing
usable, because the youngest unit on that roster is level eight with three skills
rather than level one with two — and the rest are grown units with four good
ones. No battle failed to finish; the longest ran 63 turns, which is where it was
before.

The roster reads **49.5 per cent** to the ally, down from 51.9. Cutting nine
skills to four took most of the swing `venom_blood`'s reply had added, and it
took it from the ally: the unit that loses most by choosing four is the one that
had the deepest kit to choose from.

### Choosing to evolve

Built. A level **allows** a form rather than dictating one, and the placement
says which one it fielded.

```json
{ "id": "ally.ivysaur", "character": "pokemon.bulbasaur", "level": 60,
  "stage": "Ivysaur",
  "skills": ["razor_leaf", "sleep_powder", "leech_seed", "sludge_bomb"] }
```

`Line.Resolve(level)` became `Resolve(level, stage)`. `progression.Furthest` is
what a caller passes when it is not choosing — a screen showing a character at
level 30 has no placement behind it — and it is the behaviour every caller had
before, so a roster written earlier still says what it always said.

Naming a form the level has not reached is **refused rather than clamped**: a
clamp would field a different unit from the one written down, which is the one
outcome worse than saying no. The two refusals are told apart, because they are
different mistakes — a name the line does not answer to is a typo, and a name
merely ahead of the level is a placement that has not grown into it.

#### Why the choice is not decorative

Stage curves only rise, so fielding an earlier form is fielding a weaker unit. It
is a decision only if the earlier form can hold something the grown one cannot,
so a learnset entry gained a second gate:

```json
{ "id": "sleep_powder", "at_level": 12, "stages": ["Bulbasaur", "Ivysaur"] }
```

**The stage gate is an allowlist, not a threshold, and that is the whole of it.**
A threshold could only ever say "from this form onwards" — so everything an early
form knew a grown one knew too, and giving up an evolution would buy nothing. A
list can name one member of a class, which is the same reason `skill.Restriction`
is an allowlist.

So `sleep_powder` is Bulbasaur's and Ivysaur's, and Venusaur never gets it.
Fielding Ivysaur at level 60 keeps a control skill and gives up the ace's stat
line. That is the trade, and it is now expressible in a file rather than a
paragraph.

⚠️ **The shipped roster does not take it.** Every unit is fielded as the furthest
form its level reaches, so the balance figure is unchanged at **49.5 per cent**
and `replay.golden` did not move — the mechanism exists and the roster has not
been retuned to want it. Choosing an earlier form on that roster today is simply
worse, because none of the three characters is close enough for one kept skill to
pay for a whole stage of stats. That is a cast-tuning question rather than a
mechanism one.

#### `at_stage` was not built, and is not needed

The design note asked for `at_stage` — a threshold — and said it could not be
built until a placement named its form. It could be now, and it is not, because
the allowlist above says everything it would have said and one thing more:
`at_stage: "Ivysaur"` is `stages: ["Ivysaur", "Venusaur"]`, while the case that
makes evolution a decision has no spelling as a threshold at all. Two fields
would have been two vocabularies for one idea.

**Conditions beyond a level are deliberately out of scope.** Items, a friendship
count, a number of battles fought: every one needs somewhere to persist between
battles, and there is no such place — no meta layer, no inventory, no save. A
level is what a character sheet knows.

### A regeneration that heals

Built. `regrowth` was declared, glossed, described and inert: `Battle.inflict`
computed a tick only for `status.Dot`, so a `Regen` stack went on carrying nought
and every step below it — `Set.Tick`, the per-status loop, `heal` — was already
written, already correct, and never reached.

The whole fix is one branch:

```go
case status.Regen:
    tick = b.books.Rules.Restore(b.Stats(actor)[from.Scaling], kind.TickPower)
```

**`Restore`, not `Damage`, and it drops two things on purpose.** No defence curve,
because `combat.Rules.Restore` already records why — armour turns away what is
coming *at* a unit and has nothing to do with what is helping it, so dividing here
would let a unit's own armour quietly weaken its own regeneration. And no
elemental multiplier, because the chart prices what one creature threw at another:
a grass unit healing a fire ally is not throwing anything, and reading the chart
would make the same cast worth two thirds of itself for a reason written on
neither of them.

What it keeps is the actor's scaling stat and the freeze, both the same as a
damage-over-time's. The freeze is the point, and it is what `status.Regen` already
promised and nothing was honouring: two casters stacking one regeneration each
contribute what their own attack was worth at the moment they cast.

#### The correction

The note this replaces said `aqua_ring` and *the healing half of `synthesis`* did
nothing. `synthesis` was never affected — it heals through `restores`, which
always worked. The two dead skills were **`aqua_ring` and `ingrain`**, and neither
had a working half: both are power 0 with no `restores` and nothing but the
regeneration, so casting either did *nothing whatsoever*.

#### One event, not two

Fixing the tick surfaced a second bug behind it. `tickStatuses` named the status
that healed and then healed from the total underneath, so every regeneration tick
would have logged **two** `Healed` events — a reader adding the amounts up would
have had every regeneration worth twice what it is. Nothing had caught it because
no regeneration had ever ticked.

Healing is now applied one entry at a time and `heal` carries the status id, so a
tick is exactly one event that says what healed, how much landed, and the health
it left behind. Damage stays resolved from the total, because `wound` emits
nothing and has no name to carry. The per-entry form is also the truthful
arithmetic: `heal` stops at full health, so two regenerations worth more than the
room between them have the second clamped, which a single total would have hidden
behind one number.

#### The two decisions the note asked for

**Which stat a regen scales off**: the applying skill's, read live off the actor
at the moment of application — the same expression the damage-over-time branch
uses two lines up. A heal scaling off attack is what `Skill.Restores` already
does.

**Whether an amplifier raises a regen**: no. The refusal in `passive.Amplification`
predates this and its stated reason has now expired — it used to be that accepting
the share would promise an author a multiplication of zero. It stays refused on
the ground that was always the stronger of the two: the share is described to a
player as *"its poison ticks 30% harder"*, in both languages, and a share that
heals under that sentence is a description that lies — which is the one thing
every derived description in this engine exists to prevent. Lifting it is a
wording change first and a one-line condition second, and worth doing when a trait
actually wants it.

#### No golden moved, and that is the finding

The note expected a balance diff. There is none: **no unit in the shipped roster
fields either skill**, and `Suggest` picks the highest expected damage and falls
back to the first usable non-damaging skill, so a self-cast regeneration on a unit
that can always reach somebody is one the bench never chooses. The two facts
together are the whole reason a shipped skill could do nothing for this long
without a single test noticing.

So the proof is hand-played. `TestTheShippedRegenerationHeals` casts `aqua_ring`
from the shipped books and reads the number back; eight tests in
`internal/core/battle` cover the tick, the freeze, the two dropped terms, the
clamp, the single event and the order healing resolves in. Putting a regeneration
into the roster is a balance decision and belongs with the cast work, not with a
bug fix.

⚠️ One of those eight was worthless when first written. The order test asserted
that the `healed` event came before the `status_ticked`, and a mutation putting
damage first **survived** — `wound` emits nothing, so the two events come out in
the same order either way and only the survivor changes. It now asserts survival:
a unit on 150 health with a poison worth 200 and a regeneration worth 640 is
standing at the end of the turn, or the two totals resolved the wrong way round.

### Summoning: a skill that puts somebody on the board

Built, and shipped as an engine with nothing in the cast using it yet — the first
mechanism here that adds a combatant rather than doing something to one already
standing.

```json
{ "id": "shadow_clone", "target": "self", "range": 0, "power": 0,
  "accuracy": 1000, "cooldown": 5,
  "summons": { "count": 2, "name": "phân thân", "share": 500,
               "skills": ["shuriken"], "lasts": 3, "bound": true } }
```

**Three spellings of the stat line, and exactly one per skill.** A clone and a
called-up creature are different things wearing one mechanism. `share` is a
share of the caster's stats **as they stand**, so a caster that buffed itself
first makes a better copy; `share_of_base` ignores every timed effect, which is
what an author reaches for when a copy that can be set up beforehand is an
exploit rather than a play; `stats` is a line of its own, because a toad does not
get bigger when the ninja levels. Forcing one spelling would mean either a
creature scaling off somebody it has nothing to do with, or a clone re-authored
every time its caster's curve moves.

Either share is **frozen at the cast**, the same freeze a damage-over-time's tick
takes: a copy is a copy of what was there.

**Three ways off the board.** It can be killed like anything else; `lasts` counts
**its own** turns, for the reason a cooldown counts the caster's — a slow summon
and a fast one given three turns should each get three; and `bound` sends it home
when whoever summoned it dies, which is a per-skill flag rather than a rule
because a clone is an extension of its caster and a creature that was called up
is not.

**A summon counts.** `checkEnd` sees it like any other unit, so a side holding
nothing but a clone has not lost. The alternative is a unit the win condition
cannot see, and a player watching a battle end with somebody still standing.

**It goes through `enlist`,** which is every rule about what may stand here: a
real formation slot, a free cell, a side inside its strength, a stat line inside
the progression limits, skills that exist and an element that may carry them. A
summon that built its own `Unit` would be a second answer to all of it, and the
first thing to diverge would be the one nobody tests.

**Nothing about it is in the log.** It is derived — the caster, the skill, the
board and a counter on the caster — so re-running the same decisions from the
same seed puts the same units in the same cells under the same ids. That is why
the id is built in the engine and not passed in: an id a caller chose is a fact a
log would have to carry, and `--verify` would be comparing two different fights.

⚠️ **A fallen unit keeps its slot; a departed summon does not.** The formation is
what a roster wrote down, so a side authored with three units in three named
slots is that arrangement for the whole battle and a summon appearing in a dead
comrade's cell would be a placement nobody chose. A summon was never in that
arrangement — it borrowed a slot the formation left empty — so when it is gone
the slot is empty again.

That second half is what makes a summoning skill something a unit can do twice.
Counting a departed summon would kill a repeatable skill quietly: the shipped
formations leave **two** free slots a side, so the third cast of a battle would
put nothing down and say nothing about it, and a `cooldown: 5` clone in a
forty-four turn fight would work for the first ten minutes.

⚠️ **The cell is reusable and the id is not.** The counter that names a copy is on
its caster and never resets, so two copies standing in the same place at
different times are still two units — an id is what a decision in the log names,
and one reused would make two of them the same row.

⚠️ **Front column first.** `hex.Place` puts the highest formation column against
the enemy for both sides, so walking columns forward drops a copy where a range
of one can reach somebody. Walking them backward — which is what `range
FormationCols` does — puts every summon at the far edge where most kits cannot
aim at all, and the mechanism looks broken for a reason nowhere near it.

⚠️ **A summon may not summon**, and the check needs a second pass over the
finished book: the skill being summoned may be declared below the one summoning
it, so a rule applied while reading one entry has nothing to look up yet. Without
it a single cast is unbounded — the board would stop it in practice, and "it runs
out of room" is not a rule anybody can read off the file.

⚠️ **`Suggest` will not cast one.** It takes the highest expected damage and falls
back to the first usable non-damaging skill, so a skill whose whole effect is
putting somebody down is one autopilot takes only when it can find nothing to
hit. `summoned` and `left` are proved reachable by a hand-played battle, the same
answer `passive_released` needed.

⚠️ **The test that nearly was not one.** A first draft of the vacated-cell test
set a copy's health to nought and called that a death — it is not one, nothing
reads health looking for a corpse, and `kill` is what makes a unit dead. The
board therefore never had a vacated cell on it, and a mutation freeing vacated
cells passed. It is driven through a summon running out of turns now, which is a
departure the engine actually performs.

### Growing the cast

The tooling for this exists — see *Authoring a cast* above — and so does the
thing it was blocking: three characters ship, one per element, and the seed
roster is no longer a mirror, so balance is measurable. What remains is content
rather than a design question, and two constraints shape it.

- **An archetype's kit constrains a character's affinity.** `battle.New` refuses
  a unit carrying a skill of an element it does not share, so a preset's kit
  decides what a character built from it may be. `skill.CanCarry` is the single
  declaration of that rule and `hexforge` applies it while you author, so it is a
  design constraint rather than a trap. A preset mixing three elements' skills is
  rejected outright, because no affinity can hold three.
- **The stat budget is the other one.** `progression.Limits` bounds each stat and,
  separately, bounds health and defence *together*, because those two multiply
  rather than add: a unit at both ceilings is not merely durable, it is durable
  squared. The shipped presets spend between 4036 and 11397 of the 11500
  effective-health budget, and `hexforge show` prints what is left.

Every character added moves `scenarios.golden` and `replay.golden`. That is the
point rather than a cost: those diffs are how a balance change gets read.

Squirtle is worth reading before adding another. Water is the strongest of the
three elements and Blastoise still loses the ace duel on stats, because its
attack and speed curves are the lowest in the cast — an element advantage does
not carry a passive stat line.

A third constraint arrived with species, and it is a *softer* one: a skill kept
for a lineage asks the character to be one, so adding a dragon is adding two lines
rather than one — the kind in `species.json` and the claim on the character. See
*What a unit is, not only how it fights* above.
