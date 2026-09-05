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

⚠️ **Three sections' worth of this file moved out on 2026-09-05, and nothing was
deleted.** This file is loaded in full at the start of every session; it was
396KB, about a hundred thousand tokens spent before any work began. Measured
first, because the obvious diagnosis was wrong: of 4,333 prose lines here, **six**
appear verbatim in `README.md` or `TODO.md` — the three documents do not repeat
each other at all, so there was nothing to *delete*, only somewhere better to
*put* things.

What stayed is what binds **any** edit, whatever it touches. What left is
subject matter — read it when you are in that subject:

| moved to | what is in it |
|---|---|
| `docs/architecture.md` | the runtime pieces: the two clients over one `internal/screen`, `internal/room` (a state machine with no I/O and no clock), `internal/socket` (the one boundary the WebSocket crosses), **the event log as the contract**, and how a battle ends |
| `docs/balance.md` | how the game is priced and tuned: `Suggest`'s rating, species and origins, the inert element, the strip/guard/grant/summon/amplify/detonate/`charge` categories, `hexforge weigh`, and `forge.Bout` |
| `TODO.md` and `docs/decisions.md` | the twenty-eight-item log that lived here: its three open items joined `TODO.md` § *Not done* and its twenty-five finished ones joined `docs/decisions.md`, which is where finished work and its reasoning live |

⚠️ **The one-line rules those sections rest on did NOT move**, and that is the
point of the split rather than a happy accident. *The layer rule* below still
states that a battle is a pure function of its seed and that a renderer can never
disagree with the engine — which is the whole of why the event log is a contract.
*Invariants worth knowing before editing* still holds what an edit may not break.
The moved files carry the reasoning and the measurements; the binding sentence
stays here.

**So: still read this file first.** Open `docs/architecture.md` before touching
`internal/room`, `internal/socket`, `internal/screen` or the event stream, and
`docs/balance.md` before authoring or re-pricing anything.

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

### What bubbletea v2 moved, and the five things it broke silently

The migration was made for one reason: **a terminal cannot deliver the Command
key over the classic escape sequences.** There is no encoding for it, and v1's
`tea.Key` carried only `Alt`, so ⌘S did not exist as far as a program was
concerned. The Kitty keyboard protocol does carry it, v2 parses that protocol,
and a Command key now arrives as `tea.ModSuper`. `internal/screen/savekey.go` is
the single declaration of which keystrokes save; all three forms ask `IsSaveKey`
rather than matching a string of their own. It moved there with the skill form,
and the origins form followed it — `cmd/hexforge-tui/savekey.go` is two one-line
forwarders now, because the character form has not moved and a copy on each side
of the package boundary would be the fourth spelling this exists to stop.

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
  ⚠️ **And `plainTerminal` gets to decide it, so an unset `TERM` is "dumb" only
  away from Windows.** `TERM` is terminfo's convention: cmd.exe, PowerShell and
  Windows Terminal set none at all, so reading its absence as a dumb terminal
  drew **every native Windows terminal in plain text**, with no cursor in any
  text field — while macOS and Linux always set it and never reached the branch.
  The rule is `colorprofile`'s own, copied rather than guessed at
  (`isDumb := (!ok && runtime.GOOS != "windows") || term == dumbTerm`; it reports
  TrueColor for a Windows 10 build 14931 or later, and for any `WT_SESSION`), so
  the palette and the thing that writes the escape codes agree. `plainScreen`
  takes its three inputs as parameters because `runtime.GOOS` cannot be faked and
  both answers have to be assertable from either sort of machine
  (`TestAnUnsetTermIsOnlyDumbAwayFromWindows`).
  ⚠️ **Nothing in the suite could see it**, which is the part worth keeping: every
  other test here sets `NO_COLOR` and returns at the line above, so the branch was
  unreached on the machine it was written on and unreachable in CI. A rule that
  differs by platform needs its inputs handed in, not read.
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
in `cmd/hexforge-tui` **or in `internal/screen`** (`TestNoScreenHoldsItsOwnWording`
greps its own AST — ⚠️ it reads `os.ReadDir(".")`, so it is **per package** and
there are two copies of it, one in each; a package that grows a screen and no
walker silently stops being held to the two-language rule, and the golden cannot
stand in for it, because a literal moved out of a package renders identically),
every
key is worded in both languages and no key is orphaned, and every wording
measures one cell per letter — write Vietnamese **composed**, or a combining mark
measures zero and every fixed-width column on that screen drifts. What is
deliberately *not* translated: ids of every kind, the six stat labels
`hp atk def spd acc ddg` (they are the `--hp` flag names and the data files' own
keys — see `forge.ShortStat`), and diagnostics from `internal/core`, which get a
lead-in in the reader's language in front of the parser's own English.

The minimum window is **120x24**. It was 72 while the client was English only and
80 once Vietnamese arrived — that runs 20–30% longer for the same sentence and the
busiest footers landed just past 72 — and it is 120 because a third of the client
had been trimmed flat against the old ceiling. Measured over every screen in both
languages at 200x60, on the widest line the sweep constrains: **34 of the 92
screen/language pairs sat at 76–79 cells** of the 79 there were, 29 more at 70–75.
What is pinned is almost entirely **footers**, which are catalog wording and
therefore cannot be given room any other way — the floor is the only lever for
that class, which is why widening the data cells in #173 and #175 moved the count
by one (35 → 34). The stated cost is that a terminal narrower than 120 does not
draw this front-end at all; `hexforge` needs no room and does everything it does.
`TestEveryWordingFitsTheMinimumWidth` renders every screen in both languages and
holds every line inside the floor, minus one column so a full-width line cannot
wrap.

**`frame` cuts every line to the window, and the cut says so.** It clips rather
than wraps because a wrapped row pushes every row under it down by one, which is
how the footer leaves the bottom of the screen — that part is settled and must
not be reopened. What changed is that the cut **marks**: it used to be
`lipgloss.MaxWidth`, which cuts safely and **silently**, so twenty-three of the
twenty-four sites rendering `m.lang.Error(...)` lost the tail of a sentence with
nothing saying they had, and a truncated explanation that does not say it is
truncated is worse than one that does. It is the horizontal twin of `frame`'s own
`Truncated` marker, which has said so since it was written.

- **One cutting rule for the package: `clip`, in `model.go` beside `pad` and
  `labelAt`.** It was the picker's private helper for one refusal row; `frame`
  calls it on every line of every screen now, so it moved for the reason
  `fieldValueRoom` lives there. `viewTooSmall` cuts through it too.
- ⚠️ **It has to be escape-aware and marking at once, and neither of the two
  tools it replaced could do both.** `MaxWidth` steps over an escape sequence and
  cannot mark. `clip`'s old body appended the mark but sliced `[]rune`, which on
  a styled line peels the terminating `\x1b[m` off the end one rune at a time —
  measured, not argued: a bold red ten-cell line cut to nine came back
  `"\x1b[1;31mabcdefgh…"`, right width, right letters, **no reset**, colour
  bleeding down the rest of the screen. Every caller then passed unstyled text so
  nothing showed it, and `frame`'s lines are styled, so wiring the old body into
  `frame` would have shipped the bleed on the first cut header. It is
  `ansi.Truncate` (`github.com/charmbracelet/x/ansi`, already a direct dependency)
  now, which re-closes what the cut left open.
- ⚠️ **The mark is added only when the line is genuinely longer than the room** —
  a line that exactly fills the window comes back byte for byte unchanged. That is
  the whole off-by-one risk of marking: an ellipsis on a line that fitted claims a
  tail that was never there and spends a cell of content to claim it.
  `TestALineThatExactlyFillsTheWindowIsNotMarked` crosses that boundary rather
  than approaching it (the same header at `w` and at `w-1`) and is the **only**
  test in the package that catches the mutation.
- **A marked line is exactly as wide as the unmarked cut would have been**, so
  `frame`'s row arithmetic is untouched — the mark is a cell *of* the window, not
  one past it. Asserted against `MaxWidth` itself over every room from 1 up, so
  what is held is that the widths did not move.
- ⚠️ **What still reaches that cut is all text, and that was measured before
  marking every line was chosen.** At the 120 floor, over every screen and state
  `everyScreen` registers, in both languages: the header naming the library
  directory (122 cells), the check screen's summary line, which also names it, and
  the form's archetype row — a preset id and its whole kit — at **128 (vi) / 131
  (en)**. At 160 and at 200, **nothing**. A path, a sentence, a list of ids.
- ⚠️ **No drawing can reach it**, which is why a blanket mark is safe: `tui.Board`
  is **19** cells wide against a floor of 120, `tui.Roster` likewise, and the
  preview's art is `usableWidth() - 2` **by construction**. An ellipsis on the end
  of a sentence says a tail was taken off; an ellipsis on the end of ten rows of
  hex art says something nobody can act on. `frame` is handed a joined string and
  cannot tell one from the other, so the claim is that the case never arises, and
  `TestNoDrawingIsEverWideEnoughToBeMarked` is what says so the day a wider
  drawing is added — widening the preview to the full window turns it red.

