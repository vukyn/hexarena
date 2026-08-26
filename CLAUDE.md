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
lipgloss — all three at **v2**, under `charm.land/…/v2` rather than
`github.com/charmbracelet/…`, which is where that project moved them — and the
rest of the module currently happens to need nothing beyond
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
go run ./cmd/hexforge skills                     # the declared skills and who may carry each
go run ./cmd/hexforge skills add oath --power 1200 --accuracy 900   # author a skill
go run ./cmd/hexforge skills edit oath --power 1100                 # change one already in the book
go run ./cmd/hexforge check                      # parse the books from disk and verify the art exists

go run ./cmd/hexforge-tui                        # the same authoring, full screen (needs a terminal), in Vietnamese
go run ./cmd/hexforge-tui --lang en               # ...in English; HEXARENA_LANG=en does the same, ctrl+l toggles

go test ./...
go test ./internal/core/hex ./internal/i18n ./internal/seed ./internal/tui -update   # accept new goldens
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

### What bubbletea v2 moved, and the four things it broke silently

The migration was made for one reason: **a terminal cannot deliver the Command
key over the classic escape sequences.** There is no encoding for it, and v1's
`tea.Key` carried only `Alt`, so ⌘S did not exist as far as a program was
concerned. The Kitty keyboard protocol does carry it, v2 parses that protocol,
and a Command key now arrives as `tea.ModSuper`. `cmd/hexforge-tui/savekey.go` is
the single declaration of which keystrokes save; all three forms ask `isSaveKey`
rather than matching a string of their own.

⚠️ **That does not make ⌘S universally available and nothing in this repo can.**
The terminal has to speak the protocol (kitty, Ghostty, WezTerm, foot, iTerm2
with CSI u on — Terminal.app never), it has to pass ⌘S through instead of opening
its own Save dialog, and on Linux a window manager may claim Super first. So
`ctrl+s` stays the binding that always works and `saveKeyLabel` keeps naming a
control-S on every platform. Do not "simplify" the footer to ⌘S alone on macOS.

⚠️ **The footer names `ctrl+s` and nothing else, on every platform.** Not an
oversight and not a downgrade of ⌘S, which still saves: ⌘ is East-Asian-Ambiguous
width, so `lipgloss.Width` counts one cell where many terminals draw two, and the
glyph is then drawn on top of the character after it — `⌘S` renders as two
overlapping characters. A program cannot detect which sort of terminal it has.
Spacing it apart needs a cell that is not there: the English character-form
footer is 73 cells without the label, the smallest window is 80, the last cell of
a row is left empty so writing it cannot wrap the line, and no ASCII spelling of
both keys fits in the six that leaves. ⌃/⇧/⌥ are ambiguous too, so they are not
the way out either. Guarded by `TestTheSaveLabelIsDrawableEverywhere` (label is
ASCII) and `TestEverySaveFooterFitsTheSmallestWindow` (the six cells). ⌘S is
announced in `MenuNote` instead, which `TestTheMenuFitsTheSmallestWindow` keeps
inside the window.

Four API changes matter, and three of them fail *quietly* rather than at compile
time — which is why they are written down:

- **`Model.View` returns `tea.View`, not `string`**, and the alternate screen is
  a field on it rather than `tea.WithAltScreen()` on the program. `model.View`
  wraps `model.screenContent`, which is the string the tests read.
- ⚠️ **A bare space stringifies as `"space"`, not `" "`.** `uv.Key.String`
  returns `Text` only when it is not a single space, so space falls through to
  `Keystroke()` and comes out named. Every `case " "` compiled fine and matched
  nothing.
- ⚠️ **Colour is the program's decision now, not the library's.** lipgloss v2
  writes escape codes unconditionally and the program downsamples for the
  terminal it is attached to, so a `textinput` on its own defaults keeps its
  colours under `NO_COLOR`. `newInput` / `newInputStyles` in `style.go` restore
  the palette's rule; under v1 the library detected the missing terminal and did
  it for free, which is exactly why it is named now.
- ⚠️ **A virtual cursor is drawn as reverse video**, which is an escape code, so
  `newInput` turns it off on a plain terminal. That is not a regression: v1's
  renderer stripped the attribute itself, so the plain path never had a cursor
  either.

Two mechanical ones: a key is `Code`/`Text`/`Mod` rather than `Type`/`Runes`
(see `numberKey`), and `textinput.Width` is now `SetWidth`.

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

**Where a form beats a prompt: the kit, and a skill's damage.** Two things the
full-screen client does that the command line cannot, and both are `internal/forge`
answers rather than screen logic:

- The kit is a **multi-select over the skill book** (`cmd/hexforge-tui/picker.go`),
  not a typed list. Every skill is listed, including the ones this character may
  not take, each marked and captioned with who it *is* for — a hidden skill reads
  as a skill that does not exist. The availability of a row is
  `forge.CheckSkill`'s answer, the same value the write refuses on, so the mark
  and the refusal cannot disagree. Nineteen rows do not fit beside a form (the
  form is nineteen body lines of the twenty it has in an 80x24 window), so the
  list is a **sub-screen that scrolls**, and `(*pickState).room` counts what the screen
  spends — including the empty string a trailing newline leaves when `frame`
  splits the body, which was miscounted first time and truncated the list.
