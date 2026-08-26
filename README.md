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
modest: over four thousand auto-battles, taking the pierce off moves the ally win
rate from 48.5 per cent to 46.5. The damage table was telling the truth — this is
a dial that changes who armour is good against, not one that decides battles.

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

**A gated `grants` is refused**, rather than accepted and quietly ignored. A grant
is applied once, when the unit is enlisted, and the status it puts on is permanent
precisely so nothing can take it off — so a gate on one would have to add and
remove that status as health crossed the line. That is a mechanism rather than a
term, and it is the last piece: see *What a passive still cannot do*.

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
- **There is no `at_stage`.** It would read as a different fact and today it is
  not one: `Line.StageAt` derives a stage from a level, so `at_stage: "Ivysaur"`
  *is* `at_level: 16` and nothing else. It becomes a second fact only once a
  placement names the stage it fielded — see *Learnsets, slots, and choosing to
  evolve*.
- **The gate is applied in exactly one place**, `seed.ParseRoster`, because that
  is the only place a character and a level meet. A flat roster entry writes out
  its own traits and has no level to be gated against; the engine is handed a
  resolved list exactly as it is handed a resolved stat line.
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
At the prompt, `?2` asks about the second skill offered and `?A1` asks what a
unit is carrying; both print and then draw the menu again, because reading is
part of deciding and cannot cost a turn.

```
 2) razor_leaf      grass      rng 3   arc_up        pow 1200   acc 880   x2 cd2
> ?2

  razor_leaf · phi diệp
  ----------------------------------------------
  Đánh 3 ô đối phương, 2 nhát, mỗi nhát 60% công (tổng 120%), xuyên 40% giáp.
  Tầm 3 · 88% trúng · hồi 2 lượt.
```

**Every word of it is derived from the skill.** Nothing is authored, and that is
the whole design rather than an economy: a written description is free to drift
the moment a value moves, and a line reading "doubles" survives a bonus dropping
from 1000 to 700 with nothing to catch it. This engine made that trade once
already and refused it — `Archetype.Demands` is computed from a kit precisely so
it cannot describe a kit it no longer has. The price is a uniform voice, and a
flat sentence is worth more than a wrong number.

`testdata/describe.golden` holds every shipped skill's description, so a number
moving in `skills.json` moves a line there and the diff says how the change reads
to a player. It is the balance record from the other end: `skills.golden` says
what a skill is worth, this says what it sounds like.

**Shares rather than damage figures.** "100% of attack" is true wherever it is
read; a damage number is true for one caster against one target and stops being
true at the next buff. A player comparing two skills is comparing the shares.

**Vietnamese, alone on an English screen.** The rest of this package is English
and stays that way until the whole battle screen is translated in one piece
rather than a paragraph at a time. This block is the one part read to *decide*,
so it is worth being legible before the rest catches up, and the mixed screen is
a stated cost. It borrows `i18n`'s gloss tables rather than growing a second set
of Vietnamese names, because two vocabularies for one status is how they drift.

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
`go test ./internal/core/hex ./internal/seed ./internal/tui -update` to accept a
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

Now it measures. Over four thousand auto-battles the roster sits at **48.5 per
cent** to the ally, and taking `razor_leaf`'s pierce back off moves that to 46.5
— a real, one-directional move, on a roster where the ally holds the only
Venusaur and the enemy the only Blastoise. The 40-seed sweep the test runs reads
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
- **All three forms and both trait states are in play.** The levels span the
  final, middle and base stage; Wartortle and Ivysaur sit exactly on
  `endurance`'s unlock level, and Charmander at 8 is below `blaze`'s — so a
  battle exercises a unit holding its trait and a unit that has not earned one.

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
go run ./cmd/hexforge skills add oath --power 1200 --accuracy 900
go run ./cmd/hexforge skills edit oath --power 1100   # change one already in the book
go run ./cmd/hexforge cast             # the authored characters
go run ./cmd/hexforge new              # create a character
go run ./cmd/hexforge show some.id --level 30
go run ./cmd/hexforge check            # parse from disk, verify the art, report the budget
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
new-character form, the origin catalog with an add form, and the check rendered
as a screen. Two things it can do that a sequence of prompts cannot — because
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
  compared, which is the whole reason a form may have art of its own.

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