⚠️ **Every width figure quoted below against "79" or "of the 79 there are" was
measured at the old floor and is kept as the reading it was**, not restated: they
are records of why a wording was trimmed, and the trim is still in the catalog.
The live budget is 119. Raising the floor also **loosens** that sweep — every
existing line now passes trivially — which is the promise changing rather than a
test going vacuous; what it does not loosen is the vertical side, because prose
wraps at the floor and screens budget rows around the wrap.
`TestEveryFloorWrappedBlockTakesTheRowsItTakes` pins those row counts so the next
floor move cannot change one in silence.

`TestTheFormProducesTheCharacterTheCommandLineProduces` is what holds this: the
same answers as flags and as keystrokes must resolve to the same
`cast.Character`. A full-screen program cannot run with stdin as a pipe, so
`cmd/hexforge` is not going away — it is what a script uses, and the TUI refuses
to start when stdout is not a terminal rather than painting escape codes into
one.

**Where a form beats a prompt: the kit, and a skill's damage.** Two things the
full-screen client does that the command line cannot, and both are `internal/forge`
answers rather than screen logic:

- The kit is a **multi-select over the skill book** (`internal/screen/picker.go`),
  not a typed list. Every skill is listed, including the ones this character may
  not take, each marked and captioned with who it *is* for — a hidden skill reads
  as a skill that does not exist. The availability of a row is
  `forge.CheckSkill`'s answer, the same value the write refuses on, so the mark
  and the refusal cannot disagree. Nineteen rows do not fit beside a form (the
  form is nineteen body lines of the twenty it has in a 120x24 window), so the
  list is a **sub-screen that scrolls**, and `(*draw.PickState).Room` counts what the screen
  spends — including the empty string a trailing newline leaves when `frame`
  splits the body, which was miscounted first time and truncated the list.
- The new-skill form shows **expected damage as the power is typed**, from
  `forge.Library.PreviewDamage`, which is `combat.Rules.Damage` against the
  attack ceiling and *half* the defence ceiling. Those two are not a tasteful
  guess: they are the pair `skills.golden`'s own damage column is measured from
  (800 and 400), so the figure before a write is the figure the golden shows
  after one. It truncates **per strike** rather than once over the total, as a
  battle does — three strikes of 600 are 615, not 617. The amplified figure beside
  it is the skill with **everything it asks for holding**: the target's
  `requires`, the caster's `self_requires`, and `self_gradient` at the bottom of
  the caster's health, composed through `combat.Swung` in the order the battle
  composes them. ⚠️ It read only `requires` until then, which showed `outrage` and
  `comeback` — the two skills whose whole design is a caster-side term — at their
  plain power. **The row always draws its whole reading — the two figures and the
  reference pair they are measured against — and that is now a bound rather than
  a hope.** It used to drop the pair when the line would not fit
  (`Lang.DamageWithin`, with `damageRowRoom` computing the room), and PR #177's
  floor of 120 made that branch unreachable at every window the program draws:
  the line is four numbers in fixed wording, two of them the stat ceilings (three
  digits, always) and two `int64` at nineteen digits, so its **arithmetic**
  ceiling is 89 cells in Vietnamese and 87 in English against a narrowest room of
  97 and 98. Both were deleted rather than kept as dead weight, because each way
  the branch could have been reached again already has a stronger *build-time*
  guard: a **wording** that grew is caught by
  `TestEveryWordingFitsTheMinimumWidth`, which measures this row (it is program
  wording around figures, so none of the free-text exemptions reach it); **figures**
  that grew are capped by the type; and the **floor** going back down is caught by
  `TestTheDamageRowKeepsItsReferencePairAtEveryWindow`, which derives both the
  ceiling and the room rather than writing either down and names the arithmetic
  when it fails. ⚠️ Deleting `damageRowRoom` is also what fixed its off-by-one:
  it spent `width - 2 - labelWidth - 1`, so the row could fill the window's last
  cell — the one column every other row leaves empty because a line filling it
  wraps on some terminals. The surviving test measures the room off the rendered
  row against `minWidth - 1`, which is where that cell is now accounted for.
- The **squad builder** (`internal/screen/squads.go`) is the one screen that
  writes the author's own file rather than the game's: every other file here is
  data somebody wrote for the game, and `squads.json` is a side built to be
  fought with. It **ships like the rest of them** — the `go:embed` copy means a
  squad saved here reaches a battle at the next build, so nothing treats a squad
  in that file as a mistake. ⚠️ This used to read "writes something the game does
  not ship", and `seed.Squads` had a test asserting the file held none; both were
  written before anybody had built one, and the test failed the day somebody did.
  Three modes in one screen (`squadList`/`squadEdit`/`squadUnit`), because they
  are one decision at three depths and splitting them would put the half-built
  squad somewhere two screens could reach.
  - A member is **character, level, form, cell, four skills, one trait** — the
    same six facts a roster entry carries. Everything under the character is read
    against it, so changing the character **empties the kit**; the form chooser
    offers the empty stage plus every form by name, which is what a **forking**
    line needs; the slot chooser **steps over** an occupied cell rather than
    letting the save refuse it later.
    ⚠️ **The empty stage is two different things, and the screen says which.** On
    a line that does not fork it is the furthest form the level reaches, worded
    `SquadFurthest` — unchanged, in the field and in the member's row in the
    squad. On a line that **forks** it names nothing: the level has two ends,
    `StageAt` refuses to choose, and the placement is not fieldable at all —
    `placement.Squad.Take`, which is the one call `SaveSquad` and `FightSquads`
    both make, refuses the member. So it is worded `SquadForkUnnamed` instead, and
    a line under the fields (`SquadForkArms`, prose, wrapped at the floor because
    it carries authored stage names) names the arms, the key that picks one, and
    the two costs of not picking: the save, and the **silently shortened
    loadout lists**. `SquadsScreen.Form` hands `cast.SkillsAt`/`PassivesAt` an
    empty form in that state, which holds no gate, so both pickers offer only what
    every arm learns — measured on the one shipped fork at level 60: **13 skills
    and 4 traits unnamed, against 14 and 4 as Poliwrath and 15 and 5 as
    Politoed**. A list cannot say why a row is not on it, so the screen has to.
    ⚠️ **This chooser is the prior art the three read-only views were given a
    smaller copy of.** They describe a character rather than field one, so their
    question is narrower — which grown form does this level resolve to — and their
    arms come from `cast.Character.FurthestAt` rather than `StagesAt`: one form on
    a line that does not fork, one per arm on a line that does. The choice lives on
    `BrowseScreen.Form`, is settled on every read by `screen.ChosenForm` (the
    cursor and the level both move under a chosen name), rides to the two
    describers on `Subject.Stage`, and is walked with `s`. ⚠️ **A line that does
    not fork draws no row and answers no key**, which is what keeps every other
    character's record byte for byte what it was.
  - **A character can be held back: `cast.Character.Hidden`, and the squad
    builder is the only SCREEN that reads it.** `"hidden": true` in `cast.json`,
    absent meaning offered, so the flag is written only where it is set. A hidden
    character still ships, still loads, still fights, and a squad or a roster
    naming one is as valid as any other. Naruto is the one shipped example.
    ⚠️ **This said "the ONLY thing that reads it" and "an authoring convenience
    an author flips back, not a design statement" until 2026-09-05, and
    `internal/draft` made both false.** `draft.NewPool` gates the ban-and-pick
    pool on the flag, so a held-back character cannot be banned, cannot be picked
    and cannot be fielded in a drafted match at all — which is a rule of the game
    rather than a convenience. What survives untouched is the half about the
    **engine**: nothing in `internal/core` and nothing in `battle` reads it, and
    a replay has no use for who an author was choosing between.
    ⚠️ There are now **two** callers filtering it and they filter *different*
    rules — plain Hidden for a draft, Hidden-minus-a-`keep` for this screen — so
    a `cast.Book.Offered()` accessor was **considered and refused**, not left
    open. `offeredCharacters`' own comment carries the argument; do not go and
    write the accessor its earlier wording predicted.
    - **It round-trips or it is deleted.** `hexforge new` rewrites the whole file
      on every append, so the field is on the parse shape (`characterFile`) as
      well as on `Character`, exactly as `Skill.MarshalJSON` builds the parse
      shape rather than carrying tags of its own.
      `TestWrittenCastIsStableAndReloads` authors a held-back character and is
      what catches the omission.
    - ⚠️ **`squadScreen.characters` stays the WHOLE cast** and the filter is
      `offeredCharacters`, asked at the two sites that *choose* — `addUnit` and
      `cycle`. Filtering the held slice instead looks equivalent and is not:
      `character()` looks a member's character up in it to read the forms, the
      learnset and the traits, so a squad on the file naming a since-hidden
      character would lose its forms and its kit picker would refuse to open,
      with the row still printing the id.
    - ⚠️ **The one already chosen stays offered**, keyed on
      `squadScreen.unitOpenedAs` — what the member named when it was **opened**,
      not what is chosen right now. Hidden means *not offered for a new choice*,
      never *taken away from a choice already made*: a chooser that dropped the
      row would step off it on the first arrow press and write somebody else into
      a member nobody asked to change, in the author's own saved file. Keying it
      on the live answer is the near miss — the list then changes shape while it
      is being walked, so `right` then `left` lands one row short and the
      character is unreachable for the rest of the edit. That was found by the
      round-trip assertion and by nothing else.
    - The screen says why a character nothing else offers is on the list
      (`i18n.SquadHeldBack`), and the state is registered in `everyScreen` as
      `a held-back member` — which **asserts it draws that line**, because a
      registered state that renders nothing passes every sweep.
    - ⚠️ **`pickCharacters` is deliberately NOT filtered** (see the comment on
      `characterOptions`). It answers *which characters is this skill kept for*,
      and hiding a row there would make an existing `restrict.characters` naming
      that character unauthorable — the field is a picker and nothing else writes
      it. `TestASkillRestrictionMayStillNameAHeldBackCharacter` is what stops the
      job being "finished". The cast browser, the builds screen, `hexforge list`,
      the spar and the roster are unfiltered for the same reason: none of them is
      choosing a side to fight with.
    - `cast.golden` prints one line for a held-back character and nothing for the
      rest — the record has to be able to show that a character was taken out of
      the authoring lists, and "offered" on every other row is noise.
  - ⚠️ **A squad carries no side.** `placement.Squad.Take(side, cast)` fields it
    as either half and **prefixes the unit ids with the side**, so a squad fought
    against a copy of itself has two halves a log can tell apart.
  - `Library.SaveSquad` **replaces** the squad of the same id rather than
    refusing it, which is the opposite of `SaveSkill` and deliberate: a skill
    already in the book is something units carry, while a squad is a working
    document whose whole edit loop is saving it again. It validates through
    `Take`, so nothing is written that could not be fielded.