- The new-skill form shows **expected damage as the power is typed**, from
  `forge.Library.PreviewDamage`, which is `combat.Rules.Damage` against the
  attack ceiling and *half* the defence ceiling. Those two are not a tasteful
  guess: they are the pair `skills.golden`'s own damage column is measured from
  (800 and 400), so the figure before a write is the figure the golden shows
  after one. It truncates **per strike** rather than once over the total, as a
  battle does — three strikes of 600 are 615, not 617. The row drops its
  reference pair rather than its figures when it will not fit
  (`Lang.DamageWithin`): the pair is identical on every skill and named in the
  field's own help, while the two numbers being authored are the reason the row
  exists.
- Every field carries a **help line describing the focused one**, and a shape
  chooser draws the cells it covers on a sub-screen built from `pattern.Targets`
  and `hex.Render`, so the drawing cannot disagree with what the engine catches.
  Both replaced a static footnote that stated the parts-per-thousand convention
  and was not read. A field that needs a *list* — the statuses a skill inflicts,
  and the three allowlists — opens the same picker as the kit; each picker names
  its own hint, because the kit's (order, and what this character cannot take)
  says nothing true about an allowlist.
- A skill's **Vietnamese name is data, not Go**: `skill.Skill.Name` is opaque
  display text and `internal/core` never learns what a language is. An authored
  name wins, the compiled table in `internal/i18n/gloss.go` answers when there is
  none — which is still the case for all nineteen shipped skills — and the bare
  id when there is neither. Any table showing it drops the column rather than
  drawing it empty.

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

**That prompt order is a prompt's problem, not a form's.** At a prompt an answer
once given is given, so one order has to be right. On the form both fields are on
screen and **either may be filled in first**: `forge.Carrier` says an unanswered
fact restricts nothing, so with no element yet the picker marks nothing, and with
an element settled it marks the skills that element cannot take. Going the other
way, the carry line under the form refuses an element the chosen kit cannot take
and names the skill. Neither direction mutates the other's answer — changing the
element does not empty the kit, it turns those rows into marked rows and the carry
line red, which is a state an author can see and fix rather than one that happened
behind them. Do not "tidy" that by dropping the offending skills: a silent
mutation is how a one-way order comes back, and it makes the live check a lie.

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

**Every FIGURE in a skill description is derived; exactly one clause is authored**
(`internal/i18n/describe.go`, in **both** languages). `Skill.Flavour` opens the
sentence and the numbers are appended. ⚠️ **A flavour clause may not contain a
digit** — `skill.ParseBook` refuses one that does, and that single rule is the
whole of why authored prose is safe here: no number in it, nothing a balance
change can make wrong. Digits rather than a percent sign, because "110" and "gấp
2" are the same mistake. English has none (like `Skill.Name`, it is authored once
in Vietnamese) and falls back to the derived opening.
⚠️ **A clause obeys the same body rule as the name** — `withdraw` was renamed off
"thu mai" because anybody may carry it, and the first clause written for it said
"rụt hết vào trong mai": the same defect through a field the name test does not
look at. `TestAFreeSkillsFlavourNamesNoBodyItMayNotHave` checks an **unrestricted**
skill's clause against a hand-written `bodyWords` list; a restricted skill is
exempt because its restriction guarantees the body. It cannot tell whose shell is
meant, so a body word about anything trips it — reword rather than argue.
`?N` at the battle prompt describes the Nth offered skill, `?TAG` a unit's traits;
both reprint the menu and cost no turn. Every number comes from the skill itself,
because an authored line survives its own numbers moving — "doubles" outlives a
bonus falling 1000 → 700 — and this repo already refused that trade in
`Archetype.Demands`. ⚠️ **Never add a `description` field to skills.json** — `flavour` is one clause,
not a description, and the difference is that it holds no figures.
Numbers are **shares of a stat** ("100% công"), never damage figures: a figure is
true for one caster against one target and false at the next buff. ⚠️ The block is
**Vietnamese on an otherwise English screen** — a stated cost, not an oversight;
translate the whole battle screen in one piece or not at all. It borrows
`Lang.Gloss` rather than growing a second vocabulary (two names for one status is
how they drift) — and in **English a bare id is the name**, so `poison` in an
English sentence is the reading working, not a missing gloss. Two entrances:
`?N` / `?TAG` at the battle prompt, and `?` on the hexforge-tui skill listing,
which raises `screenBlurb`. ⚠️ **The forge form is not the place for it** — 19
fields already show 13 of themselves in an 80x24 window, so a three-line block
under the form costs a quarter of the fields; a screen costs nothing until asked
for. `internal/i18n/testdata/describe.golden` covers **every** shipped skill and
trait in **both** languages, so a balance change moves a line there — that diff is
how a number change reads to a player. ⚠️ English needs singular wordings where
Vietnamese does not (`BlurbCostCooldownOne`, `BlurbStripsOne`): two keys rather
than a plural rule, because a rule would make Vietnamese pretend it has a
distinction it does not.