### What a passive still cannot do

All four of the things a passive is normally asked for are built — a stat change,
refusing a status, adding to what the holder does, and waiting until it is hurt.
One combination of two of them is not, and it is the one the canonical abilities
want.

**A gated grant: a stat change that comes and goes.** *Overgrow* and *Blaze* are
"hits harder when badly hurt", which is a stat change under a condition, and that
pairing is refused at parse rather than accepted and ignored. A grant is applied
once, at enlistment, and the status it puts on is permanent so that nothing can
dispel a trait — so gating one means adding and removing that status as health
crosses the line. What that needs, having been worked out rather than guessed:

- a door into a permanent status **for the engine only**. `Set.Remove` must keep
  refusing them, or a dispel could turn a trait off for the rest of a battle; a
  dedicated pair — `Hold` and `Release`, used by `battle.grant` and by the
  re-evaluation — leaves the guard meaning what it says.
- **an event each way.** A trait coming on or going off changes a visible number,
  and the log is the only contract a renderer has. `PassiveHeld` says it came on;
  going off has no kind yet.
- **a retune each time**, because a gated trait touching speed reorders the queue,
  and `retuneAll` is what keeps it honest.
- **one re-evaluation point.** Health moves in `wound` and `heal` and nowhere
  else, so those are the two places — for the unit whose health moved, not for
  everybody.

None of that is large. It is a mechanism rather than a term, which is why it is
its own piece rather than a clause of this one.

Whatever comes next keeps the two constraints the first slice was built under: a
trait that changes a number **emits an event**, and one that touches speed is in
force before the first wait is computed.

### Amplifying a status, which is two features

A trait that "makes its poison better" is two different things, and they land in
two different places:

- **The effect.** A stronger tick. The tick is computed once, in `battle.inflict`,
  from the applier's scaling stat, and frozen on the stack for its whole life — so
  an amplifier here folds into that one multiplication and nothing later has to
  know about it.
- **The chance.** Poison that lands more often. That is the *same site a
  resistance already bites*, so the two meet naturally: one side's trait raises
  the chance and the other's lowers it, and because a resistance already composes
  multiplicatively the order they are applied in cannot matter.

They are separate because a trait may plausibly want either without the other,
and because they read differently in play: a stronger tick is worth more the
longer a stack lives, while a better chance is worth more the more often the
skill is cast.

Three things any implementation has to do, and the third is the one that gets
skipped:

1. A field on `passive.Passive`, per status, the way `Resists` is.
2. The multiplication — into the tick for the effect, into the chance for the
   chance.
3. **The amplification has to reach the event.** `status_applied` already carries
   the frozen tick as `amount`, so an amplified poison shows up as 260 where it
   would have been 201 — and nothing in the log says why. That is exactly what
   `Pierce` on `damaged` and `Refused` on `status_applied` exist to prevent, and
   it is the third time the same trap has come up: the mechanism is the easy half.

**This one reads the *applier's* traits at a site that currently reads the
target's.** `Battle.resist` walks the target's passives; an amplifier walks the
actor's. Both at once in `inflict`, which is worth knowing before writing it,
because every other trait so far has read exactly one unit.

And the mirror of it is cheaper than it looks: a **vulnerability** — this unit is
*easier* to poison — is `Resists` with a negative share rather than a new field.
The validation bounds it at 1..1000 today, and opening the range downwards is a
one-line change that reuses the whole composition.

One thing already true and worth stating, because it limits what can be
attributed after the fact: a `Stack` remembers its frozen amount and nothing else.
It does not know who applied it, deliberately — the applier may be dead by the
time the stack resolves, so keeping the id would be keeping a pointer to something
that no longer exists. So `status_ticked` names the unit *taking* the damage, not
the one that caused it, and the only place the source is recorded is the
`status_applied` event. Two units poisoning the same target leave two stacks that
the state cannot tell apart.