- The **fight** (`cmd/hexforge-tui/fight.go`, `forge.Library.FightSquads`) is
  raised from the squad catalogue with `f`, the way the spar is raised from the
  check: that is where a squad is under a cursor. Home is read off that cursor,
  the opponent is this screen's own chooser, and the runs are cached by
  `home|away|seeds` because a value receiver throws away a field written while
  drawing.
  - ⚠️ **Both ways round is the measurement, not a refinement.** Roster order
    decides the turn-queue tie-break, so one arrangement reports the *first
    slot's* advantage as the squad's — a mirror read **58.8%** the last time one
    was measured without swapping. Both halves run the **same** seeds.
  - **A squad against itself is a control**: exactly 500 per mille, by
    construction. `TestASquadAgainstACopyOfItselfIsExactlyEven` is what breaks
    first if the swap stops cancelling. The halves are reported apart too,
    because their difference is what *standing on a side* is worth — 18 points on
    the fixture pairing.
  - ⚠️ **The slot template cancels a synergy, and it looks like an answer.**
    `TestAMenderEarnsItsSlotWhereASparCannotSeeIt` holds the two fixed carriers
    constant across **both** squads, which is right for pricing a slot and wrong
    for pricing a pairing: whatever the partner is worth to the third member, it
    is worth to the opponent's third member too, so it nets to nought. Measured —
    a sapper written to compound with the blighter read **−29‰** in that shape,
    and the event log showed the synergy plainly working on both sides at once
    (poison three deep and 338 applications with the blighter present, two deep
    and 155 without). Two nearby designs fail the same way for their own reasons:
    a **striker** as the control measures what the squad's other two slots were
    short of, and **two different partners** measures the partners. What isolates
    it is one skill — the same character in both squads, `poison_powder` against
    `leech_seed`, both powders of no power — which read **636‰**.
    → `internal/seed/sapper_test.go`, which carries the three failures in its own
    comment so the next attempt does not rediscover them.
  - ⚠️ A squad rate is **not** the roster's win rate, and the screen says so in
    prose under the figure. That line is wrapped against **minWidth**, not the
    window in hand, which is the prose half of the width rule:
    **prose wraps at the floor, a data cell spends the window.** `minWidth` is
    the width this program promises to draw in, not a ceiling on what it may
    spend, so a gloss or a list of ids takes `m.usableWidth()` — cutting one on
    a wide terminal throws away content for nothing. A *sentence* measured
    against the real terminal would have two shapes instead of one, leave
    `TestEveryWordingFitsTheMinimumWidth` nothing to hold, and run a paragraph
    across a hundred columns for a reader to lose their place in.
    `cmd/hexforge-tui/width_rule_test.go` holds **both** directions: widening
    prose is as much a failure as clipping data.
  - ⚠️ **That line used to cite the art chooser as its precedent, and no longer
    can.** `artRoom` clips a filesystem **path**, which is data, and it takes
    the window as of #173. The precedent for a sentence is this line and the
    save note in `play.go`; what holds them at the floor is
    `TestAWideWindowStillWrapsProseAtTheFloor`. The old argument — that
    widening would leave the width sweep nothing to hold — was answering the
    wrong half of a row: the sweep measures the **catalog's wording**, and the
    catalog parts of the art row are exactly what `artRoom` *subtracts*.