Event kinds, sides and outcomes serialise **by name**. Do not change that to a
number: a saved log would silently reinterpret itself the next time a constant was
inserted. Renaming one breaks existing logs, so treat the names as the wire
format.

**`hex.SideNone` is the zero value, and that is load-bearing rather than tidy.**
A side is written with `omitempty`, so whichever side is zero never reaches the
wire — with `SideAlly` at zero, every ally unit's `started` event wrote no side
at all and a reader recovered it from a missing field, correct only because ally
was declared first. A battle with no winner wrote the same thing and meant
something else entirely. Now nothing is at zero: an absent side means nobody, a
won battle names its winner, and a draw names none. `Side.Fights()` is the check
for "is this one of the two", and `battle.New` refuses a roster entry that never
said. ⚠️ **This broke the log format once** — logs written before it fail
`--verify`, because the re-run produces `ally` where the file has nothing. That
was worth doing once, on throwaway files, to stop the format depending on a
coincidence. Do not put a real value back at zero.

## How a battle ends, and why a draw is an outcome rather than an error

Every ending comes through one `Ended` event carrying a `battle.Outcome`:
`Victory` (its `Side` names the winner), `Annihilation` (both sides empty) or
`Stalemate`. **Do not read the ending out of a note or out of what stopped
happening** — a renderer and `--verify` both switch on it, and an ending only
English can tell apart is one neither can be trusted with. `Battle.Outcome()`
is the same fact for a caller holding the battle; `Winner()` cannot say which of
the two draws it was.

`Stalemate` exists because **nothing on this board moves**. Reach is fixed when a
unit is enlisted and the enemies worth reaching are not, because they die, so two
short-ranged survivors on opposite back columns is a battle that can never
finish — seed 18 once skipped 3955 of 4000 turns and came back as a battle that
never ended rather than as a result.

Three constraints hold it together, and each is a way it was nearly got wrong:

- **`checkEnd` and `settle` are separate on purpose.** `checkEnd` asks only "is a
  side empty" and is called from `kill`, which happens in the middle of a skill
  still choosing targets. `settle` adds the deadlock test and runs only where a
  turn has finished and the board is at rest — the end of `Act`, the end of
  `Pass`, and Advance's two skipped-turn returns. Asking the deadlock question
  from `kill` would read a board nobody will act from.
