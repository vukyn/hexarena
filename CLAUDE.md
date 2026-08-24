# CLAUDE.md

Guidance for Claude Code working in this repository. Read `README.md` first for
what the game is; this file is about how the code is allowed to behave.

## What this repo is

A standalone Go binary and its engine — `github.com/vukyn/hexarena`. Like `sgo`,
`gobuild` and `speedtest` under the platform root, **the platform service
conventions do not apply here**: no database, no `sarulabs/di`, no domain layers,
no Fiber, no `mprocs` or `/etc/hosts` entry, no swagger. It does not import
`kuery`, and the kuery shared-package rule is irrelevant because there is nothing
here another service would want.

Go 1.25, standard library only. Adding a dependency needs a reason that survives
the question "what does this do that the standard library will not".

## Commands

```bash
go run ./cmd/hexarena --seed 11 --side ally      # play
go run ./cmd/hexarena --auto --seed 11           # both sides play themselves
go run ./cmd/hexarena --auto --log b.json        # write a log
go run ./cmd/hexarena --replay b.json --verify   # re-run it and check every event

go test ./...
go test ./internal/core/hex ./internal/seed ./internal/tui -update   # accept new goldens
go test ./internal/core/battle -run TestControl                     # one test
gofmt -l . && go vet ./...
```

There is no Makefile and no linter config. `gofmt` and `go vet` are the gate.

`-update` is only defined in the three packages that hold golden files, so
`go test ./... -update` fails on the rest. A new package with a golden has to be
added to that command.

## The layer rule

Everything under `internal/core/` except `battle` is a **pure function of its
integer arguments**. That is not a style preference, it is the property the whole
design rests on: a battle is reproducible from a seed, so a log is a complete
record and a renderer can never disagree with the engine.

Concretely, in every core package except `battle`:

- **No floating point.** Ratios are integers in parts per thousand against
  `scale.Base`. If a formula seems to need a float, it needs one more
  multiplication before the division instead.
- **No clock.** Nothing reads `time`.
- **No randomness** except through an `*rng.Source` handed in as a parameter.
  `rng` is the only package that may generate any, and it has no global state, no
  default seed and no entropy source.
- **No map iteration in anything that reaches an output.** Maps are fine for
  lookup by key. Ordering a result by ranging over one is not, because Go
  randomises that order and a battle would stop replaying. Use a slice and an
  explicit comparator; the collections here are small enough that the cost is
  irrelevant.
- **No filesystem.** Parsers take `[]byte`. `internal/seed` owns the `go:embed`
  and the callers own any real file access.

`battle` is the only package that holds state. `tui` and `cmd` hold none either —
they render.

## The event log is the contract

`battle` emits `[]Event`. Anything that draws a battle reads that and nothing
else.

`internal/tui` is the reference: `Line`, `Summary`, `Tallies`, `TagsFromLog` and
`NamesFromLog` all take events and never touch a `*Battle`. **If a renderer needs
something the log cannot supply, add it to the log.** Reaching into the battle to
fill the gap is how a renderer becomes a second copy of the rules that then drifts
from the first.

Two tests hold the line: `TestEveryEventKindIsReachable` fails if a declared kind
is never emitted by any real battle, and `TestEveryEventKindRenders` fails if a
kind falls through the renderer's default case.

Event kinds and sides serialise **by name**. Do not change that to a number: a
saved log would silently reinterpret itself the next time a constant was inserted.
Renaming a kind breaks existing logs, so treat the names as the wire format.

## Saturate continuous values, cap discrete ones

Two different bounds, and using the wrong one is a design bug rather than a
cosmetic one.

- A **continuous** value — a stat under buffs, a hit chance under accuracy —
  saturates via `scale.Saturate`, approaching a limit it never reaches. Each
  further term is worth less than the last.
- A **discrete resource** — status stacks, block charges — takes a **hard cap**.
  Saturating a count that is the same order of magnitude as its limit would take a
  haircut off even a single application, which reads as broken.

`modifier.Bounds.MaxAffinityScale` is a hard clamp for exactly that reason, and
its doc comment says so. Block charges are capped, and `combat.Rules.GrantBlocks`
exists so no caller can push past the cap with a plain addition.

## Invariants worth knowing before editing

**Geometry.** Every distance and area calculation goes through cube coordinates.
Offset coordinates exist only for authoring and rendering. `hex.Place` maps an
enemy formation with a 180 degree rotation — remove it and the two halves stop
mirroring, silently.

**Element chart.** `element.Chart.Validate` enforces that every element is
classified exactly once, that a pair is only mutually strong when declared so, and
that every cycled element has the same number of strengths as weaknesses. Adding a
twelfth element means adding it to a cycle, not just to the constant list; the
validation will say so.

**Stat budget.** `progression.Limits` bounds each stat and, separately, bounds
health and defence *together*, because those two multiply rather than add. A unit
at both ceilings absorbs several times what either ceiling suggests.

