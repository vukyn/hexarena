# CLAUDE.md

Guidance for Claude Code working in this repository. Read `README.md` first for
what the game is; this file is about how the code is allowed to behave.
## The memory layer

@MEMORY.md

⚠️ **That import is the point of the file, not decoration.** `MEMORY.md` and
`memory/` are the distilled layer — one hard-won fact per file, with why it
matters — and they live **in the repository** because a machine's own Claude
memory directory is workspace-scoped and machine-local: this repo opened on
another machine, or outside the workspace the notes were written in, arrived with
none of them. Almost all of `.claude/` here is gitignored (worktrees and
per-agent scratch), so tracked is the only place that travels.
⚠️ **"`.claude/` is gitignored" was written flat and is not quite true** — the
`.gitignore` reads `/.claude/*` with `!/.claude/agent-memory`, so the root
agent-memory subtree **is tracked** while every nested `.claude/` a session
creates in a sub-directory is not. Worth knowing before a commit: a change under
`.claude/agent-memory/` is a change that goes into the PR.

It is a **distillation, not the record.** This file, `TODO.md` and `README.md`
stay the authority; where a note disagrees with the file that owns the subject,
the repository wins and the note is what to fix. `MEMORY.md` carries the rules
the notes are written under — one line per note in the index, one fact per file,
say why rather than only what, and delete a wrong note rather than adding a
second one beside it.
## What this repo is

A standalone Go binary and its engine — `github.com/vukyn/hexarena`. Like `sgo`,
`gobuild` and `speedtest` under the platform root, **the platform service
conventions do not apply here**: no database, no `sarulabs/di`, no domain layers,
no Fiber, no `mprocs` or `/etc/hosts` entry, no swagger. It does not import
`kuery`, and the kuery shared-package rule is irrelevant because there is nothing
here another service would want.

Go 1.27. Third-party dependencies are allowed anywhere in the module; reach for
one when it earns its place. `cmd/hexforge-tui` uses bubbletea, bubbles and
lipgloss — all three at **v2**, under `charm.land/…/v2` rather than
`github.com/charmbracelet/…`, which is where that project moved them — the art
rasterises on oksvg/rasterx, and **`internal/socket` uses
`github.com/coder/websocket`** (zero dependencies of its own; → § *The
transport* for why not gorilla). Everything else happens to need nothing beyond
the standard library — that is where it landed, not a rule to defend.

What *is* a rule is the layer contract below, and none of it is about
dependencies: `internal/core` stays a pure function of its integer arguments no
matter what the module imports. A dependency that reaches into the engine has to
answer one question first — "what happens to a replay when this dependency
changes its mind" — because a battle that stops reproducing from its seed takes
the log format, `--verify` and undo down with it.

## Where the rest of the record lives

⚠️ **Five sections' worth of this file have moved out — three on 2026-09-05 and
two more on 2026-09-07 — and nothing was deleted.** This file is loaded in full at
the start of every session; it was 396KB, about a hundred thousand tokens spent
before any work began. Measured first, because the obvious diagnosis was wrong: of
4,333 prose lines here, **six** appeared verbatim in `README.md` or `TODO.md` —
the four documents do not repeat each other at all, so there was nothing to
*delete*, only somewhere better to *put* things.

⚠️ **The second pass was needed because the first one left 60KB under a heading
that sounded binding.** Measured 2026-09-07: the 2026-09-05 split left this file
at 149,851 bytes and two days later it was 161,139, of which § *The layer rule*
alone was 64,273 — and the layer contract in it is about thirty lines. The rest
was the front-end: one bullet block, *Where a form beats a prompt*, was **42,640
bytes, 27% of the whole file**. A heading is not a rule; what binds is the
sentence, and the sentence stayed.

What stayed is what binds **any** edit, whatever it touches. What left is
subject matter — read it when you are in that subject:

| moved to | what is in it |
|---|---|
| `docs/architecture.md` | the runtime pieces: the two clients over one `internal/screen`, `internal/room` (a state machine with no I/O and no clock), `internal/socket` (the one boundary the WebSocket crosses), **the event log as the contract**, and how a battle ends |
| `docs/balance.md` | how the game is priced and tuned: `Suggest`'s rating, species and origins, the inert element, the strip/guard/grant/summon/amplify/detonate/`charge` categories, `hexforge weigh`, and `forge.Bout` |
| `docs/screens.md` | the front-ends: what bubbletea v2 broke silently, the save key and the `ctrl+s` footer, `internal/i18n` and the four rules holding its shape, `frame`'s marking cut and the prose-versus-data width rule, the kit picker, the squad builder, the played battle's row budget, the skill filter, `browseRoom`, and the paste bug |
| `docs/goldens.md` | the data files and the goldens that record them: what each golden under `testdata` is a record *of*, the shape an authoring write leaves a committed file in, why editing a skill can break what adding one cannot, `roster.json`'s four contracts and `builds.json`'s catalogue |
| `TODO.md` and `docs/decisions.md` | the twenty-eight-item log that lived here: its three open items joined `TODO.md` § *Not done* and its twenty-five finished ones joined `docs/decisions.md`, which is where finished work and its reasoning live |

⚠️ **The one-line rules those sections rest on did NOT move**, and that is the
point of the split rather than a happy accident. *The layer rule* below still
states that a battle is a pure function of its seed and that a renderer can never
disagree with the engine — which is the whole of why the event log is a contract.
*Invariants worth knowing before editing* still holds what an edit may not break.
The moved files carry the reasoning and the measurements; the binding sentence
stays here.

⚠️ **Work items are addressed by CODE, not by line number.** Every entry in
`TODO.md` carries one — `AREA-NNN`, e.g. `RAT-006` for a rating item, `SCR-008` for
a screen one — and `TODO.md` § *The codes* holds the area table and the rule that a
code is assigned once and never reused. Cite the code in a commit message, a PR and
a `docs/decisions.md` entry; a line number is stale the next time somebody writes a
paragraph above it.

**So: still read this file first.** Then open the one for the subject in hand:
`docs/architecture.md` before touching `internal/room`, `internal/socket` or the
event stream; `docs/screens.md` before touching `internal/screen` or either
client; `docs/goldens.md` before accepting a golden or editing
`internal/seed/data`; `docs/balance.md` before authoring or re-pricing anything.

## Commands

