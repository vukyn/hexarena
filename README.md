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
go test ./...
```

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

Kinds and sides are written by name, not by number, so inserting a constant later
cannot silently reinterpret every log already saved.

## Layout

```
cmd/hexarena/          terminal client: all input and output, no rules
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
| `roster.json` | a seed roster to exercise the engine with |

Changing a number there changes the game without touching Go. The tests will tell
you what moved: several of them freeze design figures deliberately, and the golden
files under `testdata` are a record of what the numbers currently produce. Run
`go test ./internal/core/hex ./internal/seed ./internal/tui -update` to accept a
change, and read the diff — that diff is the point of them.

## Roadmap

### Graphical client with ebiten

The event log is the contract, and `--verify` proves it is a faithful one, so a
graphical client is a renderer over the same log rather than a second
implementation of the rules. `internal/tui` is the reference for what that means
in practice: it never reads the battle, only the events.

Open questions before starting:

- **Asset pipeline.** Unit art is meant to be SVG. Ebiten cannot draw SVG, so
  either bake to PNG at build time or rasterise at load with `oksvg` and
  `rasterx`. Decide before any art exists, not after.
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

### A real cast

`roster.json` is a test bed, not a cast: ten units with flat stats, chosen to
cover every affinity and every mechanic rather than to be interesting. What is
missing:

- **Evolution lines.** `progression.Line` supports staged growth and nothing uses
  it yet — every seed unit is a single stage with flat numbers. A real unit
  declares a curve per stage and the level selects one.
- **Art.** Unit avatars are meant to load from SVG, which is the same pipeline
  question the graphical client faces.
- **Biography and identity.** None of the seed units have anything but a name.

The stat budget is the constraint to design against: `progression.Limits` bounds
each stat and, separately, bounds health and defence together, because those two
multiply rather than add. A unit at both ceilings is not merely durable, it is
durable squared.
