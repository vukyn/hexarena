---
name: fixture-decides-what-is-visible
description: hexarena — five measured cases where a whole suite could not see a defect because of how its fixture was built: a mode flag set by hand, two identical squads, a slice-aliasing test whose reachability is an unobservable capacity, every sweep entry aimed at a character with no fork, and SHIPPED DATA that already satisfies the property under test
metadata:
  type: feedback
---

Before believing "the suite over X is the net for X", check what the fixture
actually does. Four measured cases, all of which had gone unnoticed:

**1. A fixture that sets a mode field instead of pressing the key leaves that
key driven by nothing.** `cmd/hexforge-tui`'s `everyScreen` builds its
new-skill-form entry as `addSkill.skills.Adding = true` rather than by pressing
`a`. So when `screen.Context.Authoring` was added and its guard on the skill
listing's `a` was inverted as a mutation, the counts were:

| mutation | internal/screen | cmd/hexarena-tui | cmd/hexforge-tui |
|---|---:|---:|---:|
| `skills.go`'s `a` guard inverted | **4** | 2 | **0** |
| `squads.go`'s `n` guard inverted | **10** | 2 | reddens |

Nothing in the client that *has* the key drives it. The squad builder's keys are
driven (its tests press `n`/`enter`/`d` through the model), which is why the two
rows differ — so "the authoring client's own suite presses those keys" was true
of one screen and false of another, and only the mutation said which.

**2. A mirror fixture cannot see a swap.** `cmd/hexarena-tui`'s battle opens on
two squads. Building both fixture sides from the *same* character makes the two
halves of a battle interchangeable — same board, same roster rows, same order
line whichever way round they are fielded — so nothing, golden included, can see
a client that hands `PlayScreen.Open` its `home` and `away` the wrong way round.
The fixture walks the cast for **two different characters** and asserts they
differ, and the behaviour test names **both** halves over **every** row of the
catalogue (home alone passes on a client that fields the named side twice; a
cursor on the last row cannot tell `+1` from correct).

**3. A slice-aliasing test is only reachable while the backing array has spare
capacity — and a caller cannot observe somebody else's capacity.** `Battle.Since`
hands out `b.events[from:n:n]`; drop the third index and a consumer's `append`
scribbles into the record and gets scribbled on. But that can only happen when
`cap(b.events) > len(b.events)`, and a black-box test has no way to ask. So one
reading of it is a coin flip dressed as an assertion. Two things fix it: `cap`
is a builtin, so `cap(view) != len(view)` states the property **directly** from
outside the package; and the corruption half is a **sweep** over ten record
lengths rather than one, because append growth is amortised so only a minority
of lengths have no spare room, with a `checked` counter that `t.Fatal`s if the
sweep observed nothing. Report the cap breach with `Errorf` and not `Fatalf`, or
the cause aborts the run before the two damage readings get to say what it costs.

**4. A sweep entry aimed at the *simple* row measures nothing about the branching
one — registering a screen is not the same as covering it.** `PreviewScreen` went
into all three sweeps (#254) pointed at the first row of the cast at the level
cap. The first row does not fork. `pokemon.poliwag` — the one shipped
`Poliwag@1 → Poliwhirl@16 → (Poliwrath@32 | Politoed@32)` — could not be drawn at
all from level 32 up: `progression.Line.StageAt` refuses to name a form when two
are reached, so the preview drew that refusal in red. Three records, three
sweeps, all green, and a user found it the next day. The subject a sweep entry
points at is as much a fixture decision as the fixture file is; write the
"find the interesting row and `t.Fatal` if there is none" helper rather than
taking cursor 0.

**5. The shipped data may already satisfy the property, so the real thing is the
wrong fixture — and "use the real data" is the instinct that gets this wrong.**
`internal/draft.NewPool` must keep `cast.Book`'s **declaration order** and must
not sort by id (order is a presentation choice; five other lists in the repo are
in declaration order, and a draft screen that sorted would differ from every
screen the player just came from). Measured: `internal/seed/data/cast.json`
declares `naruto.naruto` then every `pokemon.*` alphabetically — which **is**
sorted order — so a test that built a pool from `seed.Cast()` and compared it to
the book's order passes on a `NewPool` with `slices.SortFunc(…, byID)` welded
into it. Confirmed by mutation: both `sort by id` and `reverse` are caught only
by the test that uses a five-character synthetic book whose declaration order is
deliberately not its id order, and that test's **first assertion is
`slices.IsSorted(ids) → t.Fatal`**, so a fixture drifting into sorted order takes
itself out rather than going quietly vacuous.

Note this is the opposite failure to case 4: there the fixture was too simple, so
point it at the shipped data; here the shipped data is *coincidentally* the
answer, so build a fixture. The question is not "real or synthetic", it is
**"can this input distinguish the correct implementation from the wrong one"** —
and for an *ordering* property specifically, ask whether the real collection is
already in that order before writing anything.

**Two operational traps that only appear once an entry points at a rich
character** — both measured on this one:

- **`aPictureRow` refuses a blank row on purpose**, so the moment a preview entry
  draws *shipped* art instead of the fixture's solid rectangle, the drawing's
  transparent margin — a full-width run of spaces — is read as a wording line 198
  cells wide and both clients' width sweeps go red. Fix it in the sweep (skip
  whitespace-only lines) and not in `aPictureRow`: a count of painted rows reads
  the same predicate and would go vacuous.
- **`Context.WrappedIn` spends `UsableWidth() - 1 - 2 - width - 1`**, so any
  wrapped data row is as wide as the sweep's 200-column fixture by construction.
  Nothing had ever put a long enough value in front of it; poliwag's sixteen
  glossed skills are the first, and registering the *detail pane* therefore needs
  a `kitGlosses` exemption (a twin of `whoMayCarry`/`traitCarriers`) in **both**
  client fixtures. Budget for that before promising a browse entry.

**How to apply:** when adding a guard, an ordering or anything with two
symmetric arguments, mutate it and *count failures per package* before writing
down which suite holds it. When the property is about memory rather than
behaviour, ask first whether the fixture can reach the state the fault needs —
and sweep if the answer is "only sometimes, and I cannot tell when". If a fixture reaches a state by assignment rather than
by the keystroke, the keystroke is unmeasured — and the sentence to write in the
doc comment is the measured table, not the plausible claim.

Related: [[fixture-hidden-branch]], [[two-screen-goldens]],
[[screen-extraction]], [[fixture-cast-edit-costs-goldens]],
[[a-refusal-can-be-right-for-the-wrong-reason]].