```bash
go run ./cmd/hexarena --seed 11 --side ally      # play
go run ./cmd/hexarena --auto --seed 11           # both sides play themselves
go run ./cmd/hexarena --auto --log b.json        # write a log
go run ./cmd/hexarena --replay b.json --verify   # re-run it and check every event

go run ./cmd/hexforge                            # author the cast: list the subcommands
go run ./cmd/hexforge new                        # create a character, prompting for what is missing
go run ./cmd/hexforge skills                     # the declared skills and who may carry each
go run ./cmd/hexforge skills add oath --power 1200 --accuracy 900   # author a skill
go run ./cmd/hexforge skills edit oath --power 1100                 # change one already in the book
go run ./cmd/hexforge statuses                   # the timed effects, grouped, and what each does
go run ./cmd/hexforge builds                     # the late-game catalogue: which four skills and which trait a character is for
go run ./cmd/hexforge passives                   # the declared traits and what each holds
go run ./cmd/hexforge check                      # parse the books from disk and verify the art exists
go run ./cmd/hexforge spar some.id --seeds 200   # duel it against the whole cast, both ways, report the rates

go run ./cmd/hexforge-tui                        # the same authoring, full screen (needs a terminal), in Vietnamese
go run ./cmd/hexforge-tui --lang en               # ...in English; HEXARENA_LANG=en does the same, ctrl+l toggles

go run ./cmd/hexarena-tui                        # play, full screen: the catalogues a reader wants, a battle, and a room to join
go run ./cmd/hexarena-tui --lang en              # ...in English; same flag, same variable, same ctrl+l
go run ./cmd/hexarena-tui --squads mine.json     # bring sides out of a file of your own, beside the shipped four
#   the default is under os.UserConfigDir(), resolved once in main and handed down; a
#   side there wins the id it shares with a shipped one and its row says whose it is
#   ninth menu entry: paste the twelve-character code a host printed, type the room's
#   password if it has one, pick the squad to bring, and play the other person

go run ./cmd/hexarena-host                       # host one PvP match: prints the room code, serves it, prints the result
go run ./cmd/hexarena-host -battles 3 -password nhaminh   # a bo3 behind a gate; -h says why that flag is visible in ps
go run ./cmd/hexarena-host -advertise 10.0.0.7   # say which address the code carries, where autodetection refuses

go test ./...
go test ./cmd/hexarena-tui ./cmd/hexforge-tui ./internal/core/hex ./internal/i18n ./internal/screen ./internal/seed ./internal/tui ./internal/wire -update   # accept new goldens
go test ./internal/core/battle -run TestControl                     # one test
gofmt -l . && go vet ./...
```

The `Makefile` wraps those and nothing more — `make build install run auto
play-tui forge forge-tui forge-tui-en host test golden fmt vet check clean`. `make
build` builds all **five** binaries; `make forge ARGS="show some.id"`, `make
play-tui ARGS="--lang en"` and `make host ARGS="-battles 3"` pass arguments
through. `make check` is the gate (`gofmt -l .`, `go vet ./...`,
`go test ./... -count=1`, then `-race` over `internal/room`, `internal/socket`,
**`cmd/hexarena-host` and `cmd/hexarena-tui`** — the four places concurrency
lives; the host binary prints from three goroutines at once, because the
transport calls its Joined and Report callbacks on a connection's own goroutine,
and the game client **draws a battle another goroutine is stepping**, which is
the first thing in the repository to do so. ⚠️ The client's is by far the most
expensive line: **3.6s plain, 33.1s under the detector**, and almost none of that
is the concurrency — the four tests that run a match total about 4s and the rest
is that package's five sweeps rendering every screen in both languages at two
sizes); `make golden` is the `-update` line above. The raw
commands stay listed here because they are what the targets are: reach for either.
There is no linter config — `gofmt` and `go vet` are the whole of it.

`-update` is only defined in the eight packages that hold golden files
(`cmd/hexarena-tui`, `cmd/hexforge-tui`, `internal/core/hex`, `internal/i18n`,
`internal/screen`, `internal/seed`, `internal/tui`, `internal/wire`), so
`go test ./... -update`
fails on the rest. A new package with a golden has to be added to that command
**and** to the `golden` target. ⚠️ **This list has gone stale twice**, so it is
spelled in exactly three places — that command, the `golden` target, and the
paragraph under § *Data and golden files* — and a package added to one and not
the others is the next reader running `make golden` and not accepting a golden
they have moved.

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
to. `internal/forge` now answers a second filesystem question as well — *which*
art files exist, via `ArtFiles`, so the authoring form can offer a choice
instead of asking for a path to be typed. Its results are sorted explicitly:
a directory walk has no guaranteed order and that order reaches the screen. Do not move the existence check into the parser to make it "complete": that
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

⚠️ **Everything the front-ends do with that contract lives in `docs/screens.md`
now, and it left whole on 2026-09-07.** What bubbletea v2 broke silently, the
save key and why the footer names `ctrl+s` alone, `internal/i18n` and the four
rules that hold its shape, `frame`'s marking cut and the prose-versus-data width
rule, the kit picker and the damage row, the squad builder, the played battle and
its row budget, the skill filter, `browseRoom`, the unattended `hexforge new` and
the paste bug — 60KB of subject matter under a heading that sounded binding, on a
file loaded in full at the start of every session. **Read it before touching**
`internal/screen`, `cmd/hexforge-tui` or `cmd/hexarena-tui`.

The three sentences above it are the ones that bind, and they did not move: the
layer contract, `internal/forge` as the one part of the module allowed to touch
real files, and **neither front-end may restate a rule, the wording of a refusal
included**.

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

⚠️ **A list the engine walks may not be in ABSOLUTE board order, because the
rotation reverses it.** `battle.mirroredOrder` is the aim walk and it goes by
**authoring slot**, the far half first: `hex.Cells()` is column-major over the
whole board, `Place` turns an enemy slot through 180 degrees, and a rotation
reverses rows where a column-major walk does not — so the two halves were offered
their candidates in **opposite** orders. `Suggest` keeps the first aim that
reaches the best value, so a tie between two identical targets fell to a
different unit depending on which half was asking, and the same battle fought
with the sides swapped stopped being the same battle relabelled.

Measured, a squad against a copy of itself over 400 seeds, one arm listed each
way: **one unit a side was already exact** (615‰ / 385‰) and could never show it,
because one enemy is no tie at all; two units summed to **1035‰** and three to
**1330‰**, and per seed the winner failed to swap in **24 of 200** and **66 of
200**. After the walk moved: 1000‰ at every size, and 200 of 200 seeds swap.
`TestASwappedMirrorSwapsItsWinnerAtEverySquadSize` holds the property and
`TestTheAimOrderIsTheSameSequenceOnEitherHalf` holds the mechanism under it —
both, because a fixture whose ties never arise passes the first while the order
is still wrong.

⚠️ The **far half leads** rather than the caster's own, which only an all-sided
skill can tell apart: putting the caster's own cells first moves the tie of every
such skill onto its own side, which is a balance change wearing the clothes of a
determinism fix.

