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

The full-screen authoring client is built on bubbletea, with bubbles and
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
| `archetypes.json` | the role presets: a suggested stat curve and kit per role |
| `cast.json` | the authored characters, each with an evolution line |
| `roster.json` | a seed roster to exercise the engine with |

Changing a number there changes the game without touching Go. The tests will tell
you what moved: several of them freeze design figures deliberately, and the golden
files under `testdata` are a record of what the numbers currently produce. Run
`go test ./internal/core/hex ./internal/seed ./internal/tui -update` to accept a
change, and read the diff — that diff is the point of them.

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
`--id --name --origin --archetype --image --element --bio --skills` and the
per-stat `--hp --atk --def --spd --acc --ddg` overrides (written `base:max`) turn
it into a one-liner. Choosing an archetype fills every curve and the kit, and each
prompt shows the preset as its default. The kit is asked before the element,
because the kit is what decides which elements are legal. Before writing, it
prints the resolved level 1 and level 60 lines with how much of the
effective-health budget they spend; `--yes` skips the confirmation. Every answer
is checked as it is entered, against the same parsers the game loads through — the
tool knows no rules of its own, which is why a character it writes is a character
that loads.

`hexforge skills add` and `hexforge skills edit` take the same flag names, and
the difference between them is what an absent flag means. On `add` it is a
question the wizard asks or a default it takes; on `edit` it means leave the
field alone, so `--cooldown 0` sets a cooldown to zero, no `--cooldown` leaves
it, and `--restrict-elements ""` clears a list. A skill's id cannot be edited —
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
beyond reading a data file: it verifies that the art each character names is
really there. `internal/core/cast` checks only the *shape* of an image path
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
  typed.

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

That is possible because `internal/forge` hands over *facts* rather than
sentences: a refused kit arrives as a value carrying the affinity, the skill and
the skill's element, and `internal/i18n` turns it into
`hệ fire không mang được chiêu "sever" (hệ metal)` or into
`fire cannot carry the skill "sever", which is metal`. `cmd/hexforge-tui` holds
no wording of its own — a test greps its source to keep it that way. Element
ids, skill ids and the stat labels `hp atk def spd acc ddg` are the same in both
languages on purpose: they are what you type and what the data files store.

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

### Passive skills

Every skill today is active: a unit spends its turn on one. A **passive** — what
a character *has* rather than what it uses — was part of the original design and
the engine has none. Four things a passive is normally asked to do, and where
each already has a home:

- **Raise or lower a stat.** `status.Kind` already carries `Modifiers`, and
  `modifier.Set` already saturates every term of the same target together. A
  permanent status granted when the unit is enlisted covers this with no new
  mechanism, and inherits the saturation for free — which is the point. A passive
  that *composed* with temporary buffs instead of saturating alongside them would
  be the one place in the game where stacking explodes.
- **Add an effect to what the unit already does.** `skill.Application` is the
  existing shape for "this inflicts that, at a fixed chance". A passive that
  contributes one needs the application list assembled from the unit as well as
  from the skill, which is a change in `battle`, not a new rule.
- **Fire only under a condition.** `skill.Condition` is the existing vocabulary:
  a status, a minimum stack count, a bonus, and whether it consumes. Reuse it.
  Two ways of writing "while the target is burning" is how the two drift apart.
- **Immunity.** The only one of the four with nowhere to live. Nothing can
  currently refuse an application. `status.Set.Apply` is the choke point every
  status passes through, and `status.Category` with its `Harmful()` split already
  gives immunity something to be expressed against. The open questions are
  whether it keys on a category or a status id, and whether it is absolute or a
  saturating resistance — absolute immunity is a hard cap on a continuous
  quantity, which everywhere else in this engine has been the wrong choice.

Constraints any implementation has to keep:

- **A passive that changes a number must emit an event.** The log is the only
  contract a renderer has, so a silent passive makes the log lie: a reader sees a
  damage figure the visible numbers cannot account for. This is the trap worth
  naming in advance, because the mechanism is the easy half and the logging is
  the half that gets skipped.
- **An enlist-time passive has to land before the first wait is computed.** Turn
  order is `1_000_000 / speed`, so a passive touching speed must be in place
  before the queue is built or turn one is already wrong. `retuneAll` exists
  because exactly this was got wrong once with haste.