### Answering back

The four usual jobs are the four above, and there is a fifth the shipped data
already promises. `venom_blood` is *máu độc*: blood that is poisonous does not
only refuse poison, it should cost whatever bit into it. Resisting is half the
name, and the half that was cheap.

It is not `applies` wearing a different hat. That one hands the holder's own
attack an extra application, so it fires on a target the holder chose, at a moment
`battle` is already resolving. A reply fires on the **attacker**, during somebody
else's turn, from a unit that is not acting — there is no hook for that, and
inventing one is most of the work. What it must not become is a second damage
path: the reply resolves through `battle.inflict` and `combat.Rules` like every
other effect, so a replay reads it as events and `--verify` re-runs it from the
seed.

Four rules, decided rather than left to whoever builds it.

- **A reply may kill.** It is damage and gets no exemption for arriving out of
  turn, so a battle can end on a turn nobody took. Whatever resolves a reply
  re-asks whether the battle is over, and the timeline loses a unit that was never
  in front of it. A damage-over-time tick already ends a battle, so the shape
  exists; what is new is that the unit dying is the one taking the turn.
- **A reply never triggers a reply.** Closed by rule and not by a depth counter,
  because a counter is a number somebody will raise: a reply resolves with
  retaliation off, so a trait cannot answer a trait and two holders facing each
  other settle in one exchange instead of trading until the arithmetic runs out.
- **A reply answers a use of a skill, not a strike.** Otherwise a trait's worth
  scales with somebody else's strike count, and `fire_fang` is quietly worse than
  `flamethrower` into one holder for a reason written on neither skill.
- **The holder takes every strike first and answers afterwards.** The reply is on
  the finished skill rather than partway through it, so a striker never dies
  mid-turn and the question of what happens to the strikes it had left never
  arises.

That last one leaves the holder's death as the only one that can land first, and
dead is dead: **a holder killed by the skill does not answer**, the way it cannot
be healed. Which hands retaliation a counter without anybody designing one —
killing the holder outright is how a reply is avoided, so a trait that punishes
attacking rewards hitting hard instead of taxing everybody equally.

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

What is still absent is the **gradient**: the move that hits harder the further the
*caster* has fallen. It is a multiplier on power rather than a bonus to it, belongs
in `combat`, and reads the other unit — two features, not one with an option.

### Learnsets, slots, and choosing to evolve

A character knows every skill it will ever know, from level one, and brings all
of them. The same is now true of its traits. That makes a level cap the only
thing separating a young unit from a grown one, and it means authoring a ninth
skill — or a second trait — makes a character strictly better rather than
presenting a choice.

**Skills and traits are one mechanism here, not two.** Both are "declare many,
unlock by progression, bring some", and building a second unlock system for
traits would be two vocabularies for one idea — the mistake this repository keeps
a list of. One `{id, at_level}` shape, one validator, one "what is available at
level N" function, and two lists using it: a kit with four slots and a trait
list with **one**.

Four pieces, and they are one mechanism:

- **A learnset.** A character declares *when* it learns each skill and each
  trait rather than simply holding a list: `{id, at_level}`, or `{id, at_stage}`
  for one that only a later form can hold. A stage-gated entry is gated behind
  whatever that stage requires, transitively, which is what makes evolving unlock
  something rather than merely raise numbers.

  ⚠️ **`at_stage` cannot be built before chosen evolution**, and building it
  early would be worse than not building it. `Line.StageAt` derives a stage from
  a level today, so `at_stage: "Ivysaur"` is *exactly* `at_level: 16` and nothing
  else — a second spelling of one fact, which is what this codebase refuses
  everywhere. It becomes a different fact only once a placement names the stage
  it fielded.
- **Four slots, and one trait slot.** A placement brings at most four of the
  skills the character has learned and at most one of its traits. Refused at load
  if it names something the character has not learned at that level, if it names
  too many, or if it names one twice. A trait slot is what turns a pile of traits
  into a decision, for the same reason four skill slots do — and it is worth
  little until there are traits worth choosing *between*: with only stat traits
  built, a slot picks between three numbers.