- The **played battle** (`internal/screen/play.go`) is raised from the fight
  with `p`: the same pairing, one battle, the opponent played by
  `battle.Suggest`. `↑/↓` a skill, `enter` takes it and asks *where* only when
  there is more than one cell, `?` describes the one under the cursor, `a` hands
  the turn to the engine, `p` passes, `u` undoes, `n` is another seed, and
  `[/]` scroll the log (`pgup`/`pgdown` do the same and are what the brackets
  alias) — see *the budget* for why that pair and why following the tail is a
  state rather than an offset.
  - **Every option carries a one-line summary beside its id**, from
    `i18n.Lang.SummariseSkill`, and `?` raises the full description of the one
    under the cursor. An id is a name rather than an answer — nothing in
    `venoshock` says it is the skill that doubles into a poison — so the list was
    a column of things to guess between.
    ⚠️ **This costs zero rows, and that is the whole reason it is a line beside
    each option rather than a block under the list.** The screen's body is **28
    lines** at a 1v1 and `frame` gives it `m.height - 2` less the two the header
    takes, so at the declared 120x24 minimum only twenty survive. A pane under the
    list would be a pane nobody in the smallest window ever sees. What used to
    follow from that — the option list cut with the `Truncated` marker — is fixed;
    see *the budget* below.
    ⚠️ **An unavailable option keeps its `Reason` and drops its summary.** The row
    has one slot and the two answer different questions: why it cannot be cast is
    the live question the moment the cursor steps over it, and what it does is one
    keystroke away. Do not "fix" this by drawing both — the second would be the
    half that got clipped.
    The id column is **measured over `p.pending.Options`**, not over the book, for
    the reason `menuLabelWidth` and every detail pane measure theirs: the widest
    shipped id is thirteen cells and this unit may be bringing four short ones.
    The row **clips and never wraps** (`MaxWidth`, against `minWidth - 1` rather
    than the window in hand, like the fight's caution and the trait sentences), so
    the clause order matters — reach and cooldown are last because the end is what
    goes.
    ⚠️ **`SummariseSkill` is a fourth describer, and the reason is not brevity.**
    A compact line cannot be `Describe` with the flavour dropped: in Vietnamese
    `describeOpening` builds `BlurbFlavoured` out of the authored clause **and**
    the damage figure together, so there is no seam to cut. English separates
    cleanly, because English has no flavour and falls back to the derived opening
    — and a rule that works in one language and not the other is not the rule. So
    it is a distinct composition, held to the other reading by
    `TestTheOneLineSummaryQuotesNoFigureTheDescriptionDoesNot`: every digit run
    the compact line prints must appear in `Describe`'s output for the same skill
    in the same language, over every shipped skill. One way only — the compact
    line leaves out accuracy, pierce and a critical chance on purpose, and
    **counts** a strip rather than enumerating it — `purify`'s three categories
    are 79 cells in Vietnamese before the aim and the cooldown, so an enumeration
    could only ever arrive trimmed, and the claim it makes is read off
    `status.Category.Harmful` per skill rather than assumed (5 of the 8 categories
    are harmful, so a cleanse may be called one and a dispel may not). The
    wordings differ deliberately; the numbers may not. Both readings sit one under
    the other in `describe.golden`, so a balance change has to move both.
  - `?` raises **`screenBlurb`**, which is now branched on three ways — the skill
    listing, the cast browser and this — through the **same** `skillLines` the
    listing draws, so what a player reads while choosing is the paragraph an
    author reads while tuning. ⚠️ **Nothing in that path touches the battle**: the
    option is read, the skill is looked up in the library, and `esc` puts
    `m.screen` back rather than going through `model.enter`, which would rebuild
    the battle from its seed and throw the played half away. It works while
    **aiming** too (the skill is chosen and the cell is not, so the question is
    still open) and does nothing with no prompt or no options.
  - ⚠️ **The width sweep could not see any of this, and the fixture is why.**
    `everyScreen` built one battle and copied the model three times for its three
    states — but `PlayScreen` holds a `*battle.Battle`, so playing the "over"
    state out to its end **stepped the battle the other two pointed at**. By the
    time anything drew them `p.fight.Finished()` was true, `view` returned at the
    game-over branch, and `PlayFooter`, `PlayAimFooter`, the option rows and the
    whole aim block were rendered by **nothing in the suite**. Both footers were
    over the window the entire time — **82 cells (vi) and 83 (en)** against the 79
    there are — and every width test passed. Each state enters the screen for
    itself now, and both footers were retrimmed to name `?` and fit: **77 (vi) and
    78 (en)**, with the word after `esc` dropped, which is what `BrowseFooter`
    already does at the same squeeze. No key was given up.
    `TestTheBattleFootersNameTheDescriptionKeyAndFit` holds the half a width sweep
    cannot: that the key the feature hangs on is still named after the next trim.
    This is the **second** fixture in this repository whose early return made the
    interesting branch unreachable; the first was `plainTerminal`, where every
    test set `NO_COLOR` and returned above the branch. That is the transferable
    part: **a fixture that reaches a screen's early exit measures the exit**, and
    a screen it never renders is a screen with no width and no translation test at
    all.
  - It draws with `internal/tui` — `Board`, `Roster`, `Order`, `Line` — rather
    than with drawings of its own: what is played here has to look like what the
    game client plays, and a second drawing of a battle is a second thing that
    can disagree about what happened.
  - **The budget: this screen cannot fit the window the tool declares, so it
    decides for itself what to give up.** The fit was not a hard problem, it was
    an impossible one, and the measurement is worth keeping rather than
    re-taking. At 120x24 `PlayBodyRoom` leaves the body **twenty** rows:

    | section | rows |
    |---|---:|
    | heading | 1 |
    | `tui.Board` | **10**, fixed |
    | `tui.Roster` | **1 + one a unit** |
    | `tui.Order` | 1 |
    | the log | `PlayLogWanted`, then **every row nobody else claimed** |
    | the option list | 1 + one an option |

    | squad | roster | heading + board + roster + order + options | vs 20 |
    |---|---:|---:|---:|
    | 1v1 | 3 | **20** | exactly, with no blank and no log |
    | 3v3 | 7 | **24** | over by 4 |
    | 5v5 | 11 | **28** | over by 8 |

    `hex.MaxTeamSize` is 5, so **28 rows is the floor for a legal squad** before a
    single blank or log line, and a **summon** puts units on the board past the
    five a squad brought — up to the nine formation slots a side, which is
    `board + roster = 29` on its own. No arrangement of these sections fits.
    ⚠️ **So the defect was never the height, it was where the cut landed.**
    `frame` cuts from the **bottom** and the option list was the last thing the
    body wrote, so the one thing a player has to see in order to act was the
    first thing thrown away. `playFit` reserves the **heading and the turn in
    front** — never dropped, never cut, because a battle screen that cannot show
    the moves is not a battle screen — and hands what is left to the rest in a
    stated order: the **save's own note** (the answer to a keystroke pressed a
    moment ago, naming the file; *not* reserved, because a pair of notes runs to
    four rows or more and reserving them could crowd out the list), then **`Roster`** clipped
    a row at a time (health and effects are what a turn is decided on, and it is
    the one section that compresses by degrees), then **`Board`** dropped whole
    (ten rows of drawing have no half, and what it says is recoverable — the aim
    list prints the occupant beside every cell), then **`Order`**, then the
    **log**. Measured on the fixture with the log filled to its eight rows, and
    the heights are the same in both languages because every one of these
    sections is a *count* rather than a sentence: the whole screen survives from
    **h ≥ 36** at a 1v1, **40** at a 3v3 and **44** at a 5v5; the log is gone at
    **29 / 33 / 37**, the order line at **27 / 31 / 35**, the board at
    **25 / 29 / 33**, and the roster is clipped only once the aim list is up as
    well — from **30** at a 5v5, **24** at a 3v3, and never at a 1v1, whose whole
    roster is three rows.
    ⚠️ **What disappears is not monotone in the height, and that is the priority
    working.** The board takes ten rows or none, so at the height where it still
    just fits it takes the rows the order line and the log would have had, and one
    row shorter it cannot fit at all and both come back. Only the *offering* order
    is monotone.
    ⚠️ **The screen says what it gave up**, in one dim line under the heading
    (`i18n.PlayHidden` and the five names beside it) — a screen silently missing
    its board reads as a broken screen. A **shorter log frame is not in it**: the
    log is a frame over a history that is nearly always longer than it, so two rows
    fewer is the section working, while no rows at all is not. *How much* of the
    history is off screen is a different statement and it is the one on the heading
    row.
    ⚠️ **No per-screen floor was introduced, and the reason is that `minHeight`
    already is one.** `screenContent` returns `m.tooSmall()` before any screen is
    drawn below 120x24, so this screen is never asked for a shorter window than it
    can degrade into: at 24 the body has twenty rows, the heading and a
    four-option list reserve seven and the notice one, and the twelve left hold a
    5-a-side roster whole. A second floor would be a second answer to a question
    the first one has already refused — and it would have to be per-screen inside
    `tooSmall`, which is asked in `key` as well as in `View`. The one branch this
    puts out of reach is the save note being dropped, which needs h ≤ 15; it is
    constructed deliberately in `TestTheSaveNoteOutranksTheBoard` and the comment
    there carries the arithmetic.
    ⚠️ **`PlayLogWanted` is a floor of intent and used to be a ceiling** (and
    before that it counted events, which is a third thing again — `tui.Line` opens
    a turn with a blank row of its own, so one event arrives as two rows and eight
    events measured **eleven** a few turns in). The ceiling was a defect on its
    own: `playFit` hands the log the remainder of the budget and the remainder was
    then clamped to eight, so between a window 24 rows tall and one 80 rows tall the body
    grew **20 → 42** rows and the log stood still. Measured on the fixture, 3 a
    side, mid-battle: **8 rows at h=24, h=40 and h=80 alike**. A tall terminal
    bought the history nothing.
    The log now asks for `PlayLogWanted` **first** and then takes every row still
    unspent, which is why the same fixture reads **0 / 6 / 46** rows at those three
    heights (1v1: 5 / 12 / 52; 5v5: 1 / 8 / 48) — and why *nothing above it moved*:
    ⚠️ **growing the log may only ever spend rows nobody else claimed**, because
    #162's order is save note → roster → board → order line → log and the log is
    last precisely for being history rather than state. The two-part answer is also
    what keeps "everything fits" a question with an answer: a window that gives the
    log its eight rows has nothing missing, and one that gives it forty is that
    same window with room to spare. **Re-measured after the change and every drop
    height is unchanged**: the whole screen survives from **36 / 40 / 44**, the log
    is gone at **29 / 33 / 37**, the order line at **27 / 31 / 35**, the board at
    **25 / 29 / 33**, the roster is clipped (aiming) from **30 / 24 / never**. A
    single one of those moving would be a change to the priority rather than a side
    effect, and `TestTheBattleScreenDropsInTheOrderItStates` is what says so.
    ⚠️ **A floor in the *priority* is not available and was not built.** Eight
    guaranteed rows would have to come off the roster, the board or the order line,
    which is every one of those heights moving. The constant is what the section
    *asks* for, never what it is owed.
    ⚠️ **The log is a frame over the whole history now, and it scrolls in place.**
    `p.events` always held every event (`collect` appends and never trims) and the
    view threw the rest away: 283 rows rendered, eight drawn, **275 unreachable by
    any means** — no key, and nothing on the screen saying a history existed.
    `PlayScreen.LogRows` renders all of it and `logFrame` is the window; `pgup`
    and `pgdown` walk it, which is the pair that already scrolls the trait
    description and the picker rather than a second vocabulary for one idea (`↑/↓`
    walk the options and could not be taken). They work **while aiming** and **on a
    finished battle**, which is why all three footers name a scroll key at all — the
    keys they name are now `[/]`, see below.
    ⚠️ **Following the tail is a STATE, not an offset value, because the tail
    moves.** This is the decision the whole feature hangs off. A reader is normally
    at the newest rows; store that as the offset which happens to be newest and
    every event arriving silently shifts what is under them. So `logOffset` counts
    from the **start** of the history — which is what a `145–160 / 283` reading
    means — and `logFollow` is carried **beside** it. Same rule the abandoned queue
    tie-break paid for: `Queue.Pending` answers 0 for a unit it never heard of and 0
    is *soonest*, so absence had to be declared rather than detected. A sentinel
    offset would be that mistake again, and it would read as working, because the
    sentinel is a legal offset on the turn it is written —
    `TestFollowingTheTailIsNotAnOffset` is the only test that can tell the two
    apart, and it appends an event **without a turn behind it** on purpose, since
    every real turn resets the frame and would make the test pass for the wrong
    reason. Scrolling back down to the bottom *asks to follow again* (and puts the
    offset back to nought, which is also an ordinary offset — the top — so nought is
    exactly as unusable as a sentinel).
    ⚠️ **Acting resets it**, in `record`, which is the one place every turn goes
    through: the player's, the engine's, the pass and the "let it pick". Somebody
    who scrolled back and then acted would be reading a frame from before their own
    decision. Undo and another seed reset it through `begin`.
    ⚠️ **Undo makes the history shorter**, so the offset is clamped against the
    current total **wherever it is read** and not only where it is written — `undo`
    rebuilds the battle from a cut script, so `p.events` is rebuilt too and an
    offset kept across it points past the end.
    ⚠️ **The position goes on the heading row**, not on a row of its own:
    `trận đấu  seed 1` is about seventeen cells of the seventy-nine, and a row of
    its own would cost exactly what #162 spent a whole PR proving this screen has
    not got. It is shown **whenever rows are hidden**, not only while scrolled back
    — the discoverability half of the report is that nothing said a history existed,
    and a reader who cannot see that there are 283 rows will not look for the key.
    It says nothing when the log is not drawn at all, because the notice above
    already names it as a section the window is too short for.
    ⚠️ **The footer had to be trimmed to name the new key, and no key was given
    up** — that is what was asked for. Measured, not counted (a hand-count of a
    candidate came back four cells over twice): the battle footer was **77 (vi) /
    78 (en)** of the 79 there are, and `pgdn/pgup` needs twelve. So the words after
    `↑/↓`, `enter` and `?` are dropped — the three keys whose meaning the screen
    itself shows, which is the same judgement `BrowseFooter` and this footer's own
    `esc` already took — and the battle footer is **74 / 74**, the aim footer
    **72 / 77** and the over footer **65 / 63**, with the word for scrolling kept on
    the two that had room for it. `TestTheBattleFootersNameTheDescriptionKeyAndFit`
    covers all three and logs each width.
    ⚠️ **The footers name `[/]` now, and the page keys still work — this is an
    ALIAS advertised in place of what it aliases, not a rebinding.** `[` is back
    and `]` is forward, at **all three** sites that scroll (this log, the trait
    description, the picker's reading pane), because a site aliased alone is
    exactly the second vocabulary the paragraph above refuses. The reason is a
    keyboard rather than a preference: a compact board has no PgUp and no PgDn,
    reaching them through a modifier or a layer, so a footer naming them was
    unreachable advice and the whole log below the frame was as unreachable as it
    was before #169. **Advertising both pairs does not fit** — `pgdn/pgup` is nine
    cells against the brackets' three, and the English aim footer would come to
    **86** of the 79 there are — which is why the wording is a replacement. The
    five reworded keys all came in under budget, measured with `%s` substituted
    (vi / en): battle **73 / 75**, aim **66 / 71**, over **59 / 57**,
    `PickerReadingFooter` **58 / 57**, `BlurbMore` **25 / 28**. Four of them
    dropped the full six cells; the battle footer spent five of them back on the
    verb (`[/] cuộn` / `[/] scroll`), because a bare pair of brackets is the one
    place the wording would have said less than what it replaced.
    ⚠️ **A key alias is the shape that ships dead**, so it is pressed rather than
    read: `TestABracketScrollsWhereverAPageKeyDoes` tables the three sites and both
    directions and demands the bracket and the page key land in the same state
    **after asserting the page key moved it** — two no-ops satisfy an equality.
    And the fixture's own vacuity is the half an assertion cannot see: a `key`
    helper sending `KeyPgUp` under the name `"["` passes that table completely,
    which is why `TestABracketIsTheKeystrokeItLooksLike` reads `keyPresses`
    itself. `[` is safe to take because it reaches no text field — the picker
    enters its reading pane before the typed field and `numberKey` admits only
    digits, the browse blurb has no input, and `isSaveKey` is asked ahead of the
    battle screen's switch — while `form.go` and
    `internal/screen/origins.go`, which do have fields, never handled a page key
    and are untouched.
    ⚠️ **`internal/tui` did not change and did not need a row-limited `Roster`.**
    Clipping the roster's *rows* is this screen splitting a drawing it was given;
    reformatting it would be the other thing. The old
    `TestTheBattleScreenIsNoTallerThanItAlreadyWas` tripwire is gone, replaced by
    the bound that is now true: the option list survives every window the tool
    draws, the aim list with it, and `frame`'s `Truncated` marker never appears on
    this screen at all.
  - **Undo is a shorter script replayed**, not an unwinding: the script is cut at
    the player's last decision and the battle rebuilt from the seed. The engine's
    turns are recorded too, because a half that was not written down would replay
    as a different battle.
  - ⚠️ **The only screen holding something the model does not copy.** Every other
    screen is a value; a `*battle.Battle` is a pointer, so a mutation reaches
    every copy of the model. The battle is stepped in `update` and **never**
    touched in `view`, which is what stops a redraw playing a turn.
  - **`ctrl+s` writes a `battle.Log`** through `Library.SaveBattleLog` into
    `<data>/battles/<home>-vs-<away>-seed<n>.json` — the pairing and the seed
    identify a battle, so saving twice overwrites rather than accumulating.
    Saveable mid-battle: a half-played battle replays as exactly that half.
    ⚠️ **`--verify` re-runs against the embedded copy**, not the directory being
    edited, so a log written after an unbuilt edit will not verify — which is
    what `NoteBattleVerify` says, and why it is a note of its own rather than the
    generic rebuild line.
    ⚠️ File names are built from author-typed squad ids and are made safe here
    (`fileToken`), not by tightening what an id may be: what a file name may hold
    is `forge`'s problem, and a data rule change deserves its own reason.
    ⚠️ `replaceFile` now `MkdirAll`s the target's folder and puts the temp file
    **in that folder** — a rename across folders is not an atomic swap. The test
    that proved a failed write leaves the old file alone had to change its
    mechanism: a missing folder is created now, so it fails on one under a *file*.
  - `take`/`skip` are two methods rather than one taking a `Decision`, so a
    decision with a skill and no aim — which the engine refuses and nothing here
    should be able to build — cannot be written at all.
- ⚠️ **One loadout rule, and it had quietly become two.** "Which four of the nine
  may this unit bring" existed as `seed.chooseFrom` *and* `cast.chosenFor`, both
  unexported, both worded slightly differently — and the builder needed it a
  third time. It is **`cast.ChooseLoadout` / `cast.ChooseFrom`** now and all three
  call it; the subject is a worded noun phrase (`unit "x"`, `the build "x"`) so
  each caller still says what it is talking about. The builder shows the refusal
  **as the kit is chosen**, which can only match the write because it is the same
  call.
- Every field carries a **help line describing the focused one**, and a shape
  chooser draws the cells it covers on a sub-screen built from `pattern.Targets`
  and `hex.Render`, so the drawing cannot disagree with what the engine catches.
  Both replaced a static footnote that stated the parts-per-thousand convention
  and was not read. A field that needs a *list* — the statuses a skill inflicts,
  and each of the five allowlists — opens the same picker as the kit; each picker names
  its own hint, because the kit's (order, and what this character cannot take)
  says nothing true about an allowlist.
- A skill's **Vietnamese name is data, not Go**: `skill.Skill.Name` is opaque
  display text and `internal/core` never learns what a language is. An authored
  name wins, the compiled table in `internal/i18n/gloss.go` answers when there is
  none, and the bare id when there is neither. Any table showing it drops the
  column rather than drawing it empty.
  ⚠️ **This used to read "which is still the case for all nineteen shipped
  skills", and that has been false for a while.** Re-measured: **43** shipped
  skills, **43** carrying an authored name, and `skillGloss`'s nineteen ids
  intersect `skills.json` **not at all**. The table is now reached only by
  `internal/testfixture`, which is exactly why a test built on a fixture skill
  measures the wrong path — see § *The event log is the contract*.
- The skill listing **filters by name**, typed, live, on `/`
  (`internal/screen/skills.go`). Forty-three skills is a screen and a half at
  the declared floor, and the only way to a row was the arrow keys.
  - **It is a mode, and that follows from the keyboard rather than from taste.**
    Every letter this screen has is already a command — `q`, `a`, `e`, `k`, `j`,
    `?` — so a field sharing the keyboard with them could take no query at all.
    While the field has it, only `esc` (clear and close), `enter` (keep and hand
    the rows back), `backspace` and the two **arrows** are keys; the vim pair is
    text like every other letter, which is why the arrows are the arrows here.
  - **A row matches on its id or on its Vietnamese name, case- and
    diacritic-insensitively** — typing `diep` finds `phi diệp` — through
    `i18n.Fold` / `i18n.Matches` / `i18n.MatchesSkill`. That is the point of the
    feature rather than a refinement of it: on a terminal with no Vietnamese
    input method the name half is otherwise unreachable, so the filter would be
    an id filter with a misleading name.
    ⚠️ **The fold is an explicit table and `golang.org/x/text` stays indirect.**
    NFD-plus-strip-`Mn` needs a hand-written entry anyway, because **`đ` is not a
    `d` with a mark on it**. And unlike every gloss table here, this one has to be
    **complete**: a missing letter does not fall back to an id, it silently stops
    matching. `TestEveryLetterAShippedNameUsesCanBeFolded` walks the shipped books
    and both catalogs, so a name authored later in a letter the table lacks is a
    red test rather than a row nobody can reach.
    ⚠️ **The match reads the *data* and asks nothing about the language in
    front**, even though the English listing draws no name column: `ctrl+l` works
    from every screen and keeps everything typed, so a query that found different
    rows after it would be the screen mutating behind the author. The stated cost
    is that an English reader can be handed a row whose id does not hold what
    they typed.
  - ⚠️ **`s.cursor` indexes the FILTERED view**, so every read of it goes through
    `draw.SkillsScreen.Rows` / `Selected` — the funnel `draw.PickState.Visible` already is.
    `e`, `?`, the damage row under the listing and the description screen's own
    `↑/↓` all used to index `s.skills` with it, and **the two lists are identical
    while nothing is typed**, so a wrong read would have gone on passing the whole
    suite while an author edited the wrong skill.
    `TestEveryKeyThatReadsARowReadsTheFilteredOne` filters to a query whose first
    match is the *seventeenth* skill declared and asserts the fixture is that
    discriminating before it asserts anything else.
  - `a` is deliberately **not** guarded on there being a row: it indexes nothing,
    and a filter that found nothing is exactly when an author wants to write the
    skill they were looking for. `e` and `?` decline.
  - **The filter row costs a listing row, paid unconditionally**: `skillsRoom`
    reserves **ten** now rather than nine. That is the same rule the other two
    conditional lines there follow — reserve for the busiest state — and it buys
    something visible: pressing `/` narrows the list without also shifting every
    row under it up by one. The footer had to be trimmed to name the key and no
    key was given up; the words after `↑/↓`, `esc` and `q` are dropped, which is
    `BrowseFooter`'s own squeeze. Measured (vi / en): the listing footer **77 / 78
    → 65 / 74**, and the filter footer **63 / 70**.
  - The three states are registered in `everyScreen` (`filtering skills`,
    `filtered skills`, `skills filtered to none`) and are **driven with the keys
    an author would press**. A state added without an entry there has no width
    test, no translation test and no leak test, which is a mistake this
    repository has now made four times.

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

⚠️ **The fifth: pasting stopped working in both clients and nothing failed.**
v2 enables bracketed paste by default and delivers a paste as **`tea.PasteMsg`**,
not as a run of `tea.KeyPressMsg`. `bubbles`' `textinput` handles that message
correctly — but a model whose `Update` switches only on `tea.KeyPressMsg` drops it
before the field ever sees it, and `PasteMsg` appeared **nowhere** in this
repository until #274. Every text field in both clients ignored ⌘V, and the join
screen ignored the room code somebody had just copied.
⚠️ **`ctrl+v` was worse than unhandled: it read the clipboard and threw it away.**
`bubbles` binds `ctrl+v` to `textinput.Paste` by default, which shells out to
`pbpaste` — and then delivers `textinput.pasteMsg`, which is **unexported**, so a
model outside that package cannot name it and cannot forward it. The fix reads the
clipboard through `internal/clipboard` and returns a `tea.PasteMsg`, so the
terminal's own paste and the program's own key end at the same insert.

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

**Two goldens grew with the lobby, and both are named here so the next `make
golden` reader knows what moved.** `internal/screen/testdata/screens.golden`
gained **two** entries — `a live battle` and `a live battle waiting`, the two
states `draw.PlayScreen.Live` adds — and
`cmd/hexarena-tui/testdata/screens.golden` gained **nine**: three join-screen
states, the waiting screen, three live battle states and two result screens. The
only *existing* block that moved in either is that client's **menu**, which grew
its ninth row and, in English, widened its measured label column from eight cells
to eleven because "join a room" is the longest label there now.
`cmd/hexforge-tui`'s golden is **unchanged**, which is the check that the local
`PlayScreen` drawing did not move: that client draws it in local mode only, so
any diff there would have been the regression.

`skills.json` is the exception that is **balance and tool-written at once**:
`hexforge skills add` and the full-screen client's skill form both append to it,
and `hexforge skills edit` and the same form (opened with `e` on the listing)
both change what is already in it. That is why it is committed in the form
`Book.Marshal` writes, on the same terms
as `cast.json` below, and why a save says **the golden files have moved** rather
than only that it wrote — a power reaches `skills.golden` and `describe.golden`,
so `make golden` and reading the diff is the next step and not an afterthought.

⚠️ It does **not** reach `scenarios.golden` or `progression.golden`, and this
paragraph said it did until 2026-08-31. Each golden moves for what its generator
is handed and nothing else: `scenarioReport` takes the rules, the chart, the
modifier bounds, the ceilings and the pattern book, and reads the **status** book
for its poison ladder — no skill book, no cast. `progressionReport` takes the
limits and the rules alone. The four skills #182 added moved neither.

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
  on the real data rather than on a fixture.
  ⚠️ **This said "a field added to one struct is a compile error in the other
  until it is added there too, which is the point", and that is FALSE** —
  measured 2026-09-05, from step 3 of the draft, which was about to be designed
  around it. `Skill.file()` (`internal/core/skill/skill.go:1115`) is a **keyed**
  composite literal, so a field added to either side compiles fine and is
  silently dropped by the writer; only an *unkeyed* literal would give the
  compile error the sentence describes. What actually catches it is the round
  trip over real data named above, which is why that test is the load-bearing
  half of this bullet rather than the reassurance at the end of it. **Do not
  reach for this as the precedent for "two structs the compiler keeps in step"**
  — there is no such precedent in this repository; the shapes that must not
  drift are either one struct embedded in two (`wire.DraftDecision`) or held by
  a round trip.
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
`cast.resolveArchetype`'s three restriction rules the way `CheckSkill` brings
forward `resolveCharacter`'s five. Two other things an edit must keep: an
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

`cast.json` and `origins.json` are **meant** to be committed in exactly the form
`Book.Marshal` writes — two-space indented, sorted by id. The reason is that the
tool rewrites the whole file on every addition, so a committed form that has
drifted from the written one makes the next `hexforge new` produce a diff of the
entire file instead of one block. Marshal is also the one place in `cast` that
*imposes* an order rather than preserving the authored one; everything else keeps
declaration order, because a map range would randomise it.

⚠️ **Neither file is actually in that form today, and the test that was supposed
to say so cannot see it.** Measured 2026-08-31: `cast.json`'s characters are in
declaration order, not sorted — `naruto.naruto` sits fourth where Marshal would
put it first — and `origins.json` reads `pokemon` then `naruto`. So the next real
`hexforge new` **will** reshuffle both files, which is precisely the churn this
paragraph exists to prevent. `TestWrittenCastIsStableAndReloads` passes anyway
because it reads the file out of the **scratch** directory, which
`testfixture.Inject` has already rewritten through `SaveCharacter`/`Marshal` — so
it compares Marshal's output against Marshal's own output and would pass on any
committed file whatsoever. → `TODO.md`.

**`roster.json` is an instrument, not a scenario, and it has four contracts.**
It is 3v3 by character reference — ally Venusaur 60 / Wartortle 16 / Charmander 8
against enemy Blastoise 60 / Charmeleon 30 / Ivysaur 30 — and each of those is
load-bearing:

- **No unit on both sides.** It used to be the same character three times per
  side, and a mirror cannot measure anything: a change helps both squads by
  exactly as much, so the win rate moves only by noise. That is what stopped
  `razor_leaf`'s pierce being judged by anything but its damage table.
  `TestTheShippedRosterIsNotAMirror` compares the **resolved** units — name and
  stat line — because a species and a level resolve to those, and two units
  agreeing on them are the same unit however they were authored.
- **Every unit reaches past the enemy's front rank.** `battle.New` refuses
  nothing on reach any more — it cannot, because a range of one always finds
  whoever is foremost — so the roster is held to the stricter rule by
  `TestEveryShippedUnitCanReachEveryEnemy`: every unit carries at least one skill
  of depth two or more. A squad whose whole back half is decoration until the
  enemy's front line dies is playable, and it is not what this roster is for.
  ⚠️ **This contract used to be about cells and no longer is.** Slot `1,2` was
  once **four** cells from the enemy's own `1,2` — past every range in the cast —
  and a draft that used it stalled 5 seeds in 4000, not even as a draw: a
  survivor kept refreshing a regeneration, so something was always pending and
  `frozen` correctly never fired. Distance cannot strand anybody now; **depth**
  can, which is why the check moved onto the kit.
- **Both trait states and all three stages are in play.** Charmander at 8 is below
  `blaze`'s unlock level, Wartortle at 16 sits exactly on `endurance`'s, and Ivysaur
  at 30 has earned two traits and fields neither — so a battle exercises a unit with
  its trait, one that has not earned one, and one that declined. Since `blaze`
  became gated it carries a third state as well: Charmeleon holds it from the
  opening board and only comes *into* it partway down, which is what a shipped log
  now shows a `passive_held` for mid-battle.
- ⚠️ **The formation is what makes the figure, and it is placed to be screened.**
  Every unit stands where it does for a reason: each ace holds its side's **back**
  column at `0,1`, the two young units share the middle column at `1,0` and `1,1`,
  and the **front column is empty on both sides**.
  ⚠️ **Placement is purely defensive now** — a unit's reach does not depend on
  where it stands, only on what it can be aimed past — so "ace at the back" is the
  dominant placement, and the roster ships it on **both** sides rather than handing
  it to one. The shipped roster had the aces in **front**, authored for a board
  where reach was distance, and it read **27.6%** ally over 4000 seeds once ranks
  landed. Moving the two aces to the back column and changing nothing else — not a
  level, not a loadout — reads **47.3%**, and the 40-seed smoke test moved 12/40 to
  24/40 with it. Three separate things ride on that shape, and all three were
  measured:
  - **The ace is behind a screen.** Reach is counted in occupied ranks, so the
    pair in the middle is the first rank an attacker meets and the ace is the
    second — out of reach of every depth-one skill until the screen dies, which
    is the blocking rule doing the thing it exists for.
  - **The empty front column is deliberate.** An empty rank costs no range, so the
    ace sits at depth **two** rather than three and both sides can still be fought
    to a finish. The shipped board is therefore the standing demonstration of that
    rule as well as of blocking.
  - **The screen is adjacent.** `1,0` and `1,1` touch; splitting the pair to
    `1,0` and `1,2` reads **31.1%** against the adjacent pair's 47.3% over the
    same 4000 seeds, because an area shape that catches both of them is most of
    what the young units are for.

  `TestTheShippedFormationScreensItsAce` holds the first and the third of those,
  because a flattened formation is a balance change that reads as a tidy-up.
- ⚠️ **The levels are calibrated against how well the opponent plays, so an AI
  change invalidates them.** The two young enemies were 28 and 16 until `Suggest`
  learned to price statuses; the roster then read **80.0% ally over 20,000 seeds**
  and had to be re-levelled to **Charmeleon 30 / Ivysaur 30**, which reads 49.1%.
  ⚠️ **The ace level is not a dial** — Venusaur 60 → 50 alone takes the ally side
  from 79.0% to 4.0% at 4000 seeds. Tune the young units, and change one thing at a
  time: the loadouts were deliberately left alone in that pass so the level was the
  only thing measured.
  ⚠️ **They were left alone in the blocking pass too, and the room is narrower
  than it looks.** Charmeleon cannot go below **20** (`dragon_rage` is learned
  there) and Ivysaur cannot go below **20** either (two traits earned is the
  contract above), so the whole dial is 20..30 on each. Swept over that grid on
  the screened formation it spans **40% to 82%** ally, and the shipped 30/30 sits
  at the bottom of it — which is why the placement was the answer and the levels
  were left alone.

⚠️ **The 40-seed sweep in `TestSeedBattlesFinishFromEverySeed` is a smoke test, not
a measurement.** It read 45 per cent on a draft whose true rate over 4000 seeds
was 55. Tune levels against a few thousand seeds and quote that figure; the test's
job is only that the battle finishes and that neither side is a scripted defeat.

**`builds.json` is the late-game catalogue: which four skills and which trait a
character is *for*.** A learnset of nine skills and five traits offers more
combinations than anybody would field, and before the file existed the only kit
the repository could name was "the first four declared" — the order the file
happens to list, which is not a decision. `cast.ParseBuilds` checks every entry
against the cast book at `progression.LevelCap` on the furthest form, so a build
naming a move only an earlier form knows (`sleep_powder`) is refused with the list
of what that form does know.

- A build adds exactly two things over the loadout it names — a `name` and a
  one-clause `intent` — and **nothing numeric**: everything it does is already
  described by its skills and its trait, so a figure in either field is refused at
  parse time (the same rule skill `flavour` lives under).
- **A character listed there has at least two builds.** One build is not a build,
  it is that character's kit, and a screen offering a single option tells a player
  they have a decision they do not have. A character with none is the honest case —
  Naruto today — and `TestABuildIsACatalogueOfChoicesRatherThanOfKits` is the claim.
- **Three is the shape a mechanism with a middle needs.** Magnemite was the first
  character with three, and the third is what says its question has more than a yes
  and a no in it: `trickle` converts the counter as fast as it lays it down,
  `surge` ignores the mechanism entirely, and `hoard` waits and takes the pile.
  ⚠️ **`surge` is a direction rather than an absence** — the same character built
  as though the counter were not there — and
  `TestTheThreeMagnemiteBuildsAnswerTheCounterDifferently` asserts it spends
  **nothing**, because "three answers" is otherwise two answers and a duplicate.
  ⚠️ **None of the three wins a duel** (0 or 1 of sixty against Charizard, the
  heaviest attacker in the cast against the thinnest frame in it), exactly as the
  mender's two did not. What a duel prices here is what a build spends its turns
  on, and the ordering is what is held: blows, size of each, and stacks a
  discharge (drip **1.00**, hoard **3.85**).
- **Mew has three for the opposite reason**, and the pair is worth reading
  together. Magnemite's three are three answers to *one* question; Mew has no
  question of its own — no counter to spend, no element to lean on — so its three
  are three different characters, and what is held is that no two of them spend a
  turn on the same thing: `feed` is the only one that heals at all, `wither`
  inflicts several times the statuses, and `borrowed` deals the most damage and
  misses by far the most doing it. Four columns, three builds, each leading one.
  ⚠️ **`borrowed` carries nothing written for Mew** — every skill in it is already
  carried by somebody else — which is `surge`'s move made about a whole character
  rather than about a mechanism, and `TestOneMewBuildCarriesNothingOfItsOwn`
  counts rather than spot-checks, because a build that quietly picked one of Mew's
  own back up would still work and would no longer be saying anything.
- **`mewtwo.origin` is that build's mirror**, and the pair of them is why both are
  in the catalogue: one carries nothing of its own and the other carries nothing
  but the original's. It is also a *measurement* rather than only a theme — see
  the pierce section below for what putting one loadout on two bodies said.
- ⚠️ **The catalogue and the design tables in the tests must agree.**
  `poisonBuild`/`sustainBuild` (`bulbasaur_test.go`), `fireBuild`/`dragonBuild`
  (`dragon_test.go`) and `tankBuild`/`semiBuild` (`squirtle_test.go`) are hardcoded
  on purpose — they are what was measured — and
  `TestTheShippedBuildsAreTheOnesTheTestsMeasure` fails if the data drifts from
  them, kit **or** trait. Shipping a new build means measuring it in a test first,
  then adding the row.

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
- `cmd/hexforge-tui/testdata/screens.golden` is the **rendered client**: every
  entry `everyScreen` registers, in both languages, at the 120x24 floor and at
  160x60 — 200 renders, 8200 lines. It was the only golden in a `cmd` package
  until `cmd/hexarena-tui` grew one, and
  it exists because the screens had *property* tests (width, translation, no
  leaked wording) and no byte-level one, so a misplaced space or a moved clip
  point passed everything. ⚠️ **The header line is deliberately not recorded** —
  it names the data directory, so it names the machine; the fixture also hands
  `forge.Load` a **relative** directory, because the check screen's count line
  prints one in the body too. See the doc comment on
  `TestEveryScreenDrawsWhatTheGoldenHolds` for both, and for what a golden
  written today does and does not prove about the step before it.
- `internal/screen/testdata/screens.golden` is the **moved screens, in the
  package that owns them**: the six listings — `chart`, `elements`, `species`,
  `statuses`, `traits`, `builds` — plus the two states nothing shipped can draw
  (`unclaimed kind`, `traitless build`), the **description screen in both of
  its readings** (`skill blurb`, `trait blurb`), the **five states of the
  picker** (`kit picker`, `allowlist picker`, `filtered picker`, `status picker`,
  `reading a skill`) and the **skill listing with the seven states of it** the
  client's own sweep registers (`skills`, `add a skill`, `edit a skill`,
  `edited a skill`, `filtering skills`, `filtered skills`,
  `skills filtered to none`, `shape diagram`), the **squad builder** at each of
  its three depths with the two member states and the two pickers that go with
  them, the **works catalogue** with the `add a work` form over it plus the
  two states **neither** sweep could draw before it moved (`an empty works
  catalogue`, `a refused work` — `i18n.OriginsEmpty` and `i18n.AddRefused` each
  measured at **nought hits in both goldens** beforehand; the third gap,
  `i18n.OriginAdded`, stays open because it prints `Lib.OriginsPath()` and
  `noAbsolutePath` walks the recorded body), and the **played battle** in the
  six states of it that share no line (`a battle`, `aiming`, `a battle over`,
  `a scrolled battle log`, plus two more **neither** sweep could draw —
  `a saved battle`, whose note measured **nought hits in both goldens** and whose
  path is a *relative* value here, and `a battle with no pairing`, which the
  client's fight guards its `p` against) — in both languages at the 120x24
  floor and at 160x60 — **164 renders, 3851 lines**, body and footer recorded apart
  because a screen here answers with the two separately and every wording squeeze
  in this file is a footer. ⚠️ **It exists because the layout of code in
  `internal/screen` was held by a file in another package.** Measured after #205:
  widening the status category column by one cell
  (`Pad(row.Category.String(), column+1)` → `column+2`) left **every test in
  `internal/screen` green** and was caught by
  `cmd/hexforge-tui/testdata/screens.golden` alone. ⚠️ **And it now catches what
  the client cannot.** Measured after #222: widening the squad catalogue's id
  column by one cell leaves the **whole client suite green**, because
  `scratchData` deletes `squads.json` and so no test in `cmd/hexforge-tui` ever
  draws a catalogue with a row in it. The two goldens are not one net in two
  places; each sees a screen the other is blind to.
  ⚠️ **The skill listing's entries are driven with keys where the client's are**,
  and each hand-built state asserts it drew the line it exists for. The three
  filter states are what the query decides, so a field set by hand would record a
  test's idea of the filter; the reported edit is built as a `forge.SkillChange`
  value rather than written, because nothing in this package touches the data
  directory.
  ⚠️ **The picker is handed its list, so it has no one shape and its entries are
  a decision rather than a screen each.** The five are the paths through `View`
  that share no line with one another: rows carrying a refusal and a detail
  column, rows with a filter line over them, that filter narrowed, a field and
  its percentage under the list, and the reading pane, which replaces the list
  outright. They are **hand-built** where the client's five of the same name are
  raised through a form — two of the three screens that raise a picker are in
  `cmd/hexforge-tui` still, and the skill form's five are raised here now — so the
  two records are the drawing and the raising of it, which is this pair of
  goldens' whole arrangement.
  ⚠️ **The blurb gets two entries for three subject kinds.** A listed skill and a
  battle option are one `SubjectKind` — same id, same paragraph, same footer, only
  `At`/`Of` differ — so a third entry would record the same render twice;
  `NoSubject` is the arm a raise cannot reach, and a client's applier is what
  proves that. The **art preview** has an entry here as well now, and it is the
  one that records a *drawing* rather than a sentence.
  ⚠️ **Three more entries record a line that FORKS** — `a forked cast row`,
  `a forked art preview`, `a forked trait blurb`, over `pokemon.poliwag` at level
  46. Every other entry in all three records is a line that does not fork, which
  is what let a user meet `level 46 reaches [Poliwrath Politoed], which are
  alternatives: name the one being fielded` drawn in red where a picture should
  be. The three views fail differently and that is why there are three: the
  preview drew the refusal, the detail pane stopped at the row with it in, and the
  blurb drew **neither arm's traits and said nothing at all**, which is the shape a
  record is the only thing that can catch.
  ⚠️ **Neither golden is a subset of the other and neither may be dropped**, which
  was measured both ways rather than assumed. A trailing newline left on
  `SpeciesScreen.View`'s body reddens this one and is **absorbed by the frame's
  blank padding** in the client's; the client's `frame` budgeting one row fewer
  leaves this one green and takes the caveat line off the *statuses* screen in
  that one. What the package golden cannot see is everything the client composes —
  the header, the blank, the vertical cut and its `Truncated` marker, the
  horizontal clip. `screen.Ellipsis` reddens **both**, because the traits listing
  clips its own carrier row.
  ⚠️ Unlike the client's it drops **nothing** and needs no relative directory
  trick: no file in `internal/screen` calls `.Dir()` and `check` did not move, so
  the books load straight from `../seed/data`. `noAbsolutePath` asserts that
  anyway — a property that holds by construction is one a later change breaks
  quietly.
- `cmd/hexarena-tui/testdata/screens.golden` is the **game client's** framing of
  the same screens: every entry its own `everyScreen` registers — 24 over 12 of
  its 13 views — in both languages at the 120x24 floor and at 160x60,
  **96 renders, 3936 lines**. It is the third record of one set of screens and
  none of the three replaces another: `internal/screen`'s holds the drawing, the
  authoring tool's holds *its* framing of it, and this one holds **three things
  neither of the others can draw**.
  ⚠️ **The three read-only footers.** `i18n.SkillsReadFooter`,
  `i18n.OriginsReadFooter` and `i18n.SquadsReadFooter` are what a screen draws
  when `screen.Context.Authoring` is nought, and measured before this golden
  existed they came back at **nought hits in both** of the others, in both
  languages, because there was no client to draw them.
  ⚠️ **A squad catalogue with rows on it.** Both other fixtures delete
  `squads.json` — it is the author's own working document — and this one writes
  two sides into it, so the id column that #222 measured as invisible to the
  whole authoring suite is finally recorded somewhere.
  ⚠️ **A battle at three a side.** A one-a-side board, roster, order line and
  option list come to exactly the twenty rows the floor leaves, so nothing is
  ever dropped and the notice naming what the window was too short for is drawn
  by nothing; three a side is 24 rows against 20, which is where the budget
  starts deciding.
  ⚠️ It drops its header line and hands `forge.Load` a **relative** directory,
  for the reason the authoring tool's does: `frame` names the data directory and
  a saved battle's own note names the file it wrote, both of which would
  otherwise be a machine in a committed file. The drop is **asserted** rather
  than scrubbed, and `noAbsolutePath` walks every recorded line.
  ⚠️ The **art preview** is in every sweep and in all three goldens now. Its
  picture is exempt from the width sweep and nothing else about it is — see the
  `hexforge-tui` entry below for the arithmetic. Both clients also carry
  `a forked art preview` and `a forked trait blurb`; those two are the only
  entries in either client's record drawn from **shipped** art rather than the
  fixture's flat rectangle, so they inherit `internal/screen`'s same-machine
  caveat and are worth reading as a finding rather than a gate if they ever move
  on another architecture.
- `internal/wire/testdata/messages.golden` is the **PvP protocol**: one entry per
  message kind — the exact bytes `wire.Encode` produces, indented so a diff points
  at a field rather than at a four-hundred-character line — so a **wire change
  shows up in a diff**. A field renamed, a field dropped, an `omitempty` added or
  taken away, a kind or a code renamed, or the event digest's framing changed each
  moves a line here and nothing else in the suite shows a reader what moved.
  ⚠️ **Every body is a hand-written fixture and none of it comes from the shipped
  data**, which is the whole reason this file is safe to have. A `start` body
  carrying a real roster would move on every balance commit — a stat curve retuned,
  a skill's level moved, a character added — while measuring nothing about the
  protocol, and a golden that moves for reasons unrelated to what it measures is a
  merge-conflict generator. It is the same argument that gives `internal/seed`
  **no** golden on the data digest at all, and it is held rather than promised:
  `TestTheGoldenIsBuiltFromNothingShipped` refuses a record holding any shipped id
  prefix. The `turn` entry's digest **is** computed, off the fixture events, which
  is what pins `DigestEvents`' framing in the same diff.
  ⚠️ **The other side of that safety is a blind spot, and it was measured.** A
  hand-written fixture pins the **format** and can never see the **producer**:
  taking one line out of `gate.go` so the room stops filling in `TurnCap` on the
  `wire.Welcome` it builds leaves `internal/wire` — this golden included —
  **entirely green**, because the golden's welcome is a fixture's and not a
  room's. What catches it is `internal/room`, by name, in two tests
  (`TestTwoFakeClientsFightAWholeBo3InProcess` compares every welcome field
  against the room's own `Config`, and
  `TestTheTurnCapEndsABattleAsADrawTheOutcomeAlreadyHas` asserts each client
  stopped on the turn the room stopped on). So a field added to a message needs
  **two** things: an entry here saying it travels, and an assertion in the room
  saying it is filled in.

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