- **No new randomness.** A passive that rolls, rolls through the `*rng.Source`
  that was passed in, and the roll has to reach the log — otherwise `--verify`
  fails against a replay of the same seed, which is precisely what it is for.
- Durations and cooldowns count in the holder's own turns. Parts per thousand,
  never floats.

Where a passive is *declared* is open. The natural shape is a `passives.json`
book validated cross-book the way skills are, a `Passives []string` field on
`cast.Character`, and an archetype able to suggest one — which would also give an
archetype its first mechanical weight, since today it carries none.

### Healing, draining, and a regeneration status

**Nothing in the game restores health.** `battle.wound` subtracts and no function
adds; `status.Set.Tick` returns a single unsigned damage total; `status.Category`
has damage-over-time, stat debuffs, control, buffs and shields, and no fifth kind
that heals. That absence is why Bulbasaur has no Leech Seed: it drains, and there
is nothing for it to drain into.

Three separate mechanisms are wanted, and they are not one feature:

- **A direct heal** — a skill that restores health to an ally or the caster.
- **A drain** — the attacker takes back a share of the damage it dealt. It has to
  read the damage **dealt**, `combat.DamageDealt`, not the damage rolled, so a
  strike that missed or was blocked drains nothing.
- **A regeneration status** — health returned each turn, the mirror of a poison.

The third is the cheapest and the most useful, because the damage-over-time
machinery already does the work: `status.Kind.TickPower` and the per-stack freeze
at application are exactly what a regeneration needs, applied with the sign the
other way.

What an implementation has to settle, in the order these will bite:

- **`Set.Tick` must return healing separately, not a signed total.** It returns
  one unsigned number today and `wound` consumes it. A negative damage travelling
  down that path would subtract a negative and could revive a corpse, because
  `wound` calls `kill` the moment health reaches zero.
- **A dead unit cannot be healed, and health clamps at `Unit.MaxHP`.** The first
  keeps a battle able to end; the second is what stops a regeneration from being
  a shield with no cap. Whether overheal spills into something like a shield is a
  design question, not a default.
- **Healing does not go through the defence curve.** Damage over time does — that
  is what "damage over time goes through armour" settled, *through the formula* —
  and healing must not, because defence turns away what is coming at a unit and
  has nothing to do with what is helping it. Do not add the division for
  symmetry's sake; the asymmetry is the point.
- **A heal must emit an event.** None of the seventeen event kinds explains
  health going *up*, so a renderer would draw a number changing with no cause in
  the log — the same trap a passive skill has, and the same answer: if the log
  cannot explain it, the log is wrong.
- **It breaks the stat budget from the other side.** `MaxEffectiveHP` bounds
  health and defence together because they multiply. A unit that heals is durable
  beyond its stat line, so the bound becomes an understatement — exactly the
  mirror of what piercing does to it. With both open, one figure describes
  neither case, and the tools' effective-health row has to say which it means.
- **`battle.Suggest` will not use one.** It maximises expected damage, and a heal
  has none, so it lands in the fallback beside the buffs it already never
  chooses. That is the same gap *A deeper opponent* records, and it is worth
  fixing there rather than here.

The status itself is authored in `statuses.json`, which is hand-edited like
`archetypes.json` — the tool does not write it. A new category also needs its
wording in both languages, since the category catalogue is held complete rather
than falling back to its enum spelling.

Determinism is unaffected: this is integer parts-per-thousand arithmetic with no
new randomness, which is what makes it cheap to add later.

### Piercing, as the answer to armour

Nothing in the game currently ignores defence. Every damage source goes through
the same curve — a normal hit, a splash hit, and a damage-over-time tick all
divide by `K + defence`, which is what "damage over time goes through armour"
settled: through the *formula*, not around it. So the effective-health figure
`hexforge` reports is exact today rather than an estimate.

That leaves the armour end of the stat budget without a counter. `progression.Limits`
caps health and defence *together* because they multiply, and two very different
units can sit at the same cap:

| | health | defence | absorbs |
| --- | ---: | ---: | ---: |
| sentinel | 3100 | 800 | 11397 |
| bulwark | 4800 | 400 | 11214 |

