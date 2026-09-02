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
go run ./cmd/hexforge statuses                   # the timed effects, grouped, and what each does
go run ./cmd/hexforge builds                     # the late-game catalogue: which four skills and which trait a character is for
go run ./cmd/hexforge passives                   # the declared traits and what each holds
go run ./cmd/hexforge check                      # parse the books from disk and verify the art exists
go run ./cmd/hexforge spar some.id --seeds 200   # duel it against the whole cast, both ways, report the rates

go run ./cmd/hexforge-tui                        # the same authoring, full screen (needs a terminal), in Vietnamese
go run ./cmd/hexforge-tui --lang en               # ...in English; HEXARENA_LANG=en does the same, ctrl+l toggles

go run ./cmd/hexarena-tui                        # play, full screen: the catalogues a reader wants, and a battle
go run ./cmd/hexarena-tui --lang en              # ...in English; same flag, same variable, same ctrl+l

go test ./...
go test ./cmd/hexarena-tui ./cmd/hexforge-tui ./internal/core/hex ./internal/i18n ./internal/screen ./internal/seed ./internal/tui -update   # accept new goldens
go test ./internal/core/battle -run TestControl                     # one test
gofmt -l . && go vet ./...
```

The `Makefile` wraps those and nothing more — `make build install run auto
play-tui forge forge-tui forge-tui-en test golden fmt vet check clean`. `make
build` builds all four binaries; `make forge ARGS="show some.id"` and `make
play-tui ARGS="--lang en"` pass arguments through. `make check` is the gate (`gofmt -l .`, `go vet ./...`,
`go test ./... -count=1`); `make golden` is the `-update` line above. The raw
commands stay listed here because they are what the targets are: reach for either.
There is no linter config — `gofmt` and `go vet` are the whole of it.

`-update` is only defined in the seven packages that hold golden files
(`cmd/hexarena-tui`, `cmd/hexforge-tui`, `internal/core/hex`, `internal/i18n`,
`internal/screen`, `internal/seed`, `internal/tui`), so `go test ./... -update`
fails on the rest. A new package with a golden has to be added to that command
**and** to the `golden` target. ⚠️ **This list has gone stale twice**, so it is
spelled in exactly three places — that command, the `golden` target, and the
paragraph under § *Data and golden files* — and a package added to one and not
the others is the next reader running `make golden` and not accepting a golden
they have moved.

## Rating an action: how Suggest prices what is not damage

`battle.Suggest` rates every option in one unit — damage — and everything that is
not damage is priced *in* damage, from the function that resolves it, over a capped
horizon (`price.go`). Four horizons: `buffHorizon` 3, `guardHorizon` 2,
`healHorizon` 2, `killHorizon` 1. Under-pricing costs a marginal cast; over-pricing
costs a kill.

Rules for anything added to that file:

- **Read the resolving function, never a second copy of its arithmetic.** A status
  tick comes from `inflict`'s own expression, origin and all; a restore from
  `combat.Rules.Restore`; a stat change from `modifier.Set.Stat` through
  `Battle.Stats`; a stack cap from the status book through `Set.With`. A price built
  from a second reading lets the opponent prefer a skill for something the skill
  does not do, and nothing reports the disagreement.
- **Weight a chance, never roll one.** `Suggest` may not touch `b.source`. Compose
  `amplify` (the *actor*) with `resist` (the *target*) and clamp last, exactly as
  `inflict` does. ⚠️ Those two take the same Go type: swapping them compiles,
  changes every price, and only `TestPricingAStatusUsesTheChanceThatWouldBeRolled`
  says so.
- **Worth nothing means not rated, never rated at nought.** A rating of nought is
  still a rating and would beat "found nothing", taking the turn ahead of whatever
  the fallback would have chosen.
- ⚠️ **Worth *less* than nothing is not the same as worth nothing, and may not be
  the fallback.** `rate` subtracts friendly fire and the recoil a skill puts on its
  own caster, so a negative total is reachable and means something exact: taking
  this turn leaves the board worse than not taking it. The two used to share a
  bucket — an option was skipped for scoring `<= 0` and then picked straight back
  up by the fallback — so the opponent cast skills it had just priced as a loss.
  A loss is now dropped from the fallback, and a unit with nothing but losses
  declines; `Suggest` returning false is the existing route to `Pass`, which every
  caller already drives. An option worth exactly nought is still a fine way to
  spend a turn nobody else wants.
- **Clamp every term against what the board can actually deliver.** A
  damage-over-time at the target's remaining health, a heal at the room *and* at
  what an enemy could take off, a charge at the strikes there are to eat. The
  unclamped versions are not conservative — they are the largest numbers in the
  file, and one of them was worth 19 points of the shipped roster's win rate.
- ⚠️ **A permanent status carries zero duration.** `min(duration, horizon)` prices
  it at nothing. Go through `turnsOf`.
- ⚠️ **`expected` reads the occupant of a cell**, so handing it a hypothetical unit
  silently gets the real one back. `Battle.against` takes the unit; use it.
- ⚠️ **Build a hypothetical with `status.Set.With`, never by copying a unit and
  applying to it.** A `Set`'s entries and each entry's stacks are slices, and
  `Apply` writes through them — a shallow copy refreshes the real unit's durations
  from inside the rating, and a refreshed status is indistinguishable from sustained
  pressure in every golden.
- **Tempo is priced from the stat, never the queue.** A wait is `atb.Scale / speed`,
  so a share on the stat is that share of the unit's turns; `tempo` reads that and
  asks nothing about who acts next. ⚠️ A turn is worth `turnWorth` — the **mean** over
  what a unit could point at somebody — and **not** `bestStrike`: charged at the best
  attack, `outrage`'s recoil made the dragon build avoid its own heaviest skill and
  its duel rate fell 26.6% → 20.0%, a rating playing worse while believing it had
  improved. The same figure prices both directions, so a haste and a slow cost the
  same.
- **An all-sided skill is rated by both halves.** `expected` gives the enemy half
  (it skips a unit on the caster's own side) and `friendlyFire` subtracts the own
  half — damage plus the turns lost with anything it kills, the caster **included**,
  because a shape can cover the cell it is cast from. ⚠️ The old refusal was not an
  oversight: the guard and "expected does not subtract an ally" were two halves of
  one decision, so relaxing either alone is the opponent that bombs its own squad.
  An all-sided attack also counts as an attack in `bestStrike`, `turnWorth`,
  `bestAgainst` and `worstStrikes`, through `aimedAtAnEnemy` so the four cannot
  disagree.
- **A tie is broken by what an option costs to have spent**, never by kit order:
  `take` compares value first and `declared.Cooldown` only on a tie. That is all
  *holding a skill for a later turn* honestly comes to here — damage is clamped at
  the target's health, so a nuke and a filler are worth the same against a sliver
  and the nuke was being burnt on it. ⚠️ A **discount** rather than a tie-break
  would price scarcity by guessing at turns, which is the mistake tempo was
  corrected for. Measure a change like this **head to head** (the tie-break on one
  side at a time): both sides use `Suggest`, so the roster rate shows which squad's
  kit gained, not whether the rating improved.
  ⚠️ **The FALLBACK obeys the same tie-break**, and did not until 2026-09-02.
  `Suggest` keeps one option worth nought in case nothing at all is worth doing,
  and that arm kept "the first such skill in kit order" long after `take` stopped
  doing so — so kit `[scour, wipe]` cast the three-turn cleanse and `[wipe, scour]`
  cast the free one, with kit order the whole of the decision. Options worth
  *nothing* are the sharpest case of the rule rather than an exception to it: both
  buy nought, so what casting one costs is all there is to separate them.
  ⚠️ Whether such a turn should be **passed** instead is a different and larger
  question — every gap in what `price.go` sees is an *under*-price, so "worth
  nought to the rating" is not yet "worth nought". → `TODO.md`.
- ⚠️ **A repeating count is read as its EXPECTATION, never as either end.**
  `Rules.Expected` goes through `Hit.ExpectedStrikes` (parts per thousand, so the
  division lives in `Expected` and not at the call site) and `hitAgainst` carries
  `Repeat` and `MaxStrikes` onto the `Hit`. Both halves or neither: with the
  fields dropped, `ExpectedStrikes` answers the plain count and fixing `Expected`
  alone moves nothing. `pricing.worstStrikes` reads the same figure and returns it
  in per mille — a charge cancels ONE strike, so it is worth *less* against an
  attacker that keeps going, and a whole number cannot carry a count of 3,120
  without inflating every guard priced against it.
  ⚠️ **`Rules.Total` deliberately stays on `StrikeCount`.** It is the
  deterministic figure `skills.golden`'s damage column is written from, and a
  number that moved with an expected value would stop being what an author
  compares two skills by. The two answer different questions.
  Measured: a Magnemite kit built on `spark` read **3.6%** with the floor and
  **25.0%** with the expectation — the rating would not cast its own best skill.
- **Health a skill gives back is priced as healing, from both sources.**
  `pricing.drained` reads `drainShare(declared.Drains + lifesteal(actor))` — the
  expression `resolveAgainst` pays it with, sum first and then bound — over the
  damage `expected` says will land, and runs it through `worthHealing`. Reading
  only the skill's own field prices `leech_seed` and leaves `blood_thirst` at
  nought, which is half a fix; skipping `worthHealing` makes it a flat bonus
  wearing a health check.
- **A blow is discounted by an absorbing POOL on its target, and an unblockable
  one is not.** `Battle.pastAPool`, read in `against` so every damage figure in
  the file inherits it. What a pool takes over a volley is the smaller of the pool
  and the damage, which is what `combat.Absorb` comes to across one — not a second
  copy of it. ⚠️ **A wall of block CHARGES is deliberately absent**: it was
  written and measured, it is a large gain where walls are dense (990‰ against the
  ruler, against a blind rating that cannot finish 212 of 800 battles) and reads
  as **nothing** on the ordinary squad boards (668‰ either way) while moving squad
  rates by 180‰ and breaking three balance claims, one of which reverses. Bisected
  to the charge clause alone. Do not add it without re-deriving those claims — and
  read the note in TODO.md about `ArcPower` being unpriced on the discharge first.
- **A discharge is priced on the turn that fires it, not only on the turn that
  charges.** `pricing.discharged` walks `chainFrom` — the same function the
  resolution walks, so the aim gates it identically — and prices each carrier at
  the resolving `Rules.Damage` expression with `ArcPower * Takes(held)`, over the
  expected strikes weighted by the chance to connect (an arc rides a strike that
  did not MISS, so a blocked one still fires) and capped at the rounds the carrier
  has stacks for. ⚠️ Before this, `ArcPower` appeared in one place in the file —
  `spendable`, which decides whether *laying a charge down* is worth a turn — so a
  conduit's whole payload was a free rider on the skill's own power. The aimed
  carrier's arc is clamped at what the blow itself leaves of it, or a conduit
  aimed at a sliver is worth that health twice.
- **Every status category has an arm.** `granted` takes the ones a holder wants
  — Regen, Shield, Absorb, Buff, Reserve, **Taunt** — and `inflictedOn` the ones
  it does not — Dot, Control, StatDebuff, Charge, **HealCut**. ⚠️ A taunt is a
  `granted` case even though `Category.Harmful` says otherwise, because
  `tauntStatus` **sits on the unit doing the taunting**: pricing it as harm bills
  its own caster its best strike three times over for casting it. Worth the aim it
  takes off every enemy at once, and nought against an enemy the holder was
  already the best target of — which is why it reads as exactly nothing in a duel
  and has to be measured in a squad. A heal cut is priced through `healingFor`,
  the expression a heal is paid with, over `Set.PendingIn(Regen)` — the
  regeneration visibly owed, deliberately less than the truth, because every other
  source of healing needs a lookahead this file does not have.
- ⚠️ **The queue may break a tie; it may never set a price.** A queue reading is
  **compared**, never added or multiplied. A value that reaches an arithmetic
  expression is tempo, and tempo is priced from the **speed stat** — see the tempo
  bullet above. The reason is not taste: the queue is discrete and lumpy, so a few
  points of speed buy whether one more turn lands before the other unit acts, and a
  mirror-duel win rate could not even *order* the shipped speed amounts (`+150` read
  59.0% while `+50` read 74.0%; see `swiftness` under § *Open work*). A number that
  lumpy may decide which of two equals to take. It may not say what either is worth.
  The rule stands even though nothing currently uses it, because it is the line
  between a tie-break and a second tempo.
  ⚠️ **A third key under `cooldown` was built, measured and thrown away.** It took
  the aim whose occupant acted soonest, off the non-mutating `atb.Queue.Pending`.
  Head to head against the rating without it (`forge.Bout`, 10,000 seeds, control
  exactly 500‰): **500‰ exactly, 10,000 wins and 10,000 losses.** Not "inside the
  band" — the *control signature*, meaning the two ratings played the identical
  battle every time. It moved **0 of 93,320 decisions** over 2,000 shipped battles,
  and a census found **0 prompts** where two options came out level on both value
  and cooldown. **No golden moved.**
  The premise was wrong rather than the code. "One skill pointed at several cells
  has the same cooldown on every call, so the winner is whichever cell `hex.Cells`
  lists first" is true, but a tie needs the **values** level too, and shipped units
  differ in health, defence and affinity, so two aims almost never rate to the same
  integer. Reaching the key at all needed a fixture tying two enemies at identical
  health on purpose. **Do not rebuild it without first showing the tie exists on the
  board in hand.**
  ⚠️ Two things learned building it are worth more than the key was. **Absence must
  be carried beside a queue reading, never encoded into it** — `Queue.Pending`
  answers 0 for a unit it has never heard of and 0 is *soonest*, so an aim with
  nobody to read would have outranked every aim with somebody; a self-aim has to be
  *declared* unread rather than detected, since the cell is occupied by the actor
  and a fast unit would otherwise have every self-aimed skill outrank every attack.
  And **`Suggest` may not call `Standings`**: it orders the real slice in place, so
  a rating that called it would move nothing on the board — `describeBoard` could
  not see it — while changing the order units act in afterwards, arriving in every
  golden at once with nothing naming it. Read `Pending` or `Preview`.
- **Waiting is arithmetically empty in this engine**, and is therefore *decided
  against* rather than unbuilt. `spendCooldowns` brings **every** cooldown down by
  the turn just served, and it runs at the end of `Act` (`turn.go:541`), at the end
  of `Pass` (`:323`) and on a turn control took (`:92`); `Act` then starts a
  cooldown on the one skill it cast and on nothing else. So the skill being "waited
  for" comes back on the same turn whichever the unit does, and across its next two
  turns *act now* is worth `bestValue` plus next turn's best while *wait now* is
  worth nought plus that same next turn's best. **Acting dominates by exactly
  `bestValue`**, and an option priced below nought is already declined — so there
  is no residue for a waiting rule to collect.
  `TestAPassBuysNoCooldownAnActDoesNot` holds the premise (a pass buys no cooldown
  an act does not) and `TestNothingWaitsOnPurpose` holds the consequence (a skipped
  turn carries one of three forced reasons and never a fourth).
  ⚠️ **Two further bars, if it is ever re-raised.** A *simulating* lookahead has to
  clone a `*Battle` — an unexported queue, `[]*Unit`, a `status.Set` whose entries
  are slices, and the `*rng.Source` — and then resolve into the clone. That is
  either `resolveAgainst`, which **rolls**, breaking *weight a chance, never roll
  one*; or a weighted twin of it, breaking *read the resolving function, never a
  second copy of its arithmetic*. Two available implementations, one broken rule
  each. And it costs roughly **×36 a turn** on the shipped kits, taking a
  20,000-battle sweep from ~7.5s to ~4.5 minutes.
- **An attack is charged for the reply it provokes.** `friendlyFire` is what a
  skill costs its own side; `replied` is what the units it hurts cost it back, and
  the two are subtracted side by side in `rate`. Before it, `price.go` did not
  mention `passive.Replies` at all, so attacking a `venom_blood` or a `thorns`
  holder was free — the same blind spot `friendlyFire` exists to close, pointed
  the other way. It reads `Battle.answer` rather than a second copy of it, and
  every decision that function makes is a decision here: **once per cast, never
  per strike** (a reply answers a *use*, so a three-strike skill pays one answer,
  and charging three declines it outright); **every unit the shape actually
  damages, whichever side they are on**, because `answer` never asks; not the
  caster itself; not a holder the blow kills; not a holder whose gate is shut. The
  damage is charged at face value clamped at what the caster has left and over
  **no horizon at all** — a reply is health taken off *now* — while the statuses
  go through `inflictedOn`, which already prices an inflicted status over the
  horizons above, rather than through a second copy of them. Both halves are
  weighted by the chance the attack connects, read off the one `combat.Hit`
  `Battle.hitAgainst` builds, which is the same Hit `against` weights the damage
  with.
  ⚠️ **A dead attacker is priced, and both halves of it.** A reply may kill —
  `Battle.reply` gives its damage no exemption for arriving out of turn — so the
  caster's health is spent down the walk the way `reply` spends it, a lethal
  answer is charged the turns the caster will now never take at `killHorizon`
  (exactly as `friendlyFire` charges an ally it kills), and **nothing after it is
  charged at all**, because `reply` returns before its own statuses land and
  `answer` returns before the next holder answers. Backwards in one direction the
  opponent is suicidal, in the other it is paralysed.
  ⚠️ **The charge is clamped at nought per answer, and that is the sign guard.**
  `rate` *subtracts* it, so a negative charge would make an attack **more**
  attractive for being answered — and "the opponent hunts the `venom_blood`
  holder" reads as a plausible strategy rather than as a sign pointing the wrong
  way. `TestAReplyRepelsRatherThanAttracts` asserts the direction from both sides
  of the board, because one side alone cannot tell a cost from a tie.
  ⚠️ **The gate is read on the holder as the board stands**, while `answer` reads
  it on the holder as the blow left it. That can only err one way, and it is a
  property of the type rather than a hope: `passive.Condition` carries nothing but
  `BelowHealth`, so a gate can turn *on* as its holder is hurt and can never turn
  off — this therefore misses a trait the attack itself wakes up and over-charges
  for none, which is the direction every cap in this file errs in, and it costs no
  hypothetical unit to be exact about.
  **Measured, and it is a wash — reported rather than papered over.** Head to head
  against the same rating with the term added back (`forge.Bout`, 10,000 seeds,
  control exactly 500‰): **499‰, band ±8‰, median 48 turns**. Inside the band, so
  not a finding. Against the frozen ruler it reads **779‰ (band ±8‰, median 45)**
  where the term-less rating reads **780‰** on the same seeds — the standing
  figure, reproduced exactly, which is what says the comparison is honest. The
  reason is the **data** and not the term: the shipped roster's only answering
  trait is `venom_blood` on `ally.venusaur`, a poison at a **4%** chance, so the
  charge is a rounding error and the term changes a decision in **22 of 500**
  shipped battles. Given every unit a shipped reply instead, all three are still
  a wash: `thorns` 497‰, `ballast` 499‰, `venom_blood` on all six 500‰ exactly —
  not one decision moved, because a charge both aims carry cancels.
  **And the term does win, on a board where a reply is worth something.** The same
  head to head with every unit answering off its defence: at 400‰ **503‰** (median
  38 turns against 48), at 800‰ **513‰ — clear of the ±8‰ band**, median 34, and
  at 200‰ *plus a certain poison* **607‰**, median 34. So the null on the shipped
  roster is a reading of the **data** and not of the term: the shipped replies are 80‰ and 50‰ of defence and a 4% poison, which is
  too small to change a decision worth changing. It is kept for the reason
  `friendlyFire` was — a rating that cannot see a cost prefers the option carrying
  it — with the honest note that it buys nothing today and starts paying the
  moment a reply worth authoring is authored.
- **Still out of scope:** *where* an extra turn falls in the order, as a **term**.
  As a tie-break it was built and thrown away for measuring a null (above); as a
  term it is what the rule forbids — a queue reading that priced how many turns
  something buys is tempo, and tempo is priced from the speed stat. The detonate
  setup needs neither: price the status and the payoff rates itself.

⚠️ **No balance figure carries across this change.** The shipped roster read 53.1%
ally before and 79.0% after, side-neutral (82.5% for the same squad with the sides
exchanged), 0 stalls, 4000 seeds. That is a cast finding — the roster's calibration
rested on the opponent not playing statuses. It has been re-levelled since, as a
separate data change: **Charmeleon 30 / Ivysaur 30, reading 49.1% over 20,000
seeds**. Any rate quoted anywhere in this file predating that pass is a reading of a
different instrument.

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
    offers *furthest* plus every form by name, which is what a **forking** line
    needs; the slot chooser **steps over** an occupied cell rather than letting
    the save refuse it later.
  - **A character can be held back: `cast.Character.Hidden`, and the squad
    builder is the ONLY thing that reads it.** `"hidden": true` in `cast.json`,
    absent meaning offered, so the flag is written only where it is set. It is an
    authoring convenience an author flips back, not a design statement — a hidden
    character still ships, still loads, still fights, and a squad or a roster
    naming one is as valid as any other. Naruto is the one shipped example.
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

## Two full-screen clients over one `internal/screen`

`cmd/hexforge-tui` authors the cast and **`cmd/hexarena-tui`** plays the game,
and they draw the same screens out of the same package. That is what the eleven
extraction steps were for, and it is the same shape the authoring pair already
had one layer down — a CLI for pipes, a `-tui` for the screen, one `internal/`
package under both, so the two cannot disagree.

⚠️ **`cmd/hexarena` is not that client and must not become one.** It is the
**verification contract**: `--replay --verify` re-runs a log from its seed and
checks every event, and `--auto` / `--log` are the scriptable half. Nothing about
it changed, and nothing about it may.

**What the game client offers**: a menu, the seven catalogues a reader wants
(characters, skills, elements, traits, species, works, squads) and a battle, plus
three screens reached by a keystroke rather than by the menu — the affinity chart
(`g` on the elements listing), the statuses reference (`?` on the traits listing)
and the description screen (`?` from three places). The build catalogue and the
art preview are the two `internal/screen` owns that it does not reach; both are
decisions filed in `TODO.md` rather than wiring nobody got to.

### One capability, because three shared screens author and one client cannot

`skills` has `a` and `e`, `origins` has `a`, and the squad catalogue's `n`,
`enter` and `d` reach the two depths that write `squads.json` and the deletion
that removes a side. A game client offers none of that.

`screen.Context.Authoring` is the whole answer, consulted **beside** the
keystroke it turns off so the keys and the footer are decided in one place.
`Context.Footer(authoring, reading)` is the footer half, and a read-only footer
is a **second wording** rather than the authoring one with a clause deleted by a
program: dropping `a thêm` out of a rendered line would leave the separators
either side of it, and nothing would measure what was left.

⚠️ **Nought is the read-only reading, and that is the load-bearing half.** The
safe answer has to be the one a forgotten declaration falls into, and here the
safe answer is *fewer* keys: a read-only client that quietly authored would write
the author's files off a key its footer never named, while a tool that quietly
stopped authoring loses `a`, `e`, `n`, `d` and `enter`. So the tool that can
author declares it (`model.ctx`, `newModel`, and both test fixtures in
`internal/screen`); the client that cannot has nothing to declare and therefore
nothing to forget.

⚠️ **Which suite catches a dropped declaration was measured, and it is not the
authoring client's.** Inverting one guard at a time and counting failing tests:
`skills.go`'s `a` reddens **internal/screen 4 · cmd/hexarena-tui 2 ·
cmd/hexforge-tui 0**; `squads.go`'s `n` reddens **internal/screen 10 ·
cmd/hexarena-tui 2** and cmd/hexforge-tui as well. The nought is that client's
`everyScreen` setting `SkillsScreen.Adding` **by hand** rather than pressing `a`,
so nothing in that package drives the key at all. The authoring half of these
guards is therefore held in `internal/screen` and the read-only half in
`cmd/hexarena-tui`, and neither client's suite alone covers both.

⚠️ **A key announced on a screen that ignores it is worse than one nobody was
told about** (`internal/screen/picker.go`), so the footers are measured rather
than reasoned about. `cmd/hexarena-tui/readonly_test.go` is four tests because
the claim has four independent halves: the keys do nothing (driven through the
real model), no footer the client draws names one, **the list of such keys is
derived** — every keystroke the suite can send, pressed under both readings, and
held equal to the written list in both directions — and the two footers differ,
because a read-only footer identical to the authoring one would satisfy the
second test by naming nothing at all.

### `draw.Fight` means two different things, and that is the PvP seam

`internal/screen/squads.go` raises `Action{Kind: Raise, Target: Fight}` about a
squad **by id**, and the catalogue is in the shared package — so both clients
receive it and each turns it into one of its own screens.
`TestEveryRaiseTargetNamesAScreenInThisClient` is held total **per client**,
which is what makes a Target sayable for a screen this package will never draw.

- In **cmd/hexforge-tui** it means *pick a second squad and measure the pairing*:
  two choosers, both arrangements over the same seeds, a win rate with a control
  that is exactly 500‰ by construction.
- In **cmd/hexarena-tui** it means *take this side into a battle*. The named
  squad is `Home` — the side the player fights on, since `Squad.Take` fields home
  as the ally half — and the opponent is `Away`.

⚠️ **The opponent is the seam and it is stubbed rather than invented.**
`README.md` § *PvP over a LAN* says what the real answer is: two people on a LAN,
each bringing a squad saved on their own machine, and a **server** that pairs
them. Until a server sends one, `cmd/hexarena-tui/pairing.go` takes **the next
side on the file, wrapping** — which is one side against a copy of itself when
the catalogue holds one, the pairing every fixture in `internal/screen` opens on
and the one the authoring tool's fight calls its control. `pairing` is one
function on purpose: when the server arrives it is the only thing that changes,
and the battle screen never learns that the second squad came off a socket.

⚠️ **A mirror was the obvious answer and is measurably worse.** Two identical
sides make the halves of a battle interchangeable — same board, same roster, same
order line whichever way round they are fielded — so *nothing* can see a client
that opens the battle with the sides swapped. The fixture therefore builds its
two sides around **different characters**, and
`TestABattleOpensOnTheSideTheRaiseNamed` walks **every** row of the catalogue and
names both halves, because home alone is satisfied by a client that fields the
named side twice and a cursor on the last row cannot tell `+1` from correct.

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

**A name on a log line is a caller-supplied MAP, never a `Lang`.** `tui.Line` and
`tui.Log` take `glosses map[string]string` — data id → Vietnamese name — beside
`tags`, and put it in brackets after the id at **every** occurrence
(`uses venoshock (độc kích) at 3,1`). The map is the shape it is because of the
package doc above: an event line is built from the event alone, so this package may
not be handed books, a battle or a language, and `tags` and `Summary`'s `names` are
already the same shape. **A nil or empty map reproduces the line byte for byte** —
that is what English is, what a replay drawn without books is, and the property
`opening.golden` holds. The bracket has **one** definition, `i18n.GlossBracket`;
do not re-spell `"%s <%s>"` in `tui`.
⚠️ **The gloss is written `<>` because round brackets nested.** This log names the
trait a status came from as a parenthetical of its own, so a round gloss inside it
read `(virulence (độc lực))`. The gloss is the *inner* thing wherever the two meet,
so the gloss is what changed shape — everywhere, since it has one definition, so
every reference screen reads `razor_leaf <phi diệp>` too. `<x>` and `(x)` are both
two cells, so **no width measurement moved**, the 79-of-79 `amplified` bound
included. Five hardcoded literals in two test files spell the punctuation out and
are the only things that had to move with it; **no golden holds a gloss**, which
`git diff --stat -- '*.golden'` proves in one line.
⚠️ **`Lang.Gloss` cannot name a skill and that is the whole reason
`Lang.LogGlosses` exists.** Measured over the shipped books: **43** skills, **0**
of them in `skillGloss`, **43** carrying an authored `name`; **22** statuses, **22**
in `statusGloss`, none with a name field; **11** traits, **0** in a table, **11**
authored. `skillGloss`'s nineteen ids intersect `skills.json` **not at all** —
they are the pre-`name` skills and only `internal/testfixture` still reaches them.
So the three kinds go through three accessors (`SkillName` / `Gloss` /
`PassiveName`), which is also what stops a name authored through hexforge drifting
from the log. **Do not delete `skillGloss`** and **do not build the map off it**:
a test using a fixture skill takes the id-table path and leaves the authored-name
path all 43 shipped skills take completely unexercised, which is why
`TestAShippedSkillIsGlossedFromItsAuthoredName` asserts both halves — the name is
on the line, *and* `Gloss` does not know it.
⚠️ **A colliding id is left out, never picked between.** The map is one namespace
over three books that have none, so nothing can tell which kind an event meant.
`TestNoIDIsGlossedTwice` cannot see this — it holds the compiled tables disjoint
from *each other*, the same within-a-table blind spot the category gloss had, and
`taunt` is a shipped skill id with `taunting` a shipped status id. `LogGlosses`
therefore drops the name (a bare id is this package's declared normal, and a wrong
name is worse than a missing one) and `LogGlossCollisions` is the loud half, over
the shipped books, in `TestNoLogGlossCollidesAcrossKinds`. It is read off the
**id** and not off which kinds offered a name, so a collision cannot go live the
day somebody authors the second name.
⚠️ **The coverage check is a table of ARMS, not of kinds.** `status_resisted` has
four branches and `damaged` two, so a sweep over `battle.KindCount` renders one arm
of each and reports full coverage. `sweepArms` in `internal/tui/gloss_test.go`
tables **16 glossing arms across 12 kinds** with the exact count of ids each names,
plus 13 arms that must stay bare; `glossingArms` is written down rather than counted
off the table, so extending the table is not the same act as claiming the extension
was measured. Adding a branch that prints `event.Skill`, `event.Status` or
`event.Passive` means adding a row.
⚠️ **One arm gave up a word to keep its row on the window, and the margin is now
nought.** `amplified` is the only arm printing two glossed ids in one clause, so
`dragon_drive (long xung) is amplified by expose (phá giáp) x2, power 2300` measured
**82** of the 79 cells there are — a row that fit before glossing (59) and would not
after, on a screen *the budget* measured as having no spare rows. The arm reads
`%s amplified by %s x%d, power %d` now: three cells for the word `is`, in the one
register that can spare it (this log is verb-terse everywhere else — `hits`,
`misses`, `holds`, `lets go of`), which puts the widest reachable row at **79 of
79**. That is the **only** line of English this change touched, and it is why the
en output is otherwise byte-identical rather than entirely so.
⚠️ **Nought margin is deliberate, not lucky, and `knownWide` is empty on purpose.**
The bound is recomputed from the books every run, so the day a longer skill name is
authored the test goes red rather than the log quietly wrapping — which is the whole
value of it being measured rather than declared. `knownWide` stays as the shape for
a breach somebody later decides to accept; an entry in it must carry its measured
figure and its reason.
`TestNoGlossedLogRowOutgrowsTheWindow` measures **reachable** combinations (a skill
beside *its own* condition's status, a trait beside the statuses it actually names,
a reply only from a trait that replies with damage) rather than the longest name of
each kind in one event — that bound is 102 cells and describes no event any battle
emits, and a bound nothing can reach is a bound nobody will fix.
⚠️ **The `Started` line is out of scope and stays as it is.** It measures **86–87**
cells against 79 on the client's own fixture cast and is already over today; it is
exempt from `TestEveryWordingFitsTheMinimumWidth` because it names a unit, and a
unit name is free text. Nothing on it is glossed — not the side, not the element
note, not the unit name.

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
⚠️ **`passives.json` has `flavour` too, and its ban is stricter.** A skill free for
anybody may not name a body and a *restricted* one may (`ingrain` says roots, only
a plant takes it); **a trait has no restriction mechanism at all** — no element,
archetype, species or character — so for a trait the ban is unconditional and no
future field relaxes it. It is a **lead line**, not a replacement: a skill has one
opening sentence to take over, a trait has one to six lines and no opening among
them.
⚠️ **Trait wordings name the thing, never `nó`.** Six of eleven used to lead with
a bare pronoun (`Nó gây %s mạnh thêm %s`); a description is about one unit, so the
pronoun carried nothing and only lengthened the line. Drop it wherever the meaning
survives — and where a sentence has two subjects, name them: a reply is "ai đánh
trúng thì bị phản lại … công **của người bị đánh**", not two `nó` in one clause.
⚠️ **Trait sentence order is narrative, not field order**: what the holder *is*
(grants, resists) → what its **own attacks** do (applies, amplifies, drains) →
what attacking **it** costs (replies) → **when** (while). Field order read
backwards on `venom_blood`, which replied before it said it was immune.
⚠️ **Two share renderers, on purpose.** `forge.Percent` keeps a tenth (`2.5%`) for
**hexforge's tables**, where an author tunes the number and the tenth is what is
being tuned. `i18n.share` **rounds to a whole percent, half away from zero**, for
**sentences**, because a tenth is precision a player cannot act on and the decimal
point is the only mark in the line that is not a comma. History matters: share
*truncated* first, which turned a trait's 2.5 into 2 (skills are priced in
hundreds of ‰ and lost nothing; traits are priced in tens). Rounding is not that
mistake again — 25 becomes 3.
⚠️ **Rounding is safe because of a rule on the DATA: nothing is ever tuned by
less than 1%** (`TestNoShippedShareIsUnderOnePercent`, over every shipped skill,
trait and status). Nobody feels a share that small across a battle, and a
description of one prints `0%` — a feature reading as broken. So the floor lives
in the data; carrying a tenth in the sentence to survive data the rule forbids
would be the renderer paying for a case that cannot happen.
Numbers are **shares of a stat** ("100% công"), never damage figures: a figure is
true for one caster against one target and false at the next buff. ⚠️ The block is
**Vietnamese on an otherwise English screen** — a stated cost, not an oversight;
translate the whole battle screen in one piece or not at all. It borrows
`Lang.Gloss` rather than growing a second vocabulary (two names for one status is
how they drift) — and in **English a bare id is the name**, so `poison` in an
English sentence is the reading working, not a missing gloss. Two entrances:
`?N` / `?TAG` at the battle prompt, and `?` on the hexforge-tui skill listing,
**cast browser or played battle**, all three of which raise `screenBlurb` — one
screen branching on the **kind of subject it was handed**, describing a skill from
the listing, a character's traits from the browser and the option under the cursor
from the battle. ⚠️ It used to branch on `blurb.from` and read those three screens'
state directly; the raiser pushes a `screen.Subject` now and the describer reaches
for nothing (`cmd/hexforge-tui/describe.go`). A listed skill and a battle option
turned out to be **one** subject rather than two — same id, same position, same
paragraph — so the kinds are three (`StatusSubject`, `SkillSubject`,
`CharacterSubject`) and not the four the raise sites suggest. Both describers
moved into `internal/screen` once they stopped reaching (`screen.BlurbScreen`,
`screen.PreviewScreen`); what stayed in the client is `describe.go`, the applier
that says which of *its* screens a subject lands on and which raiser an arrow key
walks. ⚠️ **`blurbScreen.from` is gone**: it was a `screen`, this binary's own
enum, so it could not travel with the describer, and it survived for as long as
one of the blurb's three raisers still wrote `m.screen` itself. All three return
a `draw.Raise` at `draw.Blurb` through `navigate` now, so `model.raisedFrom` —
the slot the client already keeps for a Back — records who raised it, and its
three readings (the `esc`, and the two branches saying which raiser an arrow key
walks) read that. ⚠️ **`Subject.Kind` could not have replaced it**, which is
worth being exact about: the collapse above measured a listed skill and a battle
option to be *one* subject, so the subject cannot tell the listing from the
battle. A **fourth** reading of a skill sits beside that one and is not it:
`Lang.SummariseSkill` is the compact line the played battle draws on every
option's own row, and *Where a form beats a prompt* says why it cannot be this
one with the prose dropped. ⚠️ **The forge form is not the place for it** — 19
fields already show 13 of themselves in a 120x24 window, so a three-line block
under the form costs a quarter of the fields; a screen costs nothing until asked
for. Statuses are the third description, and `Lang.DescribeStatus` is the same
shape from the same house — see *Looking a status up* under Open work.
`internal/i18n/testdata/describe.golden` covers **every** shipped skill, trait and
status in **both** languages, so a balance change moves a line there — that diff is
how a number change reads to a player. ⚠️ English needs singular wordings where
Vietnamese does not (`BlurbCostCooldownOne`, `BlurbStripsOne`): two keys rather
than a plural rule, because a rule would make Vietnamese pretend it has a
distinction it does not.
⚠️ **A list of three takes one conjunction and commas, and `listed` is the only
place that knows it.** The conjunction alone was put between *every* pair, so
three items read `a and b and c` — not a sentence in either language. The
**two-item case is what hid it**: every list the shipped books produce has one
item or two, so `describe.golden` held **zero** of them and both goldens still
hold zero after the fix, which means no golden could ever have caught this and
`TestAListOfThreeTakesCommasAndOneConjunction` is the whole guard. The only 3+ list
anywhere is the fixture's `purify`, three stripped categories — the same skill that
exposed the strips clause's width, and for the same reason: a fixture reaches what
the shipped data cannot.
**Both languages take the same shape and that was measured rather than assumed** —
`a, b and c` and `a, b và c`: conjunction before the final item, comma between the
rest, and no serial comma in either. So `ListComma` is **one key pair** and the
conjunction stays the caller's, which is what lets `join` (prose, `BlurbAnd`) and
`JoinIDs` (untranslated ids, `ElementJoiner`) keep saying which sort of list they
are while agreeing about the grammar. Two joiners with identical *shape* declared
twice is the drift this file keeps a list of; two joiners with their own
*vocabulary* is the distinction `JoinIDs`' doc comment exists for.
⚠️ **The caller owns the conjunction and `listed` owns the comma**, so joining
items that may themselves hold a comma would build an ambiguous list. Every caller
today joins short noun phrases. The two that join **clauses** pass a slice of at
most two — `ReadsStatus` and `ReadsHealth` are the only conditions there are — so
they never reach the comma at all; a third condition is the site to re-read, not
this function.

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

**`hex.Cell` is an `Offset` that may be nowhere, and `Event.Cell` /
`Decision.Aim` are the two fields that need it.** The opposite trap to the one
above: `{0,0}` is the ally back corner, so the coordinate has no spare value to
mean "no cell" — and `omitempty` does nothing to a struct field, so every kind
that places nobody wrote the back corner into the log, as did every passed turn's
aim. `omitzero` on the offset only inverts the error. So absence lives in a
second, unexported field, the tag is **`omitzero`** (the repo's first use; go.mod
is `go 1.27.0`), and readers unwrap with the comma-ok `Cell.Offset()`. ⚠️ **It is
a value, not a pointer, and must stay one** — `--verify` and
`internal/seed/battle_test.go` compare whole events with `==`, which a pointer
satisfies **by address** with nothing to compile-fail on, so every log would fail
verification and no test would say why. `Cell.String()` prints `"none"` when
absent, which is what keeps the TUI goldens byte-identical. ⚠️ **This broke the
log format a second time** — a log saved earlier fails `--verify` on its first
event. `Unit.Cell` and `Roster.Slot` stay plain `hex.Offset`: they are always
meaningful and they key nine `map[hex.Offset]` sites.

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
  ⚠️ **The blurb gets two entries for three subject kinds, and the art preview
  gets none.** A listed skill and a battle option are one `SubjectKind` — same id,
  same paragraph, same footer, only `At`/`Of` differ — so a third entry would
  record the same render twice; `NoSubject` is the arm a raise cannot reach, and
  a client's applier is what proves that. The preview is left out **deliberately**
  and not by omission: it draws rasterised art, so what such an entry would assert
  is an open question — → `TODO.md`, which now carries the reproducibility
  measurement that question was waiting on.
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
  ⚠️ The **art preview** is in no sweep and therefore in no golden, deliberately
  and in all three records: it draws rasterised art, so what such an entry would
  assert is an open question — → `TODO.md`.

Run `make golden` (`go test ./cmd/hexarena-tui ./cmd/hexforge-tui
./internal/core/hex ./internal/i18n ./internal/screen ./internal/seed
./internal/tui -update`) to accept a change and
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

## Species: what a unit is

Shipped. An element says what a unit is made of and an archetype says how it
fights; `cast.Species` says **what it is** — a shell, roots, a lineage — and it is
deliberately the thinnest axis in the repository: an id, a word for a screen, an
optional note, and no stats, no kit and no rule of its own.

- **A fourth allowlist, not a fourth concept.** `skill.Restriction.Species` sits
  beside the others and composes the same way, so `dragon_rage` and
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

## Origins: which story a skill is out of

Shipped. `restrict.origins` is the fifth allowlist and the **broadest**: a
species says what a holder must be and an origin says which fiction it must come
out of, so `rasengan` is closed to every Pokémon and open to every Naruto
character, including the ones nobody has authored yet. It is a list because a
crossover is a thing that happens, and because every other axis here is one.

- **The axis exists because `characters` was the wrong one.** A work outlives
  every character in it, so gating `rasengan` on `naruto.naruto` would say "only
  this one may carry it" and would have to be edited the next time that work lent
  the cast anybody — the same argument `species` makes about a lineage.
- **Enforced in `cast.resolveCharacter` against `declared.Origin`**, beside the
  species check, and brought forward to a half-filled form by
  `forge.CheckSkill`. ⚠️ On that form the origin is a **chooser**, so unlike the
  element and the species list it is answered from the moment the form opens —
  there is no keystroke at which it is unanswered, and a skill kept for another
  work is correctly greyed out straight away. A test that read "nothing answered
  ⇒ nothing unavailable" as covering every axis was wrong about this one.
- **Nearly the whole book is gated now**, which inverts a reading the golden
  report was built on: "free to anybody" used to be the majority and is now six
  skills. That list is `internal/seed`'s `sharedPool`, **written down with a
  reason each**, and `TestEverySkillSaysWhichWorkItIsFrom` fails a skill that
  declares neither. The gate is an allowlist, so an omitted one is silent — a
  Naruto skill authored without it is carried by a Pokémon and the book still
  loads — and that test is the only thing standing between the rule and the
  omission. **A pattern-based exemption would not do**: "every neutral skill" would
  have exempted `rasengan` on a technicality.
- ⚠️ **A column missing from the golden report is a restriction the design
  record cannot show.** Origins arrived and thirty-two rows appeared in
  `skills.golden`'s "who may carry" table with nothing but dashes across them,
  because the report had four columns and the book had five.

## The inert element, and what a character with no type actually buys

Shipped with `pokemon.mew`, the first character to declare `neutral`. The chart
has listed it as inert since the beginning and nothing had ever stood on it, so
this is a value the engine already supported being used rather than a feature.

- **It is a real affinity, not an absence.** `element.Single(Neutral)` builds,
  `Chart.ValidateAffinity` accepts, and `MultiplierAgainst` reads a flat thousand
  in both directions against every shipped affinity —
  `TestMewNeitherGainsNorLosesAgainstAnyShippedAffinity`, which also refuses to
  pass unless *some* other pairing in the cast moves, or it would be measuring a
  chart with no edges rather than an element with none.
  ⚠️ `Dual` still refuses to pair with it, and rightly: an inert second element
  adds no line to the multiplier. It adds a line to the *kit*, which is the next
  bullet, and that is a different question.
- **What it actually costs is the kit, and the cost is the identity.** A unit may
  carry a skill of an element it shares or a neutral one, so an inert character
  can carry **only** the neutral skills — which is the largest single pool in the
  book and, by construction, nobody's signature: everybody's plain moves and no
  element's line. That inverts into the widest learnset in the cast (23 skills
  against Magnemite's 17), and it is why Mew's own five had to be authored neutral
  and gated by `restrict.species` rather than by an element.
- ⚠️ **It does not flatten a matchup profile, and the expectation that it would
  was wrong.** Mew's spar rates run from nought to a hundred across the cast,
  exactly as polarised as every other character's — every shipped character spans
  roughly ninety points. What decides a pairing here is tempo and sustain; the
  chart is a term inside a strike, not the shape of a fight. **Removing the
  elemental term removes the elemental term.**

### A one-form line, and a stat line that is a median rather than a share

- **The two mythics are the first shipped characters that do not evolve**, and
  nothing in the engine moved for them: `progression.Line` has always resolved a
  single form
  (`TestALineWithOneFormStillResolves` predates this by a long way). A level still
  means what it meant — the curve runs base to max, the learnset still opens over
  it — so what a one-form line gives up is the threshold and the fork, not
  progression. `TestTheCastHasALineThatDoesNotEvolve` holds each of those and
  that the pair is both of them and nobody else — a third one-form line would be a
  decision rather than an accident, and this is where it would be noticed.
- ⚠️ **"The same share of every ceiling" is not the same idea as "the middle of
  the cast", and the first one is a trap.** The first draft put Mew at seventy per
  cent of all six declared ceilings, which reads like the fiction's "a hundred in
  everything". The cast uses the ceilings very unevenly — attack tops out at
  nineteen twentieths of its ceiling and **dodge at less than half of its** — so an
  even share handed Mew the cast's best dodge by half again and its best speed
  outright. It sparred **72.4%**. The midpoint of what the cast actually fields,
  axis by axis, reads **56.4%** with hp and attack *higher* than the first draft.
  `TestMewHoldsNoExtremeOnAnyStat` is the claim, and it is stated against the
  shipped top forms rather than against `progression.json`.

### ⚠️ A one-turn control status cannot be a setup for a skill on a cooldown

The most useful thing measured here, and it is about the engine rather than about
Mew. `dream_eater` reads a condition — hit harder into a stunned target — and the
character carrying it is also the one applying the stun. That looks like a two-card
combo and is not one.

`stun` lasts a single turn and the turn it lasts is the **target's**, which the
target then spends being skipped. So the whole window in which the condition holds
is one slot of the turn queue, and who owns that slot is decided by speed alone.
Sixty duels each, same kit, same stun source:

| opponent | speed | stuns landed | amplified |
|---|---:|---:|---:|
| Blastoise | 85 | 239 | **50** |
| Venusaur | 100 | 66 | 2 |
| Sennin | 134 | 77 | 2 |
| Charizard | 140 | 31 | **0** |

Two things follow, and both were measured rather than reasoned:

- **Raising the chance makes it worse.** Swapping the source for `hypnosis`, which
  lands more than twice as many stuns, took the conversion *down* — 9381 stuns to
  6 amplifications against Blastoise. The turn that lays the status down is the
  same turn that could have spent it, so a kit that stuns more attacks less and
  reaches the window less often. That is why `hypnosis` is not among the four the
  character brings by default; it lives in a build.
- **Shortening the skill's cooldown moves almost nothing** (3 → 2 → 1 changed the
  Charizard column not at all), which is the tell that the bound is the queue and
  not the availability.

So an amplifier on a duration-one control is a **speed check**, and should be
priced as a rare doubling rather than as a combo. A condition meant to be set up
on purpose wants a status that outlives the setter's next turn — `expose`, `mire`
and `weaken` all last two or three — and `venoshock`/`dragon_drive` are the shipped
skills that do it that way. `TestAOneTurnSetupIsAQueueRaceRatherThanACombo` holds
the direction rather than the figures.

### The other half of the pair: `dark`, `pierce`, and what a stat line is for

`pokemon.mewtwo` is the first character to carry **dark**, which is the other half
of the chart's only mutual pair. Light had a carrier — Cleffa — and dark had
none, so for as long as that was true Cleffa's element was, in play,
indistinguishable from an inert one: strong against nobody, weak to nobody.
`TestTheMutualPairFinallyHasBothHalves` is the claim that the pair is now
*fielded* rather than declared, and it checks both directions, because both ways
is what mutual means and what a cycle never gives.

- **What Mewtwo is for is `pierce`**, an axis three shipped skills touched and no
  character was about. ⚠️ **The obvious measurement of it cannot say anything**:
  three attackers of different elements against three targets of different armour
  is three different chart readings, and the elemental term swamps the one being
  read. What works is **one attacker carrying two skills** — `psystrike` pierces
  800 and `body_slam` pierces nothing — thrown at the same targets, so the
  elemental term is a constant that divides out. Across the whole armour range the
  cast fields (Blastoise 640 down to Magnezone 340), armour costs the piercing
  blow **175 per mille** and the plain one **381**.
- **`mewtwo.origin` carries Mew's own four on Mewtwo's frame**, which makes it the
  only reading in the package that puts one loadout on two bodies — and therefore
  the only way to say what the stat line actually changed. ⚠️ **It changed it for
  the worse on every column that kit is about**: fewer turns, less healing (a
  restore reads attack, and the clone's is lower) and less damage. A stat line is
  not "more"; it is **pointed**, and Mewtwo's points at skills that pierce —
  `mewtwo.breach` on the same body out-damages it better than two to one.
  `TestTheSameLoadoutIsWorseOnTheClone` holds all four of those.
- **`mythic` is the first species in the book with two members.** Every other one
  gates for exactly one character, so this is the first time the axis does the
  thing it was added for: five skills authored for Mew are carried by both,
  and `TestTheCloneKnowsWhatTheOriginalKnowsExceptHowToBeSomethingElse` refuses a
  species gate that one of its two members declines — that is a character gate
  written the long way. The exception is `transform`, gated on the character, so
  the one thing the clone cannot do is be anything else.

### ⚠️ At the top of the cast, speed is the dominant currency and armour is nearly free

Tuning Mewtwo took it from **98.5%** to **68.0%**, and almost none of that came
from where it was expected. Measured one dial at a time, everything else held:

| dial | reading |
|---|---|
| `psystrike` pierce 700 → 400 → 0 | 98.4 → 97.0 → **90.1** |
| speed 150 → 130 → 110 → 90 | 98.4 → 91.4 → 79.7 → **69.3** |
| attack 740 → 620 → 520 → 440 | 98.4 → 89.7 → 79.4 → **71.3** |
| hp 3200→2900 **and** defence 300→240 together | 98.4 → **96.5** |

So the eight-hundred-per-mille signature is worth about eight points, thinning
the frame past the thinnest in the cast is worth two, and **sixty points of speed
is worth twenty-nine**. A glass cannon is not paying for its glass: the enemy
needs turns to spend the opening, and a fast unit does not give them.

⚠️ **This is not the finding `swiftness` failed on and does not contradict it.**
That one is about *trait-sized* deltas — `+150` read 59.0% where `+50` read 74.0%,
which is noise wearing an ordering — and it still stands. What is ordered cleanly
here is a **sixty-point** range, four readings, monotone. A duel rate can separate
speeds that are far apart and cannot separate speeds that are close, which is the
same fact as "the queue is discrete and lumpy" seen from the other end.

The practical consequence for authoring: **the EffHP budget cannot hold a
character down.** It bounds survivability and nothing else, and survivability is
the cheap half. Mewtwo passes the budget at 5800 of 11500 — the thinnest line in
the cast — while reading 98.5%. Tune the speed and the attack; the budget will not
tell you.

## A strip pointed outward: what `dispelled` was missing, and how it was found

Shipped with `pokemon.gastly`, the first character to carry a strip aimed at an
**enemy**. `battle.Suggest` has had `dispelled` since strips were priced at all,
and it had never once fired on shipped data — both existing strips (`rapid_spin`,
`rinse`) point at the caster's own side and go through `cleansed`.

⚠️ **There are three things a strip can take and it priced one.** A stat buff
moves a number, so `dispelled`'s hypothetical reads it for free. A **shield** and
a **regeneration** move no stat at all — so both came back nought, and an
opponent handed a dispel declined it in favour of a ten-power poke. Measured
before anything was written: an actor holding a strip and a `jab` chose the `jab`
against an enemy carrying three block charges, three regeneration stacks, and both
at once.

The two terms added are the **inverses of the ones that price putting each on**,
read from those functions rather than written again — `unguarded` is `shielded`
backwards (charges times `strikeThreat`, clamped at `guardHorizon` for the same
reason), and `undone` runs the denied ticks through `worthHealing`, the same three
clamps a heal is priced by.

- ⚠️ **A regeneration on a unit at full health is worth nothing to take**, because
  the healing it owes cannot be banked. That falls out of `worthHealing`'s room
  clamp rather than being written, and it is a row in the test rather than a
  footnote.
- ⚠️ **`undone` refuses to fire when the strip names a harmful category.**
  `Pending` totals every stack's frozen tick without asking whether it heals or
  hurts, so a strip naming `dot` would be taking a poison *off* an enemy and the
  same difference would report that as a gain. Nothing shipped does that; the
  guard is there because the arithmetic cannot tell.
- **Nothing moved.** The whole suite, every golden, every measured rate: unchanged
  by the fix, because no shipped skill reached the arm. That is the tell that a
  priced branch was dead rather than wrong.

### ⚠️ Two rows of that test passed for the wrong reason, and only a mutation said so

`TestADispelIsPricedForEachOfTheThreeThingsAStripCanTake` was first written with
every enemy at half health. Deleting the shield term left it **green**: on a hurt
target the rating prefers the strip anyway, for a reason that is not the shield.
The block row now reads a target at **full** health, which is the only version
that goes red when the term it exists to hold is removed.

The same trap caught the build test one file over. Two Gengar kits made *identical*
and told apart only by their traits still read 1300 damage against 1178 and 153
statuses against 81 — `contagion` weakens what it hits and a weakened enemy takes
longer to win — so `miasma.dealt > unbind.dealt` passed on two copies of one build.
The margin asked for is now the one the two kits actually produce, plus a
disjointness check. **A separation test needs a margin taken from the mutation, not
from the reading.**

### A dispel cannot be measured by a duel either

`spite` takes a shield and a regeneration off an enemy, and neither is a thing a
lone unit brings to a duel — so `hexforge spar` reads it as a 700-power attack and
says nothing, exactly the blind spot that made Cleffa read 0 to 7 per mille. The
instrument is the mender's: one striker, one wall, and the slot under test, fought
both ways round. Three opponents differing only in their wall's fourth skill, two
Gengars differing only in `spite` against `bite`, 300 battles a row:

| opponent wall | kit | strips | my blows blocked | the wall healed |
|---|---|---:|---:|---:|
| shields (`withdraw`) | with `spite` | 508 | **1541** | 270722 |
| | without | 0 | **2434** | 276988 |
| only hits | with `spite` | **0** | 0 | 0 |
| | without | 0 | 0 | 0 |
| regenerates (`aqua_ring`) | with `spite` | 589 | 0 | **297579** |
| | without | 0 | 0 | **899834** |

⚠️ **The win rate is not what is being read** — it moves 500 to 540 per mille
against the shielding wall, because a shield is not the whole of a fight. What a
squad says exactly is what the skill *did*: a third of the blows the block would
have eaten no longer are, and the regeneration gives back barely a third.

⚠️ **The middle row is what makes the other two mean anything.** Against a squad
with nothing to take, the skill strips nought and is a plain attack, and the rate
goes very slightly the *other* way. A utility skill worth its slot everywhere
would not be a utility skill.

⚠️ **A restore is not a regeneration.** `withdraw` heals on the turn it is cast
and cannot be taken back, which is why the shielding wall's two healing figures
are level while the regenerating wall's differ by two thirds. Only the ongoing
tick is strippable.

### ⚠️ What a dispel may not name, and why `buff` is not in it

`status.Set.Cleanse` does not skip permanent statuses, and only a **gated** trait
(`While != nil`) re-applies its grant — `Battle.hold` runs once at enlistment for
everything else. So a strip naming `buff` would permanently take `toughened` off
an `endurance` holder, `evasive` off an `elusive` one, `quickened` off `swiftness`,
and never give them back. That is a whole design decision about whether a trait can
be removed, not a side effect to ship inside a character, so `spite` names `shield`
and `regen` and nothing else. Do not widen it without deciding that question first.

## Two shapes of guard: `absorb` beside `shield`, and the blow that ignores both

Shipped. `status.Absorb` is the tenth category and the second shape a guard comes
in. It is **not** a smoother `Shield`, and the difference is the whole reason it
exists:

| | `shield` (`block`) | `absorb` (`aegis`) |
|---|---|---|
| what a stack is | a **charge**, counted | a **pool**, in health |
| what it stops | one **strike**, whole | that much **damage** |
| against one heavy blow | cancels it entirely | eats what it can, the rest spills |
| against a split volley | one charge a strike | the same total, same drain |

So a wall of charges is the answer to a single heavy blow and multi-strike is the
answer to the wall — which is `warden`'s identity and `bruiser`'s counter to it —
while a pool is **indifferent to how the damage arrives**. Measured on equal
totals: against two charges a `heavy` dealt **0** and a three-strike `triple`
dealt **100**; against a barrier both dealt **100**.
`TestABarrierEatsDamageWhereAShieldEatsStrikes` holds exactly that, as the two
guards against each other rather than either against nothing.

Three things follow and each is a test:

- **A strike a pool ate still CONNECTED.** It arrived and was paid for; only its
  damage went elsewhere. So the riders it carries land, its drains drain what
  actually reached health, and `Count(attempts, Struck)` counts it — none of which
  is true of a blocked strike, which never happened. That is the complement of
  `OutlastsAShield`, not a case of it.
- **A pool spills.** A blow bigger than what is left is not cancelled; the
  remainder goes through, which a charge can never do.
- **An unblockable blow leaves the guard standing.** It goes *past* a barrier
  rather than through it, so the next attacker meets a full one. Spending what it
  ignores would make one skill quietly the strongest strip in the game.

### Where each piece lives, and why the split is not arbitrary

- ⚠️ **`Stack.Pool` is its own field, not `TickAmount`.** It is the only per-stack
  figure in the package that goes **down**. `TickAmount` is frozen and multiplied
  by the turns left — that is what `Tick` charges for and `Pending` totals — so
  sharing one field would have a barrier tick its holder for its own strength and
  `Pending` price a guard as though every point were owed once a turn. The same
  argument gives `Kind.PoolPower` its own field: the parser refuses a `tick_power`
  on anything that does not tick, and relaxing that for one category would lose
  the guarantee for every other.
- ⚠️ **`combat.Absorb` is a pass over the outcomes, not a branch inside `Roll`.**
  A charge is spent *during* the roll because it decides whether a strike happens;
  a pool is spent *after*, because it only reduces what landed and the critical
  has to be rolled before there is a figure to eat. Two mechanics, two moments.
  The pass touches no randomness and walks the attempts in the order `Roll`
  produced them, so `combat` stays a pure function of its arguments and a battle
  with a barrier on the board still replays from its seed. It also meant `Roll`'s
  signature never moved — one production caller and eighteen test callers left
  alone.
- **`Set.SpendPool` drains oldest stack first**, and the order is the replay
  contract rather than a preference — entries sit in a slice in application order
  for the same reason a tick resolves in one.
- **`skill.Unblockable` is a switch where `Pierce` is a ratio**, and that is not
  an inconsistency: defence is continuous, so a ratio walks an armoured unit's
  edge instead of jumping to the end of it, while a guard is **discrete**. "Half a
  block charge" is not a quantity this engine has. It is spelled in `battle` as
  *there is nothing to spend* — the charges are set to nought and the pool pass is
  skipped — so `combat.Roll` needs no flag at all.

### ⚠️ The pricing bug this shipped with, and the test that caught it

`guarded` first computed the pool and **multiplied it by the stack count**. That
rated putting a barrier on a unit that already had one, every turn, forever — the
same defect the "two units buffing each other" term exists to stop. The fix is the
one `shielded` and `standing` already use: build the hypothetical through
`Set.With`, so the cap, the refresh and the wasted stack are the ones `Apply`
resolves, and read the *difference*.

It was caught because the test asked the rating twice — once with nothing up and
once **at the cap** — and only the second row failed. A pricing test with one row
is a pricing test that cannot see a missing clamp.

And the reason the test exists at all is the previous PR: `dispelled` sat in the
rating for a long time pricing a shield at nothing, so an opponent handed a dispel
declined it. **A guard nobody rates is data.**

### What shipped on top of it

`aegis` (lá chắn), pool 1800 per mille of the caster's attack a stack, 2 stacks, 3
turns. `light_screen` puts it on an **ally** — Cleffa's, because a barrier a
mender hands somebody else is the half `withdraw` cannot do. `shadow_punch` is
Gengar's and is the counter to every form of mitigation at once: `pierce: 1000`,
which `Pierced` resolves to exactly nought defence, plus `unblockable`.

⚠️ **`pierce: 1000` is the switch `Pierced`'s own comment argues against**, and
shipping one is a deliberate exception rather than an oversight. The argument
stands — a switch makes an armoured unit worthless against one skill and
unaffected by the next — so the skill that takes it is paid for elsewhere: 900
power on a four-turn cooldown, which is a third of what `cross_chop` does per
turn. It is the answer to armour that costs you, not a default.

⚠️ **Neither new skill is in a build**, deliberately. A build is a measured claim
about four skills and a trait; these two are new tools in a learnset, and whether
either earns a slot is a separate question with its own measurement. Measured
firing in real battles first, though — 60 squad battles put 354 barriers up and
soaked 128,329 damage across 446 strikes — because a skill nothing casts is the
failure mode this section already has one example of.

## Three new axes: a grant with a number, a share past defence, a price in health

Shipped together because the first two are the wall and the third is what a
character pays to break one.

### A grant that carries a quantity

`passive.Grant` was `{Status, Stacks}` and every grant before this was a
**switch**: `toughened` is a status whose modifiers say what it does, so granting
it needs nothing but its name. A barrier is not a switch — it is a pool, and how
deep the pool is is the whole of what the trait says.

- ⚠️ **`Set.Hold` applied a nought amount**, on the argument that a permanent
  status can be neither a damage-over-time nor a regeneration and so has nothing
  to snapshot. That was true of every permanent status there was, and stopped
  being true the moment a guard could be permanent. A barrier granted that way
  parses, applies, shows in the log and **stops nothing at all**.
- ⚠️ **The permanent-absorb refusal was the wrong rule in the wrong place.**
  `status.ParseBook` refused it as "a guard with no clock", and that is wrong
  about a pool: a pool has a clock of its own, which is the damage in it. What is
  actually dangerous is a **gate** — `hold` and `release` run a grant again every
  time one reopens, so a barrier behind `below_health` comes back full each time
  its holder crosses the line. The rule now lives on `passive.ParseBook`, where
  the gate can be seen.
- **The pool is read off the holder's BASE line.** Grants go on one trait after
  another at enlistment, so a buffed reading would make the answer depend on which
  trait was applied first — a unit with `endurance` and a defence-scaled barrier
  would get a different barrier in one declaration order than in another, with
  nothing on either trait saying so.
- The stat comes through `skill.ParseScaling`, which is the sentence that function
  is exported under, and it brings the health refusal with it.

### ⚠️ A pool is worth most where the frame is thinnest, which makes one of the two traits self-defeating

A pool is a flat quantity, so what it is worth is a share of what its holder could
already take. Attack does not correlate with survivability, so an attack-scaled
barrier lands wherever it lands. **Defence does** — so a defence-scaled one hands
the deepest pool to the unit that needs it least, and on a wall it competes with a
permanent defence stat that is worth more precisely because the defence is already
high.

| carrier | trait | spar | the lead trait it replaces |
|---|---|---|---|
| Charizard | `projection` (attack) | **71.9%** | `blaze` 63.2% |
| Blastoise | `carapace` (defence) | **29.2%** | `endurance` 32.1% |

Both ship: a weaker option is still an option. What is written down is the
*reason*, so a later balance pass does not read the figures as a bug.

### Converting is not a bigger pierce

`passive.Converts` sends a share of every blow its holder throws past defence
entirely. It sits beside `Drains` for that field's own reason: a skill's drain
belongs to the skill and a trait's belongs to the unit.

**Piercing lowers the divisor** — armour gets smaller and the blow is still
divided by what is left, so what gets through keeps falling as armour rises.
**Converting splits the blow** — the converted share is divided by nothing, so it
arrives whole however deep the armour is and puts a floor under the blow that
armour cannot lower.

| defence | pierced 400‰ | converted 400‰ |
|---|---:|---:|
| 0 | 600 | 600 |
| 300 | 375 | 420 |
| 900 | 214 | 330 |
| 2400 | 103 | **280** |

So one is the answer to armour in general and the other is the answer to a **wall**
— which is what makes it the damage dealer's tool against a tank rather than a
second helping of the same thing. Measured on the shipped cast, Machamp with
`rending` against Blastoise: **150-0 over 30 turns**, against `blood_thirst`'s
138-12 over **71**. And it costs where there is nothing to bypass — against
Charizard it wins ten fewer duels than the drain it replaces.

- ⚠️ **Both halves are computed unfloored and the floor is applied to the sum.**
  Two `damage` calls would floor twice, handing a converting attacker a free
  minimum-damage point that grows with nothing.
- ⚠️ **A conversion of nought takes a separate branch** rather than falling
  through at a share of a thousand, which truncates differently. That is what let
  this ship without moving a single figure in a single golden —
  `TestConvertingNothingChangesNothing`.

### A price in health

`skill.Cost` is a share of the caster's **maximum** health, paid **up front** and
**whether or not anything lands**. A share taken out of the damage dealt would be
free on a turn the skill missed, and a skill that is free when it fails has no
decision in it. Of maximum rather than current, so the price is the same on the
first turn and the last.

⚠️ **It may never take the last point.** `Suggest` prices what a turn is worth to
the board and has no term at all for "and then I am not here", so a cast that
could be lethal would be rated as pure gain. The floor keeps the whole question
out of the rating.

#### ⚠️ The bug it shipped with: a cost filed in a branch that does not run

A health cost is a cost of *acting*, so it went into `friendlyFire` — which is
where every other cost of acting lives. But `rate` calls that arm only when
`declared.Target == skill.All`. So a single-target skill charging a quarter of its
caster's health was charged **nothing**: measured, a Magnezone holding one cast it
three times a battle, handed over seven tenths of itself and lost **120 of 120**
duels, against 69-51 for the same kit without it. It is subtracted in `rate` now,
for every skill.

#### ⚠️ And the clamp that made it cheapest when it was most fatal

The price was first read through the same expression that pays it — the floor
included — so the skill got *cheaper* exactly as casting it got more dangerous: a
unit at two hundred was charged a hundred and ninety nine instead of seven
hundred and fifty, and the rating therefore liked it best in the one position
where it should refuse.

`spentHealth` is now the one figure in `price.go` deliberately **not** the one the
resolving function pays. The rule the file is written under is about arithmetic
that could drift; this is the same arithmetic with one clamp left off, and that
clamp is *what a unit can afford* rather than *what the skill costs*. A unit that
cannot afford it should decline.

With both fixed, `steel_beam` is cast when the board is worth it and not
otherwise — against Blastoise (defence 640) 30 casts in 120 duels, against
Charizard (400) **129**.

## Pricing a summon, so the opponent casts one

Shipped. `battle.Suggest` rates a summoning skill in the one unit it counts in —
damage — as **the damage the copies would deal over the turns they are given**.
Before it, a summon had no power of its own, so it reached `Suggest` only as the
*fallback*: the option taken when nothing at all could be hurt. The shipped
summoner therefore never called anybody up while it had a kunai in reach.

- **A summon is the only thing in the book that buys turns rather than spending
  one**, so the turns are the price: `summonWorth` puts a hypothetical copy in
  the cell `summonPlaces` would give it, at the line `summonStats` would give it,
  with the elements `summonAffinity` would give it, and multiplies its best
  single-turn attack by the horizon. All four are the functions that do the real
  thing — ⚠️ **two of them were extracted for this**, because a rating built from
  its own reading of "where would the copies stand" is a rating that pays for a
  copy the board has no room for.
- ⚠️ **The horizon is capped, at `summonHorizon`.** The honest horizon for a
  summon that *stays* is the rest of the battle, which this rating cannot see and
  which would put such a skill above every attack in the book for ever. So a
  summon is priced for its own `lasts` when that is shorter and for the cap when
  it is longer or absent. `summon_toad` is the case that needs it: no `lasts` at
  all. The cap direction is deliberate — over-pricing costs a kill, under-pricing
  costs a cast that was marginal anyway.
- **It is an upper bound and says so**: the turns are what the skill promises,
  not what the board will grant. A copy can be killed on arrival, and a `bound`
  one leaves when its summoner does. A shallow rating knows neither.
- ⚠️ **A cast worth nothing falls through to the fallback rather than scoring
  nought.** A rating of nought is still a rating — it beats "no damaging option
  at all" — so on a full board it would take the turn ahead of a shield that
  would have done something.
- ⚠️ **No golden moved and that is not "no effect".** Naruto is cast-only, and
  nothing in the roster summons, so `scenarios.golden` and `replay.golden` cannot
  see this. What moved is the **balance answer**, at 2000 seeds a slot:

  | | before | after |
  |---|---|---|
  | naruto.naruto overall | 56.4% | **93.8%** |
  | · vs bulbasaur | 0.0% | 81.5% |
  | · vs charmander | 69.5% | 100.0% |
  | pokemon.bulbasaur overall | 66.6% | **39.5%** |
  | naruto mirror, turns | 36 | 303 |

  The summoner was **losing every single battle to Bulbasaur** and now wins 81.5%
  of them. Nothing about the character changed; the opponent started using its
  kit. So the shipped numbers were never measuring the character that ships, and
  **Naruto now needs a retune** — that is a data PR, not this one.
- The mirror's slot skew flipped from **+27.1% to −71.8%** and its battles got
  eight times longer. Not a limit being hit (4000, and no draws) — two summoners
  regenerating bodies at each other. Worth a look before the retune.

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
- **Vulnerability** — the mirror where a target is *easier* to affect — is built
  and has **no shipped user**. It is `Resists` with a negative share and reuses
  the whole composition; the decision it needed was made (`Refused` is signed, and
  the early return in `resist` no longer treats "nothing refused" as "nothing to
  do"). What is missing is a trait the share is right for.
  - It was trialled on `reckless`, which is its natural first user, at −200‰
    against each of the six harmful statuses that can actually be inflicted, and
    **it made the build worse**: the dragon-vs-fire duel fell 22.0% → 16.1% on the
    vulnerability alone. Not shipped. → `README.md` § *What `bare`'s dodge clause
    was worth*.
  - ⚠️ **A vulnerability costs more than its arithmetic, because the opponent
    steers into it.** `pricing.landed` calls `fight.resist` on the *target*, so
    the rating sees that a unit invites a status and aims one at it on purpose. A
    negative share is not a multiplier applied to whatever would have happened
    anyway — it changes what the opponent chooses to do, so a share-times-uptime
    calculation under-prices it and a reading that looks too harsh is probably
    right. Do not sanity-check one against paper and conclude the harness broke.
  - The rendering half is proven: `BlurbTraitVulnerable` gets its own sentence
    rather than the resistance one with a minus in it, and a −200‰ share prints as
    `20%` in both languages — "Tăng 20% khả năng dính bỏng" / "Takes 20% more of
    any burn aimed at it" — with the sign carried by the verb.

## Pricing a detonate, so the burst is paid for

A detonate is `requires` with `consume: true`: the skill is amplified while the
target carries a status, and eats it on the way through. The rule is that it may
beat leaving the status alone, but **not by more than a factor of two** —
otherwise applying a status and immediately spending it is the only line worth
playing, and the status is a cast animation.

- **A detonate is only as big as what it spends.** The ceiling is set by what
  consuming the status throws away, so the fuel decides the skill, not the other
  way round. `burn` throws away **548**; `expose` throws away **102**. That is why
  `inferno` bursts for 1200 and `dragon_drive` for 788, and why a line whose only
  status is a stat debuff cannot have a big detonate without first being given a
  status worth detonating.
- ⚠️ **There are two currencies and the arithmetic is different.** A
  damage-over-time is worth its remaining **ticks**, which land whatever the
  attacker does next. A stat debuff ticks for nothing and is worth the **extra
  damage ordinary attacks land while it is up** — priced by hitting the lowered
  defence and the real one with the same plain attack and charging the difference
  for every turn the debuff had left. `forgoneBy` in `skills_test.go` is the one
  place that knows both, and the golden table names which one each skill spends.
- ⚠️ **The tick-only reading was shipped and was a wrong answer, not a missing
  feature.** `tick power × stacks × duration` is **nought** for a stat debuff, so
  a detonate off one was priced as giving up *nothing* and a burst of any size
  would have passed. It went unnoticed because every detonate was off a DoT.
  Both the test and the table now **refuse** a status they can price in neither
  currency, rather than returning nought — nought and "gives up nothing" are the
  same number and only one of them is true.
- **The consume happens before the damage is computed**, in `resolveAgainst`:
  `target.Statuses.Consume` runs, *then* `b.Stats(target)` is read. So a detonate
  off `expose` hits into the defence it just handed back — it pays for its own
  burst, including against itself. Nothing had to be written for that; it is the
  ordering, and it is worth not disturbing.
- **A detonate is a mechanism, so assert it as one.** A win rate cannot see
  whether the combo is played: `TestTheDragonLineCanSpendWhatItApplies` counts
  `StatusApplied`, `Amplified` and `StatusConsumed` off the log and requires the
  last two to be **equal**. A burst that amplifies without consuming is the
  strictly-better skill this whole rule exists to refuse, and no golden would
  notice.

## A consume can be paid in shape: `charge`, the counter category

`charge` is a status category that does **nothing** to its holder — no stat, no
tick, no turn. It is ammunition, and it is the ninth category, appended last for
the reason `taunt` and `heal_cut` were. Full reasoning in `README.md` §
*Counting instead of doing*; what matters when editing:

- **It is the one category `Book.MaxStacks` does not bound.** `max_counter_stacks`
  (999) bounds it instead, and a charge carrying a modifier or a tick is refused
  at parse — the ceiling was granted on the understanding that it multiplies
  nothing. Do not "tidy" that into one cap; the two numbers bound different
  things.
- **`reserve` is the same category on the caster's own side**, and the two are
  told apart by `Harmful` alone: a charge goes on an enemy so a cleanse strips it
  (`rinse` names it), a reserve is its holder's fuel so only an enemy's dispel
  does. `Category.Counter()` is what the cap and the modifier refusal ask, so
  appending a third counter means adding it there and nowhere else. Spent through
  `self_requires`, and a reserve is priced FLAT up to `pricing.capacity` — the
  deepest single spend the holder's kit owns — rather than halved like a charge,
  because a reserve spender cashes the whole run at once. Measured: with the
  halving the shipped loop banked 456 stacks over forty duels and spent none.
- **`self_requires` gained `stack_power`**, the caster-side answer to `arc_power`:
  power per stack consumed, so "spend everything" has arithmetic in it. Bounded by
  `skill.MaxSpendPower` applied in `Condition.Takes` — on the STACKS, never on the
  power, so the leftovers stay in the tank; clamping the bonus instead is the bug
  `Skill.Cost` shipped one field along. `Skill.SelfCeiling` is the reading a bound
  must use, because `Satisfying` is the cheapest case and a scaling payment is
  worth most at the deepest.
- ⚠️ **`Battle.spend` called `Set.Consume` for as long as the field existed**, so
  `consume_stacks` on a caster's own condition parsed, round-tripped and was
  thrown away — the whole pile went whatever the author asked for. Nothing noticed
  because nothing shipped ever spent anything of its own.
- **Never permanent.** `Set.Remove` refuses a permanent status (that is what keeps
  a trait from being dispelled) and a consume goes through `Remove`, so a
  permanent counter is one nothing could ever spend.
- **`requires` gained `chains`, `damped`, `arc_power` and `consume_stacks`.** A
  condition carrying `arc_power` is a **conduit** and is a different animal from a
  detonate: it damps its own blow (`damped`) and fires the stored charge instead
  (`arc_power`), one stack **per strike**, along a chain of adjacent carriers. The
  old refusal "consumes for no bonus" is now "for neither a bonus nor a discharge",
  and a skill may take one payment, never both.
- ⚠️ **`charge` outlasts a shield, and is the SECOND category ever to.** The
  predicate's own comment warns against completing it, and that warning is about a
  *stat* getting through (mire broke the mirror invariant); a counter carries no
  stat. `TestOnlyContaminationOutlastsAShield` is a written-out table rather than a
  rule, because what Dot and Charge share is a sentence and no predicate spells it.
- ⚠️ **The arc is not the skill's damage and must not be routed through the skill's
  machinery.** Not aimed, not rolled against accuracy or dodge, **not blocked** —
  it is what was already sitting on the target. It is logged as `Damaged` with
  `Status` set, the way a reply is logged with `Passive` set, so a renderer that
  switches on kind still draws the health it took.
- ⚠️ **A miss delivers nothing, a block delivers everything.** `discharge` runs on
  every strike outcome except `Missed` — the same sentence `OutlastsAShield` is
  written on. Do not "tidy" that into one condition.
- ⚠️ **The chain stops at the midline**, and it is the one shape in the engine
  that had to be told. Every other is a pattern, and `pattern.Targets` already
  drops a splash cell on the far side — `Side.CrossesSides` is the only thing that
  lifts it. A chain reads the board instead, so it obeyed none of that: aimed at an
  enemy standing next to a charged teammate it walked straight back over the line,
  for 272 damage and two of that teammate's stacks. The *enemy's* own chargers
  arrange that for free. `TestTheChainStopsAtTheMidline`.
- **The chain is `Battle.chainFrom`: BFS from the aim over `NeighborsOnBoard`,
  through carriers only.** It is re-walked **every strike**, because it shrinks as
  it burns; and it returns nothing at all when the *aimed* unit is clean, which is
  the gate on the entire mechanism. Deterministic — the map is asked only for
  membership, and no map iteration reaches the result.
- ⚠️ **`hex` adjacency is not what a reader guesses.** `3,1`'s neighbours are
  `4,2 4,1 3,0 2,1 2,2 3,2` — **`4,0` is two cells away**, so a chain reaches it
  only through a charged `3,0`. Odd columns sit half a cell lower; check with
  `Offset.NeighborsOnBoard` rather than by eye.
- **A rolled strike count**: `repeat_chance` + `max_strikes` on the skill,
  `Hit.RollStrikes` in combat. ⚠️ **Wire BOTH into `combat.Hit` when building it in
  `resolveAgainst`** — the first cut set neither and every roll silently returned
  the floor, which no test caught because the floor is a legal count.
  `ExpectedStrikes` is the mean and is what everything outside the roll reads;
  the description quotes floor/odds/cap instead, because the mean describes no
  cast anybody will have.
- ⚠️⚠️ **A conduit's own figure is a CONSTANT — the arc is added, never traded
  for.** There was a `damped` field that cut the blow to pay for the discharge; it
  is gone, and the reasoning behind it was the mistake. A stack does not appear on
  a target by itself: the turn that put it there is the price, paid before the
  conduit was cast. A conduit is bought with **tempo** — charging turns and its own
  cooldown. Do not reintroduce the trade; the user rejected it twice.
- **One bound, carried on the skill's own face**: `arc_power < power`. A stack may
  top the blow up, never outweigh it — otherwise the skill is a delivery mechanism
  for somebody else's turn and its printed figure stops describing it.
- **`consume_stacks` has two useful values and no third.** 1 is a drip (`spark`,
  `electro_ball`); 0 is a **nuke** (`overload`) — it takes the whole pile and its
  arc multiplies by the count. A nuke may not `chain`, must need ≥ 2 stacks, and
  must sit on a longer cooldown than the drip.
- ⚠️⚠️ **The drip/nuke rule has been written both ways round and both were wrong**,
  because the answer depends on what the skills give up and that kept changing.
  While conduits damped, a nuke damped *and* waited and was owed the better rate
  (held to a poorer one it dealt 472 at a pile of six — a skill nobody brings). With
  nothing damping, a nuke's compensation for waiting is that it takes the whole
  pile, which is already the larger purchase — so **no better rate than the drip**,
  and a longer cooldown. `TestANukeGetsNoBetterRateThanADrip`.
- ⚠️ **`chainFrom` returns the head alone when `chains` is false.** Returning nil
  there is the bug that leaves a non-chaining conduit firing no charge at all.
- ⚠️ **`TestADetonateIsWorthLessThanItsBreakEven` cannot price a counter and skips
  one by category.** Its arithmetic is what leaving the status alone was worth, and
  a counter is worth neither ticks nor a defence term. The skip is on
  `kind.Category == status.Charge`, **not** on the shape of the payment.
  `TestAConduitPaysForWhatItDischarges` bounds it instead: the arc must beat the
  damped power and not beat it twice over.
- The playstyle is held by `TestAccumulatingIsAWayOfFightingRatherThanASlowerOne`
  — the damage must arrive in **more, smaller** pieces *and* at a rate within
  three fifths of the burst kit's. Shipped reading: 363‰ over 11398 blows of 97
  against 366‰ over 3114 of 366. Four skills charge (thunder_shock 1, thunderbolt
  2 as riders on damage; charge_beam 2, magnetise 3 as turns spent) and two spend
  — a conduit that bought every stack with a turn could never fire twice running.

## Open work

Detail and the open questions are in `README.md` under Roadmap. What matters here
is the constraint each piece has to respect.

- [x] **An evolution line that forks. Done** — a placement chooses **which path**
      as well as how far. `progression.Stage.After` names the stage a stage grows
      out of, so the line stops being an ordered list and becomes a **tree**; two
      arms may share a threshold, and a stage may sit past a fork on one arm only.
      ⚠️ **A line is read by order *or* by name, never both.** A line where no
      stage names an `after` is read by order — stage `i` grows out of `i-1`,
      which is what every line meant before this and why no shipped file moved.
      The moment any stage names one, every stage but the root has to, and each
      must name a predecessor **declared before it** (which is what makes a cycle
      unwritable rather than something to detect). Mixing is refused: the order of
      a file deciding parentage in a file that also states it is a wrong stat line
      rather than an error.
      ⚠️ **`Furthest` was the large half and its failure mode was silence, so it
      refuses now.** `Line.Furthest(level)` returns the tip of **every** arm;
      `StageAt` is the single-answer wrapper and errors — naming both arms — when
      there is more than one. Every caller that passes `progression.Furthest`
      already had an error path, so the whole change is compile-clean: a browser,
      a placement and a balance harness each get a refusal where they would have
      got whichever arm the file happened to list last.
      ⚠️ **`hexforge check` prices one row per arm.** The budget bites at the
      grown end of a line and a forking character has two, so `Library.Inspect`
      loops `Character.FurthestAt(LevelCap)` — art on the first row only, since
      art belongs to the character rather than to an arm.
      ⚠️ **`StageSummary` draws arms bracketed** — `Eevee@1 → (Vaporeon@32 |
      Jolteon@32)` — because joining them with the same arrow reads as a chain.
      `i18n.Lang.StageSummary` delegates to `forge.StageSummary` now instead of
      repeating the shape; the two were byte-identical.
      **Nothing shipped forks yet**, deliberately: the mechanism lands without a
      balance move, the way crit did. An Eevee is content and its own decision.
      ⚠️ **Not the same thing as the tailed-beast Naruto**, which is a separate
      *character* beside Naruto: a stage is the same unit later, and that is a
      different unit. The two want completely different mechanisms.
- [ ] **Graphical client with ebiten.** A renderer over `[]Event`, nothing more.
      It must not read `*Battle`, and it must not need the engine to know how long
      an animation takes. Asset pipeline is undecided: SVG has to be baked to PNG
      at build time or rasterised at load, because ebiten draws neither.
- [ ] **`reckless` is the dragon build's 22.1%.** The roadmap said the gap was the
      missing detonate. It was tested: the line was given one (`dragon_drive`, off
      the `expose` its own `dragon_claw` applies, and it does fire — 19 amplified
      casts in 60 battles), and fielding it reads **21.2%** against 22.0%, which is
      nothing and slightly the wrong way. ⚠️ **Measured one change at a time over
      3000 battles**: the detonate **−0.8**, `reckless → blood_thirst` **+33.1**,
      `reckless → blaze` **+16.9**, fire losing *its* detonate **+10.9**. The trait
      grants `unleashed` **and** `bare` — 30% of attack for 40% of defence *and*
      40% of dodge — into a build whose opponent amplifies its heaviest skill three
      and a half times off a status.
      ⚠️ **`TestRecklessIsATradeAndNotAGift` cannot see this.** It asks whether
      something is given up and passes; whether *too much* is given up is a
      different question, and the same shape of gap a win rate had against
      `swiftness`.
      ⚠️ **Do not just swap the build's trait.** `blood_thirst` beats `reckless` in
      the mirror **and** against the rest of the cast (100%/100% against 98.6%/96.0%),
      so the swap is a power increase rather than a rebalance. Softening the cost is
      the other lever and it is uncoupled — `bare` is granted by `reckless` and by
      nothing else, and no skill applies it — but there is **no test holding its
      magnitude**, so whatever it becomes has to be measured and written down the
      same way. Decide what the trait is *for* first.
      ⚠️ **Dropping `bare`'s dodge clause was tried and it is not the lever.** The
      argument was good — the clause is what makes the trait two-stats-for-one, and
      dodge gates whether an attack connects at all, so removing it should compound
      against a burn-and-detonate opponent. Measured on the same instrument, same
      3000 battles both ways round: **R0 shipped 22.0% · R1 dodge dropped 24.8%
      (+2.8) · R2 vulnerability alone 16.1% (−5.9) · R3 both 18.7% (−3.3)**, against
      referents re-taken on the same run of **A `blood_thirst` 55.1%** and **B
      `blaze` 38.9%**. R3 is below where the trait started and far below B, so the
      candidate was **not shipped**. The interaction is clean — `R3−R0 = −3.3`
      against `(R1−R0)+(R2−R0) = −3.1` — so the two terms **add**, and the
      superlinearity the lever was chosen for does not exist.
      ⚠️ **`bare`'s cost is 88% defence and 12% dodge**, which is why. Decomposed:
      dodge-only reads **43.4%**, no `bare` at all **46.3%**, and those add too
      (2.8 + 21.4 = 24.2 against 24.3). So the *only* lever that can move the figure
      is softening the defence magnitude — the one with no natural stopping point,
      and the one this item has always been reluctant to pull. Dropping `bare`
      outright lands at 46.3%, inside (B, A) and on the 45–50% target, but that is a
      trait with no cost and `TestRecklessIsATradeAndNotAGift` refuses it.
      ⚠️ **Half the missing guard now exists.**
      `TestRecklessSpendsNoMoreThanItBuys` prices the trait in **damage off the
      event log** rather than in wins — a win rate cannot price a stat, which is the
      `swiftness` finding — and asserts the trait buys damage and spends no more
      than twice what it buys (a declared design constant borrowed from the detonate
      rule, the only invented number in it). Shipped data reads 30956 bought for
      50084 spent, ratio 1.62. It is what caught R3: `bought` went **negative**
      (−7898) where the 18.7% win rate sat comfortably inside every existing band.
      The other half — a cast-wide *no trait may lower more distinct stats than it
      raises* (`ballast` keeps it non-vacuous; necessary and **not** sufficient,
      since it would pass a `bare` at −900) — is now **dropped rather than held**.
      No fix landed for it to land with, and the sweep below is the reason it is
      the wrong test: it counts stats and never magnitudes, so it would pass a
      `bare` at −900 and refuse a `bare` at −25, and what turned out to be worth
      holding is exactly the magnitude. The ledger already holds that, in the
      currency the trait is denominated in.
      ⚠️ **The third lever is dead too, and all three are now measured.** Softening
      `bare`'s defence magnitude was swept on the same instrument — dodge left at
      −400, `unleashed` untouched, 3000 battles a row, referents re-taken on the
      same run and identical (**A `blood_thirst` 55.1%**, **B `blaze` 38.9%**,
      **R0 22.0%**). The four amounts the item prescribed **all sit under the
      floor**: −300 **24.4%** · −250 **24.7%** · −200 **27.8%** · −150 **37.0%**.
      Swept finer: −125/−100/−95 **37.4%**, −90/−85/−80/−75 **39.9%**, −50
      **41.2%**, −25 **42.9%**, and a term of 0 is refused by the parser.
      ⚠️ **The dial is a quarter as long as it reads, because the stat saturates.**
      `modifier.Set.Stat` saturates a change towards a floor rather than applying
      it, so a −400‰ term on a base of 400 fights at **290**, not 240, and the whole
      reachable range of the lever is **290..391**. The rate also moves in *steps*:
      fourteen amounts give nine rates, with plateaus (−90..−75 are all exactly
      1197/3000) and cliffs, because a two-point defence change is worth nothing
      until it moves a strike across a kill threshold. **A dial like this cannot be
      tuned to a target** — the nearest rung to the 45–50% wanted is 42.9%, at a
      cost of nine points of a stat.
      ⚠️ **Both gates flip at the same rung, which is why nothing shipped.** The
      duel clears **B** between −95 (37.4%) and −90 (39.9%); the cast-wide pair
      saturates squirtle at **100%** between −95 (99.0%/98.6%) and −90
      (99.0%/**100.0%**). Same two points of defence, 366 → 368, both directions at
      once — because both are one event, a strike crossing a kill threshold. So
      **every amount that makes the dragon build a real matchup also makes
      `reckless` the 100%-against-the-cast trait `blood_thirst` was refused for
      being**, and there is no value that passes both. Reported and **not shipped**;
      the data is unchanged and the floor in
      `TestTheDragonBuildIsASidegradeAndNotAnUpgrade` therefore **stays at 150**,
      since the duel still reads 22.1%. What the trait needs next is a *different
      kind* of cost — one the duel prices and the cast-wide matchups do not — or the
      acceptance that the 22% is a statement about `inferno` and belongs to the fire
      line's detonate. All three of `reckless`'s own dials are spent.
      ⚠️ **A `weigh`-shaped instrument for a trait is the tool that is missing**,
      and it is its own piece of work. Everything above was measured by editing
      shipped JSON, running a duel, and putting it back: there is no way to sweep a
      trait's field the way `weigh` sweeps a skill's, so every reading costs a
      hand-managed data mutation and cannot be reproduced from the repo. A trait
      field table would have priced all four readings and both decompositions in one
      pass. The magnitude sweep found a third of it: patch the parsed
      `statuses.json` **in memory** and rebuild the status, passive and skill books
      around it, which is exact and reproducible and needs no file put back. What is
      still missing is somewhere for the answer to live that is not a test log.
      → `TODO.md` for the `weigh` field-coverage item this joins.
      → `README.md` § *What the dragon line's detonate was worth*, § *What
      `bare`'s dodge clause was worth* and § *What softening `bare`'s defence was
      worth*.
- [x] **A deeper opponent. Done** — see *Rating an action* above for the rules and
      *A deeper opponent* in `README.md` for what moved. Statuses, buffs, guards,
      heals, cleanses and kills are all priced in damage now, over capped horizons,
      and the detonate setup came free with pricing the status. **Tempo followed**
      and is priced too — off the speed stat, so nothing reads the queue; see
      *Rating an action* for why a turn is worth `turnWorth` and not the best
      strike. **All-sided skills are rated too** (`friendlyFire`, the own half
      subtracted), and *holding a skill for a later turn* is answered as far as a
      one-turn-deep rating honestly can: a **tie-break on cooldown**, so a scarce
      skill is not spent on what a common one buys. **Waiting is no longer "still
      out" — it is decided against**: a pass buys no cooldown an act does not, so
      acting dominates by exactly what the action is worth, and the two available
      lookaheads each break a rule of the file. See *Rating an action* above for
      the arithmetic. What is left is *where* in the order an extra turn falls,
      the only part that would need the queue.
      ⚠️ It cost a **balance answer, not a golden**, exactly as the summon did:
      the shipped roster went 53.1% → 79.0% ally, which is a cast finding.
      **Re-levelled since** (Charmeleon 30 / Ivysaur 30, 49.1%, and 49.4% once
      tempo was priced), so every rate quoted anywhere is on the same instrument.
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
- [x] **Two builds for one Bulbasaur — SHIPPED** (#114), as `bulbasaur.poison`
      ("rải độc") and `bulbasaur.parasite` ("ký sinh") in `builds.json`, measured
      by `TestTheTwoBulbasaurBuildsAreDifferentUnits` before they were listed.
      Two builds a character is the target and every Pokémon now has its pair;
      **Naruto is the one with none**, which the catalogue treats as the honest
      case rather than a gap to fill with a duplicate.
      ⚠️ **The build this entry originally described could not be built.** It
      asked build (1) to be *immune to poison* **and** *sharper* **and** *biting
      back* — but immunity and the reply are both `venom_blood` while the
      amplifier is `virulence`, and `TraitSlots = 1`. Two traits, one slot. What
      ships is the amplifier, so the poison build **hits harder and is not
      immune**; `venom_blood` is the reserve entry of that direction, the way
      `last_gasp` is of the other, and `TestBulbasaurCanBeBuiltTwoWays` checks
      only that neither direction loses its **last** trait.
      So `Resists` has a mechanism (✅, `heatproof` and `venom_blood`, `amount:
      1000` is total immunity) and **no shipped build carries one** — a third
      build is where that would go, not a second trait slot.
      Pieces, all ✅: `Resists` · *amplifying a status* (`virulence`) ·
      *answering back* (`venom_blood`) · **passive lifesteal** (`blood_thirst`,
      `last_gasp`) · `While` (`blaze` is gated).
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
      ⚠️ **A reply drains too — and did not, until it was looked for.**
      `resolveAgainst` paid out and `reply` did not, so a trait holding both jobs
      promised a share of an answer it never gave. What said so was the
      description: *"mọi đòn của nó hút lại…"* / *"everything it does takes
      back…"*. `TraitSlots = 1` stops a unit carrying a replier **and** a drainer;
      it stops nothing about a trait that is **both**, and `passive.Passive` holds
      both fields. Nothing shipped is both, so **no golden moved** — the same
      shape as the regeneration bug, a job rendering in the sentences and not in
      the engine.
      The trait's share only (a reply has no skill), and **before the kill**:
      `resolveAgainst` drains whether or not the target fell, so draining after
      the return would make lethal damage the one blow worth nothing.
      ⚠️ **A `damage > 0` guard beside it was written and removed** — a mutation
      deleting it survived every test, because the branch it sits in has already
      said the reply has power and `drain` refuses a share of nothing anyway. The
      skill path's `dealt > 0` is **not** the same guard: there the sum can be
      nought after every strike missed.
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
      ⚠️ **`venom_blood` is gated at 24 now, and the roster figure moved with
      it.** It was the one trait on the learnset with no level, so a Bulbasaur
      had the strongest of the five from level 1. **4000 seeds: 49.5% → 53.2%
      ally.** Every other measurement in this file reading 49.5% predates it.
      The swing is one-sided by construction: `ally.venusaur` is level 60 and
      keeps the trait, while `foe.ivysaur` is 16 and loses it — a placement
      naming a trait above its level is a **hard parse error**
      (`chooseFrom`: "has not learned at level 16"), not a silent drop, so the
      roster had to change in the same commit or stop loading.
      ⚠️ **What replaced it on `foe.ivysaur` is worth almost nothing**: measured
      both ways, `endurance` gives 52.9% against 53.2% for no trait at all. The
      3.7 points are the loss of `venom_blood`, not the gap it left, so the
      choice between them is a design one — it fields none, because handing it
      another trait would swap one for another rather than show the gate, and
      `ally.charmander` already fields none.
      ⚠️ **`virulence` on `ally.venusaur` was measured and rejected: 56.3%.** It
      is the *stronger* trait of the two at the cap, so swapping the ally's
      build to compensate pushes the figure further out, not back.
- [x] **Naruto's three forms renamed to the three the story has.**
      `Naruto@1 → Shippuden@16 → Sennin@32`, against `naruto.svg`,
      `naruto-shippuden.svg` and `naruto-sage-mode.svg`: before the two years of
      training, after them, and after learning the sage art.
      ⚠️ **The count was right and both names were one form ahead of their own
      picture** — the middle was called *Tiên nhân* while showing Shippuden art,
      the last *Vĩ thú hoá* while showing sage-mode art. The art was right the
      whole way down; the labels were off by one.
      ⚠️ **The third form was authored `Tiên nhân` and is `Sennin` now, because a
      stage name is a KEY and may not be a translation of one.** `Line.Resolve`
      looks a stage up with `candidate.Name == stage`, `Stage.After` names a
      predecessor by name, a placement writes `"stage": "Ivysaur"` and a learnset
      gate lists stage names — four hand-typed spellings of one string — and
      `Line.Validate` refuses two stages sharing a name, which is a uniqueness
      constraint only a key has. *Tiên nhân* is the Vietnamese **translation** of
      仙人, so the romaji was the name that was missing rather than a new
      invention: it keeps the meaning, matches `Shippuden`'s convention exactly,
      and is right in both languages. **A stage name is still drawn raw** at
      `browse.go`, `preview.go`, `squads.go` and through `unit.Name` in
      `play.go` — that is the house rule for an id and there is deliberately **no
      `Lang.StageName`**; what was wrong was the data. `progression.ValidateStageName`
      is the refusal, at the parser rather than over the shipped data so it binds
      every line anybody writes and an authoring form can reject a name as it is
      typed — the reason `cast.ValidateID` is exported. Printable ASCII, at least
      one letter, no edge or doubled space: `Tiên` has two Unicode encodings that
      draw identically and `==` calls them different names, so a non-ASCII key
      silently misses. ⚠️ `TestTheScreensGlossEveryDataName` **still collects no
      stage name and should not** — it asserts that an id is shown *with its
      gloss*, and a stage name has none by decision; the parser is where the rule
      belongs.
      The tailed-beast form was never a stage: **a stage is the same unit later,
      and that is a different unit**, so it becomes its own character later.
      ⚠️ **No stat moved, nothing rebalanced** — a stage name is printed, never
      read, so every measurement taken against this line still holds.
      Naruto has **no build** in `builds.json` and now has two traits, so it is
      the character the "two builds each" target is waiting on.
- [x] **A permanent speed trait — `swiftness` on Naruto.** Grants `quickened`,
      permanent, **+80** speed. Naruto's second trait, so the only character with
      a default now has a choice.
      ⚠️ **The house figure for a permanent buff is 150** (`toughened`, `kindled`,
      `unleashed`) **and it does not transfer to speed** — a point of speed is
      worth more than a point of anything else, because speed is turns and a turn
      is every other stat applied again. Priced in the **share of the turn order**
      it buys: `+30` 2.6% · `+50` 4.4% · **`+80` 7.9%** · `+100` 9.9% · `+150`
      14.8% more turns than `endurance` gets in the same battles.
      ⚠️ **A win rate does not measure this and a band over one is a trap.** Over
      300 mirror duels the rate does not even *order* the amounts — `+150` comes
      back at 59.0% while `+50` reads 74.0% — because the turn queue is discrete,
      so a few points of speed buy whether one more turn lands before the other
      unit acts, and that is lumpy per seed. `Suggest` casting no-power skills put
      a summon in the queue too and made the lumps larger. **At 150 the win-rate
      band passes and the turn-share band fails**, which is how it was caught.
      ⚠️ The first turn test compared **two separate sweeps' totals**, which
      measures battle *length* — and the faster unit ends its battles sooner. It
      passed until `Suggest` changed, then reported swiftness with **fewer** turns
      while being exactly as fast. Both sides of **one** battle is the comparison
      that cannot say that.
      ⚠️ Naming: `haste` already glosses "nhanh nhẹn" so `quickened` is "gia
      tốc" (it read "thoăn thoắt" first); the trait was "nhanh chân" until
      **`chân` turned out to be a `bodyWord`**, and is "thần tốc".
- [x] **The budget bounds a line nobody fights on**, and `hexforge check` now
      says so: a second table of the line each character actually fights on, per
      trait, via `forge.Library.Held`.
      ⚠️ `progression.Limits.CheckValues` takes **six numbers and nothing else**,
      so the bound is on the paper line. A trait is named on a *placement* and its
      grants go on at enlistment — **`battle.New` rejects a base line of 740
      defence and then hands the same unit 786 through a trait, in the same
      call.**
      ⚠️ Only **permanent** grants count, which is all a trait can give
      (`status.Set.Hold` refuses a timed one). A gated trait (`blaze`) is skipped:
      its condition reads a health no character has outside a battle.
      ⚠️ **SETTLED: the bound is the PAPER line's.** A ceiling and the budget
      bound what an **author** writes at the cap; going past them in a battle is
      the point, not a leak — for a buff, a trait, and whatever a rune becomes.
      What holds the **fought** line is the **saturation**: `ceiling × headroom`,
      so nothing reaches 3× a ceiling however much is stacked.
      So this prints a figure and raises nothing. `TestNoTraitCarriesACharacterFarPastTheBudget`
      is a **tripwire not a bound** — 120%, shipped worst Squirtle/`ballast` 113.4%.
      Found the moment the table existed: **Bulbasaur/`endurance` has 157 left**;
      `reckless` is *under* the bound (`bare` costs more than `unleashed` buys).
      ⚠️ Three existing guards caught the first wording: the TUI's 79-cell width
      test, the renamed-label ban (**"absorbs" is banned in English**), and the
      short-reach test's `len(Warnings) != 0`.
- [x] **Speed cannot reach nought, and now something says so.** Four guards stood
      between the data and a unit that never acts again — the floor at a tenth of
      base (approached, never reached), `Stat`'s return of 1, `atb.Wait`'s clamp,
      and `atb.Queue.Add`/`Reschedule`'s — and none of them stated the invariant.
      `TestNoShippedDebuffCanFreezeAUnit` stacks every harmful shipped status 50
      deep on every character and asks whether the queue still turns.
      ⚠️ **`max_stacks` binds long before the floor.** `expose` caps at 2, so
      Squirtle's defence bottoms at **410 of 640** — the floor is 64, ~6× further.
      To strip armour harder the levers are the amount and `max_stacks`, **not the
      floor**, or `pierce` (the counter armour was given).
      ⚠️ **`TestStatNeverDropsBelowOne` was passing for the wrong reason** — it
      crushed a base of 3, where the saturation lands above nought by arithmetic
      and the branch is dead; deleting the branch left it green. **Only a base of
      nought reaches the guard** (Saturate gets a gap of nought and returns the
      base), which is real: a summon is authored with a fixed line and nought
      dodge is ordinary.
- [x] **Two Squirtle builds out of one learnset**, in data only — three skills
      (`skull_bash`, `wide_guard`, `water_pulse`), one trait (`ballast`), two
      permanent statuses (`fortified`, `encumber`), no engine change at all.
      A placement spends 4 skill slots and 1 trait slot, so two kits out of one
      learnset is what that system is for.
      ⚠️ **The stat line cannot differ**: Squirtle absorbs 11285 of the 11500
      budget. The split has to live in the slots because it cannot live anywhere
      else.
      `skull_bash` is the **first shipped skill to scale off anything but
      attack** — `skill.Scaling` had no shipped user until now. def 640 / atk 460
      = **1.39×** a point of power, so a defence-scaled skill wants ~0.72× the
      power of an attack-scaled one.
      ⚠️ **`ballast` is the attacking build's trait and not the tank's**, which
      is measured, not intended: survival is gated on how often `withdraw` can be
      cast, so **a tenth off speed is worth more than a quarter onto defence** —
      the tank build survives 1/30 with `ballast` against 29/30 with `endurance`.
      ⚠️ **A test may not raise a stat**: `battle.New` checks the roster line
      against the budget. `skull_bash`'s scaling is proved by *halving attack*
      (figure comes back **exactly equal**, while `water_gun` halves). A stat
      falling and damage falling with it proves nothing — the unit also dies
      sooner and swings fewer times.
      ⚠️ **A trait's permanent statuses are outside the budget** — `CheckValues`
      takes a resolved stat line and never sees a passive. `endurance` was
      already through that gap.
      ⚠️ Tank + `endurance` is **unkillable in a duel** (29/30 hit the 4000-turn
      cap, dealing nothing); a squad puts five times the damage into it.
      ⚠️ Both build figures are understatements: `Suggest` takes a no-power skill
      only when it can find nothing to hit, so the tank kit is exercised only
      because it carries no weapon. `hexforge spar` cannot measure either build.
- [x] **A summon says how strong it arrives.** `describeSummon` prints the
      **share** and still not the fixed stat line, because they are not the same
      kind of number: a share is one figure that means the same thing wherever it
      is read, and a fixed line is six nobody can compare without the caster.
      ⚠️ It was left out because "the listing beside this carries it" — **no
      listing does.** Neither `hexforge` nor `hexforge-tui` mentions a summon at
      all, so the sentence is the only place one is described.
      ⚠️ **`share` and `share_of_base` must read differently** — one rewards
      buffing before the cast and one ignores every buff — and **one copy and
      several need different wordings**, since a pair handed the singular reads as
      a pair carrying 40% between them. Comparing the two descriptions does not
      catch that: they differ in their count either way, so the test asks about
      the share clause by splitting the wording on its blanks.
      ⚠️ **A creature is not a copy.** English falls back to a word, and the word
      was "copy" for everything: the shipped toad read as a copy of the ninja who
      called it. Told apart by the stat spelling, which is the line the engine
      already draws.
      Data rules that came out of it: a summon's flavour may claim nothing about
      its caster (`casterWords` — both shipped Naruto summons did, and the toad's
      "to hơn cả người gọi" is contradicted by its own stat line), and a
      one-strike flavour may describe no volley (`volleyWords` — `kunai` said "một
      nhúm", which is the count said twice **and** the wrong weapon, since a
      handful thrown at once is the shuriken beside it; renamed `phi đao`).
      `chùm` is off that list on the judgement that kept teeth out of `bodyWords`.
- [x] **A taunt: the choice of enemy taken away, not the turn.** Status
      `taunting`, and `Battle.aims` narrows an **enemy-aimed** skill to whoever is
      taunting. `Suggest` obeys it **with no AI change at all**, because it reads
      the aims it is offered and nothing else.
      ⚠️ **The status sits on the TAUNTER, not on the taunted.** A taunt held by
      its victim would have to remember *who* taunted it, and a `status.Stack`
      **deliberately does not remember who applied it** (that is what keeps a
      stack worth the same after its author has died). Held by the taunter it
      needs no memory: "who must I attack" is read off the board, and a corpse is
      not on it — `TestATauntDiesWithItsTaunter` needs no cleanup path.
      ⚠️ **Range is not read.** Nothing on this board moves, so a taunt that could
      be answered by standing far enough away would be ignored by exactly the
      long-ranged attackers a tank most needs to pull. A range-1 skill aimed at a
      taunter four cells away lands: nothing past the *legality* of the aim has
      ever read distance.
      ⚠️ **A new category `Taunt`, not a second `Control`.** Stun means "you do not
      act"; taunt means "you act, and may not pick" — opposite things to a turn. A
      category exists so a cleanse can name a class, and "strips a control" taking
      a taunt off with a stun is a cleanse nobody could aim. It also stopped the
      reference printing *"the holder loses its turn"* under `taunting`, which the
      shared category had it saying. Declared **last** (serialises by name, but
      `CategoryCount` and the grouped listing order are declaration order).
      ⚠️ **`Category.Harmful()` must include it** — that gates what a trait may
      **resist**, so a taunt outside it makes "cannot be provoked" unwritable.
      Nothing else noticed: the mutation passed every other test in the repo.
      ⚠️ **A taunt is spent on the TAUNTER's own turns**, so a fast taunter wastes
      it before a slow victim ever acts. The slowest unit in a squad makes the best
      taunter — which is Squirtle (speed 85, slowest in the cast) exactly.
      A taunt only ever *unfreezes* a stalemate (it adds an aim for a unit that had
      none), so `frozen()` needs no change; it already goes through `aims`.
      Shipped `taunt` on Squirtle@40 (self-aimed, cd 3).
- [x] **A reply priced off the stat its holder actually has.** `passive.Reply`
      gained **`Scaling`** (a `skill.Scaling`, so stat *and* base-or-current),
      authored as `"replies": {"power": 80, "scaling": {"stat": "defense"}}`.
      ⚠️ **Attack was the wrong default, not a missing field.** A trait that
      answers whoever hit it belongs to a unit **built to be hit** — armoured, not
      sharp — so pricing every reply off attack made thorns worth least to exactly
      the character thorns are for. Blastoise is 640 defence / 460 attack: the
      same share off the wrong stat is a **third** less.
      ⚠️ **`origin.Scaling` is now the whole `skill.Scaling`, not a bare Kind**,
      and all three read sites go through `origin.stat` → `combat.PickScaling`.
      That fixed a latent bug nobody could reach: a skill declaring
      `"source":"base"` had its damage read the base line and its **DoT tick read
      the current one**. No shipped skill declares scaling, so it was unreachable —
      and would have arrived the day one did.
      ⚠️ **`skill.ParseScaling` is exported** so a trait and a skill are read by one
      parser. Health stays refused wherever it is asked for (damage that grows as
      its owner is healed). Refused too: a reply with **no damage** naming a stat.
      ⚠️ **The word "attack" was hardcoded in THREE places** — the engine, the
      i18n reply blurb, and `hexforge passives`. The listing's is in no golden and
      nothing else reads it: the mutation that put the literal back **passed the
      whole suite**, and an author would have tuned a thorns trait against a
      number a third out. `TestThePassiveListingNamesTheStatAReplyIsPricedOff`.
      Shipped `thorns` (8% defence) on Squirtle@32. Measured in a duel — thorns'
      best case, since the holder is attacked every turn it lives: **14.2% →
      15.3%** overall, Charmander matchup 28.5% → 30.7%. ⚠️ **Squirtle still loses
      to Bulbasaur 0% either way**, which is a cast problem this does not touch.
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
      ⚠️ **The counter-damage was later sold to buy chance, and `venom_blood`
      answers with poison alone now** (`{"applies":[{"status":"poison",
      "chance":40}]}`, no `power`). 20 000 seeds: **25‰+power 40 → 53.0%**,
      **40‰+power 0 → 53.1%** — the same cost, and the poison lands **0.27 →
      0.44 per battle**, 63% more often. 50‰ alone was measured at **56.1%** and
      refused; 50‰+power 0 at 54.3%. ⚠️ **σ ≈ 0.35 points at 20 000 seeds**, so a
      gap under **0.7** is noise and the 4 000-seed sweeps used earlier cannot
      resolve one.
      ⚠️ **No shipped trait answers with damage any more** — `venom_blood` was
      the only replier in the cast, so a battle from the shipped roster emits no
      `Damaged` carrying a trait. `TestTheShippedRosterAnswersItsAttackers` was
      narrowed to the status half and logs the damage count instead of asserting
      it; the mechanism stays covered by `internal/core/battle/reply_test.go`,
      whose fixtures exist for it. The bench roster never covered it at all
      (`endurance` and `blaze` only).
      ⚠️ **It flipped the sign of a figure already in the README.** Removing
      `razor_leaf`'s pierce used to cost the ally 2.7pp; it now gains 1.1pp
      (51.9 → 53.0), at every reply size tried. Piercing helps whoever attacks,
      and this is the first thing that charges for attacking — so **no balance
      figure measured before this feature carries forward**.
- [x] **A condition a skill reads about ITSELF.** `skill.Skill.SelfRequires`, the
      same `*Condition` as `Requires` asked of the caster instead of the target —
      `Applies`/`SelfApplies` spelled again, two fields rather than one field with
      a "whose" flag. Until it existed, "hits harder while I am furied" and "hits
      harder while I am cornered" had **no spelling at all**.
      ⚠️ **Read ONCE PER USE, in `Act`, not in `resolveAgainst`** — that runs once
      per cell a shape covers, so a consumed condition would charge a column three
      times and a single-target skill once, for a difference written on neither
      skill. `Battle.spend` is that seam, and it sits **before `applyToSelf`** so a
      skill that grants and spends the same status cannot pay itself.
      ⚠️ The bonus is added **before** the splash share is taken, like the target's.
      A bonus added after still makes every target take more, which is why that
      mutation survived the first draft of the test — assert the **ratio between
      the aim and the edge**, not the rise.
      ⚠️ **`conditionCaster` is a second builder beside `conditionTarget`**, not a
      parameter on one: a skill may read one status of its target and another of
      itself, so a single builder would have to be *told* which condition it was
      reading — the exact reading-vs-resolution mismatch `conditionTarget` exists
      to prevent. `Suggest` reads it once outside its own loop, where it is read
      for real.
      ⚠️ **`resolveCondition` is now one validator for both fields**, carrying the
      field name through every message: two copies would be two sets of rules and
      the looser is the one an author finds. `TestBothConditionsAreRefusedTheSameWay`
      runs one table through both. Also refused: a bonus power on a **self-aimed**
      skill, which deals no damage for it to land on.
      ⚠️ **The reference table read only `Requires`** and would have told an author
      their skill has no amplifier while the engine amplified it — `skills.golden`
      gained a **whose** column. Shipped on `outrage`: 2200 → **3400 below 40%
      health**, a gate that fires under autopilot where a `fury` payoff never
      would, and one that pairs with the frailty `reckless` buys. Dragon-vs-fire
      unmoved at **42.5%**.
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
      identically, else the AI prefers a bonus it does not get. The **gradient** —
      smoothly harder the further the *caster* fell, rather than a threshold — is
      the separate feature below; `SelfRequires` stays the threshold version.
- [x] **A damage gradient off the caster's own health. Done** — `self_gradient`
      with one number, `at_empty`, the share it adds at the bottom of the bar.
      `combat.Gradient` is the arithmetic; `comeback` is the first user (900 power,
      1710 with nothing left). ⚠️ **A multiplier, not a bonus, which is why it is
      in `combat` and not a fourth field on `Condition`**: a bonus would have to be
      added to the *declared* power, which is not what a skill lands at once a
      detonate has amplified it; a share scales whatever power the skill arrived at,
      so the two compose instead of arguing about order. A condition could not
      express it at all — a condition answers yes or no.
      ⚠️ **`combat.Gradient` returns the share ADDED, not the multiplier**, so
      nought means "nothing happened" in the struct, in the log and in the tables —
      the shape `Pierce`/`Refused`/`Drained` already have, and the reason every log
      written before this is byte for byte what it was. `swing.applied` adds
      `PermilleBase`, once.
      ⚠️ **Read once per USE, and here that seam has teeth.** `Battle.spend` records
      the rule for a *cost*; a gradient has no cost to pay twice, so the rule looked
      like tidiness. It is not: **a draining skill heals its own caster inside the
      loop that walks a shape**, so a per-cell reading gives the second unit in a
      column a softer swing than the first, written on no skill.
      `TestTheGradientIsReadOncePerUseAndNotOncePerTarget` catches it as a **ratio** —
      edge and middle differ by design, but being hurt must multiply both the same,
      and a re-read hands the edge ~1180 per mille where the middle got 1500.
      ⚠️ **`swing{Bonus, Share}` replaced the bare `spent int`** that
      `resolveAgainst` and `against` both took. Two adjacent ints in every
      signature: handing the bonus where the share goes compiles, divides the power
      by a thousand, and reads as a balance change. `swingOf` is the single reading,
      for the reason `conditionTarget` is.
      **The share is on `skill_used`.** `Power` there is what the skill *declares*,
      so without it a hurt caster's strike lands for more than the log states with
      nothing to bridge them — worse than a pierce, which is at least the same on
      every cast.
      **One new refusal that is not arithmetic:** a gradient beside a
      `self_requires` reading **health**. Two curves off one number is unpriceable
      and unreadable; a threshold on a *status* composes fine, which is why the rule
      asks what the condition reads, not whether there is one. **No upper bound**,
      unlike `pierce` — piercing past all the armour is meaningless, a share added
      to power has no such ceiling.
      ⚠️ **A mirror duel that swaps SIDES rather than KITS measures itself.** The
      queue breaks a tie by enlistment, so leaving the roster order alone enlists
      the first-written kit first in *both* halves: a unit against an identical copy
      of itself read **58.8%**. Swap the kits.
      `TestTheMirrorIsFairBeforeAnythingIsMeasuredThroughIt` demands exactly even
      before anything else in that file is believed. ⚠️ And the swap that
      discriminates is against **rasengan**, not against the kit's filler — dropping
      `kunai` wins ~75% at *every* power from 500 to 1100, because a 700-power
      cooldown-0 skill makes the fourth slot nearly free.
      ⚠️ **The "blocks the form does not ask about" list was wrong in all three
      places it was written down** (`internal/screen/skills.go`,
      `internal/forge/skills.go`, `TestTheShippedSkillBookSurvivesBeingWritten`):
      each named `self_applies`, which the form *does* ask, and none named
      `self_requires` or `summons`, which it does not. Corrected, with
      `self_gradient` joining them.
      ⚠️ **`forge.PreviewDamage` used to read only the target's condition**, so
      neither `self_requires` nor `self_gradient` showed in the authoring preview.
      Fixed since — one change covered both, and the composition moved into
      `combat.Swung` so the preview and the battle share the expression.
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
      **Conditions beyond a level stay out of scope**: items, a friendship count,
      battles fought — each needs somewhere to persist between battles, and there is
      no meta layer, no inventory and no save. A level is what a character sheet knows.
- [x] **Looking a status up — SHIPPED.** `Lang.DescribeStatus` beside
      `Describe`/`DescribePassive`, derived, both languages, and three front-ends
      read the same sentences: `?mire` / `?*` at the battle prompt, `hexforge
      statuses` (the sixth listing), and `screenStatuses` in the tool.
      ⚠️ **A life, not a tick, and one stack's life — not the ramp.** Poison ticks
      50% to burn's 80%, so the tick alone ranks them backwards; over their lives
      it is 150/160 for one stack and **450/320** at their caps. `skills.golden`
      prints a *third* figure under the same words — the full ramp of a status
      reapplied every turn (600% for poison) — which needs a skill off cooldown
      every turn and no shipped kit has one. Two numbers, one phrase: the golden's
      is an author's ceiling, the reference's assumes nothing.
      ⚠️ **Permanent is "always"**, and a one-stack status gets **no rate**:
      `toughened` reads *tăng thủ 15%*, not *15% mỗi lớp*
      (`TestAPermanentStatusIsNeverGivenARate`).
      ⚠️ **Grouping lives in core** — `status.Book.Grouped` — because three
      front-ends working it out is three answers to "which category is this in".
      In the tool the headings are **rows**, since the listing scrolls and a
      heading drawn between rows falls off the top; the price is a cursor that can
      land on one, which `TestTheStatusCursorNeverLandsOnAHeading` refuses.
      ⚠️ The caveat ("these are the book's figures, an amplifier or a resistance
      changes what lands") is printed **once** per reference and is the **last**
      line, and `frame` cuts from the bottom —
      `TestTheStatusCaveatSurvivesTheSmallestWindow` measures it at 120x24 in both
      languages.
      ⚠️ Adding a screen to `hexforge-tui` means adding it to `everyScreen` in
      `language_test.go`, or every width and translation test silently skips it.
- [x] **Reading a trait — SHIPPED.** `?` on `screenBrowse` raises `screenBlurb`
      for the traits the character under the cursor holds **at the level it is
      walking**, and `hexforge passives` gained the `answers` and `drains` columns
      it never had — two of the six jobs the parser accepts had rendered **nowhere**
      in the tool, so `blood_thirst` printed a row blank after its name.
      ⚠️ **One screen, not two.** Which screen is behind is what `esc` had to
      answer anyway and used to answer with a constant, and it is **not a
      cursor**. A second screen would be a second copy of the framing, the footer
      and the escape.
      ⚠️ **It used to be the single field the screen kept, then it was read only
      by the client, and now it is gone** — the subject it was handed replaced it,
      because reading `m.browse`, `m.skills` and `m.play` is what made this screen
      unmovable, and `model.raisedFrom` replaced the way back once all three
      raisers returned a `draw.Raise`. The describer branches on `subject.Kind`.
      ⚠️ **It scrolls, and `scroll` is still not the refused cursor.** A cursor
      could point at a different character than the browser behind it; an offset
      selects nothing and every key that changes *what* is described resets it.
      Five traits at the cap wrap past 120x24 — the declared floor, not an odd case
      — so the frame would eat the last one.
      ⚠️ **Wrap to `minWidth`, NOT to `m.usableWidth()`** — the opposite of
      `m.wrapped`, which carries authored free text and takes whatever width there
      is. These are the program's own prose: `TestEveryWordingFitsTheMinimumWidth`
      renders at width 200 and measures against the floor less one (79 when that
      line was written, 119 now), and free text is excused while
      a derived sentence is not. Unwrapped, the reply line was cut mid-word at the
      floor ("…3% khả nă").
      ⚠️ **The `answers` column is ONE cell** — `DescribePassive` writes one
      sentence for a whole reply on purpose; a damage cell filed away from a status
      cell leaves a reader adding it up.
      ⚠️ Adding a screen (or a *state* of one) to `hexforge-tui` means adding it to
      `everyScreen` in `language_test.go`, or every width and translation test skips
      it in silence. Both blurb shapes are in it now; `screenPreview` still is not.
      **And the `nội tại` menu**, `screenPassives`: every *declared* trait with the
      description of the one under the cursor, which is the other question — `?`
      on the browser is filtered by a level, so a trait nobody has learned yet is
      reachable from nowhere.
      ⚠️ **The column is "who carries it", NOT "who may".** A trait has no
      restriction mechanism, so *may* is everybody. `Library.TraitCarriers` walks
      the **cast**, not the trait book: the edge lives on the character, and an
      index the other way round is a second place for it.
      ⚠️ **Not a column on `hexforge passives`** — nine columns already, and a
      carrier row is as long as the cast. A clippable row and a cursor are what
      make it affordable; the CLI has neither.
      **And the name in the sentence is a door.** `?` on `screenPassives` opens
      `screenStatuses` at the status the trait names, `esc` comes back, and the
      name is marked (bold, no colour) where it is printed so `?` has something
      visible to be about. `i18n.StatusesNamed(trait)` serves both halves: the ids
      a description will name, in the order it names them. Reading the sentences
      back instead would be substring matching against prose in two languages —
      it styles a name that happens to sit in a flavour clause and misses one the
      glossary lacks, since that one prints as a bare id.
      ⚠️ **It is a second reading of `DescribePassive` and drifts silently.**
      `TestATraitNamesEveryStatusItsDescriptionNames` holds them together; the
      rule the shipped book cannot show is pinned apart — **a reply names its
      first application and no more**, because one sentence has room for one
      status.
      ⚠️ **`blood_thirst` and `last_gasp` name nothing** (a drain names no
      status), so `?` must stay put rather than open whatever the status cursor
      sat on.
      ⚠️ **`statusesScreen.from` must be cleared where it is STORED.** The first
      version returned before the assignment that puts the screen back on the
      model, so a later visit through the menu inherited the earlier visit's way
      back. Caught by a test, not by reading.
      ⚠️ **Marking is ONE left-to-right pass, longest match wins** — a pass per
      name re-marks its own output in either order (`bỏng` inside the `bỏng nặng`
      just produced; or `bỏng nặng` never matching). And each **word** is marked
      whole, not the phrase: the sentences wrap afterwards and the wrap splits on
      spaces, so a style spanning two words breaks when they land on different
      lines.
- [x] **A regeneration that heals — SHIPPED.** `regrowth` was declared, glossed,
      described and **inert**: `inflict` computed a tick only for `status.Dot`, so
      a `Regen` stack went on carrying nought and every step below it was already
      correct and never reached. One branch:
      `tick = b.books.Rules.Restore(b.Stats(actor)[from.Scaling], kind.TickPower)`.
      ⚠️ **`Restore`, not `Damage` — it drops two terms deliberately.** No defence
      (`combat.Rules.Restore` says why: armour turns away what comes *at* a unit,
      so dividing lets a unit's own armour weaken its own regeneration) and no
      elemental multiplier (the chart prices what one creature threw at another; a
      grass unit healing a fire ally throws nothing). Kept: the actor's scaling
      stat and the **freeze** — the promise `status.Regen` already made and
      nothing honoured.
      ⚠️ **The old note misnamed the casualties, and the correction was wrong
      too.** The two skills fixed *here* were **`aqua_ring` and `ingrain`**, and
      neither had a working half: power 0, no `restores`, nothing but the
      regeneration, so casting either did *nothing*. What this note then claimed —
      "`synthesis` was never affected, it heals through `restores`, which always
      worked" — was **false**. A `restores` payout sat inside `resolveAgainst`,
      which `Act` returns before for a `Target: Self` skill, so `synthesis` healed
      nothing either, and `withdraw` paid out its block and dropped its five
      hundred. **Both** halves of the shipped healing were inert at once, by two
      different routes, and the note correcting one of them asserted the other was
      fine. → *Healing is not damage with a sign*.
      ⚠️ **A second bug was hiding behind the first**: `tickStatuses` named the
      status that healed and then healed again from the total, so every tick would
      have logged **two** `Healed` events. Healing is now applied per entry and
      `heal` carries the status id — one event saying what healed, how much landed
      and the health left. Damage stays on the total (`wound` emits nothing, and
      has no name to carry). Per entry is also the truthful arithmetic: `heal`
      stops at full health, so a second regeneration is clamped where one total
      would have hidden it.
      ⚠️ **The two decisions, made.** A regen scales off the applying skill's stat
      read live off the actor — the same expression the Dot branch uses. And an
      amplifier does **not** raise a regen: the refusal in `passive.Amplification`
      stays, but on the other ground, because its old reason ("a multiplication of
      zero") expired with this fix. The share reads *"its poison ticks 30% harder"*
      in both languages, and a share that heals under that sentence is a
      description that lies. Lifting it is a wording change first.
      ⚠️ **No golden moved, and that is the finding.** The plan expected a balance
      diff. **No roster unit fields either skill**, and `Suggest` never chooses a
      self-cast regeneration on a unit that can always reach somebody — those two
      facts together are why a shipped skill did nothing this long with every test
      passing. Proof is hand-played (`TestTheShippedRegenerationHeals` on the
      shipped books) plus eight tests in `internal/core/battle`. Fielding a regen
      is a **balance** decision and belongs with the cast work.
      ⚠️ **One of the eight was worthless as first written.** The order test
      asserted `healed` came before `status_ticked`, and a mutation putting damage
      first **survived**: `wound` emits nothing, so the events come out in the same
      order either way and only the survivor changes. It asserts survival now.
- [x] **A spar: measuring whether a character *belongs*.** `check` says a
      character is legal; nothing said whether it stands beside the ones already
      written. `forge.Library.Spar(id, level, seeds)` duels it against **every**
      character in the book, itself included, and reports a rate per opponent.
      Two front-ends, as always: `hexforge spar` and `s` from the TUI's **check**
      screen (raised there rather than from the browser — the two are halves of one
      question, and the browse footer was already 78 of its 79 cells).
      ⚠️ **A spar rate is NOT the roster's win rate and the two must never be
      compared.** The roster figure (49.5%) is five units a side in their authored
      slots with authored loadouts; a spar is 1v1, front column, an auto-chosen
      four-skill kit and no ally to heal or shield. Bulbasaur reads 50.0% in both
      and that is a coincidence.
      ⚠️ **Every pairing is fought BOTH WAYS, and that is the measurement rather
      than thoroughness.** `atb.Queue.order` breaks a tie by enlistment `seq`, so
      of two units with the same speed the one placed first acts first for the
      whole battle — worth **72/28** to Bulbasaur against an identical copy of
      itself. One-way, every rate would be that advantage plus the character with
      no way to tell them apart. `Matchup.First`/`Second` stay apart on the record
      precisely so the **control row** (the character against itself) can report
      what the slot alone was worth: +44.0% Bulbasaur, +23.0% Charmander, +9.0%
      Squirtle — small where duels run long, because a head start washes out.
      The control is **excluded from the headline**: it is even by construction, so
      counting it drags every answer to the middle.
      ⚠️ **A per-row `Failure` was built and then removed as unreachable.** Both
      duellists stand in the front column where `hex.ReachNeeded` is 1, every
      non-self skill must declare a range of at least 1, and a self-aimed skill
      aims at a cell that is always occupied — so `battle.New` cannot refuse one
      pairing and accept another, and what it *can* still refuse is a fault in the
      books that every row would share. A refusal is therefore an error out of
      `Spar`, and `TestTheDuelSlotAsksTheLeastOfAKit` is what says so if the board
      ever changes shape. **An ally-only kit does not trigger it** — an ally target
      reaches its own side's cells, which includes the caster's.
      ⚠️ **`Endless` is not a draw** and is in neither half of a rate's fraction: a
      pair that can never resolve would otherwise read as a pair that reliably
      loses.
      ⚠️ **A spar chooses a loadout where a roster refuses to**, and the two are
      not in conflict: a roster *is* the conditions of a battle so it has nobody to
      state them to; a spar is a measurement, and a measurement states its
      conditions — which is why `Duellist` carries the kit and both front-ends
      print it above the figures. It reads **declaration order**, which gives a
      learnset a meaning it did not have (first declared is first choice); every
      other rule would be `forge` inventing an opinion about what a character is
      for.
      Two things moved to make it possible: `SkillSlots`/`TraitSlots` are now in
      **`internal/core/cast`** (two callers read them, and a second copy of "four"
      is how a measurement stops measuring what gets fielded), and `forge.Library`
      now reads `modifiers.json`, which it never had to before — nothing an author
      writes is checked against the bounds, but a battle needs them.
      **Found immediately:** Squirtle loses to Charmander **30.5/69.5** at the cap,
      with water on the chart against fire. Cast tuning, not a mechanism bug — but
      nothing before this would have said it.
- [x] **Summoning — SHIPPED as an engine, nothing in the cast uses it yet.**
      `skill.Summon` + `battle.summon`: a skill that puts units on the caster's
      side. Two new event kinds, `Summoned` and `Left`.
      ⚠️ **Three stat spellings, exactly one per skill**: `share` (the caster's
      stats **as they stand**), `share_of_base` (ignores timed effects, for when
      a pre-buffed copy is an exploit), `stats` (a fixed line — a toad is its own
      animal). Either share is **frozen at the cast**, like a DoT tick.
      ⚠️ **Three ways off**: killed · `lasts` counts the summon's **own** turns
      (a cooldown counts the caster's, same reason) · `bound` goes with its
      summoner. `bound` is per-skill, not a rule: a clone is an extension, a
      called creature is not.
      ⚠️ **A summon COUNTS in `checkEnd`** — a side holding only a clone has not
      lost. And `dismissBound` runs **before** `checkEnd` in `kill`, or a battle
      is declared running that the next line ends.
      ⚠️ **Goes through `enlist`**, which is every rule about standing here. A
      summon building its own `Unit` is a second answer to all of them.
      ⚠️ **Nothing is in the log** — derived from caster + skill + board + a
      counter on the caster, so the id is built in the engine. An id a caller
      chose is a fact `--verify` would have to carry.
      ⚠️ **A fallen ROSTER unit keeps its slot; a departed SUMMON does not.** The
      formation is what the roster wrote down, and a summon was never in it — it
      borrowed an empty slot. Counting a departed summon kills a repeatable skill
      **silently**: shipped formations leave 2 free slots a side, so the third
      cast of a battle puts nothing down and says nothing. ⚠️ The cell is reusable
      and the **id is not** — `Unit.Summoned` never resets, because an id is what
      a log decision names. ⚠️ **The first version counted every corpse and the
      reason written down for it was WRONG** ("removing a corpse changes what
      everybody can aim at") — `Battle.occupant` already skips the dead, so a
      corpse is not a target and blocks nothing but a place to stand; the
      corpse-sensitive reach check is in `New`, over a roster. ⚠️ **Front column
      first** — `range hex.FormationCols` walks *backward* and drops every summon
      at the far edge where range 1 reaches nobody.
      ⚠️ **A summon may not summon**, checked in a **second pass** over the
      finished book: the summoned skill may be declared below the summoner.
      ⚠️ **`Suggest` never casts one** (power 0 ⇒ fallback only), so both kinds
      are proved by a hand-played battle — `aHandPlayedSummon`, beside
      `aHandPlayedGateCrossing`.
      ⚠️ **`unit.HP = 0` is NOT a death** — nothing reads health looking for a
      corpse, `kill` sets `Dead`. A test that "killed" a copy that way left the
      board with no vacated cell on it and a mutation freeing vacated cells
      survived. Drive a departure the engine performs.
- [x] **The first summoner, and the second origin — Naruto, cast only.** In the
      cast and **not the roster**, so the mechanism gets a real user and
      `replay.golden` does not move.
      New: origin `naruto` (the first non-Pokémon), preset **`summoner`** (column
      1 — it stands a row back and lets the copies spend the turns; no existing
      preset is that), species `human`, six skills, and one character with three
      stages.
      **Two summons, one of each kind**: `shadow_clone` is a share (2 copies at
      400‰, `bound`, `lasts: 4`), `summon_toad` is a fixed line with its own
      element — the case a share cannot write.
      ⚠️ **`Suggest` now casts both** — see *Pricing a summon* below. It did not
      when this shipped (power 0 ⇒ fallback only), so every figure ever measured
      of the summoner was measured with its own mechanism idle.
      ⚠️ **A new preset needs a row in `cast_test.go`'s hardcoded design table**
      and a gloss in `archetypeGloss`, or two tests fail by name.
      ⚠️ **Art is REQUIRED** — `cast.ParseBook` refuses "declares no image". Three
      pictures, one per stage, traced with **`img2svg -q balanced`** (302–420 KB,
      in line with the 21 assets already there). `faithful` came out at 849 KB for
      the busiest of them.
      ⚠️ **A PNG-saved-as-JPEG carries the transparency chequer as PIXELS**, and
      tracing that wraps the character in a grey-and-white background.
      `TestTheShippedArtIsCutOutRatherThanFramed` catches it — it measures the
      corners of the **inked** rectangle, not of the canvas, so any background
      fails and so does a body ending in a straight wide line. Strip it with
      `img2svg --decheck`: erasing by colour holes an eye highlight, a white fur
      collar and a metal headband, and a border flood alone cannot reach a patch
      enclosed between an arm and a coat.
      ⚠️ **An authored summon name is Vietnamese**, so `describeSummon` prints it
      only in `Vi` and says "copy"/"copies" in English — the division `Gloss`
      makes, and a summon has no id to fall back on.
      Its six skills are now `restrict.origins: [naruto]` — see **Origins**
      above; the `summoner` preset keeps them, which is why that ban does not
      exist.
- [ ] **Grow the cast.** Eight characters ship across two origins — Bulbasaur,
      Charmander, Squirtle, Poliwag, Machop, Cleffa and Magnemite out of Pokémon,
      Naruto out of his own — covering eight elements, water twice. This item said
      "three, one per element" until 2026-08-28, "four, one per element" until
      2026-08-31 and "five" until Magnemite; #98 landed Naruto and #182 landed
      Poliwag, which is where one-per-element stopped being true.
      ⚠️ **Magnemite is the first character to declare two elements**
      (`"element": ["electric", "metal"]`), and the field has accepted an array
      since the chart was written — `element.Dual`, `ValidateAffinity` and
      `MultiplierAgainst` were all shipped with no user. What the pairing turned
      out to be worth is written down in `internal/seed/dual_test.go`: a pair
      whose halves are unrelated mostly *cancels* — electric/metal is countered by
      four elements as two singles and by **one** as a pair — so the defensive half
      of a dual is close to nothing, and what is bought is the second skill book.
      The seed roster is no longer a mirror — so the thing this item was
      blocking, a measurable balance figure, exists. What is left is content,
      under three constraints: an archetype's kit constrains a character's
      affinity (`skill.CanCarry` enforces it while authoring, `Archetype.Demands`
      reports it), `progression.Limits` bounds health and defence **together**
      because those two multiply, and — softer than the other two — a skill kept
      for a lineage asks the character to *be* one, so adding a dragon is two
      lines: the kind in `species.json` and the claim on the character. **A new
      skill also has to say which story it is out of** — `restrict.origins`, or a
      line in `sharedPool` arguing it belongs to nobody;
      `TestEverySkillSaysWhichWorkItIsFrom` refuses the omission.

      ⚠️ **A character moves `cast.golden`, `species.golden` and `origins.golden`
      — not `scenarios.golden` and not `replay.golden`.** This item claimed the
      opposite until 2026-08-31, and the two items above that say a cast-only
      addition moves no golden were right all along: `replay.golden` renders the
      **roster**, so a character reaches it by being placed in `roster.json` and
      by nothing else. Add a preset and `archetypes.golden` moves too; add skills
      and `skills.golden` and `describe.golden` move. #182 added a character, a
      preset and four skills, and moved exactly those six.

      Read squirtle first — water is the strongest of the elements and Blastoise
      still cannot carry an ace slot, because its attack and speed curves are the
      lowest in the cast.

## Pricing one number: `hexforge weigh`, and why a roster win rate could not

`hexforge weigh <character> <skill> --field F --values a,b,c` prices **one field
on one skill** by fighting the carrier against a copy of itself whose only
difference is that field. `forge.WeighReport` is the measurement, `internal/forge/weigh.go`;
`cmd/hexforge/weigh.go` only draws it. **It authors no balance data** — the
variant lives in memory, `skill.Book.Append` marshals and re-parses it, and
`internal/seed/data` never moves.

⚠️ **The roster win rate does not price anything, and the control experiment is
why.** Over all 20,000 seeds, post-#136 roster, baseline **47.55%** ally:

| change | ally rate | Δ |
|---|---:|---:|
| `razor_leaf` crit 200 (ally lands it 3.2:1) | 45.41% | **−2.14pp** |
| `sludge_bomb` crit 200 (**ally-only**, enemy never casts it) | 45.15% | **−2.40pp** |
| `sludge_bomb` power **+5%, no crit at all** | 45.80% | **−1.75pp** |

The last row is the control: **giving the ally more damage *lowers* its win
rate**, by about as much as the crit did, with no crit involved. The roster rate
is **non-monotone in ally damage** — a dial that is sometimes worth less the more
of it you have is not priced by that dial's reading. The same change also read
**+2.40pp before #136 and −2.14pp after**: the sign flipped on an unrelated
*placement* change. The engine was never the problem — the mechanism measured
**20.46%** against **20.00%** declared over 3,670 landing strikes.

**What a weigh figure is.** A price in **parts per thousand**, on a named
carrier, at a named level, against a copy of itself, with that carrier's kit and
stats and placement all cancelled. It is **not a win rate**, it says nothing
about the roster, and **it does not carry across a data change** — a placement
move can reverse its sign, which is exactly what happened above. Quote it with
the carrier and the level attached or do not quote it.

⚠️ **Swap the KITS, not the sides.** `forge.duel` already does. The queue breaks
a tie by enlistment, so leaving the roster order alone enlists the first-written
kit first in *both* halves and a unit against an identical copy of itself reads
**58.8%** — see the `self_gradient` entry above, where that trap was first paid
for. Reusing `duel` is the whole reason a weighing can claim its two halves
cancel. A `Damaged` event carries `Actor` and **no `Side` at all**, so the strike
tally follows the kit swap too: the challenger is `first` in one half and
`second` in the other, and folding on the wrong id reports the *opponent's*
strikes with nothing on screen looking wrong.
`TestAMatchupCountsTheChallengersStrikesAndNotTheOpponents` is what holds it.

⚠️ **The control is a row of every report, not a test somewhere else.** The
sweep always inserts the skill's own declared value, and that row **must read
exactly 500‰** or the whole report is refused before a figure is printed. A test
proves the code was right once; this proves the twenty thousand battles just
fought were the twenty thousand intended — a variant leaked into both kits, a
side read backwards or a perturbed rng each produce rows that look exactly like
good rows and a control that is not even.

⚠️ **`turns` is a headline column, read beside `worth`, not a footnote.** A
mirror-duel rate once failed to even *order* speed amounts (`swiftness`, above:
`+150` read 59.0% while `+50` read 74.0%) because the turn queue is discrete, and
the fix was measuring inside one battle. Damage is lumpy at the **kill
threshold** for the same reason — a little more of it buys whether the last
strike lands this turn or next. Median turns-to-finish is the inside-one-battle
reading: more damage kills sooner, monotonically, with no win/lose boundary to be
lumpy at. **A row whose `worth` is inside the band but whose `turns` moved is a
real effect**, and the footer says so. Monotonicity is reported for the two
columns **separately**, because they can disagree and the disagreement is the
finding.

**The band.** `2σ = 1000/√N` permille over `N = 2 × seeds` battles, integer
square root and ceiling division — no float anywhere, permille throughout, and
never zero. The default of **10,000 seeds** (20,000 battles, ~7.5s, band
**±0.8pp**) comes from that arithmetic and not from taste: the effects above were
1.75–2.40pp, two to three times the band.

**Refusals, each because the row would otherwise print as an ordinary number.**
Seeds below one; a level outside `1..LevelCap`; an unknown character or skill; a
skill the carrier does not bring at that level and form (it would be *measured* —
an even row priced at nought on a skill nobody cast); a variant id the book
already declares; a value the parser refuses (returned in **the parser's own
words, unwrapped**, because every bound is `skill.resolve`'s and a second copy
would be free to disagree); a row that **landed the skill zero times** — worth
nothing means *not rated*, never *rated at nought*, which is the `fire_fang`
ally-0 lesson made permanent; a row where the mechanism never fired (crit is the
only field with a mark of its own on an event); a row whose `Endless` exceeds a
fifth of its battles; and a row with a **saturated half** (`First.Rate() >= 990`
or `Second.Rate() <= 10`), which has no room left for the field to move.

⚠️ Only **scalars** are weighable: `power`, `accuracy`, `pierce`, `crit`,
`strikes`, `drains`, `cooldown`, `range`. `self_gradient` is excluded because it
is two numbers, so a curve becomes a surface; `applies`, `strips`, `summons`,
`restriction`, `element`, `target` and `pattern` are excluded because changing one
**authors a different skill** rather than moving a dial, and the deviation would
be that other skill's worth.

⚠️ **Do not try to seed the two twins separately.** They share one `*rng.Source`,
so changing one side re-scrambles every draw after it — roll drift. It **biases
nothing** (both arrangements fight the same seeds) and fixing it would mean an
`internal/core` change to make a measurement prettier.

### Across the whole cast: `--carriers all`, and why there is no average

`hexforge weigh --carriers all <skill> --field F --values a,b,c` takes the same
price once per character whose **fielded kit** brings the skill and prints a
table. `forge.WeighCarriers` is the sweep, `internal/forge/carriers.go`;
`cmd/hexforge/carriers.go` only draws it. The character operand is dropped: the
carrier is discovered rather than named.

⚠️ **It has no headline figure and must never grow one.** A weighing is a price
against a copy of *the carrier* — that is the only reason it is a price, because
everything else has been made identical on both sides and cancelled — so two
rows were fought against **two different opponents** and are not two readings of
one quantity. An average of them has no opponent it was taken against and no
board it was taken on, which is the exact shape of number this repository has
twice been burnt by: the roster win rate (non-monotone in ally damage, sign
flipped by a placement change) and the mirror-duel speed reading (could not order
`swiftness` at all). **A carrier may be compared only to itself, at another
value, along its own row.** The footer says that in words and
`TestTheCarrierFooterRefusesTheCrossCarrierComparison` asserts the sentence is
still there.

**One row is one weighing**, not a second instrument: same control, same band,
same fold, and `TestOneCarrierWeighedAloneAndInTheTableAgreeRowForRow` holds the
row identical to `hexforge weigh` on that carrier alone. What the table adds is
who is in it and in what order:

- **Membership** is `forge.NotBroughtError` caught with `errors.As`, so exactly
  one place knows what "brings" means. A character that cannot bring the skill is
  **absent** rather than a row of noughts — the same distinction as refusing a
  row that landed nothing instead of pricing it at nought — and the footer counts
  the skips and says why.
- **Refusals are per row.** A refused carrier keeps its line with a dash where
  its figures would be, spelled out under the table, and the other rows stand.
  ⚠️ Except a **control that is not exactly 500‰**: it is marked `⚠️ HARNESS` and
  sorts above every priced row, because every other refusal says *this carrier is
  uninteresting* and that one says *the run leaked*. Drawn alike they would be
  the same dash in the same column.
- **A value is refused whole, and once.** It is a fact about the skill rather
  than about any carrier, so every variant is built before a die is rolled — one
  parser sentence, instead of the same sentence per row discovered after the
  first carrier's battles had been paid for.
- **Order** is worth at the largest swept value, descending, ties by character
  id, harness refusals first and other refusals last. It is stated in the footer
  and is never the order the cast was read in — `internal/core` bans a map
  iteration that reaches an output, and this is the same discipline one layer up.
- Each cell is **`worth/turns`**, because the two are co-equal columns here for
  the reason they are in `renderWeigh` and cannot be adjacent columns on a table
  this wide. A row whose own sweep is not monotone is marked `not ordered` on its
  own line: a dial that is not monotone is not priced, and on a table the figures
  are all a reader sees.

**Cost** is `2 × seeds × values × carriers`, printed above the table. `--seeds`
defaults to **2000** here (band **±1.6%**) rather than 10,000, because the table
multiplies — that is the count the first crit chances were actually authored at.
Re-take an interesting row on its own with `hexforge weigh` at the full ten
thousand.

⚠️ **No shipped character shares a skill with another**, so on
`internal/seed/data` today every `--carriers all` table has exactly one row and a
footer of three skips. That is the tool working rather than failing: the skips
are the part that was previously noticed by eye.

**Left at its default, deliberately:** `self_gradient` support (excluded, above).
A second field per row is not coming either — that is a surface, and a surface is
not something a column of figures can report.

### What the first crit chances cost, and what the readings taught

The first two were authored off `weigh`, one carrier each, 2000 seeds, band
±1.6%, a chance of 200‰ throughout:

| carrier | skill | worth |
|---|---|---|
| `pokemon.bulbasaur` | `razor_leaf` | **+8.4%** |
| `naruto.naruto` | `wind_shuriken` | **+6.9%** |
| `pokemon.charmander` | `fire_fang` | +1.0% — inside the band |
| `pokemon.squirtle` | `bite` | +0.2% |
| `naruto.naruto` | `kunai` | **−1.7%** |

The first two shipped and the last three did not.

⚠️ **A theme is not a price.** The rule these were picked under first was that a
crit belongs to a blow that strikes a point rather than to a spray, and the
measurement disagreed: `bite` and `kunai` are as pointed as `razor_leaf` is and
are worth nothing and less than nothing. What the price actually reads is the
**carrier's whole line** — `razor_leaf` and `wind_shuriken` are what their
carriers spend their turns on, `bite` sits in a squirtle mirror that grinds for
141 turns where damage barely decides, and `kunai` is cheap enough that making it
better gets it **cast more often**, crowding out the skill that should have been
cast. That last one is `outrage` wearing a different hat: a rating that likes a
skill more will reach for it more, and *more often* is not the same as *better*.

⚠️ `kunai` reading **negative** is the most useful row in the table, because it
is the proof that the instrument can see a buff make its carrier worse. A win
rate on the roster could not; that is what § *Pricing one number* is about.

The roster moved **47.56% → 49.73% ally** over 20,000 seeds afterwards, and
**8.02%** of all damaging strikes are now critical against nought before. Both
are recorded as observations. The roster figure prices nothing — `razor_leaf` is
carried by a level 60 ace on one side and a level 30 unit on the other, so the
shift is mostly whose hand it is in — and the strike share is the mechanism
check, not a result.

**The rest of the cast crits at nothing, and that is authoring rather than a
backlog.** Twenty-six damaging skills carry no chance. They are added **one at a
time, each off its own `weigh` reading**, by whoever is authoring — there is no
sweep to run and no target number of skills to reach, so it was taken out of
`TODO.md` rather than left standing as a task nobody was going to pick up whole.
⚠️ What must not be lost with it is the rule above: **the price is a fact about
the carrier's whole line, not about the skill's shape**, so a crit may not be
authored from a theme. `bite` and `kunai` are as pointed a blow as `razor_leaf`
is and priced +0.2% and **−1.7%** — a cheap skill given a crit gets cast *more*
and crowds out a better one, which is the `outrage` lesson wearing a different
hat.

## Fighting two ratings: `forge.Bout`, and the exact control it rests on

A roster win rate cannot see a rating. Both sides use `Suggest`, so a change that
helps both leaves the rate where it was, and what moves is whichever squad's kit
had more to gain — which is why the tie-break was measured by hand with itself
switched on for one side at a time. `Bout` is that procedure made an instrument:
two `Brain`s, the shipped roster, and a share read from the challenger.

**The two arrangements.** Every seed is fought twice — challenger driving the ally
squad, then the two exchanged — over the same seeds `1..N`, and the challenger's
wins are counted over all `2N` battles.

⚠️ **The control is exact, not statistical, and is asserted at exactly 500‰.** When
both `Brain`s are the same, the two arrangements are the *same battle* — same seed,
same roster, same decisions — so they have the same winner and the challenger holds
the winning side in exactly one of the two. That is `N` wins and `N` losses **by
construction**; a draw lands in both arrangements and counts half to each side, so
it does not disturb it either. `Bout` runs that control before it measures anything
and **refuses to print a figure** unless it comes out exactly even. Never assert it
within a band: a band is precisely what a bookkeeping leak would hide behind.

**The board does not have to be symmetric**, because the roster's asymmetry is
fought from both ends and cancels exactly — that is the whole content of the
paragraph above. So a bout measures on the **shipped** roster, which is the board
every other figure in this file is quoted on. ⚠️ The ban on a mirror roster is
about a **data** change — the shipped cast may not be flattened into two copies of
itself to make a number easier to read — and does not apply here: nothing is
authored, the roster is a parameter, and the mirror is in the procedure.

⚠️ **`FirstUsable` is a ruler and may never be improved.** It is `Suggest` before
any pricing landed: first option in kit order that is available, aimed at the first
cell offered, nothing compared. Every claim about a rating is of the shape "beats
`FirstUsable` by X‰ over N seeds", and X is comparable across this repository's
history only while the thing it is measured against never moves. A sharper ruler
would silently re-scale every figure ever quoted against it and nothing would
report the disagreement. `TestTheRulerIsNotAnOpponent` pins it. If a second, harder
baseline is wanted, **add one beside it**.

**The figure.** The shipped rating against the ruler, on the shipped roster, over
**10,000 seeds (20,000 battles): 77.9%**, band ±0.8pp, median **45** turns. The
control on the same board and the same seeds is 500‰ exactly at a median of **48**
turns — so the priced rating both wins far more often *and* finishes sooner, which
is the second reading and the one a rate alone cannot give.

**It refuses**: seeds below one; a control that is not exactly even; and a run whose
`Endless` share passes `endlessShare`, because two better opponents stalling each
other is a real risk of any change to a rating and must not be filed as a draw.

⚠️ **Roll drift, accepted and not fixed.** The two arrangements share one
`*rng.Source` per battle, so the first decision the two ratings disagree on
re-scrambles every draw after it. It **biases nothing** — both arrangements fight
the same seeds — and fixing it would mean an `internal/core` change to make a
measurement prettier. Same note `Weigh` carries; see § *Pricing one number*.

`Bout` takes the roster as a **parameter**, so `internal/forge` keeps not importing
`internal/seed`: a tool that read the embedded copy would keep reporting on the cast
as it was at the last build.
