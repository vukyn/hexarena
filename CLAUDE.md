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

Go 1.27. Third-party dependencies are allowed anywhere in the module; reach for
one when it earns its place. `cmd/hexforge-tui` uses bubbletea, bubbles and
lipgloss, and the rest of the module currently happens to need nothing beyond
the standard library — that is where it landed, not a rule to defend.

What *is* a rule is the layer contract below, and none of it is about
dependencies: `internal/core` stays a pure function of its integer arguments no
matter what the module imports. A dependency that reaches into the engine has to
answer one question first — "what happens to a replay when this dependency
changes its mind" — because a battle that stops reproducing from its seed takes
the log format, `--verify` and undo down with it.

## Commands

```bash
go run ./cmd/hexarena --seed 11 --side ally      # play
go run ./cmd/hexarena --auto --seed 11           # both sides play themselves
go run ./cmd/hexarena --auto --log b.json        # write a log
go run ./cmd/hexarena --replay b.json --verify   # re-run it and check every event

go run ./cmd/hexforge                            # author the cast: list the subcommands
go run ./cmd/hexforge new                        # create a character, prompting for what is missing
go run ./cmd/hexforge check                      # parse the books from disk and verify the art exists

go run ./cmd/hexforge-tui                        # the same authoring, full screen (needs a terminal), in Vietnamese
go run ./cmd/hexforge-tui --lang en               # ...in English; HEXARENA_LANG=en does the same, ctrl+l toggles

go test ./...
go test ./internal/core/hex ./internal/seed ./internal/tui -update   # accept new goldens
go test ./internal/core/battle -run TestControl                     # one test
gofmt -l . && go vet ./...
```

The `Makefile` wraps those and nothing more — `make build install run auto forge
forge-tui forge-tui-en test golden fmt vet check clean`. `make build` builds all three
binaries; `make forge ARGS="show some.id"` passes arguments through. `make check` is the gate (`gofmt -l .`, `go vet ./...`,
`go test ./... -count=1`); `make golden` is the `-update` line above. The raw
commands stay listed here because they are what the targets are: reach for either.
There is no linter config — `gofmt` and `go vet` are the whole of it.

`-update` is only defined in the three packages that hold golden files, so
`go test ./... -update` fails on the rest. A new package with a golden has to be
added to that command **and** to the `golden` target.

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

**Where the filesystem rule bites: a character's art.** `cast.Character.Image` is
a path, and there are two different questions about it.
`cast.ValidateImagePath` answers the first — is the path *well shaped*: relative,
no `..` segment, no drive volume, ending `.svg` or `.png` — using `path` rather
than `filepath`, because a committed data file has to mean the same thing on
every platform. Whether the file is **really there** is only asked by
`internal/forge`, because `internal/core` may not read the filesystem and,
more to the point, only the caller knows which directory the path is relative
to. Do not move the existence check into the parser to make it "complete": that
would make loading the game depend on the working directory, and the embedded
copy has no directory at all. `cast.ValidateID` is exported for the same reason
`ValidateImagePath` is — an authoring tool has to reject an answer as it is
typed, not at the end of a wizard.

**Two front-ends, one set of rules: `internal/forge`.** The authoring logic —
loading the books from a directory, `Draft.Resolve` turning answers into a
validated character, the per-answer checks a prompt or a form applies as it is
typed, `Budget`, `Inspect`, the temp-file-then-rename write, and `SaveNotes` —
lives in `internal/forge`. `cmd/hexforge` is flags and prompts over it;
`cmd/hexforge-tui` is a full-screen bubbletea client over the same thing.
Neither may restate a rule, **including the wording of a refusal**: a front-end
that phrases a rejection itself is a second declaration of the rule behind it,
which is the mistake recorded twice below (the passed-turn reason, and the
kit-versus-affinity gap). If the TUI needs a sentence the CLI already has, it
comes from the package. `internal/forge` is the one part of the module allowed
to read and write real files, and its doc comment says why: `internal/core` may
not, and `internal/seed` only ever reads the embedded copy.