Equal to within two percent, and every other defence in the game answers
something: dodge answers accuracy, a guaranteed hit answers dodge, block answers
the guaranteed hit, multi-strike answers block, cleanse answers a poison. Armour
answers all of them and nothing answers armour.

**Piercing should be a ratio, not a switch.** Measured against the two units
above, as parts per thousand of defence ignored:

| pierced | sentinel | bulwark | bulwark's edge |
| ---: | ---: | ---: | ---: |
| 0 | 11397 | 11214 | 0.98x |
| 200 | 9717 | 9937 | 1.02x |
| 400 | 8072 | 8648 | 1.07x |
| 600 | 6418 | 7361 | 1.15x |
| 1000 | 3100 | 4800 | 1.55x |

A ratio is a dial across that whole range. A switch — true damage, defence
ignored outright — jumps straight to the last row, which makes an armour unit
absolutely worthless against one skill and unaffected by the next, with nothing
in between. That is the same reason buffs saturate here instead of being clamped:
a hard cap on a continuous quantity is the wrong shape, and this engine has
already chosen against it everywhere else.

What an implementation has to settle:

- **Where the number lives.** A field on `skill.Skill`, passed into
  `combat.Rules.Damage` as another parameter. Adding it with a zero default moves
  **no golden file**; the first shipped skill given a non-zero value moves
  `scenarios.golden`, and that diff is the record of the balance change. `sever`
  is the obvious first candidate — it is the metal skill that already exists to
  cut through things.
- **Whether it reaches damage over time.** A tick's damage is computed once, when
  the stack is *applied*, and frozen on the stack. So a piercing skill that
  applies a poison would freeze a pierced tick for the rest of that stack's life,
  which is a much larger effect than a pierced single hit. Decide deliberately.
- **What it does to the budget.** `MaxEffectiveHP` bounds durability against
  non-piercing damage. Once piercing exists, that bound no longer describes the
  worst case, and the question becomes whether raw health needs a floor of its own
  so an all-armour unit cannot be deleted by one skill.
- **How it reads in the log.** A pierced hit that looks like an ordinary hit makes
  the log unable to explain its own numbers — the same trap passive skills have.
  It needs to be visible in the event, not inferred from the damage being larger
  than expected.
- **What the tools say.** `hexforge`'s effective-health figure, and the
  `máu quy đổi` / `effective hp` row, describe durability against damage that does
  not pierce. The moment piercing ships, one number stops being the answer and
  both cases have to be shown, or the label is lying.

Determinism is unaffected: this is integer parts-per-thousand arithmetic with no
new randomness, which is the one thing that makes it cheap to add later.

### A real cast

The tooling for this exists now — see *Authoring a cast* above.
`internal/core/cast` holds origins, archetype presets and characters;
`internal/forge` sequences the authoring and `cmd/hexforge` and
`cmd/hexforge-tui` are two front-ends over it; `progression.Line` is finally used, with one
single-stage and one two-stage example in `cast.json`; and a roster entry can
place a character by reference. What remains is the cast itself:

- **The characters.** `cast.json` ships two examples that say in their own
  biographies that they are examples. `roster.json` is still the ten-unit flat
  test bed, deliberately: it covers every affinity and every mechanic, and
  turning it into references would change every battle the golden files were
  measured from.
- **Art.** `hexforge check` verifies that the file a character names exists, and
  two placeholder SVGs sit under `internal/seed/data/assets/` so it passes out of
  the box. Nothing rasterises them yet, which is the same pipeline question the
  graphical client faces.
- **Elements to design around.** `battle.New` refuses a unit carrying a skill of
  an element it does not share, so an archetype's kit constrains the affinity of
  every character built from it — `skill.CanCarry` is the single declaration of
  that rule and `hexforge` applies it while you are authoring, so this is a design
  constraint rather than a trap. `hexforge archetypes` prints each preset's
  demand: bulwark needs metal, vanguard fire, sentinel water, duelist wind,
  skirmisher grass and electric. A new preset mixing three elements' skills is
  rejected, because no affinity can hold three.

The stat budget is the constraint to design against: `progression.Limits` bounds
each stat and, separately, bounds health and defence together, because those two
multiply rather than add. A unit at both ceilings is not merely durable, it is
durable squared. The five shipped presets spend between 4036 and 11397 of the
11500 effective-health budget, and `hexforge show` prints what is left.
