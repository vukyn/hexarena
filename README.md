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
What that sweep cannot say is whether 400 is the right number, because the shipped
roster is a mirror — both squads carry the same kit, so piercing helps both sides
equally and the win rate moves by noise. Judging the value needs an asymmetric
roster, which is the cast's problem rather than piercing's.

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

### Learnsets, four slots, and choosing to evolve

A character knows every skill it will ever know, from level one, and brings all
of them. That makes a level cap the only thing separating a young unit from a
grown one, and it means authoring a ninth skill makes a character strictly better
rather than presenting a choice.

Four pieces, and they are one mechanism:

- **A learnset.** A character declares *when* it learns each skill rather than
  simply holding a list: `{skill, at_level}`, or `{skill, at_stage}` for one that
  only a later form can hold. A stage-gated skill is gated behind whatever that
  stage requires, transitively, which is what makes evolving unlock something
  rather than merely raise numbers.
- **Four slots.** A placement brings at most four of what the character has
  learned. Refused at load if it names one the character has not learned at that
  level, if it names more than four, or if it names one twice.
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

Three things this has to face.

**The log has to carry the placement.** A log holds a seed, the decisions taken
and the events produced, and the roster comes from the embedded data — so a log
is already only valid against the build that wrote it. Once the loadout and the
stage are *choices*, a log without them cannot be re-run at all, and `--verify`
would be comparing two different battles. Writing the placement into the log is
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

### A battle nobody can act in has no outcome

`checkEnd` ends a battle when a side is emptied, and that is the only way one
ends. A battle where **neither side can act** is not an outcome the engine can
express: it runs to the turn limit and comes back as a failure.

This is not hypothetical. Seed 18 spent 3955 of its 4000 turns skipped, every one
a unit with nothing usable, because two survivors stood on the back column with a
kit that reaches three cells. Measured through `hex.Place`, the far corner of the
board is:

| authored column | distance to the furthest enemy slot |
| --- | ---: |
| 2, the front | 3 |
| 1 | 4 |
| 0, the back | 5 |

So a short-ranged unit behind the front stops being able to act the moment the
enemies near it die, and if both sides are left with only such units the battle
cannot finish. The shipped roster now stands its whole squad on column two, which
avoids it rather than fixes it.

**The underlying reason is that nothing moves.** A unit occupies the slot it was
placed in for the whole battle, so its reach is fixed at enlistment. In a game
with movement this situation resolves itself; here it cannot.

Two answers, and they are not alternatives:

- **Refuse it at authoring.** `battle.New` could check that every unit can reach
  *someone*, and `hexforge` could warn when a character's longest range cannot
  cover the board from the column its preset suggests. That catches the common
  case early, where an error is cheapest. It is not sufficient on its own: reach
  changes as units die, and a roster that starts fine can still end deadlocked.
- **End it at runtime, as a draw.** This is the real fix, and what it has to
  settle:
  - **A skipped turn is not a stalemate.** Turns are skipped by control and by
    cooldowns all the time, and those resolve. The condition is a full cycle in
    which nobody could act *and* nothing is pending that would change that — a
    poisoned deadlock is not a deadlock, because the poison ends it.
  - **A draw needs its own event.** A battle that simply stops leaves the log
    unable to say why, and `--verify` comparing two runs of a battle that ended
    for no stated reason proves less than it looks. Same trap as a passive
    changing a number silently.
  - **It is not an error.** `RunToEnd` returns one when it hits the limit, which
    is right for a runaway and wrong for a draw. The turn limit should stay as
    the backstop it is, and stop being the thing that reports this.
  - **Detection must be a pure function of the state.** A count of cycles is
    fine; anything reading a clock is not, and a battle that drew on one machine
    has to draw on every other from the same seed.

### A real cast, and a roster that is not a mirror

The tooling for this exists — see *Authoring a cast* above. What remains is the
cast, and the shape of the gap has changed since this was written: the example
characters are gone and `cast.json` ships exactly one, `pokemon.bulbasaur`, with
three forms and art for each.

- **One character is not a cast.** Nine skills and three forms all belong to it,
  so every design question about how two characters differ is unasked.
- **The roster is a mirror, and that is the blocking half.** `roster.json` is 3v3
  by reference at levels 60, 24 and 8 — the same character on both sides. A
  mirror cannot measure balance: a change that helps one side helps the other by
  exactly as much, so the win rate moves only by noise. Measured over forty
  auto-battles, giving `razor_leaf` its piercing value moved the ally win rate
  from 23 to 25 of 40, which says nothing at all. The damage table said the
  useful things instead. So the value of a second character is not variety, it is
  that balance becomes measurable.
- **Art is answered.** `hexforge check` verifies every picture a character names,
  its own and each form's, and `p` in `hexforge-tui` draws it — the terminal
  rasterises through `forge.ArtImage`, which narrows the graphical client's
  pipeline question rather than settling it.
- **Elements to design around.** `battle.New` refuses a unit carrying a skill of
  an element it does not share, so an archetype's kit constrains the affinity of
  every character built from it. `skill.CanCarry` is the single declaration of
  that rule and `hexforge` applies it while you author, so this is a design
  constraint rather than a trap. A preset mixing three elements' skills is
  rejected, because no affinity can hold three.

Every character added moves `scenarios.golden` and `replay.golden`. That is the
point rather than a cost: those diffs are how a balance change gets read.

The stat budget is the constraint to design against: `progression.Limits` bounds
each stat and, separately, bounds health and defence together, because those two
multiply rather than add. A unit at both ceilings is not merely durable, it is
durable squared. The five shipped presets spend between 4036 and 11397 of the
11500 effective-health budget, and `hexforge show` prints what is left.