⚠️ **Reach is counted in RANKS from the far side, not in cells from the caster.**
A skill of range N reaches the first N **occupied** columns of the opposing half,
counted from that half's own frontline (`hex.Ranks`, `Battle.reachableRanks`). An
empty column costs nothing — there is nobody there to shoot past — so a range of
one finds the enemy's foremost survivor wherever it stands. Blocking is by the
**whole rank**: one unit anywhere in a column shields every column behind it,
which is what makes killing the front rank the move that opens the board. It is
deliberately not per-file, which would let one gap expose a whole column.

*Why it changed:* a unit never moves and most skills declare range 1, so measuring
from the caster's own cell made a back-line placement unable to use its own kit —
the range it needed was a fact about where the author had put it rather than about
the skill.

**Two rules hang off it.** An **ally-aimed skill ignores range entirely**: reach is
a fact about the far side, and helping the squad you stand in is not a question of
distance. A **taunt is not filtered at all** — `aims` returns the taunters before
reach is consulted — so a taunter in the back rank drags an attack through
everything in front of it, because a taunt a front rank could wall off would be a
status the front rank cancels.

⚠️ **A board can no longer freeze for want of distance.** The reach guard in
`battle.New` was deleted because it had become unreachable, and with it
`canAimAtAnyone`, `longestRange` and `nearestTargetable`, which only ever phrased
its refusal. `Stalemate` survives for the cause that remains — no kit holding
anything to throw — and `TestNoPlacementCanPutAUnitOutOfReach` is what stops the
guard being re-added on a hunch.

⚠️ **The caster's own column is now purely defensive** (it decides who is reached
first), and rows no longer affect reach at all — only pattern shapes. The odd-q
geometry and the 180° rotation still pay for themselves through patterns; they no
longer drive targeting.

**Range is penetration depth, and the tiers mean something.** `maxRange` is
`hex.FormationCols` — three ranks is the whole of a side, so a four would have
meant what a three means. The shipped book reads: **1** for contact weapons and
the basic attacks, which have to go through the front rank and are the norm;
**2** for shapes that sweep, gas that drifts and things thrown over the line;
**3** for the two heaviest skills in the book, `solar_beam` and `hydro_pump`,
which are the only things that reach a back line through two held ranks.

⚠️ **The numbers are what turned blocking on, and they cost a balance answer.**
Under the ranges the mechanism shipped with, most skills were depth 2 or 3 and
went round a held front rank as a matter of course; at 14 of 31 skills stopping
at the first rank, holding a front line finally decides fights. The shipped
roster was levelled for a board where it did not, and the instrument moved from
19/40 ally to **12/40**. That is a placement finding rather than a skill finding
— both sides lost depth about equally — and re-levelling `roster.json` under
blocking is the follow-up, filed in `TODO.md`.

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

**`forge.ArtImage` is the only thing that rasterises, and the preview is the one
place colour is information.** Reading and drawing a picture is `internal/forge`
for the same reason `ImageExists` is: `internal/core` may not touch the
filesystem. What it returns is pixels — a terminal and a graphical client turn
those into something to look at very differently, and flattening the alpha or
picking characters here would take that decision away from both. `MaxArtPixels`
bounds a side because the cost grows with the area and a preview is redrawn on a
keystroke. In `cmd/hexforge-tui` the preview breaks the palette's rule that
colour is decoration and never information, deliberately and only there, which is
why the `NO_COLOR` path is a **different drawing** (a ramp of weights, keeping the
shading) rather than the same one with the colour stripped (a silhouette in one
character, keeping only the outline). Its cache is keyed on the file's size and
modification time, never on the path alone: a drawing that outlives its file, or
survives the art being redrawn, is this tool lying about the data directory it
exists to report on. Two earlier versions of that cache's test proved nothing —
counting map entries (a cacheless preview writes the same key every time) and
deleting the file (which froze the wrong behaviour) — so it is measured by making
the bytes unreadable while size and mtime stay put.

**A composition bonus is what a SIDE brought, and it is settled before the first
turn.** `internal/core/composition` counts one axis over a side's roster — the
elements its units carry, today — and says which rung a count reaches and who
receives the grant. Five things about it are decisions rather than details:

- **Counted once, from the roster, before anybody is enlisted.** `battle.New`
  resolves the awards while the roster is still a slice of facts, which is both
  the only shape this counting rule may be handed and the only moment early
  enough: a grant has to be on the unit before `queue.Add` reads its speed, for
  the reason a trait does — a wait is `1_000_000/speed` and the first one is
  served before anything could retune it. Nothing recounts, so focusing the odd
  unit out cannot take a bonus away and a summon never earns one.
- **Granted as permanent statuses, through `Set.Hold`**, exactly as a trait's
  grant is. That is what makes a bonus saturate alongside every other term on the
  same stat rather than composing with it, and what answers the dispel question
  without a new rule: `Remove` already refuses a permanent status, and
  `composition.ParseBook` refuses a bonus granting anything else — a timed grant
  would count down and leave a squad that built for a threshold with nothing.
- **A side is counted on its own, and an inert element forms no tribe.** Counting
  across the board would hand a side a threshold its opponent paid for. The
  inert exclusion is read off `element.Chart.Inert()` rather than naming
  `neutral` here, so a chart that gives the inert element a matchup makes it
  count without this rule being edited: sharing the element with no strengths and
  no weaknesses is sharing the absence of one.
- **Rungs are a ladder, not a set** — `Bonus.Reached` returns the highest rung a
  count satisfies and that one only. Read cumulatively the top rung would be
  worth the sum of a table nobody wrote down, and it would read as working
  because the figure only ever goes up.
- ⚠️ **`Book.Without` is the pricing instrument and it is a SET, not a switch.**
  Nothing about a bonus can be measured until it can be turned off: swapping a
  member measures the member, and putting the same bonus on both sides cancels
  it exactly. What is left is the same squad, the same members and the same seeds
  with one bonus gone — `forge.FightSquads(home, away, seeds, "same_element")` —
  and because bonuses stack, a global off would measure the system rather than
  the rung. `SquadReport.Without` carries what was disabled, so a rate cannot be
  quoted without its condition.
- The log says so: `bonus_held`, on the opening board only, carrying the bonus,
  the value shared and the count. It is not a `passive_held` — a trait is what a
  unit **is** and a bonus is what its side **brought**, and the same unit in
  another squad would not have it.

**A passive is statuses, and permanent means four things.** `passive.Passive`
grants `status` ids and nothing else — the terms belong to the status, so a trait
saturates *alongside* a temporary buff through `modifier.Set` rather than
composing with it, which is the one place stacking could explode. Every granted
status must be **permanent**, a flag on `status.Kind` rather than a duration of
nought (nought would make an absent or mistyped duration silently permanent, and
the fields around it already refuse their own zero for that reason). Permanent
means: it never counts down and never reaches `Tick`'s expiry list; `Set.Remove`
refuses it, which covers dispel, cleanse and detonate in one guard, because a
trait is granted **once** and taking a stack off would turn it off for the rest of
the battle; it may not be a `Dot` or a `Regen`, either of which would tick for the
whole battle; and `Snapshot` carries the flag so a renderer draws *always* instead
of the `0t` the countdown alone would give.