**Two languages, one set of facts: `internal/i18n`.** `cmd/hexforge-tui` speaks
Vietnamese by default and English on `--lang en` / `HEXARENA_LANG=en` / `ctrl+l`;
`cmd/hexforge` stays English because scripts read it, and its output is a
contract pinned by `TestARefusalKeepsTheWordingTheCommandLinePrints`. That is why
`internal/forge` returns **values, not sentences**: `CheckCarry` gives a
`*CarryError` holding the affinity, the skill and the skill's element,
`PresetFacts` / `StageFacts` / `SaveNoteFacts` / `Report.Problems` give data, and
the string helpers (`DemandSummary`, `PresetSummary`, `StageSummary`,
`SaveNotes`) are thin wrappers **built on** those, so the command line's English
cannot drift from the facts. `internal/i18n` holds a `Lang` (parsed by name,
unknown is an error), a `Key` enum and one array per language; it may import
`internal/forge`, and **forge must never import it**.

Three rules hold that shape, each with a test: no user-visible literal may live
in `cmd/hexforge-tui` (`TestNoScreenHoldsItsOwnWording` greps its own AST), every
key is worded in both languages and no key is orphaned, and every wording
measures one cell per letter — write Vietnamese **composed**, or a combining mark
measures zero and every fixed-width column on that screen drifts. What is
deliberately *not* translated: ids of every kind, the six stat labels
`hp atk def spd acc ddg` (they are the `--hp` flag names and the data files' own
keys — see `forge.ShortStat`), and diagnostics from `internal/core`, which get a
lead-in in the reader's language in front of the parser's own English.

The minimum window is **80x24**, up from 72: Vietnamese runs 20–30% longer and
the busiest footers no longer fit, which was measured rather than guessed
(`TestEveryWordingFitsTheMinimumWidth` renders every screen in both languages and
holds every line inside it, minus one column so a full-width line cannot wrap).

`TestTheFormProducesTheCharacterTheCommandLineProduces` is what holds this: the
same answers as flags and as keystrokes must resolve to the same
`cast.Character`. A full-screen program cannot run with stdin as a pipe, so
`cmd/hexforge` is not going away — it is what a script uses, and the TUI refuses
to start when stdout is not a terminal rather than painting escape codes into
one.

**`hexforge new` must work with nobody watching.** A preset-supplied value is
not missing, so an unattended run takes every default and errors only on a field
that has none, naming its flag. Two traps live here. `os.Stdin.Stat` cannot tell
a terminal from `/dev/null` — both are character devices — so the mode check is
only a first guess and **EOF on a read is the authoritative signal**; hitting it
turns the rest of the session unattended rather than failing on a field whose
default was fine. And the kit is asked **before** the element, because the kit is
what decides which elements are legal: asking the other way round means either
validating against a preset the author is about to replace with `--skills`, or
accepting an answer the write then refuses.

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
`cast.ParseBook` and `cast.ParseArchetypes` follow the same shape — a character's
origin, archetype, kit, affinity and every stage's stat table are checked against
the books that declare them, and an archetype preset that does not itself fit
`progression.Limits` is rejected, because a preset that fails the budget hands
every author a stat line that fails later. A character whose kit its affinity
cannot carry is rejected too — see the carry rule below.

**A roster entry never has two sources for one number.** `seed.ParseRoster`
accepts two forms: the flat one, which writes out `name`, `element`, `stats` and
`skills`, and the reference one, which names a `character` and a `level` and
resolves all four from the cast book. Mixing them is **rejected**, not resolved by
precedence — a precedence rule silently ignores half of what was authored, and the
half it ignores is the half someone just edited. `level` is required with
`character` (an evolution line cannot be resolved without one) and refused
without it (an inline stat line is already resolved). `battle.Roster` deliberately
gains no image, biography or origin field: the engine has no use for them, and the
event log is what a renderer reads.

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

Three of those files are the cast rather than the balance, and `cmd/hexforge` both
reads and **writes** them:

- `origins.json` — the works characters are borrowed from. Hand-editable;
  `hexforge origins add` appends to it.
- `archetypes.json` — the role presets: a suggested stat curve and kit per role.
  Hand-authored only; the tool never writes it. Every preset's curve must pass
  `progression.Limits.CheckTable`, and the ids match the roles `roster.json`
  already uses.
- `cast.json` — the authored characters, each an evolution line
  (`progression.Line`) plus an origin, an archetype it was tuned from, an
  affinity, a kit and a path to its art. `hexforge new` appends to it.

`cast.json` and `origins.json` are committed **in exactly the form
`Book.Marshal` writes** — two-space indented, sorted by id. That is deliberate:
the tool rewrites the whole file on every addition, so if the committed form
drifted from the written one, the next `hexforge new` would produce a diff of the
entire file instead of one block. `TestWrittenCastIsStableAndReloads` fails if it
drifts. Marshal is also the one place in `cast` that *imposes* an order rather
than preserving the authored one; everything else keeps declaration order,
because a map range would randomise it.

Art lives under `internal/seed/data/assets/` and is **not embedded** — the embed
directive names the JSON files one by one. The two placeholder SVGs there exist so
`hexforge check` passes out of the box; replace them, do not delete them without
also replacing the example characters.

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
- `origins.golden`, `archetypes.golden`, `cast.golden` are the same for the cast:
  which works are catalogued and who was borrowed from each, every preset's curve
  with what it spends of the effective-health budget, and every character
  resolved at each of its stage boundaries and at the cap.

Run `make golden` (`go test ./internal/core/hex ./internal/seed ./internal/tui
-update`) to accept a change and then
**read the diff**. That diff is what the files are for: a balance change that
moves numbers you did not expect is a finding, not noise.

Several tests deliberately hardcode design figures rather than reading them from
the data — `TestRangeLadder`, `TestShippedDualStacking`, `TestDefenseCurveAnchors`,
`TestShippedProgressionLimits`,
`TestShippedArchetypesMatchTheReferenceProfiles`. Those exist so shipped data
cannot drift from the design silently. If one fails, decide which of the two is
wrong before editing either. The last one ties `archetypes.json` to the reference
profiles `progression.golden`'s hits-to-kill table was read from, so the presets
and the balance reasoning cannot part company.

**One rule, one declaration: which skills an affinity may carry.**
`skill.CanCarry` is the whole of it. `battle.enlist` calls it and
`cast.ParseBook` calls it, so a character that writes cleanly is a character
that loads — before, `battle.New` refused a unit carrying a skill of an element
it did not share and the authoring layer had no idea, so `hexforge new
--archetype sentinel --element fire` wrote a character and `hexforge check` said
"no problems found". Do not restate the condition at a third call site; that is
the mistake the "one source for a recorded string" note above is about.

`skill.Demands` is the other half, derived rather than authored:
`Archetype.Demands` is the distinct non-neutral elements a kit requires, filled
in `ParseArchetypes` and tagged `json:"-"` so a data file cannot claim a demand
its kit does not have — an authored hint would only be caught when a character
built from the preset was refused. A kit demanding more than two elements is
rejected outright, because no affinity can hold three. Whether the two it
demands are *allowed together* needs the element chart, which a preset is
validated without, so that half lives in
`TestEveryShippedArchetypeKitIsCarryableAtAll`.

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
- [ ] **A real cast.** The tooling now exists: `internal/core/cast` holds origins,
      archetype presets and characters, `cmd/hexforge` authors them, and
      `progression.Line` is finally used — `cast.json` ships one single-stage and
      one two-stage example. What is missing is the cast itself and its art. Design
      against `progression.Limits`, particularly the joint health-and-defence
      bound, since those two multiply, and remember that an archetype's kit
      constrains a character's affinity, which `skill.CanCarry` now enforces at
      authoring time and `Archetype.Demands` reports. `roster.json` is
      still the flat test bed on purpose: turning it into character references
      would change every battle the goldens were measured from.