- **`frozen` is a pure predicate, not a counter.** For every living unit: nothing
  timed on it (`status.Set.Timed`, which ignores a trait's permanent status) and
  no skill with a legal aim, **cooldowns ignored**. Both halves are pessimistic,
  because a draw declared on a battle that would have resolved is worse than the
  turn limit catching a real runaway. A count of quiet turns would be one more
  thing two runs of the same seed could disagree about.
- **Cooldowns and control are deliberately not read.** They are what make a
  skipped turn ordinary, and not reading them is exactly why an ordinary skipped
  turn cannot be mistaken for a deadlock. A poisoned deadlock is not a deadlock:
  the poison ends the battle by emptying a side.

Note a **self-targeting skill always has an aim**, so a unit that can still buff
itself is never frozen — correctly, since it can still act. A battle of two such
units is a genuine runaway and is what the turn limit is now for; `RunToEnd`
still returns without an error there, and the caller reads `Finished()`.

The cheap half is caught earlier. `battle.New` refuses a roster holding a unit
that can aim at nobody from its slot, and `hexforge check` **warns** — a
`forge.Warning`, not a `forge.Problem`, so the check still passes — when a
character's longest range cannot reach anybody from its archetype's column. Both
are necessary and neither is sufficient: reach shrinks as units die.
`hex.ReachNeeded` is where the column-to-distance answer lives, measured through
`hex.Place` rather than written down.

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

**`passive.Condition` is not `skill.Condition`, deliberately.** A skill's
condition asks what the *target* carries; a trait asks about its *holder*, and the
question it wants is one no status answers. One term, `BelowHealth`, a **share**
of maximum health — points would be a different fraction of the bar at every
level. Read **live** at each site (`riders`, `resist`) through `inForce`, so a
trait stops applying the moment its holder is healed back; *at or under*; and a
share is not a fraction, so `333` of 3000 is 999 and a third exactly does not
pass. **A gated `grants` is refused at parse** — a grant is applied once and its
status is permanent so nothing can dispel it, so gating one is a mechanism (an
engine-only door into a permanent status, an event each way, a retune) rather than
a term. Accepting it would ship a trait whose gate was silently ignored.

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

**A health modifier on a status is refused.** It has always done nothing —
`Unit.MaxHP` reads the *base* line and nothing reads the modified health at all —
so the status would apply, appear in the log and change no visible number. It is
refused rather than fixed because raising a maximum mid-battle has to decide
whether current health follows it up, and lowering one has to decide what happens
to a unit already above the new maximum. A passive is what makes this reachable:
"more health" is the most obvious trait anybody would write.

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

**Healing is not damage with a sign.** Three mechanisms give health back — a
skill's `restores`, a skill's `drains`, and a `regen` status — and each obeys the
same four rules. `combat.Rules.Restore` deliberately does **not** divide by the
defence curve even though a damage-over-time tick does: defence turns away what
is coming *at* a unit and has nothing to do with what is helping it, so do not
add the division for symmetry. A drain reads `combat.DamageDealt`, not the damage
rolled, so a missed or blocked strike drains nothing. `status.Set.Tick` returns
**two unsigned totals**, damage and healing, never one signed number — a
negative down the damage path would subtract a negative, and `wound` calls `kill`
the moment health reaches zero, so a signed total is the one shape that could
revive a corpse. And a dead unit is not healable while health clamps at `MaxHP`,
which is what keeps a battle able to end and stops a regeneration from being an
uncapped shield. Every restore emits a `healed` event, because nothing else in
the log explains health going up. Consequence: `Suggest` never picks one, and the
joint health-and-defence budget is now an **understatement** rather than a bound.

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

`skills.json` is the exception that is **balance and tool-written at once**:
`hexforge skills add` and the full-screen client's skill form both append to it,
and `hexforge skills edit` and the same form (opened with `e` on the listing)
both change what is already in it. That is why it is committed in the form
`Book.Marshal` writes, on the same terms
as `cast.json` below, and why a save says **the golden files have moved** rather
than only that it wrote — a power reaches `skills.golden`, `scenarios.golden` and
`progression.golden`'s hits-to-kill ladder, so `make golden` and reading the diff
is the next step and not an afterthought.

Two things about that write are worth knowing before touching it:

- `skill.Book.Marshal` keeps **declaration order** where `cast.Book.Marshal`
  sorts by id. A cast is a set looked up by id; a skill book's order is authored
  information (basic attacks, then the elemental ones, then utility, which is the
  order `skills.golden`'s table reads in), so sorting would shuffle a design
  record to buy the one-block diff that appending already gives. `skill.Book.Replace`
  is the same fact for an edit and is why it keeps a skill's **position**:
  reordering on a one-field change would rewrite the whole file and the whole
  golden table.
- `Skill.MarshalJSON` builds the **parse shape** (`skillFile`) rather than
  carrying tags of its own, so the only fields that can be written are the fields
  the parser reads. That is what makes the rewrite lossless for the four blocks
  the authoring form does not ask about — `requires`, `strips`, `scaling`,
  `self_applies` — and `TestTheShippedSkillBookSurvivesBeingWritten` measures it
  on the real data rather than on a fixture. A field added to one struct is a
  compile error in the other until it is added there too, which is the point.
  Note `Scaling`'s zero value is **not** its default (the zero stat is health,
  and a skill scaling off health is refused), so a `skill.Skill` built in Go must
  set `skill.DefaultScaling()`; the refusal is loud rather than silent on purpose.

**Editing a skill can break what adding one cannot**, and that is the whole shape
of `forge.Library.EditSkill`. Nobody carries a new skill; shipped units carry an
edited one, so changing an element or narrowing a `restrict` can leave an authored
character — or an archetype preset's kit — no longer allowed to hold it. So an edit
re-parses `archetypes.json` and `cast.json` **off the disk** against the edited
book *before* writing anything, and a refusal from either is the thing to prevent
rather than to discover: a written file that then fails to load is a data
directory the game does not boot from. The re-parse is the authority; naming *who*
would break is a classification after the fact (`brokenPreset` / `brokenCharacter`
walking `CheckKit` and `CheckPresetKit`), exactly like `checkAffinity` classifying
after the element chart has already said no — nothing there can turn a no into a
yes, and a refusal neither walk recognises names nobody and keeps the parser's own
words. `forge.CheckPresetKit` is the preset half of that, bringing forward
`cast.resolveArchetype`'s two restriction rules the way `CheckSkill` brings
forward `resolveCharacter`'s three. Two other things an edit must keep: an
**absent** field and a field set to **zero** are different answers (hence
`SkillEdit`'s pointers and `FlagSet.Visit` in `cmd/hexforge`, and an explicitly
empty list is how a restriction is cleared), and a skill's **id is not editable** —
a rename has to cascade through every kit and every `restrict.characters` list, so
`SkillRenameError` says so rather than half-doing it.

Three of those files are the cast rather than the balance, and `cmd/hexforge` both
reads and **writes** them:

- `origins.json` — the works characters are borrowed from. Hand-editable;
  `hexforge origins add` appends to it.
- `archetypes.json` — the role presets: a suggested stat curve and kit per role.
  This is what stands in for a character **class**: a class was in the original
  design and was dropped deliberately, because with skills declared as data the
  curve and the kit already carry what a class name would have. So an archetype
  has **no mechanical effect** — it never reaches the engine, `battle.Roster`
  carries no archetype, and nothing branches on one. Do not add a class without
  deciding what it would do that a stat curve and a kit cannot; the live proposal
  is to give an archetype its first mechanical weight through a passive instead.
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

**`roster.json` is an instrument, not a scenario, and it has three contracts.**
It is 3v3 by character reference — ally Venusaur 60 / Wartortle 16 / Charmander 8
against enemy Blastoise 60 / Charmeleon 28 / Ivysaur 16 — and each of those is
load-bearing:

- **No unit on both sides.** It used to be the same character three times per
  side, and a mirror cannot measure anything: a change helps both squads by
  exactly as much, so the win rate moves only by noise. That is what stopped
  `razor_leaf`'s pierce being judged by anything but its damage table.
  `TestTheShippedRosterIsNotAMirror` compares the **resolved** units — name and
  stat line — because a species and a level resolve to those, and two units
  agreeing on them are the same unit however they were authored.
- **Every unit reaches every enemy.** `battle.New` only refuses a unit that can
  reach *nobody*, which is right for a game; the seed roster is held to the
  stricter rule by `TestEveryShippedUnitCanReachEveryEnemy`, because a battle that
  cannot finish measures nothing. ⚠️ Slot `1,2` is **four** cells from the enemy's
  own `1,2` — past every range in the cast — and a draft that used it stalled 5
  seeds in 4000, not even as a draw: a survivor kept refreshing a regeneration, so
  something was always pending and `frozen` correctly never fired. Check a new
  slot against `hex.Place` rather than against the picture.
- **Both trait states and all three stages are in play.** Charmander at 8 is below
  `blaze`'s unlock level and the two 16s sit exactly on `endurance`'s, so a battle
  exercises a unit with its trait and one without. Since `blaze` became gated it
  carries a third state as well: Charmeleon holds it from the opening board and
  only comes *into* it partway down, which is what a shipped log now shows a
  `passive_held` for mid-battle.

⚠️ **The 40-seed sweep in `TestSeedBattlesFinishFromEverySeed` is a smoke test, not
a measurement.** It read 45 per cent on a draft whose true rate over 4000 seeds
was 55. Tune levels against a few thousand seeds and quote that figure; the test's
job is only that the battle finishes and that neither side is a scripted defeat.

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

Run `make golden` (`go test ./internal/core/hex ./internal/i18n ./internal/seed
./internal/tui -update`) to accept a change and then
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

**What a restriction can enforce, and what it cannot.** A skill may declare
`restrict`, an optional allowlist of `elements`, `archetypes` and `characters`
(any list absent means unrestricted; a list **present and empty is an error**,
because an allowlist nobody satisfies is a mistake every time). The three are
not enforced in the same place, and the split is a layer fact rather than an
omission:

- **`elements` reaches the engine.** It is checked by `skill.WhyCannotCarry`
  beside the shared-element rule, so `battle.enlist`, `cast.ParseBook` and
  `forge.CheckSkill` all apply it from one declaration. `CanCarry` is now that
  function's yes/no. The two element refusals are *different answers*
  (`CarryWrongElement`, `CarryElementRestricted`) because they need different
  advice: one is fixed by taking the skill's element, the other cannot be, since
  the skill's element is already shared. The list is what makes a **neutral**
  skill restrictable at all.
- **`archetypes` and `characters` cannot reach the engine.** `battle.Roster`
  carries stats, skills, an affinity and a slot — no archetype and no character
  identity — because evolution and archetype are resolved *before* a battle
  starts. So both are **authoring-time only**, enforced in `cast.ParseBook`
  (`resolveCharacter`). **Do not push either into `battle` to "complete" the
  feature**: it would put a fact into the replayable core that no replay reads,
  and `battle.Roster`'s deliberate emptiness is recorded above for the same
  reason.

`skill` itself validates only what it can see — the element names are real, no
list is present-but-empty, no entry blank or repeated — because `cast` imports
`skill` and the reverse would be an import cycle. Archetype and character *names*
are therefore checked one layer up, exactly like a skill's pattern and status
names. Two consequences worth knowing:

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

## Species: what a unit is

Shipped. An element says what a unit is made of and an archetype says how it
fights; `cast.Species` says **what it is** — a shell, roots, a lineage — and it is
deliberately the thinnest axis in the repository: an id, a word for a screen, an
optional note, and no stats, no kit and no rule of its own.

- **A fourth allowlist, not a fourth concept.** `skill.Restriction.Species` sits
  beside the other three and composes the same way, so `dragon_rage` and
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
- Still absent: **vulnerability**, the mirror where a target is *easier* to
  affect. It is `Resists` with a negative share and reuses the whole composition,
  but it needs one decision first — a negative `Refused` in the log, and the
  early return in `resist` that treats "nothing refused" as "nothing to do" would
  silently drop it.

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
- [x] **A gated grant: a stat change that comes and goes.** `blaze` is now what
      it is named after — `{"grants":[{"status":"kindled"}],"while":{"below_health":333}}`
      — and its burn immunity moved to `heatproof`, because **a gate covers the
      WHOLE trait** (grants, resists and applies together) and a trait wanting one
      gated half is two traits. Four pieces: `Set.Hold`/`Set.Release` are the
      engine-only door (each refuses what `Apply`/`Remove` handle, so neither
      becomes a second one of those, and `Remove` still refuses a permanent status
      so no cleanse can dispel a trait); `PassiveHeld` now fires mid-battle too and
      `PassiveReleased` is the way back; a **retune** at the crossing — not what
      keeps the queue right (a turn already ends with a sweep) but what puts the
      `speed_changed` NEXT TO the trait that caused it; and one re-evaluation point
      per unit whose health moved, in `battle.reconsider`.
      ⚠️ **The plan was wrong about where health moves.** Not `wound` and `heal`
      and nowhere else — **three** places: the strike loop in `resolveAgainst`
      subtracts from its target directly, and that is where nearly all the damage
      is dealt. Hooking only the two named functions opens a gate for a poison tick
      and never for a sword. It is read **per strike**, not per skill, or the same
      trait would be worth less against a multi-strike skill for a reason written
      on neither. Guard on `unit.HP <= 0` as well as `Dead`: the strike loop leaves
      a target at zero and kills it afterwards, so a flag-only guard announces a
      trait to something whose `died` line is the next event.
      ⚠️ **A gated trait is nearly a one-way door on autopilot** — `Suggest` never
      heals, the only healing in 60 bench battles is a drain to its own caster
      (~1/40 of a bar against damage worth ~1/10), and across 4000 battle-seeds of
      every arrangement tried a trait came back off **once**. So `passive_released`
      is proved by a **hand-played** battle in `TestEveryEventKindIsReachable`, not
      by widening the sweep until the rare case shows up.
- [ ] **Two builds for one Bulbasaur — the target the trait entries add up to.**
      (1) *poison specialist*: immune to poison · its poison hurts more ·
      attacking it poisons the attacker. (2) *bloodsucker*: heals from damage
      dealt · heals more the closer it is to dying. Pieces: `Resists` ✅ ·
      *Amplifying a status* ✅ (`virulence`, carried from Venusaur onwards) ·
      *Answering back* ✅ (`venom_blood` replies) — **so build (1), the poison
      specialist, is complete**: immune, sharper, and it bites back ·
      **passive lifesteal** ✅ · `While` ✅ (`blaze` is gated).
- [x] **Passive lifesteal — SHIPPED.** `passive.Passive.Drains`, a share of the
      damage its holder deals, added to the skill's own drain and resolved where a
      drain already is. `blood_thirst` 250 · `last_gasp` 400 gated at 400.
      A gated share is the **cheaper** of the two gates (no door into a permanent
      status, no event either way, no retune), not the only legal one: #62 made a
      gated grant work too, so *Overgrow* and *Vladimir* are both writable and
      only the cost differs.
      ⚠️ Trait shares **add**, never compose — a share of damage dealt is not a
      chance — and the total is **capped at the base**. That cap is a
      *conservation* (cannot take back more than was dealt, the same invariant
      `skill.resolve` holds on one share), not the buff ceiling this engine
      rejects; saturating would pay 285 for a trait that says 400.
      ⚠️ The `Healed` event carries **`Drained`**, the share, because `Amount`
      alone cannot say why once a trait can drain too — the `Pierce`/`Refused`
      trap again.
      Both are on Bulbasaur's learnset now (`blood_thirst`@20, `last_gasp`@40),
      which needed the trait slot first: before it a character brought everything,
      so a fourth trait made one unit better rather than two different. At the cap
      the **one** slot decides between five traits of four kinds — resist+reply,
      amplifier, stat change, drain, gated drain. `TestBulbasaurCanBeBuiltTwoWays`
      measures that the *choice* exists, and fails if `TraitSlots` grows or a
      direction loses its last entry.
      ⚠️ `Applies` is NOT retaliation, and is still not: it adds to what the
      holder's **own** attack inflicts (touch → poisoned). Retaliation is
      `Replies`, which is built.
      ⚠️ **Circular, so choose rather than discover it**: a character brings every
      trait it has, so all five pieces make ONE better unit, not two different
      ones — choosing needs the **trait slot** (*Learnsets and slots*, built), and that entry
      says a slot is only a decision once traits differ in **kind**. Suggested
      order: nothing. The gate, the reply, the drain, the amplifier **and** the
      trait slot are all built, so the five pieces are five things a placement now
      chooses **one** of — a resistance, a gated grant, a reply, a drain and an
      amplifier are five different sorts of thing, which is exactly the "differ in
      kind" the slot was waiting for. What is left is putting them on characters.
      See README → *Two builds for one character*.
- [x] **Answering back — the fifth job.** `venom_blood` now costs whatever bit
      into it: `"replies": {"power": 40, "applies": [{"status":"poison","chance":25}]}`.
      ⚠️ **Not `applies` reworded** — that fires on a target the holder chose,
      during the holder's turn; a reply fires on the **attacker**, on somebody
      else's turn, from a unit that is not acting. **Not a second damage path**:
      damage through `combat.Rules.Damage`, statuses through `battle.inflict`, so
      the same resistances, the same rolls and the same event kinds. `inflict` no
      longer takes a `skill.Skill` but an **`origin`** (`{Skill|Passive, Element,
      Scaling}`) — the three things it ever wanted from one — which is what lets a
      reply share it rather than fork it. A reply has **no element and no
      accuracy**: it is neutral and it lands, because the chart prices what one
      creature *threw* and an accuracy roll asks whether contact happened, which
      it already has.
      Four rules, three of which are just where `b.answer` sits (after the whole
      skill, once, per holder): a reply **may kill** (a battle can end on a turn
      nobody took); a reply **never triggers a reply** — closed because the
      answer list is built from the skill's own targets and a reply is not one,
      *not* by a depth counter; **once per USE, not per strike**; **every target
      takes the whole skill first**. A holder the skill killed does not answer,
      and once one reply kills the attacker the holders behind it do not either.
      ⚠️ **A reply is priced by how often its holder is attacked and how long it
      survives, and no number on the trait can say that.** Both Bulbasaurs hold
      `venom_blood`, but the ally fields Venusaur at 60 and the enemy Ivysaur at
      16: over 4000 battles the ally makes 69% of the replies and deals **86%** of
      all reply damage. That is the roster's asymmetry, not the trait's — and it
      caps how big the trait may be. Measured: power 40 alone 49.5%, **40+25
      (shipped) 51.9%**, 40+200 73.5%, 250+500 **98.3%**. Tune over thousands of
      seeds; the 40-seed sweep cannot see a move this size.
      ⚠️ **It flipped the sign of a figure already in the README.** Removing
      `razor_leaf`'s pierce used to cost the ally 2.7pp; it now gains 1.1pp
      (51.9 → 53.0), at every reply size tried. Piercing helps whoever attacks,
      and this is the first thing that charges for attacking — so **no balance
      figure measured before this feature carries forward**.
- [x] **A health threshold a *skill* can read.** `skill.Condition.BelowHealth`
      (permille) reads the **target**; `passive.Condition` reads its **holder**.
      `brine` moved onto it and is finally its canon move — 1000 power → 2000 at or
      below half health. ⚠️ **They share the arithmetic, not the type**:
      `scale.AtOrBelowShare` is the one comparison and both call it. One shared
      `Condition` could not say *whose* health it meant (and `passive` imports
      `skill`, so it was unwritable). A condition may read a status, health, or
      both — **both is AND**, so a clause narrows rather than widens. Refused, not
      defaulted: asks nothing; stacks with no status; share outside 1..1000;
      consumes a status it never names. The reading is a **`skill.Target`** struct
      (stacks, health, maximum) — not three params, because two int64 healths swap
      silently; `Amplified`/`PowerAgainst` take it, `Condition.Satisfying()` is the
      cheapest holding target for previews/reports. ⚠️ `battle.conditionTarget` is
      the **single** builder because `Suggest` and `resolveAgainst` must read
      identically, else the AI prefers a bonus it does not get. Still absent: the
      **gradient** (harder the further the *caster* fell) — a multiplier in
      `combat`, reads the other unit, a separate feature.
- [x] **Learnsets and slots.** A character now holds a **learnset** —
      `Character.Skills` is `[]cast.Unlock`, the *same type as the traits* — and a
      placement **chooses** from it: `SkillSlots = 4`, `TraitSlots = 1`, in
      `internal/seed/roster.go`. One `{id, at_level}` shape, one validator, one
      `UnlockedIDs`, and even one renderer (`forge.UnlockSummary` draws a kit and a
      trait list alike, `razor_leaf poison_powder@8`).
      ⚠️ **`skills`/`passives` on a reference entry changed meaning**: they used to
      be a restatement of the character sheet and were REFUSED as one; they are now
      the loadout. Refused instead: naming nothing (a slot is a decision — no
      default), naming more than the slots hold, naming one twice, naming what the
      level has not learned. A refusal lists what *was* available.
      ⚠️ **The kit is required and the trait slot is not**, and that asymmetry is
      deliberate: a unit with no skills cannot act, so an empty kit is never a
      choice; a unit with no trait is ordinary, so insisting would make "the plain
      version" unwritable. `required`/`optional` are named constants at the call
      site for exactly this.
      ⚠️ **An archetype's kit stays `[]string`** — a preset has no level to gate
      against, so it is a suggestion for authoring; `cast.Learn` turns it into a
      learnset when hexforge builds a character from it, all at level 1.
      **`battle.Roster` is unchanged** and still takes a resolved kit: a learnset
      settles before a battle exactly as an evolution does.
      **The log now carries the placement.** `battle.Log.Roster` holds the resolved
      roster and `--verify` rebuilds from it, because once a placement picks four
      of nine, re-running the embedded data would compare two different battles and
      call the difference corruption. It carries the RESOLVED form, which makes a
      log readable across a data edit. `Log.Replayable()` is false for a log
      written before this: it still renders (that reads events only) and refuses to
      verify, saying why. `battle.Roster` gained json tags so the log is snake case
      like every other file here.
      **Measured, 4000 seeds:** roster **49.5%** ally (was 51.9 before slots), 0
      stalls, longest 63, and only **2.2%** of turns idle — not the 30% the design
      note predicted, because that was a level-1 two-skill unit and the youngest
      shipped unit is level 8 with three. ⚠️ Pierce flipped sign *again*: 49.2→46.5
      before replies, 51.9→53.0 with them, **49.5→46.0** with slots. **No balance
      figure carries across a feature.**
- [x] **Choosing to evolve.** A level **allows** a form; the placement names
      which it fielded. `Line.Resolve(level)` → `Resolve(level, stage)`, plus
      `Line.Allowed(level)` and `progression.Furthest` (the empty string, named)
      for every caller that has no placement behind it — a browse screen showing
      a character at level 30 is describing, not fielding. Roster entry gained
      `"stage"`, optional, absent = furthest, so an older roster still says what
      it said.
      ⚠️ **A form ahead of the level is REFUSED, never clamped** — a clamp fields
      a different unit from the one written down. Two refusals, told apart: a name
      the line does not answer to is a typo; a name merely ahead of the level is a
      placement that has not grown into it.
      ⚠️ **`at_stage` was never built and is not wanted.** The second gate is
      `Unlock.Stages`, an **allowlist**: a threshold could only say "from this
      form on", which a level already says, so everything an early form knew a
      grown one knew too and giving up an evolution bought nothing. A list says
      `["Bulbasaur","Ivysaur"]` — a move Venusaur never gets. Same reason
      `skill.Restriction` is an allowlist. `at_stage: "Ivysaur"` is just
      `stages: ["Ivysaur","Venusaur"]`; two fields would be two vocabularies.
      Refused: a form the line lacks · one named twice · **every** form (that is
      what naming none means) · a stage list on a preset (no line) · a character
      that learns nothing its **first form at level 1** can use.
      `forge.UnlockSummary` prints `sleep_powder@12[Bulbasaur,Ivysaur]` — the mark
      shows at every level, unlike a level gate, because it never stops being true.
      ⚠️ **The shipped roster does not take the trade.** Every unit is fielded as
      its furthest form, balance is unchanged at **49.5%** and `replay.golden` did
      not move. None of the three characters is close enough for one kept skill to
      pay for a stage of stats — a cast-tuning question, not a mechanism one.
- [ ] **Looking a status up — the layer with the most reading and the least said.**
      `hexforge` lists origins, species, archetypes, passives and skills; **statuses
      are not one of them**, so `statuses.json` is the only way to learn what `mire`
      does. In battle it is worse: the log says *resisted mire* and nothing says it
      is a quarter off speed for two turns. A skill's description names the status
      and stops — *gây sa lầy, 70% khả năng* — which says something will happen, not
      what. Shape: `Lang.DescribeStatus` beside `Describe`/`DescribePassive`,
      **derived**, both languages, reached by `?` at the battle prompt (like
      `?N`/`?TAG`) and a screen in the tool (like `screenBlurb`). ⚠️ Print a **life,
      not a tick** — poison 500×3 against burn 800×2 is compared over its life, and
      `skills.golden` already has that column; same for a percent against its stack
      cap. ⚠️ **Permanent is "always", never "0 turns"** (`Snapshot.Permanent`).
      ⚠️ **Group by category** or a cleanse is unreadable — `rapid_spin` strips a
      `stat_debuff` and a `dot`, and nothing shows which statuses those are.
      ⚠️ The one that will be got wrong: **it describes the declared kind, not what
      a unit feels** — `virulence` amplifies and a resistance refuses, so the 500 in
      the book is not what lands. Say so, or a reader meets it when the log shows
      650.
- [ ] **Choosing to evolve.** `Line.StageAt` derives a stage from a level and
      there is no decision in it; a level should *allow* a stage and the placement
      name which it fielded, so `Resolve(level)` becomes `Resolve(level, stage)`
      with the chosen stage's threshold no higher than the level.
      ⚠️ **`at_stage` cannot be built before this** — while a stage is derived,
      `at_stage: "Ivysaur"` *is* `at_level: 16`, a second spelling of one fact.
      ⚠️ **Choosing not to evolve is strictly worse as designed**: stage curves only
      rise, so the choice is decorative unless an earlier stage can learn something
      the later one cannot — which is now one field away rather than a mechanism
      away, since the learnset exists.
      **No condition beyond a level**: items, friendship and battles-fought all
      need somewhere to persist, and there is no meta layer, no inventory, no save.
- [ ] **Grow the cast.** Three characters ship, one per element, and the seed
      roster is no longer a mirror — so the thing this item was blocking, a
      measurable balance figure, exists. What is left is content, under two
      constraints: an archetype's kit constrains a character's affinity
      (`skill.CanCarry` enforces it while authoring, `Archetype.Demands` reports
      it), and `progression.Limits` bounds health and defence **together** because
      those two multiply. Every character added moves `scenarios.golden` and
      `replay.golden`, which is the point rather than a cost: those diffs are how
      the balance change gets read. Read squirtle first — water is the strongest
      of the three elements and Blastoise still cannot carry an ace slot, because
      its attack and speed curves are the lowest in the cast.