**A trait's riders go through the skill's own application list, and only on a
damaging skill.** `passive.Passive.Applies` reuses `skill.Application` and
`battle.riders` feeds it to the same `inflict`, so a rider takes the same roll,
the same resistance and the same event — a second pass would be a second place for
all three to go wrong. The `power > 0` guard is what keeps a hostile rider off a
cleanse: `resolveAgainst` never asks which side a target is on, so "already
dealing damage to it" is the available way to say hostile. ⚠️ Test that guard with
a skill aimed at an **ally** (`mend`), never a self-aimed one — `Act` returns
before `resolveAgainst` for `Target: Self`, so a self-shield passes with the guard
deleted, and that was the first version of the test.

**A shield stops the blow and the wear, but not the contamination.** A strike a
block charge ate used to deliver **nothing** — `connected` was set only in the
`Damaged` arm and `if connected { … }` gated every rider, so a blocked strike
applied no status and did not even roll for one. It now lands the riders whose
category **outlasts a shield**, which is `status.Category.OutlastsAShield` and is
`Dot` **and nothing else**: fire still burns you through a shield and poison
still gets on you, while a stat the blow never bent and a turn it never took are
stopped with the strike. The chance is unchanged — every rider still goes through
`inflict` at its own declared chance, so the same amplifiers, the same
resistances and the same `status_applied` / `status_resisted` events apply. The
same filter is applied to `b.riders(actor)`, because a trait's rider surviving a
block on a different rule from a skill's own application would be a difference no
reader could find on either.
⚠️ **A MISSED strike is unchanged and delivers nothing, a tick included**, which
is the entire justification for the rule: a block means the blow arrived and was
stopped, a miss means nothing touched the target. That is why `blocked` is carried
**beside** `connected` rather than widening it, and why the decisive test has a
missed arm of its own. `combat.Roll` checks accuracy before it offers a charge, so
a miss never even spends one.
⚠️ **Letting `stat_debuff` through as well was measured and REJECTED, and it is
not a balance number.** With `mire` unstoppable — 25% off speed a stack, two
stacks — `pokemon.squirtle` against itself **stops resolving**: 0 of 20 duels
finished inside spar's 4000-turn limit (`Endless` 40 of 40 across the row's two
arrangements) against 20 of 20 finishing with a kill, mire applications went
373 → 12875, and nothing was close to dying — every unit sat at **45%** health or
better when the limit was hit and the lowest any was driven to at any point was
**29%**. That breaks `TestABothWaysMirrorIsExactlyEven`, a **fairness invariant** — a
character duelling an identical copy of itself comes to exactly 500‰. So
`OutlastsAShield` is one case on purpose. Do not "complete" it, and **do not fold
it into `Harmful`**, which is `Dot|StatDebuff|Control|Taunt|HealCut` and answers
what a cleanse may strip; the near miss is the whole risk, which is why
`TestOnlyATickOutlastsAShield` asserts the two splits **apart**. `HealCut` was
refused entry on the **reading** rather than on a measurement: a cut is a share
taken off a number some later effect produces, so a stopped strike leaves nothing
on the target for it to be about.
⚠️ **The rule reaches 5 shipped skills and 0 shipped traits.** Ten of the 43
shipped skills both damage and apply, and only the five carrying a `dot` are
touched (`sludge_bomb`, `ember`, `flamethrower`, `fire_spin`, `heat_wave`); the
`stat_debuff` four (`bubble`, `whirlpool`, `bite`, `dragon_claw`) and the one
`control` (`water_pulse`) are unaffected, and the one `heal_cut` (`fire_fang`) is
not either — see *Healing is not damage with a sign* for why a cut is not
contamination. **No shipped trait declares `applies` at all** — the eleven use
`grants`, `resists`, `amplifies`, `replies` and `drains` — so the trait half is a
**latent** branch and
`TestATraitsRiderGoesThroughAShieldOnTheSameRuleAsASkillsOwn` is the only thing
that exercises it.
⚠️ **The "one strike eaten, one through" branch is no longer latent.** It was:
`shieldedCast` braces exactly once and every skill it measured struck once, so no
cast in the repository had two halves taking different paths, and the whole
`connected || blocked` / `throughAShield := !connected` arrangement was exercised
only at its two extremes. `fire_fang` is two strikes and now carries a rider, so
the middle case is shipped data. `TestOneStrikeEatenAndOneThroughDeliversOnce`
tables it over a `dot` and a `heal_cut` together, because the claim worth holding is
that the two reach the same answer — applied **once** — by opposite routes: the tick
outlasts the eaten strike and then rides the landing one, the cut is stopped with
the first and rides the second. A rider counted per strike would apply the dot
twice; a rider gated on "the cast connected at all" would apply the cut on the
blocked half; no single-strike fixture can tell either from correct.
⚠️ **`price.go` did not change and does not have to.** `inflictedOn` prices a
skill's own `Applies` off the *status's* chance and never weights it by whether
the strike connects (only the reply half reads `combat.Hit`), so the rating
already priced a rider as landing independently of the blow — this brings the
engine closer to that for a dot and leaves it exactly as it was for the other
categories. What is now slightly **over**-priced is the guard: `shielded` values a
charge at the strike damage it eats, and a charge no longer stops the tick riding
on that strike. It is a defensive option, so the cost is a marginal cast rather
than a kill, and correcting it is a measured change of its own.
⚠️ **The player is told, in the statuses reference and nowhere else.** The rule is
global, so it may not live in a per-skill description — those are derived, and one
clause per skill would be the rule declared 43 times. It is a property of the
**shield category**, which is where `BlurbStatusShields` already lives, so
`BlurbStatusSeeps` sits beside it in `describeStatusEffect` and
reaches `?block` at the battle prompt, `hexforge statuses` and the tool's
statuses screen, in both languages. `describe.golden` moved by exactly those two
lines and no balance golden moved at all.

**`passive.Condition` is not `skill.Condition`, deliberately.** A skill's
condition asks what the *target* carries; a trait asks about its *holder*, and the
question it wants is one no status answers. One term, `BelowHealth`, a **share**
of maximum health — points would be a different fraction of the bar at every
level. Read **live** at each site (`riders`, `resist`) through `inForce`, so a
trait stops applying the moment its holder is healed back; *at or under*; and a
share is not a fraction, so `333` of 3000 is 999 and a third exactly does not
pass. ⚠️ **A gated `grants` used to be refused at parse and is not any more** —
that refusal was removed when the engine learned to hold and release a grant as
its gate opens and closes (an event each way, a retune), so a gate over a grant
is a term now rather than an engine-only door. What is still refused is a gated
grant that **raises health** (a gate closing would take the room away, leaving a
healed unit above its own maximum) or that **holds a pool** (hold and release run
the grant again every time the gate reopens, so a barrier behind one comes back
full each time it is crossed). Both read the **effective** gate — the trait's or
the grant's own — because they are rules about the grant. → `passive.GateOver`.