- **Gating is separable from slots, and much cheaper.** A first slice can gate
  traits by level and bring **every unlocked one**, which is not a choice and so
  needs no change to the log. The slots are the expensive half, and they are
  where the log work below is paid for — once, for both lists.
- **Evolution is chosen, not derived.** Today `progression.Line.StageAt` reads a
  stage out of a level, and there is no decision in it. Reaching the threshold
  should instead *allow* a stage, and the placement names which one it fielded, so
  `Resolve(level)` becomes `Resolve(level, stage)` with the chosen stage's
  threshold no higher than the level.
- **Conditions beyond a level are deliberately out of scope.** Items, a
  friendship count, a number of battles fought — every one of those needs
  somewhere to persist between battles, and there is no such place: hexarena has
  no meta layer, no inventory and no save. A level is what a character sheet
  knows. An item system is its own future thing, not a clause of this one.

So the order to build it in, and the reason for it:

1. ~~**Gate the traits.**~~ **Done** — see *A trait comes in at a level*.
   `cast.Unlock` is the shape the slots will need, settled and tested first on the
   smaller of the two lists.
2. **Then the missing three passive jobs** — see *What a passive still cannot
   do*. This one is here rather than last because a slot between three stat
   traits is not a decision, and a resistance or a conditional trait is. Building
   the slot first would ship a choice with nothing to choose.
3. **Then the slots**, four skills and one trait, and the log carrying the
   placement. The expensive half, paid once for both lists.

Step 2 sits in the middle on purpose: it is the only one of the three that is
*not* about this mechanism, and it is what decides whether the mechanism is worth
having.

Three things this has to face.

**The log has to carry the placement.** `battle.Log` holds a seed, the decisions
taken and the events produced — and nothing else. The roster comes from the
embedded data, so a log is already only valid against the build that wrote it.
Once the loadout, the trait and the stage are *choices*, a log without them cannot
be re-run at all, and `--verify` would be comparing two different battles. Writing the placement into the log is
part of this change rather than a follow-up, and it has the side benefit of
making a log portable across a data edit, which today it is not.

**Four slots is a per-turn nerf, and cooldowns set its size.** A skill on
cooldown *N* is usable every *N+1* turns, so it contributes `1/(N+1)` actions per
turn. Measured against Bulbasaur's own cooldowns:

| loadout | actions per turn | turns idle |
| --- | ---: | ---: |
| four attacks, cooldowns 1, 2, 2, 2 | 1.50 | none, a third wasted |
| four support skills, cooldowns 3, 4, 4, 4 | 0.85 | 15% |
| level one, two skills, cooldowns 1 and 4 | 0.70 | **30%** |

So the mechanism works: a level-one unit spends about a third of its turns unable
to act, which is what being young should feel like. But it **compounds the
stalemate gap above** — more turns with nothing usable is more chance of a battle
that cannot end — and it makes a low-cooldown basic close to mandatory, which is
why every game of this shape has a move you can spam. `strike`, at cooldown zero,
was exactly that and has been retired; a four-slot world probably needs a
successor to it.

**Choosing not to evolve is currently strictly worse.** Stage curves only rise,
so a placement that fields an earlier stage fields a weaker unit for no
compensation, and the choice is decorative. It becomes a real decision only if an
earlier stage can hold something a later one cannot — an earlier learn entry the
grown form never gets. That is worth deciding deliberately rather than
discovering: as it stands, "may evolve" and "does evolve" are the same thing.

The engine learns none of this. `battle.Roster` keeps taking a resolved kit and a
resolved stat line, because a learnset and a stage choice are settled *before* a
battle, exactly as evolution already is. What changes is the authoring layer and
the placement, and `hexforge` gains a reason to show a character's learnset by
level and to refuse a loadout the level cannot hold. Choosing the four is a
player's decision, and until there is something to make it with, an automatic
battle may take any four.

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