**Skill validation is cross-book.** `skill.ParseBook` takes the pattern and status
books and checks every name a skill uses. A skill naming a shape or a status that
does not exist fails at load, not at the moment it would have mattered.

**Where balance numbers live.** Tick power and modifier terms belong to the
*status*, not to the skill that applies it, so two skills inflicting the same
debuff inflict the same thing. A skill contributes the attack behind it, which is
why two attackers stacking one poison produce stacks of different weight.

## Mistakes already made here

Each of these was written the obvious way, failed a test, and was fixed. The
comment in the code says why; this list is so the same shape is not reintroduced
somewhere new.

**Ordering inside a turn.** `battle.Advance` does these in a fixed order and each
step's position was earned:

1. Check control **before** spending durations. A one-turn stun applied on one
   turn must cost the next; spending its duration first expires it in the very
   turn it was meant to prevent, so one-turn control does nothing at all.
2. `retuneAll` after **anything** that changes a stat, not only at the start of
   the changed unit's own turn. A haste cast on a unit's turn has to shorten the
   wait it is about to serve; noticing it only after that wait elapsed makes the
   buff worthless for the turn it was cast on.
3. Spend cooldowns when a turn **ends**, in `Act` and `Pass`. Spending at the
   start means the options a unit is offered and the action it is allowed to take
   read different numbers — a skill on cooldown gets accepted — and a cooldown of
   three only costs two turns.

**Do not backfill an event after emitting it.** Damage is subtracted per strike as
it resolves so each event carries the health that was actually left. Totalling and
patching the last event produced a log where the second strike of a pair reported
more health than the first.

**A forced turn is not a decision.** `Replay` walks through skipped turns on its
own. Requiring the script to record them meant a battle ending on a poison tick
could not be replayed to its own conclusion, and `--verify` caught it.

**One source for a recorded string.** A passed turn's reason lives on
`battle.Decision`, not on whoever calls `Pass`, and `battle.NoActionReason` is the
single declaration. Two callers wording the same choice differently made a replay
diverge from the log it was replaying.

**`Replay` must not open a turn it cannot decide.** It hands back the pending
prompt when it stops, and takes up an already-open turn rather than advancing past
one, so it is resumable. Without both, undo left the battle waiting for an action
nobody was going to supply.

**Watch the arithmetic, not the intent.** Several assertions written from a hand
calculation were wrong while the code was right — a saturation gap taken from the
wrong side, a drift bound of one when each of two truncations can lose one, a
guess that cleansing earlier is always better when the attacker simply reapplies.
When a test disagrees with a hand-derived number, re-derive before touching the
code.

## Data and golden files

Balance lives in `internal/seed/data`, embedded with `go:embed`. Changing a number
there changes the game without touching Go.

The golden files under `testdata` are **the design record**, not fixtures to be
regenerated on autopilot:

- `internal/seed/testdata/scenarios.golden` is the largest one and the most
  useful: it holds the measured behaviour of buffs, debuffs, elemental
  effectiveness, accuracy, dodge, block, multi-strike, area coverage, the turn
  economy and the damage-over-time ramp, with the numbers the design decisions
  were made from.
- `replay.golden` is a whole battle rendered from its log.
- `skills.golden`, `elements.golden`, `progression.golden`, `combat.golden` are
  the tables each book produces.

Run `go test ./internal/core/hex ./internal/seed ./internal/tui -update` to accept a change and then
**read the diff**. That diff is what the files are for: a balance change that
moves numbers you did not expect is a finding, not noise.

Several tests deliberately hardcode design figures rather than reading them from
the data — `TestRangeLadder`, `TestShippedDualStacking`, `TestDefenseCurveAnchors`,
`TestShippedProgressionLimits`. Those exist so shipped data cannot drift from the
design silently. If one fails, decide which of the two is wrong before editing
either.

## Open work

Detail and the open questions are in `README.md` under Roadmap. What matters here
is the constraint each piece has to respect.

- [ ] **Graphical client with ebiten.** A renderer over `[]Event`, nothing more.
      It must not read `*Battle`, and it must not need the engine to know how long
      an animation takes. Asset pipeline is undecided: SVG has to be baked to PNG
      at build time or rasterised at load, because ebiten draws neither.
- [ ] **A deeper opponent.** `battle.Suggest` currently picks the highest expected
      damage and never buffs, cleanses, shields or sets up a detonate, so the
      whole timed-effect layer is tested but not played. A replacement must read
      no randomness and mutate nothing — a client calls it for a hint mid-turn —
      and two identical battles must still produce identical logs.
- [ ] **A real cast.** `roster.json` is a test bed with flat stats.
      `progression.Line` supports staged evolution and nothing uses it yet. Design
      against `progression.Limits`, particularly the joint health-and-defence
      bound, since those two multiply.