**A resistance belongs at `battle.inflict`, never at `status.Set.Apply`.**
`Apply` is the choke point every status passes through, which makes it the obvious
home and the wrong one: it has no dice, so a resistance there could only refuse
outright — a hard cap on a continuous quantity, which this engine rejects
everywhere. `inflict` is where the chance is rolled, so `Battle.resist` takes its
share off that. Sources **multiply** what each lets through (two of 600 leave 160),
so stacking diminishes for free, needs no saturation helper, and can never reach
the absolute — while a declared 1000 does, which is the same division as a skill
declaring full accuracy. A single resistance is exact by construction: `surviving`
comes back as `scale.Base - amount` with nothing lost, so the chance takes one
truncation. Resistance is **by status id, not by category** — a category cannot
say "poison but not burn", and an id can name a class by listing it; only a
`Harmful()` category may be resisted, because refusing a buff is refusing your own
side's help. And the event carries `Refused`, because `status_resisted` is emitted
whether the roll failed *or* the target refused it, and a reader given only that
word cannot tell luck from a property of the unit.

**A stack does not know who applied it, and that is deliberate.** `status.Stack`
holds its frozen tick amount and its remaining turns — no applier id, because the
applier may be dead by the time the stack resolves and keeping the id would be
keeping a pointer to something that no longer exists. The consequence to know
before answering a question about attribution: `status_applied` is the **only**
place a source is recorded (it carries the actor, the skill and the frozen
amount), `status_ticked` names the unit *taking* the damage rather than the one
that caused it, and two units poisoning one target leave two stacks the state
cannot tell apart. Attribution is a property of the log, not of the state.

**`cast.Unlock` is the learnset shape, and it is about an id rather than a
trait.** `{id, at_level}`, with `UnlockedIDs` the one function answering "what is
in force at level N" — it takes the *list* rather than reading a character, so the
kit gets it unchanged when skills gain their levels. Do not write a second shape
for the kit: two vocabularies for one idea is the mistake this file keeps a list
of. Four things that are decisions: an unstated level is **one**, normalised at
parse so exactly one value in memory means "from the start" — which is why
`Unlock` carries a `MarshalJSON` that omits a level of one rather than an
`omitempty` tag; the second gate is `Stages`, an **allowlist and not a
threshold** (a threshold says "from this form on", which a level already says —
only a list can say "the bulb forms only", which is what makes giving up an
evolution buy something), and **`at_stage` was never built** because the list
says everything it would have; both gates are applied in **exactly one place**,
`seed.ParseRoster`, the only place a character, a level and a chosen form meet (a
flat entry writes out its own traits and has neither); and bringing every
unlocked trait is **not a choice**, which is why that half needed no change to
the log — the slot is where that is paid for.

**A trait is on before `queue.Add`, not corrected after it.** `battle.enlist`
calls `grant` and then adds the unit at `b.Stats(unit)` speed. A wait is
`1_000_000/speed` and the first one has been served by the time `retuneAll` would
notice, so a correction is not a fix. ⚠️ The test for it makes the holder the
**slower** unit at its base and faster only with the trait counted — two units of
equal speed pass on the tie-break whether the trait was applied first or not, and
that was the first version of it. Events are emitted in `Begin`, not `enlist`: a
battle has no log until the opening board, and a line naming a unit the log has
not introduced is one a renderer cannot place.

**Do not restate a dependency list.** `Library.ArchetypeDeps` and
`Library.CastDeps` exist because two callers parse those books — a load, and the
re-parse `EditSkill` does off the disk — and `recheckCarriers` takes them and
swaps the skill book rather than writing its own. It used to write its own, and
when passives arrived every skill edit in the repository began failing with
"archetype blighter names passives, which cannot be checked without the passive
book": a re-parse missing a book refuses on the missing book instead of on the
edit, and names a preset the author never touched.

**A health modifier on a status is allowed, and a maximum only ever rises.**
`Battle.MaxHP` resolves the health line through the modifiers, the way every
other stat resolves — it used to read `Unit.Base` and a health term was therefore
a number nothing read, which is why one used to be refused outright. Three
refusals replace the blanket one, and together they are what lets current health
stay put when a maximum moves: a health term may sit only on a **permanent**
status (`status.ParseBook`), may only be **positive** (same), and may not be
granted by a **gated** trait, whose gate closing would take the raise back off
(`passive.ParseBook`). What follows is the design: at enlistment current health
is set to the maximum *after* the traits and the bonuses are on, so a squad that
built for a health bonus starts holding it; after that a rise opens room and
never fills it. `grass_growth` is the one thing that ships on this.

**A form's art is optional, and the fallback has one home.**
`progression.Stage.Image` is a stage's own picture and most stages declare none;
`cast.Character.StageArt` is the **only** place that falls back to the
character's. A caller reading `Stage.Image` directly draws nothing for the
ordinary stage, and a second caller inventing the fallback again is how one
character ends up with two pictures depending on which screen asks. It sits in
`progression` beside `Name` because both are facts about the form rather than
rules — and a parallel list keyed by stage name would be a second thing to keep
in step, stale exactly when a stage is renamed. `progression` does **not** check
the path: `cast.ValidateImagePath` does, at parse time, per stage, and only for a
stage that names one — the empty string is a real refusal there, so an absent
image must never reach it. A character therefore has a *set* of pictures
(`Character.Art()`, distinct by path, declaration order, the character's own
first) and `Library.Inspect` walks all of them, because art only a grown form
uses is art nobody looks at until the character has grown.

**Piercing is a ratio, and it stops at the strike.** `skill.Skill.Pierce` is the
share of the target's defence a skill ignores, in parts per thousand, applied by
`combat.Pierced` inside `Rules.Strike`. `Rules.Damage` itself takes the defence
*as it applies* and knows nothing about piercing — deliberately, because five
positional integers is a signature a mis-ordered argument passes silently, and
because a caller reaching it directly is asking for the raw curve. A
damage-over-time tick is exactly such a caller and **must stay that way**: a tick
is computed once when the stack is applied and frozen for its whole life, so
piercing one is worth as many pierced hits as the stack has turns left (400 a
turn for three turns against 171, measured on the shipped poison), which is a
different skill from the one the author wrote. Three consequences: a ratio rather
than a switch, for the reason buffs saturate — a hard cap on a continuous
quantity is the shape this engine rejects; the `damaged` event **carries the
share**, because a reader who cannot see it cannot reproduce the figure and a log
its reader cannot reproduce is the log lying; and `progression.EffectiveHP` now
describes one case of two, so anything showing it to an author must show
`EffectiveHPAgainst(…, scale.Base)` beside it — which comes to the raw health.
`razor_leaf` carries the only non-zero value, 400, which buys nothing against a
bare target and 41% against the defence ceiling.

**Raw health needs no floor of its own, and not because we decided against one.**
A ratio floor — raw health as a share of effective health — is algebraically
`DefenseReduction(defence)`, a function of defence alone, so it **is** the defence
ceiling that already exists; and an absolute floor cannot work at all, because
`CheckTable` walks every level from one and every unit's health is small at level
one. Measured, it is also not needed: among lines that saturate the joint bound,
raw health runs 3128 (at the 800 defence ceiling) to 4800, a 1.53x edge fully
pierced, against the 2.25x the worst elemental matchup already swings. If that
ever proves too much, the knob to turn is `ceilings.defense`, not a new bound.

**The damage numerator is 128 bits, and the division stays single.** `Rules.damage`
multiplies five factors — attack, the skill, the affinity, the crit and
`DefenseConstant` — and that product does not fit an `int64`. It used to be written
as one `int64` expression and **wrapped silently**: at the attack ceiling against
half the defence ceiling, a power of ninety million came to four and a half million
— a large, plausible, wrong figure rather than a visibly broken one — and the
wrapped expression is **not monotone in power**, so no reading taken off a single
figure could catch it. ⚠️ **The obvious repair is refused**: dividing earlier
truncates twice, and the whole package rests on truncating once — `floor(1000a/1000b)
== floor(a/b)` is the identity the crit mechanic was built on and why adding crit
moved no damage figure anywhere. So the intermediate widened (`wide`, off
`math/bits`, carrying an `exact` flag because a silent wrap at 128 bits is the same
defect only rarer) and the division did not move. `over` saturates at
`math.MaxInt64` rather than letting `bits.Div64` panic — a panic in the damage
formula is strictly worse than the wrap it replaces, and `max_effective_hp` is
11,500, so a figure reaching that guard already kills whatever it touches. ⚠️ **That
saturation is a bound on the type and not an authored ceiling**: `Skill.Power` still
has none, deliberately — see `TODO.md` § *Decided against*.

⚠️ **`combat.Swung` is the same story and had to be widened too, because it sits
*upstream* of that numerator.** It is the one expression that composes a caster's
own terms — `(power + bonus) * (1000 + share) / 1000` — and its result **becomes**
`skillMultiplier`, so the 128 bits below it could not protect it: written in one
`int` it wrapped at a power around 9.2×10¹⁵ and handed `damage` a **negative**
multiplier, which that function's first line refuses, so an enormous power came
back dealing `MinimumDamage`. It reuses `wide` rather than getting arithmetic of
its own, which is the point of the function existing at all — the battle and the
authoring preview read *one* expression, so it may not fork into two. ⚠️ **A
cheaper "does the product still fit an int64" check was refused, and the reason is
that the divisor is one of the multiplicands' own scale**: a skill with no bonus
and no share is `power * 1000 / 1000`, which is `power` for every power the type
holds, and such a guard would answer `math.MaxInt64` to that — refusing a figure it
was handed and could return untouched, three orders of magnitude below where the
quotient stops fitting. Widening keeps that identity and saturates only where the
answer genuinely does not fit; the two saturations then compose, since a pinned
multiplier makes the damage below it pin as well. ⚠️ Its **three clamps at nought
are a refusal, not a preservation** — the one input whose answer moved — and they
exist because the arithmetic under them is unsigned: `power`, `bonus` and `share`
are non-negative on every path (`Validate` refuses a negative of the first two,
`Gradient` cannot return a negative third), and a negative power, a bonus that is a
penalty and a wound that weakens are three things no field expresses. ⚠️ **Nothing
outside `internal/core/combat` can see any of this**: the shipped book's largest
landable multiplier is 3,500, twelve orders of magnitude below the wrap, so
reverting the widening moves **no golden and fails no other package** — `swung_test.go`
is the whole guard, and `narrowSwung` in it is the pre-fix expression kept verbatim
and marked for deletion.

⚠️ **A saturation that is only produced is not a saturation — it has to be carried.**
Both widenings above end at one `return math.MaxInt64`, and the figure then travels:
into a splash share, a strike count, a weighted average, a wall of block charges, a
tally of attempts. Every one of those was a plain narrow product, so the value that
saturated at the widest the type holds came back out the other side **small, and
often negative** — the exact defect the widening was written to remove, moved one
line later. So `combat.Scaled` (a ratio in parts per thousand) and `combat.Repeated`
(a whole count) reuse `wide`/`over` rather than multiplying narrowly, `wide.plus`
exists for the one expression whose *answer* always fitted and whose working did not
(`ExpectedStrike`'s weighted average, which wrapped to **−1**), and the two tallies
`DamageDealt`/`AbsorbedBy` saturate their sums. Both `battle`'s splash shares, the
rating's own splash share, its `perStrike × connecting` and its wall of charges go
through the same two functions. ⚠️ **The property to test is monotonicity, not a
table**: a table says what the arithmetic does today, and *"never comes back smaller
for more power"* is the one thing a wrap always does and a saturation never does, so
it catches a narrow product nobody has written yet — `TestNoFigureFallsAsPowerRises`
in `internal/core/combat/carry_test.go`. ⚠️ **A strike count is the one input with no
ceiling anywhere**: `Skill.Validate` refuses a negative and says nothing about a
large one, so `Hit.ExpectedStrikes` guards its own product. ⚠️ And **the rating's
`landed > target.HP` clamp hides overflow**: a wrapped figure that stays positive is
clamped to health like a correct one, which is why the wall-of-charges product has an
arithmetic test and no board — see `TODO.md` § *Not done*.

**Healing is not damage with a sign.** Three mechanisms give health back — a
skill's `restores`, a skill's `drains`, and a `regen` status — and each obeys the
same four rules. `combat.Rules.Restore` deliberately does **not** divide by the
defence curve even though a damage-over-time tick does: defence turns away what
is coming *at* a unit and has nothing to do with what is helping it, so do not
add the division for symmetry. A drain reads `combat.DamageDealt`, not the damage
rolled, so a missed or blocked strike drains nothing — ⚠️ **and a drain is
deliberately not the rule a rider follows**: a drain is a share of damage, so no
damage is no drain, while a blocked strike's `dot` rider still lands (see *A
shield stops the blow and the wear, but not the contamination*). The two look like
one sentence about blocked strikes and are two different questions.
`status.Set.Tick` returns
**two unsigned totals**, damage and healing, never one signed number — a
negative down the damage path would subtract a negative, and `wound` calls `kill`
the moment health reaches zero, so a signed total is the one shape that could
revive a corpse. And a dead unit is not healable while health clamps at `MaxHP`,
which is what keeps a battle able to end and stops a regeneration from being an
uncapped shield. Every restore emits a `healed` event, because nothing else in
the log explains health going up. Consequence: the joint health-and-defence
budget is an **understatement** rather than a bound.

**Healing can be CUT, and the cut has one definition because there are only two
places health goes up.** `status.HealCut` is a category whose whole job is one
number — `Kind.HealShare`, permille, negative, summed **per stack** by
`Set.HealShare` the way `Set.Modifiers` sums its terms — and `battle.healingFor`
is the single expression that applies it. `heal` and `drain` both call it; `drain`
is separate from `heal` only because its event carries `Drained`, so writing the
arithmetic twice would be two answers to one question. Five sources reach those
two functions (a skill's `restores`, a skill's `drains`, a `regen` tick, a trait's
`drains`, `comeback`'s `at_empty`) and
`TestEveryHealingPathTakesTheHealCut` tables four of them against a written-down
list, so a healing source added without a row is a red test.
⚠️ **The cut comes off BEFORE the amount is capped at the room left to full.**
Capping first hands the reduction a number that is already the room rather than the
heal, so on a nearly-full unit the cut is taken out of health that was going to be
thrown away — the debuff is invisible on exactly the unit a sustain build spends its
turns being. `TestTheCutComesOffBeforeTheHealIsCappedAtTheRoom` builds the one unit
where the two orders differ (room = half the payout) and names all three figures.
⚠️ **Floored at nought, in `healingFor`, and it is ONE floor.** The callers' own
post-reduction check is written `amount == 0` rather than `amount <= 0` on purpose:
with `<= 0` there, **deleting the floor reddened nothing in the whole suite** —
the caller's guard swallowed the negative and behaviour was identical. Two floors
for one invariant is a guard a mutation deletes for free, which is the note beside
the reply drain's `damage > 0` all over again.
⚠️ **`Event.Reduced` carries the share, and it is not `Refused`.** Without it a
reader sees `heals 244` where the book says 900 and every figure they could check
against says the log is wrong. `Refused` is a share of a status application's
*chance*, already signed with negative meaning invited; a second meaning on one
field is the thing this file keeps a list of. All three arms of `tui.Line`'s
`Healed` branch print it (`reducedNote`), because each builds its own sentence.
⚠️ **`price.go` did not change, so a heal cut is priced at nothing.**
`inflictedOn` has arms for `Dot`, `Control` and `StatDebuff` and falls through for
the rest — *worth nothing means not rated* — exactly as a `taunt` does. So the
opponent never aims one at a healer on purpose and never discounts a heal it is
about to have cut; both errors run the direction every cap in that file errs in,
and every figure quoted for the status is therefore a **floor** on what it is worth.
⚠️ **A permanent heal cut is legal** (nothing refuses it the way a permanent `Dot`
or `Regen` is refused) so a trait may grant one. Nothing shipped does.
Shipped as `fester` — a verb, which is the rule for a debuff id — `max_stacks` 2,
`duration` 2, **−400 a stack**, so two stacks cut 80% and **healing is never fully
off**: the engine's standing preference is to saturate rather than hard-clamp, and
full negation at the cap is a shutdown rather than a cost. `statuses.json` takes no
comment, so that is the reason, here. Delivered by `fire_fang` at 500‰, one stack,
and `rapid_spin` strips the category. → `README.md` § *Cutting the healing* for the
measurements, the `rapid_spin` answer and the shipped-roster null.

⚠️ **A `restores` payout has TWO callers and one of them was missing.**
`Battle.restore` is the single expression; `resolveAgainst` calls it per unit a
shape reached, and the `Target: Self` branch of `Act` calls it for the caster,
because that branch **returns before the shape walk** — a self-aimed skill has no
shape to walk. It lived inline in `resolveAgainst` until it was looked for, so
every self-aimed restore paid **nothing**: `synthesis`, whose entire body is a 900
restore on itself, and `withdraw`, which paid out its block and dropped its 500.
Those are the only two shipped skills that declare `restores` at all, so the field
did nothing anywhere in the game.
⚠️ **The rating could see it and the engine could not**, which is the exact shape
*Rating an action* forbids: `pricing.restored` prices a restore off
`combat.Rules.Restore`, so `Suggest` **chose `synthesis`** on a hurt caster
expecting up to nine hundred health and received none. "A price built from a
second reading lets the opponent prefer a skill for something the skill does not
do" — except here the second reading was the honest one and the resolving function
was the copy that had gone missing.
⚠️ **No golden moved, and that is the finding rather than a relief.** No roster
unit brings either skill, which with `Act`'s early return is the whole of why a
declared field did nothing this long with every test passing — the same pair of
facts as the regeneration bug, and the same lesson: **a mechanism no shipped
placement fields is a mechanism nothing measures.** The tests are therefore a
fixture pair differing in their aim (`internal/core/battle`) and a walk over
**every** shipped skill that declares a restore (`internal/seed`), rather than a
test naming `synthesis`.
⚠️ **A balance figure taken on `withdraw` predates this.** The tank build's
survival reading — "gated on how often it can cast withdraw" — was measured with
that skill's restore dead, so it is a reading of the block clause alone.

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

**A window is not closed by the fact that you have not opened it.** The
chooser's answer slot was drained on entry, on the premise that nothing could be
in it for the turn now opening because the chooser had not yet sent *"it is your
turn"*. But the screen does not learn whose turn it is from that message — it
learns it from `socket.Mirror.Asking`, which is true the moment the room's batch
is taken in, a message and a redraw **earlier**. So a player answering off the
board already in front of them lands in the slot first, the drain ate a real
decision, and the screen — which had recorded the turn as answered — would not
offer it again. Both ends then stood still for a whole allowance. The fix is that
the answer says **which turn it is for** and the chooser asks; the `*battle.Prompt`
it had always been handed, and had never read, was the whole answer. When
something buffered has to be told apart from something stale, the discriminator
is the turn, never the moment.

**A `select` over two ready arms is a coin flip, so decide on a reading.**
`Server.Shutdown` bounded its wait with `select { case <-settled: ...; case
<-ctx.Done(): ... }`, and both arms are ready whenever the last room ends around
the moment the bound does — so the same shutdown of the same server returned
success or a refusal at random, and the refusal it wrote on that path read *"0
room(s) and 0 connected room(s) still running"*: a give-up naming nothing to act
on. Ask the thing itself (`rooms.Running()`), not the channel — a channel can
also be un-ready merely because its goroutine has not been scheduled. And a
refusal that carries numbers must be **handed** the numbers that made it refuse:
`gaveUp` takes its counts as parameters so the reading that decided and the
reading that is reported cannot disagree.

**A test that only reddens under load is a test the next person re-runs.** Both
of the above failed inside `make check` and passed alone, which is the shape that
teaches a reader to hit re-run. Neither was a flake and neither was fixed by
widening a bound: one test was wrong about its own premise (an already-done
context proves the bound is *available*, not that anything is *waiting* — so the
test now holds a table open across the call and can assert the count by value),
and the other was watching a real deadlock. Every end-to-end bound in this
repository is a **hang detector**, not a performance budget: when one fires,
what it was waiting on is the finding.

**Watch the arithmetic, not the intent.** Several assertions written from a hand
calculation were wrong while the code was right — a saturation gap taken from the
wrong side, a drift bound of one when each of two truncations can lose one, a
guess that cleansing earlier is always better when the attacker simply reapplies.
When a test disagrees with a hand-derived number, re-derive before touching the
code.

## Data and golden files

Balance lives in `internal/seed/data`, embedded with `go:embed`. Changing a number
there changes the game without touching Go.

⚠️ **There are SIXTEEN of them since `bonuses.json`, and the name is spelled in
three independent places** — the `//go:embed` directive in `internal/seed/seed.go`,
the `dataFiles` slice in `internal/seed/digest.go`, and a `XxxFile()` accessor
beside the other fifteen. Missing the second is the silent one: the file loads and
the **data digest stops covering it**, so two peers on different bonus data would
pass the gate and then diverge, which is the failure the digest exists to prevent.

⚠️ **The rest of this section moved to `docs/goldens.md` on 2026-09-07**, whole:
what each golden under `testdata` is a record *of* and what moved it, the shape an
authoring write leaves a committed file in (`skills.json`, `cast.json`,
`origins.json`), why editing a skill can break what adding one cannot,
`roster.json`'s four contracts and `builds.json`'s catalogue. Read it before
accepting a golden or touching `internal/seed/data`.

Run `make golden` (`go test ./cmd/hexarena-tui ./cmd/hexforge-tui
./internal/core/hex ./internal/i18n ./internal/screen ./internal/seed
./internal/tui ./internal/wire -update`) to accept a change and
then **read the diff**. That diff is what the files are for: a balance change that
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

**What a restriction can enforce, and what it cannot.** A skill may declare
`restrict`, an optional allowlist of `elements`, `archetypes`, `characters`,
`species` and `origins` (any list absent means unrestricted; a list **present and
empty is an error**, because an allowlist nobody satisfies is a mistake every
time). They are not enforced in the same place, and the split is a layer fact
rather than an omission:

- **`elements` reaches the engine.** It is checked by `skill.WhyCannotCarry`
  beside the shared-element rule, so `battle.enlist`, `cast.ParseBook` and
  `forge.CheckSkill` all apply it from one declaration. `CanCarry` is now that
  function's yes/no. The two element refusals are *different answers*
  (`CarryWrongElement`, `CarryElementRestricted`) because they need different
  advice: one is fixed by taking the skill's element, the other cannot be, since
  the skill's element is already shared. The list is what makes a **neutral**
  skill restrictable at all.
- **The other four cannot reach the engine.** `battle.Roster` carries stats,
  skills, an affinity and a slot — no archetype, no character identity, no
  species and no origin — because all of those are resolved *before* a battle
  starts. So they are **authoring-time only**, enforced in `cast.ParseBook`
  (`resolveCharacter`). **Do not push any of them into `battle` to "complete" the
  feature**: it would put a fact into the replayable core that no replay reads,
  and `battle.Roster`'s deliberate emptiness is recorded above for the same
  reason.

`skill` itself validates only what it can see — the element names are real, no
list is present-but-empty, no entry blank or repeated — because `cast` imports
`skill` and the reverse would be an import cycle. Archetype, character, species
and origin *names* are therefore checked one layer up, exactly like a skill's
pattern and status names. Two consequences worth knowing:

- The character allowlist is checked **after the whole cast has been read**
  (`checkCharacterRestrictions`), so it may name somebody declared further down
  the file.
- It is checked **only for skills somebody carries**, and that avoids a real
  deadlock: a unique skill cannot be authored after the character that carries it
  (the kit names the skill) and that character cannot be authored after the skill
  (the restriction names the character). The skill goes in first, carried by
  nobody, and is checked the moment a carrier exists.

**A skill kept for named characters may not sit in an archetype preset's kit**,
and that check lives in `cast.resolveArchetype` — the only place holding both
the preset's id and each skill's restriction without a second lookup. A preset is
the starting point for *every* character built from it, so a kit entry only
certain characters may carry would refuse everyone else, and the refusal would
land on the author of the character rather than the author of the preset. The
same function refuses a preset whose kit holds a skill kept for a *different*
archetype. **The species half of that rule is what keeps the two axes apart**: a
preset says how a character fights and nothing about what it is. ⚠️ **The price is
real and both presets have paid it**: `scorcher` gave up the two lineage skills and
`blighter` gave up `ingrain` and `synthesis` when those moved onto `plant`, so each
suggests **seven** skills while its character carries nine. Do not "fix" that gap by
moving a skill back onto `elements` — grass was only ever a proxy for "something
that grows", and a grass construct with no roots could take both. A preset losing an
entry is the smaller loss.

⚠️ **There is deliberately no origin version of that ban, and it looks like a
hole.** The sentence is just as true of a work — a preset is shared, so the
refusal would land on the next author — but the arithmetic is not: a lineage is
exceptional and a work is universal. Every skill in the book comes out of some
fiction, so banning world-restricted skills from presets would **empty every
preset in the directory** rather than trim two entries off two of them, and
`resolveSkills` refuses an empty kit ("knows no skills"). `summoner` is the
proof: its kit is one origin's six skills exactly. What is left of the harm lands
where it belongs — a character out of another work built from a preset *without
naming its own skills* is refused by `forge.CheckKit`, in a sentence that says
which work the kit is out of. `TestAPresetMayHoldASkillKeptForAWork` holds the
asymmetry down so it is not "fixed" later.

What `resolveArchetype` deliberately does not attempt is whether an element allowlist
and the kit's `Demands` are jointly satisfiable — that needs the element chart,
which a preset is validated without, the same gap
`TestEveryShippedArchetypeKitIsCarryableAtAll` covers for `Demands`.

`skill.Demands` is the other half, derived rather than authored:
`Archetype.Demands` is the distinct non-neutral elements a kit requires, filled
in `ParseArchetypes` and tagged `json:"-"` so a data file cannot claim a demand
its kit does not have — an authored hint would only be caught when a character
built from the preset was refused. A kit demanding more than two elements is
rejected outright, because no affinity can hold three. Whether the two it
demands are *allowed together* needs the element chart, which a preset is
validated without, so that half lives in
`TestEveryShippedArchetypeKitIsCarryableAtAll`.
